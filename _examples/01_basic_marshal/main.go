// Example: basic marshal/unmarshal using reflection codec.
package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/Pay8583/iso8583"
	"github.com/Pay8583/iso8583/spec"
)

type AuthRequest struct {
	MTI          string `iso8583:"mti"`
	PAN          string `iso8583:"2,llvar,bcd,numeric"`
	ProcCode     string `iso8583:"3,bcd,numeric,len=6"`
	Amount       int64  `iso8583:"4,bcd,numeric,len=12"`
	STAN         string `iso8583:"11,bcd,numeric,len=6"`
	TimeLocal    string `iso8583:"12,bcd,numeric,len=6"`
	DateLocal    string `iso8583:"13,bcd,numeric,len=4"`
	Track2       string `iso8583:"35,llvar,ascii"`
	RetRefNum    string `iso8583:"37,ascii,len=12"`
	TerminalID   string `iso8583:"41,ascii,len=8"`
	CurrencyCode string `iso8583:"49,ascii,len=3"`
}

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

	data, err := iso8583.Marshal(&req, s)
	if err != nil {
		log.Fatalf("Marshal: %v", err)
	}
	fmt.Printf("Packed (%d bytes): %s\n", len(data), hex.EncodeToString(data))

	var resp AuthRequest
	if err := iso8583.Unmarshal(data, &resp, s); err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	fmt.Printf("Unpacked MTI: %s, PAN: %s, Amount: %d, STAN: %s\n",
		resp.MTI, resp.PAN, resp.Amount, resp.STAN)
}
