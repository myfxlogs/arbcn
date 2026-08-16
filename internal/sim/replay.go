// 回放证伪门禁引擎（D-065 修订：业主指令「不做可选，是每个策略都自动做，做成门禁」）。
//
// 核心：每个策略在**自己的高费率过滤档**下回放历史，验证「历史上高费率窗口出现时
// 门禁会不会正确触发、扣摩擦后是否为正」。证伪不证真（practices #38）：falsified =
// 结构性否定可信；pass = 证伪未发生，**非收益预测**；no_window = D-061 ② 环境无窗口，
// 门禁休眠 = 正确输出。判据由 Driver 在 buildSignal 时预计算（SignalToOrder 保持纯
// 函数无 I/O），并供 simapi GetReplayState 只读证据面（P4 可检查性：门禁休眠也可见）。
package sim

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// 回放判据参数（D-065，值改走 D#）。单位：%（pct_annualized 百分点点数）。
const (
	// ReplayHistoryDays 回放历史窗口（天）：funding 历史 ~365d（D-037 回填 binance），
	// reverse_repo/defi_rate 受采集起点约束；窗口内无高费率档 → no_window（诚实）。
	ReplayHistoryDays = 365.0
	// replayMinWindowDays 摩擦年化摊薄下限：窗口时长 <1 天按 1 天摊（「最短可捕获窗口」
	// 假设，防几小时读数把年化摩擦摊出天文数字；更短的窗口本就抓不住）。
	replayMinWindowDays = 1.0
	// ReplayMaxWindowsShown per_pair 明细窗口数 cap（防爆前端）。
	ReplayMaxWindowsShown = 10
	// replayMaxFacts 回放查询上限（backfill 同款 500k，个人数据量远够）。
	replayMaxFacts = 500_000
	// replayStableBasePct 稳定币基档（D-021/D-026）：watch 上界——净年化不抵无风险档 =
	// 门禁在该阈值下无经济意义（宁缺毋滥 D-019）。dashboard/oppcalc.go 有同值常量
	// stableBasePct（独立包，同 D-021 源；改 D-021 须两边同步，P3 文档化）。
	replayStableBasePct = 4.5
)

// 回放证伪判定类值域（D-065）。
const (
	ReplayFalsified = "falsified" // 证伪：有窗口但均值净年化 ≤0（结构性否定，practices #38）
	ReplayWatch     = "watch"     // 观察：有窗口但均值净年化 ∈(0, 稳定币基档]（门禁无经济意义）
	ReplayPass      = "pass"      // 通过：有窗口且均值净年化 > 稳定币基档（证伪未发生，非收益预测）
	ReplayNoWindow  = "no_window" // 环境无窗口：历史无该策略高费率档窗口（D-061 ② 门禁休眠 = 正确输出）
)

// replayGateCfg 每策略自己的高费率过滤档（D-061 核心子句泛化）。rateKind = 回放数据面
// 事实 kind（与 settleFactKind 结算分派同源，SSOT 在此表）；tierPct = 该策略高费率窗口
// 档（≥ 计入）；frictionPct = 一次性进出摩擦 %（按实际窗口时长年化摊薄）。
//
// [值锚点 D#] funding_hedge 15%/0.3% = D-016 15% 档（同 dashboard WindowTierHigh）+
// D-046 双开双平 taker 已核实；carry_asset 8% = D-021 收益阶梯 sUSDe 类档下沿 + D-046；
// repo 5% = D-061 ① 民营定期 5% 档，摩擦 0%（OTC 协议存款/逆回购无 taker 费——若按
// CEX 0.3% 摊，单读数窗口 frictionAnn=109.5% 会误证伪，real 数据 repo 5~6% 全被错杀）。
type replayGateCfg struct {
	rateKind    string
	tierPct     float64
	frictionPct float64
}

var replayGateCfgs = map[string]replayGateCfg{
	store.SimKindFundingHedge: {rateKind: fact.KindFunding, tierPct: 15, frictionPct: 0.3},
	store.SimKindCarryAsset:   {rateKind: fact.KindDefiRate, tierPct: 8, frictionPct: 0.3},
	store.SimKindRepo:         {rateKind: fact.KindReverseRepo, tierPct: 5, frictionPct: 0},
}

// ReplayKindConfig 查询策略的回放档配置（rateKind 数据面 / tierPct 高费率档 /
// frictionPct 一次性摩擦）。未知 kind → ok=false（不判不拒）。simapi GetReplayState
// 证据面与 Driver 门禁共用本表（P3 单源）；settleFactKind 结算分派亦读本表。
func ReplayKindConfig(kind string) (rateKind string, tierPct, frictionPct float64, ok bool) {
	c, ok := replayGateCfgs[kind]
	return c.rateKind, c.tierPct, c.frictionPct, ok
}

// ReplayWindow 一次高费率窗口回放。Start/End 窗口起止（单样本 End=Start）；Samples
// 窗口内 ≥tierPct 读数数；MeanFunding/MinFunding 窗口均值/最低年化；FrictionAnn 一次性
// 摩擦按实际窗口时长年化摊薄 = friction×365/max(days, 1)；NetAnn = MeanFunding −
// FrictionAnn（扣摩擦净年化）。
type ReplayWindow struct {
	Start, End  time.Time
	Samples     int
	MeanFunding float64
	MinFunding  float64
	FrictionAnn float64
	NetAnn      float64
}

// ReplayPair 单对回放结果。MeanNetAnn 为样本加权均值净年化，BestNetAnn/WorstNetAnn
// 为各窗口 NetAnn 极值；Verdict 由 classifyReplay 判定；Windows 明细 cap 前 10。
type ReplayPair struct {
	Venue, Symbol string
	WindowCount   int
	HighSamples   int // ≥ tierPct 的读数数
	TotalSamples  int
	MeanNetAnn    float64
	BestNetAnn    float64
	WorstNetAnn   float64
	Verdict       string
	Note          string
	Windows       []ReplayWindow
}

// ComputeReplay 单对 rate 历史 → 回放结果。tierPct 该策略高费率档，frictionPct 一次性
// 摩擦 %。facts 空 → WindowCount=0 + Verdict=no_window + Note 明示无数据（不 panic 不
// 编造，practices #7）。输入未按 ts 排序 → 内部先升序（采样乱序不破坏窗口扫描）。
func ComputeReplay(fs []fact.Fact, tierPct, frictionPct float64) ReplayPair {
	p := ReplayPair{TotalSamples: len(fs)}
	if len(fs) == 0 {
		p.Verdict = ReplayNoWindow
		p.Note = "无历史数据，回放不可用（不编造）"
		return p
	}
	p.Venue, p.Symbol = fs[0].Venue, fs[0].Symbol

	sorted := make([]fact.Fact, len(fs))
	copy(sorted, fs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Ts.Before(sorted[j].Ts) })

	// 扫描连续 ≥ tierPct 的 run（读数落到档下即断开 → 窗口边界）。
	var windows []ReplayWindow
	var cur []fact.Fact
	flush := func() {
		if len(cur) == 0 {
			return
		}
		windows = append(windows, windowFrom(cur, frictionPct))
		cur = nil
	}
	for _, f := range sorted {
		if f.Value >= tierPct {
			cur = append(cur, f)
			p.HighSamples++
		} else {
			flush()
		}
	}
	flush()

	p.WindowCount = len(windows)
	if p.WindowCount == 0 {
		p.Verdict = ReplayNoWindow
		p.Note = fmt.Sprintf("历史 %d 份读数均 < %v%%：无高费率窗口档（D-061 ② 环境无窗口，门禁休眠 = 正确输出，非门禁故障）", len(fs), tierPct)
		return p
	}

	sumNet := 0.0
	p.BestNetAnn, p.WorstNetAnn = math.Inf(-1), math.Inf(1)
	for _, w := range windows {
		sumNet += w.NetAnn * float64(w.Samples)
		if w.NetAnn > p.BestNetAnn {
			p.BestNetAnn = w.NetAnn
		}
		if w.NetAnn < p.WorstNetAnn {
			p.WorstNetAnn = w.NetAnn
		}
	}
	p.MeanNetAnn = sumNet / float64(p.HighSamples) // 样本加权均值净年化
	p.Verdict, p.Note = classifyReplay(p, tierPct)

	if len(windows) > ReplayMaxWindowsShown {
		windows = windows[:ReplayMaxWindowsShown]
	}
	p.Windows = windows
	return p
}

// windowFrom 一段连续 ≥tierPct 读数 → ReplayWindow。窗口时长 = end−start；单样本/不足
// 1 天的窗口按 replayMinWindowDays 摊薄（最短可捕获窗口假设）。
func windowFrom(run []fact.Fact, frictionPct float64) ReplayWindow {
	start, end := run[0].Ts, run[len(run)-1].Ts
	if end.Before(start) {
		end = start
	}
	sum, min := 0.0, math.Inf(1)
	for _, f := range run {
		sum += f.Value
		if f.Value < min {
			min = f.Value
		}
	}
	mean := sum / float64(len(run))
	days := end.Sub(start).Hours() / 24
	if days < replayMinWindowDays {
		days = replayMinWindowDays
	}
	frictionAnn := frictionPct * 365 / days
	return ReplayWindow{
		Start: start, End: end, Samples: len(run),
		MeanFunding: mean, MinFunding: min,
		FrictionAnn: frictionAnn, NetAnn: mean - frictionAnn,
	}
}

// classifyReplay 回放证伪判定（D-065）。均值净年化：
//
//	≤ 0                       → falsified（结构性证伪：历史最高费率时段也覆盖不了摩擦）
//	∈ (0, stableBase]         → watch（有窗口但净不抵稳定币基档 D-021，门禁无经济意义）
//	> stableBase              → pass（证伪未发生：门禁在历史高费率窗口能抓净正——非收益预测）
func classifyReplay(p ReplayPair, tierPct float64) (string, string) {
	if p.MeanNetAnn <= 0 {
		return ReplayFalsified, fmt.Sprintf("回放证伪：%d 个 ≥%v%% 窗口均值净年化 %.2f%% ≤ 0：历史高费率时段也覆盖不了摩擦，门禁机制被结构性证伪（practices #38）", p.WindowCount, tierPct, p.MeanNetAnn)
	}
	if p.MeanNetAnn <= replayStableBasePct {
		return ReplayWatch, fmt.Sprintf("回放观察：%d 个 ≥%v%% 窗口均值净年化 %.2f%% ∈ (0, 稳定币基档 %v%%]：有窗口但净不抵无风险档，门禁在该阈值下无经济意义", p.WindowCount, tierPct, p.MeanNetAnn, replayStableBasePct)
	}
	return ReplayPass, fmt.Sprintf("回放通过：%d 个 ≥%v%% 窗口均值净年化 %.2f%% > 稳定币基档 %v%%：门禁在历史高费率窗口能抓净正（证伪未发生；非收益预测，practices #38）", p.WindowCount, tierPct, p.MeanNetAnn, replayStableBasePct)
}

// OverallReplay 跨对聚合（overall）：合并全部窗口，样本加权均值净年化，Verdict 复用
// classifyReplay。全部对无窗口 → no_window + Note 明示环境无窗口（D-061 ②）。
func OverallReplay(pairs []ReplayPair, tierPct float64) ReplayPair {
	o := ReplayPair{Venue: "overall", Symbol: "监控面"}
	totalHigh, sumNet := 0, 0.0
	o.BestNetAnn, o.WorstNetAnn = math.Inf(-1), math.Inf(1)
	for _, p := range pairs {
		o.WindowCount += p.WindowCount
		o.HighSamples += p.HighSamples
		o.TotalSamples += p.TotalSamples
		sumNet += p.MeanNetAnn * float64(p.HighSamples)
		if p.BestNetAnn > o.BestNetAnn {
			o.BestNetAnn = p.BestNetAnn
		}
		if p.WorstNetAnn < o.WorstNetAnn {
			o.WorstNetAnn = p.WorstNetAnn
		}
		totalHigh += p.HighSamples
	}
	if totalHigh == 0 {
		o.Verdict = ReplayNoWindow
		o.Note = "监控面历史无高费率窗口（D-061 ② 环境无窗口，门禁休眠 = 正确输出，非门禁故障）"
		return o
	}
	o.MeanNetAnn = sumNet / float64(totalHigh)
	o.Verdict, o.Note = classifyReplay(o, tierPct)
	return o
}

// replayGate 回放证伪门禁（D-065 修订，业主指令：每个策略强制自动）。按 kind 自己的
// 高费率档回放历史，返回 (verdict, note)。Driver 在 buildSignal 对每个建单信号调用；
// 查询失败/无事实 → no_window（D-061 ②，门禁不因查询失败静默放行也不误拒）。
// carry 的 venue 经 Driver 归一化（空 → sim_local）可能 miss 事实 venue → 精确查询
// 为空时放宽到 kind×symbol 全 venue 重试（防门禁因 venue 错配而误判环境无窗口）。
func (d *Driver) replayGate(ctx context.Context, kind, venue, symbol string) (string, string) {
	rateKind, tier, friction, ok := ReplayKindConfig(kind)
	if !ok {
		return ReplayNoWindow, fmt.Sprintf("回放门禁：策略 %s 无回放配置，不判不拒", kind)
	}
	from := d.now().Add(-ReplayHistoryDays * 24 * time.Hour)
	fs, err := d.st.QueryFacts(ctx, store.FactQuery{Kind: rateKind, Venue: venue, Symbol: symbol, From: from, Limit: replayMaxFacts})
	if err != nil {
		return ReplayNoWindow, fmt.Sprintf("回放门禁：查询 %s 历史失败，按环境无窗口处理（%v）", rateKind, err)
	}
	if len(fs) == 0 && kind == store.SimKindCarryAsset {
		fs, err = d.st.QueryFacts(ctx, store.FactQuery{Kind: rateKind, Symbol: symbol, From: from, Limit: replayMaxFacts})
		if err != nil {
			return ReplayNoWindow, fmt.Sprintf("回放门禁：查询 %s 历史失败，按环境无窗口处理（%v）", rateKind, err)
		}
	}
	p := ComputeReplay(fs, tier, friction)
	return p.Verdict, p.Note
}
