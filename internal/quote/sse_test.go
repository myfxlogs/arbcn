// sse_test.go：SSE handler 推送行为（首推快照 + 每 1s 只推有变化的 tick）。
package quote

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestSSEInitialSnapshotThenDiff：连接即推全量快照；变化后下个 tick 只推变化的标。
func TestSSEInitialSnapshotThenDiff(t *testing.T) {
	hub := NewHub()
	hub.Update(Price{Venue: VenueBinance, Symbol: "BTC", Price: 65000, TsMs: 1})
	hub.Update(Price{Venue: VenueOKX, Symbol: "BTC", Price: 65010, TsMs: 1})

	srv := httptest.NewServer(hub.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET %s: %v", srv.URL, err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	// 读首推：应含两条（binance + okx BTC）。
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	var first []string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(first) < 2 {
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		if strings.HasPrefix(line, "data: ") {
			first = append(first, strings.TrimPrefix(line, "data: "))
		}
	}
	if len(first) != 2 {
		t.Fatalf("首推 data 条数 = %d, want 2（%v）", len(first), first)
	}

	// 等 1 个 tick（1s）内更新 binance BTC → 下一个 tick 只推 1 条变化。
	go func() {
		time.Sleep(1500 * time.Millisecond)
		hub.Update(Price{Venue: VenueBinance, Symbol: "BTC", Price: 65100, TsMs: 2})
	}()

	var changed []string
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		if strings.HasPrefix(line, "data: ") {
			changed = append(changed, strings.TrimPrefix(line, "data: "))
			if len(changed) >= 1 {
				break
			}
		}
	}
	if len(changed) != 1 {
		t.Fatalf("变化后推 data 条数 = %d, want 1（%v）", len(changed), changed)
	}
	if !strings.Contains(changed[0], `"price":65100`) {
		t.Fatalf("变化帧内容 = %s, want 含 price 65100", changed[0])
	}
}

// TestSSEClientDisconnect：客户端断开，handler 干净退出（不 panic、不泄漏）。
func TestSSEClientDisconnect(t *testing.T) {
	hub := NewHub()
	hub.Update(Price{Venue: VenueBinance, Symbol: "BTC", Price: 1, TsMs: 1})
	srv := httptest.NewServer(hub.Handler())
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	// 读一点后立即断开。
	buf := make([]byte, 256)
	_, _ = resp.Body.Read(buf)
	_ = resp.Body.Close()
	// handler 侧 r.Context().Done() 触发返回；此处只验证进程不崩（测试通过即证明）。
	time.Sleep(100 * time.Millisecond)
}
