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

	_ = parseCheckAndCompile(t, realFungibleTokenContractInterface, ftLocation, programs)

	// ----- Deploy FlowToken Contract -----

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")

	flowTokenProgram := parseCheckAndCompile(t, realFlowContract, flowTokenLocation, programs)

	config := vm.NewConfig(storage)

	accountHandler := &testAccountHandler{}

	config.TypeLoader = typeLoader
	config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		imported, ok := programs[location]
		if !ok {
			return nil
		}
		return imported.Program
	}

	flowTokenVM := vm.NewVM(
		flowTokenLocation,
		flowTokenProgram,
		config,
	)

	authAccount := vm.NewAuthAccountReferenceValue(config, accountHandler, contractsAddress)
	flowTokenContractValue, err := flowTokenVM.InitializeContract(authAccount)
	require.NoError(t, err)

	// ----- Run setup account transaction -----
	vmConfig := vm.NewConfig(storage)

	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		imported, ok := programs[location]
		if !ok {
			return nil
		}
		return imported.Program
	}
	vmConfig.ContractValueHandler = func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
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
	}

	vmConfig.TypeLoader = typeLoader

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		program := parseCheckAndCompile(t, realSetupFlowTokenAccountTransaction, nil, programs)

		setupTxVM := vm.NewVM(txLocation(), program, vmConfig)

		authorizer := vm.NewAuthAccountReferenceValue(vmConfig, accountHandler, address)
		err = setupTxVM.ExecuteTransaction(nil, authorizer)
		require.NoError(t, err)
		require.Equal(t, 0, setupTxVM.StackSize())
	}

	// Mint FLOW to sender

	program := parseCheckAndCompile(t, realMintFlowTokenTransaction, nil, programs)

	mintTxVM := vm.NewVM(txLocation(), program, vmConfig)

	total := uint64(1000000) * sema.Fix64Factor

	mintTxArgs := []vm.Value{
		interpreter.AddressValue(senderAddress),
		interpreter.NewUnmeteredUFix64Value(total),
	}

	mintTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, accountHandler, contractsAddress)
	err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
	require.NoError(t, err)
	require.Equal(t, 0, mintTxVM.StackSize())

	// ----- Run token transfer transaction -----

	tokenTransferTxProgram := parseCheckAndCompile(t, realFlowTokenTransferTransaction, nil, programs)

	tokenTransferTxVM := vm.NewVM(txLocation(), tokenTransferTxProgram, vmConfig)

	transferAmount := uint64(1) * sema.Fix64Factor

	tokenTransferTxArgs := []vm.Value{
		interpreter.NewUnmeteredUFix64Value(transferAmount),
		interpreter.AddressValue(receiverAddress),
	}

	tokenTransferTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, accountHandler, senderAddress)
	err = tokenTransferTxVM.ExecuteTransaction(tokenTransferTxArgs, tokenTransferTxAuthorizer)
	require.NoError(t, err)
	require.Equal(t, 0, tokenTransferTxVM.StackSize())

	// Run validation scripts

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		program := parseCheckAndCompile(t, realFlowTokenBalanceScript, nil, programs)

		validationScriptVM := vm.NewVM(scriptLocation(), program, vmConfig)

		addressValue := interpreter.AddressValue(address)
		result, err := validationScriptVM.Invoke("main", addressValue)
		require.NoError(t, err)
		require.Equal(t, 0, validationScriptVM.StackSize())

		if address == senderAddress {
			assert.Equal(t, interpreter.NewUnmeteredUFix64Value(total-transferAmount), result)
		} else {
			assert.Equal(t, interpreter.NewUnmeteredUFix64Value(transferAmount), result)
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
	_ = parseCheckAndCompile(b, realFungibleTokenContractInterface, ftLocation, programs)

	// ----- Deploy FlowToken Contract -----

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")
	flowTokenProgram := parseCheckAndCompile(b, realFlowContract, flowTokenLocation, programs)

	accountHandler := &testAccountHandler{}

	config := vm.NewConfig(storage)
	config.TypeLoader = typeLoader

	flowTokenVM := vm.NewVM(
		flowTokenLocation,
		flowTokenProgram,
		config,
	)

	authAccount := vm.NewAuthAccountReferenceValue(config, accountHandler, contractsAddress)

	flowTokenContractValue, err := flowTokenVM.InitializeContract(authAccount)
	require.NoError(b, err)

	// ----- Run setup account transaction -----

	vmConfig := vm.NewConfig(storage)

	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		imported, ok := programs[location]
		if !ok {
			return nil
		}
		return imported.Program
	}
	vmConfig.ContractValueHandler = func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
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
	}

	vmConfig.TypeLoader = typeLoader

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		program := parseCheckAndCompile(b, realSetupFlowTokenAccountTransaction, nil, programs)

		setupTxVM := vm.NewVM(txLocation(), program, vmConfig)

		authorizer := vm.NewAuthAccountReferenceValue(vmConfig, accountHandler, address)
		err = setupTxVM.ExecuteTransaction(nil, authorizer)
		require.NoError(b, err)
		require.Equal(b, 0, setupTxVM.StackSize())
	}

	// Mint FLOW to sender

	program := parseCheckAndCompile(b, realMintFlowTokenTransaction, nil, programs)

	mintTxVM := vm.NewVM(txLocation(), program, vmConfig)

	total := int64(1000000)

	mintTxArgs := []vm.Value{
		interpreter.AddressValue(senderAddress),
		interpreter.NewUnmeteredIntValueFromInt64(total),
	}

	mintTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, accountHandler, contractsAddress)
	err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
	require.NoError(b, err)
	require.Equal(b, 0, mintTxVM.StackSize())

	// ----- Run token transfer transaction -----

	transferAmount := int64(1)

	tokenTransferTxArgs := []vm.Value{
		interpreter.NewUnmeteredIntValueFromInt64(transferAmount),
		interpreter.AddressValue(receiverAddress),
	}

	tokenTransferTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, accountHandler, senderAddress)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tokenTransferTxProgram := parseCheckAndCompile(b, realFlowTokenTransferTransaction, nil, programs)
		tokenTransferTxVM := vm.NewVM(txLocation(), tokenTransferTxProgram, vmConfig)
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
		program := parseCheckAndCompile(b, realFlowTokenBalanceScript, nil, programs)

		validationScriptVM := vm.NewVM(scriptLocation(), program, vmConfig)

		addressValue := interpreter.AddressValue(address)
		result, err := validationScriptVM.Invoke("main", addressValue)
		require.NoError(b, err)
		require.Equal(b, 0, validationScriptVM.StackSize())

		if address == senderAddress {
			assert.Equal(b, interpreter.NewUnmeteredIntValueFromInt64(total-actualTransferAmount), result)
		} else {
			assert.Equal(b, interpreter.NewUnmeteredIntValueFromInt64(actualTransferAmount), result)
		}
	}
}
