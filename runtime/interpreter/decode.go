/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"io"
	"math"
	"math/big"
	"math/bits"
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/sema"
)

type DecodingCallback func(value interface{}, path []string)

// A DecoderV4 decodes CBOR-encoded representations of values.
// It can decode storage format version 4 and later.
//
type DecoderV4 struct {
	decoder        *cbor.StreamDecoder
	owner          *common.Address
	version        uint16
	decodeCallback DecodingCallback
}

// maxInt is math.MaxInt32 or math.MaxInt64 depending on arch.
const maxInt = 1<<(bits.UintSize-1) - 1

// DecodeValue returns a value decoded from its CBOR-encoded representation,
// for the given owner (can be `nil`).  It can decode storage format
// version 4 and later.
//
// The given path is used to identify values in the object graph.
// For example, path elements are appended for array elements (the index),
// dictionary values (the key), and composites (the field name).
//
func DecodeValue(
	data []byte,
	owner *common.Address,
	path []string,
	version uint16,
	decodeCallback DecodingCallback,
) (
	Value,
	error,
) {
	decoder, err := NewByteDecoder(data, owner, version, decodeCallback)
	if err != nil {
		return nil, err
	}

	v, err := decoder.Decode(path)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// NewDecoder initializes a DecoderV4 that will decode CBOR-encoded bytes from the
// given io.Reader.
//
// It sets the given address as the owner (can be `nil`).
//
func NewDecoder(
	reader io.Reader,
	owner *common.Address,
	version uint16,
	decodeCallback DecodingCallback,
) (
	*DecoderV4,
	error,
) {
	return &DecoderV4{
		decoder:        decMode.NewStreamDecoder(reader),
		owner:          owner,
		version:        version,
		decodeCallback: decodeCallback,
	}, nil
}

func NewByteDecoder(
	data []byte,
	owner *common.Address,
	version uint16,
	decodeCallback DecodingCallback,
) (
	*DecoderV4,
	error,
) {
	return &DecoderV4{
		decoder:        decMode.NewByteStreamDecoder(data),
		owner:          owner,
		version:        version,
		decodeCallback: decodeCallback,
	}, nil
}

var decMode = func() cbor.DecMode {
	decMode, err := cbor.DecOptions{
		IntDec:           cbor.IntDecConvertNone,
		MaxArrayElements: maxInt,
		MaxMapPairs:      maxInt,
		MaxNestedLevels:  math.MaxInt16,
	}.DecMode()
	if err != nil {
		panic(err)
	}
	return decMode
}()

// Decode reads CBOR-encoded bytes and decodes them to a value.
//
func (d *DecoderV4) Decode(path []string) (Value, error) {
	return d.decodeValue(path)
}

func (d *DecoderV4) decodeValue(path []string) (Value, error) {
	var value Value
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
		value = BoolValue(v)

	case cbor.TextStringType:
		v, err := d.decoder.DecodeString()
		if err != nil {
			return nil, err
		}
		value = d.decodeString(v)

	case cbor.NilType:
		err := d.decoder.DecodeNil()
		if err != nil {
			return nil, err
		}
		value = NilValue{}

	case cbor.ArrayType:
		value, err = d.decodeArray(path)

	case cbor.TagType:
		var num uint64
		num, err = d.decoder.DecodeTagNumber()
		if err != nil {
			return nil, err
		}

		switch num {

		case cborTagVoidValue:
			err := d.decoder.Skip()
			if err != nil {
				return nil, err
			}
			value = VoidValue{}

		case cborTagDictionaryValue:
			value, err = d.decodeDictionary(path)

		case cborTagSomeValue:
			value, err = d.decodeSome(path)

		case cborTagAddressValue:
			value, err = d.decodeAddress()

		case cborTagCompositeValue:
			value, err = d.decodeComposite(path)

		// Int*

		case cborTagIntValue:
			value, err = d.decodeInt()

		case cborTagInt8Value:
			value, err = d.decodeInt8()

		case cborTagInt16Value:
			value, err = d.decodeInt16()

		case cborTagInt32Value:
			value, err = d.decodeInt32()

		case cborTagInt64Value:
			value, err = d.decodeInt64()

		case cborTagInt128Value:
			value, err = d.decodeInt128()

		case cborTagInt256Value:
			value, err = d.decodeInt256()

		// UInt*

		case cborTagUIntValue:
			value, err = d.decodeUInt()

		case cborTagUInt8Value:
			value, err = d.decodeUInt8()

		case cborTagUInt16Value:
			value, err = d.decodeUInt16()

		case cborTagUInt32Value:
			value, err = d.decodeUInt32()

		case cborTagUInt64Value:
			value, err = d.decodeUInt64()

		case cborTagUInt128Value:
			value, err = d.decodeUInt128()

		case cborTagUInt256Value:
			value, err = d.decodeUInt256()

		// Word*

		case cborTagWord8Value:
			value, err = d.decodeWord8()

		case cborTagWord16Value:
			value, err = d.decodeWord16()

		case cborTagWord32Value:
			value, err = d.decodeWord32()

		case cborTagWord64Value:
			value, err = d.decodeWord64()

		// Fix*

		case cborTagFix64Value:
			value, err = d.decodeFix64()

		// UFix*

		case cborTagUFix64Value:
			value, err = d.decodeUFix64()

		// Storage

		case cborTagPathValue:
			value, err = d.decodePath()

		case cborTagCapabilityValue:
			value, err = d.decodeCapability()

		case cborTagLinkValue:
			value, err = d.decodeLink()

		case cborTagTypeValue:
			value, err = d.decodeType()

		default:
			return nil, fmt.Errorf(
				"unsupported decoded tag (@ %s): %d",
				strings.Join(path, "."),
				num,
			)
		}

	default:
		return nil, fmt.Errorf(
			"unsupported decoded type (@ %s): %s",
			strings.Join(path, "."),
			t.String(),
		)
	}

	if d.decodeCallback != nil {
		d.decodeCallback(value, path)
	}

	return value, err
}

func (d *DecoderV4) decodeString(v string) Value {
	value := NewStringValue(v)
	value.modified = false
	return value
}

func (d *DecoderV4) decodeArray(path []string) (*ArrayValue, error) {
	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid array encoding (@ %s): expected []interface{}, got %s",
				strings.Join(path, "."),
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	values := make([]Value, size)

	// Pre-allocate and reuse valuePath.
	valuePath := append(path, "")

	lastValuePathIndex := len(path)

	for i := 0; i < int(size); i++ {
		valuePath[lastValuePathIndex] = strconv.Itoa(i)

		res, err := d.decodeValue(valuePath)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid array element encoding (@ %s, %d): %w",
				strings.Join(path, "."),
				i,
				err,
			)
		}
		values[i] = res
	}

	return &ArrayValue{
		Values:   values,
		Owner:    d.owner,
		modified: false,
	}, nil
}

func (d *DecoderV4) decodeDictionary(path []string) (*DictionaryValue, error) {

	const expectedLength = encodedDictionaryValueLength

	size, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid dictionary encoding (@ %s): expected [%d]interface{}, got %s",
				strings.Join(path, "."),
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, fmt.Errorf("invalid dictionary encoding (@ %s): expected [%d]interface{}, got [%d]interface{}",
			strings.Join(path, "."),
			expectedLength,
			size,
		)
	}

	// Decode keys at array index encodedDictionaryValueKeysFieldKey
	keysPath := append(path, dictionaryKeyPathPrefix)
	keys, err := d.decodeArray(keysPath)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid dictionary keys encoding (@ %s): %w",
			strings.Join(path, "."),
			err,
		)
	}

	// Decode entries at array index encodedDictionaryValueEntriesFieldKey
	entryCount, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid dictionary entries encoding (@ %s): %s",
				strings.Join(path, "."),
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	keyCount := keys.Count()

	// The number of entries must either match the number of keys,
	// or be zero in case the values are deferred

	countMismatch := int(entryCount) != keyCount
	if countMismatch && entryCount != 0 {
		return nil, fmt.Errorf(
			"invalid dictionary encoding (@ %s): key and entry count mismatch: expected %d, got %d",
			strings.Join(path, "."),
			keyCount,
			entryCount,
		)
	}

	entries := NewStringValueOrderedMap()

	var deferred *orderedmap.StringStructOrderedMap
	var deferredOwner *common.Address
	var deferredStorageKeyBase string

	// Are the values in the dictionary deferred, i.e. are they encoded
	// separately and stored in separate storage keys?

	isDeferred := countMismatch && entryCount == 0

	if isDeferred {

		deferred = orderedmap.NewStringStructOrderedMap()
		deferredOwner = d.owner
		deferredStorageKeyBase = joinPath(append(path, dictionaryValuePathPrefix))
		for _, keyValue := range keys.Values {
			key := dictionaryKey(keyValue)
			deferred.Set(key, struct{}{})
		}

	} else {

		// Pre-allocate and reuse valuePath.
		valuePath := append(path, dictionaryValuePathPrefix, "")

		lastValuePathIndex := len(path) + 1

		keyIndex := 0

		for _, keyValue := range keys.Values {
			keyStringValue, ok := keyValue.(HasKeyString)
			if !ok {
				return nil, fmt.Errorf(
					"invalid dictionary key encoding (@ %s, %d): %T",
					strings.Join(path, "."),
					keyIndex,
					keyValue,
				)
			}

			keyString := keyStringValue.KeyString()
			valuePath[lastValuePathIndex] = keyString

			decodedValue, err := d.decodeValue(valuePath)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid dictionary value encoding (@ %s, %s): %w",
					strings.Join(path, "."),
					keyString,
					err,
				)
			}

			entries.Set(keyString, decodedValue)

			keyIndex++
		}
	}

	return &DictionaryValue{
		Keys:                   keys,
		Entries:                entries,
		Owner:                  d.owner,
		modified:               false,
		DeferredOwner:          deferredOwner,
		DeferredKeys:           deferred,
		DeferredStorageKeyBase: deferredStorageKeyBase,
	}, nil
}

func (d *DecoderV4) decodeLocation() (common.Location, error) {
	number, err := d.decoder.DecodeTagNumber()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid location encoding: %s", e.ActualType.String())
		}
		return nil, err
	}

	switch number {
	case cborTagAddressLocation:
		return d.decodeAddressLocation()

	case cborTagStringLocation:
		return d.decodeStringLocation()

	case cborTagIdentifierLocation:
		return d.decodeIdentifierLocation()

	default:
		return nil, fmt.Errorf("invalid location encoding tag: %d", number)
	}
}

func (d *DecoderV4) decodeStringLocation() (common.Location, error) {
	s, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid string location encoding: %s", e.ActualType.String())
		}
		return nil, err
	}
	return common.StringLocation(s), nil
}

func (d *DecoderV4) decodeIdentifierLocation() (common.Location, error) {
	s, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid identifier location encoding: %s", e.ActualType.String())
		}
		return nil, err
	}
	return common.IdentifierLocation(s), nil
}

func (d *DecoderV4) decodeAddressLocation() (common.Location, error) {

	const expectedLength = encodedAddressLocationLength

	size, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid address location encoding: expected [%d]interface{}, got %s",
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
			return nil, fmt.Errorf("invalid address location address encoding: %s", e.ActualType.String())
		}
		return nil, err
	}

	err = d.checkAddressLength(encodedAddress)
	if err != nil {
		return nil, err
	}

	// Name

	// Decode name at array index encodedAddressLocationNameFieldKey
	name, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid address location name encoding: %s", e.ActualType.String())
		}
		return nil, err
	}

	return common.AddressLocation{
		Address: common.BytesToAddress(encodedAddress),
		Name:    name,
	}, nil
}

func (d *DecoderV4) decodeComposite(path []string) (*CompositeValue, error) {

	const expectedLength = encodedCompositeValueLength

	size, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid composite encoding (@ %s): expected [%d]interface{}, got %s",
				strings.Join(path, "."),
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, fmt.Errorf("invalid composite encoding (@ %s): expected [%d]interface{}, got [%d]interface{}",
			strings.Join(path, "."),
			expectedLength,
			size,
		)
	}

	// Location

	// Decode location at array index encodedCompositeValueLocationFieldKey
	location, err := d.decodeLocation()
	if err != nil {
		return nil, fmt.Errorf(
			"invalid composite location encoding (@ %s): %w",
			strings.Join(path, "."),
			err,
		)
	}

	// Skip obsolete element at array index encodedCompositeValueTypeIDFieldKey
	err = d.decoder.Skip()
	if err != nil {
		return nil, err
	}

	// Kind

	// Decode kind at array index encodedCompositeValueKindFieldKey
	encodedKind, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite kind encoding (@ %s): %s",
				strings.Join(path, "."),
				e.ActualType.String(),
			)
		}
		return nil, err
	}
	kind := common.CompositeKind(encodedKind)

	// Fields

	// Decode fields at array index encodedCompositeValueFieldsFieldKey
	fieldsSize, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite fields encoding (@ %s): %s",
				strings.Join(path, "."),
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if fieldsSize%2 == 1 {
		return nil, fmt.Errorf(
			"invalid composite fields encoding (@ %s): fields should have even number of elements: got %d",
			strings.Join(path, "."),
			fieldsSize,
		)
	}

	fields := NewStringValueOrderedMap()

	// Pre-allocate and reuse valuePath.
	valuePath := append(path, "")

	lastValuePathIndex := len(path)

	for i := 0; i < int(fieldsSize); i += 2 {

		// field name
		fieldName, err := d.decoder.DecodeString()
		if err != nil {
			if e, ok := err.(*cbor.WrongTypeError); ok {
				return nil, fmt.Errorf(
					"invalid composite field name encoding (@ %s, %d): %s",
					strings.Join(path, "."),
					i/2,
					e.ActualType.String(),
				)
			}
			return nil, err
		}

		// field value

		valuePath[lastValuePathIndex] = fieldName

		decodedValue, err := d.decodeValue(valuePath)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid composite field value encoding (@ %s, %s): %w",
				strings.Join(path, "."),
				fieldName,
				err,
			)
		}

		fields.Set(fieldName, decodedValue)
	}

	// Qualified identifier

	// Decode qualified identifier at array index encodedCompositeValueQualifiedIdentifierFieldKey
	qualifiedIdentifier, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid composite qualified identifier encoding (@ %s): %s",
				strings.Join(path, "."),
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	compositeValue := NewCompositeValue(location, qualifiedIdentifier, kind, fields, d.owner)
	compositeValue.modified = false
	return compositeValue, nil
}

var bigOne = big.NewInt(1)

func (d *DecoderV4) decodeInt() (IntValue, error) {
	bigInt, err := d.decoder.DecodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return IntValue{}, fmt.Errorf("invalid Int encoding: %s", e.ActualType.String())
		}
		return IntValue{}, err
	}

	return NewIntValueFromBigInt(bigInt), nil
}

func (d *DecoderV4) decodeInt8() (Int8Value, error) {
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

func (d *DecoderV4) decodeInt16() (Int16Value, error) {
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

func (d *DecoderV4) decodeInt32() (Int32Value, error) {
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

func (d *DecoderV4) decodeInt64() (Int64Value, error) {
	v, err := d.decoder.DecodeInt64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Int64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}

	return Int64Value(v), nil
}

func (d *DecoderV4) decodeInt128() (Int128Value, error) {
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

func (d *DecoderV4) decodeInt256() (Int256Value, error) {
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

func (d *DecoderV4) decodeUInt() (UIntValue, error) {
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

func (d *DecoderV4) decodeUInt8() (UInt8Value, error) {
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

func (d *DecoderV4) decodeUInt16() (UInt16Value, error) {
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

func (d *DecoderV4) decodeUInt32() (UInt32Value, error) {
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

func (d *DecoderV4) decodeUInt64() (UInt64Value, error) {
	value, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UInt64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return UInt64Value(value), nil
}

func (d *DecoderV4) decodeUInt128() (UInt128Value, error) {
	bigInt, err := d.decoder.DecodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return UInt128Value{}, fmt.Errorf("invalid UInt128 encoding: %s", e.ActualType.String())
		}
		return UInt128Value{}, nil
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

func (d *DecoderV4) decodeUInt256() (UInt256Value, error) {
	bigInt, err := d.decoder.DecodeBigInt()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return UInt256Value{}, fmt.Errorf("invalid UInt256 encoding: %s", e.ActualType.String())
		}
		return UInt256Value{}, nil
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

func (d *DecoderV4) decodeWord8() (Word8Value, error) {
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

func (d *DecoderV4) decodeWord16() (Word16Value, error) {
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

func (d *DecoderV4) decodeWord32() (Word32Value, error) {
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

func (d *DecoderV4) decodeWord64() (Word64Value, error) {
	value, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Word64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return Word64Value(value), nil
}

func (d *DecoderV4) decodeFix64() (Fix64Value, error) {
	value, err := d.decoder.DecodeInt64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown Fix64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return Fix64Value(value), nil
}

func (d *DecoderV4) decodeUFix64() (UFix64Value, error) {
	value, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return 0, fmt.Errorf("unknown UFix64 encoding: %s", e.ActualType.String())
		}
		return 0, err
	}
	return UFix64Value(value), nil
}

func (d *DecoderV4) decodeSome(path []string) (*SomeValue, error) {
	value, err := d.decodeValue(path)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid some value encoding (@ %s): %w",
			strings.Join(path, "."),
			err,
		)
	}

	return &SomeValue{
		Value: value,
		Owner: d.owner,
	}, nil
}

func (d *DecoderV4) checkAddressLength(addressBytes []byte) error {
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

func (d *DecoderV4) decodeAddress() (AddressValue, error) {
	addressBytes, err := d.decoder.DecodeBytes()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return AddressValue{}, fmt.Errorf("invalid address encoding: %s", e.ActualType.String())
		}
		return AddressValue{}, nil
	}

	err = d.checkAddressLength(addressBytes)
	if err != nil {
		return AddressValue{}, err
	}

	address := NewAddressValueFromBytes(addressBytes)
	return address, nil
}

func (d *DecoderV4) decodePath() (PathValue, error) {

	const expectedLength = encodedPathValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PathValue{}, fmt.Errorf("invalid path encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return PathValue{}, err
	}

	if size != expectedLength {
		return PathValue{}, fmt.Errorf("invalid path encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode domain at array index encodedPathValueDomainFieldKey
	domain, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PathValue{}, fmt.Errorf("invalid path domain encoding: %s", e.ActualType.String())
		}
		return PathValue{}, err
	}

	// Decode identifier at array index encodedPathValueIdentifierFieldKey
	identifier, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return PathValue{}, fmt.Errorf("invalid path identifier encoding: %s", e.ActualType.String())
		}
		return PathValue{}, err
	}

	return PathValue{
		Domain:     common.PathDomain(domain),
		Identifier: identifier,
	}, nil
}

func (d *DecoderV4) decodeCapability() (CapabilityValue, error) {

	const expectedLength = encodedCapabilityValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return CapabilityValue{}, fmt.Errorf("invalid capability encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return CapabilityValue{}, err
	}

	if size != expectedLength {
		return CapabilityValue{}, fmt.Errorf("invalid capability encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// address

	// Decode address at array index encodedCapabilityValueAddressFieldKey
	field1Value, err := d.decodeValue(nil)
	if err != nil {
		return CapabilityValue{}, fmt.Errorf("invalid capability address: %w", err)
	}

	address, ok := field1Value.(AddressValue)
	if !ok {
		return CapabilityValue{}, fmt.Errorf("invalid capability address: %T", address)
	}

	// path

	// Decode path at array index encodedCapabilityValuePathFieldKey
	field2Value, err := d.decodeValue(nil)
	if err != nil {
		return CapabilityValue{}, fmt.Errorf("invalid capability path: %w", err)
	}

	path, ok := field2Value.(PathValue)
	if !ok {
		return CapabilityValue{}, fmt.Errorf("invalid capability path: %T", path)
	}

	// Decode borrow type at array index encodedCapabilityValueBorrowTypeFieldKey

	// borrow type (optional, for backwards compatibility)
	// Capabilities used to be untyped, i.e. they didn't have a borrow type.
	// Later an optional type paramater, the borrow type, was added to it,
	// which specifies as what type the capability should be borrowed.
	//
	// The decoding must be backwards-compatible and support both capability values
	// with a borrow type and ones without
	borrowType, err := d.decodeStaticType()
	if err != nil {
		return CapabilityValue{}, fmt.Errorf("invalid capability borrow type encoding: %w", err)
	}

	return CapabilityValue{
		Address:    address,
		Path:       path,
		BorrowType: borrowType,
	}, nil
}

func (d *DecoderV4) decodeLink() (LinkValue, error) {

	const expectedLength = encodedLinkValueLength

	size, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return LinkValue{}, fmt.Errorf("invalid link encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return LinkValue{}, err
	}

	if size != expectedLength {
		return LinkValue{}, fmt.Errorf("invalid link encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode path at array index encodedLinkValueTargetPathFieldKey
	decodedPath, err := d.decodeValue(nil)
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link target path encoding: %w", err)
	}

	pathValue, ok := decodedPath.(PathValue)
	if !ok {
		return LinkValue{}, fmt.Errorf("invalid link target path encoding: %T", decodedPath)
	}

	// Decode type at array index encodedLinkValueTypeFieldKey
	staticType, err := d.decodeStaticType()
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link type encoding: %w", err)
	}

	return LinkValue{
		TargetPath: pathValue,
		Type:       staticType,
	}, nil
}

func (d *DecoderV4) decodeStaticType() (StaticType, error) {
	number, err := d.decoder.DecodeTagNumber()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {

			// CBOR nil is interpreted as nil static type.
			if e.ActualType == cbor.NilType {
				if err := d.decoder.Skip(); err != nil {
					return nil, err
				}
				return nil, nil
			}

			return nil, fmt.Errorf("invalid static type encoding: %s", e.ActualType.String())
		}
		return nil, err
	}

	switch number {
	case cborTagPrimitiveStaticType:
		return d.decodePrimitiveStaticType()

	case cborTagOptionalStaticType:
		return d.decodeOptionalStaticType()

	case cborTagCompositeStaticType:
		return d.decodeCompositeStaticType()

	case cborTagInterfaceStaticType:
		return d.decodeInterfaceStaticType()

	case cborTagVariableSizedStaticType:
		return d.decodeVariableSizedStaticType()

	case cborTagConstantSizedStaticType:
		return d.decodeConstantSizedStaticType()

	case cborTagReferenceStaticType:
		return d.decodeReferenceStaticType()

	case cborTagDictionaryStaticType:
		return d.decodeDictionaryStaticType()

	case cborTagRestrictedStaticType:
		return d.decodeRestrictedStaticType()

	case cborTagCapabilityStaticType:
		return d.decodeCapabilityStaticType()

	default:
		return nil, fmt.Errorf("invalid static type encoding tag: %d", number)
	}
}

func (d *DecoderV4) decodePrimitiveStaticType() (PrimitiveStaticType, error) {
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

func (d *DecoderV4) decodeOptionalStaticType() (StaticType, error) {
	staticType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf("invalid optional static type inner type encoding: %w", err)
	}
	return OptionalStaticType{
		Type: staticType,
	}, nil
}

func (d *DecoderV4) decodeCompositeStaticType() (StaticType, error) {
	const expectedLength = encodedCompositeStaticTypeLength

	size, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid composite static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, fmt.Errorf("invalid composite static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode location at array index encodedCompositeStaticTypeLocationFieldKey
	location, err := d.decodeLocation()
	if err != nil {
		return nil, fmt.Errorf("invalid composite static type location encoding: %w", err)
	}

	// Skip obsolete element at array index encodedCompositeStaticTypeTypeIDFieldKey
	err = d.decoder.Skip()
	if err != nil {
		return nil, err
	}

	// Decode qualified identifier at array index encodedCompositeStaticTypeQualifiedIdentifierFieldKey
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

func (d *DecoderV4) decodeInterfaceStaticType() (StaticType, error) {
	const expectedLength = encodedInterfaceStaticTypeLength

	size, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid interface static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if size != expectedLength {
		return nil, fmt.Errorf("invalid interface static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			size,
		)
	}

	// Decode location at array index encodedInterfaceStaticTypeLocationFieldKey
	location, err := d.decodeLocation()
	if err != nil {
		return nil, fmt.Errorf("invalid interface static type location encoding: %w", err)
	}

	// Skip obsolete element at array index encodedInterfaceStaticTypeTypeIDFieldKey
	err = d.decoder.Skip()
	if err != nil {
		return nil, err
	}

	// Decode qualified identifier at array index encodedInterfaceStaticTypeQualifiedIdentifierFieldKey
	qualifiedIdentifier, err := d.decoder.DecodeString()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf(
				"invalid interface static type qualified identifier encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	return InterfaceStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
	}, nil
}

func (d *DecoderV4) decodeVariableSizedStaticType() (StaticType, error) {
	staticType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf("invalid variable-sized static type encoding: %w", err)
	}
	return VariableSizedStaticType{
		Type: staticType,
	}, nil
}

func (d *DecoderV4) decodeConstantSizedStaticType() (StaticType, error) {

	const expectedLength = encodedConstantSizedStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid constant-sized static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, fmt.Errorf("invalid constant-sized static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode size at array index encodedConstantSizedStaticTypeSizeFieldKey
	size, err := d.decoder.DecodeUint64()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid constant-sized static type size encoding: %s", e.ActualType.String())
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
	staticType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf("invalid constant-sized static type inner type encoding: %w", err)
	}

	return ConstantSizedStaticType{
		Type: staticType,
		Size: int64(size),
	}, nil
}

func (d *DecoderV4) decodeReferenceStaticType() (StaticType, error) {
	const expectedLength = encodedReferenceStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid reference static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, fmt.Errorf("invalid reference static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode authorized at array index encodedReferenceStaticTypeAuthorizedFieldKey
	authorized, err := d.decoder.DecodeBool()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid reference static type authorized encoding: %s", e.ActualType.String())
		}
		return nil, err
	}

	// Decode type at array index encodedReferenceStaticTypeTypeFieldKey
	staticType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf("invalid reference static type inner type encoding: %w", err)
	}

	return ReferenceStaticType{
		Authorized: authorized,
		Type:       staticType,
	}, nil
}

func (d *DecoderV4) decodeDictionaryStaticType() (StaticType, error) {
	const expectedLength = encodedDictionaryStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid dictionary static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, fmt.Errorf("invalid dictionary static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode key type at array index encodedDictionaryStaticTypeKeyTypeFieldKey
	keyType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf("invalid dictionary static type key type encoding: %w", err)
	}

	// Decode value type at array index encodedDictionaryStaticTypeValueTypeFieldKey
	valueType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf("invalid dictionary static type value type encoding: %w", err)
	}

	return DictionaryStaticType{
		KeyType:   keyType,
		ValueType: valueType,
	}, nil
}

func (d *DecoderV4) decodeRestrictedStaticType() (StaticType, error) {
	const expectedLength = encodedRestrictedStaticTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid restricted static type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	if arraySize != expectedLength {
		return nil, fmt.Errorf("invalid restricted static type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode restricted type at array index encodedRestrictedStaticTypeTypeFieldKey
	restrictedType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf("invalid restricted static type key type encoding: %w", err)
	}

	// Decode restrictions at array index encodedRestrictedStaticTypeRestrictionsFieldKey
	restrictionSize, err := d.decoder.DecodeArrayHead()
	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return nil, fmt.Errorf("invalid restricted static type restrictions encoding: %s",
				e.ActualType.String(),
			)
		}
		return nil, err
	}

	restrictions := make([]InterfaceStaticType, restrictionSize)
	for i := 0; i < int(restrictionSize); i++ {
		r, err := d.decodeStaticType()
		if err != nil {
			return nil, err
		}
		restriction, ok := r.(InterfaceStaticType)
		if !ok {
			return nil, fmt.Errorf("invalid restricted static type restriction encoding: %T", r)
		}
		restrictions[i] = restriction
	}

	return &RestrictedStaticType{
		Type:         restrictedType,
		Restrictions: restrictions,
	}, nil
}

func (d *DecoderV4) decodeType() (TypeValue, error) {
	const expectedLength = encodedTypeValueTypeLength

	arraySize, err := d.decoder.DecodeArrayHead()

	if err != nil {
		if e, ok := err.(*cbor.WrongTypeError); ok {
			return TypeValue{}, fmt.Errorf("invalid type encoding: expected [%d]interface{}, got %s",
				expectedLength,
				e.ActualType.String(),
			)
		}
		return TypeValue{}, err
	}

	if arraySize != expectedLength {
		return TypeValue{}, fmt.Errorf("invalid type encoding: expected [%d]interface{}, got [%d]interface{}",
			expectedLength,
			arraySize,
		)
	}

	// Decode type at array index encodedTypeValueTypeFieldKey
	staticType, err := d.decodeStaticType()
	if err != nil {
		return TypeValue{}, fmt.Errorf("invalid type encoding: %w", err)
	}

	return TypeValue{
		Type: staticType,
	}, nil
}

func (d *DecoderV4) decodeCapabilityStaticType() (StaticType, error) {
	borrowStaticType, err := d.decodeStaticType()
	if err != nil {
		return nil, fmt.Errorf("invalid capability static type borrow type encoding: %w", err)
	}

	return CapabilityStaticType{
		BorrowType: borrowStaticType,
	}, nil
}
