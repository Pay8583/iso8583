// iso8583gen generates fast, allocation-free MarshalISO8583 and UnmarshalISO8583
// methods for Go structs with iso8583 tags. It is similar in spirit to easyjson
// or ffjson but specialized for ISO 8583 message encoding.
//
// Usage:
//
//	iso8583gen --type AuthRequest --type AuthResponse [--output file] <input.go>
//
// Or via go:generate:
//
//	//go:generate iso8583gen --type AuthRequest
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	types := &typeList{}
	flag.Var(types, "type", "struct type name to generate methods for (may be repeated)")
	output := flag.String("output", "", "output file (default: <input>_iso8583.go)")
	flag.Parse()

	if len(types.types) == 0 {
		fmt.Fprintln(os.Stderr, "iso8583gen: at least one --type is required")
		flag.Usage()
		os.Exit(2)
	}
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "iso8583gen: input file required")
		flag.Usage()
		os.Exit(2)
	}

	inputFile := flag.Arg(0)
	outFile := *output
	if outFile == "" {
		outFile = inputFile[:len(inputFile)-3] + "_iso8583.go"
	}

	if err := Generate(inputFile, outFile, types.types); err != nil {
		fmt.Fprintf(os.Stderr, "iso8583gen: %v\n", err)
		os.Exit(1)
	}
}

type typeList struct {
	types []string
}

func (t *typeList) String() string { return fmt.Sprint(t.types) }

func (t *typeList) Set(v string) error {
	if v == "" {
		return fmt.Errorf("empty type name")
	}
	t.types = append(t.types, v)
	return nil
}
