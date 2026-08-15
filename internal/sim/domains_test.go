package sim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoMainnetDomains：internal/sim 包内禁止出现真实交易主网域（04-m3-spec §2/§7
// 审查门禁，编码为测试）。真实主网域 = 潜在真金路径；只允许 testnet/demo/sim 域。
// 豁免 *_test.go 自身（域名字符串出现在本测试），只审非测试源文件。
func TestNoMainnetDomains(t *testing.T) {
	// 主网域列表（连接符拼装，避免字面量污染 grep 面；仅测试文件本身豁免仍安全）。
	mainnet := []string{
		"fapi." + "binance.com",
		"api." + "binance.com",
		"data-api." + "binance.vision",
		"www." + "okx.com",
		"aws." + "okx.com",
	}
	bad := map[string]bool{}
	for _, d := range mainnet {
		bad[d] = true
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read pkg dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(".", e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, d := range mainnet {
			if strings.Contains(string(body), d) {
				t.Errorf("%s 含真实主网域 %q（§2 审查门禁：internal/sim 禁主网域）", e.Name(), d)
			}
		}
	}
}

// TestNoNetworkImports：internal/sim 零网络（04-m3-spec §9.4 S3 纵深防御）——
// 非测试源文件不得 import net/http / net / crypto/tls（纯计算，网络承载在 simtestnet）。
func TestNoNetworkImports(t *testing.T) {
	banned := []string{`"net/http"`, `"net"`, `"crypto/tls"`, `"crypto/hmac"`}
	for _, f := range nonTestGoFiles(t) {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, b := range banned {
			if strings.Contains(string(body), b) {
				t.Errorf("%s import %s（§9.4：sim 保持零网络零密钥）", f, b)
			}
		}
	}
}

// TestNoOrderEndpoints：internal/sim 零下单端点路径（04-m3-spec §9.8 domains_test 增项）。
// sim 纯计算，不应出现任何下单端点字符串；出现 → 必红。
func TestNoOrderEndpoints(t *testing.T) {
	orderTokens := []string{
		"/fapi/v1/order", "/api/v5/trade/order",
		"placeOrder", "newOrder", "cancelOrder",
	}
	for _, f := range nonTestGoFiles(t) {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, tok := range orderTokens {
			if strings.Contains(string(body), tok) {
				t.Errorf("%s 含下单端点片段 %q（§9.8：sim 禁下单端点）", f, tok)
			}
		}
	}
}

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
