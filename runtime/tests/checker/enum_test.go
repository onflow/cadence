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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckInvalidNonEnumCompositeEnumCases(t *testing.T) {

	t.Parallel()

	test := func(kind common.CompositeKind) {

		kindKeyword := kind.Keyword()

		t.Run(kindKeyword, func(t *testing.T) {

			t.Parallel()

			var baseType string
			if kind == common.CompositeKindAttachment {
				baseType = "for AnyStruct"
			}

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[2]s { case a }
                    `,
					kindKeyword,
					baseType,
				),
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidEnumCaseError{}, errs[0])
		})
	}

	for _, kind := range common.CompositeKindsWithFieldsAndFunctions {

		if kind == common.CompositeKindEnum {
			continue
		}

		test(kind)
	}
}

func TestCheckInvalidEnumCompositeNonEnumCases(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      enum E: Int {
          let a: Int
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidNonEnumCaseError{}, errs[0])
	assert.IsType(t, &sema.MissingInitializerError{}, errs[1])
}

func TestCheckEnumRawType(t *testing.T) {

	t.Parallel()

	t.Run("missing", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          enum E {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.MissingEnumRawTypeError{}, errs[0])
	})

	t.Run("one raw type, non-Integer subtype", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct interface SI {}
          enum E: SI {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidEnumRawTypeError{}, errs[0])
	})

	t.Run("one raw type, Integer subtype", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          enum E: Int {}
        `)

		require.NoError(t, err)
	})

	t.Run("more than one conformance", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct interface S {}

          enum E: Int, S {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidEnumConformancesError{}, errs[0])
	})
}

func TestCheckInvalidEnumInterface(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      enum interface E {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidInterfaceDeclarationError{}, errs[0])
}

func TestCheckInvalidEnumCaseDuplicate(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      enum E: Int {
          case a
          case a
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckInvalidNonPublicEnumCase(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      enum E: Int {
          priv case a
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
}

func TestCheckEnumCaseRawValueField(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      enum E: Int {
          case a
      }

      let e: E = E.a
      let rawValue: Int = e.rawValue
    `)

	require.NoError(t, err)
}

func TestCheckEnumConstructor(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      enum E: Int {
          case a
          case b
          case unknown
      }

      fun test(): E {
          return E(rawValue: 0) ?? E.unknown
      }
    `)

	require.NoError(t, err)
}

func TestCheckEnumInContract(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract C {
          enum E: UInt8 {
              pub case a
              pub case b
          }

          var e: E

          init() {
              self.e = E.a
          }
      }
    `)

	require.NoError(t, err)
}
