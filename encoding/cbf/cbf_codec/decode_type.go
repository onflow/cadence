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
	"github.com/onflow/cadence/runtime/common"
)

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
	case EncodedTypeBytes:
		t = cadence.NewMeteredBytesType(d.memoryGauge)
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

func (d *Decoder) DecodeTypeIdentifier() (t EncodedType, err error) {
	b, err := d.read(1)
	t = EncodedType(b[0])
	return
}

func (d *Decoder) DecodeOptionalType() (t cadence.OptionalType, err error) {
	isNil, err := common_codec.DecodeBool(&d.r)
	if isNil || err != nil {
		return
	}

	elementType, err := d.DecodeType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredOptionalType(d.memoryGauge, elementType)
	return
}

func (d *Decoder) DecodeVariableArrayType() (t cadence.VariableSizedArrayType, err error) {
	elementType, err := d.DecodeType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredVariableSizedArrayType(d.memoryGauge, elementType)
	return
}

func (d *Decoder) DecodeConstantArrayType() (t cadence.ConstantSizedArrayType, err error) {
	elementType, err := d.DecodeType()
	if err != nil {
		return
	}

	size, err := common_codec.DecodeLength(&d.r)
	if err != nil {
		return
	}
	t = cadence.NewMeteredConstantSizedArrayType(d.memoryGauge, uint(size), elementType)
	return
}

func (d *Decoder) DecodeDictionaryType() (t cadence.DictionaryType, err error) {
	keyType, err := d.DecodeType()
	if err != nil {
		return
	}
	elementType, err := d.DecodeType()
	if err != nil {
		return
	}
	t = cadence.NewMeteredDictionaryType(d.memoryGauge, keyType, elementType)
	return
}

func (d *Decoder) DecodeStructType() (t *cadence.StructType, err error) {
	location, err := common_codec.DecodeLocation(&d.r, d.memoryGauge)
	if err != nil {
		return
	}

	qualifiedIdentifier, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (field cadence.Field, err error) {
		return d.DecodeField()
	})
	if err != nil {
		return
	}

	initializers, err := DecodeArray(d, func() ([]cadence.Parameter, error) {
		return DecodeArray(d, func() (cadence.Parameter, error) {
			return d.DecodeParameter()
		})
	})

	t = cadence.NewMeteredStructType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeResourceType() (t *cadence.ResourceType, err error) {
	location, err := common_codec.DecodeLocation(&d.r, d.memoryGauge)
	if err != nil {
		return
	}

	qualifiedIdentifier, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (field cadence.Field, err error) {
		return d.DecodeField()
	})
	if err != nil {
		return
	}

	initializers, err := DecodeArray(d, func() ([]cadence.Parameter, error) {
		return DecodeArray(d, func() (cadence.Parameter, error) {
			return d.DecodeParameter()
		})
	})

	t = cadence.NewMeteredResourceType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeEventType() (t *cadence.EventType, err error) {
	location, err := common_codec.DecodeLocation(&d.r, d.memoryGauge)
	if err != nil {
		return
	}

	qualifiedIdentifier, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (field cadence.Field, err error) {
		return d.DecodeField()
	})
	if err != nil {
		return
	}

	initializers, err := DecodeArray(d, func() (cadence.Parameter, error) {
		return d.DecodeParameter()
	})

	t = cadence.NewMeteredEventType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeContractType() (t *cadence.ContractType, err error) {
	location, err := common_codec.DecodeLocation(&d.r, d.memoryGauge)
	if err != nil {
		return
	}

	qualifiedIdentifier, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (field cadence.Field, err error) {
		return d.DecodeField()
	})
	if err != nil {
		return
	}

	initializers, err := DecodeArray(d, func() ([]cadence.Parameter, error) {
		return DecodeArray(d, func() (cadence.Parameter, error) {
			return d.DecodeParameter()
		})
	})

	t = cadence.NewMeteredContractType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeStructInterfaceType() (t *cadence.StructInterfaceType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeInterfaceType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredStructInterfaceType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeResourceInterfaceType() (t *cadence.ResourceInterfaceType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeInterfaceType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredResourceInterfaceType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeContractInterfaceType() (t *cadence.ContractInterfaceType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeInterfaceType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredContractInterfaceType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) decodeInterfaceType() (
	location common.Location,
	qualifiedIdentifier string,
	fields []cadence.Field,
	initializers [][]cadence.Parameter,
	err error,
) {
	location, err = common_codec.DecodeLocation(&d.r, d.memoryGauge)
	if err != nil {
		return
	}

	qualifiedIdentifier, err = common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	fields, err = DecodeArray(d, func() (field cadence.Field, err error) {
		return d.DecodeField()
	})
	if err != nil {
		return
	}

	initializers, err = DecodeArray(d, func() ([]cadence.Parameter, error) {
		return DecodeArray(d, func() (cadence.Parameter, error) {
			return d.DecodeParameter()
		})
	})

	return
}

func (d *Decoder) DecodeFunctionType() (t *cadence.FunctionType, err error) {
	id, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	parameters, err := DecodeArray(d, d.DecodeParameter)
	if err != nil {
		return
	}

	returnType, err := d.DecodeType()

	if err != nil {
		return
	}

	t = cadence.NewMeteredFunctionType(d.memoryGauge, id, parameters, returnType)
	return
}

func (d *Decoder) DecodeReferenceType() (t cadence.ReferenceType, err error) {
	authorized, err := common_codec.DecodeBool(&d.r)
	if err != nil {
		return
	}

	innerType, err := d.DecodeType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredReferenceType(d.memoryGauge, authorized, innerType)
	return
}

func (d *Decoder) DecodeRestrictedType() (t *cadence.RestrictedType, err error) {
	id, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	innerType, err := d.DecodeType()
	if err != nil {
		return
	}

	restrictions, err := DecodeArray(d, d.DecodeType)
	if err != nil {
		return
	}

	t = cadence.NewMeteredRestrictedType(d.memoryGauge, id, innerType, restrictions)
	return
}

func (d *Decoder) DecodeEnumType() (t *cadence.EnumType, err error) {
	location, err := common_codec.DecodeLocation(&d.r, d.memoryGauge)
	if err != nil {
		return
	}

	qualifiedIdentifier, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	rawType, err := d.DecodeType()
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (field cadence.Field, err error) {
		return d.DecodeField()
	})
	if err != nil {
		return
	}

	initializers, err := DecodeArray(d, func() ([]cadence.Parameter, error) {
		return DecodeArray(d, func() (cadence.Parameter, error) {
			return d.DecodeParameter()
		})
	})

	t = cadence.NewMeteredEnumType(d.memoryGauge, location, qualifiedIdentifier, rawType, fields, initializers)
	return
}

func (d *Decoder) DecodeField() (field cadence.Field, err error) {
	// TODO meter
	field.Identifier, err = common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	field.Type, err = d.DecodeType()
	return
}

func (d *Decoder) DecodeParameter() (parameter cadence.Parameter, err error) {
	// TODO meter?
	parameter.Label, err = common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	parameter.Identifier, err = common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	parameter.Type, err = d.DecodeType()
	return
}
