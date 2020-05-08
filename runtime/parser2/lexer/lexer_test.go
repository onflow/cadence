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

func withTokens(tokenChan chan Token, fn func([]Token)) {
	tokens := make([]Token, 0)
	for {
		token, ok := <-tokenChan
		if !ok {
			fn(tokens)
			return
		}
		tokens = append(tokens, token)
	}
}

func TestLex(t *testing.T) {

	t.Run("single char number", func(t *testing.T) {
		withTokens(Lex("0"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenNumber,
						Value: "0",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
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
				tokens,
			)
		})
	})

	t.Run("two char number", func(t *testing.T) {
		withTokens(Lex("01"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenNumber,
						Value: "01",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
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
				tokens,
			)
		})
	})

	t.Run("two numbers separated by whitespace", func(t *testing.T) {
		withTokens(Lex(" 01\t  10"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{

						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					{
						Type:  TokenNumber,
						Value: "01",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					{
						Type:  TokenSpace,
						Value: "\t  ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					{
						Type:  TokenNumber,
						Value: "10",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
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
				tokens,
			)
		})
	})

	t.Run("simple arithmetic: plus and times", func(t *testing.T) {
		withTokens(Lex("(2 + 3) * 4"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type: TokenParenOpen,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					{
						Type:  TokenNumber,
						Value: "2",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					{
						Type: TokenPlus,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					{
						Type:  TokenNumber,
						Value: "3",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					{
						Type: TokenParenClose,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					{
						Type: TokenStar,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					{
						Type:  TokenNumber,
						Value: "4",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
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
				tokens,
			)
		})
	})

	t.Run("simple arithmetic: minus and div", func(t *testing.T) {
		withTokens(Lex("(2 - 3) / 4"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type: TokenParenOpen,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					{
						Type:  TokenNumber,
						Value: "2",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					{
						Type: TokenMinus,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					{
						Type:  TokenNumber,
						Value: "3",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					{
						Type: TokenParenClose,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					{
						Type: TokenSlash,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					{
						Type:  TokenNumber,
						Value: "4",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
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
				tokens,
			)
		})
	})

	t.Run("multiple lines", func(t *testing.T) {
		withTokens(Lex("1 \n  2\n"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenNumber,
						Value: "1",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					{
						Type:  TokenSpace,
						Value: " \n  ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 2, Column: 1, Offset: 4},
						},
					},
					{
						Type:  TokenNumber,
						Value: "2",
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 2, Offset: 5},
							EndPos:   ast.Position{Line: 2, Column: 2, Offset: 5},
						},
					},
					{
						Type:  TokenSpace,
						Value: "\n",
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 3, Offset: 6},
							EndPos:   ast.Position{Line: 2, Column: 3, Offset: 6},
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
				tokens,
			)
		})
	})

	t.Run("nil-coalesce", func(t *testing.T) {
		withTokens(Lex("1 ?? 2"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenNumber,
						Value: "1",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type: TokenNilCoalesce,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					{
						Type:  TokenSpace,
						Value: " ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					{
						Type:  TokenNumber,
						Value: "2",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("identifier", func(t *testing.T) {
		withTokens(Lex("test"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenIdentifier,
						Value: "test",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("identifier with leading underscore and trailing numbers", func(t *testing.T) {
		withTokens(Lex("_test_123"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenIdentifier,
						Value: "_test_123",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("colon, comma, question mark", func(t *testing.T) {
		withTokens(Lex(":,?"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type: TokenColon,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					{
						Type: TokenComma,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type: TokenQuestionMark,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("brackets and braces", func(t *testing.T) {
		withTokens(Lex("[}]{"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type: TokenBracketOpen,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					{
						Type: TokenBraceClose,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type: TokenBracketClose,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},

					{
						Type: TokenBraceOpen,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("less than, greater than, and left arrow", func(t *testing.T) {
		withTokens(Lex("<><-"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type: TokenLess,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					{
						Type: TokenGreater,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type: TokenLeftArrow,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
				tokens,
			)
		})
	})
}

func TestLexString(t *testing.T) {

	t.Run("valid, empty", func(t *testing.T) {
		withTokens(Lex(`""`), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: `""`,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
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
				tokens,
			)
		})
	})

	t.Run("valid, non-empty", func(t *testing.T) {
		withTokens(Lex(`"test"`), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: `"test"`,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("valid, with valid tab escape", func(t *testing.T) {
		withTokens(Lex(`"te\tst"`), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: `"te\tst"`,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
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
				tokens,
			)
		})
	})

	t.Run("valid, with invalid escape character", func(t *testing.T) {
		withTokens(Lex(`"te\Xst"`), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: `"te\Xst"`,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
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
				tokens,
			)
		})
	})

	t.Run("valid, with valid quote escape", func(t *testing.T) {
		withTokens(Lex(`"te\"st"`), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: `"te\"st"`,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
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
				tokens,
			)
		})
	})

	t.Run("invalid, empty, not terminated at line end", func(t *testing.T) {
		withTokens(Lex("\"\n"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: "\"",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					{
						Type:  TokenSpace,
						Value: "\n",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 2},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 2},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("invalid, non-empty, not terminated at line end", func(t *testing.T) {
		withTokens(Lex("\"te\n"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: "\"te",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					{
						Type:  TokenSpace,
						Value: "\n",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 4},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 4},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("invalid, empty, not terminated at end of file", func(t *testing.T) {
		withTokens(Lex("\""), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: "\"",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
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
				tokens,
			)
		})
	})

	t.Run("invalid, non-empty, not terminated at end of file", func(t *testing.T) {
		withTokens(Lex("\"te"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: "\"te",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("invalid, missing escape character", func(t *testing.T) {
		withTokens(Lex("\"\\\n"), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type:  TokenString,
						Value: "\"\\",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type:  TokenSpace,
						Value: "\n",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 3},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 3},
						},
					},
				},
				tokens,
			)
		})
	})
}

func TestLexComment(t *testing.T) {

	t.Run("nested 1", func(t *testing.T) {
		withTokens(Lex(`/*  // *X /* \\*  */`), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type: TokenBlockCommentStart,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type:  TokenBlockCommentContent,
						Value: `  // *X `,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					{
						Type: TokenBlockCommentStart,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					{
						Type:  TokenBlockCommentContent,
						Value: ` \\*  `,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
							EndPos:   ast.Position{Line: 1, Column: 17, Offset: 17},
						},
					},
					{
						Type: TokenBlockCommentEnd,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 18, Offset: 18},
							EndPos:   ast.Position{Line: 1, Column: 19, Offset: 19},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 20, Offset: 20},
							EndPos:   ast.Position{Line: 1, Column: 20, Offset: 20},
						},
					},
				},
				tokens,
			)
		})
	})

	t.Run("nested 2", func(t *testing.T) {
		withTokens(Lex(`/* test foo /* bar */ asd */  `), func(tokens []Token) {
			assert.Equal(t,
				[]Token{
					{
						Type: TokenBlockCommentStart,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					{
						Type:  TokenBlockCommentContent,
						Value: ` test foo `,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					{
						Type: TokenBlockCommentStart,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
							EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
						},
					},
					{
						Type:  TokenBlockCommentContent,
						Value: ` bar `,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
						},
					},
					{
						Type: TokenBlockCommentEnd,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 19, Offset: 19},
							EndPos:   ast.Position{Line: 1, Column: 20, Offset: 20},
						},
					},
					{
						Type:  TokenBlockCommentContent,
						Value: ` asd `,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 21, Offset: 21},
							EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
						},
					},
					{
						Type: TokenBlockCommentEnd,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 26, Offset: 26},
							EndPos:   ast.Position{Line: 1, Column: 27, Offset: 27},
						},
					},
					{
						Type:  TokenSpace,
						Value: "  ",
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 28, Offset: 28},
							EndPos:   ast.Position{Line: 1, Column: 29, Offset: 29},
						},
					},
					{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 30, Offset: 30},
							EndPos:   ast.Position{Line: 1, Column: 30, Offset: 30},
						},
					},
				},
				tokens,
			)
		})
	})

}
