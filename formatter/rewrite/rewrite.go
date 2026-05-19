package rewrite

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/formatter/trivia"
)

// Rewriter transforms an AST program in place. Rewriters run in a fixed
// order; changing the order may break idempotence.
type Rewriter interface {
	Name() string
	Rewrite(prog *ast.Program, cm *trivia.CommentMap) error
}

// Apply runs all rewriters in the canonical fixed order.
// If you change the pass order or add/remove passes,
// bump format.CurrentFormatVersion in options.go.
func Apply(prog *ast.Program, cm *trivia.CommentMap, sortImports bool) error {
	var rewriters []Rewriter
	if sortImports {
		rewriters = append(rewriters, &importsSorter{})
	}
	// modifiers: canonical ordering is enforced by the parser, so no rewrite needed
	// parens: conservative removal deferred to later phase
	for _, rw := range rewriters {
		if err := rw.Rewrite(prog, cm); err != nil {
			return err
		}
	}
	return nil
}
