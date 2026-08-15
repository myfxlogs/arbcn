// 台账 RPC（M2-b §6）：手工录入出入金流水 + 按档位归因汇总。
// 资金动作永远人工（§1）——本层只读/写人工录入的流水，无任何自动执行能力。
package dashboard

import (
	"context"
	"fmt"
	"math"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/store"
)

// 台账分页约束（与 proto 注释一致）。
const (
	defaultLedgerLimit = 100
	maxLedgerLimit     = 500
)

// AddLedgerEntry 手工录入台账行（M2-b §6 出入金流水）。
// 只写人工录入的流水，无任何自动执行能力。date/channel/currency 必填，amount 有限。
func (s *Service) AddLedgerEntry(ctx context.Context, req *connect.Request[dashboardv1.AddLedgerEntryRequest]) (*connect.Response[dashboardv1.AddLedgerEntryResponse], error) {
	m := req.Msg
	if m.Date == nil || m.Channel == "" || m.Currency == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("dashboard: ledger: date, channel and currency required"))
	}
	if math.IsNaN(m.Amount) || math.IsInf(m.Amount, 0) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("dashboard: ledger: amount must be finite"))
	}
	id, err := s.st.InsertLedgerEntry(ctx, store.LedgerEntry{
		Date:     m.Date.AsTime(),
		Channel:  m.Channel,
		Currency: m.Currency,
		Amount:   m.Amount,
		FeeRate:  m.FeeRate,
		Tier:     m.Tier,
		Note:     m.Note,
	})
	if err != nil {
		return nil, storeErr(err)
	}
	return connect.NewResponse(&dashboardv1.AddLedgerEntryResponse{Id: id}), nil
}

// ListLedgerEntries 台账流水（date DESC, id DESC 分页）。
func (s *Service) ListLedgerEntries(ctx context.Context, req *connect.Request[dashboardv1.ListLedgerEntriesRequest]) (*connect.Response[dashboardv1.ListLedgerEntriesResponse], error) {
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = defaultLedgerLimit
	}
	if limit > maxLedgerLimit {
		limit = maxLedgerLimit
	}
	offset := int(req.Msg.Offset)
	entries, err := s.st.ListLedgerEntries(ctx, limit, offset)
	if err != nil {
		return nil, storeErr(err)
	}
	out := make([]*dashboardv1.LedgerEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, toLedgerEntry(e))
	}
	return connect.NewResponse(&dashboardv1.ListLedgerEntriesResponse{Entries: out}), nil
}

// LedgerSummary 按档位归因汇总（GROUP BY tier 简单分组；M2-b §6）。
func (s *Service) LedgerSummary(ctx context.Context, _ *connect.Request[dashboardv1.LedgerSummaryRequest]) (*connect.Response[dashboardv1.LedgerSummaryResponse], error) {
	items, err := s.st.LedgerSummary(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	out := make([]*dashboardv1.TierSummary, 0, len(items))
	for _, ts := range items {
		out = append(out, &dashboardv1.TierSummary{
			Tier:       ts.Tier,
			Inflow:     ts.Inflow,
			Outflow:    ts.Outflow,
			Net:        ts.Net,
			EntryCount: int32(ts.EntryCount),
		})
	}
	return connect.NewResponse(&dashboardv1.LedgerSummaryResponse{Items: out}), nil
}

// toLedgerEntry 映射 store.LedgerEntry → proto。
func toLedgerEntry(e store.LedgerEntry) *dashboardv1.LedgerEntry {
	return &dashboardv1.LedgerEntry{
		Id:       e.ID,
		Date:     timestamppb.New(e.Date),
		Channel:  e.Channel,
		Currency: e.Currency,
		Amount:   e.Amount,
		FeeRate:  e.FeeRate,
		Tier:     e.Tier,
		Note:     e.Note,
	}
}
