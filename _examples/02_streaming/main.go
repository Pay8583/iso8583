// Example: streaming Writer/Reader for dynamic field selection.
// Use this when you don't know at compile time which fields will be present.
package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/Pay8583/iso8583"
	"github.com/Pay8583/iso8583/spec"
)

func main() {
	p := spec.MustGet("1987")

	// ── Write ─────────────────────────────────────────────────────
	var buf bytes.Buffer
	w := iso8583.NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	w.WriteString(2, "4000001234567890") // PAN
	w.WriteString(3, "301000")            // Processing Code
	w.WriteInt(4, 1000)                   // Amount
	w.WriteString(11, "123456")           // STAN
	w.WriteString(41, "TERM0001")         // Terminal ID
	w.WriteString(49, "840")              // Currency Code

	if err := w.Close(); err != nil {
		log.Fatalf("Close: %v", err)
	}

	wire := buf.Bytes()
	fmt.Printf("Packed (%d bytes): %s\n", len(wire), hex.EncodeToString(wire))

	// Raw bytes for MAC computation (before MAC field is written).
	raw := w.Bytes()
	fmt.Printf("Raw for MAC (%d bytes): %s\n", len(raw), hex.EncodeToString(raw))

	// ── Read ──────────────────────────────────────────────────────
	r := iso8583.NewReader(p, bytes.NewReader(wire))

	mti, err := r.ReadMTI()
	if err != nil {
		log.Fatalf("ReadMTI: %v", err)
	}
	fmt.Printf("MTI: %#x\n", mti)

	var pan, procCode, stan, termID, currCode string
	var amount int64

	r.ReadString(2, &pan)
	r.ReadString(3, &procCode)
	r.ReadInt(4, &amount)
	r.ReadString(11, &stan)
	r.ReadString(41, &termID)
	r.ReadString(49, &currCode)

	fmt.Printf("PAN: %s, Proc: %s, Amount: %d, STAN: %s, Term: %s, Curr: %s\n",
		pan, procCode, amount, stan, termID, currCode)

	// Security-aware export (secure fields like PAN are masked).
	export := r.Export()
	fmt.Printf("Export (secure masked): %v\n", export)
}
