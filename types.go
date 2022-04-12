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

func NewAnyType(gauge common.MemoryGauge) AnyType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return AnyType{}
}

func (AnyType) isType() {}

func (AnyType) ID() string {
	return "Any"
}

// AnyStructType

type AnyStructType struct{}

func NewAnyStructType(gauge common.MemoryGauge) AnyStructType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return AnyStructType{}
}

func (AnyStructType) isType() {}

func (AnyStructType) ID() string {
	return "AnyStruct"
}

// AnyResourceType

type AnyResourceType struct{}

func NewAnyResourceType(gauge common.MemoryGauge) AnyResourceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return AnyResourceType{}
}

func (AnyResourceType) isType() {}

func (AnyResourceType) ID() string {
	return "AnyResource"
}

// OptionalType

type OptionalType struct {
	Type Type
}

func NewOptionalType(gauge common.MemoryGauge, typ Type) OptionalType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceOptionalType)
	return OptionalType{Type: typ}
}

func (OptionalType) isType() {}

func (t OptionalType) ID() string {
	return fmt.Sprintf("%s?", t.Type.ID())
}

// MetaType

type MetaType struct{}

func NewMetaType(gauge common.MemoryGauge) MetaType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return MetaType{}
}

func (MetaType) isType() {}

func (MetaType) ID() string {
	return "Type"
}

// VoidType

type VoidType struct{}

func NewVoidType(gauge common.MemoryGauge) VoidType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return VoidType{}
}

func (VoidType) isType() {}

func (VoidType) ID() string {
	return "Void"
}

// NeverType

type NeverType struct{}

func NewNeverType(gauge common.MemoryGauge) NeverType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NeverType{}
}

func (NeverType) isType() {}

func (NeverType) ID() string {
	return "Never"
}

// BoolType

type BoolType struct{}

func NewBoolType(gauge common.MemoryGauge) BoolType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return BoolType{}
}

func (BoolType) isType() {}

func (BoolType) ID() string {
	return "Bool"
}

// StringType

type StringType struct{}

func NewStringType(gauge common.MemoryGauge) StringType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return StringType{}
}

func (StringType) isType() {}

func (StringType) ID() string {
	return "String"
}

// CharacterType

type CharacterType struct{}

func NewCharacterType(gauge common.MemoryGauge) CharacterType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return CharacterType{}
}

func (CharacterType) isType() {}

func (CharacterType) ID() string {
	return "Character"
}

// BytesType

type BytesType struct{}

func NewBytesType(gauge common.MemoryGauge) BytesType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return BytesType{}
}

func (BytesType) isType() {}

func (BytesType) ID() string {
	return "Bytes"
}

// AddressType

type AddressType struct{}

func NewAddressType(gauge common.MemoryGauge) AddressType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return AddressType{}
}

func (AddressType) isType() {}

func (AddressType) ID() string {
	return "Address"
}

// NumberType

type NumberType struct{}

func NewNumberType(gauge common.MemoryGauge) NumberType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return NumberType{}
}

func (NumberType) isType() {}

func (NumberType) ID() string {
	return "Number"
}

// SignedNumberType

type SignedNumberType struct{}

func NewSignedNumberType(gauge common.MemoryGauge) SignedNumberType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return SignedNumberType{}
}

func (SignedNumberType) isType() {}

func (SignedNumberType) ID() string {
	return "SignedNumber"
}

// IntegerType

type IntegerType struct{}

func NewIntegerType(gauge common.MemoryGauge) IntegerType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return IntegerType{}
}

func (IntegerType) isType() {}

func (IntegerType) ID() string {
	return "Integer"
}

// SignedIntegerType

type SignedIntegerType struct{}

func NewSignedIntegerType(gauge common.MemoryGauge) SignedIntegerType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return SignedIntegerType{}
}

func (SignedIntegerType) isType() {}

func (SignedIntegerType) ID() string {
	return "SignedInteger"
}

// FixedPointType

type FixedPointType struct{}

func NewFixedPointType(gauge common.MemoryGauge) FixedPointType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return FixedPointType{}
}

func (FixedPointType) isType() {}

func (FixedPointType) ID() string {
	return "FixedPoint"
}

// SignedFixedPointType

type SignedFixedPointType struct{}

func NewSignedFixedPointType(gauge common.MemoryGauge) SignedFixedPointType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return SignedFixedPointType{}
}

func (SignedFixedPointType) isType() {}

func (SignedFixedPointType) ID() string {
	return "SignedFixedPoint"
}

// IntType

type IntType struct{}

func NewIntType(gauge common.MemoryGauge) IntType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return IntType{}
}

func (IntType) isType() {}

func (IntType) ID() string {
	return "Int"
}

// Int8Type

type Int8Type struct{}

func NewInt8Type(gauge common.MemoryGauge) Int8Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Int8Type{}
}

func (Int8Type) isType() {}

func (Int8Type) ID() string {
	return "Int8"
}

// Int16Type

type Int16Type struct{}

func NewInt16Type(gauge common.MemoryGauge) Int16Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Int16Type{}
}

func (Int16Type) isType() {}

func (Int16Type) ID() string {
	return "Int16"
}

// Int32Type

type Int32Type struct{}

func NewInt32Type(gauge common.MemoryGauge) Int32Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Int32Type{}
}

func (Int32Type) isType() {}

func (Int32Type) ID() string {
	return "Int32"
}

// Int64Type

type Int64Type struct{}

func NewInt64Type(gauge common.MemoryGauge) Int64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Int64Type{}
}

func (Int64Type) isType() {}

func (Int64Type) ID() string {
	return "Int64"
}

// Int128Type

type Int128Type struct{}

func NewInt128Type(gauge common.MemoryGauge) Int128Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Int128Type{}
}

func (Int128Type) isType() {}

func (Int128Type) ID() string {
	return "Int128"
}

// Int256Type

type Int256Type struct{}

func NewInt256Type(gauge common.MemoryGauge) Int256Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Int256Type{}
}

func (Int256Type) isType() {}

func (Int256Type) ID() string {
	return "Int256"
}

// UIntType

type UIntType struct{}

func NewUIntType(gauge common.MemoryGauge) UIntType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return UIntType{}
}

func (UIntType) isType() {}

func (UIntType) ID() string {
	return "UInt"
}

// UInt8Type

type UInt8Type struct{}

func NewUInt8Type(gauge common.MemoryGauge) UInt8Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return UInt8Type{}
}

func (UInt8Type) isType() {}

func (UInt8Type) ID() string {
	return "UInt8"
}

// UInt16Type

type UInt16Type struct{}

func NewUInt16Type(gauge common.MemoryGauge) UInt16Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return UInt16Type{}
}

func (UInt16Type) isType() {}

func (UInt16Type) ID() string {
	return "UInt16"
}

// UInt32Type

type UInt32Type struct{}

func NewUInt32Type(gauge common.MemoryGauge) UInt32Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return UInt32Type{}
}

func (UInt32Type) isType() {}

func (UInt32Type) ID() string {
	return "UInt32"
}

// UInt64Type

type UInt64Type struct{}

func NewUInt64Type(gauge common.MemoryGauge) UInt64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return UInt64Type{}
}

func (UInt64Type) isType() {}

func (UInt64Type) ID() string {
	return "UInt64"
}

// UInt128Type

type UInt128Type struct{}

func NewUInt128Type(gauge common.MemoryGauge) UInt128Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return UInt128Type{}
}

func (UInt128Type) isType() {}

func (UInt128Type) ID() string {
	return "UInt128"
}

// UInt256Type

type UInt256Type struct{}

func NewUInt256Type(gauge common.MemoryGauge) UInt256Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return UInt256Type{}
}

func (UInt256Type) isType() {}

func (UInt256Type) ID() string {
	return "UInt256"
}

// Word8Type

type Word8Type struct{}

func NewWord8Type(gauge common.MemoryGauge) Word8Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Word8Type{}
}

func (Word8Type) isType() {}

func (Word8Type) ID() string {
	return "Word8"
}

// Word16Type

type Word16Type struct{}

func NewWord16Type(gauge common.MemoryGauge) Word16Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Word16Type{}
}

func (Word16Type) isType() {}

func (Word16Type) ID() string {
	return "Word16"
}

// Word32Type

type Word32Type struct{}

func NewWord32Type(gauge common.MemoryGauge) Word32Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Word32Type{}
}

func (Word32Type) isType() {}

func (Word32Type) ID() string {
	return "Word32"
}

// Word64Type

type Word64Type struct{}

func NewWord64Type(gauge common.MemoryGauge) Word64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Word64Type{}
}

func (Word64Type) isType() {}

func (Word64Type) ID() string {
	return "Word64"
}

// Fix64Type

type Fix64Type struct{}

func NewFix64Type(gauge common.MemoryGauge) Fix64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return Fix64Type{}
}

func (Fix64Type) isType() {}

func (Fix64Type) ID() string {
	return "Fix64"
}

// UFix64Type

type UFix64Type struct{}

func NewUFix64Type(gauge common.MemoryGauge) UFix64Type {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return UFix64Type{}
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
	gauge common.MemoryGauge,
	elementType Type,
) VariableSizedArrayType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceVariableSizedArrayType)
	return VariableSizedArrayType{ElementType: elementType}
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
	gauge common.MemoryGauge,
	size uint,
	elementType Type,
) ConstantSizedArrayType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceConstantSizedArrayType)
	return ConstantSizedArrayType{
		Size:        size,
		ElementType: elementType,
	}
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
	gauge common.MemoryGauge,
	keyType Type,
	elementType Type,
) DictionaryType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceDictionaryType)
	return DictionaryType{
		KeyType:     keyType,
		ElementType: elementType,
	}
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
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *StructType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceStructType)
	return &StructType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func (*StructType) isType() {}

func (t *StructType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(t.QualifiedIdentifier))
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
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceResourceType)
	return &ResourceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func (*ResourceType) isType() {}

func (t *ResourceType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(t.QualifiedIdentifier))
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
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializer []Parameter,
) *EventType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceEventType)
	return &EventType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializer:         initializer,
	}
}

func (*EventType) isType() {}

func (t *EventType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(t.QualifiedIdentifier))
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
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ContractType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceContractType)
	return &ContractType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func (*ContractType) isType() {}

func (t *ContractType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(t.QualifiedIdentifier))
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
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *StructInterfaceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceStructInterfaceType)
	return &StructInterfaceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func (*StructInterfaceType) isType() {}

func (t *StructInterfaceType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(t.QualifiedIdentifier))
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
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ResourceInterfaceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceResourceInterfaceType)
	return &ResourceInterfaceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func (*ResourceInterfaceType) isType() {}

func (t *ResourceInterfaceType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(t.QualifiedIdentifier))
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
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *ContractInterfaceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceContractInterfaceType)
	return &ContractInterfaceType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifer,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func (*ContractInterfaceType) isType() {}

func (t *ContractInterfaceType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(t.QualifiedIdentifier))
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
	gauge common.MemoryGauge,
	typeID string,
	parameters []Parameter,
	returnType Type,
) FunctionType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceFunctionType)
	return FunctionType{
		typeID:     typeID,
		Parameters: parameters,
		ReturnType: returnType,
	}
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
	gauge common.MemoryGauge,
	authorized bool,
	typ Type,
) ReferenceType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceReferenceType)
	return ReferenceType{
		Authorized: authorized,
		Type:       typ,
	}
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
	gauge common.MemoryGauge,
	typeID string,
	typ Type,
	restrictions []Type,
) *RestrictedType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceRestrictedType)
	return &RestrictedType{
		typeID:       typeID,
		Type:         typ,
		Restrictions: restrictions,
	}
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

func NewBlockType(
	gauge common.MemoryGauge,
) BlockType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return BlockType{}
}

func (BlockType) isType() {}

func (BlockType) ID() string {
	return "Block"
}

// PathType

type PathType struct{}

func NewPathType(
	gauge common.MemoryGauge,
) PathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return PathType{}
}

func (PathType) isType() {}

func (PathType) ID() string {
	return "Path"
}

// CapabilityPathType

type CapabilityPathType struct{}

func NewCapabilityPathType(
	gauge common.MemoryGauge,
) CapabilityPathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return CapabilityPathType{}
}

func (CapabilityPathType) isType() {}

func (CapabilityPathType) ID() string {
	return "CapabilityPath"
}

// StoragePathType

type StoragePathType struct{}

func NewStoragePathType(
	gauge common.MemoryGauge,
) StoragePathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return StoragePathType{}
}

func (StoragePathType) isType() {}

func (StoragePathType) ID() string {
	return "StoragePath"
}

// PublicPathType

type PublicPathType struct{}

func NewPublicPathType(
	gauge common.MemoryGauge,
) PublicPathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return PublicPathType{}
}

func (PublicPathType) isType() {}

func (PublicPathType) ID() string {
	return "PublicPath"
}

// PrivatePathType

type PrivatePathType struct{}

func NewPrivatePathType(
	gauge common.MemoryGauge,
) PrivatePathType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return PrivatePathType{}
}

func (PrivatePathType) isType() {}

func (PrivatePathType) ID() string {
	return "PrivatePath"
}

// CapabilityType

type CapabilityType struct {
	BorrowType Type
}

func NewCapabilityType(
	gauge common.MemoryGauge,
	borrowType Type,
) CapabilityType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceCapabilityType)
	return CapabilityType{BorrowType: borrowType}
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
	gauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	rawType Type,
	fields []Field,
	initializers [][]Parameter,
) *EnumType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceEnumType)
	return &EnumType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		RawType:             rawType,
		Fields:              fields,
		Initializers:        initializers,
	}
}

func (*EnumType) isType() {}

func (t *EnumType) ID() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}

	return string(t.Location.TypeID(t.QualifiedIdentifier))
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

func NewAuthAccountType(
	gauge common.MemoryGauge,
) AuthAccountType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return AuthAccountType{}
}

func (AuthAccountType) isType() {}

func (AuthAccountType) ID() string {
	return "AuthAccount"
}

// PublicAccountType
type PublicAccountType struct{}

func NewPublicAccountType(
	gauge common.MemoryGauge,
) PublicAccountType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return PublicAccountType{}
}

func (PublicAccountType) isType() {}

func (PublicAccountType) ID() string {
	return "PublicAccount"
}

// DeployedContractType
type DeployedContractType struct{}

func NewDeployedContractType(
	gauge common.MemoryGauge,
) DeployedContractType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return DeployedContractType{}
}

func (DeployedContractType) isType() {}

func (DeployedContractType) ID() string {
	return "DeployedContract"
}

// AuthAccountContractsType
type AuthAccountContractsType struct{}

func NewAuthAccountContractsType(
	gauge common.MemoryGauge,
) AuthAccountContractsType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return AuthAccountContractsType{}
}

func (AuthAccountContractsType) isType() {}

func (AuthAccountContractsType) ID() string {
	return "AuthAccount.Contracts"
}

// PublicAccountContractsType
type PublicAccountContractsType struct{}

func NewPublicAccountContractsType(
	gauge common.MemoryGauge,
) PublicAccountContractsType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return PublicAccountContractsType{}
}

func (PublicAccountContractsType) isType() {}

func (PublicAccountContractsType) ID() string {
	return "PublicAccount.Contracts"
}

// AuthAccountKeysType
type AuthAccountKeysType struct{}

func NewAuthAccountKeysType(
	gauge common.MemoryGauge,
) AuthAccountKeysType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return AuthAccountKeysType{}
}

func (AuthAccountKeysType) isType() {}

func (AuthAccountKeysType) ID() string {
	return "AuthAccount.Keys"
}

// PublicAccountContractsType
type PublicAccountKeysType struct{}

func NewPublicAccountKeysType(
	gauge common.MemoryGauge,
) PublicAccountKeysType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return PublicAccountKeysType{}
}

func (PublicAccountKeysType) isType() {}

func (PublicAccountKeysType) ID() string {
	return "PublicAccount.Keys"
}

// AccountKeyType
type AccountKeyType struct{}

func NewAccountKeyType(
	gauge common.MemoryGauge,
) AccountKeyType {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceSimpleType)
	return AccountKeyType{}
}

func (AccountKeyType) isType() {}

func (AccountKeyType) ID() string {
	return "AccountKey"
}
