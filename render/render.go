package render

import (
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/turbolent/prettier"
)

// Program renders an *ast.Program with interleaved comments from the CommentMap.
func Program(prog *ast.Program, cm *trivia.CommentMap, lineWidth int, indent string) prettier.Doc {
	parts := prettier.Concat{}

	// Header comments
	header := cm.TakeHeader()
	for _, g := range header {
		parts = append(parts, renderCommentGroup(g), prettier.HardLine{})
	}
	if len(header) > 0 {
		// Blank line between header and first declaration
		parts = append(parts, prettier.HardLine{})
	}

	decls := prog.Declarations()
	for i, decl := range decls {
		if i > 0 {
			// Blank line between declarations
			parts = append(parts, prettier.HardLine{}, prettier.HardLine{})
		}
		doc := decl.Doc()
		doc = wrapWithComments(decl, doc, cm)
		parts = append(parts, doc)
	}

	// Footer comments
	footer := cm.TakeFooter()
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
