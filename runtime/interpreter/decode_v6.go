/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2021 Dapper Labs, Inc.
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

	"github.com/fxamacker/atree"
	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

func DecodeStorableV6(
	decoder *cbor.StreamDecoder,
	storage atree.SlabStorage,
) (atree.Storable, error) {
	return DecoderV6{
		decoder: decoder,
		storage: storage,
	}.decodeStorable()
}

type DecoderV6 struct {
	decoder *cbor.StreamDecoder
	storage atree.SlabStorage
}

func (d DecoderV6) decodeStorable() (atree.Storable, error) {
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
		storable = BoolValue(v)

	case cbor.TextStringType:
		v, err := d.decoder.DecodeString()
		if err != nil {
			return nil, err
		}
		storable = d.decodeString(v)

	case cbor.NilType:
		err := d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
		storable = NilValue{}

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
			err := d.decoder.Skip()
			if err != nil {
				return nil, err
			}
			storable = VoidValue{}

		case CBORTagDictionaryValue:
			var value *DictionaryValue
			value, err = d.decodeDictionary()
			storable = DictionaryStorable{
				Dictionary: value,
			}

		case CBORTagSomeValue:
			storable, err = d.decodeSome()

		case CBORTagAddressValue:
			storable, err = d.decodeAddress()

		case CBORTagCompositeValue:
			var value *CompositeValue
			value, err = d.decodeComposite()
			storable = CompositeStorable{
				Composite: value,
			}

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

func (d DecoderV6) decodeString(v string) *StringValue {
	return NewStringValue(v)
}

func (d DecoderV6) decodeLocation() (common.Location, error) {
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

	default:
		return nil, fmt.Errorf("invalid location encoding tag: %d", number)
	}
}

func (d DecoderV6) decodeStringLocation() (common.Location, error) {
	s, err := d.decoder.DecodeString()
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

func (d DecoderV6) decodeIdentifierLocation() (common.Location, error) {
	s, err := d.decoder.DecodeString()
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

func (d DecoderV6) decodeAddressLocation() (common.Location, error) {

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

	// Decode address at array index encodedAddressLocationAddressFieldKeyV6
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

	err = d.checkAddressLength(encodedAddress)
	if err != nil {
		return nil, err
	}

	// Name

	// Decode name at array index encodedAddressLocationNameFieldKeyV6
	name, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid address location name encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return common.AddressLocation{
		Address: common.BytesToAddress(encodedAddress),
		Name:    name,
	}, nil
}

func (d DecoderV6) decodeComposite() (*CompositeValue, error) {

	const expectedLength = encodedCompositeValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, fmt.Errorf(
			"invalid composite encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Location

	// Decode location at array index encodedCompositeValueLocationFieldKeyV6
	location, err := d.decodeLocation()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid composite location encoding: %w",
			err,
		)
	}

	// Kind

	// Decode kind at array index encodedCompositeValueKindFieldKeyV6
	encodedKind, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite kind encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if encodedKind >= uint64(common.CompositeKindCount()) {
		return nil, fmt.Errorf(
			"invalid composite kind: %d",
			encodedKind,
		)
	}

	kind := common.CompositeKind(encodedKind)

	// Fields

	// Decode fields at array index encodedCompositeValueFieldsFieldKeyV6
	fieldsSize, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite fields encoding: %s",
				e.ActualType.String(),
			)
		}

		return nil, err
	}

	if fieldsSize%2 == 1 {
		return nil, fmt.Errorf(
			"invalid composite fields encoding: fields should have even number of elements: got %d",
			fieldsSize,
		)
	}

	fields := NewStringValueOrderedMap()

	for i := 0; i < int(fieldsSize); i += 2 {

		// field name
		fieldName, err := d.decoder.DecodeString()
		if err != nil {
			if e, ok := err.(*cbor.WrongTypeError); ok {
				return nil, fmt.Errorf(
					"invalid composite field name encoding (%d): %s",
					i/2,
					e.ActualType.String(),
				)
			}
			return nil, err
		}

		// field value

		decodedStorable, err := d.decodeStorable()
		if err != nil {
			return nil, fmt.Errorf(
				"invalid composite field value encoding (%s): %w",
				fieldName,
				err,
			)
		}

		value, err := StoredValue(decodedStorable, d.storage)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to decode composite field value (%s): %w",
				fieldName,
				err,
			)
		}

		fields.Set(fieldName, value)
	}

	// Qualified identifier

	// Decode qualified identifier at array index encodedCompositeValueQualifiedIdentifierFieldKeyV6
	qualifiedIdentifier, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite qualified identifier encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return &CompositeValue{
		// TODO: StorageID
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Kind:                kind,
		Fields:              fields,
		// TODO: Owner
	}, nil
}

func (d DecoderV6) decodeInt() (IntValue, error) {
	bigInt, err := d.decoder.DecodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return IntValue{}, fmt.Errorf("invalid Int encoding: %s", e.ActualType.String())
		}
		return IntValue{}, err
	}

	return NewIntValueFromBigInt(bigInt), nil
}

func (d DecoderV6) decodeInt8() (Int8Value, error) {
	v, err := d.decoder.DecodeInt64()
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

	return Int8Value(v), nil
}

func (d DecoderV6) decodeInt16() (Int16Value, error) {
	v, err := d.decoder.DecodeInt64()
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
	return Int16Value(v), nil
}

func (d DecoderV6) decodeInt32() (Int32Value, error) {
	v, err := d.decoder.DecodeInt64()
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
	return Int32Value(v), nil
}

func (d DecoderV6) decodeInt64() (Int64Value, error) {
	v, err := d.decoder.DecodeInt64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Int64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	return Int64Value(v), nil
}

func (d DecoderV6) decodeInt128() (Int128Value, error) {
	bigInt, err := d.decoder.DecodeBigInt()
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

	return NewInt128ValueFromBigInt(bigInt), nil
}

func (d DecoderV6) decodeInt256() (Int256Value, error) {
	bigInt, err := d.decoder.DecodeBigInt()
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

	return NewInt256ValueFromBigInt(bigInt), nil
}

func (d DecoderV6) decodeUInt() (UIntValue, error) {
	bigInt, err := d.decoder.DecodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return UIntValue{}, fmt.Errorf("invalid UInt encoding: %s", e.ActualType.String())
		}
		return UIntValue{}, err
	}

	if bigInt.Sign() < 0 {
		return UIntValue{}, fmt.Errorf("invalid UInt: got %s, expected positive", bigInt)
	}

	return NewUIntValueFromBigInt(bigInt), nil
}

func (d DecoderV6) decodeUInt8() (UInt8Value, error) {
	value, err := d.decoder.DecodeUint64()
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
	return UInt8Value(value), nil
}

func (d DecoderV6) decodeUInt16() (UInt16Value, error) {
	value, err := d.decoder.DecodeUint64()
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
	return UInt16Value(value), nil
}

func (d DecoderV6) decodeUInt32() (UInt32Value, error) {
	value, err := d.decoder.DecodeUint64()
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
	return UInt32Value(value), nil
}

func (d DecoderV6) decodeUInt64() (UInt64Value, error) {
	value, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UInt64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return UInt64Value(value), nil
}

func (d DecoderV6) decodeUInt128() (UInt128Value, error) {
	bigInt, err := d.decoder.DecodeBigInt()
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

	return NewUInt128ValueFromBigInt(bigInt), nil
}

func (d DecoderV6) decodeUInt256() (UInt256Value, error) {
	bigInt, err := d.decoder.DecodeBigInt()
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

	return NewUInt256ValueFromBigInt(bigInt), nil
}

func (d DecoderV6) decodeWord8() (Word8Value, error) {
	value, err := d.decoder.DecodeUint64()
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
	return Word8Value(value), nil
}

func (d DecoderV6) decodeWord16() (Word16Value, error) {
	value, err := d.decoder.DecodeUint64()
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
	return Word16Value(value), nil
}

func (d DecoderV6) decodeWord32() (Word32Value, error) {
	value, err := d.decoder.DecodeUint64()
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
	return Word32Value(value), nil
}

func (d DecoderV6) decodeWord64() (Word64Value, error) {
	value, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Word64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return Word64Value(value), nil
}

func (d DecoderV6) decodeFix64() (Fix64Value, error) {
	value, err := d.decoder.DecodeInt64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Fix64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return Fix64Value(value), nil
}

func (d DecoderV6) decodeUFix64() (UFix64Value, error) {
	value, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UFix64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return UFix64Value(value), nil
}

func (d DecoderV6) decodeSome() (*SomeValue, error) {
	// TODO:
	return nil, nil
	//
	//value, err := d.decodeValue()
	//if err != nil {
	//	return nil, fmt.Errorf(
	//		"invalid some value encoding: %w",
	//		err,
	//	)
	//}
	//
	//return &SomeValue{
	//	Value: value,
	//	Owner: d.owner,
	//}, nil
}

func (d DecoderV6) checkAddressLength(addressBytes []byte) error {
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

func (d DecoderV6) decodeAddress() (AddressValue, error) {
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

	err = d.checkAddressLength(addressBytes)
	if err != nil {
		return AddressValue{}, err
	}

	address := NewAddressValueFromBytes(addressBytes)
	return address, nil
}

func (d DecoderV6) decodePath() (PathValue, error) {

	const expectedLength = encodedPathValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PathValue{}, fmt.Errorf(
				"invalid path encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return PathValue{}, err
	}

	if size != expectedLength {
		return PathValue{}, fmt.Errorf(
			"invalid path encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode domain at array index encodedPathValueDomainFieldKeyV6
	domain, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PathValue{}, fmt.Errorf(
				"invalid path domain encoding: %s",
				e.ActualType.String(),
			)
		}
		return PathValue{}, err
	}

	// Decode identifier at array index encodedPathValueIdentifierFieldKeyV6
	identifier, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PathValue{}, fmt.Errorf(
				"invalid path identifier encoding: %s",
				e.ActualType.String(),
			)
		}
		return PathValue{}, err
	}

	return PathValue{
		Domain:     common.PathDomain(domain),
		Identifier: identifier,
	}, nil
}

func (d DecoderV6) decodeCapability() (CapabilityValue, error) {

	const expectedLength = encodedCapabilityValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return CapabilityValue{}, fmt.Errorf(
				"invalid capability encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return CapabilityValue{}, err
	}

	if size != expectedLength {
		return CapabilityValue{}, fmt.Errorf(
			"invalid capability encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// address

	// Decode address at array index encodedCapabilityValueAddressFieldKeyV6
	var num uint64
	num, err = d.decoder.DecodeTagNumber()
	if err != nil {
		return CapabilityValue{}, fmt.Errorf(
			"invalid capability address: %w",
			err,
		)
	}
	if num != CBORTagAddressValue {
		return CapabilityValue{}, fmt.Errorf(
			"invalid capability address: wrong tag %d",
			num,
		)
	}
	address, err := d.decodeAddress()
	if err != nil {
		return CapabilityValue{}, fmt.Errorf(
			"invalid capability address: %w",
			err,
		)
	}

	// path

	// Decode path at array index encodedCapabilityValuePathFieldKeyV6
	num, err = d.decoder.DecodeTagNumber()
	if err != nil {
		return CapabilityValue{}, fmt.Errorf("invalid capability path: %w", err)
	}
	if num != CBORTagPathValue {
		return CapabilityValue{}, fmt.Errorf("invalid capability path: wrong tag %d", num)
	}
	path, err := d.decodePath()
	if err != nil {
		return CapabilityValue{}, fmt.Errorf("invalid capability path: %w", err)
	}

	// Decode borrow type at array index encodedCapabilityValueBorrowTypeFieldKeyV6

	// borrow type (optional, for backwards compatibility)
	// Capabilities used to be untyped, i.e. they didn't have a borrow type.
	// Later an optional type paramater, the borrow type, was added to it,
	// which specifies as what type the capability should be borrowed.
	//
	// The decoding must be backwards-compatible and support both capability values
	// with a borrow type and ones without

	var borrowType StaticType

	// Optional borrow type can be CBOR nil.
	err = d.decoder.DecodeNil()
	if _, ok := err.(*cbor.WrongTypeError); ok {
		borrowType, err = d.decodeStaticType()
	}

	if err != nil {
		return CapabilityValue{}, fmt.Errorf("invalid capability borrow type encoding: %w", err)
	}

	return CapabilityValue{
		Address:    address,
		Path:       path,
		BorrowType: borrowType,
	}, nil
}

func (d DecoderV6) decodeLink() (LinkValue, error) {

	const expectedLength = encodedLinkValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return LinkValue{}, fmt.Errorf(
				"invalid link encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return LinkValue{}, err
	}

	if size != expectedLength {
		return LinkValue{}, fmt.Errorf(
			"invalid link encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode path at array index encodedLinkValueTargetPathFieldKeyV6
	num, err := d.decoder.DecodeTagNumber()
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link target path encoding: %w", err)
	}
	if num != CBORTagPathValue {
		return LinkValue{}, fmt.Errorf("invalid link target path encoding: expected CBOR tag %d, got %d", CBORTagPathValue, num)
	}
	pathValue, err := d.decodePath()
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link target path encoding: %w", err)
	}

	// Decode type at array index encodedLinkValueTypeFieldKeyV6
	staticType, err := d.decodeStaticType()
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link type encoding: %w", err)
	}

	return LinkValue{
		TargetPath: pathValue,
		Type:       staticType,
	}, nil
}

func (d DecoderV6) decodeStaticType() (StaticType, error) {
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

func (d DecoderV6) decodePrimitiveStaticType() (PrimitiveStaticType, error) {
	encoded, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PrimitiveStaticTypeUnknown,
				fmt.Errorf("invalid primitive static type encoding: %s", e.ActualType.String())
		}
		return PrimitiveStaticTypeUnknown, err
	}
	return PrimitiveStaticType(encoded), nil
}

func (d DecoderV6) decodeOptionalStaticType() (StaticType, error) {
	staticType, err := d.decodeStaticType()
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

func (d DecoderV6) decodeCompositeStaticType() (StaticType, error) {
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

	// Decode location at array index encodedCompositeStaticTypeLocationFieldKeyV6
	location, err := d.decodeLocation()
	if err != nil {
		return nil, fmt.Errorf("invalid composite static type location encoding: %w", err)
	}

	// Decode qualified identifier at array index encodedCompositeStaticTypeQualifiedIdentifierFieldKeyV6
	qualifiedIdentifier, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite static type qualified identifier encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return CompositeStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
	}, nil
}

func (d DecoderV6) decodeInterfaceStaticType() (InterfaceStaticType, error) {
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

	// Decode location at array index encodedInterfaceStaticTypeLocationFieldKeyV6
	location, err := d.decodeLocation()
	if err != nil {
		return InterfaceStaticType{}, fmt.Errorf(
			"invalid interface static type location encoding: %w",
			err,
		)
	}

	// Decode qualified identifier at array index encodedInterfaceStaticTypeQualifiedIdentifierFieldKeyV6
	qualifiedIdentifier, err := d.decoder.DecodeString()
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

func (d DecoderV6) decodeVariableSizedStaticType() (StaticType, error) {
	staticType, err := d.decodeStaticType()
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

func (d DecoderV6) decodeConstantSizedStaticType() (StaticType, error) {

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

	// Decode size at array index encodedConstantSizedStaticTypeSizeFieldKeyV6
	size, err := d.decoder.DecodeUint64()
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

	// Decode type at array index encodedConstantSizedStaticTypeTypeFieldKeyV6
	staticType, err := d.decodeStaticType()
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

func (d DecoderV6) decodeReferenceStaticType() (StaticType, error) {
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

	// Decode authorized at array index encodedReferenceStaticTypeAuthorizedFieldKeyV6
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

	// Decode type at array index encodedReferenceStaticTypeTypeFieldKeyV6
	staticType, err := d.decodeStaticType()
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

func (d DecoderV6) decodeDictionaryStaticType() (StaticType, error) {
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

	// Decode key type at array index encodedDictionaryStaticTypeKeyTypeFieldKeyV6
	keyType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid dictionary static type key type encoding: %w",
			err,
		)
	}

	// Decode value type at array index encodedDictionaryStaticTypeValueTypeFieldKeyV6
	valueType, err := d.decodeStaticType()
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

func (d DecoderV6) decodeRestrictedStaticType() (StaticType, error) {
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

	// Decode restricted type at array index encodedRestrictedStaticTypeTypeFieldKeyV6
	restrictedType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid restricted static type key type encoding: %w",
			err,
		)
	}

	// Decode restrictions at array index encodedRestrictedStaticTypeRestrictionsFieldKeyV6
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

func (d DecoderV6) decodeType() (TypeValue, error) {
	const expectedLength = encodedTypeValueTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return TypeValue{}, fmt.Errorf(
				"invalid type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return TypeValue{}, err
	}

	if arraySize != expectedLength {
		return TypeValue{}, fmt.Errorf(
			"invalid type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode type at array index encodedTypeValueTypeFieldKeyV6
	var staticType StaticType

	// Optional type can be CBOR nil.
	err = d.decoder.DecodeNil()
	if _, ok := err.(*cbor.WrongTypeError); ok {
		staticType, err = d.decodeStaticType()
	}

	if err != nil {
		return TypeValue{}, fmt.Errorf("invalid type encoding: %w", err)
	}

	return TypeValue{
		Type: staticType,
	}, nil
}

func (d DecoderV6) decodeCapabilityStaticType() (StaticType, error) {
	var borrowStaticType StaticType

	// Optional borrow type can be CBOR nil.
	err := d.decoder.DecodeNil()
	if _, ok := err.(*cbor.WrongTypeError); ok {
		borrowStaticType, err = d.decodeStaticType()
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

func (d DecoderV6) decodeDictionary() (*DictionaryValue, error) {

	const expectedLength = encodedDictionaryValueLength

	size, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid dictionary encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, fmt.Errorf(
			"invalid dictionary encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode type
	staticType, err := d.decodeStaticType()
	if err != nil {
		return nil, err
	}

	dictionaryStaticType, ok := staticType.(DictionaryStaticType)
	if !ok {
		return nil, fmt.Errorf(
			"invalid dictionary static type encoding: %s",
			staticType.String(),
		)
	}

	// Decode keys at array index encodedDictionaryValueKeysFieldKeyV6
	keysStorable, err := d.decodeStorable()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid dictionary keys encoding: %w",
			err,
		)
	}

	keysStorableIDStorable, ok := keysStorable.(atree.StorageIDStorable)
	if !ok {
		return nil, fmt.Errorf(
			"invalid dictionary keys encoding: %T",
			keysStorableIDStorable,
		)
	}

	keysValue, err := StoredValue(keysStorableIDStorable, d.storage)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid dictionary keys encoding: %w",
			err,
		)
	}

	keys, ok := keysValue.(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf(
			"invalid dictionary keys encoding: %T",
			keysValue,
		)
	}

	// Decode entries at array index encodedDictionaryValueEntriesFieldKeyV6
	entryCount, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid dictionary entries encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	entries := NewStringValueOrderedMap()

	// TODO:
	_ = entryCount
	//for _, keyValue := range keys.Elements() {
	//	keyStringValue, ok := keyValue.(HasKeyString)
	//	if !ok {
	//		return nil, fmt.Errorf(
	//			"invalid dictionary key encoding (%d): %T",
	//			keyIndex,
	//			keyValue,
	//		)
	//	}
	//
	//	keyString := keyStringValue.KeyString()
	//	valuePath[lastValuePathIndex] = keyString
	//
	//	decodedValue, err := d.decodeValue(valuePath)
	//	if err != nil {
	//		return nil, fmt.Errorf(
	//			"invalid dictionary value encoding (%s): %w",
	//			keyString,
	//			err,
	//		)
	//	}
	//
	//	entries.Set(keyString, decodedValue)
	//
	//	keyIndex++
	//}

	return &DictionaryValue{
		Type:    dictionaryStaticType,
		Keys:    keys,
		Entries: entries,
		// TODO: Owner
		// TODO: StorageID
	}, nil
}
