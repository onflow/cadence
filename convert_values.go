package cadence

import (
	"fmt"
	"sort"

	"github.com/dapperlabs/cadence/runtime"
	"github.com/dapperlabs/cadence/runtime/interpreter"
)

// ConvertValue converts a runtime value to its native Go representation.
func ConvertValue(value runtime.Value) Value {
	return convertValue(value.Value, value.Interpreter())
}

func convertValue(value interpreter.Value, inter *interpreter.Interpreter) Value {
	switch v := value.(type) {
	case interpreter.VoidValue:
		return NewVoid()
	case interpreter.NilValue:
		return NewNil()
	case *interpreter.SomeValue:
		return convertSomeValue(v, inter)
	case interpreter.BoolValue:
		return NewBool(bool(v))
	case *interpreter.StringValue:
		return NewString(v.Str)
	case *interpreter.ArrayValue:
		return convertArrayValue(v, inter)
	case interpreter.IntValue:
		return NewIntFromBig(v.Int)
	case interpreter.Int8Value:
		return NewInt8(int8(v))
	case interpreter.Int16Value:
		return NewInt16(int16(v))
	case interpreter.Int32Value:
		return NewInt32(int32(v))
	case interpreter.Int64Value:
		return NewInt64(int64(v))
	case interpreter.Int128Value:
		return NewInt128FromBig(v.Int)
	case interpreter.Int256Value:
		return NewInt256FromBig(v.Int)
	case interpreter.UIntValue:
		return NewUIntFromBig(v.Int)
	case interpreter.UInt8Value:
		return NewUInt8(uint8(v))
	case interpreter.UInt16Value:
		return NewUInt16(uint16(v))
	case interpreter.UInt32Value:
		return NewUInt32(uint32(v))
	case interpreter.UInt64Value:
		return NewUInt64(uint64(v))
	case interpreter.UInt128Value:
		return NewUInt128FromBig(v.Int)
	case interpreter.UInt256Value:
		return NewUInt256FromBig(v.Int)
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

// ConvertEvent converts a runtime event to its native Go representation.
func ConvertEvent(event runtime.Event) Event {
	fields := make([]Value, len(event.Fields))

	for i, field := range event.Fields {
		fields[i] = convertValue(field, field.Interpreter())
	}

	return NewEvent(fields)
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

	dynamicType := v.DynamicType(inter).(interpreter.CompositeType)

	t := ConvertType(dynamicType.StaticType)

	return NewComposite(fields).WithType(t)
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
