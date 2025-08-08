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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/goleak"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/parser/lexer"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func (p *parser) mustToken(tokenType lexer.TokenType, string string) (lexer.Token, error) {
	t := p.current
	if !p.isToken(t, tokenType, string) {
		return lexer.Token{}, p.newSyntaxError("expected token %s with string value %s", tokenType, string)
	}
	p.next()
	return t, nil
}

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

func checkErrorsPrintable(errs []error, code string) {
	if len(errs) == 0 {
		return
	}
	err := Error{
		Code:   []byte(code),
		Errors: errs,
	}
	_ = err.Error()
}

func testParseStatements(s string) ([]ast.Statement, []error) {
	return testParseStatementsWithConfig(s, Config{})
}

func testParseStatementsWithConfig(s string, config Config) ([]ast.Statement, []error) {
	statements, errs := ParseStatements(nil, []byte(s), config)
	checkErrorsPrintable(errs, s)
	return statements, errs
}

func testParseDeclarations(s string) ([]ast.Declaration, []error) {
	return testParseDeclarationsWithConfig(s, Config{})
}

func testParseDeclarationsWithConfig(s string, config Config) ([]ast.Declaration, []error) {
	declarations, errs := ParseDeclarations(nil, []byte(s), config)
	checkErrorsPrintable(errs, s)
	return declarations, errs
}

func testParseProgram(s string) (*ast.Program, error) {
	return testParseProgramWithConfig(s, Config{})
}

func testParseProgramWithConfig(s string, config Config) (*ast.Program, error) {
	program, err := ParseProgram(nil, []byte(s), config)
	if err != nil {
		_ = err.Error()
	}
	return program, err
}

func testParseExpression(s string) (ast.Expression, []error) {
	return testParseExpressionWithConfig(s, Config{})
}

func testParseExpressionWithConfig(s string, config Config) (ast.Expression, []error) {
	expression, errs := ParseExpression(nil, []byte(s), config)
	checkErrorsPrintable(errs, s)
	return expression, errs
}

func testParseArgumentList(s string) (ast.Arguments, []error) {
	return testParseArgumentListWithConfig(s, Config{})
}

func testParseArgumentListWithConfig(s string, config Config) (ast.Arguments, []error) {
	arguments, errs := ParseArgumentList(nil, []byte(s), config)
	checkErrorsPrintable(errs, s)
	return arguments, errs
}

func testParseType(s string) (ast.Type, []error) {
	return testParseTypeWithConfig(s, Config{})
}

func testParseTypeWithConfig(s string, config Config) (ast.Type, []error) {
	ty, errs := ParseType(nil, []byte(s), config)
	checkErrorsPrintable(errs, s)
	return ty, errs
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
			nil,
			[]byte("a b c d"),
			func(p *parser) (struct{}, error) {
				_, err := p.mustToken(lexer.TokenIdentifier, "a")
				if err != nil {
					return struct{}{}, err
				}

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}

				p.startBuffering()

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return struct{}{}, err
				}

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return struct{}{}, err
				}

				p.acceptBuffered()

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "d")
				if err != nil {
					return struct{}{}, err
				}

				return struct{}{}, nil
			},
			Config{},
		)

		assert.Empty(t, errs)
	})

	t.Run("buffer and accept, invalid", func(t *testing.T) {

		t.Parallel()

		_, errs := Parse(
			nil,
			[]byte("a b x d"),
			func(p *parser) (struct{}, error) {
				_, err := p.mustToken(lexer.TokenIdentifier, "a")
				if err != nil {
					return struct{}{}, err
				}

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}

				p.startBuffering()

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return struct{}{}, err
				}

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return struct{}{}, err
				}

				p.acceptBuffered()

				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "d")
				if err != nil {
					return struct{}{}, err
				}

				return struct{}{}, nil
			},
			Config{},
		)

		AssertEqualWithDiff(t,
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
			nil,
			[]byte("a b c d"),
			func(p *parser) (struct{}, error) {
				_, err := p.mustToken(lexer.TokenIdentifier, "a")
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}

				p.startBuffering()

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return struct{}{}, err
				}

				err = p.replayBuffered()
				if err != nil {
					return struct{}{}, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "d")
				if err != nil {
					return struct{}{}, err
				}

				return struct{}{}, nil
			},
			Config{},
		)

		assert.Empty(t, errs)
	})

	t.Run("buffer and replay, invalid first", func(t *testing.T) {

		t.Parallel()

		_, errs := Parse(
			nil,
			[]byte("a b c d"),
			func(p *parser) (struct{}, error) {
				_, err := p.mustToken(lexer.TokenIdentifier, "a")
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
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
					return struct{}{}, err
				}

				_, err = p.mustToken(lexer.TokenIdentifier, "b")
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "c")
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustOne(lexer.TokenSpace)
				if err != nil {
					return struct{}{}, err
				}
				_, err = p.mustToken(lexer.TokenIdentifier, "d")
				if err != nil {
					return struct{}{}, err
				}

				return struct{}{}, nil
			},
			Config{},
		)

		assert.Empty(t, errs)
	})

	t.Run("buffer and replay, invalid first and invalid second", func(t *testing.T) {

		t.Parallel()

		_, errs := Parse(
			nil,
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
			Config{},
		)

		AssertEqualWithDiff(t,
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
                     <                             /* ************/((TODO?/*))************ *//
                    -x                             /* maybe it says NaNs are not negative?  */
          }
        `
		_, err := testParseProgram(code)
		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message:       "expected token identifier",
					Pos:           ast.Position{Offset: 398, Line: 9, Column: 94},
					Secondary:     "check for missing punctuation, operators, or syntax elements",
					Documentation: "https://cadence-lang.org/docs/language/syntax",
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

		AssertEqualWithDiff(t,
			[]error{
				&RestrictedTypeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 138, Line: 4, Column: 55},
						EndPos:   ast.Position{Offset: 139, Line: 4, Column: 56},
					},
				},
			},
			err.(Error).Errors,
		)
	})

	t.Run("nested buffering, valid; accept,accept,replay", func(t *testing.T) {

		t.Parallel()

		src := `
            access(all) struct interface Y {}
            access(all) struct X : Y {}
            access(all) fun main():String {
                fun f(a:Bool, _:String):String { return _; }
                let S = 1
                if false {
                    let Type_X_Y__qp_identifier =
                                    Type<{Y}>().identifier; // parses fine
                    return f(a:S<S, Type_X_Y__qp_identifier)
                } else {
                    return f(a:S<S, Type<{Y}>().identifier) // should also parse fine
                }
            }`

		_, err := testParseProgram(src)
		assert.NoError(t, err)
	})

	t.Run("nested buffering, valid; overlapped", func(t *testing.T) {

		t.Parallel()

		src := `
            transaction { }
            access(all) fun main():String {
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
		nil,
		[]byte("a b"),
		func(p *parser) (struct{}, error) {
			_, err := p.mustToken(lexer.TokenIdentifier, "a")
			if err != nil {
				return struct{}{}, err
			}
			p.skipSpaceAndComments()
			_, err = p.mustToken(lexer.TokenIdentifier, "b")
			if err != nil {
				return struct{}{}, err
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

			return struct{}{}, nil
		},
		Config{},
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

		assert.NotNil(t, actual)

		if validExpected {
			assert.NoError(t, err)
		} else {
			assert.IsType(t, Error{}, err)
		}
	}
}

func TestParseArgumentList(t *testing.T) {

	t.Parallel()

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseArgumentList(`xyz`)
		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message:       "expected token '('",
					Pos:           ast.Position{Offset: 0, Line: 1, Column: 0},
					Secondary:     "check for missing punctuation, operators, or syntax elements",
					Documentation: "https://cadence-lang.org/docs/language/syntax",
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

		AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("fatal error from lack of memory", func(t *testing.T) {
		gauge := makeLimitingMemoryGauge()
		gauge.Limit(common.MemoryKindTypeToken, 0)

		_, errs := ParseArgumentList(gauge, []byte(`(1, b: true)`), Config{})
		require.Len(t, errs, 1)

		require.IsType(t, errors.MemoryMeteringError{}, errs[0])

		fatalError, _ := errs[0].(errors.MemoryMeteringError)
		var expectedError limitingMemoryGaugeError
		assert.ErrorAs(t, fatalError, &expectedError)
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseArgumentList(`(1, b: true)`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
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
						Offset: 5,
						Line:   1,
						Column: 5,
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
	AssertEqualWithDiff(t,
		[]error{
			&SyntaxError{
				Message:       "missing type annotation after comma",
				Pos:           ast.Position{Offset: 4, Line: 1, Column: 4},
				Secondary:     "after a comma, a type annotation is required to complete the list",
				Documentation: "https://cadence-lang.org/docs/language/types-and-type-system/type-annotations",
			},
			&MissingClosingParenInArgumentListError{
				Pos: ast.Position{Offset: 6, Line: 1, Column: 6},
			},
		},
		errs,
	)
}

func TestParseInvalidSingleQuoteImport(t *testing.T) {

	t.Parallel()

	_, err := testParseProgram(`import 'X'`)

	require.ErrorContains(t, err, "Parsing failed:\nerror: unrecognized character: U+0027 '''")
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

	AssertEqualWithDiff(t,
		[]error{
			ExpressionDepthLimitReachedError{
				Pos: ast.Position{
					Offset: 87,
					Line:   1,
					Column: 87,
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

	AssertEqualWithDiff(t,
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

	code := builder.String()
	_, err := testParseProgram(code)
	AssertEqualWithDiff(t,
		Error{
			Code: []byte(code),
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

	code := builder.String()
	_, err := testParseProgram(code)
	AssertEqualWithDiff(t,
		Error{
			Code: []byte(code),
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

func TestParseWhitespaceAtEnd(t *testing.T) {

	t.Parallel()

	_, errs := Parse(
		nil,
		[]byte("a  "),
		func(p *parser) (lexer.Token, error) {
			return p.mustToken(lexer.TokenIdentifier, "a")
		},
		Config{},
	)

	assert.Empty(t, errs)
}
