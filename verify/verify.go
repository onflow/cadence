package verify

import (
	"fmt"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/parser"
)

// RoundTrip parses both original and formatted source and structurally
// compares the ASTs. Returns nil if they are equivalent (ignoring positions
// and whitespace). Returns an error describing the first difference found.
func RoundTrip(original, formatted []byte) error {
	origProg, err := parser.ParseProgram(nil, original, parser.Config{})
	if err != nil {
		return fmt.Errorf("original parse error: %w", err)
	}

	fmtProg, err := parser.ParseProgram(nil, formatted, parser.Config{})
	if err != nil {
		return fmt.Errorf("formatted parse error: %w", err)
	}

	return comparePrograms(origProg, fmtProg)
}

func comparePrograms(a, b *ast.Program) error {
	aDecls := a.Declarations()
	bDecls := b.Declarations()

	if len(aDecls) != len(bDecls) {
		return fmt.Errorf("declaration count mismatch: original=%d formatted=%d",
			len(aDecls), len(bDecls))
	}

	// Split imports from non-imports — the formatter may reorder imports
	aImports, aNonImports := splitDecls(aDecls)
	bImports, bNonImports := splitDecls(bDecls)

	// Imports: compare as multiset (same imports, any order)
	if err := compareImportSets(aImports, bImports); err != nil {
		return err
	}

	// Non-imports: compare in order
	if len(aNonImports) != len(bNonImports) {
		return fmt.Errorf("non-import declaration count mismatch: original=%d formatted=%d",
			len(aNonImports), len(bNonImports))
	}
	for i := range aNonImports {
		if err := compareElements(aNonImports[i], bNonImports[i], fmt.Sprintf("decl[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

func splitDecls(decls []ast.Declaration) (imports, other []ast.Declaration) {
	for _, d := range decls {
		if _, ok := d.(*ast.ImportDeclaration); ok {
			imports = append(imports, d)
		} else {
			other = append(other, d)
		}
	}
	return
}

func compareImportSets(a, b []ast.Declaration) error {
	if len(a) != len(b) {
		return fmt.Errorf("import count mismatch: original=%d formatted=%d", len(a), len(b))
	}

	aSet := make(map[string]bool)
	for _, d := range a {
		aSet[fmt.Sprintf("%s", d)] = true
	}
	for _, d := range b {
		key := fmt.Sprintf("%s", d)
		if !aSet[key] {
			return fmt.Errorf("formatted has extra import: %s", key)
		}
	}
	return nil
}

func compareElements(a, b ast.Element, path string) error {
	if a == nil && b == nil {
		return nil
	}
	if a == nil || b == nil {
		return fmt.Errorf("%s: nil mismatch (original=%v formatted=%v)", path, a, b)
	}

	// Compare element types
	if a.ElementType() != b.ElementType() {
		return fmt.Errorf("%s: element type mismatch (original=%s formatted=%s)",
			path, a.ElementType(), b.ElementType())
	}

	// Compare string representation (captures identifiers, operators, etc.)
	// This is a pragmatic comparison — it catches semantic differences while
	// ignoring whitespace/position changes.
	aStr := fmt.Sprintf("%s", a)
	bStr := fmt.Sprintf("%s", b)
	if aStr != bStr {
		return fmt.Errorf("%s: content mismatch\n  original:  %s\n  formatted: %s",
			path, truncate(aStr, 200), truncate(bStr, 200))
	}

	// Recursively compare children
	aChildren := collectChildren(a)
	bChildren := collectChildren(b)

	if len(aChildren) != len(bChildren) {
		return fmt.Errorf("%s: child count mismatch (original=%d formatted=%d)",
			path, len(aChildren), len(bChildren))
	}

	for i := range aChildren {
		childPath := fmt.Sprintf("%s.child[%d]", path, i)
		if err := compareElements(aChildren[i], bChildren[i], childPath); err != nil {
			return err
		}
	}

	return nil
}

func collectChildren(elem ast.Element) []ast.Element {
	var children []ast.Element
	elem.Walk(func(child ast.Element) {
		if child != nil {
			children = append(children, child)
		}
	})
	return children
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
