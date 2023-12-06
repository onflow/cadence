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
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser/lexer"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestParseSimpleInfixExpression(t *testing.T) {

	t.Parallel()

	t.Run("no spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("1+2*3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationMul,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("3"),
						Value:           big.NewInt(3),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("with spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("  1   +   2  *   3 ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
						EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationMul,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("3"),
						Value:           big.NewInt(3),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
							EndPos:   ast.Position{Line: 1, Column: 17, Offset: 17},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("repeated infix, same operator, left associative", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("1 + 2 + 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("3"),
					Value:           big.NewInt(3),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
					},
				},
			},
			result,
		)
	})

	t.Run("repeated infix, same operator, right associative", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("1 ?? 2 ?? 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationNilCoalesce,
				Left: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationNilCoalesce,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("3"),
						Value:           big.NewInt(3),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
				},
			},
			result,
		)
	})
}

func TestParseAdvancedExpression(t *testing.T) {

	t.Parallel()

	t.Run("mixed infix and prefix", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("1 +- 2 -- 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationMinus,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(-2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("3"),
					Value:           big.NewInt(-3),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("nested expression", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("(1 + 2) * 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationMul,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("3"),
					Value:           big.NewInt(3),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
						EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("MemoryError in setExprMetaLeftDenotation", func(t *testing.T) {

		t.Parallel()

		gauge := makeLimitingMemoryGauge()
		gauge.debug = true
		gauge.Limit(common.MemoryKindPosition, 11)

		var panicMsg any
		(func() {
			defer func() {
				panicMsg = recover()
			}()
			ParseExpression(gauge, []byte("1 < 2"), Config{})
		})()

		require.IsType(t, errors.MemoryError{}, panicMsg)

		fatalError, _ := panicMsg.(errors.MemoryError)
		var expectedError limitingMemoryGaugeError
		assert.ErrorAs(t, fatalError, &expectedError)
	})

	t.Run("MemoryError in parser.report", func(t *testing.T) {

		t.Parallel()

		gauge := makeLimitingMemoryGauge()
		gauge.Limit(common.MemoryKindIntegerExpression, 1)

		var panicMsg any
		(func() {
			defer func() {
				panicMsg = recover()
			}()

			ParseExpression(gauge, []byte("1 < 2 > 3"), Config{})
		})()

		require.IsType(t, errors.MemoryError{}, panicMsg)

		fatalError, _ := panicMsg.(errors.MemoryError)
		var expectedError limitingMemoryGaugeError
		assert.ErrorAs(t, fatalError, &expectedError)
	})

	t.Run("less and greater", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("1 < 2 > 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationGreater,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationLess,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("3"),
					Value:           big.NewInt(3),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
					},
				},
			},
			result,
		)
	})

	t.Run("conditional", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("a ? b : c ? d : e")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ConditionalExpression{
				Test: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Then: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "b",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
				Else: &ast.ConditionalExpression{
					Test: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "c",
							Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Then: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "d",
							Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
						},
					},
					Else: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "e",
							Pos:        ast.Position{Line: 1, Column: 16, Offset: 16},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("boolean expressions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("true + false")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.BoolExpression{
					Value: true,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
					},
				},
				Right: &ast.BoolExpression{
					Value: false,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
						EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
					},
				},
			},
			result,
		)
	})

	t.Run("move operator, nested", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("(<-x)")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.UnaryExpression{
				Operation: ast.OperationMove,
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 3, Offset: 3},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
			},
			result,
		)
	})

}

func TestParseArrayExpression(t *testing.T) {

	t.Parallel()

	t.Run("single line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("[ 1,2 + 3, 4  ,  5 ]")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ArrayExpression{
				Values: []ast.Expression{
					&ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					&ast.BinaryExpression{
						Operation: ast.OperationPlus,
						Left: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
								EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
							},
						},
						Right: &ast.IntegerExpression{
							PositiveLiteral: []byte("3"),
							Value:           big.NewInt(3),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
								EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
							},
						},
					},
					&ast.IntegerExpression{
						PositiveLiteral: []byte("4"),
						Value:           big.NewInt(4),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					&ast.IntegerExpression{
						PositiveLiteral: []byte("5"),
						Value:           big.NewInt(5),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
							EndPos:   ast.Position{Line: 1, Column: 17, Offset: 17},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 19, Offset: 19},
				},
			},
			result,
		)
	})

	t.Run("multi line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("[ 1 , \n 2 \n , \n\n 3 \n\n\n]")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ArrayExpression{
				Values: []ast.Expression{
					&ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					&ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 1, Offset: 8},
							EndPos:   ast.Position{Line: 2, Column: 1, Offset: 8},
						},
					},
					&ast.IntegerExpression{
						PositiveLiteral: []byte("3"),
						Value:           big.NewInt(3),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 5, Column: 1, Offset: 17},
							EndPos:   ast.Position{Line: 5, Column: 1, Offset: 17},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 8, Column: 0, Offset: 22},
				},
			},
			result,
		)
	})

	t.Run("empty, separated by newline", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("[\n]")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ArrayExpression{
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 2, Column: 0, Offset: 2},
				},
			},
			result,
		)
	})

}

func TestParseDictionaryExpression(t *testing.T) {

	t.Parallel()

	t.Run("single line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("{ 1:2 + 3, 4  :  5 }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.DictionaryExpression{
				Entries: []ast.DictionaryEntry{
					{
						Key: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
								EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
							},
						},
						Value: &ast.BinaryExpression{
							Operation: ast.OperationPlus,
							Left: &ast.IntegerExpression{
								PositiveLiteral: []byte("2"),
								Value:           big.NewInt(2),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
									EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
								},
							},
							Right: &ast.IntegerExpression{
								PositiveLiteral: []byte("3"),
								Value:           big.NewInt(3),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
									EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
								},
							},
						},
					},
					{
						Key: &ast.IntegerExpression{
							PositiveLiteral: []byte("4"),
							Value:           big.NewInt(4),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
								EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
							},
						},
						Value: &ast.IntegerExpression{
							PositiveLiteral: []byte("5"),
							Value:           big.NewInt(5),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
								EndPos:   ast.Position{Line: 1, Column: 17, Offset: 17},
							},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 19, Offset: 19},
				},
			},
			result,
		)
	})

	t.Run("multi line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("{ 1 : 2 , \n 3 \n : \n 4 \n , \n\n 5 \n\n : \n\n 6 \n\n }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.DictionaryExpression{
				Entries: []ast.DictionaryEntry{
					{
						Key: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
								EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
							},
						},
						Value: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
								EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
							},
						},
					},
					{
						Key: &ast.IntegerExpression{
							PositiveLiteral: []byte("3"),
							Value:           big.NewInt(3),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 2, Column: 1, Offset: 12},
								EndPos:   ast.Position{Line: 2, Column: 1, Offset: 12},
							},
						},
						Value: &ast.IntegerExpression{
							PositiveLiteral: []byte("4"),
							Value:           big.NewInt(4),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 4, Column: 1, Offset: 20},
								EndPos:   ast.Position{Line: 4, Column: 1, Offset: 20},
							},
						},
					},
					{
						Key: &ast.IntegerExpression{
							PositiveLiteral: []byte("5"),
							Value:           big.NewInt(5),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 7, Column: 1, Offset: 29},
								EndPos:   ast.Position{Line: 7, Column: 1, Offset: 29},
							},
						},
						Value: &ast.IntegerExpression{
							PositiveLiteral: []byte("6"),
							Value:           big.NewInt(6),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 11, Column: 1, Offset: 39},
								EndPos:   ast.Position{Line: 11, Column: 1, Offset: 39},
							},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 13, Column: 1, Offset: 44},
				},
			},
			result,
		)
	})

	t.Run("empty, separated by newline", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("{\n}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.DictionaryExpression{
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 2, Column: 0, Offset: 2},
				},
			},
			result,
		)
	})
}

func TestParseIndexExpression(t *testing.T) {
	t.Parallel()

	t.Run("index expression", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseExpression("a[0]")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IndexExpression{
				TargetExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				IndexingExpression: &ast.IntegerExpression{
					PositiveLiteral: []byte("0"),
					Value:           new(big.Int),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
						EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
					EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
				},
			},
			result,
		)
	})

	t.Run("index expression with whitespace", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseExpression("a [ 0 ]")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IndexExpression{
				TargetExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				IndexingExpression: &ast.IntegerExpression{
					PositiveLiteral: []byte("0"),
					Value:           new(big.Int),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
						EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
				},
			},
			result,
		)
	})

	t.Run("index expression with identifier", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseExpression("a [foo]")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IndexExpression{
				TargetExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				IndexingExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 3, Offset: 3},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
				},
			},
			result,
		)
	})
}

func TestParseIdentifier(t *testing.T) {

	t.Parallel()

	t.Run("identifier in addition", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("a + 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("3"),
					Value:           big.NewInt(3),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
						EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
			},
			result,
		)
	})
}

func TestParsePath(t *testing.T) {

	t.Parallel()

	result, errs := testParseExpression("/foo/bar")
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		&ast.PathExpression{
			Domain: ast.Identifier{
				Identifier: "foo",
				Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
			},
			Identifier: ast.Identifier{
				Identifier: "bar",
				Pos:        ast.Position{Line: 1, Column: 5, Offset: 5},
			},
			StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
		},
		result,
	)
}

func TestParseString(t *testing.T) {

	t.Parallel()

	t.Run("valid, empty", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("\"\"")
		assert.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("invalid, empty, missing end at end of file", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("\"")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid end of string literal: missing '\"'",
					Pos:     ast.Position{Offset: 1, Line: 1, Column: 1},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("invalid, empty, missing end at end of line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("\"\n")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid end of string literal: missing '\"'",
					Pos:     ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("invalid, non-empty, missing end at end of file", func(t *testing.T) {

		t.Parallel()
		result, errs := testParseExpression("\"t")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid end of string literal: missing '\"'",
					Pos:     ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "t",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("invalid, non-empty, missing end at end of line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("\"t\n")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid end of string literal: missing '\"'",
					Pos:     ast.Position{Line: 1, Column: 2, Offset: 2},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "t",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("invalid, non-empty, missing escape character", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("\"\\")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "incomplete escape sequence: missing character after escape character",
					Pos:     ast.Position{Offset: 2, Line: 1, Column: 2},
				},
				&SyntaxError{
					Message: "invalid end of string literal: missing '\"'",
					Pos:     ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("valid, with escapes", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`"te\tst\"te\u{1F3CE}\u{FE0F}xt"`)
		assert.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "te\tst\"te\U0001F3CE\uFE0Fxt",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 30, Offset: 30},
				},
			},
			result,
		)
	})

	t.Run("invalid, unknown escape character", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`"te\Xst"`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid escape character: 'X'",
					Pos:     ast.Position{Offset: 8, Line: 1, Column: 8},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "test",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)
	})

	t.Run("invalid, missing '{' after Unicode escape character", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`"te\u`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "incomplete Unicode escape sequence: missing character '{' after escape character",
					Pos:     ast.Position{Offset: 5, Line: 1, Column: 5},
				},
				&SyntaxError{
					Message: "invalid end of string literal: missing '\"'",
					Pos:     ast.Position{Offset: 5, Line: 1, Column: 5},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "te",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
				},
			},
			result,
		)
	})

	t.Run("invalid, invalid character after Unicode escape character", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`"te\us`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid Unicode escape sequence: expected '{', got 's'",
					Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
				},
				&SyntaxError{
					Message: "invalid end of string literal: missing '\"'",
					Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "te",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
				},
			},
			result,
		)
	})

	t.Run("invalid, missing '}' after Unicode escape sequence digits", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`"te\u{`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "incomplete Unicode escape sequence: missing character '}' after escape character",
					Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
				},
				&SyntaxError{
					Message: "invalid end of string literal: missing '\"'",
					Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "te",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
				},
			},
			result,
		)
	})

	t.Run("valid, empty Unicode escape sequence", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`"te\u{}"`)
		assert.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "te",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)
	})

	t.Run("valid, non-empty Unicode escape sequence", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(
			`"te\u{73}t ` +
				`\u{4A}J\u{4a}J ` +
				`\u{4B}K\u{4b}K ` +
				`\u{4C}L\u{4c}L ` +
				`\u{4D}M\u{4d}M ` +
				`\u{4E}N\u{4e}N ` +
				`\u{4F}O\u{4f}O"`,
		)
		assert.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "test JJJJ KKKK LLLL MMMM NNNN OOOO",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 100, Offset: 100},
				},
			},
			result,
		)
	})

	t.Run("invalid, non-empty Unicode escape sequence", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`"te\u{X}st"`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid Unicode escape sequence: expected hex digit, got 'X'",
					Pos:     ast.Position{Offset: 11, Line: 1, Column: 11},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.StringExpression{
				Value: "test",
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
				},
			},
			result,
		)
	})
}

func TestParseInvocation(t *testing.T) {

	t.Parallel()

	t.Run("no arguments", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments:         nil,
				ArgumentsStartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
				EndPos:            ast.Position{Offset: 2, Line: 1, Column: 2},
			},
			result,
		)
	})

	t.Run("no arguments, with whitespace", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f ()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments:         nil,
				ArgumentsStartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
				EndPos:            ast.Position{Offset: 3, Line: 1, Column: 3},
			},
			result,
		)
	})

	t.Run("no arguments, with whitespace within params", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f ( )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments:         nil,
				ArgumentsStartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
				EndPos:            ast.Position{Offset: 4, Line: 1, Column: 4},
			},
			result,
		)
	})

	t.Run("with arguments", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f(1)")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments: []*ast.Argument{
					{
						Label: "",
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
						TrailingSeparatorPos: ast.Position{Offset: 3, Line: 1, Column: 3},
					},
				},
				ArgumentsStartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
				EndPos:            ast.Position{Offset: 3, Line: 1, Column: 3},
			},
			result,
		)
	})

	t.Run("with labeled arguments", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f(label:1)")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments: []*ast.Argument{
					{
						Label:         "label",
						LabelStartPos: &ast.Position{Offset: 2, Line: 1, Column: 2},
						LabelEndPos:   &ast.Position{Offset: 7, Line: 1, Column: 7},
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
								EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
							},
						},
						TrailingSeparatorPos: ast.Position{Offset: 9, Line: 1, Column: 9},
					},
				},
				ArgumentsStartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
				EndPos:            ast.Position{Offset: 9, Line: 1, Column: 9},
			},
			result,
		)
	})

	t.Run("with arguments, multiple", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f(1,2)")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments: []*ast.Argument{
					{
						Label: "",
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
						TrailingSeparatorPos: ast.Position{Offset: 3, Line: 1, Column: 3},
					},
					{
						Label: "",
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
						TrailingSeparatorPos: ast.Position{Offset: 5, Line: 1, Column: 5},
					},
				},
				ArgumentsStartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
				EndPos:            ast.Position{Offset: 5, Line: 1, Column: 5},
			},
			result,
		)
	})

	t.Run("with arguments, multiple, labeled", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f(a:1,b:2)")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments: []*ast.Argument{
					{
						Label:         "a",
						LabelStartPos: &ast.Position{Offset: 2, Line: 1, Column: 2},
						LabelEndPos:   &ast.Position{Offset: 3, Line: 1, Column: 3},
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
						TrailingSeparatorPos: ast.Position{Offset: 5, Line: 1, Column: 5},
					},
					{
						Label:         "b",
						LabelStartPos: &ast.Position{Offset: 6, Line: 1, Column: 6},
						LabelEndPos:   &ast.Position{Offset: 7, Line: 1, Column: 7},
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
								EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
							},
						},
						TrailingSeparatorPos: ast.Position{Offset: 9, Line: 1, Column: 9},
					},
				},
				ArgumentsStartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
				EndPos:            ast.Position{Offset: 9, Line: 1, Column: 9},
			},
			result,
		)
	})

	t.Run("invalid: no arguments, multiple commas", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression("f(,,)")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected argument or end of argument list, got ','",
					Pos:     ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			errs,
		)
	})

	t.Run("invalid: with argument, multiple commas", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression("f(1,,)")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected argument or end of argument list, got ','",
					Pos:     ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)
	})

	t.Run("invalid: with multiple argument, no commas", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression("f(1 2)")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected argument in argument list (expecting delimiter or end of argument list)," +
						" got decimal integer",
					Pos: ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)
	})

	t.Run("with arguments, nested", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f(1,g(2))")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments: []*ast.Argument{
					{
						Label: "",
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
						TrailingSeparatorPos: ast.Position{Offset: 3, Line: 1, Column: 3},
					},
					{
						Label: "",
						Expression: &ast.InvocationExpression{
							InvokedExpression: &ast.IdentifierExpression{
								Identifier: ast.Identifier{
									Identifier: "g",
									Pos:        ast.Position{Offset: 4, Line: 1, Column: 4},
								},
							},
							Arguments: []*ast.Argument{
								{
									Label: "",
									Expression: &ast.IntegerExpression{
										PositiveLiteral: []byte("2"),
										Value:           big.NewInt(2),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 6, Line: 1, Column: 6},
											EndPos:   ast.Position{Offset: 6, Line: 1, Column: 6},
										},
									},
									TrailingSeparatorPos: ast.Position{Offset: 7, Line: 1, Column: 7},
								},
							},
							ArgumentsStartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
							EndPos:            ast.Position{Offset: 7, Line: 1, Column: 7},
						},
						TrailingSeparatorPos: ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
				ArgumentsStartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
				EndPos:            ast.Position{Offset: 8, Line: 1, Column: 8},
			},
			result,
		)
	})

	t.Run("with arguments, nested, string", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f(1,g(\"test\"))")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments: []*ast.Argument{
					{
						Label: "",
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
						TrailingSeparatorPos: ast.Position{Offset: 3, Line: 1, Column: 3},
					},
					{
						Label: "",
						Expression: &ast.InvocationExpression{
							InvokedExpression: &ast.IdentifierExpression{
								Identifier: ast.Identifier{
									Identifier: "g",
									Pos:        ast.Position{Offset: 4, Line: 1, Column: 4},
								},
							},
							Arguments: []*ast.Argument{
								{
									Label: "",
									Expression: &ast.StringExpression{
										Value: "test",
										Range: ast.Range{
											StartPos: ast.Position{Offset: 6, Line: 1, Column: 6},
											EndPos:   ast.Position{Offset: 11, Line: 1, Column: 11},
										},
									},
									TrailingSeparatorPos: ast.Position{Offset: 12, Line: 1, Column: 12},
								},
							},
							ArgumentsStartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
							EndPos:            ast.Position{Offset: 12, Line: 1, Column: 12},
						},
						TrailingSeparatorPos: ast.Position{Offset: 13, Line: 1, Column: 13},
					},
				},
				ArgumentsStartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
				EndPos:            ast.Position{Offset: 13, Line: 1, Column: 13},
			},
			result,
		)
	})
}

func TestParseMemberExpression(t *testing.T) {

	t.Parallel()

	t.Run("identifier, no space", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f.n")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.MemberExpression{
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				AccessPos: ast.Position{Offset: 1, Line: 1, Column: 1},
				Identifier: ast.Identifier{
					Identifier: "n",
					Pos:        ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			result,
		)
	})

	t.Run("whitespace before", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f .n")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.MemberExpression{
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				AccessPos: ast.Position{Offset: 2, Line: 1, Column: 2},
				Identifier: ast.Identifier{
					Identifier: "n",
					Pos:        ast.Position{Offset: 3, Line: 1, Column: 3},
				},
			},
			result,
		)
	})

	t.Run("missing name", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f.")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected member name, got EOF",
					Pos:     ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.MemberExpression{
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				AccessPos: ast.Position{Offset: 1, Line: 1, Column: 1},
			},
			result,
		)
	})

	t.Run("precedence, left", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f.n * 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationMul,
				Left: &ast.MemberExpression{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "f",
							Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
						},
					},
					AccessPos: ast.Position{Offset: 1, Line: 1, Column: 1},
					Identifier: ast.Identifier{
						Identifier: "n",
						Pos:        ast.Position{Offset: 2, Line: 1, Column: 2},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("3"),
					Value:           big.NewInt(3),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 6, Line: 1, Column: 6},
						EndPos:   ast.Position{Offset: 6, Line: 1, Column: 6},
					},
				},
			},
			result,
		)
	})

	t.Run("precedence, right", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("3 * f.n")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationMul,
				Left: &ast.IntegerExpression{
					PositiveLiteral: []byte("3"),
					Value:           big.NewInt(3),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Right: &ast.MemberExpression{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "f",
							Pos:        ast.Position{Offset: 4, Line: 1, Column: 4},
						},
					},
					AccessPos: ast.Position{Offset: 5, Line: 1, Column: 5},
					Identifier: ast.Identifier{
						Identifier: "n",
						Pos:        ast.Position{Offset: 6, Line: 1, Column: 6},
					},
				},
			},
			result,
		)
	})

	t.Run("identifier, optional", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("f?.n")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.MemberExpression{
				Optional: true,
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				AccessPos: ast.Position{Offset: 2, Line: 1, Column: 2},
				Identifier: ast.Identifier{
					Identifier: "n",
					Pos:        ast.Position{Offset: 3, Line: 1, Column: 3},
				},
			},
			result,
		)
	})

	t.Run("in declaration", func(t *testing.T) {

		t.Parallel()

		const code = `
          let a = b.c
        `
		result, errs := testParseProgram(code)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					Access:     ast.AccessNotSpecified,
					IsConstant: true,
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Offset: 15, Line: 2, Column: 14},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       ast.Position{Offset: 17, Line: 2, Column: 16},
					},
					Value: &ast.MemberExpression{
						Expression: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "b",
								Pos:        ast.Position{Offset: 19, Line: 2, Column: 18},
							},
						},
						Identifier: ast.Identifier{
							Identifier: "c",
							Pos:        ast.Position{Offset: 21, Line: 2, Column: 20},
						},
						AccessPos: ast.Position{Offset: 20, Line: 2, Column: 19},
					},
					StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
				},
			},
			result.Declarations(),
		)
	})
}

func TestParseBlockComment(t *testing.T) {

	t.Parallel()

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(" /* test  foo/* bar  */ asd*/ true")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BoolExpression{
				Value: true,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 30, Offset: 30},
					EndPos:   ast.Position{Line: 1, Column: 33, Offset: 33},
				},
			},
			result,
		)
	})

	t.Run("two comments", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(" /*test  foo*/ /* bar  */ true")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BoolExpression{
				Value: true,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 26, Offset: 26},
					EndPos:   ast.Position{Line: 1, Column: 29, Offset: 29},
				},
			},
			result,
		)
	})

	t.Run("in infix", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(" 1/*test  foo*/+/* bar  */ 2  ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("2"),
					Value:           big.NewInt(2),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 27, Offset: 27},
						EndPos:   ast.Position{Line: 1, Column: 27, Offset: 27},
					},
				},
			},
			result,
		)
	})

	t.Run("nested, extra closing", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression(" /* test  foo/* bar  */ asd*/ true */ bar")
		utils.AssertEqualWithDiff(t,
			[]error{
				// `true */ bar` is parsed as infix operation of path
				&SyntaxError{
					Message: "expected token identifier",
					Pos: ast.Position{
						Offset: 37,
						Line:   1,
						Column: 37,
					},
				},
			},
			errs,
		)
	})

	t.Run("nested, missing closing", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression(" /* test  foo/* bar  */ asd true ")
		utils.AssertEqualWithDiff(t,
			[]error{
				// `true */ bar` is parsed as infix operation of path
				&SyntaxError{
					Message: "missing comment end '*/'",
					Pos: ast.Position{
						Offset: 33,
						Line:   1,
						Column: 33,
					},
				},
				&SyntaxError{
					Message: "unexpected end of program",
					Pos: ast.Position{
						Offset: 33,
						Line:   1,
						Column: 33,
					},
				},
			},
			errs,
		)
	})

	t.Run("invalid content", func(t *testing.T) {

		t.Parallel()

		// The lexer should never produce such an invalid token stream in the first place

		tokens := &testTokenStream{
			tokens: []lexer.Token{
				{
					Type: lexer.TokenBlockCommentStart,
					Range: ast.Range{
						StartPos: ast.Position{
							Line:   1,
							Offset: 0,
							Column: 0,
						},
						EndPos: ast.Position{
							Line:   1,
							Offset: 1,
							Column: 1,
						},
					},
				},
				{
					Type: lexer.TokenIdentifier,
					Range: ast.Range{
						StartPos: ast.Position{
							Line:   1,
							Offset: 2,
							Column: 2,
						},
						EndPos: ast.Position{
							Line:   1,
							Offset: 4,
							Column: 4,
						},
					},
				},
				{Type: lexer.TokenEOF},
			},
			input: []byte(`/*foo`),
		}

		_, errs := ParseTokenStream(
			nil,
			tokens,
			func(p *parser) (ast.Expression, error) {
				return parseExpression(p, lowestBindingPower)
			},
			Config{},
		)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token identifier in block comment",
					Pos: ast.Position{
						Line:   1,
						Offset: 2,
						Column: 2,
					},
				},
			},
			errs,
		)
	})
}

func BenchmarkParseInfix(b *testing.B) {

	for i := 0; i < b.N; i++ {
		_, errs := testParseExpression("(8 - 1 + 3) * 6 - ((3 + 7) * 2)")
		if len(errs) > 0 {
			b.Fatalf("parsing expression failed: %s", errs)
		}
	}
}

func BenchmarkParseArray(b *testing.B) {

	var builder strings.Builder
	for i := 0; i < 10_000; i++ {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(strconv.Itoa(rand.Intn(math.MaxUint8)))
	}

	lit := fmt.Sprintf(`[%s]`, builder.String())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, errs := testParseExpression(lit)
		if len(errs) > 0 {
			b.Fatalf("parsing expression failed: %s", errs)
		}
	}
}

func TestParseReference(t *testing.T) {

	t.Parallel()

	t.Run("valid: simple", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("& t")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ReferenceExpression{
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "t",
						Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("valid: casting expression", func(t *testing.T) {

		t.Parallel()

		const code = `&y[z]`

		result, errs := testParseExpression(code)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ReferenceExpression{
				Expression: &ast.IndexExpression{
					TargetExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "y",
							Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					IndexingExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "z",
							Pos:        ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
						EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("invalid: optional referenced value", func(t *testing.T) {

		t.Parallel()

		const code = `&x[y]? as &Z?`

		_, errs := testParseExpression(code)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected end of program",
					Pos: ast.Position{
						Offset: 13,
						Line:   1,
						Column: 13,
					},
				},
			},
			errs,
		)
	})
}

func TestParseNilCoelesceReference(t *testing.T) {

	t.Parallel()

	result, errs := testParseExpression(`
          &xs["a"] as &Int? ?? 1
        `)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		&ast.BinaryExpression{
			Operation: ast.OperationNilCoalesce,
			Left: &ast.CastingExpression{
				Expression: &ast.ReferenceExpression{
					Expression: &ast.IndexExpression{
						TargetExpression: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "xs",
								Pos: ast.Position{
									Offset: 12,
									Line:   2,
									Column: 11,
								},
							},
						},
						IndexingExpression: &ast.StringExpression{
							Value: "a",
							Range: ast.Range{
								StartPos: ast.Position{
									Offset: 15,
									Line:   2,
									Column: 14,
								},
								EndPos: ast.Position{
									Offset: 17,
									Line:   2,
									Column: 16,
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{
								Offset: 14,
								Line:   2,
								Column: 13,
							},
							EndPos: ast.Position{
								Offset: 18,
								Line:   2,
								Column: 17,
							},
						},
					},
					StartPos: ast.Position{
						Offset: 11,
						Line:   2,
						Column: 10,
					},
				},
				Operation: ast.OperationCast,
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.OptionalType{
						Type: &ast.ReferenceType{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos: ast.Position{
										Offset: 24,
										Line:   2,
										Column: 23,
									},
								},
							},
							StartPos: ast.Position{
								Offset: 23,
								Line:   2,
								Column: 22,
							},
						},
						EndPos: ast.Position{
							Offset: 27,
							Line:   2,
							Column: 26,
						},
					},
					StartPos: ast.Position{
						Offset: 23,
						Line:   2,
						Column: 22,
					},
				},
			},
			Right: &ast.IntegerExpression{
				PositiveLiteral: []byte("1"),
				Value:           big.NewInt(1),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{
						Offset: 32,
						Line:   2,
						Column: 31,
					},
					EndPos: ast.Position{
						Offset: 32,
						Line:   2,
						Column: 31,
					},
				},
			},
		},
		result,
	)
}

func TestParseCasts(t *testing.T) {

	t.Parallel()

	t.Run("non-failable", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(" t as T")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.CastingExpression{
				Operation: ast.OperationCast,
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "t",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "T",
							Pos:        ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
				},
			},
			result,
		)
	})

	t.Run("failable", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(" t as? T")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.CastingExpression{
				Operation: ast.OperationFailableCast,
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "t",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "T",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)

	})

	t.Run("force", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(" t as! T")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.CastingExpression{
				Operation: ast.OperationForceCast,
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "t",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "T",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)
	})
}

func TestParseForceExpression(t *testing.T) {

	t.Parallel()

	t.Run("identifier", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("t!")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ForceExpression{
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "t",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				EndPos: ast.Position{Line: 1, Column: 1, Offset: 1},
			},
			result,
		)
	})

	t.Run("with whitespace", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(" t ! ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ForceExpression{
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "t",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				EndPos: ast.Position{Line: 1, Column: 3, Offset: 3},
			},
			result,
		)
	})

	t.Run("precedence, force unwrap before move", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("<-t!")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.UnaryExpression{
				Operation: ast.OperationMove,
				Expression: &ast.ForceExpression{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "t",
							Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					EndPos: ast.Position{Line: 1, Column: 3, Offset: 3},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("precedence", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("10 *  t!")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationMul,
				Left: &ast.IntegerExpression{
					PositiveLiteral: []byte("10"),
					Value:           big.NewInt(10),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				Right: &ast.ForceExpression{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "t",
							Pos:        ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					EndPos: ast.Position{Line: 1, Column: 7, Offset: 7},
				}},
			result,
		)
	})

	t.Run("newline", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("x\n!y")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "x",
							Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
				},
				&ast.ExpressionStatement{
					Expression: &ast.UnaryExpression{
						Operation: ast.OperationNegate,
						Expression: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "y",
								Pos:        ast.Position{Line: 2, Column: 1, Offset: 3},
							},
						},
						StartPos: ast.Position{Line: 2, Column: 0, Offset: 2},
					},
				},
			},
			result,
		)
	})

	t.Run("member access, newline", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("x\n.y!")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.ForceExpression{
						Expression: &ast.MemberExpression{
							Expression: &ast.IdentifierExpression{
								Identifier: ast.Identifier{
									Identifier: "x",
									Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
								},
							},
							AccessPos: ast.Position{Line: 2, Column: 0, Offset: 2},
							Identifier: ast.Identifier{
								Identifier: "y",
								Pos:        ast.Position{Line: 2, Column: 1, Offset: 3},
							},
						},
						EndPos: ast.Position{Line: 2, Column: 2, Offset: 4},
					},
				},
			},
			result,
		)
	})

	t.Run("member access, whitespace", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("x. y")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid whitespace after '.'",
					Pos:     ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.MemberExpression{
						Expression: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
							},
						},
						AccessPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						Identifier: ast.Identifier{
							Identifier: "y",
							Pos:        ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
				},
			},
			result,
		)
	})
}

func TestParseCreate(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("create T()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.CreateExpression{
				InvocationExpression: &ast.InvocationExpression{
					InvokedExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "T",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					ArgumentsStartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					EndPos:            ast.Position{Line: 1, Column: 9, Offset: 9},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})
}

func TestParseNil(t *testing.T) {

	t.Parallel()

	result, errs := testParseExpression(" nil")
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		&ast.NilExpression{
			Pos: ast.Position{Line: 1, Column: 1, Offset: 1},
		},
		result,
	)
}

func TestParseDestroy(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("destroy t")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.DestroyExpression{
				Expression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "t",
						Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})
}

func TestParseAttach(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("attach E() to r")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.AttachExpression{
				Base: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "r",
						Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
					},
				},
				Attachment: &ast.InvocationExpression{
					InvokedExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "E",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					ArgumentsStartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					EndPos:            ast.Position{Line: 1, Column: 9, Offset: 9},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("with provided entitlements", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression("attach E() to r with (X)")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 16, Line: 1, Column: 16},
				},
			},
			errs,
		)
	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("attach A() to attach B() to r")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.AttachExpression{
				Base: &ast.AttachExpression{
					Base: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "r",
							Pos:        ast.Position{Line: 1, Column: 28, Offset: 28},
						},
					},
					Attachment: &ast.InvocationExpression{
						InvokedExpression: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "B",
								Pos:        ast.Position{Line: 1, Column: 21, Offset: 21},
							},
						},
						ArgumentsStartPos: ast.Position{Line: 1, Column: 22, Offset: 22},
						EndPos:            ast.Position{Line: 1, Column: 23, Offset: 23},
					},
					StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
				},
				Attachment: &ast.InvocationExpression{
					InvokedExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "A",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					ArgumentsStartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					EndPos:            ast.Position{Line: 1, Column: 9, Offset: 9},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("missing to", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression("attach A()")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected 'to', got EOF",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
		)
	})

	t.Run("missing base", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression("attach E() to")
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
}

func TestParseLineComment(t *testing.T) {

	t.Parallel()

	result, errs := testParseExpression(" //// // this is a comment\n 1 / 2")
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		&ast.BinaryExpression{
			Operation: ast.OperationDiv,
			Left: &ast.IntegerExpression{
				PositiveLiteral: []byte("1"),
				Value:           big.NewInt(1),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 2, Column: 1, Offset: 28},
					EndPos:   ast.Position{Line: 2, Column: 1, Offset: 28},
				},
			},
			Right: &ast.IntegerExpression{
				PositiveLiteral: []byte("2"),
				Value:           big.NewInt(2),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 2, Column: 5, Offset: 32},
					EndPos:   ast.Position{Line: 2, Column: 5, Offset: 32},
				},
			},
		},
		result,
	)
}

func TestParseFunctionExpression(t *testing.T) {

	t.Parallel()

	t.Run("without return type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("fun () { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FunctionExpression{
				ParameterList: &ast.ParameterList{
					Parameters: nil,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
						EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("with return type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("fun (): X { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FunctionExpression{
				ParameterList: &ast.ParameterList{
					Parameters: nil,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
						EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "X",
							Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
						},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("with purity", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("view fun (): X { }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FunctionExpression{
				Purity: 1,
				ParameterList: &ast.ParameterList{
					Parameters: nil,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
						EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "X",
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
			result,
		)
	})

	t.Run("view with wrong keyword", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression("view for (): X { }")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected token: identifier",
					Pos:     ast.Position{Offset: 5, Line: 1, Column: 5},
				},
			},
			errs,
		)
	})
}

func TestParseAdjacentViewKeyword(t *testing.T) {
	// ensure that spaces and comments between adjacent keywords are treated the same, i.e. ignored

	t.Parallel()

	code := `
		view /* UwU */ fun(){}
	`

	result, errs := testParseExpression(code)

	require.Empty(t, errs)

	expected := &ast.FunctionExpression{
		Purity: ast.FunctionPurityView,
		ParameterList: &ast.ParameterList{
			Range: ast.NewUnmeteredRange(
				ast.Position{Line: 2, Column: 20, Offset: 21},
				ast.Position{Line: 2, Column: 21, Offset: 22},
			),
		},
		FunctionBlock: &ast.FunctionBlock{
			Block: &ast.Block{
				Range: ast.NewUnmeteredRange(
					ast.Position{Line: 2, Column: 22, Offset: 23},
					ast.Position{Line: 2, Column: 23, Offset: 24},
				),
			},
		},
		StartPos: ast.Position{Line: 2, Column: 2, Offset: 3},
	}
	utils.AssertEqualWithDiff(t, expected, result)

}

func TestParseIntegerLiterals(t *testing.T) {

	t.Parallel()

	t.Run("binary prefix, missing trailing digits", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0b`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "missing digits",
					Pos:     ast.Position{Offset: 1, Line: 1, Column: 1},
				},
				&InvalidIntegerLiteralError{
					Literal:                   "0b",
					IntegerLiteralKind:        common.IntegerLiteralKindBinary,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindMissingDigits,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0b"),
				Value:           new(big.Int),
				Base:            2,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("binary", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0b101010`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0b101010"),
				Value:           big.NewInt(42),
				Base:            2,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)
	})

	t.Run("binary with leading zeros", func(t *testing.T) {

		result, errs := testParseExpression(`0b001000`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0b001000"),
				Value:           big.NewInt(8),
				Base:            2,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)
	})

	t.Run("binary with underscores", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0b101010_101010`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0b101010_101010"),
				Value:           big.NewInt(2730),
				Base:            2,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
				},
			},
			result,
		)
	})

	t.Run("binary with leading underscore", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0b_101010_101010`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&InvalidIntegerLiteralError{
					Literal:                   "0b_101010_101010",
					IntegerLiteralKind:        common.IntegerLiteralKindBinary,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindLeadingUnderscore,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0b_101010_101010"),
				Value:           big.NewInt(2730),
				Base:            2,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
				},
			},
			result,
		)
	})

	t.Run("binary with trailing underscore", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0b101010_101010_`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&InvalidIntegerLiteralError{
					Literal:                   "0b101010_101010_",
					IntegerLiteralKind:        common.IntegerLiteralKindBinary,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindTrailingUnderscore,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0b101010_101010_"),
				Value:           big.NewInt(2730),
				Base:            2,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
				},
			},
			result,
		)
	})

	t.Run("octal prefix, missing trailing digits", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0o`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "missing digits",
					Pos:     ast.Position{Line: 1, Column: 1, Offset: 1},
				},
				&InvalidIntegerLiteralError{
					Literal:                   `0o`,
					IntegerLiteralKind:        common.IntegerLiteralKindOctal,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindMissingDigits,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0o"),
				Value:           new(big.Int),
				Base:            8,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("octal", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0o32`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0o32"),
				Value:           big.NewInt(26),
				Base:            8,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
				},
			},
			result,
		)
	})

	t.Run("octal with underscores", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0o32_45`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0o32_45"),
				Value:           big.NewInt(1701),
				Base:            8,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
				},
			},
			result,
		)
	})

	t.Run("octal with trailing underscore", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0o_32_45`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&InvalidIntegerLiteralError{
					Literal:                   "0o_32_45",
					IntegerLiteralKind:        common.IntegerLiteralKindOctal,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindLeadingUnderscore,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0o_32_45"),
				Value:           big.NewInt(1701),
				Base:            8,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)
	})

	t.Run("octal with leading underscore", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0o32_45_`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&InvalidIntegerLiteralError{
					Literal:                   "0o32_45_",
					IntegerLiteralKind:        common.IntegerLiteralKindOctal,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindTrailingUnderscore,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0o32_45_"),
				Value:           big.NewInt(1701),
				Base:            8,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)
	})

	t.Run("decimal", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`1234567890`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("1234567890"),
				Value:           big.NewInt(1234567890),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
				},
			},
			result,
		)
	})

	t.Run("decimal with underscores", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`1_234_567_890`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("1_234_567_890"),
				Value:           big.NewInt(1234567890),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
				},
			},
			result,
		)
	})

	t.Run("decimal with trailing underscore", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`1_234_567_890_`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&InvalidIntegerLiteralError{
					Literal:                   "1_234_567_890_",
					IntegerLiteralKind:        common.IntegerLiteralKindDecimal,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindTrailingUnderscore,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("1_234_567_890_"),
				Value:           big.NewInt(1234567890),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
				},
			},
			result,
		)
	})

	t.Run("hexadecimal prefix, missing trailing digits", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0x`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "missing digits",
					Pos:     ast.Position{Line: 1, Column: 1, Offset: 1},
				},
				&InvalidIntegerLiteralError{
					Literal:                   `0x`,
					IntegerLiteralKind:        common.IntegerLiteralKindHexadecimal,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindMissingDigits,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0x"),
				Value:           new(big.Int),
				Base:            16,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("hexadecimal", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0xf2`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0xf2"),
				Value:           big.NewInt(242),
				Base:            16,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
				},
			},
			result,
		)
	})

	t.Run("hexadecimal with underscores", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0xf2_09`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0xf2_09"),
				Value:           big.NewInt(61961),
				Base:            16,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
				},
			},
			result,
		)
	})

	t.Run("hexadecimal with leading underscore", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0x_f2_09`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&InvalidIntegerLiteralError{
					Literal:                   "0x_f2_09",
					IntegerLiteralKind:        common.IntegerLiteralKindHexadecimal,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindLeadingUnderscore,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0x_f2_09"),
				Value:           big.NewInt(61961),
				Base:            16,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)
	})

	t.Run("hexadecimal with trailing underscore", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0xf2_09_`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&InvalidIntegerLiteralError{
					Literal:                   `0xf2_09_`,
					IntegerLiteralKind:        common.IntegerLiteralKindHexadecimal,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindTrailingUnderscore,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0xf2_09_"),
				Value:           big.NewInt(61961),
				Base:            16,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			result,
		)
	})

	t.Run("0", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0"),
				Value:           big.NewInt(0),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("01", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`01`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("01"),
				Value:           big.NewInt(1),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("09", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`09`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("09"),
				Value:           big.NewInt(9),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("leading zeros", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("00123")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("00123"),
				Value:           big.NewInt(123),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
				},
			},
			result,
		)
	})

	t.Run("invalid prefix", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0z123`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid number literal prefix: 'z'",
					Pos:     ast.Position{Line: 1, Column: 1, Offset: 1},
				},
				&InvalidIntegerLiteralError{
					Literal:                   `0z123`,
					IntegerLiteralKind:        common.IntegerLiteralKindUnknown,
					InvalidIntegerLiteralKind: InvalidNumberLiteralKindUnknownPrefix,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0z123"),
				Value:           new(big.Int),
				Base:            1,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
				},
			},
			result,
		)
	})

	t.Run("leading zero and underscore", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`0_100`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("0_100"),
				Value:           big.NewInt(100),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
				},
			},
			result,
		)
	})

	t.Run("leading one and underscore", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(`1_100`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IntegerExpression{
				PositiveLiteral: []byte("1_100"),
				Value:           big.NewInt(1100),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
				},
			},
			result,
		)
	})

	t.Run("leading underscore", func(t *testing.T) {

		t.Parallel()

		// NOTE: a leading underscore introduces an identifier

		result, errs := testParseExpression(`_100`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.IdentifierExpression{
				Identifier: ast.Identifier{
					Identifier: "_100",
					Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			result,
		)
	})
}

func TestParseFixedPoint(t *testing.T) {

	t.Parallel()

	t.Run("with underscores", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("1234_5678_90.0009_8765_4321")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FixedPointExpression{
				PositiveLiteral: []byte("1234_5678_90.0009_8765_4321"),
				Negative:        false,
				UnsignedInteger: big.NewInt(1234567890),
				Fractional:      big.NewInt(987654321),
				Scale:           12,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 26, Offset: 26},
				},
			},
			result,
		)
	})

	t.Run("leading zero", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("0.1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.FixedPointExpression{
				PositiveLiteral: []byte("0.1"),
				Negative:        false,
				UnsignedInteger: big.NewInt(0),
				Fractional:      big.NewInt(1),
				Scale:           1,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
				},
			},
			result,
		)
	})

	t.Run("missing fractional digits", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("0.")
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "missing fractional digits",
					Pos:     ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			errs,
		)

		utils.AssertEqualWithDiff(t,
			&ast.FixedPointExpression{
				PositiveLiteral: []byte("0."),
				Negative:        false,
				UnsignedInteger: big.NewInt(0),
				Fractional:      big.NewInt(0),
				Scale:           1,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})
}

func TestParseLessThanOrTypeArguments(t *testing.T) {

	t.Parallel()

	t.Run("binary expression with less operator", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("1 < 2")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationLess,
				Left: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("2"),
					Value:           big.NewInt(2),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
						EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
			},
			result,
		)
	})

	t.Run("invocation, zero type arguments, zero arguments", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("a < > ()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments:     nil,
				Arguments:         nil,
				ArgumentsStartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
				EndPos:            ast.Position{Line: 1, Column: 7, Offset: 7},
			},
			result,
		)
	})

	t.Run("invocation, one type argument, one argument", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("a < { K : V } > ( 1 )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.DictionaryType{
							KeyType: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "K",
									Pos:        ast.Position{Line: 1, Column: 6, Offset: 6},
								},
							},
							ValueType: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "V",
									Pos:        ast.Position{Line: 1, Column: 10, Offset: 10},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
								EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
				Arguments: []*ast.Argument{
					{
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 18, Offset: 18},
								EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
							},
						},
						TrailingSeparatorPos: ast.Position{Line: 1, Column: 20, Offset: 20},
					},
				},
				ArgumentsStartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
				EndPos:            ast.Position{Line: 1, Column: 20, Offset: 20},
			},
			result,
		)
	})

	t.Run("invocation, three type arguments, two arguments", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("a < { K : V } , @R , [ S ] > ( 1 , 2 )")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.DictionaryType{
							KeyType: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "K",
									Pos:        ast.Position{Line: 1, Column: 6, Offset: 6},
								},
							},
							ValueType: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "V",
									Pos:        ast.Position{Line: 1, Column: 10, Offset: 10},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
								EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					{
						IsResource: true,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "R",
								Pos:        ast.Position{Line: 1, Column: 17, Offset: 17},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
					},
					{
						IsResource: false,
						Type: &ast.VariableSizedType{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "S",
									Pos:        ast.Position{Line: 1, Column: 23, Offset: 23},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 21, Offset: 21},
								EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 21, Offset: 21},
					},
				},
				Arguments: []*ast.Argument{
					{
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 31, Offset: 31},
								EndPos:   ast.Position{Line: 1, Column: 31, Offset: 31},
							},
						},
						TrailingSeparatorPos: ast.Position{Line: 1, Column: 33, Offset: 33},
					},
					{
						Expression: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 35, Offset: 35},
								EndPos:   ast.Position{Line: 1, Column: 35, Offset: 35},
							},
						},
						TrailingSeparatorPos: ast.Position{Line: 1, Column: 37, Offset: 37},
					},
				},
				ArgumentsStartPos: ast.Position{Line: 1, Column: 29, Offset: 29},
				EndPos:            ast.Position{Line: 1, Column: 37, Offset: 37},
			},
			result,
		)
	})

	t.Run("precedence, invocation in binary expression", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("1 + a<>()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Right: &ast.InvocationExpression{
					InvokedExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "a",
							Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					TypeArguments:     nil,
					Arguments:         nil,
					ArgumentsStartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
					EndPos:            ast.Position{Line: 1, Column: 8, Offset: 8},
				},
			},
			result,
		)
	})

	t.Run("invocation, one type argument, nested type, no spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("a<T<U>>()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.InstantiationType{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "T",
									Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
								},
							},
							TypeArguments: []*ast.TypeAnnotation{
								{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "U",
											Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
										},
									},
									StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
								},
							},
							TypeArgumentsStartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:                ast.Position{Line: 1, Column: 5, Offset: 5},
						},
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				ArgumentsStartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
				EndPos:            ast.Position{Line: 1, Column: 8, Offset: 8},
			},
			result,
		)
	})

	t.Run("invocation, one type argument, nested type, spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("a<T< U > >()")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.InstantiationType{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "T",
									Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
								},
							},
							TypeArguments: []*ast.TypeAnnotation{
								{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "U",
											Pos:        ast.Position{Line: 1, Column: 5, Offset: 5},
										},
									},
									StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
								},
							},
							TypeArgumentsStartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:                ast.Position{Line: 1, Column: 7, Offset: 7},
						},
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				ArgumentsStartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
				EndPos:            ast.Position{Line: 1, Column: 11, Offset: 11},
			},
			result,
		)
	})

	t.Run("precedence, binary expressions, less than", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("0 + 1 < 2")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationLess,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("0"),
						Value:           new(big.Int),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("2"),
					Value:           big.NewInt(2),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
					},
				},
			},
			result,
		)
	})

	t.Run("precedence, binary expressions, left shift", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("0 + 1 << 2")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationBitwiseLeftShift,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("0"),
						Value:           new(big.Int),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("2"),
					Value:           big.NewInt(2),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
						EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
					},
				},
			},
			result,
		)
	})

	t.Run("precedence, binary expressions, greater than", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("0 + 1 > 2")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationGreater,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("0"),
						Value:           new(big.Int),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("2"),
					Value:           big.NewInt(2),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
					},
				},
			},
			result,
		)
	})

	t.Run("precedence, binary expressions, right shift", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression("0 + 1 >> 2")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationBitwiseRightShift,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("0"),
						Value:           new(big.Int),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					PositiveLiteral: []byte("2"),
					Value:           big.NewInt(2),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
						EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
					},
				},
			},
			result,
		)
	})
}

func TestParseBoolExpression(t *testing.T) {

	t.Parallel()

	const code = `
	    let a = true
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.BoolExpression{
					Value: true,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseIdentifierExpression(t *testing.T) {

	t.Parallel()

	const code = `
	    let b = a
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "b",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
						Pos:        ast.Position{Offset: 14, Line: 2, Column: 13},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseArrayExpressionInVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
	    let a = [1, 2]
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{Identifier: "a",
					Pos: ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.ArrayExpression{
					Values: []ast.Expression{
						&ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
								EndPos:   ast.Position{Offset: 15, Line: 2, Column: 14},
							},
						},
						&ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
								EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 19, Line: 2, Column: 18},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseDictionaryExpressionInVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
	    let x = {"a": 1, "b": 2}
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{Identifier: "x",
					Pos: ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.DictionaryExpression{
					Entries: []ast.DictionaryEntry{
						{
							Key: &ast.StringExpression{
								Value: "a",
								Range: ast.Range{
									StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
									EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
								},
							},
							Value: &ast.IntegerExpression{
								PositiveLiteral: []byte("1"),
								Value:           big.NewInt(1),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
									EndPos:   ast.Position{Offset: 20, Line: 2, Column: 19},
								},
							},
						},
						{
							Key: &ast.StringExpression{
								Value: "b",
								Range: ast.Range{
									StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
									EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
								},
							},
							Value: &ast.IntegerExpression{
								PositiveLiteral: []byte("2"),
								Value:           big.NewInt(2),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 28, Line: 2, Column: 27},
									EndPos:   ast.Position{Offset: 28, Line: 2, Column: 27},
								},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 29, Line: 2, Column: 28},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseInvocationExpressionWithoutLabels(t *testing.T) {

	t.Parallel()

	const code = `
	    let a = b(1, 2)
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.InvocationExpression{
					InvokedExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Offset: 14, Line: 2, Column: 13},
						},
					},
					Arguments: []*ast.Argument{
						{
							Label: "",
							Expression: &ast.IntegerExpression{
								PositiveLiteral: []byte("1"),
								Value:           big.NewInt(1),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
									EndPos:   ast.Position{Offset: 16, Line: 2, Column: 15},
								},
							},
							TrailingSeparatorPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						},
						{
							Label: "",
							Expression: &ast.IntegerExpression{
								PositiveLiteral: []byte("2"),
								Value:           big.NewInt(2),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 19, Line: 2, Column: 18},
									EndPos:   ast.Position{Offset: 19, Line: 2, Column: 18},
								},
							},
							TrailingSeparatorPos: ast.Position{Offset: 20, Line: 2, Column: 19},
						},
					},
					ArgumentsStartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
					EndPos:            ast.Position{Offset: 20, Line: 2, Column: 19},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseInvocationExpressionWithLabels(t *testing.T) {

	t.Parallel()

	const code = `
	    let a = b(x: 1, y: 2, z : 3)
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.InvocationExpression{
					InvokedExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Offset: 14, Line: 2, Column: 13},
						},
					},
					Arguments: []*ast.Argument{
						{
							Label:         "x",
							LabelStartPos: &ast.Position{Offset: 16, Line: 2, Column: 15},
							LabelEndPos:   &ast.Position{Offset: 17, Line: 2, Column: 16},
							Expression: &ast.IntegerExpression{
								PositiveLiteral: []byte("1"),
								Value:           big.NewInt(1),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 19, Line: 2, Column: 18},
									EndPos:   ast.Position{Offset: 19, Line: 2, Column: 18},
								},
							},
							TrailingSeparatorPos: ast.Position{Offset: 20, Line: 2, Column: 19},
						},
						{
							Label:         "y",
							LabelStartPos: &ast.Position{Offset: 22, Line: 2, Column: 21},
							LabelEndPos:   &ast.Position{Offset: 23, Line: 2, Column: 22},
							Expression: &ast.IntegerExpression{
								PositiveLiteral: []byte("2"),
								Value:           big.NewInt(2),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
									EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
								},
							},
							TrailingSeparatorPos: ast.Position{Offset: 26, Line: 2, Column: 25},
						},
						{
							Label:         "z",
							LabelStartPos: &ast.Position{Offset: 28, Line: 2, Column: 27},
							LabelEndPos:   &ast.Position{Offset: 30, Line: 2, Column: 29},
							Expression: &ast.IntegerExpression{
								PositiveLiteral: []byte("3"),
								Value:           big.NewInt(3),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 32, Line: 2, Column: 31},
									EndPos:   ast.Position{Offset: 32, Line: 2, Column: 31},
								},
							},
							TrailingSeparatorPos: ast.Position{Offset: 33, Line: 2, Column: 32},
						},
					},
					ArgumentsStartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
					EndPos:            ast.Position{Offset: 33, Line: 2, Column: 32},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseOptionalMemberExpression(t *testing.T) {

	t.Parallel()

	const code = `
	    let a = b?.c
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.MemberExpression{
					Optional: true,
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Offset: 14, Line: 2, Column: 13},
						},
					},
					Identifier: ast.Identifier{
						Identifier: "c",
						Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
					},
					AccessPos: ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseIndexExpressionInVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
	    let a = b[1]
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.IndexExpression{
					TargetExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Offset: 14, Line: 2, Column: 13},
						},
					},
					IndexingExpression: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
							EndPos:   ast.Position{Offset: 16, Line: 2, Column: 15},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
						EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseUnaryExpression(t *testing.T) {

	t.Parallel()

	const code = `
	    let foo = -boo
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "foo",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 14, Line: 2, Column: 13},
				},
				Value: &ast.UnaryExpression{
					Operation: ast.OperationMinus,
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "boo",
							Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
						},
					},
					StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseOrExpression(t *testing.T) {

	t.Parallel()

	const code = `
        let a = false || true
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationOr,
					Left: &ast.BoolExpression{
						Value: false,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
						},
					},
					Right: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:   ast.Position{Offset: 29, Line: 2, Column: 28},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseAndExpression(t *testing.T) {

	t.Parallel()

	const code = `
        let a = false && true
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationAnd,
					Left: &ast.BoolExpression{
						Value: false,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
						},
					},
					Right: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:   ast.Position{Offset: 29, Line: 2, Column: 28},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseEqualityExpression(t *testing.T) {

	t.Parallel()

	const code = `
        let a = false == true
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationEqual,
					Left: &ast.BoolExpression{
						Value: false,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
						},
					},
					Right: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:   ast.Position{Offset: 29, Line: 2, Column: 28},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseRelationalExpression(t *testing.T) {

	t.Parallel()

	const code = `
        let a = 1 < 2
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationLess,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseAdditiveExpression(t *testing.T) {

	t.Parallel()

	const code = `
        let a = 1 + 2
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseMultiplicativeExpression(t *testing.T) {

	t.Parallel()

	const code = `
        let a = 1 * 2
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationMul,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("2"),
						Value:           big.NewInt(2),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseFunctionExpressionAndReturn(t *testing.T) {

	t.Parallel()

	const code = `
	    let test = fun (): Int { return 1 }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.FunctionExpression{
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							EndPos:   ast.Position{Offset: 22, Line: 2, Column: 21},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "Int",
								Pos:        ast.Position{Offset: 25, Line: 2, Column: 24},
							},
						},
						StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
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
											StartPos: ast.Position{Offset: 38, Line: 2, Column: 37},
											EndPos:   ast.Position{Offset: 38, Line: 2, Column: 37},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 31, Line: 2, Column: 30},
										EndPos:   ast.Position{Offset: 38, Line: 2, Column: 37},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 29, Line: 2, Column: 28},
								EndPos:   ast.Position{Offset: 40, Line: 2, Column: 39},
							},
						},
					},
					StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseLeftAssociativity(t *testing.T) {

	t.Parallel()

	const code = `
        let a = 1 + 2 + 3
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.BinaryExpression{
						Operation: ast.OperationPlus,
						Left: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
								EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
							},
						},
						Right: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
								EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
							},
						},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("3"),
						Value:           big.NewInt(3),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
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

func TestParseNegativeInteger(t *testing.T) {

	t.Parallel()

	const code = `
      let a = -42
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 11, Line: 2, Column: 10},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Value: &ast.IntegerExpression{
					PositiveLiteral: []byte("42"),
					Value:           big.NewInt(-42),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
						EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
					},
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

func TestParseNegativeFixedPoint(t *testing.T) {

	t.Parallel()

	const code = `
      let a = -42.3
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 11, Line: 2, Column: 10},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Value: &ast.FixedPointExpression{
					PositiveLiteral: []byte("42.3"),
					Negative:        true,
					UnsignedInteger: big.NewInt(42),
					Fractional:      big.NewInt(3),
					Scale:           1,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
						EndPos:   ast.Position{Offset: 19, Line: 2, Column: 18},
					},
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

func TestParseTernaryRightAssociativity(t *testing.T) {

	t.Parallel()

	const code = `
        let a = 2 > 1
          ? 0
          : 3 > 2 ? 1 : 2
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.ConditionalExpression{
					Test: &ast.BinaryExpression{
						Operation: ast.OperationGreater,
						Left: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
								EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
							},
						},
						Right: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
								EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
							},
						},
					},
					Then: &ast.IntegerExpression{
						PositiveLiteral: []byte("0"),
						Value:           new(big.Int),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 35, Line: 3, Column: 12},
							EndPos:   ast.Position{Offset: 35, Line: 3, Column: 12},
						},
					},
					Else: &ast.ConditionalExpression{
						Test: &ast.BinaryExpression{
							Operation: ast.OperationGreater,
							Left: &ast.IntegerExpression{
								PositiveLiteral: []byte("3"),
								Value:           big.NewInt(3),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 49, Line: 4, Column: 12},
									EndPos:   ast.Position{Offset: 49, Line: 4, Column: 12},
								},
							},
							Right: &ast.IntegerExpression{
								PositiveLiteral: []byte("2"),
								Value:           big.NewInt(2),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 53, Line: 4, Column: 16},
									EndPos:   ast.Position{Offset: 53, Line: 4, Column: 16},
								},
							},
						},
						Then: &ast.IntegerExpression{
							PositiveLiteral: []byte("1"),
							Value:           big.NewInt(1),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 57, Line: 4, Column: 20},
								EndPos:   ast.Position{Offset: 57, Line: 4, Column: 20},
							},
						},
						Else: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 61, Line: 4, Column: 24},
								EndPos:   ast.Position{Offset: 61, Line: 4, Column: 24},
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

func TestParseVoidLiteral(t *testing.T) {
	t.Parallel()

	const code = `
		let void: Void = ()
	`

	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "void",
					Pos:        ast.Position{Offset: 7, Line: 2, Column: 6},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Void",
							Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
						},
						NestedIdentifiers: nil,
					},
					StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Value: &ast.VoidExpression{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
						EndPos:   ast.Position{Offset: 23, Line: 3, Column: 0},
					},
				},
				Transfer: &ast.Transfer{
					Operation: 1,
					Pos:       ast.Position{Offset: 18, Line: 2, Column: 17},
				},
				StartPos: ast.Position{Offset: 3, Line: 2, Column: 2},
			},
		}, result.Declarations())
}

func TestParseMissingReturnType(t *testing.T) {

	t.Parallel()

	const code = `
		let noop: fun(): Void =
            fun () { return }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "noop",
					Pos:        ast.Position{Offset: 7, Line: 2, Column: 6},
				},

				IsConstant: true,
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.FunctionType{
						ReturnTypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Void",
									Pos:        ast.Position{Offset: 20, Line: 2, Column: 19},
								},
							},
							StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
						},
					},
					StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 25, Line: 2, Column: 24},
				},
				Value: &ast.FunctionExpression{
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 43, Line: 3, Column: 16},
							EndPos:   ast.Position{Offset: 44, Line: 3, Column: 17},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Statements: []ast.Statement{
								&ast.ReturnStatement{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 48, Line: 3, Column: 21},
										EndPos:   ast.Position{Offset: 53, Line: 3, Column: 26},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 46, Line: 3, Column: 19},
								EndPos:   ast.Position{Offset: 55, Line: 3, Column: 28},
							},
						},
					},
					StartPos: ast.Position{Offset: 39, Line: 3, Column: 12},
				},
				StartPos: ast.Position{Offset: 3, Line: 2, Column: 2},
			},
		},
		result.Declarations(),
	)
}

func TestParseExpression(t *testing.T) {

	t.Parallel()

	actual, errs := testParseExpression(`
        before(x + before(y)) + z
	`)
	var err error
	if len(errs) > 0 {
		err = Error{
			Errors: errs,
		}
	}
	require.NoError(t, err)

	expected := &ast.BinaryExpression{
		Operation: ast.OperationPlus,
		Left: &ast.InvocationExpression{
			InvokedExpression: &ast.IdentifierExpression{
				Identifier: ast.Identifier{
					Identifier: "before",
					Pos:        ast.Position{Offset: 9, Line: 2, Column: 8},
				},
			},
			Arguments: []*ast.Argument{
				{
					Label:         "",
					LabelStartPos: nil,
					LabelEndPos:   nil,
					Expression: &ast.BinaryExpression{
						Operation: ast.OperationPlus,
						Left: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
							},
						},
						Right: &ast.InvocationExpression{
							InvokedExpression: &ast.IdentifierExpression{
								Identifier: ast.Identifier{
									Identifier: "before",
									Pos:        ast.Position{Offset: 20, Line: 2, Column: 19},
								},
							},
							Arguments: []*ast.Argument{
								{
									Label:         "",
									LabelStartPos: nil,
									LabelEndPos:   nil,
									Expression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "y",
											Pos:        ast.Position{Offset: 27, Line: 2, Column: 26},
										},
									},
									TrailingSeparatorPos: ast.Position{Offset: 28, Line: 2, Column: 27},
								},
							},
							ArgumentsStartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:            ast.Position{Offset: 28, Line: 2, Column: 27},
						},
					},
					TrailingSeparatorPos: ast.Position{Offset: 29, Line: 2, Column: 28},
				},
			},
			ArgumentsStartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
			EndPos:            ast.Position{Offset: 29, Line: 2, Column: 28},
		},
		Right: &ast.IdentifierExpression{
			Identifier: ast.Identifier{
				Identifier: "z",
				Pos:        ast.Position{Offset: 33, Line: 2, Column: 32},
			},
		},
	}

	utils.AssertEqualWithDiff(t, expected, actual)
}

func TestParseStringEscapes(t *testing.T) {

	t.Parallel()

	actual, errs := testParseExpression(`
       "test \0\n\r\t\"\'\\ xyz"
	`)

	var err error
	if len(errs) > 0 {
		err = Error{
			Errors: errs,
		}
	}

	require.NoError(t, err)

	expected := &ast.StringExpression{
		Value: "test \x00\n\r\t\"'\\ xyz",
		Range: ast.Range{
			StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			EndPos:   ast.Position{Offset: 32, Line: 2, Column: 31},
		},
	}

	utils.AssertEqualWithDiff(t, expected, actual)
}

func TestParseStringWithUnicode(t *testing.T) {

	t.Parallel()

	actual, errs := testParseExpression(`
      "this is a test \t\\new line and race car:\n\u{1F3CE}\u{FE0F}"
	`)

	var err error
	if len(errs) > 0 {
		err = Error{
			Errors: errs,
		}
	}

	require.NoError(t, err)

	expected := &ast.StringExpression{
		Value: "this is a test \t\\new line and race car:\n\U0001F3CE\uFE0F",
		Range: ast.Range{
			StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			EndPos:   ast.Position{Offset: 68, Line: 2, Column: 67},
		},
	}

	utils.AssertEqualWithDiff(t, expected, actual)
}

func TestParseNilCoalescing(t *testing.T) {

	t.Parallel()

	const code = `
       let x = nil ?? 1
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
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 14, Line: 2, Column: 13},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationNilCoalesce,
					Left: &ast.NilExpression{
						Pos: ast.Position{Offset: 16, Line: 2, Column: 15},
					},
					Right: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
							EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
						},
					},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result.Declarations(),
	)
}

func TestParseNilCoalescingRightAssociativity(t *testing.T) {

	t.Parallel()

	// NOTE: only syntactically, not semantically valid
	const code = `
       let x = 1 ?? 2 ?? 3
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
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 14, Line: 2, Column: 13},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationNilCoalesce,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
							EndPos:   ast.Position{Offset: 16, Line: 2, Column: 15},
						},
					},
					Right: &ast.BinaryExpression{
						Operation: ast.OperationNilCoalesce,
						Left: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
								EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
							},
						},
						Right: &ast.IntegerExpression{
							PositiveLiteral: []byte("3"),
							Value:           big.NewInt(3),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
								EndPos:   ast.Position{Offset: 26, Line: 2, Column: 25},
							},
						},
					},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result.Declarations(),
	)
}

func TestParseFailableCasting(t *testing.T) {

	t.Parallel()

	const code = `
       let x = 0 as? Int
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	failableDowncast := &ast.CastingExpression{
		Expression: &ast.IntegerExpression{
			PositiveLiteral: []byte("0"),
			Value:           new(big.Int),
			Base:            10,
			Range: ast.Range{
				StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
				EndPos:   ast.Position{Offset: 16, Line: 2, Column: 15},
			},
		},
		Operation: ast.OperationFailableCast,
		TypeAnnotation: &ast.TypeAnnotation{
			IsResource: false,
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "Int",
					Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
				},
			},
			StartPos: ast.Position{Offset: 22, Line: 2, Column: 21},
		},
	}

	variableDeclaration := &ast.VariableDeclaration{
		Access:     ast.AccessNotSpecified,
		IsConstant: true,
		Identifier: ast.Identifier{
			Identifier: "x",
			Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 14, Line: 2, Column: 13},
		},
		Value:    failableDowncast,
		StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
	}

	failableDowncast.ParentVariableDeclaration = variableDeclaration

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			variableDeclaration,
		},
		result.Declarations(),
	)
}

func TestParseMoveOperator(t *testing.T) {

	t.Parallel()

	const code = `
      let x = foo(<-y)
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
					Pos:        ast.Position{Offset: 11, Line: 2, Column: 10},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Value: &ast.InvocationExpression{
					InvokedExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "foo",
							Pos:        ast.Position{Offset: 15, Line: 2, Column: 14},
						},
					},
					Arguments: []*ast.Argument{
						{
							Label:         "",
							LabelStartPos: nil,
							LabelEndPos:   nil,
							Expression: &ast.UnaryExpression{
								Operation: ast.OperationMove,
								Expression: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "y",
										Pos:        ast.Position{Offset: 21, Line: 2, Column: 20},
									},
								},
								StartPos: ast.Position{Offset: 19, Line: 2, Column: 18},
							},
							TrailingSeparatorPos: ast.Position{Offset: 22, Line: 2, Column: 21},
						},
					},
					ArgumentsStartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
					EndPos:            ast.Position{Offset: 22, Line: 2, Column: 21},
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

func TestParseFunctionExpressionWithResourceTypeAnnotation(t *testing.T) {

	t.Parallel()

	const code = `
        let f = fun (): @R { return X }
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{

			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "f",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.FunctionExpression{
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							EndPos:   ast.Position{Offset: 22, Line: 2, Column: 21},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						IsResource: true,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "R",
								Pos:        ast.Position{Offset: 26, Line: 2, Column: 25},
							},
						},
						StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Statements: []ast.Statement{
								&ast.ReturnStatement{
									Expression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "X",
											Pos:        ast.Position{Offset: 37, Line: 2, Column: 36},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 30, Line: 2, Column: 29},
										EndPos:   ast.Position{Offset: 37, Line: 2, Column: 36},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 28, Line: 2, Column: 27},
								EndPos:   ast.Position{Offset: 39, Line: 2, Column: 38},
							},
						},
					},
					StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseFailableCastingResourceTypeAnnotation(t *testing.T) {

	t.Parallel()

	const code = `
        let y = x as? @R
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	failableDowncast := &ast.CastingExpression{
		Expression: &ast.IdentifierExpression{
			Identifier: ast.Identifier{
				Identifier: "x",
				Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
			},
		},
		Operation: ast.OperationFailableCast,
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
	}

	variableDeclaration := &ast.VariableDeclaration{
		Access:     ast.AccessNotSpecified,
		IsConstant: true,
		Identifier: ast.Identifier{
			Identifier: "y",
			Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
		},
		Value:    failableDowncast,
		StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
	}

	failableDowncast.ParentVariableDeclaration = variableDeclaration

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			variableDeclaration,
		},
		result.Declarations(),
	)
}

func TestParseCasting(t *testing.T) {

	t.Parallel()

	const code = `
        let y = x as Y
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	cast := &ast.CastingExpression{
		Expression: &ast.IdentifierExpression{
			Identifier: ast.Identifier{
				Identifier: "x",
				Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
			},
		},
		Operation: ast.OperationCast,
		TypeAnnotation: &ast.TypeAnnotation{
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "Y",
					Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
				},
			},
			StartPos: ast.Position{Offset: 22, Line: 2, Column: 21},
		},
	}

	variableDeclaration := &ast.VariableDeclaration{
		Access:     ast.AccessNotSpecified,
		IsConstant: true,
		Identifier: ast.Identifier{
			Identifier: "y",
			Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
		},
		Value:    cast,
		StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
	}

	cast.ParentVariableDeclaration = variableDeclaration

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			variableDeclaration,
		},
		result.Declarations(),
	)
}

func testParseIdentifiersWith(t *testing.T, identifiers []string, condition func(*testing.T, string, error)) {
	for _, identifier := range identifiers {
		// to ensure proper name capture
		name := identifier
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`let %s = 0`, name)
			_, errs := testParseProgram(code)
			condition(t, name, errs)
		})
	}
}

func TestParseIdentifiers(t *testing.T) {

	t.Parallel()

	names := []string{
		"foo",
		"_foo",
		"foo123",
		"foo________",
		"FOO_______",
		"Fo123__21341278AAAAAAAAAAAAA",
	}

	testParseIdentifiersWith(t, names, func(t *testing.T, _ string, errs error) {
		require.Empty(t, errs)
	})
}

func TestParseHardKeywords(t *testing.T) {
	t.Parallel()

	testParseIdentifiersWith(t, hardKeywords, func(t *testing.T, keyword string, err error) {
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Pos:     ast.Position{Line: 1, Column: 4, Offset: 4},
					Message: "expected identifier after start of variable declaration, got keyword " + keyword,
				},
			},
			err.(Error).Errors,
		)
	})
}

func TestParseSoftKeywords(t *testing.T) {
	t.Parallel()

	testParseIdentifiersWith(t, softKeywords, func(t *testing.T, _ string, err error) {
		require.Empty(t, err)
	})
}

func TestParseReferenceInVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
       let x = &account.storage[R] as &R
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	expected := &ast.VariableDeclaration{
		Access:     ast.AccessNotSpecified,
		IsConstant: true,
		Identifier: ast.Identifier{
			Identifier: "x",
			Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
		},
		Value: &ast.CastingExpression{
			Operation: ast.OperationCast,
			Expression: &ast.ReferenceExpression{
				Expression: &ast.IndexExpression{
					TargetExpression: &ast.MemberExpression{
						Expression: &ast.IdentifierExpression{
							Identifier: ast.Identifier{
								Identifier: "account",
								Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
							},
						},
						AccessPos: ast.Position{Offset: 24, Line: 2, Column: 23},
						Identifier: ast.Identifier{
							Identifier: "storage",
							Pos:        ast.Position{Offset: 25, Line: 2, Column: 24},
						},
					},
					IndexingExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "R",
							Pos:        ast.Position{Offset: 33, Line: 2, Column: 32},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 32, Line: 2, Column: 31},
						EndPos:   ast.Position{Offset: 34, Line: 2, Column: 33},
					},
				},
				StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
			},
			TypeAnnotation: &ast.TypeAnnotation{
				Type: &ast.ReferenceType{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "R",
							Pos:        ast.Position{Offset: 40, Line: 2, Column: 39},
						},
					},
					StartPos: ast.Position{Offset: 39, Line: 2, Column: 38},
				},
				StartPos: ast.Position{Offset: 39, Line: 2, Column: 38},
			},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 14, Line: 2, Column: 13},
		},
		StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
	}

	expected.Value.(*ast.CastingExpression).ParentVariableDeclaration = expected

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			expected,
		},
		result.Declarations(),
	)
}

func TestParseFixedPointExpression(t *testing.T) {

	t.Parallel()

	const code = `
	    let a = -1234_5678_90.0009_8765_4321
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{Identifier: "a",
					Pos: ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.FixedPointExpression{
					PositiveLiteral: []byte("1234_5678_90.0009_8765_4321"),
					Negative:        true,
					UnsignedInteger: big.NewInt(1234567890),
					Fractional:      big.NewInt(987654321),
					Scale:           12,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 41, Line: 2, Column: 40},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseFixedPointExpressionZeroInteger(t *testing.T) {

	t.Parallel()

	const code = `
	    let a = -0.1
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{Identifier: "a",
					Pos: ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.FixedPointExpression{
					PositiveLiteral: []byte("0.1"),
					Negative:        true,
					UnsignedInteger: new(big.Int),
					Fractional:      big.NewInt(1),
					Scale:           1,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParsePathLiteral(t *testing.T) {

	t.Parallel()

	const code = `
	    let a = /foo/bar
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{Identifier: "a",
					Pos: ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Value: &ast.PathExpression{
					StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
					Domain: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Offset: 15, Line: 2, Column: 14},
					},
					Identifier: ast.Identifier{
						Identifier: "bar",
						Pos:        ast.Position{Offset: 19, Line: 2, Column: 18},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseBitwiseExpression(t *testing.T) {

	t.Parallel()

	const code = `
      let a = 1 | 2 ^ 3 & 4 << 5 >> 6
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 11, Line: 2, Column: 10},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Value: &ast.BinaryExpression{
					Operation: ast.OperationBitwiseOr,
					Left: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
							EndPos:   ast.Position{Offset: 15, Line: 2, Column: 14},
						},
					},
					Right: &ast.BinaryExpression{
						Operation: ast.OperationBitwiseXor,
						Left: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 19, Line: 2, Column: 18},
								EndPos:   ast.Position{Offset: 19, Line: 2, Column: 18},
							},
						},
						Right: &ast.BinaryExpression{
							Operation: ast.OperationBitwiseAnd,
							Left: &ast.IntegerExpression{
								PositiveLiteral: []byte("3"),
								Value:           big.NewInt(3),
								Base:            10,
								Range: ast.Range{
									StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
									EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
								},
							},
							Right: &ast.BinaryExpression{
								Operation: ast.OperationBitwiseRightShift,
								Left: &ast.BinaryExpression{
									Operation: ast.OperationBitwiseLeftShift,
									Left: &ast.IntegerExpression{
										PositiveLiteral: []byte("4"),
										Value:           big.NewInt(4),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 27, Line: 2, Column: 26},
											EndPos:   ast.Position{Offset: 27, Line: 2, Column: 26},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("5"),
										Value:           big.NewInt(5),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 32, Line: 2, Column: 31},
											EndPos:   ast.Position{Offset: 32, Line: 2, Column: 31},
										},
									},
								},
								Right: &ast.IntegerExpression{
									PositiveLiteral: []byte("6"),
									Value:           big.NewInt(6),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 37, Line: 2, Column: 36},
										EndPos:   ast.Position{Offset: 37, Line: 2, Column: 36},
									},
								},
							},
						},
					},
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

func TestParseInvalidNegativeIntegerLiteralWithIncorrectPrefix(t *testing.T) {

	t.Parallel()

	const code = `
	    let e = -0K0
	`
	_, err := testParseProgram(code)

	require.Error(t, err)
}

type limitingMemoryGauge struct {
	limited map[common.MemoryKind]bool   // which kinds to limit
	limits  map[common.MemoryKind]uint64 // limits of limited kinds
	totals  map[common.MemoryKind]uint64 // metered memory. for debugging
	debug   bool                         // print totals after each allocation
}

type limitingMemoryGaugeError string

func (e limitingMemoryGaugeError) Error() string {
	return string(e)
}

func makeLimitingMemoryGauge() *limitingMemoryGauge {
	g := limitingMemoryGauge{
		limited: make(map[common.MemoryKind]bool),
		limits:  make(map[common.MemoryKind]uint64),
		totals:  make(map[common.MemoryKind]uint64),
	}
	return &g
}

func (g *limitingMemoryGauge) Limit(kind common.MemoryKind, limit uint64) {
	g.limited[kind] = true
	g.limits[kind] = limit
}

func (g *limitingMemoryGauge) MeterMemory(usage common.MemoryUsage) error {
	g.totals[usage.Kind] += usage.Amount

	if g.debug {
		fmt.Println(g.totals)
	}

	if !g.limited[usage.Kind] {
		return nil
	}

	if g.limits[usage.Kind] < usage.Amount {
		return limitingMemoryGaugeError(fmt.Sprintf(`reached limit for "%s"`, usage.Kind.String()))
	}

	g.limits[usage.Kind] -= usage.Amount

	return nil
}
