package trivia

import (
	"testing"

	"github.com/onflow/cadence/parser"
)

func TestScanSemicolons_Found(t *testing.T) {
	source := []byte("access(all) let x: Int = 1;\n")
	prog, err := parser.ParseProgram(nil, source, parser.Config{})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	result := ScanSemicolons(source, prog)
	if len(result) == 0 {
		t.Fatal("expected to find semicolons")
	}
}

func TestScanSemicolons_None(t *testing.T) {
	source := []byte("access(all) let x: Int = 1\n")
	prog, err := parser.ParseProgram(nil, source, parser.Config{})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	result := ScanSemicolons(source, prog)
	// The map may have entries but none should mark a real semicolon
	for elem, has := range result {
		if has {
			t.Fatalf("unexpected semicolon on element at %v", elem.StartPosition())
		}
	}
}

func TestScanSemicolons_InsideString(t *testing.T) {
	source := []byte(`access(all) let x: String = "hello;"` + "\n")
	prog, err := parser.ParseProgram(nil, source, parser.Config{})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	result := ScanSemicolons(source, prog)
	// The semicolon is inside a string — it should not be detected as a
	// trailing semicolon on any AST node. The only declaration is the
	// variable, which ends with the closing quote.
	decls := prog.Declarations()
	if len(decls) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(decls))
	}
	if result[decls[0]] {
		t.Error("semicolon inside string should not be detected")
	}
}
