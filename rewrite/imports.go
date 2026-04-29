package rewrite

import (
	"sort"

	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
)

type importsSorter struct{}

func (r *importsSorter) Name() string { return "imports" }

func (r *importsSorter) Rewrite(prog *ast.Program, _ *trivia.CommentMap) error {
	decls := prog.Declarations()

	// Collect import declarations and their indices
	var imports []*ast.ImportDeclaration
	var indices []int
	for i, d := range decls {
		if imp, ok := d.(*ast.ImportDeclaration); ok {
			imports = append(imports, imp)
			indices = append(indices, i)
		}
	}

	if len(imports) <= 1 {
		return nil
	}

	// Note: we do not deduplicate imports here because we cannot remove
	// declarations from the AST. Dedup would shrink the imports slice but
	// leave the declaration at the removed index in place, causing
	// non-idempotent output. Stable sort puts duplicates adjacent.

	// Stable sort preserves relative order of equal imports
	sort.SliceStable(imports, func(i, j int) bool {
		return importLess(imports[i], imports[j])
	})

	// Place sorted imports back at their original positions
	for i, idx := range indices {
		decls[idx] = imports[i]
	}

	return nil
}

// importGroupOrder returns the sort group for an import:
// 0 = identifier (standard), 1 = address, 2 = string.
func importGroupOrder(imp *ast.ImportDeclaration) int {
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

// importLess defines the sort order for import declarations.
func importLess(a, b *ast.ImportDeclaration) bool {
	ga, gb := importGroupOrder(a), importGroupOrder(b)
	if ga != gb {
		return ga < gb
	}

	switch la := a.Location.(type) {
	case common.IdentifierLocation:
		lb := b.Location.(common.IdentifierLocation)
		return string(la) < string(lb)
	case common.AddressLocation:
		lb := b.Location.(common.AddressLocation)
		addrA, addrB := la.Address.String(), lb.Address.String()
		if addrA != addrB {
			return addrA < addrB
		}
		return importName(a) < importName(b)
	case common.StringLocation:
		lb := b.Location.(common.StringLocation)
		return string(la) < string(lb)
	}

	return false
}

// importName returns the primary identifier name for an import.
func importName(imp *ast.ImportDeclaration) string {
	if len(imp.Imports) > 0 {
		return imp.Imports[0].Identifier.Identifier
	}
	switch l := imp.Location.(type) {
	case common.IdentifierLocation:
		return string(l)
	case common.AddressLocation:
		return l.Name
	case common.StringLocation:
		return string(l)
	}
	return ""
}

