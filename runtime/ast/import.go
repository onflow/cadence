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

package ast

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/common"
)

// Identifier

type Identifier struct {
	Identifier string
	Pos        Position
}

func (i Identifier) String() string {
	return i.Identifier
}

func (i Identifier) StartPosition() Position {
	return i.Pos
}

func (i Identifier) EndPosition() Position {
	length := len(i.Identifier)
	return i.Pos.Shifted(length - 1)
}

func (i Identifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Identifier string
		Range
	}{
		Identifier: i.Identifier,
		Range:      NewRangeFromPositioned(i),
	})
}

// ImportDeclaration

type ImportDeclaration struct {
	Identifiers []Identifier
	Location    Location
	LocationPos Position
	Range
}

func (*ImportDeclaration) isDeclaration() {}

func (*ImportDeclaration) isStatement() {}

func (d *ImportDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitImportDeclaration(d)
}

func (d *ImportDeclaration) DeclarationIdentifier() *Identifier {
	return nil
}

func (d *ImportDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindImport
}

func (d *ImportDeclaration) DeclarationAccess() Access {
	return AccessNotSpecified
}

func (d *ImportDeclaration) MarshalJSON() ([]byte, error) {
	type Alias ImportDeclaration
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "ImportDeclaration",
		Alias: (*Alias)(d),
	})
}

// Location describes the origin of a Cadence script.
// This could be a file, a transaction, or a smart contract.
//
type Location interface {
	// ID returns the canonical ID for this import location.
	ID() LocationID
}

func LocationsMatch(first, second Location) bool {
	if first == nil && second == nil {
		return true
	}

	if (first == nil && second != nil) || (first != nil && second == nil) {
		return false
	}

	// if one is an Address contract location, compare the address location instead
	firstLocation, a1acl := first.(AddressContractLocation)
	secondLocation, a2acl := second.(AddressContractLocation)
	if a1acl != a2acl {
		if a1acl {
			first = firstLocation.AddressLocation
		}
		if a2acl {
			second = secondLocation.AddressLocation
		}
	}

	return first.ID() == second.ID()
}

func LocationFromTypeID(typeID string) Location {
	pieces := strings.Split(typeID, ".")

	if len(pieces) < 3 {
		return nil
	}

	switch pieces[0] {
	case IdentifierLocationPrefix:
		return IdentifierLocation(pieces[1])

	case StringLocationPrefix:
		return StringLocation(pieces[1])

	case AddressLocationPrefix:
		address, err := hex.DecodeString(pieces[1])
		if err != nil {
			return nil
		}

		return AddressLocation(address)

	case AddressContractLocationPrefix:
		address, err := hex.DecodeString(pieces[1])
		if err != nil {
			return nil
		}

		return AddressContractLocation{
			AddressLocation: address,
			Name:            pieces[2],
		}
	}

	return nil
}

// LocationID

type LocationID string

func NewLocationID(parts ...string) LocationID {
	return LocationID(strings.Join(parts, "."))
}

// IdentifierLocation

const IdentifierLocationPrefix = "I"

type IdentifierLocation string

func (l IdentifierLocation) ID() LocationID {
	return NewLocationID(IdentifierLocationPrefix, string(l))
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

// StringLocation

const StringLocationPrefix = "S"

type StringLocation string

func (l StringLocation) ID() LocationID {
	return NewLocationID(StringLocationPrefix, string(l))
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

// AddressLocation

const AddressLocationPrefix = "A"

type AddressLocation []byte

func (l AddressLocation) String() string {
	return l.ToAddress().String()
}

func (l AddressLocation) ID() LocationID {
	return NewLocationID(AddressLocationPrefix, l.ToAddress().Hex())
}

func (l AddressLocation) ToAddress() common.Address {
	return common.BytesToAddress(l)
}

func (l AddressLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    string
		Address string
	}{
		Type:    "AddressLocation",
		Address: l.ToAddress().ShortHexWithPrefix(),
	})
}

const AddressContractLocationPrefix = "AC"

// AddressContractLocation is the location of a contract/contract interface at an address

type AddressContractLocation struct {
	AddressLocation AddressLocation
	Name            string
}

func (l AddressContractLocation) String() string {
	return fmt.Sprintf("%s.%s",
		l.AddressLocation.String(),
		l.Name,
	)
}

func (l AddressContractLocation) ID() LocationID {
	return NewLocationID(
		AddressContractLocationPrefix,
		l.AddressLocation.ToAddress().Hex(),
		l.Name,
	)
}

func (l AddressContractLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    string
		Address string
		Name    string
	}{
		Type:    "AddressContractLocation",
		Address: l.AddressLocation.ToAddress().ShortHexWithPrefix(),
		Name:    l.Name,
	})
}

// HasImportLocation

type HasImportLocation interface {
	ImportLocation() Location
}
