/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

// Members

type Members struct {
	declarations []Declaration
	indices      memberIndices
}

func NewMembers(memoryGauge common.MemoryGauge, declarations []Declaration) *Members {
	common.UseMemory(memoryGauge, common.NewMembersMemoryUsage(len(declarations)))
	return NewUnmeteredMembers(declarations)
}

func NewUnmeteredMembers(declarations []Declaration) *Members {
	return &Members{
		declarations: declarations,
	}
}

func (m *Members) Declarations() []Declaration {
	return m.declarations
}

func (m *Members) Fields() []*FieldDeclaration {
	return m.indices.Fields(m.declarations)
}

func (m *Members) Functions() []*FunctionDeclaration {
	return m.indices.Functions(m.declarations)
}

func (m *Members) SpecialFunctions() []*SpecialFunctionDeclaration {
	return m.indices.SpecialFunctions(m.declarations)
}

func (m *Members) Interfaces() []*InterfaceDeclaration {
	return m.indices.Interfaces(m.declarations)
}

func (m *Members) Entitlements() []*EntitlementDeclaration {
	return m.indices.Entitlements(m.declarations)
}

func (m *Members) EntitlementMaps() []*EntitlementMappingDeclaration {
	return m.indices.EntitlementMappings(m.declarations)
}

func (m *Members) Composites() []*CompositeDeclaration {
	return m.indices.Composites(m.declarations)
}

func (m *Members) Attachments() []*AttachmentDeclaration {
	return m.indices.Attachments(m.declarations)
}

func (m *Members) EnumCases() []*EnumCaseDeclaration {
	return m.indices.EnumCases(m.declarations)
}

func (m *Members) Pragmas() []*PragmaDeclaration {
	return m.indices.Pragmas(m.declarations)
}

func (m *Members) FieldsByIdentifier() map[string]*FieldDeclaration {
	return m.indices.FieldsByIdentifier(m.declarations)
}

func (m *Members) FunctionsByIdentifier() map[string]*FunctionDeclaration {
	return m.indices.FunctionsByIdentifier(m.declarations)
}

func (m *Members) CompositesByIdentifier() map[string]*CompositeDeclaration {
	return m.indices.CompositesByIdentifier(m.declarations)
}

func (m *Members) AttachmentsByIdentifier() map[string]*AttachmentDeclaration {
	return m.indices.AttachmentsByIdentifier(m.declarations)
}

func (m *Members) EntitlementsByIdentifier() map[string]*EntitlementDeclaration {
	return m.indices.EntitlementsByIdentifier(m.declarations)
}

func (m *Members) EntitlementMappingsByIdentifier() map[string]*EntitlementMappingDeclaration {
	return m.indices.EntitlementMappingsByIdentifier(m.declarations)
}

func (m *Members) InterfacesByIdentifier() map[string]*InterfaceDeclaration {
	return m.indices.InterfacesByIdentifier(m.declarations)
}

func (m *Members) Initializers() []*SpecialFunctionDeclaration {
	return m.indices.Initializers(m.declarations)
}

func (m *Members) FieldPosition(name string, compositeKind common.CompositeKind) Position {
	if compositeKind == common.CompositeKindEvent {
		parameters := m.Initializers()[0].FunctionDeclaration.ParameterList.ParametersByIdentifier()
		parameter := parameters[name]
		return parameter.Identifier.Pos
	} else {
		fields := m.FieldsByIdentifier()
		field := fields[name]
		return field.Identifier.Pos
	}
}

func (m *Members) MarshalJSON() ([]byte, error) {
	type Alias Members
	return json.Marshal(&struct {
		*Alias
		Declarations []Declaration
	}{
		Declarations: m.declarations,
		Alias:        (*Alias)(m),
	})
}

var membersStartDoc prettier.Doc = prettier.Text("{")
var membersEndDoc prettier.Doc = prettier.Text("}")
var membersEmptyDoc prettier.Doc = prettier.Text("{}")

func (m *Members) docWithNoBraces() prettier.Concat {
	var docs []prettier.Doc

	for _, decl := range m.declarations {
		docs = append(
			docs,
			prettier.Concat{
				prettier.HardLine{},
				decl.Doc(),
			},
		)
	}

	return prettier.Concat{
		prettier.Indent{
			Doc: prettier.Join(
				prettier.HardLine{},
				docs...,
			),
		},
		prettier.HardLine{},
	}
}

func (m *Members) Doc() prettier.Doc {
	if len(m.declarations) == 0 {
		return membersEmptyDoc
	}

	membersDoc := m.docWithNoBraces()
	membersDoc = append(prettier.Concat{membersStartDoc}, membersDoc...)
	membersDoc = append(membersDoc, membersEndDoc)
	return membersDoc
}
