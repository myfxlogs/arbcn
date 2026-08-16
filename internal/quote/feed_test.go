// feed_test.go：WS feed 用本地 fake 服务器验证（httptest + gorilla upgrader）。
// 只测本包逻辑（拨号 → 订阅 → 解析入库 → ping/pong），不触真实交易所。
package quote

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeWS 本地 WS 服务器：按脚本依次回帧，记录收到的文本（验证订阅/保活）。
type fakeWS struct {
	srv       *httptest.Server
	url       string
	received  chan string // 客户端发来的文本帧（subscribe / pong）
	frames    chan []byte // 服务器下发的帧
}

func newFakeWS(t *testing.T, script [][]byte) *fakeWS {
	t.Helper()
	up := websocket.Upgrader{}
	f := &fakeWS{received: make(chan string, 16), frames: make(chan []byte, 16)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// 先发脚本帧。
		for _, b := range script {
			if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		}
		// 再循环读客户端帧（订阅/保活）+ 可再发帧。
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			_, body, err := conn.ReadMessage()
			if err != nil {
				return
			}
			f.received <- string(body)
		}
	}))
	t.Cleanup(srv.Close)
	f.srv = srv
	f.url = "ws" + strings.TrimPrefix(srv.URL, "http")
	return f
}

// binanceMiniTickerFrame 构造 binance 合流外层帧（真实帧形：{stream, data:{e,s,c,E}}）。
// [对抗测试锚点 D-056 Part B] 用真实 envelope + 大小写 e/E 并存形——若实现删 data 包裹
// 或漏声明 "e" 字段（Go json 大小写不敏感匹配 → "e" 撞上 EventT → 整帧丢弃），此测试必红。
func binanceMiniTickerFrame(symbol, close string, e int64) []byte {
	b, _ := json.Marshal(binanceStreamFrame{
		Stream: symbol + "@miniTicker",
		Data: binanceMiniTicker{
			EventType: "24hrMiniTicker",
			Symbol:    symbol,
			Close:     close,
			EventT:    e,
		},
	})
	return b
}

// TestBinanceFeedParsesFrame：binance feed 把 miniTicker 帧解析入库（TrimSuffix USDT）。
func TestBinanceFeedParsesFrame(t *testing.T) {
	hub := NewHub()
	f := newFakeWS(t, [][]byte{binanceMiniTickerFrame("BTCUSDT", "65000.5", 1234)})
	bf := newBinanceFeed([]string{"BTC"})
	bf.base = f.url

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- bf.Run(ctx, slog.Default(), hub) }()

	// 等 hub 收到。
	waitFor(t, func() bool { return len(hub.Snapshot()) == 1 })
	cancel()
	if got := hub.Snapshot()[0]; got.Symbol != "BTC" || got.Price != 65000.5 || got.TsMs != 1234 {
		t.Fatalf("binance 解析入库错误：%+v", got)
	}
}

// TestOKXFeedParsesFrame + ping 保活：tickers 帧入库，ping 收到回 pong。
func TestOKXFeedParsesFrame(t *testing.T) {
	hub := NewHub()
	// 脚本：先 ping（保活），再 tickers 帧。
	tk := `{"data":[{"instId":"ETH-USDT-SWAP","last":"3500.25","ts":"999"}]}`
	f := newFakeWS(t, [][]byte{[]byte("ping"), []byte(tk)})
	of := newOKXFeed([]string{"ETH"})
	of.base = f.url

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- of.Run(ctx, slog.Default(), hub) }()

	waitFor(t, func() bool { return len(hub.Snapshot()) == 1 })
	cancel()
	if got := hub.Snapshot()[0]; got.Symbol != "ETH" || got.Price != 3500.25 || got.TsMs != 999 {
		t.Fatalf("okx 解析入库错误：%+v", got)
	}
	// 收到的客户端帧应含 pong（ping 保活应答）+ subscribe（op 帧）。
	ok := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case msg := <-f.received:
			if strings.Contains(msg, "pong") {
				ok = true
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	if !ok {
		t.Fatal("okx 收到 ping 未回 pong")
	}
}

// TestRetryLoopReconnect：断线后 retryLoop 自动重连（feed 重启拨号成功，hub 持续入库）。
func TestRetryLoopReconnect(t *testing.T) {
	hub := NewHub()
	f := newFakeWS(t, [][]byte{binanceMiniTickerFrame("BTCUSDT", "1", 1)})
	bf := newBinanceFeed([]string{"BTC"})
	bf.base = f.url

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- bf.Run(ctx, slog.Default(), hub) }()

	waitFor(t, func() bool { return len(hub.Snapshot()) == 1 })
	// 不取消 ctx，feed 应保持运行（connectOnce 内部 read 循环阻塞在 fake 服务器连接上）。
	select {
	case err := <-done:
		t.Fatalf("feed 在连接存活时退出：%v", err)
	case <-time.After(300 * time.Millisecond):
		// 正常：连接存活，feed 阻塞运行。
	}
	cancel()
}

// waitFor 轮询等待条件成立。
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}
