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

	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckReferenceInFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          test
      }
    `)

	require.NoError(t, err)
}

func TestCheckParameterNameWithFunctionName(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(test: Int) {
          test
      }
    `)

	require.NoError(t, err)
}

func TestCheckMutuallyRecursiveFunctions(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun isEven(_ n: Int): Bool {
          if n == 0 {
              return true
          }
          return isOdd(n - 1)
      }

      fun isOdd(_ n: Int): Bool {
          if n == 0 {
              return false
          }
          return isEven(n - 1)
      }
    `)

	require.NoError(t, err)
}

func TestCheckMutuallyRecursiveScoping(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(): Int {
         return g()
      }

      let x = f()
      let y = 0

      fun g(): Int {
          return y
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidFunctionDeclarations(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          fun foo() {}
          fun foo() {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckInvalidFunctionRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun foo() {
          fun foo() {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckFunctionAccess(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       pub fun test() {}
    `)

	require.NoError(t, err)
}

func TestCheckInvalidFunctionAccess(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       pub(set) fun test() {}
    `)

	expectInvalidAccessModifierError(t, err)
}

func TestCheckReturnWithoutExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun returnNothing() {
           return
       }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionUseInsideFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun foo() {
          foo()
      }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionReturnFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun foo(): ((Int): Void) {
          return bar
      }

      fun bar(_ n: Int) {}
    `)

	require.NoError(t, err)
}

func TestCheckInvalidParameterTypes(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: X, y: Y) {}
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])

	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])

}

func TestCheckInvalidParameterNameRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(a: Int, a: Int) {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckParameterRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(a: Int) {
          let a = 1
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidParameterAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(a: Int) {
          a = 1
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
}

func TestCheckInvalidArgumentLabelRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x a: Int, x b: Int) {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckArgumentLabelRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(_ a: Int, _ b: Int) {}
    `)

	require.NoError(t, err)
}

func TestCheckInvalidFunctionDeclarationReturnValue(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Int {
          return true
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchErrorNew{}, errs[0])
}

func TestCheckInvalidResourceCapturingThroughVariable(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Kitty {}

      fun makeKittyCloner(): ((): @Kitty) {
          let kitty <- create Kitty()
          return fun (): @Kitty {
              return <-kitty
          }
      }

      let test = makeKittyCloner()
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceCapturingError{}, errs[0])
}

func TestCheckInvalidResourceCapturingThroughParameter(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Kitty {}

      fun makeKittyCloner(kitty: @Kitty): ((): @Kitty) {
          return fun (): @Kitty {
              return <-kitty
          }
      }

      let test = makeKittyCloner(kitty: <-create Kitty())
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceCapturingError{}, errs[0])
}

func TestCheckInvalidSelfResourceCapturing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Kitty {
          fun makeCloner(): ((): @Kitty) {
              return fun (): @Kitty {
                  return <-self
              }
          }
      }

      let kitty <- create Kitty()
      let test = kitty.makeCloner()
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceCapturingError{}, errs[0])
	assert.IsType(t, &sema.InvalidSelfInvalidationError{}, errs[1])
}

func TestCheckInvalidResourceCapturingJustMemberAccess(t *testing.T) {

	t.Parallel()

	// Resource capturing even just for read access (e.g. reading a member) is invalid

	_, err := ParseAndCheck(t, `
      resource Kitty {
          let id: Int

          init(id: Int) {
              self.id = id
          }
      }

      fun makeKittyIdGetter(): ((): Int) {
          let kitty <- create Kitty(id: 1)
          let getId = fun (): Int {
              return kitty.id
          }
          destroy kitty
          return getId
      }

      let test = makeKittyIdGetter()
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceCapturingError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckInvalidFunctionWithResult(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     fun test(): Int {
         let result = 0
         return result
     }
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckFunctionNonExistingField(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f() {}

      let x = f.y
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckStaticFunctionDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithOptions(t,
		`
          static fun test() {}
        `,
		ParseAndCheckOptions{
			ParseOptions: parser.Config{
				StaticModifierEnabled: true,
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidStaticModifierError{}, errs[0])
}

func TestCheckNativeFunctionDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithOptions(t,
		`
          native fun test() {}
        `,
		ParseAndCheckOptions{
			ParseOptions: parser.Config{
				NativeModifierEnabled: true,
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidNativeModifierError{}, errs[0])
}
