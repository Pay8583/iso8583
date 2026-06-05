//go:generate iso8583gen --type AuthRequest

package main

// AuthRequest is an example struct for code generation.
// Run `go generate` or `iso8583gen --type AuthRequest message.go` to
// generate allocation-free MarshalISO8583 and UnmarshalISO8583 methods.
type AuthRequest struct {
	MTI          uint   `iso8583:"mti"`
	PAN          string `iso8583:"2,llvar,bcd,n,secure"`
	ProcCode     string `iso8583:"3,fixed=6,bcd,n"`
	Amount       int64  `iso8583:"4,fixed=12,rbcd,n"`
	STAN         string `iso8583:"11,fixed=6,bcd,n"`
	TimeLocal    string `iso8583:"12,fixed=6,bcd,n"`
	DateLocal    string `iso8583:"13,fixed=4,bcd,n"`
	Track2       string `iso8583:"35,llvar,ascii,z,secure"`
	RetRefNum    string `iso8583:"37,fixed=12,ascii,ans"`
	TerminalID   string `iso8583:"41,fixed=8,ascii,ans"`
	CurrencyCode string `iso8583:"49,fixed=3,ascii,ans"`
}
