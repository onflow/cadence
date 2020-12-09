/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	// TypeID returns a type ID for the given qualified identifier
	TypeID(qualifiedIdentifier string) TypeID
	// QualifiedIdentifier returns the qualified identifier for the given type ID
	QualifiedIdentifier(typeID TypeID) string
}

func LocationsMatch(first, second Location) bool {
	if first == nil && second == nil {
		return true
	}

	if (first == nil && second != nil) || (first != nil && second == nil) {
		return false
	}

	return first.ID() == second.ID()
}

// LocationID
//
type LocationID string

func NewLocationID(parts ...string) LocationID {
	return LocationID(strings.Join(parts, "."))
}

// TypeID
//
type TypeID string

func NewTypeID(parts ...string) TypeID {
	return TypeID(strings.Join(parts, "."))
}

type TypeIDDecoder func(typeID string) (location Location, qualifiedIdentifier string, err error)

var typeIDDecoders = map[string]TypeIDDecoder{}

func RegisterTypeIDDecoder(prefix string, decoder TypeIDDecoder) {
	if _, ok := typeIDDecoders[prefix]; ok {
		panic(fmt.Errorf("cannot register type ID decoder for already registered prefix: %s", prefix))
	}
	typeIDDecoders[prefix] = decoder
}

func DecodeTypeID(typeID string) (location Location, qualifiedIdentifier string, err error) {
	pieces := strings.Split(typeID, ".")

	if len(pieces) < 1 {
		return nil, "", errors.New("invalid type ID: missing prefix")
	}

	prefix := pieces[0]

	decoder, ok := typeIDDecoders[prefix]
	if !ok {
		return nil, "", fmt.Errorf("invalid type ID: cannot decode prefix %q", prefix)
	}

	return decoder(typeID)
}

// HasImportLocation

type HasImportLocation interface {
	ImportLocation() Location
}
