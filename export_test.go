package iso8583

import (
	"testing"

	"github.com/Pay8583/iso8583/spec"
)

func TestExportStruct(t *testing.T) {
	p := &spec.Protocol{
		Name: "test",
		Fields: []spec.Field{
			{Name: "F2", Secure: true},  // field 2
			{Name: "F3", Secure: false}, // field 3
		},
	}

	type TestMsg struct {
		PAN  string `iso8583:"2"`
		Code string `iso8583:"3"`
		MTI  uint   `iso8583:"mti"`
	}

	msg := TestMsg{PAN: "4000001234567890", Code: "301000", MTI: 0x0200}

	exp, err := ExportStruct(&msg, p)
	if err != nil {
		t.Fatalf("ExportStruct: %v", err)
	}

	// Secure field masked.
	if exp[2] != "" {
		t.Errorf("Export[2] = %q, want \"\" (secure)", exp[2])
	}
	// Non-secure field visible.
	if exp[3] != "301000" {
		t.Errorf("Export[3] = %q, want 301000", exp[3])
	}
	// MTI not exported.
	if _, ok := exp[0]; ok {
		t.Error("MTI should not appear in export")
	}
}

func TestExportStructMasked(t *testing.T) {
	p := &spec.Protocol{
		Name: "test",
		Fields: []spec.Field{
			{Name: "F2", Secure: true},
		},
	}

	type TestMsg struct {
		PAN string `iso8583:"2"`
	}

	msg := TestMsg{PAN: "4000001234567890"}

	exp, err := ExportStructMasked(&msg, p, "***")
	if err != nil {
		t.Fatalf("ExportStructMasked: %v", err)
	}
	if exp[2] != "***" {
		t.Errorf("Export[2] = %q, want \"***\"", exp[2])
	}
}

func TestExportStruct_IntField(t *testing.T) {
	p := &spec.Protocol{
		Name: "test",
		Fields: []spec.Field{
			{Name: "F4", Secure: false}, // field 4
		},
	}

	type TestMsg struct {
		Amount int64 `iso8583:"4"`
	}

	msg := TestMsg{Amount: 1000}
	exp, err := ExportStruct(&msg, p)
	if err != nil {
		t.Fatalf("ExportStruct: %v", err)
	}
	if exp[4] != "1000" {
		t.Errorf("Export[4] = %q, want 1000", exp[4])
	}
}

func TestExportStruct_NotStruct(t *testing.T) {
	p := &spec.Protocol{}
	_, err := ExportStruct("not a struct", p)
	if err == nil {
		t.Error("expected error for non-struct")
	}
}

func TestExportStruct_FieldNotInProtocol(t *testing.T) {
	p := &spec.Protocol{
		Name:   "test",
		Fields: []spec.Field{}, // empty
	}

	type TestMsg struct {
		PAN string `iso8583:"2"`
	}

	msg := TestMsg{PAN: "4000001234567890"}
	exp, err := ExportStruct(&msg, p)
	if err != nil {
		t.Fatalf("ExportStruct: %v", err)
	}
	// Field not in protocol — not masked.
	if exp[2] != "4000001234567890" {
		t.Errorf("Export[2] = %q, want unmasked value", exp[2])
	}
}
