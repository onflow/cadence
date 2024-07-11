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

package vm

import (
	"fmt"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/bbq/compiler"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/checker"
)

func TestFTTransfer(t *testing.T) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")
	ftChecker, err := ParseAndCheckWithOptions(t, realFungibleTokenContractInterface,
		ParseAndCheckOptions{
			Location: ftLocation,
		},
	)
	require.NoError(t, err)

	ftCompiler := compiler.NewCompiler(ftChecker.Program, ftChecker.Elaboration)
	ftProgram := ftCompiler.Compile()

	// ----- Deploy FlowToken Contract -----

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")
	flowTokenChecker, err := ParseAndCheckWithOptions(t, realFlowContract,
		ParseAndCheckOptions{
			Location: flowTokenLocation,
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
					switch location {
					case ftLocation:
						return sema.ElaborationImport{
							Elaboration: ftChecker.Elaboration,
						}, nil
					default:
						return nil, fmt.Errorf("cannot find contract in location %s", location)
					}
				},
				LocationHandler: singleIdentifierLocationResolver(t),
			},
		},
	)
	require.NoError(t, err)

	flowTokenCompiler := compiler.NewCompiler(flowTokenChecker.Program, flowTokenChecker.Elaboration)
	flowTokenCompiler.Config.ImportHandler = func(location common.Location) *bbq.Program {
		return ftProgram
	}

	flowTokenProgram := flowTokenCompiler.Compile()
	printProgram(flowTokenProgram)

	flowTokenVM := NewVM(
		flowTokenProgram,
		&Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
		},
	)

	authAccount := NewAuthAccountReferenceValue(contractsAddress)

	flowTokenContractValue, err := flowTokenVM.InitializeContract(authAccount)
	require.NoError(t, err)

	// ----- Run setup account transaction -----

	checkerImportHandler := func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
		require.IsType(t, common.AddressLocation{}, location)
		addressLocation := location.(common.AddressLocation)
		var elaboration *sema.Elaboration

		switch addressLocation {
		case ftLocation:
			elaboration = ftChecker.Elaboration
		case flowTokenLocation:
			elaboration = flowTokenChecker.Elaboration
		default:
			assert.FailNow(t, "invalid location")
		}

		return sema.ElaborationImport{
			Elaboration: elaboration,
		}, nil
	}

	compilerImportHandler := func(location common.Location) *bbq.Program {
		switch location {
		case ftLocation:
			return ftProgram
		case flowTokenLocation:
			return flowTokenProgram
		default:
			assert.FailNow(t, "invalid location")
			return nil
		}
	}

	vmConfig := &Config{
		Storage: storage,
		ImportHandler: func(location common.Location) *bbq.Program {
			switch location {
			case ftLocation:
				return ftProgram
			case flowTokenLocation:
				return flowTokenProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		},
		ContractValueHandler: func(_ *Config, location common.Location) *CompositeValue {
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
	}

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		setupTxChecker, err := ParseAndCheckWithOptions(
			t,
			realSetupFlowTokenAccountTransaction,
			ParseAndCheckOptions{
				Config: &sema.Config{
					ImportHandler:   checkerImportHandler,
					LocationHandler: singleIdentifierLocationResolver(t),
				},
			},
		)
		require.NoError(t, err)

		setupTxCompiler := compiler.NewCompiler(setupTxChecker.Program, setupTxChecker.Elaboration)
		setupTxCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
		setupTxCompiler.Config.ImportHandler = compilerImportHandler

		program := setupTxCompiler.Compile()
		printProgram(program)

		setupTxVM := NewVM(program, vmConfig)

		authorizer := NewAuthAccountReferenceValue(address)
		err = setupTxVM.ExecuteTransaction(nil, authorizer)
		require.NoError(t, err)
		require.Empty(t, setupTxVM.stack)
	}

	// Mint FLOW to sender

	mintTxChecker, err := ParseAndCheckWithOptions(
		t,
		realMintFlowTokenTransaction,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler:              checkerImportHandler,
				BaseValueActivationHandler: baseActivation,
				LocationHandler:            singleIdentifierLocationResolver(t),
			},
		},
	)
	require.NoError(t, err)

	mintTxCompiler := compiler.NewCompiler(mintTxChecker.Program, mintTxChecker.Elaboration)
	mintTxCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
	mintTxCompiler.Config.ImportHandler = compilerImportHandler

	program := mintTxCompiler.Compile()
	printProgram(program)

	mintTxVM := NewVM(program, vmConfig)

	total := int64(1000000)

	mintTxArgs := []Value{
		IntValue{total},
		AddressValue(senderAddress),
	}

	mintTxAuthorizer := NewAuthAccountReferenceValue(contractsAddress)
	err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
	require.NoError(t, err)
	require.Empty(t, mintTxVM.stack)

	// ----- Run token transfer transaction -----

	tokenTransferTxChecker, err := ParseAndCheckWithOptions(
		t,
		realFlowTokenTransferTransaction,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler:              checkerImportHandler,
				BaseValueActivationHandler: baseActivation,
				LocationHandler:            singleIdentifierLocationResolver(t),
			},
		},
	)
	require.NoError(t, err)

	tokenTransferTxCompiler := compiler.NewCompiler(tokenTransferTxChecker.Program, tokenTransferTxChecker.Elaboration)
	tokenTransferTxCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
	tokenTransferTxCompiler.Config.ImportHandler = compilerImportHandler

	tokenTransferTxProgram := tokenTransferTxCompiler.Compile()
	printProgram(tokenTransferTxProgram)

	tokenTransferTxVM := NewVM(tokenTransferTxProgram, vmConfig)

	transferAmount := int64(1)

	tokenTransferTxArgs := []Value{
		IntValue{transferAmount},
		AddressValue(receiverAddress),
	}

	tokenTransferTxAuthorizer := NewAuthAccountReferenceValue(senderAddress)
	err = tokenTransferTxVM.ExecuteTransaction(tokenTransferTxArgs, tokenTransferTxAuthorizer)
	require.NoError(t, err)
	require.Empty(t, tokenTransferTxVM.stack)

	// Run validation scripts

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		validationScriptChecker, err := ParseAndCheckWithOptions(
			t,
			realFlowTokenBalanceScript,
			ParseAndCheckOptions{
				Config: &sema.Config{
					ImportHandler:              checkerImportHandler,
					BaseValueActivationHandler: baseActivation,
					LocationHandler:            singleIdentifierLocationResolver(t),
				},
			},
		)
		require.NoError(t, err)

		validationScriptCompiler := compiler.NewCompiler(validationScriptChecker.Program, validationScriptChecker.Elaboration)
		validationScriptCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
		validationScriptCompiler.Config.ImportHandler = compilerImportHandler

		program := validationScriptCompiler.Compile()
		printProgram(program)

		validationScriptVM := NewVM(program, vmConfig)

		addressValue := AddressValue(address)
		result, err := validationScriptVM.Invoke("main", addressValue)
		require.NoError(t, err)
		require.Empty(t, validationScriptVM.stack)

		if address == senderAddress {
			assert.Equal(t, IntValue{total - transferAmount}, result)
		} else {
			assert.Equal(t, IntValue{transferAmount}, result)
		}
	}
}

func baseActivation(common.Location) *sema.VariableActivation {
	// Only need to make the checker happy
	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)
	baseValueActivation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
		"getAccount",
		stdlib.GetAccountFunctionType,
		"",
		nil,
	))
	return baseValueActivation
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
        let receiverCap = adminAccount.capabilities.storage
            .issue<&FlowToken.Vault>(/storage/flowTokenVault)
        adminAccount.capabilities.publish(receiverCap, at: /public/flowTokenReceiver)

        // Create a public capability to the stored Vault that only exposes
        // the 'balance' field through the 'Balance' interface
        //
        let balanceCap = adminAccount.capabilities.storage
            .issue<&FlowToken.Vault>(/storage/flowTokenVault)
        adminAccount.capabilities.publish(balanceCap, at: /public/flowTokenBalance)

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

        if signer.storage.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault) != nil {
            return
        }

        var storagePath = /storage/flowTokenVault

        // Create a new flowToken Vault and put it in storage
        signer.storage.save(<-FlowToken.createEmptyVault(), to: storagePath)

        // Create a public capability to the Vault that exposes the Vault interfaces
        let vaultCap = signer.capabilities.storage.issue<&FlowToken.Vault>(
            storagePath
        )
        signer.capabilities.publish(vaultCap, at: /public/flowTokenVault)

        // Create a public Capability to the Vault's Receiver functionality
        let receiverCap = signer.capabilities.storage.issue<&FlowToken.Vault>(
            storagePath
        )
        signer.capabilities.publish(receiverCap, at: /public/flowTokenReceiver)
    }
}
`

const realFlowTokenTransferTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction(recipient: Address, amount: Int) {
    let tokenAdmin: &FlowToken.Administrator
    let tokenReceiver: &{FungibleToken.Receiver}

    prepare(signer: auth(BorrowValue) &Account) {
        self.tokenAdmin = signer.storage.borrow<&FlowToken.Administrator>(from: /storage/flowTokenAdmin)
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

const realMintFlowTokenTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction(amount: Int, to: Address) {

    // The Vault resource that holds the tokens that are being transferred
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

const realFlowTokenBalanceScript = `
import FungibleToken from 0x1
import FlowToken from 0x1

access(all) fun main(account: Address): Int {

    let vaultRef = getAccount(account)
        .getCapability(/public/flowTokenBalance)
        .borrow<&FlowToken.Vault>()
        ?? panic("Could not borrow Balance reference to the Vault")

    return vaultRef.balance
}
`

func BenchmarkFTTransfer(b *testing.B) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")
	ftChecker, err := ParseAndCheckWithOptions(b, realFungibleTokenContractInterface,
		ParseAndCheckOptions{
			Location: ftLocation,
		},
	)
	require.NoError(b, err)

	ftCompiler := compiler.NewCompiler(ftChecker.Program, ftChecker.Elaboration)
	ftProgram := ftCompiler.Compile()

	// ----- Deploy FlowToken Contract -----

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")
	flowTokenChecker, err := ParseAndCheckWithOptions(b, realFlowContract,
		ParseAndCheckOptions{
			Location: flowTokenLocation,
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
					switch location {
					case ftLocation:
						return sema.ElaborationImport{
							Elaboration: ftChecker.Elaboration,
						}, nil
					default:
						return nil, fmt.Errorf("cannot find contract in location %s", location)
					}
				},
				LocationHandler: singleIdentifierLocationResolver(b),
			},
		},
	)
	require.NoError(b, err)

	flowTokenCompiler := compiler.NewCompiler(flowTokenChecker.Program, flowTokenChecker.Elaboration)
	flowTokenCompiler.Config.ImportHandler = func(location common.Location) *bbq.Program {
		return ftProgram
	}

	flowTokenProgram := flowTokenCompiler.Compile()

	flowTokenVM := NewVM(
		flowTokenProgram,
		&Config{
			Storage: storage,
		},
	)

	authAccount := NewAuthAccountReferenceValue(contractsAddress)

	flowTokenContractValue, err := flowTokenVM.InitializeContract(authAccount)
	require.NoError(b, err)

	// ----- Run setup account transaction -----

	checkerImportHandler := func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
		require.IsType(b, common.AddressLocation{}, location)
		addressLocation := location.(common.AddressLocation)
		var elaboration *sema.Elaboration

		switch addressLocation {
		case ftLocation:
			elaboration = ftChecker.Elaboration
		case flowTokenLocation:
			elaboration = flowTokenChecker.Elaboration
		default:
			assert.FailNow(b, "invalid location")
		}

		return sema.ElaborationImport{
			Elaboration: elaboration,
		}, nil
	}

	compilerImportHandler := func(location common.Location) *bbq.Program {
		switch location {
		case ftLocation:
			return ftProgram
		case flowTokenLocation:
			return flowTokenProgram
		default:
			assert.FailNow(b, "invalid location")
			return nil
		}
	}

	vmConfig := &Config{
		Storage: storage,
		ImportHandler: func(location common.Location) *bbq.Program {
			switch location {
			case ftLocation:
				return ftProgram
			case flowTokenLocation:
				return flowTokenProgram
			default:
				assert.FailNow(b, "invalid location")
				return nil
			}
		},
		ContractValueHandler: func(_ *Config, location common.Location) *CompositeValue {
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
	}

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		setupTxChecker, err := ParseAndCheckWithOptions(
			b,
			realSetupFlowTokenAccountTransaction,
			ParseAndCheckOptions{
				Config: &sema.Config{
					ImportHandler:   checkerImportHandler,
					LocationHandler: singleIdentifierLocationResolver(b),
				},
			},
		)
		require.NoError(b, err)

		setupTxCompiler := compiler.NewCompiler(setupTxChecker.Program, setupTxChecker.Elaboration)
		setupTxCompiler.Config.LocationHandler = singleIdentifierLocationResolver(b)
		setupTxCompiler.Config.ImportHandler = compilerImportHandler

		program := setupTxCompiler.Compile()

		setupTxVM := NewVM(program, vmConfig)

		authorizer := NewAuthAccountReferenceValue(address)
		err = setupTxVM.ExecuteTransaction(nil, authorizer)
		require.NoError(b, err)
	}

	// Mint FLOW to sender

	mintTxChecker, err := ParseAndCheckWithOptions(
		b,
		realMintFlowTokenTransaction,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler:              checkerImportHandler,
				BaseValueActivationHandler: baseActivation,
				LocationHandler:            singleIdentifierLocationResolver(b),
			},
		},
	)
	require.NoError(b, err)

	mintTxCompiler := compiler.NewCompiler(mintTxChecker.Program, mintTxChecker.Elaboration)
	mintTxCompiler.Config.LocationHandler = singleIdentifierLocationResolver(b)
	mintTxCompiler.Config.ImportHandler = compilerImportHandler

	program := mintTxCompiler.Compile()

	mintTxVM := NewVM(program, vmConfig)

	total := int64(1000000)

	mintTxArgs := []Value{
		AddressValue(senderAddress),
		IntValue{total},
	}

	mintTxAuthorizer := NewAuthAccountReferenceValue(contractsAddress)
	err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
	require.NoError(b, err)

	// ----- Run token transfer transaction -----

	tokenTransferTxChecker, err := ParseAndCheckWithOptions(
		b,
		realFlowTokenTransferTransaction,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler:              checkerImportHandler,
				BaseValueActivationHandler: baseActivation,
				LocationHandler:            singleIdentifierLocationResolver(b),
			},
		},
	)
	require.NoError(b, err)

	transferAmount := int64(1)

	tokenTransferTxArgs := []Value{
		IntValue{transferAmount},
		AddressValue(receiverAddress),
	}

	tokenTransferTxAuthorizer := NewAuthAccountReferenceValue(senderAddress)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tokenTransferTxCompiler := compiler.NewCompiler(tokenTransferTxChecker.Program, tokenTransferTxChecker.Elaboration)
		tokenTransferTxCompiler.Config.LocationHandler = singleIdentifierLocationResolver(b)
		tokenTransferTxCompiler.Config.ImportHandler = compilerImportHandler

		tokenTransferTxProgram := tokenTransferTxCompiler.Compile()

		tokenTransferTxVM := NewVM(tokenTransferTxProgram, vmConfig)
		err = tokenTransferTxVM.ExecuteTransaction(tokenTransferTxArgs, tokenTransferTxAuthorizer)
		require.NoError(b, err)
	}

	b.StopTimer()
}
