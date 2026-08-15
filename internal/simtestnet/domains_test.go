package simtestnet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nonTestGoFiles 返回包内非 _test.go 的 .go 文件。
func nonTestGoFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read pkg dir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			files = append(files, e.Name())
		}
	}
	return files
}

// TestOnlyTestnetDomains：[对抗测试锚点 §9.4 S3] simtestnet 只允许 testnet/demo 域。
// 出现任何主网交易域 → 必红（key 承载层一旦放开主网域 = 潜在真金路径）。
func TestOnlyTestnetDomains(t *testing.T) {
	allowed := []string{
		"testnet.binancefuture.com", // Binance USDT-M futures testnet
		"www.okx.com",               // OKX demo（x-simulated-trading:1 头）
	}
	mainnet := []string{
		"fapi." + "binance.com",
		"api." + "binance.com",
		"data-api." + "binance.vision",
		"aws." + "okx.com",
	}
	for _, f := range nonTestGoFiles(t) {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		s := string(body)
		for _, m := range mainnet {
			if strings.Contains(s, m) {
				t.Errorf("%s 含主网交易域 %q（§9.4 S3：simtestnet 仅 testnet/demo 域）", f, m)
			}
		}
		// 反向：出现的 https:// 域必须 ⊆ 允许集（防拼错/混入新域）。
		for _, tok := range strings.Fields(s) {
			if !strings.HasPrefix(tok, "https://") {
				continue
			}
			host := strings.TrimPrefix(tok, "https://")
			if i := strings.IndexAny(host, "/ \t\"',;:)"); i >= 0 {
				host = host[:i]
			}
			if !containsStr(allowed, host) {
				t.Errorf("%s 含未允许域 https://%s（允许集：%v）", f, host, allowed)
			}
		}
	}
}

// TestNoOrderEndpoints：[对抗测试锚点 §9.4 S3] simtestnet 零下单路径——
// 不得出现任何 order/place/trade 下单端点片段。出现 → 必红。
func TestNoOrderEndpoints(t *testing.T) {
	orderTokens := []string{
		"/fapi/v1/order", "/fapi/v1/order/test",
		"/api/v5/trade/order", "/api/v5/trade/place-order",
		"placeOrder", "newOrder", "cancelOrder",
		"/order", "/orders", "/trade",
	}
	for _, f := range nonTestGoFiles(t) {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		s := string(body)
		for _, tok := range orderTokens {
			if strings.Contains(s, tok) {
				t.Errorf("%s 含下单端点片段 %q（§9.4 S3：simtestnet 零下单路径）", f, tok)
			}
		}
	}
}

// TestProbePathsReadOnly：探针只访问只读端点（time / balance），无写/下单端点路径。
func TestProbePathsReadOnly(t *testing.T) {
	body, err := os.ReadFile("probe.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{"/fapi/v1/time", "/fapi/v2/balance", "/api/v5/public/time", "/api/v5/account/balance"} {
		if !strings.Contains(s, want) {
			t.Errorf("probe.go 缺只读端点 %q", want)
		}
	}
}

func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
