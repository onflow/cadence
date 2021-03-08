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
}

type ReferenceDynamicType interface {
	DynamicType
	isReferenceType()
	Authorized() bool
	InnerType() DynamicType
}

// MetaTypeDynamicType

type MetaTypeDynamicType struct{}

func (MetaTypeDynamicType) IsDynamicType() {}

// VoidDynamicType

type VoidDynamicType struct{}

func (VoidDynamicType) IsDynamicType() {}

// StringDynamicType

type StringDynamicType struct{}

func (StringDynamicType) IsDynamicType() {}

// BoolDynamicType

type BoolDynamicType struct{}

func (BoolDynamicType) IsDynamicType() {}

// ArrayDynamicType

type ArrayDynamicType struct {
	ElementTypes []DynamicType
}

func (ArrayDynamicType) IsDynamicType() {}

// NumberDynamicType

type NumberDynamicType struct {
	StaticType sema.Type
}

func (NumberDynamicType) IsDynamicType() {}

// CompositeDynamicType

type CompositeDynamicType struct {
	StaticType sema.Type
}

func (CompositeDynamicType) IsDynamicType() {}

// DictionaryDynamicType

type DictionaryDynamicType struct {
	EntryTypes []struct{ KeyType, ValueType DynamicType }
}

func (DictionaryDynamicType) IsDynamicType() {}

// NilDynamicType

type NilDynamicType struct{}

func (NilDynamicType) IsDynamicType() {}

// SomeDynamicType

type SomeDynamicType struct {
	InnerType DynamicType
}

func (SomeDynamicType) IsDynamicType() {}

// StorageReferenceDynamicType

type StorageReferenceDynamicType struct {
	authorized bool
	innerType  DynamicType
}

func (StorageReferenceDynamicType) IsDynamicType() {}

func (StorageReferenceDynamicType) isReferenceType() {}

func (t StorageReferenceDynamicType) Authorized() bool {
	return t.authorized
}

func (t StorageReferenceDynamicType) InnerType() DynamicType {
	return t.innerType
}

// EphemeralReferenceDynamicType

type EphemeralReferenceDynamicType struct {
	authorized bool
	innerType  DynamicType
}

func (EphemeralReferenceDynamicType) IsDynamicType() {}

func (EphemeralReferenceDynamicType) isReferenceType() {}

func (t EphemeralReferenceDynamicType) Authorized() bool {
	return t.authorized
}

func (t EphemeralReferenceDynamicType) InnerType() DynamicType {
	return t.innerType
}

// AddressDynamicType

type AddressDynamicType struct{}

func (AddressDynamicType) IsDynamicType() {}

// FunctionDynamicType

type FunctionDynamicType struct{}

func (FunctionDynamicType) IsDynamicType() {}

// PrivatePathDynamicType

type PrivatePathDynamicType struct{}

func (PrivatePathDynamicType) IsDynamicType() {}

// PublicPathDynamicType

type PublicPathDynamicType struct{}

func (PublicPathDynamicType) IsDynamicType() {}

// StoragePathDynamicType

type StoragePathDynamicType struct{}

func (StoragePathDynamicType) IsDynamicType() {}

// CapabilityDynamicType

type CapabilityDynamicType struct {
	BorrowType *sema.ReferenceType
}

func (CapabilityDynamicType) IsDynamicType() {}

// DeployedContractDynamicType

type DeployedContractDynamicType struct{}

func (DeployedContractDynamicType) IsDynamicType() {}

// BlockDynamicType

type BlockDynamicType struct{}

func (BlockDynamicType) IsDynamicType() {}
