/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/common"
)

type Type interface {
	IsType()
	ID() string
}

// TypeID is a type which is only known by its type ID.
// This type should not be used when encoding values,
// and should only be used for decoding values that were encoded
// using an older format of the JSON encoding (<v0.3.0)
//
type TypeID string

func (TypeID) IsType() {}

func (t TypeID) ID() string {
	return string(t)
}

// AnyType

type AnyType struct{}

var _ Type = AnyType{}

func NewAnyType() AnyType {
	return AnyType{}
}

func NewMeteredAnyType(gauge common.MemoryGauge) AnyType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAnyType()
}

func (AnyType) IsType() {}

func (AnyType) ID() string {
	return "Any"
}

// AnyStructType

type AnyStructType struct{}

var _ Type = AnyStructType{}

func NewAnyStructType() AnyStructType {
	return AnyStructType{}
}

func NewMeteredAnyStructType(gauge common.MemoryGauge) AnyStructType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAnyStructType()
}

func (AnyStructType) IsType() {}

func (AnyStructType) ID() string {
	return "AnyStruct"
}

// AnyResourceType

type AnyResourceType struct{}

var _ Type = AnyResourceType{}

func NewAnyResourceType() AnyResourceType {
	return AnyResourceType{}
}

func NewMeteredAnyResourceType(gauge common.MemoryGauge) AnyResourceType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAnyResourceType()
}

func (AnyResourceType) IsType() {}

func (AnyResourceType) ID() string {
	return "AnyResource"
}

// OptionalType

type OptionalType struct {
	Type Type
}

var _ Type = OptionalType{}

func NewOptionalType(typ Type) OptionalType {
	return OptionalType{Type: typ}
}

func NewMeteredOptionalType(gauge common.MemoryGauge, typ Type) OptionalType {
	common.UseMemory(gauge, common.CadenceOptionalTypeMemoryUsage)
	return NewOptionalType(typ)
}

func (OptionalType) IsType() {}

func (t OptionalType) ID() string {
	return fmt.Sprintf("%s?", t.Type.ID())
}

// MetaType

type MetaType struct{}

var _ Type = MetaType{}

func NewMetaType() MetaType {
	return MetaType{}
}

func NewMeteredMetaType(gauge common.MemoryGauge) MetaType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewMetaType()
}

func (MetaType) IsType() {}

func (MetaType) ID() string {
	return "Type"
}

// VoidType

type VoidType struct{}

var _ Type = VoidType{}

func NewVoidType() VoidType {
	return VoidType{}
}

func NewMeteredVoidType(gauge common.MemoryGauge) VoidType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewVoidType()
}

func (VoidType) IsType() {}

func (VoidType) ID() string {
	return "Void"
}

// NeverType

type NeverType struct{}

var _ Type = NeverType{}

func NewNeverType() NeverType {
	return NeverType{}
}

func NewMeteredNeverType(gauge common.MemoryGauge) NeverType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewNeverType()
}

func (NeverType) IsType() {}

func (NeverType) ID() string {
	return "Never"
}

// BoolType

type BoolType struct{}

var _ Type = BoolType{}

func NewBoolType() BoolType {
	return BoolType{}
}

func NewMeteredBoolType(gauge common.MemoryGauge) BoolType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewBoolType()
}

func (BoolType) IsType() {}

func (BoolType) ID() string {
	return "Bool"
}

// StringType

type StringType struct{}

var _ Type = StringType{}

func NewStringType() StringType {
	return StringType{}
}

func NewMeteredStringType(gauge common.MemoryGauge) StringType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewStringType()
}

func (StringType) IsType() {}

func (StringType) ID() string {
	return "String"
}

// CharacterType

type CharacterType struct{}

var _ Type = CharacterType{}

func NewCharacterType() CharacterType {
	return CharacterType{}
}

func NewMeteredCharacterType(gauge common.MemoryGauge) CharacterType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewCharacterType()
}

func (CharacterType) IsType() {}

func (CharacterType) ID() string {
	return "Character"
}

// BytesType

type BytesType struct{}

var _ Type = BytesType{}

func NewBytesType() BytesType {
	return BytesType{}
}

func NewMeteredBytesType(gauge common.MemoryGauge) BytesType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewBytesType()
}

func (BytesType) IsType() {}

func (BytesType) ID() string {
	return "Bytes"
}

// AddressType

type AddressType struct{}

var _ Type = AddressType{}

func NewAddressType() AddressType {
	return AddressType{}
}

func NewMeteredAddressType(gauge common.MemoryGauge) AddressType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAddressType()
}

func (AddressType) IsType() {}

func (AddressType) ID() string {
	return "Address"
}

// NumberType

type NumberType struct{}

var _ Type = NumberType{}

func NewNumberType() NumberType {
	return NumberType{}
}

func NewMeteredNumberType(gauge common.MemoryGauge) NumberType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewNumberType()
}

func (NumberType) IsType() {}

func (NumberType) ID() string {
	return "Number"
}

// SignedNumberType

type SignedNumberType struct{}

var _ Type = SignedNumberType{}

func NewSignedNumberType() SignedNumberType {
	return SignedNumberType{}
}

func NewMeteredSignedNumberType(gauge common.MemoryGauge) SignedNumberType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewSignedNumberType()
}

func (SignedNumberType) IsType() {}

func (SignedNumberType) ID() string {
	return "SignedNumber"
}

// IntegerType

type IntegerType struct{}

var _ Type = IntegerType{}

func NewIntegerType() IntegerType {
	return IntegerType{}
}

func NewMeteredIntegerType(gauge common.MemoryGauge) IntegerType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewIntegerType()
}

func (IntegerType) IsType() {}

func (IntegerType) ID() string {
	return "Integer"
}

// SignedIntegerType

type SignedIntegerType struct{}

var _ Type = SignedIntegerType{}

func NewSignedIntegerType() SignedIntegerType {
	return SignedIntegerType{}
}

func NewMeteredSignedIntegerType(gauge common.MemoryGauge) SignedIntegerType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewSignedIntegerType()
}

func (SignedIntegerType) IsType() {}

func (SignedIntegerType) ID() string {
	return "SignedInteger"
}

// FixedPointType

type FixedPointType struct{}

var _ Type = FixedPointType{}

func NewFixedPointType() FixedPointType {
	return FixedPointType{}
}

func NewMeteredFixedPointType(gauge common.MemoryGauge) FixedPointType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewFixedPointType()
}

func (FixedPointType) IsType() {}

func (FixedPointType) ID() string {
	return "FixedPoint"
}

// SignedFixedPointType

type SignedFixedPointType struct{}

var _ Type = SignedFixedPointType{}

func NewSignedFixedPointType() SignedFixedPointType {
	return SignedFixedPointType{}
}

func NewMeteredSignedFixedPointType(gauge common.MemoryGauge) SignedFixedPointType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewSignedFixedPointType()
}

func (SignedFixedPointType) IsType() {}

func (SignedFixedPointType) ID() string {
	return "SignedFixedPoint"
}

// IntType

type IntType struct{}

var _ Type = IntType{}

func NewIntType() IntType {
	return IntType{}
}

func NewMeteredIntType(gauge common.MemoryGauge) IntType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewIntType()
}

func (IntType) IsType() {}

func (IntType) ID() string {
	return "Int"
}

// Int8Type

type Int8Type struct{}

var _ Type = Int8Type{}

func NewInt8Type() Int8Type {
	return Int8Type{}
}

func NewMeteredInt8Type(gauge common.MemoryGauge) Int8Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt8Type()
}

func (Int8Type) IsType() {}

func (Int8Type) ID() string {
	return "Int8"
}

// Int16Type

type Int16Type struct{}

var _ Type = Int16Type{}

func NewInt16Type() Int16Type {
	return Int16Type{}
}

func NewMeteredInt16Type(gauge common.MemoryGauge) Int16Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt16Type()
}

func (Int16Type) IsType() {}

func (Int16Type) ID() string {
	return "Int16"
}

// Int32Type

type Int32Type struct{}

var _ Type = Int32Type{}

func NewInt32Type() Int32Type {
	return Int32Type{}
}

func NewMeteredInt32Type(gauge common.MemoryGauge) Int32Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt32Type()
}

func (Int32Type) IsType() {}

func (Int32Type) ID() string {
	return "Int32"
}

// Int64Type

type Int64Type struct{}

var _ Type = Int64Type{}

func NewInt64Type() Int64Type {
	return Int64Type{}
}

func NewMeteredInt64Type(gauge common.MemoryGauge) Int64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt64Type()
}

func (Int64Type) IsType() {}

func (Int64Type) ID() string {
	return "Int64"
}

// Int128Type

type Int128Type struct{}

var _ Type = Int128Type{}

func NewInt128Type() Int128Type {
	return Int128Type{}
}

func NewMeteredInt128Type(gauge common.MemoryGauge) Int128Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt128Type()
}

func (Int128Type) IsType() {}

func (Int128Type) ID() string {
	return "Int128"
}

// Int256Type

type Int256Type struct{}

var _ Type = Int256Type{}

func NewInt256Type() Int256Type {
	return Int256Type{}
}

func NewMeteredInt256Type(gauge common.MemoryGauge) Int256Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt256Type()
}

func (Int256Type) IsType() {}

func (Int256Type) ID() string {
	return "Int256"
}

// UIntType

type UIntType struct{}

var _ Type = UIntType{}

func NewUIntType() UIntType {
	return UIntType{}
}

func NewMeteredUIntType(gauge common.MemoryGauge) UIntType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUIntType()
}

func (UIntType) IsType() {}

func (UIntType) ID() string {
	return "UInt"
}

// UInt8Type

type UInt8Type struct{}

var _ Type = UInt8Type{}

func NewUInt8Type() UInt8Type {
	return UInt8Type{}
}

func NewMeteredUInt8Type(gauge common.MemoryGauge) UInt8Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt8Type()
}

func (UInt8Type) IsType() {}

func (UInt8Type) ID() string {
	return "UInt8"
}

// UInt16Type

type UInt16Type struct{}

var _ Type = UInt16Type{}

func NewUInt16Type() UInt16Type {
	return UInt16Type{}
}

func NewMeteredUInt16Type(gauge common.MemoryGauge) UInt16Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt16Type()
}

func (UInt16Type) IsType() {}

func (UInt16Type) ID() string {
	return "UInt16"
}

// UInt32Type

type UInt32Type struct{}

var _ Type = UInt32Type{}

func NewUInt32Type() UInt32Type {
	return UInt32Type{}
}

func NewMeteredUInt32Type(gauge common.MemoryGauge) UInt32Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt32Type()
}

func (UInt32Type) IsType() {}

func (UInt32Type) ID() string {
	return "UInt32"
}

// UInt64Type

type UInt64Type struct{}

var _ Type = UInt64Type{}

func NewUInt64Type() UInt64Type {
	return UInt64Type{}
}

func NewMeteredUInt64Type(gauge common.MemoryGauge) UInt64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt64Type()
}

func (UInt64Type) IsType() {}

func (UInt64Type) ID() string {
	return "UInt64"
}

// UInt128Type

type UInt128Type struct{}

var _ Type = UInt128Type{}

func NewUInt128Type() UInt128Type {
	return UInt128Type{}
}

func NewMeteredUInt128Type(gauge common.MemoryGauge) UInt128Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt128Type()
}

func (UInt128Type) IsType() {}

func (UInt128Type) ID() string {
	return "UInt128"
}

// UInt256Type

type UInt256Type struct{}

var _ Type = UInt256Type{}

func NewUInt256Type() UInt256Type {
	return UInt256Type{}
}

func NewMeteredUInt256Type(gauge common.MemoryGauge) UInt256Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt256Type()
}

func (UInt256Type) IsType() {}

func (UInt256Type) ID() string {
	return "UInt256"
}

// Word8Type

type Word8Type struct{}

var _ Type = Word8Type{}

func NewWord8Type() Word8Type {
	return Word8Type{}
}

func NewMeteredWord8Type(gauge common.MemoryGauge) Word8Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewWord8Type()
}

func (Word8Type) IsType() {}

func (Word8Type) ID() string {
	return "Word8"
}

// Word16Type

type Word16Type struct{}

var _ Type = Word16Type{}

func NewWord16Type() Word16Type {
	return Word16Type{}
}

func NewMeteredWord16Type(gauge common.MemoryGauge) Word16Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewWord16Type()
}

func (Word16Type) IsType() {}

func (Word16Type) ID() string {
	return "Word16"
}

// Word32Type

type Word32Type struct{}

var _ Type = Word32Type{}

func NewWord32Type() Word32Type {
	return Word32Type{}
}

func NewMeteredWord32Type(gauge common.MemoryGauge) Word32Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewWord32Type()
}

func (Word32Type) IsType() {}

func (Word32Type) ID() string {
	return "Word32"
}

// Word64Type

type Word64Type struct{}

var _ Type = Word64Type{}

func NewWord64Type() Word64Type {
	return Word64Type{}
}

func NewMeteredWord64Type(gauge common.MemoryGauge) Word64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewWord64Type()
}

func (Word64Type) IsType() {}

func (Word64Type) ID() string {
	return "Word64"
}

// Fix64Type

type Fix64Type struct{}

var _ Type = Fix64Type{}

func NewFix64Type() Fix64Type {
	return Fix64Type{}
}

func NewMeteredFix64Type(gauge common.MemoryGauge) Fix64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewFix64Type()
}

func (Fix64Type) IsType() {}

func (Fix64Type) ID() string {
	return "Fix64"
}

// UFix64Type

type UFix64Type struct{}

var _ Type = UFix64Type{}

func NewUFix64Type() UFix64Type {
	return UFix64Type{}
}

func NewMeteredUFix64Type(gauge common.MemoryGauge) UFix64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUFix64Type()
}

func (UFix64Type) IsType() {}

func (UFix64Type) ID() string {
	return "UFix64"
}

type ArrayType interface {
	Type
	Element() Type
}

// VariableSizedArrayType

type VariableSizedArrayType struct {
	ElementType Type
}

var _ ArrayType = VariableSizedArrayType{}

func NewVariableSizedArrayType(
	elementType Type,
) VariableSizedArrayType {
	return VariableSizedArrayType{ElementType: elementType}
}

func NewMeteredVariableSizedArrayType(
	gauge common.MemoryGauge,
	elementType Type,
) VariableSizedArrayType {
	common.UseMemory(gauge, common.CadenceVariableSizedArrayTypeMemoryUsage)
	return NewVariableSizedArrayType(elementType)
}

func (VariableSizedArrayType) IsType() {}

func (t VariableSizedArrayType) ID() string {
	return fmt.Sprintf("[%s]", t.ElementType.ID())
}

func (t VariableSizedArrayType) Element() Type {
	return t.ElementType
}

// ConstantSizedArrayType

type ConstantSizedArrayType struct {
	Size        uint
	ElementType Type
}

var _ ArrayType = ConstantSizedArrayType{}

func NewConstantSizedArrayType(
	size uint,
	elementType Type,
) ConstantSizedArrayType {
	return ConstantSizedArrayType{
		Size:        size,
		ElementType: elementType,
	}
}

func NewMeteredConstantSizedArrayType(
	gauge common.MemoryGauge,
	size uint,
	elementType Type,
) ConstantSizedArrayType {
	common.UseMemory(gauge, common.CadenceConstantSizedArrayTypeMemoryUsage)
	return NewConstantSizedArrayType(size, elementType)
}

func (ConstantSizedArrayType) IsType() {}

func (t ConstantSizedArrayType) ID() string {
	return fmt.Sprintf("[%s;%d]", t.ElementType.ID(), t.Size)
}

func (t ConstantSizedArrayType) Element() Type {
	return t.ElementType
}

// DictionaryType

type DictionaryType struct {
	KeyType     Type
	ElementType Type
}

var _ Type = DictionaryType{}

func NewDictionaryType(
	keyType Type,
	elementType Type,
) DictionaryType {
	return DictionaryType{
		KeyType:     keyType,
		ElementType: elementType,
	}
}

func NewMeteredDictionaryType(
	gauge common.MemoryGauge,
	keyType Type,
	elementType Type,
) DictionaryType {
	common.UseMemory(gauge, common.CadenceDictionaryTypeMemoryUsage)
	return NewDictionaryType(keyType, elementType)
}

func (DictionaryType) IsType() {}

func (t DictionaryType) ID() string {
	return fmt.Sprintf(
		"{%s:%s}",
		t.KeyType.ID(),
		t.ElementType.ID(),
	)
}

// Field

type Field struct {
	Identifier string
	Type       Type
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
	Label      string
	Identifier string
	Type       Type
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

var _ CompositeType = &StructType{}

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
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *StructType {
	common.UseMemory(gauge, common.CadenceStructTypeMemoryUsage)
	return NewStructType(location, qualifiedIdentifer, fields, initializers)
}

func (*StructType) IsType() {}

func (t *StructType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
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

// ResourceType

type ResourceType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

var _ CompositeType = &ResourceType{}

func NewResourceType(
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceType {
	return &ResourceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredResourceType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceType {
	common.UseMemory(gauge, common.CadenceResourceTypeMemoryUsage)
	return NewResourceType(location, qualifiedIdentifer, fields, initializers)
}

func (*ResourceType) IsType() {}

func (t *ResourceType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
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

// EventType

type EventType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializer         []Parameter
}

var _ CompositeType = &EventType{}

func NewEventType(
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializer []Parameter,
) *EventType {
	return &EventType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializer:         initializer,
	}
}

func NewMeteredEventType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializer []Parameter,
) *EventType {
	common.UseMemory(gauge, common.CadenceEventTypeMemoryUsage)
	return NewEventType(location, qualifiedIdentifer, fields, initializer)
}

func (*EventType) IsType() {}

func (t *EventType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
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

// ContractType

type ContractType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

var _ CompositeType = &ContractType{}

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

func (*ContractType) IsType() {}

func (t *ContractType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
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

var _ InterfaceType = &StructInterfaceType{}

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

func (*StructInterfaceType) IsType() {}

func (t *StructInterfaceType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
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

// ResourceInterfaceType

type ResourceInterfaceType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

var _ InterfaceType = &ResourceInterfaceType{}

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

func (*ResourceInterfaceType) IsType() {}

func (t *ResourceInterfaceType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
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

// ContractInterfaceType

type ContractInterfaceType struct {
	Location            common.Location
	QualifiedIdentifier string
	Fields              []Field
	Initializers        [][]Parameter
}

var _ InterfaceType = &ContractInterfaceType{}

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

func (*ContractInterfaceType) IsType() {}

func (t *ContractInterfaceType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
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

// Function

type FunctionType struct {
	typeID     string
	Parameters []Parameter
	ReturnType Type
}

var _ Type = &FunctionType{}

func NewFunctionType(
	typeID string,
	parameters []Parameter,
	returnType Type,
) *FunctionType {
	return &FunctionType{
		typeID:     typeID,
		Parameters: parameters,
		ReturnType: returnType,
	}
}

func NewMeteredFunctionType(
	gauge common.MemoryGauge,
	typeID string,
	parameters []Parameter,
	returnType Type,
) *FunctionType {
	common.UseMemory(gauge, common.CadenceFunctionTypeMemoryUsage)
	return NewFunctionType(typeID, parameters, returnType)
}

func (*FunctionType) IsType() {}

func (t *FunctionType) ID() string {
	return t.typeID
}

func (t *FunctionType) WithID(id string) *FunctionType {
	t.typeID = id
	return t
}

// ReferenceType

type ReferenceType struct {
	Authorized bool
	Type       Type
}

var _ Type = ReferenceType{}

func NewReferenceType(
	authorized bool,
	typ Type,
) ReferenceType {
	return ReferenceType{
		Authorized: authorized,
		Type:       typ,
	}
}

func NewMeteredReferenceType(
	gauge common.MemoryGauge,
	authorized bool,
	typ Type,
) ReferenceType {
	common.UseMemory(gauge, common.CadenceReferenceTypeMemoryUsage)
	return NewReferenceType(authorized, typ)
}

func (ReferenceType) IsType() {}

func (t ReferenceType) ID() string {
	id := fmt.Sprintf("&%s", t.Type.ID())
	if t.Authorized {
		id = "auth" + id
	}
	return id
}

// RestrictedType

type RestrictedType struct {
	typeID       string
	Type         Type
	Restrictions []Type
}

var _ Type = &RestrictedType{}

func NewRestrictedType(
	typeID string,
	typ Type,
	restrictions []Type,
) *RestrictedType {
	return &RestrictedType{
		typeID:       typeID,
		Type:         typ,
		Restrictions: restrictions,
	}
}

func NewMeteredRestrictedType(
	gauge common.MemoryGauge,
	typeID string,
	typ Type,
	restrictions []Type,
) *RestrictedType {
	common.UseMemory(gauge, common.CadenceRestrictedTypeMemoryUsage)
	return NewRestrictedType(typeID, typ, restrictions)
}

func (*RestrictedType) IsType() {}

func (t *RestrictedType) ID() string {
	return t.typeID
}

func (t *RestrictedType) WithID(id string) *RestrictedType {
	t.typeID = id
	return t
}

// BlockType

type BlockType struct{}

var _ Type = BlockType{}

func NewBlockType() BlockType {
	return BlockType{}
}

func NewMeteredBlockType(
	gauge common.MemoryGauge,
) BlockType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewBlockType()
}

func (BlockType) IsType() {}

func (BlockType) ID() string {
	return "Block"
}

// PathType

type PathType struct{}

var _ Type = PathType{}

func NewPathType() PathType {
	return PathType{}
}

func NewMeteredPathType(
	gauge common.MemoryGauge,
) PathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPathType()
}

func (PathType) IsType() {}

func (PathType) ID() string {
	return "Path"
}

// CapabilityPathType

type CapabilityPathType struct{}

var _ Type = CapabilityPathType{}

func NewCapabilityPathType() CapabilityPathType {
	return CapabilityPathType{}
}

func NewMeteredCapabilityPathType(
	gauge common.MemoryGauge,
) CapabilityPathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewCapabilityPathType()
}

func (CapabilityPathType) IsType() {}

func (CapabilityPathType) ID() string {
	return "CapabilityPath"
}

// StoragePathType

type StoragePathType struct{}

var _ Type = StoragePathType{}

func NewStoragePathType() StoragePathType {
	return StoragePathType{}
}

func NewMeteredStoragePathType(
	gauge common.MemoryGauge,
) StoragePathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewStoragePathType()
}

func (StoragePathType) IsType() {}

func (StoragePathType) ID() string {
	return "StoragePath"
}

// PublicPathType

type PublicPathType struct{}

var _ Type = PublicPathType{}

func NewPublicPathType() PublicPathType {
	return PublicPathType{}
}

func NewMeteredPublicPathType(
	gauge common.MemoryGauge,
) PublicPathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPublicPathType()
}

func (PublicPathType) IsType() {}

func (PublicPathType) ID() string {
	return "PublicPath"
}

// PrivatePathType

type PrivatePathType struct{}

var _ Type = PrivatePathType{}

func NewPrivatePathType() PrivatePathType {
	return PrivatePathType{}
}

func NewMeteredPrivatePathType(
	gauge common.MemoryGauge,
) PrivatePathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPrivatePathType()
}

func (PrivatePathType) IsType() {}

func (PrivatePathType) ID() string {
	return "PrivatePath"
}

// CapabilityType

type CapabilityType struct {
	BorrowType Type
}

var _ Type = CapabilityType{}

func NewCapabilityType(borrowType Type) CapabilityType {
	return CapabilityType{BorrowType: borrowType}
}

func NewMeteredCapabilityType(
	gauge common.MemoryGauge,
	borrowType Type,
) CapabilityType {
	common.UseMemory(gauge, common.CadenceCapabilityTypeMemoryUsage)
	return NewCapabilityType(borrowType)
}

func (CapabilityType) IsType() {}

func (t CapabilityType) ID() string {
	if t.BorrowType != nil {
		return fmt.Sprintf("Capability<%s>", t.BorrowType.ID())
	}
	return "Capability"
}

// EnumType
type EnumType struct {
	Location            common.Location
	QualifiedIdentifier string
	RawType             Type
	Fields              []Field
	Initializers        [][]Parameter
}

var _ Type = &EnumType{}

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

func (*EnumType) IsType() {}

func (t *EnumType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
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

// AuthAccountType
type AuthAccountType struct{}

var _ Type = AuthAccountType{}

func NewAuthAccountType() AuthAccountType {
	return AuthAccountType{}
}

func NewMeteredAuthAccountType(
	gauge common.MemoryGauge,
) AuthAccountType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAuthAccountType()
}

func (AuthAccountType) IsType() {}

func (AuthAccountType) ID() string {
	return "AuthAccount"
}

// PublicAccountType
type PublicAccountType struct{}

func NewPublicAccountType() PublicAccountType {
	return PublicAccountType{}
}

func NewMeteredPublicAccountType(
	gauge common.MemoryGauge,
) PublicAccountType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPublicAccountType()
}

func (PublicAccountType) IsType() {}

func (PublicAccountType) ID() string {
	return "PublicAccount"
}

// DeployedContractType
type DeployedContractType struct{}

var _ Type = DeployedContractType{}

func NewDeployedContractType() DeployedContractType {
	return DeployedContractType{}
}

func NewMeteredDeployedContractType(
	gauge common.MemoryGauge,
) DeployedContractType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewDeployedContractType()
}

func (DeployedContractType) IsType() {}

func (DeployedContractType) ID() string {
	return "DeployedContract"
}

// AuthAccountContractsType
type AuthAccountContractsType struct{}

var _ Type = AuthAccountContractsType{}

func NewAuthAccountContractsType() AuthAccountContractsType {
	return AuthAccountContractsType{}
}

func NewMeteredAuthAccountContractsType(
	gauge common.MemoryGauge,
) AuthAccountContractsType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAuthAccountContractsType()
}

func (AuthAccountContractsType) IsType() {}

func (AuthAccountContractsType) ID() string {
	return "AuthAccount.Contracts"
}

// PublicAccountContractsType
type PublicAccountContractsType struct{}

var _ Type = PublicAccountContractsType{}

func NewPublicAccountContractsType() PublicAccountContractsType {
	return PublicAccountContractsType{}
}

func NewMeteredPublicAccountContractsType(
	gauge common.MemoryGauge,
) PublicAccountContractsType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPublicAccountContractsType()
}

func (PublicAccountContractsType) IsType() {}

func (PublicAccountContractsType) ID() string {
	return "PublicAccount.Contracts"
}

// AuthAccountKeysType
type AuthAccountKeysType struct{}

var _ Type = AuthAccountKeysType{}

func NewAuthAccountKeysType() AuthAccountKeysType {
	return AuthAccountKeysType{}
}

func NewMeteredAuthAccountKeysType(
	gauge common.MemoryGauge,
) AuthAccountKeysType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAuthAccountKeysType()
}

func (AuthAccountKeysType) IsType() {}

func (AuthAccountKeysType) ID() string {
	return "AuthAccount.Keys"
}

// PublicAccountContractsType
type PublicAccountKeysType struct{}

var _ Type = PublicAccountKeysType{}

func NewPublicAccountKeysType() PublicAccountKeysType {
	return PublicAccountKeysType{}
}

func NewMeteredPublicAccountKeysType(
	gauge common.MemoryGauge,
) PublicAccountKeysType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPublicAccountKeysType()
}

func (PublicAccountKeysType) IsType() {}

func (PublicAccountKeysType) ID() string {
	return "PublicAccount.Keys"
}

// AccountKeyType
type AccountKeyType struct{}

var _ Type = AccountKeyType{}

func NewAccountKeyType() AccountKeyType {
	return AccountKeyType{}
}

func NewMeteredAccountKeyType(
	gauge common.MemoryGauge,
) AccountKeyType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAccountKeyType()
}

func (AccountKeyType) IsType() {}

func (AccountKeyType) ID() string {
	return "AccountKey"
}
