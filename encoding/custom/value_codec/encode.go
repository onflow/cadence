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

package value_codec

import (
	"bytes"
	"fmt"
	"io"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/custom/common_codec"
)

// An Encoder converts Cadence values into custom-encoded bytes.
type Encoder struct {
	w common_codec.LengthyWriter
}

// EncodeValue returns the custom-encoded representation of the given value.
//
// This function returns an error if the Cadence value cannot be represented in the custom format.
func EncodeValue(value cadence.Value) ([]byte, error) {
	var w bytes.Buffer
	enc := NewEncoder(&w)

	err := enc.Encode(value)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MustEncode returns the custom-encoded representation of the given value, or panics
// if the value cannot be represented in the custom format.
func MustEncode(value cadence.Value) []byte {
	b, err := EncodeValue(value)
	if err != nil {
		panic(err)
	}
	return b
}

// NewEncoder initializes an Encoder that will write custom-encoded bytes to the
// given io.Writer.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: common_codec.NewLengthyWriter(w),
	}
}

// TODO include leading byte with version information

// Encode writes the custom-encoded representation of the given value to this
// encoder's io.Writer.
//
// This function returns an error if the given value's type is not supported
// by this encoder.
func (e *Encoder) Encode(value cadence.Value) (err error) {
	return e.EncodeValue(value)
}

//
// Values
//

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
	case cadence.Bytes:
		err = e.EncodeValueIdentifier(EncodedValueBytes)
		if err != nil {
			return
		}
		return common_codec.EncodeBytes(&e.w, v)
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
	case cadence.Word8: // TODO add test
		err = e.EncodeValueIdentifier(EncodedValueWord8)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, uint8(v))
	case cadence.Word16: // TODO add test
		err = e.EncodeValueIdentifier(EncodedValueWord16)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, uint16(v))
	case cadence.Word32: // TODO add test
		err = e.EncodeValueIdentifier(EncodedValueWord32)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, uint32(v))
	case cadence.Word64: // TODO add test
		err = e.EncodeValueIdentifier(EncodedValueWord64)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, uint64(v))
	case cadence.Fix64: // TODO add test
		err = e.EncodeValueIdentifier(EncodedValueFix64)
		if err != nil {
			return
		}
		return common_codec.EncodeNumber(&e.w, int64(v))
	case cadence.UFix64: // TODO add test
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
			err = fmt.Errorf("unknown array type: %s", arrayType)
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

	return fmt.Errorf("unexpected value: %s (type=%s)", value, value.Type())
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
			return fmt.Errorf("constant size array size=%d but has %d elements", v.Size, len(value.Values))
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
	// TODO if dictionary type is concrete for key or value, don't encode type info for them
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

//
// Types
//

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
	}

	return fmt.Errorf("unknown type: %s", t)
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

//
// Other
//

func (e *Encoder) write(b []byte) (err error) {
	_, err = e.w.Write(b)
	return
}

func (e *Encoder) writeByte(b byte) (err error) {
	_, err = e.w.Write([]byte{b})
	return
}

func EncodeArray[T any](e *Encoder, arr []T, encodeFn func(T) error) (err error) {
	// TODO save a bit in the array length for nil check?
	err = common_codec.EncodeBool(&e.w, arr == nil)
	if arr == nil || err != nil {
		return
	}

	err = common_codec.EncodeLength(&e.w, len(arr))
	if err != nil {
		return
	}

	for _, element := range arr {
		// TODO does this need to include pointer logic for recursive types in arrays to be handled correctly?
		err = encodeFn(element)
		if err != nil {
			return
		}
	}

	return
}

func DecodeArray[T any](d *Decoder, decodeFn func() (T, error)) (arr []T, err error) {
	isNil, err := common_codec.DecodeBool(&d.r)
	if isNil || err != nil {
		return
	}

	length, err := common_codec.DecodeLength(&d.r)
	if err != nil {
		return
	}

	arr = make([]T, length)
	for i := 0; i < length; i++ {
		var element T
		element, err = decodeFn()
		if err != nil {
			return
		}

		arr[i] = element
	}

	return
}
