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

package interpreter_test

import (
	"testing"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestArrayMutation(t *testing.T) {

	t.Parallel()

	t.Run("simple array valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): [String] {
                let names: [String] = ["foo", "bar"]
                names[0] = "baz"
                return names
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, value)
		array := value.(*interpreter.ArrayValue)

		utils.RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.Address{},
				interpreter.NewStringValue("baz"),
				interpreter.NewStringValue("bar"),
			),
			array,
		)
	})

	t.Run("simple array invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names[0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("nested array invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [[AnyStruct]] = [["foo", "bar"]] as [[String]]
                names[0][0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("array append valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): [AnyStruct] {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.append("baz")
                return names
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, value)
		array := value.(*interpreter.ArrayValue)

		utils.RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.Address{},
				interpreter.NewStringValue("foo"),
				interpreter.NewStringValue("bar"),
				interpreter.NewStringValue("baz"),
			),
			array,
		)
	})

	t.Run("array append invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.append(5)
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("array appendAll invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.appendAll(["baz", 5] as [AnyStruct])
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("array insert valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): [AnyStruct] {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.insert(at: 1, "baz")
                return names
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, value)
		array := value.(*interpreter.ArrayValue)

		utils.RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.Address{},
				interpreter.NewStringValue("foo"),
				interpreter.NewStringValue("baz"),
				interpreter.NewStringValue("bar"),
			),
			array,
		)
	})

	t.Run("array insert invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.insert(at: 1, 4)
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("array concat mismatching values", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let names: [AnyStruct] = ["foo", "bar"] as [String]
            
            fun test(): [AnyStruct] {
                return names.concat(["baz", 5] as [AnyStruct])
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		require.ErrorAs(t, err, &interpreter.ContainerMutationError{})

		// Check original array

		namesVal := inter.Globals["names"].GetValue()
		require.IsType(t, &interpreter.ArrayValue{}, namesVal)
		namesValArray := namesVal.(*interpreter.ArrayValue)

		utils.RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.Address{},
				interpreter.NewStringValue("foo"),
				interpreter.NewStringValue("bar"),
			),
			namesValArray,
		)
	})

	t.Run("invalid update through reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                let namesRef = &names as &[AnyStruct]
                namesRef[0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("function array mutation", func(t *testing.T) {
		t.Parallel()

		standardLibraryFunctions :=
			stdlib.StandardLibraryFunctions{
				stdlib.LogFunction,
			}

		valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
		values := standardLibraryFunctions.ToInterpreterValueDeclarations()

		inter, err := parseCheckAndInterpretWithOptions(t, `
                fun test() {
                    let array: [AnyStruct] = [nil] as [((AnyStruct):Void)?]

                    let x = 5
                    array[0] =  log
                }
            `,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(valueDeclarations),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(values),
				},
			},
		)

		require.NoError(t, err)

		// TODO: Shouldn't throw an error once dynamic subtyping for functions is implemented.
		_, err = inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		require.IsType(t, &sema.OptionalType{}, mutationError.ExpectedType)
		optionalType := mutationError.ExpectedType.(*sema.OptionalType)

		require.IsType(t, &sema.FunctionType{}, optionalType.Type)
		funcType := optionalType.Type.(*sema.FunctionType)

		assert.Equal(t, sema.VoidType, funcType.ReturnTypeAnnotation.Type)
		assert.Nil(t, funcType.ReceiverType)
		assert.Len(t, funcType.Parameters, 1)
		assert.Equal(t, sema.AnyStructType, funcType.Parameters[0].TypeAnnotation.Type)
	})

	t.Run("invalid function array mutation", func(t *testing.T) {
		t.Parallel()

		standardLibraryFunctions :=
			stdlib.StandardLibraryFunctions{
				stdlib.LogFunction,
			}

		valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
		values := standardLibraryFunctions.ToInterpreterValueDeclarations()

		inter, err := parseCheckAndInterpretWithOptions(t, `
                fun test() {
                    let array: [AnyStruct] = [nil] as [(():Void)?]

                    let x = 5
                    array[0] =  log
                }
            `,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(valueDeclarations),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(values),
				},
			},
		)

		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		require.IsType(t, &sema.OptionalType{}, mutationError.ExpectedType)
		optionalType := mutationError.ExpectedType.(*sema.OptionalType)

		require.IsType(t, &sema.FunctionType{}, optionalType.Type)
		funcType := optionalType.Type.(*sema.FunctionType)

		assert.Equal(t, sema.VoidType, funcType.ReturnTypeAnnotation.Type)
		assert.Nil(t, funcType.ReceiverType)
		assert.Empty(t, funcType.Parameters)
	})
}

func TestDictionaryMutation(t *testing.T) {

	t.Parallel()

	t.Run("simple dictionary valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): {String: String} {
                let names: {String: String} = {"foo": "bar"}
                names["foo"] = "baz"
                return names
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.DictionaryValue{}, value)
		dictionary := value.(*interpreter.DictionaryValue)

		require.Equal(t, 1, dictionary.Count())

		val, present := dictionary.Get(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.NewStringValue("foo"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewStringValue("baz"), val)
	})

	t.Run("simple dictionary invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                names["foo"] = 5
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t,
			&sema.OptionalType{
				Type: sema.StringType,
			},
			mutationError.ExpectedType,
		)
	})

	t.Run("optional dictionary valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): {String: String?} {
                let names: {String: String?} = {"foo": "bar"}
                names["foo"] = nil
                return names
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.DictionaryValue{}, value)
		dictionary := value.(*interpreter.DictionaryValue)

		require.Equal(t, 0, dictionary.Count())
	})

	t.Run("dictionary insert valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): {String: AnyStruct} {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                names.insert(key: "foo", "baz")
                return names
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.DictionaryValue{}, value)
		dictionary := value.(*interpreter.DictionaryValue)

		require.Equal(t, 1, dictionary.Count())

		val, present := dictionary.Get(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.NewStringValue("foo"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewStringValue("baz"), val)
	})

	t.Run("dictionary insert invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                names.insert(key: "foo", 5)
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("dictionary insert invalid key", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {Path: AnyStruct} = {/public/path: "foo"} as {PublicPath: String}
                names.insert(key: /private/path, "bar")
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.PublicPathType, mutationError.ExpectedType)
	})

	t.Run("invalid update through reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                let namesRef = &names as &{String: AnyStruct}
                namesRef["foo"] = 5
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t,
			&sema.OptionalType{
				Type: sema.StringType,
			},
			mutationError.ExpectedType,
		)
	})
}
