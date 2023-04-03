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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
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

		errs := RequireCheckerErrors(t, err, 1)

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

		errs := RequireCheckerErrors(t, err, 1)

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

		errs := RequireCheckerErrors(t, err, 1)

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

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchErr := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.VoidType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.IntType, typeMismatchErr.ActualType)
	})
}

func TestCheckFunctionArgumentTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("required args", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            let x = foo(a: [1, 2, 3])

            fun foo(a: [Int8]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("with generics", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test<[Int8]>([1, 2, 3])
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				Parameters: []sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "value",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
				},
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
		)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		typeParamMismatchErr := errs[0].(*sema.TypeParameterTypeMismatchError)
		assert.Equal(
			t,
			&sema.VariableSizedType{
				Type: sema.Int8Type,
			},
			typeParamMismatchErr.ExpectedType,
		)

		assert.Equal(
			t,
			&sema.VariableSizedType{
				Type: sema.IntType,
			},
			typeParamMismatchErr.ActualType,
		)

		require.IsType(t, &sema.TypeMismatchError{}, errs[1])
		typeMismatchErr := errs[1].(*sema.TypeMismatchError)

		assert.Equal(
			t,
			&sema.VariableSizedType{
				Type: sema.Int8Type,
			},
			typeMismatchErr.ExpectedType,
		)

		assert.Equal(
			t,
			&sema.VariableSizedType{
				Type: sema.IntType,
			},
			typeMismatchErr.ActualType,
		)

	})
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

		errs := RequireCheckerErrors(t, err, 1)

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

		errs := RequireCheckerErrors(t, err, 1)

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

		errs := RequireCheckerErrors(t, err, 1)

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

		errs := RequireCheckerErrors(t, err, 1)

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

		errs := RequireCheckerErrors(t, err, 3)

		for _, err := range errs {
			require.IsType(t, &sema.NotDeclaredError{}, err)
			notDeclaredError := err.(*sema.NotDeclaredError)
			assert.Equal(t, "UnknownType", notDeclaredError.Name)
		}
	})
}

func TestCheckArraySupertypeInference(t *testing.T) {

	t.Parallel()

	t.Run("has supertype", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name                string
			code                string
			expectedElementType sema.Type
		}{
			{
				name:                "mixed simple values",
				code:                `let x = [0, true]`,
				expectedElementType: sema.AnyStructType,
			},
			{
				name:                "signed integer values",
				code:                `let x = [0, 6, 275]`,
				expectedElementType: sema.IntType,
			},
			{
				name:                "signed and unsigned integer values",
				code:                `let x = [UInt(65), 6, 275, 13423]`,
				expectedElementType: sema.IntegerType,
			},
			{
				name:                "unsigned integers values",
				code:                `let x = [UInt(0), UInt(6), UInt(275), UInt(13423)]`,
				expectedElementType: sema.UIntType,
			},
			{
				name: "values with nil",
				code: `let x = ["hello", nil, nil, nil]`,
				expectedElementType: &sema.OptionalType{
					Type: sema.StringType,
				},
			},
			{
				name: "common interfaced values",
				code: `
                    let x = [Foo(), Bar(), Baz()]

                    pub struct interface I1 {}

                    pub struct interface I2 {}

                    pub struct interface I3 {}

                    pub struct Foo: I1, I2 {}

                    pub struct Bar: I2, I3 {}

                    pub struct Baz: I1, I2, I3 {}
                `,
				expectedElementType: &sema.RestrictedType{
					Type: sema.AnyStructType,
					Restrictions: []*sema.InterfaceType{
						{
							Location:      common.StringLocation("test"),
							Identifier:    "I2",
							CompositeKind: common.CompositeKindStructure,
						},
					},
				},
			},
			{
				name: "implicit covariant to interface",
				code: `
                    let x = [[Bar()], [Baz()]]

                    pub struct interface Foo {}

                    pub struct Bar: Foo {}

                    pub struct Baz: Foo {}
                `,
				expectedElementType: &sema.VariableSizedType{
					Type: &sema.RestrictedType{
						Type: sema.AnyStructType,
						Restrictions: []*sema.InterfaceType{
							{
								Location:      common.StringLocation("test"),
								Identifier:    "Foo",
								CompositeKind: common.CompositeKindStructure,
							},
						},
					},
				},
			},
			{
				name: "explicit covariant to interface",
				code: `
                    // Covariance is supported with explicit type annotation.
                    let x = [[Bar()], [Baz()]] as [[{Foo}]]

                    pub struct interface Foo {}

                    pub struct Bar: Foo {}

                    pub struct Baz: Foo {}
                `,
				expectedElementType: &sema.VariableSizedType{
					Type: &sema.RestrictedType{
						Type: sema.AnyStructType,
						Restrictions: []*sema.InterfaceType{
							{
								Location:      common.StringLocation("test"),
								Identifier:    "Foo",
								CompositeKind: common.CompositeKindStructure,
							},
						},
					},
				},
			},
			{
				name: "nested covariant var sized",
				code: `let x = [[[1, 2]], [["foo", "bar"]], [[5.3, 6.4]]]`,
				expectedElementType: &sema.VariableSizedType{
					Type: &sema.VariableSizedType{
						Type: sema.AnyStructType,
					},
				},
			},
			{
				name: "nested covariant constant sized",
				code: `let x = [[[1, 2] as [Int; 2]], [["foo", "bar"] as [String; 2]], [[5.3, 6.4] as [Fix64; 2]]]`,
				expectedElementType: &sema.VariableSizedType{
					Type: &sema.ConstantSizedType{
						Type: sema.AnyStructType,
						Size: 2,
					},
				},
			},
			{
				name: "nested non-covariant constant sized",
				code: `let x = [[[1, 2] as [Int; 2]], [["foo", "bar", "baz"] as [String; 3]], [[5.3, 6.4] as [Fix64; 2]]]`,
				expectedElementType: &sema.VariableSizedType{
					Type: sema.AnyStructType,
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				checker, err := ParseAndCheck(t, test.code)
				require.NoError(t, err)

				xType := RequireGlobalValue(t, checker.Elaboration, "x")

				require.IsType(t, &sema.VariableSizedType{}, xType)
				arrayType := xType.(*sema.VariableSizedType)

				assert.Equal(t, test.expectedElementType.ID(), arrayType.Type.ID())
			})
		}
	})

	t.Run("no supertype", func(t *testing.T) {
		t.Parallel()

		code := `
            let x = [<- create Foo(), Bar()]

            pub resource Foo {}

            pub struct Bar {}
        `
		_, err := ParseAndCheck(t, code)
		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeAnnotationRequiredError{}, errs[0])
	})

	t.Run("empty array", func(t *testing.T) {
		t.Parallel()

		code := `
            let x = [].getType()
        `
		_, err := ParseAndCheck(t, code)
		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeAnnotationRequiredError{}, errs[0])
	})
}

func TestCheckDictionarySupertypeInference(t *testing.T) {

	t.Parallel()

	t.Run("has supertype", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name              string
			code              string
			expectedKeyType   sema.Type
			expectedValueType sema.Type
		}{
			{
				name:              "mixed simple values",
				code:              `let x = {0: 0, 1: true}`,
				expectedKeyType:   sema.IntType,
				expectedValueType: sema.AnyStructType,
			},
			{
				name:              "signed integer values",
				code:              `let x = {0: 0, 1: 6, 2: 275}`,
				expectedKeyType:   sema.IntType,
				expectedValueType: sema.IntType,
			},
			{
				name:              "signed and unsigned integer values",
				code:              `let x = {0: UInt(65), 1: 6, 2: 275, 3: 13423}`,
				expectedKeyType:   sema.IntType,
				expectedValueType: sema.IntegerType,
			},
			{
				name:              "unsigned integers values",
				code:              `let x = {0: UInt(0), 1: UInt(6), 2: UInt(275), 3: UInt(13423)}`,
				expectedKeyType:   sema.IntType,
				expectedValueType: sema.UIntType,
			},
			{
				name:              "unsigned integers keys",
				code:              `let x = {UInt(0): true, UInt(6): false, UInt(275): false, UInt(13423): true}`,
				expectedKeyType:   sema.UIntType,
				expectedValueType: sema.BoolType,
			},
			{
				name:            "values with nil",
				code:            `let x = {0: "hello", 1: nil, 2: nil, 2: nil}`,
				expectedKeyType: sema.IntType,
				expectedValueType: &sema.OptionalType{
					Type: sema.StringType,
				},
			},
			{
				name: "common interfaced values",
				code: `
                    let x = {0: Foo(), 1: Bar(), 2: Baz()}

                    pub struct interface I1 {}

                    pub struct interface I2 {}

                    pub struct interface I3 {}

                    pub struct Foo: I1, I2 {}

                    pub struct Bar: I2, I3 {}

                    pub struct Baz: I1, I2, I3 {}
                `,
				expectedKeyType: sema.IntType,
				expectedValueType: &sema.RestrictedType{
					Type: sema.AnyStructType,
					Restrictions: []*sema.InterfaceType{
						{
							Location:      common.StringLocation("test"),
							Identifier:    "I2",
							CompositeKind: common.CompositeKindStructure,
						},
					},
				},
			},
			{
				name: "implicit covariant values",
				code: `
                    let x = { 0: {100: Bar()}, 1: {200: Baz()} }

                    pub struct interface Foo {}

                    pub struct Bar: Foo {}

                    pub struct Baz: Foo {}
                `,
				expectedKeyType: sema.IntType,
				expectedValueType: &sema.DictionaryType{
					KeyType: sema.IntType,
					ValueType: &sema.RestrictedType{
						Type: sema.AnyStructType,
						Restrictions: []*sema.InterfaceType{
							{
								Location:      common.StringLocation("test"),
								Identifier:    "Foo",
								CompositeKind: common.CompositeKindStructure,
							},
						},
					},
				},
			},
			{
				name: "explicit covariant values",
				code: `
                    // Covariance is supported with explicit type annotation.
                    let x = { 0: {100: Bar()}, 1: {200: Baz()} } as {Int: {Int: {Foo}}}

                    pub struct interface Foo {}

                    pub struct Bar: Foo {}

                    pub struct Baz: Foo {}
                `,
				expectedKeyType: sema.IntType,
				expectedValueType: &sema.DictionaryType{
					KeyType: sema.IntType,
					ValueType: &sema.RestrictedType{
						Type: sema.AnyStructType,
						Restrictions: []*sema.InterfaceType{
							{
								Location:      common.StringLocation("test"),
								Identifier:    "Foo",
								CompositeKind: common.CompositeKindStructure,
							},
						},
					},
				},
			},
			{
				name:              "no supertype for inner keys",
				code:              `let x = {0: {10: 1, 20: 2}, 1: {"one": 1, "two": 2}}`,
				expectedKeyType:   sema.IntType,
				expectedValueType: sema.AnyStructType,
			},
			{
				name: "no supertype for inner keys with resource values",
				code: `
                    let x <- {0: <- {10: <- create Foo()}, 1: <- {"one": <- create Foo()}}

                    pub resource Foo {}
                `,
				expectedKeyType:   sema.IntType,
				expectedValueType: sema.AnyResourceType,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				checker, err := ParseAndCheck(t, test.code)
				require.NoError(t, err)

				xType := RequireGlobalValue(t, checker.Elaboration, "x")

				require.IsType(t, &sema.DictionaryType{}, xType)
				dictionaryType := xType.(*sema.DictionaryType)

				assert.Equal(t, test.expectedKeyType.ID(), dictionaryType.KeyType.ID())
				assert.Equal(t, test.expectedValueType.ID(), dictionaryType.ValueType.ID())
			})
		}
	})

	t.Run("no supertype for values", func(t *testing.T) {
		t.Parallel()

		code := `
            let x = {0: <- create Foo(), 1: Bar()}

            pub resource Foo {}

            pub struct Bar {}
        `
		_, err := ParseAndCheck(t, code)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeAnnotationRequiredError{}, errs[0])
	})

	t.Run("no supertype for keys", func(t *testing.T) {
		t.Parallel()

		code := `
            let x = {1: 1, "two": 2}
        `
		_, err := ParseAndCheck(t, code)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
	})

	t.Run("unsupported supertype for keys", func(t *testing.T) {
		t.Parallel()

		code := `
            let x = {0: 1, "hello": 2}
        `
		_, err := ParseAndCheck(t, code)
		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
		invalidKeyError := errs[0].(*sema.InvalidDictionaryKeyTypeError)

		assert.Equal(t, sema.AnyStructType, invalidKeyError.Type)
	})

	t.Run("empty dictionary", func(t *testing.T) {
		t.Parallel()

		code := `
            let x = {}.getType()
        `
		_, err := ParseAndCheck(t, code)
		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeAnnotationRequiredError{}, errs[0])
	})
}

func TestCheckTypeInferenceForTypesWithDifferentTypeMaskRanges(t *testing.T) {

	t.Parallel()

	t.Run("array expression", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            let x: @AnyResource{Foo} <- create Bar()
            let y = [<-x, 6]

            resource interface Foo {}

            resource Bar: Foo {}
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeAnnotationRequiredError{}, errs[0])
	})

	t.Run("conditional expression", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            let x: AnyStruct{Foo} = Bar()
            let y = true ? x : nil

            struct interface Foo {}

            struct Bar: Foo {}
        `)

		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "y")
		require.IsType(t, &sema.OptionalType{Type: sema.AnyStructType}, xType)
	})
}
