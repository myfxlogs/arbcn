package exchange

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"arbcn/internal/fact"
)

// historyFixtureServer 用本地 httptest 模拟两个公开历史端点（无网络）：
//   - /fapi/v1/fundingRate：8h 间隔、费率 0.0001，limit=1000 翻页（365d 窗口 → 2 页）
//   - /api/v5/public/funding-history：8h 间隔、费率 0.0001，after 分页（新→旧）
func historyFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/fapi/v1/fundingRate", func(w http.ResponseWriter, r *http.Request) {
		start, _ := strconv.ParseInt(r.URL.Query().Get("startTime"), 10, 64)
		end, _ := strconv.ParseInt(r.URL.Query().Get("endTime"), 10, 64)
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit == 0 {
			limit = 1000
		}
		var rows []binanceFundingRateHistory
		for t := start; t <= end; t += 8 * 3600 * 1000 {
			rows = append(rows, binanceFundingRateHistory{
				Symbol: "BTCUSDT", FundingTime: t, FundingRate: "0.00010000",
			})
		}
		if len(rows) > limit {
			rows = rows[:limit]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
	})

	mux.HandleFunc("/api/v5/public/funding-history", func(w http.ResponseWriter, r *http.Request) {
		after := r.URL.Query().Get("after")
		now := time.Now().UnixMilli()
		var rows []okxFundingHistory
		for i := 0; i < 120; i++ { // 120 条 × 8h = 40 天，足 2 页（100 + 20）
			t := now - int64(i)*8*3600*1000
			rows = append(rows, okxFundingHistory{
				InstID: "BTC-USDT-SWAP", FundingRate: "0.00010000",
				FundingTime: strconv.FormatInt(t, 10),
			})
		}
		if after != "" {
			afterMs, _ := strconv.ParseInt(after, 10, 64)
			filtered := rows[:0]
			for _, row := range rows {
				ts, _ := strconv.ParseInt(row.FundingTime, 10, 64)
				if ts < afterMs {
					filtered = append(filtered, row)
				}
			}
			rows = filtered
		}
		if len(rows) > 100 {
			rows = rows[:100]
		}
		data, _ := json.Marshal(rows)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(okxResp{Code: "0", Data: data})
	})

	return httptest.NewServer(mux)
}

// TestHistoryAnnualizes：[对抗测试锚点 §9.5 S4] 年化折算正确——删 annualize 调用
// （如直接落原始费率）→ 本测试 Value 断言必红。0.0001 费率、8h 结算 → 10.95%。
func TestHistoryAnnualizes(t *testing.T) {
	srv := historyFixtureServer(t)
	defer srv.Close()
	cfg := Config{Symbols: []string{"BTC"}, BinanceHistoryBaseURL: srv.URL, OKXBaseURL: srv.URL}

	// Binance：365d 窗口 → 每 8h 一条（含两端点，365×3+1=1096 条）；
	// 全部 0.0001 × 1095 × 100 = 10.95。
	binFacts, err := NewBinanceFundingHistory(cfg, 365).Poll(context.Background())
	if err != nil {
		t.Fatalf("binance history poll: %v", err)
	}
	if len(binFacts) != 1096 {
		t.Fatalf("binance facts = %d, want 1096（365d / 8h 含端点，翻页拉满）", len(binFacts))
	}
	for _, f := range binFacts {
		if f.Kind != fact.KindFunding || f.Venue != VenueBinance || f.Symbol != "BTC" ||
			f.Unit != fact.UnitPctAnnualized {
			t.Fatalf("fact = %+v, want funding/binance/BTC/pct_annualized", f)
		}
		if math.Abs(f.Value-10.95) > 1e-9 {
			t.Fatalf("value = %v, want 10.95（0.0001×1095×100 年化）", f.Value)
		}
		if f.Ts.Equal(time.Unix(0, 0)) {
			t.Fatalf("Ts = 零值, want 结算时刻")
		}
	}

	// OKX：120 条 × 8h；费率同 → 10.95%。
	okxFacts, err := NewOKXFundingHistory(cfg, 365).Poll(context.Background())
	if err != nil {
		t.Fatalf("okx history poll: %v", err)
	}
	if len(okxFacts) != 120 {
		t.Fatalf("okx facts = %d, want 120（after 分页拉满 2 页）", len(okxFacts))
	}
	for _, f := range okxFacts {
		if f.Venue != VenueOKX || math.Abs(f.Value-10.95) > 1e-9 {
			t.Fatalf("fact = %+v, want okx value 10.95", f)
		}
	}
}

// TestHistoryDaysZero：days<=0 → Poll 返回空（禁用），不触网。
func TestHistoryDaysZero(t *testing.T) {
	cfg := Config{Symbols: []string{"BTC"}, BinanceHistoryBaseURL: "http://127.0.0.1:1", OKXBaseURL: "http://127.0.0.1:1"}
	f, err := NewBinanceFundingHistory(cfg, 0).Poll(context.Background())
	if err != nil || len(f) != 0 {
		t.Fatalf("binance days=0: facts=%d err=%v, want 空/无错", len(f), err)
	}
	f2, err := NewOKXFundingHistory(cfg, 0).Poll(context.Background())
	if err != nil || len(f2) != 0 {
		t.Fatalf("okx days=0: facts=%d err=%v, want 空/无错", len(f2), err)
	}
}

// TestHistoryBinancePagination：365d 窗口 1096 条 > 单页 1000 → 必须翻页拉满；
// 若 paginate 只取首页（不推进 startTime）→ 1096 断言必红。
func TestHistoryBinancePagination(t *testing.T) {
	srv := historyFixtureServer(t)
	defer srv.Close()
	cfg := Config{Symbols: []string{"BTC"}, BinanceHistoryBaseURL: srv.URL}
	fs, err := NewBinanceFundingHistory(cfg, 365).Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(fs) != 1096 {
		t.Fatalf("facts = %d, want 1096（翻页拉满）", len(fs))
	}
}

// TestHistoryOKXPagination：120 条 > 单页 100 → 必须 after 翻页；只取首页 → 120 断言必红。
func TestHistoryOKXPagination(t *testing.T) {
	srv := historyFixtureServer(t)
	defer srv.Close()
	cfg := Config{Symbols: []string{"BTC"}, OKXBaseURL: srv.URL}
	fs, err := NewOKXFundingHistory(cfg, 365).Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(fs) != 120 {
		t.Fatalf("facts = %d, want 120（after 分页拉满）", len(fs))
	}
}
