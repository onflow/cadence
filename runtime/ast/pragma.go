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

// Pragma

type PragmaDeclaration struct {
	Expression Expression
	Range
}

var _ Declaration = &PragmaDeclaration{}

func NewPragmaDeclaration(gauge common.MemoryGauge, expression Expression, declRange Range) *PragmaDeclaration {
	common.UseMemory(gauge, common.PragmaDeclarationMemoryUsage)

	return &PragmaDeclaration{
		Expression: expression,
		Range:      declRange,
	}
}

func (*PragmaDeclaration) isDeclaration() {}

func (*PragmaDeclaration) isStatement() {}

func (d *PragmaDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitPragmaDeclaration(d)
}

func (d *PragmaDeclaration) Walk(walkChild func(Element)) {
	walkChild(d.Expression)
}

func (d *PragmaDeclaration) DeclarationIdentifier() *Identifier {
	return nil
}

func (d *PragmaDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindPragma
}

func (d *PragmaDeclaration) DeclarationAccess() Access {
	return AccessNotSpecified
}

func (d *PragmaDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *PragmaDeclaration) DeclarationDocString() string {
	return ""
}

func (d *PragmaDeclaration) MarshalJSON() ([]byte, error) {
	type Alias PragmaDeclaration
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "PragmaDeclaration",
		Alias: (*Alias)(d),
	})
}

func (d *PragmaDeclaration) Doc() prettier.Doc {
	return prettier.Concat{
		prettier.Text("#"),
		d.Expression.Doc(),
	}
}

func (d *PragmaDeclaration) String() string {
	return Prettier(d)
}
