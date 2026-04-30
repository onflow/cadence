package render

import (
	"strings"

	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/turbolent/prettier"
)

// expression dispatches to custom renderers for expression types that
// need fixes (invocations with displaced comments, casts with missing indent),
// otherwise falls back to the upstream Doc() with full comment draining.
func (r *renderer) expression(expr ast.Expression) prettier.Doc {
	switch e := expr.(type) {
	case *ast.InvocationExpression:
		return r.wrapComments(e, r.invocationExpression(e))
	case *ast.StringTemplateExpression:
		return r.wrapComments(e, r.stringTemplate(e))
	}
	return r.wrapAllComments(expr, expr.Doc())
}

// argumentDoc renders an invocation argument using our expression renderer
// for the value, so custom expression renderers (string templates, invocations,
// casts) are applied. Mirrors upstream Argument.Doc() structure.
func (r *renderer) argumentDoc(arg *ast.Argument) prettier.Doc {
	exprDoc := r.expression(arg.Expression)
	if arg.Label == "" {
		return exprDoc
	}
	return prettier.Concat{
		prettier.Text(arg.Label + ": "),
		exprDoc,
	}
}

// invocationArg holds a rendered argument and any associated comments that
// must be placed relative to the comma separator (same-line comments go
// after the comma on the same line, not before it).
type invocationArg struct {
	doc      prettier.Doc           // argument rendering (label: expr)
	leading  []*trivia.CommentGroup // comments before the argument
	sameLine *trivia.CommentGroup   // same-line comment (after arg, before next)
	trailing []*trivia.CommentGroup // comments after the argument
	extras   []prettier.Doc         // drained descendant comment docs
}

// invocationExpression renders a function call with comments preserved
// inside the argument list. Without this, wrapAllComments + upstream Doc()
// displaces argument comments outside the closing paren.
func (r *renderer) invocationExpression(e *ast.InvocationExpression) prettier.Doc {
	parts := prettier.Concat{}

	// Take comments from the invoked expression separately. Trailing comments
	// sit between the function name and the opening paren.
	leading, sameLine, trailing := r.cm.Take(e.InvokedExpression)
	invokedDoc := r.expression(e.InvokedExpression)

	// Re-apply leading and same-line to the invoked expression.
	if len(leading) > 0 || sameLine != nil {
		wrapped := prettier.Concat{}
		for _, g := range leading {
			wrapped = append(wrapped, renderCommentGroup(g), prettier.HardLine{})
		}
		wrapped = append(wrapped, invokedDoc)
		if sameLine != nil {
			wrapped = append(wrapped, prettier.Text("  "), renderCommentGroup(sameLine))
		}
		invokedDoc = wrapped
	}
	parts = append(parts, invokedDoc)

	// Type arguments
	if len(e.TypeArguments) > 0 {
		typeArgDocs := make([]prettier.Doc, len(e.TypeArguments))
		for i, ta := range e.TypeArguments {
			typeArgDocs[i] = r.wrapAllComments(ta, ta.Doc())
		}
		parts = append(parts,
			prettier.Wrap(
				prettier.Text("<"),
				prettier.Join(
					prettier.Concat{prettier.Text(","), prettier.Line{}},
					typeArgDocs...,
				),
				prettier.Text(">"),
				prettier.SoftLine{},
			),
		)
	}

	// No arguments
	if len(e.Arguments) == 0 {
		parts = append(parts, prettier.Text("()"))
		for _, g := range trailing {
			parts = append(parts, prettier.HardLine{}, renderCommentGroup(g))
		}
		return parts
	}

	// Collect argument docs with their comments
	args := make([]invocationArg, len(e.Arguments))
	hasComments := len(trailing) > 0
	for i, arg := range e.Arguments {
		// Render the argument using our expression renderer so custom expression
		// renderers (e.g., string templates) are applied to argument values.
		a := invocationArg{doc: r.argumentDoc(arg)}

		// Collect comments from the Argument element and its Expression.
		argLeading, argSameLine, argTrailing := r.cm.Take(arg)
		exprLeading, exprSameLine, exprTrailing := r.cm.Take(arg.Expression)

		a.leading = append(argLeading, exprLeading...)
		a.trailing = append(argTrailing, exprTrailing...)
		// Same-line: prefer argument-level (closer to the text)
		a.sameLine = argSameLine
		if a.sameLine == nil {
			a.sameLine = exprSameLine
		}

		// Drain deeper descendants
		var extras []prettier.Doc
		r.drainDescendants(arg, &extras)
		// Convert extras to trailing comment groups (render as-is)
		if len(extras) > 0 {
			hasComments = true
		}

		if len(a.leading) > 0 || a.sameLine != nil || len(a.trailing) > 0 || len(extras) > 0 {
			hasComments = true
		}
		// Store extras as additional trailing docs
		a.extras = extras
		args[i] = a
	}

	if hasComments {
		// Comments force arguments to break across lines.
		inner := prettier.Concat{}
		for _, g := range trailing {
			inner = append(inner, renderCommentGroup(g), prettier.HardLine{})
		}
		for i, a := range args {
			if i > 0 {
				inner = append(inner, prettier.Text(","))
				// Previous arg's same-line comment goes after the comma
				if args[i-1].sameLine != nil {
					inner = append(inner, prettier.Text("  "), renderCommentGroup(args[i-1].sameLine))
				}
				inner = append(inner, prettier.HardLine{})
				// Previous arg's trailing comments
				for _, g := range args[i-1].trailing {
					inner = append(inner, renderCommentGroup(g), prettier.HardLine{})
				}
				for _, e := range args[i-1].extras {
					inner = append(inner, e, prettier.HardLine{})
				}
			}
			// Leading comments for this arg
			for _, g := range a.leading {
				inner = append(inner, renderCommentGroup(g), prettier.HardLine{})
			}
			inner = append(inner, a.doc)
		}
		// Handle last arg's same-line and trailing
		lastArg := args[len(args)-1]
		if lastArg.sameLine != nil {
			inner = append(inner, prettier.Text("  "), renderCommentGroup(lastArg.sameLine))
		}
		for _, g := range lastArg.trailing {
			inner = append(inner, prettier.HardLine{}, renderCommentGroup(g))
		}
		for _, e := range lastArg.extras {
			inner = append(inner, prettier.HardLine{}, e)
		}

		parts = append(parts,
			prettier.Text("("),
			prettier.Indent{Doc: prettier.Concat{
				prettier.HardLine{},
				inner,
			}},
			prettier.HardLine{},
			prettier.Text(")"),
		)
	} else {
		// No comments: use soft-breaking argument list
		plainDocs := make([]prettier.Doc, len(args))
		for i, a := range args {
			plainDocs[i] = a.doc
		}
		argSep := prettier.Concat{prettier.Text(","), prettier.Line{}}
		parts = append(parts,
			prettier.WrapParentheses(
				prettier.Join(argSep, plainDocs...),
				prettier.SoftLine{},
			),
		)
	}

	return parts
}

// stringTemplate renders a string template with interpolation expressions
// kept flat (no line breaks inside \(...)). The upstream Doc() renders each
// interpolation via expr.Doc() which can include Line{} breaks. We render
// each interpolation as a flat Text node using expr.String().
func (r *renderer) stringTemplate(e *ast.StringTemplateExpression) prettier.Doc {
	if len(e.Expressions) == 0 {
		return prettier.Text(ast.QuoteString(e.Values[0]))
	}

	concat := make(prettier.Concat, 0, 2+len(e.Values)+(3*len(e.Expressions)))
	concat = append(concat, prettier.Text(`"`))
	for i, value := range e.Values {
		var sb strings.Builder
		ast.QuoteStringInner(value, &sb)
		concat = append(concat, prettier.Text(sb.String()))

		if i < len(e.Expressions) {
			expr := e.Expressions[i]
			// Render interpolation expression as flat text to prevent
			// line breaks inside \(...). Drain any comments on it.
			r.cm.Take(expr)
			r.drainDescendants(expr, nil)
			concat = append(concat,
				prettier.Text(`\(`),
				prettier.Text(expr.String()),
				prettier.Text(`)`),
			)
		}
	}
	concat = append(concat, prettier.Text(`"`))
	return concat
}
