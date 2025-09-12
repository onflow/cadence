package main

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

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

// generateCheckSubTypeWithoutEqualityFunction generates the complete checkSubTypeWithoutEquality function.
func (gen *SubTypeCheckGenerator) generateCheckSubTypeWithoutEqualityFunction(rules []Rule) (string, error) {
	// Create the file AST
	file := &dst.File{
		Name: dst.NewIdent("main"),
		Decls: []dst.Decl{
			gen.createImportDecl(),
			gen.createCheckSubTypeFunction(rules),
		},
	}

	// Convert AST to string
	var buf strings.Builder
	if err := decorator.Fprint(&buf, file); err != nil {
		return "", fmt.Errorf("error formatting AST: %w", err)
	}

	return buf.String(), nil
}

// createImportDecl creates the import declaration
func (gen *SubTypeCheckGenerator) createImportDecl() dst.Decl {
	return &dst.GenDecl{
		Tok: token.IMPORT,
		Specs: []dst.Spec{
			&dst.ImportSpec{
				Path: &dst.BasicLit{
					Kind:  token.STRING,
					Value: fmt.Sprintf("%q", gen.typePkgName),
				},
			},
		},
	}
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
		Name: dst.NewIdent("checkSubTypeWithoutEquality"),
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
	superType, err := parseType(rule.Super)
	if err != nil {
		panic(fmt.Errorf("error parsing super type: %w", err))
	}

	// Generate case condition
	caseExpr := gen.parseCaseCondition(superType)

	// Parse rule condition directly to AST
	condition, err := parseRulePredicate(rule.Predicate)
	if err != nil {
		panic(fmt.Errorf("error parsing rule condition: %w", err))
	}

	// Generate AST directly from predicate
	bodyStmts := gen.generatePredicateAST(condition)

	return &dst.CaseClause{
		List: []dst.Expr{caseExpr},
		Body: bodyStmts,
	}
}

// generatePredicateAST generates AST directly from a rule predicate
func (gen *SubTypeCheckGenerator) generatePredicateAST(predicate RulePredicate) []dst.Stmt {
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
				Results: []dst.Expr{dst.NewIdent("true")},
			},
		)
	case dst.Stmt:
		stmts = append(stmts, lastNode)
	default:
		panic(fmt.Errorf("error generating predicate AST: unexpected node type: %T", lastNode))
	}

	return stmts
}

// generatePredicate generates an AST expression from a rule predicate
func (gen *SubTypeCheckGenerator) generatePredicate(predicate RulePredicate) (result []dst.Node) {
	switch p := predicate.(type) {
	case AlwaysPredicate:
		return []dst.Node{dst.NewIdent("true")}

	case IsResourcePredicate:
		return []dst.Node{
			&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   dst.NewIdent("subType"),
					Sel: dst.NewIdent("IsResourceType"),
				},
			},
		}

	case IsAttachmentPredicate:
		return []dst.Node{
			&dst.CallExpr{
				Fun:  dst.NewIdent("isAttachmentType"),
				Args: []dst.Expr{dst.NewIdent("subType")},
			},
		}

	case IsHashableStructPredicate:
		return []dst.Node{
			&dst.CallExpr{
				Fun:  dst.NewIdent("IsHashableStructType"),
				Args: []dst.Expr{dst.NewIdent("subType")},
			},
		}

	case IsStorablePredicate:
		return []dst.Node{
			&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   dst.NewIdent("subType"),
					Sel: dst.NewIdent("IsStorable"),
				},
				Args: []dst.Expr{
					&dst.CompositeLit{
						Type: &dst.MapType{
							Key:   &dst.StarExpr{X: dst.NewIdent("Member")},
							Value: dst.NewIdent("bool"),
						},
					},
				},
			},
		}

	case NotPredicate:
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

	case AndPredicate:
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

	case OrPredicate:
		var exprs []dst.Expr
		for _, condition := range p.Predicates {
			generatedPredicatedNodes := gen.generatePredicate(condition)
			for _, node := range generatedPredicatedNodes {
				switch node := node.(type) {
				case dst.Stmt:
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

	case EqualsPredicate:
		return []dst.Node{gen.generateEqualsExpr(p)}

	case SubtypePredicate:
		return []dst.Node{gen.generateSubtypeExpr(p)}

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

	case TypeParamsEqualPredicate:
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

	case ParamsContravariantPredicate:
		return []dst.Node{dst.NewIdent("false")} // TODO: Implement params contravariant check

	case ReturnCovariantPredicate:
		return []dst.Node{dst.NewIdent("false")} // TODO: Implement return covariant check

	case ConstructorEqualPredicate:
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

// generateEqualsExpr generates AST for equals conditions
func (gen *SubTypeCheckGenerator) generateEqualsExpr(equals EqualsPredicate) dst.Node {
	switch target := equals.Target.(type) {
	case string:
		return &dst.BinaryExpr{
			X:  dst.NewIdent("subType"),
			Op: token.EQL,
			Y:  gen.qualifiedTypeIdent(target),
		}
	case KeyValues:
		for key, value := range target {
			switch key {
			case "oneOf":
				oneOf := value.([]any)

				value := oneOf[1].(string)
				if len(oneOf) == 1 {
					return &dst.BinaryExpr{
						X:  dst.NewIdent("subType"),
						Op: token.EQL,
						Y:  gen.qualifiedTypeIdent(value),
					}
				} else {
					var cases []dst.Expr
					for _, t := range oneOf {
						if str, ok := t.(string); ok {
							cases = append(cases, gen.qualifiedTypeIdent(str))
						}
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

					return &dst.SwitchStmt{
						Tag: dst.NewIdent("subType"),
						Body: &dst.BlockStmt{
							List: caseClauses,
						},
					}
				}
			default:
				panic(fmt.Errorf("unsupported rule `%s` for `target` of `equals` rule", key))
			}
		}
	default:
		panic(fmt.Errorf("unknown target type %t in `equals` rule", target))
	}
	return dst.NewIdent("false")
}

// generateSubtypeExpr generates AST for subtype conditions
func (gen *SubTypeCheckGenerator) generateSubtypeExpr(subtype SubtypePredicate) dst.Expr {
	switch superType := subtype.Super.(type) {
	case string:
		return &dst.CallExpr{
			Fun: dst.NewIdent("IsSubType"),
			Args: []dst.Expr{
				dst.NewIdent("subType"),
				gen.qualifiedTypeIdent(superType),
			},
		}
	case KeyValues:
		for key, value := range superType {
			switch key {
			case "oneOf":
				oneOf := value.([]any)
				var conditions []dst.Expr
				for _, t := range oneOf {
					if str, ok := t.(string); ok {
						conditions = append(conditions, &dst.CallExpr{
							Fun: dst.NewIdent("IsSubType"),
							Args: []dst.Expr{
								dst.NewIdent("subType"),
								gen.qualifiedTypeIdent(str),
							},
						})
					}
				}

				if len(conditions) == 0 {
					return dst.NewIdent("false")
				}

				result := conditions[0]
				for i := 1; i < len(conditions); i++ {
					result = &dst.BinaryExpr{
						X:  result,
						Op: token.LOR,
						Y:  conditions[i],
					}
				}
				return result
			default:
				panic(fmt.Errorf("unsupported rule `%s` for `super` of `subtype` rule", key))
			}
		}
	default:
		panic(fmt.Errorf("unknown super type %t in `subtype` rule", superType))
	}
	return dst.NewIdent("false")
}

// qualifiedIdentifier creates a qualified identifier
func (gen *SubTypeCheckGenerator) qualifiedIdentifier(name string) dst.Expr {
	return &dst.SelectorExpr{
		X:   dst.NewIdent(gen.typePkgQualifier),
		Sel: dst.NewIdent(name),
	}
}

// qualifiedTypeIdent creates a qualified type identifier, by
// prepending the package-qualifier,prefix, and appending `Type` suffix.
func (gen *SubTypeCheckGenerator) qualifiedTypeIdent(name string) dst.Expr {
	typeConstant := gen.getTypeConstant(name)
	if typeConstant == "" {
		panic(fmt.Errorf("empty type constant for name: %s", name))
	}
	return &dst.SelectorExpr{
		X:   dst.NewIdent(gen.typePkgQualifier),
		Sel: dst.NewIdent(typeConstant),
	}
}

// parseCaseCondition parses a case condition string to AST
func (gen *SubTypeCheckGenerator) parseCaseCondition(superType *TypeInfo) dst.Expr {
	// For now, handle simple type constants
	superTypeName := superType.TypeName
	if superTypeName != "" && !superType.IsGeneric {
		return &dst.SelectorExpr{
			X:   dst.NewIdent(gen.typePkgQualifier),
			Sel: dst.NewIdent(gen.getTypeConstant(superTypeName)),
		}
	}

	// Handle pointer types
	return &dst.StarExpr{
		X: &dst.SelectorExpr{
			X:   dst.NewIdent(gen.typePkgQualifier),
			Sel: dst.NewIdent(gen.getTypeConstant(superTypeName)),
		},
	}
}

// getTypeConstant converts a type name to its Go constant
func (gen *SubTypeCheckGenerator) getTypeConstant(placeholderTypeName string) string {
	return placeholderTypeName + "Type"
}
