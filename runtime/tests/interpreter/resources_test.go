/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretOptionalResourceBindingWithSecondValue(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {
          let field: Int

          init() {
              self.field = 1
          }
      }

      resource Test {

          var r: @R?

          init() {
              self.r <- create R()
          }

          destroy () {
              destroy self.r
          }

          fun duplicate(): @R? {
              if let r <- self.r <- nil {
                  let r2 <- self.r <- nil
                  self.r <-! r2
                  return <-r
              } else {
                  return nil
              }
          }
      }

      fun test(): Bool {
          let test <- create Test()

          let copy <- test.duplicate()

          // "copy" here is actually expected to hold resource,
          // the important test is that the field was properly set to nil
          let res = copy != nil && test.r == nil

          destroy copy
          destroy test

          return res
      }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)
	require.Equal(t, interpreter.BoolValue(true), result)
}

func TestInterpretImplicitResourceRemovalFromContainer(t *testing.T) {

	t.Parallel()

	t.Run("resource, shift statement, member expression", func(t *testing.T) {

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

                init(r2: @R2) {
                    self.r2 <- r2
                }

                destroy() {
                    destroy self.r2
                }
            }

            fun test(): String? {
                let r1 <- create R1(r2: <- create R2())
                // The second assignment should not lead to the resource being cleared,
                // it must be fully moved out of this container before,
                // not just assigned to the new variable
                let optR2 <- r1.r2 <- nil
                let value = optR2?.value
                destroy r1
                destroy optR2
                return value
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewStringValue("test"),
			),
			value,
		)
	})

	t.Run("reference, shift statement, member expression", func(t *testing.T) {

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
            }

            fun createR1(): @R1 {
                return <- create R1()
            }

            fun test(r1: &R1): String? {
                r1.r2 <-! create R2()
                // The second assignment should not lead to the resource being cleared,
                // it must be fully moved out of this container before,
                // not just assigned to the new variable
                let optR2 <- r1.r2 <- nil
                let value = optR2?.value
                destroy optR2
                return value
            }
        `)

		r1, err := inter.Invoke("createR1")
		require.NoError(t, err)

		r1 = r1.Transfer(inter, interpreter.ReturnEmptyLocationRange, atree.Address{1}, false, nil)

		r1Type := checker.RequireGlobalType(t, inter.Program.Elaboration, "R1")

		ref := &interpreter.EphemeralReferenceValue{
			Value:        r1,
			BorrowedType: r1Type,
		}

		value, err := inter.Invoke("test", ref)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewStringValue("test"),
			),
			value,
		)
	})

	t.Run("resource, if-let statement, member expression", func(t *testing.T) {

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

                init(r2: @R2) {
                    self.r2 <- r2
                }

                destroy() {
                    destroy self.r2
                }
            }

            fun test(): String? {
                let r1 <- create R1(r2: <- create R2())
                // The second assignment should not lead to the resource being cleared,
                // it must be fully moved out of this container before,
                // not just assigned to the new variable
                if let r2 <- r1.r2 <- nil {
                    let value = r2.value
                    destroy r1
                    destroy r2
                    return value
                }
                destroy r1
                return nil
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewStringValue("test"),
			),
			value,
		)
	})

	t.Run("reference, if-let statement, member expression", func(t *testing.T) {

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
            }

            fun createR1(): @R1 {
                return <- create R1()
            }

            fun test(r1: &R1): String? {
                r1.r2 <-! create R2()
                // The second assignment should not lead to the resource being cleared,
                // it must be fully moved out of this container before,
                // not just assigned to the new variable
                if let r2 <- r1.r2 <- nil {
                    let value = r2.value
                    destroy r2
                    return value
                }
                return nil
            }
        `)

		r1, err := inter.Invoke("createR1")
		require.NoError(t, err)

		r1 = r1.Transfer(inter, interpreter.ReturnEmptyLocationRange, atree.Address{1}, false, nil)

		r1Type := checker.RequireGlobalType(t, inter.Program.Elaboration, "R1")

		ref := &interpreter.EphemeralReferenceValue{
			Value:        r1,
			BorrowedType: r1Type,
		}

		value, err := inter.Invoke("test", ref)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewStringValue("test"),
			),
			value,
		)
	})

	t.Run("resource, shift statement, index expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R2 {
                let value: String

                init() {
                    self.value = "test"
                }
            }

            resource R1 {
                var r2s: @{Int: R2}

                init(r2s: @{Int: R2}) {
                    self.r2s <- r2s
                }

                destroy() {
                    destroy self.r2s
                }
            }

            fun test(): String? {
                let r1 <- create R1(r2s: <-{0: <-create R2()})
                // The second assignment should not lead to the resource being cleared,
                // it must be fully moved out of this container before,
                // not just assigned to the new variable
                let optR2 <- r1.r2s[0] <- nil
                let value = optR2?.value
                destroy r1
                destroy optR2
                return value
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewStringValue("test"),
			),
			value,
		)
	})

	t.Run("reference, shift statement, index expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R2 {
                let value: String

                init() {
                    self.value = "test"
                }
            }

            resource R1 {
                pub(set) var r2s: @{Int: R2}

                init() {
                    self.r2s <- {}
                }

                destroy() {
                    destroy self.r2s
                }
            }

            fun createR1(): @R1 {
                return <- create R1()
            }

            fun test(r1: &R1): String? {
                r1.r2s[0] <-! create R2()
                // The second assignment should not lead to the resource being cleared,
                // it must be fully moved out of this container before,
                // not just assigned to the new variable
                let optR2 <- r1.r2s[0] <- nil
                let value = optR2?.value
                destroy optR2
                return value
            }
        `)

		r1, err := inter.Invoke("createR1")
		require.NoError(t, err)

		r1 = r1.Transfer(inter, interpreter.ReturnEmptyLocationRange, atree.Address{1}, false, nil)

		r1Type := checker.RequireGlobalType(t, inter.Program.Elaboration, "R1")

		ref := &interpreter.EphemeralReferenceValue{
			Value:        r1,
			BorrowedType: r1Type,
		}

		value, err := inter.Invoke("test", ref)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewStringValue("test"),
			),
			value,
		)
	})

	t.Run("resource, if-let statement, index expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R2 {
                let value: String

                init() {
                    self.value = "test"
                }
            }

            resource R1 {
                var r2s: @{Int: R2}

                init(r2s: @{Int: R2}) {
                    self.r2s <- r2s
                }

                destroy() {
                    destroy self.r2s
                }
            }

            fun test(): String? {
                let r1 <- create R1(r2s: <- {0: <- create R2()})
                // The second assignment should not lead to the resource being cleared,
                // it must be fully moved out of this container before,
                // not just assigned to the new variable
                if let r2 <- r1.r2s[0] <- nil {
                    let value = r2.value
                    destroy r1
                    destroy r2
                    return value
                }
                destroy r1
                return nil
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewStringValue("test"),
			),
			value,
		)
	})

	t.Run("reference, if-let statement, index expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R2 {
                let value: String

                init() {
                    self.value = "test"
                }
            }

            resource R1 {
                var r2s: @{Int: R2}

                init() {
                    self.r2s <- {}
                }

                destroy() {
                    destroy self.r2s
                }
            }

            fun createR1(): @R1 {
                return <- create R1()
            }

            fun test(r1: &R1): String? {
                r1.r2s[0] <-! create R2()
                // The second assignment should not lead to the resource being cleared,
                // it must be fully moved out of this container before,
                // not just assigned to the new variable
                if let r2 <- r1.r2s[0] <- nil {
                    let value = r2.value
                    destroy r2
                    return value
                }
                return nil
            }
        `)

		r1, err := inter.Invoke("createR1")
		require.NoError(t, err)

		r1 = r1.Transfer(inter, interpreter.ReturnEmptyLocationRange, atree.Address{1}, false, nil)

		r1Type := checker.RequireGlobalType(t, inter.Program.Elaboration, "R1")

		ref := &interpreter.EphemeralReferenceValue{
			Value:        r1,
			BorrowedType: r1Type,
		}

		value, err := inter.Invoke("test", ref)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewStringValue("test"),
			),
			value,
		)
	})
}

func TestCheckResourceInvalidationWithMove(t *testing.T) {

	t.Parallel()

	t.Run("in casting expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}

            fun test() {
                let r <- create R()
                let copy <- (<- r) as @R
                destroy r
                destroy copy
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("in reference expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}

            fun test() {
                let r <- create R()
                let ref = &(<- r) as &AnyResource
                destroy r
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("in conditional expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}

            fun test() {
                let r1 <- create R()
                let r2 <- create R()

                let r3 <- true ? <- r1 : <- r2
                destroy r3
                destroy r1
                destroy r2
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("in force expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}

            fun test() {
                let r <- create R()
                let copy <- (<- r)!
                destroy r
                destroy copy
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("in destroy expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}

            fun test() {
                let r <- create R()
	            destroy (<- r)
	            destroy r
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})


	t.Run("in function invocation expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}

            fun f(_ r: @R) {
                destroy r
            }

            fun test() {
                let r <- create R()
	            f(<- (<- r))
                destroy r
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})


	t.Run("in array expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}

            fun test() {
                let r <- create R()
                let rs <- [<- (<- r)]
                destroy r
                destroy rs
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})


	t.Run("in dictionary expression", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}

            fun test() {
                let r <- create R()
                let rs <- {"test": <- (<- r)}
                destroy r
                destroy rs
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})
}
