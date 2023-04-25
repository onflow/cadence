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

// ImportDeclaration

type ImportDeclaration struct {
	Location    common.Location
	Identifiers []Identifier
	Range
	LocationPos Position
}

var _ Element = &ImportDeclaration{}
var _ Declaration = &ImportDeclaration{}

func NewImportDeclaration(
	gauge common.MemoryGauge,
	identifiers []Identifier,
	location common.Location,
	declRange Range,
	locationPos Position,
) *ImportDeclaration {
	common.UseMemory(gauge, common.ImportDeclarationMemoryUsage)

	return &ImportDeclaration{
		Identifiers: identifiers,
		Location:    location,
		Range:       declRange,
		LocationPos: locationPos,
	}
}

func (*ImportDeclaration) ElementType() ElementType {
	return ElementTypeImportDeclaration
}

func (*ImportDeclaration) isDeclaration() {}

func (*ImportDeclaration) isStatement() {}

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
		*Alias
		Type string
	}{
		Type:  "ImportDeclaration",
		Alias: (*Alias)(d),
	})
}

const importDeclarationImportKeywordDoc = prettier.Text("import")
const importDeclarationFromKeywordDoc = prettier.Text("from ")

var importDeclarationSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func (d *ImportDeclaration) Doc() prettier.Doc {
	doc := prettier.Concat{
		importDeclarationImportKeywordDoc,
	}

	if len(d.Identifiers) > 0 {

		identifiersDoc := prettier.Concat{
			prettier.Line{},
		}

		for i, identifier := range d.Identifiers {
			if i > 0 {
				identifiersDoc = append(
					identifiersDoc,
					importDeclarationSeparatorDoc,
				)
			}

			identifiersDoc = append(
				identifiersDoc,
				prettier.Text(identifier.Identifier),
			)
		}

		identifiersDoc = append(
			identifiersDoc,
			prettier.Line{},
			importDeclarationFromKeywordDoc,
		)

		doc = append(
			doc,
			prettier.Group{
				Doc: prettier.Indent{
					Doc: identifiersDoc,
				},
			},
		)
	} else {
		doc = append(
			doc,
			prettier.Space,
		)
	}

	return append(
		doc,
		LocationDoc(d.Location),
	)
}

func (d *ImportDeclaration) String() string {
	return Prettier(d)
}

func LocationDoc(location common.Location) prettier.Doc {
	switch location := location.(type) {
	case common.AddressLocation:
		return prettier.Text(location.Address.ShortHexWithPrefix())
	case common.IdentifierLocation:
		return prettier.Text(location)
	case common.StringLocation:
		return prettier.Text(QuoteString(string(location)))
	default:
		return nil
	}
}
