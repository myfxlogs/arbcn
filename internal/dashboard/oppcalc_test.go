package dashboard

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// TestFundingCardSpikeTrap：ETH@okx 瞬时 9.14% vs 均值 4.16%，0.3% 摩擦 → 尖峰陷阱。
// [对抗测试锚点] 删公式（BreakEvenDays=f×365/Inst / NetAnnualized=Avg30−f×365/30）
// 必红；删尖峰分支 → 判定漂移必红。
func TestFundingCardSpikeTrap(t *testing.T) {
	c := fundingCard("okx", "ETHUSDT", 9.14, 4.16, 0.3)
	if want := 0.3 * 365 / 9.14; math.Abs(c.BreakEvenDays-want) > 1e-9 {
		t.Errorf("BreakEvenDays = %v, want %v（保本 ~12 天）", c.BreakEvenDays, want)
	}
	if want := 4.16 - 0.3*365/oppAvgDays; math.Abs(c.NetAnnualized-want) > 1e-9 {
		t.Errorf("NetAnnualized = %v, want %v（30 日持有扣摩擦）", c.NetAnnualized, want)
	}
	if c.Rating != RatingTrap {
		t.Errorf("Rating = %q, want trap（9.14 > 2×4.16 尖峰）", c.Rating)
	}
	if !strings.Contains(c.Narrative, "尖峰陷阱") {
		t.Errorf("Narrative = %q, want 含「尖峰陷阱」", c.Narrative)
	}
	if c.Kind != store.SimKindFundingHedge || c.Friction != 0.3 {
		t.Errorf("Kind/Friction = %s/%.2f, want funding_hedge/0.30", c.Kind, c.Friction)
	}
}

// TestFundingCardGrab：均值 15%（funding 窗口档）扣摩擦净 11.35% > 4.5% → 可抓（与 D-016 一致）。
func TestFundingCardGrab(t *testing.T) {
	c := fundingCard("binance", "BTCUSDT", 16, 15, 0.3)
	if c.Rating != RatingGrab {
		t.Errorf("Rating = %q, want grab（净 11.35 > 4.5）", c.Rating)
	}
	if !strings.Contains(c.Narrative, "可抓") {
		t.Errorf("Narrative = %q, want 含「可抓」", c.Narrative)
	}
}

// TestFundingCardBreakeven：均值 4.0% 扣摩擦净 0.35%，介于 0 与 4.5、非尖峰 → 打平。
func TestFundingCardBreakeven(t *testing.T) {
	c := fundingCard("binance", "BTCUSDT", 5, 4.0, 0.3)
	if c.Rating != RatingBreakeven {
		t.Errorf("Rating = %q, want breakeven", c.Rating)
	}
	if !strings.Contains(c.Narrative, "打平") {
		t.Errorf("Narrative = %q, want 含「打平」", c.Narrative)
	}
}

// TestFundingCardNetNegative：均值 2% 扣摩擦净 −1.65 ≤0 → 坑（费率不覆盖成本）。
func TestFundingCardNetNegative(t *testing.T) {
	c := fundingCard("binance", "BTCUSDT", 3, 2.0, 0.3)
	if c.Rating != RatingTrap {
		t.Errorf("Rating = %q, want trap（净 ≤0）", c.Rating)
	}
	if !strings.Contains(c.Narrative, "不抓") {
		t.Errorf("Narrative = %q, want 含「不抓」", c.Narrative)
	}
}

// TestFundingCardInstNegative：当前费率 ≤0 → 坑（开仓反向付费）。
func TestFundingCardInstNegative(t *testing.T) {
	c := fundingCard("binance", "BTCUSDT", -1.5, 5.0, 0.3)
	if c.Rating != RatingTrap {
		t.Errorf("Rating = %q, want trap（瞬时 ≤0）", c.Rating)
	}
}

// TestFundingCardNoAvg30：均值样本不足（NaN）→ 打平观望（宁缺毋滥），不判尖峰。
func TestFundingCardNoAvg30(t *testing.T) {
	c := fundingCard("binance", "NEWUSDT", 8.0, math.NaN(), 0.3)
	if c.Rating != RatingBreakeven {
		t.Errorf("Rating = %q, want breakeven（均值不足观望）", c.Rating)
	}
	if !math.IsNaN(c.NetAnnualized) {
		t.Errorf("NetAnnualized = %v, want NaN", c.NetAnnualized)
	}
	if !strings.Contains(c.Narrative, "观望") {
		t.Errorf("Narrative = %q, want 含「观望」", c.Narrative)
	}
}

// TestRatingFrictionSensitivity：「不抓」结论对合理摩擦区间（0.1–0.5）稳健——低均值费率
// 在区间内绝不判「可抓」（计划明示：标签精细度 trap/breakeven 受摩擦影响，卡上显式展示
// 摩擦假设）；尖峰判定与摩擦无关恒 trap。业主核实普通主户费率 0.3 在区间内。
func TestRatingFrictionSensitivity(t *testing.T) {
	for _, f := range []float64{0.1, 0.3, 0.5} {
		spike := fundingCard("okx", "ETHUSDT", 9.14, 4.16, f)
		if spike.Rating != RatingTrap {
			t.Errorf("friction=%.1f 尖峰 Rating = %q, want trap（尖峰判定与摩擦无关）", f, spike.Rating)
		}
		low := fundingCard("binance", "BTCUSDT", 3, 2.0, f)
		if low.Rating == RatingGrab {
			t.Errorf("friction=%.1f 均值 2%% Rating = %q, want 非 grab（「不抓」对摩擦稳健）", f, low.Rating)
		}
	}
	// 摩擦对净年化的方向正确：0.1 → 净 0.78 打平；0.5 → 净 −4.08 坑。
	if c := fundingCard("binance", "BTCUSDT", 3, 2.0, 0.1); c.Rating != RatingBreakeven {
		t.Errorf("friction=0.1 净 >0 应打平, got %q", c.Rating)
	}
	if c := fundingCard("binance", "BTCUSDT", 3, 2.0, 0.5); c.Rating != RatingTrap {
		t.Errorf("friction=0.5 净 ≤0 应坑, got %q", c.Rating)
	}
}

// TestCarryCard：生息无摩擦，净年化 ≈ 均值。
func TestCarryCard(t *testing.T) {
	c := carryCard("ethena-usde", "sUSDE", 5.2, 5.0)
	if c.Friction != 0 {
		t.Errorf("Friction = %v, want 0（持有生息无方向摩擦）", c.Friction)
	}
	if math.Abs(c.NetAnnualized-5.0) > 1e-9 {
		t.Errorf("NetAnnualized = %v, want 5.0", c.NetAnnualized)
	}
	if c.Rating != RatingGrab {
		t.Errorf("Rating = %q, want grab（5.0 > 4.5）", c.Rating)
	}
}

// TestRepoCard：时点利率即机会本身；季末冲高是机会不是陷阱（尖峰判定跳过），低于基档 → 打平。
func TestRepoCard(t *testing.T) {
	if c := repoCard("sina", "GC001", 5.5, 2.5); c.Rating != RatingGrab {
		t.Errorf("高时点利率 Rating = %q, want grab", c.Rating)
	}
	if c := repoCard("sina", "GC001", 2.1, 2.0); c.Rating != RatingBreakeven {
		t.Errorf("低时点利率 Rating = %q, want breakeven", c.Rating)
	}
	if c := repoCard("sina", "GC001", 2.1, 2.0); !strings.Contains(c.Narrative, "时点") {
		t.Errorf("Narrative = %q, want 标注时点属性", c.Narrative)
	}
}

// TestMeanFacts：求和均值 + NaN 跳过 + 空 → NaN（practices #7）。
func TestMeanFacts(t *testing.T) {
	if m := meanFacts([]fact.Fact{{Value: 3.0}, {Value: 5.0}}); m != 4.0 {
		t.Errorf("mean = %v, want 4", m)
	}
	if m := meanFacts([]fact.Fact{{Value: math.NaN()}, {Value: 5.0}}); m != 5.0 {
		t.Errorf("NaN 跳过 mean = %v, want 5", m)
	}
	if m := meanFacts(nil); !math.IsNaN(m) {
		t.Errorf("空 mean = %v, want NaN", m)
	}
}

// TestListOppCards：端到端——funding 尖峰 + defi 生息 + repo 时点 → RPC 返回卡，公式与
// 摩擦明示正确。删公式/删编排任一数据面 → 必红。
func TestListOppCards(t *testing.T) {
	ctx := context.Background()
	now := t0
	st := &fakeStore{facts: fundingSpikeFacts(now)}
	st.facts = append(st.facts,
		// defi 稳定生息（均值样本充足 → 净年化 = 均值）。
		fact.Fact{Kind: fact.KindDefiRate, Venue: "ethena-usde", Symbol: "sUSDE", Value: 5.0, Ts: now.Add(-time.Hour)},
		fact.Fact{Kind: fact.KindDefiRate, Venue: "ethena-usde", Symbol: "sUSDE", Value: 5.0, Ts: now.Add(-48 * time.Hour)},
		// repo 时点逆回购。
		fact.Fact{Kind: fact.KindReverseRepo, Venue: "sina", Symbol: "GC001", Value: 2.1, Ts: now.Add(-time.Hour)},
	)
	svc := New(st, nil, nil, nil)
	svc.Now = func() time.Time { return now }
	client := newTestServer(t, svc)

	resp, err := client.ListOppCards(ctx, connect.NewRequest(&dashboardv1.ListOppCardsRequest{}))
	if err != nil {
		t.Fatalf("ListOppCards: %v", err)
	}
	byKey := map[string]*dashboardv1.OpportunityCard{}
	for _, c := range resp.Msg.Cards {
		byKey[c.Kind+"/"+c.Venue+"/"+c.Symbol] = c
	}

	// funding ETH@okx：20×4.0 + 9.14 → avg≈4.24，瞬时 9.14 > 2×avg → 尖峰陷阱。
	f := byKey["funding_hedge/okx/ETHUSDT"]
	if f == nil {
		t.Fatalf("缺 funding_hedge/okx/ETHUSDT; cards=%+v", byKey)
	}
	if f.Rating != RatingTrap {
		t.Errorf("ETH@okx rating = %q, want trap", f.Rating)
	}
	avg := (20*4.0 + 9.14) / 21
	if want := avg - 0.3*365/oppAvgDays; math.Abs(f.NetAnnualized-want) > 0.01 {
		t.Errorf("ETH@okx NetAnnualized = %.3f, want %.3f", f.NetAnnualized, want)
	}
	if want := 0.3 * 365 / 9.14; math.Abs(f.BreakEvenDays-want) > 0.5 {
		t.Errorf("ETH@okx BreakEvenDays = %.3f, want ~%.3f（12 天保本）", f.BreakEvenDays, want)
	}
	// 摩擦假设明示（业主核实值：普通主户双 taker）。
	if f.FrictionPct != 0.3 {
		t.Errorf("ETH@okx FrictionPct = %v, want 0.3", f.FrictionPct)
	}
	if !strings.Contains(f.Narrative, "0.30%") && !strings.Contains(f.Narrative, "0.3%") {
		t.Errorf("Narrative = %q, want 明示摩擦假设", f.Narrative)
	}

	// carry + repo 卡存在且判定正确。
	if c := byKey["carry_asset/ethena-usde/sUSDE"]; c == nil || c.Rating != RatingGrab {
		t.Errorf("carry 卡 = %+v, want grab（5.0 > 4.5）", c)
	}
	if c := byKey["repo/sina/GC001"]; c == nil || c.Rating != RatingBreakeven {
		t.Errorf("repo 卡 = %+v, want breakeven（2.1 < 4.5）", c)
	}
}

// fundingSpikeFacts 构造 funding 尖峰事实集：20 个 ~4.0% 样本 + 最新 9.14%（avg≈4.24）。
func fundingSpikeFacts(now time.Time) []fact.Fact {
	out := []fact.Fact{}
	base := now.Add(-30 * 24 * time.Hour)
	for i := 0; i < 20; i++ {
		out = append(out, fact.Fact{
			Kind: fact.KindFunding, Venue: "okx", Symbol: "ETHUSDT", Value: 4.0,
			Ts: base.Add(time.Duration(i) * 30 * time.Hour),
		})
	}
	out = append(out, fact.Fact{
		Kind: fact.KindFunding, Venue: "okx", Symbol: "ETHUSDT", Value: 9.14, Ts: now.Add(-time.Hour),
	})
	return out
}
