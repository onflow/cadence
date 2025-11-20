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
	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
	. "github.com/onflow/cadence/bbq/test_utils"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
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
					%s
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
		createTransaction("EmptyLoop", "", transactions.EmptyLoopTransaction(6000).GetPrepareBlock()),
		createTransaction("AssertTrue", "", transactions.AssertTrueTransaction(3000).GetPrepareBlock()),
		createTransaction("GetSignerAddress", "", transactions.GetSignerAddressTransaction(4000).GetPrepareBlock()),
		createTransaction("GetSignerPublicAccount", "", transactions.GetSignerPublicAccountTransaction(3000).GetPrepareBlock()),
		createTransaction("GetSignerAccountBalance", "", transactions.GetSignerAccountBalanceTransaction(30).GetPrepareBlock()),
		createTransaction("GetSignerAccountAvailableBalance", "", transactions.GetSignerAccountAvailableBalanceTransaction(30).GetPrepareBlock()),
		createTransaction("GetSignerAccountStorageUsed", "", transactions.GetSignerAccountStorageUsedTransaction(700).GetPrepareBlock()),
		createTransaction("GetSignerAccountStorageCapacity", "", transactions.GetSignerAccountStorageCapacityTransaction(30).GetPrepareBlock()),
		createTransaction("BorrowSignerAccountFlowTokenVault", "import \"FungibleToken\"\nimport \"FlowToken\"", transactions.BorrowSignerAccountFlowTokenVaultTransaction(700).GetPrepareBlock()),
	}
}

func BenchmarkTransactions(b *testing.B) {
	// create addresses
	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})

	locationHandler := newSingleAddressOrStringLocationHandler(b, contractsAddress)

	// make contracts available
	compiledPrograms := CompiledPrograms{}

	burnerLocation := common.NewAddressLocation(nil, contractsAddress, "Burner")
	viewResolverLocation := common.NewAddressLocation(nil, contractsAddress, "ViewResolver")
	fungibleTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")
	metadataViewsLocation := common.NewAddressLocation(nil, contractsAddress, "MetadataViews")
	fungibleTokenMetadataViewsLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleTokenMetadataViews")
	nonFungibleTokenLocation := common.NewAddressLocation(nil, contractsAddress, "NonFungibleToken")
	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")

	codes := map[common.Location][]byte{
		burnerLocation:                     []byte(realBurnerContract),
		viewResolverLocation:               []byte(realViewResolverContract),
		fungibleTokenLocation:              []byte(realFungibleTokenContract),
		metadataViewsLocation:              []byte(realMetadataViewsContract),
		fungibleTokenMetadataViewsLocation: []byte(realFungibleTokenMetadataViewsContract),
		nonFungibleTokenLocation:           []byte(realNonFungibleTokenContract),
		flowTokenLocation:                  []byte(realFlowContract),
	}

	importHandler := func(location common.Location) *bbq.InstructionProgram {
		imported, ok := compiledPrograms[location]
		if !ok {
			return nil
		}
		return imported.Program
	}

	accountHandler := &testAccountHandler{
		emitEvent: func(
			_ interpreter.ValueExportContext,
			_ *sema.CompositeType,
			_ []interpreter.Value,
		) {
			// ignore
		},
		getAccountBalance: func(address common.Address) (uint64, error) {
			return 1000000000000000000, nil
		},
		getAccountAvailableBalance: func(address common.Address) (uint64, error) {
			return 1000000000000000000, nil
		},
		getStorageUsed: func(address common.Address) (uint64, error) {
			return 1000000000000000000, nil
		},
		getStorageCapacity: func(address common.Address) (uint64, error) {
			return 1000000000000000000, nil
		},
		commitStorageTemporarily: func(context interpreter.ValueTransferContext) error {
			return nil
		},
	}

	// set up sema/compiler
	semaConfig := &sema.Config{
		LocationHandler:            locationHandler,
		BaseValueActivationHandler: TestBaseValueActivation,
	}

	compilerConfig := &compiler.Config{
		LocationHandler: locationHandler,
		ImportHandler:   importHandler,
		ElaborationResolver: func(location common.Location) (*compiler.DesugaredElaboration, error) {
			imported, ok := compiledPrograms[location]
			if !ok {
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}
			return imported.DesugaredElaboration, nil
		},
		BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
			activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())

			activation.Set(
				stdlib.AssertFunctionName,
				compiler.NewGlobalImport(stdlib.AssertFunctionName),
			)

			activation.Set(
				stdlib.GetAccountFunctionName,
				compiler.NewGlobalImport(stdlib.GetAccountFunctionName),
			)

			activation.Set(
				stdlib.PanicFunctionName,
				compiler.NewGlobalImport(stdlib.PanicFunctionName),
			)

			return activation
		},
	}

	// Parse and check contracts

	for _, location := range []common.Location{
		burnerLocation,
		viewResolverLocation,
		fungibleTokenLocation,
		nonFungibleTokenLocation,
		metadataViewsLocation,
		fungibleTokenMetadataViewsLocation,
		flowTokenLocation,
	} {
		_ = ParseCheckAndCompileCodeWithOptions(
			b,
			string(codes[location]),
			location,
			ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location:      location,
					CheckerConfig: semaConfig,
				},
				CompilerConfig: compilerConfig,
			},
			compiledPrograms,
		)
	}

	// set up VM
	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

	vmConfig.AccountHandlerFunc = func(
		context interpreter.AccountCreationContext,
		address interpreter.AddressValue,
	) interpreter.Value {
		return stdlib.NewAccountValue(context, accountHandler, address)
	}

	contractValues := make(map[common.Location]*interpreter.CompositeValue)
	vmConfig.ContractValueHandler = func(
		_ *vm.Context,
		location common.Location,
	) *interpreter.CompositeValue {
		return contractValues[location]
	}

	vmConfig.ImportHandler = importHandler

	vmConfig.InjectedCompositeFieldsHandler = func(
		context interpreter.AccountCreationContext,
		_ common.Location,
		_ string,
		_ common.CompositeKind,
	) map[string]interpreter.Value {

		accountRef := stdlib.NewAccountReferenceValue(
			context,
			accountHandler,
			interpreter.NewAddressValue(nil, contractsAddress),
			interpreter.FullyEntitledAccountAccess,
		)

		return map[string]interpreter.Value{
			sema.ContractAccountFieldName: accountRef,
		}
	}

	vmConfig.BuiltinGlobalsProvider = func(_ common.Location) *activations.Activation[vm.Variable] {
		activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

		panicVariable := &interpreter.SimpleVariable{}
		panicVariable.InitializeWithValue(stdlib.VMPanicFunction.Value)
		activation.Set(
			stdlib.PanicFunctionName,
			panicVariable,
		)

		assertVariable := &interpreter.SimpleVariable{}
		assertVariable.InitializeWithValue(stdlib.VMAssertFunction.Value)
		activation.Set(
			stdlib.AssertFunctionName,
			assertVariable,
		)

		getAccountVariable := &interpreter.SimpleVariable{}
		getAccountVariable.InitializeWithValue(stdlib.NewVMGetAccountFunction(accountHandler).Value)
		activation.Set(
			stdlib.GetAccountFunctionName,
			getAccountVariable,
		)

		for _, vmFunction := range []stdlib.VMFunction{
			stdlib.NewVMAccountCapabilitiesPublishFunction(accountHandler),
			stdlib.NewVMAccountStorageCapabilitiesIssueFunction(accountHandler),
			stdlib.NewVMAccountCapabilitiesGetFunction(accountHandler, true),
		} {
			variable := &interpreter.SimpleVariable{}
			variable.InitializeWithValue(vmFunction.FunctionValue)
			activation.Set(
				commons.TypeQualifiedName(
					vmFunction.BaseType,
					vmFunction.FunctionValue.Name,
				),
				variable,
			)
		}

		return activation
	}

	vmConfig = PrepareVMConfig(b, vmConfig, compiledPrograms)

	// Initialize contracts
	for _, location := range []common.Location{
		metadataViewsLocation,
		fungibleTokenMetadataViewsLocation,
		flowTokenLocation,
	} {
		compiledProgram := compiledPrograms[location]
		_, contractValue := initializeContract(
			b,
			location,
			compiledProgram.Program,
			vmConfig,
		)

		contractValues[location] = contractValue
	}

	// all transactions use the same location, this prevents compiledPrograms from blowing up
	nextTransactionLocation := NewTransactionLocationGenerator()
	txLocation := nextTransactionLocation()

	for _, transaction := range testTransactions {
		b.Run(transaction.Name, func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()

				setupAccountProgram := ParseCheckAndCompileCodeWithOptions(
					b,
					`
					import "FungibleToken"
					import "FlowToken"

					transaction {
   						prepare(signer: auth(Capabilities, Storage) &Account) {
							if signer.storage.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault) == nil {
								// Create a new flowToken Vault and put it in storage
								signer.storage.save(<-FlowToken.createEmptyVault(vaultType: Type<@FlowToken.Vault>()), to: /storage/flowTokenVault)
							}
						}
					}
					`,
					txLocation,
					ParseCheckAndCompileOptions{
						ParseAndCheckOptions: &ParseAndCheckOptions{
							Location:      txLocation,
							CheckerConfig: semaConfig,
						},
						CompilerConfig: compilerConfig,
					},
					compiledPrograms,
				)

				setupAccountVM := vm.NewVM(txLocation, setupAccountProgram, vmConfig)

				setupAccountAuthorizer := stdlib.NewAccountReferenceValue(
					setupAccountVM.Context(),
					accountHandler,
					interpreter.NewAddressValue(nil, senderAddress),
					interpreter.FullyEntitledAccountAccess,
				)

				err := setupAccountVM.InvokeTransaction(nil, setupAccountAuthorizer)
				require.NoError(b, err)

				program := ParseCheckAndCompileCodeWithOptions(
					b,
					transaction.Body,
					txLocation,
					ParseCheckAndCompileOptions{
						ParseAndCheckOptions: &ParseAndCheckOptions{
							Location:      txLocation,
							CheckerConfig: semaConfig,
						},
						CompilerConfig: compilerConfig,
					},
					compiledPrograms,
				)

				vmInstance := vm.NewVM(txLocation, program, vmConfig)

				authorizer := stdlib.NewAccountReferenceValue(
					vmInstance.Context(),
					accountHandler,
					interpreter.NewAddressValue(nil, senderAddress),
					interpreter.FullyEntitledAccountAccess,
				)

				b.StartTimer()

				err = vmInstance.InvokeTransaction(nil, authorizer)

				b.StopTimer()
				require.NoError(b, err)
				b.StartTimer()
			}
		})
	}
}
