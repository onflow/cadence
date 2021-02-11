/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

func (d *FunctionDeclaration) StartPosition() Position {
	return d.StartPos
}

func (d *FunctionDeclaration) EndPosition() Position {
	if d.FunctionBlock != nil {
		return d.FunctionBlock.EndPosition()
	}
	if d.ReturnTypeAnnotation != nil {
		return d.ReturnTypeAnnotation.EndPosition()
	}
	return d.ParameterList.EndPosition()
}

func (d *FunctionDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitFunctionDeclaration(d)
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

func (d *FunctionDeclaration) ToExpression() *FunctionExpression {
	return &FunctionExpression{
		ParameterList:        d.ParameterList,
		ReturnTypeAnnotation: d.ReturnTypeAnnotation,
		FunctionBlock:        d.FunctionBlock,
		StartPos:             d.StartPos,
	}
}

func (d *FunctionDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *FunctionDeclaration) MarshalJSON() ([]byte, error) {
	type Alias FunctionDeclaration
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "FunctionDeclaration",
		Range: NewRangeFromPositioned(d),
		Alias: (*Alias)(d),
	})
}

// SpecialFunctionDeclaration

type SpecialFunctionDeclaration struct {
	Kind                common.DeclarationKind
	FunctionDeclaration *FunctionDeclaration
}

func (d *SpecialFunctionDeclaration) StartPosition() Position {
	return d.FunctionDeclaration.StartPosition()
}

func (d *SpecialFunctionDeclaration) EndPosition() Position {
	return d.FunctionDeclaration.EndPosition()
}

func (d *SpecialFunctionDeclaration) Accept(visitor Visitor) Repr {
	return d.FunctionDeclaration.Accept(visitor)
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
	return nil
}

func (d *SpecialFunctionDeclaration) MarshalJSON() ([]byte, error) {
	type Alias SpecialFunctionDeclaration
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "SpecialFunctionDeclaration",
		Range: NewRangeFromPositioned(d),
		Alias: (*Alias)(d),
	})
}
