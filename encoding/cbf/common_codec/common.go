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

package common_codec

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
)

//
// LengthyWriter
//

type LengthyWriter struct {
	w      io.Writer
	length int
}

func NewLengthyWriter(w io.Writer) LengthyWriter {
	return LengthyWriter{w: w}
}

func (l *LengthyWriter) Write(p []byte) (n int, err error) {
	n, err = l.w.Write(p)
	l.length += n
	return
}

func (l *LengthyWriter) Len() int {
	return l.length
}

//
// LocatedReader
//

type LocatedReader struct {
	r        io.Reader
	location int
}

func NewLocatedReader(r io.Reader) LocatedReader {
	return LocatedReader{r: r}
}

func (l *LocatedReader) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	l.location += n
	return
}

func (l *LocatedReader) Location() int {
	return l.location
}

//
// Bool
//

type EncodedBool byte

const (
	EncodedBoolUnknown EncodedBool = iota
	EncodedBoolFalse
	EncodedBoolTrue
)

func EncodeBool(w io.Writer, boolean bool) (err error) {
	b := EncodedBoolFalse
	if boolean {
		b = EncodedBoolTrue
	}

	_, err = w.Write([]byte{byte(b)})
	return
}

func DecodeBool(r io.Reader) (boolean bool, err error) {
	b := make([]byte, 1)
	_, err = r.Read(b)
	if err != nil {
		return
	}

	switch EncodedBool(b[0]) {
	case EncodedBoolFalse:
		boolean = false
	case EncodedBoolTrue:
		boolean = true
	default:
		err = fmt.Errorf("invalid boolean value: %d", b[0])
	}

	return
}

//
// Length
//

// TODO encode length with variable-sized encoding?
//      e.g. first byte starting with `0` is the last byte in the length
//      will usually save 3 bytes. the question is if it saves or costs encode and/or decode time

// EncodeLength encodes a non-negative length as a uint32.
// It uses 4 bytes.
func EncodeLength(w io.Writer, length int) (err error) {
	if length < 0 { // TODO is this safety check useful?
		return fmt.Errorf("cannot encode length below zero: %d", length)
	}

	l := uint32(length)

	return binary.Write(w, binary.BigEndian, l)
}

func DecodeLength(r io.Reader) (length int, err error) {
	b := make([]byte, 4)
	_, err = r.Read(b)
	if err != nil {
		return
	}

	asUint32 := binary.BigEndian.Uint32(b)
	length = int(asUint32)
	return
}

//
// Bytes
//

func EncodeBytes(w io.Writer, bytes []byte) (err error) {
	err = EncodeLength(w, len(bytes))
	if err != nil {
		return
	}
	_, err = w.Write(bytes)
	return
}

func DecodeBytes(r io.Reader) (bytes []byte, err error) {
	length, err := DecodeLength(r)
	if err != nil {
		return
	}

	bytes = make([]byte, length)

	_, err = r.Read(bytes)
	return
}

//
// String
//

func EncodeString(w io.Writer, s string) (err error) {
	return EncodeBytes(w, []byte(s))
}

func DecodeString(r io.Reader) (s string, err error) {
	b, err := DecodeBytes(r)
	if err != nil {
		return
	}
	s = string(b)
	return
}

//
// Address
//

func EncodeAddress[Address common.Address | cadence.Address](w io.Writer, a Address) (err error) {
	_, err = w.Write(a[:])
	return
}

func DecodeAddress(r io.Reader) (a common.Address, err error) {
	bytes := make([]byte, common.AddressLength)

	_, err = r.Read(bytes)
	if err != nil {
		return
	}

	return common.BytesToAddress(bytes)
}

//
// U?Int(8,16,32,64)
//

// TODO use a more efficient encoder than `binary` (they say to in their top source comment)

func EncodeNumber[T int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64](w io.Writer, i T) (err error) {
	return binary.Write(w, binary.BigEndian, i)
}

func DecodeNumber[T int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64](r io.Reader) (i T, err error) {
	err = binary.Read(r, binary.BigEndian, &i)
	return
}

//
// BigInt
//

func EncodeBigInt(w io.Writer, i *big.Int) (err error) {
	isNegative := i.Sign() == -1
	err = EncodeBool(w, isNegative)
	if err != nil {
		return
	}

	return EncodeBytes(w, i.Bytes())
}

func DecodeBigInt(r io.Reader) (i *big.Int, err error) {
	isNegative, err := DecodeBool(r)
	if err != nil {
		return
	}

	bytes, err := DecodeBytes(r)
	if err != nil {
		return
	}

	i = big.NewInt(0)
	i.SetBytes(bytes)
	if isNegative {
		i.Neg(i)
	}
	return
}

// Location

func EncodeLocation(w io.Writer, location common.Location) (err error) {
	switch concreteType := location.(type) {
	case common.AddressLocation:
		return EncodeAddressLocation(w, concreteType)
	case common.IdentifierLocation:
		return EncodeIdentifierLocation(w, concreteType)
	case common.ScriptLocation:
		return EncodeScriptLocation(w, concreteType)
	case common.StringLocation:
		return EncodeStringLocation(w, concreteType)
	case common.TransactionLocation:
		return EncodeTransactionLocation(w, concreteType)
	case common.REPLLocation:
		return EncodeREPLLocation(w)
	case nil:
		return EncodeNilLocation(w)
	default:
		return fmt.Errorf("unexpected location type: %s", concreteType)
	}
}

// The location prefixes are stored as strings but are always* a single ascii character,
// so they can be stored in a single byte.
// * The exception is the REPL location but its first ascii character is unique anyway.
func EncodeLocationPrefix(w io.Writer, prefix string) (err error) {
	char := prefix[0]
	_, err = w.Write([]byte{char})
	return
}

var NilLocationPrefix = "\x00"

// EncodeNilLocation encodes a value that indicates that no location is specified
func EncodeNilLocation(w io.Writer) (err error) {
	return EncodeLocationPrefix(w, NilLocationPrefix)
}

func EncodeAddressLocation(w io.Writer, t common.AddressLocation) (err error) {
	err = EncodeLocationPrefix(w, common.AddressLocationPrefix)
	if err != nil {
		return
	}

	err = EncodeAddress(w, t.Address)
	if err != nil {
		return
	}

	return EncodeString(w, t.Name)
}

func EncodeIdentifierLocation(w io.Writer, t common.IdentifierLocation) (err error) {
	err = EncodeLocationPrefix(w, common.IdentifierLocationPrefix)
	if err != nil {
		return
	}

	return EncodeString(w, string(t))
}

func EncodeScriptLocation(w io.Writer, t common.ScriptLocation) (err error) {
	err = EncodeLocationPrefix(w, common.ScriptLocationPrefix)
	if err != nil {
		return
	}

	_, err = w.Write(t[:])
	return
}

func EncodeStringLocation(w io.Writer, t common.StringLocation) (err error) {
	err = EncodeLocationPrefix(w, common.StringLocationPrefix)
	if err != nil {
		return
	}

	return EncodeString(w, string(t))
}

func EncodeTransactionLocation(w io.Writer, t common.TransactionLocation) (err error) {
	err = EncodeLocationPrefix(w, common.TransactionLocationPrefix)
	if err != nil {
		return
	}

	_, err = w.Write(t[:])
	return
}

func EncodeREPLLocation(w io.Writer) (err error) {
	return EncodeLocationPrefix(w, common.REPLLocationPrefix)
}

func DecodeLocation(r io.Reader, memoryGauge common.MemoryGauge) (location common.Location, err error) {
	prefix, err := DecodeLocationPrefix(r)

	switch prefix {
	case common.AddressLocationPrefix:
		return DecodeAddressLocation(r, memoryGauge)
	case common.IdentifierLocationPrefix:
		return DecodeIdentifierLocation(r, memoryGauge)
	case common.ScriptLocationPrefix:
		return DecodeScriptLocation(r, memoryGauge)
	case common.StringLocationPrefix:
		return DecodeStringLocation(r, memoryGauge)
	case common.TransactionLocationPrefix:
		return DecodeTransactionLocation(r, memoryGauge)
	case string(common.REPLLocationPrefix[0]):
		location = common.REPLLocation{}
	case NilLocationPrefix:
		return

	// TODO more locations
	default:
		err = fmt.Errorf("unknown location prefix: %s", prefix)
	}
	return
}

func DecodeLocationPrefix(r io.Reader) (prefix string, err error) {
	b := make([]byte, 1)
	_, err = r.Read(b)
	prefix = string(b)
	return
}

func DecodeAddressLocation(r io.Reader, memoryGauge common.MemoryGauge) (location common.AddressLocation, err error) {
	address, err := DecodeAddress(r)
	if err != nil {
		return
	}

	name, err := DecodeString(r)
	if err != nil {
		return
	}

	location = common.NewAddressLocation(memoryGauge, address, name)

	return
}

func DecodeIdentifierLocation(r io.Reader, memoryGauge common.MemoryGauge) (location common.IdentifierLocation, err error) {
	s, err := DecodeString(r)
	location = common.NewIdentifierLocation(memoryGauge, s)
	return
}

func DecodeScriptLocation(r io.Reader, memoryGauge common.MemoryGauge) (location common.ScriptLocation, err error) {
	byteArray := make([]byte, len(location)) // len(common.ScriptLocation) == 32
	_, err = r.Read(byteArray)
	if err != nil {
		return
	}

	for i, b := range byteArray {
		location[i] = b
	}

	location = common.NewScriptLocation(memoryGauge, byteArray)
	return
}

func DecodeStringLocation(r io.Reader, memoryGauge common.MemoryGauge) (location common.StringLocation, err error) {
	s, err := DecodeString(r)
	location = common.NewStringLocation(memoryGauge, s)
	return
}

func DecodeTransactionLocation(r io.Reader, memoryGauge common.MemoryGauge) (location common.TransactionLocation, err error) {
	byteArray := make([]byte, len(location)) // len(common.TransactionLocation) == 32
	_, err = r.Read(byteArray)
	if err != nil {
		return
	}

	for i, b := range byteArray {
		location[i] = b
	}

	location = common.NewTransactionLocation(memoryGauge, byteArray)
	return
}

//
// Misc
//

func Concat[T any](deep ...[]T) []T {
	length := 0
	for _, b := range deep {
		length += len(b)
	}

	flat := make([]T, 0, length)
	for _, b := range deep {
		flat = append(flat, b...)
	}

	return flat
}
