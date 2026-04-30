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

// wrapWithAllComments wraps a node's Doc with its own comments AND drains
// comments from all descendant nodes, emitting them inline. Use this for
// nodes rendered via upstream Doc() where we don't control child rendering.
func wrapWithAllComments(elem ast.Element, doc prettier.Doc, cm *trivia.CommentMap) prettier.Doc {
	doc = wrapWithComments(elem, doc, cm)
	var extras []prettier.Doc
	drainDescendantComments(elem, cm, &extras)
	if len(extras) > 0 {
		// Interleave descendant comments into the doc.
		// These won't be perfectly positioned but they're preserved.
		parts := prettier.Concat{doc}
		for _, e := range extras {
			parts = append(parts, prettier.HardLine{}, e)
		}
		return parts
	}
	return doc
}

// walkable is anything with a Walk method (ast.Element, ParameterList, etc.)
type walkable interface {
	Walk(func(ast.Element))
}

// drainWalkable drains comments from all children of a walkable node.
func drainWalkable(w walkable, cm *trivia.CommentMap) {
	w.Walk(func(child ast.Element) {
		if child == nil {
			return
		}
		cm.Take(child)
		var discard []prettier.Doc
		drainDescendantComments(child, cm, &discard)
	})
}

// drainDescendantComments recursively removes and collects all comments
// from child nodes of elem.
func drainDescendantComments(elem ast.Element, cm *trivia.CommentMap, out *[]prettier.Doc) {
	elem.Walk(func(child ast.Element) {
		if child == nil {
			return
		}
		leading, sameLine, trailing := cm.Take(child)
		if out != nil {
			for _, g := range leading {
				*out = append(*out, renderCommentGroup(g))
			}
			if sameLine != nil {
				*out = append(*out, renderCommentGroup(sameLine))
			}
			for _, g := range trailing {
				*out = append(*out, renderCommentGroup(g))
			}
		}
		drainDescendantComments(child, cm, out)
	})
}
