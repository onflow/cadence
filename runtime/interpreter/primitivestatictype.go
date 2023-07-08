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

package interpreter

import (
	"fmt"
	"strconv"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=PrimitiveStaticType -trimprefix=PrimitiveStaticType

// PrimitiveStaticType

var PrimitiveStaticTypes = _PrimitiveStaticType_map

type PrimitiveStaticType uint

func (t PrimitiveStaticType) Equal(other StaticType) bool {
	otherPrimitiveType, ok := other.(PrimitiveStaticType)
	if !ok {
		return false
	}

	return t == otherPrimitiveType
}

const primitiveStaticTypePrefix = "PrimitiveStaticType"

var primitiveStaticTypeConstantLength = len(primitiveStaticTypePrefix) + 2 // + 2 for parentheses

func (t PrimitiveStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	if str, ok := PrimitiveStaticTypes[t]; ok {
		common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(str)))
		return str
	}

	memoryAmount := primitiveStaticTypeConstantLength + OverEstimateIntStringLength(int(t))
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(memoryAmount))

	rawValueStr := strconv.FormatInt(int64(t), 10)
	return fmt.Sprintf("%s(%s)", primitiveStaticTypePrefix, rawValueStr)
}

func NewPrimitiveStaticType(
	memoryGauge common.MemoryGauge,
	staticType PrimitiveStaticType,
) PrimitiveStaticType {
	common.UseMemory(memoryGauge, common.PrimitiveStaticTypeMemoryUsage)
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
	PrimitiveStaticTypeWord128
	PrimitiveStaticTypeWord256
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
	PrimitiveStaticTypeAuthAccountInbox
	PrimitiveStaticTypeStorageCapabilityController
	PrimitiveStaticTypeAccountCapabilityController
	PrimitiveStaticTypeAuthAccountStorageCapabilities
	PrimitiveStaticTypeAuthAccountAccountCapabilities
	PrimitiveStaticTypeAuthAccountCapabilities
	PrimitiveStaticTypePublicAccountCapabilities
	_
	_
	_
	_
	_
	PrimitiveStaticTypeAccount
	PrimitiveStaticTypeAccountContracts
	PrimitiveStaticTypeAccountKeys
	PrimitiveStaticTypeAccountInbox
	PrimitiveStaticTypeAccountStorageCapabilities
	PrimitiveStaticTypeAccountAccountCapabilities
	PrimitiveStaticTypeAccountCapabilities
	PrimitiveStaticTypeAccountStorage

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
		return uint(len(cborVoidValue))
	case PrimitiveStaticTypeNever:
		return cborTagSize + 1
	case PrimitiveStaticTypeBool:
		return cborTagSize + 1
	case PrimitiveStaticTypeAddress:
		return cborTagSize + 8 // address length is 8 bytes
	case PrimitiveStaticTypeString,
		PrimitiveStaticTypeCharacter,
		PrimitiveStaticTypeMetaType,
		PrimitiveStaticTypeBlock:
		return UnknownElementSize

	case PrimitiveStaticTypeFixedPoint,
		PrimitiveStaticTypeSignedFixedPoint:
		return cborTagSize + 8

	// values of these types may wrap big.Int
	case PrimitiveStaticTypeInt,
		PrimitiveStaticTypeUInt,
		PrimitiveStaticTypeUInt128,
		PrimitiveStaticTypeUInt256,
		PrimitiveStaticTypeInt128,
		PrimitiveStaticTypeInt256,
		PrimitiveStaticTypeWord128,
		PrimitiveStaticTypeWord256,
		PrimitiveStaticTypeInteger,
		PrimitiveStaticTypeSignedInteger,
		PrimitiveStaticTypeNumber,
		PrimitiveStaticTypeSignedNumber:
		return UnknownElementSize

	case PrimitiveStaticTypeInt8,
		PrimitiveStaticTypeUInt8,
		PrimitiveStaticTypeWord8:
		return cborTagSize + 2
	case PrimitiveStaticTypeInt16,
		PrimitiveStaticTypeUInt16,
		PrimitiveStaticTypeWord16:
		return cborTagSize + 3
	case PrimitiveStaticTypeInt32,
		PrimitiveStaticTypeUInt32,
		PrimitiveStaticTypeWord32:
		return cborTagSize + 5
	case PrimitiveStaticTypeInt64,
		PrimitiveStaticTypeUInt64,
		PrimitiveStaticTypeWord64,
		PrimitiveStaticTypeFix64,
		PrimitiveStaticTypeUFix64:
		return cborTagSize + 9

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
		PrimitiveStaticTypeAuthAccountInbox,
		PrimitiveStaticTypeAuthAccountKeys,
		PrimitiveStaticTypePublicAccountKeys,
		PrimitiveStaticTypeAccountKey,
		PrimitiveStaticTypeStorageCapabilityController,
		PrimitiveStaticTypeAccountCapabilityController,
		PrimitiveStaticTypeAuthAccountStorageCapabilities,
		PrimitiveStaticTypeAuthAccountAccountCapabilities,
		PrimitiveStaticTypeAuthAccountCapabilities,
		PrimitiveStaticTypePublicAccountCapabilities:
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
		return sema.TheAddressType

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
	case PrimitiveStaticTypeWord128:
		return sema.Word128Type
	case PrimitiveStaticTypeWord256:
		return sema.Word256Type

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
		return nil
	case PrimitiveStaticTypePublicAccount:
		return nil
	case PrimitiveStaticTypeDeployedContract:
		return sema.DeployedContractType
	case PrimitiveStaticTypeAuthAccountContracts:
		return nil
	case PrimitiveStaticTypePublicAccountContracts:
		return nil
	case PrimitiveStaticTypeAuthAccountKeys:
		return nil
	case PrimitiveStaticTypePublicAccountKeys:
		return nil
	case PrimitiveStaticTypeAccountKey:
		return sema.AccountKeyType
	case PrimitiveStaticTypeAuthAccountInbox:
		return nil
	case PrimitiveStaticTypeStorageCapabilityController:
		return sema.StorageCapabilityControllerType
	case PrimitiveStaticTypeAccountCapabilityController:
		return sema.AccountCapabilityControllerType
	case PrimitiveStaticTypeAuthAccountStorageCapabilities:
		return nil
	case PrimitiveStaticTypeAuthAccountAccountCapabilities:
		return nil
	case PrimitiveStaticTypeAuthAccountCapabilities:
		return nil
	case PrimitiveStaticTypePublicAccountCapabilities:
		return nil

	case PrimitiveStaticTypeAccount:
		return sema.AccountType
	case PrimitiveStaticTypeAccountContracts:
		return sema.Account_ContractsType
	case PrimitiveStaticTypeAccountKeys:
		return sema.Account_KeysType
	case PrimitiveStaticTypeAccountInbox:
		return sema.Account_InboxType
	case PrimitiveStaticTypeAccountStorageCapabilities:
		return sema.Account_StorageCapabilitiesType
	case PrimitiveStaticTypeAccountAccountCapabilities:
		return sema.Account_AccountCapabilitiesType
	case PrimitiveStaticTypeAccountCapabilities:
		return sema.AccountCapabilitiesType
	case PrimitiveStaticTypeAccountStorage:
		return sema.Account_StorageType

	default:
		panic(errors.NewUnexpectedError("missing case for %s", i))
	}
}

// ConvertSemaToPrimitiveStaticType converts a `sema.Type` to a `PrimitiveStaticType`.
//
// Returns `PrimitiveStaticTypeUnknown` if the given type is not a primitive type.
func ConvertSemaToPrimitiveStaticType(
	memoryGauge common.MemoryGauge,
	t sema.Type,
) (typ PrimitiveStaticType) {
	switch t {

	case sema.StringType:
		typ = PrimitiveStaticTypeString

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
	case sema.Word128Type:
		typ = PrimitiveStaticTypeWord128
	case sema.Word256Type:
		typ = PrimitiveStaticTypeWord256

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
	case sema.BlockType:
		typ = PrimitiveStaticTypeBlock
	case sema.DeployedContractType:
		typ = PrimitiveStaticTypeDeployedContract
	case sema.AccountKeyType:
		typ = PrimitiveStaticTypeAccountKey
	case sema.StorageCapabilityControllerType:
		typ = PrimitiveStaticTypeStorageCapabilityController
	case sema.AccountCapabilityControllerType:
		typ = PrimitiveStaticTypeAccountCapabilityController

	case sema.AccountType:
		return PrimitiveStaticTypeAccount
	case sema.Account_ContractsType:
		return PrimitiveStaticTypeAccountContracts
	case sema.Account_KeysType:
		return PrimitiveStaticTypeAccountKeys
	case sema.Account_InboxType:
		return PrimitiveStaticTypeAccountInbox
	case sema.Account_StorageCapabilitiesType:
		return PrimitiveStaticTypeAccountStorageCapabilities
	case sema.Account_AccountCapabilitiesType:
		return PrimitiveStaticTypeAccountAccountCapabilities
	case sema.AccountCapabilitiesType:
		return PrimitiveStaticTypeAccountCapabilities
	case sema.Account_StorageType:
		return PrimitiveStaticTypeAccountStorage
	}

	switch t := t.(type) {
	case *sema.AddressType:
		typ = PrimitiveStaticTypeAddress

	// Storage
	case *sema.CapabilityType:
		// Only convert unparameterized Capability type
		if t.BorrowType == nil {
			typ = PrimitiveStaticTypeCapability
		}
	}

	if typ == PrimitiveStaticTypeUnknown {
		return
	}

	return NewPrimitiveStaticType(memoryGauge, typ) // default is 0 aka PrimitiveStaticTypeUnknown
}
