// Example: using the code-generated MarshalISO8583/UnmarshalISO8583 methods.
//
// iso8583.Marshal checks if the value implements the Marshaler interface
// (by having a MarshalISO8583 method). If the code generator has been run,
// it uses the generated method automatically, avoiding reflection.
// Otherwise it falls back to tag-based reflection.
package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/Pay8583/iso8583"
	"github.com/Pay8583/iso8583/spec"
)

func main() {
	p := spec.MustGet("1987")

	req := AuthRequest{
		MTI:          0x0200,
		PAN:          "4000001234567890",
		ProcCode:     "301000",
		Amount:       1000,
		STAN:         "123456",
		TimeLocal:    "143015",
		DateLocal:    "0503",
		Track2:       "4000001234567890=250510100000000",
		RetRefNum:    "123456789012",
		TerminalID:   "TERM0001",
		CurrencyCode: "840",
	}

	// iso8583.Marshal detects the generated MarshalISO8583 method and uses it.
	data, err := iso8583.Marshal(&req, p)
	if err != nil {
		log.Fatalf("Marshal: %v", err)
	}
	fmt.Printf("Packed (%d bytes): %s\n", len(data), hex.EncodeToString(data))

	// iso8583.Unmarshal detects the generated UnmarshalISO8583 method.
	var resp AuthRequest
	if err := iso8583.Unmarshal(data, &resp, p); err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	fmt.Printf("MTI: %#x, PAN: %s, Amount: %d, STAN: %s\n",
		resp.MTI, resp.PAN, resp.Amount, resp.STAN)
}
