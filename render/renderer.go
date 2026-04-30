package render

import (
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
)

// renderer holds state shared across rendering of a single program: the
// CommentMap (drained as comments are emitted), the original source bytes
// (used for blank-line detection that can't rely on AST line numbers), and
// the optional explicit-semicolon set (populated only when StripSemicolons
// is false). All render functions are methods on *renderer so this state
// doesn't need to be threaded through every call.
type renderer struct {
	cm         *trivia.CommentMap
	source     []byte
	semicolons map[ast.Element]bool
}

// hasSemicolon reports whether elem had a trailing semicolon in the source.
// Returns false when semicolons is nil (StripSemicolons mode).
func (r *renderer) hasSemicolon(elem ast.Element) bool {
	return r.semicolons[elem]
}
