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
	client := newTestServer(t, New(st, nil, nil, nil))
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
	client := newTestServer(t, New(st, nil, nil, nil))
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
	client := newTestServer(t, New(st, nil, nil, nil))

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
			client := newTestServer(t, New(&fakeStore{}, tc.db, tc.mig, nil))
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
	client := newTestServer(t, New(st, nil, nil, nil))
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
	if _, err := client.ListUnacked(ctx, connect.NewRequest(&dashboardv1.ListUnackedRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("ListUnacked err = %v, want Unavailable", err)
	}
	if _, err := client.AckAll(ctx, connect.NewRequest(&dashboardv1.AckAllRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("AckAll err = %v, want Unavailable", err)
	}
	// ListSourceHealth 无源时不查询存储（空列表）；须带源才走存储层错误路径。
	shClient := newTestServer(t, New(st, nil, nil, []SourceInfo{{Name: "src", IntervalSec: 10, Kind: fact.KindFunding}}))
	if _, err := shClient.ListSourceHealth(ctx, connect.NewRequest(&dashboardv1.ListSourceHealthRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("ListSourceHealth err = %v, want Unavailable", err)
	}
}

// TestListUnacked：未读告警列表 + 计数（排除 acked=true）；逐条 ack 后未读递减。
func TestListUnacked(t *testing.T) {
	st := &fakeStore{alerts: []store.Alert{
		{ID: 1, RuleID: 1, RuleName: "funding 预警", Ts: t0.Add(time.Minute), Level: store.LevelWarn, Message: "a", Acked: true},
		{ID: 2, RuleID: 2, RuleName: "IV 机会", Ts: t0.Add(2 * time.Minute), Level: store.LevelInfo, Message: "b"},
		{ID: 3, RuleID: 1, RuleName: "funding 预警", Ts: t0.Add(3 * time.Minute), Level: store.LevelCritical, Message: "c"},
	}}
	client := newTestServer(t, New(st, nil, nil, nil))
	ctx := context.Background()

	resp, err := client.ListUnacked(ctx, connect.NewRequest(&dashboardv1.ListUnackedRequest{}))
	if err != nil {
		t.Fatalf("ListUnacked: %v", err)
	}
	if resp.Msg.Total != 2 || len(resp.Msg.Items) != 2 {
		t.Fatalf("total/items = %d/%d, want 2/2（id=1 已 acked 排除）", resp.Msg.Total, len(resp.Msg.Items))
	}
	if resp.Msg.Items[0].Id != 3 || resp.Msg.Items[1].Id != 2 {
		t.Errorf("未读顺序 = %v, want 3,2（ts DESC）", resp.Msg.Items)
	}
	if resp.Msg.Items[0].Rule != "funding 预警" || resp.Msg.Items[0].Level != store.LevelCritical {
		t.Errorf("items[0] = %+v（JOIN 规则名/级别字段）", resp.Msg.Items[0])
	}
	if resp.Msg.Items[0].Ts.AsTime() != t0.Add(3*time.Minute) {
		t.Errorf("items[0].ts = %v", resp.Msg.Items[0].Ts.AsTime())
	}

	// 逐条 ack → 未读递减。
	if _, err := client.AckAlert(ctx, connect.NewRequest(&dashboardv1.AckAlertRequest{Id: 3})); err != nil {
		t.Fatalf("AckAlert: %v", err)
	}
	resp2, err := client.ListUnacked(ctx, connect.NewRequest(&dashboardv1.ListUnackedRequest{}))
	if err != nil {
		t.Fatalf("ListUnacked(after ack): %v", err)
	}
	if resp2.Msg.Total != 1 || len(resp2.Msg.Items) != 1 || resp2.Msg.Items[0].Id != 2 {
		t.Fatalf("ack 后未读 = %+v, want 只剩 id=2", resp2.Msg)
	}
}

// TestAckAll：全部已读（单事务），返回确认数；重复调用幂等归零。
func TestAckAll(t *testing.T) {
	st := &fakeStore{alerts: []store.Alert{
		{ID: 1, RuleID: 1, RuleName: "r1", Ts: t0, Level: store.LevelWarn, Message: "a"},
		{ID: 2, RuleID: 2, RuleName: "r2", Ts: t0, Level: store.LevelInfo, Message: "b", Acked: true},
		{ID: 3, RuleID: 1, RuleName: "r1", Ts: t0, Level: store.LevelCritical, Message: "c"},
	}}
	client := newTestServer(t, New(st, nil, nil, nil))
	ctx := context.Background()

	resp, err := client.AckAll(ctx, connect.NewRequest(&dashboardv1.AckAllRequest{}))
	if err != nil {
		t.Fatalf("AckAll: %v", err)
	}
	if resp.Msg.AckedCount != 2 {
		t.Fatalf("acked_count = %d, want 2（id=2 已 acked 不计）", resp.Msg.AckedCount)
	}
	// 再 AckAll → 0（幂等）。
	resp2, err := client.AckAll(ctx, connect.NewRequest(&dashboardv1.AckAllRequest{}))
	if err != nil {
		t.Fatalf("AckAll(again): %v", err)
	}
	if resp2.Msg.AckedCount != 0 {
		t.Fatalf("第二次 acked_count = %d, want 0", resp2.Msg.AckedCount)
	}
}

// TestListSourceHealth：RPC 数据面——live/stale/down + 无心跳 → down；
// interval_sec/last_poll_at/last_fact_at 字段回填。
func TestListSourceHealth(t *testing.T) {
	now := time.Now()
	const iv = int64(10)
	// 心跳 fact：value = 错过的窗口数 → lastOK ≈ Ts − value×interval。
	hb := func(src string, missed float64) fact.Fact {
		return fact.Fact{Kind: fact.KindHeartbeat, Venue: "collector", Symbol: src,
			Value: missed, Unit: fact.UnitRatio, Ts: now.Add(-2 * time.Second), Src: "heartbeat"}
	}
	sources := []SourceInfo{
		{Name: "live_source", IntervalSec: iv, Kind: fact.KindFunding},
		{Name: "stale_source", IntervalSec: iv, Kind: fact.KindTicker},
		{Name: "down_source", IntervalSec: iv, Kind: fact.KindFX},
		{Name: "never_seen", IntervalSec: iv, Kind: fact.KindIV},
	}
	st := &fakeStore{facts: []fact.Fact{
		hb("live_source", 0.2),  // lastOK = now-4s ≤ 20s
		hb("stale_source", 0.1), // lastOK = now-3s ≤ 20s
		hb("down_source", 2.3),  // lastOK = now-25s > 20s
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Ts: now.Add(-2 * time.Second)}, // funding 最新 → live
		{Kind: fact.KindTicker, Venue: "okx", Symbol: "BTC", Ts: now.Add(-30 * time.Second)},     // ticker 最新旧 → stale
	}}
	// R4#7：注入固定时钟（svc.Now），消除测试 now 与服务内 time.Now 的漂移（墙钟 flake）。
	svc := New(st, nil, nil, sources)
	svc.Now = func() time.Time { return now }
	client := newTestServer(t, svc)
	ctx := context.Background()

	resp, err := client.ListSourceHealth(ctx, connect.NewRequest(&dashboardv1.ListSourceHealthRequest{}))
	if err != nil {
		t.Fatalf("ListSourceHealth: %v", err)
	}
	got := resp.Msg.Items
	if len(got) != 4 {
		t.Fatalf("items = %d, want 4", len(got))
	}
	byName := map[string]*dashboardv1.SourceHealth{}
	for _, it := range got {
		byName[it.Name] = it
	}
	if s := byName["live_source"]; s.Status != StatusLive || s.IntervalSec != iv {
		t.Errorf("live_source = %+v, want live/10s", s)
	}
	if s := byName["stale_source"]; s.Status != StatusStale {
		t.Errorf("stale_source = %+v, want stale", s)
	}
	if s := byName["down_source"]; s.Status != StatusDown {
		t.Errorf("down_source = %+v, want down", s)
	}
	if s := byName["never_seen"]; s.Status != StatusDown || s.LastPollAt != nil || s.LastFactAt != nil {
		t.Errorf("never_seen = %+v, want down/无时间戳（无 heartbeat 视为 down）", s)
	}
	// live_source 时间戳回填：last_poll_at ≈ now-4s；last_fact_at ≈ now-2s。
	ls := byName["live_source"]
	if ls.LastPollAt == nil || ls.LastFactAt == nil {
		t.Fatalf("live_source 缺时间戳: %+v", ls)
	}
	if d := now.Sub(ls.LastPollAt.AsTime()); d < 3*time.Second || d > 6*time.Second {
		t.Errorf("live_source last_poll_at 距 now %v, want ~4s", d)
	}
	if d := now.Sub(ls.LastFactAt.AsTime()); d < time.Second || d > 3*time.Second {
		t.Errorf("live_source last_fact_at 距 now %v, want ~2s", d)
	}
}

// TestSourceHealthBoundaries：sourceHealth 纯判定（注入固定 now，避免时钟漂移）——
// 2×interval 的切换点（03-m2-spec §2.1；R4#2 裁定：stale 阈值从 1×iv 改为 2×iv，
// 给 Scheduler ±10% 抖动留余量）。
func TestSourceHealthBoundaries(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	src := SourceInfo{Name: "src", IntervalSec: 10, Kind: fact.KindFunding}
	hb := func(ts time.Time, missed float64) fact.Fact {
		return fact.Fact{Kind: fact.KindHeartbeat, Venue: "collector", Symbol: "src", Value: missed, Ts: ts}
	}
	latest := func(ts time.Time) fact.Fact {
		return fact.Fact{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Ts: ts}
	}
	s := &Service{}

	cases := []struct {
		name    string
		hbTS    time.Time
		hbVal   float64
		factTS  time.Time
		hasFact bool
		want    string
	}{
		// live：lastOK 新（≤2×interval），lastFact 新（≤2×interval）
		{"live", now.Add(-time.Second), 0.1, now.Add(-time.Second), true, StatusLive},
		// [回归锚点] 健康源抖动：lastFact 1.1×iv（11s，Scheduler nextWait 上限）→ 必须 live，
		// 不得周期性误报 stale（R4#2：原 1×iv 阈值在 0.9–1.1iv 区间反复闪烁）
		{"抖动 1.1×interval → live", now, 0.1, now.Add(-11 * time.Second), true, StatusLive},
		// 边界：lastOK 恰 2×interval（20s）→ 非 > → 非 down
		{"lastPoll 恰 2×interval → live", now, 2.0, now.Add(-time.Second), true, StatusLive},
		// 边界：lastFact 恰 2×interval（20s）→ 非 > → 非 stale
		{"lastFact 恰 2×interval → live", now, 0.1, now.Add(-20 * time.Second), true, StatusLive},
		// stale：lastOK 新，lastFact 老（> 2×interval）
		{"stale", now, 0.1, now.Add(-21 * time.Second), true, StatusStale},
		// stale：无该 kind fact（从未产出）→ lastFact 零 → stale
		{"no facts yet → stale", now, 0.1, time.Time{}, false, StatusStale},
		// down：lastOK 老（> 2×interval）
		{"down", now, 3.0, now.Add(-time.Second), true, StatusDown},
		// 边界：lastOK 略过 2×interval（21s）→ down
		{"lastPoll 略过 2×interval → down", now, 2.1, now.Add(-time.Second), true, StatusDown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := []fact.Fact{hb(tc.hbTS, tc.hbVal)}
			if tc.hasFact {
				facts = append(facts, latest(tc.factTS))
			}
			s.st = &fakeStore{facts: facts}
			status, _, _, err := s.sourceHealth(ctx, src, now)
			if err != nil {
				t.Fatalf("sourceHealth: %v", err)
			}
			if status != tc.want {
				t.Errorf("status = %q, want %q", status, tc.want)
			}
		})
	}

	// 无 heartbeat 记录 → down（含 last_poll 零值）。
	s.st = &fakeStore{facts: []fact.Fact{latest(now.Add(-time.Second))}}
	status, lastPoll, _, err := s.sourceHealth(ctx, src, now)
	if err != nil {
		t.Fatalf("sourceHealth(no hb): %v", err)
	}
	if status != StatusDown || !lastPoll.IsZero() {
		t.Errorf("no heartbeat: status=%q lastPoll=%v, want down/零值", status, lastPoll)
	}
}
