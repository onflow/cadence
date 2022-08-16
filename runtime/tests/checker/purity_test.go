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

	t.Run("impure method call error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct S {
			fun bar() {}
		}
		pure fun foo(_ s: S) {
			s.bar()
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 62, Line: 6, Column: 3},
			EndPos:   ast.Position{Offset: 68, Line: 6, Column: 9},
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
			StartPos: ast.Position{Offset: 58, Line: 5, Column: 4},
			EndPos:   ast.Position{Offset: 62, Line: 5, Column: 8},
		})
	})

	t.Run("pure function call nested failure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		fun bar() {}
		pure fun foo() {
			let f = fun() {
				bar()
			}
			f()
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 72, Line: 7, Column: 3},
			EndPos:   ast.Position{Offset: 74, Line: 7, Column: 5},
		})
	})

	t.Run("save", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			authAccount.save(3, to: /storage/foo)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 23, Line: 3, Column: 3},
			EndPos:   ast.Position{Offset: 59, Line: 3, Column: 39},
		})
	})

	t.Run("load", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			authAccount.load<Int>(from: /storage/foo)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 23, Line: 3, Column: 3},
			EndPos:   ast.Position{Offset: 63, Line: 3, Column: 43},
		})
	})

	t.Run("type", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			authAccount.type(at: /storage/foo)
		}
		`)

		require.NoError(t, err)
	})

	t.Run("link", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			authAccount.link<&Int>(/private/foo, target: /storage/foo)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 23, Line: 3, Column: 3},
			EndPos:   ast.Position{Offset: 80, Line: 3, Column: 60},
		})
	})

	t.Run("unlink", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			authAccount.unlink(/private/foo)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 23, Line: 3, Column: 3},
			EndPos:   ast.Position{Offset: 54, Line: 3, Column: 34},
		})
	})

	t.Run("add contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			authAccount.contracts.add(name: "", code: [])
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 23, Line: 3, Column: 3},
			EndPos:   ast.Position{Offset: 67, Line: 3, Column: 47},
		})
	})

	t.Run("update contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			authAccount.contracts.update__experimental(name: "", code: [])
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 23, Line: 3, Column: 3},
			EndPos:   ast.Position{Offset: 84, Line: 3, Column: 64},
		})
	})

	t.Run("remove contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			authAccount.contracts.remove(name: "")
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 23, Line: 3, Column: 3},
			EndPos:   ast.Position{Offset: 60, Line: 3, Column: 40},
		})
	})

	t.Run("revoke key", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			authAccount.keys.revoke(keyIndex: 0)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 23, Line: 3, Column: 3},
			EndPos:   ast.Position{Offset: 58, Line: 3, Column: 38},
		})
	})

	t.Run("alias", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		pure fun foo() {
			let f = authAccount.contracts.remove
			f(name: "")
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 63, Line: 4, Column: 3},
			EndPos:   ast.Position{Offset: 73, Line: 4, Column: 13},
		})
	})

	t.Run("emit", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
		event FooEvent()
		pure fun foo() {
			emit FooEvent()
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 47, Line: 4, Column: 8},
			EndPos:   ast.Position{Offset: 56, Line: 4, Column: 17},
		})
	})

	t.Run("external write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		var a = 3
		pure fun foo() {
			a = 4
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 35, Line: 4, Column: 3},
			EndPos:   ast.Position{Offset: 39, Line: 4, Column: 7},
		})
	})

	t.Run("external array write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		var a = [3]
		pure fun foo() {
			a[0] = 4
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 4, Column: 4},
			EndPos:   ast.Position{Offset: 44, Line: 4, Column: 10},
		})
	})

	t.Run("internal write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pure fun foo() {
			var a = 3
			a = 4
		}
		`)

		require.NoError(t, err)
	})

	t.Run("internal array write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pure fun foo() {
			var a = [3]
			a[0] = 4
		}
		`)

		require.NoError(t, err)
	})

	t.Run("internal param write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pure fun foo(_ a: [Int]) {
			a[0] = 4
		}
		`)

		require.NoError(t, err)
	})

	t.Run("struct external write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pub struct R {
			pub(set) var x: Int
			init(x: Int) {
				self.x = x
			}
		}
		
		let r = R(x: 0)
		pure fun foo(){
			r.x = 3
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 125, Line: 11, Column: 3},
			EndPos:   ast.Position{Offset: 131, Line: 11, Column: 9},
		})
	})

	t.Run("struct param write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pub struct R {
			pub(set) var x: Int
			init(x: Int) {
				self.x = x
			}
		}
		
		pure fun foo(_ r: R): R {
			r.x = 3
			return r
		}
		`)

		require.NoError(t, err)
	})

	t.Run("struct param nested write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pub struct R {
			pub(set) var x: Int
			init(x: Int) {
				self.x = x
			}
		}
		
		pure fun foo(_ r: R): R {
			if true {
				while true {
					r.x = 3
				}
			}
			return r
		}
		`)

		require.NoError(t, err)
	})

	t.Run("indeterminate write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a: [Int] = []
		pure fun foo() {
			let b: [Int] = []
        	let c = [a, b]
       		c[0][0] = 4
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 98, Line: 6, Column: 13},
			EndPos:   ast.Position{Offset: 104, Line: 6, Column: 19},
		})
	})

	t.Run("indeterminate append", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a: [Int] = []
		pure fun foo() {
			let b: [Int] = []
        	let c = [a, b]
       		c[0].append(4)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 95, Line: 6, Column: 10},
			EndPos:   ast.Position{Offset: 107, Line: 6, Column: 22},
		})
	})

	t.Run("nested write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		fun foo() {
			var a = 3
			let b = pure fun() {
				while true {
					a = 4
				}
			}
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 74, Line: 6, Column: 5},
			EndPos:   ast.Position{Offset: 78, Line: 6, Column: 9},
		})
	})

	t.Run("nested write success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		var a = 3
		pure fun foo() {
			let b = fun() {
				a = 4
			}
		}
		`)

		require.NoError(t, err)
	})

	t.Run("nested scope legal write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pure fun foo() {
			var a = 3
			while true {
				a = 4
			}
		}
		`)

		require.NoError(t, err)
	})

	t.Run("reference write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct S {
			pub(set) var x: Int
			init(x: Int) {
				self.x = x
			}
		}
		
		pure fun foo(_ s: &S) {
			s.x = 3
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 111, Line: 10, Column: 3},
			EndPos:   ast.Position{Offset: 117, Line: 10, Column: 9},
		})
	})

	t.Run("missing variable write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct S {
			pub(set) var x: Int
			init(x: Int) {
				self.x = x
			}
		}

		pure fun foo() {
			z.x = 3
		}
		`)

		errs := ExpectCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.PurityError{}, errs[1])
		assert.Equal(t, errs[1].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 102, Line: 10, Column: 3},
			EndPos:   ast.Position{Offset: 108, Line: 10, Column: 9},
		})
	})
}

func TestCheckResourceWritePurity(t *testing.T) {
	t.Run("resource param write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pub resource R {
			pub(set) var x: Int
			init(x: Int) {
				self.x = x
			}
		}
		
		pure fun foo(_ r: @R): @R {
			r.x = 3
			return <-r
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 121, Line: 10, Column: 3},
			EndPos:   ast.Position{Offset: 127, Line: 10, Column: 9},
		})
	})

	t.Run("destroy", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pub resource R {}
		
		pure fun foo(_ r: @R){
			destroy r
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 52, Line: 5, Column: 3},
			EndPos:   ast.Position{Offset: 60, Line: 5, Column: 11},
		})
	})

	t.Run("resource param nested write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pub resource R {
			pub(set) var x: Int
			init(x: Int) {
				self.x = x
			}
		}
		
		pure fun foo(_ r: @R): @R {
			if true {
				while true {
					r.x = 3
				}
			}
			return <-r
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 153, Line: 12, Column: 5},
			EndPos:   ast.Position{Offset: 159, Line: 12, Column: 11},
		})
	})

	t.Run("internal resource write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pub resource R {
			pub(set) var x: Int
			pure init(x: Int) {
				self.x = x
			}
		}
		
		pure fun foo(): @R {
			let r <- create R(x: 0)
			r.x = 1
			return <-r 
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 146, Line: 11, Column: 3},
			EndPos:   ast.Position{Offset: 152, Line: 11, Column: 9},
		})
	})

	t.Run("external resource move", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pub resource R {
			pub(set) var x: Int
			init(x: Int) {
				self.x = x
			}
		}
		
		pure fun foo(_ f: @R): @R {
			let b <- f 
			b.x = 3
			return <-b
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 136, Line: 11, Column: 3},
			EndPos:   ast.Position{Offset: 142, Line: 11, Column: 9},
		})
	})

	t.Run("resource moves", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		pub resource R {
			pub(set) var x: Int
			init(x: Int) {
				self.x = x
			}
		}
		
		pure fun foo(_ r1: @R, _ r2: @R): @[R] {
			return <-[<-r1, <-r2]
		}
		
		`)

		require.NoError(t, err)
	})
}

func TestCheckCompositeWritePurity(t *testing.T) {
	t.Run("self struct modification", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct S {
			var b: Int

			init(b: Int) {
				self.b = b
			}

			pure fun foo() {
				self.b = 3
			}
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 92, Line: 10, Column: 4},
			EndPos:   ast.Position{Offset: 101, Line: 10, Column: 13},
		})
	})

	t.Run("safe struct modification", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct S {
			var b: Int

			init(b: Int) {
				self.b = b
			}

			pure fun foo(_ s: S) {
				s.b = 3
			}
		}
		`)

		require.NoError(t, err)
	})

	t.Run("struct init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		struct S {
			var b: Int

			pure init(b: Int) {
				self.b = b
			}
		}

		pure fun foo() {
			let s = S(b: 3)
		}
		`)

		require.NoError(t, err)
	})

	t.Run("resource init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		resource R {
			var b: Int 

			pure init(b: Int) {
				self.b = b
			}
		}

		pure fun foo(): @R {
			return <-create R(b: 3)
		}
		`)

		require.NoError(t, err)
	})

	t.Run("impure struct init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [0]
		struct S {
			var b: Int

			pure init(b: Int) {
				a[1] = 4
				self.b = b
			}
		}

		pure fun foo() {
			let s = S(b: 3)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 71, Line: 7, Column: 5},
			EndPos:   ast.Position{Offset: 77, Line: 7, Column: 11},
		})
	})

	t.Run("impure resource init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [0]
		resource R {
			var b: Int 

			pure init(b: Int) {
				a[1] = 4
				self.b = b
			}
		}

		pure fun foo(): @R {
			return <-create R(b: 3)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 74, Line: 7, Column: 5},
			EndPos:   ast.Position{Offset: 80, Line: 7, Column: 11},
		})
	})
}

func TestCheckContainerMethodPurity(t *testing.T) {
	t.Run("array contains", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.contains(0)
		}
		`)

		require.NoError(t, err)
	})

	t.Run("array concat", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.concat([0])
		}
		`)

		require.NoError(t, err)
	})

	t.Run("array firstIndex", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.firstIndex(of: 0)
		}
		`)

		require.NoError(t, err)
	})

	t.Run("array slice", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.slice(from: 0, upTo: 1)
		}
		`)

		require.NoError(t, err)
	})

	t.Run("array append", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.append(0)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array appendAll", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.appendAll([0])
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array insert", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.insert(at:0, 0)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array remove", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.remove(at:0)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array removeFirst", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.removeFirst()
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array removeLast", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = [3]
		pure fun foo() {
			a.removeLast()
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("dict insert", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = {0:0}
		pure fun foo() {
			a.insert(key: 0, 0)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("dict remove", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = {0:0}
		pure fun foo() {
			a.remove(key: 0)
		}
		`)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("dict containsKey", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		let a = {0:0}
		pure fun foo() {
			a.containsKey(0)
		}
		`)

		require.NoError(t, err)
	})
}
