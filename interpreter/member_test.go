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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
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

				// The passed in argument is created with VM.
				// Therefore, the updates are only visible to VM.
				// So comparing the storage is not possible.
				inter := parseCheckAndPrepareWithoutStorageComparison(t, `
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

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly("get", value) //nolint:staticcheck
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly("set", value) //nolint:staticcheck
				RequireError(t, err)
				require.ErrorAs(t, err, &memberAccessTypeError)
			})

			t.Run("invalid, built-in", func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t, `
                    struct Foo {
                        let publicKey: [UInt8]
                        init() {
                            self.publicKey = []
                        }
                    }

                    fun get(pubKey: PublicKey) {
                        pubKey.publicKey
                    }
                `)

				// Construct an instance of type Foo
				// by calling its constructor function of the same name

				value, err := inter.Invoke("Foo")
				require.NoError(t, err)

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly("get", value) //nolint:staticcheck
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
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

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly( //nolint:staticcheck
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

				// The passed in argument is created with VM.
				// Therefore, the updates are only visible to VM.
				// So comparing the storage is not possible.
				inter := parseCheckAndPrepareWithoutStorageComparison(t, `
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

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly("get", value) //nolint:staticcheck
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly("set", value) //nolint:staticcheck
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

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly( //nolint:staticcheck
					"get",
					interpreter.NewUnmeteredSomeValueNonCopying(value),
				)
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)
			})
		})
	})

	t.Run("ephemeral reference", func(t *testing.T) {

		t.Run("non-optional", func(t *testing.T) {

			t.Run("valid", func(t *testing.T) {

				t.Parallel()

				// The passed in argument is created with VM.
				// Therefore, the updates are only visible to VM.
				// So comparing the storage is not possible.
				inter := parseCheckAndPrepareWithoutStorageComparison(t, `
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
				)

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly("get", ref) //nolint:staticcheck
				RequireError(t, err)

				var memberAccessTypeError *interpreter.MemberAccessTypeError
				require.ErrorAs(t, err, &memberAccessTypeError)

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly("set", ref) //nolint:staticcheck
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
				)

				// Intentionally passing wrong type of value
				_, err = inter.InvokeUncheckedForTestingOnly( //nolint:staticcheck
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

	t.Run("storage reference", func(t *testing.T) {

		t.Run("valid", func(t *testing.T) {

			t.Parallel()

			address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

			inter, _, _ := testAccount(t,
				address,
				true,
				nil,
				`
                struct S {
                    var foo: Int

                    init() {
                        self.foo = 1
                    }
                }

                fun getStorageRef(): auth(Mutate) &S {
                    account.storage.save(S(), to: /storage/x)
                    return account.storage.borrow<auth(Mutate) &S>(from: /storage/x)!
                }

                fun get(ref: &S) {
                    ref.foo
                }

                fun set(ref: &S) {
                    ref.foo = 2
                }
            `,
				sema.Config{},
			)

			storageRef, err := inter.Invoke("getStorageRef")
			require.NoError(t, err)
			require.IsType(t, &interpreter.StorageReferenceValue{}, storageRef)

			_, err = inter.Invoke("get", storageRef)
			require.NoError(t, err)

			_, err = inter.Invoke("set", storageRef)
			require.NoError(t, err)
		})

		t.Run("invalid", func(t *testing.T) {

			t.Parallel()

			address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

			inter, _, _ := testAccount(
				t,
				address,
				true,
				nil,
				`
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

                fun getStorageRef(): auth(Mutate) &S2 {
                    account.storage.save(S2(), to: /storage/x)
                    return account.storage.borrow<auth(Mutate) &S2>(from: /storage/x)!
                }

                fun get(ref: &S) {
                    ref.foo
                }

                fun set(ref: &S) {
                    ref.foo = 3
                }
                `,
				sema.Config{},
			)

			storageRef, err := inter.Invoke("getStorageRef")
			require.NoError(t, err)
			require.IsType(t, &interpreter.StorageReferenceValue{}, storageRef)

			// Intentionally passing wrong type of value
			_, err = inter.InvokeUncheckedForTestingOnly("get", storageRef) //nolint:staticcheck
			RequireError(t, err)
			var dereferenceError *interpreter.DereferenceError
			require.ErrorAs(t, err, &dereferenceError)
			require.Equal(t, common.TypeID("S.test.S"), dereferenceError.ExpectedType.ID())
			require.Equal(t, common.TypeID("S.test.S2"), dereferenceError.ActualType.ID())

			// Intentionally passing wrong type of value
			_, err = inter.InvokeUncheckedForTestingOnly("set", storageRef) //nolint:staticcheck
			RequireError(t, err)
			require.ErrorAs(t, err, &dereferenceError)
			require.Equal(t, common.TypeID("S.test.S"), dereferenceError.ExpectedType.ID())
			require.Equal(t, common.TypeID("S.test.S2"), dereferenceError.ActualType.ID())
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

		inter := parseCheckAndPrepare(t, `
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

		inter := parseCheckAndPrepare(t, `
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
                let authTRef = tRef as! &T
            }
            
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("array reference, authorized reference typed element", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let v1: [Int]? = [1]

                let array: [auth(Mutate) &[Int]?] = [&v1 as auth(Mutate) &[Int]?]
                let arrayRef = &array as &[auth(Mutate) &[Int]?]

                let x: &[Int]? = arrayRef[0]

                // Down-casting should fail
                let y: auth(Mutate) &[Int] = x as! auth(Mutate) &[Int]
            }
        `)

		_, err := inter.Invoke("test")
		var forceCastTypeMismatchError *interpreter.ForceCastTypeMismatchError
		require.ErrorAs(t, err, &forceCastTypeMismatchError)
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

	t.Run("non-optional field is wrapped", func(t *testing.T) {

		t.Parallel()

		// `s?.x` has type `Int?`. When boxed into `[AnyStruct]` and read back,
		// its dynamic type must remain `Int?`, not `Int`.
		inter := parseCheckAndPrepare(t, `
            struct S {
                let x: Int
                init() {
                    self.x = 7
                }
            }
            fun test(): AnyStruct {
                let s: S? = S()
                return s?.x
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(7),
			),
			value,
		)
	})

	t.Run("optional member is not double-wrapped", func(t *testing.T) {

		t.Parallel()

		// Optional chaining flattens nested optionals: `s?.x` where `x: Int?`
		// has type `Int?`, NOT `Int??`.
		inter := parseCheckAndPrepare(t, `
            struct S {
                let x: Int
                init() {
                    self.x = 7
                }
            }

            fun test(): AnyStruct {
                let s: S? = S()
                return s?.x
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(7),
			),
			value,
		)
	})

	t.Run("optional member as AnyStruct is not double-wrapped", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct S {
                let x: AnyStruct
                init() {
                    let temp: Int? = 7
                    self.x = temp
                }
            }

            fun test(): AnyStruct {
                let s: S? = S()
                return s?.x
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		// `x` is statically `AnyStruct` but holds an `Int?` (`Some(7)`) at runtime.
		// Optional chaining must flatten: `s?.x` is `Some(7)`, NOT `Some(Some(7))`.
		// The wrap decision can only be made at runtime (the static member type
		// `AnyStruct` is not an optional), which is why the VM defers it to
		// `BoxOptional`.
		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(7),
			),
			value,
		)
	})

	t.Run("enum rawValue is wrapped", func(t *testing.T) {

		t.Parallel()

		// `a?.rawValue` has type `UInt8?`.
		inter := parseCheckAndPrepare(t, `
            enum Color: UInt8 {
                case red
                case green
                case blue
            }

            fun test(): AnyStruct {
                let a = Color(rawValue: 2)
                return a?.rawValue
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredUInt8Value(2),
			),
			value,
		)
	})

	t.Run("method call non optional is wrapped", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct S {
                fun getX(): Int {
                   return 7
                }
            }

            fun test(): AnyStruct {
                var s: S? = S()
                return s?.getX()
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(7),
			),
			value,
		)
	})

	t.Run("method call optional is double wrapped", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct S {
                fun getX(): Int? {
                   return 7
                }
            }

            fun test(): AnyStruct {
                var s: S? = S()
                return s?.getX()
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredSomeValueNonCopying(
					interpreter.NewUnmeteredIntValueFromInt64(7),
				),
			),
			value,
		)
	})
}

func TestInterpretMultiLevelOptionalChaining(t *testing.T) {

	t.Parallel()

	// Two-level chaining: `foo?.bar?.id`.
	// `bar` is itself an optional field, so the chain can short-circuit
	// either at the root (`foo` is nil) or at the intermediate link
	// (`foo.bar` is nil). Both must produce `nil` without evaluating
	// the remainder of the chain.

	const twoLevelFieldCode = `
        struct Bar {
            var id: Int
            init(_ id: Int) {
                self.id = id
            }
        }

        struct Foo {
            var bar: Bar?
            init(bar: Bar?) {
                self.bar = bar
            }
        }

        fun test(makeFoo: Bool, makeBar: Bool): Int? {
            var foo: Foo? = nil
            if makeFoo {
                var bar: Bar? = nil
                if makeBar {
                    bar = Bar(5)
                }
                foo = Foo(bar: bar)
            }
            return foo?.bar?.id
        }
    `

	t.Run("two-level field, all non-nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, twoLevelFieldCode)

		value, err := inter.Invoke(
			"test",
			interpreter.TrueValue,
			interpreter.TrueValue,
		)
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(5),
			),
			value,
		)
	})

	t.Run("two-level field, intermediate nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, twoLevelFieldCode)

		// `foo` is non-nil, but `foo.bar` is nil:
		// the chain must short-circuit at the intermediate link.
		value, err := inter.Invoke(
			"test",
			interpreter.TrueValue,
			interpreter.FalseValue,
		)
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	t.Run("two-level field, root nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, twoLevelFieldCode)

		value, err := inter.Invoke(
			"test",
			interpreter.FalseValue,
			interpreter.FalseValue,
		)
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	// Two-level chaining ending in a method call: `foo?.bar?.getID()`.
	// When the intermediate link is nil, the method must not be invoked.

	const twoLevelMethodCode = `
        struct Bar {
            var id: Int
            init(_ id: Int) {
                self.id = id
            }
            fun getID(): Int {
                return self.id
            }
        }

        struct Foo {
            var bar: Bar?
            init(bar: Bar?) {
                self.bar = bar
            }
        }

        fun test(makeBar: Bool): Int? {
            var bar: Bar? = nil
            if makeBar {
                bar = Bar(7)
            }
            let foo: Foo? = Foo(bar: bar)
            return foo?.bar?.getID()
        }
    `

	t.Run("two-level method, all non-nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, twoLevelMethodCode)

		value, err := inter.Invoke("test", interpreter.TrueValue)
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(7),
			),
			value,
		)
	})

	t.Run("two-level method, intermediate nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, twoLevelMethodCode)

		value, err := inter.Invoke("test", interpreter.FalseValue)
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	// Three-level chaining: `a?.b?.c?.id`, exercising short-circuiting
	// at every link of the chain.

	const threeLevelCode = `
        struct C {
            var id: Int
            init(_ id: Int) {
                self.id = id
            }
        }

        struct B {
            var c: C?
            init(c: C?) {
                self.c = c
            }
        }

        struct A {
            var b: B?
            init(b: B?) {
                self.b = b
            }
        }

        fun test(makeA: Bool, makeB: Bool, makeC: Bool): Int? {
            var a: A? = nil
            if makeA {
                var b: B? = nil
                if makeB {
                    var c: C? = nil
                    if makeC {
                        c = C(9)
                    }
                    b = B(c: c)
                }
                a = A(b: b)
            }
            return a?.b?.c?.id
        }
    `

	t.Run("three-level, all non-nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, threeLevelCode)

		value, err := inter.Invoke(
			"test",
			interpreter.TrueValue,
			interpreter.TrueValue,
			interpreter.TrueValue,
		)
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(9),
			),
			value,
		)
	})

	t.Run("three-level, last link nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, threeLevelCode)

		value, err := inter.Invoke(
			"test",
			interpreter.TrueValue,
			interpreter.TrueValue,
			interpreter.FalseValue,
		)
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	t.Run("three-level, middle link nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, threeLevelCode)

		value, err := inter.Invoke(
			"test",
			interpreter.TrueValue,
			interpreter.FalseValue,
			interpreter.FalseValue,
		)
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	t.Run("three-level, root nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, threeLevelCode)

		value, err := inter.Invoke(
			"test",
			interpreter.FalseValue,
			interpreter.FalseValue,
			interpreter.FalseValue,
		)
		require.NoError(t, err)

		AssertValuesEqual(t, inter, interpreter.Nil, value)
	})

	// Multi-level chaining combined with the nil-coalescing operator:
	// `foo?.bar?.id ?? -1`. Short-circuiting to nil must fall through
	// to the default value.

	const chainWithCoalesceCode = `
        struct Bar {
            var id: Int
            init(_ id: Int) {
                self.id = id
            }
        }

        struct Foo {
            var bar: Bar?
            init(bar: Bar?) {
                self.bar = bar
            }
        }

        fun test(makeBar: Bool): Int {
            var bar: Bar? = nil
            if makeBar {
                bar = Bar(42)
            }
            let foo: Foo? = Foo(bar: bar)
            return foo?.bar?.id ?? -1
        }
    `

	t.Run("chain with nil-coalescing, present", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, chainWithCoalesceCode)

		value, err := inter.Invoke("test", interpreter.TrueValue)
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(42),
			value,
		)
	})

	t.Run("chain with nil-coalescing, short-circuit to default", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, chainWithCoalesceCode)

		value, err := inter.Invoke("test", interpreter.FalseValue)
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(-1),
			value,
		)
	})
}
