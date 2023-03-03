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
)

// programIndices is a container for all indices of a program's declarations
type programIndices struct {
	once sync.Once
	// Use `pragmaDeclarations` instead
	_pragmaDeclarations []*PragmaDeclaration
	// Use `importDeclarations` instead
	_importDeclarations []*ImportDeclaration
	// Use `interfaceDeclarations` instead
	_interfaceDeclarations []*InterfaceDeclaration
	// Use `interfaceDeclarations` instead
	_entitlementDeclarations []*EntitlementDeclaration
	// Use `compositeDeclarations` instead
	_compositeDeclarations []*CompositeDeclaration
	// Use `attachmentDeclarations` instead
	_attachmentDeclarations []*AttachmentDeclaration
	// Use `functionDeclarations()` instead
	_functionDeclarations []*FunctionDeclaration
	// Use `transactionDeclarations()` instead
	_transactionDeclarations []*TransactionDeclaration
	// Use `variableDeclarations()` instead
	_variableDeclarations []*VariableDeclaration
}

func (i *programIndices) pragmaDeclarations(declarations []Declaration) []*PragmaDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._pragmaDeclarations
}

func (i *programIndices) importDeclarations(declarations []Declaration) []*ImportDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._importDeclarations
}

func (i *programIndices) interfaceDeclarations(declarations []Declaration) []*InterfaceDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._interfaceDeclarations
}

func (i *programIndices) entitlementDeclarations(declarations []Declaration) []*EntitlementDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._entitlementDeclarations
}

func (i *programIndices) compositeDeclarations(declarations []Declaration) []*CompositeDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._compositeDeclarations
}

func (i *programIndices) attachmentDeclarations(declarations []Declaration) []*AttachmentDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._attachmentDeclarations
}

func (i *programIndices) functionDeclarations(declarations []Declaration) []*FunctionDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._functionDeclarations
}

func (i *programIndices) transactionDeclarations(declarations []Declaration) []*TransactionDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._transactionDeclarations
}

func (i *programIndices) variableDeclarations(declarations []Declaration) []*VariableDeclaration {
	i.once.Do(i.initializer(declarations))
	return i._variableDeclarations
}

func (i *programIndices) initializer(declarations []Declaration) func() {
	return func() {
		i.init(declarations)
	}
}

func (i *programIndices) init(declarations []Declaration) {

	// Important: allocate instead of nil

	i._pragmaDeclarations = make([]*PragmaDeclaration, 0)
	i._importDeclarations = make([]*ImportDeclaration, 0)
	i._compositeDeclarations = make([]*CompositeDeclaration, 0)
	i._attachmentDeclarations = make([]*AttachmentDeclaration, 0)
	i._interfaceDeclarations = make([]*InterfaceDeclaration, 0)
	i._entitlementDeclarations = make([]*EntitlementDeclaration, 0)
	i._functionDeclarations = make([]*FunctionDeclaration, 0)
	i._transactionDeclarations = make([]*TransactionDeclaration, 0)

	for _, declaration := range declarations {

		switch declaration := declaration.(type) {
		case *PragmaDeclaration:
			i._pragmaDeclarations = append(i._pragmaDeclarations, declaration)

		case *ImportDeclaration:
			i._importDeclarations = append(i._importDeclarations, declaration)

		case *CompositeDeclaration:
			i._compositeDeclarations = append(i._compositeDeclarations, declaration)

		case *AttachmentDeclaration:
			i._attachmentDeclarations = append(i._attachmentDeclarations, declaration)

		case *InterfaceDeclaration:
			i._interfaceDeclarations = append(i._interfaceDeclarations, declaration)

		case *EntitlementDeclaration:
			i._entitlementDeclarations = append(i._entitlementDeclarations, declaration)

		case *FunctionDeclaration:
			i._functionDeclarations = append(i._functionDeclarations, declaration)

		case *TransactionDeclaration:
			i._transactionDeclarations = append(i._transactionDeclarations, declaration)

		case *VariableDeclaration:
			i._variableDeclarations = append(i._variableDeclarations, declaration)
		}
	}
}
