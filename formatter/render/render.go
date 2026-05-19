package render

import (
	"github.com/janezpodhostnik/cadencefmt/internal/format/rewrite"
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/turbolent/prettier"
)

// Program renders an *ast.Program with interleaved comments from the CommentMap.
// source is the original input bytes (used for blank-line detection); semicolons
// is nil unless the caller wants explicit ; preserved.
func Program(prog *ast.Program, cm *trivia.CommentMap, source []byte, semicolons map[ast.Element]bool) prettier.Doc {
	r := &renderer{cm: cm, source: source, semicolons: semicolons}
	return r.program(prog)
}

func (r *renderer) program(prog *ast.Program) prettier.Doc {
	parts := prettier.Concat{}

	// Header comments
	header := r.cm.TakeHeader()
	for _, g := range header {
		parts = append(parts, renderCommentGroup(g), prettier.HardLine{})
	}
	if len(header) > 0 {
		parts = append(parts, prettier.HardLine{})
	}

	decls := prog.Declarations()
	for i, decl := range decls {
		if i > 0 {
			sep := declSeparation(decls[i-1], decl)
			for range sep {
				parts = append(parts, prettier.HardLine{})
			}
		}
		parts = append(parts, r.declaration(decl))
	}

	// Footer comments
	footer := r.cm.TakeFooter()
	if len(footer) > 0 {
		parts = append(parts, prettier.HardLine{})
	}
	for _, g := range footer {
		parts = append(parts, prettier.HardLine{}, renderCommentGroup(g))
	}

	// Trailing newline
	parts = append(parts, prettier.HardLine{})

	return parts
}

// declSeparation returns the number of HardLines to insert between
// two consecutive declarations. Imports in the same group get 1 (no blank line);
// imports in different groups or non-imports get 2 (one blank line).
func declSeparation(prev, next ast.Declaration) int {
	prevImp, prevIsImport := prev.(*ast.ImportDeclaration)
	nextImp, nextIsImport := next.(*ast.ImportDeclaration)

	if prevIsImport && nextIsImport {
		if rewrite.ImportGroupOrder(prevImp) == rewrite.ImportGroupOrder(nextImp) {
			return 1 // same import group: no blank line
		}
		return 2 // different import groups: blank line
	}

	return 2 // default: blank line between declarations
}
