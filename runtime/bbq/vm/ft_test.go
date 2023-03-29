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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/bbq/compiler"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/checker"
)

func TestFTTransfer(t *testing.T) {

	// Deploy FT Contract
	ftLocation := common.NewAddressLocation(nil, common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}, "FungibleToken")
	ftChecker, err := ParseAndCheckWithOptions(t, realFungibleTokenContractInterface,
		ParseAndCheckOptions{Location: ftLocation},
	)
	require.NoError(t, err)

	ftCompiler := compiler.NewCompiler(ftChecker.Program, ftChecker.Elaboration)
	ftProgram := ftCompiler.Compile()

	//vm := NewVM(ftProgram, nil)
	//importedContractValue, err := vm.InitializeContract()
	//require.NoError(t, err)

	// Deploy FlowToken Contract
	flowTokenLocation := common.NewAddressLocation(nil, common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}, "FlowToken")
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

	vm := NewVM(flowTokenProgram, nil)

	authAcount := NewCompositeValue(
		nil,
		"AuthAccount",
		common.CompositeKindStructure,
		common.Address{},
		vm.config.Storage,
	)

	printProgram(flowTokenProgram)

	flowTokenContractValue, err := vm.InitializeContract(authAcount)
	require.NoError(t, err)

	// Run script

	checker, err := ParseAndCheckWithOptions(t, `
      import FungibleToken from 0x01

      fun test(): String {
          return "hello"
      }
  `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
					switch location {
					case flowTokenLocation:
						return sema.ElaborationImport{
							Elaboration: flowTokenChecker.Elaboration,
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

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
		return flowTokenProgram
	}

	program := comp.Compile()

	vmConfig := &Config{
		ImportHandler: func(location common.Location) *bbq.Program {
			return ftProgram
		},
		ContractValueHandler: func(*Config, common.Location) *CompositeValue {
			return flowTokenContractValue
		},
	}

	vm = NewVM(program, vmConfig)

	result, err := vm.Invoke("test")
	require.NoError(t, err)
	require.Equal(t, StringValue{Str: []byte("global function of the imported program")}, result)
}

const realFungibleTokenContractInterface = `
/// FungibleToken
///
/// The interface that fungible token contracts implement.
///
pub contract interface FungibleToken {

    /// The total number of tokens in existence.
    /// It is up to the implementer to ensure that the total supply
    /// stays accurate and up to date
    ///
    pub var totalSupply: Int

    ///// TokensInitialized
    /////
    ///// The event that is emitted when the contract is created
    /////
    //pub event TokensInitialized(initialSupply: Int)
	//
    ///// TokensWithdrawn
    /////
    ///// The event that is emitted when tokens are withdrawn from a Vault
    /////
    //pub event TokensWithdrawn(amount: Int, from: Address?)
	//
    ///// TokensDeposited
    /////
    ///// The event that is emitted when tokens are deposited into a Vault
    /////
    //pub event TokensDeposited(amount: Int, to: Address?)

    /// Provider
    ///
    /// The interface that enforces the requirements for withdrawing
    /// tokens from the implementing type.
    ///
    /// It does not enforce requirements on 'balance' here,
    /// because it leaves open the possibility of creating custom providers
    /// that do not necessarily need their own balance.
    ///
    pub resource interface Provider {

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
        pub fun withdraw(amount: Int): @Vault {
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
    pub resource interface Receiver {

        /// deposit takes a Vault and deposits it into the implementing resource type
        ///
        pub fun deposit(from: @Vault)
    }

    /// Balance
    ///
    /// The interface that contains the 'balance' field of the Vault
    /// and enforces that when new Vaults are created, the balance
    /// is initialized correctly.
    ///
    pub resource interface Balance {

        /// The total balance of a vault
        ///
        pub var balance: Int

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
    pub resource Vault: Provider, Receiver, Balance {

        // The declaration of a concrete type in a contract interface means that
        // every Fungible Token contract that implements the FungibleToken interface
        // must define a concrete 'Vault' resource that conforms to the 'Provider', 'Receiver',
        // and 'Balance' interfaces, and declares their required fields and functions

        /// The total balance of the vault
        ///
        pub var balance: Int

        // The conforming type must declare an initializer
        // that allows prioviding the initial balance of the Vault
        //
        init(balance: Int)

        /// withdraw subtracts 'amount' from the Vault's balance
        /// and returns a new Vault with the subtracted balance
        ///
        pub fun withdraw(amount: Int): @Vault {
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
        pub fun deposit(from: @Vault) {
            post {
                self.balance == before(self.balance) + before(from.balance):
                    "New Vault balance must be the sum of the previous balance and the deposited Vault"
            }
        }
    }

    /// createEmptyVault allows any user to create a new Vault that has a zero balance
    ///
    pub fun createEmptyVault(): @Vault {
        post {
            result.balance == 0: "The newly created Vault must have zero balance"
        }
    }
}
`

const realFlowContract = `
import FungibleToken from 0x1

pub contract FlowToken: FungibleToken {

    // Total supply of Flow tokens in existence
    pub var totalSupply: Int
	
    //// Event that is emitted when the contract is created
    //pub event TokensInitialized(initialSupply: Int)
	//
    //// Event that is emitted when tokens are withdrawn from a Vault
    //pub event TokensWithdrawn(amount: Int, from: Address?)
	//
    //// Event that is emitted when tokens are deposited to a Vault
    //pub event TokensDeposited(amount: Int, to: Address?)
	//
    //// Event that is emitted when new tokens are minted
    //pub event TokensMinted(amount: Int)
	//
    //// Event that is emitted when tokens are destroyed
    //pub event TokensBurned(amount: Int)
	//
    //// Event that is emitted when a new minter resource is created
    //pub event MinterCreated(allowedAmount: Int)
	//
    //// Event that is emitted when a new burner resource is created
    //pub event BurnerCreated()

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
    pub resource Vault: FungibleToken.Provider, FungibleToken.Receiver, FungibleToken.Balance {

        // holds the balance of a users tokens
        pub var balance: Int

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
        pub fun withdraw(amount: Int): @FungibleToken.Vault {
            self.balance = self.balance - amount
            // emit TokensWithdrawn(amount: amount, from: self.owner?.address)
            return <-create Vault(balance: amount)
        }

        // deposit
        //
        // Function that takes a Vault object as an argument and adds
        // its balance to the balance of the owners Vault.
        // It is allowed to destroy the sent Vault because the Vault
        // was a temporary holder of the tokens. The Vault's balance has
        // been consumed and therefore can be destroyed.
        pub fun deposit(from: @FungibleToken.Vault) {
            let vault <- from as! @FlowToken.Vault
            self.balance = self.balance + vault.balance
            // emit TokensDeposited(amount: vault.balance, to: self.owner?.address)
            vault.balance = 0
            destroy vault
        }

        destroy() {
            FlowToken.totalSupply = FlowToken.totalSupply - self.balance
        }
    }

    // createEmptyVault
    //
    // Function that creates a new Vault with a balance of zero
    // and returns it to the calling context. A user must call this function
    // and store the returned Vault in their storage in order to allow their
    // account to be able to receive deposits of this token type.
    //
    pub fun createEmptyVault(): @FungibleToken.Vault {
        return <-create Vault(balance: 0)
    }

    pub resource Administrator {

        // createNewMinter
        //
        // Function that creates and returns a new minter resource
        //
        pub fun createNewMinter(allowedAmount: Int): @Minter {
            // emit MinterCreated(allowedAmount: allowedAmount)
            return <-create Minter(allowedAmount: allowedAmount)
        }

        // createNewBurner
        //
        // Function that creates and returns a new burner resource
        //
        pub fun createNewBurner(): @Burner {
            // emit BurnerCreated()
            return <-create Burner()
        }
    }

    // Minter
    //
    // Resource object that token admin accounts can hold to mint new tokens.
    //
    pub resource Minter {

        // the amount of tokens that the minter is allowed to mint
        pub var allowedAmount: Int

        // mintTokens
        //
        // Function that mints new tokens, adds them to the total supply,
        // and returns them to the calling context.
        //
        pub fun mintTokens(amount: Int): @FlowToken.Vault {
            pre {
                amount > Int(0): "Amount minted must be greater than zero"
                amount <= self.allowedAmount: "Amount minted must be less than the allowed amount"
            }
            FlowToken.totalSupply = FlowToken.totalSupply + amount
            self.allowedAmount = self.allowedAmount - amount
            // emit TokensMinted(amount: amount)
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
    pub resource Burner {

        // burnTokens
        //
        // Function that destroys a Vault instance, effectively burning the tokens.
        //
        // Note: the burned tokens are automatically subtracted from the
        // total supply in the Vault destructor.
        //
        pub fun burnTokens(from: @FungibleToken.Vault) {
            let vault <- from as! @FlowToken.Vault
            let amount = vault.balance
            destroy vault
            // emit TokensBurned(amount: amount)
        }
    }

    init(adminAccount: AuthAccount) {
        self.totalSupply = 0

        // Create the Vault with the total supply of tokens and save it in storage
        //
        let vault <- create Vault(balance: self.totalSupply)
        adminAccount.save(<-vault, to: /storage/flowTokenVault)

        // Create a public capability to the stored Vault that only exposes
        // the 'deposit' method through the 'Receiver' interface
        //
        adminAccount.link<&FlowToken.Vault{FungibleToken.Receiver}>(
            /public/flowTokenReceiver,
            target: /storage/flowTokenVault
        )

        // Create a public capability to the stored Vault that only exposes
        // the 'balance' field through the 'Balance' interface
        //
        adminAccount.link<&FlowToken.Vault{FungibleToken.Balance}>(
            /public/flowTokenBalance,
            target: /storage/flowTokenVault
        )

        let admin <- create Administrator()
        adminAccount.save(<-admin, to: /storage/flowTokenAdmin)

        // Emit an event that shows that the contract was initialized
        // emit TokensInitialized(initialSupply: self.totalSupply)
    }
}
`
