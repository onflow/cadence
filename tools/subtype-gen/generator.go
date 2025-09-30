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
	"github.com/dave/dst"
	"go/token"
)

var neverType = SimpleType{
	name: "Never",
}

var interfaceType = ComplexType{}

type SubTypeCheckGenerator struct {
	config Config

	currentSuperType Type
	currentSubType   Type

	superTypeVarName string
	subTypeVarName   string
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

// GenerateCheckSubTypeWithoutEqualityFunction generates the complete checkSubTypeWithoutEquality function.
func (gen *SubTypeCheckGenerator) GenerateCheckSubTypeWithoutEqualityFunction(rules []Rule) []dst.Decl {
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
	superTypeParam := &dst.Field{
		Names: []*dst.Ident{
			dst.NewIdent(superTypeVarName),
		},
		Type: gen.qualifiedTypeIdentifier(interfaceType),
	}

	// Create function body
	var stmts []dst.Stmt

	// Add early return for Never type
	neverTypeCheck := &dst.IfStmt{
		Cond: &dst.BinaryExpr{
			X:  dst.NewIdent(subTypeVarName),
			Op: token.EQL,
			Y:  gen.qualifiedTypeIdentifier(neverType),
		},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				&dst.ReturnStmt{
					Results: []dst.Expr{dst.NewIdent("true")},
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
	switchStmtForSimpleTypes := gen.createSwitchStatement(rules, true)
	stmts = append(stmts, switchStmtForSimpleTypes)

	// Create switch statement for complex types.
	switchStmtForComplexTypes := gen.createSwitchStatement(rules, false)
	stmts = append(stmts, switchStmtForComplexTypes)

	// Add final return false
	stmts = append(stmts, &dst.ReturnStmt{
		Results: []dst.Expr{dst.NewIdent("false")},
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

// createSwitchStatement creates the switch statement for superType
func (gen *SubTypeCheckGenerator) createSwitchStatement(rules []Rule, forSimpleTypes bool) dst.Stmt {
	var cases []dst.Stmt

	if forSimpleTypes {
		gen.superTypeVarName = superTypeVarName
	} else {
		gen.superTypeVarName = typedSuperTypeVarName
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
			Tag: dst.NewIdent(gen.superTypeVarName),
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
				dst.NewIdent(gen.superTypeVarName),
			},
			Tok: token.DEFINE,
			Rhs: []dst.Expr{
				&dst.TypeAssertExpr{
					X:    dst.NewIdent(superTypeVarName),
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

	if rule.Sub == "" {
		gen.subTypeVarName = subTypeVarName
	} else {
		gen.subTypeVarName = typedSubTypeVarName

		subType := parseType(rule.Sub)

		assignment := &dst.AssignStmt{
			Lhs: []dst.Expr{
				dst.NewIdent(gen.subTypeVarName),
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
							dst.NewIdent("false"),
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

	predicate, err := parseRulePredicate(rule.Predicate)
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
				Results: []dst.Expr{dst.NewIdent("false")},
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
		return []dst.Node{dst.NewIdent("true")}

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

	case ContainsPredicate:
		// TODO: Implement contains condition
		return []dst.Node{dst.NewIdent("false")}

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
	innerPredicateNodes := gen.generatePredicate(p.Predicate)
	if len(innerPredicateNodes) != 1 {
		panic("can only handle one node in `not` predicate")
	}

	innerPredicateExpr, ok := innerPredicateNodes[0].(dst.Expr)
	if !ok {
		panic("cannot handle statements in `not` predicate")
	}

	return []dst.Node{
		&dst.UnaryExpr{
			Op: token.NOT,
			X:  &dst.ParenExpr{X: innerPredicateExpr},
		},
	}
}

func (gen *SubTypeCheckGenerator) andPredicate(p AndPredicate) []dst.Node {
	var result []dst.Node
	var exprs []dst.Expr

	for _, condition := range p.Predicates {
		generatedPredicatedNodes := gen.generatePredicate(condition)
		for _, node := range generatedPredicatedNodes {
			switch node := node.(type) {
			case dst.Stmt:
				// TODO: cannot handle statements in `and` predicate"
				// Ignore for now
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

		binaryExpr = &dst.BinaryExpr{
			X:  binaryExpr,
			Op: token.LAND,
			Y:  expr,
		}
	}

	result = append(result, binaryExpr)
	return result
}

func (gen *SubTypeCheckGenerator) orPredicate(p OrPredicate) []dst.Node {
	var result []dst.Node

	var exprs []dst.Expr
	for _, condition := range p.Predicates {
		generatedPredicatedNodes := gen.generatePredicate(condition)
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

		binaryExpr = &dst.BinaryExpr{
			X:  binaryExpr,
			Op: token.LOR,
			Y:  expr,
		}
	}

	result = append(result, binaryExpr)

	return result
}

func (gen *SubTypeCheckGenerator) isAttachmentPredicate(predicate IsAttachmentPredicate) []dst.Node {
	return []dst.Node{
		&dst.CallExpr{
			Fun: dst.NewIdent("isAttachmentType"),
			Args: []dst.Expr{
				gen.expression(predicate.Expression),
			},
		},
	}
}

func (gen *SubTypeCheckGenerator) isResourcePredicate(predicate IsResourcePredicate) []dst.Node {
	return []dst.Node{
		&dst.CallExpr{
			Fun: dst.NewIdent("IsResourceType"),
			Args: []dst.Expr{
				gen.expression(predicate.Expression),
			},
		},
	}
}

// equalsPredicate generates AST for equals predicate
func (gen *SubTypeCheckGenerator) equalsPredicate(equals EqualsPredicate) []dst.Node {
	switch target := equals.Target.(type) {
	case TypeExpression, MemberExpression, IdentifierExpression:
		return []dst.Node{
			&dst.BinaryExpr{
				// TODO:
				X:  gen.expression(equals.Source),
				Op: token.EQL,
				Y:  gen.expression(target),
			},
		}
	case OneOfExpression:
		exprs := target.Expressions
		// If there's only one type to match, use `==`.
		if len(exprs) == 1 {
			expr := exprs[0]
			return []dst.Node{
				&dst.BinaryExpr{
					X:  gen.expression(equals.Source),
					Op: token.EQL,
					Y:  gen.expression(expr),
				},
			}
		}

		// Otherwise, if there are more than one type, then generate a switch-case.
		var cases []dst.Expr
		for _, expr := range exprs {
			generatedExpr := gen.expression(expr)
			generatedExpr.Decorations().After = dst.NewLine
			cases = append(cases, generatedExpr)
		}

		// Generate switch expression
		var caseClauses []dst.Stmt
		caseClauses = append(caseClauses, &dst.CaseClause{
			List: cases,
			Body: []dst.Stmt{
				&dst.ReturnStmt{
					Results: []dst.Expr{dst.NewIdent("true")},
				},
			},
		})

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
	switch expr := expr.(type) {
	case IdentifierExpression:
		switch expr.Name {
		case "sub":
			return &dst.Ident{Name: gen.subTypeVarName}
		case "super":
			return &dst.Ident{Name: gen.superTypeVarName}
		default:
			return dst.NewIdent(expr.Name)
		}

	case TypeExpression:
		return gen.qualifiedTypeIdentifier(expr.Type)

	case MemberExpression:
		selectorExpr := &dst.SelectorExpr{
			X:   gen.expression(expr.Parent),
			Sel: dst.NewIdent(expr.MemberName),
		}

		if expr.MemberName == "ElementType" {
			var args []dst.Expr
			for _, arg := range gen.config.ArrayElementTypeMethodArgs {
				args = append(args, dst.NewIdent(fmt.Sprint(arg)))
			}

			return &dst.CallExpr{
				Fun:  selectorExpr,
				Args: args,
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
	case TypeExpression, MemberExpression:
		args := gen.isSubTypeMethodArguments(subtype.Sub, superType)
		return []dst.Node{
			&dst.CallExpr{
				Fun:  dst.NewIdent("IsSubType"),
				Args: args,
			},
		}
	case OneOfExpression:
		var conditions []dst.Expr
		for _, expr := range superType.Expressions {
			args := gen.isSubTypeMethodArguments(subtype.Sub, expr)
			// TODO: Recursively call `generatePredicate`
			conditions = append(
				conditions,
				&dst.CallExpr{
					Fun:  dst.NewIdent("IsSubType"),
					Args: args,
				},
			)
		}

		if len(conditions) == 0 {
			return []dst.Node{dst.NewIdent("false")}
		}

		result := conditions[0]
		for i := 1; i < len(conditions); i++ {
			nextCondition := conditions[i]
			nextCondition.Decorations().Before = dst.NewLine
			result = &dst.BinaryExpr{
				X:  result,
				Op: token.LOR,
				Y:  nextCondition,
			}
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
