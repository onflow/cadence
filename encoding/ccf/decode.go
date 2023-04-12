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
	"errors"
	"fmt"
	"math"
	"math/big"
	goRuntime "runtime"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	cadenceErrors "github.com/onflow/cadence/runtime/errors"
)

// CBORDecMode
//
// See https://github.com/fxamacker/cbor:
// "For best performance, reuse EncMode and DecMode after creating them."
//
// Security Considerations in Section 10 of RFC 8949 states:
//
//	"Hostile input may be constructed to overrun buffers, to overflow or underflow integer arithmetic,
//	or to cause other decoding disruption. CBOR data items might have lengths or sizes that are
//	intentionally extremely large or too short. Resource exhaustion attacks might attempt to lure a
//	decoder into allocating very big data items (strings, arrays, maps, or even arbitrary precision numbers)
//	or exhaust the stack depth by setting up deeply nested items. Decoders need to have appropriate resource
//	management to mitigate these attacks."
var CBORDecMode = func() cbor.DecMode {
	decMode, err := cbor.DecOptions{
		IndefLength:      cbor.IndefLengthForbidden,
		IntDec:           cbor.IntDecConvertNone,
		MaxArrayElements: 20_000_000, // 20 MB is current grpc size limit so this is more than enough
		MaxMapPairs:      20_000_000, // 20 MB is current grpc size limit so this is more than enough
		MaxNestedLevels:  math.MaxInt16,
	}.DecMode()
	if err != nil {
		panic(err)
	}
	return decMode
}()

// Decoder decodes CCF-encoded representations of Cadence values.
// Since CBOR security considerations apply to CCF, the CBOR
// codec used by CCF Decoder uses limits (e.g. MaxArrayElements,
// MaxMapPairs, MaxNestedLevels) specified by CBORDecMode.
type Decoder struct {
	// CCF codec uses CBOR codec under the hood.
	dec   *cbor.StreamDecoder
	gauge common.MemoryGauge
}

// Decode returns a Cadence value decoded from its CCF-encoded representation.
//
// This function returns an error if the bytes represent CCF that is malformed,
// invalid, or does not comply with requirements in the CCF specification.
func Decode(gauge common.MemoryGauge, b []byte) (cadence.Value, error) {
	dec := NewDecoder(gauge, b)

	v, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	if dec.dec.NumBytesDecoded() != len(b) {
		return nil, cadenceErrors.NewDefaultUserError("ccf: failed to decode: decoded %d bytes, received %d bytes", dec.dec.NumBytesDecoded(), len(b))
	}

	return v, nil
}

// NewDecoder initializes a Decoder that will decode CCF-encoded bytes from the
// given bytes.
func NewDecoder(gauge common.MemoryGauge, b []byte) *Decoder {
	// NOTE: encoded data is not copied by decoder.
	// CCF codec uses CBOR codec under the hood.
	return &Decoder{
		dec:   CBORDecMode.NewByteStreamDecoder(b),
		gauge: gauge,
	}
}

// Decode reads CCF-encoded bytes and decodes them to a Cadence value.
//
// This function returns an error if the bytes represent CCF that is malformed,
// invalid, or does not comply with requirements in the CCF specification.
func (d *Decoder) Decode() (value cadence.Value, err error) {
	// Capture panics that occur during decoding.
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
			err = cadenceErrors.NewDefaultUserError("ccf: failed to decode: %s", err)
		}
	}()

	// Decode top level message.
	tagNum, err := d.dec.DecodeTagNumber()
	if err != nil {
		return nil, err
	}

	switch tagNum {
	case CBORTagTypeDefAndValue:
		// Decode ccf-typedef-and-value-message.
		return d.decodeTypeDefAndValue()

	case CBORTagTypeAndValue:
		// Decode ccf-type-and-value-message.
		return d.decodeTypeAndValue(newCadenceTypeByCCFTypeID())

	default:
		return nil, fmt.Errorf(
			"unsupported top level CCF message with CBOR tag number %d",
			tagNum,
		)
	}
}

// decodeTypeDefAndValue decodes encoded ccf-typedef-and-value-message
// without tag number as
// language=CDDL
// ccf-typedef-and-value-message =
//
//	; cbor-tag-typedef-and-value
//	#6.129([
//	  typedef: composite-typedef,
//	  type-and-value: inline-type-and-value
//	])
func (d *Decoder) decodeTypeDefAndValue() (cadence.Value, error) {
	// Decode array head of length 2
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return nil, err
	}

	// element 0: typedef
	types, err := d.decodeTypeDefs()
	if err != nil {
		return nil, err
	}

	// element 1: type and value
	val, err := d.decodeTypeAndValue(types)
	if err != nil {
		return nil, err
	}

	// Check if there is any unreferenced type definition.
	if types.hasUnreferenced() {
		return nil, errors.New("found unreferenced type definition")
	}

	return val, nil
}

// decodeTypeAndValue decodes encoded ccf-type-and-value-message
// without tag number as
// language=CDDL
// ccf-type-and-value-message =
//
//	; cbor-tag-type-and-value
//	#6.130(inline-type-and-value)
//
// inline-type-and-value = [
//
//	type: inline-type,
//	value: value,
//
// ]
func (d *Decoder) decodeTypeAndValue(types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	// Decode array head of length 2.
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return nil, err
	}

	// element 0: inline-type
	t, err := d.decodeInlineType(types)
	if err != nil {
		return nil, err
	}

	// element 1: value
	return d.decodeValue(t, types)
}

// decodeValue decodes encoded value of type t.
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
//	/ capability-value
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
//	/ fix64-value
//	/ ufix64-value
func (d *Decoder) decodeValue(t cadence.Type, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	if t == nil {
		return nil, fmt.Errorf("unexpected nil type")
	}

	// "Deterministic CCF Encoding Requirements" in CCF specs:
	//
	//   'inline-type-and-value MUST NOT be used when type can be omitted as described
	//    in "Cadence Types and Values Encoding".'
	// If type t for the value to be decoded is a concrete type (e.g. IntType),
	// value MUST NOT be ccf-type-and-value-message.

	switch t := t.(type) {
	case cadence.VoidType:
		return d.decodeVoid()

	case *cadence.OptionalType:
		return d.decodeOptional(t, types)

	case cadence.BoolType:
		return d.decodeBool()

	case cadence.CharacterType:
		return d.decodeCharacter()

	case cadence.StringType:
		return d.decodeString()

	case cadence.AddressType:
		return d.decodeAddress()

	case cadence.IntType:
		return d.decodeInt()

	case cadence.Int8Type:
		return d.decodeInt8()

	case cadence.Int16Type:
		return d.decodeInt16()

	case cadence.Int32Type:
		return d.decodeInt32()

	case cadence.Int64Type:
		return d.decodeInt64()

	case cadence.Int128Type:
		return d.decodeInt128()

	case cadence.Int256Type:
		return d.decodeInt256()

	case cadence.UIntType:
		return d.decodeUInt()

	case cadence.UInt8Type:
		return d.decodeUInt8()

	case cadence.UInt16Type:
		return d.decodeUInt16()

	case cadence.UInt32Type:
		return d.decodeUInt32()

	case cadence.UInt64Type:
		return d.decodeUInt64()

	case cadence.UInt128Type:
		return d.decodeUInt128()

	case cadence.UInt256Type:
		return d.decodeUInt256()

	case cadence.Word8Type:
		return d.decodeWord8()

	case cadence.Word16Type:
		return d.decodeWord16()

	case cadence.Word32Type:
		return d.decodeWord32()

	case cadence.Word64Type:
		return d.decodeWord64()

	case cadence.Fix64Type:
		return d.decodeFix64()

	case cadence.UFix64Type:
		return d.decodeUFix64()

	case *cadence.VariableSizedArrayType:
		return d.decodeArray(t, false, 0, types)

	case *cadence.ConstantSizedArrayType:
		return d.decodeArray(t, true, uint64(t.Size), types)

	case *cadence.DictionaryType:
		return d.decodeDictionary(t, types)

	case *cadence.ResourceType:
		return d.decodeResource(t, types)

	case *cadence.StructType:
		return d.decodeStruct(t, types)

	case *cadence.EventType:
		return d.decodeEvent(t, types)

	case *cadence.ContractType:
		return d.decodeContract(t, types)

	case cadence.StoragePathType:
		return d.decodePath()

	case cadence.PublicPathType:
		return d.decodePath()

	case cadence.PrivatePathType:
		return d.decodePath()

	case cadence.MetaType:
		typeValue, err := d.decodeNullableTypeValue(newCadenceTypeByCCFTypeID())
		if err != nil {
			return nil, err
		}
		return cadence.NewMeteredTypeValue(d.gauge, typeValue), nil

	case *cadence.CapabilityType:
		return d.decodeCapability(t, types)

	case *cadence.EnumType:
		return d.decodeEnum(t, types)

	case *cadence.ReferenceType:
		// When static type is a reference type, encoded value is its deferenced type.
		return d.decodeValue(t.Type, types)

	default:
		err := decodeCBORTagWithKnownNumber(d.dec, CBORTagTypeAndValue)
		if err != nil {
			return nil, fmt.Errorf("unexpected encoded value of Cadence type %s (%T): %s", t.ID(), t, err.Error())
		}

		// Decode ccf-type-and-value-message.
		return d.decodeTypeAndValue(types)
	}
}

// decodeVoid decodes encoded void-value as
// language=CDDL
// void-value = nil
func (d *Decoder) decodeVoid() (cadence.Value, error) {
	err := d.dec.DecodeNil()
	if err != nil {
		return nil, err
	}
	return cadence.NewMeteredVoid(d.gauge), nil
}

// decodeBool decodes encoded bool-value as
// language=CDDL
// bool-value = bool
func (d *Decoder) decodeBool() (cadence.Value, error) {
	b, err := d.dec.DecodeBool()
	if err != nil {
		return nil, err
	}
	return cadence.NewMeteredBool(d.gauge, b), nil
}

// decodeCharacter decodes encoded character-value as
// language=CDDL
// character-value = tstr
func (d *Decoder) decodeCharacter() (cadence.Value, error) {
	s, err := d.dec.DecodeString()
	if err != nil {
		return nil, err
	}
	return cadence.NewMeteredCharacter(
		d.gauge,
		common.NewCadenceCharacterMemoryUsage(len(s)),
		func() string {
			return s
		})
}

// decodeString decodes encoded string-value as
// language=CDDL
// string-value = tstr
// NOTE: invalid UTF-8 is rejected.
func (d *Decoder) decodeString() (cadence.Value, error) {
	s, err := d.dec.DecodeString()
	if err != nil {
		return nil, err
	}

	return cadence.NewMeteredString(
		d.gauge,
		common.NewCadenceStringMemoryUsage(len(s)),
		func() string {
			return s
		},
	)
}

// decodeAddress decodes address-value as
// language=CDDL
// address-value = bstr .size 8
func (d *Decoder) decodeAddress() (cadence.Value, error) {
	b, err := d.dec.DecodeBytes()
	if err != nil {
		return nil, err
	}
	if len(b) != 8 {
		return nil, fmt.Errorf("encoded address-value has length %d (expected 8 bytes)", len(b))
	}
	return cadence.BytesToMeteredAddress(d.gauge, b), nil
}

// decodeInt decodes int-value as
// language=CDDL
// int-value = bigint
func (d *Decoder) decodeInt() (cadence.Value, error) {
	bigInt, err := d.dec.DecodeBigInt()
	if err != nil {
		return nil, err
	}

	return cadence.NewMeteredIntFromBig(
		d.gauge,
		common.NewCadenceIntMemoryUsage(
			common.BigIntByteLength(bigInt),
		),
		func() *big.Int {
			return bigInt
		},
	), nil
}

// decodeInt8 decodes int8-value as
// language=CDDL
// int8-value = (int .ge -128) .le 127
func (d *Decoder) decodeInt8() (cadence.Value, error) {
	i, err := d.dec.DecodeInt64()
	if err != nil {
		return nil, err
	}
	if i < math.MinInt8 || i > math.MaxInt8 {
		return nil, fmt.Errorf(
			"encoded int8-value %d is outside range of Int8 [%d, %d]",
			i,
			math.MinInt8,
			math.MaxInt8,
		)
	}
	return cadence.NewMeteredInt8(d.gauge, int8(i)), nil
}

// decodeInt16 decodes int16-value as
// language=CDDL
// int16-value = (int .ge -32768) .le 32767
func (d *Decoder) decodeInt16() (cadence.Value, error) {
	i, err := d.dec.DecodeInt64()
	if err != nil {
		return nil, err
	}
	if i < math.MinInt16 || i > math.MaxInt16 {
		return nil, fmt.Errorf(
			"encoded int16-value %d is outside range of Int16 [%d, %d]",
			i,
			math.MinInt16,
			math.MaxInt16,
		)
	}
	return cadence.NewMeteredInt16(d.gauge, int16(i)), nil
}

// decodeInt32 decodes int32-value as
// language=CDDL
// int32-value = (int .ge -2147483648) .le 2147483647
func (d *Decoder) decodeInt32() (cadence.Value, error) {
	i, err := d.dec.DecodeInt64()
	if err != nil {
		return nil, err
	}
	if i < math.MinInt32 || i > math.MaxInt32 {
		return nil, fmt.Errorf(
			"encoded int32-value %d is outside range of Int32 [%d, %d]",
			i,
			math.MinInt32,
			math.MaxInt32,
		)
	}
	return cadence.NewMeteredInt32(d.gauge, int32(i)), nil
}

// decodeInt64 decodes int64-value as
// language=CDDL
// int64-value = (int .ge -9223372036854775808) .le 9223372036854775807
func (d *Decoder) decodeInt64() (cadence.Value, error) {
	i, err := d.dec.DecodeInt64()
	if err != nil {
		return nil, err
	}
	return cadence.NewMeteredInt64(d.gauge, i), nil
}

// decodeInt128 decodes int128-value as
// language=CDDL
// int128-value = bigint
func (d *Decoder) decodeInt128() (cadence.Value, error) {
	return cadence.NewMeteredInt128FromBig(
		d.gauge,
		func() *big.Int {
			bigInt, err := d.dec.DecodeBigInt()
			if err != nil {
				panic(fmt.Errorf("failed to decode Int128: %s", err))
			}
			return bigInt
		},
	)
}

// decodeInt256 decodes int256-value as
// language=CDDL
// int256-value = bigint
func (d *Decoder) decodeInt256() (cadence.Value, error) {
	return cadence.NewMeteredInt256FromBig(
		d.gauge,
		func() *big.Int {
			bigInt, err := d.dec.DecodeBigInt()
			if err != nil {
				panic(fmt.Errorf("failed to decode Int256: %s", err))
			}
			return bigInt
		},
	)
}

// decodeUInt decodes uint-value as
// language=CDDL
// uint-value = bigint .ge 0
func (d *Decoder) decodeUInt() (cadence.Value, error) {
	bigInt, err := d.dec.DecodeBigInt()
	if err != nil {
		return nil, err
	}
	if bigInt.Sign() < 0 {
		return nil, errors.New("encoded uint-value is negative")
	}
	return cadence.NewMeteredUIntFromBig(
		d.gauge,
		common.NewCadenceIntMemoryUsage(
			common.BigIntByteLength(bigInt),
		),
		func() *big.Int {
			return bigInt
		},
	)
}

// decodeUInt8 decodes uint8-value as
// language=CDDL
// uint8-value = uint .le 255
func (d *Decoder) decodeUInt8() (cadence.Value, error) {
	i, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}
	if i > math.MaxUint8 {
		return nil, fmt.Errorf(
			"encoded uint8-value %d is outside range of Uint8 [0, %d]",
			i,
			math.MaxUint8,
		)
	}
	return cadence.NewMeteredUInt8(d.gauge, uint8(i)), nil
}

// decodeUInt16 decodes uint16-value as
// language=CDDL
// uint16-value = uint .le 65535
func (d *Decoder) decodeUInt16() (cadence.Value, error) {
	i, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}
	if i > math.MaxUint16 {
		return nil, fmt.Errorf(
			"encoded uint16-value %d is outside range of Uint16 [0, %d]",
			i,
			math.MaxUint16,
		)
	}
	return cadence.NewMeteredUInt16(d.gauge, uint16(i)), nil
}

// decodeUInt32 decodes uint32-value as
// language=CDDL
// uint32-value = uint .le 4294967295
func (d *Decoder) decodeUInt32() (cadence.Value, error) {
	i, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}
	if i > math.MaxUint32 {
		return nil, fmt.Errorf(
			"encoded uint32-value %d is outside range of Uint32 [0, %d]",
			i,
			math.MaxUint32,
		)
	}
	return cadence.NewMeteredUInt32(d.gauge, uint32(i)), nil
}

// decodeUInt64 decodes uint64-value as
// language=CDDL
// uint64-value = uint .le 18446744073709551615
func (d *Decoder) decodeUInt64() (cadence.Value, error) {
	i, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}
	return cadence.NewMeteredUInt64(d.gauge, i), nil
}

// decodeUInt128 decodes uint128-value as
// language=CDDL
// uint128-value = bigint .ge 0
func (d *Decoder) decodeUInt128() (cadence.Value, error) {
	// NewMeteredUInt128FromBig checks if decoded big.Int is positive.
	return cadence.NewMeteredUInt128FromBig(
		d.gauge,
		func() *big.Int {
			bigInt, err := d.dec.DecodeBigInt()
			if err != nil {
				panic(fmt.Errorf("failed to decode UInt128: %s", err))
			}
			return bigInt
		},
	)
}

// decodeUInt256 decodes uint256-value as
// language=CDDL
// uint256-value = bigint .ge 0
func (d *Decoder) decodeUInt256() (cadence.Value, error) {
	// NewMeteredUInt256FromBig checks if decoded big.Int is positive.
	return cadence.NewMeteredUInt256FromBig(
		d.gauge,
		func() *big.Int {
			bigInt, err := d.dec.DecodeBigInt()
			if err != nil {
				panic(fmt.Errorf("failed to decode UInt256: %s", err))
			}
			return bigInt
		},
	)
}

// decodeWord8 decodes word8-value as
// language=CDDL
// word8-value = uint .le 255
func (d *Decoder) decodeWord8() (cadence.Value, error) {
	i, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}
	if i > math.MaxUint8 {
		return nil, fmt.Errorf(
			"encoded word8-value %d is outside range of Word8 [0, %d]",
			i,
			math.MaxUint8,
		)
	}
	return cadence.NewMeteredWord8(d.gauge, uint8(i)), nil
}

// decodeWord16 decodes word16-value as
// language=CDDL
// word16-value = uint .le 65535
func (d *Decoder) decodeWord16() (cadence.Value, error) {
	i, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}
	if i > math.MaxUint16 {
		return nil, fmt.Errorf(
			"encoded word16-value %d is outside range of Word16 [0, %d]",
			i,
			math.MaxUint16,
		)
	}
	return cadence.NewMeteredWord16(d.gauge, uint16(i)), nil
}

// decodeWord32 decodes word32-value as
// language=CDDL
// word32-value = uint .le 4294967295
func (d *Decoder) decodeWord32() (cadence.Value, error) {
	i, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}
	if i > math.MaxUint32 {
		return nil, fmt.Errorf(
			"encoded word32-value %d is outside range of Word32 [0, %d]",
			i,
			math.MaxUint32,
		)
	}
	return cadence.NewMeteredWord32(d.gauge, uint32(i)), nil
}

// decodeWord64 decodes word64-value as
// language=CDDL
// word64-value = uint .le 18446744073709551615
func (d *Decoder) decodeWord64() (cadence.Value, error) {
	i, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}
	return cadence.NewMeteredWord64(d.gauge, i), nil
}

// decodeFix64 decodes fix64-value as
// language=CDDL
// fix64-value = (int .ge -9223372036854775808) .le 9223372036854775807
func (d *Decoder) decodeFix64() (cadence.Value, error) {
	i, err := d.dec.DecodeInt64()
	if err != nil {
		return nil, err
	}
	return cadence.NewMeteredFix64FromRawFixedPointNumber(d.gauge, i)
}

// decodeUFix64 decodes ufix64-value as
// language=CDDL
// ufix64-value = uint .le 18446744073709551615
func (d *Decoder) decodeUFix64() (cadence.Value, error) {
	i, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}
	return cadence.NewMeteredUFix64FromRawFixedPointNumber(d.gauge, i)
}

// decodeOptional decodes encoded optional-value as
// language=CDDL
// optional-value = nil / value
func (d *Decoder) decodeOptional(typ *cadence.OptionalType, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	// Peek ahead for next CBOR data item type
	nextType, err := d.dec.NextType()
	if err != nil {
		return nil, err
	}

	switch nextType {
	case cbor.NilType:
		// Decode nil.
		err := d.dec.DecodeNil()
		if err != nil {
			return nil, err
		}
		return newNilOptionalValue(d.gauge, typ), nil

	default:
		// Decode value.
		value, err := d.decodeValue(typ.Type, types)
		if err != nil {
			return nil, err
		}
		return cadence.NewMeteredOptional(d.gauge, value), nil
	}
}

// decodeArray decodes encoded array-value as
// language=CDDL
// array-value = [* value]
func (d *Decoder) decodeArray(typ cadence.ArrayType, hasKnownSize bool, knownSize uint64, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	// Decode array length.
	n, err := d.dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	if hasKnownSize && knownSize != n {
		return nil, fmt.Errorf(
			"encoded array-value has %d elements (expected %d elements)",
			n,
			knownSize,
		)
	}

	elementType := typ.Element()

	values := make([]cadence.Value, n)
	for i := 0; i < int(n); i++ {
		// Decode value.
		element, err := d.decodeValue(elementType, types)
		if err != nil {
			return nil, err
		}
		values[i] = element
	}

	v, err := cadence.NewMeteredArray(
		d.gauge,
		len(values),
		func() ([]cadence.Value, error) {
			return values, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return v.WithType(typ), nil
}

// decodeDictionary decodes encoded dict-value as
// language=CDDL
// dict-value = [* (key: value, value: value)]
func (d *Decoder) decodeDictionary(typ *cadence.DictionaryType, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	// Decode array length.
	n, err := d.dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	// Check if number of elements is even.
	if n%2 != 0 {
		return nil, fmt.Errorf(
			"encoded dict-value has %d elements (expected even number of elements)",
			n,
		)
	}

	pairCount := int(n / 2)

	value, err := cadence.NewMeteredDictionary(
		d.gauge,
		pairCount,
		func() ([]cadence.KeyValuePair, error) {
			pairs := make([]cadence.KeyValuePair, pairCount)

			// previousKeyRawBytes is used to determine if dictionary keys are sorted
			var previousKeyRawBytes []byte

			for i := 0; i < pairCount; i++ {
				// element i: key

				// Decode key as raw bytes to check that key pairs are sorted by key.
				keyRawBytes, err := d.dec.DecodeRawBytes()
				if err != nil {
					return nil, err
				}

				// "Deterministic CCF Encoding Requirements" in CCF specs:
				//
				//   "dict-value key-value pairs MUST be sorted by key."
				if !bytesAreSortedBytewise(previousKeyRawBytes, keyRawBytes) {
					return nil, fmt.Errorf("encoded dict-value keys are not sorted")
				}

				previousKeyRawBytes = keyRawBytes

				// decode key from raw bytes
				keyDecoder := NewDecoder(d.gauge, keyRawBytes)
				key, err := keyDecoder.decodeValue(typ.KeyType, types)
				if err != nil {
					return nil, err
				}

				// element i+1: value
				element, err := d.decodeValue(typ.ElementType, types)
				if err != nil {
					return nil, err
				}

				pairs[i] = cadence.NewMeteredKeyValuePair(d.gauge, key, element)
			}

			// "Valid CCF Encoding Requirements" in CCF specs:
			//
			//   "Keys MUST be unique in dict-value. Decoders are not always required to check
			//   for duplicate dictionary keys. In some cases, checking for duplicate dictionary
			//   key is not necessary or it may be delegated to the application."
			//
			// Here, decoder doesn't check uniqueness of dictionary keys
			// because checking is delegated (entrusted) to Cadence runtime.
			return pairs, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return value.WithType(typ), nil
}

// decodeComposite decodes encoded composite-value as
// language=CDDL
// composite-value = [* (field: value)]
func (d *Decoder) decodeComposite(fieldTypes []cadence.Field, types *cadenceTypeByCCFTypeID) ([]cadence.Value, error) {
	fieldCount := len(fieldTypes)

	// Decode number of fields.
	err := decodeCBORArrayWithKnownSize(d.dec, uint64(fieldCount))
	if err != nil {
		return nil, err
	}

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceField,
		Amount: uint64(fieldCount),
	})

	fieldValues := make([]cadence.Value, fieldCount)

	for i := 0; i < fieldCount; i++ {
		// Decode field.
		field, err := d.decodeValue(fieldTypes[i].Type, types)
		if err != nil {
			return nil, err
		}
		fieldValues[i] = field
	}

	return fieldValues, nil
}

// decodeStruct decodes encoded composite-value as
// language=CDDL
// composite-value = [* (field: value)]
func (d *Decoder) decodeStruct(typ *cadence.StructType, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	fieldValues, err := d.decodeComposite(typ.Fields, types)
	if err != nil {
		return nil, err
	}

	v, err := cadence.NewMeteredStruct(
		d.gauge,
		len(fieldValues),
		func() ([]cadence.Value, error) {
			return fieldValues, nil
		},
	)
	if err != nil {
		return nil, err
	}

	// typ is already metered at creation.
	return v.WithType(typ), nil
}

// decodeResource decodes encoded composite-value as
// language=CDDL
// composite-value = [* (field: value)]
func (d *Decoder) decodeResource(typ *cadence.ResourceType, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	fieldValues, err := d.decodeComposite(typ.Fields, types)
	if err != nil {
		return nil, err
	}

	resource, err := cadence.NewMeteredResource(
		d.gauge,
		len(fieldValues),
		func() ([]cadence.Value, error) {
			return fieldValues, nil
		},
	)
	if err != nil {
		return nil, err
	}

	// typ is already metered at creation.
	return resource.WithType(typ), nil
}

// decodeEvent decodes encoded composite-value as
// language=CDDL
// composite-value = [* (field: value)]
func (d *Decoder) decodeEvent(typ *cadence.EventType, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	fieldValues, err := d.decodeComposite(typ.Fields, types)
	if err != nil {
		return nil, err
	}

	v, err := cadence.NewMeteredEvent(
		d.gauge,
		len(fieldValues),
		func() ([]cadence.Value, error) {
			return fieldValues, nil
		},
	)
	if err != nil {
		return nil, err
	}

	// typ is already metered at creation.
	return v.WithType(typ), nil
}

// decodeContract decodes encoded composite-value as
// language=CDDL
// composite-value = [* (field: value)]
func (d *Decoder) decodeContract(typ *cadence.ContractType, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	fieldValues, err := d.decodeComposite(typ.Fields, types)
	if err != nil {
		return nil, err
	}

	v, err := cadence.NewMeteredContract(
		d.gauge,
		len(fieldValues),
		func() ([]cadence.Value, error) {
			return fieldValues, nil
		},
	)
	if err != nil {
		return nil, err
	}

	// typ is already metered at creation.
	return v.WithType(typ), nil
}

// decodeEnum decodes encoded composite-value as
// language=CDDL
// composite-value = [* (field: value)]
func (d *Decoder) decodeEnum(typ *cadence.EnumType, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	fieldValues, err := d.decodeComposite(typ.Fields, types)
	if err != nil {
		return nil, err
	}

	v, err := cadence.NewMeteredEnum(
		d.gauge,
		len(fieldValues),
		func() ([]cadence.Value, error) {
			return fieldValues, nil
		},
	)
	if err != nil {
		return nil, err
	}

	// typ is already metered at creation.
	return v.WithType(typ), nil
}

// decodePath decodes path-value as
// language=CDDL
// path-value = [
//
//	domain: uint,
//	identifier: tstr,
//
// ]
func (d *Decoder) decodePath() (cadence.Value, error) {
	// Decode array head of length 2.
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return nil, err
	}

	// Decode domain.
	pathDomain, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}

	// Decode identifier.
	identifier, err := d.dec.DecodeString()
	if err != nil {
		return nil, err
	}

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind: common.MemoryKindRawString,
		// No need to add 1 to account for empty string: string is metered in Path struct.
		Amount: uint64(len(identifier)),
	})

	return cadence.NewMeteredPath(d.gauge, common.PathDomain(pathDomain), identifier)
}

// decodeCapability decodes encoded capability-value as
// language=CDDL
// capability-value = [
//
//	address: address-value,
//	path: path-value
//
// ]
func (d *Decoder) decodeCapability(typ *cadence.CapabilityType, types *cadenceTypeByCCFTypeID) (cadence.Value, error) {
	// typ can be different from runtime CapabilityType because borrow type can be nil.
	// In this case, runtime type is encoded with the value (as tag content for tag CBORTagTypeAndValue).

	// Check next encoded CBOR type.
	nextType, err := d.dec.NextType()
	if err != nil {
		return nil, err
	}

	if nextType == cbor.TagType {
		err := decodeCBORTagWithKnownNumber(d.dec, CBORTagTypeAndValue)
		if err != nil {
			return nil, fmt.Errorf("unexpected encoded value of Cadence type %s (%T): %s", typ.ID(), typ, err.Error())
		}

		// Decode ccf-type-and-value-message.
		return d.decodeTypeAndValue(types)
	}

	// Decode array head of length 2.
	err = decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return nil, err
	}

	// Decode address.
	address, err := d.decodeAddress()
	if err != nil {
		return nil, err
	}

	// Decode path.
	path, err := d.decodePath()
	if err != nil {
		return nil, err
	}

	return cadence.NewMeteredStorageCapability(
			d.gauge,
			path.(cadence.Path),
			address.(cadence.Address),
			typ.BorrowType),
		nil
}

// decodeTypeValue decodes encoded type-value as
// language=CDDL
// type-value = simple-type-value
//
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
//	/ restricted-type-value
//	/ capability-type-value
//	/ type-value-ref
func (d *Decoder) decodeTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	// Decode tag number.
	tagNum, err := d.dec.DecodeTagNumber()
	if err != nil {
		return nil, err
	}

	switch tagNum {

	case CBORTagTypeValueRef:
		return d.decodeTypeRef(visited)

	case CBORTagSimpleTypeValue:
		return d.decodeSimpleTypeID()

	case CBORTagOptionalTypeValue:
		return d.decodeOptionalType(visited, d.decodeTypeValue)

	case CBORTagVarsizedArrayTypeValue:
		return d.decodeVarSizedArrayType(visited, d.decodeTypeValue)

	case CBORTagConstsizedArrayTypeValue:
		return d.decodeConstantSizedArrayType(visited, d.decodeTypeValue)

	case CBORTagDictTypeValue:
		return d.decodeDictType(visited, d.decodeTypeValue)

	case CBORTagCapabilityTypeValue:
		return d.decodeCapabilityType(visited, d.decodeNullableTypeValue)

	case CBORTagReferenceTypeValue:
		return d.decodeReferenceType(visited, d.decodeTypeValue)

	case CBORTagRestrictedTypeValue:
		return d.decodeRestrictedType(visited, d.decodeNullableTypeValue, d.decodeTypeValue)

	case CBORTagFunctionTypeValue:
		return d.decodeFunctionTypeValue(visited)

	case CBORTagStructTypeValue:
		return d.decodeStructTypeValue(visited)

	case CBORTagResourceTypeValue:
		return d.decodeResourceTypeValue(visited)

	case CBORTagEventTypeValue:
		return d.decodeEventTypeValue(visited)

	case CBORTagContractTypeValue:
		return d.decodeContractTypeValue(visited)

	case CBORTagEnumTypeValue:
		return d.decodeEnumTypeValue(visited)

	case CBORTagStructInterfaceTypeValue:
		return d.decodeStructInterfaceTypeValue(visited)

	case CBORTagResourceInterfaceTypeValue:
		return d.decodeResourceInterfaceTypeValue(visited)

	case CBORTagContractInterfaceTypeValue:
		return d.decodeContractInterfaceTypeValue(visited)

	default:
		return nil, fmt.Errorf("unsupported type-value with CBOR tag number %d", tagNum)
	}
}

// decodeNullableTypeValue decodes encoded type-value or nil.
func (d *Decoder) decodeNullableTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	cborType, err := d.dec.NextType()
	if err != nil {
		return nil, err
	}
	if cborType == cbor.NilType {
		err = d.dec.DecodeNil()
		return nil, err
	}
	return d.decodeTypeValue(visited)
}

// decodeStructTypeValue decodes struct-type-value as
// language=CDDL
// struct-type-value =
//
//	; cbor-tag-struct-type-value
//	#6.208(composite-type-value)
func (d *Decoder) decodeStructTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	ctr := func(
		location common.Location,
		qualifiedIdentifier string,
		typ cadence.Type,
	) (cadence.Type, error) {
		if typ != nil {
			return nil, fmt.Errorf(
				"encoded struct-type-value has type %s (expected nil type)",
				typ.ID(),
			)
		}
		return cadence.NewMeteredStructType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			nil,
		), nil
	}

	return d.decodeCompositeTypeValue(visited, ctr)
}

// decodeResourceTypeValue decodes resource-type-value as
// language=CDDL
// resource-type-value =
//
//	; cbor-tag-resource-type-value
//	#6.209(composite-type-value)
func (d *Decoder) decodeResourceTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	ctr := func(
		location common.Location,
		qualifiedIdentifier string,
		typ cadence.Type,
	) (cadence.Type, error) {
		if typ != nil {
			return nil, fmt.Errorf(
				"encoded resource-type-value has type %s (expected nil type)",
				typ.ID(),
			)
		}
		return cadence.NewMeteredResourceType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			nil,
		), nil
	}

	return d.decodeCompositeTypeValue(visited, ctr)
}

// decodeEventTypeValue decodes event-type-value as
// language=CDDL
// event-type-value =
//
//	; cbor-tag-event-type-value
//	#6.210(composite-type-value)
func (d *Decoder) decodeEventTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	ctr := func(
		location common.Location,
		qualifiedIdentifier string,
		typ cadence.Type,
	) (cadence.Type, error) {
		if typ != nil {
			return nil, fmt.Errorf(
				"encoded event-type-value has type %s (expected nil type)",
				typ.ID(),
			)
		}
		return cadence.NewMeteredEventType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			nil,
		), nil
	}

	return d.decodeCompositeTypeValue(visited, ctr)
}

// decodeContractTypeValue decodes contract-type-value as
// language=CDDL
// contract-type-value =
//
//	; cbor-tag-contract-type-value
//	#6.211(composite-type-value)
func (d *Decoder) decodeContractTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	ctr := func(
		location common.Location,
		qualifiedIdentifier string,
		typ cadence.Type,
	) (cadence.Type, error) {
		if typ != nil {
			return nil, fmt.Errorf(
				"encoded contract-type-value has type %s (expected nil type)",
				typ.ID(),
			)
		}
		return cadence.NewMeteredContractType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			nil,
		), nil
	}

	return d.decodeCompositeTypeValue(visited, ctr)
}

// decodeEnumTypeValue decodes enum-type-value as
// language=CDDL
// enum-type-value =
//
//	; cbor-tag-enum-type-value
//	#6.212(composite-type-value)
func (d *Decoder) decodeEnumTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	ctr := func(
		location common.Location,
		qualifiedIdentifier string,
		typ cadence.Type,
	) (cadence.Type, error) {
		if typ == nil {
			return nil, fmt.Errorf("encoded enum-type-value has nil type")
		}
		return cadence.NewMeteredEnumType(
			d.gauge,
			location,
			qualifiedIdentifier,
			typ,
			nil,
			nil,
		), nil
	}

	return d.decodeCompositeTypeValue(visited, ctr)
}

// decodeStructInterfaceTypeValue decodes struct-inteface-type-value as
// language=CDDL
// struct-interface-type-value =
//
//	; cbor-tag-struct-interface-type-value
//	#6.224(composite-type-value)
func (d *Decoder) decodeStructInterfaceTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	ctr := func(
		location common.Location,
		qualifiedIdentifier string,
		typ cadence.Type,
	) (cadence.Type, error) {
		if typ != nil {
			return nil, fmt.Errorf(
				"encoded struct-interface-type-value has type %s (expected nil type)",
				typ.ID(),
			)
		}
		return cadence.NewMeteredStructInterfaceType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			nil,
		), nil
	}

	return d.decodeCompositeTypeValue(visited, ctr)
}

// decodeResourceInterfaceTypeValue decodes resource-inteface-type-value as
// language=CDDL
// resource-interface-type-value =
//
//	; cbor-tag-resource-interface-type-value
//	#6.225(composite-type-value)
func (d *Decoder) decodeResourceInterfaceTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	ctr := func(
		location common.Location,
		qualifiedIdentifier string,
		typ cadence.Type,
	) (cadence.Type, error) {
		if typ != nil {
			return nil, fmt.Errorf(
				"encoded resource-interface-type-value has type %s (expected nil type)",
				typ.ID(),
			)
		}
		return cadence.NewMeteredResourceInterfaceType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			nil,
		), nil
	}

	return d.decodeCompositeTypeValue(visited, ctr)
}

// decodeContractInterfaceTypeValue decodes contract-inteface-type-value as
// language=CDDL
// contract-interface-type-value =
//
//	; cbor-tag-contract-interface-type-value
//	#6.226(composite-type-value)
func (d *Decoder) decodeContractInterfaceTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	ctr := func(
		location common.Location,
		qualifiedIdentifier string,
		typ cadence.Type,
	) (cadence.Type, error) {
		if typ != nil {
			return nil, fmt.Errorf(
				"encoded contract-interface-type-value has type %s (expected nil type)",
				typ.ID(),
			)
		}
		return cadence.NewMeteredContractInterfaceType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			nil,
		), nil
	}
	return d.decodeCompositeTypeValue(visited, ctr)
}

type compositeTypeConstructor func(
	location common.Location,
	qualifiedIdentifier string,
	typ cadence.Type,
) (cadence.Type, error)

type compositeTypeValue struct {
	ccfID           ccfTypeID
	location        common.Location
	identifier      string
	typ             cadence.Type
	rawFields       []byte
	rawInitializers []byte
}

// decodeCompositeTypeValue decodes composite-type-value.
// See _decodeCompositeTypeValue for details.
func (d *Decoder) decodeCompositeTypeValue(
	visited *cadenceTypeByCCFTypeID,
	constructor compositeTypeConstructor,
) (cadence.Type, error) {
	compTypeValue, err := d._decodeCompositeTypeValue(visited)
	if err != nil {
		return nil, err
	}

	compositeType, err := constructor(
		compTypeValue.location,
		compTypeValue.identifier,
		compTypeValue.typ,
	)
	if err != nil {
		return nil, err
	}

	if compositeType == nil {
		// Sanity check that compositeType isn't nil.
		return nil, errors.New("unexpected nil composite type value")
	}

	// "Deterministic CCF Encoding Requirements" in CCF specs:
	//
	//   "composite-type-value.id MUST be identical to the zero-based encoding order type-value."
	if compTypeValue.ccfID != newCCFTypeIDFromUint64(uint64(visited.count())) {
		return nil, fmt.Errorf(
			"encoded composite-type-value's CCF type ID %d doesn't match zero-based encoding order composite-type-value",
			compTypeValue.ccfID,
		)
	}

	newType := visited.add(compTypeValue.ccfID, compositeType)
	if !newType {
		// "Valid CCF Encoding Requirements" in CCF specs:
		//
		//   "composite-type-value.id MUST be unique in the same composite-type-value data item."
		return nil, fmt.Errorf("found duplicate CCF type ID %d in encoded composite-type-value", compTypeValue.ccfID)
	}

	// Decode fields after type is resolved to handle recursive types.
	dec := NewDecoder(d.gauge, compTypeValue.rawFields)
	fields, err := dec.decodeCompositeFields(visited, dec.decodeTypeValue)
	if err != nil {
		return nil, err
	}

	// Decode initializers after type is resolved to handle recursive types.
	dec = NewDecoder(d.gauge, compTypeValue.rawInitializers)
	initializers, err := dec.decodeInitializerTypeValues(visited)
	if err != nil {
		return nil, err
	}

	switch compositeType := compositeType.(type) {
	case *cadence.StructType:
		compositeType.Fields = fields
		compositeType.Initializers = initializers

	case *cadence.ResourceType:
		compositeType.Fields = fields
		compositeType.Initializers = initializers

	case *cadence.EventType:
		if len(initializers) != 1 {
			return nil, fmt.Errorf(
				"encoded event-type-value has %d initializations (expected 1 initialization)",
				len(initializers),
			)
		}
		compositeType.Fields = fields
		compositeType.Initializer = initializers[0]

	case *cadence.ContractType:
		compositeType.Fields = fields
		compositeType.Initializers = initializers

	case *cadence.EnumType:
		compositeType.Fields = fields
		compositeType.Initializers = initializers

	case *cadence.StructInterfaceType:
		compositeType.Fields = fields
		compositeType.Initializers = initializers

	case *cadence.ResourceInterfaceType:
		compositeType.Fields = fields
		compositeType.Initializers = initializers

	case *cadence.ContractInterfaceType:
		compositeType.Fields = fields
		compositeType.Initializers = initializers
	}

	return compositeType, nil
}

// _decodeCompositeTypeValue decodes composite-type-value as
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
func (d *Decoder) _decodeCompositeTypeValue(visited *cadenceTypeByCCFTypeID) (*compositeTypeValue, error) {
	// Decode array of length 5
	err := decodeCBORArrayWithKnownSize(d.dec, 5)
	if err != nil {
		return nil, err
	}

	// element 0: id (used to lookup repeated or recursive types)
	ccfID, err := d.decodeCCFTypeID()
	if err != nil {
		return nil, err
	}

	// element 1: cadence-type-id
	_, location, identifier, err := d.decodeCadenceTypeID()
	if err != nil {
		return nil, err
	}

	// element 2: type (only used by enum type value)
	typ, err := d.decodeNullableTypeValue(visited)
	if err != nil {
		return nil, err
	}

	// element 3: fields
	rawFields, err := d.dec.DecodeRawBytes()
	if err != nil {
		return nil, err
	}

	// element 4: initializers
	rawInitializers, err := d.dec.DecodeRawBytes()
	if err != nil {
		return nil, err
	}

	return &compositeTypeValue{
		ccfID:           ccfID,
		location:        location,
		identifier:      identifier,
		typ:             typ,
		rawFields:       rawFields,
		rawInitializers: rawInitializers,
	}, nil
}

// decodeInitializerTypeValues decodes composite initializers as
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
func (d *Decoder) decodeInitializerTypeValues(visited *cadenceTypeByCCFTypeID) ([][]cadence.Parameter, error) {
	// Decode number of initializers.
	count, err := d.dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	if count > 1 {
		return nil, fmt.Errorf("expect 0 or 1 initializer, got %d initializers", count)
	}

	// Unmetered because this is created as an array of nil arrays, not Parameter structs.
	initializerTypes := make([][]cadence.Parameter, count)
	for i := 0; i < int(count); i++ {
		initializerTypes[i], err = d.decodeParameterTypeValues(visited)
		if err != nil {
			return nil, err
		}
	}

	return initializerTypes, nil
}

// decodeTypeParameterTypeValues decodes type parameters as
// language=CDDL
//
//	 type-parameters: [
//		* [
//			name: tstr,
//			type-bound: type-value / nil
//		  ]
//	 ]
func (d *Decoder) decodeTypeParameterTypeValues(visited *cadenceTypeByCCFTypeID) ([]cadence.TypeParameter, error) {
	// Decode number of parameters.
	count, err := d.dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	if count == 0 {
		return []cadence.TypeParameter{}, nil
	}

	typeParameterTypes := make([]cadence.TypeParameter, count)
	typeParameterNames := make(map[string]struct{}, count)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceTypeParameter,
		Amount: count,
	})

	for i := 0; i < int(count); i++ {
		// Decode type parameter.
		typeParam, err := d.decodeTypeParameterTypeValue(visited)
		if err != nil {
			return nil, err
		}

		// "Valid CCF Encoding Requirements" in CCF specs:
		//
		//   "All parameter lists MUST have unique identifier"
		if _, ok := typeParameterNames[typeParam.Name]; ok {
			return nil, fmt.Errorf("found duplicate type parameter name %s", typeParam.Name)
		}

		typeParameterNames[typeParam.Name] = struct{}{}
		typeParameterTypes[i] = typeParam
	}

	return typeParameterTypes, nil
}

// decodeTypeParameterTypeValue decodes type parameter as
// language=CDDL
//
//	 [
//		name: tstr,
//		type-bound: type-value / nil
//	 ]
func (d *Decoder) decodeTypeParameterTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.TypeParameter, error) {
	// Decode array head of length 2
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return cadence.TypeParameter{}, err
	}

	// element 0: name
	name, err := d.dec.DecodeString()
	if err != nil {
		return cadence.TypeParameter{}, err
	}

	// element 2: type
	t, err := d.decodeNullableTypeValue(visited)
	if err != nil {
		return cadence.TypeParameter{}, err
	}

	// Unmetered because decodeTypeParamTypeValue is metered in decodeTypeParamTypeValues and called nowhere else
	// Type is metered.
	return cadence.NewTypeParameter(name, t), nil
}

// decodeParameterTypeValues decodes composite initializer parameter types as
// language=CDDL
//
//	 [
//	    * [
//	        label: tstr,
//	        identifier: tstr,
//	        type: type-value
//	    ]
//	]
func (d *Decoder) decodeParameterTypeValues(visited *cadenceTypeByCCFTypeID) ([]cadence.Parameter, error) {
	// Decode number of parameters.
	count, err := d.dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	if count == 0 {
		return []cadence.Parameter{}, nil
	}

	parameterTypes := make([]cadence.Parameter, count)
	parameterLabels := make(map[string]struct{}, count)
	parameterIdentifiers := make(map[string]struct{}, count)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceParameter,
		Amount: count,
	})

	for i := 0; i < int(count); i++ {
		// Decode parameter.
		param, err := d.decodeParameterTypeValue(visited)
		if err != nil {
			return nil, err
		}

		// "Valid CCF Encoding Requirements" in CCF specs:
		//
		//   "All parameter lists MUST have unique identifier"
		if _, ok := parameterLabels[param.Label]; ok {
			return nil, fmt.Errorf("found duplicate parameter label %s", param.Label)
		}

		if _, ok := parameterIdentifiers[param.Identifier]; ok {
			return nil, fmt.Errorf("found duplicate parameter identifier %s", param.Identifier)
		}

		parameterLabels[param.Label] = struct{}{}
		parameterIdentifiers[param.Identifier] = struct{}{}

		parameterTypes[i] = param
	}

	return parameterTypes, nil
}

// decodeParameterTypeValue decodes composite initializer parameter as
// language=CDDL
//
//	 [
//	    label: tstr,
//	    identifier: tstr,
//	    type: type-value
//	]
func (d *Decoder) decodeParameterTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Parameter, error) {
	// Decode array head of length 3
	err := decodeCBORArrayWithKnownSize(d.dec, 3)
	if err != nil {
		return cadence.Parameter{}, err
	}

	// element 0: label
	label, err := d.dec.DecodeString()
	if err != nil {
		return cadence.Parameter{}, err
	}

	// element 1: identifier
	identifier, err := d.dec.DecodeString()
	if err != nil {
		return cadence.Parameter{}, err
	}

	// element 2: type
	t, err := d.decodeTypeValue(visited)
	if err != nil {
		return cadence.Parameter{}, err
	}

	if t == nil {
		return cadence.Parameter{}, errors.New("unexpected nil parameter type")
	}

	// Unmetered because decodeParamTypeValue is metered in decodeParamTypeValues and called nowhere else
	// Type is metered.
	return cadence.NewParameter(label, identifier, t), nil
}

// decodeFunctionTypeValue decodes encoded function-value as
// language=CDDL
// function-value = [
//
//	 type-parameters: [
//		* [
//			name: tstr,
//			type-bound: type-value / nil
//		  ]
//	 ]
//	parameters: [
//	    * [
//	        label: tstr,
//	        identifier: tstr,
//	        type: type-value
//	    ]
//	]
//	return-type: type-value
//
// ]
func (d *Decoder) decodeFunctionTypeValue(visited *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	// Decode array head of length 3
	err := decodeCBORArrayWithKnownSize(d.dec, 3)
	if err != nil {
		return nil, err
	}

	// element 0: type parameters
	typeParameters, err := d.decodeTypeParameterTypeValues(visited)
	if err != nil {
		return nil, err
	}

	// element 1: parameters
	parameters, err := d.decodeParameterTypeValues(visited)
	if err != nil {
		return nil, err
	}

	// element 2: return-type
	returnType, err := d.decodeTypeValue(visited)
	if err != nil {
		return nil, err
	}

	if returnType == nil {
		return nil, errors.New("unexpected nil function return type")
	}

	return cadence.NewMeteredFunctionType(
		d.gauge,
		typeParameters,
		parameters,
		returnType,
	), nil
}

func decodeCBORArrayWithKnownSize(dec *cbor.StreamDecoder, n uint64) error {
	c, err := dec.DecodeArrayHead()
	if err != nil {
		return err
	}
	if c != n {
		return fmt.Errorf("CBOR array has %d elements (expected %d elements)", c, n)
	}
	return nil
}

func decodeCBORTagWithKnownNumber(dec *cbor.StreamDecoder, n uint64) error {
	tagNum, err := dec.DecodeTagNumber()
	if err != nil {
		return err
	}
	if tagNum != n {
		return fmt.Errorf("CBOR tag number is %d (expected %d)", tagNum, n)
	}
	return nil
}

// newNilOptionalValue returns (nested) cadence.Optional nil value.
func newNilOptionalValue(gauge common.MemoryGauge, ot *cadence.OptionalType) cadence.Optional {
	v := cadence.NewMeteredOptional(gauge, nil)
	for {
		var ok bool
		ot, ok = ot.Type.(*cadence.OptionalType)
		if !ok {
			break
		}
		v = cadence.NewMeteredOptional(gauge, v)
	}
	return v
}
