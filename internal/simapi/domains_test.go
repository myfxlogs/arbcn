package simapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoRealTradeTokens：simapi 独立域（arbcn.sim.v1，D-038 ①）零真实账户/下单路径
// （04-m3-spec §10.6 C5 grep 断言，编码为测试）。模拟执行确认后仍是模拟（SIMULATED），
// 不接真实资金（§6/§8，不赌原则 D-019）——任何真实账户/提现/转账/下单端点字符串出现在
// handler 源码 = 潜在真金路径，必红。
// 豁免 *_test.go（断言字符串本身出现在本测试）；gen/ 为生成代码（由 proto 全文生成，
// 无真实端点；check-lines 已豁免）。
func TestNoRealTradeTokens(t *testing.T) {
	// 连接符拼装，避免字面量污染本包源码面（与 internal/sim/domains_test.go 同手法）。
	tokens := []string{
		// 真实交易主网域（§7：只读公开 API 例外，但绝不接交易端点）。
		"fapi." + "binance.com",
		"www." + "okx.com",
		"aws." + "okx.com",
		// 下单端点路径（§10.6：无 order 下单路径）。
		"/fapi/v1/order",
		"/api/v5/trade/order",
		"placeOrder", "newOrder", "cancelOrder",
		// 账户/资金端点（§10.6：无 account / withdraw / transfer）。
		"account", "withdraw", "transfer",
	}
	for _, f := range simapiSourceFiles(t) {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, tok := range tokens {
			if strings.Contains(string(body), tok) {
				t.Errorf("%s 含真实交易端点片段 %q（§10.6：simapi 零真实账户/下单路径）", f, tok)
			}
		}
	}
}

// TestNoAutoConfirmTimer：ConfirmSimOrder 是唯一写路径——包内无定时器/自动确认
// （§10.6 C5：grep 无 time.Ticker）。模拟盘确认永远人工（资金动作永远人工，§1），
// 禁止自动确认定时器把建议订单自动成交。
func TestNoAutoConfirmTimer(t *testing.T) {
	banned := []string{"time.Ticker", "time.NewTicker"}
	for _, f := range simapiSourceFiles(t) {
		body, err := os.ReadFile(filepath.Join(".", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, b := range banned {
			if strings.Contains(string(body), b) {
				t.Errorf("%s 含 %q（§10.6：禁止自动确认定时器，ConfirmSimOrder 唯一写路径）", f, b)
			}
		}
	}
}

// TestSimExecBadgeRenderable：前端模拟执行徽标（SIMULATED / 「模拟」）可 grep
// （§10.5/§10.6 C5 可检查锚点，对话 #59 重构后徽标定义集中到共享 sim.tsx）。
//   - sim.tsx：SIMULATED / 「模拟」字面量定义处（删徽标 → 必红）。
//   - SimExec.tsx（模拟执行 tab）、ConfirmPanel.tsx（总览页确认下单面板）：
//     必须引用 SimulatedBadge（面板顶部徽标渲染；防抽出确认面板时漏挂徽标）。
// 永不出现真金按钮/路径。
func TestSimExecBadgeRenderable(t *testing.T) {
	base := filepath.Join("..", "..", "web", "src", "components")
	def := filepath.Join(base, "sim.tsx")
	body, err := os.ReadFile(def)
	if err != nil {
		t.Fatalf("read %s: %v（web 源缺失 = 交付不完整）", def, err)
	}
	for _, tok := range []string{"SIMULATED", "模拟"} {
		if !strings.Contains(string(body), tok) {
			t.Errorf("%s 缺固定 %q 渲染（§10.5 验收锚点：SIMULATED 徽标可 grep）", def, tok)
		}
	}
	for _, f := range []string{"SimExec.tsx", "ConfirmPanel.tsx"} {
		p := filepath.Join(base, f)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v（web 源缺失 = 交付不完整）", p, err)
		}
		if !strings.Contains(string(b), "SimulatedBadge") {
			t.Errorf("%s 缺 SimulatedBadge 引用（模拟执行面板顶部须渲染 SIMULATED 徽标）", p)
		}
	}
}

// simapiSourceFiles 返回包内非 _test.go 的 .go 源文件（不含 gen/ 子目录生成物）。
func simapiSourceFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read pkg dir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			files = append(files, e.Name())
		}
	}
	return files
}
