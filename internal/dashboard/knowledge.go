// 市场结构经验库数据面（D-046）：knowledge_match 签名匹配信号 + ListKnowledgeEntries 浏览 RPC。
// 只读证据表面（practices #20）——系统只匹配与呈现，不吸收、不改 verdict；吸收 = 人工 + D#。
package dashboard

import (
	"context"
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
	out := make([]*dashboardv1.KnowledgeEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, toKnowledgeEntryProto(e))
	}
	return connect.NewResponse(&dashboardv1.ListKnowledgeEntriesResponse{Entries: out}), nil
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
		poolLatest := map[string]float64{} // venue\x00symbol → 30d 内最新值
		for _, f := range defi {
			if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
				continue
			}
			poolLatest[f.Venue+"\x00"+f.Symbol] = f.Value // ts ASC，覆盖即最新
		}
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
		latestByVenue := map[string]float64{} // venue\x00symbol → 最新值
		for _, f := range funding {
			if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
				continue
			}
			latestByVenue[f.Venue+"\x00"+f.Symbol] = f.Value // ts ASC，覆盖即最新
		}
		venueVals := map[string][]float64{} // symbol → 各 venue 瞬时费率
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
func toKnowledgeEntryProto(e store.KnowledgeEntry) *dashboardv1.KnowledgeEntry {
	out := &dashboardv1.KnowledgeEntry{
		Id:             e.ID,
		Ts:             timestamppb.New(e.Ts),
		Signature:      e.Signature,
		Venue:          e.Venue,
		Symbol:         e.Symbol,
		Verdict:        e.Verdict,
		Rationale:      e.Rationale,
		Source:         e.Source,
		Status:         e.Status,
		ValidationNote: e.ValidationNote,
	}
	if e.ValidatedAt != nil {
		out.ValidatedAt = timestamppb.New(*e.ValidatedAt)
	}
	return out
}
