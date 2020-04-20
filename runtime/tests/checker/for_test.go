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

	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckForVariableSized(t *testing.T) {

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

func TestCheckForEmpty(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          for x in [] {}
      }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidForValueNonArray(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          for x in 1 { }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchWithDescriptionError{}, errs[0])
}

func TestCheckInvalidForValueResource(t *testing.T) {

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let xs <- [<-create R()]
          for x in xs { }
          destroy xs
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UnsupportedResourceForLoopError{}, errs[0])
}

func TestCheckInvalidForBlock(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          for x in [1, 2, 3] { y }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckForBreakStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               break
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidForBreakStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               fun () {
                   break
               }
           }
       }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckForContinueStatement(t *testing.T) {

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

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               fun () {
                   continue
               }
           }
       }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}
