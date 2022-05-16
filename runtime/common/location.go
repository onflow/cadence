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

package common

import (
	"errors"
	"fmt"
	"strings"
)

// Location describes the origin of a Cadence script.
// This could be a file, a transaction, or a smart contract.
//
type Location interface {
	fmt.Stringer
	// ID returns the canonical ID for this import location.
	ID() LocationID
	// MeteredID returns the canonical ID for this import location, and the generated ID is memory metered.
	MeteredID(memoryGauge MemoryGauge) LocationID
	// TypeID returns a type ID for the given qualified identifier
	TypeID(memoryGauge MemoryGauge, qualifiedIdentifier string) TypeID
	// QualifiedIdentifier returns the qualified identifier for the given type ID
	QualifiedIdentifier(typeID TypeID) string
}

// LocationsMatch returns true if both locations are nil or their IDs are the same.
//
func LocationsMatch(first, second Location) bool {

	if first == nil {
		return second == nil
	}

	if second == nil {
		return false
	}

	return first.ID() == second.ID()
}

// LocationsInSameAccount returns true if both locations are nil,
// if both locations are address locations when both locations have the same address,
// or otherwise if their IDs are the same.
//
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

	return first.ID() == second.ID()
}

// LocationID
//
type LocationID string

func NewLocationID(parts ...string) LocationID {
	return LocationID(strings.Join(parts, "."))
}

func NewMeteredLocationID(memoryGauge MemoryGauge, parts ...string) LocationID {
	jointString := joinStrings(memoryGauge, parts)
	return LocationID(jointString)
}

func joinStrings(memoryGauge MemoryGauge, parts []string) string {
	l := 0
	for _, part := range parts {
		l += len(part) + 1
	}
	UseMemory(memoryGauge, NewRawStringMemoryUsage(l))

	return strings.Join(parts, ".")
}

// TypeID
//
type TypeID string

func NewTypeID(parts ...string) TypeID {
	return TypeID(strings.Join(parts, "."))
}

func NewMeteredTypeID(memoryGauge MemoryGauge, parts ...string) TypeID {
	jointString := joinStrings(memoryGauge, parts)
	return TypeID(jointString)
}

func NewTypeIDFromQualifiedName(location Location, qualifiedIdentifier string) TypeID {
	if location == nil {
		return TypeID(qualifiedIdentifier)
	}

	return location.TypeID(nil, qualifiedIdentifier)
}

type TypeIDDecoder func(gauge MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error)

var typeIDDecoders = map[string]TypeIDDecoder{}

func RegisterTypeIDDecoder(prefix string, decoder TypeIDDecoder) {
	if _, ok := typeIDDecoders[prefix]; ok {
		panic(fmt.Errorf("cannot register type ID decoder for already registered prefix: %s", prefix))
	}
	typeIDDecoders[prefix] = decoder
}

func DecodeTypeID(gauge MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error) {
	pieces := strings.Split(typeID, ".")

	if len(pieces) < 1 {
		return nil, "", errors.New("invalid type ID: missing type name")
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

// HasImportLocation

type HasImportLocation interface {
	ImportLocation() Location
}
