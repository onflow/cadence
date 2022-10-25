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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/goleak"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser/lexer"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type testTokenStream struct {
	tokens []lexer.Token
	input  []byte
	cursor int
}

var _ lexer.TokenStream = &testTokenStream{}

func (t *testTokenStream) Next() lexer.Token {
	if t.cursor >= len(t.tokens) {

		// At the end of the token stream,
		// emit a synthetic EOF token

		return lexer.Token{
			Type: lexer.TokenEOF,
		}

	}
	token := t.tokens[t.cursor]
	t.cursor++
	return token
}

func (t *testTokenStream) Cursor() int {
	return t.cursor
}

func (t *testTokenStream) Revert(cursor int) {
	t.cursor = cursor
}

func (t *testTokenStream) Input() []byte {
	return t.input
}

func (*testTokenStream) Reclaim() {
	// NO-OP
}

func testParseStatements(s string) ([]ast.Statement, []error) {
	return ParseStatements([]byte(s), nil)
}

func testParseDeclarations(s string) ([]ast.Declaration, []error) {
	return ParseDeclarations([]byte(s), nil)
}

func testParseProgram(s string) (*ast.Program, error) {
	return ParseProgram([]byte(s), nil)
}

func testParseExpression(s string) (ast.Expression, []error) {
	return ParseExpression([]byte(s), nil)
}

func testParseArgumentList(s string) (ast.Arguments, []error) {
	return ParseArgumentList([]byte(s), nil)
}

func testParseType(s string) (ast.Type, []error) {
	return ParseType([]byte(s), nil)
}

func TestParseInvalid(t *testing.T) {
	t.Parallel()

	type test struct {
		msg  string
		code string
	}

	unexpectedToken := "Parsing failed:\nerror: unexpected token: identifier"
	unexpectedEndOfProgram := "Parsing failed:\nerror: unexpected end of program"
	missingTypeAnnotation := "Parsing failed:\nerror: missing type annotation after comma"

	for _, test := range []test{
		{unexpectedToken, "X"},
		{unexpectedToken, "paste your code in here"},
		{unexpectedEndOfProgram, "# a ( b > c > d > e > f > g > h > i > j > k > l > m > n > o > p > q > r >"},
		{missingTypeAnnotation, "#0x0<{},>()"},
	} {
		t.Run(test.code, func(t *testing.T) {
			_, err := testParseProgram(test.code)
			require.ErrorContains(t, err, test.msg)
		})
	}
}

func TestParseBuffering(t *testing.T) {

	t.Parallel()

	t.Run("buffer and accept, valid", func(t *testing.T) {

		t.Parallel()

		_, errs := Parse(
			[]byte("a b c d"),
			func(p *parser) (any, error) {
				_, err := p.mustToken(lexer.TokenIdentifier, "a")
				if err != nil {
					return nil, err
				}

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}

				p.startBuffering()

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return nil, err
				}

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return nil, err
				}

				p.acceptBuffered()

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "d")
				if err != nil {
					return nil, err
				}

				return nil, nil
			},
			nil,
		)

		assert.Empty(t, errs)
	})

	t.Run("buffer and accept, invalid", func(t *testing.T) {

		t.Parallel()

		_, errs := Parse(
			[]byte("a b x d"),
			func(p *parser) (any, error) {
				_, err := p.mustToken(lexer.TokenIdentifier, "a")
				if err != nil {
					return nil, err
				}

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}

				p.startBuffering()

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return nil, err
				}

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return nil, err
				}

				p.acceptBuffered()

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "d")
				if err != nil {
					return nil, err
				}

				return nil, nil
			},
			nil,
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token identifier with string value c",
					Pos:     ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)
	})

	t.Run("buffer and replay, valid", func(t *testing.T) {

		t.Parallel()

		_, errs := Parse(
			[]byte("a b c d"),
			func(p *parser) (any, error) {
				_, err := p.mustToken(lexer.TokenIdentifier, "a")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}

				p.startBuffering()

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return nil, err
				}

				err = p.replayBuffered()
				if err != nil {
					return nil, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "d")
				if err != nil {
					return nil, err
				}

				return nil, nil
			},
			nil,
		)

		assert.Empty(t, errs)
	})

	t.Run("buffer and replay, invalid first", func(t *testing.T) {

		t.Parallel()

		_, errs := Parse(
			[]byte("a b c d"),
			func(p *parser) (any, error) {
				_, err := p.mustToken(lexer.TokenIdentifier, "a")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}

				p.startBuffering()

				firstSucceeded := false
				firstFailed := false

				// Ignore error
				func() {
					var bufferingError error

					defer func() {
						if r := recover(); r != nil || bufferingError != nil {
							firstFailed = true
						}
					}()

					_, bufferingError = p.mustToken(lexer.TokenIdentifier, "x")
					if bufferingError != nil {
						return
					}
					_, bufferingError = p.mustOne(lexer.TokenSpace)
					if bufferingError != nil {
						return
					}
					_, bufferingError = p.mustToken(lexer.TokenIdentifier, "c")
					if bufferingError != nil {
						return
					}

					firstSucceeded = true
				}()

				assert.True(t, firstFailed)
				assert.False(t, firstSucceeded)

				err = p.replayBuffered()
				if err != nil {
					return nil, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "d")
				if err != nil {
					return nil, err
				}

				return nil, nil
			},
			nil,
		)

		assert.Empty(t, errs)
	})

	t.Run("buffer and replay, invalid first and invalid second", func(t *testing.T) {

		t.Parallel()

		_, errs := Parse(
			[]byte("a b c x"),
			func(p *parser) (any, error) {
				_, err := p.mustToken(lexer.TokenIdentifier, "a")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}

				p.startBuffering()

				firstSucceeded := false
				firstFailed := false

				func() {
					var bufferingError error

					defer func() {
						if r := recover(); r != nil || bufferingError != nil {
							firstFailed = true
						}
					}()

					_, bufferingError = p.mustToken(lexer.TokenIdentifier, "x")
					if bufferingError != nil {
						return
					}
					_, bufferingError = p.mustOne(lexer.TokenSpace)
					if bufferingError != nil {
						return
					}
					_, bufferingError = p.mustToken(lexer.TokenIdentifier, "c")
					if bufferingError != nil {
						return
					}

					firstSucceeded = true
				}()

				assert.True(t, firstFailed)
				assert.False(t, firstSucceeded)

				err = p.replayBuffered()
				if err != nil {
					return nil, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return nil, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return nil, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "d")
				if err != nil {
					return nil, err
				}

				return nil, nil
			},
			nil,
		)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token identifier with string value d",
					Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
				},
			},
			errs,
		)
	})

	t.Run("nested buffering, invalid", func(t *testing.T) {

		t.Parallel()

		const code = `
          fun main() {
              assert(isneg(x:-1.0))
              assert(!isneg(x:-0.0/0.0))
          }

          fun isneg(x: SignedFixedPoint): Bool {   /* I kinda forget what this is all about */
              return x                             /* but we probably need to figure it out */
                     <                             /* ************/((TODO?{/*))************ *//
                    -x                             /* maybe it says NaNs are not negative?  */
          }
        `
		_, err := testParseProgram(code)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token identifier",
					Pos:     ast.Position{Offset: 420, Line: 10, Column: 20},
				},
			},
			err.(Error).Errors,
		)
	})

	t.Run("nested buffering, invalid; apparent invocation elision", func(t *testing.T) {

		t.Parallel()

		const code = `
          fun main() {
              fun abs(_:Int):Int { return _ > 0 ? _ : -_ }
              let sanity = 0 <          /*****/((TODO?{/*****//
                               abs(-1)
              assert(sanity)
          }
        `
		_, err := testParseProgram(code)

		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token '/'",
					Pos:     ast.Position{Offset: 181, Line: 5, Column: 34},
				},
			},
			err.(Error).Errors,
		)
	})

	t.Run("nested buffering, valid; accept,accept,replay", func(t *testing.T) {

		t.Parallel()

		src := `
            pub struct interface Y {}
            pub struct X : Y {}
            pub fun main():String {
                fun f(a:Bool, _:String):String { return _; }
                let S = 1
                if false {
                    let Type_X_Y__qp_identifier =
                                    Type<X{Y}>().identifier; // parses fine
                    return f(a:S<S, Type_X_Y__qp_identifier)
                } else {
                    return f(a:S<S, Type<X{Y}>().identifier) // should also parse fine
                }
            }`

		_, err := testParseProgram(src)
		assert.NoError(t, err)
	})

	t.Run("nested buffering, valid; overlapped", func(t *testing.T) {

		t.Parallel()

		src := `
            transaction { }
            pub fun main():String {
                let A = 1
                let B = 2
                let C = 3
                let D = 4
                fun g(a:Bool, _:Bool):String { return _ ? "y" : "n" }
                return g(a:A<B, C<(D>>(5)))
            }`

		_, err := testParseProgram(src)
		assert.NoError(t, err)
	})

}

func TestParseEOF(t *testing.T) {

	t.Parallel()

	_, errs := Parse(
		[]byte("a b"),
		func(p *parser) (any, error) {
			_, err := p.mustToken(lexer.TokenIdentifier, "a")
			if err != nil {
				return nil, err
			}
			p.skipSpaceAndComments()
			_, err = p.mustToken(lexer.TokenIdentifier, "b")
			if err != nil {
				return nil, err
			}

			p.next()

			assert.Equal(t,
				lexer.Token{
					Type: lexer.TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 3, Line: 1, Column: 3},
						EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
					},
				},
				p.current,
			)

			p.next()

			assert.Equal(t,
				lexer.Token{
					Type: lexer.TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 3, Line: 1, Column: 3},
						EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
					},
				},
				p.current,
			)

			return nil, nil
		},
		nil,
	)

	assert.Empty(t, errs)
}

func TestParseNames(t *testing.T) {

	t.Parallel()

	names := map[string]bool{
		// Valid: title-case
		//
		"PersonID": true,

		// Valid: with underscore
		//
		"token_name": true,

		// Valid: leading underscore and characters
		//
		"_balance": true,

		// Valid: leading underscore and numbers
		"_8264": true,

		// Valid: characters and number
		//
		"account2": true,

		// Invalid: leading number
		//
		"1something": false,

		// Invalid: invalid character #
		"_#1": false,

		// Invalid: various invalid characters
		//
		"!@#$%^&*": false,
	}

	for name, validExpected := range names {

		code := fmt.Sprintf(`let %s = 1`, name)

		actual, err := testParseProgram(code)

		if validExpected {
			assert.NotNil(t, actual)
			assert.NoError(t, err)

		} else {
			assert.Nil(t, actual)
			assert.IsType(t, Error{}, err)
		}
	}
}

func TestParseArgumentList(t *testing.T) {

	t.Parallel()

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseArgumentList(`xyz`)
		utils.AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected token '('",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseArgumentList(`()`)
		require.Empty(t, errs)

		var expected ast.Arguments

		utils.AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("fatal error from lack of memory", func(t *testing.T) {
		gauge := makeLimitingMemoryGauge()
		gauge.Limit(common.MemoryKindTypeToken, 0)

		var panicMsg any
		(func() {
			defer func() {
				panicMsg = recover()
			}()

			ParseArgumentList([]byte(`(1, b: true)`), gauge)
		})()

		require.IsType(t, errors.MemoryError{}, panicMsg)

		fatalError, _ := panicMsg.(errors.MemoryError)
		var expectedError limitingMemoryGaugeError
		assert.ErrorAs(t, fatalError, &expectedError)
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseArgumentList(`(1, b: true)`)
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			ast.Arguments{
				{
					Label:         "",
					LabelStartPos: nil,
					LabelEndPos:   nil,
					Expression: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{
								Offset: 1,
								Line:   1,
								Column: 1,
							},
							EndPos: ast.Position{
								Offset: 1,
								Line:   1,
								Column: 1,
							},
						},
					},
					TrailingSeparatorPos: ast.Position{
						Offset: 2,
						Line:   1,
						Column: 2,
					},
				},
				{
					Label: "b",
					LabelStartPos: &ast.Position{
						Offset: 4,
						Line:   1,
						Column: 4,
					},
					LabelEndPos: &ast.Position{
						Offset: 4,
						Line:   1,
						Column: 4,
					},
					Expression: &ast.BoolExpression{
						Value: true,
						Range: ast.Range{
							StartPos: ast.Position{
								Offset: 7,
								Line:   1,
								Column: 7,
							},
							EndPos: ast.Position{
								Offset: 10,
								Line:   1,
								Column: 10,
							},
						},
					},
					TrailingSeparatorPos: ast.Position{
						Offset: 11,
						Line:   1,
						Column: 11,
					},
				},
			},
			result,
		)
	})

}

func TestParseBufferedErrors(t *testing.T) {

	t.Parallel()

	// Test that both top-level and buffered errors are reported.
	//
	// Test this using type argument lists, which are parsed through buffering:
	// Only a subsequent open parenthesis will determine if a less-than sign
	// introduced a type argument list of a function call,
	// or if the expression is a less-than comparison.
	//
	// Inside the potential type argument list there is an error (missing type after comma),
	// and outside (at the top-level, after buffering of the type argument list),
	// there is another error (missing closing parenthesis after).

	_, errs := testParseExpression("a<b,>(")
	utils.AssertEqualWithDiff(t,
		[]error{
			&SyntaxError{
				Message: "missing type annotation after comma",
				Pos:     ast.Position{Offset: 4, Line: 1, Column: 4},
			},
			&SyntaxError{
				Message: "missing ')' at end of invocation argument list",
				Pos:     ast.Position{Offset: 6, Line: 1, Column: 6},
			},
		},
		errs,
	)
}

func TestParseInvalidSingleQuoteImport(t *testing.T) {

	t.Parallel()

	_, err := testParseProgram(`import 'X'`)

	require.EqualError(t, err, "Parsing failed:\nerror: unrecognized character: U+0027 '''\n --> :1:7\n  |\n1 | import 'X'\n  |        ^\n\nerror: unexpected end in import declaration: expected string, address, or identifier\n --> :1:7\n  |\n1 | import 'X'\n  |        ^\n")
}

func TestParseExpressionDepthLimit(t *testing.T) {

	t.Parallel()

	var builder strings.Builder
	builder.WriteString("let x = y")
	for i := 0; i < 20; i++ {
		builder.WriteString(" ?? z")
	}

	code := builder.String()

	_, err := testParseProgram(code)
	require.Error(t, err)

	utils.AssertEqualWithDiff(t,
		[]error{
			ExpressionDepthLimitReachedError{
				Pos: ast.Position{
					Offset: 88,
					Line:   1,
					Column: 88,
				},
			},
		},
		err.(Error).Errors,
	)
}

func TestParseTypeDepthLimit(t *testing.T) {

	t.Parallel()

	const nesting = 20

	var builder strings.Builder
	builder.WriteString("let x: T<")
	for i := 0; i < nesting; i++ {
		builder.WriteString("T<")
	}
	builder.WriteString("U")
	for i := 0; i < nesting; i++ {
		builder.WriteString(">")
	}
	builder.WriteString(">? = nil")

	code := builder.String()

	_, err := testParseProgram(code)
	require.Error(t, err)

	utils.AssertEqualWithDiff(t,
		[]error{
			TypeDepthLimitReachedError{
				Pos: ast.Position{
					Offset: 39,
					Line:   1,
					Column: 39,
				},
			},
		},
		err.(Error).Errors,
	)
}

func TestParseLocalReplayLimit(t *testing.T) {
	t.Parallel()

	var builder strings.Builder
	builder.WriteString("let t = T")
	for i := 0; i < 30; i++ {
		builder.WriteString("<T")
	}
	builder.WriteString(">()")

	code := []byte(builder.String())
	_, err := ParseProgram(code, nil)
	utils.AssertEqualWithDiff(t,
		Error{
			Code: code,
			Errors: []error{
				&SyntaxError{
					Message: fmt.Sprintf(
						"program too ambiguous, local replay limit of %d tokens exceeded",
						localTokenReplayCountLimit,
					),
					Pos: ast.Position{
						Offset: 44,
						Line:   1,
						Column: 44,
					},
				},
			},
		},
		err,
	)
}

func TestParseGlobalReplayLimit(t *testing.T) {

	t.Parallel()

	var builder strings.Builder
	for j := 0; j < 2; j++ {
		builder.WriteString(";let t = T")
		for i := 0; i < 16; i++ {
			builder.WriteString("<T")
		}
	}

	code := []byte(builder.String())
	_, err := ParseProgram(code, nil)
	utils.AssertEqualWithDiff(t,
		Error{
			Code: code,
			Errors: []error{
				&SyntaxError{
					Message: fmt.Sprintf(
						"program too ambiguous, global replay limit of %d tokens exceeded",
						globalTokenReplayCountLimit,
					),
					Pos: ast.Position{
						Offset: 84,
						Line:   1,
						Column: 84,
					},
				},
			},
		},
		err,
	)
}
