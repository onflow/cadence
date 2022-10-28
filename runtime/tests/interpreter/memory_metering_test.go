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
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/onflow/cadence/runtime/activations"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/utils"
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

func TestInterpretArrayMetering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {
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

		assert.Equal(t, uint64(25), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(25), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindVariable))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))
		// 1 Int8 for type
		// 2 String: 1 for type, 1 for value
		// 3 Bool: 1 for type, 2 for value
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindPrimitiveStaticType))
		// 0 for `x`
		// 1 for `y`
		// 4 for `z`
		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindVariableSizedStaticType))
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		script := `
    pub fun main() {
        let values: [[Int128]] = [[], [], []]
        for value in values {
        let a = value
        }
    }
`

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(30), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(33), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindVariable))

		// 4 Int8: 1 for type, 3 for values
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindPrimitiveStaticType))
		// 3: 1 for each [] in `values`
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindVariableSizedStaticType))
	})

	t.Run("contains", func(t *testing.T) {
		t.Parallel()

		script := `
        pub fun main() {
                let x: [Int128] = []
                x.contains(5)
        }
`

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindPrimitiveStaticType))
	})

	t.Run("append with packing", func(t *testing.T) {
		t.Parallel()

		script := `
        pub fun main() {
                let x: [Int8] = []
                x.append(3)
                x.append(4)
        }
`

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("append many", func(t *testing.T) {
		t.Parallel()

		script := `
        pub fun main() {
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
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("append very many", func(t *testing.T) {
		t.Parallel()

		script := `
        pub fun main() {
				var i = 0;
                let x: [Int128] = [] // 2 data slabs
                while i < 120 { // should result in 4 meta data slabs and 60 slabs
					x.append(0)
					i = i + 1
				}
        }
`

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(61), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(120), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("insert without packing", func(t *testing.T) {
		t.Parallel()

		script := `
        pub fun main() {
                let x: [Int128] = []
                x.insert(at:0, 3)
                x.insert(at:1, 3)
        }
`
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("insert with packing", func(t *testing.T) {
		t.Parallel()

		script := `
                pub fun main() {
                let x: [Int8] = []
                x.insert(at:0, 3)
                x.insert(at:1, 3)
                }
`
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindPrimitiveStaticType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindVariableSizedStaticType))
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		script := `
    pub fun main() {
        let x: [Int128] = [0, 1, 2, 3] // uses 2 data slabs and 1 metadata slab
        x[0] = 1 // adds 1 data and 1 metadata slab 
        x[2] = 1  // adds 1 data and 1 metadata slab 
    }
`
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(10), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("update fits in slab", func(t *testing.T) {
		t.Parallel()

		script := `
                pub fun main() {
                        let x: [Int128] = [0, 1, 2] // uses 2 data slabs and 1 metadata slab
                        x[0] = 1 // fits in existing slab
                        x[2] = 1 // fits in existing slab
                }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))
	})

	t.Run("constant", func(t *testing.T) {
		t.Parallel()

		script := `
    pub fun main() {
        let x: [Int8; 0] = []
        let y: [Int8; 1] = [2]
        let z: [Int8; 2] = [2, 4]
        let w: [[Int8; 2]] = [[2, 4]]
        let r: [[Int8; 2]] = [[2, 4], [8, 16]]
        let q: [[Int8; 2]; 2] = [[2, 4], [8, 16]]
    }
`
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(37), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(37), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(66), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))

		// 1 for `w`: 1 for the element
		// 2 for `r`: 1 for each element
		// 2 for `q`: 1 for each element
		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindConstantSizedStaticType))
		// 2 for `q` type
		// 1 for each other type
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindConstantSizedType))
	})

	t.Run("insert many", func(t *testing.T) {
		t.Parallel()

		script := `
    pub fun main() {
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
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindArrayValueBase))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeArrayDataSlab))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeArrayMetaDataSlab))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindAtreeArrayElementOverhead))

		// 6 Int8 for types
		// 1 Int8 for `w` element
		// 2 Int8 for `r` elements
		// 2 Int8 for `q` elements
		assert.Equal(t, uint64(19), meter.getMemory(common.MemoryKindPrimitiveStaticType))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindVariableSizedStaticType))
	})
}

func TestInterpretDictionaryMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
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

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindStringValue))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(159), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindVariable))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindPrimitiveStaticType))
		// 1 for `x`
		// 7 for `y`: 2 for type, 5 for value
		//   Note that the number of static types allocated raises 1 with each value.
		//   1, 2, 3, ... elements each use 5, 6, 7, ... static types.
		//   This is cumulative so 3 elements allocate 5+6+7=18 static types.
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindDictionaryStaticType))
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		script := `
	            pub fun main() {
	                let values: [{Int8: String}] = [{}, {}, {}]
	                for value in values {
	                  let a = value
	                }
	            }
	        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(27), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindVariable))

		// 4 Int8: 1 for type, 3 for values
		// 4 String: 1 for type, 3 for values
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindPrimitiveStaticType))
		// 1 for type
		// 6: 2 for each element
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindDictionaryStaticType))

		assert.Equal(t, uint64(480), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
	})

	t.Run("contains", func(t *testing.T) {
		t.Parallel()

		script := `
	            pub fun main() {
	                let x: {Int8: String} = {}
	                x.containsKey(5)
	            }
	        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindPrimitiveStaticType))
	})

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		script := `
	            pub fun main() {
	                let x: {Int8: String} = {} 
	                x.insert(key: 5, "")
	                x.insert(key: 4, "")
	            }
	        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindDictionaryValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(10), meter.getMemory(common.MemoryKindPrimitiveStaticType))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindDictionaryStaticType))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
	})

	t.Run("insert many no packing", func(t *testing.T) {
		t.Parallel()

		script := `
	            pub fun main() {
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
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
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
	            pub fun main() {
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
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
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
            pub fun main() {
                let x: {Int8: String} = {3: "a"} // 2 data slabs
                x[3] = "b" // fits in existing slab
                x[3] = "c" // fits in existing slab
                x[4] = "d" // fits in existing slab
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
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
            pub fun main() {
                let x: {Int8: String} = {3: "a"} // 2 data slabs
                x[3] = "b" // fits in existing slab
                x[4] = "d" // fits in existing slab
                x[3] = "c" // adds 1 data slab and metadata slab
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
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
            pub struct S {}

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

		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindStringValue))
		assert.Equal(t, uint64(66), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindCompositeValueBase))
		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindVariable))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCompositeStaticType))
		assert.Equal(t, uint64(9), meter.getMemory(common.MemoryKindCompositeTypeInfo))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCompositeField))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindInvocation))
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct S {}

            pub fun main() {
                let values = [S(), S(), S()]
                for value in values {
                  let a = value
                }
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(27), meter.getMemory(common.MemoryKindCompositeValueBase))
		assert.Equal(t, uint64(27), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(480), meter.getMemory(common.MemoryKindAtreeMapPreAllocatedElement))
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindVariable))

		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindCompositeStaticType))
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindCompositeTypeInfo))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCompositeField))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindInvocation))
	})
}

func TestInterpretSimpleCompositeMetering(t *testing.T) {
	t.Parallel()

	t.Run("auth account", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: AuthAccount) {
            
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main", newTestAuthAccountValue(meter, randomAddressValue()))
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindSimpleCompositeValueBase))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindSimpleCompositeValue))
	})

	t.Run("public account", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: PublicAccount) {
            
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main", newTestPublicAccountValue(meter, randomAddressValue()))
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindSimpleCompositeValueBase))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindSimpleCompositeValue))
	})
}

func TestInterpretCompositeFieldMetering(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct S {}
            pub fun main() {
                let s = S()
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
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
            pub struct S {
                pub let a: String
                init(_ a: String) {
                    self.a = a
                }
            }
            pub fun main() {
                let s = S("a")
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCompositeValueBase))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAtreeMapElementOverhead))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAtreeMapDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeMapMetaDataSlab))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCompositeField))
	})

	t.Run("2 field", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct S {
                pub let a: String
                pub let b: String
                init(_ a: String, _ b: String) {
                    self.a = a
                    self.b = b
                }
            }
            pub fun main() {
                let s = S("a", "b")
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(34), meter.getMemory(common.MemoryKindRawString))
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
            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInterpretedFunctionValue))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInvocation))
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
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunctionValue))
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
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunctionValue))

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInvocation))
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
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunctionValue))
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
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunctionValue))

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInvocation))
	})
}

func TestInterpretHostFunctionMetering(t *testing.T) {
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
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunctionValue))
	})

	t.Run("function pointers", func(t *testing.T) {
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
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunctionValue))
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

		// 1 for the struct method.
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindHostFunctionValue))
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

		// 1 for the struct init.
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindHostFunctionValue))
	})

	t.Run("builtin functions", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let a = Int8(5)

                let b = CompositeType("PublicKey")
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// builtin functions are not metered
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunctionValue))

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCompositeStaticType))
	})

	t.Run("stdlib function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                assert(true)
            }
        `

		meter := newTestMemoryGauge()

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseActivation := activations.NewActivation[*interpreter.Variable](nil, interpreter.BaseActivation)
		for _, valueDeclaration := range stdlib.DefaultStandardLibraryValues(nil) {
			baseValueActivation.DeclareValue(valueDeclaration)
			interpreter.Declare(baseActivation, valueDeclaration)
		}

		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
			t,
			script,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
			},
			meter,
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
            pub fun main() {
                let publicKey = PublicKey(
                    publicKey: "0102".decodeHex(),
                    signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
                )
            }
        `

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseActivation := activations.NewActivation[*interpreter.Variable](nil, interpreter.BaseActivation)
		for _, valueDeclaration := range []stdlib.StandardLibraryValue{
			stdlib.NewPublicKeyConstructor(
				assumeValidPublicKeyValidator{},
				nil,
				nil,
			),
			stdlib.SignatureAlgorithmConstructor,
		} {
			baseValueActivation.DeclareValue(valueDeclaration)
			interpreter.Declare(baseActivation, valueDeclaration)
		}

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
			t,
			script,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
			},
			meter,
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 1 host function created for 'decodeHex' of String value
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindHostFunctionValue))
	})

	t.Run("multiple public key creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
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
		baseActivation := activations.NewActivation[*interpreter.Variable](nil, interpreter.BaseActivation)
		for _, valueDeclaration := range []stdlib.StandardLibraryValue{
			stdlib.NewPublicKeyConstructor(
				assumeValidPublicKeyValidator{},
				nil,
				nil,
			),
			stdlib.SignatureAlgorithmConstructor,
		} {
			baseValueActivation.DeclareValue(valueDeclaration)
			interpreter.Declare(baseActivation, valueDeclaration)
		}

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
			t,
			script,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
			},
			meter,
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 2 = 2x 1 host function created for 'decodeHex' of String value
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindHostFunctionValue))
	})
}

func TestInterpretBoundFunctionMetering(t *testing.T) {
	t.Parallel()

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

		// No bound functions are created without usages.
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoundFunctionValue))
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

		// No bound functions are created without usages.
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoundFunctionValue))
	})

	t.Run("struct method usage", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                pub fun bar() {}
            }

            pub fun main() {
                let foo = Foo()
                foo.bar()
                foo.bar()
                foo.bar()
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
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
            pub fun main() {
                let x: String? = "hello"
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindOptionalValue))
	})

	t.Run("dictionary get", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: {Int8: String} = {1: "foo", 2: "bar"}
                let y = x[0]
                let z = x[1]
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 2 for `z`
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindOptionalValue))

		assert.Equal(t, uint64(14), meter.getMemory(common.MemoryKindPrimitiveStaticType))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindDictionaryStaticType))
	})

	t.Run("dictionary set", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: {Int8: String} = {1: "foo", 2: "bar"}
                x[0] = "a"
                x[1] = "b"
            }
        `

		meter := newTestMemoryGauge()

		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 from creating new entry by setting x[0]
		// 2 from overwriting entry by setting x[1]
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindOptionalValue))
	})

	t.Run("OptionalType", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let type: Type = Type<Int>()
                let a = OptionalType(type)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
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
            pub fun main() {
                let x = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1
                let y = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretUIntMetering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt + 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt - 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt).saturatingSubtract(2 as UInt)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt * 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt / 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt % 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt | 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt ^ 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt & 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt << 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt >> 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1
                let y = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UInt = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretUInt8Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt8 + 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt8).saturatingAdd(2 as UInt8)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt8 - 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt8).saturatingSubtract(2 as UInt8)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt8 * 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt8).saturatingMultiply(2 as UInt8)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 / 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 % 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 | 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 ^ 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 & 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 << 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 >> 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UInt8 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretUInt16Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt16 + 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt16).saturatingAdd(2 as UInt16)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt16 - 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt16).saturatingSubtract(2 as UInt16)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt16 * 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt16).saturatingMultiply(2 as UInt16)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 / 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 % 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 | 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 ^ 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 & 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 << 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 >> 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UInt16 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretUInt32Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt32 + 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt32).saturatingAdd(2 as UInt32)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt32 - 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt32).saturatingSubtract(2 as UInt32)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt32 * 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt32).saturatingMultiply(2 as UInt32)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 / 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 % 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 | 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 ^ 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 & 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 << 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 >> 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UInt32 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretUInt64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt64 + 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt64).saturatingAdd(2 as UInt64)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt64 - 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt64).saturatingSubtract(2 as UInt64)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt64 * 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt64).saturatingMultiply(2 as UInt64)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 / 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 % 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 | 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 ^ 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 & 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 << 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 >> 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UInt64 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretUInt128Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt128 + 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt128).saturatingAdd(2 as UInt128)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt128 - 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt128).saturatingSubtract(2 as UInt128)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt128 * 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt128).saturatingMultiply(2 as UInt128)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 / 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 % 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 | 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 ^ 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 & 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 << 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 >> 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UInt128 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretUInt256Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt256 + 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt256).saturatingAdd(2 as UInt256)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt256 - 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt256).saturatingSubtract(2 as UInt256)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt256 * 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt256).saturatingMultiply(2 as UInt256)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 / 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 % 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 | 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 ^ 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 & 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 << 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 >> 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UInt256 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretInt8Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
                let y: Int8 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
                let y: Int8 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
                let y: Int8 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3
                let y: Int8 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
                let y: Int8 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 1
		// y: 1
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretInt16Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
                let y: Int16 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
                let y: Int16 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
                let y: Int16 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3
                let y: Int16 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
                let y: Int16 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 2
		// y: 2
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretInt32Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
                let y: Int32 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
                let y: Int32 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
                let y: Int32 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3
                let y: Int32 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
                let y: Int32 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 4
		// y: 4
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretInt64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
                let y: Int64 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
                let y: Int64 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
                let y: Int64 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3
                let y: Int64 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
                let y: Int64 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 8
		// y: 8
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretInt128Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
                let y: Int128 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
                let y: Int128 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
                let y: Int128 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3
                let y: Int128 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
                let y: Int128 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 16
		// y: 16
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
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
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))
	})
}

func TestInterpretInt256Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
                let y: Int256 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
                let y: Int256 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
                let y: Int256 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3
                let y: Int256 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
                let y: Int256 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 32
		// y: 32
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretWord8Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindElaboration))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word8 + 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as Word8 - 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word8 * 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 / 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 % 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 | 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 ^ 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 & 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 << 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 >> 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Word8 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretWord16Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word16 + 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as Word16 - 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word16 * 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 / 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 % 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 | 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 ^ 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 & 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 << 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 >> 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Word16 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretWord32Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word32 + 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as Word32 - 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word32 * 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 / 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 % 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 | 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 ^ 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 & 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 << 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 >> 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Word32 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretWord64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word64 + 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as Word64 - 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word64 * 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 / 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 % 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 | 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 ^ 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 & 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 << 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 >> 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Word64 = 1
                x == 1
                x != 1
                x > 1
                x >= 1
                x < 1
                x <= 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretBoolMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: Bool = true
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})

	t.Run("negation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                !true
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})

	t.Run("equality, true", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                true == true
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})

	t.Run("equality, false", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                true == false
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})

	t.Run("inequality", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                true != false
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretNilMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: Bool? = nil
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindNilValue))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
	})
}

func TestInterpretVoidMetering(t *testing.T) {
	t.Parallel()

	t.Run("returnless function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindVoidValue))
	})

	t.Run("returning function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Bool {
                return true
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindVoidValue))
	})
}

func TestInterpretStorageReferenceValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
              resource R {}

              pub fun main(account: AuthAccount) {
                  account.borrow<&R>(from: /storage/r)
              }
            `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		account := newTestAuthAccountValue(meter, interpreter.AddressValue{})
		_, err := inter.Invoke("main", account)
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

          pub fun main(): &Int {
              let x: Int = 1
              let y = &x as &Int
              return y
          }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindEphemeralReferenceValue))
	})

	t.Run("creation, optional", func(t *testing.T) {
		t.Parallel()

		script := `
          resource R {}

          pub fun main(): &Int {
              let x: Int? = 1
              let y = &x as &Int?
              return y!
          }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindEphemeralReferenceValue))
	})
}

func TestInterpretStringMetering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = "a"
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("assignment", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = "a"
                let y = x
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("Unicode", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = "İ"
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("toLower, ASCII", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = "ABC".toLower()
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 + 3 (abc)
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("toLower, Unicode", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = "İ".toLower()
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
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
            pub fun main() {
                let x: Character = "a"
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCharacterValue))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("assignment", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: Character = "a"
                let y = x
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCharacterValue))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindStringValue))
	})

	t.Run("from string GetKey", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: String = "a"
                let y: Character = x[0]
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCharacterValue))
	})
}

func TestInterpretAddressValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: Address = 0x0
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddressValue))
	})

	t.Run("convert", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x = Address(0x0)
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddressValue))
	})
}

func TestInterpretPathValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x = /public/bar
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPathValue))
	})

	t.Run("convert", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x = PublicPath(identifier: "bar")
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPathValue))
	})
}

func TestInterpretCapabilityValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            resource R {}

            pub fun main(account: AuthAccount) {
                let r <- create R()
                account.save(<-r, to: /storage/r)
                let x = account.link<&R>(/public/capo, target: /storage/r)
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		account := newTestAuthAccountValue(meter, interpreter.AddressValue{})
		_, err := inter.Invoke("main", account)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCapabilityValue))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindPathValue))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindReferenceStaticType))
	})

	t.Run("array element", func(t *testing.T) {
		t.Parallel()

		script := `
            resource R {}

            pub fun main(account: AuthAccount) {
                let r <- create R()
                account.save(<-r, to: /storage/r)
                let x = account.link<&R>(/public/capo, target: /storage/r)

                let y = [x]
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		account := newTestAuthAccountValue(meter, interpreter.AddressValue{})
		_, err := inter.Invoke("main", account)
		require.NoError(t, err)

		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindCapabilityStaticType))
	})
}

func TestInterpretLinkValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            resource R {}

            pub fun main(account: AuthAccount) {
                account.link<&R>(/public/capo, target: /private/p)
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		account := newTestAuthAccountValue(meter, interpreter.AddressValue{})
		_, err := inter.Invoke("main", account)
		require.NoError(t, err)

		// Metered twice only when Atree validation is enabled.
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindLinkValue))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindReferenceStaticType))
	})
}

func TestInterpretTypeValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("static constructor", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let t: Type = Type<Int>()
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindTypeValue))
	})

	t.Run("dynamic constructor", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let t: Type = ConstantSizedArrayType(type: Type<Int>(), size: 2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindTypeValue))
	})

	t.Run("getType", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let v = 5
                let t: Type = v.getType()
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
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

            pub fun main() {
                
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindVariable))
	})

	t.Run("params", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: String, b: Bool) {
                
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke(
			"main",
			interpreter.NewUnmeteredStringValue(""),
			interpreter.FalseValue,
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindVariable))
	})

	t.Run("nested params", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                var x = fun (x: String, y: Bool) {}
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindVariable))
	})

	t.Run("applied nested params", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                var x = fun (x: String, y: Bool) {}
                x("", false)
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindVariable))
	})
}

func TestInterpretFix64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(80), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4 + 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
                let y: Fix64 = x.saturatingAdd(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4 - 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
                let y: Fix64 = x.saturatingSubtract(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4 * 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
                let y: Fix64 = x.saturatingMultiply(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 3.4 / 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 3.4
                let y: Fix64 = x.saturatingMultiply(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 3.4 % 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// quotient (div) : 8
		// truncatedQuotient: 8
		// truncatedQuotient.Mul(o): 8
		// result: 8
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
                let y: Fix64 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 8
		// y: 8
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(80), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("creation as supertype", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: FixedPoint = -1.4
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(80), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
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
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))
		assert.Equal(t, uint64(560), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretUFix64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))
		assert.Equal(t, uint64(80), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4 + 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4
                let y: UFix64 = x.saturatingAdd(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 2.5 - 1.4 
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4
                let y: UFix64 = x.saturatingSubtract(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4 * 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4
                let y: UFix64 = x.saturatingMultiply(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 3.4 / 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 3.4
                let y: UFix64 = x.saturatingMultiply(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 3.4 % 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// quotient (div) : 8
		// truncatedQuotient: 8
		// truncatedQuotient.Mul(o): 8
		// result: 8
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(160), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("creation as supertype", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: FixedPoint = 1.4
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumberValue))

		assert.Equal(t, uint64(80), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("logical operations", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
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
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoolValue))

		assert.Equal(t, uint64(560), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretTokenMetering(t *testing.T) {
	t.Parallel()

	t.Run("identifier tokens", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                var x: String = "hello"
            }

            pub struct foo {
                var x: Int

                init() {
                    self.x = 4
                }
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindTypeToken))
		assert.Equal(t, uint64(25), meter.getMemory(common.MemoryKindSpaceToken))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
	})

	t.Run("syntax tokens", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                var a: [String] = []
                var b = 4 + 6
                var c = true && false != false
                var d = 4 as! AnyStruct
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(36), meter.getMemory(common.MemoryKindTypeToken))
		assert.Equal(t, uint64(31), meter.getMemory(common.MemoryKindSpaceToken))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
	})

	t.Run("comments", func(t *testing.T) {
		t.Parallel()

		script := `
            /*  first line
                second line
            */

            // single line comment
            pub fun main() {}
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(11), meter.getMemory(common.MemoryKindTypeToken))
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindSpaceToken))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
	})

	t.Run("numeric literals", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                var a = 1
                var b = 0b1
                var c = 0o1
                var d = 0x1
                var e = 1.4
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(27), meter.getMemory(common.MemoryKindTypeToken))
		assert.Equal(t, uint64(26), meter.getMemory(common.MemoryKindSpaceToken))
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

            pub fun main(account: AuthAccount) {
                let s = CompositeType("")
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)
		account := newTestAuthAccountValue(meter, interpreter.AddressValue{})
		_, err := inter.Invoke("main", account)
		require.NoError(t, err)

		emptyLocationStringCount := meter.getMemory(common.MemoryKindRawString)

		// Raw string count with non-empty location

		script = `
            struct S {}

            pub fun main(account: AuthAccount) {
                let s = CompositeType("S.test.S")
            }
        `

		meter = newTestMemoryGauge()
		inter = parseCheckAndInterpretWithMemoryMetering(t, script, meter)
		account = newTestAuthAccountValue(meter, interpreter.AddressValue{})
		_, err = inter.Invoke("main", account)
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
            pub fun main() {
                let foo = 4
                let bar = 5
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 'main', 'foo', 'bar', empty-return-type
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindIdentifier))
	})

	t.Run("parameters", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(foo: String, bar: String) {
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke(
			"main",
			interpreter.NewUnmeteredStringValue("x"),
			interpreter.NewUnmeteredStringValue("y"),
		)
		require.NoError(t, err)

		// 'main', 'foo', 'String', 'bar', 'String', empty-return-type
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindIdentifier))
	})

	t.Run("composite declaration", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {}

            pub struct foo {
                var x: String
                var y: String

                init() {
                    self.x = "a"
                    self.y = "b"
                }

                pub fun bar() {}
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindIdentifier))
	})

	t.Run("member resolvers", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {            // 2 - 'main', empty-return-type
                let foo = ["a", "b"]    // 1
                foo.length              // 3 - 'foo', 'length', constant field resolver
                foo.length              // 3 - 'foo', 'length', constant field resolver (not re-used)
                foo.removeFirst()       // 3 - 'foo', 'removeFirst', function resolver
                foo.removeFirst()       // 3 - 'foo', 'removeFirst', function resolver (not re-used)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(15), meter.getMemory(common.MemoryKindIdentifier))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindPrimitiveStaticType))
	})
}

func TestInterpretInterfaceStaticType(t *testing.T) {
	t.Parallel()

	t.Run("RestrictedType", func(t *testing.T) {
		t.Parallel()

		script := `
            struct interface I {}

            pub fun main() {
                let type = Type<AnyStruct{I}>()

                RestrictedType(
                    identifier: type.identifier,
                    restrictions: [type.identifier]
                )
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInterfaceStaticType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindRestrictedStaticType))
	})
}

func TestInterpretFunctionStaticType(t *testing.T) {
	t.Parallel()

	t.Run("FunctionType", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                FunctionType(parameters: [], return: Type<Never>())
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindFunctionStaticType))
	})

	t.Run("array element", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun hello() {}

            pub fun main() {
                let a = [hello]
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindFunctionStaticType))
	})

	t.Run("set bound function to variable", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct S {
                fun naught() {}
            }

            pub fun main() {
                let x = S()
                let y = x.naught
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindFunctionStaticType))
	})

	t.Run("isInstance", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct S {
                fun naught() {}
            }

            pub fun main() {
                let x = S()
                x.naught.isInstance(Type<Int>())
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindFunctionStaticType))
	})
}

func TestInterpretASTMetering(t *testing.T) {
	t.Parallel()

	t.Run("arguments", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                foo(a: "hello", b: 23)
                bar("hello", 23)
            }

            pub fun foo(a: String, b: Int) {
            }

            pub fun bar(_ a: String, _ b: Int) {
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindArgument))
	})

	t.Run("blocks", func(t *testing.T) {
		script := `
            pub fun main() {
                var i = 0
                if i != 0 {
                    i = 0
                }

                while i < 2 {
                    i = i + 1
                }

                var a = "foo"
                switch i {
                    case 1:
                        a = "foo_1"
                    case 2:
                        a = "foo_2"
                    case 3:
                        a = "foo_3"
                }
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindBlock))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindFunctionBlock))
	})

	t.Run("declarations", func(t *testing.T) {
		script := `
            import Foo from 0x42

            pub let x = 1
            pub var y = 2

            pub fun main() {
                var z = 3
            }

            pub fun foo(_ x: String, _ y: Int) {}

            pub struct A {
                pub var a: String

                init() {
                    self.a = "hello"
                }
            }

            pub struct interface B {}

            pub resource C {
                let a: Int

                init() {
                    self.a = 6
                }
            }

            pub resource interface D {}

            pub enum E: Int8 {
                pub case a
                pub case b
                pub case c
            }

            transaction {}

            #pragma
        `

		importedChecker, err := checker.ParseAndCheckWithOptions(t,
			`
                pub let Foo = 1
            `,
			checker.ParseAndCheckOptions{
				Location: utils.ImportedLocation,
			},
		)
		require.NoError(t, err)

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
			t,
			script,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
				Config: &interpreter.Config{
					ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
						require.IsType(t, common.AddressLocation{}, location)
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
			},
			meter,
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindFunctionDeclaration))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindCompositeDeclaration))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterfaceDeclaration))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindEnumCaseDeclaration))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindFieldDeclaration))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindTransactionDeclaration))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindImportDeclaration))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindVariableDeclaration))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindSpecialFunctionDeclaration))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPragmaDeclaration))

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindFunctionBlock))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindParameter))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindParameterList))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindProgram))
		assert.Equal(t, uint64(13), meter.getMemory(common.MemoryKindMembers))
	})

	t.Run("statements", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                var a = 5

                while a < 10 {               // while
                    if a == 5 {              // if
                        a = a + 1            // assignment
                        continue             // continue
                    }
                    break                    // break
                }

                foo()                        // expression statement

                for value in [1, 2, 3] {}    // for

                var r1 <- create bar()
                var r2 <- create bar()
                r1 <-> r2                    // swap

                destroy r1                   // expression statement
                destroy r2                   // expression statement

                switch a {                   // switch
                    case 1:
                        a = 2                // assignment
                }
            }

            pub fun foo(): Int {
                 return 5                    // return
            }

            resource bar {}

            pub contract Events {
                event FooEvent(x: Int, y: Int)

                fun events() {
                    emit FooEvent(x: 1, y: 2)    // emit
                }
            }
        `
		meter := newTestMemoryGauge()

		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
			t,
			script,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ContractValueHandler: func(
						inter *interpreter.Interpreter,
						compositeType *sema.CompositeType,
						constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
						invocationRange ast.Range,
					) interpreter.ContractValue {
						// Just return a dummy value
						return &interpreter.CompositeValue{}
					},
				},
			},
			meter,
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAssignmentStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindBreakStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindContinueStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindIfStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindForStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindWhileStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindReturnStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindSwapStatement))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindExpressionStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindSwitchStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindEmitStatement))

		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindTransfer))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindMembers))
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindPrimitiveStaticType))
	})

	t.Run("expressions", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                var a = 5                                // integer expr
                var b = 1.2 + 2.3                        // binary, fixed-point expr
                var c = !true                            // unary, boolean expr
                var d: String? = "hello"                 // string expr
                var e = nil                              // nil expr
                var f: [AnyStruct] = [[], [], []]        // array expr
                var g: {Int: {Int: AnyStruct}} = {1:{}}  // nil expr
                var h <- create bar()                    // create, identifier, invocation
                var i = h.baz                            // member access, identifier x2
                destroy h                                // destroy
                var j = f[0]                             // index access, identifier, integer
                var k = fun() {}                         // function expr
                k()                                      // identifier, invocation
                var l = c ? 1 : 2                        // conditional, identifier, integer x2
                var m = d as AnyStruct                   // casting, identifier
                var n = &d as &AnyStruct                 // reference, casting, identifier
                var o = d!                               // force, identifier
                var p = /public/somepath                 // path
            }

            resource bar {
                let baz: Int
                init() {
                    self.baz = 0x4
                }
            }
        `
		meter := newTestMemoryGauge()

		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindBooleanExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNilExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindStringExpression))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindIntegerExpression))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindFixedPointExpression))
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindArrayExpression))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindDictionaryExpression))
		assert.Equal(t, uint64(10), meter.getMemory(common.MemoryKindIdentifierExpression))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInvocationExpression))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindMemberExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindIndexExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindConditionalExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindUnaryExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindBinaryExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindFunctionExpression))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCastingExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCreateExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindDestroyExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindReferenceExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindForceExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPathExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindDictionaryEntry))
		assert.Equal(t, uint64(25), meter.getMemory(common.MemoryKindPrimitiveStaticType))
	})

	t.Run("types", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                var a: Int = 5                                     // nominal type
                var b: String? = "hello"                           // optional type
                var c: [Int; 2] = [1, 2]                           // constant sized type
                var d: [String] = []                               // variable sized type
                var e: {Int: String} = {}                          // dictionary type

                var f: ((String):Int) = fun(_a: String): Int {     // function type
                    return 1
                }

                var g = &a as &Int                                 // reference type
                var h: AnyStruct{foo} = bar()                      // restricted type
                var i: Capability<&bar>? = nil                     // instantiation type
            }

            struct interface foo {}

            struct bar: foo {}
        `
		meter := newTestMemoryGauge()

		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindConstantSizedType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindDictionaryType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindFunctionType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInstantiationType))
		assert.Equal(t, uint64(17), meter.getMemory(common.MemoryKindNominalType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindOptionalType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindReferenceType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindRestrictedType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindVariableSizedType))

		assert.Equal(t, uint64(15), meter.getMemory(common.MemoryKindTypeAnnotation))
	})

	t.Run("position info", func(t *testing.T) {
		script := `
            pub let x = 1
            pub var y = 2

            pub fun main() {
                var z = 3
            }

            pub fun foo(_ x: String, _ y: Int) {}

            pub struct A {
                pub var a: String

                init() {
                    self.a = "hello"
                }
            }

            pub struct interface B {}
        `

		meter := newTestMemoryGauge()

		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(231), meter.getMemory(common.MemoryKindPosition))
		assert.Equal(t, uint64(126), meter.getMemory(common.MemoryKindRange))
	})

	t.Run("locations", func(t *testing.T) {
		script := `
            import A from 0x42
            import B from "string-location"
        `

		importedChecker, err := checker.ParseAndCheckWithOptions(t,
			`
                pub let A = 1
                pub let B = 1
            `,
			checker.ParseAndCheckOptions{
				Location: utils.ImportedLocation,
			},
		)
		require.NoError(t, err)

		meter := newTestMemoryGauge()
		_, err = parseCheckAndInterpretWithOptionsAndMemoryMetering(
			t,
			script,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
				Config: &interpreter.Config{
					ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
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
			},
			meter,
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddressLocation))
	})

}

func TestInterpretVariableActivationMetering(t *testing.T) {
	t.Parallel()

	t.Run("single function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindActivation))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindActivationEntries))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInvocation))
	})

	t.Run("nested function call", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                foo(a: "hello", b: 23)
            }

            pub fun foo(a: String, b: Int) {
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindActivation))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindActivationEntries))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInvocation))
	})

	t.Run("local scope", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                if true {
                    let a = 1
                }
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindActivation))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindActivationEntries))
	})
}

func TestInterpretStaticTypeConversionMetering(t *testing.T) {
	t.Parallel()

	t.Run("primitive static types", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let a: {Int: AnyStruct{Foo}} = {}           // dictionary + restricted
                let b: [&Int] = []                          // variable-sized + reference
                let c: [Int?; 2] = [1, 2]                   // constant-sized + optional
                let d: [Capability<&Bar>] = []             //  capability + variable-sized + reference
            }

            pub struct interface Foo {}

            pub struct Bar: Foo {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindDictionarySemaType))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindVariableSizedSemaType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindConstantSizedSemaType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindOptionalSemaType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindRestrictedSemaType))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindReferenceSemaType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCapabilitySemaType))
	})
}

func TestInterpretStorageMapMetering(t *testing.T) {
	t.Parallel()

	script := `
        resource R {}

        pub fun main(account: AuthAccount) {
            let r <- create R()
            account.save(<-r, to: /storage/r)
            account.link<&R>(/public/capo, target: /storage/r)
            account.borrow<&R>(from: /storage/r)
        }
    `

	meter := newTestMemoryGauge()
	inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

	account := newTestAuthAccountValue(meter, interpreter.AddressValue{})
	_, err := inter.Invoke("main", account)
	require.NoError(t, err)

	assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindStorageMap))
	assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindStorageKey))
}

func TestInterpretValueStringConversion(t *testing.T) {
	t.Parallel()

	testValueStringConversion := func(t *testing.T, script string, args ...interpreter.Value) {
		meter := newTestMemoryGauge()

		var loggedString string

		logFunction := stdlib.NewStandardLibraryFunction(
			"log",
			&sema.FunctionType{
				Parameters: []*sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.NewTypeAnnotation(sema.AnyStructType),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					sema.VoidType,
				),
			},
			``,
			func(invocation interpreter.Invocation) interpreter.Value {
				// Reset gauge, to only capture the values metered during string conversion
				meter.meter = make(map[common.MemoryKind]uint64)

				loggedString = invocation.Arguments[0].MeteredString(invocation.Interpreter, interpreter.SeenReferences{})
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(logFunction)

		baseActivation := activations.NewActivation[*interpreter.Variable](nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, logFunction)

		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(t, script,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
			},
			meter,
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
                    pub fun main() {
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
            pub fun main() {
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
            pub fun main() {
                let x = 4
                log(&x as &AnyStruct)
            }
        `

		testValueStringConversion(t, script)
	})

	t.Run("Interpreted Function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x = fun(a: String, b: Bool) {}
                log(&x as &AnyStruct)
            }
        `

		testValueStringConversion(t, script)
	})

	t.Run("Bound Function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x = Foo()
                log(x.bar)
            }

            struct Foo {
                pub fun bar(a: String, b: Bool) {}
            }
        `

		testValueStringConversion(t, script)
	})

	t.Run("Void", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
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
            pub fun main(a: Capability<&{Foo}>) {
                log(a)
            }

            struct interface Foo {}
            struct Bar: Foo {}
        `

		testValueStringConversion(t, script,
			interpreter.NewUnmeteredCapabilityValue(
				interpreter.AddressValue{1},
				interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: "somepath",
				},
				interpreter.CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "Bar",
					TypeID:              "S.test.Bar",
				},
			))
	})

	t.Run("Type", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
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

		logFunction := stdlib.NewStandardLibraryFunction(
			"log",
			&sema.FunctionType{
				Parameters: []*sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.NewTypeAnnotation(sema.AnyStructType),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					sema.VoidType,
				),
			},
			``,
			func(invocation interpreter.Invocation) interpreter.Value {
				// Reset gauge, to only capture the values metered during string conversion
				meter.meter = make(map[common.MemoryKind]uint64)

				loggedString = invocation.Arguments[0].MeteredString(invocation.Interpreter, interpreter.SeenReferences{})
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(logFunction)

		baseActivation := activations.NewActivation[*interpreter.Variable](nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, logFunction)

		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(t, script,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
			},
			meter,
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

		for staticType, typeName := range interpreter.PrimitiveStaticTypes {
			switch staticType {
			case interpreter.PrimitiveStaticTypeUnknown,
				interpreter.PrimitiveStaticTypeAny,
				interpreter.PrimitiveStaticTypeAuthAccountContracts,
				interpreter.PrimitiveStaticTypePublicAccountContracts,
				interpreter.PrimitiveStaticTypeAuthAccountKeys,
				interpreter.PrimitiveStaticTypePublicAccountKeys,
				interpreter.PrimitiveStaticTypeAuthAccountInbox,
				interpreter.PrimitiveStaticTypeAccountKey,
				interpreter.PrimitiveStaticType_Count:
				continue
			case interpreter.PrimitiveStaticTypeAnyResource:
				typeName = "@" + typeName
			case interpreter.PrimitiveStaticTypeMetaType:
				typeName = "Type"
			}

			script := fmt.Sprintf(`
                pub fun main() {
                    log(Type<%s>())
                }`,
				typeName,
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
				constructor: "((String): AnyStruct)",
			},
			{
				name:        "Reference",
				constructor: "&Int",
			},
			{
				name:        "Auth Reference",
				constructor: "auth &AnyStruct",
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
                    pub fun main() {
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
            pub fun main() {
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

	t.Run("Restricted type", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                log(Type<AnyStruct{Foo}>())
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
	inter := parseCheckAndInterpretWithMemoryMetering(t, code, meter)

	stringValue := interpreter.NewUnmeteredStringValue("abc")

	_, err := inter.Invoke("test", stringValue)
	require.NoError(t, err)

	// 1 + 3
	assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindBytes))
}

func TestOverEstimateBigIntFromString(t *testing.T) {

	t.Parallel()

	for _, v := range []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(9),
		big.NewInt(10),
		big.NewInt(99),
		big.NewInt(100),
		big.NewInt(999),
		big.NewInt(1000),
		big.NewInt(math.MaxUint16),
		big.NewInt(math.MaxUint32),
		new(big.Int).SetUint64(math.MaxUint64),
		func() *big.Int {
			v := new(big.Int).SetUint64(math.MaxUint64)
			return v.Mul(v, big.NewInt(2))
		}(),
		func() *big.Int {
			v := new(big.Int).SetUint64(math.MaxUint64)
			return v.Mul(v, new(big.Int).SetUint64(math.MaxUint64))
		}(),
	} {

		// Always should be equal or overestimate
		assert.LessOrEqual(t,
			common.BigIntByteLength(v),
			common.OverEstimateBigIntFromString(v.String()),
		)

		neg := new(big.Int).Neg(v)
		assert.LessOrEqual(t,
			common.BigIntByteLength(neg),
			common.OverEstimateBigIntFromString(neg.String()),
		)
	}
}
