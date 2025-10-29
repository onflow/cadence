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

package subtype_gen

import (
	"testing"

	"github.com/dave/dst"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsingRules(t *testing.T) {
	t.Parallel()

	rules, err := ParseRules()
	require.NoError(t, err)
	assert.Len(t, rules.Rules, 26)
}

func TestGeneratingCode(t *testing.T) {
	t.Parallel()

	rules, err := ParseRules()
	require.NoError(t, err)

	gen := NewSubTypeCheckGenerator(Config{})
	decls := gen.GenerateCheckSubTypeWithoutEqualityFunction(rules)

	require.Len(t, decls, 1)
	decl := decls[0]

	require.IsType(t, &dst.FuncDecl{}, decl)
	funcDecl := decl.(*dst.FuncDecl)

	// Assert function name
	assert.Equal(t, subtypeCheckFuncName, funcDecl.Name.Name)

	// Assert function body
	statements := funcDecl.Body.List
	require.Len(t, statements, 4)

	// If check for never type
	require.IsType(t, &dst.IfStmt{}, statements[0])
	// Switch statement for simple types
	require.IsType(t, &dst.SwitchStmt{}, statements[1])
	// Type-switch for complex types
	require.IsType(t, &dst.TypeSwitchStmt{}, statements[2])
	// The final return
	require.IsType(t, &dst.ReturnStmt{}, statements[3])
}
