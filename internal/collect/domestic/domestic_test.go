package domestic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"

	"arbcn/internal/fact"
)

// fixtureServer 服务新浪逆回购脚本与 BOC 两跳页面（testdata/），无网络可测；校验 Referer 头。
func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/list="): // 新浪 hq 的 list= 在路径上（非查询参数）
			if r.Referer() != DefaultReferer {
				t.Errorf("Referer = %q, want %q", r.Referer(), DefaultReferer)
			}
			http.ServeFile(w, r, "testdata/sina_repo.txt")
		case r.URL.Path == "/fimarkets/lilv/fd31/":
			http.ServeFile(w, r, "testdata/boc_index.html")
		case r.URL.Path == "/fimarkets/lilv/fd31/202505/t20250520_25356440.html":
			http.ServeFile(w, r, "testdata/boc_rate.html")
		default:
			http.NotFound(w, r)
		}
	}))
}

// TestRepoFixture：两标的价（= 年化利率 %）与 Ts（CST 交易日收盘）逐项比对。
func TestRepoFixture(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewReverseRepo(Config{BaseURL: srv.URL, Repos: defaultRepos})
	if c.Kind() != fact.KindReverseRepo {
		t.Fatalf("Kind() = %q, want %q", c.Kind(), fact.KindReverseRepo)
	}
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(fs) != 2 {
		t.Fatalf("Poll = %d facts, want 2", len(fs))
	}
	wantTs := time.Date(2026, 8, 14, 15, 30, 0, 0, time.FixedZone("CST", 8*3600))
	want := []struct {
		symbol string
		value  float64
	}{{"GC001", 0.865}, {"R-001", 0.840}}
	for i, w := range want {
		f := fs[i]
		if err := f.Validate(); err != nil {
			t.Errorf("facts[%d]: Validate = %v", i, err)
		}
		if f.Venue != VenueSina || f.Unit != fact.UnitPctAnnualized {
			t.Errorf("facts[%d]: venue/unit = %q/%q", i, f.Venue, f.Unit)
		}
		if f.Symbol != w.symbol || f.Value != w.value {
			t.Errorf("facts[%d]: symbol/value = %q/%v, want %q/%v", i, f.Symbol, f.Value, w.symbol, w.value)
		}
		if !f.Ts.Equal(wantTs) {
			t.Errorf("facts[%d]: Ts = %v, want %v", i, f.Ts, wantTs)
		}
		if f.Src != "hq.sinajs.cn/list="+defaultRepos[i].Code {
			t.Errorf("facts[%d]: Src = %q", i, f.Src)
		}
	}
}

// TestRepoMissingCode：响应缺配置标的 → 整个 Poll 失败（口径同 exchange）。
func TestRepoMissingCode(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewReverseRepo(Config{BaseURL: srv.URL, Repos: []Repo{{Code: "sz000001", Symbol: "SZ1"}}})
	_, err := c.Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("Poll = %v, want error containing \"missing\"", err)
	}
}

// TestRepoBadPrice：价字段非法 → 失败（含代码定位）。
func TestRepoBadPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `var hq_str_sh204001="GC001,1,1,abc,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,2026-08-14,15:30:00,00";`)
	}))
	defer srv.Close()
	c := NewReverseRepo(Config{BaseURL: srv.URL, Repos: defaultRepos})
	_, err := c.Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "bad price") {
		t.Fatalf("Poll = %v, want error containing \"bad price\"", err)
	}
}

// TestRepoNoRepos：空标的清单 → 不发请求、返回空。
func TestRepoNoRepos(t *testing.T) {
	c := NewReverseRepo(Config{BaseURL: "http://127.0.0.1:1"})
	fs, err := c.Poll(context.Background())
	if err != nil || len(fs) != 0 {
		t.Fatalf("Poll = %d facts, %v; want 0, nil", len(fs), err)
	}
}

// TestBankRateFixture：两跳解析 7 档挂牌利率；"一年"取整存整取首个出现（0.95 非 0.65）。
func TestBankRateFixture(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewBankRate(Config{BankRateURL: srv.URL + "/fimarkets/lilv/fd31/"})
	if c.Kind() != fact.KindDepositRate {
		t.Fatalf("Kind() = %q, want %q", c.Kind(), fact.KindDepositRate)
	}
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	want := []struct {
		symbol string
		value  float64
	}{
		{"活期", 0.05}, {"三个月", 0.65}, {"半年", 0.85}, {"一年", 0.95},
		{"二年", 1.05}, {"三年", 1.25}, {"五年", 1.30},
	}
	if len(fs) != len(want) {
		t.Fatalf("Poll = %d facts, want %d", len(fs), len(want))
	}
	for i, w := range want {
		f := fs[i]
		if err := f.Validate(); err != nil {
			t.Errorf("facts[%d]: Validate = %v", i, err)
		}
		if f.Venue != VenueBOC || f.Unit != fact.UnitPctAnnualized {
			t.Errorf("facts[%d]: venue/unit = %q/%q", i, f.Venue, f.Unit)
		}
		if f.Symbol != w.symbol || f.Value != w.value {
			t.Errorf("facts[%d]: symbol/value = %q/%v, want %q/%v", i, f.Symbol, f.Value, w.symbol, w.value)
		}
		if f.Src != "boc/fimarkets/lilv/fd31 表2025-05-20" {
			t.Errorf("facts[%d]: Src = %q", i, f.Src)
		}
		if d := time.Since(f.Ts); d < -2*time.Second || d > 2*time.Second {
			t.Errorf("facts[%d]: Ts = %v, want ~now", i, f.Ts)
		}
	}
}

// TestBankRateGBK：GB18030 编码页面同样可解析（BOC 历史页面编码）。
func TestBankRateGBK(t *testing.T) {
	index, _ := simplifiedchinese.GB18030.NewEncoder().String(
		`<html><a href="./202505/t.html">人民币存款利率表2025-05-20</a></html>`)
	table, _ := simplifiedchinese.GB18030.NewEncoder().String(
		`<html><table><tr><td>活期</td><td>0.05</td></tr><tr><td>三个月</td><td>0.65</td></tr>` +
			`<tr><td>半年</td><td>0.85</td></tr><tr><td>一年</td><td>0.95</td></tr>` +
			`<tr><td>二年</td><td>1.05</td></tr><tr><td>三年</td><td>1.25</td></tr>` +
			`<tr><td>五年</td><td>1.30</td></tr></table></html>`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fimarkets/lilv/fd31/" {
			fmt.Fprint(w, index)
			return
		}
		fmt.Fprint(w, table)
	}))
	defer srv.Close()
	fs, err := NewBankRate(Config{BankRateURL: srv.URL + "/fimarkets/lilv/fd31/"}).Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(fs) != 7 || fs[6].Symbol != "五年" || fs[6].Value != 1.30 {
		t.Fatalf("Poll = %d facts, last = %+v", len(fs), fs)
	}
}

// TestBankRateMissingLink：索引页无利率表链接 → 失败。
func TestBankRateMissingLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<html><body>nothing here</body></html>`)
	}))
	defer srv.Close()
	_, err := NewBankRate(Config{BankRateURL: srv.URL}).Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no rate table link") {
		t.Fatalf("Poll = %v, want error containing \"no rate table link\"", err)
	}
}

// TestBankRateIncompleteTable：缺档 → 失败（部分表不可信）。
func TestBankRateIncompleteTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fmt.Fprint(w, `<html><a href="./t.html">人民币存款利率表2025-05-20</a></html>`)
			return
		}
		fmt.Fprint(w, `<html><table><tr><td>活期</td><td>0.05</td></tr></table></html>`)
	}))
	defer srv.Close()
	_, err := NewBankRate(Config{BankRateURL: srv.URL}).Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("Poll = %v, want error containing \"incomplete\"", err)
	}
}

// TestBankRateHTTPError：任一跳非 200 → 失败（含 status）。
func TestBankRateHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer srv.Close()
	_, err := NewBankRate(Config{BankRateURL: srv.URL}).Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("Poll = %v, want error containing status 404", err)
	}
}

// TestFromEnv：默认标的/BOC 地址；自定义标的；bank off。
func TestFromEnv(t *testing.T) {
	cfg := FromEnv(func(string) string { return "" })
	if len(cfg.Repos) != 2 || cfg.BankRateURL != DefaultBankRateURL {
		t.Fatalf("defaults = %v %q", cfg.Repos, cfg.BankRateURL)
	}
	cfg = FromEnv(func(k string) string {
		if k == "ARBCN_REPO_SYMBOLS" {
			return "sh204007:GC007"
		}
		return ""
	})
	if len(cfg.Repos) != 1 || cfg.Repos[0] != (Repo{Code: "sh204007", Symbol: "GC007"}) {
		t.Fatalf("custom repos = %v", cfg.Repos)
	}
	cfg = FromEnv(func(k string) string {
		if k == "ARBCN_BANK_RATE_URL" {
			return bankRateDisableFlag
		}
		return ""
	})
	if cfg.BankRateURL != "" {
		t.Fatalf("bank off = %q, want empty", cfg.BankRateURL)
	}
}

// TestAll：repo 5m + bank_rate 1h；BankRateURL 空 → 仅 repo。
func TestAll(t *testing.T) {
	ns := All(Config{BankRateURL: DefaultBankRateURL})
	if len(ns) != 2 || ns[0].Name != "repo" || ns[0].Interval != 5*time.Minute ||
		ns[1].Name != "bank_rate" || ns[1].Interval != time.Hour {
		t.Fatalf("All() = %d sources", len(ns))
	}
	ns = All(Config{BankRateURL: ""})
	if len(ns) != 1 || ns[0].Name != "repo" {
		t.Fatalf("All(no bank) = %d sources", len(ns))
	}
}
