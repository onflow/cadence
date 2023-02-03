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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

// TODO: test multiple initializers once overloading is supported

func TestCheckInvalidFieldInitializationEmptyInitializer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          var foo: Int
          var bar: Int

          init(foo: Int) {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	assert.IsType(t, &sema.FieldUninitializedError{}, errs[1])
}

func TestCheckFieldInitializationFromArgument(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       struct Test {
           var foo: Int

           init(foo: Int) {
               self.foo = foo
           }
       }
   `)

	require.NoError(t, err)
}

func TestCheckFieldInitializationWithFunctionCallAfterAllFieldsInitialized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          var foo: Int

          init() {
              self.foo = 1
              self.bar()
          }

          fun bar() {}
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidFieldInitializationWithFunctionCallBeforeAllFieldsInitialized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          var foo: Int

          init() {
              self.bar()
              self.foo = 1
          }

          fun bar() {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UninitializedUseError{}, errs[0])
}

func TestCheckInvalidFieldInitializationWithUseBeforeAllFieldsInitialized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          var foo: Int
          var bar: Int

          init() {
              self.foo = 1
              bar(self)
              self.bar = 2
          }
      }

      fun bar(_ test: Test) {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UninitializedUseError{}, errs[0])
}

func TestCheckConstantFieldInitialization(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          let foo: Int

          init() {
              self.foo = 1
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidRepeatedFieldInitialization(t *testing.T) {

	t.Parallel()

	t.Run("constant, struct-kinded", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct Test {
              let foo: Int

              init() {
                  self.foo = 1
                  self.foo = 1
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.FieldReinitializationError{}, errs[0])

	})

	t.Run("variable, struct-kinded", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct Test {
              var foo: Int

              init() {
                  self.foo = 1
                  self.foo = 1
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("variable, resourced-kinded", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  resource R {}

          resource Test {
              var r: @R

              init() {
                  self.r <- create R()
                  self.r <- create R()
              }

		      destroy() {
		          destroy self.r
		      }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.FieldReinitializationError{}, errs[0])
	})
}

func TestCheckInvalidResourceMoveAfterInitialization(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		  resource R {}

          resource Test {
              var r: @R

              init() {
                  self.r <- create R()
                  let r <- self.r
		          destroy r
              }

		      destroy() {
		          destroy self.r
		      }
          }
        `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[0])
}

func TestCheckFieldInitializationInIfStatement(t *testing.T) {

	t.Parallel()

	t.Run("ValidIfStatement", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int

               init() {
                   if 1 > 2 {
                       self.foo = 1
                   } else {
                       self.foo = 2
                   }
               }
           }
       `)

		require.NoError(t, err)
	})

	t.Run("InvalidIfStatement", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int

               init() {
                   if 1 > 2 {
                       self.foo = 1
                   }
               }
           }
       `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})
}

func TestCheckFieldInitializationInWhileStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        struct Test {
            var foo: Int

            init() {
                while 1 < 2 {
                    self.foo = 1
                }
            }
        }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
}

func TestCheckFieldInitializationFromField(t *testing.T) {

	t.Parallel()

	t.Run("FromInitializedField", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int
               var bar: Int

               init() {
                   self.foo = 1
                   self.bar = self.foo + 1
               }
           }
       `)

		require.NoError(t, err)
	})

	t.Run("FromUninitializedField", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int
               var bar: Int

               init() {
                   self.bar = self.foo + 1
               }
           }
       `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UninitializedFieldAccessError{}, errs[0])
		assert.IsType(t, &sema.FieldUninitializedError{}, errs[1])
	})
}

func TestCheckFieldInitializationUsages(t *testing.T) {

	t.Parallel()

	t.Run("InitializedUsage", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           fun myFunc(_ x: Int) {}

           struct Test {
               var foo: Int

               init() {
                   self.foo = 1
                   myFunc(self.foo)
               }
           }
       `)

		require.NoError(t, err)

	})

	t.Run("UninitializedUsage", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           fun myFunc(_ x: Int) {}

           struct Test {
               var foo: Int

               init() {
                   myFunc(self.foo)
                   self.foo = 1
               }
           }
       `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UninitializedFieldAccessError{}, errs[0])
	})

	t.Run("IfStatementUsage", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int

               init() {
                   if self.foo > 0 {

                   }

                   self.foo = 1
               }
           }
       `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UninitializedFieldAccessError{}, errs[0])
	})

	t.Run("ArrayLiteralUsage", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int

               init() {
                   var x = [self.foo]

                   self.foo = 1
               }
           }
       `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UninitializedFieldAccessError{}, errs[0])
	})

	t.Run("BinaryOperationUsage", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int

               init() {
                   var x = 4 + self.foo

                   self.foo = 1
               }
           }
       `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UninitializedFieldAccessError{}, errs[0])
	})

	t.Run("ComplexUsages", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var a: Int
               var b: Int
               var c: Int

               init(x: Int) {
                   self.a = x

                   if self.a < 4 {
                       self.b = self.a + 2
                   } else {
                       self.b = 0
                   }

                   if self.a + self.b < 5 {
                       self.c = self.a
                   } else {
                       self.c = self.b
                   }
               }
           }
       `)

		require.NoError(t, err)
	})
}

func TestCheckFieldInitializationWithReturn(t *testing.T) {

	t.Parallel()

	t.Run("Direct", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int

               init(foo: Int) {
                   return
                   self.foo = foo
               }
           }
       `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.FieldUninitializedError{}, errs[1])
	})

	t.Run("InsideIf", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int

               init(foo: Int) {
                   if false {
                       return
                   }
                   self.foo = foo
               }
           }
       `)

		// NOTE: at the moment the definite initialization analysis only considers
		// an initialization definitive if there is no return or jump,
		// even if it is only potential

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})

	t.Run("InsideWhile", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
           struct Test {
               var foo: Int

               init(foo: Int) {
                   while false {
                       return
                   }
                   self.foo = foo
               }
           }
       `)

		// NOTE: at the moment the definite initialization analysis only considers
		// an initialization definitive if there is no return or jump,
		// even if it is only potential

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})

	t.Run("inside for", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Test {
                var foo: Int

                init(foo: Int) {
                    for i in [] {
                        return
                    }
                    self.foo = foo
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})

	t.Run("inside switch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct Test {
              let n: Int

              init(n: Int) {
                  switch n {
                  case 1:
                      return
                  }
                  self.n = n
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})
}

func TestCheckFieldInitializationWithPotentialNeverCallInElse(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(
		t,
		`
          struct Test {
              let foo: Int

              init(foo: Int?) {
                  if let foo = foo {
                      self.foo = foo
                  } else {
                      panic("no x")
                  }
              }
          }
        `,
	)

	require.NoError(t, err)
}

func TestCheckFieldInitializationWithPotentialNeverCallInNilCoalescing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(t,
		`
          struct Test {
              let foo: Int

              init(foo: Int?) {
                  self.foo = foo ?? panic("no x")
              }
          }
        `,
	)

	require.NoError(t, err)
}

func TestCheckInvalidFieldInitializationWithUseOfUninitializedInPrecondition(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          var on: Bool

          init() {
              pre { self.on }
              self.on = true
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UninitializedFieldAccessError{}, errs[0])
}

func TestCheckFieldInitializationSwitchCase(t *testing.T) {

	t.Parallel()

	t.Run("only initialized in one case, missing default", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
         struct Test {
             let n: Int

             init(n: Int) {
                 switch n {
                 case 1:
                     self.n = n
                 }
             }
         }
       `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})

	t.Run("initialized in all cases", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct Test {
              let n: Int

              init(n: Int) {
                  switch n {
                  case 1:
                      self.n = n
                  default:
                      self.n = n
                  }
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("uninitialized due to break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct Test {
              let n: Int

              init(n: Int) {
                  switch n {
                  case 1:
                      break
                      self.n = n
                  default:
                      self.n = n
                  }
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.FieldUninitializedError{}, errs[1])
	})

	t.Run("definite initialization after statement", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct Test {
              let n: Int

              init(n: Int) {
                  switch n {
                  case 1:
                      self.n = n
                      return
                  }
                  self.n = n
              }
          }
        `)

		// NOTE: at the moment the definite initialization analysis only considers
		// an initialization definitive if there is no return or jump,
		// even if it is only potential

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})
}

func TestCheckFieldInitializationAfterJump(t *testing.T) {

	t.Parallel()

	t.Run("while, continue", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Test {
                var foo: Int

                init(foo: Int) {
                    while true {
                        continue
                    }
                    self.foo = foo
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("while, break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Test {
                var foo: Int

                init(foo: Int) {
                    while true {
                        break
                    }
                    self.foo = foo
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("while, conditional break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Test {
                var foo: Int

                init(foo: Int) {
                    while true {
                        if true {
                           break
                        }

                        self.foo = foo
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})

	t.Run("for, continue", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Test {
                var foo: Int

                init(foo: Int) {
                    for i in [] {
                        continue
                    }
                    self.foo = foo
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("for, break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Test {
                var foo: Int

                init(foo: Int) {
                    for i in [] {
                        break
                    }
                    self.foo = foo
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("for, conditional break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Test {
                var foo: Int

                init(foo: Int) {
                    for i in [] {
                        if true {
                           break
                        }

                        self.foo = foo
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})

	t.Run("switch, break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct Test {
              let n: Int

              init(n: Int) {
                  switch n {
                  case 1:
                      break
                  }
                  self.n = n
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("switch, conditional break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct Test {
              let n: Int

              init(n: Int) {
                  switch n {
                  case 1:
                      if true {
                         break
                      }
                      self.n = n
                  }
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})

	t.Run("nested if, empty", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct Test {

              let n: Int

              init() {
                  if false {
                      self.n = 1
                  } else {
                      if true {}
                      self.n = 2
                  }
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("nested if, missing branch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct Test {

              let n: Int

              init() {
                  if false {
                      self.n = 1
                  } else {
                      if false {
                         self.n = 2
                      }
                  }
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})

	t.Run("nested if, branch missing", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct Test {

              let n: Int

              init() {
                  if false {
                      self.n = 1
                  } else {
                      if false {
                         self.n = 2
                      } else {
                         self.n = 3
                      }
                  }
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("nested if, repeated", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          struct test {

              let a: Int
              let b: Int

              init() {
                  if false {
                      self.a = 1
                  } else {
                      if false {
                         self.a = 1
                      } else {
                         self.a = 2
                      }
                  }

                  if false {
                      self.b = 1
                  } else {
                      if false {
                         self.b = 2
                      } else {
                         self.b = 3
                      }
                  }
              }
          }
        `)

		require.NoError(t, err)
	})
}
