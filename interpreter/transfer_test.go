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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretTransferCheck(t *testing.T) {

	t.Parallel()

	t.Run("String value as composite", func(t *testing.T) {

		t.Parallel()

		ty := &sema.CompositeType{
			Location:   TestLocation,
			Identifier: "Fruit",
			Kind:       common.CompositeKindStructure,
		}

		valueDeclaration := stdlib.StandardLibraryValue{
			Name: "fruit",
			Type: ty,
			// NOTE: not an instance of the type
			Value: interpreter.NewUnmeteredStringValue("fruit"),
			Kind:  common.DeclarationKindConstant,
		}

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)

		baseValueActivation.DeclareValue(valueDeclaration)

		baseTypeActivation := sema.NewVariableActivation(sema.BaseTypeActivation)
		baseTypeActivation.DeclareType(stdlib.StandardLibraryType{
			Name: ty.Identifier,
			Type: ty,
			Kind: common.DeclarationKindStructure,
		})

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndPrepareWithOptions(t,
			`
              fun test() {
                  let alsoFruit: Fruit = fruit
              }
            `,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseTypeActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseTypeActivation
						},
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		var transferTypeError *interpreter.ValueTransferTypeError
		require.ErrorAs(t, err, &transferTypeError)
	})

	t.Run("contract and intersection type", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndPrepareWithOptions(t,
			`
		      contract interface CI {
		          resource interface RI {}
		      }

              contract C: CI {
		          resource R: CI.RI {}

		          fun createR(): @R {
		              return <- create R()
		          }
		      }

              fun test() {
                  let r <- C.createR()
                  let r2: @C.R <- r as @C.R
                  let r3: @{CI.RI} <- r2
                  destroy r3
              }
            `,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("contract and intersection type, reference", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndPrepareWithOptions(t,
			`
		      contract interface CI {
		          resource interface RI {}
		      }

              contract C: CI {
		          resource R: CI.RI {}

		          fun createR(): @R {
		              return <- create R()
		          }
		      }

              fun test() {
                  let r <- C.createR()
                  let ref: &C.R = &r as &C.R
                  let intersectionRef: &{CI.RI} = ref
                  destroy r
              }
            `,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})
}

func TestInterpretConversionOnTransfer(t *testing.T) {

	t.Parallel()

	t.Run("array literal without type annotation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t,
			`
              resource R {}

              fun test(): @[R?] {
                  // No type annotation: type is inferred based on each expression's type.
                  // So the unary-move '<- create R()' has an expected type of 'R' (not 'R?').
                  let array <- [ <- create R(), nil]
                  return <- array
              }
            `,
		)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, result)
		array := result.(*interpreter.ArrayValue)

		require.Equal(t, 2, array.Count())

		// Elements must be boxed.

		element := array.Get(nil, interpreter.EmptyLocationRange, 0)

		require.IsType(t, &interpreter.SomeValue{}, element)
		someValue := element.(*interpreter.SomeValue)

		innerValue := someValue.InnerValue()
		require.IsType(t, &interpreter.CompositeValue{}, innerValue)
		composite := innerValue.(*interpreter.CompositeValue)

		assert.Equal(t, common.TypeID("S.test.R"), composite.TypeID())
	})

	t.Run("array literal with type annotation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t,
			`
              resource R {}

              fun test(): @[R?] {
                  let array: @[R?] <- [ <- create R(), nil]
                  return <- array
              }
            `,
		)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, result)
		array := result.(*interpreter.ArrayValue)

		require.Equal(t, 2, array.Count())

		// Elements must be boxed.

		element := array.Get(nil, interpreter.EmptyLocationRange, 0)

		require.IsType(t, &interpreter.SomeValue{}, element)
		someValue := element.(*interpreter.SomeValue)

		innerValue := someValue.InnerValue()
		require.IsType(t, &interpreter.CompositeValue{}, innerValue)
		composite := innerValue.(*interpreter.CompositeValue)

		assert.Equal(t, common.TypeID("S.test.R"), composite.TypeID())
	})

	t.Run("dictionary literal without type annotations", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t,
			`
              resource R {}

              fun test(): @{Int: R?} {
                  // No type annotation: type is inferred based on each expression's type.
                  // So the unary-move '<- create R()' has an expected type of 'R' (not 'R?').
                  let dictionary <- { 1: <- create R(), 2: nil}
                  return <- dictionary
              }
            `,
		)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.DictionaryValue{}, result)
		dictionary := result.(*interpreter.DictionaryValue)

		require.Equal(t, 2, dictionary.Count())

		// Elements must be boxed.

		// Use 'Get' to get the value as-is.
		// 'GetKey' method would explicitly box the value before returning.
		element, ok := dictionary.Get(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.NewIntValueFromInt64(nil, 1),
		)

		require.True(t, ok)

		require.IsType(t, &interpreter.SomeValue{}, element)
		someValue := element.(*interpreter.SomeValue)

		innerValue := someValue.InnerValue()
		require.IsType(t, &interpreter.CompositeValue{}, innerValue)
		composite := innerValue.(*interpreter.CompositeValue)

		assert.Equal(t, common.TypeID("S.test.R"), composite.TypeID())
	})

	t.Run("dictionary literal with type annotation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t,
			`
              resource R {}

              fun test(): @{Int: R?} {
                  let dictionary: @{Int: R?} <- { 1: <- create R(), 2: nil}
                  return <- dictionary
              }
            `,
		)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.DictionaryValue{}, result)
		dictionary := result.(*interpreter.DictionaryValue)

		require.Equal(t, 2, dictionary.Count())

		// Elements must be boxed.

		// Use 'Get' to get the value as-is.
		// 'GetKey' method would explicitly box the value before returning.
		element, ok := dictionary.Get(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.NewIntValueFromInt64(nil, 1),
		)

		require.True(t, ok)

		require.IsType(t, &interpreter.SomeValue{}, element)
		someValue := element.(*interpreter.SomeValue)

		innerValue := someValue.InnerValue()
		require.IsType(t, &interpreter.CompositeValue{}, innerValue)
		composite := innerValue.(*interpreter.CompositeValue)

		assert.Equal(t, common.TypeID("S.test.R"), composite.TypeID())
	})
}
