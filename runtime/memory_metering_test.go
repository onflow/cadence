package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func testUseMemory(meter map[common.MemoryKind]uint64) func(common.MemoryUsage) {
	return func(usage common.MemoryUsage) {
		current, ok := meter[usage.Kind]
		if !ok {
			current = 0
		}
		meter[usage.Kind] = current + usage.Amount
	}
}

func TestRuntimeInterpretedFunctionMetering(t *testing.T) {
	t.Parallel()

	t.Run("top level function", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
		}

		script := []byte(`
            pub fun main() {}
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, uint64(1), meter[common.MemoryKindInterpretedFunction])
	})

	t.Run("function pointer creation", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
		}

		script := []byte(`
            pub fun main() {
                let funcPointer = fun(a: String): String {
                    return a
                }
            }
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// 1 for the main, and 1 for the anon-func
		assert.Equal(t, uint64(2), meter[common.MemoryKindInterpretedFunction])
	})

	t.Run("function pointer passing", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
		}

		script := []byte(`
            pub fun main() {
                let funcPointer1 = fun(a: String): String {
                    return a
                }

                let funcPointer2 = funcPointer1
                let funcPointer3 = funcPointer2

                let value = funcPointer3("hello")
            }
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// 1 for the main, and 1 for the anon-func.
		// Assignment shouldn't allocate new memory, as the value is immutable and shouldn't be copied.
		assert.Equal(t, uint64(2), meter[common.MemoryKindInterpretedFunction])
	})

	t.Run("struct method", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
		}

		script := []byte(`
            pub struct Foo {
                pub fun bar() {}
            }

            pub fun main() {}
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// 1 for the main, and 1 for the struct method.
		assert.Equal(t, uint64(2), meter[common.MemoryKindInterpretedFunction])
	})

	t.Run("struct init", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
		}

		script := []byte(`
            pub struct Foo {
                init() {}
            }

            pub fun main() {}
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// 1 for the main, and 1 for the struct init.
		assert.Equal(t, uint64(2), meter[common.MemoryKindInterpretedFunction])
	})
}

func TestRuntimeHostFunctionMetering(t *testing.T) {
	t.Parallel()

	t.Run("top level function", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
		}

		script := []byte(`
            pub fun main() {}
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, uint64(0), meter[common.MemoryKindHostFunction])
	})

	t.Run("function pointers", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
		}

		script := []byte(`
            pub fun main() {
                let funcPointer1 = fun(a: String): String {
                    return a
                }

                let funcPointer2 = funcPointer1
                let funcPointer3 = funcPointer2

                let value = funcPointer3("hello")
            }
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, uint64(0), meter[common.MemoryKindHostFunction])
	})

	t.Run("struct method", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
		}

		script := []byte(`
            pub struct Foo {
                pub fun bar() {}
            }

            pub fun main() {}
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// 1 for the struct method.
		assert.Equal(t, uint64(1), meter[common.MemoryKindHostFunction])
	})

	t.Run("struct init", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
		}

		script := []byte(`
            pub struct Foo {
                init() {}
            }

            pub fun main() {}
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// 1 for the struct init.
		assert.Equal(t, uint64(1), meter[common.MemoryKindHostFunction])
	})

	t.Run("public key creation", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
			validatePublicKey: func(publicKey *PublicKey) error {
				return nil
			},
		}

		script := []byte(`
            pub fun main() {
                let publicKey = PublicKey(
                    publicKey: "0102".decodeHex(),
                    signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
                )
            }
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// 3 host functions created for:
		//    - 'decodeHex' function of String value
		//    - 'publicKeyVerify' function of PublicKey value
		//    - 'publicKeyVerifyPop' function of PublicKey value
		assert.Equal(t, uint64(3), meter[common.MemoryKindHostFunction])
	})

	t.Run("multiple public key creation", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
			validatePublicKey: func(publicKey *PublicKey) error {
				return nil
			},
		}

		script := []byte(`
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
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// 5 = 3 host functions x 2:
		//    - 'decodeHex' function of String value
		//    - 'publicKeyVerify' function of PublicKey value
		//    - 'publicKeyVerifyPop' function of PublicKey value
		assert.Equal(t, uint64(6), meter[common.MemoryKindHostFunction])
	})

	t.Run("builtin functions", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
			validatePublicKey: func(publicKey *PublicKey) error {
				return nil
			},
		}

		script := []byte(`
            pub fun main() {
                let a = Int8(5)

                let b = CompositeType("PublicKey")
            }
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// builtin functions are not metered
		assert.Equal(t, uint64(0), meter[common.MemoryKindHostFunction])
	})

	t.Run("stdlib function", func(t *testing.T) {
		meter := make(map[common.MemoryKind]uint64)
		runtimeInterface := &testRuntimeInterface{
			storage:   newTestLedger(nil, nil),
			useMemory: testUseMemory(meter),
			validatePublicKey: func(publicKey *PublicKey) error {
				return nil
			},
		}

		script := []byte(`
            pub fun main() {
                assert(true)
            }
        `)

		runtime := newTestInterpreterRuntime()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		// stdlib functions are not metered
		assert.Equal(t, uint64(0), meter[common.MemoryKindHostFunction])
	})
}
