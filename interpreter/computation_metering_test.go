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

type computationGaugeFunc func(usage common.ComputationUsage) error

var _ common.ComputationGauge = computationGaugeFunc(nil)

func (f computationGaugeFunc) MeterComputation(usage common.ComputationUsage) error {
	return f(usage)
}

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

func (g *testComputationGauge) getComputation(kind common.ComputationKind) uint64 {
	return g.meter[kind]
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

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x = [1, 2, 3]
                let y = x.reverse()
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
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
		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x = [1, 2, 3, 4, 5, 6]
                let y = x.slice(from: 1, upTo: 4)
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
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
		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let x = [1, 2, 3]
                let y = x.concat([4, 5, 6])
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
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

func TestInterpretComputationMeteringStdlib(t *testing.T) {

	t.Parallel()

	t.Run("string join", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = String.join(["one", "two", "three", "four"], separator: ", ")
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
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

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = "a b c".concat("1 2 3")
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
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

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = "abcadeaf".replaceAll(of: "a", with: "z")
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
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

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = "ABCdef".toLower()
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
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

		assert.Equal(t, uint64(6), computationMeteredValues[common.ComputationKindStringToLower])
	})

	t.Run("string split", func(t *testing.T) {
		t.Parallel()

		computationMeteredValues := make(map[common.ComputationKind]uint64)

		inter, err := parseCheckAndInterpretWithOptions(t, `
            fun main() {
                let s = "abc/d/ef//".split(separator: "/")
            }`,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
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
