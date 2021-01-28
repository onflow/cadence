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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// Import

type Import interface {
	AllValueElements() map[string]ImportElement
	IsImportableValue(name string) bool
	AllTypeElements() map[string]ImportElement
	IsImportableType(name string) bool
	IsChecking() bool
}

// ImportElement
//
type ImportElement struct {
	DeclarationKind common.DeclarationKind
	Access          ast.Access
	Type            Type
	ArgumentLabels  []string
}

// ElaborationImport
//
type ElaborationImport struct {
	Elaboration *Elaboration
}

func variablesToImportElements(variables map[string]*Variable) map[string]ImportElement {
	elements := make(map[string]ImportElement, len(variables))
	for name, variable := range variables {
		elements[name] = ImportElement{
			DeclarationKind: variable.DeclarationKind,
			Access:          variable.Access,
			Type:            variable.Type,
			ArgumentLabels:  variable.ArgumentLabels,
		}
	}
	return elements
}

func (i ElaborationImport) AllValueElements() map[string]ImportElement {
	return variablesToImportElements(i.Elaboration.GlobalValues)
}

func (i ElaborationImport) IsImportableValue(name string) bool {
	if _, ok := BaseValues[name]; ok {
		return false
	}

	_, isPredeclaredValue := i.Elaboration.EffectivePredeclaredValues[name]
	return !isPredeclaredValue
}

func (i ElaborationImport) AllTypeElements() map[string]ImportElement {
	return variablesToImportElements(i.Elaboration.GlobalTypes)
}

func (i ElaborationImport) IsImportableType(name string) bool {
	if _, ok := baseTypes[name]; ok {
		return false
	}

	_, isPredeclaredType := i.Elaboration.EffectivePredeclaredTypes[name]
	return !isPredeclaredType
}

func (i ElaborationImport) IsChecking() bool {
	return i.Elaboration.IsChecking()
}

// VirtualImport

type VirtualImport struct {
	ValueElements map[string]ImportElement
	TypeElements  map[string]ImportElement
}

func (i VirtualImport) AllValueElements() map[string]ImportElement {
	return i.ValueElements
}

func (i VirtualImport) IsImportableValue(_ string) bool {
	return true
}

func (i VirtualImport) AllTypeElements() map[string]ImportElement {
	return i.TypeElements
}

func (VirtualImport) IsImportableType(_ string) bool {
	return true
}

func (VirtualImport) IsChecking() bool {
	return false
}
