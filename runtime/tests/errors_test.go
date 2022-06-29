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

package tests

import (
	"fmt"
	"go/types"
	"reflect"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
)

// TestErrorInterfaceConformance checks whether all the error structs implement
// one of the interfaces.
//
func TestErrorInterfaceConformance(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(
		&packages.Config{
			Mode: packages.NeedImports | packages.NeedTypes,
		},
		"github.com/onflow/cadence/runtime/errors",
	)
	require.NoError(t, err)

	pkg := pkgs[0]
	errorsPkgScope := pkg.Types.Scope()

	// Get the builtin scope. Builtin scope is a parent of any pkg scope
	builtinScope := errorsPkgScope.Parent()

	// Get the builtin 'error' interface type
	errorType := builtinScope.Lookup("error").Type()
	errorInterfaceType, isInterface := errorType.Underlying().(*types.Interface)
	require.True(t, isInterface)

	// Get the 'UserError' interface type
	userErrorType := errorsPkgScope.Lookup("UserError").Type()
	userErrorInterfaceType, isInterface := userErrorType.Underlying().(*types.Interface)
	require.True(t, isInterface)

	// Get the 'InternalError' interface type
	internalErrorType := errorsPkgScope.Lookup("InternalError").Type()
	internalErrorInterfaceType, isInterface := internalErrorType.Underlying().(*types.Interface)
	require.True(t, isInterface)

	// Wrapper errors doesn't implement any interfaces.
	// hence, skip them from the check.
	wrapperErrors := []error{
		parser.Error{},
		interpreter.Error{},
		runtime.Error{},
		interpreter.StackTraceError{},
	}

	errorsToSkip := make(map[string]any)
	for _, err := range wrapperErrors {
		typ := reflect.TypeOf(err)
		fullyQualifiedErrStr := fmt.Sprintf("%s.%s", typ.PkgPath(), typ.Name())
		errorsToSkip[fullyQualifiedErrStr] = nil
	}

	// Iterate through all error structs defined in cadence,
	// and ensure they implement the interfaces.

	pkgs, err = packages.Load(
		&packages.Config{
			Mode: packages.NeedImports | packages.NeedTypes,
		},
		"github.com/onflow/cadence/runtime",
		"github.com/onflow/cadence/runtime/interpreter",
		"github.com/onflow/cadence/runtime/sema",
		"github.com/onflow/cadence/runtime/parser",
		"github.com/onflow/cadence/runtime/stdlib",

		// Currently, doesnt support:
		//"github.com/onflow/cadence/runtime/compiler/wasm",
	)
	require.NoError(t, err)

	for _, pkg := range pkgs {
		// Should test only valid packages
		require.Len(t, pkg.Errors, 0)

		scope := pkg.Types.Scope()

		for _, name := range scope.Names() {
			object := scope.Lookup(name)
			_, ok := object.(*types.TypeName)
			if !ok {
				continue
			}

			implementationType := object.Type()

			// Filter out non 'error' types
			if !types.Implements(implementationType, errorInterfaceType) {
				continue
			}

			// All known error types should implement 'UserError' or 'InternalError'.
			implementsUserError := types.Implements(implementationType, userErrorInterfaceType)
			implementsInternalError := types.Implements(implementationType, internalErrorInterfaceType)

			if implementsUserError && implementsInternalError {
				assert.Fail(t,
					fmt.Sprintf("'%s' implements both 'UserError' and 'InternalError'", implementationType))
			}

			if implementsUserError || implementsInternalError {
				continue
			}

			// Only errors that do not implement above are the wrapper errors
			_, ok = errorsToSkip[implementationType.String()]
			assert.True(
				t,
				ok,
				fmt.Sprintf("'%s' does not implement 'UserError' or 'InternalError'", implementationType),
			)
		}
	}
}
