//go:build live

// 真机连通性冒烟（不随常规 go test 运行）：
//
//	go test -tags=live -run TestLivePoll -v ./internal/collect/defirate/
//
// 中国大陆直连 DefiLlama 可能被墙；失败时错误即证据（含端点 URL），
// 由决策层裁决换源/代理/降级，施工方不自行决策。
package defirate

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLivePoll：对真实 DefiLlama /pools 拉取一轮，事实逐条打日志。
func TestLivePoll(t *testing.T) {
	cfg := FromEnv(os.Getenv)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
