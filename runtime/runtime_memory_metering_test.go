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
	"testing"

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
		assert.Equal(t, uint64(92), meter.getMemory(common.MemoryKindRawString))
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

	for imports := range contracts {

		t.Run(fmt.Sprintf("import %d", imports), func(t *testing.T) {

			t.Parallel()

			script := "pub fun main() {}"
			for j := 0; j <= imports; j++ {
				script = importExpressions[j] + script
			}

			runtime := newTestInterpreterRuntime()

			meter := newTestMemoryGauge()

			addressValue := cadence.BytesToAddress(meter, []byte{byte(1)})

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
