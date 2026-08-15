// M3-b 规则→Signal 运行驱动（04-m3-spec §9.2 S1 + §9.3 S2）。
//
// 本包保持零网络零密钥（M3-a 复审验证的 D-010 属性，§9.4 纵深防御）：Driver 只做
// 纯计算——规则命中 → 按 §3.1.1 映射表组装 Signal → Simulator.Generate 落库；
// 8h 结算调度按 (symbol,venue) 分组喂入真实市场 funding 事实。网络/key 承载在
// internal/simtestnet（S3，物理隔离）。
package sim

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// settleInterval 8h 三班资金费率结算周期（04-m3-spec §3.2）。
const settleInterval = 8 * time.Hour

// reportEveryTicks 每 7 个 8h tick（周）渲染一次周频统计报告（§9.5）。
const reportEveryTicks = 7

// signalMapper 按规则名组装 Signal 的函数（映射表值类型）。
type signalMapper func(ctx context.Context, d *Driver, r store.Rule, h store.EntityHit) (*Signal, error)

// signalMappers 规则名 → Signal 组装（04-m3-spec §3.1.1 表编码，不可变包内常量）。
//
// [对抗测试锚点] §9.2 S1：删除任一映射 → sim/driver_test.go
// TestDriverFundingHitCreatesOrder（funding_warn→funding_hedge）/TestDriverRepoBuildsOrder
// 必红。未在表中的规则（defi_large_tier_change / ladder_trap / iv_opportunity /
// usdcnh_buy_line / collector_heartbeat / nonstable_quote_change 等）→ 不建单（宁缺毋滥），
// 但命中标的在白名单时仍可映射 carry_asset（§9.6）。
var signalMappers = map[string]signalMapper{
	"funding_warn":         fundingHedgeSignal,
	"funding_critical":     fundingHedgeSignal,
	"trx_funding_positive": fundingHedgeSignal,
	"reverse_repo_timing":  repoSignal,
}

// Driver 是规则→Signal 驱动（M3-b §9.2）。
type Driver struct {
	st  store.Store
	cfg Config
	now func() time.Time
	log *slog.Logger
	sim *Simulator

	// Probe 每次 settle tick 执行的只读连通性探针（M3-b §9.7 ⑤，S3）；main.go 注入，
	// nil = 跳过（key 不可用 / 降级）。失败仅 warn 不阻断 settle 主循环（D-032 同口径）。
	Probe func(context.Context) error
}

// NewDriver 构造驱动；st 必须非 nil。cfg 由调用方保证合法（非法 → main 不构造，nil 降级）。
func NewDriver(st store.Store, cfg Config) *Driver {
	d := &Driver{st: st, cfg: cfg, now: time.Now, log: slog.Default(), sim: New(st, cfg)}
	d.sim.Now = d.now
	return d
}

// OnRuleActive 规则 armed→active 回调（接 rule.Config.OnActive，M3-b §9.7 ③）：
// 每个命中实体按 §3.1.1 组装 Signal → Generate 落库（suggested/rejected 均落，负样本）。
// 无法映射的规则不建单。单次激活一实体会生成至多一单（armed→active 转变语义防重复）。
func (d *Driver) OnRuleActive(ctx context.Context, r store.Rule, entities []store.EntityHit) error {
	for _, h := range entities {
		sig, ok, err := d.buildSignal(ctx, r, h)
		if err != nil {
			return fmt.Errorf("sim driver: %s: %w", r.Name, err)
		}
		if !ok {
			continue
		}
		if _, err := d.sim.Generate(ctx, *sig); err != nil {
			return fmt.Errorf("sim driver: %s: generate: %w", r.Name, err)
		}
	}
	return nil
}

// buildSignal 按映射表组装 Signal。返回 ok=false = 不建单（未知规则且未白名单）。
func (d *Driver) buildSignal(ctx context.Context, r store.Rule, h store.EntityHit) (*Signal, bool, error) {
	if m, ok := signalMappers[r.Name]; ok {
		sig, err := m(ctx, d, r, h)
		if err != nil {
			return nil, false, err
		}
		return sig, true, nil
	}
	// carry 白名单（§9.6）：命中标的 ∈ CarryWhitelist → carry_asset。
	if slices.Contains(d.cfg.CarryWhitelist, h.Symbol) {
		sig, err := d.carrySignal(ctx, r, h)
		if err != nil {
			return nil, false, err
		}
		return sig, true, nil
	}
	return nil, false, nil
}

// fundingHedgeSignal 组装 funding_hedge Signal（04-m3-spec §3.1.1 首行 + §9.2 诚实标注）。
func fundingHedgeSignal(ctx context.Context, d *Driver, r store.Rule, h store.EntityHit) (*Signal, error) {
	// 现货/永续最新价 = LatestFacts(kind=ticker, venue, symbol) 最新一条。
	// 诚实标注（§9.2）：系统无现货 collector，ticker 即永续价；现货/永续腿存在性由
	// 门禁把关（>0），basis/现货腿差留真实执行层，M3 只验证 funding 机制。
	var ref float64
	if fs, err := d.st.LatestFacts(ctx, fact.KindTicker, h.Venue, h.Symbol); err == nil && len(fs) > 0 {
		ref = fs[0].Value
	}
	return &Signal{
		RuleName: r.Name, Kind: store.SimKindFundingHedge,
		Symbol: h.Symbol, Venue: h.Venue,
		RefPrice: ref, SpotPrice: ref, PerpPrice: ref,
		FundingAnn: h.Value, ExpectedSpread: 0, // 由 FundingAnn 回填（SignalToOrder）
		Notional: 0, Ts: d.now(),
	}, nil
}

// repoSignal 组装 repo Signal（04-m3-spec §3.1.1 第二行）：全局模式命中，单信号。
func repoSignal(ctx context.Context, d *Driver, r store.Rule, h store.EntityHit) (*Signal, error) {
	// 当日回购年化 = 人工补录 reverse_repo 事实最新值；无 → 命中 value 兜底。
	spread := h.Value
	if fs, err := d.st.LatestFacts(ctx, fact.KindReverseRepo, "", ""); err == nil && len(fs) > 0 {
		spread = fs[0].Value
	}
	return &Signal{
		RuleName: r.Name, Kind: store.SimKindRepo,
		Symbol: "GC001", Venue: "domestic", // 交易所逆回购（现金等价，面值 100）
		RefPrice: 100, ExpectedSpread: spread, FundingAnn: spread,
		Notional: 0, Ts: d.now(),
	}, nil
}

// carrySignal 组装 carry_asset Signal（§9.6）：标的已在白名单（调用方已校验）。
func (d *Driver) carrySignal(ctx context.Context, r store.Rule, h store.EntityHit) (*Signal, error) {
	spread := h.Value // 命中值兜底（生息年化）
	if fs, err := d.st.LatestFacts(ctx, fact.KindDefiRate, h.Venue, h.Symbol); err == nil && len(fs) > 0 {
		spread = fs[0].Value
	}
	ref := 1.0 // 稳定币面值锚（sUSDe/USDe ≈ 1）；有 ticker 则覆盖
	if fs, err := d.st.LatestFacts(ctx, fact.KindTicker, h.Venue, h.Symbol); err == nil && len(fs) > 0 {
		ref = fs[0].Value
	}
	return &Signal{
		RuleName: r.Name, Kind: store.SimKindCarryAsset,
		Symbol: h.Symbol, Venue: d.venue(h.Venue),
		RefPrice: ref, ExpectedSpread: spread, FundingAnn: spread,
		Notional: 0, CarryWhite: true, Ts: d.now(),
	}, nil
}

// venue 归一 venue；空（全局命中）→ 默认模拟 venue。
func (d *Driver) venue(v string) string {
	if v == "" {
		return "sim_local"
	}
	return v
}

// RunSettleLoop 每 8h tick 结算 open funding 腿 + 每 7 tick 渲染周报 + 执行探针钩子；
// 阻塞至 ctx 取消。真实运行 = time.NewTicker(settleInterval)。
func (d *Driver) RunSettleLoop(ctx context.Context) error {
	t := time.NewTicker(settleInterval)
	defer t.Stop()
	return d.settleLoop(ctx, t.C)
}

// settleLoop 可注入 tick 通道的结算循环（测试注入固定 tick）。
func (d *Driver) settleLoop(ctx context.Context, ticks <-chan time.Time) error {
	n := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticks:
			if err := d.settleOnce(ctx); err != nil {
				d.log.Warn("sim settle once failed", "err", err)
			}
			n++
			if n%reportEveryTicks == 0 {
				if err := d.renderReport(ctx); err != nil {
					d.log.Warn("sim report render failed", "err", err)
				}
			}
			if d.Probe != nil {
				if err := d.Probe(ctx); err != nil {
					d.log.Warn("simtestnet probe failed", "err", err)
				}
			}
		}
	}
}

// settleOnce 执行一轮结算：列出全部 open funding 腿 → 按 (symbol,venue) 分组 →
// 每组取 LatestFacts(kind=funding, venue, symbol) 最新值结算；无事实 skip+warn。
//
// [对抗测试锚点] §9.3 S2：按 (symbol,venue) 分组（不按 symbol 合并）→ 删除分组中的
// venue 维度 → sim/driver_test.go TestSettleByVenue 跨 venue 污染断言必红。
func (d *Driver) settleOnce(ctx context.Context) error {
	legs, err := d.st.ListOpenSimPositions(ctx, "", "")
	if err != nil {
		return fmt.Errorf("sim settle: list open: %w", err)
	}
	seen := map[[2]string]bool{}
	var keys [][2]string
	for _, l := range legs {
		if !l.Funding {
			continue
		}
		key := [2]string{l.Symbol, l.Venue}
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool { // 确定性日志顺序
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	for _, key := range keys {
		sym, venue := key[0], key[1]
		fs, err := d.st.LatestFacts(ctx, fact.KindFunding, venue, sym)
		if err != nil {
			return fmt.Errorf("sim settle: latest funding %s@%s: %w", sym, venue, err)
		}
		if len(fs) == 0 {
			d.log.Warn("sim settle: no funding fact, skip", "symbol", sym, "venue", venue)
			continue
		}
		rate := fs[0].Value // 真实市场公开 funding（§9.0 裁决：非 testnet）
		if _, err := d.sim.SettleFunding(ctx, sym, venue, rate); err != nil {
			return fmt.Errorf("sim settle: settle %s@%s: %w", sym, venue, err)
		}
	}
	return nil
}
