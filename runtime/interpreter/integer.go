/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

import "github.com/onflow/cadence/runtime/errors"

var cachedIntegerValues map[StaticType]map[int8]IntegerValue

func init() {
	cachedIntegerValues = make(map[StaticType]map[int8]IntegerValue)
}

// Get the provided int8 value in the required staticType.
// Note: Assumes that the provided value fits within the constraints of the staticType.
func GetValueForIntegerType(value int8, staticType StaticType) IntegerValue {
	typeCache, ok := cachedIntegerValues[staticType]
	if !ok {
		typeCache = make(map[int8]IntegerValue)
		cachedIntegerValues[staticType] = typeCache
	}

	val, ok := typeCache[value]
	if !ok {
		val = getValueForIntegerType(value, staticType)
		typeCache[value] = val
	}

	return val
}

// It is important to not meter the memory usage in this function, as it would lead to
// non-determinism as the values produced by this function are cached.
// It could happen that on some execution nodes the value might be cached due to executing a
// transaction or script that needed the value previously, while on other execution nodes it might
// not be cached yet.
func getValueForIntegerType(value int8, staticType StaticType) IntegerValue {
	switch staticType {
	case PrimitiveStaticTypeInt:
		return NewUnmeteredIntValueFromInt64(int64(value))
	case PrimitiveStaticTypeInt8:
		return NewUnmeteredInt8Value(value)
	case PrimitiveStaticTypeInt16:
		return NewUnmeteredInt16Value(int16(value))
	case PrimitiveStaticTypeInt32:
		return NewUnmeteredInt32Value(int32(value))
	case PrimitiveStaticTypeInt64:
		return NewUnmeteredInt64Value(int64(value))
	case PrimitiveStaticTypeInt128:
		return NewUnmeteredInt128ValueFromInt64(int64(value))
	case PrimitiveStaticTypeInt256:
		return NewUnmeteredInt256ValueFromInt64(int64(value))

	case PrimitiveStaticTypeUInt:
		return NewUnmeteredUIntValueFromUint64(uint64(value))
	case PrimitiveStaticTypeUInt8:
		return NewUnmeteredUInt8Value(uint8(value))
	case PrimitiveStaticTypeUInt16:
		return NewUnmeteredUInt16Value(uint16(value))
	case PrimitiveStaticTypeUInt32:
		return NewUnmeteredUInt32Value(uint32(value))
	case PrimitiveStaticTypeUInt64:
		return NewUnmeteredUInt64Value(uint64(value))
	case PrimitiveStaticTypeUInt128:
		return NewUnmeteredUInt128ValueFromUint64(uint64(value))
	case PrimitiveStaticTypeUInt256:
		return NewUnmeteredUInt256ValueFromUint64(uint64(value))

	case PrimitiveStaticTypeWord8:
		return NewUnmeteredWord8Value(uint8(value))
	case PrimitiveStaticTypeWord16:
		return NewUnmeteredWord16Value(uint16(value))
	case PrimitiveStaticTypeWord32:
		return NewUnmeteredWord32Value(uint32(value))
	case PrimitiveStaticTypeWord64:
		return NewUnmeteredWord64Value(uint64(value))
	case PrimitiveStaticTypeWord128:
		return NewUnmeteredWord128ValueFromUint64(uint64(value))
	case PrimitiveStaticTypeWord256:
		return NewUnmeteredWord256ValueFromUint64(uint64(value))

	default:
		panic(errors.NewUnreachableError())
	}
}
