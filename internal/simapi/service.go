// Package simapi：模拟执行 ConnectRPC 服务（04-m3-spec §10 M3-c，独立域 arbcn.sim.v1）。
// 5 个 RPC：ListSimOrders / ConfirmSimOrder / ListSimPositions / GetSimReport /
// GetTestnetAccounts（D-040 测试网账户区）。
// **ConfirmSimOrder 是本包唯一写路径**（无自动确认定时器、无其他改单状态入口，§10.6 C5）；
// 确认后仍是模拟（SIMULATED），无任何通往真实资金的按钮/路径（§6/§8，不赌原则 D-019）。
// 本包零网络零密钥（D-010）：只读 store + 纯函数门禁，无任何真实账户/下单端点。
package simapi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"

	"arbcn/internal/fact"
	"arbcn/internal/sim"
	simv1 "arbcn/internal/simapi/gen/arbcn/sim/v1"
	"arbcn/internal/simapi/gen/arbcn/sim/v1/simv1connect"
	"arbcn/internal/store"
)

// simOrderListLimit 列表拉取上限（M3-c 个人模拟盘：单日订单个位数，1000 远覆盖；
// status 过滤在服务层做——store.ListSimOrders 无 status 维度，量小不值得扩接口）。
const simOrderListLimit = 1000

// fxVenue / fxSymbol 即期汇率事实键（internal/collect/fx：新浪公开行情 USDCNH）。
const (
	fxVenue  = "sina"
	fxSymbol = "USDCNH"
)

// Service 实现 simv1connect.SimServiceHandler；直读 Store，无网络/key。
type Service struct {
	st  store.Store
	cfg sim.Config
	// Now 为测试注入时钟（确认成交腿时间戳等）；0 = time.Now。
	// practices #10：时钟注入覆盖全 RPC 路径（入口层最容易漏 time.Now()）。
	Now func() time.Time
}

// NewService 构造模拟执行服务。cfg 为 sim 配置（main 降级时传零值：GetSimReport 返回
// 未启用说明，其余 RPC 照常直读 store——sim 表已由迁移 0005 建好，不依赖 sim 驱动）。
func NewService(st store.Store, cfg sim.Config) *Service {
	return &Service{st: st, cfg: cfg}
}

// Handler 返回 ConnectRPC 挂载路径与处理器（main.go mux.Handle(path, h)）。
func (s *Service) Handler() (string, http.Handler) {
	return simv1connect.NewSimServiceHandler(s)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// ListSimOrders 建议订单列表（ts DESC 稳定序；status 空 = 全部）。空库返回 [] 不报错。
func (s *Service) ListSimOrders(ctx context.Context, req *connect.Request[simv1.ListSimOrdersRequest]) (*connect.Response[simv1.ListSimOrdersResponse], error) {
	orders, err := s.st.ListSimOrders(ctx, simOrderListLimit, 0)
	if err != nil {
		return nil, storeErr(err)
	}
	out := make([]*simv1.SimOrder, 0, len(orders))
	for _, o := range orders {
		if req.Msg.Status != "" && o.Status != req.Msg.Status {
			continue
		}
		out = append(out, toSimOrder(o))
	}
	return connect.NewResponse(&simv1.ListSimOrdersResponse{Orders: out}), nil
}

// ConfirmSimOrder 人工确认 → 本地模拟成交（**唯一写路径**，04-m3-spec §10.4 C3）：
//  1. GetSimOrder(id) → 未知 id 报错；status != suggested 报错（防重复确认）。
//  2. 二次门禁（§10.3 SPREAD_DRIFT + D-039 kind 分派数据面）：确认时刻重查该 kind
//     权威数据源 → ConfirmDriftCheck；权威源查不到 → fail-closed 拒（无数据不确认，从严）。
//  3. 拒 → RejectSimOrder（原子 rejected + risk_flags 追加 SPREAD_DRIFT + note）→
//     {accepted:false}（拒单 = 负样本保留，§4）。
//  4. 过 → 组 legs → store 原子 AcceptSimOrder（suggested→confirmed→filled + INSERT
//     全腿，WHERE status='suggested' 守卫拦并发双确认）→ {accepted:true}。
func (s *Service) ConfirmSimOrder(ctx context.Context, req *connect.Request[simv1.ConfirmSimOrderRequest]) (*connect.Response[simv1.ConfirmSimOrderResponse], error) {
	if req.Msg.Id <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("simapi: invalid order id %d", req.Msg.Id))
	}
	o, err := s.st.GetSimOrder(ctx, req.Msg.Id)
	if err != nil {
		return nil, storeErr(err)
	}
	if o.Status != store.SimStatusSuggested {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("simapi: order %d status = %q, want suggested（防重复确认）", o.ID, o.Status))
	}

	// 二次门禁（§10.3 SPREAD_DRIFT + D-039 kind 分派数据面）。
	reject, reason, err := s.confirmDrift(ctx, o)
	if err != nil {
		return nil, storeErr(err)
	}
	if reject {
		if err := s.st.RejectSimOrder(ctx, o.ID, reason, sim.RiskSpreadDrift); err != nil {
			return nil, storeErr(err)
		}
		updated, err := s.st.GetSimOrder(ctx, o.ID)
		if err != nil {
			return nil, storeErr(err)
		}
		return connect.NewResponse(&simv1.ConfirmSimOrderResponse{Order: toSimOrder(updated), Accepted: false}), nil
	}

	// 通过 → 组 legs（共享 sim.BuildLegs）→ 原子成交。note 用订单生成价
	// （ConfirmDriftCheck 已保证确认时刻漂移 <2%，成交腿 ref_price 口径一致）。
	legs := sim.BuildLegs(o, s.now())
	note := fmt.Sprintf("人工确认成交 @ ref_price %.2f（二次门禁通过）", o.RefPrice)
	if err := s.st.AcceptSimOrder(ctx, o.ID, note, legs); err != nil {
		return nil, storeErr(err)
	}
	updated, err := s.st.GetSimOrder(ctx, o.ID)
	if err != nil {
		return nil, storeErr(err)
	}
	return connect.NewResponse(&simv1.ConfirmSimOrderResponse{Order: toSimOrder(updated), Accepted: true}), nil
}

// ListSimPositions 持仓腿列表 + 即期 RMB 折算（§10.5 C4 关键口径）。
// pnl_rmb = pnl × 即期 USDCNH（绝对金额用即期，**非** RMBDayEnd 年化口径——H1/R6#1
// 刻度线：那是费率折算，绝对金额不适用）；汇率缺失 → pnl_rmb=0 + 前端标注「USD 原值」。
func (s *Service) ListSimPositions(ctx context.Context, _ *connect.Request[simv1.ListSimPositionsRequest]) (*connect.Response[simv1.ListSimPositionsResponse], error) {
	positions, err := s.st.ListSimPositions(ctx, simOrderListLimit, 0)
	if err != nil {
		return nil, storeErr(err)
	}
	fxRate, fxOK, err := s.latestValue(ctx, fact.KindFX, fxVenue, fxSymbol)
	if err != nil {
		return nil, storeErr(err)
	}
	out := make([]*simv1.SimPosition, 0, len(positions))
	for _, p := range positions {
		rmb := 0.0
		if fxOK {
			rmb = p.PnL * fxRate
		}
		out = append(out, toSimPosition(p, rmb))
	}
	return connect.NewResponse(&simv1.ListSimPositionsResponse{Positions: out}), nil
}

// GetTestnetAccounts 测试网账户快照列表（D-040 SimExec 测试网账户区数据面）。
// 数据 = 探针余额查询结果持久化（main.go 启动一次 + 8h tick 刷新）；空库 = 空列表
// （探针未启用或尚未完成首次查询）。零网络零密钥：只读 store。
func (s *Service) GetTestnetAccounts(ctx context.Context, _ *connect.Request[simv1.GetTestnetAccountsRequest]) (*connect.Response[simv1.GetTestnetAccountsResponse], error) {
	accts, err := s.st.ListTestnetAccounts(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	out := make([]*simv1.TestnetAccount, 0, len(accts))
	for _, a := range accts {
		out = append(out, toTestnetAccount(a))
	}
	return connect.NewResponse(&simv1.GetTestnetAccountsResponse{Accounts: out}), nil
}

// GetSimReport 周频对账报告（04-m3-spec §5.3/§9.5）。文件由 sim.settleLoop 每 7×8h tick
// 渲染到 cfg.ReportPath。文件不存在 → exists=false + note（说明渲染周期）；报告禁用
// （HistoryDays≤0 / 路径未配置）→ 同样 exists=false + 说明。
func (s *Service) GetSimReport(ctx context.Context, _ *connect.Request[simv1.GetSimReportRequest]) (*connect.Response[simv1.GetSimReportResponse], error) {
	resp := &simv1.GetSimReportResponse{}
	if s.cfg.ReportPath == "" || s.cfg.HistoryDays <= 0 {
		resp.Note = "周频报告未启用（ARBCN_SIM_HISTORY_DAYS=0 或报告路径未配置）"
		return connect.NewResponse(resp), nil
	}
	data, err := os.ReadFile(s.cfg.ReportPath)
	if err != nil {
		if os.IsNotExist(err) {
			resp.Note = "周频报告每 7×8h tick 渲染（尚未生成）"
			return connect.NewResponse(resp), nil
		}
		resp.Note = "读取报告失败: " + err.Error()
		return connect.NewResponse(resp), nil
	}
	resp.Markdown, resp.Exists = string(data), true
	return connect.NewResponse(resp), nil
}

// latestValue 返回每 (kind, venue, symbol) 最新事实值；无事实 → ok=false。
func (s *Service) latestValue(ctx context.Context, kind, venue, symbol string) (float64, bool, error) {
	fs, err := s.st.LatestFacts(ctx, kind, venue, symbol)
	if err != nil {
		return 0, false, err
	}
	if len(fs) == 0 {
		return 0, false, nil
	}
	return fs[0].Value, true, nil
}

// confirmDrift 确认时刻二次门禁数据面（§10.3 SPREAD_DRIFT，D-039 kind 分派）。
// spec §10.3「查不到 ticker/funding → fail-closed 拒」是 funding_hedge 语义；repo/carry
// 的数据源不同，硬编码双查会让它们**恒拒**（M3-c 验收发现）：repo 无价格行情（面值锚），
// carry 无 funding（稳定币生息无资金费率）。故按 kind 选**权威数据源**，fail-closed 语义
// 保持——每类订单的权威源查不到 → 拒（宁缺毋滥，与生成侧同口径）。ConfirmDriftCheck
// 纯函数签名不变（spec §10.3 锚点稳定）。
//
//   - funding_hedge：ticker → curRef，funding → curSpread（双查，缺一 fail-closed）。
//   - repo：ref = 面值锚（GC001 面值 100，curRef=genRef 漂移恒 0，无价格行情可查）；
//     spread = 当日逆回购利率（KindReverseRepo，与 repoSignal 同权威源）；查不到 → 拒。
//   - carry_asset：spread = 生息年化（KindDefiRate，权威源）；查不到 → 拒。ref = ticker
//     有则查（稳定币现价），无 → curRef=genRef（面值锚 1.0 漂移恒 0，跳过 ref 检查——
//     稳定币无方向风险，核心漂移是生息年化，D-019 白名单已对冲）。
//   - 未知 kind → fail-closed 拒（与 SignalToOrder L1 同口径）。
//
// [对抗测试锚点]（D-039）：repo/carry 恒拒回归——删 kind 分派 → 新测试
// TestConfirmSimOrderRepoAccept/TestConfirmSimOrderCarryAccept 必红。
func (s *Service) confirmDrift(ctx context.Context, o store.SimOrder) (bool, string, error) {
	var curRef, curSpread float64
	var refOK, spreadOK bool
	var err error

	switch o.Kind {
	case store.SimKindFundingHedge:
		curRef, refOK, err = s.latestValue(ctx, fact.KindTicker, o.Venue, o.Symbol)
		if err != nil {
			return false, "", err
		}
		curSpread, spreadOK, err = s.latestValue(ctx, fact.KindFunding, o.Venue, o.Symbol)
		if err != nil {
			return false, "", err
		}
		if !refOK {
			return true, "SPREAD_DRIFT: 确认时刻查不到 ticker 行情（fail-closed 拒单）", nil
		}
		if !spreadOK {
			return true, "SPREAD_DRIFT: 确认时刻查不到 funding 费率（fail-closed 拒单）", nil
		}
	case store.SimKindRepo:
		// 面值锚：GC001 面值恒定（RefPrice=100），无价格行情；curRef=genRef 漂移恒 0。
		curRef = o.RefPrice
		curSpread, spreadOK, err = s.latestValue(ctx, fact.KindReverseRepo, "", "")
		if err != nil {
			return false, "", err
		}
		if !spreadOK {
			return true, "SPREAD_DRIFT: 确认时刻查不到 reverse_repo 当日利率（fail-closed 拒单）", nil
		}
		refOK = true
	case store.SimKindCarryAsset:
		curSpread, spreadOK, err = s.latestValue(ctx, fact.KindDefiRate, o.Venue, o.Symbol)
		if err != nil {
			return false, "", err
		}
		if !spreadOK {
			return true, "SPREAD_DRIFT: 确认时刻查不到 defi_rate 生息年化（fail-closed 拒单）", nil
		}
		curRef, refOK, err = s.latestValue(ctx, fact.KindTicker, o.Venue, o.Symbol)
		if err != nil {
			return false, "", err
		}
		if !refOK {
			curRef = o.RefPrice // 稳定币面值锚 1.0：无 ticker → 漂移恒 0，只查年化变化（D-039）
			refOK = true
		}
	default:
		return true, "SPREAD_DRIFT: 未知套利 kind " + o.Kind + "（fail-closed 拒单）", nil
	}

	reject, reason := sim.ConfirmDriftCheck(o.RefPrice, o.ExpectedSpread, curRef, curSpread)
	return reject, reason, nil
}

// toSimOrder 映射 store.SimOrder → proto。ts 用毫秒时间戳（前端 bigint 承载）。
func toSimOrder(o store.SimOrder) *simv1.SimOrder {
	return &simv1.SimOrder{
		Id: o.ID, TsMs: o.Ts.UnixMilli(), SrcRule: o.SrcRule, Kind: o.Kind,
		Venue: o.Venue, Symbol: o.Symbol, Side: o.Side, Qty: o.Qty,
		RefPrice: o.RefPrice, ExpectedSpread: o.ExpectedSpread,
		RiskFlags: o.RiskFlags, Status: o.Status, Note: o.Note,
	}
}

// toTestnetAccount 映射 store.TestnetAccount → proto（D-040）。updated_at 毫秒时间戳。
func toTestnetAccount(a store.TestnetAccount) *simv1.TestnetAccount {
	out := &simv1.TestnetAccount{
		Source: a.Source, AccountAlias: a.AccountAlias, EquityUsd: a.EquityUSD,
		UpdatedAtMs: a.UpdatedAt.UnixMilli(),
	}
	out.Details = make([]*simv1.TestnetAccountDetail, 0, len(a.Details))
	for _, d := range a.Details {
		out.Details = append(out.Details, &simv1.TestnetAccountDetail{
			Asset: d.Asset, Balance: d.Balance, EquityUsd: d.EquityUSD,
		})
	}
	return out
}

// toSimPosition 映射 store.SimPosition → proto（pnlRmb 由调用方按即期折算）。
func toSimPosition(p store.SimPosition, pnlRmb float64) *simv1.SimPosition {
	return &simv1.SimPosition{
		Id: p.ID, OrderId: p.OrderID, TsMs: p.Ts.UnixMilli(), Kind: p.Kind,
		Venue: p.Venue, Symbol: p.Symbol, Side: p.Side, Qty: p.Qty,
		RefPrice: p.RefPrice, Funding: p.Funding, Pnl: p.PnL, PnlRmb: pnlRmb, Status: p.Status,
	}
}

// storeErr 统一存储层错误映射：依赖 DB 不可用 = Unavailable（与 dashboard 同口径）。
func storeErr(err error) error {
	return connect.NewError(connect.CodeUnavailable, err)
}
