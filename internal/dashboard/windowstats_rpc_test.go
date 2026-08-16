package dashboard

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
)

func TestListFundingWindowStats(t *testing.T) {
	base := t0 // now（注入时钟；facts 全部取 t0 附近真实时刻，fake 未设 To 时兜底 time.Now()）
	day := 24 * time.Hour
	// 窗口内（[now−7d, now)）：
	//   okx/BTC：5 份全正（均值 6）→ tradable
	//   binance/ETH：4 份含 1 负（均值 2.5，75% 正）→ watch
	//   okx/TRX：3 份全负（均值 −2）→ not（均值为负，D-019 不造可交易假象）
	// 窗口外（−20d）：okx/BTC −99 → 不计入（窗口边界过滤）。
	st := &fakeStore{facts: []fact.Fact{
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 5, Ts: base.Add(-day)},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 6, Ts: base.Add(-2 * day)},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 7, Ts: base.Add(-3 * day)},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 5, Ts: base.Add(-4 * day)},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 7, Ts: base.Add(-5 * day)},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "ETH", Value: 4, Ts: base.Add(-day)},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "ETH", Value: 5, Ts: base.Add(-2 * day)},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "ETH", Value: -2, Ts: base.Add(-3 * day)},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "ETH", Value: 3, Ts: base.Add(-4 * day)},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "TRX", Value: -1, Ts: base.Add(-day)},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "TRX", Value: -2, Ts: base.Add(-2 * day)},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "TRX", Value: -3, Ts: base.Add(-3 * day)},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: -99, Ts: base.Add(-20 * day)}, // 窗口外，须被过滤
	}}
	srv := New(st, nil, nil, nil)
	srv.Now = func() time.Time { return base }
	client := newTestServer(t, srv)
	ctx := context.Background()

	resp, err := client.ListFundingWindowStats(ctx, connect.NewRequest(&dashboardv1.ListFundingWindowStatsRequest{}))
	if err != nil {
		t.Fatalf("ListFundingWindowStats: %v", err)
	}
	m := resp.Msg

	if m.WindowDays != 7 {
		t.Errorf("window_days = %v, want 7", m.WindowDays)
	}

	// overall：12 份（窗口外 −99 不计入），mean = 34/12 ≈ 2.83，正费率占比 8/12 ≈ 0.667 → watch。
	o := m.Overall
	if o.Count != 12 {
		t.Errorf("overall.count = %d, want 12（窗口外 −99 被过滤）", o.Count)
	}
	if o.Class != WindowWatch {
		t.Errorf("overall.class = %q, want watch（67%% 正 / 均值正）", o.Class)
	}
	if o.PositiveShare < 0.66 || o.PositiveShare > 0.68 {
		t.Errorf("overall.positive_share = %v, want 0.667（8 正 / 12）", o.PositiveShare)
	}
	if o.Min != -3 || o.Max != 7 {
		t.Errorf("overall min/max = %v/%v, want -3/7", o.Min, o.Max)
	}

	// per_pair：3 行，均值降序 okx/BTC → binance/ETH → okx/TRX。
	if len(m.PerPair) != 3 {
		t.Fatalf("per_pair = %d, want 3", len(m.PerPair))
	}
	got := []string{m.PerPair[0].Venue + "/" + m.PerPair[0].Symbol,
		m.PerPair[1].Venue + "/" + m.PerPair[1].Symbol,
		m.PerPair[2].Venue + "/" + m.PerPair[2].Symbol}
	want := []string{"okx/BTC", "binance/ETH", "okx/TRX"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("per_pair[%d] = %s, want %s（均值降序）", i, got[i], want[i])
		}
	}
	if m.PerPair[0].Stats.Class != WindowTradable {
		t.Errorf("okx/BTC class = %q, want tradable", m.PerPair[0].Stats.Class)
	}
	if m.PerPair[1].Stats.Class != WindowWatch {
		t.Errorf("binance/ETH class = %q, want watch", m.PerPair[1].Stats.Class)
	}
	if m.PerPair[2].Stats.Class != WindowNot {
		t.Errorf("okx/TRX class = %q, want not（均值为负）", m.PerPair[2].Stats.Class)
	}
}

func TestListFundingWindowStats_Empty(t *testing.T) {
	// 无数据 → overall class=not + note 明示无数据，不 panic 不编造。
	srv := New(&fakeStore{}, nil, nil, nil)
	srv.Now = func() time.Time { return t0 }
	client := newTestServer(t, srv)
	resp, err := client.ListFundingWindowStats(context.Background(), connect.NewRequest(&dashboardv1.ListFundingWindowStatsRequest{}))
	if err != nil {
		t.Fatalf("ListFundingWindowStats: %v", err)
	}
	o := resp.Msg.Overall
	if o.Count != 0 || o.Class != WindowNot {
		t.Errorf("空窗口 overall = count %d class %q, want 0/not", o.Count, o.Class)
	}
	if o.Note == "" {
		t.Error("空窗口 note 应明示无数据")
	}
	if len(resp.Msg.PerPair) != 0 {
		t.Errorf("per_pair 应为空，got %d", len(resp.Msg.PerPair))
	}
}
