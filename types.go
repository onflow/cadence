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

func (AnyType) isType() {}

func (AnyType) ID() string {
	return "Any"
}

// AnyStructType

type AnyStructType struct{}

func (AnyStructType) isType() {}

func (AnyStructType) ID() string {
	return "AnyStruct"
}

// AnyResourceType

type AnyResourceType struct{}

func (AnyResourceType) isType() {}

func (AnyResourceType) ID() string {
	return "AnyResource"
}

// OptionalType

type OptionalType struct {
	Type Type
}

func (OptionalType) isType() {}

func (t OptionalType) ID() string {
	return fmt.Sprintf("%s?", t.Type.ID())
}

// MetaType

type MetaType struct{}

func (MetaType) isType() {}

func (MetaType) ID() string {
	return "Type"
}

// VoidType

type VoidType struct{}

func (VoidType) isType() {}

func (VoidType) ID() string {
	return "Void"
}

// NeverType

type NeverType struct{}

func (NeverType) isType() {}

func (NeverType) ID() string {
	return "Never"
}

// BoolType

type BoolType struct{}

func (BoolType) isType() {}

func (BoolType) ID() string {
	return "Bool"
}

// StringType

type StringType struct{}

func (StringType) isType() {}

func (StringType) ID() string {
	return "String"
}

// CharacterType

type CharacterType struct{}

func (CharacterType) isType() {}

func (CharacterType) ID() string {
	return "Character"
}

// BytesType

type BytesType struct{}

func (BytesType) isType() {}

func (BytesType) ID() string {
	return "Bytes"
}

// AddressType

type AddressType struct{}

func (AddressType) isType() {}

func (AddressType) ID() string {
	return "Address"
}

// NumberType

type NumberType struct{}

func (NumberType) isType() {}

func (NumberType) ID() string {
	return "Number"
}

// SignedNumberType

type SignedNumberType struct{}

func (SignedNumberType) isType() {}

func (SignedNumberType) ID() string {
	return "SignedNumber"
}

// IntegerType

type IntegerType struct{}

func (IntegerType) isType() {}

func (IntegerType) ID() string {
	return "Integer"
}

// SignedIntegerType

type SignedIntegerType struct{}

func (SignedIntegerType) isType() {}

func (SignedIntegerType) ID() string {
	return "SignedInteger"
}

// FixedPointType

type FixedPointType struct{}

func (FixedPointType) isType() {}

func (FixedPointType) ID() string {
	return "FixedPoint"
}

// SignedFixedPointType

type SignedFixedPointType struct{}

func (SignedFixedPointType) isType() {}

func (SignedFixedPointType) ID() string {
	return "SignedFixedPoint"
}

// IntType

type IntType struct{}

func (IntType) isType() {}

func (IntType) ID() string {
	return "Int"
}

// Int8Type

type Int8Type struct{}

func (Int8Type) isType() {}

func (Int8Type) ID() string {
	return "Int8"
}

// Int16Type

type Int16Type struct{}

func (Int16Type) isType() {}

func (Int16Type) ID() string {
	return "Int16"
}

// Int32Type

type Int32Type struct{}

func (Int32Type) isType() {}

func (Int32Type) ID() string {
	return "Int32"
}

// Int64Type

type Int64Type struct{}

func (Int64Type) isType() {}

func (Int64Type) ID() string {
	return "Int64"
}

// Int128Type

type Int128Type struct{}

func (Int128Type) isType() {}

func (Int128Type) ID() string {
	return "Int128"
}

// Int256Type

type Int256Type struct{}

func (Int256Type) isType() {}

func (Int256Type) ID() string {
	return "Int256"
}

// UIntType

type UIntType struct{}

func (UIntType) isType() {}

func (UIntType) ID() string {
	return "UInt"
}

// UInt8Type

type UInt8Type struct{}

func (UInt8Type) isType() {}

func (UInt8Type) ID() string {
	return "UInt8"
}

// UInt16Type

type UInt16Type struct{}

func (UInt16Type) isType() {}

func (UInt16Type) ID() string {
	return "UInt16"
}

// UInt32Type

type UInt32Type struct{}

func (UInt32Type) isType() {}

func (UInt32Type) ID() string {
	return "UInt32"
}

// UInt64Type

type UInt64Type struct{}

func (UInt64Type) isType() {}

func (UInt64Type) ID() string {
	return "UInt64"
}

// UInt128Type

type UInt128Type struct{}

func (UInt128Type) isType() {}

func (UInt128Type) ID() string {
	return "UInt128"
}

// UInt256Type

type UInt256Type struct{}

func (UInt256Type) isType() {}

func (UInt256Type) ID() string {
	return "UInt256"
}

// Word8Type

type Word8Type struct{}

func (Word8Type) isType() {}

func (Word8Type) ID() string {
	return "Word8"
}

// Word16Type

type Word16Type struct{}

func (Word16Type) isType() {}

func (Word16Type) ID() string {
	return "Word16"
}

// Word32Type

type Word32Type struct{}

func (Word32Type) isType() {}

func (Word32Type) ID() string {
	return "Word32"
}

// Word64Type

type Word64Type struct{}

func (Word64Type) isType() {}

func (Word64Type) ID() string {
	return "Word64"
}

// Fix64Type

type Fix64Type struct{}

func (Fix64Type) isType() {}

func (Fix64Type) ID() string {
	return "Fix64"
}

// UFix64Type

type UFix64Type struct{}

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

// Parameter

type Parameter struct {
	Label      string
	Identifier string
	Type       Type
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

func (BlockType) isType() {}

func (BlockType) ID() string {
	return "Block"
}

// PathType

type PathType struct{}

func (PathType) isType() {}

func (PathType) ID() string {
	return "Path"
}

// CapabilityPathType

type CapabilityPathType struct{}

func (CapabilityPathType) isType() {}

func (CapabilityPathType) ID() string {
	return "CapabilityPath"
}

// StoragePathType

type StoragePathType struct{}

func (StoragePathType) isType() {}

func (StoragePathType) ID() string {
	return "StoragePath"
}

// PublicPathType

type PublicPathType struct{}

func (PublicPathType) isType() {}

func (PublicPathType) ID() string {
	return "PublicPath"
}

// PrivatePathType

type PrivatePathType struct{}

func (PrivatePathType) isType() {}

func (PrivatePathType) ID() string {
	return "PrivatePath"
}

// CapabilityType

type CapabilityType struct {
	BorrowType Type
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

func (AuthAccountType) isType() {}

func (AuthAccountType) ID() string {
	return "AuthAccount"
}

// PublicAccountType
type PublicAccountType struct{}

func (PublicAccountType) isType() {}

func (PublicAccountType) ID() string {
	return "PublicAccount"
}

// DeployedContractType
type DeployedContractType struct{}

func (DeployedContractType) isType() {}

func (DeployedContractType) ID() string {
	return "DeployedContract"
}

// AuthAccountContractsType
type AuthAccountContractsType struct{}

func (AuthAccountContractsType) isType() {}

func (AuthAccountContractsType) ID() string {
	return "AuthAccount.Contracts"
}

// PublicAccountContractsType
type PublicAccountContractsType struct{}

func (PublicAccountContractsType) isType() {}

func (PublicAccountContractsType) ID() string {
	return "PublicAccount.Contracts"
}

// AuthAccountKeysType
type AuthAccountKeysType struct{}

func (AuthAccountKeysType) isType() {}

func (AuthAccountKeysType) ID() string {
	return "AuthAccount.Keys"
}

// PublicAccountContractsType
type PublicAccountKeysType struct{}

func (PublicAccountKeysType) isType() {}

func (PublicAccountKeysType) ID() string {
	return "PublicAccount.Keys"
}

// AccountKeyType
type AccountKeyType struct{}

func (AccountKeyType) isType() {}

func (AccountKeyType) ID() string {
	return "AccountKey"
}
