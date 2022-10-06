/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRLPDecodeString(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`

      pub fun main(_ data: [UInt8]): [UInt8] {
          return RLP.decodeString(data)
      }
    `)

	type testCase struct {
		name           string
		input          []cadence.Value
		output         []cadence.Value
		expectedErrMsg string
	}

	tests := []testCase{
		{
			name:           "empty input",
			input:          []cadence.Value{},
			output:         nil,
			expectedErrMsg: "failed to RLP-decode string: input data is empty",
		},
		{
			name: "empty string",
			input: []cadence.Value{
				cadence.UInt8(128),
			},
			output: []cadence.Value{},
		},
		{
			name: "single char",
			input: []cadence.Value{
				cadence.UInt8(47),
			},
			output: []cadence.Value{
				cadence.UInt8(47),
			},
		},
		{
			name: "single char with an extra trailing byte",
			input: []cadence.Value{
				cadence.UInt8(65),
				cadence.UInt8(1),
			},
			output:         nil,
			expectedErrMsg: "failed to RLP-decode string: input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes.",
		},
		{
			name: "dog",
			input: []cadence.Value{
				cadence.UInt8(0x83),
				cadence.UInt8(0x64),
				cadence.UInt8(0x6f),
				cadence.UInt8(0x67),
			},
			output: []cadence.Value{
				cadence.UInt8('d'),
				cadence.UInt8('o'),
				cadence.UInt8('g'),
			},
		},
		{
			name: "dog, and extra trailing byte",
			input: []cadence.Value{
				cadence.UInt8(0x83),
				cadence.UInt8(0x64),
				cadence.UInt8(0x6f),
				cadence.UInt8(0x67),
				cadence.UInt8(1), // extra byte
			},
			output:         nil,
			expectedErrMsg: "failed to RLP-decode string: input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes.",
		},
		{
			name: "handling lower level errors - incomplete data case",
			input: []cadence.Value{
				cadence.UInt8(131),
			},
			output:         nil,
			expectedErrMsg: "failed to RLP-decode string: incomplete input! not enough bytes to read",
		},
	}

	test := func(test testCase) {
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			runtimeInterface := &testRuntimeInterface{
				storage: newTestLedger(nil, nil),
				meterMemory: func(_ common.MemoryUsage) error {
					return nil
				},
			}
			runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(runtimeInterface, b)
			}

			result, err := runtime.ExecuteScript(
				Script{
					Source: script,
					Arguments: encodeArgs([]cadence.Value{
						cadence.Array{
							ArrayType: cadence.VariableSizedArrayType{
								ElementType: cadence.UInt8Type{},
							},
							Values: test.input,
						},
					}),
				},
				Context{
					Interface: runtimeInterface,
					Location:  utils.TestLocation,
				},
			)
			if len(test.expectedErrMsg) > 0 {
				require.Error(t, err)
				_ = err.Error()

				assert.ErrorContains(t, err, test.expectedErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t,
					cadence.Array{
						Values: test.output,
					}.WithType(cadence.VariableSizedArrayType{
						ElementType: cadence.UInt8Type{},
					}),
					result,
				)
			}
		})
	}

	for _, testCase := range tests {
		test(testCase)
	}
}

func TestRLPDecodeList(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`

      pub fun main(_ data: [UInt8]): [[UInt8]] {
          return RLP.decodeList(data)
      }
    `)

	type testCase struct {
		name           string
		input          []cadence.Value
		output         [][]cadence.Value
		expectedErrMsg string
	}

	tests := []testCase{
		{
			name:           "empty input",
			input:          []cadence.Value{},
			output:         nil,
			expectedErrMsg: "failed to RLP-decode list: input data is empty",
		},
		{
			name: "empty list",
			input: []cadence.Value{
				cadence.UInt8(192),
			},
			output: [][]cadence.Value{},
		},

		{
			name: "single element list",
			input: []cadence.Value{
				cadence.UInt8(193),
				cadence.UInt8(65),
			},
			output: [][]cadence.Value{
				{
					cadence.UInt8('A'),
				},
			},
		},
		{
			name: "single element list with trailing extra bytes",
			input: []cadence.Value{
				cadence.UInt8(193),
				cadence.UInt8(65),
				cadence.UInt8(65), // extra byte
			},
			output:         nil,
			expectedErrMsg: "failed to RLP-decode list: input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes.",
		},
		{
			name: "multiple member list",
			input: []cadence.Value{
				cadence.UInt8(200),
				cadence.UInt8(131),
				cadence.UInt8(65),
				cadence.UInt8(66),
				cadence.UInt8(67),
				cadence.UInt8(131),
				cadence.UInt8(69),
				cadence.UInt8(70),
				cadence.UInt8(71),
			},
			output: [][]cadence.Value{
				{
					cadence.UInt8(131),
					cadence.UInt8(65),
					cadence.UInt8(66),
					cadence.UInt8(67),
				},
				{
					cadence.UInt8(131),
					cadence.UInt8(69),
					cadence.UInt8(70),
					cadence.UInt8(71),
				},
			},
		},
		{
			name: "multiple member list with an extra trailing byte",
			input: []cadence.Value{
				cadence.UInt8(200),
				cadence.UInt8(131),
				cadence.UInt8(65),
				cadence.UInt8(66),
				cadence.UInt8(67),
				cadence.UInt8(131),
				cadence.UInt8(69),
				cadence.UInt8(70),
				cadence.UInt8(71),
				cadence.UInt8(55),
			},
			output:         nil,
			expectedErrMsg: "failed to RLP-decode list: input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes.",
		},
	}

	test := func(test testCase) {
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			runtimeInterface := &testRuntimeInterface{
				storage: newTestLedger(nil, nil),
				meterMemory: func(_ common.MemoryUsage) error {
					return nil
				},
			}
			runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(runtimeInterface, b)
			}

			result, err := runtime.ExecuteScript(
				Script{
					Source: script,
					Arguments: encodeArgs([]cadence.Value{
						cadence.Array{
							ArrayType: cadence.VariableSizedArrayType{
								ElementType: cadence.UInt8Type{},
							},
							Values: test.input,
						},
					}),
				},
				Context{
					Interface: runtimeInterface,
					Location:  utils.TestLocation,
				},
			)
			if len(test.expectedErrMsg) > 0 {
				require.Error(t, err)
				_ = err.Error()

				assert.ErrorContains(t, err, test.expectedErrMsg)
			} else {
				require.NoError(t, err)

				arrays := make([]cadence.Value, 0, len(test.output))
				for _, values := range test.output {
					arrays = append(arrays,
						cadence.Array{Values: values}.
							WithType(cadence.VariableSizedArrayType{
								ElementType: cadence.UInt8Type{},
							}))
				}

				assert.Equal(t,
					cadence.Array{
						Values: arrays,
					}.WithType(cadence.VariableSizedArrayType{
						ElementType: cadence.VariableSizedArrayType{
							ElementType: cadence.UInt8Type{},
						},
					}),
					result,
				)
			}
		})
	}

	for _, testCase := range tests {
		test(testCase)
	}
}
