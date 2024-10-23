/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/vm"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
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

const realFungibleTokenContractInterface = `
/// FungibleToken
///
/// The interface that fungible token contracts implement.
///
access(all) contract interface FungibleToken {

    /// The total number of tokens in existence.
    /// It is up to the implementer to ensure that the total supply
    /// stays accurate and up to date
    ///
    access(all) var totalSupply: Int

    /// TokensInitialized
    ///
    /// The event that is emitted when the contract is created
    ///
    access(all) event TokensInitialized(initialSupply: Int)

    /// TokensWithdrawn
    ///
    /// The event that is emitted when tokens are withdrawn from a Vault
    ///
    access(all) event TokensWithdrawn(amount: Int, from: Address?)

    /// TokensDeposited
    ///
    /// The event that is emitted when tokens are deposited into a Vault
    ///
    access(all) event TokensDeposited(amount: Int, to: Address?)

    /// Provider
    ///
    /// The interface that enforces the requirements for withdrawing
    /// tokens from the implementing type.
    ///
    /// It does not enforce requirements on 'balance' here,
    /// because it leaves open the possibility of creating custom providers
    /// that do not necessarily need their own balance.
    ///
    access(all) resource interface Provider {

        /// withdraw subtracts tokens from the owner's Vault
        /// and returns a Vault with the removed tokens.
        ///
        /// The function's access level is public, but this is not a problem
        /// because only the owner storing the resource in their account
        /// can initially call this function.
        ///
        /// The owner may grant other accounts access by creating a private
        /// capability that allows specific other users to access
        /// the provider resource through a reference.
        ///
        /// The owner may also grant all accounts access by creating a public
        /// capability that allows all users to access the provider
        /// resource through a reference.
        ///
        access(all) fun withdraw(amount: Int): @{Vault} {
            post {
                // 'result' refers to the return value
                result.balance == amount:
                    "Withdrawal amount must be the same as the balance of the withdrawn Vault"
            }
        }
    }

    /// Receiver
    ///
    /// The interface that enforces the requirements for depositing
    /// tokens into the implementing type.
    ///
    /// We do not include a condition that checks the balance because
    /// we want to give users the ability to make custom receivers that
    /// can do custom things with the tokens, like split them up and
    /// send them to different places.
    ///
    access(all) resource interface Receiver {

        /// deposit takes a Vault and deposits it into the implementing resource type
        ///
        access(all) fun deposit(from: @{Vault})
    }

    /// Balance
    ///
    /// The interface that contains the 'balance' field of the Vault
    /// and enforces that when new Vaults are created, the balance
    /// is initialized correctly.
    ///
    access(all) resource interface Balance {

        /// The total balance of a vault
        ///
        access(all) var balance: Int

        init(balance: Int) {
            post {
                self.balance == balance:
                    "Balance must be initialized to the initial balance"
            }
        }
    }

    /// Vault
    ///
    /// The resource that contains the functions to send and receive tokens.
    ///
    access(all) resource interface Vault: Provider, Receiver, Balance {

        // The declaration of a concrete type in a contract interface means that
        // every Fungible Token contract that implements the FungibleToken interface
        // must define a concrete 'Vault' resource that conforms to the 'Provider', 'Receiver',
        // and 'Balance' interfaces, and declares their required fields and functions

        /// The total balance of the vault
        ///
        access(all) var balance: Int

        // The conforming type must declare an initializer
        // that allows prioviding the initial balance of the Vault
        //
        init(balance: Int)

        /// withdraw subtracts 'amount' from the Vault's balance
        /// and returns a new Vault with the subtracted balance
        ///
        access(all) fun withdraw(amount: Int): @{Vault} {
            pre {
                self.balance >= amount:
                    "Amount withdrawn must be less than or equal than the balance of the Vault"
            }
            post {
                // use the special function 'before' to get the value of the 'balance' field
                // at the beginning of the function execution
                //
                self.balance == before(self.balance) - amount:
                    "New Vault balance must be the difference of the previous balance and the withdrawn Vault"
            }
        }

        /// deposit takes a Vault and adds its balance to the balance of this Vault
        ///
        access(all) fun deposit(from: @{Vault}) {
            post {
                self.balance == before(self.balance) + before(from.balance):
                    "New Vault balance must be the sum of the previous balance and the deposited Vault"
            }
        }
    }

    /// createEmptyVault allows any user to create a new Vault that has a zero balance
    ///
    access(all) fun createEmptyVault(): @{Vault} {
        post {
            result.balance == 0: "The newly created Vault must have zero balance"
        }
    }
}
`

const realFlowContract = `
import FungibleToken from 0x1

access(all) contract FlowToken: FungibleToken {

    // Total supply of Flow tokens in existence
    access(all) var totalSupply: Int

    // Vault
    //
    // Each user stores an instance of only the Vault in their storage
    // The functions in the Vault and governed by the pre and post conditions
    // in FungibleToken when they are called.
    // The checks happen at runtime whenever a function is called.
    //
    // Resources can only be created in the context of the contract that they
    // are defined in, so there is no way for a malicious user to create Vaults
    // out of thin air. A special Minter resource needs to be defined to mint
    // new tokens.
    //
    access(all) resource Vault: FungibleToken.Vault {

        // holds the balance of a users tokens
        access(all) var balance: Int

        // initialize the balance at resource creation time
        init(balance: Int) {
            self.balance = balance
        }

        // withdraw
        //
        // Function that takes an integer amount as an argument
        // and withdraws that amount from the Vault.
        // It creates a new temporary Vault that is used to hold
        // the money that is being transferred. It returns the newly
        // created Vault to the context that called so it can be deposited
        // elsewhere.
        //
        access(all) fun withdraw(amount: Int): @{FungibleToken.Vault} {
            self.balance = self.balance - amount
            return <-create Vault(balance: amount)
        }

        // deposit
        //
        // Function that takes a Vault object as an argument and adds
        // its balance to the balance of the owners Vault.
        // It is allowed to destroy the sent Vault because the Vault
        // was a temporary holder of the tokens. The Vault's balance has
        // been consumed and therefore can be destroyed.
        access(all) fun deposit(from: @{FungibleToken.Vault}) {
            let vault <- from as! @FlowToken.Vault
            self.balance = self.balance + vault.balance
            vault.balance = 0
            destroy vault
        }
    }

    // createEmptyVault
    //
    // Function that creates a new Vault with a balance of zero
    // and returns it to the calling context. A user must call this function
    // and store the returned Vault in their storage in order to allow their
    // account to be able to receive deposits of this token type.
    //
    access(all) fun createEmptyVault(): @{FungibleToken.Vault} {
        return <-create Vault(balance: 0)
    }

    access(all) resource Administrator {
        // createNewMinter
        //
        // Function that creates and returns a new minter resource
        //
        access(all) fun createNewMinter(allowedAmount: Int): @Minter {
            return <-create Minter(allowedAmount: allowedAmount)
        }

        // createNewBurner
        //
        // Function that creates and returns a new burner resource
        //
        access(all) fun createNewBurner(): @Burner {
            return <-create Burner()
        }
    }

    // Minter
    //
    // Resource object that token admin accounts can hold to mint new tokens.
    //
    access(all) resource Minter {

        // the amount of tokens that the minter is allowed to mint
        access(all) var allowedAmount: Int

        // mintTokens
        //
        // Function that mints new tokens, adds them to the total supply,
        // and returns them to the calling context.
        //
        access(all) fun mintTokens(amount: Int): @FlowToken.Vault {
            pre {
                amount > 0: "Amount minted must be greater than zero"
                amount <= self.allowedAmount: "Amount minted must be less than the allowed amount"
            }
            FlowToken.totalSupply = FlowToken.totalSupply + amount
            self.allowedAmount = self.allowedAmount - amount
            return <-create Vault(balance: amount)
        }

        init(allowedAmount: Int) {
            self.allowedAmount = allowedAmount
        }
    }

    // Burner
    //
    // Resource object that token admin accounts can hold to burn tokens.
    //
    access(all) resource Burner {

        // burnTokens
        //
        // Function that destroys a Vault instance, effectively burning the tokens.
        //
        // Note: the burned tokens are automatically subtracted from the
        // total supply in the Vault destructor.
        //
        access(all) fun burnTokens(from: @{FungibleToken.Vault}) {
            let vault <- from as! @FlowToken.Vault
            let amount = vault.balance
            destroy vault
        }
    }

    init(adminAccount: auth(Storage, Capabilities) &Account) {
        self.totalSupply = 0

        // Create the Vault with the total supply of tokens and save it in storage
        //
        let vault <- create Vault(balance: self.totalSupply)

        adminAccount.storage.save(<-vault, to: /storage/flowTokenVault)

        // Create a public capability to the stored Vault that only exposes
        // the 'deposit' method through the 'Receiver' interface
        //
        let receiverCapability = adminAccount.capabilities.storage.issue<&FlowToken.Vault>(/storage/flowTokenVault)
        adminAccount.capabilities.publish(receiverCapability, at: /public/flowTokenReceiver)

        // Create a public capability to the stored Vault that only exposes
        // the 'balance' field through the 'Balance' interface
        //
        let balanceCapability = adminAccount.capabilities.storage.issue<&FlowToken.Vault>(/storage/flowTokenVault)
        adminAccount.capabilities.publish(balanceCapability, at: /public/flowTokenBalance)

        let admin <- create Administrator()
        adminAccount.storage.save(<-admin, to: /storage/flowTokenAdmin)
    }
}
`

const realSetupFlowTokenAccountTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction {

    prepare(signer: auth(BorrowValue, IssueStorageCapabilityController, PublishCapability, SaveValue) &Account) {

        var storagePath = /storage/flowTokenVault

        if signer.storage.borrow<&FlowToken.Vault>(from: storagePath) != nil {
            return
        }

        // Create a new flowToken Vault and put it in storage
        signer.storage.save(<-FlowToken.createEmptyVault(), to: storagePath)

        // Create a public capability to the Vault that only exposes
        // the deposit function through the Receiver interface
        let vaultCap = signer.capabilities.storage.issue<&FlowToken.Vault>(storagePath)

        signer.capabilities.publish(
            vaultCap,
            at: /public/flowTokenReceiver
        )

        // Create a public capability to the Vault that only exposes
        // the balance field through the Balance interface
        let balanceCap = signer.capabilities.storage.issue<&FlowToken.Vault>(storagePath)

        signer.capabilities.publish(
            balanceCap,
            at: /public/flowTokenBalance
        )
    }
}
`

const realFlowTokenTransferTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction(amount: Int, to: Address) {
    let sentVault: @{FungibleToken.Vault}

    prepare(signer: auth(BorrowValue) &Account) {
        // Get a reference to the signer's stored vault
        let vaultRef = signer.storage.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault)
			?? panic("Could not borrow reference to the owner's Vault!")

        // Withdraw tokens from the signer's stored vault
        self.sentVault <- vaultRef.withdraw(amount: amount)
    }

    execute {
        // Get a reference to the recipient's Receiver
        let receiverRef =  getAccount(to)
            .capabilities.get<&{FungibleToken.Receiver}>(/public/flowTokenReceiver)
            .borrow()
			?? panic("Could not borrow receiver reference to the recipient's Vault")

        // Deposit the withdrawn tokens in the recipient's receiver
        receiverRef.deposit(from: <-self.sentVault)
    }
}
`

const realMintFlowTokenTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction(recipient: Address, amount: Int) {

    /// Reference to the FlowToken Minter Resource object
    let tokenAdmin: &FlowToken.Administrator

    /// Reference to the Fungible Token Receiver of the recipient
    let tokenReceiver: &{FungibleToken.Receiver}

    prepare(signer: auth(BorrowValue) &Account) {
         self.tokenAdmin = signer.storage
            .borrow<&FlowToken.Administrator>(from: /storage/flowTokenAdmin)
            ?? panic("Signer is not the token admin")

        self.tokenReceiver = getAccount(recipient)
            .capabilities.get<&{FungibleToken.Receiver}>(/public/flowTokenReceiver)
            .borrow()
            ?? panic("Unable to borrow receiver reference")
    }

    execute {
        let minter <- self.tokenAdmin.createNewMinter(allowedAmount: amount)
        let mintedVault <- minter.mintTokens(amount: amount)

        self.tokenReceiver.deposit(from: <-mintedVault)

        destroy minter
    }
}
`

const realFlowTokenBalanceScript = `
import FungibleToken from 0x1
import FlowToken from 0x1

access(all) fun main(account: Address): Int {

    let vaultRef = getAccount(account)
        .capabilities.get<&FlowToken.Vault>(/public/flowTokenBalance)
        .borrow()
        ?? panic("Could not borrow Balance reference to the Vault")

    return vaultRef.balance
}
`

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
