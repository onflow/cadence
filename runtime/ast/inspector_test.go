/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package ast_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/tests/examples"
)

// TestInspector_Elements compares Inspector against Inspect.
//
func TestInspector_Elements(t *testing.T) {

	t.Parallel()

	program, err := parser2.ParseProgram(examples.FungibleTokenContractInterface)
	require.NoError(t, err)

	inspector := ast.NewInspector(program)

	t.Run("all elements", func(t *testing.T) {

		t.Parallel()

		var elementsA []ast.Element
		inspector.Elements(nil, func(n ast.Element, push bool) bool {
			if push {
				elementsA = append(elementsA, n)
			}
			return true
		})

		var elementsB []ast.Element
		ast.Inspect(program, func(n ast.Element) bool {
			if n != nil {
				elementsB = append(elementsB, n)
			}
			return true
		})

		require.Equal(t, elementsA, elementsB)
	})

	t.Run("pruning", func(t *testing.T) {

		t.Parallel()

		var elementsA []ast.Element
		inspector.Elements(nil, func(n ast.Element, push bool) bool {
			if push {
				elementsA = append(elementsA, n)
				_, isCall := n.(*ast.InvocationExpression)
				return !isCall // don't descend into function calls
			}
			return false
		})

		var elementsB []ast.Element
		ast.Inspect(program, func(n ast.Element) bool {
			if n != nil {
				elementsB = append(elementsB, n)
				_, isCall := n.(*ast.InvocationExpression)
				return !isCall // don't descend into function calls
			}
			return false
		})

		require.Equal(t, elementsA, elementsB)
	})
}

func TestInspectorTypeFiltering(t *testing.T) {

	t.Parallel()

	const code = `
	  import 0x1

      fun f() {
	      print("hi")
          panic("oops")
      }
    `

	program, err := parser2.ParseProgram(code)
	require.NoError(t, err)

	inspector := ast.NewInspector(program)

	t.Run("Elements, no type filtering", func(t *testing.T) {

		var got []ast.ElementType

		inspector.Elements(nil, func(element ast.Element, push bool) bool {
			if push {
				got = append(got, element.ElementType())
			}
			return true
		})

		require.Equal(t,
			[]ast.ElementType{
				ast.ElementTypeProgram,
				ast.ElementTypeImportDeclaration,
				ast.ElementTypeFunctionDeclaration,
				ast.ElementTypeFunctionBlock,
				ast.ElementTypeBlock,
				ast.ElementTypeExpressionStatement,
				ast.ElementTypeInvocationExpression,
				ast.ElementTypeIdentifierExpression,
				ast.ElementTypeStringExpression,
				ast.ElementTypeExpressionStatement,
				ast.ElementTypeInvocationExpression,
				ast.ElementTypeIdentifierExpression,
				ast.ElementTypeStringExpression,
			},
			got,
		)
	})

	t.Run("Elements, type filtering", func(t *testing.T) {

		var got []ast.ElementType

		inspector.Elements(
			[]ast.Element{
				(*ast.StringExpression)(nil),
				(*ast.InvocationExpression)(nil),
			},
			func(element ast.Element, push bool) bool {
				if push {
					got = append(got, element.ElementType())
				}
				return true
			},
		)

		require.Equal(t,
			[]ast.ElementType{
				ast.ElementTypeInvocationExpression,
				ast.ElementTypeStringExpression,
				ast.ElementTypeInvocationExpression,
				ast.ElementTypeStringExpression,
			},
			got,
		)
	})

	t.Run("WithStack", func(t *testing.T) {

		var got [][]ast.ElementType

		inspector.WithStack(
			[]ast.Element{
				(*ast.StringExpression)(nil),
				(*ast.InvocationExpression)(nil),
			},
			func(element ast.Element, push bool, stack []ast.Element) bool {
				if push {
					var stackTypes []ast.ElementType
					for _, element := range stack {
						stackTypes = append(
							stackTypes,
							element.ElementType(),
						)
					}
					got = append(got, stackTypes)
				}
				return true
			},
		)

		require.Equal(t,
			[][]ast.ElementType{
				{
					ast.ElementTypeProgram,
					ast.ElementTypeFunctionDeclaration,
					ast.ElementTypeFunctionBlock,
					ast.ElementTypeBlock,
					ast.ElementTypeExpressionStatement,
					ast.ElementTypeInvocationExpression,
				},
				{
					ast.ElementTypeProgram,
					ast.ElementTypeFunctionDeclaration,
					ast.ElementTypeFunctionBlock,
					ast.ElementTypeBlock,
					ast.ElementTypeExpressionStatement,
					ast.ElementTypeInvocationExpression,
					ast.ElementTypeStringExpression,
				},
				{
					ast.ElementTypeProgram,
					ast.ElementTypeFunctionDeclaration,
					ast.ElementTypeFunctionBlock,
					ast.ElementTypeBlock,
					ast.ElementTypeExpressionStatement,
					ast.ElementTypeInvocationExpression,
				},
				{
					ast.ElementTypeProgram,
					ast.ElementTypeFunctionDeclaration,
					ast.ElementTypeFunctionBlock,
					ast.ElementTypeBlock,
					ast.ElementTypeExpressionStatement,
					ast.ElementTypeInvocationExpression,
					ast.ElementTypeStringExpression,
				},
			},
			got,
		)
	})

}
