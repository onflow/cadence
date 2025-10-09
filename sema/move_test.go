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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckInvalidMoves(t *testing.T) {

	t.Parallel()

	t.Run("contract", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            access(all) contract Foo {
                access(all) fun moveSelf() {
                    var x = self!
                }
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)
		var invalidMoveError *sema.InvalidMoveError
		require.ErrorAs(t, errors[0], &invalidMoveError)
	})

	t.Run("transaction", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            transaction {
                prepare() {
                    var x = true ? self : self
                }
                execute {}
            }
        `)

		errors := RequireCheckerErrors(t, err, 2)
		var invalidMoveError *sema.InvalidMoveError
		require.ErrorAs(t, errors[0], &invalidMoveError)
		require.ErrorAs(t, errors[1], &invalidMoveError)
	})
}

func TestCheckCastedMove(t *testing.T) {

	t.Parallel()

	t.Run("force", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {}

			fun foo(): @R {
				let r: @AnyResource <- create R()
				return <-r as! @R
			}
		`)

		require.NoError(t, err)
	})

	t.Run("static", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {}

			fun foo(): @AnyResource {
				let r <- create R()
				return <-r as @AnyResource
			}
		`)

		require.NoError(t, err)
	})

	t.Run("parenthesized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {}

			fun foo(): @R {
				let r: @AnyResource <- create R()
				return <-(r as! @R)
			}
		`)

		require.NoError(t, err)
	})

	t.Run("function call", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {}

			fun bar(_ r: @R) {
				destroy r
			}

			fun foo() {
				let r: @AnyResource <- create R()
				bar(<-r as! @R)
			}
		`)

		require.NoError(t, err)
	})

	t.Run("function call, parenthesized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {}

			fun bar(_ r: @R) {
				destroy r
			}

			fun foo() {
				let r: @AnyResource <- create R()
				bar(<-(r as! @R))
			}
		`)

		require.NoError(t, err)
	})
}
