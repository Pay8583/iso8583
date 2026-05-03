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

### Library

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

s := spec.MustGet("1993")
req := AuthRequest{
    MTI: "0200", PAN: "4000001234567890",
    Amount: 1000, STAN: "123456",
    TerminalID: "TERM0001", CurrencyCode: "840",
}
data, err := iso8583.Marshal(&req, s)
```

### Code Generator

```go
//go:generate iso8583gen --type AuthRequest

type AuthRequest struct { ... }
```

Generates `MarshalISO8583(*spec.Spec) ([]byte, error)` and `UnmarshalISO8583([]byte, *spec.Spec) error` methods.

### CLI

```sh
# Decode a hex-encoded ISO 8583 message
iso8583 decode --spec 1987 --format hex --input message.hex

# Decode with JSON output
iso8583 decode --spec 1993 --json < message.hex

# Encode field=value pairs
iso8583 encode --spec 1987 --mti 0200 \
  --fields "2=4000001234567890,3=301000,4=000000001000,11=123456,41=TERM0001,49=840"

# Sign a message
iso8583 sign --alg hmac-sha256 --key key.bin --input message.bin

# Verify a signature
iso8583 verify --alg hmac-sha256 --key key.bin --signature sig.hex --input message.bin

# List available specs
iso8583 specs
```

## Supported Encodings

| Encoding  | Description |
|-----------|-------------|
| `bcd`     | Packed BCD (two digits per byte, left-pad odd-length with zero nibble) |
| `ascii`   | ASCII passthrough |
| `ebcdic`  | EBCDIC code page 037 (US/Canada) |
| `binary`  | Raw byte passthrough |

## Supported Padding

| Padding      | Description |
|-------------|-------------|
| `pkcs7`     | PKCS#7 (byte-count padding) |
| `iso9797-1` | ISO 9797-1 Method 1 (zero padding) |
| `iso9797-2` | ISO 9797-1 Method 2 (0x80 + zeros) |
| `iso10126`  | ISO 10126 (random bytes + count) |
| `zero`      | Simple zero-byte padding |

## Supported Signing

| Algorithm           | Description |
|---------------------|-------------|
| `hmac-sha256`       | HMAC with SHA-256 |
| `hmac-sha512`       | HMAC with SHA-512 |
| `rsa-sha256`        | RSA PKCS#1 v1.5 with SHA-256 |
| `ecdsa-sha256`      | ECDSA P-256 with SHA-256 |
| `iso9797-algo1-aes` | ISO 9797-1 Algorithm 1 (CBC-MAC) with AES |
| `iso9797-algo1-des` | ISO 9797-1 Algorithm 1 (CBC-MAC) with DES |
| `iso9797-algo3`     | ISO 9797-1 Algorithm 3 (Retail MAC / ANSI X9.19) |

## License

MIT
