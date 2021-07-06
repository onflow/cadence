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
func exportValue(value exportableValue) cadence.Value {
	return exportValueWithInterpreter(value.Value, value.Interpreter(), exportResults{})
}

// ExportValue converts a runtime value to its native Go representation.
func ExportValue(value interpreter.Value, inter *interpreter.Interpreter) cadence.Value {
	return exportValueWithInterpreter(value, inter, exportResults{})
}

// exportEvent converts a runtime event to its native Go representation.
func exportEvent(event exportableEvent) cadence.Event {
	fields := make([]cadence.Value, len(event.Fields))

	results := exportResults{}

	for i, field := range event.Fields {
		fields[i] = exportValueWithInterpreter(field.Value, field.Interpreter(), results)
	}

	eventType := ExportType(event.Type, map[sema.TypeID]cadence.Type{}).(*cadence.EventType)
	return cadence.NewEvent(fields).WithType(eventType)
}

type exportResults map[interpreter.Value]cadence.Value

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
) cadence.Value {

	if result, ok := results[value]; ok {
		return result
	}

	results[value] = nil

	result := func() cadence.Value {

		switch v := value.(type) {
		case interpreter.VoidValue:
			return cadence.NewVoid()
		case interpreter.NilValue:
			return cadence.NewOptional(nil)
		case *interpreter.SomeValue:
			return exportSomeValue(v, inter, results)
		case interpreter.BoolValue:
			return cadence.NewBool(bool(v))
		case *interpreter.StringValue:
			return cadence.NewString(v.Str)
		case *interpreter.ArrayValue:
			return exportArrayValue(v, inter, results)
		case interpreter.IntValue:
			return cadence.NewIntFromBig(v.ToBigInt())
		case interpreter.Int8Value:
			return cadence.NewInt8(int8(v))
		case interpreter.Int16Value:
			return cadence.NewInt16(int16(v))
		case interpreter.Int32Value:
			return cadence.NewInt32(int32(v))
		case interpreter.Int64Value:
			return cadence.NewInt64(int64(v))
		case interpreter.Int128Value:
			return cadence.NewInt128FromBig(v.ToBigInt())
		case interpreter.Int256Value:
			return cadence.NewInt256FromBig(v.ToBigInt())
		case interpreter.UIntValue:
			return cadence.NewUIntFromBig(v.ToBigInt())
		case interpreter.UInt8Value:
			return cadence.NewUInt8(uint8(v))
		case interpreter.UInt16Value:
			return cadence.NewUInt16(uint16(v))
		case interpreter.UInt32Value:
			return cadence.NewUInt32(uint32(v))
		case interpreter.UInt64Value:
			return cadence.NewUInt64(uint64(v))
		case interpreter.UInt128Value:
			return cadence.NewUInt128FromBig(v.ToBigInt())
		case interpreter.UInt256Value:
			return cadence.NewUInt256FromBig(v.ToBigInt())
		case interpreter.Word8Value:
			return cadence.NewWord8(uint8(v))
		case interpreter.Word16Value:
			return cadence.NewWord16(uint16(v))
		case interpreter.Word32Value:
			return cadence.NewWord32(uint32(v))
		case interpreter.Word64Value:
			return cadence.NewWord64(uint64(v))
		case interpreter.Fix64Value:
			return cadence.Fix64(v)
		case interpreter.UFix64Value:
			return cadence.UFix64(v)
		case *interpreter.CompositeValue:
			return exportCompositeValue(v, inter, results)
		case *interpreter.DictionaryValue:
			return exportDictionaryValue(v, inter, results)
		case interpreter.AddressValue:
			return cadence.NewAddress(v)
		case interpreter.LinkValue:
			return exportLinkValue(v, inter)
		case interpreter.PathValue:
			return exportPathValue(v)
		case interpreter.TypeValue:
			return exportTypeValue(v, inter)
		case interpreter.CapabilityValue:
			return exportCapabilityValue(v, inter)
		case *interpreter.EphemeralReferenceValue:
			return exportValueWithInterpreter(v.Value, inter, results)
		case *interpreter.StorageReferenceValue:
			referencedValue := v.ReferencedValue(inter)
			if referencedValue == nil {
				return nil
			}
			return exportValueWithInterpreter(*referencedValue, inter, results)
		}

		panic(fmt.Sprintf("cannot export value of type %T", value))
	}()

	results[value] = result

	return result
}

func exportSomeValue(v *interpreter.SomeValue, inter *interpreter.Interpreter, results exportResults) cadence.Optional {
	if v.Value == nil {
		return cadence.NewOptional(nil)
	}

	value := exportValueWithInterpreter(v.Value, inter, results)

	return cadence.NewOptional(value)
}

func exportArrayValue(v *interpreter.ArrayValue, inter *interpreter.Interpreter, results exportResults) cadence.Array {
	elements := v.Elements()
	values := make([]cadence.Value, len(elements))

	for i, value := range elements {
		values[i] = exportValueWithInterpreter(value, inter, results)
	}

	return cadence.NewArray(values)
}

func exportCompositeValue(v *interpreter.CompositeValue, inter *interpreter.Interpreter, results exportResults) cadence.Value {

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

		fields[i] = exportValueWithInterpreter(fieldValue, inter, results)
	}

	// NOTE: when modifying the cases below,
	// also update the error message below!

	switch staticType.Kind {
	case common.CompositeKindStructure:
		return cadence.NewStruct(fields).WithType(t.(*cadence.StructType))
	case common.CompositeKindResource:
		return cadence.NewResource(fields).WithType(t.(*cadence.ResourceType))
	case common.CompositeKindEvent:
		return cadence.NewEvent(fields).WithType(t.(*cadence.EventType))
	case common.CompositeKindContract:
		return cadence.NewContract(fields).WithType(t.(*cadence.ContractType))
	case common.CompositeKindEnum:
		return cadence.NewEnum(fields).WithType(t.(*cadence.EnumType))
	}

	panic(fmt.Errorf(
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
	))
}

func exportDictionaryValue(
	v *interpreter.DictionaryValue,
	inter *interpreter.Interpreter,
	results exportResults,
) cadence.Dictionary {

	pairs := make([]cadence.KeyValuePair, v.Count())

	for i, keyValue := range v.Keys().Elements() {

		// NOTE: use `Get` instead of accessing `Entries`,
		// so that the potentially deferred values are loaded from storage

		value := v.Get(inter, interpreter.ReturnEmptyLocationRange, keyValue).(*interpreter.SomeValue).Value

		convertedKey := exportValueWithInterpreter(keyValue, inter, results)
		convertedValue := exportValueWithInterpreter(value, inter, results)

		pairs[i] = cadence.KeyValuePair{
			Key:   convertedKey,
			Value: convertedValue,
		}
	}

	return cadence.NewDictionary(pairs)
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

// importValue converts a Cadence value to a runtime value.
func importValue(inter *interpreter.Interpreter, value cadence.Value, expectedType sema.Type) interpreter.Value {
	switch v := value.(type) {
	case cadence.Void:
		return interpreter.VoidValue{}
	case cadence.Optional:
		return importOptionalValue(inter, v, expectedType)
	case cadence.Bool:
		return interpreter.BoolValue(v)
	case cadence.String:
		return interpreter.NewStringValue(string(v))
	case cadence.Bytes:
		return interpreter.ByteSliceToByteArrayValue(v)
	case cadence.Address:
		return interpreter.NewAddressValueFromBytes(v.Bytes())
	case cadence.Int:
		return interpreter.NewIntValueFromBigInt(v.Big())
	case cadence.Int8:
		return interpreter.Int8Value(v)
	case cadence.Int16:
		return interpreter.Int16Value(v)
	case cadence.Int32:
		return interpreter.Int32Value(v)
	case cadence.Int64:
		return interpreter.Int64Value(v)
	case cadence.Int128:
		return interpreter.NewInt128ValueFromBigInt(v.Big())
	case cadence.Int256:
		return interpreter.NewInt256ValueFromBigInt(v.Big())
	case cadence.UInt:
		return interpreter.NewUIntValueFromBigInt(v.Big())
	case cadence.UInt8:
		return interpreter.UInt8Value(v)
	case cadence.UInt16:
		return interpreter.UInt16Value(v)
	case cadence.UInt32:
		return interpreter.UInt32Value(v)
	case cadence.UInt64:
		return interpreter.UInt64Value(v)
	case cadence.UInt128:
		return interpreter.NewUInt128ValueFromBigInt(v.Big())
	case cadence.UInt256:
		return interpreter.NewUInt256ValueFromBigInt(v.Big())
	case cadence.Word8:
		return interpreter.Word8Value(v)
	case cadence.Word16:
		return interpreter.Word16Value(v)
	case cadence.Word32:
		return interpreter.Word32Value(v)
	case cadence.Word64:
		return interpreter.Word64Value(v)
	case cadence.Fix64:
		return interpreter.Fix64Value(v)
	case cadence.UFix64:
		return interpreter.UFix64Value(v)
	case cadence.Array:
		return importArrayValue(inter, v, expectedType)
	case cadence.Dictionary:
		return importDictionaryValue(inter, v, expectedType)
	case cadence.Struct:
		return importCompositeValue(
			inter,
			common.CompositeKindStructure,
			v.StructType.Location,
			v.StructType.QualifiedIdentifier,
			v.StructType.Fields,
			v.Fields,
			expectedType,
		)
	case cadence.Resource:
		return importCompositeValue(
			inter,
			common.CompositeKindResource,
			v.ResourceType.Location,
			v.ResourceType.QualifiedIdentifier,
			v.ResourceType.Fields,
			v.Fields,
			expectedType,
		)
	case cadence.Event:
		return importCompositeValue(
			inter,
			common.CompositeKindEvent,
			v.EventType.Location,
			v.EventType.QualifiedIdentifier,
			v.EventType.Fields,
			v.Fields,
			expectedType,
		)
	case cadence.Path:
		return importPathValue(v)
	case cadence.Enum:
		return importCompositeValue(
			inter,
			common.CompositeKindEnum,
			v.EnumType.Location,
			v.EnumType.QualifiedIdentifier,
			v.EnumType.Fields,
			v.Fields,
			expectedType,
		)
	}

	panic(fmt.Sprintf("cannot import value of type %T", value))
}

func importPathValue(v cadence.Path) interpreter.PathValue {
	return interpreter.PathValue{
		Domain:     common.PathDomainFromIdentifier(v.Domain),
		Identifier: v.Identifier,
	}
}

func importOptionalValue(inter *interpreter.Interpreter, v cadence.Optional, expectedType sema.Type) interpreter.Value {
	if v.Value == nil {
		return interpreter.NilValue{}
	}

	var innerType sema.Type
	if optionalType, ok := expectedType.(*sema.OptionalType); ok {
		innerType = optionalType.Type
	}

	innerValue := importValue(inter, v.Value, innerType)
	return interpreter.NewSomeValueOwningNonCopying(innerValue)
}

func importArrayValue(inter *interpreter.Interpreter, v cadence.Array, expectedType sema.Type) *interpreter.ArrayValue {
	values := make([]interpreter.Value, len(v.Values))

	var arrayType sema.ArrayType
	var elementType sema.Type
	if expectedArrayType, ok := expectedType.(sema.ArrayType); ok {
		arrayType = expectedArrayType
		elementType = arrayType.ElementType(false)
	}

	for i, element := range v.Values {
		values[i] = importValue(inter, element, elementType)
	}

	var staticArrayType interpreter.ArrayStaticType
	if arrayType != nil {
		staticArrayType = interpreter.ConvertSemaArrayTypeToStaticArrayType(arrayType)
	} else {
		staticArrayType = interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeAnyStruct,
		}
	}

	return interpreter.NewArrayValueUnownedNonCopying(staticArrayType, values...)
}

func importDictionaryValue(
	inter *interpreter.Interpreter,
	v cadence.Dictionary,
	expectedType sema.Type,
) *interpreter.DictionaryValue {
	keysAndValues := make([]interpreter.Value, len(v.Pairs)*2)

	var keyType sema.Type
	var valueType sema.Type

	dictionaryType, ok := expectedType.(*sema.DictionaryType)
	if ok {
		keyType = dictionaryType.KeyType
		valueType = dictionaryType.ValueType
	}

	for i, pair := range v.Pairs {
		keysAndValues[i*2] = importValue(inter, pair.Key, keyType)
		keysAndValues[i*2+1] = importValue(inter, pair.Value, valueType)
	}

	var dictionaryStaticType interpreter.DictionaryStaticType
	if dictionaryType != nil {
		dictionaryStaticType = interpreter.ConvertSemaDictionaryTypeToStaticDictionaryType(dictionaryType)
	} else {
		var keyStaticType, valueStaticType interpreter.StaticType

		if len(keysAndValues) == 0 {
			keyStaticType = interpreter.PrimitiveStaticTypeNever
			valueStaticType = interpreter.PrimitiveStaticTypeNever
		} else {
			valueStaticType = interpreter.PrimitiveStaticTypeAnyStruct
			keyStaticType = keysAndValues[0].StaticType()

			keySemaType := inter.ConvertStaticToSemaType(keyStaticType)

			if sema.IsSubType(keySemaType, sema.NumberType) {
				keyStaticType = interpreter.PrimitiveStaticTypeNumber
			} else if sema.IsSubType(keySemaType, sema.PathType) {
				keyStaticType = interpreter.PrimitiveStaticTypePath
			} else if sema.IsValidDictionaryKeyType(keySemaType) {
				// NO-OP. Use 'keyStaticType' as static type for keys
			} else {
				panic(fmt.Errorf(
					"cannot import dictionary: unsupported key: %v",
					keysAndValues[0],
				))
			}
		}

		dictionaryStaticType = interpreter.DictionaryStaticType{
			KeyType:   keyStaticType,
			ValueType: valueStaticType,
		}
	}

	return interpreter.NewDictionaryValueUnownedNonCopying(
		dictionaryStaticType,
		keysAndValues...,
	)
}

func importCompositeValue(
	inter *interpreter.Interpreter,
	kind common.CompositeKind,
	location Location,
	qualifiedIdentifier string,
	fieldTypes []cadence.Field,
	fieldValues []cadence.Value,
	expectedType sema.Type,
) *interpreter.CompositeValue {
	fields := interpreter.NewStringValueOrderedMap()

	expectedCompositeType, ok := expectedType.(*sema.CompositeType)

	for i := 0; i < len(fieldTypes) && i < len(fieldValues); i++ {
		fieldType := fieldTypes[i]
		fieldValue := fieldValues[i]

		var expectedFieldType sema.Type

		if ok {
			member, ok := expectedCompositeType.Members.Get(fieldType.Identifier)
			if ok {
				expectedFieldType = member.TypeAnnotation.Type
			}
		}

		importedFieldValue := importValue(inter, fieldValue, expectedFieldType)

		fields.Set(fieldType.Identifier, importedFieldValue)
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
			panic(fmt.Errorf(
				"cannot import value of type %s",
				qualifiedIdentifier,
			))
		}
	}

	return interpreter.NewCompositeValue(
		location,
		qualifiedIdentifier,
		kind,
		fields,
		nil,
	)
}

func importPublicKey(
	inter *interpreter.Interpreter,
	fields *interpreter.StringValueOrderedMap,
) *interpreter.CompositeValue {

	var publicKeyValue *interpreter.ArrayValue
	var signAlgoValue *interpreter.CompositeValue

	ty := sema.PublicKeyType

	fields.Foreach(func(fieldName string, value interpreter.Value) {
		switch fieldName {
		case sema.PublicKeyPublicKeyField:
			arrayValue, ok := value.(*interpreter.ArrayValue)
			if !ok {
				panic(fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					fieldName,
					value,
				))
			}

			publicKeyValue = arrayValue

		case sema.PublicKeySignAlgoField:
			compositeValue, ok := value.(*interpreter.CompositeValue)
			if !ok {
				panic(fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					fieldName,
					value,
				))
			}

			signAlgoValue = compositeValue

		case sema.PublicKeyIsValidField:
			// 'isValid' field set by the user must be ignored.
			// This is calculated when creating the public key.
			return

		default:
			panic(fmt.Errorf(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				fieldName,
			))
		}
	})

	if publicKeyValue == nil {
		panic(fmt.Errorf(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.PublicKeyPublicKeyField,
		))
	}

	if signAlgoValue == nil {
		panic(fmt.Errorf(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.PublicKeySignAlgoField,
		))
	}

	return interpreter.NewPublicKeyValue(
		publicKeyValue,
		signAlgoValue,
		inter.PublicKeyValidationHandler,
	)
}

func importHashAlgorithm(
	fields *interpreter.StringValueOrderedMap,
) *interpreter.CompositeValue {

	var foundRawValue bool
	var rawValue interpreter.UInt8Value

	ty := sema.HashAlgorithmType

	fields.Foreach(func(fieldName string, value interpreter.Value) {
		switch fieldName {
		case sema.EnumRawValueFieldName:
			rawValue, foundRawValue = value.(interpreter.UInt8Value)
			if !foundRawValue {
				panic(fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					fieldName,
					value,
				))
			}

		default:
			panic(fmt.Errorf(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				fieldName,
			))
		}
	})

	if !foundRawValue {
		panic(fmt.Errorf(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.EnumRawValueFieldName,
		))
	}

	return stdlib.NewHashAlgorithmCase(uint8(rawValue))
}
