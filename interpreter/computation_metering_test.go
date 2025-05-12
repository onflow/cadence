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

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

type testComputationGauge struct {
	meter   map[common.ComputationKind]uint64
	kindSet map[common.ComputationKind]struct{}
	usages  []common.ComputationUsage
}

var _ common.ComputationGauge = &testComputationGauge{}

func (g *testComputationGauge) MeterComputation(usage common.ComputationUsage) error {
	if g.meter == nil {
		g.meter = make(map[common.ComputationKind]uint64)
	}
	g.meter[usage.Kind] += usage.Intensity

	_, ok := g.kindSet[usage.Kind]
	if g.kindSet == nil || ok {
		g.usages = append(g.usages, usage)
	}

	return nil
}

func newTestComputationGauge(
	kinds ...common.ComputationKind,
) *testComputationGauge {

	var kindSet map[common.ComputationKind]struct{}
	if len(kinds) > 0 {
		kindSet = make(map[common.ComputationKind]struct{}, len(kinds))
		for _, kind := range kinds {
			kindSet[kind] = struct{}{}
		}
	}

	return &testComputationGauge{
		kindSet: kindSet,
	}
}

func TestInterpretComputationMeteringArrayFunctions(t *testing.T) {

	t.Parallel()

	t.Run("reverse", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x = [1, 2, 3]
                let y = x.reverse()
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},

				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayGet, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayGet, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayGet, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},
			},
			computationGauge.usages,
		)
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x = [1, 2, 3, 4]
                let trueForEven = fun (_ x: Int): Bool {
                    return x % 2 == 0
                }
                let y = x.map(trueForEven)
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 4},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 4},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 4},
			},
			computationGauge.usages,
		)
	})

	t.Run("filter", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x = [1, 2, 3, 4, 5]
                let onlyEven = view fun (_ x: Int): Bool {
                    return x % 2 == 0
                }
                let y = x.filter(onlyEven)
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 5},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 5},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 5},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 5},

				{Kind: common.ComputationKindStatement, Intensity: 1},

				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 5},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 2},
			},
			computationGauge.usages,
		)
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x = [1, 2, 3, 4, 5, 6]
                let y = x.slice(from: 1, upTo: 4)
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 6},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 6},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 6},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 6},

				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},
			},
			computationGauge.usages,
		)
	})

	t.Run("concat", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x = [1, 2, 3]
                let y = x.concat([4, 5, 6])
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// Computation is (arrayLength +1). It's an overestimate.
		// The last one is for checking the end of array.
		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},

				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 6},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 6},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 6},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 6},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 6},
			},
			computationGauge.usages,
		)
	})
}

func TestInterpretComputationMeteringStdlib(t *testing.T) {

	t.Parallel()

	t.Run("string join", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = String.join(["one", "two", "three", "four"], separator: ", ")
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("string concat", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = "a b c".concat("1 2 3")
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 10},
			},
			computationGauge.usages,
		)
	})

	t.Run("string replace all", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = "abcadeaf".replaceAll(of: "a", with: "z")
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 8},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 8},
				{Kind: common.ComputationKindLoop, Intensity: 7},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 7},
				{Kind: common.ComputationKindLoop, Intensity: 4},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 4},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 8},
				{Kind: common.ComputationKindLoop, Intensity: 8},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 8},
				{Kind: common.ComputationKindLoop, Intensity: 7},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 7},
				{Kind: common.ComputationKindLoop, Intensity: 4},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 4},
			},
			computationGauge.usages,
		)
	})

	t.Run("string to lower", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = "ABCdef".toLower()
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindStringToLower, Intensity: 6},
			},
			computationGauge.usages,
		)
	})

	t.Run("string split", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = "abc/d/ef//".split(separator: "/")
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 10},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 10},
				{Kind: common.ComputationKindLoop, Intensity: 6},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 6},
				{Kind: common.ComputationKindLoop, Intensity: 4},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 4},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 5},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 10},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 3},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 10},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 6},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 6},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 4},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 2},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 4},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 5},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 5},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 5},
			},
			computationGauge.usages,
		)
	})
}

func TestInterpretComputationMeteringStatements(t *testing.T) {

	t.Parallel()

	t.Run("function statements", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge(
			common.ComputationKindStatement,
		)

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t, `
              fun a() {
                  true
                  true
                  true
              }

              fun b() {
                  true
                  true
                  a()
                  true
                  true
              }

              fun c() {
                  true
                  b()
                  true
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("c")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("pre and post conditions", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge(
			common.ComputationKindStatement,
		)

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t, `
              fun test() {
                  pre { true}
                  post { true }
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("global declarations", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge(
			common.ComputationKindStatement,
		)

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t, `
              let x = 1 + 2
              let y = 3 * 4
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		assert.Equal(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
			},
			computationGauge.usages,
		)
	})
}

func TestInterpretComputationMeteringLoopIteration(t *testing.T) {

	t.Parallel()

	t.Run("while", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge(
			common.ComputationKindStatement,
			common.ComputationKindLoop,
		)

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  var i = 1
                  while i <= 3 {
                      i = i + 1
                  }
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				// statement before loop
				{Kind: common.ComputationKindStatement, Intensity: 1},

				// test expression
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				// statement in loop body
				{Kind: common.ComputationKindStatement, Intensity: 1},

				// test expression
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				// statement in loop body
				{Kind: common.ComputationKindStatement, Intensity: 1},

				// test expression
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				// statement in loop body
				{Kind: common.ComputationKindStatement, Intensity: 1},

				// test expression
				{Kind: common.ComputationKindStatement, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("for", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge(
			common.ComputationKindLoop,
		)

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  for n in [1, 2, 3] {}
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				// loop iterations
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
			},
			computationGauge.usages,
		)
	})
}

func TestInterpretComputationMeteringFunctionInvocation(t *testing.T) {

	t.Parallel()

	computationGauge := newTestComputationGauge(
		common.ComputationKindStatement,
		common.ComputationKindFunctionInvocation,
	)

	storage := newUnmeteredInMemoryStorage()
	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          fun a() {
              true
          }

          fun b() {
              true
              a()
              true
          }

          fun c() {
              true
              true
              b()
              true
              true
          }

          fun d() {
              true
              true
              true
              c()
              true
              true
              true
          }
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				Storage:          storage,
				ComputationGauge: computationGauge,
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("d")
	require.NoError(t, err)

	AssertEqualWithDiff(t,
		[]common.ComputationUsage{
			// start of d
			{Kind: common.ComputationKindStatement, Intensity: 1},
			{Kind: common.ComputationKindStatement, Intensity: 1},
			{Kind: common.ComputationKindStatement, Intensity: 1},
			// c()
			{Kind: common.ComputationKindStatement, Intensity: 1},
			{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
			// start of c
			{Kind: common.ComputationKindStatement, Intensity: 1},
			{Kind: common.ComputationKindStatement, Intensity: 1},
			// b()
			{Kind: common.ComputationKindStatement, Intensity: 1},
			{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
			// start of b
			{Kind: common.ComputationKindStatement, Intensity: 1},
			// a()
			{Kind: common.ComputationKindStatement, Intensity: 1},
			{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
			// a
			{Kind: common.ComputationKindStatement, Intensity: 1},
			// rest of b
			{Kind: common.ComputationKindStatement, Intensity: 1},
			// rest of c
			{Kind: common.ComputationKindStatement, Intensity: 1},
			{Kind: common.ComputationKindStatement, Intensity: 1},
			// rest of d
			{Kind: common.ComputationKindStatement, Intensity: 1},
			{Kind: common.ComputationKindStatement, Intensity: 1},
			{Kind: common.ComputationKindStatement, Intensity: 1},
		},
		computationGauge.usages,
	)
}

func TestInterpretComputationMeteringArray(t *testing.T) {

	t.Parallel()

	t.Run("construction and transfer", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              let x = [1, 2, 3]
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},
			},
			computationGauge.usages,
		)
	})

	t.Run("destruction", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              resource R {}

              fun test() {
                  let x: @[R?] <- [nil]
                  destroy x
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindDestroyArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
               let x = [1, 2, 3][1]
             `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayGet, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  let x = [1]
                  x[0] = 2
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeArraySet, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("append", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              let x = [1, 2, 3].append(4)
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayAppend, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              let x = [1, 2, 3].insert(at: 1, 1)
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayInsert, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              let x = [1, 2, 3].remove(at: 1)
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayRemove, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("contains", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              let x = [1, 2, 3].contains(4)
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindWordSliceComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindWordSliceComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindWordSliceComparison, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("appendAll", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
               fun test() {
                   [1, 2, 3].appendAll([4, 5])
               }
             `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayAppend, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayAppend, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("firstIndex", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
               fun test() {
                   ["a", "b", "c"].firstIndex(of: "b")
               }
             `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("concat", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
               fun test() {
                   ["a", "b", "c"].concat(["d", "e"])
               }
             `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 5},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 5},
			},
			computationGauge.usages,
		)
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
               fun test() {
                   ["a", "b", "c", "d"].slice(from: 1, upTo: 3)
               }
             `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 2},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
			},
			computationGauge.usages,
		)
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
               fun test() {
                   for n in [1, 2, 3] {}
               }
             `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("toVariableSized", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
               fun test() {
                   let chars: [Character; 3] = ["a", "b", "c"]
                   chars.toVariableSized()
               }
             `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
			},
			computationGauge.usages,
		)
	})

	t.Run("toConstantSized", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
               fun test() {
                   let chars: [Character] = ["a", "b", "c"]
                   chars.toConstantSized<[Character; 3]>()
               }
             `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 3},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
			},
			computationGauge.usages,
		)
	})

}

func TestInterpretComputationMeteringDictionary(t *testing.T) {

	t.Parallel()

	t.Run("construction and transfer", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              let x = {"a": 1, "b": 2, "c": 3}
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindTransferDictionaryValue, Intensity: 3},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 3},
				{Kind: common.ComputationKindAtreeMapBatchConstruction, Intensity: 3},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("destruction", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              resource R {}

              fun test() {
                  let x: @{String: R?} <- {"r": nil}
                  destroy x
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindTransferDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindDestroyDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              let x = {"a": 1, "b": 2}["b"]
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapGet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("containsKey", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              let x = {"a": 1, "b": 2}.containsKey("b")
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapHas, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  let x = {"a": 1}
                  x["a"] = 2
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindTransferDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapBatchConstruction, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              let x= {"a": 1}.remove(key: "a")
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapRemove, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("keys", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  {"a": 1, "b": 2}.keys
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 2},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
			},
			computationGauge.usages,
		)
	})

	t.Run("values", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  {"a": 1, "b": 2}.values
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 2},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
			},
			computationGauge.usages,
		)
	})

	t.Run("forEachKey", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  {"a": 1, "b": 2}.forEachKey(fun (key: String): Bool {
                      return true
                  })
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateDictionaryValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindStringComparison, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
			},
			computationGauge.usages,
		)
	})
}

func TestInterpretComputationMeteringComposite(t *testing.T) {

	t.Parallel()

	t.Run("construction and transfer", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              struct S {
                  let x: Int

                  init() {
                      self.x = 1
                  }
              }

              let s = S()
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindCreateCompositeValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindTransferCompositeValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapBatchConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapReadIteration, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("destruction", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              resource R {}

              fun test() {
                  let x <- create R()
                  destroy x
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindCreateCompositeValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindTransferCompositeValue, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindDestroyCompositeValue, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
              struct S {
                  let x: Int

                  init() {
                      self.x = 1
                  }
              }

              let x = S().x
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindCreateCompositeValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapGet, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              struct S {
                  let x: Int

                  init() {
                      self.x = 1
                  }
              }

              fun test() {
                  S()
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindCreateCompositeValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapConstruction, Intensity: 1},
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindAtreeMapSet, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

}

func TestInterpretComputationMeteringString(t *testing.T) {

	t.Parallel()

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
             let x = "abc"[2]
           `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				// TODO: optimize
				// length (bounds check)
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				//
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 3},
			},
			computationGauge.usages,
		)
	})

	t.Run("length", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
             let x = "abc".length
           `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
			},
			computationGauge.usages,
		)
	})

	t.Run("toLower", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
             let x = "abc".toLower()
           `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindStringToLower, Intensity: 3},
			},
			computationGauge.usages,
		)
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		_, err := parseCheckAndInterpretWithOptions(t,
			`
             let x = "abcd".slice(from: 1, upTo: 3)
           `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 3},
			},
			computationGauge.usages,
		)
	})

	t.Run("decodeHex", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
             fun test() {
                 "0D15EA5E".decodeHex()
             }
           `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindStringDecodeHex, Intensity: 8},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
			},
			computationGauge.usages,
		)
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
             fun test() {
                 for n in "abc" {}
             }
           `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				// loop iterations
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
				{Kind: common.ComputationKindLoop, Intensity: 1},
				{Kind: common.ComputationKindGraphemesIteration, Intensity: 1},
			},
			computationGauge.usages,
		)
	})
}

func TestInterpretComputationMeteringRLP(t *testing.T) {

	t.Parallel()

	t.Run("decodeString", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.RLPContract)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, stdlib.RLPContract)

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
             fun test() {
                 // "dog"
                 RLP.decodeString([0x83, 0x64, 0x6f, 0x67])
             }
           `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 4},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindSTDLIBRLPDecodeString, Intensity: 4},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 3},
			},
			computationGauge.usages,
		)
	})

	t.Run("decodeList", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.RLPContract)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, stdlib.RLPContract)

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
             fun test() {
                 // [['a']]
                 RLP.decodeList([193, 65])
             }
           `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindTransferArrayValue, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 2},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayReadIteration, Intensity: 1},
				{Kind: common.ComputationKindSTDLIBRLPDecodeList, Intensity: 2},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 1},
				{Kind: common.ComputationKindCreateArrayValue, Intensity: 1},
				{Kind: common.ComputationKindAtreeArrayBatchConstruction, Intensity: 1},
			},
			computationGauge.usages,
		)
	})
}

func TestInterpretComputationMeteringIntegerParsing(t *testing.T) {

	t.Parallel()

	t.Run("big int", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  Int.fromString("100000")
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindBigIntParse, Intensity: 6},
			},
			computationGauge.usages,
		)
	})

	t.Run("signed", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  Int8.fromString("42")
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindIntParse, Intensity: 2},
			},
			computationGauge.usages,
		)
	})

	t.Run("unsigned", func(t *testing.T) {
		t.Parallel()

		computationGauge := newTestComputationGauge()

		storage := newUnmeteredInMemoryStorage()
		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  UInt8.fromString("42")
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					Storage:          storage,
					ComputationGauge: computationGauge,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		AssertEqualWithDiff(t,
			[]common.ComputationUsage{
				{Kind: common.ComputationKindStatement, Intensity: 1},
				{Kind: common.ComputationKindFunctionInvocation, Intensity: 1},
				{Kind: common.ComputationKindUintParse, Intensity: 2},
			},
			computationGauge.usages,
		)
	})

}
