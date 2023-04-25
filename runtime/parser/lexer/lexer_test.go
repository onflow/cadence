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

package lexer

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func withTokens(tokenStream TokenStream, fn func([]Token)) {
	tokens := make([]Token, 0)
	for {
		token := tokenStream.Next()
		tokens = append(tokens, token)
		if token.Is(TokenEOF) {
			fn(tokens)
			return
		}
	}
}

type token struct {
	Source string
	Token
}

func testLex(t *testing.T, input string, expected []token) {

	t.Parallel()

	expectedTokens := make([]Token, len(expected))
	for i, e := range expected {
		expectedTokens[i] = e.Token
	}

	bytes := []byte(input)

	withTokens(Lex(bytes, nil), func(actualTokens []Token) {
		utils.AssertEqualWithDiff(t, expectedTokens, actualTokens)

		require.Len(t, actualTokens, len(expectedTokens))
		for i, expectedToken := range expected {
			actualToken := actualTokens[i]
			if actualToken.Type == TokenEOF {
				continue
			}
			assert.Equal(t,
				expectedToken.Source,
				string(actualToken.Source(bytes)),
			)
		}
	})
}

func TestLexBasic(t *testing.T) {

	t.Parallel()

	t.Run("two numbers separated by whitespace", func(t *testing.T) {
		testLex(t,
			" 01\t  10",
			[]token{
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: "01",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					Source: "\t  ",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: "10",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("assignment", func(t *testing.T) {
		testLex(t,
			"x=1",
			[]token{
				{
					Token: Token{
						Type: TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "x",
				},
				{
					Token: Token{
						Type: TokenEqual,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "=",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: "1",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
				},
			},
		)
	})

	t.Run("simple arithmetic: plus and times", func(t *testing.T) {
		testLex(t,
			"(2 + 3) * 4",
			[]token{
				{
					Token: Token{
						Type: TokenParenOpen,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "(",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "2",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenPlus,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "+",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					Source: "3",
				},
				{
					Token: Token{
						Type: TokenParenClose,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Source: ")",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenStar,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Source: "*",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
					Source: "4",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
				},
			},
		)
	})

	t.Run("simple arithmetic: minus and div", func(t *testing.T) {
		testLex(t,
			"(2 - 3) / 4",
			[]token{
				{
					Token: Token{
						Type: TokenParenOpen,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "(",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "2",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenMinus,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "-",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					Source: "3",
				},
				{
					Token: Token{
						Type: TokenParenClose,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Source: ")",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenSlash,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Source: "/",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
					Source: "4",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
				},
			},
		)
	})

	t.Run("multiple lines", func(t *testing.T) {
		testLex(t,
			"1 \n  2\n",
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "1",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: true,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 2, Column: 1, Offset: 4},
						},
					},
					Source: " \n  ",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 2, Offset: 5},
							EndPos:   ast.Position{Line: 2, Column: 2, Offset: 5},
						},
					},
					Source: "2",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: true,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 3, Offset: 6},
							EndPos:   ast.Position{Line: 2, Column: 3, Offset: 6},
						},
					},
					Source: "\n",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 3, Column: 0, Offset: 7},
							EndPos:   ast.Position{Line: 3, Column: 0, Offset: 7},
						},
					},
				},
			},
		)
	})

	t.Run("nil-coalesce", func(t *testing.T) {
		testLex(t,
			"1 ?? 2",
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "1",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenDoubleQuestionMark,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "??",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					Source: "2",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
				},
			},
		)
	})

	t.Run("identifier", func(t *testing.T) {
		testLex(t,
			"test",
			[]token{
				{
					Token: Token{
						Type: TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "test",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
			},
		)
	})

	t.Run("identifier with leading underscore and trailing numbers", func(t *testing.T) {
		testLex(t,
			"_test_123",
			[]token{
				{
					Token: Token{
						Type: TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Source: "_test_123",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
				},
			},
		)
	})

	t.Run("colon, comma, semicolon, question mark", func(t *testing.T) {
		testLex(t,
			":,;.?",

			[]token{
				{
					Token: Token{
						Type: TokenColon,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: ":",
				},
				{
					Token: Token{
						Type: TokenComma,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: ",",
				},
				{
					Token: Token{
						Type: TokenSemicolon,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: ";",
				},
				{
					Token: Token{
						Type: TokenDot,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: ".",
				},
				{
					Token: Token{
						Type: TokenQuestionMark,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: "?",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
				},
			},
		)
	})

	t.Run("brackets and braces", func(t *testing.T) {
		testLex(t,
			"[}]{",
			[]token{
				{
					Token: Token{
						Type: TokenBracketOpen,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "[",
				},
				{
					Token: Token{
						Type: TokenBraceClose,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "}",
				},
				{
					Token: Token{
						Type: TokenBracketClose,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: "]",
				},
				{
					Token: Token{
						Type: TokenBraceOpen,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "{",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
			},
		)
	})

	t.Run("comparisons", func(t *testing.T) {
		testLex(t,
			"=<><-<=>=",
			[]token{
				{
					Token: Token{
						Type: TokenEqual,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "=",
				},
				{
					Token: Token{
						Type: TokenLess,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "<",
				},
				{
					Token: Token{
						Type: TokenGreater,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: ">",
				},
				{
					Token: Token{
						Type: TokenLeftArrow,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: "<-",
				},
				{
					Token: Token{
						Type: TokenLessEqual,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Source: "<=",
				},
				{
					Token: Token{
						Type: TokenGreaterEqual,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Source: ">=",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
				},
			},
		)
	})
}

func TestLexString(t *testing.T) {

	t.Parallel()

	t.Run("valid, empty", func(t *testing.T) {
		testLex(t,
			`""`,
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: `""`,
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
			},
		)
	})

	t.Run("valid, non-empty", func(t *testing.T) {
		testLex(t,
			`"test"`,
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
					Source: `"test"`,
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
				},
			},
		)
	})

	t.Run("valid, with valid tab escape", func(t *testing.T) {
		testLex(t,
			`"te\tst"`,
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: `"te\tst"`,
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("valid, with invalid escape character", func(t *testing.T) {
		testLex(t,
			`"te\Xst"`,
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: `"te\Xst"`,
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("valid, with valid quote escape", func(t *testing.T) {
		testLex(t,
			`"te\"st"`,
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: `"te\"st"`,
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("invalid, empty, not terminated at line end", func(t *testing.T) {
		testLex(t,
			"\"\n",
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "\"",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: true,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "\n",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 2},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 2},
						},
					},
				},
			},
		)
	})

	t.Run("invalid, non-empty, not terminated at line end", func(t *testing.T) {
		testLex(t,
			"\"te\n",
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: "\"te",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: true,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "\n",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 4},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 4},
						},
					},
				},
			},
		)
	})

	t.Run("invalid, empty, not terminated at end of file", func(t *testing.T) {
		testLex(t,
			"\"",
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "\"",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
				},
			},
		)
	})

	t.Run("invalid, non-empty, not terminated at end of file", func(t *testing.T) {
		testLex(t,
			"\"te",
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: "\"te",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
				},
			},
		)
	})

	t.Run("invalid, missing escape character", func(t *testing.T) {
		testLex(t,
			"\"\\\n",
			[]token{
				{
					Token: Token{
						Type: TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "\"\\",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: true,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: "\n",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 3},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 3},
						},
					},
				},
			},
		)
	})
}

func TestLexBlockComment(t *testing.T) {

	t.Parallel()

	t.Run("nested 1", func(t *testing.T) {
		testLex(t,
			`/*  // *X /* \\*  */`,
			[]token{
				{
					Token: Token{
						Type: TokenBlockCommentStart,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "/*",
				},
				{
					Token: Token{
						Type: TokenBlockCommentContent,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Source: `  // *X `,
				},
				{
					Token: Token{
						Type: TokenBlockCommentStart,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					Source: "/*",
				},
				{
					Token: Token{
						Type: TokenBlockCommentContent,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
							EndPos:   ast.Position{Line: 1, Column: 17, Offset: 17},
						},
					},
					Source: ` \\*  `,
				},
				{
					Token: Token{
						Type: TokenBlockCommentEnd,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 18, Offset: 18},
							EndPos:   ast.Position{Line: 1, Column: 19, Offset: 19},
						},
					},
					Source: "*/",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 20, Offset: 20},
							EndPos:   ast.Position{Line: 1, Column: 20, Offset: 20},
						},
					},
				},
			},
		)
	})

	t.Run("nested 2", func(t *testing.T) {
		testLex(t,
			`/* test foo /* bar */ asd */  `,
			[]token{
				{
					Token: Token{
						Type: TokenBlockCommentStart,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "/*",
				},
				{
					Token: Token{
						Type: TokenBlockCommentContent,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					Source: ` test foo `,
				},
				{
					Token: Token{
						Type: TokenBlockCommentStart,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
							EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
						},
					},
					Source: "/*",
				},
				{
					Token: Token{
						Type: TokenBlockCommentContent,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
						},
					},
					Source: ` bar `,
				},
				{
					Token: Token{
						Type: TokenBlockCommentEnd,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 19, Offset: 19},
							EndPos:   ast.Position{Line: 1, Column: 20, Offset: 20},
						},
					},
					Source: "*/",
				},
				{
					Token: Token{
						Type: TokenBlockCommentContent,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 21, Offset: 21},
							EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
						},
					},
					Source: ` asd `,
				},
				{
					Token: Token{
						Type: TokenBlockCommentEnd,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 26, Offset: 26},
							EndPos:   ast.Position{Line: 1, Column: 27, Offset: 27},
						},
					},
					Source: "*/",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 28, Offset: 28},
							EndPos:   ast.Position{Line: 1, Column: 29, Offset: 29},
						},
					},
					Source: "  ",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 30, Offset: 30},
							EndPos:   ast.Position{Line: 1, Column: 30, Offset: 30},
						},
					},
				},
			},
		)
	})
}

func TestLexIntegerLiterals(t *testing.T) {

	t.Parallel()

	t.Run("binary prefix, missing trailing digits", func(t *testing.T) {
		testLex(t,
			`0b`,
			[]token{
				{
					Token: Token{
						Type:         TokenError,
						SpaceOrError: errors.New("missing digits"),
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "b",
				},
				{
					Token: Token{
						Type: TokenBinaryIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "0b",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
			},
		)
	})

	t.Run("binary", func(t *testing.T) {
		testLex(t,
			`0b101010`,
			[]token{
				{
					Token: Token{
						Type: TokenBinaryIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: "0b101010",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("binary with leading zeros", func(t *testing.T) {
		testLex(t,
			`0b001000`,
			[]token{
				{
					Token: Token{
						Type: TokenBinaryIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: "0b001000",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("binary with underscores", func(t *testing.T) {
		testLex(t,
			`0b101010_101010`,
			[]token{
				{
					Token: Token{
						Type: TokenBinaryIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					Source: "0b101010_101010",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 15, Offset: 15},
							EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
						},
					},
				},
			},
		)
	})

	t.Run("binary with leading underscore", func(t *testing.T) {
		testLex(t,
			`0b_101010_101010`,
			[]token{
				{
					Token: Token{
						Type: TokenBinaryIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
						},
					},
					Source: "0b_101010_101010",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
							EndPos:   ast.Position{Line: 1, Column: 16, Offset: 16},
						},
					},
				},
			},
		)
	})

	t.Run("binary with trailing underscore", func(t *testing.T) {
		testLex(t,
			`0b101010_101010_`,
			[]token{
				{
					Token: Token{
						Type: TokenBinaryIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
						},
					},
					Source: "0b101010_101010_",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
							EndPos:   ast.Position{Line: 1, Column: 16, Offset: 16},
						},
					},
				},
			},
		)
	})

	t.Run("octal prefix, missing trailing digits", func(t *testing.T) {
		testLex(t,
			`0o`,
			[]token{
				{
					Token: Token{
						Type:         TokenError,
						SpaceOrError: errors.New("missing digits"),
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "o",
				},
				{
					Token: Token{
						Type: TokenOctalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "0o",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
			},
		)
	})

	t.Run("octal", func(t *testing.T) {
		testLex(t,
			`0o32`,
			[]token{
				{
					Token: Token{
						Type: TokenOctalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "0o32",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
			},
		)
	})

	t.Run("octal with underscores", func(t *testing.T) {
		testLex(t,
			`0o32_45`,
			[]token{
				{
					Token: Token{
						Type: TokenOctalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Source: "0o32_45",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
				},
			},
		)
	})

	t.Run("octal with leading underscore", func(t *testing.T) {
		testLex(t,
			`0o_32_45`,
			[]token{
				{
					Token: Token{
						Type: TokenOctalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: "0o_32_45",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("octal with trailing underscore", func(t *testing.T) {
		testLex(t,
			`0o32_45_`,
			[]token{
				{
					Token: Token{
						Type: TokenOctalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: "0o32_45_",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("decimal", func(t *testing.T) {
		testLex(t,
			`1234567890`,
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Source: "1234567890",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 10, Offset: 10},
							EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
						},
					},
				},
			},
		)
	})

	t.Run("decimal with underscores", func(t *testing.T) {
		testLex(t,
			`1_234_567_890`,
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
						},
					},
					Source: "1_234_567_890",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 13, Offset: 13},
							EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
						},
					},
				},
			},
		)
	})

	t.Run("decimal with trailing underscore", func(t *testing.T) {
		testLex(t,
			`1_234_567_890_`,
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
						},
					},
					Source: "1_234_567_890_",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
				},
			},
		)
	})

	t.Run("hexadecimal prefix, missing trailing digits", func(t *testing.T) {
		testLex(t,
			`0x`,
			[]token{
				{
					Token: Token{
						Type:         TokenError,
						SpaceOrError: errors.New("missing digits"),
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "x",
				},
				{
					Token: Token{
						Type: TokenHexadecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "0x",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
			},
		)
	})

	t.Run("hexadecimal", func(t *testing.T) {
		testLex(t,
			`0xf2`,
			[]token{
				{
					Token: Token{
						Type: TokenHexadecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "0xf2",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
				},
			},
		)
	})

	t.Run("hexadecimal with underscores", func(t *testing.T) {
		testLex(t,
			`0xf2_09`,
			[]token{
				{
					Token: Token{
						Type: TokenHexadecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					Source: "0xf2_09",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
				},
			},
		)
	})

	t.Run("hexadecimal with leading underscore", func(t *testing.T) {
		testLex(t,
			`0x_f2_09`,
			[]token{
				{
					Token: Token{
						Type: TokenHexadecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: "0x_f2_09",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("hexadecimal with trailing underscore", func(t *testing.T) {
		testLex(t,
			`0xf2_09_`,
			[]token{
				{
					Token: Token{
						Type: TokenHexadecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Source: "0xf2_09_",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
				},
			},
		)
	})

	t.Run("0", func(t *testing.T) {
		testLex(t,
			"0",
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "0",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
				},
			},
		)
	})

	t.Run("01", func(t *testing.T) {
		testLex(t,
			"01",
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "01",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
			},
		)
	})

	t.Run("whitespace after 0", func(t *testing.T) {
		testLex(t,
			"0\n",
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: "0",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: true,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "\n",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 2},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 2},
						},
					},
				},
			},
		)
	})

	t.Run("leading zeros", func(t *testing.T) {
		testLex(t,
			"00123",
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: "00123",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
				},
			},
		)
	})

	t.Run("invalid prefix", func(t *testing.T) {
		testLex(t,
			"0z123",
			[]token{
				{
					Token: Token{
						Type:         TokenError,
						SpaceOrError: errors.New("invalid number literal prefix: 'z'"),
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "z",
				},
				{
					Token: Token{
						Type: TokenUnknownBaseIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: "0z123",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
				},
			},
		)
	})

	t.Run("leading zero and underscore", func(t *testing.T) {

		testLex(t,
			"0_100",
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: "0_100",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
				},
			},
		)
	})

	t.Run("leading one and underscore", func(t *testing.T) {

		testLex(t,
			"1_100",
			[]token{
				{
					Token: Token{
						Type: TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: "1_100",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
						},
					},
				},
			},
		)
	})
}

func TestLexFixedPoint(t *testing.T) {

	t.Parallel()

	t.Run("with underscores", func(t *testing.T) {
		testLex(t,
			"1234_5678_90.0009_8765_4321",
			[]token{
				{
					Token: Token{
						Type: TokenFixedPointNumberLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 26, Offset: 26},
						},
					},
					Source: "1234_5678_90.0009_8765_4321",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 27, Offset: 27},
							EndPos:   ast.Position{Line: 1, Column: 27, Offset: 27},
						},
					},
				},
			},
		)
	})

	t.Run("leading zero", func(t *testing.T) {
		testLex(t,
			"0.1",
			[]token{
				{
					Token: Token{
						Type: TokenFixedPointNumberLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
					Source: "0.1",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
				},
			},
		)
	})

	t.Run("missing fractional digits", func(t *testing.T) {
		testLex(t,
			"0.",
			[]token{
				{
					Token: Token{
						Type:         TokenError,
						SpaceOrError: errors.New("missing fractional digits"),
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: ".",
				},
				{
					Token: Token{
						Type: TokenFixedPointNumberLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					Source: "0.",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
							EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
			},
		)
	})
}

func TestLexLineComment(t *testing.T) {

	t.Parallel()

	t.Run("no newline", func(t *testing.T) {

		testLex(t,
			` foo // bar `,
			[]token{
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "foo",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenLineComment,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					Source: "// bar ",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
							EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
						},
					},
				},
			},
		)
	})

	t.Run("newline", func(t *testing.T) {

		testLex(
			t,
			" foo // bar \n baz",
			[]token{
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
							EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
							EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					Source: "foo",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: false,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					Source: " ",
				},
				{
					Token: Token{
						Type: TokenLineComment,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					Source: "// bar ",
				},
				{
					Token: Token{
						Type: TokenSpace,
						SpaceOrError: Space{
							ContainsNewline: true,
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 13},
						},
					},
					Source: "\n ",
				},
				{
					Token: Token{
						Type: TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 1, Offset: 14},
							EndPos:   ast.Position{Line: 2, Column: 3, Offset: 16},
						},
					},
					Source: "baz",
				},
				{
					Token: Token{
						Type: TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 4, Offset: 17},
							EndPos:   ast.Position{Line: 2, Column: 4, Offset: 17},
						},
					},
				},
			},
		)
	})
}

func TestRevert(t *testing.T) {

	t.Parallel()

	tokenStream := Lex([]byte("1 2 3"), nil)

	// Assert all tokens

	assert.Equal(t,
		Token{
			Type: TokenDecimalIntegerLiteral,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
			},
		},
		tokenStream.Next(),
	)

	assert.Equal(t,
		Token{
			Type: TokenSpace,
			SpaceOrError: Space{
				ContainsNewline: false,
			},
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
				EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
			},
		},
		tokenStream.Next(),
	)

	twoCursor := tokenStream.Cursor()

	assert.Equal(t,
		Token{
			Type: TokenDecimalIntegerLiteral,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
				EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
			},
		},
		tokenStream.Next(),
	)

	assert.Equal(t,
		Token{
			Type: TokenSpace,
			SpaceOrError: Space{
				ContainsNewline: false,
			},
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
				EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
			},
		},
		tokenStream.Next(),
	)

	assert.Equal(t,
		Token{
			Type: TokenDecimalIntegerLiteral,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
				EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
			},
		},
		tokenStream.Next(),
	)

	// Assert EOF keeps on being returned for Next()
	// at the end of the stream

	assert.Equal(t,
		Token{
			Type: TokenEOF,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
				EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
			},
		},
		tokenStream.Next(),
	)

	assert.Equal(t,
		Token{
			Type: TokenEOF,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
				EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
			},
		},
		tokenStream.Next(),
	)

	// Revert back to token '2'

	tokenStream.Revert(twoCursor)

	// Re-assert tokens

	assert.Equal(t,
		Token{
			Type: TokenDecimalIntegerLiteral,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
				EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
			},
		},
		tokenStream.Next(),
	)

	assert.Equal(t,
		Token{
			Type: TokenSpace,
			SpaceOrError: Space{
				ContainsNewline: false,
			},
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
				EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
			},
		},
		tokenStream.Next(),
	)

	assert.Equal(t,
		Token{
			Type: TokenDecimalIntegerLiteral,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
				EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
			},
		},
		tokenStream.Next(),
	)

	// Re-assert EOF keeps on being returned for Next()
	// at the end of the stream

	assert.Equal(t,
		Token{
			Type: TokenEOF,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
				EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
			},
		},
		tokenStream.Next(),
	)

	assert.Equal(t,
		Token{
			Type: TokenEOF,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
				EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
			},
		},
		tokenStream.Next(),
	)

}

func TestEOFsAfterError(t *testing.T) {

	t.Parallel()

	tokenStream := Lex([]byte(`1 ''`), nil)

	// Assert all tokens

	assert.Equal(t,
		Token{
			Type: TokenDecimalIntegerLiteral,
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
			},
		},
		tokenStream.Next(),
	)

	assert.Equal(t,
		Token{
			Type: TokenSpace,
			SpaceOrError: Space{
				ContainsNewline: false,
			},
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
				EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
			},
		},
		tokenStream.Next(),
	)

	assert.Equal(t,
		Token{
			Type:         TokenError,
			SpaceOrError: errors.New(`unrecognized character: U+0027 '''`),
			Range: ast.Range{
				StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
				EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
			},
		},
		tokenStream.Next(),
	)

	// Assert EOFs keep on being returned for Next()
	// at the end of the stream

	for i := 0; i < 10; i++ {

		require.Equal(t,
			Token{
				Type: TokenEOF,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
				},
			},
			tokenStream.Next(),
		)
	}
}

func TestEOFsAfterEmptyInput(t *testing.T) {

	t.Parallel()

	tokenStream := Lex(nil, nil)

	// Assert EOFs keep on being returned for Next()
	// at the end of the stream

	for i := 0; i < 10; i++ {

		require.Equal(t,
			Token{
				Type: TokenEOF,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			tokenStream.Next(),
		)
	}
}

func TestLimit(t *testing.T) {

	t.Parallel()

	var b strings.Builder
	for i := 0; i < 300000; i++ {
		b.WriteString("x ")
	}

	code := b.String()

	assert.PanicsWithValue(t,
		TokenLimitReachedError{},
		func() {
			_ = Lex([]byte(code), nil)
		},
	)
}
