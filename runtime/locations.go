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

	"github.com/onflow/cadence/runtime/ast"
)

type (
	Location                = ast.Location
	LocationID              = ast.LocationID
	StringLocation          = ast.StringLocation
	AddressLocation         = ast.AddressLocation
	AddressContractLocation = ast.AddressContractLocation
	Identifier              = ast.Identifier
)

const (
	IdentifierLocationPrefix      = ast.IdentifierLocationPrefix
	StringLocationPrefix          = ast.StringLocationPrefix
	AddressLocationPrefix         = ast.AddressLocationPrefix
	AddressContractLocationPrefix = ast.AddressContractLocationPrefix
	TransactionLocationPrefix     = "t"
	ScriptLocationPrefix          = "s"
)

// TransactionLocation

type TransactionLocation []byte

func (l TransactionLocation) ID() ast.LocationID {
	return ast.NewLocationID(TransactionLocationPrefix, l.String())
}

func (l TransactionLocation) String() string {
	return hex.EncodeToString(l)
}

// ScriptLocation

type ScriptLocation []byte

func (l ScriptLocation) ID() ast.LocationID {
	return ast.NewLocationID(ScriptLocationPrefix, l.String())
}

func (l ScriptLocation) String() string {
	return hex.EncodeToString(l)
}

// FileLocation

type FileLocation string

func (l FileLocation) ID() ast.LocationID {
	return LocationID(l.String())
}

func (l FileLocation) String() string {
	return string(l)
}

// REPLLocation

type REPLLocation struct{}

func (l REPLLocation) ID() LocationID {
	return LocationID(l.String())
}

func (l REPLLocation) String() string {
	return "REPL"
}

// UndeployedContractLocation

type UndeployedContractLocation struct{}

func (l UndeployedContractLocation) ID() LocationID {
	return LocationID(l.String())
}

func (l UndeployedContractLocation) String() string {
	return "UndeployedContract"
}
