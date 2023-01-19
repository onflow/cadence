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

func NewAnyType() AnyType {
	return AnyType{}
}

func NewMeteredAnyType(gauge common.MemoryGauge) AnyType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAnyType()
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

func NewAnyStructType() AnyStructType {
	return AnyStructType{}
}

func NewMeteredAnyStructType(gauge common.MemoryGauge) AnyStructType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAnyStructType()
}

func (AnyStructType) isType() {}

func (AnyStructType) ID() string {
	return "AnyStruct"
}

func (t AnyStructType) Equal(other Type) bool {
	return t == other
}

// AnyResourceType

type AnyResourceType struct{}

func NewAnyResourceType() AnyResourceType {
	return AnyResourceType{}
}

func NewMeteredAnyResourceType(gauge common.MemoryGauge) AnyResourceType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAnyResourceType()
}

func (AnyResourceType) isType() {}

func (AnyResourceType) ID() string {
	return "AnyResource"
}

func (t AnyResourceType) Equal(other Type) bool {
	return t == other
}

// OptionalType

type OptionalType struct {
	Type Type
}

func NewOptionalType(typ Type) OptionalType {
	return OptionalType{Type: typ}
}

func NewMeteredOptionalType(gauge common.MemoryGauge, typ Type) OptionalType {
	common.UseMemory(gauge, common.CadenceOptionalTypeMemoryUsage)
	return NewOptionalType(typ)
}

func (OptionalType) isType() {}

func (t OptionalType) ID() string {
	return fmt.Sprintf("%s?", t.Type.ID())
}

func (t OptionalType) Equal(other Type) bool {
	otherOptional, ok := other.(OptionalType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherOptional.Type)
}

// MetaType

type MetaType struct{}

func NewMetaType() MetaType {
	return MetaType{}
}

func NewMeteredMetaType(gauge common.MemoryGauge) MetaType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewMetaType()
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

func NewVoidType() VoidType {
	return VoidType{}
}

func NewMeteredVoidType(gauge common.MemoryGauge) VoidType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewVoidType()
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

func NewNeverType() NeverType {
	return NeverType{}
}

func NewMeteredNeverType(gauge common.MemoryGauge) NeverType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewNeverType()
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

func NewBoolType() BoolType {
	return BoolType{}
}

func NewMeteredBoolType(gauge common.MemoryGauge) BoolType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewBoolType()
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

func NewStringType() StringType {
	return StringType{}
}

func NewMeteredStringType(gauge common.MemoryGauge) StringType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewStringType()
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

func NewCharacterType() CharacterType {
	return CharacterType{}
}

func NewMeteredCharacterType(gauge common.MemoryGauge) CharacterType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewCharacterType()
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

func NewBytesType() BytesType {
	return BytesType{}
}

func NewMeteredBytesType(gauge common.MemoryGauge) BytesType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewBytesType()
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

func NewAddressType() AddressType {
	return AddressType{}
}

func NewMeteredAddressType(gauge common.MemoryGauge) AddressType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAddressType()
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

func NewNumberType() NumberType {
	return NumberType{}
}

func NewMeteredNumberType(gauge common.MemoryGauge) NumberType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewNumberType()
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

func NewSignedNumberType() SignedNumberType {
	return SignedNumberType{}
}

func NewMeteredSignedNumberType(gauge common.MemoryGauge) SignedNumberType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewSignedNumberType()
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

func NewIntegerType() IntegerType {
	return IntegerType{}
}

func NewMeteredIntegerType(gauge common.MemoryGauge) IntegerType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewIntegerType()
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

func NewSignedIntegerType() SignedIntegerType {
	return SignedIntegerType{}
}

func NewMeteredSignedIntegerType(gauge common.MemoryGauge) SignedIntegerType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewSignedIntegerType()
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

func NewFixedPointType() FixedPointType {
	return FixedPointType{}
}

func NewMeteredFixedPointType(gauge common.MemoryGauge) FixedPointType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewFixedPointType()
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

func NewSignedFixedPointType() SignedFixedPointType {
	return SignedFixedPointType{}
}

func NewMeteredSignedFixedPointType(gauge common.MemoryGauge) SignedFixedPointType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewSignedFixedPointType()
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

func NewIntType() IntType {
	return IntType{}
}

func NewMeteredIntType(gauge common.MemoryGauge) IntType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewIntType()
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

func NewInt8Type() Int8Type {
	return Int8Type{}
}

func (t Int8Type) Equal(other Type) bool {
	return t == other
}

func NewMeteredInt8Type(gauge common.MemoryGauge) Int8Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
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
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt16Type()
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

func NewInt32Type() Int32Type {
	return Int32Type{}
}

func NewMeteredInt32Type(gauge common.MemoryGauge) Int32Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt32Type()
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

func NewInt64Type() Int64Type {
	return Int64Type{}
}

func NewMeteredInt64Type(gauge common.MemoryGauge) Int64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt64Type()
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

func NewInt128Type() Int128Type {
	return Int128Type{}
}

func NewMeteredInt128Type(gauge common.MemoryGauge) Int128Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt128Type()
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

func NewInt256Type() Int256Type {
	return Int256Type{}
}

func NewMeteredInt256Type(gauge common.MemoryGauge) Int256Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewInt256Type()
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

func NewUIntType() UIntType {
	return UIntType{}
}

func NewMeteredUIntType(gauge common.MemoryGauge) UIntType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUIntType()
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

func NewUInt8Type() UInt8Type {
	return UInt8Type{}
}

func NewMeteredUInt8Type(gauge common.MemoryGauge) UInt8Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt8Type()
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

func NewUInt16Type() UInt16Type {
	return UInt16Type{}
}

func NewMeteredUInt16Type(gauge common.MemoryGauge) UInt16Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt16Type()
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

func NewUInt32Type() UInt32Type {
	return UInt32Type{}
}

func NewMeteredUInt32Type(gauge common.MemoryGauge) UInt32Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt32Type()
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

func NewUInt64Type() UInt64Type {
	return UInt64Type{}
}

func NewMeteredUInt64Type(gauge common.MemoryGauge) UInt64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt64Type()
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

func NewUInt128Type() UInt128Type {
	return UInt128Type{}
}

func NewMeteredUInt128Type(gauge common.MemoryGauge) UInt128Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt128Type()
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

func NewUInt256Type() UInt256Type {
	return UInt256Type{}
}

func NewMeteredUInt256Type(gauge common.MemoryGauge) UInt256Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUInt256Type()
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

func NewWord8Type() Word8Type {
	return Word8Type{}
}

func NewMeteredWord8Type(gauge common.MemoryGauge) Word8Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewWord8Type()
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

func NewWord16Type() Word16Type {
	return Word16Type{}
}

func NewMeteredWord16Type(gauge common.MemoryGauge) Word16Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewWord16Type()
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

func NewWord32Type() Word32Type {
	return Word32Type{}
}

func NewMeteredWord32Type(gauge common.MemoryGauge) Word32Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewWord32Type()
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

func NewWord64Type() Word64Type {
	return Word64Type{}
}

func NewMeteredWord64Type(gauge common.MemoryGauge) Word64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewWord64Type()
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

func NewFix64Type() Fix64Type {
	return Fix64Type{}
}

func NewMeteredFix64Type(gauge common.MemoryGauge) Fix64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewFix64Type()
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

func NewUFix64Type() UFix64Type {
	return UFix64Type{}
}

func NewMeteredUFix64Type(gauge common.MemoryGauge) UFix64Type {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewUFix64Type()
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
	common.UseMemory(gauge, common.CadenceVariableSizedArrayTypeMemoryUsage)
	return NewVariableSizedArrayType(elementType)
}

func (VariableSizedArrayType) isType() {}

func (t VariableSizedArrayType) ID() string {
	return fmt.Sprintf("[%s]", t.ElementType.ID())
}

func (t VariableSizedArrayType) Element() Type {
	return t.ElementType
}

func (t VariableSizedArrayType) Equal(other Type) bool {
	otherType, ok := other.(VariableSizedArrayType)
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

func (ConstantSizedArrayType) isType() {}

func (t ConstantSizedArrayType) ID() string {
	return fmt.Sprintf("[%s;%d]", t.ElementType.ID(), t.Size)
}

func (t ConstantSizedArrayType) Element() Type {
	return t.ElementType
}

func (t ConstantSizedArrayType) Equal(other Type) bool {
	otherType, ok := other.(ConstantSizedArrayType)
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

func (DictionaryType) isType() {}

func (t DictionaryType) ID() string {
	return fmt.Sprintf(
		"{%s:%s}",
		t.KeyType.ID(),
		t.ElementType.ID(),
	)
}

func (t DictionaryType) Equal(other Type) bool {
	otherType, ok := other.(DictionaryType)
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
	qualifiedIdentifer string,
	fields []Field,
	initializers [][]Parameter,
) *StructType {
	common.UseMemory(gauge, common.CadenceStructTypeMemoryUsage)
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
	common.UseMemory(gauge, common.CadenceEventTypeMemoryUsage)
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

func (t *ContractInterfaceType) Equal(other Type) bool {
	otherType, ok := other.(*ContractInterfaceType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier
}

// Function

// TODO: type parameters
type FunctionType struct {
	ReturnType Type
	typeID     string
	Parameters []Parameter
}

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

func (*FunctionType) isType() {}

func (t *FunctionType) ID() string {
	return t.typeID
}

func (t *FunctionType) WithID(id string) *FunctionType {
	t.typeID = id
	return t
}

func (t *FunctionType) Equal(other Type) bool {
	otherType, ok := other.(*FunctionType)
	if !ok {
		return false
	}

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
	common.UseMemory(gauge, common.CadenceReferenceTypeMemoryUsage)
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

func (t ReferenceType) Equal(other Type) bool {
	otherType, ok := other.(*ReferenceType)
	if !ok {
		return false
	}

	return t.Authorized == otherType.Authorized &&
		t.Type.Equal(otherType.Type)
}

// RestrictedType

type restrictionSet = map[Type]struct{}

type RestrictedType struct {
	typeID             string
	Type               Type
	Restrictions       []Type
	restrictionSet     restrictionSet
	restrictionSetOnce sync.Once
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
	common.UseMemory(gauge, common.CadenceRestrictedTypeMemoryUsage)
	return NewRestrictedType(typeID, typ, restrictions)
}

func (*RestrictedType) isType() {}

func (t *RestrictedType) ID() string {
	return t.typeID
}

func (t *RestrictedType) WithID(id string) *RestrictedType {
	t.typeID = id
	return t
}

func (t *RestrictedType) Equal(other Type) bool {
	otherType, ok := other.(*RestrictedType)
	if !ok {
		return false
	}

	if !t.Type.Equal(otherType.Type) {
		return false
	}

	t.initializeRestrictionSet()
	otherType.initializeRestrictionSet()

	if len(t.restrictionSet) != len(otherType.restrictionSet) {
		return false
	}

	for restriction := range t.restrictionSet { //nolint:maprange
		_, ok := otherType.restrictionSet[restriction]
		if !ok {
			return false
		}
	}

	return true
}

func (t *RestrictedType) initializeRestrictionSet() {
	t.restrictionSetOnce.Do(func() {
		t.restrictionSet = restrictionSet{}
		for _, restriction := range t.Restrictions {
			t.restrictionSet[restriction] = struct{}{}
		}
	})
}

// BlockType

type BlockType struct{}

func NewBlockType() BlockType {
	return BlockType{}
}

func NewMeteredBlockType(
	gauge common.MemoryGauge,
) BlockType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewBlockType()
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

func NewPathType() PathType {
	return PathType{}
}

func NewMeteredPathType(
	gauge common.MemoryGauge,
) PathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPathType()
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

func NewCapabilityPathType() CapabilityPathType {
	return CapabilityPathType{}
}

func NewMeteredCapabilityPathType(
	gauge common.MemoryGauge,
) CapabilityPathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewCapabilityPathType()
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

func NewStoragePathType() StoragePathType {
	return StoragePathType{}
}

func NewMeteredStoragePathType(
	gauge common.MemoryGauge,
) StoragePathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewStoragePathType()
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

func NewPublicPathType() PublicPathType {
	return PublicPathType{}
}

func NewMeteredPublicPathType(
	gauge common.MemoryGauge,
) PublicPathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPublicPathType()
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

func NewPrivatePathType() PrivatePathType {
	return PrivatePathType{}
}

func NewMeteredPrivatePathType(
	gauge common.MemoryGauge,
) PrivatePathType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPrivatePathType()
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
}

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

func (CapabilityType) isType() {}

func (t CapabilityType) ID() string {
	if t.BorrowType != nil {
		return fmt.Sprintf("Capability<%s>", t.BorrowType.ID())
	}
	return "Capability"
}

func (t CapabilityType) Equal(other Type) bool {
	otherType, ok := other.(CapabilityType)
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

func (t *EnumType) Equal(other Type) bool {
	otherType, ok := other.(*EnumType)
	if !ok {
		return false
	}

	return t.Location == otherType.Location &&
		t.QualifiedIdentifier == otherType.QualifiedIdentifier &&
		t.RawType.Equal(otherType.RawType)
}

// AuthAccountType
type AuthAccountType struct{}

func NewAuthAccountType() AuthAccountType {
	return AuthAccountType{}
}

func NewMeteredAuthAccountType(
	gauge common.MemoryGauge,
) AuthAccountType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAuthAccountType()
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

func NewPublicAccountType() PublicAccountType {
	return PublicAccountType{}
}

func NewMeteredPublicAccountType(
	gauge common.MemoryGauge,
) PublicAccountType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPublicAccountType()
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

func NewDeployedContractType() DeployedContractType {
	return DeployedContractType{}
}

func NewMeteredDeployedContractType(
	gauge common.MemoryGauge,
) DeployedContractType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewDeployedContractType()
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

func NewAuthAccountContractsType() AuthAccountContractsType {
	return AuthAccountContractsType{}
}

func NewMeteredAuthAccountContractsType(
	gauge common.MemoryGauge,
) AuthAccountContractsType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAuthAccountContractsType()
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

func NewPublicAccountContractsType() PublicAccountContractsType {
	return PublicAccountContractsType{}
}

func NewMeteredPublicAccountContractsType(
	gauge common.MemoryGauge,
) PublicAccountContractsType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPublicAccountContractsType()
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

func NewAuthAccountKeysType() AuthAccountKeysType {
	return AuthAccountKeysType{}
}

func NewMeteredAuthAccountKeysType(
	gauge common.MemoryGauge,
) AuthAccountKeysType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAuthAccountKeysType()
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

func NewPublicAccountKeysType() PublicAccountKeysType {
	return PublicAccountKeysType{}
}

func NewMeteredPublicAccountKeysType(
	gauge common.MemoryGauge,
) PublicAccountKeysType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewPublicAccountKeysType()
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

func NewAccountKeyType() AccountKeyType {
	return AccountKeyType{}
}

func NewMeteredAccountKeyType(
	gauge common.MemoryGauge,
) AccountKeyType {
	common.UseMemory(gauge, common.CadenceSimpleTypeMemoryUsage)
	return NewAccountKeyType()
}

func (AccountKeyType) isType() {}

func (AccountKeyType) ID() string {
	return "AccountKey"
}

func (t AccountKeyType) Equal(other Type) bool {
	return t == other
}
