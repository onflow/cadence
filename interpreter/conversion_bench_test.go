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

	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/interpreter"
)

func BenchmarkByteArrayValueToByteSlice(b *testing.B) {
	const elementCount = 32

	inter := newTestInterpreter(b)

	elements := make([]Value, elementCount)
	for i := range elements {
		elements[i] = NewUnmeteredUInt8Value(uint8(i))
	}

	value := NewArrayValue(
		inter,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeUInt8,
		},
		common.ZeroAddress,
		elements...,
	)

	var result []byte

	for b.Loop() {
		result, _ = ByteArrayValueToByteSlice(inter, value)
	}

	_ = result
}

func BenchmarkByteSliceToByteArrayValue(b *testing.B) {
	const elementCount = 32

	inter := newTestInterpreter(b)

	data := make([]byte, elementCount)
	for i := range data {
		data[i] = uint8(i)
	}

	var result *ArrayValue

	for b.Loop() {
		result = ByteSliceToByteArrayValue(inter, data)
	}

	_ = result
}
