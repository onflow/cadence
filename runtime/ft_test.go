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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

const modifiedFungibleTokenContractInterface = `
/// FungibleToken
///
/// Fungible Token implementations should implement the fungible token
/// interface.
access(all) contract interface FungibleToken {

    // An entitlement for allowing the withdrawal of tokens from a Vault
    access(all) entitlement Withdraw

    /// The event that is emitted when tokens are withdrawn from a Vault
    access(all) event Withdrawn(type: String, amount: UFix64, from: Address?, fromUUID: UInt64, withdrawnUUID: UInt64, balanceAfter: UFix64)

    /// The event that is emitted when tokens are deposited to a Vault
    access(all) event Deposited(type: String, amount: UFix64, to: Address?, toUUID: UInt64, depositedUUID: UInt64, balanceAfter: UFix64)

    /// Event that is emitted when the global burn method is called with a non-zero balance
    access(all) event Burned(type: String, amount: UFix64, fromUUID: UInt64)

    /// Balance
    ///
    /// The interface that provides standard functions\
    /// for getting balance information
    ///
    access(all) resource interface Balance {
        access(all) var balance: UFix64
    }

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

        /// Function to ask a provider if a specific amount of tokens
        /// is available to be withdrawn
        /// This could be useful to avoid panicing when calling withdraw
        /// when the balance is unknown
        /// Additionally, if the provider is pulling from multiple vaults
        /// it only needs to check some of the vaults until the desired amount
        /// is reached, potentially helping with performance.
        ///
        access(all) view fun isAvailableToWithdraw(amount: UFix64): Bool

        /// withdraw subtracts tokens from the implementing resource
        /// and returns a Vault with the removed tokens.
        ///
        /// The function's access level is 'access(Withdraw)'
        /// So in order to access it, one would either need the object itself
        /// or an entitled reference with 'Withdraw'.
        ///
        access(Withdraw) fun withdraw(amount: UFix64): @{Vault} {
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

        /// getSupportedVaultTypes returns a dictionary of Vault types
        /// and whether the type is currently supported by this Receiver
        access(all) view fun getSupportedVaultTypes(): {Type: Bool}

        /// Returns whether or not the given type is accepted by the Receiver
        /// A vault that can accept any type should just return true by default
        access(all) view fun isSupportedVaultType(type: Type): Bool
    }

    /// Vault
    ///
    /// The resource that contains the functions to send and receive tokens.
    ///
    access(all) resource interface Vault: Receiver, Provider, Balance {

        /// Field that tracks the balance of a vault
        access(all) var balance: UFix64

        /// Called when a fungible token is burned via the 'Burner.burn()' method
        /// Implementations can do any bookkeeping or emit any events
        /// that should be emitted when a vault is destroyed.
        /// Many implementations will want to update the token's total supply
        /// to reflect that the tokens have been burned and removed from the supply.
        /// Implementations also need to set the balance to zero before the end of the function
        /// This is to prevent vault owners from spamming fake Burned events.
        access(contract) fun burnCallback() {
            pre {
                emit Burned(type: self.getType().identifier, amount: self.balance, fromUUID: self.uuid)
            }
            post {
                self.balance == 0.0: "The balance must be set to zero during the burnCallback method so that it cannot be spammed"
            }
            self.balance = 0.0
        }

        /// getSupportedVaultTypes returns a dictionary of vault types and whether this receiver accepts the indexed type
        /// The default implementation is included here because vaults are expected
        /// to only accepted their own type, so they have no need to provide an implementation
        /// for this function
        access(all) view fun getSupportedVaultTypes(): {Type: Bool} {
            // Below check is implemented to make sure that run-time type would
            // only get returned when the parent resource conforms with 'FungibleToken.Vault'.
            if self.getType().isSubtype(of: Type<@{FungibleToken.Vault}>()) {
                return {self.getType(): true}
            } else {
                // Return an empty dictionary as the default value for resource who don't
                // implement 'FungibleToken.Vault', such as 'FungibleTokenSwitchboard', 'TokenForwarder' etc.
                return {}
            }
        }

        /// Checks if the given type is supported by this Vault
        access(all) view fun isSupportedVaultType(type: Type): Bool {
            return self.getSupportedVaultTypes()[type] ?? false
        }

        /// withdraw subtracts 'amount' from the Vault's balance
        /// and returns a new Vault with the subtracted balance
        ///
        access(Withdraw) fun withdraw(amount: UFix64): @{Vault} {
            pre {
                self.balance >= amount:
                    "Amount withdrawn must be less than or equal than the balance of the Vault"
            }
            post {
                result.getType() == self.getType(): "Must return the same vault type as self"
                // use the special function 'before' to get the value of the 'balance' field
                // at the beginning of the function execution
                //
                self.balance == before(self.balance) - amount:
                    "New Vault balance must be the difference of the previous balance and the withdrawn Vault balance"
                emit Withdrawn(
                        type: result.getType().identifier,
                        amount: amount,
                        from: self.owner?.address,
                        fromUUID: self.uuid,
                        withdrawnUUID: result.uuid,
                        balanceAfter: self.balance
                )
            }
        }

        /// deposit takes a Vault and adds its balance to the balance of this Vault
        ///
        access(all) fun deposit(from: @{FungibleToken.Vault}) {
            // Assert that the concrete type of the deposited vault is the same
            // as the vault that is accepting the deposit
            pre {
                from.isInstance(self.getType()):
                    "Cannot deposit an incompatible token type"
            }
            post {
                emit Deposited(
                        type: before(from.getType().identifier),
                        amount: before(from.balance),
                        to: self.owner?.address,
                        toUUID: self.uuid,
                        depositedUUID: before(from.uuid),
                        balanceAfter: self.balance
                )
                self.balance == before(self.balance) + before(from.balance):
                    "New Vault balance must be the sum of the previous balance and the deposited Vault"
            }
        }

        /// createEmptyVault allows any user to create a new Vault that has a zero balance
        ///
        access(all) fun createEmptyVault(): @{Vault} {
            post {
                result.balance == 0.0: "The newly created Vault must have zero balance"
                result.getType() == self.getType(): "The newly created Vault must have the same type as the creating vault"
            }
        }
    }

    /// createEmptyVault allows any user to create a new Vault that has a zero balance
    ///
    access(all) fun createEmptyVault(vaultType: Type): @{FungibleToken.Vault} {
        post {
            result.getType() == vaultType: "The returned vault does not match the desired type"
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

    // Event that is emitted when tokens are withdrawn from a Vault
    access(all) event TokensWithdrawn(amount: UFix64, from: Address?)

    // Event that is emitted when tokens are deposited to a Vault
    access(all) event TokensDeposited(amount: UFix64, to: Address?)

    // Event that is emitted when new tokens are minted
    access(all) event TokensMinted(amount: UFix64)

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

        /// Called when a fungible token is burned via the 'Burner.burn()' method
        access(contract) fun burnCallback() {
            if self.balance > 0.0 {
                FlowToken.totalSupply = FlowToken.totalSupply - self.balance
            }
            self.balance = 0.0
        }

        /// getSupportedVaultTypes optionally returns a list of vault types that this receiver accepts
        access(all) view fun getSupportedVaultTypes(): {Type: Bool} {
            return {self.getType(): true}
        }

        access(all) view fun isSupportedVaultType(type: Type): Bool {
            if (type == self.getType()) { return true } else { return false }
        }

        /// Asks if the amount can be withdrawn from this vault
        access(all) view fun isAvailableToWithdraw(amount: UFix64): Bool {
            return amount <= self.balance
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
        access(FungibleToken.Withdraw) fun withdraw(amount: UFix64): @{FungibleToken.Vault} {
            self.balance = self.balance - amount

            // If the owner is the staking account, do not emit the contract defined events
            // this is to help with the performance of the epoch transition operations
            // Either way, event listeners should be paying attention to the
            // FungibleToken.Withdrawn events anyway because those contain
            // much more comprehensive metadata
            // Additionally, these events will eventually be removed from this contract completely
            // in favor of the FungibleToken events
            if let address = self.owner?.address {
                if address != 0xf8d6e0586b0a20c7 &&
                   address != 0xf4527793ee68aede &&
                   address != 0x9eca2b38b18b5dfe &&
                   address != 0x8624b52f9ddcd04a
                {
                    emit TokensWithdrawn(amount: amount, from: address)
                }
            } else {
                emit TokensWithdrawn(amount: amount, from: nil)
            }
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

            // If the owner is the staking account, do not emit the contract defined events
            // this is to help with the performance of the epoch transition operations
            // Either way, event listeners should be paying attention to the
            // FungibleToken.Deposited events anyway because those contain
            // much more comprehensive metadata
            // Additionally, these events will eventually be removed from this contract completely
            // in favor of the FungibleToken events
            if let address = self.owner?.address {
                if address != 0xf8d6e0586b0a20c7 &&
                   address != 0xf4527793ee68aede &&
                   address != 0x9eca2b38b18b5dfe &&
                   address != 0x8624b52f9ddcd04a
                {
                    emit TokensDeposited(amount: vault.balance, to: address)
                }
            } else {
                emit TokensDeposited(amount: vault.balance, to: nil)
            }
            vault.balance = 0.0
            destroy vault
        }

        access(all) fun createEmptyVault(): @{FungibleToken.Vault} {
            return <-create Vault(balance: 0.0)
        }
    }

    // createEmptyVault
    //
    // Function that creates a new Vault with a balance of zero
    // and returns it to the calling context. A user must call this function
    // and store the returned Vault in their storage in order to allow their
    // account to be able to receive deposits of this token type.
    //
    access(all) fun createEmptyVault(vaultType: Type): @FlowToken.Vault {
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

    /// Gets the Flow Logo XML URI from storage
    access(all) view fun getLogoURI(): String {
        return FlowToken.account.storage.copy<String>(from: /storage/flowTokenLogoURI) ?? ""
    }

    init() {
        self.totalSupply = 0.0

        // Create the Vault with the total supply of tokens and save it in storage
        //
        let vault <- create Vault(balance: self.totalSupply)

        self.account.storage.save(<-vault, to: /storage/flowTokenVault)

        // Create a public capability to the stored Vault that only exposes
        // the 'deposit' method through the 'Receiver' interface
        //
        let receiverCapability = self.account.capabilities.storage.issue<&FlowToken.Vault>(/storage/flowTokenVault)
        self.account.capabilities.publish(receiverCapability, at: /public/flowTokenReceiver)

        // Create a public capability to the stored Vault that only exposes
        // the 'balance' field through the 'Balance' interface
        //
        let balanceCapability = self.account.capabilities.storage.issue<&FlowToken.Vault>(/storage/flowTokenVault)
        self.account.capabilities.publish(balanceCapability, at: /public/flowTokenBalance)

        let admin <- create Administrator()
        self.account.storage.save(<-admin, to: /storage/flowTokenAdmin)
    }
}
`

const realSetupFlowTokenAccountTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction {

    prepare(signer: auth(Storage, Capabilities) &Account) {

        if signer.storage.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault) == nil {
            // Create a new flowToken Vault and put it in storage
            var vault <- FlowToken.createEmptyVault(vaultType: Type<@FlowToken.Vault>())
            signer.storage.save(<- vault, to: /storage/flowTokenVault)

            // Create a public capability to the Vault that only exposes
            // the deposit function through the Receiver interface
            let vaultCap = signer.capabilities.storage.issue<&FlowToken.Vault>(
                /storage/flowTokenVault
            )

            signer.capabilities.publish(
                vaultCap,
                at: /public/flowTokenReceiver
            )

            // Create a public capability to the Vault that only exposes
            // the balance field through the Balance interface
            let balanceCap = signer.capabilities.storage.issue<&FlowToken.Vault>(
                /storage/flowTokenVault
            )

            signer.capabilities.publish(
                balanceCap,
                at: /public/flowTokenBalance
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

    prepare(signer: auth(BorrowValue) &Account) {

        self.tokenAdmin = signer.storage
            .borrow<&FlowToken.Administrator>(from: /storage/flowTokenAdmin)
            ?? panic("Signer is not the token admin")

        self.tokenReceiver = getAccount(recipient)
            .capabilities.borrow<&{FungibleToken.Receiver}>(/public/flowTokenReceiver)
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

    prepare(signer: auth(BorrowValue) &Account) {

        // Get a reference to the signer's stored vault
        let vaultRef = signer.storage.borrow<auth(FungibleToken.Withdraw) &FlowToken.Vault>(from: /storage/flowTokenVault)
			?? panic("Could not borrow reference to the owner's Vault!")

        // Withdraw tokens from the signer's stored vault
        self.sentVault <- vaultRef.withdraw(amount: amount)
    }

    execute {

        // Get a reference to the recipient's Receiver
        let receiverRef =  getAccount(to)
            .capabilities.borrow<&{FungibleToken.Receiver}>(/public/flowTokenReceiver)
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
        .capabilities.borrow<&FlowToken.Vault>(/public/flowTokenBalance)
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
			Source: utils.DeploymentTransaction("FlowToken", []byte(modifiedFlowContract)),
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
			Arguments: encodeArgs([]cadence.Value{
				cadence.Address(senderAddress),
				mintAmount,
			}),
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
				Arguments: encodeArgs([]cadence.Value{
					sendAmount,
					cadence.Address(receiverAddress),
				}),
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

	nextScriptLocation := NewScriptLocationGenerator()

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {

		result, err := runtime.ExecuteScript(
			Script{
				Source: []byte(realFlowTokenBalanceScript),
				Arguments: encodeArgs([]cadence.Value{
					cadence.Address(address),
				}),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextScriptLocation(),
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

// TODO:
//const oldExampleToken = `
//import FungibleToken from 0x1
//
//pub contract ExampleToken: FungibleToken {
//
//    pub var totalSupply: UFix64
//
//    pub resource Vault: FungibleToken.Provider, FungibleToken.Receiver, FungibleToken.Balance {
//
//        pub var balance: UFix64
//
//        init(balance: UFix64) {
//            self.balance = balance
//        }
//
//        pub fun withdraw(amount: UFix64): @FungibleToken.Vault {
//            self.balance = self.balance - amount
//            emit TokensWithdrawn(amount: amount, from: self.owner?.address)
//            return <-create Vault(balance: amount)
//        }
//
//        pub fun deposit(from: @FungibleToken.Vault) {
//            let vault <- from as! @ExampleToken.Vault
//            self.balance = self.balance + vault.balance
//            emit TokensDeposited(amount: vault.balance, to: self.owner?.address)
//            vault.balance = 0.0
//            destroy vault
//        }
//
//        destroy() {
//            if self.balance > 0.0 {
//                ExampleToken.totalSupply = ExampleToken.totalSupply - self.balance
//            }
//        }
//    }
//
//    pub fun createEmptyVault(): @Vault {
//        return <-create Vault(balance: 0.0)
//    }
//
//    init() {
//        self.totalSupply = 0.0
//    }
//}
//`

const oldExampleToken = `
import FungibleToken from 0x1

access(all)
contract ExampleToken {

    access(all)
    var totalSupply: UFix64

    access(all)
    resource Vault {

        access(all)
        var balance: UFix64

        init(balance: UFix64) {
            self.balance = balance
        }
    }

    init() {
        self.totalSupply = 4321.0
    }
}
`

func TestRuntimeBrokenFungibleTokenRecovery(t *testing.T) {

	runtime := NewTestInterpreterRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	userAddress := common.MustBytesToAddress([]byte{0x2})

	const contractName = "ExampleToken"

	accountCodes := map[Location][]byte{
		// We cannot deploy the ExampleToken contract because it is broken
		common.NewAddressLocation(nil, contractsAddress, contractName): []byte(oldExampleToken),
	}

	var events []cadence.Event
	var logs []string

	signerAccount := contractsAddress

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
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
		OnProgramLog: func(message string) {
			logs = append(logs, message)
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
	require.NoError(t, err)

	// We cannot deploy the ExampleToken contract because it is broken.
	// Manually storage the contract value for the ExampleToken contract in the contract account's storage

	storage, inter, err := runtime.Storage(Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	contractValue := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		common.NewAddressLocation(nil, contractsAddress, contractName),
		contractName,
		common.CompositeKindContract,
		[]interpreter.CompositeField{
			{
				Name: "totalSupply",
				Value: interpreter.NewUnmeteredUFix64ValueWithInteger(
					4321,
					interpreter.EmptyLocationRange,
				),
			},
		},
		contractsAddress,
	)

	contractStorage := storage.GetStorageMap(
		contractsAddress,
		StorageDomainContract,
		true,
	)
	contractStorage.SetValue(
		inter,
		interpreter.StringStorageMapKey(contractName),
		contractValue,
	)

	// Manually store a broken ExampleToken.Vault in the user account's storage

	vaultValue := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		common.NewAddressLocation(nil, contractsAddress, contractName),
		fmt.Sprintf("%s.Vault", contractName),
		common.CompositeKindResource,
		[]interpreter.CompositeField{
			{
				Name:  sema.ResourceUUIDFieldName,
				Value: interpreter.NewUnmeteredUInt64Value(42),
			},
			{
				Name: "balance",
				Value: interpreter.NewUnmeteredUFix64ValueWithInteger(
					1234,
					interpreter.EmptyLocationRange,
				),
			},
		},
		userAddress,
	)

	userStorage := storage.GetStorageMap(
		userAddress,
		common.PathDomainStorage.Identifier(),
		true,
	)
	const storagePathIdentifier = "exampleTokenVault"
	userStorage.SetValue(
		inter,
		interpreter.StringStorageMapKey(storagePathIdentifier),
		vaultValue,
	)

	err = storage.Commit(inter, false)
	require.NoError(t, err)

	// Send a transaction that loads the broken ExampleToken contract and the broken ExampleToken.Vault

	const transaction = `
      import FungibleToken from 0x1
      import ExampleToken from 0x1

      transaction {

          let vault: @ExampleToken.Vault

          prepare(signer: auth(LoadValue) &Account) {
              self.vault <- signer.storage.load<@ExampleToken.Vault>(from: /storage/exampleTokenVault)!
          }

          execute {
              log(ExampleToken.totalSupply)
              log(self.vault.balance)

              destroy self.vault
          }
      }
    `

	signerAccount = userAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(transaction),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(t, err)

	require.Equal(t,
		[]string{"4321.00000000", "1234.00000000"},
		logs,
	)
}
