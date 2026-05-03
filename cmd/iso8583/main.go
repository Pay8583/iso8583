// iso8583 is a command-line tool for decoding, encoding, signing, and verifying
// ISO 8583 financial transaction messages.
//
// Usage:
//
//	iso8583 decode --spec 1987 [--json] [--format hex] [--input file]
//	iso8583 encode --spec 1993 --fields "2=PAN,3=301000,4=1000" [--mti 0200]
//	iso8583 sign   --alg hmac-sha256 --key key.bin [--input file]
//	iso8583 verify --alg hmac-sha256 --key key.bin --signature sig.hex [--input file]
//	iso8583 specs
package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/Pay8583/iso8583"
	"github.com/Pay8583/iso8583/mac"
	"github.com/Pay8583/iso8583/padding"
	"github.com/Pay8583/iso8583/spec"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "decode":
		decodeCmd(os.Args[2:])
	case "encode":
		encodeCmd(os.Args[2:])
	case "sign":
		signCmd(os.Args[2:])
	case "verify":
		verifyCmd(os.Args[2:])
	case "specs":
		specsCmd()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage:
  iso8583 decode --spec <name> [--json] [--format hex|base64|raw] [--input <file>]
  iso8583 encode --spec <name> --fields "2=val,3=val" [--mti 0200] [--format hex]
  iso8583 sign   --alg <algo> --key <file> [--input <file>]
  iso8583 verify --alg <algo> --key <file> --signature <file> [--input <file>]
  iso8583 specs
`)
}

func readInput(input string) ([]byte, error) {
	if input == "" || input == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(input)
}

func decodeInput(raw []byte, format string) ([]byte, error) {
	switch format {
	case "hex":
		return hex.DecodeString(strings.TrimSpace(string(raw)))
	case "base64":
		return base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	case "raw":
		return raw, nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// ── decode ───────────────────────────────────────────────────────────────────────

func decodeCmd(args []string) {
	var specName, input, format string
	var useJSON bool
	fs := newFlagSet("decode")
	fs.StringVar(&specName, "spec", "1987", "spec name from registry")
	fs.StringVar(&input, "input", "", "input file (default stdin)")
	fs.StringVar(&format, "format", "hex", "input format: hex, base64, raw")
	fs.BoolVar(&useJSON, "json", false, "output as JSON")
	fs.Parse(args)

	raw, err := readInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}
	data, err := decodeInput(raw, format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		os.Exit(1)
	}

	s, err := spec.Get(specName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spec: %v\navailable: %s\n", err, strings.Join(spec.List(), ", "))
		os.Exit(1)
	}

	msg := iso8583.NewMessage(s, "")
	if err := msg.Unpack(data); err != nil {
		fmt.Fprintf(os.Stderr, "unpack: %v\n", err)
		os.Exit(1)
	}

	if useJSON {
		outputJSON(msg)
	} else {
		outputTable(msg)
	}
}

func outputTable(msg *iso8583.Message) {
	fmt.Printf("MTI:    %s\n", msg.MTI)
	fmt.Printf("Spec:   %s (%s)\n", msg.Spec.Name, msg.Spec.Version)
	fmt.Printf("Fields: %d\n\n", len(msg.Fields))

	// Column widths.
	fmt.Printf("%-6s %-32s %s\n", "Field", "Name", "Value")
	fmt.Printf("%-6s %-32s %s\n", "-----", "----", "-----")

	numbers := make([]int, 0, len(msg.Fields))
	for n := range msg.Fields {
		numbers = append(numbers, n)
	}
	sort.Ints(numbers)

	names := msg.FieldNames()
	for _, n := range numbers {
		name := names[n]
		val := msg.Fields[n]
		// Truncate long values for display.
		display := val
		if len(val) > 60 {
			display = val[:57] + "..."
		}
		fmt.Printf("%-6d %-32s %s\n", n, name, display)
	}
}

func outputJSON(msg *iso8583.Message) {
	type field struct {
		Index int    `json:"index"`
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	out := struct {
		MTI    string  `json:"mti"`
		Spec   string  `json:"spec"`
		Fields []field `json:"fields"`
	}{
		MTI:  msg.MTI,
		Spec: msg.Spec.Name,
	}

	names := msg.FieldNames()
	numbers := make([]int, 0, len(msg.Fields))
	for n := range msg.Fields {
		numbers = append(numbers, n)
	}
	sort.Ints(numbers)
	for _, n := range numbers {
		out.Fields = append(out.Fields, field{Index: n, Name: names[n], Value: msg.Fields[n]})
	}
	if out.Fields == nil {
		out.Fields = []field{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

// ── encode ───────────────────────────────────────────────────────────────────────

func encodeCmd(args []string) {
	var specName, fields, mti, format string
	fs := newFlagSet("encode")
	fs.StringVar(&specName, "spec", "1987", "spec name")
	fs.StringVar(&fields, "fields", "", "comma-separated field=value pairs (e.g. \"2=400...,3=301000\")")
	fs.StringVar(&mti, "mti", "0200", "message type indicator")
	fs.StringVar(&format, "format", "hex", "output format: hex, base64, raw")
	fs.Parse(args)

	s, err := spec.Get(specName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spec: %v\n", err)
		os.Exit(1)
	}

	msg := iso8583.NewMessage(s, mti)
	for _, pair := range strings.Split(fields, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			fmt.Fprintf(os.Stderr, "bad field format: %q (expected field=value)\n", pair)
			os.Exit(1)
		}
		n := 0
		if _, err := fmt.Sscanf(kv[0], "%d", &n); err != nil || n < 2 {
			fmt.Fprintf(os.Stderr, "bad field number: %q\n", kv[0])
			os.Exit(1)
		}
		msg.Set(n, kv[1])
	}

	data, err := msg.Pack()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack: %v\n", err)
		os.Exit(1)
	}

	switch format {
	case "hex":
		fmt.Println(hex.EncodeToString(data))
	case "base64":
		fmt.Println(base64.StdEncoding.EncodeToString(data))
	case "raw":
		os.Stdout.Write(data)
	}
}

// ── sign ────────────────────────────────────────────────────────────────────────

func signCmd(args []string) {
	var alg, keyFile, input string
	fs := newFlagSet("sign")
	fs.StringVar(&alg, "alg", "hmac-sha256", "algorithm: hmac-sha256, hmac-sha512, iso9797-algo1-aes, iso9797-algo3")
	fs.StringVar(&keyFile, "key", "", "key file (required)")
	fs.StringVar(&input, "input", "", "input file (default stdin)")
	fs.Parse(args)

	if keyFile == "" {
		fmt.Fprintln(os.Stderr, "sign: --key is required")
		os.Exit(2)
	}

	data, err := readInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}
	key, err := os.ReadFile(keyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "key: %v\n", err)
		os.Exit(1)
	}

	signer, err := makeSigner(alg, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "signer: %v\n", err)
		os.Exit(1)
	}

	sig, err := iso8583.SignMessage(data, signer, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(hex.EncodeToString(sig))
}

// ── verify ──────────────────────────────────────────────────────────────────────

func verifyCmd(args []string) {
	var alg, keyFile, input, sigFile string
	fs := newFlagSet("verify")
	fs.StringVar(&alg, "alg", "hmac-sha256", "algorithm")
	fs.StringVar(&keyFile, "key", "", "key file (required)")
	fs.StringVar(&input, "input", "", "input file (default stdin)")
	fs.StringVar(&sigFile, "signature", "", "signature file (hex-encoded, required)")
	fs.Parse(args)

	if keyFile == "" || sigFile == "" {
		fmt.Fprintln(os.Stderr, "verify: --key and --signature are required")
		os.Exit(2)
	}

	data, err := readInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}
	key, err := os.ReadFile(keyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "key: %v\n", err)
		os.Exit(1)
	}
	sigHex, err := os.ReadFile(sigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "signature: %v\n", err)
		os.Exit(1)
	}
	sig, err := hex.DecodeString(strings.TrimSpace(string(sigHex)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "signature: invalid hex: %v\n", err)
		os.Exit(1)
	}

	signer, err := makeSigner(alg, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "signer: %v\n", err)
		os.Exit(1)
	}

	if err := iso8583.VerifyMessage(data, signer, sig, nil); err != nil {
		fmt.Fprintf(os.Stderr, "verify: FAIL — %v\n", err)
		os.Exit(1)
	}
	fmt.Println("verify: OK")
}

// ── specs ───────────────────────────────────────────────────────────────────────

func specsCmd() {
	for _, name := range spec.List() {
		s, err := spec.Get(name)
		if err != nil {
			continue
		}
		n := 0
		for _, fs := range s.Fields {
			if fs != nil && !fs.Optional {
				n++
			}
		}
		fmt.Printf("%-8s version=%-6s fields=%d (required=%d) secondary=%v\n  %s\n",
			s.Name, s.Version, len(s.Fields), n, s.HasSecondaryBitmap, s.Description)
	}
}

// ── signer factory ──────────────────────────────────────────────────────────────

func makeSigner(alg string, key []byte) (iso8583.Signer, error) {
	switch alg {
	case "hmac-sha256":
		return &mac.HMACSigner{Key: key, Hash: "sha256"}, nil
	case "hmac-sha512":
		return &mac.HMACSigner{Key: key, Hash: "sha512"}, nil
	case "iso9797-algo1-aes":
		return &mac.Algo1Signer{Key: key, Cipher: "aes", Padder: padding.Get("iso9797-2")}, nil
	case "iso9797-algo1-des":
		return &mac.Algo1Signer{Key: key, Cipher: "des", Padder: padding.Get("iso9797-2")}, nil
	case "iso9797-algo3":
		if len(key) != 16 {
			return nil, fmt.Errorf("iso9797-algo3 requires 16-byte key (key1||key2)")
		}
		return &mac.Algo3Signer{Key1: key[:8], Key2: key[8:], Padder: padding.Get("iso9797-2")}, nil
	default:
		return nil, fmt.Errorf("unknown algorithm: %s", alg)
	}
}

// ── flag helpers ────────────────────────────────────────────────────────────────

// newFlagSet creates a flag.FlagSet that continues on error so we can print our own
// usage messages.
func newFlagSet(name string) *flagSet { return &flagSet{name: name} }

type flagSet struct {
	name string
	fs   flagSetImpl
}

func (f *flagSet) StringVar(p *string, name, value, usage string) {
	if f.fs.flags == nil {
		f.fs.flags = make(map[string]*string)
	}
	f.fs.flags[name] = p
	*p = value
}

func (f *flagSet) BoolVar(p *bool, name string, value bool, usage string) {
	if f.fs.bools == nil {
		f.fs.bools = make(map[string]*bool)
	}
	f.fs.bools[name] = p
	*p = value
}

func (f *flagSet) Parse(args []string) {
	// Simple flag parser: --key value or --flag
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			continue
		}
		name := a[2:]
		if bp, ok := f.fs.bools[name]; ok {
			*bp = true
			continue
		}
		if i+1 < len(args) {
			if sp, ok := f.fs.flags[name]; ok {
				*sp = args[i+1]
				i++
			}
		}
	}
}

type flagSetImpl struct {
	flags map[string]*string
	bools map[string]*bool
}
