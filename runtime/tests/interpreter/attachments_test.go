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

package interpreter_test

import (
	"testing"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"

	. "github.com/onflow/cadence/runtime/tests/utils"
	"github.com/stretchr/testify/require"
)

func TestInterpretAttachmentStruct(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        struct S {}
        attachment A for S {
            fun foo(): Int { return 3 }
        }
        fun test(): Int {
            var s = S()
            s = attach A() to s
            return s[A]?.foo()!
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("duplicate attach", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        struct S {}
        attachment A for S {
            fun foo(): Int { return 3 }
        }
        fun test(): Int {
            var s = S()
            s = attach A() to s
            remove A from s
            s = attach A() to s
            return s[A]?.foo()!
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("attach and remove", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        struct S {}
        attachment A for S {
            fun foo(): Int { return 3 }
        }
        fun test(): Int {
            var s = S()
            s = attach A() to s
            s = attach A() to s
            return s[A]?.foo()!
        }
    `)

		_, err := inter.Invoke("test")
		require.ErrorAs(t, err, &interpreter.DuplicateAttachmentError{})
	})

	t.Run("reference", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        struct S {}
        attachment A for S {
            fun foo(): Int { return 3 }
        }
        fun test(): Int {
            var s = S()
            s = attach A() to s
            let ref = &s as &S
            return ref[A]?.foo()!
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("removed", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        struct S {}
        attachment A for S {
            fun foo(): Int { return 3 }
        }
        fun test(): Int? {
            var s = S()
            s = attach A() to s
            remove A from s
            return s[A]?.foo()
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	t.Run("not removed", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        struct S {}
        attachment B for S {}
        attachment A for S {
            fun foo(): Int { return 3 }
        }
        fun test(): Int {
            var s = S()
            s = attach A() to s
            remove B from s
            return s[A]?.foo()!
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("missing", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        struct S {}
        attachment A for S {
            fun foo(): Int { return 3 }
        }
        fun test(): Int? {
            var s = S()
            return s[A]?.foo()
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	t.Run("iteration", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        struct S {
            let i: Int
            init(i: Int) {
                self.i = i
            }
        }
        attachment A for S {
            fun foo(): Int { return base.i }
        }
        fun test(): Int {
            let arr: [S] = []
            var i = 0
            while i < 10 {
                arr.append(S(i: i))
                arr[i] = attach A() to arr[i]
                i = i + 1
            }
            var ret = 0 
            for s in arr {
                ret = ret + s[A]!.foo()
            }
            return ret
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(45), value)
	})

	t.Run("attachment does not mutate original", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        struct S {}
        attachment A for S {
            fun foo(): Int { return 3 }
        }
        fun test(): Int? {
            var s = S()
            var s2 = attach A() to s
            return s[A]?.foo()
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})
}

func TestInterpretAttachmentResource(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {}
       attachment A for R {
          fun foo(): Int { return 3 }
       }
       fun test(): Int {
           let r <- create R()
           let r2 <- attach A() to <-r
           let i = r2[A]?.foo()!
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("duplicate attach", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {}
       attachment A for R {
          fun foo(): Int { return 3 }
       }
       fun test(): Int {
           let r <- create R()
           let r2 <- attach A() to <-r
           let r3 <- attach A() to <-r2
           let i = r3[A]?.foo()!
           destroy r3
           return i
       }
    `)

		_, err := inter.Invoke("test")
		require.ErrorAs(t, err, &interpreter.DuplicateAttachmentError{})
	})

	t.Run("attach and remove", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {}
       attachment A for R {
          let x: Int
          init(x: Int) {
            self.x = x
          }
          fun foo(): Int { return self.x }
       }
       fun test(): Int {
           let r <- create R()
           let r2 <- attach A(x: 4) to <-r
           remove A from r2
           let r3 <- attach A(x: 3) to <-r2
           let i = r3[A]?.foo()!
           destroy r3
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("reference", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {}
       attachment A for R {
          fun foo(): Int { return 3 }
       }
       fun test(): Int {
           let r <- create R()
           let r2 <- attach A() to <-r
           let ref = &r2 as &R
           let i = ref[A]?.foo()!
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("removed", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {}
       attachment A for R {
          fun foo(): Int { return 3 }
       }
       fun test(): Int? {
           let r <- create R()
           let r2 <- attach A() to <-r
           remove A from r2
           let i = r2[A]?.foo()
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	t.Run("not removed", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {}
       attachment B for R {}
       attachment A for R {
          fun foo(): Int { return 3 }
       }
       fun test(): Int {
           let r <- create R()
           let r2 <- attach A() to <-r
           remove B from r2
           let i = r2[A]?.foo()!
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("missing", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {}
       attachment A for R {
          fun foo(): Int { return 3 }
       }
       fun test(): Int? {
           let r <- create R()
           let i = r[A]?.foo()
           destroy r
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	t.Run("iteration", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        resource R {
            let i: Int
            init(i: Int) {
                self.i = i
            }
        }
        attachment A for R {
            fun foo(): Int { return base.i }
        }
        fun test(): Int {
            let arr: @[R] <- []
            var i = 0
            while i < 10 {
                arr.append(<-create R(i: i))
                arr.insert(at: i, <-attach A() to <-arr.remove(at: i))
                i = i + 1
            }
            i = 0
            var ret = 0 
            while i < 10 {
                ret = ret + (&arr[i] as &R)[A]!.foo()
                i = i + 1
            }
            destroy arr
            return ret
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(45), value)
	})

	t.Run("attachment does not mutate original, but original is moved", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})
}

func TestAttachExecutionOrdering(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {
                let x: Int
                fun foo(): Int { return self.x }
                init() { self.x = base[B]!.x }
            }
            attachment B for S {
                let x: Int 
                init() {
                    self.x = 3
                }
            }
            fun test(): Int {
                var s = S()
                var s2 = attach A() to attach B() to s
                return s2[A]?.foo()!
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		// base must already have `B` attached to it during A's initializer
		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("self already attached", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {
                fun bar(): Int? {
                    return self[A]?.bar()
                }
            }
            attachment A for S {
                let x: Int?
                fun foo(): Int? { return self.x }
                fun bar(): Int { return 3 }
                init() { self.x = base.bar() }
            }
            fun test(): Int? {
                var s = S()
                var s2 = attach A() to s
                return s2[A]?.foo()!
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		// base does not yet have `A` attached to it during A's initializer
		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})
}

func TestInterpretAttachmentNestedBaseUse(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        resource R {
            let x: Int
            init (x: Int) {
                self.x = x
            }
        }
        attachment A for R {
            let y: Int 
            init (y: Int) {
                self.y = y
            }
            fun foo(): Int { 
                let r <- create R(x: 10)
                let r2 <- attach A(y: base.x) to <-r
                let i = self.y + r2[A]?.y!
                destroy r2
                return i
            }
        }
        fun test(): Int {
            let r <- create R(x: 3)
            let r2 <- attach A(y: 2) to <-r
            let i = r2[A]?.foo()!
            destroy r2
            return i
        }
        `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(5), value)
}

func TestInterpretNestedAttach(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        resource X {
            let i: Int 
            init() {
                self.i = 3
            }
        }
        resource Y {
            let i: Int 
            init() {
                self.i = 5
            }
        }
        attachment A for X {
            let y: @Y
            let i: Int
            init(_ y: @Y) {
                self.y <- y
                self.i = base.i
            }
            destroy() {
                destroy self.y
            }
        }
        attachment B for Y { }
        fun test(): Int {
            let v <- attach A(<- attach B() to <- create Y()) to <- create X()
            let i = v[A]!.i
            destroy v
            return i
        }
        `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
}

func TestInterpretNestedAttachFunction(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        resource X {
            let i: Int 
            init() {
                self.i = 3
            }
        }
        resource Y {
            let i: Int 
            init() {
                self.i = 5
            }
        }
        attachment A for X {
            let y: @Y
            let i: Int
            init(_ y: @Y) {
                self.y <- y
                self.i = base.i
            }
            destroy() {
                destroy self.y
            }
        }
        attachment B for Y { }
        fun foo(): @Y {
            return <- attach B() to <- create Y()
        }

        fun test(): Int {
            let v <- attach A(<-foo()) to <- create X()
            let i = v[A]!.i
            destroy v
            return i
        }
        `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
}

func TestInterpretAttachmentBaseUse(t *testing.T) {
	t.Parallel()

	t.Run("basic use", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {
          let x: Int
          init (x: Int) {
            self.x = x
          }
       }
       attachment A for R {
          fun foo(): Int { return base.x }
       }
       fun test(): Int {
           let r <- create R(x: 3)
           let r2 <- attach A() to <-r
           let i = r2[A]?.foo()!
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("basic use", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {
          let x: Int
          init (x: Int) {
            self.x = x
          }
       }
       attachment A for R {
          let x: Int 
          init () {
            self.x = base.x
          }
          fun foo(): Int { return self.x }
       }
       fun test(): Int {
           let r <- create R(x: 3)
           let r2 <- attach A() to <-r
           let i = r2[A]?.foo()!
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {
          let x: Int
          init (x: Int) {
            self.x = x
          }
       }
       attachment A for R {
          let x: Int 
          init () {
            self.x = base.x
          }
          fun foo(): Int { return self.x }
       }
       attachment B for R {
            let x: Int 
            init () {
                let r <- create R(x: 4)
                let r2 <- attach A() to <-r
                self.x = base.x + r2[A]?.foo()!
                destroy r2
            }
            fun foo(): Int { return self.x }
        }
        fun test(): Int {
           let r <- create R(x: 3)
           let r2 <- attach B() to <-r
           let i = r2[B]?.foo()!
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(7), value)
	})

	t.Run("access other attachments", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {
          let x: Int
          init (x: Int) {
            self.x = x
          }
       }
       attachment A for R {
          fun foo(): Int { return base.x }
       }
       attachment B for R {
        let x: Int 
        init() {
            self.x = base[A]?.foo()! + base.x
        }
        fun foo(): Int { return self.x }
     }
       fun test(): Int {
           let r <- create R(x: 3)
           let r2 <- attach B() to <- attach A() to  <-r
           let i = r2[B]?.foo()!
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(6), value)
	})

	t.Run("store in field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource interface I {
                fun foo(): Int
            }
            resource R: I {
                fun foo(): Int {
                    return 3
                }
            }
            attachment A for I {
                let base: &{I} 
                init() {
                    self.base = base
                }
            }
            fun test(): Int {
                let r <- attach A() to <-create R()
                let ref = &r as &{I}
                let i = ref[A]!.base.foo()
                destroy r
                return i
            }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("return from function", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource interface I {
                fun foo(): Int
            }
            resource R: I {
                fun foo(): Int {
                    return 3
                }
            }
            attachment A for I {
                fun base(): &{I} {
                    return base
                }
            }
            fun test(): Int {
                let r <- attach A() to <-create R()
                let ref = &r as &{I}
                let i = ref[A]!.base().foo()
                destroy r
                return i
            }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("pass as argument", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource interface I {
                fun foo(): Int
            }
            resource R: I {
                fun foo(): Int {
                    return 3
                }
            }
            attachment A for I {
                fun foo(): Int {
                    return bar(base)
                }
            }
            fun bar(_ ref: &{I}): Int {
                return ref.foo()
            }
            fun test(): Int {
                let r <- attach A() to <-create R()
                let ref = &r as &{I}
                let i = ref[A]!.foo()
                destroy r
                return i
            }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})
}

func TestInterpretAttachmentSelfUse(t *testing.T) {
	t.Parallel()

	t.Run("basic use", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R { }
       attachment A for R {
          let y: Int
          init (y: Int) {
            self.y = y
          }
          fun foo(): Int { return self.y }
       }
       fun test(): Int {
           let r <- create R()
           let r2 <- attach A(y: 4) to <-r
           let i = r2[A]?.foo()!
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(4), value)
	})

	t.Run("with base", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
       resource R {
          let x: Int
          init (x: Int) {
            self.x = x
          }
       }
       attachment A for R {
          let y: Int
          init (y: Int) {
            self.y = y
          }
          fun foo(): Int { return self.y + base.x }
       }
       fun test(): Int {
           let r <- create R(x: 3)
           let r2 <- attach A(y: 4) to <-r
           let i = r2[A]?.foo()!
           destroy r2
           return i
       }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(7), value)
	})

	t.Run("store in field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R { }
            attachment A for R {
                let self: &A
                init() {
                    self.self = self
                }
                fun foo(): Int {
                    return 3
                }
            }
            fun test(): Int {
                let r <- attach A() to <-create R()
                let i = r[A]!.self.foo()
                destroy r
                return i
            }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("return from function", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R { }
            attachment A for R {
                fun self(): &A {
                    return self
                }
                fun foo(): Int {
                    return 3
                }
            }
            fun test(): Int {
                let r <- attach A() to <-create R()
                let i = r[A]!.self().foo()
                destroy r
                return i
            }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("pass as argument", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R { }
            attachment A for R {
                fun selfFn(): Int {
                    return bar(self)
                }
                fun foo(): Int {
                    return 3
                }
            }
            fun bar(_ a: &A): Int {
                return a.foo()
            }
            fun test(): Int {
                let r <- attach A() to <-create R()
                let i = r[A]!.selfFn()
                destroy r
                return i
            }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})
}

func TestInterpretAttachmentNameConflict(t *testing.T) {
	t.Parallel()

	t.Run("base field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                let A: Int
                init (a: Int) {
                    self.A = a
                }
            }
            attachment A for R {
                let base: Int
                fun foo(): Int { return self.base + base.A }
                init(b: Int) {
                    self.base = b
                }
            }
            fun test(): Int {
                let r <- create R(a: 3)
                let r2 <- attach A(b: 3) to <-r
                let i = r2[A]?.foo()!
                destroy r2
                return i
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(6), value)
	})
}

func TestInterpretAttachmentRestrictedType(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        resource interface I {
            fun foo(): Int
        }
        resource R: I {
            fun foo(): Int {
                return 3
            }
        }
        attachment A for I {
            fun foo(): Int {
                return base.foo()
            }
        }
        fun test(): Int {
            let r <- attach A() to <-create R()
            let ref = &r as &{I}
            let i = ref[A]!.foo()
            destroy r
            return i
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("constructor", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        resource interface I {
            fun foo(): Int
        }
        resource R: I {
            fun foo(): Int {
                return 3
            }
        }
        attachment A for I {
            let x: Int 
            init() {
                self.x = base.foo()
            }
        }
        fun test(): Int {
            let r <- attach A() to <-create R()
            let ref = &r as &{I}
            let i = ref[A]!.x
            destroy r
            return i
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("constructor on restricted", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        resource interface I {
            fun foo(): Int
        }
        resource R: I {
            fun foo(): Int {
                return 3
            }
        }
        attachment A for I {
            let x: Int 
            init() {
                self.x = base.foo()
            }
        }
        fun test(): Int {
            let r: @{I} <- create R()
            let withAttachment <- attach A() to <-r
            let ref = &withAttachment as &{I}
            let i = ref[A]!.x
            destroy withAttachment
            return i
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("base", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        resource interface I {
            fun foo(): Int
        }
        resource R: I {
            fun foo(): Int {
                return 3
            }
        }
        attachment A for I {
            fun foo(): Int {
                return base.foo()
            }
            fun getBaseFoo(): Int {
                return base[A]!.foo()
            }
        }
        fun test(): Int {
            let r <- attach A() to <-create R()
            let ref = &r as &{I}
            let i = ref[A]!.getBaseFoo()
            destroy r
            return i
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

}

func TestInterpretAttachmentStorage(t *testing.T) {
	t.Parallel()

	t.Run("save and load", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
            resource R {}
            attachment A for R {
                fun foo(): Int { return 3 }
            }
            fun test(): Int {
                let r <- create R()
                let r2 <- attach A() to <-r
                authAccount.save(<-r2, to: /storage/foo)
                let r3 <- authAccount.load<@R>(from: /storage/foo)!
                let i = r3[A]?.foo()!
                destroy r3
                return i
            }
        `, sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("save and borrow", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
            resource R {}
            attachment A for R {
                fun foo(): Int { return 3 }
            }
            fun test(): Int {
                let r <- create R()
                let r2 <- attach A() to <-r
                authAccount.save(<-r2, to: /storage/foo)
                let r3 = authAccount.borrow<&R>(from: /storage/foo)!
                let i = r3[A]?.foo()!
                return i
            }
        `, sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("capability", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
            resource R {}
            attachment A for R {
                fun foo(): Int { return 3 }
            }
            fun test(): Int {
                let r <- create R()
                let r2 <- attach A() to <-r
                authAccount.save(<-r2, to: /storage/foo)
                authAccount.link<&R>(/public/foo, target: /storage/foo)
                let cap = pubAccount.getCapability<&R>(/public/foo)!
                let i = cap.borrow()![A]?.foo()!
                return i
            }
        `, sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("capability interface", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
            resource R: I {}
            resource interface I {}
            attachment A for I {
                fun foo(): Int { return 3 }
            }
            fun test(): Int {
                let r <- create R()
                let r2 <- attach A() to <-r
                authAccount.save(<-r2, to: /storage/foo)
                authAccount.link<&R{I}>(/public/foo, target: /storage/foo)
                let cap = pubAccount.getCapability<&R{I}>(/public/foo)!
                let i = cap.borrow()![A]?.foo()!
                return i
            }
        `, sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})
}

func TestInterpretAttachmentDestructor(t *testing.T) {

	t.Parallel()

	t.Run("destructor run on base destroy", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            var destructorRun = false
            resource R {}
            attachment A for R {
                destroy() {
                    destructorRun = true
                }
            }
            fun test() {
                let r <- create R()
                let r2 <- attach A() to <-r
                destroy r2
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.TrueValue, inter.Globals.Get("destructorRun").GetValue())
	})

	t.Run("base destructor executed last", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            var lastDestructorRun = ""
            resource R {
                destroy() {
                    lastDestructorRun = "R"
                }
            }
            attachment A for R {
                destroy() {
                    lastDestructorRun = "A"
                }
            }
            attachment B for R {
                destroy() {
                    lastDestructorRun = "B"
                }
            }
            attachment C for R {
                destroy() {
                    lastDestructorRun = "C"
                }
            }
            fun test() {
                let r <- create R()
                let r2 <- attach A() to <- attach B() to <- attach C() to <-r
                destroy r2
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredStringValue("R"), inter.Globals.Get("lastDestructorRun").GetValue())
	})

	t.Run("base destructor cannot add mutate attachments mid-destroy", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {
                fun foo() {
                    remove B from self
                }
                destroy() {}
            }
            attachment A for R {
                destroy() {
                   
                }
            }
            attachment B for R {
                destroy() {}
            }
            attachment C for R {
                destroy() {
                    base.foo()
                }
            }
            fun test() {
                let r <- create R()
                let r2 <- attach A() to <- attach B() to <- attach C() to <-r
                destroy r2
            }
        `)

		_, err := inter.Invoke("test")
		require.ErrorAs(t, err, &interpreter.AttachmentIterationMutationError{})
	})

	t.Run("remove runs destroy", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
            var destructorRun = false
            resource R {}
            attachment A for R {
                destroy() {
                    destructorRun = true
                }
            }
            fun test(): @R {
                let r <- create R()
                let r2 <- attach A() to <-r
                remove A from r2
                return <-r2
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.TrueValue, inter.Globals.Get("destructorRun").GetValue())
	})

	t.Run("remove runs resource field destroy", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
            var destructorRun = false
            resource R {}
            resource R2 {
                destroy() {
                    destructorRun = true
                }
            }
            attachment A for R {
                let r2: @R2
                init() {
                    self.r2 <- create R2()
                }
                destroy() {
                    destroy self.r2
                }
            }
            fun test(): @R {
                let r <- create R()
                let r2 <- attach A() to <-r
                remove A from r2
                return <-r2
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.TrueValue, inter.Globals.Get("destructorRun").GetValue())
	})

	t.Run("nested attachments destroyed", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
            var destructorRun = false
            resource R {}
            resource R2 {}
            attachment B for R2 {
                destroy() {
                    destructorRun = true
                }
            }
            attachment A for R {
                let r2: @R2
                init() {
                    self.r2 <- attach B() to <-create R2()
                }
                destroy() {
                    destroy self.r2
                }
            }
            fun test(): @R {
                let r <- create R()
                let r2 <- attach A() to <-r
                remove A from r2
                return <-r2
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.TrueValue, inter.Globals.Get("destructorRun").GetValue())
	})
}

func TestInterpretAttachmentResourceReferenceInvalidation(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
            resource R {}
            attachment A for R {
                fun foo(): Int { return 3 }
            }
            fun test() {
                let r <- create R()
                let r2 <- attach A() to <-r
                let a = r2[A]!
                authAccount.save(<-r2, to: /storage/foo)
                let i = a.foo()
            }
        `, sema.Config{},
		)

		// TODO: in the stable cadence branch, with the new resource reference invalidation,
		// this should be an error, as `a` shoudl be invalidated after the save
		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
            resource R {}
            resource R2 {
                let r: @R 
                init(r: @R) {
                    self.r <- r
                }
                destroy() {
                    destroy self.r
                }
            }
            attachment A for R {
                fun foo(): Int { return 3 }
            }
            fun test() {
                let r2 <- create R2(r: <-attach A() to <-create R())
                let a = r2.r[A]!
                authAccount.save(<-r2, to: /storage/foo)
                let i = a.foo()
            }
        
        `, sema.Config{},
		)

		// TODO: in the stable cadence branch, with the new resource reference invalidation,
		// this should be an error, as `a` shoudl be invalidated after the save
		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})
}
