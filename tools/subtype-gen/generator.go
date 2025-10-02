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
	"strings"

	"github.com/dave/dst"

	"github.com/onflow/cadence/common/orderedmap"
)

var neverType = SimpleType{
	name: "Never",
}

var interfaceType = ComplexType{}

var super = IdentifierExpression{Name: "super"}
var sub = IdentifierExpression{Name: "sub"}

type SubTypeCheckGenerator struct {
	config Config

	currentSuperType Type
	currentSubType   Type

	nextVarIndex int

	negate bool

	scope          []map[Expression]string
	predicateChain *PredicateChain
}

type Config struct {
	SimpleTypePrefix  string
	SimpleTypeSuffix  string
	ComplexTypePrefix string
	ComplexTypeSuffix string

	ExtraParams []ExtraParam
	SkipTypes   map[string]struct{}

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

	prevSuperType := gen.expression(super)

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
		caseStmt := gen.createCaseStatement(rule, forSimpleTypes)
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

// createCaseStatement creates a case statement for a rule
func (gen *SubTypeCheckGenerator) createCaseStatement(rule Rule, forSimpleTypes bool) dst.Stmt {
	prevSuperType := gen.currentSuperType
	prevSubType := gen.currentSubType
	defer func() {
		gen.currentSuperType = prevSuperType
		gen.currentSubType = prevSubType
	}()

	// Parse types

	superType := parseType(rule.Super)
	gen.currentSuperType = superType

	// Skip the given types.
	// Some types are only exist during type-checking, but not at runtime. e.g: Storable type
	if _, ok := gen.config.SkipTypes[superType.Name()]; ok {
		return nil
	}

	_, isSimpleType := superType.(SimpleType)
	if isSimpleType != forSimpleTypes {
		return nil
	}

	gen.currentSubType = parseType(rule.Sub)

	// Generate case condition
	caseExpr := gen.parseCaseCondition(superType)

	var bodyStmts []dst.Stmt

	// If the subtype needs to be of a certain type,
	// then add a type assertion.

	if rule.Sub != "" {
		// A type-assertion is created for the `sub`.
		// So register a new variable to hold the type-value of `sub`.
		// During the nested generations, `sub` will refer to this variable.
		typedVarName := gen.newTypedVariableNameFor(sub)
		gen.pushScope()
		gen.addToScope(
			sub,
			typedVarName,
		)
		defer gen.popScope()

		subType := parseType(rule.Sub)

		assignment := &dst.AssignStmt{
			Lhs: []dst.Expr{
				dst.NewIdent(gen.newTypedVariableNameFor(sub)),
				dst.NewIdent("ok"),
			},
			Tok: token.DEFINE,
			Rhs: []dst.Expr{
				&dst.TypeAssertExpr{
					X: dst.NewIdent(subTypeVarName),
					Type: &dst.StarExpr{
						X: gen.qualifiedTypeIdentifier(subType),
					},
				},
			},
		}

		ifStmt := &dst.IfStmt{
			Cond: &dst.UnaryExpr{
				X:  dst.NewIdent("ok"),
				Op: token.NOT,
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
			Decs: dst.IfStmtDecorations{
				NodeDecs: dst.NodeDecs{
					Before: dst.NewLine,
					After:  dst.EmptyLine,
				},
			},
		}

		bodyStmts = append(
			bodyStmts,
			assignment,
			ifStmt,
		)
	}

	// Generate statements for the predicate.

	predicate, err := parsePredicate(rule.Predicate)
	if err != nil {
		panic(fmt.Errorf("error parsing rule predicate: %w", err))
	}

	bodyStmts = append(
		bodyStmts,
		gen.generatePredicateStatements(predicate)...,
	)

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

	var stmts []dst.Stmt

	if len(nodes) > 1 {
		for _, node := range nodes[:lastIndex] {
			switch node := node.(type) {
			case dst.Expr:
				panic("predicate should produce at most one expression")
			case dst.Stmt:
				stmts = append(stmts, node)
			default:
				panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", node))
			}
		}
	}

	// Make sure the last statement always returns.
	switch lastNode := lastNode.(type) {
	case dst.Expr:
		stmts = append(stmts,
			&dst.ReturnStmt{
				Results: []dst.Expr{lastNode},
			},
		)
	case *dst.SwitchStmt:
		// Switch statements are generated without the default return,
		// so that they can be combined with other statements.
		// If the last statement if a switch-case, then
		// append a return statement.
		stmts = append(stmts, lastNode)
		stmts = append(stmts,
			&dst.ReturnStmt{
				Results: []dst.Expr{
					gen.booleanExpression(false),
				},
			},
		)
	case dst.Stmt:
		// TODO: Maybe panic? - because we only generate either an expression, or a switch statement for now.
		stmts = append(stmts, lastNode)
	default:
		panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", lastNode))
	}

	return stmts
}

// generatePredicate recursively generates one or more expression/statement for a given predicate.
func (gen *SubTypeCheckGenerator) generatePredicate(predicate Predicate) (result []dst.Node) {
	switch p := predicate.(type) {
	case AlwaysPredicate:
		return []dst.Node{
			gen.booleanExpression(true),
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

	case SubtypePredicate:
		return gen.isSubTypePredicate(p)

	case PermitsPredicate:
		return gen.permitsPredicate(p)

	case TypeAssertionPredicate:
		return gen.typeAssertion(p)

	case PurityPredicate:
		// TODO: Implement purity check
		return []dst.Node{dst.NewIdent("false")}

	case TypeParamsEqualPredicate:
		// TODO: Implement type-params-equal check
		return []dst.Node{dst.NewIdent("false")}

	case ParamsContravariantPredicate:
		// TODO: Implement params-contravariant check
		return []dst.Node{dst.NewIdent("false")}

	case ReturnCovariantPredicate:
		// TODO: Implement return-covariant check
		return []dst.Node{dst.NewIdent("false")}

	case ConstructorEqualPredicate:
		// TODO: Implement constructor-equal check
		return []dst.Node{dst.NewIdent("false")}

	case SetContainsPredicate:
		return gen.setContains(p)

	default:
		panic(fmt.Errorf("unsupported predicate: %T", p))
	}
}

func (gen *SubTypeCheckGenerator) isHashableStructPredicate(predicate IsHashableStructPredicate) []dst.Node {
	args := gen.extraArguments()

	args = append(
		args,
		gen.expression(predicate.Expression),
	)

	return []dst.Node{
		&dst.CallExpr{
			Fun:  dst.NewIdent("IsHashableStructType"),
			Args: args,
		},
	}
}

func (gen *SubTypeCheckGenerator) isStorablePredicate(predicate IsStorablePredicate) []dst.Node {
	return []dst.Node{
		&dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   gen.expression(predicate.Expression),
				Sel: dst.NewIdent("IsStorable"),
			},
			Args: []dst.Expr{
				&dst.CompositeLit{
					Type: &dst.MapType{
						Key: &dst.StarExpr{
							X: dst.NewIdent("Member"),
						},
						Value: dst.NewIdent("bool"),
					},
				},
			},
		},
	}
}

func (gen *SubTypeCheckGenerator) notPredicate(p NotPredicate) []dst.Node {
	prevNegate := gen.negate
	gen.negate = true
	defer func() {
		gen.negate = prevNegate
	}()

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
	var result []dst.Node
	var exprs []dst.Expr

	prevPredicateChain := gen.predicateChain
	gen.predicateChain = NewPredicateChain(p.Predicates)
	defer func() {
		gen.predicateChain = prevPredicateChain
	}()

	for gen.predicateChain.hasMore() {
		predicate := gen.predicateChain.next()
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

	//for _, condition := range p.Predicates {
	//	generatedPredicatedNodes := gen.generatePredicate(condition)
	//	for _, node := range generatedPredicatedNodes {
	//		switch node := node.(type) {
	//		case dst.Stmt:
	//			// Add statements as-is, since they are all conditional-statements.
	//			result = append(result, node)
	//		case dst.Expr:
	//			exprs = append(exprs, node)
	//		default:
	//			panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", node))
	//		}
	//	}
	//}

	var binaryExpr dst.Expr
	for _, expr := range exprs {
		if binaryExpr == nil {
			binaryExpr = expr
			continue
		}

		expr.Decorations().Before = dst.NewLine

		binaryExpr = gen.binaryExpression(
			binaryExpr,
			expr,
			token.LAND,
		)
	}

	if binaryExpr != nil {
		result = append(result, binaryExpr)
	}

	return result
}

func (gen *SubTypeCheckGenerator) orPredicate(p OrPredicate) []dst.Node {
	var result []dst.Node
	var exprs []dst.Expr

	var typeAssertions *orderedmap.OrderedMap[Expression, []TypeAssertionPredicate]

	for _, predicate := range p.Predicates {

		// Combine all type assertions as generate a single switch-statement.
		if typeAssertion, ok := predicate.(TypeAssertionPredicate); ok {
			if typeAssertions == nil {
				typeAssertions = orderedmap.New[orderedmap.OrderedMap[Expression, []TypeAssertionPredicate]](1)
			}

			typeAssertionsForSource, _ := typeAssertions.Get(typeAssertion.Source)
			typeAssertionsForSource = append(typeAssertionsForSource, typeAssertion)
			typeAssertions.Set(typeAssertion.Source, typeAssertionsForSource)

			continue
		}

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

	if typeAssertions != nil {
		typeAssertions.Foreach(func(source Expression, assertionsForExpression []TypeAssertionPredicate) {
			switchStmt := gen.typeAssertionsForSource(source, assertionsForExpression...)
			result = append(result, switchStmt)
		})
	}

	var binaryExpr dst.Expr
	for _, expr := range exprs {
		if binaryExpr == nil {
			binaryExpr = expr
			continue
		}

		expr.Decorations().Before = dst.NewLine

		binaryExpr = gen.binaryExpression(
			binaryExpr,
			expr,
			token.LOR,
		)
	}

	if binaryExpr != nil {
		result = append(result, binaryExpr)
	}

	return result
}

func (gen *SubTypeCheckGenerator) isAttachmentPredicate(predicate IsAttachmentPredicate) []dst.Node {
	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("isAttachmentType"),
			gen.expression(predicate.Expression),
		),
	}
}

func (gen *SubTypeCheckGenerator) isResourcePredicate(predicate IsResourcePredicate) []dst.Node {
	return []dst.Node{
		gen.callExpression(
			dst.NewIdent("IsResourceType"),
			gen.expression(predicate.Expression),
		),
	}
}

// equalsPredicate generates AST for equals predicate
func (gen *SubTypeCheckGenerator) equalsPredicate(equals EqualsPredicate) []dst.Node {
	switch target := equals.Target.(type) {
	case TypeExpression, MemberExpression, IdentifierExpression:
		return []dst.Node{
			gen.binaryExpression(
				gen.expression(equals.Source),
				gen.expression(target),
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
					gen.expression(equals.Source),
					gen.expression(expr),
					token.EQL,
				),
			}
		}

		// Otherwise, if there are more than one type, then generate a switch-case.
		var cases []dst.Expr
		for _, expr := range exprs {
			generatedExpr := gen.expression(expr)
			generatedExpr.Decorations().After = dst.NewLine
			cases = append(cases, generatedExpr)
		}

		var body []dst.Stmt

		// If there are chained/nested predicates (originating from AND),
		// then they should be generated instead of the return.
		// However, if there is a negate, then do not nest, but rather early exit
		// by adding a return.
		if !gen.negate &&
			gen.predicateChain != nil &&
			gen.predicateChain.hasMore() {

			innerPredicate := gen.predicateChain.next()
			body = gen.generatePredicateStatements(innerPredicate)

		} else {
			body = []dst.Stmt{
				&dst.ReturnStmt{
					Results: []dst.Expr{
						gen.booleanExpression(true),
					},
				},
			}
		}

		// Generate switch expression
		caseClauses := []dst.Stmt{
			&dst.CaseClause{
				List: cases,
				Body: body,
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
				Tag: gen.expression(equals.Source),
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

func (gen *SubTypeCheckGenerator) expression(expr Expression) dst.Expr {
	name, ok := gen.findInScope(expr)
	if ok {
		return &dst.Ident{Name: name}
	}

	switch expr := expr.(type) {
	case IdentifierExpression:
		return dst.NewIdent(expr.Name)
		//switch expr.Name {
		//case "sub":
		//	return &dst.Ident{Name: gen.subTypeVarName}
		//case "super":
		//	return &dst.Ident{Name: gen.superTypeVarName}
		//default:
		//	return dst.NewIdent(expr.Name)
		//}

	case TypeExpression:
		return gen.qualifiedTypeIdentifier(expr.Type)

	case MemberExpression:
		selectorExpr := &dst.SelectorExpr{
			X:   gen.expression(expr.Parent),
			Sel: dst.NewIdent(expr.MemberName),
		}

		switch expr.MemberName {
		case "ElementType":
			var args []dst.Expr
			for _, arg := range gen.config.ArrayElementTypeMethodArgs {
				args = append(args, dst.NewIdent(fmt.Sprint(arg)))
			}

			return &dst.CallExpr{
				Fun:  selectorExpr,
				Args: args,
			}
		case "EffectiveInterfaceConformanceSet",
			"EffectiveIntersectionSet":
			return &dst.CallExpr{
				Fun: selectorExpr,
			}
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
			// TODO: Recursively call `generatePredicate`
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
			nextCondition.Decorations().Before = dst.NewLine
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
		gen.expression(subType),
		gen.expression(superType),
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

// qualifiedTypeIdentifier creates a qualified type identifier, by
// prepending the package-qualifier,prefix, and appending `Type` suffix.
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
	// Use type assertion to determine the specific type
	typeName := gen.qualifiedTypeIdentifier(superType)
	switch superType.(type) {
	case SimpleType:
		// For simple types, use the type directly
		return typeName
	default:
		return &dst.StarExpr{
			X: typeName,
		}
	}
}

func (gen *SubTypeCheckGenerator) permitsPredicate(permits PermitsPredicate) []dst.Node {
	args := []dst.Expr{
		gen.expression(permits.Super),
		gen.expression(permits.Sub),
	}

	return []dst.Node{
		&dst.CallExpr{
			Fun:  dst.NewIdent("PermitsAccess"),
			Args: args,
		},
	}
}

func (gen *SubTypeCheckGenerator) typeAssertion(typeAssertion TypeAssertionPredicate) []dst.Node {
	return []dst.Node{
		gen.typeAssertionsForSource(typeAssertion.Source, typeAssertion),
	}
}

func (gen *SubTypeCheckGenerator) typeAssertionsForSource(source Expression, typeAssertions ...TypeAssertionPredicate) *dst.TypeSwitchStmt {

	sourceExpr := gen.expression(source)

	// A type-switch is created for the source-expression.
	// So register a new variable to hold the typed-value of the source expression.
	// During the nested generations, re-using the source expression will refer to this variable.
	typedVariableName := gen.newTypedVariableNameFor(source)
	gen.pushScope()
	gen.addToScope(
		source,
		typedVariableName,
	)
	defer gen.popScope()

	var cases []dst.Stmt
	for _, assertion := range typeAssertions {

		// Generate case condition
		caseExpr := gen.parseCaseCondition(assertion.Type)

		// Generate case body
		bodyStmts := gen.generatePredicateStatements(assertion.IfMatch)

		caseStmt := &dst.CaseClause{
			List: []dst.Expr{caseExpr},
			Body: bodyStmts,
			Decs: dst.CaseClauseDecorations{
				NodeDecs: dst.NodeDecs{
					Before: dst.NewLine,
					After:  dst.EmptyLine,
				},
			},
		}

		cases = append(cases, caseStmt)
	}

	return &dst.TypeSwitchStmt{
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

func (gen *SubTypeCheckGenerator) binaryExpression(lhs, rhs dst.Expr, operator token.Token) *dst.BinaryExpr {
	if gen.negate {
		operator = gen.negateOperator(operator)
	}

	return &dst.BinaryExpr{
		X:  lhs,
		Op: operator,
		Y:  rhs,
	}
}

func (gen *SubTypeCheckGenerator) negateOperator(operator token.Token) token.Token {
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

func (gen *SubTypeCheckGenerator) callExpression(invokedExpr dst.Expr, args ...dst.Expr) dst.Expr {
	var expr dst.Expr = &dst.CallExpr{
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
		X:   gen.expression(p.Source),
		Sel: dst.NewIdent("Contains"),
	}

	args := []dst.Expr{
		gen.expression(p.Target),
	}

	callExpr := &dst.CallExpr{
		Fun:  selectExpr,
		Args: args,
	}

	return []dst.Node{
		callExpr,
	}
}

type PredicateChain struct {
	size       int
	index      int
	predicates []Predicate
}

func NewPredicateChain(predicates []Predicate) *PredicateChain {
	return &PredicateChain{
		size:       len(predicates),
		index:      0,
		predicates: predicates,
	}
}

func (p *PredicateChain) hasMore() bool {
	return p.index < p.size
}

func (p *PredicateChain) next() Predicate {
	predicate := p.predicates[p.index]
	p.index++
	return predicate
}
