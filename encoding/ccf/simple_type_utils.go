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

// IMPORTANT:
//
// Don't change existing simple type IDs.
//
// When new simple cadence.Type is added,
// - add new ID to the end of existing IDs,
// - add new simple cadence.Type and its ID in simpleTypeIDByType()

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
	TypeAuthAccount
	TypePublicAccount
	TypeAuthAccountKeys
	TypePublicAccountKeys
	TypeAuthAccountContracts
	TypePublicAccountContracts
	TypeDeployedContract
	TypeAccountKey
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
)

// NOTE: cadence.FunctionType isn't included in simpleTypeIDByType
// because this function is used by both inline-type and type-value.
// cadence.FunctionType needs to be handled differently when this
// function is used by inline-type and type-value.
func simpleTypeIDByType(typ cadence.Type) (uint64, bool) {
	switch typ.(type) {
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

	case cadence.BytesType:
		return TypeBytes, true

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

	case cadence.AccountKeyType:
		return TypeAccountKey, true

	case cadence.AuthAccountContractsType:
		return TypeAuthAccountContracts, true

	case cadence.AuthAccountKeysType:
		return TypeAuthAccountKeys, true

	case cadence.AuthAccountType:
		return TypeAuthAccount, true

	case cadence.PublicAccountContractsType:
		return TypePublicAccountContracts, true

	case cadence.PublicAccountKeysType:
		return TypePublicAccountKeys, true

	case cadence.PublicAccountType:
		return TypePublicAccount, true

	case cadence.DeployedContractType:
		return TypeDeployedContract, true

	case cadence.AnyStructAttachmentType:
		return TypeAnyStructAttachmentType, true

	case cadence.AnyResourceAttachmentType:
		return TypeAnyResourceAttachmentType, true
	}

	return 0, false
}
