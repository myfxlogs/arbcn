package sim

import (
	"arbcn/internal/fact"
	"arbcn/internal/rmb"
)

// 资金费率周期：8h 三班（04-m3-spec §3.2）。
const (
	PeriodsPerDay = 3
	DaysPerYear   = 365
)

// SettleFundingPnl 资金费率单期结算：pnl = funding_rate × notional（04-m3-spec §3.2）。
// rate 为结算期间资金费率（与 notional 币种一致，如 8h 周期费率），notional 为名义。
// 纯函数：对抗测试锚点（§11）——删除 `rate * notional` 的乘法 → 累计 PnL 断言必红。
func SettleFundingPnl(rate, notional float64) float64 {
	return rate * notional
}

// Per8hRate 年化费率（百分点点数，10.95 = 10.95%）→ 单期（8h）**分数**费率：
// 先 ÷100 把点数转分数费率，再 ÷（3 班/日 × 365 日）。funding 周期 8h 三班（§3.2）。
// 追溯复审 H1：原实现缺 ÷100，把点数当分数 → 结算 PnL 虚高 100 倍，模拟盈亏失真。
func Per8hRate(annualized float64) float64 {
	return annualized / 100 / (PeriodsPerDay * DaysPerYear)
}

// RMBDayEnd 日终 RMB 折算（04-m3-spec §3.2"复用 M2-b 折算函数"，不重复实现）：
// 复用 internal/rmb.Convert 把 USD 计价年化费率折算为 RMB 净收益年化费率
// （USD 年化 − 年化人民币升值，D-023 口径），再按持有天数换算为 RMB PnL：
//
//	RMB PnL = rmbRate × notional × 持有天数/365
//
// fx 为当日 USDCNH 事实（nil = 汇率缺失 → 回退 USD 原值口径，与 rmb 同语义）；
// appreciation 为年化人民币升值率（调用方经 rmb.AnnualizedRMBAppreciation 计算，
// 百分点点数，3.0 = 3 点）。
// 追溯复审 H1：conv[0].RMBValue 是百分点点数（R6#1 刻度），乘名义前必须 ÷100 转
// 分数费率——原实现把点数当分数，RMB PnL 虚高 100 倍。
func RMBDayEnd(annualizedUSD, notional, days float64, fx *fact.Fact, appreciation float64) float64 {
	conv := rmb.Convert([]fact.Fact{{
		Kind: fact.KindFunding, Value: annualizedUSD, Unit: fact.UnitPctAnnualized,
	}}, fx, appreciation)
	return conv[0].RMBValue / 100 * notional * days / DaysPerYear
}
