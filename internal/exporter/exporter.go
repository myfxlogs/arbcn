// Package exporter：facts.md 自动导出（M2-b §5 / D-028 闭环）。
// 把监控最新值渲染进 docs/handoff/facts.md 的「监控快照」段：
// 定时（日）+ 关键规则触发事件两种触发；旧快照标「已过期」不删除（D-028 规则）。
// 机器可读投影 = DashboardService.ListFacts（web 前端事实快照视图，§5 挂起项闭合）。
//
// 段所有权：本组件独占 facts.md 中由 begin/endMarker 包裹的自包含区块（P3
// 单一事实源 + 机械可识别）；区块外的手工内容逐字保留。写文件原子（tmp+rename），
// 进程崩溃不留半截文件。
package exporter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// 段标记：facts.md 中本组件独占区块的定界（机械可识别，P4）。
const (
	beginMarker = "<!-- ARBCN-EXPORT-BEGIN -->"
	endMarker   = "<!-- ARBCN-EXPORT-END -->"
)

// 默认导出间隔（日）。
const defaultInterval = 24 * time.Hour

// 快照时间戳格式（facts.md「核实」列同风格）。
const timeLayout = "2006-01-02 15:04"

// skipKinds：不进入快照的内部遥测事实（heartbeat 是采集器自身状态，非市场事实）。
var skipKinds = map[string]bool{fact.KindHeartbeat: true}

// factsSource 是 exporter 依赖的最小存储面（只读最新事实；避免整棵 Store 接口）。
type factsSource interface {
	LatestFacts(ctx context.Context, kind, venue, symbol string) ([]fact.Fact, error)
}

// Exporter 渲染监控最新值到 facts.md 快照段。
// 并发安全：Export 由互斥锁串行（定时与规则触发可能同时到达）。
type Exporter struct {
	st   factsSource
	path string

	// Interval 导出周期（≤0 = 默认 24h）；测试注入。
	Interval time.Duration
	// Now 测试注入时钟；0 = time.Now。
	Now func() time.Time
	// Log 日志；0 = slog.Default()。
	Log *slog.Logger

	trigger chan struct{} // 规则触发信号（容量 1，突发合并）
	mu      sync.Mutex    // 串行化 Export（定时 goroutine 与规则回调并发）
}

// New 构造 exporter。st 需满足 factsSource（store.Store 天然满足）。
func New(st factsSource, path string) *Exporter {
	return &Exporter{st: st, path: path, trigger: make(chan struct{}, 1)}
}

// Export 立即导出一轮：读监控最新值 → 渲染新快照 → 旧「现行」快照标已过期 →
// 原子写回 facts.md。删除标「已过期」的代码 → exporter_test 必红（§11 对抗锚点）。
func (x *Exporter) Export(ctx context.Context) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	now := x.now()
	facts, err := x.st.LatestFacts(ctx, "", "", "")
	if err != nil {
		return fmt.Errorf("exporter: latest facts: %w", err)
	}
	snap := renderSnapshot(facts, now)
	return x.writeSection(snap, now)
}

// Run 导出循环：启动立即导出一轮（boot 刷新），此后每 Interval 一轮 +
// 规则触发信号（OnRuleActive）立即一轮；阻塞至 ctx 取消。单轮失败只 warn
// 不退出（D-032 同口径：监控自身可用性优先，错过一轮换下一轮）。
func (x *Exporter) Run(ctx context.Context) error {
	iv := x.Interval
	if iv <= 0 {
		iv = defaultInterval
	}
	t := time.NewTicker(iv)
	defer t.Stop()
	if err := x.Export(ctx); err != nil {
		x.log().Warn("exporter: boot export failed", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := x.Export(ctx); err != nil {
				x.log().Warn("exporter: scheduled export failed", "err", err)
			}
		case <-x.trigger:
			if err := x.Export(ctx); err != nil {
				x.log().Warn("exporter: rule-trigger export failed", "err", err)
			}
		}
	}
}

// OnRuleActive 关键规则激活事件回调（armed→active 转变；接 rule.Config.OnActive）。
// 非阻塞投递触发信号：突发合并为一次导出，不阻塞规则引擎。
func (x *Exporter) OnRuleActive(context.Context, store.Rule) {
	select {
	case x.trigger <- struct{}{}:
	default: // 已排队，合并本次触发
	}
}

// writeSection 把新快照段写入 facts.md：
//   - 无段标记（首次）→ 文件尾追加新段，既有手工内容保留；
//   - 段标记存在 → 段内第一个「现行」快照改标「已过期（被 <新时刻> 取代）」，
//     旧值逐行保留；新快照置于旧段之前；段外内容（前后手工块）逐字保留。
func (x *Exporter) writeSection(snap string, now time.Time) error {
	content, err := os.ReadFile(x.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("exporter: read %s: %w", x.path, err)
	}
	doc := string(content)

	before, rest, found := strings.Cut(doc, beginMarker)
	if !found {
		doc = appendSection(doc, snap)
	} else if oldSection, after, hasEnd := cutEnd(rest); hasEnd {
		// [对抗测试锚点] 旧「现行」快照 → 已过期（保留旧值不删除）：
		// 删除本行 → TestExportMarksOldExpiredNotDeleted 必红（§11②）。
		oldSection = strings.Replace(oldSection, "（现行）", expiredMark(now), 1)
		doc = before + beginMarker + "\n" + snap + oldSection + endMarker + after
	} else {
		// 段标记残缺（end 丢失）：按首次处理，追加新段。
		doc = appendSection(doc, snap)
	}
	return atomicWrite(x.path, []byte(doc))
}

// appendSection 在文件尾追加完整快照段（保留既有内容；空文件不加空行前缀）。
func appendSection(doc, snap string) string {
	sep := ""
	if strings.TrimSpace(doc) != "" {
		sep = "\n\n"
	}
	return doc + sep + beginMarker + "\n" + sectionHeader() + snap + endMarker + "\n"
}

// cutEnd 从段内截出 endMarker 前的旧段与之后的尾部。
func cutEnd(rest string) (oldSection, after string, ok bool) {
	old, tail, has := strings.Cut(rest, endMarker)
	return old, tail, has
}

// sectionHeader 快照段固定头部（段序号不写死，避免与手工节编号冲突）。
func sectionHeader() string {
	return "## 监控快照（arbcn 自动导出 · M2-b §5 / D-028 闭环）\n\n" +
		"> 机器生成：监控最新值渲染进事实库；新快照到来 → 旧快照标「已过期」不删除。\n" +
		"> 机器可读投影：DashboardService.ListFacts（web 前端事实快照视图）。\n\n"
}

// expiredMark 已过期状态标注（保留旧值；D-028 不删除规则）。
func expiredMark(now time.Time) string {
	return "（已过期 · 被 " + now.Format(timeLayout) + " 快照取代）"
}

// renderSnapshot 渲染单次快照表格（按 kind/symbol/venue 稳定排序；
// heartbeat 内部遥测排除；时间精确到分钟）。
func renderSnapshot(facts []fact.Fact, now time.Time) string {
	rows := make([]string, 0, len(facts))
	for _, f := range facts {
		if skipKinds[f.Kind] {
			continue
		}
		rows = append(rows, fmt.Sprintf("| %s %s@%s | %s | %s | %s | %s |",
			f.Kind, f.Symbol, f.Venue, formatValue(f.Value), f.Unit, f.Ts.Format(timeLayout), f.Src))
	}
	sort.Strings(rows)
	var b strings.Builder
	b.WriteString("### 快照 " + now.Format(timeLayout) + "（现行）\n\n")
	b.WriteString("| 事实 | 值 | 单位 | 采集时刻 | 来源 |\n")
	b.WriteString("|------|-----|------|---------|------|\n")
	for _, r := range rows {
		b.WriteString(r)
		b.WriteByte('\n')
	}
	return b.String()
}

// formatValue 值格式（%4g，与规则告警消息同风格：6.84 / 7.03 / 0.5）。
func formatValue(v float64) string {
	return fmt.Sprintf("%.4g", v)
}

// now 返回注入时钟或 time.Now。
func (x *Exporter) now() time.Time {
	if x.Now != nil {
		return x.Now()
	}
	return time.Now()
}

// log 返回注入 logger 或 slog.Default()。
func (x *Exporter) log() *slog.Logger {
	if x.Log != nil {
		return x.Log
	}
	return slog.Default()
}

// atomicWrite 临时文件 + rename 原子落盘（进程崩溃不留半截文件）。
// 父目录不存在则创建（exporter 自举：facts.md 所在目录缺失也能写）。
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("exporter: mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".facts-*.tmp")
	if err != nil {
		return fmt.Errorf("exporter: create temp: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("exporter: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("exporter: close temp: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("exporter: rename: %w", err)
	}
	return nil
}
