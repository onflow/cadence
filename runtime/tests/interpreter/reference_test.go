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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
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
		interpreter.TrueValue,
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
		interpreter.TrueValue,
		value,
	)
}

func TestInterpretContainerVariance(t *testing.T) {

	t.Parallel()

	t.Run("invocation of struct function, reference", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S1 {
              access(all) fun getSecret(): Int {
                  return 0
              }
          }

          struct S2 {
              access(self) fun getSecret(): Int {
                  return 42
              }
          }

          fun test(): Int {
              let dict: {Int: &S1} = {}
              let dictRef = &dict as auth(Mutate) &{Int: &AnyStruct}

              let s2 = S2()
              dictRef[0] = &s2 as &AnyStruct

              return dict.values[0].getSecret()
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var containerMutationErr interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationErr)
	})

	t.Run("invocation of struct function, value", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S1 {
              access(all) fun getSecret(): Int {
                  return 0
              }
          }

          struct S2 {
              access(self) fun getSecret(): Int {
                  return 42
              }
          }

          fun test(): Int {
              let dict: {Int: S1} = {}
              let dictRef = &dict as auth(Mutate) &{Int: AnyStruct}

              dictRef[0] = S2()

              return dict.values[0].getSecret()
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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
             access(self) var value: Int

             init() {
                 self.value = 1
             }
         }

         fun test(): Int {
             let dict: {Int: &S1} = {}
             let dictRef = &dict as auth(Mutate) &{Int: &AnyStruct}

             let s2 = S2()
             dictRef[0] = &s2 as &AnyStruct

             return dict.values[0].value
         }
       `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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
             access(self) var value: Int

             init() {
                 self.value = 1
             }
         }

         fun test(): Int {
             let dict: {Int: S1} = {}
             let dictRef = &dict as auth(Mutate) &{Int: AnyStruct}

             dictRef[0] = S2()

             return dict.values[0].value
         }
       `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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
              access(all) var value: Int

              init() {
                  self.value = 0
              }
          }

          fun test() {
              let dict: {Int: &S1} = {}

              let s2 = S2()

              let dictRef = &dict as auth(Mutate) &{Int: &AnyStruct}
              dictRef[0] = &s2 as &AnyStruct

              dict.values[0].value = 1

             // NOTE: intentionally not reading,
             // the test checks writes
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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
              access(all) var value: Int

              init() {
                  self.value = 0
              }
          }

          fun test() {
              let dict: {Int: S1} = {}
              let dictRef = &dict as auth(Mutate) &{Int: AnyStruct}

              dictRef[0] = S2()

              dict.values[0].value = 1

             // NOTE: intentionally not reading,
             // the test checks writes
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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

              let dictRef = &dict as auth(Mutate) &{Int: AnyStruct}
              dictRef[0] = s2

              let x = dict.values[0]
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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
              let dict: {Int: fun(): Int} = {}
              let dictRef = &dict as auth(Mutate) &{Int: AnyStruct}

              dictRef[0] = f2

              return dict.values[0]()
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("interpreted function argument", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun f(_ value: [UInt8]) {}

          fun test() {
              let dict: {Int: [UInt8]} = {}
              let dictRef = &dict as auth(Mutate) &{Int: AnyStruct}

              dictRef[0] = "not an [UInt8] array, but a String"

              f(dict.values[0])
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var containerMutationError interpreter.ContainerMutationError
		require.ErrorAs(t, err, &containerMutationError)
	})

	t.Run("native function argument", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun f(_ value: [UInt8]) {}

          fun test() {
              let dict: {Int: [UInt8]} = {}
              let dictRef = &dict as auth(Mutate) &{Int: AnyStruct}

              dictRef[0] = "not an [UInt8] array, but a String"

              String.encodeHex(dict.values[0])
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

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

		value := inter.Globals.Get("ref").GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.EmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {}

          let s: S? = S()
          let ref = &s as &S?
        `)

		value := inter.Globals.Get("ref").GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.EmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let i: Int? = 1
          let ref = &i as &Int?
        `)

		value := inter.Globals.Get("ref").GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.EmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
	})

	t.Run("as optional, some", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let i: Int? = 1
          let ref = &i as &Int?
        `)

		value := inter.Globals.Get("ref").GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.EmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
	})

	t.Run("as optional, nil", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let i: Int? = nil
          let ref = &i as &Int?
        `)

		value := inter.Globals.Get("ref").GetValue()
		require.IsType(t, interpreter.Nil, value)
	})
}

func TestInterpretResourceReferenceInvalidationOnMove(t *testing.T) {

	t.Parallel()

	errorHandler := func(tt *testing.T) func(err error) {
		return func(err error) {
			errors := checker.RequireCheckerErrors(tt, err, 1)
			invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
			assert.ErrorAs(tt, errors[0], &invalidatedRefError)
		}
	}

	t.Run("stack to account", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccountWithErrorHandler(t, address, true, nil, `
            resource R {
                access(all) var id: Int

                access(all) fun setID(_ id: Int) {
                    self.id = id
                }

                init() {
                    self.id = 1
                }
            }

            fun test() {
                let r <-create R()
                let ref = &r as &R

                // Move the resource into the account
                account.storage.save(<-r, to: /storage/r)

                // Update the reference
                ref.setID(2)
            }`, sema.Config{}, errorHandler(t))

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("stack to account readonly", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccountWithErrorHandler(t, address, true, nil, `
            resource R {
                access(all) var id: Int

                init() {
                    self.id = 1
                }
            }

            fun test() {
                let r <-create R()
                let ref = &r as &R

                // Move the resource into the account
                account.storage.save(<-r, to: /storage/r)

                // 'Read' a field from the reference
                let id = ref.id
            }`, sema.Config{}, errorHandler(t))

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("account to stack", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                access(all) var id: Int

                access(all) fun setID(_ id: Int) {
                    self.id = id
                }

                init() {
                    self.id = 1
                }
            }

            fun test(target: auth(Mutate) &[R]) {
                target.append(<- create R())

                // Take reference while in the account
                let ref = target[0]

                // Move the resource out of the account onto the stack
                let movedR <- target.remove(at: 0)

                // Update the reference
                ref.setID(2)

                destroy movedR
            }
        `)

		address := common.Address{0x1}

		rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)

		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			address,
		)

		arrayRef := interpreter.NewUnmeteredEphemeralReferenceValue(
			inter,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"Mutate"} },
				1,
				sema.Conjunction,
			),
			array,
			&sema.VariableSizedType{
				Type: rType,
			},
			interpreter.EmptyLocationRange,
		)

		_, err := inter.Invoke("test", arrayRef)
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("stack to stack", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(
			t,
			`
            resource R {
                access(all) var id: Int

                access(all) fun setID(_ id: Int) {
                    self.id = id
                }

                init() {
                    self.id = 1
                }
            }

            fun test() {
                let r1 <-create R()
                let ref = &r1 as &R

                // Move the resource onto the same stack
                let r2 <- r1

                // Update the reference
                ref.setID(2)

                destroy r2
            }`,

			ParseCheckAndInterpretOptions{
				HandleCheckerError: errorHandler(t),
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("one account to another account", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                access(all) var id: Int

                access(all) fun setID(_ id: Int) {
                    self.id = id
                }

                init() {
                    self.id = 1
                }
            }

            fun test(target1: auth(Mutate) &[R], target2: auth(Mutate) &[R]) {
                target1.append(<- create R())

                // Take reference while in the account_1
                let ref = target1[0]

                // Move the resource out of the account_1 into the account_2
                target2.append(<- target1.remove(at: 0))

                // Update the reference
                ref.setID(2)
            }
        `)

		rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)

		// Resource array in account 0x01

		array1 := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			common.Address{0x1},
		)

		arrayRef1 := interpreter.NewUnmeteredEphemeralReferenceValue(
			inter,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"Mutate"} },
				1,
				sema.Conjunction,
			),
			array1,
			&sema.VariableSizedType{
				Type: rType,
			},
			interpreter.EmptyLocationRange,
		)

		// Resource array in account 0x02

		array2 := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			common.Address{0x2},
		)

		arrayRef2 := interpreter.NewUnmeteredEphemeralReferenceValue(
			inter,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"Mutate"} },
				1,
				sema.Conjunction,
			),
			array2,
			&sema.VariableSizedType{
				Type: rType,
			},
			interpreter.EmptyLocationRange,
		)

		_, err := inter.Invoke("test", arrayRef1, arrayRef2)
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("account to stack to same account", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                access(all) var id: Int

                access(all) fun setID(_ id: Int) {
                    self.id = id
                }

                init() {
                    self.id = 1
                }
            }

            fun test(target: auth(Mutate) &[R]): Int {
                target.append(<- create R())

                // Take reference while in the account
                let ref = target[0]

                // Move the resource out of the account onto the stack. This should invalidate the reference.
                let movedR <- target.remove(at: 0)

                // Append an extra resource just to force an index change
                target.append(<- create R())

                // Move the resource back into the account (now at a different index)
                // Despite the resource being back in its original account, reference is still invalid.
                target.append(<- movedR)

                // Update the reference
                ref.setID(2)

                return target[1].id
            }
        `)

		address := common.Address{0x1}

		rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)

		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			address,
		)

		arrayRef := interpreter.NewUnmeteredEphemeralReferenceValue(
			inter,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"Mutate"} },
				1,
				sema.Conjunction,
			),
			array,
			&sema.VariableSizedType{
				Type: rType,
			},
			interpreter.EmptyLocationRange,
		)

		_, err := inter.Invoke("test", arrayRef)
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("account to stack storage reference", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            resource R {
                access(all) var id: Int

                access(all) fun setID(_ id: Int) {
                    self.id = id
                }

                init() {
                    self.id = 1
                }
            }

             fun test() {
                let r1 <-create R()
                account.storage.save(<-r1, to: /storage/r)

                let r1Ref = account.storage.borrow<&R>(from: /storage/r)!

                let r2 <- account.storage.load<@R>(from: /storage/r)!

                r1Ref.setID(2)
                destroy r2
            }`, sema.Config{})

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.DereferenceError{})
	})

	t.Run("multiple references with moves", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            resource R {
                access(all) var id: Int

                init() {
                    self.id = 5
                }
            }

            var ref1: &R? = nil
            var ref2: &R? = nil
            var ref3: &R? = nil

            fun setup(collection: auth(Mutate) &[R]) {
                collection.append(<- create R())

                // Take reference while in the account
                ref1 = collection[0]

                // Move the resource out of the account onto the stack. This should invalidate ref1.
                let movedR <- collection.remove(at: 0)

                // Take a reference while on stack
                ref2 = &movedR as &R

                // Append an extra resource just to force an index change
                collection.append(<- create R())

                // Move the resource again into the account (now at a different index)
                collection.append(<- movedR)

                // Take another reference
                ref3 = collection[1]
            }

            fun getRef1Id(): Int {
                return ref1!.id
            }

            fun getRef2Id(): Int {
                return ref2!.id
            }

            fun getRef3Id(): Int {
                return ref3!.id
            }
        `,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: errorHandler(t),
			},
		)
		require.NoError(t, err)

		address := common.Address{0x1}

		rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)

		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			address,
		)

		arrayRef := interpreter.NewUnmeteredEphemeralReferenceValue(
			inter,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"Mutate"} },
				1,
				sema.Conjunction,
			),
			array,
			&sema.VariableSizedType{
				Type: rType,
			},
			interpreter.EmptyLocationRange,
		)

		_, err = inter.Invoke("setup", arrayRef)
		require.NoError(t, err)

		// First reference must be invalid
		_, err = inter.Invoke("getRef1Id")
		RequireError(t, err)
		assert.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})

		// Second reference must be invalid
		_, err = inter.Invoke("getRef2Id")
		RequireError(t, err)
		assert.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})

		// Third reference must be valid
		result, err := inter.Invoke("getRef3Id")
		assert.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			result,
		)
	})

	t.Run("ref source is field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(
			t,
			`
            access(all) fun test() {
                let r <- create R()
                let s = S()
                s.setB(&r as &R)

                let x = s.b!     // get reference from a struct field
                let movedR <- r  // move the resource
                x.a

                destroy movedR
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }

            access(all) struct S {
                access(all) var b: &R?

                access(all) fun setB(_ b: &R) {
                    self.b = b
                }

                init() {
                    self.b = nil
                }
            }`,
		)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("ref target is field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(
			t,
			`
            access(all) fun test() {
                let r <- create R()
                let s = S()

                s.setB(&r as &R)  // assign reference to a struct field
                let movedR <- r  // move the resource
                s.b!.a

                destroy movedR
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }

            access(all) struct S {
                access(all) var b: &R?

                access(all) fun setB(_ b: &R) {
                    self.b = b
                }

                init() {
                    self.b = nil
                }
            }`,
		)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("resource is array element", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(
			t,
			`
            resource R {
                access(all) var id: Int

                access(all) fun setID(_ id: Int) {
                    self.id = id
                }

                init() {
                    self.id = 1
                }
            }

            fun test() {
                let array <- [<- create R()]
                let ref = &array[0] as &R

                // remove the resource from array
                let r <- array.remove(at: 0)

                // Update the reference
                ref.setID(2)

                destroy r
                destroy array
            }`,
		)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("resource is dictionary entry", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(
			t,
			`
            resource R {
                access(all) var id: Int

                access(all) fun setID(_ id: Int) {
                    self.id = id
                }

                init() {
                    self.id = 1
                }
            }

            fun test() {
                let dictionary <- {0: <- create R()}
                let ref = (&dictionary[0] as &R?)!

                // remove the resource from array
                let r <- dictionary.remove(key: 0)

                // Update the reference
                ref.setID(2)

                destroy r
                destroy dictionary
            }`,
		)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("nested resource in composite", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource Foo {
                let id: UInt8  // non resource typed field
                let bar: @Bar   // resource typed field
                init() {
                    self.id = 1
                    self.bar <-create Bar()
                }
            }

            resource Bar {
                let baz: @Baz
                init() {
                    self.baz <-create Baz()
                }
            }

            resource Baz {
                let id: UInt8
                init() {
                    self.id = 1
                }
            }

            fun main() {
                var foo <- create Foo()

                // Get a reference to the inner resource.
                // Function call is just to trick the checker.
                var bazRef = getRef(&foo.bar.baz as &Baz)

                // Move the outer resource
                var foo2 <- foo

                // Access the moved resource
                bazRef.id

                destroy foo2
            }

            fun getRef(_ ref: &Baz): &Baz {
                return ref
            }
        `,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("nested resource in dictionary", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource Foo {}

            fun main() {
                var dict <- {"levelOne": <- {"levelTwo": <- create Foo()}}

                // Get a reference to the inner resource.
                // Function call is just to trick the checker.
                var dictRef = getRef(&dict["levelOne"] as &{String: Foo}?)!

                // Move the outer resource
                var dict2 <- dict

                // Access the inner moved resource
                var fooRef = dictRef["levelTwo"]

                destroy dict2
            }

            fun getRef(_ ref: &{String: Foo}?): &{String: Foo}? {
                return ref
            }
        `,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("nested resource in array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource Foo {}

            fun main() {
                var array <- [<-[<- create Foo()]]

                // Get a reference to the inner resource.
                // Function call is just to trick the checker.
                var arrayRef = getRef(&array[0] as &[Foo])

                // Move the outer resource
                var array2 <- array

                // Access the inner moved resource
                var fooRef = arrayRef[0]

                destroy array2
            }

            fun getRef(_ ref: &[Foo]): &[Foo] {
                return ref
            }
        `,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("nested optional resource", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource Foo {
                let optionalBar: @Bar?
                init() {
                    self.optionalBar <-create Bar()
                }
            }

            resource Bar {
                let id: UInt8
                init() {
                    self.id = 1
                }
            }

            fun main() {
                var foo <- create Foo()

                // Get a reference to the inner resource.
                // Function call is just to trick the checker.
                var barRef = getRef(&foo.optionalBar as &Bar?)

                // Move the outer resource
                var foo2 <- foo

                // Access the moved resource
                barRef!.id

                destroy foo2
            }

            fun getRef(_ ref: &Bar?): &Bar? {
                return ref
            }
        `,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("reference created by field access", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource Foo {
                let bar: @Bar
                init() {
                    self.bar <-create Bar()
                }
            }

            resource Bar {
                let id: UInt8
                init() {
                    self.id = 1
                }
            }

            fun main() {
                var foo <- create Foo()
                var fooRef = &foo as &Foo

                // Get a reference to the inner resource.
                // Function call is just to trick the checker.
                var barRef = getRef(fooRef.bar)

                // Move the outer resource
                var foo2 <- foo

                // Access the moved resource
                barRef.id

                destroy foo2
            }

            fun getRef(_ ref: &Bar): &Bar {
                return ref
            }
        `,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("reference created by index access", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource Foo {
                let id: UInt8
                init() {
                    self.id = 1
                }
            }

            fun main() {
                let array <- [<- create Foo()]
                var arrayRef = &array as &[Foo]

                // Get a reference to the inner resource.
                // Function call is just to trick the checker.
                var fooRef = getRef(arrayRef[0])

                // Move the outer resource
                var array2 <- array

                // Access the moved resource
                fooRef.id

                destroy array2
            }

            fun getRef(_ ref: &Foo): &Foo {
                return ref
            }
        `,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("reference created by field and index access", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
             resource Foo {
                let bar: @Bar
                init() {
                    self.bar <-create Bar()
                }
            }

            resource Bar {
                let id: UInt8
                init() {
                    self.id = 1
                }
            }

            fun main() {
                let array <- [<- create Foo()]
                var arrayRef = &array as &[Foo]

                // Get a reference to the inner resource.
                // Function call is just to trick the checker.
                var barRef = getRef(arrayRef[0].bar)

                // Move the outer resource
                var array2 <- array

                // Access the moved resource
                barRef.id

                destroy array2
            }

            fun getRef(_ ref: &Bar): &Bar {
                return ref
            }
        `,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("downcasted reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
             resource Foo {
                let id: UInt8
                init() {
                    self.id = 1
                }
            }

            fun main() {
                var foo <- create Foo()
                var fooRef = &foo as &Foo

                var anyStruct: AnyStruct = fooRef

                var downCastedRef = anyStruct as! &Foo

                // Move the outer resource
                var foo2 <- foo

                // Access the moved resource
                downCastedRef.id

                destroy foo2
            }
        `,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})
}

func TestInterpretResourceReferenceInvalidationOnDestroy(t *testing.T) {

	t.Parallel()

	errorHandler := func(tt *testing.T) func(err error) {
		return func(err error) {
			errors := checker.RequireCheckerErrors(tt, err, 1)
			invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
			assert.ErrorAs(tt, errors[0], &invalidatedRefError)
		}
	}

	t.Run("on stack", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccountWithErrorHandler(t, address, true, nil, `
            resource R {
                access(all) var id: Int

                access(all) fun setID(_ id: Int) {
                    self.id = id
                }

                init() {
                    self.id = 1
                }
            }

            fun test() {
                let r <-create R()
                let ref = &r as &R

                destroy r

                // Update the reference
                ref.setID(2)
            }`, sema.Config{}, errorHandler(t))

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("ref source is field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(
			t,
			`
            access(all) fun test() {
                let r <- create R()
                let s = S()
                s.setB(&r as &R)

                let x = s.b!     // get reference from a struct field
                destroy r        // destroy the resource
                x.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }

            access(all) struct S {
                access(all) var b: &R?

                access(all) fun setB(_ b: &R) {
                    self.b = b
                }

                init() {
                    self.b = nil
                }
            }`,
		)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})

	})
}

func TestInterpretReferenceTrackingOnInvocation(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      access(all) resource Foo {

          access(all) let id: UInt8

          init() {
              self.id = 12
          }

          access(all) fun something() {}
      }

      fun returnSameRef(_ ref: &Foo): &Foo {
          return ref
      }

      fun main() {
          var foo <- create Foo()
          var fooRef = &foo as &Foo

          // Invocation should not un-track the reference
          fooRef.something()

          // just to trick the checker
          fooRef = returnSameRef(fooRef)

          // Moving the resource should update the tracking
          var newFoo <- foo

            fooRef.id

            destroy newFoo
      }
    `)

	_, err := inter.Invoke("main")
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
}

func TestInterpretInvalidReferenceToOptionalConfusion(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct S {
         fun foo() {}
      }

      fun main() {
        let y: AnyStruct? = nil
        let z: AnyStruct = y
        let ref = &z as &AnyStruct
        let s = ref as! &S
        s.foo()
      }
    `)

	_, err := inter.Invoke("main")
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
}

func TestInterpretReferenceToOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun main(): AnyStruct {
        let y: Int? = nil
        let z: AnyStruct = y
        return &z as &AnyStruct
      }
    `)

	value, err := inter.Invoke("main")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		&interpreter.EphemeralReferenceValue{
			Value:         interpreter.Nil,
			BorrowedType:  sema.AnyStructType,
			Authorization: interpreter.UnauthorizedAccess,
		},
		value,
	)
}

func TestInterpretInvalidatedReferenceToOptional(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        resource Foo {}

        fun main(): AnyStruct {
            let y: @Foo? <- create Foo()
            let z: @AnyResource <- y

            var ref1 = &z as &AnyResource

            var ref2 = returnSameRef(ref1)

            destroy z
            return ref2
        }

        fun returnSameRef(_ ref: &AnyResource): &AnyResource {
            return ref
        }
    `)

	_, err := inter.Invoke("main")
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
}

func TestInterpretReferenceToReference(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x = &1 as &Int
                let y = &x as & &Int
            }
        `, ParseCheckAndInterpretOptions{
			HandleCheckerError: func(err error) {
				errs := checker.RequireCheckerErrors(t, err, 1)
				require.IsType(t, &sema.NestedReferenceError{}, errs[0])
			},
		})
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.NestedReferenceError{})
	})

	t.Run("upcast to anystruct", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun main() {
                let x = &1 as &Int as AnyStruct
                let y = &x as &AnyStruct
            }
        `)

		_, err := inter.Invoke("main")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.NestedReferenceError{})
	})

	t.Run("optional", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x: (&Int)? = &1 as &Int
                let y: (&(&Int))? = &x 
            }
        `, ParseCheckAndInterpretOptions{
			HandleCheckerError: func(err error) {
				errs := checker.RequireCheckerErrors(t, err, 1)
				require.IsType(t, &sema.NestedReferenceError{}, errs[0])
			},
		})
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.NestedReferenceError{})
	})

	t.Run("upcast to optional anystruct", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun main() {
                let x = &1 as &Int as AnyStruct?
                let y = &x as &AnyStruct?
            }
        `)

		_, err := inter.Invoke("main")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.NestedReferenceError{})
	})

	t.Run("reference to storage reference", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            resource R {}

            fun test(): Void {

                let r <- [<- create R()]
                account.storage.save(<-r, to: /storage/foo)
                let unauthRef = account.storage.borrow<&[R]>(from: /storage/foo)!

                let maskedUnauthRef = unauthRef as AnyStruct
                let doubleRef = &maskedUnauthRef as auth(Mutate) &AnyStruct
                let typedDoubleRef : auth(Mutate) &(&[R]) = doubleRef as! auth(Mutate) &(&[R])
            }`, sema.Config{})

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.NestedReferenceError{})
	})
}

func TestInterpretDereference(t *testing.T) {
	t.Parallel()

	runTestCase := func(
		t *testing.T,
		name, code string,
		expectedValueFunc func(*interpreter.Interpreter) interpreter.Value,
	) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inter := parseCheckAndInterpret(t, code)

			value, err := inter.Invoke("main")
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				expectedValueFunc(inter),
				value,
			)
		})
	}

	t.Run("Integers", func(t *testing.T) {
		t.Parallel()

		expectedValues := map[sema.Type]interpreter.IntegerValue{
			sema.IntType:     interpreter.NewUnmeteredIntValueFromInt64(42),
			sema.UIntType:    interpreter.NewUnmeteredUIntValueFromUint64(42),
			sema.UInt8Type:   interpreter.NewUnmeteredUInt8Value(42),
			sema.UInt16Type:  interpreter.NewUnmeteredUInt16Value(42),
			sema.UInt32Type:  interpreter.NewUnmeteredUInt32Value(42),
			sema.UInt64Type:  interpreter.NewUnmeteredUInt64Value(42),
			sema.UInt128Type: interpreter.NewUnmeteredUInt128ValueFromUint64(42),
			sema.UInt256Type: interpreter.NewUnmeteredUInt256ValueFromUint64(42),
			sema.Word8Type:   interpreter.NewUnmeteredWord8Value(42),
			sema.Word16Type:  interpreter.NewUnmeteredWord16Value(42),
			sema.Word32Type:  interpreter.NewUnmeteredWord32Value(42),
			sema.Word64Type:  interpreter.NewUnmeteredWord64Value(42),
			sema.Word128Type: interpreter.NewUnmeteredWord128ValueFromUint64(42),
			sema.Word256Type: interpreter.NewUnmeteredWord256ValueFromUint64(42),
			sema.Int8Type:    interpreter.NewUnmeteredInt8Value(42),
			sema.Int16Type:   interpreter.NewUnmeteredInt16Value(42),
			sema.Int32Type:   interpreter.NewUnmeteredInt32Value(42),
			sema.Int64Type:   interpreter.NewUnmeteredInt64Value(42),
			sema.Int128Type:  interpreter.NewUnmeteredInt128ValueFromInt64(42),
			sema.Int256Type:  interpreter.NewUnmeteredInt256ValueFromInt64(42),
		}

		for _, typ := range sema.AllIntegerTypes {
			// Only test leaf types
			switch typ {
			case sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType:
				continue
			}

			integerType := typ
			typString := typ.QualifiedString()

			runTestCase(
				t,
				typString,
				fmt.Sprintf(
					`
                        fun main(): %[1]s {
                            let x: &%[1]s = &42
                            return *x
                        }
                    `,
					integerType,
				),
				func(_ *interpreter.Interpreter) interpreter.Value {
					return expectedValues[integerType]
				},
			)
		}
	})

	t.Run("Fixed-point numbers", func(t *testing.T) {
		t.Parallel()

		expectedValues := map[sema.Type]interpreter.FixedPointValue{
			sema.UFix64Type: interpreter.NewUnmeteredUFix64Value(4224_000_000),
			sema.Fix64Type:  interpreter.NewUnmeteredFix64Value(4224_000_000),
		}

		for _, typ := range sema.AllFixedPointTypes {
			// Only test leaf types
			switch typ {
			case sema.FixedPointType, sema.SignedFixedPointType:
				continue
			}

			fixedPointType := typ
			typString := typ.QualifiedString()

			runTestCase(
				t,
				typString,
				fmt.Sprintf(
					`
                        fun main(): %[1]s {
                            let x: &%[1]s = &42.24
                            return *x
                        }
                    `,
					fixedPointType,
				),
				func(_ *interpreter.Interpreter) interpreter.Value {
					return expectedValues[fixedPointType]
				},
			)
		}
	})

	t.Run("Variable-sized array of integers", func(t *testing.T) {
		t.Parallel()

		for _, typ := range sema.AllIntegerTypes {
			// Only test leaf types
			switch typ {
			case sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType:
				continue
			}

			integerType := typ
			typString := typ.QualifiedString()

			createArrayValue := func(
				inter *interpreter.Interpreter,
				innerStaticType interpreter.StaticType,
				values ...interpreter.Value,
			) interpreter.Value {
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.VariableSizedStaticType{
						Type: innerStaticType,
					},
					common.ZeroAddress,
					values...,
				)
			}

			t.Run(fmt.Sprintf("[%s]", typString), func(t *testing.T) {
				inter := parseCheckAndInterpret(
					t,
					fmt.Sprintf(
						`
                            let originalArray: [%[1]s] = [1, 2, 3]

                            fun main(): [%[1]s] {
                                let ref: &[%[1]s] = &originalArray

                                // Even a temporary value shouldn't affect originalArray.
                                (*ref).append(4)

                                let deref = *ref
                                deref.append(4)
                                return deref
                            }
                        `,
						integerType,
					),
				)

				value, err := inter.Invoke("main")
				require.NoError(t, err)

				var expectedValue, expectedOriginalValue interpreter.Value
				switch integerType {
				// Int*
				case sema.IntType:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt,
						interpreter.NewUnmeteredIntValueFromInt64(1),
						interpreter.NewUnmeteredIntValueFromInt64(2),
						interpreter.NewUnmeteredIntValueFromInt64(3),
						interpreter.NewUnmeteredIntValueFromInt64(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt,
						interpreter.NewUnmeteredIntValueFromInt64(1),
						interpreter.NewUnmeteredIntValueFromInt64(2),
						interpreter.NewUnmeteredIntValueFromInt64(3),
					)

				case sema.Int8Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt8,
						interpreter.NewUnmeteredInt8Value(1),
						interpreter.NewUnmeteredInt8Value(2),
						interpreter.NewUnmeteredInt8Value(3),
						interpreter.NewUnmeteredInt8Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt8,
						interpreter.NewUnmeteredInt8Value(1),
						interpreter.NewUnmeteredInt8Value(2),
						interpreter.NewUnmeteredInt8Value(3),
					)

				case sema.Int16Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt16,
						interpreter.NewUnmeteredInt16Value(1),
						interpreter.NewUnmeteredInt16Value(2),
						interpreter.NewUnmeteredInt16Value(3),
						interpreter.NewUnmeteredInt16Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt16,
						interpreter.NewUnmeteredInt16Value(1),
						interpreter.NewUnmeteredInt16Value(2),
						interpreter.NewUnmeteredInt16Value(3),
					)

				case sema.Int32Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt32,
						interpreter.NewUnmeteredInt32Value(1),
						interpreter.NewUnmeteredInt32Value(2),
						interpreter.NewUnmeteredInt32Value(3),
						interpreter.NewUnmeteredInt32Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt32,
						interpreter.NewUnmeteredInt32Value(1),
						interpreter.NewUnmeteredInt32Value(2),
						interpreter.NewUnmeteredInt32Value(3),
					)

				case sema.Int64Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt64,
						interpreter.NewUnmeteredInt64Value(1),
						interpreter.NewUnmeteredInt64Value(2),
						interpreter.NewUnmeteredInt64Value(3),
						interpreter.NewUnmeteredInt64Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt64,
						interpreter.NewUnmeteredInt64Value(1),
						interpreter.NewUnmeteredInt64Value(2),
						interpreter.NewUnmeteredInt64Value(3),
					)

				case sema.Int128Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt128,
						interpreter.NewUnmeteredInt128ValueFromInt64(1),
						interpreter.NewUnmeteredInt128ValueFromInt64(2),
						interpreter.NewUnmeteredInt128ValueFromInt64(3),
						interpreter.NewUnmeteredInt128ValueFromInt64(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt128,
						interpreter.NewUnmeteredInt128ValueFromInt64(1),
						interpreter.NewUnmeteredInt128ValueFromInt64(2),
						interpreter.NewUnmeteredInt128ValueFromInt64(3),
					)

				case sema.Int256Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt256,
						interpreter.NewUnmeteredInt256ValueFromInt64(1),
						interpreter.NewUnmeteredInt256ValueFromInt64(2),
						interpreter.NewUnmeteredInt256ValueFromInt64(3),
						interpreter.NewUnmeteredInt256ValueFromInt64(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt256,
						interpreter.NewUnmeteredInt256ValueFromInt64(1),
						interpreter.NewUnmeteredInt256ValueFromInt64(2),
						interpreter.NewUnmeteredInt256ValueFromInt64(3),
					)

				// UInt*
				case sema.UIntType:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt,
						interpreter.NewUnmeteredUIntValueFromUint64(1),
						interpreter.NewUnmeteredUIntValueFromUint64(2),
						interpreter.NewUnmeteredUIntValueFromUint64(3),
						interpreter.NewUnmeteredUIntValueFromUint64(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt,
						interpreter.NewUnmeteredUIntValueFromUint64(1),
						interpreter.NewUnmeteredUIntValueFromUint64(2),
						interpreter.NewUnmeteredUIntValueFromUint64(3),
					)

				case sema.UInt8Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt8,
						interpreter.NewUnmeteredUInt8Value(1),
						interpreter.NewUnmeteredUInt8Value(2),
						interpreter.NewUnmeteredUInt8Value(3),
						interpreter.NewUnmeteredUInt8Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt8,
						interpreter.NewUnmeteredUInt8Value(1),
						interpreter.NewUnmeteredUInt8Value(2),
						interpreter.NewUnmeteredUInt8Value(3),
					)

				case sema.UInt16Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt16,
						interpreter.NewUnmeteredUInt16Value(1),
						interpreter.NewUnmeteredUInt16Value(2),
						interpreter.NewUnmeteredUInt16Value(3),
						interpreter.NewUnmeteredUInt16Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt16,
						interpreter.NewUnmeteredUInt16Value(1),
						interpreter.NewUnmeteredUInt16Value(2),
						interpreter.NewUnmeteredUInt16Value(3),
					)

				case sema.UInt32Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt32,
						interpreter.NewUnmeteredUInt32Value(1),
						interpreter.NewUnmeteredUInt32Value(2),
						interpreter.NewUnmeteredUInt32Value(3),
						interpreter.NewUnmeteredUInt32Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt32,
						interpreter.NewUnmeteredUInt32Value(1),
						interpreter.NewUnmeteredUInt32Value(2),
						interpreter.NewUnmeteredUInt32Value(3),
					)

				case sema.UInt64Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt64,
						interpreter.NewUnmeteredUInt64Value(1),
						interpreter.NewUnmeteredUInt64Value(2),
						interpreter.NewUnmeteredUInt64Value(3),
						interpreter.NewUnmeteredUInt64Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt64,
						interpreter.NewUnmeteredUInt64Value(1),
						interpreter.NewUnmeteredUInt64Value(2),
						interpreter.NewUnmeteredUInt64Value(3),
					)

				case sema.UInt128Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt128,
						interpreter.NewUnmeteredUInt128ValueFromUint64(1),
						interpreter.NewUnmeteredUInt128ValueFromUint64(2),
						interpreter.NewUnmeteredUInt128ValueFromUint64(3),
						interpreter.NewUnmeteredUInt128ValueFromUint64(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt128,
						interpreter.NewUnmeteredUInt128ValueFromUint64(1),
						interpreter.NewUnmeteredUInt128ValueFromUint64(2),
						interpreter.NewUnmeteredUInt128ValueFromUint64(3),
					)

				case sema.UInt256Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt256,
						interpreter.NewUnmeteredUInt256ValueFromUint64(1),
						interpreter.NewUnmeteredUInt256ValueFromUint64(2),
						interpreter.NewUnmeteredUInt256ValueFromUint64(3),
						interpreter.NewUnmeteredUInt256ValueFromUint64(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt256,
						interpreter.NewUnmeteredUInt256ValueFromUint64(1),
						interpreter.NewUnmeteredUInt256ValueFromUint64(2),
						interpreter.NewUnmeteredUInt256ValueFromUint64(3),
					)

				// Word*
				case sema.Word8Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord8,
						interpreter.NewUnmeteredWord8Value(1),
						interpreter.NewUnmeteredWord8Value(2),
						interpreter.NewUnmeteredWord8Value(3),
						interpreter.NewUnmeteredWord8Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord8,
						interpreter.NewUnmeteredWord8Value(1),
						interpreter.NewUnmeteredWord8Value(2),
						interpreter.NewUnmeteredWord8Value(3),
					)

				case sema.Word16Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord16,
						interpreter.NewUnmeteredWord16Value(1),
						interpreter.NewUnmeteredWord16Value(2),
						interpreter.NewUnmeteredWord16Value(3),
						interpreter.NewUnmeteredWord16Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord16,
						interpreter.NewUnmeteredWord16Value(1),
						interpreter.NewUnmeteredWord16Value(2),
						interpreter.NewUnmeteredWord16Value(3),
					)

				case sema.Word32Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord32,
						interpreter.NewUnmeteredWord32Value(1),
						interpreter.NewUnmeteredWord32Value(2),
						interpreter.NewUnmeteredWord32Value(3),
						interpreter.NewUnmeteredWord32Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord32,
						interpreter.NewUnmeteredWord32Value(1),
						interpreter.NewUnmeteredWord32Value(2),
						interpreter.NewUnmeteredWord32Value(3),
					)

				case sema.Word64Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord64,
						interpreter.NewUnmeteredWord64Value(1),
						interpreter.NewUnmeteredWord64Value(2),
						interpreter.NewUnmeteredWord64Value(3),
						interpreter.NewUnmeteredWord64Value(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord64,
						interpreter.NewUnmeteredWord64Value(1),
						interpreter.NewUnmeteredWord64Value(2),
						interpreter.NewUnmeteredWord64Value(3),
					)

				case sema.Word128Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord128,
						interpreter.NewUnmeteredWord128ValueFromUint64(1),
						interpreter.NewUnmeteredWord128ValueFromUint64(2),
						interpreter.NewUnmeteredWord128ValueFromUint64(3),
						interpreter.NewUnmeteredWord128ValueFromUint64(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord128,
						interpreter.NewUnmeteredWord128ValueFromUint64(1),
						interpreter.NewUnmeteredWord128ValueFromUint64(2),
						interpreter.NewUnmeteredWord128ValueFromUint64(3),
					)

				case sema.Word256Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord256,
						interpreter.NewUnmeteredWord256ValueFromUint64(1),
						interpreter.NewUnmeteredWord256ValueFromUint64(2),
						interpreter.NewUnmeteredWord256ValueFromUint64(3),
						interpreter.NewUnmeteredWord256ValueFromUint64(4),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord256,
						interpreter.NewUnmeteredWord256ValueFromUint64(1),
						interpreter.NewUnmeteredWord256ValueFromUint64(2),
						interpreter.NewUnmeteredWord256ValueFromUint64(3),
					)
				}

				AssertValuesEqual(
					t,
					inter,
					expectedValue,
					value,
				)

				AssertValuesEqual(
					t,
					inter,
					expectedOriginalValue,
					inter.Globals.Get("originalArray").GetValue(),
				)
			})
		}
	})

	t.Run("Constant-sized array of integers", func(t *testing.T) {
		t.Parallel()

		for _, typ := range sema.AllIntegerTypes {
			// Only test leaf types
			switch typ {
			case sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType:
				continue
			}

			integerType := typ
			typString := typ.QualifiedString()

			createArrayValue := func(
				inter *interpreter.Interpreter,
				innerStaticType interpreter.StaticType,
				values ...interpreter.Value,
			) interpreter.Value {
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.ConstantSizedStaticType{
						Type: innerStaticType,
						Size: 3,
					},
					common.ZeroAddress,
					values...,
				)
			}

			t.Run(fmt.Sprintf("[%s]", typString), func(t *testing.T) {
				inter := parseCheckAndInterpret(
					t,
					fmt.Sprintf(
						`
                            let originalArray: [%[1]s; 3] = [1, 2, 3]

                            fun main(): [%[1]s; 3] {
                                let ref: &[%[1]s; 3] = &originalArray

                                let deref = *ref
                                deref[2] = 30
                                return deref
                            }
                        `,
						integerType,
					),
				)

				value, err := inter.Invoke("main")
				require.NoError(t, err)

				var expectedValue, expectedOriginalValue interpreter.Value
				switch integerType {
				// Int*
				case sema.IntType:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt,
						interpreter.NewUnmeteredIntValueFromInt64(1),
						interpreter.NewUnmeteredIntValueFromInt64(2),
						interpreter.NewUnmeteredIntValueFromInt64(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt,
						interpreter.NewUnmeteredIntValueFromInt64(1),
						interpreter.NewUnmeteredIntValueFromInt64(2),
						interpreter.NewUnmeteredIntValueFromInt64(3),
					)

				case sema.Int8Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt8,
						interpreter.NewUnmeteredInt8Value(1),
						interpreter.NewUnmeteredInt8Value(2),
						interpreter.NewUnmeteredInt8Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt8,
						interpreter.NewUnmeteredInt8Value(1),
						interpreter.NewUnmeteredInt8Value(2),
						interpreter.NewUnmeteredInt8Value(3),
					)

				case sema.Int16Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt16,
						interpreter.NewUnmeteredInt16Value(1),
						interpreter.NewUnmeteredInt16Value(2),
						interpreter.NewUnmeteredInt16Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt16,
						interpreter.NewUnmeteredInt16Value(1),
						interpreter.NewUnmeteredInt16Value(2),
						interpreter.NewUnmeteredInt16Value(3),
					)

				case sema.Int32Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt32,
						interpreter.NewUnmeteredInt32Value(1),
						interpreter.NewUnmeteredInt32Value(2),
						interpreter.NewUnmeteredInt32Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt32,
						interpreter.NewUnmeteredInt32Value(1),
						interpreter.NewUnmeteredInt32Value(2),
						interpreter.NewUnmeteredInt32Value(3),
					)

				case sema.Int64Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt64,
						interpreter.NewUnmeteredInt64Value(1),
						interpreter.NewUnmeteredInt64Value(2),
						interpreter.NewUnmeteredInt64Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt64,
						interpreter.NewUnmeteredInt64Value(1),
						interpreter.NewUnmeteredInt64Value(2),
						interpreter.NewUnmeteredInt64Value(3),
					)

				case sema.Int128Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt128,
						interpreter.NewUnmeteredInt128ValueFromInt64(1),
						interpreter.NewUnmeteredInt128ValueFromInt64(2),
						interpreter.NewUnmeteredInt128ValueFromInt64(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt128,
						interpreter.NewUnmeteredInt128ValueFromInt64(1),
						interpreter.NewUnmeteredInt128ValueFromInt64(2),
						interpreter.NewUnmeteredInt128ValueFromInt64(3),
					)

				case sema.Int256Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt256,
						interpreter.NewUnmeteredInt256ValueFromInt64(1),
						interpreter.NewUnmeteredInt256ValueFromInt64(2),
						interpreter.NewUnmeteredInt256ValueFromInt64(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeInt256,
						interpreter.NewUnmeteredInt256ValueFromInt64(1),
						interpreter.NewUnmeteredInt256ValueFromInt64(2),
						interpreter.NewUnmeteredInt256ValueFromInt64(3),
					)

				// UInt*
				case sema.UIntType:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt,
						interpreter.NewUnmeteredUIntValueFromUint64(1),
						interpreter.NewUnmeteredUIntValueFromUint64(2),
						interpreter.NewUnmeteredUIntValueFromUint64(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt,
						interpreter.NewUnmeteredUIntValueFromUint64(1),
						interpreter.NewUnmeteredUIntValueFromUint64(2),
						interpreter.NewUnmeteredUIntValueFromUint64(3),
					)

				case sema.UInt8Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt8,
						interpreter.NewUnmeteredUInt8Value(1),
						interpreter.NewUnmeteredUInt8Value(2),
						interpreter.NewUnmeteredUInt8Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt8,
						interpreter.NewUnmeteredUInt8Value(1),
						interpreter.NewUnmeteredUInt8Value(2),
						interpreter.NewUnmeteredUInt8Value(3),
					)

				case sema.UInt16Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt16,
						interpreter.NewUnmeteredUInt16Value(1),
						interpreter.NewUnmeteredUInt16Value(2),
						interpreter.NewUnmeteredUInt16Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt16,
						interpreter.NewUnmeteredUInt16Value(1),
						interpreter.NewUnmeteredUInt16Value(2),
						interpreter.NewUnmeteredUInt16Value(3),
					)

				case sema.UInt32Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt32,
						interpreter.NewUnmeteredUInt32Value(1),
						interpreter.NewUnmeteredUInt32Value(2),
						interpreter.NewUnmeteredUInt32Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt32,
						interpreter.NewUnmeteredUInt32Value(1),
						interpreter.NewUnmeteredUInt32Value(2),
						interpreter.NewUnmeteredUInt32Value(3),
					)

				case sema.UInt64Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt64,
						interpreter.NewUnmeteredUInt64Value(1),
						interpreter.NewUnmeteredUInt64Value(2),
						interpreter.NewUnmeteredUInt64Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt64,
						interpreter.NewUnmeteredUInt64Value(1),
						interpreter.NewUnmeteredUInt64Value(2),
						interpreter.NewUnmeteredUInt64Value(3),
					)

				case sema.UInt128Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt128,
						interpreter.NewUnmeteredUInt128ValueFromUint64(1),
						interpreter.NewUnmeteredUInt128ValueFromUint64(2),
						interpreter.NewUnmeteredUInt128ValueFromUint64(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt128,
						interpreter.NewUnmeteredUInt128ValueFromUint64(1),
						interpreter.NewUnmeteredUInt128ValueFromUint64(2),
						interpreter.NewUnmeteredUInt128ValueFromUint64(3),
					)

				case sema.UInt256Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt256,
						interpreter.NewUnmeteredUInt256ValueFromUint64(1),
						interpreter.NewUnmeteredUInt256ValueFromUint64(2),
						interpreter.NewUnmeteredUInt256ValueFromUint64(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeUInt256,
						interpreter.NewUnmeteredUInt256ValueFromUint64(1),
						interpreter.NewUnmeteredUInt256ValueFromUint64(2),
						interpreter.NewUnmeteredUInt256ValueFromUint64(3),
					)

				// Word*
				case sema.Word8Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord8,
						interpreter.NewUnmeteredWord8Value(1),
						interpreter.NewUnmeteredWord8Value(2),
						interpreter.NewUnmeteredWord8Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord8,
						interpreter.NewUnmeteredWord8Value(1),
						interpreter.NewUnmeteredWord8Value(2),
						interpreter.NewUnmeteredWord8Value(3),
					)

				case sema.Word16Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord16,
						interpreter.NewUnmeteredWord16Value(1),
						interpreter.NewUnmeteredWord16Value(2),
						interpreter.NewUnmeteredWord16Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord16,
						interpreter.NewUnmeteredWord16Value(1),
						interpreter.NewUnmeteredWord16Value(2),
						interpreter.NewUnmeteredWord16Value(3),
					)

				case sema.Word32Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord32,
						interpreter.NewUnmeteredWord32Value(1),
						interpreter.NewUnmeteredWord32Value(2),
						interpreter.NewUnmeteredWord32Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord32,
						interpreter.NewUnmeteredWord32Value(1),
						interpreter.NewUnmeteredWord32Value(2),
						interpreter.NewUnmeteredWord32Value(3),
					)

				case sema.Word64Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord64,
						interpreter.NewUnmeteredWord64Value(1),
						interpreter.NewUnmeteredWord64Value(2),
						interpreter.NewUnmeteredWord64Value(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord64,
						interpreter.NewUnmeteredWord64Value(1),
						interpreter.NewUnmeteredWord64Value(2),
						interpreter.NewUnmeteredWord64Value(3),
					)

				case sema.Word128Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord128,
						interpreter.NewUnmeteredWord128ValueFromUint64(1),
						interpreter.NewUnmeteredWord128ValueFromUint64(2),
						interpreter.NewUnmeteredWord128ValueFromUint64(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord128,
						interpreter.NewUnmeteredWord128ValueFromUint64(1),
						interpreter.NewUnmeteredWord128ValueFromUint64(2),
						interpreter.NewUnmeteredWord128ValueFromUint64(3),
					)

				case sema.Word256Type:
					expectedValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord256,
						interpreter.NewUnmeteredWord256ValueFromUint64(1),
						interpreter.NewUnmeteredWord256ValueFromUint64(2),
						interpreter.NewUnmeteredWord256ValueFromUint64(30),
					)
					expectedOriginalValue = createArrayValue(
						inter,
						interpreter.PrimitiveStaticTypeWord256,
						interpreter.NewUnmeteredWord256ValueFromUint64(1),
						interpreter.NewUnmeteredWord256ValueFromUint64(2),
						interpreter.NewUnmeteredWord256ValueFromUint64(3),
					)
				}

				AssertValuesEqual(
					t,
					inter,
					expectedValue,
					value,
				)

				AssertValuesEqual(
					t,
					inter,
					expectedOriginalValue,
					inter.Globals.Get("originalArray").GetValue(),
				)
			})
		}
	})

	t.Run("Dictionary", func(t *testing.T) {
		t.Parallel()

		t.Run("{Int: String}", func(t *testing.T) {
			inter := parseCheckAndInterpret(
				t,
				`
                    fun main(): {Int: String} {
                        let original = {1: "ABC", 2: "DEF"}
                        let x: &{Int : String} = &original
                        return *x
                    }
                `,
			)

			value, err := inter.Invoke("main")
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeInt,
						ValueType: interpreter.PrimitiveStaticTypeString,
					},
					interpreter.NewUnmeteredIntValueFromInt64(1),
					interpreter.NewUnmeteredStringValue("ABC"),
					interpreter.NewUnmeteredIntValueFromInt64(2),
					interpreter.NewUnmeteredStringValue("DEF"),
				),
				value,
			)
		})

		t.Run("{Int: [String]}", func(t *testing.T) {
			inter := parseCheckAndInterpret(
				t,
				`
                    fun main(): {Int: [String]} {
                        let original = {1: ["ABC", "XYZ"], 2: ["DEF"]}
                        let x: &{Int: [String]} = &original
                        return *x
                    }
                `,
			)

			value, err := inter.Invoke("main")
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.DictionaryStaticType{
						KeyType: interpreter.PrimitiveStaticTypeInt,
						ValueType: &interpreter.VariableSizedStaticType{
							Type: interpreter.PrimitiveStaticTypeString,
						},
					},
					interpreter.NewUnmeteredIntValueFromInt64(1),
					interpreter.NewArrayValue(
						inter,
						interpreter.EmptyLocationRange,
						&interpreter.VariableSizedStaticType{
							Type: interpreter.PrimitiveStaticTypeString,
						},
						common.ZeroAddress,
						interpreter.NewUnmeteredStringValue("ABC"),
						interpreter.NewUnmeteredStringValue("XYZ"),
					),
					interpreter.NewUnmeteredIntValueFromInt64(2),
					interpreter.NewArrayValue(
						inter,
						interpreter.EmptyLocationRange,
						&interpreter.VariableSizedStaticType{
							Type: interpreter.PrimitiveStaticTypeString,
						},
						common.ZeroAddress,
						interpreter.NewUnmeteredStringValue("DEF"),
					),
				),
				value,
			)
		})
	})

	t.Run("Character", func(t *testing.T) {
		t.Parallel()

		runTestCase(
			t,
			"Character",
			`
                fun main(): Character {
                    let original: Character = "S"
                    let x: &Character = &original
                    return *x
                }
            `,
			func(_ *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewUnmeteredCharacterValue("S")
			},
		)
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		runTestCase(
			t,
			"String",
			`
                fun main(): String {
                    let original: String = "STxy"
                    let x: &String = &original
                    return *x
                }
            `,
			func(_ *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewUnmeteredStringValue("STxy")
			},
		)
	})

	runTestCase(
		t,
		"Bool",
		`
            fun main(): Bool {
                let original: Bool = true
                let x: &Bool = &original
                return *x
            }
        `,
		func(_ *interpreter.Interpreter) interpreter.Value {
			return interpreter.BoolValue(true)
		},
	)

	address, err := common.HexToAddress("0x0000000000000231")
	assert.NoError(t, err)

	runTestCase(
		t,
		"Address",
		`
            fun main(): Address {
                let original: Address = 0x0000000000000231
                let x: &Address = &original
                return *x
            }
        `,
		func(_ *interpreter.Interpreter) interpreter.Value {
			return interpreter.NewAddressValue(nil, address)
		},
	)

	t.Run("Path", func(t *testing.T) {
		t.Parallel()

		runTestCase(
			t,
			"PrivatePath",
			`
                fun main(): Path {
                    let original: Path = /private/temp
                    let x: &Path = &original
                    return *x
                }
            `,
			func(_ *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewUnmeteredPathValue(common.PathDomainPrivate, "temp")
			},
		)

		runTestCase(
			t,
			"PublicPath",
			`
                fun main(): Path {
                    let original: Path = /public/temp
                    let x: &Path = &original
                    return *x
                }
            `,
			func(_ *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewUnmeteredPathValue(common.PathDomainPublic, "temp")
			},
		)
	})

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		runTestCase(
			t,
			"nil",
			`
              fun main(): Int? {
                  let ref: &Int? = nil
                  return *ref
              }
            `,
			func(_ *interpreter.Interpreter) interpreter.Value {
				return interpreter.Nil
			},
		)

		runTestCase(
			t,
			"some",
			`
              fun main(): Int? {
                  let ref: &Int? = &42 as &Int
                  return *ref
              }
            `,
			func(_ *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewUnmeteredSomeValueNonCopying(
					interpreter.NewIntValueFromInt64(nil, 42),
				)
			},
		)
	})

	t.Run("Resource", func(t *testing.T) {
		t.Parallel()

		t.Run("direct", func(t *testing.T) {
			t.Parallel()

			inter, err := parseCheckAndInterpretWithOptions(t,
				`
                  resource R {}

                  fun main() {
                      let r1 <- create R()
                      let r1Ref: &R = &r1
                      let r2 <- *r1Ref
                      destroy r1
                      destroy r2
                  }
                `,
				ParseCheckAndInterpretOptions{
					HandleCheckerError: func(err error) {
						errs := checker.RequireCheckerErrors(t, err, 1)

						require.IsType(t, &sema.InvalidUnaryOperandError{}, errs[0])
					},
				},
			)
			require.NoError(t, err)

			_, err = inter.Invoke("main")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ResourceReferenceDereferenceError{})
		})

		t.Run("array", func(t *testing.T) {
			t.Parallel()

			inter, err := parseCheckAndInterpretWithOptions(t,
				`
                  resource R {}

                  fun main() {
                      let rs1 <- [<- create R()]
                      let rs1Ref: &[R] = &rs1
                      let rs2 <- *rs1Ref
                      destroy rs1
                      destroy rs2
                  }
                `,
				ParseCheckAndInterpretOptions{
					HandleCheckerError: func(err error) {
						errs := checker.RequireCheckerErrors(t, err, 1)

						require.IsType(t, &sema.InvalidUnaryOperandError{}, errs[0])
					},
				},
			)
			require.NoError(t, err)

			_, err = inter.Invoke("main")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ResourceReferenceDereferenceError{})
		})
	})

	t.Run("Struct", func(t *testing.T) {

		sStaticType := interpreter.NewCompositeStaticType(
			nil,
			TestLocation,
			"S",
			TestLocation.TypeID(nil, "S"),
		)

		newS := func(inter *interpreter.Interpreter) interpreter.Value {
			return interpreter.NewCompositeValue(
				inter,
				interpreter.EmptyLocationRange,
				TestLocation,
				"S",
				common.CompositeKindStructure,
				nil,
				common.ZeroAddress,
			)
		}

		runTestCase(
			t,
			"variable-sized array",
			`
		      struct S {}

		      fun main(): [S] {
		          let s1: [S] = [S()]
		          let s1Ref: &[S] = &s1
		          let s2 = *s1Ref
                  return s2
		      }
            `,
			func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewArrayValue(inter,
					interpreter.EmptyLocationRange,
					interpreter.NewVariableSizedStaticType(
						nil,
						sStaticType,
					),
					common.ZeroAddress,
					newS(inter),
				)
			},
		)

		runTestCase(
			t,
			"constant-sized array",
			`
		      struct S {}

		      fun main(): [S; 2] {
		          let s1: [S; 2] = [S(), S()]
		          let s1Ref: &[S; 2] = &s1
		          let s2 = *s1Ref
                  return s2
		      }
            `,
			func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewArrayValue(inter,
					interpreter.EmptyLocationRange,
					interpreter.NewConstantSizedStaticType(
						nil,
						sStaticType,
						2,
					),
					common.ZeroAddress,
					newS(inter),
					newS(inter),
				)
			},
		)

	})
}

func TestInterpretOptionalReference(t *testing.T) {

	t.Parallel()

	t.Run("present", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          fun present(): &Int {
              let x: Int? = 1
              let y = &x as &Int?
              return y!
          }
        `)

		value, err := inter.Invoke("present")
		require.NoError(t, err)
		require.Equal(
			t,
			&interpreter.EphemeralReferenceValue{
				Value:         interpreter.NewUnmeteredIntValueFromInt64(1),
				BorrowedType:  sema.IntType,
				Authorization: interpreter.UnauthorizedAccess,
			},
			value,
		)

	})

	t.Run("absent", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun absent(): &Int {
              let x: Int? = nil
              let y = &x as &Int?
              return y!
          }
        `)

		_, err := inter.Invoke("absent")
		RequireError(t, err)

		var forceNilError interpreter.ForceNilError
		require.ErrorAs(t, err, &forceNilError)
	})

	t.Run("nested optional reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun main() {
                var dict: {String: Foo?} = {}
                var ref: (&Foo)?? = &dict["foo"] as &Foo??
            }

            struct Foo {}
        `)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
	})

	t.Run("reference to nested optional", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun main() {
                var dict: {String: Foo?} = {}
                var ref: &(Foo??) = &dict["foo"] as &(Foo??)
            }

            struct Foo {}
        `)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
	})
}
