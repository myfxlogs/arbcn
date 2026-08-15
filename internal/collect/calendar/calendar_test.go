package calendar

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arbcn/internal/collect"
	"arbcn/internal/fact"
)

func cst(y int, m time.Month, d int, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, collect.CST)
}

// TestComputedEvents：2026-08-15（周六，CST）→ 月末 16 天 / 季末 46 天 / 周四 5 天。
func TestComputedEvents(t *testing.T) {
	c := NewCalendar(Config{Now: func() time.Time { return cst(2026, 8, 15, 10, 0) }})
	if c.Kind() != fact.KindCalendar {
		t.Fatalf("Kind() = %q, want %q", c.Kind(), fact.KindCalendar)
	}
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(fs) != 3 {
		t.Fatalf("Poll = %d facts, want 3", len(fs))
	}
	want := []struct {
		symbol string
		days   float64
	}{
		{"month_end", 16}, {"quarter_end", 46}, {"thursday", 5},
	}
	for i, w := range want {
		f := fs[i]
		if err := f.Validate(); err != nil {
			t.Errorf("facts[%d]: Validate = %v", i, err)
		}
		if f.Venue != VenueRule || f.Unit != fact.UnitDays || f.Symbol != w.symbol {
			t.Errorf("facts[%d]: venue/unit/symbol = %q/%q/%q", i, f.Venue, f.Unit, f.Symbol)
		}
		if f.Value != w.days {
			t.Errorf("facts[%d]: Value = %v, want %v", i, f.Value, w.days)
		}
		if f.Src != "rule" {
			t.Errorf("facts[%d]: Src = %q", i, f.Src)
		}
	}
}

// TestComputedOnEventDay：当日即事件 → 0 天（月末/季末/周四同日三合一）。
func TestComputedOnEventDay(t *testing.T) {
	// 2026-09-30 是周三：恰为月末+季末；周四单独验证 2026-08-20
	cases := []struct {
		name string
		now  time.Time
		sym  string
	}{
		{"month-end-today", cst(2026, 9, 30, 12, 0), "month_end"},
		{"quarter-end-today", cst(2026, 9, 30, 12, 0), "quarter_end"},
		{"thursday-today", cst(2026, 8, 20, 12, 0), "thursday"},
	}
	for _, tc := range cases {
		fs, err := NewCalendar(Config{Now: func() time.Time { return tc.now }}).Poll(context.Background())
		if err != nil {
			t.Fatalf("%s: Poll: %v", tc.name, err)
		}
		for _, f := range fs {
			if f.Symbol == tc.sym && f.Value != 0 {
				t.Errorf("%s: Value = %v, want 0", tc.name, f.Value)
			}
		}
	}
}

// TestYearBoundary：12-31 为月末+季末；次年 1 月 → 3-31 季末；闰年 2 月 → 2-29。
func TestYearBoundary(t *testing.T) {
	fs, err := NewCalendar(Config{Now: func() time.Time { return cst(2026, 12, 31, 8, 0) }}).Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	for _, f := range fs {
		if (f.Symbol == "month_end" || f.Symbol == "quarter_end") && f.Value != 0 {
			t.Errorf("12-31 %s: Value = %v, want 0", f.Symbol, f.Value)
		}
	}
	fs, err = NewCalendar(Config{Now: func() time.Time { return cst(2027, 1, 15, 8, 0) }}).Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	got := map[string]float64{}
	for _, f := range fs {
		got[f.Symbol] = f.Value
	}
	// 2027-01-15 → 1-31 = 16 天；3-31 = 75 天
	if got["month_end"] != 16 {
		t.Errorf("2027-01-15 month_end = %v, want 16", got["month_end"])
	}
	if got["quarter_end"] != 75 {
		t.Errorf("2027-01-15 quarter_end = %v, want 75", got["quarter_end"])
	}
	// 2028-02-10（闰年）→ 2-29 = 19 天
	fs, err = NewCalendar(Config{Now: func() time.Time { return cst(2028, 2, 10, 8, 0) }}).Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll leap: %v", err)
	}
	for _, f := range fs {
		if f.Symbol == "month_end" && f.Value != 19 {
			t.Errorf("2028-02-10 month_end = %v, want 19", f.Value)
		}
	}
}

// TestManualEvents：过期事件跳过、当日 0 天、未来事件计天；Venue=manual（fixture 文件）。
func TestManualEvents(t *testing.T) {
	c := NewCalendar(Config{Now: func() time.Time { return cst(2026, 8, 15, 9, 0) }, File: "testdata/calendar_events.json"})
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	var manual []fact.Fact
	for _, f := range fs {
		if f.Venue == VenueManual {
			manual = append(manual, f)
		}
	}
	if len(manual) != 2 {
		t.Fatalf("manual facts = %d, want 2", len(manual))
	}
	if manual[0].Symbol != "Launchpool-当日" || manual[0].Value != 0 {
		t.Errorf("manual[0] = %q %v, want Launchpool-当日 0", manual[0].Symbol, manual[0].Value)
	}
	if manual[1].Symbol != "储蓄国债发行" || manual[1].Value != 26 {
		t.Errorf("manual[1] = %q %v, want 储蓄国债发行 26", manual[1].Symbol, manual[1].Value)
	}
	if !strings.Contains(manual[0].Src, "calendar-file") {
		t.Errorf("manual Src = %q, want calendar-file prefix", manual[0].Src)
	}
}

// TestManualFileErrors：配置了文件但缺失/坏 JSON/坏日期 → Poll 失败（配置错误 fail fast）。
func TestManualFileErrors(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name, content string
		want          string
	}{
		{"missing", "", "read"},
		{"bad-json", `not json`, "parse"},
		{"bad-date", `[{"date":"2026/09/10","name":"x"}]`, "bad date"},
		{"empty-name", `[{"date":"2026-09-10","name":""}]`, "empty name"},
	}
	for _, tc := range cases {
		var path string
		if tc.name != "missing" {
			path = filepath.Join(dir, tc.name+".json")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
		} else {
			path = filepath.Join(dir, "nope.json")
		}
		_, err := NewCalendar(Config{Now: time.Now, File: path}).Poll(context.Background())
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: Poll = %v, want error containing %q", tc.name, err, tc.want)
		}
	}
}

// TestFromEnv / TestAll：环境变量与命名 collector（calendar，24h）。
func TestFromEnvAndAll(t *testing.T) {
	cfg := FromEnv(func(string) string { return "" })
	if cfg.File != "" {
		t.Fatalf("File = %q, want empty", cfg.File)
	}
	cfg = FromEnv(func(string) string { return " /tmp/e.json " })
	if cfg.File != "/tmp/e.json" {
		t.Fatalf("File = %q", cfg.File)
	}
	ns := All(Config{})
	if len(ns) != 1 || ns[0].Name != "calendar" || ns[0].Interval != 24*time.Hour || ns[0].Collector == nil || ns[0].Collector.Kind() != fact.KindCalendar {
		t.Fatalf("All() = %+v", ns)
	}
}
