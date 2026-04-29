package render

import (
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/turbolent/prettier"
)

// renderDeclaration dispatches to a custom renderer for the declaration type
// if we need to override the upstream Doc() behavior, otherwise falls back
// to the default Doc().
func renderDeclaration(decl ast.Declaration, cm *trivia.CommentMap) prettier.Doc {
	var doc prettier.Doc

	switch d := decl.(type) {
	case *ast.FunctionDeclaration:
		doc = renderFunction(d)
	case *ast.CompositeDeclaration:
		doc = renderComposite(d, cm)
	case *ast.InterfaceDeclaration:
		doc = renderInterface(d, cm)
	case *ast.VariableDeclaration:
		doc = renderVariable(d)
	case *ast.FieldDeclaration:
		doc = renderField(d)
	case *ast.SpecialFunctionDeclaration:
		doc = renderSpecialFunction(d)
	default:
		doc = decl.Doc()
	}

	return wrapWithComments(decl, doc, cm)
}

// renderFunction renders a function declaration with access on the same line.
func renderFunction(d *ast.FunctionDeclaration) prettier.Doc {
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
		parts = append(parts, d.ParameterList.Doc())
	}

	// Return type
	if d.ReturnTypeAnnotation != nil && d.ReturnTypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), d.ReturnTypeAnnotation.Doc())
	}

	// Function body
	if d.FunctionBlock != nil {
		parts = append(parts, prettier.Space, d.FunctionBlock.Doc())
	}

	return parts
}

// renderComposite renders a composite declaration (resource, struct, contract, etc.)
// with access on the same line.
func renderComposite(d *ast.CompositeDeclaration, cm *trivia.CommentMap) prettier.Doc {
	// Events render differently
	if d.CompositeKind == 0 { // event
		return d.Doc()
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
func renderVariable(d *ast.VariableDeclaration) prettier.Doc {
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
		parts = append(parts, prettier.Text(": "), d.TypeAnnotation.Doc())
	}

	// Transfer and value
	if d.Value != nil {
		parts = append(parts, prettier.Space)
		parts = append(parts, prettier.Text(d.Transfer.Operation.Operator()))
		parts = append(parts, prettier.Space)
		parts = append(parts, d.Value.Doc())
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

// renderSpecialFunction renders init/destroy/prepare declarations.
// These don't use the "fun" keyword.
func renderSpecialFunction(d *ast.SpecialFunctionDeclaration) prettier.Doc {
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
		parts = append(parts, fn.ParameterList.Doc())
	}

	// Return type
	if fn.ReturnTypeAnnotation != nil && fn.ReturnTypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), fn.ReturnTypeAnnotation.Doc())
	}

	// Body
	if fn.FunctionBlock != nil {
		parts = append(parts, prettier.Space, fn.FunctionBlock.Doc())
	}

	return parts
}

// renderField renders a field declaration (inside composites) with access on the same line.
func renderField(d *ast.FieldDeclaration) prettier.Doc {
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
		parts = append(parts, prettier.Text(": "), d.TypeAnnotation.Doc())
	}

	return parts
}
