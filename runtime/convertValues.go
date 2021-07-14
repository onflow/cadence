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

package runtime

import (
	"fmt"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

// exportValue converts a runtime value to its native Go representation.
func exportValue(value exportableValue) (cadence.Value, error) {
	return exportValueWithInterpreter(value.Value, value.Interpreter(), exportResults{})
}

// ExportValue converts a runtime value to its native Go representation.
func ExportValue(value interpreter.Value, inter *interpreter.Interpreter) (cadence.Value, error) {
	return exportValueWithInterpreter(value, inter, exportResults{})
}

type exportResults map[*interpreter.EphemeralReferenceValue]struct{}

// exportValueWithInterpreter exports the given internal (interpreter) value to an external value.
//
// The export is recursive, the results parameter prevents cycles:
// it is checked at the start of the recursively called function,
// and pre-set before a recursive call.
//
func exportValueWithInterpreter(
	value interpreter.Value,
	inter *interpreter.Interpreter,
	results exportResults,
) (
	cadence.Value,
	error,
) {
	switch v := value.(type) {
	case interpreter.VoidValue:
		return cadence.NewVoid(), nil
	case interpreter.NilValue:
		return cadence.NewOptional(nil), nil
	case *interpreter.SomeValue:
		return exportSomeValue(v, inter, results)
	case interpreter.BoolValue:
		return cadence.NewBool(bool(v)), nil
	case *interpreter.StringValue:
		return cadence.NewString(v.Str)
	case *interpreter.ArrayValue:
		return exportArrayValue(v, inter, results)
	case interpreter.IntValue:
		return cadence.NewIntFromBig(v.ToBigInt()), nil
	case interpreter.Int8Value:
		return cadence.NewInt8(int8(v)), nil
	case interpreter.Int16Value:
		return cadence.NewInt16(int16(v)), nil
	case interpreter.Int32Value:
		return cadence.NewInt32(int32(v)), nil
	case interpreter.Int64Value:
		return cadence.NewInt64(int64(v)), nil
	case interpreter.Int128Value:
		return cadence.NewInt128FromBig(v.ToBigInt())
	case interpreter.Int256Value:
		return cadence.NewInt256FromBig(v.ToBigInt())
	case interpreter.UIntValue:
		return cadence.NewUIntFromBig(v.ToBigInt())
	case interpreter.UInt8Value:
		return cadence.NewUInt8(uint8(v)), nil
	case interpreter.UInt16Value:
		return cadence.NewUInt16(uint16(v)), nil
	case interpreter.UInt32Value:
		return cadence.NewUInt32(uint32(v)), nil
	case interpreter.UInt64Value:
		return cadence.NewUInt64(uint64(v)), nil
	case interpreter.UInt128Value:
		return cadence.NewUInt128FromBig(v.ToBigInt())
	case interpreter.UInt256Value:
		return cadence.NewUInt256FromBig(v.ToBigInt())
	case interpreter.Word8Value:
		return cadence.NewWord8(uint8(v)), nil
	case interpreter.Word16Value:
		return cadence.NewWord16(uint16(v)), nil
	case interpreter.Word32Value:
		return cadence.NewWord32(uint32(v)), nil
	case interpreter.Word64Value:
		return cadence.NewWord64(uint64(v)), nil
	case interpreter.Fix64Value:
		return cadence.Fix64(v), nil
	case interpreter.UFix64Value:
		return cadence.UFix64(v), nil
	case *interpreter.CompositeValue:
		return exportCompositeValue(v, inter, results)
	case *interpreter.DictionaryValue:
		return exportDictionaryValue(v, inter, results)
	case interpreter.AddressValue:
		return cadence.NewAddress(v), nil
	case interpreter.LinkValue:
		return exportLinkValue(v, inter), nil
	case interpreter.PathValue:
		return exportPathValue(v), nil
	case interpreter.TypeValue:
		return exportTypeValue(v, inter), nil
	case interpreter.CapabilityValue:
		return exportCapabilityValue(v, inter), nil
	case *interpreter.EphemeralReferenceValue:
		// Break recursion through ephemeral references
		if _, ok := results[v]; ok {
			return nil, nil
		}
		defer delete(results, v)
		results[v] = struct{}{}
		return exportValueWithInterpreter(v.Value, inter, results)
	case *interpreter.StorageReferenceValue:
		referencedValue := v.ReferencedValue(inter)
		if referencedValue == nil {
			return nil, nil
		}
		return exportValueWithInterpreter(*referencedValue, inter, results)
	}

	return nil, fmt.Errorf("cannot export value of type %T", value)

}

func exportSomeValue(
	v *interpreter.SomeValue,
	inter *interpreter.Interpreter,
	results exportResults,
) (
	cadence.Optional,
	error,
) {
	if v.Value == nil {
		return cadence.NewOptional(nil), nil
	}

	value, err := exportValueWithInterpreter(v.Value, inter, results)
	if err != nil {
		return cadence.Optional{}, err
	}

	return cadence.NewOptional(value), nil
}

func exportArrayValue(
	v *interpreter.ArrayValue,
	inter *interpreter.Interpreter,
	results exportResults,
) (
	cadence.Array,
	error,
) {
	elements := v.Elements()
	values := make([]cadence.Value, len(elements))

	for i, value := range elements {
		exportedValue, err := exportValueWithInterpreter(value, inter, results)
		if err != nil {
			return cadence.Array{}, err
		}
		values[i] = exportedValue
	}

	return cadence.NewArray(values), nil
}

func exportCompositeValue(
	v *interpreter.CompositeValue,
	inter *interpreter.Interpreter,
	results exportResults,
) (
	cadence.Value,
	error,
) {

	dynamicTypeResults := interpreter.DynamicTypeResults{}

	dynamicType := v.DynamicType(inter, dynamicTypeResults).(interpreter.CompositeDynamicType)
	staticType := dynamicType.StaticType.(*sema.CompositeType)
	// TODO: consider making the results map "global", by moving it up to exportValueWithInterpreter
	t := exportCompositeType(staticType, map[sema.TypeID]cadence.Type{})

	// NOTE: use the exported type's fields to ensure fields in type
	// and value are in sync

	fieldNames := t.CompositeFields()
	fields := make([]cadence.Value, len(fieldNames))

	fieldsMap := v.Fields()
	for i, field := range fieldNames {
		fieldName := field.Identifier
		fieldValue, ok := fieldsMap.Get(fieldName)

		if !ok && v.ComputedFields != nil {
			if computedField, ok := v.ComputedFields.Get(fieldName); ok {
				fieldValue = computedField(inter)
			}
		}

		exportedFieldValue, err := exportValueWithInterpreter(fieldValue, inter, results)
		if err != nil {
			return nil, err
		}
		fields[i] = exportedFieldValue
	}

	// NOTE: when modifying the cases below,
	// also update the error message below!

	switch staticType.Kind {
	case common.CompositeKindStructure:
		return cadence.NewStruct(fields).WithType(t.(*cadence.StructType)), nil
	case common.CompositeKindResource:
		return cadence.NewResource(fields).WithType(t.(*cadence.ResourceType)), nil
	case common.CompositeKindEvent:
		return cadence.NewEvent(fields).WithType(t.(*cadence.EventType)), nil
	case common.CompositeKindContract:
		return cadence.NewContract(fields).WithType(t.(*cadence.ContractType)), nil
	case common.CompositeKindEnum:
		return cadence.NewEnum(fields).WithType(t.(*cadence.EnumType)), nil
	}

	return nil, fmt.Errorf(
		"invalid composite kind `%s`, must be %s",
		staticType.Kind,
		common.EnumerateWords(
			[]string{
				common.CompositeKindStructure.Name(),
				common.CompositeKindResource.Name(),
				common.CompositeKindEvent.Name(),
				common.CompositeKindContract.Name(),
				common.CompositeKindEnum.Name(),
			},
			"or",
		),
	)
}

func exportDictionaryValue(
	v *interpreter.DictionaryValue,
	inter *interpreter.Interpreter,
	results exportResults,
) (
	cadence.Dictionary,
	error,
) {
	pairs := make([]cadence.KeyValuePair, v.Count())

	for i, keyValue := range v.Keys().Elements() {

		// NOTE: use `Get` instead of accessing `Entries`,
		// so that the potentially deferred values are loaded from storage

		value := v.Get(inter, interpreter.ReturnEmptyLocationRange, keyValue).(*interpreter.SomeValue).Value

		convertedKey, err := exportValueWithInterpreter(keyValue, inter, results)
		if err != nil {
			return cadence.Dictionary{}, err
		}
		convertedValue, err := exportValueWithInterpreter(value, inter, results)
		if err != nil {
			return cadence.Dictionary{}, err
		}

		pairs[i] = cadence.KeyValuePair{
			Key:   convertedKey,
			Value: convertedValue,
		}
	}

	return cadence.NewDictionary(pairs), nil
}

func exportLinkValue(v interpreter.LinkValue, inter *interpreter.Interpreter) cadence.Link {
	path := exportPathValue(v.TargetPath)
	ty := string(inter.ConvertStaticToSemaType(v.Type).ID())
	return cadence.NewLink(path, ty)
}

func exportPathValue(v interpreter.PathValue) cadence.Path {
	return cadence.Path{
		Domain:     v.Domain.Identifier(),
		Identifier: v.Identifier,
	}
}

func exportTypeValue(v interpreter.TypeValue, inter *interpreter.Interpreter) cadence.TypeValue {
	var typeID string
	staticType := v.Type
	if staticType != nil {
		typeID = string(inter.ConvertStaticToSemaType(staticType).ID())
	}
	return cadence.TypeValue{
		StaticType: typeID,
	}
}

func exportCapabilityValue(v interpreter.CapabilityValue, inter *interpreter.Interpreter) cadence.Capability {
	var borrowType string
	if v.BorrowType != nil {
		borrowType = string(inter.ConvertStaticToSemaType(v.BorrowType).ID())
	}

	return cadence.Capability{
		Path:       exportPathValue(v.Path),
		Address:    cadence.NewAddress(v.Address),
		BorrowType: borrowType,
	}
}

// exportEvent converts a runtime event to its native Go representation.
func exportEvent(event exportableEvent) (cadence.Event, error) {
	fields := make([]cadence.Value, len(event.Fields))

	results := exportResults{}

	for i, field := range event.Fields {
		value, err := exportValueWithInterpreter(field.Value, field.Interpreter(), results)
		if err != nil {
			return cadence.Event{}, err
		}
		fields[i] = value
	}

	eventType := ExportType(event.Type, map[sema.TypeID]cadence.Type{}).(*cadence.EventType)
	return cadence.NewEvent(fields).WithType(eventType), nil
}

// importValue converts a Cadence value to a runtime value.
func importValue(inter *interpreter.Interpreter, value cadence.Value) (interpreter.Value, error) {
	switch v := value.(type) {
	case cadence.Void:
		return interpreter.VoidValue{}, nil
	case cadence.Optional:
		return importOptionalValue(inter, v)
	case cadence.Bool:
		return interpreter.BoolValue(v), nil
	case cadence.String:
		return interpreter.NewStringValue(string(v)), nil
	case cadence.Bytes:
		return interpreter.ByteSliceToByteArrayValue(v), nil
	case cadence.Address:
		return interpreter.NewAddressValue(common.Address(v)), nil
	case cadence.Int:
		return interpreter.NewIntValueFromBigInt(v.Value), nil
	case cadence.Int8:
		return interpreter.Int8Value(v), nil
	case cadence.Int16:
		return interpreter.Int16Value(v), nil
	case cadence.Int32:
		return interpreter.Int32Value(v), nil
	case cadence.Int64:
		return interpreter.Int64Value(v), nil
	case cadence.Int128:
		return interpreter.NewInt128ValueFromBigInt(v.Value), nil
	case cadence.Int256:
		return interpreter.NewInt256ValueFromBigInt(v.Value), nil
	case cadence.UInt:
		return interpreter.NewUIntValueFromBigInt(v.Value), nil
	case cadence.UInt8:
		return interpreter.UInt8Value(v), nil
	case cadence.UInt16:
		return interpreter.UInt16Value(v), nil
	case cadence.UInt32:
		return interpreter.UInt32Value(v), nil
	case cadence.UInt64:
		return interpreter.UInt64Value(v), nil
	case cadence.UInt128:
		return interpreter.NewUInt128ValueFromBigInt(v.Value), nil
	case cadence.UInt256:
		return interpreter.NewUInt256ValueFromBigInt(v.Value), nil
	case cadence.Word8:
		return interpreter.Word8Value(v), nil
	case cadence.Word16:
		return interpreter.Word16Value(v), nil
	case cadence.Word32:
		return interpreter.Word32Value(v), nil
	case cadence.Word64:
		return interpreter.Word64Value(v), nil
	case cadence.Fix64:
		return interpreter.Fix64Value(v), nil
	case cadence.UFix64:
		return interpreter.UFix64Value(v), nil
	case cadence.Path:
		return importPathValue(v), nil
	case cadence.Array:
		return importArrayValue(inter, v)
	case cadence.Dictionary:
		return importDictionaryValue(inter, v)
	case cadence.Struct:
		return importCompositeValue(
			inter,
			common.CompositeKindStructure,
			v.StructType.Location,
			v.StructType.QualifiedIdentifier,
			v.StructType.Fields,
			v.Fields,
		)
	case cadence.Resource:
		return importCompositeValue(
			inter,
			common.CompositeKindResource,
			v.ResourceType.Location,
			v.ResourceType.QualifiedIdentifier,
			v.ResourceType.Fields,
			v.Fields,
		)
	case cadence.Event:
		return importCompositeValue(
			inter,
			common.CompositeKindEvent,
			v.EventType.Location,
			v.EventType.QualifiedIdentifier,
			v.EventType.Fields,
			v.Fields,
		)
	case cadence.Enum:
		return importCompositeValue(
			inter,
			common.CompositeKindEnum,
			v.EnumType.Location,
			v.EnumType.QualifiedIdentifier,
			v.EnumType.Fields,
			v.Fields,
		)
	}

	return nil, fmt.Errorf("cannot import value of type %T", value)
}

func importPathValue(v cadence.Path) interpreter.PathValue {
	return interpreter.PathValue{
		Domain:     common.PathDomainFromIdentifier(v.Domain),
		Identifier: v.Identifier,
	}
}

func importOptionalValue(
	inter *interpreter.Interpreter,
	v cadence.Optional,
) (
	interpreter.Value,
	error,
) {
	if v.Value == nil {
		return interpreter.NilValue{}, nil
	}

	innerValue, err := importValue(inter, v.Value)
	if err != nil {
		return nil, err
	}

	return interpreter.NewSomeValueOwningNonCopying(innerValue), nil
}

func importArrayValue(
	inter *interpreter.Interpreter,
	v cadence.Array,
) (
	*interpreter.ArrayValue,
	error,
) {
	values := make([]interpreter.Value, len(v.Values))

	for i, elem := range v.Values {
		value, err := importValue(inter, elem)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}

	return interpreter.NewArrayValueUnownedNonCopying(values...), nil
}

func importDictionaryValue(
	inter *interpreter.Interpreter,
	v cadence.Dictionary,
) (
	*interpreter.DictionaryValue,
	error,
) {
	keysAndValues := make([]interpreter.Value, len(v.Pairs)*2)

	for i, pair := range v.Pairs {
		key, err := importValue(inter, pair.Key)
		if err != nil {
			return nil, err
		}
		keysAndValues[i*2] = key

		value, err := importValue(inter, pair.Value)
		if err != nil {
			return nil, err
		}
		keysAndValues[i*2+1] = value
	}

	return interpreter.NewDictionaryValueUnownedNonCopying(keysAndValues...), nil
}

func importCompositeValue(
	inter *interpreter.Interpreter,
	kind common.CompositeKind,
	location Location,
	qualifiedIdentifier string,
	fieldTypes []cadence.Field,
	fieldValues []cadence.Value,
) (
	*interpreter.CompositeValue,
	error,
) {
	fields := interpreter.NewStringValueOrderedMap()

	for i := 0; i < len(fieldTypes) && i < len(fieldValues); i++ {
		fieldType := fieldTypes[i]
		fieldValue := fieldValues[i]
		value, err := importValue(inter, fieldValue)
		if err != nil {
			return nil, err
		}
		fields.Set(
			fieldType.Identifier,
			value,
		)
	}

	if location == nil {
		switch sema.NativeCompositeTypes[qualifiedIdentifier] {
		case sema.PublicKeyType:
			// PublicKey has a dedicated constructor
			// (e.g. it has computed fields that must be initialized)
			return importPublicKey(inter, fields)

		case sema.HashAlgorithmType:
			// HashAlgorithmType has a dedicated constructor
			// (e.g. it has host functions)
			return importHashAlgorithm(fields)

		case sema.SignatureAlgorithmType:
			// continue in the normal path

		default:
			return nil, fmt.Errorf(
				"cannot import value of type %s",
				qualifiedIdentifier,
			)
		}
	}

	return interpreter.NewCompositeValue(
		location,
		qualifiedIdentifier,
		kind,
		fields,
		nil,
	), nil
}

func importPublicKey(
	inter *interpreter.Interpreter,
	fields *interpreter.StringValueOrderedMap,
) (
	*interpreter.CompositeValue,
	error,
) {

	var publicKeyValue *interpreter.ArrayValue
	var signAlgoValue *interpreter.CompositeValue

	ty := sema.PublicKeyType

	err := fields.ForeachWithError(func(fieldName string, value interpreter.Value) error {
		switch fieldName {
		case sema.PublicKeyPublicKeyField:
			arrayValue, ok := value.(*interpreter.ArrayValue)
			if !ok {
				return fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					fieldName,
					value,
				)
			}

			publicKeyValue = arrayValue

		case sema.PublicKeySignAlgoField:
			compositeValue, ok := value.(*interpreter.CompositeValue)
			if !ok {
				return fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					fieldName,
					value,
				)
			}

			signAlgoValue = compositeValue

		case sema.PublicKeyIsValidField:
			// 'isValid' field set by the user must be ignored.
			// This is calculated when creating the public key.

		default:
			return fmt.Errorf(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				fieldName,
			)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if publicKeyValue == nil {
		return nil, fmt.Errorf(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.PublicKeyPublicKeyField,
		)
	}

	if signAlgoValue == nil {
		return nil, fmt.Errorf(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.PublicKeySignAlgoField,
		)
	}

	return interpreter.NewPublicKeyValue(
		publicKeyValue,
		signAlgoValue,
		inter.PublicKeyValidationHandler,
	), nil
}

func importHashAlgorithm(
	fields *interpreter.StringValueOrderedMap,
) (
	*interpreter.CompositeValue,
	error,
) {

	var foundRawValue bool
	var rawValue interpreter.UInt8Value

	ty := sema.HashAlgorithmType

	err := fields.ForeachWithError(func(fieldName string, value interpreter.Value) error {
		switch fieldName {
		case sema.EnumRawValueFieldName:
			rawValue, foundRawValue = value.(interpreter.UInt8Value)
			if !foundRawValue {
				return fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					fieldName,
					value,
				)
			}

		default:
			return fmt.Errorf(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				fieldName,
			)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if !foundRawValue {
		return nil, fmt.Errorf(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.EnumRawValueFieldName,
		)
	}

	return stdlib.NewHashAlgorithmCase(uint8(rawValue)), nil
}
