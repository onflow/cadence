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

package cbf_codec

import (
	"bytes"
	"io"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/cbf/common_codec"
)

// An Encoder converts Cadence values into custom-encoded bytes.
type Encoder struct {
	w common_codec.LengthyWriter
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
func MustEncode(value cadence.Value) []byte {
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
		w: common_codec.NewLengthyWriter(w),
	}
}

// Encode writes the custom-encoded representation of the given value to this
// encoder's io.Writer.
//
// This function returns an error if the given value's type is not supported
// by this encoder.
func (e *Encoder) Encode(value cadence.Value) (err error) {
	return e.EncodeValue(value)
}

func (e *Encoder) write(b []byte) (err error) {
	_, err = e.w.Write(b)
	return
}

func (e *Encoder) writeByte(b byte) (err error) {
	_, err = e.w.Write([]byte{b})
	return
}
