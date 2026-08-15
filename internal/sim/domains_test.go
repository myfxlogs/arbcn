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
