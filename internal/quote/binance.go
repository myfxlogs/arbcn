// binance.go：Binance USDT-M 永续公共行情 WebSocket feed（D-056 Part B 上游）。
// 端点：wss://fstream.binancefuture.com/stream?streams=btcusdt@miniTicker/...
// 公共只读行情，无密钥（D-010 合规）。多标的单连接合流（&streams=a/b/c），
// 比逐标的开连接省资源。合流端点每帧外层包 {stream, data}，miniTicker 本体在
// data 内（c = 最新价、E = 事件毫秒、s = 大写 BASEUSDT）。部署机实测（2026-08-16）。
package quote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// binanceStreamBase 是 Binance USDT-M futures 公共合流 WS 端点。
// D-056 部署机实测（2026-08-16）：fstream.binance.com 在此机连接 i/o timeout（geo-block），
// fstream.binancefuture.com 直连通（同数据、同帧形）——与 REST 侧 fapi→fapi.binancefuture
// 同源（practices #12：端点必须部署机实测）。保留 .com 注释防误改回。
const binanceStreamBase = "wss://fstream.binancefuture.com/stream"

// binanceFeed 订阅 binance USDT-M 永续合流行情；symbols 为基础币种（BTC/ETH/TRX…）。
type binanceFeed struct {
	symbols []string
	base    string // 测试注入；空 = 默认
}

func newBinanceFeed(symbols []string) *binanceFeed {
	return &binanceFeed{symbols: symbols}
}

// streamURL 组装合流 URL：streams=btcusdt@miniTicker/ethusdt@miniTicker/...
func (f *binanceFeed) streamURL() string {
	if f.base != "" {
		return f.base
	}
	parts := make([]string, 0, len(f.symbols))
	for _, s := range f.symbols {
		parts = append(parts, strings.ToLower(s)+"usdt@miniTicker")
	}
	// Binance 合流格式固定为 ?streams=a@miniTicker/b@miniTicker（@ 与 / 按原样，不转义；
	// QueryEscape 会把它们编码成 %40/%2F 导致服务器不识别）。
	return binanceStreamBase + "?streams=" + strings.Join(parts, "/")
}

// binanceMiniTicker 是 miniTicker 帧最小面（c = 最新价；E = 事件毫秒；s = 符号如 BTCUSDT）。
// EventType 字段不能省——Go json 按字段名大小写不敏感匹配，帧里同时有 "e"（事件类型字符串）
// 和 "E"（毫秒时间戳数值）。若不显式声明 "e"，"e" 会大小写不敏感地撞上 EventT(int64)，
// 导致整帧 Unmarshal 失败被丢弃（部署机实测 D-056 Part B 教训）。
type binanceMiniTicker struct {
	EventType string `json:"e"`
	Symbol    string `json:"s"`
	Close     string `json:"c"`
	EventT    int64  `json:"E"`
}

// binanceStreamFrame 是合流端点外层帧：每条流的 miniTicker 包在 data 里。
// 单流 /ws/ 端点是扁平帧，但本 feed 只用合流（一连接多标的），故只解 envelope。
type binanceStreamFrame struct {
	Stream string            `json:"stream"`
	Data   binanceMiniTicker `json:"data"`
}

// connectOnce 拨号 → 读循环 → 每帧解析入库。返回 error 由 retryLoop 退避重连。
func (f *binanceFeed) connectOnce(ctx context.Context, hub *Hub) error {
	conn, err := wsDial(ctx, f.streamURL())
	if err != nil {
		return fmt.Errorf("binance ws dial: %w", err)
	}
	defer conn.Close()
	for {
		conn.SetReadDeadline(time.Now().Add(wsReadIdle)) // 静默连接 45s 超时 → 重连
		_, body, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var env binanceStreamFrame
		if err := json.Unmarshal(body, &env); err != nil {
			continue // 非 miniTicker 帧（如订阅确认/错误帧）跳过，不中断流
		}
		t := env.Data
		if t.Symbol == "" || t.Close == "" {
			continue
		}
		price, err := strconv.ParseFloat(t.Close, 64)
		if err != nil {
			continue
		}
		hub.Update(Price{
			Venue:  VenueBinance,
			Symbol: strings.TrimSuffix(t.Symbol, "USDT"),
			Price:  price,
			TsMs:   t.EventT,
		})
	}
}

// Run 阻塞运行 binance feed（retryLoop 断线自愈重连）。仅 ctx 取消返回。
func (f *binanceFeed) Run(ctx context.Context, log *slog.Logger, hub *Hub) error {
	return retryLoop(ctx, log, VenueBinance, func(ctx context.Context) error {
		return f.connectOnce(ctx, hub)
	})
}

// RunBinance 以默认符号源运行 binance feed（main 装配用）。
func RunBinance(ctx context.Context, log *slog.Logger, hub *Hub, symbols []string) error {
	return newBinanceFeed(symbols).Run(ctx, log, hub)
}
