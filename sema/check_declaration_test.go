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

func TestCheckSelfReferencingDeclaration(t *testing.T) {
	t.Parallel()

	t.Run("self-attaching attachment, direct, check instantiated", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            attachment A for A {}

			fun f(_: A) {}
	    `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
		assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[1])
	})

	t.Run("self-attaching attachment, transitive, check instantiated", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            attachment A for B {}

            attachment B for A {}

			fun f(_: A) {}
	    `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[2])
	})

	t.Run("self-conforming interface, direct, supported entitlements", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			struct interface SI: SI {
				init()
			}
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CyclicConformanceError{}, errs[0])
	})

	t.Run("self-conforming interface, transitive, supported entitlements", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			struct interface SI1: SI2 {
				init()
			}

            struct interface SI2: SI1 {
				init()
			}
	    `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.CyclicConformanceError{}, errs[0])
		assert.IsType(t, &sema.CyclicConformanceError{}, errs[1])
	})

	t.Run("self-conforming interface, direct, members", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
           struct interface SI: SI {
               fun foo() {
                   self.foo
               }
           }
	    `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.CyclicConformanceError{}, errs[0])
		assert.IsType(t, &sema.InterfaceMemberConflictError{}, errs[1])
	})

	t.Run("self-conforming interface, transitive, members", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
           struct interface SI1: SI2 {
               fun bar() {
                   self.foo()
               }
           }

           struct interface SI2: SI1 {
               fun foo() {
                   self.bar()
               }
           }
	    `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.CyclicConformanceError{}, errs[0])
		assert.IsType(t, &sema.InterfaceMemberConflictError{}, errs[1])
		assert.IsType(t, &sema.CyclicConformanceError{}, errs[2])
		assert.IsType(t, &sema.InterfaceMemberConflictError{}, errs[3])
	})

	t.Run("contract interface", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
           contract interface CI {
               // Note: use the interface type directly, rather than as a intersection type "{CI}"
               init(v: CI) {}
           }
	    `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[0])
	})

	t.Run("contract interface with cyclic conformance", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
           contract interface CI1: CI2 {
               // Note1: Use the interface type directly, rather than as a intersection type "{CI2}"
               // Note2: Use the cyclic conforming interface type "CI2", rather than the enclosing type "CI1" itself.
               init(v: CI2) {}
           }

           contract interface CI2: CI1 {
               // Note1: Use the interface type directly, rather than as a intersection type "{CI1}"
               // Note2: Use the cyclic conforming interface type "CI1", rather than the enclosing type "CI2" itself.
               init(v: CI1) {}
           }
	    `)

		errs := RequireCheckerErrors(t, err, 4)
		assert.IsType(t, &sema.CyclicConformanceError{}, errs[0])
		assert.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[1])
		assert.IsType(t, &sema.CyclicConformanceError{}, errs[2])
		assert.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[3])
	})

	t.Run("contract interface in reference type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
           contract interface CI {
               // Note: use the interface type directly, rather than as a intersection type "{CI}"
               init(v: &CI) {}
           }
	    `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[0])
	})
}
