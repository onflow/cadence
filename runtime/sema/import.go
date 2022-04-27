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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// Import

type Import interface {
	AllValueElements() *StringImportElementOrderedMap
	IsImportableValue(name string) bool
	AllTypeElements() *StringImportElementOrderedMap
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

func variablesToImportElements(variables *StringVariableOrderedMap) *StringImportElementOrderedMap {

	elements := NewStringImportElementOrderedMap()

	variables.Foreach(func(name string, variable *Variable) {

		elements.Set(name, ImportElement{
			DeclarationKind: variable.DeclarationKind,
			Access:          variable.Access,
			Type:            variable.Type,
			ArgumentLabels:  variable.ArgumentLabels,
		})
	})

	return elements
}

func (i ElaborationImport) AllValueElements() *StringImportElementOrderedMap {
	return variablesToImportElements(i.Elaboration.GlobalValues)
}

func (i ElaborationImport) IsImportableValue(name string) bool {
	if BaseValueActivation.Find(name) != nil {
		return false
	}

	_, isPredeclaredValue := i.Elaboration.EffectivePredeclaredValues[name]
	return !isPredeclaredValue
}

func (i ElaborationImport) AllTypeElements() *StringImportElementOrderedMap {
	return variablesToImportElements(i.Elaboration.GlobalTypes)
}

func (i ElaborationImport) IsImportableType(name string) bool {
	if BaseTypeActivation.Find(name) != nil {
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
	ValueElements *StringImportElementOrderedMap
	TypeElements  *StringImportElementOrderedMap
}

func (i VirtualImport) AllValueElements() *StringImportElementOrderedMap {
	return i.ValueElements
}

func (i VirtualImport) IsImportableValue(_ string) bool {
	return true
}

func (i VirtualImport) AllTypeElements() *StringImportElementOrderedMap {
	return i.TypeElements
}

func (VirtualImport) IsImportableType(_ string) bool {
	return true
}

func (VirtualImport) IsChecking() bool {
	return false
}
