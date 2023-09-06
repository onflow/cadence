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

import "github.com/onflow/cadence"

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

	// !!! *WARNING* !!!
	// ADD NEW TYPES *BEFORE* THIS WARNING.
	// DO *NOT* ADD NEW TYPES AFTER THIS LINE!
	SimpleType_Count
)

// NOTE: cadence.FunctionType isn't included in simpleTypeIDByType
// because this function is used by both inline-type and type-value.
// cadence.FunctionType needs to be handled differently when this
// function is used by inline-type and type-value.
func simpleTypeIDByType(typ cadence.Type) (SimpleType, bool) {

	switch typ {
	case cadence.AnyType:
		return SimpleTypeAny, true
	case cadence.AnyStructType:
		return SimpleTypeAnyStruct, true
	case cadence.AnyResourceType:
		return SimpleTypeAnyResource, true
	case cadence.AddressType:
		return SimpleTypeAddress, true
	case cadence.MetaType:
		return SimpleTypeMetaType, true
	case cadence.VoidType:
		return SimpleTypeVoid, true
	case cadence.NeverType:
		return SimpleTypeNever, true
	case cadence.BoolType:
		return SimpleTypeBool, true
	case cadence.StringType:
		return SimpleTypeString, true
	case cadence.CharacterType:
		return SimpleTypeCharacter, true
	case cadence.NumberType:
		return SimpleTypeNumber, true
	case cadence.SignedNumberType:
		return SimpleTypeSignedNumber, true
	case cadence.IntegerType:
		return SimpleTypeInteger, true
	case cadence.SignedIntegerType:
		return SimpleTypeSignedInteger, true
	case cadence.FixedPointType:
		return SimpleTypeFixedPoint, true
	case cadence.SignedFixedPointType:
		return SimpleTypeSignedFixedPoint, true
	case cadence.IntType:
		return SimpleTypeInt, true
	case cadence.Int8Type:
		return SimpleTypeInt8, true
	case cadence.Int16Type:
		return SimpleTypeInt16, true
	case cadence.Int32Type:
		return SimpleTypeInt32, true
	case cadence.Int64Type:
		return SimpleTypeInt64, true
	case cadence.Int128Type:
		return SimpleTypeInt128, true
	case cadence.Int256Type:
		return SimpleTypeInt256, true
	case cadence.UIntType:
		return SimpleTypeUInt, true
	case cadence.UInt8Type:
		return SimpleTypeUInt8, true
	case cadence.UInt16Type:
		return SimpleTypeUInt16, true
	case cadence.UInt32Type:
		return SimpleTypeUInt32, true
	case cadence.UInt64Type:
		return SimpleTypeUInt64, true
	case cadence.UInt128Type:
		return SimpleTypeUInt128, true
	case cadence.UInt256Type:
		return SimpleTypeUInt256, true
	case cadence.Word8Type:
		return SimpleTypeWord8, true
	case cadence.Word16Type:
		return SimpleTypeWord16, true
	case cadence.Word32Type:
		return SimpleTypeWord32, true
	case cadence.Word64Type:
		return SimpleTypeWord64, true
	case cadence.Word128Type:
		return SimpleTypeWord128, true
	case cadence.Word256Type:
		return SimpleTypeWord256, true
	case cadence.Fix64Type:
		return SimpleTypeFix64, true
	case cadence.UFix64Type:
		return SimpleTypeUFix64, true
	case cadence.BlockType:
		return SimpleTypeBlock, true
	case cadence.PathType:
		return SimpleTypePath, true
	case cadence.CapabilityPathType:
		return SimpleTypeCapabilityPath, true
	case cadence.StoragePathType:
		return SimpleTypeStoragePath, true
	case cadence.PublicPathType:
		return SimpleTypePublicPath, true
	case cadence.PrivatePathType:
		return SimpleTypePrivatePath, true
	case cadence.DeployedContractType:
		return SimpleTypeDeployedContract, true
	case cadence.AnyStructAttachmentType:
		return SimpleTypeAnyStructAttachmentType, true
	case cadence.AnyResourceAttachmentType:
		return SimpleTypeAnyResourceAttachmentType, true
	case cadence.StorageCapabilityControllerType:
		return SimpleTypeStorageCapabilityController, true
	case cadence.AccountCapabilityControllerType:
		return SimpleTypeAccountCapabilityController, true
	case cadence.AccountType:
		return SimpleTypeAccount, true
	case cadence.Account_ContractsType:
		return SimpleTypeAccount_Contracts, true
	case cadence.Account_KeysType:
		return SimpleTypeAccount_Keys, true
	case cadence.Account_InboxType:
		return SimpleTypeAccount_Inbox, true
	case cadence.Account_StorageCapabilitiesType:
		return SimpleTypeAccount_StorageCapabilities, true
	case cadence.Account_AccountCapabilitiesType:
		return SimpleTypeAccount_AccountCapabilities, true
	case cadence.Account_CapabilitiesType:
		return SimpleTypeAccount_Capabilities, true
	case cadence.Account_StorageType:
		return SimpleTypeAccount_Storage, true
	case cadence.MutateType:
		return SimpleTypeMutate, true
	case cadence.InsertType:
		return SimpleTypeInsert, true
	case cadence.RemoveType:
		return SimpleTypeRemove, true
	case cadence.IdentityType:
		return SimpleTypeIdentity, true
	case cadence.StorageType:
		return SimpleTypeStorage, true
	case cadence.SaveValueType:
		return SimpleTypeSaveValue, true
	case cadence.LoadValueType:
		return SimpleTypeLoadValue, true
	case cadence.CopyValueType:
		return SimpleTypeCopyValue, true
	case cadence.BorrowValueType:
		return SimpleTypeBorrowValue, true
	case cadence.ContractsType:
		return SimpleTypeContracts, true
	case cadence.AddContractType:
		return SimpleTypeAddContract, true
	case cadence.UpdateContractType:
		return SimpleTypeUpdateContract, true
	case cadence.RemoveContractType:
		return SimpleTypeRemoveContract, true
	case cadence.KeysType:
		return SimpleTypeKeys, true
	case cadence.AddKeyType:
		return SimpleTypeAddKey, true
	case cadence.RevokeKeyType:
		return SimpleTypeRevokeKey, true
	case cadence.InboxType:
		return SimpleTypeInbox, true
	case cadence.PublishInboxCapabilityType:
		return SimpleTypePublishInboxCapability, true
	case cadence.UnpublishInboxCapabilityType:
		return SimpleTypeUnpublishInboxCapability, true
	case cadence.ClaimInboxCapabilityType:
		return SimpleTypeClaimInboxCapability, true
	case cadence.CapabilitiesType:
		return SimpleTypeCapabilities, true
	case cadence.StorageCapabilitiesType:
		return SimpleTypeStorageCapabilities, true
	case cadence.AccountCapabilitiesType:
		return SimpleTypeAccountCapabilities, true
	case cadence.PublishCapabilityType:
		return SimpleTypePublishCapability, true
	case cadence.UnpublishCapabilityType:
		return SimpleTypeUnpublishCapability, true
	case cadence.GetStorageCapabilityControllerType:
		return SimpleTypeGetStorageCapabilityController, true
	case cadence.IssueStorageCapabilityControllerType:
		return SimpleTypeIssueStorageCapabilityController, true
	case cadence.GetAccountCapabilityControllerType:
		return SimpleTypeGetAccountCapabilityController, true
	case cadence.IssueAccountCapabilityControllerType:
		return SimpleTypeIssueAccountCapabilityController, true
	case cadence.CapabilitiesMappingType:
		return SimpleTypeCapabilitiesMapping, true
	case cadence.AccountMappingType:
		return SimpleTypeAccountMapping, true

	}

	switch typ.(type) {
	case cadence.BytesType:
		return SimpleTypeBytes, true
	}

	return 0, false
}

func typeBySimpleTypeID(simpleTypeID SimpleType) cadence.Type {
	switch simpleTypeID {
	case SimpleTypeBool:
		return cadence.BoolType
	case SimpleTypeString:
		return cadence.StringType
	case SimpleTypeCharacter:
		return cadence.CharacterType
	case SimpleTypeAddress:
		return cadence.AddressType
	case SimpleTypeInt:
		return cadence.IntType
	case SimpleTypeInt8:
		return cadence.Int8Type
	case SimpleTypeInt16:
		return cadence.Int16Type
	case SimpleTypeInt32:
		return cadence.Int32Type
	case SimpleTypeInt64:
		return cadence.Int64Type
	case SimpleTypeInt128:
		return cadence.Int128Type
	case SimpleTypeInt256:
		return cadence.Int256Type
	case SimpleTypeUInt:
		return cadence.UIntType
	case SimpleTypeUInt8:
		return cadence.UInt8Type
	case SimpleTypeUInt16:
		return cadence.UInt16Type
	case SimpleTypeUInt32:
		return cadence.UInt32Type
	case SimpleTypeUInt64:
		return cadence.UInt64Type
	case SimpleTypeUInt128:
		return cadence.UInt128Type
	case SimpleTypeUInt256:
		return cadence.UInt256Type
	case SimpleTypeWord8:
		return cadence.Word8Type
	case SimpleTypeWord16:
		return cadence.Word16Type
	case SimpleTypeWord32:
		return cadence.Word32Type
	case SimpleTypeWord64:
		return cadence.Word64Type
	case SimpleTypeWord128:
		return cadence.Word128Type
	case SimpleTypeWord256:
		return cadence.Word256Type
	case SimpleTypeFix64:
		return cadence.Fix64Type
	case SimpleTypeUFix64:
		return cadence.UFix64Type
	case SimpleTypePath:
		return cadence.PathType
	case SimpleTypeCapabilityPath:
		return cadence.CapabilityPathType
	case SimpleTypeStoragePath:
		return cadence.StoragePathType
	case SimpleTypePublicPath:
		return cadence.PublicPathType
	case SimpleTypePrivatePath:
		return cadence.PrivatePathType
	case SimpleTypeDeployedContract:
		return cadence.DeployedContractType
	case SimpleTypeBlock:
		return cadence.BlockType
	case SimpleTypeAny:
		return cadence.AnyType
	case SimpleTypeAnyStruct:
		return cadence.AnyStructType
	case SimpleTypeAnyResource:
		return cadence.AnyResourceType
	case SimpleTypeMetaType:
		return cadence.MetaType
	case SimpleTypeNever:
		return cadence.NeverType
	case SimpleTypeNumber:
		return cadence.NumberType
	case SimpleTypeSignedNumber:
		return cadence.SignedNumberType
	case SimpleTypeInteger:
		return cadence.IntegerType
	case SimpleTypeSignedInteger:
		return cadence.SignedIntegerType
	case SimpleTypeFixedPoint:
		return cadence.FixedPointType
	case SimpleTypeSignedFixedPoint:
		return cadence.SignedFixedPointType
	case SimpleTypeBytes:
		return cadence.TheBytesType
	case SimpleTypeVoid:
		return cadence.VoidType
	case SimpleTypeAnyStructAttachmentType:
		return cadence.AnyStructAttachmentType
	case SimpleTypeAnyResourceAttachmentType:
		return cadence.AnyResourceAttachmentType
	case SimpleTypeStorageCapabilityController:
		return cadence.StorageCapabilityControllerType
	case SimpleTypeAccountCapabilityController:
		return cadence.AccountCapabilityControllerType
	case SimpleTypeAccount:
		return cadence.AccountType
	case SimpleTypeAccount_Contracts:
		return cadence.Account_ContractsType
	case SimpleTypeAccount_Keys:
		return cadence.Account_KeysType
	case SimpleTypeAccount_Inbox:
		return cadence.Account_InboxType
	case SimpleTypeAccount_StorageCapabilities:
		return cadence.Account_StorageCapabilitiesType
	case SimpleTypeAccount_AccountCapabilities:
		return cadence.Account_AccountCapabilitiesType
	case SimpleTypeAccount_Capabilities:
		return cadence.Account_CapabilitiesType
	case SimpleTypeAccount_Storage:
		return cadence.Account_StorageType
	case SimpleTypeMutate:
		return cadence.MutateType
	case SimpleTypeInsert:
		return cadence.InsertType
	case SimpleTypeRemove:
		return cadence.RemoveType
	case SimpleTypeIdentity:
		return cadence.IdentityType
	case SimpleTypeStorage:
		return cadence.StorageType
	case SimpleTypeSaveValue:
		return cadence.SaveValueType
	case SimpleTypeLoadValue:
		return cadence.LoadValueType
	case SimpleTypeCopyValue:
		return cadence.CopyValueType
	case SimpleTypeBorrowValue:
		return cadence.BorrowValueType
	case SimpleTypeContracts:
		return cadence.ContractsType
	case SimpleTypeAddContract:
		return cadence.AddContractType
	case SimpleTypeUpdateContract:
		return cadence.UpdateContractType
	case SimpleTypeRemoveContract:
		return cadence.RemoveContractType
	case SimpleTypeKeys:
		return cadence.KeysType
	case SimpleTypeAddKey:
		return cadence.AddKeyType
	case SimpleTypeRevokeKey:
		return cadence.RevokeKeyType
	case SimpleTypeInbox:
		return cadence.InboxType
	case SimpleTypePublishInboxCapability:
		return cadence.PublishInboxCapabilityType
	case SimpleTypeUnpublishInboxCapability:
		return cadence.UnpublishInboxCapabilityType
	case SimpleTypeClaimInboxCapability:
		return cadence.ClaimInboxCapabilityType
	case SimpleTypeCapabilities:
		return cadence.CapabilitiesType
	case SimpleTypeStorageCapabilities:
		return cadence.StorageCapabilitiesType
	case SimpleTypeAccountCapabilities:
		return cadence.AccountCapabilitiesType
	case SimpleTypePublishCapability:
		return cadence.PublishCapabilityType
	case SimpleTypeUnpublishCapability:
		return cadence.UnpublishCapabilityType
	case SimpleTypeGetStorageCapabilityController:
		return cadence.GetStorageCapabilityControllerType
	case SimpleTypeIssueStorageCapabilityController:
		return cadence.IssueStorageCapabilityControllerType
	case SimpleTypeGetAccountCapabilityController:
		return cadence.GetAccountCapabilityControllerType
	case SimpleTypeIssueAccountCapabilityController:
		return cadence.IssueAccountCapabilityControllerType
	case SimpleTypeCapabilitiesMapping:
		return cadence.CapabilitiesMappingType
	case SimpleTypeAccountMapping:
		return cadence.AccountMappingType
	}

	return nil
}
