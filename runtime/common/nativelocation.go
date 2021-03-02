/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

const NativeLocationPrefix = "N"

// NativeLocation
//
type NativeLocation struct{}

func (l NativeLocation) ID() LocationID {
	return NativeLocationPrefix
}

func (l NativeLocation) TypeID(qualifiedIdentifier string) TypeID {
	return NewTypeID(
		NativeLocationPrefix,
		qualifiedIdentifier,
	)
}

func (l NativeLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 2)

	if len(pieces) < 2 {
		return ""
	}

	return pieces[1]
}

func (l NativeLocation) String() string {
	return NativeLocationPrefix
}

func (l NativeLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type string
	}{
		Type: "NativeLocation",
	})
}

func init() {
	RegisterTypeIDDecoder(
		NativeLocationPrefix,
		func(typeID string) (location Location, qualifiedIdentifier string, err error) {
			return decodeNativeLocationID(typeID)
		},
	)
}

func decodeNativeLocationID(typeID string) (NativeLocation, string, error) {

	const errorMessagePrefix = "invalid native location type ID"

	newError := func(message string) (NativeLocation, string, error) {
		return NativeLocation{}, "", fmt.Errorf("%s: %s", errorMessagePrefix, message)
	}

	if typeID == "" {
		return newError("missing prefix")
	}

	parts := strings.SplitN(typeID, ".", 2)

	pieceCount := len(parts)
	if pieceCount == 1 {
		return newError("missing qualified identifier")
	}

	prefix := parts[0]

	if prefix != NativeLocationPrefix {
		return NativeLocation{}, "", fmt.Errorf(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			NativeLocationPrefix,
			prefix,
		)
	}

	qualifiedIdentifier := parts[1]

	return NativeLocation{}, qualifiedIdentifier, nil
}
