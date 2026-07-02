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
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

// native helpers
func newAssertHelloLogFunction(t *testing.T, invoked *bool) stdlib.StandardLibraryValue {
	return stdlib.NewInterpreterStandardLibraryStaticFunction(
		"log",
		stdlib.LogFunctionType,
		"",
		func(
			_ interpreter.NativeFunctionContext,
			_ interpreter.TypeArgumentsIterator,
			_ interpreter.ArgumentTypesIterator,
			_ interpreter.Value,
			args []interpreter.Value,
		) interpreter.Value {
			*invoked = true
			assert.Equal(t, "\"hello\"", args[0].String())
			return interpreter.Void
		},
	)
}

func newAssertUnexpectedLogFunction(t *testing.T) stdlib.StandardLibraryValue {
	return stdlib.NewInterpreterStandardLibraryStaticFunction(
		"log",
		stdlib.LogFunctionType,
		"",
		func(
			_ interpreter.NativeFunctionContext,
			_ interpreter.TypeArgumentsIterator,
			_ interpreter.ArgumentTypesIterator,
			_ interpreter.Value,
			_ []interpreter.Value,
		) interpreter.Value {
			assert.Fail(t, "unexpected call of log")
			return interpreter.Void
		},
	)
}

func TestInterpretArrayMutation(t *testing.T) {

	t.Parallel()

	t.Run("simple array valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
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

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("baz"),
				interpreter.NewUnmeteredStringValue("bar"),
			),
			array,
		)
	})

	t.Run("simple array invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names[0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var mutationError *interpreter.ContainerMutationError
		require.ErrorAs(t, err, &mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
		assert.Equal(t, sema.IntType, mutationError.ActualType)
	})

	t.Run("nested array invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: [[AnyStruct]] = [["foo", "bar"]] as [[String]]
                names[0][0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var mutationError *interpreter.ContainerMutationError
		require.ErrorAs(t, err, &mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
		assert.Equal(t, sema.IntType, mutationError.ActualType)
	})

	t.Run("array append valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
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

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("foo"),
				interpreter.NewUnmeteredStringValue("bar"),
				interpreter.NewUnmeteredStringValue("baz"),
			),
			array,
		)
	})

	t.Run("array append invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.append(5)
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var argumentTypeError *interpreter.InvalidArgumentTypeError
		require.ErrorAs(t, err, &argumentTypeError)

		assert.Equal(t, sema.StringType, argumentTypeError.ExpectedType)
		assert.Equal(t, sema.IntType, argumentTypeError.ActualType)
	})

	t.Run("array appendAll invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.appendAll(["baz", 5] as [AnyStruct])
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var argumentTypeError *interpreter.InvalidArgumentTypeError
		require.ErrorAs(t, err, &argumentTypeError)

		assert.Equal(
			t,
			&sema.VariableSizedType{
				Type: sema.StringType,
			},
			argumentTypeError.ExpectedType,
		)
		assert.Equal(
			t,
			&sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
			argumentTypeError.ActualType,
		)
	})

	t.Run("array insert valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
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

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("foo"),
				interpreter.NewUnmeteredStringValue("baz"),
				interpreter.NewUnmeteredStringValue("bar"),
			),
			array,
		)
	})

	t.Run("array insert invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.insert(at: 1, 4)
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var argumentTypeError *interpreter.InvalidArgumentTypeError
		require.ErrorAs(t, err, &argumentTypeError)

		assert.Equal(t, sema.StringType, argumentTypeError.ExpectedType)
		assert.Equal(t, sema.IntType, argumentTypeError.ActualType)
	})

	t.Run("array concat mismatching values", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            let names: [AnyStruct] = ["foo", "bar"] as [String]

            fun test(): [AnyStruct] {
                return names.concat(["baz", 5] as [AnyStruct])
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var argumentTypeError *interpreter.InvalidArgumentTypeError
		require.ErrorAs(t, err, &argumentTypeError)

		// Check original array

		namesVal := inter.GetGlobal("names")
		require.IsType(t, &interpreter.ArrayValue{}, namesVal)
		namesValArray := namesVal.(*interpreter.ArrayValue)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("foo"),
				interpreter.NewUnmeteredStringValue("bar"),
			),
			namesValArray,
		)
	})

	t.Run("invalid update through reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                let namesRef = &names as auth(Mutate) &[AnyStruct]
                namesRef[0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var mutationError *interpreter.ContainerMutationError
		require.ErrorAs(t, err, &mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
		assert.Equal(t, sema.IntType, mutationError.ActualType)
	})

	t.Run("host function mutation", func(t *testing.T) {
		t.Parallel()

		invoked := false

		valueDeclaration := newAssertHelloLogFunction(t, &invoked)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun test() {
                let array: [AnyStruct] = [nil] as [(fun(AnyStruct):Void)?]

                array[0] = log

                let logger = array[0] as! (fun(AnyStruct): Void)
                logger("hello")
            }`,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
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
		require.NoError(t, err)

		assert.True(t, invoked)
	})

	t.Run("function mutation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): [String] {
                let array: [AnyStruct] = [nil, nil] as [(fun():String)?]

                array[0] = foo
                array[1] = bar

                let callFoo = array[0] as! fun(): String
                let callBar = array[1] as! fun(): String
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
			array.Get(inter, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, 1),
		)
	})

	t.Run("bound function mutation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
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
                let array: [AnyStruct] = [nil, nil] as [(fun():String)?]

                let a = Foo()
                let b = Bar()

                array[0] = a.foo
                array[1] = b.bar

                let callFoo = array[0] as! fun():String
                let callBar = array[1] as! fun():String

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
			array.Get(inter, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, 1),
		)
	})

	t.Run("invalid function mutation", func(t *testing.T) {

		t.Parallel()

		valueDeclaration := newAssertUnexpectedLogFunction(t)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndPrepareWithOptions(t, `
                fun test() {
                    let array: [AnyStruct] = [nil] as [(fun():Void)?]

                    array[0] = log
                }
            `,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
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

		var mutationError *interpreter.ContainerMutationError
		require.ErrorAs(t, err, &mutationError)

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

func TestInterpretDictionaryMutation(t *testing.T) {

	t.Parallel()

	t.Run("simple dictionary valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
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
			interpreter.NewUnmeteredStringValue("foo"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("baz"), val)
	})

	t.Run("simple dictionary invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                names["foo"] = 5
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var mutationError *interpreter.ContainerMutationError
		require.ErrorAs(t, err, &mutationError)

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

		inter := parseCheckAndPrepare(t, `
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

		inter := parseCheckAndPrepare(t, `
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
			interpreter.NewUnmeteredStringValue("foo"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("baz"), val)
	})

	t.Run("dictionary insert invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                names.insert(key: "foo", 5)
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var argumentTypeError *interpreter.InvalidArgumentTypeError
		require.ErrorAs(t, err, &argumentTypeError)

		assert.Equal(t, sema.StringType, argumentTypeError.ExpectedType)
		assert.Equal(t, sema.IntType, argumentTypeError.ActualType)
	})

	t.Run("dictionary insert invalid key", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: {Path: AnyStruct} = {/public/path: "foo"} as {PublicPath: String}
                names.insert(key: /private/path, "bar")
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var argumentTypeError *interpreter.InvalidArgumentTypeError
		require.ErrorAs(t, err, &argumentTypeError)

		assert.Equal(t, sema.PublicPathType, argumentTypeError.ExpectedType)
		assert.Equal(t, sema.PrivatePathType, argumentTypeError.ActualType)
	})

	t.Run("invalid update through reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                let namesRef = &names as auth(Mutate) &{String: AnyStruct}
                namesRef["foo"] = 5
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var mutationError *interpreter.ContainerMutationError
		require.ErrorAs(t, err, &mutationError)

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

		valueDeclaration := newAssertHelloLogFunction(t, &invoked)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun test() {
                let dict: {String: AnyStruct} = {}

                dict["test"] = log

                let logger = dict["test"]! as! fun(AnyStruct): Void
                logger("hello")
            }`,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
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
		require.NoError(t, err)

		assert.True(t, invoked)
	})

	t.Run("function mutation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
           fun test(): [String] {
               let dict: {String: AnyStruct} = {}

               dict["foo"] = foo
               dict["bar"] = bar

               let callFoo = dict["foo"]! as! fun():String
               let callBar = dict["bar"]! as! fun():String
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
			array.Get(inter, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, 1),
		)
	})

	t.Run("bound function mutation", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
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

               let callFoo = dict["foo"]! as! fun():String
               let callBar = dict["bar"]! as! fun():String

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
			array.Get(inter, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, 1),
		)
	})

	t.Run("invalid function mutation", func(t *testing.T) {

		t.Parallel()

		valueDeclaration := newAssertUnexpectedLogFunction(t)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndPrepareWithOptions(t, `
               fun test() {
                   let dict: {String: AnyStruct} = {} as {String: fun():Void}

                   dict["log"] = log
               }
           `,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
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

		var mutationError *interpreter.ContainerMutationError
		require.ErrorAs(t, err, &mutationError)

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

		inter := parseCheckAndPrepare(t, `
            struct S {}

            fun test(owner: &Account) {
                let funcs: {String: fun(&Account, [UInt64]): [S]} = {}

                funcs["test"] = fun (owner: &Account, ids: [UInt64]): [S] { return [] }

                funcs["test"]!(owner: owner, ids: [1])
            }
        `)

		owner := stdlib.NewAccountReferenceValue(
			inter,
			nil,
			interpreter.AddressValue{1},
			interpreter.UnauthorizedAccess,
		)

		_, err := inter.Invoke("test", owner)
		require.NoError(t, err)
	})
}

func TestInterpretContainerMutationAfterNilCoalescing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      fun test(): String? {
          let xs: {UInt32: String}? = nil
          let ys: {UInt32: String} = xs ?? {}
          ys[0] = "test"
          return ys[0]
      }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("test"),
		),
		result,
	)
}

func TestInterpretContainerMutationWhileIterating(t *testing.T) {

	t.Run("array, append", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): [String] {
                let array: [String] = ["foo", "bar"]

                var i = 0
                for element in array {
                    if i == 0 {
                        array.append("baz")
                    }
                    array[i] = "hello"
                    i = i + 1
                }

                return array
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var containerMutationError *interpreter.ContainerMutatedDuringIterationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("array, remove", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [String] = ["foo", "bar", "baz"]

                var i = 0
                for element in array {
                    if i == 0 {
                        array.remove(at: 1)
                    }
                }
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		var containerMutationError *interpreter.ContainerMutatedDuringIterationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("dictionary, add", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): {String: String} {
                let dictionary: {String: String} = {"a": "foo", "b": "bar"}

                var i = 0
                dictionary.forEachKey(fun (key: String): Bool {
                    if i == 0 {
                        dictionary["c"] = "baz"
                    }

                    dictionary[key] = "hello"
                    return true
                })

                return dictionary
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		var containerMutationError *interpreter.ContainerMutatedDuringIterationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("dictionary, remove", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): {String: String} {
                let dictionary: {String: String} = {"a": "foo", "b": "bar", "c": "baz"}

                var i = 0
                dictionary.forEachKey(fun (key: String): Bool {
                    if i == 0 {
                        dictionary.remove(key: "b")
                    }
                    return true
                })

                return dictionary
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		var containerMutationError *interpreter.ContainerMutatedDuringIterationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("resource dictionary, remove", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            resource Foo {}

            fun test(): @{String: Foo} {
                let dictionary: @{String: Foo} <- {"a": <- create Foo(), "b": <- create Foo(), "c": <- create Foo()}

                var dictionaryRef = &dictionary as auth(Mutate) &{String: Foo}

                var i = 0
                dictionary.forEachKey(fun (key: String): Bool {
                    if i == 0 {
                        destroy dictionaryRef.remove(key: "b")
                    }
                    return true
                })

                return <- dictionary
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		var containerMutationError *interpreter.ContainerMutatedDuringIterationError
		require.ErrorAs(t, err, &containerMutationError)
	})
}

func TestInterpretInnerContainerMutationWhileIteratingOuter(t *testing.T) {

	t.Parallel()

	t.Run("nested array, directly mutating inner", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): [String] {
                let nestedArrays: [[String]] = [["foo", "bar"], ["apple", "orange"]]
                for array in nestedArrays {
                    array[0] = "hello"
                }

                return nestedArrays[0]
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		// array `["foo", "bar"]` should stay unchanged, because what's mutated is a copy.

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("foo"),
				interpreter.NewUnmeteredStringValue("bar"),
			),
			result,
		)
	})

	t.Run("nested array, mutating inner via outer", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): [String] {
                let nestedArrays: [[String]] = [["foo", "bar"], ["apple", "orange"]]
                for array in nestedArrays {
                    nestedArrays[0][0] = "hello"
                }

                return nestedArrays[0]
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("hello"),
				interpreter.NewUnmeteredStringValue("bar"),
			),
			result,
		)
	})

	t.Run("dictionary inside array, mutating inner via outer", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): {String: String} {
                let dictionaryArray: [{String: String}] = [{"name": "foo"}, {"name": "bar"}]
                for dictionary in dictionaryArray {
                    dictionaryArray[0]["name"] = "hello"
                }

                return dictionaryArray[0]
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.DictionaryValue{}, result)
		dictionary := result.(*interpreter.DictionaryValue)

		require.Equal(t, 1, dictionary.Count())

		val, present := dictionary.Get(
			inter,
			interpreter.NewUnmeteredStringValue("name"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("hello"), val)
	})

	t.Run("dictionary", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): {String: String} {
                let nestedDictionary: {String: {String: String}} = {"a": {"name": "foo"}, "b": {"name": "bar"}}
                nestedDictionary.forEachKey(fun (key: String): Bool {
                    var dictionary = nestedDictionary[key]!
                    dictionary["name"] = "hello"
                    return true
                })

                return nestedDictionary["a"]!
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		// dictionary `{"name": "foo"}` should stay unchanged, because what's mutated is a copy.

		require.IsType(t, &interpreter.DictionaryValue{}, result)
		dictionary := result.(*interpreter.DictionaryValue)

		require.Equal(t, 1, dictionary.Count())

		val, present := dictionary.Get(
			inter,
			interpreter.NewUnmeteredStringValue("name"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("foo"), val)
	})
}

// Iterating over a container in a for-in loop defensively checks the container,
// like explicit container indexing does:
// the container's actual (static) type must conform to the type expected by sema.
// A type-confused container is rejected with an IndexedTypeError
// before iteration begins.
func TestInterpretForLoopFunctionElementTypeConfusion(t *testing.T) {

	t.Parallel()

	t.Run("variable-sized array", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [fun(): Void]) {
                for f in arr {
                    f()
                }
            }
        `)

		// Sema sees [fun(): Void], but the array actually holds an IntValue.
		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("constant-sized array", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [fun(): Void; 1]) {
                for f in arr {
                    f()
                }
            }
        `)

		// Sema sees [fun(): Void; 1], but the array actually holds an IntValue.
		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
				Size: 1,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("reference to array", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: &[fun(): Void]) {
                for f in arr {
                    f()
                }
            }
        `)

		// Sema sees &[fun(): Void], but the referenced array actually holds an IntValue.
		// The static type of a reference is derived from the referenced value,
		// so the defensive check must reject the reference.
		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)

		arrayReference := interpreter.NewUnmeteredEphemeralReferenceValue(
			noopReferenceTracker{},
			interpreter.UnauthorizedAccess,
			confusedArray,
			sema.NewVariableSizedType(
				nil,
				sema.NewSimpleFunctionType(
					sema.FunctionPurityImpure,
					nil,
					sema.VoidTypeAnnotation,
				),
			),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", arrayReference) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})
}

func TestInterpretIndexExpressionFunctionElementTypeConfusion(t *testing.T) {

	t.Parallel()

	t.Run("variable-sized array", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [fun(): Void]) {
                let f = arr[0]
                f()
            }
        `)

		// Sema sees [fun(): Void], but the array actually holds an IntValue.
		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("constant-sized array", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [fun(): Void; 1]) {
                let f = arr[0]
                f()
            }
        `)

		// Sema sees [fun(): Void; 1], but the array actually holds an IntValue.
		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
				Size: 1,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("dictionary", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(dict: {String: fun(): Void}) {
                let f = dict["x"]!
                f()
            }
        `)

		// Sema sees {String: fun(): Void}, but the dictionary actually holds an IntValue.
		confusedDictionary := interpreter.NewDictionaryValue(
			inter,
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeString,
				interpreter.PrimitiveStaticTypeInt,
			),
			interpreter.NewUnmeteredStringValue("x"),
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedDictionary) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})
}

// Member access on a container with a type-confused element type is caught
// by the defensive MemberAccessTypeError check before the function ever sees
// the wrong-typed receiver. Covers every container function exercised by
// TestInterpretContainerMethodElementCascading.
//
// For every test case, sema sees a container whose element/value type is
// String, but the runtime value's static type carries Int. The defensive
// check on the receiver's static type fires at member access time, before
// any of the function's logic runs.
func TestInterpretContainerFunctionElementTypeConfusion(t *testing.T) {

	t.Parallel()

	type receiverShape int
	const (
		variableSizedArray receiverShape = iota
		constantSizedArray
		dictionary
	)

	buildConfusedReceiver := func(inter Invokable, shape receiverShape) interpreter.Value {
		switch shape {
		case variableSizedArray:
			return interpreter.NewArrayValue(
				inter,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(42),
			)
		case constantSizedArray:
			return interpreter.NewArrayValue(
				inter,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
					Size: 1,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(42),
			)
		case dictionary:
			return interpreter.NewDictionaryValue(
				inter,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeInt,
				),
				interpreter.NewUnmeteredStringValue("x"),
				interpreter.NewUnmeteredIntValueFromInt64(42),
			)
		}
		panic("unknown shape")
	}

	type testCase struct {
		name  string
		shape receiverShape
		code  string
	}

	cases := []testCase{
		// Array — mutating methods.
		{
			name:  "array append",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]) {
                    arr.append("hello")
                }
            `,
		},
		{
			name:  "array appendAll",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]) {
                    arr.appendAll([])
                }
            `,
		},
		{
			name:  "array insert",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]) {
                    arr.insert(at: 0, "hello")
                }
            `,
		},
		{
			name:  "array remove",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): String {
                    return arr.remove(at: 0)
                }
            `,
		},
		{
			name:  "array removeFirst",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): String {
                    return arr.removeFirst()
                }
            `,
		},
		{
			name:  "array removeLast",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): String {
                    return arr.removeLast()
                }
            `,
		},

		// Array — read methods returning new arrays.
		{
			name:  "array concat",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): [String] {
                    return arr.concat([])
                }
            `,
		},
		{
			name:  "array slice",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): [String] {
                    return arr.slice(from: 0, upTo: 1)
                }
            `,
		},
		{
			name:  "array reverse",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): [String] {
                    return arr.reverse()
                }
            `,
		},
		{
			name:  "array filter",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): [String] {
                    return arr.filter(view fun (_: String): Bool { return true })
                }
            `,
		},
		{
			name:  "array map",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): [Int] {
                    return arr.map(view fun (_: String): Int { return 0 })
                }
            `,
		},
		{
			name:  "array toVariableSized",
			shape: constantSizedArray,
			code: `
                fun test(arr: [String; 1]): [String] {
                    return arr.toVariableSized()
                }
            `,
		},

		// Constant-sized array — read methods.
		{
			name:  "constant-sized array reverse",
			shape: constantSizedArray,
			code: `
                fun test(arr: [String; 1]): [String; 1] {
                    return arr.reverse()
                }
            `,
		},
		{
			name:  "constant-sized array filter",
			shape: constantSizedArray,
			code: `
                fun test(arr: [String; 1]): [String] {
                    return arr.filter(view fun (_: String): Bool { return true })
                }
            `,
		},
		{
			name:  "constant-sized array map",
			shape: constantSizedArray,
			code: `
                fun test(arr: [String; 1]): [Int; 1] {
                    return arr.map(view fun (_: String): Int { return 0 })
                }
            `,
		},
		{
			name:  "constant-sized array firstIndex",
			shape: constantSizedArray,
			code: `
                fun test(arr: [String; 1]): Int? {
                    return arr.firstIndex(of: "hello")
                }
            `,
		},
		{
			name:  "constant-sized array contains",
			shape: constantSizedArray,
			code: `
                fun test(arr: [String; 1]): Bool {
                    return arr.contains("hello")
                }
            `,
		},
		{
			name:  "array toConstantSized",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): [String; 1]? {
                    return arr.toConstantSized<[String; 1]>()
                }
            `,
		},

		// Array — read methods returning scalars.
		{
			name:  "array firstIndex",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): Int? {
                    return arr.firstIndex(of: "hello")
                }
            `,
		},
		{
			name:  "array contains",
			shape: variableSizedArray,
			code: `
                fun test(arr: [String]): Bool {
                    return arr.contains("hello")
                }
            `,
		},

		// Dictionary methods.
		{
			name:  "dictionary remove",
			shape: dictionary,
			code: `
                fun test(dict: {String: String}): String? {
                    return dict.remove(key: "x")
                }
            `,
		},
		{
			name:  "dictionary insert",
			shape: dictionary,
			code: `
                fun test(dict: {String: String}): String? {
                    return dict.insert(key: "x", "hello")
                }
            `,
		},
		{
			name:  "dictionary containsKey",
			shape: dictionary,
			code: `
                fun test(dict: {String: String}): Bool {
                    return dict.containsKey("x")
                }
            `,
		},
		{
			name:  "dictionary forEachKey",
			shape: dictionary,
			code: `
                fun test(dict: {String: String}) {
                    dict.forEachKey(fun (_: String): Bool { return true })
                }
            `,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			inter := parseCheckAndPrepare(t, tc.code)
			confusedReceiver := buildConfusedReceiver(inter, tc.shape)

			_, err := inter.InvokeUncheckedForTestingOnly("test", confusedReceiver) //nolint:staticcheck
			RequireError(t, err)

			var memberAccessTypeError *interpreter.MemberAccessTypeError
			require.ErrorAs(t, err, &memberAccessTypeError)
		})
	}
}

// Element-type confusion is one axis of array shape mismatch. The other axes
// — variable-vs-constant kind mismatch and constant-array size mismatch —
// also must be caught by the defensive subtyping checks, since the runtime
// value's static type would otherwise lie about the array's capacity and
// shape.
func TestInterpretConstantSizedArrayShapeConfusion(t *testing.T) {

	t.Parallel()

	t.Run("for-loop: variable-sized expected, constant-sized actual", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [String]) {
                for s in arr { }
            }
        `)

		// Sema sees [String], but the array is actually a constant-sized [String; 1].
		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
				Size: 1,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredStringValue("hello"),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("for-loop: constant-sized expected, variable-sized actual", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [String; 1]) {
                for s in arr { }
            }
        `)

		// Sema sees [String; 1], but the array is actually a variable-sized [String].
		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredStringValue("hello"),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("for-loop: constant-sized size mismatch", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [String; 2]) {
                for s in arr { }
            }
        `)

		// Sema sees [String; 2], but the array is actually a [String; 1].
		// Even though the element type matches, the size differs, so iteration
		// would loop fewer times than the declared shape implies.
		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
				Size: 1,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredStringValue("hello"),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("indexing: variable-sized expected, constant-sized actual", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [String]): String {
                return arr[0]
            }
        `)

		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
				Size: 1,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredStringValue("hello"),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("indexing: constant-sized expected, variable-sized actual", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [String; 1]): String {
                return arr[0]
            }
        `)

		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredStringValue("hello"),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("indexing: constant-sized size mismatch", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [String; 2]): String {
                return arr[1]
            }
        `)

		// Out-of-bounds at runtime if the smaller array escaped the check.
		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
				Size: 1,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredStringValue("hello"),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("member: constant-sized size mismatch", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(arr: [String; 2]): [String; 2] {
                return arr.reverse()
            }
        `)

		confusedArray := interpreter.NewArrayValue(
			inter,
			&interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
				Size: 1,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredStringValue("hello"),
		)

		_, err := inter.InvokeUncheckedForTestingOnly("test", confusedArray) //nolint:staticcheck
		RequireError(t, err)

		var memberAccessTypeError *interpreter.MemberAccessTypeError
		require.ErrorAs(t, err, &memberAccessTypeError)
	})
}
