/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence-standard-transactions/transactions"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

type Transaction struct {
	Name  string
	Body  string
	Setup string
}

var testTransactions []Transaction

func createTransaction(name string, imports string, prepare string, setup string) Transaction {
	return Transaction{
		Name: name,
		Body: fmt.Sprintf(
			`
			// %s
			%s

			transaction(){
				prepare(signer: auth(Storage, Contracts, Keys, Inbox, Capabilities) &Account) {
					%s
				}
			}`,
			name,
			imports,
			prepare,
		),
		Setup: fmt.Sprintf(
			`
			transaction(){
				prepare(signer: auth(Storage, Contracts, Keys, Inbox, Capabilities) &Account) {
					%s
				}
			}`,
			setup,
		),
	}
}

func stringOfLen(length uint64) string {
	someString := make([]byte, length)
	for i := 0; i < len(someString); i++ {
		someString[i] = 'x'
	}
	return string(someString)
}

func stringArrayOfLen(arrayLen uint64, stringLen uint64) string {
	builder := strings.Builder{}
	builder.WriteRune('[')
	for i := uint64(0); i < arrayLen; i++ {
		if i > 0 {
			builder.WriteRune(',')
		}
		builder.WriteRune('"')
		builder.WriteString(stringOfLen(stringLen))
		builder.WriteRune('"')
	}
	builder.WriteRune(']')
	return builder.String()
}

func init() {
	testTransactions = []Transaction{
		createTransaction(
			"EmptyLoop",
			"",
			transactions.EmptyLoopTransaction(6000).GetPrepareBlock(),
			"",
		),
		createTransaction("AssertTrue", "", transactions.AssertTrueTransaction(3000).GetPrepareBlock(), ""),
		createTransaction("GetSignerAddress", "", transactions.GetSignerAddressTransaction(4000).GetPrepareBlock(), ""),
		createTransaction("GetSignerPublicAccount", "", transactions.GetSignerPublicAccountTransaction(3000).GetPrepareBlock(), ""),
		createTransaction("GetSignerAccountBalance", "", transactions.GetSignerAccountBalanceTransaction(30).GetPrepareBlock(), ""),
		createTransaction("GetSignerAccountAvailableBalance", "", transactions.GetSignerAccountAvailableBalanceTransaction(30).GetPrepareBlock(), ""),
		createTransaction("GetSignerAccountStorageUsed", "", transactions.GetSignerAccountStorageUsedTransaction(700).GetPrepareBlock(), ""),
		createTransaction("GetSignerAccountStorageCapacity", "", transactions.GetSignerAccountStorageCapacityTransaction(30).GetPrepareBlock(), ""),
		createTransaction("BorrowSignerAccountFlowTokenVault", "import FungibleToken from 0x1\nimport FlowToken from 0x1", transactions.BorrowSignerAccountFlowTokenVaultTransaction(700).GetPrepareBlock(), ""),
		createTransaction("BorrowSignerAccountFungibleTokenReceiver", "import FungibleToken from 0x1\nimport FlowToken from 0x1", transactions.BorrowSignerAccountFungibleTokenReceiverTransaction(400).GetPrepareBlock(), ""),
		createTransaction("TransferTokensToSelf", "import FungibleToken from 0x1\nimport FlowToken from 0x1", transactions.TransferTokensToSelfTransaction(30).GetPrepareBlock(), ""),
		createTransaction("CreateNewAccount", "", transactions.CreateNewAccountTransaction(10).GetPrepareBlock(), ""),
		createTransaction("CreateNewAccountWithContract", "", transactions.CreateNewAccountWithContractTransaction(10).GetPrepareBlock(), ""),
		createTransaction("DecodeHex", "", transactions.DecodeHexTransaction(900).GetPrepareBlock(), ""),
		createTransaction("RevertibleRandomNumber", "", transactions.RevertibleRandomTransaction(2000).GetPrepareBlock(), ""),
		createTransaction("NumberToStringConversion", "", transactions.NumberToStringConversionTransaction(3000).GetPrepareBlock(), ""),
		createTransaction("ConcatenateString", "", transactions.ConcatenateStringTransaction(2000).GetPrepareBlock(), ""),
		createTransaction("BorrowString", "", transactions.BorrowStringTransaction.GetPrepareBlock(), fmt.Sprintf(transactions.BorrowStringTransaction.GetSetupTemplate(), stringArrayOfLen(20, 2000))),
		createTransaction("CopyString", "", transactions.CopyStringTransaction.GetPrepareBlock(), fmt.Sprintf(transactions.CopyStringTransaction.GetSetupTemplate(), stringArrayOfLen(20, 2000))),
	}
}

func benchmarkRuntimeTransactions(b *testing.B, useVM bool) {
	runtime := NewTestRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	accountCodes := map[common.Location][]byte{}

	var events []cadence.Event

	signerAccount := contractsAddress

	// Counter for generating unique account addresses
	// Use uint64 to avoid overflow issues with large iteration counts
	var accountCounter uint64 = 4

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]common.Address, error) {
			return []common.Address{signerAccount}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(b),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
		OnGetAccountBalance: func(address common.Address) (uint64, error) {
			return 0, nil
		},
		OnGetAccountAvailableBalance: func(address common.Address) (uint64, error) {
			return 0, nil
		},
		OnGetStorageUsed: func(address common.Address) (uint64, error) {
			return 0, nil
		},
		OnGetStorageCapacity: func(address common.Address) (uint64, error) {
			return 0, nil
		},
		OnCreateAccount: func(payer Address) (address Address, err error) {
			// Generate unique address from counter using binary encoding
			accountCounter++
			addressBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(addressBytes, accountCounter)
			result := interpreter.NewUnmeteredAddressValueFromBytes(addressBytes)
			return result.ToAddress(), nil
		},
	}

	var environment Environment
	if useVM {
		environment = NewBaseVMEnvironment(Config{})
	} else {
		environment = NewBaseInterpreterEnvironment(Config{})
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy Fungible Token contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
				"FungibleToken",
				[]byte(modifiedFungibleTokenContractInterface),
			),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
			UseVM:       useVM,
		},
	)
	require.NoError(b, err)

	// Deploy Flow Token contract

	err = runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction("FlowToken", []byte(modifiedFlowContract)),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
			UseVM:       useVM,
		},
	)
	require.NoError(b, err)

	// Setup both user accounts for Flow Token

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {

		signerAccount = address

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(realSetupFlowTokenAccountTransaction),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
				UseVM:       useVM,
			},
		)
		require.NoError(b, err)
	}

	// Mint 1000 FLOW to sender

	mintAmount, err := cadence.NewUFix64("100000000000.0")
	require.NoError(b, err)

	signerAccount = contractsAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(realMintFlowTokenTransaction),
			Arguments: encodeArgs([]cadence.Value{
				cadence.Address(senderAddress),
				mintAmount,
			}),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
			UseVM:       useVM,
		},
	)
	require.NoError(b, err)

	// Set signer account to sender for benchmark transactions
	signerAccount = senderAddress

	// all benchmark transactions reuse the same location
	for _, transaction := range testTransactions {
		b.Run(transaction.Name, func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				err = runtime.ExecuteTransaction(
					Script{
						Source:    []byte(transaction.Setup),
						Arguments: nil,
					},
					Context{
						Interface:   runtimeInterface,
						Location:    nextTransactionLocation(),
						Environment: environment,
						UseVM:       useVM,
					},
				)
				require.NoError(b, err)

				b.StartTimer()

				err = runtime.ExecuteTransaction(
					Script{
						Source:    []byte(transaction.Body),
						Arguments: nil,
					},
					Context{
						Interface:   runtimeInterface,
						Location:    nextTransactionLocation(),
						Environment: environment,
						UseVM:       useVM,
					},
				)

				b.StopTimer()
				require.NoError(b, err)
				b.StartTimer()
			}
		})
	}
}

func BenchmarkRuntimeTransactionsInterpreter(b *testing.B) {
	benchmarkRuntimeTransactions(b, false)
}

func BenchmarkRuntimeTransactionsVM(b *testing.B) {
	benchmarkRuntimeTransactions(b, true)
}
