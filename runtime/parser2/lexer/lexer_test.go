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

package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/ast"
)

func TestLex(t *testing.T) {

	t.Run("single char number", func(t *testing.T) {
		assert.Equal(t,
			[]Token{
				{
					Type:  TokenNumber,
					Value: "0",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				{
					Type: TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
			},
			Lex("0"),
		)
	})

	t.Run("two char number", func(t *testing.T) {

		assert.Equal(t,
			[]Token{
				{
					Type:  TokenNumber,
					Value: "01",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				{
					Type: TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
						EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
			},
			Lex("01"),
		)
	})

	t.Run("two numbers separated by whitespace", func(t *testing.T) {

		assert.Equal(t,
			[]Token{
				{

					Type:  TokenSpace,
					Value: " ",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				{
					Type:  TokenNumber,
					Value: "01",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
					},
				},
				{
					Type:  TokenSpace,
					Value: "\t  ",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
						EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
					},
				},
				{
					Type:  TokenNumber,
					Value: "10",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
						EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
					},
				},
				{
					Type: TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
					},
				},
			},
			Lex(" 01\t  10"),
		)
	})

	t.Run("simple arithmetic", func(t *testing.T) {

		assert.Equal(t,
			[]Token{
				{
					Type: TokenParenOpen,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				{
					Type:  TokenNumber,
					Value: "2",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				{
					Type:  TokenSpace,
					Value: " ",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
						EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
					},
				},
				{
					Type: TokenOperatorPlus,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
						EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
				{
					Type:  TokenSpace,
					Value: " ",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
						EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
					},
				},
				{
					Type:  TokenNumber,
					Value: "3",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
						EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
					},
				},
				{
					Type: TokenParenClose,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
						EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
					},
				},
				{
					Type:  TokenSpace,
					Value: " ",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
						EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
					},
				},
				{
					Type: TokenOperatorMul,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
					},
				},
				{
					Type:  TokenSpace,
					Value: " ",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
						EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
					},
				},
				{
					Type:  TokenNumber,
					Value: "4",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
						EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
					},
				},
				{
					Type: TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
						EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
					},
				},
			},
			Lex("(2 + 3) * 4"),
		)
	})

	t.Run("multiple lines", func(t *testing.T) {

		assert.Equal(t,
			[]Token{
				{
					Type:  TokenNumber,
					Value: "1",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				{
					Type:  TokenSpace,
					Value: " \n  ",
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 2, Column: 2, Offset: 5},
					},
				},
				{
					Type:  TokenNumber,
					Value: "2",
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 2, Offset: 5},
						EndPos:   ast.Position{Line: 2, Column: 3, Offset: 6},
					},
				},
				{
					Type:  TokenSpace,
					Value: "\n",
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 3, Offset: 6},
						EndPos:   ast.Position{Line: 3, Column: 0, Offset: 7},
					},
				},
				{
					Type: TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Line: 3, Column: 0, Offset: 7},
						EndPos:   ast.Position{Line: 3, Column: 0, Offset: 7},
					},
				},
			},
			Lex("1 \n  2\n"),
		)
	})
}
