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

package sema_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestElaborationDeclarationForType(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        struct S {}

        struct interface I {}

        entitlement E

        entitlement mapping M {}
    `)
	require.NoError(t, err)

	elaboration := checker.Elaboration

	compositeType := RequireGlobalType(t, elaboration, "S").(*sema.CompositeType)
	interfaceType := RequireGlobalType(t, elaboration, "I").(*sema.InterfaceType)
	entitlementType := elaboration.EntitlementType("S.test.E")
	require.NotNil(t, entitlementType)
	entitlementMapType := elaboration.EntitlementMapType("S.test.M")
	require.NotNil(t, entitlementMapType)

	t.Run("composite type", func(t *testing.T) {
		t.Parallel()

		expected := checker.Program.CompositeDeclarations()[0]
		require.Same(t, expected, elaboration.DeclarationForType(compositeType))
	})

	t.Run("interface type", func(t *testing.T) {
		t.Parallel()

		expected := checker.Program.InterfaceDeclarations()[0]
		require.Same(t, expected, elaboration.DeclarationForType(interfaceType))
	})

	t.Run("entitlement type", func(t *testing.T) {
		t.Parallel()

		expected := checker.Program.EntitlementDeclarations()[0]
		require.Same(t, expected, elaboration.DeclarationForType(entitlementType))
	})

	t.Run("entitlement map type", func(t *testing.T) {
		t.Parallel()

		expected := checker.Program.EntitlementMappingDeclarations()[0]
		require.Same(t, expected, elaboration.DeclarationForType(entitlementMapType))
	})

	t.Run("unhandled type kind", func(t *testing.T) {
		t.Parallel()

		// Anything outside the four handled kinds should fall through the switch
		// and return an untyped nil ast.Declaration.
		result := elaboration.DeclarationForType(sema.IntType)
		require.True(t, result == nil)
	})

	// An empty elaboration has no declaration recorded for these types.
	// The helpers return a typed-nil value, but DeclarationForType must
	// return an untyped nil ast.Declaration so callers can compare against nil.
	emptyElaboration := sema.NewElaboration(nil)

	t.Run("composite type not in elaboration", func(t *testing.T) {
		t.Parallel()

		result := emptyElaboration.DeclarationForType(compositeType)
		require.True(t, result == nil)
	})

	t.Run("interface type not in elaboration", func(t *testing.T) {
		t.Parallel()

		result := emptyElaboration.DeclarationForType(interfaceType)
		require.True(t, result == nil)
	})

	t.Run("entitlement type not in elaboration", func(t *testing.T) {
		t.Parallel()

		result := emptyElaboration.DeclarationForType(entitlementType)
		require.True(t, result == nil)
	})

	t.Run("entitlement map type not in elaboration", func(t *testing.T) {
		t.Parallel()

		result := emptyElaboration.DeclarationForType(entitlementMapType)
		require.True(t, result == nil)
	})
}
