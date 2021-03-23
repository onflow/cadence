/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/common"
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

	// Currently the type-vs-value mismatch could only happen for composite types, as
	// that's the only place where the type is loaded from the runtime, based on the name,
	// rather than based on the value structure of the user input.
	// Hence the possible set of malformed values are:
	//     - Composite types.
	//     - Arrays/Dictionaries/Optionals of composite types.

	switch typ := dynamicType.(type) {
	case interpreter.ArrayDynamicType:
		ok = valueConformsToArrayDynamicType(value, typ)
	case interpreter.CompositeDynamicType:
		ok = valueConformsToSemaType(value, typ.StaticType)
	case interpreter.DictionaryDynamicType:
		ok = valueConformsToDictionaryDynamicType(value, typ)
	case interpreter.SomeDynamicType:
		ok = valueConformsToSomeDynamicType(value, typ)

	// Following types are guaranteed to be decoded correctly by the json.decode().
	// However, this additional layer is added to protect the Cadence runtime,
	// since the decoding currently relies on the host env, and it is external to Cadence.
	case interpreter.StringDynamicType:
		_, ok = value.(*interpreter.StringValue)
	case interpreter.BoolDynamicType:
		_, ok = value.(interpreter.BoolValue)
	case interpreter.NumberDynamicType:
		ok = valueConformsToSemaType(value, typ.StaticType)
	case interpreter.NilDynamicType:
		_, ok = value.(interpreter.NilValue)
	case interpreter.AddressDynamicType:
		_, ok = value.(interpreter.AddressValue)
	case interpreter.MetaTypeDynamicType:
		_, ok = value.(interpreter.TypeValue)
	case interpreter.CapabilityDynamicType:
		_, ok = value.(interpreter.CapabilityValue)
	case interpreter.PrivatePathDynamicType, interpreter.PublicPathDynamicType, interpreter.StoragePathDynamicType:
		_, ok = value.(interpreter.PathValue)

	// Following types cannot be used as arguments to a script/transaction.
	// However, still validate and allow wherever possible, so that this validation
	// is less conservative and would seamlessly cater future changes.
	case interpreter.FunctionDynamicType:
		_, ok = value.(interpreter.FunctionValue)
	case interpreter.BlockDynamicType:
		_, ok = value.(interpreter.BlockValue)
	case interpreter.DeployedContractDynamicType:
		_, ok = value.(interpreter.DeployedContractValue)
	case interpreter.StorageReferenceDynamicType:
		// Currently this only checks whether the the value is `interpreter.StorageReferenceValue`.
		// It doesn't check whether the 'referenced' value conforms to the innerType.
		// TODO: add support for checking the referenced value conformance (if importing is supported).
		_, ok = value.(*interpreter.StorageReferenceValue)
	case interpreter.EphemeralReferenceDynamicType:
		ok = valueConformsToEphemeralReferenceDynamicType(value, typ)
	case interpreter.VoidDynamicType:
		// Void type cannot have a value.
		ok = false
	}

	return
}

func valueConformsToArrayDynamicType(value interpreter.Value, arrayType interpreter.ArrayDynamicType) bool {
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

func valueConformsToDictionaryDynamicType(
	value interpreter.Value,
	dictionaryType interpreter.DictionaryDynamicType,
) bool {

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

func valueConformsToSomeDynamicType(value interpreter.Value, someType interpreter.SomeDynamicType) bool {
	someValue, ok := value.(*interpreter.SomeValue)
	if !ok {
		return false
	}

	return valueConformsToDynamicType(someValue.Value, someType.InnerType)
}

func valueConformsToEphemeralReferenceDynamicType(
	value interpreter.Value,
	refType interpreter.EphemeralReferenceDynamicType,
) bool {

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
		_, ok = value.(interpreter.AddressValue)
	case *sema.VariableSizedType:
		ok = valueConformsToVariableSizedArrayType(value, typ)
	case *sema.ConstantSizedType:
		ok = valueConformsToConstantSizedArrayType(value, typ)
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
		// false
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
		_, ok = value.(interpreter.BoolValue)
	case sema.PublicPathType:
		ok = valueConformsToPathType(value, common.PathDomainPublic)
	case sema.PrivatePathType:
		ok = valueConformsToPathType(value, common.PathDomainPrivate)
	case sema.StoragePathType:
		ok = valueConformsToPathType(value, common.PathDomainStorage)
	case sema.PathType, sema.CapabilityPathType:
		_, ok = value.(interpreter.PathValue)
	case sema.MetaType:
		_, ok = value.(interpreter.TypeValue)
	case sema.VoidType:
		// false
	}

	return
}

func valueConformsToPathType(value interpreter.Value, domain common.PathDomain) bool {
	path, ok := value.(interpreter.PathValue)
	if !ok {
		return false
	}

	return path.Domain == domain
}

func valueConformsToVariableSizedArrayType(value interpreter.Value, arrayType *sema.VariableSizedType) bool {
	arrayValue, ok := value.(*interpreter.ArrayValue)
	if !ok {
		return false
	}

	return checkArrayMembers(arrayValue, arrayType.Type)
}

func valueConformsToConstantSizedArrayType(value interpreter.Value, arrayType *sema.ConstantSizedType) bool {
	arrayValue, ok := value.(*interpreter.ArrayValue)
	if !ok || int64(len(arrayValue.Values)) != arrayType.Size {
		return false
	}

	return checkArrayMembers(arrayValue, arrayType.Type)
}

func checkArrayMembers(arrayValue *interpreter.ArrayValue, memberType sema.Type) bool {
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

	for _, key := range dictionaryValue.Keys.Values {
		// Check the key
		if !valueConformsToSemaType(key, dictionaryType.KeyType) {
			return false
		}

		// Check the value. Here it is assumed an imported value can only have
		// static entries, but not deferred keys/values.
		dictionaryKey := interpreter.DictionaryKey(key)
		entryValue, ok := dictionaryValue.Entries.Get(dictionaryKey)
		if !ok || !valueConformsToSemaType(entryValue, dictionaryType.ValueType) {
			return false
		}
	}

	return true
}
