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
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestParseBlockComment(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseExpression(`/**/ true`)
		require.Empty(t, errs)
	})

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

		// Extracting comments attached to the infix operator is more difficult and also an edge case, so ignore that for now.
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
}
