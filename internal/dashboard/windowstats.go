// 7d 费率窗口统计纯函数（D-064）：滚动 7d funding 环境 →「当前是否处于可交易
// 窗口」判据（D-061 候选落地，backpack-basis-trading-monitor 借鉴）。只读决策
// 支持（dashboard 域，D-046 oppcalc 同模式），零网络零密钥（D-010）。
// 本文件所有判据参数（窗口/高费率档/占比）为具名常量，改动走 D#（AGENTS.md 门禁纪律）。
package dashboard

import (
	"fmt"
	"math"

	"arbcn/internal/fact"
)

// 7d 费率窗口判据参数（D-064，改走 D#）。单位：%（pct_annualized 百分点点数）。
const (
	FundingWindowDays  = 7.0  // 窗口天数（D-064）
	WindowTierHigh     = 15.0 // 高费率窗口档（D-016 15% 档，与 sim 域 gateHighFunding 同源）
	WindowPositiveShare = 0.9 // 可交易：正费率占比 ≥90%（「持续为正」判据，backpack 系借鉴）
	WindowWatchShare    = 0.5 // 临界：正费率占比 ≥50%（正负交替）
	WindowMinSamples    = 3   // 样本下限：少于 = 附加「样本过少仅供参考」
)

// 窗口判据类值域（D-064）。
const (
	WindowHigh     = "high"     // 高费率窗口档 active（D-016 15–30% 档）
	WindowTradable = "tradable" // 可交易窗口（basis/carry 环境可行）
	WindowWatch    = "watch"    // 临界窗口（正负交替，观察）
	WindowNot      = "not"      // 非窗口（basis 不可行，宁缺毋滥 D-019）
)

// FundingWindowStats 7d 费率窗口统计。Min/Max/Mean 为 pct_annualized 百分点点数，
// PositiveShare 为 0-1 占比。Class 由 ClassifyFundingWindow 判定。
type FundingWindowStats struct {
	Count         int
	Min, Max, Mean float64
	PositiveShare float64
	Class         string
	Note          string
}

// ComputeFundingWindowStats 从 funding facts 算窗口统计。facts 空 → Count=0、
// 其余零值 + Class=not + Note「无数据」（不编造，practices #7）。样本 < 3 → Note
// 前置「样本过少仅供参考」（诚实标注，不虚称窗口可信）。
func ComputeFundingWindowStats(fs []fact.Fact) FundingWindowStats {
	if len(fs) == 0 {
		return FundingWindowStats{Class: WindowNot, Note: "无数据（窗口内无 funding 落点），窗口判定不可用"}
	}
	min, max := math.Inf(1), math.Inf(-1)
	sum, pos := 0.0, 0
	for _, f := range fs {
		v := f.Value
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
		if v >= 0 {
			pos++
		}
	}
	s := FundingWindowStats{
		Count:         len(fs),
		Min:           min,
		Max:           max,
		Mean:          sum / float64(len(fs)),
		PositiveShare: float64(pos) / float64(len(fs)),
	}
	s.Class, s.Note = ClassifyFundingWindow(s)
	if s.Count < WindowMinSamples {
		s.Note = fmt.Sprintf("样本过少（%d 份），仅供参考；%s", s.Count, s.Note)
	}
	return s
}

// ClassifyFundingWindow 窗口判据（D-064）：
//   high      7d 均值 ≥ 15%（D-016 高费率窗口档 active）
//   tradable  正费率占比 ≥ 90% 且均值 ≥ 0（费率持续为正，basis/carry 环境可行）
//   watch     正费率占比 ≥ 50% 且均值 ≥ 0（费率正负交替，可交易性临界）
//   not       其余（正费率占比 < 50% 或均值 < 0，basis 不可行，宁缺毋滥 D-019）
// 输入 Count=0 → not +「无数据」。note 含判据关键数值（自明，practices #35 同源）。
func ClassifyFundingWindow(s FundingWindowStats) (string, string) {
	if s.Count == 0 {
		return WindowNot, "无数据（窗口内无 funding 落点），窗口判定不可用"
	}
	if s.Mean >= WindowTierHigh {
		return WindowHigh, fmt.Sprintf("7d 均值 %.2f%% ≥ %v%%：当前处于高费率窗口档（D-016 15–30%% 档），机会信号应密集", s.Mean, WindowTierHigh)
	}
	if s.PositiveShare >= WindowPositiveShare && s.Mean >= 0 {
		return WindowTradable, fmt.Sprintf("正费率占比 %.0f%%（≥%d%%）且均值 %.2f%% ≥ 0：费率持续为正，basis/carry 环境可行", s.PositiveShare*100, int(WindowPositiveShare*100), s.Mean)
	}
	if s.PositiveShare >= WindowWatchShare && s.Mean >= 0 {
		return WindowWatch, fmt.Sprintf("正费率占比 %.0f%%（%d–%d%%）：费率正负交替，可交易性临界，观察", s.PositiveShare*100, int(WindowWatchShare*100), int(WindowPositiveShare*100)-1)
	}
	return WindowNot, fmt.Sprintf("正费率占比 %.0f%%（<%d%%）或均值为负：费率不稳定/为负，basis 交易不可行（宁缺毋滥 D-019）", s.PositiveShare*100, int(WindowWatchShare*100))
}
