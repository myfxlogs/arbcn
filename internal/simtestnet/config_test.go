package simtestnet

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadMissingSimulatedMarker：[对抗测试锚点 §9.4 S3] 缺 SIMULATED=true → 拒绝加载。
// 删 LoadFromMap 的 SIMULATED 校验 → 本测试必红。
func TestLoadMissingSimulatedMarker(t *testing.T) {
	if _, err := LoadFromMap(map[string]string{
		"SIM_BINANCE_API_KEY": "abc",
		"SIM_BINANCE_SECRET":  "def",
	}); err == nil {
		t.Fatal("LoadFromMap(缺 SIMULATED) = nil, want error（缺标记拒绝加载）")
	}
}

// TestLoadRejectsNonTrueMarker：SIMULATED 非 true（false/空）→ 拒绝。
func TestLoadRejectsNonTrueMarker(t *testing.T) {
	for _, v := range []string{"false", "0", "yes"} {
		if _, err := LoadFromMap(map[string]string{
			SimulatedMarker: v,
			"SIM_OKX_API_KEY": "abc",
		}); err == nil {
			t.Fatalf("LoadFromMap(SIMULATED=%q) = nil, want error", v)
		}
	}
}

// TestLoadFromMapOK：SIMULATED=true + 完整 key → Config 解析。
func TestLoadFromMapOK(t *testing.T) {
	cfg, err := LoadFromMap(map[string]string{
		SimulatedMarker:    "true",
		"SIM_BINANCE_API_KEY": "bkey",
		"SIM_BINANCE_SECRET":  "bsec",
		"SIM_OKX_API_KEY":     "okey",
		"SIM_OKX_SECRET":      "osec",
		"SIM_OKX_PASSPHRASE":  "opass",
	})
	if err != nil {
		t.Fatalf("LoadFromMap: %v", err)
	}
	if cfg.BinanceAPIKey != "bkey" || cfg.BinanceSecret != "bsec" ||
		cfg.OKXAPIKey != "okey" || cfg.OKXSecret != "osec" || cfg.OKXPassphrase != "opass" {
		t.Fatalf("cfg = %+v, want 全部 key 解析", cfg)
	}
	if cfg.Empty() {
		t.Fatal("cfg.Empty() = true, want false（有 key）")
	}
}

// TestLoadFromMapEmpty：无任何 key → Empty() true（main 降级禁用探针）。
func TestLoadFromMapEmpty(t *testing.T) {
	cfg, err := LoadFromMap(map[string]string{SimulatedMarker: "true"})
	if err != nil {
		t.Fatalf("LoadFromMap: %v", err)
	}
	if !cfg.Empty() {
		t.Fatalf("cfg = %+v, want Empty()=true（无 key）", cfg)
	}
}

// TestLoadMissingFile：key 文件不存在 → (zero, false, nil)（业主未提供 → S3 降级，不报错）。
func TestLoadMissingFile(t *testing.T) {
	cfg, ok, err := Load(filepath.Join(t.TempDir(), "no-such-file"))
	if err != nil || ok || !cfg.Empty() {
		t.Fatalf("Load(missing) = %+v ok=%v err=%v, want 零值/ok=false/nil", cfg, ok, err)
	}
}

// TestLoadFromFile：真实文件解析（# 注释、空行、引号值）。
func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "arbcn-sim.env")
	content := `# arbcn sim testnet keys
SIMULATED=true

SIM_BINANCE_API_KEY="bkey"
SIM_BINANCE_SECRET=bsec
SIM_OKX_API_KEY=okey
SIM_OKX_SECRET=osec
SIM_OKX_PASSPHRASE=opass
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, ok, err := Load(path)
	if err != nil || !ok {
		t.Fatalf("Load = ok=%v err=%v, want ok=true/nil", ok, err)
	}
	if cfg.BinanceAPIKey != "bkey" || cfg.BinanceSecret != "bsec" ||
		cfg.OKXAPIKey != "okey" || cfg.OKXSecret != "osec" || cfg.OKXPassphrase != "opass" {
		t.Fatalf("cfg = %+v, want 文件解析", cfg)
	}
}

// TestLoadFileMissingMarker：文件存在但无 SIMULATED → 拒绝（文件级对抗锚点）。
func TestLoadFileMissingMarker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arbcn-sim.env")
	if err := os.WriteFile(path, []byte("SIM_BINANCE_API_KEY=abc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(path); err == nil {
		t.Fatal("Load(file 缺 SIMULATED) = nil, want error")
	}
}

// TestParseEnvFile：解析细节（注释/空行/引号/空格）。
func TestParseEnvFile(t *testing.T) {
	kv := parseEnvFile([]byte("A=1\n# comment\nB = \"two\"\n\nC=three \n"))
	if kv["A"] != "1" || kv["B"] != "two" || kv["C"] != "three" {
		t.Fatalf("kv = %v, want A=1 B=two C=three", kv)
	}
}
