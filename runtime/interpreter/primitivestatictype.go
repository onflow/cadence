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

package interpreter

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=PrimitiveStaticType -trimprefix=PrimitiveStaticType

// PrimitiveStaticType

type PrimitiveStaticType uint

func (t PrimitiveStaticType) Equal(other StaticType) bool {
	otherPrimitiveType, ok := other.(PrimitiveStaticType)
	if !ok {
		return false
	}

	return t == otherPrimitiveType
}

func NewPrimitiveStaticType(
	memoryGauge common.MemoryGauge,
	staticType PrimitiveStaticType,
) PrimitiveStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindPrimitiveStaticType)
	return staticType
}

// !!! *WARNING* !!!
//
// Only add new types by:
// - replacing existing placeholders (`_`) with new types
// - appending new types
//
// Only remove types by:
// - replace existing types with a placeholder `_`
//
// DO *NOT* REPLACE EXISTING TYPES!
// DO *NOT* ADD NEW TYPES IN BETWEEN!

const (
	PrimitiveStaticTypeUnknown PrimitiveStaticType = iota
	PrimitiveStaticTypeVoid
	PrimitiveStaticTypeAny
	PrimitiveStaticTypeNever
	PrimitiveStaticTypeAnyStruct
	PrimitiveStaticTypeAnyResource
	PrimitiveStaticTypeBool
	PrimitiveStaticTypeAddress
	PrimitiveStaticTypeString
	PrimitiveStaticTypeCharacter
	PrimitiveStaticTypeMetaType
	PrimitiveStaticTypeBlock
	_
	_
	_
	_
	_
	_

	// Number
	PrimitiveStaticTypeNumber
	PrimitiveStaticTypeSignedNumber
	_
	_
	_
	_

	// Integer
	PrimitiveStaticTypeInteger
	PrimitiveStaticTypeSignedInteger
	_
	_
	_
	_

	// FixedPoint
	PrimitiveStaticTypeFixedPoint
	PrimitiveStaticTypeSignedFixedPoint
	_
	_
	_
	_

	// Int*
	PrimitiveStaticTypeInt
	PrimitiveStaticTypeInt8
	PrimitiveStaticTypeInt16
	PrimitiveStaticTypeInt32
	PrimitiveStaticTypeInt64
	PrimitiveStaticTypeInt128
	PrimitiveStaticTypeInt256
	_

	// UInt*
	PrimitiveStaticTypeUInt
	PrimitiveStaticTypeUInt8
	PrimitiveStaticTypeUInt16
	PrimitiveStaticTypeUInt32
	PrimitiveStaticTypeUInt64
	PrimitiveStaticTypeUInt128
	PrimitiveStaticTypeUInt256
	_

	// Word*
	_
	PrimitiveStaticTypeWord8
	PrimitiveStaticTypeWord16
	PrimitiveStaticTypeWord32
	PrimitiveStaticTypeWord64
	_ // future: Word128
	_ // future: Word256
	_

	// Fix*
	_
	_ // future: Fix8
	_ // future: Fix16
	_ // future: Fix32
	PrimitiveStaticTypeFix64
	_ // future: Fix128
	_ // future: Fix256
	_

	// UFix*
	_
	_ // future: UFix8
	_ // future: UFix16
	_ // future: UFix32
	PrimitiveStaticTypeUFix64
	_ // future: UFix128
	_ // future: UFix256
	_

	// Storage

	PrimitiveStaticTypePath
	PrimitiveStaticTypeCapability
	PrimitiveStaticTypeStoragePath
	PrimitiveStaticTypeCapabilityPath
	PrimitiveStaticTypePublicPath
	PrimitiveStaticTypePrivatePath
	_
	_
	_
	_
	_
	_
	_
	_
	PrimitiveStaticTypeAuthAccount
	PrimitiveStaticTypePublicAccount
	PrimitiveStaticTypeDeployedContract
	PrimitiveStaticTypeAuthAccountContracts
	PrimitiveStaticTypePublicAccountContracts
	PrimitiveStaticTypeAuthAccountKeys
	PrimitiveStaticTypePublicAccountKeys
	PrimitiveStaticTypeAccountKey

	// !!! *WARNING* !!!
	// ADD NEW TYPES *BEFORE* THIS WARNING.
	// DO *NOT* ADD NEW TYPES AFTER THIS LINE!
	PrimitiveStaticType_Count
)

func (PrimitiveStaticType) isStaticType() {}

func (t PrimitiveStaticType) elementSize() uint {
	switch t {
	case
		PrimitiveStaticTypeAnyStruct,
		PrimitiveStaticTypeAnyResource,
		PrimitiveStaticTypeAny:
		return UnknownElementSize
	case PrimitiveStaticTypeVoid:
		return 3 // see VoidValue ByteLength
	case PrimitiveStaticTypeNever:
		return 1
	case PrimitiveStaticTypeBool:
		return 1
	case PrimitiveStaticTypeAddress:
		return 8 // address length is 8 bytes
	case PrimitiveStaticTypeString,
		PrimitiveStaticTypeCharacter,
		PrimitiveStaticTypeMetaType,
		PrimitiveStaticTypeBlock:
		return UnknownElementSize

	case PrimitiveStaticTypeFixedPoint,
		PrimitiveStaticTypeSignedFixedPoint:
		return 8

	// values of these types may wrap big.Int
	case PrimitiveStaticTypeInt,
		PrimitiveStaticTypeUInt,
		PrimitiveStaticTypeUInt128,
		PrimitiveStaticTypeUInt256,
		PrimitiveStaticTypeInt128,
		PrimitiveStaticTypeInt256,
		PrimitiveStaticTypeInteger,
		PrimitiveStaticTypeSignedInteger,
		PrimitiveStaticTypeNumber,
		PrimitiveStaticTypeSignedNumber:
		return UnknownElementSize

	case PrimitiveStaticTypeInt8,
		PrimitiveStaticTypeInt16,
		PrimitiveStaticTypeInt32,
		PrimitiveStaticTypeInt64,

		PrimitiveStaticTypeUInt8,
		PrimitiveStaticTypeUInt16,
		PrimitiveStaticTypeUInt32,
		PrimitiveStaticTypeUInt64,

		PrimitiveStaticTypeWord8,
		PrimitiveStaticTypeWord16,
		PrimitiveStaticTypeWord32,
		PrimitiveStaticTypeWord64,

		PrimitiveStaticTypeFix64,

		PrimitiveStaticTypeUFix64:
		return 8

	case PrimitiveStaticTypePath,
		PrimitiveStaticTypeCapability,
		PrimitiveStaticTypeStoragePath,
		PrimitiveStaticTypeCapabilityPath,
		PrimitiveStaticTypePublicPath,
		PrimitiveStaticTypePrivatePath,
		PrimitiveStaticTypeAuthAccount,
		PrimitiveStaticTypePublicAccount,
		PrimitiveStaticTypeDeployedContract,
		PrimitiveStaticTypeAuthAccountContracts,
		PrimitiveStaticTypePublicAccountContracts,
		PrimitiveStaticTypeAuthAccountKeys,
		PrimitiveStaticTypePublicAccountKeys,
		PrimitiveStaticTypeAccountKey:
		return UnknownElementSize
	}
	return UnknownElementSize
}

func (i PrimitiveStaticType) SemaType() sema.Type {
	switch i {
	case PrimitiveStaticTypeVoid:
		return sema.VoidType

	case PrimitiveStaticTypeAny:
		return sema.AnyType

	case PrimitiveStaticTypeNever:
		return sema.NeverType

	case PrimitiveStaticTypeAnyStruct:
		return sema.AnyStructType

	case PrimitiveStaticTypeAnyResource:
		return sema.AnyResourceType

	case PrimitiveStaticTypeBool:
		return sema.BoolType

	case PrimitiveStaticTypeAddress:
		return &sema.AddressType{}

	case PrimitiveStaticTypeString:
		return sema.StringType

	case PrimitiveStaticTypeCharacter:
		return sema.CharacterType

	case PrimitiveStaticTypeMetaType:
		return sema.MetaType

	case PrimitiveStaticTypeBlock:
		return sema.BlockType

	// Number

	case PrimitiveStaticTypeNumber:
		return sema.NumberType
	case PrimitiveStaticTypeSignedNumber:
		return sema.SignedNumberType

	// Integer
	case PrimitiveStaticTypeInteger:
		return sema.IntegerType
	case PrimitiveStaticTypeSignedInteger:
		return sema.SignedIntegerType

	// FixedPoint
	case PrimitiveStaticTypeFixedPoint:
		return sema.FixedPointType
	case PrimitiveStaticTypeSignedFixedPoint:
		return sema.SignedFixedPointType

	// Int*
	case PrimitiveStaticTypeInt:
		return sema.IntType
	case PrimitiveStaticTypeInt8:
		return sema.Int8Type
	case PrimitiveStaticTypeInt16:
		return sema.Int16Type
	case PrimitiveStaticTypeInt32:
		return sema.Int32Type
	case PrimitiveStaticTypeInt64:
		return sema.Int64Type
	case PrimitiveStaticTypeInt128:
		return sema.Int128Type
	case PrimitiveStaticTypeInt256:
		return sema.Int256Type

	// UInt*
	case PrimitiveStaticTypeUInt:
		return sema.UIntType
	case PrimitiveStaticTypeUInt8:
		return sema.UInt8Type
	case PrimitiveStaticTypeUInt16:
		return sema.UInt16Type
	case PrimitiveStaticTypeUInt32:
		return sema.UInt32Type
	case PrimitiveStaticTypeUInt64:
		return sema.UInt64Type
	case PrimitiveStaticTypeUInt128:
		return sema.UInt128Type
	case PrimitiveStaticTypeUInt256:
		return sema.UInt256Type

	// Word *

	case PrimitiveStaticTypeWord8:
		return sema.Word8Type
	case PrimitiveStaticTypeWord16:
		return sema.Word16Type
	case PrimitiveStaticTypeWord32:
		return sema.Word32Type
	case PrimitiveStaticTypeWord64:
		return sema.Word64Type

	// Fix*
	case PrimitiveStaticTypeFix64:
		return sema.Fix64Type

	// UFix*
	case PrimitiveStaticTypeUFix64:
		return sema.UFix64Type

	// Storage

	case PrimitiveStaticTypePath:
		return sema.PathType
	case PrimitiveStaticTypeStoragePath:
		return sema.StoragePathType
	case PrimitiveStaticTypeCapabilityPath:
		return sema.CapabilityPathType
	case PrimitiveStaticTypePublicPath:
		return sema.PublicPathType
	case PrimitiveStaticTypePrivatePath:
		return sema.PrivatePathType
	case PrimitiveStaticTypeCapability:
		return &sema.CapabilityType{}
	case PrimitiveStaticTypeAuthAccount:
		return sema.AuthAccountType
	case PrimitiveStaticTypePublicAccount:
		return sema.PublicAccountType
	case PrimitiveStaticTypeDeployedContract:
		return sema.DeployedContractType
	case PrimitiveStaticTypeAuthAccountContracts:
		return sema.AuthAccountContractsType
	case PrimitiveStaticTypePublicAccountContracts:
		return sema.PublicAccountContractsType
	case PrimitiveStaticTypeAuthAccountKeys:
		return sema.AuthAccountKeysType
	case PrimitiveStaticTypePublicAccountKeys:
		return sema.PublicAccountKeysType
	case PrimitiveStaticTypeAccountKey:
		return sema.AccountKeyType
	default:
		panic(errors.NewUnreachableError())
	}
}

// ConvertSemaToPrimitiveStaticType converts a `sema.Type` to a `PrimitiveStaticType`.
//
// Returns `PrimitiveStaticTypeUnknown` if the given type is not a primitive type.
//
func ConvertSemaToPrimitiveStaticType(
	memoryGauge common.MemoryGauge,
	t sema.Type,
) (typ PrimitiveStaticType) {
	switch t {

	// Number

	case sema.NumberType:
		typ = PrimitiveStaticTypeNumber
	case sema.SignedNumberType:
		typ = PrimitiveStaticTypeSignedNumber

	// Integer
	case sema.IntegerType:
		typ = PrimitiveStaticTypeInteger
	case sema.SignedIntegerType:
		typ = PrimitiveStaticTypeSignedInteger

	// FixedPoint
	case sema.FixedPointType:
		typ = PrimitiveStaticTypeFixedPoint
	case sema.SignedFixedPointType:
		typ = PrimitiveStaticTypeSignedFixedPoint

	// Int*
	case sema.IntType:
		typ = PrimitiveStaticTypeInt
	case sema.Int8Type:
		typ = PrimitiveStaticTypeInt8
	case sema.Int16Type:
		typ = PrimitiveStaticTypeInt16
	case sema.Int32Type:
		typ = PrimitiveStaticTypeInt32
	case sema.Int64Type:
		typ = PrimitiveStaticTypeInt64
	case sema.Int128Type:
		typ = PrimitiveStaticTypeInt128
	case sema.Int256Type:
		typ = PrimitiveStaticTypeInt256

	// UInt*
	case sema.UIntType:
		typ = PrimitiveStaticTypeUInt
	case sema.UInt8Type:
		typ = PrimitiveStaticTypeUInt8
	case sema.UInt16Type:
		typ = PrimitiveStaticTypeUInt16
	case sema.UInt32Type:
		typ = PrimitiveStaticTypeUInt32
	case sema.UInt64Type:
		typ = PrimitiveStaticTypeUInt64
	case sema.UInt128Type:
		typ = PrimitiveStaticTypeUInt128
	case sema.UInt256Type:
		typ = PrimitiveStaticTypeUInt256

	// Word*
	case sema.Word8Type:
		typ = PrimitiveStaticTypeWord8
	case sema.Word16Type:
		typ = PrimitiveStaticTypeWord16
	case sema.Word32Type:
		typ = PrimitiveStaticTypeWord32
	case sema.Word64Type:
		typ = PrimitiveStaticTypeWord64

	// Fix*
	case sema.Fix64Type:
		typ = PrimitiveStaticTypeFix64

	// UFix*
	case sema.UFix64Type:
		typ = PrimitiveStaticTypeUFix64

	case sema.PathType:
		typ = PrimitiveStaticTypePath
	case sema.StoragePathType:
		typ = PrimitiveStaticTypeStoragePath
	case sema.CapabilityPathType:
		typ = PrimitiveStaticTypeCapabilityPath
	case sema.PublicPathType:
		typ = PrimitiveStaticTypePublicPath
	case sema.PrivatePathType:
		typ = PrimitiveStaticTypePrivatePath
	case sema.NeverType:
		typ = PrimitiveStaticTypeNever
	case sema.VoidType:
		typ = PrimitiveStaticTypeVoid
	case sema.MetaType:
		typ = PrimitiveStaticTypeMetaType
	case sema.BoolType:
		typ = PrimitiveStaticTypeBool
	case sema.CharacterType:
		typ = PrimitiveStaticTypeCharacter
	case sema.AnyType:
		typ = PrimitiveStaticTypeAny
	case sema.AnyStructType:
		typ = PrimitiveStaticTypeAnyStruct
	case sema.AnyResourceType:
		typ = PrimitiveStaticTypeAnyResource
	case sema.AuthAccountType:
		typ = PrimitiveStaticTypeAuthAccount
	case sema.PublicAccountType:
		typ = PrimitiveStaticTypePublicAccount
	case sema.BlockType:
		typ = PrimitiveStaticTypeBlock
	case sema.DeployedContractType:
		typ = PrimitiveStaticTypeDeployedContract
	case sema.AuthAccountContractsType:
		typ = PrimitiveStaticTypeAuthAccountContracts
	case sema.PublicAccountContractsType:
		typ = PrimitiveStaticTypePublicAccountContracts
	case sema.AuthAccountKeysType:
		typ = PrimitiveStaticTypeAuthAccountKeys
	case sema.PublicAccountKeysType:
		typ = PrimitiveStaticTypePublicAccountKeys
	case sema.AccountKeyType:
		typ = PrimitiveStaticTypeAccountKey
	case sema.StringType:
		typ = PrimitiveStaticTypeString
	}

	switch t.(type) {
	case *sema.AddressType:
		typ = PrimitiveStaticTypeAddress

	// Storage
	case *sema.CapabilityType:
		typ = PrimitiveStaticTypeCapability
	}

	return NewPrimitiveStaticType(memoryGauge, typ) // default is 0 aka PrimitiveStaticTypeUnknown
}
