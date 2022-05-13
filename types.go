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
	isType()
	ID() string
}

// AnyType

type AnyType struct{}

func NewAnyType() AnyType {
	return AnyType{}
}

func NewMeteredAnyType(gauge common.MemoryGauge) AnyType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewAnyType()
}

func (AnyType) isType() {}

func (AnyType) ID() string {
	return "Any"
}

// AnyStructType

type AnyStructType struct{}

func NewAnyStructType() AnyStructType {
	return AnyStructType{}
}

func NewMeteredAnyStructType(gauge common.MemoryGauge) AnyStructType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewAnyStructType()
}

func (AnyStructType) isType() {}

func (AnyStructType) ID() string {
	return "AnyStruct"
}

// AnyResourceType

type AnyResourceType struct{}

func NewAnyResourceType() AnyResourceType {
	return AnyResourceType{}
}

func NewMeteredAnyResourceType(gauge common.MemoryGauge) AnyResourceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewAnyResourceType()
}

func (AnyResourceType) isType() {}

func (AnyResourceType) ID() string {
	return "AnyResource"
}

// OptionalType

type OptionalType struct {
	Type Type
}

func NewOptionalType(typ Type) OptionalType {
	return OptionalType{Type: typ}
}

func NewMeteredOptionalType(gauge common.MemoryGauge, typ Type) OptionalType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceOptionalType)
	return NewOptionalType(typ)
}

func (OptionalType) isType() {}

func (t OptionalType) ID() string {
	return fmt.Sprintf("%s?", t.Type.ID())
}

// MetaType

type MetaType struct{}

func NewMetaType() MetaType {
	return MetaType{}
}

func NewMeteredMetaType(gauge common.MemoryGauge) MetaType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewMetaType()
}

func (MetaType) isType() {}

func (MetaType) ID() string {
	return "Type"
}

// VoidType

type VoidType struct{}

func NewVoidType() VoidType {
	return VoidType{}
}

func NewMeteredVoidType(gauge common.MemoryGauge) VoidType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewVoidType()
}

func (VoidType) isType() {}

func (VoidType) ID() string {
	return "Void"
}

// NeverType

type NeverType struct{}

func NewNeverType() NeverType {
	return NeverType{}
}

func NewMeteredNeverType(gauge common.MemoryGauge) NeverType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewNeverType()
}

func (NeverType) isType() {}

func (NeverType) ID() string {
	return "Never"
}

// BoolType

type BoolType struct{}

func NewBoolType() BoolType {
	return BoolType{}
}

func NewMeteredBoolType(gauge common.MemoryGauge) BoolType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewBoolType()
}

func (BoolType) isType() {}

func (BoolType) ID() string {
	return "Bool"
}

// StringType

type StringType struct{}

func NewStringType() StringType {
	return StringType{}
}

func NewMeteredStringType(gauge common.MemoryGauge) StringType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewStringType()
}

func (StringType) isType() {}

func (StringType) ID() string {
	return "String"
}

// CharacterType

type CharacterType struct{}

func NewCharacterType() CharacterType {
	return CharacterType{}
}

func NewMeteredCharacterType(gauge common.MemoryGauge) CharacterType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewCharacterType()
}

func (CharacterType) isType() {}

func (CharacterType) ID() string {
	return "Character"
}

// BytesType

type BytesType struct{}

func NewBytesType() BytesType {
	return BytesType{}
}

func NewMeteredBytesType(gauge common.MemoryGauge) BytesType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewBytesType()
}

func (BytesType) isType() {}

func (BytesType) ID() string {
	return "Bytes"
}

// AddressType

type AddressType struct{}

func NewAddressType() AddressType {
	return AddressType{}
}

func NewMeteredAddressType(gauge common.MemoryGauge) AddressType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewAddressType()
}

func (AddressType) isType() {}

func (AddressType) ID() string {
	return "Address"
}

// NumberType

type NumberType struct{}

func NewNumberType() NumberType {
	return NumberType{}
}

func NewMeteredNumberType(gauge common.MemoryGauge) NumberType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewNumberType()
}

func (NumberType) isType() {}

func (NumberType) ID() string {
	return "Number"
}

// SignedNumberType

type SignedNumberType struct{}

func NewSignedNumberType() SignedNumberType {
	return SignedNumberType{}
}

func NewMeteredSignedNumberType(gauge common.MemoryGauge) SignedNumberType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewSignedNumberType()
}

func (SignedNumberType) isType() {}

func (SignedNumberType) ID() string {
	return "SignedNumber"
}

// IntegerType

type IntegerType struct{}

func NewIntegerType() IntegerType {
	return IntegerType{}
}

func NewMeteredIntegerType(gauge common.MemoryGauge) IntegerType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewIntegerType()
}

func (IntegerType) isType() {}

func (IntegerType) ID() string {
	return "Integer"
}

// SignedIntegerType

type SignedIntegerType struct{}

func NewSignedIntegerType() SignedIntegerType {
	return SignedIntegerType{}
}

func NewMeteredSignedIntegerType(gauge common.MemoryGauge) SignedIntegerType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewSignedIntegerType()
}

func (SignedIntegerType) isType() {}

func (SignedIntegerType) ID() string {
	return "SignedInteger"
}

// FixedPointType

type FixedPointType struct{}

func NewFixedPointType() FixedPointType {
	return FixedPointType{}
}

func NewMeteredFixedPointType(gauge common.MemoryGauge) FixedPointType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewFixedPointType()
}

func (FixedPointType) isType() {}

func (FixedPointType) ID() string {
	return "FixedPoint"
}

// SignedFixedPointType

type SignedFixedPointType struct{}

func NewSignedFixedPointType() SignedFixedPointType {
	return SignedFixedPointType{}
}

func NewMeteredSignedFixedPointType(gauge common.MemoryGauge) SignedFixedPointType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewSignedFixedPointType()
}

func (SignedFixedPointType) isType() {}

func (SignedFixedPointType) ID() string {
	return "SignedFixedPoint"
}

// IntType

type IntType struct{}

func NewIntType() IntType {
	return IntType{}
}

func NewMeteredIntType(gauge common.MemoryGauge) IntType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewIntType()
}

func (IntType) isType() {}

func (IntType) ID() string {
	return "Int"
}

// Int8Type

type Int8Type struct{}

func NewInt8Type() Int8Type {
	return Int8Type{}
}

func NewMeteredInt8Type(gauge common.MemoryGauge) Int8Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewInt8Type()
}

func (Int8Type) isType() {}

func (Int8Type) ID() string {
	return "Int8"
}

// Int16Type

type Int16Type struct{}

func NewInt16Type() Int16Type {
	return Int16Type{}
}

func NewMeteredInt16Type(gauge common.MemoryGauge) Int16Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewInt16Type()
}

func (Int16Type) isType() {}

func (Int16Type) ID() string {
	return "Int16"
}

// Int32Type

type Int32Type struct{}

func NewInt32Type() Int32Type {
	return Int32Type{}
}

func NewMeteredInt32Type(gauge common.MemoryGauge) Int32Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewInt32Type()
}

func (Int32Type) isType() {}

func (Int32Type) ID() string {
	return "Int32"
}

// Int64Type

type Int64Type struct{}

func NewInt64Type() Int64Type {
	return Int64Type{}
}

func NewMeteredInt64Type(gauge common.MemoryGauge) Int64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewInt64Type()
}

func (Int64Type) isType() {}

func (Int64Type) ID() string {
	return "Int64"
}

// Int128Type

type Int128Type struct{}

func NewInt128Type() Int128Type {
	return Int128Type{}
}

func NewMeteredInt128Type(gauge common.MemoryGauge) Int128Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewInt128Type()
}

func (Int128Type) isType() {}

func (Int128Type) ID() string {
	return "Int128"
}

// Int256Type

type Int256Type struct{}

func NewInt256Type() Int256Type {
	return Int256Type{}
}

func NewMeteredInt256Type(gauge common.MemoryGauge) Int256Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewInt256Type()
}

func (Int256Type) isType() {}

func (Int256Type) ID() string {
	return "Int256"
}

// UIntType

type UIntType struct{}

func NewUIntType() UIntType {
	return UIntType{}
}

func NewMeteredUIntType(gauge common.MemoryGauge) UIntType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewUIntType()
}

func (UIntType) isType() {}

func (UIntType) ID() string {
	return "UInt"
}

// UInt8Type

type UInt8Type struct{}

func NewUInt8Type() UInt8Type {
	return UInt8Type{}
}

func NewMeteredUInt8Type(gauge common.MemoryGauge) UInt8Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewUInt8Type()
}

func (UInt8Type) isType() {}

func (UInt8Type) ID() string {
	return "UInt8"
}

// UInt16Type

type UInt16Type struct{}

func NewUInt16Type() UInt16Type {
	return UInt16Type{}
}

func NewMeteredUInt16Type(gauge common.MemoryGauge) UInt16Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewUInt16Type()
}

func (UInt16Type) isType() {}

func (UInt16Type) ID() string {
	return "UInt16"
}

// UInt32Type

type UInt32Type struct{}

func NewUInt32Type() UInt32Type {
	return UInt32Type{}
}

func NewMeteredUInt32Type(gauge common.MemoryGauge) UInt32Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewUInt32Type()
}

func (UInt32Type) isType() {}

func (UInt32Type) ID() string {
	return "UInt32"
}

// UInt64Type

type UInt64Type struct{}

func NewUInt64Type() UInt64Type {
	return UInt64Type{}
}

func NewMeteredUInt64Type(gauge common.MemoryGauge) UInt64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewUInt64Type()
}

func (UInt64Type) isType() {}

func (UInt64Type) ID() string {
	return "UInt64"
}

// UInt128Type

type UInt128Type struct{}

func NewUInt128Type() UInt128Type {
	return UInt128Type{}
}

func NewMeteredUInt128Type(gauge common.MemoryGauge) UInt128Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewUInt128Type()
}

func (UInt128Type) isType() {}

func (UInt128Type) ID() string {
	return "UInt128"
}

// UInt256Type

type UInt256Type struct{}

func NewUInt256Type() UInt256Type {
	return UInt256Type{}
}

func NewMeteredUInt256Type(gauge common.MemoryGauge) UInt256Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewUInt256Type()
}

func (UInt256Type) isType() {}

func (UInt256Type) ID() string {
	return "UInt256"
}

// Word8Type

type Word8Type struct{}

func NewWord8Type() Word8Type {
	return Word8Type{}
}

func NewMeteredWord8Type(gauge common.MemoryGauge) Word8Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewWord8Type()
}

func (Word8Type) isType() {}

func (Word8Type) ID() string {
	return "Word8"
}

// Word16Type

type Word16Type struct{}

func NewWord16Type() Word16Type {
	return Word16Type{}
}

func NewMeteredWord16Type(gauge common.MemoryGauge) Word16Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewWord16Type()
}

func (Word16Type) isType() {}

func (Word16Type) ID() string {
	return "Word16"
}

// Word32Type

type Word32Type struct{}

func NewWord32Type() Word32Type {
	return Word32Type{}
}

func NewMeteredWord32Type(gauge common.MemoryGauge) Word32Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewWord32Type()
}

func (Word32Type) isType() {}

func (Word32Type) ID() string {
	return "Word32"
}

// Word64Type

type Word64Type struct{}

func NewWord64Type() Word64Type {
	return Word64Type{}
}

func NewMeteredWord64Type(gauge common.MemoryGauge) Word64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewWord64Type()
}

func (Word64Type) isType() {}

func (Word64Type) ID() string {
	return "Word64"
}

// Fix64Type

type Fix64Type struct{}

func NewFix64Type() Fix64Type {
	return Fix64Type{}
}

func NewMeteredFix64Type(gauge common.MemoryGauge) Fix64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewFix64Type()
}

func (Fix64Type) isType() {}

func (Fix64Type) ID() string {
	return "Fix64"
}

// UFix64Type

type UFix64Type struct{}

func NewUFix64Type() UFix64Type {
	return UFix64Type{}
}

func NewMeteredUFix64Type(gauge common.MemoryGauge) UFix64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewUFix64Type()
}

func (UFix64Type) isType() {}

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

func NewVariableSizedArrayType(
	elementType Type,
) VariableSizedArrayType {
	return VariableSizedArrayType{ElementType: elementType}
}

func NewMeteredVariableSizedArrayType(
	gauge common.MemoryGauge,
	elementType Type,
) VariableSizedArrayType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceVariableSizedArrayType)
	return NewVariableSizedArrayType(elementType)
}

func (VariableSizedArrayType) isType() {}

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
	common.UseConstantMemory(gauge, common.MemoryKindCadenceConstantSizedArrayType)
	return NewConstantSizedArrayType(size, elementType)
}

func (ConstantSizedArrayType) isType() {}

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
	common.UseConstantMemory(gauge, common.MemoryKindCadenceDictionaryType)
	return NewDictionaryType(keyType, elementType)
}

func (DictionaryType) isType() {}

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
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *StructType {
	return &StructType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
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
	common.UseConstantMemory(gauge, common.MemoryKindCadenceStructType)
	return NewStructType(location, qualifiedIdentifer, fields, initializers)
}

func (*StructType) isType() {}

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
	common.UseConstantMemory(gauge, common.MemoryKindCadenceResourceType)
	return NewResourceType(location, qualifiedIdentifer, fields, initializers)
}

func (*ResourceType) isType() {}

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
	common.UseConstantMemory(gauge, common.MemoryKindCadenceEventType)
	return NewEventType(location, qualifiedIdentifer, fields, initializer)
}

func (*EventType) isType() {}

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

func NewContractType(
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ContractType {
	return &ContractType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredContractType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ContractType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceContractType)
	return NewContractType(location, qualifiedIdentifer, fields, initializers)
}

func (*ContractType) isType() {}

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
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *StructInterfaceType {
	return &StructInterfaceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredStructInterfaceType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *StructInterfaceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceStructInterfaceType)
	return NewStructInterfaceType(location, qualifiedIdentifer, fields, initializers)
}

func (*StructInterfaceType) isType() {}

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

func NewResourceInterfaceType(
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceInterfaceType {
	return &ResourceInterfaceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredResourceInterfaceType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceInterfaceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceResourceInterfaceType)
	return NewResourceInterfaceType(location, qualifiedIdentifer, fields, initializers)
}

func (*ResourceInterfaceType) isType() {}

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

func NewContractInterfaceType(
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ContractInterfaceType {
	return &ContractInterfaceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func NewMeteredContractInterfaceType(
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ContractInterfaceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceContractInterfaceType)
	return NewContractInterfaceType(location, qualifiedIdentifer, fields, initializers)
}

func (*ContractInterfaceType) isType() {}

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

func (t *ContractInterfaceType) InterfaceInitializers() [][]Parameter {
	return t.Initializers
}

// Function

type FunctionType struct {
	typeID     string
	Parameters []Parameter
	ReturnType Type
}

func NewFunctionType(
	typeID string,
	parameters []Parameter,
	returnType Type,
) FunctionType {
	return FunctionType{
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
) FunctionType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceFunctionType)
	return NewFunctionType(typeID, parameters, returnType)
}

func (FunctionType) isType() {}

func (t FunctionType) ID() string {
	return t.typeID
}

func (t FunctionType) WithID(id string) FunctionType {
	t.typeID = id
	return t
}

// ReferenceType

type ReferenceType struct {
	Authorized bool
	Type       Type
}

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
	common.UseConstantMemory(gauge, common.MemoryKindCadenceReferenceType)
	return NewReferenceType(authorized, typ)
}

func (ReferenceType) isType() {}

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
	common.UseConstantMemory(gauge, common.MemoryKindCadenceRestrictedType)
	return NewRestrictedType(typeID, typ, restrictions)
}

func (RestrictedType) isType() {}

func (t RestrictedType) ID() string {
	return t.typeID
}

func (t RestrictedType) WithID(id string) RestrictedType {
	t.typeID = id
	return t
}

// BlockType

type BlockType struct{}

func NewBlockType() BlockType {
	return BlockType{}
}

func NewMeteredBlockType(
	gauge common.MemoryGauge,
) BlockType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewBlockType()
}

func (BlockType) isType() {}

func (BlockType) ID() string {
	return "Block"
}

// PathType

type PathType struct{}

func NewPathType() PathType {
	return PathType{}
}

func NewMeteredPathType(
	gauge common.MemoryGauge,
) PathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewPathType()
}

func (PathType) isType() {}

func (PathType) ID() string {
	return "Path"
}

// CapabilityPathType

type CapabilityPathType struct{}

func NewCapabilityPathType() CapabilityPathType {
	return CapabilityPathType{}
}

func NewMeteredCapabilityPathType(
	gauge common.MemoryGauge,
) CapabilityPathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewCapabilityPathType()
}

func (CapabilityPathType) isType() {}

func (CapabilityPathType) ID() string {
	return "CapabilityPath"
}

// StoragePathType

type StoragePathType struct{}

func NewStoragePathType() StoragePathType {
	return StoragePathType{}
}

func NewMeteredStoragePathType(
	gauge common.MemoryGauge,
) StoragePathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewStoragePathType()
}

func (StoragePathType) isType() {}

func (StoragePathType) ID() string {
	return "StoragePath"
}

// PublicPathType

type PublicPathType struct{}

func NewPublicPathType() PublicPathType {
	return PublicPathType{}
}

func NewMeteredPublicPathType(
	gauge common.MemoryGauge,
) PublicPathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewPublicPathType()
}

func (PublicPathType) isType() {}

func (PublicPathType) ID() string {
	return "PublicPath"
}

// PrivatePathType

type PrivatePathType struct{}

func NewPrivatePathType() PrivatePathType {
	return PrivatePathType{}
}

func NewMeteredPrivatePathType(
	gauge common.MemoryGauge,
) PrivatePathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewPrivatePathType()
}

func (PrivatePathType) isType() {}

func (PrivatePathType) ID() string {
	return "PrivatePath"
}

// CapabilityType

type CapabilityType struct {
	BorrowType Type
}

func NewCapabilityType(borrowType Type) CapabilityType {
	return CapabilityType{BorrowType: borrowType}
}

func NewMeteredCapabilityType(
	gauge common.MemoryGauge,
	borrowType Type,
) CapabilityType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceCapabilityType)
	return NewCapabilityType(borrowType)
}

func (CapabilityType) isType() {}

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
	common.UseConstantMemory(gauge, common.MemoryKindCadenceEnumType)
	return NewEnumType(location, qualifiedIdentifier, rawType, fields, initializers)
}

func (*EnumType) isType() {}

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

func (t *EnumType) CompositeInitializers() [][]Parameter {
	return t.Initializers
}

// AuthAccountType
type AuthAccountType struct{}

func NewAuthAccountType() AuthAccountType {
	return AuthAccountType{}
}

func NewMeteredAuthAccountType(
	gauge common.MemoryGauge,
) AuthAccountType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewAuthAccountType()
}

func (AuthAccountType) isType() {}

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
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewPublicAccountType()
}

func (PublicAccountType) isType() {}

func (PublicAccountType) ID() string {
	return "PublicAccount"
}

// DeployedContractType
type DeployedContractType struct{}

func NewDeployedContractType() DeployedContractType {
	return DeployedContractType{}
}

func NewMeteredDeployedContractType(
	gauge common.MemoryGauge,
) DeployedContractType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewDeployedContractType()
}

func (DeployedContractType) isType() {}

func (DeployedContractType) ID() string {
	return "DeployedContract"
}

// AuthAccountContractsType
type AuthAccountContractsType struct{}

func NewAuthAccountContractsType() AuthAccountContractsType {
	return AuthAccountContractsType{}
}

func NewMeteredAuthAccountContractsType(
	gauge common.MemoryGauge,
) AuthAccountContractsType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewAuthAccountContractsType()
}

func (AuthAccountContractsType) isType() {}

func (AuthAccountContractsType) ID() string {
	return "AuthAccount.Contracts"
}

// PublicAccountContractsType
type PublicAccountContractsType struct{}

func NewPublicAccountContractsType() PublicAccountContractsType {
	return PublicAccountContractsType{}
}

func NewMeteredPublicAccountContractsType(
	gauge common.MemoryGauge,
) PublicAccountContractsType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewPublicAccountContractsType()
}

func (PublicAccountContractsType) isType() {}

func (PublicAccountContractsType) ID() string {
	return "PublicAccount.Contracts"
}

// AuthAccountKeysType
type AuthAccountKeysType struct{}

func NewAuthAccountKeysType() AuthAccountKeysType {
	return AuthAccountKeysType{}
}

func NewMeteredAuthAccountKeysType(
	gauge common.MemoryGauge,
) AuthAccountKeysType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewAuthAccountKeysType()
}

func (AuthAccountKeysType) isType() {}

func (AuthAccountKeysType) ID() string {
	return "AuthAccount.Keys"
}

// PublicAccountContractsType
type PublicAccountKeysType struct{}

func NewPublicAccountKeysType() PublicAccountKeysType {
	return PublicAccountKeysType{}
}

func NewMeteredPublicAccountKeysType(
	gauge common.MemoryGauge,
) PublicAccountKeysType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewPublicAccountKeysType()
}

func (PublicAccountKeysType) isType() {}

func (PublicAccountKeysType) ID() string {
	return "PublicAccount.Keys"
}

// AccountKeyType
type AccountKeyType struct{}

func NewAccountKeyType() AccountKeyType {
	return AccountKeyType{}
}

func NewMeteredAccountKeyType(
	gauge common.MemoryGauge,
) AccountKeyType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NewAccountKeyType()
}

func (AccountKeyType) isType() {}

func (AccountKeyType) ID() string {
	return "AccountKey"
}
