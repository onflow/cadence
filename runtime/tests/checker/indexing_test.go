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

func TestCheckArrayIndexingWithInteger(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          let z = [0, 3]
          z[0]
      }
    `)

	require.NoError(t, err)
}

func TestCheckNestedArrayIndexingWithInteger(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          let z = [[0, 1], [2, 3]]
          z[0][1]
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayIndexingWithBool(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          let z = [0, 3]
          z[true]
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotIndexingTypeError{}, errs[0])
}

func TestCheckInvalidArrayIndexingIntoBool(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test(): Int {
          return true[0]
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotIndexableTypeError{}, errs[0])
}

func TestCheckInvalidArrayIndexingIntoInteger(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test(): Int {
          return 2[0]
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotIndexableTypeError{}, errs[0])
}

func TestCheckInvalidArrayIndexingAssignmentWithBool(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          let z = [0, 3]
          z[true] = 2
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotIndexingTypeError{}, errs[0])
}

func TestCheckArrayIndexingAssignmentWithInteger(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          let z = [0, 3]
          z[0] = 2
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayIndexingAssignmentWithWrongType(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          let z = [0, 3]
          z[0] = true
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidStringIndexingWithBool(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          let z = "abc"
          z[true]
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotIndexingTypeError{}, errs[0])
}

func TestCheckInvalidUnknownDeclarationIndexing(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          x[0]
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidUnknownDeclarationIndexingAssignment(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          x[0] = 2
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}
