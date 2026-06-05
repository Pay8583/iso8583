package spec

import (
	"strings"
	"testing"
)

func TestValidator_N(t *testing.T) {
	valid := []string{"", "0", "1234567890", "0000", "9"}
	for _, s := range valid {
		if !N.Ok(s) {
			t.Errorf("N.Ok(%q) = false, want true", s)
		}
	}
	invalid := []string{"a", "1a", " 1", "1 ", "-1", "1.0", "١٢٣", "ABC", "\x00", "\x7F"}
	for _, s := range invalid {
		if N.Ok(s) {
			t.Errorf("N.Ok(%q) = true, want false", s)
		}
	}
}

func TestValidator_AN(t *testing.T) {
	valid := []string{"", "abc", "ABC", "aBc123", "A1b2C3", "000"}
	for _, s := range valid {
		if !AN.Ok(s) {
			t.Errorf("AN.Ok(%q) = false, want true", s)
		}
	}
	invalid := []string{"abc def", "abc-def", "abc.def", "abc_def", "\x00", "\x7F", "@#$"}
	for _, s := range invalid {
		if AN.Ok(s) {
			t.Errorf("AN.Ok(%q) = true, want false", s)
		}
	}
}

func TestValidator_ANS(t *testing.T) {
	valid := []string{"", "hello world", "ABC 123", "test@example.com", "a-b_c.d",
		"printable!@#$%^&*()", "0x20 to 0x7E"}
	for _, s := range valid {
		if !ANS.Ok(s) {
			t.Errorf("ANS.Ok(%q) = false, want true", s)
		}
	}
	invalid := []string{"\x00", "\x01", "\x1F", "\x7F", "\x80", "\xFF",
		"tab\tchar", "newline\n", "carriage\r"}
	for _, s := range invalid {
		if ANS.Ok(s) {
			t.Errorf("ANS.Ok(%q) = true, want false", s)
		}
	}
}

func TestValidator_B(t *testing.T) {
	// B accepts everything, always.
	tests := []string{"", "abc", "123", "\x00\x01\x02", "\xFF\xFE", "\x7F", "\t\n\r",
		strings.Repeat("x", 1000)}
	for _, s := range tests {
		if !B.Ok(s) {
			t.Errorf("B.Ok(%q) = false, want true", s)
		}
	}
}

func TestValidator_Z(t *testing.T) {
	valid := []string{"", "4000001234567890=25052011234567890000", ";1234567890123456=2512?",
		"ABC DEF", "track1 data %B4815881002867896^SMITH/JOHN^2512?"}
	for _, s := range valid {
		if !Z.Ok(s) {
			t.Errorf("Z.Ok(%q) = false, want true", s)
		}
	}
	invalid := []string{"\x00", "\x01", "\x1F", "\x7F", "\x80",
		"binary\x00data"}
	for _, s := range invalid {
		if Z.Ok(s) {
			t.Errorf("Z.Ok(%q) = true, want false", s)
		}
	}
}

func TestValidator_NS(t *testing.T) {
	valid := []string{"", "123", "123 456", "12.34", "12-34", "1 2.3-4"}
	for _, s := range valid {
		if !NS.Ok(s) {
			t.Errorf("NS.Ok(%q) = false, want true", s)
		}
	}
	invalid := []string{"abc", "12_34", "12/34", "@12", "12#", "\x00", "+12"}
	for _, s := range invalid {
		if NS.Ok(s) {
			t.Errorf("NS.Ok(%q) = true, want false", s)
		}
	}
}

func TestValidator_XN(t *testing.T) {
	valid := []string{"", "0", "123", "+123", "-123", "+0", "-0"}
	for _, s := range valid {
		if !XN.Ok(s) {
			t.Errorf("XN.Ok(%q) = false, want true", s)
		}
	}
	invalid := []string{"+", "-", "+-123", "-+123", "12.34", " 12", "12 ", "abc", "+12a", "1+2"}
	for _, s := range invalid {
		if XN.Ok(s) {
			t.Errorf("XN.Ok(%q) = true, want false", s)
		}
	}
}

func TestValidator_String(t *testing.T) {
	tests := []struct {
		v    Validator
		want string
	}{
		{N, "N"},
		{AN, "AN"},
		{ANS, "ANS"},
		{B, "B"},
		{Z, "Z"},
		{NS, "NS"},
		{XN, "XN"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("%s.String() = %q, want %q", tt.want, got, tt.want)
		}
	}
}

func TestValidatorByName(t *testing.T) {
	tests := []struct {
		name string
		want Validator
	}{
		{"N", N},
		{"n", N},
		{"AN", AN},
		{"an", AN},
		{"ANS", ANS},
		{"ans", ANS},
		{"B", B},
		{"b", B},
		{"Z", Z},
		{"z", Z},
		{"NS", NS},
		{"ns", NS},
		{"XN", XN},
		{"xn", XN},
		{"unknown", nil},
		{"", nil},
		{"XYZ", nil},
	}
	for _, tt := range tests {
		got := ValidatorByName(tt.name)
		if got != tt.want {
			t.Errorf("ValidatorByName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAllValidators(t *testing.T) {
	all := AllValidators()
	if len(all) != 7 {
		t.Errorf("AllValidators() returned %d items, want 7", len(all))
	}
	seen := make(map[string]bool)
	for _, v := range all {
		if seen[v.String()] {
			t.Errorf("duplicate validator %q in AllValidators", v.String())
		}
		seen[v.String()] = true
	}
}
