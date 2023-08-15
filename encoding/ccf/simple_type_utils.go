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

const ( // Cadence simple type IDs
	TypeBool = iota
	TypeString
	TypeCharacter
	TypeAddress
	TypeInt
	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt64
	TypeInt128
	TypeInt256
	TypeUInt
	TypeUInt8
	TypeUInt16
	TypeUInt32
	TypeUInt64
	TypeUInt128
	TypeUInt256
	TypeWord8
	TypeWord16
	TypeWord32
	TypeWord64
	TypeFix64
	TypeUFix64
	TypePath
	TypeCapabilityPath
	TypeStoragePath
	TypePublicPath
	TypePrivatePath
	_ // DO NOT REUSE: was AuthAccount
	_ // DO NOT REUSE: was PublicAccount
	_ // DO NOT REUSE: was AuthAccountKeys
	_ // DO NOT REUSE: was PublicAccountKeys
	_ // DO NOT REUSE: was AuthAccountContracts
	_ // DO NOT REUSE: was PublicAccountContracts
	TypeDeployedContract
	_ // DO NOT REUSE: was AccountKey
	TypeBlock
	TypeAny
	TypeAnyStruct
	TypeAnyResource
	TypeMetaType
	TypeNever
	TypeNumber
	TypeSignedNumber
	TypeInteger
	TypeSignedInteger
	TypeFixedPoint
	TypeSignedFixedPoint
	TypeBytes
	TypeVoid
	TypeFunction
	TypeWord128
	TypeWord256
	TypeAnyStructAttachmentType
	TypeAnyResourceAttachmentType
	TypeStorageCapabilityController
	TypeAccountCapabilityController
	TypeAccount
	TypeAccount_Contracts
	TypeAccount_Keys
	TypeAccount_Inbox
	TypeAccount_StorageCapabilities
	TypeAccount_AccountCapabilities
	TypeAccount_Capabilities
	TypeAccount_Storage
	TypeMutate
	TypeInsert
	TypeRemove
	TypeIdentity
	TypeStorage
	TypeSaveValue
	TypeLoadValue
	TypeBorrowValue
	TypeContracts
	TypeAddContract
	TypeUpdateContract
	TypeRemoveContract
	TypeKeys
	TypeAddKey
	TypeRevokeKey
	TypeInbox
	TypePublishInboxCapability
	TypeUnpublishInboxCapability
	TypeClaimInboxCapability
	TypeCapabilities
	TypeStorageCapabilities
	TypeAccountCapabilities
	TypePublishCapability
	TypeUnpublishCapability
	TypeGetStorageCapabilityController
	TypeIssueStorageCapabilityController
	TypeGetAccountCapabilityController
	TypeIssueAccountCapabilityController
	TypeCapabilitiesMapping
	TypeAccountMapping
)

// NOTE: cadence.FunctionType isn't included in simpleTypeIDByType
// because this function is used by both inline-type and type-value.
// cadence.FunctionType needs to be handled differently when this
// function is used by inline-type and type-value.
func simpleTypeIDByType(typ cadence.Type) (uint64, bool) {

	switch typ {
	case cadence.AnyType:
		return TypeAny, true
	case cadence.AnyStructType:
		return TypeAnyStruct, true
	case cadence.AnyResourceType:
		return TypeAnyResource, true
	case cadence.AddressType:
		return TypeAddress, true
	case cadence.MetaType:
		return TypeMetaType, true
	case cadence.VoidType:
		return TypeVoid, true
	case cadence.NeverType:
		return TypeNever, true
	case cadence.BoolType:
		return TypeBool, true
	case cadence.StringType:
		return TypeString, true
	case cadence.CharacterType:
		return TypeCharacter, true
	case cadence.NumberType:
		return TypeNumber, true
	case cadence.SignedNumberType:
		return TypeSignedNumber, true
	case cadence.IntegerType:
		return TypeInteger, true
	case cadence.SignedIntegerType:
		return TypeSignedInteger, true
	case cadence.FixedPointType:
		return TypeFixedPoint, true
	case cadence.SignedFixedPointType:
		return TypeSignedFixedPoint, true
	case cadence.IntType:
		return TypeInt, true
	case cadence.Int8Type:
		return TypeInt8, true
	case cadence.Int16Type:
		return TypeInt16, true
	case cadence.Int32Type:
		return TypeInt32, true
	case cadence.Int64Type:
		return TypeInt64, true
	case cadence.Int128Type:
		return TypeInt128, true
	case cadence.Int256Type:
		return TypeInt256, true
	case cadence.UIntType:
		return TypeUInt, true
	case cadence.UInt8Type:
		return TypeUInt8, true
	case cadence.UInt16Type:
		return TypeUInt16, true
	case cadence.UInt32Type:
		return TypeUInt32, true
	case cadence.UInt64Type:
		return TypeUInt64, true
	case cadence.UInt128Type:
		return TypeUInt128, true
	case cadence.UInt256Type:
		return TypeUInt256, true
	case cadence.Word8Type:
		return TypeWord8, true
	case cadence.Word16Type:
		return TypeWord16, true
	case cadence.Word32Type:
		return TypeWord32, true
	case cadence.Word64Type:
		return TypeWord64, true
	case cadence.Word128Type:
		return TypeWord128, true
	case cadence.Word256Type:
		return TypeWord256, true
	case cadence.Fix64Type:
		return TypeFix64, true
	case cadence.UFix64Type:
		return TypeUFix64, true
	case cadence.BlockType:
		return TypeBlock, true
	case cadence.PathType:
		return TypePath, true
	case cadence.CapabilityPathType:
		return TypeCapabilityPath, true
	case cadence.StoragePathType:
		return TypeStoragePath, true
	case cadence.PublicPathType:
		return TypePublicPath, true
	case cadence.PrivatePathType:
		return TypePrivatePath, true
	case cadence.DeployedContractType:
		return TypeDeployedContract, true
	case cadence.AnyStructAttachmentType:
		return TypeAnyStructAttachmentType, true
	case cadence.AnyResourceAttachmentType:
		return TypeAnyResourceAttachmentType, true
	case cadence.StorageCapabilityControllerType:
		return TypeStorageCapabilityController, true
	case cadence.AccountCapabilityControllerType:
		return TypeAccountCapabilityController, true
	case cadence.AccountType:
		return TypeAccount, true
	case cadence.Account_ContractsType:
		return TypeAccount_Contracts, true
	case cadence.Account_KeysType:
		return TypeAccount_Keys, true
	case cadence.Account_InboxType:
		return TypeAccount_Inbox, true
	case cadence.Account_StorageCapabilitiesType:
		return TypeAccount_StorageCapabilities, true
	case cadence.Account_AccountCapabilitiesType:
		return TypeAccount_AccountCapabilities, true
	case cadence.Account_CapabilitiesType:
		return TypeAccount_Capabilities, true
	case cadence.Account_StorageType:
		return TypeAccount_Storage, true
	case cadence.MutateType:
		return TypeMutate, true
	case cadence.InsertType:
		return TypeInsert, true
	case cadence.RemoveType:
		return TypeRemove, true
	case cadence.IdentityType:
		return TypeIdentity, true
	case cadence.StorageType:
		return TypeStorage, true
	case cadence.SaveValueType:
		return TypeSaveValue, true
	case cadence.LoadValueType:
		return TypeLoadValue, true
	case cadence.BorrowValueType:
		return TypeBorrowValue, true
	case cadence.ContractsType:
		return TypeContracts, true
	case cadence.AddContractType:
		return TypeAddContract, true
	case cadence.UpdateContractType:
		return TypeUpdateContract, true
	case cadence.RemoveContractType:
		return TypeRemoveContract, true
	case cadence.KeysType:
		return TypeKeys, true
	case cadence.AddKeyType:
		return TypeAddKey, true
	case cadence.RevokeKeyType:
		return TypeRevokeKey, true
	case cadence.InboxType:
		return TypeInbox, true
	case cadence.PublishInboxCapabilityType:
		return TypePublishInboxCapability, true
	case cadence.UnpublishInboxCapabilityType:
		return TypeUnpublishInboxCapability, true
	case cadence.ClaimInboxCapabilityType:
		return TypeClaimInboxCapability, true
	case cadence.CapabilitiesType:
		return TypeCapabilities, true
	case cadence.StorageCapabilitiesType:
		return TypeStorageCapabilities, true
	case cadence.AccountCapabilitiesType:
		return TypeAccountCapabilities, true
	case cadence.PublishCapabilityType:
		return TypePublishCapability, true
	case cadence.UnpublishCapabilityType:
		return TypeUnpublishCapability, true
	case cadence.GetStorageCapabilityControllerType:
		return TypeGetStorageCapabilityController, true
	case cadence.IssueStorageCapabilityControllerType:
		return TypeIssueStorageCapabilityController, true
	case cadence.GetAccountCapabilityControllerType:
		return TypeGetAccountCapabilityController, true
	case cadence.IssueAccountCapabilityControllerType:
		return TypeIssueAccountCapabilityController, true
	case cadence.CapabilitiesMappingType:
		return TypeCapabilitiesMapping, true
	case cadence.AccountMappingType:
		return TypeAccountMapping, true

	}

	switch typ.(type) {
	case cadence.BytesType:
		return TypeBytes, true
	}

	return 0, false
}

func typeBySimpleTypeID(simpleTypeID uint64) cadence.Type {
	switch simpleTypeID {
	case TypeBool:
		return cadence.BoolType
	case TypeString:
		return cadence.StringType
	case TypeCharacter:
		return cadence.CharacterType
	case TypeAddress:
		return cadence.AddressType
	case TypeInt:
		return cadence.IntType
	case TypeInt8:
		return cadence.Int8Type
	case TypeInt16:
		return cadence.Int16Type
	case TypeInt32:
		return cadence.Int32Type
	case TypeInt64:
		return cadence.Int64Type
	case TypeInt128:
		return cadence.Int128Type
	case TypeInt256:
		return cadence.Int256Type
	case TypeUInt:
		return cadence.UIntType
	case TypeUInt8:
		return cadence.UInt8Type
	case TypeUInt16:
		return cadence.UInt16Type
	case TypeUInt32:
		return cadence.UInt32Type
	case TypeUInt64:
		return cadence.UInt64Type
	case TypeUInt128:
		return cadence.UInt128Type
	case TypeUInt256:
		return cadence.UInt256Type
	case TypeWord8:
		return cadence.Word8Type
	case TypeWord16:
		return cadence.Word16Type
	case TypeWord32:
		return cadence.Word32Type
	case TypeWord64:
		return cadence.Word64Type
	case TypeWord128:
		return cadence.Word128Type
	case TypeWord256:
		return cadence.Word256Type
	case TypeFix64:
		return cadence.Fix64Type
	case TypeUFix64:
		return cadence.UFix64Type
	case TypePath:
		return cadence.PathType
	case TypeCapabilityPath:
		return cadence.CapabilityPathType
	case TypeStoragePath:
		return cadence.StoragePathType
	case TypePublicPath:
		return cadence.PublicPathType
	case TypePrivatePath:
		return cadence.PrivatePathType
	case TypeDeployedContract:
		return cadence.DeployedContractType
	case TypeBlock:
		return cadence.BlockType
	case TypeAny:
		return cadence.AnyType
	case TypeAnyStruct:
		return cadence.AnyStructType
	case TypeAnyResource:
		return cadence.AnyResourceType
	case TypeMetaType:
		return cadence.MetaType
	case TypeNever:
		return cadence.NeverType
	case TypeNumber:
		return cadence.NumberType
	case TypeSignedNumber:
		return cadence.SignedNumberType
	case TypeInteger:
		return cadence.IntegerType
	case TypeSignedInteger:
		return cadence.SignedIntegerType
	case TypeFixedPoint:
		return cadence.FixedPointType
	case TypeSignedFixedPoint:
		return cadence.SignedFixedPointType
	case TypeBytes:
		return cadence.TheBytesType
	case TypeVoid:
		return cadence.VoidType
	case TypeAnyStructAttachmentType:
		return cadence.AnyStructAttachmentType
	case TypeAnyResourceAttachmentType:
		return cadence.AnyResourceAttachmentType
	case TypeStorageCapabilityController:
		return cadence.StorageCapabilityControllerType
	case TypeAccountCapabilityController:
		return cadence.AccountCapabilityControllerType
	case TypeAccount:
		return cadence.AccountType
	case TypeAccount_Contracts:
		return cadence.Account_ContractsType
	case TypeAccount_Keys:
		return cadence.Account_KeysType
	case TypeAccount_Inbox:
		return cadence.Account_InboxType
	case TypeAccount_StorageCapabilities:
		return cadence.Account_StorageCapabilitiesType
	case TypeAccount_AccountCapabilities:
		return cadence.Account_AccountCapabilitiesType
	case TypeAccount_Capabilities:
		return cadence.Account_CapabilitiesType
	case TypeAccount_Storage:
		return cadence.Account_StorageType
	case TypeMutate:
		return cadence.MutateType
	case TypeInsert:
		return cadence.InsertType
	case TypeRemove:
		return cadence.RemoveType
	case TypeIdentity:
		return cadence.IdentityType
	case TypeStorage:
		return cadence.StorageType
	case TypeSaveValue:
		return cadence.SaveValueType
	case TypeLoadValue:
		return cadence.LoadValueType
	case TypeBorrowValue:
		return cadence.BorrowValueType
	case TypeContracts:
		return cadence.ContractsType
	case TypeAddContract:
		return cadence.AddContractType
	case TypeUpdateContract:
		return cadence.UpdateContractType
	case TypeRemoveContract:
		return cadence.RemoveContractType
	case TypeKeys:
		return cadence.KeysType
	case TypeAddKey:
		return cadence.AddKeyType
	case TypeRevokeKey:
		return cadence.RevokeKeyType
	case TypeInbox:
		return cadence.InboxType
	case TypePublishInboxCapability:
		return cadence.PublishInboxCapabilityType
	case TypeUnpublishInboxCapability:
		return cadence.UnpublishInboxCapabilityType
	case TypeClaimInboxCapability:
		return cadence.ClaimInboxCapabilityType
	case TypeCapabilities:
		return cadence.CapabilitiesType
	case TypeStorageCapabilities:
		return cadence.StorageCapabilitiesType
	case TypeAccountCapabilities:
		return cadence.AccountCapabilitiesType
	case TypePublishCapability:
		return cadence.PublishCapabilityType
	case TypeUnpublishCapability:
		return cadence.UnpublishCapabilityType
	case TypeGetStorageCapabilityController:
		return cadence.GetStorageCapabilityControllerType
	case TypeIssueStorageCapabilityController:
		return cadence.IssueStorageCapabilityControllerType
	case TypeGetAccountCapabilityController:
		return cadence.GetAccountCapabilityControllerType
	case TypeIssueAccountCapabilityController:
		return cadence.IssueAccountCapabilityControllerType
	case TypeCapabilitiesMapping:
		return cadence.CapabilitiesMappingType
	case TypeAccountMapping:
		return cadence.AccountMappingType
	}

	return nil
}
