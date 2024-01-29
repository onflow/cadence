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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckErrorShortCircuiting(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
              let x: Type<X<X<X>>>? = nil
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					ErrorShortCircuitingEnabled: true,
				},
			},
		)

		// There are actually 6 errors in total,
		// 3 "cannot find type in this scope",
		// and 3 "cannot instantiate non-parameterized type",
		// but we enabled error short-circuiting

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("with import", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
               import "imported"

               let a = A
               let b = B
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					ErrorShortCircuitingEnabled: true,
					ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {

						_, err := ParseAndCheckWithOptions(t,
							`
                              access(all) let x = X
                              access(all) let y = Y
                            `,
							ParseAndCheckOptions{
								Location: utils.ImportedLocation,
								Config: &sema.Config{
									ErrorShortCircuitingEnabled: true,
								},
							},
						)
						errs := RequireCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.NotDeclaredError{}, errs[0])

						return nil, err
					},
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ImportedProgramError{}, errs[0])

		err = errs[0].(*sema.ImportedProgramError).Err

		errs = RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckEntitlementsErrorMessage(t *testing.T) {

	t.Parallel()

	t.Run("reference subtyping", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
				access(all) entitlement E 
				fun foo() {
					let x = 1
					let refX: auth(E) &Int = &x as auth(F) &Int
				}
            `,
		)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.NotDeclaredError{}, errs[0])
		require.IsType(t, &sema.TypeMismatchError{}, errs[1])
		require.Equal(t, "expected `auth(E) &Int`, got `auth(<<INVALID>>) &Int`", errs[1].(*sema.TypeMismatchError).SecondaryError())
	})

	t.Run("invalid access", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
				access(all) entitlement E 
				access(all) resource R {
					access(E) fun foo() {}
				}
				fun foo() {
					let r <- create R() 
					let refR = &r as auth(F) &R
					refR.foo()
					destroy r
				}
            `,
		)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.NotDeclaredError{}, errs[0])
		require.IsType(t, &sema.InvalidAccessError{}, errs[1])
		require.Equal(t, "cannot access `foo`: function requires `E` authorization, but reference only has `<<INVALID>>` authorization", errs[1].(*sema.InvalidAccessError).Error())
	})

	t.Run("interface as type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
				access(all) entitlement E 
				access(all) resource interface I {
					access(E) fun foo()
				}
				access(all) resource R: I  {
					access(E) fun foo() {}
				}
				fun foo() {
					let r <- create R() 
					let refR = &r as auth(F) &I
					destroy r
				}
            `,
		)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.NotDeclaredError{}, errs[0])
		require.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[1])
		require.Equal(t, "got `auth(<<INVALID>>) &I`; consider using `auth(<<INVALID>>) &{I}`", errs[1].(*sema.InvalidInterfaceTypeError).SecondaryError())
	})
}
