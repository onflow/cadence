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

type VariableDeclaration struct {
	Access            Access
	IsConstant        bool
	Identifier        Identifier
	TypeAnnotation    *TypeAnnotation
	Value             Expression
	Transfer          *Transfer
	StartPos          Position `json:"-"`
	SecondTransfer    *Transfer
	SecondValue       Expression
	ParentIfStatement *IfStatement `json:"-"`
	DocString         string
}

var _ Element = &VariableDeclaration{}
var _ Statement = &VariableDeclaration{}
var _ Declaration = &VariableDeclaration{}

func (*VariableDeclaration) ElementType() ElementType {
	return ElementTypeVariableDeclaration
}

func (d *VariableDeclaration) StartPosition() Position {
	return d.StartPos
}

func (d *VariableDeclaration) EndPosition() Position {
	if d.SecondValue != nil {
		return d.SecondValue.EndPosition()
	}
	return d.Value.EndPosition()
}

func (*VariableDeclaration) isIfStatementTest() {}

func (*VariableDeclaration) isDeclaration() {}

func (*VariableDeclaration) isStatement() {}

func (d *VariableDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitVariableDeclaration(d)
}

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

func (d *VariableDeclaration) MarshalJSON() ([]byte, error) {
	type Alias VariableDeclaration
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "VariableDeclaration",
		Range: NewRangeFromPositioned(d),
		Alias: (*Alias)(d),
	})
}
