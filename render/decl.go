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
func (r *renderer) hasBlankLineBetween(prev, curr ast.Statement) bool {
	if len(r.source) == 0 {
		return false
	}

	// Find the last byte offset of prev (including trailing comments).
	endOffset := prev.EndPosition(nil).Offset
	if trailing := r.cm.Trailing[prev]; len(trailing) > 0 {
		if tEnd := trailing[len(trailing)-1].EndPos().Offset; tEnd > endOffset {
			endOffset = tEnd
		}
	}

	// Find the first byte offset of curr (including leading comments).
	startOffset := curr.StartPosition().Offset
	if leading := r.cm.Leading[curr]; len(leading) > 0 {
		if lStart := leading[0].StartPos().Offset; lStart < startOffset {
			startOffset = lStart
		}
	}

	// Scan the source bytes between the two positions for a blank line:
	// two newlines with only whitespace between them.
	if endOffset >= startOffset || endOffset >= len(r.source) {
		return false
	}
	sawNewline := false
	for i := endOffset; i < startOffset && i < len(r.source); i++ {
		b := r.source[i]
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

// declaration dispatches to a custom renderer for the declaration type
// if we need to override the upstream Doc() behavior, otherwise falls back
// to the default Doc().
func (r *renderer) declaration(decl ast.Declaration) prettier.Doc {
	var doc prettier.Doc

	switch d := decl.(type) {
	case *ast.FunctionDeclaration:
		doc = r.function(d)
	case *ast.CompositeDeclaration:
		doc = r.composite(d)
	case *ast.InterfaceDeclaration:
		doc = r.interfaceDecl(d)
	case *ast.VariableDeclaration:
		doc = r.variable(d)
	case *ast.FieldDeclaration:
		doc = r.field(d)
	case *ast.SpecialFunctionDeclaration:
		doc = r.specialFunction(d)
	case *ast.EntitlementMappingDeclaration:
		doc = r.entitlementMapping(d)
	case *ast.TransactionDeclaration:
		doc = r.transaction(d)
	default:
		// For unknown declaration types, use upstream Doc() and drain
		// any descendant comments so they're not orphaned.
		doc = decl.Doc()
		return r.wrapAllComments(decl, doc)
	}

	// Drain any remaining descendant comments (e.g., NominalType nodes
	// inside entitlement access modifiers) that specific renderers didn't take.
	r.drainDescendants(decl, nil)

	doc = r.wrapComments(decl, doc)
	if r.hasSemicolon(decl) {
		doc = prettier.Concat{doc, prettier.Text(";")}
	}
	return doc
}

// access renders an access modifier and takes any comments attached
// to its child NominalType nodes (entitlement types). Comments are rendered
// between the access modifier and the following keyword.
func (r *renderer) access(access ast.Access) prettier.Doc {
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
		r.cm.Take(child)
	})
	return prettier.Concat{access.Doc(), prettier.Space}
}

// function renders a function declaration with access on the same line.
func (r *renderer) function(d *ast.FunctionDeclaration) prettier.Doc {
	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, r.access(d.Access))
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
		paramDoc, _ := r.parameterList(d.ParameterList)
		parts = append(parts, paramDoc)
	}

	// Return type
	if d.ReturnTypeAnnotation != nil && d.ReturnTypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), r.wrapAllComments(d.ReturnTypeAnnotation, d.ReturnTypeAnnotation.Doc()))
	}

	// Function body
	if d.FunctionBlock != nil {
		parts = append(parts, prettier.Space, r.functionBlock(d.FunctionBlock))
	}

	return parts
}

// functionBlock renders a { pre { } post { } stmts } block with
// comment interleaving between statements.
func (r *renderer) functionBlock(b *ast.FunctionBlock) prettier.Doc {
	if b.IsEmpty() {
		return prettier.Text("{}")
	}

	body := prettier.Concat{}
	needSep := false

	// Pre-conditions
	if b.PreConditions != nil && !b.PreConditions.IsEmpty() {
		condDoc := b.PreConditions.Doc(prettier.Text("pre"))
		r.drainConditions(b.PreConditions)
		body = append(body, condDoc)
		needSep = true
	}

	// Post-conditions
	if b.PostConditions != nil && !b.PostConditions.IsEmpty() {
		if needSep {
			body = append(body, prettier.HardLine{})
		}
		condDoc := b.PostConditions.Doc(prettier.Text("post"))
		r.drainConditions(b.PostConditions)
		body = append(body, condDoc)
		needSep = true
	}

	// Statements
	if b.Block != nil {
		// Drain any comments attached to the Block node itself
		// (e.g., comments inside post{} blocks in interface functions)
		leading, _, trailing := r.cm.Take(b.Block)
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
			blankBefore[i] = r.hasBlankLineBetween(stmts[i-1], stmts[i])
		}
		for i, stmt := range stmts {
			if needSep {
				body = append(body, prettier.HardLine{})
				if blankBefore[i] {
					body = append(body, prettier.HardLine{})
				}
			}
			body = append(body, r.statement(stmt))
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

// statement dispatches to custom renderers for specific statement types,
// otherwise falls back to the upstream Doc().
func (r *renderer) statement(stmt ast.Statement) prettier.Doc {
	var doc prettier.Doc
	switch s := stmt.(type) {
	case *ast.ReturnStatement:
		doc = r.wrapComments(s, r.returnStatement(s))
	case *ast.ForStatement:
		doc = r.wrapComments(s, r.forStatement(s))
	case *ast.WhileStatement:
		doc = r.wrapComments(s, r.whileStatement(s))
	case *ast.IfStatement:
		doc = r.wrapComments(s, r.ifStatement(s))
	case *ast.VariableDeclaration:
		doc = r.wrapComments(s, r.variable(s))
	case *ast.AssignmentStatement:
		doc = r.wrapComments(s, r.assignmentStatement(s))
	case *ast.ExpressionStatement:
		doc = r.wrapComments(s, r.expression(s.Expression))
	default:
		doc = r.wrapAllComments(stmt, stmt.Doc())
	}
	if r.hasSemicolon(stmt) {
		doc = prettier.Concat{doc, prettier.Text(";")}
	}
	return doc
}

// block renders the body of a block by iterating statements and
// interleaving comments. Returns the body content without braces.
func (r *renderer) block(b *ast.Block) prettier.Doc {
	if b == nil || len(b.Statements) == 0 {
		return nil
	}

	blankBefore := make([]bool, len(b.Statements))
	for i := 1; i < len(b.Statements); i++ {
		blankBefore[i] = r.hasBlankLineBetween(b.Statements[i-1], b.Statements[i])
	}

	body := prettier.Concat{}
	for i, stmt := range b.Statements {
		if i > 0 {
			body = append(body, prettier.HardLine{})
			if blankBefore[i] {
				body = append(body, prettier.HardLine{})
			}
		}
		body = append(body, r.statement(stmt))
	}
	return body
}

// blockBraces wraps a block body in { ... } with indentation.
func (r *renderer) blockBraces(b *ast.Block) prettier.Doc {
	body := r.block(b)
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

// forStatement renders a for-in loop with comment interleaving in the body.
func (r *renderer) forStatement(s *ast.ForStatement) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, prettier.Text("for "))
	parts = append(parts, prettier.Text(s.Identifier.Identifier))
	parts = append(parts, prettier.Text(" in "))
	parts = append(parts, r.expression(s.Value))
	parts = append(parts, prettier.Space)
	parts = append(parts, r.blockBraces(s.Block))

	return parts
}

// whileStatement renders a while loop with comment interleaving in the body.
func (r *renderer) whileStatement(s *ast.WhileStatement) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, prettier.Text("while "))
	parts = append(parts, r.expression(s.Test))
	parts = append(parts, prettier.Space)
	parts = append(parts, r.blockBraces(s.Block))

	return parts
}

// ifStatement renders an if/else-if/else chain with comment interleaving.
func (r *renderer) ifStatement(s *ast.IfStatement) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, prettier.Text("if "))
	parts = append(parts, r.wrapAllComments(s.Test, s.Test.Doc()))
	parts = append(parts, prettier.Space)
	parts = append(parts, r.blockBraces(s.Then))

	if s.Else != nil && len(s.Else.Statements) > 0 {
		// Check if the else block is a single if-statement (else-if chain)
		if len(s.Else.Statements) == 1 {
			if elseIf, ok := s.Else.Statements[0].(*ast.IfStatement); ok {
				parts = append(parts, prettier.Text(" else "))
				parts = append(parts, r.wrapComments(elseIf, r.ifStatement(elseIf)))
				return parts
			}
		}
		parts = append(parts, prettier.Text(" else "))
		parts = append(parts, r.blockBraces(s.Else))
	}

	return parts
}

// assignmentStatement renders target = value without the upstream's
// extra Indent wrapper that over-indents function call arguments.
func (r *renderer) assignmentStatement(s *ast.AssignmentStatement) prettier.Doc {
	parts := prettier.Concat{}

	parts = append(parts, r.expression(s.Target))
	parts = append(parts, prettier.Space)
	parts = append(parts, s.Transfer.Doc())
	parts = append(parts, prettier.Space)
	parts = append(parts, r.expression(s.Value))

	return parts
}

// returnStatement renders a return statement. For binary expressions
// (e.g., ?? nil-coalescing), wraps in Indent so continuation lines are
// indented relative to "return". Other expressions render directly to
// avoid over-indenting function call arguments.
func (r *renderer) returnStatement(s *ast.ReturnStatement) prettier.Doc {
	if s.Expression == nil {
		return prettier.Text("return")
	}

	// Binary expressions need Indent for proper continuation line indentation.
	// Drain descendant comments outside the Indent so they don't pick up
	// expression-level indentation that isn't stable across re-formats.
	if _, ok := s.Expression.(*ast.BinaryExpression); ok {
		exprDoc := r.wrapComments(s.Expression, s.Expression.Doc())
		parts := prettier.Concat{
			prettier.Text("return "),
			prettier.Indent{Doc: exprDoc},
		}
		var extras []prettier.Doc
		r.drainDescendants(s.Expression, &extras)
		for _, e := range extras {
			parts = append(parts, prettier.HardLine{}, e)
		}
		return parts
	}

	return prettier.Concat{
		prettier.Text("return "),
		r.expression(s.Expression),
	}
}

// composite renders a composite declaration (resource, struct, contract, etc.)
// with access on the same line.
func (r *renderer) composite(d *ast.CompositeDeclaration) prettier.Doc {
	// Events use a special compact format (no members block with braces)
	if d.CompositeKind == common.CompositeKindEvent {
		return r.event(d)
	}

	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, r.access(d.Access))
	}

	// Kind keyword
	parts = append(parts, prettier.Text(d.CompositeKind.Keyword()), prettier.Space)

	// Name
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	// Conformances
	parts = r.conformances(parts, d.Conformances, d.Members)

	// Members
	parts = append(parts, r.membersBlock(d.Members))
	return parts
}

// conformances appends a comma-separated conformance list (": A, B, C")
// to parts, draining each conformance's comments. Trailing comments on a
// conformance are hoisted onto the leading slot of the first member, since
// they logically describe the first field, not the conformance type.
// The upstream ast.Walk yields conformances as children, so the trivia layer
// may attach comments to them; we must drain so they don't become orphaned.
func (r *renderer) conformances(
	parts prettier.Concat,
	conformances []*ast.NominalType,
	members *ast.Members,
) prettier.Concat {
	if len(conformances) == 0 {
		return parts
	}
	parts = append(parts, prettier.Text(":"), prettier.Space)
	for i, c := range conformances {
		if i > 0 {
			parts = append(parts, prettier.Text(","), prettier.Space)
		}
		parts = append(parts, c.Doc())
		_, _, trailing := r.cm.Take(c)
		if len(trailing) > 0 && members != nil {
			decls := members.Declarations()
			if len(decls) > 0 {
				r.cm.Leading[decls[0]] = append(trailing, r.cm.Leading[decls[0]]...)
			}
		}
	}
	return parts
}

// event renders an event declaration with comments interleaved between
// parameters. The upstream EventDoc() + drain approach displaces parameter
// comments outside the closing paren.
func (r *renderer) event(d *ast.CompositeDeclaration) prettier.Doc {
	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, r.access(d.Access))
	}

	// "event Name"
	parts = append(parts, prettier.Text(d.CompositeKind.Keyword()), prettier.Space)
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	// Get parameters from the event's initializer
	initializers := d.Members.Initializers()
	if len(initializers) != 1 {
		// Fallback: no valid initializer, use upstream
		r.drainDescendants(d, nil)
		return parts
	}

	paramList := initializers[0].FunctionDeclaration.ParameterList
	paramDoc, _ := r.parameterList(paramList)
	parts = append(parts, paramDoc)

	// Drain any remaining descendant comments (type annotations, etc.)
	r.drainDescendants(d, nil)

	return parts
}

// transaction renders a transaction declaration with comment
// interleaving inside prepare/execute blocks. Without this, the default
// wrapAllComments path drains all block-interior comments and appends
// them after the closing brace.
func (r *renderer) transaction(d *ast.TransactionDeclaration) prettier.Doc {
	doc := prettier.Concat{prettier.Text("transaction")}

	// Parameters
	paramDoc, paramTrailing := r.parameterList(d.ParameterList)
	doc = append(doc, paramDoc)

	// Move trailing comments from last parameter to leading of first field
	if len(paramTrailing) > 0 && len(d.Fields) > 0 {
		r.cm.Leading[d.Fields[0]] = append(paramTrailing, r.cm.Leading[d.Fields[0]]...)
	}

	// Build body contents
	var contents []prettier.Doc

	// Fields
	for _, field := range d.Fields {
		contents = append(contents, r.declaration(field))
	}

	// Prepare block
	if d.Prepare != nil {
		contents = append(contents, r.declaration(d.Prepare))
	}

	// Pre-conditions
	if d.PreConditions != nil && !d.PreConditions.IsEmpty() {
		condDoc := d.PreConditions.Doc(prettier.Text("pre"))
		r.drainWalk(d.PreConditions.Walk)
		contents = append(contents, condDoc)
	}

	// Execute block
	if d.Execute != nil {
		contents = append(contents, r.declaration(d.Execute))
	}

	// Post-conditions
	if d.PostConditions != nil && !d.PostConditions.IsEmpty() {
		condDoc := d.PostConditions.Doc(prettier.Text("post"))
		r.drainWalk(d.PostConditions.Walk)
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

// parameterList renders a function/event parameter list with comments
// interleaved between parameters. ParameterList.Walk() yields TypeAnnotation
// nodes (not Parameter), so comments are attached to TypeAnnotation nodes.
// Returns the rendered doc and any trailing comments from the last parameter
// that the caller should place after the parameter list.
func (r *renderer) parameterList(paramList *ast.ParameterList) (prettier.Doc, []*trivia.CommentGroup) {
	if paramList == nil || len(paramList.Parameters) == 0 {
		r.drainWalk(paramList.Walk)
		return prettier.Text("()"), nil
	}

	// Collect parameters with their comments
	params := make([]paramInfo, len(paramList.Parameters))
	hasComments := false
	var pendingTrailing []*trivia.CommentGroup

	for i, param := range paramList.Parameters {
		p := paramInfo{doc: param.Doc()}
		if param.TypeAnnotation != nil {
			leading, sameLine, trailing := r.cm.Take(param.TypeAnnotation)
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
		r.drainWalk(paramList.Walk)
		return paramList.Doc(), pendingTrailing
	}

	// Drain any remaining descendant comments (e.g., on NominalType children
	// of TypeAnnotation nodes) so they don't become orphaned.
	r.drainWalk(paramList.Walk)

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

// interfaceDecl renders an interface declaration with access on the same line.
func (r *renderer) interfaceDecl(d *ast.InterfaceDeclaration) prettier.Doc {
	parts := prettier.Concat{}

	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, r.access(d.Access))
	}

	parts = append(parts, prettier.Text(d.CompositeKind.Keyword()), prettier.Space)
	parts = append(parts, prettier.Text("interface"), prettier.Space)
	parts = append(parts, prettier.Text(d.Identifier.Identifier))

	parts = r.conformances(parts, d.Conformances, d.Members)

	parts = append(parts, r.membersBlock(d.Members))
	return parts
}

// membersBlock renders a { members } block with each member using
// our custom declaration renderers.
func (r *renderer) membersBlock(members *ast.Members) prettier.Doc {
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
		body = append(body, r.declaration(decl))
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

// variable renders a variable declaration with access on the same line.
func (r *renderer) variable(d *ast.VariableDeclaration) prettier.Doc {
	parts := prettier.Concat{}

	// Access modifier
	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, r.access(d.Access))
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
			r.cm.MoveTrailingToLeading(d.TypeAnnotation, d.Value)
			r.cm.MoveSameLineToLeading(d.TypeAnnotation, d.Value)
		}
		parts = append(parts, prettier.Text(": "), r.wrapAllComments(d.TypeAnnotation, d.TypeAnnotation.Doc()))
	}

	// Transfer and value
	if d.Value != nil {
		parts = append(parts, prettier.Space)
		parts = append(parts, prettier.Text(d.Transfer.Operation.Operator()))
		// Peek before rendering since renderExpression / wrapWithComments
		// drains the value's leading comments.
		valueHasLineComment := r.cm.HasLeadingLineComment(d.Value)
		// Binary expressions (e.g., ?? nil-coalescing) need Indent for
		// continuation line indentation. Other expressions render directly
		// to avoid over-indenting function call arguments.
		if _, ok := d.Value.(*ast.BinaryExpression); ok {
			// Don't use wrapAllComments here — drained descendant
			// comments would end up inside the Indent, gaining indentation
			// that isn't stable across re-formats.
			valueDoc := r.wrapComments(d.Value, d.Value.Doc())
			parts = append(parts, prettier.Group{
				Doc: prettier.Indent{
					Doc: prettier.Concat{
						prettier.Line{},
						valueDoc,
					},
				},
			})
			var extras []prettier.Doc
			r.drainDescendants(d.Value, &extras)
			for _, e := range extras {
				parts = append(parts, prettier.HardLine{}, e)
			}
		} else {
			valueDoc := r.expression(d.Value)
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

// drainConditions drains any comments attached to Conditions' children.
func (r *renderer) drainConditions(conds *ast.Conditions) {
	conds.Walk(func(child ast.Element) {
		if child == nil {
			return
		}
		r.cm.Take(child)
		var discard []prettier.Doc
		r.drainDescendants(child, &discard)
	})
}

// specialFunction renders init/destroy/prepare declarations.
// These don't use the "fun" keyword.
func (r *renderer) specialFunction(d *ast.SpecialFunctionDeclaration) prettier.Doc {
	fn := d.FunctionDeclaration
	parts := prettier.Concat{}

	// Access modifier (rare for special functions but possible)
	if fn.Access != ast.AccessNotSpecified {
		parts = append(parts, r.access(fn.Access))
	}

	// Purity
	if fn.Purity != ast.FunctionPurityUnspecified {
		parts = append(parts, prettier.Text(fn.Purity.Keyword()), prettier.Space)
	}

	// Name (init/destroy/prepare)
	parts = append(parts, prettier.Text(fn.Identifier.Identifier))

	// Parameters — use custom rendering to preserve comments between params
	if fn.ParameterList != nil {
		paramDoc, _ := r.parameterList(fn.ParameterList)
		parts = append(parts, paramDoc)
	}

	// Return type
	if fn.ReturnTypeAnnotation != nil && fn.ReturnTypeAnnotation.Type != nil {
		parts = append(parts, prettier.Text(": "), fn.ReturnTypeAnnotation.Doc())
	}

	// Body
	if fn.FunctionBlock != nil {
		parts = append(parts, prettier.Space, r.functionBlock(fn.FunctionBlock))
	}

	return parts
}

// field renders a field declaration (inside composites) with access on the same line.
func (r *renderer) field(d *ast.FieldDeclaration) prettier.Doc {
	parts := prettier.Concat{}

	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, r.access(d.Access))
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
		parts = append(parts, prettier.Text(": "), r.wrapAllComments(d.TypeAnnotation, d.TypeAnnotation.Doc()))
	}

	return parts
}

// entitlementMapping renders an entitlement mapping declaration with
// access on the same line and elements in a braced block. The upstream Doc()
// wraps in Group (fixing access modifier line) but doesn't indent elements.
func (r *renderer) entitlementMapping(d *ast.EntitlementMappingDeclaration) prettier.Doc {
	parts := prettier.Concat{}

	if d.Access != ast.AccessNotSpecified {
		parts = append(parts, r.access(d.Access))
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
