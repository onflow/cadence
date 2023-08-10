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
          view fun foo(x: fun(): Void) {}

          let x: view fun(view fun(): Void): Void = foo
        `)
		require.NoError(t, err)
	})

	t.Run("contravariant error", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          view fun foo(f: view fun(): Void) {}

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
	t.Parallel()

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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 66, Line: 5, Column: 14},
				EndPos:   ast.Position{Offset: 70, Line: 5, Column: 18},
			},
			errs[0].(*sema.PurityError).Range,
		)
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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 109, Line: 7, Column: 14},
				EndPos:   ast.Position{Offset: 115, Line: 7, Column: 20},
			},
			errs[0].(*sema.PurityError).Range,
		)
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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 100, Line: 6, Column: 18},
				EndPos:   ast.Position{Offset: 104, Line: 6, Column: 22},
			},
			errs[0].(*sema.PurityError).Range,
		)
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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 136, Line: 8, Column: 14},
				EndPos:   ast.Position{Offset: 138, Line: 8, Column: 16},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("emit", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event FooEvent()

          view fun foo() {
              emit FooEvent()
          }
        `)
		require.NoError(t, err)
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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 62, Line: 4, Column: 14},
				EndPos:   ast.Position{Offset: 66, Line: 4, Column: 18},
			},
			errs[0].(*sema.PurityError).Range,
		)
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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 66, Line: 5, Column: 15},
				EndPos:   ast.Position{Offset: 72, Line: 5, Column: 21},
			},
			errs[0].(*sema.PurityError).Range,
		)
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
          struct R {
              var x: Int

              fun setX(_ x: Int) {
                  self.x = x
              }

              init(x: Int) {
                  self.x = x
              }
          }

          let r = R(x: 0)
          view fun foo(){
              r.setX(3)
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 282, Line: 16, Column: 14},
				EndPos:   ast.Position{Offset: 290, Line: 16, Column: 22},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("struct param write", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct R {
              var x: Int

              init(x: Int) {
                  self.x = x
              }

              view fun foo(_ r: R): R {
                  r.x = 3
                  return r
              }
          }
        `)
		require.NoError(t, err)
	})

	t.Run("struct param nested write", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct R {
              var x: Int

              init(x: Int) {
                  self.x = x
              }

              view fun foo(_ r: R): R {
                  if true {
                      while true {
                          r.x = 3
                      }
                  }
                  return r
              }
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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 133, Line: 7, Column: 15},
				EndPos:   ast.Position{Offset: 145, Line: 7, Column: 27},
			},
			errs[0].(*sema.PurityError).Range,
		)
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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 135, Line: 6, Column: 22},
				EndPos:   ast.Position{Offset: 139, Line: 6, Column: 26},
			},
			errs[0].(*sema.PurityError).Range,
		)
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
              var x: Int
              init(x: Int) {
                  self.x = x
              }

              view fun foo(_ s: &S) {
                  s.x = 3
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 178, Line: 9, Column: 18},
				EndPos:   ast.Position{Offset: 184, Line: 9, Column: 24},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("reference write, nested", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              var x: Int
              init(_ x: Int) {
                  self.x = x
              }

              view fun foo(_ s: [&S]) {
                  s[0].x = 3
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 183, Line: 9, Column: 19},
				EndPos:   ast.Position{Offset: 191, Line: 9, Column: 27},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("missing variable write", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              var x: Int
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
		require.IsType(t, &sema.PurityError{}, errs[1])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 175, Line: 10, Column: 14},
				EndPos:   ast.Position{Offset: 181, Line: 10, Column: 20},
			},
			errs[1].(*sema.PurityError).Range,
		)
	})

	t.Run("bound function", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              fun f() {}
          }

          view fun foo() {
              let f = S().f
              f()
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 129, Line: 8, Column: 14},
				EndPos:   ast.Position{Offset: 131, Line: 8, Column: 16},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("bound function, view", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              view fun f() {}
          }

          view fun foo() {
              let f = S().f
              f()
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckResourceWritePurity(t *testing.T) {
	t.Parallel()

	t.Run("resource param write", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var x: Int

              init(x: Int) {
                  self.x = x
              }

              view fun foo(_ r: @R): @R {
                  r.x = 3
                  return <-r
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 185, Line: 10, Column: 18},
				EndPos:   ast.Position{Offset: 191, Line: 10, Column: 24},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("destroy", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          view fun foo(_ r: @R){
              destroy r
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 73, Line: 5, Column: 14},
				EndPos:   ast.Position{Offset: 81, Line: 5, Column: 22},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("resource param nested write", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var x: Int

              init(x: Int) {
                  self.x = x
              }

              view fun foo(_ r: @R): @R {
                  if true {
                      while true {
                          r.x = 3
                      }
                  }
                  return <-r
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 256, Line: 12, Column: 26},
				EndPos:   ast.Position{Offset: 262, Line: 12, Column: 32},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("internal resource write", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var x: Int

              view init(x: Int) {
                  self.x = x
              }

              view fun foo(): @R {
                  let r <- create R(x: 0)
                  r.x = 1
                  return <-r
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 225, Line: 11, Column: 18},
				EndPos:   ast.Position{Offset: 231, Line: 11, Column: 24},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("external resource move", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var x: Int

              init(x: Int) {
                  self.x = x
              }

              view fun foo(_ f: @R): @R {
                  let b <- f
                  b.x = 3
                  return <-b
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 214, Line: 11, Column: 18},
				EndPos:   ast.Position{Offset: 220, Line: 11, Column: 24},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("resource array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var x: Int

              init(_ x: Int) {
                  self.x = x
              }

              view fun foo(_ a: @[R], _ x: Int): @[R] {
                  a[x].x = 4
                  return <-a
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 202, Line: 10, Column: 19},
				EndPos:   ast.Position{Offset: 210, Line: 10, Column: 27},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("nested resource array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var x: Int

              init(_ x: Int) {
                  self.x = x
              }

              view fun foo(_ a: @[[R]], _ x: Int): @[[R]] {
                  a[x][x].x = 4
                  return <-a
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 209, Line: 10, Column: 22},
				EndPos:   ast.Position{Offset: 217, Line: 10, Column: 30},
			},
			errs[0].(*sema.PurityError).Range,
		)
	})

	t.Run("resource moves", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var x: Int

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
	t.Parallel()

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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 172, Line: 10, Column: 18},
				EndPos:   ast.Position{Offset: 181, Line: 10, Column: 27},
			},
			errs[0].(*sema.PurityError).Range,
		)
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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(
			t,
			ast.Range{
				StartPos: ast.Position{Offset: 124, Line: 8, Column: 19},
				EndPos:   ast.Position{Offset: 130, Line: 8, Column: 25},
			},
			errs[0].(*sema.PurityError).Range,
		)
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

		require.IsType(t, &sema.PurityError{}, errs[0])
		assert.Equal(t,
			ast.Range{
				StartPos: ast.Position{Offset: 126, Line: 8, Column: 19},
				EndPos:   ast.Position{Offset: 132, Line: 8, Column: 25},
			},
			errs[0].(*sema.PurityError).Range)
	})
}

func TestCheckContainerMethodPurity(t *testing.T) {
	t.Parallel()

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
          let a = {0: 0}

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
          let a = {0: 0}

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
          let a = {0: 0}

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
          view fun foo(): Int {
              return 0
          }

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
          view fun foo(): Int {
              return 0
          }

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
          fun foo(): Int {
              return 0
          }

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

func TestCheckAccountPurity(t *testing.T) {
	t.Parallel()

	t.Run("storage", func(t *testing.T) {

		t.Parallel()

		t.Run("save", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: auth(Storage) &Account.Storage) {
                  storage.save(3, to: /storage/foo)
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 89, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 121, Line: 3, Column: 50},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("type", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: auth(Storage) &Account.Storage) {
                  storage.type(at: /storage/foo)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("load", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: auth(Storage) &Account.Storage) {
                  storage.load<Int>(from: /storage/foo)
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 89, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 125, Line: 3, Column: 54},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("copy", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: auth(Storage) &Account.Storage) {
                  storage.copy<Int>(from: /storage/foo)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("borrow", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: auth(Storage) &Account.Storage) {
                  storage.borrow<&Int>(from: /storage/foo)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("check", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: &Account.Storage) {
                  storage.check<Int>(from: /storage/foo)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("forEachPublic", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: &Account.Storage) {
                  storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                      return true
                  })
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 75, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 193, Line: 5, Column: 19},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("forEachStored", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: &Account.Storage) {
                  storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                      return true
                  })
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 75, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 194, Line: 5, Column: 19},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})
	})

	t.Run("contracts", func(t *testing.T) {
		t.Parallel()

		t.Run("add", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(contracts: auth(Contracts) &Account.Contracts) {
                  contracts.add(name: "", code: [])
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 95, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 127, Line: 3, Column: 50},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("update", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(contracts: auth(Contracts) &Account.Contracts) {
                  contracts.update(name: "", code: [])
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 95, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 130, Line: 3, Column: 53},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("get", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(contracts: &Account.Contracts) {
                  contracts.get(name: "")
              }
            `)

			require.NoError(t, err)
		})

		t.Run("remove", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(contracts: auth(Contracts) &Account.Contracts) {
                  contracts.remove(name: "")
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 95, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 120, Line: 3, Column: 43},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("borrow", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(contracts: &Account.Contracts) {
                  contracts.borrow<&Int>(name: "")
              }
            `)
			require.NoError(t, err)
		})
	})

	t.Run("keys", func(t *testing.T) {
		t.Parallel()

		t.Run("add", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(keys: auth(Keys) &Account.Keys) {
                  keys.add(
                      publicKey: key,
                      hashAlgorithm: algo,
                      weight: 100.0
                  )
              }
            `)

			errs := RequireCheckerErrors(t, err, 3)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 80, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 225, Line: 7, Column: 18},
				},
				errs[0].(*sema.PurityError).Range,
			)
			assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
			assert.IsType(t, &sema.NotDeclaredError{}, errs[2])
		})

		t.Run("get", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(keys: &Account.Keys) {
                  keys.get(keyIndex: 0)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("revoke", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(keys: auth(Keys) &Account.Keys) {
                  keys.revoke(keyIndex: 0)
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 80, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 103, Line: 3, Column: 41},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("forEach", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(keys: &Account.Keys) {
                  keys.forEach(fun(key: AccountKey): Bool {
                      return true
                  })
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 69, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 164, Line: 5, Column: 19},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})
	})

	t.Run("inbox", func(t *testing.T) {
		t.Parallel()

		t.Run("publish", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(inbox: auth(Inbox) &Account.Inbox) {
                  inbox.publish(
                      cap,
                      name: "cap",
                      recipient: 0x1
                  )
              }
            `)

			errs := RequireCheckerErrors(t, err, 2)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 83, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 215, Line: 7, Column: 18},
				},
				errs[0].(*sema.PurityError).Range,
			)
			assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
		})

		t.Run("unpublish", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(inbox: auth(Inbox) &Account.Inbox) {
                  inbox.unpublish<&Int>("cap")
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 83, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 110, Line: 3, Column: 45},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("claim", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(inbox: auth(Inbox) &Account.Inbox) {
                  inbox.claim<&Int>("cap", provider: 0x1)
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 83, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 121, Line: 3, Column: 56},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})
	})

	t.Run("capabilities", func(t *testing.T) {
		t.Parallel()

		t.Run("get", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(capabilities: &Account.Capabilities) {
                  capabilities.get<&Int>(/public/foo)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("borrow", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(capabilities: &Account.Capabilities) {
                  capabilities.borrow<&Int>(/public/foo)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("publish", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(capabilities: auth(Capabilities) &Account.Capabilities) {
                  capabilities.publish(
                      cap,
                      at: /public/foo
                  )
              }
            `)

			errs := RequireCheckerErrors(t, err, 2)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 104, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 209, Line: 6, Column: 18},
				},
				errs[0].(*sema.PurityError).Range,
			)
			assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
		})

		t.Run("unpublish", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(capabilities: auth(Capabilities) &Account.Capabilities) {
                  capabilities.unpublish(/public/foo)
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 104, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 138, Line: 3, Column: 52},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})
	})

	t.Run("capabilities.storage", func(t *testing.T) {
		t.Parallel()

		t.Run("getController", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: auth(StorageCapabilities) &Account.StorageCapabilities) {
                  storage.getController(byCapabilityID: 1)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("getControllers", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: auth(StorageCapabilities) &Account.StorageCapabilities) {
                  storage.getControllers(forPath: /storage/foo)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("forEachController", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: auth(StorageCapabilities) &Account.StorageCapabilities) {
                  storage.forEachController(
                      forPath: /storage/foo,
                      fun (controller: &StorageCapabilityController): Bool {
                          return true
                      }
                  )
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 113, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 342, Line: 8, Column: 18},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("issue", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(storage: auth(StorageCapabilities) &Account.StorageCapabilities) {
                  storage.issue<&Int>(/storage/foo)
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 113, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 145, Line: 3, Column: 50},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})
	})

	t.Run("capabilities.account", func(t *testing.T) {
		t.Parallel()

		t.Run("getController", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(account: auth(AccountCapabilities) &Account.AccountCapabilities) {
                  account.getController(byCapabilityID: 1)
              }
            `)
			require.NoError(t, err)
		})

		t.Run("getControllers", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(account: auth(AccountCapabilities) &Account.AccountCapabilities) {
                  account.getControllers()
              }
            `)
			require.NoError(t, err)
		})

		t.Run("forEachController", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(account: auth(AccountCapabilities) &Account.AccountCapabilities) {
                  account.forEachController(
                      fun (controller: &AccountCapabilityController): Bool {
                          return true
                      }
                  )
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 113, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 297, Line: 7, Column: 18},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})

		t.Run("issue", func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, `
              view fun foo(account: auth(AccountCapabilities) &Account.AccountCapabilities) {
                  account.issue<&Account>()
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.PurityError{}, errs[0])
			assert.Equal(
				t,
				ast.Range{
					StartPos: ast.Position{Offset: 113, Line: 3, Column: 18},
					EndPos:   ast.Position{Offset: 137, Line: 3, Column: 42},
				},
				errs[0].(*sema.PurityError).Range,
			)
		})
	})
}
