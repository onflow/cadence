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

	t.Run("optional", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: String?) {}
        `)

		meter := make(map[common.MemoryKind]uint64)

		runtimeInterface := &testRuntimeInterface{
			useMemory: testUseMemory(meter),
			decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
				return jsoncdc.Decode(b)
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewOptional(cadence.String("hello")),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, uint64(1), meter[common.MemoryKindOptional])
	})
}
