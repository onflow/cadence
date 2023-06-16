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
