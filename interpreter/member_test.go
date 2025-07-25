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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretMemberAccessType(t *testing.T) {

	t.Parallel()

	t.Run("direct", func(t *testing.T) {

		t.Run("non-optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct S {
                        var foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    fun get(s: S) {
                        s.foo
                    }

                    fun set(s: S) {
                        s.foo = 2
                    }
                `)

				// Construct an instance of type S
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S")
				require.NoError(t, err)

				_, err = inter.Invoke("get", value)
				require.NoError(t, err)

				_, err = inter.Invoke("set", value)
				require.NoError(t, err)
			})

			t.Run("invalid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct S {
                        var foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    struct S2 {
                        var foo: Int

                        init() {
                            self.foo = 2
                        }
                    }

                    fun get(s: S) {
                        s.foo
                    }

                    fun set(s: S) {
                        s.foo = 3
                    }
                `)

				// Construct an instance of type S2
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S2")
				require.NoError(t, err)

				_, err = inter.Invoke("get", value)
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)

				_, err = inter.Invoke("set", value)
				RequireError(t, err)
				require.ErrorAs(t, err, &memberAccessTypeError)
			})
		})

		t.Run("optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct S {
                        var foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    fun get(s: S?) {
                        s?.foo
                    }
                `)

				// Construct an instance of type S
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S")
				require.NoError(t, err)

				_, err = inter.Invoke(
					"get",
					interpreter.NewUnmeteredSomeValueNonCopying(value),
				)
				require.NoError(t, err)
			})

			t.Run("invalid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct S {
                        let foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    struct S2 {
                        let foo: Int

                        init() {
                            self.foo = 2
                        }
                    }

                    fun get(s: S?) {
                        s?.foo
                    }
                `)

				// Construct an instance of type S2
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S2")
				require.NoError(t, err)

				_, err = inter.Invoke(
					"get",
					interpreter.NewUnmeteredSomeValueNonCopying(value),
				)
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)
			})
		})
	})

	t.Run("interface", func(t *testing.T) {

		t.Run("non-optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct interface SI {
                        var foo: Int
                    }

                    struct S: SI {
                        var foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    fun get(si: {SI}) {
                        si.foo
                    }

                    fun set(si: {SI}) {
                        si.foo = 2
                    }
                `)

				// Construct an instance of type S
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S")
				require.NoError(t, err)

				_, err = inter.Invoke("get", value)
				require.NoError(t, err)

				_, err = inter.Invoke("set", value)
				require.NoError(t, err)
			})

			t.Run("invalid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct interface SI {
                        var foo: Int
                    }

                    struct S: SI {
                        var foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    struct S2 {
                        var foo: Int

                        init() {
                            self.foo = 2
                        }
                    }

                    fun get(si: {SI}) {
                        si.foo
                    }

                    fun set(si: {SI}) {
                        si.foo = 3
                    }
                `)

				// Construct an instance of type S2
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S2")
				require.NoError(t, err)

				_, err = inter.Invoke("get", value)
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)

				_, err = inter.Invoke("set", value)
				RequireError(t, err)
				require.ErrorAs(t, err, &memberAccessTypeError)
			})
		})

		t.Run("optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct interface SI {
                        let foo: Int
                    }

                    struct S: SI {
                        let foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    fun get(si: {SI}?) {
                        si?.foo
                    }
                `)

				// Construct an instance of type S
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S")
				require.NoError(t, err)

				_, err = inter.Invoke(
					"get",
					interpreter.NewUnmeteredSomeValueNonCopying(value),
				)
				require.NoError(t, err)
			})

			t.Run("invalid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct interface SI {
                        let foo: Int
                    }

                    struct S: SI {
                        let foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    struct S2 {
                        let foo: Int

                        init() {
                            self.foo = 2
                        }
                    }

                    fun get(si: {SI}?) {
                        si?.foo
                    }
                `)

				// Construct an instance of type S2
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S2")
				require.NoError(t, err)

				_, err = inter.Invoke(
					"get",
					interpreter.NewUnmeteredSomeValueNonCopying(value),
				)
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)
			})
		})
	})

	t.Run("reference", func(t *testing.T) {

		t.Run("non-optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct S {
                        var foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    fun get(ref: &S) {
                        ref.foo
                    }

                    fun set(ref: &S) {
                        ref.foo = 2
                    }
                `)

				// Construct an instance of type S
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S")
				require.NoError(t, err)

				sType := RequireGlobalType(t, inter, "S")

				ref := interpreter.NewUnmeteredEphemeralReferenceValue(
					inter,
					interpreter.UnauthorizedAccess,
					value,
					sType,
					interpreter.EmptyLocationRange,
				)

				_, err = inter.Invoke("get", ref)
				require.NoError(t, err)

				_, err = inter.Invoke("set", ref)
				require.NoError(t, err)
			})

			t.Run("invalid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct S {
                        var foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    struct S2 {
                        var foo: Int

                        init() {
                            self.foo = 2
                        }
                    }

                    fun get(ref: &S) {
                        ref.foo
                    }

                    fun set(ref: &S) {
                        ref.foo = 3
                    }
                `)

				// Construct an instance of type S2
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S2")
				require.NoError(t, err)

				sType := RequireGlobalType(t, inter, "S")

				ref := interpreter.NewUnmeteredEphemeralReferenceValue(
					inter,
					interpreter.UnauthorizedAccess,
					value,
					sType,
					interpreter.EmptyLocationRange,
				)

				_, err = inter.Invoke("get", ref)
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)

				_, err = inter.Invoke("set", ref)
				RequireError(t, err)
				require.ErrorAs(t, err, &memberAccessTypeError)
			})
		})

		t.Run("optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct S {
                        let foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    fun get(ref: &S?) {
                        ref?.foo
                    }
                `)

				// Construct an instance of type S
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S")
				require.NoError(t, err)

				sType := RequireGlobalType(t, inter, "S")

				ref := interpreter.NewUnmeteredEphemeralReferenceValue(
					inter,
					interpreter.UnauthorizedAccess,
					value,
					sType,
					interpreter.EmptyLocationRange,
				)

				_, err = inter.Invoke(
					"get",
					interpreter.NewUnmeteredSomeValueNonCopying(
						ref,
					),
				)
				require.NoError(t, err)
			})

			t.Run("invalid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct S {
                        let foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    struct S2 {
                        let foo: Int

                        init() {
                            self.foo = 2
                        }
                    }

                    fun get(ref: &S?) {
                        ref?.foo
                    }
                `)

				// Construct an instance of type S2
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S2")
				require.NoError(t, err)

				sType := RequireGlobalType(t, inter, "S")

				ref := interpreter.NewUnmeteredEphemeralReferenceValue(
					inter,
					interpreter.UnauthorizedAccess,
					value,
					sType,
					interpreter.EmptyLocationRange,
				)

				_, err = inter.Invoke(
					"get",
					interpreter.NewUnmeteredSomeValueNonCopying(
						ref,
					),
				)
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)
			})
		})
	})
}

func TestInterpretMemberAccess(t *testing.T) {

	t.Parallel()

	t.Run("composite, field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Test {
                var x: [Int]
                init() {
                    self.x = []
                }
            }

            fun test(): [Int] {
                let test = Test()
                return test.x
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("composite, function", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Test {
                access(all) fun foo(): Int {
                    return 1
                }
            }

            fun test(): Int {
                let test = Test()
                var foo: (fun(): Int) = test.foo
                return foo()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), result)
	})

	t.Run("composite reference, field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Test {
                var x: [Int]
                init() {
                    self.x = []
                }
            }

            fun test() {
                let test = Test()
                let testRef = &test as &Test
                var x: &[Int] = testRef.x
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("composite reference, optional field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Test {
                var x: [Int]?
                init() {
                    self.x = []
                }
            }

            fun test() {
                let test = Test()
                let testRef = &test as &Test
                var x: &[Int]? = testRef.x
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("composite reference, nil in optional field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Test {
                var x: [Int]?
                init() {
                    self.x = nil
                }
            }

            fun test(): &[Int]? {
                let test = Test()
                let testRef = &test as &Test
                return testRef.x
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.Nil,
			value,
		)
	})

	t.Run("composite reference, nil in anystruct field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Test {
                var x: AnyStruct
                init() {
                    self.x = nil
                }
            }

            fun test(): &AnyStruct {
                let test = Test()
                let testRef = &test as &Test
                return testRef.x
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var referenceToNilError *interpreter.NonOptionalReferenceToNilError
		require.ErrorAs(t, err, &referenceToNilError)
	})

	t.Run("composite reference, primitive field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Test {
                var x: Int
                init() {
                    self.x = 1
                }
            }

            fun test() {
                let test = Test()
                let testRef = &test as &Test
                var x: Int = testRef.x
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("composite reference, function", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Test {
                access(all) fun foo(): Int {
                    return 1
                }
            }

            fun test(): Int {
                let test = Test()
                let testRef = &test as &Test
                var foo: (fun(): Int) = testRef.foo
                return foo()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), result)
	})

	t.Run("resource reference, nested", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            resource Foo {
                var bar: @Bar
                init() {
                    self.bar <- create Bar()
                }
            }

            resource Bar {
                var baz: @Baz
                init() {
                    self.baz <- create Baz()
                }
            }

            resource Baz {
                var x: &[Int]
                init() {
                    self.x = &[] as &[Int]
                }
            }

            fun test() {
                let foo <- create Foo()
                let fooRef = &foo as &Foo

                // Nested container fields must return references
                var barRef: &Bar = fooRef.bar
                var bazRef: &Baz = fooRef.bar.baz

                // Reference typed field should return as is (no double reference must be created)
                var x: &[Int] = fooRef.bar.baz.x

                destroy foo
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("composite reference, anystruct typed field, with reference value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Test {
                var x: AnyStruct
                init() {
                    var s = "hello"
                    self.x = &s as &String
                }
            }

            fun test():&AnyStruct  {
                let test = Test()
                let testRef = &test as &Test
                return testRef.x
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, result)
		ref := result.(*interpreter.EphemeralReferenceValue)

		// Must only have one level of references.
		require.IsType(t, &interpreter.StringValue{}, ref.Value)
	})

	t.Run("array, element", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [[Int]] = [[1, 2]]
                var x: [Int] = array[0]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("array reference, element", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [[Int]] = [[1, 2]]
                let arrayRef = &array as &[[Int]]
                var x: &[Int] = arrayRef[0]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("array authorized reference, element", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            entitlement A

            fun test() {
                let array: [[Int]] = [[1, 2]]
                let arrayRef = &array as auth(A) &[[Int]]

                // Must return an unauthorized reference.
                var x: &[Int] = arrayRef[0]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("array reference, element, in assignment", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [[Int]] = [[1, 2]]
                let arrayRef = &array as &[[Int]]
                var x: &[Int] = &[] as &[Int]
                x = arrayRef[0]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("array reference, optional typed element", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [[Int]?] = [[1, 2]]
                let arrayRef = &array as &[[Int]?]
                var x: &[Int]? = arrayRef[0]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("array reference, primitive typed element", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [Int] = [1, 2]
                let arrayRef = &array as &[Int]
                var x: Int = arrayRef[0]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("array reference, anystruct typed element, with reference value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): &AnyStruct {
                var s = "hello"
                let array: [AnyStruct] = [&s as &String]
                let arrayRef = &array as &[AnyStruct]
                return arrayRef[0]
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, result)
		ref := result.(*interpreter.EphemeralReferenceValue)

		// Must only have one level of references.
		require.IsType(t, &interpreter.StringValue{}, ref.Value)
	})

	t.Run("dictionary, value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let dict: {String: {String: Int}} = {"one": {"two": 2}}
                var x: {String: Int}? = dict["one"]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("dictionary reference, value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let dict: {String: {String: Int} } = {"one": {"two": 2}}
                let dictRef = &dict as &{String: {String: Int}}
                var x: &{String: Int}? = dictRef["one"]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("dictionary authorized reference, value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            entitlement A

            fun test() {
                let dict: {String: {String: Int} } = {"one": {"two": 2}}
                let dictRef = &dict as auth(A) &{String: {String: Int}}

                // Must return an unauthorized reference.
                var x: &{String: Int}? = dictRef["one"]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("dictionary reference, value, in assignment", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let dict: {String: {String: Int} } = {"one": {"two": 2}}
                let dictRef = &dict as &{String: {String: Int}}
                var x: &{String: Int}? = &{} as &{String: Int}
                x = dictRef["one"]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("dictionary reference, optional typed value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let dict: {String: {String: Int}?} = {"one": {"two": 2}}
                let dictRef = &dict as &{String: {String: Int}?}
                var x: (&{String: Int})?? = dictRef["one"]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("dictionary reference, primitive typed value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let dict: {String: Int} = {"one": 1}
                let dictRef = &dict as &{String: Int}
                var x: Int? = dictRef["one"]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource reference, attachment", func(t *testing.T) {
		t.Parallel()

		// TODO: requires support for attachments in the VM
		inter := parseCheckAndInterpret(t, `
            resource R {}

            attachment A for R {}

            fun test() {
                let r <- create R()
                let rRef = &r as &R

                var a: &A? = rRef[A]
                destroy r
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("attachment nested member", func(t *testing.T) {
		t.Parallel()

		// TODO: requires support for attachments in the VM
		inter := parseCheckAndInterpret(t, `
            resource R {}

            attachment A for R {
                var foo: Foo
                init() {
                    self.foo = Foo()
                }

                access(all) fun getNestedMember(): [Int] {
                    return self.foo.array
                }
            }

            struct Foo {
                var array: [Int]
                init() {
                    self.array = []
                }
            }

            fun test() {
                let r <- attach A() to <- create R()
                let rRef = &r as &R

                var a: &A? = rRef[A]

                var array = a!.getNestedMember()

                destroy r
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("anystruct swap on reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Foo {
                var array: [Int]
                init() {
                    self.array = []
                }
            }

            fun test() {
                let dict: {String: AnyStruct} = {"foo": Foo(), "bar": Foo()}
                let dictRef = &dict as auth(Mutate) &{String: AnyStruct}

                dictRef["foo"] <-> dictRef["bar"]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("entitlement map access on field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            entitlement A
            entitlement B
            entitlement mapping M {
                A -> B
            }

            struct S {
                access(mapping M) let foo: [String]
                init() {
                    self.foo = []
                }
            }

            fun test() {
                let s = S()
                let sRef = &s as auth(A) &S
                var foo: auth(B) &[String] = sRef.foo
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("all member types", func(t *testing.T) {
		t.Parallel()

		test := func(tt *testing.T, typeName string) {
			code := fmt.Sprintf(`
                struct Foo {
                    var a: %[1]s?

                    init() {
                        self.a = nil
                    }
                }

                struct Bar {}

                struct interface I {}

                fun test() {
                    let foo = Foo()
                    let fooRef = &foo as &Foo
                    var a: &%[1]s? = fooRef.a
                }`,

				typeName,
			)

			inter := parseCheckAndPrepare(t, code)

			_, err := inter.Invoke("test")
			require.NoError(t, err)
		}

		types := []string{
			"Bar",
			"{I}",
			"[Int]",
			"{Bool: String}",
		}

		// Test all built-in composite types
		for ty := interpreter.PrimitiveStaticType(1); ty < interpreter.PrimitiveStaticType_Count; ty++ {
			if !ty.IsDefined() || ty.IsDeprecated() { //nolint:staticcheck
				continue
			}

			semaType := ty.SemaType()

			if !semaType.ContainFieldsOrElements() ||
				semaType.IsResourceType() {

				continue
			}

			types = append(types, semaType.QualifiedString())
		}

		for _, typeName := range types {
			t.Run(typeName, func(t *testing.T) {
				test(t, typeName)
			})
		}
	})
}

func TestInterpretNestedReferenceMemberAccess(t *testing.T) {

	t.Parallel()

	t.Run("indexing", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            resource R {}

            fun test() {
                let r <- create R()
                let arrayRef = &[&r as &R] as &[AnyStruct]
                let ref: &AnyStruct = arrayRef[0]  // <--- run-time error here
                destroy r
            }
        `)

		_, err := inter.Invoke("test")
		var invalidMemberReferenceError *interpreter.InvalidMemberReferenceError
		require.ErrorAs(t, err, &invalidMemberReferenceError)
	})

	t.Run("field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            resource R {}

            struct Container {
                let value: AnyStruct
            
                init(value: AnyStruct) {
                    self.value = value
                }
            }
            
            fun test() {
                let r <- create R()
                let containerRef = &Container(value: &r as &R) as &Container
                let ref: &AnyStruct = containerRef.value  // <--- run-time error here
                destroy r
            }        
        `)

		_, err := inter.Invoke("test")
		var invalidMemberReferenceError *interpreter.InvalidMemberReferenceError
		require.ErrorAs(t, err, &invalidMemberReferenceError)
	})

	t.Run("referenceArray", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `

            entitlement E

            struct T {
                access(E) fun foo() {}
            }

            fun test() {
                let t1 = T()
                let t2 = T()
                let arr: [AnyStruct] = [&t1 as auth(E) &T, &t2 as auth(E) &T]
                let arrRef = &arr as &[AnyStruct]
                let tRef = arrRef[0]
            }
            
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})
}

func TestInterpretOptionalChaining(t *testing.T) {

	t.Parallel()

	t.Run("method call", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            var x: Int? = nil

            struct S {
                fun getX(): Int? {
                   return x
                }
            }

            fun test(): Int? {
                var s: S? = S()
                return s?.getX()!
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)
		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	t.Run("field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct S {
                var x: Int?

                init() {
                   self.x = nil
                }
            }

            fun test(): Int {
                var s: S? = S()
                return s?.x!
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		forceNilError := &interpreter.ForceNilError{}
		require.ErrorAs(t, err, &forceNilError)
	})
}
