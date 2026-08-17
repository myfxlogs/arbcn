package main

import (
	"context"
	"log/slog"
	"time"
)

// simSeeder 收窄到本文件所需的最小接口（P3：只依赖 InitSimAccount 一个事实）。
type simSeeder interface {
	InitSimAccount(ctx context.Context, capital float64) error
}

// seedSimAccountRetry（D-068）：开机 PG 竞态修复。
//
// 事故（2026-08-17 08:45）：机器重启后 arbcn 先于 docker PostgreSQL（127.0.0.1:5434）
// 就绪被 systemd 拉起，InitSimAccount 连接拒绝 -> 原 fail-fast 直接退出 -> 退化为
// systemd 反复重启循环，反代 502 窗口 ~2 分钟。main.go 的 PG 不可达路径本就设计为
// 「Warn + 管线退避重连」（dialogue #22），本函数把同一口径补齐到 boot seed：
// InitSimAccount 幂等（D-056：重启不重置账本），失败按固定间隔重试，耗尽仍失败
// 才交给 systemd Restart=on-failure（外层兜底不变）。
//
// 参数注入 sleep 供测试（对抗锚点：删重试 / 删有界 -> 测试必红）。
func seedSimAccountRetry(ctx context.Context, st simSeeder, capital float64, attempts int, wait time.Duration, sleep func(time.Duration) bool) error {
	var err error
	for i := 1; i <= attempts; i++ {
		if err = st.InitSimAccount(ctx, capital); err == nil {
			return nil
		}
		if i == attempts {
			break
		}
		slog.Warn("sim account seed failed, pg may still be booting", "attempt", i, "retry_in", wait.String(), "err", err)
		if !sleep(wait) {
			return ctx.Err()
		}
	}
	return err
}

// sleepCtx 返回绑定 ctx 的生产 sleep：可被取消打断（优雅停机不被重试窗口卡住）。
func sleepCtx(ctx context.Context) func(time.Duration) bool {
	return func(d time.Duration) bool {
		select {
		case <-time.After(d):
			return true
		case <-ctx.Done():
			return false
		}
	}
}
