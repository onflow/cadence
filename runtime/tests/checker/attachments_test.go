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
	"fmt"
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

	t.Run("invalid type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for B {}`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[1])
	})
}

func TestCheckBuiltin(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`attachment Test for AuthAccount {}`,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[0])
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
			access(all) attachment A for C.S {
				fun bar() {
					base.foo()
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
			access(all) attachment A for S {
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

	t.Run("base type Basetype", func(t *testing.T) {

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

	t.Run("resource field no destroy", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment Test for R {
				let x: @R
				init(x: @R) {
					self.x <- x
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.MissingDestructorError{}, errs[0])
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

func TestCheckBase(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment Test for S {
				fun foo() {
					let x: &S = base
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
					let x: &{I} = base
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
					self.x = base.foo()
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
					base.foo()
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("interface base", func(t *testing.T) {

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
					self.x = base.foo()
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("base in struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {
				let x: Int
				init() {
					self.x = base
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("base in resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource S {
				let x: Int
				init() {
					self.x = base
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("base in contract", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract S {
				let x: Int
				init() {
					self.x = base
				}
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("base outside composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun foo() {
				let x = base
			}`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckBaseScoping(t *testing.T) {

	t.Parallel()

	t.Run("access(all) member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			access(all) struct S {
				access(all) fun foo() {}
			}
			access(all) attachment Test for S {
				fun foo() {
					base.foo()
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("access(self) member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			access(all) struct S {
				access(self) fun foo() {}
			}
			access(all) attachment Test for S {
				fun foo() {
					base.foo()
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
			access(all) contract C {
				access(all) struct S {
					access(contract) fun foo() {}
				}
			}
			access(all) attachment Test for C.S {
				fun foo() {
					base.foo()
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
			access(all) contract C {
				access(all) struct S {
					access(contract) fun foo() {}
				}
				access(all) attachment Test for S {
					fun foo() {
						base.foo()
					}
				}
			}`,
		)

		require.NoError(t, err)
	})
}

func TestCheckBaseTyping(t *testing.T) {

	t.Parallel()

	t.Run("struct cast", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct R: I {}
			struct interface I {}
			attachment Test for I {
				fun foo() {
					let x = base as! &R{I}
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("resource cast", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R: I {}
			resource interface I {}
			attachment Test for I {
				fun foo() {
					let x = base as! &R{I}
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("struct return", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			attachment Test for I {
				fun foo(): &{I} {
					return base
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("resource return", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			attachment Test for I {
				fun foo(): &{I} {
					return base
				}
			}`,
		)

		require.NoError(t, err)
	})
}

func TestCheckSelfTyping(t *testing.T) {

	t.Parallel()

	t.Run("return self", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct R {}
			attachment Test for R {
				fun foo(): &Test {
					return self
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("return self", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct R {}
			attachment Test for R {
				fun foo(): &AnyStruct {
					return self
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("return self resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment Test for R {
				fun foo(): &AnyResource {
					return self
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("return self struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct R {}
			struct interface I {}
			attachment Test for R: I {
				fun foo(): &{I} {
					return self
				}
			}`,
		)

		require.NoError(t, err)
	})

	t.Run("return self resource interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {}
			attachment Test for R: I {
				fun foo(): &{I} {
					return self
				}
			}`,
		)

		require.NoError(t, err)
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
			access(all) fun foo() {
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
			access(all) fun A() {}
			access(all) fun foo() {
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
			access(all) struct S {}
			access(all) fun foo() {
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
			access(all) resource R {}
			access(all) fun foo() {
				attach R() to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.MissingCreateError{}, errs[0])
		assert.IsType(t, &sema.MissingMoveOperationError{}, errs[1])
		assert.IsType(t, &sema.AttachNonAttachmentError{}, errs[2])
	})

	t.Run("event", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			event E()
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
				attach A() to 4
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		assert.IsType(t, &sema.MissingMoveOperationError{}, errs[0])
	})

	t.Run("array", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
				attach A() to S()
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("cannot attach directly to anystruct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for AnyStruct {}
			access(all) fun foo() {
				attach A() to (S() as AnyStruct)
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachToInvalidTypeError{}, errs[0])
	})

	t.Run("cannot attach directly to anyresource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource S {}
			attachment A for AnyResource {}
			access(all) fun foo() {
				destroy attach A() to <-(create S() as @AnyResource)
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AttachToInvalidTypeError{}, errs[0])
	})

	t.Run("resource AnyResource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for AnyResource {}
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
				let r <- attach A() to S()
				destroy r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.MissingMoveOperationError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("resource struct mismatch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			struct interface I {}
			attachment A for I {}
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
				let r <- attach A() to S()
				destroy r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.MissingMoveOperationError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("attach struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct S: I {}
			attachment A for AnyStruct {}
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
				let s: S{I} = S()
				attach A() to s
			}
		`,
		)

		// there is no reason to error here; the owner of this
		// restricted type is always able to unrestrict

		require.NoError(t, err)
	})

	t.Run("attach anystruct interface to struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct S: I {}
			attachment A for S {}
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
				let r: @R{I} <- create R()
				destroy attach A() to <-r
			}
		`,
		)

		// owner can unrestrict `r` as they wish, so there is no reason to
		// limit attach here

		require.NoError(t, err)
	})

	t.Run("attach anyresource interface to resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			resource R: I {}
			attachment A for R {}
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
				attach A(x: 3) to S()
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("attach base argument", func(t *testing.T) {

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
			access(all) fun foo() {
				attach A(x: base) to S()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
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
			access(all) fun foo() {
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
			access(all) fun foo() {
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
			access(all) fun foo() {
				attach A(z: 3, a: "") to S()
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
		assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[1])
	})
}

func TestCheckAttachInvalidType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
		resource C {}
		attachment A for B {}
		access(all) fun foo() {
			destroy attach A() to <- create C()
		}`,
	)

	errs := RequireCheckerErrors(t, err, 3)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	assert.IsType(t, &sema.InvalidBaseTypeError{}, errs[1])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
}

func TestCheckAnyAttachmentTypes(t *testing.T) {

	type TestCase struct {
		subType         string
		setupCode       string
		expectedSuccess bool
	}

	testCases := func(resource bool) []TestCase {
		return []TestCase{
			{
				subType:         "Int",
				expectedSuccess: false,
			},
			{
				subType:         "AnyStruct",
				expectedSuccess: false,
			},
			{
				subType:         "AnyResource",
				expectedSuccess: false,
			},
			{
				setupCode:       "attachment S for AnyStruct {}",
				subType:         "S",
				expectedSuccess: !resource,
			},
			{
				setupCode:       "struct S2 {}",
				subType:         "S2",
				expectedSuccess: false,
			},
			{
				setupCode:       "struct S {}\nattachment S3 for S {}",
				subType:         "S3",
				expectedSuccess: !resource,
			},
			{
				setupCode:       "attachment R for AnyResource {}",
				subType:         "R",
				expectedSuccess: resource,
			},
			{
				setupCode:       "resource R2 {}",
				subType:         "R2",
				expectedSuccess: false,
			},
			{
				setupCode:       "resource R {}\nattachment R3 for R {}",
				subType:         "R3",
				expectedSuccess: resource,
			},
			{
				setupCode:       "event E()",
				subType:         "E",
				expectedSuccess: false,
			},
			{
				setupCode:       "contract C {}",
				subType:         "C",
				expectedSuccess: false,
			},
			{
				setupCode:       "contract interface CI {}",
				subType:         "CI",
				expectedSuccess: false,
			},
		}
	}

	t.Run("AnyStructAttachmentType", func(t *testing.T) {

		for _, testCase := range testCases(false) {
			t.Run(testCase.subType, func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(`
					%s
					access(all) fun foo(x: &%s): &AnyStructAttachment {
						return x
					}
				`, testCase.setupCode, testCase.subType),
				)

				if testCase.expectedSuccess {
					require.NoError(t, err)
				} else {
					errs := RequireCheckerErrors(t, err, 1)
					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})
		}
	})

	t.Run("AnyResourceAttachmentType", func(t *testing.T) {
		for _, testCase := range testCases(true) {
			t.Run(testCase.subType, func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(`
					%s
					access(all) fun foo(x: &%s): &AnyResourceAttachment {
						return x
					}
				`, testCase.setupCode, testCase.subType),
				)

				if testCase.expectedSuccess {
					require.NoError(t, err)
				} else {
					errs := RequireCheckerErrors(t, err, 1)
					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})
		}
	})
}

func TestCheckRemove(t *testing.T) {

	t.Parallel()

	t.Run("basic struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for S {}
			access(all) fun foo(s: S) {
				remove A from s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("basic resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for R {}
			access(all) fun foo(r: @R) {
				remove A from r
				destroy r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("resource lost", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for R {}
			access(all) fun foo(r: @R) {
				remove A from r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("struct with anystruct base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for AnyStruct {}
			access(all) fun foo(s: S) {
				remove A from s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("struct with struct interface base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S: I {}
			struct interface I {}
			attachment A for I {}
			access(all) fun foo(s: S) {
				remove A from s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("struct with no implement", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			struct interface I {}
			attachment A for I {}
			access(all) fun foo(s: S) {
				remove A from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("resource with anyresource base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			attachment A for AnyResource {}
			access(all) fun foo(r: @R) {
				remove A from r
				destroy r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("resource with interface base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R: I {}
			resource interface I {}
			attachment A for I {}
			access(all) fun foo(r: @R) {
				remove A from r
				destroy r
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("resource with interface no implements", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource R {}
			resource interface I {}
			attachment A for I {}
			access(all) fun foo(r: @R) {
				remove A from r
				destroy r
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("qualified type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			contract C {
				struct S {}
				attachment A for S {}
			}
			access(all) fun foo(s: C.S) {
				remove C.A from s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("noncomposite base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for S {}
			access(all) fun foo(s: Int) {
				remove A from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("cannot remove from anystruct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			access(all) fun foo(s: AnyStruct) {
				remove A from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("cannot remove from anyresource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyResource {}
			access(all) fun foo(s: @AnyResource) {
				remove A from s
				destroy s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("noncomposite base anystruct declaration", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			access(all) fun foo(s: Int) {
				remove A from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove non-attachment struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for S {}
			access(all) fun foo(s: S) {
				remove S from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove non-attachment resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource S {}
			attachment A for S {}
			access(all) fun foo(s: @S) {
				remove S from s
				destroy s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove nondeclared", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			attachment A for S {}
			access(all) fun foo(s: S) {
				remove X from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("remove event", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			event E()
			access(all) fun foo(s: S) {
				remove E from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove contract", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			contract C {}
			access(all) fun foo(s: S) {
				remove C from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove resource interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			resource interface C {}
			access(all) fun foo(s: S) {
				remove C from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			resource interface C {}
			access(all) fun foo(s: S) {
				remove C from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove anystruct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			access(all) fun foo(s: S) {
				remove AnyStruct from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove anyresource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource S {}
			access(all) fun foo(s: @S) {
				remove AnyResource from s
				destroy s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove anystructattachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S {}
			access(all) fun foo(s: S) {
				remove AnyStructAttachment from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("remove anyresourceattachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource S {}
			access(all) fun foo(s: @S) {
				remove AnyResourceAttachment from s
				destroy s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

}

func TestCheckRemoveFromRestricted(t *testing.T) {

	t.Parallel()

	t.Run("basic struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S: I {}
			struct interface I {}
			attachment A for S {}
			access(all) fun foo(s: S{I}) {
				remove A from s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("basic struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S: I {}
			struct interface I {}
			attachment A for I {}
			access(all) fun foo(s: S{I}) {
				remove A from s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("basic resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource S: I {}
			resource interface I {}
			attachment A for S {}
			access(all) fun foo(s: @S{I}) {
				remove A from s
				destroy s
			}
		`,
		)

		// owner can always unrestrict `s`, so no need to prevent removal of A

		require.NoError(t, err)
	})

	t.Run("basic resource interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource S: I {}
			resource interface I {}
			attachment A for I {}
			access(all) fun foo(s: @S{I}) {
				remove A from s
				destroy s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("struct base anystruct restricted", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S: I {}
			struct interface I {}
			attachment A for S {}
			access(all) fun foo(s: {I}) {
				remove A from s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("resource base anyresource restricted", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource S: I {}
			resource interface I {}
			attachment A for S {}
			access(all) fun foo(s: @{I}) {
				remove A from s
				destroy s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidAttachmentRemoveError{}, errs[0])
	})

	t.Run("interface base anystruct restricted", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			attachment A for I {}
			access(all) fun foo(s: {I}) {
				remove A from s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("interface base anyresource restricted", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			resource interface I {}
			attachment A for I {}
			access(all) fun foo(s: @{I}) {
				remove A from s
				destroy s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("multiple restriction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct S: I, J {}
			struct interface I {}
			struct interface J {}
			attachment A for I {}
			access(all) fun foo(s: S{I, J}) {
				remove A from s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("anystruct multiple restriction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			struct interface I {}
			struct interface J {}
			attachment A for I {}
			access(all) fun foo(s: {I, J}) {
				remove A from s
			}
		`,
		)

		require.NoError(t, err)
	})
}

func TestCheckAccessAttachment(t *testing.T) {

	t.Parallel()

	runTests := func(suffix, sigil, destructor string) {
		t.Run(fmt.Sprintf("basic %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					attachment A for R {}
					access(all) fun foo(r: %sR) {
						let a: &A? = r[A]
						%s
					}`, sigil, destructor),
			)
			require.NoError(t, err)
		})

		t.Run(fmt.Sprintf("non-composite %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					attachment A for R {}
					access(all) fun foo(r: %sR) {
						let a: &A? = r[Int]
						%s
					}`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("struct %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					struct D {}
					access(all) fun foo(r: %sR) {
						r[D]
						%s
					}`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("resource %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					resource X {}
					access(all) fun foo(r: %sR) {
						r[X]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("contract %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					contract X {}
					access(all) fun foo(r: %sR) {
						r[X]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("event %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					event X()
					access(all) fun foo(r: %sR) {
						r[X]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("enum %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					enum X: Int {}
					access(all) fun foo(r: %sR) {
						r[X]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("AnyStructAttachment %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					access(all) fun foo(r: %sR) {
						r[AnyStructAttachment]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("AnyResourceAttachment %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					access(all) fun foo(r: %sR) {
						r[AnyResourceAttachment]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("AnyStruct %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					access(all) fun foo(r: %sR) {
						r[AnyStruct]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("AnyResource %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					access(all) fun foo(r: %sR) {
						r[AnyResource]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("AnyResource index %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					attachment A for AnyResource {}
					access(all) fun foo(r: %sAnyResource) {
						r[A]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.NotIndexableTypeError{}, errs[0])
		})

		t.Run(fmt.Sprintf("interface %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource interface I {}
					resource R: I {}
					attachment A for I {}
					access(all) fun foo(r: %sR) {
						let a: &A? = r[A]
						%s
					}
				`, sigil, destructor),
			)
			require.NoError(t, err)
		})

		t.Run(fmt.Sprintf("interface indexer %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					resource interface I {}
					attachment A for AnyResource: I {}
					access(all) fun foo(r: %sR) {
						r[I]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("interface nonconforming %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource interface I {}
					resource R {}
					attachment A for I {}
					access(all) fun foo(r: %sR) {
						let a: &A? = r[A]
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
		})

		t.Run(fmt.Sprintf("not writeable %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					attachment A for R {}
					access(all) fun foo(r: %sR) {
						r[A] = 3
						%s
					}
				`, sigil, destructor),
			)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.NotIndexingAssignableTypeError{}, errs[0])
		})

		t.Run(fmt.Sprintf("qualified %s", suffix), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
					resource R {}
					contract C {
						attachment A for R {}
					}
					access(all) fun foo(r: %sR) {
						let a: &C.A? = r[C.A]
						%s
					}
				`, sigil, destructor),
			)
			require.NoError(t, err)
		})
	}

	runTests("resource", "@", "destroy r")
	runTests("reference", "&", "")

	t.Run("AnyStruct index", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for AnyStruct {}
			access(all) fun foo(r: AnyStruct) {
				r[A]
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotIndexableTypeError{}, errs[0])
	})

	t.Run("non-nominal array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			attachment A for S {}
			struct S {}
			access(all) fun foo(r: S) {
				r[[A]]
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
	})
}

func TestCheckAccessAttachmentRestricted(t *testing.T) {

	t.Parallel()

	t.Run("restricted", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct R: I {}
		struct interface I {}
		attachment A for I {}
		access(all) fun foo(r: R{I}) {
			r[A]
		}
		`,
		)
		require.NoError(t, err)
	})

	t.Run("restricted concrete base", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct R: I {}
		struct interface I {}
		attachment A for R {}
		access(all) fun foo(r: {I}) {
			r[A]
		}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
	})

	t.Run("restricted concrete base reference", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct R: I {}
		struct interface I {}
		attachment A for R {}
		access(all) fun foo(r: &R{I}) {
			r[A]
		}
		`,
		)
		require.NoError(t, err)
	})

	t.Run("restricted concrete base reference to interface", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct R: I {}
		struct interface I {}
		attachment A for R {}
		access(all) fun foo(r: &{I}) {
			r[A]
		}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
	})

	t.Run("restricted anystruct base", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct interface I {}
		attachment A for I {}
		access(all) fun foo(r: {I}) {
			r[A]
		}
		`,
		)
		require.NoError(t, err)
	})

	t.Run("restricted anystruct base interface", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct interface I {}
		attachment A for I {}
		access(all) fun foo(r: &{I}) {
			r[A]
		}
		`,
		)
		require.NoError(t, err)
	})

	t.Run("restricted invalid base", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct interface I {}
		struct interface J {}
		attachment A for I {}
		access(all) fun foo(r: {J}) {
			r[A]
		}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
	})

	t.Run("restricted multiply extended base", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct R: I, J {}
		struct interface I {}
		struct interface J {}
		attachment A for I {}
		access(all) fun foo(r: R{J}) {
			r[A]
		}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
	})

	t.Run("restricted multiply extended base reference", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct R: I, J {}
		struct interface I {}
		struct interface J {}
		attachment A for I {}
		access(all) fun foo(r: &R{J}) {
			r[A]
		}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
	})

	t.Run("restricted multiply restricted base", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct interface I {}
		struct interface J {}
		attachment A for I {}
		access(all) fun foo(r: {I, J}) {
			r[A]
		}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("restricted multiply restricted base interface", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		struct interface I {}
		struct interface J {}
		attachment A for I {}
		access(all) fun foo(r: &{I, J}) {
			r[A]
		}
		`,
		)

		require.NoError(t, err)
	})
}

func TestCheckAttachmentsExternalMutation(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
				access(all) resource R {}
				access(all) attachment A for R {
					access(all) let x: [String] 
					init() {
						self.x = ["x"]
					}
				}

				fun main(r: @R) {
					r[A]!.x.append("y")
					destroy r
				}
				`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ExternalMutationError{}, errs[0])
	})

	t.Run("in base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
				access(all) resource R {
					access(all) fun foo() {
						self[A]!.x.append("y")
					}
				}
				access(all) attachment A for R {
					access(all) let x: [String] 
					init() {
						self.x = ["x"]
					}
				}
				
				`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ExternalMutationError{}, errs[0])
	})

	t.Run("in self, through base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
				access(all) resource R {}
				access(all) attachment A for R {
					access(all) let x: [String] 
					init() {
						self.x = ["x"]
					}
					access(all) fun foo() {
						base[A]!.x.append("y")
					}
				}
				
				`,
		)

		require.NoError(t, err)
	})
}

func TestInterpretAttachmentBaseNonMember(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
		access(all) resource R {}
		access(all) attachment A for R {
			access(all) let base: &R
			init() {
				self.base = base
			}
		}
	`,
		)

		require.NoError(t, err)
	})

	t.Run("array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			access(all) resource R {}
			access(all) attachment A for R {
				access(all) let bases: [&R]
				init() {
					self.bases = [base]
				}
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("array append", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			access(all) resource R {}
			access(all) attachment A for R {
				access(all) let bases: [&R]
				init() {
					self.bases = []
					self.bases.append(base)
				}
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("array index", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			access(all) resource R {}
			access(all) attachment A for R {
				access(all) let bases: [&R]
				init() {
					self.bases = []
					self.bases[0] = base
				}
			}
		`,
		)

		require.NoError(t, err)
	})
}

func TestCheckAttachmentsResourceReference(t *testing.T) {
	t.Parallel()

	t.Run("attachment base moved", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
        resource R {}
        attachment A for R {
            fun foo(): Int { return 3 }
        }
        fun test(): Int {
            var r <- create R()
            let ref = &r as &R
            var r2 <- attach A() to <-r
            let i = ref[A]?.foo()!
            destroy r2
            return i
        }
    `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidatedResourceReferenceError{}, errs[0])
	})
}

func TestCheckAttachmentsNotEnabled(t *testing.T) {

	t.Parallel()

	parseAndCheckWithoutAttachments := func(t *testing.T, code string) (*sema.Checker, error) {
		return ParseAndCheckWithOptions(t, code, ParseAndCheckOptions{})
	}

	t.Run("declaration", func(t *testing.T) {

		t.Parallel()

		_, err := parseAndCheckWithoutAttachments(t,
			`
			struct S {}
			attachment Test for S {}`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.AttachmentsNotEnabledError{}, errs[0])
	})

	t.Run("attach", func(t *testing.T) {

		t.Parallel()

		_, err := parseAndCheckWithoutAttachments(t,
			`
			struct S {}
			let s = attach A() to S() 
			`,
		)

		errs := RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.AttachmentsNotEnabledError{}, errs[0])
		assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
	})

	t.Run("remove", func(t *testing.T) {

		t.Parallel()

		_, err := parseAndCheckWithoutAttachments(t,
			`
			struct S {}
			fun foo() {
				remove A from S() 
			}
			`,
		)

		errs := RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.AttachmentsNotEnabledError{}, errs[0])
		assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
	})

	t.Run("type indexing", func(t *testing.T) {

		t.Parallel()

		_, err := parseAndCheckWithoutAttachments(t,
			`
			struct S {}
			attachment A for S {}
			let s = S()
			let r = s[A]
			`,
		)

		errs := RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.AttachmentsNotEnabledError{}, errs[0])
		assert.IsType(t, &sema.AttachmentsNotEnabledError{}, errs[1])
	})

	t.Run("regular indexing ok", func(t *testing.T) {

		t.Parallel()

		_, err := parseAndCheckWithoutAttachments(t,
			`
			let x = [1, 2, 3]
			let y = x[2]
			`,
		)

		require.NoError(t, err)
	})
}

func TestCheckForEachAttachment(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStructAttachment) {}
			struct A {}
			access(all) fun foo(s: A) {
				s.forEachAttachment(bar)
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("type check return", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStructAttachment): Bool { return false }
			struct A {}
			access(all) fun foo(s: A) {
				s.forEachAttachment(bar)
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("param not reference", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: AnyStructAttachment) { }
			struct A {}
			access(all) fun foo(s: A) {
				s.forEachAttachment(bar)
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("param mismatch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyResource) { }
			struct A {}
			access(all) fun foo(s: A) {
				s.forEachAttachment(bar)
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("param supertype", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStruct) { }
			struct A {}
			access(all) fun foo(s: A) {
				s.forEachAttachment(bar)
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyResourceAttachment) {}
			resource A {}
			access(all) fun foo(s: @A) {
				s.forEachAttachment(bar)
				destroy s
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("resource type mismatch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStructAttachment) {}
			resource A {}
			access(all) fun foo(s: @A) {
				s.forEachAttachment(bar)
				destroy s
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("not on anystruct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyResourceAttachment) {}
			access(all) fun foo(s: AnyStruct) {
				s.forEachAttachment(bar)
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("not on anyresource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyResourceAttachment) {}
			access(all) fun foo(s: @AnyResource) {
				s.forEachAttachment(bar)
				destroy s
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("not on anyresourceAttachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyResourceAttachment) {}
			access(all) fun foo(s: &AnyResourceAttachment) {
				s.forEachAttachment(bar)
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("not on anyStructAttachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStructAttachment) {}
			access(all) fun foo(s: &AnyStructAttachment) {
				s.forEachAttachment(bar)
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("not on event", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStructAttachment) {}
			event E()
			access(all) fun foo(s: E) {
				s.forEachAttachment(bar)
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("not on contract", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStructAttachment) {}
			contract C {}
			access(all) fun foo(s: C) {
				s.forEachAttachment(bar)
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("not on enum", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStructAttachment) {}
			enum S:Int {}
			access(all) fun foo(s: S) {
				s.forEachAttachment(bar)
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("not on struct attachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStructAttachment) {}
			attachment S for AnyStruct {}
			access(all) fun foo(s: &S) {
				s.forEachAttachment(bar)
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("not on resource attachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			fun bar (_: &AnyStructAttachment) {}
			attachment R for AnyResource {}
			access(all) fun foo(s: &R) {
				s.forEachAttachment(bar)
			}
		`,
		)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("cannot redeclare forEachAttachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			access(all) struct S {
				access(all) fun forEachAttachment() {}
			}
		`,
		)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
	})

	t.Run("downcasting reference with entitlements", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
			entitlement F
			entitlement E
			entitlement mapping M {
				E -> F
			}
			fun bar (attachment: &AnyResourceAttachment) {
				if let a = attachment as? auth(F) &A {
					a.foo()
				}
			}
			resource R {}
			access(M) attachment A for R {
				access(F) fun foo() {}
			}
			access(all) fun foo(s: @R) {
				s.forEachAttachment(bar)
				destroy s
			}
		`,
		)

		require.NoError(t, err)
	})
}
