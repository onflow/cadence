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

package cbf_codec

type EncodedValue byte

const (
	EncodedValueUnknown EncodedValue = iota

	EncodedValueVoid
	EncodedValueOptional
	EncodedValueBool
	EncodedValueString
	EncodedValueCharacter
	EncodedValueAddress
	EncodedValueInt
	EncodedValueInt8
	EncodedValueInt16
	EncodedValueInt32
	EncodedValueInt64
	EncodedValueInt128
	EncodedValueInt256
	EncodedValueUInt
	EncodedValueUInt8
	EncodedValueUInt16
	EncodedValueUInt32
	EncodedValueUInt64
	EncodedValueUInt128
	EncodedValueUInt256
	EncodedValueWord8
	EncodedValueWord16
	EncodedValueWord32
	EncodedValueWord64
	EncodedValueFix64
	EncodedValueUFix64
	EncodedValueUntypedArray
	EncodedValueVariableArray
	EncodedValueConstantArray
	EncodedValueDictionary
	EncodedValueStruct
	EncodedValueResource
	EncodedValueEvent
	EncodedValueContract
	EncodedValueLink
	EncodedValuePath
	EncodedValueCapability
	EncodedValueEnum
)

type EncodedType byte

const (
	EncodedTypeUnknown EncodedType = iota

	// Concrete Types

	EncodedTypeVoid
	EncodedTypeBool
	EncodedTypeOptional
	EncodedTypeString
	EncodedTypeCharacter
	EncodedTypeBytes
	EncodedTypeAddress
	EncodedTypeInt
	EncodedTypeInt8
	EncodedTypeInt16
	EncodedTypeInt32
	EncodedTypeInt64
	EncodedTypeInt128
	EncodedTypeInt256
	EncodedTypeUInt
	EncodedTypeUInt8
	EncodedTypeUInt16
	EncodedTypeUInt32
	EncodedTypeUInt64
	EncodedTypeUInt128
	EncodedTypeUInt256
	EncodedTypeWord8
	EncodedTypeWord16
	EncodedTypeWord32
	EncodedTypeWord64
	EncodedTypeFix64
	EncodedTypeUFix64
	EncodedTypeVariableSizedArray
	EncodedTypeConstantSizedArray
	EncodedTypeDictionary
	EncodedTypeStruct
	EncodedTypeResource
	EncodedTypeEvent
	EncodedTypeContract
	EncodedTypeStructInterface
	EncodedTypeResourceInterface
	EncodedTypeContractInterface
	EncodedTypeFunction
	EncodedTypeReference
	EncodedTypeRestricted
	EncodedTypeBlock
	EncodedTypeCapabilityPath
	EncodedTypeStoragePath
	EncodedTypePublicPath
	EncodedTypePrivatePath
	EncodedTypeCapability
	EncodedTypeEnum
	EncodedTypeAuthAccount
	EncodedTypePublicAccount
	EncodedTypeDeployedContract
	EncodedTypeAuthAccountContracts
	EncodedTypePublicAccountContracts
	EncodedTypeAuthAccountKeys
	EncodedTypePublicAccountKeys

	// Abstract Types

	EncodedTypeNever
	EncodedTypeNumber
	EncodedTypeSignedNumber
	EncodedTypeInteger
	EncodedTypeSignedInteger
	EncodedTypeFixedPoint
	EncodedTypeSignedFixedPoint
	EncodedTypeAnyType
	EncodedTypeAnyStructType
	EncodedTypeAnyResourceType
	EncodedTypePath
	EncodedTypeMetaType
)
