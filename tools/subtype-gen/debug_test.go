package main

import (
	"fmt"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func debugAST() {
	// Create a simple AST
	file := &dst.File{
		Name: dst.NewIdent("main"),
		Decls: []dst.Decl{
			&dst.FuncDecl{
				Name: dst.NewIdent("test"),
				Type: &dst.FuncType{
					Results: &dst.FieldList{
						List: []*dst.Field{
							{Type: dst.NewIdent("bool")},
						},
					},
				},
				Body: &dst.BlockStmt{
					List: []dst.Stmt{
						&dst.ReturnStmt{
							Results: []dst.Expr{dst.NewIdent("true")},
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
	debugAST()
}
