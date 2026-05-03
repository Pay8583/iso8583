//go:generate iso8583gen --type AuthRequest

package main

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
