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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckPuritySubtyping(t *testing.T) {

	t.Parallel()

	t.Run("pure <: impure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pure fun foo() {}
		let x: ((): Void) = foo
		`)

		require.NoError(t, err)
	})

	t.Run("pure <: pure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pure fun foo() {}
		let x: (pure (): Void) = foo
		`)

		require.NoError(t, err)
	})

	t.Run("impure <: impure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		fun foo() {}
		let x: ((): Void) = foo
		`)

		require.NoError(t, err)
	})

	t.Run("impure <: pure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		fun foo() {}
		let x: (pure (): Void) = foo
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("contravariant ok", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pure fun foo(x:((): Void)) {}
		let x: (pure ((pure (): Void)): Void) = foo
		`)

		require.NoError(t, err)
	})

	t.Run("contravariant error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pure fun foo(f:(pure (): Void)) {}
		let x: (pure (((): Void)): Void) = foo
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("interface implementation member success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct interface I {
			pure fun foo()
			fun bar()
		}

		struct S: I {
			pure fun foo() {}
			pure fun bar() {}
		}
		`)

		require.NoError(t, err)
	})

	t.Run("interface implementation member failure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct interface I {
			pure fun foo()
			fun bar() 
		}

		struct S: I {
			fun foo() {}
			fun bar() {}
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("interface implementation initializer success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct interface I {
			pure init()
		}

		struct S: I {
			pure init() {}
		}
		`)

		require.NoError(t, err)
	})

	t.Run("interface implementation initializer explicit success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct interface I {
			pure init()
		}

		struct S: I {
			init() {}
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("interface implementation initializer success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct interface I {
			init()
		}

		struct S: I {
			pure init() {}
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("interface implementation initializer success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct interface I {
			init()
		}

		struct S: I {
			init() {}
		}
		`)

		require.NoError(t, err)
	})
}

func TestCheckPurityEnforcement(t *testing.T) {
	t.Run("pure function call", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pure fun bar() {}
		pure fun foo() {
			bar()
		}
		`)

		require.NoError(t, err)
	})

	t.Run("impure function call error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		fun bar() {}
		pure fun foo() {
			bar()
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 4, Column: 3},
			EndPos:   ast.Position{Offset: 42, Line: 4, Column: 7},
		})
	})

	t.Run("pure function call nested", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		fun bar() {}
		pure fun foo() {
			let f = fun() {
				bar()
			}
		}
		`)

		require.NoError(t, err)
	})

	t.Run("impure function call nested", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		fun bar() {}
		fun foo() {
			let f = pure fun() {
				bar()
			}
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 5, Column: 4},
			EndPos:   ast.Position{Offset: 42, Line: 5, Column: 8},
		})
	})
}
