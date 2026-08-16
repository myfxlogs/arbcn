// okx.go：OKX v5 公共行情 WebSocket feed（D-056 Part B 上游）。
// 端点：wss://ws.okx.com:8443/ws/v5/public。公共只读行情，无密钥（D-010 合规）。
// 流程：拨号 → 发 subscribe（tickers 频道）→ 读循环。tickers 帧 data[].last = 最新价、
// ts = 事件毫秒、instId = 大写 BASE-USDT-SWAP。订阅确认帧（event:subscribe）data 空，
// 跳过不中断流。
package quote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// okxStreamBase 是 OKX v5 公共行情 WS 端点。
const okxStreamBase = "wss://ws.okx.com:8443/ws/v5/public"

// okxFeed 订阅 OKX v5 tickers 频道（USDT-SWAP 永续）；symbols 为基础币种（BTC/ETH/TRX…）。
type okxFeed struct {
	symbols []string
	base    string // 测试注入；空 = 默认
}

func newOKXFeed(symbols []string) *okxFeed {
	return &okxFeed{symbols: symbols}
}

// instID 把基础币种映射为 OKX instId（如 BTC → BTC-USDT-SWAP）。
func instID(symbol string) string {
	return strings.ToUpper(symbol) + "-USDT-SWAP"
}

// subscribeMsg 构造 subscribe 帧（一次性订阅全部 symbol）。
func (f *okxFeed) subscribeMsg() []byte {
	args := make([]map[string]string, 0, len(f.symbols))
	for _, s := range f.symbols {
		args = append(args, map[string]string{"channel": "tickers", "instId": instID(s)})
	}
	b, _ := json.Marshal(map[string]any{"op": "subscribe", "args": args})
	return b
}

// okxTicker 是 tickers 帧最小面。
// data[].last = 最新价（字符串）；ts = 事件毫秒（字符串）；instId = BASE-USDT-SWAP。
type okxTicker struct {
	Data []struct {
		InstID string `json:"instId"`
		Last   string `json:"last"`
		Ts     string `json:"ts"`
	} `json:"data"`
}

// connectOnce 拨号 → 订阅 → 读循环 → 每帧解析入库。返回 error 由 retryLoop 退避重连。
func (f *okxFeed) connectOnce(ctx context.Context, hub *Hub) error {
	base := f.base
	if base == "" {
		base = okxStreamBase
	}
	conn, err := wsDial(ctx, base)
	if err != nil {
		return fmt.Errorf("okx ws dial: %w", err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, f.subscribeMsg()); err != nil {
		return fmt.Errorf("okx ws subscribe: %w", err)
	}
	for {
		conn.SetReadDeadline(time.Now().Add(wsReadIdle)) // 静默连接 45s 超时 → 重连
		_, body, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		// OKX 保活：服务器发文本 "ping"（非 WS 控制帧），客户端须回 "pong"，否则断连。
		if string(body) == "ping" {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
			continue
		}
		var t okxTicker
		if err := json.Unmarshal(body, &t); err != nil {
			continue // 非 tickers 帧（订阅确认 event:subscribe / 心跳）跳过
		}
		if len(t.Data) == 0 {
			continue
		}
		d := t.Data[0]
		if d.InstID == "" || d.Last == "" {
			continue
		}
		price, err := strconv.ParseFloat(d.Last, 64)
		if err != nil {
			continue
		}
		var ts int64
		if v, err := strconv.ParseInt(d.Ts, 10, 64); err == nil {
			ts = v
		} else {
			ts = time.Now().UnixMilli()
		}
		hub.Update(Price{
			Venue:  VenueOKX,
			Symbol: strings.TrimSuffix(d.InstID, "-USDT-SWAP"),
			Price:  price,
			TsMs:   ts,
		})
	}
}

// Run 阻塞运行 okx feed（retryLoop 断线自愈重连）。仅 ctx 取消返回。
func (f *okxFeed) Run(ctx context.Context, log *slog.Logger, hub *Hub) error {
	return retryLoop(ctx, log, VenueOKX, func(ctx context.Context) error {
		return f.connectOnce(ctx, hub)
	})
}

// RunOKX 以默认符号源运行 okx feed（main 装配用）。
func RunOKX(ctx context.Context, log *slog.Logger, hub *Hub, symbols []string) error {
	return newOKXFeed(symbols).Run(ctx, log, hub)
}
