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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckEventNonTypeRequirementConformance(t *testing.T) {

	t.Parallel()
	_, err := ParseAndCheck(t, `
      access(all) contract interface CI {

          access(all) event E(a: Int)
      }

      access(all) contract C: CI {

          access(all) event E(b: String)
      }
    `)

	require.NoError(t, err)
}

func TestCheckConformanceWithFunctionSubtype(t *testing.T) {

	t.Parallel()

	t.Run("valid, return type is subtype", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun get(): @{RI}
          }

          struct S: SI {
              fun get(): @R {
                  return <- create R()
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("invalid, return type is supertype", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun get(): @R
          }

          struct S: SI {
              fun get(): @{RI} {
                  return <- create R()
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("valid, return type is the same", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun get(): @R
          }

          struct S: SI {
              fun get(): @R {
                  return <- create R()
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("valid, parameter type is the same", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun set(r: @{RI})
          }

          struct S: SI {
              fun set(r: @{RI}) {
                  destroy r
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("invalid, parameter type is subtype", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun set(r: @{RI})
          }

          struct S: SI {
              fun set(r: @R) {
                  destroy r
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("invalid, parameter type is supertype", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun set(r: @R)
          }

          struct S: SI {
              fun set(r: @{RI}) {
                  destroy r
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})
}

func TestCheckInitializerConformanceErrorMessages(t *testing.T) {

	t.Parallel()

	t.Run("initializer notes", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
      access(all) resource interface I {
          let x: Int 
          init(x: Int)
      }

      access(all) resource R: I {
        let x: Int 
        init() {
            self.x = 1
        }
      }
    `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])

		conformanceErr := errs[0].(*sema.ConformanceError)
		require.NotNil(t, conformanceErr.InitializerMismatch)
		notes := conformanceErr.ErrorNotes()
		require.Len(t, notes, 1)

		require.Equal(t, &sema.MemberMismatchNote{
			Range: ast.Range{
				StartPos: ast.Position{Offset: 158, Line: 9, Column: 8},
				EndPos:   ast.Position{Offset: 161, Line: 9, Column: 11},
			},
		}, notes[0])
	})

	t.Run("1 missing member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
        access(all) resource interface I {
            fun foo(): Int
        }

        access(all) resource R: I {
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
		conformanceErr := errs[0].(*sema.ConformanceError)
		require.Equal(t, "`R` is missing definitions for members: `foo`", conformanceErr.SecondaryError())
	})

	t.Run("2 missing member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
        access(all) resource interface I {
            fun foo(): Int
            fun bar(): Int
        }

        access(all) resource R: I {
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
		conformanceErr := errs[0].(*sema.ConformanceError)
		require.Equal(t, "`R` is missing definitions for members: `foo`, `bar`", conformanceErr.SecondaryError())
	})
}
