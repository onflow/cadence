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

package interpreter

import (
	"fmt"
	"math"

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

func DecodeStorable(
	decoder *cbor.StreamDecoder,
	slabStorageID atree.StorageID,
) (atree.Storable, error) {
	return Decoder{
		decoder:       decoder,
		slabStorageID: slabStorageID,
	}.decodeStorable()
}

type Decoder struct {
	decoder       *cbor.StreamDecoder
	slabStorageID atree.StorageID
}

func (d Decoder) decodeStorable() (atree.Storable, error) {
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

	case cbor.NilType:
		err := d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
		storable = NilValue{}

	case cbor.TextStringType:
		v, err := d.decoder.DecodeString()
		if err != nil {
			return nil, err
		}
		storable = StringAtreeValue(v)

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

		case CBORTagStringValue:
			v, err := d.decoder.DecodeString()
			if err != nil {
				return nil, err
			}
			storable = d.decodeString(v)

		case CBORTagCharacterValue:
			v, err := d.decoder.DecodeString()
			if err != nil {
				return nil, err
			}
			storable, err = d.decodeCharacter(v)
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

func (d Decoder) decodeString(v string) *StringValue {
	return NewStringValue(v)
}

func (d Decoder) decodeCharacter(v string) (CharacterValue, error) {
	if !sema.IsValidCharacter(v) {
		return "", fmt.Errorf(
			"invalid character encoding: %s",
			v,
		)
	}
	return NewCharacterValue(v), nil
}

func decodeLocation(dec *cbor.StreamDecoder) (common.Location, error) {
	// Location can be CBOR nil.
	err := dec.DecodeNil()
	if err == nil {
		return nil, nil
	}

	_, ok := err.(*cbor.WrongTypeError)
	if !ok {
		return nil, err
	}

	number, err := dec.DecodeTagNumber()
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
		return decodeAddressLocation(dec)

	case CBORTagStringLocation:
		return decodeStringLocation(dec)

	case CBORTagIdentifierLocation:
		return decodeIdentifierLocation(dec)

	case CBORTagTransactionLocation:
		return decodeTransactionLocation(dec)

	case CBORTagScriptLocation:
		return decodeScriptLocation(dec)

	default:
		return nil, fmt.Errorf("invalid location encoding tag: %d", number)
	}
}

func decodeStringLocation(dec *cbor.StreamDecoder) (common.Location, error) {
	s, err := dec.DecodeString()
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

func decodeIdentifierLocation(dec *cbor.StreamDecoder) (common.Location, error) {
	s, err := dec.DecodeString()
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

func decodeAddressLocation(dec *cbor.StreamDecoder) (common.Location, error) {

	const expectedLength = encodedAddressLocationLength

	size, err := dec.DecodeArrayHead()

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
	encodedAddress, err := dec.DecodeBytes()
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
	name, err := dec.DecodeString()
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

func decodeTransactionLocation(dec *cbor.StreamDecoder) (common.Location, error) {
	s, err := dec.DecodeBytes()
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

func decodeScriptLocation(dec *cbor.StreamDecoder) (common.Location, error) {
	s, err := dec.DecodeBytes()
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

func (d Decoder) decodeInt() (IntValue, error) {
	bigInt, err := d.decoder.DecodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return IntValue{}, fmt.Errorf("invalid Int encoding: %s", e.ActualType.String())
		}
		return IntValue{}, err
	}

	return NewIntValueFromBigInt(bigInt), nil
}

func (d Decoder) decodeInt8() (Int8Value, error) {
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

func (d Decoder) decodeInt16() (Int16Value, error) {
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

func (d Decoder) decodeInt32() (Int32Value, error) {
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

func (d Decoder) decodeInt64() (Int64Value, error) {
	v, err := d.decoder.DecodeInt64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Int64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	return Int64Value(v), nil
}

func (d Decoder) decodeInt128() (Int128Value, error) {
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

func (d Decoder) decodeInt256() (Int256Value, error) {
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

func (d Decoder) decodeUInt() (UIntValue, error) {
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

func (d Decoder) decodeUInt8() (UInt8Value, error) {
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

func (d Decoder) decodeUInt16() (UInt16Value, error) {
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

func (d Decoder) decodeUInt32() (UInt32Value, error) {
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

func (d Decoder) decodeUInt64() (UInt64Value, error) {
	value, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UInt64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return UInt64Value(value), nil
}

func (d Decoder) decodeUInt128() (UInt128Value, error) {
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

func (d Decoder) decodeUInt256() (UInt256Value, error) {
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

func (d Decoder) decodeWord8() (Word8Value, error) {
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

func (d Decoder) decodeWord16() (Word16Value, error) {
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

func (d Decoder) decodeWord32() (Word32Value, error) {
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

func (d Decoder) decodeWord64() (Word64Value, error) {
	value, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Word64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return Word64Value(value), nil
}

func (d Decoder) decodeFix64() (Fix64Value, error) {
	value, err := d.decoder.DecodeInt64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Fix64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return Fix64Value(value), nil
}

func (d Decoder) decodeUFix64() (UFix64Value, error) {
	value, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UFix64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return UFix64Value(value), nil
}

func (d Decoder) decodeSome() (SomeStorable, error) {
	storable, err := d.decodeStorable()
	if err != nil {
		return SomeStorable{}, fmt.Errorf(
			"invalid some value encoding: %w",
			err,
		)
	}

	return SomeStorable{
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

func (d Decoder) decodeAddress() (AddressValue, error) {
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

	address := NewAddressValueFromBytes(addressBytes)
	return address, nil
}

func (d Decoder) decodePath() (PathValue, error) {

	const expectedLength = encodedPathValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
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
	domain, err := d.decoder.DecodeUint64()
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
	identifier, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return EmptyPathValue, fmt.Errorf(
				"invalid path identifier encoding: %s",
				e.ActualType.String(),
			)
		}
		return EmptyPathValue, err
	}

	return PathValue{
		Domain:     common.PathDomain(domain),
		Identifier: identifier,
	}, nil
}

func (d Decoder) decodeCapability() (*CapabilityValue, error) {

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
		borrowType, err = decodeStaticType(d.decoder)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid capability borrow type encoding: %w", err)
	}

	return &CapabilityValue{
		Address:    address,
		Path:       pathValue,
		BorrowType: borrowType,
	}, nil
}

func (d Decoder) decodeLink() (LinkValue, error) {

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

	// Decode path at array index encodedLinkValueTargetPathFieldKey
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

	// Decode type at array index encodedLinkValueTypeFieldKey
	staticType, err := decodeStaticType(d.decoder)
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link type encoding: %w", err)
	}

	return LinkValue{
		TargetPath: pathValue,
		Type:       staticType,
	}, nil
}

func (d Decoder) decodeType() (TypeValue, error) {
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

	// Decode type at array index encodedTypeValueTypeFieldKey
	var staticType StaticType

	// Optional type can be CBOR nil.
	err = d.decoder.DecodeNil()
	if _, ok := err.(*cbor.WrongTypeError); ok {
		staticType, err = decodeStaticType(d.decoder)
	}

	if err != nil {
		return TypeValue{}, fmt.Errorf("invalid type encoding: %w", err)
	}

	return TypeValue{
		Type: staticType,
	}, nil
}

func decodeStaticType(dec *cbor.StreamDecoder) (StaticType, error) {
	number, err := dec.DecodeTagNumber()
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
		return decodePrimitiveStaticType(dec)

	case CBORTagOptionalStaticType:
		return decodeOptionalStaticType(dec)

	case CBORTagCompositeStaticType:
		return decodeCompositeStaticType(dec)

	case CBORTagInterfaceStaticType:
		return decodeInterfaceStaticType(dec)

	case CBORTagVariableSizedStaticType:
		return decodeVariableSizedStaticType(dec)

	case CBORTagConstantSizedStaticType:
		return decodeConstantSizedStaticType(dec)

	case CBORTagReferenceStaticType:
		return decodeReferenceStaticType(dec)

	case CBORTagDictionaryStaticType:
		return decodeDictionaryStaticType(dec)

	case CBORTagRestrictedStaticType:
		return decodeRestrictedStaticType(dec)

	case CBORTagCapabilityStaticType:
		return decodeCapabilityStaticType(dec)

	default:
		return nil, fmt.Errorf("invalid static type encoding tag: %d", number)
	}
}

func decodePrimitiveStaticType(dec *cbor.StreamDecoder) (PrimitiveStaticType, error) {
	encoded, err := dec.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PrimitiveStaticTypeUnknown,
				fmt.Errorf("invalid primitive static type encoding: %s", e.ActualType.String())
		}
		return PrimitiveStaticTypeUnknown, err
	}
	return PrimitiveStaticType(encoded), nil
}

func decodeOptionalStaticType(dec *cbor.StreamDecoder) (StaticType, error) {
	staticType, err := decodeStaticType(dec)
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

func decodeCompositeStaticType(dec *cbor.StreamDecoder) (StaticType, error) {
	const expectedLength = encodedCompositeStaticTypeLength

	size, err := dec.DecodeArrayHead()

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
	location, err := decodeLocation(dec)
	if err != nil {
		return nil, fmt.Errorf("invalid composite static type location encoding: %w", err)
	}

	// Decode qualified identifier at array index encodedCompositeStaticTypeQualifiedIdentifierFieldKey
	qualifiedIdentifier, err := dec.DecodeString()
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

func decodeInterfaceStaticType(dec *cbor.StreamDecoder) (InterfaceStaticType, error) {
	const expectedLength = encodedInterfaceStaticTypeLength

	size, err := dec.DecodeArrayHead()

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
	location, err := decodeLocation(dec)
	if err != nil {
		return InterfaceStaticType{}, fmt.Errorf(
			"invalid interface static type location encoding: %w",
			err,
		)
	}

	// Decode qualified identifier at array index encodedInterfaceStaticTypeQualifiedIdentifierFieldKey
	qualifiedIdentifier, err := dec.DecodeString()
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

func decodeVariableSizedStaticType(dec *cbor.StreamDecoder) (StaticType, error) {
	staticType, err := decodeStaticType(dec)
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

func decodeConstantSizedStaticType(dec *cbor.StreamDecoder) (StaticType, error) {

	const expectedLength = encodedConstantSizedStaticTypeLength

	arraySize, err := dec.DecodeArrayHead()

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
	size, err := dec.DecodeUint64()
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
	staticType, err := decodeStaticType(dec)
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

func decodeReferenceStaticType(dec *cbor.StreamDecoder) (StaticType, error) {
	const expectedLength = encodedReferenceStaticTypeLength

	arraySize, err := dec.DecodeArrayHead()

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
	authorized, err := dec.DecodeBool()
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
	staticType, err := decodeStaticType(dec)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid reference static type inner type encoding: %w",
			err,
		)
	}

	return ReferenceStaticType{
		Authorized:   authorized,
		BorrowedType: staticType,
	}, nil
}

func decodeDictionaryStaticType(dec *cbor.StreamDecoder) (StaticType, error) {
	const expectedLength = encodedDictionaryStaticTypeLength

	arraySize, err := dec.DecodeArrayHead()

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
	keyType, err := decodeStaticType(dec)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid dictionary static type key type encoding: %w",
			err,
		)
	}

	// Decode value type at array index encodedDictionaryStaticTypeValueTypeFieldKey
	valueType, err := decodeStaticType(dec)
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

func decodeRestrictedStaticType(dec *cbor.StreamDecoder) (StaticType, error) {
	const expectedLength = encodedRestrictedStaticTypeLength

	arraySize, err := dec.DecodeArrayHead()

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
	restrictedType, err := decodeStaticType(dec)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid restricted static type key type encoding: %w",
			err,
		)
	}

	// Decode restrictions at array index encodedRestrictedStaticTypeRestrictionsFieldKey
	restrictionSize, err := dec.DecodeArrayHead()
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

		number, err := dec.DecodeTagNumber()
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

		restriction, err := decodeInterfaceStaticType(dec)
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

func decodeCapabilityStaticType(dec *cbor.StreamDecoder) (StaticType, error) {
	var borrowStaticType StaticType

	// Optional borrow type can be CBOR nil.
	err := dec.DecodeNil()
	if _, ok := err.(*cbor.WrongTypeError); ok {
		borrowStaticType, err = decodeStaticType(dec)
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

func decodeCompositeTypeInfo(dec *cbor.StreamDecoder) (atree.TypeInfo, error) {

	length, err := dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	if length != encodedCompositeTypeInfoLength {
		return nil, fmt.Errorf(
			"invalid composite type info: expected %d elements, got %d",
			encodedCompositeTypeInfoLength, length,
		)
	}

	location, err := decodeLocation(dec)
	if err != nil {
		return nil, err
	}

	qualifiedIdentifier, err := dec.DecodeString()
	if err != nil {
		return nil, err
	}

	kind, err := dec.DecodeUint64()
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
