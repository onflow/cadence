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

	t.Run("view <: impure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        view fun foo() {}
        let x: fun(): Void = foo
        `)

		require.NoError(t, err)
	})

	t.Run("view <: view", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        view fun foo() {}
        let x: view fun(): Void = foo
        `)

		require.NoError(t, err)
	})

	t.Run("impure <: impure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        fun foo() {}
        let x: fun(): Void = foo
        `)

		require.NoError(t, err)
	})

	t.Run("impure <: view", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        fun foo() {}
        let x: view fun(): Void = foo
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("contravariant ok", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        view fun foo(x:fun(): Void) {}
        let x: view fun(view fun(): Void): Void = foo
        `)

		require.NoError(t, err)
	})

	t.Run("contravariant error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        view fun foo(f:view fun(): Void) {}
        let x: view fun(fun(): Void): Void = foo
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("interface implementation member success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        struct interface I {
            view fun foo()
            fun bar()
        }

        struct S: I {
            view fun foo() {}
            view fun bar() {}
        }
        `)

		require.NoError(t, err)
	})

	t.Run("interface implementation member failure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        struct interface I {
            view fun foo()
            fun bar() 
        }

        struct S: I {
            fun foo() {}
            fun bar() {}
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("interface implementation initializer success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        struct interface I {
            view init()
        }

        struct S: I {
            view init() {}
        }
        `)

		require.NoError(t, err)
	})

	t.Run("interface implementation initializer explicit success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        struct interface I {
            view init()
        }

        struct S: I {
            init() {}
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("interface implementation initializer success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        struct interface I {
            init()
        }

        struct S: I {
            view init() {}
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

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
	t.Run("view function call", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        view fun bar() {}
        view fun foo() {
            bar()
        }
        `)

		require.NoError(t, err)
	})

	t.Run("impure function call error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        fun bar() {}
        view fun foo() {
            bar()
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 59, Line: 4, Column: 12},
			EndPos:   ast.Position{Offset: 63, Line: 4, Column: 16},
		})
	})

	t.Run("impure method call error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        struct S {
            fun bar() {}
        }
        view fun foo(_ s: S) {
            s.bar()
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 98, Line: 6, Column: 12},
			EndPos:   ast.Position{Offset: 104, Line: 6, Column: 18},
		})
	})

	t.Run("view function call nested", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        fun bar() {}
        view fun foo() {
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
            let f = view fun() {
                bar()
            }
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 91, Line: 5, Column: 16},
			EndPos:   ast.Position{Offset: 95, Line: 5, Column: 20},
		})
	})

	t.Run("view function call nested failure", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        fun bar() {}
        view fun foo() {
            let f = fun() {
                bar()
            }
            f()
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 123, Line: 7, Column: 12},
			EndPos:   ast.Position{Offset: 125, Line: 7, Column: 14},
		})
	})

	t.Run("save", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            authAccount.save(3, to: /storage/foo)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 3, Column: 12},
			EndPos:   ast.Position{Offset: 74, Line: 3, Column: 48},
		})
	})

	t.Run("load", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            authAccount.load<Int>(from: /storage/foo)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 3, Column: 12},
			EndPos:   ast.Position{Offset: 78, Line: 3, Column: 52},
		})
	})

	t.Run("type", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            authAccount.type(at: /storage/foo)
        }
        `)

		require.NoError(t, err)
	})

	t.Run("link", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            authAccount.link<&Int>(/private/foo, target: /storage/foo)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 3, Column: 12},
			EndPos:   ast.Position{Offset: 95, Line: 3, Column: 69},
		})
	})

	t.Run("unlink", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            authAccount.unlink(/private/foo)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 3, Column: 12},
			EndPos:   ast.Position{Offset: 69, Line: 3, Column: 43},
		})
	})

	t.Run("add contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            authAccount.contracts.add(name: "", code: [])
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 3, Column: 12},
			EndPos:   ast.Position{Offset: 82, Line: 3, Column: 56},
		})
	})

	t.Run("update contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            authAccount.contracts.update__experimental(name: "", code: [])
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 3, Column: 12},
			EndPos:   ast.Position{Offset: 99, Line: 3, Column: 73},
		})
	})

	t.Run("remove contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            authAccount.contracts.remove(name: "")
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 3, Column: 12},
			EndPos:   ast.Position{Offset: 75, Line: 3, Column: 49},
		})
	})

	t.Run("revoke key", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            authAccount.keys.revoke(keyIndex: 0)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 38, Line: 3, Column: 12},
			EndPos:   ast.Position{Offset: 73, Line: 3, Column: 47},
		})
	})

	t.Run("alias", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        view fun foo() {
            let f = authAccount.contracts.remove
            f(name: "")
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 87, Line: 4, Column: 12},
			EndPos:   ast.Position{Offset: 97, Line: 4, Column: 22},
		})
	})

	t.Run("emit", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
        event FooEvent()
        view fun foo() {
            emit FooEvent()
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 68, Line: 4, Column: 17},
			EndPos:   ast.Position{Offset: 77, Line: 4, Column: 26},
		})
	})

	t.Run("external write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        var a = 3
        view fun foo() {
            a = 4
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 56, Line: 4, Column: 12},
			EndPos:   ast.Position{Offset: 60, Line: 4, Column: 16},
		})
	})

	t.Run("external array write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        var a = [3]
        view fun foo() {
            a[0] = 4
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 59, Line: 4, Column: 13},
			EndPos:   ast.Position{Offset: 65, Line: 4, Column: 19},
		})
	})

	t.Run("internal write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        view fun foo() {
            var a = 3
            a = 4
        }
        `)

		require.NoError(t, err)
	})

	t.Run("internal array write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        view fun foo() {
            var a = [3]
            a[0] = 4
        }
        `)

		require.NoError(t, err)
	})

	t.Run("internal param write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        view fun foo(_ a: [Int]) {
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
        view fun foo(){
            r.x = 3
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 203, Line: 11, Column: 12},
			EndPos:   ast.Position{Offset: 209, Line: 11, Column: 18},
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
        
        view fun foo(_ r: R): R {
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
        
        view fun foo(_ r: R): R {
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
        view fun foo() {
            let b: [Int] = []
            let c = [a, b]
            c[0][0] = 4
        }
        `)

		require.NoError(t, err)
	})

	t.Run("indeterminate append", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a: [Int] = []
        view fun foo() {
            let b: [Int] = []
            let c = [a, b]
               c[0].append(4)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 125, Line: 6, Column: 16},
			EndPos:   ast.Position{Offset: 137, Line: 6, Column: 28},
		})
	})

	t.Run("nested write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        fun foo() {
            var a = 3
            let b = view fun() {
                while true {
                    a = 4
                }
            }
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 125, Line: 6, Column: 20},
			EndPos:   ast.Position{Offset: 129, Line: 6, Column: 24},
		})
	})

	t.Run("nested write success", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        var a = 3
        view fun foo() {
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
        view fun foo() {
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
        
        view fun foo(_ s: &S) {
            s.x = 3
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 183, Line: 10, Column: 12},
			EndPos:   ast.Position{Offset: 189, Line: 10, Column: 18},
		})
	})

	t.Run("reference write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        struct S {
            pub(set) var x: Int
            init(_ x: Int) {
                self.x = x
            }
        }

		let s = [&S(0) as &S]
        
        view fun foo() {
            s[0].x = 3
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 204, Line: 12, Column: 13},
			EndPos:   ast.Position{Offset: 212, Line: 12, Column: 21},
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

        view fun foo() {
            z.x = 3
        }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.PurityError{}, errs[1])
		assert.Equal(t, errs[1].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 168, Line: 10, Column: 12},
			EndPos:   ast.Position{Offset: 174, Line: 10, Column: 18},
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

            view fun foo(_ r: @R): @R {
                r.x = 3
                return <-r
            }
            `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 217, Line: 10, Column: 16},
			EndPos:   ast.Position{Offset: 223, Line: 10, Column: 22},
		})
	})

	t.Run("destroy", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            pub resource R {}

            view fun foo(_ r: @R){
                destroy r
            }
            `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 83, Line: 5, Column: 16},
			EndPos:   ast.Position{Offset: 91, Line: 5, Column: 24},
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

            view fun foo(_ r: @R): @R {
                if true {
                    while true {
                        r.x = 3
                    }
                }
                return <-r
            }
            `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 284, Line: 12, Column: 24},
			EndPos:   ast.Position{Offset: 290, Line: 12, Column: 30},
		})
	})

	t.Run("internal resource write", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            pub resource R {
                pub(set) var x: Int
                view init(x: Int) {
                    self.x = x
                }
            }

            view fun foo(): @R {
                let r <- create R(x: 0)
                r.x = 1
                return <-r
            }
            `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 255, Line: 11, Column: 16},
			EndPos:   ast.Position{Offset: 261, Line: 11, Column: 22},
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

            view fun foo(_ f: @R): @R {
                let b <- f
                b.x = 3
                return <-b
            }
            `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 244, Line: 11, Column: 16},
			EndPos:   ast.Position{Offset: 250, Line: 11, Column: 22},
		})
	})

	t.Run("resource array", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        resource R {
            pub(set) var x: Int
            init(_ x: Int) {
                self.x = x
            }
        }
        
        view fun foo(_ a: @[R], _ x: Int): @[R] {
            a[x].x = 4
            return <-a
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 206, Line: 10, Column: 13},
			EndPos:   ast.Position{Offset: 214, Line: 10, Column: 21},
		})
	})

	t.Run("nested resource array", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        resource R {
            pub(set) var x: Int
            init(_ x: Int) {
                self.x = x
            }
        }
        
        view fun foo(_ a: @[[R]], _ x: Int): @[[R]] {
            a[x][x].x = 4
            return <-a
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 213, Line: 10, Column: 16},
			EndPos:   ast.Position{Offset: 221, Line: 10, Column: 24},
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

            view fun foo(_ r1: @R, _ r2: @R): @[R] {
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

            view fun foo() {
                self.b = 3
            }
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 158, Line: 10, Column: 16},
			EndPos:   ast.Position{Offset: 167, Line: 10, Column: 25},
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

            view fun foo(_ s: S) {
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

            view init(b: Int) {
                self.b = b
            }
        }

        view fun foo() {
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

            view init(b: Int) {
                self.b = b
            }
        }

        view fun foo(): @R {
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

            view init(b: Int) {
                a[1] = 4
                self.b = b
            }
        }

        view fun foo() {
            let s = S(b: 3)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 113, Line: 7, Column: 17},
			EndPos:   ast.Position{Offset: 119, Line: 7, Column: 23},
		})
	})

	t.Run("impure resource init", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [0]
        resource R {
            var b: Int 

            view init(b: Int) {
                a[1] = 4
                self.b = b
            }
        }

        view fun foo(): @R {
            return <-create R(b: 3)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t, errs[0].(*sema.PurityError).Range, ast.Range{
			StartPos: ast.Position{Offset: 116, Line: 7, Column: 17},
			EndPos:   ast.Position{Offset: 122, Line: 7, Column: 23},
		})
	})
}

func TestCheckContainerMethodPurity(t *testing.T) {
	t.Run("array contains", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.contains(0)
        }
        `)

		require.NoError(t, err)
	})

	t.Run("array concat", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.concat([0])
        }
        `)

		require.NoError(t, err)
	})

	t.Run("array firstIndex", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.firstIndex(of: 0)
        }
        `)

		require.NoError(t, err)
	})

	t.Run("array slice", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.slice(from: 0, upTo: 1)
        }
        `)

		require.NoError(t, err)
	})

	t.Run("array append", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.append(0)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array appendAll", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.appendAll([0])
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array insert", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.insert(at:0, 0)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array remove", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.remove(at:0)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array removeFirst", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.removeFirst()
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("array removeLast", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = [3]
        view fun foo() {
            a.removeLast()
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("dict insert", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = {0:0}
        view fun foo() {
            a.insert(key: 0, 0)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("dict remove", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = {0:0}
        view fun foo() {
            a.remove(key: 0)
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("dict containsKey", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
        let a = {0:0}
        view fun foo() {
            a.containsKey(0)
        }
        `)

		require.NoError(t, err)
	})
}

func TestCheckConditionPurity(t *testing.T) {
	t.Parallel()

	t.Run("view pre", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		view fun foo(): Int { return 0 }
		fun bar() {
			pre {
				foo() > 3: "bar"
			}
		}
		`)

		require.NoError(t, err)
	})

	t.Run("view post", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		view fun foo(): Int { return 0 }
		fun bar() {
			post {
				foo() > 3: "bar"
			}
		}
		`)

		require.NoError(t, err)
	})

	t.Run("impure post", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		fun foo(): Int { return 0 }
		fun bar() {
			post {
				foo() > 3: "bar"
			}
		}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})

	t.Run("impure pre", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
		fun foo(): Int { return 0 }
		fun bar() {
			pre {
				foo() > 3: "bar"
			}
		}
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.PurityError{}, errs[0])
	})
}
