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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/stretchr/testify/require"
)

func ParseAndCheckCapcon(t *testing.T, code string) (*sema.Checker, error) {
	baseActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseActivation.DeclareValue(stdlib.StandardLibraryValue{
		Name: "controller",
		Type: sema.CapabilityControllerType,
		Kind: common.DeclarationKindConstant,
	})

	return ParseAndCheckWithOptions(
		t,
		code,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivation: baseActivation,
			},
		},
	)
}

func TestCapconNonEquatable(t *testing.T) {
	_, err := ParseAndCheckCapcon(t, `
		let kaboom: Bool = controller == controller
	`)

	errs := RequireCheckerErrors(t, err, 1)
	require.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[0])
}

func TestCapconTypeInScope(t *testing.T) {
	_, err := ParseAndCheckCapcon(t, `
		let typ = Type<CapabilityController>()
	`)
	require.NoError(t, err)
}
