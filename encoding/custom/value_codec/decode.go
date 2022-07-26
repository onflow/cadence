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

package value_codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/custom/common_codec"
	"github.com/onflow/cadence/runtime/common"
)

// A Decoder decodes custom-encoded representations of Cadence values.
type Decoder struct {
	r           common_codec.LocatedReader
	buf         []byte
	memoryGauge common.MemoryGauge
	types       map[common.TypeID]*cadence.CompositeType
	// abi any // TODO abi for cutting down on what needs to be transferred
}

// Decode returns a Cadence value decoded from its custom-encoded representation.
//
// This function returns an error if the bytes represent a custom encoding that
// is malformed, does not conform to the custom Cadence specification, or contains
// an unknown composite type.
func Decode(gauge common.MemoryGauge, b []byte) (cadence.Value, error) {
	r := bytes.NewReader(b)
	dec := NewDecoder(gauge, r)

	v, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	return v, nil
}

// NewDecoder initializes a Decoder that will decode custom-encoded bytes from the
// given io.Reader.
func NewDecoder(memoryGauge common.MemoryGauge, r io.Reader) *Decoder {
	return &Decoder{
		r:           common_codec.NewLocatedReader(r),
		memoryGauge: memoryGauge,
	}
}

// Decode reads custom-encoded bytes from the io.Reader and decodes them to a
// Cadence value.
//
// This function returns an error if the bytes represent a custom encoding that
// is malformed, does not conform to the custom Cadence specification, or contains
// an unknown composite type.
func (d *Decoder) Decode() (value cadence.Value, err error) {
	return d.DecodeValue()
}

// TODO need a way to decode values with known type vs values with unknown type
//      if type is known then no identifier is needed, such as for elements in constant sized array
func (d *Decoder) DecodeValue() (value cadence.Value, err error) {
	identifier, err := d.DecodeIdentifier()
	if err != nil {
		return
	}

	switch identifier {
	case EncodedValueVoid:
		value = cadence.NewMeteredVoid(d.memoryGauge)
	}

	switch identifier {
	case EncodedValueVoid:
		value = cadence.NewMeteredVoid(d.memoryGauge)
	case EncodedValueBool:
		value, err = d.DecodeBool()
	case EncodedValueOptional:
		value, err = d.DecodeOptional()
	case EncodedValueArray:
		value, err = d.DecodeArray()
	}

	return
}

func (d *Decoder) DecodeIdentifier() (id EncodedValue, err error) {
	b, err := d.read(1)
	if err != nil {
		return
	}

	id = EncodedValue(b[0])
	return
}

func (d *Decoder) DecodeVoid() (value cadence.Void, err error) {
	_, err = d.read(1)
	value = cadence.NewMeteredVoid(d.memoryGauge)
	return
}

func (d *Decoder) DecodeOptional() (value cadence.Optional, err error) {
	isNil, err := d.DecodeBool()
	if isNil || err != nil {
		return
	}

	innerValue, err := d.DecodeValue()
	value = cadence.NewMeteredOptional(d.memoryGauge, innerValue)
	return
}

func (d *Decoder) DecodeBool() (value cadence.Bool, err error) {
	boolean, err := common_codec.DecodeBool(&d.r)
	if err != nil {
		return
	}

	value = cadence.NewMeteredBool(d.memoryGauge, boolean)
	return
}

// TODO how am I to represent complex types?
//      like a CompositeValue has a type... do I put it inside the CompositeValue encoding?
//      what I want to do is put it early enough that the CompositeValue encoding can be very light
//      maybe do it like: identifier type value
//      have it so the identifier tells you the baset type

func (d *Decoder) DecodeArray() (array cadence.Array, err error) {
	arrayType, err := d.DecodeArrayType()
	if err != nil {
		return
	}

	var size int
	switch t := arrayType.(type) {
	case cadence.ConstantSizedArrayType:
		size = int(t.Size)
	case cadence.VariableSizedArrayType:
		size, err = d.DecodeLength()
		if err != nil {
			return
		}
	}
	array, err = cadence.NewMeteredArray(d.memoryGauge, size, func() (elements []cadence.Value, err error) {
		elements = make([]cadence.Value, 0, size)
		for i := 0; i < size; i++ {
			// TODO if `elementType` is concrete then each element needn't encode its type
			var elementValue cadence.Value
			elementValue, err = d.DecodeValue()
			if err != nil {
				return
			}
			elements = append(elements, elementValue)
		}

		return elements, nil
	})

	array = array.WithType(arrayType)

	return
}

func (d *Decoder) DecodeArrayType() (t cadence.ArrayType, err error) {
	b, err := d.read(1)
	if err != nil {
		return
	}

	elementType, err := d.DecodeType()
	if err != nil {
		return
	}

	switch EncodedArrayType(b[0]) {
	case EncodedArrayTypeVariable:
		t = cadence.NewMeteredVariableSizedArrayType(d.memoryGauge, elementType)
	case EncodedArrayTypeConstant:
		var size int
		size, err = d.DecodeLength()
		if err != nil {
			return
		}
		t = cadence.NewMeteredConstantSizedArrayType(d.memoryGauge, uint(size), elementType)
	default:
		err = fmt.Errorf("invalid array type encoding: %d", b[0])
	}
	return
}

//
// Types
//

func (d *Decoder) DecodeType() (t cadence.Type, err error) {
	typeIdentifer, err := d.DecodeTypeIdentifier()

	switch typeIdentifer {
	case EncodedTypeVoid:
		t = cadence.NewMeteredVoidType(d.memoryGauge)
	case EncodedTypeOptional:
		t, err = d.DecodeOptionalType()
	case EncodedTypeBool:
		t = cadence.NewMeteredBoolType(d.memoryGauge)
	case EncodedTypeArray:
		t, err = d.DecodeArrayType()
	case EncodedTypeAnyType:
		t = cadence.NewMeteredAnyType(d.memoryGauge)
	case EncodedTypeAnyStructType:
		t = cadence.NewMeteredAnyStructType(d.memoryGauge)
	default:
		err = fmt.Errorf("unknown type identifier: %d", typeIdentifer)
	}
	return
}

func (d *Decoder) DecodeTypeIdentifier() (t EncodedType, err error) {
	b, err := d.read(1)
	t = EncodedType(b[0])
	return
}

func (d *Decoder) DecodeOptionalType() (t cadence.OptionalType, err error) {
	isNil, err := common_codec.DecodeBool(&d.r)
	if isNil || err != nil {
		return
	}

	elementType, err := d.DecodeType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredOptionalType(d.memoryGauge, elementType)
	return
}

//
// Other
//

func (d *Decoder) DecodeLength() (length int, err error) {
	b, err := d.read(4)
	if err != nil {
		return
	}

	asUint32 := binary.BigEndian.Uint32(b)

	length = int(asUint32)
	return
}

func (d *Decoder) read(howManyBytes int) (b []byte, err error) {
	b = make([]byte, howManyBytes)
	_, err = d.r.Read(b)
	return
}
