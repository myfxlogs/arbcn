package pgstore

import (
	"context"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// TestDashboardReadPath：仪表盘读取路径数据面（M1-g）——
// LatestFacts 每键最新 / ListAlerts 降序分页 + ack / 触发器视图 NULL 投影。
func TestDashboardReadPath(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "facts", "rules", "alerts", "trigger_states")

	s := New(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	facts := []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 1, Unit: fact.UnitPctAnnualized, Ts: now.Add(-2 * time.Minute), Src: "test"},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 2, Unit: fact.UnitPctAnnualized, Ts: now.Add(-time.Minute), Src: "test"},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 3, Unit: fact.UnitPctAnnualized, Ts: now.Add(-time.Minute), Src: "test"},
		{Kind: fact.KindIV, Venue: "deribit", Symbol: "BTC", Value: 45, Unit: fact.UnitPct, Ts: now, Src: "test"},
	}
	if err := s.InsertFacts(ctx, facts); err != nil {
		t.Fatalf("InsertFacts: %v", err)
	}

	// LatestFacts：每 (kind, venue, symbol) 只留 ts 最新一条。
	got, err := s.LatestFacts(ctx, "", "", "")
	if err != nil {
		t.Fatalf("LatestFacts: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("LatestFacts len = %d, want 3（binance/BTC 两行合并）", len(got))
	}
	for _, f := range got {
		if f.Venue == "binance" && f.Symbol == "BTC" && f.Value != 2 {
			t.Errorf("LatestFacts binance/BTC = %v, want 2（最新）", f.Value)
		}
	}
	got, err = s.LatestFacts(ctx, fact.KindFunding, "", "")
	if err != nil {
		t.Fatalf("LatestFacts(funding): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("LatestFacts(funding) len = %d, want 2", len(got))
	}

	// 规则 + 告警 + 触发器视图（含未评估规则的 NULL 投影）。
	id1, err := s.UpsertRule(ctx, store.Rule{Name: "r1", Kind: fact.KindFunding, Cond: "avg_30d > 15", Level: store.LevelWarn})
	if err != nil {
		t.Fatalf("UpsertRule(r1): %v", err)
	}
	if _, err := s.UpsertRule(ctx, store.Rule{Name: "r2", Kind: fact.KindIV, Cond: "avg_30d > 20", Level: store.LevelInfo}); err != nil {
		t.Fatalf("UpsertRule(r2): %v", err)
	}
	ts1 := now.Add(-time.Minute)
	ts2 := now
	if err := s.InsertAlert(ctx, store.Alert{RuleID: id1, Ts: ts1, Level: store.LevelWarn, Message: "r1 active"}); err != nil {
		t.Fatalf("InsertAlert(1): %v", err)
	}
	if err := s.InsertAlert(ctx, store.Alert{RuleID: id1, Ts: ts2, Level: store.LevelInfo, Message: "r1 resolved"}); err != nil {
		t.Fatalf("InsertAlert(2): %v", err)
	}

	alerts, err := s.ListAlerts(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListAlerts: %v", err)
	}
	if len(alerts) != 2 || !alerts[0].Ts.Equal(ts2) || !alerts[1].Ts.Equal(ts1) {
		t.Fatalf("ListAlerts 降序 = %+v", alerts)
	}
	if alerts[0].RuleName != "r1" || alerts[0].Acked {
		t.Errorf("ListAlerts join/acked = %+v", alerts[0])
	}
	page, err := s.ListAlerts(ctx, 1, 1)
	if err != nil {
		t.Fatalf("ListAlerts(page): %v", err)
	}
	if len(page) != 1 || !page[0].Ts.Equal(ts1) {
		t.Fatalf("ListAlerts page = %+v, want 第二条", page)
	}

	if err := s.AckAlert(ctx, alerts[0].ID); err != nil {
		t.Fatalf("AckAlert: %v", err)
	}
	if err := s.AckAlert(ctx, 999999); err != nil { // 未知 id 幂等无错
		t.Fatalf("AckAlert(unknown): %v", err)
	}
	alerts, err = s.ListAlerts(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListAlerts(after ack): %v", err)
	}
	if !alerts[0].Acked || alerts[1].Acked {
		t.Fatalf("acked = %v/%v, want true/false", alerts[0].Acked, alerts[1].Acked)
	}

	if err := s.PutTriggerState(ctx, store.TriggerState{RuleID: id1, State: store.StateActive, Since: ts1, LastValue: 0.25}); err != nil {
		t.Fatalf("PutTriggerState: %v", err)
	}
	states, err := s.ListTriggerStates(ctx)
	if err != nil {
		t.Fatalf("ListTriggerStates: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("ListTriggerStates len = %d, want 2", len(states))
	}
	if states[0].RuleName != "r1" || states[0].State != store.StateActive || !states[0].Since.Equal(ts1) ||
		states[0].LastValue == nil || *states[0].LastValue != 0.25 {
		t.Errorf("states[0] = %+v, want r1/active/ts1/0.25", states[0])
	}
	if states[1].RuleName != "r2" || states[1].State != store.StateArmed ||
		!states[1].Since.IsZero() || states[1].LastValue != nil {
		t.Errorf("states[1] = %+v, want r2/armed/零 since/nil last", states[1])
	}
}

// TestUnackedAndAckAll：未读告警数据面（M2-a §1.1/§1.2）——
// ListUnacked 只回未读 + 降序；AckAll 单事务全清并返回确认数；重复调用幂等归零。
func TestUnackedAndAckAll(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "rules", "alerts")

	s := New(pool)
	id1, err := s.UpsertRule(ctx, store.Rule{Name: "r1", Kind: fact.KindFunding, Cond: "x > 1", Level: store.LevelWarn})
	if err != nil {
		t.Fatalf("UpsertRule: %v", err)
	}
	id2, err := s.UpsertRule(ctx, store.Rule{Name: "r2", Kind: fact.KindIV, Cond: "x > 1", Level: store.LevelInfo})
	if err != nil {
		t.Fatalf("UpsertRule(r2): %v", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	for i, a := range []store.Alert{
		{RuleID: id1, Ts: now.Add(time.Minute), Level: store.LevelWarn, Message: "a"},
		{RuleID: id2, Ts: now.Add(2 * time.Minute), Level: store.LevelInfo, Message: "b"},
		{RuleID: id1, Ts: now.Add(3 * time.Minute), Level: store.LevelCritical, Message: "c"},
	} {
		if err := s.InsertAlert(ctx, a); err != nil {
			t.Fatalf("InsertAlert(%d): %v", i, err)
		}
	}
	// 主键是串行序列（TRUNCATE 不重置），按消息定位 "a" 告警的真实 id 再 ack。
	all, err := s.ListAlerts(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListAlerts: %v", err)
	}
	for _, a := range all {
		if a.Message == "a" {
			if err := s.AckAlert(ctx, a.ID); err != nil {
				t.Fatalf("AckAlert(%d): %v", a.ID, err)
			}
			break
		}
	}

	unacked, err := s.ListUnacked(ctx)
	if err != nil {
		t.Fatalf("ListUnacked: %v", err)
	}
	if len(unacked) != 2 {
		t.Fatalf("ListUnacked len = %d, want 2（id=1 已 acked 排除）", len(unacked))
	}
	if !unacked[0].Ts.Equal(now.Add(3*time.Minute)) || !unacked[1].Ts.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("ListUnacked 降序 = %+v, want c,b", unacked)
	}
	if unacked[0].RuleName != "r1" || unacked[0].Acked || unacked[1].RuleName != "r2" {
		t.Errorf("ListUnacked JOIN/acked = %+v", unacked)
	}

	n, err := s.AckAll(ctx)
	if err != nil {
		t.Fatalf("AckAll: %v", err)
	}
	if n != 2 {
		t.Fatalf("AckAll = %d, want 2", n)
	}
	unacked, err = s.ListUnacked(ctx)
	if err != nil {
		t.Fatalf("ListUnacked(after): %v", err)
	}
	if len(unacked) != 0 {
		t.Fatalf("ListUnacked after AckAll = %+v, want 空", unacked)
	}
	if n2, err := s.AckAll(ctx); err != nil || n2 != 0 { // 幂等归零
		t.Fatalf("AckAll(again) = %d, %v, want 0/nil", n2, err)
	}
}
