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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckPragmaInvalidExpr(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  #"string"
	`)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.InvalidPragmaError{}, errs[0])
}

func TestCheckPragmaValidIdentifierExpr(t *testing.T) {

	t.Parallel()
	_, err := ParseAndCheck(t, `
		#pedantic
	`)

	require.NoError(t, err)
}

func TestCheckPragmaValidInvocationExpr(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		#version("1.0")
	`)

	require.NoError(t, err)
}

func TestCheckPragmaInvalidLocation(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  fun test() {
		  #version
	  }
	`)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckPragmaInvalidInvocationExprNonStringExprArgument(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		#version(y)
	`)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.InvalidPragmaError{Message: "invalid arguments"}, errs[0])
}

func TestCheckPragmaInvalidInvocationExprTypeArgs(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		#version<X>()
	`)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.InvalidPragmaError{Message: "type arguments not supported"}, errs[0])
}
