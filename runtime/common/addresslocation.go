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

	"github.com/onflow/cadence/runtime/errors"
)

const AddressLocationPrefix = "A"

// AddressLocation is the location of a contract/contract interface at an address
//
type AddressLocation struct {
	Address Address
	Name    string
}

func NewAddressLocation(gauge MemoryGauge, addr Address, name string) AddressLocation {
	UseMemory(gauge, NewConstantMemoryUsage(MemoryKindAddressLocation))
	return AddressLocation{
		Address: addr,
		Name:    name,
	}
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
	return l.MeteredID(nil)
}

func (l AddressLocation) MeteredID(memoryGauge MemoryGauge) LocationID {
	if l.Name == "" {
		return NewMeteredLocationID(
			memoryGauge,
			AddressLocationPrefix,
			l.Address.Hex(),
		)
	}

	return NewMeteredLocationID(
		memoryGauge,
		AddressLocationPrefix,
		l.Address.Hex(),
		l.Name,
	)
}

func (l AddressLocation) TypeID(memoryGauge MemoryGauge, qualifiedIdentifier string) TypeID {
	return NewMeteredTypeID(
		memoryGauge,
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
		Address: l.Address.HexWithPrefix(),
		Name:    l.Name,
	})
}

func init() {
	RegisterTypeIDDecoder(
		AddressLocationPrefix,
		func(gauge MemoryGauge, typeID string) (location Location, qualifiedIdentifier string, err error) {
			return decodeAddressLocationTypeID(gauge, typeID)
		},
	)
}

func decodeAddressLocationTypeID(gauge MemoryGauge, typeID string) (AddressLocation, string, error) {

	const errorMessagePrefix = "invalid address location type ID"

	newError := func(message string) (AddressLocation, string, error) {
		return AddressLocation{}, "", fmt.Errorf("%s: %s", errorMessagePrefix, message)
	}

	if typeID == "" {
		return newError("missing prefix")
	}

	// The type ID with an address location must have the format `prefix>.<location>.<qualifiedIdentifier>`,
	// where `<qualifiedIdentifier>` itself is one or more identifiers separated by a dot.
	//
	// `<prefix>` must be AddressLocationPrefix.
	// `<location>` must be a hex string.
	//  The first part of `<qualifiedIdentifier>` is also the contract name.
	//
	// So we split by at most 4 components – we don't need to split `<qualifiedIdentifier>` completely,
	// just the first part for the name, and the rest.

	parts := strings.SplitN(typeID, ".", 4)

	// Report an appropriate error message for the two invalid count cases.

	partCount := len(parts)
	switch partCount {
	case 0:
		// strings.SplitN will always return at minimum one item,
		// even for an empty type ID.
		panic(errors.NewUnreachableError())
	case 1:
		return newError("missing location")
	case 2:
		return newError("missing qualified identifier")
	case 3, 4:
		break
	default:
		// strings.SplitN will never return more than 4 parts
		panic(errors.NewUnreachableError())
	}

	// `<prefix>`, the first part, must be AddressLocationPrefix.

	prefix := parts[0]

	if prefix != AddressLocationPrefix {
		return AddressLocation{}, "", fmt.Errorf(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			AddressLocationPrefix,
			prefix,
		)
	}

	// `<location>`, the second part, must be a hex string.

	rawAddress, err := hex.DecodeString(parts[1])
	if err != nil {
		return AddressLocation{}, "", fmt.Errorf(
			"%s: invalid address: %w",
			errorMessagePrefix,
			err,
		)
	}

	var name string
	var qualifiedIdentifier string

	switch partCount {
	case 3:
		// If there are only 3 parts,
		// then `<qualifiedIdentifier>` is both the contract name and the qualified identifier.

		name = parts[2]
		qualifiedIdentifier = name

	case 4:
		// If there are 4 parts,
		// then `<qualifiedIdentifier>` contains both a contract name and a remainder.
		// In this case, the third part is the contract name,
		// and the qualified identifier is reconstructed from the contract name and the remainder (the fourth part).

		name = parts[2]
		qualifiedIdentifier = fmt.Sprintf("%s.%s", name, parts[3])

	default:
		panic(errors.NewUnreachableError())
	}

	address, err := BytesToAddress(rawAddress)
	if err != nil {
		return AddressLocation{}, "", err
	}

	location := NewAddressLocation(gauge, address, name)

	return location, qualifiedIdentifier, nil
}
