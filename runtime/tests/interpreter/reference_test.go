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

	"github.com/onflow/atree"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"

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
		CheckErrorMessage(err)

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
		CheckErrorMessage(err)

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
		CheckErrorMessage(err)

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
		CheckErrorMessage(err)

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
		CheckErrorMessage(err)

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
		CheckErrorMessage(err)

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
		CheckErrorMessage(err)

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
		CheckErrorMessage(err)

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
		CheckErrorMessage(err)

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
		CheckErrorMessage(err)

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})
}

func TestInterpretResourceReferenceAfterMove(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                let value: String

                init(value: String) {
                    self.value = value
                }
            }

            fun test(target: &[R]): String {
                let r <- create R(value: "testValue")
                let ref = &r as &R
                target.append(<-r)
                return ref.value
            }
        `)

		address := common.Address{0x1}

		rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)

		array := interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			address,
		)

		arrayRef := &interpreter.EphemeralReferenceValue{
			Authorized: false,
			Value:      array,
			BorrowedType: &sema.VariableSizedType{
				Type: rType,
			},
		}

		value, err := inter.Invoke("test", arrayRef)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("testValue"),
			value,
		)
	})

	t.Run("array", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                let value: String

                init(value: String) {
                    self.value = value
                }
            }

            fun test(target: &[[R]]): String {
                let rs <- [<-create R(value: "testValue")]
                let ref = &rs as &[R]
                target.append(<-rs)
                return ref[0].value
            }
        `)

		address := common.Address{0x1}

		rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)

		array := interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.VariableSizedStaticType{
					Type: interpreter.ConvertSemaToStaticType(nil, rType),
				},
			},
			address,
		)

		arrayRef := &interpreter.EphemeralReferenceValue{
			Authorized: false,
			Value:      array,
			BorrowedType: &sema.VariableSizedType{
				Type: &sema.VariableSizedType{
					Type: rType,
				},
			},
		}

		value, err := inter.Invoke("test", arrayRef)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("testValue"),
			value,
		)
	})

	t.Run("dictionary", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                let value: String

                init(value: String) {
                    self.value = value
                }
            }

            fun test(target: &[{Int: R}]): String? {
                let rs <- {1: <-create R(value: "testValue")}
                let ref = &rs as &{Int: R}
                target.append(<-rs)
                return ref[1]?.value
            }
        `)

		address := common.Address{0x1}

		rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)

		array := interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeInt,
					ValueType: interpreter.ConvertSemaToStaticType(nil, rType),
				},
			},
			address,
		)

		arrayRef := &interpreter.EphemeralReferenceValue{
			Authorized: false,
			Value:      array,
			BorrowedType: &sema.VariableSizedType{
				Type: &sema.DictionaryType{
					KeyType:   sema.IntType,
					ValueType: rType,
				},
			},
		}

		value, err := inter.Invoke("test", arrayRef)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("testValue"),
			),
			value,
		)
	})
}

func TestInterpretReferenceUseAfterShiftStatementMove(t *testing.T) {

	t.Parallel()

	t.Run("container on stack", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R2 {
              let value: String

              init() {
                  self.value = "test"
              }
          }

          resource R1 {
              var r2: @R2?

              init() {
                  self.r2 <- nil
              }

              destroy() {
                  destroy self.r2
              }

              fun borrowR2(): &R2? {
                  let optR2 <- self.r2 <- nil
                  let r2 <- optR2!
                  let ref = &r2 as &R2
                  self.r2 <-! r2
                  return ref
              }
          }

          fun test(): String {
              let r2 <- create R2()
              let r1 <- create R1()
              r1.r2 <-! r2
              let optRef = r1.borrowR2()
              let value = optRef!.value
              destroy r1
              return value
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("test"),
			value,
		)

	})

	t.Run("container in account", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              resource R2 {
                  let value: String

                  init() {
                      self.value = "test"
                  }
              }

              resource R1 {
                  var r2: @R2?

                  init() {
                      self.r2 <- nil
                  }

                  destroy() {
                      destroy self.r2
                  }

                  fun borrowR2(): &R2? {
                      let optR2 <- self.r2 <- nil
                      let r2 <- optR2!
                      let ref = &r2 as &R2
                      self.r2 <-! r2
                      return ref
                  }
              }

              fun createR1(): @R1 {
                  return <- create R1()
              }

              fun getOwnerR1(r1: &R1): Address? {
                  return r1.owner?.address
              }

              fun getOwnerR2(r1: &R1): Address? {
                  return r1.r2?.owner?.address
              }

              fun test(r1: &R1): String {
                  let r2 <- create R2()
                  r1.r2 <-! r2
                  let optRef = r1.borrowR2()
                  let value = optRef!.value
                  return value
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					PublicAccountHandler: func(address interpreter.AddressValue) interpreter.Value {
						return newTestPublicAccountValue(nil, address)
					},
				},
			},
		)
		require.NoError(t, err)

		r1, err := inter.Invoke("createR1")
		require.NoError(t, err)

		r1 = r1.Transfer(inter, interpreter.ReturnEmptyLocationRange, atree.Address{1}, false, nil)

		r1Type := checker.RequireGlobalType(t, inter.Program.Elaboration, "R1")

		ref := &interpreter.EphemeralReferenceValue{
			Value:        r1,
			BorrowedType: r1Type,
		}

		// Test

		value, err := inter.Invoke("test", ref)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("test"),
			value,
		)

		// Check R1 owner

		r1Address, err := inter.Invoke("getOwnerR1", ref)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.AddressValue{1},
			),
			r1Address,
		)

		// Check R2 owner

		r2Address, err := inter.Invoke("getOwnerR2", ref)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.AddressValue{1},
			),
			r2Address,
		)
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
		require.IsType(t, interpreter.Nil, value)
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
