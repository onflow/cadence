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
	"encoding/hex"
	"encoding/json"
	"strings"
)

const TransactionLocationPrefix = "t"

// TransactionLocation
//
type TransactionLocation []byte

func (l TransactionLocation) ID() LocationID {
	return NewLocationID(
		TransactionLocationPrefix,
		l.String(),
	)
}

func (l TransactionLocation) TypeID(qualifiedIdentifier string) TypeID {
	return NewTypeID(
		TransactionLocationPrefix,
		l.String(),
		qualifiedIdentifier,
	)
}

func (l TransactionLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 3)

	if len(pieces) < 3 {
		return ""
	}

	return pieces[2]
}

func (l TransactionLocation) String() string {
	return hex.EncodeToString(l)
}

func (l TransactionLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type        string
		Transaction string
	}{
		Type:        "TransactionLocation",
		Transaction: l.String(),
	})
}
