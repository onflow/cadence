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
)

type VariableDeclaration struct {
	Value             Expression
	SecondValue       Expression
	TypeAnnotation    *TypeAnnotation
	Transfer          *Transfer
	SecondTransfer    *Transfer
	ParentIfStatement *IfStatement `json:"-"`
	DocString         string
	Identifier        Identifier
	StartPos          Position `json:"-"`
	Access            Access
	IsConstant        bool
}

var _ Element = &VariableDeclaration{}
var _ Statement = &VariableDeclaration{}
var _ Declaration = &VariableDeclaration{}

func NewVariableDeclaration(
	gauge common.MemoryGauge,
	access Access,
	isLet bool,
	identifier Identifier,
	typeAnnotation *TypeAnnotation,
	value Expression,
	transfer *Transfer,
	startPos Position,
	secondTransfer *Transfer,
	secondValue Expression,
	docString string,
) *VariableDeclaration {
	common.UseMemory(gauge, common.VariableDeclarationMemoryUsage)

	return &VariableDeclaration{
		Access:         access,
		IsConstant:     isLet,
		Identifier:     identifier,
		TypeAnnotation: typeAnnotation,
		Value:          value,
		Transfer:       transfer,
		StartPos:       startPos,
		SecondTransfer: secondTransfer,
		SecondValue:    secondValue,
		DocString:      docString,
	}
}

func NewEmptyVariableDeclaration(gauge common.MemoryGauge) *VariableDeclaration {
	common.UseMemory(gauge, common.VariableDeclarationMemoryUsage)
	return &VariableDeclaration{Access: AccessNotSpecified}
}

func (*VariableDeclaration) isDeclaration() {}

func (*VariableDeclaration) isStatement() {}

func (*VariableDeclaration) ElementType() ElementType {
	return ElementTypeVariableDeclaration
}

func (d *VariableDeclaration) StartPosition() Position {
	return d.StartPos
}

func (d *VariableDeclaration) EndPosition(memoryGauge common.MemoryGauge) Position {
	if d.SecondValue != nil {
		return d.SecondValue.EndPosition(memoryGauge)
	}
	return d.Value.EndPosition(memoryGauge)
}

func (*VariableDeclaration) isIfStatementTest() {}

func (d *VariableDeclaration) Walk(walkChild func(Element)) {
	// TODO: walk type
	walkChild(d.Value)
	if d.SecondValue != nil {
		walkChild(d.SecondValue)
	}
}

func (d *VariableDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *VariableDeclaration) DeclarationKind() common.DeclarationKind {
	if d.IsConstant {
		return common.DeclarationKindConstant
	}
	return common.DeclarationKindVariable
}

func (d *VariableDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *VariableDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *VariableDeclaration) DeclarationDocString() string {
	return d.DocString
}

var varKeywordDoc prettier.Doc = prettier.Text("var")
var letKeywordDoc prettier.Doc = prettier.Text("let")

func (d *VariableDeclaration) Doc() prettier.Doc {
	keywordDoc := varKeywordDoc
	if d.IsConstant {
		keywordDoc = letKeywordDoc
	}

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

	valueDoc := d.Value.Doc()

	var valuesDoc prettier.Doc

	if d.SecondValue == nil {
		// Put transfer before the break

		valuesDoc = prettier.Concat{
			prettier.Group{
				Doc: identifierTypeDoc,
			},
			prettier.Space,
			d.Transfer.Doc(),
			prettier.Group{
				Doc: prettier.Indent{
					Doc: prettier.Concat{
						prettier.Line{},
						valueDoc,
					},
				},
			},
		}
	} else {
		secondValueDoc := d.SecondValue.Doc()

		// Put transfers at start of value lines,
		// and break both values at once

		valuesDoc = prettier.Concat{
			prettier.Group{
				Doc: identifierTypeDoc,
			},
			prettier.Group{
				Doc: prettier.Indent{
					Doc: prettier.Concat{
						prettier.Line{},
						d.Transfer.Doc(),
						prettier.Space,
						valueDoc,
						prettier.Line{},
						d.SecondTransfer.Doc(),
						prettier.Space,
						secondValueDoc,
					},
				},
			},
		}
	}

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
		keywordDoc,
		prettier.Space,
		prettier.Group{
			Doc: valuesDoc,
		},
	)

	return prettier.Group{
		Doc: doc,
	}
}

func (d *VariableDeclaration) MarshalJSON() ([]byte, error) {
	type Alias VariableDeclaration
	return json.Marshal(&struct {
		*Alias
		Type string
		Range
	}{
		Type:  "VariableDeclaration",
		Range: NewUnmeteredRangeFromPositioned(d),
		Alias: (*Alias)(d),
	})
}

func (d *VariableDeclaration) String() string {
	return Prettier(d)
}
