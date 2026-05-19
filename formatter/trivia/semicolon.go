package trivia

import "github.com/onflow/cadence/ast"

// ScanSemicolons walks the AST and checks the original source bytes after
// each statement/declaration's end position for a trailing semicolon.
// Returns a set of elements that had trailing semicolons in the source.
func ScanSemicolons(source []byte, prog *ast.Program) map[ast.Element]bool {
	result := make(map[ast.Element]bool)
	for _, decl := range prog.Declarations() {
		checkSemicolon(source, decl, result)
		decl.Walk(func(child ast.Element) {
			if child != nil {
				checkSemicolon(source, child, result)
			}
		})
	}
	return result
}

func checkSemicolon(source []byte, elem ast.Element, result map[ast.Element]bool) {
	end := elem.EndPosition(nil)
	if end.Offset < 0 || end.Offset >= len(source) {
		return
	}
	// Scan forward from end position, skipping spaces/tabs (not newlines).
	i := end.Offset + 1
	for i < len(source) && (source[i] == ' ' || source[i] == '\t') {
		i++
	}
	if i < len(source) && source[i] == ';' {
		result[elem] = true
	}
}
