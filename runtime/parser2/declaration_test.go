/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package parser2

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestParseVariableDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("var, no type annotation, copy, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseStatements("var x = 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.VariableDeclaration{
					IsConstant: false,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       ast.Position{Line: 1, Column: 6, Offset: 6},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("let, no type annotation, copy, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseStatements("let x = 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.VariableDeclaration{
					IsConstant: true,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       ast.Position{Line: 1, Column: 6, Offset: 6},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("let, no type annotation, move, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseStatements("let x <- 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.VariableDeclaration{
					IsConstant: true,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationMove,
						Pos:       ast.Position{Line: 1, Column: 6, Offset: 6},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("let, resource type annotation, move, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseStatements("let r2: @R <- r")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.VariableDeclaration{
					IsConstant: true,
					Identifier: ast.Identifier{
						Identifier: "r2",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					TypeAnnotation: &ast.TypeAnnotation{
						IsResource: true,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "R",
								Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					},
					Value: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "r",
							Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationMove,
						Pos:       ast.Position{Line: 1, Column: 11, Offset: 11},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})
}

func TestParseParameterList(t *testing.T) {

	t.Parallel()

	parse := func(input string) (interface{}, []error) {
		return Parse(
			input,
			func(p *parser) interface{} {
				return parseParameterList(p)
			},
		)
	}

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ParameterList{
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("space", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(" (   )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ParameterList{
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
					EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
				},
			},
			result,
		)
	})

	t.Run("one, without argument label", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("( a : Int )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ParameterList{
				Parameters: []*ast.Parameter{
					{
						Label: "",
						Identifier: ast.Identifier{
							Identifier: "a",
							Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Line: 1, Column: 6, Offset: 6},
								},
							},
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
				},
			},
			result,
		)
	})

	t.Run("one, with argument label", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("( a b : Int )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ParameterList{
				Parameters: []*ast.Parameter{
					{
						Label: "a",
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
								},
							},
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
				},
			},
			result,
		)
	})

	t.Run("two, with and without argument label", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("( a b : Int , c : Int )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ParameterList{
				Parameters: []*ast.Parameter{
					{
						Label: "a",
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
								},
							},
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
					{
						Label: "",
						Identifier: ast.Identifier{
							Identifier: "c",
							Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
								},
							},
							StartPos: ast.Position{Line: 1, Column: 18, Offset: 18},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							EndPos:   ast.Position{Line: 1, Column: 20, Offset: 20},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 22, Offset: 22},
				},
			},
			result,
		)
	})
}

func TestParseFunctionDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("without return type", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseStatements("fun foo () { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.FunctionDeclaration{
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "",
								Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
								EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("with return type", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseStatements("fun foo (): X { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.FunctionDeclaration{
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "X",
								Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
								EndPos:   ast.Position{Line: 1, Column: 16, Offset: 16},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})
}
