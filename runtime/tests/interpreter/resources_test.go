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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
							errs := checker.ExpectCheckerErrors(t, err, 1)
							require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
						},
					},
				)
				require.NoError(t, err)

				_, err = inter.Invoke("test")
				require.Error(t, err)

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
					errs := checker.ExpectCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 4)
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[1])
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[2])
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[3])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 2)
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[1])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 2)
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
					require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[1])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
				errs := checker.ExpectCheckerErrors(t, err, 2)
				require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
				require.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.Error(t, err)
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
					errs := checker.ExpectCheckerErrors(t, err, 2)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[1])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)

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
					errs := checker.ExpectCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
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
		require.Error(t, err)

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
		require.Error(t, err)

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
		require.Error(t, err)

		var invalidatedResourceErr interpreter.DestroyedResourceError
		require.ErrorAs(t, err, &invalidatedResourceErr)

		assert.Equal(t, 24, invalidatedResourceErr.StartPosition().Line)
	})
}

func TestInterpretResourceDestroyedInPreCondition(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
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
        }`,

		ParseCheckAndInterpretOptions{},
	)

	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.Error(t, err)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}

func TestInterpretResourceReferenceInvalidationOnMove(t *testing.T) {

	t.Parallel()

	t.Run("stack to account", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(
			t,
			address,
			true,
			`
            resource R {
                pub(set) var id: Int

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
                ref.id = 2
            }`,
		)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		require.ErrorAs(t, err, &interpreter.MovedResourceReferenceError{})
	})

	t.Run("stack to account readonly", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(
			t,
			address,
			true,
			`
            resource R {
                pub(set) var id: Int

                init() {
                    self.id = 1
                }
            }

            fun test() {
                let r <-create R()
                let ref = &r as &R

                // Move the resource into the account
                account.save(<-r, to: /storage/r)

                // 'Read'' a field from the reference
                let id = ref.id
            }`,
		)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		require.ErrorAs(t, err, &interpreter.MovedResourceReferenceError{})
	})

	t.Run("account to stack", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                pub(set) var id: Int

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
                ref.id = 2

                destroy movedR
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

		arrayRef := interpreter.NewUnmeteredEphemeralReferenceValue(
			false,
			array,
			&sema.VariableSizedType{
				Type: rType,
			},
		)

		_, err := inter.Invoke("test", arrayRef)
		require.Error(t, err)
		require.ErrorAs(t, err, &interpreter.MovedResourceReferenceError{})
	})

	t.Run("stack to stack", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(
			t,
			address,
			true,
			`
            resource R {
                pub(set) var id: Int

                init() {
                    self.id = 1
                }
            }

            fun test() {
                let r1 <-create R()
                let ref = &r1 as &R

                // Move the resource into the account
                let r2 <- r1

                // Update the reference
                ref.id = 2

                destroy r2
            }`,
		)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("one account to another account", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                pub(set) var id: Int

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
                ref.id = 2
            }
        `)

		rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)

		// Resource array in account 0x01

		array1 := interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			common.Address{0x1},
		)

		arrayRef1 := interpreter.NewUnmeteredEphemeralReferenceValue(
			false,
			array1,
			&sema.VariableSizedType{
				Type: rType,
			},
		)

		// Resource array in account 0x02

		array2 := interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.ConvertSemaToStaticType(nil, rType),
			},
			common.Address{0x2},
		)

		arrayRef2 := interpreter.NewUnmeteredEphemeralReferenceValue(
			false,
			array2,
			&sema.VariableSizedType{
				Type: rType,
			},
		)

		_, err := inter.Invoke("test", arrayRef1, arrayRef2)
		require.Error(t, err)
		require.ErrorAs(t, err, &interpreter.MovedResourceReferenceError{})
	})

	t.Run("account to stack to same account", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                pub(set) var id: Int

                init() {
                    self.id = 1
                }
            }

            fun test(target: &[R]): Int {
                target.append(<- create R())

                // Take reference while in the account
                let ref = &target[0] as &R

                // Move the resource out of the account onto the stack
                let movedR <- target.remove(at: 0)

                // Append an extra resource just to force an index change
                target.append(<- create R())

                // Move the resource back into the account (now at a different index)
                target.append(<- movedR)

                // Update the reference
                ref.id = 2

                return target[1].id
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

		arrayRef := interpreter.NewUnmeteredEphemeralReferenceValue(
			false,
			array,
			&sema.VariableSizedType{
				Type: rType,
			},
		)

		val, err := inter.Invoke("test", arrayRef)
		require.NoError(t, err)
		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(2), val)
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
                pub(set) var id: Int

                init() {
                    self.id = 1
                }
            }

             fun test() {
                let r1 <-create R()
                account.save(<-r1, to: /storage/r)

                let r1Ref = account.borrow<&R>(from: /storage/r)!

                let r2 <- account.load<@R>(from: /storage/r)!

                r1Ref.id = 2
                destroy r2
            }`,
		)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		require.ErrorAs(t, err, &interpreter.DereferenceError{})
	})
}
