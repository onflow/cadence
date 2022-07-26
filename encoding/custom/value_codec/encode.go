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
)

// An Encoder converts Cadence values into custom-encoded bytes.
type Encoder struct {
	w        common_codec.LengthyWriter
	typeDefs map[cadence.Type]int
}

// EncodeValue returns the custom-encoded representation of the given value.
//
// This function returns an error if the Cadence value cannot be represented in the custom format.
func EncodeValue(value cadence.Value) ([]byte, error) {
	var w bytes.Buffer
	enc := NewEncoder(&w)

	err := enc.Encode(value)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MustEncode returns the custom-encoded representation of the given value, or panics
// if the value cannot be represented in the custom format.
func MustEncodeValue(value cadence.Value) []byte {
	b, err := EncodeValue(value)
	if err != nil {
		panic(err)
	}
	return b
}

// NewEncoder initializes an Encoder that will write custom-encoded bytes to the
// given io.Writer.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:        common_codec.NewLengthyWriter(w),
		typeDefs: map[cadence.Type]int{},
	}
}

// TODO include leading byte with version information
//      maybe include other metadata too, like the size the decoder's typeDefs map will be

// Encode writes the custom-encoded representation of the given value to this
// encoder's io.Writer.
//
// This function returns an error if the given value's type is not supported
// by this encoder.
func (e *Encoder) Encode(value cadence.Value) (err error) {
	return e.EncodeValue(value)
}

//
// Values
//

// EncodeValue encodes any supported cadence.Value.
func (e *Encoder) EncodeValue(value cadence.Value) (err error) {
	// Non-recursable types
	switch v := value.(type) {
	case cadence.Void:
		return e.EncodeValueIdentifier(EncodedValueVoid)
	case cadence.Bool:
		err = e.EncodeValueIdentifier(EncodedValueBool)
		if err != nil {
			return
		}
		return common_codec.EncodeBool(&e.w, bool(v))
	}

	switch v := value.(type) {
	case cadence.Optional:
		err = e.EncodeValueIdentifier(EncodedValueOptional)
		if err != nil {
			return
		}
		return e.EncodeOptional(v)
	case cadence.Array:
		// TODO if an array type is known ahead of time, what needs to be encoded and what can be excluded?
		//      specifically, how much of ArrayType is needed? Just constant vs variable?
		//		aka is Size part of what's known ahead of time? it's certainly encoded in the sema type
		err = e.EncodeValueIdentifier(EncodedValueArray)
		if err != nil {
			return
		}
		return e.EncodeArray(v)
	}

	return fmt.Errorf("unexpected value: ${value}")
}

type EncodedValue byte

const (
	EncodedValueUnknown EncodedValue = iota

	EncodedValueVoid
	EncodedValueBool
	EncodedValueArray
	EncodedValueOptional
)

func (e *Encoder) EncodeValueIdentifier(id EncodedValue) (err error) {
	return e.write([]byte{byte(id)})
}

func (e *Encoder) EncodeOptional(value cadence.Optional) (err error) {
	isNil := value.Value == nil
	err = common_codec.EncodeBool(&e.w, isNil)
	if isNil || err != nil {
		return
	}

	return e.EncodeValue(value.Value)
}

// TODO handle encode/decode of the two array types in a cleaner way
func (e *Encoder) EncodeArray(value cadence.Array) (err error) {
	err = e.EncodeArrayType(value.ArrayType)
	if err != nil {
		return
	}

	switch v := value.ArrayType.(type) {
	case cadence.VariableSizedArrayType:
		err = e.EncodeLength(len(value.Values))
		if err != nil {
			return
		}
	case cadence.ConstantSizedArrayType:
		if len(value.Values) != int(v.Size) {
			return fmt.Errorf("constant size array size=%d but has %d elements", v.Size, len(value.Values))
		}
	}

	for _, element := range value.Values {
		err = e.EncodeValue(element)
		if err != nil {
			return err
		}
	}

	return
}

//
// Types
//

func (e *Encoder) EncodeType(t cadence.Type) (err error) {
	switch actualType := t.(type) {
	case cadence.VoidType:
		return e.EncodeTypeIdentifier(EncodedTypeVoid)
	case cadence.BoolType:
		return e.EncodeTypeIdentifier(EncodedTypeBool)
	case cadence.OptionalType:
		err = e.EncodeTypeIdentifier(EncodedTypeVoid)
		if err != nil {
			return
		}
		return e.EncodeOptionalType(actualType)
	case cadence.AnyType:
		return e.EncodeTypeIdentifier(EncodedTypeAnyType)
	case cadence.AnyStructType:
		return e.EncodeTypeIdentifier(EncodedTypeAnyStructType)
	}

	if bufferOffset, usePointer := e.typeDefs[t]; usePointer {
		return e.EncodePointer(bufferOffset)
	}
	e.typeDefs[t] = e.w.Len() + 1 // point to encoded type, not its identifier

	switch actualType := t.(type) {
	case cadence.ArrayType:
		return e.EncodeArrayType(actualType)
	}

	return fmt.Errorf("unknown type: %s", t)
}

func (e *Encoder) EncodeTypeIdentifier(id EncodedType) (err error) {
	return e.write([]byte{byte(id)})
}

type EncodedType byte

const (
	EncodedTypeUnknown EncodedType = iota

	// Concrete Types

	EncodedTypeVoid
	EncodedTypeBool

	// Abstract Types

	EncodedTypeAnyType
	EncodedTypeAnyStructType

	// Pointable Types

	EncodedTypeArray
	EncodedTypeOptional

	// Other Types

	EncodedTypePointer
)

func (e *Encoder) EncodePointer(bufferOffset int) (err error) {
	err = e.write([]byte{byte(EncodedTypePointer)})
	if err != nil {
		return
	}

	return e.EncodeLength(bufferOffset)
}

func (e *Encoder) EncodeOptionalType(t cadence.OptionalType) (err error) {
	return e.EncodeType(t.Type)
}

type EncodedArrayType byte

const (
	EncodedArrayTypeUnknown EncodedArrayType = iota
	EncodedArrayTypeVariable
	EncodedArrayTypeConstant
)

func (e *Encoder) EncodeArrayType(t cadence.ArrayType) (err error) {
	var b EncodedArrayType
	switch t.(type) {
	case cadence.VariableSizedArrayType:
		b = EncodedArrayTypeVariable
	case cadence.ConstantSizedArrayType:
		b = EncodedArrayTypeConstant
	default:
		return fmt.Errorf("unknown array type: %s", t)
	}
	err = e.write([]byte{byte(b)})
	if err != nil {
		return
	}

	err = e.EncodeType(t.Element())
	if err != nil {
		return
	}

	switch concreteType := t.(type) {
	case cadence.ConstantSizedArrayType:
		err = e.EncodeLength(int(concreteType.Size))
	}

	return
}

//
// Other
//

// EncodeLength encodes a non-negative length as a uint32.
// It uses 4 bytes.
func (e *Encoder) EncodeLength(length int) (err error) {
	if length < 0 { // TODO is this safety check useful?
		return fmt.Errorf("cannot encode length below zero: %d", length)
	}

	l := uint32(length)

	return binary.Write(&e.w, binary.BigEndian, l)
}

func (e *Encoder) write(b []byte) (err error) {
	_, err = e.w.Write(b)
	return
}
