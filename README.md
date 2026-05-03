# iso8583

Go library for packing/unpacking ISO 8583 financial transaction messages, with signing/verification, a code generator, and a CLI tool.

ISO 8583 is the wire protocol used by Visa, Mastercard, and virtually every payment network worldwide. Messages consist of an MTI (4 bytes), one or two bitmaps (8/16 bytes), and data elements (fields 2–128) in fixed or variable-length encodings (BCD, ASCII, EBCDIC, binary).

## Features

- **Pack/Unpack** — Marshal/unmarshal ISO 8583 messages from/to Go structs using struct tags
- **Version Variants** — Built-in specs for ISO 8583:1987, 1993, 2003; extensible custom specs
- **Sign/Verify** — HMAC-SHA256/512, RSA, ECDSA signing; ISO 9797-1 MAC algorithms
- **Code Generator** — `iso8583gen` generates allocation-free marshal/unmarshal methods (like easyjson)
- **CLI Tool** — Decode/encode/sign/verify raw ISO 8583 messages from the command line
- **Zero Dependencies** — Core library uses only the Go standard library

## Quick Start

```go
import "github.com/Pay8583/iso8583"

type AuthRequest struct {
    MTI          string `iso8583:"mti"`
    PAN          string `iso8583:"2,llvar,bcd,numeric"`
    Amount       int64  `iso8583:"4,bcd,numeric,len=12"`
    STAN         string `iso8583:"11,bcd,numeric,len=6"`
    TerminalID   string `iso8583:"41,ascii,len=8"`
    CurrencyCode string `iso8583:"49,ascii,len=3"`
}

spec := iso8583.MustGetSpec("1993")
msg := AuthRequest{
    MTI:          "0200",
    PAN:          "4000001234567890",
    Amount:       1000,
    STAN:         "123456",
    TerminalID:   "TERM0001",
    CurrencyCode: "840",
}
data, err := iso8583.Marshal(msg, spec)
```

## License

MIT
