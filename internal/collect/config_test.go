package collect

import (
	"testing"
	"time"

	"arbcn/internal/fact"
)

func defaultSources() []Named {
	c := &fakeCollector{kind: fact.KindFunding}
	return []Named{
		{Name: "binance_funding", Interval: 5 * time.Minute, Collector: c},
		{Name: "binance_ticker", Interval: time.Minute, Collector: c},
	}
}

// TestLoadSources：空 spec 全开默认；off 全关；列出的覆盖间隔；错误路径 fail fast。
func TestLoadSources(t *testing.T) {
	d := defaultSources()
	got, err := LoadSources("", d)
	if err != nil || len(got) != 2 {
		t.Fatalf(`LoadSources("") = %d srcs, %v; want 2, nil`, len(got), err)
	}
	if got, err := LoadSources("off", d); err != nil || got != nil {
		t.Fatalf(`LoadSources("off") = %v, %v; want nil, nil`, got, err)
	}
	got, err = LoadSources("binance_ticker=30s", d)
	if err != nil {
		t.Fatalf("LoadSources(30s): %v", err)
	}
	if len(got) != 1 || got[0].Name != "binance_ticker" || got[0].Interval != 30*time.Second {
		t.Fatalf("LoadSources(30s) = %+v", got)
	}
	for _, spec := range []string{
		"unknown=1m", "binance_funding", "binance_funding=0s",
		"binance_funding=abc", "binance_funding=1m,binance_funding=2m", "=1m",
	} {
		if _, err := LoadSources(spec, d); err == nil {
			t.Errorf("LoadSources(%q) = nil error, want error", spec)
		}
	}
}
