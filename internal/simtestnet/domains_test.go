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

// testnetOrderPaths 是 simtestnet 允许的测试网/demo 下单端点路径字面量（executor*.go，
// D-098 测试网执行层）。写路径只允许这三条；probe.go/config.go 保持只读。
var testnetOrderPaths = []string{
	"/fapi/v1/order",             // Binance USDT-M 期货 testnet 下单（query 签名）
	"/api/v5/trade/order",        // OKX demo 下单（body 签名 + x-simulated-trading:1）
	"/api/v5/trade/cancel-order", // OKX demo 撤单
}

// bannedOrderTokens 任何情况下都不许出现的下单 token：非测试网/非标准路径、或命名不符
// 现有签名风格。出现 → 必红（误配/拼错主网路径 = 潜在真金路径）。
var bannedOrderTokens = []string{
	"/fapi/v1/order/test",       // 现货 test 端点（USDT-M 期货 testnet 无此物）
	"/api/v5/trade/place-order", // 不存在的路径（拼错）
	"newOrder",                  // 命名风格不符（现有 HMAC 签名走路径字面量）
}

// TestOnlyTestnetOrderEndpoints：[对抗测试锚点 §9.4 S3 + D-034 ② 修订] simtestnet 由
// 「零下单路径」放宽为「仅测试网/demo 下单路径」——写路径只允许 testnetOrderPaths，且只
// 出现在 executor*.go（probe/config 保持只读）。域仍由 TestOnlyTestnetDomains 把关
// （任何主网 host → 必红）。这是 D-034 ② 修订的精确条款：testnet key 可下单，但仅限
// 测试网/demo，SIMULATED 隔离与主网禁入不变。
func TestOnlyTestnetOrderEndpoints(t *testing.T) {
	for _, f := range nonTestGoFiles(t) {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		s := string(body)
		for _, tok := range bannedOrderTokens {
			if strings.Contains(s, tok) {
				t.Errorf("%s 含禁下单 token %q（仅允许 testnetOrderPaths：%v）", f, tok, testnetOrderPaths)
			}
		}
		// 下单端点 token 只允许在 executor*.go 出现（probe/config 保持只读；出现 → 必红）。
		if !strings.HasPrefix(f, "executor") {
			for _, tok := range []string{"/order", "/trade", "cancelOrder", "placeOrder"} {
				if strings.Contains(s, tok) {
					t.Errorf("%s 含下单端点 %q（写路径仅限 executor*.go；probe/config 只读）", f, tok)
				}
			}
		}
	}
	// 正向锚点：executor 文件并集必须含全部允许的下单路径（写路径真实存在、可 grep，
	// D-034 ② 修订落档）。删除任一下单端点 → 必红。
	all := ""
	for _, f := range []string{"executor.go", "executor_okx.go"} {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		all += string(body)
	}
	for _, want := range testnetOrderPaths {
		if !strings.Contains(all, want) {
			t.Errorf("executor 文件缺测试网下单端点 %q（写路径不可 grep = 交付不完整）", want)
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
