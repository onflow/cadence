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

package parser

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestParseVariableDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("var, no type annotation, copy, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("var x = 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					IsConstant: false,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
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

	t.Run("var, no type annotation, copy, one value, pub", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" pub var x = 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					Access:     ast.AccessPublic,
					IsConstant: false,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 13, Offset: 13},
							EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       ast.Position{Line: 1, Column: 11, Offset: 11},
					},
					StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("let, no type annotation, copy, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("let x = 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					IsConstant: true,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
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

		result, errs := testParseDeclarations("let x <- 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					IsConstant: true,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
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

		result, errs := testParseDeclarations("let r2: @R <- r")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
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

	t.Run("var, no type annotation, copy, two values ", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("var x <- y <- z")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.VariableDeclaration{
					IsConstant: false,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "y",
							Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationMove,
						Pos:       ast.Position{Line: 1, Column: 6, Offset: 6},
					},
					SecondValue: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "z",
							Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					SecondTransfer: &ast.Transfer{
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

	parse := func(input string) (any, []error) {
		return Parse(
			[]byte(input),
			func(p *parser) (any, error) {
				return parseParameterList(p)
			},
			nil,
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

	t.Run("two, with and without argument label, missing comma", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("( a b : Int   c : Int )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&MissingCommaInParameterListError{
					Pos: ast.Position{Offset: 14, Line: 1, Column: 14},
				},
			},
			errs,
		)
	})
}

func TestParseFunctionDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("without return type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("fun foo () { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
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

	t.Run("without return type, pub", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("pub fun foo () { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessPublic,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
							EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "",
								Pos:        ast.Position{Line: 1, Column: 13, Offset: 13},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 13, Offset: 13},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 15, Offset: 15},
								EndPos:   ast.Position{Line: 1, Column: 17, Offset: 17},
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

		result, errs := testParseDeclarations("fun foo (): X { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
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

	t.Run("without return type, with pre and post conditions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements(`
          fun foo () {
              pre {
                 true : "test"
                 2 > 1 : "foo"
              }

              post {
                 false
              }

              bar()
          }
        `)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.FunctionDeclaration{
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 2, Column: 14, Offset: 15},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 18, Offset: 19},
							EndPos:   ast.Position{Line: 2, Column: 19, Offset: 20},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "",
								Pos:        ast.Position{Line: 2, Column: 19, Offset: 20},
							},
						},
						StartPos: ast.Position{Line: 2, Column: 19, Offset: 20},
					},
					FunctionBlock: &ast.FunctionBlock{
						PreConditions: &ast.Conditions{
							{
								Kind: ast.ConditionKindPre,
								Test: &ast.BoolExpression{
									Value: true,
									Range: ast.Range{
										StartPos: ast.Position{Line: 4, Column: 17, Offset: 61},
										EndPos:   ast.Position{Line: 4, Column: 20, Offset: 64},
									},
								},
								Message: &ast.StringExpression{
									Value: "test",
									Range: ast.Range{
										StartPos: ast.Position{Line: 4, Column: 24, Offset: 68},
										EndPos:   ast.Position{Line: 4, Column: 29, Offset: 73},
									},
								},
							},
							{
								Kind: ast.ConditionKindPre,
								Test: &ast.BinaryExpression{
									Operation: ast.OperationGreater,
									Left: &ast.IntegerExpression{
										PositiveLiteral: []byte("2"),
										Value:           big.NewInt(2),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Line: 5, Column: 17, Offset: 92},
											EndPos:   ast.Position{Line: 5, Column: 17, Offset: 92},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("1"),
										Value:           big.NewInt(1),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Line: 5, Column: 21, Offset: 96},
											EndPos:   ast.Position{Line: 5, Column: 21, Offset: 96},
										},
									},
								},
								Message: &ast.StringExpression{
									Value: "foo",
									Range: ast.Range{
										StartPos: ast.Position{Line: 5, Column: 25, Offset: 100},
										EndPos:   ast.Position{Line: 5, Column: 29, Offset: 104},
									},
								},
							},
						},
						PostConditions: &ast.Conditions{
							{
								Kind: ast.ConditionKindPost,
								Test: &ast.BoolExpression{
									Value: false,
									Range: ast.Range{
										StartPos: ast.Position{Line: 9, Column: 17, Offset: 161},
										EndPos:   ast.Position{Line: 9, Column: 21, Offset: 165},
									},
								},
								Message: nil,
							},
						},
						Block: &ast.Block{
							Statements: []ast.Statement{
								&ast.ExpressionStatement{
									Expression: &ast.InvocationExpression{
										InvokedExpression: &ast.IdentifierExpression{
											Identifier: ast.Identifier{
												Identifier: "bar",
												Pos:        ast.Position{Line: 12, Column: 14, Offset: 198},
											},
										},
										ArgumentsStartPos: ast.Position{Line: 12, Column: 17, Offset: 201},
										EndPos:            ast.Position{Line: 12, Column: 18, Offset: 202},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Line: 2, Column: 21, Offset: 22},
								EndPos:   ast.Position{Line: 13, Column: 10, Offset: 214},
							},
						},
					},
					StartPos: ast.Position{Line: 2, Column: 10, Offset: 11},
				},
			},
			result,
		)
	})

	t.Run("with docstring, single line comment", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("/// Test\nfun foo() {}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 2, Column: 4, Offset: 13},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 7, Offset: 16},
							EndPos:   ast.Position{Line: 2, Column: 8, Offset: 17},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "",
								Pos:        ast.Position{Line: 2, Column: 8, Offset: 17},
							},
						},
						StartPos: ast.Position{Line: 2, Column: 8, Offset: 17},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 2, Column: 10, Offset: 19},
								EndPos:   ast.Position{Line: 2, Column: 11, Offset: 20},
							},
						},
					},
					DocString: " Test",
					StartPos:  ast.Position{Line: 2, Column: 0, Offset: 9},
				},
			},
			result,
		)
	})

	t.Run("with docstring, two line comments", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("\n  /// First line\n  \n/// Second line\n\n\nfun foo() {}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 7, Column: 4, Offset: 43},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 7, Column: 7, Offset: 46},
							EndPos:   ast.Position{Line: 7, Column: 8, Offset: 47},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "",
								Pos:        ast.Position{Line: 7, Column: 8, Offset: 47},
							},
						},
						StartPos: ast.Position{Line: 7, Column: 8, Offset: 47},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 7, Column: 10, Offset: 49},
								EndPos:   ast.Position{Line: 7, Column: 11, Offset: 50},
							},
						},
					},
					DocString: " First line\n Second line",
					StartPos:  ast.Position{Line: 7, Column: 0, Offset: 39},
				},
			},
			result,
		)
	})

	t.Run("with docstring, block comment", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("\n    /** Cool dogs.\n\n Cool cats!! */\n\n\nfun foo() {}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 7, Column: 4, Offset: 43},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 7, Column: 7, Offset: 46},
							EndPos:   ast.Position{Line: 7, Column: 8, Offset: 47},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "",
								Pos:        ast.Position{Line: 7, Column: 8, Offset: 47},
							},
						},
						StartPos: ast.Position{Line: 7, Column: 8, Offset: 47},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 7, Column: 10, Offset: 49},
								EndPos:   ast.Position{Line: 7, Column: 11, Offset: 50},
							},
						},
					},
					DocString: " Cool dogs.\n\n Cool cats!! ",
					StartPos:  ast.Position{Line: 7, Column: 0, Offset: 39},
				},
			},
			result,
		)
	})

	t.Run("without space after return type", func(t *testing.T) {

		// A brace after the return type is ambiguous:
		// It could be the start of a restricted type.
		// However, if there is space after the brace, which is most common
		// in function declarations, we consider it not a restricted type

		t.Parallel()

		result, errs := testParseDeclarations("fun main(): Int{ return 1 }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Identifier: ast.Identifier{
						Identifier: "main",
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
								Identifier: "Int",
								Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Statements: []ast.Statement{
								&ast.ReturnStatement{
									Expression: &ast.IntegerExpression{
										PositiveLiteral: []byte("1"),
										Value:           big.NewInt(1),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Line: 1, Column: 24, Offset: 24},
											EndPos:   ast.Position{Line: 1, Column: 24, Offset: 24},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
										EndPos:   ast.Position{Line: 1, Column: 24, Offset: 24},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 15, Offset: 15},
								EndPos:   ast.Position{Line: 1, Column: 26, Offset: 26},
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

func TestParseAccess(t *testing.T) {

	t.Parallel()

	parse := func(input string) (any, []error) {
		return Parse(
			[]byte(input),
			func(p *parser) (any, error) {
				return parseAccess(p)
			},
			nil,
		)
	}

	t.Run("pub", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("pub")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.AccessPublic,
			result,
		)
	})

	t.Run("pub(set)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("pub ( set )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.AccessPublicSettable,
			result,
		)
	})

	t.Run("pub, missing set keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("pub ( ")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected keyword \"set\", got EOF",
					Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			nil,
			result,
		)
	})

	t.Run("pub, missing closing paren", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("pub ( set ")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token ')'",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			nil,
			result,
		)
	})

	t.Run("pub, invalid inner keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("pub ( foo )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected keyword \"set\", got \"foo\"",
					Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			nil,
			result,
		)
	})

	t.Run("priv", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("priv")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.AccessPrivate,
			result,
		)
	})

	t.Run("access(all)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( all )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.AccessPublic,
			result,
		)
	})

	t.Run("access(account)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( account )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.AccessAccount,
			result,
		)
	})

	t.Run("access(contract)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( contract )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.AccessContract,
			result,
		)
	})

	t.Run("access(self)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( self )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.AccessPrivate,
			result,
		)
	})

	t.Run("access, missing keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( ")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected keyword \"all\", \"account\", \"contract\", or \"self\", got EOF",
					Pos:     ast.Position{Offset: 9, Line: 1, Column: 9},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			nil,
			result,
		)
	})

	t.Run("access, missing closing paren", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( self ")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token ')'",
					Pos:     ast.Position{Offset: 14, Line: 1, Column: 14},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			nil,
			result,
		)
	})

	t.Run("access, invalid inner keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected keyword \"all\", \"account\", \"contract\", or \"self\", got \"foo\"",
					Pos:     ast.Position{Offset: 9, Line: 1, Column: 9},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			nil,
			result,
		)
	})
}

func TestParseImportDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("no identifiers, missing location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected end in import declaration: expected string, address, or identifier",
					Pos:     ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		utils.AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("no identifiers, string location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import "foo"`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Identifiers: nil,
					Location:    common.StringLocation("foo"),
					LocationPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
					},
				},
			},
			result,
		)
	})

	t.Run("no identifiers, address location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import 0x42`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Identifiers: nil,
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
					},
				},
			},
			result,
		)
	})

	t.Run("no identifiers, address location, address too long", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import 0x10000000000000001`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "address too large",
					Pos:     ast.Position{Offset: 8, Line: 1, Column: 8},
				},
			},
			errs,
		)

		expected := []ast.Declaration{
			&ast.ImportDeclaration{
				Identifiers: nil,
				Location: common.AddressLocation{
					Address: common.MustBytesToAddress([]byte{0x0}),
				},
				LocationPos: ast.Position{Line: 1, Column: 8, Offset: 8},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
					EndPos:   ast.Position{Line: 1, Column: 26, Offset: 26},
				},
			},
		}

		utils.AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("no identifiers, integer location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import 1`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token in import declaration: " +
						"got decimal integer, expected string, address, or identifier",
					Pos: ast.Position{Offset: 8, Line: 1, Column: 8},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		utils.AssertEqualWithDiff(t,
			expected,
			result,
		)

	})

	t.Run("one identifier, string location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo from "bar"`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Identifiers: []ast.Identifier{
						{
							Identifier: "foo",
							Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Location:    common.StringLocation("bar"),
					LocationPos: ast.Position{Line: 1, Column: 17, Offset: 17},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 21, Offset: 21},
					},
				},
			},
			result,
		)
	})

	t.Run("one identifier, string location, missing from keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo "bar"`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token in import declaration: " +
						"got string, expected keyword \"from\" or ','",
					Pos: ast.Position{Offset: 12, Line: 1, Column: 12},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		utils.AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("three identifiers, address location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo , bar , baz from 0x42`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Identifiers: []ast.Identifier{
						{
							Identifier: "foo",
							Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
						},
						{
							Identifier: "bar",
							Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
						},
						{
							Identifier: "baz",
							Pos:        ast.Position{Line: 1, Column: 20, Offset: 20},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 1, Column: 29, Offset: 29},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 32, Offset: 32},
					},
				},
			},
			result,
		)
	})

	t.Run("two identifiers, address location, extra comma", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo , bar , from 0x42`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: `expected identifier, got keyword "from"`,
					Pos:     ast.Position{Offset: 20, Line: 1, Column: 20},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		utils.AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("no identifiers, identifier location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Identifiers: nil,
					Location:    common.IdentifierLocation("foo"),
					LocationPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("from keyword as second identifier", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
			import foo, from from 0x42
			import foo, from, bar from 0x42
		`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Identifiers: []ast.Identifier{
						{
							Identifier: "foo",
							Pos:        ast.Position{Line: 2, Column: 10, Offset: 11},
						},
						{
							Identifier: "from",
							Pos:        ast.Position{Line: 2, Column: 15, Offset: 16},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 2, Column: 25, Offset: 26},
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 3, Offset: 4},
						EndPos:   ast.Position{Line: 2, Column: 28, Offset: 29},
					},
				},
				&ast.ImportDeclaration{
					Identifiers: []ast.Identifier{
						{
							Identifier: "foo",
							Pos:        ast.Position{Line: 3, Column: 10, Offset: 41},
						},
						{
							Identifier: "from",
							Pos:        ast.Position{Line: 3, Column: 15, Offset: 46},
						},
						{
							Identifier: "bar",
							Pos:        ast.Position{Line: 3, Column: 21, Offset: 52},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 3, Column: 30, Offset: 61},
					Range: ast.Range{
						StartPos: ast.Position{Line: 3, Column: 3, Offset: 34},
						EndPos:   ast.Position{Line: 3, Column: 33, Offset: 64},
					},
				},
			},
			result,
		)
	})
}

func TestParseEvent(t *testing.T) {

	t.Parallel()

	t.Run("no parameters", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("event E()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					CompositeKind: common.CompositeKindEvent,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Offset: 6, Line: 1, Column: 6},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindInitializer,
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
											EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
										},
									},
									StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
								},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
			},
			result,
		)
	})

	t.Run("two parameters, private", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" priv event E2 ( a : Int , b : String )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{

				&ast.CompositeDeclaration{
					Access:        ast.AccessPrivate,
					CompositeKind: common.CompositeKindEvent,
					Identifier: ast.Identifier{
						Identifier: "E2",
						Pos:        ast.Position{Offset: 12, Line: 1, Column: 12},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindInitializer,
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												Label: "",
												Identifier: ast.Identifier{
													Identifier: "a",
													Pos:        ast.Position{Offset: 17, Line: 1, Column: 17},
												},
												TypeAnnotation: &ast.TypeAnnotation{
													IsResource: false,
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "Int",
															Pos:        ast.Position{Offset: 21, Line: 1, Column: 21},
														},
													},
													StartPos: ast.Position{Offset: 21, Line: 1, Column: 21},
												},
												Range: ast.Range{
													StartPos: ast.Position{Offset: 17, Line: 1, Column: 17},
													EndPos:   ast.Position{Offset: 23, Line: 1, Column: 23},
												},
											},
											{
												Label: "",
												Identifier: ast.Identifier{
													Identifier: "b",
													Pos:        ast.Position{Offset: 27, Line: 1, Column: 27},
												},
												TypeAnnotation: &ast.TypeAnnotation{
													IsResource: false,
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "String",
															Pos:        ast.Position{Offset: 31, Line: 1, Column: 31},
														},
													},
													StartPos: ast.Position{Offset: 31, Line: 1, Column: 31},
												},
												Range: ast.Range{
													StartPos: ast.Position{Offset: 27, Line: 1, Column: 27},
													EndPos:   ast.Position{Offset: 36, Line: 1, Column: 36},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 15, Line: 1, Column: 15},
											EndPos:   ast.Position{Offset: 38, Line: 1, Column: 38},
										},
									},
									StartPos: ast.Position{Offset: 15, Line: 1, Column: 15},
								},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 38, Line: 1, Column: 38},
					},
				},
			},
			result,
		)
	})
}

func TestParseFieldWithVariableKind(t *testing.T) {

	t.Parallel()

	parse := func(input string) (any, []error) {
		return Parse(
			[]byte(input),
			func(p *parser) (any, error) {
				return parseFieldWithVariableKind(p, ast.AccessNotSpecified, nil, "")
			},
			nil,
		)
	}

	t.Run("variable", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("var x : Int")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				VariableKind: ast.VariableKindVariable,
				Identifier: ast.Identifier{
					Identifier: "x",
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
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
				},
			},
			result,
		)
	})

	t.Run("constant", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("let x : Int")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				VariableKind: ast.VariableKindConstant,
				Identifier: ast.Identifier{
					Identifier: "x",
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
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
				},
			},
			result,
		)
	})
}

func TestParseCompositeDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("struct, no conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" pub struct S { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessPublic,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
					},
					Members: &ast.Members{},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 16, Offset: 16},
					},
				},
			},
			result,
		)
	})

	t.Run("resource, one conformance", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" pub resource R : RI { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessPublic,
					CompositeKind: common.CompositeKindResource,
					Identifier: ast.Identifier{
						Identifier: "R",
						Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
					},
					Conformances: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "RI",
								Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
							},
						},
					},
					Members: &ast.Members{},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 23, Offset: 23},
					},
				},
			},
			result,
		)
	})

	t.Run("struct, with fields, functions, and special functions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct Test {
              pub(set) var foo: Int

              init(foo: Int) {
                  self.foo = foo
              }

              pub fun getFoo(): Int {
                  return self.foo
              }
          }
	    `)

		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FieldDeclaration{
								Access:       ast.AccessPublicSettable,
								VariableKind: ast.VariableKindVariable,
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos:        ast.Position{Offset: 52, Line: 3, Column: 27},
								},
								TypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 57, Line: 3, Column: 32},
										},
									},
									StartPos: ast.Position{Offset: 57, Line: 3, Column: 32},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 39, Line: 3, Column: 14},
									EndPos:   ast.Position{Offset: 59, Line: 3, Column: 34},
								},
							},
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindInitializer,
								FunctionDeclaration: &ast.FunctionDeclaration{
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos:        ast.Position{Offset: 76, Line: 5, Column: 14},
									},
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												Label: "",
												Identifier: ast.Identifier{
													Identifier: "foo",
													Pos:        ast.Position{Offset: 81, Line: 5, Column: 19},
												},
												TypeAnnotation: &ast.TypeAnnotation{
													IsResource: false,
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "Int",
															Pos:        ast.Position{Offset: 86, Line: 5, Column: 24},
														},
													},
													StartPos: ast.Position{Offset: 86, Line: 5, Column: 24},
												},
												Range: ast.Range{
													StartPos: ast.Position{Offset: 81, Line: 5, Column: 19},
													EndPos:   ast.Position{Offset: 88, Line: 5, Column: 26},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 80, Line: 5, Column: 18},
											EndPos:   ast.Position{Offset: 89, Line: 5, Column: 27},
										},
									},
									FunctionBlock: &ast.FunctionBlock{
										Block: &ast.Block{
											Statements: []ast.Statement{
												&ast.AssignmentStatement{
													Target: &ast.MemberExpression{
														Expression: &ast.IdentifierExpression{
															Identifier: ast.Identifier{
																Identifier: "self",
																Pos:        ast.Position{Offset: 111, Line: 6, Column: 18},
															},
														},
														AccessPos: ast.Position{Offset: 115, Line: 6, Column: 22},
														Identifier: ast.Identifier{
															Identifier: "foo",
															Pos:        ast.Position{Offset: 116, Line: 6, Column: 23},
														},
													},
													Transfer: &ast.Transfer{
														Operation: ast.TransferOperationCopy,
														Pos:       ast.Position{Offset: 120, Line: 6, Column: 27},
													},
													Value: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "foo",
															Pos:        ast.Position{Offset: 122, Line: 6, Column: 29},
														},
													},
												},
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 91, Line: 5, Column: 29},
												EndPos:   ast.Position{Offset: 140, Line: 7, Column: 14},
											},
										},
									},
									StartPos: ast.Position{Offset: 76, Line: 5, Column: 14},
								},
							},
							&ast.FunctionDeclaration{
								Access: ast.AccessPublic,
								Identifier: ast.Identifier{
									Identifier: "getFoo",
									Pos:        ast.Position{Offset: 165, Line: 9, Column: 22},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 171, Line: 9, Column: 28},
										EndPos:   ast.Position{Offset: 172, Line: 9, Column: 29},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 175, Line: 9, Column: 32},
										},
									},
									StartPos: ast.Position{Offset: 175, Line: 9, Column: 32},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Statements: []ast.Statement{
											&ast.ReturnStatement{
												Expression: &ast.MemberExpression{
													Expression: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "self",
															Pos:        ast.Position{Offset: 206, Line: 10, Column: 25},
														},
													},
													AccessPos: ast.Position{Offset: 210, Line: 10, Column: 29},
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos:        ast.Position{Offset: 211, Line: 10, Column: 30},
													},
												},
												Range: ast.Range{
													StartPos: ast.Position{Offset: 199, Line: 10, Column: 18},
													EndPos:   ast.Position{Offset: 213, Line: 10, Column: 32},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 179, Line: 9, Column: 36},
											EndPos:   ast.Position{Offset: 229, Line: 11, Column: 14},
										},
									},
								},
								StartPos: ast.Position{Offset: 157, Line: 9, Column: 14},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 241, Line: 12, Column: 10},
					},
				},
			},
			result,
		)
	})
}

func TestParseInterfaceDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("struct, no conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" pub struct interface S { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.InterfaceDeclaration{
					Access:        ast.AccessPublic,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 1, Column: 22, Offset: 22},
					},
					Members: &ast.Members{},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 26, Offset: 26},
					},
				},
			},
			result,
		)
	})

	t.Run("struct, interface keyword as name", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" pub struct interface interface { }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected interface name, got keyword \"interface\"",
					Pos:     ast.Position{Offset: 22, Line: 1, Column: 22},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		utils.AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("struct, with fields, functions, and special functions; with and without blocks", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct interface Test {
              pub(set) var foo: Int

              init(foo: Int)

              pub fun getFoo(): Int

              pub fun getBar(): Int {}

              destroy() {}
          }
	    `)

		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.InterfaceDeclaration{
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Offset: 28, Line: 2, Column: 27},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FieldDeclaration{
								Access:       ast.AccessPublicSettable,
								VariableKind: ast.VariableKindVariable,
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos:        ast.Position{Offset: 62, Line: 3, Column: 27},
								},
								TypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 67, Line: 3, Column: 32},
										},
									},
									StartPos: ast.Position{Offset: 67, Line: 3, Column: 32},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 49, Line: 3, Column: 14},
									EndPos:   ast.Position{Offset: 69, Line: 3, Column: 34},
								},
							},
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindInitializer,
								FunctionDeclaration: &ast.FunctionDeclaration{
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos:        ast.Position{Offset: 86, Line: 5, Column: 14},
									},
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												Label: "",
												Identifier: ast.Identifier{
													Identifier: "foo",
													Pos:        ast.Position{Offset: 91, Line: 5, Column: 19},
												},
												TypeAnnotation: &ast.TypeAnnotation{
													IsResource: false,
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "Int",
															Pos:        ast.Position{Offset: 96, Line: 5, Column: 24},
														},
													},
													StartPos: ast.Position{Offset: 96, Line: 5, Column: 24},
												},
												Range: ast.Range{
													StartPos: ast.Position{Offset: 91, Line: 5, Column: 19},
													EndPos:   ast.Position{Offset: 98, Line: 5, Column: 26},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 90, Line: 5, Column: 18},
											EndPos:   ast.Position{Offset: 99, Line: 5, Column: 27},
										},
									},
									StartPos: ast.Position{Offset: 86, Line: 5, Column: 14},
								},
							},
							&ast.FunctionDeclaration{
								Access: ast.AccessPublic,
								Identifier: ast.Identifier{
									Identifier: "getFoo",
									Pos:        ast.Position{Offset: 124, Line: 7, Column: 22},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 130, Line: 7, Column: 28},
										EndPos:   ast.Position{Offset: 131, Line: 7, Column: 29},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 134, Line: 7, Column: 32},
										},
									},
									StartPos: ast.Position{Offset: 134, Line: 7, Column: 32},
								},
								StartPos: ast.Position{Offset: 116, Line: 7, Column: 14},
							},
							&ast.FunctionDeclaration{
								Access: ast.AccessPublic,
								Identifier: ast.Identifier{
									Identifier: "getBar",
									Pos:        ast.Position{Offset: 161, Line: 9, Column: 22},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 167, Line: 9, Column: 28},
										EndPos:   ast.Position{Offset: 168, Line: 9, Column: 29},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 171, Line: 9, Column: 32},
										},
									},
									StartPos: ast.Position{Offset: 171, Line: 9, Column: 32},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 175, Line: 9, Column: 36},
											EndPos:   ast.Position{Offset: 176, Line: 9, Column: 37},
										},
									},
								},
								StartPos: ast.Position{Offset: 153, Line: 9, Column: 14},
							},
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindDestructor,
								FunctionDeclaration: &ast.FunctionDeclaration{
									Identifier: ast.Identifier{
										Identifier: "destroy",
										Pos:        ast.Position{Offset: 193, Line: 11, Column: 14},
									},
									ParameterList: &ast.ParameterList{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 200, Line: 11, Column: 21},
											EndPos:   ast.Position{Offset: 201, Line: 11, Column: 22},
										},
									},
									FunctionBlock: &ast.FunctionBlock{
										Block: &ast.Block{
											Range: ast.Range{
												StartPos: ast.Position{Offset: 203, Line: 11, Column: 24},
												EndPos:   ast.Position{Offset: 204, Line: 11, Column: 25},
											},
										},
									},
									StartPos: ast.Position{Offset: 193, Line: 11, Column: 14},
								},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 216, Line: 12, Column: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("enum, two cases one one line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" pub enum E { case c ; pub case d }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessPublic,
					CompositeKind: common.CompositeKindEnum,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Line: 1, Column: 10, Offset: 10},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.EnumCaseDeclaration{
								Access: ast.AccessNotSpecified,
								Identifier: ast.Identifier{
									Identifier: "c",
									Pos:        ast.Position{Line: 1, Column: 19, Offset: 19},
								},
								StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							},
							&ast.EnumCaseDeclaration{
								Access: ast.AccessPublic,
								Identifier: ast.Identifier{
									Identifier: "d",
									Pos:        ast.Position{Line: 1, Column: 32, Offset: 32},
								},
								StartPos: ast.Position{Line: 1, Column: 23, Offset: 23},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 34, Offset: 34},
					},
				},
			},
			result,
		)
	})
}

func TestParseTransactionDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("no prepare, execute", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("transaction { execute {} }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Execute: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindExecute,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "execute",
								Pos:        ast.Position{Offset: 14, Line: 1, Column: 14},
							},
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 22, Line: 1, Column: 22},
										EndPos:   ast.Position{Offset: 23, Line: 1, Column: 23},
									},
								},
							},
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
					},
				},
			},
			result,
		)
	})

	t.Run("EmptyTransaction", func(t *testing.T) {

		const code = `
		  transaction {}
		`
		result, errs := testParseProgram(code)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Fields:         nil,
					Prepare:        nil,
					PreConditions:  nil,
					PostConditions: nil,
					Execute:        nil,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 5, Line: 2, Column: 4},
						EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
					},
				},
			},
			result.Declarations(),
		)
	})

	t.Run("SimpleTransaction", func(t *testing.T) {
		const code = `
		  transaction {

		    var x: Int

		    prepare(signer: AuthAccount) {
	          x = 0
			}

		    execute {
	          x = 1 + 1
			}
		  }
		`
		result, errs := testParseProgram(code)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Fields: []*ast.FieldDeclaration{
						{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindVariable,
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 30, Line: 4, Column: 10},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 33, Line: 4, Column: 13},
									},
								},
								StartPos: ast.Position{Offset: 33, Line: 4, Column: 13},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 26, Line: 4, Column: 6},
								EndPos:   ast.Position{Offset: 35, Line: 4, Column: 15},
							},
						},
					},
					Prepare: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindPrepare,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "prepare",
								Pos:        ast.Position{Offset: 44, Line: 6, Column: 6},
							},
							ParameterList: &ast.ParameterList{
								Parameters: []*ast.Parameter{
									{
										Label: "",
										Identifier: ast.Identifier{
											Identifier: "signer",
											Pos:        ast.Position{Offset: 52, Line: 6, Column: 14},
										},
										TypeAnnotation: &ast.TypeAnnotation{
											IsResource: false,
											Type: &ast.NominalType{
												Identifier: ast.Identifier{
													Identifier: "AuthAccount",
													Pos:        ast.Position{Offset: 60, Line: 6, Column: 22},
												},
											},
											StartPos: ast.Position{Offset: 60, Line: 6, Column: 22},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 52, Line: 6, Column: 14},
											EndPos:   ast.Position{Offset: 70, Line: 6, Column: 32},
										},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 51, Line: 6, Column: 13},
									EndPos:   ast.Position{Offset: 71, Line: 6, Column: 33},
								},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 86, Line: 7, Column: 11},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 88, Line: 7, Column: 13},
											},
											Value: &ast.IntegerExpression{
												PositiveLiteral: []byte("0"),
												Value:           new(big.Int),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 90, Line: 7, Column: 15},
													EndPos:   ast.Position{Offset: 90, Line: 7, Column: 15},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 73, Line: 6, Column: 35},
										EndPos:   ast.Position{Offset: 95, Line: 8, Column: 3},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 44, Line: 6, Column: 6},
						},
					},
					PreConditions:  nil,
					PostConditions: nil,
					Execute: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindExecute,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "execute",
								Pos:        ast.Position{Offset: 104, Line: 10, Column: 6},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 125, Line: 11, Column: 11},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 127, Line: 11, Column: 13},
											},
											Value: &ast.BinaryExpression{
												Operation: ast.OperationPlus,
												Left: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 129, Line: 11, Column: 15},
														EndPos:   ast.Position{Offset: 129, Line: 11, Column: 15},
													},
												},
												Right: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 133, Line: 11, Column: 19},
														EndPos:   ast.Position{Offset: 133, Line: 11, Column: 19},
													},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 112, Line: 10, Column: 14},
										EndPos:   ast.Position{Offset: 138, Line: 12, Column: 3},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 104, Line: 10, Column: 6},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 5, Line: 2, Column: 4},
						EndPos:   ast.Position{Offset: 144, Line: 13, Column: 4},
					},
				},
			},
			result.Declarations(),
		)
	})

	t.Run("PreExecutePost", func(t *testing.T) {
		const code = `
		  transaction {

		    var x: Int

		    prepare(signer: AuthAccount) {
	          x = 0
			}

			pre {
	      	  x == 0
			}

		    execute {
	          x = 1 + 1
			}

		    post {
	          x == 2
	        }
		  }
		`
		result, errs := testParseProgram(code)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Fields: []*ast.FieldDeclaration{
						{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindVariable,
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 30, Line: 4, Column: 10},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 33, Line: 4, Column: 13},
									},
								},
								StartPos: ast.Position{Offset: 33, Line: 4, Column: 13},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 26, Line: 4, Column: 6},
								EndPos:   ast.Position{Offset: 35, Line: 4, Column: 15},
							},
						},
					},
					Prepare: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindPrepare,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "prepare",
								Pos:        ast.Position{Offset: 44, Line: 6, Column: 6},
							},
							ParameterList: &ast.ParameterList{
								Parameters: []*ast.Parameter{
									{
										Label: "",
										Identifier: ast.Identifier{
											Identifier: "signer",
											Pos:        ast.Position{Offset: 52, Line: 6, Column: 14},
										},
										TypeAnnotation: &ast.TypeAnnotation{
											IsResource: false,
											Type: &ast.NominalType{
												Identifier: ast.Identifier{
													Identifier: "AuthAccount",
													Pos:        ast.Position{Offset: 60, Line: 6, Column: 22},
												},
											},
											StartPos: ast.Position{Offset: 60, Line: 6, Column: 22},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 52, Line: 6, Column: 14},
											EndPos:   ast.Position{Offset: 70, Line: 6, Column: 32},
										},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 51, Line: 6, Column: 13},
									EndPos:   ast.Position{Offset: 71, Line: 6, Column: 33},
								},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 86, Line: 7, Column: 11},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 88, Line: 7, Column: 13},
											},
											Value: &ast.IntegerExpression{
												PositiveLiteral: []byte("0"),
												Value:           new(big.Int),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 90, Line: 7, Column: 15},
													EndPos:   ast.Position{Offset: 90, Line: 7, Column: 15},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 73, Line: 6, Column: 35},
										EndPos:   ast.Position{Offset: 95, Line: 8, Column: 3},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 44, Line: 6, Column: 6},
						},
					},
					PreConditions: &ast.Conditions{
						{
							Kind: ast.ConditionKindPre,
							Test: &ast.BinaryExpression{
								Operation: ast.OperationEqual,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "x",
										Pos:        ast.Position{Offset: 117, Line: 11, Column: 10},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 122, Line: 11, Column: 15},
										EndPos:   ast.Position{Offset: 122, Line: 11, Column: 15},
									},
								},
							},
						},
					},
					PostConditions: &ast.Conditions{
						{
							Kind: ast.ConditionKindPost,
							Test: &ast.BinaryExpression{
								Operation: ast.OperationEqual,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "x",
										Pos:        ast.Position{Offset: 197, Line: 19, Column: 11},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("2"),
									Value:           big.NewInt(2),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 202, Line: 19, Column: 16},
										EndPos:   ast.Position{Offset: 202, Line: 19, Column: 16},
									},
								},
							},
						},
					},
					Execute: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindExecute,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "execute",
								Pos:        ast.Position{Offset: 136, Line: 14, Column: 6},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 157, Line: 15, Column: 11},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 159, Line: 15, Column: 13},
											},
											Value: &ast.BinaryExpression{
												Operation: ast.OperationPlus,
												Left: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 161, Line: 15, Column: 15},
														EndPos:   ast.Position{Offset: 161, Line: 15, Column: 15},
													},
												},
												Right: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 165, Line: 15, Column: 19},
														EndPos:   ast.Position{Offset: 165, Line: 15, Column: 19},
													},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 144, Line: 14, Column: 14},
										EndPos:   ast.Position{Offset: 170, Line: 16, Column: 3},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 136, Line: 14, Column: 6},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 5, Line: 2, Column: 4},
						EndPos:   ast.Position{Offset: 219, Line: 21, Column: 4},
					},
				},
			},
			result.Declarations(),
		)
	})

	t.Run("PrePostExecute", func(t *testing.T) {
		const code = `
		  transaction {

		    var x: Int

		    prepare(signer: AuthAccount) {
	          x = 0
			}

			pre {
	      	  x == 0
			}

		    post {
	          x == 2
	        }

		    execute {
	          x = 1 + 1
			}
		  }
		`
		result, errs := testParseProgram(code)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Fields: []*ast.FieldDeclaration{
						{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindVariable,
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 30, Line: 4, Column: 10},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 33, Line: 4, Column: 13},
									},
								},
								StartPos: ast.Position{Offset: 33, Line: 4, Column: 13},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 26, Line: 4, Column: 6},
								EndPos:   ast.Position{Offset: 35, Line: 4, Column: 15},
							},
						},
					},
					Prepare: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindPrepare,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "prepare",
								Pos:        ast.Position{Offset: 44, Line: 6, Column: 6},
							},
							ParameterList: &ast.ParameterList{
								Parameters: []*ast.Parameter{
									{
										Label: "",
										Identifier: ast.Identifier{
											Identifier: "signer",
											Pos:        ast.Position{Offset: 52, Line: 6, Column: 14},
										},
										TypeAnnotation: &ast.TypeAnnotation{
											IsResource: false,
											Type: &ast.NominalType{
												Identifier: ast.Identifier{
													Identifier: "AuthAccount",
													Pos:        ast.Position{Offset: 60, Line: 6, Column: 22},
												},
											},
											StartPos: ast.Position{Offset: 60, Line: 6, Column: 22},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 52, Line: 6, Column: 14},
											EndPos:   ast.Position{Offset: 70, Line: 6, Column: 32},
										},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 51, Line: 6, Column: 13},
									EndPos:   ast.Position{Offset: 71, Line: 6, Column: 33},
								},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 86, Line: 7, Column: 11},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 88, Line: 7, Column: 13},
											},
											Value: &ast.IntegerExpression{
												PositiveLiteral: []byte("0"),
												Value:           new(big.Int),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 90, Line: 7, Column: 15},
													EndPos:   ast.Position{Offset: 90, Line: 7, Column: 15},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 73, Line: 6, Column: 35},
										EndPos:   ast.Position{Offset: 95, Line: 8, Column: 3},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 44, Line: 6, Column: 6},
						},
					},
					PreConditions: &ast.Conditions{
						{
							Kind: ast.ConditionKindPre,
							Test: &ast.BinaryExpression{
								Operation: ast.OperationEqual,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "x",
										Pos:        ast.Position{Offset: 117, Line: 11, Column: 10},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 122, Line: 11, Column: 15},
										EndPos:   ast.Position{Offset: 122, Line: 11, Column: 15},
									},
								},
							},
						},
					},
					PostConditions: &ast.Conditions{
						{
							Kind: ast.ConditionKindPost,
							Test: &ast.BinaryExpression{
								Operation: ast.OperationEqual,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "x",
										Pos:        ast.Position{Offset: 154, Line: 15, Column: 11},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("2"),
									Value:           big.NewInt(2),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 159, Line: 15, Column: 16},
										EndPos:   ast.Position{Offset: 159, Line: 15, Column: 16},
									},
								},
							},
						},
					},
					Execute: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindExecute,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "execute",
								Pos:        ast.Position{Offset: 179, Line: 18, Column: 6},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 200, Line: 19, Column: 11},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 202, Line: 19, Column: 13},
											},
											Value: &ast.BinaryExpression{
												Operation: ast.OperationPlus,
												Left: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 204, Line: 19, Column: 15},
														EndPos:   ast.Position{Offset: 204, Line: 19, Column: 15},
													},
												},
												Right: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 208, Line: 19, Column: 19},
														EndPos:   ast.Position{Offset: 208, Line: 19, Column: 19},
													},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 187, Line: 18, Column: 14},
										EndPos:   ast.Position{Offset: 213, Line: 20, Column: 3},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 179, Line: 18, Column: 6},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 5, Line: 2, Column: 4},
						EndPos:   ast.Position{Offset: 219, Line: 21, Column: 4},
					},
				},
			},
			result.Declarations(),
		)
	})
}

func TestParseFunctionAndBlock(t *testing.T) {

	t.Parallel()

	result, errs := testParseDeclarations(`
	    fun test() { return }
	`)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				ParameterList: &ast.ParameterList{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 15, Line: 2, Column: 14},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Pos: ast.Position{Offset: 15, Line: 2, Column: 14},
						},
					},
					StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.ReturnStatement{
								Range: ast.Range{
									StartPos: ast.Position{Offset: 19, Line: 2, Column: 18},
									EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 26, Line: 2, Column: 25},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result,
	)
}

func TestParseFunctionParameterWithoutLabel(t *testing.T) {

	t.Parallel()

	result, errs := testParseDeclarations(`
	    fun test(x: Int) { }
	`)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "",
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 15, Line: 2, Column: 14},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
									},
								},
								StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
								EndPos:   ast.Position{Offset: 20, Line: 2, Column: 19},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Pos: ast.Position{Offset: 21, Line: 2, Column: 20},
						},
					},
					StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
							EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result,
	)
}

func TestParseFunctionParameterWithLabel(t *testing.T) {

	t.Parallel()

	result, errs := testParseDeclarations(`
	    fun test(x y: Int) { }
	`)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "x",
							Identifier: ast.Identifier{
								Identifier: "y",
								Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 20, Line: 2, Column: 19},
									},
								},
								StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
								EndPos:   ast.Position{Offset: 22, Line: 2, Column: 21},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Pos: ast.Position{Offset: 23, Line: 2, Column: 22},
						},
					},
					StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
							EndPos:   ast.Position{Offset: 27, Line: 2, Column: 26},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result,
	)
}

func TestParseStructure(t *testing.T) {

	t.Parallel()

	const code = `
        struct Test {
            pub(set) var foo: Int

            init(foo: Int) {
                self.foo = foo
            }

            pub fun getFoo(): Int {
                return self.foo
            }
        }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "Test",
					Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							Access:       ast.AccessPublicSettable,
							VariableKind: ast.VariableKindVariable,
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 48, Line: 3, Column: 25},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 53, Line: 3, Column: 30},
									},
								},
								StartPos: ast.Position{Offset: 53, Line: 3, Column: 30},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 35, Line: 3, Column: 12},
								EndPos:   ast.Position{Offset: 55, Line: 3, Column: 32},
							},
						},
						&ast.SpecialFunctionDeclaration{
							Kind: common.DeclarationKindInitializer,
							FunctionDeclaration: &ast.FunctionDeclaration{
								Identifier: ast.Identifier{
									Identifier: "init",
									Pos:        ast.Position{Offset: 70, Line: 5, Column: 12},
								},
								ParameterList: &ast.ParameterList{
									Parameters: []*ast.Parameter{
										{
											Label: "",
											Identifier: ast.Identifier{
												Identifier: "foo",
												Pos:        ast.Position{Offset: 75, Line: 5, Column: 17},
											},
											TypeAnnotation: &ast.TypeAnnotation{
												IsResource: false,
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Int",
														Pos:        ast.Position{Offset: 80, Line: 5, Column: 22},
													},
												},
												StartPos: ast.Position{Offset: 80, Line: 5, Column: 22},
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 75, Line: 5, Column: 17},
												EndPos:   ast.Position{Offset: 82, Line: 5, Column: 24},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 74, Line: 5, Column: 16},
										EndPos:   ast.Position{Offset: 83, Line: 5, Column: 25},
									},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Statements: []ast.Statement{
											&ast.AssignmentStatement{
												Target: &ast.MemberExpression{
													Expression: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "self",
															Pos:        ast.Position{Offset: 103, Line: 6, Column: 16},
														},
													},
													AccessPos: ast.Position{Offset: 107, Line: 6, Column: 20},
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos:        ast.Position{Offset: 108, Line: 6, Column: 21},
													},
												},
												Transfer: &ast.Transfer{
													Operation: ast.TransferOperationCopy,
													Pos:       ast.Position{Offset: 112, Line: 6, Column: 25},
												},
												Value: &ast.IdentifierExpression{
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos:        ast.Position{Offset: 114, Line: 6, Column: 27},
													},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 85, Line: 5, Column: 27},
											EndPos:   ast.Position{Offset: 130, Line: 7, Column: 12},
										},
									},
								},
								StartPos: ast.Position{Offset: 70, Line: 5, Column: 12},
							},
						},
						&ast.FunctionDeclaration{
							Access: ast.AccessPublic,
							Identifier: ast.Identifier{
								Identifier: "getFoo",
								Pos:        ast.Position{Offset: 153, Line: 9, Column: 20},
							},
							ParameterList: &ast.ParameterList{
								Range: ast.Range{
									StartPos: ast.Position{Offset: 159, Line: 9, Column: 26},
									EndPos:   ast.Position{Offset: 160, Line: 9, Column: 27},
								},
							},
							ReturnTypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 163, Line: 9, Column: 30},
									},
								},
								StartPos: ast.Position{Offset: 163, Line: 9, Column: 30},
							},
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.ReturnStatement{
											Expression: &ast.MemberExpression{
												Expression: &ast.IdentifierExpression{
													Identifier: ast.Identifier{
														Identifier: "self",
														Pos:        ast.Position{Offset: 192, Line: 10, Column: 23},
													},
												},
												AccessPos: ast.Position{Offset: 196, Line: 10, Column: 27},
												Identifier: ast.Identifier{
													Identifier: "foo",
													Pos:        ast.Position{Offset: 197, Line: 10, Column: 28},
												},
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 185, Line: 10, Column: 16},
												EndPos:   ast.Position{Offset: 199, Line: 10, Column: 30},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 167, Line: 9, Column: 34},
										EndPos:   ast.Position{Offset: 213, Line: 11, Column: 12},
									},
								},
							},
							StartPos: ast.Position{Offset: 145, Line: 9, Column: 12},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 223, Line: 12, Column: 8},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseStructureWithConformances(t *testing.T) {

	t.Parallel()

	const code = `
        struct Test: Foo, Bar {}
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "Test",
					Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Conformances: []*ast.NominalType{
					{
						Identifier: ast.Identifier{
							Identifier: "Foo",
							Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
						},
					},
					{
						Identifier: ast.Identifier{
							Identifier: "Bar",
							Pos:        ast.Position{Offset: 27, Line: 2, Column: 26},
						},
					},
				},
				Members: &ast.Members{},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 32, Line: 2, Column: 31},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParsePreAndPostConditions(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(n: Int) {
            pre {
                n != 0
                n > 0
            }
            post {
                result == 0
            }
            return 0
        }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "",
							Identifier: ast.Identifier{
								Identifier: "n",
								Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 21, Line: 2, Column: 20},
									},
								},
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
								EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "",
							Pos:        ast.Position{Offset: 24, Line: 2, Column: 23},
						},
					},
					StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.ReturnStatement{
								Expression: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 185, Line: 10, Column: 19},
										EndPos:   ast.Position{Offset: 185, Line: 10, Column: 19},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 178, Line: 10, Column: 12},
									EndPos:   ast.Position{Offset: 185, Line: 10, Column: 19},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:   ast.Position{Offset: 195, Line: 11, Column: 8},
						},
					},
					PreConditions: &ast.Conditions{
						{
							Kind: ast.ConditionKindPre,
							Test: &ast.BinaryExpression{
								Operation: ast.OperationNotEqual,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "n",
										Pos:        ast.Position{Offset: 62, Line: 4, Column: 16},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 67, Line: 4, Column: 21},
										EndPos:   ast.Position{Offset: 67, Line: 4, Column: 21},
									},
								},
							},
						},
						{
							Kind: ast.ConditionKindPre,
							Test: &ast.BinaryExpression{
								Operation: ast.OperationGreater,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "n",
										Pos:        ast.Position{Offset: 85, Line: 5, Column: 16},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 89, Line: 5, Column: 20},
										EndPos:   ast.Position{Offset: 89, Line: 5, Column: 20},
									},
								},
							},
						},
					},
					PostConditions: &ast.Conditions{
						{
							Kind: ast.ConditionKindPost,
							Test: &ast.BinaryExpression{
								Operation: ast.OperationEqual,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "result",
										Pos:        ast.Position{Offset: 140, Line: 8, Column: 16},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 150, Line: 8, Column: 26},
										EndPos:   ast.Position{Offset: 150, Line: 8, Column: 26},
									},
								},
							},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseConditionMessage(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(n: Int) {
            pre {
                n >= 0: "n must be positive"
            }
            return n
        }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "",
							Identifier: ast.Identifier{Identifier: "n",
								Pos: ast.Position{Offset: 18, Line: 2, Column: 17},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 21, Line: 2, Column: 20},
									},
								},
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
								EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "",
							Pos:        ast.Position{Offset: 24, Line: 2, Column: 23},
						},
					},
					StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.ReturnStatement{
								Expression: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "n",
										Pos:        ast.Position{Offset: 124, Line: 6, Column: 19},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 117, Line: 6, Column: 12},
									EndPos:   ast.Position{Offset: 124, Line: 6, Column: 19},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:   ast.Position{Offset: 134, Line: 7, Column: 8},
						},
					},
					PreConditions: &ast.Conditions{
						{
							Kind: ast.ConditionKindPre,
							Test: &ast.BinaryExpression{
								Operation: ast.OperationGreaterEqual,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "n",
										Pos:        ast.Position{Offset: 62, Line: 4, Column: 16},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 67, Line: 4, Column: 21},
										EndPos:   ast.Position{Offset: 67, Line: 4, Column: 21},
									},
								},
							},
							Message: &ast.StringExpression{
								Value: "n must be positive",
								Range: ast.Range{
									StartPos: ast.Position{Offset: 70, Line: 4, Column: 24},
									EndPos:   ast.Position{Offset: 89, Line: 4, Column: 43},
								},
							},
						},
					},
					PostConditions: nil,
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseInterface(t *testing.T) {

	t.Parallel()

	for _, kind := range common.CompositeKindsWithFieldsAndFunctions {
		code := fmt.Sprintf(`
            %s interface Test {
                foo: Int

                init(foo: Int)

                fun getFoo(): Int
            }
	    `, kind.Keyword())
		actual, err := testParseProgram(code)

		require.NoError(t, err)

		// only compare AST for one kind: structs

		if kind != common.CompositeKindStructure {
			continue
		}

		test := &ast.InterfaceDeclaration{
			CompositeKind: common.CompositeKindStructure,
			Identifier: ast.Identifier{
				Identifier: "Test",
				Pos:        ast.Position{Offset: 30, Line: 2, Column: 29},
			},
			Members: ast.NewUnmeteredMembers(
				[]ast.Declaration{
					&ast.FieldDeclaration{
						Access:       ast.AccessNotSpecified,
						VariableKind: ast.VariableKindNotSpecified,
						Identifier: ast.Identifier{
							Identifier: "foo",
							Pos:        ast.Position{Offset: 53, Line: 3, Column: 16},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Offset: 58, Line: 3, Column: 21},
								},
							},
							StartPos: ast.Position{Offset: 58, Line: 3, Column: 21},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 53, Line: 3, Column: 16},
							EndPos:   ast.Position{Offset: 60, Line: 3, Column: 23},
						},
					},
					&ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindInitializer,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Identifier: ast.Identifier{
								Identifier: "init",
								Pos:        ast.Position{Offset: 79, Line: 5, Column: 16},
							},
							ParameterList: &ast.ParameterList{
								Parameters: []*ast.Parameter{
									{
										Label: "",
										Identifier: ast.Identifier{
											Identifier: "foo",
											Pos:        ast.Position{Offset: 84, Line: 5, Column: 21},
										},
										TypeAnnotation: &ast.TypeAnnotation{
											IsResource: false,
											Type: &ast.NominalType{
												Identifier: ast.Identifier{
													Identifier: "Int",
													Pos:        ast.Position{Offset: 89, Line: 5, Column: 26},
												},
											},
											StartPos: ast.Position{Offset: 89, Line: 5, Column: 26},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 84, Line: 5, Column: 21},
											EndPos:   ast.Position{Offset: 91, Line: 5, Column: 28},
										},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 83, Line: 5, Column: 20},
									EndPos:   ast.Position{Offset: 92, Line: 5, Column: 29},
								},
							},
							FunctionBlock: nil,
							StartPos:      ast.Position{Offset: 79, Line: 5, Column: 16},
						},
					},
					&ast.FunctionDeclaration{
						Access: ast.AccessNotSpecified,
						Identifier: ast.Identifier{
							Identifier: "getFoo",
							Pos:        ast.Position{Offset: 115, Line: 7, Column: 20},
						},
						ParameterList: &ast.ParameterList{
							Range: ast.Range{
								StartPos: ast.Position{Offset: 121, Line: 7, Column: 26},
								EndPos:   ast.Position{Offset: 122, Line: 7, Column: 27},
							},
						},
						ReturnTypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Offset: 125, Line: 7, Column: 30},
								},
							},
							StartPos: ast.Position{Offset: 125, Line: 7, Column: 30},
						},
						FunctionBlock: nil,
						StartPos:      ast.Position{Offset: 111, Line: 7, Column: 16},
					},
				},
			),
			Range: ast.Range{
				StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
				EndPos:   ast.Position{Offset: 141, Line: 8, Column: 12},
			},
		}

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{test},
			actual.Declarations(),
		)
	}
}

func TestParsePragmaNoArguments(t *testing.T) {

	t.Parallel()

	const code = `#pedantic`
	result, err := testParseProgram(code)
	require.NoError(t, err)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.PragmaDeclaration{
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "pedantic",
						Pos:        ast.Position{Offset: 1, Line: 1, Column: 1},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParsePragmaArguments(t *testing.T) {

	t.Parallel()

	const code = `#version("1.0")`
	actual, err := testParseProgram(code)
	require.NoError(t, err)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.PragmaDeclaration{
				Expression: &ast.InvocationExpression{
					InvokedExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "version",
							Pos:        ast.Position{Offset: 1, Line: 1, Column: 1},
						},
					},
					Arguments: ast.Arguments{
						{
							Expression: &ast.StringExpression{
								Value: "1.0",
								Range: ast.Range{
									StartPos: ast.Position{Offset: 9, Line: 1, Column: 9},
									EndPos:   ast.Position{Offset: 13, Line: 1, Column: 13},
								},
							},
							TrailingSeparatorPos: ast.Position{Offset: 14, Line: 1, Column: 14},
						},
					},
					ArgumentsStartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
					EndPos:            ast.Position{Offset: 14, Line: 1, Column: 14},
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
				},
			},
		},
		actual.Declarations(),
	)
}

func TestParseImportWithString(t *testing.T) {

	t.Parallel()

	const code = `
        import "test.cdc"
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.ImportDeclaration{
				Identifiers: nil,
				Location:    common.StringLocation("test.cdc"),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
				},
				LocationPos: ast.Position{Offset: 16, Line: 2, Column: 15},
			},
		},
		result.Declarations(),
	)
}

func TestParseImportWithAddress(t *testing.T) {

	t.Parallel()

	const code = `
        import 0x1234
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.ImportDeclaration{
				Identifiers: nil,
				Location: common.AddressLocation{
					Address: common.MustBytesToAddress([]byte{0x12, 0x34}),
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
				},
				LocationPos: ast.Position{Offset: 16, Line: 2, Column: 15},
			},
		},
		result.Declarations(),
	)
}

func TestParseImportWithIdentifiers(t *testing.T) {

	t.Parallel()

	const code = `
        import A, b from 0x1
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.ImportDeclaration{
				Identifiers: []ast.Identifier{
					{
						Identifier: "A",
						Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
					},
					{
						Identifier: "b",
						Pos:        ast.Position{Offset: 19, Line: 2, Column: 18},
					},
				},
				Location: common.AddressLocation{
					Address: common.MustBytesToAddress([]byte{0x1}),
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 28, Line: 2, Column: 27},
				},
				LocationPos: ast.Position{Offset: 26, Line: 2, Column: 25},
			},
		},
		result.Declarations(),
	)
}

func TestParseFieldWithFromIdentifier(t *testing.T) {

	t.Parallel()

	const code = `
      struct S {
          let from: String
      }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				Access:        ast.AccessNotSpecified,
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "S",
					Pos:        ast.Position{Offset: 14, Line: 2, Column: 13},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindConstant,
							Identifier: ast.Identifier{
								Identifier: "from",
								Pos:        ast.Position{Offset: 32, Line: 3, Column: 14},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "String",
										Pos:        ast.Position{Offset: 38, Line: 3, Column: 20},
									},
								},
								StartPos: ast.Position{Offset: 38, Line: 3, Column: 20},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 28, Line: 3, Column: 10},
								EndPos:   ast.Position{Offset: 43, Line: 3, Column: 25},
							},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
					EndPos:   ast.Position{Offset: 51, Line: 4, Column: 6},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseFunctionWithFromIdentifier(t *testing.T) {

	t.Parallel()

	const code = `
        fun send(from: String, to: String) {}
	`
	_, errs := testParseProgram(code)
	require.Empty(t, errs)
}

func TestParseImportWithFromIdentifier(t *testing.T) {

	t.Parallel()

	const code = `
        import from from 0x1
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.ImportDeclaration{
				Identifiers: []ast.Identifier{
					{
						Identifier: "from",
						Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
					},
				},
				Location: common.AddressLocation{
					Address: common.MustBytesToAddress([]byte{0x1}),
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 28, Line: 2, Column: 27},
				},
				LocationPos: ast.Position{Offset: 26, Line: 2, Column: 25},
			},
		},
		result.Declarations(),
	)
}

func TestParseSemicolonsBetweenDeclarations(t *testing.T) {

	t.Parallel()

	const code = `
        import from from 0x0;
        fun foo() {};
	`
	_, errs := testParseProgram(code)
	require.Empty(t, errs)
}

func TestParseResource(t *testing.T) {

	t.Parallel()

	const code = `
        resource Test {}
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				CompositeKind: common.CompositeKindResource,
				Identifier: ast.Identifier{
					Identifier: "Test",
					Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
				},
				Members: &ast.Members{},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseEventDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
        event Transfer(to: Address, from: Address)
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				CompositeKind: common.CompositeKindEvent,
				Identifier: ast.Identifier{
					Identifier: "Transfer",
					Pos:        ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.SpecialFunctionDeclaration{
							Kind: common.DeclarationKindInitializer,
							FunctionDeclaration: &ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Parameters: []*ast.Parameter{
										{
											Label: "",
											Identifier: ast.Identifier{
												Identifier: "to",
												Pos:        ast.Position{Offset: 24, Line: 2, Column: 23},
											},
											TypeAnnotation: &ast.TypeAnnotation{
												IsResource: false,
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Address",
														Pos:        ast.Position{Offset: 28, Line: 2, Column: 27},
													},
												},
												StartPos: ast.Position{Offset: 28, Line: 2, Column: 27},
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
												EndPos:   ast.Position{Offset: 34, Line: 2, Column: 33},
											},
										},
										{
											Label: "",
											Identifier: ast.Identifier{
												Identifier: "from",
												Pos:        ast.Position{Offset: 37, Line: 2, Column: 36},
											},
											TypeAnnotation: &ast.TypeAnnotation{
												IsResource: false,
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Address",
														Pos:        ast.Position{Offset: 43, Line: 2, Column: 42},
													},
												},
												StartPos: ast.Position{Offset: 43, Line: 2, Column: 42},
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 37, Line: 2, Column: 36},
												EndPos:   ast.Position{Offset: 49, Line: 2, Column: 48},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
										EndPos:   ast.Position{Offset: 50, Line: 2, Column: 49},
									},
								},
								StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
							},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 50, Line: 2, Column: 49},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseEventEmitStatement(t *testing.T) {

	t.Parallel()

	const code = `
      fun test() {
        emit Transfer(to: 1, from: 2)
      }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 11, Line: 2, Column: 10},
				},
				ParameterList: &ast.ParameterList{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
						EndPos:   ast.Position{Offset: 16, Line: 2, Column: 15},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "",
							Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
						},
					},
					StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.EmitStatement{
								InvocationExpression: &ast.InvocationExpression{
									InvokedExpression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "Transfer",
											Pos:        ast.Position{Offset: 33, Line: 3, Column: 13},
										},
									},
									Arguments: ast.Arguments{
										{
											Label:         "to",
											LabelStartPos: &ast.Position{Offset: 42, Line: 3, Column: 22},
											LabelEndPos:   &ast.Position{Offset: 43, Line: 3, Column: 23},
											Expression: &ast.IntegerExpression{
												PositiveLiteral: []byte("1"),
												Value:           big.NewInt(1),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 46, Line: 3, Column: 26},
													EndPos:   ast.Position{Offset: 46, Line: 3, Column: 26},
												},
											},
											TrailingSeparatorPos: ast.Position{Offset: 47, Line: 3, Column: 27},
										},
										{
											Label:         "from",
											LabelStartPos: &ast.Position{Offset: 49, Line: 3, Column: 29},
											LabelEndPos:   &ast.Position{Offset: 52, Line: 3, Column: 32},
											Expression: &ast.IntegerExpression{
												PositiveLiteral: []byte("2"),
												Value:           big.NewInt(2),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 55, Line: 3, Column: 35},
													EndPos:   ast.Position{Offset: 55, Line: 3, Column: 35},
												},
											},
											TrailingSeparatorPos: ast.Position{Offset: 56, Line: 3, Column: 36},
										},
									},
									ArgumentsStartPos: ast.Position{Offset: 41, Line: 3, Column: 21},
									EndPos:            ast.Position{Offset: 56, Line: 3, Column: 36},
								},
								StartPos: ast.Position{Offset: 28, Line: 3, Column: 8},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
							EndPos:   ast.Position{Offset: 64, Line: 4, Column: 6},
						},
					},
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

func TestParseResourceReturnType(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(): @X {}
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				ParameterList: &ast.ParameterList{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: true,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "X",
							Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
						},
					},
					StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
							EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseMovingVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
        let x <- y
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "y",
						Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
					},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationMove,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseResourceParameterType(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(x: @X) {}
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "",
							Pos:        ast.Position{Offset: 23, Line: 2, Column: 22},
						},
					},
					StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "",
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: true,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "X",
										Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
									},
								},
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
								EndPos:   ast.Position{Offset: 22, Line: 2, Column: 21},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
							EndPos:   ast.Position{Offset: 26, Line: 2, Column: 25},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseMovingVariableDeclarationWithTypeAnnotation(t *testing.T) {

	t.Parallel()

	const code = `
        let x: @R <- y
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: true,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "R",
							Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
						},
					},
					StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "y",
						Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
					},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationMove,
					Pos:       ast.Position{Offset: 19, Line: 2, Column: 18},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseFieldDeclarationWithMoveTypeAnnotation(t *testing.T) {

	t.Parallel()

	const code = `
        struct X { x: @R }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "X",
					Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 20, Line: 2, Column: 19},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: true,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "R",
										Pos:        ast.Position{Offset: 24, Line: 2, Column: 23},
									},
								},
								StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
								EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
							},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 26, Line: 2, Column: 25},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseDestructor(t *testing.T) {

	t.Parallel()

	const code = `
        resource Test {
            destroy() {}
        }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				CompositeKind: common.CompositeKindResource,
				Identifier: ast.Identifier{
					Identifier: "Test",
					Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.SpecialFunctionDeclaration{
							Kind: common.DeclarationKindDestructor,
							FunctionDeclaration: &ast.FunctionDeclaration{
								Identifier: ast.Identifier{
									Identifier: "destroy",
									Pos:        ast.Position{Offset: 37, Line: 3, Column: 12},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 44, Line: 3, Column: 19},
										EndPos:   ast.Position{Offset: 45, Line: 3, Column: 20},
									},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 47, Line: 3, Column: 22},
											EndPos:   ast.Position{Offset: 48, Line: 3, Column: 23},
										},
									},
								},
								StartPos: ast.Position{Offset: 37, Line: 3, Column: 12},
							},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 58, Line: 4, Column: 8},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseCompositeDeclarationWithSemicolonSeparatedMembers(t *testing.T) {

	t.Parallel()

	const code = `
        struct Kitty { let id: Int ; init(id: Int) { self.id = id } }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "Kitty",
					Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							VariableKind: ast.VariableKindConstant,
							Identifier: ast.Identifier{
								Identifier: "id",
								Pos:        ast.Position{Offset: 28, Line: 2, Column: 27},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 32, Line: 2, Column: 31},
									},
								},
								StartPos: ast.Position{Offset: 32, Line: 2, Column: 31},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
								EndPos:   ast.Position{Offset: 34, Line: 2, Column: 33},
							},
						},
						&ast.SpecialFunctionDeclaration{
							Kind: common.DeclarationKindInitializer,
							FunctionDeclaration: &ast.FunctionDeclaration{
								Identifier: ast.Identifier{
									Identifier: "init",
									Pos:        ast.Position{Offset: 38, Line: 2, Column: 37},
								},
								ParameterList: &ast.ParameterList{
									Parameters: []*ast.Parameter{
										{
											Identifier: ast.Identifier{
												Identifier: "id",
												Pos:        ast.Position{Offset: 43, Line: 2, Column: 42},
											},
											TypeAnnotation: &ast.TypeAnnotation{
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Int",
														Pos:        ast.Position{Offset: 47, Line: 2, Column: 46},
													},
												},
												StartPos: ast.Position{Offset: 47, Line: 2, Column: 46},
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 43, Line: 2, Column: 42},
												EndPos:   ast.Position{Offset: 49, Line: 2, Column: 48},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 42, Line: 2, Column: 41},
										EndPos:   ast.Position{Offset: 50, Line: 2, Column: 49},
									},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Statements: []ast.Statement{
											&ast.AssignmentStatement{
												Target: &ast.MemberExpression{
													Expression: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "self",
															Pos:        ast.Position{Offset: 54, Line: 2, Column: 53},
														},
													},
													AccessPos: ast.Position{Offset: 58, Line: 2, Column: 57},
													Identifier: ast.Identifier{
														Identifier: "id",
														Pos:        ast.Position{Offset: 59, Line: 2, Column: 58},
													},
												},
												Transfer: &ast.Transfer{
													Operation: ast.TransferOperationCopy,
													Pos:       ast.Position{Offset: 62, Line: 2, Column: 61},
												},
												Value: &ast.IdentifierExpression{
													Identifier: ast.Identifier{
														Identifier: "id",
														Pos:        ast.Position{Offset: 64, Line: 2, Column: 63},
													},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 52, Line: 2, Column: 51},
											EndPos:   ast.Position{Offset: 67, Line: 2, Column: 66},
										},
									},
								},
								StartPos: ast.Position{Offset: 38, Line: 2, Column: 37},
							},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 69, Line: 2, Column: 68},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseAccessModifiers(t *testing.T) {

	t.Parallel()

	type declaration struct {
		name, code string
	}

	declarations := []declaration{
		{"variable", "%s var test = 1"},
		{"constant", "%s let test = 1"},
		{"function", "%s fun test() {}"},
	}

	for _, compositeKind := range common.AllCompositeKinds {

		for _, isInterface := range []bool{true, false} {

			if !compositeKind.SupportsInterfaces() && isInterface {
				continue
			}

			interfaceKeyword := ""
			if isInterface {
				interfaceKeyword = "interface"
			}

			formatName := func(name string) string {
				return fmt.Sprintf(
					"%s %s %s",
					compositeKind.Keyword(),
					interfaceKeyword,
					name,
				)
			}

			formatCode := func(format string) string {
				return fmt.Sprintf(format, compositeKind.Keyword(), interfaceKeyword)
			}

			if compositeKind == common.CompositeKindEvent {
				declarations = append(declarations,
					declaration{
						formatName("itself"),
						formatCode("%%s %s %s Test()"),
					},
				)
			} else {
				declarations = append(declarations,
					declaration{
						formatName("itself"),
						formatCode("%%s %s %s Test {}"),
					},
					declaration{
						formatName("field"),
						formatCode("%s %s Test { %%s let test: Int ; init() { self.test = 1 } }"),
					},
					declaration{
						formatName("function"),
						formatCode("%s %s Test { %%s fun test() {} }"),
					},
				)
			}
		}
	}

	for _, declaration := range declarations {
		for _, access := range ast.BasicAccesses {
			testName := fmt.Sprintf("%s/%s", declaration.name, access)
			t.Run(testName, func(t *testing.T) {
				program := fmt.Sprintf(declaration.code, access.Keyword())
				_, errs := testParseProgram(program)

				require.Empty(t, errs)
			})
		}
	}
}

func TestParsePreconditionWithUnaryNegation(t *testing.T) {

	t.Parallel()

	const code = `
	  fun test() {
          pre {
              true: "one"
              !false: "two"
          }
      }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 8, Line: 2, Column: 7},
				},
				ParameterList: &ast.ParameterList{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 12, Line: 2, Column: 11},
						EndPos:   ast.Position{Offset: 13, Line: 2, Column: 12},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Pos: ast.Position{Offset: 13, Line: 2, Column: 12},
						},
					},
					StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
							EndPos:   ast.Position{Offset: 105, Line: 7, Column: 6},
						},
					},
					PreConditions: &ast.Conditions{
						{
							Kind: ast.ConditionKindPre,
							Test: &ast.BoolExpression{
								Value: true,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 47, Line: 4, Column: 14},
									EndPos:   ast.Position{Offset: 50, Line: 4, Column: 17},
								},
							},
							Message: &ast.StringExpression{
								Value: "one",
								Range: ast.Range{
									StartPos: ast.Position{Offset: 53, Line: 4, Column: 20},
									EndPos:   ast.Position{Offset: 57, Line: 4, Column: 24},
								},
							},
						},
						{
							Kind: ast.ConditionKindPre,
							Test: &ast.UnaryExpression{
								Operation: ast.OperationNegate,
								Expression: &ast.BoolExpression{
									Value: false,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 74, Line: 5, Column: 15},
										EndPos:   ast.Position{Offset: 78, Line: 5, Column: 19},
									},
								},
								StartPos: ast.Position{Offset: 73, Line: 5, Column: 14},
							},
							Message: &ast.StringExpression{
								Value: "two",
								Range: ast.Range{
									StartPos: ast.Position{Offset: 81, Line: 5, Column: 22},
									EndPos:   ast.Position{Offset: 85, Line: 5, Column: 26},
								},
							},
						},
					},
				},
				StartPos: ast.Position{Offset: 4, Line: 2, Column: 3},
			},
		},
		result.Declarations(),
	)
}

func TestParseInvalidAccessModifiers(t *testing.T) {

	t.Parallel()

	t.Run("pragma", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("pub #test")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid access modifier for pragma",
					Pos:     ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)
	})

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("pub transaction {}")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid access modifier for transaction",
					Pos:     ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)
	})

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("pub priv let x = 1")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid second access modifier",
					Pos:     ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)
	})
}
