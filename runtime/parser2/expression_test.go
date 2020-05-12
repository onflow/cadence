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
	"errors"
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
	oldParser "github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/parser2/lexer"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestParseSimpleInfixExpression(t *testing.T) {

	t.Run("no spaces", func(t *testing.T) {
		result, errs := ParseExpression("1+2*3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IntegerExpression{
					Value: big.NewInt(1),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationMul,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
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
		result, errs := ParseExpression("  1   +   2  *   3 ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IntegerExpression{
					Value: big.NewInt(1),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
						EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationMul,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
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
		result, errs := ParseExpression("1 + 2 + 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
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
		result, errs := ParseExpression("1 ?? 2 ?? 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationNilCoalesce,
				Left: &ast.IntegerExpression{
					Value: big.NewInt(1),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationNilCoalesce,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
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

	t.Run("mixed infix and prefix", func(t *testing.T) {
		result, errs := ParseExpression("1 +- 2 ++ 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.UnaryExpression{
						Operation: ast.OperationMinus,
						Expression: &ast.IntegerExpression{
							Value: big.NewInt(2),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
								EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
					},
				},
				Right: &ast.UnaryExpression{
					Operation: ast.OperationPlus,
					Expression: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
				},
			},
			result,
		)
	})

	t.Run("nested expression", func(t *testing.T) {
		result, errs := ParseExpression("(1 + 2) * 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationMul,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
						EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("less and greater", func(t *testing.T) {
		result, errs := ParseExpression("1 < 2 > 3")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationGreater,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationLess,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
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
		result, errs := ParseExpression("a ? b : c ? d : e")
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
		result, errs := ParseExpression("true + false")
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
		result, errs := ParseExpression("(<-x)")
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

	t.Run("array expression", func(t *testing.T) {
		result, errs := ParseExpression("[ 1,2 + 3, 4  ,  5 ]")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ArrayExpression{
				Values: []ast.Expression{
					&ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					&ast.BinaryExpression{
						Operation: ast.OperationPlus,
						Left: &ast.IntegerExpression{
							Value: big.NewInt(2),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
								EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
							},
						},
						Right: &ast.IntegerExpression{
							Value: big.NewInt(3),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
								EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
							},
						},
					},
					&ast.IntegerExpression{
						Value: big.NewInt(4),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					&ast.IntegerExpression{
						Value: big.NewInt(5),
						Base:  10,
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
}

func TestParseDictionaryExpression(t *testing.T) {

	t.Run("dictionary expression", func(t *testing.T) {
		result, errs := ParseExpression("{ 1:2 + 3, 4  :  5 }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.DictionaryExpression{
				Entries: []ast.Entry{
					{
						Key: &ast.IntegerExpression{
							Value: big.NewInt(1),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
								EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
							},
						},
						Value: &ast.BinaryExpression{
							Operation: ast.OperationPlus,
							Left: &ast.IntegerExpression{
								Value: big.NewInt(2),
								Base:  10,
								Range: ast.Range{
									StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
									EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
								},
							},
							Right: &ast.IntegerExpression{
								Value: big.NewInt(3),
								Base:  10,
								Range: ast.Range{
									StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
									EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
								},
							},
						},
					},
					{
						Key: &ast.IntegerExpression{
							Value: big.NewInt(4),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
								EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
							},
						},
						Value: &ast.IntegerExpression{
							Value: big.NewInt(5),
							Base:  10,
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
}

func TestParseIdentifier(t *testing.T) {

	t.Run("identifier in addition", func(t *testing.T) {
		result, errs := ParseExpression("a + 3")
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
					Value: big.NewInt(3),
					Base:  10,
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

	result, errs := ParseExpression("/foo/bar")
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

	t.Run("valid, empty", func(t *testing.T) {
		result, errs := ParseExpression("\"\"")
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
		result, errs := ParseExpression("\"")
		assert.Equal(t,
			[]error{
				errors.New("invalid end of string literal: missing '\"'"),
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
		result, errs := ParseExpression("\"\n")
		assert.Equal(t,
			[]error{
				errors.New("invalid end of string literal: missing '\"'"),
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
		result, errs := ParseExpression("\"t")
		assert.Equal(t,
			[]error{
				errors.New("invalid end of string literal: missing '\"'"),
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
		result, errs := ParseExpression("\"t\n")
		assert.Equal(t,
			[]error{
				errors.New("invalid end of string literal: missing '\"'"),
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
		result, errs := ParseExpression("\"\\")
		assert.Equal(t,
			[]error{
				errors.New("incomplete escape sequence: missing character after escape character"),
				errors.New("invalid end of string literal: missing '\"'"),
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
		result, errs := ParseExpression(`"te\tst\"te\u{1F3CE}\u{FE0F}xt"`)
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
		result, errs := ParseExpression(`"te\Xst"`)
		assert.Equal(t,
			[]error{
				errors.New("invalid escape character: 'X'"),
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
		result, errs := ParseExpression(`"te\u`)
		assert.Equal(t,
			[]error{
				errors.New("incomplete Unicode escape sequence: missing character '{' after escape character"),
				errors.New("invalid end of string literal: missing '\"'"),
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
		result, errs := ParseExpression(`"te\us`)
		assert.Equal(t,
			[]error{
				errors.New("invalid Unicode escape sequence: expected '{', got 's'"),
				errors.New("invalid end of string literal: missing '\"'"),
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
		result, errs := ParseExpression(`"te\u{`)
		assert.Equal(t,
			[]error{
				errors.New("incomplete Unicode escape sequence: missing character '}' after escape character"),
				errors.New("invalid end of string literal: missing '\"'"),
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
		result, errs := ParseExpression(`"te\u{}"`)
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
		result, errs := ParseExpression(
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
		result, errs := ParseExpression(`"te\u{X}st"`)
		assert.Equal(t,
			[]error{
				errors.New("invalid Unicode escape sequence: expected hex digit, got 'X'"),
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

func TestInvocation(t *testing.T) {
	t.Run("invocation, no parameters", func(t *testing.T) {
		result, errors := ParseExpression("f()")
		require.Empty(t, errors)
		assert.Equal(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments: nil,
				EndPos:    ast.Position{Offset: 2, Line: 1, Column: 2},
			},
			result,
		)
	})
	t.Run("invocation, no parameters, with whitespace", func(t *testing.T) {
		result, errors := ParseExpression("f ()")
		require.Empty(t, errors)
		assert.Equal(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments: nil,
				EndPos:    ast.Position{Offset: 3, Line: 1, Column: 3},
			},
			result,
		)
	})
	t.Run("invocation, no parameters, with whitespace within params", func(t *testing.T) {
		result, errors := ParseExpression("f ( )")
		require.Empty(t, errors)
		assert.Equal(t,
			&ast.InvocationExpression{
				InvokedExpression: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "f",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				Arguments: nil,
				EndPos:    ast.Position{Offset: 4, Line: 1, Column: 4},
			},
			result,
		)
	})
	t.Run("invocation, with parameters", func(t *testing.T) {
		result, errors := ParseExpression("f(1)")
		require.Empty(t, errors)
		assert.Equal(t,
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
							Value: big.NewInt(1),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
					},
				},
				EndPos: ast.Position{Offset: 3, Line: 1, Column: 3},
			},
			result,
		)
	})
	t.Run("invocation, with parameters, multiple", func(t *testing.T) {
		result, errors := ParseExpression("f(1,2)")
		require.Empty(t, errors)
		assert.Equal(t,
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
							Value: big.NewInt(1),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
					},
					{
						Label: "",
						Expression: &ast.IntegerExpression{
							Value: big.NewInt(2),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
					},
				},
				EndPos: ast.Position{Offset: 5, Line: 1, Column: 5},
			},
			result,
		)
	})
	t.Run("invocation should error, no parameters, multiple commas", func(t *testing.T) {
		_, errors := ParseExpression("f(,,)")
		require.Error(t, errors[0], fmt.Errorf(
			"expected argument or end of argument list, got %q",
			lexer.TokenComma,
		))
	})
	t.Run("invocation should error, with parameter, multiple commas", func(t *testing.T) {
		_, errors := ParseExpression("f(1,,)")
		require.Error(t, errors[0], fmt.Errorf(
			"expected argument or end of argument list, got %q",
			lexer.TokenComma,
		))
	})
	t.Run("invocation should error, with multiple parameters, no commas", func(t *testing.T) {
		_, errors := ParseExpression("f(1 2)")
		require.NotNil(t, errors)
		require.Error(t, errors[0], fmt.Errorf("unexpected argument in argument list (expecting delimiter of end of argument list), got %q", lexer.TokenIdentifier))
	})
	t.Run("invocation, with parameters, nested", func(t *testing.T) {
		result, errors := ParseExpression("f(1,g(2))")
		require.Empty(t, errors)
		assert.Equal(t,
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
							Value: big.NewInt(1),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
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
										Value: big.NewInt(2),
										Base:  10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 6, Line: 1, Column: 6},
											EndPos:   ast.Position{Offset: 6, Line: 1, Column: 6},
										},
									},
								},
							},
							EndPos: ast.Position{Offset: 7, Line: 1, Column: 7},
						},
					},
				},
				EndPos: ast.Position{Offset: 8, Line: 1, Column: 8},
			},
			result,
		)
	})
	t.Run("invocation, with parameters, nested, string", func(t *testing.T) {
		result, errors := ParseExpression("f(1,g(\"test\"))")
		require.Empty(t, errors)
		assert.Equal(t,
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
							Value: big.NewInt(1),
							Base:  10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
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
								},
							},
							EndPos: ast.Position{Offset: 12, Line: 1, Column: 12},
						},
					},
				},
				EndPos: ast.Position{Offset: 13, Line: 1, Column: 13},
			},
			result,
		)
	})
}

func TestParseBlockComment(t *testing.T) {

	t.Run("nested comment, nothing else", func(t *testing.T) {

		result, errs := ParseExpression(" /* test  foo/* bar  */ asd*/ true")
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

		result, errs := ParseExpression(" /*test  foo*/ /* bar  */ true")
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

		result, errs := ParseExpression(" 1/*test  foo*/+/* bar  */ 2  ")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IntegerExpression{
					Value: big.NewInt(1),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(2),
					Base:  10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 27, Offset: 27},
						EndPos:   ast.Position{Line: 1, Column: 27, Offset: 27},
					},
				},
			},
			result,
		)
	})
}

func BenchmarkParseInfix(b *testing.B) {

	b.Run("new", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ParseExpression("(8 - 1 + 3) * 6 - ((3 + 7) * 2)")
		}
	})

	b.Run("old", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = oldParser.ParseExpression("(8 - 1 + 3) * 6 - ((3 + 7) * 2)")
		}
	})
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

	b.Run("new", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ParseExpression(lit)
		}
	})

	b.Run("old", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = oldParser.ParseExpression(lit)
		}
	})
}

func TestParseReference(t *testing.T) {

	result, errs := ParseExpression("& t as T")
	require.Empty(t, errs)

	utils.AssertEqualWithDiff(t,
		&ast.ReferenceExpression{
			Expression: &ast.IdentifierExpression{
				Identifier: ast.Identifier{
					Identifier: "t",
					Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
				},
			},
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "T",
					Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
				},
			},
			StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
		},
		result,
	)
}
