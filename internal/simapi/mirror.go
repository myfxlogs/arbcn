// 镜像下单接线（D-098 测试网执行层）：ConfirmSimOrder 本地成交前对 testnet/demo venue
// 逐腿镜像下单 → 回读成交 → 落 sim_order_executions（best-effort，execution 成败不影响
// 本地成交；D-037 本地 sim 仍是 PnL 大脑，本表只记录「执行机制验证」对账数据）。
// 本文件仍零网络零密钥：只经注入的 Executor 接口与 store，不直接触碰网络/key
// （domains_test TestNoRealTradeTokens 把关）。
package simapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"arbcn/internal/sim"
	"arbcn/internal/simtestnet"
	"arbcn/internal/store"
)

// mirrorBudget 整段镜像执行预算（用户确认点击不卡死；PlaceOrder 内部各带 10s 超时，
// 这里再包一层总闸——超时执行收敛为 ExecStatusError 记录，不阻断本地成交）。
const mirrorBudget = 20 * time.Second

// mirrorToExec 在本地 AcceptSimOrder 前镜像下单（D-098，best-effort）。返回 "" = 镜像关
// （Exec 未注入 / ExecVenue 未配 / 非 funding_hedge / venue 非法——宁缺毋滥，M3-c 零回归）。
// 仅 funding_hedge 在 testnet 有可放腿：
//   - okx_demo：现货 + 永续双放（OKX demo 有现货，完整 delta 中性对冲可镜像）；
//   - binance_testnet：仅永续腿（USDT-M 期货 testnet 无现货，spot 腿跳过记 note）。
//
// 逐腿 PlaceOrder → 回读成交 → InsertSimExecution 落执行记录；网络/拒单/落库失败都收敛
// 进执行记录与返回摘要（附订单 note），不阻断本地成交。
func (s *Service) mirrorToExec(ctx context.Context, o store.SimOrder) string {
	if s.Exec == nil || s.cfg.ExecVenue == "" {
		return ""
	}
	if o.Kind != store.SimKindFundingHedge {
		return "" // carry/repo 无 testnet 可放腿（稳定币/GC001 无模拟合约）
	}
	if s.cfg.ExecVenue != simtestnet.VenueOKXDemo && s.cfg.ExecVenue != simtestnet.VenueBinanceTestnet {
		return ""
	}
	legs := sim.BuildLegs(o, s.now())
	mctx, cancel := context.WithTimeout(ctx, mirrorBudget)
	defer cancel()

	var parts []string
	for _, leg := range legs {
		legName := "spot"
		if leg.Funding {
			legName = "perp"
		}
		// binance_testnet 只有永续（USDT-M 期货无现货）：spot 腿跳过，记录原因（假钱无害）。
		if s.cfg.ExecVenue == simtestnet.VenueBinanceTestnet && legName == "spot" {
			parts = append(parts, "binance_testnet spot 腿跳过（USDT-M 期货无现货）")
			continue
		}
		res, err := s.Exec.PlaceOrder(mctx, simtestnet.ExecOrder{
			OrderID: o.ID, Venue: s.cfg.ExecVenue, Symbol: leg.Symbol, Side: leg.Side,
			Kind: o.Kind, Leg: legName, Qty: leg.Qty, RefPrice: leg.RefPrice,
		})
		status := res.Status
		if status == "" {
			status = simtestnet.ExecStatusError
		}
		if err != nil {
			status = simtestnet.ExecStatusError
			res.Note = err.Error()
		}
		exchSym := res.Symbol
		if exchSym == "" {
			exchSym = leg.Symbol // 执行层 error 路径未回填交易所 instrument 时用基础标的
		}
		if _, ierr := s.st.InsertSimExecution(mctx, store.SimExecution{
			OrderID: o.ID, Leg: legName, Venue: s.cfg.ExecVenue,
			ExchangeOrderID: res.ExchangeOrderID, Symbol: exchSym, Side: leg.Side,
			Qty: res.Qty, FillPrice: res.FillPrice, FillQty: res.FillQty,
			Status: status, Note: res.Note,
		}); ierr != nil {
			// 落库失败：不阻断本地成交（best-effort），摘要可见。
			status = simtestnet.ExecStatusError
			res.Note = "execution 落库失败: " + ierr.Error()
		}
		parts = append(parts, fmt.Sprintf("%s %s=%s", s.cfg.ExecVenue, legName, status))
	}
	if len(parts) == 0 {
		return ""
	}
	return "｜镜像下单: " + strings.Join(parts, "; ")
}
