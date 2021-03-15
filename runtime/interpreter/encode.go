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
	"math/big"
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/cadence/runtime/common"
)

type cborMap = map[uint64]interface{}

// Cadence needs to encode different kinds of objects in CBOR, for instance,
// dictionaries, structs, resources, etc.
//
// However, CBOR only provides one native map type, and no support
// for directly representing e.g. structs or resources.
//
// To be able to encode/decode such semantically different values,
// we define custom CBOR tags.

// !!! *WARNING* !!!
//
// Only add new fields to encoded structs by
// appending new fields with the next highest key.
//
// DO *NOT* REPLACE EXISTING FIELDS!

const cborTagBase = 128

// !!! *WARNING* !!!
//
// Only add new types by:
// - replacing existing placeholders (`_`) with new types
// - appending new types
//
// Only remove types by:
// - replace existing types with a placeholder `_`
//
// DO *NOT* REPLACE EXISTING TYPES!
// DO *NOT* ADD NEW TYPES IN BETWEEN!

const (
	cborTagVoidValue = cborTagBase + iota
	cborTagDictionaryValue
	cborTagSomeValue
	cborTagAddressValue
	cborTagCompositeValue
	cborTagTypeValue
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_

	// Int*
	cborTagIntValue
	cborTagInt8Value
	cborTagInt16Value
	cborTagInt32Value
	cborTagInt64Value
	cborTagInt128Value
	cborTagInt256Value
	_

	// UInt*
	cborTagUIntValue
	cborTagUInt8Value
	cborTagUInt16Value
	cborTagUInt32Value
	cborTagUInt64Value
	cborTagUInt128Value
	cborTagUInt256Value
	_

	// Word*
	_
	cborTagWord8Value
	cborTagWord16Value
	cborTagWord32Value
	cborTagWord64Value
	_ // future: Word128
	_ // future: Word256
	_

	// Fix*
	_
	_ // future: Fix8
	_ // future: Fix16
	_ // future: Fix32
	cborTagFix64Value
	_ // future: Fix128
	_ // future: Fix256
	_

	// UFix*
	_
	_ // future: UFix8
	_ // future: UFix16
	_ // future: UFix32
	cborTagUFix64Value
	_ // future: UFix128
	_ // future: UFix256
	_

	// Locations
	cborTagAddressLocation
	cborTagStringLocation
	cborTagIdentifierLocation
	_
	_
	_
	_
	_

	// Storage

	cborTagPathValue
	cborTagCapabilityValue
	cborTagStorageReferenceValue
	cborTagLinkValue
	_
	_
	_
	_
	_
	_
	_
	_

	// Static Types
	cborTagPrimitiveStaticType
	cborTagCompositeStaticType
	cborTagInterfaceStaticType
	cborTagVariableSizedStaticType
	cborTagConstantSizedStaticType
	cborTagDictionaryStaticType
	cborTagOptionalStaticType
	cborTagReferenceStaticType
	cborTagRestrictedStaticType
	cborTagCapabilityStaticType
)

type EncodingDeferralMove struct {
	DeferredOwner      common.Address
	DeferredStorageKey string
	NewOwner           common.Address
	NewStorageKey      string
}

type EncodingDeferralValue struct {
	Key   string
	Value Value
}

type EncodingDeferrals struct {
	Values []EncodingDeferralValue
	Moves  []EncodingDeferralMove
}

type EncodingPrepareCallback func(value Value, path []string)

// Encoder converts Values into CBOR-encoded bytes.
//
type Encoder struct {
	enc             *cbor.Encoder
	deferred        bool
	prepareCallback EncodingPrepareCallback
}

// EncodeValue returns the CBOR-encoded representation of the given value.
//
// The given path is used to identify values in the object graph.
// For example, path elements are appended for array elements (the index),
// dictionary values (the key), and composites (the field name).
//
// The deferred flag determines if child values should be deferred,
// i.e. should not be encoded into the result,
// but e.g. be eventually written to separate storage keys.
// If true, the deferrals result will contain the values
// which have not been encoded, and which values need to be moved
// from a previous storage key to another storage key.
//
func EncodeValue(value Value, path []string, deferred bool, prepareCallback EncodingPrepareCallback) (
	encoded []byte,
	deferrals *EncodingDeferrals,
	err error,
) {
	var w bytes.Buffer
	enc, err := NewEncoder(&w, deferred, prepareCallback)
	if err != nil {
		return nil, nil, err
	}

	deferrals = &EncodingDeferrals{}

	err = enc.Encode(value, path, deferrals)
	if err != nil {
		return nil, nil, err
	}

	data := w.Bytes()
	err = decMode.Valid(data)
	if err != nil {
		return nil, nil, fmt.Errorf("encoder produced invalid data: %w", err)
	}

	return data, deferrals, nil
}

// See https://github.com/fxamacker/cbor:
// "For best performance, reuse EncMode and DecMode after creating them."
//
var encMode = func() cbor.EncMode {
	options := cbor.CanonicalEncOptions()
	options.BigIntConvert = cbor.BigIntConvertNone
	encMode, err := options.EncMode()
	if err != nil {
		panic(err)
	}
	return encMode
}()

// NewEncoder initializes an Encoder that will write CBOR-encoded bytes
// to the given io.Writer.
//
func NewEncoder(w io.Writer, deferred bool, prepareCallback EncodingPrepareCallback) (*Encoder, error) {
	enc := encMode.NewEncoder(w)
	return &Encoder{
		enc:             enc,
		deferred:        deferred,
		prepareCallback: prepareCallback,
	}, nil
}

// Encode writes the CBOR-encoded representation of the given value to this
// encoder's io.Writer.
//
// This function returns an error if the given value's type is not supported
// by this encoder.
//
func (e *Encoder) Encode(
	v Value,
	path []string,
	deferrals *EncodingDeferrals,
) error {
	prepared, err := e.prepare(v, path, deferrals)
	if err != nil {
		return err
	}

	return e.enc.Encode(prepared)
}

// prepare traverses the object graph of the provided value and returns
// the representation for the value that can be marshalled to CBOR.
//
func (e *Encoder) prepare(
	v Value,
	path []string,
	deferrals *EncodingDeferrals,
) (
	interface{},
	error,
) {
	if e.prepareCallback != nil {
		e.prepareCallback(v, path)
	}

	switch v := v.(type) {

	case NilValue:
		return e.prepareNil(), nil

	case VoidValue:
		return e.prepareVoid(), nil

	case BoolValue:
		return e.prepareBool(v), nil

	case AddressValue:
		return e.prepareAddressValue(v), nil

	// Int*

	case IntValue:
		return e.prepareInt(v), nil

	case Int8Value:
		return e.prepareInt8(v), nil

	case Int16Value:
		return e.prepareInt16(v), nil

	case Int32Value:
		return e.prepareInt32(v), nil

	case Int64Value:
		return e.prepareInt64(v), nil

	case Int128Value:
		return e.prepareInt128(v), nil

	case Int256Value:
		return e.prepareInt256(v), nil

	// UInt*

	case UIntValue:
		return e.prepareUInt(v), nil

	case UInt8Value:
		return e.prepareUInt8(v), nil

	case UInt16Value:
		return e.prepareUInt16(v), nil

	case UInt32Value:
		return e.prepareUInt32(v), nil

	case UInt64Value:
		return e.prepareUInt64(v), nil

	case UInt128Value:
		return e.prepareUInt128(v), nil

	case UInt256Value:
		return e.prepareUInt256(v), nil

	// Word*

	case Word8Value:
		return e.prepareWord8(v), nil

	case Word16Value:
		return e.prepareWord16(v), nil

	case Word32Value:
		return e.prepareWord32(v), nil

	case Word64Value:
		return e.prepareWord64(v), nil

	// Fix*

	case Fix64Value:
		return e.prepareFix64(v), nil

	// UFix*

	case UFix64Value:
		return e.prepareUFix64(v), nil

	// String

	case *StringValue:
		return e.prepareString(v), nil

	// Collections

	case *ArrayValue:
		return e.prepareArray(v, path, deferrals)

	case *DictionaryValue:
		return e.prepareDictionaryValue(v, path, deferrals)

	// Composites

	case *CompositeValue:
		return e.prepareCompositeValue(v, path, deferrals)

	// Some

	case *SomeValue:
		return e.prepareSomeValue(v, path, deferrals)

	// Storage

	case *StorageReferenceValue:
		return e.prepareStorageReferenceValue(v), nil

	case PathValue:
		return e.preparePathValue(v), nil

	case CapabilityValue:
		return e.prepareCapabilityValue(v)

	case LinkValue:
		return e.prepareLinkValue(v)

	// Type

	case TypeValue:
		return e.prepareTypeValue(v)

	default:
		return nil, EncodingUnsupportedValueError{
			Path:  path,
			Value: v,
		}
	}
}

func (e *Encoder) prepareNil() interface{} {
	return nil
}

func (e *Encoder) prepareVoid() cbor.Tag {

	// TODO: optimize: use 0xf7, but decoded by github.com/fxamacker/cbor/v2 as Go `nil`:
	//   https://github.com/fxamacker/cbor/blob/a6ed6ff68e99cbb076997a08d19f03c453851555/README.md#limitations

	return cbor.Tag{
		Number: cborTagVoidValue,
	}
}

func (e *Encoder) prepareBool(v BoolValue) bool {
	return bool(v)
}

func (e *Encoder) prepareInt(v IntValue) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagIntValue,
		Content: prepareBigInt(v.BigInt),
	}
}

func (e *Encoder) prepareInt8(v Int8Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagInt8Value,
		Content: v,
	}
}

func (e *Encoder) prepareInt16(v Int16Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagInt16Value,
		Content: v,
	}
}

func (e *Encoder) prepareInt32(v Int32Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagInt32Value,
		Content: v,
	}
}

func (e *Encoder) prepareInt64(v Int64Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagInt64Value,
		Content: v,
	}
}

func (e *Encoder) prepareInt128(v Int128Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagInt128Value,
		Content: prepareBigInt(v.BigInt),
	}
}

func (e *Encoder) prepareInt256(v Int256Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagInt256Value,
		Content: prepareBigInt(v.BigInt),
	}
}

func (e *Encoder) prepareUInt(v UIntValue) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagUIntValue,
		Content: prepareBigInt(v.BigInt),
	}
}

func (e *Encoder) prepareUInt8(v UInt8Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagUInt8Value,
		Content: v,
	}
}

func (e *Encoder) prepareUInt16(v UInt16Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagUInt16Value,
		Content: v,
	}
}

func (e *Encoder) prepareUInt32(v UInt32Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagUInt32Value,
		Content: v,
	}
}

func (e *Encoder) prepareUInt64(v UInt64Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagUInt64Value,
		Content: v,
	}
}

func (e *Encoder) prepareUInt128(v UInt128Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagUInt128Value,
		Content: prepareBigInt(v.BigInt),
	}
}

func (e *Encoder) prepareUInt256(v UInt256Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagUInt256Value,
		Content: prepareBigInt(v.BigInt),
	}
}

func prepareBigInt(v *big.Int) cbor.Tag {
	sign := v.Sign()

	var tagNum uint64 = 2

	if sign < 0 {
		tagNum = 3

		// For negative number, convert to CBOR encoded number (-v-1).
		v = new(big.Int).Abs(v)
		v.Sub(v, bigOne)
	}

	return cbor.Tag{
		Number:  tagNum,
		Content: v.Bytes(),
	}
}

func (e *Encoder) prepareWord8(v Word8Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagWord8Value,
		Content: v,
	}
}

func (e *Encoder) prepareWord16(v Word16Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagWord16Value,
		Content: v,
	}
}

func (e *Encoder) prepareWord32(v Word32Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagWord32Value,
		Content: v,
	}
}

func (e *Encoder) prepareWord64(v Word64Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagWord64Value,
		Content: v,
	}
}

func (e *Encoder) prepareFix64(v Fix64Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagFix64Value,
		Content: v,
	}
}

func (e *Encoder) prepareUFix64(v UFix64Value) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagUFix64Value,
		Content: v,
	}
}

func (e *Encoder) prepareString(v *StringValue) string {
	return v.Str
}

// \x1F = Information Separator One
//
const pathSeparator = "\x1F"

// joinPath returns the path for a nested item, for example the index of an array,
// the key of a dictionary, or the field name of a composite.
//
func joinPath(elements []string) string {
	return strings.Join(elements, pathSeparator)
}

// joinPathElements returns the path for a nested item, for example the index of an array,
// the key of a dictionary, or the field name of a composite.
//
func joinPathElements(elements ...string) string {
	return strings.Join(elements, pathSeparator)
}

func (e *Encoder) prepareArray(
	v *ArrayValue,
	path []string,
	deferrals *EncodingDeferrals,
) (
	[]interface{},
	error,
) {
	result := make([]interface{}, len(v.Values))

	for i, value := range v.Values {
		valuePath := append(path[:], strconv.Itoa(i))
		prepared, err := e.prepare(value, valuePath, deferrals)
		if err != nil {
			return nil, err
		}
		result[i] = prepared
	}

	return result, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedDictionaryValueKeysFieldKey    uint64 = 0
	encodedDictionaryValueEntriesFieldKey uint64 = 1
)

const dictionaryKeyPathPrefix = "k"
const dictionaryValuePathPrefix = "v"

// TODO: optimize: use CBOR map, but unclear how to preserve ordering
func (e *Encoder) prepareDictionaryValue(
	v *DictionaryValue,
	path []string,
	deferrals *EncodingDeferrals,
) (
	interface{},
	error,
) {
	keysPath := append(path[:], dictionaryKeyPathPrefix)

	keys, err := e.prepareArray(v.Keys, keysPath, deferrals)
	if err != nil {
		return nil, err
	}

	entries := make(map[string]interface{}, v.Entries.Len())

	// Deferring the encoding of values is only supported if all
	// values are resources: resource typed dictionaries are moved

	deferred := e.deferred
	if deferred {

		// Iterating over the map in a non-deterministic way is OK,
		// we only determine check if all values are resources.

		for pair := v.Entries.Oldest(); pair != nil; pair = pair.Next() {
			compositeValue, ok := pair.Value.(*CompositeValue)
			if !ok || compositeValue.Kind != common.CompositeKindResource {
				deferred = false
				break
			}
		}
	}

	for _, keyValue := range v.Keys.Values {
		key := DictionaryKey(keyValue)
		entryValue, _ := v.Entries.Get(key)
		valuePath := append(path[:], dictionaryValuePathPrefix, key)

		if deferred {

			var isDeferred bool
			if v.DeferredKeys != nil {
				_, isDeferred = v.DeferredKeys.Get(key)
			}

			// If the value is not deferred, i.e. it is in memory,
			// then it must be stored under a separate storage key
			// in the owner's storage.

			if !isDeferred {
				deferrals.Values = append(deferrals.Values,
					EncodingDeferralValue{
						Key:   joinPath(valuePath),
						Value: entryValue,
					},
				)
			} else {

				// If the value is deferred, and the deferred value
				// is stored in another account's storage,
				// it must be moved.

				deferredOwner := *v.DeferredOwner
				owner := *v.Owner

				if deferredOwner != owner {

					deferredStorageKey := joinPathElements(v.DeferredStorageKeyBase, key)

					deferrals.Moves = append(deferrals.Moves,
						EncodingDeferralMove{
							DeferredOwner:      deferredOwner,
							DeferredStorageKey: deferredStorageKey,
							NewOwner:           owner,
							NewStorageKey:      joinPath(valuePath),
						},
					)
				}
			}
		} else {
			var prepared interface{}
			prepared, err = e.prepare(entryValue, valuePath, deferrals)
			if err != nil {
				return nil, err
			}
			entries[key] = prepared
		}
	}

	return cbor.Tag{
		Number: cborTagDictionaryValue,
		Content: cborMap{
			encodedDictionaryValueKeysFieldKey:    keys,
			encodedDictionaryValueEntriesFieldKey: entries,
		},
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedCompositeValueLocationFieldKey            uint64 = 0
	encodedCompositeValueTypeIDFieldKey              uint64 = 1
	encodedCompositeValueKindFieldKey                uint64 = 2
	encodedCompositeValueFieldsFieldKey              uint64 = 3
	encodedCompositeValueQualifiedIdentifierFieldKey uint64 = 4
)

func (e *Encoder) prepareCompositeValue(
	v *CompositeValue,
	path []string,
	deferrals *EncodingDeferrals,
) (
	interface{},
	error,
) {
	fields := make(map[string]interface{}, v.Fields.Len())

	for pair := v.Fields.Oldest(); pair != nil; pair = pair.Next() {
		fieldName := pair.Key
		value := pair.Value

		valuePath := append(path[:], fieldName)

		prepared, err := e.prepare(value, valuePath, deferrals)
		if err != nil {
			return nil, err
		}
		fields[fieldName] = prepared
	}

	location, err := e.prepareLocation(v.Location)
	if err != nil {
		return nil, err
	}

	return cbor.Tag{
		Number: cborTagCompositeValue,
		Content: cborMap{
			encodedCompositeValueLocationFieldKey:            location,
			encodedCompositeValueKindFieldKey:                uint(v.Kind),
			encodedCompositeValueFieldsFieldKey:              fields,
			encodedCompositeValueQualifiedIdentifierFieldKey: v.QualifiedIdentifier,
		},
	}, nil
}

func (e *Encoder) prepareSomeValue(
	v *SomeValue,
	path []string,
	deferrals *EncodingDeferrals,
) (
	interface{},
	error,
) {
	prepared, err := e.prepare(v.Value, path, deferrals)
	if err != nil {
		return nil, err
	}

	return cbor.Tag{
		Number:  cborTagSomeValue,
		Content: prepared,
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedStorageReferenceValueAuthorizedFieldKey           uint64 = 0
	encodedStorageReferenceValueTargetStorageAddressFieldKey uint64 = 1
	encodedStorageReferenceValueTargetKeyFieldKey            uint64 = 2
)

func (e *Encoder) prepareStorageReferenceValue(v *StorageReferenceValue) interface{} {
	return cbor.Tag{
		Number: cborTagStorageReferenceValue,
		Content: cborMap{
			encodedStorageReferenceValueAuthorizedFieldKey:           v.Authorized,
			encodedStorageReferenceValueTargetStorageAddressFieldKey: v.TargetStorageAddress.Bytes(),
			encodedStorageReferenceValueTargetKeyFieldKey:            v.TargetKey,
		},
	}
}

func (e *Encoder) prepareAddressValue(v AddressValue) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagAddressValue,
		Content: v.ToAddress().Bytes(),
	}
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedPathValueDomainFieldKey     uint64 = 0
	encodedPathValueIdentifierFieldKey uint64 = 1
)

func (e *Encoder) preparePathValue(v PathValue) cbor.Tag {
	return cbor.Tag{
		Number: cborTagPathValue,
		Content: cborMap{
			encodedPathValueDomainFieldKey:     uint(v.Domain),
			encodedPathValueIdentifierFieldKey: v.Identifier,
		},
	}
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedCapabilityValueAddressFieldKey    uint64 = 0
	encodedCapabilityValuePathFieldKey       uint64 = 1
	encodedCapabilityValueBorrowTypeFieldKey uint64 = 2
)

func (e *Encoder) prepareCapabilityValue(v CapabilityValue) (interface{}, error) {

	var preparedBorrowType interface{}

	if v.BorrowType != nil {
		var err error
		preparedBorrowType, err = e.prepareStaticType(v.BorrowType)
		if err != nil {
			return nil, err
		}
	}

	return cbor.Tag{
		Number: cborTagCapabilityValue,
		Content: cborMap{
			encodedCapabilityValueAddressFieldKey:    e.prepareAddressValue(v.Address),
			encodedCapabilityValuePathFieldKey:       e.preparePathValue(v.Path),
			encodedCapabilityValueBorrowTypeFieldKey: preparedBorrowType,
		},
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedAddressLocationAddressFieldKey uint64 = 0
	encodedAddressLocationNameFieldKey    uint64 = 1
)

func (e *Encoder) prepareLocation(l common.Location) (interface{}, error) {
	switch l := l.(type) {

	case common.StringLocation:
		return cbor.Tag{
			Number:  cborTagStringLocation,
			Content: string(l),
		}, nil

	case common.IdentifierLocation:
		return cbor.Tag{
			Number:  cborTagIdentifierLocation,
			Content: string(l),
		}, nil

	case common.AddressLocation:
		var content interface{}

		if l.Name == "" {
			content = l.Address.Bytes()
		} else {
			content = cborMap{
				encodedAddressLocationAddressFieldKey: l.Address.Bytes(),
				encodedAddressLocationNameFieldKey:    l.Name,
			}
		}

		return cbor.Tag{
			Number:  cborTagAddressLocation,
			Content: content,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported location: %T", l)
	}
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedLinkValueTargetPathFieldKey uint64 = 0
	encodedLinkValueTypeFieldKey       uint64 = 1
)

func (e *Encoder) prepareLinkValue(v LinkValue) (interface{}, error) {
	staticType, err := e.prepareStaticType(v.Type)
	if err != nil {
		return nil, err
	}
	return cbor.Tag{
		Number: cborTagLinkValue,
		Content: cborMap{
			encodedLinkValueTargetPathFieldKey: e.preparePathValue(v.TargetPath),
			encodedLinkValueTypeFieldKey:       staticType,
		},
	}, nil
}

func (e *Encoder) prepareStaticType(t StaticType) (interface{}, error) {
	switch v := t.(type) {
	case PrimitiveStaticType:
		return e.preparePrimitiveStaticType(v), nil

	case OptionalStaticType:
		return e.prepareOptionalStaticType(v)

	case CompositeStaticType:
		return e.prepareCompositeStaticType(v)

	case InterfaceStaticType:
		return e.prepareInterfaceStaticType(v)

	case VariableSizedStaticType:
		return e.prepareVariableSizedStaticType(v)

	case ConstantSizedStaticType:
		return e.prepareConstantSizedStaticType(v)

	case ReferenceStaticType:
		return e.prepareReferenceStaticType(v)

	case DictionaryStaticType:
		return e.prepareDictionaryStaticType(v)

	case *RestrictedStaticType:
		return e.prepareRestrictedStaticType(v)

	case CapabilityStaticType:
		return e.prepareCapabilityStaticType(v)

	default:
		return nil, fmt.Errorf("unsupported static type: %T", t)
	}
}

func (e *Encoder) preparePrimitiveStaticType(v PrimitiveStaticType) cbor.Tag {
	return cbor.Tag{
		Number:  cborTagPrimitiveStaticType,
		Content: uint(v),
	}
}

func (e *Encoder) prepareOptionalStaticType(v OptionalStaticType) (interface{}, error) {
	staticType, err := e.prepareStaticType(v.Type)
	if err != nil {
		return nil, err
	}

	return cbor.Tag{
		Number:  cborTagOptionalStaticType,
		Content: staticType,
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedCompositeStaticTypeLocationFieldKey            uint64 = 0
	encodedCompositeStaticTypeTypeIDFieldKey              uint64 = 1
	encodedCompositeStaticTypeQualifiedIdentifierFieldKey uint64 = 2
)

func (e *Encoder) prepareCompositeStaticType(v CompositeStaticType) (interface{}, error) {
	location, err := e.prepareLocation(v.Location)
	if err != nil {
		return nil, err
	}

	return cbor.Tag{
		Number: cborTagCompositeStaticType,
		Content: cborMap{
			encodedCompositeStaticTypeLocationFieldKey:            location,
			encodedCompositeStaticTypeQualifiedIdentifierFieldKey: v.QualifiedIdentifier,
		},
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedInterfaceStaticTypeLocationFieldKey            uint64 = 0
	encodedInterfaceStaticTypeTypeIDFieldKey              uint64 = 1
	encodedInterfaceStaticTypeQualifiedIdentifierFieldKey uint64 = 2
)

func (e *Encoder) prepareInterfaceStaticType(v InterfaceStaticType) (interface{}, error) {
	location, err := e.prepareLocation(v.Location)
	if err != nil {
		return nil, err
	}

	return cbor.Tag{
		Number: cborTagInterfaceStaticType,
		Content: cborMap{
			encodedInterfaceStaticTypeLocationFieldKey:            location,
			encodedInterfaceStaticTypeQualifiedIdentifierFieldKey: v.QualifiedIdentifier,
		},
	}, nil
}

func (e *Encoder) prepareVariableSizedStaticType(v VariableSizedStaticType) (interface{}, error) {
	staticType, err := e.prepareStaticType(v.Type)
	if err != nil {
		return nil, err
	}

	return cbor.Tag{
		Number:  cborTagVariableSizedStaticType,
		Content: staticType,
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedConstantSizedStaticTypeSizeFieldKey uint64 = 0
	encodedConstantSizedStaticTypeTypeFieldKey uint64 = 1
)

func (e *Encoder) prepareConstantSizedStaticType(v ConstantSizedStaticType) (interface{}, error) {
	staticType, err := e.prepareStaticType(v.Type)
	if err != nil {
		return nil, err
	}

	return cbor.Tag{
		Number: cborTagConstantSizedStaticType,
		Content: cborMap{
			encodedConstantSizedStaticTypeSizeFieldKey: v.Size,
			encodedConstantSizedStaticTypeTypeFieldKey: staticType,
		},
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedReferenceStaticTypeAuthorizedFieldKey uint64 = 0
	encodedReferenceStaticTypeTypeFieldKey       uint64 = 1
)

func (e *Encoder) prepareReferenceStaticType(v ReferenceStaticType) (interface{}, error) {
	staticType, err := e.prepareStaticType(v.Type)
	if err != nil {
		return nil, err
	}

	return cbor.Tag{
		Number: cborTagReferenceStaticType,
		Content: cborMap{
			encodedReferenceStaticTypeAuthorizedFieldKey: v.Authorized,
			encodedReferenceStaticTypeTypeFieldKey:       staticType,
		},
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedDictionaryStaticTypeKeyTypeFieldKey   uint64 = 0
	encodedDictionaryStaticTypeValueTypeFieldKey uint64 = 1
)

func (e *Encoder) prepareDictionaryStaticType(v DictionaryStaticType) (interface{}, error) {
	keyType, err := e.prepareStaticType(v.KeyType)
	if err != nil {
		return nil, err
	}

	valueType, err := e.prepareStaticType(v.ValueType)
	if err != nil {
		return nil, err
	}

	return cbor.Tag{
		Number: cborTagDictionaryStaticType,
		Content: cborMap{
			encodedDictionaryStaticTypeKeyTypeFieldKey:   keyType,
			encodedDictionaryStaticTypeValueTypeFieldKey: valueType,
		},
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedRestrictedStaticTypeTypeFieldKey         uint64 = 0
	encodedRestrictedStaticTypeRestrictionsFieldKey uint64 = 1
)

func (e *Encoder) prepareRestrictedStaticType(v *RestrictedStaticType) (interface{}, error) {
	restrictedType, err := e.prepareStaticType(v.Type)
	if err != nil {
		return nil, err
	}

	encodedRestrictions := make([]interface{}, len(v.Restrictions))
	for i, restriction := range v.Restrictions {
		encodedRestriction, err := e.prepareStaticType(restriction)
		if err != nil {
			return nil, err
		}

		encodedRestrictions[i] = encodedRestriction
	}

	return cbor.Tag{
		Number: cborTagRestrictedStaticType,
		Content: cborMap{
			encodedRestrictedStaticTypeTypeFieldKey:         restrictedType,
			encodedRestrictedStaticTypeRestrictionsFieldKey: encodedRestrictions,
		},
	}, nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedTypeValueTypeFieldKey uint64 = 0
)

func (e *Encoder) prepareTypeValue(v TypeValue) (interface{}, error) {

	content := cborMap{}

	staticType := v.Type
	if staticType != nil {
		preparedStaticType, err := e.prepareStaticType(staticType)
		if err != nil {
			return nil, err
		}

		content[encodedTypeValueTypeFieldKey] = preparedStaticType
	}

	return cbor.Tag{
		Number:  cborTagTypeValue,
		Content: content,
	}, nil
}

func (e *Encoder) prepareCapabilityStaticType(v CapabilityStaticType) (interface{}, error) {
	var borrowStaticType interface{}
	if v.BorrowType != nil {
		var err error
		borrowStaticType, err = e.prepareStaticType(v.BorrowType)
		if err != nil {
			return nil, err
		}
	}

	return cbor.Tag{
		Number:  cborTagCapabilityStaticType,
		Content: borrowStaticType,
	}, nil
}
