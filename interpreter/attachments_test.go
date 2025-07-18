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

package interpreter_test

import (
	"testing"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"

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
		var duplicateAttachmentError *interpreter.DuplicateAttachmentError
		require.ErrorAs(t, err, &duplicateAttachmentError)
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
		var duplicateAttachmentError *interpreter.DuplicateAttachmentError
		require.ErrorAs(t, err, &duplicateAttachmentError)
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
}

func TestInterpretAttachExecutionOrdering(t *testing.T) {
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

		_, err := inter.Invoke("test")
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
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

		_, err := inter.Invoke("test")
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	})

	t.Run("return from function", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R { }
            attachment A for R {
                fun returnSelf(): &A {
                    return self
                }
                fun foo(): Int {
                    return 3
                }
            }
            fun test(): Int {
                let r <- attach A() to <-create R()
                let i = r[A]!.returnSelf().foo()
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

func TestInterpretAttachmentIntersectionType(t *testing.T) {
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

	t.Run("constructor on intersection", func(t *testing.T) {

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

func TestInterpretAttachmentDestructor(t *testing.T) {

	t.Parallel()

	t.Run("destructor run on base destroy", func(t *testing.T) {

		t.Parallel()

		var eventTypes []*sema.CompositeType

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
                resource R {
                    event ResourceDestroyed()
                }
                attachment A for R {
                    event ResourceDestroyed()
                }
                fun test() {
                    let r <- create R()
                    let r2 <- attach A() to <-r
                    destroy r2
                }
            `,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					OnEventEmitted: func(
						_ interpreter.ValueExportContext,
						_ interpreter.LocationRange,
						eventType *sema.CompositeType,
						_ []interpreter.Value,
					) error {
						eventTypes = append(eventTypes, eventType)
						return nil
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		require.Len(t, eventTypes, 2)
		require.Equal(t, "A.ResourceDestroyed", eventTypes[0].QualifiedIdentifier())
		require.Equal(t, "R.ResourceDestroyed", eventTypes[1].QualifiedIdentifier())
	})

	t.Run("base destructor executed last", func(t *testing.T) {

		t.Parallel()

		var eventTypes []*sema.CompositeType

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
                resource R {
                    event ResourceDestroyed()
                }

                attachment A for R {
                    event ResourceDestroyed()
                }

                attachment B for R {
                    event ResourceDestroyed()
                }

                attachment C for R {
                    event ResourceDestroyed()
                }

                fun test() {
                    let r <- create R()
                    let r2 <- attach A() to <- attach B() to <- attach C() to <-r
                    destroy r2
                }
            `,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					OnEventEmitted: func(
						_ interpreter.ValueExportContext,
						_ interpreter.LocationRange,
						eventType *sema.CompositeType,
						_ []interpreter.Value,
					) error {
						eventTypes = append(eventTypes, eventType)
						return nil
					},
				},
			})
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		require.Len(t, eventTypes, 4)
		// the only part of this order that is important is that `R` is last
		require.Equal(t, "B.ResourceDestroyed", eventTypes[0].QualifiedIdentifier())
		require.Equal(t, "C.ResourceDestroyed", eventTypes[1].QualifiedIdentifier())
		require.Equal(t, "A.ResourceDestroyed", eventTypes[2].QualifiedIdentifier())
		require.Equal(t, "R.ResourceDestroyed", eventTypes[3].QualifiedIdentifier())
	})

	t.Run("remove runs destroy", func(t *testing.T) {

		var eventTypes []*sema.CompositeType

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
                resource R {
                    event ResourceDestroyed()
                }

                attachment A for R {
                    event ResourceDestroyed()
                }

                fun test(): @R {
                    let r <- create R()
                    let r2 <- attach A() to <-r
                    remove A from r2
                    return <-r2
                }
            `,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					OnEventEmitted: func(
						_ interpreter.ValueExportContext,
						_ interpreter.LocationRange,
						eventType *sema.CompositeType,
						_ []interpreter.Value,
					) error {
						eventTypes = append(eventTypes, eventType)
						return nil
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		require.Len(t, eventTypes, 1)
		require.Equal(t, "A.ResourceDestroyed", eventTypes[0].QualifiedIdentifier())
	})

	t.Run("remove runs resource field destroy", func(t *testing.T) {

		var eventTypes []*sema.CompositeType

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
                resource R {
                    event ResourceDestroyed()
                }

                resource R2 {
                    event ResourceDestroyed()
                }

                attachment A for R {
                    event ResourceDestroyed()

                    let r2: @R2

                    init() {
                        self.r2 <- create R2()
                    }
                }

                fun test(): @R {
                    let r <- create R()
                    let r2 <- attach A() to <-r
                    remove A from r2
                    return <-r2
                }
            `,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					OnEventEmitted: func(
						_ interpreter.ValueExportContext,
						_ interpreter.LocationRange,
						eventType *sema.CompositeType,
						_ []interpreter.Value,
					) error {
						eventTypes = append(eventTypes, eventType)
						return nil
					},
				},
			})
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		require.Len(t, eventTypes, 2)
		require.Equal(t, "R2.ResourceDestroyed", eventTypes[0].QualifiedIdentifier())
		require.Equal(t, "A.ResourceDestroyed", eventTypes[1].QualifiedIdentifier())
	})

	t.Run("nested attachments destroyed", func(t *testing.T) {

		var eventTypes []*sema.CompositeType

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
                resource R {
                    event ResourceDestroyed()
                }

                resource R2 {
                    event ResourceDestroyed()
                }

                attachment B for R2 {
                    event ResourceDestroyed()
                }

                attachment A for R {
                    event ResourceDestroyed()

                    let r2: @R2

                    init() {
                        self.r2 <- attach B() to <-create R2()
                    }
                }

                fun test(): @R {
                    let r <- create R()
                    let r2 <- attach A() to <-r
                    remove A from r2
                    return <-r2
                }
            `,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					OnEventEmitted: func(
						_ interpreter.ValueExportContext,
						_ interpreter.LocationRange,
						eventType *sema.CompositeType,
						_ []interpreter.Value,
					) error {
						eventTypes = append(eventTypes, eventType)
						return nil
					},
				},
			})
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		require.Len(t, eventTypes, 3)
		require.Equal(t, "B.ResourceDestroyed", eventTypes[0].QualifiedIdentifier())
		require.Equal(t, "R2.ResourceDestroyed", eventTypes[1].QualifiedIdentifier())
		require.Equal(t, "A.ResourceDestroyed", eventTypes[2].QualifiedIdentifier())
	})

	t.Run("attachment default args properly scoped", func(t *testing.T) {

		t.Parallel()

		var eventTypes []*sema.CompositeType
		var eventsFields [][]interpreter.Value

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
                resource R {
                    var foo: String

                    event ResourceDestroyed()

                    init() {
                        self.foo = "baz"
                    }

                    fun setFoo(arg: String) {
                        self.foo = arg
                    }
                }

                attachment A for R {
                    var bar: Int

                    event ResourceDestroyed(foo: String = base.foo, bar: Int = self.bar)

                    init() {
                        self.bar = 1
                    }

                    fun setBar(arg: Int) {
                        self.bar = arg
                    }
                }

                fun test() {
                    let r <- create R()
                    let r2 <- attach A() to <-r
                    r2.setFoo(arg: "foo")
                    r2[A]!.setBar(arg: 2)
                    destroy r2
                }
            `,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					OnEventEmitted: func(
						_ interpreter.ValueExportContext,
						_ interpreter.LocationRange,
						eventType *sema.CompositeType,
						eventFields []interpreter.Value,
					) error {
						eventTypes = append(eventTypes, eventType)
						eventsFields = append(eventsFields, eventFields)
						return nil
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		require.Len(t, eventTypes, 2)
		require.Equal(t, "A.ResourceDestroyed", eventTypes[0].QualifiedIdentifier())
		require.Equal(t, "R.ResourceDestroyed", eventTypes[1].QualifiedIdentifier())
		require.Equal(t,
			[][]interpreter.Value{
				{
					interpreter.NewUnmeteredStringValue("foo"),
					interpreter.NewIntValueFromInt64(nil, 2),
				},
				nil,
			},
			eventsFields,
		)
	})
}

func TestInterpretAttachmentResourceReferenceInvalidation(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            resource R {}
            attachment A for R {
                access(all) var id: UInt8

                access(all) fun setID(_ id: UInt8) {
					self.id = id
				}

                init() {
                    self.id = 1
                }
            }
            fun test(): UInt8 {
                let r <- create R()
                let r2 <- attach A() to <-r
                let a = returnSameRef(r2[A]!)


                // Move the resource after taking a reference to the attachment.
                // Then update the field of the attachment.
                var r3 <- r2
                let a2 = r3[A]!
                a2.setID(5)
                authAccount.storage.save(<-r3, to: /storage/foo)

                // Access the attachment filed from the previous reference.
                return a.id
            }

		    access(all) fun returnSameRef(_ ref: &A): &A {
		        return ref
		    }`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	})

	t.Run("destroyed", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            resource R {}
            attachment A for R {
                fun foo(): Int { return 3 }
            }
            fun test() {
                let r <- create R()
                let r2 <- attach A() to <-r
                let a = returnSameRef(r2[A]!)
                destroy r2
                let i = a.foo()
            }

		    access(all) fun returnSameRef(_ ref: &A): &A {
		        return ref
		    }`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            resource R {}
            resource R2 {
                let r: @R
                init(r: @R) {
                    self.r <- r
                }
            }
            attachment A for R {
                access(all) var id: UInt8

                access(all) fun setID(_ id: UInt8) {
					self.id = id
				}

                init() {
                    self.id = 1
                }
            }
            fun test(): UInt8 {
                let r2 <- create R2(r: <-attach A() to <-create R())
                let a = returnSameRef(r2.r[A]!)

                // Move the resource after taking a reference to the attachment.
                // Then update the field of the attachment.
                var r3 <- r2
                let a2 = r3.r[A]!
                a2.setID(5)
                authAccount.storage.save(<-r3, to: /storage/foo)

                // Access the attachment filed from the previous reference.
                return a.id
            }

		    access(all) fun returnSameRef(_ ref: &A): &A {
		        return ref
		    }`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	})

	t.Run("base reference", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            access(all) resource R {
                access(all) var id: UInt8

                access(all) fun setID(_ id: UInt8) {
					self.id = id
				}

                init() {
                    self.id = 1
                }
            }

            var ref: &R? = nil

            attachment A for R {
                fun saveBaseRef() {
                    ref = base
                }
            }

            fun test(): UInt8 {
                let r <- attach A() to <-create R()
                let a = r[A]!

                a.saveBaseRef()

                var r2 <- r
                r2.setID(5)
                authAccount.storage.save(<-r2, to: /storage/foo)
                return ref!.id
            }`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	})

	t.Run("nested destroyed", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            resource R {}

            resource R2 {
                let r: @R

                init(r: @R) {
                    self.r <- r
                }
            }

            attachment A for R {
                fun foo(): Int { return 3 }
            }

            fun test() {
                let r2 <- create R2(r: <-attach A() to <-create R())
                let a = returnSameRef(r2.r[A]!)
                destroy r2
                let i = a.foo()
            }

		    access(all) fun returnSameRef(_ ref: &A): &A {
		        return ref
		    }`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	})

	t.Run("self reference", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            access(all) resource R {}

            var ref: &A? = nil

            attachment A for R {
                access(all) var id: UInt8

                access(all) fun setID(_ id: UInt8) {
					self.id = id
				}

                init() {
                    self.id = 1
                }

                fun saveSelfRef() {
                    ref = self as &A
                }
            }

            fun test(): UInt8 {
                let r <- attach A() to <-create R()
                r[A]!.saveSelfRef()

                var r2 <- r
                let a = r2[A]!
                a.setID(5)
                authAccount.storage.save(<-r2, to: /storage/foo)
                return ref!.id
            }`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	})
}

func TestInterpretAttachmentsRuntimeType(t *testing.T) {

	t.Parallel()

	t.Run("getType()", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}
            attachment A for R {}
            fun test(): Type {
                let r <- create R()
                let r2 <- attach A() to <-r
                let a: Type = r2[A]!.getType()
                destroy r2
                return a
            }
        `)

		a, err := inter.Invoke("test")
		require.NoError(t, err)
		require.IsType(t, interpreter.TypeValue{}, a)
		require.Equal(t, "S.test.A", a.(interpreter.TypeValue).Type.String())
	})
}

func TestInterpretAttachmentDefensiveCheck(t *testing.T) {
	t.Parallel()

	t.Run("reference attach", func(t *testing.T) {

		t.Parallel()

		inter, _ := parseCheckAndInterpretWithOptions(t, `
        struct S {}
        attachment A for S {}
        fun test() {
            var s = S()
            var ref = &s as &S
            ref = attach A() to ref
        }
        `, ParseCheckAndInterpretOptions{
			HandleCheckerError: func(_ error) {},
		})

		_, err := inter.Invoke("test")
		var attachmentError *interpreter.InvalidAttachmentOperationTargetError
		require.ErrorAs(t, err, &attachmentError)
	})

	t.Run("reference remove", func(t *testing.T) {

		t.Parallel()

		inter, _ := parseCheckAndInterpretWithOptions(t, `
        struct S {}
        attachment A for S {}
        fun test() {
            var s = S()
            var ref = &s as &S
            remove A from S
        }
        `, ParseCheckAndInterpretOptions{
			HandleCheckerError: func(_ error) {},
		})

		_, err := inter.Invoke("test")
		var attachmentError *interpreter.InvalidAttachmentOperationTargetError
		require.ErrorAs(t, err, &attachmentError)
	})

	t.Run("array attach", func(t *testing.T) {

		t.Parallel()

		inter, _ := parseCheckAndInterpretWithOptions(t, `
        struct S {}
        attachment A for S {}
        fun test() {
            var s = S()
            var a = attach A() to [s]
        }
        `, ParseCheckAndInterpretOptions{
			HandleCheckerError: func(_ error) {},
		})

		_, err := inter.Invoke("test")
		var attachmentError *interpreter.InvalidAttachmentOperationTargetError
		require.ErrorAs(t, err, &attachmentError)
	})

	t.Run("array remove", func(t *testing.T) {

		t.Parallel()

		inter, _ := parseCheckAndInterpretWithOptions(t, `
        struct S {}
        attachment A for S {}
        fun test() {
            var s = S()
            remove A from [s]
        }
        `, ParseCheckAndInterpretOptions{
			HandleCheckerError: func(_ error) {},
		})

		_, err := inter.Invoke("test")
		var attachmentError *interpreter.InvalidAttachmentOperationTargetError
		require.ErrorAs(t, err, &attachmentError)
	})

	t.Run("enum attach", func(t *testing.T) {

		t.Parallel()

		inter, _ := parseCheckAndInterpretWithOptions(t, `
        struct S {}
        attachment A for S {}
        enum E: UInt8 {
            case a
        }
        fun test() {
            var s = S()
            var e: E = E.a
            ref = attach A() to e
        }
        `, ParseCheckAndInterpretOptions{
			HandleCheckerError: func(_ error) {},
		})

		_, err := inter.Invoke("test")
		var attachmentError *interpreter.InvalidAttachmentOperationTargetError
		require.ErrorAs(t, err, &attachmentError)
	})

	t.Run("enum remove", func(t *testing.T) {

		t.Parallel()

		inter, _ := parseCheckAndInterpretWithOptions(t, `
        struct S {}
        attachment A for S {}
        enum E: UInt8 {
            case a
        }
        fun test() {
            var s = S()
            var e: E = E.a
            remove A from e
        }
        `, ParseCheckAndInterpretOptions{
			HandleCheckerError: func(_ error) {},
		})

		_, err := inter.Invoke("test")
		var attachmentError *interpreter.InvalidAttachmentOperationTargetError
		require.ErrorAs(t, err, &attachmentError)
	})
}

func TestInterpretAttachmentSelfAccessMembers(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
            access(all) resource R{
                access(all) fun baz() {}
            }
            access(all) attachment A for R{
                access(all) fun foo() {}
                access(self) fun qux1() {
                    self.foo()
                    base.baz()
                }
                access(contract) fun qux2() {
                    self.foo()
                    base.baz()
                }
                access(account) fun qux3() {
                    self.foo()
                    base.baz()
                }
                access(all) fun bar() {
                    self.qux1()
                    self.qux2()
                    self.qux3()
                }
            }

            access(all) fun main() {
                var r <- attach A() to <- create R()
                r[A]!.bar()
                destroy r
            }
        `)

	_, err := inter.Invoke("main")
	require.NoError(t, err)
}

func TestInterpretForEachAttachment(t *testing.T) {

	t.Parallel()

	t.Run("count resource", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}
            attachment A for R {}
            attachment B for R {}
            attachment C for R {}
            fun test(): Int {
                var r <- attach C() to <- attach B() to <- attach A() to <- create R()
                var i = 0
                r.forEachAttachment(fun(attachmentRef: &AnyResourceAttachment) {
                    i = i + 1
                })
                destroy r
                return i
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("count struct", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {}
            attachment B for S {}
            attachment C for S {}
            fun test(): Int {
                var s = attach C() to attach B() to attach A() to S()
                var i = 0
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    i = i + 1
                })
                return i
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("invoke foos", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {
                fun foo(_ x: Int): Int { return 7 + x }
            }
            attachment B for S {
                fun foo(): Int { return 10 }
            }
            attachment C for S {
                fun foo(_ x: Int): Int { return 8 + x }
            }
            fun test(): Int {
                var s = attach C() to attach B() to attach A() to S()
                var i = 0
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    if let a = attachmentRef as? &A {
                        i = i + a.foo(1)
                    } else if let b = attachmentRef as? &B {
                        i = i + b.foo()
                    } else if let c = attachmentRef as? &C {
                        i = i + c.foo(1)
                    }
                })
                return i
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(27), value)
	})

	t.Run("invoke foos with auth", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement F
            entitlement Y
            struct S {
                access(F, Y) fun foo() {}
            }
            access(all) attachment A for S {
                access(F) fun foo(_ x: Int): Int { return 7 + x }
            }
            access(all) attachment B for S {
                access(Y) fun foo(): Int { return 10 }
            }
            access(all) attachment C for S {
                access(Y) fun foo(_ x: Int): Int { return 8 + x }
            }
            fun test(): Int {
                var s = attach C() to attach B() to attach A() to S()
                let ref = &s as auth(F, Y) &S
                var i = 0
                ref.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    if let a = attachmentRef as? auth(F) &A {
                        i = i + a.foo(1)
                    } else if let b = attachmentRef as? auth(Y) &B {
                        i = i + b.foo()
                    } else if let c = attachmentRef as? auth(Y) &C {
                        i = i + c.foo(1)
                    }
                })
                return i
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		// the attachment reference is never entitled
		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(0), value)
	})

	t.Run("bound function", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement F

            access(all) struct S {
                access(F) let x: Int
                init() {
                    self.x = 3
                }
            }
            access(all) attachment A for S {
                access(F) var funcPtr: fun(): auth(F) &Int;
                init() {
                    self.funcPtr = self.foo
                }
                access(F) fun foo(): auth(F) &Int {
                    return &base.x
                }
            }
            fun test(): &Int {
                let r = attach A() to S()
                let rRef = &r as auth(F) &S
                let a = rRef[A]!
                let i = a.foo()
                return i
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			value.(*interpreter.EphemeralReferenceValue).Value,
		)
	})

	t.Run("access fields", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource Sub {
                let name: String
                init(_ name: String) {
                    self.name = name
                }
            }
            resource R {}
            attachment A for R {
                let r: @Sub
                init(_ name: String) {
                    self.r <- create Sub(name)
                }
            }
            attachment B for R {}
            attachment C for R {
                let r: @Sub
                init(_ name: String) {
                    self.r <- create Sub(name)
                }
            }
            fun test(): String {
                var r <- attach C("World") to <- attach B() to <- attach A("Hello") to <- create R()
                var text = ""
                r.forEachAttachment(fun(attachmentRef: &AnyResourceAttachment) {
                    if let a = attachmentRef as? &A {
                        text = text.concat(a.r.name)
                    } else if let a = attachmentRef as? &C {
                        text = text.concat(a.r.name)
                    } else {
                        text = text.concat(" ")
                    }
                })
                destroy r
                return text
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		// order of interation over the attachment is not defined, but must be deterministic nonetheless
		AssertValuesEqual(t, inter, interpreter.NewUnmeteredStringValue(" WorldHello"), value)
	})

	t.Run("box and convert argument", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R {}

            attachment A for R {
                fun map(f: fun(AnyStruct): String): String {
                    return "A.map"
                }
            }

            fun test(): String? {
                var res: String? = nil
                var r <- attach A() to <- create R()
                // NOTE: The function has a parameter of type &AnyResourceAttachment?
                // instead of just &AnyResourceAttachment?
                r.forEachAttachment(fun (ref: &AnyResourceAttachment?) {
                    // The map should call Optional.map, not fail,
                    // because path is &AnyResourceAttachment?, not &AnyResourceAttachment
                    res = ref.map(fun(string: AnyStruct): String {
                        return "Optional.map"
                    })
                })
                destroy r
                return res
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewSomeValueNonCopying(
				nil,
				interpreter.NewUnmeteredStringValue("Optional.map"),
			),
			value,
		)
	})
}

func TestInterpretMutationDuringForEachAttachment(t *testing.T) {
	t.Parallel()

	t.Run("basic attach", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {}
            attachment B for S {}
            fun test() {
                var s = attach A() to S()
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    s = attach B() to s
                })
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var attachmentError *interpreter.AttachmentIterationMutationError
		require.ErrorAs(t, err, &attachmentError)
	})

	t.Run("basic remove", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {}
            attachment B for S {}
            fun test() {
                var s = attach B() to attach A() to S()
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    remove A from s
                })
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var attachmentError *interpreter.AttachmentIterationMutationError
		require.ErrorAs(t, err, &attachmentError)
	})

	t.Run("attach to other", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {}
            attachment B for S {}
            fun test() {
                var s = attach A() to S()
                var s2 = attach A() to S()
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    s = attach B() to s2
                })
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("remove from other", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {}
            attachment B for S {}
            fun test() {
                var s = attach B() to attach A() to S()
                var s2 = attach B() to attach A() to S()
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    remove A from s2
                })
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("nested iteration", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {}
            attachment B for S {}
            fun test() {
                var s = attach B() to attach A() to S()
                var s2 = attach B() to attach A() to S()
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    s2.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                        remove A from s2
                    })
                })
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var attachmentError *interpreter.AttachmentIterationMutationError
		require.ErrorAs(t, err, &attachmentError)
	})

	t.Run("nested iteration of same", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {}
            attachment B for S {}
            fun test() {
                var s = attach B() to attach A() to S()
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {})
                    remove A from s
                })
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var attachmentError *interpreter.AttachmentIterationMutationError
		require.ErrorAs(t, err, &attachmentError)
	})

	t.Run("nested iteration ok", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {}
            attachment B for S {}
            fun test(): Int {
                var s = attach B() to attach A() to S()
                var s2 = attach B() to attach A() to S()
                var i = 0
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    remove A from s2
                    s2.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                        i = i + 1
                    })
                })
                return i
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(2), value)
	})

	t.Run("nested iteration ok after", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {}
            attachment A for S {}
            attachment B for S {}
            fun test(): Int {
                var s = attach B() to attach A() to S()
                var s2 = attach B() to attach A() to S()
                var i = 0
                s.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                    s2.forEachAttachment(fun(attachmentRef: &AnyStructAttachment) {
                        i = i + 1
                    })
                    remove A from s2
                })
                return i
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("callback", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            access(all) resource R {
                let foo: Int
                init() {
                    self.foo = 9
                }
            }
            access(all) attachment A for R {
                access(all) fun touchBase(): Int {
                    var foo = base.foo
                    return foo
                }
            }
            access(all) fun main(): Int {
                var r <- attach A() to <- create R()
                var id: Int = 0
                r.forEachAttachment(fun(a: &AnyResourceAttachment) {
                    id = (a as! &A).touchBase()
                });
                destroy r
                return id
            }
        `)

		value, err := inter.Invoke("main")
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.NewUnmeteredIntValueFromInt64(9), value)
	})
}

func TestInterpretBuiltinCompositeAttachment(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	for _, valueDeclaration := range []stdlib.StandardLibraryValue{
		stdlib.NewInterpreterPublicKeyConstructor(
			assumeValidPublicKeyValidator{},
		),
		stdlib.InterpreterSignatureAlgorithmConstructor,
	} {
		baseValueActivation.DeclareValue(valueDeclaration)
		interpreter.Declare(baseActivation, valueDeclaration)
	}

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          attachment A for AnyStruct {
              fun foo(): Int {
                  return 42
              }
          }

          fun main(): Int {
              var key = PublicKey(
                  publicKey: "0102".decodeHex(),
                  signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
              )
			  key = attach A() to key
              return key[A]!.foo()
          }
        `,
		ParseCheckAndInterpretOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
			InterpreterConfig: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("main")
	require.NoError(t, err)
}

func TestInterpretAttachmentSelfInvalidationInIteration(t *testing.T) {
	t.Parallel()

	t.Run("with iteration", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            access(all) resource R{
                init() {}
            }
            access(all) attachment A for R{
                access(all) fun zombieFunction() {}
            }

            access(all) fun main() {
                var r <- create R()
                var r2 <- attach A() to <- r
                var aRef: &A? = nil
                r2.forEachAttachment(fun(a: &AnyResourceAttachment) {
                    aRef = (a as! &A)
                })
                remove A from r2
                // Should not succeed, the attachment pointed to by aRef was destroyed
                aRef!.zombieFunction()
                destroy r2
            }
        `)

		_, err := inter.Invoke("main")
		RequireError(t, err)

		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	})
}
