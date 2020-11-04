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

import "github.com/onflow/cadence/runtime/common"

// Members

type Members struct {
	Declarations []Declaration
	indices      memberIndices
}

func (m *Members) Fields() []*FieldDeclaration {
	return m.indices.Fields(m.Declarations)
}

func (m *Members) Functions() []*FunctionDeclaration {
	return m.indices.Functions(m.Declarations)
}

func (m *Members) SpecialFunctions() []*SpecialFunctionDeclaration {
	return m.indices.SpecialFunctions(m.Declarations)
}

func (m *Members) Interfaces() []*InterfaceDeclaration {
	return m.indices.Interfaces(m.Declarations)
}

func (m *Members) Composites() []*CompositeDeclaration {
	return m.indices.Composites(m.Declarations)
}

func (m *Members) EnumCases() []*EnumCaseDeclaration {
	return m.indices.EnumCases(m.Declarations)
}

func (m *Members) FieldsByIdentifier() map[string]*FieldDeclaration {
	return m.indices.FieldsByIdentifier(m.Declarations)
}

func (m *Members) FunctionsByIdentifier() map[string]*FunctionDeclaration {
	return m.indices.FunctionsByIdentifier(m.Declarations)
}

func (m *Members) Initializers() []*SpecialFunctionDeclaration {
	return m.indices.Initializers(m.Declarations)
}

func (m *Members) Destructors() []*SpecialFunctionDeclaration {
	return m.indices.Destructors(m.Declarations)
}

// Destructor returns the first destructor, if any
func (m *Members) Destructor() *SpecialFunctionDeclaration {
	destructors := m.Destructors()
	if len(destructors) == 0 {
		return nil
	}
	return destructors[0]
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
