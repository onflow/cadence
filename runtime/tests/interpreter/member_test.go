/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretMemberAccessType(t *testing.T) {

	t.Parallel()

	t.Run("direct", func(t *testing.T) {

		t.Run("non-optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndInterpret(t, `
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

				inter := parseCheckAndInterpret(t, `
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

				require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})

				_, err = inter.Invoke("set", value)
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})
			})
		})

		t.Run("optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndInterpret(t, `
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

				inter := parseCheckAndInterpret(t, `
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

				require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})
			})
		})
	})

	t.Run("interface", func(t *testing.T) {

		t.Run("non-optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndInterpret(t, `
                    struct interface SI {
                        var foo: Int
                    }

                    struct S: SI {
                        var foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    fun get(si: AnyStruct{SI}) {
                        si.foo
                    }

                    fun set(si: AnyStruct{SI}) {
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

				inter := parseCheckAndInterpret(t, `
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

                    fun get(si: AnyStruct{SI}) {
                        si.foo
                    }

                    fun set(si: AnyStruct{SI}) {
                        si.foo = 3
                    }
                `)

				// Construct an instance of type S2
				// by calling its constructor function of the same name

				value, err := inter.Invoke("S2")
				require.NoError(t, err)

				_, err = inter.Invoke("get", value)
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})

				_, err = inter.Invoke("set", value)
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})
			})
		})

		t.Run("optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndInterpret(t, `
                    struct interface SI {
                        let foo: Int
                    }

                    struct S: SI {
                        let foo: Int

                        init() {
                            self.foo = 1
                        }
                    }

                    fun get(si: AnyStruct{SI}?) {
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

				inter := parseCheckAndInterpret(t, `
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

                    fun get(si: AnyStruct{SI}?) {
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

				require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})
			})
		})
	})

	t.Run("reference", func(t *testing.T) {

		t.Run("non-optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndInterpret(t, `
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

				sType := checker.RequireGlobalType(t, inter.Program.Elaboration, "S")

				ref := interpreter.NewUnmeteredEphemeralReferenceValue(interpreter.UnauthorizedAccess, value, sType)

				_, err = inter.Invoke("get", ref)
				require.NoError(t, err)

				_, err = inter.Invoke("set", ref)
				require.NoError(t, err)
			})

			t.Run("invalid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndInterpret(t, `
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

				sType := checker.RequireGlobalType(t, inter.Program.Elaboration, "S")

				ref := interpreter.NewUnmeteredEphemeralReferenceValue(interpreter.UnauthorizedAccess, value, sType)

				_, err = inter.Invoke("get", ref)
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})

				_, err = inter.Invoke("set", ref)
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})
			})
		})

		t.Run("optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndInterpret(t, `
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

				sType := checker.RequireGlobalType(t, inter.Program.Elaboration, "S")

				ref := interpreter.NewUnmeteredEphemeralReferenceValue(interpreter.UnauthorizedAccess, value, sType)

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

				inter := parseCheckAndInterpret(t, `
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

				sType := checker.RequireGlobalType(t, inter.Program.Elaboration, "S")

				ref := interpreter.NewUnmeteredEphemeralReferenceValue(interpreter.UnauthorizedAccess, value, sType)

				_, err = inter.Invoke(
					"get",
					interpreter.NewUnmeteredSomeValueNonCopying(
						ref,
					),
				)
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})
			})
		})
	})
}

func TestInterpretMemberAccess(t *testing.T) {

	t.Parallel()

	t.Run("composite, field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
            struct Test {
                access(all) fun foo(): Int {
                    return 1
                }
            }

            fun test() {
                let test = Test()
                var foo: (fun(): Int) = test.foo
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("composite reference, field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

	t.Run("composite reference, primitive field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
            struct Test {
                access(all) fun foo(): Int {
                    return 1
                }
            }

            fun test() {
                let test = Test()
                let testRef = &test as &Test
                var foo: (fun(): Int) = testRef.foo
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource reference, nested", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource Foo {
                var bar: @Bar
                init() {
                    self.bar <- create Bar()
                }
                destroy() {
                    destroy self.bar
                }
            }

            resource Bar {
                var baz: @Baz
                init() {
                    self.baz <- create Baz()
                }
                destroy() {
                    destroy self.baz
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
            struct Foo {
                var array: [Int]
                init() {
                    self.array = []
                }
            }

            fun test() {
                let dict: {String: AnyStruct} = {"foo": Foo(), "bar": Foo()}
                let dictRef = &dict as auth(Mutable) &{String: AnyStruct}

                dictRef["foo"] <-> dictRef["bar"]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("entitlement map access on field", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement A
            entitlement B
            entitlement mapping M {
                A -> B
            }

            struct S {
                access(M) let foo: [String]
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

			inter := parseCheckAndInterpret(t, code)

			_, err := inter.Invoke("test")
			require.NoError(t, err)
		}

		types := []string{
			"Bar",
			"{I}",
			"[Int]",
			"{Bool: String}",
			"AnyStruct",
			"Block",
		}

		// Test all built-in composite types
		for i := interpreter.PrimitiveStaticTypeAuthAccount; i < interpreter.PrimitiveStaticType_Count; i++ {
			semaType := i.SemaType()
			types = append(types, semaType.QualifiedString())
		}

		for _, typeName := range types {
			t.Run(typeName, func(t *testing.T) {
				test(t, typeName)
			})
		}
	})
}
