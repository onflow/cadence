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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckRoundingModeCases(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, value := range stdlib.InterpreterDefaultScriptStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(value)
	}

	test := func(mode sema.NativeEnumCase) {

		_, err := ParseAndCheckWithOptions(t,
			fmt.Sprintf(
				`
               let mode: RoundingMode = RoundingMode.%s
            `,
				mode.Name(),
			),
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)

		require.NoError(t, err)
	}

	for _, mode := range sema.RoundingModes {
		test(mode)
	}
}

func TestCheckRoundingModeConstructor(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InterpreterRoundingModeConstructor)

	_, err := ParseAndCheckWithOptions(t,
		`
           let mode = RoundingMode(rawValue: 0)
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckRoundingModeRawValue(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, value := range stdlib.InterpreterDefaultScriptStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(value)
	}

	_, err := ParseAndCheckWithOptions(t,
		`
           let mode = RoundingMode.towardZero
           let rawValue: UInt8 = mode.rawValue
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckFix64WithRoundingMode(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, value := range stdlib.InterpreterDefaultScriptStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(value)
	}

	t.Run("with rounding", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
               let x: Fix64 = Fix64(1, rounding: RoundingMode.towardZero)
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)

		require.NoError(t, err)
	})

	t.Run("without rounding", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
               let x: Fix64 = Fix64(1)
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)

		require.NoError(t, err)
	})

	t.Run("invalid rounding type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
               let x: Fix64 = Fix64(1, rounding: 42)
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)

		require.Error(t, err)
	})
}

func TestCheckUFix64WithRoundingMode(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, value := range stdlib.InterpreterDefaultScriptStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(value)
	}

	t.Run("with rounding", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
               let x: UFix64 = UFix64(1, rounding: RoundingMode.nearestHalfEven)
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)

		require.NoError(t, err)
	})

	t.Run("without rounding", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
               let x: UFix64 = UFix64(1)
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)

		require.NoError(t, err)
	})
}
