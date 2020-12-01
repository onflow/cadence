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
	"fmt"
	"strings"
)

const AddressLocationPrefix = "A"

// AddressLocation is the location of a contract/contract interface at an address
//
type AddressLocation struct {
	Address Address
	Name    string
}

func (l AddressLocation) String() string {
	if l.Name == "" {
		return l.Address.String()
	}

	return fmt.Sprintf(
		"%s.%s",
		l.Address.String(),
		l.Name,
	)
}

func (l AddressLocation) ID() LocationID {
	if l.Name == "" {
		return NewLocationID(
			AddressLocationPrefix,
			l.Address.Hex(),
		)
	}

	return NewLocationID(
		AddressLocationPrefix,
		l.Address.Hex(),
		l.Name,
	)
}

func (l AddressLocation) TypeID(qualifiedIdentifier string) TypeID {
	return NewTypeID(
		AddressLocationPrefix,
		l.Address.Hex(),
		qualifiedIdentifier,
	)
}

func (l AddressLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 3)

	if len(pieces) < 3 {
		return ""
	}

	return pieces[2]
}

func (l AddressLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    string
		Address string
		Name    string
	}{
		Type:    "AddressLocation",
		Address: l.Address.ShortHexWithPrefix(),
		Name:    l.Name,
	})
}

func init() {
	RegisterTypeIDDecoder(
		AddressLocationPrefix,
		func(typeID string) (location Location, qualifiedIdentifier string, err error) {
			return decodeAddressLocationTypeID(typeID)
		},
	)
}

func decodeAddressLocationTypeID(typeID string) (AddressLocation, string, error) {

	const errorMessagePrefix = "invalid address location type ID"

	newError := func(message string) (AddressLocation, string, error) {
		return AddressLocation{}, "", fmt.Errorf("%s: %s", errorMessagePrefix, message)
	}

	if typeID == "" {
		return newError("missing prefix")
	}

	parts := strings.SplitN(typeID, ".", 4)

	pieceCount := len(parts)
	switch pieceCount {
	case 1:
		return newError("missing location")
	case 2:
		return newError("missing qualified identifier")
	}

	prefix := parts[0]

	if prefix != AddressLocationPrefix {
		return AddressLocation{}, "", fmt.Errorf(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			AddressLocationPrefix,
			prefix,
		)
	}

	address, err := hex.DecodeString(parts[1])
	if err != nil {
		return AddressLocation{}, "", fmt.Errorf(
			"%s: invalid address: %w",
			errorMessagePrefix,
			err,
		)
	}

	name := parts[2]
	qualifiedIdentifier := name

	if pieceCount > 3 {
		qualifiedIdentifier = fmt.Sprintf("%s.%s", qualifiedIdentifier, parts[3])
	}

	location := AddressLocation{
		Address: BytesToAddress(address),
		Name:    name,
	}

	return location, qualifiedIdentifier, nil
}
