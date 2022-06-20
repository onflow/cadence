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

const StringLocationPrefix = "S"

// StringLocation
//
type StringLocation string

func NewStringLocation(gauge MemoryGauge, id string) StringLocation {
	UseMemory(gauge, NewRawStringMemoryUsage(len(id)))
	return StringLocation(id)
}

func (l StringLocation) ID() LocationID {
	return l.MeteredID(nil)
}

func (l StringLocation) MeteredID(memoryGauge MemoryGauge) LocationID {
	return NewMeteredLocationID(
		memoryGauge,
		StringLocationPrefix,
		string(l),
	)
}

func (l StringLocation) TypeID(memoryGauge MemoryGauge, qualifiedIdentifier string) TypeID {
	return NewMeteredTypeID(
		memoryGauge,
		StringLocationPrefix,
		string(l),
		qualifiedIdentifier,
	)
}

func (l StringLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 3)

	if len(pieces) < 3 {
		return ""
	}

	return pieces[2]
}

func (l StringLocation) String() string {
	return string(l)
}

func (l StringLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type   string
		String string
	}{
		Type:   "StringLocation",
		String: string(l),
	})
}

func init() {
	RegisterTypeIDDecoder(
		StringLocationPrefix,
		func(gauge MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error) {
			return decodeStringLocationTypeID(gauge, typeID)
		},
	)
}

func decodeStringLocationTypeID(gauge MemoryGauge, typeID string) (StringLocation, string, error) {

	const errorMessagePrefix = "invalid string location type ID"

	newError := func(message string) (StringLocation, string, error) {
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

	if prefix != StringLocationPrefix {
		return "", "", fmt.Errorf(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			StringLocationPrefix,
			prefix,
		)
	}

	location := NewStringLocation(gauge, parts[1])
	qualifiedIdentifier := parts[2]

	return location, qualifiedIdentifier, nil
}
