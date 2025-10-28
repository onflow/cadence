/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package subtype_gen

import (
	"fmt"
	"go/token"
	"reflect"
	"strings"

	"github.com/dave/dst"
)

var neverType = SimpleType{
	name: "Never",
}

var interfaceType = ComplexType{}

var super = IdentifierExpression{Name: "super"}

type SubTypeCheckGenerator struct {
	config Config

	nextVarIndex int
	negate       bool

	scope            []map[Expression]string
	nestedPredicates *Predicates
}

type Config struct {
	// Prefixes and the suffixes to be added to the type-placeholder
	// to customize the type-names to match the naming conventions.
	// e.g: `PrimitiveStaticTypeString` at runtime, vs `StringType` at checking time.
	SimpleTypePrefix  string
	SimpleTypeSuffix  string
	ComplexTypePrefix string
	ComplexTypeSuffix string

	// Extra parameters to be added to the generated function signatures,
	// other than the common parameters.
	// For e.g: runtime generated function takes a `TypeConverter` as an extra argument.
	ExtraParams []ExtraParam

	// Types to be skipped from generating a subtype-check.
	// For e.g: Runtime doesn't have a `StorableType`
	SkipTypes map[string]struct{}

	// A set indicating what complex types needed to be treated as
	// non-pointer types in the generated Go code.
	NonPointerTypes map[string]struct{}

	// A mapping to customize the generated names.
	// For e.g: A field named `Foo` in the `rules.yaml`,
	// can be generated as `Bar` in the generated Go code,
	// by adding a mapping from `Foo -> Bar`.
	NameMapping map[string]string

	// List of arguments to be passed on to the `ElementType` method on array-type.
	ArrayElementTypeMethodArgs []any
}

type ExtraParam struct {
	Name    string
	Type    string
	PkgPath string
}

func NewSubTypeCheckGenerator(config Config) *SubTypeCheckGenerator {
	return &SubTypeCheckGenerator{
		config: config,
	}
}

func (gen *SubTypeCheckGenerator) pushScope() {
	gen.scope = append(gen.scope, map[Expression]string{})
}

func (gen *SubTypeCheckGenerator) popScope() {
	gen.scope = gen.scope[:len(gen.scope)-1]
}

func (gen *SubTypeCheckGenerator) addToScope(expr Expression, name string) {
	currentScope := gen.scope[len(gen.scope)-1]
	currentScope[expr] = name
}

func (gen *SubTypeCheckGenerator) findInScope(expr Expression) (string, bool) {
	for i := len(gen.scope) - 1; i >= 0; i-- {
		currentScope := gen.scope[i]
		name, ok := currentScope[expr]
		if ok {
			return name, true
		}
	}
	return "", false
}

// GenerateCheckSubTypeWithoutEqualityFunction generates the complete checkSubTypeWithoutEquality function.
func (gen *SubTypeCheckGenerator) GenerateCheckSubTypeWithoutEqualityFunction(rules []Rule) []dst.Decl {
	gen.pushScope()
	defer gen.popScope()

	checkSubTypeFunction := gen.createCheckSubTypeFunction(rules)
	return []dst.Decl{
		checkSubTypeFunction,
	}
}

// createCheckSubTypeFunction creates the main checkSubTypeWithoutEquality function
func (gen *SubTypeCheckGenerator) createCheckSubTypeFunction(rules []Rule) dst.Decl {
	// Create function parameters
	subTypeParam := &dst.Field{
		Names: []*dst.Ident{
			dst.NewIdent(subTypeVarName),
		},
		Type: gen.qualifiedTypeIdentifier(interfaceType),
	}
	gen.addToScope(
		IdentifierExpression{Name: "sub"},
		subTypeVarName,
	)

	superTypeParam := &dst.Field{
		Names: []*dst.Ident{
			dst.NewIdent(superTypeVarName),
		},
		Type: gen.qualifiedTypeIdentifier(interfaceType),
	}
	gen.addToScope(
		IdentifierExpression{Name: "super"},
		superTypeVarName,
	)

	params := make([]*dst.Field, 0)
	for _, param := range gen.config.ExtraParams {
		extraParam := &dst.Field{
			Names: []*dst.Ident{
				dst.NewIdent(param.Name),
			},
			Type: &dst.Ident{
				Name: param.Type,
				Path: param.PkgPath,
			},
		}

		params = append(params, extraParam)
	}

	params = append(params, subTypeParam, superTypeParam)

	// Create function body
	var stmts []dst.Stmt

	// Add early return for Never type
	neverTypeCheck := &dst.IfStmt{
		Cond: gen.binaryExpression(
			dst.NewIdent(subTypeVarName),
			gen.qualifiedTypeIdentifier(neverType),
			token.EQL,
		),
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				&dst.ReturnStmt{
					Results: []dst.Expr{
						gen.booleanExpression(true),
					},
				},
			},
		},
		Decs: dst.IfStmtDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				After:  dst.EmptyLine,
			},
		},
	}
	stmts = append(stmts, neverTypeCheck)

	// Create switch statement for simple types.
	switchStmtForSimpleTypes := gen.createSwitchStatementForRules(rules, true)
	stmts = append(stmts, switchStmtForSimpleTypes)

	// Create switch statement for complex types.
	switchStmtForComplexTypes := gen.createSwitchStatementForRules(rules, false)
	stmts = append(stmts, switchStmtForComplexTypes)

	// Add final return false
	stmts = append(stmts, &dst.ReturnStmt{
		Results: []dst.Expr{
			gen.booleanExpression(false),
		},
	})

	return &dst.FuncDecl{
		Name: dst.NewIdent(subtypeCheckFuncName),
		Type: &dst.FuncType{
			Params: &dst.FieldList{
				List: params,
			},
			Results: &dst.FieldList{
				List: []*dst.Field{
					{
						Type: dst.NewIdent("bool"),
					},
				},
			},
		},
		Body: &dst.BlockStmt{
			List: stmts,
		},
	}
}

// createSwitchStatementForRules creates the switch statement for superType
func (gen *SubTypeCheckGenerator) createSwitchStatementForRules(rules []Rule, forSimpleTypes bool) dst.Stmt {
	var cases []dst.Stmt

	prevSuperType := gen.expressionIgnoreNegation(super)

	var typedVariableName string
	if !forSimpleTypes {
		// For complex types, a type-switch is created for `super`.
		// So register a new variable to hold the type-value for `super`.
		// During the nested generations, `super` will refer to this variable.
		typedVariableName = gen.newTypedVariableNameFor(super)
		gen.pushScope()
		gen.addToScope(
			super,
			typedVariableName,
		)
		defer gen.popScope()
	}

	for _, rule := range rules {
		caseStmt := gen.createCaseStatementForRule(rule, forSimpleTypes)
		if caseStmt != nil {
			cases = append(cases, caseStmt)
		}
	}

	nodeDecs := dst.NodeDecs{
		Before: dst.NewLine,
		After:  dst.EmptyLine,
	}

	// For simple types, use a value-switch.
	if forSimpleTypes {
		return &dst.SwitchStmt{
			Tag: prevSuperType,
			Body: &dst.BlockStmt{
				List: cases,
			},
			Decs: dst.SwitchStmtDecorations{
				NodeDecs: nodeDecs,
			},
		}
	}

	// For complex types, use a type-switch.
	return &dst.TypeSwitchStmt{
		Assign: &dst.AssignStmt{
			Lhs: []dst.Expr{
				dst.NewIdent(typedVariableName),
			},
			Tok: token.DEFINE,
			Rhs: []dst.Expr{
				&dst.TypeAssertExpr{
					X:    prevSuperType,
					Type: dst.NewIdent("type"),
				},
			},
		},
		Body: &dst.BlockStmt{
			List: cases,
		},
		Decs: dst.TypeSwitchStmtDecorations{
			NodeDecs: nodeDecs,
		},
	}
}

// createCaseStatementForRule creates a case statement for a rule.
func (gen *SubTypeCheckGenerator) createCaseStatementForRule(rule Rule, forSimpleTypes bool) dst.Stmt {
	// Parse types
	superType := rule.SuperType

	// Skip the given types.
	// Some types are only exist during type-checking, but not at runtime. e.g: Storable type
	if _, ok := gen.config.SkipTypes[superType.Name()]; ok {
		return nil
	}

	_, isSimpleType := superType.(SimpleType)
	if isSimpleType != forSimpleTypes {
		return nil
	}

	// Generate case condition
	caseExpr := gen.parseCaseCondition(superType)

	// Generate statements for the predicate.
	bodyStmts := gen.generatePredicateStatements(rule.Predicate)

	return &dst.CaseClause{
		List: []dst.Expr{caseExpr},
		Body: bodyStmts,
		Decs: dst.CaseClauseDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				After:  dst.EmptyLine,
			},
		},
	}
}

func (gen *SubTypeCheckGenerator) booleanExpression(boolean bool) *dst.Ident {
	// use XOR with the `negate` flag.
	if boolean != gen.negate {
		return dst.NewIdent("true")
	}

	return dst.NewIdent("false")
}

// generatePredicateStatements generates statements for a given predicate.
// A predicate may generate one or more statements.
func (gen *SubTypeCheckGenerator) generatePredicateStatements(predicate Predicate) []dst.Stmt {
	nodes := gen.generatePredicate(predicate)
	if len(nodes) == 0 {
		return nil
	}

	lastIndex := len(nodes) - 1
	lastNode := nodes[lastIndex]
	remainingNodes := nodes[:lastIndex]

	var stmts []dst.Stmt

	for _, node := range remainingNodes {
		switch node := node.(type) {
		case dst.Expr:
			panic("predicate should produce at most one expression")
		case dst.Stmt:
			stmts = append(stmts, node)
		default:
			panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", node))
		}
	}

	// Make sure the last statement always returns.
	switch lastNode := lastNode.(type) {
	case dst.Expr:
		stmts = append(
			stmts,
			returnStatementWith(lastNode),
		)
	case dst.Stmt:
		// Switch statements are generated without the default return,
		// so that they can be combined with other statements.
		// If the last statement is a switch-case, then
		// append a return statement.
		stmts = append(stmts, lastNode)
		stmts = append(
			stmts,
			&dst.ReturnStmt{
				Results: []dst.Expr{
					gen.booleanExpression(false),
				},
			},
		)
	default:
		panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", lastNode))
	}

	return stmts
}

// generatePredicate recursively generates one or more expression/statement for a given predicate.
func (gen *SubTypeCheckGenerator) generatePredicate(predicate Predicate) (result []dst.Node) {

	// Pop the scope, for TypeAssertionPredicates.
	switch predicate.(type) {
	case TypeAssertionPredicate:
		defer gen.popScope()
	}

	prevNodes := gen.generatePredicateInternal(predicate)

	defer func() {
		description := predicate.Description()
		if len(result) > 0 && description != "" {
			firstNodeDecs := result[0].Decorations()
			description = descriptionAsLineComments(description)
			firstNodeDecs.Before = dst.EmptyLine
			firstNodeDecs.Start.Append(description)
		}
	}()

	// If there are no chained/nested predicates (originating from AND),
	// then add a return and complete the statements.
	// Also, if there is a negation, then do not nest, but rather early exit by adding a return.
	if gen.negate ||
		gen.nestedPredicates == nil ||
		!gen.nestedPredicates.hasMore() {

		// Add a return for switch statements, since they were generated without a return.

		if len(prevNodes) > 0 {
			lastIndex := len(prevNodes) - 1
			lastNode := prevNodes[lastIndex]

			var body *dst.BlockStmt

			// TODO: Also for if-statements?
			switch lastNode := lastNode.(type) {
			case *dst.TypeSwitchStmt:
				body = lastNode.Body

			case *dst.SwitchStmt:
				body = lastNode.Body
			}

			if body != nil {
				caseClauses := body.List
				if len(caseClauses) == 0 {
					panic("switch-statement must have at-least one cases clause")
				}

				lastCase := caseClauses[len(caseClauses)-1]
				caseClause := lastCase.(*dst.CaseClause)

				// Only add the return if the body is empty.
				// Non-empty body means a return is already present.
				if len(caseClause.Body) == 0 {
					caseClause.Body = append(
						caseClause.Body,
						&dst.ReturnStmt{
							Results: []dst.Expr{
								gen.booleanExpression(true),
							},
						},
					)
				}
			}
		}

		return prevNodes
	}

	// If there are chained/nested predicates (originating from AND),
	// then they should be generated instead of the return.

	nextPredicate := gen.nestedPredicates.next()
	nestedNodes := gen.generatePredicate(nextPredicate)

	// `combineAsAnd` indicates whether to combine the nested statement
	// using an AND operator (or otherwise as OR operator)
	_, combineAsAnd := nextPredicate.(*AndPredicate)
	if gen.negate {
		combineAsAnd = !combineAsAnd
	}

	// If previous nodes are nil, simply return the nested nodes.
	if len(prevNodes) == 0 {
		return nestedNodes
	}

	// If not (both previous nodes and nested nodes exist),
	// then merge the two set of nodes.
	// Merging happens via the last statement of the previous nodes.

	lastIndex := len(prevNodes) - 1
	lastNode := prevNodes[lastIndex]

	// Add all nodes upto the last-1.
	// Use the last node to merge the nested nodes.
	result = append(result, prevNodes[:lastIndex]...)

	switch lastNode := lastNode.(type) {
	case nil:
		panic(fmt.Errorf("generated node is nil"))

	case dst.Expr:
		result = gen.mergeNestedNodesWithExpression(
			result,
			lastNode,
			nestedNodes,
		)

	case *dst.TypeSwitchStmt:
		result = gen.mergeNestedNodeWithSwitchStatement(
			result,
			lastNode,
			lastNode.Body,
			nestedNodes,
			combineAsAnd,
		)

	case *dst.SwitchStmt:
		result = gen.mergeNestedNodeWithSwitchStatement(
			result,
			lastNode,
			lastNode.Body,
			nestedNodes,
			combineAsAnd,
		)

	case *dst.RangeStmt:
		stmts := gen.mergeNodesAsStatements(nestedNodes, combineAsAnd)
		result = append(result, lastNode)
		for _, stmt := range stmts {
			result = append(result, stmt)
		}

	case *dst.ReturnStmt:
		// Merging with a return statement:
		//  - If the return value is `false`, then no point of merging,
		//    since the result will always be false. This should never occur.
		//  - If the return value is `true`, then we can simply ignore the return,
		//    and append the nested statements.
		stmts := gen.mergeNodesAsStatements(nestedNodes, combineAsAnd)
		if len(lastNode.Results) != 1 {
			panic(fmt.Errorf("error generating predicate AST: expected only one return value"))
		}

		returnValue, ok := lastNode.Results[0].(*dst.Ident)
		if !ok || returnValue.Name != "true" {
			panic(fmt.Errorf("error generating predicate AST: expected `return true` statement"))
		}

		// Drop the return, and append the nested statements.
		for _, stmt := range stmts {
			result = append(result, stmt)
		}

	default:
		panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", lastNode))
	}

	return
}

func descriptionAsLineComments(description string) string {
	return strings.ReplaceAll(description, "#", "//")
}

func (gen *SubTypeCheckGenerator) mergeNestedNodesWithExpression(
	result []dst.Node,
	expr dst.Expr,
	nestedNodes []dst.Node,
) []dst.Node {
	// Previous nodes ended with an expression.
	// Then either merge the nested ones using a:
	//  - Binary expression: for nested expressions.
	//  - If statement: for nested statement.

	var stmts []dst.Stmt

	for _, nestedNode := range nestedNodes {
		switch nestedNode := nestedNode.(type) {
		case nil:
			// Skip empty node.
			// Ideally shouldn't reach here.

		case dst.Expr:
			expr = gen.binaryExpression(
				expr,
				nestedNode,
				token.LAND,
			)
		case dst.Stmt:
			stmts = append(stmts, nestedNode)

		default:
			panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", expr))
		}
	}

	if stmts == nil {
		// Only expressions were generated.
		result = append(result, expr)
	} else {
		// There are both expressions and statements generated.
		// Convert the conditional-expression into a statement,
		// by putting them as the condition of an if-statement.
		ifStmt := mergeUsingIfStatement(expr, stmts)

		result = append(
			result,
			ifStmt,
		)
	}
	return result
}

func mergeUsingIfStatement(expr dst.Expr, stmts []dst.Stmt) *dst.IfStmt {
	ifStmt := &dst.IfStmt{
		Cond: expr,
		Body: &dst.BlockStmt{
			List: stmts,
		},
		Decs: dst.IfStmtDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				After:  dst.EmptyLine,
			},
		},
	}

	conditionDecs := expr.Decorations()
	if conditionDecs != nil {
		ifStmt.Decorations().Start = conditionDecs.Start
		conditionDecs.Start = nil
	}

	return ifStmt
}

func (gen *SubTypeCheckGenerator) mergeNestedNodeWithSwitchStatement(
	combinedNodes []dst.Node,
	switchStmt dst.Stmt,
	switchStmtBody *dst.BlockStmt,
	nestedNodes []dst.Node,
	combineAsAnd bool,
) []dst.Node {
	stmts := gen.mergeNodesAsStatements(nestedNodes, combineAsAnd)

	caseClauses := switchStmtBody.List
	if len(caseClauses) == 0 {
		panic("switch-statement must have at-least one cases clause")
	}

	lastCase := caseClauses[len(caseClauses)-1]
	caseClause := lastCase.(*dst.CaseClause)

	if len(caseClause.Body) == 0 {
		// If the case statement is empty, then include the nested nodes
		// inside the case-body.
		caseClause.Body = append(caseClause.Body, stmts...)
		combinedNodes = append(combinedNodes, switchStmt)
	} else {
		// If the case statement is non-empty, that means it's probably a rule with negation,
		// and a return must have been added as an early-exit strategy.
		// Verify this.
		lastStmtInsideCase := caseClause.Body[len(caseClause.Body)-1]
		if _, isReturnStmt := lastStmtInsideCase.(*dst.ReturnStmt); !isReturnStmt {
			panic("last statement of a case-clause must be a return statement")
		}

		// Then the nested-conditions must be included after the switch statement.
		combinedNodes = append(combinedNodes, switchStmt)
		for _, stmt := range stmts {
			combinedNodes = append(combinedNodes, stmt)
		}
	}
	return combinedNodes
}

func (gen *SubTypeCheckGenerator) mergeNodesAsStatements(nodes []dst.Node, combineAsAnd bool) []dst.Stmt {
	var conditionalExpr dst.Expr
	var stmts []dst.Stmt

	var operator token.Token
	if combineAsAnd {
		operator = token.LAND
	} else {
		operator = token.LOR
	}

	for _, nestedNode := range nodes {
		switch nestedNode := nestedNode.(type) {
		case nil:
			// Skip empty node.
			// Ideally shouldn't reach here.

		case dst.Expr:
			if conditionalExpr == nil {
				conditionalExpr = nestedNode
				continue
			}

			conditionalExpr = gen.binaryExpression(
				conditionalExpr,
				nestedNode,
				operator,
			)

		case dst.Stmt:
			stmts = append(stmts, nestedNode)

		default:
			panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", nestedNode))
		}
	}

	// Only expressions were generated.
	if stmts == nil {
		return []dst.Stmt{
			returnStatementWith(conditionalExpr),
		}
	}

	// Both expressions and statements were generated.
	if conditionalExpr != nil {
		if combineAsAnd {
			return []dst.Stmt{
				mergeUsingIfStatement(conditionalExpr, stmts),
			}
		} else {
			combined := []dst.Stmt{
				mergeUsingIfStatement(
					conditionalExpr,
					[]dst.Stmt{
						&dst.ReturnStmt{
							Results: []dst.Expr{
								gen.booleanExpression(true),
							},
						},
					},
				),
			}

			// TODO: Does the order matter?
			combined = append(combined, stmts...)
			return combined
		}
	}

	// Only statements were generated
	return stmts
}

func returnStatementWith(returnValue dst.Expr) *dst.ReturnStmt {
	returnStmt := &dst.ReturnStmt{
		Results: []dst.Expr{
			returnValue,
		},
	}

	returnValueDecs := returnValue.Decorations()
	comments := returnValueDecs.Start

	if len(comments) > 0 {
		returnStmt.Decorations().Start.Append(comments...)
		returnValueDecs.Start.Clear()
		returnStmt.Decs.Before = dst.EmptyLine
	}
	return returnStmt
}

// generatePredicate recursively generates one or more expression/statement for a given predicate.
func (gen *SubTypeCheckGenerator) generatePredicateInternal(predicate Predicate) (result []dst.Node) {
	switch p := predicate.(type) {
	case AlwaysPredicate:
		return []dst.Node{
			gen.booleanExpression(true),
		}

	case NeverPredicate:
		return []dst.Node{
			gen.booleanExpression(false),
		}

	case IsResourcePredicate:
		return gen.isResourcePredicate(p)

	case IsAttachmentPredicate:
		return gen.isAttachmentPredicate(p)

	case IsHashableStructPredicate:
		return gen.isHashableStructPredicate(p)

	case IsStorablePredicate:
		return gen.isStorablePredicate(p)

	case NotPredicate:
		return gen.notPredicate(p)

	case AndPredicate:
		return gen.andPredicate(p)

	case OrPredicate:
		return gen.orPredicate(p)

	case EqualsPredicate:
		return gen.equalsPredicate(p)

	case DeepEqualsPredicate:
		return gen.deepEqualsPredicate(p)

	case SubtypePredicate:
		return gen.isSubTypePredicate(p)

	case PermitsPredicate:
		return gen.permitsPredicate(p)

	case TypeAssertionPredicate:
		return gen.typeAssertion(p)

	case SetContainsPredicate:
		return gen.setContains(p)

	case IsIntersectionSubsetPredicate:
		return gen.isIntersectionSubset(p)

	case ReturnCovariantPredicate:
		return gen.returnsCovariantCheck(p)

	case IsParameterizedSubtypePredicate:
		return gen.isParameterizedSubtype(p)

	case ForAllPredicate:
		return gen.forAllPredicate(p)

	default:
		panic(fmt.Errorf("unsupported predicate: %T", p))
	}
}

func (gen *SubTypeCheckGenerator) isHashableStructPredicate(predicate IsHashableStructPredicate) []dst.Node {
	args := gen.extraArguments()

	args = append(
		args,
		gen.expressionIgnoreNegation(predicate.Expression),
	)

	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("IsHashableStructType"),
			args...,
		),
	}
}

func (gen *SubTypeCheckGenerator) isStorablePredicate(predicate IsStorablePredicate) []dst.Node {
	function := &dst.SelectorExpr{
		X:   gen.expressionIgnoreNegation(predicate.Expression),
		Sel: dst.NewIdent("IsStorable"),
	}

	argument := &dst.CompositeLit{
		Type: &dst.MapType{
			Key: &dst.StarExpr{
				X: dst.NewIdent("Member"),
			},
			Value: dst.NewIdent("bool"),
		},
	}

	return []dst.Node{
		gen.callExpression(function, argument),
	}
}

func (gen *SubTypeCheckGenerator) notPredicate(p NotPredicate) []dst.Node {
	prevNegate := gen.negate
	defer func() {
		gen.negate = prevNegate
	}()

	// negate the current negation.
	gen.negate = !gen.negate

	innerPredicateNodes := gen.generatePredicate(p.Predicate)
	if len(innerPredicateNodes) != 1 {
		panic("can only handle one node in `not` predicate")
	}

	innerPredicateExpr := innerPredicateNodes[0]

	return []dst.Node{
		innerPredicateExpr,
	}
}

func (gen *SubTypeCheckGenerator) andPredicate(p AndPredicate) []dst.Node {
	if gen.negate {
		// Inversion of `AND` is `OR`
		return gen.generateOrPredicate(p.Predicates)
	}

	return gen.generateAndPredicate(p.Predicates)
}

func (gen *SubTypeCheckGenerator) generateAndPredicate(predicates []Predicate) (result []dst.Node) {
	prevPredicateChain := gen.nestedPredicates
	gen.nestedPredicates = NewPredicateChain(predicates)
	defer func() {
		gen.nestedPredicates = prevPredicateChain
	}()

	var exprs []dst.Expr

	for gen.nestedPredicates.hasMore() {
		predicate := gen.nestedPredicates.next()
		generatedPredicatedNodes := gen.generatePredicate(predicate)

		for _, node := range generatedPredicatedNodes {
			switch node := node.(type) {
			case dst.Stmt:
				// Add statements as-is, since they are all conditional-statements.
				result = append(result, node)
			case dst.Expr:
				exprs = append(exprs, node)
			default:
				panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", node))
			}
		}
	}

	var binaryExpr dst.Expr
	for _, expr := range exprs {
		if binaryExpr == nil {
			binaryExpr = expr
			continue
		}

		expr.Decorations().Before = dst.NewLine

		// Don't negate again here (i.e: don't use `binaryExpression` method),
		// since negation is already done before calling this method `generateAndPredicate`.
		binaryExpr = &dst.BinaryExpr{
			X:  binaryExpr,
			Op: token.LAND,
			Y:  expr,
		}
	}

	if binaryExpr != nil {
		result = append(result, binaryExpr)
	}

	return result
}

func (gen *SubTypeCheckGenerator) orPredicate(p OrPredicate) []dst.Node {
	if gen.negate {
		// Inversion of `OR` is `AND`
		return gen.generateAndPredicate(p.Predicates)
	}

	return gen.generateOrPredicate(p.Predicates)
}

func (gen *SubTypeCheckGenerator) generateOrPredicate(predicates []Predicate) (result []dst.Node) {
	prevPredicateChain := gen.nestedPredicates
	gen.nestedPredicates = nil
	defer func() {
		gen.nestedPredicates = prevPredicateChain
	}()

	var exprs []dst.Expr
	var prevTypeSwitch *dst.TypeSwitchStmt

	for _, predicate := range predicates {
		generatedPredicatedNodes := gen.generatePredicate(predicate)

		for _, node := range generatedPredicatedNodes {
			switch node := node.(type) {
			case *dst.TypeSwitchStmt:
				if prevTypeSwitch != nil &&
					reflect.DeepEqual(prevTypeSwitch.Assign, node.Assign) {
					mergeTypeSwitches(prevTypeSwitch, node)
				} else {
					prevTypeSwitch = node
					result = append(result, node)
				}
			case dst.Stmt:
				// Add statements as-is, since they are all conditional-statements.
				result = append(result, node)
			case dst.Expr:
				exprs = append(exprs, node)
			default:
				panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", node))
			}
		}
	}

	var binaryExpr dst.Expr
	for _, expr := range exprs {
		if binaryExpr == nil {
			binaryExpr = expr
			continue
		}

		expr.Decorations().Before = dst.NewLine

		// Don't negate again here (i.e: don't use `binaryExpression` method),
		// since negation is already done before calling this method `generateOrPredicate`.
		binaryExpr = &dst.BinaryExpr{
			X:  binaryExpr,
			Op: token.LOR,
			Y:  expr,
		}
	}

	if binaryExpr != nil {
		result = append(result, binaryExpr)
	}

	return result
}

func mergeTypeSwitches(existingTypeSwitch, newTypeSwitch *dst.TypeSwitchStmt) {
	existingTypeSwitch.Body.List = append(
		existingTypeSwitch.Body.List,
		newTypeSwitch.Body.List...,
	)
}

func (gen *SubTypeCheckGenerator) isAttachmentPredicate(predicate IsAttachmentPredicate) []dst.Node {
	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("isAttachmentType"),
			gen.expressionIgnoreNegation(predicate.Expression),
		),
	}
}

func (gen *SubTypeCheckGenerator) isResourcePredicate(predicate IsResourcePredicate) []dst.Node {
	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("IsResourceType"),
			gen.expressionIgnoreNegation(predicate.Expression),
		),
	}
}

// equalsPredicate generates AST for equals predicate
func (gen *SubTypeCheckGenerator) equalsPredicate(equals EqualsPredicate) []dst.Node {
	switch target := equals.Target.(type) {
	case TypeExpression, MemberExpression, IdentifierExpression:
		return []dst.Node{
			gen.binaryExpression(
				gen.expressionIgnoreNegation(equals.Source),
				gen.expressionIgnoreNegation(target),
				token.EQL,
			),
		}
	case OneOfExpression:
		exprs := target.Expressions
		// If there's only one type to match, use `==`.
		if len(exprs) == 1 {
			expr := exprs[0]
			return []dst.Node{
				gen.binaryExpression(
					gen.expressionIgnoreNegation(equals.Source),
					gen.expressionIgnoreNegation(expr),
					token.EQL,
				),
			}
		}

		// Otherwise, if there are more than one type, then generate a switch-case.
		var cases []dst.Expr
		for _, expr := range exprs {
			generatedExpr := gen.expressionIgnoreNegation(expr)
			generatedExpr.Decorations().After = dst.NewLine
			cases = append(cases, generatedExpr)
		}

		// Generate switch expression
		caseClauses := []dst.Stmt{
			&dst.CaseClause{
				List: cases,
				Decs: dst.CaseClauseDecorations{
					NodeDecs: dst.NodeDecs{
						Before: dst.NewLine,
						After:  dst.NewLine,
					},
				},
			},
		}

		return []dst.Node{
			&dst.SwitchStmt{
				Tag: gen.expressionIgnoreNegation(equals.Source),
				Body: &dst.BlockStmt{
					List: caseClauses,
				},
				Decs: dst.SwitchStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Before: dst.NewLine,
						After:  dst.EmptyLine,
					},
				},
			},
		}
	default:
		panic(fmt.Errorf("unknown target type %t in `equals` rule", target))
	}
}

// deepEqualsPredicate generates AST for deep-equals predicate
func (gen *SubTypeCheckGenerator) deepEqualsPredicate(equals DeepEqualsPredicate) []dst.Node {
	switch target := equals.Target.(type) {
	case TypeExpression, MemberExpression, IdentifierExpression:
		return []dst.Node{
			gen.callExpression(
				dst.NewIdent("deepEquals"),
				gen.expressionIgnoreNegation(equals.Source),
				gen.expressionIgnoreNegation(target),
			),
		}
	default:
		panic(fmt.Errorf("unknown target type %t in `equals` rule", target))
	}
}

func (gen *SubTypeCheckGenerator) expressionIgnoreNegation(expr Expression) dst.Expr {
	return gen.expression(expr, true)
}

func (gen *SubTypeCheckGenerator) expression(expr Expression, ignoreNegation bool) dst.Expr {
	if ignoreNegation {
		// If negation to be ignored,
		// set the `negate` flag to false.
		prevNegation := gen.negate
		defer func() {
			gen.negate = prevNegation
		}()
		gen.negate = false
	}

	name, ok := gen.findInScope(expr)
	if ok {
		return &dst.Ident{Name: name}
	}

	switch expr := expr.(type) {
	case IdentifierExpression:
		return gen.newIdentifier(expr.Name)

	case TypeExpression:
		return gen.qualifiedTypeIdentifier(expr.Type)

	case MemberExpression:
		selectorExpr := &dst.SelectorExpr{
			X:   gen.expressionIgnoreNegation(expr.Parent),
			Sel: gen.newIdentifier(expr.MemberName),
		}

		switch expr.MemberName {
		case "ElementType":
			var args []dst.Expr
			for _, arg := range gen.config.ArrayElementTypeMethodArgs {
				args = append(args, dst.NewIdent(fmt.Sprint(arg)))
			}

			return gen.callExpression(selectorExpr, args...)

		case "EffectiveInterfaceConformanceSet",
			"EffectiveIntersectionSet",
			"BaseType",
			"TypeArguments":
			return gen.callExpression(selectorExpr)
		}

		return selectorExpr

	default:
		panic(fmt.Errorf("unsupported expression to convert to string: %t", expr))
	}
}

// isSubTypePredicate generates AST for subtype conditions
func (gen *SubTypeCheckGenerator) isSubTypePredicate(subtype SubtypePredicate) []dst.Node {
	switch superType := subtype.Super.(type) {
	case IdentifierExpression, TypeExpression, MemberExpression:
		args := gen.isSubTypeMethodArguments(subtype.Sub, superType)
		return []dst.Node{
			gen.callExpression(
				dst.NewIdent("IsSubType"),
				args...,
			),
		}
	case OneOfExpression:
		var conditions []dst.Expr
		for _, expr := range superType.Expressions {
			args := gen.isSubTypeMethodArguments(subtype.Sub, expr)
			conditions = append(
				conditions,
				gen.callExpression(
					dst.NewIdent("IsSubType"),
					args...,
				),
			)
		}

		if len(conditions) == 0 {
			return []dst.Node{
				gen.booleanExpression(false),
			}
		}

		result := conditions[0]
		for i := 1; i < len(conditions); i++ {
			nextCondition := conditions[i]
			result = gen.binaryExpression(
				result,
				nextCondition,
				token.LOR,
			)
		}

		return []dst.Node{result}
	default:
		panic(fmt.Errorf("unknown super type `%T` in `subtype` rule", superType))
	}
}

func (gen *SubTypeCheckGenerator) isSubTypeMethodArguments(subType, superType Expression) []dst.Expr {
	args := gen.extraArguments()

	args = append(args,
		gen.expressionIgnoreNegation(subType),
		gen.expressionIgnoreNegation(superType),
	)

	return args
}

func (gen *SubTypeCheckGenerator) extraArguments() []dst.Expr {
	args := make([]dst.Expr, 0)
	for _, param := range gen.config.ExtraParams {
		args = append(
			args,
			dst.NewIdent(param.Name),
		)
	}
	return args
}

// qualifiedTypeIdentifier creates a qualified type identifier,
// by prepending the package-qualifier,prefix, and appending `Type` suffix.
func (gen *SubTypeCheckGenerator) qualifiedTypeIdentifier(typ Type) dst.Expr {
	var typeName string
	if _, ok := typ.(SimpleType); ok {
		typeName = gen.config.SimpleTypePrefix + typ.Name() + gen.config.SimpleTypeSuffix
	} else {
		typeName = gen.config.ComplexTypePrefix + typ.Name() + gen.config.ComplexTypeSuffix
	}

	return dst.NewIdent(typeName)
}

// parseCaseCondition parses a case condition to AST using Cadence types
func (gen *SubTypeCheckGenerator) parseCaseCondition(superType Type) dst.Expr {
	typeName := gen.qualifiedTypeIdentifier(superType)
	switch superType.(type) {
	case SimpleType:
		// For simple types, use the type directly
		return typeName
	default:
		if _, ok := gen.config.NonPointerTypes[superType.Name()]; ok {
			return typeName
		}

		return &dst.StarExpr{
			X: typeName,
		}
	}
}

func (gen *SubTypeCheckGenerator) permitsPredicate(permits PermitsPredicate) []dst.Node {
	args := []dst.Expr{
		gen.expressionIgnoreNegation(permits.Super),
		gen.expressionIgnoreNegation(permits.Sub),
	}

	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("PermitsAccess"),
			args...,
		),
	}
}

func (gen *SubTypeCheckGenerator) typeAssertion(typeAssertion TypeAssertionPredicate) []dst.Node {

	source := typeAssertion.Source

	sourceExpr := gen.expressionIgnoreNegation(source)

	// A type-switch is created for the source-expression.
	// So register a new variable to hold the typed-value of the source expression.
	// During the nested generations, re-using the source expression will refer to this variable.
	typedVariableName := gen.newTypedVariableNameFor(source)
	gen.pushScope()
	gen.addToScope(
		source,
		typedVariableName,
	)
	// Note: Popping scope must be done after visiting all nested predicates.
	// Therefore, it is done in `generatePredicate` method.

	// Generate case condition
	caseExpr := gen.parseCaseCondition(typeAssertion.Type)
	caseClause := &dst.CaseClause{
		List: []dst.Expr{caseExpr},
		Decs: dst.CaseClauseDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				After:  dst.NewLine,
			},
		},
	}

	caseClauses := []dst.Stmt{
		caseClause,
	}

	if gen.negate {
		panic(fmt.Errorf("negating a type assertion is not supported yet"))
	}

	statement := &dst.TypeSwitchStmt{
		Assign: &dst.AssignStmt{
			Lhs: []dst.Expr{
				dst.NewIdent(typedVariableName),
			},
			Tok: token.DEFINE,
			Rhs: []dst.Expr{
				&dst.TypeAssertExpr{
					X:    sourceExpr,
					Type: dst.NewIdent("type"),
				},
			},
		},
		Body: &dst.BlockStmt{
			List: caseClauses,
		},
		Decs: dst.TypeSwitchStmtDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				After:  dst.EmptyLine,
			},
		},
	}

	return []dst.Node{
		statement,
	}
}

func (gen *SubTypeCheckGenerator) binaryExpression(lhs, rhs dst.Expr, operator token.Token) *dst.BinaryExpr {
	if gen.negate {
		operator = negateOperator(operator)
	}

	switch operator {
	case token.LAND, token.LOR:
		rhs.Decorations().Before = dst.NewLine
	}

	return &dst.BinaryExpr{
		X:  lhs,
		Op: operator,
		Y:  rhs,
	}
}

func negateOperator(operator token.Token) token.Token {
	switch operator {
	case token.EQL:
		return token.NEQ
	case token.LAND:
		return token.LOR
	case token.LOR:
		return token.LAND
	default:
		panic(fmt.Errorf("unknown operator %#q", operator))
	}
}

func (gen *SubTypeCheckGenerator) callExpression(invokedExpr dst.Expr, args ...dst.Expr) (expr dst.Expr) {
	expr = &dst.CallExpr{
		Fun:  invokedExpr,
		Args: args,
	}

	if gen.negate {
		expr = &dst.UnaryExpr{
			Op: token.NOT,
			X:  &dst.ParenExpr{X: expr},
		}
	}

	return expr
}

func (gen *SubTypeCheckGenerator) newTypedVariableNameFor(source Expression) string {
	switch expr := source.(type) {
	case IdentifierExpression:
		// prepend "type" prefix to the camel-cased name.
		name := expr.Name

		// For better readability, specially handle known keywords "sub" and "super"
		var camelCaseName string
		switch name {
		case "sub":
			camelCaseName = "SubType"
		case "super":
			camelCaseName = "SuperType"
		default:
			camelCaseName = strings.ToUpper(string(name[0])) + name[1:]
		}

		return fmt.Sprintf("typed%s", camelCaseName)

	case MemberExpression:
		return gen.newTypedVariableNameFor(expr.Parent) + expr.MemberName

	default:
		name := fmt.Sprintf("v%d", gen.nextVarIndex)
		gen.nextVarIndex++
		return name
	}
}

func (gen *SubTypeCheckGenerator) setContains(p SetContainsPredicate) []dst.Node {
	selectExpr := &dst.SelectorExpr{
		X:   gen.expressionIgnoreNegation(p.Set),
		Sel: dst.NewIdent("Contains"),
	}

	args := []dst.Expr{
		gen.expressionIgnoreNegation(p.Element),
	}

	callExpr := gen.callExpression(selectExpr, args...)

	return []dst.Node{
		callExpr,
	}
}

func (gen *SubTypeCheckGenerator) isIntersectionSubset(p IsIntersectionSubsetPredicate) []dst.Node {
	args := []dst.Expr{
		gen.expressionIgnoreNegation(p.Super),
		gen.expressionIgnoreNegation(p.Sub),
	}

	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("IsIntersectionSubset"),
			args...,
		),
	}
}

func (gen *SubTypeCheckGenerator) returnsCovariantCheck(p ReturnCovariantPredicate) []dst.Node {
	args := []dst.Expr{
		gen.expressionIgnoreNegation(p.Source),
		gen.expressionIgnoreNegation(p.Target),
	}

	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("AreReturnsCovariant"),
			args...,
		),
	}
}

func (gen *SubTypeCheckGenerator) isParameterizedSubtype(p IsParameterizedSubtypePredicate) []dst.Node {
	args := []dst.Expr{
		gen.expressionIgnoreNegation(p.Sub),
		gen.expressionIgnoreNegation(p.Super),
	}

	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("IsParameterizedSubType"),
			args...,
		),
	}
}

func (gen *SubTypeCheckGenerator) newIdentifier(name string) *dst.Ident {
	if mappedName, ok := gen.config.NameMapping[name]; ok {
		return dst.NewIdent(mappedName)
	}

	return dst.NewIdent(name)
}

func (gen *SubTypeCheckGenerator) forAllPredicate(p ForAllPredicate) []dst.Node {
	sourceListVarName := gen.newTypedVariableNameFor(p.Source)
	targetListVarName := gen.newTypedVariableNameFor(p.Target)

	sourceListVar := &dst.AssignStmt{
		Lhs: []dst.Expr{
			gen.newIdentifier(sourceListVarName),
		},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{
			gen.expressionIgnoreNegation(p.Source),
		},
	}

	targetListVar := &dst.AssignStmt{
		Lhs: []dst.Expr{
			gen.newIdentifier(targetListVarName),
		},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{
			gen.expressionIgnoreNegation(p.Target),
		},
	}

	// Generate:
	//   if len(sourceList) != len(targetList) {
	//       return false
	//   }
	lengthCheck := &dst.IfStmt{
		Cond: &dst.BinaryExpr{
			X: &dst.CallExpr{
				Fun: dst.NewIdent("len"),
				Args: []dst.Expr{
					gen.newIdentifier(sourceListVarName),
				},
			},
			Op: token.NEQ,
			Y: &dst.CallExpr{
				Fun: dst.NewIdent("len"),
				Args: []dst.Expr{
					gen.newIdentifier(targetListVarName),
				},
			},
		},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				&dst.ReturnStmt{
					Results: []dst.Expr{
						gen.booleanExpression(false),
					},
				},
			},
		},
	}

	// Generate:
	//   target := targetList[i]
	targetElement := &dst.AssignStmt{
		Lhs: []dst.Expr{
			gen.newIdentifier(targetVarName),
		},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{
			&dst.IndexExpr{
				X:     gen.newIdentifier(targetListVarName),
				Index: dst.NewIdent("i"),
			},
		},
	}

	// The inner predicate is for each element for the source.
	// Therefore, stop combining the inner predicates with the nested-predicates of the parent.
	prevPredicateChain := gen.nestedPredicates
	gen.nestedPredicates = nil
	defer func() {
		gen.nestedPredicates = prevPredicateChain
	}()

	// We want to return if the inner predicate fails.
	// Therefore, negate the inner predicate when generating.
	innerPredicates := gen.generateNegatedPredicate(p.Predicate)

	if len(innerPredicates) > 1 {
		panic(fmt.Errorf(
			"only support one node for the inner predicate. found %d",
			len(innerPredicates),
		))
	}

	loopStmts := []dst.Stmt{
		targetElement,
	}

	innerPredicate := innerPredicates[0]

	switch innerPredicate := innerPredicate.(type) {
	case dst.Expr:
		ifMismatch := &dst.IfStmt{
			Cond: innerPredicate,
			Body: &dst.BlockStmt{
				List: []dst.Stmt{
					&dst.ReturnStmt{
						Results: []dst.Expr{
							gen.booleanExpression(false),
						},
					},
				},
			},
		}

		loopStmts = append(loopStmts, ifMismatch)
	default:
		panic(fmt.Errorf(
			"only support expressions for the inner predicate. found %T",
			innerPredicate,
		))
	}

	forLoop := &dst.RangeStmt{
		Key:   dst.NewIdent("i"),
		Value: gen.newIdentifier(sourceVarName),
		Tok:   token.DEFINE,
		X:     gen.newIdentifier(sourceListVarName),
		Body: &dst.BlockStmt{
			List: loopStmts,
		},
		Decs: dst.RangeStmtDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.EmptyLine,
				After:  dst.EmptyLine,
			},
		},
	}

	ifAllMatches := &dst.ReturnStmt{
		Results: []dst.Expr{
			gen.booleanExpression(true),
		},
	}

	return []dst.Node{
		sourceListVar,
		targetListVar,
		lengthCheck,
		forLoop,
		ifAllMatches,
	}

}

func (gen *SubTypeCheckGenerator) generateNegatedPredicate(predicate Predicate) []dst.Node {
	prevNegate := gen.negate
	gen.negate = true
	defer func() {
		gen.negate = prevNegate
	}()

	return gen.generatePredicate(predicate)
}
