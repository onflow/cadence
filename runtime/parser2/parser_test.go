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

func TestParseExpression(t *testing.T) {

	t.Run("simple, no spaces", func(t *testing.T) {
		result, errors := Parse("1+2*3")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IntegerExpression{
					Value: big.NewInt(1),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationMul,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
							EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("simple, spaces", func(t *testing.T) {
		result, errors := Parse("  1   +   2  *   3 ")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IntegerExpression{
					Value: big.NewInt(1),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
						EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
					},
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationMul,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 10, Line: 1, Column: 10},
							EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 1, Column: 17},
							EndPos:   ast.Position{Offset: 17, Line: 1, Column: 17},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("repeated infix, same operator, left associative", func(t *testing.T) {
		result, errors := Parse("1 + 2 + 3")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 0, Line: 1, Column: 0},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
						EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
			},
			result,
		)
	})

	t.Run("repeated infix, same operator, right associative", func(t *testing.T) {
		result, errors := Parse("1 ?? 2 ?? 3")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationNilCoalesce,
				Left: &ast.IntegerExpression{
					Value: big.NewInt(1),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationNilCoalesce,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 10, Line: 1, Column: 10},
							EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("mixed infix and prefix", func(t *testing.T) {
		result, errors := Parse("1 +- 2 ++ 3")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 0, Line: 1, Column: 0},
						},
					},
					Right: &ast.UnaryExpression{
						Operation: ast.OperationMinus,
						Expression: &ast.IntegerExpression{
							Value: big.NewInt(2),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
								EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
							},
						},
						StartPos: ast.Position{Offset: 3, Line: 1, Column: 3},
					},
				},
				Right: &ast.UnaryExpression{
					Operation: ast.OperationPlus,
					Expression: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 10, Line: 1, Column: 10},
							EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
						},
					},
					StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
				},
			},
			result,
		)
	})

	t.Run("nested expression", func(t *testing.T) {
		result, errors := Parse("(1 + 2) * 3")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationMul,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
							EndPos:   ast.Position{Offset: 1, Line: 1, Column: 1},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 10, Line: 1, Column: 10},
						EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("less and greater", func(t *testing.T) {
		result, errors := Parse("1 < 2 > 3")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationGreater,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationLess,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 0, Line: 1, Column: 0},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
						EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
			},
			result,
		)
	})

	t.Run("array expression", func(t *testing.T) {
		result, errors := Parse("[ 1,2 + 3, 4  ,  5 ]")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.ArrayExpression{
				Values: []ast.Expression{
					&ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
							EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
						},
					},
					&ast.BinaryExpression{
						Operation: ast.OperationPlus,
						Left: &ast.IntegerExpression{
							Value: big.NewInt(2),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
						Right: &ast.IntegerExpression{
							Value: big.NewInt(3),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
								EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
							},
						},
					},
					&ast.IntegerExpression{
						Value: big.NewInt(4),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 11, Line: 1, Column: 11},
							EndPos:   ast.Position{Offset: 11, Line: 1, Column: 11},
						},
					},
					&ast.IntegerExpression{
						Value: big.NewInt(5),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 1, Column: 17},
							EndPos:   ast.Position{Offset: 17, Line: 1, Column: 17},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					EndPos:   ast.Position{Offset: 19, Line: 1, Column: 19},
				},
			},
			result,
		)
	})

	t.Run("dictionary expression", func(t *testing.T) {
		result, errors := Parse("{ 1:2 + 3, 4  :  5 }")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.DictionaryExpression{
				Entries: []ast.Entry{
					{
						Key: &ast.IntegerExpression{
							Value: big.NewInt(1),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
						Value: &ast.BinaryExpression{
							Operation: ast.OperationPlus,
							Left: &ast.IntegerExpression{
								Value: big.NewInt(2),
								Base:  10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
									EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
								},
							},
							Right: &ast.IntegerExpression{
								Value: big.NewInt(3),
								Base:  10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
									EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
								},
							},
						},
					},
					{
						Key: &ast.IntegerExpression{
							Value: big.NewInt(4),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 11, Line: 1, Column: 11},
								EndPos:   ast.Position{Offset: 11, Line: 1, Column: 11},
							},
						},
						Value: &ast.IntegerExpression{
							Value: big.NewInt(5),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 17, Line: 1, Column: 17},
								EndPos:   ast.Position{Offset: 17, Line: 1, Column: 17},
							},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					EndPos:   ast.Position{Offset: 19, Line: 1, Column: 19},
				},
			},
			result,
		)
	})

	t.Run("identifier in addition", func(t *testing.T) {
		result, errors := Parse("a + 3")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
						EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
					},
				},
			},
			result,
		)
	})

	t.Run("path expression", func(t *testing.T) {
		result, errors := Parse("/foo/bar")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.PathExpression{
				Domain: ast.Identifier{
					Identifier: "foo",
					Pos:        ast.Position{Offset: 1, Line: 1, Column: 1},
				},
				Identifier: ast.Identifier{
					Identifier: "bar",
					Pos:        ast.Position{Offset: 5, Line: 1, Column: 5},
				},
				StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
			},
			result,
		)
	})

	t.Run("conditional", func(t *testing.T) {
		result, errors := Parse("a ? b : c ? d : e")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.ConditionalExpression{
				Test: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Then: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "b",
						Pos:        ast.Position{Offset: 4, Line: 1, Column: 4},
					},
				},
				Else: &ast.ConditionalExpression{
					Test: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "c",
							Pos:        ast.Position{Offset: 8, Line: 1, Column: 8},
						},
					},
					Then: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "d",
							Pos:        ast.Position{Offset: 12, Line: 1, Column: 12},
						},
					},
					Else: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "e",
							Pos:        ast.Position{Offset: 16, Line: 1, Column: 16},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("boolean expressions", func(t *testing.T) {
		result, errors := Parse("true + false")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.BoolExpression{
					Value: true,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
					},
				},
				Right: &ast.BoolExpression{
					Value: false,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
						EndPos:   ast.Position{Offset: 11, Line: 1, Column: 11},
					},
				},
			},
			result,
		)
	})

	t.Run("move operator, nested", func(t *testing.T) {
		result, errors := Parse("(<-x)")
		require.Empty(t, errors)

		utils.AssertEqualWithDiff(t,
			&ast.UnaryExpression{
				Operation: ast.OperationMove,
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Offset: 3, Line: 1, Column: 3},
					},
				},
				StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
			},
			result,
		)
	})
}
