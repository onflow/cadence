/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=PrimitiveStaticType -trimprefix=PrimitiveStaticType

// PrimitiveStaticType

var PrimitiveStaticTypes = _PrimitiveStaticType_map

type PrimitiveStaticType uint

var _ StaticType = PrimitiveStaticType(0)

const primitiveStaticTypePrefix = "PrimitiveStaticType"

var primitiveStaticTypeConstantLength = len(primitiveStaticTypePrefix) + 2 // + 2 for parentheses

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
//
// NOTE: The following types are not primitive types, but CompositeType-s
// - HashAlgorithm
// - SigningAlgorithm
// - AccountKey
// - PublicKey
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
	PrimitiveStaticTypeAnyResourceAttachment
	PrimitiveStaticTypeAnyStructAttachment
	PrimitiveStaticTypeHashableStruct
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
	PrimitiveStaticTypeFixedSizeUnsignedInteger
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
	// Deprecated: PrimitiveStaticTypeAuthAccount only exists for migration purposes.
	PrimitiveStaticTypeAuthAccount
	// Deprecated: PrimitiveStaticTypePublicAccount only exists for migration purposes.
	PrimitiveStaticTypePublicAccount
	PrimitiveStaticTypeDeployedContract
	// Deprecated: PrimitiveStaticTypeAuthAccountContracts only exists for migration purposes.
	PrimitiveStaticTypeAuthAccountContracts
	// Deprecated: PrimitiveStaticTypePublicAccountContracts only exists for migration purposes.
	PrimitiveStaticTypePublicAccountContracts
	// Deprecated: PrimitiveStaticTypeAuthAccountKeys only exists for migration purposes.
	PrimitiveStaticTypeAuthAccountKeys
	// Deprecated: PrimitiveStaticTypePublicAccountKeys only exists for migration purposes.
	PrimitiveStaticTypePublicAccountKeys
	// Deprecated: PrimitiveStaticTypeAccountKey only exists for migration purposes
	PrimitiveStaticTypeAccountKey
	// Deprecated: PrimitiveStaticTypeAuthAccountInbox only exists for migration purposes.
	PrimitiveStaticTypeAuthAccountInbox
	PrimitiveStaticTypeStorageCapabilityController
	PrimitiveStaticTypeAccountCapabilityController
	// Deprecated: PrimitiveStaticTypeAuthAccountStorageCapabilities only exists for migration purposes.
	PrimitiveStaticTypeAuthAccountStorageCapabilities
	// Deprecated: PrimitiveStaticTypeAuthAccountAccountCapabilities only exists for migration purposes.
	PrimitiveStaticTypeAuthAccountAccountCapabilities
	// Deprecated: PrimitiveStaticTypeAuthAccountCapabilities only exists for migration purposes.
	PrimitiveStaticTypeAuthAccountCapabilities
	// Deprecated: PrimitiveStaticTypePublicAccountCapabilities only exists for migration purposes.
	PrimitiveStaticTypePublicAccountCapabilities
	PrimitiveStaticTypeAccount
	PrimitiveStaticTypeAccount_Contracts
	PrimitiveStaticTypeAccount_Keys
	PrimitiveStaticTypeAccount_Inbox
	PrimitiveStaticTypeAccount_StorageCapabilities
	PrimitiveStaticTypeAccount_AccountCapabilities
	PrimitiveStaticTypeAccount_Capabilities
	PrimitiveStaticTypeAccount_Storage
	_
	_
	_
	_
	_
	PrimitiveStaticTypeMutate
	PrimitiveStaticTypeInsert
	PrimitiveStaticTypeRemove
	PrimitiveStaticTypeIdentity
	_
	_
	_
	PrimitiveStaticTypeStorage
	PrimitiveStaticTypeSaveValue
	PrimitiveStaticTypeLoadValue
	PrimitiveStaticTypeCopyValue
	PrimitiveStaticTypeBorrowValue
	PrimitiveStaticTypeContracts
	PrimitiveStaticTypeAddContract
	PrimitiveStaticTypeUpdateContract
	PrimitiveStaticTypeRemoveContract
	PrimitiveStaticTypeKeys
	PrimitiveStaticTypeAddKey
	PrimitiveStaticTypeRevokeKey
	PrimitiveStaticTypeInbox
	PrimitiveStaticTypePublishInboxCapability
	PrimitiveStaticTypeUnpublishInboxCapability
	PrimitiveStaticTypeClaimInboxCapability
	PrimitiveStaticTypeCapabilities
	PrimitiveStaticTypeStorageCapabilities
	PrimitiveStaticTypeAccountCapabilities
	PrimitiveStaticTypePublishCapability
	PrimitiveStaticTypeUnpublishCapability
	PrimitiveStaticTypeGetStorageCapabilityController
	PrimitiveStaticTypeIssueStorageCapabilityController
	PrimitiveStaticTypeGetAccountCapabilityController
	PrimitiveStaticTypeIssueAccountCapabilityController
	PrimitiveStaticTypeCapabilitiesMapping
	PrimitiveStaticTypeAccountMapping

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
		PrimitiveStaticTypeAny,
		PrimitiveStaticTypeAnyStructAttachment,
		PrimitiveStaticTypeAnyResourceAttachment,
		PrimitiveStaticTypeHashableStruct:
		return UnknownElementSize

	case PrimitiveStaticTypeVoid:
		return uint(len(cborVoidValue))

	case PrimitiveStaticTypeNever:
		return values.CBORTagSize + 1

	case PrimitiveStaticTypeBool:
		return values.CBORTagSize + 1

	case PrimitiveStaticTypeAddress:
		return values.CBORTagSize + 8 // address length is 8 bytes

	case PrimitiveStaticTypeString,
		PrimitiveStaticTypeCharacter,
		PrimitiveStaticTypeMetaType,
		PrimitiveStaticTypeBlock:
		return UnknownElementSize

	case PrimitiveStaticTypeFixedPoint,
		PrimitiveStaticTypeSignedFixedPoint:
		return values.CBORTagSize + 8

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
		PrimitiveStaticTypeFixedSizeUnsignedInteger,
		PrimitiveStaticTypeNumber,
		PrimitiveStaticTypeSignedNumber:
		return UnknownElementSize

	case PrimitiveStaticTypeInt8,
		PrimitiveStaticTypeUInt8,
		PrimitiveStaticTypeWord8:
		return values.CBORTagSize + 2

	case PrimitiveStaticTypeInt16,
		PrimitiveStaticTypeUInt16,
		PrimitiveStaticTypeWord16:
		return values.CBORTagSize + 3

	case PrimitiveStaticTypeInt32,
		PrimitiveStaticTypeUInt32,
		PrimitiveStaticTypeWord32:
		return values.CBORTagSize + 5

	case PrimitiveStaticTypeInt64,
		PrimitiveStaticTypeUInt64,
		PrimitiveStaticTypeWord64,
		PrimitiveStaticTypeFix64,
		PrimitiveStaticTypeUFix64:
		return values.CBORTagSize + 9

	case PrimitiveStaticTypePath,
		PrimitiveStaticTypeCapability,
		PrimitiveStaticTypeStoragePath,
		PrimitiveStaticTypeCapabilityPath,
		PrimitiveStaticTypePublicPath,
		PrimitiveStaticTypePrivatePath,
		PrimitiveStaticTypeDeployedContract,
		PrimitiveStaticTypeStorageCapabilityController,
		PrimitiveStaticTypeAccountCapabilityController,
		PrimitiveStaticTypeAccount,
		PrimitiveStaticTypeAccount_Contracts,
		PrimitiveStaticTypeAccount_Keys,
		PrimitiveStaticTypeAccount_Inbox,
		PrimitiveStaticTypeAccount_StorageCapabilities,
		PrimitiveStaticTypeAccount_AccountCapabilities,
		PrimitiveStaticTypeAccount_Capabilities,
		PrimitiveStaticTypeAccount_Storage,
		PrimitiveStaticTypeMutate,
		PrimitiveStaticTypeInsert,
		PrimitiveStaticTypeRemove,
		PrimitiveStaticTypeIdentity,
		PrimitiveStaticTypeStorage,
		PrimitiveStaticTypeSaveValue,
		PrimitiveStaticTypeLoadValue,
		PrimitiveStaticTypeCopyValue,
		PrimitiveStaticTypeBorrowValue,
		PrimitiveStaticTypeContracts,
		PrimitiveStaticTypeAddContract,
		PrimitiveStaticTypeUpdateContract,
		PrimitiveStaticTypeRemoveContract,
		PrimitiveStaticTypeKeys,
		PrimitiveStaticTypeAddKey,
		PrimitiveStaticTypeRevokeKey,
		PrimitiveStaticTypeInbox,
		PrimitiveStaticTypePublishInboxCapability,
		PrimitiveStaticTypeUnpublishInboxCapability,
		PrimitiveStaticTypeClaimInboxCapability,
		PrimitiveStaticTypeCapabilities,
		PrimitiveStaticTypeStorageCapabilities,
		PrimitiveStaticTypeAccountCapabilities,
		PrimitiveStaticTypePublishCapability,
		PrimitiveStaticTypeUnpublishCapability,
		PrimitiveStaticTypeGetStorageCapabilityController,
		PrimitiveStaticTypeIssueStorageCapabilityController,
		PrimitiveStaticTypeGetAccountCapabilityController,
		PrimitiveStaticTypeIssueAccountCapabilityController,
		PrimitiveStaticTypeCapabilitiesMapping,
		PrimitiveStaticTypeAccountMapping:
		return UnknownElementSize

	case PrimitiveStaticTypeAuthAccount,
		PrimitiveStaticTypePublicAccount,
		PrimitiveStaticTypeAuthAccountContracts,
		PrimitiveStaticTypePublicAccountContracts,
		PrimitiveStaticTypeAuthAccountKeys,
		PrimitiveStaticTypePublicAccountKeys,
		PrimitiveStaticTypeAuthAccountInbox,
		PrimitiveStaticTypeAuthAccountStorageCapabilities,
		PrimitiveStaticTypeAuthAccountAccountCapabilities,
		PrimitiveStaticTypeAuthAccountCapabilities,
		PrimitiveStaticTypePublicAccountCapabilities,
		PrimitiveStaticTypeAccountKey:
		// These types are deprecated, and only exist for migration purposes
		return UnknownElementSize

	case PrimitiveStaticTypeUnknown:
	case PrimitiveStaticType_Count:
	}

	panic(errors.NewUnexpectedError("missing case for %s", t))
}

func (t PrimitiveStaticType) Equal(other StaticType) bool {
	otherPrimitiveType, ok := other.(PrimitiveStaticType)
	if !ok {
		return false
	}

	return t == otherPrimitiveType
}

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

func (t PrimitiveStaticType) ID() TypeID {

	// Handle deprecated types specially, because they do not have a sema type equivalent anymore
	switch t {
	case PrimitiveStaticTypeAuthAccount: //nolint:staticcheck
		return "AuthAccount"
	case PrimitiveStaticTypePublicAccount: //nolint:staticcheck
		return "PublicAccount"
	case PrimitiveStaticTypeAuthAccountContracts: //nolint:staticcheck
		return "AuthAccount.Contracts"
	case PrimitiveStaticTypePublicAccountContracts: //nolint:staticcheck
		return "PublicAccount.Contracts"
	case PrimitiveStaticTypeAuthAccountKeys: //nolint:staticcheck
		return "AuthAccount.Keys"
	case PrimitiveStaticTypePublicAccountKeys: //nolint:staticcheck
		return "PublicAccount.Keys"
	case PrimitiveStaticTypeAuthAccountInbox: //nolint:staticcheck
		return "AuthAccount.Inbox"
	case PrimitiveStaticTypeAuthAccountStorageCapabilities: //nolint:staticcheck
		return "AuthAccount.StorageCapabilities"
	case PrimitiveStaticTypeAuthAccountAccountCapabilities: //nolint:staticcheck
		return "AuthAccount.AccountCapabilities"
	case PrimitiveStaticTypeAuthAccountCapabilities: //nolint:staticcheck
		return "AuthAccount.Capabilities"
	case PrimitiveStaticTypePublicAccountCapabilities: //nolint:staticcheck
		return "PublicAccount.Capabilities"
	case PrimitiveStaticTypeAccountKey: //nolint:staticcheck
		return "AccountKey"
	}

	return t.SemaType().ID()
}

func (t PrimitiveStaticType) SemaType() sema.Type {
	switch t {
	case PrimitiveStaticTypeVoid:
		return sema.VoidType

	case PrimitiveStaticTypeAny:
		return sema.AnyType

	case PrimitiveStaticTypeNever:
		return sema.NeverType

	case PrimitiveStaticTypeAnyStruct:
		return sema.AnyStructType

	case PrimitiveStaticTypeHashableStruct:
		return sema.HashableStructType

	case PrimitiveStaticTypeAnyResource:
		return sema.AnyResourceType

	case PrimitiveStaticTypeAnyResourceAttachment:
		return sema.AnyResourceAttachmentType

	case PrimitiveStaticTypeAnyStructAttachment:
		return sema.AnyStructAttachmentType

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
	case PrimitiveStaticTypeFixedSizeUnsignedInteger:
		return sema.FixedSizeUnsignedIntegerType

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
	case PrimitiveStaticTypeDeployedContract:
		return sema.DeployedContractType
	case PrimitiveStaticTypeStorageCapabilityController:
		return sema.StorageCapabilityControllerType
	case PrimitiveStaticTypeAccountCapabilityController:
		return sema.AccountCapabilityControllerType

	case PrimitiveStaticTypeAccount:
		return sema.AccountType
	case PrimitiveStaticTypeAccount_Contracts:
		return sema.Account_ContractsType
	case PrimitiveStaticTypeAccount_Keys:
		return sema.Account_KeysType
	case PrimitiveStaticTypeAccount_Inbox:
		return sema.Account_InboxType
	case PrimitiveStaticTypeAccount_StorageCapabilities:
		return sema.Account_StorageCapabilitiesType
	case PrimitiveStaticTypeAccount_AccountCapabilities:
		return sema.Account_AccountCapabilitiesType
	case PrimitiveStaticTypeAccount_Capabilities:
		return sema.Account_CapabilitiesType
	case PrimitiveStaticTypeAccount_Storage:
		return sema.Account_StorageType

	case PrimitiveStaticTypeMutate:
		return sema.MutateType
	case PrimitiveStaticTypeInsert:
		return sema.InsertType
	case PrimitiveStaticTypeRemove:
		return sema.RemoveType
	case PrimitiveStaticTypeIdentity:
		return sema.IdentityType

	case PrimitiveStaticTypeStorage:
		return sema.StorageType
	case PrimitiveStaticTypeSaveValue:
		return sema.SaveValueType
	case PrimitiveStaticTypeLoadValue:
		return sema.LoadValueType
	case PrimitiveStaticTypeCopyValue:
		return sema.CopyValueType
	case PrimitiveStaticTypeBorrowValue:
		return sema.BorrowValueType
	case PrimitiveStaticTypeContracts:
		return sema.ContractsType
	case PrimitiveStaticTypeAddContract:
		return sema.AddContractType
	case PrimitiveStaticTypeUpdateContract:
		return sema.UpdateContractType
	case PrimitiveStaticTypeRemoveContract:
		return sema.RemoveContractType
	case PrimitiveStaticTypeKeys:
		return sema.KeysType
	case PrimitiveStaticTypeAddKey:
		return sema.AddKeyType
	case PrimitiveStaticTypeRevokeKey:
		return sema.RevokeKeyType
	case PrimitiveStaticTypeInbox:
		return sema.InboxType
	case PrimitiveStaticTypePublishInboxCapability:
		return sema.PublishInboxCapabilityType
	case PrimitiveStaticTypeUnpublishInboxCapability:
		return sema.UnpublishInboxCapabilityType
	case PrimitiveStaticTypeClaimInboxCapability:
		return sema.ClaimInboxCapabilityType
	case PrimitiveStaticTypeCapabilities:
		return sema.CapabilitiesType
	case PrimitiveStaticTypeStorageCapabilities:
		return sema.StorageCapabilitiesType
	case PrimitiveStaticTypeAccountCapabilities:
		return sema.AccountCapabilitiesType
	case PrimitiveStaticTypePublishCapability:
		return sema.PublishCapabilityType
	case PrimitiveStaticTypeUnpublishCapability:
		return sema.UnpublishCapabilityType
	case PrimitiveStaticTypeGetStorageCapabilityController:
		return sema.GetStorageCapabilityControllerType
	case PrimitiveStaticTypeIssueStorageCapabilityController:
		return sema.IssueStorageCapabilityControllerType
	case PrimitiveStaticTypeGetAccountCapabilityController:
		return sema.GetAccountCapabilityControllerType
	case PrimitiveStaticTypeIssueAccountCapabilityController:
		return sema.IssueAccountCapabilityControllerType

	case PrimitiveStaticTypeCapabilitiesMapping:
		return sema.CapabilitiesMappingType
	case PrimitiveStaticTypeAccountMapping:
		return sema.AccountMappingType

	case PrimitiveStaticTypeAuthAccount: //nolint:staticcheck
		// deprecated, but needed for migration purposes
		return sema.FullyEntitledAccountReferenceType
	case PrimitiveStaticTypePublicAccount: //nolint:staticcheck
		// deprecated, but needed for migration purposes
		return sema.AccountReferenceType

	case PrimitiveStaticTypeAuthAccountContracts:
	case PrimitiveStaticTypePublicAccountContracts:
	case PrimitiveStaticTypeAuthAccountKeys:
	case PrimitiveStaticTypePublicAccountKeys:
	case PrimitiveStaticTypeAccountKey:
	case PrimitiveStaticTypeAuthAccountInbox:
	case PrimitiveStaticTypeAuthAccountStorageCapabilities:
	case PrimitiveStaticTypeAuthAccountAccountCapabilities:
	case PrimitiveStaticTypeAuthAccountCapabilities:
	case PrimitiveStaticTypePublicAccountCapabilities:
		panic(errors.NewUnexpectedError("cannot convert deprecated type %s", t))

	case PrimitiveStaticTypeUnknown:
	case PrimitiveStaticType_Count:
	}

	if t.IsDeprecated() {
		panic(errors.NewUnexpectedError("cannot convert deprecated type %s", t))
	}

	panic(errors.NewUnexpectedError("missing case for %s", t))
}

func (t PrimitiveStaticType) IsDefined() bool {
	_, ok := _PrimitiveStaticType_map[t]
	return ok
}

// Deprecated: IsDeprecated only exists for migration purposes.
func (t PrimitiveStaticType) IsDeprecated() bool {
	switch t {
	case PrimitiveStaticTypeAuthAccount, //nolint:staticcheck
		PrimitiveStaticTypePublicAccount,                  //nolint:staticcheck
		PrimitiveStaticTypeAuthAccountContracts,           //nolint:staticcheck
		PrimitiveStaticTypePublicAccountContracts,         //nolint:staticcheck
		PrimitiveStaticTypeAuthAccountKeys,                //nolint:staticcheck
		PrimitiveStaticTypePublicAccountKeys,              //nolint:staticcheck
		PrimitiveStaticTypeAuthAccountInbox,               //nolint:staticcheck
		PrimitiveStaticTypeAuthAccountStorageCapabilities, //nolint:staticcheck
		PrimitiveStaticTypeAuthAccountAccountCapabilities, //nolint:staticcheck
		PrimitiveStaticTypeAuthAccountCapabilities,        //nolint:staticcheck
		PrimitiveStaticTypePublicAccountCapabilities,      //nolint:staticcheck
		PrimitiveStaticTypeAccountKey:                     //nolint:staticcheck
		return true
	}

	return false
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
	case sema.FixedSizeUnsignedIntegerType:
		typ = PrimitiveStaticTypeFixedSizeUnsignedInteger

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
	case sema.HashableStructType:
		typ = PrimitiveStaticTypeHashableStruct
	case sema.AnyResourceType:
		typ = PrimitiveStaticTypeAnyResource
	case sema.AnyStructAttachmentType:
		typ = PrimitiveStaticTypeAnyStructAttachment
	case sema.AnyResourceAttachmentType:
		typ = PrimitiveStaticTypeAnyResourceAttachment
	case sema.BlockType:
		typ = PrimitiveStaticTypeBlock
	case sema.DeployedContractType:
		typ = PrimitiveStaticTypeDeployedContract
	case sema.StorageCapabilityControllerType:
		typ = PrimitiveStaticTypeStorageCapabilityController
	case sema.AccountCapabilityControllerType:
		typ = PrimitiveStaticTypeAccountCapabilityController

	case sema.AccountType:
		typ = PrimitiveStaticTypeAccount
	case sema.Account_ContractsType:
		typ = PrimitiveStaticTypeAccount_Contracts
	case sema.Account_KeysType:
		typ = PrimitiveStaticTypeAccount_Keys
	case sema.Account_InboxType:
		typ = PrimitiveStaticTypeAccount_Inbox
	case sema.Account_StorageCapabilitiesType:
		typ = PrimitiveStaticTypeAccount_StorageCapabilities
	case sema.Account_AccountCapabilitiesType:
		typ = PrimitiveStaticTypeAccount_AccountCapabilities
	case sema.Account_CapabilitiesType:
		typ = PrimitiveStaticTypeAccount_Capabilities
	case sema.Account_StorageType:
		typ = PrimitiveStaticTypeAccount_Storage

	case sema.MutateType:
		typ = PrimitiveStaticTypeMutate
	case sema.InsertType:
		typ = PrimitiveStaticTypeInsert
	case sema.RemoveType:
		typ = PrimitiveStaticTypeRemove
	case sema.IdentityType:
		typ = PrimitiveStaticTypeIdentity

	case sema.StorageType:
		typ = PrimitiveStaticTypeStorage
	case sema.SaveValueType:
		typ = PrimitiveStaticTypeSaveValue
	case sema.LoadValueType:
		typ = PrimitiveStaticTypeLoadValue
	case sema.CopyValueType:
		typ = PrimitiveStaticTypeCopyValue
	case sema.BorrowValueType:
		typ = PrimitiveStaticTypeBorrowValue
	case sema.ContractsType:
		typ = PrimitiveStaticTypeContracts
	case sema.AddContractType:
		typ = PrimitiveStaticTypeAddContract
	case sema.UpdateContractType:
		typ = PrimitiveStaticTypeUpdateContract
	case sema.RemoveContractType:
		typ = PrimitiveStaticTypeRemoveContract
	case sema.KeysType:
		typ = PrimitiveStaticTypeKeys
	case sema.AddKeyType:
		typ = PrimitiveStaticTypeAddKey
	case sema.RevokeKeyType:
		typ = PrimitiveStaticTypeRevokeKey
	case sema.InboxType:
		typ = PrimitiveStaticTypeInbox
	case sema.PublishInboxCapabilityType:
		typ = PrimitiveStaticTypePublishInboxCapability
	case sema.UnpublishInboxCapabilityType:
		typ = PrimitiveStaticTypeUnpublishInboxCapability
	case sema.ClaimInboxCapabilityType:
		typ = PrimitiveStaticTypeClaimInboxCapability
	case sema.CapabilitiesType:
		typ = PrimitiveStaticTypeCapabilities
	case sema.StorageCapabilitiesType:
		typ = PrimitiveStaticTypeStorageCapabilities
	case sema.AccountCapabilitiesType:
		typ = PrimitiveStaticTypeAccountCapabilities
	case sema.PublishCapabilityType:
		typ = PrimitiveStaticTypePublishCapability
	case sema.UnpublishCapabilityType:
		typ = PrimitiveStaticTypeUnpublishCapability
	case sema.GetStorageCapabilityControllerType:
		typ = PrimitiveStaticTypeGetStorageCapabilityController
	case sema.IssueStorageCapabilityControllerType:
		typ = PrimitiveStaticTypeIssueStorageCapabilityController
	case sema.GetAccountCapabilityControllerType:
		typ = PrimitiveStaticTypeGetAccountCapabilityController
	case sema.IssueAccountCapabilityControllerType:
		typ = PrimitiveStaticTypeIssueAccountCapabilityController

	case sema.CapabilitiesMappingType:
		typ = PrimitiveStaticTypeCapabilitiesMapping
	case sema.AccountMappingType:
		typ = PrimitiveStaticTypeAccountMapping
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
		// default is 0 aka PrimitiveStaticTypeUnknown
		return
	}

	return NewPrimitiveStaticType(memoryGauge, typ)
}

var primitiveStaticTypesByTypeID = map[TypeID]PrimitiveStaticType{}

func init() {
	// Check all defined primitive static types,
	// and construct a type ID to primitive static type mapping
	for ty := PrimitiveStaticTypeUnknown + 1; ty < PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() {
			continue
		}

		_ = ty.elementSize()

		primitiveStaticTypesByTypeID[ty.ID()] = ty
	}
}

func PrimitiveStaticTypeFromTypeID(typeID TypeID) PrimitiveStaticType {
	ty, ok := primitiveStaticTypesByTypeID[typeID]
	if !ok {
		return PrimitiveStaticTypeUnknown
	}
	return ty
}
