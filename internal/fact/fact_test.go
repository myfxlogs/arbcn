package fact

import "testing"

// TestValidateKnownKinds：常量表内全部 Kind 必须通过校验（与 validKinds 表一致）。
func TestValidateKnownKinds(t *testing.T) {
	known := []string{
		KindFunding, KindDefiRate, KindReverseRepo, KindFX, KindIV, KindCalendar, KindTicker,
	}
	for _, k := range known {
		if !ValidKind(k) {
			t.Errorf("ValidKind(%q) = false, want true", k)
		}
		if err := (Fact{Kind: k, Venue: "v", Symbol: "s"}).Validate(); err != nil {
			t.Errorf("Validate(kind=%q) = %v, want nil", k, err)
		}
	}
}

// TestValidateUnknownKind：未知 Kind（含空串/大小写变体/采集器名）必须拒绝。
func TestValidateUnknownKind(t *testing.T) {
	unknown := []string{"", "fund", "FUNDING", "Funding", "stake", "domestic", "bank_rate"}
	for _, k := range unknown {
		if ValidKind(k) {
			t.Errorf("ValidKind(%q) = true, want false", k)
		}
		if err := (Fact{Kind: k}).Validate(); err == nil {
			t.Errorf("Validate(kind=%q) = nil, want error", k)
		}
	}
}
