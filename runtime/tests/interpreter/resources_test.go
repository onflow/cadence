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

func TestInterpretResourceInvalidationWithConditionalExprInDestroy(t *testing.T) {

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
          access(all) fun deposit(from: @S) {
              pre {
                  from != nil: ""
              }
          }
      }

      struct Vault: Receiver {
          access(all) fun deposit(from: @S) {
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
          access(all) fun deposit(from: @S) {
              post {
                  from != nil: ""  // This is an error. Resource is destroyed at this point
              }
          }
      }

      struct Vault: Receiver {
          access(all) fun deposit(from: @S) {
              destroy from
          }
      }

      fun test() {
          Vault().deposit(from: <-create S())
      }
	`)

	_, err := inter.Invoke("test")
	RequireError(t, err)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}

func TestInterpreterResourcePreAndPostCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource S {}

      struct interface Receiver {
          access(all) fun deposit(from: @S) {
              pre {
                  from != nil: ""  // This is OK
              }
              post {
                  from != nil: ""  // This is an error: Resource is destroyed at this point
              }
          }
      }

      struct Vault: Receiver {
          access(all) fun deposit(from: @S) {
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
	RequireError(t, err)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}

func TestInterpreterResourceConditionAdditionalParam(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource S {}

      struct interface Receiver {
          access(all) fun deposit(from: @S, other: UInt64) {
              pre {
                  from != nil: ""
              }
              post {
                  other > 0: ""
              }
          }
      }

      struct Vault: Receiver {
          access(all) fun deposit(from: @S, other: UInt64) {
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

func TestInterpreterResourceDoubleWrappedPreCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource S {}

      struct interface A {
          access(all) fun deposit(from: @S) {
              pre {
                  from != nil: ""
              }
          }
      }

      struct interface B {
          access(all) fun deposit(from: @S) {
              pre {
                  from != nil: ""
              }
          }
      }

      struct Vault: A, B {
          access(all) fun deposit(from: @S) {
              pre {
                  from != nil: ""
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

func TestInterpreterResourceDoubleWrappedPostCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource S {}

      struct interface A {
          access(all) fun deposit(from: @S) {
              post {
                  from != nil: ""
              }
          }
      }

      struct interface B {
          access(all) fun deposit(from: @S) {
              post {
                  from != nil: ""
              }
          }
      }

      struct Vault: A, B {
          access(all) fun deposit(from: @S) {
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
	RequireError(t, err)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}

func TestInterpretOptionalResourceReference(t *testing.T) {

	t.Parallel()

	address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

	inter, _ := testAccount(t, address, true, nil, `
          resource R {
              access(all) let id: Int

              init() {
                  self.id = 1
              }
          }

          fun test() {
              account.storage.save(<-{0 : <-create R()}, to: /storage/x)
              let collection = account.storage.borrow<auth(Remove) &{Int: R}>(from: /storage/x)!

              let resourceRef = collection[0]!
              let token <- collection.remove(key: 0)

              let x = resourceRef.id
              destroy token
          }
        `, sema.Config{})

	_, err := inter.Invoke("test")
	require.Error(t, err)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
}

func TestInterpretArrayOptionalResourceReference(t *testing.T) {

	t.Parallel()

	address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

	inter, _ := testAccount(t, address, true, nil, `
          resource R {
              access(all) let id: Int

              init() {
                  self.id = 1
              }
          }

          fun test() {
              account.storage.save(<-[<-create R()], to: /storage/x)
              let collection = account.storage.borrow<auth(Remove) &[R?]>(from: /storage/x)!

              let resourceRef = collection[0]!
              let token <- collection.remove(at: 0)

              let x = resourceRef.id
              destroy token
          }
        `, sema.Config{})

	_, err := inter.Invoke("test")
	require.Error(t, err)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
}

func TestInterpretResourceDestroyedInPreCondition(t *testing.T) {
	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(
		t,
		`
            resource interface I {
                 access(all) fun receiveResource(_ r: @Bar) {
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
                 access(all) fun receiveResource(_ r: @Bar) {
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
				errs := checker.RequireCheckerErrors(t, err, 2)
				require.IsType(t, &sema.PurityError{}, errs[0])
				require.IsType(t, &sema.InvalidInterfaceConditionResourceInvalidationError{}, errs[1])
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}

func TestInterpretResourceFunctionReferenceValidity(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        access(all) resource Vault {
            access(all) fun foo(_ ref: &Vault): &Vault {
                return ref
            }
        }

        access(all) resource Attacker {
            access(all) var vault: @Vault

            init() {
                self.vault <- create Vault()
            }

            access(all) fun shenanigans1(): &Vault {
                // Create a reference in a nested call
                return &self.vault as &Vault
            }

            access(all) fun shenanigans2(_ ref: &Vault): &Vault {
                return ref
            }
        }

        access(all) fun main() {
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
        access(all) resource Vault {
            access(all) fun foo(_ dummy: Bool): Bool {
                return dummy
            }
        }

        access(all) resource Attacker {
            access(all) var vault: @Vault

            init() {
                self.vault <- create Vault()
            }

            access(all) fun shenanigans(_ n: Int): Bool {
                if n > 0 {
                    return self.vault.foo(self.shenanigans(n - 1))
                }
                return true
            }
        }

        access(all) fun main() {
            let a <- create Attacker()

            a.vault.foo(a.shenanigans(10))

            destroy a
        }
    `)

	_, err := inter.Invoke("main")
	require.NoError(t, err)
}

func TestInterpretImplicitDestruction(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `

            resource R {}

            fun test() {
                let r <- create R()
                destroy r
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})
}

func TestInterpretResourceInterfaceDefaultDestroyEvent(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
		resource interface I {
			access(all) let id: Int
			event ResourceDestroyed(id: Int = self.id)
		}

		resource A: I {
			access(all) let id: Int

			init(id: Int) {
				self.id = id
			}

			event ResourceDestroyed(id: Int = self.id)
		} 

		resource B: I {
			access(all) let id: Int

			init(id: Int) {
				self.id = id
			}

			event ResourceDestroyed(id: Int = self.id)
		} 

		fun test() {
			let a <- create A(id: 1)	
			let b <- create B(id: 2)	
			let is: @[AnyResource] <- [<-a, <-b]
			destroy is
		}
        `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(inter *interpreter.Interpreter, locationRange interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})

	require.NoError(t, err)
	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 4)
	require.Equal(t, "I.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[0].GetField(inter, interpreter.EmptyLocationRange, "id"))
	require.Equal(t, "A.ResourceDestroyed", events[1].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[1].GetField(inter, interpreter.EmptyLocationRange, "id"))
	require.Equal(t, "I.ResourceDestroyed", events[2].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 2), events[2].GetField(inter, interpreter.EmptyLocationRange, "id"))
	require.Equal(t, "B.ResourceDestroyed", events[3].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 2), events[3].GetField(inter, interpreter.EmptyLocationRange, "id"))
}

func TestInterpretResourceInterfaceDefaultDestroyEventMultipleInheritance(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
		resource interface I {
			access(all) let id: Int
			event ResourceDestroyed(id: Int = self.id)
		}

		resource interface J {
			access(all) let id: Int
			event ResourceDestroyed(id: Int = self.id)
		}

		resource A: I, J {
			access(all) let id: Int

			init(id: Int) {
				self.id = id
			}

			event ResourceDestroyed(id: Int = self.id)
		} 

		fun test() {
			let a <- create A(id: 1)	
			destroy a
		}
        `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(inter *interpreter.Interpreter, locationRange interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})

	require.NoError(t, err)
	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 3)
	require.Equal(t, "I.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[0].GetField(inter, interpreter.EmptyLocationRange, "id"))
	require.Equal(t, "J.ResourceDestroyed", events[1].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[1].GetField(inter, interpreter.EmptyLocationRange, "id"))
	require.Equal(t, "A.ResourceDestroyed", events[2].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[2].GetField(inter, interpreter.EmptyLocationRange, "id"))
}

func TestInterpretResourceInterfaceDefaultDestroyEventIndirectInheritance(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
		resource interface I {
			access(all) let id: Int
			event ResourceDestroyed(id: Int = self.id)
		}

		resource interface J: I {
			access(all) let id: Int
			event ResourceDestroyed(id: Int = self.id)
		}

		resource A: J {
			access(all) let id: Int

			init(id: Int) {
				self.id = id
			}

			event ResourceDestroyed(id: Int = self.id)
		} 

		fun test() {
			let a <- create A(id: 1)	
			destroy a
		}
        `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(inter *interpreter.Interpreter, locationRange interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})

	require.NoError(t, err)
	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 3)
	require.Equal(t, "J.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[0].GetField(inter, interpreter.EmptyLocationRange, "id"))
	require.Equal(t, "I.ResourceDestroyed", events[1].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[1].GetField(inter, interpreter.EmptyLocationRange, "id"))
	require.Equal(t, "A.ResourceDestroyed", events[2].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[2].GetField(inter, interpreter.EmptyLocationRange, "id"))
}

func TestInterpretResourceInterfaceDefaultDestroyEventNoCompositeEvent(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
		resource interface I {
			access(all) let id: Int
			event ResourceDestroyed(id: Int = self.id)
		}

		resource interface J: I {
			access(all) let id: Int
		}

		resource A: J {
			access(all) let id: Int

			init(id: Int) {
				self.id = id
			}
		} 

		fun test() {
			let a <- create A(id: 1)	
			destroy a
		}
        `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(inter *interpreter.Interpreter, locationRange interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})

	require.NoError(t, err)
	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 1)
	require.Equal(t, "I.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[0].GetField(inter, interpreter.EmptyLocationRange, "id"))
}

func TestInterpretDefaultDestroyEventArgumentScoping(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
		let x = 1

		resource R {
			event ResourceDestroyed(x: Int = x)
		}
		
		fun test() {
			let x = 2
			let r <- create R()
			// should emit R.ResourceDestroyed(x: 1), not R.ResourceDestroyed(x: 2)
			destroy r
		}
        `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(inter *interpreter.Interpreter, locationRange interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
		HandleCheckerError: func(err error) {
			errs := checker.RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.DefaultDestroyInvalidArgumentError{}, errs[0])
			// ...
		},
	})

	require.NoError(t, err)
	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 1)
	require.Equal(t, "R.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[0].GetField(inter, interpreter.EmptyLocationRange, "x"))
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

func TestInterpretMovedResourceInOptionalBinding(t *testing.T) {

	t.Parallel()

	inter, _, err := parseCheckAndInterpretWithLogs(t, `
        access(all) resource R{}

        access(all) fun collect(copy2: @R?, _ arrRef: auth(Mutate) &[R]): @R {
            arrRef.append(<- copy2!)
            return <- create R()
        }

        access(all) fun main() {
            var victim: @R? <- create R()
            var arr: @[R] <- []

            // In the optional binding below, the 'victim' must be invalidated
            // before evaluation of the collect() call
            if let copy1 <- victim <- collect(copy2: <- victim, &arr as auth(Mutate) &[R]) {
                arr.append(<- copy1)
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

        access(all) fun collect(copy2: @R?, _ arrRef: auth(Mutate) &[R]): @R {
            arrRef.append(<- copy2!)
            return <- create R()
        }

        access(all) fun main() {
            var victim: @R? <- create R()
            var arr: @[R] <- []

            // In the optional binding below, the 'victim' must be invalidated
            // before evaluation of the collect() call
            let copy1 <- victim <- collect(copy2: <- victim, &arr as auth(Mutate) &[R])

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

func TestInterpretPreConditionResourceMove(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t, `
        access(all) resource Vault { }
        access(all) resource interface Interface {
            access(all) fun foo(_ r: @AnyResource) {
                pre {
                    consume(&r as &AnyResource, <- r)
                }
            }
        }
        access(all) resource Implementation: Interface {
            access(all) fun foo(_ r: @AnyResource) {
                pre {
                    consume(&r as &AnyResource, <- r)
                }
            }
        }
        access(all) fun consume(_ unusedRef: &AnyResource?, _ r: @AnyResource): Bool {
            destroy r
            return true
        }
        access(all) fun main() {
            let a <- create Implementation()
            let b <- create Vault()
            a.foo(<-b)
            destroy a
        }`,
		ParseCheckAndInterpretOptions{
			HandleCheckerError: func(err error) {
				checkerErrors := checker.RequireCheckerErrors(t, err, 3)
				require.IsType(t, &sema.PurityError{}, checkerErrors[0])
				require.IsType(t, &sema.InvalidInterfaceConditionResourceInvalidationError{}, checkerErrors[1])
				require.IsType(t, &sema.PurityError{}, checkerErrors[2])
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("main")
	RequireError(t, err)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}
