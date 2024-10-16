/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package interpreter

import (
	"fmt"
	"math"
	"math/big"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

var CBORDecMode = func() cbor.DecMode {
	decMode, err := cbor.DecOptions{
		IntDec:           cbor.IntDecConvertNone,
		MaxArrayElements: math.MaxInt64,
		MaxMapPairs:      math.MaxInt64,
		MaxNestedLevels:  math.MaxInt16,
	}.DecMode()
	if err != nil {
		panic(err)
	}
	return decMode
}()

type UnsupportedTagDecodingError struct {
	Tag uint64
}

var _ errors.InternalError = UnsupportedTagDecodingError{}

func (UnsupportedTagDecodingError) IsInternalError() {}

func (e UnsupportedTagDecodingError) Error() string {
	return fmt.Sprintf(
		"internal error: unsupported decoded tag: %d",
		e.Tag,
	)
}

type InvalidStringLengthError struct {
	Length uint64
}

var _ errors.InternalError = InvalidStringLengthError{}

func (InvalidStringLengthError) IsInternalError() {}

func (e InvalidStringLengthError) Error() string {
	return fmt.Sprintf(
		"internal error: invalid string length: got %d, expected max %d",
		e.Length,
		goMaxInt,
	)
}

func decodeCharacter(dec *cbor.StreamDecoder, memoryGauge common.MemoryGauge) (string, error) {
	length, err := dec.NextSize()
	if err != nil {
		return "", err
	}
	if length > goMaxInt {
		return "", InvalidStringLengthError{
			Length: length,
		}
	}

	common.UseMemory(memoryGauge, common.NewCharacterMemoryUsage(int(length)))
	return dec.DecodeString()
}

func decodeString(dec *cbor.StreamDecoder, memoryGauge common.MemoryGauge, stringKind common.MemoryKind) (string, error) {
	length, err := dec.NextSize()
	if err != nil {
		return "", err
	}
	if length > goMaxInt {
		return "", InvalidStringLengthError{
			Length: length,
		}
	}

	common.UseMemory(memoryGauge, common.MemoryUsage{
		Kind: stringKind,
		// + 1 to account for empty string
		Amount: length + 1,
	})

	return dec.DecodeString()
}

func decodeInt64(d StorableDecoder) (int64, error) {
	common.UseMemory(d.memoryGauge, Int64MemoryUsage)
	return d.decoder.DecodeInt64()
}

func DecodeStorable(
	decoder *cbor.StreamDecoder,
	slabID atree.SlabID,
	inlinedExtraData []atree.ExtraData,
	memoryGauge common.MemoryGauge,
) (
	atree.Storable,
	error,
) {
	return NewStorableDecoder(decoder, slabID, inlinedExtraData, memoryGauge).decodeStorable()
}

func newStorableDecoderFunc(memoryGauge common.MemoryGauge) atree.StorableDecoder {
	return func(
		decoder *cbor.StreamDecoder,
		slabID atree.SlabID,
		inlinedExtraData []atree.ExtraData,
	) (
		atree.Storable,
		error,
	) {
		return NewStorableDecoder(decoder, slabID, inlinedExtraData, memoryGauge).decodeStorable()
	}
}

func NewStorableDecoder(
	decoder *cbor.StreamDecoder,
	slabID atree.SlabID,
	inlinedExtraData []atree.ExtraData,
	memoryGauge common.MemoryGauge,
) StorableDecoder {
	return StorableDecoder{
		decoder:          decoder,
		memoryGauge:      memoryGauge,
		slabID:           slabID,
		inlinedExtraData: inlinedExtraData,
		TypeDecoder: NewTypeDecoder(
			decoder,
			memoryGauge,
		),
	}
}

type StorableDecoder struct {
	TypeDecoder
	memoryGauge      common.MemoryGauge
	decoder          *cbor.StreamDecoder
	slabID           atree.SlabID
	inlinedExtraData []atree.ExtraData
}

func (d StorableDecoder) decodeStorable() (atree.Storable, error) {
	var storable atree.Storable
	var err error

	t, err := d.decoder.NextType()
	if err != nil {
		return nil, err
	}

	switch t {

	// CBOR Types

	case cbor.BoolType:
		v, err := d.decoder.DecodeBool()
		if err != nil {
			return nil, err
		}
		storable = AsBoolValue(v)

	case cbor.NilType:
		err := d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
		storable = NilStorable

	case cbor.TextStringType:
		str, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
		if err != nil {
			return nil, err
		}
		// already metered by decodeString
		storable = StringAtreeValue(str)

	case cbor.UintType:
		n, err := decodeUint64(d.decoder, d.memoryGauge)
		if err != nil {
			return nil, err
		}
		storable = Uint64AtreeValue(n)

	case cbor.TagType:
		var num uint64
		num, err = d.decoder.DecodeTagNumber()
		if err != nil {
			return nil, err
		}

		switch num {

		case atree.CBORTagSlabID:
			return atree.DecodeSlabIDStorable(d.decoder)

		case atree.CBORTagInlinedArray:
			return atree.DecodeInlinedArrayStorable(
				d.decoder,
				newStorableDecoderFunc(d.memoryGauge),
				d.slabID,
				d.inlinedExtraData)

		case atree.CBORTagInlinedMap:
			return atree.DecodeInlinedMapStorable(
				d.decoder,
				newStorableDecoderFunc(d.memoryGauge),
				d.slabID,
				d.inlinedExtraData,
			)

		case atree.CBORTagInlinedCompactMap:
			return atree.DecodeInlinedCompactMapStorable(
				d.decoder,
				newStorableDecoderFunc(d.memoryGauge),
				d.slabID,
				d.inlinedExtraData,
			)

		case CBORTagVoidValue:
			err := d.decoder.Skip()
			if err != nil {
				return nil, err
			}
			storable = VoidStorable

		case CBORTagStringValue:
			storable, err = d.decodeStringValue()
			if err != nil {
				return nil, err
			}

		case CBORTagCharacterValue:
			storable, err = d.decodeCharacter()
			if err != nil {
				return nil, err
			}

		case CBORTagSomeValue:
			storable, err = d.decodeSome()

		case CBORTagSomeValueWithNestedLevels:
			storable, err = d.decodeSomeWithNestedLevels()

		case CBORTagAddressValue:
			storable, err = d.decodeAddress()

		// Int*

		case CBORTagIntValue:
			storable, err = d.decodeInt()

		case CBORTagInt8Value:
			storable, err = d.decodeInt8()

		case CBORTagInt16Value:
			storable, err = d.decodeInt16()

		case CBORTagInt32Value:
			storable, err = d.decodeInt32()

		case CBORTagInt64Value:
			storable, err = d.decodeInt64()

		case CBORTagInt128Value:
			storable, err = d.decodeInt128()

		case CBORTagInt256Value:
			storable, err = d.decodeInt256()

		// UInt*

		case CBORTagUIntValue:
			storable, err = d.decodeUInt()

		case CBORTagUInt8Value:
			storable, err = d.decodeUInt8()

		case CBORTagUInt16Value:
			storable, err = d.decodeUInt16()

		case CBORTagUInt32Value:
			storable, err = d.decodeUInt32()

		case CBORTagUInt64Value:
			storable, err = d.decodeUInt64()

		case CBORTagUInt128Value:
			storable, err = d.decodeUInt128()

		case CBORTagUInt256Value:
			storable, err = d.decodeUInt256()

		// Word*

		case CBORTagWord8Value:
			storable, err = d.decodeWord8()

		case CBORTagWord16Value:
			storable, err = d.decodeWord16()

		case CBORTagWord32Value:
			storable, err = d.decodeWord32()

		case CBORTagWord64Value:
			storable, err = d.decodeWord64()

		case CBORTagWord128Value:
			storable, err = d.decodeWord128()

		case CBORTagWord256Value:
			storable, err = d.decodeWord256()

		// Fix*

		case CBORTagFix64Value:
			storable, err = d.decodeFix64()

		// UFix*

		case CBORTagUFix64Value:
			storable, err = d.decodeUFix64()

		// Storage

		case CBORTagPathValue:
			storable, err = d.decodePath()

		case CBORTagCapabilityValue:
			storable, err = d.decodeCapability()

		case CBORTagPublishedValue:
			storable, err = d.decodePublishedValue()

		case CBORTagTypeValue:
			storable, err = d.decodeType()

		case CBORTagStorageCapabilityControllerValue:
			storable, err = d.decodeStorageCapabilityController()

		case CBORTagAccountCapabilityControllerValue:
			storable, err = d.decodeAccountCapabilityController()

		case CBORTagPathCapabilityValue:
			storable, err = d.decodePathCapability()

		case CBORTagPathLinkValue:
			storable, err = d.decodePathLink()

		case CBORTagAccountLinkValue:
			storable, err = d.decodeAccountLink()

		default:
			return nil, UnsupportedTagDecodingError{
				Tag: num,
			}
		}

	default:
		return nil, errors.NewUnexpectedError(
			"unsupported decoded CBOR type: %s",
			t.String(),
		)
	}

	if err != nil {
		return nil, err
	}

	return storable, nil
}

func (d StorableDecoder) decodeCharacter() (CharacterValue, error) {
	v, err := decodeCharacter(d.decoder, d.memoryGauge)
	if err != nil {
		if err, ok := err.(*cbor.WrongTypeError); ok {
			return CharacterValue{}, errors.NewUnexpectedError(
				"invalid Character encoding: %s",
				err.ActualType.String(),
			)
		}
		return CharacterValue{}, err
	}
	if !sema.IsValidCharacter(v) {
		return CharacterValue{}, errors.NewUnexpectedError(
			"invalid character encoding: %s",
			v,
		)
	}

	// NOTE: character value memory usage already metered by decodeCharacter,
	// but NewUnmeteredCharacterValue normalizes (= allocates)
	common.UseMemory(d.memoryGauge, common.NewRawStringMemoryUsage(len(v)))
	return NewUnmeteredCharacterValue(v), nil
}

func (d StorableDecoder) decodeStringValue() (*StringValue, error) {
	str, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindStringValue)
	if err != nil {
		if err, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid String encoding: %s",
				err.ActualType.String(),
			)
		}
		return nil, err
	}

	// NOTE: character value memory usage already metered by decodeString,
	// but NewUnmeteredStringValue normalizes (= allocates)
	common.UseMemory(d.memoryGauge, common.NewRawStringMemoryUsage(len(str)))
	return NewUnmeteredStringValue(str), nil
}

func decodeUint64(dec *cbor.StreamDecoder, memoryGauge common.MemoryGauge) (uint64, error) {
	common.UseMemory(memoryGauge, UInt64MemoryUsage)
	return dec.DecodeUint64()
}

func (d StorableDecoder) decodeBigInt() (*big.Int, error) {
	length, err := d.decoder.NextSize()
	if err != nil {
		return nil, err
	}

	common.UseMemory(d.memoryGauge, common.NewBigIntMemoryUsage(int(length)))

	return d.decoder.DecodeBigInt()
}

func (d StorableDecoder) decodeInt() (IntValue, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if err, ok := err.(*cbor.WrongTypeError); ok {
			return IntValue{}, errors.NewUnexpectedError(
				"invalid Int encoding: %s",
				err.ActualType.String(),
			)
		}
		return IntValue{}, err
	}

	// NOTE: already metered by decodeBigInt
	return NewUnmeteredIntValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeInt8() (Int8Value, error) {
	v, err := decodeInt64(d)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown Int8 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const min = math.MinInt8
	const max = math.MaxInt8

	if v < min {
		return 0, errors.NewUnexpectedError("invalid Int8: got %d, expected min %d", v, min)
	}

	if v > max {
		return 0, errors.NewUnexpectedError("invalid Int8: got %d, expected max %d", v, max)
	}

	// Already metered at `decodeInt64` function
	return NewUnmeteredInt8Value(int8(v)), nil
}

func (d StorableDecoder) decodeInt16() (Int16Value, error) {
	v, err := decodeInt64(d)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown Int16 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const min = math.MinInt16
	const max = math.MaxInt16

	if v < min {
		return 0, errors.NewUnexpectedError("invalid Int16: got %d, expected min %d", v, min)
	}

	if v > max {
		return 0, errors.NewUnexpectedError("invalid Int16: got %d, expected max %d", v, max)
	}

	// Already metered at `decodeInt64` function
	return NewUnmeteredInt16Value(int16(v)), nil
}

func (d StorableDecoder) decodeInt32() (Int32Value, error) {
	v, err := decodeInt64(d)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown Int32 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const min = math.MinInt32
	const max = math.MaxInt32

	if v < min {
		return 0, errors.NewUnexpectedError("invalid Int32: got %d, expected min %d", v, min)
	}
	if v > max {
		return 0, errors.NewUnexpectedError("invalid Int32: got %d, expected max %d", v, max)
	}

	// Already metered at `decodeInt64` function
	return NewUnmeteredInt32Value(int32(v)), nil
}

func (d StorableDecoder) decodeInt64() (Int64Value, error) {
	v, err := decodeInt64(d)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown Int64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	// Already metered at `decodeInt64` function
	return NewUnmeteredInt64Value(v), nil
}

func (d StorableDecoder) decodeInt128() (Int128Value, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return Int128Value{}, errors.NewUnexpectedError("invalid Int128 encoding: %s", e.ActualType.String())
		}
		return Int128Value{}, err
	}

	min := sema.Int128TypeMinIntBig
	if bigInt.Cmp(min) < 0 {
		return Int128Value{}, errors.NewUnexpectedError("invalid Int128: got %s, expected min %s", bigInt, min)
	}

	max := sema.Int128TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return Int128Value{}, errors.NewUnexpectedError("invalid Int128: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredInt128ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeInt256() (Int256Value, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return Int256Value{}, errors.NewUnexpectedError("invalid Int256 encoding: %s", e.ActualType.String())
		}
		return Int256Value{}, err
	}

	min := sema.Int256TypeMinIntBig
	if bigInt.Cmp(min) < 0 {
		return Int256Value{}, errors.NewUnexpectedError("invalid Int256: got %s, expected min %s", bigInt, min)
	}

	max := sema.Int256TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return Int256Value{}, errors.NewUnexpectedError("invalid Int256: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredInt256ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeUInt() (UIntValue, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return UIntValue{}, errors.NewUnexpectedError("invalid UInt encoding: %s", e.ActualType.String())
		}
		return UIntValue{}, err
	}

	if bigInt.Sign() < 0 {
		return UIntValue{}, errors.NewUnexpectedError("invalid UInt: got %s, expected positive", bigInt)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredUIntValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeUInt8() (UInt8Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown UInt8 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	const max = math.MaxUint8
	if value > max {
		return 0, errors.NewUnexpectedError("invalid UInt8: got %d, expected max %d", value, max)
	}
	// NOTE: already metered by `decodeUint64`
	return NewUnmeteredUInt8Value(uint8(value)), nil
}

func (d StorableDecoder) decodeUInt16() (UInt16Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown UInt16 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const max = math.MaxUint16
	if value > max {
		return 0, errors.NewUnexpectedError("invalid UInt16: got %d, expected max %d", value, max)
	}
	// NOTE: already metered by `decodeUint64`
	return NewUnmeteredUInt16Value(uint16(value)), nil
}

func (d StorableDecoder) decodeUInt32() (UInt32Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown UInt32 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const max = math.MaxUint32
	if value > max {
		return 0, errors.NewUnexpectedError("invalid UInt32: got %d, expected max %d", value, max)
	}
	// NOTE: already metered by `decodeUint64`
	return NewUnmeteredUInt32Value(uint32(value)), nil
}

func (d StorableDecoder) decodeUInt64() (UInt64Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown UInt64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	// NOTE: already metered by `decodeUint64`
	return NewUnmeteredUInt64Value(value), nil
}

func (d StorableDecoder) decodeUInt128() (UInt128Value, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return UInt128Value{}, errors.NewUnexpectedError("invalid UInt128 encoding: %s", e.ActualType.String())
		}
		return UInt128Value{}, err
	}

	if bigInt.Sign() < 0 {
		return UInt128Value{}, errors.NewUnexpectedError("invalid UInt128: got %s, expected positive", bigInt)
	}

	max := sema.UInt128TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return UInt128Value{}, errors.NewUnexpectedError("invalid UInt128: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredUInt128ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeUInt256() (UInt256Value, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return UInt256Value{}, errors.NewUnexpectedError("invalid UInt256 encoding: %s", e.ActualType.String())
		}
		return UInt256Value{}, err
	}

	if bigInt.Sign() < 0 {
		return UInt256Value{}, errors.NewUnexpectedError("invalid UInt256: got %s, expected positive", bigInt)
	}

	max := sema.UInt256TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return UInt256Value{}, errors.NewUnexpectedError("invalid UInt256: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by decodeBigInta
	return NewUnmeteredUInt256ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeWord8() (Word8Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown Word8 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	const max = math.MaxUint8
	if value > max {
		return 0, errors.NewUnexpectedError("invalid Word8: got %d, expected max %d", value, max)
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredWord8Value(uint8(value)), nil
}

func (d StorableDecoder) decodeWord16() (Word16Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown Word16 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	const max = math.MaxUint16
	if value > max {
		return 0, errors.NewUnexpectedError("invalid Word16: got %d, expected max %d", value, max)
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredWord16Value(uint16(value)), nil
}

func (d StorableDecoder) decodeWord32() (Word32Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown Word32 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	const max = math.MaxUint32
	if value > max {
		return 0, errors.NewUnexpectedError("invalid Word32: got %d, expected max %d", value, max)
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredWord32Value(uint32(value)), nil
}

func (d StorableDecoder) decodeWord64() (Word64Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown Word64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredWord64Value(value), nil
}

func (d StorableDecoder) decodeWord128() (Word128Value, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return Word128Value{}, errors.NewUnexpectedError("invalid Word128 encoding: %s", e.ActualType.String())
		}
		return Word128Value{}, err
	}

	if bigInt.Sign() < 0 {
		return Word128Value{}, errors.NewUnexpectedError("invalid Word128: got %s, expected positive", bigInt)
	}

	max := sema.Word128TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return Word128Value{}, errors.NewUnexpectedError("invalid Word128: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredWord128ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeWord256() (Word256Value, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return Word256Value{}, errors.NewUnexpectedError("invalid Word256 encoding: %s", e.ActualType.String())
		}
		return Word256Value{}, err
	}

	if bigInt.Sign() < 0 {
		return Word256Value{}, errors.NewUnexpectedError("invalid Word256: got %s, expected positive", bigInt)
	}

	max := sema.Word256TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return Word256Value{}, errors.NewUnexpectedError("invalid Word256: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredWord256ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeFix64() (Fix64Value, error) {
	value, err := decodeInt64(d)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown Fix64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	// Already metered at `decodeInt64`
	return NewUnmeteredFix64Value(value), nil
}

func (d StorableDecoder) decodeUFix64() (UFix64Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, errors.NewUnexpectedError("unknown UFix64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredUFix64Value(value), nil
}

func (d StorableDecoder) decodeSome() (SomeStorable, error) {
	storable, err := d.decodeStorable()
	if err != nil {
		return SomeStorable{}, errors.NewUnexpectedError(
			"invalid some value encoding: %w",
			err,
		)
	}

	return SomeStorable{
		gauge:    d.memoryGauge,
		Storable: storable,
	}, nil
}

func (d StorableDecoder) decodeSomeWithNestedLevels() (SomeStorable, error) {
	count, err := d.decoder.DecodeArrayHead()
	if err != nil {
		return SomeStorable{}, errors.NewUnexpectedError(
			"invalid some value with nested levels encoding: %w",
			err,
		)
	}

	if count != someStorableWithMultipleNestedLevelsArrayCount {
		return SomeStorable{}, errors.NewUnexpectedError(
			"invalid array count for some value with nested levels encoding: got %d, expect %d",
			count, someStorableWithMultipleNestedLevelsArrayCount,
		)
	}

	nestedLevels, err := d.decoder.DecodeUint64()
	if err != nil {
		return SomeStorable{}, errors.NewUnexpectedError(
			"invalid nested levels for some value with nested levels encoding: %w",
			err,
		)
	}

	if nestedLevels <= 1 {
		return SomeStorable{}, errors.NewUnexpectedError(
			"invalid nested levels for some value with nested levels encoding: got %d, expect > 1",
			nestedLevels,
		)
	}

	nonSomeStorable, err := d.decodeStorable()
	if err != nil {
		return SomeStorable{}, errors.NewUnexpectedError(
			"invalid nonSomeStorable for some value with nested levels encoding: %w",
			err,
		)
	}

	storable := SomeStorable{
		gauge:    d.memoryGauge,
		Storable: nonSomeStorable,
	}
	for i := uint64(1); i < nestedLevels; i++ {
		storable = SomeStorable{
			gauge:    d.memoryGauge,
			Storable: storable,
		}
	}

	return storable, nil
}

func checkEncodedAddressLength(actualLength int) error {
	const expectedLength = common.AddressLength
	if actualLength > expectedLength {
		return errors.NewUnexpectedError(
			"invalid address length: got %d, expected max %d",
			actualLength,
			expectedLength,
		)
	}
	return nil
}

func (d StorableDecoder) decodeAddress() (AddressValue, error) {
	addressBytes, err := d.decodeAddressBytes()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return AddressValue{}, errors.NewUnexpectedError(
				"invalid address encoding: %s",
				e.ActualType.String(),
			)
		}
		return AddressValue{}, err
	}

	// Already metered at `decodeAddressBytes()`
	return NewUnmeteredAddressValueFromBytes(addressBytes), nil
}

func (d LocationDecoder) decodeAddressBytes() ([]byte, error) {
	// Check the address length and validate before decoding.
	length, err := d.decoder.NextSize()
	if err != nil {
		return nil, err
	}

	lengthErr := checkEncodedAddressLength(int(length))
	if lengthErr != nil {
		return nil, lengthErr
	}

	common.UseMemory(d.memoryGauge, common.AddressValueMemoryUsage)

	return d.decoder.DecodeBytes()
}

func (d StorableDecoder) decodePath() (PathValue, error) {

	const expectedLength = encodedPathValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			// No need to meter EmptyPathValue here or below because it's ignored for the error
			return EmptyPathValue, errors.NewUnexpectedError(
				"invalid path encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return EmptyPathValue, err
	}

	if size != expectedLength {
		return EmptyPathValue, errors.NewUnexpectedError(
			"invalid path encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// Decode domain at array index encodedPathValueDomainFieldKey
	domain, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return EmptyPathValue, errors.NewUnexpectedError(
				"invalid path domain encoding: %s",
				e.ActualType.String(),
			)
		}
		return EmptyPathValue, err
	}

	// Decode identifier at array index encodedPathValueIdentifierFieldKey
	identifier, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return EmptyPathValue, errors.NewUnexpectedError(
				"invalid path identifier encoding: %s",
				e.ActualType.String(),
			)
		}
		return EmptyPathValue, err
	}

	return NewPathValue(
		d.memoryGauge,
		common.PathDomain(domain),
		identifier,
	), nil
}

func (d StorableDecoder) decodeCapability() (*IDCapabilityValue, error) {

	const expectedLength = encodedCapabilityValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid capability encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid capability encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// address

	// Decode address at array index encodedCapabilityValueAddressFieldKey
	var num uint64
	num, err = d.decoder.DecodeTagNumber()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid capability address: %w",
			err,
		)
	}
	if num != CBORTagAddressValue {
		return nil, errors.NewUnexpectedError(
			"invalid capability address: wrong tag %d",
			num,
		)
	}
	address, err := d.decodeAddress()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid capability address: %w",
			err,
		)
	}

	// Decode ID at array index encodedCapabilityValueIDFieldKey

	id, err := d.decoder.DecodeUint64()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid capability ID: %w",
			err,
		)
	}

	// Decode borrow type at array index encodedCapabilityValueBorrowTypeFieldKey

	borrowType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid capability borrow type encoding: %w", err)
	}

	return NewCapabilityValue(
		d.memoryGauge,
		UInt64Value(id),
		address,
		borrowType,
	), nil
}

func (d StorableDecoder) decodeStorageCapabilityController() (*StorageCapabilityControllerValue, error) {

	const expectedLength = encodedStorageCapabilityControllerValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid storage capability controller encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid storage capability controller encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// Decode borrow type at array index encodedStorageCapabilityControllerValueBorrowTypeFieldKey
	borrowStaticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid storage capability controller borrow type encoding: %w", err)
	}
	borrowReferenceStaticType, ok := borrowStaticType.(*ReferenceStaticType)
	if !ok {
		return nil, errors.NewUnexpectedError(
			"invalid storage capability controller borrow type encoding: expected reference static type, got %T",
			borrowStaticType,
		)
	}

	// Decode capability ID at array index encodedStorageCapabilityControllerValueCapabilityIDFieldKey

	capabilityID, err := d.decoder.DecodeUint64()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid storage capability controller capability ID: %w",
			err,
		)
	}

	// Decode path at array index encodedStorageCapabilityControllerValueTargetPathFieldKey
	num, err := d.decoder.DecodeTagNumber()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid storage capability controller target path encoding: %w", err)
	}
	if num != CBORTagPathValue {
		return nil, errors.NewUnexpectedError(
			"invalid storage capability controller target path encoding: expected CBOR tag %d, got %d",
			CBORTagPathValue,
			num,
		)
	}
	pathValue, err := d.decodePath()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid storage capability controller target path encoding: %w", err)
	}

	return NewStorageCapabilityControllerValue(
		d.memoryGauge,
		borrowReferenceStaticType,
		UInt64Value(capabilityID),
		pathValue,
	), nil
}

func (d StorableDecoder) decodeAccountCapabilityController() (*AccountCapabilityControllerValue, error) {

	const expectedLength = encodedAccountCapabilityControllerValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid account capability controller encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid account capability controller encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// Decode borrow type at array index encodedAccountCapabilityControllerValueBorrowTypeFieldKey
	borrowStaticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid account capability controller borrow type encoding: %w", err)
	}
	borrowReferenceStaticType, ok := borrowStaticType.(*ReferenceStaticType)
	if !ok {
		return nil, errors.NewUnexpectedError(
			"invalid account capability controller borrow type encoding: expected reference static type, got %T",
			borrowStaticType,
		)
	}

	// Decode capability ID at array index encodedAccountCapabilityControllerValueCapabilityIDFieldKey

	capabilityID, err := d.decoder.DecodeUint64()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid account capability controller capability ID: %w",
			err,
		)
	}

	return NewAccountCapabilityControllerValue(
		d.memoryGauge,
		borrowReferenceStaticType,
		UInt64Value(capabilityID),
	), nil
}

func (d StorableDecoder) decodePublishedValue() (*PublishedValue, error) {

	const expectedLength = encodedPublishedValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid published value encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid published value encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// Decode address at array index encodedPublishedValueRecipientFieldKey
	num, err := d.decoder.DecodeTagNumber()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid published value recipient encoding: %w", err)
	}
	if num != CBORTagAddressValue {
		return nil, errors.NewUnexpectedError(
			"invalid published value recipient encoding: expected CBOR tag %d, got %d",
			CBORTagAddressValue,
			num,
		)
	}
	addressValue, err := d.decodeAddress()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid published value recipient encoding: %w", err)
	}

	// Decode value at array index encodedPublishedValueValueFieldKey
	value, err := d.decodeStorable()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid published value value encoding: %w", err)
	}

	capabilityValue, ok := value.(CapabilityValue)
	if !ok {
		return nil, errors.NewUnexpectedError(
			"invalid published value value encoding: expected capability, got %T",
			value,
		)
	}

	return NewPublishedValue(d.memoryGauge, addressValue, capabilityValue), nil
}

func (d StorableDecoder) decodeType() (TypeValue, error) {
	const expectedLength = encodedTypeValueTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return EmptyTypeValue, errors.NewUnexpectedError(
				"invalid type encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		// unmetered here and below because value is tossed in decodeStorable when err != nil
		return EmptyTypeValue, err
	}

	if arraySize != expectedLength {
		return EmptyTypeValue, errors.NewUnexpectedError(
			"invalid type encoding: expected [%d]any, got [%d]any",
			expectedLength,
			arraySize,
		)
	}

	// Decode type at array index encodedTypeValueTypeFieldKey
	var staticType StaticType

	// Optional type can be CBOR nil.
	err = d.decoder.DecodeNil()
	if _, ok := err.(*cbor.WrongTypeError); ok {
		staticType, err = d.DecodeStaticType()
	}

	if err != nil {
		return EmptyTypeValue, errors.NewUnexpectedError("invalid type encoding: %w", err)
	}

	return NewTypeValue(d.memoryGauge, staticType), nil
}

// Deprecated: decodePathCapability
func (d StorableDecoder) decodePathCapability() (*PathCapabilityValue, error) {

	const expectedLength = encodedPathCapabilityValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid capability encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid capability encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// address

	// Decode address at array index encodedPathCapabilityValueAddressFieldKey
	var num uint64
	num, err = d.decoder.DecodeTagNumber()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid capability address: %w",
			err,
		)
	}
	if num != CBORTagAddressValue {
		return nil, errors.NewUnexpectedError(
			"invalid capability address: wrong tag %d",
			num,
		)
	}
	address, err := d.decodeAddress()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid capability address: %w",
			err,
		)
	}

	// path

	// Decode path at array index encodedPathCapabilityValuePathFieldKey
	pathStorable, err := d.decodeStorable()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid capability path: %w", err)
	}
	pathValue, ok := pathStorable.(PathValue)
	if !ok {
		return nil, errors.NewUnexpectedError("invalid capability path: invalid type %T", pathValue)
	}

	// Decode borrow type at array index encodedPathCapabilityValueBorrowTypeFieldKey

	// borrow type (optional, for backwards compatibility)
	// Capabilities used to be untyped, i.e. they didn't have a borrow type.
	// Later an optional type parameter, the borrow type, was added to it,
	// which specifies as what type the capability should be borrowed.
	//
	// The decoding must be backwards-compatible and support both capability values
	// with a borrow type and ones without

	var borrowType StaticType

	// Optional borrow type can be CBOR nil.
	err = d.decoder.DecodeNil()
	if _, ok := err.(*cbor.WrongTypeError); ok {
		borrowType, err = d.DecodeStaticType()
	}
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid capability borrow type encoding: %w", err)
	}

	return &PathCapabilityValue{
		address:    address,
		Path:       pathValue,
		BorrowType: borrowType,
	}, nil
}

// Deprecated: decodePathLink
func (d StorableDecoder) decodePathLink() (PathLinkValue, error) {

	const expectedLength = encodedPathLinkValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return EmptyPathLinkValue, errors.NewUnexpectedError(
				"invalid link encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return EmptyPathLinkValue, err
	}

	if size != expectedLength {
		return EmptyPathLinkValue, errors.NewUnexpectedError(
			"invalid link encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// Decode path at array index encodedPathLinkValueTargetPathFieldKey
	num, err := d.decoder.DecodeTagNumber()
	if err != nil {
		return EmptyPathLinkValue, errors.NewUnexpectedError("invalid link target path encoding: %w", err)
	}
	if num != CBORTagPathValue {
		return EmptyPathLinkValue, errors.NewUnexpectedError("invalid link target path encoding: expected CBOR tag %d, got %d", CBORTagPathValue, num)
	}
	pathValue, err := d.decodePath()
	if err != nil {
		return EmptyPathLinkValue, errors.NewUnexpectedError("invalid link target path encoding: %w", err)
	}

	// Decode type at array index encodedPathLinkValueTypeFieldKey
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return EmptyPathLinkValue, errors.NewUnexpectedError("invalid link type encoding: %w", err)
	}

	return PathLinkValue{
		Type:       staticType,
		TargetPath: pathValue,
	}, nil
}

// Deprecated: decodeAccountLink
func (d StorableDecoder) decodeAccountLink() (AccountLinkValue, error) {
	err := d.decoder.Skip()
	if err != nil {
		return AccountLinkValue{}, err
	}

	return AccountLinkValue{}, nil
}

type TypeDecoder struct {
	decoder     *cbor.StreamDecoder
	memoryGauge common.MemoryGauge
	LocationDecoder
}

func NewTypeDecoder(
	decoder *cbor.StreamDecoder,
	memoryGauge common.MemoryGauge,
) TypeDecoder {
	return TypeDecoder{
		decoder:     decoder,
		memoryGauge: memoryGauge,
		LocationDecoder: NewLocationDecoder(
			decoder,
			memoryGauge,
		),
	}
}

func (d TypeDecoder) DecodeStaticType() (StaticType, error) {
	number, err := d.decoder.DecodeTagNumber()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid static type encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	switch number {
	case CBORTagPrimitiveStaticType:
		return d.decodePrimitiveStaticType()

	case CBORTagOptionalStaticType:
		return d.decodeOptionalStaticType()

	case CBORTagCompositeStaticType:
		return d.decodeCompositeStaticType()

	case CBORTagInterfaceStaticType:
		return d.decodeInterfaceStaticType()

	case CBORTagVariableSizedStaticType:
		return d.decodeVariableSizedStaticType()

	case CBORTagConstantSizedStaticType:
		return d.decodeConstantSizedStaticType()

	case CBORTagReferenceStaticType:
		return d.decodeReferenceStaticType()

	case CBORTagDictionaryStaticType:
		return d.decodeDictionaryStaticType()

	case CBORTagIntersectionStaticType:
		return d.decodeIntersectionStaticType()

	case CBORTagCapabilityStaticType:
		return d.decodeCapabilityStaticType()

	case CBORTagInclusiveRangeStaticType:
		return d.decodeInclusiveRangeStaticType()

	default:
		return nil, errors.NewUnexpectedError("invalid static type encoding tag: %d", number)
	}
}

func (d TypeDecoder) decodePrimitiveStaticType() (PrimitiveStaticType, error) {
	encoded, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PrimitiveStaticTypeUnknown,
				errors.NewUnexpectedError("invalid primitive static type encoding: %s", e.ActualType.String())
		}
		return PrimitiveStaticTypeUnknown, err
	}
	return PrimitiveStaticType(encoded), nil
}

func (d TypeDecoder) decodeOptionalStaticType() (StaticType, error) {
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid optional static type inner type encoding: %w",
			err,
		)
	}
	return NewOptionalStaticType(
		d.memoryGauge,
		staticType,
	), nil
}

func (d TypeDecoder) decodeCompositeStaticType() (StaticType, error) {
	const expectedLength = encodedCompositeStaticTypeLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid composite static type encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid composite static type encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// Decode location at array index encodedCompositeStaticTypeLocationFieldKey
	location, err := d.DecodeLocation()
	if err != nil {
		return nil, errors.NewUnexpectedError("invalid composite static type location encoding: %w", err)
	}

	// Decode qualified identifier at array index encodedCompositeStaticTypeQualifiedIdentifierFieldKey
	qualifiedIdentifier, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid composite static type qualified identifier encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return NewCompositeStaticTypeComputeTypeID(d.memoryGauge, location, qualifiedIdentifier), nil
}

func (d TypeDecoder) decodeInterfaceStaticType() (*InterfaceStaticType, error) {
	const expectedLength = encodedInterfaceStaticTypeLength

	size, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid interface static type encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid interface static type encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// Decode location at array index encodedInterfaceStaticTypeLocationFieldKey
	location, err := d.DecodeLocation()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid interface static type location encoding: %w",
			err,
		)
	}

	// Decode qualified identifier at array index encodedInterfaceStaticTypeQualifiedIdentifierFieldKey
	qualifiedIdentifier, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid interface static type qualified identifier encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return NewInterfaceStaticTypeComputeTypeID(d.memoryGauge, location, qualifiedIdentifier), nil
}

func (d TypeDecoder) decodeVariableSizedStaticType() (*VariableSizedStaticType, error) {
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid variable-sized static type encoding: %w",
			err,
		)
	}
	return NewVariableSizedStaticType(d.memoryGauge, staticType), nil
}

func (d TypeDecoder) decodeConstantSizedStaticType() (*ConstantSizedStaticType, error) {

	const expectedLength = encodedConstantSizedStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid constant-sized static type encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid constant-sized static type encoding: expected [%d]any, got [%d]any",
			expectedLength,
			arraySize,
		)
	}

	// Decode size at array index encodedConstantSizedStaticTypeSizeFieldKey
	size, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid constant-sized static type size encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	const max = math.MaxInt64
	if size > max {
		return nil, errors.NewUnexpectedError(
			"invalid constant-sized static type size: got %d, expected max %d",
			size,
			max,
		)
	}

	// Decode type at array index encodedConstantSizedStaticTypeTypeFieldKey
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid constant-sized static type inner type encoding: %w",
			err,
		)
	}

	return NewConstantSizedStaticType(
		d.memoryGauge,
		staticType,
		int64(size),
	), nil
}

func (d TypeDecoder) decodeStaticAuthorization() (Authorization, error) {
	number, err := d.decoder.DecodeTagNumber()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid static authorization encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}
	switch number {
	case CBORTagUnauthorizedStaticAuthorization:
		err := d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
		return UnauthorizedAccess, nil
	case CBORTagInaccessibleStaticAuthorization:
		err := d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
		return InaccessibleAccess, nil
	case CBORTagEntitlementMapStaticAuthorization:
		typeID, err := d.decoder.DecodeString()
		if err != nil {
			return nil, err
		}
		return NewEntitlementMapAuthorization(d.memoryGauge, common.TypeID(typeID)), nil
	case CBORTagEntitlementSetStaticAuthorization:
		const expectedLength = encodedSetAuthorizationStaticTypeLength

		arraySize, err := d.decoder.DecodeArrayHead()

		if err != nil {
			if e, ok := err.(*cbor.WrongTypeError); ok {
				return nil, errors.NewUnexpectedError(
					"invalid set authorization type encoding: expected [%d]any, got %s",
					expectedLength,
					e.ActualType.String(),
				)
			}
			return nil, err
		}

		if arraySize != expectedLength {
			return nil, errors.NewUnexpectedError(
				"invalid set authorization type encoding: expected [%d]any, got [%d]any",
				expectedLength,
				arraySize,
			)
		}

		setKind, err := d.decoder.DecodeUint64()
		if err != nil {
			if e, ok := err.(*cbor.WrongTypeError); ok {
				return nil, errors.NewUnexpectedError(
					"invalid entitlement set static authorization encoding: %s",
					e.ActualType.String(),
				)
			}
			return nil, err
		}

		entitlementsSize, err := d.decoder.DecodeArrayHead()
		if err != nil {
			if e, ok := err.(*cbor.WrongTypeError); ok {
				return nil, errors.NewUnexpectedError(
					"invalid entitlement set static authorization encoding: %s",
					e.ActualType.String(),
				)
			}
			return nil, err
		}

		var setCreationErr error

		entitlementSet := NewEntitlementSetAuthorization(
			d.memoryGauge,
			func() (entitlements []common.TypeID) {
				if entitlementsSize > 0 {
					entitlements = make([]common.TypeID, entitlementsSize)
					for i := 0; i < int(entitlementsSize); i++ {
						typeID, err := d.decoder.DecodeString()
						if err != nil {
							setCreationErr = err
							return nil
						}
						entitlements[i] = common.TypeID(typeID)
					}
				}
				return
			},
			int(entitlementsSize),
			sema.EntitlementSetKind(setKind),
		)

		if setCreationErr != nil {
			return nil, setCreationErr
		}

		return entitlementSet, nil
	}
	return nil, errors.NewUnexpectedError("invalid static authorization encoding tag: %d", number)
}

func (d TypeDecoder) decodeReferenceStaticType() (StaticType, error) {
	const expectedLength = encodedReferenceStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid reference static type encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid reference static type encoding: expected [%d]any, got [%d]any",
			expectedLength,
			arraySize,
		)
	}

	var authorization Authorization

	t, err := d.decoder.NextType()
	if err != nil {
		return nil, err
	}

	var hasLegacyIsAuthorized bool
	var legacyIsAuthorized bool

	if t == cbor.BoolType {
		// if we saw a bool here, this is a reference encoded in the old format
		hasLegacyIsAuthorized = true

		legacyIsAuthorized, err = d.decoder.DecodeBool()
		if err != nil {
			return nil, err
		}

		authorization = UnauthorizedAccess
	} else {
		// Decode authorized at array index encodedReferenceStaticTypeAuthorizationFieldKey
		authorization, err = d.decodeStaticAuthorization()
		if err != nil {
			if e, ok := err.(*cbor.WrongTypeError); ok {
				return nil, errors.NewUnexpectedError(
					"invalid reference static type authorized encoding: %s",
					e.ActualType.String(),
				)
			}
			return nil, err
		}
	}

	// Decode type at array index encodedReferenceStaticTypeTypeFieldKey
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid reference static type inner type encoding: %w",
			err,
		)
	}

	referenceType := NewReferenceStaticType(
		d.memoryGauge,
		authorization,
		staticType,
	)

	referenceType.HasLegacyIsAuthorized = hasLegacyIsAuthorized
	referenceType.LegacyIsAuthorized = legacyIsAuthorized

	return referenceType, nil
}

func (d TypeDecoder) decodeDictionaryStaticType() (*DictionaryStaticType, error) {
	const expectedLength = encodedDictionaryStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid dictionary static type encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid dictionary static type encoding: expected [%d]any, got [%d]any",
			expectedLength,
			arraySize,
		)
	}

	// Decode key type at array index encodedDictionaryStaticTypeKeyTypeFieldKey
	keyType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid dictionary static type key type encoding: %w",
			err,
		)
	}

	// Decode value type at array index encodedDictionaryStaticTypeValueTypeFieldKey
	valueType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid dictionary static type value type encoding: %w",
			err,
		)
	}

	return NewDictionaryStaticType(d.memoryGauge, keyType, valueType), nil
}

func (d TypeDecoder) decodeIntersectionStaticType() (StaticType, error) {
	const expectedLength = encodedIntersectionStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid intersection static type encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, errors.NewUnexpectedError(
			"invalid intersection static type encoding: expected [%d]any, got [%d]any",
			expectedLength,
			arraySize,
		)
	}

	var legacyRestrictedType StaticType

	t, err := d.decoder.NextType()
	if err != nil {
		return nil, err
	}

	if t == cbor.NilType {
		err = d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
	} else {
		// Decode intersection type at array index encodedIntersectionStaticTypeLegacyTypeFieldKey
		legacyRestrictedType, err = d.DecodeStaticType()
		if err != nil {
			return nil, errors.NewUnexpectedError(
				"invalid intersection static type key type encoding: %w",
				err,
			)
		}
	}

	// Decode intersected types at array index encodedIntersectionStaticTypeTypesFieldKey
	intersectionSize, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid intersection static type intersections encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	var intersections []*InterfaceStaticType
	if intersectionSize > 0 {
		intersections = make([]*InterfaceStaticType, intersectionSize)
		for i := 0; i < int(intersectionSize); i++ {

			number, err := d.decoder.DecodeTagNumber()
			if err != nil {
				if e, ok := err.(*cbor.WrongTypeError); ok {
					return nil, errors.NewUnexpectedError(
						"invalid intersection static type intersection encoding: expected CBOR tag, got %s",
						e.ActualType.String(),
					)
				}
				return nil, errors.NewUnexpectedError(
					"invalid intersection static type intersection encoding: %w",
					err,
				)
			}

			if number != CBORTagInterfaceStaticType {
				return nil, errors.NewUnexpectedError(
					"invalid intersection static type intersection encoding: expected CBOR tag %d, got %d",
					CBORTagInterfaceStaticType,
					number,
				)
			}

			intersectedType, err := d.decodeInterfaceStaticType()
			if err != nil {
				return nil, errors.NewUnexpectedError(
					"invalid intersection static type intersection encoding: %w",
					err,
				)
			}

			intersections[i] = intersectedType
		}
	}

	staticType := NewIntersectionStaticType(
		d.memoryGauge,
		intersections,
	)
	staticType.LegacyType = legacyRestrictedType

	return staticType, nil
}

func (d TypeDecoder) decodeCapabilityStaticType() (StaticType, error) {
	var borrowStaticType StaticType

	// Optional borrow type can be CBOR nil.
	err := d.decoder.DecodeNil()
	if _, ok := err.(*cbor.WrongTypeError); ok {
		borrowStaticType, err = d.DecodeStaticType()
	}

	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid capability static type borrow type encoding: %w",
			err,
		)
	}

	return NewCapabilityStaticType(
		d.memoryGauge,
		borrowStaticType,
	), nil
}

func (d TypeDecoder) decodeCompositeTypeInfo() (atree.TypeInfo, error) {

	length, err := d.decoder.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	if length != encodedCompositeTypeInfoLength {
		return nil, errors.NewUnexpectedError(
			"invalid composite type info: expected %d elements, got %d",
			encodedCompositeTypeInfoLength, length,
		)
	}

	location, err := d.DecodeLocation()
	if err != nil {
		return nil, err
	}

	qualifiedIdentifier, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		return nil, err
	}

	kind, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		return nil, err
	}

	if kind >= uint64(common.CompositeKindCount()) {
		return nil, errors.NewUnexpectedError(
			"invalid composite ordered map type info: invalid kind %d",
			kind,
		)
	}

	return NewCompositeTypeInfo(
		d.memoryGauge,
		location,
		qualifiedIdentifier,
		common.CompositeKind(kind),
	), nil
}

func (d TypeDecoder) decodeInclusiveRangeStaticType() (StaticType, error) {
	elementType, err := d.DecodeStaticType()
	if err != nil {
		return nil, errors.NewUnexpectedError(
			"invalid inclusive range static type encoding: %w",
			err,
		)
	}
	return NewInclusiveRangeStaticType(d.memoryGauge, elementType), nil
}

func DecodeTypeInfo(decoder *cbor.StreamDecoder, memoryGauge common.MemoryGauge) (atree.TypeInfo, error) {
	d := NewTypeDecoder(decoder, memoryGauge)

	ty, err := d.decoder.NextType()
	if err != nil {
		return nil, err
	}

	switch ty {
	case cbor.TagType:
		tag, err := d.decoder.DecodeTagNumber()
		if err != nil {
			return nil, err
		}

		switch tag {
		case CBORTagConstantSizedStaticType:
			return d.decodeConstantSizedStaticType()
		case CBORTagVariableSizedStaticType:
			return d.decodeVariableSizedStaticType()
		case CBORTagDictionaryStaticType:
			return d.decodeDictionaryStaticType()
		case CBORTagCompositeValue:
			return d.decodeCompositeTypeInfo()
		default:
			return nil, errors.NewUnexpectedError("invalid type info CBOR tag: %d", tag)
		}

	case cbor.NilType:
		err = d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
		return emptyTypeInfo, nil

	default:
		return nil, errors.NewUnexpectedError("invalid type info CBOR type: %d", ty)
	}
}

type LocationDecoder struct {
	decoder     *cbor.StreamDecoder
	memoryGauge common.MemoryGauge
}

func NewLocationDecoder(
	decoder *cbor.StreamDecoder,
	memoryGauge common.MemoryGauge,
) LocationDecoder {
	return LocationDecoder{
		decoder:     decoder,
		memoryGauge: memoryGauge,
	}
}

func (d LocationDecoder) DecodeLocation() (common.Location, error) {
	// Location can be CBOR nil.
	err := d.decoder.DecodeNil()
	if err == nil {
		return nil, nil
	}

	_, ok := err.(*cbor.WrongTypeError)
	if !ok {
		return nil, err
	}

	number, err := d.decoder.DecodeTagNumber()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid location encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	switch number {
	case CBORTagAddressLocation:
		return d.decodeAddressLocation()

	case CBORTagStringLocation:
		return d.decodeStringLocation()

	case CBORTagIdentifierLocation:
		return d.decodeIdentifierLocation()

	case CBORTagTransactionLocation:
		return d.decodeTransactionLocation()

	case CBORTagScriptLocation:
		return d.decodeScriptLocation()

	default:
		return nil, errors.NewUnexpectedError("invalid location encoding tag: %d", number)
	}
}

func (d LocationDecoder) decodeStringLocation() (common.Location, error) {
	s, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid string location encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return common.NewStringLocation(d.memoryGauge, s), nil
}

func (d LocationDecoder) decodeIdentifierLocation() (common.Location, error) {
	s, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid identifier location encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return common.IdentifierLocation(s), nil
}

func (d LocationDecoder) decodeAddressLocation() (common.Location, error) {

	const expectedLength = encodedAddressLocationLength

	size, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid address location encoding: expected [%d]any, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, errors.NewUnexpectedError("invalid address location encoding: expected [%d]any, got [%d]any",
			expectedLength,
			size,
		)
	}

	// Address

	// Decode address at array index encodedAddressLocationAddressFieldKey
	encodedAddress, err := d.decodeAddressBytes()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid address location address encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	// Name

	// Decode name at array index encodedAddressLocationNameFieldKey
	name, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid address location name encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	address, err := common.BytesToAddress(encodedAddress)
	if err != nil {
		return nil, err
	}
	return common.NewAddressLocation(d.memoryGauge, address, name), nil
}

func (d LocationDecoder) decodeTransactionLocation() (common.Location, error) {
	s, err := d.decoder.DecodeBytes()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid transaction location encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return common.NewTransactionLocation(d.memoryGauge, s), nil
}

func (d LocationDecoder) decodeScriptLocation() (common.Location, error) {
	s, err := d.decoder.DecodeBytes()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, errors.NewUnexpectedError(
				"invalid script location encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return common.NewScriptLocation(d.memoryGauge, s), nil
}
