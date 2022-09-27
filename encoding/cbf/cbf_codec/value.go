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

func (d *Decoder) DecodeValue() (value cadence.Value, err error) {
	identifier, err := d.DecodeIdentifier()
	if err != nil {
		return
	}

	switch identifier {
	case EncodedValueVoid:
		value = cadence.NewMeteredVoid(d.memoryGauge)
	case EncodedValueOptional:
		value, err = d.DecodeOptional()
	case EncodedValueBool:
		value, err = d.DecodeBool()
	case EncodedValueString:
		value, err = d.DecodeString()
	case EncodedValueCharacter:
		value, err = d.DecodeCharacter()
	case EncodedValueAddress:
		value, err = d.DecodeAddress()
	case EncodedValueInt:
		value, err = d.DecodeInt()
	case EncodedValueInt8:
		value, err = d.DecodeInt8()
	case EncodedValueInt16:
		value, err = d.DecodeInt16()
	case EncodedValueInt32:
		value, err = d.DecodeInt32()
	case EncodedValueInt64:
		value, err = d.DecodeInt64()
	case EncodedValueInt128:
		value, err = d.DecodeInt128()
	case EncodedValueInt256:
		value, err = d.DecodeInt256()
	case EncodedValueUInt:
		value, err = d.DecodeUInt()
	case EncodedValueUInt8:
		value, err = d.DecodeUInt8()
	case EncodedValueUInt16:
		value, err = d.DecodeUInt16()
	case EncodedValueUInt32:
		value, err = d.DecodeUInt32()
	case EncodedValueUInt64:
		value, err = d.DecodeUInt64()
	case EncodedValueUInt128:
		value, err = d.DecodeUInt128()
	case EncodedValueUInt256:
		value, err = d.DecodeUInt256()
	case EncodedValueWord8:
		value, err = d.DecodeWord8()
	case EncodedValueWord16:
		value, err = d.DecodeWord16()
	case EncodedValueWord32:
		value, err = d.DecodeWord32()
	case EncodedValueWord64:
		value, err = d.DecodeWord64()
	case EncodedValueFix64:
		value, err = d.DecodeFix64()
	case EncodedValueUFix64:
		value, err = d.DecodeUFix64()
	case EncodedValueUntypedArray:
		value, err = d.DecodeUntypedArray()
	case EncodedValueVariableArray:
		var t cadence.VariableSizedArrayType
		t, err = d.DecodeVariableArrayType()
		if err != nil {
			return
		}
		value, err = d.DecodeVariableArray(t)
	case EncodedValueConstantArray:
		var t cadence.ConstantSizedArrayType
		t, err = d.DecodeConstantArrayType()
		if err != nil {
			return
		}
		value, err = d.DecodeConstantArray(t)
	case EncodedValueDictionary:
		value, err = d.DecodeDictionary()
	case EncodedValueStruct:
		value, err = d.DecodeStruct()
	case EncodedValueResource:
		value, err = d.DecodeResource()
	case EncodedValueEvent:
		value, err = d.DecodeEvent()
	case EncodedValueContract:
		value, err = d.DecodeContract()
	case EncodedValueLink:
		value, err = d.DecodeLink()
	case EncodedValuePath:
		value, err = d.DecodePath()
	case EncodedValueCapability:
		value, err = d.DecodeCapability()
	case EncodedValueEnum:
		value, err = d.DecodeEnum()

	default:
		err = common_codec.CodecError(fmt.Sprintf("unknown cadence.Value: %s", value))
	}

	return
}

func (e *Encoder) EncodeValueIdentifier(id EncodedValue) (err error) {
	return e.writeByte(byte(id))
}

func (d *Decoder) DecodeIdentifier() (id EncodedValue, err error) {
	b, err := d.read(1)
	if err != nil {
		return
	}

	id = EncodedValue(b[0])
	return
}
