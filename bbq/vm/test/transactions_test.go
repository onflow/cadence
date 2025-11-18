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

package test

import (
	"fmt"
	"testing"

	"github.com/onflow/cadence-standard-transactions/transactions"
	. "github.com/onflow/cadence/bbq/test_utils"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
	"github.com/stretchr/testify/require"
)

type Transaction struct {
	Name string
	Body string
}

var testTransactions []Transaction

func createTransaction(name string, imports string, prepare string) Transaction {
	return Transaction{
		Name: name,
		Body: fmt.Sprintf(
			`
			// %s
			%s

			transaction(){
				prepare(signer: auth(Storage, Contracts, Keys, Inbox, Capabilities) &Account) {
					var f: fun(): Void = fun(){}
					f = fun() { %s }
					f()
				}
			}`,
			name,
			imports,
			prepare,
		),
	}
}

func init() {
	testTransactions = []Transaction{
		createTransaction("EmptyLoopTransaction", "", transactions.EmptyLoopTransaction(5000).GetPrepareBlock()),
	}
}

func BenchmarkTransactions(b *testing.B) {
	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

	// Set up account handler for creating signers
	vmConfig.AccountHandlerFunc = func(
		context interpreter.AccountCreationContext,
		address interpreter.AddressValue,
	) interpreter.Value {
		return stdlib.NewAccountValue(context, nil, address)
	}

	// Create a signer address
	signerAddress := interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1})

	for _, transaction := range testTransactions {

		b.Run(transaction.Name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()

				vmInstance, err := CompileAndPrepareToInvoke(
					b,
					transaction.Body,
					CompilerAndVMOptions{
						VMConfig: vmConfig,
						ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
							ParseAndCheckOptions: &ParseAndCheckOptions{
								Location: common.TransactionLocation{},
							},
						},
					},
				)
				require.NoError(b, err)

				// Create signer for the transaction
				signer := stdlib.NewAccountReferenceValue(
					vmInstance.Context(),
					nil,
					signerAddress,
					interpreter.FullyEntitledAccountAccess,
				)

				err = vmInstance.InvokeTransaction(nil, signer)
				require.NoError(b, err)

				// Rerun the same again using internal functions, to get the access to the transaction value.

				transaction, err := vmInstance.InvokeTransactionWrapper()
				require.NoError(b, err)

				b.StartTimer()

				// Invoke 'prepare' with signer
				err = vmInstance.InvokeTransactionPrepare(transaction, []vm.Value{signer})

				b.StopTimer()
				require.NoError(b, err)
			}
		})
	}
}
