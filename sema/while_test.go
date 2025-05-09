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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckInvalidWhileTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          while 1 {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckWhileTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          while true {}
      }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidWhileBlock(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          while true { x }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckWhileBreakStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               break
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidWhileBreakStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               fun () {
                   break
               }
           }
       }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckWhileContinueStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               continue
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidWhileContinueStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               fun () {
                   continue
               }
           }
       }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckInvalidBreakStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          break
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckInvalidContinueStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          continue
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}
