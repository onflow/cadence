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

const (
	commonsPkgPath = "github.com/onflow/cadence/common"
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
func (gen *SubTypeCheckGenerator) GenerateCheckSubTypeWithoutEqualityFunction(rules RulesFile) []dst.Decl {
	gen.pushScope()
	defer gen.popScope()

	checkSubTypeFunction := gen.createCheckSubTypeFunction(rules.Rules)
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

	simpleTypeRules, complexTypeRules, defaultRule := gen.separateRules(rules)

	// Create switch statement for simple types.
	switchStmtForSimpleTypes := gen.createSwitchStatementForRules(simpleTypeRules)
	if switchStmtForSimpleTypes != nil {
		stmts = append(stmts, switchStmtForSimpleTypes)
	}

	// Create a type-switch for complex types.
	switchStmtForComplexTypes := gen.createTypeSwitchStatementForRules(complexTypeRules)
	if switchStmtForComplexTypes != nil {
		stmts = append(stmts, switchStmtForComplexTypes)
	}

	if defaultRule != nil {
		// If there is a default rule, then generate it.
		// `generatePredicateStatements` always ensures the last statement is a return.
		// So we don't have to explicitly add a return here.
		defaultStmts := gen.generatePredicateStatements(defaultRule.Predicate)
		stmts = append(stmts, defaultStmts...)
	} else {
		// If there are no default rules, then add a final `return false`
		stmts = append(stmts, &dst.ReturnStmt{
			Results: []dst.Expr{
				gen.booleanExpression(false),
			},
		})
	}

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
func (gen *SubTypeCheckGenerator) createSwitchStatementForRules(rules []Rule) dst.Stmt {
	prevSuperType := gen.expressionIgnoreNegation(super)

	var cases []dst.Stmt
	for _, rule := range rules {
		caseStmt := gen.createCaseStatementForRule(rule)
		if caseStmt != nil {
			cases = append(cases, caseStmt)
		}
	}

	if cases == nil {
		return nil
	}

	// For simple types, use a value-switch.
	return &dst.SwitchStmt{
		Tag: prevSuperType,
		Body: &dst.BlockStmt{
			List: cases,
		},
		Decs: dst.SwitchStmtDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				After:  dst.EmptyLine,
			},
		},
	}
}

// createSwitchStatementForRules creates the switch statement for superType
func (gen *SubTypeCheckGenerator) createTypeSwitchStatementForRules(rules []Rule) dst.Stmt {
	prevSuperType := gen.expressionIgnoreNegation(super)

	// For complex types, a type-switch is created for `super`.
	// So register a new variable to hold the type-value for `super`.
	// During the nested generations, `super` will refer to this variable.
	typedVariableName := gen.newTypedVariableNameFor(super)
	gen.pushScope()
	gen.addToScope(
		super,
		typedVariableName,
	)
	defer gen.popScope()

	var cases []dst.Stmt
	for _, rule := range rules {
		caseStmt := gen.createCaseStatementForRule(rule)
		if caseStmt != nil {
			cases = append(cases, caseStmt)
		}
	}

	if cases == nil {
		return nil
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
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				After:  dst.EmptyLine,
			},
		},
	}
}

// createCaseStatementForRule creates a case statement for a rule.
func (gen *SubTypeCheckGenerator) createCaseStatementForRule(rule Rule) dst.Stmt {
	// Parse types
	superType := rule.SuperType

	// Skip the given types.
	// Some types are only exist during type-checking, but not at runtime. e.g: Storable type
	if _, ok := gen.config.SkipTypes[superType.Name()]; ok {
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
	var lastExpr dst.Expr

	for _, node := range remainingNodes {
		switch node := node.(type) {
		case dst.Expr:
			if lastExpr != nil {
				panic("predicate should produce at most one expression")
			}
			lastExpr = node
		case dst.Stmt:
			stmts = append(stmts, node)
		default:
			panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", node))
		}
	}

	// Make sure the last statement always returns.
	switch lastNode := lastNode.(type) {
	case dst.Expr:
		// If the last node is an expression, convert it to a return.
		// However, if there is already a return, then merge this expression to the existing return.
		if len(remainingNodes) > 1 {
			oneBeforeLast := remainingNodes[lastIndex-1]
			if existingReturnStmt, hasReturn := oneBeforeLast.(*dst.ReturnStmt); hasReturn {
				validateReturn(existingReturnStmt)
				existingReturnStmt.Results = []dst.Expr{lastNode}
				transferCommentsToTheReturn(lastNode, existingReturnStmt)

			}
			break
		}

		// Otherwise, create a new return statement with this expressions
		returnStmt := returnStatementWith(lastNode)
		stmts = append(stmts, returnStmt)

	case dst.Stmt:
		stmts = append(stmts, lastNode)
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
			lineComments := descriptionAsLineComments(description)
			firstNodeDecs.Before = dst.EmptyLine
			firstNodeDecs.Start.Append(lineComments...)
		}
	}()

	// If there are no chained/nested predicates (originating from AND),
	// then add return the statements as-is.
	// Also, if there is a negation, then do not nest, but rather early.
	if gen.negate ||
		gen.nestedPredicates == nil ||
		!gen.nestedPredicates.hasMore() {
		return prevNodes
	}

	// If there are chained/nested predicates (originating from AND),
	// then they should be generated instead of the return.

	nextPredicate := gen.nestedPredicates.next()
	nestedNodes := gen.generatePredicate(nextPredicate)

	// If previous nodes are nil, simply return the nested nodes.
	if len(prevNodes) == 0 {
		return nestedNodes
	}

	// If not (both previous nodes and nested nodes exist),
	// then merge the two set of nodes.
	// Merging happens via the last statement of the previous nodes.
	return gen.mergeNestedNodesWithLastNode(prevNodes, nestedNodes)
}

// mergeNestedNodesWithLastNode merges the nested nodes as if they should be nested
// inside the existing nodes.
func (gen *SubTypeCheckGenerator) mergeNestedNodesWithLastNode(
	existingNodes []dst.Node,
	nestedNodes []dst.Node,
) []dst.Node {
	lastIndex := len(existingNodes) - 1
	lastNode := existingNodes[lastIndex]

	// Add all nodes upto the last-1.
	// Use the last node to merge the nested nodes.
	mergedNodes := existingNodes[:lastIndex]

	switch lastNode := lastNode.(type) {
	case nil:
		panic(fmt.Errorf("generated node is nil"))

	case dst.Expr:
		mergedNodes = gen.mergeNestedNodesWithExpression(
			mergedNodes,
			lastNode,
			nestedNodes,
		)

	case *dst.TypeSwitchStmt:
		mergedNodes = gen.mergeNestedNodeWithSwitchStatement(
			mergedNodes,
			lastNode,
			lastNode.Body,
			nestedNodes,
		)

	case *dst.SwitchStmt:
		mergedNodes = gen.mergeNestedNodeWithSwitchStatement(
			mergedNodes,
			lastNode,
			lastNode.Body,
			nestedNodes,
		)

	case *dst.RangeStmt:
		stmts := gen.mergeNodesAsStatements(nestedNodes)
		mergedNodes = append(mergedNodes, lastNode)
		for _, stmt := range stmts {
			mergedNodes = append(mergedNodes, stmt)
		}

	case *dst.ReturnStmt:
		// Merging with a return statement: Return statements are added during generation
		// to make the generated rule self-sufficient.
		// When merging, it is safe to remove these return statements.

		// However, makesure we are not dropping vital information:
		// by checking the return is a one that was only added for "completeness"
		// i.e: the dropping return should only be either `return true/false`.
		// NOTE: validate the one that we are dropping.
		validateReturn(lastNode)

		// Drop the return (i.e: ignore the lastNode),
		// and merge with the one before the node.
		mergedNodes = gen.mergeNestedNodesWithLastNode(
			mergedNodes,
			nestedNodes,
		)

		// However, if there was no return added during merging, then keep the return, for completeness.
		lastMergedNode := mergedNodes[len(mergedNodes)-1]
		if _, isReturn := lastMergedNode.(*dst.ReturnStmt); !isReturn {
			mergedNodes = append(mergedNodes, lastNode)
		}

	default:
		panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", lastNode))
	}

	return mergedNodes
}

func validateReturn(returnStmt *dst.ReturnStmt) {
	if len(returnStmt.Results) != 1 {
		panic(fmt.Errorf("error generating predicate AST: expected only one return value"))
	}

	returnValue, ok := returnStmt.Results[0].(*dst.Ident)
	if !ok || (returnValue.Name != "true" && returnValue.Name != "false") {
		panic(fmt.Errorf("error generating predicate AST: expected `return true/false` statement"))
	}
}

func descriptionAsLineComments(description string) []string {
	description = strings.ReplaceAll(description, "#", "//")
	return strings.Split(description, "\n")
}

func (gen *SubTypeCheckGenerator) mergeNestedNodesWithExpression(
	result []dst.Node,
	expr dst.Expr,
	nestedNodes []dst.Node,
) []dst.Node {
	// Previous nodes ended with an expression.
	// Then either merge the nested ones using a:
	//  - Binary expression: for nested expressions.
	//  - If statement: for nested statements.

	// The nested nodes can have only one expression.
	//
	// Note: it is not possible to generate more than one expression.
	// TODO: Maybe validate that.
	if len(nestedNodes) == 1 {
		onlyExpr, ok := nestedNodes[0].(dst.Expr)
		if ok {
			result = append(
				result,
				gen.binaryExpression(
					expr,
					onlyExpr,
					token.LAND,
				),
			)

			return result
		}
	}

	// There are both expressions and statements generated.
	// Convert the conditional-expression into a statement,
	// by putting them as the condition of an if-statement.
	stmts := gen.mergeNodesAsStatements(nestedNodes)

	result = append(
		result,
		gen.mergeUsingIfStatement(expr, stmts),
	)

	return result
}

func (gen *SubTypeCheckGenerator) mergeUsingIfStatement(expr dst.Expr, stmts []dst.Stmt) *dst.IfStmt {
	lastStmt := stmts[len(stmts)-1]
	if _, isReturn := lastStmt.(*dst.ReturnStmt); !isReturn {
		stmts = append(
			stmts,
			&dst.ReturnStmt{
				Results: []dst.Expr{
					gen.booleanExpression(false),
				},
			},
		)
	}

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
) []dst.Node {
	stmts := gen.mergeNodesAsStatements(nestedNodes)

	caseClauses := switchStmtBody.List
	if len(caseClauses) == 0 {
		panic("switch-statement must have at-least one cases clause")
	}

	lastCase := caseClauses[len(caseClauses)-1]
	caseClause := lastCase.(*dst.CaseClause)

	// If the case statement is non-empty, and the only statement is a `return false`,
	// that means it's a rule with negation,
	// and a return must have been added as an early-exit strategy.
	// If so, add the nested conditions after the switch statement.
	switch len(caseClause.Body) {
	case 0:
		// If the case statement is empty, then include the nested nodes
		// inside the case-body.
		caseClause.Body = append(caseClause.Body, stmts...)
		combinedNodes = append(combinedNodes, switchStmt)
		return combinedNodes
	case 1:
		// Only one statement
		lastStmtInsideCase := caseClause.Body[0]
		returnStmt, isReturnStmt := lastStmtInsideCase.(*dst.ReturnStmt)
		if !isReturnStmt {
			panic("last statement of a case-clause must be a return statement")
		}

		if len(returnStmt.Results) != 1 {
			panic(fmt.Errorf("error generating predicate AST: expected only one return value"))
		}

		returnValue, ok := returnStmt.Results[0].(*dst.Ident)
		if !ok || returnValue.Name != "true" {
			break
		}

		// If the statement is a `return true`, then this is a return added for completeness.
		// Ignore the return the merge the nested statements in-place of the return.
		caseClause.Body = stmts
		combinedNodes = append(combinedNodes, switchStmt)
		return combinedNodes
	}

	clauseLastStmtIndex := len(caseClause.Body) - 1
	lastStmtInsideCase := caseClause.Body[clauseLastStmtIndex]
	if _, isReturnStmt := lastStmtInsideCase.(*dst.ReturnStmt); !isReturnStmt {
		panic("last statement of a case-clause must be a return statement")
	}

	// Then the nested-conditions must be included after the switch statement.
	combinedNodes = append(combinedNodes, switchStmt)
	for _, stmt := range stmts {
		combinedNodes = append(combinedNodes, stmt)
	}

	return combinedNodes
}

func (gen *SubTypeCheckGenerator) mergeNodesAsStatements(nodes []dst.Node) (result []dst.Stmt) {
	var lastExpr dst.Expr

	for _, nestedNode := range nodes {
		switch nestedNode := nestedNode.(type) {
		case dst.Expr:
			if lastExpr != nil {
				panic("predicate should produce at most one expression")
			}
			lastExpr = nestedNode

		case dst.Stmt:
			// If there has been an expression preceding this statement,
			// then convert that previous expression into a statement
			// by putting that expression as a condition of an if-statement.
			if lastExpr != nil {
				ifStmt := gen.mergeUsingIfStatement(
					lastExpr,
					[]dst.Stmt{
						&dst.ReturnStmt{
							Results: []dst.Expr{
								gen.booleanExpression(true),
							},
						},
					},
				)

				// Then append the current statement
				result = append(result, ifStmt)

				// Clear
				lastExpr = nil
			}

			result = append(result, nestedNode)

		default:
			panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", nestedNode))
		}
	}

	// If there was still an expression left (i.e: last node is an expression),
	// then convert it to a statement by putting it in a return statement.
	if lastExpr != nil {
		result = append(result, returnStatementWith(lastExpr))
	}

	return result
}

func returnStatementWith(returnValue dst.Expr) *dst.ReturnStmt {
	returnStmt := &dst.ReturnStmt{
		Results: []dst.Expr{
			returnValue,
		},
	}

	transferCommentsToTheReturn(returnValue, returnStmt)

	return returnStmt
}

func transferCommentsToTheReturn(returnValue dst.Expr, returnStmt *dst.ReturnStmt) {
	returnValueDecs := returnValue.Decorations()
	comments := returnValueDecs.Start

	if len(comments) > 0 {
		returnStmt.Decorations().Start.Append(comments...)
		returnValueDecs.Start.Clear()
		returnStmt.Decs.Before = dst.EmptyLine
	}
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
	args := gen.extraArguments()

	args = append(
		args,
		gen.expressionIgnoreNegation(predicate.Expression),
	)

	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("IsStorableType"),
			args...,
		),
	}
}

func (gen *SubTypeCheckGenerator) notPredicate(p NotPredicate) []dst.Node {
	prevNegate := gen.negate
	defer func() {
		gen.negate = prevNegate
	}()

	// negate the current negation.
	gen.negate = !gen.negate

	return gen.generatePredicate(p.Predicate)
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

	var (
		lastNode       dst.Node
		prevTypeSwitch *dst.TypeSwitchStmt
		returnStmt     *dst.ReturnStmt
	)

	for _, predicate := range predicates {
		generatedPredicatedNodes := gen.generatePredicate(predicate)

		for _, currentNode := range generatedPredicatedNodes {
			switch typedCurrentNode := currentNode.(type) {
			case *dst.TypeSwitchStmt:
				if prevTypeSwitch != nil &&
					reflect.DeepEqual(prevTypeSwitch.Assign, typedCurrentNode.Assign) {
					mergeTypeSwitches(prevTypeSwitch, typedCurrentNode)
				} else {
					prevTypeSwitch = typedCurrentNode
					result = append(result, typedCurrentNode)
				}
			case *dst.ReturnStmt:
				// There can be multiple return statements generated
				// because each generated predicate is "self-complete".
				// In that case, only keep the last return statement.

				// But, makesure we are not dropping vital information:
				// by checking the return is a one that was only added for "completeness"
				// i.e: the dropping return should only be either `return true/false`.
				// NOTE: validate the one that we are dropping.
				if returnStmt != nil {
					validateReturn(returnStmt)
				}

				returnStmt = typedCurrentNode
			case dst.Stmt:
				switch typedLastNode := lastNode.(type) {
				case dst.Expr:
					ifStmt := gen.mergeUsingIfStatement(
						typedLastNode,
						[]dst.Stmt{
							&dst.ReturnStmt{
								Results: []dst.Expr{
									gen.booleanExpression(true),
								},
							},
						},
					)

					result[len(result)-1] = ifStmt
				}

				result = append(result, typedCurrentNode)

			case dst.Expr:
				switch typedLastNode := lastNode.(type) {
				case dst.Expr:

					typedCurrentNode.Decorations().Before = dst.NewLine

					// Don't negate again here (i.e: don't use `binaryExpression` method),
					// since negation is already done before calling this method `generateOrPredicate`.

					existingDecs := typedLastNode.Decorations().Start
					typedLastNode.Decorations().Start = nil

					currentNode = &dst.BinaryExpr{
						X:  typedLastNode,
						Op: token.LOR,
						Y:  typedCurrentNode,
						Decs: dst.BinaryExprDecorations{
							NodeDecs: dst.NodeDecs{
								Start: existingDecs,
							},
						},
					}

					// We merged it with the last expr. So replace the existing one with the
					// merged expression, rather than appending.
					result[len(result)-1] = currentNode

				default:
					result = append(result, typedCurrentNode)
				}

			default:
				panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", typedCurrentNode))
			}

			lastNode = currentNode
		}
	}

	// Append the return statement as the last statement.
	if returnStmt != nil {
		if lastExpr, ok := lastNode.(dst.Expr); ok {
			result = append(result, returnStatementWith(lastExpr))
		} else {
			result = append(result, returnStmt)
		}
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
	args := gen.extraArguments()

	args = append(
		args,
		gen.expressionIgnoreNegation(predicate.Expression),
	)

	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("isAttachmentType"),
			args...,
		),
	}
}

func (gen *SubTypeCheckGenerator) isResourcePredicate(predicate IsResourcePredicate) []dst.Node {
	args := gen.extraArguments()

	args = append(
		args,
		gen.expressionIgnoreNegation(predicate.Expression),
	)

	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("IsResourceType"),
			args...,
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
				Body: []dst.Stmt{
					// Always add a return for the completeness.
					// This return will be removed/kept as desired, when other statements
					// are merged with this list of statements.
					&dst.ReturnStmt{
						Results: []dst.Expr{
							gen.booleanExpression(true),
						},
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

			// Always add a return for the completeness.
			// This return will be removed/kept as desired, when other statements
			// are merged with this list of statements.
			&dst.ReturnStmt{
				Results: []dst.Expr{
					gen.booleanExpression(false),
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
				&dst.Ident{
					Path: commonsPkgPath,
					Name: "DeepEquals",
				},
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
		return dst.NewIdent(name)
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
	args := gen.extraArguments()

	args = append(
		args,
		gen.expressionIgnoreNegation(permits.Super),
		gen.expressionIgnoreNegation(permits.Sub),
	)

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
		Body: []dst.Stmt{
			// Always add a return for the completeness.
			// This return will be removed/kept as desired, when other statements
			// are merged with this list of statements.
			&dst.ReturnStmt{
				Results: []dst.Expr{
					gen.booleanExpression(true),
				},
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

		// Always add a return for the completeness.
		// This return will be removed/kept as desired, when other statements
		// are merged with this list of statements.
		&dst.ReturnStmt{
			Results: []dst.Expr{
				gen.booleanExpression(false),
			},
		},
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
	args := gen.extraArguments()

	args = append(
		args,
		gen.expressionIgnoreNegation(p.Super),
		gen.expressionIgnoreNegation(p.Sub),
	)

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
	args := gen.extraArguments()

	args = append(
		args,
		gen.expressionIgnoreNegation(p.Sub),
		gen.expressionIgnoreNegation(p.Super),
	)

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

func (gen *SubTypeCheckGenerator) separateRules(rules []Rule) (
	simpleTypeRules []Rule,
	complexTypeRules []Rule,
	defaultRule *Rule,
) {
	for _, rule := range rules {
		superType := rule.SuperType

		if superType == nil {
			if defaultRule != nil {
				panic("can only have one default rule")
			}

			defaultRule = &rule
			continue
		}

		_, isSimpleType := superType.(SimpleType)
		if isSimpleType {
			simpleTypeRules = append(simpleTypeRules, rule)
			continue
		}

		complexTypeRules = append(complexTypeRules, rule)
	}

	return
}
