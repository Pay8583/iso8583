package spec

import "github.com/Pay8583/iso8583/encoding"

// ISO8583_1987 is the built-in spec for ISO 8583:1987 (fields 1–64).
var ISO8583_1987 *Spec

func init() {
	asciiEnc := encoding.MustGet("ascii")
	bcdEnc := encoding.MustGet("bcd")

	s := NewSpec("1987", "1987")
	s.MtiEncoder = asciiEnc
	s.MaxField = 64
	s.HasSecondaryBitmap = false
	s.Description = "ISO 8583:1987 — original standard, fields 1–64, no secondary bitmap"

	fields := []FieldSpec{
		{Index: 2, Name: "Primary Account Number", LengthType: LLVAR, ContentType: Numeric, Encoder: bcdEnc, MaxLen: 19},
		{Index: 3, Name: "Processing Code", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 6},
		{Index: 4, Name: "Transaction Amount", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 12},
		{Index: 7, Name: "Transmission Date & Time", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 10},
		{Index: 11, Name: "Systems Trace Audit Number", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 6},
		{Index: 12, Name: "Time Local Transaction", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 6},
		{Index: 13, Name: "Date Local Transaction", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 4},
		{Index: 14, Name: "Expiration Date", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 4},
		{Index: 15, Name: "Settlement Date", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 4},
		{Index: 18, Name: "Merchant Type", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 4},
		{Index: 22, Name: "POS Entry Mode", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 3},
		{Index: 23, Name: "Card Sequence Number", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 3},
		{Index: 25, Name: "POS Condition Code", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 2},
		{Index: 28, Name: "Transaction Fee Amount", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 8},
		{Index: 32, Name: "Acquiring Institution ID", LengthType: LLVAR, ContentType: Numeric, Encoder: bcdEnc, MaxLen: 11},
		{Index: 33, Name: "Forwarding Institution ID", LengthType: LLVAR, ContentType: Numeric, Encoder: bcdEnc, MaxLen: 11},
		{Index: 35, Name: "Track 2 Data", LengthType: LLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 37},
		{Index: 37, Name: "Retrieval Reference Number", LengthType: Fixed, ContentType: Alpha, Encoder: asciiEnc, FixedLen: 12},
		{Index: 38, Name: "Authorization ID Response", LengthType: Fixed, ContentType: Alpha, Encoder: asciiEnc, FixedLen: 6},
		{Index: 39, Name: "Response Code", LengthType: Fixed, ContentType: Alpha, Encoder: asciiEnc, FixedLen: 2},
		{Index: 41, Name: "Card Acceptor Terminal ID", LengthType: Fixed, ContentType: Alpha, Encoder: asciiEnc, FixedLen: 8},
		{Index: 42, Name: "Card Acceptor ID", LengthType: Fixed, ContentType: Alpha, Encoder: asciiEnc, FixedLen: 15},
		{Index: 43, Name: "Card Acceptor Name/Location", LengthType: Fixed, ContentType: Alpha, Encoder: asciiEnc, FixedLen: 40},
		{Index: 45, Name: "Track 1 Data", LengthType: LLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 76},
		{Index: 48, Name: "Additional Data — Private", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999},
		{Index: 49, Name: "Currency Code (Transaction)", LengthType: Fixed, ContentType: Alpha, Encoder: asciiEnc, FixedLen: 3},
		{Index: 51, Name: "Currency Code (Cardholder Billing)", LengthType: Fixed, ContentType: Alpha, Encoder: asciiEnc, FixedLen: 3},
		{Index: 52, Name: "PIN Data", LengthType: Fixed, ContentType: Binary, Encoder: encoding.MustGet("ascii"), FixedLen: 8},
		{Index: 53, Name: "Security Control Info", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 16},
		{Index: 54, Name: "Additional Amounts", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 120},
		{Index: 55, Name: "ICC System Related Data", LengthType: LLLVAR, ContentType: Binary, Encoder: asciiEnc, MaxLen: 255},
		{Index: 60, Name: "Private Data (60)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999},
		{Index: 61, Name: "Private Data (61)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999},
		{Index: 62, Name: "Private Data (62)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999},
		{Index: 63, Name: "Private Data (63)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999},
		{Index: 64, Name: "MAC", LengthType: Fixed, ContentType: Binary, Encoder: encoding.MustGet("ascii"), FixedLen: 8},
	}

	for i := range fields {
		s.AddField(fields[i].Index, fields[i])
	}

	ISO8583_1987 = s
	MustRegister(s)
}
