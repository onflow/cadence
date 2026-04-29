package render

import (
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
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
		doc := renderDeclaration(decl, cm)
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

// declSeparation returns the number of HardLines to insert between
// two consecutive declarations. Imports in the same group get 1 (just a newline);
// imports in different groups or non-imports get 2 (blank line).
func declSeparation(prev, next ast.Declaration) int {
	prevImp, prevIsImport := prev.(*ast.ImportDeclaration)
	nextImp, nextIsImport := next.(*ast.ImportDeclaration)

	if prevIsImport && nextIsImport {
		if importGroupType(prevImp) == importGroupType(nextImp) {
			return 1 // same import group: no blank line
		}
		return 2 // different import groups: blank line
	}

	return 2 // default: blank line between declarations
}

// importGroupType returns the sort group for an import: 0=identifier, 1=address, 2=string.
func importGroupType(imp *ast.ImportDeclaration) int {
	switch imp.Location.(type) {
	case common.IdentifierLocation:
		return 0
	case common.AddressLocation:
		return 1
	case common.StringLocation:
		return 2
	default:
		return 3
	}
}
