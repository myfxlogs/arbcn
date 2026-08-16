package quote

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

// TestOnlyPublicMarketDomains：[对抗测试锚点 D-056 Part B] quote 只允许公共行情 WS 域。
// 出现任何交易/账户域（fapi/api/aws/data-api）→ 必红（报价层一旦触交易域 = 潜在真金路径）。
// 与 simtestnet.TestOnlyTestnetDomains 同型，但本包是公共行情（无密钥），允许主网公共域。
func TestOnlyPublicMarketDomains(t *testing.T) {
	allowed := []string{
		"fstream.binancefuture.com", // Binance USDT-M futures 公共行情合流 WS（部署机实测可用）
		"ws.okx.com",                // OKX v5 公共行情 WS
		"localhost",                 // 测试注入的 httptest 域（feed_test 用）
	}
	blocked := []string{
		"fapi." + "binance.com",
		"api." + "binance.com",
		"data-api." + "binance.vision",
		"aws." + "okx.com",
		"www." + "okx.com",
	}
	for _, f := range nonTestGoFiles(t) {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		s := string(body)
		for _, b := range blocked {
			if strings.Contains(s, b) {
				t.Errorf("%s 含禁止交易/账户域 %q（quote 仅公共行情 WS）", f, b)
			}
		}
		// 反向：出现的 wss:// / https:// 域必须 ⊆ 允许集（防拼错/混入新域）。
		for _, tok := range strings.Fields(s) {
			proto := ""
			switch {
			case strings.HasPrefix(tok, "wss://"):
				proto = "wss://"
			case strings.HasPrefix(tok, "https://"):
				proto = "https://"
			default:
				continue
			}
			host := strings.TrimPrefix(tok, proto)
			if i := strings.IndexAny(host, "/ \t\"',;:)"); i >= 0 {
				host = host[:i]
			}
			if !containsStr(allowed, host) {
				t.Errorf("%s 含未允许域 %s%s（允许集：%v）", f, proto, host, allowed)
			}
		}
	}
}

// TestNoKeysOrOrders：[对抗测试锚点 D-056 Part B] quote 无密钥、无下单路径——
// 公共行情只需要读；任何 key/order 字段出现 → 必红。
func TestNoKeysOrOrders(t *testing.T) {
	bad := []string{"apikey", "apiKey", "APIKey", "secret", "Secret", "passphrase", "Passphrase"}
	for _, f := range nonTestGoFiles(t) {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		s := string(body)
		for _, tok := range bad {
			if strings.Contains(s, tok) {
				t.Errorf("%s 含密钥字段 %q（quote 公共行情，无密钥）", f, tok)
			}
		}
	}
}

// TestFeedSymbolMapping：symbol → 交易所流名映射（binance 合流 / okx instId）。
func TestFeedSymbolMapping(t *testing.T) {
	b := newBinanceFeed([]string{"BTC", "ETH", "TRX"})
	u := b.streamURL()
	if !strings.Contains(u, "btcusdt@miniTicker") || !strings.Contains(u, "trxusdt@miniTicker") {
		t.Errorf("binance streamURL 缺 miniTicker 流：%s", u)
	}
	sub := newOKXFeed([]string{"BTC", "TRX"}).subscribeMsg()
	if !strings.Contains(string(sub), `"BTC-USDT-SWAP"`) || !strings.Contains(string(sub), `"TRX-USDT-SWAP"`) {
		t.Errorf("okx subscribe 缺 instId：%s", sub)
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
