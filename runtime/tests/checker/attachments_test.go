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

	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
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
}

func TestCheckConformance(t *testing.T) {

	t.Parallel()

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
