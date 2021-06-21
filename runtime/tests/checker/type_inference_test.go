/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckArrayElementTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("numeric array", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: [Int8] = [1, 2, 3]
          let y: [Int8]? = [1, 2, 3]
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		yType := RequireGlobalValue(t, checker.Elaboration, "y")

		assert.Equal(t,
			&sema.VariableSizedType{
				Type: sema.Int8Type,
			},
			xType,
		)

		assert.Equal(t,
			&sema.OptionalType{
				Type: &sema.VariableSizedType{
					Type: sema.Int8Type,
				},
			},
			yType,
		)
	})

	t.Run("AnyStruct array", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: [AnyStruct] = [1, 2, 3]
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			&sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
			xType,
		)
	})

	t.Run("invalid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x: [Int8]? = [1, 534, 3]
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidIntegerLiteralRangeError{}, errs[0])
		intRangeErr := errs[0].(*sema.InvalidIntegerLiteralRangeError)

		assert.Equal(t, sema.Int8Type, intRangeErr.ExpectedType)
		assert.Equal(t, sema.Int8Type.MinInt(), intRangeErr.ExpectedMinInt)
		assert.Equal(t, sema.Int8Type.MaxInt(), intRangeErr.ExpectedMaxInt)
	})

	t.Run("AnyStruct", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: AnyStruct = [1, 534534, 3]
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			sema.AnyStructType,
			xType,
		)
	})

	t.Run("inferring from rhs", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x = [1, 534534, 3]
          let y: Int = x[2]
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		yType := RequireGlobalValue(t, checker.Elaboration, "y")

		assert.Equal(t,
			&sema.VariableSizedType{
				Type: sema.IntType,
			},
			xType,
		)

		assert.Equal(t,
			sema.IntType,
			yType,
		)
	})

	t.Run("nested array", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: [[[Int8]]] = [[[1, 2, 3], [4]], []]
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			&sema.VariableSizedType{
				Type: &sema.VariableSizedType{
					Type: &sema.VariableSizedType{
						Type: sema.Int8Type,
					},
				},
			},
			xType,
		)
	})
}

func TestCheckDictionaryTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("numerics", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: {Int8: Int64} = {0: 6, 1: 5}
          let y: {Int8: Int64?} = {0: 6, 1: 5}
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		yType := RequireGlobalValue(t, checker.Elaboration, "y")

		assert.Equal(t,
			&sema.DictionaryType{
				KeyType:   sema.Int8Type,
				ValueType: sema.Int64Type,
			},
			xType,
		)

		assert.Equal(t,
			&sema.DictionaryType{
				KeyType: sema.Int8Type,
				ValueType: &sema.OptionalType{
					Type: sema.Int64Type,
				},
			},
			yType,
		)
	})

	t.Run("heterogeneous", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: {Int: AnyStruct} = {0: 6, 1: "hello", 2: nil}
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			&sema.DictionaryType{
				KeyType:   sema.IntType,
				ValueType: sema.AnyStructType,
			},
			xType,
		)
	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: {Int: {Int: {Int: AnyStruct}}} = {0: {0: {0: 6}, 1: {0: 7}}}
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			&sema.DictionaryType{
				KeyType: sema.IntType,
				ValueType: &sema.DictionaryType{
					KeyType: sema.IntType,
					ValueType: &sema.DictionaryType{
						KeyType:   sema.IntType,
						ValueType: sema.AnyStructType,
					},
				},
			},
			xType,
		)
	})

	t.Run("invalid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x: {Int8: Int64} = {23423: 6, 1: 5}
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidIntegerLiteralRangeError{}, errs[0])
		intRangeErr := errs[0].(*sema.InvalidIntegerLiteralRangeError)

		assert.Equal(t, sema.Int8Type, intRangeErr.ExpectedType)
		assert.Equal(t, sema.Int8Type.MinInt(), intRangeErr.ExpectedMinInt)
		assert.Equal(t, sema.Int8Type.MaxInt(), intRangeErr.ExpectedMaxInt)
	})

	t.Run("nested invalid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x: {Int: {Int: {Int: Int8}}} = {0: {0: {0: "hello"}, 1: {0: 7}}}
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchErr := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.Int8Type, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchErr.ActualType)
	})
}

func TestCheckReturnTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("array type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test(): [Int8] {
                return [1, 2, 3]
            }
        `)
		require.NoError(t, err)
	})

	t.Run("void", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test() {
                return 5
            }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchErr := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.VoidType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.IntType, typeMismatchErr.ActualType)
	})
}

func TestCheckFunctionArgumentTypeInference(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x = foo(a: [1, 2, 3])

      fun foo(a: [Int8]) {}
    `)

	// Type inferring for function arguments is not supported yet.
	errs := ExpectCheckerErrors(t, err, 1)

	require.IsType(t, &sema.TypeMismatchError{}, errs[0])

	typeMismatchErr := errs[0].(*sema.TypeMismatchError)

	assert.Equal(t,
		&sema.VariableSizedType{
			Type: sema.Int8Type,
		},
		typeMismatchErr.ExpectedType,
	)

	assert.Equal(t,
		&sema.VariableSizedType{
			Type: sema.IntType,
		},
		typeMismatchErr.ActualType,
	)
}

func TestCheckBinaryExpressionTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("integer add", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: Int8 = 5 + 6
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			sema.Int8Type,
			xType,
		)
	})

	t.Run("fixed point add", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: Fix64 = 5.0 + 6.0
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			sema.Fix64Type,
			xType,
		)
	})

	t.Run("integer bitwise", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: UInt8 = 1 >> 2
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			sema.UInt8Type,
			xType,
		)
	})

	t.Run("contextually expected type, type annotation", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x = 1 as UInt8
          let y: Integer = x + 1
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		yType := RequireGlobalValue(t, checker.Elaboration, "y")

		assert.Equal(t,
			sema.UInt8Type,
			xType,
		)

		assert.Equal(t,
			sema.IntegerType,
			yType,
		)
	})

	t.Run("contextually expected type, indexing type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let string = "this is a test"
          let index = 1 as UInt8
          let character = string[index + 1]
        `)
		require.NoError(t, err)

		indexType := RequireGlobalValue(t, checker.Elaboration, "index")
		characterType := RequireGlobalValue(t, checker.Elaboration, "character")

		assert.Equal(t,
			sema.UInt8Type,
			indexType,
		)

		assert.Equal(t,
			sema.CharacterType,
			characterType,
		)
	})

	t.Run("no contextually expected type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x = 1 as UInt8
          let y = x + 1
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		yType := RequireGlobalValue(t, checker.Elaboration, "y")

		assert.Equal(t,
			sema.UInt8Type,
			xType,
		)

		assert.Equal(t,
			sema.UInt8Type,
			yType,
		)
	})
}

func TestCheckUnaryExpressionTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("invalid negate", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          let x: Bool = !"string"
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidUnaryOperandError{}, errs[0])
		invalidUnaryOpKindErr := errs[0].(*sema.InvalidUnaryOperandError)

		assert.Equal(t, sema.BoolType, invalidUnaryOpKindErr.ExpectedType)
		assert.Equal(t, sema.StringType, invalidUnaryOpKindErr.ActualType)
	})
}

func TestCheckForceExpressionTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("array forced", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: [Int8] = [5, 7, 2]!
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			&sema.VariableSizedType{
				Type: sema.Int8Type,
			},
			xType,
		)
	})

	t.Run("double-optional repeated forced", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: Int?? = 4
          let y: Int = x!!
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		yType := RequireGlobalValue(t, checker.Elaboration, "y")

		assert.Equal(t,
			&sema.OptionalType{
				Type: &sema.OptionalType{
					Type: sema.IntType,
				},
			},
			xType,
		)

		assert.Equal(t,
			sema.IntType,
			yType,
		)
	})

	t.Run("optional repeated forced", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: Int? = 4
          let y: Int = x!!
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		yType := RequireGlobalValue(t, checker.Elaboration, "y")

		assert.Equal(t,
			&sema.OptionalType{
				Type: sema.IntType,
			},
			xType,
		)

		assert.Equal(t,
			sema.IntType,
			yType,
		)
	})

	t.Run("non-optional repeated forced", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x: Int = 4
          let y: Int = x!!
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		yType := RequireGlobalValue(t, checker.Elaboration, "y")

		assert.Equal(t,
			sema.IntType,
			xType,
		)

		assert.Equal(t,
			sema.IntType,
			yType,
		)
	})
}

func TestCastExpressionTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("array", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let x = [1, 3] as [Int8]
        `)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		assert.Equal(t,
			&sema.VariableSizedType{
				Type: sema.Int8Type,
			},
			xType,
		)
	})

	t.Run("number out of range", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x = [1, 764] as [Int8]
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidIntegerLiteralRangeError{}, errs[0])
		intRangeErr := errs[0].(*sema.InvalidIntegerLiteralRangeError)

		assert.Equal(t, sema.Int8Type, intRangeErr.ExpectedType)
		assert.Equal(t, sema.Int8Type.MinInt(), intRangeErr.ExpectedMinInt)
		assert.Equal(t, sema.Int8Type.MaxInt(), intRangeErr.ExpectedMaxInt)
	})

	t.Run("mismatching types", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x = [1, "hello"] as [Int8]
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchErr := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.Int8Type, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchErr.ActualType)
	})
}

func TestCheckVoidTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("void type annotation", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x: Void = 5 + 6
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchErr := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.VoidType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.IntType, typeMismatchErr.ActualType)
	})
}

func TestCheckInferenceWithCheckerErrors(t *testing.T) {

	t.Parallel()

	t.Run("undefined type reference", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Foo {
                let ownedNFTs: @{Int: UnknownType}

                init() {
                    self.ownedNFTs = {}
                }

                fun borrowNFT(id: Int): &UnknownType {
                    return &self.ownedNFTs[id] as &UnknownType
                }
            }
        `)

		errs := ExpectCheckerErrors(t, err, 4)

		for _, err := range errs {
			require.IsType(t, &sema.NotDeclaredError{}, err)
			notDeclaredError := err.(*sema.NotDeclaredError)
			assert.Equal(t, "UnknownType", notDeclaredError.Name)
		}
	})
}

func TestCheckArraySupertypeInference(t *testing.T) {

	t.Parallel()

	tests := []struct {
		literal             string
		expectedElementType sema.Type
	}{
		{
			literal:             `[0, true]`,
			expectedElementType: sema.AnyStructType,
		},
		{
			literal:             `[0, 6, 275]`,
			expectedElementType: sema.IntType,
		},
		{
			literal:             `[UInt(65), 6, 275, 13423]`,
			expectedElementType: sema.IntegerType,
		},
		{
			literal:             `[UInt(0), UInt(6), UInt(275), UInt(13423)]`,
			expectedElementType: sema.UIntType,
		},
		{
			literal: `["hello", nil, nil, nil]`,
			expectedElementType: &sema.OptionalType{
				Type: sema.StringType,
			},
		},
	}

	for _, test := range tests {
		code := fmt.Sprintf(
			"let x = %s",
			test.literal,
		)

		checker, err := ParseAndCheck(t, code)
		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")

		require.IsType(t, &sema.VariableSizedType{}, xType)
		arrayType := xType.(*sema.VariableSizedType)

		assert.Equal(t, test.expectedElementType, arrayType.Type)
	}
}
