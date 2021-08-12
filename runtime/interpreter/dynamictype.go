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

package interpreter

import (
	"github.com/onflow/cadence/runtime/sema"
)

type DynamicType interface {
	IsDynamicType()
	IsImportable() bool
}

type ReferenceDynamicType interface {
	DynamicType
	isReferenceType()
	Authorized() bool
	InnerType() DynamicType
	BorrowedType() sema.Type
}

// MetaTypeDynamicType

type MetaTypeDynamicType struct{}

func (MetaTypeDynamicType) IsDynamicType() {}

func (MetaTypeDynamicType) IsImportable() bool {
	return sema.MetaType.Importable
}

// VoidDynamicType

type VoidDynamicType struct{}

func (VoidDynamicType) IsDynamicType() {}

func (VoidDynamicType) IsImportable() bool {
	return sema.VoidType.Importable
}

// StringDynamicType

type StringDynamicType struct{}

func (StringDynamicType) IsDynamicType() {}

func (StringDynamicType) IsImportable() bool {
	return sema.StringType.Importable
}

// BoolDynamicType

type BoolDynamicType struct{}

func (BoolDynamicType) IsDynamicType() {}

func (BoolDynamicType) IsImportable() bool {
	return sema.BoolType.Importable
}

// ArrayDynamicType

type ArrayDynamicType struct {
	ElementTypes []DynamicType
	StaticType   ArrayStaticType
}

func (*ArrayDynamicType) IsDynamicType() {}

func (t *ArrayDynamicType) IsImportable() bool {
	for _, elementType := range t.ElementTypes {
		if !elementType.IsImportable() {
			return false
		}
	}

	return true
}

// NumberDynamicType

type NumberDynamicType struct {
	StaticType sema.Type
}

func (NumberDynamicType) IsDynamicType() {}

func (NumberDynamicType) IsImportable() bool {
	return true
}

// CompositeDynamicType

type CompositeDynamicType struct {
	StaticType sema.Type
}

func (CompositeDynamicType) IsDynamicType() {}

func (t CompositeDynamicType) IsImportable() bool {
	return t.StaticType.IsImportable(map[*sema.Member]bool{})
}

// DictionaryDynamicType

type DictionaryStaticTypeEntry struct {
	KeyType   DynamicType
	ValueType DynamicType
}

type DictionaryDynamicType struct {
	EntryTypes []DictionaryStaticTypeEntry
	StaticType DictionaryStaticType
}

func (*DictionaryDynamicType) IsDynamicType() {}

func (t *DictionaryDynamicType) IsImportable() bool {
	for _, entryType := range t.EntryTypes {
		if !entryType.KeyType.IsImportable() ||
			!entryType.ValueType.IsImportable() {
			return false
		}
	}

	return true
}

// NilDynamicType

type NilDynamicType struct{}

func (NilDynamicType) IsDynamicType() {}

func (NilDynamicType) IsImportable() bool {
	return true
}

// SomeDynamicType

type SomeDynamicType struct {
	InnerType DynamicType
}

func (SomeDynamicType) IsDynamicType() {}

func (t SomeDynamicType) IsImportable() bool {
	return t.InnerType.IsImportable()
}

// StorageReferenceDynamicType

type StorageReferenceDynamicType struct {
	authorized   bool
	innerType    DynamicType
	borrowedType sema.Type
}

func (StorageReferenceDynamicType) IsDynamicType() {}

func (StorageReferenceDynamicType) isReferenceType() {}

func (t StorageReferenceDynamicType) Authorized() bool {
	return t.authorized
}

func (t StorageReferenceDynamicType) InnerType() DynamicType {
	return t.innerType
}

func (t StorageReferenceDynamicType) BorrowedType() sema.Type {
	return t.borrowedType
}

func (StorageReferenceDynamicType) IsImportable() bool {
	return false
}

// EphemeralReferenceDynamicType

type EphemeralReferenceDynamicType struct {
	authorized   bool
	innerType    DynamicType
	borrowedType sema.Type
}

func (EphemeralReferenceDynamicType) IsDynamicType() {}

func (EphemeralReferenceDynamicType) isReferenceType() {}

func (t EphemeralReferenceDynamicType) Authorized() bool {
	return t.authorized
}

func (t EphemeralReferenceDynamicType) InnerType() DynamicType {
	return t.innerType
}

func (t EphemeralReferenceDynamicType) BorrowedType() sema.Type {
	return t.borrowedType
}

func (EphemeralReferenceDynamicType) IsImportable() bool {
	return false
}

// AddressDynamicType

type AddressDynamicType struct{}

func (AddressDynamicType) IsDynamicType() {}

func (AddressDynamicType) IsImportable() bool {
	return true
}

// FunctionDynamicType

type FunctionDynamicType struct{}

func (FunctionDynamicType) IsDynamicType() {}

func (FunctionDynamicType) IsImportable() bool {
	return false
}

// PrivatePathDynamicType

type PrivatePathDynamicType struct{}

func (PrivatePathDynamicType) IsDynamicType() {}

func (PrivatePathDynamicType) IsImportable() bool {
	return sema.PrivatePathType.Importable
}

// PublicPathDynamicType

type PublicPathDynamicType struct{}

func (PublicPathDynamicType) IsDynamicType() {}

func (PublicPathDynamicType) IsImportable() bool {
	return sema.PublicPathType.Importable
}

// StoragePathDynamicType

type StoragePathDynamicType struct{}

func (StoragePathDynamicType) IsDynamicType() {}

func (StoragePathDynamicType) IsImportable() bool {
	return sema.StoragePathType.Importable

}

// CapabilityDynamicType

type CapabilityDynamicType struct {
	BorrowType *sema.ReferenceType
}

func (CapabilityDynamicType) IsDynamicType() {}

func (CapabilityDynamicType) IsImportable() bool {
	return false
}

// DeployedContractDynamicType

type DeployedContractDynamicType struct{}

func (DeployedContractDynamicType) IsDynamicType() {}

func (DeployedContractDynamicType) IsImportable() bool {
	return sema.DeployedContractType.Importable
}

// BlockDynamicType

type BlockDynamicType struct{}

func (BlockDynamicType) IsDynamicType() {}

func (BlockDynamicType) IsImportable() bool {
	return sema.BlockType.Importable
}

// UnwrapOptionalDynamicType returns the type if it is not an optional type,
// or the inner-most type if it is (optional types are repeatedly unwrapped)
//
func UnwrapOptionalDynamicType(ty DynamicType) DynamicType {
	for {
		someDynamicType, ok := ty.(SomeDynamicType)
		if !ok {
			return ty
		}
		ty = someDynamicType.InnerType
	}
}
