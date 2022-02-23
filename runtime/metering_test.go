package runtime

import (
	"testing"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRuntimeArrayMetering(t *testing.T) {
	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`
	pub fun main() {
		let x: [Int8] = []
		let y: [[String]] = [[]]
		let z: [[[Bool]]] = [[[]]]
	}
    `)

	meter := make(map[common.MemoryKind]uint64)

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage:   storage,
		useMemory: testUseMemory(meter),
	}

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)

	require.NoError(t, err)

	assert.Equal(t, meter[common.MemoryKindArray], uint64(6))
}
