package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func moduleRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("cannot find module root")
		}
		dir = parent
	}
}

func TestGenerator_Compiles(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "types.go")
	outputFile := filepath.Join(tmpDir, "types_iso8583.go")
	mainFile := filepath.Join(tmpDir, "main.go")

	structSrc := `package main

type AuthRequest struct {
	MTI        uint   ` + "`iso8583:\"mti\"`" + `
	PAN        string ` + "`iso8583:\"2,llvar,bcd,n,secure\"`" + `
	ProcCode   string ` + "`iso8583:\"3,fixed=6,bcd,n\"`" + `
	Amount     int64  ` + "`iso8583:\"4,fixed=12,rbcd,n\"`" + `
	STAN       string ` + "`iso8583:\"11,fixed=6,bcd,n\"`" + `
	TerminalID string ` + "`iso8583:\"41,fixed=8,ascii,ans\"`" + `
}
`
	mainSrc := `package main
func main() {}
`
	if err := os.WriteFile(inputFile, []byte(structSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainFile, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}

	gomod := `module testgen

go 1.26

require github.com/Pay8583/iso8583 v0.0.0
replace github.com/Pay8583/iso8583 => ` + moduleRoot() + `
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatal(err)
	}

	// Run generator.
	if err := Generate(inputFile, outputFile, []string{"AuthRequest"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Read generated code to verify it contains expected elements.
	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Read generated: %v", err)
	}
	genStr := string(generated)

	// Verify key patterns in generated code.
	checks := []string{
		"func (m *AuthRequest) MarshalISO8583(p *spec.Protocol)",
		"func (m *AuthRequest) UnmarshalISO8583(data []byte, p *spec.Protocol)",
		"iso8583.NewWriter(p, &buf)",
		"iso8583.NewReader(p, bytes.NewReader(data))",
		"w.WriteMTI(uint(m.MTI))",
		"w.WriteString(2, m.PAN)",
		"w.WriteString(3, m.ProcCode)",
		"w.WriteInt(4, int64(m.Amount))",
		"w.WriteString(11, m.STAN)",
		"w.WriteString(41, m.TerminalID)",
	}
	for _, c := range checks {
		if !containsString(genStr, c) {
			t.Errorf("generated code missing: %s", c)
		}
	}

	// Verify generated code compiles.
	cmd := exec.Command("go", "build", tmpDir)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build of generated code failed:\n%s\n\nGenerated code:\n%s", out, genStr)
	}
	t.Log("generated code compiles successfully")
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
