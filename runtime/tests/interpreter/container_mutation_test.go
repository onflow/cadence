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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"

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
				interpreter.NewUnmeteredStringValue("baz"),
				interpreter.NewUnmeteredStringValue("bar"),
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
		assert.Equal(t, sema.IntType, mutationError.ActualType)
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
		assert.Equal(t, sema.IntType, mutationError.ActualType)
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
				interpreter.NewUnmeteredStringValue("foo"),
				interpreter.NewUnmeteredStringValue("bar"),
				interpreter.NewUnmeteredStringValue("baz"),
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
		assert.Equal(t, sema.IntType, mutationError.ActualType)
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
		assert.Equal(t, sema.IntType, mutationError.ActualType)
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
				interpreter.NewUnmeteredStringValue("foo"),
				interpreter.NewUnmeteredStringValue("baz"),
				interpreter.NewUnmeteredStringValue("bar"),
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
		assert.Equal(t, sema.IntType, mutationError.ActualType)
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
				interpreter.NewUnmeteredStringValue("foo"),
				interpreter.NewUnmeteredStringValue("bar"),
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
		assert.Equal(t, sema.IntType, mutationError.ActualType)
	})

	t.Run("host function mutation", func(t *testing.T) {
		t.Parallel()

		invoked := false

		standardLibraryFunctions :=
			stdlib.StandardLibraryFunctions{
				stdlib.NewStandardLibraryFunction(
					"log",
					stdlib.LogFunctionType,
					"",
					func(invocation interpreter.Invocation) interpreter.Value {
						invoked = true
						assert.Equal(t, "\"hello\"", invocation.Arguments[0].String())
						return interpreter.VoidValue{}
					},
				),
			}

		valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
		values := standardLibraryFunctions.ToInterpreterValueDeclarations()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun test() {
                let array: [AnyStruct] = [nil] as [((AnyStruct):Void)?]

                array[0] = log

                let logger = array[0] as! ((AnyStruct):Void)
                logger("hello")
            }`,
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
		require.NoError(t, err)

		assert.True(t, invoked)
	})

	t.Run("function mutation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): [String] {
                let array: [AnyStruct] = [nil, nil] as [(():String)?]

                array[0] = foo
                array[1] = bar

                let callFoo = array[0] as! (():String)
                let callBar = array[1] as! (():String)
                return [callFoo(), callBar()]
            }

            fun foo(): String {
                return "hello from foo"
            }

            fun bar(): String {
                return "hello from bar"
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, value)
		array := value.(*interpreter.ArrayValue)

		require.Equal(t, 2, array.Count())
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from foo"),
			array.Get(inter, interpreter.ReturnEmptyLocationRange, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, interpreter.ReturnEmptyLocationRange, 1),
		)
	})

	t.Run("bound function mutation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct Foo {
                fun foo(): String {
                    return "hello from foo"
                }
            }

            struct Bar {
                fun bar(): String {
                    return "hello from bar"
                }
            }

            fun test(): [String] {
                let array: [AnyStruct] = [nil, nil] as [(():String)?]

                let a = Foo()
                let b = Bar()

                array[0] = a.foo
                array[1] = b.bar

                let callFoo = array[0] as! (():String)
                let callBar = array[1] as! (():String)

                return [callFoo(), callBar()]
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, value)
		array := value.(*interpreter.ArrayValue)

		require.Equal(t, 2, array.Count())
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from foo"),
			array.Get(inter, interpreter.ReturnEmptyLocationRange, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, interpreter.ReturnEmptyLocationRange, 1),
		)
	})

	t.Run("invalid function mutation", func(t *testing.T) {
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

                    array[0] = log
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

		// Expected type
		require.IsType(t, &sema.OptionalType{}, mutationError.ExpectedType)
		optionalType := mutationError.ExpectedType.(*sema.OptionalType)

		require.IsType(t, &sema.FunctionType{}, optionalType.Type)
		funcType := optionalType.Type.(*sema.FunctionType)

		assert.Equal(t, sema.VoidType, funcType.ReturnTypeAnnotation.Type)
		assert.Empty(t, funcType.Parameters)

		// Actual type
		assert.IsType(t, &sema.FunctionType{}, mutationError.ActualType)
		actualFuncType := mutationError.ActualType.(*sema.FunctionType)

		assert.Equal(t, sema.VoidType, actualFuncType.ReturnTypeAnnotation.Type)
		assert.Len(t, actualFuncType.Parameters, 1)
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
			interpreter.NewUnmeteredStringValue("foo"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("baz"), val)
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

		assert.Equal(t,
			&sema.OptionalType{
				Type: sema.IntType,
			},
			mutationError.ActualType,
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
			interpreter.NewUnmeteredStringValue("foo"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("baz"), val)
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
		assert.Equal(t, sema.IntType, mutationError.ActualType)
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
		assert.Equal(t, sema.PrivatePathType, mutationError.ActualType)
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
		assert.Equal(t,
			&sema.OptionalType{
				Type: sema.IntType,
			},
			mutationError.ActualType,
		)
	})

	t.Run("host function mutation", func(t *testing.T) {

		t.Parallel()

		invoked := false

		standardLibraryFunctions :=
			stdlib.StandardLibraryFunctions{
				stdlib.NewStandardLibraryFunction(
					"log",
					stdlib.LogFunctionType,
					"",
					func(invocation interpreter.Invocation) interpreter.Value {
						invoked = true
						assert.Equal(t, "\"hello\"", invocation.Arguments[0].String())
						return interpreter.VoidValue{}
					},
				),
			}

		valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
		values := standardLibraryFunctions.ToInterpreterValueDeclarations()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun test() {
                let dict: {String: AnyStruct} = {}

                dict["test"] = log

                let logger = dict["test"]! as! ((AnyStruct): Void)
                logger("hello")
            }`,
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
		require.NoError(t, err)

		assert.True(t, invoked)
	})

	t.Run("function mutation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           fun test(): [String] {
               let dict: {String: AnyStruct} = {}

               dict["foo"] = foo
               dict["bar"] = bar

               let callFoo = dict["foo"]! as! (():String)
               let callBar = dict["bar"]! as! (():String)
               return [callFoo(), callBar()]
           }

           fun foo(): String {
               return "hello from foo"
           }

           fun bar(): String {
               return "hello from bar"
           }
       `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, value)
		array := value.(*interpreter.ArrayValue)

		require.Equal(t, 2, array.Count())
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from foo"),
			array.Get(inter, interpreter.ReturnEmptyLocationRange, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, interpreter.ReturnEmptyLocationRange, 1),
		)
	})

	t.Run("bound function mutation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           struct Foo {
               fun foo(): String {
                   return "hello from foo"
               }
           }

           struct Bar {
               fun bar(): String {
                   return "hello from bar"
               }
           }

           fun test(): [String] {
               let dict: {String: AnyStruct} = {}

               let a = Foo()
               let b = Bar()

               dict["foo"] = a.foo
               dict["bar"] = b.bar

               let callFoo = dict["foo"]! as! (():String)
               let callBar = dict["bar"]! as! (():String)

               return [callFoo(), callBar()]
           }
       `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, value)
		array := value.(*interpreter.ArrayValue)

		require.Equal(t, 2, array.Count())
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from foo"),
			array.Get(inter, interpreter.ReturnEmptyLocationRange, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, interpreter.ReturnEmptyLocationRange, 1),
		)
	})

	t.Run("invalid function mutation", func(t *testing.T) {
		t.Parallel()

		standardLibraryFunctions :=
			stdlib.StandardLibraryFunctions{
				stdlib.LogFunction,
			}

		valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
		values := standardLibraryFunctions.ToInterpreterValueDeclarations()

		inter, err := parseCheckAndInterpretWithOptions(t, `
               fun test() {
                   let dict: {String: AnyStruct} = {} as {String: (():Void)}

                   dict["log"] = log
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

		// Expected type
		require.IsType(t, &sema.OptionalType{}, mutationError.ExpectedType)
		optionalType := mutationError.ExpectedType.(*sema.OptionalType)

		require.IsType(t, &sema.FunctionType{}, optionalType.Type)
		funcType := optionalType.Type.(*sema.FunctionType)

		assert.Equal(t, sema.VoidType, funcType.ReturnTypeAnnotation.Type)
		assert.Empty(t, funcType.Parameters)

		// Actual type
		require.IsType(t, &sema.OptionalType{}, mutationError.ActualType)
		actualOptionalType := mutationError.ActualType.(*sema.OptionalType)

		require.IsType(t, &sema.FunctionType{}, actualOptionalType.Type)
		actualFuncType := actualOptionalType.Type.(*sema.FunctionType)

		assert.Equal(t, sema.VoidType, actualFuncType.ReturnTypeAnnotation.Type)
		assert.Len(t, actualFuncType.Parameters, 1)
	})

	t.Run("valid function mutation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}

            fun test(owner: PublicAccount) {
                let funcs: {String: ((PublicAccount, [UInt64]): [S])} = {}

                funcs["test"] = fun (owner: PublicAccount, ids: [UInt64]): [S] { return [] }

                funcs["test"]!(owner: owner, ids: [1])
            }
        `)

		owner := newTestPublicAccountValue(
			inter,
			interpreter.NewUnmeteredAddressValueFromBytes(common.Address{0x1}.Bytes()),
		)

		_, err := inter.Invoke("test", owner)
		require.NoError(t, err)

	})
}

func TestInterpretContainerMutationAfterNilCoalescing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): String? {
          let xs: {UInt32: String}? = nil
          let ys: {UInt32: String} = xs ?? {}
          ys[0] = "test"
          return ys[0]
      }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	utils.RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("test"),
		),
		result,
	)
}
