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

import (
	"fmt"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/cbf/common_codec"
)

func (e *Encoder) EncodeType(t cadence.Type) (err error) {
	switch actualType := t.(type) {
	case cadence.VoidType:
		return e.EncodeTypeIdentifier(EncodedTypeVoid)
	case cadence.BoolType:
		return e.EncodeTypeIdentifier(EncodedTypeBool)
	case cadence.OptionalType:
		err = e.EncodeTypeIdentifier(EncodedTypeOptional)
		if err != nil {
			return
		}
		return e.EncodeOptionalType(actualType)
	case cadence.StringType:
		return e.EncodeTypeIdentifier(EncodedTypeString)
	case cadence.CharacterType:
		return e.EncodeTypeIdentifier(EncodedTypeCharacter)
	case cadence.AddressType:
		return e.EncodeTypeIdentifier(EncodedTypeAddress)
	case cadence.IntType:
		return e.EncodeTypeIdentifier(EncodedTypeInt)
	case cadence.Int8Type:
		return e.EncodeTypeIdentifier(EncodedTypeInt8)
	case cadence.Int16Type:
		return e.EncodeTypeIdentifier(EncodedTypeInt16)
	case cadence.Int32Type:
		return e.EncodeTypeIdentifier(EncodedTypeInt32)
	case cadence.Int64Type:
		return e.EncodeTypeIdentifier(EncodedTypeInt64)
	case cadence.Int128Type:
		return e.EncodeTypeIdentifier(EncodedTypeInt128)
	case cadence.Int256Type:
		return e.EncodeTypeIdentifier(EncodedTypeInt256)
	case cadence.UIntType:
		return e.EncodeTypeIdentifier(EncodedTypeUInt)
	case cadence.UInt8Type:
		return e.EncodeTypeIdentifier(EncodedTypeUInt8)
	case cadence.UInt16Type:
		return e.EncodeTypeIdentifier(EncodedTypeUInt16)
	case cadence.UInt32Type:
		return e.EncodeTypeIdentifier(EncodedTypeUInt32)
	case cadence.UInt64Type:
		return e.EncodeTypeIdentifier(EncodedTypeUInt64)
	case cadence.UInt128Type:
		return e.EncodeTypeIdentifier(EncodedTypeUInt128)
	case cadence.UInt256Type:
		return e.EncodeTypeIdentifier(EncodedTypeUInt256)
	case cadence.Word8Type:
		return e.EncodeTypeIdentifier(EncodedTypeWord8)
	case cadence.Word16Type:
		return e.EncodeTypeIdentifier(EncodedTypeWord16)
	case cadence.Word32Type:
		return e.EncodeTypeIdentifier(EncodedTypeWord32)
	case cadence.Word64Type:
		return e.EncodeTypeIdentifier(EncodedTypeWord64)
	case cadence.Fix64Type:
		return e.EncodeTypeIdentifier(EncodedTypeFix64)
	case cadence.UFix64Type:
		return e.EncodeTypeIdentifier(EncodedTypeUFix64)
	case cadence.VariableSizedArrayType:
		err = e.EncodeTypeIdentifier(EncodedTypeVariableSizedArray)
		if err != nil {
			return
		}
		return e.EncodeVariableArrayType(actualType)
	case cadence.ConstantSizedArrayType:
		err = e.EncodeTypeIdentifier(EncodedTypeConstantSizedArray)
		if err != nil {
			return
		}
		return e.EncodeConstantArrayType(actualType)
	case cadence.DictionaryType:
		err = e.EncodeTypeIdentifier(EncodedTypeDictionary)
		if err != nil {
			return
		}
		return e.EncodeDictionaryType(actualType)
	case *cadence.StructType:
		err = e.EncodeTypeIdentifier(EncodedTypeStruct)
		if err != nil {
			return
		}
		return e.EncodeCompositeType(actualType)
	case *cadence.ResourceType:
		err = e.EncodeTypeIdentifier(EncodedTypeResource)
		if err != nil {
			return
		}
		return e.EncodeCompositeType(actualType)
	case *cadence.EventType:
		err = e.EncodeTypeIdentifier(EncodedTypeEvent)
		if err != nil {
			return
		}
		return e.EncodeCompositeType(actualType)
	case *cadence.ContractType:
		err = e.EncodeTypeIdentifier(EncodedTypeContract)
		if err != nil {
			return
		}
		return e.EncodeCompositeType(actualType)
	case *cadence.StructInterfaceType:
		err = e.EncodeTypeIdentifier(EncodedTypeStructInterface)
		if err != nil {
			return
		}
		return e.EncodeInterfaceType(actualType)
	case *cadence.ResourceInterfaceType:
		err = e.EncodeTypeIdentifier(EncodedTypeResourceInterface)
		if err != nil {
			return
		}
		return e.EncodeInterfaceType(actualType)
	case *cadence.ContractInterfaceType:
		err = e.EncodeTypeIdentifier(EncodedTypeContractInterface)
		if err != nil {
			return
		}
		return e.EncodeInterfaceType(actualType)
	case *cadence.FunctionType:
		err = e.EncodeTypeIdentifier(EncodedTypeFunction)
		if err != nil {
			return
		}
		return e.EncodeFunctionType(actualType)
	case cadence.ReferenceType:
		err = e.EncodeTypeIdentifier(EncodedTypeReference)
		if err != nil {
			return
		}
		return e.EncodeReferenceType(actualType)
	case *cadence.RestrictedType:
		err = e.EncodeTypeIdentifier(EncodedTypeRestricted)
		if err != nil {
			return
		}
		return e.EncodeRestrictedType(actualType)
	case cadence.BlockType:
		return e.EncodeTypeIdentifier(EncodedTypeBlock)
	case cadence.AuthAccountType:
		return e.EncodeTypeIdentifier(EncodedTypeAuthAccount)
	case cadence.PublicAccountType:
		return e.EncodeTypeIdentifier(EncodedTypePublicAccount)
	case cadence.DeployedContractType:
		return e.EncodeTypeIdentifier(EncodedTypeDeployedContract)
	case cadence.AuthAccountContractsType:
		return e.EncodeTypeIdentifier(EncodedTypeAuthAccountContracts)
	case cadence.PublicAccountContractsType:
		return e.EncodeTypeIdentifier(EncodedTypePublicAccountContracts)
	case cadence.AuthAccountKeysType:
		return e.EncodeTypeIdentifier(EncodedTypeAuthAccountKeys)
	case cadence.PublicAccountKeysType:
		return e.EncodeTypeIdentifier(EncodedTypePublicAccountKeys)
	case cadence.CapabilityPathType:
		return e.EncodeTypeIdentifier(EncodedTypeCapabilityPath)
	case cadence.StoragePathType:
		return e.EncodeTypeIdentifier(EncodedTypeStoragePath)
	case cadence.PublicPathType:
		return e.EncodeTypeIdentifier(EncodedTypePublicPath)
	case cadence.PrivatePathType:
		return e.EncodeTypeIdentifier(EncodedTypePrivatePath)
	case cadence.CapabilityType:
		err = e.EncodeTypeIdentifier(EncodedTypeCapability)
		if err != nil {
			return
		}
		return e.EncodeType(actualType.BorrowType)
	case *cadence.EnumType:
		err = e.EncodeTypeIdentifier(EncodedTypeEnum)
		if err != nil {
			return
		}
		return e.EncodeEnumType(actualType)

	case cadence.NeverType:
		return e.EncodeTypeIdentifier(EncodedTypeNever)
	case cadence.NumberType:
		return e.EncodeTypeIdentifier(EncodedTypeNumber)
	case cadence.SignedNumberType:
		return e.EncodeTypeIdentifier(EncodedTypeSignedNumber)
	case cadence.IntegerType:
		return e.EncodeTypeIdentifier(EncodedTypeInteger)
	case cadence.SignedIntegerType:
		return e.EncodeTypeIdentifier(EncodedTypeSignedInteger)
	case cadence.FixedPointType:
		return e.EncodeTypeIdentifier(EncodedTypeFixedPoint)
	case cadence.SignedFixedPointType:
		return e.EncodeTypeIdentifier(EncodedTypeSignedFixedPoint)
	case cadence.AnyType:
		return e.EncodeTypeIdentifier(EncodedTypeAnyType)
	case cadence.AnyStructType:
		return e.EncodeTypeIdentifier(EncodedTypeAnyStructType)
	case cadence.AnyResourceType:
		return e.EncodeTypeIdentifier(EncodedTypeAnyResourceType)
	case cadence.PathType:
		return e.EncodeTypeIdentifier(EncodedTypePath)
	case cadence.MetaType:
		return e.EncodeTypeIdentifier(EncodedTypeMetaType)
	}

	return common_codec.CodecError(fmt.Sprintf("unknown type: %s", t))
}

func (d *Decoder) DecodeType() (t cadence.Type, err error) {
	typeIdentifer, err := d.DecodeTypeIdentifier()

	if err != nil {
		return
	}

	switch typeIdentifer {
	case EncodedTypeVoid:
		t = cadence.NewMeteredVoidType(d.memoryGauge)
	case EncodedTypeNever:
		t = cadence.NewMeteredNeverType(d.memoryGauge)
	case EncodedTypeOptional:
		t, err = d.DecodeOptionalType()
	case EncodedTypeBool:
		t = cadence.NewMeteredBoolType(d.memoryGauge)
	case EncodedTypeString:
		t = cadence.NewMeteredStringType(d.memoryGauge)
	case EncodedTypeCharacter:
		t = cadence.NewMeteredCharacterType(d.memoryGauge)
	case EncodedTypeAddress:
		t = cadence.NewMeteredAddressType(d.memoryGauge)
	case EncodedTypeNumber:
		t = cadence.NewMeteredNumberType(d.memoryGauge)
	case EncodedTypeSignedNumber:
		t = cadence.NewMeteredSignedNumberType(d.memoryGauge)
	case EncodedTypeInteger:
		t = cadence.NewMeteredIntegerType(d.memoryGauge)
	case EncodedTypeSignedInteger:
		t = cadence.NewMeteredSignedIntegerType(d.memoryGauge)
	case EncodedTypeFixedPoint:
		t = cadence.NewMeteredFixedPointType(d.memoryGauge)
	case EncodedTypeSignedFixedPoint:
		t = cadence.NewMeteredSignedFixedPointType(d.memoryGauge)
	case EncodedTypeInt:
		t = cadence.NewMeteredIntType(d.memoryGauge)
	case EncodedTypeInt8:
		t = cadence.NewMeteredInt8Type(d.memoryGauge)
	case EncodedTypeInt16:
		t = cadence.NewMeteredInt16Type(d.memoryGauge)
	case EncodedTypeInt32:
		t = cadence.NewMeteredInt32Type(d.memoryGauge)
	case EncodedTypeInt64:
		t = cadence.NewMeteredInt64Type(d.memoryGauge)
	case EncodedTypeInt128:
		t = cadence.NewMeteredInt128Type(d.memoryGauge)
	case EncodedTypeInt256:
		t = cadence.NewMeteredInt256Type(d.memoryGauge)
	case EncodedTypeUInt128:
		t = cadence.NewMeteredUInt128Type(d.memoryGauge)
	case EncodedTypeUInt256:
		t = cadence.NewMeteredUInt256Type(d.memoryGauge)
	case EncodedTypeUInt:
		t = cadence.NewMeteredUIntType(d.memoryGauge)
	case EncodedTypeUInt8:
		t = cadence.NewMeteredUInt8Type(d.memoryGauge)
	case EncodedTypeUInt16:
		t = cadence.NewMeteredUInt16Type(d.memoryGauge)
	case EncodedTypeUInt32:
		t = cadence.NewMeteredUInt32Type(d.memoryGauge)
	case EncodedTypeUInt64:
		t = cadence.NewMeteredUInt64Type(d.memoryGauge)
	case EncodedTypeWord8:
		t = cadence.NewMeteredWord8Type(d.memoryGauge)
	case EncodedTypeWord16:
		t = cadence.NewMeteredWord16Type(d.memoryGauge)
	case EncodedTypeWord32:
		t = cadence.NewMeteredWord32Type(d.memoryGauge)
	case EncodedTypeWord64:
		t = cadence.NewMeteredWord64Type(d.memoryGauge)
	case EncodedTypeFix64:
		t = cadence.NewMeteredFix64Type(d.memoryGauge)
	case EncodedTypeUFix64:
		t = cadence.NewMeteredUFix64Type(d.memoryGauge)
	case EncodedTypeVariableSizedArray:
		t, err = d.DecodeVariableArrayType()
	case EncodedTypeConstantSizedArray:
		t, err = d.DecodeConstantArrayType()
	case EncodedTypeDictionary:
		t, err = d.DecodeDictionaryType()
	case EncodedTypeStruct:
		t, err = d.DecodeStructType()
	case EncodedTypeResource:
		t, err = d.DecodeResourceType()
	case EncodedTypeEvent:
		t, err = d.DecodeEventType()
	case EncodedTypeContract:
		t, err = d.DecodeContractType()
	case EncodedTypeStructInterface:
		t, err = d.DecodeStructInterfaceType()
	case EncodedTypeResourceInterface:
		t, err = d.DecodeResourceInterfaceType()
	case EncodedTypeContractInterface:
		t, err = d.DecodeContractInterfaceType()
	case EncodedTypeFunction:
		t, err = d.DecodeFunctionType()
	case EncodedTypeReference:
		t, err = d.DecodeReferenceType()
	case EncodedTypeRestricted:
		t, err = d.DecodeRestrictedType()
	case EncodedTypeBlock:
		t = cadence.NewMeteredBlockType(d.memoryGauge)
	case EncodedTypeAuthAccount:
		t = cadence.NewMeteredAuthAccountType(d.memoryGauge)
	case EncodedTypePublicAccount:
		t = cadence.NewMeteredPublicAccountType(d.memoryGauge)
	case EncodedTypeDeployedContract:
		t = cadence.NewMeteredDeployedContractType(d.memoryGauge)
	case EncodedTypeAuthAccountContracts:
		t = cadence.NewMeteredAuthAccountContractsType(d.memoryGauge)
	case EncodedTypePublicAccountContracts:
		t = cadence.NewMeteredPublicAccountContractsType(d.memoryGauge)
	case EncodedTypeAuthAccountKeys:
		t = cadence.NewMeteredAuthAccountKeysType(d.memoryGauge)
	case EncodedTypePublicAccountKeys:
		t = cadence.NewMeteredPublicAccountKeysType(d.memoryGauge)
	case EncodedTypeCapabilityPath:
		t = cadence.NewMeteredCapabilityPathType(d.memoryGauge)
	case EncodedTypeStoragePath:
		t = cadence.NewMeteredStoragePathType(d.memoryGauge)
	case EncodedTypePublicPath:
		t = cadence.NewMeteredPublicPathType(d.memoryGauge)
	case EncodedTypePrivatePath:
		t = cadence.NewMeteredPrivatePathType(d.memoryGauge)
	case EncodedTypeCapability:
		var borrowType cadence.Type
		borrowType, err = d.DecodeType()
		if err != nil {
			return
		}
		t = cadence.NewMeteredCapabilityType(d.memoryGauge, borrowType)
	case EncodedTypeEnum:
		return d.DecodeEnumType()

	case EncodedTypeAnyType:
		t = cadence.NewMeteredAnyType(d.memoryGauge)
	case EncodedTypeAnyStructType:
		t = cadence.NewMeteredAnyStructType(d.memoryGauge)
	case EncodedTypeAnyResourceType:
		t = cadence.NewMeteredAnyResourceType(d.memoryGauge)
	case EncodedTypePath:
		t = cadence.NewMeteredPathType(d.memoryGauge)
	case EncodedTypeMetaType:
		t = cadence.NewMeteredMetaType(d.memoryGauge)
	default:
		err = common_codec.CodecError(fmt.Sprintf("unknown type identifier: %d", typeIdentifer))
	}
	return
}

func (e *Encoder) EncodeTypeIdentifier(id EncodedType) (err error) {
	return e.writeByte(byte(id))
}

func (d *Decoder) DecodeTypeIdentifier() (t EncodedType, err error) {
	b, err := d.read(1)
	t = EncodedType(b[0])
	return
}
