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

package runtime

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type ValueDeclaration struct {
	Name           string
	Type           sema.Type
	DocString      string
	Kind           common.DeclarationKind
	IsConstant     bool
	ArgumentLabels []string
	Available      func(common.Location) bool
	Value          interpreter.Value
}

func (v ValueDeclaration) ValueDeclarationName() string {
	return v.Name
}

func (v ValueDeclaration) ValueDeclarationType() sema.Type {
	return v.Type
}

func (v ValueDeclaration) ValueDeclarationDocString() string {
	return v.DocString
}

func (v ValueDeclaration) ValueDeclarationValue() interpreter.Value {
	return v.Value
}

func (v ValueDeclaration) ValueDeclarationKind() common.DeclarationKind {
	return v.Kind
}

func (v ValueDeclaration) ValueDeclarationPosition() ast.Position {
	return ast.Position{}
}

func (v ValueDeclaration) ValueDeclarationIsConstant() bool {
	return v.IsConstant
}

func (v ValueDeclaration) ValueDeclarationArgumentLabels() []string {
	return v.ArgumentLabels
}

func (v ValueDeclaration) ValueDeclarationAvailable(location common.Location) bool {
	if v.Available == nil {
		return true
	}
	return v.Available(location)
}
