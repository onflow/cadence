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

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/parser/lexer"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestParseVariableDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("var, no type annotation, copy, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("var x = 1")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					Access:     ast.AccessNotSpecified,
					IsConstant: false,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       ast.Position{Line: 1, Column: 6, Offset: 6},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("var, no type annotation, copy, one value, access(all)", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) var x = 1")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					Value: &ast.IntegerExpression{
						Value: big.NewInt(1),
						PositiveLiteral: []uint8{
							0x31,
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 21, Line: 1, Column: 21},
							EndPos:   ast.Position{Offset: 21, Line: 1, Column: 21},
						},
						Base: 10,
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       ast.Position{Offset: 19, Line: 1, Column: 19},
					},
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Offset: 17, Line: 1, Column: 17},
					},
					StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
					Access:   ast.AccessAll,
				},
			},
			result,
		)
	})

	t.Run("let, no type annotation, copy, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("let x = 1")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					Access:     ast.AccessNotSpecified,
					IsConstant: true,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationCopy,
						Pos:       ast.Position{Line: 1, Column: 6, Offset: 6},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("let, no type annotation, move, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("let x <- 1")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					Access:     ast.AccessNotSpecified,
					IsConstant: true,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IntegerExpression{
						PositiveLiteral: []byte("1"),
						Value:           big.NewInt(1),
						Base:            10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationMove,
						Pos:       ast.Position{Line: 1, Column: 6, Offset: 6},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("let, resource type annotation, move, one value", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("let r2: @R <- r")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.VariableDeclaration{
					Access:     ast.AccessNotSpecified,
					IsConstant: true,
					Identifier: ast.Identifier{
						Identifier: "r2",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					TypeAnnotation: &ast.TypeAnnotation{
						IsResource: true,
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "R",
								Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					},
					Value: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "r",
							Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationMove,
						Pos:       ast.Position{Line: 1, Column: 11, Offset: 11},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("var, no type annotation, copy, two values ", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements("var x <- y <- z")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.VariableDeclaration{
					Access:     ast.AccessNotSpecified,
					IsConstant: false,
					Identifier: ast.Identifier{
						Identifier: "x",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					Value: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "y",
							Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Transfer: &ast.Transfer{
						Operation: ast.TransferOperationMove,
						Pos:       ast.Position{Line: 1, Column: 6, Offset: 6},
					},
					SecondValue: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "z",
							Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					SecondTransfer: &ast.Transfer{
						Operation: ast.TransferOperationMove,
						Pos:       ast.Position{Line: 1, Column: 11, Offset: 11},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("with purity", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("view var x = 1")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidViewModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindVariable,
				},
			},
			errs,
		)
	})

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		const code = "static var x = 1"
		_, errs := testParseDeclarationsWithConfig(
			code,
			Config{
				StaticModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindVariable,
				},
			},
			errs,
		)

		var invalidError *InvalidStaticModifierError
		require.ErrorAs(t, errs[0], &invalidError)

		fixes := invalidError.SuggestFixes(code)
		AssertEqualWithDiff(
			t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Remove `static` modifier",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
								EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			` var x = 1`,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("static var x = 1")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		const code = "native var x = 1"
		_, errs := testParseDeclarationsWithConfig(
			code,
			Config{
				NativeModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindVariable,
				},
			},
			errs,
		)

		var invalidError *InvalidNativeModifierError
		require.ErrorAs(t, errs[0], &invalidError)

		fixes := invalidError.SuggestFixes(code)
		AssertEqualWithDiff(
			t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Remove `native` modifier",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
								EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			` var x = 1`,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("native var x = 1")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("missing transfer", func(t *testing.T) {

		t.Parallel()

		const code = "let x 1"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingTransferError{
					Pos: ast.Position{Offset: 6, Line: 1, Column: 6},
				},
			},
			errs,
		)

		var missingTransferErr *MissingTransferError
		require.ErrorAs(t, errs[0], &missingTransferErr)

		fixes := missingTransferErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert `=` (for struct)",
					TextEdits: []ast.TextEdit{
						{
							Insertion: " =",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
								EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
							},
						},
					},
				},
				{
					Message: "Insert `<-` (for resource)",
					TextEdits: []ast.TextEdit{
						{
							Insertion: " <-",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
								EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t, "let x = 1", fixes[0].TextEdits[0].ApplyTo(code))
		assert.Equal(t, "let x <- 1", fixes[1].TextEdits[0].ApplyTo(code))
	})
}

func TestParseParameterList(t *testing.T) {

	t.Parallel()

	parse := func(input string) (*ast.ParameterList, []error) {
		return Parse(
			nil,
			[]byte(input),
			func(p *parser) (*ast.ParameterList, error) {
				return parseParameterList(p, false)
			},
			Config{},
		)
	}

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("()")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ParameterList{
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("space", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(" (   )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ParameterList{
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
					EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
				},
			},
			result,
		)
	})

	t.Run("one, without argument label", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("( a : Int )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ParameterList{
				Parameters: []*ast.Parameter{
					{
						Label: "",
						Identifier: ast.Identifier{
							Identifier: "a",
							Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Line: 1, Column: 6, Offset: 6},
								},
							},
							StartPos: ast.Position{Line: 1, Column: 6, Offset: 6},
						},
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
				},
			},
			result,
		)
	})

	t.Run("one, with argument label", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("( a b : Int )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ParameterList{
				Parameters: []*ast.Parameter{
					{
						Label: "a",
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
								},
							},
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						},
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
				},
			},
			result,
		)
	})

	t.Run("two, with and without argument label", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("( a b : Int , c : Int )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.ParameterList{
				Parameters: []*ast.Parameter{
					{
						Label: "a",
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
								},
							},
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
						},
						StartPos: ast.Position{Line: 1, Column: 2, Offset: 2},
					},
					{
						Label: "",
						Identifier: ast.Identifier{
							Identifier: "c",
							Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
								},
							},
							StartPos: ast.Position{Line: 1, Column: 18, Offset: 18},
						},
						StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 22, Offset: 22},
				},
			},
			result,
		)
	})

	t.Run("two, with and without argument label, missing comma", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("( a b : Int   c : Int )")
		AssertEqualWithDiff(t,
			[]error{
				&MissingCommaInParameterListError{
					Pos: ast.Position{Offset: 14, Line: 1, Column: 14},
				},
			},
			errs,
		)
	})
}

func TestParseFunctionDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("without return type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("fun foo () { }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					Purity: ast.FunctionPurityUnspecified,
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 11, Offset: 11},
								EndPos:   ast.Position{Line: 1, Column: 13, Offset: 13},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("without return type, access(all)", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("access(all) fun foo () { }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 20, Line: 1, Column: 20},
							EndPos:   ast.Position{Offset: 21, Line: 1, Column: 21},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Offset: 23, Line: 1, Column: 23},
								EndPos:   ast.Position{Offset: 25, Line: 1, Column: 25},
							},
						},
					},
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Offset: 16, Line: 1, Column: 16},
					},
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					Access:   ast.AccessAll,
				},
			},
			result,
		)
	})

	t.Run("missing parameter list start", func(t *testing.T) {

		t.Parallel()

		const code = "fun foo x: Int) {}"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingStartOfParameterListError{
					GotToken: lexer.Token{
						Type: lexer.TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
							EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
						},
					},
				},
			},
			errs,
		)

		var missingStartErr *MissingStartOfParameterListError
		require.ErrorAs(t, errs[0], &missingStartErr)

		fixes := missingStartErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert opening parenthesis",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "(",
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

		assert.Equal(t,
			`fun foo( x: Int) {}`,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("unexpected token in parameter list", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("fun foo(-) {}")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenInParameterListError{
					GotToken: lexer.Token{
						Type: lexer.TokenMinus,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
							EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
						},
					},
				},
			},
			errs,
		)
	})

	t.Run("missing parameter list end", func(t *testing.T) {

		t.Parallel()

		const code = "fun foo("
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingParenInParameterListError{
					Pos: ast.Position{Offset: 8, Line: 1, Column: 8},
				},
			},
			errs,
		)

		var missingClosingParen *MissingClosingParenInParameterListError
		require.ErrorAs(t, errs[0], &missingClosingParen)

		assert.Equal(t,
			&MissingClosingParenInParameterListError{
				Pos: ast.Position{Offset: 8, Line: 1, Column: 8},
			},
			missingClosingParen,
		)

		fixes := missingClosingParen.SuggestFixes(code)

		require.Equal(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing parenthesis",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ")",
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

		const expected = "fun foo()"
		assert.Equal(t,
			expected,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("invalid parameter list continuation", func(t *testing.T) {

		t.Parallel()

		const code = "fun foo(a: Int -) {}"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&ExpectedCommaOrEndOfParameterListError{
					GotToken: lexer.Token{
						Type: lexer.TokenMinus,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 15, Line: 1, Column: 15},
							EndPos:   ast.Position{Offset: 15, Line: 1, Column: 15},
						},
					},
				},
			},
			errs,
		)

		var expectedErr *ExpectedCommaOrEndOfParameterListError
		require.ErrorAs(t, errs[0], &expectedErr)

		fixes := expectedErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert comma",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ",",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
								EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"fun foo(a: Int, -) {}",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("missing colon after parameter name", func(t *testing.T) {

		t.Parallel()

		const code = "fun foo(a Int) {}"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingColonAfterParameterNameError{
					GotToken: lexer.Token{
						Type: lexer.TokenParenClose,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 1, Column: 13},
							EndPos:   ast.Position{Offset: 13, Line: 1, Column: 13},
						},
					},
				},
			},
			errs,
		)

		var missingColonErr *MissingColonAfterParameterNameError
		require.ErrorAs(t, errs[0], &missingColonErr)

		fixes := missingColonErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert colon",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ":",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 13, Line: 1, Column: 13},
								EndPos:   ast.Position{Offset: 13, Line: 1, Column: 13},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"fun foo(a Int:) {}",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("invalid type parameter list continuation", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			"fun foo  < A ,, > () {}",
			Config{
				TypeParametersEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenInTypeParameterListError{
					GotToken: lexer.Token{
						Type: lexer.TokenComma,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
							EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
						},
					},
				},
			},
			errs,
		)
	})
	t.Run("with return type", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("fun foo (): X { }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "X",
								Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
								EndPos:   ast.Position{Line: 1, Column: 16, Offset: 16},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("without return type, with pre and post conditions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseStatements(`
          fun foo () {
              pre {
                 true : "test"
                 2 > 1 : "foo"
              }

              post {
                 false
              }

              bar()
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 2, Column: 14, Offset: 15},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 18, Offset: 19},
							EndPos:   ast.Position{Line: 2, Column: 19, Offset: 20},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						PreConditions: &ast.Conditions{
							Range: ast.Range{
								StartPos: ast.Position{Offset: 38, Line: 3, Column: 14},
								EndPos:   ast.Position{Offset: 120, Line: 6, Column: 14},
							},
							Conditions: []ast.Condition{
								&ast.TestCondition{
									Test: &ast.BoolExpression{
										Value: true,
										Range: ast.Range{
											StartPos: ast.Position{Line: 4, Column: 17, Offset: 61},
											EndPos:   ast.Position{Line: 4, Column: 20, Offset: 64},
										},
									},
									Message: &ast.StringExpression{
										Value: "test",
										Range: ast.Range{
											StartPos: ast.Position{Line: 4, Column: 24, Offset: 68},
											EndPos:   ast.Position{Line: 4, Column: 29, Offset: 73},
										},
									},
								},
								&ast.TestCondition{
									Test: &ast.BinaryExpression{
										Operation: ast.OperationGreater,
										Left: &ast.IntegerExpression{
											PositiveLiteral: []byte("2"),
											Value:           big.NewInt(2),
											Base:            10,
											Range: ast.Range{
												StartPos: ast.Position{Line: 5, Column: 17, Offset: 92},
												EndPos:   ast.Position{Line: 5, Column: 17, Offset: 92},
											},
										},
										Right: &ast.IntegerExpression{
											PositiveLiteral: []byte("1"),
											Value:           big.NewInt(1),
											Base:            10,
											Range: ast.Range{
												StartPos: ast.Position{Line: 5, Column: 21, Offset: 96},
												EndPos:   ast.Position{Line: 5, Column: 21, Offset: 96},
											},
										},
									},
									Message: &ast.StringExpression{
										Value: "foo",
										Range: ast.Range{
											StartPos: ast.Position{Line: 5, Column: 25, Offset: 100},
											EndPos:   ast.Position{Line: 5, Column: 29, Offset: 104},
										},
									},
								},
							},
						},
						PostConditions: &ast.Conditions{
							Range: ast.Range{
								StartPos: ast.Position{Offset: 137, Line: 8, Column: 14},
								EndPos:   ast.Position{Offset: 181, Line: 10, Column: 14},
							},
							Conditions: []ast.Condition{
								&ast.TestCondition{
									Test: &ast.BoolExpression{
										Value: false,
										Range: ast.Range{
											StartPos: ast.Position{Line: 9, Column: 17, Offset: 161},
											EndPos:   ast.Position{Line: 9, Column: 21, Offset: 165},
										},
									},
									Message: nil,
								},
							},
						},
						Block: &ast.Block{
							Statements: []ast.Statement{
								&ast.ExpressionStatement{
									Expression: &ast.InvocationExpression{
										InvokedExpression: &ast.IdentifierExpression{
											Identifier: ast.Identifier{
												Identifier: "bar",
												Pos:        ast.Position{Line: 12, Column: 14, Offset: 198},
											},
										},
										ArgumentsStartPos: ast.Position{Line: 12, Column: 17, Offset: 201},
										EndPos:            ast.Position{Line: 12, Column: 18, Offset: 202},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Line: 2, Column: 21, Offset: 22},
								EndPos:   ast.Position{Line: 13, Column: 10, Offset: 214},
							},
						},
					},
					StartPos: ast.Position{Line: 2, Column: 10, Offset: 11},
				},
			},
			result,
		)
	})

	t.Run("with docstring, single line comment", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("/// Test\nfun foo() {}")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 2, Column: 4, Offset: 13},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 7, Offset: 16},
							EndPos:   ast.Position{Line: 2, Column: 8, Offset: 17},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 2, Column: 10, Offset: 19},
								EndPos:   ast.Position{Line: 2, Column: 11, Offset: 20},
							},
						},
					},
					DocString: " Test",
					StartPos:  ast.Position{Line: 2, Column: 0, Offset: 9},
				},
			},
			result,
		)
	})

	t.Run("with docstring, two line comments", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("\n  /// First line\n  \n/// Second line\n\n\nfun foo() {}")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 7, Column: 4, Offset: 43},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 7, Column: 7, Offset: 46},
							EndPos:   ast.Position{Line: 7, Column: 8, Offset: 47},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 7, Column: 10, Offset: 49},
								EndPos:   ast.Position{Line: 7, Column: 11, Offset: 50},
							},
						},
					},
					DocString: " First line\n Second line",
					StartPos:  ast.Position{Line: 7, Column: 0, Offset: 39},
				},
			},
			result,
		)
	})

	t.Run("with docstring, block comment", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("\n    /** Cool dogs.\n\n Cool cats!! */\n\n\nfun foo() {}")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 7, Column: 4, Offset: 43},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 7, Column: 7, Offset: 46},
							EndPos:   ast.Position{Line: 7, Column: 8, Offset: 47},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 7, Column: 10, Offset: 49},
								EndPos:   ast.Position{Line: 7, Column: 11, Offset: 50},
							},
						},
					},
					DocString: " Cool dogs.\n\n Cool cats!! ",
					StartPos:  ast.Position{Line: 7, Column: 0, Offset: 39},
				},
			},
			result,
		)
	})

	t.Run("without space after return type", func(t *testing.T) {

		// A brace after the return type is ambiguous:
		// It could be the start of an intersection type.
		// However, if there is space after the brace, which is most common
		// in function declarations, we consider it not an intersection type

		t.Parallel()

		result, errs := testParseDeclarations("fun main(): Int{ return 1 }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "main",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
							EndPos:   ast.Position{Line: 1, Column: 9, Offset: 9},
						},
					},
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "Int",
								Pos:        ast.Position{Line: 1, Column: 12, Offset: 12},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 12, Offset: 12},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Statements: []ast.Statement{
								&ast.ReturnStatement{
									Expression: &ast.IntegerExpression{
										PositiveLiteral: []byte("1"),
										Value:           big.NewInt(1),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Line: 1, Column: 24, Offset: 24},
											EndPos:   ast.Position{Line: 1, Column: 24, Offset: 24},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
										EndPos:   ast.Position{Line: 1, Column: 24, Offset: 24},
									},
								},
							},
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 15, Offset: 15},
								EndPos:   ast.Position{Line: 1, Column: 26, Offset: 26},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("view function", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("view fun foo (): X { }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 9, Offset: 9},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 13, Offset: 13},
							EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					Purity: ast.FunctionPurityView,
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						Type: &ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "X",
								Pos:        ast.Position{Line: 1, Column: 17, Offset: 17},
							},
						},
						StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 19, Offset: 19},
								EndPos:   ast.Position{Line: 1, Column: 21, Offset: 21},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarationsWithConfig(
			"native fun foo() {}",
			Config{
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Flags:  ast.FunctionDeclarationFlagsIsNative,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 11, Offset: 11},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
								EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("double purity annot", func(t *testing.T) {

		t.Parallel()

		const code = "view view fun foo (): X { }"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&DuplicateViewModifierError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
						EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
			},
			errs,
		)

		var duplicateViewError *DuplicateViewModifierError
		require.ErrorAs(t, errs[0], &duplicateViewError)

		fixes := duplicateViewError.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Remove duplicate `view` modifier",
					TextEdits: []ast.TextEdit{
						{
							Replacement: "",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
								EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
							},
						},
					},
				},
			},
			fixes,
		)

		const expected = "view  fun foo (): X { }"
		assert.Equal(t,
			expected,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("native fun foo() {}")

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("static", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarationsWithConfig(
			"static fun foo() {}",
			Config{
				StaticModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Flags:  ast.FunctionDeclarationFlagsIsStatic,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 11, Offset: 11},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 14, Offset: 14},
							EndPos:   ast.Position{Line: 1, Column: 15, Offset: 15},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 17, Offset: 17},
								EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("static fun foo() {}")

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("static native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarationsWithConfig(
			"static native fun foo() {}",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Flags:  ast.FunctionDeclarationFlagsIsStatic | ast.FunctionDeclarationFlagsIsNative,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 21, Offset: 21},
							EndPos:   ast.Position{Line: 1, Column: 22, Offset: 22},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 24, Offset: 24},
								EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("static native fun foo() {}")

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("native static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			"native static fun foo() {}",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid `static` modifier after `native` modifier",
					Pos:     ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)
	})

	t.Run("native static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("native static fun foo() {}")

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("access(all) static native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarationsWithConfig(
			"access(all) static native fun foo() {}",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Purity:            0,
					TypeParameterList: (*ast.TypeParameterList)(nil),
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 33, Line: 1, Column: 33},
							EndPos:   ast.Position{Offset: 34, Line: 1, Column: 34},
						},
					},
					ReturnTypeAnnotation: (*ast.TypeAnnotation)(nil),
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Offset: 36, Line: 1, Column: 36},
								EndPos:   ast.Position{Offset: 37, Line: 1, Column: 37},
							},
						},
						PreConditions:  (*ast.Conditions)(nil),
						PostConditions: (*ast.Conditions)(nil),
					},
					DocString: "",
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Offset: 30, Line: 1, Column: 30},
					},
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					Access:   ast.AccessAll,
					Flags:    ast.FunctionDeclarationFlagsIsStatic | ast.FunctionDeclarationFlagsIsNative,
				},
			},
			result,
		)
	})

	t.Run("access(all) static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("access(all) static native fun foo() {}")

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
							EndPos:   ast.Position{Offset: 17, Line: 1, Column: 17},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("with empty type parameters, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarationsWithConfig(
			"fun foo  < > () {}",
			Config{
				TypeParametersEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					TypeParameterList: &ast.TypeParameterList{
						TypeParameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
						},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 13, Offset: 13},
							EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
								EndPos:   ast.Position{Line: 1, Column: 17, Offset: 17},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("with type parameters, single type parameter, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarationsWithConfig(
			"fun foo  < A  > () {}",
			Config{
				TypeParametersEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					TypeParameterList: &ast.TypeParameterList{
						TypeParameters: []*ast.TypeParameter{
							{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos:        ast.Position{Offset: 11, Line: 1, Column: 11},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 14, Offset: 14},
						},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
							EndPos:   ast.Position{Line: 1, Column: 17, Offset: 17},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 19, Offset: 19},
								EndPos:   ast.Position{Line: 1, Column: 20, Offset: 20},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("with type parameters, multiple parameters, type bound, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarationsWithConfig(
			"fun foo  < A  , B : C > () {}",
			Config{
				TypeParametersEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
					},
					TypeParameterList: &ast.TypeParameterList{
						TypeParameters: []*ast.TypeParameter{
							{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos:        ast.Position{Offset: 11, Line: 1, Column: 11},
								},
							},
							{
								Identifier: ast.Identifier{
									Identifier: "B",
									Pos:        ast.Position{Offset: 16, Line: 1, Column: 16},
								},
								TypeBound: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "C",
											Pos:        ast.Position{Offset: 20, Line: 1, Column: 20},
										},
									},
									StartPos: ast.Position{Offset: 20, Line: 1, Column: 20},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 9, Offset: 9},
							EndPos:   ast.Position{Line: 1, Column: 22, Offset: 22},
						},
					},
					ParameterList: &ast.ParameterList{
						Parameters: nil,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 24, Offset: 24},
							EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
						},
					},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: ast.Position{Line: 1, Column: 27, Offset: 27},
								EndPos:   ast.Position{Line: 1, Column: 28, Offset: 28},
							},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
				},
			},
			result,
		)
	})

	t.Run("with type parameters, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("fun foo<A>() {}")

		AssertEqualWithDiff(t,
			[]error{
				&MissingStartOfParameterListError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
							EndPos:   ast.Position{Offset: 7, Line: 1, Column: 7},
						},
						Type: lexer.TokenLess,
					},
				},
			},
			errs,
		)
	})

	t.Run("missing type parameter list end, enabled", func(t *testing.T) {

		t.Parallel()

		const code = "fun foo  < "
		_, errs := testParseDeclarationsWithConfig(
			code,
			Config{
				TypeParametersEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingGreaterInTypeParameterListError{
					Pos: ast.Position{Offset: 11, Line: 1, Column: 11},
				},
			},
			errs,
		)

		var missingClosingGreater *MissingClosingGreaterInTypeParameterListError
		require.ErrorAs(t, errs[0], &missingClosingGreater)

		fixes := missingClosingGreater.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing angle bracket",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ">",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 10, Line: 1, Column: 10},
								EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			`fun foo  <> `,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("missing type parameter list separator, enabled", func(t *testing.T) {

		t.Parallel()

		const code = "fun foo  < A B > () { } "
		_, errs := testParseDeclarationsWithConfig(
			code,
			Config{
				TypeParametersEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&MissingCommaInTypeParameterListError{
					Pos: ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)

		var missingCommaErr *MissingCommaInTypeParameterListError
		require.ErrorAs(t, errs[0], &missingCommaErr)

		fixes := missingCommaErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert comma",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ",",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
								EndPos:   ast.Position{Offset: 12, Line: 1, Column: 12},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			`fun foo  < A, B > () { } `,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("invalid type parameter list separator", func(t *testing.T) {

		t.Parallel()

		const code = "fun foo  < A - > () { } "
		_, errs := testParseDeclarationsWithConfig(
			code,
			Config{
				TypeParametersEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&ExpectedCommaOrEndOfTypeParameterListError{
					GotToken: lexer.Token{
						Type: lexer.TokenMinus,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 1, Column: 13},
							EndPos:   ast.Position{Offset: 13, Line: 1, Column: 13},
						},
					},
				},
			},
			errs,
		)

		var expectedErr *ExpectedCommaOrEndOfTypeParameterListError
		require.ErrorAs(t, errs[0], &expectedErr)

		fixes := expectedErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert comma",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ",",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
								EndPos:   ast.Position{Offset: 12, Line: 1, Column: 12},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			`fun foo  < A, - > () { } `,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("missing closing > in type arguments", func(t *testing.T) {

		t.Parallel()

		const code = "let x: Foo<Bar"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingClosingGreaterInTypeArgumentsError{
					Pos: ast.Position{Offset: 14, Line: 1, Column: 14},
				},
				&MissingTransferError{
					Pos: ast.Position{Offset: 14, Line: 1, Column: 14},
				},
				UnexpectedEOFError{
					Pos: ast.Position{Offset: 14, Line: 1, Column: 14},
				},
			},
			errs,
		)

		var missingClosingGreater *MissingClosingGreaterInTypeArgumentsError
		require.ErrorAs(t, errs[0], &missingClosingGreater)

		fixes := missingClosingGreater.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing angle bracket",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ">",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
								EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			`let x: Foo<Bar>`,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})
}

func TestParseAccess(t *testing.T) {

	t.Parallel()

	parse := func(input string) (ast.Access, []error) {
		return Parse(
			nil,
			[]byte(input),
			func(p *parser) (ast.Access, error) {
				access, _, err := parseAccess(p)
				return access, err
			},
			Config{},
		)
	}

	t.Run("access(all)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( all )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			ast.AccessAll,
			result,
		)
	})

	t.Run("access(account)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( account )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			ast.AccessAccount,
			result,
		)
	})

	t.Run("access(contract)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( contract )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			ast.AccessContract,
			result,
		)
	})

	t.Run("access(self)", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( self )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			ast.AccessSelf,
			result,
		)
	})

	t.Run("access, missing keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( ")
		AssertEqualWithDiff(t,
			[]error{
				&MissingAccessKeywordError{
					GotToken: lexer.Token{
						Type: lexer.TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 9, Line: 1, Column: 9},
							EndPos:   ast.Position{Offset: 9, Line: 1, Column: 9},
						},
					},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, missing keyword", func(t *testing.T) {

		t.Parallel()

		const code = "access("
		result, errs := parse(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingAccessKeywordError{
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

		AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)

		var missingKeywordErr *MissingAccessKeywordError
		require.ErrorAs(t, errs[0], &missingKeywordErr)

		fixes := missingKeywordErr.SuggestFixes(code)
		keywords := []string{"all", "account", "contract", "self"}
		require.Len(t, fixes, len(keywords))

		for i, keyword := range keywords {
			AssertEqualWithDiff(t,
				errors.SuggestedFix[ast.TextEdit]{
					Message: fmt.Sprintf("Insert `%s`", keyword),
					TextEdits: []ast.TextEdit{
						{
							Insertion: keyword,
							Range: ast.Range{
								StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
								EndPos:   ast.Position{Offset: 7, Line: 1, Column: 7},
							},
						},
					},
				},
				fixes[i],
			)

			assert.Equal(t,
				fmt.Sprintf("access(%s", keyword),
				fixes[i].TextEdits[0].ApplyTo(code),
			)
		}
	})

	t.Run("access, missing opening paren before identifier", func(t *testing.T) {

		t.Parallel()

		const code = "access self"
		result, errs := parse(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingAccessOpeningParenError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
							EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
						},
						Type: lexer.TokenIdentifier,
					},
				},
				&MissingAccessClosingParenError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 11, Line: 1, Column: 11},
							EndPos:   ast.Position{Offset: 11, Line: 1, Column: 11},
						},
						Type: lexer.TokenEOF,
					},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessSelf,
			result,
		)

		var missingParenErr *MissingAccessOpeningParenError
		require.ErrorAs(t, errs[0], &missingParenErr)

		fixes := missingParenErr.SuggestFixes(code)
		AssertEqualWithDiff(
			t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Enclose in parentheses",
					TextEdits: []ast.TextEdit{
						{
							Replacement: "(self)",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
								EndPos:   ast.Position{Offset: 10, Line: 1, Column: 10},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"access (self)",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("access, missing opening paren at end", func(t *testing.T) {

		t.Parallel()

		const code = "access "
		_, errs := parse(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingAccessOpeningParenError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
							EndPos:   ast.Position{Offset: 7, Line: 1, Column: 7},
						},
						Type: lexer.TokenEOF,
					},
				},
				&MissingAccessKeywordError{
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

		var missingParenErr *MissingAccessOpeningParenError
		require.ErrorAs(t, errs[0], &missingParenErr)

		fixes := missingParenErr.SuggestFixes(code)
		AssertEqualWithDiff(
			t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert opening parenthesis",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "(",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 6, Line: 1, Column: 6},
								EndPos:   ast.Position{Offset: 6, Line: 1, Column: 6},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"access( ",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("access, missing closing paren", func(t *testing.T) {

		t.Parallel()

		const code = "access ( self "
		result, errs := parse(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingAccessClosingParenError{
					GotToken: lexer.Token{
						Type: lexer.TokenEOF,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
							EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
						},
					},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessSelf,
			result,
		)

		var missingParenErr *MissingAccessClosingParenError
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
								StartPos: ast.Position{Offset: 13, Line: 1, Column: 13},
								EndPos:   ast.Position{Offset: 13, Line: 1, Column: 13},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"access ( self) ",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("access, single entitlement", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			ast.EntitlementAccess{
				EntitlementSet: &ast.ConjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("access, multiple conjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo , bar )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			ast.EntitlementAccess{
				EntitlementSet: &ast.ConjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Offset: 15, Line: 1, Column: 15},
							},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("access, multiple disjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo | bar )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			ast.EntitlementAccess{
				EntitlementSet: &ast.DisjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Offset: 15, Line: 1, Column: 15},
							},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("access, mixed disjunctive and conjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo | bar , baz )")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenInsteadOfSeparatorError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 19, Line: 1, Column: 19},
							EndPos:   ast.Position{Offset: 19, Line: 1, Column: 19},
						},
						Type: lexer.TokenComma,
					},
					ExpectedSeparator: lexer.TokenVerticalBar,
					ExpectedEndToken:  lexer.TokenParenClose,
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, mixed conjunctive and disjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo , bar | baz )")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenInsteadOfSeparatorError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 19, Line: 1, Column: 19},
							EndPos:   ast.Position{Offset: 19, Line: 1, Column: 19},
						},
						Type: lexer.TokenVerticalBar,
					},
					ExpectedSeparator: lexer.TokenComma,
					ExpectedEndToken:  lexer.TokenParenClose,
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, conjunctive entitlements list starting with keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( self , bar )")
		AssertEqualWithDiff(t,
			[]error{
				&MissingAccessClosingParenError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
							EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
						},
						Type: lexer.TokenComma,
					},
				},
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
							EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
						},
						Type: lexer.TokenComma,
					},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessSelf,
			result,
		)
	})

	t.Run("access, disjunctive entitlements list starting with keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( self | bar )")
		AssertEqualWithDiff(t,
			[]error{
				&MissingAccessClosingParenError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
							EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
						},
						Type: lexer.TokenVerticalBar,
					},
				},
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
							EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
						},
						Type: lexer.TokenVerticalBar,
					},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessSelf,
			result,
		)
	})

	t.Run("access, conjunctive entitlements list ending with keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo , self )")
		AssertEqualWithDiff(t,
			[]error{
				&AccessKeywordEntitlementNameError{
					Keyword: "self",
					Range: ast.Range{
						StartPos: ast.Position{Offset: 15, Line: 1, Column: 15},
						EndPos:   ast.Position{Offset: 18, Line: 1, Column: 18},
					},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.EntitlementAccess{
				EntitlementSet: &ast.ConjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "self",
								Pos:        ast.Position{Offset: 15, Line: 1, Column: 15},
							},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("access, conjunctive entitlements list with trailing comma", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo , bar , )")
		AssertEqualWithDiff(t,
			[]error{
				&MissingTypeAfterSeparatorError{
					Pos:       ast.Position{Offset: 21, Line: 1, Column: 21},
					Separator: lexer.TokenComma,
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.EntitlementAccess{
				EntitlementSet: &ast.ConjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Offset: 15, Line: 1, Column: 15},
							},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("access, conjunctive entitlements list with leading comma", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo , , )")
		AssertEqualWithDiff(t,
			[]error{
				&ExpectedTypeInsteadSeparatorError{
					Pos:       ast.Position{Offset: 15, Line: 1, Column: 15},
					Separator: lexer.TokenComma,
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, conjunctive entitlements list with missing separator", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo, bar baz )")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenInsteadOfSeparatorError{
					GotToken: lexer.Token{
						Type: lexer.TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 18, Line: 1, Column: 18},
							EndPos:   ast.Position{Offset: 20, Line: 1, Column: 20},
						},
					},
					ExpectedSeparator: lexer.TokenComma,
					ExpectedEndToken:  lexer.TokenParenClose,
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

	t.Run("access, disjunctive entitlements list ending with keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( foo | self )")
		AssertEqualWithDiff(t,
			[]error{
				&AccessKeywordEntitlementNameError{
					Keyword: "self",
					Range: ast.Range{
						StartPos: ast.Position{Offset: 15, Line: 1, Column: 15},
						EndPos:   ast.Position{Offset: 18, Line: 1, Column: 18},
					},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.EntitlementAccess{
				EntitlementSet: &ast.DisjunctiveEntitlementSet{
					Elements: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 9, Line: 1, Column: 9},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "self",
								Pos:        ast.Position{Offset: 15, Line: 1, Column: 15},
							},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("access, multiple entitlements no separator", func(t *testing.T) {

		t.Parallel()

		const code = "access ( foo bar )"
		_, errs := parse(code)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidEntitlementSeparatorError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 1, Column: 13},
							EndPos:   ast.Position{Offset: 15, Line: 1, Column: 15},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)

		var invalidSepErr *InvalidEntitlementSeparatorError
		require.ErrorAs(t, errs[0], &invalidSepErr)

		fixes := invalidSepErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert comma (conjunction)",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ",",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
								EndPos:   ast.Position{Offset: 12, Line: 1, Column: 12},
							},
						},
					},
				},
				{
					Message: "Insert vertical bar (disjunction)",
					TextEdits: []ast.TextEdit{
						{
							Insertion: " |",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
								EndPos:   ast.Position{Offset: 12, Line: 1, Column: 12},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t, "access ( foo, bar )", fixes[0].TextEdits[0].ApplyTo(code))
		assert.Equal(t, "access ( foo | bar )", fixes[1].TextEdits[0].ApplyTo(code))
	})

	t.Run("access, invalid separator", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("access ( foo & bar )")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidEntitlementSeparatorError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 1, Column: 13},
							EndPos:   ast.Position{Offset: 13, Line: 1, Column: 13},
						},
						Type: lexer.TokenAmpersand,
					},
				},
				// & bar is parsed as a reference type
				&NonNominalTypeError{
					Pos: ast.Position{Offset: 13, Line: 1, Column: 13},
					Type: &ast.ReferenceType{
						Type: &ast.NominalType{
							NestedIdentifiers: []ast.Identifier{},
							Identifier: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Offset: 15, Line: 1, Column: 15},
							},
						},
						StartPos: ast.Position{Offset: 13, Line: 1, Column: 13},
					},
				},
			},
			errs,
		)
	})

	t.Run("access, entitlement map", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( mapping foo )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.MappedAccess{
				EntitlementMap: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "foo",
						Pos:        ast.Position{Offset: 17, Line: 1, Column: 17},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 1, Column: 9},
			},
			result,
		)
	})

	t.Run("access, entitlement map no name", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("access ( mapping )")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 1, Column: 17},
							EndPos:   ast.Position{Offset: 17, Line: 1, Column: 17},
						},
						Type: lexer.TokenParenClose,
					},
				},
			},
			errs,
		)

		AssertEqualWithDiff(t,
			ast.AccessNotSpecified,
			result,
		)
	})

}

func TestParseImportDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("no identifiers, missing location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import`)
		AssertEqualWithDiff(t,
			[]error{
				&MissingImportLocationError{
					Pos: ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("no identifiers, string location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import "foo"`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports:     nil,
					Location:    common.StringLocation("foo"),
					LocationPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 12, Offset: 12},
					},
				},
			},
			result,
		)
	})

	t.Run("no identifiers, address location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import 0x42`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports: nil,
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 11, Offset: 11},
					},
				},
			},
			result,
		)
	})

	t.Run("no identifiers, address location, address too long", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import 0x10000000000000001`)
		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "address too large",
					Pos:     ast.Position{Offset: 8, Line: 1, Column: 8},
				},
			},
			errs,
		)

		expected := []ast.Declaration{
			&ast.ImportDeclaration{
				Imports: nil,
				Location: common.AddressLocation{
					Address: common.MustBytesToAddress([]byte{0x0}),
				},
				LocationPos: ast.Position{Line: 1, Column: 8, Offset: 8},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
					EndPos:   ast.Position{Line: 1, Column: 26, Offset: 26},
				},
			},
		}

		AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("no identifiers, integer location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import 1`)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidImportLocationError{
					GotToken: lexer.Token{
						Type: lexer.TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
							EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
						},
					},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		AssertEqualWithDiff(t,
			expected,
			result,
		)

	})

	t.Run("one identifier, string location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo from "bar"`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports: []ast.Import{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
							},
						},
					},
					Location:    common.StringLocation("bar"),
					LocationPos: ast.Position{Line: 1, Column: 17, Offset: 17},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 21, Offset: 21},
					},
				},
			},
			result,
		)
	})

	t.Run("one identifier, string location, missing from keyword", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo "bar"`)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidImportContinuationError{
					GotToken: lexer.Token{
						Type: lexer.TokenString,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
							EndPos:   ast.Position{Offset: 16, Line: 1, Column: 16},
						},
					},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("three identifiers, address location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo , bar , baz from 0x42`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports: []ast.Import{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Line: 1, Column: 14, Offset: 14},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "baz",
								Pos:        ast.Position{Line: 1, Column: 20, Offset: 20},
							},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 1, Column: 29, Offset: 29},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 32, Offset: 32},
					},
				},
			},
			result,
		)
	})

	t.Run("two identifiers, address location, extra comma", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo , bar , from 0x42`)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidFromKeywordAsIdentifierError{
					GotToken: lexer.Token{
						Type: lexer.TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 20, Line: 1, Column: 20},
							EndPos:   ast.Position{Offset: 23, Line: 1, Column: 23},
						},
					},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("two identifiers, address location, repeated commas", func(t *testing.T) {
		t.Parallel()

		result, errs := testParseDeclarations(`import foo, , bar from 0xaaaa`)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidTokenInImportListError{
					GotToken: lexer.Token{
						Type: lexer.TokenComma,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
							EndPos:   ast.Position{Offset: 12, Line: 1, Column: 12},
						},
					},
				},
			},
			errs,
		)
		var expected []ast.Declaration

		AssertEqualWithDiff(t, expected, result)
	})

	t.Run("no identifiers, identifier location", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(` import foo`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports:     nil,
					Location:    common.IdentifierLocation("foo"),
					LocationPos: ast.Position{Line: 1, Column: 8, Offset: 8},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("unexpected token as identifier", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations(`import foo, bar, baz, @ from 0x42`)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidImportContinuationError{
					GotToken: lexer.Token{
						Type: lexer.TokenAt,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 22, Line: 1, Column: 22},
							EndPos:   ast.Position{Offset: 22, Line: 1, Column: 22},
						},
					},
				},
			},
			errs,
		)
	})

	t.Run("one identifier, missing second identifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`import foo , `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedEOFInImportListError{
					Pos: ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)
	})

	t.Run("from keyword as second identifier", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          import foo, from from 0x42
          import foo, from, bar from 0x42
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports: []ast.Import{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 2, Column: 17, Offset: 18},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "from",
								Pos:        ast.Position{Line: 2, Column: 22, Offset: 23},
							},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 2, Column: 32, Offset: 33},
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 10, Offset: 11},
						EndPos:   ast.Position{Line: 2, Column: 35, Offset: 36},
					},
				},
				&ast.ImportDeclaration{
					Imports: []ast.Import{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 3, Column: 17, Offset: 55},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "from",
								Pos:        ast.Position{Line: 3, Column: 22, Offset: 60},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Line: 3, Column: 28, Offset: 66},
							},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 3, Column: 37, Offset: 75},
					Range: ast.Range{
						StartPos: ast.Position{Line: 3, Column: 10, Offset: 48},
						EndPos:   ast.Position{Line: 3, Column: 40, Offset: 78},
					},
				},
			},
			result,
		)
	})

	t.Run("single import alias", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
			import foo as bar, lorem from 0x42
		`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports: []ast.Import{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 2, Column: 10, Offset: 11},
							},
							Alias: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Line: 2, Column: 17, Offset: 18},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "lorem",
								Pos:        ast.Position{Line: 2, Column: 22, Offset: 23},
							},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 2, Column: 33, Offset: 34},
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 3, Offset: 4},
						EndPos:   ast.Position{Line: 2, Column: 36, Offset: 37},
					},
				},
			},
			result,
		)
	})

	t.Run("multiple import alias", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
			import foo as bar, lorem as ipsum from 0x42
		`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports: []ast.Import{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 2, Column: 10, Offset: 11},
							},
							Alias: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Line: 2, Column: 17, Offset: 18},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "lorem",
								Pos:        ast.Position{Line: 2, Column: 22, Offset: 23},
							},
							Alias: ast.Identifier{
								Identifier: "ipsum",
								Pos:        ast.Position{Line: 2, Column: 31, Offset: 32},
							},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 2, Column: 42, Offset: 43},
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 3, Offset: 4},
						EndPos:   ast.Position{Line: 2, Column: 45, Offset: 46},
					},
				},
			},
			result,
		)
	})

	t.Run("combination import aliases", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
			import foo as bar, test as from, from from 0x42
		`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports: []ast.Import{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 2, Column: 10, Offset: 11},
							},
							Alias: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Line: 2, Column: 17, Offset: 18},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "test",
								Pos:        ast.Position{Line: 2, Column: 22, Offset: 23},
							},
							Alias: ast.Identifier{
								Identifier: "from",
								Pos:        ast.Position{Line: 2, Column: 30, Offset: 31},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "from",
								Pos:        ast.Position{Line: 2, Column: 36, Offset: 37},
							},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 2, Column: 46, Offset: 47},
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 3, Offset: 4},
						EndPos:   ast.Position{Line: 2, Column: 49, Offset: 50},
					},
				},
			},
			result,
		)
	})

	t.Run("alias same imported function", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
			import foo as bar from 0x42
			import foo as cab from 0x42
		`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.ImportDeclaration{
					Imports: []ast.Import{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 2, Column: 10, Offset: 11},
							},
							Alias: ast.Identifier{
								Identifier: "bar",
								Pos:        ast.Position{Line: 2, Column: 17, Offset: 18},
							},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 2, Column: 26, Offset: 27},
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 3, Offset: 4},
						EndPos:   ast.Position{Line: 2, Column: 29, Offset: 30},
					},
				},
				&ast.ImportDeclaration{
					Imports: []ast.Import{
						{
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Line: 3, Column: 10, Offset: 42},
							},
							Alias: ast.Identifier{
								Identifier: "cab",
								Pos:        ast.Position{Line: 3, Column: 17, Offset: 49},
							},
						},
					},
					Location: common.AddressLocation{
						Address: common.MustBytesToAddress([]byte{0x42}),
					},
					LocationPos: ast.Position{Line: 3, Column: 26, Offset: 58},
					Range: ast.Range{
						StartPos: ast.Position{Line: 3, Column: 3, Offset: 35},
						EndPos:   ast.Position{Line: 3, Column: 29, Offset: 61},
					},
				},
			},
			result,
		)
	})

	t.Run("invalid, non identifier alias", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
			import foo as 1 from 0x42
		`)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidTokenInImportAliasError{
					GotToken: lexer.Token{
						Type: lexer.TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
					},
				},
				&InvalidImportContinuationError{
					GotToken: lexer.Token{
						Type: lexer.TokenDecimalIntegerLiteral,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
					},
				},
			},
			errs,
		)
	})
}

func TestParseEvent(t *testing.T) {

	t.Parallel()

	t.Run("no parameters", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("event E()")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindEvent,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Offset: 6, Line: 1, Column: 6},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindInitializer,
								FunctionDeclaration: &ast.FunctionDeclaration{
									Access: ast.AccessNotSpecified,
									ParameterList: &ast.ParameterList{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
											EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
										},
									},
									StartPos: ast.Position{Offset: 7, Line: 1, Column: 7},
								},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
			},
			result,
		)
	})

	t.Run("two parameters, private", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(self) event E2 ( a : Int , b : String )")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "Int",
															Pos:        ast.Position{Offset: 29, Line: 1, Column: 29},
														},
													},
													StartPos: ast.Position{Offset: 29, Line: 1, Column: 29},
												},
												Identifier: ast.Identifier{
													Identifier: "a",
													Pos:        ast.Position{Offset: 25, Line: 1, Column: 25},
												},
												StartPos: ast.Position{Offset: 25, Line: 1, Column: 25},
											},
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "String",
															Pos:        ast.Position{Offset: 39, Line: 1, Column: 39},
														},
													},
													StartPos: ast.Position{Offset: 39, Line: 1, Column: 39},
												},
												Identifier: ast.Identifier{
													Identifier: "b",
													Pos:        ast.Position{Offset: 35, Line: 1, Column: 35},
												},
												StartPos: ast.Position{Offset: 35, Line: 1, Column: 35},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 23, Line: 1, Column: 23},
											EndPos:   ast.Position{Offset: 46, Line: 1, Column: 46},
										},
									},
									StartPos: ast.Position{Offset: 23, Line: 1, Column: 23},
									Access:   ast.AccessSelf,
								},
								Kind: common.DeclarationKindInitializer,
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "E2",
						Pos:        ast.Position{Offset: 20, Line: 1, Column: 20},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 46, Line: 1, Column: 46},
					},
					Access:        ast.AccessSelf,
					CompositeKind: common.CompositeKindEvent,
				},
			},
			result,
		)
	})

	t.Run("default event", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          access(all) event ResourceDestroyed ( a : String = "foo")
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "String",
															Pos:        ast.Position{Offset: 53, Line: 2, Column: 52},
														},
													},
													StartPos: ast.Position{Offset: 53, Line: 2, Column: 52},
												},
												DefaultArgument: &ast.StringExpression{
													Value: "foo",
													Range: ast.Range{
														StartPos: ast.Position{Offset: 62, Line: 2, Column: 61},
														EndPos:   ast.Position{Offset: 66, Line: 2, Column: 65},
													},
												},
												Identifier: ast.Identifier{
													Identifier: "a",
													Pos:        ast.Position{Offset: 49, Line: 2, Column: 48},
												},
												StartPos: ast.Position{Offset: 49, Line: 2, Column: 48},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 47, Line: 2, Column: 46},
											EndPos:   ast.Position{Offset: 67, Line: 2, Column: 66},
										},
									},
									StartPos: ast.Position{Offset: 47, Line: 2, Column: 46},
									Access:   ast.AccessAll,
								},
								Kind: common.DeclarationKindInitializer,
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "ResourceDestroyed",
						Pos:        ast.Position{Offset: 29, Line: 2, Column: 28},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 67, Line: 2, Column: 66},
					},
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindEvent,
				},
			},
			result,
		)
	})

	t.Run("default event with no default arg", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations(`
          access(all) event ResourceDestroyed ( a : Int )
        `)

		AssertEqualWithDiff(t,
			[]error{
				&MissingDefaultArgumentError{
					GotToken: lexer.Token{
						Type: lexer.TokenParenClose,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 56, Offset: 57},
							EndPos:   ast.Position{Line: 2, Column: 56, Offset: 57},
						},
					},
				},
			},
			errs,
		)
	})

	t.Run("non-default event with default arg", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations(`
          access(all) event Foo ( a : Int = 3)
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedDefaultArgumentError{
					Pos: ast.Position{Line: 2, Column: 42, Offset: 43},
				},
			},
			errs,
		)
	})

	t.Run("invalid event name", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations(`
          event continue {}
        `)

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Pos:     ast.Position{Line: 2, Column: 16, Offset: 17},
					Message: "expected identifier after start of event declaration, got keyword continue",
				},
			},
			errs,
		)
	})

	t.Run("leading separator in conformances", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations("struct Test: , I {}")

		AssertEqualWithDiff(t,
			[]error{
				&ExpectedTypeInsteadSeparatorError{
					Pos:       ast.Position{Offset: 13, Line: 1, Column: 13},
					Separator: lexer.TokenComma,
				},
			},
			errs,
		)
	})
}

func TestParseFieldWithVariableKind(t *testing.T) {

	t.Parallel()

	parse := func(input string) (*ast.FieldDeclaration, []error) {
		return Parse(
			nil,
			[]byte(input),
			func(p *parser) (*ast.FieldDeclaration, error) {
				return parseFieldWithVariableKind(
					p,
					ast.AccessNotSpecified,
					nil,
					nil,
					nil,
					"",
				)
			},
			Config{},
		)
	}

	t.Run("variable", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("var x : Int")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				VariableKind: ast.VariableKindVariable,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
				},
			},
			result,
		)
	})

	t.Run("constant", func(t *testing.T) {

		t.Parallel()

		result, errs := parse("let x : Int")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				VariableKind: ast.VariableKindConstant,
				Identifier: ast.Identifier{
					Identifier: "x",
					Pos:        ast.Position{Line: 1, Column: 4, Offset: 4},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: false,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 8, Offset: 8},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 8, Offset: 8},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 10, Offset: 10},
				},
			},
			result,
		)
	})

	t.Run("missing identifier", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("let : Int")

		AssertEqualWithDiff(t,
			[]error{
				&MissingFieldNameError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 4, Line: 1, Column: 4},
							EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
						},
						Type: lexer.TokenColon,
					},
				},
			},
			errs,
		)
	})

	t.Run("missing colon", func(t *testing.T) {

		t.Parallel()

		const code = "let x Int"
		_, errs := parse(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingColonAfterFieldNameError{
					GotToken: lexer.Token{
						Type: lexer.TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 6, Line: 1, Column: 6},
							EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
						},
					},
				},
			},
			errs,
		)

		var missingColonErr *MissingColonAfterFieldNameError
		require.ErrorAs(t, errs[0], &missingColonErr)

		fixes := missingColonErr.SuggestFixes(code)
		AssertEqualWithDiff(
			t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert colon",
					TextEdits: []ast.TextEdit{
						{
							Insertion: ":",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 5, Line: 1, Column: 5},
								EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"let x: Int",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})
}

func TestParseField(t *testing.T) {

	t.Parallel()

	parse := func(input string, config Config) (ast.Declaration, []error) {
		return Parse(
			nil,
			[]byte(input),
			func(p *parser) (ast.Declaration, error) {
				return parseMemberOrNestedDeclaration(
					p,
					"",
				)
			},
			config,
		)
	}

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(
			"native let foo: Int",
			Config{
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				Flags:        ast.FieldDeclarationFlagsIsNative,
				VariableKind: ast.VariableKindConstant,
				Identifier: ast.Identifier{
					Identifier: "foo",
					Pos:        ast.Position{Line: 1, Column: 11, Offset: 11},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 16, Offset: 16},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
				},
			},
			result,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("native let foo: Int", Config{})

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(
			"static let foo: Int",
			Config{
				StaticModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				Flags:        ast.FieldDeclarationFlagsIsStatic,
				VariableKind: ast.VariableKindConstant,
				Identifier: ast.Identifier{
					Identifier: "foo",
					Pos:        ast.Position{Line: 1, Column: 11, Offset: 11},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 16, Offset: 16},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 16, Offset: 16},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 18, Offset: 18},
				},
			},
			result,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"static let foo: Int",
			Config{},
		)

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(
			"static native let foo: Int",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				Access:       ast.AccessNotSpecified,
				Flags:        ast.FieldDeclarationFlagsIsStatic | ast.FieldDeclarationFlagsIsNative,
				VariableKind: ast.VariableKindConstant,
				Identifier: ast.Identifier{
					Identifier: "foo",
					Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Line: 1, Column: 23, Offset: 23},
						},
					},
					StartPos: ast.Position{Line: 1, Column: 23, Offset: 23},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
				},
			},
			result,
		)
	})

	t.Run("static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("static native let foo: Int", Config{})

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"native static let foo: Int",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid `static` modifier after `native` modifier",
					Pos:     ast.Position{Offset: 7, Line: 1, Column: 7},
				},
			},
			errs,
		)
	})

	t.Run("access(all) static native, enabled", func(t *testing.T) {

		t.Parallel()

		result, errs := parse(
			"access(all) static native let foo: Int",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			&ast.FieldDeclaration{
				TypeAnnotation: &ast.TypeAnnotation{
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "Int",
							Pos:        ast.Position{Offset: 35, Line: 1, Column: 35},
						},
					},
					StartPos:   ast.Position{Offset: 35, Line: 1, Column: 35},
					IsResource: false,
				},
				Identifier: ast.Identifier{
					Identifier: "foo",
					Pos:        ast.Position{Offset: 30, Line: 1, Column: 30},
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					EndPos:   ast.Position{Offset: 37, Line: 1, Column: 37},
				},
				Access:       ast.AccessAll,
				VariableKind: ast.VariableKindConstant,
				Flags:        ast.FieldDeclarationFlagsIsStatic | ast.FieldDeclarationFlagsIsNative,
			},
			result,
		)
	})

	t.Run("access(all) static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("access(all) static native let foo: Int", Config{})

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 12, Line: 1, Column: 12},
				},
			},
			errs,
		)
	})

}

func TestParseCompositeDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("struct, no conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) struct S { }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(nil),
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Offset: 20, Line: 1, Column: 20},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 24, Line: 1, Column: 24},
					},
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindStructure,
				},
			},
			result,
		)
	})

	t.Run("resource, one conformance", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) resource R : RI { }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindResource,
					Identifier: ast.Identifier{
						Identifier: "R",
						Pos:        ast.Position{Line: 1, Column: 22, Offset: 22},
					},
					Conformances: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "RI",
								Pos:        ast.Position{Line: 1, Column: 26, Offset: 26},
							},
						},
					},
					Members: &ast.Members{},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 31, Offset: 31},
					},
				},
			},
			result,
		)
	})

	t.Run("struct, one conformance, missing body", func(t *testing.T) {

		t.Parallel()

		const code = "access(all) struct S: RI"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&DeclarationMissingOpeningBraceError{
					Kind: common.DeclarationKindStructure,
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 24, Line: 1, Column: 24},
							EndPos:   ast.Position{Offset: 24, Line: 1, Column: 24},
						},
						Type: lexer.TokenEOF,
					},
				},
				&DeclarationMissingClosingBraceError{
					Kind: common.DeclarationKindStructure,
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 24, Line: 1, Column: 24},
							EndPos:   ast.Position{Offset: 24, Line: 1, Column: 24},
						},
						Type: lexer.TokenEOF,
					},
				},
			},
			errs,
		)

		var missingOpeningBraceErr *DeclarationMissingOpeningBraceError
		require.ErrorAs(t, errs[0], &missingOpeningBraceErr)

		fixes := missingOpeningBraceErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert opening brace",
					TextEdits: []ast.TextEdit{
						{
							Insertion: " {",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 24, Line: 1, Column: 24},
								EndPos:   ast.Position{Offset: 24, Line: 1, Column: 24},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"access(all) struct S: RI {",
			fixes[0].TextEdits[0].ApplyTo(code),
		)

		var missingClosingBraceErr *DeclarationMissingClosingBraceError
		require.ErrorAs(t, errs[1], &missingClosingBraceErr)

		fixes = missingClosingBraceErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing brace",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "}",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 24, Line: 1, Column: 24},
								EndPos:   ast.Position{Offset: 24, Line: 1, Column: 24},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"access(all) struct S: RI}",
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("struct, one conformance, missing type after comma", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("access(all) struct S: RI, ")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedEOFExpectedTypeError{
					Pos: ast.Position{Offset: 26, Line: 1, Column: 26},
				},
			},
			errs,
		)
	})

	t.Run("struct, with fields, functions, and special functions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct Test {
              access(all) var foo: Int

              init(foo: Int) {
                  self.foo = foo
              }

              access(all) fun getFoo(): Int {
                  return self.foo
              }
          }
        `)

		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FieldDeclaration{
								TypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 60, Line: 3, Column: 35},
										},
									},
									StartPos:   ast.Position{Offset: 60, Line: 3, Column: 35},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos:        ast.Position{Offset: 55, Line: 3, Column: 30},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 39, Line: 3, Column: 14},
									EndPos:   ast.Position{Offset: 62, Line: 3, Column: 37},
								},
								Access:       ast.AccessAll,
								VariableKind: ast.VariableKindVariable,
								Flags:        0,
							},
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "Int",
															Pos:        ast.Position{Offset: 89, Line: 5, Column: 24},
														},
													},
													StartPos:   ast.Position{Offset: 89, Line: 5, Column: 24},
													IsResource: false,
												},
												Identifier: ast.Identifier{
													Identifier: "foo",
													Pos:        ast.Position{Offset: 84, Line: 5, Column: 19},
												},
												StartPos: ast.Position{Offset: 84, Line: 5, Column: 19},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 83, Line: 5, Column: 18},
											EndPos:   ast.Position{Offset: 92, Line: 5, Column: 27},
										},
									},
									FunctionBlock: &ast.FunctionBlock{
										Block: &ast.Block{
											Statements: []ast.Statement{
												&ast.AssignmentStatement{
													Target: &ast.MemberExpression{
														Expression: &ast.IdentifierExpression{
															Identifier: ast.Identifier{
																Identifier: "self",
																Pos:        ast.Position{Offset: 114, Line: 6, Column: 18},
															},
														},
														Identifier: ast.Identifier{
															Identifier: "foo",
															Pos:        ast.Position{Offset: 119, Line: 6, Column: 23},
														},
														AccessEndPos: ast.Position{Offset: 118, Line: 6, Column: 22},
														Optional:     false,
													},
													Transfer: &ast.Transfer{
														Operation: ast.TransferOperationCopy,
														Pos:       ast.Position{Offset: 123, Line: 6, Column: 27},
													},
													Value: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "foo",
															Pos:        ast.Position{Offset: 125, Line: 6, Column: 29},
														},
													},
												},
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 94, Line: 5, Column: 29},
												EndPos:   ast.Position{Offset: 143, Line: 7, Column: 14},
											},
										},
									},
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos:        ast.Position{Offset: 79, Line: 5, Column: 14},
									},
									StartPos: ast.Position{Offset: 79, Line: 5, Column: 14},
									Access:   ast.AccessNotSpecified,
									Flags:    0,
								},
								Kind: common.DeclarationKindInitializer,
							},
							&ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 182, Line: 9, Column: 36},
										EndPos:   ast.Position{Offset: 183, Line: 9, Column: 37},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 186, Line: 9, Column: 40},
										},
									},
									StartPos:   ast.Position{Offset: 186, Line: 9, Column: 40},
									IsResource: false,
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Statements: []ast.Statement{
											&ast.ReturnStatement{
												Expression: &ast.MemberExpression{
													Expression: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "self",
															Pos:        ast.Position{Offset: 217, Line: 10, Column: 25},
														},
													},
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos:        ast.Position{Offset: 222, Line: 10, Column: 30},
													},
													AccessEndPos: ast.Position{Offset: 221, Line: 10, Column: 29},
													Optional:     false,
												},
												Range: ast.Range{
													StartPos: ast.Position{Offset: 210, Line: 10, Column: 18},
													EndPos:   ast.Position{Offset: 224, Line: 10, Column: 32},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 190, Line: 9, Column: 44},
											EndPos:   ast.Position{Offset: 240, Line: 11, Column: 14},
										},
									},
								},
								DocString: "",
								Identifier: ast.Identifier{
									Identifier: "getFoo",
									Pos:        ast.Position{Offset: 176, Line: 9, Column: 30},
								},
								StartPos: ast.Position{Offset: 160, Line: 9, Column: 14},
								Access:   ast.AccessAll,
								Flags:    0,
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 252, Line: 12, Column: 10},
					},
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
				},
			},
			result,
		)
	})

	t.Run("struct with view member", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct S {
              view fun foo() {}
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,

			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 2, Column: 17, Offset: 18},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FunctionDeclaration{
								Purity: ast.FunctionPurityView,
								Access: ast.AccessNotSpecified,
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 48, Line: 3, Column: 26},
										EndPos:   ast.Position{Offset: 49, Line: 3, Column: 27},
									},
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos:        ast.Position{Offset: 45, Line: 3, Column: 23},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 51, Line: 3, Column: 29},
											EndPos:   ast.Position{Offset: 52, Line: 3, Column: 30},
										},
									},
								},
								StartPos: ast.Position{Offset: 36, Line: 3, Column: 14},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 10, Offset: 11},
						EndPos:   ast.Position{Line: 4, Column: 10, Offset: 64},
					},
				},
			},
			result,
		)
	})

	t.Run("struct with view initializer", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct S {
              view init() {}
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 2, Column: 17, Offset: 18},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{&ast.SpecialFunctionDeclaration{
							Kind: common.DeclarationKindInitializer,
							FunctionDeclaration: &ast.FunctionDeclaration{
								Access: ast.AccessNotSpecified,
								Purity: ast.FunctionPurityView,
								Identifier: ast.Identifier{
									Identifier: "init",
									Pos:        ast.Position{Offset: 41, Line: 3, Column: 19},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 45, Line: 3, Column: 23},
										EndPos:   ast.Position{Offset: 46, Line: 3, Column: 24},
									},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 48, Line: 3, Column: 26},
											EndPos:   ast.Position{Offset: 49, Line: 3, Column: 27},
										},
									},
								},
								StartPos: ast.Position{Offset: 36, Line: 3, Column: 14},
							},
						}},
					),
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 10, Offset: 11},
						EndPos:   ast.Position{Line: 4, Column: 10, Offset: 61},
					},
				},
			},
			result,
		)
	})

	t.Run("composite with view field", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          struct S {
              view foo: Int
          }
        `)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidViewModifierError{
					Pos:             ast.Position{Offset: 36, Line: 3, Column: 14},
					DeclarationKind: common.DeclarationKindField,
				},
			},
			errs,
		)
	})
}

func TestParseInvalidCompositeFunctionWithSelfParameter(t *testing.T) {

	t.Parallel()

	for _, kind := range common.CompositeKindsWithFieldsAndFunctions {
		t.Run(kind.Keyword(), func(t *testing.T) {

			var baseType = ""

			if kind == common.CompositeKindAttachment {
				baseType = "for AnyStruct"
			}

			code := fmt.Sprintf(`%s Foo %s { fun test(_ self: Int) {} }`, kind.Keyword(), baseType)

			selfKeywordPos := strings.Index(code, "self")

			expectedErrPos := ast.Position{Line: 1, Column: selfKeywordPos, Offset: selfKeywordPos}

			_, err := testParseDeclarations(code)

			AssertEqualWithDiff(
				t,
				[]error{
					&SyntaxError{
						Pos:     expectedErrPos,
						Message: "expected identifier for parameter name, got keyword self",
					},
				},
				err,
			)
		})
	}
}

func TestParseInvalidParameterWithoutLabel(t *testing.T) {
	t.Parallel()

	_, errs := testParseDeclarations(`
      access(all) fun foo(continue: Int) {}
    `)

	AssertEqualWithDiff(t,
		[]error{
			&SyntaxError{
				Pos:     ast.Position{Line: 2, Column: 26, Offset: 27},
				Message: "expected identifier for argument label or parameter name, got keyword continue",
			},
		},
		errs,
	)
}

func TestParseParametersWithExtraLabels(t *testing.T) {
	t.Parallel()

	_, errs := testParseDeclarations(`
      access(all) fun foo(_ foo: String, label fable table: Int) {}
    `)

	AssertEqualWithDiff(t,
		[]error{
			&MissingColonAfterParameterNameError{
				GotToken: lexer.Token{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 54, Line: 2, Column: 53},
						EndPos:   ast.Position{Offset: 58, Line: 2, Column: 57},
					},
					Type: lexer.TokenIdentifier,
				},
			},
		},
		errs,
	)
}

func TestParseAttachmentDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("no conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("access(all) attachment E for S {} ")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Line: 1, Column: 23, Offset: 23},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos:        ast.Position{Line: 1, Column: 29, Offset: 29},
						},
					},
					Members: &ast.Members{},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 32, Offset: 32},
					},
				},
			},
			result,
		)
	})

	t.Run("nested in contract", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          contract Test {
              access(all) attachment E for S {}
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,

			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindContract,
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Line: 2, Column: 19, Offset: 20},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 10, Offset: 11},
						EndPos:   ast.Position{Line: 4, Column: 10, Offset: 85},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.AttachmentDeclaration{
								Access: ast.AccessAll,
								Identifier: ast.Identifier{
									Identifier: "E",
									Pos:        ast.Position{Line: 3, Column: 37, Offset: 64},
								},
								BaseType: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "S",
										Pos:        ast.Position{Line: 3, Column: 43, Offset: 70},
									},
								},
								Members: &ast.Members{},
								Range: ast.Range{
									StartPos: ast.Position{Line: 3, Column: 14, Offset: 41},
									EndPos:   ast.Position{Line: 3, Column: 46, Offset: 73},
								},
							},
						},
					),
				},
			},
			result,
		)
	})

	t.Run("missing for keyword", func(t *testing.T) {

		t.Parallel()

		const code = `
          attachment E {}
        `
		_, errs := testParseDeclarations(code)
		AssertEqualWithDiff(t,
			[]error{
				&MissingForKeywordInAttachmentDeclarationError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
							EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
						},
						Type: lexer.TokenBraceOpen,
					},
				},
				&InvalidAttachmentBaseTypeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
						EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
					},
				},
				&DeclarationMissingOpeningBraceError{
					Kind: common.DeclarationKindAttachment,
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 35, Line: 3, Column: 8},
							EndPos:   ast.Position{Offset: 35, Line: 3, Column: 8},
						},
						Type: lexer.TokenEOF,
					},
				},
				&DeclarationMissingClosingBraceError{
					Kind: common.DeclarationKindAttachment,
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 35, Line: 3, Column: 8},
							EndPos:   ast.Position{Offset: 35, Line: 3, Column: 8},
						},
						Type: lexer.TokenEOF,
					},
				},
			},
			errs,
		)

		var missingForErr *MissingForKeywordInAttachmentDeclarationError
		require.ErrorAs(t, errs[0], &missingForErr)

		fixes := missingForErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert `for`",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "for ",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
								EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
							},
						},
					},
				},
			},
			fixes,
		)

		const expected = `
          attachment E for {}
        `
		assert.Equal(t,
			expected,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("missing for keyword at end", func(t *testing.T) {

		t.Parallel()

		const code = `
          attachment E`

		_, errs := testParseDeclarations(code)
		AssertEqualWithDiff(t,
			[]error{
				&MissingForKeywordInAttachmentDeclarationError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
							EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
						},
						Type: lexer.TokenEOF,
					},
				},
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						SpaceOrError: nil,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
							EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
						},
						Type: lexer.TokenEOF,
					},
				},
			},
			errs,
		)

		var missingForErr *MissingForKeywordInAttachmentDeclarationError
		require.ErrorAs(t, errs[0], &missingForErr)

		fixes := missingForErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert `for`",
					TextEdits: []ast.TextEdit{
						{
							Insertion: " for ",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
								EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
							},
						},
					},
				},
			},
			fixes,
		)

		const expected = `
          attachment E for `
		assert.Equal(t,
			expected,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("one conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("access(all) attachment E for S: I {} ")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Line: 1, Column: 23, Offset: 23},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos:        ast.Position{Line: 1, Column: 29, Offset: 29},
						},
					},
					Members: &ast.Members{},
					Conformances: []*ast.NominalType{
						ast.NewNominalType(
							nil,
							ast.Identifier{
								Identifier: "I",
								Pos:        ast.Position{Line: 1, Column: 32, Offset: 32},
							},
							nil,
						),
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 35, Offset: 35},
					},
				},
			},
			result,
		)
	})

	t.Run("two conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("access(all) attachment E for S: I1, I2 {} ")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Offset: 23, Line: 1, Column: 23},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos:        ast.Position{Offset: 29, Line: 1, Column: 29},
						},
					},
					Conformances: []*ast.NominalType{
						{
							Identifier: ast.Identifier{
								Identifier: "I1",
								Pos:        ast.Position{Offset: 32, Line: 1, Column: 32},
							},
						},
						{
							Identifier: ast.Identifier{
								Identifier: "I2",
								Pos:        ast.Position{Offset: 36, Line: 1, Column: 36},
							},
						},
					},
					Members: ast.NewUnmeteredMembers(nil),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 40, Line: 1, Column: 40},
					},
				},
			},
			result,
		)
	})

	t.Run("fields, functions and special functions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          access(all) attachment E for S {
              access(all) var foo: Int

              init() {}

              access(all) fun getFoo(): Int {}
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,

			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Offset: 34, Line: 2, Column: 33},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos:        ast.Position{Offset: 40, Line: 2, Column: 39},
						},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FieldDeclaration{
								TypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 79, Line: 3, Column: 35},
										},
									},
									StartPos:   ast.Position{Offset: 79, Line: 3, Column: 35},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos:        ast.Position{Offset: 74, Line: 3, Column: 30},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 58, Line: 3, Column: 14},
									EndPos:   ast.Position{Offset: 81, Line: 3, Column: 37},
								},
								Access:       ast.AccessAll,
								VariableKind: ast.VariableKindVariable,
							},
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 102, Line: 5, Column: 18},
											EndPos:   ast.Position{Offset: 103, Line: 5, Column: 19},
										},
									},
									FunctionBlock: &ast.FunctionBlock{
										Block: &ast.Block{
											Range: ast.Range{
												StartPos: ast.Position{Offset: 105, Line: 5, Column: 21},
												EndPos:   ast.Position{Offset: 106, Line: 5, Column: 22},
											},
										},
									},
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos:        ast.Position{Offset: 98, Line: 5, Column: 14},
									},
									StartPos: ast.Position{Offset: 98, Line: 5, Column: 14},
									Access:   ast.AccessNotSpecified,
								},
								Kind: common.DeclarationKindInitializer,
							},
							&ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 145, Line: 7, Column: 36},
										EndPos:   ast.Position{Offset: 146, Line: 7, Column: 37},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 149, Line: 7, Column: 40},
										},
									},
									StartPos:   ast.Position{Offset: 149, Line: 7, Column: 40},
									IsResource: false,
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 153, Line: 7, Column: 44},
											EndPos:   ast.Position{Offset: 154, Line: 7, Column: 45},
										},
									},
								},
								Identifier: ast.Identifier{
									Identifier: "getFoo",
									Pos:        ast.Position{Offset: 139, Line: 7, Column: 30},
								},
								StartPos: ast.Position{Offset: 123, Line: 7, Column: 14},
								Access:   ast.AccessAll,
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 166, Line: 8, Column: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("required entitlements error", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          access(all) attachment E for S {
              require entitlement X
          }
        `)
		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Pos:     ast.Position{Line: 3, Column: 14, Offset: 58},
					Message: "unexpected identifier",
				},
			},
			errs,
		)
	})

	t.Run("entitlement access", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          access(all) attachment E for S {
              access(X) var foo: Int
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,

			[]ast.Declaration{
				&ast.AttachmentDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Offset: 34, Line: 2, Column: 33},
					},
					BaseType: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "S",
							Pos:        ast.Position{Offset: 40, Line: 2, Column: 39},
						},
					},
					Members: ast.NewUnmeteredMembers(

						[]ast.Declaration{
							&ast.FieldDeclaration{
								TypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 77, Line: 3, Column: 33},
										},
									},
									StartPos:   ast.Position{Offset: 77, Line: 3, Column: 33},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos:        ast.Position{Offset: 72, Line: 3, Column: 28},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 58, Line: 3, Column: 14},
									EndPos:   ast.Position{Offset: 79, Line: 3, Column: 35},
								},
								Access: ast.EntitlementAccess{
									EntitlementSet: &ast.ConjunctiveEntitlementSet{
										Elements: []*ast.NominalType{
											{
												Identifier: ast.Identifier{
													Identifier: "X",
													Pos:        ast.Position{Offset: 65, Line: 3, Column: 21},
												},
											},
										},
									},
								},
								VariableKind: ast.VariableKindVariable,
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 91, Line: 4, Column: 10},
					},
				},
			},
			result,
		)
	})
}

func TestParseInterfaceDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("struct, no conformances", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) struct interface S { }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.InterfaceDeclaration{
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 1, Column: 30, Offset: 30},
					},
					Members: &ast.Members{},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 34, Offset: 34},
					},
				},
			},
			result,
		)
	})

	t.Run("struct, interface keyword as name", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) struct interface interface { }")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidInterfaceNameError{
					GotToken: lexer.Token{
						Type: lexer.TokenIdentifier,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 30, Line: 1, Column: 30},
							EndPos:   ast.Position{Offset: 38, Line: 1, Column: 38},
						},
					},
				},
			},
			errs,
		)

		var expected []ast.Declaration

		AssertEqualWithDiff(t,
			expected,
			result,
		)
	})

	t.Run("struct, with fields, functions, and special functions; with and without blocks", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct interface Test {
              access(all) var foo: Int

              init(foo: Int)

              access(all) fun getFoo(): Int

              access(all) fun getBar(): Int {}
          }
        `)

		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.InterfaceDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FieldDeclaration{
								TypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 70, Line: 3, Column: 35},
										},
									},
									StartPos:   ast.Position{Offset: 70, Line: 3, Column: 35},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos:        ast.Position{Offset: 65, Line: 3, Column: 30},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 49, Line: 3, Column: 14},
									EndPos:   ast.Position{Offset: 72, Line: 3, Column: 37},
								},
								Access:       ast.AccessAll,
								VariableKind: ast.VariableKindVariable,
								Flags:        0,
							},
							&ast.SpecialFunctionDeclaration{
								FunctionDeclaration: &ast.FunctionDeclaration{
									ParameterList: &ast.ParameterList{
										Parameters: []*ast.Parameter{
											{
												TypeAnnotation: &ast.TypeAnnotation{
													Type: &ast.NominalType{
														Identifier: ast.Identifier{
															Identifier: "Int",
															Pos:        ast.Position{Offset: 99, Line: 5, Column: 24},
														},
													},
													StartPos:   ast.Position{Offset: 99, Line: 5, Column: 24},
													IsResource: false,
												},
												Identifier: ast.Identifier{
													Identifier: "foo",
													Pos:        ast.Position{Offset: 94, Line: 5, Column: 19},
												},
												StartPos: ast.Position{Offset: 94, Line: 5, Column: 19},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 93, Line: 5, Column: 18},
											EndPos:   ast.Position{Offset: 102, Line: 5, Column: 27},
										},
									},
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos:        ast.Position{Offset: 89, Line: 5, Column: 14},
									},
									StartPos: ast.Position{Offset: 89, Line: 5, Column: 14},
									Access:   ast.AccessNotSpecified,
									Flags:    0,
								},
								Kind: common.DeclarationKindInitializer,
							},
							&ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 141, Line: 7, Column: 36},
										EndPos:   ast.Position{Offset: 142, Line: 7, Column: 37},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 145, Line: 7, Column: 40},
										},
									},
									StartPos:   ast.Position{Offset: 145, Line: 7, Column: 40},
									IsResource: false,
								},
								Identifier: ast.Identifier{
									Identifier: "getFoo",
									Pos:        ast.Position{Offset: 135, Line: 7, Column: 30},
								},
								StartPos: ast.Position{Offset: 119, Line: 7, Column: 14},
								Access:   ast.AccessAll,
								Flags:    0,
							},
							&ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 186, Line: 9, Column: 36},
										EndPos:   ast.Position{Offset: 187, Line: 9, Column: 37},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 190, Line: 9, Column: 40},
										},
									},
									StartPos:   ast.Position{Offset: 190, Line: 9, Column: 40},
									IsResource: false,
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 194, Line: 9, Column: 44},
											EndPos:   ast.Position{Offset: 195, Line: 9, Column: 45},
										},
									},
								},
								Identifier: ast.Identifier{
									Identifier: "getBar",
									Pos:        ast.Position{Offset: 180, Line: 9, Column: 30},
								},
								StartPos: ast.Position{Offset: 164, Line: 9, Column: 14},
								Access:   ast.AccessAll,
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Offset: 28, Line: 2, Column: 27},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 207, Line: 10, Column: 10},
					},
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
				},
			},
			result,
		)
	})

	t.Run("invalid interface name", func(t *testing.T) {
		_, errs := testParseDeclarations(`
          access(all) struct interface continue {}
        `)

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Pos:     ast.Position{Line: 2, Column: 39, Offset: 40},
					Message: "expected identifier following struct declaration, got keyword continue",
				},
			},
			errs,
		)
	})

	t.Run("struct with view member", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct interface S {
              view fun foo() {}
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.InterfaceDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "S",
						Pos:        ast.Position{Line: 2, Column: 27, Offset: 28},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FunctionDeclaration{
								Purity: ast.FunctionPurityView,
								Access: ast.AccessNotSpecified,
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 58, Line: 3, Column: 26},
										EndPos:   ast.Position{Offset: 59, Line: 3, Column: 27},
									},
								},
								Identifier: ast.Identifier{
									Identifier: "foo",
									Pos:        ast.Position{Offset: 55, Line: 3, Column: 23},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 61, Line: 3, Column: 29},
											EndPos:   ast.Position{Offset: 62, Line: 3, Column: 30},
										},
									},
								},
								StartPos: ast.Position{Offset: 46, Line: 3, Column: 14},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Line: 2, Column: 10, Offset: 11},
						EndPos:   ast.Position{Line: 4, Column: 10, Offset: 74},
					},
				},
			},
			result,
		)
	})
}

func TestParseEnumDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("enum, two cases one one line", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) enum E { case c ; access(all) case d }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindEnum,
					Identifier: ast.Identifier{
						Identifier: "E",
						Pos:        ast.Position{Line: 1, Column: 18, Offset: 18},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.EnumCaseDeclaration{
								Access: ast.AccessNotSpecified,
								Identifier: ast.Identifier{
									Identifier: "c",
									Pos:        ast.Position{Line: 1, Column: 27, Offset: 27},
								},
								StartPos: ast.Position{Line: 1, Column: 22, Offset: 22},
							},
							&ast.EnumCaseDeclaration{
								Access: ast.AccessAll,
								Identifier: ast.Identifier{
									Identifier: "d",
									Pos:        ast.Position{Line: 1, Column: 48, Offset: 48},
								},
								StartPos: ast.Position{Line: 1, Column: 31, Offset: 31},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 50, Offset: 50},
					},
				},
			},
			result,
		)
	})

	t.Run("enum case with view modifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" enum E { view case e }")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidViewModifierError{
					Pos:             ast.Position{Offset: 10, Line: 1, Column: 10},
					DeclarationKind: common.DeclarationKindEnumCase,
				},
			},
			errs,
		)
	})

	t.Run("enum case with static modifier, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			" enum E { static case e }",
			Config{
				StaticModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 10, Line: 1, Column: 10},
					DeclarationKind: common.DeclarationKindEnumCase,
				},
			},
			errs,
		)
	})

	t.Run("enum case with static modifier, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" enum E { static case e }")

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
		)
	})

	t.Run("enum case with native modifier, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			" enum E { native case e }",
			Config{
				NativeModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 10, Line: 1, Column: 10},
					DeclarationKind: common.DeclarationKindEnumCase,
				},
			},
			errs,
		)
	})

	t.Run("enum case with native modifier, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" enum E { native case e }")

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 10, Line: 1, Column: 10},
				},
			},
			errs,
		)
	})

	t.Run("enum case missing identifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("enum E { case }")
		AssertEqualWithDiff(t,
			[]error{
				&MissingEnumCaseNameError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
							EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
						},
						Type: lexer.TokenBraceClose,
					},
				},
			},
			errs,
		)
	})
}

func TestParseTransactionDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("no prepare, execute", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations("transaction { execute {} }")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Execute: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindExecute,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "execute",
								Pos:        ast.Position{Offset: 14, Line: 1, Column: 14},
							},
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 22, Line: 1, Column: 22},
										EndPos:   ast.Position{Offset: 23, Line: 1, Column: 23},
									},
								},
							},
							StartPos: ast.Position{Offset: 14, Line: 1, Column: 14},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 25, Offset: 25},
					},
				},
			},
			result,
		)
	})

	t.Run("duplicate execute", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("transaction { execute {}  execute {}}")
		AssertEqualWithDiff(t,
			[]error{
				&DuplicateExecuteBlockError{
					Pos: ast.Position{Offset: 26, Line: 1, Column: 26},
				},
			},
			errs,
		)
	})

	t.Run("unexpected initial identifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("transaction { foo }")
		AssertEqualWithDiff(t,
			[]error{
				&ExpectedPrepareOrExecuteError{
					GotIdentifier: "foo",
					Pos:           ast.Position{Offset: 14, Line: 1, Column: 14},
				},
			},
			errs,
		)
	})

	t.Run("duplicate post", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("transaction { execute {} post {} post {} }")
		AssertEqualWithDiff(t,
			[]error{
				&DuplicatePostConditionsError{
					Pos: ast.Position{Offset: 33, Line: 1, Column: 33},
				},
			},
			errs,
		)
	})

	t.Run("unexpected identifier after prepare", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("transaction { prepare() {} foo }")
		AssertEqualWithDiff(t,
			[]error{
				&ExpectedExecuteOrPostError{
					GotIdentifier: "foo",
					Pos:           ast.Position{Offset: 27, Line: 1, Column: 27},
				},
			},
			errs,
		)
	})

	t.Run("unexpected identifier after execute", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("transaction { execute {} foo }")
		AssertEqualWithDiff(t,
			[]error{
				&ExpectedExecuteOrPostError{
					GotIdentifier: "foo",
					Pos:           ast.Position{Offset: 25, Line: 1, Column: 25},
				},
			},
			errs,
		)
	})

	t.Run("unexpected token at end", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("transaction { execute {} .")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 25, Line: 1, Column: 25},
							EndPos:   ast.Position{Offset: 25, Line: 1, Column: 25},
						},
						Type: lexer.TokenDot,
					},
				},
			},
			errs,
		)
	})

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		const code = `
          transaction {}
        `
		result, errs := testParseProgram(code)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
				},
			},
			result.Declarations(),
		)
	})

	t.Run("SimpleTransaction", func(t *testing.T) {

		t.Parallel()

		const code = `
          transaction {

            var x: Int

            prepare(signer: &Account) {
                x = 0
            }

            execute {
                x = 1 + 1
            }
          }
        `
		result, errs := testParseProgram(code)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Fields: []*ast.FieldDeclaration{
						{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindVariable,
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 42, Line: 4, Column: 16},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 45, Line: 4, Column: 19},
									},
								},
								StartPos: ast.Position{Offset: 45, Line: 4, Column: 19},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 38, Line: 4, Column: 12},
								EndPos:   ast.Position{Offset: 47, Line: 4, Column: 21},
							},
						},
					},
					Prepare: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindPrepare,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "prepare",
								Pos:        ast.Position{Offset: 62, Line: 6, Column: 12},
							},
							ParameterList: &ast.ParameterList{
								Parameters: []*ast.Parameter{
									{
										Label: "",
										Identifier: ast.Identifier{
											Identifier: "signer",
											Pos:        ast.Position{Offset: 70, Line: 6, Column: 20},
										},
										TypeAnnotation: &ast.TypeAnnotation{
											IsResource: false,
											Type: &ast.ReferenceType{
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Account",
														Pos:        ast.Position{Offset: 79, Line: 6, Column: 29},
													},
												},
												StartPos: ast.Position{Offset: 78, Line: 6, Column: 28},
											},
											StartPos: ast.Position{Offset: 78, Line: 6, Column: 28},
										},
										StartPos: ast.Position{Offset: 70, Line: 6, Column: 20},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 69, Line: 6, Column: 19},
									EndPos:   ast.Position{Offset: 86, Line: 6, Column: 36},
								},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 106, Line: 7, Column: 16},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 108, Line: 7, Column: 18},
											},
											Value: &ast.IntegerExpression{
												PositiveLiteral: []byte("0"),
												Value:           new(big.Int),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 110, Line: 7, Column: 20},
													EndPos:   ast.Position{Offset: 110, Line: 7, Column: 20},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 88, Line: 6, Column: 38},
										EndPos:   ast.Position{Offset: 124, Line: 8, Column: 12},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 62, Line: 6, Column: 12},
						},
					},
					PreConditions:  nil,
					PostConditions: nil,
					Execute: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindExecute,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "execute",
								Pos:        ast.Position{Offset: 139, Line: 10, Column: 12},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 165, Line: 11, Column: 16},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 167, Line: 11, Column: 18},
											},
											Value: &ast.BinaryExpression{
												Operation: ast.OperationPlus,
												Left: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 169, Line: 11, Column: 20},
														EndPos:   ast.Position{Offset: 169, Line: 11, Column: 20},
													},
												},
												Right: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 173, Line: 11, Column: 24},
														EndPos:   ast.Position{Offset: 173, Line: 11, Column: 24},
													},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 147, Line: 10, Column: 20},
										EndPos:   ast.Position{Offset: 187, Line: 12, Column: 12},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 139, Line: 10, Column: 12},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 199, Line: 13, Column: 10},
					},
				},
			},
			result.Declarations(),
		)
	})

	t.Run("PreExecutePost", func(t *testing.T) {

		t.Parallel()

		const code = `
          transaction {

              var x: Int

              prepare(signer: AuthAccount) {
                  x = 0
              }

              pre {
                  x == 0
              }

              execute {
                  x = 1 + 1
              }

              post {
                  x == 2
              }
          }
        `
		result, errs := testParseProgram(code)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Fields: []*ast.FieldDeclaration{
						{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindVariable,
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 44, Line: 4, Column: 18},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 47, Line: 4, Column: 21},
									},
								},
								StartPos: ast.Position{Offset: 47, Line: 4, Column: 21},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 40, Line: 4, Column: 14},
								EndPos:   ast.Position{Offset: 49, Line: 4, Column: 23},
							},
						},
					},
					Prepare: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindPrepare,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "prepare",
								Pos:        ast.Position{Offset: 66, Line: 6, Column: 14},
							},
							ParameterList: &ast.ParameterList{
								Parameters: []*ast.Parameter{
									{
										Label: "",
										Identifier: ast.Identifier{
											Identifier: "signer",
											Pos:        ast.Position{Offset: 74, Line: 6, Column: 22},
										},
										TypeAnnotation: &ast.TypeAnnotation{
											IsResource: false,
											Type: &ast.NominalType{
												Identifier: ast.Identifier{
													Identifier: "AuthAccount",
													Pos:        ast.Position{Offset: 82, Line: 6, Column: 30},
												},
											},
											StartPos: ast.Position{Offset: 82, Line: 6, Column: 30},
										},
										StartPos: ast.Position{Offset: 74, Line: 6, Column: 22},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 73, Line: 6, Column: 21},
									EndPos:   ast.Position{Offset: 93, Line: 6, Column: 41},
								},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 115, Line: 7, Column: 18},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 117, Line: 7, Column: 20},
											},
											Value: &ast.IntegerExpression{
												PositiveLiteral: []byte("0"),
												Value:           new(big.Int),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 119, Line: 7, Column: 22},
													EndPos:   ast.Position{Offset: 119, Line: 7, Column: 22},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 95, Line: 6, Column: 43},
										EndPos:   ast.Position{Offset: 135, Line: 8, Column: 14},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 66, Line: 6, Column: 14},
						},
					},
					PreConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 152, Line: 10, Column: 14},
							EndPos:   ast.Position{Offset: 197, Line: 12, Column: 14},
						},
						Conditions: []ast.Condition{
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationEqual,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "x",
											Pos:        ast.Position{Offset: 176, Line: 11, Column: 18},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("0"),
										Value:           new(big.Int),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 181, Line: 11, Column: 23},
											EndPos:   ast.Position{Offset: 181, Line: 11, Column: 23},
										},
									},
								},
							},
						},
					},
					PostConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 283, Line: 18, Column: 14},
							EndPos:   ast.Position{Offset: 329, Line: 20, Column: 14},
						},
						Conditions: []ast.Condition{
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationEqual,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "x",
											Pos:        ast.Position{Offset: 308, Line: 19, Column: 18},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("2"),
										Value:           big.NewInt(2),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 313, Line: 19, Column: 23},
											EndPos:   ast.Position{Offset: 313, Line: 19, Column: 23},
										},
									},
								},
							},
						},
					},
					Execute: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindExecute,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "execute",
								Pos:        ast.Position{Offset: 214, Line: 14, Column: 14},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 242, Line: 15, Column: 18},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 244, Line: 15, Column: 20},
											},
											Value: &ast.BinaryExpression{
												Operation: ast.OperationPlus,
												Left: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 246, Line: 15, Column: 22},
														EndPos:   ast.Position{Offset: 246, Line: 15, Column: 22},
													},
												},
												Right: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 250, Line: 15, Column: 26},
														EndPos:   ast.Position{Offset: 250, Line: 15, Column: 26},
													},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 222, Line: 14, Column: 22},
										EndPos:   ast.Position{Offset: 266, Line: 16, Column: 14},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 214, Line: 14, Column: 14},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 341, Line: 21, Column: 10},
					},
				},
			},
			result.Declarations(),
		)
	})

	t.Run("PrePostExecute", func(t *testing.T) {

		t.Parallel()

		const code = `
          transaction {

            var x: Int

            prepare(signer: AuthAccount) {
                x = 0
            }

            pre {
                x == 0
            }

            post {
                x == 2
            }

            execute {
                x = 1 + 1
            }
          }
        `
		result, errs := testParseProgram(code)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.TransactionDeclaration{
					Fields: []*ast.FieldDeclaration{
						{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindVariable,
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 42, Line: 4, Column: 16},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 45, Line: 4, Column: 19},
									},
								},
								StartPos: ast.Position{Offset: 45, Line: 4, Column: 19},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 38, Line: 4, Column: 12},
								EndPos:   ast.Position{Offset: 47, Line: 4, Column: 21},
							},
						},
					},
					Prepare: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindPrepare,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "prepare",
								Pos:        ast.Position{Offset: 62, Line: 6, Column: 12},
							},
							ParameterList: &ast.ParameterList{
								Parameters: []*ast.Parameter{
									{
										Label: "",
										Identifier: ast.Identifier{
											Identifier: "signer",
											Pos:        ast.Position{Offset: 70, Line: 6, Column: 20},
										},
										TypeAnnotation: &ast.TypeAnnotation{
											IsResource: false,
											Type: &ast.NominalType{
												Identifier: ast.Identifier{
													Identifier: "AuthAccount",
													Pos:        ast.Position{Offset: 78, Line: 6, Column: 28},
												},
											},
											StartPos: ast.Position{Offset: 78, Line: 6, Column: 28},
										},
										StartPos: ast.Position{Offset: 70, Line: 6, Column: 20},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 69, Line: 6, Column: 19},
									EndPos:   ast.Position{Offset: 89, Line: 6, Column: 39},
								},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 109, Line: 7, Column: 16},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 111, Line: 7, Column: 18},
											},
											Value: &ast.IntegerExpression{
												PositiveLiteral: []byte("0"),
												Value:           new(big.Int),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 113, Line: 7, Column: 20},
													EndPos:   ast.Position{Offset: 113, Line: 7, Column: 20},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 91, Line: 6, Column: 41},
										EndPos:   ast.Position{Offset: 127, Line: 8, Column: 12},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 62, Line: 6, Column: 12},
						},
					},
					PreConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 142, Line: 10, Column: 12},
							EndPos:   ast.Position{Offset: 183, Line: 12, Column: 12},
						},
						Conditions: []ast.Condition{
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationEqual,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "x",
											Pos:        ast.Position{Offset: 164, Line: 11, Column: 16},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("0"),
										Value:           new(big.Int),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 169, Line: 11, Column: 21},
											EndPos:   ast.Position{Offset: 169, Line: 11, Column: 21},
										},
									},
								},
							},
						},
					},
					PostConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 198, Line: 14, Column: 12},
							EndPos:   ast.Position{Offset: 240, Line: 16, Column: 12},
						},
						Conditions: []ast.Condition{
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationEqual,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "x",
											Pos:        ast.Position{Offset: 221, Line: 15, Column: 16},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("2"),
										Value:           big.NewInt(2),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 226, Line: 15, Column: 21},
											EndPos:   ast.Position{Offset: 226, Line: 15, Column: 21},
										},
									},
								},
							},
						},
					},
					Execute: &ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindExecute,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "execute",
								Pos:        ast.Position{Offset: 255, Line: 18, Column: 12},
							},
							ReturnTypeAnnotation: nil,
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.AssignmentStatement{
											Target: &ast.IdentifierExpression{
												Identifier: ast.Identifier{
													Identifier: "x",
													Pos:        ast.Position{Offset: 281, Line: 19, Column: 16},
												},
											},
											Transfer: &ast.Transfer{
												Operation: ast.TransferOperationCopy,
												Pos:       ast.Position{Offset: 283, Line: 19, Column: 18},
											},
											Value: &ast.BinaryExpression{
												Operation: ast.OperationPlus,
												Left: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 285, Line: 19, Column: 20},
														EndPos:   ast.Position{Offset: 285, Line: 19, Column: 20},
													},
												},
												Right: &ast.IntegerExpression{
													PositiveLiteral: []byte("1"),
													Value:           big.NewInt(1),
													Base:            10,
													Range: ast.Range{
														StartPos: ast.Position{Offset: 289, Line: 19, Column: 24},
														EndPos:   ast.Position{Offset: 289, Line: 19, Column: 24},
													},
												},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 263, Line: 18, Column: 20},
										EndPos:   ast.Position{Offset: 303, Line: 20, Column: 12},
									},
								},
								PreConditions:  nil,
								PostConditions: nil,
							},
							StartPos: ast.Position{Offset: 255, Line: 18, Column: 12},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 315, Line: 21, Column: 10},
					},
				},
			},
			result.Declarations(),
		)
	})

	t.Run("invalid identifiers instead of special function declarations", func(t *testing.T) {
		t.Parallel()

		code := `
          transaction {
             var x: Int

             uwu(signer: AuthAccount) {}

             pre {
                x > 1
             }

             post {
                x == 2
             }
          }
        `

		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&ExpectedPrepareOrExecuteError{
					GotIdentifier: "uwu",
					Pos:           ast.Position{Offset: 63, Line: 5, Column: 13},
				},
			},
			errs,
		)
	})
}

func TestParseFunctionAndBlock(t *testing.T) {

	t.Parallel()

	result, errs := testParseDeclarations(`
       fun test() { return }
    `)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				ParameterList: &ast.ParameterList{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
						EndPos:   ast.Position{Offset: 17, Line: 2, Column: 16},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.ReturnStatement{
								Range: ast.Range{
									StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
									EndPos:   ast.Position{Offset: 26, Line: 2, Column: 25},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 19, Line: 2, Column: 18},
							EndPos:   ast.Position{Offset: 28, Line: 2, Column: 27},
						},
					},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result,
	)
}

func TestParseFunctionParameterWithoutLabel(t *testing.T) {

	t.Parallel()

	result, errs := testParseDeclarations(`
       fun test(x: Int) { }
    `)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "",
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 20, Line: 2, Column: 19},
									},
								},
								StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
							},
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
						EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
							EndPos:   ast.Position{Offset: 27, Line: 2, Column: 26},
						},
					},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result,
	)
}

func TestParseFunctionParameterWithLabel(t *testing.T) {

	t.Parallel()

	result, errs := testParseDeclarations(`
       fun test(x y: Int) { }
    `)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 12, Line: 2, Column: 11},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "x",
							Identifier: ast.Identifier{
								Identifier: "y",
								Pos:        ast.Position{Offset: 19, Line: 2, Column: 18},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
									},
								},
								StartPos: ast.Position{Offset: 22, Line: 2, Column: 21},
							},
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
						EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 27, Line: 2, Column: 26},
							EndPos:   ast.Position{Offset: 29, Line: 2, Column: 28},
						},
					},
				},
				StartPos: ast.Position{Offset: 8, Line: 2, Column: 7},
			},
		},
		result,
	)
}

func TestParseStructure(t *testing.T) {

	t.Parallel()

	const code = `
        struct Test {
            access(all) var foo: Int

            init(foo: Int) {
                self.foo = foo
            }

            access(all) fun getFoo(): Int {
                return self.foo
            }
        }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							TypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 56, Line: 3, Column: 33},
									},
								},
								StartPos:   ast.Position{Offset: 56, Line: 3, Column: 33},
								IsResource: false,
							},
							DocString: "",
							Identifier: ast.Identifier{
								Identifier: "foo",
								Pos:        ast.Position{Offset: 51, Line: 3, Column: 28},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 35, Line: 3, Column: 12},
								EndPos:   ast.Position{Offset: 58, Line: 3, Column: 35},
							},
							Access:       ast.AccessAll,
							VariableKind: ast.VariableKindVariable,
						},
						&ast.SpecialFunctionDeclaration{
							FunctionDeclaration: &ast.FunctionDeclaration{
								ParameterList: &ast.ParameterList{
									Parameters: []*ast.Parameter{
										{
											TypeAnnotation: &ast.TypeAnnotation{
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Int",
														Pos:        ast.Position{Offset: 83, Line: 5, Column: 22},
													},
												},
												StartPos:   ast.Position{Offset: 83, Line: 5, Column: 22},
												IsResource: false,
											},
											Identifier: ast.Identifier{
												Identifier: "foo",
												Pos:        ast.Position{Offset: 78, Line: 5, Column: 17},
											},
											StartPos: ast.Position{Offset: 78, Line: 5, Column: 17},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 77, Line: 5, Column: 16},
										EndPos:   ast.Position{Offset: 86, Line: 5, Column: 25},
									},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Statements: []ast.Statement{
											&ast.AssignmentStatement{
												Target: &ast.MemberExpression{
													Expression: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "self",
															Pos:        ast.Position{Offset: 106, Line: 6, Column: 16},
														},
													},
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos:        ast.Position{Offset: 111, Line: 6, Column: 21},
													},
													AccessEndPos: ast.Position{Offset: 110, Line: 6, Column: 20},
													Optional:     false,
												},
												Transfer: &ast.Transfer{
													Operation: ast.TransferOperationCopy,
													Pos:       ast.Position{Offset: 115, Line: 6, Column: 25},
												},
												Value: &ast.IdentifierExpression{
													Identifier: ast.Identifier{
														Identifier: "foo",
														Pos:        ast.Position{Offset: 117, Line: 6, Column: 27},
													},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 88, Line: 5, Column: 27},
											EndPos:   ast.Position{Offset: 133, Line: 7, Column: 12},
										},
									},
								},
								Identifier: ast.Identifier{
									Identifier: "init",
									Pos:        ast.Position{Offset: 73, Line: 5, Column: 12},
								},
								StartPos: ast.Position{Offset: 73, Line: 5, Column: 12},
								Access:   ast.AccessNotSpecified,
							},
							Kind: common.DeclarationKindInitializer,
						},
						&ast.FunctionDeclaration{
							ParameterList: &ast.ParameterList{
								Range: ast.Range{
									StartPos: ast.Position{Offset: 170, Line: 9, Column: 34},
									EndPos:   ast.Position{Offset: 171, Line: 9, Column: 35},
								},
							},
							ReturnTypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 174, Line: 9, Column: 38},
									},
								},
								StartPos:   ast.Position{Offset: 174, Line: 9, Column: 38},
								IsResource: false,
							},
							FunctionBlock: &ast.FunctionBlock{
								Block: &ast.Block{
									Statements: []ast.Statement{
										&ast.ReturnStatement{
											Expression: &ast.MemberExpression{
												Expression: &ast.IdentifierExpression{
													Identifier: ast.Identifier{
														Identifier: "self",
														Pos:        ast.Position{Offset: 203, Line: 10, Column: 23},
													},
												},
												Identifier: ast.Identifier{
													Identifier: "foo",
													Pos:        ast.Position{Offset: 208, Line: 10, Column: 28},
												},
												AccessEndPos: ast.Position{Offset: 207, Line: 10, Column: 27},
												Optional:     false,
											},
											Range: ast.Range{
												StartPos: ast.Position{Offset: 196, Line: 10, Column: 16},
												EndPos:   ast.Position{Offset: 210, Line: 10, Column: 30},
											},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 178, Line: 9, Column: 42},
										EndPos:   ast.Position{Offset: 224, Line: 11, Column: 12},
									},
								},
							},
							Identifier: ast.Identifier{
								Identifier: "getFoo",
								Pos:        ast.Position{Offset: 164, Line: 9, Column: 28},
							},
							StartPos: ast.Position{Offset: 148, Line: 9, Column: 12},
							Access:   ast.AccessAll,
							Flags:    0,
						},
					},
				),
				Identifier: ast.Identifier{
					Identifier: "Test",
					Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 234, Line: 12, Column: 8},
				},
				Access:        ast.AccessNotSpecified,
				CompositeKind: common.CompositeKindStructure,
			},
		},
		result.Declarations(),
	)
}

func TestParseStructureWithConformances(t *testing.T) {

	t.Parallel()

	const code = `
        struct Test: Foo, Bar {}
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				Access:        ast.AccessNotSpecified,
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "Test",
					Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Conformances: []*ast.NominalType{
					{
						Identifier: ast.Identifier{
							Identifier: "Foo",
							Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
						},
					},
					{
						Identifier: ast.Identifier{
							Identifier: "Bar",
							Pos:        ast.Position{Offset: 27, Line: 2, Column: 26},
						},
					},
				},
				Members: &ast.Members{},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 32, Line: 2, Column: 31},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseInvalidConformances(t *testing.T) {

	t.Parallel()

	t.Run("no conformances", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations("struct Test: {}")
		AssertEqualWithDiff(t,
			[]error{
				&MissingConformanceError{
					Pos: ast.Position{Offset: 13, Line: 1, Column: 13},
				},
			},
			errs,
		)
	})

	t.Run("missing type after comma", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations("struct Test: I, {}")
		AssertEqualWithDiff(t,
			[]error{
				&MissingTypeAfterSeparatorError{
					Pos:       ast.Position{Offset: 16, Line: 1, Column: 16},
					Separator: lexer.TokenComma,
				},
			},
			errs,
		)
	})
}

func TestParseInvalidMember(t *testing.T) {

	t.Parallel()

	const code = `
        struct Test {
            foo let x: Int
        }
    `

	t.Run("ignore", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			code,
			Config{
				IgnoreLeadingIdentifierEnabled: true,
			},
		)
		require.Empty(t, errs)

	})

	t.Run("report", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			code,
			Config{
				IgnoreLeadingIdentifierEnabled: false,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 35, Line: 3, Column: 12},
				},
			},
			errs,
		)
	})
}

func TestParsePreAndPostConditions(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(n: Int) {
            pre {
                n != 0
                n > 0
            }
            post {
                result == 0
            }
            return 0
        }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "",
							Identifier: ast.Identifier{
								Identifier: "n",
								Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 21, Line: 2, Column: 20},
									},
								},
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							},
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.ReturnStatement{
								Expression: &ast.IntegerExpression{
									PositiveLiteral: []byte("0"),
									Value:           new(big.Int),
									Base:            10,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 185, Line: 10, Column: 19},
										EndPos:   ast.Position{Offset: 185, Line: 10, Column: 19},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 178, Line: 10, Column: 12},
									EndPos:   ast.Position{Offset: 185, Line: 10, Column: 19},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:   ast.Position{Offset: 195, Line: 11, Column: 8},
						},
					},
					PreConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 40, Line: 3, Column: 12},
							EndPos:   ast.Position{Offset: 103, Line: 6, Column: 12},
						},
						Conditions: []ast.Condition{
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationNotEqual,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "n",
											Pos:        ast.Position{Offset: 62, Line: 4, Column: 16},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("0"),
										Value:           new(big.Int),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 67, Line: 4, Column: 21},
											EndPos:   ast.Position{Offset: 67, Line: 4, Column: 21},
										},
									},
								},
							},
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationGreater,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "n",
											Pos:        ast.Position{Offset: 85, Line: 5, Column: 16},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("0"),
										Value:           new(big.Int),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 89, Line: 5, Column: 20},
											EndPos:   ast.Position{Offset: 89, Line: 5, Column: 20},
										},
									},
								},
							},
						},
					},
					PostConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 117, Line: 7, Column: 12},
							EndPos:   ast.Position{Offset: 164, Line: 9, Column: 12},
						},
						Conditions: []ast.Condition{
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationEqual,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "result",
											Pos:        ast.Position{Offset: 140, Line: 8, Column: 16},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("0"),
										Value:           new(big.Int),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 150, Line: 8, Column: 26},
											EndPos:   ast.Position{Offset: 150, Line: 8, Column: 26},
										},
									},
								},
							},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseConditionMessage(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(n: Int) {
            pre {
                n >= 0: "n must be positive"
            }
            return n
        }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "",
							Identifier: ast.Identifier{Identifier: "n",
								Pos: ast.Position{Offset: 18, Line: 2, Column: 17},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 21, Line: 2, Column: 20},
									},
								},
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							},
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.ReturnStatement{
								Expression: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "n",
										Pos:        ast.Position{Offset: 124, Line: 6, Column: 19},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 117, Line: 6, Column: 12},
									EndPos:   ast.Position{Offset: 124, Line: 6, Column: 19},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:   ast.Position{Offset: 134, Line: 7, Column: 8},
						},
					},
					PreConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 40, Line: 3, Column: 12},
							EndPos:   ast.Position{Offset: 103, Line: 5, Column: 12},
						},
						Conditions: []ast.Condition{
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationGreaterEqual,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "n",
											Pos:        ast.Position{Offset: 62, Line: 4, Column: 16},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("0"),
										Value:           new(big.Int),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 67, Line: 4, Column: 21},
											EndPos:   ast.Position{Offset: 67, Line: 4, Column: 21},
										},
									},
								},
								Message: &ast.StringExpression{
									Value: "n must be positive",
									Range: ast.Range{
										StartPos: ast.Position{Offset: 70, Line: 4, Column: 24},
										EndPos:   ast.Position{Offset: 89, Line: 4, Column: 43},
									},
								},
							},
						},
					},
					PostConditions: nil,
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseInvalidEmitConditionNonInvocation(t *testing.T) {

	t.Parallel()

	t.Run("pre-condition", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          fun test(n: Int) {
              pre {
                  emit Foo
              }
          }
        `)

		AssertEqualWithDiff(t,
			[]error{
				&MissingOpeningParenInNominalTypeInvocationError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 91, Line: 5, Column: 14},
							EndPos:   ast.Position{Offset: 91, Line: 5, Column: 14},
						},
						Type: lexer.TokenBraceClose,
					},
				},
				&UnexpectedExpressionStartError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 91, Line: 5, Column: 14},
							EndPos:   ast.Position{Offset: 91, Line: 5, Column: 14},
						},
						Type: lexer.TokenBraceClose,
					},
				},
			},
			errs,
		)
	})

	t.Run("post-condition", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          fun test(n: Int) {
              post {
                  emit Foo
              }
          }
        `)

		AssertEqualWithDiff(t,
			[]error{
				&MissingOpeningParenInNominalTypeInvocationError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 92, Line: 5, Column: 14},
							EndPos:   ast.Position{Offset: 92, Line: 5, Column: 14},
						},
						Type: lexer.TokenBraceClose,
					},
				},
				&UnexpectedExpressionStartError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 92, Line: 5, Column: 14},
							EndPos:   ast.Position{Offset: 92, Line: 5, Column: 14},
						},
						Type: lexer.TokenBraceClose,
					},
				},
			},
			errs,
		)
	})
}

func TestParseEmitAndTestCondition(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(n: Int) {
            pre {
                emit Foo()
                n > 0
            }
            post {
                n > 0
                emit Bar()
            }
            return n
        }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "",
							Identifier: ast.Identifier{Identifier: "n",
								Pos: ast.Position{Offset: 18, Line: 2, Column: 17},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: false,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 21, Line: 2, Column: 20},
									},
								},
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							},
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.ReturnStatement{
								Expression: &ast.IdentifierExpression{
									Identifier: ast.Identifier{
										Identifier: "n",
										Pos:        ast.Position{Offset: 210, Line: 11, Column: 19},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 203, Line: 11, Column: 12},
									EndPos:   ast.Position{Offset: 210, Line: 11, Column: 19},
								},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 26, Line: 2, Column: 25},
							EndPos:   ast.Position{Offset: 220, Line: 12, Column: 8},
						},
					},
					PreConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 40, Line: 3, Column: 12},
							EndPos:   ast.Position{Offset: 107, Line: 6, Column: 12},
						},
						Conditions: []ast.Condition{
							&ast.EmitCondition{
								InvocationExpression: &ast.InvocationExpression{
									InvokedExpression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "Foo",
											Pos:        ast.Position{Offset: 67, Line: 4, Column: 21},
										},
									},
									ArgumentsStartPos: ast.Position{Offset: 70, Line: 4, Column: 24},
									EndPos:            ast.Position{Offset: 71, Line: 4, Column: 25},
								},
								StartPos: ast.Position{Offset: 62, Line: 4, Column: 16},
							},
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationGreater,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "n",
											Pos:        ast.Position{Offset: 89, Line: 5, Column: 16},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("0"),
										Value:           new(big.Int),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 93, Line: 5, Column: 20},
											EndPos:   ast.Position{Offset: 93, Line: 5, Column: 20},
										},
									},
								},
							},
						},
					},
					PostConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 121, Line: 7, Column: 12},
							EndPos:   ast.Position{Offset: 189, Line: 10, Column: 12},
						},
						Conditions: []ast.Condition{
							&ast.TestCondition{
								Test: &ast.BinaryExpression{
									Operation: ast.OperationGreater,
									Left: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "n",
											Pos:        ast.Position{Offset: 144, Line: 8, Column: 16},
										},
									},
									Right: &ast.IntegerExpression{
										PositiveLiteral: []byte("0"),
										Value:           new(big.Int),
										Base:            10,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 148, Line: 8, Column: 20},
											EndPos:   ast.Position{Offset: 148, Line: 8, Column: 20},
										},
									},
								},
							},
							&ast.EmitCondition{
								InvocationExpression: &ast.InvocationExpression{
									InvokedExpression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "Bar",
											Pos:        ast.Position{Offset: 171, Line: 9, Column: 21},
										},
									},
									ArgumentsStartPos: ast.Position{Offset: 174, Line: 9, Column: 24},
									EndPos:            ast.Position{Offset: 175, Line: 9, Column: 25},
								},
								StartPos: ast.Position{Offset: 166, Line: 9, Column: 16},
							},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseInterface(t *testing.T) {

	t.Parallel()

	for _, kind := range common.InstantiableCompositeKindsWithFieldsAndFunctions {

		code := fmt.Sprintf(`
            %s interface Test {
                foo: Int

                init(foo: Int)

                fun getFoo(): Int
            }
       `, kind.Keyword())
		actual, err := testParseProgram(code)

		require.NoError(t, err)

		// only compare AST for one kind: structs

		if kind != common.CompositeKindStructure {
			continue
		}

		test := &ast.InterfaceDeclaration{
			Access:        ast.AccessNotSpecified,
			CompositeKind: common.CompositeKindStructure,
			Identifier: ast.Identifier{
				Identifier: "Test",
				Pos:        ast.Position{Offset: 30, Line: 2, Column: 29},
			},
			Members: ast.NewUnmeteredMembers(
				[]ast.Declaration{
					&ast.FieldDeclaration{
						Access:       ast.AccessNotSpecified,
						VariableKind: ast.VariableKindNotSpecified,
						Identifier: ast.Identifier{
							Identifier: "foo",
							Pos:        ast.Position{Offset: 53, Line: 3, Column: 16},
						},
						TypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Offset: 58, Line: 3, Column: 21},
								},
							},
							StartPos: ast.Position{Offset: 58, Line: 3, Column: 21},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 53, Line: 3, Column: 16},
							EndPos:   ast.Position{Offset: 60, Line: 3, Column: 23},
						},
					},
					&ast.SpecialFunctionDeclaration{
						Kind: common.DeclarationKindInitializer,
						FunctionDeclaration: &ast.FunctionDeclaration{
							Access: ast.AccessNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "init",
								Pos:        ast.Position{Offset: 79, Line: 5, Column: 16},
							},
							ParameterList: &ast.ParameterList{
								Parameters: []*ast.Parameter{
									{
										Label: "",
										Identifier: ast.Identifier{
											Identifier: "foo",
											Pos:        ast.Position{Offset: 84, Line: 5, Column: 21},
										},
										TypeAnnotation: &ast.TypeAnnotation{
											IsResource: false,
											Type: &ast.NominalType{
												Identifier: ast.Identifier{
													Identifier: "Int",
													Pos:        ast.Position{Offset: 89, Line: 5, Column: 26},
												},
											},
											StartPos: ast.Position{Offset: 89, Line: 5, Column: 26},
										},
										StartPos: ast.Position{Offset: 84, Line: 5, Column: 21},
									},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 83, Line: 5, Column: 20},
									EndPos:   ast.Position{Offset: 92, Line: 5, Column: 29},
								},
							},
							FunctionBlock: nil,
							StartPos:      ast.Position{Offset: 79, Line: 5, Column: 16},
						},
					},
					&ast.FunctionDeclaration{
						Access: ast.AccessNotSpecified,
						Identifier: ast.Identifier{
							Identifier: "getFoo",
							Pos:        ast.Position{Offset: 115, Line: 7, Column: 20},
						},
						ParameterList: &ast.ParameterList{
							Range: ast.Range{
								StartPos: ast.Position{Offset: 121, Line: 7, Column: 26},
								EndPos:   ast.Position{Offset: 122, Line: 7, Column: 27},
							},
						},
						ReturnTypeAnnotation: &ast.TypeAnnotation{
							IsResource: false,
							Type: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "Int",
									Pos:        ast.Position{Offset: 125, Line: 7, Column: 30},
								},
							},
							StartPos: ast.Position{Offset: 125, Line: 7, Column: 30},
						},
						FunctionBlock: nil,
						StartPos:      ast.Position{Offset: 111, Line: 7, Column: 16},
					},
				},
			),
			Range: ast.Range{
				StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
				EndPos:   ast.Position{Offset: 141, Line: 8, Column: 12},
			},
		}

		AssertEqualWithDiff(t,
			[]ast.Declaration{test},
			actual.Declarations(),
		)
	}
}

func TestParsePragmaNoArguments(t *testing.T) {

	t.Parallel()

	t.Run("identifier", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`#pedantic`)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.PragmaDeclaration{
					Expression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "pedantic",
							Pos:        ast.Position{Offset: 1, Line: 1, Column: 1},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
			},
			result,
		)
	})

	t.Run("with purity", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("view #foo")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidViewModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			"static #foo",
			Config{
				StaticModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("static #foo")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			"native #foo",
			Config{
				NativeModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("native #foo")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
							EndPos:   ast.Position{Offset: 5, Line: 1, Column: 5},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})
}

func TestParsePragmaArguments(t *testing.T) {

	t.Parallel()

	const code = `#version("1.0")`
	actual, err := testParseProgram(code)
	require.NoError(t, err)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.PragmaDeclaration{
				Expression: &ast.InvocationExpression{
					InvokedExpression: &ast.IdentifierExpression{
						Identifier: ast.Identifier{
							Identifier: "version",
							Pos:        ast.Position{Offset: 1, Line: 1, Column: 1},
						},
					},
					Arguments: ast.Arguments{
						{
							Expression: &ast.StringExpression{
								Value: "1.0",
								Range: ast.Range{
									StartPos: ast.Position{Offset: 9, Line: 1, Column: 9},
									EndPos:   ast.Position{Offset: 13, Line: 1, Column: 13},
								},
							},
							TrailingSeparatorPos: ast.Position{Offset: 14, Line: 1, Column: 14},
						},
					},
					ArgumentsStartPos: ast.Position{Offset: 8, Line: 1, Column: 8},
					EndPos:            ast.Position{Offset: 14, Line: 1, Column: 14},
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					EndPos:   ast.Position{Offset: 14, Line: 1, Column: 14},
				},
			},
		},
		actual.Declarations(),
	)
}

func TestParseImportWithString(t *testing.T) {

	t.Parallel()

	const code = `
        import "test.cdc"
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.ImportDeclaration{
				Imports:  nil,
				Location: common.StringLocation("test.cdc"),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
				},
				LocationPos: ast.Position{Offset: 16, Line: 2, Column: 15},
			},
		},
		result.Declarations(),
	)
}

func TestParseImportWithAddress(t *testing.T) {

	t.Parallel()

	const code = `
        import 0x1234
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.ImportDeclaration{
				Imports: nil,
				Location: common.AddressLocation{
					Address: common.MustBytesToAddress([]byte{0x12, 0x34}),
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 21, Line: 2, Column: 20},
				},
				LocationPos: ast.Position{Offset: 16, Line: 2, Column: 15},
			},
		},
		result.Declarations(),
	)
}

func TestParseImportWithIdentifiers(t *testing.T) {

	t.Parallel()

	const code = `
        import A, b from 0x1
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.ImportDeclaration{
				Imports: []ast.Import{
					{
						Identifier: ast.Identifier{
							Identifier: "A",
							Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
						},
					},
					{
						Identifier: ast.Identifier{
							Identifier: "b",
							Pos:        ast.Position{Offset: 19, Line: 2, Column: 18},
						},
					},
				},
				Location: common.AddressLocation{
					Address: common.MustBytesToAddress([]byte{0x1}),
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 28, Line: 2, Column: 27},
				},
				LocationPos: ast.Position{Offset: 26, Line: 2, Column: 25},
			},
		},
		result.Declarations(),
	)
}

func TestParseFieldWithFromIdentifier(t *testing.T) {

	t.Parallel()

	const code = `
      struct S {
          let from: String
      }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				Access:        ast.AccessNotSpecified,
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "S",
					Pos:        ast.Position{Offset: 14, Line: 2, Column: 13},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindConstant,
							Identifier: ast.Identifier{
								Identifier: "from",
								Pos:        ast.Position{Offset: 32, Line: 3, Column: 14},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "String",
										Pos:        ast.Position{Offset: 38, Line: 3, Column: 20},
									},
								},
								StartPos: ast.Position{Offset: 38, Line: 3, Column: 20},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 28, Line: 3, Column: 10},
								EndPos:   ast.Position{Offset: 43, Line: 3, Column: 25},
							},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
					EndPos:   ast.Position{Offset: 51, Line: 4, Column: 6},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseFunctionWithFromIdentifier(t *testing.T) {

	t.Parallel()

	const code = `
        fun send(from: String, to: String) {}
    `
	_, errs := testParseProgram(code)
	require.Empty(t, errs)
}

func TestParseImportWithFromIdentifier(t *testing.T) {

	t.Parallel()

	const code = `
        import from from 0x1
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.ImportDeclaration{
				Imports: []ast.Import{
					{
						Identifier: ast.Identifier{
							Identifier: "from",
							Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
						},
					},
				},
				Location: common.AddressLocation{
					Address: common.MustBytesToAddress([]byte{0x1}),
				},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 28, Line: 2, Column: 27},
				},
				LocationPos: ast.Position{Offset: 26, Line: 2, Column: 25},
			},
		},
		result.Declarations(),
	)
}

func TestParseInvalidImportWithPurity(t *testing.T) {

	t.Parallel()

	const code = `
        view import x from 0x1
    `
	_, errs := testParseDeclarations(code)

	AssertEqualWithDiff(t,
		[]error{
			&InvalidViewModifierError{
				Pos:             ast.Position{Offset: 9, Line: 2, Column: 8},
				DeclarationKind: common.DeclarationKindImport,
			},
		},
		errs,
	)
}

func TestParseInvalidDefaultArgument(t *testing.T) {

	t.Parallel()

	t.Run("function declaration ", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) fun foo ( a : Int = 3) { } ")

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedDefaultArgumentError{
					Pos: ast.Position{Line: 1, Column: 31, Offset: 31},
				},
			},
			errs,
		)
	})

	t.Run("function expression ", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" let foo = fun ( a : Int = 3) { } ")

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedDefaultArgumentError{
					Pos: ast.Position{Line: 1, Column: 25, Offset: 25},
				},
			},
			errs,
		)
	})
}

func TestParseInvalidEventWithPurity(t *testing.T) {

	t.Parallel()

	const code = `
        view event Foo()
    `
	_, errs := testParseDeclarations(code)

	AssertEqualWithDiff(t,
		[]error{
			&InvalidViewModifierError{
				Pos:             ast.Position{Offset: 9, Line: 2, Column: 8},
				DeclarationKind: common.DeclarationKindEvent,
			},
		},
		errs,
	)
}

func TestParseInvalidCompositeWithPurity(t *testing.T) {

	t.Parallel()

	const code = `
        view struct S {}
    `
	_, errs := testParseDeclarations(code)

	AssertEqualWithDiff(t,
		[]error{
			&InvalidViewModifierError{
				Pos:             ast.Position{Offset: 9, Line: 2, Column: 8},
				DeclarationKind: common.DeclarationKindStructure,
			},
		},
		errs,
	)
}

func TestParseInvalidTransactionWithPurity(t *testing.T) {

	t.Parallel()

	const code = `
        view transaction {}
    `
	_, errs := testParseDeclarations(code)

	AssertEqualWithDiff(t,
		[]error{
			&InvalidViewModifierError{
				Pos:             ast.Position{Offset: 9, Line: 2, Column: 8},
				DeclarationKind: common.DeclarationKindTransaction,
			},
		},
		errs,
	)
}

func TestParseSemicolonsBetweenDeclarations(t *testing.T) {

	t.Parallel()

	const code = `
        import from from 0x0;
        fun foo() {};
    `
	_, errs := testParseProgram(code)
	require.Empty(t, errs)
}

func TestParseResource(t *testing.T) {

	t.Parallel()

	const code = `
        resource Test {}
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				Access:        ast.AccessNotSpecified,
				CompositeKind: common.CompositeKindResource,
				Identifier: ast.Identifier{
					Identifier: "Test",
					Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
				},
				Members: &ast.Members{},
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseEventDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
        event Transfer(to: Address, from: Address)
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				Access:        ast.AccessNotSpecified,
				CompositeKind: common.CompositeKindEvent,
				Identifier: ast.Identifier{
					Identifier: "Transfer",
					Pos:        ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.SpecialFunctionDeclaration{
							Kind: common.DeclarationKindInitializer,
							FunctionDeclaration: &ast.FunctionDeclaration{
								Access: ast.AccessNotSpecified,
								ParameterList: &ast.ParameterList{
									Parameters: []*ast.Parameter{
										{
											Label: "",
											Identifier: ast.Identifier{
												Identifier: "to",
												Pos:        ast.Position{Offset: 24, Line: 2, Column: 23},
											},
											TypeAnnotation: &ast.TypeAnnotation{
												IsResource: false,
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Address",
														Pos:        ast.Position{Offset: 28, Line: 2, Column: 27},
													},
												},
												StartPos: ast.Position{Offset: 28, Line: 2, Column: 27},
											},
											StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
										},
										{
											Label: "",
											Identifier: ast.Identifier{
												Identifier: "from",
												Pos:        ast.Position{Offset: 37, Line: 2, Column: 36},
											},
											TypeAnnotation: &ast.TypeAnnotation{
												IsResource: false,
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Address",
														Pos:        ast.Position{Offset: 43, Line: 2, Column: 42},
													},
												},
												StartPos: ast.Position{Offset: 43, Line: 2, Column: 42},
											},
											StartPos: ast.Position{Offset: 37, Line: 2, Column: 36},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
										EndPos:   ast.Position{Offset: 50, Line: 2, Column: 49},
									},
								},
								StartPos: ast.Position{Offset: 23, Line: 2, Column: 22},
							},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 50, Line: 2, Column: 49},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseEventEmitStatement(t *testing.T) {

	t.Parallel()

	const code = `
      fun test() {
        emit Transfer(to: 1, from: 2)
      }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 11, Line: 2, Column: 10},
				},
				ParameterList: &ast.ParameterList{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 15, Line: 2, Column: 14},
						EndPos:   ast.Position{Offset: 16, Line: 2, Column: 15},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Statements: []ast.Statement{
							&ast.EmitStatement{
								InvocationExpression: &ast.InvocationExpression{
									InvokedExpression: &ast.IdentifierExpression{
										Identifier: ast.Identifier{
											Identifier: "Transfer",
											Pos:        ast.Position{Offset: 33, Line: 3, Column: 13},
										},
									},
									Arguments: ast.Arguments{
										{
											Label:         "to",
											LabelStartPos: &ast.Position{Offset: 42, Line: 3, Column: 22},
											LabelEndPos:   &ast.Position{Offset: 44, Line: 3, Column: 24},
											Expression: &ast.IntegerExpression{
												PositiveLiteral: []byte("1"),
												Value:           big.NewInt(1),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 46, Line: 3, Column: 26},
													EndPos:   ast.Position{Offset: 46, Line: 3, Column: 26},
												},
											},
											TrailingSeparatorPos: ast.Position{Offset: 47, Line: 3, Column: 27},
										},
										{
											Label:         "from",
											LabelStartPos: &ast.Position{Offset: 49, Line: 3, Column: 29},
											LabelEndPos:   &ast.Position{Offset: 53, Line: 3, Column: 33},
											Expression: &ast.IntegerExpression{
												PositiveLiteral: []byte("2"),
												Value:           big.NewInt(2),
												Base:            10,
												Range: ast.Range{
													StartPos: ast.Position{Offset: 55, Line: 3, Column: 35},
													EndPos:   ast.Position{Offset: 55, Line: 3, Column: 35},
												},
											},
											TrailingSeparatorPos: ast.Position{Offset: 56, Line: 3, Column: 36},
										},
									},
									ArgumentsStartPos: ast.Position{Offset: 41, Line: 3, Column: 21},
									EndPos:            ast.Position{Offset: 56, Line: 3, Column: 36},
								},
								StartPos: ast.Position{Offset: 28, Line: 3, Column: 8},
							},
						},
						Range: ast.Range{
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
							EndPos:   ast.Position{Offset: 64, Line: 4, Column: 6},
						},
					},
				},
				StartPos: ast.Position{Offset: 7, Line: 2, Column: 6},
			},
		},
		result.Declarations(),
	)
}

func TestParseResourceReturnType(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(): @X {}
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				ParameterList: &ast.ParameterList{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
					},
				},
				ReturnTypeAnnotation: &ast.TypeAnnotation{
					IsResource: true,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "X",
							Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
						},
					},
					StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
							EndPos:   ast.Position{Offset: 25, Line: 2, Column: 24},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseMovingVariableDeclaration(t *testing.T) {

	t.Parallel()

	const code = `
        let x <- y
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
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "y",
						Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
					},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationMove,
					Pos:       ast.Position{Offset: 15, Line: 2, Column: 14},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseResourceParameterType(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(x: @X) {}
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				ParameterList: &ast.ParameterList{
					Parameters: []*ast.Parameter{
						{
							Label: "",
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								IsResource: true,
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "X",
										Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
									},
								},
								StartPos: ast.Position{Offset: 21, Line: 2, Column: 20},
							},
							StartPos: ast.Position{Offset: 18, Line: 2, Column: 17},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
						EndPos:   ast.Position{Offset: 23, Line: 2, Column: 22},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 25, Line: 2, Column: 24},
							EndPos:   ast.Position{Offset: 26, Line: 2, Column: 25},
						},
					},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseMovingVariableDeclarationWithTypeAnnotation(t *testing.T) {

	t.Parallel()

	const code = `
        let x: @R <- y
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
					Pos:        ast.Position{Offset: 13, Line: 2, Column: 12},
				},
				TypeAnnotation: &ast.TypeAnnotation{
					IsResource: true,
					Type: &ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: "R",
							Pos:        ast.Position{Offset: 17, Line: 2, Column: 16},
						},
					},
					StartPos: ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: "y",
						Pos:        ast.Position{Offset: 22, Line: 2, Column: 21},
					},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationMove,
					Pos:       ast.Position{Offset: 19, Line: 2, Column: 18},
				},
				StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
			},
		},
		result.Declarations(),
	)
}

func TestParseFieldDeclarationWithMoveTypeAnnotation(t *testing.T) {

	t.Parallel()

	const code = `
        struct X { x: @R }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				Access:        ast.AccessNotSpecified,
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "X",
					Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindNotSpecified,
							Identifier: ast.Identifier{
								Identifier: "x",
								Pos:        ast.Position{Offset: 20, Line: 2, Column: 19},
							},
							TypeAnnotation: &ast.TypeAnnotation{
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
								StartPos: ast.Position{Offset: 20, Line: 2, Column: 19},
								EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
							},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 26, Line: 2, Column: 25},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseDestructor(t *testing.T) {

	t.Parallel()

	const code = `
        resource Test {
            destroy() {}
        }
    `
	_, errs := testParseDeclarations(code)

	AssertEqualWithDiff(t,
		[]error{
			&CustomDestructorError{
				Pos: ast.Position{Offset: 37, Line: 3, Column: 12},
				DestructorRange: ast.Range{
					StartPos: ast.Position{Offset: 37, Line: 3, Column: 12},
					EndPos:   ast.Position{Offset: 48, Line: 3, Column: 23},
				},
			},
		},
		errs,
	)

	var customDestructorError *CustomDestructorError
	require.ErrorAs(t, errs[0], &customDestructorError)

	fixes := customDestructorError.SuggestFixes(code)
	require.Equal(t,
		[]errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Remove the deprecated custom destructor",
				TextEdits: []ast.TextEdit{
					{
						Replacement: "",
						Range: ast.Range{
							StartPos: ast.Position{Offset: 37, Line: 3, Column: 12},
							EndPos:   ast.Position{Offset: 48, Line: 3, Column: 23},
						},
					},
				},
			},
		},
		fixes,
	)

	const expected = `
        resource Test {
            
        }
    `
	assert.Equal(t,
		expected,
		fixes[0].TextEdits[0].ApplyTo(code),
	)

	assert.NotEmpty(t, customDestructorError.MigrationNote())
}

func TestParseCompositeDeclarationWithSemicolonSeparatedMembers(t *testing.T) {

	t.Parallel()

	const code = `
        struct Kitty { let id: Int ; init(id: Int) { self.id = id } }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.CompositeDeclaration{
				Access:        ast.AccessNotSpecified,
				CompositeKind: common.CompositeKindStructure,
				Identifier: ast.Identifier{
					Identifier: "Kitty",
					Pos:        ast.Position{Offset: 16, Line: 2, Column: 15},
				},
				Members: ast.NewUnmeteredMembers(
					[]ast.Declaration{
						&ast.FieldDeclaration{
							Access:       ast.AccessNotSpecified,
							VariableKind: ast.VariableKindConstant,
							Identifier: ast.Identifier{
								Identifier: "id",
								Pos:        ast.Position{Offset: 28, Line: 2, Column: 27},
							},
							TypeAnnotation: &ast.TypeAnnotation{
								Type: &ast.NominalType{
									Identifier: ast.Identifier{
										Identifier: "Int",
										Pos:        ast.Position{Offset: 32, Line: 2, Column: 31},
									},
								},
								StartPos: ast.Position{Offset: 32, Line: 2, Column: 31},
							},
							Range: ast.Range{
								StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
								EndPos:   ast.Position{Offset: 34, Line: 2, Column: 33},
							},
						},
						&ast.SpecialFunctionDeclaration{
							Kind: common.DeclarationKindInitializer,
							FunctionDeclaration: &ast.FunctionDeclaration{
								Access: ast.AccessNotSpecified,
								Identifier: ast.Identifier{
									Identifier: "init",
									Pos:        ast.Position{Offset: 38, Line: 2, Column: 37},
								},
								ParameterList: &ast.ParameterList{
									Parameters: []*ast.Parameter{
										{
											Identifier: ast.Identifier{
												Identifier: "id",
												Pos:        ast.Position{Offset: 43, Line: 2, Column: 42},
											},
											TypeAnnotation: &ast.TypeAnnotation{
												Type: &ast.NominalType{
													Identifier: ast.Identifier{
														Identifier: "Int",
														Pos:        ast.Position{Offset: 47, Line: 2, Column: 46},
													},
												},
												StartPos: ast.Position{Offset: 47, Line: 2, Column: 46},
											},
											StartPos: ast.Position{Offset: 43, Line: 2, Column: 42},
										},
									},
									Range: ast.Range{
										StartPos: ast.Position{Offset: 42, Line: 2, Column: 41},
										EndPos:   ast.Position{Offset: 50, Line: 2, Column: 49},
									},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Statements: []ast.Statement{
											&ast.AssignmentStatement{
												Target: &ast.MemberExpression{
													Expression: &ast.IdentifierExpression{
														Identifier: ast.Identifier{
															Identifier: "self",
															Pos:        ast.Position{Offset: 54, Line: 2, Column: 53},
														},
													},
													AccessEndPos: ast.Position{Offset: 58, Line: 2, Column: 57},
													Identifier: ast.Identifier{
														Identifier: "id",
														Pos:        ast.Position{Offset: 59, Line: 2, Column: 58},
													},
												},
												Transfer: &ast.Transfer{
													Operation: ast.TransferOperationCopy,
													Pos:       ast.Position{Offset: 62, Line: 2, Column: 61},
												},
												Value: &ast.IdentifierExpression{
													Identifier: ast.Identifier{
														Identifier: "id",
														Pos:        ast.Position{Offset: 64, Line: 2, Column: 63},
													},
												},
											},
										},
										Range: ast.Range{
											StartPos: ast.Position{Offset: 52, Line: 2, Column: 51},
											EndPos:   ast.Position{Offset: 67, Line: 2, Column: 66},
										},
									},
								},
								StartPos: ast.Position{Offset: 38, Line: 2, Column: 37},
							},
						},
					},
				),
				Range: ast.Range{
					StartPos: ast.Position{Offset: 9, Line: 2, Column: 8},
					EndPos:   ast.Position{Offset: 69, Line: 2, Column: 68},
				},
			},
		},
		result.Declarations(),
	)
}

func TestParseInvalidCompositeFunctionNames(t *testing.T) {

	t.Parallel()

	interfacePossibilities := []bool{true, false}

	for _, kind := range common.CompositeKindsWithFieldsAndFunctions {
		for _, isInterface := range interfacePossibilities {

			interfaceKeyword := ""
			if isInterface {
				interfaceKeyword = "interface"
			}

			var baseType = ""

			if kind == common.CompositeKindAttachment {
				if isInterface {
					continue
				}
				baseType = "for AnyStruct"
			}

			body := "{}"
			if isInterface {
				body = ""
			}

			testName := fmt.Sprintf("%s_%s", kind.Keyword(), interfaceKeyword)

			t.Run(testName, func(t *testing.T) {

				_, err := testParseProgram(
					fmt.Sprintf(
						`
                          %[1]s %[2]s Test %[4]s {
                              fun init() %[3]s
                          }
                        `,
						kind.Keyword(),
						interfaceKeyword,
						body,
						baseType,
					),
				)

				errs, ok := err.(Error)
				assert.True(t, ok, "Parser error does not conform to parser.Error")
				syntaxErr := errs.Errors[0].(*SyntaxError)

				AssertEqualWithDiff(
					t,
					"expected identifier after start of function declaration, got keyword init",
					syntaxErr.Message,
				)
			})
		}
	}
}

func TestParseAccessModifiers(t *testing.T) {

	t.Parallel()

	type declaration struct {
		name, code string
	}

	declarations := []declaration{
		{"variable", "%s var test = 1"},
		{"constant", "%s let test = 1"},
		{"function", "%s fun test() {}"},
	}

	for _, compositeKind := range common.AllCompositeKinds {

		for _, isInterface := range []bool{true, false} {

			if !compositeKind.SupportsInterfaces() && isInterface {
				continue
			}

			interfaceKeyword := ""
			if isInterface {
				interfaceKeyword = "interface"
			}

			formatName := func(name string) string {
				return fmt.Sprintf(
					"%s %s %s",
					compositeKind.Keyword(),
					interfaceKeyword,
					name,
				)
			}

			var baseType = ""

			if compositeKind == common.CompositeKindAttachment {
				baseType = "for AnyStruct"
			}

			formatCode := func(format string) string {
				return fmt.Sprintf(format, compositeKind.Keyword(), interfaceKeyword, baseType)
			}

			if compositeKind == common.CompositeKindEvent {
				declarations = append(declarations,
					declaration{
						formatName("itself"),
						formatCode("%%s %s %s Test()%s"),
					},
				)
			} else {
				declarations = append(declarations,
					declaration{
						formatName("itself"),
						formatCode("%%s %s %s Test %s {}"),
					},
					declaration{
						formatName("field"),
						formatCode("%s %s Test %s { %%s let test: Int ; init() { self.test = 1 } }"),
					},
					declaration{
						formatName("function"),
						formatCode("%s %s Test %s { %%s fun test() {} }"),
					},
				)
			}
		}
	}

	for _, declaration := range declarations {
		for _, access := range ast.BasicAccesses {
			testName := fmt.Sprintf("%s/%s", declaration.name, access)
			t.Run(testName, func(t *testing.T) {
				program := fmt.Sprintf(declaration.code, access.Keyword())
				_, errs := testParseProgram(program)

				require.Empty(t, errs)
			})
		}
	}
}

func TestParsePreconditionWithUnaryNegation(t *testing.T) {

	t.Parallel()

	const code = `
     fun test() {
          pre {
              true: "one"
              !false: "two"
          }
      }
    `
	result, errs := testParseProgram(code)
	require.Empty(t, errs)

	AssertEqualWithDiff(t,
		[]ast.Declaration{
			&ast.FunctionDeclaration{
				Access: ast.AccessNotSpecified,
				Identifier: ast.Identifier{
					Identifier: "test",
					Pos:        ast.Position{Offset: 10, Line: 2, Column: 9},
				},
				ParameterList: &ast.ParameterList{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 14, Line: 2, Column: 13},
						EndPos:   ast.Position{Offset: 15, Line: 2, Column: 14},
					},
				},
				FunctionBlock: &ast.FunctionBlock{
					Block: &ast.Block{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 17, Line: 2, Column: 16},
							EndPos:   ast.Position{Offset: 107, Line: 7, Column: 6},
						},
					},
					PreConditions: &ast.Conditions{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 29, Line: 3, Column: 10},
							EndPos:   ast.Position{Offset: 99, Line: 6, Column: 10},
						},
						Conditions: []ast.Condition{
							&ast.TestCondition{
								Test: &ast.BoolExpression{
									Value: true,
									Range: ast.Range{
										StartPos: ast.Position{Offset: 49, Line: 4, Column: 14},
										EndPos:   ast.Position{Offset: 52, Line: 4, Column: 17},
									},
								},
								Message: &ast.StringExpression{
									Value: "one",
									Range: ast.Range{
										StartPos: ast.Position{Offset: 55, Line: 4, Column: 20},
										EndPos:   ast.Position{Offset: 59, Line: 4, Column: 24},
									},
								},
							},
							&ast.TestCondition{
								Test: &ast.UnaryExpression{
									Operation: ast.OperationNegate,
									Expression: &ast.BoolExpression{
										Value: false,
										Range: ast.Range{
											StartPos: ast.Position{Offset: 76, Line: 5, Column: 15},
											EndPos:   ast.Position{Offset: 80, Line: 5, Column: 19},
										},
									},
									StartPos: ast.Position{Offset: 75, Line: 5, Column: 14},
								},
								Message: &ast.StringExpression{
									Value: "two",
									Range: ast.Range{
										StartPos: ast.Position{Offset: 83, Line: 5, Column: 22},
										EndPos:   ast.Position{Offset: 87, Line: 5, Column: 26},
									},
								},
							},
						},
					},
				},
				StartPos: ast.Position{Offset: 6, Line: 2, Column: 5},
			},
		},
		result.Declarations(),
	)
}

func TestParseInvalidAccessModifiers(t *testing.T) {

	t.Parallel()

	t.Run("pragma", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("access(all) #test")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidAccessModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations("access(all) transaction {}")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidAccessModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindTransaction,
				},
			},
			errs,
		)
	})

	t.Run("variable", func(t *testing.T) {

		t.Parallel()

		const code = "access(all) access(self) let x = 1"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&DuplicateAccessModifierError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
						EndPos:   ast.Position{Offset: 23, Line: 1, Column: 23},
					},
				},
			},
			errs,
		)

		var duplicateAccessError *DuplicateAccessModifierError
		require.ErrorAs(t, errs[0], &duplicateAccessError)

		fixes := duplicateAccessError.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Remove duplicate access modifier",
					TextEdits: []ast.TextEdit{
						{
							Replacement: "",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
								EndPos:   ast.Position{Offset: 23, Line: 1, Column: 23},
							},
						},
					},
				},
			},
			fixes,
		)

		const expected = "access(all)  let x = 1"
		assert.Equal(t,
			expected,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})
}

func TestParseInvalidImportWithModifier(t *testing.T) {

	t.Parallel()

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			`
                static import x from 0x1
            `,
			Config{
				StaticModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 17, Line: 2, Column: 16},
					DeclarationKind: common.DeclarationKindImport,
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            static import x from 0x1
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			`
                native import x from 0x1
            `,
			Config{
				NativeModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 17, Line: 2, Column: 16},
					DeclarationKind: common.DeclarationKindImport,
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            native import x from 0x1
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})
}

func TestParseInvalidEventWithModifier(t *testing.T) {

	t.Parallel()

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			`
                static event Foo()
            `,
			Config{
				StaticModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 17, Line: 2, Column: 16},
					DeclarationKind: common.DeclarationKindEvent,
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            static event Foo()
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			`
                native event Foo()
            `,
			Config{
				NativeModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 17, Line: 2, Column: 16},
					DeclarationKind: common.DeclarationKindEvent,
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            native event Foo()
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

}

func TestParseCompositeWithModifier(t *testing.T) {

	t.Parallel()

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			`
                static struct Foo {}
            `,
			Config{
				StaticModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 17, Line: 2, Column: 16},
					DeclarationKind: common.DeclarationKindStructure,
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            static struct Foo {}
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			`
                native struct Foo {}
            `,
			Config{
				NativeModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 17, Line: 2, Column: 16},
					DeclarationKind: common.DeclarationKindStructure,
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            native struct Foo()
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})
}

func TestParseTransactionWithModifier(t *testing.T) {

	t.Parallel()

	t.Run("static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			`
                static transaction {}
            `,
			Config{
				StaticModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 17, Line: 2, Column: 16},
					DeclarationKind: common.DeclarationKindTransaction,
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            static transaction {}
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarationsWithConfig(
			`
                native transaction {}
            `,
			Config{
				NativeModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 17, Line: 2, Column: 16},
					DeclarationKind: common.DeclarationKindTransaction,
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            native transaction {}
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
							EndPos:   ast.Position{Offset: 18, Line: 2, Column: 17},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})
}

func TestParseNestedPragma(t *testing.T) {

	t.Parallel()

	parse := func(input string, config Config) (ast.Declaration, []error) {
		return Parse(
			nil,
			[]byte(input),
			func(p *parser) (ast.Declaration, error) {
				return parseMemberOrNestedDeclaration(
					p,
					"",
				)
			},
			config,
		)
	}

	t.Run("native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"native #foo",
			Config{
				NativeModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("native #pragma", Config{})

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message:   "unexpected token: identifier",
					Secondary: "remove the identifier before the pragma declaration",
					Pos:       ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"static #pragma",
			Config{
				StaticModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("static, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"static #pragma",
			Config{},
		)

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message:   "unexpected token: identifier",
					Secondary: "remove the identifier before the pragma declaration",
					Pos:       ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("static native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"static native #pragma",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 7, Line: 1, Column: 7},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("static native #pragma", Config{})

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 0, Line: 1, Column: 0},
				},
			},
			errs,
		)
	})

	t.Run("native static, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"native static #pragma",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "invalid `static` modifier after `native` modifier",
					Pos:     ast.Position{Offset: 7, Line: 1, Column: 7},
				},
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 7, Line: 1, Column: 7},
					DeclarationKind: common.DeclarationKindPragma,
				},
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("access(all)", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("access(all) #pragma", Config{})

		AssertEqualWithDiff(t,
			[]error{
				&InvalidAccessModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("access(all) static native, enabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse(
			"access(all) static native #pragma",
			Config{
				StaticModifierEnabled: true,
				NativeModifierEnabled: true,
			},
		)
		AssertEqualWithDiff(t,
			[]error{
				&InvalidAccessModifierError{
					Pos:             ast.Position{Offset: 0, Line: 1, Column: 0},
					DeclarationKind: common.DeclarationKindPragma,
				},
				&InvalidStaticModifierError{
					Pos:             ast.Position{Offset: 12, Line: 1, Column: 12},
					DeclarationKind: common.DeclarationKindPragma,
				},
				&InvalidNativeModifierError{
					Pos:             ast.Position{Offset: 19, Line: 1, Column: 19},
					DeclarationKind: common.DeclarationKindPragma,
				},
			},
			errs,
		)
	})

	t.Run("access(all) static native, disabled", func(t *testing.T) {

		t.Parallel()

		_, errs := parse("access(all) static native #pragma", Config{})

		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "unexpected identifier",
					Pos:     ast.Position{Offset: 12, Line: 1, Column: 12},
				},
			},
			errs,
		)
	})

}

func TestParseEntitlementDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) entitlement ABC ")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.EntitlementDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "ABC",
						Pos:        ast.Position{Line: 1, Column: 25, Offset: 25},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 27, Offset: 27},
					},
				},
			},
			result,
		)
	})

	t.Run("nested entitlement", func(t *testing.T) {

		t.Parallel()

		// at static checking time, all entitlements nested inside non-contract-kinded composites
		// will be rejected
		result, errs := testParseDeclarations(`
            access(all) contract C {
                access(all) entitlement E
            }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.EntitlementDeclaration{
								Access: ast.AccessAll,
								Identifier: ast.Identifier{
									Identifier: "E",
									Pos:        ast.Position{Offset: 78, Line: 3, Column: 40},
								},
								Range: ast.Range{
									StartPos: ast.Position{Offset: 54, Line: 3, Column: 16},
									EndPos:   ast.Position{Offset: 78, Line: 3, Column: 40},
								},
							},
						},
					),
					Identifier: ast.Identifier{
						Identifier: "C",
						Pos:        ast.Position{Offset: 34, Line: 2, Column: 33},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 13, Line: 2, Column: 12},
						EndPos:   ast.Position{Offset: 92, Line: 4, Column: 12},
					},
					Access:        ast.AccessAll,
					CompositeKind: common.CompositeKindContract,
				},
			},
			result,
		)
	})

	t.Run("no identifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement")
		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected identifier, got EOF",
					Pos:     ast.Position{Offset: 24, Line: 1, Column: 24},
				},
			},
			errs,
		)
	})

	t.Run("view modifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) view entitlement E")
		AssertEqualWithDiff(t,
			[]error{
				&InvalidViewModifierError{
					Pos:             ast.Position{Offset: 13, Line: 1, Column: 13},
					DeclarationKind: common.DeclarationKindEntitlement,
				},
			},
			errs,
		)
	})
}

func TestParseMemberDocStrings(t *testing.T) {

	t.Parallel()

	t.Run("functions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct Test {

              /// noReturnNoBlock
              fun noReturnNoBlock()

              /// returnNoBlock
              fun returnNoBlock(): Int

              /// returnAndBlock
              fun returnAndBlock(): String {}
          }
        `)

		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.FunctionDeclaration{
								Access:    ast.AccessNotSpecified,
								DocString: " noReturnNoBlock",
								Identifier: ast.Identifier{
									Identifier: "noReturnNoBlock",
									Pos:        ast.Position{Offset: 78, Line: 5, Column: 18},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 93, Line: 5, Column: 33},
										EndPos:   ast.Position{Offset: 94, Line: 5, Column: 34},
									},
								},
								StartPos: ast.Position{Offset: 74, Line: 5, Column: 14},
							},
							&ast.FunctionDeclaration{
								Access:    ast.AccessNotSpecified,
								DocString: " returnNoBlock",
								Identifier: ast.Identifier{
									Identifier: "returnNoBlock",
									Pos:        ast.Position{Offset: 147, Line: 8, Column: 18},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 160, Line: 8, Column: 31},
										EndPos:   ast.Position{Offset: 161, Line: 8, Column: 32},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "Int",
											Pos:        ast.Position{Offset: 164, Line: 8, Column: 35},
										},
									},
									StartPos: ast.Position{Offset: 164, Line: 8, Column: 35},
								},
								StartPos: ast.Position{Offset: 143, Line: 8, Column: 14},
							},
							&ast.FunctionDeclaration{
								Access:    ast.AccessNotSpecified,
								DocString: " returnAndBlock",
								Identifier: ast.Identifier{
									Identifier: "returnAndBlock",
									Pos:        ast.Position{Offset: 220, Line: 11, Column: 18},
								},
								ParameterList: &ast.ParameterList{
									Range: ast.Range{
										StartPos: ast.Position{Offset: 234, Line: 11, Column: 32},
										EndPos:   ast.Position{Offset: 235, Line: 11, Column: 33},
									},
								},
								ReturnTypeAnnotation: &ast.TypeAnnotation{
									IsResource: false,
									Type: &ast.NominalType{
										Identifier: ast.Identifier{
											Identifier: "String",
											Pos:        ast.Position{Offset: 238, Line: 11, Column: 36},
										},
									},
									StartPos: ast.Position{Offset: 238, Line: 11, Column: 36},
								},
								FunctionBlock: &ast.FunctionBlock{
									Block: &ast.Block{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 245, Line: 11, Column: 43},
											EndPos:   ast.Position{Offset: 246, Line: 11, Column: 44},
										},
									},
								},
								StartPos: ast.Position{Offset: 216, Line: 11, Column: 14},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 258, Line: 12, Column: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("special functions", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          struct Test {

              /// unknown
              unknown()

              /// initNoBlock
              init()
          }
        `)

		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.CompositeDeclaration{
					Access:        ast.AccessNotSpecified,
					CompositeKind: common.CompositeKindStructure,
					Identifier: ast.Identifier{
						Identifier: "Test",
						Pos:        ast.Position{Offset: 18, Line: 2, Column: 17},
					},
					Members: ast.NewUnmeteredMembers(
						[]ast.Declaration{
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindUnknown,
								FunctionDeclaration: &ast.FunctionDeclaration{
									Access:    ast.AccessNotSpecified,
									DocString: " unknown",
									Identifier: ast.Identifier{
										Identifier: "unknown",
										Pos:        ast.Position{Offset: 66, Line: 5, Column: 14},
									},
									ParameterList: &ast.ParameterList{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 73, Line: 5, Column: 21},
											EndPos:   ast.Position{Offset: 74, Line: 5, Column: 22},
										},
									},
									StartPos: ast.Position{Offset: 66, Line: 5, Column: 14},
								},
							},
							&ast.SpecialFunctionDeclaration{
								Kind: common.DeclarationKindInitializer,
								FunctionDeclaration: &ast.FunctionDeclaration{
									Access:    ast.AccessNotSpecified,
									DocString: " initNoBlock",
									Identifier: ast.Identifier{
										Identifier: "init",
										Pos:        ast.Position{Offset: 121, Line: 8, Column: 14},
									},
									ParameterList: &ast.ParameterList{
										Range: ast.Range{
											StartPos: ast.Position{Offset: 125, Line: 8, Column: 18},
											EndPos:   ast.Position{Offset: 126, Line: 8, Column: 19},
										},
									},
									StartPos: ast.Position{Offset: 121, Line: 8, Column: 14},
								},
							},
						},
					),
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 138, Line: 9, Column: 10},
					},
				},
			},
			result,
		)
	})

}

func TestParseEntitlementMappingDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(" access(all) entitlement mapping M { } ")
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.EntitlementMappingDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "M",
						Pos:        ast.Position{Line: 1, Column: 33, Offset: 33},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 1, Offset: 1},
						EndPos:   ast.Position{Line: 1, Column: 37, Offset: 37},
					},
				},
			},
			result,
		)
	})

	t.Run("mappings", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          access(all) entitlement mapping M {
              A -> B
              C -> D
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,
			[]ast.Declaration{
				&ast.EntitlementMappingDeclaration{
					Access:    ast.AccessAll,
					DocString: "",
					Identifier: ast.Identifier{
						Identifier: "M",
						Pos:        ast.Position{Offset: 43, Line: 2, Column: 42},
					},
					Elements: []ast.EntitlementMapElement{
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos:        ast.Position{Offset: 61, Line: 3, Column: 14},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "B",
									Pos:        ast.Position{Offset: 66, Line: 3, Column: 19},
								},
							},
						},
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "C",
									Pos:        ast.Position{Offset: 82, Line: 4, Column: 14},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "D",
									Pos:        ast.Position{Offset: 87, Line: 4, Column: 19},
								},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 99, Line: 5, Column: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("mappings with includes", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          access(all) entitlement mapping M {
             include Y
             A -> B
             C -> D
             include X
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,

			[]ast.Declaration{
				&ast.EntitlementMappingDeclaration{
					Access:    ast.AccessAll,
					DocString: "",
					Identifier: ast.Identifier{
						Identifier: "M",
						Pos:        ast.Position{Offset: 43, Line: 2, Column: 42},
					},
					Elements: []ast.EntitlementMapElement{
						&ast.NominalType{
							Identifier: ast.Identifier{
								Identifier: "Y",
								Pos:        ast.Position{Offset: 68, Line: 3, Column: 21},
							},
						},
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos:        ast.Position{Offset: 83, Line: 4, Column: 13},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "B",
									Pos:        ast.Position{Offset: 88, Line: 4, Column: 18},
								},
							},
						},
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "C",
									Pos:        ast.Position{Offset: 103, Line: 5, Column: 13},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "D",
									Pos:        ast.Position{Offset: 108, Line: 5, Column: 18},
								},
							},
						},
						&ast.NominalType{
							Identifier: ast.Identifier{Identifier: "X",
								Pos: ast.Position{Offset: 131, Line: 6, Column: 21},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 143, Line: 7, Column: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("same line mappings", func(t *testing.T) {

		t.Parallel()

		result, errs := testParseDeclarations(`
          access(all) entitlement mapping M {
              A -> B C -> D
          }
        `)
		require.Empty(t, errs)

		AssertEqualWithDiff(t,

			[]ast.Declaration{
				&ast.EntitlementMappingDeclaration{
					Access: ast.AccessAll,
					Identifier: ast.Identifier{
						Identifier: "M",
						Pos:        ast.Position{Offset: 43, Line: 2, Column: 42},
					},
					Elements: []ast.EntitlementMapElement{
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "A",
									Pos:        ast.Position{Offset: 61, Line: 3, Column: 14},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "B",
									Pos:        ast.Position{Offset: 66, Line: 3, Column: 19},
								},
							},
						},
						&ast.EntitlementMapRelation{
							Input: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "C",
									Pos:        ast.Position{Offset: 68, Line: 3, Column: 21},
								},
							},
							Output: &ast.NominalType{
								Identifier: ast.Identifier{
									Identifier: "D",
									Pos:        ast.Position{Offset: 73, Line: 3, Column: 26},
								},
							},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 2, Column: 10},
						EndPos:   ast.Position{Offset: 85, Line: 4, Column: 10},
					},
				},
			},
			result,
		)
	})

	t.Run("missing entitlement keyword", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) mapping M {} ")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 13, Line: 1, Column: 13},
							EndPos:   ast.Position{Offset: 19, Line: 1, Column: 19},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)
	})

	t.Run("missing mapping keyword", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement M {} ")
		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTokenAtEndError{
					Token: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 27, Line: 1, Column: 27},
							EndPos:   ast.Position{Offset: 27, Line: 1, Column: 27},
						},
						Type: lexer.TokenBraceOpen,
					},
				},
			},
			errs,
		)
	})

	t.Run("missing body", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement mapping M ")
		AssertEqualWithDiff(t,
			[]error{
				&DeclarationMissingOpeningBraceError{
					Kind: common.DeclarationKindEntitlementMapping,
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 35, Line: 1, Column: 35},
							EndPos:   ast.Position{Offset: 35, Line: 1, Column: 35},
						},
						Type: lexer.TokenEOF,
					},
				},
				&DeclarationMissingClosingBraceError{
					Kind: common.DeclarationKindEntitlementMapping,
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 35, Line: 1, Column: 35},
							EndPos:   ast.Position{Offset: 35, Line: 1, Column: 35},
						},
						Type: lexer.TokenEOF,
					},
				},
			},
			errs,
		)
	})

	t.Run("missing close brace", func(t *testing.T) {

		t.Parallel()

		const code = `
          access(all) entitlement mapping M {`
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&DeclarationMissingClosingBraceError{
					Kind: common.DeclarationKindEntitlementMapping,
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 46, Line: 2, Column: 45},
							EndPos:   ast.Position{Offset: 46, Line: 2, Column: 45},
						},
						Type: lexer.TokenEOF,
					},
				},
			},
			errs,
		)

		var missingClosingBraceErr *DeclarationMissingClosingBraceError
		require.ErrorAs(t, errs[0], &missingClosingBraceErr)

		fixes := missingClosingBraceErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert closing brace",
					TextEdits: []ast.TextEdit{
						{
							Insertion: "}",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 46, Line: 2, Column: 45},
								EndPos:   ast.Position{Offset: 46, Line: 2, Column: 45},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"access(all) entitlement mapping M {}",
			strings.TrimSpace(fixes[0].TextEdits[0].ApplyTo(code)),
		)
	})

	t.Run("missing open brace", func(t *testing.T) {

		t.Parallel()

		const code = `
          access(all) entitlement mapping M }
        `
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&DeclarationMissingOpeningBraceError{
					Kind: common.DeclarationKindEntitlementMapping,
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 45, Line: 2, Column: 44},
							EndPos:   ast.Position{Offset: 45, Line: 2, Column: 44},
						},
						Type: lexer.TokenBraceClose,
					},
				},
			},
			errs,
		)

		var missingBraceErr *DeclarationMissingOpeningBraceError
		require.ErrorAs(t, errs[0], &missingBraceErr)

		fixes := missingBraceErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert opening brace",
					TextEdits: []ast.TextEdit{
						{
							Insertion: " {",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 44, Line: 2, Column: 43},
								EndPos:   ast.Position{Offset: 44, Line: 2, Column: 43},
							},
						},
					},
				},
			},
			fixes,
		)

		assert.Equal(t,
			"access(all) entitlement mapping M { }",
			strings.TrimSpace(fixes[0].TextEdits[0].ApplyTo(code)),
		)
	})

	t.Run("missing identifier", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" access(all) entitlement mapping {}")
		AssertEqualWithDiff(t,
			[]error{
				&SyntaxError{
					Message: "expected identifier following entitlement mapping declaration, got `{`",
					Pos:     ast.Position{Offset: 33, Line: 1, Column: 33},
				},
			},
			errs,
		)
	})

	t.Run("non-nominal mapping first", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          access(all) entitlement mapping M {
              &A -> B
          }
        `)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidEntitlementMappingTypeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 61, Line: 3, Column: 14},
						EndPos:   ast.Position{Offset: 62, Line: 3, Column: 15},
					},
				},
			},
			errs,
		)
	})

	t.Run("non-nominal mapping second", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          access(all) entitlement mapping M {
              A -> [B]
          }
        `)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidEntitlementMappingTypeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 66, Line: 3, Column: 19},
						EndPos:   ast.Position{Offset: 68, Line: 3, Column: 21},
					},
				},
			},
			errs,
		)
	})

	t.Run("missing arrow", func(t *testing.T) {

		t.Parallel()

		const code = `
          access(all) entitlement mapping M {
              A B
          }
        `
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&MissingRightArrowInEntitlementMappingError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 63, Line: 3, Column: 16},
							EndPos:   ast.Position{Offset: 63, Line: 3, Column: 16},
						},
						Type: lexer.TokenIdentifier,
					},
				},
			},
			errs,
		)

		var missingArrowErr *MissingRightArrowInEntitlementMappingError
		require.ErrorAs(t, errs[0], &missingArrowErr)

		fixes := missingArrowErr.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Insert `->`",
					TextEdits: []ast.TextEdit{
						{
							Insertion: " ->",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 62, Line: 3, Column: 15},
								EndPos:   ast.Position{Offset: 62, Line: 3, Column: 15},
							},
						},
					},
				},
			},
			fixes,
		)

		const expected = `
          access(all) entitlement mapping M {
              A -> B
          }
        `

		assert.Equal(t,
			expected,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("wrong mapping separator", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          access(all) entitlement mapping M {
              A - B
          }
        `)

		AssertEqualWithDiff(t,
			[]error{
				&MissingRightArrowInEntitlementMappingError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 63, Line: 3, Column: 16},
							EndPos:   ast.Position{Offset: 63, Line: 3, Column: 16},
						},
						Type: lexer.TokenMinus,
					},
				},
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 63, Line: 3, Column: 16},
							EndPos:   ast.Position{Offset: 63, Line: 3, Column: 16},
						},
						Type: lexer.TokenMinus,
					},
				},
			},
			errs,
		)
	})

	t.Run("non-nominal include", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
          access(all) entitlement mapping M {
              include &A
          }
        `)

		AssertEqualWithDiff(t,
			[]error{
				&InvalidEntitlementMappingIncludeTypeError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 69, Line: 3, Column: 22},
						EndPos:   ast.Position{Offset: 70, Line: 3, Column: 23},
					},
				},
			},
			errs,
		)
	})

	t.Run("include with arrow", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(`
            access(all) entitlement mapping M {
                include -> B
            }
        `)

		AssertEqualWithDiff(t,
			[]error{
				&UnexpectedTypeStartError{
					GotToken: lexer.Token{
						Range: ast.Range{
							StartPos: ast.Position{Offset: 73, Line: 3, Column: 24},
							EndPos:   ast.Position{Offset: 74, Line: 3, Column: 25},
						},
						Type: lexer.TokenRightArrow,
					},
				},
			},
			errs,
		)
	})
}

func TestParseInvalidSpecialFunctionReturnTypeAnnotation(t *testing.T) {

	t.Parallel()

	const code = `
      struct Test {

          init(): Int
      }
    `
	_, errs := testParseDeclarations(code)
	AssertEqualWithDiff(t,
		[]error{
			&SpecialFunctionReturnTypeError{
				DeclarationKind: common.DeclarationKindInitializer,
				Range: ast.Range{
					StartPos: ast.Position{Offset: 40, Line: 4, Column: 18},
					EndPos:   ast.Position{Offset: 42, Line: 4, Column: 20},
				},
			},
		},
		errs,
	)

	var returnTypeError *SpecialFunctionReturnTypeError
	require.ErrorAs(t, errs[0], &returnTypeError)

	fixes := returnTypeError.SuggestFixes(code)
	AssertEqualWithDiff(t,
		[]errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Remove return type from special function",
				TextEdits: []ast.TextEdit{
					{
						Replacement: "",
						Range: ast.Range{
							StartPos: ast.Position{Offset: 38, Line: 4, Column: 16},
							EndPos:   ast.Position{Offset: 42, Line: 4, Column: 20},
						},
					},
				},
			},
		},
		fixes,
	)

	const expected = `
      struct Test {

          init()
      }
    `
	assert.Equal(t,
		expected,
		fixes[0].TextEdits[0].ApplyTo(code),
	)
}

func TestSoftKeywordsInFunctionDeclaration(t *testing.T) {
	t.Parallel()

	posFromName := func(name string, offset int) ast.Position {
		offsetPos := len(name) + offset
		return ast.Position{Line: 1, Offset: offsetPos, Column: offsetPos}
	}

	testSoftKeyword := func(name string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`fun %s() {}`, name)

			result, errs := testParseDeclarations(code)
			require.Empty(t, errs)

			expected := []ast.Declaration{
				&ast.FunctionDeclaration{
					Access: ast.AccessNotSpecified,
					Identifier: ast.Identifier{
						Identifier: name,
						Pos:        ast.Position{Offset: 4, Line: 1, Column: 4},
					},
					StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
					FunctionBlock: &ast.FunctionBlock{
						Block: &ast.Block{
							Range: ast.Range{
								StartPos: posFromName(name, 7),
								EndPos:   posFromName(name, 8),
							},
						},
					},
					ParameterList: &ast.ParameterList{
						Range: ast.Range{
							StartPos: posFromName(name, 4),
							EndPos:   posFromName(name, 5),
						},
					},
				},
			}
			AssertEqualWithDiff(t, expected, result)

		})
	}

	for _, keyword := range SoftKeywords {
		testSoftKeyword(keyword)
	}
}

func TestParseDeprecatedAccessModifiers(t *testing.T) {

	t.Parallel()

	t.Run("pub", func(t *testing.T) {

		t.Parallel()

		const code = " pub fun foo ( ) { }"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&PubAccessError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
					},
				},
			},
			errs,
		)

		var pubAccessError *PubAccessError
		require.ErrorAs(t, errs[0], &pubAccessError)

		fixes := pubAccessError.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Replace with `access(all)`",
					TextEdits: []ast.TextEdit{
						{
							Replacement: "access(all)",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
								EndPos:   ast.Position{Offset: 3, Line: 1, Column: 3},
							},
						},
					},
				},
			},
			fixes,
		)

		const expected = " access(all) fun foo ( ) { }"
		assert.Equal(t,
			expected,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("priv", func(t *testing.T) {

		t.Parallel()

		const code = " priv fun foo ( ) { }"
		_, errs := testParseDeclarations(code)

		AssertEqualWithDiff(t,
			[]error{
				&PrivAccessError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
					},
				},
			},
			errs,
		)

		var privAccessError *PrivAccessError
		require.ErrorAs(t, errs[0], &privAccessError)

		fixes := privAccessError.SuggestFixes(code)
		AssertEqualWithDiff(t,
			[]errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Replace with `access(self)`",
					TextEdits: []ast.TextEdit{
						{
							Replacement: "access(self)",
							Range: ast.Range{
								StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
								EndPos:   ast.Position{Offset: 4, Line: 1, Column: 4},
							},
						},
					},
				},
			},
			fixes,
		)

		const expected = " access(self) fun foo ( ) { }"
		assert.Equal(t,
			expected,
			fixes[0].TextEdits[0].ApplyTo(code),
		)
	})

	t.Run("pub(set)", func(t *testing.T) {

		t.Parallel()

		_, errs := testParseDeclarations(" pub(set) fun foo ( ) { }")
		AssertEqualWithDiff(t,
			[]error{
				&PubSetAccessError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 1, Line: 1, Column: 1},
						EndPos:   ast.Position{Offset: 8, Line: 1, Column: 8},
					},
				},
			},
			errs,
		)
	})

	t.Run("pub(foo)", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations("pub(foo) fun x() {}")

		AssertEqualWithDiff(t,
			[]error{
				&PubAccessError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 7, Line: 1, Column: 7},
					},
				},
			},
			errs,
		)
	})

	t.Run("pub(set, missing closing paren", func(t *testing.T) {
		t.Parallel()

		_, errs := testParseDeclarations("pub(set")

		AssertEqualWithDiff(t,
			[]error{
				&PubSetAccessError{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
						EndPos:   ast.Position{Offset: 6, Line: 1, Column: 6},
					},
				},
			},
			errs,
		)
	})
}

func TestParseMissingCommaInParameterListError(t *testing.T) {

	t.Parallel()

	const code = `
        fun test(a: Int b: Int) {
            return a + b
        }
    `

	_, errs := testParseDeclarations(code)
	require.Len(t, errs, 1)

	var missingCommaErr *MissingCommaInParameterListError
	require.ErrorAs(t, errs[0], &missingCommaErr)

	assert.Equal(t,
		&MissingCommaInParameterListError{
			Pos: ast.Position{Offset: 25, Line: 2, Column: 24},
		},
		missingCommaErr,
	)

	fixes := missingCommaErr.SuggestFixes(code)

	require.Equal(t,
		[]errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Insert comma",
				TextEdits: []ast.TextEdit{
					{
						Insertion: ",",
						Range: ast.Range{
							StartPos: ast.Position{Offset: 24, Line: 2, Column: 23},
							EndPos:   ast.Position{Offset: 24, Line: 2, Column: 23},
						},
					},
				},
			},
		},
		fixes,
	)

	const expected = `
        fun test(a: Int, b: Int) {
            return a + b
        }
    `
	assert.Equal(t,
		expected,
		fixes[0].TextEdits[0].ApplyTo(code),
	)
}

func TestParseKeywordsAsFieldNames(t *testing.T) {

	t.Parallel()

	for _, keyword := range []string{
		"event",
		"contract",
		"default",
	} {
		keyword := keyword

		t.Run(keyword, func(t *testing.T) {
			t.Parallel()

			_, errs := testParseDeclarations(fmt.Sprintf(
				"struct Foo { var %s: String }",
				keyword,
			))
			require.Empty(t, errs)
		})
	}
}

func TestParseStructNamedTransaction(t *testing.T) {
	t.Parallel()

	code := `
        struct transaction {}

        fun test(): transaction {
            return transaction()
        }
    `

	_, errs := testParseProgram(code)

	// The compiler relies on the type-name `transaction`,
	// to distinguish between constructing a transaction value vs any other composite value.
	// So defining composite types with the name `transaction` must not be allwoed.
	AssertEqualWithDiff(
		t,
		Error{
			Code: []uint8(code),
			Errors: []error{
				&SyntaxError{
					Pos:     ast.Position{Line: 2, Column: 15, Offset: 16},
					Message: "expected identifier following struct declaration, got keyword transaction",
				},
			},
		},
		errs,
	)
}

func TestParseTransactionDeclarationMissingOpeningBrace(t *testing.T) {
	t.Parallel()

	const code = `transaction }`
	_, errs := testParseStatements(code)

	AssertEqualWithDiff(t,
		[]error{
			&DeclarationMissingOpeningBraceError{
				Kind: common.DeclarationKindTransaction,
				GotToken: lexer.Token{
					Type: lexer.TokenBraceClose,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 12, Line: 1, Column: 12},
						EndPos:   ast.Position{Offset: 12, Line: 1, Column: 12},
					},
				},
			},
		},
		errs,
	)

	var missingBraceErr *DeclarationMissingOpeningBraceError
	require.ErrorAs(t, errs[0], &missingBraceErr)

	fixes := missingBraceErr.SuggestFixes(code)
	AssertEqualWithDiff(t,
		[]errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Insert opening brace",
				TextEdits: []ast.TextEdit{
					{
						Insertion: " {",
						Range: ast.Range{
							StartPos: ast.Position{Offset: 11, Line: 1, Column: 11},
							EndPos:   ast.Position{Offset: 11, Line: 1, Column: 11},
						},
					},
				},
			},
		},
		fixes,
	)

	assert.Equal(t,
		"transaction { }",
		fixes[0].TextEdits[0].ApplyTo(code),
	)
}

func TestParseTransactionDeclarationMissingOpeningBraceEOF(t *testing.T) {
	t.Parallel()

	const code = `transaction`
	_, errs := testParseStatements(code)

	AssertEqualWithDiff(t,
		[]error{
			&DeclarationMissingOpeningBraceError{
				Kind: common.DeclarationKindTransaction,
				GotToken: lexer.Token{
					Type: lexer.TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 1, Column: 11},
						EndPos:   ast.Position{Offset: 11, Line: 1, Column: 11},
					},
				},
			},
			&UnexpectedTokenAtEndError{
				Token: lexer.Token{
					Type: lexer.TokenEOF,
					Range: ast.Range{
						StartPos: ast.Position{Offset: 11, Line: 1, Column: 11},
						EndPos:   ast.Position{Offset: 11, Line: 1, Column: 11},
					},
				},
			},
		},
		errs,
	)

	var missingBraceErr *DeclarationMissingOpeningBraceError
	require.ErrorAs(t, errs[0], &missingBraceErr)

	fixes := missingBraceErr.SuggestFixes(code)
	AssertEqualWithDiff(t,
		[]errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Insert opening brace",
				TextEdits: []ast.TextEdit{
					{
						Insertion: " {",
						Range: ast.Range{
							StartPos: ast.Position{Offset: 11, Line: 1, Column: 11},
							EndPos:   ast.Position{Offset: 11, Line: 1, Column: 11},
						},
					},
				},
			},
		},
		fixes,
	)

	assert.Equal(t,
		"transaction {",
		fixes[0].TextEdits[0].ApplyTo(code),
	)
}
