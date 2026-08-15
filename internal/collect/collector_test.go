package collect

import (
	"reflect"
	"testing"

	"arbcn/internal/fact"
)

// TestRegistryRegisterGet：注册后可取回；空名 / nil / 重名拒绝。
func TestRegistryRegisterGet(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Get("missing"); ok {
		t.Fatal("Get(missing) = true, want false")
	}
	c := &fakeCollector{kind: fact.KindFunding}
	if err := r.Register("binance_funding", c); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, ok := r.Get("binance_funding")
	if !ok || got != c {
		t.Fatalf("Get = %v, %v; want %v, true", got, ok, c)
	}
	if err := r.Register("binance_funding", c); err == nil {
		t.Fatal("duplicate Register = nil, want error")
	}
	for name, col := range map[string]Collector{"": c, "okx_funding": nil} {
		if err := r.Register(name, col); err == nil {
			t.Errorf("Register(%q, %v) = nil, want error", name, col)
		}
	}
}

// TestRegistryNames：全部注册名升序返回。
func TestRegistryNames(t *testing.T) {
	r := NewRegistry()
	c := &fakeCollector{kind: fact.KindFunding}
	for _, n := range []string{"okx_funding", "binance_funding", "binance_ticker"} {
		if err := r.Register(n, c); err != nil {
			t.Fatalf("Register(%q): %v", n, err)
		}
	}
	want := []string{"binance_funding", "binance_ticker", "okx_funding"}
	if got := r.Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}
