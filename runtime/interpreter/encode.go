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
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/cadence/runtime/common"
)

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
	cborTagStorageReferenceValue // deprecated
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
	enc             *cbor.StreamEncoder
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

	// Write streamed data to writer.
	err = enc.enc.Flush()
	if err != nil {
		return nil, nil, err
	}

	data := w.Bytes()

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
	enc := encMode.NewStreamEncoder(w)
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
	if e.prepareCallback != nil {
		e.prepareCallback(v, path)
	}

	switch v := v.(type) {

	case NilValue:
		return e.enc.EncodeNil()

	case VoidValue:
		return e.encodeVoid()

	case BoolValue:
		return e.enc.EncodeBool(bool(v))

	case AddressValue:
		return e.encodeAddressValue(v)

	// Int*

	case IntValue:
		return e.encodeInt(v)

	case Int8Value:
		return e.encodeInt8(v)

	case Int16Value:
		return e.encodeInt16(v)

	case Int32Value:
		return e.encodeInt32(v)

	case Int64Value:
		return e.encodeInt64(v)

	case Int128Value:
		return e.encodeInt128(v)

	case Int256Value:
		return e.encodeInt256(v)

	// UInt*

	case UIntValue:
		return e.encodeUInt(v)

	case UInt8Value:
		return e.encodeUInt8(v)

	case UInt16Value:
		return e.encodeUInt16(v)

	case UInt32Value:
		return e.encodeUInt32(v)

	case UInt64Value:
		return e.encodeUInt64(v)

	case UInt128Value:
		return e.encodeUInt128(v)

	case UInt256Value:
		return e.encodeUInt256(v)

	// Word*

	case Word8Value:
		return e.encodeWord8(v)

	case Word16Value:
		return e.encodeWord16(v)

	case Word32Value:
		return e.encodeWord32(v)

	case Word64Value:
		return e.encodeWord64(v)

	// Fix*

	case Fix64Value:
		return e.encodeFix64(v)

	// UFix*

	case UFix64Value:
		return e.encodeUFix64(v)

	// String

	case *StringValue:
		return e.enc.EncodeString(v.Str)

	// Collections

	case *ArrayValue:
		return e.encodeArray(v, path, deferrals)

	case *DictionaryValue:
		return e.encodeDictionaryValue(v, path, deferrals)

	// Composites

	case *CompositeValue:
		return e.encodeCompositeValue(v, path, deferrals)

	// Some

	case *SomeValue:
		return e.encodeSomeValue(v, path, deferrals)

	// Storage

	case PathValue:
		return e.encodePathValue(v)

	case CapabilityValue:
		return e.encodeCapabilityValue(v)

	case LinkValue:
		return e.encodeLinkValue(v)

	// Type

	case TypeValue:
		return e.encodeTypeValue(v)

	default:
		return EncodingUnsupportedValueError{
			Path:  path,
			Value: v,
		}
	}
}

// cborVoidValue represents the CBOR value:
//
// 	cbor.Tag{
// 		Number: cborTagVoidValue,
// 		Content: nil
// 	}
//
var cborVoidValue = []byte{
	// tag
	0xd8, cborTagVoidValue,
	// null
	0xf6,
}

// encodeVoid writes a value of type Void to the encoder
//
func (e *Encoder) encodeVoid() error {

	// TODO: optimize: use 0xf7, but decoded by github.com/fxamacker/cbor/v2 as Go `nil`:
	//   https://github.com/fxamacker/cbor/blob/a6ed6ff68e99cbb076997a08d19f03c453851555/README.md#limitations

	return e.enc.EncodeRawBytes(cborVoidValue)
}

// encodeInt encodes IntValue as
// cbor.Tag{
//		Number:  cborTagIntValue,
//		Content: *big.Int(v.BigInt),
// }
func (e *Encoder) encodeInt(v IntValue) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagIntValue,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeBigInt(v.BigInt)
}

// encodeInt8 encodes Int8Value as
// cbor.Tag{
//		Number:  cborTagInt8Value,
//		Content: int8(v),
// }
func (e *Encoder) encodeInt8(v Int8Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagInt8Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeInt8(int8(v))
}

// encodeInt16 encodes Int16Value as
// cbor.Tag{
//		Number:  cborTagInt16Value,
//		Content: int16(v),
// }
func (e *Encoder) encodeInt16(v Int16Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagInt16Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeInt16(int16(v))
}

// encodeInt32 encodes Int32Value as
// cbor.Tag{
//		Number:  cborTagInt32Value,
//		Content: int32(v),
// }
func (e *Encoder) encodeInt32(v Int32Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagInt32Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeInt32(int32(v))
}

// encodeInt64 encodes Int64Value as
// cbor.Tag{
//		Number:  cborTagInt64Value,
//		Content: int64(v),
// }
func (e *Encoder) encodeInt64(v Int64Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagInt64Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeInt64(int64(v))
}

// encodeInt128 encodes Int128Value as
// cbor.Tag{
//		Number:  cborTagInt128Value,
//		Content: *big.Int(v.BigInt),
// }
func (e *Encoder) encodeInt128(v Int128Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagInt128Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeBigInt(v.BigInt)
}

// encodeInt256 encodes Int256Value as
// cbor.Tag{
//		Number:  cborTagInt256Value,
//		Content: *big.Int(v.BigInt),
// }
func (e *Encoder) encodeInt256(v Int256Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagInt256Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeBigInt(v.BigInt)
}

// encodeUInt encodes UIntValue as
// cbor.Tag{
//		Number:  cborTagUIntValue,
//		Content: *big.Int(v.BigInt),
// }
func (e *Encoder) encodeUInt(v UIntValue) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagUIntValue,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeBigInt(v.BigInt)
}

// encodeUInt8 encodes UInt8Value as
// cbor.Tag{
//		Number:  cborTagUInt8Value,
//		Content: uint8(v),
// }
func (e *Encoder) encodeUInt8(v UInt8Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagUInt8Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint8(uint8(v))
}

// encodeUInt16 encodes UInt16Value as
// cbor.Tag{
//		Number:  cborTagUInt16Value,
//		Content: uint16(v),
// }
func (e *Encoder) encodeUInt16(v UInt16Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagUInt16Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint16(uint16(v))
}

// encodeUInt32 encodes UInt32Value as
// cbor.Tag{
//		Number:  cborTagUInt32Value,
//		Content: uint32(v),
// }
func (e *Encoder) encodeUInt32(v UInt32Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagUInt32Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint32(uint32(v))
}

// encodeUInt64 encodes UInt64Value as
// cbor.Tag{
//		Number:  cborTagUInt64Value,
//		Content: uint64(v),
// }
func (e *Encoder) encodeUInt64(v UInt64Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagUInt64Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint64(uint64(v))
}

// encodeUInt128 encodes UInt128Value as
// cbor.Tag{
//		Number:  cborTagUInt128Value,
//		Content: *big.Int(v.BigInt),
// }
func (e *Encoder) encodeUInt128(v UInt128Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagUInt128Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeBigInt(v.BigInt)
}

// encodeUInt256 encodes UInt256Value as
// cbor.Tag{
//		Number:  cborTagUInt256Value,
//		Content: *big.Int(v.BigInt),
// }
func (e *Encoder) encodeUInt256(v UInt256Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagUInt256Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeBigInt(v.BigInt)
}

// encodeWord8 encodes Word8Value as
// cbor.Tag{
//		Number:  cborTagWord8Value,
//		Content: uint8(v),
// }
func (e *Encoder) encodeWord8(v Word8Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagWord8Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint8(uint8(v))
}

// encodeWord16 encodes Word16Value as
// cbor.Tag{
//		Number:  cborTagWord16Value,
//		Content: uint16(v),
// }
func (e *Encoder) encodeWord16(v Word16Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagWord16Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint16(uint16(v))
}

// encodeWord32 encodes Word32Value as
// cbor.Tag{
//		Number:  cborTagWord32Value,
//		Content: uint32(v),
// }
func (e *Encoder) encodeWord32(v Word32Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagWord32Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint32(uint32(v))
}

// encodeWord64 encodes Word64Value as
// cbor.Tag{
//		Number:  cborTagWord64Value,
//		Content: uint64(v),
// }
func (e *Encoder) encodeWord64(v Word64Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagWord64Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint64(uint64(v))
}

// encodeFix64 encodes Fix64Value as
// cbor.Tag{
//		Number:  cborTagFix64Value,
//		Content: int64(v),
// }
func (e *Encoder) encodeFix64(v Fix64Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagFix64Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeInt64(int64(v))
}

// encodeUFix64 encodes UFix64Value as
// cbor.Tag{
//		Number:  cborTagUFix64Value,
//		Content: uint64(v),
// }
func (e *Encoder) encodeUFix64(v UFix64Value) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagUFix64Value,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint64(uint64(v))
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

// encodeArray encodes ArrayValue as []interface{}(v)
func (e *Encoder) encodeArray(
	v *ArrayValue,
	path []string,
	deferrals *EncodingDeferrals,
) error {
	err := e.enc.EncodeArrayHead(uint64(len(v.Values)))
	if err != nil {
		return err
	}

	// Pre-allocate and reuse valuePath.
	//nolint:gocritic
	valuePath := append(path, "")

	lastValuePathIndex := len(path)

	for i, value := range v.Values {
		valuePath[lastValuePathIndex] = strconv.Itoa(i)

		err := e.Encode(value, valuePath, deferrals)
		if err != nil {
			return err
		}
	}

	return nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedDictionaryValueKeysFieldKey    uint64 = 0
	encodedDictionaryValueEntriesFieldKey uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedDictionaryValueLength MUST be updated when new element is added.
	// It is used to verify encoded dictionaries length during decoding.
	encodedDictionaryValueLength = 2
)

const dictionaryKeyPathPrefix = "k"
const dictionaryValuePathPrefix = "v"

// encodeDictionaryValue encodes DictionaryValue as
// cbor.Tag{
//			Number: cborTagDictionaryValue,
//			Content: cborArray{
//				encodedDictionaryValueKeysFieldKey:    []interface{}(keys),
//				encodedDictionaryValueEntriesFieldKey: []interface{}(entries),
//			},
// }
func (e *Encoder) encodeDictionaryValue(
	v *DictionaryValue,
	path []string,
	deferrals *EncodingDeferrals,
) error {
	// Encode CBOR tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagDictionaryValue,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	//nolint:gocritic
	keysPath := append(path, dictionaryKeyPathPrefix)

	// Encode keys (as array) at array index encodedDictionaryValueKeysFieldKey
	err = e.encodeArray(v.Keys, keysPath, deferrals)
	if err != nil {
		return err
	}

	// Deferring the encoding of values is only supported if all
	// values are resources: resource typed dictionaries are moved

	deferred := e.deferred
	if deferred {

		// Iterating over the map in a non-deterministic way is OK,
		// we only determine check if all values are resources.

		for pair := v.Entries.Oldest(); pair != nil; pair = pair.Next() {
			compositeValue, ok := pair.Value.(*CompositeValue)
			if !ok || compositeValue.Kind() != common.CompositeKindResource {
				deferred = false
				break
			}
		}
	}

	// entries is empty if encoding of values is deferred,
	// otherwise entries size is the same as keys size.
	entriesLength := len(v.Keys.Values)
	if deferred {
		entriesLength = 0
	}

	// Encode values (as array) at array index encodedDictionaryValueEntriesFieldKey
	err = e.enc.EncodeArrayHead(uint64(entriesLength))
	if err != nil {
		return err
	}

	// Pre-allocate and reuse valuePath.
	//nolint:gocritic
	valuePath := append(path, dictionaryValuePathPrefix, "")

	lastValuePathIndex := len(path) + 1

	for _, keyValue := range v.Keys.Values {
		key := dictionaryKey(keyValue)
		entryValue, _ := v.Entries.Get(key)
		valuePath[lastValuePathIndex] = key

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
			// Encode value as element in values array
			err = e.Encode(entryValue, valuePath, deferrals)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedCompositeValueLocationFieldKey            uint64 = 0
	encodedCompositeValueTypeIDFieldKey              uint64 = 1
	encodedCompositeValueKindFieldKey                uint64 = 2
	encodedCompositeValueFieldsFieldKey              uint64 = 3
	encodedCompositeValueQualifiedIdentifierFieldKey uint64 = 4

	// !!! *WARNING* !!!
	//
	// encodedCompositeValueLength MUST be updated when new element is added.
	// It is used to verify encoded composites length during decoding.
	encodedCompositeValueLength = 5
)

// encodeCompositeValue encodes CompositeValue as
// cbor.Tag{
//		Number: cborTagCompositeValue,
//		Content: cborArray{
//			encodedCompositeValueLocationFieldKey:            common.Location(location),
//			encodedCompositeValueTypeIDFieldKey:              nil,
//			encodedCompositeValueKindFieldKey:                uint(v.Kind),
//			encodedCompositeValueFieldsFieldKey:              []interface{}(fields),
//			encodedCompositeValueQualifiedIdentifierFieldKey: string(v.QualifiedIdentifier),
//		},
// }
func (e *Encoder) encodeCompositeValue(
	v *CompositeValue,
	path []string,
	deferrals *EncodingDeferrals,
) error {

	// Encode CBOR tag number
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagCompositeValue,
	})

	if err != nil {
		return err
	}

	// If the value is not loaded, dump the raw content as it is.
	if v.content != nil {
		err = e.enc.EncodeRawBytes(v.content)
		if err != nil {
			return err
		}

		return nil
	}

	// Encode array head
	err = e.enc.EncodeRawBytes([]byte{
		// array, 5 items follow
		0x85,
	})
	if err != nil {
		return err
	}

	// Encode location at array index encodedCompositeValueLocationFieldKey
	err = e.encodeLocation(v.Location())
	if err != nil {
		return err
	}

	// Encode nil (obsolete) at array index encodedCompositeValueTypeIDFieldKey
	err = e.enc.EncodeNil()
	if err != nil {
		return err
	}

	// Encode kind at array index encodedCompositeValueKindFieldKey
	err = e.enc.EncodeUint(uint(v.Kind()))
	if err != nil {
		return err
	}

	// Encode fields (as array) at array index encodedCompositeValueFieldsFieldKey

	// If the fields are not loaded, dump the raw fields content as it is.
	if v.fieldsContent != nil {
		err := e.enc.EncodeRawBytes(v.fieldsContent)
		if err != nil {
			return err
		}
	} else {
		fields := v.Fields()
		err = e.enc.EncodeArrayHead(uint64(fields.Len() * 2))
		if err != nil {
			return err
		}

	// Pre-allocate and reuse valuePath.
	//nolint:gocritic
	valuePath := append(path, "")

		lastValuePathIndex := len(path)

		for pair := fields.Oldest(); pair != nil; pair = pair.Next() {
			fieldName := pair.Key

			// Encode field name as fields array element
			err := e.enc.EncodeString(fieldName)
			if err != nil {
				return err
			}

			value := pair.Value

			valuePath[lastValuePathIndex] = fieldName

			// Encode value as fields array element
			err = e.Encode(value, valuePath, deferrals)
			if err != nil {
				return err
			}
		}
	}

	// Encode qualified identifier at array index encodedCompositeValueQualifiedIdentifierFieldKey
	err = e.enc.EncodeString(v.QualifiedIdentifier())
	if err != nil {
		return err
	}

	return nil
}

// encodeSomeValue encodes SomeValue as
// cbor.Tag{
//		Number: cborTagSomeValue,
//		Content: Value(v.Value),
// }
func (e *Encoder) encodeSomeValue(
	v *SomeValue,
	path []string,
	deferrals *EncodingDeferrals,
) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagSomeValue,
	})
	if err != nil {
		return err
	}
	return e.Encode(v.Value, path, deferrals)
}

// encodeAddressValue encodes AddressValue as
// cbor.Tag{
//		Number:  cborTagAddressValue,
//		Content: []byte(v.ToAddress().Bytes()),
// }
func (e *Encoder) encodeAddressValue(v AddressValue) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagAddressValue,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeBytes(v.ToAddress().Bytes())
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedPathValueDomainFieldKey     uint64 = 0
	encodedPathValueIdentifierFieldKey uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedPathValueLength MUST be updated when new element is added.
	// It is used to verify encoded path length during decoding.
	encodedPathValueLength = 2
)

// encodePathValue encodes PathValue as
// cbor.Tag{
//			Number: cborTagPathValue,
//			Content: []interface{}{
//				encodedPathValueDomainFieldKey:     uint(v.Domain),
//				encodedPathValueIdentifierFieldKey: string(v.Identifier),
//			},
// }
func (e *Encoder) encodePathValue(v PathValue) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagPathValue,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// Encode domain at array index encodedPathValueDomainFieldKey
	err = e.enc.EncodeUint(uint(v.Domain))
	if err != nil {
		return err
	}

	// Encode identifier at array index encodedPathValueIdentifierFieldKey
	return e.enc.EncodeString(v.Identifier)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedCapabilityValueAddressFieldKey    uint64 = 0
	encodedCapabilityValuePathFieldKey       uint64 = 1
	encodedCapabilityValueBorrowTypeFieldKey uint64 = 2

	// !!! *WARNING* !!!
	//
	// encodedCapabilityValueLength MUST be updated when new element is added.
	// It is used to verify encoded capability length during decoding.
	encodedCapabilityValueLength = 3
)

// encodeCapabilityValue encodes CapabilityValue as
// cbor.Tag{
//			Number: cborTagCapabilityValue,
//			Content: []interface{}{
//					encodedCapabilityValueAddressFieldKey:    AddressValue(v.Address),
// 					encodedCapabilityValuePathFieldKey:       PathValue(v.Path),
// 					encodedCapabilityValueBorrowTypeFieldKey: StaticType(v.BorrowType),
// 				},
// }
func (e *Encoder) encodeCapabilityValue(v CapabilityValue) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagCapabilityValue,
		// array, 3 items follow
		0x83,
	})
	if err != nil {
		return err
	}

	// Encode address at array index encodedCapabilityValueAddressFieldKey
	err = e.encodeAddressValue(v.Address)
	if err != nil {
		return err
	}

	// Encode path at array index encodedCapabilityValuePathFieldKey
	err = e.encodePathValue(v.Path)
	if err != nil {
		return err
	}

	// Encode borrow type at array index encodedCapabilityValueBorrowTypeFieldKey
	return e.encodeStaticType(v.BorrowType)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedAddressLocationAddressFieldKey uint64 = 0
	encodedAddressLocationNameFieldKey    uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedAddressLocationLength MUST be updated when new element is added.
	// It is used to verify encoded address location length during decoding.
	encodedAddressLocationLength = 2
)

func (e *Encoder) encodeLocation(l common.Location) error {
	switch l := l.(type) {

	case common.StringLocation:
		// common.StringLocation is encoded as
		// cbor.Tag{
		//		Number:  cborTagStringLocation,
		//		Content: string(l),
		// }
		err := e.enc.EncodeRawBytes([]byte{
			// tag number
			0xd8, cborTagStringLocation,
		})
		if err != nil {
			return err
		}
		return e.enc.EncodeString(string(l))

	case common.IdentifierLocation:
		// common.IdentifierLocation is encoded as
		// cbor.Tag{
		//		Number:  cborTagIdentifierLocation,
		//		Content: string(l),
		// }
		err := e.enc.EncodeRawBytes([]byte{
			// tag number
			0xd8, cborTagIdentifierLocation,
		})
		if err != nil {
			return err
		}
		return e.enc.EncodeString(string(l))

	case common.AddressLocation:
		// common.AddressLocation is encoded as
		// cbor.Tag{
		//		Number: cborTagAddressLocation,
		//		Content: []interface{}{
		//			encodedAddressLocationAddressFieldKey: []byte{l.Address.Bytes()},
		//			encodedAddressLocationNameFieldKey:    string(l.Name),
		//		},
		// }
		// Encode tag number and array head
		err := e.enc.EncodeRawBytes([]byte{
			// tag number
			0xd8, cborTagAddressLocation,
			// array, 2 items follow
			0x82,
		})
		if err != nil {
			return err
		}
		// Encode address at array index encodedAddressLocationAddressFieldKey
		err = e.enc.EncodeBytes(l.Address.Bytes())
		if err != nil {
			return err
		}
		// Encode name at array index encodedAddressLocationNameFieldKey
		return e.enc.EncodeString(l.Name)
	default:
		return fmt.Errorf("unsupported location: %T", l)
	}
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedLinkValueTargetPathFieldKey uint64 = 0
	encodedLinkValueTypeFieldKey       uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedLinkValueLength MUST be updated when new element is added.
	// It is used to verify encoded link length during decoding.
	encodedLinkValueLength = 2
)

// encodeLinkValue encodes LinkValue as
// cbor.Tag{
//			Number: cborTagLinkValue,
//			Content: []interface{}{
//				encodedLinkValueTargetPathFieldKey: PathValue(v.TargetPath),
//				encodedLinkValueTypeFieldKey:       StaticType(v.Type),
//			},
// }
func (e *Encoder) encodeLinkValue(v LinkValue) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagLinkValue,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode path at array index encodedLinkValueTargetPathFieldKey
	err = e.encodePathValue(v.TargetPath)
	if err != nil {
		return err
	}
	// Encode type at array index encodedLinkValueTypeFieldKey
	return e.encodeStaticType(v.Type)
}

func (e *Encoder) encodeStaticType(t StaticType) error {
	if t == nil {
		return e.enc.EncodeNil()
	}

	switch v := t.(type) {
	case PrimitiveStaticType:
		return e.encodePrimitiveStaticType(v)

	case OptionalStaticType:
		return e.encodeOptionalStaticType(v)

	case CompositeStaticType:
		return e.encodeCompositeStaticType(v)

	case InterfaceStaticType:
		return e.encodeInterfaceStaticType(v)

	case VariableSizedStaticType:
		return e.encodeVariableSizedStaticType(v)

	case ConstantSizedStaticType:
		return e.encodeConstantSizedStaticType(v)

	case ReferenceStaticType:
		return e.encodeReferenceStaticType(v)

	case DictionaryStaticType:
		return e.encodeDictionaryStaticType(v)

	case *RestrictedStaticType:
		return e.encodeRestrictedStaticType(v)

	case CapabilityStaticType:
		return e.encodeCapabilityStaticType(v)

	default:
		return fmt.Errorf("unsupported static type: %T", t)
	}
}

// encodePrimitiveStaticType encodes PrimitiveStaticType as
// cbor.Tag{
//		Number:  cborTagPrimitiveStaticType,
//		Content: uint(v),
// }
func (e *Encoder) encodePrimitiveStaticType(v PrimitiveStaticType) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagPrimitiveStaticType,
	})
	if err != nil {
		return err
	}
	return e.enc.EncodeUint(uint(v))
}

// encodeOptionalStaticType encodes OptionalStaticType as
// cbor.Tag{
//		Number:  cborTagOptionalStaticType,
//		Content: StaticType(v.Type),
// }
func (e *Encoder) encodeOptionalStaticType(v OptionalStaticType) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagOptionalStaticType,
	})
	if err != nil {
		return err
	}
	return e.encodeStaticType(v.Type)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedCompositeStaticTypeLocationFieldKey            uint64 = 0
	encodedCompositeStaticTypeTypeIDFieldKey              uint64 = 1
	encodedCompositeStaticTypeQualifiedIdentifierFieldKey uint64 = 2

	// !!! *WARNING* !!!
	//
	// encodedCompositeStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded composite static type length during decoding.
	encodedCompositeStaticTypeLength = 3
)

// encodeCompositeStaticType encodes CompositeStaticType as
// cbor.Tag{
//			Number: cborTagCompositeStaticType,
// 			Content: cborArray{
//				encodedCompositeStaticTypeLocationFieldKey:            Location(v.Location),
// 				encodedCompositeStaticTypeTypeIDFieldKey:              nil,
//				encodedCompositeStaticTypeQualifiedIdentifierFieldKey: string(v.QualifiedIdentifier),
//		},
// }
func (e *Encoder) encodeCompositeStaticType(v CompositeStaticType) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagCompositeStaticType,
		// array, 3 items follow
		0x83,
	})
	if err != nil {
		return err
	}
	// Encode location at array index encodedCompositeStaticTypeLocationFieldKey
	err = e.encodeLocation(v.Location)
	if err != nil {
		return err
	}
	// Encode nil (obsolete) at array index encodedCompositeStaticTypeTypeIDFieldKey
	err = e.enc.EncodeNil()
	if err != nil {
		return err
	}
	// Encode qualified identifier at array index encodedCompositeStaticTypeQualifiedIdentifierFieldKey
	return e.enc.EncodeString(v.QualifiedIdentifier)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedInterfaceStaticTypeLocationFieldKey            uint64 = 0
	encodedInterfaceStaticTypeTypeIDFieldKey              uint64 = 1
	encodedInterfaceStaticTypeQualifiedIdentifierFieldKey uint64 = 2

	// !!! *WARNING* !!!
	//
	// encodedInterfaceStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded interface static type length during decoding.
	encodedInterfaceStaticTypeLength = 3
)

// encodeInterfaceStaticType encodes InterfaceStaticType as
// cbor.Tag{
//		Number: cborTagInterfaceStaticType,
//		Content: cborArray{
//				encodedInterfaceStaticTypeLocationFieldKey:            Location(v.Location),
// 				encodedInterfaceStaticTypeTypeIDFieldKey:              nil,
//				encodedInterfaceStaticTypeQualifiedIdentifierFieldKey: string(v.QualifiedIdentifier),
//		},
// }
func (e *Encoder) encodeInterfaceStaticType(v InterfaceStaticType) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagInterfaceStaticType,
		// array, 3 items follow
		0x83,
	})
	if err != nil {
		return err
	}
	// Encode location at array index encodedInterfaceStaticTypeLocationFieldKey
	err = e.encodeLocation(v.Location)
	if err != nil {
		return err
	}
	// Encode nil (obsolete) at array index encodedInterfaceStaticTypeTypeIDFieldKey
	err = e.enc.EncodeNil()
	if err != nil {
		return err
	}
	// Encode qualified identifier at array index encodedInterfaceStaticTypeQualifiedIdentifierFieldKey
	return e.enc.EncodeString(v.QualifiedIdentifier)
}

// encodeVariableSizedStaticType encodes VariableSizedStaticType as
// cbor.Tag{
//		Number:  cborTagVariableSizedStaticType,
//		Content: StaticType(v.Type),
// }
func (e *Encoder) encodeVariableSizedStaticType(v VariableSizedStaticType) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagVariableSizedStaticType,
	})
	if err != nil {
		return err
	}
	return e.encodeStaticType(v.Type)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedConstantSizedStaticTypeSizeFieldKey uint64 = 0
	encodedConstantSizedStaticTypeTypeFieldKey uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedConstantSizedStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded constant sized static type length during decoding.
	encodedConstantSizedStaticTypeLength = 2
)

// encodeConstantSizedStaticType encodes ConstantSizedStaticType as
// cbor.Tag{
//		Number: cborTagConstantSizedStaticType,
//		Content: cborArray{
//				encodedConstantSizedStaticTypeSizeFieldKey: int64(v.Size),
//				encodedConstantSizedStaticTypeTypeFieldKey: StaticType(v.Type),
//		},
// }
func (e *Encoder) encodeConstantSizedStaticType(v ConstantSizedStaticType) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagConstantSizedStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode size at array index encodedConstantSizedStaticTypeSizeFieldKey
	err = e.enc.EncodeInt64(v.Size)
	if err != nil {
		return err
	}
	// Encode type at array index encodedConstantSizedStaticTypeTypeFieldKey
	return e.encodeStaticType(v.Type)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedReferenceStaticTypeAuthorizedFieldKey uint64 = 0
	encodedReferenceStaticTypeTypeFieldKey       uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedReferenceStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded reference static type length during decoding.
	encodedReferenceStaticTypeLength = 2
)

// encodeReferenceStaticType encodes ReferenceStaticType as
// cbor.Tag{
//		Number: cborTagReferenceStaticType,
//		Content: cborArray{
//				encodedReferenceStaticTypeAuthorizedFieldKey: bool(v.Authorized),
//				encodedReferenceStaticTypeTypeFieldKey:       StaticType(v.Type),
//		},
//	}
func (e *Encoder) encodeReferenceStaticType(v ReferenceStaticType) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagReferenceStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode authorized at array index encodedReferenceStaticTypeAuthorizedFieldKey
	err = e.enc.EncodeBool(v.Authorized)
	if err != nil {
		return err
	}
	// Encode type at array index encodedReferenceStaticTypeTypeFieldKey
	return e.encodeStaticType(v.Type)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedDictionaryStaticTypeKeyTypeFieldKey   uint64 = 0
	encodedDictionaryStaticTypeValueTypeFieldKey uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedDictionaryStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded dictionary static type length during decoding.
	encodedDictionaryStaticTypeLength = 2
)

// encodeDictionaryStaticType encodes DictionaryStaticType as
// cbor.Tag{
//		Number: cborTagDictionaryStaticType,
// 		Content: []interface{}{
//				encodedDictionaryStaticTypeKeyTypeFieldKey:   StaticType(v.KeyType),
//				encodedDictionaryStaticTypeValueTypeFieldKey: StaticType(v.ValueType),
//		},
// }
func (e *Encoder) encodeDictionaryStaticType(v DictionaryStaticType) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagDictionaryStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode key type at array index encodedDictionaryStaticTypeKeyTypeFieldKey
	err = e.encodeStaticType(v.KeyType)
	if err != nil {
		return err
	}
	// Encode value type at array index encodedDictionaryStaticTypeValueTypeFieldKey
	return e.encodeStaticType(v.ValueType)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedRestrictedStaticTypeTypeFieldKey         uint64 = 0
	encodedRestrictedStaticTypeRestrictionsFieldKey uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedRestrictedStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded restricted static type length during decoding.
	encodedRestrictedStaticTypeLength = 2
)

// encodeRestrictedStaticType encodes RestrictedStaticType as
// cbor.Tag{
//		Number: cborTagRestrictedStaticType,
//		Content: cborArray{
//				encodedRestrictedStaticTypeTypeFieldKey:         StaticType(v.Type),
//				encodedRestrictedStaticTypeRestrictionsFieldKey: []interface{}(v.Restrictions),
//		},
// }
func (e *Encoder) encodeRestrictedStaticType(v *RestrictedStaticType) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagRestrictedStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode type at array index encodedRestrictedStaticTypeTypeFieldKey
	err = e.encodeStaticType(v.Type)
	if err != nil {
		return err
	}
	// Encode restrictions (as array) at array index encodedRestrictedStaticTypeRestrictionsFieldKey
	err = e.enc.EncodeArrayHead(uint64(len(v.Restrictions)))
	if err != nil {
		return err
	}
	for _, restriction := range v.Restrictions {
		// Encode restriction as array restrictions element
		err = e.encodeInterfaceStaticType(restriction)
		if err != nil {
			return err
		}
	}
	return nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	encodedTypeValueTypeFieldKey uint64 = 0

	// !!! *WARNING* !!!
	//
	// encodedTypeValueTypeLength MUST be updated when new element is added.
	// It is used to verify encoded type length during decoding.
	encodedTypeValueTypeLength = 1
)

// encodeTypeValue encodes TypeValue as
// cbor.Tag{
//			Number: cborTagTypeValue,
//			Content: cborArray{
//				encodedTypeValueTypeFieldKey: StaticType(v.Type),
//			},
//	}
func (e *Encoder) encodeTypeValue(v TypeValue) error {
	// Encode tag number and array head
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagTypeValue,
		// array, 1 item follow
		0x81,
	})
	if err != nil {
		return err
	}
	// Encode type at array index encodedTypeValueTypeFieldKey
	return e.encodeStaticType(v.Type)
}

// encodeCapabilityStaticType encodes CapabilityStaticType as
// cbor.Tag{
//		Number:  cborTagCapabilityStaticType,
//		Content: StaticType(v.BorrowType),
// }
func (e *Encoder) encodeCapabilityStaticType(v CapabilityStaticType) error {
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, cborTagCapabilityStaticType,
	})
	if err != nil {
		return err
	}
	return e.encodeStaticType(v.BorrowType)
}
