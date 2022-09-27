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

// EncodeValue encodes any supported cadence.Value.
func (e *Encoder) EncodeValue(value cadence.Value) (err error) {
	switch v := value.(type) {
	case cadence.Void:
		return e.EncodeValueIdentifier(EncodedValueVoid)
	case cadence.Optional:
		err = e.EncodeValueIdentifier(EncodedValueOptional)
		if err != nil {
			return
		}
		return e.EncodeOptional(v)
	case cadence.Bool:
		err = e.EncodeValueIdentifier(EncodedValueBool)
		if err != nil {
			return
		}
		return common_codec.EncodeBool(&e.w, bool(v))
	case cadence.String:
		err = e.EncodeValueIdentifier(EncodedValueString)
		if err != nil {
			return
		}
		return common_codec.EncodeString(&e.w, string(v))
	case cadence.Character:
		err = e.EncodeValueIdentifier(EncodedValueCharacter)
		if err != nil {
			return
		}
		return common_codec.EncodeString(&e.w, string(v))
	case cadence.Address:
		err = e.EncodeValueIdentifier(EncodedValueAddress)
		if err != nil {
			return
		}
		return common_codec.EncodeAddress(&e.w, v)
	case cadence.Int:
		err = e.EncodeValueIdentifier(EncodedValueInt)
		if err != nil {
			return
		}
		return common_codec.EncodeBigInt(&e.w, v.Big())
	case cadence.Int8:
		err = e.EncodeValueIdentifier(EncodedValueInt8)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int8(v))
	case cadence.Int16:
		err = e.EncodeValueIdentifier(EncodedValueInt16)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int16(v))
	case cadence.Int32:
		err = e.EncodeValueIdentifier(EncodedValueInt32)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int32(v))
	case cadence.Int64:
		err = e.EncodeValueIdentifier(EncodedValueInt64)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int64(v))
	case cadence.Int128: // TODO encode more efficiently, as not a BigInt
		err = e.EncodeValueIdentifier(EncodedValueInt128)
		if err != nil {
			return
		}
		return common_codec.EncodeBigInt(&e.w, v.Big())
	case cadence.Int256:
		err = e.EncodeValueIdentifier(EncodedValueInt256)
		if err != nil {
			return
		}
		return common_codec.EncodeBigInt(&e.w, v.Big())
	case cadence.UInt:
		err = e.EncodeValueIdentifier(EncodedValueUInt)
		if err != nil {
			return
		}
		return common_codec.EncodeBigInt(&e.w, v.Big())
	case cadence.UInt8:
		err = e.EncodeValueIdentifier(EncodedValueUInt8)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int8(v))
	case cadence.UInt16:
		err = e.EncodeValueIdentifier(EncodedValueUInt16)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int16(v))
	case cadence.UInt32:
		err = e.EncodeValueIdentifier(EncodedValueUInt32)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int32(v))
	case cadence.UInt64:
		err = e.EncodeValueIdentifier(EncodedValueUInt64)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int64(v))
	case cadence.UInt128:
		err = e.EncodeValueIdentifier(EncodedValueUInt128)
		if err != nil {
			return
		}
		return common_codec.EncodeBigInt(&e.w, v.Big())
	case cadence.UInt256:
		err = e.EncodeValueIdentifier(EncodedValueUInt256)
		if err != nil {
			return
		}
		return common_codec.EncodeBigInt(&e.w, v.Big())
	case cadence.Word8:
		err = e.EncodeValueIdentifier(EncodedValueWord8)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, uint8(v))
	case cadence.Word16:
		err = e.EncodeValueIdentifier(EncodedValueWord16)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, uint16(v))
	case cadence.Word32:
		err = e.EncodeValueIdentifier(EncodedValueWord32)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, uint32(v))
	case cadence.Word64:
		err = e.EncodeValueIdentifier(EncodedValueWord64)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, uint64(v))
	case cadence.Fix64:
		err = e.EncodeValueIdentifier(EncodedValueFix64)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int64(v))
	case cadence.UFix64:
		err = e.EncodeValueIdentifier(EncodedValueUFix64)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, uint64(v))
	case cadence.Array:
		switch arrayType := v.ArrayType.(type) {
		case cadence.VariableSizedArrayType:
			err = e.EncodeValueIdentifier(EncodedValueVariableArray)
			if err != nil {
				return
			}
			err = e.EncodeVariableArrayType(arrayType)
		case cadence.ConstantSizedArrayType:
			err = e.EncodeValueIdentifier(EncodedValueConstantArray)
			if err != nil {
				return
			}
			err = e.EncodeConstantArrayType(arrayType)
		case nil:
			err = e.EncodeValueIdentifier(EncodedValueUntypedArray)
		default:
			err = common_codec.CodecError(fmt.Sprintf("unknown array type: %s", arrayType))
		}
		if err != nil {
			return
		}
		return e.EncodeArray(v)
	case cadence.Dictionary:
		err = e.EncodeValueIdentifier(EncodedValueDictionary)
		if err != nil {
			return
		}
		return e.EncodeDictionary(v)
	case cadence.Struct:
		err = e.EncodeValueIdentifier(EncodedValueStruct)
		if err != nil {
			return
		}
		return e.EncodeStruct(v)
	case cadence.Resource:
		err = e.EncodeValueIdentifier(EncodedValueResource)
		if err != nil {
			return
		}
		return e.EncodeResource(v)
	case cadence.Event:
		err = e.EncodeValueIdentifier(EncodedValueEvent)
		if err != nil {
			return
		}
		return e.EncodeEvent(v)
	case cadence.Contract:
		err = e.EncodeValueIdentifier(EncodedValueContract)
		if err != nil {
			return
		}
		return e.EncodeContract(v)
	case cadence.Link:
		err = e.EncodeValueIdentifier(EncodedValueLink)
		if err != nil {
			return
		}
		return e.EncodeLink(v)
	case cadence.Path:
		err = e.EncodeValueIdentifier(EncodedValuePath)
		if err != nil {
			return
		}
		return e.EncodePath(v)
	case cadence.Capability:
		err = e.EncodeValueIdentifier(EncodedValueCapability)
		if err != nil {
			return
		}
		return e.EncodeCapability(v)
	case cadence.Enum:
		err = e.EncodeValueIdentifier(EncodedValueEnum)
		if err != nil {
			return
		}
		return e.EncodeEnum(v)
	}

	return common_codec.CodecError(fmt.Sprintf("unexpected value: %s (type=%s)", value, value.Type()))
}

func (e *Encoder) EncodeValueIdentifier(id EncodedValue) (err error) {
	return e.writeByte(byte(id))
}

func (e *Encoder) EncodeOptional(value cadence.Optional) (err error) {
	isNil := value.Value == nil
	err = common_codec.EncodeBool(&e.w, isNil)
	if isNil || err != nil {
		return
	}

	return e.EncodeValue(value.Value)
}

func (e *Encoder) EncodeArray(value cadence.Array) (err error) {
	switch v := value.ArrayType.(type) {
	case cadence.VariableSizedArrayType, nil: // unknown type still needs length
		err = common_codec.EncodeLength(&e.w, len(value.Values))
		if err != nil {
			return
		}
	case cadence.ConstantSizedArrayType:
		if len(value.Values) != int(v.Size) {
			return common_codec.CodecError(fmt.Sprintf("constant size array size=%d but has %d elements", v.Size, len(value.Values)))
		}
	}

	for _, element := range value.Values {
		err = e.EncodeValue(element)
		if err != nil {
			return err
		}
	}

	return
}

func (e *Encoder) EncodeDictionary(value cadence.Dictionary) (err error) {
	err = e.EncodeDictionaryType(value.DictionaryType)
	if err != nil {
		return
	}
	err = common_codec.EncodeLength(&e.w, len(value.Pairs))
	if err != nil {
		return
	}
	for _, kv := range value.Pairs {
		err = e.EncodeValue(kv.Key)
		if err != nil {
			return
		}
		err = e.EncodeValue(kv.Value)
		if err != nil {
			return
		}
	}
	return
}

func (e *Encoder) EncodeStruct(value cadence.Struct) (err error) {
	err = e.EncodeStructType(value.StructType)
	if err != nil {
		return
	}

	return EncodeArray(e, value.Fields, func(field cadence.Value) (err error) {
		return e.EncodeValue(field)
	})
}

func (e *Encoder) EncodeResource(value cadence.Resource) (err error) {
	err = e.EncodeResourceType(value.ResourceType)
	if err != nil {
		return
	}
	return EncodeArray(e, value.Fields, func(field cadence.Value) (err error) {
		return e.EncodeValue(field)
	})
}

func (e *Encoder) EncodeEvent(value cadence.Event) (err error) {
	err = e.EncodeEventType(value.EventType)
	if err != nil {
		return
	}
	return EncodeArray(e, value.Fields, func(field cadence.Value) (err error) {
		return e.EncodeValue(field)
	})
}

func (e *Encoder) EncodeContract(value cadence.Contract) (err error) {
	err = e.EncodeContractType(value.ContractType)
	if err != nil {
		return
	}
	return EncodeArray(e, value.Fields, func(field cadence.Value) (err error) {
		return e.EncodeValue(field)
	})
}

func (e *Encoder) EncodeLink(value cadence.Link) (err error) {
	err = e.EncodePath(value.TargetPath)
	if err != nil {
		return
	}

	return common_codec.EncodeString(&e.w, value.BorrowType)
}

func (e *Encoder) EncodePath(value cadence.Path) (err error) {
	err = common_codec.EncodeString(&e.w, value.Domain)
	if err != nil {
		return
	}

	return common_codec.EncodeString(&e.w, value.Identifier)
}

func (e *Encoder) EncodeCapability(value cadence.Capability) (err error) {
	err = e.EncodePath(value.Path)
	if err != nil {
		return
	}

	err = common_codec.EncodeAddress(&e.w, value.Address)
	if err != nil {
		return
	}

	return e.EncodeType(value.BorrowType)
}

func (e *Encoder) EncodeEnum(value cadence.Enum) (err error) {
	err = e.EncodeEnumType(value.EnumType)
	if err != nil {
		return
	}
	return EncodeArray(e, value.Fields, func(field cadence.Value) (err error) {
		return e.EncodeValue(field)
	})
}
