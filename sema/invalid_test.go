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
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckSpuriousIdentifierAssignmentInvalidValueTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          fun test() {
              var x = 1
              x = y
          }
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSpuriousIdentifierAssignmentInvalidTargetTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          fun test() {
              var x: X = 1
              x = 1
          }
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSpuriousIndexAssignmentInvalidValueTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          fun test() {
              let values: {String: Int} = {}
              values["x"] = x
          }
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSpuriousIndexAssignmentInvalidElementTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          fun test() {
              let values: {String: X} = {}
              values["x"] = 1
          }
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSpuriousMemberAssignmentInvalidValueTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          struct X {
              var x: Int
              init() {
                  self.x = y
              }
          }
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSpuriousMemberAssignmentInvalidMemberTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
         struct X {
              var y: Y
              init() {
                  self.y = 0
              }
          }
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSpuriousReturnWithInvalidValueTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          fun test(): Int {
              return x
          }
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSpuriousReturnWithInvalidReturnTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          fun test(): X {
              return 1
          }
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSpuriousCastWithInvalidTargetTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          let y = 1 as X
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSpuriousCastWithInvalidValueTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          let y = x as Int
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidInvocationFunctionReturnType(t *testing.T) {

	t.Parallel()

	typeParameter := &sema.TypeParameter{
		Name: "T",
	}

	fType := &sema.FunctionType{
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			&sema.GenericType{
				TypeParameter: typeParameter,
			},
		),
	}

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.StandardLibraryValue{
		Type: fType,
		Name: "f",
		Kind: common.DeclarationKindFunction,
	})

	_, err := ParseAndCheckWithOptions(t,
		`
          let res = [f].reverse()
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvocationTypeInferenceError{}, errs[0])
}

func TestCheckInvalidTypeDefensiveCheck(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.StandardLibraryValue{
		Type: sema.InvalidType,
		Name: "invalid",
		Kind: common.DeclarationKindConstant,
	})

	var r any
	func() {
		defer func() {
			r = recover()
		}()

		_, _ = ParseAndCheckWithOptions(t,
			`
                  let res = invalid
                `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
	}()

	require.IsType(t, errors.UnexpectedError{}, r)
	err := r.(errors.UnexpectedError)
	require.ErrorContains(t, err, "invalid type produced without error")
}

func TestCheckInvalidTypeIndexing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct S {}
      let s = S()
      let res = s[[]]
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
}

func TestCheckInvalidRemove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct S {}

      attachment A for S {}

      fun test() {
          let s = S()
          remove B from s
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}
