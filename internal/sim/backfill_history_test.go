package sim

import (
	"context"
	"errors"
	"testing"
	"time"

	"arbcn/internal/fact"
)

var errFake = errors.New("fake error")

// fakeHistoryCollector 测试用历史数据源（venue + 固定批次）。
type fakeHistoryCollector struct {
	venue string
	batch []fact.Fact
}

func (f *fakeHistoryCollector) Venue() string { return f.venue }
func (f *fakeHistoryCollector) Poll(context.Context) ([]fact.Fact, error) {
	return f.batch, nil
}

// funcHistoryCollector 在 fakeHistoryCollector 外包一层 Poll 钩子（统计调用）。
type funcHistoryCollector struct {
	fake    *fakeHistoryCollector
	onPoll  func()
}

func (f *funcHistoryCollector) Venue() string { return f.fake.Venue() }
func (f *funcHistoryCollector) Poll(ctx context.Context) ([]fact.Fact, error) {
	f.onPoll()
	return f.fake.Poll(ctx)
}

// factsByTs 取 stub 中某 venue 全部 funding 事实（测试断言辅助，stub 数据面）。
func factsByTs(st *storeStub, venue string) []fact.Fact {
	var out []fact.Fact
	for _, f := range st.facts {
		if f.Kind == fact.KindFunding && f.Venue == venue {
			out = append(out, f)
		}
	}
	return out
}

// TestBackfillHistoryIdempotent：[对抗测试锚点 §9.5 S4] 回填编排幂等——跑两遍不重复。
// 删 UncoveredFacts 覆盖跳过 → 第二遍重复落库断言必红。
func TestBackfillHistoryIdempotent(t *testing.T) {
	st := &storeStub{}
	batch := fundingFacts(t0.Add(-30*24*time.Hour), 3, 10.95) // 30 天前 3 期
	// 注入跨 symbol：同结算时刻两条（校验键含 symbol）。
	batch = append(batch, fact.Fact{Kind: fact.KindFunding, Venue: "binance", Symbol: "ETH",
		Value: 11.5, Ts: batch[2].Ts, Src: "data-api"})
	collector := &fakeHistoryCollector{venue: "binance", batch: batch}

	// 第一遍：全量回填（stub.InsertFacts 追加到 st.facts）。
	if err := BackfillHistory(context.Background(), st, []HistoryCollector{collector}, 365); err != nil {
		t.Fatalf("第一遍: %v", err)
	}
	if got := len(factsByTs(st, "binance")); got != 4 {
		t.Fatalf("第一遍落库 = %d, want 4（3 期 BTC + 1 ETH）", got)
	}

	// 第二遍：同样批次 → 0 条新增（全部已覆盖）。
	if err := BackfillHistory(context.Background(), st, []HistoryCollector{collector}, 365); err != nil {
		t.Fatalf("第二遍: %v", err)
	}
	if got := len(factsByTs(st, "binance")); got != 4 {
		t.Fatalf("第二遍落库 = %d, want 4（跑两遍不重复，§9.5 幂等）", got)
	}
}

// TestBackfillHistoryDaysZero：days<=0 → 禁用，不调用 collector（返回 nil）。
func TestBackfillHistoryDaysZero(t *testing.T) {
	st := &storeStub{}
	called := false
	collector := &fakeHistoryCollector{venue: "binance", batch: fundingFacts(t0, 1, 10.95)}
	calledCollector := &funcHistoryCollector{fake: collector, onPoll: func() { called = true }}
	if err := BackfillHistory(context.Background(), st, []HistoryCollector{calledCollector}, 0); err != nil {
		t.Fatalf("BackfillHistory(0): %v", err)
	}
	if called {
		t.Fatal("days=0 不应调用 collector（禁用）")
	}
}

// TestBackfillHistoryInsertError：InsertFacts 失败 → 整体返回错误（boot 一次性任务显式报错）。
func TestBackfillHistoryInsertError(t *testing.T) {
	st := &storeStub{insertFactsErr: errFake}
	collector := &fakeHistoryCollector{venue: "binance", batch: fundingFacts(t0, 1, 10.95)}
	if err := BackfillHistory(context.Background(), st, []HistoryCollector{collector}, 365); err == nil {
		t.Fatal("BackfillHistory(insert error) = nil, want error")
	}
}
