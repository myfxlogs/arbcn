// Package calendar：日历事件采集（docs/design/02-monitor-architecture.md §5 Calendar 行）。
// ① 规则计算：月末 / 季末 / 周四（逆回购时点约定，§7 规则"事件前 1 天 + 当日 10:30"的输入）。
// ② 人工维护表：国债发行 / Launchpool 等（JSON 文件，ARBCN_CALENDAR_FILE；未配置 = 无人工事件）。
// 纯本地计算，无网络；日期口径 CST（UTC+8，无夏令时）。Fact.Value = 距事件的天数（当日 = 0）。
package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"arbcn/internal/collect"
	"arbcn/internal/fact"
)

// VenueRule / VenueManual 区分规则计算与人工维护表两条来源。
const (
	VenueRule   = "rule"
	VenueManual = "manual"
)

// Config 是 calendar 采集的最小配置面。
type Config struct {
	Now  func() time.Time // 时钟注入（测试）；nil = time.Now
	File string           // 人工事件表 JSON；空 = 无人工事件
}

// FromEnv 从环境变量构建 Config（ARBCN_CALENDAR_FILE）。
func FromEnv(getenv func(string) string) Config {
	return Config{File: strings.TrimSpace(getenv("ARBCN_CALENDAR_FILE"))}
}

// All 返回命名 collector：calendar，默认间隔 24h（§5 日频）。
func All(cfg Config) []collect.Named {
	return []collect.Named{
		{Name: "calendar", Interval: 24 * time.Hour, Collector: NewCalendar(cfg)},
	}
}

// event 是人工事件表的条目。
type event struct {
	Date time.Time
	Name string
}

// Calendar 产出规则事件 + 人工事件（Kind=calendar）。
type Calendar struct{ cfg Config }

// NewCalendar 构造日历 collector。
func NewCalendar(cfg Config) *Calendar { return &Calendar{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*Calendar) Kind() string { return fact.KindCalendar }

// Poll 计算月末/季末/周四三个规则事件 + 文件人工事件；文件配置了但读不出 → 失败（配置错误）。
func (c *Calendar) Poll(_ context.Context) ([]fact.Fact, error) {
	now := c.now()
	today := midnight(now)
	out := []fact.Fact{
		calendarFact(VenueRule, "month_end", nextMonthEnd(today), now, "rule"),
		calendarFact(VenueRule, "quarter_end", nextQuarterEnd(today), now, "rule"),
		calendarFact(VenueRule, "thursday", nextThursday(today), now, "rule"),
	}
	events, err := c.loadManual()
	if err != nil {
		return nil, err
	}
	for _, e := range events {
		if e.Date.Before(today) {
			continue // 过期事件不入库（历史留痕不靠日历流）
		}
		out = append(out, calendarFact(VenueManual, e.Name, e.Date, now, "calendar-file "+c.cfg.File))
	}
	return out, nil
}

func (c *Calendar) now() time.Time {
	if c.cfg.Now != nil {
		return c.cfg.Now()
	}
	return time.Now()
}

// loadManual 读取人工事件表（每次 Poll 重读，改表免重启）；空路径 = 无事件。
func (c *Calendar) loadManual() ([]event, error) {
	if c.cfg.File == "" {
		return nil, nil
	}
	b, err := os.ReadFile(c.cfg.File)
	if err != nil {
		return nil, fmt.Errorf("calendar: read %s: %w", c.cfg.File, err)
	}
	var raw []struct {
		Date string `json:"date"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("calendar: parse %s: %w", c.cfg.File, err)
	}
	out := make([]event, 0, len(raw))
	for i, e := range raw {
		if e.Name == "" {
			return nil, fmt.Errorf("calendar: %s: events[%d]: empty name", c.cfg.File, i)
		}
		d, err := time.ParseInLocation("2006-01-02", e.Date, collect.CST)
		if err != nil {
			return nil, fmt.Errorf("calendar: %s: events[%d]: bad date %q", c.cfg.File, i, e.Date)
		}
		out = append(out, event{Date: d, Name: e.Name})
	}
	return out, nil
}

// calendarFact 组装日历 Fact；Value = 距事件天数（日历日，当日 = 0）。
func calendarFact(venue, symbol string, date, ts time.Time, src string) fact.Fact {
	return fact.Fact{
		Kind:   fact.KindCalendar,
		Venue:  venue,
		Symbol: symbol,
		Value:  float64(daysUntil(midnight(ts), date)),
		Unit:   fact.UnitDays,
		Ts:     ts,
		Src:    src,
	}
}

// midnight 对齐到 CST 零点（日期计算基准）。
func midnight(t time.Time) time.Time {
	y, m, d := t.In(collect.CST).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, collect.CST)
}

// daysUntil 日历日差（to − from 的整天数；当日 = 0）。
func daysUntil(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}

// nextMonthEnd 返回 ≥ d 的最近一个自然月末（当日为月末 → 当日）。
func nextMonthEnd(d time.Time) time.Time {
	if e := monthEnd(d); !e.Before(d) {
		return e
	}
	return monthEnd(time.Date(d.Year(), d.Month()+1, 1, 0, 0, 0, 0, d.Location()))
}

// monthEnd 返回 d 所在月的最后一天。
func monthEnd(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location()).AddDate(0, 1, -1)
}

// nextQuarterEnd 返回 ≥ d 的最近一个季末（3/6/9/12 最后一天；当日为季末 → 当日）。
func nextQuarterEnd(d time.Time) time.Time {
	qm := ((int(d.Month())-1)/3 + 1) * 3 // 当前季度末月 3/6/9/12
	for i := 0; i < 4; i++ {
		// time.Date 自动进位（月 > 12 → 次年），4 次迭代必覆盖全部季末月
		e := time.Date(d.Year(), time.Month(qm+3*i), 1, 0, 0, 0, 0, d.Location()).AddDate(0, 1, -1)
		if !e.Before(d) {
			return e
		}
	}
	return time.Time{} // 不可达
}

// nextThursday 返回 ≥ d 的最近一个周四（当日为周四 → 当日；逆回购周四计息约定）。
func nextThursday(d time.Time) time.Time {
	days := (time.Thursday - d.Weekday() + 7) % 7
	return d.AddDate(0, 0, int(days))
}
