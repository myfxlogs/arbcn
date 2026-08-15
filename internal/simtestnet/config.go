// Package simtestnet：testnet 只读探针 + key 承载层（04-m3-spec §9.4 S3，D-034 ② 物理隔离）。
//
// 与 internal/sim（零网络零密钥）物理隔离：本包是唯一触碰网络/密钥的地方。职责：
//   - 加载 /etc/arbcn/arbcn-sim.env 的 SIM_* 配置；每 key 必须显式 SIMULATED=true，
//     缺标记拒绝加载（对抗测试锚点）——密钥不会在无"这是模拟盘"声明时被接受。
//   - 只读探针：binance_testnet / okx_demo 公共行情 + 账户只读查询，验证 key 连通；
//     成功经 alert.Heartbeat.Record("sim_testnet_binance"/"sim_testnet_okx") 登记。
//   - 零下单路径：本包只有只读端点，不含任何下单（订单委托）端点代码（domains_test 把关）。
//
// 依赖：key 由业主提供；缺失 → 降级禁用（不阻塞 S1/S2/S4/S5）。
package simtestnet

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// SimulatedMarker 是 key 文件显式声明"这是模拟盘密钥"的标记（缺它拒绝加载）。
const SimulatedMarker = "SIMULATED"

// DefaultKeyPath 是 sim key 文件的默认路径（root:root 0600，独立于主配置）。
const DefaultKeyPath = "/etc/arbcn/arbcn-sim.env"

// Config 是 testnet 只读探针的 key 配置。
type Config struct {
	BinanceAPIKey string
	BinanceSecret string
	OKXAPIKey     string
	OKXSecret     string
	OKXPassphrase string
}

// Empty 是否无任何 key（无 binance 也无 okx）→ 探针无可执行，main 降级禁用。
func (c Config) Empty() bool {
	return c.BinanceAPIKey == "" && c.OKXAPIKey == ""
}

// Load 从 path 读取 SIM_* key 文件（默认 /etc/arbcn/arbcn-sim.env）。
// 文件不存在 → (zero, false, nil)：业主未提供 key，S3 降级禁用（不阻塞其他子任务）。
// 文件存在但 SIMULATED 标记缺失/非 true → 错误（缺标记拒绝加载，对抗测试锚点）。
func Load(path string) (Config, bool, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, fmt.Errorf("simtestnet: read key file %s: %w", path, err)
	}
	cfg, err := LoadFromMap(parseEnvFile(body))
	if err != nil {
		return Config{}, false, err
	}
	return cfg, true, nil
}

// LoadFromMap 从 k=v map 加载（测试注入）。要求显式 SIMULATED=true。
//
// [对抗测试锚点] §9.4 S3：删 SIMULATED 校验 → config_test.go
// TestLoadMissingSimulatedMarker（缺标记拒绝加载断言）必红。
func LoadFromMap(v map[string]string) (Config, error) {
	if strings.TrimSpace(v[SimulatedMarker]) != "true" {
		return Config{}, fmt.Errorf("simtestnet: %s 标记缺失/非 true（每 key 必须显式 SIMULATED=true，D-034 ②）",
			SimulatedMarker)
	}
	cfg := Config{
		BinanceAPIKey: strings.TrimSpace(v["SIM_BINANCE_API_KEY"]),
		BinanceSecret: strings.TrimSpace(v["SIM_BINANCE_SECRET"]),
		OKXAPIKey:     strings.TrimSpace(v["SIM_OKX_API_KEY"]),
		OKXSecret:     strings.TrimSpace(v["SIM_OKX_SECRET"]),
		OKXPassphrase: strings.TrimSpace(v["SIM_OKX_PASSPHRASE"]),
	}
	return cfg, nil
}

// parseEnvFile 解析 k=v 行式 key 文件（# 注释、空行跳过；value 去引号）。
func parseEnvFile(body []byte) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(string(body)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		if k != "" {
			out[k] = v
		}
	}
	return out
}
