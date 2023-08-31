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

package cadence

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type Type interface {
	isType()
	ID() string
	Equal(other Type) bool
}

// TypeID is a type which is only known by its type ID.
// This type should not be used when encoding values,
// and should only be used for decoding values that were encoded
// using an older format of the JSON encoding (<v0.3.0)
type TypeID common.TypeID

func (TypeID) isType() {}

func (t TypeID) ID() string {
	return string(t)
}

func (t TypeID) Equal(other Type) bool {
	return t == other
}

// OptionalType

type OptionalType struct {
	Type Type
}

var _ Type = &OptionalType{}

func NewOptionalType(typ Type) *OptionalType {
	return &OptionalType{Type: typ}
}

func NewMeteredOptionalType(gauge common.MemoryGauge, typ Type) *OptionalType {
	common.UseMemory(gauge, common.CadenceOptionalTypeMemoryUsage)
	return NewOptionalType(typ)
}

func (*OptionalType) isType() {}

func (t *OptionalType) ID() string {
	return sema.FormatOptionalTypeID(t.Type.ID())
}

func (t *OptionalType) Equal(other Type) bool {
	otherOptional, ok := other.(*OptionalType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherOptional.Type)
}

// BytesType

type BytesType struct{}

var TheBytesType = BytesType{}

func (BytesType) isType() {}

func (BytesType) ID() string {
	return "Bytes"
}

func (t BytesType) Equal(other Type) bool {
	return t == other
}

// PrimitiveType

type PrimitiveType interpreter.PrimitiveStaticType

var _ Type = PrimitiveType(interpreter.PrimitiveStaticTypeUnknown)

func (p PrimitiveType) isType() {}

func (p PrimitiveType) ID() string {
	return string(interpreter.PrimitiveStaticType(p).ID())
}

func (p PrimitiveType) Equal(other Type) bool {
	otherP, ok := other.(PrimitiveType)
	return ok && p == otherP
}

var VoidType = PrimitiveType(interpreter.PrimitiveStaticTypeVoid)
var AnyType = PrimitiveType(interpreter.PrimitiveStaticTypeAny)
var NeverType = PrimitiveType(interpreter.PrimitiveStaticTypeNever)
var AnyStructType = PrimitiveType(interpreter.PrimitiveStaticTypeAnyStruct)
var AnyResourceType = PrimitiveType(interpreter.PrimitiveStaticTypeAnyResource)
var AnyStructAttachmentType = PrimitiveType(interpreter.PrimitiveStaticTypeAnyStructAttachment)
var AnyResourceAttachmentType = PrimitiveType(interpreter.PrimitiveStaticTypeAnyResourceAttachment)

var BoolType = PrimitiveType(interpreter.PrimitiveStaticTypeBool)
var AddressType = PrimitiveType(interpreter.PrimitiveStaticTypeAddress)
var StringType = PrimitiveType(interpreter.PrimitiveStaticTypeString)
var CharacterType = PrimitiveType(interpreter.PrimitiveStaticTypeCharacter)
var MetaType = PrimitiveType(interpreter.PrimitiveStaticTypeMetaType)
var BlockType = PrimitiveType(interpreter.PrimitiveStaticTypeBlock)

var NumberType = PrimitiveType(interpreter.PrimitiveStaticTypeNumber)
var SignedNumberType = PrimitiveType(interpreter.PrimitiveStaticTypeSignedNumber)

var IntegerType = PrimitiveType(interpreter.PrimitiveStaticTypeInteger)
var SignedIntegerType = PrimitiveType(interpreter.PrimitiveStaticTypeSignedInteger)

var FixedPointType = PrimitiveType(interpreter.PrimitiveStaticTypeFixedPoint)
var SignedFixedPointType = PrimitiveType(interpreter.PrimitiveStaticTypeSignedFixedPoint)

var IntType = PrimitiveType(interpreter.PrimitiveStaticTypeInt)
var Int8Type = PrimitiveType(interpreter.PrimitiveStaticTypeInt8)
var Int16Type = PrimitiveType(interpreter.PrimitiveStaticTypeInt16)
var Int32Type = PrimitiveType(interpreter.PrimitiveStaticTypeInt32)
var Int64Type = PrimitiveType(interpreter.PrimitiveStaticTypeInt64)
var Int128Type = PrimitiveType(interpreter.PrimitiveStaticTypeInt128)
var Int256Type = PrimitiveType(interpreter.PrimitiveStaticTypeInt256)

var UIntType = PrimitiveType(interpreter.PrimitiveStaticTypeUInt)
var UInt8Type = PrimitiveType(interpreter.PrimitiveStaticTypeUInt8)
var UInt16Type = PrimitiveType(interpreter.PrimitiveStaticTypeUInt16)
var UInt32Type = PrimitiveType(interpreter.PrimitiveStaticTypeUInt32)
var UInt64Type = PrimitiveType(interpreter.PrimitiveStaticTypeUInt64)
var UInt128Type = PrimitiveType(interpreter.PrimitiveStaticTypeUInt128)
var UInt256Type = PrimitiveType(interpreter.PrimitiveStaticTypeUInt256)

var Word8Type = PrimitiveType(interpreter.PrimitiveStaticTypeWord8)
var Word16Type = PrimitiveType(interpreter.PrimitiveStaticTypeWord16)
var Word32Type = PrimitiveType(interpreter.PrimitiveStaticTypeWord32)
var Word64Type = PrimitiveType(interpreter.PrimitiveStaticTypeWord64)
var Word128Type = PrimitiveType(interpreter.PrimitiveStaticTypeWord128)
var Word256Type = PrimitiveType(interpreter.PrimitiveStaticTypeWord256)

var Fix64Type = PrimitiveType(interpreter.PrimitiveStaticTypeFix64)
var UFix64Type = PrimitiveType(interpreter.PrimitiveStaticTypeUFix64)

var PathType = PrimitiveType(interpreter.PrimitiveStaticTypePath)
var CapabilityPathType = PrimitiveType(interpreter.PrimitiveStaticTypeCapabilityPath)
var StoragePathType = PrimitiveType(interpreter.PrimitiveStaticTypeStoragePath)
var PublicPathType = PrimitiveType(interpreter.PrimitiveStaticTypePublicPath)
var PrivatePathType = PrimitiveType(interpreter.PrimitiveStaticTypePrivatePath)

var DeployedContractType = PrimitiveType(interpreter.PrimitiveStaticTypeDeployedContract)

var StorageCapabilityControllerType = PrimitiveType(interpreter.PrimitiveStaticTypeStorageCapabilityController)
var AccountCapabilityControllerType = PrimitiveType(interpreter.PrimitiveStaticTypeAccountCapabilityController)

var AccountType = PrimitiveType(interpreter.PrimitiveStaticTypeAccount)
var Account_ContractsType = PrimitiveType(interpreter.PrimitiveStaticTypeAccount_Contracts)
var Account_KeysType = PrimitiveType(interpreter.PrimitiveStaticTypeAccount_Keys)
var Account_StorageType = PrimitiveType(interpreter.PrimitiveStaticTypeAccount_Storage)
var Account_InboxType = PrimitiveType(interpreter.PrimitiveStaticTypeAccount_Inbox)
var Account_CapabilitiesType = PrimitiveType(interpreter.PrimitiveStaticTypeAccount_Capabilities)
var Account_StorageCapabilitiesType = PrimitiveType(interpreter.PrimitiveStaticTypeAccount_StorageCapabilities)
var Account_AccountCapabilitiesType = PrimitiveType(interpreter.PrimitiveStaticTypeAccount_AccountCapabilities)

var MutateType = PrimitiveType(interpreter.PrimitiveStaticTypeMutate)
var InsertType = PrimitiveType(interpreter.PrimitiveStaticTypeInsert)
var RemoveType = PrimitiveType(interpreter.PrimitiveStaticTypeRemove)
var IdentityType = PrimitiveType(interpreter.PrimitiveStaticTypeIdentity)

var StorageType = PrimitiveType(interpreter.PrimitiveStaticTypeStorage)
var SaveValueType = PrimitiveType(interpreter.PrimitiveStaticTypeSaveValue)
var LoadValueType = PrimitiveType(interpreter.PrimitiveStaticTypeLoadValue)
var BorrowValueType = PrimitiveType(interpreter.PrimitiveStaticTypeBorrowValue)
var ContractsType = PrimitiveType(interpreter.PrimitiveStaticTypeContracts)
var AddContractType = PrimitiveType(interpreter.PrimitiveStaticTypeAddContract)
var UpdateContractType = PrimitiveType(interpreter.PrimitiveStaticTypeUpdateContract)
var RemoveContractType = PrimitiveType(interpreter.PrimitiveStaticTypeRemoveContract)
var KeysType = PrimitiveType(interpreter.PrimitiveStaticTypeKeys)
var AddKeyType = PrimitiveType(interpreter.PrimitiveStaticTypeAddKey)
var RevokeKeyType = PrimitiveType(interpreter.PrimitiveStaticTypeRevokeKey)
var InboxType = PrimitiveType(interpreter.PrimitiveStaticTypeInbox)
var PublishInboxCapabilityType = PrimitiveType(interpreter.PrimitiveStaticTypePublishInboxCapability)
var UnpublishInboxCapabilityType = PrimitiveType(interpreter.PrimitiveStaticTypeUnpublishInboxCapability)
var ClaimInboxCapabilityType = PrimitiveType(interpreter.PrimitiveStaticTypeClaimInboxCapability)
var CapabilitiesType = PrimitiveType(interpreter.PrimitiveStaticTypeCapabilities)
var StorageCapabilitiesType = PrimitiveType(interpreter.PrimitiveStaticTypeStorageCapabilities)
var AccountCapabilitiesType = PrimitiveType(interpreter.PrimitiveStaticTypeAccountCapabilities)
var PublishCapabilityType = PrimitiveType(interpreter.PrimitiveStaticTypePublishCapability)
var UnpublishCapabilityType = PrimitiveType(interpreter.PrimitiveStaticTypeUnpublishCapability)
var GetStorageCapabilityControllerType = PrimitiveType(interpreter.PrimitiveStaticTypeGetStorageCapabilityController)
var IssueStorageCapabilityControllerType = PrimitiveType(interpreter.PrimitiveStaticTypeIssueStorageCapabilityController)
var GetAccountCapabilityControllerType = PrimitiveType(interpreter.PrimitiveStaticTypeGetAccountCapabilityController)
var IssueAccountCapabilityControllerType = PrimitiveType(interpreter.PrimitiveStaticTypeIssueAccountCapabilityController)

var CapabilitiesMappingType = PrimitiveType(interpreter.PrimitiveStaticTypeCapabilitiesMapping)
var AccountMappingType = PrimitiveType(interpreter.PrimitiveStaticTypeAccountMapping)

type ArrayType interface {
	Type
	Element() Type
}

// VariableSizedArrayType

type VariableSizedArrayType struct {
	ElementType Type
}

var _ Type = &VariableSizedArrayType{}

func NewVariableSizedArrayType(
	elementType Type,
) *VariableSizedArrayType {
	return &VariableSizedArrayType{ElementType: elementType}
}

func NewMeteredVariableSizedArrayType(
	gauge common.MemoryGauge,
	elementType Type,
) *VariableSizedArrayType {
	common.UseMemory(gauge, common.CadenceVariableSizedArrayTypeMemoryUsage)
	return NewVariableSizedArrayType(elementType)
}

func (*VariableSizedArrayType) isType() {}

func (t *VariableSizedArrayType) ID() string {
	return sema.FormatVariableSizedTypeID(t.ElementType.ID())
}

func (t *VariableSizedArrayType) Element() Type {
	return t.ElementType
}

func (t *VariableSizedArrayType) Equal(other Type) bool {
	otherType, ok := other.(*VariableSizedArrayType)
	if !ok {
		return false
	}

	return t.ElementType.Equal(otherType.ElementType)
}

// ConstantSizedArrayType

type ConstantSizedArrayType struct {
	ElementType Type
	Size        uint
	typeID      string
}

var _ Type = &ConstantSizedArrayType{}

func NewConstantSizedArrayType(
	size uint,
	elementType Type,
) *ConstantSizedArrayType {
	return &ConstantSizedArrayType{
		Size:        size,
		ElementType: elementType,
	}
}

func NewMeteredConstantSizedArrayType(
	gauge common.MemoryGauge,
	size uint,
	elementType Type,
) *ConstantSizedArrayType {
	common.UseMemory(gauge, common.CadenceConstantSizedArrayTypeMemoryUsage)
	return NewConstantSizedArrayType(size, elementType)
}

func (*ConstantSizedArrayType) isType() {}

func (t *ConstantSizedArrayType) ID() string {
	return sema.FormatConstantSizedTypeID(t.ElementType.ID(), int64(t.Size))
}

func (t *ConstantSizedArrayType) Element() Type {
	return t.ElementType
}

func (t *ConstantSizedArrayType) Equal(other Type) bool {
	otherType, ok := other.(*ConstantSizedArrayType)
	if !ok {
		return false
	}

	return t.ElementType.Equal(otherType.ElementType) &&
		t.Size == otherType.Size
}

// DictionaryType

type DictionaryType struct {
	KeyType     Type
	ElementType Type
}

var _ Type = &DictionaryType{}

func NewDictionaryType(
	keyType Type,
	elementType Type,
) *DictionaryType {
	return &DictionaryType{
		KeyType:     keyType,
		ElementType: elementType,
	}
}

func NewMeteredDictionaryType(
	gauge common.MemoryGauge,
	keyType Type,
	elementType Type,
) *DictionaryType {
	common.UseMemory(gauge, common.CadenceDictionaryTypeMemoryUsage)
	return NewDictionaryType(keyType, elementType)
}

func (*DictionaryType) isType() {}

func (t *DictionaryType) ID() string {
	return sema.FormatDictionaryTypeID(
		t.KeyType.ID(),
		t.ElementType.ID(),
	)
}

func (t *DictionaryType) Equal(other Type) bool {
	otherType, ok := other.(*DictionaryType)
	if !ok {
		return false
	}

	return t.KeyType.Equal(otherType.KeyType) &&
		t.ElementType.Equal(otherType.ElementType)
}

// Field

type Field struct {
	Type       Type
	Identifier string
}

// Fields are always created in an array, which must be metered ahead of time.
// So no metering here.
func NewField(identifier string, typ Type) Field {
	return Field{
		Identifier: identifier,
		Type:       typ,
	}
}

type HasFields interface {
	GetFields() []Field
	GetFieldValues() []Value
}

func GetFieldByName(v HasFields, fieldName string) Value {
	fieldValues := v.GetFieldValues()
	fields := v.GetFields()

	if fieldValues == nil || fields == nil {
		return nil
	}

	for i, field := range v.GetFields() {
		if field.Identifier == fieldName {
			return v.GetFieldValues()[i]
		}
	}
	return nil
}

func GetFieldsMappedByName(v HasFields) map[string]Value {
	fieldValues := v.GetFieldValues()
	fields := v.GetFields()

	if fieldValues == nil || fields == nil {
		return nil
	}

	fieldsMap := make(map[string]Value, len(fields))
	for i, field := range fields {
		fieldsMap[field.Identifier] = fieldValues[i]
	}
	return fieldsMap
}

// DecodeFields decodes a HasFields into a struct
func DecodeFields(hasFields HasFields, s interface{}) error {
	v := reflect.ValueOf(s)
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("s must be a pointer to a struct")
	}

	v = v.Elem()
	t := v.Type()

	fieldsMap := GetFieldsMappedByName(hasFields)

	for i := 0; i < v.NumField(); i++ {
		structField := t.Field(i)
		tag := structField.Tag
		fieldValue := v.Field(i)

		cadenceFieldNameTag := tag.Get("cadence")
		if cadenceFieldNameTag == "" {
			continue
		}

		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			return fmt.Errorf("cannot set field %s", structField.Name)
		}

		cadenceField := fieldsMap[cadenceFieldNameTag]
		if cadenceField == nil {
			return fmt.Errorf("%s field not found", cadenceFieldNameTag)
		}

		cadenceFieldValue := reflect.ValueOf(cadenceField)

		var decodeSpecialFieldFunc func(p reflect.Type, value Value) (*reflect.Value, error)

		switch fieldValue.Kind() {
		case reflect.Ptr:
			decodeSpecialFieldFunc = decodeOptional
		case reflect.Map:
			decodeSpecialFieldFunc = decodeDict
		case reflect.Array, reflect.Slice:
			decodeSpecialFieldFunc = decodeSlice
		}

		if decodeSpecialFieldFunc != nil {
			cadenceFieldValuePtr, err := decodeSpecialFieldFunc(fieldValue.Type(), cadenceField)
			if err != nil {
				return fmt.Errorf("cannot decode %s field %s: %w", fieldValue.Kind(), structField.Name, err)
			}
			cadenceFieldValue = *cadenceFieldValuePtr
		}

		if !cadenceFieldValue.CanConvert(fieldValue.Type()) {
			return fmt.Errorf(
				"cannot convert cadence field %s of type %s to struct field %s of type %s",
				cadenceFieldNameTag,
				cadenceField.Type().ID(),
				structField.Name,
				fieldValue.Type(),
			)
		}

		fieldValue.Set(cadenceFieldValue.Convert(fieldValue.Type()))
	}

	return nil
}

func decodeOptional(valueType reflect.Type, cadenceField Value) (*reflect.Value, error) {
	optional, ok := cadenceField.(Optional)
	if !ok {
		return nil, fmt.Errorf("field is not an optional")
	}

	// if optional is nil, skip and default the field to nil
	if optional.ToGoValue() == nil {
		zeroValue := reflect.Zero(valueType)
		return &zeroValue, nil
	}

	optionalValue := reflect.ValueOf(optional.Value)

	// Check the type
	if valueType.Elem() != optionalValue.Type() && valueType.Elem().Kind() != reflect.Interface {
		return nil, fmt.Errorf("cannot set field: expected %v, got %v",
			valueType.Elem(), optionalValue.Type())
	}

	if valueType.Elem().Kind() == reflect.Interface {
		newInterfaceVal := reflect.New(reflect.TypeOf((*interface{})(nil)).Elem())
		newInterfaceVal.Elem().Set(optionalValue)

		return &newInterfaceVal, nil
	}

	// Create a new pointer for optionalValue
	newPtr := reflect.New(optionalValue.Type())
	newPtr.Elem().Set(optionalValue)

	return &newPtr, nil
}

func decodeDict(valueType reflect.Type, cadenceField Value) (*reflect.Value, error) {
	dict, ok := cadenceField.(Dictionary)
	if !ok {
		return nil, fmt.Errorf("field is not a dictionary")
	}

	mapKeyType := valueType.Key()
	mapValueType := valueType.Elem()

	mapValue := reflect.MakeMap(valueType)
	for _, pair := range dict.Pairs {

		// Convert key and value to their Go counterparts
		var key, value reflect.Value
		if mapKeyType.Kind() == reflect.Ptr {
			return nil, fmt.Errorf("map key cannot be a pointer (optional) type")
		}
		key = reflect.ValueOf(pair.Key)

		if mapValueType.Kind() == reflect.Ptr {
			// If the map value is a pointer type, unwrap it from optional
			valueOptional, err := decodeOptional(mapValueType, pair.Value)
			if err != nil {
				return nil, fmt.Errorf("cannot decode optional map value for key %s: %w", pair.Key.String(), err)
			}
			value = *valueOptional
		} else {
			value = reflect.ValueOf(pair.Value)
		}

		if mapKeyType != key.Type() {
			return nil, fmt.Errorf("map key type mismatch: expected %v, got %v", mapKeyType, key.Type())
		}
		if mapValueType != value.Type() && mapValueType.Kind() != reflect.Interface {
			return nil, fmt.Errorf("map value type mismatch: expected %v, got %v", mapValueType, value.Type())
		}

		// Add key-value pair to the map
		mapValue.SetMapIndex(key, value)
	}

	return &mapValue, nil
}

func decodeSlice(valueType reflect.Type, cadenceField Value) (*reflect.Value, error) {
	array, ok := cadenceField.(Array)
	if !ok {
		return nil, fmt.Errorf("field is not an array")
	}

	var arrayValue reflect.Value

	constantSizeArray, ok := array.ArrayType.(*ConstantSizedArrayType)
	if ok {
		arrayValue = reflect.New(reflect.ArrayOf(int(constantSizeArray.Size), valueType.Elem())).Elem()
	} else {
		// If the array is not constant sized, create a slice
		arrayValue = reflect.MakeSlice(valueType, len(array.Values), len(array.Values))
	}

	for i, value := range array.Values {
		var elementValue reflect.Value
		if valueType.Elem().Kind() == reflect.Ptr {
			// If the array value is a pointer type, unwrap it from optional
			valueOptional, err := decodeOptional(valueType.Elem(), value)
			if err != nil {
				return nil, fmt.Errorf("error decoding array element optional: %w", err)
			}
			elementValue = *valueOptional
		} else {
			elementValue = reflect.ValueOf(value)
		}
		if elementValue.Type() != valueType.Elem() && valueType.Elem().Kind() != reflect.Interface {
			return nil, fmt.Errorf(
				"array element type mismatch at index %d: expected %v, got %v",
				i,
				valueType.Elem(),
				elementValue.Type(),
			)
		}

		arrayValue.Index(i).Set(elementValue)
	}

	return &arrayValue, nil
}

// Parameter

type Parameter struct {
	Type       Type
	Label      string
	Identifier string
}

func NewParameter(
	label string,
	identifier string,
	typ Type,
) Parameter {
	return Parameter{
		Label:      label,
		Identifier: identifier,
		Type:       typ,
	}
}

// TypeParameter

type TypeParameter struct {
	Name      string
	TypeBound Type
}

func NewTypeParameter(
	name string,
	typeBound Type,
) TypeParameter {
	return TypeParameter{
		Name:      name,
		TypeBound: typeBound,
	}
}

// CompositeType

type CompositeType interface {
	Type
	isCompositeType()
	CompositeTypeLocation() common.Location
	CompositeTypeQualifiedIdentifier() string
	CompositeFields() []Field
	SetCompositeFields([]Field)
	CompositeInitializers() [][]Parameter
}

// StructType

type StructType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

func NewStructType(
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *StructType {
	return &StructType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredStructType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *StructType {
	common.UseMemory(gauge, common.CadenceStructTypeMemoryUsage)
	return NewStructType(location, qualifiedIdentifier, fields, initializers)
}

func (*StructType) isType() {}

func (t *StructType) ID() string {
	return string(common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier))
}

func (*StructType) isCompositeType() {}

func (t *StructType) CompositeTypeLocation() common.Location {
	return t.Location
}

func (t *StructType) CompositeTypeQualifiedIdentifier() string {
	return t.QualifiedIdentifier
}

func (t *StructType) CompositeFields() []Field {
	return t.Fields
}

func (t *StructType) SetCompositeFields(fields []Field) {
	t.Fields = fields
}

func (t *StructType) CompositeInitializers() [][]Parameter {
	return t.Initializers
}

func (t *StructType) Equal(other Type) bool {
	otherType, ok := other.(*StructType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}

// ResourceType

type ResourceType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

func NewResourceType(
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceType {
	return &ResourceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredResourceType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceType {
	common.UseMemory(gauge, common.CadenceResourceTypeMemoryUsage)
	return NewResourceType(location, qualifiedIdentifier, fields, initializers)
}

func (*ResourceType) isType() {}

func (t *ResourceType) ID() string {
	return string(common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier))
}

func (*ResourceType) isCompositeType() {}

func (t *ResourceType) CompositeTypeLocation() common.Location {
	return t.Location
}

func (t *ResourceType) CompositeTypeQualifiedIdentifier() string {
	return t.QualifiedIdentifier
}

func (t *ResourceType) CompositeFields() []Field {
	return t.Fields
}

func (t *ResourceType) SetCompositeFields(fields []Field) {
	t.Fields = fields
}

func (t *ResourceType) CompositeInitializers() [][]Parameter {
	return t.Initializers
}

func (t *ResourceType) Equal(other Type) bool {
	otherType, ok := other.(*ResourceType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}

// AttachmentType

type AttachmentType struct {
	Location            common.Location
	BaseType            Type
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

func NewAttachmentType(
	location common.Location,
	baseType Type,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *AttachmentType {
	return &AttachmentType{
		Location:            location,
		BaseType:            baseType,
		QualifiedIdentifier: qualifiedIdentifier,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredAttachmentType(
	gauge common.MemoryGauge,
	location common.Location,
	baseType Type,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *AttachmentType {
	common.UseMemory(gauge, common.CadenceAttachmentTypeMemoryUsage)
	return NewAttachmentType(
		location,
		baseType,
		qualifiedIdentifier,
		fields,
		initializers,
	)
}

func (*AttachmentType) isType() {}

func (t *AttachmentType) ID() string {
	return string(common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier))
}

func (*AttachmentType) isCompositeType() {}

func (t *AttachmentType) CompositeTypeLocation() common.Location {
	return t.Location
}

func (t *AttachmentType) CompositeTypeQualifiedIdentifier() string {
	return t.QualifiedIdentifier
}

func (t *AttachmentType) CompositeFields() []Field {
	return t.Fields
}

func (t *AttachmentType) SetCompositeFields(fields []Field) {
	t.Fields = fields
}

func (t *AttachmentType) CompositeInitializers() [][]Parameter {
	return t.Initializers
}

func (t *AttachmentType) Base() Type {
	return t.BaseType
}

func (t *AttachmentType) Equal(other Type) bool {
	otherType, ok := other.(*AttachmentType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}

// EventType

type EventType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializer         []Parameter
}

func NewEventType(
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializer []Parameter,
) *EventType {
	return &EventType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Fields:              fields,
		Initializer:         initializer,
	}
}

func NewMeteredEventType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializer []Parameter,
) *EventType {
	common.UseMemory(gauge, common.CadenceEventTypeMemoryUsage)
	return NewEventType(location, qualifiedIdentifier, fields, initializer)
}

func (*EventType) isType() {}

func (t *EventType) ID() string {
	return string(common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier))
}

func (*EventType) isCompositeType() {}

func (t *EventType) CompositeTypeLocation() common.Location {
	return t.Location
}

func (t *EventType) CompositeTypeQualifiedIdentifier() string {
	return t.QualifiedIdentifier
}

func (t *EventType) CompositeFields() []Field {
	return t.Fields
}

func (t *EventType) SetCompositeFields(fields []Field) {
	t.Fields = fields
}

func (t *EventType) CompositeInitializers() [][]Parameter {
	return [][]Parameter{t.Initializer}
}

func (t *EventType) Equal(other Type) bool {
	otherType, ok := other.(*EventType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}

// ContractType

type ContractType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

func NewContractType(
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *ContractType {
	return &ContractType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredContractType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *ContractType {
	common.UseMemory(gauge, common.CadenceContractTypeMemoryUsage)
	return NewContractType(location, qualifiedIdentifier, fields, initializers)
}

func (*ContractType) isType() {}

func (t *ContractType) ID() string {
	return string(common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier))
}

func (*ContractType) isCompositeType() {}

func (t *ContractType) CompositeTypeLocation() common.Location {
	return t.Location
}

func (t *ContractType) CompositeTypeQualifiedIdentifier() string {
	return t.QualifiedIdentifier
}

func (t *ContractType) CompositeFields() []Field {
	return t.Fields
}

func (t *ContractType) SetCompositeFields(fields []Field) {
	t.Fields = fields
}

func (t *ContractType) CompositeInitializers() [][]Parameter {
	return t.Initializers
}

func (t *ContractType) Equal(other Type) bool {
	otherType, ok := other.(*ContractType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}

// InterfaceType

type InterfaceType interface {
	Type
	isInterfaceType()
	InterfaceTypeLocation() common.Location
	InterfaceTypeQualifiedIdentifier() string
	InterfaceFields() []Field
	SetInterfaceFields(fields []Field)
	InterfaceInitializers() [][]Parameter
}

// StructInterfaceType

type StructInterfaceType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

func NewStructInterfaceType(
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *StructInterfaceType {
	return &StructInterfaceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredStructInterfaceType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *StructInterfaceType {
	common.UseMemory(gauge, common.CadenceStructInterfaceTypeMemoryUsage)
	return NewStructInterfaceType(location, qualifiedIdentifier, fields, initializers)
}

func (*StructInterfaceType) isType() {}

func (t *StructInterfaceType) ID() string {
	return string(common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier))
}

func (*StructInterfaceType) isInterfaceType() {}

func (t *StructInterfaceType) InterfaceTypeLocation() common.Location {
	return t.Location
}

func (t *StructInterfaceType) InterfaceTypeQualifiedIdentifier() string {
	return t.QualifiedIdentifier
}

func (t *StructInterfaceType) InterfaceFields() []Field {
	return t.Fields
}

func (t *StructInterfaceType) SetInterfaceFields(fields []Field) {
	t.Fields = fields
}

func (t *StructInterfaceType) InterfaceInitializers() [][]Parameter {
	return t.Initializers
}

func (t *StructInterfaceType) Equal(other Type) bool {
	otherType, ok := other.(*StructInterfaceType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}

// ResourceInterfaceType

type ResourceInterfaceType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

func NewResourceInterfaceType(
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceInterfaceType {
	return &ResourceInterfaceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredResourceInterfaceType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceInterfaceType {
	common.UseMemory(gauge, common.CadenceResourceInterfaceTypeMemoryUsage)
	return NewResourceInterfaceType(location, qualifiedIdentifier, fields, initializers)
}

func (*ResourceInterfaceType) isType() {}

func (t *ResourceInterfaceType) ID() string {
	return string(common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier))
}

func (*ResourceInterfaceType) isInterfaceType() {}

func (t *ResourceInterfaceType) InterfaceTypeLocation() common.Location {
	return t.Location
}

func (t *ResourceInterfaceType) InterfaceTypeQualifiedIdentifier() string {
	return t.QualifiedIdentifier
}

func (t *ResourceInterfaceType) InterfaceFields() []Field {
	return t.Fields
}

func (t *ResourceInterfaceType) SetInterfaceFields(fields []Field) {
	t.Fields = fields
}

func (t *ResourceInterfaceType) InterfaceInitializers() [][]Parameter {
	return t.Initializers
}

func (t *ResourceInterfaceType) Equal(other Type) bool {
	otherType, ok := other.(*ResourceInterfaceType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}

// ContractInterfaceType

type ContractInterfaceType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

func NewContractInterfaceType(
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *ContractInterfaceType {
	return &ContractInterfaceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredContractInterfaceType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	fields []Field,
	initializers [][]Parameter,
) *ContractInterfaceType {
	common.UseMemory(gauge, common.CadenceContractInterfaceTypeMemoryUsage)
	return NewContractInterfaceType(location, qualifiedIdentifier, fields, initializers)
}

func (*ContractInterfaceType) isType() {}

func (t *ContractInterfaceType) ID() string {
	return string(common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier))
}

func (*ContractInterfaceType) isInterfaceType() {}

func (t *ContractInterfaceType) InterfaceTypeLocation() common.Location {
	return t.Location
}

func (t *ContractInterfaceType) InterfaceTypeQualifiedIdentifier() string {
	return t.QualifiedIdentifier
}

func (t *ContractInterfaceType) InterfaceFields() []Field {
	return t.Fields
}

func (t *ContractInterfaceType) SetInterfaceFields(fields []Field) {
	t.Fields = fields
}

func (t *ContractInterfaceType) InterfaceInitializers() [][]Parameter {
	return t.Initializers
}

func (t *ContractInterfaceType) Equal(other Type) bool {
	otherType, ok := other.(*ContractInterfaceType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}

// Function

type FunctionPurity int

const (
	FunctionPurityUnspecified FunctionPurity = iota
	FunctionPurityView
)

type FunctionType struct {
	TypeParameters []TypeParameter
	Parameters     []Parameter
	ReturnType     Type
	Purity         FunctionPurity
}

func NewFunctionType(
	purity FunctionPurity,
	typeParameters []TypeParameter,
	parameters []Parameter,
	returnType Type,
) *FunctionType {
	return &FunctionType{
		Purity:         purity,
		TypeParameters: typeParameters,
		Parameters:     parameters,
		ReturnType:     returnType,
	}
}

func NewMeteredFunctionType(
	gauge common.MemoryGauge,
	purity FunctionPurity,
	typeParameters []TypeParameter,
	parameters []Parameter,
	returnType Type,
) *FunctionType {
	common.UseMemory(gauge, common.CadenceFunctionTypeMemoryUsage)
	return NewFunctionType(purity, typeParameters, parameters, returnType)
}

func (*FunctionType) isType() {}

func (t *FunctionType) ID() string {

	var purity string
	if t.Purity == FunctionPurityView {
		purity = "view"
	}

	typeParameterCount := len(t.TypeParameters)
	var typeParameters []string
	if typeParameterCount > 0 {
		typeParameters = make([]string, typeParameterCount)
		for i, typeParameter := range t.TypeParameters {
			typeParameters[i] = typeParameter.Name
		}
	}

	parameterCount := len(t.Parameters)
	var parameters []string
	if parameterCount > 0 {
		parameters = make([]string, parameterCount)
		for i, parameter := range t.Parameters {
			parameters[i] = parameter.Type.ID()
		}
	}

	returnType := t.ReturnType.ID()

	return sema.FormatFunctionTypeID(
		purity,
		typeParameters,
		parameters,
		returnType,
	)
}

func (t *FunctionType) Equal(other Type) bool {
	otherType, ok := other.(*FunctionType)
	if !ok {
		return false
	}

	// Type parameters

	if len(t.TypeParameters) != len(otherType.TypeParameters) {
		return false
	}

	for i, typeParameter := range t.TypeParameters {
		otherTypeParameter := otherType.TypeParameters[i]

		if typeParameter.TypeBound == nil {
			if otherTypeParameter.TypeBound != nil {
				return false
			}
		} else if otherTypeParameter.TypeBound == nil ||
			!typeParameter.TypeBound.Equal(otherTypeParameter.TypeBound) {

			return false
		}
	}

	// Parameters

	if len(t.Parameters) != len(otherType.Parameters) {
		return false
	}

	for i, parameter := range t.Parameters {
		otherParameter := otherType.Parameters[i]
		if !parameter.Type.Equal(otherParameter.Type) {
			return false
		}
	}

	return t.ReturnType.Equal(otherType.ReturnType)
}

// Authorization

type Authorization interface {
	isAuthorization()
	ID() string
	Equal(auth Authorization) bool
}

type Unauthorized struct{}

var UnauthorizedAccess Authorization = Unauthorized{}

func (Unauthorized) isAuthorization() {}

func (Unauthorized) ID() string {
	panic(errors.NewUnreachableError())
}

func (Unauthorized) Equal(other Authorization) bool {
	_, ok := other.(Unauthorized)
	return ok
}

type EntitlementSetKind = sema.EntitlementSetKind

const Conjunction = sema.Conjunction
const Disjunction = sema.Disjunction

type EntitlementSetAuthorization struct {
	Entitlements []common.TypeID
	Kind         EntitlementSetKind
}

var _ Authorization = EntitlementSetAuthorization{}

func NewEntitlementSetAuthorization(
	gauge common.MemoryGauge,
	entitlements []common.TypeID,
	kind EntitlementSetKind,
) EntitlementSetAuthorization {
	common.UseMemory(gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceEntitlementSetAccess,
		Amount: uint64(len(entitlements)),
	})
	return EntitlementSetAuthorization{
		Entitlements: entitlements,
		Kind:         kind,
	}
}

func (EntitlementSetAuthorization) isAuthorization() {}

func (e EntitlementSetAuthorization) ID() string {
	entitlementTypeIDs := make([]string, 0, len(e.Entitlements))
	for _, typeID := range e.Entitlements {
		entitlementTypeIDs = append(
			entitlementTypeIDs,
			string(typeID),
		)
	}

	// FormatEntitlementSetTypeID sorts
	return sema.FormatEntitlementSetTypeID(entitlementTypeIDs, e.Kind)
}

func (e EntitlementSetAuthorization) Equal(auth Authorization) bool {
	switch auth := auth.(type) {
	case EntitlementSetAuthorization:
		if len(e.Entitlements) != len(auth.Entitlements) {
			return false
		}

		for i, entitlement := range e.Entitlements {
			if auth.Entitlements[i] != entitlement {
				return false
			}
		}
		return e.Kind == auth.Kind
	}
	return false
}

type EntitlementMapAuthorization struct {
	TypeID common.TypeID
}

var _ Authorization = EntitlementMapAuthorization{}

func NewEntitlementMapAuthorization(gauge common.MemoryGauge, id common.TypeID) EntitlementMapAuthorization {
	common.UseMemory(gauge, common.NewConstantMemoryUsage(common.MemoryKindCadenceEntitlementMapAccess))
	return EntitlementMapAuthorization{
		TypeID: id,
	}
}

func (EntitlementMapAuthorization) isAuthorization() {}

func (e EntitlementMapAuthorization) ID() string {
	return string(e.TypeID)
}

func (e EntitlementMapAuthorization) Equal(other Authorization) bool {
	auth, ok := other.(EntitlementMapAuthorization)
	if !ok {
		return false
	}
	return e.TypeID == auth.TypeID
}

// ReferenceType

type ReferenceType struct {
	Type          Type
	Authorization Authorization
}

var _ Type = &ReferenceType{}

func NewReferenceType(
	authorization Authorization,
	typ Type,
) *ReferenceType {
	return &ReferenceType{
		Authorization: authorization,
		Type:          typ,
	}
}

func NewMeteredReferenceType(
	gauge common.MemoryGauge,
	authorization Authorization,
	typ Type,
) *ReferenceType {
	common.UseMemory(gauge, common.CadenceReferenceTypeMemoryUsage)
	return NewReferenceType(authorization, typ)
}

func (*ReferenceType) isType() {}

func (t *ReferenceType) ID() string {
	var authorization string
	if t.Authorization != UnauthorizedAccess {
		authorization = t.Authorization.ID()
	}
	return sema.FormatReferenceTypeID(
		authorization,
		t.Type.ID(),
	)
}

func (t *ReferenceType) Equal(other Type) bool {
	otherType, ok := other.(*ReferenceType)
	if !ok {
		return false
	}

	return t.Authorization.Equal(otherType.Authorization) &&
		t.Type.Equal(otherType.Type)
}

// IntersectionType

type IntersectionSet = map[Type]struct{}

type IntersectionType struct {
	Types               []Type
	intersectionSet     IntersectionSet
	intersectionSetOnce sync.Once
}

func NewIntersectionType(
	types []Type,
) *IntersectionType {
	return &IntersectionType{
		Types: types,
	}
}

func NewMeteredIntersectionType(
	gauge common.MemoryGauge,
	types []Type,
) *IntersectionType {
	common.UseMemory(gauge, common.CadenceIntersectionTypeMemoryUsage)
	return NewIntersectionType(types)
}

func (*IntersectionType) isType() {}

func (t *IntersectionType) ID() string {
	var interfaceTypeIDs []string
	typeCount := len(t.Types)
	if typeCount > 0 {
		interfaceTypeIDs = make([]string, 0, typeCount)
		for _, typ := range t.Types {
			interfaceTypeIDs = append(interfaceTypeIDs, typ.ID())
		}
	}
	// FormatIntersectionTypeID sorts
	return sema.FormatIntersectionTypeID(interfaceTypeIDs)
}

func (t *IntersectionType) Equal(other Type) bool {
	otherType, ok := other.(*IntersectionType)
	if !ok {
		return false
	}

	intersectionSet := t.IntersectionSet()
	otherIntersectionSet := otherType.IntersectionSet()

	if len(intersectionSet) != len(otherIntersectionSet) {
		return false
	}

	for typ := range intersectionSet { //nolint:maprange
		_, ok := otherIntersectionSet[typ]
		if !ok {
			return false
		}
	}

	return true
}

func (t *IntersectionType) initializeIntersectionSet() {
	t.intersectionSetOnce.Do(func() {
		t.intersectionSet = make(IntersectionSet, len(t.Types))
		for _, typ := range t.Types {
			t.intersectionSet[typ] = struct{}{}
		}
	})
}

func (t *IntersectionType) IntersectionSet() IntersectionSet {
	t.initializeIntersectionSet()
	return t.intersectionSet
}

// CapabilityType

type CapabilityType struct {
	BorrowType Type
}

var _ Type = &CapabilityType{}

func NewCapabilityType(borrowType Type) *CapabilityType {
	return &CapabilityType{BorrowType: borrowType}
}

func NewMeteredCapabilityType(
	gauge common.MemoryGauge,
	borrowType Type,
) *CapabilityType {
	common.UseMemory(gauge, common.CadenceCapabilityTypeMemoryUsage)
	return NewCapabilityType(borrowType)
}

func (*CapabilityType) isType() {}

func (t *CapabilityType) ID() string {
	var borrowTypeID string
	borrowType := t.BorrowType
	if borrowType != nil {
		borrowTypeID = borrowType.ID()
	}
	return sema.FormatCapabilityTypeID(borrowTypeID)
}

func (t *CapabilityType) Equal(other Type) bool {
	otherType, ok := other.(*CapabilityType)
	if !ok {
		return false
	}

	if t.BorrowType == nil {
		return otherType.BorrowType == nil
	}

	return t.BorrowType.Equal(otherType.BorrowType)
}

// EnumType

type EnumType struct {
	Location            common.Location
	QualifiedIdentifier string
	RawType             Type
	Fields              []Field
	Initializers        [][]Parameter
}

func NewEnumType(
	location common.Location,
	qualifiedIdentifier string,
	rawType Type,
	fields []Field,
	initializers [][]Parameter,
) *EnumType {
	return &EnumType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		RawType:             rawType,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredEnumType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	rawType Type,
	fields []Field,
	initializers [][]Parameter,
) *EnumType {
	common.UseMemory(gauge, common.CadenceEnumTypeMemoryUsage)
	return NewEnumType(location, qualifiedIdentifier, rawType, fields, initializers)
}

func (*EnumType) isType() {}

func (t *EnumType) ID() string {
	return string(common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier))
}

func (*EnumType) isCompositeType() {}

func (t *EnumType) CompositeTypeLocation() common.Location {
	return t.Location
}

func (t *EnumType) CompositeTypeQualifiedIdentifier() string {
	return t.QualifiedIdentifier
}

func (t *EnumType) CompositeFields() []Field {
	return t.Fields
}

func (t *EnumType) SetCompositeFields(fields []Field) {
	t.Fields = fields
}

func (t *EnumType) CompositeInitializers() [][]Parameter {
	return t.Initializers
}

func (t *EnumType) Equal(other Type) bool {
	otherType, ok := other.(*EnumType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}
