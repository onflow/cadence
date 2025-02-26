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

const realFungibleTokenContractInterface = `
/// FungibleToken
///
/// Fungible Token implementations should implement the fungible token
/// interface.
access(all) contract interface FungibleToken {

    // An entitlement for allowing the withdrawal of tokens from a Vault
    access(all) entitlement Withdraw

    /// The event that is emitted when tokens are withdrawn
    /// from any Vault that implements the 'Vault' interface
    access(all) event Withdrawn(type: String,
                                amount: UFix64,
                                from: Address?,
                                fromUUID: UInt64,
                                withdrawnUUID: UInt64,
                                balanceAfter: UFix64)

    /// The event that is emitted when tokens are deposited to
    /// any Vault that implements the 'Vault' interface
    access(all) event Deposited(type: String,
                                amount: UFix64,
                                to: Address?,
                                toUUID: UInt64,
                                depositedUUID: UInt64,
                                balanceAfter: UFix64)

    /// Balance
    ///
    /// The interface that provides a standard field
    /// for representing balance
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
        /// @param amount the amount of tokens requested to potentially withdraw
        /// @return Bool Whether or not this amount is available to withdraw
        ///
        access(all) view fun isAvailableToWithdraw(amount: UFix64): Bool

        /// withdraw subtracts tokens from the implementing resource
        /// and returns a Vault with the removed tokens.
        ///
        /// The function's access level is 'access(Withdraw)'
        /// So in order to access it, one would either need the object itself
        /// or an entitled reference with 'Withdraw'.
        ///
        /// @param amount the amount of tokens to withdraw from the resource
        /// @return The Vault with the withdrawn tokens
        ///
        access(all) fun withdraw(amount: UFix64): @{Vault} {
            post {
                // 'result' refers to the return value
                result.balance == amount:
                    "FungibleToken.Provider.withdraw: Cannot withdraw tokens!"
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
        /// @param from the Vault that contains the tokens to deposit
        ///
        access(all) fun deposit(from: @{Vault})
    }

    /// Vault
    /// Conforms to all other interfaces so that implementations
    /// only have to conform to 'Vault'
    ///
    access(all) resource interface Vault: Receiver, Provider, Balance {

        /// Field that tracks the balance of a vault
        access(all) var balance: UFix64

        /// withdraw subtracts 'amount' from the Vault's balance
        /// and returns a new Vault with the subtracted balance
        ///
        access(all) fun withdraw(amount: UFix64): @{Vault} {
            pre {
                self.balance >= amount:
                    "FungibleToken.Vault.withdraw: Cannot withdraw tokens! "
            }
            post {
                // use the special function 'before' to get the value of the 'balance' field
                // at the beginning of the function execution
                //
                self.balance == before(self.balance) - amount:
                    "FungibleToken.Vault.withdraw: Cannot withdraw tokens! "
            }
        }

        /// deposit takes a Vault and adds its balance to the balance of this Vault
        ///
        access(all) fun deposit(from: @{FungibleToken.Vault}) {
            post {
                self.balance == before(self.balance) + before(from.balance):
                    "FungibleToken.Vault.deposit: Cannot deposit tokens! "
            }
        }

        /// createEmptyVault allows any user to create a new Vault that has a zero balance
        ///
        /// @return A Vault of the same type that has a balance of zero
        access(all) fun createEmptyVault(): @{Vault} {
            post {
                result.balance == 0.0:
                    "FungibleToken.Vault.createEmptyVault: Empty Vault creation failed! "
            }
        }
    }

    /// createEmptyVault allows any user to create a new Vault that has a zero balance
    ///
    /// @return A Vault of the requested type that has a balance of zero
    access(all) fun createEmptyVault(): @{FungibleToken.Vault} {
        post {
            result.balance == 0.0:
                "FungibleToken.createEmptyVault: Empty Vault creation failed! "
        }
    }
}
`

const realFlowContract = `
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
        access(all) fun withdraw(amount: UFix64): @{FungibleToken.Vault} {
            self.balance = self.balance - amount

			emit TokensWithdrawn(amount: amount, from: nil)

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

			emit TokensDeposited(amount: vault.balance, to: nil)

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
    access(all) fun createEmptyVault(): @FlowToken.Vault {
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
                amount > 0.0: "Amount minted must be greater than zero"
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

    init(adminAccount: auth(Storage, Capabilities) &Account) {
        self.totalSupply = 0.0

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

    prepare(signer: auth(Capabilities, Storage) &Account) {

        if signer.storage.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault) == nil {
            // Create a new flowToken Vault and put it in storage
            signer.storage.save(<-FlowToken.createEmptyVault(), to: /storage/flowTokenVault)

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

const realFlowTokenTransferTransaction = `
import FungibleToken from 0x1
import FlowToken from 0x1

transaction(amount: UFix64, to: Address) {

    // The Vault resource that holds the tokens that are being transferred
    let sentVault: @{FungibleToken.Vault}

    prepare(signer: auth(BorrowValue) &Account) {

        // Get a reference to the signer's stored vault
        let vaultRef = signer.storage.borrow<auth(FungibleToken.Withdraw) &FlowToken.Vault>(from: /storage/flowTokenVault)
            ?? panic("The signer does not store a FlowToken Vault object at the path ")

        // Withdraw tokens from the signer's stored vault
        self.sentVault <- vaultRef.withdraw(amount: amount)
    }

    execute {

        // Get a reference to the recipient's Receiver
        let receiverRef =  getAccount(to)
            .capabilities.get<&{FungibleToken.Receiver}>(/public/flowTokenReceiver).borrow()
            ?? panic("Could not borrow a Receiver reference to the FlowToken Vault in account ")

        // Deposit the withdrawn tokens in the recipient's receiver
        receiverRef.deposit(from: <-self.sentVault)
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
            ?? panic("Cannot mint: Signer does not store the FlowToken Admin Resource in their account")

        self.tokenReceiver = getAccount(recipient)
            .capabilities.get<&{FungibleToken.Receiver}>(/public/flowTokenReceiver).borrow()
            ?? panic("Could not borrow a Receiver reference to the FlowToken Vault in account ")
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

access(all) fun main(account: Address): UFix64 {

    let vaultRef = getAccount(account)
        .capabilities.get<&FlowToken.Vault>(/public/flowTokenBalance).borrow()
        ?? panic("Could not borrow a balance reference to the FlowToken Vault in account ")

    return vaultRef.balance
}
`
