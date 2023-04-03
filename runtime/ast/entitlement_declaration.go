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
	Range
}

var _ Element = &EntitlementDeclaration{}
var _ Declaration = &EntitlementDeclaration{}
var _ Statement = &EntitlementDeclaration{}

func NewEntitlementDeclaration(
	gauge common.MemoryGauge,
	access Access,
	identifier Identifier,
	docString string,
	declRange Range,
) *EntitlementDeclaration {
	common.UseMemory(gauge, common.EntitlementDeclarationMemoryUsage)

	return &EntitlementDeclaration{
		Access:     access,
		Identifier: identifier,
		DocString:  docString,
		Range:      declRange,
	}
}

func (*EntitlementDeclaration) ElementType() ElementType {
	return ElementTypeEntitlementDeclaration
}

func (*EntitlementDeclaration) Walk(_ func(Element)) {}

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
	return nil
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
	)

	return doc
}

func (d *EntitlementDeclaration) String() string {
	return Prettier(d)
}

type EntitlementMapElement struct {
	Input  *NominalType
	Output *NominalType
}

func NewEntitlementMapElement(
	gauge common.MemoryGauge,
	input *NominalType,
	output *NominalType,
) *EntitlementMapElement {
	common.UseMemory(gauge, common.EntitlementMappingElementMemoryUsage)

	return &EntitlementMapElement{
		Input:  input,
		Output: output,
	}
}

var arrowKeywordSpaceDoc = prettier.Text(" -> ")

func (d EntitlementMapElement) Doc() prettier.Doc {
	var doc prettier.Concat

	return append(
		doc,
		d.Input.Doc(),
		arrowKeywordSpaceDoc,
		d.Output.Doc(),
	)
}

// EntitlementMappingDeclaration
type EntitlementMappingDeclaration struct {
	Access       Access
	DocString    string
	Identifier   Identifier
	Associations []*EntitlementMapElement
	Range
}

var _ Element = &EntitlementMappingDeclaration{}
var _ Declaration = &EntitlementMappingDeclaration{}
var _ Statement = &EntitlementMappingDeclaration{}

func NewEntitlementMappingDeclaration(
	gauge common.MemoryGauge,
	access Access,
	identifier Identifier,
	associations []*EntitlementMapElement,
	docString string,
	declRange Range,
) *EntitlementMappingDeclaration {
	common.UseMemory(gauge, common.EntitlementDeclarationMemoryUsage)

	return &EntitlementMappingDeclaration{
		Access:       access,
		Identifier:   identifier,
		Associations: associations,
		DocString:    docString,
		Range:        declRange,
	}
}

func (*EntitlementMappingDeclaration) ElementType() ElementType {
	return ElementTypeEntitlementDeclaration
}

func (*EntitlementMappingDeclaration) Walk(_ func(Element)) {}

func (*EntitlementMappingDeclaration) isDeclaration() {}

// NOTE: statement, so it can be represented in the AST,
// but will be rejected in semantic analysis
func (*EntitlementMappingDeclaration) isStatement() {}

func (d *EntitlementMappingDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *EntitlementMappingDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *EntitlementMappingDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindEntitlementMapping
}

func (d *EntitlementMappingDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *EntitlementMappingDeclaration) DeclarationDocString() string {
	return d.DocString
}

func (d *EntitlementMappingDeclaration) MarshalJSON() ([]byte, error) {
	type Alias EntitlementMappingDeclaration
	return json.Marshal(&struct {
		*Alias
		Type string
	}{
		Type:  "EntitlementMappingDeclaration",
		Alias: (*Alias)(d),
	})
}

var mappingKeywordSpaceDoc = prettier.Text("mapping ")
var mappingStartDoc prettier.Doc = prettier.Text("{")
var mappingEndDoc prettier.Doc = prettier.Text("}")

func (d *EntitlementMappingDeclaration) Doc() prettier.Doc {
	var doc prettier.Concat

	if d.Access != AccessNotSpecified {
		doc = append(
			doc,
			prettier.Text(d.Access.Keyword()),
			prettier.Space,
		)
	}

	var mappingAssociationsDoc prettier.Concat

	for _, decl := range d.Associations {
		mappingAssociationsDoc = append(
			mappingAssociationsDoc,
			prettier.Concat{
				prettier.HardLine{},
				decl.Doc(),
			},
		)
	}

	doc = append(
		doc,
		entitlementKeywordSpaceDoc,
		mappingKeywordSpaceDoc,
		prettier.Text(d.Identifier.Identifier),
		prettier.Space,
		mappingStartDoc,
		prettier.Indent{
			Doc: prettier.Join(
				prettier.HardLine{},
				mappingAssociationsDoc...,
			),
		},
		prettier.HardLine{},
		mappingEndDoc,
	)

	return doc
}

func (d *EntitlementMappingDeclaration) String() string {
	return Prettier(d)
}
