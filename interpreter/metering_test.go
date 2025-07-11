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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretStatementHandler(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          access(all) fun a() {
              true
              true
          }
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := ParseAndCheckWithOptions(t,
		`
          import a from "imported"

          fun b() {
              true
              true
              a()
              true
              true
          }

          fun c() {
              true
              true
              b()
              true
              true
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				ImportHandler: func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					assert.Equal(t,
						ImportedLocation,
						importedLocation,
					)

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	type occurrence struct {
		interpreterID int
		line          int
	}

	var occurrences []occurrence
	var nextInterpreterID int
	interpreterIDs := map[*interpreter.Interpreter]int{}

	storage := newUnmeteredInMemoryStorage()
	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		&interpreter.Config{
			Storage: storage,
			OnStatement: func(interpreter *interpreter.Interpreter, statement ast.Statement) {
				id, ok := interpreterIDs[interpreter]
				if !ok {
					id = nextInterpreterID
					nextInterpreterID++
					interpreterIDs[interpreter] = id
				}

				occurrences = append(occurrences, occurrence{
					interpreterID: id,
					line:          statement.StartPosition().Line,
				})
			},
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

				program := interpreter.ProgramFromChecker(importedChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("c")
	require.NoError(t, err)

	assert.Equal(t,
		[]occurrence{
			{0, 13},
			{0, 14},
			{0, 15},
			{0, 5},
			{0, 6},
			{0, 7},
			{1, 3},
			{1, 4},
			{0, 8},
			{0, 9},
			{0, 16},
			{0, 17},
		},
		occurrences,
	)
}

func TestInterpretLoopIterationHandler(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          access(all) fun a() {
              var i = 1
              while i <= 4 {
                  i = i + 1
              }

              for n in [1, 2, 3, 4, 5] {}
          }
        `,
		ParseAndCheckOptions{},
	)
	require.NoError(t, err)

	importingChecker, err := ParseAndCheckWithOptions(t,
		`
          import a from "imported"

          fun b() {
              var i = 1
              while i <= 2 {
                  i = i + 1
              }

              for n in [1, 2, 3] {}

              a()
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				ImportHandler: func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					assert.Equal(t,
						ImportedLocation,
						importedLocation,
					)

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	type occurrence struct {
		interpreterID int
		line          int
	}

	var occurrences []occurrence
	var nextInterpreterID int
	interpreterIDs := map[*interpreter.Interpreter]int{}

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		&interpreter.Config{
			Storage: storage,
			OnLoopIteration: func(inter *interpreter.Interpreter, line int) {
				id, ok := interpreterIDs[inter]
				if !ok {
					id = nextInterpreterID
					nextInterpreterID++
					interpreterIDs[inter] = id
				}

				occurrences = append(occurrences, occurrence{
					interpreterID: id,
					line:          line,
				})
			},
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

				program := interpreter.ProgramFromChecker(importedChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("b")
	require.NoError(t, err)

	assert.Equal(t,
		[]occurrence{
			{0, 6},
			{0, 6},
			{0, 10},
			{0, 10},
			{0, 10},
			{1, 4},
			{1, 4},
			{1, 4},
			{1, 4},
			{1, 8},
			{1, 8},
			{1, 8},
			{1, 8},
			{1, 8},
		},
		occurrences,
	)
}

func TestInterpretFunctionInvocationHandler(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          access(all) fun a() {}

          access(all) fun b() {
              true
              true
              a()
              true
              true
          }
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := ParseAndCheckWithOptions(t,
		`
          import b from "imported"

          access(all) fun c() {
              true
              true
              b()
              true
              true
          }

          access(all) fun d() {
              true
              true
              c()
              true
              true
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				ImportHandler: func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					assert.Equal(t,
						ImportedLocation,
						importedLocation,
					)

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	var occurrences []int
	var nextInterpreterID int
	interpreterIDs := map[*interpreter.Interpreter]int{}

	storage := newUnmeteredInMemoryStorage()
	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		&interpreter.Config{
			Storage: storage,
			OnFunctionInvocation: func(inter *interpreter.Interpreter) {

				id, ok := interpreterIDs[inter]
				if !ok {
					id = nextInterpreterID
					nextInterpreterID++
					interpreterIDs[inter] = id
				}

				occurrences = append(occurrences, id)
			},
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

				program := interpreter.ProgramFromChecker(importedChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("d")
	require.NoError(t, err)

	assert.Equal(t,
		[]int{0, 0, 1},
		occurrences,
	)
}

type computationGaugeFunc func(usage common.ComputationUsage) error

var _ common.ComputationGauge = computationGaugeFunc(nil)

func (f computationGaugeFunc) MeterComputation(usage common.ComputationUsage) error {
	return f(usage)
}

func TestInterpretArrayFunctionsComputationMetering(t *testing.T) {

	t.Parallel()

	t.Run("reverse", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let x = [1, 2, 3]
                let y = x.reverse()
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(3), computationMeteredValues[common.ComputationKindLoop])
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let x = [1, 2, 3, 4]
			    let trueForEven = fun (_ x: Int): Bool {
					return x % 2 == 0
				}
                let y = x.map(trueForEven)
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(5), computationMeteredValues[common.ComputationKindLoop])
	})

	t.Run("filter", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let x = [1, 2, 3, 4, 5]
			    let onlyEven = view fun (_ x: Int): Bool {
					return x % 2 == 0
				}
                let y = x.filter(onlyEven)
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(6), computationMeteredValues[common.ComputationKindLoop])
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)
		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let x = [1, 2, 3, 4, 5, 6]
                let y = x.slice(from: 1, upTo: 4)
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), computationMeteredValues[common.ComputationKindLoop])
	})

	t.Run("concat", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)
		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let x = [1, 2, 3]
                let y = x.concat([4, 5, 6])
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// Computation is (arrayLength +1). It's an overestimate.
		// The last one is for checking the end of array.
		assert.Equal(t, uint64(7), computationMeteredValues[common.ComputationKindLoop])
	})
}

func TestInterpretStdlibComputationMetering(t *testing.T) {

	t.Parallel()

	t.Run("string join", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let s = String.join(["one", "two", "three", "four"], separator: ", ")
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), computationMeteredValues[common.ComputationKindLoop])
	})

	t.Run("string concat", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let s = "a b c".concat("1 2 3")
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(10), computationMeteredValues[common.ComputationKindLoop])
	})

	t.Run("string replace all", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let s = "abcadeaf".replaceAll(of: "a", with: "z")
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(55), computationMeteredValues[common.ComputationKindLoop])
	})

	t.Run("string to lower", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let s = "ABCdef".toLower()
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(6), computationMeteredValues[common.ComputationKindLoop])
	})

	t.Run("string split", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndPrepareWithOptions(t, `
            fun main() {
                let s = "abc/d/ef//".split(separator: "/")
            }`,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					ComputationGauge: computationGaugeFunc(func(usage common.ComputationUsage) error {
						computationMeteredValues[usage.Kind] += usage.Intensity
						return nil
					}),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(58), computationMeteredValues[common.ComputationKindLoop])
	})
}
