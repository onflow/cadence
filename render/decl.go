package render

import (
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/turbolent/prettier"
)

// hasBlankLineBetween checks the source bytes between two statements for a
// blank line (a line containing only whitespace). This is more reliable than
// comparing AST line numbers, which can be inaccurate for multi-line expressions.
// Must be called BEFORE cm.Take() drains comments, since it uses comment
// positions to narrow the byte range.
func hasBlankLineBetween(prev, curr ast.Statement, cm *trivia.CommentMap, source []byte) bool {
	if len(source) == 0 {
		return false
	}

	// Find the last byte offset of prev (including trailing comments).
	endOffset := prev.EndPosition(nil).Offset
	if trailing := cm.Trailing[prev]; len(trailing) > 0 {
		if tEnd := trailing[len(trailing)-1].EndPos().Offset; tEnd > endOffset {
			endOffset = tEnd
		}
	}

	// Find the first byte offset of curr (including leading comments).
	startOffset := curr.StartPosition().Offset
	if leading := cm.Leading[curr]; len(leading) > 0 {
		if lStart := leading[0].StartPos().Offset; lStart < startOffset {
			startOffset = lStart
		}
	}

	// Scan the source bytes between the two positions for a blank line:
	// two newlines with only whitespace between them.
	if endOffset >= startOffset || endOffset >= len(source) {
		return false
	}
	sawNewline := false
	for i := endOffset; i < startOffset && i < len(source); i++ {
		b := source[i]
		if b == '\n' {
			if sawNewline {
				return true
			}
			sawNewline = true
		} else if b != ' ' && b != '\t' && b != '\r' {
			sawNewline = false
		}
	}
	return false
}

// renderDeclaration dispatches to a custom renderer for the declaration type
// if we need to override the upstream Doc() behavior, otherwise falls back
// to the default Doc().
func renderDeclaration(decl ast.Declaration, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	var doc prettier.Doc

	switch d := decl.(type) {
	case *ast.FunctionDeclaration:
		doc = renderFunction(d, cm, ctx)
	case *ast.CompositeDeclaration:
		doc = renderComposite(d, cm, ctx)
	case *ast.InterfaceDeclaration:
		doc = renderInterface(d, cm, ctx)
	case *ast.VariableDeclaration:
		doc = renderVariable(d, cm, ctx)
	case *ast.FieldDeclaration:
		doc = renderField(d, cm)
	case *ast.SpecialFunctionDeclaration:
		doc = renderSpecialFunction(d, cm, ctx)
	case *ast.EntitlementMappingDeclaration:
		doc = renderEntitlementMapping(d, cm)
	case *ast.TransactionDeclaration:
		doc = renderTransaction(d, cm, ctx)
	default:
		// For unknown declaration types, use upstream Doc() and drain
		// any descendant comments so they're not orphaned.
		doc = decl.Doc()
		return wrapWithAllComments(decl, doc, cm)
	}

	// Drain any remaining descendant comments (e.g., NominalType nodes
	// inside entitlement access modifiers) that specific renderers didn't take.
	drainDescendantComments(decl, cm, nil)

	doc = wrapWithComments(decl, doc, cm)
	if ctx.HasSemicolon(decl) {
		doc = prettier.Concat{doc, prettier.Text(";")}
	}
	return doc
}

// renderAccess renders an access modifier and takes any comments attached
// to its child NominalType nodes (entitlement types). Comments are rendered
// between the access modifier and the following keyword.
func renderAccess(access ast.Access, cm *trivia.CommentMap) prettier.Doc {
	if access == ast.AccessNotSpecified {
		return nil
	}
	// Drain comments from entitlement NominalType children so they don't
	// become orphaned. These comments are on AST nodes that the upstream
	// Access.Doc() renders as flat text (e.g., "access(A)"), so there's
	// no natural position for them in the output.
	access.Walk(func(child ast.Element) {
		if child == nil {
			return
		}
		cm.Take(child)
	})
	return prettier.Concat{access.Doc(), prettier.Space}
}

// renderFunction renders a function declaration with access on the same line.
func renderFunction(d *ast.FunctionDeclaration, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, renderAccess(d.Access, cm))
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

	// Parameters — use custom rendering to preserve comments between params
	if d.ParameterList != nil {
		paramDoc, _ := renderParameterList(d.ParameterList, cm)
		parts = append(parts, paramDoc)
	}

	// Return type
	if d.ReturnTypeAnnotation != nil && d.ReturnTypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), wrapWithAllComments(d.ReturnTypeAnnotation, d.ReturnTypeAnnotation.Doc(), cm))
	}

	// Function body
	if d.FunctionBlock != nil {
		parts = append(parts, prettier.Space, renderFunctionBlock(d.FunctionBlock, cm, ctx))
	}

	return parts
}

// renderFunctionBlock renders a { pre { } post { } stmts } block with
// comment interleaving between statements.
func renderFunctionBlock(b *ast.FunctionBlock, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
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
		// Drain any comments attached to the Block node itself
		// (e.g., comments inside post{} blocks in interface functions)
		leading, _, trailing := cm.Take(b.Block)
		for _, g := range leading {
			if needSep {
				body = append(body, prettier.HardLine{})
			}
			body = append(body, renderCommentGroup(g))
			needSep = true
		}
		// Pre-compute blank line flags before rendering drains the CommentMap.
		stmts := b.Block.Statements
		blankBefore := make([]bool, len(stmts))
		for i := 1; i < len(stmts); i++ {
			blankBefore[i] = hasBlankLineBetween(stmts[i-1], stmts[i], cm, ctx.Source)
		}
		for i, stmt := range stmts {
			if needSep {
				body = append(body, prettier.HardLine{})
				if blankBefore[i] {
					body = append(body, prettier.HardLine{})
				}
			}
			doc := renderStatement(stmt, cm, ctx)
			body = append(body, doc)
			needSep = true
		}
		for _, g := range trailing {
			if needSep {
				body = append(body, prettier.HardLine{})
			}
			body = append(body, renderCommentGroup(g))
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
func renderStatement(stmt ast.Statement, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	var doc prettier.Doc
	switch s := stmt.(type) {
	case *ast.ReturnStatement:
		doc = wrapWithComments(s, renderReturnStatement(s, cm, ctx), cm)
	case *ast.ForStatement:
		doc = wrapWithComments(s, renderForStatement(s, cm, ctx), cm)
	case *ast.WhileStatement:
		doc = wrapWithComments(s, renderWhileStatement(s, cm, ctx), cm)
	case *ast.IfStatement:
		doc = wrapWithComments(s, renderIfStatement(s, cm, ctx), cm)
	case *ast.VariableDeclaration:
		doc = wrapWithComments(s, renderVariable(s, cm, ctx), cm)
	case *ast.AssignmentStatement:
		doc = wrapWithComments(s, renderAssignmentStatement(s, cm, ctx), cm)
	case *ast.ExpressionStatement:
		doc = wrapWithComments(s, renderExpression(s.Expression, cm, ctx), cm)
	default:
		doc = wrapWithAllComments(stmt, stmt.Doc(), cm)
	}
	if ctx.HasSemicolon(stmt) {
		doc = prettier.Concat{doc, prettier.Text(";")}
	}
	return doc
}

// renderBlock renders the body of a block by iterating statements and
// interleaving comments. Returns the body content without braces.
func renderBlock(b *ast.Block, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	if b == nil || len(b.Statements) == 0 {
		return nil
	}

	blankBefore := make([]bool, len(b.Statements))
	for i := 1; i < len(b.Statements); i++ {
		blankBefore[i] = hasBlankLineBetween(b.Statements[i-1], b.Statements[i], cm, ctx.Source)
	}

	body := prettier.Concat{}
	for i, stmt := range b.Statements {
		if i > 0 {
			body = append(body, prettier.HardLine{})
			if blankBefore[i] {
				body = append(body, prettier.HardLine{})
			}
		}
		doc := renderStatement(stmt, cm, ctx)
		body = append(body, doc)
	}
	return body
}

// renderBlockBraces wraps a block body in { ... } with indentation.
func renderBlockBraces(b *ast.Block, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	body := renderBlock(b, cm, ctx)
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
func renderForStatement(s *ast.ForStatement, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, prettier.Text("for "))
	parts = append(parts, prettier.Text(s.Identifier.Identifier))
	parts = append(parts, prettier.Text(" in "))
	parts = append(parts, renderExpression(s.Value, cm, ctx))
	parts = append(parts, prettier.Space)
	parts = append(parts, renderBlockBraces(s.Block, cm, ctx))

	return parts
}

// renderWhileStatement renders a while loop with comment interleaving in the body.
func renderWhileStatement(s *ast.WhileStatement, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, prettier.Text("while "))
	parts = append(parts, renderExpression(s.Test, cm, ctx))
	parts = append(parts, prettier.Space)
	parts = append(parts, renderBlockBraces(s.Block, cm, ctx))

	return parts
}

// renderIfStatement renders an if/else-if/else chain with comment interleaving.
func renderIfStatement(s *ast.IfStatement, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, prettier.Text("if "))
	parts = append(parts, wrapWithAllComments(s.Test, s.Test.Doc(), cm))
	parts = append(parts, prettier.Space)
	parts = append(parts, renderBlockBraces(s.Then, cm, ctx))

	if s.Else != nil && len(s.Else.Statements) > 0 {
		// Check if the else block is a single if-statement (else-if chain)
		if len(s.Else.Statements) == 1 {
			if elseIf, ok := s.Else.Statements[0].(*ast.IfStatement); ok {
				parts = append(parts, prettier.Text(" else "))
				parts = append(parts, wrapWithComments(elseIf, renderIfStatement(elseIf, cm, ctx), cm))
				return parts
			}
		}
		parts = append(parts, prettier.Text(" else "))
		parts = append(parts, renderBlockBraces(s.Else, cm, ctx))
	}

	return parts
}

// renderAssignmentStatement renders target = value without the upstream's
// extra Indent wrapper that over-indents function call arguments.
func renderAssignmentStatement(s *ast.AssignmentStatement, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, renderExpression(s.Target, cm, ctx))
	parts = append(parts, prettier.Space)
	parts = append(parts, s.Transfer.Doc())
	parts = append(parts, prettier.Space)
	parts = append(parts, renderExpression(s.Value, cm, ctx))

	return parts
}

// renderReturnStatement renders a return statement. For binary expressions
// (e.g., ?? nil-coalescing), wraps in Indent so continuation lines are
// indented relative to "return". Other expressions render directly to
// avoid over-indenting function call arguments.
func renderReturnStatement(s *ast.ReturnStatement, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	if s.Expression == nil {
		return prettier.Text("return")
	}

	// Binary expressions need Indent for proper continuation line indentation.
	// Drain descendant comments outside the Indent so they don't pick up
	// expression-level indentation that isn't stable across re-formats.
	if _, ok := s.Expression.(*ast.BinaryExpression); ok {
		exprDoc := wrapWithComments(s.Expression, s.Expression.Doc(), cm)
		parts := prettier.Concat{
			prettier.Text("return "),
			prettier.Indent{Doc: exprDoc},
		}
		var extras []prettier.Doc
		drainDescendantComments(s.Expression, cm, &extras)
		for _, e := range extras {
			parts = append(parts, prettier.HardLine{}, e)
		}
		return parts
	}

	exprDoc := renderExpression(s.Expression, cm, ctx)
	return prettier.Concat{
		prettier.Text("return "),
		exprDoc,
	}
}

// renderComposite renders a composite declaration (resource, struct, contract, etc.)
// with access on the same line.
func renderComposite(d *ast.CompositeDeclaration, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	// Events use a special compact format (no members block with braces)
	if d.CompositeKind == common.CompositeKindEvent {
		return renderEvent(d, cm, ctx)
	}

	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, renderAccess(d.Access, cm))
	}

	// Kind keyword
	parts = append(parts, prettier.Text(d.CompositeKind.Keyword()), prettier.Space)

	// Name
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	// Conformances — the upstream Walk() now yields these as children,
	// so comments may be attached to them. Drain conformance comments
	// and move trailing comments to be leading of the first member
	// (they logically describe the first field, not the conformance type).
	conformances := d.Conformances
	if len(conformances) > 0 {
		parts = append(parts, prettier.Text(":"), prettier.Space)
		for i, c := range conformances {
			if i > 0 {
				parts = append(parts, prettier.Text(","), prettier.Space)
			}
			parts = append(parts, c.Doc())
			_, _, trailing := cm.Take(c)
			if len(trailing) > 0 {
				decls := d.Members.Declarations()
				if len(decls) > 0 {
					cm.Leading[decls[0]] = append(trailing, cm.Leading[decls[0]]...)
				}
			}
		}
	}

	// Members
	parts = append(parts, renderMembersBlock(d.Members, cm, ctx))
	return parts
}

// renderEvent renders an event declaration with comments interleaved between
// parameters. The upstream EventDoc() + drain approach displaces parameter
// comments outside the closing paren.
func renderEvent(d *ast.CompositeDeclaration, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, renderAccess(d.Access, cm))
	}

	// "event Name"
	parts = append(parts, prettier.Text(d.CompositeKind.Keyword()), prettier.Space)
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	// Get parameters from the event's initializer
	initializers := d.Members.Initializers()
	if len(initializers) != 1 {
		// Fallback: no valid initializer, use upstream
		drainDescendantComments(d, cm, nil)
		return parts
	}

	paramList := initializers[0].FunctionDeclaration.ParameterList
	paramDoc, _ := renderParameterList(paramList, cm)
	parts = append(parts, paramDoc)

	// Drain any remaining descendant comments (type annotations, etc.)
	drainDescendantComments(d, cm, nil)

	return parts
}

// renderTransaction renders a transaction declaration with comment
// interleaving inside prepare/execute blocks. Without this, the default
// wrapWithAllComments path drains all block-interior comments and appends
// them after the closing brace.
func renderTransaction(d *ast.TransactionDeclaration, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	doc := prettier.Concat{prettier.Text("transaction")}

	// Parameters
	paramDoc, paramTrailing := renderParameterList(d.ParameterList, cm)
	doc = append(doc, paramDoc)

	// Move trailing comments from last parameter to leading of first field
	if len(paramTrailing) > 0 && len(d.Fields) > 0 {
		cm.Leading[d.Fields[0]] = append(paramTrailing, cm.Leading[d.Fields[0]]...)
	}

	// Build body contents
	var contents []prettier.Doc

	// Fields
	for _, field := range d.Fields {
		fieldDoc := renderDeclaration(field, cm, ctx)
		contents = append(contents, fieldDoc)
	}

	// Prepare block
	if d.Prepare != nil {
		prepareDoc := renderDeclaration(d.Prepare, cm, ctx)
		contents = append(contents, prepareDoc)
	}

	// Pre-conditions
	if d.PreConditions != nil && !d.PreConditions.IsEmpty() {
		condDoc := d.PreConditions.Doc(prettier.Text("pre"))
		drainWalkable(d.PreConditions, cm)
		contents = append(contents, condDoc)
	}

	// Execute block
	if d.Execute != nil {
		executeDoc := renderDeclaration(d.Execute, cm, ctx)
		contents = append(contents, executeDoc)
	}

	// Post-conditions
	if d.PostConditions != nil && !d.PostConditions.IsEmpty() {
		condDoc := d.PostConditions.Doc(prettier.Text("post"))
		drainWalkable(d.PostConditions, cm)
		contents = append(contents, condDoc)
	}

	// Build the braced body
	if len(contents) == 0 {
		doc = append(doc, prettier.Text(" {}"))
		return doc
	}

	body := prettier.Concat{}
	for i, content := range contents {
		if i > 0 {
			body = append(body, prettier.HardLine{})
		}
		body = append(body, content)
	}

	doc = append(doc,
		prettier.Space,
		prettier.Text("{"),
		prettier.Indent{Doc: prettier.Concat{
			prettier.HardLine{},
			body,
		}},
		prettier.HardLine{},
		prettier.Text("}"),
	)

	return doc
}

// paramInfo holds a rendered parameter and its associated comments.
type paramInfo struct {
	doc      prettier.Doc
	leading  []*trivia.CommentGroup
	sameLine *trivia.CommentGroup
	trailing []*trivia.CommentGroup
}

// renderParameterList renders a function/event parameter list with comments
// interleaved between parameters. ParameterList.Walk() yields TypeAnnotation
// nodes (not Parameter), so comments are attached to TypeAnnotation nodes.
// Returns the rendered doc and any trailing comments from the last parameter
// that the caller should place after the parameter list.
func renderParameterList(paramList *ast.ParameterList, cm *trivia.CommentMap) (prettier.Doc, []*trivia.CommentGroup) {
	if paramList == nil || len(paramList.Parameters) == 0 {
		drainWalkable(paramList, cm)
		return prettier.Text("()"), nil
	}

	// Collect parameters with their comments
	params := make([]paramInfo, len(paramList.Parameters))
	hasComments := false
	var pendingTrailing []*trivia.CommentGroup

	for i, param := range paramList.Parameters {
		p := paramInfo{doc: param.Doc()}
		if param.TypeAnnotation != nil {
			leading, sameLine, trailing := cm.Take(param.TypeAnnotation)
			p.leading = append(pendingTrailing, leading...)
			p.sameLine = sameLine
			p.trailing = trailing
			if len(p.leading) > 0 || p.sameLine != nil {
				hasComments = true
			}
			pendingTrailing = trailing
		} else {
			if len(pendingTrailing) > 0 {
				p.leading = pendingTrailing
				hasComments = true
			}
			pendingTrailing = nil
		}
		params[i] = p
	}

	if !hasComments {
		// No comments: use upstream soft-breaking layout
		drainWalkable(paramList, cm)
		return paramList.Doc(), pendingTrailing
	}

	// Drain any remaining descendant comments (e.g., on NominalType children
	// of TypeAnnotation nodes) so they don't become orphaned.
	drainWalkable(paramList, cm)

	// Comments present: force parameters to break across lines.
	// Same-line comments go after the comma on the same line.
	inner := prettier.Concat{}
	for i, p := range params {
		if i > 0 {
			inner = append(inner, prettier.Text(","))
			// Previous param's same-line comment after comma
			if params[i-1].sameLine != nil {
				inner = append(inner, prettier.Text("  "), renderCommentGroup(params[i-1].sameLine))
			}
			inner = append(inner, prettier.HardLine{})
		}
		// Leading comments for this param
		for _, g := range p.leading {
			inner = append(inner, renderCommentGroup(g), prettier.HardLine{})
		}
		inner = append(inner, p.doc)
	}
	// Last param's same-line comment
	lastParam := params[len(params)-1]
	if lastParam.sameLine != nil {
		inner = append(inner, prettier.Text("  "), renderCommentGroup(lastParam.sameLine))
	}

	return prettier.Concat{
		prettier.Text("("),
		prettier.Indent{Doc: prettier.Concat{
			prettier.HardLine{},
			inner,
		}},
		prettier.HardLine{},
		prettier.Text(")"),
	}, pendingTrailing
}

// renderInterface renders an interface declaration with access on the same line.
func renderInterface(d *ast.InterfaceDeclaration, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	parts := prettier.Concat{}

	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, renderAccess(d.Access, cm))
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
			_, _, trailing := cm.Take(c)
			if len(trailing) > 0 {
				decls := d.Members.Declarations()
				if len(decls) > 0 {
					cm.Leading[decls[0]] = append(trailing, cm.Leading[decls[0]]...)
				}
			}
		}
	}

	parts = append(parts, renderMembersBlock(d.Members, cm, ctx))
	return parts
}

// renderMembersBlock renders a { members } block with each member using
// our custom declaration renderers.
func renderMembersBlock(members *ast.Members, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
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
		doc := renderDeclaration(decl, cm, ctx)
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
func renderVariable(d *ast.VariableDeclaration, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, renderAccess(d.Access, cm))
	}

	// let/var keyword
	if d.IsConstant {
		parts = append(parts, prettier.Text("let"), prettier.Space)
	} else {
		parts = append(parts, prettier.Text("var"), prettier.Space)
	}

	// Identifier
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	// Type annotation. If the type annotation has same-line or trailing `//`
	// line comments AND there is a value, hoist them to leading of the value:
	// otherwise the type's `//` renders followed by ` = <value>` on the same
	// doc line, and the comment swallows the assignment in the output.
	if d.TypeAnnotation != nil && d.TypeAnnotation.Type != nil {
		if d.Value != nil {
			// Move in reverse source order so the prepends produce source order:
			// trailing comments are between the type and value, same-line is on
			// the type's own line (earlier in source than trailing).
			cm.MoveTrailingToLeading(d.TypeAnnotation, d.Value)
			cm.MoveSameLineToLeading(d.TypeAnnotation, d.Value)
		}
		parts = append(parts, prettier.Text(": "), wrapWithAllComments(d.TypeAnnotation, d.TypeAnnotation.Doc(), cm))
	}

	// Transfer and value
	if d.Value != nil {
		parts = append(parts, prettier.Space)
		parts = append(parts, prettier.Text(d.Transfer.Operation.Operator()))
		// Peek before rendering since renderExpression / wrapWithComments
		// drains the value's leading comments.
		valueHasLineComment := cm.HasLeadingLineComment(d.Value)
		// Binary expressions (e.g., ?? nil-coalescing) need Indent for
		// continuation line indentation. Other expressions render directly
		// to avoid over-indenting function call arguments.
		if _, ok := d.Value.(*ast.BinaryExpression); ok {
			// Don't use wrapWithAllComments here — drained descendant
			// comments would end up inside the Indent, gaining indentation
			// that isn't stable across re-formats.
			valueDoc := wrapWithComments(d.Value, d.Value.Doc(), cm)
			parts = append(parts, prettier.Group{
				Doc: prettier.Indent{
					Doc: prettier.Concat{
						prettier.Line{},
						valueDoc,
					},
				},
			})
			var extras []prettier.Doc
			drainDescendantComments(d.Value, cm, &extras)
			for _, e := range extras {
				parts = append(parts, prettier.HardLine{}, e)
			}
		} else {
			valueDoc := renderExpression(d.Value, cm, ctx)
			if valueHasLineComment {
				parts = append(parts, prettier.HardLine{})
			} else {
				parts = append(parts, prettier.Space)
			}
			parts = append(parts, valueDoc)
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
func renderSpecialFunction(d *ast.SpecialFunctionDeclaration, cm *trivia.CommentMap, ctx *Context) prettier.Doc {
	fn := d.FunctionDeclaration
	parts := prettier.Concat{}

	// Access modifier (rare for special functions but possible)
	if fn.Access != ast.AccessNotSpecified {
		parts = append(parts, renderAccess(fn.Access, cm))
	}

	// Purity
	if fn.Purity != ast.FunctionPurityUnspecified {
		parts = append(parts, prettier.Text(fn.Purity.Keyword()), prettier.Space)
	}

	// Name (init/destroy/prepare)
	parts = append(parts, prettier.Text(fn.Identifier.Identifier))

	// Parameters — use custom rendering to preserve comments between params
	if fn.ParameterList != nil {
		paramDoc, _ := renderParameterList(fn.ParameterList, cm)
		parts = append(parts, paramDoc)
	}

	// Return type
	if fn.ReturnTypeAnnotation != nil && fn.ReturnTypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), fn.ReturnTypeAnnotation.Doc())
	}

	// Body
	if fn.FunctionBlock != nil {
		parts = append(parts, prettier.Space, renderFunctionBlock(fn.FunctionBlock, cm, ctx))
	}

	return parts
}

// renderField renders a field declaration (inside composites) with access on the same line.
func renderField(d *ast.FieldDeclaration, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, renderAccess(d.Access, cm))
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

// renderEntitlementMapping renders an entitlement mapping declaration with
// access on the same line and elements in a braced block. The upstream Doc()
// wraps in Group (fixing access modifier line) but doesn't indent elements.
func renderEntitlementMapping(d *ast.EntitlementMappingDeclaration, cm *trivia.CommentMap) prettier.Doc {
	parts := prettier.Concat{}

	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, renderAccess(d.Access, cm))
	}

	parts = append(parts, prettier.Text("entitlement"), prettier.Space)
	parts = append(parts, prettier.Text("mapping"), prettier.Space)
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	if len(d.Elements) == 0 {
		parts = append(parts, prettier.Text(" {}"))
		return parts
	}

	body := prettier.Concat{}
	for i, element := range d.Elements {
		if i > 0 {
			body = append(body, prettier.HardLine{})
		}
		if _, isNominalType := element.(*ast.NominalType); isNominalType {
			body = append(body, prettier.Text("include "), element.Doc())
		} else if element != nil {
			body = append(body, element.Doc())
		}
	}

	parts = append(parts,
		prettier.Space,
		prettier.Text("{"),
		prettier.Indent{Doc: prettier.Concat{
			prettier.HardLine{},
			body,
		}},
		prettier.HardLine{},
		prettier.Text("}"),
	)

	return parts
}
