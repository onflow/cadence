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

package interpreter

import (
	"fmt"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

// Block

var blockStaticType StaticType = PrimitiveStaticTypeBlock // unmetered
var blockFieldNames = []string{
	sema.BlockTypeHeightFieldName,
	sema.BlockTypeViewFieldName,
	sema.BlockTypeIdFieldName,
	sema.BlockTypeTimestampFieldName,
}
var blockFieldFormatters = func(inter *Interpreter) map[string]func(common.MemoryGauge, Value, SeenReferences) string {
	return map[string]func(common.MemoryGauge, Value, SeenReferences) string{
		sema.BlockTypeIdFieldName: func(memoryGauge common.MemoryGauge, value Value, references SeenReferences) string {
			bytes, err := ByteArrayValueToByteSlice(inter, value, EmptyLocationRange)
			if err != nil {
				panic(err)
			}

			common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(bytes)*2+2))
			return fmt.Sprintf("0x%x", bytes)
		},
	}
}

func NewBlockValue(
	inter *Interpreter,
	height UInt64Value,
	view UInt64Value,
	id *ArrayValue,
	timestamp UFix64Value,
) *SimpleCompositeValue {
	return NewSimpleCompositeValue(
		inter,
		sema.BlockType.TypeID,
		blockStaticType,
		blockFieldNames,
		map[string]Value{
			sema.BlockTypeHeightFieldName:    height,
			sema.BlockTypeViewFieldName:      view,
			sema.BlockTypeIdFieldName:        id,
			sema.BlockTypeTimestampFieldName: timestamp,
		},
		nil,
		blockFieldFormatters(inter),
		nil,
	)
}
