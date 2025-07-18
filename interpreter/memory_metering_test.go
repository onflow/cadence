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
	"fmt"
	"testing"

	"github.com/onflow/cadence/activations"
	. "github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

type assumeValidPublicKeyValidator struct{}

var _ stdlib.PublicKeyValidator = assumeValidPublicKeyValidator{}

func (assumeValidPublicKeyValidator) ValidatePublicKey(_ *stdlib.PublicKey) error {
	return nil
}

type testMemoryGauge struct {
	meter map[common.MemoryKind]uint64
}

func newTestMemoryGauge() *testMemoryGauge {
	return &testMemoryGauge{
		meter: make(map[common.MemoryKind]uint64),
	}
}

func (g *testMemoryGauge) MeterMemory(usage common.MemoryUsage) error {
	g.meter[usage.Kind] += usage.Amount
	return nil
}

func (g *testMemoryGauge) getMemory(kind common.MemoryKind) uint64 {
	return g.meter[kind]
}

func parseCheckAndPrepareWithMemoryMetering(
	t *testing.T,
	code string,
	gauge common.MemoryGauge,
) (Invokable, error) {
	return parseCheckAndPrepareWithOptions(
		t,
		code,
		ParseCheckAndInterpretOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				MemoryGauge: gauge,
			},
			InterpreterConfig: &interpreter.Config{
				MemoryGauge: gauge,
			},
		},
	)
}

func ifCompile[T any](compileValue, interpretValue T) T {
	if *compile {
		return compileValue
	}
	return interpretValue
}

func TestInterpretArrayMetering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int8] = []
              let y: [[String]] = [[]]
              let z: [[[Bool]]] = [[[]]]
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(25), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(20), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindVariable))
			// 1 Int8 for type
			// 2 String: 1 for type, 1 for value
			// 3 Bool: 1 for type, 2 for value
			assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindPrimitiveStaticType))
			assert.Equal(t, uint64(10), meter.getMemory(common.MemoryKindVariableSizedStaticType))
		}
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		const script = `
          fun main() {
              let values: [[Int128]] = [[], [], []]
              for value in values {
                  let a = value
              }
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(26), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(22), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindVariable))

			// 4 Int8: 1 for type, 3 for values
			assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindPrimitiveStaticType))
			assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindVariableSizedStaticType))
		}
	})

	t.Run("contains", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int128] = []
              x.contains(5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindPrimitiveStaticType))
	})

	t.Run("append with packing", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int8] = []
              x.append(3)
              x.append(4)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("append many", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int128] = [] // 2 data slabs
              x.append(0) // fits in existing slab
              x.append(1) // fits in existing slab
              x.append(2) // adds 1 data and metadata slab
              x.append(3) // fits in existing slab
              x.append(4) // adds 1 data slab
              x.append(5) // fits in existing slab
              x.append(6) // adds 1 data slab
              x.append(7) // fits in existing slab
              x.append(8) // adds 1 data slab
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("append very many", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              var i = 0;
              let x: [Int128] = [] // 2 data slabs
              while i < 120 { // should result in 4 meta data slabs and 60 slabs
                  x.append(0)
                  i = i + 1
              }
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(61), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(120), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("insert without packing", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int128] = []
              x.insert(at: 0, 3)
              x.insert(at: 1, 3)
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("insert with packing", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int8] = []
              x.insert(at: 0, 3)
              x.insert(at: 1, 3)
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))

		assert.Equal(t, ifCompile[uint64](10, 7), meter.getMemory(common.MemoryKindPrimitiveStaticType))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindVariableSizedStaticType))
		}
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int128] = [0, 1, 2, 3] // uses 2 data slabs and 1 metadata slab
              x[0] = 1 // adds 1 data and 1 metadata slab
              x[2] = 1  // adds 1 data and 1 metadata slab
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(10), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("update fits in slab", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int128] = [0, 1, 2] // uses 2 data slabs and 1 metadata slab
              x[0] = 1 // fits in existing slab
              x[2] = 1 // fits in existing slab
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("constant", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int8; 0] = []
              let y: [Int8; 1] = [2]
              let z: [Int8; 2] = [2, 4]
              let w: [[Int8; 2]] = [[2, 4]]
              let r: [[Int8; 2]] = [[2, 4], [8, 16]]
              let q: [[Int8; 2]; 2] = [[2, 4], [8, 16]]
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(37), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))

		// 2 for `q` type
		// 1 for each other type
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindConstantSizedType))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindConstantSizedStaticType))
		}
	})

	t.Run("insert many", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: [Int128] = [] // 2 data slabs
              x.insert(at:0, 3) // fits in existing slab
              x.insert(at:1, 3) // fits in existing slab
              x.insert(at:2, 3) // adds 1 metadata and data slab
              x.insert(at:3, 3) // fits in existing slab
              x.insert(at:4, 3) // adds 1 data slab
              x.insert(at:5, 3) // fits in existing slab
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))

		// 6 Int8 for types
		// 1 Int8 for `w` element
		// 2 Int8 for `r` elements
		// 2 Int8 for `q` elements
		assert.Equal(t, ifCompile[uint64](30, 19), meter.getMemory(common.MemoryKindPrimitiveStaticType))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindVariableSizedStaticType))
		}
	})
}

func TestInterpretDictionaryMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: {Int8: String} = {}
              let y: {String: {Int8: String}} = {"a": {}}
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(159), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
		assert.Equal(t, ifCompile[uint64](3, 9), meter.getMemory(common.MemoryKindPrimitiveStaticType))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindVariable))
			assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindDictionaryStaticType))
		}
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let values: [{Int8: String}] = [{}, {}, {}]
              for value in values {
                  let a = value
              }
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(480), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindVariable))

			// 4 Int8: 1 for type, 3 for values
			// 4 String: 1 for type, 3 for values
			assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindPrimitiveStaticType))
			assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindDictionaryStaticType))
		}

	})

	t.Run("contains", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: {Int8: String} = {}
              x.containsKey(5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](2, 3), meter.getMemory(common.MemoryKindPrimitiveStaticType))
	})

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: {Int8: String} = {}
              x.insert(key: 5, "")
              x.insert(key: 4, "")
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))

		assert.Equal(t, ifCompile[uint64](12, 10), meter.getMemory(common.MemoryKindPrimitiveStaticType))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindDictionaryStaticType))
		}
	})

	t.Run("insert many no packing", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: {Int8: String} = {} // 2 data slabs
              x.insert(key: 0, "") // fits in slab
              x.insert(key: 1, "") // fits in slab
              x.insert(key: 2, "") // adds 1 data and metadata slab
              x.insert(key: 3, "") // fits in slab
              x.insert(key: 4, "") // adds 1 data slab
              x.insert(key: 5, "") // fits in slab
              x.insert(key: 6, "") // adds 1 data slab
              x.insert(key: 7, "") // fits in slab
              x.insert(key: 8, "") // adds 1 data slab
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
	})

	t.Run("insert many with packing", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: {Int8: Int8} = {} // 2 data slabs
              x.insert(key: 0, 0) // all fit in slab
              x.insert(key: 1, 1)
              x.insert(key: 2, 2)
              x.insert(key: 3, 3)
              x.insert(key: 4, 4)
              x.insert(key: 5, 5)
              x.insert(key: 6, 6)
              x.insert(key: 7, 7)
              x.insert(key: 8, 8)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
	})

	t.Run("update fits in slab", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: {Int8: String} = {3: "a"} // 2 data slabs
              x[3] = "b" // fits in existing slab
              x[3] = "c" // fits in existing slab
              x[4] = "d" // fits in existing slab
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(31), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
	})

	t.Run("update does not fit in slab", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: {Int8: String} = {3: "a"} // 2 data slabs
              x[3] = "b" // fits in existing slab
              x[4] = "d" // fits in existing slab
              x[3] = "c" // adds 1 data slab and metadata slab
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
	})
}

func TestInterpretCompositeMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
          struct S {}

          resource R {
              let a: String
              let b: String

              init(a: String, b: String) {
                  self.a = a
                  self.b = b
              }
          }

          fun main() {
              let s = S()
              let r <- create R(a: "a", b: "b")
              destroy r
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindCompositeValueBase))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCompositeStaticType))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindCompositeTypeInfo))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindVariable))
			assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindInvocation))
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCompositeField))
		}
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		script := `
          struct S {}

          fun main() {
              let values = [S(), S(), S()]
              for value in values {
                  let a = value
              }
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindCompositeValueBase))
		assert.Equal(t, uint64(18), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(480), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))

		assert.Equal(t, ifCompile[uint64](6, 7), meter.getMemory(common.MemoryKindCompositeStaticType))
		assert.Equal(t, uint64(21), meter.getMemory(common.MemoryKindCompositeTypeInfo))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCompositeField))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindVariable))
			assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindInvocation))
		}
	})
}

func TestInterpretSimpleCompositeMetering(t *testing.T) {
	t.Parallel()

	t.Run("Account", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main(a: &Account) {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		address := common.MustBytesToAddress([]byte{0x1})

		account := stdlib.NewAccountReferenceValue(
			inter,
			nil,
			interpreter.AddressValue(address),
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		_, err = inter.Invoke("main", account)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindSimpleCompositeValueBase))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindSimpleCompositeValue))
	})
}

func TestInterpretCompositeFieldMetering(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		script := `
            struct S {}

            fun main() {
                let s = S()
            }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCompositeValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCompositeField))
	})

	t.Run("1 field", func(t *testing.T) {
		t.Parallel()

		script := `
          struct S {
              let a: String

              init(_ a: String) {
                  self.a = a
              }
          }

          fun main() {
              let s = S("a")
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCompositeValueBase))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCompositeField))
	})

	t.Run("2 field", func(t *testing.T) {
		t.Parallel()

		script := `
          struct S {
              let a: String
              let b: String

              init(_ a: String, _ b: String) {
                  self.a = a
                  self.b = b
              }
          }

          fun main() {
              let s = S("a", "b")
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCompositeValueBase))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCompositeField))
	})
}

func TestInterpretInterpretedFunctionMetering(t *testing.T) {
	t.Parallel()

	t.Run("top level function", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInterpretedFunctionValue))
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInvocation))
		}
	})

	t.Run("function pointer creation", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let funcPointer = fun(a: String): String {
                  return a
              }
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			// 1 for the main, and 1 for the anon-func
			assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunctionValue))
		}
	})

	t.Run("function pointer passing", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let funcPointer1 = fun(a: String): String {
                  return a
              }

              let funcPointer2 = funcPointer1
              let funcPointer3 = funcPointer2

              let value = funcPointer3("hello")
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			// 1 for the main, and 1 for the anon-func.
			// Assignment shouldn't allocate new memory, as the value is immutable and shouldn't be copied.
			assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunctionValue))

			assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInvocation))
		}
	})

	t.Run("struct method", func(t *testing.T) {
		t.Parallel()

		script := `
          struct Foo {
              fun bar() {}
          }

          fun main() {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			// 1 for the main, and 1 for the struct method.
			assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunctionValue))
		}
	})

	t.Run("struct init", func(t *testing.T) {
		t.Parallel()

		script := `
          struct Foo {
              init() {}
          }

          fun main() {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			// 1 for the main, and 1 for the struct init.
			assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunctionValue))

			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInvocation))
		}
	})
}

func TestInterpretHostFunctionMetering(t *testing.T) {
	t.Parallel()

	// HostFunctionValue is only used in the interpreter, not the compiler/VM.
	if *compile {
		return
	}

	t.Run("top level function", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunctionValue))
	})

	t.Run("function pointers", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let funcPointer1 = fun(a: String): String {
                  return a
              }

              let funcPointer2 = funcPointer1
              let funcPointer3 = funcPointer2

              let value = funcPointer3("hello")
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunctionValue))
	})

	t.Run("struct method", func(t *testing.T) {
		t.Parallel()

		script := `
          struct Foo {
              fun bar() {}
          }

          fun main() {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the struct method.
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindHostFunctionValue))
	})

	t.Run("struct init", func(t *testing.T) {
		t.Parallel()

		script := `
          struct Foo {
              init() {}
          }

          fun main() {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the struct init.
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindHostFunctionValue))
	})

	t.Run("builtin functions", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let a = Int8(5)

              let b = CompositeType("PublicKey")
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// builtin functions are not metered
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunctionValue))

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCompositeStaticType))
	})

	t.Run("stdlib function", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              assert(true)
          }
        `

		meter := newTestMemoryGauge()

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		for _, valueDeclaration := range stdlib.InterpreterDefaultStandardLibraryValues(nil) {
			baseValueActivation.DeclareValue(valueDeclaration)
			interpreter.Declare(baseActivation, valueDeclaration)
		}

		inter, err := parseCheckAndPrepareWithOptions(
			t,
			script,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					MemoryGauge: meter,
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					MemoryGauge: meter,
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// stdlib functions are not metered
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunctionValue))
	})

	t.Run("public key creation", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let publicKey = PublicKey(
                  publicKey: "0102".decodeHex(),
                  signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
              )
          }
        `

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		for _, valueDeclaration := range []stdlib.StandardLibraryValue{
			stdlib.NewInterpreterPublicKeyConstructor(
				assumeValidPublicKeyValidator{},
			),
			stdlib.InterpreterSignatureAlgorithmConstructor,
		} {
			baseValueActivation.DeclareValue(valueDeclaration)
			interpreter.Declare(baseActivation, valueDeclaration)
		}

		meter := newTestMemoryGauge()

		inter, err := parseCheckAndPrepareWithOptions(
			t,
			script,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					MemoryGauge: meter,
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					MemoryGauge: meter,
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 1 host function created for 'decodeHex' of String value
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindHostFunctionValue))
	})

	t.Run("multiple public key creation", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let publicKey1 = PublicKey(
                  publicKey: "0102".decodeHex(),
                  signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
              )

              let publicKey2 = PublicKey(
                  publicKey: "0102".decodeHex(),
                  signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
              )
          }
        `

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		for _, valueDeclaration := range []stdlib.StandardLibraryValue{
			stdlib.NewInterpreterPublicKeyConstructor(
				assumeValidPublicKeyValidator{},
			),
			stdlib.InterpreterSignatureAlgorithmConstructor,
		} {
			baseValueActivation.DeclareValue(valueDeclaration)
			interpreter.Declare(baseActivation, valueDeclaration)
		}

		meter := newTestMemoryGauge()

		inter, err := parseCheckAndPrepareWithOptions(
			t,
			script,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					MemoryGauge: meter,
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					MemoryGauge: meter,
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 2 = 2x 1 host function created for 'decodeHex' of String value
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindHostFunctionValue))
	})
}

func TestInterpretBoundFunctionMetering(t *testing.T) {
	t.Parallel()

	// BoundFunctionValue is only used in the interpreter, not the compiler/VM.
	if *compile {
		return
	}

	t.Run("struct method", func(t *testing.T) {
		t.Parallel()

		script := `
          struct Foo {
              fun bar() {}
          }

          fun main() {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// No bound functions are created without usages.
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoundFunctionValue))
	})

	t.Run("struct init", func(t *testing.T) {
		t.Parallel()

		script := `
          struct Foo {
              init() {}
          }

          fun main() {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// No bound functions are created without usages.
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoundFunctionValue))
	})

	t.Run("struct method usage", func(t *testing.T) {
		t.Parallel()

		script := `
          struct Foo {
              fun bar() {}
          }

          fun main() {
              let foo = Foo()
              foo.bar()
              foo.bar()
              foo.bar()
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 3 bound functions are created for the 3 invocations of 'bar()'.
		// No bound functions are created for init invocation.
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindBoundFunctionValue))
	})
}

func TestInterpretOptionalValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("simple optional value", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: String? = "hello"
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindOptionalValue))
	})

	t.Run("dictionary get", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: {Int8: String} = {1: "foo", 2: "bar"}
              let y = x[0]
              let z = x[1]
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 2 for `z`
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindOptionalValue))

		assert.Equal(t, ifCompile[uint64](20, 14), meter.getMemory(common.MemoryKindPrimitiveStaticType))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindDictionaryStaticType))
		}
	})

	t.Run("dictionary set", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: {Int8: String} = {1: "foo", 2: "bar"}
              x[0] = "a"
              x[1] = "b"
          }
        `

		meter := newTestMemoryGauge()

		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 1 from creating new entry by setting x[0]
		// 2 from overwriting entry by setting x[1]
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindOptionalValue))
	})

	t.Run("OptionalType", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let type: Type = Type<Int>()
              let a = OptionalType(type)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 0: optional type is created here, not an optional value
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindOptionalValue))
		// 1: `a`
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindOptionalStaticType))
	})
}

func TestInterpretIntMetering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](16, 8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 + 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](80, 64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 - 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 * 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](80, 64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 / 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 % 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 | 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 ^ 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 & 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 << 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](80, 64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 >> 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1
              let y = -x
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](56, 48), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretUIntMetering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, ifCompile[uint64](16, 8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt + 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](80, 64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as UInt - 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt).saturatingSubtract(2 as UInt)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt * 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](80, 64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt / 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt % 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt | 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt ^ 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt & 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt << 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](80, 64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt >> 2 as UInt
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](72, 56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1
              let y = -x
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](56, 48), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretUInt8Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt8 + 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt8).saturatingAdd(2 as UInt8)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as UInt8 - 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt8).saturatingSubtract(2 as UInt8)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt8 * 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt8).saturatingMultiply(2 as UInt8)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt8 / 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt8 % 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt8 | 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt8 ^ 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt8 & 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt8 << 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt8 >> 2 as UInt8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretUInt16Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt16 + 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt16).saturatingAdd(2 as UInt16)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as UInt16 - 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt16).saturatingSubtract(2 as UInt16)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt16 * 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt16).saturatingMultiply(2 as UInt16)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt16 / 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt16 % 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt16 | 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt16 ^ 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt16 & 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt16 << 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt16 >> 2 as UInt16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretUInt32Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt32 + 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt32).saturatingAdd(2 as UInt32)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as UInt32 - 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt32).saturatingSubtract(2 as UInt32)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt32 * 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt32).saturatingMultiply(2 as UInt32)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt32 / 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt32 % 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt32 | 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt32 ^ 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt32 & 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt32 << 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt32 >> 2 as UInt32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretUInt64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt64 + 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt64).saturatingAdd(2 as UInt64)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as UInt64 - 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt64).saturatingSubtract(2 as UInt64)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt64 * 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt64).saturatingMultiply(2 as UInt64)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt64 / 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt64 % 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt64 | 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt64 ^ 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt64 & 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt64 << 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt64 >> 2 as UInt64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretUInt128Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, ifCompile[uint64](24, 8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt128 + 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt128).saturatingAdd(2 as UInt128)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as UInt128 - 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt128).saturatingSubtract(2 as UInt128)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt128 * 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt128).saturatingMultiply(2 as UInt128)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt128 / 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt128 % 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt128 | 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt128 ^ 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt128 & 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt128 << 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8 + 16
		// result: 16
		assert.Equal(t, ifCompile[uint64](80, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt128 >> 2 as UInt128
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretUInt256Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, ifCompile[uint64](40, 8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt256 + 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt256).saturatingAdd(2 as UInt256)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as UInt256 - 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt256).saturatingSubtract(2 as UInt256)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as UInt256 * 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = (1 as UInt256).saturatingMultiply(2 as UInt256)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt256 / 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
            let x = 10 as UInt256 % 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt256 | 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt256 ^ 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt256 & 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt256 << 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8 + 32
		// result: 32
		assert.Equal(t, ifCompile[uint64](144, 80), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as UInt256 >> 2 as UInt256
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretInt8Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 1
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 1 + 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 1
              let y: Int8 = x.saturatingAdd(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 1 - 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 1
              let y: Int8 = x.saturatingSubtract(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 1 * 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 1
              let y: Int8 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 3 / 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 3
              let y: Int8 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 3 % 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 1
              let y: Int8 = -x
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// x: 1
		// y: 1
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 3 | 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 3 ^ 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 3 & 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 3 << 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int8 = 3 >> 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

}

func TestInterpretInt16Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 1
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 1 + 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 1
              let y: Int16 = x.saturatingAdd(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 1 - 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 1
              let y: Int16 = x.saturatingSubtract(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 1 * 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 1
              let y: Int16 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 3 / 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 3
              let y: Int16 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 3 % 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 1
              let y: Int16 = -x
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// x: 2
		// y: 2
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 3 | 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 3 ^ 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 3 & 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 3 << 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int16 = 3 >> 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretInt32Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 1
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 1 + 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 1
              let y: Int32 = x.saturatingAdd(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 1 - 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 1
              let y: Int32 = x.saturatingSubtract(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 1 * 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 1
              let y: Int32 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 3 / 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 3
              let y: Int32 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 3 % 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 1
              let y: Int32 = -x
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// x: 4
		// y: 4
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 3 | 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 3 ^ 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 3 & 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 3 << 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int32 = 3 >> 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretInt64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 1
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 1 + 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 1
              let y: Int64 = x.saturatingAdd(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 1 - 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 1
              let y: Int64 = x.saturatingSubtract(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 1 * 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 1
              let y: Int64 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 3 / 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 3
              let y: Int64 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 3 % 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 1
              let y: Int64 = -x
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// x: 8
		// y: 8
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 3 | 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 3 ^ 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 3 & 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 3 << 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int64 = 3 >> 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretInt128Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 1
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](24, 8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 1 + 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 1
              let y: Int128 = x.saturatingAdd(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 1 - 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 1
              let y: Int128 = x.saturatingSubtract(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 1 * 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 1
              let y: Int128 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 3 / 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 3
              let y: Int128 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 3 % 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 1
              let y: Int128 = -x
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// x: 16
		// y: 16
		assert.Equal(t, ifCompile[uint64](40, 24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 3 | 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 3 ^ 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 3 & 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 3 << 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8 + 16
		// result: 16
		assert.Equal(t, ifCompile[uint64](80, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 3 >> 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 16
		assert.Equal(t, ifCompile[uint64](64, 32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int128 = 1
              x == 1
              x != 1
              x > 1
              x >= 1
              x < 1
              x <= 1
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))
	})
}

func TestInterpretInt256Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 1
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](40, 8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 1 + 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 1
              let y: Int256 = x.saturatingAdd(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 1 - 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 1
              let y: Int256 = x.saturatingSubtract(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 1 * 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 1
              let y: Int256 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 3 / 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 3
              let y: Int256 = x.saturatingMultiply(2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 3 % 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 1
              let y: Int256 = -x
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// x: 32
		// y: 32
		assert.Equal(t, ifCompile[uint64](72, 40), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 3 | 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 3 ^ 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 3 & 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 3 << 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8 + 32
		// result: 32
		assert.Equal(t, ifCompile[uint64](144, 80), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Int256 = 3 >> 2
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 32
		assert.Equal(t, ifCompile[uint64](112, 48), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretWord8Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word8 + 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as Word8 - 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word8 * 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word8 / 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word8 % 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word8 | 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word8 ^ 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word8 & 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word8 << 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word8 >> 2 as Word8
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretWord16Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word16 + 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as Word16 - 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word16 * 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word16 / 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word16 % 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word16 | 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word16 ^ 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word16 & 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word16 << 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word16 >> 2 as Word16
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretWord32Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word32 + 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as Word32 - 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word32 * 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word32 / 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word32 % 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word32 | 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word32 ^ 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word32 & 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word32 << 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word32 >> 2 as Word32
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretWord64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word64 + 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 3 as Word64 - 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 1 as Word64 * 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word64 / 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word64 % 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word64 | 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word64 ^ 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word64 & 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word64 << 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = 10 as Word64 >> 2 as Word64
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})
}

func TestInterpretStorageReferenceValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            resource R {}

            fun main(account: auth(Storage) &Account) {
                account.storage.borrow<&R>(from: /storage/r)
            }
          `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		address := common.MustBytesToAddress([]byte{0x1})
		authorization := interpreter.NewEntitlementSetAuthorization(
			meter,
			func() []common.TypeID {
				return []common.TypeID{
					sema.StorageType.ID(),
				}
			},
			1,
			sema.Conjunction,
		)
		account := stdlib.NewAccountReferenceValue(
			inter,
			nil,
			interpreter.AddressValue(address),
			authorization,
			interpreter.EmptyLocationRange,
		)

		_, err = inter.Invoke("main", account)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindStorageReferenceValue))
	})
}

func TestInterpretEphemeralReferenceValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
          resource R {}

          fun main(): &Int {
            let x: Int = 1
            let y = &x as &Int
            return y
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindEphemeralReferenceValue))
	})

	t.Run("creation, optional", func(t *testing.T) {
		t.Parallel()

		script := `
          resource R {}

          fun main(): &Int {
            let x: Int? = 1
            let y = &x as &Int?
            return y!
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindEphemeralReferenceValue))
	})
}

func TestInterpretStringMetering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = "a"
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("assignment", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = "a"
              let y = x
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("Unicode", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = "İ"
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("toLower, ASCII", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = "ABC".toLower()
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 1 + 3 (abc)
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("toLower, Unicode", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x = "İ".toLower()
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 1 + 4 (max UTF8 encoding)
		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindStringValue))
	})
}

func TestInterpretCharacterMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: Character = "a"
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCharacterValue))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("assignment", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: Character = "a"
              let y = x
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCharacterValue))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("from string GetKey", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: String = "a"
              let y: Character = x[0]
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCharacterValue))
	})
}

func TestInterpretAddressValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: Address = 0x0
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddressValue))
	})

	t.Run("convert", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x = Address(0x0)
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddressValue))
	})
}

func TestInterpretPathValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x = /public/bar
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPathValue))
	})

	t.Run("convert", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x = PublicPath(identifier: "bar")
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPathValue))
	})
}

func TestInterpretCapabilityValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		meter := newTestMemoryGauge()

		_ = interpreter.NewCapabilityValue(meter, 1, interpreter.AddressValue{}, nil)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCapabilityValue))
	})
}

func TestInterpretTypeValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("static constructor", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let t: Type = Type<Int>()
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindTypeValue))
	})

	t.Run("dynamic constructor", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let t: Type = ConstantSizedArrayType(type: Type<Int>(), size: 2)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindTypeValue))
	})

	t.Run("getType", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let v = 5
              let t: Type = v.getType()
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindTypeValue))
	})
}

func TestInterpretVariableMetering(t *testing.T) {
	t.Parallel()

	t.Run("globals", func(t *testing.T) {
		t.Parallel()

		script := `
          var a = 3
          let b = false

          fun main() {

          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindVariable))
		}
	})

	t.Run("params", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main(a: String, b: Bool) {

          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke(
			"main",
			interpreter.NewUnmeteredStringValue(""),
			interpreter.FalseValue,
		)
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindVariable))
		}
	})

	t.Run("nested params", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              var x = fun (x: String, y: Bool) {}
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindVariable))
		}
	})

	t.Run("applied nested params", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              var x = fun (x: String, y: Bool) {}
              x("", false)
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindVariable))
		}
	})
}

func TestInterpretFix64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 1.4
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 1.4 + 2.5
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 1.4
              let y: Fix64 = x.saturatingAdd(2.5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 1.4 - 2.5
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 1.4
              let y: Fix64 = x.saturatingSubtract(2.5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 1.4 * 2.5
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 1.4
              let y: Fix64 = x.saturatingMultiply(2.5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 3.4 / 2.5
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 3.4
              let y: Fix64 = x.saturatingMultiply(2.5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 3.4 % 2.5
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// quotient (div) : 8
		// truncatedQuotient: 8
		// truncatedQuotient.Mul(o): 8
		// result: 8
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 1.4
              let y: Fix64 = -x
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// x: 8
		// y: 8
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("creation as supertype", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: FixedPoint = -1.4
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: Fix64 = 1.0
              x == 1.0
              x != 1.0
              x > 1.0
              x >= 1.0
              x < 1.0
              x <= 1.0
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(224), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretUFix64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 1.4
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 1.4 + 2.5
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 1.4
              let y: UFix64 = x.saturatingAdd(2.5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 2.5 - 1.4
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 1.4
              let y: UFix64 = x.saturatingSubtract(2.5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 1.4 * 2.5
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 1.4
              let y: UFix64 = x.saturatingMultiply(2.5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 3.4 / 2.5
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 3.4
              let y: UFix64 = x.saturatingMultiply(2.5)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 3.4 % 2.5
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// quotient (div) : 8
		// truncatedQuotient: 8
		// truncatedQuotient.Mul(o): 8
		// result: 8
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("creation as supertype", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: FixedPoint = 1.4
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
          fun main() {
              let x: UFix64 = 1.0
              x == 1.0
              x != 1.0
              x > 1.0
              x >= 1.0
              x < 1.0
              x <= 1.0
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(224), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretTokenMetering(t *testing.T) {
	t.Parallel()

	t.Run("identifier tokens", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              var x: String = "hello"
          }

          struct foo {
              var x: Int

              init() {
                  self.x = 4
              }
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(30), meter.getMemory(common.MemoryKindTypeToken))
		assert.Equal(t, uint64(23), meter.getMemory(common.MemoryKindSpaceToken))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
	})

	t.Run("syntax tokens", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              var a: [String] = []
              var b = 4 + 6
              var c = true && false != false
              var d = 4 as! AnyStruct
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(35), meter.getMemory(common.MemoryKindTypeToken))
		assert.Equal(t, uint64(30), meter.getMemory(common.MemoryKindSpaceToken))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
	})

	t.Run("comments", func(t *testing.T) {
		t.Parallel()

		script := `
          /*  first line
              second line
          */

          // single line comment
          fun main() {}
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(10), meter.getMemory(common.MemoryKindTypeToken))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindSpaceToken))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
	})

	t.Run("numeric literals", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              var a = 1
              var b = 0b1
              var c = 0o1
              var d = 0x1
              var e = 1.4
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(26), meter.getMemory(common.MemoryKindTypeToken))
		assert.Equal(t, uint64(25), meter.getMemory(common.MemoryKindSpaceToken))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
	})
}

func TestInterpreterStringLocationMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		// Raw string count with empty location

		script := `
          struct S {}

          fun main() {
              let s = CompositeType("")
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		emptyLocationStringCount := meter.getMemory(common.MemoryKindRawString)

		// Raw string count with non-empty location

		script = `
          struct S {}

          fun main() {
              let s = CompositeType("S.test.S")
          }
        `

		meter = newTestMemoryGauge()
		inter, err = parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		testLocationStringCount := meter.getMemory(common.MemoryKindRawString)

		// raw string location is "test" + locationIDs
		assert.Equal(t, uint64(5), testLocationStringCount-emptyLocationStringCount)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCompositeStaticType))

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCompositeStaticType))
	})
}

func TestInterpretIdentifierMetering(t *testing.T) {
	t.Parallel()

	t.Run("variable", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let foo = 4
              let bar = 5
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 'main', 'foo', 'bar', empty-return-type
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindIdentifier))
	})

	t.Run("parameters", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main(foo: String, bar: String) {
          }
        `
		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke(
			"main",
			interpreter.NewUnmeteredStringValue("x"),
			interpreter.NewUnmeteredStringValue("y"),
		)
		require.NoError(t, err)

		// 'main', 'foo', 'String', 'bar', 'String', empty-return-type
		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindIdentifier))
	})

	t.Run("composite declaration", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {}

          struct foo {
              var x: String
              var y: String

              init() {
                  self.x = "a"
                  self.y = "b"
              }

              fun bar() {}
            }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(15), meter.getMemory(common.MemoryKindIdentifier))
	})

	t.Run("member resolvers", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {            // 2 - 'main', empty-return-type
              let foo = ["a", "b"]    // 1
              foo.length              // 3 - 'foo', 'length', constant field resolver
              foo.length              // 3 - 'foo', 'length', constant field resolver (not re-used)
              foo.removeFirst()       // 3 - 'foo', 'removeFirst', function resolver
              foo.removeFirst()       // 3 - 'foo', 'removeFirst', function resolver (not re-used)
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(14), meter.getMemory(common.MemoryKindIdentifier))
		assert.Equal(t, ifCompile[uint64](4, 3), meter.getMemory(common.MemoryKindPrimitiveStaticType))
	})
}

func TestInterpretInterfaceStaticType(t *testing.T) {
	t.Parallel()

	t.Run("IntersectionType", func(t *testing.T) {
		t.Parallel()

		script := `
          struct interface I {}

          fun main() {
              let type = Type<{I}>()

              IntersectionType(
                  types: [type.identifier]
              )
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInterfaceStaticType))
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindIntersectionStaticType))
		}
	})
}

func TestInterpretFunctionStaticType(t *testing.T) {
	t.Parallel()

	t.Run("FunctionType", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              FunctionType(parameters: [], return: Type<Never>())
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindFunctionStaticType))
	})

	t.Run("array element", func(t *testing.T) {
		t.Parallel()

		script := `
          fun hello() {}

          fun main() {
              let a = [hello]
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindFunctionStaticType))
		}
	})

	t.Run("set bound function to variable", func(t *testing.T) {
		t.Parallel()

		script := `
          struct S {
              fun naught() {}
          }

          fun main() {
              let x = S()
              let y = x.naught
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](2, 1), meter.getMemory(common.MemoryKindFunctionStaticType))
	})

	t.Run("isInstance", func(t *testing.T) {
		t.Parallel()

		script := `
          struct S {
              fun naught() {}
          }

          fun main() {
              let x = S()
              x.naught.isInstance(Type<Int>())
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(
			t,
			ifCompile[uint64](2, 3),
			meter.getMemory(common.MemoryKindFunctionStaticType),
		)
	})
}

func TestInterpretVariableActivationMetering(t *testing.T) {
	t.Parallel()

	t.Run("single function", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](1, 3), meter.getMemory(common.MemoryKindActivation))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindActivationEntries))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInvocation))
		}
	})

	t.Run("nested function call", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              foo(a: "hello", b: 23)
          }

          fun foo(a: String, b: Int) {
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](1, 5), meter.getMemory(common.MemoryKindActivation))
		assert.Equal(t, ifCompile[uint64](1, 2), meter.getMemory(common.MemoryKindActivationEntries))

		// TODO: assert equivalent for compiler/VM
		if !*compile {
			assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInvocation))
		}
	})

	t.Run("local scope", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              if true {
                  let a = 1
              }
          }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, ifCompile[uint64](1, 4), meter.getMemory(common.MemoryKindActivation))
		assert.Equal(t, ifCompile[uint64](1, 2), meter.getMemory(common.MemoryKindActivationEntries))
	})
}

func TestInterpretStaticTypeConversionMetering(t *testing.T) {
	t.Parallel()

	t.Run("primitive static types", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let a: {Int: {Foo}} = {}           // dictionary + intersection
              let b: [&Int] = []                          // variable-sized + reference
              let c: [Int?; 2] = [1, 2]                   // constant-sized + optional
              let d: [Capability<&Bar>] = []             //  capability + variable-sized + reference
          }

          struct interface Foo {}

          struct Bar: Foo {}
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindDictionarySemaType))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindVariableSizedSemaType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindConstantSizedSemaType))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindIntersectionSemaType))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindReferenceSemaType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCapabilitySemaType))
		// TODO: investigate why this is different for the compiler/VM
		assert.Equal(t, ifCompile[uint64](3, 2), meter.getMemory(common.MemoryKindOptionalSemaType))
	})
}

func TestInterpretStorageMapMetering(t *testing.T) {
	t.Parallel()

	script := `
      resource R {}

      fun main(account: auth(Storage) &Account) {
          let r <- create R()
          account.storage.save(<-r, to: /storage/r)
      }
    `

	meter := newTestMemoryGauge()
	inter, err := parseCheckAndPrepareWithMemoryMetering(t, script, meter)
	require.NoError(t, err)

	address := interpreter.AddressValue(common.MustBytesToAddress([]byte{0x1}))
	authorization := interpreter.NewEntitlementSetAuthorization(
		meter,
		func() []common.TypeID {
			return []common.TypeID{
				sema.StorageType.ID(),
			}
		},
		1,
		sema.Conjunction,
	)
	account := stdlib.NewAccountReferenceValue(
		inter,
		nil,
		address,
		authorization,
		interpreter.EmptyLocationRange,
	)

	_, err = inter.Invoke("main", account)
	require.NoError(t, err)

	assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindStorageMap))
	assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindStorageKey))
}

func TestInterpretValueStringConversion(t *testing.T) {
	t.Parallel()

	testValueStringConversion := func(t *testing.T, script string, args ...interpreter.Value) {
		meter := newTestMemoryGauge()

		var loggedString string

		logFunction := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"log",
			&sema.FunctionType{
				Parameters: []sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
			``,
			func(invocation interpreter.Invocation) interpreter.Value {
				// Reset gauge, to only capture the values metered during string conversion
				meter.meter = make(map[common.MemoryKind]uint64)

				loggedString = invocation.Arguments[0].MeteredString(
					invocation.InvocationContext,
					interpreter.SeenReferences{},
					invocation.LocationRange,
				)
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(logFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, logFunction)

		inter, err := parseCheckAndPrepareWithOptions(
			t,
			script,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					MemoryGauge: meter,
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
				ParseAndCheckOptions: &ParseAndCheckOptions{
					MemoryGauge: meter,
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main", args...)
		require.NoError(t, err)

		meteredAmount := meter.getMemory(common.MemoryKindRawString)

		// Metered amount must be an overestimation compared to the actual logged string.
		assert.GreaterOrEqual(t, int(meteredAmount), len(loggedString))
	}

	t.Run("Simple values", func(t *testing.T) {
		t.Parallel()

		type testCase struct {
			name        string
			constructor string
		}

		testCases := []testCase{
			{
				name:        "Int",
				constructor: "3",
			},
			{
				name:        "Int8",
				constructor: "Int8(3)",
			},
			{
				name:        "Int16",
				constructor: "Int16(3)",
			},
			{
				name:        "Int32",
				constructor: "Int32(3)",
			},
			{
				name:        "Int64",
				constructor: "Int64(3)",
			},
			{
				name:        "Int128",
				constructor: "Int128(3)",
			},
			{
				name:        "Int256",
				constructor: "Int256(3)",
			},
			{
				name:        "UInt",
				constructor: "3",
			},
			{
				name:        "UInt8",
				constructor: "UInt8(3)",
			},
			{
				name:        "UInt16",
				constructor: "UInt16(3)",
			},
			{
				name:        "UInt32",
				constructor: "UInt32(3)",
			},
			{
				name:        "UInt64",
				constructor: "UInt64(3)",
			},
			{
				name:        "UInt128",
				constructor: "UInt128(3)",
			},
			{
				name:        "UInt256",
				constructor: "UInt256(3)",
			},
			{
				name:        "Word8",
				constructor: "Word8(3)",
			},
			{
				name:        "Word16",
				constructor: "Word16(3)",
			},
			{
				name:        "Word32",
				constructor: "Word32(3)",
			},
			{
				name:        "Word64",
				constructor: "Word64(3)",
			},
			{
				name:        "Fix64",
				constructor: "Fix64(3.45)",
			},
			{
				name:        "UFix64",
				constructor: "UFix64(3.45)",
			},
			{
				name:        "String",
				constructor: "\"hello\"",
			},
			{
				name:        "Escaped String",
				constructor: "\"hello\tworld!\t\"",
			},
			{
				name:        "Unicode String",
				constructor: "\"\\u{75}\\u{308}\" as String",
			},
			{
				name:        "Bool",
				constructor: "false",
			},
			{
				name:        "Nil",
				constructor: "nil",
			},
			{
				name:        "Address",
				constructor: "Address(0x1234)",
			},
			{
				name:        "Character",
				constructor: "\"c\" as Character",
			},
			{
				name:        "Escaped Character",
				constructor: "\"\t\" as Character",
			},
			{
				name:        "Unicode Character",
				constructor: "\"\\u{75}\\u{308}\" as Character",
			},
			{
				name:        "Array",
				constructor: "[1, 2, 3]",
			},
			{
				name:        "Dictionary",
				constructor: "{\"John\": \"Doe\", \"Country\": \"CA\"}",
			},
			{
				name:        "Path",
				constructor: "/public/somepath",
			},
			{
				name:        "Some",
				constructor: "true as Bool?",
			},
		}

		testSimpleValueStringConversion := func(test testCase) {

			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				script := fmt.Sprintf(`
                      fun main() {
                          let x = %s
                          log(x)
                      }
                    `,
					test.constructor,
				)

				testValueStringConversion(t, script)
			})
		}

		for _, test := range testCases {
			testSimpleValueStringConversion(test)
		}
	})

	t.Run("Composite", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x = Foo()
              log(x)
          }

          struct Foo {
              var a: Word8
              init() {
                  self.a = 4
              }
          }
        `

		testValueStringConversion(t, script)
	})

	t.Run("Ephemeral Reference", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x = 4
              log(&x as &AnyStruct)
          }
        `

		testValueStringConversion(t, script)
	})

	t.Run("Interpreted Function", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x = fun(a: String, b: Bool) {}
              log(&x as &AnyStruct)
          }
        `

		testValueStringConversion(t, script)
	})

	t.Run("Bound Function", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x = Foo()
              log(x.bar)
          }

          struct Foo {
              fun bar(a: String, b: Bool) {}
          }
        `

		testValueStringConversion(t, script)
	})

	t.Run("Void", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              let x: Void = foo()
              log(x)
          }

          fun foo() {}
        `

		testValueStringConversion(t, script)
	})

	t.Run("Capability", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main(a: Capability<&{Foo}>) {
              log(a)
          }

          struct interface Foo {}
          struct Bar: Foo {}
        `

		testValueStringConversion(t,
			script,
			interpreter.NewUnmeteredCapabilityValue(
				4,
				interpreter.AddressValue{1},
				interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "Bar"),
			))
	})

	t.Run("Type", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              log(Type<Int>())
          }
        `

		testValueStringConversion(t, script)
	})
}

func TestInterpretStaticTypeStringConversion(t *testing.T) {
	t.Parallel()

	testStaticTypeStringConversion := func(t *testing.T, script string) {
		meter := newTestMemoryGauge()

		var loggedString string

		logFunction := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"log",
			&sema.FunctionType{
				Parameters: []sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
			``,
			func(invocation interpreter.Invocation) interpreter.Value {
				// Reset gauge, to only capture the values metered during string conversion
				meter.meter = make(map[common.MemoryKind]uint64)

				loggedString = invocation.Arguments[0].MeteredString(
					invocation.InvocationContext,
					interpreter.SeenReferences{},
					invocation.LocationRange,
				)
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(logFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, logFunction)

		inter, err := parseCheckAndPrepareWithOptions(
			t,
			script,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					MemoryGauge: meter,
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					MemoryGauge: meter,
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		meteredAmount := meter.getMemory(common.MemoryKindRawString)

		// Metered amount must be an overestimation compared to the actual logged string.
		assert.GreaterOrEqual(t, int(meteredAmount), len(loggedString))
	}

	t.Run("Primitive static types", func(t *testing.T) {
		t.Parallel()

		for primitiveStaticType := range interpreter.PrimitiveStaticTypes {

			if !primitiveStaticType.IsDefined() || primitiveStaticType.IsDeprecated() { //nolint:staticcheck
				continue
			}

			switch primitiveStaticType {
			case interpreter.PrimitiveStaticTypeAny,
				interpreter.PrimitiveStaticTypeUnknown,
				interpreter.PrimitiveStaticType_Count:
				continue
			}

			semaType := primitiveStaticType.SemaType()

			switch semaType.(type) {
			case *sema.EntitlementType,
				*sema.EntitlementMapType:
				continue
			}

			script := fmt.Sprintf(
				`
                  fun main() {
                      log(Type<%s>())
                  }
                `,
				sema.NewTypeAnnotation(semaType).QualifiedString(),
			)

			testStaticTypeStringConversion(t, script)
		}
	})

	t.Run("Derived types", func(t *testing.T) {
		t.Parallel()

		type testCase struct {
			name        string
			constructor string
		}

		testCases := []testCase{
			{
				name:        "Variable-Sized",
				constructor: "[Int]",
			},
			{
				name:        "Fixed-Sized",
				constructor: "[Int;3]",
			},
			{
				name:        "Dictionary",
				constructor: "{String: Int}",
			},
			{
				name:        "Optional",
				constructor: "Bool?",
			},
			{
				name:        "Function",
				constructor: "fun(String): AnyStruct",
			},
			{
				name:        "Reference",
				constructor: "&Int",
			},
			{
				name:        "Auth Reference",
				constructor: "auth(X) &AnyStruct",
			},
			{
				name:        "Capability",
				constructor: "Capability<&AnyStruct>",
			},
		}

		testSimpleValueStringConversion := func(test testCase) {

			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				script := fmt.Sprintf(`
                    entitlement X
                    fun main() {
                        log(Type<%s>())
                    }
                `,
					test.constructor,
				)

				testStaticTypeStringConversion(t, script)
			})
		}

		for _, test := range testCases {
			testSimpleValueStringConversion(test)
		}
	})

	t.Run("Composite type", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              log(Type<Foo>())
          }

          struct Foo {
              var a: Word8
              init() {
                  self.a = 4
              }
          }
        `

		testStaticTypeStringConversion(t, script)
	})

	t.Run("Intersection type", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              log(Type<{Foo}>())
          }

          struct interface Foo {}
        `

		testStaticTypeStringConversion(t, script)
	})
}

func TestInterpretBytesMetering(t *testing.T) {

	t.Parallel()

	const code = `
      fun test(string: String) {
          let utf8 = string.utf8
      }
    `

	meter := newTestMemoryGauge()
	inter, err := parseCheckAndPrepareWithMemoryMetering(t, code, meter)
	require.NoError(t, err)

	stringValue := interpreter.NewUnmeteredStringValue("abc")

	_, err = inter.Invoke("test", stringValue)
	require.NoError(t, err)

	// 1 + 3
	assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindBytes))
}
