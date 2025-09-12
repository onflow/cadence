package main

import (
	"fmt"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func testMinimal() {
	// Create a minimal AST with just one case
	file := &dst.File{
		Name: dst.NewIdent("main"),
		Decls: []dst.Decl{
			&dst.FuncDecl{
				Name: dst.NewIdent("checkSubTypeWithoutEquality"),
				Type: &dst.FuncType{
					Params: &dst.FieldList{
						List: []*dst.Field{
							{
								Names: []*dst.Ident{dst.NewIdent("subType")},
								Type:  &dst.SelectorExpr{X: dst.NewIdent("sema"), Sel: dst.NewIdent("Type")},
							},
							{
								Names: []*dst.Ident{dst.NewIdent("superType")},
								Type:  &dst.SelectorExpr{X: dst.NewIdent("sema"), Sel: dst.NewIdent("Type")},
							},
						},
					},
					Results: &dst.FieldList{
						List: []*dst.Field{
							{Type: dst.NewIdent("bool")},
						},
					},
				},
				Body: &dst.BlockStmt{
					List: []dst.Stmt{
						&dst.SwitchStmt{
							Tag: dst.NewIdent("superType"),
							Body: &dst.BlockStmt{
								List: []dst.Stmt{
									&dst.CaseClause{
										List: []dst.Expr{
											&dst.SelectorExpr{
												X:   dst.NewIdent("sema"),
												Sel: dst.NewIdent("AnyType"),
											},
										},
										Body: []dst.Stmt{
											&dst.ReturnStmt{
												Results: []dst.Expr{dst.NewIdent("true")},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	var buf strings.Builder
	if err := decorator.Fprint(&buf, file); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(buf.String())
}

func main() {
	testMinimal()
}
