// Package quote：秒级实时报价（D-056 Part B）。交易所公共行情 WebSocket → Hub 内存
// 存储 → SSE 推前端（EventSource）。上下游选型不一（对话 #73 已答复业主）：
//   上游（交易所→后端）= WebSocket：公共行情只提供 WS 推送（无 SSE），REST 秒级轮询
//     会被限频；无密钥（D-010 合规，公共端点）。
//   下游（后端→前端）= SSE：单向 server→client 推报价，前端无上行需求；原生
//     EventSource 自动重连、零新依赖、过现有 HTTP mux。不引 websocket 库（用不上双向）。
// 报价流只做展示，不喂策略、不落库（facts ticker 1min 仍是 equity/positions 的真相源，
// 诚实标注；本包是独立网络承载层，如 internal/simtestnet，不受 internal/sim 零网络约束）。
// 订阅集合由 main 传入（exchange.FromEnv 的 ARBCN_COLLECT_SYMBOLS，P3 单一来源）。
package quote

import (
	"sort"
	"sync"
)

// Price 单标的最新价快照（Hub 存储单元 + SSE 负载；ts_ms 交易所毫秒时间戳）。
type Price struct {
	Venue  string  `json:"venue"`  // binance / okx
	Symbol string  `json:"symbol"` // 基础币种，如 BTC / ETH / TRX
	Price  float64 `json:"price"`
	TsMs   int64   `json:"ts_ms"`
}

// Hub 报价内存存储（并发安全）：WS feed 写入 Update，SSE handler 每 1s 读 Snapshot
// diff 推送（下游无需推送通道——秒级 cadence 由 handler 自己的 ticker 决定，少一层
// 订阅机制，B 原则：直接对应问题本质）。
type Hub struct {
	mu     sync.RWMutex
	prices map[string]Price // key = venue + "|" + symbol
}

// NewHub 构造空 Hub（未收到任何行情前，Snapshot 返回空——SSE 首推空快照，前端标 —）。
func NewHub() *Hub {
	return &Hub{prices: map[string]Price{}}
}

// Update 写入单标的最新价（WS feed 每帧调用）。
func (h *Hub) Update(p Price) {
	h.mu.Lock()
	h.prices[p.Venue+"|"+p.Symbol] = p
	h.mu.Unlock()
}

// Snapshot 返回当前全部报价（按 venue|symbol 排序，diff 计算稳定）。
func (h *Hub) Snapshot() []Price {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]Price, 0, len(h.prices))
	for _, p := range h.prices {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Venue != out[j].Venue {
			return out[i].Venue < out[j].Venue
		}
		return out[i].Symbol < out[j].Symbol
	})
	return out
}
