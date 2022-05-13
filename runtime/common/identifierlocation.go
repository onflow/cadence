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
	"encoding/json"
	"fmt"
	"strings"
)

const IdentifierLocationPrefix = "I"

// IdentifierLocation
//
type IdentifierLocation string

func NewIdentifierLocation(gauge MemoryGauge, id string) IdentifierLocation {
	UseMemory(gauge, NewRawStringMemoryUsage(len(id)))
	return IdentifierLocation(id)
}

func (l IdentifierLocation) ID() LocationID {
	return l.MeteredID(nil)
}

func (l IdentifierLocation) MeteredID(memoryGauge MemoryGauge) LocationID {
	return NewMeteredLocationID(
		memoryGauge,
		IdentifierLocationPrefix,
		string(l),
	)
}

func (l IdentifierLocation) TypeID(memoryGauge MemoryGauge, qualifiedIdentifier string) TypeID {
	return NewMeteredTypeID(
		memoryGauge,
		IdentifierLocationPrefix,
		string(l),
		qualifiedIdentifier,
	)
}

func (l IdentifierLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 3)

	if len(pieces) < 3 {
		return ""
	}

	return pieces[2]
}

func (l IdentifierLocation) String() string {
	return string(l)
}

func (l IdentifierLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       string
		Identifier string
	}{
		Type:       "IdentifierLocation",
		Identifier: string(l),
	})
}

func init() {
	RegisterTypeIDDecoder(
		IdentifierLocationPrefix,
		func(gauge MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error) {
			return decodeIdentifierLocationTypeID(gauge, typeID)
		},
	)
}

func decodeIdentifierLocationTypeID(gauge MemoryGauge, typeID string) (IdentifierLocation, string, error) {

	const errorMessagePrefix = "invalid identifier location type ID"

	newError := func(message string) (IdentifierLocation, string, error) {
		return "", "", fmt.Errorf("%s: %s", errorMessagePrefix, message)
	}

	if typeID == "" {
		return newError("missing prefix")
	}

	parts := strings.SplitN(typeID, ".", 3)

	pieceCount := len(parts)
	switch pieceCount {
	case 1:
		return newError("missing location")
	case 2:
		return newError("missing qualified identifier")
	}

	prefix := parts[0]

	if prefix != IdentifierLocationPrefix {
		return "", "", fmt.Errorf(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			IdentifierLocationPrefix,
			prefix,
		)
	}

	location := IdentifierLocation(parts[1])
	qualifiedIdentifier := parts[2]

	return location, qualifiedIdentifier, nil
}
