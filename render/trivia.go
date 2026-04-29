package render

import (
	"strings"

	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/turbolent/prettier"
)

// wrapWithComments wraps an element's Doc with its leading, same-line, and
// trailing comments from the CommentMap. Comments are removed from the map
// via Take() so each comment is emitted exactly once.
func wrapWithComments(elem ast.Element, doc prettier.Doc, cm *trivia.CommentMap) prettier.Doc {
	leading, sameLine, trailing := cm.Take(elem)

	if len(leading) == 0 && sameLine == nil && len(trailing) == 0 {
		return doc
	}

	parts := prettier.Concat{}

	for _, g := range leading {
		parts = append(parts, renderCommentGroup(g), prettier.HardLine{})
	}

	parts = append(parts, doc)

	if sameLine != nil {
		parts = append(parts, prettier.Text("  "), renderCommentGroupInline(sameLine))
	}

	for _, g := range trailing {
		parts = append(parts, prettier.HardLine{}, renderCommentGroup(g))
	}

	return parts
}

// renderCommentGroup renders a group of comments, each on its own line.
func renderCommentGroup(g *trivia.CommentGroup) prettier.Doc {
	if len(g.Comments) == 1 {
		return renderComment(g.Comments[0])
	}

	parts := prettier.Concat{}
	for i, c := range g.Comments {
		if i > 0 {
			parts = append(parts, prettier.HardLine{})
		}
		parts = append(parts, renderComment(c))
	}
	return parts
}

// renderCommentGroupInline renders a comment group for same-line placement
// (no leading HardLine).
func renderCommentGroupInline(g *trivia.CommentGroup) prettier.Doc {
	return renderCommentGroup(g)
}

// renderComment renders a single comment. Line comments have trailing
// whitespace trimmed.
func renderComment(c trivia.Comment) prettier.Doc {
	text := c.Text
	switch c.Kind {
	case trivia.KindLine, trivia.KindDocLine:
		text = strings.TrimRight(text, " \t")
	}
	return prettier.Text(text)
}
