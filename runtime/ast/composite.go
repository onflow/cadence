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

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

// CompositeDeclaration

// NOTE: For events, only an empty initializer is declared

type CompositeDeclaration struct {
	Members      *Members
	DocString    string
	Conformances []*NominalType
	Identifier   Identifier
	Range
	Access        Access
	CompositeKind common.CompositeKind
}

var _ Element = &CompositeDeclaration{}
var _ Declaration = &CompositeDeclaration{}
var _ Statement = &CompositeDeclaration{}

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

func (*CompositeDeclaration) ElementType() ElementType {
	return ElementTypeCompositeDeclaration
}

func (d *CompositeDeclaration) Walk(walkChild func(Element)) {
	walkDeclarations(walkChild, d.Members.declarations)
}

func (*CompositeDeclaration) isDeclaration() {}

// NOTE: statement, so it can be represented in the AST,
// but will be rejected in semantic analysis
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
		*Alias
		Type string
	}{
		Type:  "CompositeDeclaration",
		Alias: (*Alias)(d),
	})
}

func (d *CompositeDeclaration) Doc() prettier.Doc {

	if d.CompositeKind == common.CompositeKindEvent {
		return d.EventDoc()
	}

	return CompositeDocument(
		d.Access,
		d.CompositeKind,
		false,
		d.Identifier.Identifier,
		d.Conformances,
		d.Members,
	)
}

func (d *CompositeDeclaration) EventDoc() prettier.Doc {
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
		prettier.Text(d.CompositeKind.Keyword()),
		prettier.Space,
		prettier.Text(d.Identifier.Identifier),
	)

	initializers := d.Members.Initializers()
	if len(initializers) != 1 {
		return nil
	}

	initializer := initializers[0]
	paramsDoc := initializer.FunctionDeclaration.ParameterList.Doc()

	return append(doc, paramsDoc)
}

func (d *CompositeDeclaration) String() string {
	return Prettier(d)
}

var interfaceKeywordSpaceDoc = prettier.Text("interface ")
var compositeConformancesSeparatorDoc = prettier.Text(":")
var compositeConformanceSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func CompositeDocument(
	access Access,
	kind common.CompositeKind,
	isInterface bool,
	identifier string,
	conformances []*NominalType,
	members *Members,
) prettier.Doc {

	var doc prettier.Concat

	if access != AccessNotSpecified {
		doc = append(
			doc,
			prettier.Text(access.Keyword()),
			prettier.Space,
		)
	}

	doc = append(
		doc,
		prettier.Text(kind.Keyword()),
		prettier.Space,
	)

	if isInterface {
		doc = append(
			doc,
			interfaceKeywordSpaceDoc,
		)
	}

	doc = append(
		doc,
		prettier.Text(identifier),
	)

	if len(conformances) > 0 {

		conformancesDoc := prettier.Concat{
			prettier.Line{},
		}

		for i, conformance := range conformances {
			if i > 0 {
				conformancesDoc = append(
					conformancesDoc,
					compositeConformanceSeparatorDoc,
				)
			}

			conformancesDoc = append(
				conformancesDoc,
				conformance.Doc(),
			)
		}

		conformancesDoc = append(
			conformancesDoc,
			prettier.Dedent{
				Doc: prettier.Concat{
					prettier.Line{},
					members.Doc(),
				},
			},
		)

		doc = append(
			doc,
			compositeConformancesSeparatorDoc,
			prettier.Group{
				Doc: prettier.Indent{
					Doc: conformancesDoc,
				},
			},
		)

	} else {
		doc = append(
			doc,
			prettier.Space,
			members.Doc(),
		)
	}

	return doc
}

// FieldDeclarationFlags

type FieldDeclarationFlags uint8

const (
	FieldDeclarationFlagsIsStatic FieldDeclarationFlags = 1 << iota
	FieldDeclarationFlagsIsNative
)

// FieldDeclaration

type FieldDeclaration struct {
	TypeAnnotation *TypeAnnotation
	DocString      string
	Identifier     Identifier
	Range
	Access       Access
	VariableKind VariableKind
	Flags        FieldDeclarationFlags
}

var _ Element = &FieldDeclaration{}
var _ Declaration = &FieldDeclaration{}

func NewFieldDeclaration(
	memoryGauge common.MemoryGauge,
	access Access,
	isStatic bool,
	isNative bool,
	variableKind VariableKind,
	identifier Identifier,
	typeAnnotation *TypeAnnotation,
	docString string,
	declRange Range,
) *FieldDeclaration {
	common.UseMemory(memoryGauge, common.FieldDeclarationMemoryUsage)

	var flags FieldDeclarationFlags
	if isStatic {
		flags |= FieldDeclarationFlagsIsStatic
	}
	if isNative {
		flags |= FieldDeclarationFlagsIsNative
	}

	return &FieldDeclaration{
		Access:         access,
		Flags:          flags,
		VariableKind:   variableKind,
		Identifier:     identifier,
		TypeAnnotation: typeAnnotation,
		DocString:      docString,
		Range:          declRange,
	}
}

func (*FieldDeclaration) ElementType() ElementType {
	return ElementTypeFieldDeclaration
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
		*Alias
		Type     string
		Flags    FieldDeclarationFlags `json:",omitempty"`
		IsStatic bool
		IsNative bool
	}{
		Type:     "FieldDeclaration",
		Alias:    (*Alias)(d),
		IsStatic: d.IsStatic(),
		IsNative: d.IsNative(),
		Flags:    0,
	})
}

func VariableKindDoc(kind VariableKind) prettier.Doc {
	switch kind {
	case VariableKindNotSpecified:
		return nil
	case VariableKindConstant:
		return letKeywordDoc
	case VariableKindVariable:
		return varKeywordDoc
	default:
		panic(errors.NewUnreachableError())
	}
}

var staticKeywordDoc prettier.Doc = prettier.Text("static")
var nativeKeywordDoc prettier.Doc = prettier.Text("native")

func (d *FieldDeclaration) Doc() prettier.Doc {
	identifierTypeDoc := prettier.Concat{
		prettier.Text(d.Identifier.Identifier),
	}

	if d.TypeAnnotation != nil {
		identifierTypeDoc = append(
			identifierTypeDoc,
			typeSeparatorSpaceDoc,
			d.TypeAnnotation.Doc(),
		)
	}

	var docs []prettier.Doc

	if d.Access != AccessNotSpecified {
		docs = append(
			docs,
			prettier.Text(d.Access.Keyword()),
		)
	}

	if d.IsStatic() {
		docs = append(
			docs,
			staticKeywordDoc,
		)
	}

	if d.IsNative() {
		docs = append(
			docs,
			nativeKeywordDoc,
		)
	}

	keywordDoc := VariableKindDoc(d.VariableKind)

	if keywordDoc != nil {
		docs = append(
			docs,
			keywordDoc,
		)
	}

	var doc prettier.Doc

	if len(docs) > 0 {
		docs = append(
			docs,
			prettier.Group{
				Doc: identifierTypeDoc,
			},
		)

		doc = prettier.Join(prettier.Space, docs...)
	} else {
		doc = identifierTypeDoc
	}

	return prettier.Group{
		Doc: doc,
	}
}

func (d *FieldDeclaration) String() string {
	return Prettier(d)
}

func (d *FieldDeclaration) IsStatic() bool {
	return d.Flags&FieldDeclarationFlagsIsStatic != 0
}

func (d *FieldDeclaration) IsNative() bool {
	return d.Flags&FieldDeclarationFlagsIsNative != 0
}

// EnumCaseDeclaration

type EnumCaseDeclaration struct {
	DocString  string
	Identifier Identifier
	StartPos   Position `json:"-"`
	Access     Access
}

var _ Element = &EnumCaseDeclaration{}
var _ Declaration = &EnumCaseDeclaration{}

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

func (*EnumCaseDeclaration) ElementType() ElementType {
	return ElementTypeEnumCaseDeclaration
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
		*Alias
		Type string
		Range
	}{
		Type:  "EnumCaseDeclaration",
		Range: NewUnmeteredRangeFromPositioned(d),
		Alias: (*Alias)(d),
	})
}

const enumCaseKeywordSpaceDoc = prettier.Text("case ")

func (d *EnumCaseDeclaration) Doc() prettier.Doc {
	var doc prettier.Concat

	if d.Access != AccessNotSpecified {
		doc = append(
			doc,
			prettier.Text(d.Access.Keyword()),
			prettier.Space,
		)
	}

	return append(
		doc,
		enumCaseKeywordSpaceDoc,
		prettier.Text(d.Identifier.Identifier),
	)
}

func (d *EnumCaseDeclaration) String() string {
	return Prettier(d)
}
