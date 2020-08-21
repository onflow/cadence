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
	// Use `Fields()` instead
	_fields []*FieldDeclaration
	// Use `FieldsByIdentifier()` instead
	_fieldsByIdentifier map[string]*FieldDeclaration
	// All special functions, such as initializers and destructors.
	// Use `SpecialFunctions()` to get all special functions instead,
	// or `Initializers()` and `Destructors()` to get subsets
	_specialFunctions []*SpecialFunctionDeclaration
	// Use `Initializers()` instead
	_initializers []*SpecialFunctionDeclaration
	// Semantically only one destructor is allowed,
	// but the program might illegally declare multiple.
	// Use `Destructors()` instead
	_destructors []*SpecialFunctionDeclaration
	// Use `Functions()`
	_functions []*FunctionDeclaration
	// Use `FunctionsByIdentifier()` instead
	_functionsByIdentifier map[string]*FunctionDeclaration
	// Use `Interfaces()` instead
	_interfaces []*InterfaceDeclaration
	// Use `Composites()` instead
	_composites []*CompositeDeclaration
	// Use `EnumCases()` instead
	_enumCases []*EnumCaseDeclaration
}

func (m *Members) FieldsByIdentifier() map[string]*FieldDeclaration {
	if m._fieldsByIdentifier == nil {
		fields := m.Fields()
		fieldsByIdentifier := make(map[string]*FieldDeclaration, len(fields))
		for _, field := range fields {
			fieldsByIdentifier[field.Identifier.Identifier] = field
		}
		m._fieldsByIdentifier = fieldsByIdentifier
	}
	return m._fieldsByIdentifier
}

func (m *Members) FunctionsByIdentifier() map[string]*FunctionDeclaration {
	if m._functionsByIdentifier == nil {
		functions := m.Functions()
		functionsByIdentifier := make(map[string]*FunctionDeclaration, len(functions))
		for _, function := range functions {
			functionsByIdentifier[function.Identifier.Identifier] = function
		}
		m._functionsByIdentifier = functionsByIdentifier
	}
	return m._functionsByIdentifier
}

func (m *Members) Initializers() []*SpecialFunctionDeclaration {
	if m._initializers == nil {
		initializers := []*SpecialFunctionDeclaration{}
		specialFunctions := m.SpecialFunctions()
		for _, function := range specialFunctions {
			if function.Kind != common.DeclarationKindInitializer {
				continue
			}
			initializers = append(initializers, function)
		}
		m._initializers = initializers
	}
	return m._initializers
}

func (m *Members) Destructors() []*SpecialFunctionDeclaration {
	if m._destructors == nil {
		destructors := []*SpecialFunctionDeclaration{}
		specialFunctions := m.SpecialFunctions()
		for _, function := range specialFunctions {
			if function.Kind != common.DeclarationKindDestructor {
				continue
			}
			destructors = append(destructors, function)
		}
		m._destructors = destructors
	}
	return m._destructors
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

func (m *Members) Fields() []*FieldDeclaration {
	if m._fields == nil {
		m.updateIndices()
	}
	return m._fields
}

func (m *Members) Functions() []*FunctionDeclaration {
	if m._functions == nil {
		m.updateIndices()
	}
	return m._functions
}

func (m *Members) SpecialFunctions() []*SpecialFunctionDeclaration {
	if m._specialFunctions == nil {
		m.updateIndices()
	}
	return m._specialFunctions
}

func (m *Members) Interfaces() []*InterfaceDeclaration {
	if m._interfaces == nil {
		m.updateIndices()
	}
	return m._interfaces
}

func (m *Members) Composites() []*CompositeDeclaration {
	if m._composites == nil {
		m.updateIndices()
	}
	return m._composites
}

func (m *Members) EnumCases() []*EnumCaseDeclaration {
	if m._enumCases == nil {
		m.updateIndices()
	}
	return m._enumCases
}

// updateIndices updates the indices of all declarations
//
func (m *Members) updateIndices() {
	// Important: allocate instead of nil

	m._fields = make([]*FieldDeclaration, 0)
	m._functions = make([]*FunctionDeclaration, 0)
	m._specialFunctions = make([]*SpecialFunctionDeclaration, 0)
	m._interfaces = make([]*InterfaceDeclaration, 0)
	m._composites = make([]*CompositeDeclaration, 0)
	m._enumCases = make([]*EnumCaseDeclaration, 0)

	for _, declaration := range m.Declarations {
		switch declaration := declaration.(type) {
		case *FieldDeclaration:
			m._fields = append(m._fields, declaration)

		case *FunctionDeclaration:
			m._functions = append(m._functions, declaration)

		case *SpecialFunctionDeclaration:
			m._specialFunctions = append(m._specialFunctions, declaration)

		case *InterfaceDeclaration:
			m._interfaces = append(m._interfaces, declaration)

		case *CompositeDeclaration:
			m._composites = append(m._composites, declaration)

		case *EnumCaseDeclaration:
			m._enumCases = append(m._enumCases, declaration)
		}
	}
}
