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

package cadence

import (
	"fmt"
	"reflect"
	"strings"
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
var HashableStructType = PrimitiveType(interpreter.PrimitiveStaticTypeHashableStruct)

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
var FixedSizeUnsignedIntegerType = PrimitiveType(interpreter.PrimitiveStaticTypeFixedSizeUnsignedInteger)

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
var CopyValueType = PrimitiveType(interpreter.PrimitiveStaticTypeCopyValue)
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

// InclusiveRangeType

type InclusiveRangeType struct {
	ElementType Type
	typeID      string
}

var _ Type = &InclusiveRangeType{}

func NewInclusiveRangeType(
	elementType Type,
) *InclusiveRangeType {
	return &InclusiveRangeType{
		ElementType: elementType,
	}
}

func NewMeteredInclusiveRangeType(
	gauge common.MemoryGauge,
	elementType Type,
) *InclusiveRangeType {
	common.UseMemory(gauge, common.CadenceInclusiveRangeTypeMemoryUsage)
	return NewInclusiveRangeType(elementType)
}

func (*InclusiveRangeType) isType() {}

func (t *InclusiveRangeType) ID() string {
	if t.typeID == "" {
		t.typeID = fmt.Sprintf(
			"InclusiveRange<%s>",
			t.ElementType.ID(),
		)
	}
	return t.typeID
}

func (t *InclusiveRangeType) Equal(other Type) bool {
	otherType, ok := other.(*InclusiveRangeType)
	if !ok {
		return false
	}

	return t.ElementType.Equal(otherType.ElementType)
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

// SearchCompositeFieldTypeByName searches for the field with the given name in the composite type,
// and returns the type of the field, or nil if the field is not found.
//
// WARNING: This function performs a linear search, so is not efficient for accessing multiple fields.
// Prefer using CompositeFieldTypesMappedByName if you need to access multiple fields.
func SearchCompositeFieldTypeByName(compositeType CompositeType, fieldName string) Type {
	fields := compositeType.compositeFields()

	if fields == nil {
		return nil
	}

	for _, field := range fields {
		if field.Identifier == fieldName {
			return field.Type
		}
	}
	return nil
}

func CompositeFieldTypesMappedByName(compositeType CompositeType) map[string]Type {
	fields := compositeType.compositeFields()

	if fields == nil {
		return nil
	}

	fieldsMap := make(map[string]Type, len(fields))
	for _, field := range fields {
		fieldsMap[field.Identifier] = field.Type
	}

	return fieldsMap
}

// SearchInterfaceFieldTypeByName searches for the field with the given name in the interface type,
// and returns the type of the field, or nil if the field is not found.
//
// WARNING: This function performs a linear search, so is not efficient for accessing multiple fields.
// Prefer using InterfaceFieldTypesMappedByName if you need to access multiple fields.
func SearchInterfaceFieldTypeByName(interfaceType InterfaceType, fieldName string) Type {
	fields := interfaceType.interfaceFields()

	if fields == nil {
		return nil
	}

	for _, field := range fields {
		if field.Identifier == fieldName {
			return field.Type
		}
	}
	return nil
}

func InterfaceFieldTypesMappedByName(interfaceType InterfaceType) map[string]Type {
	fields := interfaceType.interfaceFields()

	if fields == nil {
		return nil
	}

	fieldsMap := make(map[string]Type, len(fields))
	for _, field := range fields {
		fieldsMap[field.Identifier] = field.Type
	}

	return fieldsMap
}

// DecodeFields decodes a HasFields into a struct
func DecodeFields(composite Composite, s interface{}) error {
	v := reflect.ValueOf(s)
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("s must be a pointer to a struct")
	}

	v = v.Elem()
	targetType := v.Type()

	_, err := decodeStructInto(v, targetType, composite)
	if err != nil {
		return err
	}

	return nil
}

func decodeFieldValue(targetType reflect.Type, value Value) (reflect.Value, error) {
	var decodeSpecialFieldFunc func(p reflect.Type, value Value) (reflect.Value, error)

	switch targetType.Kind() {
	case reflect.Ptr:
		decodeSpecialFieldFunc = decodeOptional
	case reflect.Map:
		decodeSpecialFieldFunc = decodeDict
	case reflect.Array, reflect.Slice:
		decodeSpecialFieldFunc = decodeSlice
	case reflect.Struct:
		if !targetType.Implements(reflect.TypeOf((*Value)(nil)).Elem()) {
			decodeSpecialFieldFunc = decodeStruct
		}
	}

	var reflectedValue reflect.Value

	if decodeSpecialFieldFunc != nil {
		var err error
		reflectedValue, err = decodeSpecialFieldFunc(targetType, value)
		if err != nil {
			ty := value.Type()
			if ty == nil {
				return reflect.Value{}, fmt.Errorf(
					"cannot convert Cadence value to Go type %s: %w",
					targetType,
					err,
				)
			} else {
				return reflect.Value{}, fmt.Errorf(
					"cannot convert Cadence value of type %s to Go type %s: %w",
					ty.ID(),
					targetType,
					err,
				)
			}
		}
	} else {
		reflectedValue = reflect.ValueOf(value)
	}

	if !reflectedValue.CanConvert(targetType) {
		ty := value.Type()
		if ty == nil {
			return reflect.Value{}, fmt.Errorf(
				"cannot convert Cadence value to Go type %s",
				targetType,
			)
		} else {
			return reflect.Value{}, fmt.Errorf(
				"cannot convert Cadence value of type %s to Go type %s",
				ty.ID(),
				targetType,
			)
		}
	}

	return reflectedValue.Convert(targetType), nil
}

func decodeOptional(pointerTargetType reflect.Type, cadenceValue Value) (reflect.Value, error) {
	cadenceOptional, ok := cadenceValue.(Optional)
	if !ok {
		return reflect.Value{}, fmt.Errorf("field is not an optional")
	}

	// If Cadence optional is nil, skip and default the field to Go nil
	cadenceInnerValue := cadenceOptional.Value
	if cadenceInnerValue == nil {
		return reflect.Zero(pointerTargetType), nil
	}

	// Create a new pointer
	newPtr := reflect.New(pointerTargetType.Elem())

	innerValue, err := decodeFieldValue(
		pointerTargetType.Elem(),
		cadenceInnerValue,
	)
	if err != nil {
		return reflect.Value{}, fmt.Errorf(
			"cannot decode optional value: %w",
			err,
		)
	}

	newPtr.Elem().Set(innerValue)

	return newPtr, nil
}

func decodeDict(mapTargetType reflect.Type, cadenceValue Value) (reflect.Value, error) {
	cadenceDictionary, ok := cadenceValue.(Dictionary)
	if !ok {
		return reflect.Value{}, fmt.Errorf(
			"cannot decode non-Cadence dictionary %T to Go map",
			cadenceValue,
		)
	}

	keyTargetType := mapTargetType.Key()
	valueTargetType := mapTargetType.Elem()

	mapValue := reflect.MakeMap(mapTargetType)

	for _, pair := range cadenceDictionary.Pairs {

		key, err := decodeFieldValue(keyTargetType, pair.Key)
		if err != nil {
			return reflect.Value{}, fmt.Errorf(
				"cannot decode dictionary key: %w",
				err,
			)
		}

		value, err := decodeFieldValue(valueTargetType, pair.Value)
		if err != nil {
			return reflect.Value{}, fmt.Errorf(
				"cannot decode dictionary value: %w",
				err,
			)
		}

		mapValue.SetMapIndex(key, value)
	}

	return mapValue, nil
}

func decodeSlice(arrayTargetType reflect.Type, cadenceValue Value) (reflect.Value, error) {
	cadenceArray, ok := cadenceValue.(Array)
	if !ok {
		return reflect.Value{}, fmt.Errorf(
			"cannot decode non-Cadence array %T to Go slice",
			cadenceValue,
		)
	}

	elementTargetType := arrayTargetType.Elem()

	var arrayValue reflect.Value

	cadenceConstantSizeArrayType, ok := cadenceArray.ArrayType.(*ConstantSizedArrayType)
	if ok {
		// If the Cadence array is constant-sized, create a Go array
		size := int(cadenceConstantSizeArrayType.Size)
		arrayValue = reflect.New(reflect.ArrayOf(size, elementTargetType)).Elem()
	} else {
		// If the Cadence array is not constant-sized, create a Go slice
		size := len(cadenceArray.Values)
		arrayValue = reflect.MakeSlice(arrayTargetType, size, size)
	}

	for i, cadenceElement := range cadenceArray.Values {
		elementValue, err := decodeFieldValue(elementTargetType, cadenceElement)
		if err != nil {
			return reflect.Value{}, fmt.Errorf(
				"cannot decode array element %d: %w",
				i,
				err,
			)
		}

		arrayValue.Index(i).Set(elementValue)
	}

	return arrayValue, nil
}

func decodeStruct(structTargetType reflect.Type, cadenceValue Value) (reflect.Value, error) {
	structValue := reflect.New(structTargetType)
	return decodeStructInto(structValue.Elem(), structTargetType, cadenceValue)
}

func decodeStructInto(
	structValue reflect.Value,
	structTargetType reflect.Type,
	cadenceValue Value,
) (reflect.Value, error) {
	composite, ok := cadenceValue.(Composite)
	if !ok {
		return reflect.Value{}, fmt.Errorf(
			"cannot decode non-Cadence composite %T to Go struct",
			cadenceValue,
		)
	}

	fieldsMap := FieldsMappedByName(composite)

	for i := 0; i < structValue.NumField(); i++ {
		structField := structTargetType.Field(i)
		tag := structField.Tag
		fieldValue := structValue.Field(i)

		cadenceFieldNameTag := tag.Get("cadence")
		if cadenceFieldNameTag == "" {
			continue
		}

		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			return reflect.Value{}, fmt.Errorf("cannot set field %s", structField.Name)
		}

		value := fieldsMap[cadenceFieldNameTag]
		if value == nil {
			return reflect.Value{}, fmt.Errorf("%s field not found", cadenceFieldNameTag)
		}

		converted, err := decodeFieldValue(fieldValue.Type(), value)
		if err != nil {
			return reflect.Value{}, fmt.Errorf(
				"cannot convert Cadence field %s into Go field %s: %w",
				cadenceFieldNameTag,
				structField.Name,
				err,
			)
		}

		fieldValue.Set(converted)
	}

	return structValue, nil
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
	compositeFields() []Field
	setCompositeFields([]Field)

	CompositeTypeLocation() common.Location
	CompositeTypeQualifiedIdentifier() string
	CompositeInitializers() [][]Parameter
	SearchFieldByName(fieldName string) Type
	FieldsMappedByName() map[string]Type
}

// linked in by packages that need access to CompositeType.setCompositeFields,
// e.g. JSON and CCF codecs
func setCompositeTypeFields(compositeType CompositeType, fields []Field) { //nolint:unused
	compositeType.setCompositeFields(fields)
}

// linked in by packages that need access to CompositeType.compositeFields,
// e.g. JSON and CCF codecs
func getCompositeTypeFields(compositeType CompositeType) []Field { //nolint:unused
	return compositeType.compositeFields()
}

// StructType

type StructType struct {
	Location            common.Location
	QualifiedIdentifier string
	fields              []Field
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
		fields:              fields,
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

func (t *StructType) compositeFields() []Field {
	return t.fields
}

func (t *StructType) setCompositeFields(fields []Field) {
	t.fields = fields
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

func (t *StructType) SearchFieldByName(fieldName string) Type {
	return SearchCompositeFieldTypeByName(t, fieldName)
}

func (t *StructType) FieldsMappedByName() map[string]Type {
	return CompositeFieldTypesMappedByName(t)
}

// ResourceType

type ResourceType struct {
	Location            common.Location
	QualifiedIdentifier string
	fields              []Field
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
		fields:              fields,
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

func (t *ResourceType) compositeFields() []Field {
	return t.fields
}

func (t *ResourceType) setCompositeFields(fields []Field) {
	t.fields = fields
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

func (t *ResourceType) SearchFieldByName(fieldName string) Type {
	return SearchCompositeFieldTypeByName(t, fieldName)
}

func (t *ResourceType) FieldsMappedByName() map[string]Type {
	return CompositeFieldTypesMappedByName(t)
}

// AttachmentType

type AttachmentType struct {
	Location            common.Location
	BaseType            Type
	QualifiedIdentifier string
	fields              []Field
	Initializers        [][]Parameter
}

func NewAttachmentType(
	location common.Location,
	qualifiedIdentifier string,
	baseType Type,
	fields []Field,
	initializers [][]Parameter,
) *AttachmentType {
	return &AttachmentType{
		Location:            location,
		BaseType:            baseType,
		QualifiedIdentifier: qualifiedIdentifier,
		fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredAttachmentType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	baseType Type,
	fields []Field,
	initializers [][]Parameter,
) *AttachmentType {
	common.UseMemory(gauge, common.CadenceAttachmentTypeMemoryUsage)
	return NewAttachmentType(
		location,
		qualifiedIdentifier,
		baseType,
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

func (t *AttachmentType) compositeFields() []Field {
	return t.fields
}

func (t *AttachmentType) setCompositeFields(fields []Field) {
	t.fields = fields
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

func (t *AttachmentType) SearchFieldByName(fieldName string) Type {
	return SearchCompositeFieldTypeByName(t, fieldName)
}

func (t *AttachmentType) FieldsMappedByName() map[string]Type {
	return CompositeFieldTypesMappedByName(t)
}

// EventType

type EventType struct {
	Location            common.Location
	QualifiedIdentifier string
	fields              []Field
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
		fields:              fields,
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

func (t *EventType) compositeFields() []Field {
	return t.fields
}

func (t *EventType) setCompositeFields(fields []Field) {
	t.fields = fields
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

func (t *EventType) SearchFieldByName(fieldName string) Type {
	return SearchCompositeFieldTypeByName(t, fieldName)
}

func (t *EventType) FieldsMappedByName() map[string]Type {
	return CompositeFieldTypesMappedByName(t)
}

// ContractType

type ContractType struct {
	Location            common.Location
	QualifiedIdentifier string
	fields              []Field
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
		fields:              fields,
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

func (t *ContractType) compositeFields() []Field {
	return t.fields
}

func (t *ContractType) setCompositeFields(fields []Field) {
	t.fields = fields
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

func (t *ContractType) SearchFieldByName(fieldName string) Type {
	return SearchCompositeFieldTypeByName(t, fieldName)
}

func (t *ContractType) FieldsMappedByName() map[string]Type {
	return CompositeFieldTypesMappedByName(t)
}

// InterfaceType

type InterfaceType interface {
	Type

	isInterfaceType()
	interfaceFields() []Field
	setInterfaceFields(fields []Field)

	InterfaceTypeLocation() common.Location
	InterfaceTypeQualifiedIdentifier() string
	InterfaceInitializers() [][]Parameter
}

// linked in by packages that need access to InterfaceType.interfaceFields,
// e.g. JSON and CCF codecs
func getInterfaceTypeFields(interfaceType InterfaceType) []Field { //nolint:unused
	return interfaceType.interfaceFields()
}

// linked in by packages that need access to InterfaceType.setInterfaceFields,
// e.g. JSON and CCF codecs
func setInterfaceTypeFields(interfaceType InterfaceType, fields []Field) { //nolint:unused
	interfaceType.setInterfaceFields(fields)
}

// StructInterfaceType

type StructInterfaceType struct {
	Location            common.Location
	QualifiedIdentifier string
	fields              []Field
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
		fields:              fields,
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

func (t *StructInterfaceType) interfaceFields() []Field {
	return t.fields
}

func (t *StructInterfaceType) setInterfaceFields(fields []Field) {
	t.fields = fields
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
	fields              []Field
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
		fields:              fields,
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

func (t *ResourceInterfaceType) interfaceFields() []Field {
	return t.fields
}

func (t *ResourceInterfaceType) setInterfaceFields(fields []Field) {
	t.fields = fields
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
	fields              []Field
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
		fields:              fields,
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

func (t *ContractInterfaceType) interfaceFields() []Field {
	return t.fields
}

func (t *ContractInterfaceType) setInterfaceFields(fields []Field) {
	t.fields = fields
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

	// DO NOT add item after maxFunctionPurity
	maxFunctionPurity
)

func NewFunctionaryPurity(rawPurity int) (FunctionPurity, error) {
	if rawPurity < 0 || rawPurity >= int(maxFunctionPurity) {
		return FunctionPurityUnspecified, fmt.Errorf("failed to convert %d to FunctionPurity", rawPurity)
	}
	return FunctionPurity(rawPurity), nil
}

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

	// Purity

	if t.Purity != otherType.Purity {
		return false
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
	Entitlements       []common.TypeID
	Kind               EntitlementSetKind
	entitlementSet     map[common.TypeID]struct{}
	entitlementSetOnce sync.Once
}

var _ Authorization = &EntitlementSetAuthorization{}

func NewEntitlementSetAuthorization(
	gauge common.MemoryGauge,
	entitlements []common.TypeID,
	kind EntitlementSetKind,
) *EntitlementSetAuthorization {
	common.UseMemory(gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceEntitlementSetAccess,
		Amount: uint64(len(entitlements)),
	})
	return &EntitlementSetAuthorization{
		Entitlements: entitlements,
		Kind:         kind,
	}
}

func (*EntitlementSetAuthorization) isAuthorization() {}

func (e *EntitlementSetAuthorization) ID() string {
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

func (e *EntitlementSetAuthorization) Equal(auth Authorization) bool {
	switch auth := auth.(type) {
	case *EntitlementSetAuthorization:
		if len(e.Entitlements) != len(auth.Entitlements) {
			return false
		}

		// sets are equivalent if they contain the same elements, regardless of order
		otherEntitlementSet := auth.getEntitlementSet()

		for _, entitlement := range e.Entitlements {
			if _, exist := otherEntitlementSet[entitlement]; !exist {
				return false
			}
		}
		return e.Kind == auth.Kind
	}
	return false
}

func (t *EntitlementSetAuthorization) initializeEntitlementSet() {
	t.entitlementSetOnce.Do(func() {
		t.entitlementSet = make(map[common.TypeID]struct{}, len(t.Entitlements))
		for _, e := range t.Entitlements {
			t.entitlementSet[e] = struct{}{}
		}
	})
}

func (t *EntitlementSetAuthorization) getEntitlementSet() map[common.TypeID]struct{} {
	t.initializeEntitlementSet()
	return t.entitlementSet
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

// DeprecatedReferenceType
// Deprecated: removed in v1.0.0
type DeprecatedReferenceType struct {
	Type       Type
	Authorized bool
	typeID     string
}

var _ Type = &DeprecatedReferenceType{}

func NewDeprecatedReferenceType(
	authorized bool,
	typ Type,
) *DeprecatedReferenceType {
	return &DeprecatedReferenceType{
		Authorized: authorized,
		Type:       typ,
	}
}

func NewDeprecatedMeteredReferenceType(
	gauge common.MemoryGauge,
	authorized bool,
	typ Type,
) *DeprecatedReferenceType {
	common.UseMemory(gauge, common.CadenceReferenceTypeMemoryUsage)
	return NewDeprecatedReferenceType(authorized, typ)
}

func (*DeprecatedReferenceType) isType() {}

func (t *DeprecatedReferenceType) ID() string {
	if t.typeID == "" {
		t.typeID = formatDeprecatedReferenceTypeID(t.Authorized, t.Type.ID()) //nolint:staticcheck
	}
	return t.typeID
}

func (t *DeprecatedReferenceType) Equal(other Type) bool {
	otherType, ok := other.(*DeprecatedReferenceType)
	if !ok {
		return false
	}

	return t.Authorized == otherType.Authorized &&
		t.Type.Equal(otherType.Type)
}

// Deprecated: use FormatReferenceTypeID
func formatDeprecatedReferenceTypeID(authorized bool, typeString string) string {
	return formatDeprecatedReferenceType("", authorized, typeString)
}

// Deprecated: use FormatReferenceTypeID
func formatDeprecatedReferenceType(
	separator string,
	authorized bool,
	typeString string,
) string {
	var builder strings.Builder
	if authorized {
		builder.WriteString("auth")
		builder.WriteString(separator)
	}
	builder.WriteByte('&')
	builder.WriteString(typeString)
	return builder.String()
}

// DeprecatedRestrictedType
// Deprecated: removed in v1.0.0
type DeprecatedRestrictedType struct {
	typeID             string
	Type               Type
	Restrictions       []Type
	restrictionSet     DeprecatedRestrictionSet
	restrictionSetOnce sync.Once
}

// Deprecated: removed in v1.0.0
type DeprecatedRestrictionSet = map[Type]struct{}

func NewDeprecatedRestrictedType(
	typ Type,
	restrictions []Type,
) *DeprecatedRestrictedType {
	return &DeprecatedRestrictedType{
		Type:         typ,
		Restrictions: restrictions,
	}
}

func NewDeprecatedMeteredRestrictedType(
	gauge common.MemoryGauge,
	typ Type,
	restrictions []Type,
) *DeprecatedRestrictedType {
	common.UseMemory(gauge, common.CadenceDeprecatedRestrictedTypeMemoryUsage)
	return NewDeprecatedRestrictedType(typ, restrictions)
}

func (*DeprecatedRestrictedType) isType() {}

func (t *DeprecatedRestrictedType) ID() string {
	if t.typeID == "" {
		var restrictionStrings []string
		restrictionCount := len(t.Restrictions)
		if restrictionCount > 0 {
			restrictionStrings = make([]string, 0, restrictionCount)
			for _, restriction := range t.Restrictions {
				restrictionStrings = append(restrictionStrings, restriction.ID())
			}
		}
		var typeString string
		if t.Type != nil {
			typeString = t.Type.ID()
		}
		t.typeID = formatDeprecatedRestrictedTypeID(typeString, restrictionStrings)
	}
	return t.typeID
}

func (t *DeprecatedRestrictedType) Equal(other Type) bool {
	otherType, ok := other.(*DeprecatedRestrictedType)
	if !ok {
		return false
	}

	if t.Type == nil && otherType.Type != nil {
		return false
	}
	if t.Type != nil && otherType.Type == nil {
		return false
	}
	if t.Type != nil && !t.Type.Equal(otherType.Type) {
		return false
	}

	restrictionSet := t.RestrictionSet()
	otherRestrictionSet := otherType.RestrictionSet()

	if len(restrictionSet) != len(otherRestrictionSet) {
		return false
	}

	for restriction := range restrictionSet { //nolint:maprange
		_, ok := otherRestrictionSet[restriction]
		if !ok {
			return false
		}
	}

	return true
}

func (t *DeprecatedRestrictedType) initializeRestrictionSet() {
	t.restrictionSetOnce.Do(func() {
		t.restrictionSet = make(DeprecatedRestrictionSet, len(t.Restrictions))
		for _, restriction := range t.Restrictions {
			t.restrictionSet[restriction] = struct{}{}
		}
	})
}

func (t *DeprecatedRestrictedType) RestrictionSet() DeprecatedRestrictionSet {
	t.initializeRestrictionSet()
	return t.restrictionSet
}

func formatDeprecatedRestrictedType(typeString string, restrictionStrings []string) string {
	var result strings.Builder
	result.WriteString(typeString)
	result.WriteByte('{')
	for i, restrictionString := range restrictionStrings {
		if i > 0 {
			result.WriteByte(',')
		}
		result.WriteString(restrictionString)
	}
	result.WriteByte('}')
	return result.String()
}

func formatDeprecatedRestrictedTypeID(typeString string, restrictionStrings []string) string {
	return formatDeprecatedRestrictedType(typeString, restrictionStrings)
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
	fields              []Field
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
		fields:              fields,
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

func (t *EnumType) compositeFields() []Field {
	return t.fields
}

func (t *EnumType) setCompositeFields(fields []Field) {
	t.fields = fields
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

func (t *EnumType) SearchFieldByName(fieldName string) Type {
	return SearchCompositeFieldTypeByName(t, fieldName)
}

func (t *EnumType) FieldsMappedByName() map[string]Type {
	return CompositeFieldTypesMappedByName(t)
}
