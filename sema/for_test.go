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

package sema_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckForVariableSized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let xs: [Int] = [1, 2, 3]
          for x in xs {
              x
          }
      }
    `)

	assert.NoError(t, err)
}

func TestCheckForConstantSized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let xs: [Int; 3] = [1, 2, 3]
          for x in xs {
              x
          }
      }
    `)

	assert.NoError(t, err)
}

func TestCheckForString(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Character] {
          let characters: [Character] = []
          let hello = "hello"
          for c in hello {
              characters.append(c)
          }
          return characters
      }
    `)

	assert.NoError(t, err)
}

func TestCheckForInclusiveRange(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InterpreterInclusiveRangeConstructor)

	test := func(typ sema.Type) {
		t.Run(typ.String(), func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(
				`
                  fun test() {
                      let start : %[1]s = 1
                      let end : %[1]s = 2
                      let step : %[1]s = 1
                      let range: InclusiveRange<%[1]s> = InclusiveRange(start, end, step: step)

                      for value in range {
                          var typedValue: %[1]s = value
                      }
                  }
                `,
				typ.String(),
			)

			_, err := ParseAndCheckWithOptions(t, code,
				ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
			)
			require.NoError(t, err)
		})
	}

	for _, typ := range sema.AllIntegerTypes {
		// Only test leaf integer types
		switch typ {
		case sema.IntegerType,
			sema.SignedIntegerType,
			sema.FixedSizeUnsignedIntegerType:
			continue
		}

		test(typ)
	}

}

func TestCheckForEmpty(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          for x in [] {}
      }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidForValueNonArray(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          for x in 1 { }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchWithDescriptionError{}, errs[0])
}

func TestCheckInvalidForValueResource(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let xs <- [<-create R()]
          for x in xs { }
          destroy xs
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UnsupportedResourceForLoopError{}, errs[0])
}

func TestCheckForValueDictionaryResource(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let xs <- {"a": <-create R()}
          for x in xs { }
          destroy xs
      }
    `)
	require.NoError(t, err)
}

func TestCheckInvalidForValueDictionaryResource(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          for x in {"a": <-create R()} { }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidForBlock(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          for x in [1, 2, 3] { y }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckForBreakStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               break
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckForIndexBinding(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           for index, x in ["", "", ""] {
                let y: Int = index
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckForIndexBindingTypeErr(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           for index, x in ["", "", ""] {
                let y: String = index
           }
       }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckForIndexBindingReferenceErr(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           for index, x in ["", "", ""] {
                
           }
           let y = index
       }
    `)
	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidForBreakStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               fun () {
                   break
               }
           }
       }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckForContinueStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               continue
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidForContinueStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               fun () {
                   continue
               }
           }
       }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckInvalidForShadowing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		fun x() {
			var array = ["Hello", "World", "Foo", "Bar"]
			var element = "Hi"
			// Not permitted to use previously declared variable as the
			// iteration variable.
			for element in array {
				element
			}
		}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckReferencesInForLoop(t *testing.T) {

	t.Parallel()

	t.Run("Primitive array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun main() {
                var array = ["Hello", "World", "Foo", "Bar"]
                var arrayRef = &array as &[String]

                for element in arrayRef {
                    let e: String = element
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("Struct array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Foo{}

            fun main() {
                var array = [Foo(), Foo()]
                var arrayRef = &array as &[Foo]

                for element in arrayRef {
                    let e: &Foo = element
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("Resource array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource Foo{}

            fun main() {
                var array <- [ <- create Foo(), <- create Foo()]
                var arrayRef = &array as &[Foo]

                for element in arrayRef {
                    let e: &Foo = element
                }

                destroy array
            }
        `)

		require.NoError(t, err)
	})

	t.Run("array with unauthorized reference ", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun main() {
                let array: [&[Int]] = [
                    &[1] as &[Int],
                    &[2] as &[Int]
                ]

                let arrayRef = &array as &[&[Int]]

                for element in arrayRef {
                    let e: &[Int] = element
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("array with authorized reference ", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun main() {
                let array: [auth(Mutate) &[Int]] = [
                    &[1] as auth(Mutate) &[Int],
                    &[2] as auth(Mutate) &[Int]
                ]

                let arrayRef = &array as &[auth(Mutate) &[Int]]

                for element in arrayRef {
                    // OK
                    let e1: &[Int] = element

                    // Error: element type should be unauthorized
                    let e2: auth(Mutate) &[Int] = element
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errs[0], &typeMismatchError)
		assert.Equal(t, 15, typeMismatchError.StartPos.Line)
	})

	t.Run("authorized reference to array with authorized reference ", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun main() {
                let array: [auth(Mutate) &[Int]] = [
                    &[1] as auth(Mutate) &[Int],
                    &[2] as auth(Mutate) &[Int]
                ]

                let arrayRef = &array as auth(Mutate) &[auth(Mutate) &[Int]]

                for element in arrayRef {
                    // OK
                    let e1: &[Int] = element

                    // OK
                    let e2: auth(Mutate) &[Int] = element
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("authorized reference to array with authorized reference, intrersection ", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E1
            entitlement E2
            entitlement E3

            fun main() {
                let array: [auth(E1, E2) &[Int]] = [
                    &[1] as auth(E1, E2) &[Int],
                    &[2] as auth(E1, E2) &[Int]
                ]

                let arrayRef = &array as auth(E2, E3) &[auth(E1, E2) &[Int]]

                for element in arrayRef {
                    // OK
                    let e1: &[Int] = element

                    // OK
                    let e2: auth(E2) &[Int] = element

                    // Error: element type should only have the intersection (E2)
                    let e3: auth(E1, E2) &[Int] = element

                    // Error: element type should only have the intersection (E2)
                    let e4: auth(E2, E3) &[Int] = element
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errs[0], &typeMismatchError)
		assert.Equal(t, 22, typeMismatchError.StartPos.Line)

		require.ErrorAs(t, errs[1], &typeMismatchError)
		assert.Equal(t, 25, typeMismatchError.StartPos.Line)
	})

	t.Run("Dictionary", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Foo{}

            fun main() {
                var foo = {"foo": Foo()}
                var fooRef = &foo as &{String: Foo}

                for key in fooRef {
                    let e: String = key
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("Non iterable", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Foo{}

            fun main() {
                var foo = Foo()
                var fooRef = &foo as &Foo

                for element in fooRef {
                    let e: &Foo = element
                }
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchWithDescriptionError{}, errors[0])
	})

	t.Run("Non existing type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun main() {
                var foo = Foo()
                var fooRef = &foo as &Foo

                for element in fooRef {}
            }
        `)

		errors := RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.NotDeclaredError{}, errors[0])
		assert.IsType(t, &sema.NotDeclaredError{}, errors[1])
	})

	t.Run("Auth ref", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Foo{}

            fun main() {
                var array = [Foo(), Foo()]
                var arrayRef = &array as auth(Mutate) &[Foo]

                for element in arrayRef {
                    let e: &Foo = element    // should be non-auth
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("Auth ref invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Foo{}

            fun main() {
                var array = [Foo(), Foo()]
                var arrayRef = &array as auth(Mutate) &[Foo]

                for element in arrayRef {
                    let e: auth(Mutate) &Foo = element    // should be non-auth
                }
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})

	t.Run("Optional array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Foo{}

            fun main() {
                var array: [Foo?] = [Foo(), Foo()]
                var arrayRef = &array as &[Foo?]

                for element in arrayRef {
                    let e: &Foo? = element    // Should be an optional reference
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("Enum array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            enum Status: Int {
                case On
                case Off
            }

            fun main() {
                var array = [Status.On, Status.Off]
                var arrayRef = &array as &[Status]

                for element in arrayRef {
                    let e: Status = element
                }
            }
        `)

		require.NoError(t, err)
	})
}

func TestCheckForDictionary(t *testing.T) {

	t.Parallel()

	t.Run("basic dictionary iteration", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test() {
                let dict: {String: Int} = {"a": 1, "b": 2, "c": 3}
                for key in dict {
                    let k: String = key
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("empty dictionary", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test() {
                let empty: {Int: String} = {}
                for key in empty {
                    let k: Int = key
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("dictionary reference", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test() {
                let dict: {String: Int} = {"a": 1, "b": 2}
                let dictRef = &dict as &{String: Int}
                for key in dictRef {
                    let k: String = key
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("dictionary with resource values", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let dict <- {"a": <-create R()}
                for key in dict {
                    let k: String = key
                }
                destroy dict
            }
        `)

		require.NoError(t, err)
	})

	t.Run("dictionary with index binding - error", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test() {
                let dict: {String: Int} = {"a": 1, "b": 2}
                for index, key in dict {
                    let i: Int = index
                    let k: String = key
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidDictionaryIndexBindingError{}, errs[0])
	})

	t.Run("integer key dictionary", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test() {
                let dict: {Int: String} = {1: "a", 2: "b", 3: "c"}
                for key in dict {
                    let k: Int = key
                }
            }
        `)

		require.NoError(t, err)
	})
}

func TestCheckBreakInForLoopBodyDoesNotPreventOuterReturn(t *testing.T) {

	t.Parallel()

	// A `break` inside the for-loop body targets the loop, not the enclosing function.
	// The trailing `return 1` must therefore still mark the function as definitely returning.

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            for _ in [1] {
                break
            }
            return 1
        }
    `)

	require.NoError(t, err)
}

func TestCheckContinueInForLoopBodyDoesNotPreventOuterReturn(t *testing.T) {

	t.Parallel()

	// A `continue` inside the for-loop body targets the loop, not the enclosing function.
	// The trailing `return 1` must therefore still mark the function as definitely returning.

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            for _ in [1] {
                continue
            }
            return 1
        }
    `)

	require.NoError(t, err)
}

// TestCheckForLoopBodyMixedExitVariants exercises every unique pair of distinct exit kinds
// (return, halt, break, continue) used as the two branches of an `if-else` inside the for-loop body.
// Every path through the if-else terminates control flow (in some way),
// so any trailing statement must be reported as unreachable.
func TestCheckForLoopBodyMixedExitVariants(t *testing.T) {

	t.Parallel()

	t.Run("break and continue", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                for _ in [1] {
                    if true { break } else { continue }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("break and halt", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                for _ in [1] {
                    if true { break } else { panic("x") }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("break and return", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                for _ in [1] {
                    if true { break } else { return }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("continue and halt", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                for _ in [1] {
                    if true { continue } else { panic("x") }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("continue and return", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                for _ in [1] {
                    if true { continue } else { return }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("halt and return", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                for _ in [1] {
                    if true { panic("x") } else { return }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})
}

// TestCheckForLoopConditionalJumpThenTermination covers the
// "maybe-jump on one path, definite terminator on the other" pattern in
// a for-loop body.
//
// For each (JUMP, TERMINATOR) combination, two assertions:
//   - Code AFTER the loop is reachable: the jump path falls past the
//     loop, so the loop body's `DefinitelyReturned`/`DefinitelyHalted`
//     claim must NOT propagate to the function.
//   - A statement AFTER the terminator inside the body is unreachable:
//     within the body, every path through the if-else does terminate.
func TestCheckForLoopConditionalJumpThenTermination(t *testing.T) {

	t.Parallel()

	t.Run("if break then return; code after loop reachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test(): Int {
                for _ in [1] {
                    if true { break }
                    return 1
                }
                return 2
            }
        `)
		require.NoError(t, err)
	})

	t.Run("if break then return; statement after return is unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                for _ in [1] {
                    if true { break }
                    return
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("if break then halt; code after loop reachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                for _ in [1] {
                    if true { break }
                    panic("x")
                }
                let y = 1
            }
        `)
		require.NoError(t, err)
	})

	t.Run("if break then halt; statement after halt is unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                for _ in [1] {
                    if true { break }
                    panic("x")
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("if continue then return; code after loop reachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test(): Int {
                for _ in [1] {
                    if true { continue }
                    return 1
                }
                return 2
            }
        `)
		require.NoError(t, err)
	})

	t.Run("if continue then return; statement after return is unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                for _ in [1] {
                    if true { continue }
                    return
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("if continue then halt; code after loop reachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                for _ in [1] {
                    if true { continue }
                    panic("x")
                }
                let y = 1
            }
        `)
		require.NoError(t, err)
	})

	t.Run("if continue then halt; statement after halt is unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                for _ in [1] {
                    if true { continue }
                    panic("x")
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})
}

func TestCheckNestedForLoopBreakDoesNotEscapeOuterLoop(t *testing.T) {

	t.Parallel()

	// A `break` inside the inner for-loop targets the inner loop only.
	// Code after the inner loop, but still in the outer loop body,
	// must remain reachable.

	_, err := ParseAndCheck(t, `
        fun test() {
            for i in [1] {
                for j in [2] {
                    break
                }
                let x = 1
            }
        }
    `)

	require.NoError(t, err)
}

func TestCheckNestedForLoopMaybeJumpedDoesNotEscape(t *testing.T) {

	t.Parallel()

	// A `MaybeJumpedLoop` set inside an inner for-loop body must not leak
	// into the outer loop's body state — `WithLoop` save/restores
	// `MaybeJumpedLoop`.
	_, err := ParseAndCheck(t, `
        fun test(): Int {
            for i in [1] {
                for j in [2] {
                    if true { break }
                    return 1
                }
                let x = 1
            }
            return 2
        }
    `)

	require.NoError(t, err)
}

// TestCheckForLoopWithSwitchInBody verifies that a switch nested in a for-loop body
// interacts correctly with the loop's control flow:
// switch-targeting `break` is consumed by the switch,
// `continue` propagates past the switch to the enclosing loop.
func TestCheckForLoopWithSwitchInBody(t *testing.T) {

	t.Parallel()

	t.Run("switch break does not escape loop body", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                for _ in [1] {
                    switch 1 {
                    case 1:
                        break
                    default:
                        break
                    }
                    let x = 1
                }
            }
        `)
		require.NoError(t, err)
	})

	t.Run("all-cases continue makes post-switch in loop unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                for _ in [1] {
                    switch 1 {
                    case 1:
                        continue
                    default:
                        continue
                    }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("nested switch case with maybe-break does not affect outer return", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test(): Int {
                for _ in [1] {
                    switch 1 {
                    case 1:
                        if true { break }
                        return 1
                    default:
                        return 2
                    }
                }
                return 3
            }
        `)
		require.NoError(t, err)
	})

	t.Run("nested switch case with maybe-continue does not over-claim", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test(): Int {
                for _ in [1] {
                    switch 1 {
                    case 1:
                        if true { continue }
                        return 1
                    default:
                        return 2
                    }
                }
                return 3
            }
        `)
		require.NoError(t, err)
	})
}

func TestCheckResourceInForLoopBodyMaybeBreak(t *testing.T) {

	t.Parallel()

	// A for-loop body whose destroy/return path is guarded by a maybe-break:
	// on the break path, the resource is not destroyed and the loop is exited,
	// so the resource is potentially lost.

	_, err := ParseAndCheck(t, `
        resource R {}
        fun test(cond: Bool) {
            let r <- create R()
            for _ in [1] {
                if cond { break }
                destroy r
                return
            }
        }
    `)

	errs := RequireCheckerErrors(t, err, 2)
	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckGuardElseBreakInForLoop(t *testing.T) {

	t.Parallel()

	// A `guard ... else { break }` inside a for-loop body
	// must propagate the potential loop-targeting jump out of the (potentially-unevaluated) else block,
	// so code after the loop remains reachable.

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            for _ in [1] {
                guard let y = (nil as Int?) else { break }
                return y
            }
            return 3
        }
    `)

	require.NoError(t, err)
}

func TestCheckGuardElseContinueInForLoop(t *testing.T) {

	t.Parallel()

	// A `continue` is a valid definite exit for a guard's else block.
	// Like `break`, the potential loop-targeting jump must propagate
	// out of the (potentially-unevaluated) else block,
	// so code after the loop remains reachable.

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            for _ in [1] {
                guard let y = (nil as Int?) else { continue }
                return y
            }
            return 3
        }
    `)

	require.NoError(t, err)
}

func TestCheckResourceInvalidationInForLoop(t *testing.T) {

	t.Parallel()

	t.Run("break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    break
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("continue", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    continue
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    return
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})
}

func TestCheckResourceInvalidationInForLoopWithIfElse(t *testing.T) {

	t.Parallel()

	t.Run("if break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    if true {
                        break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if continue", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    if true {
                        continue
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if-else both break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    if true {
                        break
                    } else {
                        break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if break, destroy after", func(t *testing.T) {
		t.Parallel()

		// The `destroy r` only runs on the non-break path,
		// so `r` leaks if the `break` is taken.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    if true {
                        break
                    }
                    destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if break else destroy", func(t *testing.T) {
		t.Parallel()

		// The `destroy r` only runs on the else path,
		// so `r` leaks if the `break` is taken.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    if true {
                        break
                    } else {
                        destroy r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if destroy else break", func(t *testing.T) {
		t.Parallel()

		// The `destroy r` only runs on the then path,
		// so `r` leaks if the `break` is taken.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    if true {
                        destroy r
                    } else {
                        break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if destroy else destroy break", func(t *testing.T) {
		t.Parallel()

		// `r` is destroyed in both branches before the `break`, so no loss.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    if true {
                        destroy r
                        break
                    } else {
                        destroy r
                    }
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("nested if break in inner if", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    if true {
                        if true {
                            break
                        }
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("resource outside loop, if break", func(t *testing.T) {
		t.Parallel()

		// `r` is declared outside the loop and destroyed after the loop,
		// so the `break` does not leak it.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let r <- create R()
                for i in [2] {
                    if true {
                        break
                    }
                }
                destroy r
            }
        `)

		require.NoError(t, err)
	})
}

func TestCheckResourceInvalidationInNestedForLoops(t *testing.T) {

	t.Parallel()

	t.Run("break in inner loop, destroy in outer", func(t *testing.T) {
		t.Parallel()

		// `r` is declared in the outer loop and destroyed there.
		// The inner `break` only exits the inner loop, so no leak.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    for j in [2] {
                        break
                    }
                    destroy r
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("resource in inner loop, inner break", func(t *testing.T) {
		t.Parallel()

		// `r` is declared in the inner loop body and the `break` exits it
		// without destroying `r`.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    for j in [2] {
                        let r <- create R()
                        break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("outer break inside inner loop", func(t *testing.T) {
		t.Parallel()

		// `break` always targets the innermost loop,
		// so the inner `break` only exits the inner loop.
		// The outer loop's `r` is destroyed at the end of each outer iteration.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                for i in [2] {
                    let r <- create R()
                    for j in [2] {
                        if true {
                            break
                        }
                    }
                    destroy r
                }
            }
        `)

		require.NoError(t, err)
	})
}
