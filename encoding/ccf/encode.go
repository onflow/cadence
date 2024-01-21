/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package ccf

import (
	"bytes"
	"fmt"
	"io"
	goRuntime "runtime"
	"sort"
	"sync"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/cadence"
	cadenceErrors "github.com/onflow/cadence/runtime/errors"
)

// defaultCBOREncMode
//
// See https://github.com/fxamacker/cbor:
// "For best performance, reuse EncMode and DecMode after creating them."
var defaultCBOREncMode = func() cbor.EncMode {
	options := cbor.CoreDetEncOptions()
	options.BigIntConvert = cbor.BigIntConvertNone
	encMode, err := options.EncMode()
	if err != nil {
		panic(err)
	}
	return encMode
}()

// An Encoder converts Cadence values into CCF-encoded bytes.
type Encoder struct {
	// CCF codec uses CBOR codec under the hood.
	enc *cbor.StreamEncoder

	// cachedSortedFieldIndex contains sorted field index of Cadence composite types.
	cachedSortedFieldIndex map[string][]int // key: composite type ID, value: sorted field indexes

	// CCF encoding mode containing CCF encoding options
	em *encMode
}

type EncMode interface {
	// Encode returns the CCF-encoded representation of the given value.
	//
	// This function returns an error if the Cadence value cannot be represented in CCF.
	Encode(value cadence.Value) ([]byte, error)

	// MustEncode returns the CCF-encoded representation of the given value, or panics
	// if the value cannot be represented in CCF.
	MustEncode(value cadence.Value) []byte

	// NewEncoder initializes an Encoder that will write CCF-encoded bytes to the
	// given io.Writer.
	NewEncoder(w io.Writer) *Encoder
}

type SortMode int

const (
	// SortNone means no sorting.
	SortNone SortMode = iota

	// SortBytewiseLexical means bytewise lexicographic order.
	SortBytewiseLexical

	maxSortMode
)

func (sm SortMode) valid() bool {
	return sm < maxSortMode
}

// EncOptions specifies CCF encoding options.
type EncOptions struct {
	// SortCompositeFields specifies sort order of Cadence composite fields.
	SortCompositeFields SortMode

	// SortIntersectionTypes specifies sort order of Cadence intersection types.
	SortIntersectionTypes SortMode
}

// EventsEncMode is CCF encoding mode for events which contains
// immutable CCF encoding options.  It is safe for concurrent use.
var EventsEncMode = &encMode{
	sortCompositeFields:   SortNone,
	sortIntersectionTypes: SortNone,
}

type encMode struct {
	sortCompositeFields   SortMode
	sortIntersectionTypes SortMode
}

// EncMode returns CCF encoding mode, which contains immutable encoding options
// and is safe for concurrent use.
func (opts EncOptions) EncMode() (EncMode, error) {
	if !opts.SortCompositeFields.valid() {
		return nil, fmt.Errorf("ccf: invalid SortCompositeFields %d", opts.SortCompositeFields)
	}
	if !opts.SortIntersectionTypes.valid() {
		return nil, fmt.Errorf("ccf: invalid SortIntersectionTypes %d", opts.SortIntersectionTypes)
	}
	return &encMode{
		sortCompositeFields:   opts.SortCompositeFields,
		sortIntersectionTypes: opts.SortIntersectionTypes,
	}, nil
}

// Encode returns the CCF-encoded representation of the given value.
//
// This function returns an error if the Cadence value cannot be represented in CCF.
func (em *encMode) Encode(value cadence.Value) ([]byte, error) {
	var w bytes.Buffer

	enc := em.NewEncoder(&w)
	defer enc.enc.Close()

	err := enc.Encode(value)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MustEncode returns the CCF-encoded representation of the given value, or panics
// if the value cannot be represented in CCF.
func (em *encMode) MustEncode(value cadence.Value) []byte {
	b, err := em.Encode(value)
	if err != nil {
		panic(err)
	}
	return b
}

// NewEncoder initializes an Encoder that will write CCF-encoded bytes to the
// given io.Writer.
func (em *encMode) NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		enc:                    defaultCBOREncMode.NewStreamEncoder(w),
		cachedSortedFieldIndex: make(map[string][]int),
		em:                     em,
	}
}

var defaultEncMode = &encMode{}

// Encode returns the CCF-encoded representation of the given value
// by using default CCF encoding options.  This function returns an
// error if the Cadence value cannot be represented in CCF.
func Encode(value cadence.Value) ([]byte, error) {
	return defaultEncMode.Encode(value)
}

// MustEncode returns the CCF-encoded representation of the given value, or panics
// if the value cannot be represented in CCF.  Default CCF encoding options are used.
func MustEncode(value cadence.Value) []byte {
	return defaultEncMode.MustEncode(value)
}

// NewEncoder initializes an Encoder that will write CCF-encoded bytes to the
// given io.Writer. Default CCF encoding options are used.
func NewEncoder(w io.Writer) *Encoder {
	return defaultEncMode.NewEncoder(w)
}

// Encode writes the CCF-encoded representation of the given value to this
// encoder's io.Writer.
//
// This function returns an error if the given value's type is not supported
// by the encoder.
func (e *Encoder) Encode(value cadence.Value) (err error) {
	// capture panics
	defer func() {
		// Recover panic error if there is any.
		if r := recover(); r != nil {
			// Don't recover Go errors, internal errors, or non-errors.
			switch r := r.(type) {
			case goRuntime.Error, cadenceErrors.InternalError:
				panic(r)
			case error:
				err = r
			default:
				panic(r)
			}
		}

		// Add context to error if there is any.
		if err != nil {
			err = fmt.Errorf(
				"ccf: failed to encode value (type %T, %q): %s",
				value,
				value.Type().ID(),
				err,
			)
		}
	}()

	// Traverse value to find all composite types.
	types, tids := compositeTypesFromValue(value)

	if len(types) == 0 {
		// Encode top level message: ccf-type-and-value-message.
		err = e.encodeTypeAndValue(value, tids)
	} else {
		// Encode top level message: ccf-typedef-and-value-message.
		err = e.encodeTypeDefAndValue(value, types, tids)
	}
	if err != nil {
		return err
	}

	return e.enc.Flush()
}

// encodeTypeDefAndValue encodes type definition and value as
// language=CDDL
// ccf-typedef-and-value-message =
//
//	; cbor-tag-typedef-and-value
//	#6.129([
//	  typedef: composite-typedef,
//	  type-and-value: inline-type-and-value
//	])
func (e *Encoder) encodeTypeDefAndValue(
	value cadence.Value,
	types []cadence.Type,
	tids ccfTypeIDByCadenceType,
) error {
	// Encode tag number cbor-tag-typedef-and-value and array head of length 2.
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagTypeDefAndValue,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// element 0: typedef
	err = e.encodeTypeDefs(types, tids)
	if err != nil {
		return err
	}

	// element 1: type and value
	return e.encodeInlineTypeAndValue(value, tids)
}

// encodeTypeAndValue encodes type and value as
// language=CDDL
// ccf-type-and-value-message =
//
//	; cbor-tag-type-and-value
//	#6.130(inline-type-and-value)
func (e *Encoder) encodeTypeAndValue(value cadence.Value, tids ccfTypeIDByCadenceType) error {
	// Encode tag number
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagTypeAndValue,
	})
	if err != nil {
		return err
	}

	return e.encodeInlineTypeAndValue(value, tids)
}

// encodeTypeAndValueWithNoTag encodes inline type and value as
// language=CDDL
// inline-type-and-value = [
//
//	type: inline-type,
//	value: value,
//
// ]
func (e *Encoder) encodeInlineTypeAndValue(value cadence.Value, tids ccfTypeIDByCadenceType) error {
	// Encode array head of length 2.
	err := e.enc.EncodeRawBytes([]byte{
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	runtimeType := value.Type()

	// element 0: inline-type
	err = e.encodeInlineType(runtimeType, tids)
	if err != nil {
		return err
	}

	// element 1: value
	return e.encodeValue(value, runtimeType, tids)
}

// encodeTypeDefs encodes composite/interface type definitions as
// language=CDDL
// composite-typedef = [
//
//	; one-or-more instead of zero-or-more because:
//	; - when encoding a primitive type, such as boolean or string, `ccf-type-and-value-message` is used (no `composite-typedef` at all)
//	; - when encoding a composite type, such as event, `ccf-typedef-and-value-message` is used, which encodes at least one `composite-typedef`
//	+ (
//	  struct-type
//	  / resource-type
//	  / contract-type
//	  / event-type
//	  / enum-type
//	  / struct-interface-type
//	  / resource-interface-type
//	  / contract-interface-type
//	  )]
func (e *Encoder) encodeTypeDefs(types []cadence.Type, tids ccfTypeIDByCadenceType) error {
	// Encode array head with number of type definitions.
	err := e.enc.EncodeArrayHead(uint64(len(types)))
	if err != nil {
		return err
	}

	for _, typ := range types {

		switch typ := typ.(type) {
		case cadence.CompositeType:
			// Encode struct-type, resource-type, contract-type, event-type, or enum-type.
			err = e.encodeCompositeType(typ, tids)
			if err != nil {
				return err
			}

		case cadence.InterfaceType:
			// Encode struct-interface-type, resource-interface-type, or contract-interface-type.
			err = e.encodeInterfaceType(typ, tids)
			if err != nil {
				return err
			}

		default:
			panic(cadenceErrors.NewUnexpectedError("unexpected type %s in type definition", typ.ID()))
		}
	}

	return nil
}

// encodeValue traverses the object graph of the provided value and encodes it as
// language=CDDL
// value =
//
//	ccf-type-and-value-message
//	/ simple-value
//	/ optional-value
//	/ array-value
//	/ dict-value
//	/ composite-value
//	/ path-value
//	/ path-capability-value
//	/ id-capability-value
//	/ inclusiverange-value
//	/ function-value
//	/ type-value
//
// ccf-type-and-value-message =
//
//	    ; cbor-tag-type-and-value
//	    #6.130([
//	        type: inline-type,
//		value: value
//	    ])
//
// simple-value =
//
//	void-value
//	/ bool-value
//	/ character-value
//	/ string-value
//	/ address-value
//	/ uint-value
//	/ uint8-value
//	/ uint16-value
//	/ uint32-value
//	/ uint64-value
//	/ uint128-value
//	/ uint256-value
//	/ int-value
//	/ int8-value
//	/ int16-value
//	/ int32-value
//	/ int64-value
//	/ int128-value
//	/ int256-value
//	/ word8-value
//	/ word16-value
//	/ word32-value
//	/ word64-value
//	/ word128-value
//	/ word256-value
//	/ fix64-value
//	/ ufix64-value
//
// IMPORTANT:
// "Valid CCF Encoding Requirements" in CCF Specification states:
//
//	"Encoders are not required to check for invalid input items
//	(e.g. invalid UTF-8 strings, duplicate dictionary keys, etc.)
//	Applications MUST NOT provide invalid items to encoders."
//
// cadence.String and cadence.Character must be valid UTF-8
// and it is the application's responsibility to provide
// the CCF encoder with valid UTF-8 strings.
func (e *Encoder) encodeValue(
	v cadence.Value,
	staticType cadence.Type,
	tids ccfTypeIDByCadenceType,
) error {

	if v == nil {
		return e.enc.EncodeNil()
	}

	runtimeType := v.Type()

	// CCF requires value to have non-nil type.
	if runtimeType == nil {
		panic(cadenceErrors.NewUnexpectedError("value (%T) has nil type", v))
	}

	if needToEncodeRuntimeType(staticType, runtimeType) {
		// Get type that needs to be encoded as inline type.
		inlineType := getTypeToEncodeAsCCFInlineType(staticType, runtimeType)

		// Encode ccf-type-and-value-message.

		// Encode tag number and array head of length 2.
		err := e.enc.EncodeRawBytes([]byte{
			// tag number
			0xd8, CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
		})
		if err != nil {
			return err
		}

		// element 0: type as inline-type
		err = e.encodeInlineType(inlineType, tids)
		if err != nil {
			return err
		}

		// element 1: value
	}

	switch v := v.(type) {
	case cadence.Void:
		return e.encodeVoid(v)

	case cadence.Optional:
		return e.encodeOptional(v, tids)

	case cadence.Bool:
		return e.encodeBool(v)

	case cadence.Character:
		return e.encodeCharacter(v)

	case cadence.String:
		return e.encodeString(v)

	case cadence.Address:
		return e.encodeAddress(v)

	case cadence.Int:
		return e.encodeInt(v)

	case cadence.Int8:
		return e.encodeInt8(v)

	case cadence.Int16:
		return e.encodeInt16(v)

	case cadence.Int32:
		return e.encodeInt32(v)

	case cadence.Int64:
		return e.encodeInt64(v)

	case cadence.Int128:
		return e.encodeInt128(v)

	case cadence.Int256:
		return e.encodeInt256(v)

	case cadence.UInt:
		return e.encodeUInt(v)

	case cadence.UInt8:
		return e.encodeUInt8(v)

	case cadence.UInt16:
		return e.encodeUInt16(v)

	case cadence.UInt32:
		return e.encodeUInt32(v)

	case cadence.UInt64:
		return e.encodeUInt64(v)

	case cadence.UInt128:
		return e.encodeUInt128(v)

	case cadence.UInt256:
		return e.encodeUInt256(v)

	case cadence.Word8:
		return e.encodeWord8(v)

	case cadence.Word16:
		return e.encodeWord16(v)

	case cadence.Word32:
		return e.encodeWord32(v)

	case cadence.Word64:
		return e.encodeWord64(v)

	case cadence.Word128:
		return e.encodeWord128(v)

	case cadence.Word256:
		return e.encodeWord256(v)

	case cadence.Fix64:
		return e.encodeFix64(v)

	case cadence.UFix64:
		return e.encodeUFix64(v)

	case cadence.Array:
		return e.encodeArray(v, tids)

	case cadence.Dictionary:
		return e.encodeDictionary(v, tids)

	case *cadence.InclusiveRange:
		return e.encodeInclusiveRange(v, tids)

	case cadence.Struct:
		return e.encodeStruct(v, tids)

	case cadence.Resource:
		return e.encodeResource(v, tids)

	case cadence.Event:
		return e.encodeEvent(v, tids)

	case cadence.Contract:
		return e.encodeContract(v, tids)

	case cadence.Path:
		return e.encodePath(v)

	case cadence.TypeValue:
		// cadence.TypeValue is encoded as self-contained, without any
		// reference to tids.  So tids isn't passed to encodeTypeValue().
		//
		// encodeTypeValue() receives a new ccfTypeIDByCadenceType to deduplicate
		// composite type values within the same CCF type value encoding.
		// For example, when a composite type appears more than once
		// (recursive or repeated as nested type) within the same type value,
		// it is only encoded once and is subsequently represented by its CCF ID.
		// For type value encoding, CCF type ID is sequentially generated by
		// traversal order.
		//
		// If x.StaticType is nil, type value is encoded as nil.
		return e.encodeNullableTypeValue(v.StaticType, ccfTypeIDByCadenceType{})

	case cadence.Capability:
		return e.encodeCapability(v)

	case cadence.Enum:
		return e.encodeEnum(v, tids)

	case cadence.Function:
		// cadence.Function is encoded as self-contained, without any
		// reference to tids.  So tids isn't passed to encodeFunction().
		//
		// encodeFunction() receives a new ccfTypeIDByCadenceType to deduplicate
		// composite type values within the same CCF function value encoding.
		// For example, when a composite type appears more than once
		// (recursive or repeated as nested type) within the same function value,
		// it is only encoded once and is subsequently represented by its CCF ID.
		// For function value encoding, CCF type ID is sequentially generated by
		// traversal order of sorted parameters and return type.
		return e.encodeFunction(v.FunctionType, ccfTypeIDByCadenceType{})

	default:
		panic(cadenceErrors.NewUnexpectedError("cannot encode unsupported value (%T)", v))
	}
}

// encodeVoid encodes cadence.Void as
// language=CDDL
// void-value = nil
func (e *Encoder) encodeVoid(v cadence.Void) error {
	return e.enc.EncodeNil()
}

// encodeOptional encodes cadence.Optional as
// language=CDDL
// optional-value = nil / value
func (e *Encoder) encodeOptional(v cadence.Optional, tids ccfTypeIDByCadenceType) error {
	innerValue := v.Value
	if innerValue == nil {
		return e.enc.EncodeNil()
	}
	// Use innerValue.Type() as static type to avoid encoding type
	// because OptionalType is already encoded.
	return e.encodeValue(innerValue, innerValue.Type(), tids)
}

// encodeBool encodes cadence.Bool as
// language=CDDL
// bool-value = bool
func (e *Encoder) encodeBool(v cadence.Bool) error {
	return e.enc.EncodeBool(bool(v))
}

// encodeCharacter encodes cadence.Character as
// language=CDDL
// character-value = tstr
func (e *Encoder) encodeCharacter(v cadence.Character) error {
	return e.enc.EncodeString(string(v))
}

// encodeString encodes cadence.String as
// language=CDDL
// string-value = tstr
func (e *Encoder) encodeString(v cadence.String) error {
	return e.enc.EncodeString(string(v))
}

// encodeAddress encodes cadence.Address as
// language=CDDL
// address-value = bstr .size 8
func (e *Encoder) encodeAddress(v cadence.Address) error {
	return e.enc.EncodeBytes(v[:])
}

// encodeInt encodes cadence.Int as
// language=CDDL
// int-value = bigint
func (e *Encoder) encodeInt(v cadence.Int) error {
	return e.enc.EncodeBigInt(v.Big())
}

// encodeInt8 encodes cadence.Int8 as
// language=CDDL
// int8-value = (int .ge -128) .le 127
func (e *Encoder) encodeInt8(v cadence.Int8) error {
	return e.enc.EncodeInt8(int8(v))
}

// encodeInt16 encodes cadence.Int16 as
// language=CDDL
// int16-value = (int .ge -32768) .le 32767
func (e *Encoder) encodeInt16(v cadence.Int16) error {
	return e.enc.EncodeInt16(int16(v))
}

// encodeInt32 encodes cadence.Int32 as
// language=CDDL
// int32-value = (int .ge -2147483648) .le 2147483647
func (e *Encoder) encodeInt32(v cadence.Int32) error {
	return e.enc.EncodeInt32(int32(v))
}

// encodeInt64 encodes cadence.Int64 as
// language=CDDL
// int64-value = (int .ge -9223372036854775808) .le 9223372036854775807
func (e *Encoder) encodeInt64(v cadence.Int64) error {
	return e.enc.EncodeInt64(int64(v))
}

// encodeInt128 encodes cadence.Int128 as
// language=CDDL
// int128-value = bigint
func (e *Encoder) encodeInt128(v cadence.Int128) error {
	return e.enc.EncodeBigInt(v.Big())
}

// encodeInt256 encodes cadence.Int256 as
// language=CDDL
// int256-value = bigint
func (e *Encoder) encodeInt256(v cadence.Int256) error {
	return e.enc.EncodeBigInt(v.Big())
}

// encodeUInt encodes cadence.UInt as
// language=CDDL
// uint-value = bigint .ge 0
func (e *Encoder) encodeUInt(v cadence.UInt) error {
	return e.enc.EncodeBigInt(v.Big())
}

// encodeUInt8 encodes cadence.UInt8 as
// language=CDDL
// uint8-value = uint .le 255
func (e *Encoder) encodeUInt8(v cadence.UInt8) error {
	return e.enc.EncodeUint8(uint8(v))
}

// encodeUInt16 encodes cadence.UInt16 as
// language=CDDL
// uint16-value = uint .le 65535
func (e *Encoder) encodeUInt16(v cadence.UInt16) error {
	return e.enc.EncodeUint16(uint16(v))
}

// encodeUInt32 encodes cadence.UInt32 as CBOR uint.
// language=CDDL
// uint32-value = uint .le 4294967295
func (e *Encoder) encodeUInt32(v cadence.UInt32) error {
	return e.enc.EncodeUint32(uint32(v))
}

// encodeUInt64 encodes cadence.UInt64 as
// language=CDDL
// uint64-value = uint .le 18446744073709551615
func (e *Encoder) encodeUInt64(v cadence.UInt64) error {
	return e.enc.EncodeUint64(uint64(v))
}

// encodeUInt128 encodes cadence.UInt128 as
// language=CDDL
// uint128-value = bigint .ge 0
func (e *Encoder) encodeUInt128(v cadence.UInt128) error {
	return e.enc.EncodeBigInt(v.Big())
}

// encodeUInt256 encodes cadence.UInt256 as
// language=CDDL
// uint256-value = bigint .ge 0
func (e *Encoder) encodeUInt256(v cadence.UInt256) error {
	return e.enc.EncodeBigInt(v.Big())
}

// encodeWord8 encodes cadence.Word8 as
// language=CDDL
// word8-value = uint .le 255
func (e *Encoder) encodeWord8(v cadence.Word8) error {
	return e.enc.EncodeUint8(uint8(v))
}

// encodeWord16 encodes cadence.Word16 as
// language=CDDL
// word16-value = uint .le 65535
func (e *Encoder) encodeWord16(v cadence.Word16) error {
	return e.enc.EncodeUint16(uint16(v))
}

// encodeWord32 encodes cadence.Word32 as
// language=CDDL
// word32-value = uint .le 4294967295
func (e *Encoder) encodeWord32(v cadence.Word32) error {
	return e.enc.EncodeUint32(uint32(v))
}

// encodeWord64 encodes cadence.Word64 as
// language=CDDL
// word64-value = uint .le 18446744073709551615
func (e *Encoder) encodeWord64(v cadence.Word64) error {
	return e.enc.EncodeUint64(uint64(v))
}

// encodeWord128 encodes cadence.Word128 as
// language=CDDL
// word128-value = uint .ge 0
func (e *Encoder) encodeWord128(v cadence.Word128) error {
	return e.enc.EncodeBigInt(v.Big())
}

// encodeWord256 encodes cadence.Word256 as
// language=CDDL
// word256-value = uint .ge 0
func (e *Encoder) encodeWord256(v cadence.Word256) error {
	return e.enc.EncodeBigInt(v.Big())
}

// encodeFix64 encodes cadence.Fix64 as
// language=CDDL
// fix64-value = (int .ge -9223372036854775808) .le 9223372036854775807
func (e *Encoder) encodeFix64(v cadence.Fix64) error {
	return e.enc.EncodeInt64(int64(v))
}

// encodeUFix64 encodes cadence.UFix64 as
// language=CDDL
// ufix64-value = uint .le 18446744073709551615
func (e *Encoder) encodeUFix64(v cadence.UFix64) error {
	return e.enc.EncodeUint64(uint64(v))
}

// encodeArray encodes cadence.Array as
// language=CDDL
// array-value = [* value]
func (e *Encoder) encodeArray(v cadence.Array, tids ccfTypeIDByCadenceType) error {
	staticElementType := v.ArrayType.Element()

	// Encode array head with number of array elements.
	err := e.enc.EncodeArrayHead(uint64(len(v.Values)))
	if err != nil {
		return err
	}

	for _, element := range v.Values {
		// Encode element as value.
		err = e.encodeValue(element, staticElementType, tids)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeDictionary encodes cadence.Dictionary as
// language=CDDL
// dict-value = [* (key: value, value: value)]
func (e *Encoder) encodeDictionary(v cadence.Dictionary, tids ccfTypeIDByCadenceType) error {
	if len(v.Pairs) > 1 {
		return e.encodeSortedDictionary(v, tids)
	}

	staticKeyType := v.DictionaryType.KeyType
	staticElementType := v.DictionaryType.ElementType

	// Encode array head with array size of 2 * number of pairs.
	err := e.enc.EncodeArrayHead(uint64(len(v.Pairs)) * 2)
	if err != nil {
		return err
	}

	for _, pair := range v.Pairs {

		// Encode dictionary key as value.
		err = e.encodeValue(pair.Key, staticKeyType, tids)
		if err != nil {
			return err
		}

		// Encode dictionary value as value.
		err = e.encodeValue(pair.Value, staticElementType, tids)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeDictionary encodes cadence.Dictionary as
// language=CDDL
// dict-value = [* (key: value, value: value)]
func (e *Encoder) encodeSortedDictionary(v cadence.Dictionary, tids ccfTypeIDByCadenceType) error {
	// "Deterministic CCF Encoding Requirements" in CCF specs:
	//
	//   "dict-value key-value pairs MUST be sorted by key."

	// Use a new buffer for sorting key value pairs.
	buf := getBuffer()
	defer putBuffer(buf)

	// Encode and sort key value pairs.
	sortedPairs, err := encodeAndSortKeyValuePairs(buf, v, tids, e.em)
	if err != nil {
		return err
	}

	// Encode array head with 2 * number of pairs.
	err = e.enc.EncodeArrayHead(uint64(len(v.Pairs)) * 2)
	if err != nil {
		return err
	}

	for _, pair := range sortedPairs {
		// Encode key and value.
		err = e.enc.EncodeRawBytes(pair.encodedPair)
		if err != nil {
			return err
		}
	}

	return nil
}

func encodeAndSortKeyValuePairs(
	buf *bytes.Buffer,
	v cadence.Dictionary,
	tids ccfTypeIDByCadenceType,
	em *encMode,

) (
	[]encodedKeyValuePair,
	error,
) {
	staticKeyType := v.DictionaryType.KeyType
	staticElementType := v.DictionaryType.ElementType

	encodedPairs := make([]encodedKeyValuePair, len(v.Pairs))

	e := em.NewEncoder(buf)

	for i, pair := range v.Pairs {

		off := buf.Len()

		// Encode dictionary key as value.
		err := e.encodeValue(pair.Key, staticKeyType, tids)
		if err != nil {
			return nil, err
		}

		// Get encoded key length (must flush first).
		e.enc.Flush()
		keyLength := buf.Len() - off

		// Encode dictionary value as value.
		err = e.encodeValue(pair.Value, staticElementType, tids)
		if err != nil {
			return nil, err
		}

		// Get encoded pair length (must flush first).
		e.enc.Flush()
		pairLength := buf.Len() - off

		encodedPairs[i] = encodedKeyValuePair{keyLength: keyLength, pairLength: pairLength}
	}

	// Reslice buf for encoded key and pair by offset and length.
	b := buf.Bytes()
	off := 0
	for i := 0; i < len(encodedPairs); i++ {
		encodedPairs[i].encodedKey = b[off : off+encodedPairs[i].keyLength]
		encodedPairs[i].encodedPair = b[off : off+encodedPairs[i].pairLength]
		off += encodedPairs[i].pairLength
	}
	if off != len(b) {
		// Sanity check
		panic(cadenceErrors.NewUnexpectedError("encoded dictionary pairs' offset %d doesn't match buffer length %d", off, len(b)))
	}

	sort.Sort(bytewiseKeyValuePairSorter(encodedPairs))

	return encodedPairs, nil
}

// encodeInclusiveRange encodes cadence.InclusiveRange as
// language=CDDL
// inclusiverange-value = [3*3 (key: value, value: value)]
func (e *Encoder) encodeInclusiveRange(v *cadence.InclusiveRange, tids ccfTypeIDByCadenceType) error {
	staticElementType := v.InclusiveRangeType.ElementType

	// Encode array head with array size of 3.
	err := e.enc.EncodeArrayHead(3)
	if err != nil {
		return err
	}

	// Encode start key as value.
	err = e.encodeValue(v.Start, staticElementType, tids)
	if err != nil {
		return err
	}

	// Encode end as value.
	err = e.encodeValue(v.End, staticElementType, tids)
	if err != nil {
		return err
	}

	// Encode step key as value.
	return e.encodeValue(v.Step, staticElementType, tids)
}

// encodeStruct encodes cadence.Struct as
// language=CDDL
// composite-value = [* (field: value)]
func (e *Encoder) encodeStruct(v cadence.Struct, tids ccfTypeIDByCadenceType) error {
	return e.encodeComposite(v.StructType, v.Fields, tids)
}

// encodeResource encodes cadence.Resource as
// language=CDDL
// composite-value = [* (field: value)]
func (e *Encoder) encodeResource(v cadence.Resource, tids ccfTypeIDByCadenceType) error {
	return e.encodeComposite(v.ResourceType, v.Fields, tids)
}

// encodeEvent encodes cadence.Event as
// language=CDDL
// composite-value = [* (field: value)]
func (e *Encoder) encodeEvent(v cadence.Event, tids ccfTypeIDByCadenceType) error {
	return e.encodeComposite(v.EventType, v.Fields, tids)
}

// encodeContract encodes cadence.Contract as
// language=CDDL
// composite-value = [* (field: value)]
func (e *Encoder) encodeContract(v cadence.Contract, tids ccfTypeIDByCadenceType) error {
	return e.encodeComposite(v.ContractType, v.Fields, tids)
}

// encodeEnum encodes cadence.Enum as
// language=CDDL
// composite-value = [* (field: value)]
func (e *Encoder) encodeEnum(v cadence.Enum, tids ccfTypeIDByCadenceType) error {
	return e.encodeComposite(v.EnumType, v.Fields, tids)
}

// encodeComposite encodes composite types as
// language=CDDL
// composite-value = [* (field: value)]
func (e *Encoder) encodeComposite(
	typ cadence.CompositeType,
	fields []cadence.Value,
	tids ccfTypeIDByCadenceType,
) error {
	staticFieldTypes := typ.CompositeFields()

	if len(staticFieldTypes) != len(fields) {
		panic(cadenceErrors.NewUnexpectedError(
			"%s field count %d doesn't match declared field type count %d",
			typ.ID(),
			len(fields),
			len(staticFieldTypes),
		))
	}

	// Encode array head with number of fields.
	err := e.enc.EncodeArrayHead(uint64(len(fields)))
	if err != nil {
		return err
	}

	switch e.em.sortCompositeFields {
	case SortNone:
		// Encode fields without sorting.
		for i, field := range fields {
			err = e.encodeValue(field, staticFieldTypes[i].Type, tids)
			if err != nil {
				return err
			}
		}
		return nil

	case SortBytewiseLexical:
		switch len(fields) {
		case 0:
			// Short-circuit if there is no field.
			return nil

		case 1:
			// Avoid overhead of sorting if there is only one field.
			return e.encodeValue(fields[0], staticFieldTypes[0].Type, tids)

		default:
			sortedIndexes := e.getSortedFieldIndex(typ)

			if len(sortedIndexes) != len(staticFieldTypes) {
				panic(cadenceErrors.NewUnexpectedError("number of sorted indexes doesn't match number of field types"))
			}

			for _, index := range sortedIndexes {
				// Encode sorted field as value.
				err = e.encodeValue(fields[index], staticFieldTypes[index].Type, tids)
				if err != nil {
					return err
				}
			}

			return nil
		}

	default:
		panic(cadenceErrors.NewUnexpectedError("unsupported sort option for composite fields: %d", e.em.sortCompositeFields))
	}
}

// encodePath encodes cadence.Path as
// language=CDDL
// path-value = [
//
//	domain: uint,
//	identifier: tstr,
//
// ]
func (e *Encoder) encodePath(x cadence.Path) error {
	// Encode array head with length 2.
	err := e.enc.EncodeRawBytes([]byte{
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// element 0: domain as CBOR uint.
	err = e.enc.EncodeUint8(uint8(x.Domain))
	if err != nil {
		return err
	}

	// element 1: identifier as CBOR tstr.
	return e.enc.EncodeString(x.Identifier)
}

// encodeCapability encodes cadence.Capability as
// language=CDDL
// id-capability-value = [
//
//	address: address-value,
//	id: uint64-value
//
// ]
func (e *Encoder) encodeCapability(capability cadence.Capability) error {
	// Encode array head with length 2.
	err := e.enc.EncodeRawBytes([]byte{
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// element 0: address
	err = e.encodeAddress(capability.Address)
	if err != nil {
		return err
	}

	// element 1: id
	return e.encodeUInt64(capability.ID)
}

// encodeFunction encodes cadence.FunctionType as
// language=CDDL
// function-value = [
//
//	 type-parameters: [
//		* [
//			name: tstr,
//			type-bound: type-value / nil
//		  ]
//	 ]
//	 parameters: [
//		    * [
//		        label: tstr,
//		        identifier: tstr,
//		        type: type-value
//		    ]
//	 ]
//	 return-type: type-value
//
// ]
func (e *Encoder) encodeFunction(typ *cadence.FunctionType, visited ccfTypeIDByCadenceType) error {
	// Encode array head of length 3.
	err := e.enc.EncodeRawBytes([]byte{
		// array, 3 items follow
		0x83,
	})
	if err != nil {
		return err
	}

	// element 0: type parameters as array.
	err = e.encodeTypeParameterTypeValues(typ.TypeParameters, visited)
	if err != nil {
		return err
	}

	// element 1: parameters as array.
	err = e.encodeParameterTypeValues(typ.Parameters, visited)
	if err != nil {
		return err
	}

	// element 2: return type as type-value.
	return e.encodeTypeValue(typ.ReturnType, visited)
}

// encodeTypeValue encodes cadence.Type as
// language=CDDL
// type-value =
//
//	simple-type-value
//	/ optional-type-value
//	/ varsized-array-type-value
//	/ constsized-array-type-value
//	/ dict-type-value
//	/ struct-type-value
//	/ resource-type-value
//	/ contract-type-value
//	/ event-type-value
//	/ enum-type-value
//	/ struct-interface-type-value
//	/ resource-interface-type-value
//	/ contract-interface-type-value
//	/ function-type-value
//	/ reference-type-value
//	/ intersection-type-value
//	/ capability-type-value
//	/ inclusiverange-type-value
//	/ type-value-ref
//
// TypeValue is used differently from inline type or type definition.
// Inline type and type definition are used to decode exported value,
// while TypeValue is exported value, which type is cadence.MetaType.
// Thus, TypeValue can encode more information than type or type definition.
func (e *Encoder) encodeTypeValue(typ cadence.Type, visited ccfTypeIDByCadenceType) error {
	switch typ.(type) {
	case cadence.CompositeType, cadence.InterfaceType:
		cadenceTypeID := typ.ID()
		if ccfID, ok := visited[cadenceTypeID]; ok {
			// Encode visited composite/interface type value
			// using CCF type id for compactness.
			return e.encodeTypeValueRef(ccfID)
		}
		// type value ccf type id is index of visited composite/interface
		// type value depth first.
		visited[cadenceTypeID] = ccfTypeID(len(visited))
	}

	simpleTypeID, ok := simpleTypeIDByType(typ)
	if ok {
		return e.encodeSimpleTypeValue(simpleTypeID)
	}

	switch typ := typ.(type) {
	case *cadence.OptionalType:
		return e.encodeOptionalTypeValue(typ, visited)

	case *cadence.VariableSizedArrayType:
		return e.encodeVarSizedArrayTypeValue(typ, visited)

	case *cadence.ConstantSizedArrayType:
		return e.encodeConstantSizedArrayTypeValue(typ, visited)

	case *cadence.DictionaryType:
		return e.encodeDictTypeValue(typ, visited)

	case *cadence.StructType:
		return e.encodeStructTypeValue(typ, visited)

	case *cadence.ResourceType:
		return e.encodeResourceTypeValue(typ, visited)

	case *cadence.EventType:
		return e.encodeEventTypeValue(typ, visited)

	case *cadence.ContractType:
		return e.encodeContractTypeValue(typ, visited)

	case *cadence.InclusiveRangeType:
		return e.encodeInclusiveRangeTypeValue(typ, visited)

	case *cadence.StructInterfaceType:
		return e.encodeStructInterfaceTypeValue(typ, visited)

	case *cadence.ResourceInterfaceType:
		return e.encodeResourceInterfaceTypeValue(typ, visited)

	case *cadence.ContractInterfaceType:
		return e.encodeContractInterfaceTypeValue(typ, visited)

	case *cadence.FunctionType:
		return e.encodeFunctionTypeValue(typ, visited)

	case *cadence.ReferenceType:
		return e.encodeReferenceTypeValue(typ, visited)

	case *cadence.IntersectionType:
		return e.encodeIntersectionTypeValue(typ, visited)

	case *cadence.CapabilityType:
		return e.encodeCapabilityTypeValue(typ, visited)

	case *cadence.EnumType:
		return e.encodeEnumTypeValue(typ, visited)

	default:
		panic(cadenceErrors.NewUnexpectedError("unsupported type value %s (%T)", typ.ID(), typ))
	}
}

// encodeNullableTypeValue encodes cadence.Type as type-value or nil.
func (e *Encoder) encodeNullableTypeValue(typ cadence.Type, visited ccfTypeIDByCadenceType) error {
	if typ == nil {
		return e.encodeNilTypeValue()
	}
	return e.encodeTypeValue(typ, visited)
}

// encodeTypeValueRef encodes type value ID as
// language=CDDL
// type-value-ref =
//
//	; cbor-tag-type-value-ref
//	#6.184(id)
func (e *Encoder) encodeTypeValueRef(id ccfTypeID) error {
	rawTagNum := []byte{0xd8, CBORTagTypeValueRef}
	return e.encodeTypeRefWithRawTag(id, rawTagNum)
}

// encodeSimpleTypeValue encodes cadence simple type value as
// language=CDDL
// simple-type-value =
//
//	; cbor-tag-simple-type-value
//	#6.185(simple-type-id)
func (e *Encoder) encodeSimpleTypeValue(id SimpleType) error {
	rawTagNum := []byte{0xd8, CBORTagSimpleTypeValue}
	return e.encodeSimpleTypeWithRawTag(uint64(id), rawTagNum)
}

// encodeOptionalTypeValue encodes cadence.OptionalType as
// language=CDDL
// optional-type-value =
//
//	; cbor-tag-optional-type-value
//	#6.186(type-value)
func (e *Encoder) encodeOptionalTypeValue(typ *cadence.OptionalType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagOptionalTypeValue}
	return e.encodeOptionalTypeWithRawTag(
		typ,
		visited,
		e.encodeTypeValue,
		rawTagNum,
	)
}

// encodeVarSizedArrayTypeValue encodes cadence.VariableSizedArrayType as
// language=CDDL
// varsized-array-type-value =
//
//	; cbor-tag-varsized-array-type-value
//	#6.187(type-value)
func (e *Encoder) encodeVarSizedArrayTypeValue(typ *cadence.VariableSizedArrayType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagVarsizedArrayTypeValue}
	return e.encodeVarSizedArrayTypeWithRawTag(
		typ,
		visited,
		e.encodeTypeValue,
		rawTagNum,
	)
}

// encodeConstantSizedArrayTypeValue encodes cadence.ConstantSizedArrayType as
// language=CDDL
// constsized-array-type-value =
//
//	; cbor-tag-constsized-array-type-value
//	#6.188([
//	    array-size: uint,
//	    element-type: type-value
//	])
func (e *Encoder) encodeConstantSizedArrayTypeValue(typ *cadence.ConstantSizedArrayType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagConstsizedArrayTypeValue}
	return e.encodeConstantSizedArrayTypeWithRawTag(
		typ,
		visited,
		e.encodeTypeValue,
		rawTagNum,
	)
}

// encodeDictTypeValue encodes cadence.DictionaryType as
// language=CDDL
// dict-type-value =
//
//	; cbor-tag-dict-type-value
//	#6.189([
//	    key-type: type-value,
//	    element-type: type-value
//	])
func (e *Encoder) encodeDictTypeValue(typ *cadence.DictionaryType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagDictTypeValue}
	return e.encodeDictTypeWithRawTag(
		typ,
		visited,
		e.encodeTypeValue,
		rawTagNum,
	)
}

// encodeInclusiveRangeTypeValue encodes cadence.InclusiveRangeType as
// language=CDDL
// inclusiverange-type-value =
//
//	; cbor-tag-inclusiverange-type-value
//	#6.194(type-value)
func (e *Encoder) encodeInclusiveRangeTypeValue(typ *cadence.InclusiveRangeType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagInclusiveRangeTypeValue}
	return e.encodeInclusiveRangeTypeWithRawTag(
		typ,
		visited,
		e.encodeTypeValue,
		rawTagNum,
	)
}

// encodeReferenceTypeValue encodes cadence.ReferenceType as
// language=CDDL
// reference-type-value =
//
//	; cbor-tag-reference-type-value
//	#6.190([
//	  authorized: bool,
//	  type: type-value,
//	])
func (e *Encoder) encodeReferenceTypeValue(typ *cadence.ReferenceType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagReferenceTypeValue}
	return e.encodeReferenceTypeWithRawTag(
		typ,
		visited,
		e.encodeTypeValue,
		rawTagNum,
	)
}

// encodeIntersectionTypeValue encodes cadence.IntersectionType as
// language=CDDL
// intersection-type-value =
//
//	; cbor-tag-intersection-type-value
//	#6.191([
//	  type: type-value / nil,
//	  types: [* type-value]
//	])
func (e *Encoder) encodeIntersectionTypeValue(typ *cadence.IntersectionType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagIntersectionTypeValue}
	return e.encodeIntersectionTypeWithRawTag(
		typ,
		visited,
		e.encodeNullableTypeValue,
		e.encodeTypeValue,
		rawTagNum,
	)
}

// encodeCapabilityTypeValue encodes cadence.CapabilityType as
// language=CDDL
// capability-type-value =
//
//	; cbor-tag-capability-type-value
//	; use an array as an extension point
//	#6.192([
//	  ; borrow-type
//	  type-value / nil
//	])
func (e *Encoder) encodeCapabilityTypeValue(typ *cadence.CapabilityType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagCapabilityTypeValue}
	return e.encodeCapabilityTypeWithRawTag(
		typ,
		visited,
		e.encodeNullableTypeValue,
		rawTagNum,
	)
}

// encodeStructTypeValue encodes cadence.StructType as
// language=CDDL
// struct-type-value =
//
//	; cbor-tag-struct-type-value
//	#6.208(composite-type-value)
func (e *Encoder) encodeStructTypeValue(typ *cadence.StructType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagStructTypeValue}
	return e.encodeCompositeTypeValue(
		typ.ID(),
		nil,
		typ.Fields,
		typ.Initializers,
		visited,
		rawTagNum,
	)
}

// encodeResourceTypeValue encodes cadence.ResourceType as
// language=CDDL
// resource-type-value =
//
//	; cbor-tag-resource-type-value
//	#6.209(composite-type-value)
func (e *Encoder) encodeResourceTypeValue(typ *cadence.ResourceType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagResourceTypeValue}
	return e.encodeCompositeTypeValue(
		typ.ID(),
		nil,
		typ.Fields,
		typ.Initializers,
		visited,
		rawTagNum,
	)
}

// encodeEventTypeValue encodes cadence.EventType as
// language=CDDL
// event-type-value =
//
//	; cbor-tag-event-type-value
//	#6.210(composite-type-value)
func (e *Encoder) encodeEventTypeValue(typ *cadence.EventType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagEventTypeValue}
	return e.encodeCompositeTypeValue(
		typ.ID(),
		nil,
		typ.Fields,
		[][]cadence.Parameter{typ.Initializer},
		visited,
		rawTagNum,
	)
}

// encodeContractTypeValue encodes cadence.ContractType as
// language=CDDL
// contract-type-value =
//
//	; cbor-tag-contract-type-value
//	#6.211(composite-type-value)
func (e *Encoder) encodeContractTypeValue(typ *cadence.ContractType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagContractTypeValue}
	return e.encodeCompositeTypeValue(
		typ.ID(),
		nil,
		typ.Fields,
		typ.Initializers,
		visited,
		rawTagNum,
	)
}

// encodeEnumTypeValue encodes cadence.EnumType as
// language=CDDL
// enum-type-value =
//
//	; cbor-tag-enum-type-value
//	#6.212(composite-type-value)
func (e *Encoder) encodeEnumTypeValue(typ *cadence.EnumType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagEnumTypeValue}
	return e.encodeCompositeTypeValue(
		typ.ID(),
		typ.RawType,
		typ.Fields,
		typ.Initializers,
		visited,
		rawTagNum,
	)
}

// encodeStructInterfaceTypeValue encodes cadence.StructInterfaceType as
// language=CDDL
// struct-interface-type-value =
//
//	; cbor-tag-struct-interface-type-value
//	#6.224(composite-type-value)
func (e *Encoder) encodeStructInterfaceTypeValue(typ *cadence.StructInterfaceType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagStructInterfaceTypeValue}
	return e.encodeCompositeTypeValue(
		typ.ID(),
		nil,
		typ.Fields,
		typ.Initializers,
		visited,
		rawTagNum,
	)
}

// encodeResourceInterfaceTypeValue encodes cadence.ResourceInterfaceType as
// language=CDDL
// resource-interface-type-value =
//
//	; cbor-tag-resource-interface-type-value
//	#6.225(composite-type-value)
func (e *Encoder) encodeResourceInterfaceTypeValue(typ *cadence.ResourceInterfaceType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagResourceInterfaceTypeValue}
	return e.encodeCompositeTypeValue(
		typ.ID(),
		nil,
		typ.Fields,
		typ.Initializers,
		visited,
		rawTagNum,
	)
}

// encodeContractInterfaceTypeValue encodes cadence.ContractInterfaceType as
// language=CDDL
// contract-interface-type-value =
//
//	; cbor-tag-contract-interface-type-value
//	#6.226(composite-type-value)
func (e *Encoder) encodeContractInterfaceTypeValue(typ *cadence.ContractInterfaceType, visited ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagContractInterfaceTypeValue}
	return e.encodeCompositeTypeValue(
		typ.ID(),
		nil,
		typ.Fields,
		typ.Initializers,
		visited,
		rawTagNum,
	)
}

// encodeCompositeTypeValue encodes composite and interface type values as
// language=CDDL
// composite-type-value = [
//
//	id: id,
//	cadence-type-id: cadence-type-id,
//	; type is only used by enum type value
//	type: nil / type-value,
//	fields: [
//	    * [
//	        name: tstr,
//	        type: type-value
//	    ]
//	]
//	initializers: [
//	    ? [
//	        * [
//	            label: tstr,
//	            identifier: tstr,
//	            type: type-value
//	        ]
//	    ]
//	]
//
// ]
func (e *Encoder) encodeCompositeTypeValue(
	cadenceTypeID string,
	typ cadence.Type,
	fieldTypes []cadence.Field,
	initializerTypes [][]cadence.Parameter,
	visited ccfTypeIDByCadenceType,
	rawTagNum []byte,
) error {
	ccfID, ok := visited[cadenceTypeID]
	if !ok {
		panic(cadenceErrors.NewUnexpectedError("CCF type ID not found for composite type value %s", cadenceTypeID))
	}

	// Encode given tag number indicating cadence type value.
	err := e.enc.EncodeRawBytes(rawTagNum)
	if err != nil {
		return err
	}

	// Encode CBOR array head with length 5.
	err = e.enc.EncodeArrayHead(5)
	if err != nil {
		return err
	}

	// element 0: ccf type id as bstr.
	// It is used to lookup repeated or recursive types within the same encoded type value.
	err = e.encodeCCFTypeID(ccfID)
	if err != nil {
		return err
	}

	// element 1: cadence type id as tstr.
	err = e.encodeCadenceTypeID(cadenceTypeID)
	if err != nil {
		return err
	}

	// element 2: type as nil or type-value.
	// Type is only used by enum type value.
	err = e.encodeNullableTypeValue(typ, visited)
	if err != nil {
		return err
	}

	// element 3: fields as array.
	err = e.encodeFieldTypeValues(fieldTypes, visited)
	if err != nil {
		return err
	}

	// element 4: initializers as array.
	return e.encodeInitializerTypeValues(initializerTypes, visited)
}

// encodeFieldTypeValues encodes composite field types as
// language=CDDL
//
//	fields: [
//	    * [
//	        name: tstr,
//	        type: type-value
//	    ]
//	]
func (e *Encoder) encodeFieldTypeValues(fieldTypes []cadence.Field, visited ccfTypeIDByCadenceType) error {
	// Encode CBOR array head with number of field types.
	err := e.enc.EncodeArrayHead(uint64(len(fieldTypes)))
	if err != nil {
		return err
	}

	switch e.em.sortCompositeFields {
	case SortNone:
		// Encode fields without sorting.
		for _, fieldType := range fieldTypes {
			err = e.encodeFieldTypeValue(fieldType, visited)
			if err != nil {
				return err
			}
		}
		return nil

	case SortBytewiseLexical:
		switch len(fieldTypes) {
		case 0:
			// Short-circuit if there is no field type.
			return nil

		case 1:
			// Avoid overhead of sorting if there is only one field type.
			return e.encodeFieldTypeValue(fieldTypes[0], visited)

		default:
			// "Deterministic CCF Encoding Requirements" in CCF specs:
			//
			//   "composite-type-value.fields MUST be sorted by name."

			// NOTE: bytewiseFieldIdentifierSorter doesn't sort fieldTypes in place.
			// bytewiseFieldIdentifierSorter.indexes is used as sorted fieldTypes
			// index.
			sorter := newBytewiseFieldSorter(fieldTypes)

			sort.Sort(sorter)

			// Encode sorted field types.
			for _, index := range sorter.indexes {
				err = e.encodeFieldTypeValue(fieldTypes[index], visited)
				if err != nil {
					return err
				}
			}

			return nil
		}

	default:
		panic(cadenceErrors.NewUnexpectedError("unsupported sort option for composite field type values: %d", e.em.sortCompositeFields))
	}
}

// encodeFieldTypeValue encodes one composite field type as
// language=CDDL
//
//	[
//	    name: tstr,
//	    type: type-value
//	]
func (e *Encoder) encodeFieldTypeValue(fieldType cadence.Field, visited ccfTypeIDByCadenceType) error {
	// Encode CBOR array head with length 2.
	err := e.enc.EncodeRawBytes([]byte{
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// element 0: field name as tstr.
	err = e.enc.EncodeString(fieldType.Identifier)
	if err != nil {
		return err
	}

	// element 1: field type as type-value.
	return e.encodeTypeValue(fieldType.Type, visited)
}

// encodeInitializerTypeValues encodes composite initializers as
// language=CDDL
//
//	initializers: [
//	    ? [
//	        * [
//	            label: tstr,
//	            identifier: tstr,
//	            type: type-value
//	        ]
//	    ]
//	]
func (e *Encoder) encodeInitializerTypeValues(initializerTypes [][]cadence.Parameter, visited ccfTypeIDByCadenceType) error {
	if len(initializerTypes) > 1 {
		return fmt.Errorf("got %d initializers, want 0 or 1 initializer", len(initializerTypes))
	}

	// Encode CBOR array head with number of initializers.
	err := e.enc.EncodeArrayHead(uint64(len(initializerTypes)))
	if err != nil {
		return err
	}

	// Encode initializers.
	for _, params := range initializerTypes {
		err = e.encodeParameterTypeValues(params, visited)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeTypeParameterTypeValues encodes type parameters as
// language=CDDL
//
// type-parameters: [
//
//	*[
//	  name: tstr,
//	  type-bound: type-value / nil
//	  ]
//
// ]
func (e *Encoder) encodeTypeParameterTypeValues(typeParameters []cadence.TypeParameter, visited ccfTypeIDByCadenceType) error {
	// Encode CBOR array head with number of type parameters.
	err := e.enc.EncodeArrayHead(uint64(len(typeParameters)))
	if err != nil {
		return err
	}

	// Encode type parameters.
	for _, param := range typeParameters {
		err = e.encodeTypeParameterTypeValue(param, visited)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeTypeParameterTypeValue encodes type parameter as
// language=CDDL
//
//	[
//	     name: tstr,
//	     type-bound: type-value / nil
//	]
func (e *Encoder) encodeTypeParameterTypeValue(typeParameter cadence.TypeParameter, visited ccfTypeIDByCadenceType) error {
	// Encode CBOR array head with length 2
	err := e.enc.EncodeRawBytes([]byte{
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// element 0: name as tstr.
	err = e.enc.EncodeString(typeParameter.Name)
	if err != nil {
		return err
	}

	// element 1: type as type-bound.
	return e.encodeNullableTypeValue(typeParameter.TypeBound, visited)
}

// encodeParameterTypeValues encodes composite initializer parameter types as
// language=CDDL
//
//	 [
//	    * [
//	        label: tstr,
//	        identifier: tstr,
//	        type: type-value
//	    ]
//	]
func (e *Encoder) encodeParameterTypeValues(parameterTypes []cadence.Parameter, visited ccfTypeIDByCadenceType) error {
	// Encode CBOR array head with number of parameter types.
	err := e.enc.EncodeArrayHead(uint64(len(parameterTypes)))
	if err != nil {
		return err
	}

	// Encode parameter types.
	for _, param := range parameterTypes {
		err = e.encodeParameterTypeValue(param, visited)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeParameterTypeValue encodes composite initializer parameter as
// language=CDDL
//
//	 [
//	    label: tstr,
//	    identifier: tstr,
//	    type: type-value
//	]
func (e *Encoder) encodeParameterTypeValue(parameterType cadence.Parameter, visited ccfTypeIDByCadenceType) error {
	// Encode CBOR array head with length 3
	err := e.enc.EncodeRawBytes([]byte{
		// array, 3 items follow
		0x83,
	})
	if err != nil {
		return err
	}

	// element 0: label as tstr.
	err = e.enc.EncodeString(parameterType.Label)
	if err != nil {
		return err
	}

	// element 1: identifier as tstr.
	err = e.enc.EncodeString(parameterType.Identifier)
	if err != nil {
		return err
	}

	// element 2: type as type-value.
	return e.encodeTypeValue(parameterType.Type, visited)
}

// encodeFunctionTypeValue encodes cadence.FunctionType as
// language=CDDL
// function-type-value =
//
//	; cbor-tag-function-type-value
//	#6.193(function-value)
func (e *Encoder) encodeFunctionTypeValue(typ *cadence.FunctionType, visited ccfTypeIDByCadenceType) error {
	// Encode tag number and array head of length 3.
	err := e.enc.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagFunctionTypeValue,
	})
	if err != nil {
		return err
	}

	// Encode function-value.
	return e.encodeFunction(typ, visited)
}

// encodeNilTypeValue encodes nil type value as CBOR nil.
func (e *Encoder) encodeNilTypeValue() error {
	return e.enc.EncodeNil()
}

// needToEncodeRuntimeType returns true if runtimeType needs to be encoded because:
// - static type is missing (top level value doesn't have static type)
// - static type is different from runtime type (static type is abstract type)
func needToEncodeRuntimeType(staticType cadence.Type, runtimeType cadence.Type) bool {
	if staticType == nil {
		return true
	}
	if staticType.Equal(runtimeType) {
		return false
	}

	// Here, static type is different from runtime type.

	switch staticType := staticType.(type) {
	case *cadence.OptionalType:
		// Handle special case of runtime type being OptionalType{NeverType}.
		// We handle special case of Optional{nil} because its runtime type is OptionalType{NeverType}
		// while its static type can be different, such as OptionalType{AddressType}.
		// For example, TokensDeposited event is defined as `TokensDeposited(amount: UFix64, to: Address?)`,
		// field to's type is OptionalType{AddressType} and its value can be nil with runtime type
		// OptionalType{NeverType}. Even though runtime type is different from static type (field type),
		// encoder encodes nil value without encoding its runtime type.
		if isOptionalNeverType(runtimeType) {
			return false
		}

		// If both staticType and runtimeType are optional types, check again
		// with unwrapped inner types.  For example, runtimeType shouldn't be
		// encoded if staticType is optional reference to string type and
		// runtimeType is optional string type.  After unwrapping optional
		// types, needToEncodeRuntimeType returns false because staticType is
		// reference to string type and runtimeType is string type.
		if or, ok := runtimeType.(*cadence.OptionalType); ok {
			return needToEncodeRuntimeType(staticType.Type, or.Type)
		}

	case *cadence.ReferenceType:
		// Handle special case of static type being ReferenceType.
		// Encoder doesn't need to encode runtime type if runtime type is the deferenced type of static type.
		return needToEncodeRuntimeType(staticType.Type, runtimeType)
	}

	return true
}

// getTypeToEncodeAsCCFInlineType returns runtime type to be encoded after
// removing redundant type info that is present in staticType because
// staticType is already encoded at higher level.
func getTypeToEncodeAsCCFInlineType(staticType cadence.Type, runtimeType cadence.Type) cadence.Type {
	for {
		switch st := staticType.(type) {
		case *cadence.OptionalType:
			rot, ok := runtimeType.(*cadence.OptionalType)
			if !ok {
				// staticType is optional type while runtime type isn't.
				panic(cadenceErrors.NewUnexpectedError("static type (%T) is optional type while runtime type (%T) isn't", staticType, runtimeType))
			}

			// Unwrap optional type container from both staticType and runtimeType.
			// Static type is encoded at higher level, so encoded runtime type shouldn't repeat
			// the same info. Here, inline type is runtime type after unwrapping optional type
			// that is present in both static and runtime types.
			staticType = st.Type
			runtimeType = rot.Type

		case *cadence.ReferenceType:
			// Unwrap reference type from static type and try again.
			staticType = st.Type

		default:
			return runtimeType
		}
	}
}

// isOptionalNeverType returns true if t is (nested) optional never type.
func isOptionalNeverType(t cadence.Type) bool {
	for {
		ot, ok := t.(*cadence.OptionalType)
		if !ok {
			return false
		}

		if ot.Type.Equal(cadence.NeverType) {
			return true
		}

		t = ot.Type
	}
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		e := new(bytes.Buffer)
		e.Grow(64)
		return e
	},
}

func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func putBuffer(e *bytes.Buffer) {
	e.Reset()
	bufferPool.Put(e)
}

func (e *Encoder) getSortedFieldIndex(t cadence.CompositeType) []int {
	cadenceTypeID := t.ID()

	if indexes, ok := e.cachedSortedFieldIndex[cadenceTypeID]; ok {
		return indexes
	}

	// NOTE: bytewiseFieldIdentifierSorter doesn't sort fields in place.
	// bytewiseFieldIdentifierSorter.indexes is used as sorted fieldTypes
	// index.
	sorter := newBytewiseFieldSorter(t.CompositeFields())

	sort.Sort(sorter)

	e.cachedSortedFieldIndex[cadenceTypeID] = sorter.indexes
	return sorter.indexes
}
