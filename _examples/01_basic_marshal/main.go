// Example: basic marshal/unmarshal using struct tags and the reflection codec.
package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/Pay8583/iso8583"
	"github.com/Pay8583/iso8583/spec"
)

type AuthRequest struct {
	MTI          uint   `iso8583:"mti"`
	PAN          string `iso8583:"2,llvar,bcd,n"`
	ProcCode     string `iso8583:"3,fixed=6,bcd,n"`
	Amount       int64  `iso8583:"4,fixed=12,rbcd,n"`
	STAN         string `iso8583:"11,fixed=6,bcd,n"`
	TimeLocal    string `iso8583:"12,fixed=6,bcd,n"`
	DateLocal    string `iso8583:"13,fixed=4,bcd,n"`
	Track2       string `iso8583:"35,llvar,ascii,z"`
	RetRefNum    string `iso8583:"37,fixed=12,ascii,ans"`
	TerminalID   string `iso8583:"41,fixed=8,ascii,ans"`
	CurrencyCode string `iso8583:"49,fixed=3,ascii,ans"`
}

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

	data, err := iso8583.Marshal(&req, p)
	if err != nil {
		log.Fatalf("Marshal: %v", err)
	}
	fmt.Printf("Packed (%d bytes): %s\n", len(data), hex.EncodeToString(data))

	var resp AuthRequest
	if err := iso8583.Unmarshal(data, &resp, p); err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	fmt.Printf("Unpacked MTI: %#x, PAN: %s, Amount: %d, STAN: %s\n",
		resp.MTI, resp.PAN, resp.Amount, resp.STAN)

	// Security-aware export (secure fields like PAN are masked).
	export, _ := iso8583.ExportStruct(&resp, p)
	fmt.Printf("Export (PAN masked): %v\n", export)
}
