// feed.go：WS feed 公共骨架（重连退避）。binance.go / okx.go 各自实现
// connectOnce（拨号 + 订阅 + 读循环），共用 retryLoop 做断线指数退避重连。
package quote

import (
	"context"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

// Venue 常量（Price.Venue 值域，与 collect/exchange 同语义）。
const (
	VenueBinance = "binance"
	VenueOKX     = "okx"
)

// 连接健康参数（部署机实测 2026-08-16）：
// wsDialTimeout 限拨号上限——交易所 WS 拨号可能停滞（DNS/TLS/代理），无上限会卡死首连
// （gorilla 默认 HandshakeTimeout 45s，TCP connect 停滞则更久）。8s 内没连上 = 交给重试。
// wsReadIdle 读空闲上限——静默连接视为死连接 → 断线重连。正常行情 1s 一帧（binance miniTicker）
// / OKX 保活 ping 约 30s 一次，45s 在两者之上不误伤，只是死连接的兜底探测。
const (
	wsDialTimeout = 8 * time.Second
	wsReadIdle    = 45 * time.Second
)

// wsDial 拨号（8s 上限）。停滞拨号 8s 即返回错误，由 retryLoop 退避重连，不再卡死首连。
func wsDial(ctx context.Context, url string) (*websocket.Conn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, wsDialTimeout)
	defer cancel()
	conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, url, nil)
	return conn, err
}

// retryLoop 阻塞运行 connect 直至 ctx 取消；失败指数退避（1s → 2s → … 上限 30s）重连。
// 不返回错误——断线自愈重连是职责（SSE 继续服务旧快照，D-032 同口径：失败 warn 不退出）。
// 重连间隔在阈值内静默（slog.Debug），超过 10s 才 Warn 一次（持续断线要给运维可见信号，
// 但避免正常网络抖动刷屏）。
func retryLoop(ctx context.Context, log *slog.Logger, name string, connect func(ctx context.Context) error) error {
	backoff := 1 * time.Second
	warned := false
	for {
		if err := connect(ctx); err != nil {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			if backoff > 10*time.Second && !warned {
				if log != nil {
					log.Warn("quote feed 持续断线，退避重连中", "feed", name, "err", err, "backoff", backoff)
				}
				warned = true
			} else if log != nil {
				log.Debug("quote feed 断线重连", "feed", name, "err", err, "backoff", backoff)
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			continue
		}
		// connect 正常返回 = 连接被远端正常关闭（读循环干净退出）→ 立即重连（不放大退避）。
		backoff = 1 * time.Second
		warned = false
	}
}
