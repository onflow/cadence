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
	"encoding/binary"
	"fmt"
	"io"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/cbf/common_codec"
	"github.com/onflow/cadence/runtime/common"
)

// A Decoder decodes custom-encoded representations of Cadence values.
type Decoder struct {
	r           common_codec.LocatedReader
	buf         []byte
	memoryGauge common.MemoryGauge
	// TODO abi for cutting down on what needs to be transferred
}

// Decode returns a Cadence value decoded from its custom-encoded representation.
//
// This function returns an error if the bytes represent a custom encoding that
// is malformed, does not conform to the custom Cadence specification, or contains
// an unknown composite type.
func Decode(gauge common.MemoryGauge, b []byte) (v cadence.Value, err error) {
	r := bytes.NewReader(b)
	dec := NewDecoder(gauge, r)

	return dec.Decode()
}

func MustDecode(gauge common.MemoryGauge, b []byte) cadence.Value {
	v, err := Decode(gauge, b)
	if err != nil {
		panic(err)
	}
	return v
}

// NewDecoder initializes a Decoder that will decode custom-encoded bytes from the
// given io.Reader.
func NewDecoder(memoryGauge common.MemoryGauge, r io.Reader) *Decoder {
	return &Decoder{
		r:           common_codec.NewLocatedReader(r),
		memoryGauge: memoryGauge,
	}
}

// TODO need a way to decode values with known type vs values with unknown type
//      if type is known then no identifier is needed, such as for elements in constant sized array

// DecodeValue reads custom-encoded bytes from the io.Reader and decodes them to a
// Cadence value.
//
// This function returns an error if the bytes represent a custom encoding that
// is malformed, does not conform to the custom Cadence specification, or contains
// an unknown composite type.

func (d *Decoder) Decode() (v cadence.Value, err error) {
	version, err := d.read(1)
	if err != nil {
		return nil, err
	}

	if version[0] != VERSION {
		err = fmt.Errorf("expected CBF version %d but got  %d", VERSION, version[0])
		return
	}

	return d.DecodeValue()
}

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
