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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const ScriptLocationPrefix = "s"

// ScriptLocation
//
type ScriptLocation []byte

func NewScriptLocation(gauge MemoryGauge, script []byte) ScriptLocation {
	UseMemory(gauge, NewBytesMemoryUsage(len(script)))
	return ScriptLocation(script)
}

func (l ScriptLocation) ID() LocationID {
	return l.MeteredID(nil)
}

func (l ScriptLocation) MeteredID(memoryGauge MemoryGauge) LocationID {
	return NewMeteredLocationID(
		memoryGauge,
		ScriptLocationPrefix,
		l.String(),
	)
}

func (l ScriptLocation) TypeID(memoryGauge MemoryGauge, qualifiedIdentifier string) TypeID {
	return NewMeteredTypeID(
		memoryGauge,
		ScriptLocationPrefix,
		l.String(),
		qualifiedIdentifier,
	)
}

func (l ScriptLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 3)

	if len(pieces) < 3 {
		return ""
	}

	return pieces[2]
}

func (l ScriptLocation) String() string {
	return hex.EncodeToString(l)
}

func (l ScriptLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type   string
		Script string
	}{
		Type:   "ScriptLocation",
		Script: l.String(),
	})
}

func init() {
	RegisterTypeIDDecoder(
		ScriptLocationPrefix,
		func(gauge MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error) {
			return decodeScriptLocationTypeID(gauge, typeID)
		},
	)
}

func decodeScriptLocationTypeID(gauge MemoryGauge, typeID string) (ScriptLocation, string, error) {

	const errorMessagePrefix = "invalid script location type ID"

	newError := func(message string) (ScriptLocation, string, error) {
		return nil, "", fmt.Errorf("%s: %s", errorMessagePrefix, message)
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

	if prefix != ScriptLocationPrefix {
		return nil, "", fmt.Errorf(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			ScriptLocationPrefix,
			prefix,
		)
	}

	location, err := hex.DecodeString(parts[1])
	UseMemory(gauge, NewBytesMemoryUsage(len(location)))

	if err != nil {
		return nil, "", fmt.Errorf(
			"%s: invalid location: %w",
			errorMessagePrefix,
			err,
		)
	}

	qualifiedIdentifier := parts[2]

	return location, qualifiedIdentifier, nil
}
