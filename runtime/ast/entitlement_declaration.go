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

package ast

import (
	"encoding/json"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
)

// EntitlementDeclaration

type EntitlementDeclaration struct {
	Access     Access
	Identifier Identifier
	Range
	Comments
}

var _ Element = &EntitlementDeclaration{}
var _ Declaration = &EntitlementDeclaration{}
var _ Statement = &EntitlementDeclaration{}

func NewEntitlementDeclaration(
	gauge common.MemoryGauge,
	access Access,
	identifier Identifier,
	declRange Range,
	comments Comments,
) *EntitlementDeclaration {
	common.UseMemory(gauge, common.EntitlementDeclarationMemoryUsage)

	return &EntitlementDeclaration{
		Access:     access,
		Identifier: identifier,
		Range:      declRange,
		Comments:   comments,
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
	return d.Comments.LeadingDocString()
}

func (d *EntitlementDeclaration) MarshalJSON() ([]byte, error) {
	type Alias EntitlementDeclaration
	return json.Marshal(&struct {
		*Alias
		Type      string
		DocString string
	}{
		Type:      "EntitlementDeclaration",
		Alias:     (*Alias)(d),
		DocString: d.DeclarationDocString(),
	})
}

var entitlementKeywordSpaceDoc = prettier.Text("entitlement ")

func (d *EntitlementDeclaration) Doc() prettier.Doc {
	var doc prettier.Concat

	if d.Access != AccessNotSpecified {
		doc = append(
			doc,
			prettier.Text(d.Access.Keyword()),
			prettier.HardLine{},
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

type EntitlementMapElement interface {
	isEntitlementMapElement()
	Doc() prettier.Doc
}

type EntitlementMapRelation struct {
	Input  *NominalType
	Output *NominalType
}

var _ EntitlementMapElement = &EntitlementMapRelation{}

func NewEntitlementMapRelation(
	gauge common.MemoryGauge,
	input *NominalType,
	output *NominalType,
) *EntitlementMapRelation {
	common.UseMemory(gauge, common.EntitlementMappingElementMemoryUsage)

	return &EntitlementMapRelation{
		Input:  input,
		Output: output,
	}
}

var arrowKeywordSpaceDoc = prettier.Text(" -> ")

func (*EntitlementMapRelation) isEntitlementMapElement() {}

func (d *EntitlementMapRelation) Doc() prettier.Doc {
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
	Access     Access
	Identifier Identifier
	Elements   []EntitlementMapElement
	Range
	Comments
}

var _ Element = &EntitlementMappingDeclaration{}
var _ Declaration = &EntitlementMappingDeclaration{}
var _ Statement = &EntitlementMappingDeclaration{}

func NewEntitlementMappingDeclaration(
	gauge common.MemoryGauge,
	access Access,
	identifier Identifier,
	elements []EntitlementMapElement,
	declRange Range,
	comments Comments,
) *EntitlementMappingDeclaration {
	common.UseMemory(gauge, common.EntitlementMappingDeclarationMemoryUsage)

	return &EntitlementMappingDeclaration{
		Access:     access,
		Identifier: identifier,
		Elements:   elements,
		Range:      declRange,
		Comments:   comments,
	}
}

func (*EntitlementMappingDeclaration) ElementType() ElementType {
	return ElementTypeEntitlementMappingDeclaration
}

func (*EntitlementMappingDeclaration) Walk(_ func(Element)) {}

func (*EntitlementMappingDeclaration) isDeclaration() {}

func (d *EntitlementMappingDeclaration) Inclusions() (inclusions []*NominalType) {
	for _, element := range d.Elements {
		if inclusion, isNominalType := element.(*NominalType); isNominalType {
			inclusions = append(inclusions, inclusion)
		}
	}
	return
}

func (d *EntitlementMappingDeclaration) Relations() (relations []*EntitlementMapRelation) {
	for _, element := range d.Elements {
		if relation, isRelation := element.(*EntitlementMapRelation); isRelation {
			relations = append(relations, relation)
		}
	}
	return
}

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
	return d.Comments.LeadingDocString()
}

func (d *EntitlementMappingDeclaration) MarshalJSON() ([]byte, error) {
	type Alias EntitlementMappingDeclaration
	return json.Marshal(&struct {
		*Alias
		Type      string
		DocString string
	}{
		Type:      "EntitlementMappingDeclaration",
		Alias:     (*Alias)(d),
		DocString: d.DeclarationDocString(),
	})
}

var mappingKeywordSpaceDoc = prettier.Text("mapping ")
var includeKeywordSpaceDoc = prettier.Text("include ")
var mappingStartDoc prettier.Doc = prettier.Text("{")
var mappingEndDoc prettier.Doc = prettier.Text("}")

func (d *EntitlementMappingDeclaration) Doc() prettier.Doc {
	var doc prettier.Concat

	if d.Access != AccessNotSpecified {
		doc = append(
			doc,
			prettier.Text(d.Access.Keyword()),
			prettier.HardLine{},
		)
	}

	var mappingElementsDoc prettier.Concat

	for _, element := range d.Elements {
		var elementDoc prettier.Concat

		if _, isNominalType := element.(*NominalType); isNominalType {
			elementDoc = append(elementDoc, includeKeywordSpaceDoc)
		}

		elementDoc = append(elementDoc, element.Doc())

		mappingElementsDoc = append(
			mappingElementsDoc,
			elementDoc,
		)
	}

	doc = append(
		doc,
		entitlementKeywordSpaceDoc,
		mappingKeywordSpaceDoc,
		prettier.Text(d.Identifier.Identifier),
		prettier.Space,
		mappingStartDoc,
		prettier.HardLine{},
		prettier.Indent{
			Doc: prettier.Join(
				prettier.HardLine{},
				mappingElementsDoc...,
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
