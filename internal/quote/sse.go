// sse.go：报价 SSE 出口（D-056 Part B 下游）。后端 → 前端单向推送，EventSource 消费。
// 连接即推全量快照；其后每 1s 推有变化的 tick（diff 自上次推送，减带宽）。前端无上行，
// 不引 websocket 库（用不上双向）；原生 EventSource 自动重连，无需服务端重连逻辑。
// 纯展示：不喂策略、不落库（facts ticker 1min 仍是 equity/positions 真相源）。
package quote

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// sseInterval SSE 推送间隔（秒级实时报价的最小节拍）。
const sseInterval = 1 * time.Second

// Handler 返回 /quote/stream SSE handler。
func (h *Hub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		// 长连接豁免 WriteTimeout：HTTP server 配了 30s 写超时（R5#6），SSE 是持续流，
		// 不清除 deadline 会在 30s 后强制断开。ResponseController 置零 = 移除写超时
		//（Go 1.20+；idle/read 超时仍由 server 兜底）。
		if rc := http.NewResponseController(w); rc.SetWriteDeadline(time.Time{}) != nil {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		// 连接即推全量快照（前端立即有数，不等第一个 tick）。
		last := map[string]Price{}
		for _, p := range h.Snapshot() {
			if err := writeSSE(w, fl, p); err != nil {
				return
			}
			last[p.Venue+"|"+p.Symbol] = p
		}

		ticker := time.NewTicker(sseInterval)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return // 客户端断开（EventSource 会自动重连建立新连接）
			case <-ticker.C:
				for _, p := range h.Snapshot() {
					prev, ok := last[p.Venue+"|"+p.Symbol]
					if ok && prev.Price == p.Price && prev.TsMs == p.TsMs {
						continue // 无变化不重复推
					}
					if err := writeSSE(w, fl, p); err != nil {
						return
					}
					last[p.Venue+"|"+p.Symbol] = p
				}
			}
		}
	}
}

// writeSSE 写一条 data: <json>\n\n 并 flush；写失败（客户端断开）返回 error。
func writeSSE(w http.ResponseWriter, fl http.Flusher, p Price) error {
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("quote sse marshal: %w", err)
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
		return err
	}
	fl.Flush()
	return nil
}
