package cbf_codec

import (
	"fmt"
	"math/big"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/cbf/common_codec"
	"github.com/onflow/cadence/runtime/common"
)

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
	case EncodedValueBytes:
		value, err = d.DecodeBytes()
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

func (d *Decoder) DecodeIdentifier() (id EncodedValue, err error) {
	b, err := d.read(1)
	if err != nil {
		return
	}

	id = EncodedValue(b[0])
	return
}

func (d *Decoder) DecodeVoid() (value cadence.Void, err error) {
	_, err = d.read(1)
	value = cadence.NewMeteredVoid(d.memoryGauge)
	return
}

func (d *Decoder) DecodeOptional() (value cadence.Optional, err error) {
	isNil, err := d.DecodeBool()
	if isNil || err != nil {
		return
	}

	innerValue, err := d.DecodeValue()
	value = cadence.NewMeteredOptional(d.memoryGauge, innerValue)
	return
}

func (d *Decoder) DecodeBool() (value cadence.Bool, err error) {
	boolean, err := common_codec.DecodeBool(&d.r)
	if err != nil {
		return
	}

	value = cadence.NewMeteredBool(d.memoryGauge, boolean)
	return
}

func (d *Decoder) DecodeString() (value cadence.String, err error) {
	s, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	value, err = cadence.NewMeteredString(
		d.memoryGauge,
		common.NewCadenceStringMemoryUsage(len(s)),
		func() string {
			return s
		},
	)
	return
}

func (d *Decoder) DecodeCharacter() (value cadence.Character, err error) {
	s, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	value, err = cadence.NewMeteredCharacter(
		d.memoryGauge,
		common.NewCadenceStringMemoryUsage(len(s)),
		func() string {
			return s
		},
	)
	return
}

func (d *Decoder) DecodeAddress() (value cadence.Address, err error) {
	address, err := common_codec.DecodeAddress(&d.r)
	if err != nil {
		return
	}

	value = cadence.NewMeteredAddress(
		d.memoryGauge,
		address,
	)
	return
}

func (d *Decoder) DecodeInt() (value cadence.Int, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	value = cadence.NewMeteredIntFromBig(
		d.memoryGauge,
		common.NewBigIntMemoryUsage(common.BigIntByteLength(i)),
		func() *big.Int {
			return i
		},
	)
	return
}

func (d *Decoder) DecodeInt8() (value cadence.Int8, err error) {
	i, err := common_codec.DecodeNumber[int8](&d.r)
	value = cadence.Int8(i)
	return
}

func (d *Decoder) DecodeInt16() (value cadence.Int16, err error) {
	i, err := common_codec.DecodeNumber[int16](&d.r)
	value = cadence.Int16(i)
	return
}

func (d *Decoder) DecodeInt32() (value cadence.Int32, err error) {
	i, err := common_codec.DecodeNumber[int32](&d.r)
	value = cadence.Int32(i)
	return
}

func (d *Decoder) DecodeInt64() (value cadence.Int64, err error) {
	i, err := common_codec.DecodeNumber[int64](&d.r)
	value = cadence.Int64(i)
	return
}

func (d *Decoder) DecodeInt128() (value cadence.Int128, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredInt128FromBig(
		d.memoryGauge,
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeInt256() (value cadence.Int256, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredInt256FromBig(
		d.memoryGauge,
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeUInt() (value cadence.UInt, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredUIntFromBig(
		d.memoryGauge,
		common.NewBigIntMemoryUsage(common.BigIntByteLength(i)),
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeUInt8() (value cadence.UInt8, err error) {
	i, err := common_codec.DecodeNumber[uint8](&d.r)
	value = cadence.UInt8(i)
	return
}

func (d *Decoder) DecodeUInt16() (value cadence.UInt16, err error) {
	i, err := common_codec.DecodeNumber[uint16](&d.r)
	value = cadence.UInt16(i)
	return
}

func (d *Decoder) DecodeUInt32() (value cadence.UInt32, err error) {
	i, err := common_codec.DecodeNumber[uint32](&d.r)
	value = cadence.UInt32(i)
	return
}

func (d *Decoder) DecodeUInt64() (value cadence.UInt64, err error) {
	i, err := common_codec.DecodeNumber[uint64](&d.r)
	value = cadence.UInt64(i)
	return
}

func (d *Decoder) DecodeUInt128() (value cadence.UInt128, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredUInt128FromBig(
		d.memoryGauge,
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeUInt256() (value cadence.UInt256, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredUInt256FromBig(
		d.memoryGauge,
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeWord8() (value cadence.Word8, err error) {
	i, err := common_codec.DecodeNumber[uint8](&d.r)
	if err != nil {
		return
	}
	value = cadence.Word8(i)
	return
}

func (d *Decoder) DecodeWord16() (value cadence.Word16, err error) {
	i, err := common_codec.DecodeNumber[uint16](&d.r)
	if err != nil {
		return
	}
	value = cadence.Word16(i)
	return
}

func (d *Decoder) DecodeWord32() (value cadence.Word32, err error) {
	i, err := common_codec.DecodeNumber[uint32](&d.r)
	if err != nil {
		return
	}
	value = cadence.Word32(i)
	return
}

func (d *Decoder) DecodeWord64() (value cadence.Word64, err error) {
	i, err := common_codec.DecodeNumber[uint64](&d.r)
	if err != nil {
		return
	}
	value = cadence.Word64(i)
	return
}

func (d *Decoder) DecodeFix64() (value cadence.Fix64, err error) {
	i, err := common_codec.DecodeNumber[int64](&d.r)
	if err != nil {
		return
	}
	value = cadence.Fix64(i)
	return
}

func (d *Decoder) DecodeUFix64() (value cadence.UFix64, err error) {
	i, err := common_codec.DecodeNumber[uint64](&d.r)
	if err != nil {
		return
	}
	value = cadence.UFix64(i)
	return
}

func (d *Decoder) DecodeBytes() (value cadence.Bytes, err error) {
	s, err := common_codec.DecodeBytes(&d.r)
	if err != nil {
		return
	}

	value = cadence.NewBytes(s)
	return
}

func (d *Decoder) DecodeUntypedArray() (array cadence.Array, err error) {
	size, err := common_codec.DecodeLength(&d.r)
	if err != nil {
		return
	}
	return d.decodeArray(nil, size)
}

func (d *Decoder) DecodeVariableArray(arrayType cadence.VariableSizedArrayType) (array cadence.Array, err error) {
	size, err := common_codec.DecodeLength(&d.r)
	if err != nil {
		return
	}
	return d.decodeArray(arrayType, size)
}

func (d *Decoder) DecodeConstantArray(arrayType cadence.ConstantSizedArrayType) (array cadence.Array, err error) {
	size := int(arrayType.Size)
	return d.decodeArray(arrayType, size)
}

func (d *Decoder) decodeArray(arrayType cadence.ArrayType, size int) (array cadence.Array, err error) {
	array, err = cadence.NewMeteredArray(d.memoryGauge, size, func() (elements []cadence.Value, err error) {
		elements = make([]cadence.Value, 0, size)
		for i := 0; i < size; i++ {
			// TODO if `elementType` is concrete then each element needn't encode its type
			var elementValue cadence.Value
			elementValue, err = d.DecodeValue()
			if err != nil {
				return
			}
			elements = append(elements, elementValue)
		}

		return elements, nil
	})

	array = array.WithType(arrayType)

	return
}

func (d *Decoder) DecodeDictionary() (dict cadence.Dictionary, err error) {
	dictType, err := d.DecodeDictionaryType()
	if err != nil {
		return
	}

	size, err := common_codec.DecodeLength(&d.r)
	if err != nil {
		return
	}

	dict, err = cadence.NewMeteredDictionary(d.memoryGauge, size, func() (pairs []cadence.KeyValuePair, err error) {
		pairs = make([]cadence.KeyValuePair, 0, size)
		var key, value cadence.Value
		for i := 0; i < size; i++ {
			key, err = d.DecodeValue()
			if err != nil {
				return
			}
			value, err = d.DecodeValue()
			if err != nil {
				return
			}
			pairs = append(pairs, cadence.NewMeteredKeyValuePair(d.memoryGauge, key, value))
		}
		return
	})

	dict = dict.WithType(dictType)

	return
}

func (d *Decoder) DecodeStruct() (s cadence.Struct, err error) {
	structType, err := d.DecodeStructType()
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (cadence.Value, error) {
		return d.DecodeValue()
	})
	if err != nil {
		return
	}

	s, err = cadence.NewMeteredStruct(
		d.memoryGauge,
		len(fields),
		func() ([]cadence.Value, error) {
			return fields, nil
		},
	)
	if err != nil {
		return
	}

	s = s.WithType(structType)
	return
}

func (d *Decoder) DecodeResource() (s cadence.Resource, err error) {
	resourceType, err := d.DecodeResourceType()
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (cadence.Value, error) {
		return d.DecodeValue()
	})
	if err != nil {
		return
	}

	s, err = cadence.NewMeteredResource(
		d.memoryGauge,
		len(fields),
		func() ([]cadence.Value, error) {
			return fields, nil
		},
	)
	if err != nil {
		return
	}

	s = s.WithType(resourceType)
	return
}

func (d *Decoder) DecodeEvent() (s cadence.Event, err error) {
	eventType, err := d.DecodeEventType()
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (cadence.Value, error) {
		return d.DecodeValue()
	})
	if err != nil {
		return
	}

	s, err = cadence.NewMeteredEvent(
		d.memoryGauge,
		len(fields),
		func() ([]cadence.Value, error) {
			return fields, nil
		},
	)
	if err != nil {
		return
	}

	s = s.WithType(eventType)
	return
}

func (d *Decoder) DecodeContract() (s cadence.Contract, err error) {
	contractType, err := d.DecodeContractType()
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (cadence.Value, error) {
		return d.DecodeValue()
	})
	if err != nil {
		return
	}

	s, err = cadence.NewMeteredContract(
		d.memoryGauge,
		len(fields),
		func() ([]cadence.Value, error) {
			return fields, nil
		},
	)
	if err != nil {
		return
	}

	s = s.WithType(contractType)
	return
}

func (d *Decoder) DecodeLink() (link cadence.Link, err error) {
	path, err := d.DecodePath()
	if err != nil {
		return
	}

	borrowType, err := d.DecodeString()
	if err != nil {
		return
	}

	link = cadence.NewMeteredLink(d.memoryGauge, path, string(borrowType))
	return
}

func (d *Decoder) DecodePath() (path cadence.Path, err error) {
	domain, err := d.DecodeString()
	if err != nil {
		return
	}

	identifier, err := d.DecodeString()
	if err != nil {
		return
	}

	path = cadence.NewMeteredPath(d.memoryGauge, string(domain), string(identifier))
	return
}

func (d *Decoder) DecodeCapability() (capability cadence.Capability, err error) {
	path, err := d.DecodePath()
	if err != nil {
		return
	}

	address, err := d.DecodeAddress()
	if err != nil {
		return
	}

	borrowType, err := d.DecodeType()
	if err != nil {
		return
	}

	capability = cadence.NewMeteredCapability(d.memoryGauge, path, address, borrowType)
	return
}

func (d *Decoder) DecodeEnum() (s cadence.Enum, err error) {
	enumType, err := d.DecodeEnumType()
	if err != nil {
		return
	}

	fields, err := DecodeArray(d, func() (cadence.Value, error) {
		return d.DecodeValue()
	})
	if err != nil {
		return
	}

	s, err = cadence.NewMeteredEnum(
		d.memoryGauge,
		len(fields),
		func() ([]cadence.Value, error) {
			return fields, nil
		},
	)
	if err != nil {
		return
	}

	s = s.WithType(enumType)
	return
}
