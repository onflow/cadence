package render

import (
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/turbolent/prettier"
)

// renderDeclaration dispatches to a custom renderer for the declaration type
// if we need to override the upstream Doc() behavior, otherwise falls back
// to the default Doc().
func renderDeclaration(decl ast.Declaration, cm *trivia.CommentMap) prettier.Doc {
	var doc prettier.Doc

	switch d := decl.(type) {
	case *ast.FunctionDeclaration:
		doc = renderFunction(d, cm)
	case *ast.CompositeDeclaration:
		doc = renderComposite(d, cm)
	case *ast.InterfaceDeclaration:
		doc = renderInterface(d, cm)
	case *ast.VariableDeclaration:
		doc = renderVariable(d, cm)
	case *ast.FieldDeclaration:
		doc = renderField(d, cm)
	case *ast.SpecialFunctionDeclaration:
		doc = renderSpecialFunction(d, cm)
	default:
		// For unknown declaration types, use upstream Doc() and drain
		// any descendant comments so they're not orphaned.
		doc = decl.Doc()
		return wrapWithAllComments(decl, doc, cm)
	}

	return wrapWithComments(decl, doc, cm)
}

// renderFunction renders a function declaration with access on the same line.
func renderFunction(d *ast.FunctionDeclaration, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, d.Access.Doc(), prettier.Space)
	}

	// Purity (view)
	if d.Purity != ast.FunctionPurityUnspecified {
		parts = append(parts, prettier.Text(d.Purity.Keyword()), prettier.Space)
	}

	// Static/native flags
	if d.IsStatic() {
		parts = append(parts, prettier.Text("static"), prettier.Space)
	}
	if d.IsNative() {
		parts = append(parts, prettier.Text("native"), prettier.Space)
	}

	// "fun" keyword + name
	parts = append(parts, prettier.Text("fun"), prettier.Space)
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	// Type parameters
	if d.TypeParameterList != nil {
		parts = append(parts, d.TypeParameterList.Doc())
	}

	// Parameters
	if d.ParameterList != nil {
		paramDoc := d.ParameterList.Doc()
		drainWalkable(d.ParameterList, cm)
		parts = append(parts, paramDoc)
	}

	// Return type
	if d.ReturnTypeAnnotation != nil && d.ReturnTypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), wrapWithAllComments(d.ReturnTypeAnnotation, d.ReturnTypeAnnotation.Doc(), cm))
	}

	// Function body
	if d.FunctionBlock != nil {
		parts = append(parts, prettier.Space, renderFunctionBlock(d.FunctionBlock, cm))
	}

	return parts
}

// renderFunctionBlock renders a { pre { } post { } stmts } block with
// comment interleaving between statements.
func renderFunctionBlock(b *ast.FunctionBlock, cm *trivia.CommentMap) prettier.Doc {
	if b.IsEmpty() {
		return prettier.Text("{}")
	}

	body := prettier.Concat{}
	needSep := false

	// Pre-conditions
	if b.PreConditions != nil && !b.PreConditions.IsEmpty() {
		condDoc := b.PreConditions.Doc(prettier.Text("pre"))
		drainConditionComments(b.PreConditions, cm)
		body = append(body, condDoc)
		needSep = true
	}

	// Post-conditions
	if b.PostConditions != nil && !b.PostConditions.IsEmpty() {
		if needSep {
			body = append(body, prettier.HardLine{})
		}
		condDoc := b.PostConditions.Doc(prettier.Text("post"))
		drainConditionComments(b.PostConditions, cm)
		body = append(body, condDoc)
		needSep = true
	}

	// Statements
	if b.Block != nil {
		for _, stmt := range b.Block.Statements {
			if needSep {
				body = append(body, prettier.HardLine{})
			}
			doc := renderStatement(stmt, cm)
			body = append(body, doc)
			needSep = true
		}
	}

	return prettier.Concat{
		prettier.Text("{"),
		prettier.Indent{Doc: prettier.Concat{
			prettier.HardLine{},
			body,
		}},
		prettier.HardLine{},
		prettier.Text("}"),
	}
}

// renderStatement dispatches to custom renderers for specific statement types,
// otherwise falls back to the upstream Doc().
func renderStatement(stmt ast.Statement, cm *trivia.CommentMap) prettier.Doc {
	switch s := stmt.(type) {
	case *ast.ReturnStatement:
		return wrapWithComments(s, renderReturnStatement(s, cm), cm)
	case *ast.ForStatement:
		return wrapWithComments(s, renderForStatement(s, cm), cm)
	case *ast.WhileStatement:
		return wrapWithComments(s, renderWhileStatement(s, cm), cm)
	case *ast.IfStatement:
		return wrapWithComments(s, renderIfStatement(s, cm), cm)
	case *ast.VariableDeclaration:
		return wrapWithComments(s, renderVariable(s, cm), cm)
	default:
		return wrapWithAllComments(stmt, stmt.Doc(), cm)
	}
}

// renderBlock renders the body of a block by iterating statements and
// interleaving comments. Returns the body content without braces.
func renderBlock(b *ast.Block, cm *trivia.CommentMap) prettier.Doc {
	if b == nil || len(b.Statements) == 0 {
		return nil
	}

	body := prettier.Concat{}
	for i, stmt := range b.Statements {
		if i > 0 {
			body = append(body, prettier.HardLine{})
		}
		doc := renderStatement(stmt, cm)
		body = append(body, doc)
	}
	return body
}

// renderBlockBraces wraps a block body in { ... } with indentation.
func renderBlockBraces(b *ast.Block, cm *trivia.CommentMap) prettier.Doc {
	body := renderBlock(b, cm)
	if body == nil {
		return prettier.Text("{}")
	}
	return prettier.Concat{
		prettier.Text("{"),
		prettier.Indent{Doc: prettier.Concat{
			prettier.HardLine{},
			body,
		}},
		prettier.HardLine{},
		prettier.Text("}"),
	}
}

// renderForStatement renders a for-in loop with comment interleaving in the body.
func renderForStatement(s *ast.ForStatement, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, prettier.Text("for "))
	parts = append(parts, prettier.Text(s.Identifier.Identifier))
	parts = append(parts, prettier.Text(" in "))
	parts = append(parts, wrapWithAllComments(s.Value, s.Value.Doc(), cm))
	parts = append(parts, prettier.Space)
	parts = append(parts, renderBlockBraces(s.Block, cm))

	return parts
}

// renderWhileStatement renders a while loop with comment interleaving in the body.
func renderWhileStatement(s *ast.WhileStatement, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, prettier.Text("while "))
	parts = append(parts, wrapWithAllComments(s.Test, s.Test.Doc(), cm))
	parts = append(parts, prettier.Space)
	parts = append(parts, renderBlockBraces(s.Block, cm))

	return parts
}

// renderIfStatement renders an if/else-if/else chain with comment interleaving.
func renderIfStatement(s *ast.IfStatement, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, prettier.Text("if "))
	parts = append(parts, wrapWithAllComments(s.Test, s.Test.Doc(), cm))
	parts = append(parts, prettier.Space)
	parts = append(parts, renderBlockBraces(s.Then, cm))

	if s.Else != nil && len(s.Else.Statements) > 0 {
		// Check if the else block is a single if-statement (else-if chain)
		if len(s.Else.Statements) == 1 {
			if elseIf, ok := s.Else.Statements[0].(*ast.IfStatement); ok {
				parts = append(parts, prettier.Text(" else "))
				parts = append(parts, wrapWithComments(elseIf, renderIfStatement(elseIf, cm), cm))
				return parts
			}
		}
		parts = append(parts, prettier.Text(" else "))
		parts = append(parts, renderBlockBraces(s.Else, cm))
	}

	return parts
}

// renderReturnStatement renders a return statement. For binary expressions
// (e.g., ?? nil-coalescing), wraps in Indent so continuation lines are
// indented relative to "return". Other expressions render directly to
// avoid over-indenting function call arguments.
func renderReturnStatement(s *ast.ReturnStatement, cm *trivia.CommentMap) prettier.Doc {
	if s.Expression == nil {
		return prettier.Text("return")
	}

	exprDoc := wrapWithAllComments(s.Expression, s.Expression.Doc(), cm)

	// Binary expressions need Indent for proper continuation line indentation
	if _, ok := s.Expression.(*ast.BinaryExpression); ok {
		return prettier.Concat{
			prettier.Text("return "),
			prettier.Indent{Doc: exprDoc},
		}
	}

	return prettier.Concat{
		prettier.Text("return "),
		exprDoc,
	}
}

// renderComposite renders a composite declaration (resource, struct, contract, etc.)
// with access on the same line.
func renderComposite(d *ast.CompositeDeclaration, cm *trivia.CommentMap) prettier.Doc {
	// Events use a special compact format (no members block with braces)
	if d.CompositeKind == common.CompositeKindEvent {
		doc := d.EventDoc()
		// Drain any comments attached to event children (parameter types, etc.)
		var extras []prettier.Doc
		drainDescendantComments(d, cm, &extras)
		if len(extras) > 0 {
			parts := prettier.Concat{doc}
			for _, e := range extras {
				parts = append(parts, prettier.HardLine{}, e)
			}
			return parts
		}
		return doc
	}

	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, d.Access.Doc(), prettier.Space)
	}

	// Kind keyword
	parts = append(parts, prettier.Text(d.CompositeKind.Keyword()), prettier.Space)

	// Name
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	// Conformances
	conformances := d.Conformances
	if len(conformances) > 0 {
		parts = append(parts, prettier.Text(":"), prettier.Space)
		for i, c := range conformances {
			if i > 0 {
				parts = append(parts, prettier.Text(","), prettier.Space)
			}
			parts = append(parts, c.Doc())
		}
	}

	// Members
	parts = append(parts, renderMembersBlock(d.Members, cm))
	return parts
}

// renderInterface renders an interface declaration with access on the same line.
func renderInterface(d *ast.InterfaceDeclaration, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, d.Access.Doc(), prettier.Space)
	}

	parts = append(parts, prettier.Text(d.CompositeKind.Keyword()), prettier.Space)
	parts = append(parts, prettier.Text("interface"), prettier.Space)
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	conformances := d.Conformances
	if len(conformances) > 0 {
		parts = append(parts, prettier.Text(":"), prettier.Space)
		for i, c := range conformances {
			if i > 0 {
				parts = append(parts, prettier.Text(","), prettier.Space)
			}
			parts = append(parts, c.Doc())
		}
	}

	parts = append(parts, renderMembersBlock(d.Members, cm))
	return parts
}

// renderMembersBlock renders a { members } block with each member using
// our custom declaration renderers.
func renderMembersBlock(members *ast.Members, cm *trivia.CommentMap) prettier.Doc {
	if members == nil {
		return prettier.Text(" {}")
	}

	decls := members.Declarations()
	if len(decls) == 0 {
		return prettier.Text(" {}")
	}

	body := prettier.Concat{}
	for i, decl := range decls {
		if i > 0 {
			body = append(body, prettier.HardLine{}, prettier.HardLine{})
		}
		doc := renderDeclaration(decl, cm)
		body = append(body, doc)
	}

	return prettier.Concat{
		prettier.Space,
		prettier.Text("{"),
		prettier.Indent{Doc: prettier.Concat{
			prettier.HardLine{},
			body,
		}},
		prettier.HardLine{},
		prettier.Text("}"),
	}
}

// renderVariable renders a variable declaration with access on the same line.
func renderVariable(d *ast.VariableDeclaration, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, d.Access.Doc(), prettier.Space)
	}

	// let/var keyword
	if d.IsConstant {
		parts = append(parts, prettier.Text("let"), prettier.Space)
	} else {
		parts = append(parts, prettier.Text("var"), prettier.Space)
	}

	// Identifier
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	// Type annotation
	if d.TypeAnnotation != nil && d.TypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), wrapWithAllComments(d.TypeAnnotation, d.TypeAnnotation.Doc(), cm))
	}

	// Transfer and value
	if d.Value != nil {
		parts = append(parts, prettier.Space)
		parts = append(parts, prettier.Text(d.Transfer.Operation.Operator()))
		// Binary expressions (e.g., ?? nil-coalescing) need Indent for
		// continuation line indentation. Other expressions render directly
		// to avoid over-indenting function call arguments.
		if _, ok := d.Value.(*ast.BinaryExpression); ok {
			parts = append(parts, prettier.Group{
				Doc: prettier.Indent{
					Doc: prettier.Concat{
						prettier.Line{},
						d.Value.Doc(),
					},
				},
			})
		} else {
			parts = append(parts, prettier.Space)
			parts = append(parts, d.Value.Doc())
		}
	}

	// Second transfer (for swap operations)
	if d.SecondValue != nil {
		parts = append(parts, prettier.Space)
		parts = append(parts, prettier.Text(d.SecondTransfer.Operation.Operator()))
		parts = append(parts, prettier.Space)
		parts = append(parts, d.SecondValue.Doc())
	}

	return parts
}

// drainConditionComments drains any comments attached to Conditions' children.
func drainConditionComments(conds *ast.Conditions, cm *trivia.CommentMap) {
	conds.Walk(func(child ast.Element) {
		if child == nil {
			return
		}
		cm.Take(child)
		var discard []prettier.Doc
		drainDescendantComments(child, cm, &discard)
	})
}

// renderSpecialFunction renders init/destroy/prepare declarations.
// These don't use the "fun" keyword.
func renderSpecialFunction(d *ast.SpecialFunctionDeclaration, cm *trivia.CommentMap) prettier.Doc {
	fn := d.FunctionDeclaration
	parts := prettier.Concat{}

	// Access modifier (rare for special functions but possible)
	if fn.Access != ast.AccessNotSpecified {
		parts = append(parts, fn.Access.Doc(), prettier.Space)
	}

	// Purity
	if fn.Purity != ast.FunctionPurityUnspecified {
		parts = append(parts, prettier.Text(fn.Purity.Keyword()), prettier.Space)
	}

	// Name (init/destroy/prepare)
	parts = append(parts, prettier.Text(fn.Identifier.Identifier))

	// Parameters
	if fn.ParameterList != nil {
		paramDoc := fn.ParameterList.Doc()
		drainWalkable(fn.ParameterList, cm)
		parts = append(parts, paramDoc)
	}

	// Return type
	if fn.ReturnTypeAnnotation != nil && fn.ReturnTypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), fn.ReturnTypeAnnotation.Doc())
	}

	// Body
	if fn.FunctionBlock != nil {
		parts = append(parts, prettier.Space, renderFunctionBlock(fn.FunctionBlock, cm))
	}

	return parts
}

// renderField renders a field declaration (inside composites) with access on the same line.
func renderField(d *ast.FieldDeclaration, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, d.Access.Doc(), prettier.Space)
	}

	if d.IsStatic() {
		parts = append(parts, prettier.Text("static"), prettier.Space)
	}
	if d.IsNative() {
		parts = append(parts, prettier.Text("native"), prettier.Space)
	}

	parts = append(parts, prettier.Text(d.VariableKind.Keyword()), prettier.Space)
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	if d.TypeAnnotation != nil && d.TypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), wrapWithAllComments(d.TypeAnnotation, d.TypeAnnotation.Doc(), cm))
	}

	return parts
}
