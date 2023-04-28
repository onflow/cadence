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

// AttachmentDeclaration

type AttachmentDeclaration struct {
	Access               Access
	Identifier           Identifier
	BaseType             *NominalType
	Conformances         []*NominalType
	RequiredEntitlements []*NominalType
	Members              *Members
	DocString            string
	Range
}

var _ Element = &AttachmentDeclaration{}
var _ Declaration = &AttachmentDeclaration{}
var _ Statement = &AttachmentDeclaration{}
var _ CompositeLikeDeclaration = &AttachmentDeclaration{}

func NewAttachmentDeclaration(
	memoryGauge common.MemoryGauge,
	access Access,
	identifier Identifier,
	baseType *NominalType,
	conformances []*NominalType,
	requiredEntitlements []*NominalType,
	members *Members,
	docString string,
	declarationRange Range,
) *AttachmentDeclaration {
	common.UseMemory(memoryGauge, common.AttachmentDeclarationMemoryUsage)

	return &AttachmentDeclaration{
		Access:               access,
		Identifier:           identifier,
		BaseType:             baseType,
		Conformances:         conformances,
		RequiredEntitlements: requiredEntitlements,
		Members:              members,
		DocString:            docString,
		Range:                declarationRange,
	}
}

func (*AttachmentDeclaration) ElementType() ElementType {
	return ElementTypeAttachmentDeclaration
}

func (d *AttachmentDeclaration) Walk(walkChild func(Element)) {
	walkDeclarations(walkChild, d.Members.declarations)
}

func (*AttachmentDeclaration) isDeclaration() {}

// NOTE: statement, so it can be represented in the AST,
// but will be rejected in semantic analysis
func (*AttachmentDeclaration) isStatement() {}

func (d *AttachmentDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *AttachmentDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindAttachment
}

func (d *AttachmentDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *AttachmentDeclaration) DeclarationMembers() *Members {
	return d.Members
}

func (d *AttachmentDeclaration) DeclarationDocString() string {
	return d.DocString
}

func (*AttachmentDeclaration) Kind() common.CompositeKind {
	return common.CompositeKindAttachment
}

func (d *AttachmentDeclaration) ConformanceList() []*NominalType {
	return d.Conformances
}

func (d *AttachmentDeclaration) RequiredEntitlementsToAttach() []*NominalType {
	return d.RequiredEntitlements
}

const attachmentStatementDoc = prettier.Text("attachment")
const attachmentStatementForDoc = prettier.Text("for")
const attachmentConformancesSeparatorDoc = prettier.Text(":")
const attachmentEntitlementDoc = prettier.Text("entitlement")
const attachmentRequireDoc = prettier.Text("require")

var attachmentConformanceSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func (e *AttachmentDeclaration) Doc() prettier.Doc {
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
		attachmentStatementDoc,
		prettier.Space,
		prettier.Text(e.Identifier.Identifier),
		prettier.Space,
		attachmentStatementForDoc,
		prettier.Space,
		e.BaseType.Doc(),
	)

	var membersDoc prettier.Concat

	if e.RequiredEntitlements != nil && len(e.RequiredEntitlements) > 0 {
		membersDoc = append(membersDoc, membersStartDoc)
		for _, entitlement := range e.RequiredEntitlements {
			var entitlementRequiredDoc = prettier.Indent{
				Doc: prettier.Concat{
					attachmentRequireDoc,
					prettier.Space,
					attachmentEntitlementDoc,
					prettier.Space,
					entitlement.Doc(),
				},
			}
			membersDoc = append(
				membersDoc,
				prettier.HardLine{},
				entitlementRequiredDoc,
			)
		}
		if len(e.Members.declarations) > 0 {
			membersDoc = append(membersDoc, prettier.HardLine{}, e.Members.docWithNoBraces())
		}
		membersDoc = append(membersDoc, prettier.HardLine{}, membersEndDoc)
	} else {
		membersDoc = append(membersDoc, prettier.Line{}, e.Members.Doc())
	}

	if len(e.Conformances) > 0 {
		conformancesDoc := prettier.Concat{
			prettier.Line{},
		}

		for i, conformance := range e.Conformances {
			if i > 0 {
				conformancesDoc = append(
					conformancesDoc,
					attachmentConformanceSeparatorDoc,
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
					membersDoc,
				},
			},
		)

		doc = append(
			doc,
			attachmentConformancesSeparatorDoc,
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
			membersDoc,
		)
	}

	return doc
}

func (d *AttachmentDeclaration) MarshalJSON() ([]byte, error) {
	type Alias AttachmentDeclaration
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "AttachmentDeclaration",
		Alias: (*Alias)(d),
	})
}

func (d *AttachmentDeclaration) String() string {
	return Prettier(d)
}

// AttachExpression
type AttachExpression struct {
	Base         Expression
	Attachment   *InvocationExpression
	Entitlements []*NominalType
	StartPos     Position `json:"-"`
}

var _ Element = &AttachExpression{}
var _ Expression = &AttachExpression{}

func (*AttachExpression) ElementType() ElementType {
	return ElementTypeAttachExpression
}

func (*AttachExpression) isExpression() {}

func (*AttachExpression) isIfStatementTest() {}

func (e *AttachExpression) Walk(walkChild func(Element)) {
	walkChild(e.Base)
	walkChild(e.Attachment)
}

func NewAttachExpression(
	gauge common.MemoryGauge,
	base Expression,
	attachment *InvocationExpression,
	entitlements []*NominalType,
	startPos Position,
) *AttachExpression {
	common.UseMemory(gauge, common.AttachExpressionMemoryUsage)

	return &AttachExpression{
		Base:         base,
		Attachment:   attachment,
		Entitlements: entitlements,
		StartPos:     startPos,
	}
}

func (e *AttachExpression) String() string {
	return Prettier(e)
}

const attachExpressionDoc = prettier.Text("attach")
const attachExpressionToDoc = prettier.Text("to")
const attachExpressionWithDoc = prettier.Text("with")
const attachExpressionCommaDoc = prettier.Text(",")

func (e *AttachExpression) Doc() prettier.Doc {
	var doc prettier.Concat

	doc = append(
		doc,
		attachExpressionDoc,
		prettier.Space,
		e.Attachment.Doc(),
		prettier.Space,
		attachExpressionToDoc,
		prettier.Space,
		e.Base.Doc(),
	)
	if e.Entitlements != nil && len(e.Entitlements) > 0 {
		entitlementsLen := len(e.Entitlements)
		doc = append(doc, prettier.Space, attachExpressionWithDoc, prettier.Space, openParenthesisDoc)
		for i, entitlement := range e.Entitlements {
			doc = append(doc, entitlement.Doc())
			if i < entitlementsLen-1 {
				doc = append(doc, attachExpressionCommaDoc, prettier.Space)
			}
		}
		doc = append(doc, closeParenthesisDoc)
	}
	return doc
}

func (e *AttachExpression) StartPosition() Position {
	return e.StartPos
}

func (e *AttachExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	if e.Entitlements != nil && len(e.Entitlements) > 0 {
		return e.Entitlements[len(e.Entitlements)-1].EndPosition(memoryGauge)
	}
	return e.Base.EndPosition(memoryGauge)
}

func (*AttachExpression) precedence() precedence {
	return precedenceLiteral
}

func (e *AttachExpression) MarshalJSON() ([]byte, error) {
	type Alias AttachExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "AttachExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// RemoveStatement
type RemoveStatement struct {
	Attachment *NominalType
	Value      Expression
	StartPos   Position `json:"-"`
}

var _ Element = &RemoveStatement{}
var _ Statement = &RemoveStatement{}

func NewRemoveStatement(
	gauge common.MemoryGauge,
	attachment *NominalType,
	value Expression,
	startPos Position,
) *RemoveStatement {
	common.UseMemory(gauge, common.RemoveStatementMemoryUsage)

	return &RemoveStatement{
		Attachment: attachment,
		Value:      value,
		StartPos:   startPos,
	}
}

func (*RemoveStatement) ElementType() ElementType {
	return ElementTypeRemoveStatement
}

func (*RemoveStatement) isStatement() {}

func (s *RemoveStatement) Walk(walkChild func(Element)) {
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

func (s *RemoveStatement) Doc() prettier.Doc {
	return prettier.Concat{
		removeStatementRemoveKeywordDoc,
		prettier.Space,
		s.Attachment.Doc(),
		prettier.Space,
		removeStatementFromKeywordDoc,
		prettier.Space,
		s.Value.Doc(),
	}
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
