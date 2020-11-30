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
	"encoding/json"
	"strings"
)

const IdentifierLocationPrefix = "I"

// IdentifierLocation
//
type IdentifierLocation string

func (l IdentifierLocation) ID() LocationID {
	return NewLocationID(
		IdentifierLocationPrefix,
		string(l),
	)
}

func (l IdentifierLocation) TypeID(qualifiedIdentifier string) TypeID {
	return NewTypeID(
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

func (l IdentifierLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       string
		Identifier string
	}{
		Type:       "IdentifierLocation",
		Identifier: string(l),
	})
}
