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
	"strings"

	"github.com/onflow/cadence/runtime/errors"
)

const REPLLocationPrefix = "REPL"

// REPLLocation
type REPLLocation struct{}

var _ Location = REPLLocation{}

func (l REPLLocation) TypeID(memoryGauge MemoryGauge, qualifiedIdentifier string) TypeID {
	var i int

	// REPLLocationPrefix '.' qualifiedIdentifier
	length := len(REPLLocationPrefix) + 1 + len(qualifiedIdentifier)

	UseMemory(memoryGauge, NewRawStringMemoryUsage(length))

	b := make([]byte, length)

	copy(b, REPLLocationPrefix)
	i += len(REPLLocationPrefix)

	b[i] = '.'
	i += 1

	copy(b[i:], qualifiedIdentifier)

	return TypeID(b)
}

func (l REPLLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 2)

	if len(pieces) < 2 {
		return ""
	}

	return pieces[1]
}

func (l REPLLocation) String() string {
	return REPLLocationPrefix
}

func (l REPLLocation) Description() string {
	return REPLLocationPrefix
}

func (l REPLLocation) ID() string {
	return REPLLocationPrefix
}

func (l REPLLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type string
	}{
		Type: "REPLLocation",
	})
}

func init() {
	RegisterTypeIDDecoder(
		REPLLocationPrefix,
		func(_ MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error) {
			return decodeREPLLocationTypeID(typeID)
		},
	)
}

func decodeREPLLocationTypeID(typeID string) (REPLLocation, string, error) {

	const errorMessagePrefix = "invalid REPL location type ID"

	newError := func(message string) (REPLLocation, string, error) {
		return REPLLocation{}, "", errors.NewDefaultUserError("%s: %s", errorMessagePrefix, message)
	}

	if typeID == "" {
		return newError("missing prefix")
	}

	parts := strings.SplitN(typeID, ".", 2)

	prefix := parts[0]

	if prefix != REPLLocationPrefix {
		return REPLLocation{}, "", errors.NewDefaultUserError(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			REPLLocationPrefix,
			prefix,
		)
	}

	pieceCount := len(parts)
	var qualifiedIdentifier string
	if pieceCount > 1 {
		qualifiedIdentifier = parts[1]
	}

	return REPLLocation{}, qualifiedIdentifier, nil
}
