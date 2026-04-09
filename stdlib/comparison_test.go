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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestMinFunction(t *testing.T) {
	t.Parallel()

	parseAndCheck := func(t *testing.T, code string) (*sema.Checker, error) {
		return ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					ImportHandler: func(
						_ *sema.Checker,
						importedLocation common.Location,
						_ ast.Range,
					) (sema.Import, error) {
						if importedLocation == ComparisonContractLocation {
							return ComparisonContractSemaImport, nil
						}
						return nil, fmt.Errorf("unexpected import: %s", importedLocation)
					},
				},
			},
		)
	}

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: Int = min(5, 10)
        `)

		require.NoError(t, err)
	})

	t.Run("Int8", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: Int8 = min<Int8>(5, 10)
        `)

		require.NoError(t, err)
	})

	t.Run("UFix64", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: UFix64 = min<UFix64>(5.5, 10.5)
        `)

		require.NoError(t, err)
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: String = min("a", "b")
        `)

		require.NoError(t, err)
	})

	t.Run("non-comparable type", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            fun foo(): Void {}
            fun bar(): Void {}
            let result = min<fun(): Void>(foo, bar)
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.InvalidTypeArgumentError{}, errs[0])
	})

	t.Run("mismatched types", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result = min(5, 10.5)
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestMaxFunction(t *testing.T) {
	t.Parallel()

	parseAndCheck := func(t *testing.T, code string) (*sema.Checker, error) {
		return ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					ImportHandler: func(
						_ *sema.Checker,
						importedLocation common.Location,
						_ ast.Range,
					) (sema.Import, error) {
						if importedLocation == ComparisonContractLocation {
							return ComparisonContractSemaImport, nil
						}
						return nil, fmt.Errorf("unexpected import: %s", importedLocation)
					},
				},
			},
		)
	}

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: Int = max(5, 10)
        `)

		require.NoError(t, err)
	})

	t.Run("Int16", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: Int16 = max<Int16>(5, 10)
        `)

		require.NoError(t, err)
	})

	t.Run("Fix64", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
 		   import Comparison

            let result: Fix64 = max<Fix64>(5.5, 10.5)
        `)

		require.NoError(t, err)
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
			import Comparison

            let result: String = max("a", "b")
        `)

		require.NoError(t, err)
	})

	t.Run("non-comparable type", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result = max<{String: Int}>({}, {})
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.InvalidTypeArgumentError{}, errs[0])
	})

	t.Run("mismatched types", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result = max(5.5, 10)
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

// TODO: test with compiler/VM
func newInterpreterWithComparison(t *testing.T, code string) *interpreter.Interpreter {
	program, err := parser.ParseProgram(
		nil,
		[]byte(code),
		parser.Config{},
	)
	require.NoError(t, err)

	checker, err := sema.NewChecker(
		program,
		TestLocation,
		nil,
		&sema.Config{
			ImportHandler: func(
				_ *sema.Checker,
				importedLocation common.Location,
				_ ast.Range,
			) (sema.Import, error) {
				if importedLocation == ComparisonContractLocation {
					return ComparisonContractSemaImport, nil
				}
				return nil, fmt.Errorf("unexpected import: %s", importedLocation)
			},
			AccessCheckMode: sema.AccessCheckModeStrict,
		},
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	storage := NewUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			Storage: storage,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				if location == ComparisonContractLocation {
					return ComparisonContractInterpreterImport
				}
				return nil
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	return inter
}

func TestMinFunctionRuntime(t *testing.T) {
	t.Parallel()

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = min(5, 10)
       `)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), result)
	})

	t.Run("Int, reversed", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = min(10, 5)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), result)
	})

	t.Run("UFix64, explicit type argument", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = min<UFix64>(5.5, 10.5)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		expected := interpreter.NewUnmeteredUFix64Value(550_000_000)
		assert.Equal(t, expected, result)
	})
}

func TestMaxFunctionRuntime(t *testing.T) {
	t.Parallel()

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = max(5, 10)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
	})

	t.Run("Int, reversed", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = max(10, 5)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
	})

	t.Run("UFix64, explicit type argument", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = max<UFix64>(5.5, 10.5)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		expected := interpreter.NewUnmeteredUFix64Value(1_050_000_000)
		assert.Equal(t, expected, result)
	})
}

func TestClampFunction(t *testing.T) {
	t.Parallel()

	parseAndCheck := func(t *testing.T, code string) (*sema.Checker, error) {
		return ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					ImportHandler: func(
						_ *sema.Checker,
						importedLocation common.Location,
						_ ast.Range,
					) (sema.Import, error) {
						if importedLocation == ComparisonContractLocation {
							return ComparisonContractSemaImport, nil
						}
						return nil, fmt.Errorf("unexpected import: %s", importedLocation)
					},
				},
			},
		)
	}

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: Int = clamp(7, min: 1, max: 10)
        `)

		require.NoError(t, err)
	})

	t.Run("Int8", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: Int8 = clamp<Int8>(7, min: 1, max: 10)
        `)

		require.NoError(t, err)
	})

	t.Run("UFix64", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: UFix64 = clamp<UFix64>(7.5, min: 1.0, max: 10.0)
        `)

		require.NoError(t, err)
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result: String = clamp("d", min: "a", max: "f")
        `)

		require.NoError(t, err)
	})

	t.Run("non-comparable type", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result = clamp<{String: Int}>({}, min: {}, max: {})
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.InvalidTypeArgumentError{}, errs[0])
	})

	t.Run("mismatched types", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            import Comparison

            let result = clamp(5, min: 1, max: 10.0)
        `)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestClampFunctionRuntime(t *testing.T) {
	t.Parallel()

	t.Run("Int, within range", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = clamp(7, min: 1, max: 10)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(7), result)
	})

	t.Run("Int, below min", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = clamp(0, min: 1, max: 10)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), result)
	})

	t.Run("Int, above max", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = clamp(20, min: 1, max: 10)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
	})

	t.Run("Int, equal to min", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = clamp(1, min: 1, max: 10)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), result)
	})

	t.Run("Int, equal to max", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = clamp(10, min: 1, max: 10)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
	})

	t.Run("UFix64, explicit type argument", func(t *testing.T) {
		t.Parallel()

		inter := newInterpreterWithComparison(t, `
            import Comparison

            access(all) let result = clamp<UFix64>(7.5, min: 1.0, max: 10.0)
        `)

		result := inter.Globals.Get("result").GetValue(inter)
		expected := interpreter.NewUnmeteredUFix64Value(750_000_000)
		assert.Equal(t, expected, result)
	})
}
