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

package runtime

import (
	"encoding/hex"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
)

type (
	Location        = ast.Location
	LocationID      = ast.LocationID
	TypeID          = ast.TypeID
	StringLocation  = ast.StringLocation
	AddressLocation = ast.AddressLocation
	Identifier      = ast.Identifier
)

const (
	IdentifierLocationPrefix  = ast.IdentifierLocationPrefix
	StringLocationPrefix      = ast.StringLocationPrefix
	AddressLocationPrefix     = ast.AddressLocationPrefix
	TransactionLocationPrefix = "t"
	ScriptLocationPrefix      = "s"
)

// TransactionLocation

type TransactionLocation []byte

func (l TransactionLocation) ID() ast.LocationID {
	return ast.NewLocationID(
		TransactionLocationPrefix,
		l.String(),
	)
}

func (l TransactionLocation) TypeID(qualifiedIdentifier string) TypeID {
	return ast.NewTypeID(
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

// ScriptLocation

type ScriptLocation []byte

func (l ScriptLocation) ID() ast.LocationID {
	return ast.NewLocationID(
		ScriptLocationPrefix,
		l.String(),
	)
}

func (l ScriptLocation) TypeID(qualifiedIdentifier string) TypeID {
	return ast.NewTypeID(
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

// FileLocation

type FileLocation string

func (l FileLocation) ID() ast.LocationID {
	return LocationID(l.String())
}

func (l FileLocation) TypeID(qualifiedIdentifier string) TypeID {
	return ast.NewTypeID(
		l.String(),
		qualifiedIdentifier,
	)
}

func (l FileLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 2)

	if len(pieces) < 2 {
		return ""
	}

	return pieces[1]
}

func (l FileLocation) String() string {
	return string(l)
}

// REPLLocation

type REPLLocation struct{}

func (l REPLLocation) ID() LocationID {
	return LocationID(l.String())
}

func (l REPLLocation) TypeID(qualifiedIdentifier string) TypeID {
	return ast.NewTypeID(
		l.String(),
		qualifiedIdentifier,
	)
}

func (l REPLLocation) QualifiedIdentifier(typeID TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 2)

	if len(pieces) < 2 {
		return ""
	}

	return pieces[1]
}

func (l REPLLocation) String() string {
	return "REPL"
}
