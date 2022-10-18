/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretResourceReferenceInstanceOf(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        resource R {}

        fun test(): Bool {
            let r <- create R()
            let ref = &r as &R
            let isInstance = ref.isInstance(Type<@R>())
            destroy r
            return isInstance
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		value,
	)
}

func TestInterpretResourceReferenceFieldComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        resource R {
            let n: Int
            init() {
                self.n = 1
            }
        }

        fun test(): Bool {
            let r <- create R()
            let ref = &r as &R
            let isOne = ref.n == 1
            destroy r
            return isOne
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		value,
	)
}

func TestInterpretContainerVariance(t *testing.T) {

	t.Parallel()

	t.Run("invocation of struct function, reference", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S1 {
              pub fun getSecret(): Int {
                  return 0
              }
          }

          struct S2 {
              priv fun getSecret(): Int {
                  return 42
              }
          }

          fun test(): Int {
              let dict: {Int: &S1} = {}
              let dictRef = &dict as &{Int: &AnyStruct}

              let s2 = S2()
              dictRef[0] = &s2 as &AnyStruct

              return dict.values[0].getSecret()
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationErr interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationErr)
	})

	t.Run("invocation of struct function, value", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S1 {
              pub fun getSecret(): Int {
                  return 0
              }
          }

          struct S2 {
              priv fun getSecret(): Int {
                  return 42
              }
          }

          fun test(): Int {
              let dict: {Int: S1} = {}
              let dictRef = &dict as &{Int: AnyStruct}

              dictRef[0] = S2()

              return dict.values[0].getSecret()
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationErr interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationErr)
	})

	t.Run("field read, reference", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
         struct S1 {
             var value: Int

             init() {
                 self.value = 0
             }
         }

         struct S2 {
             priv var value: Int

             init() {
                 self.value = 1
             }
         }

         fun test(): Int {
             let dict: {Int: &S1} = {}
             let dictRef = &dict as &{Int: &AnyStruct}

             let s2 = S2()
             dictRef[0] = &s2 as &AnyStruct

             return dict.values[0].value
         }
       `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("field read, value", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
         struct S1 {
             var value: Int

             init() {
                 self.value = 0
             }
         }

         struct S2 {
             priv var value: Int

             init() {
                 self.value = 1
             }
         }

         fun test(): Int {
             let dict: {Int: S1} = {}
             let dictRef = &dict as &{Int: AnyStruct}

             dictRef[0] = S2()

             return dict.values[0].value
         }
       `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("field write, reference", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S1 {
              var value: Int

              init() {
                  self.value = 0
              }
          }

          struct S2 {
              // field is only publicly readable, not writeable
              pub var value: Int

              init() {
                  self.value = 0
              }
          }

          fun test() {
              let dict: {Int: &S1} = {}

              let s2 = S2()

              let dictRef = &dict as &{Int: &AnyStruct}
              dictRef[0] = &s2 as &AnyStruct

              dict.values[0].value = 1

             // NOTE: intentionally not reading,
             // the test checks writes
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("field write, value", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S1 {
              var value: Int

              init() {
                  self.value = 0
              }
          }

          struct S2 {
              // field is only publicly readable, not writeable
              pub var value: Int

              init() {
                  self.value = 0
              }
          }

          fun test() {
              let dict: {Int: S1} = {}
              let dictRef = &dict as &{Int: AnyStruct}

              dictRef[0] = S2()

              dict.values[0].value = 1

             // NOTE: intentionally not reading,
             // the test checks writes
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("value transfer", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S1 {}

          struct S2 {}

          fun test() {
              let dict: {Int: S1} = {}

              let s2 = S2()

              let dictRef = &dict as &{Int: AnyStruct}
              dictRef[0] = s2

              let x = dict.values[0]
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("invocation of function, value", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun f1(): Int {
              return 0
          }

          fun f2(): String {
              return "0"
          }

          fun test(): Int {
              let dict: {Int: ((): Int)} = {}
              let dictRef = &dict as &{Int: AnyStruct}

              dictRef[0] = f2

              return dict.values[0]()
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("interpreted function argument", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun f(_ value: [UInt8]) {}

          fun test() {
              let dict: {Int: [UInt8]} = {}
              let dictRef = &dict as &{Int: AnyStruct}

              dictRef[0] = "not an [UInt8] array, but a String"

              f(dict.values[0])
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("native function argument", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun f(_ value: [UInt8]) {}

          fun test() {
              let dict: {Int: [UInt8]} = {}
              let dictRef = &dict as &{Int: AnyStruct}

              dictRef[0] = "not an [UInt8] array, but a String"

              String.encodeHex(dict.values[0])
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})
}

func TestInterpretReferenceExpressionOfOptional(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {}

          let r: @R? <- create R()
          let ref = &r as &R?
        `)

		value := inter.Globals["ref"].GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.ReturnEmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {}

          let s: S? = S()
          let ref = &s as &S?
        `)

		value := inter.Globals["ref"].GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.ReturnEmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let i: Int? = 1
          let ref = &i as &Int?
        `)

		value := inter.Globals["ref"].GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.ReturnEmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
	})

	t.Run("as optional, some", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let i: Int? = 1
          let ref = &i as &Int?
        `)

		value := inter.Globals["ref"].GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.ReturnEmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
	})

	t.Run("as optional, nil", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let i: Int? = nil
          let ref = &i as &Int?
        `)

		value := inter.Globals["ref"].GetValue()
		require.IsType(t, interpreter.NilValue{}, value)
	})

	t.Run("upcast to optional", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let i: Int = 1
          let ref = &i as &Int?
        `)

		value := inter.Globals["ref"].GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.ReturnEmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
	})
}
