/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

	"github.com/onflow/cadence/errors"
)

const IdentifierLocationPrefix = "I"

// IdentifierLocation
type IdentifierLocation string

var _ Location = IdentifierLocation("")

func NewIdentifierLocation(gauge MemoryGauge, id string) IdentifierLocation {
	UseMemory(gauge, NewRawStringMemoryUsage(len(id)))
	return IdentifierLocation(id)
}

func (l IdentifierLocation) TypeID(memoryGauge MemoryGauge, qualifiedIdentifier string) TypeID {
	return idLocationTypeID(
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

func (l IdentifierLocation) Description() string {
	return string(l)
}

func (l IdentifierLocation) ID() string {
	return fmt.Sprintf("%s.%s", IdentifierLocationPrefix, l)
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

func decodeIdentifierLocationTypeID(_ MemoryGauge, typeID string) (IdentifierLocation, string, error) {

	const errorMessagePrefix = "invalid identifier location type ID"

	newError := func(message string) (IdentifierLocation, string, error) {
		return "", "", errors.NewDefaultUserError("%s: %s", errorMessagePrefix, message)
	}

	if typeID == "" {
		return newError("missing prefix")
	}

	parts := strings.SplitN(typeID, ".", 3)

	pieceCount := len(parts)
	if pieceCount == 1 {
		return newError("missing location")
	}

	prefix := parts[0]

	if prefix != IdentifierLocationPrefix {
		return "", "", errors.NewDefaultUserError(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			IdentifierLocationPrefix,
			prefix,
		)
	}

	location := IdentifierLocation(parts[1])

	var qualifiedIdentifier string
	if pieceCount > 2 {
		qualifiedIdentifier = parts[2]
	}

	return location, qualifiedIdentifier, nil
}
