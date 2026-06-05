package spec

import (
	"testing"

	"github.com/Pay8583/iso8583/encoding"
)

func TestProtocol_GetField(t *testing.T) {
	p := &Protocol{
		Name: "test",
		Fields: []Field{
			{Name: "Field2", Len: Fixed(6, '0'), Valid: N, Value: ASCII},
			{Name: "Field3", Len: Fixed(12, '0'), Valid: N, Value: ASCII},
		},
	}

	// Valid field indices.
	if f := p.GetField(2); f == nil || f.Name != "Field2" {
		t.Errorf("GetField(2) = %v, want Field2", f)
	}
	if f := p.GetField(3); f == nil || f.Name != "Field3" {
		t.Errorf("GetField(3) = %v, want Field3", f)
	}

	// Out of range.
	if f := p.GetField(1); f != nil {
		t.Errorf("GetField(1) = %v, want nil", f)
	}
	if f := p.GetField(5); f != nil {
		t.Errorf("GetField(5) = %v, want nil", f)
	}
	if f := p.GetField(128); f != nil {
		t.Errorf("GetField(128) = %v, want nil (only 2 fields defined)", f)
	}
}

func TestProtocol_NumFields(t *testing.T) {
	p := &Protocol{Fields: []Field{{}, {}, {}}}
	if p.NumFields() != 3 {
		t.Errorf("NumFields() = %d, want 3", p.NumFields())
	}
}

func TestProtocol_HasSecondaryBitmap(t *testing.T) {
	short := &Protocol{Fields: make([]Field, 63)}
	if short.HasSecondaryBitmap() {
		t.Error("63 fields should not have secondary bitmap")
	}

	long := &Protocol{Fields: make([]Field, 64)}
	if !long.HasSecondaryBitmap() {
		t.Error("64 fields should have secondary bitmap (field 65 requires it)")
	}

	long2 := &Protocol{Fields: make([]Field, 127)}
	if !long2.HasSecondaryBitmap() {
		t.Error("127 fields should have secondary bitmap")
	}
}

func TestField_IsAllowed(t *testing.T) {
	nonSecure := &Field{Name: "Test", Secure: false}
	if !nonSecure.IsAllowed(SecurityLevelLow) {
		t.Error("non-secure field should be allowed at low security")
	}
	if !nonSecure.IsAllowed(SecurityLevelHigh) {
		t.Error("non-secure field should be allowed at high security")
	}

	secure := &Field{Name: "PAN", Secure: true}
	if secure.IsAllowed(SecurityLevelLow) {
		t.Error("secure field should NOT be allowed at low security")
	}
	if !secure.IsAllowed(SecurityLevelHigh) {
		t.Error("secure field should be allowed at high security")
	}
}

func TestRegistry(t *testing.T) {
	// Get built-in specs.
	p1987, err := Get("1987")
	if err != nil {
		t.Fatalf("Get(1987): %v", err)
	}
	if p1987.Version != "1987" {
		t.Errorf("V1987.Version = %q, want %q", p1987.Version, "1987")
	}
	if p1987.MTI != ASCIIMTI {
		t.Error("V1987.MTI should be ASCIIMTI")
	}
	if p1987.HasSecondaryBitmap() {
		t.Error("V1987 should not have secondary bitmap")
	}

	p1993, err := Get("1993")
	if err != nil {
		t.Fatalf("Get(1993): %v", err)
	}
	if !p1993.HasSecondaryBitmap() {
		t.Error("V1993 should have secondary bitmap")
	}

	p2003, err := Get("2003")
	if err != nil {
		t.Fatalf("Get(2003): %v", err)
	}
	if p2003.Version != "2003" {
		t.Errorf("V2003.Version = %q, want %q", p2003.Version, "2003")
	}

	// MustGet panics on missing.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGet(nonexistent) should panic")
			}
		}()
		MustGet("nonexistent")
	}()

	// List.
	names := List()
	if len(names) < 3 {
		t.Errorf("List() returned %d names, want at least 3", len(names))
	}
}

func TestMustRegister(t *testing.T) {
	// Register a new protocol.
	custom := &Protocol{
		Name:    "custom-test",
		Version: "custom",
		Fields:  []Field{{Name: "F2", Len: Fixed(6, '0'), Valid: N, Value: ASCII}},
	}

	// First registration succeeds.
	if err := Register(custom); err != nil {
		t.Fatalf("Register(custom): %v", err)
	}

	// Second registration of the same pointer is idempotent.
	if err := Register(custom); err != nil {
		t.Errorf("Register(same pointer): %v", err)
	}

	// Different pointer with same name fails.
	custom2 := &Protocol{Name: "custom-test", Version: "v2"}
	if err := Register(custom2); err == nil {
		t.Error("Register(different pointer, same name): expected error")
	}

	// MustRegister panics on duplicate.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustRegister(duplicate) should panic")
			}
		}()
		MustRegister(custom2)
	}()

	// Clean up: delete from registry.
	delete(registry, "custom-test")
}

func TestMTIEncoding(t *testing.T) {
	if ASCIIMTI.WireLen() != 4 {
		t.Error("ASCIIMTI.WireLen should be 4")
	}
	if BCDMTI.WireLen() != 2 {
		t.Error("BCDMTI.WireLen should be 2")
	}
	if BinaryMTI.WireLen() != 1 {
		t.Error("BinaryMTI.WireLen should be 1")
	}
}

func TestBitmapEncoding(t *testing.T) {
	if HexBitmap != 0 {
		t.Error("HexBitmap should be 0")
	}
	if BinaryBitmap != 1 {
		t.Error("BinaryBitmap should be 1")
	}
}

func TestSecurityLevel(t *testing.T) {
	if SecurityLevelLow != 0 {
		t.Error("SecurityLevelLow should be 0")
	}
	if SecurityLevelHigh != 1 {
		t.Error("SecurityLevelHigh should be 1")
	}
}

func TestBuiltInSpecs_HaveCorrectFieldCounts(t *testing.T) {
	p := MustGet("1987")
	if p.NumFields() != 63 {
		t.Errorf("V1987 has %d fields, want 63 (fields 2–64)", p.NumFields())
	}
	if p.GetField(2).Name != "PAN" {
		t.Errorf("V1987 field 2 name = %q, want PAN", p.GetField(2).Name)
	}
	if !p.GetField(2).Secure {
		t.Error("V1987 field 2 (PAN) should be Secure")
	}

	p2 := MustGet("1993")
	// Fields 2–128: 127 field definitions.
	if p2.NumFields() < 126 {
		t.Errorf("V1993 has %d fields, want at least 126 (fields 2–128)", p2.NumFields())
	}
	if f := p2.GetField(65); f == nil || f.Name != "SettlementCode" {
		t.Errorf("V1993 field 65 = %v, want SettlementCode", f)
	}

	p3 := MustGet("2003")
	if p3.NumFields() != 127 {
		t.Errorf("V2003 has %d fields, want 127 (fields 2–128)", p3.NumFields())
	}
}

func TestField_Defaults(t *testing.T) {
	f := Field{Name: "Test", Len: Fixed(1, '0'), Valid: B, Value: ASCII}
	if f.Secure {
		t.Error("zero-value Secure should be false")
	}
	if !f.IsAllowed(SecurityLevelLow) {
		t.Error("non-secure field should be allowed at low")
	}
}

func TestLVAR_WithBinaryEncoding(t *testing.T) {
	// LVAR with binary encoding (used in custom protocols like PEP ISO).
	l := LVAR(255, encoding.MustGet("binary"))
	if l.WireLen() != 1 {
		t.Errorf("WireLen() = %d, want 1", l.WireLen())
	}
}

func TestLLLVAR_MaxLimits(t *testing.T) {
	// Max values are clipped to what fits in the prefix.
	l := LVAR(500, nil)     // 500 max, but only 99 fits in 1 BCD byte
	if l.MaxLen() != 500 {   // max is NOT clipped; validation happens at write time
		t.Errorf("MaxLen() = %d, want 500 (not clipped by prefix size)", l.MaxLen())
	}
}
