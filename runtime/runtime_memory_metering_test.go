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

	"github.com/onflow/cadence/encoding/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddressLocation))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindElaboration))
		assert.Equal(t, uint64(136), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoid))
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

	for imports := range contracts {

		t.Run(fmt.Sprintf("import %d", imports), func(t *testing.T) {

			t.Parallel()

			script := "pub fun main() {}"
			for j := 0; j <= imports; j++ {
				script = importExpressions[j] + script
			}

			runtime := newTestInterpreterRuntime()

			meter := newTestMemoryGauge()

			accountCodes := map[common.LocationID][]byte{}

			runtimeInterface := &testRuntimeInterface{
				getCode: func(location Location) (bytes []byte, err error) {
					return accountCodes[location.ID()], nil
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
					accountCodes[location.ID()] = code
					return nil
				},
				getAccountContractCode: func(address Address, name string) (code []byte, err error) {
					location := common.AddressLocation{
						Address: address,
						Name:    name,
					}
					code = accountCodes[location.ID()]
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
						Source: utils.DeploymentTransaction(fmt.Sprintf("C%d", j), contracts[j]),
					},
					Context{
						Interface: runtimeInterface,
						Location:  nextTransactionLocation(),
					},
				)
				require.NoError(t, err)
				// one for each deployment transaction and one for each contract
				assert.Equal(t, uint64(2*j+2), meter.getMemory(common.MemoryKindElaboration))

				assert.Equal(t, uint64(1+j), meter.getMemory(common.MemoryKindCadenceAddress))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoid))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoid))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoid))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoid))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoid))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoid))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoid))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceSimpleType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceVoid))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindCadenceInt))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindCadenceInt))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCadenceNumber))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCadenceNumber))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindCadenceNumber))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(8), meter.getMemory(common.MemoryKindCadenceNumber))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindCadenceNumber))
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
				Location:  utils.TestLocation,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, uint64(32), meter.getMemory(common.MemoryKindCadenceNumber))
	})
}
