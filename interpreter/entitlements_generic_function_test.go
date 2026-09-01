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

// This test lives in package `interpreter` (not `interpreter_test`) so it can
// exercise the unexported authorization rewriter `semaTypeWithStrippedEntitlements`
// directly on a generic function type.

package interpreter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

// TestSemaTypeWithStrippedEntitlements_GenericFunctionBinderIdentity checks that
// rewriting a generic function type preserves type parameter binder identity.
//
// The rewriters must not clone the type parameters: the `GenericType` nodes in the
// parameter and return types, and the `TypeArgumentsCheck` callback, identify the
// binders by pointer identity. If the binders were cloned while those references
// kept pointing at the originals, type-argument resolution would fail and
// `TypeArgumentsCheck` would be silently skipped.
//
// The function type mirrors the shape of the built-in `revertibleRandom`:
// a single type parameter `T`, used in the parameter and return types, plus a
// `TypeArgumentsCheck` callback that closes over the *TypeParameter.
func TestSemaTypeWithStrippedEntitlements_GenericFunctionBinderIdentity(t *testing.T) {

	t.Parallel()

	originalBinder := &sema.TypeParameter{
		Name:      "T",
		TypeBound: sema.FixedSizeUnsignedIntegerType,
	}

	genericType := &sema.GenericType{TypeParameter: originalBinder}

	var observed sema.Type
	observedSet := false

	original := &sema.FunctionType{
		TypeParameters: []*sema.TypeParameter{originalBinder},
		Parameters: []sema.Parameter{
			{
				Identifier:     "modulo",
				TypeAnnotation: sema.NewTypeAnnotation(genericType),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(genericType),
		TypeArgumentsCheck: func(
			_ common.MemoryGauge,
			typeArguments *sema.TypeParameterTypeOrderedMap,
			_ []*ast.TypeAnnotation,
			_ ast.HasPosition,
			_ func(err error),
		) {
			// Closes over the original binder and looks it up by identity.
			observed, observedSet = typeArguments.Get(originalBinder)
		},
	}

	rewritten, ok := semaTypeWithStrippedEntitlements(nil, original).(*sema.FunctionType)
	require.True(t, ok)

	// The type parameter binder is preserved, not cloned.
	require.Len(t, rewritten.TypeParameters, 1)
	require.Same(t, originalBinder, rewritten.TypeParameters[0])

	// The `GenericType` references in the return and parameter types still reference
	// the same binder that appears in the rewritten type's TypeParameters, so the
	// invocation checker (which keys type arguments by TypeParameters) stays consistent.
	returnGeneric, ok := rewritten.ReturnTypeAnnotation.Type.(*sema.GenericType)
	require.True(t, ok)
	require.Same(t, rewritten.TypeParameters[0], returnGeneric.TypeParameter)

	parameterGeneric, ok := rewritten.Parameters[0].TypeAnnotation.Type.(*sema.GenericType)
	require.True(t, ok)
	require.Same(t, rewritten.TypeParameters[0], parameterGeneric.TypeParameter)

	// Resolution keyed by the rewritten type's binder succeeds.
	typeArguments := &sema.TypeParameterTypeOrderedMap{}
	typeArguments.Set(rewritten.TypeParameters[0], sema.UInt64Type)

	require.Same(t, sema.UInt64Type, rewritten.ReturnTypeAnnotation.Type.Resolve(typeArguments))
	require.Same(t, sema.UInt64Type, rewritten.Parameters[0].TypeAnnotation.Type.Resolve(typeArguments))

	// The `TypeArgumentsCheck` callback, invoked with a map keyed by the rewritten
	// type's binder, observes the type argument through the binder it closed over.
	require.NotNil(t, rewritten.TypeArgumentsCheck)
	rewritten.TypeArgumentsCheck(nil, typeArguments, nil, ast.EmptyRange, func(err error) {})
	require.True(t, observedSet)
	require.Same(t, sema.UInt64Type, observed)
}
