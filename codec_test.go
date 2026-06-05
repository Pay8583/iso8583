package iso8583

import (
	"testing"
)

func TestMarshal_Basic(t *testing.T) {
	p := testProtocol()

	type TestMsg struct {
		MTI  uint   `iso8583:"mti"`
		PAN  string `iso8583:"2"`
		Code string `iso8583:"3"`
		STAN string `iso8583:"11"`
	}

	msg := TestMsg{MTI: 0x0200, PAN: "4000001234567890", Code: "301000", STAN: "123456"}

	data, err := Marshal(&msg, p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("empty output")
	}
}

func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	p := testProtocol()

	type TestMsg struct {
		MTI  uint   `iso8583:"mti"`
		PAN  string `iso8583:"2"`
		Code string `iso8583:"3"`
		STAN string `iso8583:"11"`
	}

	original := TestMsg{MTI: 0x0200, PAN: "4000001234567890", Code: "301000", STAN: "123456"}

	data, err := Marshal(&original, p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded TestMsg
	if err := Unmarshal(data, &decoded, p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.MTI != original.MTI {
		t.Errorf("MTI = %#x, want %#x", decoded.MTI, original.MTI)
	}
	if decoded.PAN != original.PAN {
		t.Errorf("PAN = %q, want %q", decoded.PAN, original.PAN)
	}
	if decoded.Code != original.Code {
		t.Errorf("Code = %q, want %q", decoded.Code, original.Code)
	}
	if decoded.STAN != original.STAN {
		t.Errorf("STAN = %q, want %q", decoded.STAN, original.STAN)
	}
}

func TestMarshalUnmarshal_WithIntField(t *testing.T) {
	p := testProtocol()

	type TestMsg struct {
		MTI    uint  `iso8583:"mti"`
		Amount int64 `iso8583:"4"`
	}

	original := TestMsg{MTI: 0x0200, Amount: 1000}

	data, err := Marshal(&original, p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded TestMsg
	if err := Unmarshal(data, &decoded, p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Amount != 1000 {
		t.Errorf("Amount = %d, want 1000", decoded.Amount)
	}
}

func TestMarshal_NoMTI(t *testing.T) {
	p := testProtocol()

	type TestMsg struct {
		PAN string `iso8583:"2"`
	}

	msg := TestMsg{PAN: "1234567890"}
	_, err := Marshal(&msg, p)
	if err == nil {
		t.Error("expected error: no MTI field")
	}
}

func TestMarshal_MTIString(t *testing.T) {
	p := testProtocol()

	type TestMsg struct {
		MTI string `iso8583:"mti"`
	}

	msg := TestMsg{MTI: "0200"}
	data, err := Marshal(&msg, p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded TestMsg
	if err := Unmarshal(data, &decoded, p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.MTI != "0200" {
		t.Errorf("MTI = %q, want 0200", decoded.MTI)
	}
}

func TestUnmarshal_NotPointer(t *testing.T) {
	p := testProtocol()
	type TestMsg struct {
		MTI uint `iso8583:"mti"`
	}
	var msg TestMsg
	err := Unmarshal([]byte{}, msg, p) // value, not pointer
	if err == nil {
		t.Error("expected error for non-pointer")
	}
}

func TestUnmarshal_NilPointer(t *testing.T) {
	p := testProtocol()
	type TestMsg struct {
		MTI uint `iso8583:"mti"`
	}
	var msg *TestMsg
	err := Unmarshal([]byte{}, msg, p) // nil pointer
	if err == nil {
		t.Error("expected error for nil pointer")
	}
}

func TestMarshal_Validation(t *testing.T) {
	p := testProtocol()

	type TestMsg struct {
		MTI  uint   `iso8583:"mti"`
		Code string `iso8583:"3"` // Valid: N (numeric only)
	}

	msg := TestMsg{MTI: 0x0200, Code: "ABC"} // invalid: non-numeric
	_, err := Marshal(&msg, p)
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestMarshal_OptionalField(t *testing.T) {
	p := testProtocol()

	type TestMsg struct {
		MTI  uint   `iso8583:"mti"`
		PAN  string `iso8583:"2"` // not optional — empty will error
		Code string `iso8583:"3,optional"`
	}

	// PAN is empty and not optional — should be omitted.
	// Code is optional and empty — should be omitted.
	msg := TestMsg{MTI: 0x0200, Code: "301000"}
	data, err := Marshal(&msg, p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded TestMsg
	if err := Unmarshal(data, &decoded, p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Code != "301000" {
		t.Errorf("Code = %q, want 301000", decoded.Code)
	}
}
