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

package common

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/errors"
)

// Location describes the origin of a Cadence script.
// This could be a file, a transaction, or a smart contract.
type Location interface {
	fmt.Stringer
	// TypeID returns a type ID for the given qualified identifier
	TypeID(memoryGauge MemoryGauge, qualifiedIdentifier string) TypeID
	// QualifiedIdentifier returns the qualified identifier for the given type ID
	QualifiedIdentifier(typeID TypeID) string
	// Description returns a human-readable description. For example, it can be used in error messages
	Description() string
}

// LocationsInSameAccount returns true if both locations are nil,
// if both locations are address locations when both locations have the same address,
// or otherwise if their IDs are the same.
func LocationsInSameAccount(first, second Location) bool {

	if first == nil {
		return second == nil
	}

	if second == nil {
		return false
	}

	if firstAddressLocation, ok := first.(AddressLocation); ok {

		secondAddressLocation, ok := second.(AddressLocation)
		if !ok {
			return false
		}

		// NOTE: only check address, ignore name
		return firstAddressLocation.Address == secondAddressLocation.Address
	}

	return first == second
}

// TypeID
type TypeID string

func NewTypeIDFromQualifiedName(memoryGauge MemoryGauge, location Location, qualifiedIdentifier string) TypeID {
	if location == nil {
		return TypeID(qualifiedIdentifier)
	}

	return location.TypeID(memoryGauge, qualifiedIdentifier)
}

// hexIDLocationTypeID returns a type ID in the format
// prefix '.' hex-encoded ID '.' qualifiedIdentifier
func hexIDLocationTypeID(
	memoryGauge MemoryGauge,
	prefix string,
	idLength int,
	id []byte,
	qualifiedIdentifier string,
) TypeID {
	var i int

	// prefix '.' hex-encoded ID '.' qualifiedIdentifier
	length := len(prefix) + 1 + hex.EncodedLen(idLength) + 1 + len(qualifiedIdentifier)

	UseMemory(memoryGauge, NewRawStringMemoryUsage(length))

	b := make([]byte, length)

	copy(b, prefix)
	i += len(prefix)

	b[i] = '.'
	i += 1

	hex.Encode(b[i:], id)
	i += idLength * 2

	b[i] = '.'
	i += 1

	copy(b[i:], qualifiedIdentifier)

	return TypeID(b)
}

// idLocationTypeID returns a type ID in the format
// prefix '.' ID '.' qualifiedIdentifier
func idLocationTypeID(
	memoryGauge MemoryGauge,
	prefix string,
	id string,
	qualifiedIdentifier string,
) TypeID {
	var i int

	// prefix '.' ID '.' qualifiedIdentifier
	length := len(prefix) + 1 + len(id) + 1 + len(qualifiedIdentifier)

	UseMemory(memoryGauge, NewRawStringMemoryUsage(length))

	b := make([]byte, length)

	copy(b, prefix)
	i += len(prefix)

	b[i] = '.'
	i += 1

	copy(b[i:], id)
	i += len(id)

	b[i] = '.'
	i += 1

	copy(b[i:], qualifiedIdentifier)

	return TypeID(b)
}

type TypeIDDecoder func(gauge MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error)

var typeIDDecoders = map[string]TypeIDDecoder{}

func RegisterTypeIDDecoder(prefix string, decoder TypeIDDecoder) {
	if _, ok := typeIDDecoders[prefix]; ok {
		panic(errors.NewUnexpectedError("cannot register type ID decoder for already registered prefix: %s", prefix))
	}
	typeIDDecoders[prefix] = decoder
}

func DecodeTypeID(gauge MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error) {
	pieces := strings.Split(typeID, ".")

	if len(pieces) < 1 {
		return nil, "", errors.NewDefaultUserError("invalid type ID: missing type name")
	}

	prefix := pieces[0]

	decoder, ok := typeIDDecoders[prefix]
	if !ok {
		// If there are no decoders registered under the first piece if ID, then it could be:
		//    (1) A native composite type
		//    (2) An invalid type/prefix
		// Either way, return the typeID as the identifier with a nil location. Then, if it is case (1),
		// it will correctly continue at the downstream code. If it is (2), downstream code will throw
		// an invalid type error.
		return nil, typeID, nil
	}

	return decoder(gauge, typeID)
}

// HasLocation

type HasLocation interface {
	ImportLocation() Location
}
