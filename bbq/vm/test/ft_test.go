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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/test_utils/runtime_utils"
)

func TestFTTransfer(t *testing.T) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)
	programs := map[common.Location]*compiledProgram{}

	typeLoader := func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType {
		program, ok := programs[location]
		if !ok {
			panic(fmt.Errorf("cannot find elaboration for: %s", location))
		}
		elaboration := program.Elaboration
		compositeType := elaboration.CompositeType(typeID)
		if compositeType != nil {
			return compositeType
		}

		return elaboration.InterfaceType(typeID)
	}

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	txLocation := runtime_utils.NewTransactionLocationGenerator()
	scriptLocation := runtime_utils.NewScriptLocationGenerator()

	ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")

	ftContractProgram := compileCode(t, realFungibleTokenContractInterface, ftLocation, programs)
	printProgram("FungibleToken", ftContractProgram)

	// ----- Deploy FlowToken Contract -----

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")

	flowTokenProgram := compileCode(t, realFlowContract, flowTokenLocation, programs)
	printProgram("FlowToken", flowTokenProgram)

	config := &vm.Config{
		Storage:        storage,
		AccountHandler: &testAccountHandler{},
		TypeLoader:     typeLoader,
	}

	flowTokenVM := vm.NewVM(
		flowTokenLocation,
		flowTokenProgram,
		config,
	)

	authAccount := vm.NewAuthAccountReferenceValue(config, contractsAddress)
	flowTokenContractValue, err := flowTokenVM.InitializeContract(authAccount)
	require.NoError(t, err)

	// ----- Run setup account transaction -----

	vmConfig := &vm.Config{
		Storage: storage,
		ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
			imported, ok := programs[location]
			if !ok {
				return nil
			}
			return imported.Program
		},
		ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
			switch location {
			case ftLocation:
				// interface
				return nil
			case flowTokenLocation:
				return flowTokenContractValue
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		},

		AccountHandler: &testAccountHandler{},

		TypeLoader: typeLoader,
	}

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		program := compileCode(t, realSetupFlowTokenAccountTransaction, nil, programs)

		setupTxVM := vm.NewVM(txLocation(), program, vmConfig)

		authorizer := vm.NewAuthAccountReferenceValue(vmConfig, address)
		err = setupTxVM.ExecuteTransaction(nil, authorizer)
		require.NoError(t, err)
		require.Equal(t, 0, setupTxVM.StackSize())
	}

	// Mint FLOW to sender

	program := compileCode(t, realMintFlowTokenTransaction, nil, programs)
	printProgram("Setup FlowToken Tx", program)

	mintTxVM := vm.NewVM(txLocation(), program, vmConfig)

	total := int64(1000000)

	mintTxArgs := []vm.Value{
		vm.AddressValue(senderAddress),
		vm.NewIntValue(total),
	}

	mintTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, contractsAddress)
	err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
	require.NoError(t, err)
	require.Equal(t, 0, mintTxVM.StackSize())

	// ----- Run token transfer transaction -----

	tokenTransferTxProgram := compileCode(t, realFlowTokenTransferTransaction, nil, programs)
	printProgram("FT Transfer Tx", tokenTransferTxProgram)

	tokenTransferTxVM := vm.NewVM(txLocation(), tokenTransferTxProgram, vmConfig)

	transferAmount := int64(1)

	tokenTransferTxArgs := []vm.Value{
		vm.NewIntValue(transferAmount),
		vm.AddressValue(receiverAddress),
	}

	tokenTransferTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, senderAddress)
	err = tokenTransferTxVM.ExecuteTransaction(tokenTransferTxArgs, tokenTransferTxAuthorizer)
	require.NoError(t, err)
	require.Equal(t, 0, tokenTransferTxVM.StackSize())

	// Run validation scripts

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		program := compileCode(t, realFlowTokenBalanceScript, nil, programs)

		validationScriptVM := vm.NewVM(scriptLocation(), program, vmConfig)

		addressValue := vm.AddressValue(address)
		result, err := validationScriptVM.Invoke("main", addressValue)
		require.NoError(t, err)
		require.Equal(t, 0, validationScriptVM.StackSize())

		if address == senderAddress {
			assert.Equal(t, vm.NewIntValue(total-transferAmount), result)
		} else {
			assert.Equal(t, vm.NewIntValue(transferAmount), result)
		}
	}
}

func BenchmarkFTTransfer(b *testing.B) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)
	programs := map[common.Location]*compiledProgram{}

	typeLoader := func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType {
		program, ok := programs[location]
		if !ok {
			panic(fmt.Errorf("cannot find elaboration for: %s", location))
		}
		elaboration := program.Elaboration
		compositeType := elaboration.CompositeType(typeID)
		if compositeType != nil {
			return compositeType
		}

		return elaboration.InterfaceType(typeID)
	}

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	txLocation := runtime_utils.NewTransactionLocationGenerator()

	ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")
	_ = compileCode(b, realFungibleTokenContractInterface, ftLocation, programs)

	// ----- Deploy FlowToken Contract -----

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")
	flowTokenProgram := compileCode(b, realFlowContract, flowTokenLocation, programs)

	config := &vm.Config{
		Storage:        storage,
		AccountHandler: &testAccountHandler{},
		TypeLoader:     typeLoader,
	}

	flowTokenVM := vm.NewVM(
		flowTokenLocation,
		flowTokenProgram,
		config,
	)

	authAccount := vm.NewAuthAccountReferenceValue(config, contractsAddress)

	flowTokenContractValue, err := flowTokenVM.InitializeContract(authAccount)
	require.NoError(b, err)

	// ----- Run setup account transaction -----

	vmConfig := &vm.Config{
		Storage: storage,
		ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
			imported, ok := programs[location]
			if !ok {
				return nil
			}
			return imported.Program
		},
		ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
			switch location {
			case ftLocation:
				// interface
				return nil
			case flowTokenLocation:
				return flowTokenContractValue
			default:
				assert.FailNow(b, "invalid location")
				return nil
			}
		},

		AccountHandler: &testAccountHandler{},

		TypeLoader: typeLoader,
	}

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		program := compileCode(b, realSetupFlowTokenAccountTransaction, nil, programs)

		setupTxVM := vm.NewVM(txLocation(), program, vmConfig)

		authorizer := vm.NewAuthAccountReferenceValue(vmConfig, address)
		err = setupTxVM.ExecuteTransaction(nil, authorizer)
		require.NoError(b, err)
		require.Equal(b, 0, setupTxVM.StackSize())
	}

	// Mint FLOW to sender

	program := compileCode(b, realMintFlowTokenTransaction, nil, programs)

	mintTxVM := vm.NewVM(txLocation(), program, vmConfig)

	total := int64(1000000)

	mintTxArgs := []vm.Value{
		vm.AddressValue(senderAddress),
		vm.NewIntValue(total),
	}

	mintTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, contractsAddress)
	err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
	require.NoError(b, err)
	require.Equal(b, 0, mintTxVM.StackSize())

	// ----- Run token transfer transaction -----

	transferAmount := int64(1)

	tokenTransferTxArgs := []vm.Value{
		vm.NewIntValue(transferAmount),
		vm.AddressValue(receiverAddress),
	}

	tokenTransferTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, senderAddress)

	tokenTransferTxChecker := parseAndCheck(b, realFlowTokenTransferTransaction, nil, programs)
	tokenTransferTxProgram := compile(b, tokenTransferTxChecker, programs)
	tokenTransferTxVM := vm.NewVM(txLocation(), tokenTransferTxProgram, vmConfig)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = tokenTransferTxVM.ExecuteTransaction(tokenTransferTxArgs, tokenTransferTxAuthorizer)
		require.NoError(b, err)
	}

	b.StopTimer()

	// Run validation scripts

	// actual transfer amount = (transfer amount in one tx) * (number of time the tx/benchmark runs)
	actualTransferAmount := transferAmount * int64(b.N)

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		program := compileCode(b, realFlowTokenBalanceScript, nil, programs)

		validationScriptVM := vm.NewVM(scriptLocation(), program, vmConfig)

		addressValue := vm.AddressValue(address)
		result, err := validationScriptVM.Invoke("main", addressValue)
		require.NoError(b, err)
		require.Equal(b, 0, validationScriptVM.StackSize())

		if address == senderAddress {
			assert.Equal(b, vm.NewIntValue(total-actualTransferAmount), result)
		} else {
			assert.Equal(b, vm.NewIntValue(actualTransferAmount), result)
		}
	}
}
