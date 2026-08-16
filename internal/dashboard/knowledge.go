// 市场结构经验库数据面（D-046）：knowledge_match 签名匹配信号 + ListKnowledgeEntries 浏览 RPC。
// 只读证据表面（practices #20）——系统只匹配与呈现，不吸收、不改 verdict；吸收 = 人工 + D#。
package dashboard

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/knowledge"
	"arbcn/internal/store"
)

// ListKnowledgeEntries 经验库浏览（D-046）：返回全部条目（signature ASC）。
// 只读呈现；吸收走 internal/knowledge.Defaults + D#，不提供自动写入通道。
func (s *Service) ListKnowledgeEntries(ctx context.Context, _ *connect.Request[dashboardv1.ListKnowledgeEntriesRequest]) (*connect.Response[dashboardv1.ListKnowledgeEntriesResponse], error) {
	entries, err := s.st.ListKnowledgeEntries(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	// D-059 复核自动证据：对每条经验 × 当前数据生成核验证据（只读，供人工复核裁决）。
	// 数据面故障 → 降级「自动核验暂不可用」，不阻断经验浏览（浏览是主、证据是辅）。
	evidence := map[string]string{}
	if ev, err := s.knowledgeEvidence(ctx, entries); err != nil {
		for _, e := range entries {
			evidence[e.Signature] = "自动核验暂不可用（数据面故障）"
		}
	} else {
		evidence = ev
	}
	out := make([]*dashboardv1.KnowledgeEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, toKnowledgeEntryProto(e, evidence[e.Signature]))
	}
	return connect.NewResponse(&dashboardv1.ListKnowledgeEntriesResponse{Entries: out}), nil
}

// ReviewKnowledgeEntry 人工复核经验条目（D-054）：写 validated_at=now + verdict +
// validation_note。只改该 signature 条目的判定记录（呈现面），不改任何规则/门禁
// （D-046 边界，practices #20 同源）——复核 = 决策层人工在环动作，系统永不自动复核。
func (s *Service) ReviewKnowledgeEntry(ctx context.Context, req *connect.Request[dashboardv1.ReviewKnowledgeEntryRequest]) (*connect.Response[dashboardv1.ReviewKnowledgeEntryResponse], error) {
	if req.Msg.Signature == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("signature required"))
	}
	if req.Msg.ValidationNote == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("validation_note required（复核结论留痕）"))
	}
	switch req.Msg.Status {
	case knowledge.StatusActive, knowledge.StatusSuperseded, knowledge.StatusRetracted:
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("status must be one of %s/%s/%s", knowledge.StatusActive, knowledge.StatusSuperseded, knowledge.StatusRetracted))
	}
	if err := s.st.ReviewKnowledgeEntry(ctx, req.Msg.Signature, req.Msg.Status, req.Msg.Verdict, req.Msg.ValidationNote); err != nil {
		return nil, storeErr(err)
	}
	return connect.NewResponse(&dashboardv1.ReviewKnowledgeEntryResponse{}), nil
}

// knowledgeMatches 经验库签名匹配（D-046 信号 5）：确定性探测器对当前事实算签名 → 命中
// 已吸收条目 → 产出「当前情况 + 上回判定」只读 insight（category=knowledge, severity=info）。
// 经验库空 / 无对应签名 → 无信号（宁缺毋滥）。存储层故障 → 错误（整体 Unavailable）。
func (s *Service) knowledgeMatches(ctx context.Context, now time.Time, defi []fact.Fact) ([]insightItem, error) {
	entries, err := s.st.ListKnowledgeEntries(ctx)
	if err != nil {
		return nil, err
	}
	bySig := map[string]store.KnowledgeEntry{}
	for _, e := range entries {
		bySig[e.Signature] = e
	}
	if len(bySig) == 0 {
		return nil, nil
	}
	var out []insightItem

	// funding 30d 数据面（尖峰 / 跨所分歧共用）。
	funding, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindFunding, From: now.Add(-oppAvgWindow)})
	if err != nil {
		return nil, err
	}

	// 5a. funding 尖峰陷阱：每 (venue,symbol) 瞬时 vs 30d 均值。
	if e, ok := bySig[knowledge.SignatureFundingSpikeTrap]; ok {
		latest, avg := fundingLatestAvg(funding)
		keys := make([]string, 0, len(avg))
		for k := range avg {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !knowledge.FundingSpikeTrap(latest[key], avg[key], knowledge.Factor) {
				continue
			}
			venue, symbol := splitEntityKey(key)
			out = append(out, insightItem{
				id:       "knowledge_match:" + e.Signature + ":" + venue + ":" + symbol,
				category: "knowledge",
				severity: "info",
				title:    "命中经验「资金费率尖峰陷阱」",
				detail:   fmt.Sprintf("%s@%s 瞬时费率 %.2f%% 为 30 日均值 %.2f%% 的 %.1f 倍；上回判定【%s】：%s", symbol, venue, latest[key], avg[key], latest[key]/avg[key], e.Verdict, e.Rationale),
				actions:  []string{"对照上回判定核实当前费率结构，需调整判定走 D#"},
			})
		}
	}

	// 5b. DeFi 单池尖峰：截面中位数×2（复用 signal 2 的 defi 数据面，与 defiAnomalies 同口径）。
	if e, ok := bySig[knowledge.SignatureDefiSinglePoolSpike]; ok {
		poolLatest := latestByVenueSymbol(defi) // venue\x00symbol → 30d 内最新值
		for _, key := range knowledge.DefiPoolSpikes(poolLatest, knowledge.Factor) {
			venue, symbol := splitEntityKey(key)
			out = append(out, insightItem{
				id:       "knowledge_match:" + e.Signature + ":" + venue + ":" + symbol,
				category: "knowledge",
				severity: "info",
				title:    "命中经验「单池利率尖峰」",
				detail:   fmt.Sprintf("%s@%s 利率 %.2f%% 为截面尖峰（中位数×%.0f 判定）；上回判定【%s】：%s", symbol, venue, poolLatest[key], knowledge.Factor, e.Verdict, e.Rationale),
				actions:  []string{"按截面中位数核对是否为尖峰，实盘配置前按 D-028 核实均值"},
			})
		}
	}

	// 5c. 跨所资金费率分歧：同 symbol ≥2 venue 最新值，max−min ≥ 阈值（一正一负或大差距）。
	if e, ok := bySig[knowledge.SignatureFundingCrossVenueDiverg]; ok {
		latestByVenue := latestByVenueSymbol(funding) // venue\x00symbol → 最新值
		venueVals := map[string][]float64{}           // symbol → 各 venue 瞬时费率
		for key, v := range latestByVenue {
			_, symbol := splitEntityKey(key)
			venueVals[symbol] = append(venueVals[symbol], v)
		}
		syms := make([]string, 0, len(venueVals))
		for sym := range venueVals {
			syms = append(syms, sym)
		}
		sort.Strings(syms)
		for _, sym := range syms {
			if !knowledge.CrossVenueDivergence(venueVals[sym], knowledge.MinCrossVenueSpread()) {
				continue
			}
			lo, hi := minMax(venueVals[sym])
			out = append(out, insightItem{
				id:       "knowledge_match:" + e.Signature + ":" + sym,
				category: "knowledge",
				severity: "info",
				title:    "命中经验「跨所资金费率分歧」",
				detail:   fmt.Sprintf("%s 跨所费率差距 %.1f 个百分点（%.2f%% ~ %.2f%%）；上回判定【%s】：%s", sym, hi-lo, lo, hi, e.Verdict, e.Rationale),
				actions:  []string{"先核实是否为真实市场结构，再判定是否双边利用（走 D#）"},
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out, nil
}

// knowledgeEvidence 复核自动证据（D-059）：对已吸收条目，用当前数据面重跑确定性探测器，
// 产出「当前命中/未命中 + 关键数值 + 阈值」只读证据文本，供人工复核裁决。boundary——
// practices #20：系统只呈现证据，不自动改判定/吸收；判定仍是 ReviewKnowledgeEntry 人工
// 在环点击才写 validated_at。
// 数据面查询故障 → 错误返回（调用方降级「自动核验暂不可用」，浏览不阻断）；
// 某条所需数据缺失 → 该条给「无法自动核验」，不编造。
func (s *Service) knowledgeEvidence(ctx context.Context, entries []store.KnowledgeEntry) (map[string]string, error) {
	now := s.now()
	// funding 30d 数据面（尖峰 / 跨所分歧共用；与 knowledgeMatches 同口径，P3 同源）。
	funding, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindFunding, From: now.Add(-oppAvgWindow)})
	if err != nil {
		return nil, err
	}
	// defi 30d 数据面（单池尖峰；与 signal 2 同口径）。
	defi, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindDefiRate, From: now.Add(-defiLookback)})
	if err != nil {
		return nil, err
	}

	latestFunding, avgFunding := fundingLatestAvg(funding)
	fundingLatest := latestByVenueSymbol(funding)
	defiLatest := latestByVenueSymbol(defi)

	out := make(map[string]string, len(entries))
	for _, e := range entries {
		switch e.Signature {
		case knowledge.SignatureFundingSpikeTrap:
			out[e.Signature] = evidenceSpikeTrap(latestFunding, avgFunding)
		case knowledge.SignatureDefiSinglePoolSpike:
			out[e.Signature] = evidenceDefiSpike(defiLatest)
		case knowledge.SignatureFundingCrossVenueDiverg:
			out[e.Signature] = evidenceCrossVenue(fundingLatest)
		default:
			out[e.Signature] = "该签名无自动核验探测器，请对照 rationale 人工复核"
		}
	}
	return out, nil
}

// evidenceSpikeTrap 资金费率尖峰陷阱的当前数据证据：每 (venue,symbol) 瞬时 vs 30d 均值
// 比值，命中阈值（×factor）即列；未命中给最接近处，供复核人判断「经验还成立吗」。
func evidenceSpikeTrap(latest, avg map[string]float64) string {
	if len(avg) == 0 {
		return "当前无 funding 数据，无法自动核验"
	}
	keys := make([]string, 0, len(avg))
	for k := range avg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var hits []string
	maxKey, maxRatio := keys[0], latest[keys[0]]/avg[keys[0]]
	for _, key := range keys {
		ratio := latest[key] / avg[key]
		if ratio > maxRatio {
			maxKey, maxRatio = key, ratio
		}
		if knowledge.FundingSpikeTrap(latest[key], avg[key], knowledge.Factor) {
			venue, symbol := splitEntityKey(key)
			hits = append(hits, fmt.Sprintf("%s@%s 瞬时 %.2f%% vs 均值 %.2f%%（×%.1f）", symbol, venue, latest[key], avg[key], ratio))
		}
	}
	if len(hits) > 0 {
		return "自动核验：当前命中（瞬时/均值 ≥ ×2.0）——" + strings.Join(hits, "；")
	}
	venue, symbol := splitEntityKey(maxKey)
	return fmt.Sprintf("自动核验：当前未命中——最接近 %s@%s 瞬时 %.2f%% vs 均值 %.2f%%（×%.1f），未达阈值 ×%.1f", symbol, venue, latest[maxKey], avg[maxKey], maxRatio, knowledge.Factor)
}

// evidenceDefiSpike 单池利率尖峰的当前数据证据：截面中位数×factor 命中列 key，未命中
// 给截面最高值。样本 <3 → DefiPoolSpikes 返回空，按未命中呈现。
func evidenceDefiSpike(poolLatest map[string]float64) string {
	if len(poolLatest) == 0 {
		return "当前无 defi_rate 数据，无法自动核验"
	}
	if hits := knowledge.DefiPoolSpikes(poolLatest, knowledge.Factor); len(hits) > 0 {
		parts := make([]string, 0, len(hits))
		for _, key := range hits {
			venue, symbol := splitEntityKey(key)
			parts = append(parts, fmt.Sprintf("%s@%s %.2f%%", symbol, venue, poolLatest[key]))
		}
		return "自动核验：当前命中（截面中位数 ×2.0 判定）——" + strings.Join(parts, "；")
	}
	maxKey, maxVal := "", math.Inf(-1)
	for k, v := range poolLatest {
		if v > maxVal {
			maxKey, maxVal = k, v
		}
	}
	venue, symbol := splitEntityKey(maxKey)
	return fmt.Sprintf("自动核验：当前未命中——截面最高 %s@%s %.2f%%，未达中位数 ×%.1f 判定线", symbol, venue, maxVal, knowledge.Factor)
}

// evidenceCrossVenue 跨所资金费率分歧的当前数据证据：仅统计同 symbol ≥2 venue 者，
// max−min ≥ minSpread 命中；未命中给差距最大处。
func evidenceCrossVenue(fundingLatest map[string]float64) string {
	venueVals := map[string][]float64{} // symbol → 各 venue 最新值
	for key, v := range fundingLatest {
		_, symbol := splitEntityKey(key)
		venueVals[symbol] = append(venueVals[symbol], v)
	}
	syms := make([]string, 0, len(venueVals))
	for sym, vals := range venueVals {
		if len(vals) >= 2 {
			syms = append(syms, sym)
		}
	}
	sort.Strings(syms)
	if len(syms) == 0 {
		return "当前无跨所数据（需同 symbol ≥2 venue），无法自动核验"
	}
	minSpread := knowledge.MinCrossVenueSpread()
	var hits []string
	maxSym, maxSpread := "", 0.0
	for _, sym := range syms {
		lo, hi := minMax(venueVals[sym])
		if spread := hi - lo; spread > maxSpread {
			maxSym, maxSpread = sym, spread
		}
		if knowledge.CrossVenueDivergence(venueVals[sym], minSpread) {
			hits = append(hits, fmt.Sprintf("%s %.1fpp（%.2f%% ~ %.2f%%）", sym, hi-lo, lo, hi))
		}
	}
	if len(hits) > 0 {
		return fmt.Sprintf("自动核验：当前命中（差距 ≥ %.1fpp）——%s", minSpread, strings.Join(hits, "；"))
	}
	lo, hi := minMax(venueVals[maxSym])
	return fmt.Sprintf("自动核验：当前未命中——差距最大 %s %.1fpp（%.2f%% ~ %.2f%%），低于阈值 %.1fpp", maxSym, maxSpread, lo, hi, minSpread)
}

// fundingLatestAvg 每 (venue,symbol) 最新值 + 30d 均值（输入 ts ASC，覆盖即最新）。
// 跳过 NaN/Inf（practices #7）。
func fundingLatestAvg(funding []fact.Fact) (latest, avg map[string]float64) {
	latest = map[string]float64{}
	avg = map[string]float64{}
	count := map[string]int{}
	for _, f := range funding {
		if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
			continue
		}
		key := f.Venue + "\x00" + f.Symbol
		latest[key] = f.Value
		avg[key] += f.Value
		count[key]++
	}
	for key, sum := range avg {
		avg[key] = sum / float64(count[key])
	}
	return latest, avg
}

// latestByVenueSymbol facts（QueryFacts 按 ts ASC）→ "venue\x00symbol" 最新值；跳过
// NaN/Inf（practices #7）。数据准备共享（D-059）：knowledgeMatches 命中信号与
// knowledgeEvidence 复核证据同源，一处实现两处用（P3）。
func latestByVenueSymbol(facts []fact.Fact) map[string]float64 {
	out := map[string]float64{}
	for _, f := range facts {
		if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
			continue
		}
		out[f.Venue+"\x00"+f.Symbol] = f.Value // ts ASC，覆盖即最新
	}
	return out
}

// splitEntityKey 拆分 "venue\x00symbol" → venue, symbol。
func splitEntityKey(key string) (string, string) {
	parts := strings.SplitN(key, "\x00", 2)
	if len(parts) != 2 {
		return key, ""
	}
	return parts[0], parts[1]
}

// minMax 返回切片最小/最大值（非空输入）。
func minMax(xs []float64) (min, max float64) {
	min, max = xs[0], xs[0]
	for _, v := range xs[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

// toKnowledgeEntryProto 映射 store.KnowledgeEntry → proto。
func toKnowledgeEntryProto(e store.KnowledgeEntry, evidence string) *dashboardv1.KnowledgeEntry {
	out := &dashboardv1.KnowledgeEntry{
		Id:              e.ID,
		Ts:              timestamppb.New(e.Ts),
		Signature:       e.Signature,
		Venue:           e.Venue,
		Symbol:          e.Symbol,
		Verdict:         e.Verdict,
		Rationale:       e.Rationale,
		Source:          e.Source,
		Status:          e.Status,
		ValidationNote:  e.ValidationNote,
		CurrentEvidence: evidence,
	}
	if e.ValidatedAt != nil {
		out.ValidatedAt = timestamppb.New(*e.ValidatedAt)
	}
	return out
}
