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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestArrayElementTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("numeric array", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: [Int8] = [1, 2, 3]
				var y: [Int8]? = [1, 2, 3]
			}
		`)

		require.NoError(t, err)
	})

	t.Run("anystruct array", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: [AnyStruct] = [1, 2, 3]
			}
		`)

		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: [Int8]? = [1, 534, 3]
			}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidIntegerLiteralRangeError{}, errs[0])
		intRangeErr := errs[0].(*sema.InvalidIntegerLiteralRangeError)

		assert.Equal(t, sema.Int8Type, intRangeErr.ExpectedType)
		assert.Equal(t, sema.Int8Type.MinInt(), intRangeErr.ExpectedMinInt)
		assert.Equal(t, sema.Int8Type.MaxInt(), intRangeErr.ExpectedMaxInt)
	})

	t.Run("anystruct", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: AnyStruct = [1, 534534, 3]
			}
		`)

		require.NoError(t, err)
	})

	t.Run("inferring from rhs", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x = [1, 534534, 3]
				var y: Int = x[2]
			}
		`)

		require.NoError(t, err)
	})

	t.Run("nested array", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: [[[Int8]]] = [[[1, 2, 3], [4]], []]
			}
		`)

		require.NoError(t, err)
	})
}

func TestDictionaryTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("numerics", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: {Int8:Int64} = {0: 6, 1: 5}
				var y: {Int8:Int64?} = {0: 6, 1: 5}
			}
		`)

		require.NoError(t, err)
	})

	t.Run("heterogeneous", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: {Int:AnyStruct} = {0: 6, 1: "hello", 2: nil}
			}
		`)

		require.NoError(t, err)
	})

	t.Run("nested", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: {Int:{Int:{Int:AnyStruct}}} = {0: {0: {0: 6}, 1: {0: 7}}}
			}
		`)

		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: {Int8:Int64} = {23423:6, 1:5}
			}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidIntegerLiteralRangeError{}, errs[0])
		intRangeErr := errs[0].(*sema.InvalidIntegerLiteralRangeError)

		assert.Equal(t, sema.Int8Type, intRangeErr.ExpectedType)
		assert.Equal(t, sema.Int8Type.MinInt(), intRangeErr.ExpectedMinInt)
		assert.Equal(t, sema.Int8Type.MaxInt(), intRangeErr.ExpectedMaxInt)
	})

	t.Run("nested invalid", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: {Int:{Int:{Int:Int8}}} = {0: {0: {0: "hello"}, 1: {0: 7}}}
			}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchErr := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.Int8Type, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchErr.ActualType)
	})
}

func TestReturnTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("array type", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test(): [Int8] {
				return [1, 2, 3]
			}
		`)

		require.NoError(t, err)
	})

	t.Run("void", func(t *testing.T) {
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

func TestFunctionArgumentTypeInference(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		fun test() {
			foo(a: [1, 2, 3])
		}

		fun foo(a: [Int8]) {
		}
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

func TestBinaryExpressionTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("integer add", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: Int8 = 5 + 6
			}
		`)

		require.NoError(t, err)
	})

	t.Run("fixed point add", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: Fix64 = 5.0 + 6.0
			}
		`)

		require.NoError(t, err)
	})
}

func TestUnaryExpressionTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("invalid negate", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var b : Bool =  !"string"
			}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidUnaryOperandError{}, errs[0])
		invalidUnaryOpKindErr := errs[0].(*sema.InvalidUnaryOperandError)

		assert.Equal(t, sema.BoolType, invalidUnaryOpKindErr.ExpectedType)
		assert.Equal(t, sema.StringType, invalidUnaryOpKindErr.ActualType)
	})
}

func TestForceExpressionTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("array forced", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: [Int8] = [5, 7, 2]!
			}
		`)

		require.NoError(t, err)
	})

	t.Run("double-optional repeated forced", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: Int?? = 4
				var y: Int = x!!
			}
		`)

		require.NoError(t, err)
	})

	t.Run("optional repeated forced", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: Int? = 4
				var y: Int = x!!
			}
		`)

		require.NoError(t, err)
	})

	t.Run("non-optional repeated forced", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: Int = 4
				var y: Int = x!!
			}
		`)

		require.NoError(t, err)
	})
}

func TestCastExpressionTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("array", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x = [1, 3] as [Int8]
			}
		`)

		require.NoError(t, err)
	})

	t.Run("number out of range", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x = [1, 764] as [Int8]
			}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidIntegerLiteralRangeError{}, errs[0])
		intRangeErr := errs[0].(*sema.InvalidIntegerLiteralRangeError)

		assert.Equal(t, sema.Int8Type, intRangeErr.ExpectedType)
		assert.Equal(t, sema.Int8Type.MinInt(), intRangeErr.ExpectedMinInt)
		assert.Equal(t, sema.Int8Type.MaxInt(), intRangeErr.ExpectedMaxInt)
	})

	t.Run("mismatching types", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x = [1, "hello"] as [Int8]
			}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchErr := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.Int8Type, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchErr.ActualType)
	})
}

func TestVoidTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("void type annotation", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			fun test() {
				var x: Void = 5 + 6
			}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchErr := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.VoidType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.IntType, typeMismatchErr.ActualType)
	})
}

func TestInferenceWithCheckerErrors(t *testing.T) {

	t.Parallel()

	t.Run("undefined type reference", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
			pub struct Foo {
				pub var ownedNFTs: @{Int: UnknownType}

				init() {
					self.ownedNFTs = {}
				}

				pub fun borrowNFT(id: Int): &UnknownType {
					return &self.ownedNFTs[id] as &UnknownType
				}
			}
		`)

		errs := ExpectCheckerErrors(t, err, 4)

		require.IsType(t, &sema.NotDeclaredError{}, errs[0])
		notDeclaredError := errs[0].(*sema.NotDeclaredError)
		assert.Equal(t, "UnknownType", notDeclaredError.Name)

		require.IsType(t, &sema.NotDeclaredError{}, errs[1])
		notDeclaredError = errs[1].(*sema.NotDeclaredError)
		assert.Equal(t, "UnknownType", notDeclaredError.Name)

		require.IsType(t, &sema.NotDeclaredError{}, errs[2])
		notDeclaredError = errs[2].(*sema.NotDeclaredError)
		assert.Equal(t, "UnknownType", notDeclaredError.Name)

		require.IsType(t, &sema.NotDeclaredError{}, errs[3])
		notDeclaredError = errs[3].(*sema.NotDeclaredError)
		assert.Equal(t, "UnknownType", notDeclaredError.Name)

	})
}
