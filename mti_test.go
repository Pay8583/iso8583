package iso8583

import "testing"

func TestMTIVersion(t *testing.T) {
	tests := []struct {
		mti  string
		want int
	}{
		{"0100", 1987},
		{"0110", 1993},
		{"0120", 2003},
		{"0200", 1987},
		{"0210", 1993},
		{"0800", 1987},
	}
	for _, tt := range tests {
		got, err := MTIVersion(tt.mti)
		if err != nil {
			t.Errorf("MTIVersion(%q): %v", tt.mti, err)
			continue
		}
		if got != tt.want {
			t.Errorf("MTIVersion(%q) = %d, want %d", tt.mti, got, tt.want)
		}
	}
}

func TestMTIVersion_Invalid(t *testing.T) {
	_, err := MTIVersion("01")
	if err == nil {
		t.Error("expected error for short MTI")
	}
	_, err = MTIVersion("01X0")
	if err == nil {
		t.Error("expected error for invalid version digit")
	}
}

func TestIsRequest_Response(t *testing.T) {
	if !IsRequest("0200") {
		t.Error("0200 should be a request")
	}
	if !IsRequest("0212") {
		t.Error("0212 should be a request")
	}
	if !IsResponse("0211") {
		t.Error("0211 should be a response")
	}
	if !IsResponse("0113") {
		t.Error("0113 should be a response")
	}
	if IsRequest("") {
		t.Error("empty string should not be a request")
	}
}

func TestMTIClass(t *testing.T) {
	class, err := MTIClass("0200")
	if err != nil {
		t.Fatal(err)
	}
	if class != "02" {
		t.Errorf("MTIClass(0200) = %q, want %q", class, "02")
	}
}
