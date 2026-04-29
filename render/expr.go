package render

import (
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/turbolent/prettier"
)

// renderExpression dispatches to custom renderers for expression types that
// need fixes (invocations with displaced comments, casts with missing indent),
// otherwise falls back to the upstream Doc() with full comment draining.
func renderExpression(expr ast.Expression, cm *trivia.CommentMap) prettier.Doc {
	switch e := expr.(type) {
	case *ast.InvocationExpression:
		if cm.HasTrailing(e.InvokedExpression) {
			return wrapWithComments(e, renderInvocationWithComments(e, cm), cm)
		}
	case *ast.CastingExpression:
		return wrapWithComments(e, renderCastingExpression(e, cm), cm)
	}
	return wrapWithAllComments(expr, expr.Doc(), cm)
}

// renderIndentedExpression renders an expression wrapped in Indent so that
// continuation lines (from Line{} breaks inside the expression's Doc) are
// indented. Used for while/if conditions where the upstream BinaryExpression
// Doc() uses Line{} without Indent for the operator continuation.
func renderIndentedExpression(expr ast.Expression, cm *trivia.CommentMap) prettier.Doc {
	doc := renderExpression(expr, cm)
	return prettier.Indent{Doc: doc}
}

// renderInvocationWithComments renders a function call where comments between
// the function name and the opening paren need to be placed inside the
// argument list. This forces arguments to break across lines.
func renderInvocationWithComments(e *ast.InvocationExpression, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	// Take comments from the invoked expression.
	leading, sameLine, trailing := cm.Take(e.InvokedExpression)
	invokedDoc := renderExpression(e.InvokedExpression, cm)

	// Re-apply leading and same-line.
	if len(leading) > 0 || sameLine != nil {
		wrapped := prettier.Concat{}
		for _, g := range leading {
			wrapped = append(wrapped, renderCommentGroup(g), prettier.HardLine{})
		}
		wrapped = append(wrapped, invokedDoc)
		if sameLine != nil {
			wrapped = append(wrapped, prettier.Text("  "), renderCommentGroupInline(sameLine))
		}
		invokedDoc = wrapped
	}
	parts = append(parts, invokedDoc)

	// Type arguments (use upstream rendering)
	if len(e.TypeArguments) > 0 {
		typeArgDocs := make([]prettier.Doc, len(e.TypeArguments))
		for i, ta := range e.TypeArguments {
			typeArgDocs[i] = wrapWithAllComments(ta, ta.Doc(), cm)
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

	if len(e.Arguments) == 0 {
		parts = append(parts, prettier.Text("()"))
		for _, g := range trailing {
			parts = append(parts, prettier.HardLine{}, renderCommentGroup(g))
		}
		return parts
	}

	// Build argument list with trailing comments before first arg.
	// Use upstream arg.Doc() for each argument, draining descendant
	// comments to prevent orphans.
	inner := prettier.Concat{}
	for _, g := range trailing {
		inner = append(inner, renderCommentGroup(g), prettier.HardLine{})
	}
	var leftovers []prettier.Doc
	for i, arg := range e.Arguments {
		if i > 0 {
			inner = append(inner, prettier.Text(","), prettier.HardLine{})
		}
		inner = append(inner, arg.Doc())
		// Drain comments from the Argument element (now walkable since
		// onflow/cadence PR #4485) and its descendant expression.
		cm.Take(arg)
		drainDescendantComments(arg, cm, &leftovers)
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

	// Emit any leftover descendant comments after the invocation
	for _, e := range leftovers {
		parts = append(parts, prettier.HardLine{}, e)
	}

	return parts
}

// renderCastingExpression renders a cast (as/as!/as?) with the operator and
// target type indented on continuation lines. The upstream Doc() places the
// operator at the same indent level as the expression, which looks wrong.
func renderCastingExpression(e *ast.CastingExpression, cm *trivia.CommentMap) prettier.Doc {
	exprDoc := renderExpression(e.Expression, cm)
	typeDoc := wrapWithAllComments(e.TypeAnnotation, e.TypeAnnotation.Doc(), cm)

	return prettier.Group{
		Doc: prettier.Concat{
			prettier.Group{Doc: exprDoc},
			prettier.Indent{
				Doc: prettier.Concat{
					prettier.Line{},
					e.Operation.Doc(),
					prettier.Space,
					typeDoc,
				},
			},
		},
	}
}
