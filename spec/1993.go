package spec

import "github.com/Pay8583/iso8583/encoding"

var ISO8583_1993 *Spec

func init() {
	asciiEnc := encoding.MustGet("ascii")
	bcdEnc := encoding.MustGet("bcd")
	binEnc := encoding.MustGet("binary")

	s := NewSpec("1993", "1993")
	s.MtiEncoder = asciiEnc
	s.MaxField = 128
	s.HasSecondaryBitmap = true
	s.Description = "ISO 8583:1993 — extended standard, fields 1–128, secondary bitmap supported"

	// Fields 1–64: identical to 1987 but with binary encoder for binary fields.
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
		{Index: 52, Name: "PIN Data", LengthType: Fixed, ContentType: Binary, Encoder: binEnc, FixedLen: 8},
		{Index: 53, Name: "Security Control Info", LengthType: Fixed, ContentType: Binary, Encoder: binEnc, FixedLen: 8},
		{Index: 54, Name: "Additional Amounts", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 120},
		{Index: 55, Name: "ICC System Related Data", LengthType: LLLVAR, ContentType: Binary, Encoder: binEnc, MaxLen: 255},
		{Index: 60, Name: "Private Data (60)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999},
		{Index: 61, Name: "Private Data (61)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999},
		{Index: 62, Name: "Private Data (62)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999},
		{Index: 63, Name: "Private Data (63)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999},

		// Fields 65–128 (new in 1993).
		{Index: 65, Name: "Settlement Code", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 1, Optional: true},
		{Index: 70, Name: "Network Management Info Code", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 3, Optional: true},
		{Index: 90, Name: "Original Data Elements", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 95, Name: "Replacement Amounts", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 96, Name: "Message Security Code", LengthType: Fixed, ContentType: Binary, Encoder: binEnc, FixedLen: 8, Optional: true},
		{Index: 97, Name: "Net Settlement Amount", LengthType: Fixed, ContentType: Numeric, Encoder: bcdEnc, FixedLen: 17, Optional: true},
		{Index: 100, Name: "Receiving Institution ID", LengthType: LLVAR, ContentType: Numeric, Encoder: bcdEnc, MaxLen: 11, Optional: true},
		{Index: 102, Name: "Account ID 1", LengthType: LLVAR, ContentType: Numeric, Encoder: bcdEnc, MaxLen: 28, Optional: true},
		{Index: 103, Name: "Account ID 2", LengthType: LLVAR, ContentType: Numeric, Encoder: bcdEnc, MaxLen: 28, Optional: true},
		{Index: 120, Name: "Record Data (120)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 121, Name: "Record Data (121)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 122, Name: "Record Data (122)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 123, Name: "Record Data (123)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 124, Name: "Record Data (124)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 125, Name: "Record Data (125)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 126, Name: "Record Data (126)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 127, Name: "Private Data (127)", LengthType: LLLVAR, ContentType: Alpha, Encoder: asciiEnc, MaxLen: 999, Optional: true},
		{Index: 128, Name: "MAC", LengthType: Fixed, ContentType: Binary, Encoder: binEnc, FixedLen: 8, Optional: true},
	}

	for i := range fields {
		s.AddField(fields[i].Index, fields[i])
	}

	ISO8583_1993 = s
	MustRegister(s)
}
