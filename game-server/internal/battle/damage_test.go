package battle

import "testing"

func TestCalcTapDamage(t *testing.T) {
	matchups := TypeMatchup{
		"static_typing": {"dynamic_typing": 2.0},
	}

	tests := []struct {
		name       string
		attack     int32
		atkType    string
		defType    string
		wantDamage int32
	}{
		{"normal effectiveness", 100, "static_typing", "static_typing", 100},
		{"super effective", 100, "static_typing", "dynamic_typing", 200},
		{"unknown matchup defaults to 1.0", 100, "functional", "procedural", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcTapDamage(tt.attack, tt.atkType, tt.defType, matchups)
			if got != tt.wantDamage {
				t.Errorf("CalcTapDamage() = %d, want %d", got, tt.wantDamage)
			}
		})
	}
}

func TestCalcSpecialDamage(t *testing.T) {
	matchups := TypeMatchup{
		"static_typing": {"dynamic_typing": 2.0},
	}
	got := CalcSpecialDamage(500, "static_typing", "dynamic_typing", matchups)
	if got != 1000 {
		t.Errorf("CalcSpecialDamage() = %d, want 1000", got)
	}
}

func TestRequiredTapsForSpecial(t *testing.T) {
	// base_taps=20, speed=100, coefficient=10 -> 20 - (100/10) = 10
	got := RequiredTapsForSpecial(100, 20, 10)
	if got != 10 {
		t.Errorf("RequiredTapsForSpecial() = %d, want 10", got)
	}

	// Result should not go below 1
	got = RequiredTapsForSpecial(9999, 20, 10)
	if got < 1 {
		t.Errorf("RequiredTapsForSpecial() = %d, want >= 1", got)
	}
}
