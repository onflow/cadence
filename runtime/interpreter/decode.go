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
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/sema"
)

type DecodingCallback func(value interface{}, path []string)

// A Decoder decodes CBOR-encoded representations of values.
//
type Decoder struct {
	decoder        *cbor.Decoder
	owner          *common.Address
	version        uint16
	decodeCallback DecodingCallback
}

// Decode returns a value decoded from its CBOR-encoded representation,
// for the given owner (can be `nil`).
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

// NewDecoder initializes a Decoder that will decode CBOR-encoded bytes from the
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
	*Decoder,
	error,
) {
	return &Decoder{
		decoder:        decMode.NewDecoder(reader),
		owner:          owner,
		version:        version,
		decodeCallback: decodeCallback,
	}, nil
}

var decMode = func() cbor.DecMode {
	decMode, err := cbor.DecOptions{
		IntDec:           cbor.IntDecConvertNone,
		MaxArrayElements: 512 * 1024,
		MaxMapPairs:      512 * 1024,
		MaxNestedLevels:  256,
	}.DecMode()
	if err != nil {
		panic(err)
	}
	return decMode
}()

// Decode reads CBOR-encoded bytes from the io.Reader and decodes them to a value.
//
func (d *Decoder) Decode(path []string) (Value, error) {
	var v interface{}
	err := d.decoder.Decode(&v)
	if err != nil {
		return nil, err
	}

	return d.decodeValue(v, path)
}

func (d *Decoder) decodeValue(v interface{}, path []string) (Value, error) {

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

		case cborTagStorageReferenceValue:
			return d.decodeStorageReference(v.Content)

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

func (d *Decoder) decodeString(v string) Value {
	value := NewStringValue(v)
	value.modified = false
	return value
}

func (d *Decoder) decodeArray(v []interface{}, path []string) (*ArrayValue, error) {
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

func (d *Decoder) decodeDictionary(v interface{}, path []string) (*DictionaryValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf(
			"invalid dictionary encoding (@ %s): %T",
			strings.Join(path, "."),
			v,
		)
	}

	keysField := encoded[encodedDictionaryValueKeysFieldKey]
	encodedKeys, ok := keysField.([]interface{})
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
	encodedEntries, ok := entriesField.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf(
			"invalid dictionary entries encoding (@ %s): %T",
			strings.Join(path, "."),
			entriesField,
		)
	}

	keyCount := keys.Count()
	entryCount := len(encodedEntries)

	// In versions <= 2, the dictionary key string function
	// was accidentally, temporarily changed without a version change.
	//
	// The key string format for address values is:
	// prefix the address with 0x, encode in hex, and strip leading zeros.
	//
	// Temporarily and accidentally the format was:
	// no 0x prefix, and encode and in full length hex.
	//
	// Detect this temporary format and correct it

	var hasAddressValueKeyInWrongPre3Format bool

	if d.version <= 2 {
		for _, keyValue := range keys.Values {
			keyAddressValue, ok := keyValue.(AddressValue)
			if !ok {
				continue
			}

			currentKeyString := keyAddressValue.KeyString()
			wrongKeyString := hex.EncodeToString(keyAddressValue[:])

			// Is there a value stored with the current format?
			// Then no migration is necessary.

			if encodedEntries[currentKeyString] != nil {
				continue
			}

			// Is there at least a value stored in the wrong key string format?

			if encodedEntries[wrongKeyString] == nil {

				return nil, fmt.Errorf(
					"invalid dictionary address value key: "+
						"could neither find entry for wrong format key %s, nor for current format key %s",
					wrongKeyString,
					currentKeyString,
				)
			}

			// Migrate the value from the wrong format to the current format

			hasAddressValueKeyInWrongPre3Format = true

			encodedEntries[currentKeyString] = encodedEntries[wrongKeyString]
			delete(encodedEntries, wrongKeyString)
		}
	}

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

		if hasAddressValueKeyInWrongPre3Format {
			return nil, fmt.Errorf(
				"invalid dictionary (@ %s): dictionary with address values in pre-3 format and deferred values",
				strings.Join(path, "."),
			)
		}

		deferred = orderedmap.NewStringStructOrderedMap()
		deferredOwner = d.owner
		deferredStorageKeyBase = joinPath(append(path[:], dictionaryValuePathPrefix))
		for _, keyValue := range keys.Values {
			key := DictionaryKey(keyValue)
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
			value, ok := encodedEntries[keyString]
			if !ok {
				return nil, fmt.Errorf(
					"missing dictionary value for key (@ %s, %d): %s",
					strings.Join(path, "."),
					index,
					keyString,
				)
			}

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

func (d *Decoder) decodeLocation(l interface{}) (common.Location, error) {
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

func (d *Decoder) decodeStringLocation(v interface{}) (common.Location, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("invalid string location encoding: %T", v)
	}
	return common.StringLocation(s), nil
}

func (d *Decoder) decodeIdentifierLocation(v interface{}) (common.Location, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("invalid identifier location encoding: %T", v)
	}
	return common.IdentifierLocation(s), nil
}

func (d *Decoder) decodeAddressLocation(v interface{}) (common.Location, error) {

	// If the encoded location is just a byte slice,
	// it is the address and no name is provided

	encodedAddress, ok := v.([]byte)
	if ok {
		err := d.checkAddressLength(encodedAddress)
		if err != nil {
			return nil, err
		}

		return common.AddressLocation{
			Address: common.BytesToAddress(encodedAddress),
		}, nil
	}

	// Otherwise, the encoded location is expected to be a map,
	// which includes both address and name

	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid address location encoding: %T", v)
	}

	// Address

	field1 := encoded[encodedAddressLocationAddressFieldKey]
	encodedAddress, ok = field1.([]byte)
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

func (d *Decoder) decodeComposite(v interface{}, path []string) (*CompositeValue, error) {

	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf(
			"invalid composite encoding (@ %s): %T",
			strings.Join(path, "."),
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

	// Qualified identifier or Type ID.
	//
	// An earlier version of the format stored the whole type ID.
	// However, the composite already stores the location,
	// so the current version of the format only stores the qualified identifier.

	var qualifiedIdentifier string

	qualifiedIdentifierField := encoded[encodedCompositeValueQualifiedIdentifierFieldKey]
	if qualifiedIdentifierField != nil {
		qualifiedIdentifier, ok = qualifiedIdentifierField.(string)
		if !ok {
			return nil, fmt.Errorf(
				"invalid composite qualified identifier encoding (@ %s): %T",
				strings.Join(path, "."),
				qualifiedIdentifierField,
			)
		}
	} else {
		typeIDField := encoded[encodedCompositeValueTypeIDFieldKey]
		if typeIDField != nil {

			encodedTypeID, ok := typeIDField.(string)
			if !ok {
				return nil, fmt.Errorf(
					"invalid composite type ID encoding (@ %s): %T",
					strings.Join(path, "."),
					typeIDField,
				)
			}

			_, qualifiedIdentifier, err = common.DecodeTypeID(encodedTypeID)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid composite type ID (@ %s): %w",
					strings.Join(path, "."),
					err,
				)
			}

			// Special case: The decoded location might be an address location which has no name

			location = d.inferAddressLocationName(location, qualifiedIdentifier)
		} else {
			return nil, fmt.Errorf(
				"missing composite qualified identifier or type ID (@ %s)",
				strings.Join(path, "."),
			)
		}
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
	encodedFields, ok := fieldsField.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf(
			"invalid composite fields encoding (@ %s): %T",
			strings.Join(path, "."),
			fieldsField,
		)
	}

	// Gather all field names and sort them lexicographically

	var fieldNames []string

	index := 0

	for fieldName := range encodedFields { //nolint:maprangecheck
		nameString, ok := fieldName.(string)
		if !ok {
			return nil, fmt.Errorf(
				"invalid composite field name encoding (@ %s, %d): %T",
				strings.Join(path, "."),
				index,
				fieldName,
			)
		}

		fieldNames = append(fieldNames, nameString)

		index++
	}

	// Decode all fields in lexicographic order

	sort.Strings(fieldNames)

	fields := NewStringValueOrderedMap()

	for _, fieldName := range fieldNames {

		value := encodedFields[fieldName]

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

func (d *Decoder) decodeBig(v interface{}) (*big.Int, error) {
	bigInt, ok := v.(big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid bignum encoding: %T", v)
	}

	// The encoding of negative bignums is specified in
	// https://tools.ietf.org/html/rfc7049#section-2.4.2:
	// "For tag value 3, the value of the bignum is -1 - n."
	//
	// Negative bignums were encoded incorrectly in version < 2,
	// as just -n.
	//
	// Fix this by adjusting by one.

	if bigInt.Sign() < 0 && d.version < 2 {
		bigInt.Add(&bigInt, bigOne)
	}

	return &bigInt, nil
}

func (d *Decoder) decodeInt(v interface{}) (IntValue, error) {
	bigInt, err := d.decodeBig(v)
	if err != nil {
		return IntValue{}, fmt.Errorf("invalid Int encoding: %w", err)
	}

	return NewIntValueFromBigInt(bigInt), nil
}

func (d *Decoder) decodeInt8(v interface{}) (Int8Value, error) {
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

func (d *Decoder) decodeInt16(v interface{}) (Int16Value, error) {
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

func (d *Decoder) decodeInt32(v interface{}) (Int32Value, error) {
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

func (d *Decoder) decodeInt64(v interface{}) (Int64Value, error) {
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

func (d *Decoder) decodeInt128(v interface{}) (Int128Value, error) {
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

func (d *Decoder) decodeInt256(v interface{}) (Int256Value, error) {
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

func (d *Decoder) decodeUInt(v interface{}) (UIntValue, error) {
	bigInt, err := d.decodeBig(v)
	if err != nil {
		return UIntValue{}, fmt.Errorf("invalid UInt encoding: %w", err)
	}

	if bigInt.Sign() < 0 {
		return UIntValue{}, fmt.Errorf("invalid UInt: got %s, expected positive", bigInt)
	}

	return NewUIntValueFromBigInt(bigInt), nil
}

func (d *Decoder) decodeUInt8(v interface{}) (UInt8Value, error) {
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

func (d *Decoder) decodeUInt16(v interface{}) (UInt16Value, error) {
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

func (d *Decoder) decodeUInt32(v interface{}) (UInt32Value, error) {
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

func (d *Decoder) decodeUInt64(v interface{}) (UInt64Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown UInt64 encoding: %T", v)
	}
	return UInt64Value(value), nil
}

func (d *Decoder) decodeUInt128(v interface{}) (UInt128Value, error) {
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

func (d *Decoder) decodeUInt256(v interface{}) (UInt256Value, error) {
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

func (d *Decoder) decodeWord8(v interface{}) (Word8Value, error) {
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

func (d *Decoder) decodeWord16(v interface{}) (Word16Value, error) {
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

func (d *Decoder) decodeWord32(v interface{}) (Word32Value, error) {
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

func (d *Decoder) decodeWord64(v interface{}) (Word64Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown Word64 encoding: %T", v)
	}
	return Word64Value(value), nil
}

func (d *Decoder) decodeFix64(v interface{}) (Fix64Value, error) {
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

func (d *Decoder) decodeUFix64(v interface{}) (UFix64Value, error) {
	value, ok := v.(uint64)
	if !ok {
		return 0, fmt.Errorf("unknown UFix64 encoding: %T", v)
	}
	return UFix64Value(value), nil
}

func (d *Decoder) decodeSome(v interface{}, path []string) (*SomeValue, error) {
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

func (d *Decoder) decodeStorageReference(v interface{}) (*StorageReferenceValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid storage reference encoding: %T", v)
	}

	authorized, ok := encoded[encodedStorageReferenceValueAuthorizedFieldKey].(bool)
	if !ok {
		return nil, fmt.Errorf("invalid storage reference authorized encoding: %T", authorized)
	}

	targetStorageAddressBytes, ok := encoded[encodedStorageReferenceValueTargetStorageAddressFieldKey].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid storage reference target storage address encoding: %T", authorized)
	}

	targetStorageAddress := common.BytesToAddress(targetStorageAddressBytes)

	targetKey, ok := encoded[encodedStorageReferenceValueTargetKeyFieldKey].(string)
	if !ok {
		return nil, fmt.Errorf("invalid storage reference target key encoding: %T", targetKey)
	}

	return &StorageReferenceValue{
		Authorized:           authorized,
		TargetStorageAddress: targetStorageAddress,
		TargetKey:            targetKey,
	}, nil
}

func (d *Decoder) checkAddressLength(addressBytes []byte) error {
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

func (d *Decoder) decodeAddress(v interface{}) (AddressValue, error) {
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

func (d *Decoder) decodePath(v interface{}) (PathValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return PathValue{}, fmt.Errorf("invalid path encoding: %T", v)
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

func (d *Decoder) decodeCapability(v interface{}) (CapabilityValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return CapabilityValue{}, fmt.Errorf("invalid capability encoding: %T", v)
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

	var borrowType StaticType

	if field3, ok := encoded[encodedCapabilityValueBorrowTypeFieldKey]; ok && field3 != nil {

		decodedStaticType, err := d.decodeStaticType(field3)
		if err != nil {
			return CapabilityValue{}, fmt.Errorf("invalid capability borrow type encoding: %w", err)
		}

		borrowType, ok = decodedStaticType.(StaticType)
		if !ok {
			return CapabilityValue{}, fmt.Errorf("invalid capability borrow encoding: %T", decodedStaticType)
		}
	}

	return CapabilityValue{
		Address:    address,
		Path:       path,
		BorrowType: borrowType,
	}, nil
}

func (d *Decoder) decodeLink(v interface{}) (LinkValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return LinkValue{}, fmt.Errorf("invalid link encoding")
	}

	decodedPath, err := d.decodeValue(encoded[encodedLinkValueTargetPathFieldKey], nil)
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link target path encoding: %w", err)
	}

	pathValue, ok := decodedPath.(PathValue)
	if !ok {
		return LinkValue{}, fmt.Errorf("invalid link target path encoding: %T", decodedPath)
	}

	decodedStaticType, err := d.decodeStaticType(encoded[encodedLinkValueTypeFieldKey])
	if err != nil {
		return LinkValue{}, fmt.Errorf("invalid link type encoding: %w", err)
	}

	staticType, ok := decodedStaticType.(StaticType)
	if !ok {
		return LinkValue{}, fmt.Errorf("invalid link type encoding: %T", decodedStaticType)
	}

	return LinkValue{
		TargetPath: pathValue,
		Type:       staticType,
	}, nil
}

func (d *Decoder) decodeStaticType(v interface{}) (StaticType, error) {
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

func (d *Decoder) decodePrimitiveStaticType(v interface{}) (PrimitiveStaticType, error) {
	encoded, ok := v.(uint64)
	if !ok {
		return PrimitiveStaticTypeUnknown,
			fmt.Errorf("invalid primitive static type encoding: %T", v)
	}
	return PrimitiveStaticType(encoded), nil
}

func (d *Decoder) decodeOptionalStaticType(v interface{}) (StaticType, error) {
	staticType, err := d.decodeStaticType(v)
	if err != nil {
		return nil, fmt.Errorf("invalid optional static type inner type encoding: %w", err)
	}
	return OptionalStaticType{
		Type: staticType,
	}, nil
}

func (d *Decoder) decodeStaticTypeLocationAndQualifiedIdentifier(
	v interface{},
	locationKeyIndex uint64,
	typeIDKeyIndex uint64,
	qualifiedIdentifierIndex uint64,
) (
	common.Location,
	string,
	error,
) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid static type encoding: %T", v)
	}

	location, err := d.decodeLocation(encoded[locationKeyIndex])
	if err != nil {
		return nil, "", fmt.Errorf("invalid static type location encoding: %w", err)
	}

	var qualifiedIdentifier string

	qualifiedIdentifierField := encoded[qualifiedIdentifierIndex]
	if qualifiedIdentifierField != nil {
		qualifiedIdentifier, ok = qualifiedIdentifierField.(string)
		if !ok {
			return nil, "", fmt.Errorf(
				"invalid static type qualified identifier encoding: %T",
				qualifiedIdentifierField,
			)
		}
	} else {

		typeIDField := encoded[typeIDKeyIndex]

		if typeIDField != nil {

			encodedTypeID, ok := typeIDField.(string)
			if !ok {
				return nil, "", fmt.Errorf("invalid static type type ID encoding: %T", typeIDField)
			}
			typeID := sema.TypeID(encodedTypeID)

			// Special case: The decoded location might be an address location which has no name

			qualifiedIdentifier = location.QualifiedIdentifier(typeID)
			location = d.inferAddressLocationName(location, qualifiedIdentifier)
		} else {
			return nil, "", errors.New("missing static type qualified identifier or type ID")
		}
	}

	return location, qualifiedIdentifier, nil
}

// inferAddressLocationName infers the name for an address location from a qualified identifier.
//
// In the first version of the storage format, accounts could only store one contract
// instead of several contracts (separated by name), so composite's locations were
// address locations without a name, i.e. just the bare address.
//
// An update added support for multiple contracts per account, which added names to address locations:
// Each contract of an account is stored in a distinct location.
//
// So to keep backwards-compatibility:
// If the location is an address location without a name,
// then infer the name from the qualified identifier.
//
func (d *Decoder) inferAddressLocationName(location common.Location, qualifiedIdentifier string) common.Location {

	// Only consider address locations which have no name

	addressLocation, ok := location.(common.AddressLocation)
	if !ok || addressLocation.Name != "" {
		return location
	}

	// The first component of the type ID is the location name

	parts := strings.SplitN(qualifiedIdentifier, ".", 2)

	return common.AddressLocation{
		Address: addressLocation.Address,
		Name:    parts[0],
	}
}

func (d *Decoder) decodeCompositeStaticType(v interface{}) (StaticType, error) {
	location, qualifiedIdentifier, err := d.decodeStaticTypeLocationAndQualifiedIdentifier(
		v,
		encodedCompositeStaticTypeLocationFieldKey,
		encodedCompositeStaticTypeTypeIDFieldKey,
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

func (d *Decoder) decodeInterfaceStaticType(v interface{}) (StaticType, error) {
	location, qualifiedIdentifier, err := d.decodeStaticTypeLocationAndQualifiedIdentifier(
		v,
		encodedInterfaceStaticTypeLocationFieldKey,
		encodedInterfaceStaticTypeTypeIDFieldKey,
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

func (d *Decoder) decodeVariableSizedStaticType(v interface{}) (StaticType, error) {
	staticType, err := d.decodeStaticType(v)
	if err != nil {
		return nil, fmt.Errorf("invalid optional static type encoding: %w", err)
	}
	return VariableSizedStaticType{
		Type: staticType,
	}, nil
}

func (d *Decoder) decodeConstantSizedStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid constant-sized static type encoding: %T", v)
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

func (d *Decoder) decodeReferenceStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid reference static type encoding: %T", v)
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

func (d *Decoder) decodeDictionaryStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dictionary static type encoding: %T", v)
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

func (d *Decoder) decodeRestrictedStaticType(v interface{}) (StaticType, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid restricted static type encoding: %T", v)
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

func (d *Decoder) decodeType(v interface{}) (TypeValue, error) {

	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return TypeValue{}, fmt.Errorf("invalid type encoding")
	}

	var staticType StaticType

	staticTypeField, ok := encoded[encodedTypeValueTypeFieldKey]
	if ok {
		decodedStaticType, err := d.decodeStaticType(staticTypeField)
		if err != nil {
			return TypeValue{}, fmt.Errorf("invalid type encoding: %w", err)
		}

		staticType, ok = decodedStaticType.(StaticType)
		if !ok {
			return TypeValue{}, fmt.Errorf("invalid type encoding: %T", decodedStaticType)
		}
	}

	return TypeValue{
		Type: staticType,
	}, nil
}

func (d *Decoder) decodeCapabilityStaticType(v interface{}) (StaticType, error) {
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
