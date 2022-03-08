/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type testMemoryGauge struct {
	meter map[common.MemoryKind]uint64
}

func newTestMemoryGauge() *testMemoryGauge {
	return &testMemoryGauge{
		meter: make(map[common.MemoryKind]uint64),
	}
}

func (g *testMemoryGauge) UseMemory(usage common.MemoryUsage) {
	current, ok := g.meter[usage.Kind]
	if !ok {
		current = 0
	}
	g.meter[usage.Kind] = current + usage.Amount
}

func (g *testMemoryGauge) getMemory(kind common.MemoryKind) uint64 {
	return g.meter[kind]
}

func TestRuntimeArrayMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: [Int8] = []
                let y: [[String]] = [[]]
                let z: [[[Bool]]] = [[[]]]
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for creation of x
		// 2 for creation of y
		// 1 for transfer of y
		// 1 dynamic type check of y
		// 3 for creation of z
		// 4 for transfer of z
		// 3 for dynamic type check of z
		assert.Equal(t, uint64(15), meter.getMemory(common.MemoryKindArray))
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		script := `
	            pub fun main() {
	                let values: [[Int8]] = [[], [], []]
	                for value in values {
	                  let a = value
	                }
	            }
	        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindArray))
	})
}

func TestRuntimeDictionaryMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: {Int8: String} = {}
                let y: {String: {Int8: String}} = {"a": {}}
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindString))
		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindDictionary))
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let values: [{Int8: String}] = [{}, {}, {}]
                for value in values {
                  let a = value
                }
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(15), meter.getMemory(common.MemoryKindDictionary))
	})
}

func TestRuntimeCompositeMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct S {}

            pub resource R {
                pub let a: String
                pub let b: String

                init(a: String, b: String) {
                    self.a = a
                    self.b = b
                }
            }

            pub fun main() {
                let s = S()
                let r <- create R(a: "a", b: "b")
                destroy r
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(56), meter.getMemory(common.MemoryKindString))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindComposite))
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct S {}

            pub fun main() {
                let values = [S(), S(), S()]
                for value in values {
                  let a = value
                }
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(15), meter.getMemory(common.MemoryKindComposite))
	})
}

func TestRuntimeInterpretedFunctionMetering(t *testing.T) {
	t.Parallel()

	t.Run("top level function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInterpretedFunction))
	})

	t.Run("function pointer creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let funcPointer = fun(a: String): String {
                    return a
                }
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the main, and 1 for the anon-func
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunction))
	})

	t.Run("function pointer passing", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let funcPointer1 = fun(a: String): String {
                    return a
                }

                let funcPointer2 = funcPointer1
                let funcPointer3 = funcPointer2

                let value = funcPointer3("hello")
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the main, and 1 for the anon-func.
		// Assignment shouldn't allocate new memory, as the value is immutable and shouldn't be copied.
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunction))
	})

	t.Run("struct method", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                pub fun bar() {}
            }

            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the main, and 1 for the struct method.
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunction))
	})

	t.Run("struct init", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                init() {}
            }

            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the main, and 1 for the struct init.
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterpretedFunction))
	})
}

func TestRuntimeHostFunctionMetering(t *testing.T) {
	t.Parallel()

	t.Run("top level function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunction))
	})

	t.Run("function pointers", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let funcPointer1 = fun(a: String): String {
                    return a
                }

                let funcPointer2 = funcPointer1
                let funcPointer3 = funcPointer2

                let value = funcPointer3("hello")
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunction))
	})

	t.Run("struct method", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                pub fun bar() {}
            }

            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the struct method.
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindHostFunction))
	})

	t.Run("struct init", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                init() {}
            }

            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 1 for the struct init.
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindHostFunction))
	})

	t.Run("builtin functions", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let a = Int8(5)

                let b = CompositeType("PublicKey")
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// builtin functions are not metered
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunction))
	})

	t.Run("stdlib function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                assert(true)
            }
        `

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
			t,
			script,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(stdlib.BuiltinFunctions.ToSemaValueDeclarations()),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(stdlib.BuiltinFunctions.ToInterpreterValueDeclarations()),
				},
			},
			meter,
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// stdlib functions are not metered
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindHostFunction))
	})

	t.Run("public key creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let publicKey = PublicKey(
                    publicKey: "0102".decodeHex(),
                    signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
                )
            }
        `

		var predeclaredSemaValues []sema.ValueDeclaration
		predeclaredSemaValues = append(predeclaredSemaValues, stdlib.BuiltinFunctions.ToSemaValueDeclarations()...)
		predeclaredSemaValues = append(predeclaredSemaValues, stdlib.BuiltinValues.ToSemaValueDeclarations()...)

		var predeclaredInterpreterValues []interpreter.ValueDeclaration
		predeclaredInterpreterValues = append(
			predeclaredInterpreterValues,
			stdlib.BuiltinFunctions.ToInterpreterValueDeclarations()...,
		)
		predeclaredInterpreterValues = append(
			predeclaredInterpreterValues,
			stdlib.BuiltinValues.ToInterpreterValueDeclarations()...,
		)

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
			t,
			script,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(predeclaredSemaValues),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(predeclaredInterpreterValues),
					interpreter.WithPublicKeyValidationHandler(
						func(_ *interpreter.Interpreter, _ func() interpreter.LocationRange, _ *interpreter.CompositeValue) error {
							return nil
						},
					),
				},
			},
			meter,
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 1 host function created for 'decodeHex' of String value
		// 'publicKeyVerify' and 'publicKeyVerifyPop' functions of PublicKey value are not metered
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindHostFunction))
	})

	t.Run("multiple public key creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let publicKey1 = PublicKey(
                    publicKey: "0102".decodeHex(),
                    signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
                )

                let publicKey2 = PublicKey(
                    publicKey: "0102".decodeHex(),
                    signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
                )
            }
        `

		var predeclaredSemaValues []sema.ValueDeclaration
		predeclaredSemaValues = append(predeclaredSemaValues, stdlib.BuiltinFunctions.ToSemaValueDeclarations()...)
		predeclaredSemaValues = append(predeclaredSemaValues, stdlib.BuiltinValues.ToSemaValueDeclarations()...)

		var predeclaredInterpreterValues []interpreter.ValueDeclaration
		predeclaredInterpreterValues = append(
			predeclaredInterpreterValues,
			stdlib.BuiltinFunctions.ToInterpreterValueDeclarations()...,
		)
		predeclaredInterpreterValues = append(
			predeclaredInterpreterValues,
			stdlib.BuiltinValues.ToInterpreterValueDeclarations()...,
		)

		meter := newTestMemoryGauge()
		inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
			t,
			script,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(predeclaredSemaValues),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(predeclaredInterpreterValues),
					interpreter.WithPublicKeyValidationHandler(
						func(_ *interpreter.Interpreter, _ func() interpreter.LocationRange, _ *interpreter.CompositeValue) error {
							return nil
						},
					),
				},
			},
			meter,
		)
		require.NoError(t, err)

		_, err = inter.Invoke("main")
		require.NoError(t, err)

		// 2 = 2x 1 host function created for 'decodeHex' of String value
		// 'publicKeyVerify' and 'publicKeyVerifyPop' functions of PublicKey value are not metered
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindHostFunction))
	})
}

func TestRuntimeBoundFunctionMetering(t *testing.T) {
	t.Parallel()

	t.Run("struct method", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                pub fun bar() {}
            }

            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// No bound functions are created without usages.
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoundFunction))
	})

	t.Run("struct init", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                init() {}
            }

            pub fun main() {}
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// No bound functions are created without usages.
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBoundFunction))
	})

	t.Run("struct method usage", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct Foo {
                pub fun bar() {}
            }

            pub fun main() {
                let foo = Foo()
                foo.bar()
                foo.bar()
                foo.bar()
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// 3 bound functions are created for the 3 invocations of 'bar()'.
		// No bound functions are created for init invocation.
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindBoundFunction))
	})
}

func TestRuntimeOptionalValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("simple optional value", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: String? = "hello"
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindOptional))
	})

	t.Run("dictionary get", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: {Int8: String} = {1: "foo", 2: "bar"}
                let y = x[0]
                let z = x[1]
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindOptional))
	})
}
