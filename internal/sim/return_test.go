// 判定门① 测量引擎测试（D-062）：TWR/MWR 纯函数数学 + 判定门① 决策矩阵 +
// settle→equity 快照对抗锚点。对抗：删 return.go 任一公式 / driver.go snapshotEquity
// 写入 → 对应测试必红。零网络零密钥（D-010）——纯内存快照。
package sim

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

const near = 1e-9

// closeEnough 相对容差断言（避免 float 精确比较）。
func closeEnough(got, want float64) bool {
	return math.Abs(got-want) <= near*math.Max(1, math.Abs(want))
}

// 快照构造辅助：ts 偏移 day 天、equity 值。
func snap(day int, equity float64) store.EquitySnapshot {
	return store.EquitySnapshot{Ts: t0.Add(time.Duration(day) * 24 * time.Hour), Equity: equity}
}

// TestTwrNoFlowsSimple：无窗口内外部流 → TWR 精确 = E_last/E_first − 1（当前 paper
// 唯一真实形态：capital_in 落在首快照前）。
func TestTwrNoFlowsSimple(t *testing.T) {
	snaps := []store.EquitySnapshot{snap(0, 100), snap(365, 110)}
	twr, days, err := TwrAnnualized(snaps, nil)
	if err != nil {
		t.Fatalf("TwrAnnualized: %v", err)
	}
	if !closeEnough(twr, 0.10) {
		t.Errorf("twr = %v, want 0.10", twr)
	}
	if !closeEnough(days, 365) {
		t.Errorf("days = %v, want 365", days)
	}
}

// TestTwrNegativeAndFlat：负收益 / 零收益序列。
func TestTwrNegativeAndFlat(t *testing.T) {
	neg := []store.EquitySnapshot{snap(0, 100), snap(182, 80)}
	if twr, _, err := TwrAnnualized(neg, nil); err != nil || !closeEnough(twr, -0.20) {
		t.Errorf("neg twr = %v err=%v, want -0.20", twr, err)
	}
	flat := []store.EquitySnapshot{snap(0, 100), snap(30, 100)}
	if twr, _, err := TwrAnnualized(flat, nil); err != nil || !closeEnough(twr, 0) {
		t.Errorf("flat twr = %v err=%v, want 0", twr, err)
	}
}

// TestTwrDietzInteriorFlow：窗口内入金 → 逐段 Dietz 调整链乘（消除资金进出对收益率
// 的污染——TWR 测策略）。
func TestTwrDietzInteriorFlow(t *testing.T) {
	snaps := []store.EquitySnapshot{snap(0, 100), snap(180, 120), snap(365, 180)}
	flows := []ExternalFlow{{Ts: t0.Add(180*24*time.Hour + time.Hour), Amount: 50}}
	// 段0（0→180d）无窗口内流：r0 = 120/100 − 1 = 0.2
	// 段1（180→365d）窗口内有 +50 入金：start = 120+50=170, end=180 → r1 = 180/170 − 1
	// prod = 1.2 × 180/170 ≈ 1.270588
	twr, days, err := TwrAnnualized(snaps, flows)
	if err != nil {
		t.Fatalf("TwrAnnualized: %v", err)
	}
	want := 1.2*180.0/170.0 - 1
	if !closeEnough(twr, want) {
		t.Errorf("twr = %v, want %v", twr, want)
	}
	if !closeEnough(days, 365) {
		t.Errorf("days = %v, want 365", days)
	}
	// 同序列不含流 → 应与手算简单 180/100−1=0.8 相同（链乘退化为简单比）。
	twr0, _, err := TwrAnnualized(snaps, nil)
	if err != nil || !closeEnough(twr0, 0.8) {
		t.Errorf("no-flow twr = %v err=%v, want 0.8", twr0, err)
	}
}

// TestTwrInteriorOutflow：窗口内出金 → 段尾减流出（start 不含流出）。
func TestTwrInteriorOutflow(t *testing.T) {
	snaps := []store.EquitySnapshot{snap(0, 100), snap(180, 120), snap(365, 60)}
	flows := []ExternalFlow{{Ts: t0.Add(181 * 24 * time.Hour), Amount: -50}} // 出金 50
	// 段1：end = 60 − (−50)? 不对——outflow Amount 负 = 出金，end = E_i − Σout，Σout = −50 → end = 60 − (−50) = 110
	// 但实现里 outF 收集 Amount<0 的流，end -= f.Amount → end = 60 − (−50) = 110。
	// start = 120（无流入）。r1 = 110/120 − 1 ≈ −0.08333。
	twr, _, err := TwrAnnualized(snaps, flows)
	if err != nil {
		t.Fatalf("TwrAnnualized: %v", err)
	}
	want := 1.2 * (110.0 / 120.0) - 1
	if !closeEnough(twr, want) {
		t.Errorf("twr = %v, want %v", twr, want)
	}
}

// TestTwrErrors：数据面错误路径——<2 快照、equity≤0、days≤0。
func TestTwrErrors(t *testing.T) {
	if _, _, err := TwrAnnualized([]store.EquitySnapshot{snap(0, 100)}, nil); !errors.Is(err, ErrInsufficientData) {
		t.Errorf("单快照 err = %v, want ErrInsufficientData", err)
	}
	if _, _, err := TwrAnnualized(nil, nil); !errors.Is(err, ErrInsufficientData) {
		t.Errorf("空快照 err = %v, want ErrInsufficientData", err)
	}
	if _, _, err := TwrAnnualized([]store.EquitySnapshot{snap(0, 0), snap(30, 100)}, nil); !errors.Is(err, ErrDataAnomaly) {
		t.Errorf("首 equity=0 err = %v, want ErrDataAnomaly", err)
	}
	if _, _, err := TwrAnnualized([]store.EquitySnapshot{snap(0, 100), snap(30, -5)}, nil); !errors.Is(err, ErrDataAnomaly) {
		t.Errorf("末 equity<0 err = %v, want ErrDataAnomaly", err)
	}
	// days ≤ 0：两快照同刻。
	same := []store.EquitySnapshot{{Ts: t0, Equity: 100}, {Ts: t0, Equity: 110}}
	if _, _, err := TwrAnnualized(same, nil); !errors.Is(err, ErrDataAnomaly) {
		t.Errorf("同刻快照 err = %v, want ErrDataAnomaly", err)
	}
}

// TestMwrSingleFlow：单一入金 → IRR 收敛到解析解 r = (E/A)^(365/T) − 1。
func TestMwrSingleFlow(t *testing.T) {
	cases := []struct {
		name string
		days int
		end  float64
	}{
		{"一年翻10%", 365, 110},
		{"两年翻21%", 730, 121}, // (1.21)^0.5 − 1 = 0.1
		{"半年", 182, 110},      // (1.1)^2 − 1 = 0.21
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			flows := []ExternalFlow{{Ts: t0, Amount: 100}}
			endTs := t0.Add(time.Duration(c.days) * 24 * time.Hour)
			r, err := MwrAnnualized(flows, c.end, endTs)
			if err != nil {
				t.Fatalf("MwrAnnualized: %v", err)
			}
			want := math.Pow(c.end/100.0, 365.0/float64(c.days)) - 1
			if !closeEnough(r, want) {
				t.Errorf("mwr = %v, want %v", r, want)
			}
		})
	}
}

// TestMwrErrors：无流入 / 期末 equity≤0 → ErrDataAnomaly（不编造）。
func TestMwrErrors(t *testing.T) {
	if _, err := MwrAnnualized(nil, 110, t0.Add(365*24*time.Hour)); !errors.Is(err, ErrDataAnomaly) {
		t.Errorf("无流入 err = %v, want ErrDataAnomaly", err)
	}
	flows := []ExternalFlow{{Ts: t0, Amount: 100}}
	if _, err := MwrAnnualized(flows, 0, t0.Add(365*24*time.Hour)); !errors.Is(err, ErrDataAnomaly) {
		t.Errorf("endEquity=0 err = %v, want ErrDataAnomaly", err)
	}
	if _, err := MwrAnnualized(flows, -50, t0.Add(365*24*time.Hour)); !errors.Is(err, ErrDataAnomaly) {
		t.Errorf("endEquity<0 err = %v, want ErrDataAnomaly", err)
	}
}

// TestAnnualize：窗口收益年化 + days≤0 拒绝。
func TestAnnualize(t *testing.T) {
	if r, err := Annualize(0.10, 365); err != nil || !closeEnough(r, 0.10) {
		t.Errorf("Annualize(0.1,365) = %v err=%v, want 0.1", r, err)
	}
	if r, err := Annualize(0.10, 182.5); err != nil || !closeEnough(r, 0.21) {
		t.Errorf("Annualize(0.1,182.5) = %v err=%v, want 0.21", r, err)
	}
	if r, err := Annualize(0, 365); err != nil || r != 0 {
		t.Errorf("Annualize(0,365) = %v err=%v, want 0", r, err)
	}
	if _, err := Annualize(0.1, 0); !errors.Is(err, ErrInsufficientData) {
		t.Errorf("Annualize(0.1,0) err = %v, want ErrInsufficientData", err)
	}
	if _, err := Annualize(0.1, -10); !errors.Is(err, ErrInsufficientData) {
		t.Errorf("Annualize(0.1,-10) err = %v, want ErrInsufficientData", err)
	}
}

// TestComputeEnvironmentStats：中位数奇偶、max、≥15% 档计数、可交易面对数；空 → 零值。
func TestComputeEnvironmentStats(t *testing.T) {
	f := func(v float64, venue, sym string) fact.Fact {
		return fact.Fact{Kind: fact.KindFunding, Venue: venue, Symbol: sym, Value: v, Ts: t0}
	}
	if got := ComputeEnvironmentStats(nil); got.FundingMedian != 0 || got.FundingMax != 0 || got.HighWindowEvents != 0 || got.TradablePairs != 0 {
		t.Errorf("空 stats = %+v, want 零值", got)
	}
	odd := []fact.Fact{f(5, "binance", "BTCUSDT"), f(10, "binance", "ETHUSDT"), f(20, "okx", "BTCUSDT")}
	got := ComputeEnvironmentStats(odd)
	if !closeEnough(got.FundingMedian, 10) || !closeEnough(got.FundingMax, 20) || got.HighWindowEvents != 1 || got.TradablePairs != 3 {
		t.Errorf("odd stats = %+v, want median=10 max=20 high=1 pairs=3", got)
	}
	even := []fact.Fact{f(5, "a", "X"), f(10, "b", "X"), f(20, "a", "Y"), f(30, "b", "Y")}
	got = ComputeEnvironmentStats(even)
	if !closeEnough(got.FundingMedian, 15) {
		t.Errorf("even median = %v, want 15", got.FundingMedian)
	}
	// 同 (venue,symbol) 多条只算一对。
	dup := []fact.Fact{f(3, "binance", "BTCUSDT"), f(4, "binance", "BTCUSDT"), f(9, "binance", "BTCUSDT")}
	if got := ComputeEnvironmentStats(dup); got.TradablePairs != 1 {
		t.Errorf("dup pairs = %d, want 1", got.TradablePairs)
	}
}

// TestSnapshotCoverage 覆盖率口径：30 天满窗 90/90=1；缺 5 → 0.944；空/负 → 0；
// 超过基线 → 截 1（覆盖率是缺口度量）。
func TestSnapshotCoverage(t *testing.T) {
	if c := SnapshotCoverage(30, 90); !closeEnough(c, 1.0) {
		t.Errorf("90/90 = %v, want 1", c)
	}
	if c := SnapshotCoverage(30, 85); !closeEnough(c, 85.0/90.0) {
		t.Errorf("85/90 = %v, want %v", c, 85.0/90.0)
	}
	if c := SnapshotCoverage(30, 0); c != 0 {
		t.Errorf("0 snaps = %v, want 0", c)
	}
	if c := SnapshotCoverage(0, 90); c != 0 {
		t.Errorf("0 days = %v, want 0", c)
	}
	if c := SnapshotCoverage(30, 200); c != 1 {
		t.Errorf("超额 = %v, want 截 1", c)
	}
}

// TestExpectedSnapshots：30 天 → 90；10 天 → 30；≤0 → 0。
func TestExpectedSnapshots(t *testing.T) {
	if n := ExpectedSnapshots(30); n != 90 {
		t.Errorf("ExpectedSnapshots(30) = %d, want 90", n)
	}
	if n := ExpectedSnapshots(10); n != 30 {
		t.Errorf("ExpectedSnapshots(10) = %d, want 30", n)
	}
	if n := ExpectedSnapshots(0); n != 0 {
		t.Errorf("ExpectedSnapshots(0) = %d, want 0", n)
	}
}

// TestValidateSnapshotIntegrity：恒等式破坏 / ts 乱序 → 暴露坏索引；全好 → -1。
func TestValidateSnapshotIntegrity(t *testing.T) {
	good := []store.EquitySnapshot{
		{Ts: t0, Equity: 100, Cash: 100},
		{Ts: t0.Add(8 * time.Hour), Equity: 105, Cash: 105},
	}
	if i, d := ValidateSnapshotIntegrity(good); i != -1 || d != "" {
		t.Errorf("good → %d/%q, want -1/空", i, d)
	}
	// 恒等式破坏：equity ≠ cash + market_value。
	corrupt := []store.EquitySnapshot{
		{Ts: t0, Equity: 100, Cash: 100},
		{Ts: t0.Add(8 * time.Hour), Equity: 999, Cash: 100, MarketValue: 5},
	}
	if i, d := ValidateSnapshotIntegrity(corrupt); i != 1 || !strings.Contains(d, "恒等式") {
		t.Errorf("corrupt → %d/%q, want 1/恒等式", i, d)
	}
	// ts 乱序。
	unsorted := []store.EquitySnapshot{
		{Ts: t0.Add(8 * time.Hour), Equity: 100, Cash: 100},
		{Ts: t0, Equity: 105, Cash: 105},
	}
	if i, d := ValidateSnapshotIntegrity(unsorted); i != 1 || !strings.Contains(d, "非单调") {
		t.Errorf("unsorted → %d/%q, want 1/非单调", i, d)
	}
	// 单快照（无 ts 序可查、无破坏）→ -1。
	if i, _ := ValidateSnapshotIntegrity(good[:1]); i != -1 {
		t.Errorf("单快照 → %d, want -1", i)
	}
}

// TestGateTrustQualifier 可信度判定：损坏 → 判定不采信（任何窗口）；成熟窗口
// 覆盖<90% → 判定不采信；未成熟窗口（<30 天）→ 覆盖检查不生效（PENDING 优先，
// 防「还没数据」误判「数据坏了」）；部分缺口 → 警示不覆盖；满覆盖 → 无覆盖。
func TestGateTrustQualifier(t *testing.T) {
	// 数据损坏任何时候都不采信（未成熟窗口也拦）。
	if s, _ := GateTrustQualifier(0, 0, "快照[1] ts 非单调（段边界乱序）"); s != GateDataAnomaly {
		t.Errorf("integrity 坏（未成熟）→ %q, want data_anomaly", s)
	}
	if s, _ := GateTrustQualifier(40, 0.8, ""); s != GateDataAnomaly {
		t.Errorf("成熟窗口 coverage=0.8 → %q, want data_anomaly", s)
	}
	if s, _ := GateTrustQualifier(40, 0, ""); s != GateDataAnomaly {
		t.Errorf("成熟窗口 coverage=0 → %q, want data_anomaly", s)
	}
	// 未成熟窗口：覆盖率 0 也不拦截（PENDING 优先）。
	if s, n := GateTrustQualifier(10, 0, ""); s != "" || n != "" {
		t.Errorf("未成熟 coverage=0 → %q/%q, want 空/空", s, n)
	}
	if s, n := GateTrustQualifier(40, 0.95, ""); s != "" || !strings.Contains(n, "95%") {
		t.Errorf("coverage=0.95 → %q/%q, want 空/含95%%", s, n)
	}
	if s, n := GateTrustQualifier(40, 1.0, ""); s != "" || n != "" {
		t.Errorf("coverage=1 → %q/%q, want 空/空", s, n)
	}
}

// TestEvaluateGate 判定门① 决策矩阵（D-062）。行：数据错误 → 窗口未满 → 零成交 → 按 TWR。
func TestEvaluateGate(t *testing.T) {
	cases := []struct {
		name     string
		days     float64
		twr      float64
		orders   int
		rejected int
		high     int
		twrErr   error
		want     string
		noteSub  string
	}{
		{"数据不足→pending", 10, 0, 0, 0, 0, ErrInsufficientData, GatePending, ""},
		{"数据异常→data_anomaly", 40, 0, 0, 0, 0, ErrDataAnomaly, GateDataAnomaly, ""},
		{"窗口未满30天→pending", 29, 50, 3, 0, 0, nil, GatePending, ""},
		{"零成交零拒单→env_no_window", 40, 0, 0, 0, 0, nil, GateEnvNoWindow, ""},
		{"零成交但有拒单→watch 有机会未进场", 40, 0, 0, 2, 0, nil, GateWatch, "拒单"},
		{"零成交有高费率时段→env 带排查提示", 40, 0, 0, 0, 3, nil, GateEnvNoWindow, "高费率"},
		{"TWR≥判定线4.0→pass", 40, 4.0, 1, 0, 0, nil, GatePass, ""},
		{"TWR=4.1→pass", 40, 4.1, 1, 0, 0, nil, GatePass, ""},
		{"PASS 含高费率环境红利警示", 40, 4.5, 1, 0, 3, nil, GatePass, "环境红利"},
		{"PASS 小样本警示", 40, 4.5, 1, 0, 0, nil, GatePass, "小样本"},
		{"PASS 多样本无警示", 40, 4.5, 5, 0, 0, nil, GatePass, ""},
		{"TWR=3.5 基线区间→watch", 40, 3.5, 1, 0, 0, nil, GateWatch, ""},
		{"TWR=3.2 基线下限边界→watch", 40, 3.2, 1, 0, 0, nil, GateWatch, ""},
		{"TWR=3.0 低于下限→fail", 40, 3.0, 1, 0, 0, nil, GateFail, ""},
		{"TWR负→fail 止投", 40, -5, 1, 0, 0, nil, GateFail, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, note := EvaluateGate(c.days, c.twr, c.twr, c.orders, c.rejected, c.high, c.twrErr, nil)
			if status != c.want {
				t.Fatalf("status = %q, want %q (note=%q)", status, c.want, note)
			}
			if c.noteSub != "" && !strings.Contains(note, c.noteSub) {
				t.Errorf("note = %q, want 含 %q", note, c.noteSub)
			}
		})
	}
}

// TestSettleLoopWritesEquitySnapshot：[对抗测试锚点 D-062] settle tick 必落一份
// equity 时点快照（sim_equity_snapshots 数据面）。删 driver.go settleLoop 中
// snapshotEquity 调用 → 本测试必红。内容对账：无持仓 → equity = cash = capital。
func TestSettleLoopWritesEquitySnapshot(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	if err := st.InitSimAccount(context.Background(), 100_000); err != nil {
		t.Fatalf("InitSimAccount: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time)
	done := make(chan error, 1)
	go func() { done <- d.settleLoop(ctx, ticks) }()

	ticks <- t0
	waitFor(t, time.Second, func() bool {
		st.mu.Lock()
		defer st.mu.Unlock()
		return len(st.snaps) >= 1
	})
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("settleLoop did not return after cancel")
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.snaps) == 0 {
		t.Fatal("settle tick 未落 equity 快照（D-062 测量数据面缺失）")
	}
	s := st.snaps[0]
	if !closeEnough(s.Equity, 100_000) || !closeEnough(s.Cash, 100_000) ||
		s.Realized != 0 || s.Unrealized != 0 || s.MarketValue != 0 {
		t.Errorf("snapshot = %+v, want equity=cash=100000 无持仓", s)
	}
}
