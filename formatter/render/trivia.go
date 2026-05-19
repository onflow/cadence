package render

import (
	"strings"

	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/turbolent/prettier"
)

// wrapComments wraps an element's Doc with its leading, same-line, and
// trailing comments from the CommentMap. Comments are removed from the map
// via Take() so each comment is emitted exactly once.
func (r *renderer) wrapComments(elem ast.Element, doc prettier.Doc) prettier.Doc {
	leading, sameLine, trailing := r.cm.Take(elem)

	if len(leading) == 0 && sameLine == nil && len(trailing) == 0 {
		return doc
	}

	parts := prettier.Concat{}

	for _, g := range leading {
		parts = append(parts, renderCommentGroup(g), prettier.HardLine{})
	}

	parts = append(parts, doc)

	if sameLine != nil {
		parts = append(parts, prettier.Text("  "), renderCommentGroup(sameLine))
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

// wrapAllComments wraps a node's Doc with its own comments AND drains
// comments from all descendant nodes, emitting them inline. Use this for
// nodes rendered via upstream Doc() where we don't control child rendering.
func (r *renderer) wrapAllComments(elem ast.Element, doc prettier.Doc) prettier.Doc {
	doc = r.wrapComments(elem, doc)
	var extras []prettier.Doc
	r.drainDescendants(elem, &extras)
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

// drainWalk drains comments from all children of a node, given its Walk
// method as a bound function value. Used for ast.ParameterList, ast.Conditions,
// and other non-Element types that can't be passed to drainDescendants directly.
func (r *renderer) drainWalk(walk func(func(ast.Element))) {
	walk(func(child ast.Element) {
		if child == nil {
			return
		}
		r.cm.Take(child)
		var discard []prettier.Doc
		r.drainDescendants(child, &discard)
	})
}

// drainDescendants recursively removes and collects all comments from child
// nodes of elem. If out is nil the comments are still drained from the map
// but discarded.
func (r *renderer) drainDescendants(elem ast.Element, out *[]prettier.Doc) {
	elem.Walk(func(child ast.Element) {
		if child == nil {
			return
		}
		leading, sameLine, trailing := r.cm.Take(child)
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
		r.drainDescendants(child, out)
	})
}
