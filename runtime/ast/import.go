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
	// TypeID returns a type ID for the given qualified identifier
	TypeID(qualifiedIdentifier string) TypeID
	// QualifiedIdentifier returns the qualified identifier for the given type ID
	QualifiedIdentifier(typeID TypeID) string
}

func LocationsMatch(first, second Location) bool {
	if first == nil && second == nil {
		return true
	}

	if (first == nil && second != nil) || (first != nil && second == nil) {
		return false
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

		var name string
		if len(pieces) > 2 {
			name = pieces[2]
		}

		return AddressLocation{
			Address: common.BytesToAddress(address),
			Name:    name,
		}
	}

	return nil
}

// LocationID
//
type LocationID string

func NewLocationID(parts ...string) LocationID {
	return LocationID(strings.Join(parts, "."))
}

// TypeID
//
type TypeID string

func NewTypeID(parts ...string) TypeID {
	return TypeID(strings.Join(parts, "."))
}

// IdentifierLocation
//
const IdentifierLocationPrefix = "I"

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

const StringLocationPrefix = "S"

// StringLocation
//
type StringLocation string

func (l StringLocation) ID() LocationID {
	return NewLocationID(
		StringLocationPrefix,
		string(l),
	)
}

func (l StringLocation) TypeID(qualifiedIdentifier string) TypeID {
	return NewTypeID(
		StringLocationPrefix,
		string(l),
		qualifiedIdentifier,
	)
}

func (l StringLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 3)

	if len(pieces) < 3 {
		return ""
	}

	return pieces[2]
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

const AddressLocationPrefix = "A"

// AddressLocation is the location of a contract/contract interface at an address
//
type AddressLocation struct {
	Address common.Address
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

// HasImportLocation

type HasImportLocation interface {
	ImportLocation() Location
}
