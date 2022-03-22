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

func (g *testMemoryGauge) MeterMemory(usage common.MemoryUsage) error {
	g.meter[usage.Kind] += usage.Amount
	return nil
}

func (g *testMemoryGauge) getMemory(kind common.MemoryKind) uint64 {
	return g.meter[kind]
}

func TestInterpretArrayMetering(t *testing.T) {

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
		// 14 from value transfer
		assert.Equal(t, uint64(29), meter.getMemory(common.MemoryKindArray))
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

		assert.Equal(t, uint64(33), meter.getMemory(common.MemoryKindArray))
	})
}

func TestInterpretDictionaryMetering(t *testing.T) {
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
		assert.Equal(t, uint64(10), meter.getMemory(common.MemoryKindDictionary))
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

		assert.Equal(t, uint64(30), meter.getMemory(common.MemoryKindDictionary))
	})
}

func TestInterpretCompositeMetering(t *testing.T) {
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

		assert.Equal(t, uint64(14), meter.getMemory(common.MemoryKindString))
		assert.Equal(t, uint64(51), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindComposite))
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

		assert.Equal(t, uint64(30), meter.getMemory(common.MemoryKindComposite))
	})
}

func TestInterpretCompositeFieldMetering(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		script := `
		            pub struct S {}
		            pub fun main() {
		                let s = S()
		            }
		        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindComposite))
	})

	t.Run("1 field", func(t *testing.T) {
		t.Parallel()

		script := `
	            pub struct S {
					pub let a: String
	                init(_ a: String) {
	                    self.a = a
	                }
				}
	            pub fun main() {
	                let s = S("a")
	            }
	        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(11), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindComposite))
	})

	t.Run("2 field", func(t *testing.T) {
		t.Parallel()

		script := `
            pub struct S {
				pub let a: String
				pub let b: String
                init(_ a: String, _ b: String) {
                    self.a = a
					self.b = b
                }
			}
            pub fun main() {
                let s = S("a", "b")
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindComposite))
	})
}

func TestInterpretInterpretedFunctionMetering(t *testing.T) {
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

func TestInterpretHostFunctionMetering(t *testing.T) {
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

func TestInterpretBoundFunctionMetering(t *testing.T) {
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

func TestInterpretOptionalValueMetering(t *testing.T) {
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

func TestInterpretIntMetering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 16 = max(8, 8) + 8
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 16 = 8 + 8
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 16 = 8 + 8
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1
                let y = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8
		// result: 8 = 8
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretUIntMetering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt + 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 16 = max(8, 8) + 8
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt - 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt).saturatingSubtract(2 as UInt)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt * 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 16 = 8 + 8
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt / 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt % 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt | 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt ^ 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt & 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt << 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 16 = 8 + 8
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt >> 2 as UInt
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8 + 8
		// result: 8 = max(8, 8)
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1
                let y = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// TODO:
		// creation: 8
		// result: 8 = 8
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretUInt8Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt8 + 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt8).saturatingAdd(2 as UInt8)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt8 - 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt8).saturatingSubtract(2 as UInt8)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt8 * 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt8).saturatingMultiply(2 as UInt8)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 / 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 % 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 | 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 ^ 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 & 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 << 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt8 >> 2 as UInt8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

}

func TestInterpretUInt16Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt16 + 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt16).saturatingAdd(2 as UInt16)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt16 - 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt16).saturatingSubtract(2 as UInt16)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt16 * 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt16).saturatingMultiply(2 as UInt16)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 / 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 % 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 | 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 ^ 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 & 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 << 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt16 >> 2 as UInt16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretUInt32Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt32 + 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt32).saturatingAdd(2 as UInt32)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt32 - 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt32).saturatingSubtract(2 as UInt32)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt32 * 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt32).saturatingMultiply(2 as UInt32)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 / 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 % 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 | 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 ^ 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 & 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 << 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt32 >> 2 as UInt32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretUInt64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt64 + 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt64).saturatingAdd(2 as UInt64)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt64 - 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt64).saturatingSubtract(2 as UInt64)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt64 * 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt64).saturatingMultiply(2 as UInt64)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 / 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 % 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 | 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 ^ 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 & 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 << 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt64 >> 2 as UInt64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretUInt128Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt128 + 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt128).saturatingAdd(2 as UInt128)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt128 - 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt128).saturatingSubtract(2 as UInt128)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt128 * 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt128).saturatingMultiply(2 as UInt128)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 / 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 % 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 | 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 ^ 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 & 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 << 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt128 >> 2 as UInt128
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretUInt256Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt256 + 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt256).saturatingAdd(2 as UInt256)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as UInt256 - 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt256).saturatingSubtract(2 as UInt256)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as UInt256 * 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = (1 as UInt256).saturatingMultiply(2 as UInt256)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 / 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 % 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 | 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 ^ 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 & 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 << 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as UInt256 >> 2 as UInt256
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretInt8Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
                let y: Int8 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
                let y: Int8 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
                let y: Int8 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two operands (literals): 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3
                let y: Int8 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 1
                let y: Int8 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 1
		// y: 1
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int8 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretInt16Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
                let y: Int16 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
                let y: Int16 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
                let y: Int16 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3
                let y: Int16 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 1
                let y: Int16 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 2
		// y: 2
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int16 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretInt32Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
                let y: Int32 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
                let y: Int32 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
                let y: Int32 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3
                let y: Int32 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 1
                let y: Int32 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 4
		// y: 4
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int32 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretInt64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
                let y: Int64 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
                let y: Int64 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
                let y: Int64 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3
                let y: Int64 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 1
                let y: Int64 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 8
		// y: 8
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int64 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretInt128Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
                let y: Int128 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
                let y: Int128 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
                let y: Int128 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3
                let y: Int128 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 1
                let y: Int128 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 16
		// y: 16
		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int128 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 16 + 16
		// result: 16
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretInt256Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1 + 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
                let y: Int256 = x.saturatingAdd(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1 - 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
                let y: Int256 = x.saturatingSubtract(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1 * 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
                let y: Int256 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 / 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3
                let y: Int256 = x.saturatingMultiply(2)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 % 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 1
                let y: Int256 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 32
		// y: 32
		assert.Equal(t, uint64(64), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 | 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 ^ 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 & 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise left shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 << 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})

	t.Run("bitwise right shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Int256 = 3 >> 2
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 32 + 32
		// result: 32
		assert.Equal(t, uint64(96), meter.getMemory(common.MemoryKindBigInt))
	})
}

func TestInterpretWord8Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word8 + 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as Word8 - 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word8 * 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 / 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 % 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 | 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 ^ 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 & 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 << 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word8 >> 2 as Word8
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 1 + 1
		// result: 1
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindNumber))
	})

}

func TestInterpretWord16Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word16 + 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as Word16 - 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word16 * 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 / 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 % 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 | 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 ^ 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 & 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 << 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word16 >> 2 as Word16
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 2 + 2
		// result: 2
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretWord32Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word32 + 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as Word32 - 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word32 * 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 / 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 % 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 | 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 ^ 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 & 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 << 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word32 >> 2 as Word32
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 4 + 4
		// result: 4
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretWord64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8
		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word64 + 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 3 as Word64 - 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 1 as Word64 * 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 / 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 % 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise or", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 | 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise xor", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 ^ 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise and", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 & 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise left-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 << 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("bitwise right-shift", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x = 10 as Word64 >> 2 as Word64
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// creation: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretBoolMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: Bool = true
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindBool))
	})

	t.Run("negation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                !true
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindBool))
	})

	t.Run("equality, true", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                true == true
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindBool))
	})

	t.Run("equality, false", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                true == false
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindBool))
	})

	t.Run("inequality", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                true != false
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindBool))
	})
}

func TestInterpretNilMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: Bool? = nil
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNil))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindBool))
	})
}

func TestInterpretVoidMetering(t *testing.T) {
	t.Parallel()

	t.Run("returnless function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindVoid))
	})

	t.Run("returning function", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Bool {
                return true
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindVoid))
	})
}

func TestInterpretStorageReferenceValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
              resource R {}

              pub fun main(account: AuthAccount) {
                  account.borrow<&R>(from: /storage/r)
              }
            `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		account := newTestAuthAccountValue(inter, interpreter.AddressValue{})
		_, err := inter.Invoke("main", account)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindStorageReferenceValue))
	})
}

func TestInterpretEphemeralReferenceValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
          resource R {}

          pub fun main(): &Int {
              let x: Int = 1
              let y = &x as &Int
              return y
          }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindEphemeralReferenceValue))
	})

	t.Run("creation, optional", func(t *testing.T) {
		t.Parallel()

		script := `
          resource R {}

          pub fun main(): &Int {
              let x: Int? = 1
              let y = &x as &Int?
              return y!
          }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindEphemeralReferenceValue))
	})
}

func TestInterpretCharacterMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: Character = "a"
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// The lexer meters the literal "a" as a string.
		// To avoid double-counting, it is NOT metered as a Character as well.
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCharacter))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindString))
	})

	t.Run("assignment", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: Character = "a"
                let y = x
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// The lexer meters the literal "a" as a string.
		// To avoid double-counting, it is NOT metered as a Character as well.
		// Since characters are immutable, assigning them also does not allocate memory for them.
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCharacter))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindString))
	})

	t.Run("from string GetKey", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: String = "a"
                let y: Character = x[0]
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCharacter))
	})
}

func TestInterpretAddressValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x: Address = 0x0
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddress))
	})

	t.Run("convert", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x = Address(0x0)
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddress))
	})
}

func TestInterpretPathValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x = /public/bar
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPathValue))
	})

	t.Run("convert", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main() {
                let x = PublicPath(identifier: "bar")
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPathValue))
	})
}

func TestInterpretCapabilityValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            resource R {}

            pub fun main(account: AuthAccount) {
                let r <- create R()
                account.save(<-r, to: /storage/r)
                let x = account.link<&R>(/public/capo, target: /storage/r)
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		account := newTestAuthAccountValue(inter, interpreter.AddressValue{})
		_, err := inter.Invoke("main", account)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCapabilityValue))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindPathValue))
	})
}

func TestInterpretLinkValueMetering(t *testing.T) {
	t.Parallel()

	t.Run("creation", func(t *testing.T) {
		t.Parallel()

		script := `
            resource R {}

            pub fun main(account: AuthAccount) {
                account.link<&R>(/public/capo, target: /private/p)
            }
        `
		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		account := newTestAuthAccountValue(inter, interpreter.AddressValue{})
		_, err := inter.Invoke("main", account)
		require.NoError(t, err)

		// Metered twice only when Atree validation is enabled.
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindLinkValue))
	})
}

func TestInterpretFix64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4 + 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
                let y: Fix64 = x.saturatingAdd(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4 - 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
                let y: Fix64 = x.saturatingSubtract(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4 * 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
                let y: Fix64 = x.saturatingMultiply(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 3.4 / 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 3.4
                let y: Fix64 = x.saturatingMultiply(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 3.4 % 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// quotient (div) : 8
		// truncatedQuotient: 8
		// truncatedQuotient.Mul(o): 8
		// result: 8
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("negation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: Fix64 = 1.4
                let y: Fix64 = -x
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// x: 8
		// y: 8
		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("creation as supertype", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: FixedPoint = -1.4
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumber))
	})
}

func TestInterpretUFix64Metering(t *testing.T) {

	t.Parallel()

	t.Run("creation", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4 + 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating addition", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4
                let y: UFix64 = x.saturatingAdd(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 2.5 - 1.4 
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating subtraction", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4
                let y: UFix64 = x.saturatingSubtract(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4 * 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating multiplication", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 1.4
                let y: UFix64 = x.saturatingMultiply(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 3.4 / 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("saturating division", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 3.4
                let y: UFix64 = x.saturatingMultiply(2.5)
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// result: 8
		assert.Equal(t, uint64(24), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("modulo", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: UFix64 = 3.4 % 2.5
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		// two literals: 8 + 8
		// quotient (div) : 8
		// truncatedQuotient: 8
		// truncatedQuotient.Mul(o): 8
		// result: 8
		assert.Equal(t, uint64(48), meter.getMemory(common.MemoryKindNumber))
	})

	t.Run("creation as supertype", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main() {
                let x: FixedPoint = 1.4
            }
        `

		meter := newTestMemoryGauge()
		inter := parseCheckAndInterpretWithMemoryMetering(t, script, meter)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindNumber))
	})
}
