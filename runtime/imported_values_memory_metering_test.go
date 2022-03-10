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
	"fmt"
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

	type importTest struct {
		TypeName     string
		MemoryKind   common.MemoryKind
		Weight       uint64
		TypeInstance cadence.Value
	}

	tests := []importTest{
		{TypeName: "String", MemoryKind: common.MemoryKindString, Weight: 7 + 1, TypeInstance: cadence.String("forever")},
		{TypeName: "Character", MemoryKind: common.MemoryKindCharacter, Weight: 1, TypeInstance: cadence.Character("a")},
		{TypeName: "Type", MemoryKind: common.MemoryKindTypeValue, Weight: 3, TypeInstance: cadence.TypeValue{StaticType: cadence.AnyType{}}},
		{TypeName: "Bool", MemoryKind: common.MemoryKindBool, Weight: 1, TypeInstance: cadence.Bool(true)},
		{TypeName: "Address", MemoryKind: common.MemoryKindAddress, Weight: 8, TypeInstance: cadence.Address{}},
		{TypeName: "Path", MemoryKind: common.MemoryKindPathValue, Weight: 3, TypeInstance: cadence.Path{Domain: "storage", Identifier: "id3"}},

		// Verify Capability and its composing values, Path and Type.
		{TypeName: "Capability", MemoryKind: common.MemoryKindCapabilityValue, Weight: 1, TypeInstance: cadence.Capability{
			Path:       cadence.Path{Domain: "public", Identifier: "foobarrington"},
			Address:    cadence.Address{},
			BorrowType: cadence.ReferenceType{Authorized: true, Type: cadence.AnyType{}},
		}},
		{TypeName: "Capability", MemoryKind: common.MemoryKindPathValue, Weight: 13, TypeInstance: cadence.Capability{
			Path:       cadence.Path{Domain: "public", Identifier: "foobarrington"},
			Address:    cadence.Address{},
			BorrowType: cadence.ReferenceType{Authorized: true, Type: cadence.AnyType{}},
		}},
		{TypeName: "Capability", MemoryKind: common.MemoryKindTypeValue, Weight: 3, TypeInstance: cadence.Capability{
			Path:       cadence.Path{Domain: "public", Identifier: "foobarrington"},
			Address:    cadence.Address{},
			BorrowType: cadence.ReferenceType{Authorized: true, Type: cadence.AnyType{}},
		}},

		// Verify Optional and its composing type
		{TypeName: "String?", MemoryKind: common.MemoryKindOptional, Weight: 1, TypeInstance: cadence.NewOptional(cadence.String("hello"))},
		{TypeName: "String?", MemoryKind: common.MemoryKindString, Weight: 5 + 1, TypeInstance: cadence.NewOptional(cadence.String("hello"))},

		// Not importable: Void
		// Not a user-visible type (not in BaseTypeActivation): Link
		// TODO: Bytes, U?Int\d*, Word\d+, U?Fix64, Array, Dictionary, Struct, Resource, Event, Contract, Enum
	}
	for _, test := range tests {
		t.Run(test.TypeName, func(test importTest) func(t *testing.T) {
			return func(t *testing.T) {
				t.Parallel()

				meter := make(map[common.MemoryKind]uint64)
				runtimeInterface := &testRuntimeInterface{
					useMemory: testUseMemory(meter),
					decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
						return jsoncdc.Decode(b)
					},
				}

				script := []byte(fmt.Sprintf(`
            	pub fun main(x: %s) {}
        	`, test.TypeName))

				_, err := runtime.ExecuteScript(
					Script{
						Source: script,
						Arguments: encodeArgs([]cadence.Value{
							test.TypeInstance,
						}),
					},
					Context{
						Interface: runtimeInterface,
						Location:  utils.TestLocation,
					},
				)

				require.NoError(t, err)

				assert.Equal(t, test.Weight, meter[test.MemoryKind])
			}
		}(test))
	}
}
