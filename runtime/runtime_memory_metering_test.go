/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package runtime

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
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

func TestInterpreterAddressLocationMetering(t *testing.T) {

	t.Parallel()

	t.Run("add contract", func(t *testing.T) {
		t.Parallel()

		script := `
		pub struct S {}

		pub fun main() {
			let s = CompositeType("A.0000000000000001.S")
		}
        `
		meter := newTestMemoryGauge()
		var accountCode []byte
		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			storage: newTestLedger(nil, nil),
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
				return accountCode, nil
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddressLocation))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindElaboration))
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})
}

func TestInterpreterElaborationImportMetering(t *testing.T) {

	t.Parallel()

	contracts := [...][]byte{
		[]byte(`pub contract C0 {}`),
		[]byte(`pub contract C1 {}`),
		[]byte(`pub contract C2 {}`),
		[]byte(`pub contract C3 {}`),
	}

	importExpressions := [len(contracts)]string{}
	for i := range contracts {
		importExpressions[i] = fmt.Sprintf("import C%d from 0x1\n", i)
	}

	addressValue := cadence.BytesToAddress([]byte{byte(1)})

	test := func(imports int) {
		t.Run(fmt.Sprintf("import %d", imports), func(t *testing.T) {

			t.Parallel()

			script := "pub fun main() {}"
			for j := 0; j <= imports; j++ {
				script = importExpressions[j] + script
			}

			runtime := newTestInterpreterRuntime()

			meter := newTestMemoryGauge()

			accountCodes := map[common.Location][]byte{}

			runtimeInterface := &testRuntimeInterface{
				getCode: func(location Location) (bytes []byte, err error) {
					return accountCodes[location], nil
				},
				storage: newTestLedger(nil, nil),
				getSigningAccounts: func() ([]Address, error) {
					return []Address{Address(addressValue)}, nil
				},
				resolveLocation: singleIdentifierLocationResolver(t),
				updateAccountContractCode: func(address Address, name string, code []byte) error {
					location := common.AddressLocation{
						Address: address,
						Name:    name,
					}
					accountCodes[location] = code
					return nil
				},
				getAccountContractCode: func(address Address, name string) (code []byte, err error) {
					location := common.AddressLocation{
						Address: address,
						Name:    name,
					}
					code = accountCodes[location]
					return code, nil
				},
				meterMemory: func(usage common.MemoryUsage) error {
					return meter.MeterMemory(usage)
				},
				emitEvent: func(_ cadence.Event) error {
					return nil
				},
			}

			nextTransactionLocation := newTransactionLocationGenerator()

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
					Location:  nextTransactionLocation(),
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

func TestCadenceValueAndTypeMetering(t *testing.T) {

	t.Parallel()

	t.Run("import type Int small value", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: Int) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int large value", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: Int) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

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
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int8", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: Int8) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt8(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int16", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: Int16) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt16(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int32", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: Int32) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt32(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int64", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: Int64) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt64(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int128", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: Int128) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt128(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("import type Int256", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(a: Int256) {
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewInt256(12),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoidValue))
	})

	t.Run("return value Int small value", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Int {
				let a = Int(2)
				return a
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindCadenceIntValue))
	})

	t.Run("return value Int large value", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Int {
				let a = Int(1)
				let b = a << 64
				return b
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		// TODO:
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindCadenceIntValue))
	})

	t.Run("return value Int8", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Int8 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int16", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Int16 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int32", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Int32 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int64", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Int64 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int128", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Int128 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})

	t.Run("return value Int256", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(): Int256 {
                return 12
            }
        `
		meter := newTestMemoryGauge()
		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindCadenceNumberValue))
	})
}

func TestLogFunctionStringConversionMetering(t *testing.T) {

	t.Parallel()

	testMetering := func(strLiteral string) (meteredAmount, actualLen uint64) {

		script := fmt.Sprintf(`
                pub fun main() {
                    let s = "%s"
                    log(s)
                }
            `,
			strLiteral,
		)

		var loggedString string
		var accountCode []byte

		meter := newTestMemoryGauge()

		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			storage: newTestLedger(nil, nil),
			meterMemory: func(usage common.MemoryUsage) error {
				return meter.MeterMemory(usage)
			},
			getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
				return accountCode, nil
			},
			log: func(s string) {
				loggedString = s
			},
		}

		runtime := newTestInterpreterRuntime()

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
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

func TestStorageCommitsMetering(t *testing.T) {

	t.Parallel()

	t.Run("storage used empty", func(t *testing.T) {
		t.Parallel()

		code := []byte(`
            transaction {
                prepare(signer: AuthAccount) {
                    signer.storageUsed
                }
            }
        `)

		meter := newTestMemoryGauge()

		storageUsedInvoked := false

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			meterMemory: meter.MeterMemory,
			getStorageUsed: func(_ Address) (uint64, error) {
				// Before the storageUsed function is invoked, the deltas must have been committed.
				// So the encoded slabs must have been metered at this point.
				assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
				storageUsedInvoked = true
				return 1, nil
			},
		}

		runtime := newTestInterpreterRuntime()

		err := runtime.ExecuteTransaction(
			Script{
				Source: code,
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)

		require.NoError(t, err)
		assert.True(t, storageUsedInvoked)
		assert.Equal(t, uint64(0), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
	})

	t.Run("account save", func(t *testing.T) {
		t.Parallel()

		code := []byte(`
            transaction {
                prepare(signer: AuthAccount) {
                    signer.save([[1, 2, 3], [4, 5, 6]], to: /storage/test)
                }
            }
        `)

		meter := newTestMemoryGauge()

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			meterMemory: meter.MeterMemory,
		}

		runtime := newTestInterpreterRuntime()

		err := runtime.ExecuteTransaction(
			Script{
				Source: code,
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
	})

	t.Run("storage used non empty", func(t *testing.T) {
		t.Parallel()

		code := []byte(`
            transaction {
                prepare(signer: AuthAccount) {
                    signer.save([[1, 2, 3], [4, 5, 6]], to: /storage/test)
                    signer.storageUsed
                }
            }
        `)

		meter := newTestMemoryGauge()
		storageUsedInvoked := false

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			meterMemory: meter.MeterMemory,
			getStorageUsed: func(_ Address) (uint64, error) {
				// Before the storageUsed function is invoked, the deltas must have been committed.
				// So the encoded slabs must have been metered at this point.
				assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
				storageUsedInvoked = true
				return 1, nil
			},
		}

		runtime := newTestInterpreterRuntime()

		err := runtime.ExecuteTransaction(
			Script{
				Source: code,
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)

		require.NoError(t, err)
		assert.True(t, storageUsedInvoked)
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindAtreeEncodedSlab))
	})
}

func TestMemoryMeteringErrors(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	type memoryMeter map[common.MemoryKind]uint64

	runtimeInterface := func(meter memoryMeter) *testRuntimeInterface {
		intf := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				if usage.Kind == common.MemoryKindStringValue ||
					usage.Kind == common.MemoryKindArrayValueBase ||
					usage.Kind == common.MemoryKindErrorToken {

					return testMemoryError{}
				}
				return nil
			},
		}
		intf.decodeArgument = func(b []byte, t cadence.Type) (cadence.Value, error) {
			return jsoncdc.Decode(intf, b)
		}
		return intf
	}

	executeScript := func(script []byte, meter memoryMeter, args ...cadence.Value) error {
		_, err := runtime.ExecuteScript(
			Script{
				Source:    script,
				Arguments: encodeArgs(args),
			},
			Context{
				Interface: runtimeInterface(meter),
				Location:  TestLocation,
			},
		)

		return err
	}

	t.Run("no errors", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main() {}
        `)

		err := executeScript(script, memoryMeter{})
		assert.NoError(t, err)
	})

	t.Run("importing", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: String) {}
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
            pub fun main() {
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
            pub fun main() {
                let x: [AnyStruct] = []
            }
        `)

		err := executeScript(script, memoryMeter{})
		RequireError(t, err)

		assert.ErrorIs(t, err, testMemoryError{})
	})
}
