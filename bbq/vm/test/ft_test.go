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
)

func TestFTTransfer(t *testing.T) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)
	programs := map[common.Location]compiledProgram{}

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")

	_ = compileCode(t, realFungibleTokenContractInterface, ftLocation, programs)

	// ----- Deploy FlowToken Contract -----

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")

	flowTokenProgram := compileCode(t, realFlowContract, flowTokenLocation, programs)

	config := &vm.Config{
		Storage:        storage,
		AccountHandler: &testAccountHandler{},
	}

	flowTokenVM := vm.NewVM(
		flowTokenProgram,
		config,
	)

	authAccount := vm.NewAuthAccountReferenceValue(config, contractsAddress)
	flowTokenContractValue, err := flowTokenVM.InitializeContract(authAccount)
	require.NoError(t, err)

	// ----- Run setup account transaction -----

	vmConfig := &vm.Config{
		Storage: storage,
		ImportHandler: func(location common.Location) *bbq.Program {
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

		TypeLoader: func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType {
			imported, ok := programs[location]
			if !ok {
				panic(fmt.Errorf("cannot find contract in location %s", location))
			}

			compositeType := imported.Elaboration.CompositeType(typeID)
			if compositeType != nil {
				return compositeType
			}

			return imported.Elaboration.InterfaceType(typeID)
		},
	}

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		program := compileCode(t, realSetupFlowTokenAccountTransaction, nil, programs)

		setupTxVM := vm.NewVM(program, vmConfig)

		authorizer := vm.NewAuthAccountReferenceValue(vmConfig, address)
		err = setupTxVM.ExecuteTransaction(nil, authorizer)
		require.NoError(t, err)
		require.Equal(t, 0, setupTxVM.StackSize())
	}

	// Mint FLOW to sender

	program := compileCode(t, realMintFlowTokenTransaction, nil, programs)

	mintTxVM := vm.NewVM(program, vmConfig)

	total := int64(1000000)

	mintTxArgs := []vm.Value{
		vm.AddressValue(senderAddress),
		vm.IntValue{total},
	}

	mintTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, contractsAddress)
	err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
	require.NoError(t, err)
	require.Equal(t, 0, mintTxVM.StackSize())

	// ----- Run token transfer transaction -----

	tokenTransferTxProgram := compileCode(t, realFlowTokenTransferTransaction, nil, programs)

	tokenTransferTxVM := vm.NewVM(tokenTransferTxProgram, vmConfig)

	transferAmount := int64(1)

	tokenTransferTxArgs := []vm.Value{
		vm.IntValue{transferAmount},
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

		validationScriptVM := vm.NewVM(program, vmConfig)

		addressValue := vm.AddressValue(address)
		result, err := validationScriptVM.Invoke("main", addressValue)
		require.NoError(t, err)
		require.Equal(t, 0, validationScriptVM.StackSize())

		if address == senderAddress {
			assert.Equal(t, vm.IntValue{total - transferAmount}, result)
		} else {
			assert.Equal(t, vm.IntValue{transferAmount}, result)
		}
	}
}

func BenchmarkFTTransfer(b *testing.B) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)
	programs := map[common.Location]compiledProgram{}

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")
	_ = compileCode(b, realFungibleTokenContractInterface, ftLocation, programs)

	// ----- Deploy FlowToken Contract -----

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")
	flowTokenProgram := compileCode(b, realFlowContract, flowTokenLocation, programs)

	config := &vm.Config{
		Storage:        storage,
		AccountHandler: &testAccountHandler{},
	}

	flowTokenVM := vm.NewVM(
		flowTokenProgram,
		config,
	)

	authAccount := vm.NewAuthAccountReferenceValue(config, contractsAddress)

	flowTokenContractValue, err := flowTokenVM.InitializeContract(authAccount)
	require.NoError(b, err)

	// ----- Run setup account transaction -----

	vmConfig := &vm.Config{
		Storage: storage,
		ImportHandler: func(location common.Location) *bbq.Program {
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

		TypeLoader: func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType {
			imported, ok := programs[location]
			if !ok {
				panic(fmt.Errorf("cannot find contract in location %s", location))
			}

			compositeType := imported.Elaboration.CompositeType(typeID)
			if compositeType != nil {
				return compositeType
			}

			return imported.Elaboration.InterfaceType(typeID)
		},
	}

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		program := compileCode(b, realSetupFlowTokenAccountTransaction, nil, programs)

		setupTxVM := vm.NewVM(program, vmConfig)

		authorizer := vm.NewAuthAccountReferenceValue(vmConfig, address)
		err = setupTxVM.ExecuteTransaction(nil, authorizer)
		require.NoError(b, err)
		require.Equal(b, 0, setupTxVM.StackSize())
	}

	// Mint FLOW to sender

	program := compileCode(b, realMintFlowTokenTransaction, nil, programs)

	mintTxVM := vm.NewVM(program, vmConfig)

	total := int64(1000000)

	mintTxArgs := []vm.Value{
		vm.AddressValue(senderAddress),
		vm.IntValue{total},
	}

	mintTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, contractsAddress)
	err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
	require.NoError(b, err)
	require.Equal(b, 0, mintTxVM.StackSize())

	// ----- Run token transfer transaction -----

	tokenTransferTxChecker := parseAndCheck(b, realFlowTokenTransferTransaction, nil, programs)

	transferAmount := int64(1)

	tokenTransferTxArgs := []vm.Value{
		vm.IntValue{transferAmount},
		vm.AddressValue(receiverAddress),
	}

	tokenTransferTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, senderAddress)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tokenTransferTxProgram := compile(b, tokenTransferTxChecker, programs)

		tokenTransferTxVM := vm.NewVM(tokenTransferTxProgram, vmConfig)
		err = tokenTransferTxVM.ExecuteTransaction(tokenTransferTxArgs, tokenTransferTxAuthorizer)
		require.NoError(b, err)
	}

	b.StopTimer()
}