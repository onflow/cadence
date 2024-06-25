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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/errors"
)

const TransactionLocationPrefix = "t"

const TransactionIDLength = 32

// TransactionLocation
type TransactionLocation [TransactionIDLength]byte

var _ Location = TransactionLocation{}

func NewTransactionLocation(gauge MemoryGauge, identifier []byte) (location TransactionLocation) {
	UseMemory(gauge, NewBytesMemoryUsage(len(identifier)))
	copy(location[:], identifier)
	return
}

func (l TransactionLocation) TypeID(memoryGauge MemoryGauge, qualifiedIdentifier string) TypeID {
	return hexIDLocationTypeID(
		memoryGauge,
		TransactionLocationPrefix,
		TransactionIDLength,
		l[:],
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
	return hex.EncodeToString(l[:])
}

func (l TransactionLocation) Description() string {
	return fmt.Sprintf("transaction with ID %s", hex.EncodeToString(l[:]))
}

func (l TransactionLocation) ID() string {
	return fmt.Sprintf("%s.%s", TransactionLocationPrefix, l)
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

func init() {
	RegisterTypeIDDecoder(
		TransactionLocationPrefix,
		func(gauge MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error) {
			return decodeTransactionLocationTypeID(gauge, typeID)
		},
	)
}

func decodeTransactionLocationTypeID(gauge MemoryGauge, typeID string) (TransactionLocation, string, error) {

	const errorMessagePrefix = "invalid transaction location type ID"

	newError := func(message string) (TransactionLocation, string, error) {
		return TransactionLocation{}, "", errors.NewDefaultUserError("%s: %s", errorMessagePrefix, message)
	}

	if typeID == "" {
		return newError("missing prefix")
	}

	parts := strings.SplitN(typeID, ".", 3)

	partCount := len(parts)
	if partCount == 1 {
		return newError("missing location")
	}

	prefix := parts[0]

	if prefix != TransactionLocationPrefix {
		return TransactionLocation{}, "", errors.NewDefaultUserError(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			TransactionLocationPrefix,
			prefix,
		)
	}

	location, err := hex.DecodeString(parts[1])
	UseMemory(gauge, NewBytesMemoryUsage(len(location)))

	if err != nil {
		return TransactionLocation{}, "", errors.NewDefaultUserError(
			"%s: invalid location: %w",
			errorMessagePrefix,
			err,
		)
	}

	var qualifiedIdentifier string
	if partCount > 2 {
		qualifiedIdentifier = parts[2]
	}

	var result TransactionLocation
	copy(result[:], location)

	return result, qualifiedIdentifier, nil
}
