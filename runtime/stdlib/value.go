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

package stdlib

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type StandardLibraryValue struct {
	Type           sema.Type
	Value          interpreter.Value
	Position       *ast.Position
	Name           string
	DocString      string
	ArgumentLabels []string
	Kind           common.DeclarationKind
}

func (v StandardLibraryValue) ValueDeclarationName() string {
	return v.Name
}

func (v StandardLibraryValue) ValueDeclarationValue() interpreter.Value {
	return v.Value
}

func (v StandardLibraryValue) ValueDeclarationType() sema.Type {
	return v.Type
}

func (v StandardLibraryValue) ValueDeclarationDocString() string {
	return v.DocString
}

func (v StandardLibraryValue) ValueDeclarationKind() common.DeclarationKind {
	return v.Kind
}

func (v StandardLibraryValue) ValueDeclarationPosition() *ast.Position {
	return v.Position
}

func (v StandardLibraryValue) ValueDeclarationIsConstant() bool {
	return v.Kind != common.DeclarationKindVariable
}

func (v StandardLibraryValue) ValueDeclarationArgumentLabels() []string {
	return v.ArgumentLabels
}
