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
	"sync"

	"github.com/onflow/cadence/runtime/common"
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
type TypeID string

func (TypeID) isType() {}

func (t TypeID) ID() string {
	return string(t)
}

func (t TypeID) Equal(other Type) bool {
	return t == other
}

// AnyType

type AnyType struct{}

var TheAnyType = AnyType{}

func NewAnyType() AnyType {
	return TheAnyType
}

func (AnyType) isType() {}

func (AnyType) ID() string {
	return "Any"
}

func (t AnyType) Equal(other Type) bool {
	return t == other
}

// AnyStructType

type AnyStructType struct{}

var TheAnyStructType = AnyStructType{}

func NewAnyStructType() AnyStructType {
	return TheAnyStructType
}

func (AnyStructType) isType() {}

func (AnyStructType) ID() string {
	return "AnyStruct"
}

func (t AnyStructType) Equal(other Type) bool {
	return t == other
}

// AnyStructAttachmentType

type AnyStructAttachmentType struct{}

var TheAnyStructAttachmentType = AnyStructAttachmentType{}

func NewAnyStructAttachmentType() AnyStructAttachmentType {
	return TheAnyStructAttachmentType
}

func (AnyStructAttachmentType) isType() {}

func (AnyStructAttachmentType) ID() string {
	return "AnyStructAttachment"
}

func (t AnyStructAttachmentType) Equal(other Type) bool {
	return t == other
}

// AnyResourceType

type AnyResourceType struct{}

var TheAnyResourceType = AnyResourceType{}

func NewAnyResourceType() AnyResourceType {
	return TheAnyResourceType
}

func (AnyResourceType) isType() {}

func (AnyResourceType) ID() string {
	return "AnyResource"
}

func (t AnyResourceType) Equal(other Type) bool {
	return t == other
}

// AnyResourceAttachmentType

type AnyResourceAttachmentType struct{}

var TheAnyResourceAttachmentType = AnyResourceAttachmentType{}

func NewAnyResourceAttachmentType() AnyResourceAttachmentType {
	return TheAnyResourceAttachmentType
}

func (AnyResourceAttachmentType) isType() {}

func (AnyResourceAttachmentType) ID() string {
	return "AnyResourceAttachment"
}

func (t AnyResourceAttachmentType) Equal(other Type) bool {
	return t == other
}

// OptionalType

type OptionalType struct {
	Type   Type
	typeID string
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
	if len(t.typeID) == 0 {
		t.typeID = fmt.Sprintf("%s?", t.Type.ID())
	}
	return t.typeID
}

func (t *OptionalType) Equal(other Type) bool {
	otherOptional, ok := other.(*OptionalType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherOptional.Type)
}

// MetaType

type MetaType struct{}

var TheMetaType = MetaType{}

func NewMetaType() MetaType {
	return TheMetaType
}

func (MetaType) isType() {}

func (MetaType) ID() string {
	return "Type"
}

func (t MetaType) Equal(other Type) bool {
	return t == other
}

// VoidType

type VoidType struct{}

var TheVoidType = VoidType{}

func NewVoidType() VoidType {
	return TheVoidType
}

func (VoidType) isType() {}

func (VoidType) ID() string {
	return "Void"
}

func (t VoidType) Equal(other Type) bool {
	return t == other
}

// NeverType

type NeverType struct{}

var TheNeverType = NeverType{}

func NewNeverType() NeverType {
	return TheNeverType
}

func (NeverType) isType() {}

func (NeverType) ID() string {
	return "Never"
}

func (t NeverType) Equal(other Type) bool {
	return t == other
}

// BoolType

type BoolType struct{}

var TheBoolType = BoolType{}

func NewBoolType() BoolType {
	return TheBoolType
}

func (BoolType) isType() {}

func (BoolType) ID() string {
	return "Bool"
}

func (t BoolType) Equal(other Type) bool {
	return t == other
}

// StringType

type StringType struct{}

var TheStringType = StringType{}

func NewStringType() StringType {
	return TheStringType
}

func (StringType) isType() {}

func (StringType) ID() string {
	return "String"
}

func (t StringType) Equal(other Type) bool {
	return t == other
}

// CharacterType

type CharacterType struct{}

var TheCharacterType = CharacterType{}

func NewCharacterType() CharacterType {
	return TheCharacterType
}

func (CharacterType) isType() {}

func (CharacterType) ID() string {
	return "Character"
}

func (t CharacterType) Equal(other Type) bool {
	return t == other
}

// BytesType

type BytesType struct{}

var TheBytesType = BytesType{}

func NewBytesType() BytesType {
	return TheBytesType
}

func (BytesType) isType() {}

func (BytesType) ID() string {
	return "Bytes"
}

func (t BytesType) Equal(other Type) bool {
	return t == other
}

// AddressType

type AddressType struct{}

var TheAddressType = AddressType{}

func NewAddressType() AddressType {
	return TheAddressType
}

func (AddressType) isType() {}

func (AddressType) ID() string {
	return "Address"
}

func (t AddressType) Equal(other Type) bool {
	return t == other
}

// NumberType

type NumberType struct{}

var TheNumberType = NumberType{}

func NewNumberType() NumberType {
	return TheNumberType
}

func (NumberType) isType() {}

func (NumberType) ID() string {
	return "Number"
}

func (t NumberType) Equal(other Type) bool {
	return t == other
}

// SignedNumberType

type SignedNumberType struct{}

var TheSignedNumberType = SignedNumberType{}

func NewSignedNumberType() SignedNumberType {
	return TheSignedNumberType
}

func (SignedNumberType) isType() {}

func (SignedNumberType) ID() string {
	return "SignedNumber"
}

func (t SignedNumberType) Equal(other Type) bool {
	return t == other
}

// IntegerType

type IntegerType struct{}

var TheIntegerType = IntegerType{}

func NewIntegerType() IntegerType {
	return TheIntegerType
}

func (IntegerType) isType() {}

func (IntegerType) ID() string {
	return "Integer"
}

func (t IntegerType) Equal(other Type) bool {
	return t == other
}

// SignedIntegerType

type SignedIntegerType struct{}

var TheSignedIntegerType = SignedIntegerType{}

func NewSignedIntegerType() SignedIntegerType {
	return TheSignedIntegerType
}

func (SignedIntegerType) isType() {}

func (SignedIntegerType) ID() string {
	return "SignedInteger"
}

func (t SignedIntegerType) Equal(other Type) bool {
	return t == other
}

// FixedPointType

type FixedPointType struct{}

var TheFixedPointType = FixedPointType{}

func NewFixedPointType() FixedPointType {
	return TheFixedPointType
}

func (FixedPointType) isType() {}

func (FixedPointType) ID() string {
	return "FixedPoint"
}

func (t FixedPointType) Equal(other Type) bool {
	return t == other
}

// SignedFixedPointType

type SignedFixedPointType struct{}

var TheSignedFixedPointType = SignedFixedPointType{}

func NewSignedFixedPointType() SignedFixedPointType {
	return TheSignedFixedPointType
}

func (SignedFixedPointType) isType() {}

func (SignedFixedPointType) ID() string {
	return "SignedFixedPoint"
}

func (t SignedFixedPointType) Equal(other Type) bool {
	return t == other
}

// IntType

type IntType struct{}

var TheIntType = IntType{}

func NewIntType() IntType {
	return TheIntType
}

func (IntType) isType() {}

func (IntType) ID() string {
	return "Int"
}

func (t IntType) Equal(other Type) bool {
	return t == other
}

// Int8Type

type Int8Type struct{}

var TheInt8Type = Int8Type{}

func NewInt8Type() Int8Type {
	return TheInt8Type
}

func (t Int8Type) Equal(other Type) bool {
	return t == other
}

func (Int8Type) isType() {}

func (Int8Type) ID() string {
	return "Int8"
}

// Int16Type

type Int16Type struct{}

var TheInt16Type = Int16Type{}

func NewInt16Type() Int16Type {
	return TheInt16Type
}

func (Int16Type) isType() {}

func (Int16Type) ID() string {
	return "Int16"
}

func (t Int16Type) Equal(other Type) bool {
	return t == other
}

// Int32Type

type Int32Type struct{}

var TheInt32Type = Int32Type{}

func NewInt32Type() Int32Type {
	return TheInt32Type
}

func (Int32Type) isType() {}

func (Int32Type) ID() string {
	return "Int32"
}

func (t Int32Type) Equal(other Type) bool {
	return t == other
}

// Int64Type

type Int64Type struct{}

var TheInt64Type = Int64Type{}

func NewInt64Type() Int64Type {
	return TheInt64Type
}

func (Int64Type) isType() {}

func (Int64Type) ID() string {
	return "Int64"
}

func (t Int64Type) Equal(other Type) bool {
	return t == other
}

// Int128Type

type Int128Type struct{}

var TheInt128Type = Int128Type{}

func NewInt128Type() Int128Type {
	return TheInt128Type
}

func (Int128Type) isType() {}

func (Int128Type) ID() string {
	return "Int128"
}

func (t Int128Type) Equal(other Type) bool {
	return t == other
}

// Int256Type

type Int256Type struct{}

var TheInt256Type = Int256Type{}

func NewInt256Type() Int256Type {
	return TheInt256Type
}

func (Int256Type) isType() {}

func (Int256Type) ID() string {
	return "Int256"
}

func (t Int256Type) Equal(other Type) bool {
	return t == other
}

// UIntType

type UIntType struct{}

var TheUIntType = UIntType{}

func NewUIntType() UIntType {
	return TheUIntType
}

func (UIntType) isType() {}

func (UIntType) ID() string {
	return "UInt"
}

func (t UIntType) Equal(other Type) bool {
	return t == other
}

// UInt8Type

type UInt8Type struct{}

var TheUInt8Type = UInt8Type{}

func NewUInt8Type() UInt8Type {
	return TheUInt8Type
}

func (UInt8Type) isType() {}

func (UInt8Type) ID() string {
	return "UInt8"
}

func (t UInt8Type) Equal(other Type) bool {
	return t == other
}

// UInt16Type

type UInt16Type struct{}

var TheUInt16Type = UInt16Type{}

func NewUInt16Type() UInt16Type {
	return TheUInt16Type
}

func (UInt16Type) isType() {}

func (UInt16Type) ID() string {
	return "UInt16"
}

func (t UInt16Type) Equal(other Type) bool {
	return t == other
}

// UInt32Type

type UInt32Type struct{}

var TheUInt32Type = UInt32Type{}

func NewUInt32Type() UInt32Type {
	return TheUInt32Type
}

func (UInt32Type) isType() {}

func (UInt32Type) ID() string {
	return "UInt32"
}

func (t UInt32Type) Equal(other Type) bool {
	return t == other
}

// UInt64Type

type UInt64Type struct{}

var TheUInt64Type = UInt64Type{}

func NewUInt64Type() UInt64Type {
	return TheUInt64Type
}

func (UInt64Type) isType() {}

func (UInt64Type) ID() string {
	return "UInt64"
}

func (t UInt64Type) Equal(other Type) bool {
	return t == other
}

// UInt128Type

type UInt128Type struct{}

var TheUInt128Type = UInt128Type{}

func NewUInt128Type() UInt128Type {
	return TheUInt128Type
}

func (UInt128Type) isType() {}

func (UInt128Type) ID() string {
	return "UInt128"
}

func (t UInt128Type) Equal(other Type) bool {
	return t == other
}

// UInt256Type

type UInt256Type struct{}

var TheUInt256Type = UInt256Type{}

func NewUInt256Type() UInt256Type {
	return TheUInt256Type
}

func (UInt256Type) isType() {}

func (UInt256Type) ID() string {
	return "UInt256"
}

func (t UInt256Type) Equal(other Type) bool {
	return t == other
}

// Word8Type

type Word8Type struct{}

var TheWord8Type = Word8Type{}

func NewWord8Type() Word8Type {
	return TheWord8Type
}

func (Word8Type) isType() {}

func (Word8Type) ID() string {
	return "Word8"
}

func (t Word8Type) Equal(other Type) bool {
	return t == other
}

// Word16Type

type Word16Type struct{}

var TheWord16Type = Word16Type{}

func NewWord16Type() Word16Type {
	return TheWord16Type
}

func (Word16Type) isType() {}

func (Word16Type) ID() string {
	return "Word16"
}

func (t Word16Type) Equal(other Type) bool {
	return t == other
}

// Word32Type

type Word32Type struct{}

var TheWord32Type = Word32Type{}

func NewWord32Type() Word32Type {
	return TheWord32Type
}

func (Word32Type) isType() {}

func (Word32Type) ID() string {
	return "Word32"
}

func (t Word32Type) Equal(other Type) bool {
	return t == other
}

// Word64Type

type Word64Type struct{}

var TheWord64Type = Word64Type{}

func NewWord64Type() Word64Type {
	return TheWord64Type
}

func (Word64Type) isType() {}

func (Word64Type) ID() string {
	return "Word64"
}

func (t Word64Type) Equal(other Type) bool {
	return t == other
}

// Fix64Type

type Fix64Type struct{}

var TheFix64Type = Fix64Type{}

func NewFix64Type() Fix64Type {
	return TheFix64Type
}

func (Fix64Type) isType() {}

func (Fix64Type) ID() string {
	return "Fix64"
}

func (t Fix64Type) Equal(other Type) bool {
	return t == other
}

// UFix64Type

type UFix64Type struct{}

var TheUFix64Type = UFix64Type{}

func NewUFix64Type() UFix64Type {
	return TheUFix64Type
}

func (UFix64Type) isType() {}

func (UFix64Type) ID() string {
	return "UFix64"
}

func (t UFix64Type) Equal(other Type) bool {
	return t == other
}

type ArrayType interface {
	Type
	Element() Type
}

// VariableSizedArrayType

type VariableSizedArrayType struct {
	ElementType Type
	typeID      string
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
	if len(t.typeID) == 0 {
		t.typeID = fmt.Sprintf("[%s]", t.ElementType.ID())
	}
	return t.typeID
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
	if len(t.typeID) == 0 {
		t.typeID = fmt.Sprintf("[%s;%d]", t.ElementType.ID(), t.Size)
	}
	return t.typeID
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
	typeID      string
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
	if len(t.typeID) == 0 {
		t.typeID = fmt.Sprintf(
			"{%s:%s}",
			t.KeyType.ID(),
			t.ElementType.ID(),
		)
	}
	return t.typeID
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
	typeID              string
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
	if len(t.typeID) == 0 {
		t.typeID = generateTypeID(t.Location, t.QualifiedIdentifier)
	}
	return t.typeID
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
	typeID              string
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
	if len(t.typeID) == 0 {
		t.typeID = generateTypeID(t.Location, t.QualifiedIdentifier)
	}
	return t.typeID
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
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *AttachmentType {
	common.UseMemory(gauge, common.CadenceStructTypeMemoryUsage)
	return NewAttachmentType(location, baseType, qualifiedIdentifer, fields, initializers)
}

func (*AttachmentType) isType() {}

func (t *AttachmentType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
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
	typeID              string
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
	if len(t.typeID) == 0 {
		t.typeID = generateTypeID(t.Location, t.QualifiedIdentifier)
	}
	return t.typeID
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
	typeID              string
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
	if len(t.typeID) == 0 {
		t.typeID = generateTypeID(t.Location, t.QualifiedIdentifier)
	}
	return t.typeID
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
	typeID              string
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
	if len(t.typeID) == 0 {
		t.typeID = generateTypeID(t.Location, t.QualifiedIdentifier)
	}
	return t.typeID
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
	typeID              string
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
	if len(t.typeID) == 0 {
		t.typeID = generateTypeID(t.Location, t.QualifiedIdentifier)
	}
	return t.typeID
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
	typeID              string
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
	if len(t.typeID) == 0 {
		t.typeID = generateTypeID(t.Location, t.QualifiedIdentifier)
	}
	return t.typeID
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
	typeID         string
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
	if t.typeID == "" {

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

		t.typeID = sema.FormatFunctionTypeID(
			purity,
			typeParameters,
			parameters,
			returnType,
		)
	}
	return t.typeID
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

// ReferenceType

type ReferenceType struct {
	Type       Type
	Authorized bool
	typeID     string
}

var _ Type = &ReferenceType{}

func NewReferenceType(
	authorized bool,
	typ Type,
) *ReferenceType {
	return &ReferenceType{
		Authorized: authorized,
		Type:       typ,
	}
}

func NewMeteredReferenceType(
	gauge common.MemoryGauge,
	authorized bool,
	typ Type,
) *ReferenceType {
	common.UseMemory(gauge, common.CadenceReferenceTypeMemoryUsage)
	return NewReferenceType(authorized, typ)
}

func (*ReferenceType) isType() {}

func (t *ReferenceType) ID() string {
	if t.typeID == "" {
		t.typeID = sema.FormatReferenceTypeID(t.Authorized, t.Type.ID())
	}
	return t.typeID
}

func (t *ReferenceType) Equal(other Type) bool {
	otherType, ok := other.(*ReferenceType)
	if !ok {
		return false
	}

	return t.Authorized == otherType.Authorized &&
		t.Type.Equal(otherType.Type)
}

// RestrictedType

type RestrictionSet = map[Type]struct{}

type RestrictedType struct {
	typeID             string
	Type               Type
	Restrictions       []Type
	restrictionSet     RestrictionSet
	restrictionSetOnce sync.Once
}

func NewRestrictedType(
	typ Type,
	restrictions []Type,
) *RestrictedType {
	return &RestrictedType{
		Type:         typ,
		Restrictions: restrictions,
	}
}

func NewMeteredRestrictedType(
	gauge common.MemoryGauge,
	typ Type,
	restrictions []Type,
) *RestrictedType {
	common.UseMemory(gauge, common.CadenceRestrictedTypeMemoryUsage)
	return NewRestrictedType(typ, restrictions)
}

func (*RestrictedType) isType() {}

func (t *RestrictedType) ID() string {
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
		t.typeID = sema.FormatRestrictedTypeID(typeString, restrictionStrings)
	}
	return t.typeID
}

func (t *RestrictedType) Equal(other Type) bool {
	otherType, ok := other.(*RestrictedType)
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

func (t *RestrictedType) initializeRestrictionSet() {
	t.restrictionSetOnce.Do(func() {
		t.restrictionSet = make(RestrictionSet, len(t.Restrictions))
		for _, restriction := range t.Restrictions {
			t.restrictionSet[restriction] = struct{}{}
		}
	})
}

func (t *RestrictedType) RestrictionSet() RestrictionSet {
	t.initializeRestrictionSet()
	return t.restrictionSet
}

// BlockType

type BlockType struct{}

var TheBlockType = BlockType{}

func NewBlockType() BlockType {
	return TheBlockType
}

func (BlockType) isType() {}

func (BlockType) ID() string {
	return "Block"
}

func (t BlockType) Equal(other Type) bool {
	return t == other
}

// PathType

type PathType struct{}

var ThePathType = PathType{}

func NewPathType() PathType {
	return ThePathType
}

func (PathType) isType() {}

func (PathType) ID() string {
	return "Path"
}

func (t PathType) Equal(other Type) bool {
	return t == other
}

// CapabilityPathType

type CapabilityPathType struct{}

var TheCapabilityPathType = CapabilityPathType{}

func NewCapabilityPathType() CapabilityPathType {
	return TheCapabilityPathType
}

func (CapabilityPathType) isType() {}

func (CapabilityPathType) ID() string {
	return "CapabilityPath"
}

func (t CapabilityPathType) Equal(other Type) bool {
	return t == other
}

// StoragePathType

type StoragePathType struct{}

var TheStoragePathType = StoragePathType{}

func NewStoragePathType() StoragePathType {
	return TheStoragePathType
}

func (StoragePathType) isType() {}

func (StoragePathType) ID() string {
	return "StoragePath"
}

func (t StoragePathType) Equal(other Type) bool {
	return t == other
}

// PublicPathType

type PublicPathType struct{}

var ThePublicPathType = PublicPathType{}

func NewPublicPathType() PublicPathType {
	return ThePublicPathType
}

func (PublicPathType) isType() {}

func (PublicPathType) ID() string {
	return "PublicPath"
}

func (t PublicPathType) Equal(other Type) bool {
	return t == other
}

// PrivatePathType

type PrivatePathType struct{}

var ThePrivatePathType = PrivatePathType{}

func NewPrivatePathType() PrivatePathType {
	return ThePrivatePathType
}

func (PrivatePathType) isType() {}

func (PrivatePathType) ID() string {
	return "PrivatePath"
}

func (t PrivatePathType) Equal(other Type) bool {
	return t == other
}

// CapabilityType

type CapabilityType struct {
	BorrowType Type
	typeID     string
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
	if t.typeID == "" {
		var borrowTypeString string
		borrowType := t.BorrowType
		if borrowType != nil {
			borrowTypeString = borrowType.ID()
		}
		t.typeID = sema.FormatCapabilityTypeID(borrowTypeString)
	}
	return t.typeID
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
	typeID              string
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
	if len(t.typeID) == 0 {
		t.typeID = generateTypeID(t.Location, t.QualifiedIdentifier)
	}
	return t.typeID
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

// AuthAccountType
type AuthAccountType struct{}

var TheAuthAccountType = AuthAccountType{}

func NewAuthAccountType() AuthAccountType {
	return TheAuthAccountType
}

func (AuthAccountType) isType() {}

func (AuthAccountType) ID() string {
	return "AuthAccount"
}

func (t AuthAccountType) Equal(other Type) bool {
	return t == other
}

// PublicAccountType
type PublicAccountType struct{}

var ThePublicAccountType = PublicAccountType{}

func NewPublicAccountType() PublicAccountType {
	return ThePublicAccountType
}

func (PublicAccountType) isType() {}

func (PublicAccountType) ID() string {
	return "PublicAccount"
}

func (t PublicAccountType) Equal(other Type) bool {
	return t == other
}

// DeployedContractType
type DeployedContractType struct{}

var TheDeployedContractType = DeployedContractType{}

func NewDeployedContractType() DeployedContractType {
	return TheDeployedContractType
}

func (DeployedContractType) isType() {}

func (DeployedContractType) ID() string {
	return "DeployedContract"
}

func (t DeployedContractType) Equal(other Type) bool {
	return t == other
}

// AuthAccountContractsType
type AuthAccountContractsType struct{}

var TheAuthAccountContractsType = AuthAccountContractsType{}

func NewAuthAccountContractsType() AuthAccountContractsType {
	return TheAuthAccountContractsType
}

func (AuthAccountContractsType) isType() {}

func (AuthAccountContractsType) ID() string {
	return "AuthAccount.Contracts"
}

func (t AuthAccountContractsType) Equal(other Type) bool {
	return t == other
}

// PublicAccountContractsType
type PublicAccountContractsType struct{}

var ThePublicAccountContractsType = PublicAccountContractsType{}

func NewPublicAccountContractsType() PublicAccountContractsType {
	return ThePublicAccountContractsType
}

func (PublicAccountContractsType) isType() {}

func (PublicAccountContractsType) ID() string {
	return "PublicAccount.Contracts"
}

func (t PublicAccountContractsType) Equal(other Type) bool {
	return t == other
}

// AuthAccountKeysType
type AuthAccountKeysType struct{}

var TheAuthAccountKeysType = AuthAccountKeysType{}

func NewAuthAccountKeysType() AuthAccountKeysType {
	return TheAuthAccountKeysType
}

func (AuthAccountKeysType) isType() {}

func (AuthAccountKeysType) ID() string {
	return "AuthAccount.Keys"
}

func (t AuthAccountKeysType) Equal(other Type) bool {
	return t == other
}

// PublicAccountKeysType
type PublicAccountKeysType struct{}

var ThePublicAccountKeysType = PublicAccountKeysType{}

func NewPublicAccountKeysType() PublicAccountKeysType {
	return ThePublicAccountKeysType
}

func (PublicAccountKeysType) isType() {}

func (PublicAccountKeysType) ID() string {
	return "PublicAccount.Keys"
}

func (t PublicAccountKeysType) Equal(other Type) bool {
	return t == other
}

// AccountKeyType
type AccountKeyType struct{}

var TheAccountKeyType = AccountKeyType{}

func NewAccountKeyType() AccountKeyType {
	return TheAccountKeyType
}

func (AccountKeyType) isType() {}

func (AccountKeyType) ID() string {
	return "AccountKey"
}

func (t AccountKeyType) Equal(other Type) bool {
	return t == other
}

func generateTypeID(location common.Location, identifier string) string {
	if location == nil {
		return identifier
	}

	return string(location.TypeID(nil, identifier))
}

// TypeWithCachedTypeID recursively caches type ID of type t.
// This is needed because each type ID is lazily cached on
// its first use in ID() to avoid performance penalty.
func TypeWithCachedTypeID(t Type) Type {
	if t == nil {
		return t
	}

	// Cache type ID by calling ID()
	t.ID()

	switch t := t.(type) {

	case CompositeType:
		fields := t.CompositeFields()
		for _, f := range fields {
			TypeWithCachedTypeID(f.Type)
		}

		initializers := t.CompositeInitializers()
		for _, params := range initializers {
			for _, p := range params {
				TypeWithCachedTypeID(p.Type)
			}
		}

	case *RestrictedType:
		for _, restriction := range t.Restrictions {
			TypeWithCachedTypeID(restriction)
		}
	}

	return t
}
