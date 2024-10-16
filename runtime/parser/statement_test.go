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

package parser

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestParseReplInput(t *testing.T) {

	t.Parallel()

	actual, errs := testParseStatements(`
        struct X {}; let x = X(); x
    `)

	var err error
	if len(errs) > 0 {
		err = Error{
			Errors: errs,
		}
	}

	require.NoError(t, err)
	require.IsType(t, []ast.Statement{}, actual)

	require.Len(t, actual, 3)
	assert.IsType(t, &ast.CompositeDeclaration{}, actual[0])
	assert.IsType(t, &ast.VariableDeclaration{}, actual[1])
	assert.IsType(t, &ast.ExpressionStatement{}, actual[2])
}

func TestParseReturnStatement(t *testing.T) {

	t.Parallel()

	t.Run("no expression", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("return")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ReturnStatement{
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
					},
				},
			},
			result,
		)
	})

	t.Run("expression on same line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("return 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ReturnStatement{
					Expression: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
					},
				},
			},
			result,
		)
	})

	t.Run("expression on next line, no semicolon", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("return \n1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ReturnStatement{
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
					},
				},
				&ast.ExpressionStatement{
					Expression: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 8},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 8},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("expression on next line, semicolon", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("return ;\n1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ReturnStatement{
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
					},
				},
				&ast.ExpressionStatement{
					Expression: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 9},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 9},
						},
					},
				},
			},
			result,
		)
	})
}

func TestParseIfStatement(t *testing.T) {

	t.Parallel()

	t.Run("only empty then", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("if true { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.IfStatement{
					Test: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Then: &ast.Block{
						Statements: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("only then, two statements on one line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("if true { 1 ; 2 }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.IfStatement{
					Test: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Then: &ast.Block{
						Statements: []ast.Statement{
							&ast.ExpressionStatement{
								Expression: &ast.IntegerExpression{
									PositiveLiteral: []byte("1"),
									Value:           big.NewInt(1),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
										EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
									},
								},
							},
							&ast.ExpressionStatement{
								Expression: &ast.IntegerExpression{
									PositiveLiteral: []byte("2"),
									Value:           big.NewInt(2),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
										EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 16, Offset: 16},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("only then, two statements on multiple lines", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("if true { 1 \n 2 }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.IfStatement{
					Test: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Then: &ast.Block{
						Statements: []ast.Statement{
							&ast.ExpressionStatement{
								Expression: &ast.IntegerExpression{
									PositiveLiteral: []byte("1"),
									Value:           big.NewInt(1),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
										EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
									},
								},
							},
							&ast.ExpressionStatement{
								Expression: &ast.IntegerExpression{
									PositiveLiteral: []byte("2"),
									Value:           big.NewInt(2),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Line: 2, Column: 1, Offset: 14},
										EndPos:   ast.Position{Line: 2, Column: 1, Offset: 14},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 2, Column: 3, Offset: 16},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("with else", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("if true { 1 } else { 2 }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.IfStatement{
					Test: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Then: &ast.Block{
						Statements: []ast.Statement{
							&ast.ExpressionStatement{
								Expression: &ast.IntegerExpression{
									PositiveLiteral: []byte("1"),
									Value:           big.NewInt(1),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
										EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
						},
					},
					Else: &ast.Block{
						Statements: []ast.Statement{
							&ast.ExpressionStatement{
								Expression: &ast.IntegerExpression{
									PositiveLiteral: []byte("2"),
									Value:           big.NewInt(2),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 21, Offset: 21},
										EndPos:   ast.Position{Line: 1, Column: 21, Offset: 21},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 19, Offset: 19},
							EndPos:   ast.Position{Line: 1, Column: 23, Offset: 23},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("with else if and else, no space", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("if true{1}else if true {2} else{3}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.IfStatement{
					Test: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Then: &ast.Block{
						Statements: []ast.Statement{
							&ast.ExpressionStatement{
								Expression: &ast.IntegerExpression{
									PositiveLiteral: []byte("1"),
									Value:           big.NewInt(1),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
										EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Else: &ast.Block{
						Statements: []ast.Statement{
							&ast.IfStatement{
								Test: &ast.BoolExpression{
									Value: true,
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 18, Offset: 18},
										EndPos:   ast.Position{Line: 1, Column: 21, Offset: 21},
									},
								},
								Then: &ast.Block{
									Statements: []ast.Statement{
										&ast.ExpressionStatement{
											Expression: &ast.IntegerExpression{
												PositiveLiteral: []byte("2"),
												Value:           big.NewInt(2),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Line: 1, Column: 24, Offset: 24},
													EndPos:   ast.Position{Line: 1, Column: 24, Offset: 24},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 23, Offset: 23},
										EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
									},
								},
								Else: &ast.Block{
									Statements: []ast.Statement{
										&ast.ExpressionStatement{
											Expression: &ast.IntegerExpression{
												PositiveLiteral: []byte("3"),
												Value:           big.NewInt(3),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Line: 1, Column: 32, Offset: 32},
													EndPos:   ast.Position{Line: 1, Column: 32, Offset: 32},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 31, Offset: 31},
										EndPos:   ast.Position{Line: 1, Column: 33, Offset: 33},
									},
								},
								StartPos: ast.Position{Line: 1, Column: 15, Offset: 15},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 15, Offset: 15},
							EndPos:   ast.Position{Line: 1, Column: 33, Offset: 33},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("if-var", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("if var x = 1 { }")
		require.Empty(t, errs)

		expected := &ast.IfStatement{
			Test: &ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: false,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
				},
				Value: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
						EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
					},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Line: 1, Column: 9, Offset: 9},
				},
				StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
			},
			Then: &ast.Block{
				Statements: nil,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 13, Offset: 13},
					EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
				},
			},
			StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
		}

		expected.Test.(*ast.VariableDeclaration).ParentIfStatement = expected

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				expected,
			},
			result,
		)
	})

	t.Run("if-let", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("if let x = 1 { }")
		require.Empty(t, errs)

		expected := &ast.IfStatement{
			Test: &ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
				},
				Value: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
						EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
					},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Line: 1, Column: 9, Offset: 9},
				},
				StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
			},
			Then: &ast.Block{
				Statements: nil,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 13, Offset: 13},
					EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
				},
			},
			StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
		}

		expected.Test.(*ast.VariableDeclaration).ParentIfStatement = expected

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				expected,
			},
			result,
		)
	})

}

func TestParseWhileStatement(t *testing.T) {

	t.Parallel()

	t.Run("empty block", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("while true { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.WhileStatement{
					Test: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Block: &ast.Block{
						Statements: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
							EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})
}

func TestParseAssignmentStatement(t *testing.T) {

	t.Parallel()

	t.Run("copy, no space", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("x=1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.AssignmentStatement{
					Target: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "x",
							Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       ast.Position{Line: 1, Column: 1, Offset: 1},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("copy, spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements(" x = 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.AssignmentStatement{
					Target: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "x",
							Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       ast.Position{Line: 1, Column: 3, Offset: 3},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("move", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements(" x <- 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.AssignmentStatement{
					Target: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "x",
							Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationMove,
						Pos:       ast.Position{Line: 1, Column: 3, Offset: 3},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("force move", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements(" x <-! 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.AssignmentStatement{
					Target: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "x",
							Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationMoveForced,
						Pos:       ast.Position{Line: 1, Column: 3, Offset: 3},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
				},
			},
			result,
		)
	})
}

func TestParseSwapStatement(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements(" x <-> y")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.SwapStatement{
					Left: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "x",
							Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Right: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "y",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
				},
			},
			result,
		)
	})
}

func TestParseForStatement(t *testing.T) {

	t.Parallel()

	t.Run("empty block", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("for x in y { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ForStatement{
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
					Block: &ast.Block{
						Statements: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
							EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})
}

func TestParseForStatementIndexBinding(t *testing.T) {

	t.Parallel()

	t.Run("empty block", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("for i, x in y { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ForStatement{
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
					},
					Index: &ast.Identifier{
						Identifier: "i",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "y",
							Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
						},
					},
					Block: &ast.Block{
						Statements: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							EndPos:   ast.Position{Line: 1, Column: 16, Offset: 16},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("no comma", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseStatements("for i x in y { }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected keyword \"in\", got identifier",
					Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
				},
				&SyntaxError{
					Message: "expected token '{'",
					Pos:     ast.Position{Offset: 11, Line: 1, Column: 11},
				},
			},
			errs,
		)
	})

	t.Run("no identifiers", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseStatements("for in y { }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected identifier, got keyword \"in\"",
					Pos:     ast.Position{Offset: 4, Line: 1, Column: 4},
				},
				&SyntaxError{
					Message: "expected token identifier",
					Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
				},
			},
			errs,
		)
	})
}

func TestParseEmit(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("emit T()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.EmitStatement{
					InvocationExpression: &ast.InvocationExpression{
						InvokedExpression: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "T",
								Pos:        ast.Position{Line: 1, Column: 5, Offset: 5},
							},
						},
						ArgumentsStartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
						EndPos:            ast.Position{Line: 1, Column: 7, Offset: 7},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})
}

func TestParseFunctionStatementOrExpression(t *testing.T) {

	t.Parallel()

	t.Run("function declaration with name", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("fun foo() {}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
								EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("function expression with purity and without name", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("view fun () {}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.FunctionExpression{
						Purity: ast.FunctionPurityView,
						ParameterList: &ast.ParameterList{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
								EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
							},
						},
						FunctionBlock: &ast.FunctionBlock{
							Block: &ast.Block{
								Range: ast.Range{
									StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
									EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
								},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
			},
			result,
		)
	})

	t.Run("function declaration with purity and name", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("view fun foo() {}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.FunctionDeclaration{
					Purity: ast.FunctionPurityView,
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
					},
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
							EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 15, Offset: 15},
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

	t.Run("function expression without name", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("fun () {}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.FunctionExpression{
						ParameterList: &ast.ParameterList{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
								EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
							},
						},
						FunctionBlock: &ast.FunctionBlock{
							Block: &ast.Block{
								Range: ast.Range{
									StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
									EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
								},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
			},
			result,
		)
	})

	t.Run("function expression with keyword as name", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseStatements("fun continue() {}")

		require.Empty(t, result)

		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Message: "expected identifier after start of function declaration, got keyword continue",
				Pos:     ast.Position{Line: 1, Column: 4, Offset: 4},
			},
		}, errs)
	})

	t.Run("function expression with purity, and keyword as name", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseStatements("view fun break() {}")

		require.Empty(t, result)

		utils.AssertEqualWithDiff(t, []error{
			&SyntaxError{
				Message: "expected identifier after start of function declaration, got keyword break",
				Pos:     ast.Position{Line: 1, Column: 9, Offset: 9},
			},
		}, errs)
	})
}

func TestParseViewNonFunction(t *testing.T) {
	t.Parallel()

	_, errs := testParseStatements("view return 3")
	utils.AssertEqualWithDiff(t,
		[]error{
			&SyntaxError{
				Message: "statements on the same line must be separated with a semicolon",
				Pos:     ast.Position{Offset: 5, Line: 1, Column: 5},
			},
		},
		errs,
	)
}

func TestParseStatements(t *testing.T) {

	t.Parallel()

	t.Run("binary expression with less operator", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("a + b < c\nd")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.BinaryExpression{
						Operation: ast.OperationLess,
						Left: &ast.BinaryExpression{
							Operation: ast.OperationPlus,
							Left: &ast.IdentifierExpression{
								Identifier: ast.Identifier{
									Identifier: "a",
									Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
								},
							},
							Right: &ast.IdentifierExpression{
								Identifier: ast.Identifier{
									Identifier: "b",
									Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
								},
							},
						},
						Right: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "c",
								Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
							},
						},
					},
				},
				&ast.ExpressionStatement{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "d",
							Pos:        ast.Position{Line: 2, Column: 0, Offset: 10},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("multiple statements on same line without semicolon", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements(`assert true`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "statements on the same line must be separated with a semicolon",
					Pos:     ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "assert",
							Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
				},
				&ast.ExpressionStatement{
					Expression: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
				},
			},
			result,
		)
	})
}

func TestParseRemoveAttachmentStatement(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("remove A from b")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.RemoveStatement{
					Attachment: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "A",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Value: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("namespaced attachment", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("remove Foo.E from b")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.RemoveStatement{
					Attachment: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Foo",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
						NestedIdentifiers: []ast.Identifier{
							{
								Identifier: "E",
								Pos:        ast.Position{Line: 1, Column: 11, Offset: 11},
							},
						},
					},
					Value: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("no from", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseStatements("remove A")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected from keyword, got EOF",
					Pos:     ast.Position{Offset: 8, Line: 1, Column: 8},
				},
				&SyntaxError{
					Message: "unexpected end of program",
					Pos:     ast.Position{Offset: 8, Line: 1, Column: 8},
				},
			},
			errs,
		)
	})

	t.Run("no target", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseStatements("remove A from")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected end of program",
					Pos:     ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)
	})

	t.Run("no nominal type", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseStatements("remove [A] from e")

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected attachment nominal type, got [A]",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
		)
	})

	t.Run("complex source", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("remove A from foo()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.RemoveStatement{
					Attachment: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "A",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Value: &ast.InvocationExpression{
						InvokedExpression: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
							},
						},
						ArgumentsStartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
						EndPos:            ast.Position{Line: 1, Column: 18, Offset: 18},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})
}

func TestParseSwitchStatement(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("switch true { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.SwitchStatement{
					Expression: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
					Cases: nil,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
					},
				},
			},
			result,
		)
	})

	t.Run("two cases", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("switch x { case 1 :\n a\nb default : c\nd  }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.SwitchStatement{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "x",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Cases: []*ast.SwitchCase{
						{
							Expression: &ast.IntegerExpression{
								PositiveLiteral: []byte("1"),
								Value:           big.NewInt(1),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
									EndPos:   ast.Position{Line: 1, Column: 16, Offset: 16},
								},
							},
							Statements: []ast.Statement{
								&ast.ExpressionStatement{
									Expression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "a",
											Pos:        ast.Position{Line: 2, Column: 1, Offset: 21},
										},
									},
								},
								&ast.ExpressionStatement{
									Expression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "b",
											Pos:        ast.Position{Line: 3, Column: 0, Offset: 23},
										},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
								EndPos:   ast.Position{Line: 3, Column: 0, Offset: 23},
							},
						},
						{
							Statements: []ast.Statement{
								&ast.ExpressionStatement{
									Expression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "c",
											Pos:        ast.Position{Line: 3, Column: 12, Offset: 35},
										},
									},
								},
								&ast.ExpressionStatement{
									Expression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "d",
											Pos:        ast.Position{Line: 4, Column: 0, Offset: 37},
										},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Line: 3, Column: 2, Offset: 25},
								EndPos:   ast.Position{Line: 4, Column: 0, Offset: 37},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 4, Column: 3, Offset: 40},
					},
				},
			},
			result,
		)
	})

	t.Run("Invalid identifiers in switch cases", func(t *testing.T) {
		code := "switch 1 {AAAAA: break; case 3: break; default: break}"
		_, errs := testParseStatements(code)
		utils.AssertEqualWithDiff(t,
			`unexpected token: got identifier, expected "case" or "default"`,
			errs[0].Error(),
		)
	})
}

func TestParseIfStatementInFunctionDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
	    fun test() {
            if true {
                return
            } else if false {
                false
                1
            } else {
                2
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
							&ast.IfStatement{
								Test: &ast.BoolExpression{
									Value: true,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 34, Line: 3, Column: 15},
										EndPos:   ast.Position{Offset: 37, Line: 3, Column: 18},
									},
								},
								Then: &ast.Block{
									Statements: []ast.Statement{
										&ast.ReturnStatement{
											Expression: nil,
											Range: ast.Range{
												StartPos: ast.Position{Offset: 57, Line: 4, Column: 16},
												EndPos:   ast.Position{Offset: 62, Line: 4, Column: 21},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 39, Line: 3, Column: 20},
										EndPos:   ast.Position{Offset: 76, Line: 5, Column: 12},
									},
								},
								Else: &ast.Block{
									Statements: []ast.Statement{
										&ast.IfStatement{
											Test: &ast.BoolExpression{
												Value: false,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 86, Line: 5, Column: 22},
													EndPos:   ast.Position{Offset: 90, Line: 5, Column: 26},
												},
											},
											Then: &ast.Block{
												Statements: []ast.Statement{
													&ast.ExpressionStatement{
														Expression: &ast.BoolExpression{
															Value: false,
															Range: ast.Range{
																StartPos: ast.Position{Offset: 110, Line: 6, Column: 16},
																EndPos:   ast.Position{Offset: 114, Line: 6, Column: 20},
															},
														},
													},
													&ast.ExpressionStatement{
														Expression: &ast.IntegerExpression{
															PositiveLiteral: []byte("1"),
															Value:           big.NewInt(1),
															Base:            10,
															Range: ast.Range{
																StartPos: ast.Position{Offset: 132, Line: 7, Column: 16},
																EndPos:   ast.Position{Offset: 132, Line: 7, Column: 16},
															},
														},
													},
												},
												Range: ast.Range{
													StartPos: ast.Position{Offset: 92, Line: 5, Column: 28},
													EndPos:   ast.Position{Offset: 146, Line: 8, Column: 12},
												},
											},
											Else: &ast.Block{
												Statements: []ast.Statement{
													&ast.ExpressionStatement{
														Expression: &ast.IntegerExpression{
															PositiveLiteral: []byte("2"),
															Value:           big.NewInt(2),
															Base:            10,
															Range: ast.Range{
																StartPos: ast.Position{Offset: 171, Line: 9, Column: 16},
																EndPos:   ast.Position{Offset: 171, Line: 9, Column: 16},
															},
														},
													},
												},
												Range: ast.Range{
													StartPos: ast.Position{Offset: 153, Line: 8, Column: 19},
													EndPos:   ast.Position{Offset: 185, Line: 10, Column: 12},
												},
											},
											StartPos: ast.Position{Offset: 83, Line: 5, Column: 19},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 83, Line: 5, Column: 19},
										EndPos:   ast.Position{Offset: 185, Line: 10, Column: 12},
									},
								},
								StartPos: ast.Position{Offset: 31, Line: 3, Column: 12},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 195, Line: 11, Column: 8},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseIfStatementWithVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
	    fun test() {
            if var y = x {
                1
            } else {
                2
            }
        }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	ifStatement := &ast.IfStatement{
		Then: &ast.Block{
			Statements: []ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 62, Line: 4, Column: 16},
							EndPos:   ast.Position{Offset: 62, Line: 4, Column: 16},
						},
					},
				},
			},
			Range: ast.Range{
				StartPos: ast.Position{Offset: 44, Line: 3, Column: 25},
				EndPos:   ast.Position{Offset: 76, Line: 5, Column: 12},
			},
		},
		Else: &ast.Block{
			Statements: []ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 101, Line: 6, Column: 16},
							EndPos:   ast.Position{Offset: 101, Line: 6, Column: 16},
						},
					},
				},
			},
			Range: ast.Range{
				StartPos: ast.Position{Offset: 83, Line: 5, Column: 19},
				EndPos:   ast.Position{Offset: 115, Line: 7, Column: 12},
			},
		},
		StartPos: ast.Position{Offset: 31, Line: 3, Column: 12},
	}

	ifTestVariableDeclaration := &ast.VariableDeclaration{
		Access:     ast.AccessNotSpecified,
		IsConstant: false,
		Identifier: ast.Identifier{
			Identifier: "y",
			Pos:        ast.Position{Offset: 38, Line: 3, Column: 19},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 40, Line: 3, Column: 21},
		},
		Value: &ast.IdentifierExpression{
			Identifier: ast.Identifier{
				Identifier: "x",
				Pos:        ast.Position{Offset: 42, Line: 3, Column: 23},
			},
		},
		StartPos:          ast.Position{Offset: 34, Line: 3, Column: 15},
		ParentIfStatement: ifStatement,
	}

	ifStatement.Test = ifTestVariableDeclaration

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
							ifStatement,
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 125, Line: 8, Column: 8},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseIfStatementNoElse(t *testing.T) {

	t.Parallel()

	const code = `
	    fun test() {
            if true {
                return
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
							&ast.IfStatement{
								Test: &ast.BoolExpression{
									Value: true,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 34, Line: 3, Column: 15},
										EndPos:   ast.Position{Offset: 37, Line: 3, Column: 18},
									},
								},
								Then: &ast.Block{
									Statements: []ast.Statement{
										&ast.ReturnStatement{
											Expression: nil,
											Range: ast.Range{
												StartPos: ast.Position{Offset: 57, Line: 4, Column: 16},
												EndPos:   ast.Position{Offset: 62, Line: 4, Column: 21},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 39, Line: 3, Column: 20},
										EndPos:   ast.Position{Offset: 76, Line: 5, Column: 12},
									},
								},
								StartPos: ast.Position{Offset: 31, Line: 3, Column: 12},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 86, Line: 6, Column: 8},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseWhileStatementInFunctionDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
	    fun test() {
            while true {
              return
              break
              continue
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
							&ast.WhileStatement{
								Test: &ast.BoolExpression{
									Value: true,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 37, Line: 3, Column: 18},
										EndPos:   ast.Position{Offset: 40, Line: 3, Column: 21},
									},
								},
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.ReturnStatement{
											Expression: nil,
											Range: ast.Range{
												StartPos: ast.Position{Offset: 58, Line: 4, Column: 14},
												EndPos:   ast.Position{Offset: 63, Line: 4, Column: 19},
											},
										},
										&ast.BreakStatement{
											Range: ast.Range{
												StartPos: ast.Position{Offset: 79, Line: 5, Column: 14},
												EndPos:   ast.Position{Offset: 83, Line: 5, Column: 18},
											},
										},
										&ast.ContinueStatement{
											Range: ast.Range{
												StartPos: ast.Position{Offset: 99, Line: 6, Column: 14},
												EndPos:   ast.Position{Offset: 106, Line: 6, Column: 21},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 42, Line: 3, Column: 23},
										EndPos:   ast.Position{Offset: 120, Line: 7, Column: 12},
									},
								},
								StartPos: ast.Position{Offset: 31, Line: 3, Column: 12},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 130, Line: 8, Column: 8},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseForStatementInFunctionDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
	    fun test() {
            for x in xs {}
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
							&ast.ForStatement{
								Identifier: ast.Identifier{
									Identifier: "x",
									Pos:        ast.Position{Offset: 35, Line: 3, Column: 16},
								},
								Value: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "xs",
										Pos:        ast.Position{Offset: 40, Line: 3, Column: 21},
									},
								},
								Block: &ast.Block{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 43, Line: 3, Column: 24},
										EndPos:   ast.Position{Offset: 44, Line: 3, Column: 25},
									},
								},
								StartPos: ast.Position{Offset: 31, Line: 3, Column: 12},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 54, Line: 4, Column: 8},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseAssignment(t *testing.T) {

	t.Parallel()

	const code = `
	    fun test() {
            a = 1
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
							&ast.AssignmentStatement{
								Target: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "a",
										Pos:        ast.Position{Offset: 31, Line: 3, Column: 12},
									},
								},
								Transfer: &ast.Transfer{
									Operation: ast.TransferOperationCopy,
									Pos:       ast.Position{Offset: 33, Line: 3, Column: 14},
								},
								Value: &ast.IntegerExpression{
									PositiveLiteral: []byte("1"),
									Value:           big.NewInt(1),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 35, Line: 3, Column: 16},
										EndPos:   ast.Position{Offset: 35, Line: 3, Column: 16},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 45, Line: 4, Column: 8},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseAccessAssignment(t *testing.T) {

	t.Parallel()

	const code = `
	    fun test() {
            x.foo.bar[0][1].baz = 1
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
							&ast.AssignmentStatement{
								Target: &ast.MemberExpression{
									Expression: &ast.IndexExpression{
										TargetExpression: &ast.IndexExpression{
											TargetExpression: &ast.MemberExpression{
												Expression: &ast.MemberExpression{
													Expression: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "x",
															Pos:        ast.Position{Offset: 31, Line: 3, Column: 12},
														},
													},
													AccessPos: ast.Position{Offset: 32, Line: 3, Column: 13},
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos:        ast.Position{Offset: 33, Line: 3, Column: 14},
													},
												},
												AccessPos: ast.Position{Offset: 36, Line: 3, Column: 17},
												Identifier: ast.Identifier{
													Identifier: "bar",
													Pos:        ast.Position{Offset: 37, Line: 3, Column: 18},
												},
											},
											IndexingExpression: &ast.IntegerExpression{
												PositiveLiteral: []byte("0"),
												Value:           new(big.Int),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 41, Line: 3, Column: 22},
													EndPos:   ast.Position{Offset: 41, Line: 3, Column: 22},
												},
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 31, Line: 3, Column: 12},
												EndPos:   ast.Position{Offset: 42, Line: 3, Column: 23},
											},
										},
										IndexingExpression: &ast.IntegerExpression{
											PositiveLiteral: []byte("1"),
											Value:           big.NewInt(1),
											Base:            10,
											Range: ast.Range{
												StartPos: ast.Position{Offset: 44, Line: 3, Column: 25},
												EndPos:   ast.Position{Offset: 44, Line: 3, Column: 25},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 31, Line: 3, Column: 12},
											EndPos:   ast.Position{Offset: 45, Line: 3, Column: 26},
										},
									},
									AccessPos: ast.Position{Offset: 46, Line: 3, Column: 27},
									Identifier: ast.Identifier{
										Identifier: "baz",
										Pos:        ast.Position{Offset: 47, Line: 3, Column: 28},
									},
								},
								Transfer: &ast.Transfer{
									Operation: ast.TransferOperationCopy,
									Pos:       ast.Position{Offset: 51, Line: 3, Column: 32},
								},
								Value: &ast.IntegerExpression{
									PositiveLiteral: []byte("1"),
									Value:           big.NewInt(1),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 53, Line: 3, Column: 34},
										EndPos:   ast.Position{Offset: 53, Line: 3, Column: 34},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 63, Line: 4, Column: 8},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseExpressionStatementWithAccess(t *testing.T) {

	t.Parallel()

	const code = `
	    fun test() { x.foo.bar[0][1].baz }
	`
	result, errs := testParseProgram(code)
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
							&ast.ExpressionStatement{
								Expression: &ast.MemberExpression{
									Expression: &ast.IndexExpression{
										TargetExpression: &ast.IndexExpression{
											TargetExpression: &ast.MemberExpression{
												Expression: &ast.MemberExpression{
													Expression: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "x",
															Pos:        ast.Position{Offset: 19, Line: 2, Column: 18},
														},
													},
													AccessPos: ast.Position{Offset: 20, Line: 2, Column: 19},
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos:        ast.Position{Offset: 21, Line: 2, Column: 20},
													},
												},
												AccessPos: ast.Position{Offset: 24, Line: 2, Column: 23},
												Identifier: ast.Identifier{
													Identifier: "bar",
													Pos:        ast.Position{Offset: 25, Line: 2, Column: 24},
												},
											},
											IndexingExpression: &ast.IntegerExpression{
												PositiveLiteral: []byte("0"),
												Value:           new(big.Int),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 29, Line: 2, Column: 28},
													EndPos:   ast.Position{Offset: 29, Line: 2, Column: 28},
												},
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 19, Line: 2, Column: 18},
												EndPos:   ast.Position{Offset: 30, Line: 2, Column: 29},
											},
										},
										IndexingExpression: &ast.IntegerExpression{
											PositiveLiteral: []byte("1"),
											Value:           big.NewInt(1),
											Base:            10,
											Range: ast.Range{
												StartPos: ast.Position{Offset: 32, Line: 2, Column: 31},
												EndPos:   ast.Position{Offset: 32, Line: 2, Column: 31},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 19, Line: 2, Column: 18},
											EndPos:   ast.Position{Offset: 33, Line: 2, Column: 32},
										},
									},
									AccessPos: ast.Position{Offset: 34, Line: 2, Column: 33},
									Identifier: ast.Identifier{
										Identifier: "baz",
										Pos:        ast.Position{Offset: 35, Line: 2, Column: 34},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 39, Line: 2, Column: 38},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseMoveStatement(t *testing.T) {

	t.Parallel()

	const code = `
        fun test() {
            x <- y
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
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.AssignmentStatement{
								Target: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "x",
										Pos:        ast.Position{Offset: 34, Line: 3, Column: 12},
									},
								},
								Transfer: &ast.Transfer{
									Operation: ast.TransferOperationMove,
									Pos:       ast.Position{Offset: 36, Line: 3, Column: 14},
								},
								Value: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "y",
										Pos:        ast.Position{Offset: 39, Line: 3, Column: 17},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
							EndPos:   ast.Position{Offset: 49, Line: 4, Column: 8},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseFunctionExpressionStatementAfterVariableDeclarationWithCreateExpression(t *testing.T) {

	t.Parallel()

	const code = `
      fun test() {
          let r <- create R()
          (fun () {})()
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
							&ast.VariableDeclaration{
								Access:     ast.AccessNotSpecified,
								IsConstant: true,
								Identifier: ast.Identifier{
									Identifier: "r",
									Pos:        ast.Position{Offset: 34, Line: 3, Column: 14},
								},
								TypeAnnotation: nil,
								Value: &ast.CreateExpression{
									InvocationExpression: &ast.InvocationExpression{
										InvokedExpression: &ast.IdentifierExpression{
											Identifier: ast.Identifier{
												Identifier: "R",
												Pos:        ast.Position{Offset: 46, Line: 3, Column: 26},
											},
										},
										Arguments:         nil,
										ArgumentsStartPos: ast.Position{Offset: 47, Line: 3, Column: 27},
										EndPos:            ast.Position{Offset: 48, Line: 3, Column: 28},
									},
									StartPos: ast.Position{Offset: 39, Line: 3, Column: 19},
								},
								Transfer: &ast.Transfer{
									Operation: ast.TransferOperationMove,
									Pos:       ast.Position{Offset: 36, Line: 3, Column: 16},
								},
								StartPos: ast.Position{Offset: 30, Line: 3, Column: 10},
							},
							&ast.ExpressionStatement{
								Expression: &ast.InvocationExpression{
									InvokedExpression: &ast.FunctionExpression{
										ParameterList: &ast.ParameterList{
											Range: ast.Range{
												StartPos: ast.Position{Offset: 65, Line: 4, Column: 15},
												EndPos:   ast.Position{Offset: 66, Line: 4, Column: 16},
											},
										},
										FunctionBlock: &ast.FunctionBlock{
											Block: &ast.Block{
												Statements: nil,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 68, Line: 4, Column: 18},
													EndPos:   ast.Position{Offset: 69, Line: 4, Column: 19},
												},
											},
											PreConditions:  nil,
											PostConditions: nil,
										},
										StartPos: ast.Position{Offset: 61, Line: 4, Column: 11},
									},
									Arguments:         nil,
									ArgumentsStartPos: ast.Position{Offset: 71, Line: 4, Column: 21},
									EndPos:            ast.Position{Offset: 72, Line: 4, Column: 22},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
							EndPos:   ast.Position{Offset: 80, Line: 5, Column: 6},
						},
					},
					PreConditions:  nil,
					PostConditions: nil,
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

// TestParseExpressionStatementAfterReturnStatement tests that a return statement
// does *not* consume an expression from the next statement as the return value
func TestParseExpressionStatementAfterReturnStatement(t *testing.T) {

	t.Parallel()

	const code = `
      fun test() {
          return
          destroy x
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
							&ast.ReturnStatement{
								Expression: nil,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 30, Line: 3, Column: 10},
									EndPos:   ast.Position{Offset: 35, Line: 3, Column: 15},
								},
							},
							&ast.ExpressionStatement{
								Expression: &ast.DestroyExpression{
									Expression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "x",
											Pos:        ast.Position{Offset: 55, Line: 4, Column: 18},
										},
									},
									StartPos: ast.Position{Offset: 47, Line: 4, Column: 10},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
							EndPos:   ast.Position{Offset: 63, Line: 5, Column: 6},
						},
					},
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

func TestParseSwapStatementInFunctionDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
      fun test() {
          foo[0] <-> bar.baz
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
							&ast.SwapStatement{
								Left: &ast.IndexExpression{
									TargetExpression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "foo",
											Pos:        ast.Position{Offset: 30, Line: 3, Column: 10},
										},
									},
									IndexingExpression: &ast.IntegerExpression{
										PositiveLiteral: []byte("0"),
										Value:           new(big.Int),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 34, Line: 3, Column: 14},
											EndPos:   ast.Position{Offset: 34, Line: 3, Column: 14},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 30, Line: 3, Column: 10},
										EndPos:   ast.Position{Offset: 35, Line: 3, Column: 15},
									},
								},
								Right: &ast.MemberExpression{
									Expression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "bar",
											Pos:        ast.Position{Offset: 41, Line: 3, Column: 21},
										},
									},
									AccessPos: ast.Position{Offset: 44, Line: 3, Column: 24},
									Identifier: ast.Identifier{
										Identifier: "baz",
										Pos:        ast.Position{Offset: 45, Line: 3, Column: 25},
									},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
							EndPos:   ast.Position{Offset: 55, Line: 4, Column: 6},
						},
					},
					PreConditions:  nil,
					PostConditions: nil,
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

func TestParseReferenceExpressionStatement(t *testing.T) {

	t.Parallel()

	result, errs := testParseStatements(
		`
          let x = &1 as &Int
          (x!)
	    `,
	)
	require.Empty(t, errs)

	castingExpression := &ast.CastingExpression{
		Expression: &ast.ReferenceExpression{
			Expression: &ast.IntegerExpression{
				PositiveLiteral: []byte("1"),
				Value:           big.NewInt(1),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
					EndPos:   ast.Position{Offset: 20, Line: 2, Column: 19},
				},
			},
			StartPos: ast.Position{Offset: 19, Line: 2, Column: 18},
		},
		Operation: ast.OperationCast,
		TypeAnnotation: &ast.TypeAnnotation{
			Type: &ast.ReferenceType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Offset: 26, Line: 2, Column: 25},
					},
				},
				StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
			},
			StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
		},
	}

	expectedVariableDeclaration := &ast.VariableDeclaration{
		Access:     ast.AccessNotSpecified,
		IsConstant: true,
		Identifier: ast.Identifier{
			Identifier: "x",
			Pos:        ast.Position{Line: 2, Column: 14, Offset: 15},
		},
		StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
		Value:    castingExpression,
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 17, Line: 2, Column: 16},
		},
	}

	castingExpression.ParentVariableDeclaration = expectedVariableDeclaration

	utils.AssertEqualWithDiff(t,
		[]ast.Statement{
			expectedVariableDeclaration,
			&ast.ExpressionStatement{
				Expression: &ast.ForceExpression{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "x",
							Pos:        ast.Position{Offset: 41, Line: 3, Column: 11},
						},
					},
					EndPos: ast.Position{Offset: 42, Line: 3, Column: 12},
				},
			},
		},
		result,
	)
}

func TestSoftKeywordsInStatement(t *testing.T) {
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

			code := fmt.Sprintf(`%s = 42`, name)

			result, errs := testParseStatements(code)
			require.Empty(t, errs)

			expected := []ast.Statement{
				&ast.AssignmentStatement{
					Target: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: name,
							Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       posFromName(name, 1),
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("42"),
						Base:            10,
						Value:           big.NewInt(42),
						Range: ast.NewUnmeteredRange(
							posFromName(name, 3),
							posFromName(name, 4),
						),
					},
				},
			}
			utils.AssertEqualWithDiff(t, expected, result)

		})
	}

	for _, keyword := range SoftKeywords {
		// it's not worth the additional complexity to support assigning to `remove` or `attach`-named
		// variables, so we just accept this as a parsing error
		if keyword == KeywordAttach || keyword == KeywordRemove {
			continue
		}
		testSoftKeyword(keyword)
	}
}

func TestParseStatementsWithWhitespace(t *testing.T) {

	t.Parallel()

	t.Run("two statements: variable declaration and parenthesized expression, not one function-call", func(t *testing.T) {
		t.Parallel()

		const code = `
          a == b
          (c)
	    `

		statements, errs := testParseStatements(code)
		require.Empty(t, errs)

		require.Len(t, statements, 2)
	})

	t.Run("two statements: binary expression and array literal, not an indexing expression", func(t *testing.T) {
		t.Parallel()

		const code = `
          a == b
          [c]
	    `

		statements, errs := testParseStatements(code)
		require.Empty(t, errs)

		require.Len(t, statements, 2)
	})

	t.Run("two statements: binary expression and unary prefix negation, not unary postfix force", func(t *testing.T) {
		t.Parallel()

		const code = `
          a == b
          !c == d
	    `

		statements, errs := testParseStatements(code)
		require.Empty(t, errs)

		require.Len(t, statements, 2)
	})

	t.Run("one statement: binary expression, right-hand side with member access", func(t *testing.T) {
		t.Parallel()

		const code = `
          a == b
          .c
	    `

		statements, errs := testParseStatements(code)
		require.Empty(t, errs)

		require.Len(t, statements, 1)
	})
}
