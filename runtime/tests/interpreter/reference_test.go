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
              let dictRef = &dict as &{Int: &AnyStruct}

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
              let dictRef = &dict as &{Int: AnyStruct}

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
             let dictRef = &dict as &{Int: &AnyStruct}

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
             let dictRef = &dict as &{Int: AnyStruct}

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

              let dictRef = &dict as &{Int: &AnyStruct}
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
              let dictRef = &dict as &{Int: AnyStruct}

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

              let dictRef = &dict as &{Int: AnyStruct}
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
              let dictRef = &dict as &{Int: AnyStruct}

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
              let dictRef = &dict as &{Int: AnyStruct}

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
              let dictRef = &dict as &{Int: AnyStruct}

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

	t.Run("upcast to optional", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let i: Int = 1
          let ref = &i as &Int?
        `)

		value := inter.Globals.Get("ref").GetValue()
		require.IsType(t, &interpreter.SomeValue{}, value)

		innerValue := value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.EmptyLocationRange)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, innerValue)
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

		inter, _ := testAccountWithErrorHandler(
			t,
			address,
			true,
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
                let r <-create R()
                let ref = &r as &R

                // Move the resource into the account
                account.save(<-r, to: /storage/r)

                // Update the reference
                ref.setID(2)
            }`,
			sema.Config{},
			errorHandler(t),
		)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("stack to account readonly", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccountWithErrorHandler(
			t,
			address,
			true,
			`
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
                account.save(<-r, to: /storage/r)

                // 'Read' a field from the reference
                let id = ref.id
            }`,
			sema.Config{},
			errorHandler(t),
		)

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

            fun test(target: &[R]) {
                target.append(<- create R())

                // Take reference while in the account
                let ref = &target[0] as &R

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
			interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			address,
		)

		arrayRef := interpreter.NewUnmeteredEphemeralReferenceValue(
			interpreter.UnauthorizedAccess,
			array,
			&sema.VariableSizedType{
				Type: rType,
			},
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

            fun test(target1: &[R], target2: &[R]) {
                target1.append(<- create R())

                // Take reference while in the account_1
                let ref = &target1[0] as &R

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
			interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			common.Address{0x1},
		)

		arrayRef1 := interpreter.NewUnmeteredEphemeralReferenceValue(
			interpreter.UnauthorizedAccess,
			array1,
			&sema.VariableSizedType{
				Type: rType,
			},
		)

		// Resource array in account 0x02

		array2 := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			common.Address{0x2},
		)

		arrayRef2 := interpreter.NewUnmeteredEphemeralReferenceValue(
			interpreter.UnauthorizedAccess,
			array2,
			&sema.VariableSizedType{
				Type: rType,
			},
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

            fun test(target: &[R]): Int {
                target.append(<- create R())

                // Take reference while in the account
                let ref = &target[0] as &R

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
			interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			address,
		)

		arrayRef := interpreter.NewUnmeteredEphemeralReferenceValue(
			interpreter.UnauthorizedAccess,
			array,
			&sema.VariableSizedType{
				Type: rType,
			},
		)

		_, err := inter.Invoke("test", arrayRef)
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("account to stack storage reference", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(
			t,
			address,
			true,
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
                account.save(<-r1, to: /storage/r)

                let r1Ref = account.borrow<&R>(from: /storage/r)!

                let r2 <- account.load<@R>(from: /storage/r)!

                r1Ref.setID(2)
                destroy r2
            }`,
			sema.Config{},
		)

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

            fun setup(collection: &[R]) {
                collection.append(<- create R())

                // Take reference while in the account
                ref1 = &collection[0] as &R

                // Move the resource out of the account onto the stack. This should invalidate ref1.
                let movedR <- collection.remove(at: 0)

                // Take a reference while on stack
                ref2 = &movedR as &R

                // Append an extra resource just to force an index change
                collection.append(<- create R())

                // Move the resource again into the account (now at a different index)
                collection.append(<- movedR)

                // Take another reference
                ref3 = &collection[1] as &R
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
			interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			address,
		)

		arrayRef := interpreter.NewUnmeteredEphemeralReferenceValue(
			interpreter.UnauthorizedAccess,
			array,
			&sema.VariableSizedType{
				Type: rType,
			},
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

		inter, _ := testAccountWithErrorHandler(
			t,
			address,
			true,
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
                let r <-create R()
                let ref = &r as &R

                destroy r

                // Update the reference
                ref.setID(2)
            }`,
			sema.Config{},
			errorHandler(t),
		)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})
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
		require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})

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
	require.Error(t, err)

	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
}
