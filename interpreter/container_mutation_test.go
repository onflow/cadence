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
)

func TestInterpretArrayMutation(t *testing.T) {

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

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
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

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names[0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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
		RequireError(t, err)

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

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
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

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.append(5)
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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
		RequireError(t, err)

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

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
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

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.insert(at: 1, 4)
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ContainerMutationError{})

		// Check original array

		namesVal := inter.Globals.Get("names").GetValue(inter)
		require.IsType(t, &interpreter.ArrayValue{}, namesVal)
		namesValArray := namesVal.(*interpreter.ArrayValue)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
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

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                let namesRef = &names as auth(Mutate) &[AnyStruct]
                namesRef[0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
		assert.Equal(t, sema.IntType, mutationError.ActualType)
	})

	t.Run("host function mutation", func(t *testing.T) {
		t.Parallel()

		invoked := false

		valueDeclaration := stdlib.NewStandardLibraryStaticFunction(
			"log",
			stdlib.LogFunctionType,
			"",
			func(invocation interpreter.Invocation) interpreter.Value {
				invoked = true
				assert.Equal(t, "\"hello\"", invocation.Arguments[0].String())
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun test() {
                let array: [AnyStruct] = [nil] as [(fun(AnyStruct):Void)?]

                array[0] = log

                let logger = array[0] as! (fun(AnyStruct): Void)
                logger("hello")
            }`,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				Config: &interpreter.Config{
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

		inter := parseCheckAndInterpret(t, `
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
			array.Get(inter, interpreter.EmptyLocationRange, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, interpreter.EmptyLocationRange, 1),
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
			array.Get(inter, interpreter.EmptyLocationRange, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, interpreter.EmptyLocationRange, 1),
		)
	})

	t.Run("invalid function mutation", func(t *testing.T) {

		t.Parallel()

		valueDeclaration := stdlib.NewStandardLibraryStaticFunction(
			"log",
			stdlib.LogFunctionType,
			"",
			func(invocation interpreter.Invocation) interpreter.Value {
				assert.Fail(t, "unexpected call of log")
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndInterpretWithOptions(t, `
                fun test() {
                    let array: [AnyStruct] = [nil] as [(fun():Void)?]

                    array[0] = log
                }
            `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				Config: &interpreter.Config{
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)

		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

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

func TestInterpretDictionaryMutation(t *testing.T) {

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
			interpreter.EmptyLocationRange,
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
		RequireError(t, err)

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
			interpreter.EmptyLocationRange,
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
		RequireError(t, err)

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
		RequireError(t, err)

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
                let namesRef = &names as auth(Mutate) &{String: AnyStruct}
                namesRef["foo"] = 5
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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

		valueDeclaration := stdlib.NewStandardLibraryStaticFunction(
			"log",
			stdlib.LogFunctionType,
			"",
			func(invocation interpreter.Invocation) interpreter.Value {
				invoked = true
				assert.Equal(t, "\"hello\"", invocation.Arguments[0].String())
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun test() {
                let dict: {String: AnyStruct} = {}

                dict["test"] = log

                let logger = dict["test"]! as! fun(AnyStruct): Void
                logger("hello")
            }`,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				Config: &interpreter.Config{
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

		inter := parseCheckAndInterpret(t, `
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
			array.Get(inter, interpreter.EmptyLocationRange, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, interpreter.EmptyLocationRange, 1),
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
			array.Get(inter, interpreter.EmptyLocationRange, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("hello from bar"),
			array.Get(inter, interpreter.EmptyLocationRange, 1),
		)
	})

	t.Run("invalid function mutation", func(t *testing.T) {

		t.Parallel()

		valueDeclaration := stdlib.NewStandardLibraryStaticFunction(
			"log",
			stdlib.LogFunctionType,
			"",
			func(invocation interpreter.Invocation) interpreter.Value {
				assert.Fail(t, "unexpected call of log")
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndInterpretWithOptions(t, `
               fun test() {
                   let dict: {String: AnyStruct} = {} as {String: fun():Void}

                   dict["log"] = log
               }
           `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				Config: &interpreter.Config{
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)

		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

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

            fun test(owner: &Account) {
                let funcs: {String: fun(&Account, [UInt64]): [S]} = {}

                funcs["test"] = fun (owner: &Account, ids: [UInt64]): [S] { return [] }

                funcs["test"]!(owner: owner, ids: [1])
            }
        `)

		owner := stdlib.NewAccountReferenceValue(
			nil,
			nil,
			interpreter.AddressValue{1},
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
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

		inter := parseCheckAndInterpret(t, `
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
		assert.ErrorAs(t, err, &interpreter.ContainerMutatedDuringIterationError{})
	})

	t.Run("array, remove", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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
		assert.ErrorAs(t, err, &interpreter.ContainerMutatedDuringIterationError{})
	})

	t.Run("dictionary, add", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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
		assert.ErrorAs(t, err, &interpreter.ContainerMutatedDuringIterationError{})
	})

	t.Run("dictionary, remove", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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
		assert.ErrorAs(t, err, &interpreter.ContainerMutatedDuringIterationError{})
	})

	t.Run("resource dictionary, remove", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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
		assert.ErrorAs(t, err, &interpreter.ContainerMutatedDuringIterationError{})
	})
}

func TestInterpretInnerContainerMutationWhileIteratingOuter(t *testing.T) {

	t.Parallel()

	t.Run("nested array, directly mutating inner", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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
				interpreter.EmptyLocationRange,
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

		inter := parseCheckAndInterpret(t, `
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
				interpreter.EmptyLocationRange,
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

		inter := parseCheckAndInterpret(t, `
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
			interpreter.EmptyLocationRange,
			interpreter.NewUnmeteredStringValue("name"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("hello"), val)
	})

	t.Run("dictionary", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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
			interpreter.EmptyLocationRange,
			interpreter.NewUnmeteredStringValue("name"),
		)
		assert.True(t, present)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("foo"), val)
	})
}
