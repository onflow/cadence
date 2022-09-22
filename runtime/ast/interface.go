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

package ast

import (
	"encoding/json"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
)

// InterfaceDeclaration

type InterfaceDeclaration struct {
	Access        Access
	CompositeKind common.CompositeKind
	Identifier    Identifier
	Conformances  []*NominalType
	Members       *Members
	DocString     string
	Range
}

var _ Element = &InterfaceDeclaration{}
var _ Declaration = &InterfaceDeclaration{}
var _ Statement = &InterfaceDeclaration{}
var _ HasConformance = &CompositeDeclaration{}

func NewInterfaceDeclaration(
	gauge common.MemoryGauge,
	access Access,
	compositeKind common.CompositeKind,
	identifier Identifier,
	conformances []*NominalType,
	members *Members,
	docString string,
	declRange Range,
) *InterfaceDeclaration {
	common.UseMemory(gauge, common.InterfaceDeclarationMemoryUsage)

	return &InterfaceDeclaration{
		Access:        access,
		CompositeKind: compositeKind,
		Identifier:    identifier,
		Conformances:  conformances,
		Members:       members,
		DocString:     docString,
		Range:         declRange,
	}
}

func (*InterfaceDeclaration) ElementType() ElementType {
	return ElementTypeInterfaceDeclaration
}

func (d *InterfaceDeclaration) Walk(walkChild func(Element)) {
	walkDeclarations(walkChild, d.Members.declarations)
}

func (*InterfaceDeclaration) isDeclaration() {}

// NOTE: statement, so it can be represented in the AST,
// but will be rejected in semantic analysis
//
func (*InterfaceDeclaration) isStatement() {}

func (d *InterfaceDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *InterfaceDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *InterfaceDeclaration) DeclarationKind() common.DeclarationKind {
	return d.CompositeKind.DeclarationKind(true)
}

func (d *InterfaceDeclaration) DeclarationMembers() *Members {
	return d.Members
}

func (d *InterfaceDeclaration) DeclarationDocString() string {
	return d.DocString
}

func (d *InterfaceDeclaration) MarshalJSON() ([]byte, error) {
	type Alias InterfaceDeclaration
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "InterfaceDeclaration",
		Alias: (*Alias)(d),
	})
}

func (d *InterfaceDeclaration) Doc() prettier.Doc {
	return CompositeDocument(
		d.Access,
		d.CompositeKind,
		true,
		d.Identifier.Identifier,
		d.Conformances,
		d.Members,
	)
}

func (d *InterfaceDeclaration) String() string {
	return Prettier(d)
}

func (d *InterfaceDeclaration) InterfaceConformances() []*NominalType {
	return d.Conformances
}
