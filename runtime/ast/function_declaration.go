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

type FunctionDeclaration struct {
	Access               Access
	Identifier           Identifier
	ParameterList        *ParameterList
	ReturnTypeAnnotation *TypeAnnotation
	FunctionBlock        *FunctionBlock
	DocString            string
	StartPos             Position `json:"-"`
}

var _ Element = &FunctionDeclaration{}
var _ Declaration = &FunctionDeclaration{}
var _ Statement = &FunctionDeclaration{}

func NewFunctionDeclaration(
	gauge common.MemoryGauge,
	access Access,
	identifier Identifier,
	parameterList *ParameterList,
	returnTypeAnnotation *TypeAnnotation,
	functionBlock *FunctionBlock,
	startPos Position,
	docString string,
) *FunctionDeclaration {
	common.UseMemory(gauge, common.FunctionDeclarationMemoryUsage)

	return &FunctionDeclaration{
		Access:               access,
		Identifier:           identifier,
		ParameterList:        parameterList,
		ReturnTypeAnnotation: returnTypeAnnotation,
		FunctionBlock:        functionBlock,
		StartPos:             startPos,
		DocString:            docString,
	}
}

func (*FunctionDeclaration) ElementType() ElementType {
	return ElementTypeFunctionDeclaration
}

func (d *FunctionDeclaration) StartPosition() Position {
	return d.StartPos
}

func (d *FunctionDeclaration) EndPosition(memoryGauge common.MemoryGauge) Position {
	if d.FunctionBlock != nil {
		return d.FunctionBlock.EndPosition(memoryGauge)
	}
	if d.ReturnTypeAnnotation != nil {
		return d.ReturnTypeAnnotation.EndPosition(memoryGauge)
	}
	return d.ParameterList.EndPosition(memoryGauge)
}

func (d *FunctionDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitFunctionDeclaration(d)
}

func (d *FunctionDeclaration) Walk(walkChild func(Element)) {
	// TODO: walk parameters
	// TODO: walk return type
	if d.FunctionBlock != nil {
		walkChild(d.FunctionBlock)
	}
}

func (*FunctionDeclaration) isDeclaration() {}
func (*FunctionDeclaration) isStatement()   {}

func (d *FunctionDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *FunctionDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindFunction
}

func (d *FunctionDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *FunctionDeclaration) ToExpression(memoryGauge common.MemoryGauge) *FunctionExpression {
	return NewFunctionExpression(
		memoryGauge,
		d.ParameterList,
		d.ReturnTypeAnnotation,
		d.FunctionBlock,
		d.StartPos,
	)
}

func (d *FunctionDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *FunctionDeclaration) DeclarationDocString() string {
	return d.DocString
}

func (d *FunctionDeclaration) MarshalJSON() ([]byte, error) {
	type Alias FunctionDeclaration
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "FunctionDeclaration",
		Range: NewUnmeteredRangeFromPositioned(d),
		Alias: (*Alias)(d),
	})
}

// SpecialFunctionDeclaration

type SpecialFunctionDeclaration struct {
	Kind                common.DeclarationKind
	FunctionDeclaration *FunctionDeclaration
}

var _ Element = &SpecialFunctionDeclaration{}
var _ Declaration = &SpecialFunctionDeclaration{}
var _ Statement = &SpecialFunctionDeclaration{}

func NewSpecialFunctionDeclaration(
	gauge common.MemoryGauge,
	kind common.DeclarationKind,
	funcDecl *FunctionDeclaration,
) *SpecialFunctionDeclaration {
	common.UseMemory(gauge, common.SpecialFunctionDeclarationMemoryUsage)

	return &SpecialFunctionDeclaration{
		Kind:                kind,
		FunctionDeclaration: funcDecl,
	}
}

func (*SpecialFunctionDeclaration) ElementType() ElementType {
	return ElementTypeSpecialFunctionDeclaration
}

func (d *SpecialFunctionDeclaration) StartPosition() Position {
	return d.FunctionDeclaration.StartPosition()
}

func (d *SpecialFunctionDeclaration) EndPosition(memoryGauge common.MemoryGauge) Position {
	return d.FunctionDeclaration.EndPosition(memoryGauge)
}

func (d *SpecialFunctionDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitSpecialFunctionDeclaration(d)
}

func (d *SpecialFunctionDeclaration) Walk(walkChild func(Element)) {
	d.FunctionDeclaration.Walk(walkChild)
}

func (*SpecialFunctionDeclaration) isDeclaration() {}
func (*SpecialFunctionDeclaration) isStatement()   {}

func (d *SpecialFunctionDeclaration) DeclarationIdentifier() *Identifier {
	return d.FunctionDeclaration.DeclarationIdentifier()
}

func (d *SpecialFunctionDeclaration) DeclarationKind() common.DeclarationKind {
	return d.Kind
}

func (d *SpecialFunctionDeclaration) DeclarationAccess() Access {
	return d.FunctionDeclaration.DeclarationAccess()
}

func (d *SpecialFunctionDeclaration) DeclarationMembers() *Members {
	return d.FunctionDeclaration.DeclarationMembers()
}

func (d *SpecialFunctionDeclaration) DeclarationDocString() string {
	return d.FunctionDeclaration.DeclarationDocString()
}

func (d *SpecialFunctionDeclaration) MarshalJSON() ([]byte, error) {
	type Alias SpecialFunctionDeclaration
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "SpecialFunctionDeclaration",
		Range: NewUnmeteredRangeFromPositioned(d),
		Alias: (*Alias)(d),
	})
}
