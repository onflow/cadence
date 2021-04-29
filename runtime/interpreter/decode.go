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
	"bytes"
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
	decoder        *cbor.Decoder
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
	reader := bytes.NewReader(data)

	decoder, err := NewDecoder(reader, owner, version, decodeCallback)
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
		decoder:        decMode.NewDecoder(reader),
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

// Decode reads CBOR-encoded bytes from the io.Reader and decodes them to a value.
//
func (d *DecoderV4) Decode(path []string) (Value, error) {
	var v interface{}
	err := d.decoder.Decode(&v)
	if err != nil {
		return nil, err
	}

	return d.decodeValue(v, path)
}

func (d *DecoderV4) decodeValue(v interface{}, path []string) (Value, error) {

	if d.decodeCallback != nil {
		d.decodeCallback(v, path)
	}

	switch v := v.(type) {

	// CBOR Types

	case bool:
		return BoolValue(v), nil

	case string:
		return d.decodeString(v), nil

	case nil:
		return NilValue{}, nil

	case []interface{}:
		return d.decodeArray(v, path)

	case cbor.Tag:
		switch v.Number {

		case cborTagVoidValue:
			return VoidValue{}, nil

		case cborTagDictionaryValue:
			return d.decodeDictionary(v.Content, path)

		case cborTagSomeValue:
			return d.decodeSome(v.Content, path)

		case cborTagAddressValue:
			return d.decodeAddress(v.Content)

		case cborTagCompositeValue:
			return d.decodeComposite(v.Content, path)

		// Int*

		case cborTagIntValue:
			return d.decodeInt(v.Content)

		case cborTagInt8Value:
			return d.decodeInt8(v.Content)

		case cborTagInt16Value:
			return d.decodeInt16(v.Content)

		case cborTagInt32Value:
			return d.decodeInt32(v.Content)

		case cborTagInt64Value:
			return d.decodeInt64(v.Content)

		case cborTagInt128Value:
			return d.decodeInt128(v.Content)

		case cborTagInt256Value:
			return d.decodeInt256(v.Content)

		// UInt*

		case cborTagUIntValue:
			return d.decodeUInt(v.Content)

		case cborTagUInt8Value:
			return d.decodeUInt8(v.Content)

		case cborTagUInt16Value:
			return d.decodeUInt16(v.Content)

		case cborTagUInt32Value:
			return d.decodeUInt32(v.Content)

		case cborTagUInt64Value:
			return d.decodeUInt64(v.Content)

		case cborTagUInt128Value:
			return d.decodeUInt128(v.Content)

		case cborTagUInt256Value:
			return d.decodeUInt256(v.Content)

		// Word*

		case cborTagWord8Value:
			return d.decodeWord8(v.Content)

		case cborTagWord16Value:
			return d.decodeWord16(v.Content)

		case cborTagWord32Value:
			return d.decodeWord32(v.Content)

		case cborTagWord64Value:
			return d.decodeWord64(v.Content)

		// Fix*

		case cborTagFix64Value:
			return d.decodeFix64(v.Content)

		// UFix*

		case cborTagUFix64Value:
			return d.decodeUFix64(v.Content)

		// Storage

		case cborTagPathValue:
			return d.decodePath(v.Content)

		case cborTagCapabilityValue:
			return d.decodeCapability(v.Content)

		case cborTagLinkValue:
			return d.decodeLink(v.Content)

		case cborTagTypeValue:
			return d.decodeType(v.Content)

		default:
			return nil, fmt.Errorf(
				"unsupported decoded tag (@ %s): %d, %v",
				strings.Join(path, "."),
				v.Number,
				v.Content,
			)
		}

	default:
		return nil, fmt.Errorf(
			"unsupported decoded type (@ %s): %[2]T, %[2]v",
			strings.Join(path, "."),
			v,
		)
	}
}

func (d *DecoderV4) decodeString(v string) Value {
	value := NewStringValue(v)
	value.modified = false
	return value
}

func (d *DecoderV4) decodeArray(v []interface{}, path []string) (*ArrayValue, error) {
	values := make([]Value, len(v))

	for i, value := range v {
		valuePath := append(path[:], strconv.Itoa(i))
		res, err := d.decodeValue(value, valuePath)
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

func (d *DecoderV4) decodeDictionary(v interface{}, path []string) (*DictionaryValue, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedDictionaryValueLength
	if !ok || len(encoded) != expectedLength {
		return nil, fmt.Errorf("invalid dictionary encoding (@ %s): expected [%d]interface{}, got %T",
			strings.Join(path, "."),
			expectedLength,
			v,
		)
	}

	keysField := encoded[encodedDictionaryValueKeysFieldKey]
	encodedKeys, ok := keysField.(cborArray)
	if !ok {
		return nil, fmt.Errorf(
			"invalid dictionary keys encoding (@ %s): %T",
			strings.Join(path, "."),
			keysField,
		)
	}

	keysPath := append(path[:], dictionaryKeyPathPrefix)
	keys, err := d.decodeArray(encodedKeys, keysPath)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid dictionary keys encoding (@ %s): %w",
			strings.Join(path, "."),
			err,
		)
	}

	entriesField := encoded[encodedDictionaryValueEntriesFieldKey]
	encodedEntries, ok := entriesField.(cborArray)
	if !ok {
		return nil, fmt.Errorf(
			"invalid dictionary entries encoding (@ %s): %T",
			strings.Join(path, "."),
			entriesField,
		)
	}

	keyCount := keys.Count()
	entryCount := len(encodedEntries)

	// The number of entries must either match the number of keys,
	// or be zero in case the values are deferred

	countMismatch := entryCount != keyCount
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
		deferredStorageKeyBase = joinPath(append(path[:], dictionaryValuePathPrefix))
		for _, keyValue := range keys.Values {
			key := dictionaryKey(keyValue)
			deferred.Set(key, struct{}{})
		}

	} else {

		index := 0

		for _, keyValue := range keys.Values {
			keyStringValue, ok := keyValue.(HasKeyString)
			if !ok {
				return nil, fmt.Errorf(
					"invalid dictionary key encoding (@ %s, %d): %T",
					strings.Join(path, "."),
					index,
					keyValue,
				)
			}

			keyString := keyStringValue.KeyString()

			value := encodedEntries[index]

			valuePath := append(path[:], dictionaryValuePathPrefix, keyString)
			decodedValue, err := d.decodeValue(value, valuePath)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid dictionary value encoding (@ %s, %s): %w",
					strings.Join(path, "."),
					keyString,
					err,
				)
			}

			entries.Set(keyString, decodedValue)

			index++
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

func (d *DecoderV4) decodeLocation(l interface{}) (common.Location, error) {
	tag, ok := l.(cbor.Tag)
	if !ok {
		return nil, fmt.Errorf("invalid location encoding: %T", l)
	}

	content := tag.Content

	switch tag.Number {
	case cborTagAddressLocation:
		return d.decodeAddressLocation(content)

	case cborTagStringLocation:
		return d.decodeStringLocation(content)

	case cborTagIdentifierLocation:
		return d.decodeIdentifierLocation(content)

	default:
		return nil, fmt.Errorf("invalid location encoding tag: %d", tag.Number)
	}
}

func (d *DecoderV4) decodeStringLocation(v interface{}) (common.Location, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("invalid string location encoding: %T", v)
	}
	return common.StringLocation(s), nil
}

func (d *DecoderV4) decodeIdentifierLocation(v interface{}) (common.Location, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("invalid identifier location encoding: %T", v)
	}
	return common.IdentifierLocation(s), nil
}

func (d *DecoderV4) decodeAddressLocation(v interface{}) (common.Location, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedAddressLocationLength
	if !ok || len(encoded) != expectedLength {
		return nil, fmt.Errorf("invalid address location encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	// Address

	field1 := encoded[encodedAddressLocationAddressFieldKey]
	encodedAddress, ok := field1.([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid address location address encoding: %T", field1)
	}

	err := d.checkAddressLength(encodedAddress)
	if err != nil {
		return nil, err
	}

	// Name

	field2 := encoded[encodedAddressLocationNameFieldKey]
	name, ok := field2.(string)
	if !ok {
		return nil, fmt.Errorf("invalid address location name encoding: %T", field2)
	}

	return common.AddressLocation{
		Address: common.BytesToAddress(encodedAddress),
		Name:    name,
	}, nil
}

func (d *DecoderV4) decodeComposite(v interface{}, path []string) (*CompositeValue, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedCompositeValueLength
	if !ok || len(encoded) != expectedLength {
		return nil, fmt.Errorf("invalid composite encoding (@ %s): expected [%d]interface{}, got %T",
			strings.Join(path, "."),
			expectedLength,
			v,
		)
	}

	// Location

	location, err := d.decodeLocation(encoded[encodedCompositeValueLocationFieldKey])
	if err != nil {
		return nil, fmt.Errorf(
			"invalid composite location encoding (@ %s): %w",
			strings.Join(path, "."),
			err,
		)
	}

	// Qualified identifier

	qualifiedIdentifierField := encoded[encodedCompositeValueQualifiedIdentifierFieldKey]
	qualifiedIdentifier, ok := qualifiedIdentifierField.(string)
	if !ok {
		return nil, fmt.Errorf(
			"invalid composite qualified identifier encoding (@ %s): %T",
			strings.Join(path, "."),
			qualifiedIdentifierField,
		)
	}

	// Kind

	kindField := encoded[encodedCompositeValueKindFieldKey]
	encodedKind, ok := kindField.(uint64)
	if !ok {
		return nil, fmt.Errorf(
			"invalid composite kind encoding (@ %s): %T",
			strings.Join(path, "."),
			kindField,
		)
	}
	kind := common.CompositeKind(encodedKind)

	// Fields

	fieldsField := encoded[encodedCompositeValueFieldsFieldKey]
	encodedFields, ok := fieldsField.(cborArray)
	if !ok {
		return nil, fmt.Errorf(
			"invalid composite fields encoding (@ %s): %T",
			strings.Join(path, "."),
			fieldsField,
		)
	}

	if len(encodedFields)%2 == 1 {
		return nil, fmt.Errorf(
			"invalid composite fields encoding (@ %s): fields should have even number of elements: got %d",
			strings.Join(path, "."),
			len(encodedFields),
		)
	}

	fields := NewStringValueOrderedMap()

	for i := 0; i < len(encodedFields); i += 2 {

		// field name
		fieldName, ok := encodedFields[i].(string)
		if !ok {
			return nil, fmt.Errorf(
				"invalid composite field name encoding (@ %s, %d): %T",
				strings.Join(path, "."),
				i/2,
				encodedFields[i],
			)
		}

		// field value
		value := encodedFields[i+1]

		valuePath := append(path[:], fieldName)
		decodedValue, err := d.decodeValue(value, valuePath)
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

	compositeValue := NewCompositeValue(location, qualifiedIdentifier, kind, fields, d.owner)
	compositeValue.modified = false
	return compositeValue, nil
}

var bigOne = big.NewInt(1)

func (d *DecoderV4) decodeBig(v interface{}) (*big.Int, error) {
	bigInt, ok := v.(big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid bignum encoding: %T", v)
	}

	return &bigInt, nil
}

func (d *DecoderV4) decodeInt(v interface{}) (IntValue, error) {
	bigInt, err := d.decodeBig(v)
	if err != nil {
		return IntValue{}, fmt.Errorf("invalid Int encoding: %w", err)
	}

	return NewIntValueFromBigInt(bigInt), nil
}

func (d *DecoderV4) decodeInt8(v interface{}) (Int8Value, error) {
	switch v := v.(type) {
	case uint64:
		const max = math.MaxInt8
		if v > max {
			return 0, fmt.Errorf("invalid Int8: got %d, expected max %d", v, max)
		}
		return Int8Value(v), nil

	case int64:
		const min = math.MinInt8
		if v < min {
			return 0, fmt.Errorf("invalid Int8: got %d, expected min %d", v, min)
		}
		const max = math.MaxInt8
		if v > max {
			return 0, fmt.Errorf("invalid Int8: got %d, expected max %d", v, max)
		}
		return Int8Value(v), nil

	default:
		return 0, fmt.Errorf("unknown Int8 encoding: %T", v)
	}
}

func (d *DecoderV4) decodeInt16(v interface{}) (Int16Value, error) {
	switch v := v.(type) {
	case uint64:
		const max = math.MaxInt16
		if v > max {
			return 0, fmt.Errorf("invalid Int16: got %d, expected max %d", v, max)
		}
		return Int16Value(v), nil

	case int64:
		const min = math.MinInt16
		if v < min {
			return 0, fmt.Errorf("invalid Int16: got %d, expected min %d", v, min)
		}
		const max = math.MaxInt16
		if v > max {
			return 0, fmt.Errorf("invalid Int16: got %d, expected max %d", v, max)
		}
		return Int16Value(v), nil

	default:
		return 0, fmt.Errorf("unknown Int16 encoding: %T", v)
	}
}

func (d *DecoderV4) decodeInt32(v interface{}) (Int32Value, error) {
	switch v := v.(type) {
	case uint64:
		const max = math.MaxInt32
		if v > max {
			return 0, fmt.Errorf("invalid Int32: got %d, expected max %d", v, max)
		}
		return Int32Value(v), nil

	case int64:
		const min = math.MinInt32
		if v < min {
			return 0, fmt.Errorf("invalid Int32: got %d, expected min %d", v, min)
		}
		const max = math.MaxInt32
		if v > max {
			return 0, fmt.Errorf("invalid Int32: got %d, expected max %d", v, max)
		}
		return Int32Value(v), nil

	default:
		return 0, fmt.Errorf("unknown Int32 encoding: %T", v)
	}
}

func (d *DecoderV4) decodeInt64(v interface{}) (Int64Value, error) {
	switch v := v.(type) {
	case uint64:
		const max = math.MaxInt64
		if v > max {
			return 0, fmt.Errorf("invalid Int64: got %d, expected max %d", v, max)
		}
		return Int64Value(v), nil

	case int64:
		return Int64Value(v), nil

	default:
		return 0, fmt.Errorf("unknown Int64 encoding: %T", v)
	}
}

func (d *DecoderV4) decodeInt128(v interface{}) (Int128Value, error) {
	bigInt, err := d.decodeBig(v)
	if err != nil {
		return Int128Value{}, fmt.Errorf("invalid Int encoding: %w", err)
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

func (d *DecoderV4) decodeInt256(v interface{}) (Int256Value, error) {
	bigInt, err := d.decodeBig(v)
	if err != nil {
		return Int256Value{}, fmt.Errorf("invalid Int encoding: %w", err)
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

func (d *DecoderV4) decodeUInt(v interface{}) (UIntValue, error) {
	bigInt, err := d.decodeBig(v)
	if err != nil {
		return UIntValue{}, fmt.Errorf("invalid UInt encoding: %w", err)
	}

	if bigInt.Sign() < 0 {
		return UIntValue{}, fmt.Errorf("invalid UInt: got %s, expected positive", bigInt)
	}

	return NewUIntValueFromBigInt(bigInt), nil
}

func (d *DecoderV4) decodeUInt8(v interface{}) (UInt8Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown UInt8 encoding: %T", v)
	}
	const max = math.MaxUint8
	if value > max {
		return 0, fmt.Errorf("invalid UInt8: got %d, expected max %d", v, max)
	}
	return UInt8Value(value), nil
}

func (d *DecoderV4) decodeUInt16(v interface{}) (UInt16Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown UInt16 encoding: %T", v)
	}
	const max = math.MaxUint16
	if value > max {
		return 0, fmt.Errorf("invalid UInt16: got %d, expected max %d", v, max)
	}
	return UInt16Value(value), nil
}

func (d *DecoderV4) decodeUInt32(v interface{}) (UInt32Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown UInt32 encoding: %T", v)
	}
	const max = math.MaxUint32
	if value > max {
		return 0, fmt.Errorf("invalid UInt32: got %d, expected max %d", v, max)
	}
	return UInt32Value(value), nil
}

func (d *DecoderV4) decodeUInt64(v interface{}) (UInt64Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown UInt64 encoding: %T", v)
	}
	return UInt64Value(value), nil
}

func (d *DecoderV4) decodeUInt128(v interface{}) (UInt128Value, error) {
	bigInt, err := d.decodeBig(v)
	if err != nil {
		return UInt128Value{}, fmt.Errorf("invalid Int encoding: %w", err)
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

func (d *DecoderV4) decodeUInt256(v interface{}) (UInt256Value, error) {
	bigInt, err := d.decodeBig(v)
	if err != nil {
		return UInt256Value{}, fmt.Errorf("invalid Int encoding: %w", err)
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

func (d *DecoderV4) decodeWord8(v interface{}) (Word8Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown Word8 encoding: %T", v)
	}
	const max = math.MaxUint8
	if value > max {
		return 0, fmt.Errorf("invalid Word8: got %d, expected max %d", v, max)
	}
	return Word8Value(value), nil
}

func (d *DecoderV4) decodeWord16(v interface{}) (Word16Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown Word16 encoding: %T", v)
	}
	const max = math.MaxUint16
	if value > max {
		return 0, fmt.Errorf("invalid Word16: got %d, expected max %d", v, max)
	}
	return Word16Value(value), nil
}

func (d *DecoderV4) decodeWord32(v interface{}) (Word32Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown Word32 encoding: %T", v)
	}
	const max = math.MaxUint32
	if value > max {
		return 0, fmt.Errorf("invalid Word32: got %d, expected max %d", v, max)
	}
	return Word32Value(value), nil
}

func (d *DecoderV4) decodeWord64(v interface{}) (Word64Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown Word64 encoding: %T", v)
	}
	return Word64Value(value), nil
}

func (d *DecoderV4) decodeFix64(v interface{}) (Fix64Value, error) {
	switch v := v.(type) {
	case uint64:
		const max = math.MaxInt64
		if v > max {
			return 0, fmt.Errorf("invalid Fix64: got %d, expected max %d", v, max)
		}
		return Fix64Value(v), nil

	case int64:
		return Fix64Value(v), nil

	default:
		return 0, fmt.Errorf("unknown Fix64 encoding: %T", v)
	}
}

func (d *DecoderV4) decodeUFix64(v interface{}) (UFix64Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown UFix64 encoding: %T", v)
	}
	return UFix64Value(value), nil
}

func (d *DecoderV4) decodeSome(v interface{}, path []string) (*SomeValue, error) {
	value, err := d.decodeValue(v, path)
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

func (d *DecoderV4) decodeAddress(v interface{}) (AddressValue, error) {
	addressBytes, ok := v.([]byte)
	if !ok {
		return AddressValue{}, fmt.Errorf("invalid address encoding: %T", v)
	}

	err := d.checkAddressLength(addressBytes)
	if err != nil {
		return AddressValue{}, err
	}

	address := NewAddressValueFromBytes(addressBytes)
	return address, nil
}

func (d *DecoderV4) decodePath(v interface{}) (PathValue, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedPathValueLength
	if !ok || len(encoded) != expectedLength {
		return PathValue{}, fmt.Errorf("invalid path encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	field1 := encoded[encodedPathValueDomainFieldKey]
	domain, ok := field1.(uint64)
	if !ok {
		return PathValue{}, fmt.Errorf("invalid path domain encoding: %T", field1)
	}

	field2 := encoded[encodedPathValueIdentifierFieldKey]
	identifier, ok := field2.(string)
	if !ok {
		return PathValue{}, fmt.Errorf("invalid path identifier encoding: %T", field2)
	}

	return PathValue{
		Domain:     common.PathDomain(domain),
		Identifier: identifier,
	}, nil
}

func (d *DecoderV4) decodeCapability(v interface{}) (CapabilityValue, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedCapabilityValueLength
	if !ok || len(encoded) != expectedLength {
		return CapabilityValue{}, fmt.Errorf("invalid path encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	// address

	field1 := encoded[encodedCapabilityValueAddressFieldKey]
	field1Value, err := d.decodeValue(field1, nil)
	if err != nil {
		return CapabilityValue{}, fmt.Errorf("invalid capability address: %w", err)
	}

	address, ok := field1Value.(AddressValue)
	if !ok {
		return CapabilityValue{}, fmt.Errorf("invalid capability address: %T", address)
	}

	// path

	field2 := encoded[encodedCapabilityValuePathFieldKey]
	field2Value, err := d.decodeValue(field2, nil)
	if err != nil {
		return CapabilityValue{}, fmt.Errorf("invalid capability path: %w", err)
	}

	path, ok := field2Value.(PathValue)
	if !ok {
		return CapabilityValue{}, fmt.Errorf("invalid capability path: %T", path)
	}

	// borrow type (optional, for backwards compatibility)
	// Capabilities used to be untyped, i.e. they didn't have a borrow type.
	// Later an optional type paramater, the borrow type, was added to it,
	// which specifies as what type the capability should be borrowed.
	//
	// The decoding must be backwards-compatible and support both capability values
	// with a borrow type and ones without
	var borrowType StaticType

	if field3 := encoded[encodedCapabilityValueBorrowTypeFieldKey]; field3 != nil {

		borrowType, err = d.decodeStaticType(field3)
		if err != nil {
			return CapabilityValue{}, fmt.Errorf("invalid capability borrow type encoding: %w", err)
		}
	}

	return CapabilityValue{
		Address:    address,
		Path:       path,
		BorrowType: borrowType,
	}, nil
}

func (d *DecoderV4) decodeLink(v interface{}) (LinkValue, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedLinkValueLength
	if !ok || len(encoded) != expectedLength {
		return LinkValue{}, fmt.Errorf("invalid link encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	decodedPath, err := d.decodeValue(encoded[encodedLinkValueTargetPathFieldKey], nil)
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link target path encoding: %w", err)
	}

	pathValue, ok := decodedPath.(PathValue)
	if !ok {
		return LinkValue{}, fmt.Errorf("invalid link target path encoding: %T", decodedPath)
	}

	staticType, err := d.decodeStaticType(encoded[encodedLinkValueTypeFieldKey])
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link type encoding: %w", err)
	}

	return LinkValue{
		TargetPath: pathValue,
		Type:       staticType,
	}, nil
}

func (d *DecoderV4) decodeStaticType(v interface{}) (StaticType, error) {
	tag, ok := v.(cbor.Tag)
	if !ok {
		return nil, fmt.Errorf("invalid static type encoding: %T", v)
	}

	content := tag.Content

	switch tag.Number {
	case cborTagPrimitiveStaticType:
		return d.decodePrimitiveStaticType(content)

	case cborTagOptionalStaticType:
		return d.decodeOptionalStaticType(content)

	case cborTagCompositeStaticType:
		return d.decodeCompositeStaticType(content)

	case cborTagInterfaceStaticType:
		return d.decodeInterfaceStaticType(content)

	case cborTagVariableSizedStaticType:
		return d.decodeVariableSizedStaticType(content)

	case cborTagConstantSizedStaticType:
		return d.decodeConstantSizedStaticType(content)

	case cborTagReferenceStaticType:
		return d.decodeReferenceStaticType(content)

	case cborTagDictionaryStaticType:
		return d.decodeDictionaryStaticType(content)

	case cborTagRestrictedStaticType:
		return d.decodeRestrictedStaticType(content)

	case cborTagCapabilityStaticType:
		return d.decodeCapabilityStaticType(content)

	default:
		return nil, fmt.Errorf("invalid static type encoding tag: %d", tag.Number)
	}
}

func (d *DecoderV4) decodePrimitiveStaticType(v interface{}) (PrimitiveStaticType, error) {
	encoded, ok := v.(uint64)
	if !ok {
		return PrimitiveStaticTypeUnknown,
			fmt.Errorf("invalid primitive static type encoding: %T", v)
	}
	return PrimitiveStaticType(encoded), nil
}

func (d *DecoderV4) decodeOptionalStaticType(v interface{}) (StaticType, error) {
	staticType, err := d.decodeStaticType(v)
	if err != nil {
		return nil, fmt.Errorf("invalid optional static type inner type encoding: %w", err)
	}
	return OptionalStaticType{
		Type: staticType,
	}, nil
}

func (d *DecoderV4) decodeStaticTypeLocationAndQualifiedIdentifier(
	encoded cborArray,
	locationKeyIndex uint64,
	qualifiedIdentifierIndex uint64,
) (
	common.Location,
	string,
	error,
) {
	location, err := d.decodeLocation(encoded[locationKeyIndex])
	if err != nil {
		return nil, "", fmt.Errorf("invalid static type location encoding: %w", err)
	}

	qualifiedIdentifierField := encoded[qualifiedIdentifierIndex]
	qualifiedIdentifier, ok := qualifiedIdentifierField.(string)
	if !ok {
		return nil, "", fmt.Errorf(
			"invalid static type qualified identifier encoding: %T",
			qualifiedIdentifierField,
		)
	}

	return location, qualifiedIdentifier, nil
}

func (d *DecoderV4) decodeCompositeStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedCompositeStaticTypeLength
	if !ok || len(encoded) != expectedLength {
		return nil, fmt.Errorf("invalid composite static type encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	location, qualifiedIdentifier, err := d.decodeStaticTypeLocationAndQualifiedIdentifier(
		encoded,
		encodedCompositeStaticTypeLocationFieldKey,
		encodedCompositeStaticTypeQualifiedIdentifierFieldKey,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid composite static type encoding: %w", err)
	}

	return CompositeStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
	}, nil
}

func (d *DecoderV4) decodeInterfaceStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedInterfaceStaticTypeLength
	if !ok || len(encoded) != expectedLength {
		return nil, fmt.Errorf("invalid interface static type encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	location, qualifiedIdentifier, err := d.decodeStaticTypeLocationAndQualifiedIdentifier(
		encoded,
		encodedInterfaceStaticTypeLocationFieldKey,
		encodedInterfaceStaticTypeQualifiedIdentifierFieldKey,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid interface static type encoding: %w", err)
	}

	return InterfaceStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
	}, nil
}

func (d *DecoderV4) decodeVariableSizedStaticType(v interface{}) (StaticType, error) {
	staticType, err := d.decodeStaticType(v)
	if err != nil {
		return nil, fmt.Errorf("invalid optional static type encoding: %w", err)
	}
	return VariableSizedStaticType{
		Type: staticType,
	}, nil
}

func (d *DecoderV4) decodeConstantSizedStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedConstantSizedStaticTypeLength
	if !ok || len(encoded) != expectedLength {
		return nil, fmt.Errorf("invalid constant-sized static type encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	field1 := encoded[encodedConstantSizedStaticTypeSizeFieldKey]
	size, ok := field1.(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid constant-sized static type size encoding: %T", field1)
	}

	const max = math.MaxInt64
	if size > max {
		return nil, fmt.Errorf(
			"invalid constant-sized static type size: got %d, expected max %d",
			size,
			max,
		)
	}

	staticType, err := d.decodeStaticType(encoded[encodedConstantSizedStaticTypeTypeFieldKey])
	if err != nil {
		return nil, fmt.Errorf("invalid constant-sized static type inner type encoding: %w", err)
	}

	return ConstantSizedStaticType{
		Type: staticType,
		Size: int64(size),
	}, nil
}

func (d *DecoderV4) decodeReferenceStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedReferenceStaticTypeLength
	if !ok || len(encoded) != expectedLength {
		return nil, fmt.Errorf("invalid reference static type encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	field1 := encoded[encodedReferenceStaticTypeAuthorizedFieldKey]
	authorized, ok := field1.(bool)
	if !ok {
		return nil, fmt.Errorf("invalid reference static type authorized encoding: %T", field1)
	}

	staticType, err := d.decodeStaticType(encoded[encodedReferenceStaticTypeTypeFieldKey])
	if err != nil {
		return nil, fmt.Errorf("invalid reference static type inner type encoding: %w", err)
	}

	return ReferenceStaticType{
		Authorized: authorized,
		Type:       staticType,
	}, nil
}

func (d *DecoderV4) decodeDictionaryStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedDictionaryStaticTypeLength
	if !ok || len(encoded) != expectedLength {
		return nil, fmt.Errorf("invalid dictionary static type encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	keyType, err := d.decodeStaticType(encoded[encodedDictionaryStaticTypeKeyTypeFieldKey])
	if err != nil {
		return nil, fmt.Errorf("invalid dictionary static type key type encoding: %w", err)
	}

	valueType, err := d.decodeStaticType(encoded[encodedDictionaryStaticTypeValueTypeFieldKey])
	if err != nil {
		return nil, fmt.Errorf("invalid dictionary static type value type encoding: %w", err)
	}

	return DictionaryStaticType{
		KeyType:   keyType,
		ValueType: valueType,
	}, nil
}

func (d *DecoderV4) decodeRestrictedStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedRestrictedStaticTypeLength
	if !ok || len(encoded) != expectedLength {
		return nil, fmt.Errorf("invalid restricted static type encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	restrictedType, err := d.decodeStaticType(encoded[encodedRestrictedStaticTypeTypeFieldKey])
	if err != nil {
		return nil, fmt.Errorf("invalid restricted static type key type encoding: %w", err)
	}

	field2 := encoded[encodedRestrictedStaticTypeRestrictionsFieldKey]
	encodedRestrictions, ok := field2.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid restricted static type restrictions encoding: %T", field2)
	}

	restrictions := make([]InterfaceStaticType, len(encodedRestrictions))
	for i, encodedRestriction := range encodedRestrictions {
		r, err := d.decodeStaticType(encodedRestriction)
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

func (d *DecoderV4) decodeType(v interface{}) (TypeValue, error) {
	encoded, ok := v.(cborArray)
	const expectedLength = encodedTypeValueTypeLength
	if !ok || len(encoded) != expectedLength {
		return TypeValue{}, fmt.Errorf("invalid type encoding: expected [%d]interface{}, got %T",
			expectedLength,
			v,
		)
	}

	var staticType StaticType

	staticTypeField := encoded[encodedTypeValueTypeFieldKey]
	if staticTypeField != nil {
		var err error
		staticType, err = d.decodeStaticType(staticTypeField)
		if err != nil {
			return TypeValue{}, fmt.Errorf("invalid type encoding: %w", err)
		}
	}

	return TypeValue{
		Type: staticType,
	}, nil
}

func (d *DecoderV4) decodeCapabilityStaticType(v interface{}) (StaticType, error) {
	var borrowStaticType StaticType
	if v != nil {
		var err error
		borrowStaticType, err = d.decodeStaticType(v)
		if err != nil {
			return nil, fmt.Errorf("invalid capability static type borrow type encoding: %w", err)
		}
	}
	return CapabilityStaticType{
		BorrowType: borrowStaticType,
	}, nil
}
