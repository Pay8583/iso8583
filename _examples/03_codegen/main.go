// Example: using the code-generated MarshalISO8583/UnmarshalISO8583 methods.
package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/Pay8583/iso8583/spec"
)

func main() {
	s := spec.MustGet("1987")

	req := AuthRequest{
		MTI:          "0200",
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

	data, err := req.MarshalISO8583(s)
	if err != nil {
		log.Fatalf("Marshal: %v", err)
	}
	fmt.Printf("Packed (%d bytes): %s\n", len(data), hex.EncodeToString(data))

	var resp AuthRequest
	if err := resp.UnmarshalISO8583(data, s); err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	fmt.Printf("MTI: %s, PAN: %s, Amount: %d, STAN: %s\n",
		resp.MTI, resp.PAN, resp.Amount, resp.STAN)
}
