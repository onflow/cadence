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

package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func testUseMemory(meter map[common.MemoryKind]uint64) func(common.MemoryUsage) {
	return func(usage common.MemoryUsage) {
		current, ok := meter[usage.Kind]
		if !ok {
			current = 0
		}
		meter[usage.Kind] = current + usage.Amount
	}
}

func TestImportedValueMemoryMetering(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	runtimeInterface := func(meter map[common.MemoryKind]uint64) *testRuntimeInterface {
		return &testRuntimeInterface{
			useMemory: testUseMemory(meter),
			decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
				return jsoncdc.Decode(b)
			},
		}
	}

	executeScript := func(script []byte, meter map[common.MemoryKind]uint64, args ...cadence.Value) {
		_, err := runtime.ExecuteScript(
			Script{
				Source:    script,
				Arguments: encodeArgs(args),
			},
			Context{
				Interface: runtimeInterface(meter),
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)
	}

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: String) {}
        `)

		meter := make(map[common.MemoryKind]uint64)

		executeScript(
			script,
			meter,
			cadence.String("hello"),
		)

		assert.Equal(t, uint64(6), meter[common.MemoryKindString])
	})

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: String?) {}
        `)

		meter := make(map[common.MemoryKind]uint64)

		executeScript(
			script,
			meter,
			cadence.NewOptional(cadence.String("hello")),
		)

		assert.Equal(t, uint64(1), meter[common.MemoryKindOptional])
	})

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Int) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewInt(2))
		assert.Equal(t, uint64(8), meter[common.MemoryKindBigInt])
	})

	t.Run("UInt", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUInt(2))
		assert.Equal(t, uint64(8), meter[common.MemoryKindBigInt])
	})

	t.Run("UInt8", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt8) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUInt8(2))
		assert.Equal(t, uint64(1), meter[common.MemoryKindNumber])
	})

	t.Run("UInt16", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt16) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUInt16(2))
		assert.Equal(t, uint64(2), meter[common.MemoryKindNumber])
	})

	t.Run("UInt32", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt32) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUInt32(2))
		assert.Equal(t, uint64(4), meter[common.MemoryKindNumber])
	})

	t.Run("UInt64", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt64) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUInt64(2))
		assert.Equal(t, uint64(8), meter[common.MemoryKindNumber])
	})

	t.Run("UInt128", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt128) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUInt128(2))
		assert.Equal(t, uint64(16), meter[common.MemoryKindNumber])
	})

	t.Run("UInt256", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt256) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUInt256(2))
		assert.Equal(t, uint64(32), meter[common.MemoryKindNumber])
	})


	t.Run("Word8", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Word8) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewWord8(2))
		assert.Equal(t, uint64(1), meter[common.MemoryKindNumber])
	})

	t.Run("Word16", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Word16) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewWord16(2))
		assert.Equal(t, uint64(2), meter[common.MemoryKindNumber])
	})

	t.Run("Word32", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Word32) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewWord32(2))
		assert.Equal(t, uint64(4), meter[common.MemoryKindNumber])
	})

	t.Run("Word64", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Word64) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewWord64(2))
		assert.Equal(t, uint64(8), meter[common.MemoryKindNumber])
	})
}
