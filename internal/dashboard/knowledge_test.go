package dashboard

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/knowledge"
	"arbcn/internal/store"
)

// TestKnowledgeMatchesFundingSpike：瞬时费率尖峰命中「funding:spike_trap」条目 → 只读
// 呈现「当前情况 + 上回判定」，actions 指向 D#。经验库空 → 无信号。
// [对抗测试锚点] 删 FundingSpikeTrap 命中判断 → 必红。
func TestKnowledgeMatchesFundingSpike(t *testing.T) {
	ctx := context.Background()
	now := t0
	st := &fakeStore{
		facts: fundingSpikeFacts(now),
		knowledge: []store.KnowledgeEntry{{
			Signature: knowledge.SignatureFundingSpikeTrap,
			Verdict:   "坑", Rationale: "尖峰陷阱：别按瞬时值开仓", Source: "对话 #64", Status: "active",
		}},
	}
	svc := New(st, nil, nil, nil)
	svc.Now = func() time.Time { return now }

	items, err := svc.knowledgeMatches(ctx, now, nil)
	if err != nil {
		t.Fatalf("knowledgeMatches: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("matches = %d, want 1; %+v", len(items), items)
	}
	it := items[0]
	if it.id != "knowledge_match:"+knowledge.SignatureFundingSpikeTrap+":okx:ETHUSDT" {
		t.Errorf("id = %q, want funding spike trap 命中", it.id)
	}
	if it.category != "knowledge" || it.severity != "info" {
		t.Errorf("category/severity = %s/%s, want knowledge/info", it.category, it.severity)
	}
	if !strings.Contains(it.detail, "坑") || !strings.Contains(it.detail, "尖峰陷阱") {
		t.Errorf("detail = %q, want 含上回判定与依据", it.detail)
	}
	if len(it.actions) == 0 || !strings.Contains(it.actions[0], "D#") {
		t.Errorf("actions = %v, want 指向 D#（只读证据表面）", it.actions)
	}

	// 经验库空 → 无信号（宁缺毋滥，不查 funding）。
	st2 := &fakeStore{facts: fundingSpikeFacts(now)}
	if items, err := New(st2, nil, nil, nil).knowledgeMatches(ctx, now, nil); err != nil || len(items) != 0 {
		t.Errorf("空经验库 matches = %v, err=%v, want 无信号", items, err)
	}
}

// TestKnowledgeMatchesDefiSpike：单池利率尖峰命中「defi:single_pool_spike」。
func TestKnowledgeMatchesDefiSpike(t *testing.T) {
	ctx := context.Background()
	now := t0
	st := &fakeStore{
		knowledge: []store.KnowledgeEntry{{
			Signature: knowledge.SignatureDefiSinglePoolSpike,
			Verdict:   "坑·核实", Rationale: "单池尖峰多不可持续", Source: "对话 #63", Status: "active",
		}},
	}
	svc := New(st, nil, nil, nil)
	svc.Now = func() time.Time { return now }
	defi := []fact.Fact{
		{Kind: fact.KindDefiRate, Venue: "aave-v3", Symbol: "USDC", Value: 3.5, Ts: now.Add(-time.Hour)},
		{Kind: fact.KindDefiRate, Venue: "morpho-blue", Symbol: "USDC", Value: 3.9, Ts: now.Add(-time.Hour)},
		{Kind: fact.KindDefiRate, Venue: "ethena-usde", Symbol: "USDC", Value: 4.0, Ts: now.Add(-time.Hour)},
		{Kind: fact.KindDefiRate, Venue: "aave-v3", Symbol: "USDT", Value: 12.57, Ts: now.Add(-time.Hour)},
	}
	items, err := svc.knowledgeMatches(ctx, now, defi)
	if err != nil {
		t.Fatalf("knowledgeMatches: %v", err)
	}
	if len(items) != 1 || items[0].id != "knowledge_match:"+knowledge.SignatureDefiSinglePoolSpike+":aave-v3:USDT" {
		t.Errorf("matches = %+v, want defi single pool spike 命中 aave-v3:USDT", items)
	}
}

// TestKnowledgeMatchesCrossVenue：同 symbol 跨所费率分歧命中「funding:cross_venue_divergence」
// （TRX 案例口径：一正一负）。多 venue 合一 symbol。
func TestKnowledgeMatchesCrossVenue(t *testing.T) {
	ctx := context.Background()
	now := t0
	st := &fakeStore{
		facts: []fact.Fact{
			{Kind: fact.KindFunding, Venue: "binance", Symbol: "TRXUSDT", Value: 2.3, Ts: now.Add(-time.Hour)},
			{Kind: fact.KindFunding, Venue: "okx", Symbol: "TRXUSDT", Value: -3.5, Ts: now.Add(-time.Hour)},
			{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTCUSDT", Value: 5.0, Ts: now.Add(-time.Hour)},
			{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTCUSDT", Value: 6.0, Ts: now.Add(-time.Hour)},
		},
		knowledge: []store.KnowledgeEntry{{
			Signature: knowledge.SignatureFundingCrossVenueDiverg,
			Verdict:   "已核实·真实分歧", Rationale: "TRX 跨所分歧为真实市场结构", Source: "对话 #50", Status: "active",
		}},
	}
	svc := New(st, nil, nil, nil)
	svc.Now = func() time.Time { return now }

	items, err := svc.knowledgeMatches(ctx, now, nil)
	if err != nil {
		t.Fatalf("knowledgeMatches: %v", err)
	}
	// TRX（差 5.8 ≥ 4）命中；BTC（差 1.0 < 4）不命中。
	if len(items) != 1 || items[0].id != "knowledge_match:"+knowledge.SignatureFundingCrossVenueDiverg+":TRXUSDT" {
		t.Errorf("matches = %+v, want 仅 TRXUSDT 命中", items)
	}
	if !strings.Contains(items[0].detail, "5.8") {
		t.Errorf("detail = %q, want 含差距 5.8", items[0].detail)
	}
}

// TestListKnowledgeEntries：RPC 浏览经验库（signature ASC；只读，无自动写入）。
func TestListKnowledgeEntries(t *testing.T) {
	ctx := context.Background()
	st := &fakeStore{knowledge: []store.KnowledgeEntry{
		{Signature: "defi:single_pool_spike", Verdict: "坑·核实"},
		{Signature: "funding:spike_trap", Verdict: "坑", Rationale: "尖峰陷阱", Status: "active"},
	}}
	svc := New(st, nil, nil, nil)
	client := newTestServer(t, svc)

	resp, err := client.ListKnowledgeEntries(ctx, connect.NewRequest(&dashboardv1.ListKnowledgeEntriesRequest{}))
	if err != nil {
		t.Fatalf("ListKnowledgeEntries: %v", err)
	}
	if len(resp.Msg.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(resp.Msg.Entries))
	}
	if resp.Msg.Entries[0].Signature != "defi:single_pool_spike" || resp.Msg.Entries[1].Signature != "funding:spike_trap" {
		t.Errorf("排序 = [%s %s], want signature ASC", resp.Msg.Entries[0].Signature, resp.Msg.Entries[1].Signature)
	}
	if resp.Msg.Entries[1].Verdict != "坑" {
		t.Errorf("verdict 映射错: %+v", resp.Msg.Entries[1])
	}
}

// TestReviewKnowledgeEntry：人工复核成功写 validated_at/verdict/note（回读核对）；
// 空 signature / 空 note / 非法 verdict → InvalidArgument；未知 signature → Unavailable。
// [对抗测试锚点] 删服务端 verdict 白名单校验 → 非法值不红。
func TestReviewKnowledgeEntry(t *testing.T) {
	ctx := context.Background()
	sig := knowledge.SignatureFundingSpikeTrap
	st := &fakeStore{knowledge: []store.KnowledgeEntry{
		{Signature: sig, Verdict: "坑", Rationale: "尖峰陷阱", Status: "active"},
	}}
	svc := New(st, nil, nil, nil)
	client := newTestServer(t, svc)

	// 校验失败路径。
	for _, tc := range []struct {
		name, signature, status, note string
	}{
		{"empty signature", "", "active", "x"},
		{"empty note", sig, "active", ""},
		{"bad status", sig, "nope", "x"},
	} {
		_, err := client.ReviewKnowledgeEntry(ctx, connect.NewRequest(&dashboardv1.ReviewKnowledgeEntryRequest{
			Signature: tc.signature, Status: tc.status, ValidationNote: tc.note,
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("%s: code = %v, want InvalidArgument", tc.name, connect.CodeOf(err))
		}
	}

	// 成功：写 validated_at + status + verdict + note。
	resp, err := client.ReviewKnowledgeEntry(ctx, connect.NewRequest(&dashboardv1.ReviewKnowledgeEntryRequest{
		Signature: sig, Status: knowledge.StatusRetracted, Verdict: "结构变化判定", ValidationNote: "结构变化，撤回",
	}))
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if resp == nil {
		t.Fatal("nil response")
	}

	// 回读核对（fake 内 validated_at 已置）。
	list, err := client.ListKnowledgeEntries(ctx, connect.NewRequest(&dashboardv1.ListKnowledgeEntriesRequest{}))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Msg.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(list.Msg.Entries))
	}
	e := list.Msg.Entries[0]
	if e.ValidatedAt == nil {
		t.Fatal("validated_at 未写入（复核闭环断裂）")
	}
	if e.Status != knowledge.StatusRetracted || e.Verdict != "结构变化判定" || e.ValidationNote != "结构变化，撤回" {
		t.Fatalf("复核后 = status %q verdict %q note %q", e.Status, e.Verdict, e.ValidationNote)
	}

	// 未知 signature → 存储层错误（storeErr 映射 Unavailable）。
	_, err = client.ReviewKnowledgeEntry(ctx, connect.NewRequest(&dashboardv1.ReviewKnowledgeEntryRequest{
		Signature: "unknown:sig", Status: knowledge.StatusActive, ValidationNote: "x",
	}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("unknown sig: code = %v, want Unavailable", connect.CodeOf(err))
	}
}

// TestListInsightsKnowledgeMatch：端到端——ListInsights 出现 knowledge_match 信号
// （与四信号并存的完整性）。
func TestListInsightsKnowledgeMatch(t *testing.T) {
	ctx := context.Background()
	now := t0
	st := &fakeStore{
		facts: fundingSpikeFacts(now),
		knowledge: []store.KnowledgeEntry{{
			Signature: knowledge.SignatureFundingSpikeTrap,
			Verdict:   "坑", Rationale: "尖峰陷阱", Source: "对话 #64", Status: "active",
		}},
	}
	svc := New(st, nil, nil, nil)
	svc.Now = func() time.Time { return now }
	client := newTestServer(t, svc)

	resp, err := client.ListInsights(ctx, connect.NewRequest(&dashboardv1.ListInsightsRequest{}))
	if err != nil {
		t.Fatalf("ListInsights: %v", err)
	}
	byID := map[string]*dashboardv1.Insight{}
	for _, it := range resp.Msg.Insights {
		byID[it.Id] = it
	}
	want := "knowledge_match:" + knowledge.SignatureFundingSpikeTrap + ":okx:ETHUSDT"
	if _, ok := byID[want]; !ok {
		t.Errorf("缺 knowledge_match 信号; all=%v", ids(resp.Msg.Insights))
	}
	if byID[want].Category != "knowledge" {
		t.Errorf("category = %q, want knowledge", byID[want].Category)
	}
}
