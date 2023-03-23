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

package checker

import (
	"fmt"
	"testing"

	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckBasicEntitlementDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		checker, err := ParseAndCheck(t, `
			entitlement E
		`)

		assert.NoError(t, err)
		entitlement := checker.Elaboration.EntitlementType("S.test.E")
		assert.Equal(t, "E", entitlement.String())
	})

	t.Run("priv access", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			priv entitlement E 
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
	})
}

func TestCheckBasicEntitlementMappingDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		checker, err := ParseAndCheck(t, `
			entitlement mapping M {}
		`)

		assert.NoError(t, err)
		entitlement := checker.Elaboration.EntitlementMapType("S.test.M")
		assert.Equal(t, "M", entitlement.String())
	})

	t.Run("with mappings", func(t *testing.T) {
		t.Parallel()
		checker, err := ParseAndCheck(t, `
			entitlement A 
			entitlement B
			entitlement C
			entitlement mapping M {
				A -> B
				B -> C
			}
		`)

		assert.NoError(t, err)
		entitlement := checker.Elaboration.EntitlementMapType("S.test.M")
		assert.Equal(t, "M", entitlement.String())
		assert.Equal(t, 2, len(entitlement.Relations))
	})

	t.Run("priv access", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			priv entitlement mapping M {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
	})
}

func TestCheckBasicEntitlementMappingNonEntitlements(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement A 
			resource B {}
			entitlement mapping M {
				A -> B
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement A 
			struct B {}
			entitlement mapping M {
				A -> B
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
	})

	t.Run("attachment", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement A 
			attachment B for AnyStruct {}
			entitlement mapping M {
				A -> B
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
	})

	t.Run("interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement A 
			resource interface B {}
			entitlement mapping M {
				A -> B
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
	})

	t.Run("contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement B
			contract A {}
			entitlement mapping M {
				A -> B
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
	})

	t.Run("event", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement B
			event A()
			entitlement mapping M {
				A -> B
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
	})

	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement B
			enum A: UInt8 {}
			entitlement mapping M {
				A -> B
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
	})

	t.Run("simple type", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement B
			entitlement mapping M {
				Int -> B
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
	})

	t.Run("other mapping", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement B
			entitlement mapping A {}
			entitlement mapping M {
				A -> B
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
	})
}

func TestCheckEntitlementDeclarationNesting(t *testing.T) {
	t.Parallel()
	t.Run("in contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract C {
				entitlement E
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("in contract interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract interface C {
				entitlement E
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("in resource", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			resource R {
				entitlement E
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in resource interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			resource interface R {
				entitlement E
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in attachment", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			attachment A for AnyStruct {
				entitlement E
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in struct", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			struct S {
				entitlement E
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in struct", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			struct interface S {
				entitlement E
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in enum", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			enum X: UInt8 {
				entitlement E
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		require.IsType(t, &sema.InvalidNonEnumCaseError{}, errs[1])
	})
}

func TestCheckEntitlementMappingDeclarationNesting(t *testing.T) {
	t.Parallel()
	t.Run("in contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract C {
				entitlement mapping M {}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("in contract interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract interface C {
				entitlement mapping M {}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("in resource", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			resource R {
				entitlement mapping M {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in resource interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			resource interface R {
				entitlement mapping M {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in attachment", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			attachment A for AnyStruct {
				entitlement mapping M {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in struct", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			struct S {
				entitlement mapping M {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in struct", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			struct interface S {
				entitlement mapping M {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in enum", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			enum X: UInt8 {
				entitlement mapping M {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		require.IsType(t, &sema.InvalidNonEnumCaseError{}, errs[1])
	})
}

func TestCheckBasicEntitlementAccess(t *testing.T) {

	t.Parallel()
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface S {
				access(E) let foo: String
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("multiple entitlements conjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement A
			entitlement B
			entitlement C
			resource interface R {
				access(A, B) let foo: String
				access(B, C) fun bar()
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("multiple entitlements disjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement A
			entitlement B
			entitlement C
			resource interface R {
				access(A | B) let foo: String
				access(B | C) fun bar()
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("valid in contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract C {
				entitlement E
				struct interface S {
					access(E) let foo: String
				}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("valid in contract interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract interface C {
				entitlement E
				struct interface S {
					access(E) let foo: String
				}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("qualified", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract C {
				entitlement E
				struct interface S {
					access(E) let foo: String
				}
			}
			resource R {
				access(C.E) fun bar() {}
			}
		`)

		assert.NoError(t, err)
	})
}

func TestCheckBasicEntitlementMappingAccess(t *testing.T) {

	t.Parallel()
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			struct interface S {
				access(M) let foo: auth(M) &String
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("non-reference field", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			struct interface S {
				access(M) let foo: String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[0])
	})

	t.Run("non-auth reference field", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			struct interface S {
				access(M) let foo: &String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[0])
	})

	t.Run("mismatched entitlement mapping", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			entitlement mapping N {}
			struct interface S {
				access(M) let foo: auth(N) &String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[0])
	})

	t.Run("mismatched entitlement mapping to set", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			entitlement N
			struct interface S {
				access(M) let foo: auth(N) &String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[0])
	})

	t.Run("function", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			struct interface S {
				access(M) fun foo() 
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[0])
	})

	t.Run("multiple mappings conjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {} 
			entitlement mapping N {}
			resource interface R {
				access(M, N) let foo: String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMultipleMappedEntitlementError{}, errs[0])
	})

	t.Run("multiple mappings conjunction with regular", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {} 
			entitlement N
			resource interface R {
				access(M, N) let foo: String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMultipleMappedEntitlementError{}, errs[0])
	})

	t.Run("multiple mappings disjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {} 
			entitlement mapping N {}
			resource interface R {
				access(M | N) let foo: String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMultipleMappedEntitlementError{}, errs[0])
	})

	t.Run("multiple mappings disjunction with regular", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement M 
			entitlement mapping N {}
			resource interface R {
				access(M | N) let foo: String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})

	t.Run("valid in contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract C {
				entitlement mapping M {} 
				struct interface S {
					access(M) let foo: auth(M) &String
				}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("valid in contract interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract interface C {
				entitlement mapping M {} 
				struct interface S {
					access(M) let foo: auth(M) &String
				}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("qualified", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract C {
				entitlement mapping M {} 
				struct interface S {
					access(M) let foo: auth(M) &String
				}
			}
			resource interface R {
				access(C.M) let bar: auth(C.M) &String
			}
		`)

		assert.NoError(t, err)
	})
}

func TestCheckInvalidEntitlementAccess(t *testing.T) {

	t.Parallel()

	t.Run("invalid variable decl", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			access(E) var x: String = ""
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

	t.Run("invalid fun decl", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			access(E) fun foo() {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

	t.Run("invalid contract field", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			contract C {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

	t.Run("invalid contract interface field", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			contract interface C {
				access(E) fun foo()
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

	t.Run("invalid event", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			resource I {
				access(E) event Foo()
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		require.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[1])
	})

	t.Run("invalid enum case", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			enum X: UInt8 {
				access(E) case red
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
	})

	t.Run("missing entitlement declaration fun", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("missing entitlement declaration field", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			struct interface S {
				access(E) let foo: String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckInvalidEntitlementMappingAccess(t *testing.T) {

	t.Parallel()

	t.Run("invalid variable decl", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			access(M) var x: String = ""
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[0])
	})

	t.Run("invalid fun decl", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			access(M) fun foo() {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[0])
	})

	t.Run("invalid contract field", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			contract C {
				access(M) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[0])
	})

	t.Run("invalid contract interface field", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			contract interface C {
				access(M) fun foo()
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[0])
	})

	t.Run("invalid event", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			resource I {
				access(M) event Foo()
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		require.IsType(t, &sema.InvalidMappedEntitlementMemberError{}, errs[1])
	})

	t.Run("invalid enum case", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			enum X: UInt8 {
				access(M) case red
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
	})

	t.Run("missing entitlement mapping declaration fun", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			resource R {
				access(M) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("missing entitlement mapping declaration field", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			struct interface S {
				access(M) let foo: String
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckNonEntitlementAccess(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			resource E {}
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})

	t.Run("resource interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			resource interface E {}
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})

	t.Run("attachment", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			attachment E for AnyStruct {}
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			struct E {}
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})

	t.Run("struct interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			resource E {}
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})

	t.Run("event", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			event E()
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})

	t.Run("contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract E {}
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})

	t.Run("contract interface", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			contract interface E {}
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})

	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			enum E: UInt8 {}
			resource R {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
	})
}

func TestCheckEntitlementInheritance(t *testing.T) {

	t.Parallel()
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				access(E) fun foo() 
			}
			struct S: I {
				access(E) fun foo() {}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("valid mapped", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			struct interface I {
				access(M) let x: auth(M) &String
			}
			struct S: I {
				access(M) let x: auth(M) &String
				init() {
					self.x = &"foo" as auth(M) &String
				}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("mismatched mapped", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping M {}
			entitlement mapping N {}
			struct interface I {
				access(M) let x: auth(M) &String
			}
			struct S: I {
				access(N) let x: auth(N) &String
				init() {
					self.x = &"foo" as auth(N) &String
				}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("pub subtyping invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				pub fun foo() 
			}
			struct S: I {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("pub(set) subtyping invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				pub(set) var x: String
			}
			struct S: I {
				access(E) var x: String
				init() {
					self.x = ""
				}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("pub supertying invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				access(E) fun foo() 
			}
			struct S: I {
				pub fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("pub(set) supertyping invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				access(E) var x: String
			}
			struct S: I {
				pub(set) var x: String
				init() {
					self.x = ""
				}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("access contract subtyping invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				access(contract) fun foo() 
			}
			struct S: I {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("access account subtyping invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				access(account) fun foo() 
			}
			struct S: I {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("access account supertying invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				access(E) fun foo() 
			}
			struct S: I {
				access(account) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("access contract supertying invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				access(E) fun foo() 
			}
			struct S: I {
				access(contract) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("priv supertying invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface I {
				access(E) fun foo() 
			}
			struct S: I {
				priv fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("invalid map subtype with regular conjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement mapping M {}
			struct interface I {
				access(E, F) var x: auth(M) &String
			}
			struct S: I {
				access(M) var x: auth(M) &String 

				init() {
					self.x = &"foo" as auth(M) &String
				}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("invalid map supertype with regular conjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement mapping M {}
			struct interface I {
				access(M) var x: auth(M) &String
			}
			struct S: I {
				access(E, F) var x: auth(M) &String 

				init() {
					self.x = &"foo" as auth(M) &String
				}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("invalid map subtype with regular disjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement mapping M {}
			struct interface I {
				access(E | F) var x: auth(M) &String
			}
			struct S: I {
				access(M) var x: auth(M) &String 

				init() {
					self.x = &"foo" as auth(M) &String
				}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("invalid map supertype with regular disjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement mapping M {}
			struct interface I {
				access(M) var x: auth(M) &String
			}
			struct S: I {
				access(E | F) var x: auth(M) &String 

				init() {
					self.x = &"foo" as auth(M) &String
				}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("expanded entitlements valid in disjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F
			struct interface I {
				access(E) fun foo() 
			}
			struct interface J {
				access(F) fun foo() 
			}
			struct S: I, J {
				access(E | F) fun foo() {}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("more expanded entitlements valid in disjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F
			entitlement G
			struct interface I {
				access(E | G) fun foo() 
			}
			struct S: I {
				access(E | F | G) fun foo() {}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("reduced entitlements valid with conjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement G
			struct interface I {
				access(E, G) fun foo() 
			}
			struct interface J {
				access(E, F) fun foo() 
			}
			struct S: I, J {
				access(E) fun foo() {}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("more reduced entitlements valid with conjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement G
			struct interface I {
				access(E, F, G) fun foo() 
			}
			struct S: I {
				access(E, F) fun foo() {}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("expanded entitlements invalid in conjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F
			struct interface I {
				access(E) fun foo() 
			}
			struct interface J {
				access(F) fun foo() 
			}
			struct S: I, J {
				access(E, F) fun foo() {}
			}
		`)

		// this conforms to neither I nor J
		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
		require.IsType(t, &sema.ConformanceError{}, errs[1])
	})

	t.Run("more expanded entitlements invalid in conjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F
			entitlement G
			struct interface I {
				access(E, F) fun foo() 
			}
			struct S: I {
				access(E, F, G) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("expanded entitlements invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F
			struct interface I {
				access(E) fun foo() 
			}
			struct interface J {
				access(F) fun foo() 
			}
			struct S: I, J {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("reduced entitlements invalid with disjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			struct interface I {
				access(E) fun foo() 
			}
			struct interface J {
				access(E | F) fun foo() 
			}
			struct S: I, J {
				access(E) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("more reduced entitlements invalid with disjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement G
			struct interface I {
				access(E | F | G) fun foo() 
			}
			struct S: I {
				access(E | G) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("overlapped entitlements invalid with disjunction", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement G
			struct interface J {
				access(E | F) fun foo() 
			}
			struct S: J {
				access(E | G) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("overlapped entitlements invalid with disjunction/conjunction subtype", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement G
			struct interface J {
				access(E | F) fun foo() 
			}
			struct S: J {
				access(E, G) fun foo() {}
			}
		`)

		// implementation is more specific because it requires both, but interface only guarantees one
		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("disjunction/conjunction subtype valid when sets are the same", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			struct interface J {
				access(E | E) fun foo() 
			}
			struct S: J {
				access(E, E) fun foo() {}
			}
		`)

		assert.NoError(t, err)
	})

	t.Run("overlapped entitlements valid with conjunction/disjunction subtype", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F 
			entitlement G
			struct interface J {
				access(E, F) fun foo() 
			}
			struct S: J {
				access(E | G) fun foo() {}
			}
		`)

		// implementation is less specific because it only requires one, but interface guarantees both
		assert.NoError(t, err)
	})

	t.Run("different entitlements invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F
			entitlement G 
			struct interface I {
				access(E) fun foo() 
			}
			struct interface J {
				access(F) fun foo() 
			}
			struct S: I, J {
				access(E | G) fun foo() {}
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})
}

func TestCheckEntitlementTypeAnnotation(t *testing.T) {

	t.Parallel()

	t.Run("invalid local annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			let x: E = ""
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		require.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("invalid param annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			pub fun foo(e: E) {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid return annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			resource interface I {
				pub fun foo(): E 
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid field annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			resource interface I {
				let e: E
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid conformance annotation", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			resource R: E {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidConformanceError{}, errs[0])
	})

	t.Run("invalid array annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			resource interface I {
				let e: [E]
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid fun annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			resource interface I {
				let e: (fun (E): Void)
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid enum conformance", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			enum X: E {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEnumRawTypeError{}, errs[0])
	})

	t.Run("invalid dict annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			resource interface I {
				let e: {E: E}
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		// key
		require.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
		// value
		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[1])
	})

	t.Run("invalid fun annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			resource interface I {
				let e: (fun (E): Void)
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("runtype type", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			let e = Type<E>()
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("type arg", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement E
			let e = authAccount.load<E>(from: /storage/foo)
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		// entitlements are not storable either
		require.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("restricted", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement E
			resource interface I {
				let e: E{E}
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidRestrictionTypeError{}, errs[0])
		require.IsType(t, &sema.InvalidRestrictedTypeError{}, errs[1])
	})

	t.Run("reference", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement E
			resource interface I {
				let e: &E
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("capability", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement E
			resource interface I {
				let e: Capability<&E>
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[1])
	})

	t.Run("optional", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement E
			resource interface I {
				let e: E?
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})
}

func TestCheckEntitlementMappingTypeAnnotation(t *testing.T) {

	t.Parallel()

	t.Run("invalid local annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			let x: E = ""
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		require.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("invalid param annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			pub fun foo(e: E) {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid return annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			resource interface I {
				pub fun foo(): E 
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid field annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			resource interface I {
				let e: E
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid conformance annotation", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			resource R: E {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidConformanceError{}, errs[0])
	})

	t.Run("invalid array annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			resource interface I {
				let e: [E]
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid fun annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			resource interface I {
				let e: (fun (E): Void)
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid enum conformance", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			enum X: E {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEnumRawTypeError{}, errs[0])
	})

	t.Run("invalid dict annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			resource interface I {
				let e: {E: E}
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		// key
		require.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
		// value
		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[1])
	})

	t.Run("invalid fun annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			resource interface I {
				let e: (fun (E): Void)
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("runtype type", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			let e = Type<E>()
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("type arg", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement mapping E {}
			let e = authAccount.load<E>(from: /storage/foo)
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		// entitlements are not storable either
		require.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("restricted", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement mapping E {}
			resource interface I {
				let e: E{E}
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidRestrictionTypeError{}, errs[0])
		require.IsType(t, &sema.InvalidRestrictedTypeError{}, errs[1])
	})

	t.Run("reference", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement mapping E {}
			resource interface I {
				let e: &E
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("capability", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement mapping E {}
			resource interface I {
				let e: Capability<&E>
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[1])
	})

	t.Run("optional", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
			entitlement mapping E {}
			resource interface I {
				let e: E?
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})
}

func TestChecAttachmentEntitlementAccessAnnotation(t *testing.T) {

	t.Parallel()
	t.Run("mapping allowed", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement mapping E {}
			access(E) attachment A for AnyStruct {}
		`)

		assert.NoError(t, err)
	})

	t.Run("entitlement set not allowed", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
			entitlement E
			entitlement F
			access(E, F) attachment A for AnyStruct {}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

	t.Run("mapping allowed in contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		contract C {
			entitlement mapping E {}
			access(E) attachment A for AnyStruct {}
		}
		`)

		assert.NoError(t, err)
	})

	t.Run("entitlement set not allowed in contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		contract C {
			entitlement E
			access(E) attachment A for AnyStruct {}
		}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

}

func TestCheckEntitlementSetAccess(t *testing.T) {

	t.Parallel()

	runTest := func(refType string, memberName string, valid bool) {
		t.Run(fmt.Sprintf("%s on %s", memberName, refType), func(t *testing.T) {
			t.Parallel()
			_, err := ParseAndCheckAccount(t, fmt.Sprintf(`
				entitlement X
				entitlement Y
				entitlement Z

				struct R {
					pub fun p() {}

					access(X) fun x() {}
					access(Y) fun y() {}
					access(Z) fun z() {}

					access(X, Y) fun xy() {}
					access(Y, Z) fun yz() {}
					access(X, Z) fun xz() {}
					
					access(X, Y, Z) fun xyz() {}

					access(X | Y) fun xyOr() {}
					access(Y | Z) fun yzOr() {}
					access(X | Z) fun xzOr() {}

					access(X | Y | Z) fun xyzOr() {}

					priv fun private() {}
					access(contract) fun c() {}
					access(account) fun a() {}
				}

				fun test(ref: %s) {
					ref.%s()
				}
			`, refType, memberName))

			if valid {
				assert.NoError(t, err)
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.InvalidAccessError{}, errs[0])
			}
		})
	}

	tests := []struct {
		refType    string
		memberName string
		valid      bool
	}{
		{"&R", "p", true},
		{"&R", "x", false},
		{"&R", "xy", false},
		{"&R", "xyz", false},
		{"&R", "xyOr", false},
		{"&R", "xyzOr", false},
		{"&R", "private", false},
		{"&R", "a", true},
		{"&R", "c", false},

		{"auth(X) &R", "p", true},
		{"auth(X) &R", "x", true},
		{"auth(X) &R", "y", false},
		{"auth(X) &R", "xy", false},
		{"auth(X) &R", "xyz", false},
		{"auth(X) &R", "xyOr", true},
		{"auth(X) &R", "xyzOr", true},
		{"auth(X) &R", "private", false},
		{"auth(X) &R", "a", true},
		{"auth(X) &R", "c", false},

		{"auth(X, Y) &R", "p", true},
		{"auth(X, Y) &R", "x", true},
		{"auth(X, Y) &R", "y", true},
		{"auth(X, Y) &R", "xy", true},
		{"auth(X, Y) &R", "xyz", false},
		{"auth(X, Y) &R", "xyOr", true},
		{"auth(X, Y) &R", "xyzOr", true},
		{"auth(X, Y) &R", "private", false},
		{"auth(X, Y) &R", "a", true},
		{"auth(X, Y) &R", "c", false},

		{"auth(X, Y, Z) &R", "p", true},
		{"auth(X, Y, Z) &R", "x", true},
		{"auth(X, Y, Z) &R", "y", true},
		{"auth(X, Y, Z) &R", "z", true},
		{"auth(X, Y, Z) &R", "xy", true},
		{"auth(X, Y, Z) &R", "xyz", true},
		{"auth(X, Y, Z) &R", "xyOr", true},
		{"auth(X, Y, Z) &R", "xyzOr", true},
		{"auth(X, Y, Z) &R", "private", false},
		{"auth(X, Y, Z) &R", "a", true},
		{"auth(X, Y, Z) &R", "c", false},

		{"auth(X | Y) &R", "p", true},
		{"auth(X | Y) &R", "x", false},
		{"auth(X | Y) &R", "y", false},
		{"auth(X | Y) &R", "xy", false},
		{"auth(X | Y) &R", "xyz", false},
		{"auth(X | Y) &R", "xyOr", true},
		{"auth(X | Y) &R", "xzOr", false},
		{"auth(X | Y) &R", "yzOr", false},
		{"auth(X | Y) &R", "xyzOr", true},
		{"auth(X | Y) &R", "private", false},
		{"auth(X | Y) &R", "a", true},
		{"auth(X | Y) &R", "c", false},

		{"auth(X | Y | Z) &R", "p", true},
		{"auth(X | Y | Z) &R", "x", false},
		{"auth(X | Y | Z) &R", "y", false},
		{"auth(X | Y | Z) &R", "xy", false},
		{"auth(X | Y | Z) &R", "xyz", false},
		{"auth(X | Y | Z) &R", "xyOr", false},
		{"auth(X | Y | Z) &R", "xzOr", false},
		{"auth(X | Y | Z) &R", "yzOr", false},
		{"auth(X | Y | Z) &R", "xyzOr", true},
		{"auth(X | Y | Z) &R", "private", false},
		{"auth(X | Y | Z) &R", "a", true},
		{"auth(X | Y | Z) &R", "c", false},

		{"R", "p", true},
		{"R", "x", true},
		{"R", "y", true},
		{"R", "xy", true},
		{"R", "xyz", true},
		{"R", "xyOr", true},
		{"R", "xzOr", true},
		{"R", "yzOr", true},
		{"R", "xyzOr", true},
		{"R", "private", false},
		{"R", "a", true},
		{"R", "c", false},
	}

	for _, test := range tests {
		runTest(test.refType, test.memberName, test.valid)
	}

}

func TestCheckEntitlementMapAccess(t *testing.T) {

	t.Parallel()
	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement mapping M {
			X -> Y
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(X) &{S}) {
			let x: auth(Y) &Int = ref.x
		}
		`)

		assert.NoError(t, err)
	})

	t.Run("do not retain entitlements", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement mapping M {
			X -> Y
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(X) &{S}) {
			let x: auth(X) &Int = ref.x
		}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		// X is not retained in the entitlements for ref
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		require.Equal(t, errs[0].(*sema.TypeMismatchError).ExpectedType.QualifiedString(), "auth(X) &Int")
		require.Equal(t, errs[0].(*sema.TypeMismatchError).ActualType.QualifiedString(), "auth(Y) &Int")
	})

	t.Run("different views", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement A
		entitlement B
		entitlement mapping M {
			X -> Y
			A -> B
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(A) &{S}) {
			let x: auth(Y) &Int = ref.x
		}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		// access gives B, not Y
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		require.Equal(t, errs[0].(*sema.TypeMismatchError).ExpectedType.QualifiedString(), "auth(Y) &Int")
		require.Equal(t, errs[0].(*sema.TypeMismatchError).ActualType.QualifiedString(), "auth(B) &Int")
	})

	t.Run("safe disjoint", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement A
		entitlement B
		entitlement mapping M {
			X -> Y
			A -> B
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(A | X) &{S}) {
			let x: auth(B | Y) &Int = ref.x
		}
		`)

		assert.NoError(t, err)
	})

	t.Run("unrepresentable disjoint", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement A
		entitlement B
		entitlement C
		entitlement mapping M {
			X -> Y
			X -> C
			A -> B
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(A | X) &{S}) {
			let x = ref.x
		}
		`)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.UnrepresentableEntitlementMapOutputError{}, errs[0])
		require.IsType(t, &sema.InvalidAccessError{}, errs[1])
	})

	t.Run("unrepresentable disjoint with dedup", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement A
		entitlement B
		entitlement mapping M {
			X -> Y
			X -> B
			A -> B
			A -> Y
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(A | X) &{S}) {
			let x = ref.x
		}
		`)

		// theoretically this should be allowed, because ((Y & B) | (Y & B)) simplifies to
		// just (Y & B), but this would require us to build in a simplifier for boolean expressions,
		// which is a lot of work for an edge case that is very unlikely to come up
		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.UnrepresentableEntitlementMapOutputError{}, errs[0])
		require.IsType(t, &sema.InvalidAccessError{}, errs[1])
	})

	t.Run("multiple output", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement Z
		entitlement mapping M {
			X -> Y
			X -> Z
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(X) &{S}) {
			let x: auth(Y, Z) &Int = ref.x
		}
		`)

		assert.NoError(t, err)
	})

	t.Run("unmapped entitlements do not pass through map", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement Z
		entitlement D
		entitlement mapping M {
			X -> Y
			X -> Z
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(D) &{S}) {
			let x1: auth(D) &Int = ref.x
		}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		// access results in pub access because D is not mapped
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		require.Equal(t, errs[0].(*sema.TypeMismatchError).ExpectedType.QualifiedString(), "auth(D) &Int")
		require.Equal(t, errs[0].(*sema.TypeMismatchError).ActualType.QualifiedString(), "&Int")
	})

	t.Run("multiple output with upcasting", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement Z
		entitlement mapping M {
			X -> Y
			X -> Z
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(X) &{S}) {
			let x: auth(Z) &Int = ref.x
		}
		`)

		assert.NoError(t, err)
	})

	t.Run("multiple inputs", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement A
		entitlement B 
		entitlement C
		entitlement X
		entitlement Y
		entitlement Z
		entitlement mapping M {
			A -> C
			B -> C
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref1: auth(A) &{S}, ref2: auth(B) &{S}) {
			let x1: auth(C) &Int = ref1.x
			let x2: auth(C) &Int = ref2.x
		}
		`)

		assert.NoError(t, err)
	})

	t.Run("multiple inputs and outputs", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement A
		entitlement B 
		entitlement C
		entitlement X
		entitlement Y
		entitlement Z
		entitlement mapping M {
			A -> B
			A -> C
			X -> Y
			X -> Z
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(A, X) &{S}) {
			let x: auth(B, C, Y, Z) &Int = ref.x
			let upRef = ref as auth(A) &{S}
			let upX: auth(B, C) &Int = upRef.x
		}
		`)

		assert.NoError(t, err)
	})

	t.Run("multiple inputs and outputs mismatch", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement A
		entitlement B 
		entitlement C
		entitlement X
		entitlement Y
		entitlement Z
		entitlement mapping M {
			A -> B
			A -> C
			X -> Y
			X -> Z
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: auth(A, X) &{S}) {
			let upRef = ref as auth(A) &{S}
			let upX: auth(X, Y) &Int = upRef.x
		}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		// access gives B & C, not X & Y
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		require.Equal(t, errs[0].(*sema.TypeMismatchError).ExpectedType.QualifiedString(), "auth(X, Y) &Int")
		require.Equal(t, errs[0].(*sema.TypeMismatchError).ActualType.QualifiedString(), "auth(B, C) &Int")
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement mapping M {
			X -> Y
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: &{S}) {
			let x: &Int = ref.x
		}
		`)

		assert.NoError(t, err)
	})

	t.Run("unauthorized downcast", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement mapping M {
			X -> Y
		}
		struct interface S {
			access(M) let x: auth(M) &Int
		}
		fun foo(ref: &{S}) {
			let x: auth(Y) &Int = ref.x
		}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		// result is not authorized
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("basic with init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement Z
		entitlement mapping M {
			X -> Y
			X -> Z
		}
		struct S {
			access(M) let x: auth(M) &Int
			init() {
				self.x = &1 as auth(Y, Z) &Int
			}
		}
		let ref = &S() as auth(X) &S
		let x = ref.x
		`)

		assert.NoError(t, err)
	})

	t.Run("basic with unauthorized init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement mapping M {
			X -> Y
		}
		struct S {
			access(M) let x: auth(M) &Int
			init() {
				self.x = &1 as &Int
			}
		}
		let ref = &S() as auth(X) &S
		let x = ref.x
		`)

		errs := RequireCheckerErrors(t, err, 1)

		// init of map needs full authorization of codomain
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("basic with underauthorized init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement Z
		entitlement mapping M {
			X -> Y
			X -> Z
		}
		struct S {
			access(M) let x: auth(M) &Int
			init() {
				self.x = &1 as auth(Y) &Int
			}
		}
		let ref = &S() as auth(X) &S
		let x = ref.x
		`)

		errs := RequireCheckerErrors(t, err, 1)

		// init of map needs full authorization of codomain
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("basic with underauthorized disjunction init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement Z
		entitlement mapping M {
			X -> Y
			X -> Z
		}
		struct S {
			access(M) let x: auth(M) &Int
			init() {
				self.x = &1 as auth(Y | Z) &Int
			}
		}
		let ref = &S() as auth(X) &S
		let x = ref.x
		`)

		errs := RequireCheckerErrors(t, err, 1)

		// init of map needs full authorization of codomain
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("basic with non-reference init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		entitlement X
		entitlement Y 
		entitlement Z
		entitlement mapping M {
			X -> Y
			X -> Z
		}
		struct S {
			access(M) let x: auth(M) &Int
			init() {
				self.x = 1
			}
		}
		let ref = &S() as auth(X) &S
		let x = ref.x
		`)

		errs := RequireCheckerErrors(t, err, 1)

		// init of map needs full authorization of codomain
		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}
