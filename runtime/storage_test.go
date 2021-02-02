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

	deployTx := utils.DeploymentTransaction("Test", contract)

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
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) (err error) {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
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

	err := runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	rType := &cadence.ResourceType{
		Location: common.AddressLocation{
			Address: common.BytesToAddress([]byte{0xca, 0xde}),
			Name:    "Test",
		},
		QualifiedIdentifier: "Test.R",
		Fields: []cadence.Field{
			{
				Identifier: "uuid",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "i",
				Type:       cadence.IntType{},
			},
		},
	}

	assert.Equal(t,
		[]write{
			{
				address,
				"contract\x1fTest",
				cadence.NewContract([]cadence.Value{}).WithType(&cadence.ContractType{
					Location: common.AddressLocation{
						Address: common.BytesToAddress([]byte{0xca, 0xde}),
						Name:    "Test",
					},
					QualifiedIdentifier: "Test",
					Fields:              []cadence.Field{},
					Initializers:        nil,
				}),
			},
		},
		writes,
	)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: setupTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
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
							cadence.NewUInt64(0),
							cadence.NewInt(3),
						}).WithType(rType),
					},
					{
						Key: cadence.NewString("r2"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewUInt64(0),
							cadence.NewInt(4),
						}).WithType(rType),
					},
				}),
			},
		},
		writes,
	)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: changeTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
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
							cadence.NewUInt64(0),
							cadence.NewInt(5),
						}).WithType(rType),
					},
					{
						Key: cadence.NewString("r2"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewUInt64(0),
							cadence.NewInt(4),
						}).WithType(rType),
					},
				}),
			},
		},
		writes,
	)
}

func TestRuntimeMagic(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address := common.BytesToAddress([]byte{0x1})

	tx := []byte(`
	  transaction {
	      prepare(signer: AuthAccount) {
	          signer.save(1, to: /storage/one)
	      }
	   }
	`)

	var writes []testWrite

	onWrite := func(owner, key, value []byte) {
		writes = append(writes, testWrite{
			owner,
			key,
			value,
		})
	}

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, onWrite),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]testWrite{
			{
				[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
				[]byte("storage\x1fone"),
				[]byte{
					// magic
					0x0, 0xCA, 0xDE, 0x0, 0x2,
					// CBOR
					// - tag
					0xd8, 0x98,
					// - positive bignum
					0xc2,
					// - byte string, length 1
					0x41,
					0x1,
				},
			},
		},
		writes,
	)
}

func TestAccountStorageStorage(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
           let before = signer.storageUsed
           signer.save(42, to: /storage/answer)
           let after = signer.storageUsed
           log(after != before)
        }
      }
    `)

	var loggedMessages []string

	storage := newTestStorage(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		getStorageUsed: func(_ Address) (uint64, error) {
			var amount uint64 = 0

			for _, data := range storage.storedValues {
				amount += uint64(len(data))
			}

			return amount, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Equal(t,
		[]string{"true"},
		loggedMessages,
	)
}
