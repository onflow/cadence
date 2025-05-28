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

package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/tests/runtime_utils"
)

func TestRuntimeCCFEncodeStruct(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name   string
		value  string
		output []byte
	}

	tests := []testCase{
		{
			name:   "String",
			value:  `"test"`,
			output: []byte{0xd8, 0x82, 0x82, 0xd8, 0x89, 0x1, 0x64, 0x74, 0x65, 0x73, 0x74},
		},
		{
			name:   "Bool",
			value:  `true`,
			output: []byte{0xd8, 0x82, 0x82, 0xd8, 0x89, 0x0, 0xf5},
		},
		{
			name:   "function",
			value:  `fun (): Int { return 1 }`,
			output: []byte{0xd8, 0x82, 0x82, 0xd8, 0x89, 0x18, 0x33, 0x84, 0x80, 0x80, 0xd8, 0xb9, 0x4, 0x0},
		},
	}

	test := func(test testCase) {
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			runtime := NewTestInterpreterRuntime()

			runtimeInterface := &TestRuntimeInterface{
				Storage: NewTestLedger(nil, nil),
			}

			script := []byte(fmt.Sprintf(
				`
                  access(all) fun main(): [UInt8]? {
                      let value = %s
                      return CCF.encode(&value as &AnyStruct)
                  }
                `,
				test.value,
			))

			result, err := runtime.ExecuteScript(
				Script{
					Source: script,
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)
			require.NoError(t, err)

			assert.Equal(t,
				cadence.NewOptional(
					cadence.ByteSliceToByteArray(test.output),
				),
				result,
			)
		})
	}

	for _, testCase := range tests {
		test(testCase)
	}
}
