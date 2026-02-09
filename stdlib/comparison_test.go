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

package stdlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestMinOfFunction(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(InterpreterMinOfFunction)

	parseAndCheck := func(t *testing.T, code string) (*sema.Checker, error) {
		return ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
	}

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = minOf(5, 10)
        `)

		require.NoError(t, err)
	})

	t.Run("Int8", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = minOf<Int8>(5, 10)
        `)

		require.NoError(t, err)
	})

	t.Run("UFix64", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = minOf<UFix64>(5.5, 10.5)
        `)

		require.NoError(t, err)
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = minOf("a", "b")
        `)

		require.NoError(t, err)
	})

	t.Run("non-comparable type", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            fun foo(): Void {}
            fun bar(): Void {}
            let result = minOf<fun(): Void>(foo, bar)
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.InvalidTypeArgumentError{}, errs[0])
	})

	t.Run("mismatched types", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = minOf(5, 10.5)
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestMaxOfFunction(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(InterpreterMaxOfFunction)

	parseAndCheck := func(t *testing.T, code string) (*sema.Checker, error) {
		return ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
	}

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = maxOf(5, 10)
        `)

		require.NoError(t, err)
	})

	t.Run("Int16", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = maxOf<Int16>(5, 10)
        `)

		require.NoError(t, err)
	})

	t.Run("Fix64", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = maxOf<Fix64>(5.5, 10.5)
        `)

		require.NoError(t, err)
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = maxOf("a", "b")
        `)

		require.NoError(t, err)
	})

	t.Run("non-comparable type", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = maxOf<{String: Int}>({}, {})
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.InvalidTypeArgumentError{}, errs[0])
	})

	t.Run("mismatched types", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            let result = maxOf(5.5, 10)
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestMinOfFunctionRuntime(t *testing.T) {
	t.Parallel()

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreter(t, `
            access(all) let result = minOf(5, 10)
        `, InterpreterMinOfFunction)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), result)
	})

	t.Run("Int, reversed", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreter(t, `
            access(all) let result = minOf(10, 5)
        `, InterpreterMinOfFunction)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), result)
	})

	t.Run("UFix64, explicit type argument", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreter(t, `
            access(all) let result = minOf<UFix64>(5.5, 10.5)
        `, InterpreterMinOfFunction)

		result := inter.Globals.Get("result").GetValue(inter)
		expected := interpreter.NewUnmeteredUFix64Value(550_000_000)
		assert.Equal(t, expected, result)
	})
}

func TestMaxOfFunctionRuntime(t *testing.T) {
	t.Parallel()

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreter(t, `
            access(all) let result = maxOf(5, 10)
        `, InterpreterMaxOfFunction)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
	})

	t.Run("Int, reversed", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreter(t, `
            access(all) let result = maxOf(10, 5)
        `, InterpreterMaxOfFunction)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
	})

	t.Run("UFix64, explicit type argument", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreter(t, `
            access(all) let result = maxOf<UFix64>(5.5, 10.5)
        `, InterpreterMaxOfFunction)

		result := inter.Globals.Get("result").GetValue(inter)
		expected := interpreter.NewUnmeteredUFix64Value(1_050_000_000)
		assert.Equal(t, expected, result)
	})
}
