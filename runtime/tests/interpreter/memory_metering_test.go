/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/common"
)

type testMemoryGauge struct {
	meter map[common.MemoryKind]uint64
}

func newTestMemoryGauge() *testMemoryGauge {
	return &testMemoryGauge{
		meter: make(map[common.MemoryKind]uint64),
	}
}

func (g *testMemoryGauge) UseMemory(usage common.MemoryUsage) {
	current, ok := g.meter[usage.Kind]
	if !ok {
		current = 0
	}
	g.meter[usage.Kind] = current + usage.Amount
}

func (g *testMemoryGauge) getMemory(kind common.MemoryKind) uint64 {
	return g.meter[kind]
}

func TestRuntimeArrayMetering(t *testing.T) {
	t.Parallel()

	script := `
        pub fun main() {
            let x: [Int8] = []
            let y: [[String]] = [[]]
            let z: [[[Bool]]] = [[[]]]
        }
    `

	meter := newTestMemoryGauge()
	inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

	_, err := inter.Invoke("main")
	require.NoError(t, err)

	assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindArray))
}

func TestRuntimeDictionaryMetering(t *testing.T) {
	t.Parallel()

	script := `
        pub fun main() {
            let x: {Int8: String} = {}
            let y: {String: {Int8: String}} = {"a": {}}
        }
    `

	meter := newTestMemoryGauge()
	inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

	_, err := inter.Invoke("main")
	require.NoError(t, err)

	assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindString))
	assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindDictionary))
}

func TestRuntimeCompositeMetering(t *testing.T) {
	t.Parallel()

	script := `
        pub struct S {
        }

        pub resource R {
            pub let a: String
            pub let b: String

            init(a: String, b: String) {
                self.a = a
                self.b = b
            }
        }

        pub fun main() {
            let s = S()
            let r <- create R(a: "a", b: "b")
            destroy r
        }
    `

	meter := newTestMemoryGauge()
	inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

	_, err := inter.Invoke("main")
	require.NoError(t, err)

	assert.Equal(t, uint64(39), meter.getMemory(common.MemoryKindString))
	assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindComposite))
}

func TestRuntimeInterpretedFunctionMetering(t *testing.T) {
	t.Parallel()

	t.Run("top level function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInterpretedFunction))
	})

	t.Run("function pointer creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let funcPointer = fun(a: String): String {
                    return a
                }
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the main, and 1 for the anon-func
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunction))
	})

	t.Run("function pointer passing", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let funcPointer1 = fun(a: String): String {
                    return a
                }

                let funcPointer2 = funcPointer1
                let funcPointer3 = funcPointer2

                let value = funcPointer3("hello")
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the main, and 1 for the anon-func.
		// Assignment shouldn't allocate new memory, as the value is immutable and shouldn't be copied.
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunction))
	})

	t.Run("struct method", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                pub fun bar() {}
            }

            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the main, and 1 for the struct method.
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunction))
	})

	t.Run("struct init", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                init() {}
            }

            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the main, and 1 for the struct init.
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunction))
	})
}
