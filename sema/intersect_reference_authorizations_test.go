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

// This test lives in package `sema` (not `sema_test`) so it can exercise the
// unexported authorization rewriter `intersectReferenceAuthorizationsInType`
// directly on a generic function type.

package sema

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
)

// TestIntersectReferenceAuthorizationsInType_GenericFunctionBinderIdentity checks
// that rewriting a generic function type preserves type parameter binder identity.
//
// The rewriter must not clone the type parameters: the `GenericType` nodes in the
// parameter and return types, and the `TypeArgumentsCheck` callback, identify the
// binders by pointer identity. If the binders were cloned while those references
// kept pointing at the originals, type-argument resolution would fail and
// `TypeArgumentsCheck` would be silently skipped.
//
// The rewriter must actually rebuild the function type for this to test anything,
// so that the identity of everything else has to be shown to survive.
// A `callback` parameter carrying an authorized reference in its own parameter
// forces that: the parameter of a parameter is covariant again, so it is capped.
//
// A plain `ref` parameter carrying the same authorized reference is included
// alongside it, and asserted to be left as declared:
// a reference in a parameter is contravariant,
// so capping it would weaken what callers are required to pass.
func TestIntersectReferenceAuthorizationsInType_GenericFunctionBinderIdentity(t *testing.T) {

	t.Parallel()

	originalBinder := &TypeParameter{
		Name:      "T",
		TypeBound: IntegerType,
	}

	genericType := &GenericType{TypeParameter: originalBinder}

	authorization := NewEntitlementSetAccess(
		[]*EntitlementType{MutateType},
		Conjunction,
	)

	// An authorized reference `auth(Mutate) &Int`.
	authorizedReferenceType := NewReferenceType(
		nil,
		authorization,
		IntType,
	)

	// `fun(auth(Mutate) &Int): Void`, which places the authorized reference
	// in a covariant position when the callback type itself is a parameter,
	// so that intersecting against an unauthorized outer authorization
	// reduces it to `&Int`, forcing a rebuild.
	callbackType := NewSimpleFunctionType(
		FunctionPurityImpure,
		[]Parameter{
			{
				Identifier:     "ref",
				TypeAnnotation: NewTypeAnnotation(authorizedReferenceType),
			},
		},
		VoidTypeAnnotation,
	)

	var observed Type
	observedSet := false

	original := &FunctionType{
		TypeParameters: []*TypeParameter{originalBinder},
		Parameters: []Parameter{
			{
				Identifier:     "value",
				TypeAnnotation: NewTypeAnnotation(genericType),
			},
			{
				Identifier:     "ref",
				TypeAnnotation: NewTypeAnnotation(authorizedReferenceType),
			},
			{
				Identifier:     "callback",
				TypeAnnotation: NewTypeAnnotation(callbackType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(genericType),
		TypeArgumentsCheck: func(
			_ common.MemoryGauge,
			typeArguments *TypeParameterTypeOrderedMap,
			_ []*ast.TypeAnnotation,
			_ ast.HasPosition,
			_ func(err error),
		) {
			// Closes over the original binder and looks it up by identity.
			observed, observedSet = typeArguments.Get(originalBinder)
		},
	}

	rewritten, ok := intersectReferenceAuthorizationsInType(
		nil,
		original,
		UnauthorizedAccess,
	).(*FunctionType)
	require.True(t, ok)

	// The rewriter actually rebuilt the type: the authorized reference nested in the
	// callback's parameter is covariant, and was reduced to an unauthorized reference.
	require.NotSame(t, original, rewritten)
	rewrittenCallback, ok := rewritten.Parameters[2].TypeAnnotation.Type.(*FunctionType)
	require.True(t, ok)
	rewrittenCallbackRef, ok := rewrittenCallback.Parameters[0].TypeAnnotation.Type.(*ReferenceType)
	require.True(t, ok)
	require.Equal(t, UnauthorizedAccess, rewrittenCallbackRef.Authorization)

	// The authorized reference in the parameter itself is contravariant,
	// and was left as declared: capping it would weaken
	// what callers are required to pass.
	rewrittenRef, ok := rewritten.Parameters[1].TypeAnnotation.Type.(*ReferenceType)
	require.True(t, ok)
	require.Equal(t, authorization, rewrittenRef.Authorization)
	require.Same(t, authorizedReferenceType, rewrittenRef)

	// The type parameter binder is preserved, not cloned.
	require.Len(t, rewritten.TypeParameters, 1)
	require.Same(t, originalBinder, rewritten.TypeParameters[0])

	// The `GenericType` references in the return and parameter types still reference
	// the same binder that appears in the rewritten type's TypeParameters, so the
	// invocation checker (which keys type arguments by TypeParameters) stays consistent.
	returnGeneric, ok := rewritten.ReturnTypeAnnotation.Type.(*GenericType)
	require.True(t, ok)
	require.Same(t, rewritten.TypeParameters[0], returnGeneric.TypeParameter)

	parameterGeneric, ok := rewritten.Parameters[0].TypeAnnotation.Type.(*GenericType)
	require.True(t, ok)
	require.Same(t, rewritten.TypeParameters[0], parameterGeneric.TypeParameter)

	// Resolution keyed by the rewritten type's binder succeeds.
	typeArguments := &TypeParameterTypeOrderedMap{}
	typeArguments.Set(rewritten.TypeParameters[0], IntType)

	require.Same(t, IntType, rewritten.ReturnTypeAnnotation.Type.Resolve(typeArguments))
	require.Same(t, IntType, rewritten.Parameters[0].TypeAnnotation.Type.Resolve(typeArguments))

	// The `TypeArgumentsCheck` callback, invoked with a map keyed by the rewritten
	// type's binder, observes the type argument through the binder it closed over.
	require.NotNil(t, rewritten.TypeArgumentsCheck)
	rewritten.TypeArgumentsCheck(nil, typeArguments, nil, ast.EmptyRange, func(err error) {})
	require.True(t, observedSet)
	require.Same(t, IntType, observed)
}
