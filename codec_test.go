package iso8583

import (
	"testing"
	"time"

	"github.com/Pay8583/iso8583/spec"
)

type authRequest struct {
	MTI          string `iso8583:"mti"`
	PAN          string `iso8583:"2,llvar,bcd,numeric"`
	ProcCode     string `iso8583:"3,bcd,numeric,len=6"`
	Amount       int64  `iso8583:"4,bcd,numeric,len=12"`
	Transmission string `iso8583:"7,bcd,numeric,len=10"`
	STAN         string `iso8583:"11,bcd,numeric,len=6"`
	TimeLocal    string `iso8583:"12,bcd,numeric,len=6"`
	DateLocal    string `iso8583:"13,bcd,numeric,len=4"`
	Track2       string `iso8583:"35,llvar,ascii"`
	RetRefNum    string `iso8583:"37,ascii,len=12"`
	TerminalID   string `iso8583:"41,ascii,len=8"`
	MerchantID   string `iso8583:"42,ascii,len=15"`
	CurrencyCode string `iso8583:"49,ascii,len=3"`
	Signature    []byte `iso8583:"64,binary,len=8,optional"`
}

func TestMarshalUnmarshal_Roundtrip(t *testing.T) {
	s := spec.MustGet("1987")

	req := authRequest{
		MTI:          "0200",
		PAN:          "4000001234567890",
		ProcCode:     "301000",
		Amount:       1000,
		Transmission: "0503143015",
		STAN:         "123456",
		TimeLocal:    "143015",
		DateLocal:    "0503",
		Track2:       "4000001234567890=250510100000000",
		RetRefNum:    "123456789012",
		TerminalID:   "TERM0001",
		MerchantID:   "MERCHANT01     ",
		CurrencyCode: "840",
		Signature:    []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x00, 0x00, 0x00},
	}

	data, err := Marshal(&req, s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	t.Logf("marshaled %d bytes", len(data))

	var req2 authRequest
	if err := Unmarshal(data, &req2, s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if req2.MTI != req.MTI {
		t.Errorf("MTI = %q, want %q", req2.MTI, req.MTI)
	}
	if req2.PAN != req.PAN {
		t.Errorf("PAN = %q, want %q", req2.PAN, req.PAN)
	}
	if req2.Amount != req.Amount {
		t.Errorf("Amount = %d, want %d", req2.Amount, req.Amount)
	}
	if req2.ProcCode != req.ProcCode {
		t.Errorf("ProcCode = %q, want %q", req2.ProcCode, req.ProcCode)
	}
	if req2.STAN != req.STAN {
		t.Errorf("STAN = %q, want %q", req2.STAN, req.STAN)
	}
	if req2.TerminalID != req.TerminalID {
		t.Errorf("TerminalID = %q, want %q", req2.TerminalID, req.TerminalID)
	}
	if req2.CurrencyCode != req.CurrencyCode {
		t.Errorf("CurrencyCode = %q, want %q", req2.CurrencyCode, req.CurrencyCode)
	}
	if string(req2.Signature) != string(req.Signature) {
		t.Errorf("Signature = %x, want %x", req2.Signature, req.Signature)
	}
}

func TestMarshalUnmarshal_NoOptional(t *testing.T) {
	s := spec.MustGet("1987")
	req := authRequest{
		MTI:          "0100",
		PAN:          "4000001234567890",
		ProcCode:     "000000",
		Amount:       0,
		TerminalID:   "TERM0001",
		MerchantID:   "MERCH01        ",
		CurrencyCode: "840",
	}

	data, err := Marshal(&req, s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var req2 authRequest
	if err := Unmarshal(data, &req2, s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if req2.MTI != "0100" {
		t.Errorf("MTI = %q", req2.MTI)
	}
	if req2.PAN != "4000001234567890" {
		t.Errorf("PAN mismatch")
	}
	// Optional field (signature) should be nil/empty since not set in original.
	if len(req2.Signature) != 0 {
		t.Errorf("Signature should be empty for optional field, got %x", req2.Signature)
	}
}

func TestMarshal_NoMTI(t *testing.T) {
	type bad struct {
		PAN string `iso8583:"2,llvar,bcd"`
	}
	_, err := Marshal(&bad{PAN: "123"}, spec.MustGet("1987"))
	if err == nil {
		t.Error("expected error for missing MTI")
	}
}

func TestMarshal_NotStruct(t *testing.T) {
	s := spec.MustGet("1987")
	_, err := Marshal(42, s)
	if err == nil {
		t.Error("expected error for non-struct")
	}
}

func TestUnmarshal_NotPointer(t *testing.T) {
	var req authRequest
	err := Unmarshal(nil, req, spec.MustGet("1987"))
	if err == nil {
		t.Error("expected error for non-pointer")
	}
}

func TestMarshalUnmarshal_TimeField(t *testing.T) {
	type reversal struct {
		MTI      string    `iso8583:"mti"`
		DateTime time.Time `iso8583:"7,bcd,numeric,len=10"`
		PAN      string    `iso8583:"2,llvar,bcd"`
	}

	s := spec.MustGet("1987")
	tm := time.Date(2025, 3, 14, 15, 30, 0, 0, time.UTC)

	rev := reversal{MTI: "0400", DateTime: tm, PAN: "4000001234567890"}
	data, err := Marshal(&rev, s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var rev2 reversal
	if err := Unmarshal(data, &rev2, s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Time roundtrip through ISO format (MMDDHHMMSS) loses year, sub-second, and TZ info.
	if rev2.DateTime.Month() != tm.Month() ||
		rev2.DateTime.Day() != tm.Day() ||
		rev2.DateTime.Hour() != tm.Hour() ||
		rev2.DateTime.Minute() != tm.Minute() {
		t.Errorf("DateTime = %v, want MMDDHHMM from %v", rev2.DateTime, tm)
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkCodec_Marshal_10fields(b *testing.B) {
	s := spec.MustGet("1987")
	req := authRequest{
		MTI:          "0200",
		PAN:          "4000001234567890",
		ProcCode:     "301000",
		Amount:       1000,
		Transmission: "0503143015",
		STAN:         "123456",
		TimeLocal:    "143015",
		DateLocal:    "0503",
		Track2:       "4000001234567890=250510100000000",
		RetRefNum:    "123456789012",
		TerminalID:   "TERM0001",
		MerchantID:   "MERCHANT01     ",
		CurrencyCode: "840",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Marshal(&req, s)
	}
}

func BenchmarkCodec_Unmarshal_10fields(b *testing.B) {
	s := spec.MustGet("1987")
	req := authRequest{
		MTI:          "0200",
		PAN:          "4000001234567890",
		ProcCode:     "301000",
		Amount:       1000,
		Transmission: "0503143015",
		STAN:         "123456",
		TimeLocal:    "143015",
		DateLocal:    "0503",
		Track2:       "4000001234567890=250510100000000",
		RetRefNum:    "123456789012",
		TerminalID:   "TERM0001",
		MerchantID:   "MERCHANT01     ",
		CurrencyCode: "840",
	}
	data, _ := Marshal(&req, s)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var req2 authRequest
		Unmarshal(data, &req2, s)
	}
}
