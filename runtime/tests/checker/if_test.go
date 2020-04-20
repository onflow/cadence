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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckIfStatementTest(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          if true {}
      }
    `)

	require.NoError(t, err)
}

func TestCheckIfStatementScoping(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          if true {
              let x = 1
          }
          x
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidIfStatementTest(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          if 1 {}
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidIfStatementElse(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          if true {} else {
              x
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckIfStatementTestWithDeclaration(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test(x: Int?): Int {
          if var y = x {
              return y
          }

          return 0
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidIfStatementTestWithDeclarationReferenceInElse(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test(x: Int?) {
          if var y = x {
              // ...
          } else {
              y
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckIfStatementTestWithDeclarationNestedOptionals(t *testing.T) {

	_, err := ParseAndCheck(t, `
     fun test(x: Int??): Int? {
         if var y = x {
             return y
         }

         return nil
     }
    `)

	require.NoError(t, err)
}

func TestCheckIfStatementTestWithDeclarationNestedOptionalsExplicitAnnotation(t *testing.T) {

	_, err := ParseAndCheck(t, `
     fun test(x: Int??): Int? {
         if var y: Int? = x {
             return y
         }

         return nil
     }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidIfStatementTestWithDeclarationNonOptional(t *testing.T) {

	_, err := ParseAndCheck(t, `
     fun test(x: Int) {
         if var y = x {
             // ...
         }

         return
     }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidIfStatementTestWithDeclarationSameType(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test(x: Int?): Int? {
          if var y: Int? = x {
             return y
          }

          return nil
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}
