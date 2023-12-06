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

package ccf

//go:generate go run golang.org/x/tools/cmd/stringer -type=SimpleType

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common/bimap"
)

// Simple type ID is a compact representation of a type
// which doesn't need additional information.

// IMPORTANT:
//
// Don't change existing simple type IDs.
//
// When new simple cadence.Type is added,
// - ADD new ID to the end of existing IDs,
// - ADD new simple cadence.Type and its ID
//   to simpleTypeIDByType *and* typeBySimpleTypeID

type SimpleType uint64

const ( // Cadence simple type IDs
	SimpleTypeBool SimpleType = iota
	SimpleTypeString
	SimpleTypeCharacter
	SimpleTypeAddress
	SimpleTypeInt
	SimpleTypeInt8
	SimpleTypeInt16
	SimpleTypeInt32
	SimpleTypeInt64
	SimpleTypeInt128
	SimpleTypeInt256
	SimpleTypeUInt
	SimpleTypeUInt8
	SimpleTypeUInt16
	SimpleTypeUInt32
	SimpleTypeUInt64
	SimpleTypeUInt128
	SimpleTypeUInt256
	SimpleTypeWord8
	SimpleTypeWord16
	SimpleTypeWord32
	SimpleTypeWord64
	SimpleTypeFix64
	SimpleTypeUFix64
	SimpleTypePath
	SimpleTypeCapabilityPath
	SimpleTypeStoragePath
	SimpleTypePublicPath
	SimpleTypePrivatePath
	_ // DO NOT REUSE: was AuthAccount
	_ // DO NOT REUSE: was PublicAccount
	_ // DO NOT REUSE: was AuthAccountKeys
	_ // DO NOT REUSE: was PublicAccountKeys
	_ // DO NOT REUSE: was AuthAccountContracts
	_ // DO NOT REUSE: was PublicAccountContracts
	SimpleTypeDeployedContract
	_ // DO NOT REUSE: was AccountKey
	SimpleTypeBlock
	SimpleTypeAny
	SimpleTypeAnyStruct
	SimpleTypeAnyResource
	SimpleTypeMetaType
	SimpleTypeNever
	SimpleTypeNumber
	SimpleTypeSignedNumber
	SimpleTypeInteger
	SimpleTypeSignedInteger
	SimpleTypeFixedPoint
	SimpleTypeSignedFixedPoint
	SimpleTypeBytes
	SimpleTypeVoid
	SimpleTypeFunction
	SimpleTypeWord128
	SimpleTypeWord256
	SimpleTypeAnyStructAttachmentType
	SimpleTypeAnyResourceAttachmentType
	SimpleTypeStorageCapabilityController
	SimpleTypeAccountCapabilityController
	SimpleTypeAccount
	SimpleTypeAccount_Contracts
	SimpleTypeAccount_Keys
	SimpleTypeAccount_Inbox
	SimpleTypeAccount_StorageCapabilities
	SimpleTypeAccount_AccountCapabilities
	SimpleTypeAccount_Capabilities
	SimpleTypeAccount_Storage
	SimpleTypeMutate
	SimpleTypeInsert
	SimpleTypeRemove
	SimpleTypeIdentity
	SimpleTypeStorage
	SimpleTypeSaveValue
	SimpleTypeLoadValue
	SimpleTypeCopyValue
	SimpleTypeBorrowValue
	SimpleTypeContracts
	SimpleTypeAddContract
	SimpleTypeUpdateContract
	SimpleTypeRemoveContract
	SimpleTypeKeys
	SimpleTypeAddKey
	SimpleTypeRevokeKey
	SimpleTypeInbox
	SimpleTypePublishInboxCapability
	SimpleTypeUnpublishInboxCapability
	SimpleTypeClaimInboxCapability
	SimpleTypeCapabilities
	SimpleTypeStorageCapabilities
	SimpleTypeAccountCapabilities
	SimpleTypePublishCapability
	SimpleTypeUnpublishCapability
	SimpleTypeGetStorageCapabilityController
	SimpleTypeIssueStorageCapabilityController
	SimpleTypeGetAccountCapabilityController
	SimpleTypeIssueAccountCapabilityController
	SimpleTypeCapabilitiesMapping
	SimpleTypeAccountMapping
	SimpleTypeHashableStruct

	// !!! *WARNING* !!!
	// ADD NEW TYPES *BEFORE* THIS WARNING.
	// DO *NOT* ADD NEW TYPES AFTER THIS LINE!
	SimpleType_Count
)

// NOTE: cadence.FunctionType isn't included in simpleTypeIDByType
// because this function is used by both inline-type and type-value.
// cadence.FunctionType needs to be handled differently when this
// function is used by inline-type and type-value.
func initSimpleTypeIDBiMap() (m *bimap.BiMap[cadence.PrimitiveType, SimpleType]) {
	m = bimap.NewBiMap[cadence.PrimitiveType, SimpleType]()

	m.Insert(cadence.AnyType, SimpleTypeAny)
	m.Insert(cadence.AnyStructType, SimpleTypeAnyStruct)
	m.Insert(cadence.AnyResourceType, SimpleTypeAnyResource)
	m.Insert(cadence.AddressType, SimpleTypeAddress)
	m.Insert(cadence.MetaType, SimpleTypeMetaType)
	m.Insert(cadence.VoidType, SimpleTypeVoid)
	m.Insert(cadence.NeverType, SimpleTypeNever)
	m.Insert(cadence.BoolType, SimpleTypeBool)
	m.Insert(cadence.StringType, SimpleTypeString)
	m.Insert(cadence.CharacterType, SimpleTypeCharacter)
	m.Insert(cadence.HashableStructType, SimpleTypeHashableStruct)

	m.Insert(cadence.NumberType, SimpleTypeNumber)
	m.Insert(cadence.SignedNumberType, SimpleTypeSignedNumber)
	m.Insert(cadence.IntegerType, SimpleTypeInteger)
	m.Insert(cadence.SignedIntegerType, SimpleTypeSignedInteger)
	m.Insert(cadence.FixedPointType, SimpleTypeFixedPoint)
	m.Insert(cadence.SignedFixedPointType, SimpleTypeSignedFixedPoint)

	m.Insert(cadence.IntType, SimpleTypeInt)
	m.Insert(cadence.Int8Type, SimpleTypeInt8)
	m.Insert(cadence.Int16Type, SimpleTypeInt16)
	m.Insert(cadence.Int32Type, SimpleTypeInt32)
	m.Insert(cadence.Int64Type, SimpleTypeInt64)
	m.Insert(cadence.Int128Type, SimpleTypeInt128)
	m.Insert(cadence.Int256Type, SimpleTypeInt256)

	m.Insert(cadence.UIntType, SimpleTypeUInt)
	m.Insert(cadence.UInt8Type, SimpleTypeUInt8)
	m.Insert(cadence.UInt16Type, SimpleTypeUInt16)
	m.Insert(cadence.UInt32Type, SimpleTypeUInt32)
	m.Insert(cadence.UInt64Type, SimpleTypeUInt64)
	m.Insert(cadence.UInt128Type, SimpleTypeUInt128)
	m.Insert(cadence.UInt256Type, SimpleTypeUInt256)

	m.Insert(cadence.Word8Type, SimpleTypeWord8)
	m.Insert(cadence.Word16Type, SimpleTypeWord16)
	m.Insert(cadence.Word32Type, SimpleTypeWord32)
	m.Insert(cadence.Word64Type, SimpleTypeWord64)
	m.Insert(cadence.Word128Type, SimpleTypeWord128)
	m.Insert(cadence.Word256Type, SimpleTypeWord256)
	m.Insert(cadence.Fix64Type, SimpleTypeFix64)
	m.Insert(cadence.UFix64Type, SimpleTypeUFix64)

	m.Insert(cadence.BlockType, SimpleTypeBlock)
	m.Insert(cadence.PathType, SimpleTypePath)
	m.Insert(cadence.CapabilityPathType, SimpleTypeCapabilityPath)
	m.Insert(cadence.StoragePathType, SimpleTypeStoragePath)
	m.Insert(cadence.PublicPathType, SimpleTypePublicPath)
	m.Insert(cadence.PrivatePathType, SimpleTypePrivatePath)
	m.Insert(cadence.DeployedContractType, SimpleTypeDeployedContract)
	m.Insert(cadence.AnyStructAttachmentType, SimpleTypeAnyStructAttachmentType)
	m.Insert(cadence.AnyResourceAttachmentType, SimpleTypeAnyResourceAttachmentType)

	m.Insert(cadence.BlockType, SimpleTypeBlock)
	m.Insert(cadence.PathType, SimpleTypePath)
	m.Insert(cadence.CapabilityPathType, SimpleTypeCapabilityPath)
	m.Insert(cadence.StoragePathType, SimpleTypeStoragePath)
	m.Insert(cadence.PublicPathType, SimpleTypePublicPath)
	m.Insert(cadence.PrivatePathType, SimpleTypePrivatePath)
	m.Insert(cadence.DeployedContractType, SimpleTypeDeployedContract)
	m.Insert(cadence.AnyStructAttachmentType, SimpleTypeAnyStructAttachmentType)
	m.Insert(cadence.AnyResourceAttachmentType, SimpleTypeAnyResourceAttachmentType)

	m.Insert(cadence.StorageCapabilityControllerType, SimpleTypeStorageCapabilityController)
	m.Insert(cadence.AccountCapabilityControllerType, SimpleTypeAccountCapabilityController)
	m.Insert(cadence.AccountType, SimpleTypeAccount)
	m.Insert(cadence.Account_ContractsType, SimpleTypeAccount_Contracts)
	m.Insert(cadence.Account_KeysType, SimpleTypeAccount_Keys)
	m.Insert(cadence.Account_InboxType, SimpleTypeAccount_Inbox)
	m.Insert(cadence.Account_StorageCapabilitiesType, SimpleTypeAccount_StorageCapabilities)
	m.Insert(cadence.Account_AccountCapabilitiesType, SimpleTypeAccount_AccountCapabilities)
	m.Insert(cadence.Account_CapabilitiesType, SimpleTypeAccount_Capabilities)
	m.Insert(cadence.Account_StorageType, SimpleTypeAccount_Storage)

	m.Insert(cadence.MutateType, SimpleTypeMutate)
	m.Insert(cadence.InsertType, SimpleTypeInsert)
	m.Insert(cadence.RemoveType, SimpleTypeRemove)
	m.Insert(cadence.IdentityType, SimpleTypeIdentity)
	m.Insert(cadence.StorageType, SimpleTypeStorage)
	m.Insert(cadence.SaveValueType, SimpleTypeSaveValue)
	m.Insert(cadence.LoadValueType, SimpleTypeLoadValue)
	m.Insert(cadence.CopyValueType, SimpleTypeCopyValue)
	m.Insert(cadence.BorrowValueType, SimpleTypeBorrowValue)
	m.Insert(cadence.ContractsType, SimpleTypeContracts)
	m.Insert(cadence.AddContractType, SimpleTypeAddContract)
	m.Insert(cadence.UpdateContractType, SimpleTypeUpdateContract)
	m.Insert(cadence.RemoveContractType, SimpleTypeRemoveContract)
	m.Insert(cadence.KeysType, SimpleTypeKeys)
	m.Insert(cadence.AddKeyType, SimpleTypeAddKey)
	m.Insert(cadence.RevokeKeyType, SimpleTypeRevokeKey)
	m.Insert(cadence.InboxType, SimpleTypeInbox)
	m.Insert(cadence.PublishInboxCapabilityType, SimpleTypePublishInboxCapability)
	m.Insert(cadence.UnpublishInboxCapabilityType, SimpleTypeUnpublishInboxCapability)
	m.Insert(cadence.ClaimInboxCapabilityType, SimpleTypeClaimInboxCapability)
	m.Insert(cadence.CapabilitiesType, SimpleTypeCapabilities)
	m.Insert(cadence.StorageCapabilitiesType, SimpleTypeStorageCapabilities)
	m.Insert(cadence.AccountCapabilitiesType, SimpleTypeAccountCapabilities)
	m.Insert(cadence.PublishCapabilityType, SimpleTypePublishCapability)
	m.Insert(cadence.UnpublishCapabilityType, SimpleTypeUnpublishCapability)
	m.Insert(cadence.GetStorageCapabilityControllerType, SimpleTypeGetStorageCapabilityController)
	m.Insert(cadence.IssueStorageCapabilityControllerType, SimpleTypeIssueStorageCapabilityController)
	m.Insert(cadence.GetAccountCapabilityControllerType, SimpleTypeGetAccountCapabilityController)
	m.Insert(cadence.IssueAccountCapabilityControllerType, SimpleTypeIssueAccountCapabilityController)
	m.Insert(cadence.CapabilitiesMappingType, SimpleTypeCapabilitiesMapping)
	m.Insert(cadence.AccountMappingType, SimpleTypeAccountMapping)

	return
}

var simpleTypeIDBiMap *bimap.BiMap[cadence.PrimitiveType, SimpleType] = initSimpleTypeIDBiMap()

func simpleTypeIDByType(typ cadence.Type) (SimpleType, bool) {
	switch typ := typ.(type) {
	case cadence.BytesType:
		return SimpleTypeBytes, true
	case cadence.PrimitiveType:
		return simpleTypeIDBiMap.Get(typ)
	}

	return 0, false
}

func typeBySimpleTypeID(simpleTypeID SimpleType) cadence.Type {
	if simpleTypeID == SimpleTypeBytes {
		return cadence.TheBytesType
	}
	if typ, present := simpleTypeIDBiMap.GetInverse(simpleTypeID); present {
		return typ
	}
	return nil
}
