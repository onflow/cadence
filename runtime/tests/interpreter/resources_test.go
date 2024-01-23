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

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/sema"
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
	require.Equal(t, interpreter.TrueValue, result)
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
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("test"),
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

		r1 = r1.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address{1},
			false,
			nil,
			nil,
		)

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
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("test"),
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
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("test"),
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

		r1 = r1.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address{1},
			false,
			nil,
			nil,
		)

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
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("test"),
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
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("test"),
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

		r1 = r1.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address{1},
			false,
			nil,
			nil,
		)

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
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("test"),
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
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("test"),
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

		r1 = r1.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address{1},
			false,
			nil,
			nil,
		)

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
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("test"),
			),
			value,
		)
	})
}

func TestInterpretInvalidatedResourceValidation(t *testing.T) {

	t.Parallel()

	t.Run("after transfer", func(t *testing.T) {

		t.Run("transfer", func(t *testing.T) {

			t.Run("single", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun f(_ r: @R): Int {
                          let n = r.n
                          destroy r
                          return n
                      }

                      fun test(): Int {
                          let r <- create R(n: 1)
                          let r2 <- r
                          let n = f(<- r)
                          destroy r2
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 19, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("optional", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun f(_ r: @R?): Int {
                          let n = r?.n!
                          destroy r
                          return n
                      }

                      fun test(): Int {
                          let r: @R? <- create R(n: 1)
                          let r2 <- r
                          let n = f(<- r)
                          destroy r2
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 19, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("array", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun f(_ rs: @[R]): Int {
                          let n = rs[0].n
                          destroy rs
                          return n
                      }

                      fun test(): Int {
                          let rs <- [<- create R(n: 1)]
                          let rs2 <- rs
                          let n = f(<- rs)
                          destroy rs2
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 19, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("dictionary", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun f(_ rs: @{Int: R}): Int {
                          let n = rs[0]?.n!
                          destroy rs
                          return n
                      }

                      fun test(): Int {
                          let rs <- {0: <- create R(n: 1)}
                          let rs2 <- rs
                          let n = f(<- rs)
                          destroy rs2
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 19, invalidatedResourceErr.StartPosition().Line)
			})
		})

		t.Run("field read", func(t *testing.T) {

			t.Run("single", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test(): Int {
                          let r <- create R(n: 1)
                          let r2 <- r
                          let n = r.n
                          destroy r2
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("optional", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test(): Int {
                          let r: @R? <- create R(n: 1)
                          let r2 <- r
                          let n = r?.n!
                          destroy r2
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("array", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test(): Int {
                          let rs <- [<- create R(n: 1)]
                          let rs2 <- rs
                          let n = rs[0].n
                          destroy rs2
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("dictionary", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test(): Int {
                          let rs <- {0: <- create R(n: 1)}
                          let rs2 <- rs
                          let n = rs[0]?.n!
                          destroy rs2
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})
		})

		t.Run("field write", func(t *testing.T) {

			t.Run("single", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          var n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test() {
                          let r <- create R(n: 1)
                          let r2 <- r
                          r.n = 2
                          destroy r2
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			// TODO: optional (how?)

			t.Run("array", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          var n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test() {
                          let rs <- [<- create R(n: 1)]
                          let rs2 <- rs
                          rs[0].n = 2
                          destroy rs2
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			// TODO: dictionary (how?)
		})

		t.Run("destruction", func(t *testing.T) {

			t.Run("single", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {}

                      fun test() {
                          let r <- create R()
                          let r2 <- r
                          destroy r
                          destroy r2
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 7, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("optional", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {}

                      fun test() {
                          let r: @R? <- create R()
                          let r2 <- r
                          destroy r
                          destroy r2
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 7, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("array", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {}

                      fun test() {
                          let rs <- [<- create R()]
                          let rs2 <- rs
                          destroy rs
                          destroy rs2
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)
			})

			t.Run("dictionary", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {}

                      fun test() {
                          let rs <- {0: <- create R()}
                          let rs2 <- rs
                          destroy rs
                          destroy rs2
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 7, invalidatedResourceErr.StartPosition().Line)
			})
		})
	})

	t.Run("after destruction", func(t *testing.T) {

		t.Run("transfer", func(t *testing.T) {

			t.Run("single", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun f(_ r: @R): Int {
                          let n = r.n
                          destroy r
                          return n
                      }

                      fun test(): Int {
                          let r <- create R(n: 1)
                          destroy r
                          let n = f(<- r)
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 19, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("optional", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun f(_ r: @R?): Int {
                          let n = r?.n!
                          destroy r
                          return n
                      }

                      fun test(): Int {
                          let r: @R? <- create R(n: 1)
                          destroy r
                          let n = f(<- r)
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 19, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("array", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun f(_ rs: @[R]): Int {
                          let n = rs[0].n
                          destroy rs
                          return n
                      }

                      fun test(): Int {
                          let rs <- [<- create R(n: 1)]
                          destroy rs
                          let n = f(<- rs)
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 19, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("dictionary", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun f(_ rs: @{Int: R}): Int {
                          let n = rs[0]?.n!
                          destroy rs
                          return n
                      }

                      fun test(): Int {
                          let rs <- {0: <- create R(n: 1)}
                          destroy rs
                          let n = f(<- rs)
                          return n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 19, invalidatedResourceErr.StartPosition().Line)
			})
		})

		t.Run("field read", func(t *testing.T) {

			t.Run("single", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test(): Int {
                          let r <- create R(n: 1)
                          destroy r
                          return r.n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("optional", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test(): Int {
                          let r: @R? <- create R(n: 1)
                          destroy r
                          return r?.n!
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("array", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test(): Int {
                          let rs <- [<-create R(n: 1)]
                          destroy rs
                          return rs[0].n
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("dictionary", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          let n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test(): Int {
                          let rs <- {0: <-create R(n: 1)}
                          destroy rs
                          return rs[0]?.n!
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})
		})

		t.Run("field write", func(t *testing.T) {

			t.Run("single", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          var n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test() {
                          let r <- create R(n: 1)
                          destroy r
                          r.n = 2
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			// TODO: optional (how?)

			t.Run("array", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {
                          var n: Int

                          init(n: Int) {
                              self.n = n
                          }
                      }

                      fun test() {
                          let rs <- [<- create R(n: 1)]
                          destroy rs
                          rs[0].n = 2
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 13, invalidatedResourceErr.StartPosition().Line)
			})

			// TODO: dictionary (how?)

		})

		t.Run("destruction", func(t *testing.T) {

			t.Run("single", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {}

                      fun test() {
                          let r <- create R()
                          destroy r
                          destroy r
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 7, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("optional", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {}

                      fun test() {
                          let r: @R? <- create R()
                          destroy r
                          destroy r
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 7, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("array", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {}

                      fun test() {
                          let rs <- [<-create R()]
                          destroy rs
                          destroy rs
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 7, invalidatedResourceErr.StartPosition().Line)
			})

			t.Run("dictionary", func(t *testing.T) {

				t.Parallel()

				inter, err := parseCheckAndInterpretWithOptions(t,
					`
                      resource R {}

                      fun test() {
                          let rs <- {0: <-create R()}
                          destroy rs
                          destroy rs
                      }
                    `,
					ParseCheckAndInterpretOptions{
						HandleCheckerError: func(err error) {
							errs := checker.RequireCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				RequireError(t, err)

				var invalidatedResourceErr interpreter.InvalidatedResourceError
				require.ErrorAs(t, err, &invalidatedResourceErr)

				assert.Equal(t, 7, invalidatedResourceErr.StartPosition().Line)
			})
		})
	})

	t.Run("in casting expression", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r <- create R()
                let copy <- (<- r) as @R
                destroy r
                destroy copy
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})

	t.Run("in reference expression", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r <- create R()
                let ref = &(<- r) as &AnyResource
                destroy r
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 2)
					require.IsType(t, &sema.ResourceLossError{}, errs[0])
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[1])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})

	t.Run("in conditional expression", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r1 <- create R()
                let r2 <- create R()

                let r3 <- true ? <- r1 : <- r2
                destroy r3
                destroy r1
                destroy r2
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 6)
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[1])
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[2])
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[3])
					require.IsType(t, &sema.ResourceLossError{}, errs[4])
					require.IsType(t, &sema.ResourceLossError{}, errs[5])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})

	t.Run("in force expression", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r <- create R()
                let copy <- (<- r)!
                destroy r
                destroy copy
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})

	t.Run("in destroy expression", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r <- create R()
                destroy (<- r)
                destroy r
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})

	t.Run("in function invocation expression", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun f(_ r: @R) {
                destroy r
            }

            fun test() {
                let r <- create R()
                f(<- (<- r))
                destroy r
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})

	t.Run("in array expression", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r <- create R()
                let rs <- [<- (<- r)]
                destroy r
                destroy rs
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})

	t.Run("in dictionary expression", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r <- create R()
                let rs <- {"test": <- (<- r)}
                destroy r
                destroy rs
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})

	t.Run("with inner conditional expression", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r1 <- create R()
                let r2 <- create R()

                let r3 <- true ? r1 : r2
                destroy r3
                destroy r1
                destroy r2
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 2)
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[1])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})

	t.Run("force move", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r1 <- create R()
                let r2 <- create R()

                let r3 <-! true ? r1 : r2
                destroy r3
                destroy r1
                destroy r2
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 2)
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[1])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})
}

func TestCheckResourceInvalidationWithConditionalExprInDestroy(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
        resource R {}
        fun test() {
            let r1 <- create R()
            let r2 <- create R()

            destroy true? r1 : r2
            destroy r1
            destroy r2
        }`,
		ParseCheckAndInterpretOptions{
			HandleCheckerError: func(err error) {
				errs := checker.RequireCheckerErrors(t, err, 2)
				require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
				require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}

func TestInterpretResourceUseAfterInvalidation(t *testing.T) {

	t.Parallel()

	t.Run("field access", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {
                let s: String

                init() {
                    self.s = ""
                }
            }

            fun test() {
                let r <- create R()
                let copy <- (<- r)
                let str = r.s

                destroy r
                destroy copy
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 2)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[1])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		invalidatedResourceError := interpreter.InvalidatedResourceError{}
		require.ErrorAs(t, err, &invalidatedResourceError)

		// error must be thrown at field access
		assert.Equal(t, 13, invalidatedResourceError.StartPosition().Line)
	})

	t.Run("parameter", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
            resource R {}

            fun test() {
                let r <- create R()
                foo(<-r)
            }

            fun foo(_ r: @R) {
                let copy <- (<- r)
                destroy r
                destroy copy
            }`,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.RequireCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
	})
}

func TestInterpreterResourcePreCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource S {}

      struct interface Receiver {
          pub fun deposit(from: @S) {
              post {
                  from != nil: ""
              }
          }
      }

      struct Vault: Receiver {
          pub fun deposit(from: @S) {
              destroy from
          }
      }

      fun test() {
          Vault().deposit(from: <-create S())
      }
	`)

	_, err := inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpreterResourcePostCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource S {}

      struct interface Receiver {
          pub fun deposit(from: @S) {
              post {
                  from != nil: ""
              }
          }
      }

      struct Vault: Receiver {
          pub fun deposit(from: @S) {
              destroy from
          }
      }

      fun test() {
          Vault().deposit(from: <-create S())
      }
	`)

	_, err := inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpreterResourcePreAndPostCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource S {}

      struct interface Receiver {
          pub fun deposit(from: @S) {
              pre {
                  from != nil: ""
              }
              post {
                  from != nil: ""
              }
          }
      }

      struct Vault: Receiver {
          pub fun deposit(from: @S) {
              pre {
                  from != nil: ""
              }
              post {
                  1 > 0: ""
              }
              destroy from
          }
      }

      fun test() {
          Vault().deposit(from: <-create S())
      }
	`)

	_, err := inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpreterResourceConditionAdditionalParam(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource S {}

      struct interface Receiver {
          pub fun deposit(from: @S, other: UInt64) {
              pre {
                  from != nil: ""
              }
              post {
                  other > 0: ""
              }
          }
      }

      struct Vault: Receiver {
          pub fun deposit(from: @S, other: UInt64) {
              pre {
                  from != nil: ""
              }
              post {
                  other > 0: ""
              }
              destroy from
          }
      }

      fun test() {
          Vault().deposit(from: <-create S(), other: 42)
      }
	`)

	_, err := inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpreterResourceDoubleWrappedCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource S {}

      struct interface A {
          pub fun deposit(from: @S) {
              pre {
                  from != nil: ""
              }
              post {
                  from != nil: ""
              }
          }
      }

      struct interface B {
          pub fun deposit(from: @S) {
              pre {
                  from != nil: ""
              }
              post {
                  from != nil: ""
              }
          }
      }

      struct Vault: A, B {
          pub fun deposit(from: @S) {
              pre {
                  from != nil: ""
              }
              post {
                  1 > 0: ""
              }
              destroy from
          }
      }

      fun test() {
          Vault().deposit(from: <-create S())
      }
	`)

	_, err := inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpretOptionalResourceReference(t *testing.T) {

	t.Parallel()

	address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

	inter, _ := testAccount(
		t,
		address,
		true,
		`
          resource R {
              pub let id: Int

              init() {
                  self.id = 1
              }
          }

          fun test() {
              account.save(<-{0 : <-create R()}, to: /storage/x)
              let collection = account.borrow<&{Int: R}>(from: /storage/x)!

              let resourceRef = (&collection[0] as &R?)!
              let token <- collection.remove(key: 0)

              let x = resourceRef.id
              destroy token
          }
        `,
		sema.Config{},
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpretArrayOptionalResourceReference(t *testing.T) {

	t.Parallel()

	address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

	inter, _ := testAccount(
		t,
		address,
		true,
		`
          resource R {
              pub let id: Int

              init() {
                  self.id = 1
              }
          }

          fun test() {
              account.save(<-[<-create R()], to: /storage/x)
              let collection = account.borrow<&[R?]>(from: /storage/x)!

              let resourceRef = (&collection[0] as &R?)!
              let token <- collection.remove(at: 0)

              let x = resourceRef.id
              destroy token
          }
        `,
		sema.Config{},
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpretReferenceUseAfterTransferAndDestruction(t *testing.T) {

	t.Parallel()

	const resourceCode = `
	  resource R {
          var value: Int

          init() {
              self.value = 0
          }

          fun increment() {
              self.value = self.value + 1
          }
      }
	`

	t.Run("composite", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, resourceCode+`

          fun test(): Int {

              let resources <- {
                  "r": <-create R()
              }

              let ref = &resources["r"] as &R?
              let r <-resources.remove(key: "r")
	          destroy r
              destroy resources

              ref!.increment()
              return ref!.value
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var invalidatedResourceErr interpreter.DestroyedResourceError
		require.ErrorAs(t, err, &invalidatedResourceErr)

		assert.Equal(t, 26, invalidatedResourceErr.StartPosition().Line)
	})

	t.Run("dictionary", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, resourceCode+`

          fun test(): Int {

              let resources <- {
                  "nested": <-{"r": <-create R()}
              }

              let ref = &resources["nested"] as &{String: R}?
              let nested <-resources.remove(key: "nested")
	          destroy nested
              destroy resources

              return ref!.length
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var invalidatedResourceErr interpreter.DestroyedResourceError
		require.ErrorAs(t, err, &invalidatedResourceErr)

		assert.Equal(t, 26, invalidatedResourceErr.StartPosition().Line)
	})

	t.Run("array", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, resourceCode+`

          fun test(): Int {

              let resources <- {
                  "nested": <-[<-create R()]
              }

              let ref = &resources["nested"] as &[R]?
              let nested <-resources.remove(key: "nested")
	          destroy nested
              destroy resources

              return ref!.length
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var invalidatedResourceErr interpreter.DestroyedResourceError
		require.ErrorAs(t, err, &invalidatedResourceErr)

		assert.Equal(t, 26, invalidatedResourceErr.StartPosition().Line)
	})

	t.Run("optional", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, resourceCode+`

          fun test(): Int {

              let resources: @[R?] <- [<-create R()]

              let ref = &resources[0] as &R?
              let r <-resources.remove(at: 0)
		      destroy r
              destroy resources

              ref!.increment()
              return ref!.value
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var invalidatedResourceErr interpreter.DestroyedResourceError
		require.ErrorAs(t, err, &invalidatedResourceErr)

		assert.Equal(t, 24, invalidatedResourceErr.StartPosition().Line)
	})
}

func TestInterpretResourceDestroyedInPreCondition(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(
		t,
		`
            resource interface I {
                 pub fun receiveResource(_ r: @Bar) {
                    pre {
                        destroyResource(<-r)
                    }
                }
            }

            fun destroyResource(_ r: @Bar): Bool {
                destroy r
                return true
            }

            resource Foo: I {
                 pub fun receiveResource(_ r: @Bar) {
                    destroy r
                }
            }

            resource Bar  {}

            fun test() {
                let foo <- create Foo()
                let bar <- create Bar()

                foo.receiveResource(<- bar)
                destroy foo
            }
        `,
		ParseCheckAndInterpretOptions{
			HandleCheckerError: func(err error) {
				errs := checker.RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.InvalidInterfaceConditionResourceInvalidationError{}, errs[0])
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}

func TestInterpretInvalidReentrantResourceDestruction(t *testing.T) {

	t.Parallel()

	t.Run("composite", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `

            resource Inner {
                let outer: &Outer

                init(outer: &Outer) {
                    self.outer = outer
                }

                destroy() {
                    self.outer.reenter()
                }
            }

            resource Outer {
                var inner: @Inner?

                init() {
                    self.inner <-! create Inner(outer: &self as &Outer)
                }

                fun reenter() {
                    let inner <- self.inner <- nil
                    destroy inner
                }

                destroy() {
                    destroy self.inner
                }
            }

            fun test() {
                let outer <- create Outer()

                destroy outer
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var destroyedResourceErr interpreter.DestroyedResourceError
		require.ErrorAs(t, err, &destroyedResourceErr)
	})

	t.Run("array", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `

            resource Inner {
                let outer: &Outer

                init(outer: &Outer) {
                    self.outer = outer
                }

                destroy() {
                    self.outer.reenter()
                }
            }

            resource Outer {
                var inner: @[Inner]

                init() {
                    self.inner <- [<-create Inner(outer: &self as &Outer)]
                }

                fun reenter() {
                    let inner <- self.inner <- []
                    destroy inner
                }

                destroy() {
                    destroy self.inner
                }
            }

            fun test() {
                let outer <- create Outer()

                destroy outer
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var destroyedResourceErr interpreter.DestroyedResourceError
		require.ErrorAs(t, err, &destroyedResourceErr)
	})

	t.Run("dictionary", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `

            resource Inner {
                let outer: &Outer

                init(outer: &Outer) {
                    self.outer = outer
                }

                destroy() {
                    self.outer.reenter()
                }
            }

            resource Outer {
                var inner: @{Int: Inner}

                init() {
                    self.inner <- {0: <-create Inner(outer: &self as &Outer)}
                }

                fun reenter() {
                    let inner <- self.inner <- {}
                    destroy inner
                }

                destroy() {
                    destroy self.inner
                }
            }

            fun test() {
                let outer <- create Outer()

                destroy outer
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var destroyedResourceErr interpreter.DestroyedResourceError
		require.ErrorAs(t, err, &destroyedResourceErr)
	})
}

func TestInterpretResourceFunctionInvocationAfterDestruction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        pub resource Vault {
            pub fun foo(_ ignored: Bool) {}
        }

        pub resource Attacker {
			pub var vault: @Vault

			init() {
				self.vault <- create Vault()
			}

			pub fun shenanigans(): Bool {
				var temp <- create Vault()
				self.vault <-> temp
                destroy temp
                return true
			}

			destroy() {
				destroy self.vault
			}
		}

        pub fun main() {
            let a <- create Attacker()
            a.vault.foo(a.shenanigans())
            destroy a
        }
    `)

	_, err := inter.Invoke("main")
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})
}

func TestInterpretResourceFunctionReferenceValidity(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        pub resource Vault {
            pub fun foo(_ ref: &Vault): &Vault {
                return ref
            }
        }

        pub resource Attacker {
            pub var vault: @Vault

            init() {
                self.vault <- create Vault()
            }

            pub fun shenanigans1(): &Vault {
                // Create a reference in a nested call
                return &self.vault as &Vault
            }

            pub fun shenanigans2(_ ref: &Vault): &Vault {
                return ref
            }

            destroy() {
                destroy self.vault
            }
        }

        pub fun main() {
            let a <- create Attacker()

            // A reference to receiver get created inside the nested call 'shenanigans1()'.
            // Same reference is returned eventually.
            var vaultRef1 = a.vault.foo(a.shenanigans1())
            // Reference must be still valid, even after the invalidation of the bound function receiver.
            vaultRef1.foo(vaultRef1)

            // A reference to receiver is explicitly created as a parameter.
            // Same reference is returned eventually.
            var vaultRef2 = a.vault.foo(a.shenanigans2(&a.vault as &Vault))
            // Reference must be still valid, even after the invalidation of the bound function receiver.
            vaultRef2.foo(vaultRef2)

            destroy a
        }
    `)

	_, err := inter.Invoke("main")
	require.NoError(t, err)
}

func TestInterpretResourceFunctionResourceFunctionValidity(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        pub resource Vault {
            pub fun foo(_ dummy: Bool): Bool {
                return dummy
            }
        }

        pub resource Attacker {
            pub var vault: @Vault

            init() {
                self.vault <- create Vault()
            }

            pub fun shenanigans(_ n: Int): Bool {
                if n > 0 {
                    return self.vault.foo(self.shenanigans(n - 1))
                }
                return true
            }

            destroy() {
                destroy self.vault
            }
        }

        pub fun main() {
            let a <- create Attacker()

            a.vault.foo(a.shenanigans(10))

            destroy a
        }
    `)

	_, err := inter.Invoke("main")
	require.NoError(t, err)
}

func TestInterpretInnerResourceDestruction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        pub resource InnerResource {
            pub var name: String
            pub(set) var parent: &OuterResource?

            init(_ name: String) {
                self.name = name
                self.parent = nil
            }

            destroy() {
                self.parent!.shenanigans()
            }
        }

        pub resource OuterResource {
            pub var inner1: @InnerResource
            pub var inner2: @InnerResource

            init() {
                self.inner1 <- create InnerResource("inner1")
                self.inner2 <- create InnerResource("inner2")

                self.inner1.parent = &self as &OuterResource
                self.inner2.parent = &self as &OuterResource
            }

            pub fun shenanigans() {
                self.inner1 <-> self.inner2
            }

            destroy() {
                destroy self.inner1
                destroy self.inner2
            }
        }

        pub fun main() {
            let a <- create OuterResource()
            destroy a
        }`,
	)

	_, err := inter.Invoke("main")
	RequireError(t, err)

	var destroyedResourceErr interpreter.DestroyedResourceError
	require.ErrorAs(t, err, &destroyedResourceErr)
}

func TestInterpretInnerResourceMove(t *testing.T) {

	t.Parallel()

	t.Run("assignment", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            pub resource OuterResource {
                pub var a: @InnerResource
                pub var b: @InnerResource

                init() {
                     self.a <- create InnerResource()
                     self.b <- create InnerResource()
                }

                pub fun swap() {
                    self.a <-> self.b
                }

                destroy() {
                    // Nested resource is moved here once
                    var a <- self.a

                    // Nested resource is again moved here. This one should fail.
                    self.swap()

                    destroy a
                    destroy self.b
                }
            }

            pub resource InnerResource {}

            pub fun main() {
                let a <- create OuterResource()
                destroy a
            }`,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.UseBeforeInitializationError{})
	})

	t.Run("second transfer", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            pub resource OuterResource {
                pub var a: @InnerResource
                pub var b: @InnerResource

                init() {
                     self.a <- create InnerResource()
                     self.b <- create InnerResource()
                }

                pub fun swap() {
                    self.a <-> self.b
                }

                destroy() {
                    var a <- create InnerResource()

                    // Nested resource is moved here once
                    var temp <- a <- self.a

                    // Nested resource is again moved here. This one should fail.
                    self.swap()

                    destroy a
                    destroy temp
                    destroy self.b
                }
            }

            pub resource InnerResource {}

            pub fun main() {
                let a <- create OuterResource()
                destroy a
            }`,
		)

		_, err := inter.Invoke("main")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.UseBeforeInitializationError{})
	})
}

func TestInterpretSetMemberResourceLoss(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
    access(all) resource R {
        access(all) let id: String
        init(id: String) {
            self.id = id
        }
    }
    access(all) fun putBack(_ rl: &ResourceLoser, _ r: @R) {
        rl.inner <-! r
    }
    access(all) resource ResourceLoser {
        pub(set) var inner: @R?;
        init(toLose: @R) {
            self.inner <- toLose
        }
        destroy() {
            var ref = &self as &ResourceLoser;
            // Let's move self.inner out
            var a <- self.inner
            // And force-write it back
            putBack(ref, <- a!)
        }
    }
    access(all) fun main(): Void {
       var resource <- create R(id: "abc");
       var rl <- create ResourceLoser(toLose: <- resource);
       destroy rl
    }`,
	)

	_, err := inter.Invoke("main")
	RequireError(t, err)
	require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})
}

func TestInterpretValueTransferResourceLoss(t *testing.T) {

	t.Parallel()

	inter, getLogs, _ := parseCheckAndInterpretWithLogs(t, `
    access(all) resource R {
        access(all) let id: String
        init(_ id: String) {
            log("Creating ".concat(id))
            self.id = id
        }
        destroy() {
            log("Destroying ".concat(self.id))
        }
    }

    access(all) struct IndexSwitcher {
        access(self) var counter: Int
        init() {
            self.counter = 0
        }
        access(all) fun callback(): Int {
            self.counter = self.counter + 1
            if self.counter == 1 {
                // Which key we want to be read?
                // Let's point it to a non-existent key
                return 123
            } else {
                // Which key we want to be assigned to?
                // We point it to 0 to overwrite the victim value
                return 0
            }
        }
    }

    access(all) fun loseResource(victim: @R) {
       var in <- create R("dummy resource")
       var dict: @{Int: R} <- { 0: <- victim }
       var indexSwitcher = IndexSwitcher()
       
       // this callback should only be evaluated once, rather than twice
       var out <- dict[indexSwitcher.callback()] <- in
       destroy out
       destroy dict
    }

    access(all) fun main(): Void {
       var victim <- create R("victim resource")
       loseResource(victim: <- victim)
    }
    `,
	)

	_, err := inter.Invoke("main")
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			`"Creating victim resource"`,
			`"Creating dummy resource"`,
			`"Destroying dummy resource"`,
			`"Destroying victim resource"`,
		},
		getLogs(),
	)

}

func TestInterpretVariableDeclarationEvaluationOrder(t *testing.T) {

	t.Parallel()

	inter, getLogs, err := parseCheckAndInterpretWithLogs(t, `
      // Necessary helper interface,
      // as AnyResource does not provide a uuid field,
      // and AnyResource must be used in collect
      // to avoid the potential type confusion to be rejected
      // by the defensive argument/parameter type check

      resource interface HasID {
          fun getID(): UInt64
      }

      resource Foo: HasID {
          fun getID(): UInt64 {
              return self.uuid
          }
      }

      resource Bar: HasID {
          fun getID(): UInt64 {
              return self.uuid
          }
      }

      fun collect(_ collected: @{HasID}): @Foo {
          log("collected")
          log(collected.getID())
          log(collected.getType())

          destroy <- collected

          return <- create Foo()
      }

      fun main() {
          var foo <- create Foo()
          log("foo")
          log(foo.uuid)

          var bar <- create Bar()
          log("bar")
          log(bar.uuid)

          if (true) {
              // Check that the RHS is evaluated *before* the variable is declared
              var bar <- foo <- collect(<-bar)

              destroy foo
              destroy bar // new bar
          } else {
              destroy foo
              destroy bar // original bar
          }
      }
    `)
	require.NoError(t, err)

	_, err = inter.Invoke("main")
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			`"foo"`,
			`1`,
			`"bar"`,
			`2`,
			`"collected"`,
			`2`,
			"Type<S.test.Bar>()",
		},
		getLogs(),
	)
}

func TestInterpretIfLetElseBranchConfusion(t *testing.T) {

	t.Parallel()

	inter, _, err := parseCheckAndInterpretWithLogs(t, `
        pub resource Victim{}
        pub fun main() {
            var r: @Victim? <- nil
            var r2: @Victim?  <- create Victim()
            if let dummy <- r <- r2 {
                // unreachable token destroys to please checker
                destroy dummy
                destroy r
            } else {
                // Error: r2 is invalid here

                var ref = &r as &Victim?
                var arr: @[Victim?]<- [<- r, <- r2]
                destroy arr
            }
        }
    `)
	require.NoError(t, err)

	_, err = inter.Invoke("main")
	RequireError(t, err)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}

func TestInterpretMovedResourceInOptionalBinding(t *testing.T) {

	t.Parallel()

	inter, _, err := parseCheckAndInterpretWithLogs(t, `
        access(all) resource R{}

        access(all) fun collect(copy2: @R?, _ arrRef: &[R]): @R {
            arrRef.append(<- copy2!)
            return <- create R()
        }

        access(all) fun main() {
            var victim: @R? <- create R()
            var arr: @[R] <- []

            // In the optional binding below, the 'victim' must be invalidated
            // before evaluation of the collect() call
            if let copy1 <- victim <- collect(copy2: <- victim, &arr as &[R]) {
                arr.append(<- copy1)
            } else {
                destroy victim // Never executed
            }

            destroy arr // This crashes
        }
    `)
	require.NoError(t, err)

	_, err = inter.Invoke("main")
	RequireError(t, err)
	invalidResourceError := &interpreter.InvalidatedResourceError{}
	require.ErrorAs(t, err, invalidResourceError)

	// Error must be thrown at `copy2: <- victim`
	errorStartPos := invalidResourceError.LocationRange.StartPosition()
	assert.Equal(t, 15, errorStartPos.Line)
	assert.Equal(t, 56, errorStartPos.Column)
}

func TestInterpretMovedResourceInSecondValue(t *testing.T) {

	t.Parallel()

	inter, _, err := parseCheckAndInterpretWithLogs(t, `
        access(all) resource R{}

        access(all) fun collect(copy2: @R?, _ arrRef: &[R]): @R {
            arrRef.append(<- copy2!)
            return <- create R()
        }

        access(all) fun main() {
            var victim: @R? <- create R()
            var arr: @[R] <- []

            // In the optional binding below, the 'victim' must be invalidated
            // before evaluation of the collect() call
            let copy1 <- victim <- collect(copy2: <- victim, &arr as &[R])

            destroy copy1
            destroy arr
        }
    `)
	require.NoError(t, err)

	_, err = inter.Invoke("main")
	RequireError(t, err)
	invalidResourceError := &interpreter.InvalidatedResourceError{}
	require.ErrorAs(t, err, invalidResourceError)

	// Error must be thrown at `copy2: <- victim`
	errorStartPos := invalidResourceError.LocationRange.StartPosition()
	assert.Equal(t, 15, errorStartPos.Line)
	assert.Equal(t, 53, errorStartPos.Column)
}

func TestInterpretOptionalBindingElseBranch(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
		access(all) resource Victim {}

		access(all) fun main() {

			var r: @Victim? <- nil
			var r2: @Victim?  <- create Victim()

			if let dummy <- r <- r2 {
				// unreachable token destroys to please checker
				destroy dummy
				destroy r
			} else {
				// checker failed to notice that r2 is invalid here
				var ref = &r as &Victim?
				var arr: @[Victim?]<- [
                    <- r,
                    <- r2
                ]
				destroy arr
			}
		}
   `)

	_, err := inter.Invoke("main")
	RequireError(t, err)

	var invalidatedResourceErr interpreter.InvalidatedResourceError
	require.ErrorAs(t, err, &invalidatedResourceErr)

	// Error must be thrown at `<-r2` in the array literal
	assert.Equal(t, 18, invalidatedResourceErr.StartPosition().Line)
}
