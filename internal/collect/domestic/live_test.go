//go:build live

// 真机连通性冒烟（不随常规 go test 运行）：
//
//	go test -tags=live -run TestLivePoll -v ./internal/collect/domestic/
//
// 新浪 hq 国内直连通常可用；BOC 官网爬取可能被反爬拦截——bank_rate 失败属
// §5 允许降级路径（人工录入通道补位），不阻塞 repo 源。错误即证据（含端点 URL）。
package domestic

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLivePoll：对真实新浪行情（逆回购）与 BOC 官网（挂牌利率）各拉取一轮。
func TestLivePoll(t *testing.T) {
	cfg := FromEnv(os.Getenv)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, n := range All(cfg) {
		fs, err := n.Collector.Poll(ctx)
		if err != nil {
			t.Errorf("%s: %v", n.Name, err)
			continue
		}
		t.Logf("%s: %d facts", n.Name, len(fs))
		for _, f := range fs {
			t.Logf("  kind=%s venue=%s symbol=%s value=%v unit=%s ts=%s src=%s",
				f.Kind, f.Venue, f.Symbol, f.Value, f.Unit, f.Ts.UTC().Format(time.RFC3339), f.Src)
		}
	}
}
