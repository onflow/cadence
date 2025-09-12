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
		Type:  gen.qualifiedIdent("Type"),
	}
	superTypeParam := &dst.Field{
		Names: []*dst.Ident{dst.NewIdent("superType")},
		Type:  gen.qualifiedIdent("Type"),
	}

	// Create function body
	var stmts []dst.Stmt

	// Add early return for Never type
	neverTypeCheck := &dst.IfStmt{
		Cond: &dst.BinaryExpr{
			X:  dst.NewIdent("subType"),
			Op: token.EQL,
			Y:  gen.qualifiedIdent("Never"),
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
	caseCondition, err := gen.generateCaseCondition(superType)
	if err != nil {
		panic(fmt.Errorf("error generating case condition: %w", err))
	}

	// Parse case condition to AST
	caseExpr := gen.parseCaseCondition(caseCondition)

	// Parse rule condition directly to AST
	condition, err := parseRuleCondition(rule.Condition)
	if err != nil {
		panic(fmt.Errorf("error parsing rule condition: %w", err))
	}

	// Generate AST directly from predicate
	bodyStmt := gen.generatePredicateAST(condition)

	return &dst.CaseClause{
		List: []dst.Expr{caseExpr},
		Body: []dst.Stmt{bodyStmt},
	}
}

// generatePredicateAST generates AST directly from a rule predicate
func (gen *SubTypeCheckGenerator) generatePredicateAST(predicate RulePredicate) dst.Stmt {
	expr := gen.generatePredicateExpr(predicate)
	return &dst.ReturnStmt{
		Results: []dst.Expr{expr},
	}
}

// generatePredicateExpr generates an AST expression from a rule predicate
func (gen *SubTypeCheckGenerator) generatePredicateExpr(predicate RulePredicate) dst.Expr {
	switch p := predicate.(type) {
	case AlwaysCondition:
		return dst.NewIdent("true")

	case IsResourceCondition:
		return &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   dst.NewIdent("subType"),
				Sel: dst.NewIdent("IsResourceType"),
			},
		}

	case IsAttachmentCondition:
		return &dst.CallExpr{
			Fun:  dst.NewIdent("isAttachmentType"),
			Args: []dst.Expr{dst.NewIdent("subType")},
		}

	case IsHashableStructCondition:
		return &dst.CallExpr{
			Fun:  dst.NewIdent("IsHashableStructType"),
			Args: []dst.Expr{dst.NewIdent("subType")},
		}

	case IsStorableCondition:
		return &dst.CallExpr{
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
		}

	case NotCondition:
		innerExpr := gen.generatePredicateExpr(p.Condition)
		return &dst.UnaryExpr{
			Op: token.NOT,
			X:  &dst.ParenExpr{X: innerExpr},
		}

	case AndCondition:
		if len(p.Conditions) == 0 {
			return dst.NewIdent("true")
		}

		result := gen.generatePredicateExpr(p.Conditions[0])
		for i := 1; i < len(p.Conditions); i++ {
			right := gen.generatePredicateExpr(p.Conditions[i])
			result = &dst.BinaryExpr{
				X:  result,
				Op: token.LAND,
				Y:  right,
			}
		}
		return result

	case OrCondition:
		if len(p.Conditions) == 0 {
			return dst.NewIdent("false")
		}

		result := gen.generatePredicateExpr(p.Conditions[0])
		for i := 1; i < len(p.Conditions); i++ {
			right := gen.generatePredicateExpr(p.Conditions[i])
			result = &dst.BinaryExpr{
				X:  result,
				Op: token.LOR,
				Y:  right,
			}
		}
		return result

	case EqualsCondition:
		return gen.generateEqualsExpr(p)

	case SubtypeCondition:
		return gen.generateSubtypeExpr(p)

	case PermitsCondition:
		if len(p.Types) == 2 {
			return &dst.CallExpr{
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
		}
		return dst.NewIdent("false") // TODO: Implement permits condition

	case PurityCondition:
		return &dst.BinaryExpr{
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

	case TypeParamsEqualCondition:
		return &dst.BinaryExpr{
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

	case ParamsContravariantCondition:
		return dst.NewIdent("false") // TODO: Implement params contravariant check

	case ReturnCovariantCondition:
		return dst.NewIdent("false") // TODO: Implement return covariant check

	case ConstructorEqualCondition:
		return &dst.BinaryExpr{
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

	case ContainsCondition:
		if len(p.Types) == 2 {
			return &dst.CallExpr{
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
		}
		return dst.NewIdent("false") // TODO: Implement contains condition

	default:
		return dst.NewIdent("false") // TODO: Implement condition: " + predicate.GetType()
	}
}

// generateEqualsExpr generates AST for equals conditions
func (gen *SubTypeCheckGenerator) generateEqualsExpr(equals EqualsCondition) dst.Expr {
	switch target := equals.Target.(type) {
	case string:
		return &dst.BinaryExpr{
			X:  dst.NewIdent("subType"),
			Op: token.EQL,
			Y:  gen.qualifiedIdent(target),
		}
	case KeyValues:
		for key, value := range target {
			switch key {
			case "oneOf":
				oneOf := value.([]any)
				var cases []dst.Expr
				for _, t := range oneOf {
					if str, ok := t.(string); ok {
						cases = append(cases, gen.qualifiedIdent(str))
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
				caseClauses = append(caseClauses, &dst.CaseClause{
					Body: []dst.Stmt{
						&dst.ReturnStmt{
							Results: []dst.Expr{dst.NewIdent("false")},
						},
					},
				})

				// For now, generate a simple OR expression instead of IIFE
				if len(cases) == 0 {
					return dst.NewIdent("false")
				}

				result := &dst.BinaryExpr{
					X:  dst.NewIdent("subType"),
					Op: token.EQL,
					Y:  cases[0],
				}

				for i := 1; i < len(cases); i++ {
					right := &dst.BinaryExpr{
						X:  dst.NewIdent("subType"),
						Op: token.EQL,
						Y:  cases[i],
					}
					result = &dst.BinaryExpr{
						X:  result,
						Op: token.LOR,
						Y:  right,
					}
				}
				return result
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
func (gen *SubTypeCheckGenerator) generateSubtypeExpr(subtype SubtypeCondition) dst.Expr {
	switch superType := subtype.Super.(type) {
	case string:
		return &dst.CallExpr{
			Fun: dst.NewIdent("IsSubType"),
			Args: []dst.Expr{
				dst.NewIdent("subType"),
				gen.qualifiedIdent(superType),
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
								gen.qualifiedIdent(str),
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

// qualifiedIdent creates a qualified identifier
func (gen *SubTypeCheckGenerator) qualifiedIdent(name string) dst.Expr {
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
func (gen *SubTypeCheckGenerator) parseCaseCondition(condition string) dst.Expr {
	// For now, handle simple type constants
	if strings.HasPrefix(condition, gen.typePkgQualifier+".") {
		typeName := strings.TrimPrefix(condition, gen.typePkgQualifier+".")
		return &dst.SelectorExpr{
			X:   dst.NewIdent(gen.typePkgQualifier),
			Sel: dst.NewIdent(gen.getTypeConstant(typeName)),
		}
	}

	// Handle pointer types
	if strings.HasPrefix(condition, "*") {
		typeName := strings.TrimPrefix(condition, "*")
		return &dst.StarExpr{
			X: &dst.SelectorExpr{
				X:   dst.NewIdent(gen.typePkgQualifier),
				Sel: dst.NewIdent(gen.getTypeConstant(typeName)),
			},
		}
	}

	// Default case
	return dst.NewIdent(condition)
}

// generateCaseCondition generates the case condition for a type
func (gen *SubTypeCheckGenerator) generateCaseCondition(superType *TypeInfo) (string, error) {
	if superType.TypeName != "" && !superType.IsGeneric {
		return gen.typePkgQualifier + "." + superType.TypeName, nil
	}

	if superType.IsOptional {
		return "*OptionalType", nil
	}

	if superType.IsReference {
		return "*ReferenceType", nil
	}

	if superType.IsFunction {
		return "*FunctionType", nil
	}

	if superType.IsDictionary {
		return "*DictionaryType", nil
	}

	if superType.IsIntersection {
		return "*IntersectionType", nil
	}

	return "", fmt.Errorf("unsupported case condition type")
}

// getTypeConstant converts a type name to its Go constant
func (gen *SubTypeCheckGenerator) getTypeConstant(placeholderTypeName string) string {
	switch placeholderTypeName {
	case "Any":
		return "AnyType"
	case "AnyStruct":
		return "AnyStructType"
	case "AnyResource":
		return "AnyResourceType"
	case "AnyResourceAttachment":
		return "AnyResourceAttachmentType"
	case "AnyStructAttachment":
		return "AnyStructAttachmentType"
	case "HashableStruct":
		return "HashableStructType"
	case "Path":
		return "PathType"
	case "Storable":
		return "StorableType"
	case "CapabilityPath":
		return "CapabilityPathType"
	case "Number":
		return "NumberType"
	case "SignedNumber":
		return "SignedNumberType"
	case "Integer":
		return "IntegerType"
	case "SignedInteger":
		return "SignedIntegerType"
	case "FixedSizeUnsignedInteger":
		return "FixedSizeUnsignedIntegerType"
	case "FixedPoint":
		return "FixedPointType"
	case "SignedFixedPoint":
		return "SignedFixedPointType"
	case "UInt":
		return "UIntType"
	case "Int":
		return "IntType"
	case "Int8":
		return "Int8Type"
	case "Int16":
		return "Int16Type"
	case "Int32":
		return "Int32Type"
	case "Int64":
		return "Int64Type"
	case "Int128":
		return "Int128Type"
	case "Int256":
		return "Int256Type"
	case "UInt8":
		return "UInt8Type"
	case "UInt16":
		return "UInt16Type"
	case "UInt32":
		return "UInt32Type"
	case "UInt64":
		return "UInt64Type"
	case "UInt128":
		return "UInt128Type"
	case "UInt256":
		return "UInt256Type"
	case "Word8":
		return "Word8Type"
	case "Word16":
		return "Word16Type"
	case "Word32":
		return "Word32Type"
	case "Word64":
		return "Word64Type"
	case "Word128":
		return "Word128Type"
	case "Word256":
		return "Word256Type"
	case "UFix64":
		return "UFix64Type"
	case "UFix128":
		return "UFix128Type"
	case "Fix64":
		return "Fix64Type"
	case "Fix128":
		return "Fix128Type"
	case "StoragePath":
		return "StoragePathType"
	case "PrivatePath":
		return "PrivatePathType"
	case "PublicPath":
		return "PublicPathType"
	default:
		return placeholderTypeName + "Type"
	}
}
