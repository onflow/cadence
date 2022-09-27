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
	"fmt"
	"io"

	"github.com/onflow/cadence/runtime/common"
)

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
		return CodecError(fmt.Sprintf("unexpected location type: %s", concreteType))
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

	default:
		err = CodecError(fmt.Sprintf("unknown location prefix: %s", prefix))
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
