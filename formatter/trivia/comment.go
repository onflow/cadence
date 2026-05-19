package trivia

import "github.com/onflow/cadence/ast"

// Kind classifies a comment token.
type Kind int

const (
	KindLine     Kind = iota // //
	KindBlock                // /* */
	KindDocLine              // ///
	KindDocBlock             // /** */
)

func (k Kind) String() string {
	switch k {
	case KindLine:
		return "Line"
	case KindBlock:
		return "Block"
	case KindDocLine:
		return "DocLine"
	case KindDocBlock:
		return "DocBlock"
	default:
		return "Unknown"
	}
}

// Comment is a single comment token extracted from source bytes.
// Text includes delimiters (e.g. "// foo" or "/* bar */").
type Comment struct {
	Kind  Kind
	Start ast.Position
	End   ast.Position // position of last byte of the comment
	Text  string
}

// CommentGroup is a sequence of adjacent comments separated only by
// whitespace (no blank lines). A blank line starts a new group.
type CommentGroup struct {
	Comments []Comment
}

// StartPos returns the position of the first byte of the group.
func (g *CommentGroup) StartPos() ast.Position {
	return g.Comments[0].Start
}

// EndPos returns the position of the last byte of the group.
func (g *CommentGroup) EndPos() ast.Position {
	return g.Comments[len(g.Comments)-1].End
}
