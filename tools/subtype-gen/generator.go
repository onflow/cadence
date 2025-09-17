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
)

const subtypeCheckFuncName = "checkSubTypeWithoutEquality_gen"

type SubTypeCheckGenerator struct {
	typePkgName      string
	typePkgQualifier string
}

func NewSubTypeCheckGenerator(typePackageName string) *SubTypeCheckGenerator {
	parts := strings.Split(typePackageName, "/")
	return &SubTypeCheckGenerator{
		typePkgName:      typePackageName,
		typePkgQualifier: parts[len(parts)-1],
	}
}

// GenerateCheckSubTypeWithoutEqualityFunction generates the complete checkSubTypeWithoutEquality function.
func (gen *SubTypeCheckGenerator) GenerateCheckSubTypeWithoutEqualityFunction(rules []Rule) []dst.Decl {
	checkSubTypeFunction := gen.createCheckSubTypeFunction(rules)
	return []dst.Decl{checkSubTypeFunction}
}

// createCheckSubTypeFunction creates the main checkSubTypeWithoutEquality function
func (gen *SubTypeCheckGenerator) createCheckSubTypeFunction(rules []Rule) dst.Decl {
	// Create function parameters
	subTypeParam := &dst.Field{
		Names: []*dst.Ident{dst.NewIdent("subType")},
		Type:  gen.qualifiedIdentifier("Type"),
	}
	superTypeParam := &dst.Field{
		Names: []*dst.Ident{dst.NewIdent("superType")},
		Type:  gen.qualifiedIdentifier("Type"),
	}

	// Create function body
	var stmts []dst.Stmt

	// Add early return for Never type
	neverTypeCheck := &dst.IfStmt{
		Cond: &dst.BinaryExpr{
			X:  dst.NewIdent("subType"),
			Op: token.EQL,
			Y:  gen.qualifiedTypeIdent("Never"),
		},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				&dst.ReturnStmt{
					Results: []dst.Expr{dst.NewIdent("true")},
				},
			},
		},
	}
	stmts = append(stmts, neverTypeCheck)

	// Create switch statement
	switchStmt := gen.createSwitchStatement(rules)
	stmts = append(stmts, switchStmt)

	// Add final return false
	stmts = append(stmts, &dst.ReturnStmt{
		Results: []dst.Expr{dst.NewIdent("false")},
	})

	return &dst.FuncDecl{
		Name: dst.NewIdent(subtypeCheckFuncName),
		Type: &dst.FuncType{
			Params: &dst.FieldList{
				List: []*dst.Field{subTypeParam, superTypeParam},
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
func (gen *SubTypeCheckGenerator) createSwitchStatement(rules []Rule) dst.Stmt {
	var cases []dst.Stmt

	for _, rule := range rules {
		caseStmt := gen.createCaseStatement(rule)
		cases = append(cases, caseStmt)
	}

	return &dst.SwitchStmt{
		Tag: dst.NewIdent("superType"),
		Body: &dst.BlockStmt{
			List: cases,
		},
	}
}

// createCaseStatement creates a case statement for a rule
func (gen *SubTypeCheckGenerator) createCaseStatement(rule Rule) dst.Stmt {
	// Parse super type
	superType := parseType(rule.Super)

	// Generate case condition
	caseExpr := gen.parseCaseCondition(superType)

	predicate, err := parseRulePredicate(rule.Predicate)
	if err != nil {
		panic(fmt.Errorf("error parsing rule predicate: %w", err))
	}

	// Generate statements for the predicate
	bodyStmts := gen.generatePredicateStatements(predicate)

	return &dst.CaseClause{
		List: []dst.Expr{caseExpr},
		Body: bodyStmts,
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
		return gen.isResourcePredicate()

	case IsAttachmentPredicate:
		return gen.isAttachmentPredicate()

	case IsHashableStructPredicate:
		return gen.isHashableStructPredicate()

	case IsStorablePredicate:
		return gen.isStorablePredicate()

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
		if len(p.Types) == 2 {
			callExpr := &dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X: &dst.SelectorExpr{
						X:   dst.NewIdent("typedSuperType"),
						Sel: dst.NewIdent("Authorization"),
					},
					Sel: dst.NewIdent("PermitsAccess"),
				},
				Args: []dst.Expr{
					&dst.SelectorExpr{
						X:   dst.NewIdent("typedSubType"),
						Sel: dst.NewIdent("Authorization"),
					},
				},
			}

			return []dst.Node{callExpr}
		}
		return []dst.Node{dst.NewIdent("false")} // TODO: Implement permits condition

	case PurityPredicate:
		return gen.purityPredicate()

	case TypeParamsEqualPredicate:
		return gen.typeParamsEqualPredicate()

	case ParamsContravariantPredicate:
		return []dst.Node{dst.NewIdent("false")} // TODO: Implement params contravariant check

	case ReturnCovariantPredicate:
		return []dst.Node{dst.NewIdent("false")} // TODO: Implement return covariant check

	case ConstructorEqualPredicate:
		return gen.constructorEqualPredicate()

	case ContainsPredicate:
		if len(p.Types) == 2 {
			callExpr := &dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X: &dst.CallExpr{
						Fun: &dst.SelectorExpr{
							X:   dst.NewIdent("typedSuperType"),
							Sel: dst.NewIdent("EffectiveIntersectionSet"),
						},
					},
					Sel: dst.NewIdent("IsSubsetOf"),
				},
				Args: []dst.Expr{
					&dst.CallExpr{
						Fun: &dst.SelectorExpr{
							X:   dst.NewIdent("typedSubType"),
							Sel: dst.NewIdent("EffectiveIntersectionSet"),
						},
					},
				},
			}

			return []dst.Node{callExpr}
		}
		return []dst.Node{dst.NewIdent("false")} // TODO: Implement contains condition

	default:
		return []dst.Node{dst.NewIdent("false")} // TODO: Implement condition: " + predicate.GetType()
	}
}

func (gen *SubTypeCheckGenerator) isHashableStructPredicate() []dst.Node {
	return []dst.Node{
		&dst.CallExpr{
			Fun:  dst.NewIdent("IsHashableStructType"),
			Args: []dst.Expr{dst.NewIdent("subType")},
		},
	}
}

func (gen *SubTypeCheckGenerator) isStorablePredicate() []dst.Node {
	return []dst.Node{
		&dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   dst.NewIdent("subType"),
				Sel: dst.NewIdent("IsStorable"),
			},
			Args: []dst.Expr{
				&dst.CompositeLit{
					Type: &dst.MapType{
						Key: &dst.StarExpr{
							X: gen.qualifiedIdentifier("Member"),
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

		binaryExpr = &dst.BinaryExpr{
			X:  binaryExpr,
			Op: token.LOR,
			Y:  expr,
		}
	}

	result = append(result, binaryExpr)

	return result
}

func (gen *SubTypeCheckGenerator) purityPredicate() []dst.Node {
	binaryExpr := &dst.BinaryExpr{
		X: &dst.BinaryExpr{
			X: &dst.SelectorExpr{
				X:   dst.NewIdent("typedSubType"),
				Sel: dst.NewIdent("Purity"),
			},
			Op: token.EQL,
			Y: &dst.SelectorExpr{
				X:   dst.NewIdent("typedSuperType"),
				Sel: dst.NewIdent("Purity"),
			},
		},
		Op: token.LOR,
		Y: &dst.BinaryExpr{
			X: &dst.SelectorExpr{
				X:   dst.NewIdent("typedSubType"),
				Sel: dst.NewIdent("Purity"),
			},
			Op: token.EQL,
			Y:  dst.NewIdent("FunctionPurityView"),
		},
	}
	return []dst.Node{binaryExpr}
}

func (gen *SubTypeCheckGenerator) typeParamsEqualPredicate() []dst.Node {
	binaryExpr := &dst.BinaryExpr{
		X: &dst.CallExpr{
			Fun: dst.NewIdent("len"),
			Args: []dst.Expr{
				&dst.SelectorExpr{
					X:   dst.NewIdent("typedSubType"),
					Sel: dst.NewIdent("TypeParameters"),
				},
			},
		},
		Op: token.EQL,
		Y: &dst.CallExpr{
			Fun: dst.NewIdent("len"),
			Args: []dst.Expr{
				&dst.SelectorExpr{
					X:   dst.NewIdent("typedSuperType"),
					Sel: dst.NewIdent("TypeParameters"),
				},
			},
		},
	}

	return []dst.Node{binaryExpr}
}

func (gen *SubTypeCheckGenerator) constructorEqualPredicate() []dst.Node {
	binaryExpr := &dst.BinaryExpr{
		X: &dst.SelectorExpr{
			X:   dst.NewIdent("typedSubType"),
			Sel: dst.NewIdent("IsConstructor"),
		},
		Op: token.EQL,
		Y: &dst.SelectorExpr{
			X:   dst.NewIdent("typedSuperType"),
			Sel: dst.NewIdent("IsConstructor"),
		},
	}

	return []dst.Node{binaryExpr}
}

func (gen *SubTypeCheckGenerator) isAttachmentPredicate() []dst.Node {
	return []dst.Node{
		&dst.CallExpr{
			Fun:  dst.NewIdent("isAttachmentType"),
			Args: []dst.Expr{dst.NewIdent("subType")},
		},
	}
}

func (gen *SubTypeCheckGenerator) isResourcePredicate() []dst.Node {
	return []dst.Node{
		&dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   dst.NewIdent("subType"),
				Sel: dst.NewIdent("IsResourceType"),
			},
		},
	}
}

// equalsPredicate generates AST for equals predicate
func (gen *SubTypeCheckGenerator) equalsPredicate(equals EqualsPredicate) []dst.Node {
	switch target := equals.Target.(type) {
	case Type:
		return []dst.Node{
			&dst.BinaryExpr{
				X:  dst.NewIdent("subType"),
				Op: token.EQL,
				Y:  gen.qualifiedTypeIdent(target.Name()),
			},
		}
	case OneOfTypes:
		types := target.Types
		// If there's only one type to match, use `==`.
		if len(types) == 1 {
			typ := types[0]
			return []dst.Node{
				&dst.BinaryExpr{
					X:  dst.NewIdent("subType"),
					Op: token.EQL,
					Y:  gen.qualifiedTypeIdent(typ.Name()),
				},
			}
		}

		// Otherwise, if there are more than one type, then generate a switch-case.
		var cases []dst.Expr
		for _, typ := range types {
			cases = append(cases, gen.qualifiedTypeIdent(typ.Name()))
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
				Tag: dst.NewIdent("subType"),
				Body: &dst.BlockStmt{
					List: caseClauses,
				},
			},
		}
	default:
		panic(fmt.Errorf("unknown target type %t in `equals` rule", target))
	}
}

// isSubTypePredicate generates AST for subtype conditions
func (gen *SubTypeCheckGenerator) isSubTypePredicate(subtype SubtypePredicate) []dst.Node {
	switch superType := subtype.Super.(type) {
	case Type:
		return []dst.Node{
			&dst.CallExpr{
				Fun: gen.qualifiedIdentifier("IsSubType"),
				Args: []dst.Expr{
					dst.NewIdent("subType"),
					gen.qualifiedTypeIdent(superType.Name()),
				},
			},
		}
	case OneOfTypes:
		var conditions []dst.Expr
		for _, typ := range superType.Types {
			// TODO: Recursively call `generatePredicate`
			conditions = append(
				conditions,
				&dst.CallExpr{
					Fun: gen.qualifiedIdentifier("IsSubType"),
					Args: []dst.Expr{
						dst.NewIdent("subType"),
						gen.qualifiedTypeIdent(typ.Name()),
					},
				},
			)
		}

		if len(conditions) == 0 {
			return []dst.Node{dst.NewIdent("false")}
		}

		result := conditions[0]
		for i := 1; i < len(conditions); i++ {
			result = &dst.BinaryExpr{
				X:  result,
				Op: token.LOR,
				Y:  conditions[i],
			}
		}

		return []dst.Node{result}
	default:
		panic(fmt.Errorf("unknown super type `%t` in `subtype` rule", superType))
	}
}

// qualifiedIdentifier creates a qualified identifier
func (gen *SubTypeCheckGenerator) qualifiedIdentifier(name string) dst.Expr {
	return dst.NewIdent(name)
}

// qualifiedTypeIdent creates a qualified type identifier, by
// prepending the package-qualifier,prefix, and appending `Type` suffix.
func (gen *SubTypeCheckGenerator) qualifiedTypeIdent(name string) dst.Expr {
	typeConstant := gen.getTypeConstant(name)
	if typeConstant == "" {
		panic(fmt.Errorf("empty type constant for name: %s", name))
	}
	return dst.NewIdent(typeConstant)
}

// parseCaseCondition parses a case condition to AST using Cadence types
func (gen *SubTypeCheckGenerator) parseCaseCondition(superType Type) dst.Expr {
	// Use type assertion to determine the specific type
	switch superType.(type) {
	case *OptionalType:
		return &dst.StarExpr{
			X: gen.qualifiedTypeIdent(superType.Name()),
		}
	default:
		// For simple types, use the type directly
		return gen.qualifiedTypeIdent(superType.Name())
	}
}

// getTypeConstant converts a type name to its Go constant
func (gen *SubTypeCheckGenerator) getTypeConstant(placeholderTypeName string) string {
	return placeholderTypeName + "Type"
}
