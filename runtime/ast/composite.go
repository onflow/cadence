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

	"github.com/onflow/cadence/runtime/common"
)

// CompositeDeclaration

// NOTE: For events, only an empty initializer is declared

type CompositeDeclaration struct {
	Access        Access
	CompositeKind common.CompositeKind
	Identifier    Identifier
	Conformances  []*NominalType
	Members       *Members
	DocString     string
	Range
}

func NewCompositeDeclaration(
	memoryGauge common.MemoryGauge,
	access Access,
	compositeKind common.CompositeKind,
	identifier Identifier,
	conformances []*NominalType,
	members *Members,
	docString string,
	declarationRange Range,
) *CompositeDeclaration {
	common.UseMemory(memoryGauge, common.CompositeDeclarationMemoryUsage)

	return &CompositeDeclaration{
		Access:        access,
		CompositeKind: compositeKind,
		Identifier:    identifier,
		Conformances:  conformances,
		Members:       members,
		DocString:     docString,
		Range:         declarationRange,
	}
}

func (d *CompositeDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitCompositeDeclaration(d)
}

func (d *CompositeDeclaration) Walk(walkChild func(Element)) {
	walkDeclarations(walkChild, d.Members.declarations)
}

func (*CompositeDeclaration) isDeclaration() {}

// NOTE: statement, so it can be represented in the AST,
// but will be rejected in semantic analysis
//
func (*CompositeDeclaration) isStatement() {}

func (d *CompositeDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *CompositeDeclaration) DeclarationKind() common.DeclarationKind {
	return d.CompositeKind.DeclarationKind(false)
}

func (d *CompositeDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *CompositeDeclaration) DeclarationMembers() *Members {
	return d.Members
}

func (d *CompositeDeclaration) DeclarationDocString() string {
	return d.DocString
}

func (d *CompositeDeclaration) MarshalJSON() ([]byte, error) {
	type Alias CompositeDeclaration
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "CompositeDeclaration",
		Alias: (*Alias)(d),
	})
}

// FieldDeclaration

type FieldDeclaration struct {
	Access         Access
	VariableKind   VariableKind
	Identifier     Identifier
	TypeAnnotation *TypeAnnotation
	DocString      string
	Range
}

func NewFieldDeclaration(
	memoryGauge common.MemoryGauge,
	access Access,
	variableKind VariableKind,
	identifier Identifier,
	typeAnnotation *TypeAnnotation,
	docString string,
	declRange Range,
) *FieldDeclaration {
	common.UseMemory(memoryGauge, common.FieldDeclarationMemoryUsage)

	return &FieldDeclaration{
		Access:         access,
		VariableKind:   variableKind,
		Identifier:     identifier,
		TypeAnnotation: typeAnnotation,
		DocString:      docString,
		Range:          declRange,
	}
}

func (d *FieldDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitFieldDeclaration(d)
}

func (d *FieldDeclaration) Walk(_ func(Element)) {
	// NO-OP
	// TODO: walk type
}

func (*FieldDeclaration) isDeclaration() {}

func (d *FieldDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *FieldDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindField
}

func (d *FieldDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *FieldDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *FieldDeclaration) DeclarationDocString() string {
	return d.DocString
}

func (d *FieldDeclaration) MarshalJSON() ([]byte, error) {
	type Alias FieldDeclaration
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "FieldDeclaration",
		Alias: (*Alias)(d),
	})
}

// EnumCaseDeclaration

type EnumCaseDeclaration struct {
	Access     Access
	Identifier Identifier
	DocString  string
	StartPos   Position `json:"-"`
}

func NewEnumCaseDeclaration(
	memoryGauge common.MemoryGauge,
	access Access,
	identifier Identifier,
	docString string,
	startPos Position,
) *EnumCaseDeclaration {
	common.UseMemory(memoryGauge, common.EnumCaseDeclarationMemoryUsage)

	return &EnumCaseDeclaration{
		Access:     access,
		Identifier: identifier,
		DocString:  docString,
		StartPos:   startPos,
	}
}

func (d *EnumCaseDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitEnumCaseDeclaration(d)
}

func (*EnumCaseDeclaration) Walk(_ func(Element)) {
	// NO-OP
}

func (*EnumCaseDeclaration) isDeclaration() {}

func (d *EnumCaseDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *EnumCaseDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindEnumCase
}

func (d *EnumCaseDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *EnumCaseDeclaration) StartPosition() Position {
	return d.StartPos
}

func (d *EnumCaseDeclaration) EndPosition(memoryGauge common.MemoryGauge) Position {
	return d.Identifier.EndPosition(memoryGauge)
}

func (d *EnumCaseDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *EnumCaseDeclaration) DeclarationDocString() string {
	return d.DocString
}

func (d *EnumCaseDeclaration) MarshalJSON() ([]byte, error) {
	type Alias EnumCaseDeclaration
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "EnumCaseDeclaration",
		Range: NewUnmeteredRangeFromPositioned(d),
		Alias: (*Alias)(d),
	})
}
