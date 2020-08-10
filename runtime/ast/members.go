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
	// Use `InterfaceDeclarations()` instead
	_interfaceDeclarations []*InterfaceDeclaration
	// Use `CompositeDeclarations()` instead
	_compositeDeclarations []*CompositeDeclaration
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
		m.updateDeclarations()
	}
	return m._fields
}

func (m *Members) Functions() []*FunctionDeclaration {
	if m._functions == nil {
		m.updateDeclarations()
	}
	return m._functions
}

func (m *Members) SpecialFunctions() []*SpecialFunctionDeclaration {
	if m._specialFunctions == nil {
		m.updateDeclarations()
	}
	return m._specialFunctions
}

func (m *Members) InterfaceDeclarations() []*InterfaceDeclaration {
	if m._interfaceDeclarations == nil {
		m.updateDeclarations()
	}
	return m._interfaceDeclarations
}

func (m *Members) CompositeDeclarations() []*CompositeDeclaration {
	if m._compositeDeclarations == nil {
		m.updateDeclarations()
	}
	return m._compositeDeclarations
}

func (m *Members) updateDeclarations() {
	// Important: allocate instead of nil

	m._fields = make([]*FieldDeclaration, 0)
	m._functions = make([]*FunctionDeclaration, 0)
	m._specialFunctions = make([]*SpecialFunctionDeclaration, 0)
	m._interfaceDeclarations = make([]*InterfaceDeclaration, 0)
	m._compositeDeclarations = make([]*CompositeDeclaration, 0)

	for _, declaration := range m.Declarations {
		switch declaration := declaration.(type) {
		case *FieldDeclaration:
			m._fields = append(m._fields, declaration)

		case *FunctionDeclaration:
			m._functions = append(m._functions, declaration)

		case *SpecialFunctionDeclaration:
			m._specialFunctions = append(m._specialFunctions, declaration)

		case *InterfaceDeclaration:
			m._interfaceDeclarations = append(m._interfaceDeclarations, declaration)

		case *CompositeDeclaration:
			m._compositeDeclarations = append(m._compositeDeclarations, declaration)
		}
	}
}
