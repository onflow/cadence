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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
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

	t.Run("access(self) access", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            access(self) entitlement E
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
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

	t.Run("access(self) access", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            access(self) entitlement mapping M {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[1])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[1])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[1])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[1])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[1])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[1])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[1])
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

		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in resource interface", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface R {
                entitlement E
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in attachment", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            attachment A for AnyStruct {
                entitlement E
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in struct", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {
                entitlement E
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in struct", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface S {
                entitlement E
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in enum", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            enum X: UInt8 {
                entitlement E
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEnumCaseError{}, errs[1])
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

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in resource interface", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface R {
                entitlement mapping M {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in attachment", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            attachment A for AnyStruct {
                entitlement mapping M {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in struct", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {
                entitlement mapping M {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in struct", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface S {
                entitlement mapping M {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("in enum", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            enum X: UInt8 {
                entitlement mapping M {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEnumCaseError{}, errs[1])
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

func TestCheckInvalidEntitlementAccess(t *testing.T) {

	t.Parallel()

	t.Run("invalid variable decl", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            access(E) var x: String = ""
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

	t.Run("invalid fun decl", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            access(E) fun foo() {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		assert.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[1])
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

		assert.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
	})

	t.Run("missing entitlement declaration fun", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {
                access(E) fun foo() {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("missing entitlement declaration field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface S {
                access(E) let foo: String
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckInvalidEntitlementMappingAuth(t *testing.T) {
	t.Parallel()

	t.Run("invalid variable annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            let x: auth(mapping M) &Int = 3
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("invalid param annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            fun foo(x: auth(mapping M) &Int) {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("invalid return annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            fun foo(): auth(mapping M) &Int {}
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.MissingReturnStatementError{}, errs[1])
	})

	t.Run("invalid ref expr annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            let x = &1 as auth(mapping M) &Int
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("invalid failable annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            let x = &1 as &Int
            let y = x as? auth(mapping M) &Int
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("invalid type param annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            fun foo(x: Capability<auth(mapping M) &Int>) {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("invalid type argument annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            fun test(storage: auth(Storage) &Account.Storage) {
                let x = storage.borrow<auth(mapping M) &Int>(from: /storage/foo)
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("invalid cast annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            let x = &1 as &Int
            let y = x as auth(mapping M) &Int
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("capability field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            resource interface R {
                access(mapping M) let foo: Capability<auth(mapping M) &Int>
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})

	t.Run("optional ref field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            resource interface R {
                access(mapping M) let foo: (auth(mapping M) &Int)?
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})

	t.Run("fun ref field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            resource interface R {
                access(mapping M) let foo: fun(auth(mapping M) &Int): auth(mapping M) &Int
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[2])
	})

	t.Run("optional fun ref field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            resource interface R {
                access(mapping M) let foo: fun((auth(mapping M) &Int?))
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})

	t.Run("mapped ref unmapped field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F

            entitlement mapping M {
                E -> F
            }

            struct interface S {
                access(E) var x: auth(mapping M) &String
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("mapped nonref unmapped field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F

            entitlement mapping M {
                E -> F
            }

            struct interface S {
                access(E) var x: fun(auth(mapping M) &String): Int
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("mapped field unmapped ref", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F

            entitlement mapping M {
                E -> F
            }

            struct interface S {
                access(mapping M) var x: auth(E) &String
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[0])
	})

	t.Run("different map", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F

            entitlement mapping M {
                E -> F
            }

            entitlement mapping N {
                E -> F
            }

            struct interface S {
                access(mapping M) var x: auth(mapping N) &String
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})
}

func TestCheckInvalidEntitlementMappingAccess(t *testing.T) {

	t.Parallel()

	t.Run("invalid variable decl", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            access(mapping M) var x: String = ""
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[0])
	})

	t.Run("nonreference field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            resource interface R {
                access(mapping M) let foo: Int
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[0])
	})

	t.Run("optional nonreference field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            resource interface R {
                access(mapping M) let foo: Int?
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[0])
	})

	t.Run("invalid fun decl", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            access(mapping M) fun foo() {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[0])
	})

	t.Run("invalid contract field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            contract C {
                access(mapping M) fun foo() {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[0])
	})

	t.Run("invalid contract interface field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            contract interface C {
                access(mapping M) fun foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[0])
	})

	t.Run("invalid event", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            resource I {
                access(mapping M) event Foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[1])
	})

	t.Run("invalid enum case", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping M {}

            enum X: UInt8 {
                access(mapping M) case red
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
	})

	t.Run("missing entitlement mapping declaration fun", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {
                access(mapping M) fun foo() {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("missing entitlement mapping declaration field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface S {
                access(mapping M) let foo: String
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidNonEntitlementAccessError{}, errs[0])
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

	t.Run("valid interface", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            struct interface I {
                access(E) fun foo()
            }

            struct interface S: I {
                access(E) fun foo()
            }
        `)

		assert.NoError(t, err)
	})

	t.Run("valid mapped", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X

            entitlement Y

            entitlement mapping M {
                X -> Y
            }

            struct interface I {
                access(mapping M) let x: auth(mapping M) &String
            }

            struct S: I {
                access(mapping M) let x: auth(mapping M) &String
                init() {
                    self.x = &"foo" as auth(Y) &String
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[2])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[3])
	})

	t.Run("valid mapped interface", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            entitlement mapping M {
                X -> Y
            }

            struct interface I {
                access(mapping M) let x: auth(mapping M) &String
            }

            struct interface S: I {
                access(mapping M) let x: auth(mapping M) &String
            }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[2])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[3])
	})

	t.Run("mismatched mapped", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            entitlement mapping M {}

            entitlement mapping N {
                X -> Y
            }

            struct interface I {
                access(mapping M) let x: auth(mapping M) &String
            }

            struct S: I {
                access(mapping N) let x: auth(mapping N) &String
                init() {
                    self.x = &"foo" as auth(Y) &String
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 5)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[2])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[3])
		assert.IsType(t, &sema.ConformanceError{}, errs[4])
	})

	t.Run("mismatched mapped interfaces", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            entitlement mapping M {}

            entitlement mapping N {
                X -> Y
            }

            struct interface I {
                access(mapping M) let x: auth(mapping M) &String
            }

            struct interface S: I {
                access(mapping N) let x: auth(mapping N) &String
            }
        `)

		errs := RequireCheckerErrors(t, err, 5)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[2])
		assert.IsType(t, &sema.InterfaceMemberConflictError{}, errs[3])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[4])
	})

	t.Run("access(all) subtyping invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            struct interface I {
                access(all) fun foo()
            }

            struct S: I {
                access(E) fun foo() {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("access(all) subtyping invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            struct interface I {
                access(all) var x: String
            }

            struct S: I {
                access(E) var x: String

                init() {
                    self.x = ""
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("access(all) supertying invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            struct interface I {
                access(E) fun foo()
            }

            struct S: I {
                access(all) fun foo() {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("access(all) supertyping invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            struct interface I {
                access(E) var x: String
            }

            struct S: I {
                access(all) var x: String
                init() {
                    self.x = ""
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("access(self) supertying invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            struct interface I {
                access(E) fun foo()
            }

            struct S: I {
                access(self) fun foo() {}
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("invalid map subtype with regular conjunction", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F

            entitlement mapping M {
                E -> F
            }

            struct interface I {
                access(E, F) var x: auth(E, F) &String
            }

            struct S: I {
                access(mapping M) var x: auth(mapping M) &String

                init() {
                    self.x = &"foo" as auth(F) &String
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.ConformanceError{}, errs[2])
	})

	t.Run("invalid map supertype with regular conjunction", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F

            entitlement mapping M {
                E -> F
            }

            struct interface I {
                access(mapping M) var x: auth(mapping M) &String
            }

            struct S: I {
                access(E, F) var x: auth(E, F) &String

                init() {
                    self.x = &"foo" as auth(E, F) &String
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.ConformanceError{}, errs[2])
	})

	t.Run("invalid map subtype with regular disjunction", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F

            entitlement mapping M {
                E -> F
            }

            struct interface I {
                access(E | F) var x: auth(E | F) &String
            }

            struct S: I {
                access(mapping M) var x: auth(mapping M) &String

                init() {
                    self.x = &"foo" as auth(F) &String
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.ConformanceError{}, errs[2])
	})

	t.Run("invalid map supertype with regular disjunction", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F

            entitlement mapping M {
                E -> F
            }

            struct interface I {
                access(mapping M) var x: auth(mapping M) &String
            }

            struct S: I {
                access(E | F) var x: auth(E | F) &String

                init() {
                    self.x = &"foo" as auth(F) &String
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.ConformanceError{}, errs[2])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
		assert.IsType(t, &sema.ConformanceError{}, errs[1])
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

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
		assert.IsType(t, &sema.ConformanceError{}, errs[1])
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

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
		assert.IsType(t, &sema.ConformanceError{}, errs[1])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
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

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
		assert.IsType(t, &sema.ConformanceError{}, errs[1])
	})

	t.Run("default function entitlements", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F

            entitlement mapping M {
                E -> F
            }

            entitlement G

            struct interface I {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &1 as auth(mapping M) &Int
                }
            }

            struct S: I {}

            fun test() {
                let s = S()
                let ref = &s as auth(E) &S
                let i: auth(F) &Int = s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[3])
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

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("invalid param annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            fun foo(e: E) {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid return annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            resource interface I {
                fun foo(): E
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
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

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid conformance annotation", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            resource R: E {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
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

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid enum conformance", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            enum X: E {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidEnumRawTypeError{}, errs[0])
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
		assert.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
		// value
		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[1])
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

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("runtype type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            let e = Type<E>()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("type arg", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            fun test(storage: auth(Storage) &Account.Storage) {
                let e = storage.load<E>(from: /storage/foo)
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		// entitlements are not storable either
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("intersection", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            resource interface I {
                let e: {E}
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidIntersectedTypeError{}, errs[0])
		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[1])
	})

	t.Run("reference", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            resource interface I {
                let e: &E
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("capability", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            resource interface I {
                let e: Capability<&E>
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[1])
	})

	t.Run("optional", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            resource interface I {
                let e: E?
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
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

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("invalid param annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            fun foo(e: E) {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid return annot", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            resource interface I {
                fun foo(): E
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
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

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid conformance annotation", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            resource R: E {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidConformanceError{}, errs[0])
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

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
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

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("invalid enum conformance", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            enum X: E {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidEnumRawTypeError{}, errs[0])
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
		assert.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
		// value
		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[1])
	})

	t.Run("runtime type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            let e = Type<E>()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("type arg", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            fun test(storage: auth(Storage) &Account.Storage) {
                let e = storage.load<E>(from: /storage/foo)
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		// entitlements are not storable either
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("intersection", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            resource interface I {
                let e: {E}
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidIntersectedTypeError{}, errs[0])
		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[1])
	})

	t.Run("reference", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            resource interface I {
                let e: &E
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})

	t.Run("capability", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            resource interface I {
                let e: Capability<&E>
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[1])
	})

	t.Run("optional", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}
            resource interface I {
                let e: E?
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DirectEntitlementAnnotationError{}, errs[0])
	})
}

func TestCheckAttachmentEntitlementAccessAnnotation(t *testing.T) {

	t.Parallel()

	t.Run("mapping not allowed", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping E {}

            access(mapping E) attachment A for AnyStruct {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[0])
	})

	t.Run("entitlement set not allowed", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            entitlement F

            access(E, F) attachment A for AnyStruct {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

	t.Run("mapping not allowed in contract", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
        contract C {
            entitlement X

            entitlement Y

            entitlement mapping E {
                X -> Y
            }

            access(mapping E) attachment A for AnyStruct {
                fun foo() {}
            }
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[0])
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

		assert.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

}

func TestCheckEntitlementSetAccess(t *testing.T) {

	t.Parallel()

	runTest := func(refType string, memberName string, valid bool) {
		t.Run(fmt.Sprintf("%s on %s", memberName, refType), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, fmt.Sprintf(`
                entitlement X
                entitlement Y
                entitlement Z

                struct R {
                    access(all) fun p() {}

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

                    access(self) fun private() {}
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

				assert.IsType(t, &sema.InvalidAccessError{}, errs[0])
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

          struct Q {
              access(Y) fun foo() {}
          }

          struct interface S {
              access(mapping M) let x: auth(mapping M) &Q
          }

          fun foo(s: auth(X) &{S}) {
              s.x.foo()
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[2])
	})

	t.Run("basic with optional access", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct Q {
              access(Y) fun foo() {}
          }

          struct interface S {
              access(mapping M) let x: auth(mapping M) &Q
          }

          fun foo(s: auth(X) &{S}?) {
              s?.x?.foo()
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[2])
	})

	t.Run("basic with optional access return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct Q {}

          struct interface S {
              access(mapping M) let x: auth(mapping M) &Q
          }

          fun foo(s: auth(X) &{S}?): auth(Y) &Q? {
              return s?.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

	t.Run("owned", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y
          entitlement E
          entitlement F

          entitlement mapping M {
              X -> Y
              E -> F
          }

          struct S {
              access (mapping M) let foo: auth(mapping M) &Int
              init() {
                  self.foo = &3 as auth(F, Y) &Int
              }
          }

          fun test(): &Int {
              let s: S? = S()
              let i: auth(F, Y) &Int? = s?.foo
              return i!
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

	t.Run("basic with optional partial map", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y
          entitlement E
          entitlement F

          entitlement mapping M {
              X -> Y
              E -> F
          }

          struct S {
              access(mapping M) let foo: auth(mapping M) &Int
              init() {
                  self.foo = &3 as auth(F, Y) &Int
              }
          }

          fun test(): &Int {
              let s = S()
              let ref = &s as auth(X) &S?
              let i: auth(F, Y) &Int? = ref?.foo
              return i!
          }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchError)
		assert.Equal(t,
			"S?",
			typeMismatchError.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"S",
			typeMismatchError.ActualType.QualifiedString(),
		)

		require.ErrorAs(t, errs[3], &typeMismatchError)
		assert.Equal(t,
			"auth(F, Y) &Int?",
			typeMismatchError.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&Int?",
			typeMismatchError.ActualType.QualifiedString(),
		)
	})

	t.Run("basic with optional function call return invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y
          entitlement E
          entitlement F

          entitlement mapping M {
              X -> Y
              E -> F
          }

          struct S {
              access(mapping M) fun foo(): auth(mapping M) &Int {
                  return &1 as auth(mapping M) &Int
              }
          }

          fun foo(s: auth(X) &S?): auth(X, Y) &Int? {
              return s?.foo()
          }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[3], &typeMismatchErr)
		assert.Equal(t,
			"auth(X, Y) &Int?",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&Int?",
			typeMismatchErr.ActualType.QualifiedString(),
		)
	})

	t.Run("multiple outputs", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y
          entitlement E
          entitlement F

          entitlement mapping M {
              X -> Y
              E -> F
          }

          struct interface S {
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(X | E) &{S}) {
              let x: auth(Y | F) &Int = ref.x
              let x2: auth(Y, F) &Int = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y | F) &Int",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&Int",
			typeMismatchErr.ActualType.QualifiedString(),
		)

		require.ErrorAs(t, errs[3], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y, F) &Int",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&Int",
			typeMismatchErr.ActualType.QualifiedString(),
		)
	})

	t.Run("optional", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct interface S {
              access(mapping M) let x: auth(mapping M) &Int?
          }

          fun foo(ref: auth(X) &{S}) {
              let x: auth(Y) &Int? = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(X) &{S}) {
              let x: auth(X) &Int = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		// X is not retained in the entitlements for ref
		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchErr)
		assert.Equal(t,
			"auth(X) &Int",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&Int",
			typeMismatchErr.ActualType.QualifiedString(),
		)
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(A) &{S}) {
              let x: auth(Y) &Int = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		// access gives unauthorized, not Y
		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y) &Int",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&Int",
			typeMismatchErr.ActualType.QualifiedString(),
		)
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(A | X) &{S}) {
              let x: auth(B | Y) &Int = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(A | X) &{S}) {
              let x = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.UnrepresentableEntitlementMapOutputError{}, errs[2])

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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(A | X) &{S}) {
              let x = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		// theoretically this should be allowed, because ((Y & B) | (Y & B)) simplifies to
		// just (Y & B), but this would require us to build in a simplifier for boolean expressions,
		// which is a lot of work for an edge case that is very unlikely to come up

		assert.IsType(t, &sema.UnrepresentableEntitlementMapOutputError{}, errs[2])
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(X) &{S}) {
              let x: auth(Y, Z) &Int = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(D) &{S}) {
              let x1: auth(D) &Int = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchErr)
		assert.Equal(t,
			"auth(D) &Int",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&Int",
			typeMismatchErr.ActualType.QualifiedString(),
		)
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(X) &{S}) {
              let x: auth(Z) &Int = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref1: auth(A) &{S}, ref2: auth(B) &{S}) {
              let x1: auth(C) &Int = ref1.x
              let x2: auth(C) &Int = ref2.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[3])
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(A, X) &{S}) {
              let x: auth(B, C, Y, Z) &Int = ref.x
              let upRef = ref as auth(A) &{S}
              let upX: auth(B, C) &Int = upRef.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[3])
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: auth(A, X) &{S}) {
              let upRef = ref as auth(A) &{S}
              let upX: auth(X, Y) &Int = upRef.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		// access unauthorized, not X & Y
		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchErr)
		assert.Equal(t,
			"auth(X, Y) &Int",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&Int",
			typeMismatchErr.ActualType.QualifiedString(),
		)
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: &{S}) {
              let x: &Int = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
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
              access(mapping M) let x: auth(mapping M) &Int
          }

          fun foo(ref: &{S}) {
              let x: auth(Y) &Int = ref.x
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		// result is not authorized
		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y) &Int",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&Int",
			typeMismatchErr.ActualType.QualifiedString(),
		)
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
              access(mapping M) let x: auth(mapping M) &Int
              init() {
                  self.x = &1 as auth(Y, Z) &Int
              }
          }

          let ref = &S() as auth(X) &S
          let x = ref.x
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})

	t.Run("basic with update", func(t *testing.T) {
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
              access(mapping M) var x: auth(mapping M) &Int

              init() {
                  self.x = &1 as auth(Y, Z) &Int
              }

              fun updateX(x: auth(Y, Z) &Int) {
                  self.x = x
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})

	t.Run("basic with update error", func(t *testing.T) {
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
              access(mapping M) var x: auth(mapping M) &Int

              init() {
                  self.x = &1 as auth(Y, Z) &Int
              }

              fun updateX(x: auth(Z) &Int) {
                  self.x = x
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
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
              access(mapping M) let x: auth(mapping M) &Int
              init() {
                  self.x = &1 as &Int
              }
          }

          let ref = &S() as auth(X) &S
          let x = ref.x
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
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
              access(mapping M) let x: auth(mapping M) &Int

              init() {
                  self.x = &1 as auth(Y) &Int
              }
          }

          let ref = &S() as auth(X) &S
          let x = ref.x
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
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
              access(mapping M) let x: auth(mapping M) &Int

              init() {
                  self.x = (&1 as auth(Y) &Int) as auth(Y | Z) &Int
              }
          }

          let ref = &S() as auth(X) &S
          let x = ref.x
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
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
              access(mapping M) let x: auth(mapping M) &Int

              init() {
                  self.x = &1 as &Int
              }
          }

          let ref = &S() as auth(X) &S
          let x = ref.x
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})
}

func TestCheckAttachmentEntitlements(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          struct S {
              access(Y) fun foo() {}
          }

          attachment A for S {

              access(Y) fun entitled() {
                  let a: auth(Y) &A = self
                  let b: &S = base
              }

              fun unentitled() {
                  let a: auth(X, Y) &A = self // err
                  let b: auth(X) &S = base // err
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[0], &typeMismatchErr)
		assert.Equal(t,
			"auth(X, Y) &A",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&A",
			typeMismatchErr.ActualType.QualifiedString(),
		)

		require.ErrorAs(t, errs[1], &typeMismatchErr)
		assert.Equal(t,
			"auth(X) &S",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&S",
			typeMismatchErr.ActualType.QualifiedString(),
		)
	})

	t.Run("base type with too few requirements", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          struct S {}

          attachment A for S {

              fun unentitled() {
                  let b: &S = base
              }

              fun entitled() {
                  let b: auth(X, Y) &S = base
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[0], &typeMismatchErr)
		assert.Equal(t,
			"auth(X, Y) &S",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&S",
			typeMismatchErr.ActualType.QualifiedString(),
		)
	})

	t.Run("base type with sufficient requirements", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X

          struct S {
              access(X) fun foo() {}
          }

          attachment A for S {
              fun unentitled() {
                  let b: &S = base
              }

              access(X) fun entitled() {
                  let b: auth(X) &S = base
              }
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("base type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          struct S {

              access(X) fun foo() {}

              access(Y) fun bar() {}
          }

          attachment A for S {

              fun unentitled() {
                  let b: &S = base
              }

              access(X, Y) fun entitled() {
                  let b: auth(X, Y) &S = base
              }
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("base and self in mapped functions", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct S {
              access(X) fun foo() {}
          }

          attachment A for S {
              access(mapping M) fun foo(): auth(mapping M) &Int {
                  let b: auth(mapping M) &S = base
                  let a: auth(mapping M) &A = self

                  return &1
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[3])
	})

	t.Run("invalid base and self in mapped functions", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct S {
              access(X) fun foo() {}
          }

          attachment A for S {
              access(mapping M) fun foo(): auth(mapping M) &Int {
                  let b: auth(Y) &S = base
                  let a: auth(Y) &A = self

                  return &1
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[1])

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y) &S",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"auth(mapping M) &S",
			typeMismatchErr.ActualType.QualifiedString(),
		)

		require.ErrorAs(t, errs[3], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y) &A",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"auth(mapping M) &A",
			typeMismatchErr.ActualType.QualifiedString(),
		)
	})

	t.Run("missing in S", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y
          entitlement Z
          entitlement E

          struct S {
              access(X) fun foo() {}
              access(Y | Z) let bar: Int
              init() {
                  self.bar = 1
              }
          }

          attachment A for S {
              access(E) fun entitled() {}
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var invalidAttachmentEntitlementErr *sema.InvalidAttachmentEntitlementError
		require.ErrorAs(t, errs[0], &invalidAttachmentEntitlementErr)
		assert.Equal(t,
			"E",
			invalidAttachmentEntitlementErr.InvalidEntitlement.QualifiedString(),
		)
	})

	t.Run("missing in set", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement Y
          entitlement Z
          entitlement E

          struct S {
              access(Y, Z) fun foo() {}
          }

          attachment A for S {
              access(Y | E | Z) fun entitled() {}
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var invalidAttachmentEntitlementErr *sema.InvalidAttachmentEntitlementError
		require.ErrorAs(t, errs[0], &invalidAttachmentEntitlementErr)
		assert.Equal(t,
			"E",
			invalidAttachmentEntitlementErr.InvalidEntitlement.QualifiedString(),
		)
	})

	t.Run("multiple missing in codomain", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement E
          entitlement F

          struct S {
              access(F) fun foo() {}
          }

          attachment A for S {
              access(F, X, E) fun entitled() {}
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		var invalidAttachmentEntitlementErr *sema.InvalidAttachmentEntitlementError
		require.ErrorAs(t, errs[0], &invalidAttachmentEntitlementErr)
		assert.Equal(t,
			"X",
			invalidAttachmentEntitlementErr.InvalidEntitlement.QualifiedString(),
		)

		require.ErrorAs(t, errs[1], &invalidAttachmentEntitlementErr)
		assert.Equal(t,
			"E",
			invalidAttachmentEntitlementErr.InvalidEntitlement.QualifiedString(),
		)
	})

	t.Run("mapped field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct S {
              access(X) fun foo() {}
          }

          attachment A for S {
              access(mapping M) let x: auth(mapping M) &S

              init() {
                  self.x = &S() as auth(Y) &S
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[1])
	})

	t.Run("access(all) decl", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          struct S {}

          attachment A for S {

              access(Y) fun entitled() {}

              access(Y) let entitledField: Int

              access(all) fun unentitled() {
                  let a: auth(Y) &A = self // err
                  let b: auth(X) &S = base // err
              }

              init() {
                  self.entitledField = 3
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[0], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y) &A",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&A",
			typeMismatchErr.ActualType.QualifiedString(),
		)

		require.ErrorAs(t, errs[1], &typeMismatchErr)
		require.Equal(t,
			"auth(X) &S",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		require.Equal(t,
			"&S",
			typeMismatchErr.ActualType.QualifiedString(),
		)

		assert.IsType(t, &sema.InvalidAttachmentEntitlementError{}, errs[2])
		assert.IsType(t, &sema.InvalidAttachmentEntitlementError{}, errs[3])
	})

	t.Run("non mapped entitlement decl", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          struct S {}

          access(X) attachment A for S {}
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidEntitlementAccessError{}, errs[0])
	})

}

func TestCheckAttachmentAccessEntitlements(t *testing.T) {

	t.Parallel()

	t.Run("basic owned fully entitled", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y
          entitlement Z

          struct S {
              access(Y, Z) fun foo() {}
          }

          attachment A for S {
              access(Y, Z) fun foo() {}
          }

          let s = attach A() to S()
          let a: auth(Y, Z) &A = s[A]!
        `)

		assert.NoError(t, err)
	})

	t.Run("basic owned fully entitled missing X", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y
          entitlement Z

          struct S {
              access(Y, Z) fun foo() {}
          }

          attachment A for S {
              access(Y, Z) fun foo() {}
          }

          let s = attach A() to S()
          let a: auth(X, Y, Z) &A = s[A]!
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("basic owned intersection fully entitled", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y
          entitlement Z

          struct interface I {
              access(Y, Z) fun foo()
          }

          struct S: I {
              access(Y, Z) fun foo() {}
          }

          attachment A for I {
              access(Y, Z) fun foo() {}
          }

          let s: {I} = attach A() to S()
          let a: auth(Y, Z) &A = s[A]!
        `)

		assert.NoError(t, err)
	})

	t.Run("basic owned intersection fully entitled missing X", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y
          entitlement Z

          struct interface I {
              access(Y, Z) fun foo()
          }

          struct S: I {
              access(Y, Z) fun foo() {}
          }

          attachment A for I {
              access(Y, Z) fun foo() {}
          }

          let s: {I} = attach A() to S()
          let a: auth(X, Y, Z) &A = s[A]!
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("basic reference mapping", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement E

          struct S {
              access(X, E) fun foo() {}
          }

          attachment A for S {
              access(X, E) fun foo() {}
          }

          let s = attach A() to S()
          let xRef = &s as auth(X) &S
          let eRef = &s as auth(E) &S
          let bothRef = &s as auth(X, E) &S
          let a1: auth(X) &A = xRef[A]!
          let a2: auth(E) &A = eRef[A]!
          let a3: auth(E) &A = xRef[A]! // err
          let a4: auth(X) &A = eRef[A]! // err
          let a5: auth(X, E) &A = bothRef[A]!
        `)

		errs := RequireCheckerErrors(t, err, 2)

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[0], &typeMismatchErr)
		assert.Equal(t,
			"auth(E) &A?",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"auth(X) &A?",
			typeMismatchErr.ActualType.QualifiedString(),
		)

		require.ErrorAs(t, errs[1], &typeMismatchErr)
		assert.Equal(t,
			"auth(X) &A?",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"auth(E) &A?",
			typeMismatchErr.ActualType.QualifiedString(),
		)
	})

	t.Run("access entitled attachment", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement Y

          struct S {
              access(Y) fun foo() {}
          }

          attachment A for S {
              access(Y) fun foo() {}
          }

          let s = attach A() to S()
          let ref = &s as &S
          let a1: auth(Y) &A = ref[A]!
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[0], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y) &A?",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&A?",
			typeMismatchErr.ActualType.QualifiedString(),
		)
	})

	t.Run("access(all) access access(all) attachment", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X

          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct S {}

          access(all) attachment A for S {}

          let s = attach A() to S()
          let ref = &s as &S
          let a1: auth(Y) &A = ref[A]!
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[0], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y) &A?",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"&A?",
			typeMismatchErr.ActualType.QualifiedString(),
		)
	})

	t.Run("mapped function in attachment", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct S {
              access(X) fun foo() {}
          }

          attachment A for S {

              access(mapping M) fun foo(): auth(mapping M) &Int {
                  let s: auth(mapping M) &A = base[A]!
                  return &1
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
	})

	t.Run("invalid base attachment access in mapped function", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct S {
              access(X) fun foo() {}
          }

          attachment A for S {

              access(mapping M) fun foo(): auth(mapping M) &Int {
                  let s: auth(Y) &A? = base[A]
                  return &1
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessError{}, errs[1])

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchErr)
		assert.Equal(t,
			"auth(Y) &A?",
			typeMismatchErr.ExpectedType.QualifiedString(),
		)
		assert.Equal(t,
			"auth(mapping M) &A?",
			typeMismatchErr.ActualType.QualifiedString(),
		)
	})
}

func TestCheckEntitlementConditions(t *testing.T) {
	t.Parallel()

	t.Run("use of function on owned value", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X

          struct S {
              view access(X) fun foo(): Bool {
                  return true
              }
          }

          fun bar(r: S) {
              pre {
                  r.foo(): ""
              }
              post {
                  r.foo(): ""
              }
              r.foo()
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("use of function on entitled referenced value", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X

          struct S {
              view access(X) fun foo(): Bool {
                  return true
              }
          }

          fun bar(r: auth(X) &S) {
              pre {
                  r.foo(): ""
              }
              post {
                  r.foo(): ""
              }
              r.foo()
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("use of function on unentitled referenced value", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithOptions(t,
			`
              entitlement X

              struct S {
                  view access(X) fun foo(): Bool {
                      return true
                  }
              }

              fun bar(r: &S) {
                  pre {
                      r.foo(): ""
                  }
                  post {
                      r.foo(): ""
                  }
                  r.foo()
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					SuggestionsEnabled: true,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 3)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errs[0], &invalidAccessErr)
		assert.Equal(
			t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.X"),
				},
				sema.Conjunction,
			),
			invalidAccessErr.RestrictingAccess,
		)
		assert.Equal(t,
			sema.UnauthorizedAccess,
			invalidAccessErr.PossessedAccess,
		)

		assert.IsType(t, &sema.InvalidAccessError{}, errs[1])

		require.ErrorAs(t, errs[2], &invalidAccessErr)
		assert.Equal(t,
			"reference needs entitlement `X`",
			invalidAccessErr.SecondaryError(),
		)
	})

	t.Run("result value usage struct", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X

          struct S {
              view access(X) fun foo(): Bool {
                  return true
              }
          }

          fun bar(r: S): S {
              post {
                  result.foo(): ""
              }
              return r
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("result value usage reference", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithOptions(t,
			`
              entitlement X

              struct S {
                  view access(X) fun foo(): Bool {
                      return true
                  }
              }

              fun bar(r: S): &S {
                  post {
                      result.foo(): ""
                  }
                  return &r as auth(X) &S
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					SuggestionsEnabled: true,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errs[0], &invalidAccessErr)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.X"),
				},
				sema.Conjunction,
			),
			invalidAccessErr.RestrictingAccess,
		)
		assert.Equal(t,
			sema.UnauthorizedAccess,
			invalidAccessErr.PossessedAccess,
		)
		assert.Equal(t,
			"reference needs entitlement `X`",
			invalidAccessErr.SecondaryError(),
		)
	})

	t.Run("result value usage reference authorized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X

          struct S {
              view access(X) fun foo(): Bool {
                  return true
              }
          }

          fun bar(r: S): auth(X) &S {
              post {
                  result.foo(): ""
              }
              return &r as auth(X) &S
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("result value usage resource", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          resource R {
              view access(X) fun foo(): Bool {
                  return true
              }
              view access(X, Y) fun bar(): Bool {
                  return true
              }
          }

          fun bar(r: @R): @R {
              post {
                  result.foo(): ""
                  result.bar(): ""
              }
              return <-r
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("optional result value usage resource", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          resource R {
              view access(X) fun foo(): Bool {
                  return true
              }
              view access(X, Y) fun bar(): Bool {
                  return true
              }
          }

          fun bar(r: @R): @R? {
              post {
                  result?.foo()!: ""
                  result?.bar()!: ""
              }
              return <-r
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("result value inherited entitlement resource", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          resource interface I {
              access(X, Y) view fun foo(): Bool {
                  return true
              }
          }

          resource R: I {}

          fun bar(r: @R): @R {
              post {
                  result.foo(): ""
              }
              return <-r
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("result value usage unentitled resource", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              view fun foo(): Bool {
                  return true
              }
          }

          fun bar(r: @R): @R {
              post {
                  result.foo(): ""
              }
              return <-r
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("result value usage, variable-sized resource array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun foo(r: @[R]): @[R] {
                post {
                    bar(result): ""
                }
                return <-r
            }

            // 'result' variable should have all the entitlements available for arrays.
            view fun bar(_ r: auth(Mutate, Insert, Remove) &[R]): Bool {
                return true
            }
        `)

		assert.NoError(t, err)
	})

	t.Run("result value usage, constant-sized resource array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun foo(r: @[R; 5]): @[R; 5] {
                post {
                    bar(result): ""
                }
                return <-r
            }

            // 'result' variable should have all the entitlements available for arrays.
            view fun bar(_ r: auth(Mutate, Insert, Remove) &[R; 5]): Bool {
                return true
            }
        `)

		assert.NoError(t, err)
	})

	t.Run("result value usage, resource dictionary", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun foo(r: @{String:R}): @{String:R} {
                post {
                    bar(result): ""
                }
                return <-r
            }

            // 'result' variable should have all the entitlements available for dictionaries.
            view fun bar(_ r: auth(Mutate, Insert, Remove) &{String:R}): Bool {
                return true
            }
        `)

		assert.NoError(t, err)
	})

	t.Run("result value inherited interface entitlement resource", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          resource interface I {
              access(X) view fun foo(): Bool {
                  return true
              }
          }

          resource interface J: I {
              access(Y) view fun bar(): Bool {
                  return true
              }
          }

          fun bar(r: @{J}): @{J} {
              post {
                  result.foo(): ""
                  result.bar(): ""
              }
              return <-r
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("result inherited interface method", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          resource interface I {
              access(X, Y) view fun foo(): Bool
          }

          resource interface J: I {
              access(X, Y) view fun foo(): Bool
          }

          fun bar(r: @{J}): @{J} {
              post {
                  result.foo(): ""
              }
              return <-r
          }
        `)

		assert.NoError(t, err)
	})
}

func TestCheckEntitledWriteAndMutateNotAllowed(t *testing.T) {

	t.Parallel()

	t.Run("basic owned", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            struct S {
                access(E) var x: Int
                init() {
                    self.x = 1
                }
            }

            fun foo() {
                let s = S()
                s.x = 3
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[0])
	})

	t.Run("basic authorized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            struct S {
                access(E) var x: Int
                init() {
                    self.x = 1
                }
            }

            fun foo() {
                let s = S()
                let ref = &s as auth(E) &S
                ref.x = 3
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[0])
	})

	t.Run("mapped owned", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            entitlement mapping M {
                X -> Y
            }

            struct S {
                access(mapping M) var x: auth(mapping M) &Int
                init() {
                    self.x = &1 as auth(Y) &Int
                }
            }

            fun foo() {
                let s = S()
                s.x = &1 as auth(Y) &Int
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[2])
	})

	t.Run("mapped authorized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          entitlement Y

          entitlement mapping M {
              X -> Y
          }

          struct S {
              access(mapping M) var x: auth(mapping M) &Int
              init() {
                  self.x = &1 as auth(Y) &Int
              }
          }

          fun foo() {
              let s = S()
              let ref = &s as auth(X) &S
              ref.x = &1 as auth(Y) &Int
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[2])
	})

	t.Run("basic mutate", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E

            struct S {
                access(E) var x: [Int]
                init() {
                    self.x = [1]
                }
            }

            fun foo() {
                let s = S()
                s.x.append(3)
            }
        `)

		assert.NoError(t, err)
	})

	t.Run("basic authorized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
              entitlement E

              struct S {
                  access(E) var x: [Int]

                  init() {
                      self.x = [1]
                  }
              }

              fun foo() {
                  let s = S()
                  let ref = &s as auth(E) &S
                  ref.x.append(3)
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					SuggestionsEnabled: true,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errs[0], &invalidAccessErr)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					sema.InsertType,
					sema.MutateType,
				},
				sema.Disjunction,
			),
			invalidAccessErr.RestrictingAccess,
		)
		assert.Equal(t,
			sema.UnauthorizedAccess,
			invalidAccessErr.PossessedAccess,
		)
		assert.Equal(t,
			"reference needs one of entitlements `Insert` or `Mutate`",
			invalidAccessErr.SecondaryError(),
		)
	})
}

func TestCheckBuiltinEntitlements(t *testing.T) {

	t.Parallel()

	t.Run("builtin", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {
                access(Mutate) fun foo() {}
                access(Insert) fun bar() {}
                access(Remove) fun baz() {}
            }

            fun main() {
                let s = S()
                let mutableRef = &s as auth(Mutate) &S
                let insertableRef = &s as auth(Insert) &S
                let removableRef = &s as auth(Remove) &S
            }
        `)

		assert.NoError(t, err)
	})

	t.Run("redefine", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Mutate
            entitlement Insert
            entitlement Remove
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.RedeclarationError{}, errs[0])
		assert.IsType(t, &sema.RedeclarationError{}, errs[1])
		assert.IsType(t, &sema.RedeclarationError{}, errs[2])
	})

}

func TestCheckIdentityMapping(t *testing.T) {

	t.Parallel()

	t.Run("owned value", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {
                access(mapping Identity) fun foo(): auth(mapping Identity) &AnyStruct {
                    let a: AnyStruct = "hello"
                    return &a as auth(mapping Identity) &AnyStruct
                }
            }

            fun main() {
                let s = S()

                // OK
                let resultRef1: &AnyStruct = s.foo()

                // Error: Must return an unauthorized ref
                let resultRef2: auth(Mutate) &AnyStruct = s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errs[3], &typeMismatchError)

		require.IsType(t, &sema.ReferenceType{}, typeMismatchError.ActualType)
		actualReference := typeMismatchError.ActualType.(*sema.ReferenceType)

		assert.Equal(t, sema.UnauthorizedAccess, actualReference.Authorization)
	})

	t.Run("unauthorized ref", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {
                access(mapping Identity) fun foo(): auth(mapping Identity) &AnyStruct {
                    let a: AnyStruct = "hello"
                    return &a as auth(mapping Identity) &AnyStruct
                }
            }

            fun main() {
                let s = S()

                let ref = &s as &S

                // OK
                let resultRef1: &AnyStruct = ref.foo()

                // Error: Must return an unauthorized ref
                let resultRef2: auth(Mutate) &AnyStruct = ref.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[3])
	})

	t.Run("basic entitled ref", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {
                access(mapping Identity) fun foo(): auth(mapping Identity) &AnyStruct {
                    let a: AnyStruct = "hello"
                    return &a as auth(mapping Identity) &AnyStruct
                }
            }

            fun main() {
                let s = S()

                let mutableRef = &s as auth(Mutate) &S
                let ref1: auth(Mutate) &AnyStruct = mutableRef.foo()

                let insertableRef = &s as auth(Insert) &S
                let ref2: auth(Insert) &AnyStruct = insertableRef.foo()

                let removableRef = &s as auth(Remove) &S
                let ref3: auth(Remove) &AnyStruct = removableRef.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 6)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[3])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[4])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[5])
	})

	t.Run("entitlement set ref", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {
                access(mapping Identity) fun foo(): auth(mapping Identity) &AnyStruct {
                    let a: AnyStruct = "hello"
                    return &a as auth(mapping Identity) &AnyStruct
                }
            }

            fun main() {
                let s = S()

                let ref1 = &s as auth(Insert | Remove) &S
                let resultRef1: auth(Insert | Remove) &AnyStruct = ref1.foo()

                let ref2 = &s as auth(Insert, Remove) &S
                let resultRef2: auth(Insert, Remove) &AnyStruct = ref2.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 5)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[3])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[4])
	})

	t.Run("owned value, with entitlements", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement A
            entitlement B
            entitlement C

            struct X {
               access(A | B) var s: String

               init() {
                   self.s = "hello"
               }

               access(C) fun foo() {}
            }

            struct Y {

                // Reference
                access(mapping Identity) var x1: auth(mapping Identity) &X

                // Optional reference
                access(mapping Identity) var x2: auth(mapping Identity) &X?

                // Function returning a reference
                access(mapping Identity) fun getX(): auth(mapping Identity) &X {
                    let x = X()
                    return &x as auth(mapping Identity) &X
                }

                // Function returning an optional reference
                access(mapping Identity) fun getOptionalX(): auth(mapping Identity) &X? {
                    let x: X? = X()
                    return &x as auth(mapping Identity) &X?
                }

                init() {
                    let x = X()
                    self.x1 = &x
                    self.x2 = nil
                }
            }

            fun main() {
                let y = Y()

                let ref1: auth(A, B, C) &X = y.x1

                let ref2: auth(A, B, C) &X? = y.x2

                let ref3: auth(A, B, C) &X = y.getX()

                let ref4: auth(A, B, C) &X? = y.getOptionalX()
            }
        `)

		errs := RequireCheckerErrors(t, err, 14)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[3])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[4])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[5])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[6])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[7])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[8])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[9])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[10])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[11])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[12])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[13])
	})

	t.Run("owned value, with insufficient entitlements", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement A
            entitlement B
            entitlement C

            struct X {
               access(A | B) var s: String

               init() {
                   self.s = "hello"
               }

               access(C) fun foo() {}
            }

            struct Y {

                // Reference
                access(mapping Identity) var x1: auth(mapping Identity) &X

                // Optional reference
                access(mapping Identity) var x2: auth(mapping Identity) &X?

                // Function returning a reference
                access(mapping Identity) fun getX(): auth(mapping Identity) &X {
                    let x = X()
                    return &x as auth(mapping Identity) &X
                }

                // Function returning an optional reference
                access(mapping Identity) fun getOptionalX(): auth(mapping Identity) &X? {
                    let x: X? = X()
                    return &x as auth(mapping Identity) &X?
                }

                init() {
                    let x = X()
                    self.x1 = &x as auth(A, B, C) &X
                    self.x2 = nil
                }
            }

            fun main() {
                let y = Y()

                let ref1: auth(A, B, C) &X = y.x1

                let ref2: auth(A, B, C) &X? = y.x2

                let ref3: auth(A, B, C) &X = y.getX()

                let ref4: auth(A, B, C) &X? = y.getOptionalX()
            }
        `)

		errs := RequireCheckerErrors(t, err, 14)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[3])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[4])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[5])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[6])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[7])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[8])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[9])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[10])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[11])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[12])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[13])
	})

	t.Run("owned value, with entitlements, function typed field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement A
            entitlement B
            entitlement C

            struct X {
               access(A | B) var s: String

               init() {
                   self.s = "hello"
               }

               access(C) fun foo() {}
            }

            struct Y {

                access(mapping Identity) let fn: (fun (): X)

                init() {
                    self.fn = fun(): X {
                        return X()
                    }
                }
            }

            fun main() {
                let y = Y()
                let v = y.fn()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[0])
	})

	t.Run("owned value, with entitlements, function ref typed field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement A
            entitlement B
            entitlement C

            struct X {
               access(A | B) var s: String

               init() {
                   self.s = "hello"
               }

               access(C) fun foo() {}
            }

            struct Y {

                access(mapping Identity) let fn: auth(mapping Identity) &(fun (): X)?

                init() {
                    self.fn = nil
                }
            }

            fun main() {
                let y = Y()
                let v: auth(A, B, C) &(fun (): X) = y.fn
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchError)

		actualType := typeMismatchError.ActualType
		require.IsType(t, &sema.OptionalType{}, actualType)
		optionalType := actualType.(*sema.OptionalType)

		require.IsType(t, &sema.ReferenceType{}, optionalType.Type)
		referenceType := optionalType.Type.(*sema.ReferenceType)

		require.Equal(t, sema.UnauthorizedAccess, referenceType.Authorization)
	})

	t.Run("initializer", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X

            struct S {
                access(mapping Identity) let x: auth(mapping Identity) &String

                init(_ str: auth(X) &String) {
                    self.x = str
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})

	t.Run("initializer with owned value", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X

            struct S {
                access(mapping Identity) let x: [String]

                init(_ str: [String]) {
                    self.x = str // this should be possible, as the string array is owned here
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("initializer with inferred reference type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X

            struct S {
                access(mapping Identity) let x: auth(mapping Identity) &String

                init(_ str: String) {
                    self.x = &str // this should be possible, as we own the string and thus inference is able to give the &str
                    // reference the appropriate type
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})

	t.Run("initializer with included Identity", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X

            entitlement mapping M {
                include Identity
            }

            struct S {
                access(mapping M) let x: auth(mapping M) &String

                init(_ str: auth(X) &String) {
                    self.x = str
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})
}

func TestCheckMappingDefinitionWithInclude(t *testing.T) {

	t.Parallel()

	t.Run("cannot include non-maps", func(t *testing.T) {
		t.Parallel()
		tests := []string{
			"struct X {}",
			"struct interface X {}",
			"resource X {}",
			"resource interface X {}",
			"contract X {}",
			"contract interface X {}",
			"enum X: Int {}",
			"event X()",
			"entitlement X",
		}
		for _, typeDef := range tests {
			t.Run(typeDef, func(t *testing.T) {
				_, err := ParseAndCheck(t, fmt.Sprintf(`
                    %s
                    entitlement mapping M {
                        include X
                    }
                `, typeDef))

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, errs[0], &sema.InvalidEntitlementMappingInclusionError{})
			})
		}
	})

	t.Run("include identity", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement G

            entitlement mapping M {
                E -> F
                include Identity
                F -> G
            }
        `)

		require.NoError(t, err)

		assert.True(t, checker.Elaboration.EntitlementMapType("S.test.M").IncludesIdentity)
	})

	t.Run("no include identity", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement G

            entitlement mapping M {
                E -> F
                F -> G
            }
        `)

		require.NoError(t, err)
		require.False(t, checker.Elaboration.EntitlementMapType("S.test.M").IncludesIdentity)
	})

	t.Run("duplicate include", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement G

            entitlement mapping M {
                include Identity
                include Identity
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DuplicateEntitlementMappingInclusionError{}, errs[0])
	})

	t.Run("duplicate include non-identity", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping X {}

            entitlement mapping M {
                include X
                include Identity
                include X
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DuplicateEntitlementMappingInclusionError{}, errs[0])
	})

	t.Run("non duplicate across hierarchy", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping Y {
                include X
            }

            entitlement mapping X {}

            entitlement mapping M {
                include X
                include Y
            }
        `)

		require.NoError(t, err)
	})

	t.Run("simple cycle detection", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping Y {
                include Y
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CyclicEntitlementMappingError{}, errs[0])
	})

	t.Run("complex cycle detection", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement mapping Y {
                include X
            }

            entitlement mapping X {
                include Y
                include Z
            }

            entitlement mapping M {
                include X
                include Y
            }

            entitlement mapping Z {
                include Identity
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CyclicEntitlementMappingError{}, errs[0])
	})

}

func TestCheckIdentityIncludedMaps(t *testing.T) {

	t.Parallel()

	t.Run("only identity included", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement G

            entitlement mapping M {
                include Identity
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E, F) &S): auth(E, F) &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])

	})

	t.Run("only identity included error", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement G

            entitlement mapping M {
                include Identity
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E, F) &S): auth(E, F, G) &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errs[2], &typeMismatchError)
		assert.Equal(t,
			"&Int",
			typeMismatchError.ActualType.String(),
		)
	})

	t.Run("identity included with relations", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement G

            entitlement mapping M {
                include Identity
                F -> G
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E, F) &S): auth(E, F, G) &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

	t.Run("identity included disjoint", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement G

            // this is functionally equivalent to
            // entitlement mapping M {
                //    E -> E
                //    F -> F
                //    G -> G
                //    F -> G
            // }
            entitlement mapping M {
                include Identity
                F -> G
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E | F) &S): &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])

		// because the Identity map will always try to create conjunctions of the input with
		// any additional relations, it is functionally impossible to map a disjointly authorized
		// reference through any non-trivial map including the Identity

		assert.IsType(t, &sema.UnrepresentableEntitlementMapOutputError{}, errs[2])
	})
}

func TestCheckGeneralIncludedMaps(t *testing.T) {
	t.Run("basic include", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement X
            entitlement Y

            entitlement mapping M {
                include N
            }

            entitlement mapping N {
                E -> F
                X -> Y
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E, X) &S): auth(F, Y) &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

	t.Run("multiple includes", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement X
            entitlement Y

            entitlement mapping M {
                include A
                include B
            }

            entitlement mapping A {
                E -> F
            }

            entitlement mapping B {
                X -> Y
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E, X) &S): auth(F, Y) &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

	t.Run("multiple includes with overlap", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement X
            entitlement Y

            entitlement mapping A {
                E -> F
                F -> X
                X -> Y
            }

            entitlement mapping B {
                X -> Y
            }

            entitlement mapping M {
                include A
                include B
                F -> X
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E, X, F) &S): auth(F, Y, X) &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

	t.Run("multilayer include", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement X
            entitlement Y

            entitlement mapping M {
                include B
            }

            entitlement mapping B {
                include A
                X -> Y
            }

            entitlement mapping A {
                E -> F
                F -> X
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E, X, F) &S): auth(F, Y, X) &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

	t.Run("diamond include", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement X
            entitlement Y

            entitlement mapping M {
                include B
                include C
            }

            entitlement mapping C {
                include A
                X -> Y
            }

            entitlement mapping B {
                F -> X
                include A
            }

            entitlement mapping A {
                E -> F
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E, X, F) &S): auth(F, Y, X) &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

	t.Run("multilayer include identity", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            entitlement E
            entitlement F
            entitlement X
            entitlement Y

            entitlement mapping M {
                include B
            }

            entitlement mapping B {
                include A
                X -> Y
            }

            entitlement mapping A {
                include Identity
                E -> F
                F -> X
            }

            struct S {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &3
                }
            }

            fun foo(s: auth(E, X, F) &S): auth(E, F, Y, X) &Int {
                return s.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])

		assert.True(t, checker.Elaboration.EntitlementMapType("S.test.A").IncludesIdentity)
		assert.True(t, checker.Elaboration.EntitlementMapType("S.test.B").IncludesIdentity)
		assert.True(t, checker.Elaboration.EntitlementMapType("S.test.M").IncludesIdentity)
	})
}

func TestCheckEntitlementErrorReporting(t *testing.T) {
	t.Run("three or more conjunction", func(t *testing.T) {
		t.Parallel()
		checker, err := ParseAndCheckWithOptions(t, `
        entitlement X
        entitlement Y
        entitlement Z
        entitlement A
        entitlement B

        struct S {
            view access(X, Y, Z) fun foo(): Bool {
                return true
            }
        }

        fun bar(r: auth(A, B) &S) {
            r.foo()
        }
    `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					SuggestionsEnabled: true,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errs[0], &invalidAccessErr)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.X"),
					checker.Elaboration.EntitlementType("S.test.Y"),
					checker.Elaboration.EntitlementType("S.test.Z"),
				},
				sema.Conjunction,
			),
			invalidAccessErr.RestrictingAccess,
		)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.A"),
					checker.Elaboration.EntitlementType("S.test.B"),
				},
				sema.Conjunction,
			),
			invalidAccessErr.PossessedAccess,
		)
		assert.Equal(t,
			"reference needs all of entitlements `X`, `Y`, and `Z`",
			invalidAccessErr.SecondaryError(),
		)
	})

	t.Run("has one entitlement of three", func(t *testing.T) {
		t.Parallel()
		checker, err := ParseAndCheckWithOptions(t,
			`
              entitlement X
              entitlement Y
              entitlement Z
              entitlement A
              entitlement B

              struct S {
                  view access(X, Y, Z) fun foo(): Bool {
                      return true
                  }
              }

              fun bar(r: auth(A, B, Y) &S) {
                  r.foo()
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					SuggestionsEnabled: true,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errs[0], &invalidAccessErr)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.X"),
					checker.Elaboration.EntitlementType("S.test.Y"),
					checker.Elaboration.EntitlementType("S.test.Z"),
				},
				sema.Conjunction,
			),
			invalidAccessErr.RestrictingAccess,
		)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.A"),
					checker.Elaboration.EntitlementType("S.test.B"),
					checker.Elaboration.EntitlementType("S.test.Y"),
				},
				sema.Conjunction,
			),
			invalidAccessErr.PossessedAccess,
		)
		assert.Equal(t,
			"reference needs all of entitlements `X` and `Z`",
			invalidAccessErr.SecondaryError(),
		)
	})

	t.Run("has one entitlement of three", func(t *testing.T) {
		t.Parallel()
		checker, err := ParseAndCheckWithOptions(t,
			`
              entitlement X
              entitlement Y
              entitlement Z
              entitlement A
              entitlement B

              struct S {
                  view access(X | Y | Z) fun foo(): Bool {
                      return true
                  }
              }

              fun bar(r: auth(A, B) &S) {
                  r.foo()
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					SuggestionsEnabled: true,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errs[0], &invalidAccessErr)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.X"),
					checker.Elaboration.EntitlementType("S.test.Y"),
					checker.Elaboration.EntitlementType("S.test.Z"),
				},
				sema.Disjunction,
			),
			invalidAccessErr.RestrictingAccess,
		)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.A"),
					checker.Elaboration.EntitlementType("S.test.B"),
				},
				sema.Conjunction,
			),
			invalidAccessErr.PossessedAccess,
		)
		assert.Equal(t,
			"reference needs one of entitlements `X`, `Y`, or `Z`",
			invalidAccessErr.SecondaryError(),
		)
	})

	t.Run("no suggestion for disjoint possession set", func(t *testing.T) {
		t.Parallel()
		checker, err := ParseAndCheckWithOptions(t, `
              entitlement X
              entitlement Y
              entitlement Z
              entitlement A
              entitlement B

              struct S {
                  view access(X | Y | Z) fun foo(): Bool {
                      return true
                  }
              }

              fun bar(r: auth(A | B) &S) {
                  r.foo()
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					SuggestionsEnabled: true,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errs[0], &invalidAccessErr)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.X"),
					checker.Elaboration.EntitlementType("S.test.Y"),
					checker.Elaboration.EntitlementType("S.test.Z"),
				},
				sema.Disjunction,
			),
			invalidAccessErr.RestrictingAccess,
		)
		assert.Equal(t,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					checker.Elaboration.EntitlementType("S.test.A"),
					checker.Elaboration.EntitlementType("S.test.B"),
				},
				sema.Disjunction,
			),
			invalidAccessErr.PossessedAccess,
		)
		assert.Equal(t,
			"",
			invalidAccessErr.SecondaryError(),
		)
	})

	t.Run("no suggestion for self access requirement", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
                entitlement A
                entitlement B

                struct S {
                    view access(self) fun foo(): Bool {
                        return true
                    }
                }

                fun bar(r: auth(A, B) &S) {
                    r.foo()
                }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					SuggestionsEnabled: true,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errs[0], &invalidAccessErr)
		assert.Equal(t,
			sema.PrimitiveAccess(ast.AccessSelf),
			invalidAccessErr.RestrictingAccess,
		)
		assert.Equal(t,
			nil,
			invalidAccessErr.PossessedAccess,
		)
		assert.Equal(
			t,
			"",
			invalidAccessErr.SecondaryError(),
		)
	})
}

func TestCheckEntitlementOptionalChaining(t *testing.T) {

	t.Parallel()

	t.Run("optional chain function call", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X

            struct S {
                access(X) fun foo() {}
            }

            fun bar(r: &S?) {
                r?.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errs[0], &invalidAccessErr)
	})

	t.Run("optional chain field access", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            struct S {
                access(X, Y) let foo: Int
                init() {
                    self.foo = 0
                }
            }

            fun bar(r: auth(X) &S?) {
                r?.foo
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAccessError{}, errs[0])
	})

	t.Run("optional chain non reference", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            struct S {
                access(X, Y) let foo: Int
                init() {
                    self.foo = 0
                }
            }

            fun bar(r: S?) {
                r?.foo
            }
        `)

		require.NoError(t, err)
	})

	t.Run("optional chain mapping", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            entitlement mapping E {
                X -> Y
            }

            struct S {
                access(mapping E) let foo: auth(mapping E) &Int
                init() {
                    self.foo = &0 as auth(Y) &Int
                }
            }

            fun bar(r: (auth(X) &S)?): (auth(Y) &Int)? {
                return r?.foo
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})
}

func TestCheckEntitlementMissingInMap(t *testing.T) {

	t.Parallel()

	t.Run("missing type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X

            entitlement mapping M {
                X -> X
                NonExistingEntitlement -> X
            }

            struct S {
                access(mapping M) var foo: auth(mapping M) &Int

                init() {
                    self.foo = &3 as auth(X) &Int
                    var selfRef = &self as auth(X) &S
                    selfRef.foo
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[3])
	})

	t.Run("non entitlement type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X

          entitlement mapping M {
              X -> X
              Int -> X
          }

          struct S {
              access(mapping M) var foo: auth(mapping M) &Int

              init() {
                  self.foo = &3 as auth(X) &Int
                  var selfRef = &self as auth(X) &S
                  selfRef.foo
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidNonEntitlementTypeInMapError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[2])
	})
}

func TestCheckEntitlementMappingEscalation(t *testing.T) {

	t.Parallel()

	t.Run("escalate", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            entitlement mapping M {
                X -> Insert
                Y -> Remove
            }

            struct S {
                access(mapping M) var member: auth(mapping M) &[Int]?

                init() {
                    self.member = nil
                }

                fun grantRemovePrivileges(param: auth(Insert) &[Int]) {
                    var selfRef = &self as auth(X) &S
                    selfRef.member = param
                }
            }

            fun main() {
                var arr: [Int] = [123]
                var arrRef = &arr as auth(Insert) &[Int]
                let s = S()
                s.grantRemovePrivileges(param: arrRef)
                s.member?.removeLast()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[2])
	})

	t.Run("escalate 2", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y
            entitlement Z

            entitlement mapping M {
                X -> Z
                X -> Y
            }

            struct Attacker {
                access(mapping Identity) var s: auth(mapping Identity) &AnyStruct

                init(_ a: auth(Z) &AnyStruct) {
                    self.s = a
                }
            }

            struct Nested {
                access(X | Y) fun foo() {}
                access(X | Z) fun bar() {}
            }

            struct Attackee {
                access(mapping M) let nested: Nested
                init() {
                    self.nested = Nested()
                }
            }

            fun main() {
                let attackee = Attackee()
                let attackeeRef = &attackee as auth(Z) &Attackee
                let attacker = Attacker(attackeeRef)
                let attackerRef = &attacker as auth(X) &Attacker
                let exploit: auth(X) &Attackee = attackerRef.s as! auth(X) &Attackee
                exploit.nested.foo()
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})

	t.Run("field assign", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            entitlement mapping M {
                X -> Insert
                Y -> Remove
            }

            struct S {
                access(mapping M) var member: auth(mapping M) &[Int]?

                init() {
                    self.member = nil
                }

                fun grantRemovePrivileges(sRef: auth(X) &S, param: auth(Insert) &[Int]) {
                    sRef.member = param
                }
            }

            fun main() {
                var arr: [Int] = [123]
                var arrRef = &arr as auth(Insert) &[Int]
                let s = S()
                s.grantRemovePrivileges(sRef: &s as auth(X) &S, param: arrRef)
                s.member?.removeLast()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[2])
	})

	t.Run("double nesting", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement X
            entitlement Y

            entitlement mapping M {
                X -> Y
            }

            struct S {
                access(Y) var i: Int

                init() {
                    self.i = 11
                }

                fun bar(_ t: &T) {
                    // the t.s here should produce an unauthorized s reference which cannot modify i
                    // i.e. this should be the same as
                    //     let ref = t.s
                    //     ref. i =2
                    t.s.i = 2
                }
            }

            struct T {
                access(mapping M) var s: auth(mapping M) &S

                init() {
                    self.s = &S()
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[2])
	})

	t.Run("member expression in indexing assignment", func(t *testing.T) {

		t.Parallel()
		_, err := ParseAndCheck(t, `

            entitlement X

            entitlement mapping M {
                X -> Insert
            }

            struct S {
                access(mapping M) var arrayRefMember: auth(mapping M) &[Int]

                init() {
                    self.arrayRefMember = &[123]
                }
            }

            fun main() {
                var unauthedStructRef = &S() as &S
                unauthedStructRef.arrayRefMember[0] = 456
            }
    `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.UnauthorizedReferenceAssignmentError{}, errs[2])
	})

	t.Run("function call in indexer", func(t *testing.T) {

		t.Parallel()
		_, err := ParseAndCheck(t, `

            entitlement X
            entitlement Y

            entitlement mapping M {
                X -> Y
            }

            struct A {
                access(mapping M) fun foo(): auth(mapping M) &Int {
                    return &1
                }
            }

            fun bar(_ ref: auth(Y) &Int): Int {
                return *ref
            }

            fun main() {
                var x: [Int] = []
                let a = &A() as &A
                x[bar(a.foo())] = 456
            }
    `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

	t.Run("member expression in indexer", func(t *testing.T) {

		t.Parallel()
		_, err := ParseAndCheck(t, `

            entitlement X
            entitlement Y

            entitlement mapping M {
                X -> Y
            }

            struct A {
                access(mapping M) let b: B
                init() {
                    self.b = B()
                }
            }

            struct B {}

            fun bar(_ b: auth(Y) &B): Int {
                return 1
            }

            fun main() {
                var x: [Int] = []
                let a = &A() as &A
                x[bar(a.b)] = 456
            }
    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("capture function", func(t *testing.T) {

		t.Parallel()
		_, err := ParseAndCheck(t, `

            entitlement X

            entitlement mapping M {
                X -> Insert
            }

            struct S {
                access(mapping M) var arrayRefMember: auth(mapping M) &[Int]

                init() {
                    self.arrayRefMember = &[123]
                }
            }

            fun captureFunction(_ f: fun(): Int): Int {
                f()
                return 1
            }

            fun main() {
                var unauthedStructRef = &S() as &S
                var dict: {Int: Int} = {}
                dict[captureFunction(unauthedStructRef.arrayRefMember.removeLast)] = 1
            }
    `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[2])
	})

	t.Run("capture reference", func(t *testing.T) {

		t.Parallel()
		_, err := ParseAndCheck(t, `

                entitlement X

                entitlement mapping M {
                    X -> Insert
                }

                struct S {
                    access(mapping M) var arrayRefMember: auth(mapping M) &[Int]
                    init() {
                        self.arrayRefMember = &[123]
                    }
                }

                fun captureReference( _ ref: auth(Insert, Remove) &[Int]): Int {
                    ref.removeLast()
                    return 1
                }

                fun main() {
                    var unauthedStructRef = &S() as &S
                    var dict: {Int: Int} = {}
                    dict[captureReference(unauthedStructRef.arrayRefMember)] = 1
                }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})

}

func TestCheckEntitlementMappingComplexFields(t *testing.T) {

	t.Parallel()

	t.Run("array mapped field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Inner1
            entitlement Inner2
            entitlement Outer1
            entitlement Outer2

            entitlement mapping MyMap {
                Outer1 -> Inner1
                Outer2 -> Inner2
            }

            struct InnerObj {
                access(Inner1) fun first(): Int { return 9999 }
                access(Inner2) fun second(): Int { return 8888 }
            }

            struct Carrier {
                access(mapping MyMap) let arr: [&InnerObj]

                init() {
                    self.arr = [&InnerObj()]
                }
            }

            fun foo() {
                let x: auth(Inner1, Inner2) &InnerObj = Carrier().arr[0]
                x.first()
                x.second()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("array mapped field via reference", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Inner1
            entitlement Inner2
            entitlement Outer1
            entitlement Outer2

            entitlement mapping MyMap {
                Outer1 -> Inner1
                Outer2 -> Inner2
            }

            struct InnerObj {
                access(Inner1) fun first(): Int { return 9999 }
                access(Inner2) fun second(): Int { return 8888 }
            }

            struct Carrier {
                access(mapping MyMap) let arr: [auth(mapping MyMap) &InnerObj]

                init() {
                    self.arr = [&InnerObj()]
                }
            }

            fun foo() {
                let ref = &Carrier() as auth(Outer1) &Carrier
                let x = ref.arr[0]
                x.first()
                x.second()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[1])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[2])
	})

	t.Run("array mapped function", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Inner1
            entitlement Inner2
            entitlement Outer1
            entitlement Outer2

            entitlement mapping MyMap {
                Outer1 -> Inner1
                Outer2 -> Inner2
            }

            struct InnerObj {
                access(Inner1) fun first(): Int { return 9999 }
                access(Inner2) fun second(): Int { return 8888 }
            }

            struct Carrier {
                access(mapping MyMap) fun getArr(): [auth(mapping MyMap) &InnerObj] {
                    return [&InnerObj()]
                }
            }

            fun foo() {
                let ref = &Carrier() as auth(Outer1) &Carrier
                let x = ref.getArr()[0]
                x.first()
                x.second()
            }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[2])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[3])
	})

	t.Run("array mapped field escape", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Inner1
            entitlement Inner2
            entitlement Outer1
            entitlement Outer2

            entitlement mapping MyMap {
                Outer1 -> Inner1
                Outer2 -> Inner2
            }
            struct InnerObj {
                access(Inner1) fun first(): Int { return 9999 }
                access(Inner2) fun second(): Int { return 8888 }
            }

            struct Carrier {
                access(mapping MyMap) let arr: [auth(mapping MyMap) &InnerObj]

                init() {
                    self.arr = [&InnerObj()]
                }
            }

            struct TranslatorStruct {
                access(self) var carrier: &Carrier

                access(mapping MyMap) fun translate(): auth(mapping MyMap) &InnerObj {
                    return self.carrier.arr[0] // no type mismatch, return type is effectively unauthorized
                }

                init(_ carrier: &Carrier) {
                    self.carrier = carrier
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[2])
	})

	t.Run("dictionary mapped field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Inner1
            entitlement Inner2
            entitlement Outer1
            entitlement Outer2

            entitlement mapping MyMap {
                Outer1 -> Inner1
                Outer2 -> Inner2
            }

            struct InnerObj {
                access(Inner1) fun first(): Int { return 9999 }
                access(Inner2) fun second(): Int { return 8888 }
            }

            struct Carrier {
                access(mapping MyMap) let dict: {String: auth(mapping MyMap) &InnerObj}

                init() {
                    self.dict = {"": &InnerObj()}
                }
            }

            fun foo() {
                let x: auth(Inner1, Inner2) &InnerObj = Carrier().dict[""]!
                x.first()
                x.second()
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("dictionary mapped field via reference", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Inner1
            entitlement Inner2
            entitlement Outer1
            entitlement Outer2

            entitlement mapping MyMap {
                Outer1 -> Inner1
                Outer2 -> Inner2
            }

            struct InnerObj {
                access(Inner1) fun first(): Int { return 9999 }
                access(Inner2) fun second(): Int { return 8888 }
            }

            struct Carrier {
                access(mapping MyMap) let dict: {String: auth(mapping MyMap) &InnerObj}

                init() {
                    self.dict = {"": &InnerObj()}
                }
            }

            fun foo() {
                let ref = &Carrier() as auth(Outer1) &Carrier
                let x = ref.dict[""]!
                x.first()
                x.second()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[1])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[2])
	})

	t.Run("array mapped function", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Inner1
            entitlement Inner2
            entitlement Outer1
            entitlement Outer2

            entitlement mapping MyMap {
                Outer1 -> Inner1
                Outer2 -> Inner2
            }
            struct InnerObj {
                access(Inner1) fun first(): Int { return 9999 }
                access(Inner2) fun second(): Int { return 8888 }
            }

            struct Carrier {
                access(mapping MyMap) fun getDict(): {String: auth(mapping MyMap) &InnerObj} {
                    return {"": &InnerObj()}
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
	})

	t.Run("lambda mapped array field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Inner1
            entitlement Inner2
            entitlement Outer1
            entitlement Outer2

            entitlement mapping MyMap {
                Outer1 -> Inner1
                Outer2 -> Inner2
            }

            struct InnerObj {
                access(Inner1) fun first(): Int { return 9999 }
                access(Inner2) fun second(): Int { return 8888 }
            }

            struct Carrier {
                access(mapping MyMap) let fnArr: [fun(auth(mapping MyMap) &InnerObj): auth(mapping MyMap) &InnerObj]

                init() {
                    let innerObj = &InnerObj() as auth(Inner1, Inner2) &InnerObj
                    self.fnArr = [
                        fun(_ x: &InnerObj): auth(Inner1, Inner2) &InnerObj {
                            return innerObj
                        }
                    ]
                }
            }

            fun foo() {
                let x = (&Carrier() as auth(Outer1) &Carrier).fnArr[0]

                x(&InnerObj() as auth(Inner1) &InnerObj).first()

                x(&InnerObj() as auth(Inner2) &InnerObj).first()

                x(&InnerObj() as auth(Inner1) &InnerObj).second()
            }

        `)

		errs := RequireCheckerErrors(t, err, 5)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[1])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[2])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[3])
		assert.IsType(t, &sema.InvalidAccessError{}, errs[4])
	})

	t.Run("lambda escape", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            entitlement Inner1
            entitlement Inner2
            entitlement Outer1
            entitlement Outer2

            entitlement mapping MyMap {
                Outer1 -> Inner1
                Outer2 -> Inner2
            }

            struct InnerObj {
                access(Inner1) fun first(): Int { return 9999 }
                access(Inner2) fun second(): Int { return 8888 }
            }

            struct FuncGenerator {
                access(mapping MyMap) fun generate(): auth(mapping MyMap) &Int? {
                    // cannot declare lambda with mapped entitlement
                    fun innerFunc(_ param: auth(mapping MyMap) &InnerObj): Int {
                        return 123
                    }
                    var f = innerFunc // will fail if we're called via a reference
                    return nil
                }
            }

            fun test() {
                (&FuncGenerator() as auth(Outer1) &FuncGenerator).generate()
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[2])
	})
}

func TestCheckInvalidAuthMapping(t *testing.T) {

	t.Parallel()

	t.Run("variable", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            let x: auth(mapping Identity) &Int = &1 as &Int
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {
                let x: auth(mapping Identity) &Int

                init() {
                    self.x = &1 as &Int
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("function parameter type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test(x: auth(mapping Identity) &Int) {}
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("function return type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test(): auth(mapping Identity) &Int {
                return &1 as &Int
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("casting", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test(x: AnyStruct) {
                x as! auth(mapping Identity) &Int
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})

	t.Run("type argument", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test() {
                Type<auth(mapping Identity) &Int>()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAuthorizationError{}, errs[0])
	})
}

func TestCheckNestedReferenceMemberAccess(t *testing.T) {

	t.Parallel()

	t.Run("struct entitlement escalation", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `

          entitlement E

          struct T {
              access(E) fun foo() {}
          }

          struct S {
              access(mapping Identity) var ref: AnyStruct

              init(_ a: AnyStruct){
                  self.ref = a
              }
          }

          fun test() {
              let t = T()
              let pubTRef = &t as &T
              var s = &S(pubTRef) as auth(E) &S
              var tRef = s.ref as! auth(E) &T
              tRef.foo()
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[0])
	})

	t.Run("entitled struct escalation", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `

          entitlement E

          struct T {
              access(E) fun foo() {}
          }

          struct S {
              access(mapping Identity) var ref: AnyStruct

              init(_ a: AnyStruct){
                  self.ref = a
              }
          }

          fun test() {
              let t = T()
              let pubTRef = &t as auth(E) &T
              var s = &S(pubTRef) as auth(E) &S
              var member = s.ref
              var tRef = member as! auth(E) &T
              tRef.foo()
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[0])
	})

	t.Run("resource entitlement escalation", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `

          entitlement E

          resource R {
              access(E) fun foo() {}
          }

          struct S {
              access(mapping Identity) var ref: AnyStruct

              init(_ a: AnyStruct){
                  self.ref = a
              }
          }

          fun test() {
              let r <- create R()
              let pubRef = &r as &R
              var s = &S(pubRef) as auth(E) &S
              var entitledRef = s.ref as! auth(E) &R
              entitledRef.foo()

              destroy r
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[0])
	})
}

func TestCheckMappingAccessFieldType(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, mapping, ty string, valid bool) {

		testName := fmt.Sprintf("%s, %s", mapping, ty)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      entitlement X
                      entitlement Y

                      entitlement mapping M {
                          X -> Y
                      }

                      struct S {
                          access(mapping %[1]s) var x: %[2]s

                          init(_ x: %[2]s) {
                              self.x = x
                          }
                      }

                      struct T {
                          let ref: &Int

                          init(_ ref: &Int) {
                              self.ref = ref
                          }
                      }
                    `,
					mapping,
					ty,
				),
			)
			if valid {
				require.NoError(t, err)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, &sema.InvalidMappingAccessMemberTypeError{}, errs[0])
			}
		})
	}

	for _, mapping := range []string{"Identity", "M"} {
		// Primitive
		test(t, mapping, "Int", false)
		// Reference to primitive
		test(t, mapping, "&Int", false)

		// AnyStruct
		test(t, mapping, "AnyStruct", false)
		// Reference to AnyStruct
		test(t, mapping, "&AnyStruct", false)

		// Struct with reference field
		test(t, mapping, "T", true)
		// Reference to struct with reference field
		test(t, mapping, "&T", false)

		// Array
		test(t, mapping, "[Int]", true)
		test(t, mapping, "[&Int]", true)
		test(t, mapping, "[AnyStruct]", true)
		test(t, mapping, "[&AnyStruct]", true)
		test(t, mapping, "[T]", true)
		test(t, mapping, "[&T]", true)
		// Reference to array
		test(t, mapping, "&[Int]", false)
		test(t, mapping, "&[&Int]", false)
		test(t, mapping, "&[AnyStruct]", false)
		test(t, mapping, "&[&AnyStruct]", false)
		test(t, mapping, "&[T]", false)
		test(t, mapping, "&[&T]", false)

		// Dictionary
		test(t, mapping, "{Int: Int}", true)
		test(t, mapping, "{Int: &Int}", true)
		test(t, mapping, "{Int: AnyStruct}", true)
		test(t, mapping, "{Int: &AnyStruct}", true)
		test(t, mapping, "{Int: T}", true)
		test(t, mapping, "{Int: &T}", true)
		// Reference to dictionary
		test(t, mapping, "&{Int: Int}", false)
		test(t, mapping, "&{Int: &Int}", false)
		test(t, mapping, "&{Int: AnyStruct}", false)
		test(t, mapping, "&{Int: &AnyStruct}", false)
		test(t, mapping, "&{Int: T}", false)
		test(t, mapping, "&{Int: &T}", false)
	}
}
