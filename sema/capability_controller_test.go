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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckStorageCapabilityController(t *testing.T) {
	t.Parallel()

	parseAndCheck := func(t *testing.T, code string) (*sema.Checker, error) {
		baseActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseActivation.DeclareValue(stdlib.StandardLibraryValue{
			Name: "controller",
			Type: sema.StorageCapabilityControllerType,
			Kind: common.DeclarationKindConstant,
		})

		return ParseAndCheckWithOptions(
			t,
			code,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseActivation
					},
				},
			},
		)
	}

	t.Run("not equatable", func(t *testing.T) {

		_, err := parseAndCheck(t, `
          let equal = controller == controller
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[0])
	})

	t.Run("in scope", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
          let typ = Type<StorageCapabilityController>()
        `)
		require.NoError(t, err)
	})

	t.Run("members", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
          let cap: Capability = controller.capability
          let tag: String = controller.tag
          let borrowType: Type = controller.borrowType
          let capabilityID: UInt64 = controller.capabilityID
          let target: StoragePath = controller.target()
          let _: Void = controller.retarget(/storage/test)

          fun setTag() {
              controller.setTag("something")
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckAccountCapabilityController(t *testing.T) {
	t.Parallel()

	parseAndCheck := func(t *testing.T, code string) (*sema.Checker, error) {
		baseActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseActivation.DeclareValue(stdlib.StandardLibraryValue{
			Name: "controller",
			Type: sema.AccountCapabilityControllerType,
			Kind: common.DeclarationKindConstant,
		})

		return ParseAndCheckWithOptions(
			t,
			code,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseActivation
					},
				},
			},
		)
	}

	t.Run("not equatable", func(t *testing.T) {

		_, err := parseAndCheck(t, `
          let equal = controller == controller
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[0])
	})

	t.Run("in scope", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
          let typ = Type<AccountCapabilityController>()
        `)
		require.NoError(t, err)
	})

	t.Run("members", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
          let cap: Capability = controller.capability
          let tag: String = controller.tag
          let borrowType: Type = controller.borrowType
          let capabilityID: UInt64 = controller.capabilityID

          fun setTag() {
              controller.setTag("something")
          }
        `)

		require.NoError(t, err)
	})
}
