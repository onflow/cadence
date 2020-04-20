/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package cadence

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// ConvertValue converts a runtime value to its native Go representation.
func ConvertValue(value runtime.Value) Value {
	return convertValue(value.Value, value.Interpreter())
}

// ConvertEvent converts a runtime event to its native Go representation.
func ConvertEvent(event runtime.Event) Event {
	fields := make([]Value, len(event.Fields))

	for i, field := range event.Fields {
		fields[i] = convertValue(field.Value, field.Interpreter())
	}

	return NewEvent(fields).WithType(ConvertType(event.Type).(EventType))
}

func convertValue(value interpreter.Value, inter *interpreter.Interpreter) Value {
	switch v := value.(type) {
	case interpreter.VoidValue:
		return NewVoid()
	case interpreter.NilValue:
		return NewOptional(nil)
	case *interpreter.SomeValue:
		return convertSomeValue(v, inter)
	case interpreter.BoolValue:
		return NewBool(bool(v))
	case *interpreter.StringValue:
		return NewString(v.Str)
	case *interpreter.ArrayValue:
		return convertArrayValue(v, inter)
	case interpreter.IntValue:
		return NewIntFromBig(big.NewInt(0).Set(v.Int))
	case interpreter.Int8Value:
		return NewInt8(int8(v))
	case interpreter.Int16Value:
		return NewInt16(int16(v))
	case interpreter.Int32Value:
		return NewInt32(int32(v))
	case interpreter.Int64Value:
		return NewInt64(int64(v))
	case interpreter.Int128Value:
		return NewInt128FromBig(big.NewInt(0).Set(v.Int))
	case interpreter.Int256Value:
		return NewInt256FromBig(big.NewInt(0).Set(v.Int))
	case interpreter.UIntValue:
		return NewUIntFromBig(big.NewInt(0).Set(v.Int))
	case interpreter.UInt8Value:
		return NewUInt8(uint8(v))
	case interpreter.UInt16Value:
		return NewUInt16(uint16(v))
	case interpreter.UInt32Value:
		return NewUInt32(uint32(v))
	case interpreter.UInt64Value:
		return NewUInt64(uint64(v))
	case interpreter.UInt128Value:
		return NewUInt128FromBig(big.NewInt(0).Set(v.Int))
	case interpreter.UInt256Value:
		return NewUInt256FromBig(big.NewInt(0).Set(v.Int))
	case interpreter.Word8Value:
		return NewWord8(uint8(v))
	case interpreter.Word16Value:
		return NewWord16(uint16(v))
	case interpreter.Word32Value:
		return NewWord32(uint32(v))
	case interpreter.Word64Value:
		return NewWord64(uint64(v))
	case interpreter.Fix64Value:
		return NewFix64(int64(v))
	case interpreter.UFix64Value:
		return NewUFix64(uint64(v))
	case *interpreter.CompositeValue:
		return convertCompositeValue(v, inter)
	case *interpreter.DictionaryValue:
		return convertDictionaryValue(v, inter)
	case interpreter.AddressValue:
		return NewAddress(v)
	}

	panic(fmt.Sprintf("cannot convert value of type %T", value))
}

func convertSomeValue(v *interpreter.SomeValue, inter *interpreter.Interpreter) Value {
	if v.Value == nil {
		return NewOptional(nil)
	}

	value := convertValue(v.Value, inter)

	return NewOptional(value)
}

func convertArrayValue(v *interpreter.ArrayValue, inter *interpreter.Interpreter) Value {
	values := make([]Value, len(v.Values))

	for i, value := range v.Values {
		values[i] = convertValue(value, inter)
	}

	return NewArray(values)
}

func convertCompositeValue(v *interpreter.CompositeValue, inter *interpreter.Interpreter) Value {
	fields := make([]Value, len(v.Fields))

	keys := make([]string, 0, len(v.Fields))
	for key := range v.Fields {
		keys = append(keys, key)
	}

	// sort keys in lexicographical order
	sort.Strings(keys)

	for i, key := range keys {
		field := v.Fields[key]
		fields[i] = convertValue(field, inter)
	}

	dynamicType := v.DynamicType(inter).(interpreter.CompositeDynamicType)
	staticType := dynamicType.StaticType.(*sema.CompositeType)

	t := ConvertType(staticType)

	switch staticType.Kind {
	case common.CompositeKindStructure:
		return NewStruct(fields).WithType(t.(StructType))
	case common.CompositeKindResource:
		return NewResource(fields).WithType(t.(ResourceType))
	case common.CompositeKindEvent:
		return NewEvent(fields).WithType(t.(EventType))
	}

	panic(fmt.Errorf("invalid composite kind `%s`, must be Struct, Resource or Event", staticType.Kind))
}

func convertDictionaryValue(v *interpreter.DictionaryValue, inter *interpreter.Interpreter) Value {
	pairs := make([]KeyValuePair, v.Count())

	for i, keyValue := range v.Keys.Values {
		key := keyValue.(interpreter.HasKeyString).KeyString()
		value := v.Entries[key]

		convertedKey := convertValue(keyValue, inter)
		convertedValue := convertValue(value, inter)

		pairs[i] = KeyValuePair{
			Key:   convertedKey,
			Value: convertedValue,
		}
	}

	return NewDictionary(pairs)
}
