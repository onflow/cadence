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

func TestCheckBasic(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`attachment Test for AnyStruct {}`,
	)

	require.NoError(t, err)
}

func TestCheckRedeclare(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`struct R {} 
		 attachment R for AnyStruct {}`,
	)

	errs := RequireCheckerErrors(t, err, 2)

	// 2 redeclaration errors: one for the constructor, one for the type
	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
	assert.IsType(t, &sema.RedeclarationError{}, errs[1])
}

func TestCheckRedeclareInContract(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`contract C {
			attachment C for AnyStruct {}
		}`,
	)

	errs := RequireCheckerErrors(t, err, 2)

	// 2 redeclaration errors: one for the constructor, one for the type
	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
	assert.IsType(t, &sema.RedeclarationError{}, errs[1])
}

func TestCheckBaseType(t *testing.T) {

	t.Parallel()

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment Test for S {}`,
		)

		require.NoError(t, err)
	})

	t.Run("struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface S {}
			attachment Test for S {}`,
		)

		require.NoError(t, err)
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment Test for R {}`,
		)

		require.NoError(t, err)
	})

	t.Run("resource interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface R {}
			attachment Test for R {}`,
		)

		require.NoError(t, err)
	})

	t.Run("order", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment Test for R {}
			resource R {}`,
		)

		require.NoError(t, err)
	})

	t.Run("anystruct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment Test for AnyStruct {}`,
		)

		require.NoError(t, err)
	})

	t.Run("anyresource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment Test for AnyResource {}`,
		)

		require.NoError(t, err)
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment Test for Int {}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("contract", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract Test {}
			attachment A for Test {}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("contract interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {}
			attachment A for Test {}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("enum", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			enum E: Int {}
			attachment Test for E {}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("struct attachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment S for AnyStruct {}
			attachment Test for S {}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("resource attachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment R for AnyResource {}
			attachment Test for R {}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("event", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			event E()
			attachment Test for E {}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("recursive", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for A {}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("mutually recursive", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for B {}
			attachment B for A {}`,
		)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[1])
		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[2])
	})
}

func TestCheckNestedBaseType(t *testing.T) {

	t.Parallel()

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {
				attachment Test for S {}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface S {
				attachment Test for S {}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {
				attachment Test for R {}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("resource interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface R {
				attachment Test for R {}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("contract", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract Test {
				attachment A for Test {}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("contract interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				attachment A for Test {}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
	})

	t.Run("qualified base type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract C {
				struct S {
					fun foo() {}
				}
			}
			pub attachment A for C.S {
				fun bar() {
					super.foo()
				}
			}
			`,
		)

		require.NoError(t, err)
	})

	t.Run("unqualified base type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract C {
				struct S {
					fun foo() {}
				}
			}
			pub attachment A for S {
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		// 2 errors, for undeclared type, one for invalid type in base type
		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckTypeRequirement(t *testing.T) {

	t.Parallel()

	t.Run("no attachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				attachment A for AnyStruct {
					fun foo(): Int 
				}
			}
			contract C: Test {

			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("concrete struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				attachment A for AnyStruct {}
			}
			contract C: Test {
				struct A {}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("concrete resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				attachment A for AnyStruct {}
			}
			contract C: Test {
				resource A {}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("missing method", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				attachment A for AnyStruct {
					fun foo(): Int 
				}
			}
			contract C: Test {
				attachment A for AnyStruct {

				}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("missing field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				attachment A for AnyStruct {
					let x: Int
				}
			}
			contract C: Test {
				attachment A for AnyStruct {

				}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("incompatible base type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				struct S {}
				attachment A for S {
				}
			}
			contract C: Test {
				struct S {}
				struct S2 {}
				attachment A for S2 {
				}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("basetype subtype", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				attachment A for AnyStruct {
				}
			}
			contract C: Test {
				struct S {}
				attachment A for S {
				}
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("base type supertype", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				struct S {}
				attachment A for S {
				}
			}
			contract C: Test {
				struct S {}
				attachment A for AnyStruct {
				}
			}
			`,
		)

		require.NoError(t, err)
	})

	t.Run("conforms", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract interface Test {
				attachment A for AnyStruct {
					fun foo(): Int 
				}
			}
			contract C: Test {
				attachment A for AnyStruct {
					fun foo(): Int {return 3}
				}
			}
			`,
		)

		require.NoError(t, err)
	})
}

func TestCheckWithMembers(t *testing.T) {

	t.Parallel()

	t.Run("field missing init", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment Test for R {
				let x: Int
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.MissingInitializerError{}, errs[0])
	})

	t.Run("field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment Test for R {
				let x: Int
				init(x: Int) {
					self.x = x
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("resource field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment Test for R {
				let x: @R
				init(x: @R) {
					self.x <- x
				}
				destroy() {
					destroy self.x
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("resource field in struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment Test for AnyStruct {
				let x: @R
				init(x: @R) {
					self.x <- x
				}
				destroy() {
					destroy self.x
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidResourceFieldError{}, errs[0])
		assert.IsType(t, &sema.InvalidDestructorError{}, errs[1])
	})

	t.Run("field with same name as base type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {
				let x: Int
				init(x: Int) {
					self.x = x
				}
			}
			attachment Test for R {
				let x: Int
				init(x: Int) {
					self.x = x
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("method", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment Test for R {
				fun foo() {}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("destroy", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment Test for R {
				destroy() {}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("destroy in struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment Test for S {
				destroy() {}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidDestructorError{}, errs[0])
	})
}

func TestCheckConformance(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {
			}
			attachment Test for R: I {
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {
				let x: Int
			}
			attachment Test for R: I {
				let x: Int
				init(x: Int) {
					self.x = x
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("method", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {
				fun x(): Int
			}
			attachment Test for R: I {
				fun x(): Int { return 0 }
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("field missing", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {
				let x: Int
			}
			attachment Test for R: I {
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("field type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {
				let x: Int
			}
			attachment Test for R: I {
				let x: AnyStruct
				init(x: AnyStruct) {
					self.x = x
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("method missing", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {
				fun x(): Int
			}
			attachment Test for R: I {
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("method type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {
				fun x(): Int
			}
			attachment Test for R: I {
				fun x(): AnyStruct { return "" }
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("method missing, exists in base type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {
				fun x(): Int { return 3 }
			}
			resource interface I {
				fun x(): Int
			}
			attachment Test for R: I {
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("kind mismatch resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			struct interface I {}
			attachment Test for R: I {}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("kind mismatch struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct R {}
			resource interface I {}
			attachment Test for R: I {}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("conforms to base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			attachment A for I: I {}`,
		)

		require.NoError(t, err)
	})

	t.Run("anyresource base, resource conformance", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			attachment A for AnyResource: I {}`,
		)

		require.NoError(t, err)
	})

	t.Run("anystruct base, struct conformance", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			attachment A for AnyStruct: I {}`,
		)

		require.NoError(t, err)
	})

	t.Run("anystruct base, resource conformance", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			attachment A for AnyStruct: I {}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("anyresource base, struct conformance", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			attachment A for AnyResource: I {}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("cross-contract concrete base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract C0 {
				resource interface R {}
			}
			contract C1 {
				resource R {}
				attachment A for R: C0.R {}
			}
			`,
		)

		require.NoError(t, err)
	})

	t.Run("cross-contract interface base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract C0 {
				resource R {}
			}
			contract C1 {
				resource interface R {}
				attachment A for C0.R: R {}
			}
			`,
		)

		require.NoError(t, err)
	})
}

func TestCheckSuper(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment Test for S {
				fun foo() {
					let x: &S = super
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			attachment Test for I {
				fun foo() {
					let x: &{I} = super
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("init", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {
				fun foo(): Int {
					return 3
				}
			}
			attachment Test for S {
				let x: Int
				init() {
					self.x = super.foo()
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("destroy", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {
				fun foo() {}
			}
			attachment Test for R {
				destroy() {
					super.foo()
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("interface super", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface R {
				fun foo(): Int {
					return 3
				}
			}
			attachment Test for R {
				let x: Int
				init() {
					self.x = super.foo()
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("super in struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {
				let x: Int
				init() {
					self.x = super
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("super in resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource S {
				let x: Int
				init() {
					self.x = super
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("super in contract", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract S {
				let x: Int
				init() {
					self.x = super
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckSuperScoping(t *testing.T) {

	t.Parallel()

	t.Run("pub member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			pub struct S {
				pub fun foo() {}
			}
			pub attachment Test for S {
				fun foo() {
					super.foo()
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("priv member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			pub struct S {
				priv fun foo() {}
			}
			pub attachment Test for S {
				fun foo() {
					super.foo()
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAccessError{}, errs[0])
	})

	t.Run("contract member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			pub contract C {
				pub struct S {
					access(contract) fun foo() {}
				}
			}
			pub attachment Test for C.S {
				fun foo() {
					super.foo()
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAccessError{}, errs[0])
	})

	t.Run("contract member valid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			pub contract C {
				pub struct S {
					access(contract) fun foo() {}
				}
				pub attachment Test for S {
					fun foo() {
						super.foo()
					}
				}
			}`,
		)

		require.NoError(t, err)
	})
}

func TestCheckSuperTyping(t *testing.T) {

	t.Parallel()

	t.Run("struct cast", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct R: I {}
			struct interface I {}
			attachment Test for I {
				fun foo() {
					let x = super as! &R{I}
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		// super is not auth
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("resource cast", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R: I {}
			resource interface I {}
			attachment Test for I {
				fun foo() {
					let x = super as! &R{I}
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		// super is not auth
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestCheckAttachmentType(t *testing.T) {

	t.Parallel()

	t.Run("reference", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment T for AnyStruct {}
			fun foo(x: &T) {}
			`,
		)

		require.NoError(t, err)
	})

	t.Run("resource reference", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment T for AnyResource {}
			fun foo(x: &T) {}
			`,
		)

		require.NoError(t, err)
	})

	t.Run("struct resource annotation", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment T for AnyStruct {}
			fun foo(x: @T) {}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
	})

	t.Run("resource annotation", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment T for AnyResource {}
			fun foo(x: @T) {
				destroy x
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
	})

	t.Run("resource without resource annotation", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment T for AnyResource {}
			fun foo(x: T) {
				destroy x
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
	})

	t.Run("optional annotation", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment T for AnyStruct {}
			fun foo(x: T?) {
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment T for AnyResource {}
			fun foo(x: [T]) {
				destroy x
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
	})

	t.Run("nested reference", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment T for AnyResource {}
			fun foo(x: [&T]) {
			}
			`,
		)

		require.NoError(t, err)
	})
}

func TestCheckIllegalInit(t *testing.T) {

	t.Parallel()

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`attachment Test for AnyStruct {}
			let t = Test()
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[0])
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`attachment Test for AnyResource {}
			pub fun foo() {
				let t <- Test()
				destroy t
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[0])
	})
}

func TestCheckAttachNonAttachment(t *testing.T) {

	t.Parallel()

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			pub fun A() {}
			pub fun foo() {
				attach A() to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachNonAttachmentError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			pub struct S {}
			pub fun foo() {
				attach S() to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachNonAttachmentError{}, errs[0])
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			pub resource R {}
			pub fun foo() {
				attach R() to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.MissingCreateError{}, errs[0])
		assert.IsType(t, &sema.AttachNonAttachmentError{}, errs[1])
	})

	t.Run("event", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			event E()
			pub fun foo() {
				attach E() to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachNonAttachmentError{}, errs[0])
	})

	t.Run("enum", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			enum E: Int {}
			pub fun foo() {
				attach E(rawValue: 0) to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachNonAttachmentError{}, errs[0])
	})

	t.Run("contract", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract C {}
			pub fun foo() {
				attach C() to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotCallableError{}, errs[0])
	})
}

func TestCheckAttachToNonComposite(t *testing.T) {

	t.Parallel()

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			pub fun foo() {
				attach A() to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachToInvalidTypeError{}, errs[0])
	})

	t.Run("reference", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S{}
			attachment A for AnyStruct {}
			pub fun foo() {
				let s = S()
				attach A() to (&s as &S)
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachToInvalidTypeError{}, errs[0])
	})

	t.Run("non-composite nonresource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyResource {}
			pub fun foo() {
				attach A() to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.MissingMoveOperationError{}, errs[1])
		assert.IsType(t, &sema.AttachToInvalidTypeError{}, errs[2])
	})

	t.Run("array", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			pub fun foo() {
				attach A() to [4]
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachToInvalidTypeError{}, errs[0])
	})

	t.Run("event", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			event E()
			pub fun foo() {
				attach A() to E()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidEventUsageError{}, errs[0])
	})

	t.Run("contract", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			contract C {}
			pub fun foo() {
				attach A() to C()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotCallableError{}, errs[0])
	})

	t.Run("attachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			attachment B for AnyStruct {}
			pub fun foo() {
				attach A() to B()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[0])
	})

	t.Run("enum", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			enum E: Int { }
			pub fun foo() {
				attach A() to E(rawValue: 0)
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachToInvalidTypeError{}, errs[0])
	})

	t.Run("resource array", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for AnyResource {}
			pub fun foo() {
				let r <- attach A() to <-[<-create R()]
				destroy r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachToInvalidTypeError{}, errs[0])
	})
}

func TestCheckAttach(t *testing.T) {

	t.Parallel()

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for S {}
			pub fun foo() {
				attach A() to S()
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("loss of resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for R {}
			pub fun foo() {
				attach A() to <-create R()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for R {}
			pub fun foo() {
				let r <- attach A() to <-create R()
				destroy r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("enforce type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for R {}
			pub fun foo() {
				let r <- create R()
				let r2: @R <- attach A() to <-r
				destroy r2
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("resource not moved", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for R {}
			pub fun foo() {
				let r <- attach A() to create R()
				destroy r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.MissingMoveOperationError{}, errs[0])
	})

	t.Run("struct AnyStruct subtyping", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for AnyStruct {}
			pub fun foo() {
				attach A() to S()
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("resource AnyResource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for AnyResource {}
			pub fun foo() {
				let r <- attach A() to <-create R()
				destroy r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S: I {}
			struct interface I {}
			attachment A for I {}
			pub fun foo() {
				attach A() to S()
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("struct interface non-conform", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			struct interface I {}
			attachment A for I {}
			pub fun foo() {
				attach A() to S()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("resource interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R: I {}
			resource interface I {}
			attachment A for I {}
			pub fun foo() {
				let r <- attach A() to <-create R()
				destroy r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("resource interface non conform", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {}
			attachment A for I {}
			pub fun foo() {
				let r <- attach A() to <-create R()
				destroy r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct resource mismatch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			resource interface I {}
			attachment A for I {}
			pub fun foo() {
				let r <- attach A() to S()
				destroy r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.MissingMoveOperationError{}, errs[1])
	})

	t.Run("resource struct mismatch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			struct interface I {}
			attachment A for I {}
			pub fun foo() {
				attach A() to <-create R()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("resource anystruct mismatch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for AnyStruct {}
			pub fun foo() {
				attach A() to <-create R()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct anyresource mismatch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for AnyResource {}
			pub fun foo() {
				let r <- attach A() to S()
				destroy r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.MissingMoveOperationError{}, errs[1])
	})

	t.Run("attach struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct S: I {}
			attachment A for AnyStruct {}
			pub fun foo() {
				let s: S{I} = S()
				attach A() to s
			}
		`,
		)

		require.NoError(t, err)
	})
}

func TestCheckAttachToRestrictedType(t *testing.T) {

	t.Parallel()

	t.Run("struct restricted", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct S: I {}
			attachment A for AnyStruct {}
			pub fun foo() {
				let s: S{I} = S()
				attach A() to s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("any struct restricted", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct S: I {}
			attachment A for AnyStruct {}
			pub fun foo() {
				let s: {I} = S()
				attach A() to s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("resource restricted", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			resource R: I {}
			attachment A for AnyResource {}
			pub fun foo() {
				let r: @R{I} <- create R()
				destroy attach A() to <-r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("anyresource restricted", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			resource R: I {}
			attachment A for AnyResource {}
			pub fun foo() {
				let r: @{I} <- create R()
				destroy attach A() to <-r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach struct interface to struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct S: I {}
			attachment A for I {}
			pub fun foo() {
				let s: S{I} = S()
				attach A() to s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach struct interface to struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct S: I {}
			attachment A for S {}
			pub fun foo() {
				let s: S{I} = S()
				attach A() to s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach anystruct interface to struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct S: I {}
			attachment A for S {}
			pub fun foo() {
				let s: {I} = S()
				attach A() to s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("attach resource interface to resource interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			resource R: I {}
			attachment A for I {}
			pub fun foo() {
				let r: @R{I} <- create R()
				destroy attach A() to <-r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach resource interface to resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			resource R: I {}
			attachment A for R {}
			pub fun foo() {
				let r: @R{I} <- create R()
				destroy attach A() to <-r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach anyresource interface to resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			resource R: I {}
			attachment A for R {}
			pub fun foo() {
				let r: @{I} <- create R()
				destroy attach A() to <-r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("attach anystruct interface to struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct S: I {}
			attachment A for I {}
			pub fun foo() {
				let s: {I} = S()
				attach A() to s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach multiply restricted to struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct interface I2 {}
			struct S: I, I2 {}
			attachment A for I {}
			pub fun foo() {
				let s: {I, I2} = S()
				attach A() to s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach anyresource interface to resource interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			resource R: I {}
			attachment A for I {}
			pub fun foo() {
				let r: @{I} <- create R()
				destroy attach A() to <-r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach multiply restricted to resource interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			resource interface I2 {}
			resource R: I, I2 {}
			attachment A for I {}
			pub fun foo() {
				let r: @{I, I2} <- create R()
				destroy attach A() to <-r
			}
		`,
		)

		require.NoError(t, err)
	})

	// TODO: once interfaces can conform to interfaces, add more tests here for interface hierarchy
}

func TestCheckAttachWithArguments(t *testing.T) {

	t.Parallel()

	t.Run("attach one arg", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for S {
				let x: Int 
				init(x: Int) {
					self.x = x
				}
			}
			pub fun foo() {
				attach A(x: 3) to S()
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach two arg", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for S {
				let x: Int 
				let y: String
				init(x: Int, y: String) {
					self.x = x
					self.y = y
				}
			}
			pub fun foo() {
				attach A(x: 3, y: "") to S()
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("missing labels", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for S {
				let x: Int 
				let y: String
				init(x: Int, y: String) {
					self.x = x
					self.y = y
				}
			}
			pub fun foo() {
				attach A(3, "") to S()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[0])
		assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
	})

	t.Run("wrong labels", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for S {
				let x: Int 
				let y: String
				init(x: Int, y: String) {
					self.x = x
					self.y = y
				}
			}
			pub fun foo() {
				attach A(z: 3, a: "") to S()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
		assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[1])
	})
}
