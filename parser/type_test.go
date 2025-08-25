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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/parser/lexer"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestParseNominalType(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("Int")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "Int",
					Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("Foo.Bar")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "Foo",
					Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
				},
				NestedIdentifiers: []ast.Identifier{
					{
						Identifier: "Bar",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
			},
			result,
		)
	})

	t.Run("invalid nested", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("Foo.1")
		AssertEqualWithDiff(t,
			[]error{
				&NestedTypeMissingNameError{
					GotToken: lexer.Token{
						Type: lexer.TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
					},
				},
			},
			errs,
		)
	})

	t.Run("incomplete parameter list", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("fun(T,")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedEOFExpectedTypeAnnotationError{
					Pos: ast.Position{Offset: 6, Line: 1, Column: 6},
				},
			},
			errs,
		)
	})
}

func TestParseInvalidType(t *testing.T) {

	t.Parallel()

	t.Run("number literal", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("123")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						Type: lexer.TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
						},
					},
				},
			},
			errs,
		)
	})
}

func TestParseArrayType(t *testing.T) {

	t.Parallel()

	t.Run("variable", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("[Int]")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.VariableSizedType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
				},
			},
			result,
		)
	})

	t.Run("variable, missing end", func(t *testing.T) {
		t.Parallel()

		const code = "[Int"
		_, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingBracketInArrayTypeError{
					GotToken: lexer.Token{
						Type: lexer.TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
					},
				},
			},
			errs,
		)

		var missingBracketErr *MissingClosingBracketInArrayTypeError
		require.ErrorAs(t, errs[0], &missingBracketErr)

		fixes := missingBracketErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing bracket",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "]",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t, "[Int]", fixes[0].TextEdits[0].ApplyTo(code))
	})

	t.Run("constant", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("[Int ; 2 ]")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ConstantSizedType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				Size: &ast.IntegerExpression{
					PositiveLiteral: []byte("2"),
					Value:           big.NewInt(2),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
						EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
				},
			},
			result,
		)
	})

	t.Run("constant, negative size", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("[Int ; -2 ]")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidConstantSizedTypeSizeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
						EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
			},
			errs,
		)

		// The parser recovers and returns a variable-sized array type
		AssertEqualWithDiff(t,
			&ast.VariableSizedType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{Identifier: "Int", Pos: ast.Position{Offset: 1, Line: 1, Column: 1}},
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			result,
		)
	})

	t.Run("constant, invalid size", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("[Int ; X ]")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidConstantSizedTypeSizeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
						EndPos:   ast.Position{Offset: 7, Line: 1, Column: 7},
					},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			&ast.VariableSizedType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Offset: 1, Line: 1, Column: 1},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					EndPos:   ast.Position{Offset: 9, Line: 1, Column: 9},
				},
			},
			result,
		)
	})

	t.Run("invalid, dot", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("[.")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						Type: lexer.TokenDot,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
							EndPos:   ast.Position{Offset: 1, Line: 1, Column: 1},
						},
					},
				},
			},
			errs,
		)
	})

	t.Run("constant, invalid size, dot", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("[T;.")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedExpressionStartError{
					GotToken: lexer.Token{
						Type: lexer.TokenDot,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 3, Line: 1, Column: 3},
							EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
						},
					},
				},
			},
			errs,
		)
	})

	t.Run("constant, missing end", func(t *testing.T) {

		t.Parallel()

		const code = "[T;1"
		_, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingBracketInArrayTypeError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
						Type: lexer.TokenEOF,
					},
				},
			},
			errs,
		)

		var missingBracketErr *MissingClosingBracketInArrayTypeError
		require.ErrorAs(t, errs[0], &missingBracketErr)

		fixes := missingBracketErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing bracket",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "]",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t, "[T;1]", fixes[0].TextEdits[0].ApplyTo(code))
	})
}

func TestParseOptionalType(t *testing.T) {

	t.Parallel()

	t.Run("nominal", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("Int?")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.OptionalType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				EndPos: ast.Position{Line: 1, Column: 3, Offset: 3},
			},
			result,
		)
	})

	t.Run("double", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("Int??")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.OptionalType{
				Type: &ast.OptionalType{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
						},
					},
					EndPos: ast.Position{Line: 1, Column: 3, Offset: 3},
				},
				EndPos: ast.Position{Line: 1, Column: 4, Offset: 4},
			},
			result,
		)
	})

	t.Run("triple", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("Int???")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.OptionalType{
				Type: &ast.OptionalType{
					Type: &ast.OptionalType{
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "Int",
								Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
							},
						},
						EndPos: ast.Position{Line: 1, Column: 3, Offset: 3},
					},
					EndPos: ast.Position{Line: 1, Column: 4, Offset: 4},
				},
				EndPos: ast.Position{Line: 1, Column: 5, Offset: 5},
			},
			result,
		)
	})
}

func TestParseReferenceType(t *testing.T) {

	t.Parallel()

	t.Run("unauthorized, nominal", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("&Int")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ReferenceType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("authorized, missing parens", func(t *testing.T) {

		t.Parallel()

		const code = "auth &Int"
		_, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingStartOfAuthorizationError{
					GotToken: lexer.Token{
						Type: lexer.TokenAmpersand,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
					},
				},
			},
			errs,
		)

		var missingParenErr *MissingStartOfAuthorizationError
		require.ErrorAs(t, errs[0], &missingParenErr)

		fixes := missingParenErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert opening parenthesis",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "(",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t, "auth( &Int", fixes[0].TextEdits[0].ApplyTo(code))
	})

	t.Run("authorized, missing ampersand", func(t *testing.T) {

		t.Parallel()

		const code = "auth(X) Int"
		_, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingAmpersandInAuthReferenceError{
					GotToken: lexer.Token{
						Type: lexer.TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
							EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
						},
					},
				},
			},
			errs,
		)

		var missingAmpersandErr *MissingAmpersandInAuthReferenceError
		require.ErrorAs(t, errs[0], &missingAmpersandErr)

		fixes := missingAmpersandErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert ampersand",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "&",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
								EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"auth(X) &Int",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("authorized, one entitlement", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("auth(X) &Int")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ReferenceType{
				Authorization: &ast.ConjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "X",
								Pos:        ast.Position{Line: 1, Column: 5, Offset: 5},
							},
						},
					},
				},
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("authorized, two conjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("auth(X, Y) &Int")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ReferenceType{
				Authorization: &ast.ConjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "X",
								Pos:        ast.Position{Line: 1, Column: 5, Offset: 5},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "Y",
								Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
							},
						},
					},
				},
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("authorized, two disjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("auth(X| Y) &Int")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ReferenceType{
				Authorization: &ast.DisjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "X",
								Pos:        ast.Position{Line: 1, Column: 5, Offset: 5},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "Y",
								Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
							},
						},
					},
				},
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("authorized, empty entitlements", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("auth() &Int")

		AssertEqualWithDiff(t,
			[]error{
				// TODO: improve
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
						Type: lexer.TokenParenClose,
					},
				},
			},
			errs,
		)
	})

	t.Run("authorized, mixed entitlements conjunction", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("auth(X, Y | Z) &Int")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenInsteadOfSeparatorError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 10, Line: 1, Column: 10},
							EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
						},
						Type: lexer.TokenVerticalBar,
					},
					ExpectedSeparator: lexer.TokenComma,
					ExpectedEndToken:  lexer.TokenParenClose,
				},
			},
			errs,
		)
	})

	t.Run("authorized, mixed entitlements conjunction", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("auth(X | Y, Z) &Int")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenInsteadOfSeparatorError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 10, Line: 1, Column: 10},
							EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
						},
						Type: lexer.TokenComma,
					},
					ExpectedSeparator: lexer.TokenVerticalBar,
					ExpectedEndToken:  lexer.TokenParenClose,
				},
			},
			errs,
		)
	})

	t.Run("authorized, missing closing paren after entitlements", func(t *testing.T) {

		t.Parallel()

		const code = "auth(X, Y"
		_, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingParenInAuthError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 9, Line: 1, Column: 9},
							EndPos:   ast.Position{Offset: 9, Line: 1, Column: 9},
						},
						Type: lexer.TokenEOF,
					},
				},
				&MissingAmpersandInAuthReferenceError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 9, Line: 1, Column: 9},
							EndPos:   ast.Position{Offset: 9, Line: 1, Column: 9},
						},
						Type: lexer.TokenEOF,
					},
				},
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 9, Line: 1, Column: 9},
							EndPos:   ast.Position{Offset: 9, Line: 1, Column: 9},
						},
						Type: lexer.TokenEOF,
					},
				},
			},
			errs,
		)

		var missingParenErr *MissingClosingParenInAuthError
		require.ErrorAs(t, errs[0], &missingParenErr)

		fixes := missingParenErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing parenthesis",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ")",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 9, Line: 1, Column: 9},
								EndPos:   ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"auth(X, Y)",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("authorized, map", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("auth ( mapping X ) & Int")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ReferenceType{
				Authorization: &ast.MappedAccess{
					EntitlementMap: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "X",
							Pos:        ast.Position{Line: 1, Column: 15, Offset: 15},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
				},
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Line: 1, Column: 21, Offset: 21},
					},
				},
				StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
			},
			result,
		)
	})

	t.Run("authorized, map no name", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("auth( mapping ) &Int")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
							EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
						},
						Type: lexer.TokenParenClose,
					},
				},
			},
			errs,
		)
	})

	t.Run("invalid, dot", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("&.")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						Type: lexer.TokenDot,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
							EndPos:   ast.Position{Offset: 1, Line: 1, Column: 1},
						},
					},
				},
			},
			errs,
		)
	})
}

func TestParseOptionalReferenceType(t *testing.T) {

	t.Parallel()

	t.Run("unauthorized", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("&Int?")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.OptionalType{
				Type: &ast.ReferenceType{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
				EndPos: ast.Position{Line: 1, Column: 4, Offset: 4},
			},
			result,
		)
	})
}

func TestParseIntersectionType(t *testing.T) {

	t.Parallel()

	t.Run("with old prefix and no types", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("T{}")

		AssertEqualWithDiff(t,
			[]error{
				&RestrictedTypeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
						EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
					},
				},
			},
			errs,
		)
	})

	t.Run("with old prefix and one type", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("T{U}")
		AssertEqualWithDiff(t,
			[]error{
				&RestrictedTypeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
						EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
					},
				},
			},
			errs,
		)
	})

	t.Run("with old prefix and two types", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("T{U , V }")
		AssertEqualWithDiff(t,
			[]error{
				&RestrictedTypeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
						EndPos:   ast.Position{Offset: 2, Line: 1, Column: 2},
					},
				},
			},
			errs,
		)
	})

	t.Run("no types", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{}")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.IntersectionType{
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("one type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{ T }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.IntersectionType{
				Types: []*ast.NominalType{
					{
						Identifier: ast.Identifier{
							Identifier: "T",
							Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
				},
			},
			result,
		)
	})

	t.Run("invalid: missing type after comma", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{ T , }")
		AssertEqualWithDiff(t,
			[]error{
				&MissingTypeAfterCommaInIntersectionError{
					Pos: ast.Position{Offset: 6, Line: 1, Column: 6},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			&ast.IntersectionType{
				Types: []*ast.NominalType{
					{
						Identifier: ast.Identifier{
							Identifier: "T",
							Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 6, Offset: 6},
				},
			},
			result,
		)
	})

	t.Run("invalid: type without comma", func(t *testing.T) {

		t.Parallel()

		const code = "{ T U }"
		result, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingSeparatorInIntersectionOrDictionaryTypeError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)

		var missingSepErr *MissingSeparatorInIntersectionOrDictionaryTypeError
		require.ErrorAs(t, errs[0], &missingSepErr)

		fixes := missingSepErr.SuggestFixes(code)
		AssertEqualWithDiff(
			t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert comma",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ",",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 3, Line: 1, Column: 3},
								EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
							},
						},
					},
				},
				{
					Message: "Insert colon",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ":",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 3, Line: 1, Column: 3},
								EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t, "{ T, U }", fixes[0].TextEdits[0].ApplyTo(code))
		assert.Equal(t, "{ T: U }", fixes[1].TextEdits[0].ApplyTo(code))
	})

	t.Run("invalid: colon", func(t *testing.T) {

		t.Parallel()

		const code = "{ T , U : V }"
		result, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedColonInIntersectionTypeError{
					Pos: ast.Position{Offset: 8, Line: 1, Column: 8},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)

		var unexpectedColonErr *UnexpectedColonInIntersectionTypeError
		require.ErrorAs(t, errs[0], &unexpectedColonErr)

		fixes := unexpectedColonErr.SuggestFixes(code)
		AssertEqualWithDiff(
			t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Replace colon with comma",
					TextEdits: []ast.TextEdit{
						{
							Replacement: ",",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
								EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"{ T , U , V }",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("invalid: colon", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{U , V : W }")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedColonInIntersectionTypeError{
					Pos: ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)
	})

	t.Run("invalid: first is non-nominal", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{[T]}")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidNonNominalTypeInIntersectionError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
					},
				},
			},
			errs,
		)

		// TODO: return type with non-nominal types
		assert.Nil(t, result)
	})

	t.Run("invalid: second is non-nominal", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{T, [U]}")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidNonNominalTypeInIntersectionError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
						EndPos:   ast.Position{Offset: 6, Line: 1, Column: 6},
					},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)
	})

	t.Run("invalid: two, first is non-nominal", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{[U], T}")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidNonNominalTypeInIntersectionError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
					},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)
	})

	t.Run("invalid: missing end", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedEOFExpectedTypeError{
					Pos: ast.Position{Offset: 1, Line: 1, Column: 1},
				},
			},
			errs,
		)

		assert.Nil(t, result)
	})

	t.Run("invalid: missing end after type", func(t *testing.T) {

		t.Parallel()

		const code = "{U"
		_, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingBraceInIntersectionOrDictionaryTypeError{
					Pos: ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			errs,
		)

		var missingBraceErr *MissingClosingBraceInIntersectionOrDictionaryTypeError
		require.ErrorAs(t, errs[0], &missingBraceErr)

		fixes := missingBraceErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing brace",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "}",
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

		assert.Equal(t,
			"{U}",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("invalid: missing end after comma", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{U,")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedEOFExpectedTypeError{
					Pos: ast.Position{Offset: 3, Line: 1, Column: 3},
				},
			},
			errs,
		)

		assert.Nil(t, result)
	})

	t.Run("invalid: just comma", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{,}")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedCommaInIntersectionTypeError{
					Pos: ast.Position{Offset: 1, Line: 1, Column: 1},
				},
			},
			errs,
		)

		assert.Nil(t, result)
	})

	t.Run("invalid: leading comma", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{ , T }")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedCommaInIntersectionTypeError{
					Pos: ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)
	})
}

func TestParseDictionaryType(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{T: U}")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.DictionaryType{
				KeyType: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				ValueType: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "U",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
				},
			},
			result,
		)
	})

	t.Run("invalid, missing value type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{T:}")
		AssertEqualWithDiff(t,
			[]error{
				&MissingDictionaryValueTypeError{
					Pos: ast.Position{Offset: 3, Line: 1, Column: 3},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			&ast.DictionaryType{
				KeyType: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				ValueType: nil,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
				},
			},
			result,
		)
	})

	t.Run("invalid, missing key and value type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{:}")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedColonInDictionaryTypeError{
					Pos: ast.Position{Offset: 1, Line: 1, Column: 1},
				},
			},
			errs,
		)

		assert.Nil(t, result)
	})

	t.Run("invalid, missing key type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{:U}")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedColonInDictionaryTypeError{
					Pos: ast.Position{Offset: 1, Line: 1, Column: 1},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)
	})

	t.Run("invalid, unexpected comma after value type", func(t *testing.T) {

		t.Parallel()

		const code = "{T:U,}"
		result, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedCommaInDictionaryTypeError{
					Pos: ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)

		var unexpectedCommaErr *UnexpectedCommaInDictionaryTypeError
		require.ErrorAs(t, errs[0], &unexpectedCommaErr)

		fixes := unexpectedCommaErr.SuggestFixes(code)
		AssertEqualWithDiff(
			t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Remove comma",
					TextEdits: []ast.TextEdit{
						{
							Replacement: "",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"{T:U}",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("invalid, unexpected colon after value type", func(t *testing.T) {

		t.Parallel()

		const code = "{T:U:}"
		result, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MultipleColonInDictionaryTypeError{
					Pos: ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)

		var multipleColonErr *MultipleColonInDictionaryTypeError
		require.ErrorAs(t, errs[0], &multipleColonErr)

		fixes := multipleColonErr.SuggestFixes(code)
		AssertEqualWithDiff(
			t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Remove extra colon",
					TextEdits: []ast.TextEdit{
						{
							Replacement: "",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"{T:U}",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("invalid, unexpected colon after colon", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{T::U}")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedColonInDictionaryTypeError{
					Pos: ast.Position{Offset: 3, Line: 1, Column: 3},
				},
			},
			errs,
		)

		// TODO: return type
		assert.Nil(t, result)
	})

	t.Run("invalid, missing value type after colon", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("{T:")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedEOFExpectedTypeError{
					Pos: ast.Position{Offset: 3, Line: 1, Column: 3},
				},
			},
			errs,
		)

		assert.Nil(t, result)
	})

	t.Run("invalid, missing end after key type  and value type", func(t *testing.T) {

		t.Parallel()

		const code = "{T:U"
		_, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingBraceInIntersectionOrDictionaryTypeError{
					Pos: ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)

		var missingBraceErr *MissingClosingBraceInIntersectionOrDictionaryTypeError
		require.ErrorAs(t, errs[0], &missingBraceErr)

		fixes := missingBraceErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing brace",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "}",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"{T:U}",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})
}

func TestParseFunctionType(t *testing.T) {

	t.Parallel()

	t.Run("no parameters, Void return type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("fun():Void")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FunctionType{
				PurityAnnotation:         ast.FunctionPurityUnspecified,
				ParameterTypeAnnotations: nil,
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Void",
							Pos:        ast.Position{Line: 1, Column: 6, Offset: 6},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
				},
			},
			result,
		)
	})

	t.Run("no parameters, no return type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("fun()")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FunctionType{
				PurityAnnotation: ast.FunctionPurityUnspecified,
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "",
							Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 4, Offset: 4},
				},
			},
			result,
		)
	})

	t.Run("no parameter list", func(t *testing.T) {

		t.Parallel()

		const code = "fun"
		_, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingOpeningParenInFunctionTypeError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 3, Line: 1, Column: 3},
							EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
						},
						Type: lexer.TokenEOF,
					},
				},
				&UnexpectedEOFExpectedTypeAnnotationError{
					Pos: ast.Position{Offset: 3, Line: 1, Column: 3},
				},
			},
			errs,
		)

		var missingParenErr *MissingOpeningParenInFunctionTypeError
		require.ErrorAs(t, errs[0], &missingParenErr)

		fixes := missingParenErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert opening parenthesis",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "(",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 3, Line: 1, Column: 3},
								EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"fun(",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("view function type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("view fun ():Void")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FunctionType{
				PurityAnnotation:         ast.FunctionPurityView,
				ParameterTypeAnnotations: nil,
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Void",
							Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
					EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
				},
			},
			result,
		)
	})

	t.Run("missing closing paren, no types", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("fun(")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedEOFExpectedTypeAnnotationError{
					Pos: ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)
	})

	t.Run("missing closing paren, one type", func(t *testing.T) {

		t.Parallel()

		const code = "fun(Int"
		_, errs := testParseType(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingParenInFunctionTypeError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
							EndPos:   ast.Position{Offset: 7, Line: 1, Column: 7},
						},
						Type: lexer.TokenEOF,
					},
				},
			},
			errs,
		)

		var missingParenErr *MissingClosingParenInFunctionTypeError
		require.ErrorAs(t, errs[0], &missingParenErr)

		fixes := missingParenErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing parenthesis",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ")",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
								EndPos:   ast.Position{Offset: 7, Line: 1, Column: 7},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t, "fun(Int)", fixes[0].TextEdits[0].ApplyTo(code))
	})

	t.Run("invalid, leading comma", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("fun(,) : Void")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedCommaInTypeAnnotationListError{
					Pos: ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)
	})

	t.Run("three parameters, Int return type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("fun( String , Bool , @R ) : Int")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FunctionType{
				ParameterTypeAnnotations: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "String",
								Pos:        ast.Position{Line: 1, Column: 5, Offset: 5},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 5, Offset: 5},
					},
					{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "Bool",
								Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
					},
					{
						IsResource: true,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "R",
								Pos:        ast.Position{Line: 1, Column: 22, Offset: 22},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 21, Offset: 21},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 28, Offset: 28},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 28, Offset: 28},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 30, Offset: 30},
				},
			},
			result,
		)
	})
}

func TestParseInstantiationType(t *testing.T) {

	t.Parallel()

	t.Run("no type arguments", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("T<>")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.InstantiationType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArgumentsStartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
				EndPos:                ast.Position{Line: 1, Column: 2, Offset: 2},
			},
			result,
		)
	})

	t.Run("one type argument, no spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("T<U>")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.InstantiationType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "U",
								Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				TypeArgumentsStartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
				EndPos:                ast.Position{Line: 1, Column: 3, Offset: 3},
			},
			result,
		)
	})

	t.Run("one type argument, with spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("T< U >")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.InstantiationType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "U",
								Pos:        ast.Position{Line: 1, Column: 3, Offset: 3},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
					},
				},
				TypeArgumentsStartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
				EndPos:                ast.Position{Line: 1, Column: 5, Offset: 5},
			},
			result,
		)
	})

	t.Run("two type arguments, with spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("T< U , @V >")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.InstantiationType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "U",
								Pos:        ast.Position{Line: 1, Column: 3, Offset: 3},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
					},
					{
						IsResource: true,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "V",
								Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
					},
				},
				TypeArgumentsStartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
				EndPos:                ast.Position{Line: 1, Column: 10, Offset: 10},
			},
			result,
		)
	})

	t.Run("one type argument, no spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("T<U>")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.InstantiationType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "U",
								Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				TypeArgumentsStartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
				EndPos:                ast.Position{Line: 1, Column: 3, Offset: 3},
			},
			result,
		)
	})

	t.Run("one type argument, nested, with spaces", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("T< U< V >  >")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.InstantiationType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.InstantiationType{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "U",
									Pos:        ast.Position{Line: 1, Column: 3, Offset: 3},
								},
							},
							TypeArguments: []*ast.TypeAnnotation{
								{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "V",
											Pos:        ast.Position{Line: 1, Column: 6, Offset: 6},
										},
									},
									StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
								},
							},
							TypeArgumentsStartPos: ast.Position{Line: 1, Column: 4, Offset: 4},
							EndPos:                ast.Position{Line: 1, Column: 8, Offset: 8},
						},
						StartPos: ast.Position{Line: 1, Column: 3, Offset: 3},
					},
				},
				TypeArgumentsStartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
				EndPos:                ast.Position{Line: 1, Column: 11, Offset: 11},
			},
			result,
		)
	})

	t.Run("invalid: unexpected comma", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("T<,U>")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedCommaInTypeAnnotationListError{
					Pos: ast.Position{Offset: 2, Line: 1, Column: 2},
				},
			},
			errs,
		)

		assert.Nil(t, result)
	})

	t.Run("invalid: missing separator", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("T<U V>")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenInsteadOfSeparatorError{
					GotToken: lexer.Token{
						Type: lexer.TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
					},
					ExpectedSeparator: lexer.TokenComma,
					ExpectedEndToken:  lexer.TokenGreater,
				},
			},
			errs,
		)

		assert.Nil(t, result)
	})

	t.Run("invalid: missing closing greater", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseType("T<U")
		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingGreaterInTypeArgumentsError{
					Pos: ast.Position{Offset: 3, Line: 1, Column: 3},
				},
			},
			errs,
		)
	})

	t.Run("invalid: missing type after comma", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseType("T<U,>")
		AssertEqualWithDiff(t,
			[]error{
				&MissingTypeAnnotationAfterCommaError{
					Pos: ast.Position{Offset: 4, Line: 1, Column: 4},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			&ast.InstantiationType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Offset: 0, Line: 1, Column: 0},
					},
				},
				TypeArguments: []*ast.TypeAnnotation{
					{
						IsResource: false,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "U",
								Pos:        ast.Position{Offset: 2, Line: 1, Column: 2},
							},
						},
						StartPos: ast.Position{Offset: 2, Line: 1, Column: 2},
					},
				},
				TypeArgumentsStartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
				EndPos:                ast.Position{Offset: 4, Line: 1, Column: 4},
			},
			result,
		)
	})
}

func TestParseParametersAndArrayTypes(t *testing.T) {

	t.Parallel()

	const code = `
		access(all) fun test(a: Int32, b: [Int32; 2], c: [[Int32; 3]]): [[Int64]] {}
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							TypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int32",
										Pos: ast.Position{
											Offset: 27,
											Line:   2,
											Column: 26,
										},
									},
								},
								StartPos: ast.Position{
									Offset: 27,
									Line:   2,
									Column: 26,
								},
								IsResource: false,
							},
							Label: "",
							Identifier: ast.Identifier{
								Identifier: "a",
								Pos: ast.Position{
									Offset: 24,
									Line:   2,
									Column: 23,
								},
							},
							StartPos: ast.Position{
								Offset: 24,
								Line:   2,
								Column: 23,
							},
						},
						{
							TypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.ConstantSizedType{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int32",
											Pos: ast.Position{
												Offset: 38,
												Line:   2,
												Column: 37,
											},
										},
									},
									Size: &ast.IntegerExpression{
										Value: big.NewInt(2),
										PositiveLiteral: []uint8{
											0x32,
										},
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 45,
												Line:   2,
												Column: 44,
											},
											EndPos: ast.Position{
												Offset: 45,
												Line:   2,
												Column: 44,
											},
										},
										Base: 10,
									},
									Range: ast.Range{
										StartPos: ast.Position{
											Offset: 37,
											Line:   2,
											Column: 36,
										},
										EndPos: ast.Position{
											Offset: 46,
											Line:   2,
											Column: 45,
										},
									},
								},
								StartPos: ast.Position{
									Offset: 37,
									Line:   2,
									Column: 36,
								},
								IsResource: false,
							},
							Identifier: ast.Identifier{
								Identifier: "b",
								Pos: ast.Position{
									Offset: 34,
									Line:   2,
									Column: 33,
								},
							},
							StartPos: ast.Position{
								Offset: 34,
								Line:   2,
								Column: 33,
							},
						},
						{
							TypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.VariableSizedType{
									Type: &ast.ConstantSizedType{
										Type: &ast.NominalType{
											Identifier: ast.Identifier{
												Identifier: "Int32",
												Pos: ast.Position{
													Offset: 54,
													Line:   2,
													Column: 53,
												},
											},
										},
										Size: &ast.IntegerExpression{
											Value: big.NewInt(3),
											PositiveLiteral: []uint8{
												0x33,
											},
											Range: ast.Range{
												StartPos: ast.Position{
													Offset: 61,
													Line:   2,
													Column: 60,
												},
												EndPos: ast.Position{
													Offset: 61,
													Line:   2,
													Column: 60,
												},
											},
											Base: 10,
										},
										Range: ast.Range{
											StartPos: ast.Position{
												Offset: 53,
												Line:   2,
												Column: 52,
											},
											EndPos: ast.Position{
												Offset: 62,
												Line:   2,
												Column: 61,
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{
											Offset: 52,
											Line:   2,
											Column: 51,
										},
										EndPos: ast.Position{
											Offset: 63,
											Line:   2,
											Column: 62,
										},
									},
								},
								StartPos: ast.Position{
									Offset: 52,
									Line:   2,
									Column: 51,
								},
								IsResource: false,
							},
							Identifier: ast.Identifier{
								Identifier: "c",
								Pos: ast.Position{
									Offset: 49,
									Line:   2,
									Column: 48,
								},
							},
							StartPos: ast.Position{
								Offset: 49,
								Line:   2,
								Column: 48,
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{
							Offset: 23,
							Line:   2,
							Column: 22,
						},
						EndPos: ast.Position{
							Offset: 64,
							Line:   2,
							Column: 63,
						},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.VariableSizedType{
						Type: &ast.VariableSizedType{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int64",
									Pos: ast.Position{
										Offset: 69,
										Line:   2,
										Column: 68,
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{
									Offset: 68,
									Line:   2,
									Column: 67,
								},
								EndPos: ast.Position{
									Offset: 74,
									Line:   2,
									Column: 73,
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{
								Offset: 67,
								Line:   2,
								Column: 66,
							},
							EndPos: ast.Position{
								Offset: 75,
								Line:   2,
								Column: 74,
							},
						},
					},
					StartPos: ast.Position{
						Offset: 67,
						Line:   2,
						Column: 66,
					},
					IsResource: false,
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{
								Offset: 77,
								Line:   2,
								Column: 76,
							},
							EndPos: ast.Position{
								Offset: 78,
								Line:   2,
								Column: 77,
							},
						},
					},
				},
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos: ast.Position{
						Offset: 19,
						Line:   2,
						Column: 18,
					},
				},
				StartPos: ast.Position{
					Offset: 3,
					Line:   2,
					Column: 2,
				},
				Access: ast.AccessAll,
				Flags:  0x00,
			},
		},
		result.Declarations(),
	)
}

func TestParseDictionaryTypeInVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
	    let x: {String: Int} = {}
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{Identifier: "x",
					Pos: ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.DictionaryType{
						KeyType: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "String",
								Pos:        ast.Position{Offset: 14, Line: 2, Column: 13},
							},
						},
						ValueType: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "Int",
								Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
						},
					},
					StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 27, Line: 2, Column: 26},
				},
				Value: &ast.DictionaryExpression{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 29, Line: 2, Column: 28},
						EndPos:   ast.Position{Offset: 30, Line: 2, Column: 29},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseIntegerTypes(t *testing.T) {

	t.Parallel()

	const code = `
		let a: Int8 = 1
		let b: Int16 = 2
		let c: Int32 = 3
		let d: Int64 = 4
		let e: UInt8 = 5
		let f: UInt16 = 6
		let g: UInt32 = 7
		let h: UInt64 = 8
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	a := &ast.VariableDeclaration{
		Access: ast.AccessNotSpecified,
		Identifier: ast.Identifier{
			Identifier: "a",
			Pos:        ast.Position{Offset: 7, Line: 2, Column: 6},
		},

		IsConstant: true,
		TypeAnnotation: &ast.TypeAnnotation{
			IsResource: false,
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "Int8",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
			},
			StartPos: ast.Position{Offset: 10, Line: 2, Column: 9},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
		},
		Value: &ast.IntegerExpression{
			PositiveLiteral: []byte("1"),
			Value:           big.NewInt(1),
			Base:            10,
			Range: ast.Range{
				StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
				EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
			},
		},
		StartPos: ast.Position{Offset: 3, Line: 2, Column: 2},
	}
	b := &ast.VariableDeclaration{
		Access: ast.AccessNotSpecified,
		Identifier: ast.Identifier{
			Identifier: "b",
			Pos:        ast.Position{Offset: 25, Line: 3, Column: 6},
		},
		IsConstant: true,
		TypeAnnotation: &ast.TypeAnnotation{
			IsResource: false,
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "Int16",
					Pos:        ast.Position{Offset: 28, Line: 3, Column: 9},
				},
			},
			StartPos: ast.Position{Offset: 28, Line: 3, Column: 9},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 34, Line: 3, Column: 15},
		},
		Value: &ast.IntegerExpression{
			PositiveLiteral: []byte("2"),
			Value:           big.NewInt(2),
			Base:            10,
			Range: ast.Range{
				StartPos: ast.Position{Offset: 36, Line: 3, Column: 17},
				EndPos:   ast.Position{Offset: 36, Line: 3, Column: 17},
			},
		},
		StartPos: ast.Position{Offset: 21, Line: 3, Column: 2},
	}
	c := &ast.VariableDeclaration{
		Access: ast.AccessNotSpecified,
		Identifier: ast.Identifier{
			Identifier: "c",
			Pos:        ast.Position{Offset: 44, Line: 4, Column: 6},
		},
		IsConstant: true,
		TypeAnnotation: &ast.TypeAnnotation{
			IsResource: false,
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "Int32",
					Pos:        ast.Position{Offset: 47, Line: 4, Column: 9},
				},
			},
			StartPos: ast.Position{Offset: 47, Line: 4, Column: 9},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 53, Line: 4, Column: 15},
		},
		Value: &ast.IntegerExpression{
			PositiveLiteral: []byte("3"),
			Value:           big.NewInt(3),
			Base:            10,
			Range: ast.Range{
				StartPos: ast.Position{Offset: 55, Line: 4, Column: 17},
				EndPos:   ast.Position{Offset: 55, Line: 4, Column: 17},
			},
		},
		StartPos: ast.Position{Offset: 40, Line: 4, Column: 2},
	}
	d := &ast.VariableDeclaration{
		Access: ast.AccessNotSpecified,
		Identifier: ast.Identifier{
			Identifier: "d",
			Pos:        ast.Position{Offset: 63, Line: 5, Column: 6},
		},
		IsConstant: true,
		TypeAnnotation: &ast.TypeAnnotation{
			IsResource: false,
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "Int64",
					Pos:        ast.Position{Offset: 66, Line: 5, Column: 9},
				},
			},
			StartPos: ast.Position{Offset: 66, Line: 5, Column: 9},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 72, Line: 5, Column: 15},
		},
		Value: &ast.IntegerExpression{
			PositiveLiteral: []byte("4"),
			Value:           big.NewInt(4),
			Base:            10,
			Range: ast.Range{
				StartPos: ast.Position{Offset: 74, Line: 5, Column: 17},
				EndPos:   ast.Position{Offset: 74, Line: 5, Column: 17},
			},
		},
		StartPos: ast.Position{Offset: 59, Line: 5, Column: 2},
	}
	e := &ast.VariableDeclaration{
		Access: ast.AccessNotSpecified,
		Identifier: ast.Identifier{
			Identifier: "e",
			Pos:        ast.Position{Offset: 82, Line: 6, Column: 6},
		},
		IsConstant: true,
		TypeAnnotation: &ast.TypeAnnotation{
			IsResource: false,
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "UInt8",
					Pos:        ast.Position{Offset: 85, Line: 6, Column: 9},
				},
			},
			StartPos: ast.Position{Offset: 85, Line: 6, Column: 9},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 91, Line: 6, Column: 15},
		},
		Value: &ast.IntegerExpression{
			PositiveLiteral: []byte("5"),
			Value:           big.NewInt(5),
			Base:            10,
			Range: ast.Range{
				StartPos: ast.Position{Offset: 93, Line: 6, Column: 17},
				EndPos:   ast.Position{Offset: 93, Line: 6, Column: 17},
			},
		},
		StartPos: ast.Position{Offset: 78, Line: 6, Column: 2},
	}
	f := &ast.VariableDeclaration{
		Access: ast.AccessNotSpecified,
		Identifier: ast.Identifier{
			Identifier: "f",
			Pos:        ast.Position{Offset: 101, Line: 7, Column: 6},
		},
		IsConstant: true,
		TypeAnnotation: &ast.TypeAnnotation{
			IsResource: false,
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "UInt16",
					Pos:        ast.Position{Offset: 104, Line: 7, Column: 9},
				},
			},
			StartPos: ast.Position{Offset: 104, Line: 7, Column: 9},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 111, Line: 7, Column: 16},
		},
		Value: &ast.IntegerExpression{
			PositiveLiteral: []byte("6"),
			Value:           big.NewInt(6),
			Base:            10,
			Range: ast.Range{
				StartPos: ast.Position{Offset: 113, Line: 7, Column: 18},
				EndPos:   ast.Position{Offset: 113, Line: 7, Column: 18},
			},
		},
		StartPos: ast.Position{Offset: 97, Line: 7, Column: 2},
	}
	g := &ast.VariableDeclaration{
		Access: ast.AccessNotSpecified,
		Identifier: ast.Identifier{
			Identifier: "g",
			Pos:        ast.Position{Offset: 121, Line: 8, Column: 6},
		},
		IsConstant: true,
		TypeAnnotation: &ast.TypeAnnotation{
			IsResource: false,
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "UInt32",
					Pos:        ast.Position{Offset: 124, Line: 8, Column: 9},
				},
			},
			StartPos: ast.Position{Offset: 124, Line: 8, Column: 9},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 131, Line: 8, Column: 16},
		},
		Value: &ast.IntegerExpression{
			PositiveLiteral: []byte("7"),
			Value:           big.NewInt(7),
			Base:            10,
			Range: ast.Range{
				StartPos: ast.Position{Offset: 133, Line: 8, Column: 18},
				EndPos:   ast.Position{Offset: 133, Line: 8, Column: 18},
			},
		},
		StartPos: ast.Position{Offset: 117, Line: 8, Column: 2},
	}
	h := &ast.VariableDeclaration{
		Access: ast.AccessNotSpecified,
		Identifier: ast.Identifier{
			Identifier: "h",
			Pos:        ast.Position{Offset: 141, Line: 9, Column: 6},
		},
		IsConstant: true,
		TypeAnnotation: &ast.TypeAnnotation{
			IsResource: false,
			Type: &ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "UInt64",
					Pos:        ast.Position{Offset: 144, Line: 9, Column: 9},
				},
			},
			StartPos: ast.Position{Offset: 144, Line: 9, Column: 9},
		},
		Transfer: &ast.Transfer{
			Operation: ast.TransferOperationCopy,
			Pos:       ast.Position{Offset: 151, Line: 9, Column: 16},
		},
		Value: &ast.IntegerExpression{
			PositiveLiteral: []byte("8"),
			Value:           big.NewInt(8),
			Base:            10,
			Range: ast.Range{
				StartPos: ast.Position{Offset: 153, Line: 9, Column: 18},
				EndPos:   ast.Position{Offset: 153, Line: 9, Column: 18},
			},
		},
		StartPos: ast.Position{Offset: 137, Line: 9, Column: 2},
	}

	AssertEqualWithDiff(t,
		[]ast.Declaration{a, b, c, d, e, f, g, h},
		result.Declarations(),
	)
}

func TestParseFunctionTypeInVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
		let add: fun(Int8, Int16): Int32 = nothing
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "add",
					Pos:        ast.Position{Offset: 7, Line: 2, Column: 6},
				},
				IsConstant: true,
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.FunctionType{
						ParameterTypeAnnotations: []*ast.TypeAnnotation{
							{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int8",
										Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
									},
								},
								StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
							},
							{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int16",
										Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
									},
								},
								StartPos: ast.Position{Offset: 22, Line: 2, Column: 21},
							},
						},
						ReturnTypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int32",
									Pos:        ast.Position{Offset: 30, Line: 2, Column: 29},
								},
							},
							StartPos: ast.Position{Offset: 30, Line: 2, Column: 29},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 12, Line: 2, Column: 11},
							EndPos:   ast.Position{Offset: 34, Line: 2, Column: 33},
						},
					},
					StartPos: ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 36, Line: 2, Column: 35},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "nothing",
						Pos:        ast.Position{Offset: 38, Line: 2, Column: 37},
					},
				},
				StartPos: ast.Position{Offset: 3, Line: 2, Column: 2},
			},
		},
		result.Declarations(),
	)
}

func TestParseFunctionArrayType(t *testing.T) {

	t.Parallel()

	const code = `
		let test: [fun(Int8): Int16; 2] = []
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 7, Line: 2, Column: 6},
				},

				IsConstant: true,
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.ConstantSizedType{
						Type: &ast.FunctionType{
							ParameterTypeAnnotations: []*ast.TypeAnnotation{
								{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int8",
											Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
										},
									},
									StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
								},
							},
							ReturnTypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int16",
										Pos:        ast.Position{Offset: 25, Line: 2, Column: 24},
									},
								},
								StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
								EndPos:   ast.Position{Offset: 29, Line: 2, Column: 28},
							},
						},
						Size: &ast.IntegerExpression{
							PositiveLiteral: []byte("2"),
							Value:           big.NewInt(2),
							Base:            10,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 32, Line: 2, Column: 31},
								EndPos:   ast.Position{Offset: 32, Line: 2, Column: 31},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 33, Line: 2, Column: 32},
						},
					},
					StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 35, Line: 2, Column: 34},
				},
				Value: &ast.ArrayExpression{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 37, Line: 2, Column: 36},
						EndPos:   ast.Position{Offset: 38, Line: 2, Column: 37},
					},
				},
				StartPos: ast.Position{Offset: 3, Line: 2, Column: 2},
			},
		},
		result.Declarations(),
	)
}

func TestParseFunctionTypeWithArrayReturnType(t *testing.T) {

	t.Parallel()

	const code = `
		let test: fun(Int8): [Int16; 2] = nothing
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 7, Line: 2, Column: 6},
				},
				IsConstant: true,
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.FunctionType{
						ParameterTypeAnnotations: []*ast.TypeAnnotation{
							{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int8",
										Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
									},
								},
								StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							},
						},
						ReturnTypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.ConstantSizedType{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int16",
										Pos:        ast.Position{Offset: 25, Line: 2, Column: 24},
									},
								},
								Size: &ast.IntegerExpression{
									PositiveLiteral: []byte("2"),
									Value:           big.NewInt(2),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 32, Line: 2, Column: 31},
										EndPos:   ast.Position{Offset: 32, Line: 2, Column: 31},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
									EndPos:   ast.Position{Offset: 33, Line: 2, Column: 32},
								},
							},
							StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 33, Line: 2, Column: 32},
						},
					},
					StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 35, Line: 2, Column: 34},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "nothing",
						Pos:        ast.Position{Offset: 37, Line: 2, Column: 36},
					},
				},
				StartPos: ast.Position{Offset: 3, Line: 2, Column: 2},
			},
		},
		result.Declarations(),
	)
}

func TestParseFunctionTypeWithFunctionReturnTypeInParentheses(t *testing.T) {

	t.Parallel()

	const code = `
		let test: fun(Int8): (fun(Int16): Int32) = nothing
	`
	_, errs := testParseProgram(code)

	require.Empty(t, errs)
}

func TestParseFunctionTypeWithFunctionReturnType(t *testing.T) {

	t.Parallel()

	const code = `
		let test: fun(Int8): fun(Int16): Int32 = nothing
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 7, Line: 2, Column: 6},
				},
				IsConstant: true,
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.FunctionType{
						ParameterTypeAnnotations: []*ast.TypeAnnotation{
							{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int8",
										Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
									},
								},
								StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							},
						},
						ReturnTypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.FunctionType{
								ParameterTypeAnnotations: []*ast.TypeAnnotation{
									{
										IsResource: false,
										Type: &ast.NominalType{
											Identifier: ast.Identifier{
												Identifier: "Int16",
												Pos:        ast.Position{Offset: 28, Line: 2, Column: 27},
											},
										},
										StartPos: ast.Position{Offset: 28, Line: 2, Column: 27},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int32",
											Pos:        ast.Position{Offset: 36, Line: 2, Column: 35},
										},
									},
									StartPos: ast.Position{Offset: 36, Line: 2, Column: 35},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
									EndPos:   ast.Position{Offset: 40, Line: 2, Column: 39},
								},
							},
							StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 40, Line: 2, Column: 39},
						},
					},
					StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 42, Line: 2, Column: 41},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "nothing",
						Pos:        ast.Position{Offset: 44, Line: 2, Column: 43},
					},
				},
				StartPos: ast.Position{Offset: 3, Line: 2, Column: 2},
			},
		},
		result.Declarations(),
	)
}

func TestParseViewFunctionTypeWithNewSyntax(t *testing.T) {
	t.Parallel()

	code := `
		let test: view     fun(Int8): fun(Int16): Int32 = nothing
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	expected := []ast.Declaration{
		&ast.VariableDeclaration{
			Access:     ast.AccessNotSpecified,
			IsConstant: true,
			Identifier: ast.Identifier{
				Identifier: "test",
				Pos:        ast.Position{Offset: 7, Line: 2, Column: 6},
			},
			TypeAnnotation: &ast.TypeAnnotation{
				Type: &ast.FunctionType{
					PurityAnnotation: ast.FunctionPurityView,
					ParameterTypeAnnotations: []*ast.TypeAnnotation{
						{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int8",
									Pos:        ast.Position{Offset: 26, Line: 2, Column: 25},
								},
							},
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						Type: &ast.FunctionType{
							ParameterTypeAnnotations: []*ast.TypeAnnotation{
								{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int16",
											Pos:        ast.Position{Offset: 37, Line: 2, Column: 36},
										},
									},
									StartPos: ast.Position{Offset: 37, Line: 2, Column: 36},
								},
							},
							ReturnTypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int32",
										Pos:        ast.Position{Offset: 45, Line: 2, Column: 44},
									},
								},
								StartPos: ast.Position{Offset: 45, Line: 2, Column: 44},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 33, Line: 2, Column: 32},
								EndPos:   ast.Position{Offset: 49, Line: 2, Column: 48},
							},
						},
						StartPos: ast.Position{Offset: 33, Line: 2, Column: 32},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 49, Line: 2, Column: 48},
					},
				},
				StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
			},
			Value: &ast.IdentifierExpression{
				Identifier: ast.Identifier{
					Identifier: "nothing",
					Pos:        ast.Position{Offset: 53, Line: 2, Column: 52},
				},
			},
			Transfer: &ast.Transfer{
				Operation: 1,
				Pos:       ast.Position{Offset: 51, Line: 2, Column: 50},
			},
			StartPos: ast.Position{Offset: 3, Line: 2, Column: 2},
		},
	}
	AssertEqualWithDiff(t, expected, result.Declarations())
}

func TestParseNewSyntaxFunctionType(t *testing.T) {
	t.Parallel()

	code := `
		let test: fun(Int8): fun(Int16): Int32 = nothing
	`

	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	expected := []ast.Declaration{
		&ast.VariableDeclaration{
			Access:     ast.AccessNotSpecified,
			IsConstant: true,
			Identifier: ast.Identifier{
				Identifier: "test",
				Pos:        ast.Position{Offset: 7, Line: 2, Column: 6},
			},
			TypeAnnotation: &ast.TypeAnnotation{
				Type: &ast.FunctionType{
					ParameterTypeAnnotations: []*ast.TypeAnnotation{
						{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int8",
									Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
								},
							},
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						Type: &ast.FunctionType{
							ParameterTypeAnnotations: []*ast.TypeAnnotation{
								{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int16",
											Pos:        ast.Position{Offset: 28, Line: 2, Column: 27},
										},
									},
									StartPos: ast.Position{Offset: 28, Line: 2, Column: 27},
								},
							},
							ReturnTypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int32",
										Pos:        ast.Position{Offset: 36, Line: 2, Column: 35},
									},
								},
								StartPos: ast.Position{Offset: 36, Line: 2, Column: 35},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
								EndPos:   ast.Position{Offset: 40, Line: 2, Column: 39},
							},
						},
						StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
						EndPos:   ast.Position{Offset: 40, Line: 2, Column: 39},
					},
				},
				StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
			},
			Value: &ast.IdentifierExpression{
				Identifier: ast.Identifier{
					Identifier: "nothing",
					Pos:        ast.Position{Offset: 44, Line: 2, Column: 43},
				},
			},
			Transfer: &ast.Transfer{
				Operation: 1,
				Pos:       ast.Position{Offset: 42, Line: 2, Column: 41},
			},
			StartPos: ast.Position{Offset: 3, Line: 2, Column: 2},
		},
	}
	AssertEqualWithDiff(t, expected, result.Declarations())
}

func TestParseOptionalTypeDouble(t *testing.T) {

	t.Parallel()

	const code = `
       let x: Int?? = 1
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.OptionalType{
						Type: &ast.OptionalType{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Offset: 15, Line: 2, Column: 14},
								},
							},
							EndPos: ast.Position{Offset: 18, Line: 2, Column: 17},
						},
						EndPos: ast.Position{Offset: 19, Line: 2, Column: 18},
					},
					StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 21, Line: 2, Column: 20},
				},
				Value: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
						EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
					},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result.Declarations(),
	)
}

func TestParseFunctionTypeWithResourceTypeAnnotation(t *testing.T) {

	t.Parallel()

	const code = `
        let f: fun(): @R = g
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "f",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.FunctionType{
						ParameterTypeAnnotations: nil,
						ReturnTypeAnnotation: &ast.TypeAnnotation{
							IsResource: true,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "R",
									Pos:        ast.Position{Offset: 24, Line: 2, Column: 23},
								},
							},
							StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
							EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
						},
					},
					StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 26, Line: 2, Column: 25},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "g",
						Pos:        ast.Position{Offset: 28, Line: 2, Column: 27},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseReferenceTypeInVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
       let x: &[&R] = 1
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.ReferenceType{
						Type: &ast.VariableSizedType{
							Type: &ast.ReferenceType{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "R",
										Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
									},
								},
								StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
								EndPos:   ast.Position{Offset: 19, Line: 2, Column: 18},
							},
						},
						StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
					},
					StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 21, Line: 2, Column: 20},
				},
				Value: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
						EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
					},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result.Declarations(),
	)
}

func TestParseOptionalReference(t *testing.T) {

	t.Parallel()

	const code = `
       let x: &R? = 1
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.OptionalType{
						Type: &ast.ReferenceType{
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "R",
									Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
								},
							},
							StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
						},
						EndPos: ast.Position{Offset: 17, Line: 2, Column: 16},
					},
					StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 19, Line: 2, Column: 18},
				},
				Value: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
						EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
					},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result.Declarations(),
	)
}

func TestParseIntersectionReferenceType(t *testing.T) {

	t.Parallel()

	const code = `
       let x: &{I} = 1
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.ReferenceType{
						Type: &ast.IntersectionType{
							Types: []*ast.NominalType{
								{
									Identifier: ast.Identifier{
										Identifier: "I",
										Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
								EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
							},
						},
						StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
					},
					StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 22, Line: 2, Column: 21},
						EndPos:   ast.Position{Offset: 22, Line: 2, Column: 21},
					},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 20, Line: 2, Column: 19},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result.Declarations(),
	)
}

func TestParseOptionalIntersectionType(t *testing.T) {

	t.Parallel()

	const code = `
       let x: @{I}? = 1
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: true,
					Type: &ast.OptionalType{
						Type: &ast.IntersectionType{
							Types: []*ast.NominalType{
								{
									Identifier: ast.Identifier{
										Identifier: "I",
										Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
								EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
							},
						},
						EndPos: ast.Position{Offset: 19, Line: 2, Column: 18},
					},
					StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Value: &ast.IntegerExpression{
					PositiveLiteral: []byte("1"),
					Value:           big.NewInt(1),
					Base:            10,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
						EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
					},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 21, Line: 2, Column: 20},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result.Declarations(),
	)
}

func TestParseAuthorizedReferenceTypeWithNoEntitlements(t *testing.T) {

	t.Parallel()

	const code = `
       let x: auth &R = 1
	`
	_, errs := testParseProgram(code)

	AssertEqualWithDiff(t,
		[]error{
			&MissingStartOfAuthorizationError{
				GotToken: lexer.Token{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
						EndPos:   ast.Position{Offset: 20, Line: 2, Column: 19},
					},
					Type: lexer.TokenAmpersand,
				},
			},
		},
		errs.(Error).Errors,
	)
}

func TestParseAuthorizedReferenceTypeWithNonNominalType(t *testing.T) {

	t.Parallel()

	const code = `
       let x: auth([Int]) &R = 1
	`
	_, errs := testParseProgram(code)

	AssertEqualWithDiff(t,
		[]error{
			&NonNominalTypeError{
				Pos: ast.Position{Offset: 20, Line: 2, Column: 19},
				Type: &ast.VariableSizedType{
					Type: &ast.NominalType{
						NestedIdentifiers: []ast.Identifier{},
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Offset: 21, Line: 2, Column: 20},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
				},
			},
		},
		errs.(Error).Errors,
	)
}

func TestParseInstantiationTypeInVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
      let a: MyContract.MyStruct<Int, @R > = b
	`
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.VariableDeclaration{
				Access:     ast.AccessNotSpecified,
				IsConstant: true,
				Identifier: ast.Identifier{
					Identifier: "a",
					Pos:        ast.Position{Offset: 11, Line: 2, Column: 10},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.InstantiationType{
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "MyContract",
								Pos:        ast.Position{Offset: 14, Line: 2, Column: 13},
							},
							NestedIdentifiers: []ast.Identifier{
								{
									Identifier: "MyStruct",
									Pos:        ast.Position{Offset: 25, Line: 2, Column: 24},
								},
							},
						},
						TypeArguments: []*ast.TypeAnnotation{
							{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 34, Line: 2, Column: 33},
									},
								},
								StartPos: ast.Position{Offset: 34, Line: 2, Column: 33},
							},
							{
								IsResource: true,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "R",
										Pos:        ast.Position{Offset: 40, Line: 2, Column: 39},
									},
								},
								StartPos: ast.Position{Offset: 39, Line: 2, Column: 38},
							},
						},
						TypeArgumentsStartPos: ast.Position{Offset: 33, Line: 2, Column: 32},
						EndPos:                ast.Position{Offset: 42, Line: 2, Column: 41},
					},
					StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
					Pos:       ast.Position{Offset: 44, Line: 2, Column: 43},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "b",
						Pos:        ast.Position{Offset: 46, Line: 2, Column: 45},
					},
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

func TestParseConstantSizedSizedArrayWithTrailingUnderscoreSize(t *testing.T) {

	t.Parallel()

	_, errs := testParseDeclarations(`
	  let T:[d;0_]=0
	`)

	AssertEqualWithDiff(t,
		[]error{
			&InvalidIntegerLiteralError{
				Literal:                   "0_",
				IntegerLiteralKind:        common.IntegerLiteralKindDecimal,
				InvalidIntegerLiteralKind: InvalidNumberLiteralKindTrailingUnderscore,
				Range: ast.Range{
					StartPos: ast.Position{Line: 2, Column: 12, Offset: 13},
					EndPos:   ast.Position{Line: 2, Column: 13, Offset: 14},
				},
			},
		},
		errs,
	)
}

func TestParseParenthesizedTypes(t *testing.T) {
	t.Parallel()

	code := `let x: (Int) = 42`
	prog, errs := testParseProgram(code)
	require.Empty(t, errs)
	expected := []ast.Declaration{
		&ast.VariableDeclaration{
			Access:     ast.AccessNotSpecified,
			IsConstant: true,
			Identifier: ast.Identifier{Identifier: "x", Pos: ast.Position{Offset: 4, Line: 1, Column: 4}},
			TypeAnnotation: &ast.TypeAnnotation{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
				StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
			},
			Value: &ast.IntegerExpression{
				PositiveLiteral: []uint8("42"),
				Value:           big.NewInt(42),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Offset: 15, Line: 1, Column: 15},
					EndPos:   ast.Position{Offset: 16, Line: 1, Column: 16},
				},
			},
			Transfer: &ast.Transfer{
				Operation: 1,
				Pos:       ast.Position{Offset: 13, Line: 1, Column: 13},
			},
			StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
		},
	}

	AssertEqualWithDiff(t, expected, prog.Declarations())
}

func TestParseNestedParenthesizedTypes(t *testing.T) {
	t.Parallel()

	code := `let x: (((((((((Int))))))))) = 42`
	prog, errs := testParseProgram(code)
	require.Empty(t, errs)
	expected := []ast.Declaration{
		&ast.VariableDeclaration{
			Access:     ast.AccessNotSpecified,
			IsConstant: true,
			Identifier: ast.Identifier{Identifier: "x", Pos: ast.Position{Offset: 4, Line: 1, Column: 4}},
			TypeAnnotation: &ast.TypeAnnotation{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Offset: 16, Line: 1, Column: 16},
					},
				},
				StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
			},
			Value: &ast.IntegerExpression{
				PositiveLiteral: []uint8("42"),
				Value:           big.NewInt(42),
				Base:            10,
				Range: ast.Range{
					StartPos: ast.Position{Offset: 31, Line: 1, Column: 31},
					EndPos:   ast.Position{Offset: 32, Line: 1, Column: 32},
				},
			},
			Transfer: &ast.Transfer{
				Operation: 1,
				Pos:       ast.Position{Offset: 29, Line: 1, Column: 29},
			},
			StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
		},
	}

	AssertEqualWithDiff(t, expected, prog.Declarations())
}

func TestParseParenthesizedTypeWithInvalidType(t *testing.T) {

	t.Parallel()

	_, errs := testParseType("(.")
	AssertEqualWithDiff(t,
		[]error{
			&UnexpectedTypeStartError{
				GotToken: lexer.Token{
					Type: lexer.TokenDot,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 1, Line: 1, Column: 1},
					},
				},
			},
		},
		errs,
	)
}

func TestParseMissingEndOfParenthesizedType(t *testing.T) {

	t.Parallel()

	const code = "(Int"
	_, errs := testParseType(code)

	AssertEqualWithDiff(t,
		[]error{
			&MissingEndOfParenthesizedTypeError{
				GotToken: lexer.Token{
					Type: lexer.TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
						EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
					},
				},
			},
		},
		errs,
	)

	var missingParenErr *MissingEndOfParenthesizedTypeError
	require.ErrorAs(t, errs[0], &missingParenErr)

	fixes := missingParenErr.SuggestFixes(code)
	AssertEqualWithDiff(t,
		[]errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Insert closing parenthesis",
				TextEdits: []ast.TextEdit{
					{
						Insertion: ")",
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
					},
				},
			},
		},
		fixes,
	)

	assert.Equal(t, "(Int)", fixes[0].TextEdits[0].ApplyTo(code))
}
