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
	"sync"

	"github.com/onflow/cadence/errors"
)

func GetSmallIntegerValue(value int8, staticType StaticType) IntegerValue {
	return cachedSmallIntegerValues.Get(value, staticType)
}

type integerValueCacheKey struct {
	value      int8
	staticType StaticType
}

type smallIntegerValueCache struct {
	m sync.Map
}

var cachedSmallIntegerValues = smallIntegerValueCache{}

func (c *smallIntegerValueCache) Get(value int8, staticType StaticType) IntegerValue {
	key := integerValueCacheKey{
		value:      value,
		staticType: staticType,
	}

	existingValue, ok := c.m.Load(key)
	if ok {
		return existingValue.(IntegerValue)
	}

	newValue := c.new(value, staticType)
	c.m.Store(key, newValue)
	return newValue
}

// getValueForIntegerType returns a Cadence integer value
// of the given Cadence static type for the given Go integer value.
//
// It is important NOT to meter the memory usage in this function,
// as it would lead to non-determinism as the values produced by this function are cached.
// It could happen that on some execution nodes the value might be cached due to executing a
// transaction or script that needed the value previously, while on other execution nodes it might
// not be cached yet.
func (c *smallIntegerValueCache) new(value int8, staticType StaticType) IntegerValue {
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
