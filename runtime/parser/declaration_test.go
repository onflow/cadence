/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
					Access:     ast.AccessNotSpecified,
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

	t.Run("var, no type annotation, copy, one value, access(all)", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) var x = 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					Value: &ast.IntegerExpression{
						Value: big.NewInt(1),
						PositiveLiteral: []uint8{
							0x31,
						},
						Range: ast.Range{
							StartPos: ast.Position{
								Offset: 21,
								Line:   1,
								Column: 21,
							},
							EndPos: ast.Position{
								Offset: 21,
								Line:   1,
								Column: 21,
							},
						},
						Base: 10,
					},
					Transfer: &ast.Transfer{
						Operation: 0x1,
						Pos: ast.Position{
							Offset: 19,
							Line:   1,
							Column: 19,
						},
					},
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos: ast.Position{
							Offset: 17,
							Line:   1,
							Column: 17,
						},
					},
					StartPos: ast.Position{
						Offset: 1,
						Line:   1,
						Column: 1,
					},
					Access: ast.AccessAll,
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
					Access:     ast.AccessNotSpecified,
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
					Access:     ast.AccessNotSpecified,
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
					Access:     ast.AccessNotSpecified,
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
					Access:     ast.AccessNotSpecified,
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

	t.Run("with purity", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("view var x = 1")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid view modifier for variable",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte("static var x = 1"),
			Config{
				StaticModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier for variable",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("static var x = 1")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte("native var x = 1"),
			Config{
				NativeModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid native modifier for variable",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("native var x = 1")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})
}

func TestParseParameterList(t *testing.T) {

	t.Parallel()

	parse := func(input string) (*ast.ParameterList, []error) {
		return Parse(
			nil,
			[]byte(input),
			func(p *parser) (*ast.ParameterList, error) {
				return parseParameterList(p, false)
			},
			Config{},
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
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
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
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
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
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
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
						StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
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
					Access: ast.AccessNotSpecified,
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
					Purity: ast.FunctionPurityUnspecified,
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

	t.Run("without return type, access(all)", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("access(all) fun foo () { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: ast.Position{
								Offset: 20,
								Line:   1,
								Column: 20,
							},
							EndPos: ast.Position{
								Offset: 21,
								Line:   1,
								Column: 21,
							},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{
									Offset: 23,
									Line:   1,
									Column: 23,
								},
								EndPos: ast.Position{
									Offset: 25,
									Line:   1,
									Column: 25,
								},
							},
						},
					},
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos: ast.Position{
							Offset: 16,
							Line:   1,
							Column: 16,
						},
					},
					StartPos: ast.Position{
						Offset: 0,
						Line:   1,
						Column: 0,
					},
					Access: ast.AccessAll,
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
					Access: ast.AccessNotSpecified,
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
					Access: ast.AccessNotSpecified,
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
					FunctionBlock: &ast.FunctionBlock{
						PreConditions: &ast.Conditions{
							&ast.TestCondition{
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
							&ast.TestCondition{
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
							&ast.TestCondition{
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
					Access: ast.AccessNotSpecified,
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
					Access: ast.AccessNotSpecified,
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
					Access: ast.AccessNotSpecified,
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
		// It could be the start of a intersection type.
		// However, if there is space after the brace, which is most common
		// in function declarations, we consider it not a intersection type

		t.Parallel()

		result, errs := testParseDeclarations("fun main(): Int{ return 1 }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
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

	t.Run("view function", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("view fun foo (): X { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 13, Offset: 13},
							EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					Purity: ast.FunctionPurityView,
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "X",
								Pos:        ast.Position{Line: 1, Column: 17, Offset: 17},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 19, Offset: 19},
								EndPos:   ast.Position{Line: 1, Column: 21, Offset: 21},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseDeclarations(
			nil,
			[]byte("native fun foo() {}"),
			Config{
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Flags:  ast.FunctionDeclarationFlagsIsNative,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 11, Offset: 11},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
								EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("double purity annot", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("view view fun foo (): X { }")
		require.Equal(t, 1, len(errs))
		require.Equal(t, errs[0], &SyntaxError{
			Message: "invalid second view modifier",
			Pos:     ast.Position{Offset: 5, Line: 1, Column: 5},
		})
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("native fun foo() {}")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseDeclarations(
			nil,
			[]byte("static fun foo() {}"),
			Config{
				StaticModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Flags:  ast.FunctionDeclarationFlagsIsStatic,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 11, Offset: 11},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
								EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("static fun foo() {}")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseDeclarations(
			nil,
			[]byte("static native fun foo() {}"),
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Flags:  ast.FunctionDeclarationFlagsIsStatic | ast.FunctionDeclarationFlagsIsNative,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 21, Offset: 21},
							EndPos:   ast.Position{Line: 1, Column: 22, Offset: 22},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 24, Offset: 24},
								EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("static native fun foo() {}")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte("native static fun foo() {}"),
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier after native modifier",
					Pos:     ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)
	})

	t.Run("native static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("native static fun foo() {}")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("access(all) static native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseDeclarations(
			nil,
			[]byte("access(all) static native fun foo() {}"),
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Purity:            0,
					TypeParameterList: (*ast.TypeParameterList)(nil),
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: ast.Position{
								Offset: 33,
								Line:   1,
								Column: 33,
							},
							EndPos: ast.Position{
								Offset: 34,
								Line:   1,
								Column: 34,
							},
						},
					},
					ReturnTypeAnnotation: (*ast.TypeAnnotation)(nil),
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{
									Offset: 36,
									Line:   1,
									Column: 36,
								},
								EndPos: ast.Position{
									Offset: 37,
									Line:   1,
									Column: 37,
								},
							},
						},
						PreConditions:  (*ast.Conditions)(nil),
						PostConditions: (*ast.Conditions)(nil),
					},
					DocString: "",
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos: ast.Position{
							Offset: 30,
							Line:   1,
							Column: 30,
						},
					},
					StartPos: ast.Position{
						Offset: 0,
						Line:   1,
						Column: 0,
					},
					Access: ast.AccessAll,
					Flags:  0x03,
				},
			},
			result,
		)
	})

	t.Run("access(all) static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("access(all) static native fun foo() {}")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos: ast.Position{
						Offset: 12,
						Line:   1,
						Column: 12,
					},
				},
			},
			errs,
		)
	})

	t.Run("with empty type parameters, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseDeclarations(
			nil,
			[]byte("fun foo  < > () {}"),
			Config{
				TypeParametersEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					TypeParameterList: &ast.TypeParameterList{
						TypeParameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 13, Offset: 13},
							EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
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

	t.Run("with type parameters, single type parameter, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseDeclarations(
			nil,
			[]byte("fun foo  < A  > () {}"),
			Config{
				TypeParametersEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					TypeParameterList: &ast.TypeParameterList{
						TypeParameters: []*ast.TypeParameter{
							{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos:        ast.Position{Offset: 11, Line: 1, Column: 11},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
							EndPos:   ast.Position{Line: 1, Column: 17, Offset: 17},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 19, Offset: 19},
								EndPos:   ast.Position{Line: 1, Column: 20, Offset: 20},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("with type parameters, multiple parameters, type bound, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := ParseDeclarations(
			nil,
			[]byte("fun foo  < A  , B : C > () {}"),
			Config{
				TypeParametersEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					TypeParameterList: &ast.TypeParameterList{
						TypeParameters: []*ast.TypeParameter{
							{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos:        ast.Position{Offset: 11, Line: 1, Column: 11},
								},
							},
							{
								Identifier: ast.Identifier{
									Identifier: "B",
									Pos:        ast.Position{Offset: 16, Line: 1, Column: 16},
								},
								TypeBound: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "C",
											Pos:        ast.Position{Offset: 20, Line: 1, Column: 20},
										},
									},
									StartPos: ast.Position{Offset: 20, Line: 1, Column: 20},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 22, Offset: 22},
						},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 24, Offset: 24},
							EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 27, Offset: 27},
								EndPos:   ast.Position{Line: 1, Column: 28, Offset: 28},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("with type parameters, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("fun foo<A>() {}")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected '(' as start of parameter list, got '<'",
					Pos:     ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)
	})

	t.Run("missing type parameter list end, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte("fun foo  < "),
			Config{
				TypeParametersEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "missing '>' at end of type parameter list",
					Pos:     ast.Position{Offset: 11, Line: 1, Column: 11},
				},
			},
			errs,
		)
	})

	t.Run("missing type parameter list separator, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte("fun foo  < A B > () { } "),
			Config{
				TypeParametersEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&MissingCommaInParameterListError{
					Pos: ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)
	})

}

func TestParseAccess(t *testing.T) {

	t.Parallel()

	parse := func(input string) (ast.Access, []error) {
		return Parse(
			nil,
			[]byte(input),
			parseAccess,
			Config{},
		)
	}

	t.Run("access(all)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( all )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.AccessAll,
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
			ast.AccessSelf,
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
			ast.AccessNotSpecified,
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
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, single entitlement", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.EntitlementAccess{
				EntitlementSet: &ast.ConjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("access, multiple conjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo , bar )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.EntitlementAccess{
				EntitlementSet: &ast.ConjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Offset: 15, Line: 1, Column: 15},
							},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("access, multiple disjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo | bar )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.EntitlementAccess{
				EntitlementSet: &ast.DisjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Offset: 15, Line: 1, Column: 15},
							},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("access, mixed disjunctive and conjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo | bar , baz )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: got ',', expected '|' or ')'",
					Pos:     ast.Position{Offset: 19, Line: 1, Column: 19},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, mixed conjunctive and disjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo , bar | baz )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: got '|', expected ',' or ')'",
					Pos:     ast.Position{Offset: 19, Line: 1, Column: 19},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, conjunctive entitlements list starting with keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( self , bar )")
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
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, disjunctive entitlements list starting with keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( self | bar )")
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
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, conjunctive entitlements list ending with keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo , self )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected non-nominal type: self",
					Pos:     ast.Position{Offset: 20, Line: 1, Column: 20},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, disjunctive entitlements list ending with keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo | self )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected non-nominal type: self",
					Pos:     ast.Position{Offset: 20, Line: 1, Column: 20},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, multiple entitlements no separator", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo bar )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected entitlement separator identifier",
					Pos:     ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, invalid separator", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo & bar )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected entitlement separator '&'",
					Pos:     ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, entitlement map", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( mapping foo )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.MappedAccess{
				EntitlementMap: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Offset: 17, Line: 1, Column: 17},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 1, Column: 9},
			},
			result,
		)
	})

	t.Run("access, entitlement map no name", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( mapping )")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token in type: ')'",
					Pos:     ast.Position{Offset: 18, Line: 1, Column: 18},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
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

	t.Run("two identifiers, address location, repeated commas", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseDeclarations(`import foo, , bar from 0xaaaa`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Pos:     ast.Position{Line: 1, Column: 12, Offset: 12},
					Message: `expected identifier or keyword "from", got ','`,
				},
			},
			errs,
		)
		var expected []ast.Declaration

		utils.AssertEqualWithDiff(t, expected, result)
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

	t.Run("unexpected token as identifier", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations(`import foo, bar, baz, @ from 0x42`)

		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Pos:     ast.Position{Line: 1, Column: 22, Offset: 22},
				Message: `unexpected token in import declaration: got '@', expected keyword "from" or ','`,
			},
		}, errs)

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
					Access:        ast.AccessNotSpecified,
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
									Access: ast.AccessNotSpecified,
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

		result, errs := testParseDeclarations(" access(self) event E2 ( a : Int , b : String )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "Int",
															Pos: ast.Position{
																Offset: 29,
																Line:   1,
																Column: 29,
															},
														},
													},
													StartPos: ast.Position{
														Offset: 29,
														Line:   1,
														Column: 29,
													},
												},
												Identifier: ast.Identifier{
													Identifier: "a",
													Pos: ast.Position{
														Offset: 25,
														Line:   1,
														Column: 25,
													},
												},
												StartPos: ast.Position{
													Offset: 25,
													Line:   1,
													Column: 25,
												},
											},
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "String",
															Pos: ast.Position{
																Offset: 39,
																Line:   1,
																Column: 39,
															},
														},
													},
													StartPos: ast.Position{
														Offset: 39,
														Line:   1,
														Column: 39,
													},
												},
												Identifier: ast.Identifier{
													Identifier: "b",
													Pos: ast.Position{
														Offset: 35,
														Line:   1,
														Column: 35,
													},
												},
												StartPos: ast.Position{
													Offset: 35,
													Line:   1,
													Column: 35,
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 23,
												Line:   1,
												Column: 23,
											},
											EndPos: ast.Position{
												Offset: 46,
												Line:   1,
												Column: 46,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 23,
										Line:   1,
										Column: 23,
									},
									Access: ast.AccessNotSpecified,
								},
								Kind: common.DeclarationKindInitializer,
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "E2",
						Pos: ast.Position{
							Offset: 20,
							Line:   1,
							Column: 20,
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 1,
							Line:   1,
							Column: 1,
						},
						EndPos: ast.Position{
							Offset: 46,
							Line:   1,
							Column: 46,
						},
					},
					Access:        ast.AccessSelf,
					CompositeKind: common.CompositeKindEvent,
				},
			},
			result,
		)
	})

	t.Run("default event", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` access(all) event ResourceDestroyed ( a : String = "foo")`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "String",
															Pos: ast.Position{
																Offset: 43,
																Line:   1,
																Column: 43,
															},
														},
													},
													StartPos: ast.Position{
														Offset: 43,
														Line:   1,
														Column: 43,
													},
												},
												DefaultArgument: &ast.StringExpression{
													Value: "foo",
													Range: ast.Range{
														StartPos: ast.Position{
															Offset: 52,
															Line:   1,
															Column: 52,
														},
														EndPos: ast.Position{
															Offset: 56,
															Line:   1,
															Column: 56,
														},
													},
												},
												Identifier: ast.Identifier{
													Identifier: "a",
													Pos: ast.Position{
														Offset: 39,
														Line:   1,
														Column: 39,
													},
												},
												StartPos: ast.Position{
													Offset: 39,
													Line:   1,
													Column: 39,
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 37,
												Line:   1,
												Column: 37,
											},
											EndPos: ast.Position{
												Offset: 57,
												Line:   1,
												Column: 57,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 37,
										Line:   1,
										Column: 37,
									},
									Access: ast.AccessNotSpecified,
								},
								Kind: common.DeclarationKindInitializer,
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "ResourceDestroyed",
						Pos: ast.Position{
							Offset: 19,
							Line:   1,
							Column: 19,
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 1,
							Line:   1,
							Column: 1,
						},
						EndPos: ast.Position{
							Offset: 57,
							Line:   1,
							Column: 57,
						},
					},
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindEvent,
				},
			},
			result,
		)
	})

	t.Run("default event with no default arg", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) event ResourceDestroyed ( a : Int )")

		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Pos:     ast.Position{Line: 1, Column: 47, Offset: 47},
				Message: "expected a default argument after type annotation, got ')'",
			},
		}, errs)
	})

	t.Run("non-default event with default arg", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) event Foo ( a : Int = 3)")

		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Pos:     ast.Position{Line: 1, Column: 33, Offset: 33},
				Message: "cannot use a default argument for this function",
			},
		}, errs)
	})

	t.Run("invalid event name", func(t *testing.T) {
		_, errs := testParseDeclarations(`event continue {}`)

		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Pos:     ast.Position{Line: 1, Column: 6, Offset: 6},
				Message: "expected identifier after start of event declaration, got keyword continue",
			},
		}, errs)
	})
}

func TestParseFieldWithVariableKind(t *testing.T) {

	t.Parallel()

	parse := func(input string) (*ast.FieldDeclaration, []error) {
		return Parse(
			nil,
			[]byte(input),
			func(p *parser) (*ast.FieldDeclaration, error) {
				return parseFieldWithVariableKind(
					p,
					ast.AccessNotSpecified,
					nil,
					nil,
					nil,
					"",
				)
			},
			Config{},
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

func TestParseField(t *testing.T) {

	t.Parallel()

	parse := func(input string, config Config) (ast.Declaration, []error) {
		return Parse(
			nil,
			[]byte(input),
			func(p *parser) (ast.Declaration, error) {
				return parseMemberOrNestedDeclaration(
					p,
					"",
				)
			},
			config,
		)
	}

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(
			"native let foo: Int",
			Config{
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				Flags:        ast.FieldDeclarationFlagsIsNative,
				VariableKind: ast.VariableKindConstant,
				Identifier: ast.Identifier{
					Identifier: "foo",
					Pos:        ast.Position{Line: 1, Column: 11, Offset: 11},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 16, Offset: 16},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
				},
			},
			result,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("native let foo: Int", Config{})

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(
			"static let foo: Int",
			Config{
				StaticModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				Flags:        ast.FieldDeclarationFlagsIsStatic,
				VariableKind: ast.VariableKindConstant,
				Identifier: ast.Identifier{
					Identifier: "foo",
					Pos:        ast.Position{Line: 1, Column: 11, Offset: 11},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 16, Offset: 16},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
				},
			},
			result,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"static let foo: Int",
			Config{},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(
			"static native let foo: Int",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				Flags:        ast.FieldDeclarationFlagsIsStatic | ast.FieldDeclarationFlagsIsNative,
				VariableKind: ast.VariableKindConstant,
				Identifier: ast.Identifier{
					Identifier: "foo",
					Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 23, Offset: 23},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 23, Offset: 23},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
				},
			},
			result,
		)
	})

	t.Run("static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("static native let foo: Int", Config{})

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"native static let foo: Int",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier after native modifier",
					Pos:     ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)
	})

	t.Run("access(all) static native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(
			"access(all) static native let foo: Int",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos: ast.Position{
								Offset: 35,
								Line:   1,
								Column: 35,
							},
						},
					},
					StartPos: ast.Position{
						Offset: 35,
						Line:   1,
						Column: 35,
					},
					IsResource: false,
				},
				Identifier: ast.Identifier{
					Identifier: "foo",
					Pos: ast.Position{
						Offset: 30,
						Line:   1,
						Column: 30,
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{
						Offset: 0,
						Line:   1,
						Column: 0,
					},
					EndPos: ast.Position{
						Offset: 37,
						Line:   1,
						Column: 37,
					},
				},
				Access:       ast.AccessAll,
				VariableKind: 0x2,
				Flags:        0x03,
			},
			result,
		)
	})

	t.Run("access(all) static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("access(all) static native let foo: Int", Config{})

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 12, Line: 1, Column: 12},
				},
			},
			errs,
		)
	})

}

func TestParseCompositeDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("struct, no conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) struct S { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(nil),
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos: ast.Position{
							Offset: 20,
							Line:   1,
							Column: 20,
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 1,
							Line:   1,
							Column: 1,
						},
						EndPos: ast.Position{
							Offset: 24,
							Line:   1,
							Column: 24,
						},
					},
					Access:        ast.AccessAll,
					CompositeKind: 0x1,
				},
			},
			result,
		)
	})

	t.Run("resource, one conformance", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) resource R : RI { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindResource,
					Identifier: ast.Identifier{
						Identifier: "R",
						Pos:        ast.Position{Line: 1, Column: 22, Offset: 22},
					},
					Conformances: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "RI",
								Pos:        ast.Position{Line: 1, Column: 26, Offset: 26},
							},
						},
					},
					Members: &ast.Members{},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 31, Offset: 31},
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
              access(all) var foo: Int

              init(foo: Int) {
                  self.foo = foo
              }

              access(all) fun getFoo(): Int {
                  return self.foo
              }
          }
	    `)

		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FieldDeclaration{
								TypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos: ast.Position{
												Offset: 60,
												Line:   3,
												Column: 35,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 60,
										Line:   3,
										Column: 35,
									},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos: ast.Position{
										Offset: 55,
										Line:   3,
										Column: 30,
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{
										Offset: 39,
										Line:   3,
										Column: 14,
									},
									EndPos: ast.Position{
										Offset: 62,
										Line:   3,
										Column: 37,
									},
								},
								Access:       ast.AccessAll,
								VariableKind: 0x1,
								Flags:        0x00,
							},
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "Int",
															Pos: ast.Position{
																Offset: 89,
																Line:   5,
																Column: 24,
															},
														},
													},
													StartPos: ast.Position{
														Offset: 89,
														Line:   5,
														Column: 24,
													},
													IsResource: false,
												},
												Identifier: ast.Identifier{
													Identifier: "foo",
													Pos: ast.Position{
														Offset: 84,
														Line:   5,
														Column: 19,
													},
												},
												StartPos: ast.Position{
													Offset: 84,
													Line:   5,
													Column: 19,
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 83,
												Line:   5,
												Column: 18,
											},
											EndPos: ast.Position{
												Offset: 92,
												Line:   5,
												Column: 27,
											},
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
																Pos: ast.Position{
																	Offset: 114,
																	Line:   6,
																	Column: 18,
																},
															},
														},
														Identifier: ast.Identifier{
															Identifier: "foo",
															Pos: ast.Position{
																Offset: 119,
																Line:   6,
																Column: 23,
															},
														},
														AccessPos: ast.Position{
															Offset: 118,
															Line:   6,
															Column: 22,
														},
														Optional: false,
													},
													Transfer: &ast.Transfer{
														Operation: 0x1,
														Pos: ast.Position{
															Offset: 123,
															Line:   6,
															Column: 27,
														},
													},
													Value: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "foo",
															Pos: ast.Position{
																Offset: 125,
																Line:   6,
																Column: 29,
															},
														},
													},
												},
											},
											Range: ast.Range{
												StartPos: ast.Position{
													Offset: 94,
													Line:   5,
													Column: 29,
												},
												EndPos: ast.Position{
													Offset: 143,
													Line:   7,
													Column: 14,
												},
											},
										},
									},
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos: ast.Position{
											Offset: 79,
											Line:   5,
											Column: 14,
										},
									},
									StartPos: ast.Position{
										Offset: 79,
										Line:   5,
										Column: 14,
									},
									Access: ast.AccessNotSpecified,
									Flags:  0x00,
								},
								Kind: 0xd,
							},
							&ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{
											Offset: 182,
											Line:   9,
											Column: 36,
										},
										EndPos: ast.Position{
											Offset: 183,
											Line:   9,
											Column: 37,
										},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos: ast.Position{
												Offset: 186,
												Line:   9,
												Column: 40,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 186,
										Line:   9,
										Column: 40,
									},
									IsResource: false,
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Statements: []ast.Statement{
											&ast.ReturnStatement{
												Expression: &ast.MemberExpression{
													Expression: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "self",
															Pos: ast.Position{
																Offset: 217,
																Line:   10,
																Column: 25,
															},
														},
													},
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos: ast.Position{
															Offset: 222,
															Line:   10,
															Column: 30,
														},
													},
													AccessPos: ast.Position{
														Offset: 221,
														Line:   10,
														Column: 29,
													},
													Optional: false,
												},
												Range: ast.Range{
													StartPos: ast.Position{
														Offset: 210,
														Line:   10,
														Column: 18,
													},
													EndPos: ast.Position{
														Offset: 224,
														Line:   10,
														Column: 32,
													},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 190,
												Line:   9,
												Column: 44,
											},
											EndPos: ast.Position{
												Offset: 240,
												Line:   11,
												Column: 14,
											},
										},
									},
								},
								DocString: "",
								Identifier: ast.Identifier{
									Identifier: "getFoo",
									Pos: ast.Position{
										Offset: 176,
										Line:   9,
										Column: 30,
									},
								},
								StartPos: ast.Position{
									Offset: 160,
									Line:   9,
									Column: 14,
								},
								Access: ast.AccessAll,
								Flags:  0x00,
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos: ast.Position{
							Offset: 18,
							Line:   2,
							Column: 17,
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 11,
							Line:   2,
							Column: 10,
						},
						EndPos: ast.Position{
							Offset: 252,
							Line:   12,
							Column: 10,
						},
					},
					Access:        ast.AccessNotSpecified,
					CompositeKind: 0x1,
				},
			},
			result,
		)
	})

	t.Run("struct with view member", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`struct S { 
			view fun foo() {}
		}`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{&ast.FunctionDeclaration{
							Purity: ast.FunctionPurityView,
							Access: ast.AccessNotSpecified,
							ParameterList: &ast.ParameterList{
								Range: ast.Range{
									StartPos: ast.Position{Offset: 27, Line: 2, Column: 15},
									EndPos:   ast.Position{Offset: 28, Line: 2, Column: 16},
								},
							},
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 24, Line: 2, Column: 12},
							},
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 30, Line: 2, Column: 18},
										EndPos:   ast.Position{Offset: 31, Line: 2, Column: 19},
									},
								},
							},
							StartPos: ast.Position{Offset: 15, Line: 2, Column: 3},
						}},
					),
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 3, Column: 2, Offset: 35},
					},
				},
			},
			result,
		)
	})

	t.Run("struct with view initializer", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`struct S { 
			view init() {}
		}`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{&ast.SpecialFunctionDeclaration{
							Kind: common.DeclarationKindInitializer,
							FunctionDeclaration: &ast.FunctionDeclaration{
								Access: ast.AccessNotSpecified,
								Purity: ast.FunctionPurityView,
								Identifier: ast.Identifier{
									Identifier: "init",
									Pos:        ast.Position{Offset: 20, Line: 2, Column: 8},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 24, Line: 2, Column: 12},
										EndPos:   ast.Position{Offset: 25, Line: 2, Column: 13},
									},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 27, Line: 2, Column: 15},
											EndPos:   ast.Position{Offset: 28, Line: 2, Column: 16},
										},
									},
								},
								StartPos: ast.Position{Offset: 15, Line: 2, Column: 3},
							},
						}},
					),
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 3, Column: 2, Offset: 32},
					},
				},
			},
			result,
		)
	})

	t.Run("resource with view field", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`struct S { 
			view foo: Int
		}`)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid view modifier for variable",
					Pos:     ast.Position{Offset: 15, Line: 2, Column: 3},
				},
			},
			errs,
		)
	})
}

func TestParseInvalidCompositeFunctionWithSelfParameter(t *testing.T) {

	t.Parallel()

	for _, kind := range common.CompositeKindsWithFieldsAndFunctions {
		t.Run(kind.Keyword(), func(t *testing.T) {

			var baseType = ""

			if kind == common.CompositeKindAttachment {
				baseType = "for AnyStruct"
			}

			code := fmt.Sprintf(`%s Foo %s { fun test(_ self: Int) {} }`, kind.Keyword(), baseType)

			selfKeywordPos := strings.Index(code, "self")

			expectedErrPos := ast.Position{Line: 1, Column: selfKeywordPos, Offset: selfKeywordPos}

			_, err := testParseDeclarations(code)

			utils.AssertEqualWithDiff(
				t,
				[]error{
					&SyntaxError{
						Pos:     expectedErrPos,
						Message: "expected identifier for parameter name, got keyword self",
					},
				},
				err,
			)
		})
	}
}

func TestParseInvalidParameterWithoutLabel(t *testing.T) {
	t.Parallel()

	_, errs := testParseDeclarations(`access(all) fun foo(continue: Int) {}`)

	utils.AssertEqualWithDiff(t, []error{
		&SyntaxError{
			Pos:     ast.Position{Line: 1, Column: 20, Offset: 20},
			Message: "expected identifier for argument label or parameter name, got keyword continue",
		},
	}, errs)
}

func TestParseParametersWithExtraLabels(t *testing.T) {
	t.Parallel()

	_, errs := testParseDeclarations(`access(all) fun foo(_ foo: String, label fable table: Int) {}`)

	utils.AssertEqualWithDiff(t, []error{
		&SyntaxError{
			Pos:     ast.Position{Line: 1, Column: 47, Offset: 47},
			Message: "expected ':' after parameter name, got identifier",
		},
	}, errs)
}

func TestParseAttachmentDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("no conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("access(all) attachment E for S {} ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Line: 1, Column: 23, Offset: 23},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos:        ast.Position{Line: 1, Column: 29, Offset: 29},
						},
					},
					Members: &ast.Members{},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 32, Offset: 32},
					},
				},
			},
			result,
		)
	})

	t.Run("nested in contract", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
		contract Test {
			access(all) attachment E for S {}
		}`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindContract,
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Line: 2, Column: 11, Offset: 12},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 2, Offset: 3},
						EndPos:   ast.Position{Line: 4, Column: 2, Offset: 58},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.AttachmentDeclaration{
								Access: ast.AccessAll,
								Identifier: ast.Identifier{
									Identifier: "E",
									Pos:        ast.Position{Line: 3, Column: 26, Offset: 45},
								},
								BaseType: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "S",
										Pos:        ast.Position{Line: 3, Column: 32, Offset: 51},
									},
								},
								Members: &ast.Members{},
								Range: ast.Range{
									StartPos: ast.Position{Line: 3, Column: 3, Offset: 22},
									EndPos:   ast.Position{Line: 3, Column: 35, Offset: 54},
								},
							},
						},
					),
				},
			},
			result,
		)
	})

	t.Run("missing base type", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("attachment E {} ")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected 'for', got '{'",
					Pos:     ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)
	})

	t.Run("one conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("access(all) attachment E for S: I {} ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Line: 1, Column: 23, Offset: 23},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos:        ast.Position{Line: 1, Column: 29, Offset: 29},
						},
					},
					Members: &ast.Members{},
					Conformances: []*ast.NominalType{
						ast.NewNominalType(
							nil,
							ast.Identifier{
								Identifier: "I",
								Pos:        ast.Position{Line: 1, Column: 32, Offset: 32},
							},
							nil,
						),
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 35, Offset: 35},
					},
				},
			},
			result,
		)
	})

	t.Run("two conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("access(all) attachment E for S: I1, I2 {} ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos: ast.Position{
							Offset: 23,
							Line:   1,
							Column: 23,
						},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos: ast.Position{
								Offset: 29,
								Line:   1,
								Column: 29,
							},
						},
					},
					Conformances: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "I1",
								Pos: ast.Position{
									Offset: 32,
									Line:   1,
									Column: 32,
								},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "I2",
								Pos: ast.Position{
									Offset: 36,
									Line:   1,
									Column: 36,
								},
							},
						},
					},
					Members: ast.NewUnmeteredMembers(nil),
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 0,
							Line:   1,
							Column: 0,
						},
						EndPos: ast.Position{
							Offset: 40,
							Line:   1,
							Column: 40,
						},
					},
				},
			},
			result,
		)
	})

	t.Run("fields, functions and special functions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`access(all) attachment E for S {
			access(all) var foo: Int
			init() {}
			access(all) fun getFoo(): Int {}
		}`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos: ast.Position{
							Offset: 23,
							Line:   1,
							Column: 23,
						},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos: ast.Position{
								Offset: 29,
								Line:   1,
								Column: 29,
							},
						},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FieldDeclaration{
								TypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos: ast.Position{
												Offset: 57,
												Line:   2,
												Column: 24,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 57,
										Line:   2,
										Column: 24,
									},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos: ast.Position{
										Offset: 52,
										Line:   2,
										Column: 19,
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{
										Offset: 36,
										Line:   2,
										Column: 3,
									},
									EndPos: ast.Position{
										Offset: 59,
										Line:   2,
										Column: 26,
									},
								},
								Access:       ast.AccessAll,
								VariableKind: 0x1,
								Flags:        0x00,
							},
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 68,
												Line:   3,
												Column: 7,
											},
											EndPos: ast.Position{
												Offset: 69,
												Line:   3,
												Column: 8,
											},
										},
									},
									FunctionBlock: &ast.FunctionBlock{
										Block: &ast.Block{
											Range: ast.Range{
												StartPos: ast.Position{
													Offset: 71,
													Line:   3,
													Column: 10,
												},
												EndPos: ast.Position{
													Offset: 72,
													Line:   3,
													Column: 11,
												},
											},
										},
									},
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos: ast.Position{
											Offset: 64,
											Line:   3,
											Column: 3,
										},
									},
									StartPos: ast.Position{
										Offset: 64,
										Line:   3,
										Column: 3,
									},
									Access: ast.AccessNotSpecified,
								},
								Kind: 0xd,
							},
							&ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{
											Offset: 99,
											Line:   4,
											Column: 25,
										},
										EndPos: ast.Position{
											Offset: 100,
											Line:   4,
											Column: 26,
										},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos: ast.Position{
												Offset: 103,
												Line:   4,
												Column: 29,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 103,
										Line:   4,
										Column: 29,
									},
									IsResource: false,
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 107,
												Line:   4,
												Column: 33,
											},
											EndPos: ast.Position{
												Offset: 108,
												Line:   4,
												Column: 34,
											},
										},
									},
								},
								Identifier: ast.Identifier{
									Identifier: "getFoo",
									Pos: ast.Position{
										Offset: 93,
										Line:   4,
										Column: 19,
									},
								},
								StartPos: ast.Position{
									Offset: 77,
									Line:   4,
									Column: 3,
								},
								Access: ast.AccessAll,
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 0,
							Line:   1,
							Column: 0,
						},
						EndPos: ast.Position{
							Offset: 112,
							Line:   5,
							Column: 2,
						},
					},
				},
			},
			result,
		)
	})

	t.Run("required entitlements error", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`access(all) attachment E for S {
			require entitlement X
		}`)
		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Pos:     ast.Position{Line: 2, Column: 3, Offset: 36},
				Message: "unexpected identifier",
			},
		}, errs)
	})

	t.Run("entitlement access", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`access(all) attachment E for S {
			access(X) var foo: Int
		}`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos: ast.Position{
							Offset: 23,
							Line:   1,
							Column: 23,
						},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos: ast.Position{
								Offset: 29,
								Line:   1,
								Column: 29,
							},
						},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FieldDeclaration{
								TypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos: ast.Position{
												Offset: 55,
												Line:   2,
												Column: 22,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 55,
										Line:   2,
										Column: 22,
									},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos: ast.Position{
										Offset: 50,
										Line:   2,
										Column: 17,
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{
										Offset: 36,
										Line:   2,
										Column: 3,
									},
									EndPos: ast.Position{
										Offset: 57,
										Line:   2,
										Column: 24,
									},
								},
								Access: ast.EntitlementAccess{
									EntitlementSet: &ast.ConjunctiveEntitlementSet{
										Elements: []*ast.NominalType{
											{
												Identifier: ast.Identifier{
													Identifier: "X",
													Pos: ast.Position{
														Offset: 43,
														Line:   2,
														Column: 10,
													},
												},
											},
										},
									},
								},
								VariableKind: 0x1,
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 0,
							Line:   1,
							Column: 0,
						},
						EndPos: ast.Position{
							Offset: 61,
							Line:   3,
							Column: 2,
						},
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

		result, errs := testParseDeclarations(" access(all) struct interface S { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.InterfaceDeclaration{
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 1, Column: 30, Offset: 30},
					},
					Members: &ast.Members{},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 34, Offset: 34},
					},
				},
			},
			result,
		)
	})

	t.Run("struct, interface keyword as name", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) struct interface interface { }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected interface name, got keyword \"interface\"",
					Pos:     ast.Position{Offset: 30, Line: 1, Column: 30},
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
              access(all) var foo: Int

              init(foo: Int)

              access(all) fun getFoo(): Int

              access(all) fun getBar(): Int {}
          }
	    `)

		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.InterfaceDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FieldDeclaration{
								TypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos: ast.Position{
												Offset: 70,
												Line:   3,
												Column: 35,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 70,
										Line:   3,
										Column: 35,
									},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos: ast.Position{
										Offset: 65,
										Line:   3,
										Column: 30,
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{
										Offset: 49,
										Line:   3,
										Column: 14,
									},
									EndPos: ast.Position{
										Offset: 72,
										Line:   3,
										Column: 37,
									},
								},
								Access:       ast.AccessAll,
								VariableKind: 0x1,
								Flags:        0x00,
							},
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "Int",
															Pos: ast.Position{
																Offset: 99,
																Line:   5,
																Column: 24,
															},
														},
													},
													StartPos: ast.Position{
														Offset: 99,
														Line:   5,
														Column: 24,
													},
													IsResource: false,
												},
												Identifier: ast.Identifier{
													Identifier: "foo",
													Pos: ast.Position{
														Offset: 94,
														Line:   5,
														Column: 19,
													},
												},
												StartPos: ast.Position{
													Offset: 94,
													Line:   5,
													Column: 19,
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 93,
												Line:   5,
												Column: 18,
											},
											EndPos: ast.Position{
												Offset: 102,
												Line:   5,
												Column: 27,
											},
										},
									},
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos: ast.Position{
											Offset: 89,
											Line:   5,
											Column: 14,
										},
									},
									StartPos: ast.Position{
										Offset: 89,
										Line:   5,
										Column: 14,
									},
									Access: ast.AccessNotSpecified,
									Flags:  0x00,
								},
								Kind: 0xd,
							},
							&ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{
											Offset: 141,
											Line:   7,
											Column: 36,
										},
										EndPos: ast.Position{
											Offset: 142,
											Line:   7,
											Column: 37,
										},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos: ast.Position{
												Offset: 145,
												Line:   7,
												Column: 40,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 145,
										Line:   7,
										Column: 40,
									},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "getFoo",
									Pos: ast.Position{
										Offset: 135,
										Line:   7,
										Column: 30,
									},
								},
								StartPos: ast.Position{
									Offset: 119,
									Line:   7,
									Column: 14,
								},
								Access: ast.AccessAll,
								Flags:  0x00,
							},
							&ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{
											Offset: 186,
											Line:   9,
											Column: 36,
										},
										EndPos: ast.Position{
											Offset: 187,
											Line:   9,
											Column: 37,
										},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos: ast.Position{
												Offset: 190,
												Line:   9,
												Column: 40,
											},
										},
									},
									StartPos: ast.Position{
										Offset: 190,
										Line:   9,
										Column: 40,
									},
									IsResource: false,
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 194,
												Line:   9,
												Column: 44,
											},
											EndPos: ast.Position{
												Offset: 195,
												Line:   9,
												Column: 45,
											},
										},
									},
								},
								Identifier: ast.Identifier{
									Identifier: "getBar",
									Pos: ast.Position{
										Offset: 180,
										Line:   9,
										Column: 30,
									},
								},
								StartPos: ast.Position{
									Offset: 164,
									Line:   9,
									Column: 14,
								},
								Access: ast.AccessAll,
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos: ast.Position{
							Offset: 28,
							Line:   2,
							Column: 27,
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 11,
							Line:   2,
							Column: 10,
						},
						EndPos: ast.Position{
							Offset: 207,
							Line:   10,
							Column: 10,
						},
					},
					Access:        ast.AccessNotSpecified,
					CompositeKind: 0x1,
				},
			},
			result,
		)
	})

	t.Run("invalid interface name", func(t *testing.T) {
		_, errs := testParseDeclarations(`access(all) struct interface continue {}`)

		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Pos:     ast.Position{Line: 1, Column: 29, Offset: 29},
				Message: "expected identifier following struct declaration, got keyword continue",
			},
		}, errs)
	})

	t.Run("struct with view member", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`struct interface S { 
			view fun foo() {}
		}`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.InterfaceDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 1, Column: 17, Offset: 17},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{&ast.FunctionDeclaration{
							Purity: ast.FunctionPurityView,
							Access: ast.AccessNotSpecified,
							ParameterList: &ast.ParameterList{
								Range: ast.Range{
									StartPos: ast.Position{Offset: 37, Line: 2, Column: 15},
									EndPos:   ast.Position{Offset: 38, Line: 2, Column: 16},
								},
							},
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 34, Line: 2, Column: 12},
							},
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 40, Line: 2, Column: 18},
										EndPos:   ast.Position{Offset: 41, Line: 2, Column: 19},
									},
								},
							},
							StartPos: ast.Position{Offset: 25, Line: 2, Column: 3},
						}},
					),
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 3, Column: 2, Offset: 45},
					},
				},
			},
			result,
		)
	})
}

func TestParseEnumDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("enum, two cases one one line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) enum E { case c ; access(all) case d }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindEnum,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.EnumCaseDeclaration{
								Access: ast.AccessNotSpecified,
								Identifier: ast.Identifier{
									Identifier: "c",
									Pos:        ast.Position{Line: 1, Column: 27, Offset: 27},
								},
								StartPos: ast.Position{Line: 1, Column: 22, Offset: 22},
							},
							&ast.EnumCaseDeclaration{
								Access: ast.AccessAll,
								Identifier: ast.Identifier{
									Identifier: "d",
									Pos:        ast.Position{Line: 1, Column: 48, Offset: 48},
								},
								StartPos: ast.Position{Line: 1, Column: 31, Offset: 31},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 50, Offset: 50},
					},
				},
			},
			result,
		)
	})

	t.Run("enum case with view modifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" enum E { view case e }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid view modifier for enum case",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
		)
	})

	t.Run("enum case with static modifier, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(" enum E { static case e }"),
			Config{
				StaticModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier for enum case",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
		)
	})

	t.Run("enum case with static modifier, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" enum E { static case e }")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
		)
	})

	t.Run("enum case with native modifier, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(" enum E { native case e }"),
			Config{
				NativeModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid native modifier for enum case",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
		)
	})

	t.Run("enum case with native modifier, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" enum E { native case e }")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
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

		    prepare(signer: &Account) {
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
											Type: &ast.ReferenceType{
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Account",
														Pos: ast.Position{
															Offset: 61,
															Line:   6,
															Column: 23,
														},
													},
												},
												StartPos: ast.Position{Offset: 60, Line: 6, Column: 22},
											},
											StartPos: ast.Position{Offset: 60, Line: 6, Column: 22},
										},
										StartPos: ast.Position{Offset: 52, Line: 6, Column: 14},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 51, Line: 6, Column: 13},
									EndPos:   ast.Position{Offset: 68, Line: 6, Column: 30},
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
													Pos:        ast.Position{Offset: 83, Line: 7, Column: 11},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 85, Line: 7, Column: 13},
											},
											Value: &ast.IntegerExpression{
												PositiveLiteral: []byte("0"),
												Value:           new(big.Int),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 87, Line: 7, Column: 15},
													EndPos:   ast.Position{Offset: 87, Line: 7, Column: 15},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 70, Line: 6, Column: 32},
										EndPos:   ast.Position{Offset: 92, Line: 8, Column: 3},
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
								Pos:        ast.Position{Offset: 101, Line: 10, Column: 6},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 122, Line: 11, Column: 11},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 124, Line: 11, Column: 13},
											},
											Value: &ast.BinaryExpression{
												Operation: ast.OperationPlus,
												Left: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 126, Line: 11, Column: 15},
														EndPos:   ast.Position{Offset: 126, Line: 11, Column: 15},
													},
												},
												Right: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 130, Line: 11, Column: 19},
														EndPos:   ast.Position{Offset: 130, Line: 11, Column: 19},
													},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 109, Line: 10, Column: 14},
										EndPos:   ast.Position{Offset: 135, Line: 12, Column: 3},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 101, Line: 10, Column: 6},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 5, Line: 2, Column: 4},
						EndPos:   ast.Position{Offset: 141, Line: 13, Column: 4},
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
										StartPos: ast.Position{Offset: 52, Line: 6, Column: 14},
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
						&ast.TestCondition{
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
						&ast.TestCondition{
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
										StartPos: ast.Position{Offset: 52, Line: 6, Column: 14},
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
						&ast.TestCondition{
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
						&ast.TestCondition{
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

	t.Run("invalid identifiers instead of special function declarations", func(t *testing.T) {
		code := `
		transaction {
			var x: Int

			uwu(signer: AuthAccount) {}

			pre {
				x > 1
			}
			post {
				x == 2
			}

		}
		`

		_, errs := testParseDeclarations(code)

		utils.AssertEqualWithDiff(t,
			`unexpected identifier, expected keyword "prepare" or "execute", got "uwu"`,
			errs[0].Error(),
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
							StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
					},
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
							StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
					},
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
            access(all) var foo: Int

            init(foo: Int) {
                self.foo = foo
            }

            access(all) fun getFoo(): Int {
                return self.foo
            }
        }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							TypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos: ast.Position{
											Offset: 56,
											Line:   3,
											Column: 33,
										},
									},
								},
								StartPos: ast.Position{
									Offset: 56,
									Line:   3,
									Column: 33,
								},
								IsResource: false,
							},
							DocString: "",
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos: ast.Position{
									Offset: 51,
									Line:   3,
									Column: 28,
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{
									Offset: 35,
									Line:   3,
									Column: 12,
								},
								EndPos: ast.Position{
									Offset: 58,
									Line:   3,
									Column: 35,
								},
							},
							Access:       ast.AccessAll,
							VariableKind: 0x1,
						},
						&ast.SpecialFunctionDeclaration{
							FunctionDeclaration: &ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Parameters: []*ast.Parameter{
										{
											TypeAnnotation: &ast.TypeAnnotation{
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Int",
														Pos: ast.Position{
															Offset: 83,
															Line:   5,
															Column: 22,
														},
													},
												},
												StartPos: ast.Position{
													Offset: 83,
													Line:   5,
													Column: 22,
												},
												IsResource: false,
											},
											Identifier: ast.Identifier{
												Identifier: "foo",
												Pos: ast.Position{
													Offset: 78,
													Line:   5,
													Column: 17,
												},
											},
											StartPos: ast.Position{
												Offset: 78,
												Line:   5,
												Column: 17,
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{
											Offset: 77,
											Line:   5,
											Column: 16,
										},
										EndPos: ast.Position{
											Offset: 86,
											Line:   5,
											Column: 25,
										},
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
															Pos: ast.Position{
																Offset: 106,
																Line:   6,
																Column: 16,
															},
														},
													},
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos: ast.Position{
															Offset: 111,
															Line:   6,
															Column: 21,
														},
													},
													AccessPos: ast.Position{
														Offset: 110,
														Line:   6,
														Column: 20,
													},
													Optional: false,
												},
												Transfer: &ast.Transfer{
													Operation: 0x1,
													Pos: ast.Position{
														Offset: 115,
														Line:   6,
														Column: 25,
													},
												},
												Value: &ast.IdentifierExpression{
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos: ast.Position{
															Offset: 117,
															Line:   6,
															Column: 27,
														},
													},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 88,
												Line:   5,
												Column: 27,
											},
											EndPos: ast.Position{
												Offset: 133,
												Line:   7,
												Column: 12,
											},
										},
									},
								},
								Identifier: ast.Identifier{
									Identifier: "init",
									Pos: ast.Position{
										Offset: 73,
										Line:   5,
										Column: 12,
									},
								},
								StartPos: ast.Position{
									Offset: 73,
									Line:   5,
									Column: 12,
								},
								Access: ast.AccessNotSpecified,
							},
							Kind: 0xd,
						},
						&ast.FunctionDeclaration{
							ParameterList: &ast.ParameterList{
								Range: ast.Range{
									StartPos: ast.Position{
										Offset: 170,
										Line:   9,
										Column: 34,
									},
									EndPos: ast.Position{
										Offset: 171,
										Line:   9,
										Column: 35,
									},
								},
							},
							ReturnTypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos: ast.Position{
											Offset: 174,
											Line:   9,
											Column: 38,
										},
									},
								},
								StartPos: ast.Position{
									Offset: 174,
									Line:   9,
									Column: 38,
								},
								IsResource: false,
							},
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.ReturnStatement{
											Expression: &ast.MemberExpression{
												Expression: &ast.IdentifierExpression{
													Identifier: ast.Identifier{
														Identifier: "self",
														Pos: ast.Position{
															Offset: 203,
															Line:   10,
															Column: 23,
														},
													},
												},
												Identifier: ast.Identifier{
													Identifier: "foo",
													Pos: ast.Position{
														Offset: 208,
														Line:   10,
														Column: 28,
													},
												},
												AccessPos: ast.Position{
													Offset: 207,
													Line:   10,
													Column: 27,
												},
												Optional: false,
											},
											Range: ast.Range{
												StartPos: ast.Position{
													Offset: 196,
													Line:   10,
													Column: 16,
												},
												EndPos: ast.Position{
													Offset: 210,
													Line:   10,
													Column: 30,
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{
											Offset: 178,
											Line:   9,
											Column: 42,
										},
										EndPos: ast.Position{
											Offset: 224,
											Line:   11,
											Column: 12,
										},
									},
								},
							},
							Identifier: ast.Identifier{
								Identifier: "getFoo",
								Pos: ast.Position{
									Offset: 164,
									Line:   9,
									Column: 28,
								},
							},
							StartPos: ast.Position{
								Offset: 148,
								Line:   9,
								Column: 12,
							},
							Access: ast.AccessAll,
							Flags:  0x00,
						},
					},
				),
				Identifier: ast.Identifier{
					Identifier: "Test",
					Pos: ast.Position{
						Offset: 16,
						Line:   2,
						Column: 15,
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{
						Offset: 9,
						Line:   2,
						Column: 8,
					},
					EndPos: ast.Position{
						Offset: 234,
						Line:   12,
						Column: 8,
					},
				},
				Access:        ast.AccessNotSpecified,
				CompositeKind: 0x1,
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
				Access:        ast.AccessNotSpecified,
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

func TestParseInvalidMember(t *testing.T) {

	t.Parallel()

	const code = `
        struct Test {
            foo let x: Int
        }
	`

	t.Run("ignore", func(t *testing.T) {
		t.Parallel()

		_, errs := ParseDeclarations(nil, []byte(code), Config{
			IgnoreLeadingIdentifierEnabled: true,
		})
		require.Empty(t, errs)

	})

	t.Run("report", func(t *testing.T) {
		t.Parallel()

		_, errs := ParseDeclarations(nil, []byte(code), Config{
			IgnoreLeadingIdentifierEnabled: false,
		})

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 35, Line: 3, Column: 12},
				},
			},
			errs,
		)
	})
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
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
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
						&ast.TestCondition{
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
						&ast.TestCondition{
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
						&ast.TestCondition{
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
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
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
						&ast.TestCondition{
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

func TestParseInvalidEmitConditionNonInvocation(t *testing.T) {

	t.Parallel()

	t.Run("pre-condition", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          fun test(n: Int) {
              pre {
                  emit Foo
              }
          }
	    `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token '('",
					Pos:     ast.Position{Offset: 91, Line: 5, Column: 14},
				},
			},
			errs,
		)
	})

	t.Run("post-condition", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          fun test(n: Int) {
              post {
                  emit Foo
              }
          }
        `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token '('",
					Pos:     ast.Position{Offset: 92, Line: 5, Column: 14},
				},
			},
			errs,
		)
	})
}

func TestParseEmitAndTestCondition(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(n: Int) {
            pre {
                emit Foo()
                n > 0
            }
            post {
                n > 0
                emit Bar()
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
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.ReturnStatement{
								Expression: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "n",
										Pos:        ast.Position{Offset: 210, Line: 11, Column: 19},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 203, Line: 11, Column: 12},
									EndPos:   ast.Position{Offset: 210, Line: 11, Column: 19},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:   ast.Position{Offset: 220, Line: 12, Column: 8},
						},
					},
					PreConditions: &ast.Conditions{
						&ast.EmitCondition{
							InvocationExpression: &ast.InvocationExpression{
								InvokedExpression: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "Foo",
										Pos:        ast.Position{Offset: 67, Line: 4, Column: 21},
									},
								},
								ArgumentsStartPos: ast.Position{Offset: 70, Line: 4, Column: 24},
								EndPos:            ast.Position{Offset: 71, Line: 4, Column: 25},
							},
							StartPos: ast.Position{Offset: 62, Line: 4, Column: 16},
						},
						&ast.TestCondition{
							Test: &ast.BinaryExpression{
								Operation: ast.OperationGreater,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "n",
										Pos:        ast.Position{Offset: 89, Line: 5, Column: 16},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 93, Line: 5, Column: 20},
										EndPos:   ast.Position{Offset: 93, Line: 5, Column: 20},
									},
								},
							},
						},
					},
					PostConditions: &ast.Conditions{
						&ast.TestCondition{
							Test: &ast.BinaryExpression{
								Operation: ast.OperationGreater,
								Left: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "n",
										Pos:        ast.Position{Offset: 144, Line: 8, Column: 16},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 148, Line: 8, Column: 20},
										EndPos:   ast.Position{Offset: 148, Line: 8, Column: 20},
									},
								},
							},
						},
						&ast.EmitCondition{
							InvocationExpression: &ast.InvocationExpression{
								InvokedExpression: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "Bar",
										Pos:        ast.Position{Offset: 171, Line: 9, Column: 21},
									},
								},
								ArgumentsStartPos: ast.Position{Offset: 174, Line: 9, Column: 24},
								EndPos:            ast.Position{Offset: 175, Line: 9, Column: 25},
							},
							StartPos: ast.Position{Offset: 166, Line: 9, Column: 16},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseInterface(t *testing.T) {

	t.Parallel()

	for _, kind := range common.InstantiableCompositeKindsWithFieldsAndFunctions {

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
			Access:        ast.AccessNotSpecified,
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
							Access: ast.AccessNotSpecified,
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
										StartPos: ast.Position{Offset: 84, Line: 5, Column: 21},
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

	t.Run("identifier", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`#pedantic`)
		require.Empty(t, errs)

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
			result,
		)
	})

	t.Run("with purity", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("view #foo")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid view modifier for pragma",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte("static #foo"),
			Config{
				StaticModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier for pragma",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("static #foo")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte("native #foo"),
			Config{
				NativeModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid native modifier for pragma",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("native #foo")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})
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

func TestParseInvalidImportWithPurity(t *testing.T) {

	t.Parallel()

	const code = `
        view import x from 0x1
	`
	_, errs := testParseDeclarations(code)

	utils.AssertEqualWithDiff(t,
		[]error{
			&SyntaxError{
				Message: "invalid view modifier for import",
				Pos:     ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		errs,
	)
}

func TestParseInvalidDefaultArgument(t *testing.T) {

	t.Parallel()

	t.Run("function declaration ", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) fun foo ( a : Int = 3) { } ")

		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Pos:     ast.Position{Line: 1, Column: 31, Offset: 31},
				Message: "cannot use a default argument for this function",
			},
		}, errs)
	})

	t.Run("function expression ", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" let foo = fun ( a : Int = 3) { } ")

		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Pos:     ast.Position{Line: 1, Column: 25, Offset: 25},
				Message: "cannot use a default argument for this function",
			},
		}, errs)
	})
}

func TestParseInvalidEventWithPurity(t *testing.T) {

	t.Parallel()

	const code = `
        view event Foo()
	`
	_, errs := testParseDeclarations(code)

	utils.AssertEqualWithDiff(t,
		[]error{
			&SyntaxError{
				Message: "invalid view modifier for event",
				Pos:     ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		errs,
	)
}

func TestParseInvalidCompositeWithPurity(t *testing.T) {

	t.Parallel()

	const code = `
        view struct S {}
	`
	_, errs := testParseDeclarations(code)

	utils.AssertEqualWithDiff(t,
		[]error{
			&SyntaxError{
				Message: "invalid view modifier for struct",
				Pos:     ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		errs,
	)
}

func TestParseInvalidTransactionWithPurity(t *testing.T) {

	t.Parallel()

	const code = `
        view transaction {}
	`
	_, errs := testParseDeclarations(code)

	utils.AssertEqualWithDiff(t,
		[]error{
			&SyntaxError{
				Message: "invalid view modifier for transaction",
				Pos:     ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		errs,
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
				Access:        ast.AccessNotSpecified,
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
				Access:        ast.AccessNotSpecified,
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
								Access: ast.AccessNotSpecified,
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
											StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
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
											StartPos: ast.Position{Offset: 37, Line: 2, Column: 36},
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
											LabelEndPos:   &ast.Position{Offset: 44, Line: 3, Column: 24},
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
											LabelEndPos:   &ast.Position{Offset: 53, Line: 3, Column: 33},
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
				Access: ast.AccessNotSpecified,
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
				Access:     ast.AccessNotSpecified,
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
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
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
				Access:     ast.AccessNotSpecified,
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
				Access:        ast.AccessNotSpecified,
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
	_, errs := testParseDeclarations(code)
	utils.AssertEqualWithDiff(t,
		[]error{
			&CustomDestructorError{
				Pos: ast.Position{Offset: 37, Line: 3, Column: 12},
			},
		},
		errs,
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
				Access:        ast.AccessNotSpecified,
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "Kitty",
					Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							Access:       ast.AccessNotSpecified,
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
								Access: ast.AccessNotSpecified,
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
											StartPos: ast.Position{Offset: 43, Line: 2, Column: 42},
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

func TestParseInvalidCompositeFunctionNames(t *testing.T) {

	t.Parallel()

	interfacePossibilities := []bool{true, false}

	for _, kind := range common.CompositeKindsWithFieldsAndFunctions {
		for _, isInterface := range interfacePossibilities {

			interfaceKeyword := ""
			if isInterface {
				interfaceKeyword = "interface"
			}

			var baseType = ""

			if kind == common.CompositeKindAttachment {
				if isInterface {
					continue
				}
				baseType = "for AnyStruct"
			}

			body := "{}"
			if isInterface {
				body = ""
			}

			testName := fmt.Sprintf("%s_%s", kind.Keyword(), interfaceKeyword)

			t.Run(testName, func(t *testing.T) {

				_, err := ParseProgram(
					nil,
					[]byte(fmt.Sprintf(
						`
                          %[1]s %[2]s Test %[4]s {
                              fun init() %[3]s
                          }
                        `,
						kind.Keyword(),
						interfaceKeyword,
						body,
						baseType,
					)),
					Config{},
				)

				errs, ok := err.(Error)
				assert.True(t, ok, "Parser error does not conform to parser.Error")
				syntaxErr := errs.Errors[0].(*SyntaxError)

				utils.AssertEqualWithDiff(
					t,
					"expected identifier after start of function declaration, got keyword init",
					syntaxErr.Message,
				)
			})
		}
	}
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

			var baseType = ""

			if compositeKind == common.CompositeKindAttachment {
				baseType = "for AnyStruct"
			}

			formatCode := func(format string) string {
				return fmt.Sprintf(format, compositeKind.Keyword(), interfaceKeyword, baseType)
			}

			if compositeKind == common.CompositeKindEvent {
				declarations = append(declarations,
					declaration{
						formatName("itself"),
						formatCode("%%s %s %s Test()%s"),
					},
				)
			} else {
				declarations = append(declarations,
					declaration{
						formatName("itself"),
						formatCode("%%s %s %s Test %s {}"),
					},
					declaration{
						formatName("field"),
						formatCode("%s %s Test %s { %%s let test: Int ; init() { self.test = 1 } }"),
					},
					declaration{
						formatName("function"),
						formatCode("%s %s Test %s { %%s fun test() {} }"),
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
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
							EndPos:   ast.Position{Offset: 105, Line: 7, Column: 6},
						},
					},
					PreConditions: &ast.Conditions{
						&ast.TestCondition{
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
						&ast.TestCondition{
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

		_, errs := testParseDeclarations("access(all) #test")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid access modifier for pragma",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("access(all) transaction {}")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid access modifier for transaction",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("access(all) access(self) let x = 1")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid second access modifier",
					Pos:     ast.Position{Offset: 12, Line: 1, Column: 12},
				},
			},
			errs,
		)
	})
}

func TestParseInvalidImportWithModifier(t *testing.T) {

	t.Parallel()

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(`
                static import x from 0x1
	        `),
			Config{
				StaticModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier for import",
					Pos:     ast.Position{Offset: 17, Line: 2, Column: 16},
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            static import x from 0x1
	    `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 13, Line: 2, Column: 12},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(`
                native import x from 0x1
	        `),
			Config{
				NativeModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid native modifier for import",
					Pos:     ast.Position{Offset: 17, Line: 2, Column: 16},
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            native import x from 0x1
	    `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 13, Line: 2, Column: 12},
				},
			},
			errs,
		)
	})
}

func TestParseInvalidEventWithModifier(t *testing.T) {

	t.Parallel()

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(`
                static event Foo()
	        `),
			Config{
				StaticModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier for event",
					Pos:     ast.Position{Offset: 17, Line: 2, Column: 16},
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            static event Foo()
	    `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 13, Line: 2, Column: 12},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(`
                native event Foo()
	        `),
			Config{
				NativeModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid native modifier for event",
					Pos:     ast.Position{Offset: 17, Line: 2, Column: 16},
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            native event Foo()
	    `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 13, Line: 2, Column: 12},
				},
			},
			errs,
		)
	})

}

func TestParseCompositeWithModifier(t *testing.T) {

	t.Parallel()

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(`
                static struct Foo()
	        `),
			Config{
				StaticModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier for structure",
					Pos:     ast.Position{Offset: 17, Line: 2, Column: 16},
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            static struct Foo()
	    `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 13, Line: 2, Column: 12},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(`
                native struct Foo()
	        `),
			Config{
				NativeModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid native modifier for structure",
					Pos:     ast.Position{Offset: 17, Line: 2, Column: 16},
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            native struct Foo()
	    `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 13, Line: 2, Column: 12},
				},
			},
			errs,
		)
	})
}

func TestParseTransactionWithModifier(t *testing.T) {

	t.Parallel()

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(`
                static transaction {}
	        `),
			Config{
				StaticModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier for transaction",
					Pos:     ast.Position{Offset: 17, Line: 2, Column: 16},
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            static transaction {}
	    `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 13, Line: 2, Column: 12},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := ParseDeclarations(
			nil,
			[]byte(`
                native transaction {}
	        `),
			Config{
				NativeModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid native modifier for transaction",
					Pos:     ast.Position{Offset: 17, Line: 2, Column: 16},
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            native transaction {}
	    `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 13, Line: 2, Column: 12},
				},
			},
			errs,
		)
	})
}

func TestParseNestedPragma(t *testing.T) {

	t.Parallel()

	parse := func(input string, config Config) (ast.Declaration, []error) {
		return Parse(
			nil,
			[]byte(input),
			func(p *parser) (ast.Declaration, error) {
				return parseMemberOrNestedDeclaration(
					p,
					"",
				)
			},
			config,
		)
	}

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"native #foo",
			Config{
				NativeModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid native modifier for pragma",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("native #pragma", Config{})

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"static #pragma",
			Config{
				StaticModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier for pragma",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"static #pragma",
			Config{},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"static native #pragma",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier for pragma",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("static native #pragma", Config{})

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"native static #pragma",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid static modifier after native modifier",
					Pos:     ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)
	})

	t.Run("access(all)", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("access(all) #pragma", Config{})

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid access modifier for pragma",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("access(all) static native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"access(all) static native #pragma",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid access modifier for pragma",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("access(all) static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("access(all) static native #pragma", Config{})

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 12, Line: 1, Column: 12},
				},
			},
			errs,
		)
	})

}

func TestParseEntitlementDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) entitlement E ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.EntitlementDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Line: 1, Column: 25, Offset: 25},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
					},
				},
			},
			result,
		)
	})

	t.Run("nested entitlement", func(t *testing.T) {

		t.Parallel()

		// at static checking time, all entitlements nested inside non-contract-kinded composites
		// will be rejected
		result, errs := testParseDeclarations(`
            access(all) contract C { 
                access(all) entitlement E
            }
        `)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.EntitlementDeclaration{
								Access: ast.AccessAll,
								Identifier: ast.Identifier{
									Identifier: "E",
									Pos: ast.Position{
										Offset: 79,
										Line:   3,
										Column: 40,
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{
										Offset: 55,
										Line:   3,
										Column: 16,
									},
									EndPos: ast.Position{
										Offset: 79,
										Line:   3,
										Column: 40,
									},
								},
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "C",
						Pos: ast.Position{
							Offset: 34,
							Line:   2,
							Column: 33,
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 13,
							Line:   2,
							Column: 12,
						},
						EndPos: ast.Position{
							Offset: 93,
							Line:   4,
							Column: 12,
						},
					},
					Access:        ast.AccessAll,
					CompositeKind: 0x3,
				},
			},
			result,
		)
	})

	t.Run("no identifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected identifier, got EOF",
					Pos:     ast.Position{Offset: 24, Line: 1, Column: 24},
				},
			},
			errs,
		)
	})

	t.Run("view modifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) view entitlement E")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid view modifier for entitlement",
					Pos:     ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)
	})
}

func TestParseMemberDocStrings(t *testing.T) {

	t.Parallel()

	t.Run("functions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct Test {

              /// noReturnNoBlock
              fun noReturnNoBlock()

              /// returnNoBlock
              fun returnNoBlock(): Int

              /// returnAndBlock
              fun returnAndBlock(): String {}
          }
	    `)

		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FunctionDeclaration{
								Access:    ast.AccessNotSpecified,
								DocString: " noReturnNoBlock",
								Identifier: ast.Identifier{
									Identifier: "noReturnNoBlock",
									Pos:        ast.Position{Offset: 78, Line: 5, Column: 18},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 93, Line: 5, Column: 33},
										EndPos:   ast.Position{Offset: 94, Line: 5, Column: 34},
									},
								},
								StartPos: ast.Position{Offset: 74, Line: 5, Column: 14},
							},
							&ast.FunctionDeclaration{
								Access:    ast.AccessNotSpecified,
								DocString: " returnNoBlock",
								Identifier: ast.Identifier{
									Identifier: "returnNoBlock",
									Pos:        ast.Position{Offset: 147, Line: 8, Column: 18},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 160, Line: 8, Column: 31},
										EndPos:   ast.Position{Offset: 161, Line: 8, Column: 32},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 164, Line: 8, Column: 35},
										},
									},
									StartPos: ast.Position{Offset: 164, Line: 8, Column: 35},
								},
								StartPos: ast.Position{Offset: 143, Line: 8, Column: 14},
							},
							&ast.FunctionDeclaration{
								Access:    ast.AccessNotSpecified,
								DocString: " returnAndBlock",
								Identifier: ast.Identifier{
									Identifier: "returnAndBlock",
									Pos:        ast.Position{Offset: 220, Line: 11, Column: 18},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 234, Line: 11, Column: 32},
										EndPos:   ast.Position{Offset: 235, Line: 11, Column: 33},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "String",
											Pos:        ast.Position{Offset: 238, Line: 11, Column: 36},
										},
									},
									StartPos: ast.Position{Offset: 238, Line: 11, Column: 36},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 245, Line: 11, Column: 43},
											EndPos:   ast.Position{Offset: 246, Line: 11, Column: 44},
										},
									},
								},
								StartPos: ast.Position{Offset: 216, Line: 11, Column: 14},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 258, Line: 12, Column: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("special functions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct Test {

              /// unknown
              unknown()

              /// initNoBlock
              init()
          }
	    `)

		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindUnknown,
								FunctionDeclaration: &ast.FunctionDeclaration{
									Access:    ast.AccessNotSpecified,
									DocString: " unknown",
									Identifier: ast.Identifier{
										Identifier: "unknown",
										Pos:        ast.Position{Offset: 66, Line: 5, Column: 14},
									},
									ParameterList: &ast.ParameterList{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 73, Line: 5, Column: 21},
											EndPos:   ast.Position{Offset: 74, Line: 5, Column: 22},
										},
									},
									StartPos: ast.Position{Offset: 66, Line: 5, Column: 14},
								},
							},
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindInitializer,
								FunctionDeclaration: &ast.FunctionDeclaration{
									Access:    ast.AccessNotSpecified,
									DocString: " initNoBlock",
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos:        ast.Position{Offset: 121, Line: 8, Column: 14},
									},
									ParameterList: &ast.ParameterList{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 125, Line: 8, Column: 18},
											EndPos:   ast.Position{Offset: 126, Line: 8, Column: 19},
										},
									},
									StartPos: ast.Position{Offset: 121, Line: 8, Column: 14},
								},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 138, Line: 9, Column: 10},
					},
				},
			},
			result,
		)
	})

}

func TestParseEntitlementMappingDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) entitlement mapping M { } ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.EntitlementMappingDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "M",
						Pos:        ast.Position{Line: 1, Column: 33, Offset: 33},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 37, Offset: 37},
					},
				},
			},
			result,
		)
	})

	t.Run("mappings", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` access(all) entitlement mapping M { 
			A -> B
			C -> D
		} `)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.EntitlementMappingDeclaration{
					Access:    ast.AccessAll,
					DocString: "",
					Identifier: ast.Identifier{
						Identifier: "M",
						Pos: ast.Position{
							Offset: 33,
							Line:   1,
							Column: 33,
						},
					},
					Elements: []ast.EntitlementMapElement{
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos: ast.Position{
										Offset: 41,
										Line:   2,
										Column: 3,
									},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "B",
									Pos: ast.Position{
										Offset: 46,
										Line:   2,
										Column: 8,
									},
								},
							},
						},
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "C",
									Pos: ast.Position{
										Offset: 51,
										Line:   3,
										Column: 3,
									},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "D",
									Pos: ast.Position{
										Offset: 56,
										Line:   3,
										Column: 8,
									},
								},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 1,
							Line:   1,
							Column: 1,
						},
						EndPos: ast.Position{
							Offset: 60,
							Line:   4,
							Column: 2,
						},
					},
				},
			},
			result,
		)
	})

	t.Run("mappings with includes", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` access(all) entitlement mapping M { 
			include Y
			A -> B
			C -> D
			include X
		} `)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.EntitlementMappingDeclaration{
					Access:    ast.AccessAll,
					DocString: "",
					Identifier: ast.Identifier{
						Identifier: "M",
						Pos: ast.Position{
							Offset: 33,
							Line:   1,
							Column: 33,
						},
					},
					Elements: []ast.EntitlementMapElement{
						&ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "Y",
								Pos: ast.Position{
									Offset: 49,
									Line:   2,
									Column: 11,
								},
							},
						},
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos: ast.Position{
										Offset: 54,
										Line:   3,
										Column: 3,
									},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "B",
									Pos: ast.Position{
										Offset: 59,
										Line:   3,
										Column: 8,
									},
								},
							},
						},
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "C",
									Pos: ast.Position{
										Offset: 64,
										Line:   4,
										Column: 3,
									},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "D",
									Pos: ast.Position{
										Offset: 69,
										Line:   4,
										Column: 8,
									},
								},
							},
						},
						&ast.NominalType{
							Identifier: ast.Identifier{Identifier: "X",
								Pos: ast.Position{
									Offset: 82,
									Line:   5,
									Column: 11,
								},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 1,
							Line:   1,
							Column: 1,
						},
						EndPos: ast.Position{
							Offset: 86,
							Line:   6,
							Column: 2,
						},
					},
				},
			},
			result,
		)
	})

	t.Run("same line mappings", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` access(all) entitlement mapping M { 
			A -> B C -> D
		} `)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.EntitlementMappingDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "M",
						Pos: ast.Position{
							Offset: 33,
							Line:   1,
							Column: 33,
						},
					},
					Elements: []ast.EntitlementMapElement{
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos: ast.Position{
										Offset: 41,
										Line:   2,
										Column: 3,
									},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "B",
									Pos: ast.Position{
										Offset: 46,
										Line:   2,
										Column: 8,
									},
								},
							},
						},
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "C",
									Pos: ast.Position{
										Offset: 48,
										Line:   2,
										Column: 10,
									},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "D",
									Pos: ast.Position{
										Offset: 53,
										Line:   2,
										Column: 15,
									},
								},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 1,
							Line:   1,
							Column: 1,
						},
						EndPos: ast.Position{
							Offset: 57,
							Line:   3,
							Column: 2,
						},
					},
				},
			},
			result,
		)
	})

	t.Run("missing entitlement keyword", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) mapping M {} ")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)
	})

	t.Run("missing mapping keyword", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement M {} ")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: '{'",
					Pos:     ast.Position{Offset: 27, Line: 1, Column: 27},
				},
			},
			errs,
		)
	})

	t.Run("missing body", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement mapping M ")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token '{'",
					Pos:     ast.Position{Offset: 35, Line: 1, Column: 35},
				},
			},
			errs,
		)
	})

	t.Run("missing close brace", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement mapping M {")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token '}'",
					Pos:     ast.Position{Offset: 36, Line: 1, Column: 36},
				},
			},
			errs,
		)
	})

	t.Run("missing open brace", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement mapping M }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token '{'",
					Pos:     ast.Position{Offset: 35, Line: 1, Column: 35},
				},
			},
			errs,
		)
	})

	t.Run("missing identifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement mapping {}")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected identifier following entitlement mapping declaration, got '{'",
					Pos:     ast.Position{Offset: 33, Line: 1, Column: 33},
				},
			},
			errs,
		)
	})

	t.Run("non-nominal mapping first", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(` access(all) entitlement mapping M { 
			&A -> B
		} `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected nominal type, got &A",
					Pos:     ast.Position{Offset: 43, Line: 2, Column: 5},
				},
			},
			errs,
		)
	})

	t.Run("non-nominal mapping second", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(` access(all) entitlement mapping M { 
			A -> [B]
		} `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected nominal type, got [B]",
					Pos:     ast.Position{Offset: 49, Line: 2, Column: 11},
				},
			},
			errs,
		)
	})

	t.Run("missing arrow", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(` access(all) entitlement mapping M { 
			A B
		} `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token '->'",
					Pos:     ast.Position{Offset: 43, Line: 2, Column: 5},
				},
			},
			errs,
		)
	})

	t.Run("wrong mapping separator", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(` access(all) entitlement mapping M { 
			A - B
		} `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token '->'",
					Pos:     ast.Position{Offset: 43, Line: 2, Column: 5},
				},
			},
			errs,
		)
	})

	t.Run("non-nominal include", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(` access(all) entitlement mapping M { 
			include &A
		} `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected nominal type, got &A",
					Pos:     ast.Position{Offset: 51, Line: 2, Column: 13},
				},
			},
			errs,
		)
	})

	t.Run("include with arrow", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(` access(all) entitlement mapping M { 
			include -> B
		} `)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token in type: '->'",
					Pos:     ast.Position{Offset: 51, Line: 2, Column: 13},
				},
			},
			errs,
		)
	})
}

func TestParseInvalidSpecialFunctionReturnTypeAnnotation(t *testing.T) {

	t.Parallel()

	_, errs := testParseDeclarations(`
      struct Test {

          init(): Int
      }
	`)
	utils.AssertEqualWithDiff(t,
		[]error{
			&SyntaxError{
				Message: "invalid return type for initializer",
				Pos:     ast.Position{Offset: 40, Line: 4, Column: 18},
			},
		},
		errs,
	)
}

func TestSoftKeywordsInFunctionDeclaration(t *testing.T) {
	t.Parallel()

	posFromName := func(name string, offset int) ast.Position {
		offsetPos := len(name) + offset
		return ast.Position{
			Line:   1,
			Offset: offsetPos,
			Column: offsetPos,
		}
	}

	testSoftKeyword := func(name string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`fun %s() {}`, name)

			result, errs := testParseDeclarations(code)
			require.Empty(t, errs)

			expected := []ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: name,
						Pos:        ast.Position{Offset: 4, Line: 1, Column: 4},
					},
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: posFromName(name, 7),
								EndPos:   posFromName(name, 8),
							},
						},
					},
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: posFromName(name, 4),
							EndPos:   posFromName(name, 5),
						},
					},
				},
			}
			utils.AssertEqualWithDiff(t, expected, result)

		})
	}

	for _, keyword := range softKeywords {
		testSoftKeyword(keyword)
	}
}

func TestParseDeprecatedAccessModifiers(t *testing.T) {

	t.Parallel()

	t.Run("pub", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" pub fun foo ( ) { }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxErrorWithSuggestedReplacement{
					Message: "`pub` is no longer a valid access keyword",
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
					},
					SuggestedFix: "`access(all)`",
				},
			},
			errs,
		)

	})

	t.Run("priv", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" priv fun foo ( ) { }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxErrorWithSuggestedReplacement{
					Message: "`priv` is no longer a valid access keyword",
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
					},
					SuggestedFix: "`access(self)`",
				},
			},
			errs,
		)

	})

	t.Run("pub(set)", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" pub(set) fun foo ( ) { }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxErrorWithSuggestedReplacement{
					Message: "`pub` is no longer a valid access keyword",
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
					},
					SuggestedFix: "`access(all)`",
				},
			},
			errs,
		)

	})
}
