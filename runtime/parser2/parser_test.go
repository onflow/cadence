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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
)

func TestParseExpression(t *testing.T) {

	expectedSimpleExpression := &ast.BinaryExpression{
		Operation: ast.OperationPlus,
		Left: &ast.IntegerExpression{
			Value: big.NewInt(1),
			Base:  10,
		},
		Right: &ast.BinaryExpression{
			Operation: ast.OperationMul,
			Left: &ast.IntegerExpression{
				Value: big.NewInt(2),
				Base:  10,
			},
			Right: &ast.IntegerExpression{
				Value: big.NewInt(3),
				Base:  10,
			},
		},
	}

	t.Run("simple, no spaces", func(t *testing.T) {
		result, errors := Parse("1+2*3")
		require.Empty(t, errors)

		assert.Equal(t,
			expectedSimpleExpression,
			result,
		)
	})

	t.Run("simple, spaces", func(t *testing.T) {
		result, errors := Parse("  1   +   2  *   3 ")
		require.Empty(t, errors)

		assert.Equal(t,
			expectedSimpleExpression,
			result,
		)
	})

	t.Run("repeated infix, same operator, left associative", func(t *testing.T) {
		result, errors := Parse("1 + 2 + 3")
		require.Empty(t, errors)

		assert.Equal(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
				},
			},
			result,
		)
	})

	t.Run("repeated infix, same operator, right associative", func(t *testing.T) {
		result, errors := Parse("1 ?? 2 ?? 3")
		require.Empty(t, errors)

		assert.Equal(t,
			&ast.BinaryExpression{
				Operation: ast.OperationNilCoalesce,
				Left: &ast.IntegerExpression{
					Value: big.NewInt(1),
					Base:  10,
				},
				Right: &ast.BinaryExpression{
					Operation: ast.OperationNilCoalesce,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
					},
				},
			},
			result,
		)
	})

	t.Run("mixed infix and prefix", func(t *testing.T) {
		result, errors := Parse("1 +- 2 ++ 3")
		require.Empty(t, errors)

		assert.Equal(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
					},
					Right: &ast.UnaryExpression{
						Operation: ast.OperationMinus,
						Expression: &ast.IntegerExpression{
							Value: big.NewInt(2),
							Base:  10,
						},
					},
				},
				Right: &ast.UnaryExpression{
					Operation: ast.OperationPlus,
					Expression: &ast.IntegerExpression{
						Value: big.NewInt(3),
						Base:  10,
					},
				},
			},
			result,
		)
	})

	t.Run("nested expression", func(t *testing.T) {
		result, errors := Parse("(1 + 2) * 3")
		require.Empty(t, errors)

		assert.Equal(t,
			&ast.BinaryExpression{
				Operation: ast.OperationMul,
				Left: &ast.BinaryExpression{
					Operation: ast.OperationPlus,
					Left: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
					},
					Right: &ast.IntegerExpression{
						Value: big.NewInt(2),
						Base:  10,
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
				},
			},
			result,
		)
	})

	t.Run("array expression", func(t *testing.T) {
		result, errors := Parse("[ 1,2 + 3, 4  ,  5 ]")
		require.Empty(t, errors)

		assert.Equal(t,
			&ast.ArrayExpression{
				Values: []ast.Expression{
					&ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
					},
					&ast.BinaryExpression{
						Operation: ast.OperationPlus,
						Left: &ast.IntegerExpression{
							Value: big.NewInt(2),
							Base:  10,
						},
						Right: &ast.IntegerExpression{
							Value: big.NewInt(3),
							Base:  10,
						},
					},
					&ast.IntegerExpression{
						Value: big.NewInt(4),
						Base:  10,
					},
					&ast.IntegerExpression{
						Value: big.NewInt(5),
						Base:  10,
					},
				},
			},
			result,
		)
	})

	t.Run("dictionary expression", func(t *testing.T) {
		result, errors := Parse("{ 1:2 + 3, 4  :  5 }")
		require.Empty(t, errors)

		assert.Equal(t,
			&ast.DictionaryExpression{
				Entries: []ast.Entry{
					{
						Key: &ast.IntegerExpression{
							Value: big.NewInt(1),
							Base:  10,
						},
						Value: &ast.BinaryExpression{
							Operation: ast.OperationPlus,
							Left: &ast.IntegerExpression{
								Value: big.NewInt(2),
								Base:  10,
							},
							Right: &ast.IntegerExpression{
								Value: big.NewInt(3),
								Base:  10,
							},
						},
					},
					{
						Key: &ast.IntegerExpression{
							Value: big.NewInt(4),
							Base:  10,
						},
						Value: &ast.IntegerExpression{
							Value: big.NewInt(5),
							Base:  10,
						},
					},
				},
			},
			result,
		)
	})

	t.Run("identifier in addition", func(t *testing.T) {
		result, errors := Parse("a + 3")
		require.Empty(t, errors)

		assert.Equal(t,
			&ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "a",
					},
				},
				Right: &ast.IntegerExpression{
					Value: big.NewInt(3),
					Base:  10,
				},
			},
			result,
		)
	})

	t.Run("path expression", func(t *testing.T) {
		result, errors := Parse("/foo/bar")
		require.Empty(t, errors)

		assert.Equal(t,
			&ast.PathExpression{
				Domain:     ast.Identifier{Identifier: "foo"},
				Identifier: ast.Identifier{Identifier: "bar"},
			},
			result,
		)
	})
}
