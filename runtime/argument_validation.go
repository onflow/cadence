package runtime

import (
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// valueConformsToType checks whether the shape of the value is same as the shape
// defined by the type associated with the value. Returns `true` if the value and the
// type matches, and `false` otherwise.
//
// e.g: A value with the type as 'Foo', but having additional fields than what is
// specified in 'Foo', would return `false`.
func valueConformsToType(value interpreter.Value, semaType sema.Type) (ok bool) {
	switch typ := semaType.(type) {
	case *sema.OptionalType:
		ok = valueConformsToType(value, typ.Type)
	case *sema.AddressType:
		_, ok = value.(*interpreter.AddressValue)
	case *sema.VariableSizedType:
		ok = checkArrayTypedValueConformance(value, typ.Type)
	case *sema.ConstantSizedType:
		ok = checkArrayTypedValueConformance(value, typ.Type)
	case *sema.DictionaryType:
		ok = checkDictionaryTypedValueConformance(value, typ)
	case *sema.CompositeType:
		ok = checkCompositeTypedValueConformance(value, typ)

	// Below types are guaranteed to be decoded correctly by the json.decode().
	// However, this additional layer is added to protect the Cadence runtime,
	// since the decoding currently relies on the host env, and it is external to Cadence.
	case *sema.SimpleType:
		ok = checkSimpleTypedValueConformance(value, typ)
	case *sema.IntType:
		_, ok = value.(interpreter.IntValue)
	case *sema.Int8Type:
		_, ok = value.(*interpreter.Int8Value)
	case *sema.Int16Type:
		_, ok = value.(*interpreter.Int16Value)
	case *sema.Int32Type:
		_, ok = value.(*interpreter.Int32Value)
	case *sema.Int64Type:
		_, ok = value.(*interpreter.Int64Value)
	case *sema.Int128Type:
		_, ok = value.(*interpreter.Int128Value)
	case *sema.Int256Type:
		_, ok = value.(*interpreter.Int256Value)
	case *sema.UIntType:
		_, ok = value.(*interpreter.UIntValue)
	case *sema.UInt8Type:
		_, ok = value.(*interpreter.UInt8Value)
	case *sema.UInt16Type:
		_, ok = value.(*interpreter.UInt16Value)
	case *sema.UInt32Type:
		_, ok = value.(*interpreter.UInt32Value)
	case *sema.UInt64Type:
		_, ok = value.(*interpreter.UInt64Value)
	case *sema.UInt128Type:
		_, ok = value.(*interpreter.UInt128Value)
	case *sema.UInt256Type:
		_, ok = value.(*interpreter.UInt256Value)
	case *sema.Word8Type:
		_, ok = value.(*interpreter.Word8Value)
	case *sema.Word16Type:
		_, ok = value.(*interpreter.Word16Value)
	case *sema.Word32Type:
		_, ok = value.(*interpreter.Word32Value)
	case *sema.Word64Type:
		_, ok = value.(*interpreter.Word64Value)
	case *sema.Fix64Type:
		_, ok = value.(*interpreter.Fix64Value)
	case *sema.UFix64Type:
		_, ok = value.(*interpreter.UFix64Value)
	}

	return
}

func checkCompositeTypedValueConformance(arg interpreter.Value, compositeType *sema.CompositeType) bool {
	compositeValue, ok := arg.(*interpreter.CompositeValue)
	if !ok || compositeValue.Kind != compositeType.Kind {
		return false
	}

	// Here it is assumed that imported values can only have static fields values,
	// but not computed field values.
	if compositeValue.Fields.Len() != len(compositeType.Fields) {
		return false
	}

	for _, fieldName := range compositeType.Fields {
		field, ok := compositeValue.Fields.Get(fieldName)
		if !ok {
			return false
		}

		member, ok := compositeType.Members.Get(fieldName)
		if !ok {
			return false
		}

		if !valueConformsToType(field, member.TypeAnnotation.Type) {
			return false
		}
	}

	return true
}

func checkSimpleTypedValueConformance(value interpreter.Value, simpleType *sema.SimpleType) (ok bool) {
	switch simpleType {
	case sema.StringType:
		_, ok = value.(*interpreter.StringValue)
	case sema.BoolType:
		_, ok = value.(*interpreter.BoolValue)
	case sema.PathType,
		sema.PublicPathType,
		sema.CapabilityPathType,
		sema.PrivatePathType,
		sema.StoragePathType:

		_, ok = value.(*interpreter.PathValue)

	case sema.VoidType:
		_, ok = value.(*interpreter.BoolValue)
	}

	return
}

func checkArrayTypedValueConformance(value interpreter.Value, memberType sema.Type) bool {
	arrayValue, ok := value.(*interpreter.ArrayValue)
	if !ok {
		return false
	}

	for _, member := range arrayValue.Values {
		if !valueConformsToType(member, memberType) {
			return false
		}
	}

	return true
}

func checkDictionaryTypedValueConformance(value interpreter.Value, dictionaryType *sema.DictionaryType) bool {
	dictionaryValue, ok := value.(*interpreter.DictionaryValue)
	if !ok {
		return false
	}

	for _, entryKey := range dictionaryValue.Keys.Values {
		if !valueConformsToType(entryKey, dictionaryType.KeyType) {
			return false
		}

		key := interpreter.DictionaryKey(entryKey)
		entryValue, ok := dictionaryValue.Entries.Get(key)
		if !ok || !valueConformsToType(entryValue, dictionaryType.ValueType) {
			return false
		}
	}

	return true
}
