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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

const modifiedFungibleTokenContractInterface = `
/// FungibleToken
///
/// The interface that fungible token contracts implement.
///
access(all) contract interface FungibleToken {

    /// The total number of tokens in existence.
    /// It is up to the implementer to ensure that the total supply
    /// stays accurate and up to date
    ///
    access(all) var totalSupply: UFix64

    /// TokensInitialized
    ///
    /// The event that is emitted when the contract is created
    ///
    access(all) event TokensInitialized(initialSupply: UFix64)

    /// TokensWithdrawn
    ///
    /// The event that is emitted when tokens are withdrawn from a Vault
    ///
    access(all) event TokensWithdrawn(amount: UFix64, from: Address?)

    /// TokensDeposited
    ///
    /// The event that is emitted when tokens are deposited into a Vault
    ///
    access(all) event TokensDeposited(amount: UFix64, to: Address?)

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
        access(all) fun withdraw(amount: UFix64): @{Vault} {
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
        access(all) var balance: UFix64

        init(balance: UFix64) {
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
        access(all) var balance: UFix64

        // The conforming type must declare an initializer
        // that allows prioviding the initial balance of the Vault
        //
        init(balance: UFix64)

        /// withdraw subtracts 'amount' from the Vault's balance
        /// and returns a new Vault with the subtracted balance
        ///
        access(all) fun withdraw(amount: UFix64): @{Vault} {
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
            result.balance == 0.0: "The newly created Vault must have zero balance"
        }
    }
}
`

const modifiedFlowContract = `
import FungibleToken from 0x1

access(all) contract FlowToken: FungibleToken {

    // Total supply of Flow tokens in existence
    access(all) var totalSupply: UFix64

    // Event that is emitted when the contract is created
    access(all) event TokensInitialized(initialSupply: UFix64)

    // Event that is emitted when tokens are withdrawn from a Vault
    access(all) event TokensWithdrawn(amount: UFix64, from: Address?)

    // Event that is emitted when tokens are deposited to a Vault
    access(all) event TokensDeposited(amount: UFix64, to: Address?)

    // Event that is emitted when new tokens are minted
    access(all) event TokensMinted(amount: UFix64)

    // Event that is emitted when tokens are destroyed
    access(all) event TokensBurned(amount: UFix64)

    // Event that is emitted when a new minter resource is created
    access(all) event MinterCreated(allowedAmount: UFix64)

    // Event that is emitted when a new burner resource is created
    access(all) event BurnerCreated()

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
        access(all) var balance: UFix64

        // initialize the balance at resource creation time
        init(balance: UFix64) {
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
        access(all) fun withdraw(amount: UFix64): @{FungibleToken.Vault} {
            self.balance = self.balance - amount
            emit TokensWithdrawn(amount: amount, from: self.owner?.address)
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
            emit TokensDeposited(amount: vault.balance, to: self.owner?.address)
            vault.balance = 0.0
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
        return <-create Vault(balance: 0.0)
    }

    access(all) resource Administrator {
        // createNewMinter
        //
        // Function that creates and returns a new minter resource
        //
        access(all) fun createNewMinter(allowedAmount: UFix64): @Minter {
            emit MinterCreated(allowedAmount: allowedAmount)
            return <-create Minter(allowedAmount: allowedAmount)
        }

        // createNewBurner
        //
        // Function that creates and returns a new burner resource
        //
        access(all) fun createNewBurner(): @Burner {
            emit BurnerCreated()
            return <-create Burner()
        }
    }

    // Minter
    //
    // Resource object that token admin accounts can hold to mint new tokens.
    //
    access(all) resource Minter {

        // the amount of tokens that the minter is allowed to mint
        access(all) var allowedAmount: UFix64

        // mintTokens
        //
        // Function that mints new tokens, adds them to the total supply,
        // and returns them to the calling context.
        //
        access(all) fun mintTokens(amount: UFix64): @FlowToken.Vault {
            pre {
                amount > UFix64(0): "Amount minted must be greater than zero"
                amount <= self.allowedAmount: "Amount minted must be less than the allowed amount"
            }
            FlowToken.totalSupply = FlowToken.totalSupply + amount
            self.allowedAmount = self.allowedAmount - amount
            emit TokensMinted(amount: amount)
            return <-create Vault(balance: amount)
        }

        init(allowedAmount: UFix64) {
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
            emit TokensBurned(amount: amount)
        }
    }

    init(adminAccount: auth(Storage, Capabilities) &Account) {
        self.totalSupply = 0.0

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

        // Emit an event that shows that the contract was initialized
        emit TokensInitialized(initialSupply: self.totalSupply)
    }
}
`

const realSetupFlowTokenAccountTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction {

    prepare(signer: &Account) {

        if signer.storage.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault) == nil {
            // Create a new flowToken Vault and put it in storage
            signer.storage.save(<-FlowToken.createEmptyVault(), to: /storage/flowTokenVault)

            // Create a public capability to the Vault that only exposes
            // the deposit function through the Receiver interface
            signer.link<&FlowToken.Vault>(
                /public/flowTokenReceiver,
                target: /storage/flowTokenVault
            )

            // Create a public capability to the Vault that only exposes
            // the balance field through the Balance interface
            signer.link<&FlowToken.Vault>(
                /public/flowTokenBalance,
                target: /storage/flowTokenVault
            )
        }
    }
}
`

const realMintFlowTokenTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction(recipient: Address, amount: UFix64) {
    let tokenAdmin: &FlowToken.Administrator
    let tokenReceiver: &{FungibleToken.Receiver}

    prepare(signer: &Account) {
        self.tokenAdmin = signer
            .borrow<&FlowToken.Administrator>(from: /storage/flowTokenAdmin)
            ?? panic("Signer is not the token admin")

        self.tokenReceiver = getAccount(recipient)
            .getCapability(/public/flowTokenReceiver)
            .borrow<&{FungibleToken.Receiver}>()
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

const realFlowTokenTransferTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction(amount: UFix64, to: Address) {

    // The Vault resource that holds the tokens that are being transferred
    let sentVault: @{FungibleToken.Vault}

    prepare(signer: &Account) {

        // Get a reference to the signer's stored vault
        let vaultRef = signer.storage.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault)
			?? panic("Could not borrow reference to the owner's Vault!")

        // Withdraw tokens from the signer's stored vault
        self.sentVault <- vaultRef.withdraw(amount: amount)
    }

    execute {

        // Get a reference to the recipient's Receiver
        let receiverRef =  getAccount(to)
            .getCapability(/public/flowTokenReceiver)
            .borrow<&{FungibleToken.Receiver}>()
			?? panic("Could not borrow receiver reference to the recipient's Vault")

        // Deposit the withdrawn tokens in the recipient's receiver
        receiverRef.deposit(from: <-self.sentVault)
    }
}
`

const realFlowTokenBalanceScript = `
import FungibleToken from 0x1
import FlowToken from 0x1

access(all) fun main(account: Address): UFix64 {

    let vaultRef = getAccount(account)
        .getCapability(/public/flowTokenBalance)
        .borrow<&FlowToken.Vault>()
        ?? panic("Could not borrow Balance reference to the Vault")

    return vaultRef.balance
}
`

func BenchmarkRuntimeFungibleTokenTransfer(b *testing.B) {

	runtime := NewTestInterpreterRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	accountCodes := map[Location][]byte{}

	var events []cadence.Event

	signerAccount := contractsAddress

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
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
	}

	environment := NewBaseInterpreterEnvironment(Config{})

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy Fungible Token contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction(
				"FungibleToken",
				[]byte(modifiedFungibleTokenContractInterface),
			),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(b, err)

	// Deploy Flow Token contract

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(fmt.Sprintf(
				`
                  transaction {

                      prepare(signer: &Account) {
                          signer.contracts.add(name: "FlowToken", code: "%s".decodeHex(), signer)
                      }
                  }
                `,
				hex.EncodeToString([]byte(modifiedFlowContract)),
			)),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
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
			},
		)
		require.NoError(b, err)
	}

	// Mint 1000 FLOW to sender

	mintAmount, err := cadence.NewUFix64("100000000000.0")
	require.NoError(b, err)

	mintAmountValue := interpreter.NewUnmeteredUFix64Value(uint64(mintAmount))

	signerAccount = contractsAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(realMintFlowTokenTransaction),
			Arguments: encodeArgs(
				cadence.Address(senderAddress),
				mintAmount,
			),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(b, err)

	// Benchmark sending tokens from sender to receiver

	sendAmount, err := cadence.NewUFix64("0.00000001")
	require.NoError(b, err)

	signerAccount = senderAddress

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(realFlowTokenTransferTransaction),
				Arguments: encodeArgs(
					sendAmount,
					cadence.Address(receiverAddress),
				),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
		require.NoError(b, err)
	}

	b.StopTimer()

	// Run validation scripts

	sum := interpreter.NewUnmeteredUFix64ValueWithInteger(0, interpreter.EmptyLocationRange)

	inter := NewTestInterpreter(b)

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {

		result, err := runtime.ExecuteScript(
			Script{
				Source: []byte(realFlowTokenBalanceScript),
				Arguments: encodeArgs(
					cadence.Address(address),
				),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
		require.NoError(b, err)

		value := interpreter.NewUnmeteredUFix64Value(uint64(result.(cadence.UFix64)))

		require.True(b, bool(value.Less(inter, mintAmountValue, interpreter.EmptyLocationRange)))

		sum = sum.Plus(inter, value, interpreter.EmptyLocationRange).(interpreter.UFix64Value)
	}

	utils.RequireValuesEqual(b, nil, mintAmountValue, sum)
}
