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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func compiledFTTransfer(tb testing.TB) {

	programs := map[common.Location]*compiledProgram{}

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	// TODO:
	//senderAddress := common.MustBytesToAddress([]byte{0x2})
	//receiverAddress := common.MustBytesToAddress([]byte{0x3})

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
	}

	// TODO:
	//nextTransactionLocation := NewTransactionLocationGenerator()
	//nextScriptLocation := NewScriptLocationGenerator()

	locationHandler := newStringLocationHandler(tb, contractsAddress)

	semaConfig := &sema.Config{
		LocationHandler:            locationHandler,
		BaseValueActivationHandler: baseValueActivation,
	}

	importHandler := func(location common.Location) *bbq.InstructionProgram {
		imported, ok := programs[location]
		if !ok {
			return nil
		}
		return imported.Program
	}

	compilerConfig := &compiler.Config{
		LocationHandler: commons.LocationHandler(locationHandler),
		ImportHandler:   importHandler,
		ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
			imported, ok := programs[location]
			if !ok {
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}
			return imported.Elaboration, nil
		},
	}

	// Parse and check contract interfaces

	contractInterfaceLocations := []common.Location{
		burnerLocation,
		viewResolverLocation,
		fungibleTokenLocation,
		nonFungibleTokenLocation,
		metadataViewsLocation,
		fungibleTokenMetadataViewsLocation,
	}

	for _, location := range contractInterfaceLocations {
		_ = parseCheckAndCompileCodeWithOptions(
			tb,
			string(codes[location]),
			location,
			CompilerAndVMOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: location,
					Config:   semaConfig,
				},
				CompilerConfig: compilerConfig,
			},
			programs,
		)
	}

	// Deploy Flow Token contract

	flowTokenProgram := parseCheckAndCompileCodeWithOptions(
		tb,
		realFlowContract,
		flowTokenLocation,
		CompilerAndVMOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				Location: flowTokenLocation,
				Config:   semaConfig,
			},
			CompilerConfig: compilerConfig,
		},
		programs,
	)

	// Prepare VM

	accountHandler := &testAccountHandler{
		emitEvent: func(
			_ interpreter.ValueExportContext,
			_ interpreter.LocationRange,
			_ *sema.CompositeType,
			_ []interpreter.Value,
		) {
			// ignore
		},
	}

	storage := interpreter.NewInMemoryStorage(nil)

	typeLoader := func(location common.Location, typeID interpreter.TypeID) sema.ContainedType {
		program, ok := programs[location]
		if !ok {
			panic(fmt.Errorf("cannot find elaboration for: %s", location))
		}
		elaboration := program.Elaboration
		compositeType := elaboration.CompositeType(typeID)
		if compositeType != nil {
			return compositeType
		}

		interfaceType := elaboration.InterfaceType(typeID)
		if interfaceType != nil {
			return interfaceType
		}

		entitlementType := elaboration.EntitlementType(typeID)
		if entitlementType != nil {
			return entitlementType
		}

		return elaboration.EntitlementMapType(typeID)
	}

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.PanicFunction)
	interpreter.Declare(baseActivation, stdlib.AssertFunction)
	interpreter.Declare(baseActivation, stdlib.NewGetAccountFunction(accountHandler))

	interConfig := &interpreter.Config{
		Storage: storage,
		BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
			return baseActivation
		},
		ContractValueHandler: func(
			inter *interpreter.Interpreter,
			compositeType *sema.CompositeType,
			constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
			invocationRange ast.Range,
		) interpreter.ContractValue {

			constructor := constructorGenerator(common.ZeroAddress)

			value, err := interpreter.InvokeFunctionValue(
				inter,
				constructor,
				nil,
				nil,
				nil,
				compositeType,
				ast.EmptyRange,
			)
			if err != nil {
				panic(err)
			}

			flowTokenContractValue := value.(*interpreter.CompositeValue)
			return flowTokenContractValue
		},
		CapabilityBorrowHandler: func(
			context interpreter.BorrowCapabilityControllerContext,
			locationRange interpreter.LocationRange,
			address interpreter.AddressValue,
			capabilityID interpreter.UInt64Value,
			wantedBorrowType *sema.ReferenceType,
			capabilityBorrowType *sema.ReferenceType,
		) interpreter.ReferenceValue {
			return stdlib.BorrowCapabilityController(
				context,
				locationRange,
				address,
				capabilityID,
				wantedBorrowType,
				capabilityBorrowType,
				accountHandler,
			)
		},
		OnEventEmitted: func(
			_ *interpreter.Interpreter,
			_ interpreter.LocationRange,
			_ *interpreter.CompositeValue,
			_ *sema.CompositeType,
		) error {
			// NO-OP
			return nil
		},
		InjectedCompositeFieldsHandler: func(
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
				interpreter.EmptyLocationRange,
			)

			return map[string]interpreter.Value{
				sema.ContractAccountFieldName: accountRef,
			}
		},
		AccountHandler: func(context interpreter.AccountCreationContext, address interpreter.AddressValue) interpreter.Value {
			return stdlib.NewAccountValue(context, nil, address)
		},
	}

	vmConfig := vm.NewConfig(storage).
		WithAccountHandler(accountHandler).
		WithInterpreterConfig(interConfig)

	vmConfig.TypeLoader = typeLoader
	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		imported, ok := programs[location]
		if !ok {
			return nil
		}
		return imported.Program
	}

	flowTokenVM := vm.NewVM(
		flowTokenLocation,
		flowTokenProgram,
		vmConfig,
	)

	_, err := flowTokenVM.InitializeContract()
	require.NoError(tb, err)

	// TODO:
	//// ----- Run setup account transaction -----
	//vmConfig := vm.NewConfig(storage).
	//	WithAccountHandler(accountHandler).
	//	WithInterpreterConfig(interpreterEnv.NewInterpreterConfig())
	//
	//vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
	//	imported, ok := programs[location]
	//	if !ok {
	//		return nil
	//	}
	//	return imported.Program
	//}
	//vmConfig.ContractValueHandler = func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
	//	switch location {
	//	case ftLocation:
	//		// interface
	//		return nil
	//	case flowTokenLocation:
	//		return flowTokenContractValue
	//	default:
	//		assert.FailNow(tb, "invalid location")
	//		return nil
	//	}
	//}
	//
	//vmConfig.TypeLoader = typeLoader
	//
	//for _, address := range []common.Address{
	//	senderAddress,
	//	receiverAddress,
	//} {
	//	program := parseCheckAndCompile(tb, realFlowTokenSetupAccountTransaction, nil, programs)
	//
	//	setupTxVM := vm.NewVM(nexTransactionLocation(), program, vmConfig)
	//
	//	authorizer := vm.NewAuthAccountReferenceValue(vmConfig, accountHandler, address)
	//	err = setupTxVM.ExecuteTransaction(nil, authorizer)
	//	require.NoError(tb, err)
	//	require.Equal(tb, 0, setupTxVM.StackSize())
	//}
	//
	//// Mint FLOW to sender
	//
	//program := parseCheckAndCompile(tb, realFlowTokenMintTokensTransaction, nil, programs)
	//
	//mintTxVM := vm.NewVM(nextTransactionLocation(), program, vmConfig)
	//
	//total := uint64(1000000) * sema.Fix64Factor
	//
	//mintTxArgs := []vm.Value{
	//	interpreter.AddressValue(senderAddress),
	//	interpreter.NewUnmeteredUFix64Value(total),
	//}
	//
	//mintTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, accountHandler, contractsAddress)
	//err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
	//require.NoError(tb, err)
	//require.Equal(tb, 0, mintTxVM.StackSize())
	//
	//// ----- Run token transfer transaction -----
	//
	//tokenTransferTxProgram := parseCheckAndCompile(tb, realFlowTokenTransferTokensTransaction, nil, programs)
	//
	//tokenTransferTxVM := vm.NewVM(nextTransactionLocation(), tokenTransferTxProgram, vmConfig)
	//
	//transferAmount := uint64(1) * sema.Fix64Factor
	//
	//tokenTransferTxArgs := []vm.Value{
	//	interpreter.NewUnmeteredUFix64Value(transferAmount),
	//	interpreter.AddressValue(receiverAddress),
	//}
	//
	//tokenTransferTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, accountHandler, senderAddress)
	//
	//var transferCount int
	//
	//loop := func() bool {
	//	return transferCount == 0
	//}
	//
	//b, _ := tb.(*testing.B)
	//
	//if b != nil {
	//
	//	b.ReportAllocs()
	//	b.ResetTimer()
	//
	//	loop = func() bool {
	//		return transferCount < b.N
	//	}
	//}
	//
	//for loop() {
	//
	//	err = tokenTransferTxVM.ExecuteTransaction(tokenTransferTxArgs, tokenTransferTxAuthorizer)
	//	require.NoError(tb, err)
	//	require.Equal(tb, 0, tokenTransferTxVM.StackSize())
	//
	//	transferCount++
	//}
	//
	//if b != nil {
	//	b.StopTimer()
	//}
	//
	//// Run validation scripts
	//
	//for _, address := range []common.Address{
	//	senderAddress,
	//	receiverAddress,
	//} {
	//	program := parseCheckAndCompile(tb, realFlowTokenGetBalanceScript, nil, programs)
	//
	//	validationScriptVM := vm.NewVM(nextScriptLocation(), program, vmConfig)
	//
	//	addressValue := interpreter.AddressValue(address)
	//	result, err := validationScriptVM.Invoke("main", addressValue)
	//	require.NoError(tb, err)
	//	require.Equal(tb, 0, validationScriptVM.StackSize())
	//
	//	if address == senderAddress {
	//		assert.Equal(tb, interpreter.NewUnmeteredUFix64Value(total-transferAmount*uint64(transferCount)), result)
	//	} else {
	//		assert.Equal(tb, interpreter.NewUnmeteredUFix64Value(transferAmount*uint64(transferCount)), result)
	//	}
	//}
}

func TestFTTransfer(t *testing.T) {
	t.Parallel()

	compiledFTTransfer(t)
}

func BenchmarkFTTransfer(b *testing.B) {

	compiledFTTransfer(b)
}
