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

// ImportDeclaration

type ImportDeclaration struct {
	Identifiers []Identifier
	Location    common.Location
	LocationPos Position
	Range
}

func (*ImportDeclaration) isDeclaration() {}

func (*ImportDeclaration) isStatement() {}

func (d *ImportDeclaration) Accept(visitor Visitor) Repr {
	return visitor.VisitImportDeclaration(d)
}

func (*ImportDeclaration) Walk(_ func(Element)) {
	// NO-OP
}

func (d *ImportDeclaration) DeclarationIdentifier() *Identifier {
	return nil
}

func (d *ImportDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindImport
}

func (d *ImportDeclaration) DeclarationAccess() Access {
	return AccessNotSpecified
}

func (d *ImportDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *ImportDeclaration) DeclarationDocString() string {
	return ""
}

func (d *ImportDeclaration) MarshalJSON() ([]byte, error) {
	type Alias ImportDeclaration
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "ImportDeclaration",
		Alias: (*Alias)(d),
	})
}
