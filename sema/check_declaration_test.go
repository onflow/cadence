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

	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckSelfReferencingDeclaration(t *testing.T) {
	t.Parallel()

	t.Run("attachment", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            attachment A for A {
                fun f(_: A) {}
            }
	    `)

		errs := RequireCheckerErrors(t, err, 1)
		_ = errs
	})

	t.Run("interface, initializer (supported entitlements)", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			struct interface SI: SI {
				init()
			}
	    `)

		errs := RequireCheckerErrors(t, err, 1)
		_ = errs
	})

	t.Run("struct interface", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
           struct interface SI: SI {
               fun foo() {
                   self.foo
               }
           }
	    `)

		errs := RequireCheckerErrors(t, err, 1)
		_ = errs
	})
}
