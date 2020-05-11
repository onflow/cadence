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
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestParseNominalType(t *testing.T) {

	t.Run("simple", func(t *testing.T) {
		result, errs := ParseType("Int")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
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
		result, errs := ParseType("Foo.Bar")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
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
}

func TestParseArrayType(t *testing.T) {

	t.Run("variable", func(t *testing.T) {
		result, errs := ParseType("[Int]")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
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

	t.Run("constant", func(t *testing.T) {
		result, errs := ParseType("[Int ; 2 ]")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ConstantSizedType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "Int",
						Pos:        ast.Position{Line: 1, Column: 1, Offset: 1},
					},
				},
				Size: &ast.IntegerExpression{
					Value: big.NewInt(1),
					Base:  10,
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

}

func TestParseOptionalType(t *testing.T) {

	t.Run("nominal", func(t *testing.T) {
		result, errs := ParseType("Int?")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
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
}

func TestParseReferenceType(t *testing.T) {

	t.Run("unauthorized, nominal", func(t *testing.T) {
		result, errs := ParseType("&Int")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.ReferenceType{
				Authorized: false,
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
}

func TestParseOptionalReferenceType(t *testing.T) {

	t.Run("unauthorized", func(t *testing.T) {
		result, errs := ParseType("&Int?")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.OptionalType{
				Type: &ast.ReferenceType{
					Authorized: false,
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

func TestParseRestrictedType(t *testing.T) {

	t.Run("with restricted type, no restrictions", func(t *testing.T) {
		result, errs := ParseType("T{}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.RestrictedType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Restrictions: nil,
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 2, Offset: 2},
				},
			},
			result,
		)
	})

	t.Run("with restricted type, one restriction", func(t *testing.T) {
		result, errs := ParseType("T{U}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.RestrictedType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Restrictions: []*ast.NominalType{
					{
						Identifier: ast.Identifier{
							Identifier: "U",
							Pos:        ast.Position{Line: 1, Column: 2, Offset: 2},
						},
					},
				},
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 3, Offset: 3},
				},
			},
			result,
		)
	})

	t.Run("with restricted type, two restrictions", func(t *testing.T) {
		result, errs := ParseType("T{ U , V }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.RestrictedType{
				Type: &ast.NominalType{
					Identifier: ast.Identifier{
						Identifier: "T",
						Pos:        ast.Position{Line: 1, Column: 0, Offset: 0},
					},
				},
				Restrictions: []*ast.NominalType{
					{
						Identifier: ast.Identifier{
							Identifier: "U",
							Pos:        ast.Position{Line: 1, Column: 3, Offset: 3},
						},
					},
					{
						Identifier: ast.Identifier{
							Identifier: "V",
							Pos:        ast.Position{Line: 1, Column: 7, Offset: 7},
						},
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

	t.Run("without restricted type, no restrictions", func(t *testing.T) {
		result, errs := ParseType("{}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.RestrictedType{
				Range: ast.Range{
					StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
					EndPos:   ast.Position{Line: 1, Column: 1, Offset: 1},
				},
			},
			result,
		)
	})

	t.Run("without restricted type, one restriction", func(t *testing.T) {
		result, errs := ParseType("{ T }")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			&ast.RestrictedType{
				Restrictions: []*ast.NominalType{
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

	t.Run("invalid: without restricted type, missing type after comma", func(t *testing.T) {
		_, errs := ParseType("{ T , }")
		require.Equal(t,
			[]error{
				errors.New("missing type after comma"),
			},
			errs,
		)
	})

	t.Run("invalid: without restricted type, type without comma", func(t *testing.T) {
		_, errs := ParseType("{ T U }")
		require.Equal(t,
			[]error{
				errors.New("unexpected type"),
			},
			errs,
		)
	})

	t.Run("invalid: without restricted type, colon", func(t *testing.T) {
		_, errs := ParseType("{ T , U : V }")
		require.Equal(t,
			[]error{
				errors.New("unexpected colon in restricted type"),
			},
			errs,
		)
	})

	t.Run("invalid: with restricted type, colon", func(t *testing.T) {
		_, errs := ParseType("T{ T , U : V }")
		require.Equal(t,
			[]error{
				errors.New(`unexpected token: got ":", expected ","`),
			},
			errs,
		)
	})

	t.Run("invalid: without restricted type, first is non-nominal", func(t *testing.T) {
		_, errs := ParseType("{[T]}")
		require.Equal(t,
			[]error{
				errors.New("non-nominal type in restriction list: [T]"),
			},
			errs,
		)
	})

	t.Run("invalid: with restricted type, first is non-nominal", func(t *testing.T) {
		_, errs := ParseType("T{[U]}")
		require.Equal(t,
			[]error{
				errors.New("non-nominal type in restriction list: [U]"),
			},
			errs,
		)
	})

	t.Run("invalid: without restricted type, second is non-nominal", func(t *testing.T) {
		_, errs := ParseType("{T, [U]}")
		require.Equal(t,
			[]error{
				errors.New("non-nominal type in restriction list: [U]"),
			},
			errs,
		)
	})

	t.Run("invalid: with restricted type, second is non-nominal", func(t *testing.T) {
		_, errs := ParseType("T{U, [V]}")
		require.Equal(t,
			[]error{
				errors.New("non-nominal type in restriction list: [V]"),
			},
			errs,
		)
	})

	t.Run("invalid: without restricted type, missing end", func(t *testing.T) {
		_, errs := ParseType("{")
		require.Equal(t,
			[]error{
				errors.New("invalid end, expected type"),
			},
			errs,
		)
	})

	t.Run("invalid: with restricted type, missing end", func(t *testing.T) {
		_, errs := ParseType("T{")
		require.Equal(t,
			[]error{
				errors.New("invalid end, expected type"),
			},
			errs,
		)
	})

	t.Run("invalid: without restricted type, missing end after type", func(t *testing.T) {
		_, errs := ParseType("{U")
		require.Equal(t,
			[]error{
				errors.New("missing end, expected \"}\""),
			},
			errs,
		)
	})

	t.Run("invalid: with restricted type, missing end after type", func(t *testing.T) {
		_, errs := ParseType("T{U")
		require.Equal(t,
			[]error{
				errors.New("missing end, expected \"}\""),
			},
			errs,
		)
	})

	t.Run("invalid: without restricted type, missing end after comma", func(t *testing.T) {
		_, errs := ParseType("{U,")
		require.Equal(t,
			[]error{
				errors.New("invalid end, expected type"),
			},
			errs,
		)
	})

	t.Run("invalid: with restricted type, missing end after comma", func(t *testing.T) {
		_, errs := ParseType("T{U,")
		require.Equal(t,
			[]error{
				errors.New("invalid end, expected type"),
			},
			errs,
		)
	})

	t.Run("invalid: without restricted type, just comma", func(t *testing.T) {
		_, errs := ParseType("{,}")
		require.Equal(t,
			[]error{
				errors.New("unexpected comma in restricted type"),
			},
			errs,
		)
	})

	t.Run("invalid: with restricted type, just comma", func(t *testing.T) {
		_, errs := ParseType("T{,}")
		require.Equal(t,
			[]error{
				errors.New("unexpected comma in restricted type"),
			},
			errs,
		)
	})
}

func TestParseDictionaryType(t *testing.T) {

	t.Run("valid", func(t *testing.T) {
		result, errs := ParseType("{T: U}")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
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
		_, errs := ParseType("{T:}")
		require.Equal(t,
			[]error{
				errors.New("missing dictionary value type"),
			},
			errs,
		)
	})

	t.Run("invalid, missing key and value type", func(t *testing.T) {
		_, errs := ParseType("{:}")
		require.Equal(t,
			[]error{
				errors.New("unexpected colon in dictionary type"),
			},
			errs,
		)
	})

	t.Run("invalid, missing key type", func(t *testing.T) {
		_, errs := ParseType("{:U}")
		require.Equal(t,
			[]error{
				errors.New("unexpected colon in dictionary type"),
			},
			errs,
		)
	})

	t.Run("invalid, unexpected comma after value type", func(t *testing.T) {
		_, errs := ParseType("{T:U,}")
		require.Equal(t,
			[]error{
				errors.New("unexpected comma in dictionary type"),
			},
			errs,
		)
	})

	t.Run("invalid, unexpected colon after value type", func(t *testing.T) {
		_, errs := ParseType("{T:U:}")
		require.Equal(t,
			[]error{
				errors.New("unexpected colon in dictionary type"),
			},
			errs,
		)
	})

	t.Run("invalid, unexpected colon after colon", func(t *testing.T) {
		_, errs := ParseType("{T::U}")
		require.Equal(t,
			[]error{
				errors.New("unexpected colon in dictionary type"),
			},
			errs,
		)
	})

	t.Run("invalid, missing value type after colon", func(t *testing.T) {
		_, errs := ParseType("{T:")
		require.Equal(t,
			[]error{
				errors.New("invalid end, expected type"),
			},
			errs,
		)
	})

	t.Run("invalid, missing end after key type  and value type", func(t *testing.T) {
		_, errs := ParseType("{T:U")
		require.Equal(t,
			[]error{
				errors.New("missing end, expected \"}\""),
			},
			errs,
		)
	})

}
