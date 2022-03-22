/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

func (e UnsupportedTagDecodingError) Error() string {
	return fmt.Sprintf(
		"unsupported decoded tag: %d",
		e.Tag,
	)
}

type InvalidStringLengthError struct {
	Length uint64
}

func (e InvalidStringLengthError) Error() string {
	return fmt.Sprintf(
		"invalid string length: got %d, expected max %d",
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
	slabStorageID atree.StorageID,
	memoryGauge common.MemoryGauge,
) (
	atree.Storable,
	error,
) {
	return NewStorableDecoder(decoder, slabStorageID, memoryGauge).decodeStorable()
}

func NewStorableDecoder(
	decoder *cbor.StreamDecoder,
	slabStorageID atree.StorageID,
	memoryGauge common.MemoryGauge,
) StorableDecoder {
	return StorableDecoder{
		decoder:       decoder,
		memoryGauge:   memoryGauge,
		slabStorageID: slabStorageID,
		TypeDecoder: NewTypeDecoder(
			decoder,
			memoryGauge,
		),
	}
}

type StorableDecoder struct {
	memoryGauge   common.MemoryGauge
	decoder       *cbor.StreamDecoder
	slabStorageID atree.StorageID
	TypeDecoder
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
		common.UseConstantMemory(d.memoryGauge, common.MemoryKindBool)
		v, err := d.decoder.DecodeBool()
		if err != nil {
			return nil, err
		}
		storable = NewUnmeteredBoolValue(v)

	case cbor.NilType:
		common.UseConstantMemory(d.memoryGauge, common.MemoryKindNil)
		err := d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
		storable = NewUnmeteredNilValue()

	case cbor.TextStringType:
		str, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
		if err != nil {
			return nil, err
		}
		// already metered by decodeString
		storable = StringAtreeValue(str)

	case cbor.TagType:
		var num uint64
		num, err = d.decoder.DecodeTagNumber()
		if err != nil {
			return nil, err
		}

		switch num {

		case atree.CBORTagStorageID:
			return atree.DecodeStorageIDStorable(d.decoder)

		case CBORTagVoidValue:
			common.UseConstantMemory(d.memoryGauge, common.MemoryKindVoid)
			err := d.decoder.Skip()
			if err != nil {
				return nil, err
			}
			storable = NewVoidValue(d.memoryGauge)

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

		case CBORTagLinkValue:
			storable, err = d.decodeLink()

		case CBORTagTypeValue:
			storable, err = d.decodeType()

		default:
			return nil, UnsupportedTagDecodingError{
				Tag: num,
			}
		}

	default:
		return nil, fmt.Errorf(
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
			return "", fmt.Errorf(
				"invalid Character encoding: %s",
				err.ActualType.String(),
			)
		}
		return "", err
	}
	if !sema.IsValidCharacter(v) {
		return "", fmt.Errorf(
			"invalid character encoding: %s",
			v,
		)
	}

	// NOTE: already metered by decodeCharacter
	return NewUnmeteredCharacterValue(v), nil
}

func (d StorableDecoder) decodeStringValue() (*StringValue, error) {
	str, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindString)
	if err != nil {
		if err, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid String encoding: %s",
				err.ActualType.String(),
			)
		}
		return nil, err
	}

	// NOTE: already metered by decodeString
	return NewUnmeteredStringValue(str), nil
}

func decodeUint64(dec *cbor.StreamDecoder, memoryGauge common.MemoryGauge) (uint64, error) {
	common.UseMemory(memoryGauge, Uint64MemoryUsage)
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
			return IntValue{}, fmt.Errorf(
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
			return 0, fmt.Errorf("unknown Int8 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const min = math.MinInt8
	const max = math.MaxInt8

	if v < min {
		return 0, fmt.Errorf("invalid Int8: got %d, expected min %d", v, min)
	}

	if v > max {
		return 0, fmt.Errorf("invalid Int8: got %d, expected max %d", v, max)
	}

	// Already metered at `decodeInt64` function
	return NewUnmeteredInt8Value(int8(v)), nil
}

func (d StorableDecoder) decodeInt16() (Int16Value, error) {
	v, err := decodeInt64(d)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Int16 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const min = math.MinInt16
	const max = math.MaxInt16

	if v < min {
		return 0, fmt.Errorf("invalid Int16: got %d, expected min %d", v, min)
	}

	if v > max {
		return 0, fmt.Errorf("invalid Int16: got %d, expected max %d", v, max)
	}

	// Already metered at `decodeInt64` function
	return NewUnmeteredInt16Value(int16(v)), nil
}

func (d StorableDecoder) decodeInt32() (Int32Value, error) {
	v, err := decodeInt64(d)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Int32 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const min = math.MinInt32
	const max = math.MaxInt32

	if v < min {
		return 0, fmt.Errorf("invalid Int32: got %d, expected min %d", v, min)
	}
	if v > max {
		return 0, fmt.Errorf("invalid Int32: got %d, expected max %d", v, max)
	}

	// Already metered at `decodeInt64` function
	return NewUnmeteredInt32Value(int32(v)), nil
}

func (d StorableDecoder) decodeInt64() (Int64Value, error) {
	v, err := decodeInt64(d)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Int64 encoding: %s", e.ActualType.String())
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
			return Int128Value{}, fmt.Errorf("invalid Int128 encoding: %s", e.ActualType.String())
		}
		return Int128Value{}, err
	}

	min := sema.Int128TypeMinIntBig
	if bigInt.Cmp(min) < 0 {
		return Int128Value{}, fmt.Errorf("invalid Int128: got %s, expected min %s", bigInt, min)
	}

	max := sema.Int128TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return Int128Value{}, fmt.Errorf("invalid Int128: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredInt128ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeInt256() (Int256Value, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return Int256Value{}, fmt.Errorf("invalid Int256 encoding: %s", e.ActualType.String())
		}
		return Int256Value{}, err
	}

	min := sema.Int256TypeMinIntBig
	if bigInt.Cmp(min) < 0 {
		return Int256Value{}, fmt.Errorf("invalid Int256: got %s, expected min %s", bigInt, min)
	}

	max := sema.Int256TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return Int256Value{}, fmt.Errorf("invalid Int256: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredInt256ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeUInt() (UIntValue, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return UIntValue{}, fmt.Errorf("invalid UInt encoding: %s", e.ActualType.String())
		}
		return UIntValue{}, err
	}

	if bigInt.Sign() < 0 {
		return UIntValue{}, fmt.Errorf("invalid UInt: got %s, expected positive", bigInt)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredUIntValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeUInt8() (UInt8Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UInt8 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	const max = math.MaxUint8
	if value > max {
		return 0, fmt.Errorf("invalid UInt8: got %d, expected max %d", value, max)
	}
	// NOTE: already metered by `decodeUint64`
	return NewUnmeteredUInt8Value(uint8(value)), nil
}

func (d StorableDecoder) decodeUInt16() (UInt16Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UInt16 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const max = math.MaxUint16
	if value > max {
		return 0, fmt.Errorf("invalid UInt16: got %d, expected max %d", value, max)
	}
	// NOTE: already metered by `decodeUint64`
	return NewUnmeteredUInt16Value(uint16(value)), nil
}

func (d StorableDecoder) decodeUInt32() (UInt32Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UInt32 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	const max = math.MaxUint32
	if value > max {
		return 0, fmt.Errorf("invalid UInt32: got %d, expected max %d", value, max)
	}
	// NOTE: already metered by `decodeUint64`
	return NewUnmeteredUInt32Value(uint32(value)), nil
}

func (d StorableDecoder) decodeUInt64() (UInt64Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UInt64 encoding: %s", e.ActualType.String())
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
			return UInt128Value{}, fmt.Errorf("invalid UInt128 encoding: %s", e.ActualType.String())
		}
		return UInt128Value{}, err
	}

	if bigInt.Sign() < 0 {
		return UInt128Value{}, fmt.Errorf("invalid UInt128: got %s, expected positive", bigInt)
	}

	max := sema.UInt128TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return UInt128Value{}, fmt.Errorf("invalid UInt128: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by `decodeBigInt`
	return NewUnmeteredUInt128ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeUInt256() (UInt256Value, error) {
	bigInt, err := d.decodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return UInt256Value{}, fmt.Errorf("invalid UInt256 encoding: %s", e.ActualType.String())
		}
		return UInt256Value{}, err
	}

	if bigInt.Sign() < 0 {
		return UInt256Value{}, fmt.Errorf("invalid UInt256: got %s, expected positive", bigInt)
	}

	max := sema.UInt256TypeMaxIntBig
	if bigInt.Cmp(max) > 0 {
		return UInt256Value{}, fmt.Errorf("invalid UInt256: got %s, expected max %s", bigInt, max)
	}

	// NOTE: already metered by decodeBigInta
	return NewUnmeteredUInt256ValueFromBigInt(bigInt), nil
}

func (d StorableDecoder) decodeWord8() (Word8Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Word8 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	const max = math.MaxUint8
	if value > max {
		return 0, fmt.Errorf("invalid Word8: got %d, expected max %d", value, max)
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredWord8Value(uint8(value)), nil
}

func (d StorableDecoder) decodeWord16() (Word16Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Word16 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	const max = math.MaxUint16
	if value > max {
		return 0, fmt.Errorf("invalid Word16: got %d, expected max %d", value, max)
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredWord16Value(uint16(value)), nil
}

func (d StorableDecoder) decodeWord32() (Word32Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Word32 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	const max = math.MaxUint32
	if value > max {
		return 0, fmt.Errorf("invalid Word32: got %d, expected max %d", value, max)
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredWord32Value(uint32(value)), nil
}

func (d StorableDecoder) decodeWord64() (Word64Value, error) {
	value, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Word64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredWord64Value(value), nil
}

func (d StorableDecoder) decodeFix64() (Fix64Value, error) {
	value, err := decodeInt64(d)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Fix64 encoding: %s", e.ActualType.String())
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
			return 0, fmt.Errorf("unknown UFix64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	// Already metered at `decodeUint64`
	return NewUnmeteredUFix64Value(value), nil
}

func (d StorableDecoder) decodeSome() (SomeStorable, error) {
	storable, err := d.decodeStorable()
	if err != nil {
		return SomeStorable{}, fmt.Errorf(
			"invalid some value encoding: %w",
			err,
		)
	}

	return SomeStorable{
		gauge:    d.memoryGauge,
		Storable: storable,
	}, nil
}

func checkEncodedAddressLength(addressBytes []byte) error {
	actualLength := len(addressBytes)
	const expectedLength = common.AddressLength
	if actualLength > expectedLength {
		return fmt.Errorf(
			"invalid address length: got %d, expected max %d",
			actualLength,
			expectedLength,
		)
	}
	return nil
}

func (d StorableDecoder) decodeAddress() (AddressValue, error) {
	common.UseConstantMemory(d.memoryGauge, common.MemoryKindAddress)

	addressBytes, err := d.decoder.DecodeBytes()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return AddressValue{}, fmt.Errorf(
				"invalid address encoding: %s",
				e.ActualType.String(),
			)
		}
		return AddressValue{}, err
	}

	err = checkEncodedAddressLength(addressBytes)
	if err != nil {
		return AddressValue{}, err
	}

	// Already metered at the start of this method
	return NewUnmeteredAddressValueFromBytes(addressBytes), nil
}

func (d StorableDecoder) decodePath() (PathValue, error) {

	const expectedLength = encodedPathValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			// No need to meter EmptyPathValue here or below because it's ignored for the error
			return EmptyPathValue, fmt.Errorf(
				"invalid path encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return EmptyPathValue, err
	}

	if size != expectedLength {
		return EmptyPathValue, fmt.Errorf(
			"invalid path encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode domain at array index encodedPathValueDomainFieldKey
	domain, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return EmptyPathValue, fmt.Errorf(
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
			return EmptyPathValue, fmt.Errorf(
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

func (d StorableDecoder) decodeCapability() (*CapabilityValue, error) {

	const expectedLength = encodedCapabilityValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid capability encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, fmt.Errorf(
			"invalid capability encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// address

	// Decode address at array index encodedCapabilityValueAddressFieldKey
	var num uint64
	num, err = d.decoder.DecodeTagNumber()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid capability address: %w",
			err,
		)
	}
	if num != CBORTagAddressValue {
		return nil, fmt.Errorf(
			"invalid capability address: wrong tag %d",
			num,
		)
	}
	address, err := d.decodeAddress()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid capability address: %w",
			err,
		)
	}

	// path

	// Decode path at array index encodedCapabilityValuePathFieldKey
	pathStorable, err := d.decodeStorable()
	if err != nil {
		return nil, fmt.Errorf("invalid capability path: %w", err)
	}
	pathValue, ok := pathStorable.(PathValue)
	if !ok {
		return nil, fmt.Errorf("invalid capability path: invalid type %T", pathValue)
	}

	// Decode borrow type at array index encodedCapabilityValueBorrowTypeFieldKey

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
		return nil, fmt.Errorf("invalid capability borrow type encoding: %w", err)
	}

	return NewCapabilityValue(d.memoryGauge, address, pathValue, borrowType), nil
}

func (d StorableDecoder) decodeLink() (LinkValue, error) {

	const expectedLength = encodedLinkValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return EmptyLinkValue, fmt.Errorf(
				"invalid link encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return EmptyLinkValue, err
	}

	if size != expectedLength {
		return EmptyLinkValue, fmt.Errorf(
			"invalid link encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode path at array index encodedLinkValueTargetPathFieldKey
	num, err := d.decoder.DecodeTagNumber()
	if err != nil {
		return EmptyLinkValue, fmt.Errorf("invalid link target path encoding: %w", err)
	}
	if num != CBORTagPathValue {
		return EmptyLinkValue, fmt.Errorf("invalid link target path encoding: expected CBOR tag %d, got %d", CBORTagPathValue, num)
	}
	pathValue, err := d.decodePath()
	if err != nil {
		return EmptyLinkValue, fmt.Errorf("invalid link target path encoding: %w", err)
	}

	// Decode type at array index encodedLinkValueTypeFieldKey
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return EmptyLinkValue, fmt.Errorf("invalid link type encoding: %w", err)
	}

	return NewLinkValue(d.memoryGauge, pathValue, staticType), nil
}

func (d StorableDecoder) decodeType() (TypeValue, error) {
	const expectedLength = encodedTypeValueTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return EmptyTypeValue, fmt.Errorf(
				"invalid type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		// unmetered here and below because value is tossed in decodeStorable when err != nil
		return EmptyTypeValue, err
	}

	if arraySize != expectedLength {
		return EmptyTypeValue, fmt.Errorf(
			"invalid type encoding: expected [%d]interface{}, got [%d]interface{}",
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
		return EmptyTypeValue, fmt.Errorf("invalid type encoding: %w", err)
	}

	return NewUnmeteredTypeValue(staticType), nil
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
			return nil, fmt.Errorf(
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

	case CBORTagRestrictedStaticType:
		return d.decodeRestrictedStaticType()

	case CBORTagCapabilityStaticType:
		return d.decodeCapabilityStaticType()

	default:
		return nil, fmt.Errorf("invalid static type encoding tag: %d", number)
	}
}

func (d TypeDecoder) decodePrimitiveStaticType() (PrimitiveStaticType, error) {
	encoded, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PrimitiveStaticTypeUnknown,
				fmt.Errorf("invalid primitive static type encoding: %s", e.ActualType.String())
		}
		return PrimitiveStaticTypeUnknown, err
	}
	return PrimitiveStaticType(encoded), nil
}

func (d TypeDecoder) decodeOptionalStaticType() (StaticType, error) {
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid optional static type inner type encoding: %w",
			err,
		)
	}
	return OptionalStaticType{
		Type: staticType,
	}, nil
}

func (d TypeDecoder) decodeCompositeStaticType() (StaticType, error) {
	const expectedLength = encodedCompositeStaticTypeLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, fmt.Errorf(
			"invalid composite static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode location at array index encodedCompositeStaticTypeLocationFieldKey
	location, err := d.DecodeLocation()
	if err != nil {
		return nil, fmt.Errorf("invalid composite static type location encoding: %w", err)
	}

	// Decode qualified identifier at array index encodedCompositeStaticTypeQualifiedIdentifierFieldKey
	qualifiedIdentifier, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite static type qualified identifier encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return NewCompositeStaticType(location, qualifiedIdentifier), nil
}

func (d TypeDecoder) decodeInterfaceStaticType() (InterfaceStaticType, error) {
	const expectedLength = encodedInterfaceStaticTypeLength

	size, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return InterfaceStaticType{},
				fmt.Errorf(
					"invalid interface static type encoding: expected [%d]interface{}, got %s",
					expectedLength,
					e.ActualType.String(),
				)
		}
		return InterfaceStaticType{}, err
	}

	if size != expectedLength {
		return InterfaceStaticType{},
			fmt.Errorf(
				"invalid interface static type encoding: expected [%d]interface{}, got [%d]interface{}",
				expectedLength,
				size,
			)
	}

	// Decode location at array index encodedInterfaceStaticTypeLocationFieldKey
	location, err := d.DecodeLocation()
	if err != nil {
		return InterfaceStaticType{}, fmt.Errorf(
			"invalid interface static type location encoding: %w",
			err,
		)
	}

	// Decode qualified identifier at array index encodedInterfaceStaticTypeQualifiedIdentifierFieldKey
	qualifiedIdentifier, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return InterfaceStaticType{},
				fmt.Errorf(
					"invalid interface static type qualified identifier encoding: %s",
					e.ActualType.String(),
				)
		}
		return InterfaceStaticType{}, err
	}

	return InterfaceStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
	}, nil
}

func (d TypeDecoder) decodeVariableSizedStaticType() (StaticType, error) {
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid variable-sized static type encoding: %w",
			err,
		)
	}
	return VariableSizedStaticType{
		Type: staticType,
	}, nil
}

func (d TypeDecoder) decodeConstantSizedStaticType() (StaticType, error) {

	const expectedLength = encodedConstantSizedStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid constant-sized static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, fmt.Errorf(
			"invalid constant-sized static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode size at array index encodedConstantSizedStaticTypeSizeFieldKey
	size, err := decodeUint64(d.decoder, d.memoryGauge)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid constant-sized static type size encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	const max = math.MaxInt64
	if size > max {
		return nil, fmt.Errorf(
			"invalid constant-sized static type size: got %d, expected max %d",
			size,
			max,
		)
	}

	// Decode type at array index encodedConstantSizedStaticTypeTypeFieldKey
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid constant-sized static type inner type encoding: %w",
			err,
		)
	}

	return ConstantSizedStaticType{
		Type: staticType,
		Size: int64(size),
	}, nil
}

func (d TypeDecoder) decodeReferenceStaticType() (StaticType, error) {
	const expectedLength = encodedReferenceStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid reference static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, fmt.Errorf(
			"invalid reference static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode authorized at array index encodedReferenceStaticTypeAuthorizedFieldKey
	authorized, err := d.decoder.DecodeBool()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid reference static type authorized encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	// Decode type at array index encodedReferenceStaticTypeTypeFieldKey
	staticType, err := d.DecodeStaticType()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid reference static type inner type encoding: %w",
			err,
		)
	}

	return ReferenceStaticType{
		Authorized: authorized,
		Type:       staticType,
	}, nil
}

func (d TypeDecoder) decodeDictionaryStaticType() (StaticType, error) {
	const expectedLength = encodedDictionaryStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid dictionary static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, fmt.Errorf(
			"invalid dictionary static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode key type at array index encodedDictionaryStaticTypeKeyTypeFieldKey
	keyType, err := d.DecodeStaticType()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid dictionary static type key type encoding: %w",
			err,
		)
	}

	// Decode value type at array index encodedDictionaryStaticTypeValueTypeFieldKey
	valueType, err := d.DecodeStaticType()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid dictionary static type value type encoding: %w",
			err,
		)
	}

	return DictionaryStaticType{
		KeyType:   keyType,
		ValueType: valueType,
	}, nil
}

func (d TypeDecoder) decodeRestrictedStaticType() (StaticType, error) {
	const expectedLength = encodedRestrictedStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid restricted static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, fmt.Errorf(
			"invalid restricted static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode restricted type at array index encodedRestrictedStaticTypeTypeFieldKey
	restrictedType, err := d.DecodeStaticType()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid restricted static type key type encoding: %w",
			err,
		)
	}

	// Decode restrictions at array index encodedRestrictedStaticTypeRestrictionsFieldKey
	restrictionSize, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid restricted static type restrictions encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	restrictions := make([]InterfaceStaticType, restrictionSize)
	for i := 0; i < int(restrictionSize); i++ {

		number, err := d.decoder.DecodeTagNumber()
		if err != nil {
			if e, ok := err.(*cbor.WrongTypeError); ok {
				return nil, fmt.Errorf(
					"invalid restricted static type restriction encoding: expected CBOR tag, got %s",
					e.ActualType.String(),
				)
			}
			return nil, fmt.Errorf(
				"invalid restricted static type restriction encoding: %w",
				err,
			)
		}

		if number != CBORTagInterfaceStaticType {
			return nil, fmt.Errorf(
				"invalid restricted static type restriction encoding: expected CBOR tag %d, got %d",
				CBORTagInterfaceStaticType,
				number,
			)
		}

		restriction, err := d.decodeInterfaceStaticType()
		if err != nil {
			return nil, fmt.Errorf(
				"invalid restricted static type restriction encoding: %w",
				err,
			)
		}

		restrictions[i] = restriction
	}

	return &RestrictedStaticType{
		Type:         restrictedType,
		Restrictions: restrictions,
	}, nil
}

func (d TypeDecoder) decodeCapabilityStaticType() (StaticType, error) {
	var borrowStaticType StaticType

	// Optional borrow type can be CBOR nil.
	err := d.decoder.DecodeNil()
	if _, ok := err.(*cbor.WrongTypeError); ok {
		borrowStaticType, err = d.DecodeStaticType()
	}

	if err != nil {
		return nil, fmt.Errorf(
			"invalid capability static type borrow type encoding: %w",
			err,
		)
	}

	return CapabilityStaticType{
		BorrowType: borrowStaticType,
	}, nil
}

func (d TypeDecoder) decodeCompositeTypeInfo() (atree.TypeInfo, error) {

	length, err := d.decoder.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	if length != encodedCompositeTypeInfoLength {
		return nil, fmt.Errorf(
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
		return nil, fmt.Errorf(
			"invalid composite ordered map type info: invalid kind %d",
			kind,
		)
	}

	return compositeTypeInfo{
		location:            location,
		qualifiedIdentifier: qualifiedIdentifier,
		kind:                common.CompositeKind(kind),
	}, nil
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
			return nil, fmt.Errorf("invalid type info CBOR tag: %d", tag)
		}

	case cbor.NilType:
		err = d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
		return emptyTypeInfo, nil

	default:
		return nil, fmt.Errorf("invalid type info CBOR type: %d", ty)
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
			return nil, fmt.Errorf(
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
		return nil, fmt.Errorf("invalid location encoding tag: %d", number)
	}
}

func (d LocationDecoder) decodeStringLocation() (common.Location, error) {
	s, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid string location encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return common.StringLocation(s), nil
}

func (d LocationDecoder) decodeIdentifierLocation() (common.Location, error) {
	s, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
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
			return nil, fmt.Errorf(
				"invalid address location encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, fmt.Errorf("invalid address location encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Address

	// Decode address at array index encodedAddressLocationAddressFieldKey
	encodedAddress, err := d.decoder.DecodeBytes()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid address location address encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	err = checkEncodedAddressLength(encodedAddress)
	if err != nil {
		return nil, err
	}

	// Name

	// Decode name at array index encodedAddressLocationNameFieldKey
	name, err := decodeString(d.decoder, d.memoryGauge, common.MemoryKindRawString)
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
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

	return common.AddressLocation{
		Address: address,
		Name:    name,
	}, nil
}

func (d LocationDecoder) decodeTransactionLocation() (common.Location, error) {
	s, err := d.decoder.DecodeBytes()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid transaction location encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return common.TransactionLocation(s), nil
}

func (d LocationDecoder) decodeScriptLocation() (common.Location, error) {
	s, err := d.decoder.DecodeBytes()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid script location encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return common.ScriptLocation(s), nil
}
