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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type Variable struct {
	// Type is the type of the variable
	Type Type
	// Pos is the position where the variable was declared
	Pos        *ast.Position
	Identifier string
	// DocString is the optional docstring
	DocString string
	// referencedResourceVariables holds the resource-typed variables referenced by this variable.
	// Only applicable for reference-typed variables. Otherwise, it is nil.
	// This has to be a slice, because some variables can conditionally point to multiple resources.
	// e.g: nil-coalescing operator: `let ref = (&x as &R?) ?? (&y as &R?)`
	referencedResourceVariables []*Variable
	// ArgumentLabels are the argument labels that must be used in an invocation of the variable
	ArgumentLabels  []string
	DeclarationKind common.DeclarationKind
	// Access is the access modifier
	Access ast.Access
	// ActivationDepth is the depth of scopes in which the variable was declared
	ActivationDepth int
	// IsConstant indicates if the variable is read-only
	IsConstant bool
}
