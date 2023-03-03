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
	"sync"

	"github.com/onflow/cadence/runtime/common"
)

// programIndices is a container for all indices of members
type memberIndices struct {
	once sync.Once
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
	// Use `CompositesByIdentifier()` instead
	_compositesByIdentifier map[string]*CompositeDeclaration
	// Use `AttachmentsByIdentifier()` instead
	_attachmentsByIdentifier map[string]*AttachmentDeclaration
	// Use `InterfacesByIdentifier()` instead
	_interfacesByIdentifier map[string]*InterfaceDeclaration
	// Use `EntitlementsByIdentifier()` instead
	_entitlementsByIdentifier map[string]*EntitlementDeclaration
	// Use `Interfaces()` instead
	_interfaces []*InterfaceDeclaration
	// Use `Entitlements()` instead
	_entitlements []*EntitlementDeclaration
	// Use `Composites()` instead
	_composites []*CompositeDeclaration
	// Use `Attachments()` instead
	_attachments []*AttachmentDeclaration
	// Use `EnumCases()` instead
	_enumCases []*EnumCaseDeclaration
}

func (i *memberIndices) FieldsByIdentifier(declarations []Declaration) map[string]*FieldDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._fieldsByIdentifier
}

func (i *memberIndices) FunctionsByIdentifier(declarations []Declaration) map[string]*FunctionDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._functionsByIdentifier
}

func (i *memberIndices) CompositesByIdentifier(declarations []Declaration) map[string]*CompositeDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._compositesByIdentifier
}

func (i *memberIndices) AttachmentsByIdentifier(declarations []Declaration) map[string]*AttachmentDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._attachmentsByIdentifier
}

func (i *memberIndices) InterfacesByIdentifier(declarations []Declaration) map[string]*InterfaceDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._interfacesByIdentifier
}

func (i *memberIndices) EntitlementsByIdentifier(declarations []Declaration) map[string]*EntitlementDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._entitlementsByIdentifier
}

func (i *memberIndices) Initializers(declarations []Declaration) []*SpecialFunctionDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._initializers
}

func (i *memberIndices) Destructors(declarations []Declaration) []*SpecialFunctionDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._destructors
}

func (i *memberIndices) Fields(declarations []Declaration) []*FieldDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._fields
}

func (i *memberIndices) Functions(declarations []Declaration) []*FunctionDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._functions
}

func (i *memberIndices) SpecialFunctions(declarations []Declaration) []*SpecialFunctionDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._specialFunctions
}

func (i *memberIndices) Interfaces(declarations []Declaration) []*InterfaceDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._interfaces
}

func (i *memberIndices) Entitlements(declarations []Declaration) []*EntitlementDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._entitlements
}

func (i *memberIndices) Composites(declarations []Declaration) []*CompositeDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._composites
}

func (i *memberIndices) Attachments(declarations []Declaration) []*AttachmentDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._attachments
}

func (i *memberIndices) EnumCases(declarations []Declaration) []*EnumCaseDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._enumCases
}

func (i *memberIndices) initializer(declarations []Declaration) func() {
	return func() {
		i.init(declarations)
	}
}

func (i *memberIndices) init(declarations []Declaration) {
	// Important: allocate instead of nil

	i._fields = make([]*FieldDeclaration, 0)
	i._fieldsByIdentifier = make(map[string]*FieldDeclaration)

	i._functions = make([]*FunctionDeclaration, 0)
	i._functionsByIdentifier = make(map[string]*FunctionDeclaration)

	i._specialFunctions = make([]*SpecialFunctionDeclaration, 0)
	i._destructors = make([]*SpecialFunctionDeclaration, 0)
	i._initializers = make([]*SpecialFunctionDeclaration, 0)

	i._composites = make([]*CompositeDeclaration, 0)
	i._compositesByIdentifier = make(map[string]*CompositeDeclaration)

	i._attachments = make([]*AttachmentDeclaration, 0)
	i._attachmentsByIdentifier = make(map[string]*AttachmentDeclaration)

	i._interfaces = make([]*InterfaceDeclaration, 0)
	i._interfacesByIdentifier = make(map[string]*InterfaceDeclaration)

	i._entitlements = make([]*EntitlementDeclaration, 0)
	i._entitlementsByIdentifier = make(map[string]*EntitlementDeclaration)

	i._enumCases = make([]*EnumCaseDeclaration, 0)

	for _, declaration := range declarations {
		switch declaration := declaration.(type) {
		case *FieldDeclaration:
			i._fields = append(i._fields, declaration)
			i._fieldsByIdentifier[declaration.Identifier.Identifier] = declaration

		case *FunctionDeclaration:
			i._functions = append(i._functions, declaration)
			i._functionsByIdentifier[declaration.Identifier.Identifier] = declaration

		case *SpecialFunctionDeclaration:
			i._specialFunctions = append(i._specialFunctions, declaration)

			switch declaration.Kind {
			case common.DeclarationKindInitializer:
				i._initializers = append(i._initializers, declaration)
			case common.DeclarationKindDestructor:
				i._destructors = append(i._destructors, declaration)
			}

		case *EntitlementDeclaration:
			i._entitlements = append(i._entitlements, declaration)
			i._entitlementsByIdentifier[declaration.Identifier.Identifier] = declaration

		case *InterfaceDeclaration:
			i._interfaces = append(i._interfaces, declaration)
			i._interfacesByIdentifier[declaration.Identifier.Identifier] = declaration

		case *CompositeDeclaration:
			i._composites = append(i._composites, declaration)
			i._compositesByIdentifier[declaration.Identifier.Identifier] = declaration

		case *AttachmentDeclaration:
			i._attachments = append(i._attachments, declaration)
			i._attachmentsByIdentifier[declaration.Identifier.Identifier] = declaration

		case *EnumCaseDeclaration:
			i._enumCases = append(i._enumCases, declaration)
		}
	}
}
