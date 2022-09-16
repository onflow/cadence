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
		err = common_codec.EncodeBool(&e.w, actualType.Type == nil)
		if err != nil {
			return
		}
		return e.EncodeType(actualType.Type)
	case cadence.StringType:
		return e.EncodeTypeIdentifier(EncodedTypeString)
	case cadence.CharacterType:
		return e.EncodeTypeIdentifier(EncodedTypeCharacter)
	case cadence.BytesType:
		return e.EncodeTypeIdentifier(EncodedTypeBytes)
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
		return e.EncodeStructType(actualType)
	case *cadence.ResourceType:
		err = e.EncodeTypeIdentifier(EncodedTypeResource)
		if err != nil {
			return
		}
		return e.EncodeResourceType(actualType)
	case *cadence.EventType:
		err = e.EncodeTypeIdentifier(EncodedTypeEvent)
		if err != nil {
			return
		}
		return e.EncodeEventType(actualType)
	case *cadence.ContractType:
		err = e.EncodeTypeIdentifier(EncodedTypeContract)
		if err != nil {
			return
		}
		return e.EncodeContractType(actualType)
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

func (e *Encoder) EncodeFunctionType(t *cadence.FunctionType) (err error) {
	err = common_codec.EncodeString(&e.w, t.ID())
	if err != nil {
		return
	}

	err = EncodeArray(e, t.Parameters, e.EncodeParameter)
	if err != nil {
		return
	}

	return e.EncodeType(t.ReturnType)
}

func (e *Encoder) EncodeReferenceType(t cadence.ReferenceType) (err error) {
	err = common_codec.EncodeBool(&e.w, t.Authorized)
	if err != nil {
		return
	}

	return e.EncodeType(t.Type)
}

func (e *Encoder) EncodeRestrictedType(t *cadence.RestrictedType) (err error) {
	err = common_codec.EncodeString(&e.w, t.ID())
	if err != nil {
		return
	}

	err = e.EncodeType(t.Type)
	if err != nil {
		return
	}

	return EncodeArray(e, t.Restrictions, e.EncodeType)
}

func (e *Encoder) EncodeTypeIdentifier(id EncodedType) (err error) {
	return e.writeByte(byte(id))
}

func (e *Encoder) EncodeVariableArrayType(t cadence.VariableSizedArrayType) (err error) {
	return e.EncodeType(t.Element())
}

func (e *Encoder) EncodeConstantArrayType(t cadence.ConstantSizedArrayType) (err error) {
	err = e.EncodeType(t.Element())
	if err != nil {
		return
	}

	return common_codec.EncodeLength(&e.w, int(t.Size))
}

func (e *Encoder) EncodeDictionaryType(t cadence.DictionaryType) (err error) {
	err = e.EncodeType(t.KeyType)
	if err != nil {
		return
	}
	return e.EncodeType(t.ElementType)
}

func (e *Encoder) EncodeStructType(t *cadence.StructType) (err error) {
	err = common_codec.EncodeLocation(&e.w, t.Location)
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, t.QualifiedIdentifier)
	if err != nil {
		return
	}

	err = EncodeArray(e, t.Fields, func(field cadence.Field) (err error) {
		return e.EncodeField(field)
	})

	return EncodeArray(e, t.Initializers, func(parameters []cadence.Parameter) (err error) {
		return EncodeArray(e, parameters, func(parameter cadence.Parameter) (err error) {
			return e.EncodeParameter(parameter)
		})
	})
}

func (e *Encoder) EncodeResourceType(t *cadence.ResourceType) (err error) {
	err = common_codec.EncodeLocation(&e.w, t.Location)
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, t.QualifiedIdentifier)
	if err != nil {
		return
	}

	err = EncodeArray(e, t.Fields, func(field cadence.Field) (err error) {
		return e.EncodeField(field)
	})

	return EncodeArray(e, t.Initializers, func(parameters []cadence.Parameter) (err error) {
		return EncodeArray(e, parameters, func(parameter cadence.Parameter) (err error) {
			return e.EncodeParameter(parameter)
		})
	})
}

func (e *Encoder) EncodeEventType(t *cadence.EventType) (err error) {
	err = common_codec.EncodeLocation(&e.w, t.Location)
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, t.QualifiedIdentifier)
	if err != nil {
		return
	}

	err = EncodeArray(e, t.Fields, func(field cadence.Field) (err error) {
		return e.EncodeField(field)
	})

	return EncodeArray(e, t.Initializer, func(parameter cadence.Parameter) (err error) {
		return e.EncodeParameter(parameter)
	})
}

func (e *Encoder) EncodeContractType(t *cadence.ContractType) (err error) {
	err = common_codec.EncodeLocation(&e.w, t.Location)
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, t.QualifiedIdentifier)
	if err != nil {
		return
	}

	err = EncodeArray(e, t.Fields, func(field cadence.Field) (err error) {
		return e.EncodeField(field)
	})

	return EncodeArray(e, t.Initializers, func(parameters []cadence.Parameter) (err error) {
		return EncodeArray(e, parameters, func(parameter cadence.Parameter) (err error) {
			return e.EncodeParameter(parameter)
		})
	})
}

func (e *Encoder) EncodeInterfaceType(t cadence.InterfaceType) (err error) {
	err = common_codec.EncodeLocation(&e.w, t.InterfaceTypeLocation())
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, t.InterfaceTypeQualifiedIdentifier())
	if err != nil {
		return
	}

	err = EncodeArray(e, t.InterfaceFields(), func(field cadence.Field) (err error) {
		return e.EncodeField(field)
	})

	return EncodeArray(e, t.InterfaceInitializers(), func(parameters []cadence.Parameter) (err error) {
		return EncodeArray(e, parameters, func(parameter cadence.Parameter) (err error) {
			return e.EncodeParameter(parameter)
		})
	})
}

func (e *Encoder) EncodeEnumType(t *cadence.EnumType) (err error) {
	err = common_codec.EncodeLocation(&e.w, t.Location)
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, t.QualifiedIdentifier)
	if err != nil {
		return
	}

	err = e.EncodeType(t.RawType)
	if err != nil {
		return
	}

	err = EncodeArray(e, t.Fields, func(field cadence.Field) (err error) {
		return e.EncodeField(field)
	})

	return EncodeArray(e, t.Initializers, func(parameters []cadence.Parameter) (err error) {
		return EncodeArray(e, parameters, func(parameter cadence.Parameter) (err error) {
			return e.EncodeParameter(parameter)
		})
	})
}

func (e *Encoder) EncodeField(field cadence.Field) (err error) {
	err = common_codec.EncodeString(&e.w, field.Identifier)
	if err != nil {
		return
	}

	return e.EncodeType(field.Type)
}

func (e *Encoder) EncodeParameter(parameter cadence.Parameter) (err error) {
	err = common_codec.EncodeString(&e.w, parameter.Label)
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, parameter.Identifier)
	if err != nil {
		return
	}

	return e.EncodeType(parameter.Type)
}
