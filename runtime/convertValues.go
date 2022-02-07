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
	return exportValueWithInterpreter(value.Value, value.Interpreter(), seenReferences{})
}

// ExportValue converts a runtime value to its native Go representation.
func ExportValue(value interpreter.Value, inter *interpreter.Interpreter) (cadence.Value, error) {
	return exportValueWithInterpreter(value, inter, seenReferences{})
}

// NOTE: Do not generalize to map[interpreter.Value],
// as not all values are Go hashable, i.e. this might lead to run-time panics
type seenReferences map[*interpreter.EphemeralReferenceValue]struct{}

// exportValueWithInterpreter exports the given internal (interpreter) value to an external value.
//
// The export is recursive, the results parameter prevents cycles:
// it is checked at the start of the recursively called function,
// and pre-set before a recursive call.
//
func exportValueWithInterpreter(
	value interpreter.Value,
	inter *interpreter.Interpreter,
	seenReferences seenReferences,
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
		return exportSomeValue(v, inter, seenReferences)
	case interpreter.BoolValue:
		return cadence.NewBool(bool(v)), nil
	case *interpreter.StringValue:
		return cadence.NewString(v.Str)
	case interpreter.CharacterValue:
		return cadence.NewCharacter(string(v))
	case *interpreter.ArrayValue:
		return exportArrayValue(v, inter, seenReferences)
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
		return exportCompositeValue(v, inter, seenReferences)
	case *interpreter.SimpleCompositeValue:
		return exportSimpleCompositeValue(v, inter, seenReferences)
	case *interpreter.DictionaryValue:
		return exportDictionaryValue(v, inter, seenReferences)
	case interpreter.AddressValue:
		return cadence.NewAddress(v), nil
	case interpreter.LinkValue:
		return exportLinkValue(v, inter), nil
	case interpreter.PathValue:
		return exportPathValue(v), nil
	case interpreter.TypeValue:
		return exportTypeValue(v, inter), nil
	case *interpreter.CapabilityValue:
		return exportCapabilityValue(v, inter), nil
	case *interpreter.EphemeralReferenceValue:
		// Break recursion through ephemeral references
		if _, ok := seenReferences[v]; ok {
			return nil, nil
		}
		defer delete(seenReferences, v)
		seenReferences[v] = struct{}{}
		return exportValueWithInterpreter(v.Value, inter, seenReferences)
	case *interpreter.StorageReferenceValue:
		referencedValue := v.ReferencedValue(inter)
		if referencedValue == nil {
			return nil, nil
		}
		return exportValueWithInterpreter(*referencedValue, inter, seenReferences)
	}

	return nil, fmt.Errorf("cannot export value of type %T", value)

}

func exportSomeValue(
	v *interpreter.SomeValue,
	inter *interpreter.Interpreter,
	seenReferences seenReferences,
) (
	cadence.Optional,
	error,
) {
	if v.Value == nil {
		return cadence.NewOptional(nil), nil
	}

	value, err := exportValueWithInterpreter(v.Value, inter, seenReferences)
	if err != nil {
		return cadence.Optional{}, err
	}

	return cadence.NewOptional(value), nil
}

func exportArrayValue(
	v *interpreter.ArrayValue,
	inter *interpreter.Interpreter,
	seenReferences seenReferences,
) (
	cadence.Array,
	error,
) {
	values := make([]cadence.Value, 0, v.Count())

	var err error
	v.Iterate(func(value interpreter.Value) (resume bool) {
		var exportedValue cadence.Value
		exportedValue, err = exportValueWithInterpreter(value, inter, seenReferences)
		if err != nil {
			return false
		}
		values = append(
			values,
			exportedValue,
		)
		return true
	})
	if err != nil {
		return cadence.Array{}, err
	}

	return cadence.NewArray(values), nil
}

func exportCompositeValue(
	v *interpreter.CompositeValue,
	inter *interpreter.Interpreter,
	seenReferences seenReferences,
) (
	cadence.Value,
	error,
) {

	dynamicType := v.DynamicType(inter, interpreter.SeenReferences{}).(interpreter.CompositeDynamicType)
	staticType := dynamicType.StaticType.(*sema.CompositeType)
	// TODO: consider making the results map "global", by moving it up to exportValueWithInterpreter
	t := exportCompositeType(staticType, map[sema.TypeID]cadence.Type{})

	// NOTE: use the exported type's fields to ensure fields in type
	// and value are in sync

	fieldNames := t.CompositeFields()
	fields := make([]cadence.Value, len(fieldNames))

	for i, field := range fieldNames {
		fieldName := field.Identifier

		// TODO: provide proper location range
		fieldValue := v.GetField(fieldName)
		if fieldValue == nil && v.ComputedFields != nil {
			if computedField, ok := v.ComputedFields[fieldName]; ok {
				// TODO: provide proper location range
				fieldValue = computedField(inter, interpreter.ReturnEmptyLocationRange)
			}
		}

		exportedFieldValue, err := exportValueWithInterpreter(fieldValue, inter, seenReferences)
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

func exportSimpleCompositeValue(
	v *interpreter.SimpleCompositeValue,
	inter *interpreter.Interpreter,
	seenReferences seenReferences,
) (
	cadence.Value,
	error,
) {
	dynamicType, ok := v.DynamicType(inter, interpreter.SeenReferences{}).(interpreter.CompositeDynamicType)
	if !ok {
		return nil, fmt.Errorf(
			"unexportable composite value: %s", dynamicType.StaticType,
		)
	}
	staticType := dynamicType.StaticType.(*sema.CompositeType)
	// TODO: consider making the results map "global", by moving it up to exportValueWithInterpreter
	t := exportCompositeType(staticType, map[sema.TypeID]cadence.Type{})

	// NOTE: use the exported type's fields to ensure fields in type
	// and value are in sync

	fieldNames := t.CompositeFields()
	fields := make([]cadence.Value, len(fieldNames))

	for i, field := range fieldNames {
		fieldName := field.Identifier

		fieldValue := v.Fields[fieldName]
		if fieldValue == nil && v.ComputedFields != nil {
			if computedField, ok := v.ComputedFields[fieldName]; ok {
				// TODO: provide proper location range
				fieldValue = computedField(inter, interpreter.ReturnEmptyLocationRange)
			}
		}

		exportedFieldValue, err := exportValueWithInterpreter(fieldValue, inter, seenReferences)
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
	seenReferences seenReferences,
) (
	cadence.Dictionary,
	error,
) {
	pairs := make([]cadence.KeyValuePair, 0, v.Count())

	var err error
	v.Iterate(func(key, value interpreter.Value) (resume bool) {

		var convertedKey cadence.Value
		convertedKey, err = exportValueWithInterpreter(key, inter, seenReferences)
		if err != nil {
			return false
		}

		var convertedValue cadence.Value
		convertedValue, err = exportValueWithInterpreter(value, inter, seenReferences)
		if err != nil {
			return false
		}

		pairs = append(
			pairs,
			cadence.KeyValuePair{
				Key:   convertedKey,
				Value: convertedValue,
			},
		)

		return true
	})

	if err != nil {
		return cadence.Dictionary{}, err
	}

	return cadence.NewDictionary(pairs), nil
}

func exportLinkValue(v interpreter.LinkValue, inter *interpreter.Interpreter) cadence.Link {
	path := exportPathValue(v.TargetPath)
	ty := string(inter.MustConvertStaticToSemaType(v.Type).ID())
	return cadence.NewLink(path, ty)
}

func exportPathValue(v interpreter.PathValue) cadence.Path {
	return cadence.Path{
		Domain:     v.Domain.Identifier(),
		Identifier: v.Identifier,
	}
}

func exportTypeValue(v interpreter.TypeValue, inter *interpreter.Interpreter) cadence.TypeValue {
	var typ sema.Type
	if v.Type != nil {
		typ = inter.MustConvertStaticToSemaType(v.Type)
	}
	return cadence.TypeValue{
		StaticType: ExportType(typ, map[sema.TypeID]cadence.Type{}),
	}
}

func exportCapabilityValue(v *interpreter.CapabilityValue, inter *interpreter.Interpreter) cadence.Capability {
	var borrowType sema.Type
	if v.BorrowType != nil {
		borrowType = inter.MustConvertStaticToSemaType(v.BorrowType)
	}

	return cadence.Capability{
		Path:       exportPathValue(v.Path),
		Address:    cadence.NewAddress(v.Address),
		BorrowType: ExportType(borrowType, map[sema.TypeID]cadence.Type{}),
	}
}

// exportEvent converts a runtime event to its native Go representation.
func exportEvent(event exportableEvent, seenReferences seenReferences) (cadence.Event, error) {
	fields := make([]cadence.Value, len(event.Fields))

	for i, field := range event.Fields {
		value, err := exportValueWithInterpreter(field.Value, field.Interpreter(), seenReferences)
		if err != nil {
			return cadence.Event{}, err
		}
		fields[i] = value
	}

	eventType := ExportType(event.Type, map[sema.TypeID]cadence.Type{}).(*cadence.EventType)
	return cadence.NewEvent(fields).WithType(eventType), nil
}

// importValue converts a Cadence value to a runtime value.
func importValue(inter *interpreter.Interpreter, value cadence.Value, expectedType sema.Type) (interpreter.Value, error) {
	switch v := value.(type) {
	case cadence.Void:
		return interpreter.VoidValue{}, nil
	case cadence.Optional:
		return importOptionalValue(inter, v, expectedType)
	case cadence.Bool:
		return interpreter.BoolValue(v), nil
	case cadence.String:
		return interpreter.NewStringValue(string(v)), nil
	case cadence.Character:
		return interpreter.NewCharacterValue(string(v)), nil
	case cadence.Bytes:
		return interpreter.ByteSliceToByteArrayValue(inter, v), nil
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
	case cadence.TypeValue:
		return importTypeValue(
			inter,
			v.StaticType,
		)
	case cadence.Capability:
		return importCapability(
			inter,
			v.Path,
			v.Address,
			v.BorrowType,
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

func importTypeValue(
	inter *interpreter.Interpreter,
	v cadence.Type,
) (
	interpreter.TypeValue,
	error,
) {
	typ := ImportType(v)
	/* creating a static type performs no validation, so
	   in order to be sure the type we have created is legal,
	   we convert it to a sema type. If this fails, the
	   import is invalid */
	_, err := inter.ConvertStaticToSemaType(typ)
	if err != nil {
		return interpreter.TypeValue{}, err
	}

	return interpreter.TypeValue{
		Type: typ,
	}, nil
}

func importCapability(
	_ *interpreter.Interpreter,
	path cadence.Path,
	address cadence.Address,
	borrowType cadence.Type,
) (
	*interpreter.CapabilityValue,
	error,
) {

	_, ok := borrowType.(cadence.ReferenceType)

	if !ok {
		return nil, fmt.Errorf(
			"cannot import capability: expected reference, got '%s'",
			borrowType.ID(),
		)
	}

	return &interpreter.CapabilityValue{
		Path:       importPathValue(path),
		Address:    interpreter.NewAddressValueFromBytes(address.Bytes()),
		BorrowType: ImportType(borrowType),
	}, nil

}

func importOptionalValue(
	inter *interpreter.Interpreter,
	v cadence.Optional,
	expectedType sema.Type,
) (
	interpreter.Value,
	error,
) {
	if v.Value == nil {
		return interpreter.NilValue{}, nil
	}

	var innerType sema.Type
	if optionalType, ok := expectedType.(*sema.OptionalType); ok {
		innerType = optionalType.Type
	}

	innerValue, err := importValue(inter, v.Value, innerType)
	if err != nil {
		return nil, err
	}

	return interpreter.NewSomeValueNonCopying(innerValue), nil
}

func importArrayValue(
	inter *interpreter.Interpreter,
	v cadence.Array,
	expectedType sema.Type,
) (
	*interpreter.ArrayValue,
	error,
) {
	values := make([]interpreter.Value, len(v.Values))

	var elementType sema.Type
	arrayType, ok := expectedType.(sema.ArrayType)
	if ok {
		elementType = arrayType.ElementType(false)
	}

	for i, element := range v.Values {
		value, err := importValue(inter, element, elementType)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}

	var staticArrayType interpreter.ArrayStaticType
	if arrayType != nil {
		staticArrayType = interpreter.ConvertSemaArrayTypeToStaticArrayType(arrayType)
	} else {
		types := make([]sema.Type, len(v.Values))

		for i, value := range values {
			typ, err := inter.ConvertStaticToSemaType(value.StaticType())
			if err != nil {
				return nil, err
			}
			types[i] = typ
		}

		elementSuperType := sema.LeastCommonSuperType(types...)
		if elementSuperType == sema.InvalidType {
			return nil, fmt.Errorf("cannot import array: elements do not belong to the same type")
		}

		staticArrayType = interpreter.VariableSizedStaticType{
			Type: interpreter.ConvertSemaToStaticType(elementSuperType),
		}
	}

	return interpreter.NewArrayValue(
		inter,
		staticArrayType,
		common.Address{},
		values...,
	), nil
}

func importDictionaryValue(
	inter *interpreter.Interpreter,
	v cadence.Dictionary,
	expectedType sema.Type,
) (
	*interpreter.DictionaryValue,
	error,
) {
	keysAndValues := make([]interpreter.Value, len(v.Pairs)*2)

	var keyType sema.Type
	var valueType sema.Type

	dictionaryType, ok := expectedType.(*sema.DictionaryType)
	if ok {
		keyType = dictionaryType.KeyType
		valueType = dictionaryType.ValueType
	}

	for i, pair := range v.Pairs {
		key, err := importValue(inter, pair.Key, keyType)
		if err != nil {
			return nil, err
		}
		keysAndValues[i*2] = key

		value, err := importValue(inter, pair.Value, valueType)
		if err != nil {
			return nil, err
		}
		keysAndValues[i*2+1] = value
	}

	var dictionaryStaticType interpreter.DictionaryStaticType
	if dictionaryType != nil {
		dictionaryStaticType = interpreter.ConvertSemaDictionaryTypeToStaticDictionaryType(dictionaryType)
	} else {
		size := len(v.Pairs)
		keyTypes := make([]sema.Type, size)
		valueTypes := make([]sema.Type, size)

		for i := 0; i < size; i++ {
			keyType, err := inter.ConvertStaticToSemaType(keysAndValues[i*2].StaticType())
			if err != nil {
				return nil, err
			}
			keyTypes[i] = keyType

			valueType, err := inter.ConvertStaticToSemaType(keysAndValues[i*2+1].StaticType())
			if err != nil {
				return nil, err
			}
			valueTypes[i] = valueType
		}

		keySuperType := sema.LeastCommonSuperType(keyTypes...)
		valueSuperType := sema.LeastCommonSuperType(valueTypes...)

		if !sema.IsValidDictionaryKeyType(keySuperType) {
			return nil, fmt.Errorf(
				"cannot import dictionary: keys does not belong to the same type",
			)
		}

		if valueSuperType == sema.InvalidType {
			return nil, fmt.Errorf("cannot import dictionary: values does not belong to the same type")
		}

		dictionaryStaticType = interpreter.DictionaryStaticType{
			KeyType:   interpreter.ConvertSemaToStaticType(keySuperType),
			ValueType: interpreter.ConvertSemaToStaticType(valueSuperType),
		}
	}

	return interpreter.NewDictionaryValue(
		inter,
		dictionaryStaticType,
		keysAndValues...,
	), nil
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
	var fields []interpreter.CompositeField

	typeID := common.NewTypeIDFromQualifiedName(location, qualifiedIdentifier)
	compositeType, typeErr := inter.GetCompositeType(location, qualifiedIdentifier, typeID)
	if typeErr != nil {
		return nil, typeErr
	}

	for i := 0; i < len(fieldTypes) && i < len(fieldValues); i++ {
		fieldType := fieldTypes[i]
		fieldValue := fieldValues[i]

		var expectedFieldType sema.Type

		member, ok := compositeType.Members.Get(fieldType.Identifier)
		if ok {
			expectedFieldType = member.TypeAnnotation.Type
		}

		importedFieldValue, err := importValue(inter, fieldValue, expectedFieldType)
		if err != nil {
			return nil, err
		}

		fields = append(fields,
			interpreter.CompositeField{
				Name:  fieldType.Identifier,
				Value: importedFieldValue,
			},
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
			return importHashAlgorithm(inter, fields)

		case sema.SignatureAlgorithmType:
			// SignatureAlgorithmType has a dedicated constructor
			// (e.g. it has host functions)
			return importSignatureAlgorithm(inter, fields)

		default:
			return nil, fmt.Errorf(
				"cannot import value of type %s",
				qualifiedIdentifier,
			)
		}
	}

	return interpreter.NewCompositeValue(
		inter,
		location,
		qualifiedIdentifier,
		kind,
		fields,
		common.Address{},
	), nil
}

func importPublicKey(
	inter *interpreter.Interpreter,
	fields []interpreter.CompositeField,
) (
	*interpreter.CompositeValue,
	error,
) {

	var publicKeyValue *interpreter.ArrayValue
	var signAlgoValue *interpreter.CompositeValue

	ty := sema.PublicKeyType

	for _, field := range fields {
		switch field.Name {
		case sema.PublicKeyPublicKeyField:
			arrayValue, ok := field.Value.(*interpreter.ArrayValue)
			if !ok {
				return nil, fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

			publicKeyValue = arrayValue

		case sema.PublicKeySignAlgoField:
			compositeValue, ok := field.Value.(*interpreter.CompositeValue)
			if !ok {
				return nil, fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

			signAlgoValue = compositeValue

		default:
			return nil, fmt.Errorf(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				field.Name,
			)
		}

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

	// TODO: provide proper location range
	return interpreter.NewPublicKeyValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		publicKeyValue,
		signAlgoValue,
		inter.PublicKeyValidationHandler,
	), nil
}

func importHashAlgorithm(
	inter *interpreter.Interpreter,
	fields []interpreter.CompositeField,
) (
	*interpreter.CompositeValue,
	error,
) {

	var foundRawValue bool
	var rawValue interpreter.UInt8Value

	ty := sema.HashAlgorithmType

	for _, field := range fields {
		switch field.Name {
		case sema.EnumRawValueFieldName:
			rawValue, foundRawValue = field.Value.(interpreter.UInt8Value)
			if !foundRawValue {
				return nil, fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

		default:
			return nil, fmt.Errorf(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				field.Name,
			)
		}
	}

	if !foundRawValue {
		return nil, fmt.Errorf(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.EnumRawValueFieldName,
		)
	}

	return stdlib.NewHashAlgorithmCase(inter, uint8(rawValue)), nil
}

func importSignatureAlgorithm(
	inter *interpreter.Interpreter,
	fields []interpreter.CompositeField,
) (
	*interpreter.CompositeValue,
	error,
) {

	var foundRawValue bool
	var rawValue interpreter.UInt8Value

	ty := sema.SignatureAlgorithmType

	for _, field := range fields {
		switch field.Name {
		case sema.EnumRawValueFieldName:
			rawValue, foundRawValue = field.Value.(interpreter.UInt8Value)
			if !foundRawValue {
				return nil, fmt.Errorf(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

		default:
			return nil, fmt.Errorf(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				field.Name,
			)
		}
	}

	if !foundRawValue {
		return nil, fmt.Errorf(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.EnumRawValueFieldName,
		)
	}

	return stdlib.NewSignatureAlgorithmCase(inter, uint8(rawValue)), nil
}
