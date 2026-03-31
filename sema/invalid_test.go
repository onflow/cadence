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

	"github.com/onflow/cadence/ast"
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
			CheckerConfig: &sema.Config{
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
				CheckerConfig: &sema.Config{
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

func TestCheckInvalidTypeDefensiveCheckWithImportedError(t *testing.T) {

	t.Parallel()

	// Check a program that has errors.
	// The exported value `x` will have an invalid type
	// because of the undeclared identifier `z`
	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          access(all) let x = z
        `,
		ParseAndCheckOptions{
			Location: common.StringLocation("imported"),
		},
	)
	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])

	xVar, ok := importedChecker.Elaboration.GetGlobalValue("x")
	require.True(t, ok)
	assert.True(t, xVar.Type.IsInvalidType())
	assert.True(t, importedChecker.Elaboration.HasErrors)

	// Import the program with errors.
	// Using the imported value `x` (which has an invalid type)
	// must NOT panic, because the invalid type came from the imported program.
	_, err = ParseAndCheckWithOptions(t,
		`
          import x from "imported"
          access(all) let y = x
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	// No panic, no errors in the importing program
	require.NoError(t, err)
}

func TestCheckInvalidTypeDefensiveCheckWithTransitiveImportedError(t *testing.T) {

	t.Parallel()

	// Check a program that has errors.
	// The exported value `x` will have an invalid type
	// because of the undeclared identifier `z`
	checkerC, err := ParseAndCheckWithOptions(t,
		`
          access(all) let x = z
        `,
		ParseAndCheckOptions{
			Location: common.StringLocation("imported"),
		},
	)
	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])

	xVar, ok := checkerC.Elaboration.GetGlobalValue("x")
	require.True(t, ok)
	assert.True(t, xVar.Type.IsInvalidType())
	assert.True(t, checkerC.Elaboration.HasErrors)

	// Program B imports C (which has errors)
	checkerB, err := ParseAndCheckWithOptions(t,
		`
          import x from "C"
          access(all) let y = x
        `,
		ParseAndCheckOptions{
			Location: common.StringLocation("B"),
			CheckerConfig: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: checkerC.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)
	// B has no local errors, but transitively imported a program with errors
	require.True(t, checkerB.Elaboration.HasErrors)

	// Program A imports B.
	// Using the imported value `y` (which has an invalid type transitively from C)
	// must NOT panic.
	_, err = ParseAndCheckWithOptions(t,
		`
          import y from "B"
          access(all) let z = y
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: checkerB.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)
}
