// Package manual：人工录入降级通道（docs/design/02-monitor-architecture.md §5——
// IV / 银行利率等采集受阻时降级人工录入）。
// 简单 HTTP 端点：POST JSON fact（kind/venue/symbol/value，可选 unit/ts/src）→ 校验 → 直写 Store。
// 校验与 collector 同口径（fact.Validate + 值有限）；接 Store 写库，挂载接线属 M1-h。
// 铁律：只写事实、不写规则；资金动作永远人工（§1）。
package manual

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// entry 是 POST 请求体（Value 指针区分缺失与 0）。
type entry struct {
	Kind   string   `json:"kind"`
	Venue  string   `json:"venue"`
	Symbol string   `json:"symbol"`
	Value  *float64 `json:"value"`
	Unit   string   `json:"unit"`
	Ts     string   `json:"ts"`  // RFC3339；空 = now
	Src    string   `json:"src"` // 空 = "manual"
}

// Handler 处理人工录入请求；Store 未接线（nil）时返回 503（依赖缺失，不静默吞）。
type Handler struct {
	Store store.Store
}

// NewHandler 构造人工录入处理器。
func NewHandler(st store.Store) *Handler { return &Handler{Store: st} }

// ServeHTTP 实现 http.Handler：仅 POST；校验失败 400，Store 错误 500，成功 200。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "manual: POST only"})
		return
	}
	if h.Store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "manual: store not wired"})
		return
	}
	var e entry
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&e); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manual: bad JSON: " + err.Error()})
		return
	}
	f, err := e.fact(time.Now())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.Store.InsertFacts(r.Context(), []fact.Fact{f}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manual: insert: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "fact": f})
}

// fact 校验并组装 Fact；非法输入返回错误（400 语义）。
func (e entry) fact(now time.Time) (fact.Fact, error) {
	if !fact.ValidKind(e.Kind) {
		return fact.Fact{}, errors.New("manual: unknown kind " + e.Kind)
	}
	if e.Venue == "" || e.Symbol == "" {
		return fact.Fact{}, errors.New("manual: venue and symbol required")
	}
	if e.Value == nil || math.IsNaN(*e.Value) || math.IsInf(*e.Value, 0) {
		return fact.Fact{}, errors.New("manual: value required and must be finite")
	}
	ts := now
	if e.Ts != "" {
		t, err := time.Parse(time.RFC3339, e.Ts)
		if err != nil {
			return fact.Fact{}, errors.New("manual: bad ts (want RFC3339): " + e.Ts)
		}
		ts = t
	}
	src := e.Src
	if src == "" {
		src = "manual"
	}
	return fact.Fact{
		Kind:   e.Kind,
		Venue:  e.Venue,
		Symbol: e.Symbol,
		Value:  *e.Value,
		Unit:   e.Unit,
		Ts:     ts,
		Src:    src,
	}, nil
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
