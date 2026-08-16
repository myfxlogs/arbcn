// 判定门① 测量引擎纯函数（D-062）：TWR/MWR 跨窗口收益 + 判定门① 判定 + 环境条件。
// 阶段 0 落地——让「跨窗口 paper 收益 ≥ 诚实基线 3.2-3.7% + 摩擦余量 0.3% 才进阶段 A」
// 从「运行期定口径」变成可机械判定的测量（P4）。零网络零密钥（D-010）：只有纯数学。
// 本文件所有决策参数（窗口/基线/摩擦/高费率档）为具名常量，改动走 D#（AGENTS.md 门禁纪律）。
package sim

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// 判定门① 决策参数（D-058/D-062，改走 D#）。单位：%（pct_annualized 百分点点数）。
const (
	GateWindowDays   = 30.0 // 跨窗口天数（D-058：跨 30 天窗口）
	GateBaselineLow  = 3.2  // 诚实基线下限 %（D-026）
	GateBaselineHigh = 3.7  // 诚实基线上限 %（D-026）
	GateFrictionPct  = 0.3  // 摩擦余量 %（D-046 业主核实普通主户双 taker）
	gateHighFunding  = 15.0 // 高费率窗口档 %（D-016 15% 档；pct_annualized 点数）
)

// GateThreshold 判定线 = 基线上限 + 摩擦余量（PASS 需覆盖「无风险替代 + 实盘摩擦」）。
const GateThreshold = GateBaselineHigh + GateFrictionPct // 4.0

// 判定门① 状态值域（D-062）。
const (
	GatePending     = "pending"        // 首快照至今 < 30 天，数据不足，前向验证进行中
	GatePass        = "pass"           // TWR ≥ 判定线 4.0% → 可进阶段 A（真金冒烟 ~1 万）
	GateWatch       = "watch"          // 基线区间内未覆盖摩擦 → 继续阶段 0 积累
	GateFail        = "fail"           // < 基线下限或负 → 止投（D-058「为负/不足即止投」）
	GateEnvNoWindow = "env_no_window"  // 窗口零动作 → 环境无机会，不判失败（D-061 环境-策略分离）
	GateDataAnomaly = "data_anomaly"   // 数据异常（equity≤0 等）→ 不编造（practices #7）
)

// 测量引擎哨兵错误：数据不足 / 数据异常（RPC 映射为 GatePending / GateDataAnomaly）。
var (
	ErrInsufficientData = errors.New("sim: insufficient equity data")
	ErrDataAnomaly      = errors.New("sim: equity data anomaly")
)

// ExternalFlow 外部资金流（TWR/MWR 分段与 IRR 输入）。Amount 正 = 流入（入金），
// 负 = 流出（出金）。当前模拟盘只有首启 capital_in（+capital）；阶段 A 真金出入金后
// 扩成 capital_in/capital_out 序列——引擎现在做好，为真实账本直接复用。
type ExternalFlow struct {
	Ts     time.Time
	Amount float64
}

// TwrAnnualized 时间加权年化（TWR）。输入 = 快照（ts ASC）+ 外部资金流。
// 返回 (twr, days, err)：twr 为窗口总收益，days 为首尾快照天数（年化用）。
//
// 无窗口内外部流（当前 paper 唯一情形：capital_in 落在首快照前）→ 精确等于简单
// 期初期末年化 E_last/E_first − 1。有窗口内外部流（阶段 A 真金出入金）→ 按快照区间
// Dietz 调整链乘：段收益 r = (E_i − Σout)/(E_{i−1} + Σin) − 1，几何连乘——消除资金
// 进出对收益率的污染（TWR 测策略）。窗口内快照 < 2 或 days ≤ 0 → ErrInsufficientData。
func TwrAnnualized(snaps []store.EquitySnapshot, flows []ExternalFlow) (twr, days float64, err error) {
	if len(snaps) < 2 {
		return 0, 0, ErrInsufficientData
	}
	first, last := snaps[0].Ts, snaps[len(snaps)-1].Ts
	days = last.Sub(first).Hours() / 24
	if days <= 0 || snaps[len(snaps)-1].Equity <= 0 || snaps[0].Equity <= 0 {
		return 0, 0, ErrDataAnomaly
	}
	// 窗口内外部流（(first, last]）——流入与流出分开。
	var inF, outF []ExternalFlow
	for _, f := range flows {
		if f.Ts.After(first) && !f.Ts.After(last) {
			if f.Amount >= 0 {
				inF = append(inF, f)
			} else {
				outF = append(outF, f)
			}
		}
	}
	if len(inF) == 0 && len(outF) == 0 {
		return snaps[len(snaps)-1].Equity/snaps[0].Equity - 1, days, nil
	}
	// 有窗口内外部流：逐快照区间 Dietz 调整链乘。
	prod := 1.0
	for i := 1; i < len(snaps); i++ {
		prev, cur := snaps[i-1], snaps[i]
		start, end := prev.Equity, cur.Equity
		for _, f := range inF {
			if f.Ts.After(prev.Ts) && !f.Ts.After(cur.Ts) {
				start += f.Amount
			}
		}
		for _, f := range outF {
			if f.Ts.After(prev.Ts) && !f.Ts.After(cur.Ts) {
				end -= f.Amount
			}
		}
		if start <= 0 || end < 0 {
			return 0, 0, ErrDataAnomaly
		}
		prod *= end / start
	}
	if prod <= 0 {
		return 0, 0, ErrDataAnomaly
	}
	return prod - 1, days, nil
}

// MwrAnnualized 资金加权年化（MWR / IRR）：外部现金流 + 期末 equity，求年化 r 使
// Σ cfᵢ(1+r)^((T−tᵢ)/365) = E_T（未来值形式）。二分收敛（r ∈ (−0.9999, 1000]）。
// 无任何流入或期末 equity ≤ 0 → ErrDataAnomaly（不编造）。当前 paper 只有首启
// capital_in → MWR = TWR = 期初期末年化（诚实标注：无外部进出二者一致）；阶段 A
// 真金出入金后二者分叉（TWR 测策略、MWR 测资金使用效率）。
func MwrAnnualized(flows []ExternalFlow, endEquity float64, endTs time.Time) (float64, error) {
	var inflow bool
	for _, f := range flows {
		if f.Amount > 0 {
			inflow = true
		}
	}
	if !inflow || endEquity <= 0 {
		return 0, ErrDataAnomaly
	}
	lo, hi := -0.9999, 1000.0
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		v := mwrFutureValue(mid, flows, endTs)
		if v < endEquity {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2, nil
}

// mwrFutureValue 未来值：Σ cfᵢ(1+r)^years。years 以年计（365 天）。
func mwrFutureValue(r float64, flows []ExternalFlow, endTs time.Time) float64 {
	v := 0.0
	for _, f := range flows {
		years := endTs.Sub(f.Ts).Hours() / (365 * 24)
		v += f.Amount * math.Pow(1+r, years)
	}
	return v
}

// Annualize 把窗口总收益年化：(1+r)^(365/days) − 1。days ≤ 0 → ErrInsufficientData
// （窗口未满，年化不可靠——由调用方标 PENDING 而非给误导数值）。
func Annualize(twr, days float64) (float64, error) {
	if days <= 0 {
		return 0, ErrInsufficientData
	}
	return math.Pow(1+twr, 365/days) - 1, nil
}

// EnvironmentStats 窗口环境条件（D-061 测量口径：记录当期环境随结果留档）。
type EnvironmentStats struct {
	FundingMedian   float64 // 窗口内 funding 中位数 %（pct_annualized）
	FundingMax      float64 // 窗口内 funding max %
	HighWindowEvents int    // 窗口内 funding ≥ 15% 档的时段数（D-016 高费率窗口）
	TradablePairs   int     // 可交易面 = distinct (venue, symbol) 对数（只覆盖监控面内）
}

// ComputeEnvironmentStats 从窗口内 funding 历史事实算环境条件。facts 为空 →
// 返回零值（funding 中位数未知 = 0 + HighWindowEvents 0，诚实标注「数据面不足」由
// 调用方判断）。funding facts 值为 pct_annualized 百分点点数（15 = 15% 年化）。
func ComputeEnvironmentStats(fs []fact.Fact) EnvironmentStats {
	if len(fs) == 0 {
		return EnvironmentStats{}
	}
	vals := make([]float64, 0, len(fs))
	seen := map[[2]string]bool{}
	max := math.Inf(-1)
	high := 0
	for _, f := range fs {
		if f.Value > max {
			max = f.Value
		}
		if f.Value >= gateHighFunding {
			high++
		}
		vals = append(vals, f.Value)
		seen[[2]string{f.Venue, f.Symbol}] = true
	}
	sort.Float64s(vals)
	median := vals[len(vals)/2]
	if len(vals)%2 == 0 {
		median = (vals[len(vals)/2-1] + vals[len(vals)/2]) / 2
	}
	if max == math.Inf(-1) {
		max = 0
	}
	return EnvironmentStats{FundingMedian: median, FundingMax: max, HighWindowEvents: high, TradablePairs: len(seen)}
}

// 判定门① 可信度参数（D-063 防自欺：判定门① 的测量数据面自己会骗人——快照缺口 /
// 数据损坏会让 TWR 静默失真，gate 却照判 PASS/FAIL）。
const (
	GateMinCoverage  = 0.9 // 快照覆盖率下限：<90% 窗口测量不可信（判定不采信）
	gateIntegrityTol = 1e-6
)

// gateSnapPerDay 8h tick → 3 份/天（运行时计算：settleInterval.Hours() 非常量）。
var gateSnapPerDay = 24.0 / settleInterval.Hours()

// SnapshotCoverage 窗口内实际快照数 / 期望数（连续运行基线 = 8h tick）。
// 覆盖率 < 1 = 服务中断/落点缺失 → TWR 链乘跨缺口段被静默跳过（段边界被改）→ 测量可信度降低。
// windowDays ≤ 0 或 snapCount ≤ 0 → 0（无可采信数据）。
func SnapshotCoverage(windowDays float64, snapCount int) float64 {
	if windowDays <= 0 || snapCount <= 0 {
		return 0
	}
	expected := windowDays * gateSnapPerDay
	if expected <= 0 {
		return 0
	}
	cov := float64(snapCount) / expected
	if cov > 1 {
		cov = 1 // 超过基线（如补测）不超 1：覆盖率是「缺口」度量不是「超额」度量
	}
	return cov
}

// ExpectedSnapshots 窗口期望快照数（按 8h tick 连续运行基线；向上取整）。前端展示
// 「实际/期望」覆盖率口径。windowDays ≤ 0 → 0。
func ExpectedSnapshots(windowDays float64) int {
	if windowDays <= 0 {
		return 0
	}
	return int(math.Ceil(windowDays * gateSnapPerDay))
}

// ValidateSnapshotIntegrity 快照数据完整性校验（防口径漂移/存储损坏骗过 gate）：
//   - ts 单调递增（段边界有序）；
//   - equity ≈ cash + market_value（driver 写入口径的恒等式，损坏即暴露）。
//
// 返回第一个坏快照索引与原因（全好 → -1, ""）。
func ValidateSnapshotIntegrity(snaps []store.EquitySnapshot) (int, string) {
	for i := 1; i < len(snaps); i++ {
		if !snaps[i].Ts.After(snaps[i-1].Ts) {
			return i, "ts 非单调（段边界乱序）"
		}
	}
	for i := range snaps {
		diff := math.Abs(snaps[i].Equity - (snaps[i].Cash + snaps[i].MarketValue))
		scale := math.Max(1, math.Abs(snaps[i].Equity))
		if diff > gateIntegrityTol*scale {
			return i, "equity ≠ cash + market_value（恒等式破坏）"
		}
	}
	return -1, ""
}

// GateTrustQualifier 判定门① 可信度判定（D-063）：覆盖率不足/数据损坏 → 判定不采信，
// 映射 DATA_ANOMALY（不编造数值，practices #7）；部分缺口 → 附加警示不覆盖判定。
// 返回 (overrideStatus, trustNote)：overrideStatus 非空 = 覆盖 EvaluateGate 结果。
// windowDays < 30 → 窗口未成熟，判定门① 本就 PENDING，覆盖率检查不生效（防把
// 「还没数据」误判成「数据坏了」）；但数据完整性损坏任何时候都不采信。
func GateTrustQualifier(windowDays, coverage float64, integrityErr string) (string, string) {
	if integrityErr != "" {
		return GateDataAnomaly, "快照数据完整性损坏（" + integrityErr + "），判定门① 不予采信"
	}
	if windowDays < GateWindowDays {
		return "", ""
	}
	if coverage <= 0 {
		return GateDataAnomaly, "窗口内无 equity 快照，测量数据面未落点，判定门① 不予采信"
	}
	if coverage < GateMinCoverage {
		return GateDataAnomaly, fmt.Sprintf("快照覆盖率 %.0f%%（<%.0f%%），窗口测量跨缺口不可信，判定门① 不予采信", coverage*100, GateMinCoverage*100)
	}
	if coverage < 1 {
		return "", fmt.Sprintf("快照覆盖率 %.0f%%（连续运行基线 100%%），结果仅供参考", coverage*100)
	}
	return "", ""
}

// EvaluateGate 判定门①（D-062）：
//   - 数据不足（twr/mwr 计算错误或窗口未满 30 天）→ PENDING / DATA_ANOMALY。
//   - 窗口零动作（无 filled/closed 成交）→ ENV_NO_WINDOW（D-061：零成交测的是环境
//     不是策略；但 rejected > 0 = 有信号被门禁拒，属「有机会未进场」观察信号 → WATCH）。
//   - 有成交 → 按 TWR 年化 vs 判定线：≥ 4.0% PASS / ≥ 3.2% WATCH / 其余 FAIL。
//
// 参数：windowDays（首尾快照天数）、twrAnn/mwrAnn（年化值）、orders/rejected（窗口内
// 成交/拒单数）、highWindowEvents（环境高费率时段数）、twrErr/mwrErr（计算错误）。
func EvaluateGate(windowDays float64, twrAnn, mwrAnn float64, orders, rejected, highWindowEvents int, twrErr, mwrErr error) (string, string) {
	if twrErr != nil || mwrErr != nil {
		if errors.Is(twrErr, ErrDataAnomaly) || errors.Is(mwrErr, ErrDataAnomaly) {
			return GateDataAnomaly, "数据异常（equity/现金流不可算），不编造数值"
		}
		return GatePending, "数据不足（窗口未满或快照 < 2），前向验证进行中"
	}
	if windowDays < GateWindowDays {
		return GatePending, "跨窗口未满 30 天，前向验证进行中（诚实标注：不提前判定）"
	}
	if orders == 0 {
		if rejected > 0 {
			return GateWatch, fmt.Sprintf("窗口零成交但有 %d 个拒单（信号被门禁拒），属「有机会未进场」观察信号而非环境无机会", rejected)
		}
		note := "窗口零成交 → 环境无机会（测的是环境不是策略，D-061）；零单是正确输出（宁缺毋滥 D-019）"
		if highWindowEvents > 0 {
			note += fmt.Sprintf("；注意环境统计显示 %d 个 ≥15%% 高费率时段却未成交，需排查（策略漏窗？）", highWindowEvents)
		}
		return GateEnvNoWindow, note
	}
	if twrAnn >= GateThreshold {
		note := "TWR 年化已覆盖诚实基线上限 + 摩擦余量，可进阶段 A（真金冒烟 ~1 万 RMB，摩擦后真实盈利唯一裁决）"
		if highWindowEvents > 0 {
			note += fmt.Sprintf("；警示：窗口含 %d 个 ≥15%% 高费率时段，PASS 可能含环境红利而非策略能力——阶段 A 摩擦冒烟兜底", highWindowEvents)
		}
		if orders < 3 {
			note += fmt.Sprintf("；警示：窗口仅 %d 单，小样本 PASS 置信度有限，建议小额冒烟验证", orders)
		}
		return GatePass, note
	}
	if twrAnn >= GateBaselineLow {
		return GateWatch, "TWR 年化在诚实基线区间内但未覆盖摩擦余量，继续阶段 0 积累（不进阶段 A）"
	}
	return GateFail, "TWR 年化低于诚实基线下限（或为负），止投信号（D-058：为负/不足即止投）"
}
