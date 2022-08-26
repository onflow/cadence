/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

		errs := ExpectCheckerErrors(t, err, 1)

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
                              pub let x = X
                              pub let y = Y
                            `,
							ParseAndCheckOptions{
								Location: utils.ImportedLocation,
								Config: &sema.Config{
									ErrorShortCircuitingEnabled: true,
								},
							},
						)
						require.Error(t, err)

						return nil, err
					},
				},
			},
		)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ImportedProgramError{}, errs[0])

		err = errs[0].(*sema.ImportedProgramError).Err

		errs = ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}
