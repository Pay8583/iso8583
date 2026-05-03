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
	MTI          string ` + "`iso8583:\"mti\"`" + `
	PAN          string ` + "`iso8583:\"2,llvar,bcd,numeric\"`" + `
	ProcCode     string ` + "`iso8583:\"3,bcd,numeric,len=6\"`" + `
	Amount       int64  ` + "`iso8583:\"4,bcd,numeric,len=12\"`" + `
	STAN         string ` + "`iso8583:\"11,bcd,numeric,len=6\"`" + `
	TerminalID   string ` + "`iso8583:\"41,ascii,len=8\"`" + `
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

	// Create go.mod for the test package, pointing to the local module.
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

	// Verify generated code compiles.
	cmd := exec.Command("go", "build", tmpDir)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build of generated code failed:\n%s", out)
	}
	t.Log("generated code compiles successfully")
}
