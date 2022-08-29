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
	"github.com/turbolent/prettier"
)

// ExtensionDeclaration

type ExtensionDeclaration struct {
	Access       Access
	Identifier   Identifier
	BaseType     Identifier
	Conformances []*NominalType
	Members      *Members
	DocString    string
	Range
}

var _ Element = &ExtensionDeclaration{}
var _ Declaration = &ExtensionDeclaration{}
var _ Statement = &ExtensionDeclaration{}

func NewExtensionDeclaration(
	memoryGauge common.MemoryGauge,
	access Access,
	identifier Identifier,
	baseType Identifier,
	conformances []*NominalType,
	members *Members,
	docString string,
	declarationRange Range,
) *ExtensionDeclaration {
	common.UseMemory(memoryGauge, common.ExtensionDeclarationMemoryUsage)

	return &ExtensionDeclaration{
		Access:       access,
		Identifier:   identifier,
		BaseType:     baseType,
		Conformances: conformances,
		Members:      members,
		DocString:    docString,
		Range:        declarationRange,
	}
}

func (*ExtensionDeclaration) ElementType() ElementType {
	return ElementTypeExtensionDeclaration
}

func (d *ExtensionDeclaration) Walk(walkChild func(Element)) {
	walkDeclarations(walkChild, d.Members.declarations)
}

func (*ExtensionDeclaration) isDeclaration() {}

// NOTE: statement, so it can be represented in the AST,
// but will be rejected in semantic analysis
//
func (*ExtensionDeclaration) isStatement() {}

func (d *ExtensionDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *ExtensionDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindExtension
}

func (d *ExtensionDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *ExtensionDeclaration) DeclarationMembers() *Members {
	return d.Members
}

func (d *ExtensionDeclaration) DeclarationDocString() string {
	return d.DocString
}

const extensionStatementDoc = prettier.Text("extension")
const extensionStatementForDoc = prettier.Text("for")
const extensionConformancesSeparatorDoc = prettier.Text(":")

var extensionConformanceSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func (e *ExtensionDeclaration) Doc() prettier.Doc {
	var doc prettier.Concat

	if e.Access != AccessNotSpecified {
		doc = append(
			doc,
			prettier.Text(e.Access.Keyword()),
			prettier.Space,
		)
	}

	doc = append(
		doc,
		extensionStatementDoc,
		prettier.Space,
		prettier.Text(e.Identifier.Identifier),
		prettier.Space,
		extensionStatementForDoc,
		prettier.Space,
	)
	if len(e.Conformances) > 0 {

		conformancesDoc := prettier.Concat{
			prettier.Line{},
		}

		for i, conformance := range e.Conformances {
			if i > 0 {
				conformancesDoc = append(
					conformancesDoc,
					extensionConformanceSeparatorDoc,
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
					e.Members.Doc(),
				},
			},
		)

		doc = append(
			doc,
			extensionConformancesSeparatorDoc,
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
			e.Members.Doc(),
		)
	}

	return doc
}

func (d *ExtensionDeclaration) MarshalJSON() ([]byte, error) {
	type Alias ExtensionDeclaration
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "CompositeDeclaration",
		Alias: (*Alias)(d),
	})
}

func (d *ExtensionDeclaration) String() string {
	return Prettier(d)
}

// ExtendExpression
type ExtendExpression struct {
	Base       Expression
	Extensions []Expression
	StartPos   Position `json:"-"`
}

var _ Element = &ExtendExpression{}
var _ Expression = &ExtendExpression{}

func (*ExtendExpression) ElementType() ElementType {
	return ElementTypeExtendExpression
}

func (*ExtendExpression) isExpression() {}

func (*ExtendExpression) isIfStatementTest() {}

func (e *ExtendExpression) Walk(walkChild func(Element)) {
	walkChild(e.Base)
	for _, extension := range e.Extensions {
		walkChild(extension)
	}
}

func NewExtendExpression(
	gauge common.MemoryGauge,
	base Expression,
	extensions []Expression,
	startPos Position,
) *ExtendExpression {
	common.UseMemory(gauge, common.ExtendExpressionMemoryUsage)

	return &ExtendExpression{
		Base:       base,
		Extensions: extensions,
		StartPos:   startPos,
	}
}

func (e *ExtendExpression) String() string {
	return Prettier(e)
}

const extendExpressionDoc = prettier.Text("extend")
const extendExpressionWithDoc = prettier.Text("with")
const extendExpressionAndDoc = prettier.Text("and")

func (e *ExtendExpression) Doc() prettier.Doc {
	var doc prettier.Concat

	doc = append(
		doc,
		extendExpressionDoc,
		prettier.Space,
		e.Base.Doc(),
		prettier.Space,
		extendExpressionWithDoc,
		prettier.Space,
	)

	for i, extension := range e.Extensions {
		doc = append(doc, extension.Doc())
		if i < len(e.Extensions)-1 {
			doc = append(
				doc,
				prettier.Space,
				extendExpressionAndDoc,
				prettier.Space,
			)
		}
	}

	return doc
}

func (e *ExtendExpression) StartPosition() Position {
	return e.StartPos
}

func (e *ExtendExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	last := len(e.Extensions)
	return e.Extensions[last-1].EndPosition(memoryGauge)
}

func (*ExtendExpression) precedence() precedence {
	return precendenceExtend
}

func (e *ExtendExpression) MarshalJSON() ([]byte, error) {
	type Alias ExtendExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ExtendExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// RemoveExpression
type RemoveStatement struct {
	ValueTarget     Expression
	ExtensionTarget Expression
	Transfer        *Transfer
	Extension       Identifier
	Value           Expression
	IsDeclaration   bool
	IsConstant      bool
	StartPos        Position `json:"-"`
}

var _ Element = &RemoveStatement{}
var _ Statement = &RemoveStatement{}

func NewRemoveStatement(
	gauge common.MemoryGauge,
	valueTarget Expression,
	extensionTarget Expression,
	transfer *Transfer,
	extension Identifier,
	value Expression,
	isDeclaration bool,
	isConstant bool,
	startPos Position,
) *RemoveStatement {
	common.UseMemory(gauge, common.RemoveStatementMemoryUsage)

	return &RemoveStatement{
		ValueTarget:     valueTarget,
		ExtensionTarget: extensionTarget,
		Transfer:        transfer,
		Extension:       extension,
		Value:           value,
		IsDeclaration:   isDeclaration,
		IsConstant:      isConstant,
		StartPos:        startPos,
	}
}

func (*RemoveStatement) ElementType() ElementType {
	return ElementTypeRemoveStatement
}

func (*RemoveStatement) isStatement() {}

func (s *RemoveStatement) Walk(walkChild func(Element)) {
	walkChild(s.ValueTarget)
	walkChild(s.ExtensionTarget)
	walkChild(s.Value)
}

func (s *RemoveStatement) StartPosition() Position {
	return s.StartPos
}

func (s *RemoveStatement) EndPosition(memoryGauge common.MemoryGauge) Position {
	return s.Value.EndPosition(memoryGauge)
}

const removeStatementRemoveKeywordDoc = prettier.Text("remove")
const removeStatementFromKeywordDoc = prettier.Text("from")
const removeStatementSeparatorDoc = prettier.Text(",")

func (s *RemoveStatement) Doc() prettier.Doc {
	var doc prettier.Concat

	if s.IsDeclaration {
		if s.IsConstant {
			doc = append(doc, letKeywordDoc, prettier.Space)
		} else {
			doc = append(doc, varKeywordDoc, prettier.Space)
		}
	}

	return append(
		doc,
		s.ValueTarget.Doc(),
		removeStatementSeparatorDoc,
		prettier.Space,
		s.ExtensionTarget.Doc(),
		prettier.Space,
		s.Transfer.Doc(),
		prettier.Space,
		removeStatementRemoveKeywordDoc,
		prettier.Space,
		s.ExtensionTarget.Doc(),
		prettier.Space,
		removeStatementFromKeywordDoc,
		prettier.Space,
		s.Value.Doc(),
	)
}

func (s *RemoveStatement) String() string {
	return Prettier(s)
}

func (s *RemoveStatement) MarshalJSON() ([]byte, error) {
	type Alias RemoveStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "RemoveStatement",
		Range: NewUnmeteredRangeFromPositioned(s),
		Alias: (*Alias)(s),
	})
}
