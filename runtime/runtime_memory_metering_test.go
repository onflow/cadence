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

package runtime_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
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

func TestRuntimeInterpreterAddressLocationMetering(t *testing.T) {

	t.Parallel()

	t.Run("add contract", func(t *testing.T) {
		t.Parallel()

		script := `
		access(all) struct S {}

		access(all) fun main() {
			let s = CompositeType("A.0000000000000001.S")
		}
        `
		meter := newTestMemoryGauge()
		var accountCode []byte
		runtimeInterface := &TestRuntimeInterface{
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			Storage:       NewTestLedger(nil, nil),
			OnMeterMemory: meter.MeterMemory,
			OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
				return accountCode, nil
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddressLocation))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindElaboration))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})
}

func TestRuntimeInterpreterElaborationImportMetering(t *testing.T) {

	t.Parallel()

	contracts := [...][]byte{
		[]byte(`access(all) contract C0 {}`),
		[]byte(`access(all) contract C1 {}`),
		[]byte(`access(all) contract C2 {}`),
		[]byte(`access(all) contract C3 {}`),
	}

	importExpressions := [len(contracts)]string{}
	for i := range contracts {
		importExpressions[i] = fmt.Sprintf("import C%d from 0x1\n", i)
	}

	addressValue := cadence.BytesToAddress([]byte{byte(1)})

	test := func(imports int) {
		t.Run(fmt.Sprintf("import %d", imports), func(t *testing.T) {

			t.Parallel()

			script := "access(all) fun main() {}"
			for j := 0; j <= imports; j++ {
				script = importExpressions[j] + script
			}

			runtime := NewTestInterpreterRuntime()

			meter := newTestMemoryGauge()

			accountCodes := map[common.Location][]byte{}

			runtimeInterface := &TestRuntimeInterface{
				OnGetCode: func(location Location) (bytes []byte, err error) {
					return accountCodes[location], nil
				},
				Storage: NewTestLedger(nil, nil),
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{Address(addressValue)}, nil
				},
				OnResolveLocation: NewSingleIdentifierLocationResolver(t),
				OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					code = accountCodes[location]
					return code, nil
				},
				OnMeterMemory: func(usage common.MemoryUsage) error {
					return meter.MeterMemory(usage)
				},
				OnEmitEvent: func(_ cadence.Event) error {
					return nil
				},
			}

			nextTransactionLocation := NewTransactionLocationGenerator()

			for j := 0; j <= imports; j++ {
				err := runtime.ExecuteTransaction(
					Script{
						Source: DeploymentTransaction(fmt.Sprintf("C%d", j), contracts[j]),
					},
					Context{
						Interface: runtimeInterface,
						Location:  nextTransactionLocation(),
					},
				)
				require.NoError(t, err)
				// one for each deployment transaction and one for each contract
				assert.Equal(t, uint64(2*j+2), meter.getMemory(common.MemoryKindElaboration))

				assert.Equal(t, uint64(1+j), meter.getMemory(common.MemoryKindCadenceAddressValue))
			}

			_, err := runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)
			require.NoError(t, err)

			// in addition to the elaborations metered above, we also meter
			// one more for the script and one more for each contract imported
			assert.Equal(t, uint64(3*imports+4), meter.getMemory(common.MemoryKindElaboration))
		})
	}

	for imports := range contracts {
		test(imports)
	}
}

func TestRuntimeCadenceValueAndTypeMetering(t *testing.T) {

	t.Parallel()

	t.Run("import type Int small value", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(a: Int) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int large value", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(a: Int) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		largeBigInt := &big.Int{}
		largeBigInt.Exp(big.NewInt(2<<33), big.NewInt(6), nil)
		largeInt := cadence.NewInt(0)
		largeInt.Value = largeBigInt

		fmt.Println(largeInt.String())

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					largeInt,
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int8", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(a: Int8) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt8(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int16", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(a: Int16) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt16(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int32", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(a: Int32) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt32(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int64", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(a: Int64) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt64(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int128", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(a: Int128) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt128(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int256", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(a: Int256) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt256(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("return value Int small value", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(): Int {
				let a = Int(2)
				return a
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindCadenceIntValue))
	})

	t.Run("return value Int large value", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(): Int {
				let a = Int(1)
				let b = a << 64
				return b
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindCadenceIntValue))
	})

	t.Run("return value Int8", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(): Int8 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int16", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(): Int16 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int32", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(): Int32 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int64", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(): Int64 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int128", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(): Int128 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int256", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(): Int256 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: meter.MeterMemory,
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})
}

func TestRuntimeLogFunctionStringConversionMetering(t *testing.T) {

	t.Parallel()

	testMetering := func(strLiteral string) (meteredAmount, actualLen uint64) {

		script := fmt.Sprintf(`
                access(all) fun main() {
                    let s = "%s"
                    log(s)
                }
            `,
			strLiteral,
		)

		var loggedString string
		var accountCode []byte

		meter := newTestMemoryGauge()

		runtimeInterface := &TestRuntimeInterface{
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			Storage:       NewTestLedger(nil, nil),
			OnMeterMemory: meter.MeterMemory,
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCode, nil
			},
			OnProgramLog: func(s string) {
				loggedString = s
			},
		}

		runtime := NewTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		return meter.getMemory(common.MemoryKindRawString), uint64(len(loggedString))
	}

	emptyStrMeteredAmount, emptyStrActualLen := testMetering("")
	nonEmptyStrMeteredAmount, nonEmptyStrActualLen := testMetering("Hello, World!")

	// Compare the diffs, to eliminate the other raw-strings metered (a.g: as part of AST)
	diffOfActualLen := nonEmptyStrActualLen - emptyStrActualLen
	diffOfMeteredAmount := nonEmptyStrMeteredAmount - emptyStrMeteredAmount

	assert.Equal(t, diffOfActualLen, diffOfMeteredAmount)
}

func TestRuntimeStorageCommitsMetering(t *testing.T) {

	t.Parallel()

	t.Run("storage used empty", func(t *testing.T) {
		t.Parallel()

		code := []byte(`
            transaction {
                prepare(signer: &Account) {
                    signer.storage.used
                }
            }
        `)

		meter := newTestMemoryGauge()

		storageUsedInvoked := false

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			OnMeterMemory: meter.MeterMemory,
			OnGetStorageUsed: func(_ Address) (uint64, error) {
				// Before the storageUsed function is invoked, the deltas must have been committed.
				// So the encoded slabs must have been metered at this point.
				assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
				storageUsedInvoked = true
				return 1, nil
			},
		}

		runtime := NewTestInterpreterRuntime()

		err := runtime.ExecuteTransaction(
			Script{
				Source: code,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)

		require.NoError(t, err)
		assert.True(t, storageUsedInvoked)
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
	})

	t.Run("account.storage.save", func(t *testing.T) {
		t.Parallel()

		code := []byte(`
            transaction {
                prepare(signer: auth(Storage) &Account) {
                    signer.storage.save([[1, 2, 3], [4, 5, 6]], to: /storage/test)
                }
            }
        `)

		meter := newTestMemoryGauge()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			OnMeterMemory: meter.MeterMemory,
		}

		runtime := NewTestInterpreterRuntime()

		err := runtime.ExecuteTransaction(
			Script{
				Source: code,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)

		require.NoError(t, err)
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
	})

	t.Run("storage used non empty", func(t *testing.T) {
		t.Parallel()

		code := []byte(`
            transaction {
                prepare(signer: auth(Storage) &Account) {
                    signer.storage.save([[1, 2, 3], [4, 5, 6]], to: /storage/test)
                    signer.storage.used
                }
            }
        `)

		meter := newTestMemoryGauge()
		storageUsedInvoked := false

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			OnMeterMemory: meter.MeterMemory,
			OnGetStorageUsed: func(_ Address) (uint64, error) {
				// Before the storageUsed function is invoked, the deltas must have been committed.
				// So the encoded slabs must have been metered at this point.
				assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
				storageUsedInvoked = true
				return 1, nil
			},
		}

		runtime := NewTestInterpreterRuntime()

		err := runtime.ExecuteTransaction(
			Script{
				Source: code,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)

		require.NoError(t, err)
		assert.True(t, storageUsedInvoked)
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
	})
}

func TestRuntimeMemoryMeteringErrors(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	type memoryMeter map[common.MemoryKind]uint64

	runtimeInterface := func(meter memoryMeter) *TestRuntimeInterface {
		return &TestRuntimeInterface{
			OnMeterMemory: func(usage common.MemoryUsage) error {
				if usage.Kind == common.MemoryKindStringValue ||
					usage.Kind == common.MemoryKindArrayValueBase ||
					usage.Kind == common.MemoryKindErrorToken {

					return testMemoryError{}
				}
				return nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}
	}

	nextScriptLocation := NewScriptLocationGenerator()

	executeScript := func(script []byte, meter memoryMeter, args ...cadence.Value) error {
		_, err := runtime.ExecuteScript(
			Script{
				Source:    script,
				Arguments: encodeArgs(args),
			},
			Context{
				Interface: runtimeInterface(meter),
				Location:  nextScriptLocation(),
			},
		)

		return err
	}

	t.Run("no errors", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun main() {}
        `)

		err := executeScript(script, memoryMeter{})
		assert.NoError(t, err)
	})

	t.Run("importing", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun main(x: String) {}
        `)

		err := executeScript(
			script,
			memoryMeter{},
			cadence.String("hello"),
		)
		RequireError(t, err)

		assert.ErrorIs(t, err, testMemoryError{})
	})

	t.Run("at lexer", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun main() {
                0b
            }
        `)

		err := executeScript(script, memoryMeter{})

		require.IsType(t, Error{}, err)
		runtimeError := err.(Error)

		require.IsType(t, errors.MemoryError{}, runtimeError.Err)
		fatalError := runtimeError.Err.(errors.MemoryError)

		assert.Contains(t, fatalError.Error(), "memory limit exceeded")
	})

	t.Run("at interpreter", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun main() {
                let x: [AnyStruct] = []
            }
        `)

		err := executeScript(script, memoryMeter{})
		RequireError(t, err)

		assert.ErrorIs(t, err, testMemoryError{})
	})
}

func TestRuntimeMeterEncoding(t *testing.T) {

	t.Parallel()

	t.Run("string", func(t *testing.T) {

		t.Parallel()

		config := DefaultTestInterpreterConfig
		config.AtreeValidationEnabled = false
		rt := NewTestInterpreterRuntimeWithConfig(config)

		address := common.MustBytesToAddress([]byte{0x1})
		storage := NewTestLedger(nil, nil)
		meter := newTestMemoryGauge()

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnMeterMemory: meter.MeterMemory,
		}

		text := "A quick brown fox jumps over the lazy dog"

		err := rt.ExecuteTransaction(
			Script{
				Source: []byte(fmt.Sprintf(`
                transaction() {
                    prepare(acc: auth(Storage) &Account) {
                        var s = "%s"
                        acc.storage.save(s, to:/storage/some_path)
                    }
                }`,
					text,
				)),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)

		require.NoError(t, err)
		assert.Equal(t, 87, int(meter.getMemory(common.MemoryKindBytes)))
	})

	t.Run("string in loop", func(t *testing.T) {

		t.Parallel()

		config := DefaultTestInterpreterConfig
		config.AtreeValidationEnabled = false
		rt := NewTestInterpreterRuntimeWithConfig(config)

		address := common.MustBytesToAddress([]byte{0x1})
		storage := NewTestLedger(nil, nil)
		meter := newTestMemoryGauge()

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnMeterMemory: meter.MeterMemory,
		}

		text := "A quick brown fox jumps over the lazy dog"

		err := rt.ExecuteTransaction(
			Script{
				Source: []byte(fmt.Sprintf(`
                transaction() {
                    prepare(acc: auth(Storage) &Account) {
                        var i = 0
                        var s = "%s"
                        while i<1000 {
                            let path = StoragePath(identifier: "i".concat(i.toString()))!
                            acc.storage.save(s, to: path)
                            i=i+1
                        }
                    }
                }`,
					text,
				)),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)

		require.NoError(t, err)
		assert.Equal(t, 62787, int(meter.getMemory(common.MemoryKindBytes)))
	})

	t.Run("composite", func(t *testing.T) {

		t.Parallel()

		config := DefaultTestInterpreterConfig
		config.AtreeValidationEnabled = false
		rt := NewTestInterpreterRuntimeWithConfig(config)

		address := common.MustBytesToAddress([]byte{0x1})
		storage := NewTestLedger(nil, nil)
		meter := newTestMemoryGauge()

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnMeterMemory: meter.MeterMemory,
		}

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(`
                access(all) fun main() {
                    let acc = getAuthAccount<auth(Storage) &Account>(0x02)
                    var i = 0
                    var f = Foo()
                    while i<1000 {
                        let path = StoragePath(identifier: "i".concat(i.toString()))!
                        acc.storage.save(f, to: path)
                        i=i+1
                    }
                }

                access(all) struct Foo {
                    access(self) var id: Int
                    init() {
                        self.id = 123456789
                    }
                }`),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.NoError(t, err)
		assert.Equal(t, 76941, int(meter.getMemory(common.MemoryKindBytes)))
	})
}
