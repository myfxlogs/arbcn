// GetPerformanceReport 阶段 0 判定门① 测量（D-062）。切出本文件防 service.go 超 450 行。
// 只读编排：equity 快照 + 外部现金流 + 订单 + funding 历史 → TWR/MWR + 判定门① 判定
// + 环境条件（D-061 环境-策略分离）。零写路径（判定门① 只测量不自动执行——进阶段 A
// 仍人工决策）；零网络零密钥（D-010）。计算全部走 internal/sim 纯函数（return.go），
// 本文件只做数据面编排与组装（不重复算账逻辑，P3 单源）。
package simapi

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"arbcn/internal/fact"
	"arbcn/internal/sim"
	simv1 "arbcn/internal/simapi/gen/arbcn/sim/v1"
	"arbcn/internal/store"
)

// GetPerformanceReport 判定门① 跨窗口测量（窗口 = 最近 30 天，D-058）。
// 窗口未满 / 快照不足 → status=pending（前向验证进行中，诚实不提前判定）。
// 窗口零成交 → env_no_window（D-061：零成交测环境非策略；拒单 > 0 标「有机会未进场」）。
// 有成交 → 按 TWR 年化 vs 判定线 4.0% 判 pass/watch/fail。
func (s *Service) GetPerformanceReport(ctx context.Context, _ *connect.Request[simv1.GetPerformanceReportRequest]) (*connect.Response[simv1.GetPerformanceReportResponse], error) {
	resp := &simv1.GetPerformanceReportResponse{
		BaselineLow:    sim.GateBaselineLow,
		BaselineHigh:   sim.GateBaselineHigh,
		FrictionMargin: sim.GateFrictionPct,
		GateThreshold:  sim.GateThreshold,
	}
	now := s.now()
	since := now.Add(-time.Duration(sim.GateWindowDays) * 24 * time.Hour)

	// equity 快照（ASC，链乘顺序）。
	snaps, err := s.st.ListEquitySnapshots(ctx, since, 0)
	if err != nil {
		return nil, storeErr(err)
	}
	startTs, endTs := since, now
	if len(snaps) >= 2 {
		startTs, endTs = snaps[0].Ts, snaps[len(snaps)-1].Ts
	}

	// 外部现金流 = 全部 capital_in（limit 大防截断：capital_in 是账本最早一笔，最近
	// 100 条窗口可能漏掉，D-062 审计修正）。当前只有首启入金 → TWR = MWR = 简单年化。
	rawFlows, err := s.st.ListCashFlows(ctx, 10000, 0)
	if err != nil {
		return nil, storeErr(err)
	}
	flows := []sim.ExternalFlow{}
	for _, f := range rawFlows {
		if f.Kind == store.CashKindCapitalIn {
			flows = append(flows, sim.ExternalFlow{Ts: f.Ts, Amount: f.Amount})
		}
	}

	// TWR / MWR（纯函数；错误映射到判定门① 状态而非编造数值）。
	var twrErr error
	twr, days := 0.0, 0.0
	if len(snaps) >= 2 {
		twr, days, twrErr = sim.TwrAnnualized(snaps, flows)
		if twrErr == nil {
			var annErr error
			twr, annErr = sim.Annualize(twr, days)
			if annErr != nil {
				twrErr = annErr
			}
		}
	}
	mwr, mwrErr := 0.0, sim.ErrInsufficientData
	if len(snaps) >= 2 {
		last := snaps[len(snaps)-1]
		mwr, mwrErr = sim.MwrAnnualized(flows, last.Equity, last.Ts)
	}

	// 窗口内订单统计（成交 = filled+closed；拒单 = rejected 负样本）。
	orders, err := s.st.ListSimOrders(ctx, 10000, 0)
	if err != nil {
		return nil, storeErr(err)
	}
	orderCount, rejected := 0, 0
	for _, o := range orders {
		if o.Ts.Before(startTs) || o.Ts.After(endTs) {
			continue
		}
		switch o.Status {
		case store.SimStatusFilled, store.SimStatusClosed:
			orderCount++
		case store.SimStatusRejected:
			rejected++
		}
	}

	// 环境条件（D-061：当期 funding 统计随结果留档；只覆盖监控面内，诚实标注）。
	envFacts, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindFunding, From: startTs, Limit: 100000})
	if err != nil {
		return nil, storeErr(err)
	}
	env := sim.ComputeEnvironmentStats(envFacts)

	// D-063 判定门① 可信度自检（防「判定门自己骗人」）：快照缺口/数据损坏会让 TWR
	// 静默失真但 gate 照判。覆盖率与恒等式校验先行，不足 → 判定不采信（DATA_ANOMALY），
	// 部分缺口 → 附加警示不覆盖判定。dice：判定用数据面自己出题自己答卷，须自证可信。
	coverage := sim.SnapshotCoverage(days, len(snaps))
	integrityErr := ""
	if i, detail := sim.ValidateSnapshotIntegrity(snaps); i >= 0 {
		integrityErr = fmt.Sprintf("快照[%d] %s", i, detail)
	}
	overrideStatus, trustNote := sim.GateTrustQualifier(days, coverage, integrityErr)

	// 单位统一（P3 单源，防「判定门自己骗人」的单位错配形态）：TwrAnnualized/
	// MwrAnnualized 返回小数（0.708 = 70.8%），判定门① 阈值与 RPC 字段为百分点点数
	// （GateThreshold = 4.0 即 4.0%）。编排层 ×100 一次，gate 判定与 RPC 返回共用同
	// 一口径；纯函数保持数学惯例（fraction）。此前不经换算直接比 0.7 vs 4.0 → 永远 FAIL。
	twrPct, mwrPct := twr*100, mwr*100

	status, note := "", ""
	if overrideStatus != "" {
		status, note = overrideStatus, trustNote
	} else {
		status, note = sim.EvaluateGate(days, twrPct, mwrPct, orderCount, rejected, env.HighWindowEvents, twrErr, mwrErr)
		if trustNote != "" {
			note = trustNote + "；" + note
		}
	}

	resp.WindowDays = days
	resp.TwrAnnualized = twrPct
	resp.MwrAnnualized = mwrPct
	resp.Status = status
	resp.StatusNote = note
	resp.FundingMedian = env.FundingMedian
	resp.FundingMax = env.FundingMax
	resp.HighWindowEvents = int32(env.HighWindowEvents)
	resp.TradablePairs = int32(env.TradablePairs)
	resp.OrderCount = int32(orderCount)
	resp.RejectedCount = int32(rejected)
	resp.SnapshotCount = int32(len(snaps))
	resp.ExpectedSnapshots = int32(sim.ExpectedSnapshots(days))
	resp.SnapshotCoverage = coverage
	resp.StartTsMs = startTs.UnixMilli()
	resp.EndTsMs = endTs.UnixMilli()
	return connect.NewResponse(resp), nil
}
