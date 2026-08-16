// M3-b 规则→Signal 运行驱动（04-m3-spec §9.2 S1 + §9.3 S2）。
//
// 本包保持零网络零密钥（M3-a 复审验证的 D-010 属性，§9.4 纵深防御）：Driver 只做
// 纯计算——规则命中 → 按 §3.1.1 映射表组装 Signal → Simulator.Generate 落库；
// 8h 结算调度按 (symbol,venue) 分组喂入真实市场 funding 事实。网络/key 承载在
// internal/simtestnet（S3，物理隔离）。
package sim

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// settleInterval 8h 三班资金费率结算周期（04-m3-spec §3.2）。
const settleInterval = 8 * time.Hour

// reportEveryTicks 每 7 个 8h tick（周）渲染一次周频统计报告（§9.5）。
const reportEveryTicks = 7

// signalMapper 按规则名组装 Signal 的函数（映射表值类型）。ok=false = 不建单
// （L1 复审：repo 无 reverse_repo 事实时 fail-closed，宁缺毋滥）。
type signalMapper func(ctx context.Context, d *Driver, r store.Rule, h store.EntityHit) (*Signal, bool, error)

// signalMappers 规则名 → Signal 组装（04-m3-spec §3.1.1 表编码，不可变包内常量）。
//
// [对抗测试锚点] §9.2 S1：删除任一映射 → sim/driver_test.go
// TestDriverFundingHitCreatesOrder（funding_warn→funding_hedge）/
// TestDriverFundingDrillCreatesOrder（funding_drill→funding_hedge，D-041）/
// TestDriverRepoBuildsOrder 必红。未在表中的规则（defi_large_tier_change /
// ladder_trap / iv_opportunity / usdcnh_buy_line / collector_heartbeat /
// nonstable_quote_change 等）→ 不建单（宁缺毋滥），但命中标的在白名单时仍可
// 映射 carry_asset（§9.6）。
var signalMappers = map[string]signalMapper{
	"funding_warn":         fundingHedgeSignal,
	"funding_critical":     fundingHedgeSignal,
	"funding_drill":        fundingHedgeSignal, // D-041 演练档（band [5%,15%) 由规则 Cond 限定）
	"trx_funding_positive": fundingHedgeSignal,
	"reverse_repo_timing":  repoSignal,
}

// Driver 是规则→Signal 驱动（M3-b §9.2）。
type Driver struct {
	st  store.Store
	cfg Config
	now func() time.Time
	log *slog.Logger
	sim *Simulator

	// Probe 每次 settle tick 执行的只读连通性探针（M3-b §9.7 ⑤，S3）；main.go 注入，
	// nil = 跳过（key 不可用 / 降级）。失败仅 warn 不阻断 settle 主循环（D-032 同口径）。
	Probe func(context.Context) error
}

// NewDriver 构造驱动；st 必须非 nil。cfg 由调用方保证合法（非法 → main 不构造，nil 降级）。
func NewDriver(st store.Store, cfg Config) *Driver {
	d := &Driver{st: st, cfg: cfg, now: time.Now, log: slog.Default(), sim: New(st, cfg)}
	d.sim.Now = d.now
	return d
}

// OnRuleActive 规则 armed→active 回调（接 rule.Config.OnActive，M3-b §9.7 ③）：
// 每个命中实体按 §3.1.1 组装 Signal → Generate 落库（suggested/rejected 均落，负样本）。
// 无法映射的规则不建单。单次激活一实体会生成至多一单（armed→active 转变语义防重复）。
func (d *Driver) OnRuleActive(ctx context.Context, r store.Rule, entities []store.EntityHit) error {
	for _, h := range entities {
		sig, ok, err := d.buildSignal(ctx, r, h)
		if err != nil {
			return fmt.Errorf("sim driver: %s: %w", r.Name, err)
		}
		if !ok {
			continue
		}
		if _, err := d.sim.Generate(ctx, *sig); err != nil {
			return fmt.Errorf("sim driver: %s: generate: %w", r.Name, err)
		}
	}
	return nil
}

// buildSignal 按映射表组装 Signal。返回 ok=false = 不建单（未知规则且未白名单）。
func (d *Driver) buildSignal(ctx context.Context, r store.Rule, h store.EntityHit) (*Signal, bool, error) {
	if m, ok := signalMappers[r.Name]; ok {
		return m(ctx, d, r, h)
	}
	// carry 白名单（§9.6）：命中标的 ∈ CarryWhitelist → carry_asset。
	if slices.Contains(d.cfg.CarryWhitelist, h.Symbol) {
		return d.carrySignal(ctx, r, h)
	}
	return nil, false, nil
}

// fundingHedgeSignal 组装 funding_hedge Signal（04-m3-spec §3.1.1 首行 + §9.2 诚实标注）。
func fundingHedgeSignal(ctx context.Context, d *Driver, r store.Rule, h store.EntityHit) (*Signal, bool, error) {
	// 现货/永续最新价 = LatestFacts(kind=ticker, venue, symbol) 最新一条。
	// 诚实标注（§9.2）：系统无现货 collector，ticker 即永续价；现货/永续腿存在性由
	// 门禁把关（>0），basis/现货腿差留真实执行层，M3 只验证 funding 机制。
	var ref float64
	if fs, err := d.st.LatestFacts(ctx, fact.KindTicker, h.Venue, h.Symbol); err == nil && len(fs) > 0 {
		ref = fs[0].Value
	}
	return &Signal{
		RuleName: r.Name, Kind: store.SimKindFundingHedge,
		Symbol: h.Symbol, Venue: h.Venue,
		RefPrice: ref, SpotPrice: ref, PerpPrice: ref,
		FundingAnn: h.Value, ExpectedSpread: 0, // 由 FundingAnn 回填（SignalToOrder）
		Notional: 0, Ts: d.now(),
	}, true, nil
}

// repoSignal 组装 repo Signal（04-m3-spec §3.1.1 第二行）：全局模式命中，单信号。
// L1 复审：当日回购年化 = 人工补录 reverse_repo 事实最新值（权威）；**无事实 → 不建单**
// （fail-closed，宁缺毋滥）——h.Value 对 KindCalendar 规则是"事件计数"（last_24h ≤ 1），
// 不是利率，用作价差兜底是单位错配（预期年化价差会显示 1.00% 这类计数，误导）。
func repoSignal(ctx context.Context, d *Driver, r store.Rule, h store.EntityHit) (*Signal, bool, error) {
	fs, err := d.st.LatestFacts(ctx, fact.KindReverseRepo, "", "")
	if err != nil {
		return nil, false, fmt.Errorf("sim driver: repo: latest reverse_repo fact: %w", err)
	}
	if len(fs) == 0 {
		return nil, false, nil // 无 reverse_repo 事实 → 不建单（无法得知当日回购利率）
	}
	spread := fs[0].Value
	return &Signal{
		RuleName: r.Name, Kind: store.SimKindRepo,
		// D-045：用事实真实 (venue,symbol)（reverse_repo 事实存 venue=sina/symbol=GC001）。
		// 勿硬编码——结算按 (kind,venue,symbol) 查 reverse_repo 事实，venue 错配则
		// 建仓后永不结算（practices #22：落单 venue/symbol 取事实真实值）。
		Symbol: fs[0].Symbol, Venue: fs[0].Venue,
		RefPrice: 100, ExpectedSpread: spread, FundingAnn: spread,
		Notional: 0, Ts: d.now(),
	}, true, nil
}

// carrySignal 组装 carry_asset Signal（§9.6）：标的已在白名单（调用方已校验）。
func (d *Driver) carrySignal(ctx context.Context, r store.Rule, h store.EntityHit) (*Signal, bool, error) {
	spread := h.Value // 命中值兜底（生息年化）
	if fs, err := d.st.LatestFacts(ctx, fact.KindDefiRate, h.Venue, h.Symbol); err == nil && len(fs) > 0 {
		spread = fs[0].Value
	}
	ref := 1.0 // 稳定币面值锚（sUSDe/USDe ≈ 1）；有 ticker 则覆盖
	if fs, err := d.st.LatestFacts(ctx, fact.KindTicker, h.Venue, h.Symbol); err == nil && len(fs) > 0 {
		ref = fs[0].Value
	}
	return &Signal{
		RuleName: r.Name, Kind: store.SimKindCarryAsset,
		Symbol: h.Symbol, Venue: d.venue(h.Venue),
		RefPrice: ref, ExpectedSpread: spread, FundingAnn: spread,
		Notional: 0, CarryWhite: true, Ts: d.now(),
	}, true, nil
}

// venue 归一 venue；空（全局命中）→ 默认模拟 venue。
func (d *Driver) venue(v string) string {
	if v == "" {
		return "sim_local"
	}
	return v
}

// RunSettleLoop 每 8h tick 结算 open funding 腿 + 每 7 tick 渲染周报 + 执行探针钩子；
// 阻塞至 ctx 取消。真实运行 = time.NewTicker(settleInterval)。
func (d *Driver) RunSettleLoop(ctx context.Context) error {
	t := time.NewTicker(settleInterval)
	defer t.Stop()
	return d.settleLoop(ctx, t.C)
}

// settleLoop 可注入 tick 通道的结算循环（测试注入固定 tick）。
func (d *Driver) settleLoop(ctx context.Context, ticks <-chan time.Time) error {
	n := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticks:
			if err := d.settleOnce(ctx); err != nil {
				d.log.Warn("sim settle once failed", "err", err)
			}
			n++
			if n%reportEveryTicks == 0 {
				if err := d.renderReport(ctx); err != nil {
					d.log.Warn("sim report render failed", "err", err)
				}
			}
			if d.Probe != nil {
				if err := d.Probe(ctx); err != nil {
					d.log.Warn("simtestnet probe failed", "err", err)
				}
			}
			// D-062 判定门① 测量引擎数据面：每 tick 落 equity 时点快照（8h 粒度）。
			// 独立于结算结果（测量要忠实记录每 tick 状态）；失败仅 warn 不阻断主循环，
			// 与 report 渲染同口径（测量是辅，结算/呈现是主）。
			if err := d.snapshotEquity(ctx); err != nil {
				d.log.Warn("sim equity snapshot failed", "err", err)
			}
		}
	}
}

// snapshotEquity 落 equity 时点快照（D-062，sim_equity_snapshots）。复用 GetSimAccount
// 口径（simapi/account.go 同款五数）：cash = GetSimAccount、realized = Σ ListSimPositions
// PnL、unrealized/market_value = Σ open 腿 dir×qty×cur（dir：long+1/short−1；ticker 缺失
// 该腿按 0 计不编造）。口径一致靠本方法 + account.go 各自独立实现（driver 不依赖 simapi，
// 避免逆向依赖——internal/sim 是 simapi 的下层）。返回 8h tick 快照，供 GetPerformanceReport
// 跨窗口 TWR/MWR + 判定门① 判定。
//
// [对抗测试锚点] D-062：删快照写入 → return_test 的 settle→快照断言必红。
func (d *Driver) snapshotEquity(ctx context.Context) error {
	acct, err := d.st.GetSimAccount(ctx)
	if err != nil {
		return fmt.Errorf("sim snapshot: account: %w", err)
	}
	positions, err := d.st.ListSimPositions(ctx, 10000, 0)
	if err != nil {
		return fmt.Errorf("sim snapshot: positions: %w", err)
	}
	realized := 0.0
	for _, p := range positions {
		realized += p.PnL
	}
	open, err := d.st.ListOpenSimPositions(ctx, "", "")
	if err != nil {
		return fmt.Errorf("sim snapshot: open: %w", err)
	}
	unrealized, marketValue := 0.0, 0.0
	for _, p := range open {
		fs, err := d.st.LatestFacts(ctx, fact.KindTicker, p.Venue, p.Symbol)
		if err != nil {
			return fmt.Errorf("sim snapshot: ticker %s@%s: %w", p.Symbol, p.Venue, err)
		}
		if len(fs) == 0 {
			continue // 行情缺失该腿按 0 计（不编造）
		}
		cur := fs[0].Value
		dir := 1.0
		if p.Side == store.SimSideShort {
			dir = -1
		}
		unrealized += (cur - p.RefPrice) * p.Qty * dir
		marketValue += dir * p.Qty * cur
	}
	equity := acct.Cash + marketValue
	return d.st.InsertEquitySnapshot(ctx, store.EquitySnapshot{
		Ts:          d.now(),
		Equity:      equity,
		Cash:        acct.Cash,
		Realized:    realized,
		Unrealized:  unrealized,
		MarketValue: marketValue,
	})
}

// settleFactKind 腿 kind → 结算数据面 kind（practices #13 结算侧：数据面按实体类型
// 分派，D-045）。funding_hedge→funding / carry_asset→defi_rate / repo→reverse_repo，
// 各自权威事实；未知 kind → (false) 跳过（腿仅经 BuildLegs 产生，未知 kind 不会
// 出现，防御性 fail-closed，不 panic 不误结算）。
func settleFactKind(kind string) (string, bool) {
	switch kind {
	case store.SimKindFundingHedge:
		return fact.KindFunding, true
	case store.SimKindCarryAsset:
		return fact.KindDefiRate, true
	case store.SimKindRepo:
		return fact.KindReverseRepo, true
	default:
		return "", false
	}
}

// settleOnce 执行一轮结算：列出全部 open funding 腿 → 按 (kind,symbol,venue) 分组 →
// 每组按 settleFactKind 分派数据面，取 LatestFacts 最新值结算；无事实/未知 kind
// skip+warn。D-045：结算数据面按腿 kind 分派（此前只查 funding 事实，carry/repo 腿
// 建了仓也永不生息）。
//
// [对抗测试锚点] §9.3 S2 + D-045：①按 (kind,symbol,venue) 分组（不按 symbol 合并）→
// 删除分组中的 venue 维度 → TestSettleByVenue 跨 venue 污染断言必红；②kind 分派 →
// 删除 settleFactKind 分派/改回只查 KindFunding → TestSettleDispatchByKind 必红。
func (d *Driver) settleOnce(ctx context.Context) error {
	legs, err := d.st.ListOpenSimPositions(ctx, "", "")
	if err != nil {
		return fmt.Errorf("sim settle: list open: %w", err)
	}
	seen := map[[3]string]bool{}
	var keys [][3]string
	for _, l := range legs {
		if !l.Funding {
			continue
		}
		key := [3]string{l.Kind, l.Symbol, l.Venue}
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool { // 确定性日志顺序
		for c := 0; c < 3; c++ {
			if keys[i][c] != keys[j][c] {
				return keys[i][c] < keys[j][c]
			}
		}
		return false
	})
	for _, key := range keys {
		kind, sym, venue := key[0], key[1], key[2]
		factKind, ok := settleFactKind(kind)
		if !ok {
			d.log.Warn("sim settle: unknown leg kind, skip", "kind", kind, "symbol", sym, "venue", venue)
			continue
		}
		fs, err := d.st.LatestFacts(ctx, factKind, venue, sym)
		if err != nil {
			return fmt.Errorf("sim settle: latest %s %s@%s: %w", factKind, sym, venue, err)
		}
		if len(fs) == 0 {
			d.log.Warn("sim settle: no fact, skip", "kind", factKind, "symbol", sym, "venue", venue)
			continue
		}
		rate := fs[0].Value // 真实市场公开事实（§9.0 裁决：非 testnet）
		if _, err := d.sim.SettleFunding(ctx, kind, sym, venue, rate); err != nil {
			return fmt.Errorf("sim settle: settle %s %s@%s: %w", kind, sym, venue, err)
		}
	}
	return nil
}
