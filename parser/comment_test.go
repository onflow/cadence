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
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/errors"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/parser/lexer"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestParseBlockComment(t *testing.T) {

	t.Parallel()

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseExpression(" /* test  foo/* bar  */ asd*/ true")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
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

		AssertEqualWithDiff(t,
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

		// TODO(preserve-comments): Extracting comments from operator tokens is a bit difficult,
		// 	so let's handle this later as it seems a pretty edge case location to add comments.
		result, errs := testParseExpression(" 1/*test  foo*/+/* bar  */ 2  ")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
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
					Comments: ast.Comments{
						Trailing: []*ast.Comment{
							ast.NewComment(nil, []byte("/*test  foo*/")),
						},
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
		AssertEqualWithDiff(t,
			[]error{
				// `true */ bar` is parsed as infix operation of path
				&SyntaxError{
					Message: "expected token identifier",
					Pos: ast.Position{
						Offset: 37,
						Line:   1,
						Column: 37,
					},
					Secondary:     "check for missing punctuation, operators, or syntax elements",
					Documentation: "https://cadence-lang.org/docs/language/syntax",
				},
			},
			errs,
		)
	})

	t.Run("missing closing", func(t *testing.T) {
		t.Parallel()

		const code = `/*`
		_, errs := testParseExpression(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingCommentEndError{
					Pos: ast.Position{Offset: 2, Line: 1, Column: 2},
				},
				UnexpectedEOFError{
					Pos: ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			errs,
		)

		var missingCommentEndErr *MissingCommentEndError
		require.ErrorAs(t, errs[0], &missingCommentEndErr)

		fixes := missingCommentEndErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert `*/`",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "*/",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
								EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
					},
				},
			},
			fixes,
		)
	})

	t.Run("nested, missing closing", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseExpression(" /* test  foo/* bar  */ asd true ")
		AssertEqualWithDiff(t,
			[]error{
				&MissingCommentEndError{
					Pos: ast.Position{Offset: 33, Line: 1, Column: 33},
				},
				UnexpectedEOFError{
					Pos: ast.Position{Offset: 33, Line: 1, Column: 33},
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
				// TODO(merge): is this correct?
				{
					Type: lexer.TokenIdentifier,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Offset: 2, Column: 2},
						EndPos:   ast.Position{Line: 1, Offset: 4, Column: 4},
					},
				},
				{Type: lexer.TokenEOF},
			},
			input: []byte(`/*foo`),
		}

		// TODO(merge): move and emit UnexpectedTokenInBlockCommentError from lexer
		_, errs := ParseTokenStream(
			nil,
			tokens,
			func(p *parser) (ast.Expression, error) {
				return parseExpression(p, lowestBindingPower)
			},
			Config{},
		)
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenInBlockCommentError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})
}

func TestParseWhileStatementComment(t *testing.T) {

	t.Parallel()

	result, errs := testParseStatements(`
// before if
if true {
	// noop
} // after if
// before else-if
else if true {
	// noop
} 
// before second else-if
else if true {
	// noop
} /* after else-if */ else {
	// noop
} // after else
`)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Statement{
			&ast.IfStatement{
				Test: &ast.BoolExpression{
					Value: true,
					Range: ast.Range{
						StartPos: ast.Position{Line: 3, Column: 3, Offset: 17},
						EndPos:   ast.Position{Line: 3, Column: 6, Offset: 20},
					},
				},
				Then: &ast.Block{
					Statements: nil,
					Range: ast.Range{
						StartPos: ast.Position{Line: 3, Column: 8, Offset: 22},
						EndPos:   ast.Position{Line: 5, Column: 0, Offset: 33},
					},
					Comments: ast.Comments{
						Trailing: []*ast.Comment{
							ast.NewComment(nil, []byte("// noop")),
							ast.NewComment(nil, []byte("// after if")),
						},
					},
				},
				Else: &ast.Block{
					Statements: []ast.Statement{
						&ast.IfStatement{
							Test: &ast.BoolExpression{
								Value: true,
								Range: ast.Range{
									StartPos: ast.Position{Line: 7, Column: 8, Offset: 73},
									EndPos:   ast.Position{Line: 7, Column: 11, Offset: 76},
								},
							},
							Then: &ast.Block{
								Statements: nil,
								Range: ast.Range{
									StartPos: ast.Position{Line: 7, Column: 13, Offset: 78},
									EndPos:   ast.Position{Line: 9, Column: 0, Offset: 89},
								},
								Comments: ast.Comments{
									Trailing: []*ast.Comment{
										ast.NewComment(nil, []byte("// noop")),
									},
								},
							},
							Else: &ast.Block{
								Statements: []ast.Statement{
									&ast.IfStatement{
										Test: &ast.BoolExpression{
											Value: true,
											Range: ast.Range{
												StartPos: ast.Position{Line: 11, Column: 8, Offset: 125},
												EndPos:   ast.Position{Line: 11, Column: 11, Offset: 128},
											},
										},
										Then: &ast.Block{
											Statements: nil,
											Range: ast.Range{
												StartPos: ast.Position{Line: 11, Column: 13, Offset: 130},
												EndPos:   ast.Position{Line: 13, Column: 0, Offset: 141},
											},
											Comments: ast.Comments{
												Trailing: []*ast.Comment{
													ast.NewComment(nil, []byte("// noop")),
													ast.NewComment(nil, []byte("/* after else-if */")),
												},
											},
										},
										Else: &ast.Block{
											Statements: nil,
											Range: ast.Range{
												StartPos: ast.Position{Line: 13, Column: 27, Offset: 168},
												EndPos:   ast.Position{Line: 15, Column: 0, Offset: 179},
											},
											Comments: ast.Comments{
												Trailing: []*ast.Comment{
													ast.NewComment(nil, []byte("// noop")),
													ast.NewComment(nil, []byte("// after else")),
												},
											},
										},
										StartPos: ast.Position{Line: 11, Column: 5, Offset: 122},
										Comments: ast.Comments{
											Leading: []*ast.Comment{
												ast.NewComment(nil, []byte("// before second else-if")),
											},
										},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Line: 11, Column: 5, Offset: 122},
									EndPos:   ast.Position{Line: 15, Column: 0, Offset: 179},
								},
							},
							StartPos: ast.Position{Line: 7, Column: 5, Offset: 70},
							Comments: ast.Comments{
								Leading: []*ast.Comment{
									ast.NewComment(nil, []byte("// before else-if")),
								},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 7, Column: 5, Offset: 70},
						EndPos:   ast.Position{Line: 15, Column: 0, Offset: 179},
					},
				},
				StartPos: ast.Position{Line: 3, Column: 0, Offset: 14},
				Comments: ast.Comments{
					Leading: []*ast.Comment{
						ast.NewComment(nil, []byte("// before if")),
					},
				},
			},
		},
		result,
	)
}
