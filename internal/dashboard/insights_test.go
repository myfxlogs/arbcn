package dashboard

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// TestRejectDistribution：拒单 risk_flags 逐条展开计数、降序取前 maxN。
// 对抗：删展开计数（只数订单不数 flag）必红；maxN 截断生效；非 rejected 不计数。
func TestRejectDistribution(t *testing.T) {
	orders := []store.SimOrder{
		{ID: 1, Status: store.SimStatusRejected, RiskFlags: []string{"SPREAD_DRIFT"}},
		{ID: 2, Status: store.SimStatusRejected, RiskFlags: []string{"SPREAD_LOW"}},
		{ID: 3, Status: store.SimStatusRejected, RiskFlags: []string{"SPREAD_DRIFT", "SPREAD_LOW"}},
		{ID: 4, Status: store.SimStatusSuggested, RiskFlags: []string{"SPREAD_DRIFT"}}, // 非拒单不计
		{ID: 5, Status: store.SimStatusConfirmed, RiskFlags: []string{"SPREAD_LOW"}},   // 非拒单不计
	}
	got := rejectDistribution(orders, 3)
	if len(got) != 1 {
		t.Fatalf("insights = %d, want 1", len(got))
	}
	it := got[0]
	if it.id != "reject_dist" || it.severity != "info" {
		t.Errorf("id/severity = %s/%s, want reject_dist/info", it.id, it.severity)
	}
	// 计数：SPREAD_DRIFT=2、SPREAD_LOW=2（平局按 flag 字典序：SPREAD_DRIFT 在前）。
	if !strings.Contains(it.detail, "SPREAD_DRIFT ×2") || !strings.Contains(it.detail, "SPREAD_LOW ×2") {
		t.Errorf("detail = %q, want 两 flag 均 ×2", it.detail)
	}
	if strings.Contains(it.detail, "SPREAD_DRIFT ×3") || strings.Contains(it.detail, "SPREAD_LOW ×3") {
		t.Errorf("detail = %q, 非拒单被计入", it.detail)
	}
	// maxN 截断：只留计数最高的 1 个 flag。
	trunc := rejectDistribution(orders, 1)
	if !strings.Contains(trunc[0].detail, "SPREAD_DRIFT ×2") || strings.Contains(trunc[0].detail, "SPREAD_LOW") {
		t.Errorf("maxN=1 detail = %q, want 仅 SPREAD_DRIFT", trunc[0].detail)
	}
	// 无拒单 → nil（拒单但无 flags → counts 空 → nil）。
	if got := rejectDistribution([]store.SimOrder{{Status: store.SimStatusSuggested}}, 3); got != nil {
		t.Errorf("无拒单返回 %+v, want nil", got)
	}
	if got := rejectDistribution([]store.SimOrder{{Status: store.SimStatusRejected, RiskFlags: nil}}, 3); got != nil {
		t.Errorf("拒单无 flag 返回 %+v, want nil", got)
	}
}

// TestDefiAnomalies：截面中位数×因子标尖峰。
// 对抗：删 `>factor×median` 判断（全标/全不标）必红；样本 <3 不判定；NaN 跳过（practices #7）。
func TestDefiAnomalies(t *testing.T) {
	f := func(v float64, venue, sym string) fact.Fact {
		return fact.Fact{Kind: fact.KindDefiRate, Venue: venue, Symbol: sym, Value: v, Ts: t0}
	}
	spike := []fact.Fact{
		f(3.5, "aave-v3", "USDC"), f(3.9, "morpho-blue", "USDC"), f(4.0, "ethena-usde", "USDC"), f(12.57, "aave-v3", "USDT"),
	}
	// 中位 3.95，×2.0 → 阈值 7.9：只标 12.57。
	out := defiAnomalies(spike, anomalyFactor)
	if len(out) != 1 {
		t.Fatalf("anomalies = %d, want 1", len(out))
	}
	if out[0].id != "defi_anomaly:aave-v3:USDT" || out[0].severity != "warn" {
		t.Errorf("id/severity = %s/%s, want defi_anomaly:aave-v3:USDT/warn", out[0].id, out[0].severity)
	}
	if !strings.Contains(out[0].detail, "12.57") {
		t.Errorf("detail = %q, want 含尖峰值 12.57", out[0].detail)
	}
	// factor 放大到 4.0 → 阈值 15.8 → 无尖峰。
	if out := defiAnomalies(spike, 4.0); len(out) != 0 {
		t.Errorf("factor=4.0 anomalies = %d, want 0（阈值随因子抬高）", len(out))
	}
	// factor 缩小到 1.01 → 阈值 3.99 → 4.0 与 12.57 均标。
	if out := defiAnomalies(spike, 1.01); len(out) != 2 {
		t.Errorf("factor=1.01 anomalies = %d, want 2", len(out))
	}
	// 样本 <3 → 不判定。
	if out := defiAnomalies([]fact.Fact{f(3.0, "a", "USDC"), f(4.0, "b", "USDC")}, anomalyFactor); len(out) != 0 {
		t.Errorf("2 样本 anomalies = %d, want 0", len(out))
	}
	// NaN 跳过（practices #7）：3 条含 1 NaN → 有效仅 2 → 不判定。
	if out := defiAnomalies([]fact.Fact{f(3.0, "a", "USDC"), f(math.NaN(), "b", "USDC"), f(4.0, "c", "USDC")}, anomalyFactor); len(out) != 0 {
		t.Errorf("含 NaN anomalies = %d, want 0（NaN 跳过后样本不足）", len(out))
	}
	// 同实体最新值胜出：aave-v3/USDC 旧 4.0 新 3.5 → 截面用 3.5。
	stale := []fact.Fact{
		f(3.5, "aave-v3", "USDC"), f(3.9, "morpho-blue", "USDC"), f(4.0, "ethena-usde", "USDC"),
		{Kind: fact.KindDefiRate, Venue: "aave-v3", Symbol: "USDT", Value: 30.0, Ts: t0.Add(-24 * time.Hour)}, // 旧尖峰
		{Kind: fact.KindDefiRate, Venue: "aave-v3", Symbol: "USDT", Value: 12.57, Ts: t0},                     // 新值覆盖旧
	}
	out = defiAnomalies(stale, anomalyFactor)
	if len(out) != 1 || out[0].id != "defi_anomaly:aave-v3:USDT" {
		t.Fatalf("同实体最新值胜出：out = %+v, want 仅 aave-v3:USDT", out)
	}
}

// TestNoOrderHint：无 active 单 + funding 低于窗口门槛才提示。
func TestNoOrderHint(t *testing.T) {
	funding := func(v float64) []fact.Fact {
		return []fact.Fact{{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTCUSDT", Value: v, Ts: t0}}
	}
	// count=0 + funding 6% < 15 → 提示。
	if it, ok := noOrderHint(funding(6.0), 0); !ok || it.id != "no_order" {
		t.Errorf("低 funding 无单应提示, got %+v ok=%v", it, ok)
	}
	// 有 active 单 → 不提示（即使低 funding）。
	if _, ok := noOrderHint(funding(6.0), 1); ok {
		t.Errorf("有 active 单仍提示, want 无")
	}
	// funding ≥ 门槛 → 不提示。
	if _, ok := noOrderHint(funding(16.0), 0); ok {
		t.Errorf("funding 16 ≥ 15 仍提示, want 无")
	}
	// 无 funding 数据 → 不提示。
	if _, ok := noOrderHint(nil, 0); ok {
		t.Errorf("无 funding 数据仍提示, want 无")
	}
	// 多源取最高值：6% + 14% → top 14 < 15 → 仍提示。
	if _, ok := noOrderHint([]fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTCUSDT", Value: 6.0, Ts: t0},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "ETHUSDT", Value: 14.0, Ts: t0},
	}, 0); !ok {
		t.Errorf("多源 top 14 < 15 应提示, want 有")
	}
}

// TestMedian：奇偶长度 + 空序列 NaN + 不改输入。
func TestMedian(t *testing.T) {
	if m := median([]float64{3, 1, 2}); m != 2 {
		t.Errorf("奇数 median = %v, want 2", m)
	}
	if m := median([]float64{4, 1, 2, 3}); m != 2.5 {
		t.Errorf("偶数 median = %v, want 2.5", m)
	}
	if m := median([]float64{}); !math.IsNaN(m) {
		t.Errorf("空 median = %v, want NaN", m)
	}
	in := []float64{3, 1, 2}
	median(in)
	if in[0] != 3 || in[2] != 2 {
		t.Errorf("median 修改输入: %v", in)
	}
}

// TestListInsights：端到端——注入拒单 + DeFi 尖峰 + 停更源 + active 单，
// RPC 返回四信号、severity 排序正确（critical>warn>info）。
func TestListInsights(t *testing.T) {
	ctx := context.Background()
	now := t0
	st := &fakeStore{
		orders: []store.SimOrder{
			{ID: 4, Ts: now.Add(-24 * time.Hour), Status: store.SimStatusRejected, RiskFlags: []string{"SPREAD_DRIFT"}},
			{ID: 3, Ts: now.Add(-48 * time.Hour), Status: store.SimStatusRejected, RiskFlags: []string{"SPREAD_LOW"}},
			{ID: 2, Ts: now.Add(-72 * time.Hour), Status: store.SimStatusRejected, RiskFlags: []string{"SPREAD_DRIFT", "SPREAD_LOW"}},
			{ID: 1, Ts: now.Add(-time.Hour), Status: store.SimStatusSuggested}, // active 单 → no_order 抑制
		},
		facts: []fact.Fact{
			// funding（binance 源 live 用；近 24h 内且 ≤2×interval）
			{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTCUSDT", Value: 6.0, Ts: now.Add(-2 * time.Second)},
			{Kind: fact.KindHeartbeat, Venue: "collector", Symbol: "binance", Value: 0.1, Ts: now.Add(-time.Second), Unit: fact.UnitRatio},
			// defi 截面（30d 内）：aave-v3 USDT 尖峰
			{Kind: fact.KindDefiRate, Venue: "aave-v3", Symbol: "USDC", Value: 3.5, Ts: now.Add(-time.Hour)},
			{Kind: fact.KindDefiRate, Venue: "morpho-blue", Symbol: "USDC", Value: 3.9, Ts: now.Add(-time.Hour)},
			{Kind: fact.KindDefiRate, Venue: "ethena-usde", Symbol: "USDC", Value: 4.0, Ts: now.Add(-time.Hour)},
			{Kind: fact.KindDefiRate, Venue: "aave-v3", Symbol: "USDT", Value: 12.57, Ts: now.Add(-time.Hour)},
		},
	}
	svc := New(st, nil, nil, []SourceInfo{
		{Name: "binance", IntervalSec: 10, Kind: fact.KindFunding}, // live（hb + funding 新）
		{Name: "fx_src", IntervalSec: 10, Kind: fact.KindFX},       // 无 heartbeat → down
	})
	svc.Now = func() time.Time { return now }
	client := newTestServer(t, svc)

	resp, err := client.ListInsights(ctx, connect.NewRequest(&dashboardv1.ListInsightsRequest{}))
	if err != nil {
		t.Fatalf("ListInsights: %v", err)
	}
	items := resp.Msg.Insights
	byID := map[string]*dashboardv1.Insight{}
	for _, it := range items {
		byID[it.Id] = it
	}
	// 四信号覆盖：reject_dist + defi 尖峰 + 源 down 均在；no_order 被 active 单抑制。
	for _, want := range []string{"reject_dist", "defi_anomaly:aave-v3:USDT", "source_down:fx_src"} {
		if _, ok := byID[want]; !ok {
			t.Errorf("缺信号 %q; 全部: %v", want, ids(items))
		}
	}
	if _, ok := byID["no_order"]; ok {
		t.Errorf("active 单存在仍出 no_order: %v", ids(items))
	}
	// severity 排序：critical → warn → info（severityRank 权重序）。
	for i := 1; i < len(items); i++ {
		if severityRank(items[i-1].Severity) > severityRank(items[i].Severity) {
			t.Errorf("severity 排序乱: %v", ids(items))
		}
	}
	// 类别映射与动作非空（只读证据表面：actions 指向人工决策）。
	if byID["reject_dist"].Category != "risk" || byID["defi_anomaly:aave-v3:USDT"].Category != "anomaly" || byID["source_down:fx_src"].Category != "data" {
		t.Errorf("category 映射错: %+v", items)
	}
	for _, it := range items {
		if len(it.Actions) == 0 {
			t.Errorf("信号 %q 无候选动作", it.Id)
		}
	}
	// at 注入测试时钟（svc.Now）。
	if !items[0].At.AsTime().Equal(now) {
		t.Errorf("at = %v, want %v", items[0].At.AsTime(), now)
	}
}

// TestListInsightsStoreErr：存储层故障 → Unavailable，不 panic。
func TestListInsightsStoreErr(t *testing.T) {
	ctx := context.Background()
	svc := New(&fakeStore{err: errors.New("pg down")}, nil, nil, nil)
	svc.Now = func() time.Time { return t0 }
	client := newTestServer(t, svc)
	_, err := client.ListInsights(ctx, connect.NewRequest(&dashboardv1.ListInsightsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("store 故障 code = %v, want Unavailable", connect.CodeOf(err))
	}
}

// ids 提取 id 列表（断言辅助）。
func ids(items []*dashboardv1.Insight) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Id
	}
	return out
}
