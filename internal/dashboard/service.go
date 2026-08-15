// Package dashboard：Web 仪表盘 ConnectRPC 服务（docs/design/02-monitor-architecture.md §9）。
// 只读查询 + 告警确认（ack），无交易能力（§1 铁律）。main.go 接线属 M1-h。
package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/dashboard/gen/arbcn/dashboard/v1/dashboardv1connect"
	"arbcn/internal/fact"
	"arbcn/internal/httpapi"
	"arbcn/internal/store"
)

// 分页约束（与 proto 注释一致）。
const (
	defaultAlertLimit = 100
	maxAlertLimit     = 500
)

// 源 freshness 状态值域（与 proto SourceHealth.status、03-m2-spec §2.1 一致）。
const (
	StatusLive  = "live"  // now - last_poll ≤ 2×interval（采集器正常）
	StatusStale = "stale" // last_poll 新但 now - last_fact > interval（闭市/报价冻结）
	StatusDown  = "down"  // now - last_poll > 2×interval（采集器失联）
)

// SourceInfo 是源健康视图需要的启用源元数据（main.go 从 collect.Named 提取：
// Name / Interval / Collector.Kind()）。
type SourceInfo struct {
	Name        string
	IntervalSec int64
	Kind        string
}

// Service 实现 dashboardv1connect.DashboardServiceHandler；RPC 直读 Store。
type Service struct {
	st         store.Store
	db         httpapi.Pinger            // nil = 只报进程存活
	migrations httpapi.PendingMigrations // nil = 不检查迁移
	sources    []SourceInfo              // 启用源清单（ListSourceHealth 数据面）
	// Now 为测试注入时钟（ListFacts 的 30d 汇率窗口）；0 = time.Now。
	Now func() time.Time
}

// New 构造服务；db/migrations 与 /healthz 同源（复用 httpapi.Healthz 的依赖类型）；
// sources 是启用源健康信息（M2-a §2.2：每源 name/interval_sec/kind），可为空。
func New(st store.Store, db httpapi.Pinger, migrations httpapi.PendingMigrations, sources []SourceInfo) *Service {
	return &Service{st: st, db: db, migrations: migrations, sources: sources}
}

// Handler 返回 ConnectRPC 挂载路径与处理器（M1-h：mux.Handle(path, h)）。
func (s *Service) Handler() (string, http.Handler) {
	return dashboardv1connect.NewDashboardServiceHandler(s)
}

// ListLatestFacts 返回每 (kind, venue, symbol) 的最新事实（机会面板快照）。
func (s *Service) ListLatestFacts(ctx context.Context, req *connect.Request[dashboardv1.ListLatestFactsRequest]) (*connect.Response[dashboardv1.ListLatestFactsResponse], error) {
	facts, err := s.st.LatestFacts(ctx, req.Msg.Kind, req.Msg.Venue, req.Msg.Symbol)
	if err != nil {
		return nil, storeErr(err)
	}
	out := make([]*dashboardv1.Fact, 0, len(facts))
	for _, f := range facts {
		out = append(out, toFact(f))
	}
	return connect.NewResponse(&dashboardv1.ListLatestFactsResponse{Facts: out}), nil
}

// ListAlerts 时间降序分页返回告警流（含 acked 标记）。
func (s *Service) ListAlerts(ctx context.Context, req *connect.Request[dashboardv1.ListAlertsRequest]) (*connect.Response[dashboardv1.ListAlertsResponse], error) {
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = defaultAlertLimit
	}
	if limit > maxAlertLimit {
		limit = maxAlertLimit
	}
	offset := int(req.Msg.Offset)
	alerts, err := s.st.ListAlerts(ctx, limit, offset)
	if err != nil {
		return nil, storeErr(err)
	}
	out := make([]*dashboardv1.Alert, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, toAlert(a))
	}
	return connect.NewResponse(&dashboardv1.ListAlertsResponse{Alerts: out}), nil
}

// AckAlert 单条确认（幂等；未知 id 由存储层无操作）。
func (s *Service) AckAlert(ctx context.Context, req *connect.Request[dashboardv1.AckAlertRequest]) (*connect.Response[dashboardv1.AckAlertResponse], error) {
	if req.Msg.Id <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("dashboard: invalid alert id %d", req.Msg.Id))
	}
	if err := s.st.AckAlert(ctx, req.Msg.Id); err != nil {
		return nil, storeErr(err)
	}
	return connect.NewResponse(&dashboardv1.AckAlertResponse{}), nil
}

// ListTriggerStates 返回各规则状态机视图（未评估规则投影为 armed）。
func (s *Service) ListTriggerStates(ctx context.Context, _ *connect.Request[dashboardv1.ListTriggerStatesRequest]) (*connect.Response[dashboardv1.ListTriggerStatesResponse], error) {
	states, err := s.st.ListTriggerStates(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	out := make([]*dashboardv1.TriggerState, 0, len(states))
	for _, rs := range states {
		ts := &dashboardv1.TriggerState{
			RuleName:  rs.RuleName,
			State:     rs.State,
			LastValue: rs.LastValue,
		}
		if !rs.Since.IsZero() {
			ts.Since = timestamppb.New(rs.Since)
		}
		out = append(out, ts)
	}
	return connect.NewResponse(&dashboardv1.ListTriggerStatesResponse{States: out}), nil
}

// Health 与 /healthz 同源信息：进程存活 + DB 可达 + 迁移应用状态。
func (s *Service) Health(ctx context.Context, _ *connect.Request[dashboardv1.HealthRequest]) (*connect.Response[dashboardv1.HealthResponse], error) {
	resp := &dashboardv1.HealthResponse{Status: "ok"}
	if s.db != nil {
		if err := s.db.Ping(ctx); err != nil {
			resp.Status, resp.Reason = "degraded", "db_unreachable"
		}
	}
	if s.migrations != nil && resp.Status == "ok" {
		pending, err := s.migrations(ctx)
		switch {
		case err != nil:
			resp.Status, resp.Reason = "degraded", "migrations_check_failed"
		case len(pending) > 0:
			resp.Status, resp.Reason = "degraded", "pending_migrations"
		}
	}
	return connect.NewResponse(resp), nil
}

// ListUnacked 返回未读告警列表 + 计数（未读 = acked=false；M2-a §1.2 铃铛）。
// 未读数小一次拉全；total = len(items)（存储层 ListUnacked 返回全量未读）。
func (s *Service) ListUnacked(ctx context.Context, _ *connect.Request[dashboardv1.ListUnackedRequest]) (*connect.Response[dashboardv1.ListUnackedResponse], error) {
	alerts, err := s.st.ListUnacked(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	items := make([]*dashboardv1.UnackedAlert, 0, len(alerts))
	for _, a := range alerts {
		items = append(items, &dashboardv1.UnackedAlert{
			Id:      a.ID,
			Rule:    a.RuleName,
			Level:   a.Level,
			Message: a.Message,
			Ts:      timestamppb.New(a.Ts),
		})
	}
	return connect.NewResponse(&dashboardv1.ListUnackedResponse{Items: items, Total: int32(len(items))}), nil
}

// AckAll 全部已读（单事务 UPDATE，M2-a §1.2）；返回本次确认的告警数。
func (s *Service) AckAll(ctx context.Context, _ *connect.Request[dashboardv1.AckAllRequest]) (*connect.Response[dashboardv1.AckAllResponse], error) {
	n, err := s.st.AckAll(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	return connect.NewResponse(&dashboardv1.AckAllResponse{AckedCount: int32(n)}), nil
}

// ListSourceHealth 返回各启用源 freshness 状态（M2-a §2.1/§2.2）。
// 数据面：heartbeat 最新 fact 反推 last_poll_at；该源 kind 最新 fact ts 作 last_fact_at。
func (s *Service) ListSourceHealth(ctx context.Context, _ *connect.Request[dashboardv1.ListSourceHealthRequest]) (*connect.Response[dashboardv1.ListSourceHealthResponse], error) {
	now := time.Now()
	items := make([]*dashboardv1.SourceHealth, 0, len(s.sources))
	for _, src := range s.sources {
		status, lastPoll, lastFact, err := s.sourceHealth(ctx, src, now)
		if err != nil {
			return nil, storeErr(err)
		}
		item := &dashboardv1.SourceHealth{
			Name:        src.Name,
			IntervalSec: src.IntervalSec,
			Status:      status,
		}
		if !lastPoll.IsZero() {
			item.LastPollAt = timestamppb.New(lastPoll)
		}
		if !lastFact.IsZero() {
			item.LastFactAt = timestamppb.New(lastFact)
		}
		items = append(items, item)
	}
	return connect.NewResponse(&dashboardv1.ListSourceHealthResponse{Items: items}), nil
}

// sourceHealth 按 03-m2-spec §2.1 判定单源状态：
//   - last_poll_at：heartbeat fact（kind=heartbeat, symbol=源名）的 value = 错过的窗口数，
//     反推 lastOK ≈ emit_ts − value×interval_sec；
//   - last_fact_at：该源 kind 最新 fact ts（LatestFacts 每键最新，取最大 ts）；
//   - 无 heartbeat 记录 → 无法确认存活，视为 down（含注释说明）。
func (s *Service) sourceHealth(ctx context.Context, src SourceInfo, now time.Time) (status string, lastPoll, lastFact time.Time, err error) {
	hb, err := s.st.LatestFacts(ctx, fact.KindHeartbeat, "", src.Name)
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	if len(hb) == 0 {
		// 无 heartbeat 记录：源从未成功轮询或心跳未落库 → 语义等同失联（down）。
		return StatusDown, time.Time{}, time.Time{}, nil
	}
	iv := time.Duration(src.IntervalSec) * time.Second
	lastPoll = hb[0].Ts.Add(-time.Duration(hb[0].Value * float64(iv.Seconds()) * float64(time.Second)))

	latest, err := s.st.LatestFacts(ctx, src.Kind, "", "")
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	for _, f := range latest {
		if f.Ts.After(lastFact) {
			lastFact = f.Ts
		}
	}
	switch {
	case now.Sub(lastPoll) > 2*iv:
		return StatusDown, lastPoll, lastFact, nil
	case lastFact.IsZero() || now.Sub(lastFact) > iv:
		return StatusStale, lastPoll, lastFact, nil
	default:
		return StatusLive, lastPoll, lastFact, nil
	}
}

// now 返回注入时钟或 time.Now。
func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// toFact 映射 internal/fact.Fact → proto。
func toFact(f fact.Fact) *dashboardv1.Fact {
	return &dashboardv1.Fact{
		Kind:   f.Kind,
		Venue:  f.Venue,
		Symbol: f.Symbol,
		Value:  f.Value,
		Unit:   f.Unit,
		Ts:     timestamppb.New(f.Ts),
		Src:    f.Src,
	}
}

// toAlert 映射 store.Alert → proto。
func toAlert(a store.Alert) *dashboardv1.Alert {
	return &dashboardv1.Alert{
		Id:       a.ID,
		RuleId:   a.RuleID,
		RuleName: a.RuleName,
		Ts:       timestamppb.New(a.Ts),
		Level:    a.Level,
		Message:  a.Message,
		Acked:    a.Acked,
	}
}

// storeErr 统一存储层错误映射：依赖 DB 不可用 = Unavailable。
func storeErr(err error) error {
	return connect.NewError(connect.CodeUnavailable, err)
}
