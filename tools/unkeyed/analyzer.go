// Based on https://pkg.go.dev/golang.org/x/tools@v0.4.0/go/analysis/passes/composite
//
// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"golang.org/x/exp/typeparams"
)

var Analyzer = &analysis.Analyzer{
	Name:             "unkeyed",
	Doc:              "reports unkeyed composite literals",
	Requires:         []*analysis.Analyzer{inspect.Analyzer},
	RunDespiteErrors: true,
	Run:              run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
	}
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		cl := n.(*ast.CompositeLit)

		typ := pass.TypesInfo.Types[cl].Type
		if typ == nil {
			// cannot determine composite literals' type, skip it
			return
		}

		var structuralTypes []types.Type
		switch typ := typ.(type) {
		case *typeparams.TypeParam:
			terms, err := typeparams.NormalTerms(typ)
			if err != nil {
				return // invalid type
			}
			for _, term := range terms {
				structuralTypes = append(structuralTypes, term.Type())
			}
		default:
			structuralTypes = append(structuralTypes, typ)
		}

		for _, typ := range structuralTypes {
			under := deref(typ.Underlying())

			strct, ok := under.(*types.Struct)
			if !ok {
				// skip non-struct composite literals
				continue
			}


			// check if the struct contains an unkeyed field
			allKeyValue := true
			var suggestedFixAvailable = len(cl.Elts) == strct.NumFields()
			var missingKeys []analysis.TextEdit
			for i, e := range cl.Elts {
				if _, ok := e.(*ast.KeyValueExpr); !ok {
					allKeyValue = false
					if i >= strct.NumFields() {
						break
					}
					field := strct.Field(i)
					if !field.Exported() {
						// Adding unexported field names for structs not defined
						// locally will not work.
						suggestedFixAvailable = false
						break
					}
					missingKeys = append(missingKeys, analysis.TextEdit{
						Pos:     e.Pos(),
						End:     e.Pos(),
						NewText: []byte(fmt.Sprintf("%s: ", field.Name())),
					})
				}
			}
			if allKeyValue {
				// all the struct fields are keyed
				continue
			}
			diag := analysis.Diagnostic{
				Pos:     cl.Pos(),
				End:     cl.End(),
				Message: fmt.Sprintf("%s struct literal uses unkeyed fields", typ.String()),
			}
			if suggestedFixAvailable {
				diag.SuggestedFixes = []analysis.SuggestedFix{{
					Message:   "Add field names to struct literal",
					TextEdits: missingKeys,
				}}
			}
			pass.Report(diag)
			return
		}
	})
	return nil, nil
}

func deref(typ types.Type) types.Type {
	for {
		ptr, ok := typ.(*types.Pointer)
		if !ok {
			break
		}
		typ = ptr.Elem().Underlying()
	}
	return typ
}
