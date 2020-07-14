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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeHighLevelStorage(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address := common.BytesToAddress([]byte{0xCA, 0xDE})
	contract := []byte(`
       pub contract Test {

           pub resource R {
               pub var i: Int

               init(_ i: Int) {
                   self.i = i
               }

               pub fun update(_ i: Int) {
                   self.i = i
               }
           }

           pub fun createR(_ i: Int): @R {
               return <-create R(i)
           }
       }
    `)

	deployTx := utils.DeploymentTransaction(contract)

	setupTx := []byte(`
	  import Test from 0xCADE

	  transaction {

	      prepare(signer: AuthAccount) {
	          let rs <- {
	             "r1": <- Test.createR(3),
	             "r2": <- Test.createR(4)
	          }
	          signer.save(<-rs, to: /storage/rs)
	      }
	   }
	`)

	changeTx := []byte(`
	  import Test from 0xCADE

	  transaction {

	      prepare(signer: AuthAccount) {
	          let rs = signer.borrow<&{String: Test.R}>(from: /storage/rs)!
              rs["r1"]?.update(5)
	      }
	   }
	`)

	var accountCode []byte
	var events []cadence.Event

	type write struct {
		owner common.Address
		key   string
		value cadence.Value
	}

	var writes []write

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() []Address {
			return []Address{address}
		},
		updateAccountCode: func(address Address, code []byte) (err error) {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) {
			events = append(events, event)
		},
		setCadenceValue: func(owner Address, key string, value cadence.Value) (err error) {
			writes = append(writes, write{
				owner: owner,
				key:   key,
				value: value,
			})
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	writes = nil

	err := runtime.ExecuteTransaction(deployTx, nil, runtimeInterface, nextTransactionLocation())
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	rType := cadence.ResourceType{
		TypeID:     "A.000000000000cade.Test.R",
		Identifier: "R",
		Fields: []cadence.Field{
			{
				Identifier: "i",
				Type:       cadence.IntType{},
			},
			{
				Identifier: "update",
				Type: cadence.Function{
					Parameters: []cadence.Parameter{
						{
							Label:      "_",
							Identifier: "i",
							Type:       cadence.IntType{},
						},
					},
					ReturnType: cadence.VoidType{},
				}.WithID("((Int):Void)"),
			},
			{
				Identifier: "uuid",
				Type:       cadence.UInt64Type{},
			},
		},
	}

	assert.Equal(t,
		[]write{
			{
				address,
				"contract",
				cadence.NewContract([]cadence.Value{}).WithType(cadence.ContractType{
					TypeID:     "A.000000000000cade.Test",
					Identifier: "Test",
					Fields: []cadence.Field{
						{
							Identifier: "createR",
							Type: cadence.Function{
								Parameters: []cadence.Parameter{
									{
										Label:      "_",
										Identifier: "i",
										Type:       cadence.IntType{},
									},
								},
								ReturnType: rType,
							}.WithID("((Int):A.000000000000cade.Test.R)"),
						},
					},
					Initializers: nil,
				}),
			},
		},
		writes,
	)

	writes = nil

	err = runtime.ExecuteTransaction(setupTx, nil, runtimeInterface, nextTransactionLocation())
	require.NoError(t, err)

	assert.Equal(t,
		[]write{
			{
				address,
				"storage\x1frs",
				cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key: cadence.NewString("r1"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewInt(3),
							cadence.NewUInt64(0),
						}).WithType(rType),
					},
					{
						Key: cadence.NewString("r2"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewInt(4),
							cadence.NewUInt64(0),
						}).WithType(rType),
					},
				}),
			},
		},
		writes,
	)

	writes = nil

	err = runtime.ExecuteTransaction(changeTx, nil, runtimeInterface, nextTransactionLocation())
	require.NoError(t, err)

	assert.Equal(t,
		[]write{
			{
				address,
				"storage\x1frs",
				cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key: cadence.NewString("r1"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewInt(5),
							cadence.NewUInt64(0),
						}).WithType(rType),
					},
					{
						Key: cadence.NewString("r2"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewInt(4),
							cadence.NewUInt64(0),
						}).WithType(rType),
					},
				}),
			},
		},
		writes,
	)
}
