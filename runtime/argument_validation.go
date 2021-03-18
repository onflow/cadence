package runtime

import (
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// Utility functions for validating the arguments passed into a script or a transaction.

// valueConformsToDynamicType checks whether the shape of the value is same as the shape
// defined by the type associated with the value. Returns `true` if the value and the
// type matches, and `false` otherwise.
//
// e.g: A value with the type as 'Foo', but having additional fields than what is
// specified in 'Foo', would return `false`.
//
func valueConformsToDynamicType(value interpreter.Value, dynamicType interpreter.DynamicType) (ok bool) {
	switch typ := dynamicType.(type) {
	case interpreter.ArrayDynamicType:
		ok = valueConfirmsToArrayDynamicType(value, typ)
	case interpreter.CompositeDynamicType:
		ok = valueConformsToSemaType(value, typ.StaticType)
	case interpreter.DictionaryDynamicType:
		ok = valueConfirmsToDictionaryDynamicType(value, typ)
	case interpreter.SomeDynamicType:
		ok = valueConfirmsToSomeDynamicType(value, typ)
	case interpreter.PrivatePathDynamicType, interpreter.PublicPathDynamicType, interpreter.StoragePathDynamicType:
		_, ok = value.(interpreter.PathValue)
	case interpreter.CapabilityDynamicType:
		_, ok = value.(interpreter.CapabilityValue)
	case interpreter.BlockDynamicType:
		_, ok = value.(interpreter.BlockValue)

	// Following types are guaranteed to be decoded correctly by the json.decode().
	// However, this additional layer is added to protect the Cadence runtime,
	// since the decoding currently relies on the host env, and it is external to Cadence.
	case interpreter.StringDynamicType:
		_, ok = value.(*interpreter.StringValue)
	case interpreter.BoolDynamicType:
		_, ok = value.(*interpreter.BoolValue)
	case interpreter.NumberDynamicType:
		ok = valueConformsToSemaType(value, typ.StaticType)
	case interpreter.NilDynamicType:
		_, ok = value.(interpreter.NilValue)
	case interpreter.AddressDynamicType:
		_, ok = value.(interpreter.AddressValue)
	case interpreter.MetaTypeDynamicType:
		_, ok = value.(interpreter.TypeValue)

	// Following types cannot be used as arguments to a script/transaction.
	// However, still validate and allow wherever possible, so that this validation
	// is less conservative and would seamlessly cater future changes.
	case interpreter.FunctionDynamicType:
		_, ok = value.(interpreter.FunctionValue)
	case interpreter.DeployedContractDynamicType:
		_, ok = value.(interpreter.DeployedContractValue)
	case interpreter.StorageReferenceDynamicType:
		// TODO: check for the referenced value conformance, if and when
		// importing a storage reference value support is added.
		_, ok = value.(*interpreter.StorageReferenceValue)
	case interpreter.EphemeralReferenceDynamicType:
		ok = valueConfirmsToEphemeralReferenceDynamicType(value, typ)
	case interpreter.VoidDynamicType:
		// Void type cannot have a value.
		ok = false
	}

	return
}

func valueConfirmsToArrayDynamicType(value interpreter.Value, arrayType interpreter.ArrayDynamicType) bool {
	arrayValue, ok := value.(*interpreter.ArrayValue)
	if !ok || len(arrayValue.Values) != len(arrayType.ElementTypes) {
		return false
	}

	for index, item := range arrayValue.Values {
		if !valueConformsToDynamicType(item, arrayType.ElementTypes[index]) {
			return false
		}
	}

	return true
}

func valueConfirmsToDictionaryDynamicType(value interpreter.Value, dictionaryType interpreter.DictionaryDynamicType) bool {
	dictionaryValue, ok := value.(*interpreter.DictionaryValue)
	if !ok || len(dictionaryValue.Keys.Values) != len(dictionaryType.EntryTypes) {
		return false
	}

	for index, entryKey := range dictionaryValue.Keys.Values {
		entryType := dictionaryType.EntryTypes[index]

		// Check the key
		if !valueConformsToDynamicType(entryKey, entryType.KeyType) {
			return false
		}

		// Check the value. Here it is assumed an imported value can only have
		// static entries, but not deferred keys/values.
		key := interpreter.DictionaryKey(entryKey)
		entryValue, ok := dictionaryValue.Entries.Get(key)
		if !ok || !valueConformsToDynamicType(entryValue, entryType.ValueType) {
			return false
		}
	}

	return true
}

func valueConfirmsToSomeDynamicType(value interpreter.Value, someType interpreter.SomeDynamicType) bool {
	someValue, ok := value.(*interpreter.SomeValue)
	if !ok {
		return false
	}

	return valueConformsToDynamicType(someValue.Value, someType.InnerType)
}

func valueConfirmsToEphemeralReferenceDynamicType(value interpreter.Value, refType interpreter.EphemeralReferenceDynamicType) bool {
	referenceValue, ok := value.(*interpreter.EphemeralReferenceValue)
	if !ok {
		return false
	}

	return valueConformsToDynamicType(*referenceValue.ReferencedValue(), refType.InnerType())
}

// valueConformsToSemaType checks whether a value conforms to a given semantic type.
// Returns `true` if the value and the type matches, and `false` otherwise.
//
func valueConformsToSemaType(value interpreter.Value, semaType sema.Type) (ok bool) {
	switch typ := semaType.(type) {
	case *sema.SimpleType:
		ok = valueConformsToSimpleType(value, typ)
	case *sema.IntType:
		_, ok = value.(interpreter.IntValue)
	case *sema.Int8Type:
		_, ok = value.(interpreter.Int8Value)
	case *sema.Int16Type:
		_, ok = value.(interpreter.Int16Value)
	case *sema.Int32Type:
		_, ok = value.(interpreter.Int32Value)
	case *sema.Int64Type:
		_, ok = value.(interpreter.Int64Value)
	case *sema.Int128Type:
		_, ok = value.(interpreter.Int128Value)
	case *sema.Int256Type:
		_, ok = value.(interpreter.Int256Value)
	case *sema.UIntType:
		_, ok = value.(interpreter.UIntValue)
	case *sema.UInt8Type:
		_, ok = value.(interpreter.UInt8Value)
	case *sema.UInt16Type:
		_, ok = value.(interpreter.UInt16Value)
	case *sema.UInt32Type:
		_, ok = value.(interpreter.UInt32Value)
	case *sema.UInt64Type:
		_, ok = value.(interpreter.UInt64Value)
	case *sema.UInt128Type:
		_, ok = value.(interpreter.UInt128Value)
	case *sema.UInt256Type:
		_, ok = value.(interpreter.UInt256Value)
	case *sema.Word8Type:
		_, ok = value.(interpreter.Word8Value)
	case *sema.Word16Type:
		_, ok = value.(interpreter.Word16Value)
	case *sema.Word32Type:
		_, ok = value.(interpreter.Word32Value)
	case *sema.Word64Type:
		_, ok = value.(interpreter.Word64Value)
	case *sema.Fix64Type:
		_, ok = value.(interpreter.Fix64Value)
	case *sema.UFix64Type:
		_, ok = value.(interpreter.UFix64Value)
	case *sema.OptionalType:
		// Value must be `nil, or must be of type defined in the optional type.
		if _, ok = value.(interpreter.NilValue); !ok {
			ok = valueConformsToSemaType(value, typ.Type)
		}
	case *sema.AddressType:
		_, ok = value.(*interpreter.AddressValue)
	case *sema.VariableSizedType:
		ok = valueConformsToArrayType(value, typ.Type)
	case *sema.ConstantSizedType:
		ok = valueConformsToArrayType(value, typ.Type)
	case *sema.DictionaryType:
		ok = valueConformsToDictionaryType(value, typ)
	case *sema.CompositeType:
		ok = valueConformsToCompositeType(value, typ)
	case *sema.CapabilityType:
		_, ok = value.(*interpreter.CapabilityValue)

	// Following types cannot be used as arguments to a script/transaction
	case *sema.FunctionType:
		_, ok = value.(interpreter.FunctionValue)
	case *sema.TransactionType:
		panic(errors.NewUnreachableError())
	}

	return
}

func valueConformsToCompositeType(arg interpreter.Value, compositeType *sema.CompositeType) bool {
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

		if !valueConformsToSemaType(field, member.TypeAnnotation.Type) {
			return false
		}
	}

	return true
}

func valueConformsToSimpleType(value interpreter.Value, simpleType *sema.SimpleType) (ok bool) {
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
	case sema.MetaType:
		_, ok = value.(interpreter.TypeValue)
	}

	return
}

func valueConformsToArrayType(value interpreter.Value, memberType sema.Type) bool {
	arrayValue, ok := value.(*interpreter.ArrayValue)
	if !ok {
		return false
	}

	for _, member := range arrayValue.Values {
		if !valueConformsToSemaType(member, memberType) {
			return false
		}
	}

	return true
}

func valueConformsToDictionaryType(value interpreter.Value, dictionaryType *sema.DictionaryType) bool {
	dictionaryValue, ok := value.(*interpreter.DictionaryValue)
	if !ok {
		return false
	}

	for _, entryKey := range dictionaryValue.Keys.Values {
		// Check the key
		if !valueConformsToSemaType(entryKey, dictionaryType.KeyType) {
			return false
		}

		// Check the value. Here it is assumed an imported value can only have
		// static entries, but not deferred keys/values.
		key := interpreter.DictionaryKey(entryKey)
		entryValue, ok := dictionaryValue.Entries.Get(key)
		if !ok || !valueConformsToSemaType(entryValue, dictionaryType.ValueType) {
			return false
		}
	}

	return true
}
