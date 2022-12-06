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
	AllTypeElements() *StringImportElementOrderedMap
	IsChecking() bool
}

// ImportElement
type ImportElement struct {
	DeclarationKind common.DeclarationKind
	Access          ast.Access
	Type            Type
	ArgumentLabels  []string
}

// ElaborationImport
type ElaborationImport struct {
	Elaboration *Elaboration
}

func variablesToImportElements(f func(func(name string, variable *Variable))) *StringImportElementOrderedMap {

	elements := &StringImportElementOrderedMap{}

	f(func(name string, variable *Variable) {

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
	return variablesToImportElements(i.Elaboration.ForEachGlobalValue)
}

func (i ElaborationImport) AllTypeElements() *StringImportElementOrderedMap {
	return variablesToImportElements(i.Elaboration.ForEachGlobalType)
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

func (i VirtualImport) AllTypeElements() *StringImportElementOrderedMap {
	return i.TypeElements
}

func (VirtualImport) IsChecking() bool {
	return false
}
