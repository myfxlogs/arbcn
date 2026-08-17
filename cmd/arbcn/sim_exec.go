// 模拟执行装配（sim 配置 + testnet 探针/镜像执行器接线，D-098 测试网执行层）。
// 与 main.go 同包（main），抽出避免 run() 与 main.go 行数超限（check-lines ≤450 硬线）；
// 组装语义不变：sim 配置缺失 → 降级禁用（§7/D-032），testnet key 缺失 → 探针/镜像关
// （S3 降级不阻塞 S1/S2/S4/S5）。simnetOK 用于探针启停，镜像以 NewExecutor 实际降级判定。
package main

import (
	"log/slog"
	"os"

	"arbcn/internal/dashboard"
	"arbcn/internal/fact"
	"arbcn/internal/sim"
	"arbcn/internal/simapi"
	"arbcn/internal/simtestnet"
)

// loadSimConfig 从 ARBCN_SIM_* 加载 sim 配置；非法 → warn + 禁用（§7/D-032 降级不退出）。
func loadSimConfig() (sim.Config, bool) {
	cfg, err := sim.FromEnv(os.Getenv)
	if err != nil {
		slog.Warn("sim config invalid, sim driver disabled", "err", err)
		return sim.Config{}, false
	}
	// D-098 venue 路由校验：ARBCN_SIM_EXEC_VENUE 非空必须 ∈ {okx_demo, binance_testnet}
	//（值域定义在 simtestnet）。否则 warn + 清空（镜像禁用；宁缺毋滥，防拼错静默错配——
	// 校验放 main 而非 sim 包：sim 零网络零密钥不 import simtestnet）。
	if cfg.ExecVenue != "" && cfg.ExecVenue != simtestnet.VenueOKXDemo && cfg.ExecVenue != simtestnet.VenueBinanceTestnet {
		slog.Warn("ARBCN_SIM_EXEC_VENUE invalid, mirror disabled", "venue", cfg.ExecVenue)
		cfg.ExecVenue = ""
	}
	return cfg, true
}

// simMirrorExecutor 构造镜像执行器（D-098）：testnet key 就绪（NewExecutor 非降级）且
// ExecVenue 非空 → 挂 *simtestnet.Executor（ConfirmSimOrder 本地成交前逐腿镜像下单，
// best-effort）；否则 nil = 镜像关（M3-c 零回归）。缺 key / venue 空 → nil。
func simMirrorExecutor(simnetCfg simtestnet.Config, simCfg sim.Config) simapi.Executor {
	exec, ok := simtestnet.NewExecutor(simnetCfg)
	if !ok || simCfg.ExecVenue == "" {
		return nil
	}
	slog.Info("sim mirror executor enabled", "venue", simCfg.ExecVenue)
	return exec
}

// loadSimtestnetConfig 加载 testnet key 文件（/etc/arbcn/arbcn-sim.env）。
// 文件缺失 → 降级禁用（业主未提供 key，S3 不阻塞 S1/S2/S4/S5）；SIMULATED 标记缺失 → warn 禁用。
func loadSimtestnetConfig() (simtestnet.Config, bool) {
	cfg, ok, err := simtestnet.Load(simtestnet.DefaultKeyPath)
	if err != nil {
		slog.Warn("simtestnet key config invalid, testnet probe disabled", "err", err)
		return simtestnet.Config{}, false
	}
	if !ok {
		slog.Warn("simtestnet key file not found, testnet probe disabled (S3 degrade, 不阻塞 S1/S2/S4/S5)")
		return simtestnet.Config{}, false
	}
	return cfg, true
}

// probeEnabled testnet 探针是否启用：sim 驱动可用（settle tick 承载）且 testnet key 非空。
func probeEnabled(simOK bool, cfg simtestnet.Config) bool {
	return simOK && !cfg.Empty()
}

// probeSourceInfos 把 sim_testnet 探针源并入仪表盘源健康面（ListSourceHealth 数据面）。
func probeSourceInfos() []dashboard.SourceInfo {
	return []dashboard.SourceInfo{
		{Name: simtestnet.SourceBinanceTestnet, IntervalSec: int64(settleInterval.Seconds()), Kind: fact.KindHeartbeat},
		{Name: simtestnet.SourceOKXDemo, IntervalSec: int64(settleInterval.Seconds()), Kind: fact.KindHeartbeat},
	}
}
