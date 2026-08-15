// Package dashboard：Web 仪表盘 ConnectRPC 服务（docs/design/02-monitor-architecture.md §9）。
// 只读查询 + 告警确认（ack），无交易能力（§1 铁律）。main.go 接线属 M1-h。
package dashboard

import (
	"context"
	"fmt"
	"net/http"

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

// Service 实现 dashboardv1connect.DashboardServiceHandler；四 RPC 直读 Store。
type Service struct {
	st         store.Store
	db         httpapi.Pinger            // nil = 只报进程存活
	migrations httpapi.PendingMigrations // nil = 不检查迁移
}

// New 构造服务；db/migrations 与 /healthz 同源（复用 httpapi.Healthz 的依赖类型）。
func New(st store.Store, db httpapi.Pinger, migrations httpapi.PendingMigrations) *Service {
	return &Service{st: st, db: db, migrations: migrations}
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
