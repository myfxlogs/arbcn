package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/httpapi"
	"arbcn/internal/store"
)

func TestListLatestFacts(t *testing.T) {
	st := &fakeStore{facts: []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 1, Ts: t0},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 2, Ts: t0.Add(time.Hour)}, // 最新
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 3, Ts: t0},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "ETH", Value: 4, Ts: t0},
		{Kind: fact.KindIV, Venue: "deribit", Symbol: "BTC", Value: 45, Ts: t0},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "TRX", Value: -5, Ts: t0},
	}}
	client := newTestServer(t, New(st, nil, nil))
	ctx := context.Background()

	t.Run("全量每键最新一条", func(t *testing.T) {
		resp, err := client.ListLatestFacts(ctx, connect.NewRequest(&dashboardv1.ListLatestFactsRequest{}))
		if err != nil {
			t.Fatalf("ListLatestFacts: %v", err)
		}
		got := resp.Msg.Facts
		if len(got) != 5 {
			t.Fatalf("facts = %d, want 5（binance/BTC 两行只留最新）", len(got))
		}
		for _, f := range got {
			if f.Venue == "binance" && f.Symbol == "BTC" {
				if f.Value != 2 {
					t.Errorf("binance/BTC value = %v, want 2（最新）", f.Value)
				}
				if f.Kind != fact.KindFunding {
					t.Errorf("kind = %q", f.Kind)
				}
			}
		}
		// 排序稳定：按 kind 升序（funding < iv）
		if got[0].Kind != fact.KindFunding || got[4].Kind != fact.KindIV {
			t.Errorf("结果未按 kind, venue, symbol 排序: %v", got)
		}
	})

	t.Run("kind 过滤", func(t *testing.T) {
		resp, err := client.ListLatestFacts(ctx, connect.NewRequest(&dashboardv1.ListLatestFactsRequest{Kind: fact.KindIV}))
		if err != nil {
			t.Fatalf("ListLatestFacts: %v", err)
		}
		if len(resp.Msg.Facts) != 1 || resp.Msg.Facts[0].Symbol != "BTC" {
			t.Fatalf("iv 过滤结果 = %v", resp.Msg.Facts)
		}
	})

	t.Run("venue+symbol 过滤", func(t *testing.T) {
		req := connect.NewRequest(&dashboardv1.ListLatestFactsRequest{Kind: fact.KindFunding, Venue: "okx", Symbol: "ETH"})
		resp, err := client.ListLatestFacts(ctx, req)
		if err != nil {
			t.Fatalf("ListLatestFacts: %v", err)
		}
		if len(resp.Msg.Facts) != 1 || resp.Msg.Facts[0].Value != 4 {
			t.Fatalf("okx/ETH 过滤结果 = %v", resp.Msg.Facts)
		}
	})
}

func TestListAlertsPaginationAndAck(t *testing.T) {
	st := &fakeStore{alerts: []store.Alert{
		{ID: 1, RuleID: 1, RuleName: "funding 预警", Ts: t0.Add(time.Minute), Level: store.LevelWarn, Message: "active"},
		{ID: 2, RuleID: 2, RuleName: "IV 机会", Ts: t0.Add(2 * time.Minute), Level: store.LevelInfo, Message: "active"},
		{ID: 3, RuleID: 1, RuleName: "funding 预警", Ts: t0.Add(3 * time.Minute), Level: store.LevelCritical, Message: "active"},
	}}
	client := newTestServer(t, New(st, nil, nil))
	ctx := context.Background()

	t.Run("时间降序 + 分页", func(t *testing.T) {
		resp, err := client.ListAlerts(ctx, connect.NewRequest(&dashboardv1.ListAlertsRequest{Limit: 2}))
		if err != nil {
			t.Fatalf("ListAlerts: %v", err)
		}
		got := resp.Msg.Alerts
		if len(got) != 2 || got[0].Id != 3 || got[1].Id != 2 {
			t.Fatalf("降序分页结果 = %v", got)
		}
		if got[0].RuleName != "funding 预警" || got[0].Level != store.LevelCritical {
			t.Errorf("JOIN 规则名字段缺失: %+v", got[0])
		}

		resp2, err := client.ListAlerts(ctx, connect.NewRequest(&dashboardv1.ListAlertsRequest{Limit: 10, Offset: 2}))
		if err != nil {
			t.Fatalf("ListAlerts offset: %v", err)
		}
		if len(resp2.Msg.Alerts) != 1 || resp2.Msg.Alerts[0].Id != 1 {
			t.Fatalf("offset 分页结果 = %v", resp2.Msg.Alerts)
		}
	})

	t.Run("ack 单条确认", func(t *testing.T) {
		if _, err := client.AckAlert(ctx, connect.NewRequest(&dashboardv1.AckAlertRequest{Id: 3})); err != nil {
			t.Fatalf("AckAlert: %v", err)
		}
		resp, err := client.ListAlerts(ctx, connect.NewRequest(&dashboardv1.ListAlertsRequest{Limit: 10}))
		if err != nil {
			t.Fatalf("ListAlerts: %v", err)
		}
		if !resp.Msg.Alerts[0].Acked || resp.Msg.Alerts[1].Acked {
			t.Errorf("ack 后 acked 标记错误: %v", resp.Msg.Alerts)
		}
	})

	t.Run("ack 未知 id 幂等无错", func(t *testing.T) {
		if _, err := client.AckAlert(ctx, connect.NewRequest(&dashboardv1.AckAlertRequest{Id: 999})); err != nil {
			t.Fatalf("AckAlert 未知 id: %v", err)
		}
	})

	t.Run("ack 非法 id 拒绝", func(t *testing.T) {
		_, err := client.AckAlert(ctx, connect.NewRequest(&dashboardv1.AckAlertRequest{Id: 0}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("code = %v, want InvalidArgument", connect.CodeOf(err))
		}
	})
}

func TestListTriggerStates(t *testing.T) {
	v := 15.5
	since := t0.Add(-time.Hour)
	st := &fakeStore{states: []store.RuleState{
		{RuleName: "funding 预警", State: store.StateActive, Since: since, LastValue: &v},
		{RuleName: "IV 机会", State: store.StateArmed, LastValue: nil}, // 从未评估
	}}
	client := newTestServer(t, New(st, nil, nil))

	resp, err := client.ListTriggerStates(context.Background(), connect.NewRequest(&dashboardv1.ListTriggerStatesRequest{}))
	if err != nil {
		t.Fatalf("ListTriggerStates: %v", err)
	}
	got := resp.Msg.States
	if len(got) != 2 {
		t.Fatalf("states = %d, want 2", len(got))
	}
	if got[0].RuleName != "funding 预警" || got[0].State != store.StateActive {
		t.Errorf("state[0] = %+v", got[0])
	}
	if got[0].Since.AsTime() != since {
		t.Errorf("since = %v, want %v", got[0].Since.AsTime(), since)
	}
	if got[0].LastValue == nil || *got[0].LastValue != 15.5 {
		t.Errorf("last_value = %v", got[0].LastValue)
	}
	if got[1].Since != nil || got[1].LastValue != nil {
		t.Errorf("从未评估规则应缺省 since/last_value: %+v", got[1])
	}
}

func TestHealth(t *testing.T) {
	migOK := func(context.Context) ([]string, error) { return nil, nil }
	migPending := func(context.Context) ([]string, error) { return []string{"0004.sql"}, nil }
	migErr := func(context.Context) ([]string, error) { return nil, errors.New("boom") }

	cases := []struct {
		name   string
		db     httpapi.Pinger
		mig    httpapi.PendingMigrations
		status string
		reason string
	}{
		{"全健康", fakePinger{}, migOK, "ok", ""},
		{"db 不可达", fakePinger{err: errors.New("down")}, nil, "degraded", "db_unreachable"},
		{"迁移未应用", fakePinger{}, migPending, "degraded", "pending_migrations"},
		{"迁移检查失败", fakePinger{}, migErr, "degraded", "migrations_check_failed"},
		{"无 db 无迁移（仅存活）", nil, nil, "ok", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestServer(t, New(&fakeStore{}, tc.db, tc.mig))
			resp, err := client.Health(context.Background(), connect.NewRequest(&dashboardv1.HealthRequest{}))
			if err != nil {
				t.Fatalf("Health: %v", err)
			}
			if resp.Msg.Status != tc.status || resp.Msg.Reason != tc.reason {
				t.Errorf("health = %+v, want %s/%s", resp.Msg, tc.status, tc.reason)
			}
		})
	}
}

func TestStoreErrorMapsToUnavailable(t *testing.T) {
	st := &fakeStore{err: errors.New("pg down")}
	client := newTestServer(t, New(st, nil, nil))
	ctx := context.Background()

	if _, err := client.ListLatestFacts(ctx, connect.NewRequest(&dashboardv1.ListLatestFactsRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("ListLatestFacts err = %v, want Unavailable", err)
	}
	if _, err := client.ListAlerts(ctx, connect.NewRequest(&dashboardv1.ListAlertsRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("ListAlerts err = %v, want Unavailable", err)
	}
	if _, err := client.AckAlert(ctx, connect.NewRequest(&dashboardv1.AckAlertRequest{Id: 1})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("AckAlert err = %v, want Unavailable", err)
	}
	if _, err := client.ListTriggerStates(ctx, connect.NewRequest(&dashboardv1.ListTriggerStatesRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("ListTriggerStates err = %v, want Unavailable", err)
	}
}
