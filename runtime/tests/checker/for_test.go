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

	t.Run("Dictionary", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Foo{}

            fun main() {
                var foo = {"foo": Foo()}
                var fooRef = &foo as &{String: Foo}

                for element in fooRef {
                    let e: &Foo = element
                }
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchWithDescriptionError{}, errors[0])
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
}
