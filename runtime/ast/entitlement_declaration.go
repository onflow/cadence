/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/common"
	"github.com/turbolent/prettier"
)

// EntitlementDeclaration

type EntitlementDeclaration struct {
	Access     Access
	DocString  string
	Identifier Identifier
	Members    *Members
	Range
}

var _ Element = &EntitlementDeclaration{}
var _ Declaration = &EntitlementDeclaration{}
var _ Statement = &EntitlementDeclaration{}

func NewEntitlementDeclaration(
	gauge common.MemoryGauge,
	access Access,
	identifier Identifier,
	members *Members,
	docString string,
	declRange Range,
) *EntitlementDeclaration {
	common.UseMemory(gauge, common.EntitlementDeclarationMemoryUsage)

	return &EntitlementDeclaration{
		Access:     access,
		Identifier: identifier,
		Members:    members,
		DocString:  docString,
		Range:      declRange,
	}
}

func (*EntitlementDeclaration) ElementType() ElementType {
	return ElementTypeEntitlementDeclaration
}

func (d *EntitlementDeclaration) Walk(walkChild func(Element)) {
	walkDeclarations(walkChild, d.Members.declarations)
}

func (*EntitlementDeclaration) isDeclaration() {}

// NOTE: statement, so it can be represented in the AST,
// but will be rejected in semantic analysis
func (*EntitlementDeclaration) isStatement() {}

func (d *EntitlementDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *EntitlementDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *EntitlementDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindEntitlement
}

func (d *EntitlementDeclaration) DeclarationMembers() *Members {
	return d.Members
}

func (d *EntitlementDeclaration) DeclarationDocString() string {
	return d.DocString
}

func (d *EntitlementDeclaration) MarshalJSON() ([]byte, error) {
	type Alias EntitlementDeclaration
	return json.Marshal(&struct {
		*Alias
		Type string
	}{
		Type:  "EntitlementDeclaration",
		Alias: (*Alias)(d),
	})
}

var entitlementKeywordSpaceDoc = prettier.Text("entitlement ")

func (d *EntitlementDeclaration) Doc() prettier.Doc {
	var doc prettier.Concat

	if d.Access != AccessNotSpecified {
		doc = append(
			doc,
			prettier.Text(d.Access.Keyword()),
			prettier.Space,
		)
	}

	doc = append(
		doc,
		entitlementKeywordSpaceDoc,
		prettier.Text(d.Identifier.Identifier),
		prettier.Space,
		d.Members.Doc(),
	)

	return doc
}

func (d *EntitlementDeclaration) String() string {
	return Prettier(d)
}
