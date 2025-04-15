/**

# The Flow Fungible Token standard

## `FungibleToken` contract

If a users wants to deploy a new token contract, their contract
needs to implement the FungibleToken interface and their tokens
need to implement the interfaces defined in this contract.

/// Contributors (please add to this list if you contribute!):
/// - Joshua Hannan - https://github.com/joshuahannan
/// - Bastian Müller - https://twitter.com/turbolent
/// - Dete Shirley - https://twitter.com/dete73
/// - Bjarte Karlsen - https://twitter.com/0xBjartek
/// - Austin Kline - https://twitter.com/austin_flowty
/// - Giovanni Sanchez - https://twitter.com/gio_incognito
/// - Deniz Edincik - https://twitter.com/bluesign
/// - Jonny - https://github.com/dryruner
///
/// Repo reference: https://github.com/onflow/flow-ft

## `Vault` resource interface

Each fungible token resource type needs to implement the `Vault` resource interface.

## `Provider`, `Receiver`, and `Balance` resource interfaces

These interfaces declare pre-conditions and post-conditions that restrict
the execution of the functions in the Vault.

It gives users the ability to make custom resources that implement
these interfaces to do various things with the tokens.
For example, a faucet can be implemented by conforming
to the Provider interface.

*/

import "ViewResolver"
import "Burner"

/// FungibleToken
///
/// Fungible Token implementations should implement the fungible token
/// interface.
access(all) contract interface FungibleToken: ViewResolver {

    // An entitlement for allowing the withdrawal of tokens from a Vault
    access(all) entitlement Withdraw

    /// The event that is emitted when tokens are withdrawn
    /// from any Vault that implements the `Vault` interface
    access(all) event Withdrawn(type: String,
                                amount: UFix64,
                                from: Address?,
                                fromUUID: UInt64,
                                withdrawnUUID: UInt64,
                                balanceAfter: UFix64)

    /// The event that is emitted when tokens are deposited to
    /// any Vault that implements the `Vault` interface
    access(all) event Deposited(type: String,
                                amount: UFix64,
                                to: Address?,
                                toUUID: UInt64,
                                depositedUUID: UInt64,
                                balanceAfter: UFix64)

    /// Event that is emitted when the global `Burner.burn()` method
    /// is called with a non-zero balance
    access(all) event Burned(type: String, amount: UFix64, fromUUID: UInt64)

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
    /// It does not enforce requirements on `balance` here,
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
        /// The function's access level is `access(Withdraw)`
        /// So in order to access it, one would either need the object itself
        /// or an entitled reference with `Withdraw`.
        ///
        /// @param amount the amount of tokens to withdraw from the resource
        /// @return The Vault with the withdrawn tokens
        ///
        access(Withdraw) fun withdraw(amount: UFix64): @{Vault} {
            post {
                // `result` refers to the return value
                result.balance == amount:
                    "FungibleToken.Provider.withdraw: Cannot withdraw tokens!"
                    .concat("The balance of the withdrawn tokens (").concat(result.balance.toString())
                    .concat(") is not equal to the amount requested to be withdrawn (")
                    .concat(amount.toString()).concat(")")
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

        /// getSupportedVaultTypes returns a dictionary of Vault types
        /// and whether the type is currently supported by this Receiver
        ///
        /// @return {Type: Bool} A dictionary that indicates the supported types
        ///                      If a type is not supported, it should be `nil`, not false
        ///
        access(all) view fun getSupportedVaultTypes(): {Type: Bool}

        /// Returns whether or not the given type is accepted by the Receiver
        /// A vault that can accept any type should just return true by default
        ///
        /// @param type The type to query about
        /// @return Bool Whether or not the vault type is supported
        ///
        access(all) view fun isSupportedVaultType(type: Type): Bool
    }

    /// Vault
    /// Conforms to all other interfaces so that implementations
    /// only have to conform to `Vault`
    ///
    access(all) resource interface Vault: Receiver, Provider, Balance, ViewResolver.Resolver, Burner.Burnable {

        /// Field that tracks the balance of a vault
        access(all) var balance: UFix64

        /// Called when a fungible token is burned via the `Burner.burn()` method
        /// Implementations can do any bookkeeping or emit any events
        /// that should be emitted when a vault is destroyed.
        /// Many implementations will want to update the token's total supply
        /// to reflect that the tokens have been burned and removed from the supply.
        /// Implementations also need to set the balance to zero before the end of the function
        /// This is to prevent vault owners from spamming fake Burned events.
        access(contract) fun burnCallback() {
            pre {
                // TODO: getType
                // emit Burned(type: self.getType().identifier, amount: self.balance, fromUUID: self.uuid)
            }
            post {
                self.balance == 0.0:
                    "FungibleToken.Vault.burnCallback: Cannot burn this Vault with Burner.burn(). "
                    .concat("The balance must be set to zero during the burnCallback method so that it cannot be spammed.")
            }
            self.balance = 0.0
        }

        /// getSupportedVaultTypes
        /// The default implementation is included here because vaults are expected
        /// to only accepted their own type, so they have no need to provide an implementation
        /// for this function
        ///
        access(all) view fun getSupportedVaultTypes(): {Type: Bool} {
            // Below check is implemented to make sure that run-time type would
            // only get returned when the parent resource conforms with `FungibleToken.Vault`.
            if self.getType().isSubtype(of: Type<@{FungibleToken.Vault}>()) {
                return {self.getType(): true}
            } else {
                // Return an empty dictionary as the default value for resource who don't
                // implement `FungibleToken.Vault`, such as `FungibleTokenSwitchboard`, `TokenForwarder` etc.
                return {}
            }
        }

        /// Checks if the given type is supported by this Vault
        access(all) view fun isSupportedVaultType(type: Type): Bool {
            return self.getSupportedVaultTypes()[type] ?? false
        }

        /// withdraw subtracts `amount` from the Vault's balance
        /// and returns a new Vault with the subtracted balance
        ///
        access(Withdraw) fun withdraw(amount: UFix64): @{Vault} {
            pre {
                self.balance >= amount:
                    "FungibleToken.Vault.withdraw: Cannot withdraw tokens! "
                    .concat("The amount requested to be withdrawn (").concat(amount.toString())
                    .concat(") is greater than the balance of the Vault (")
                    .concat(self.balance.toString()).concat(").")
            }
            post {
                // TODO: getType
                // result.getType() == self.getType():
                //     "FungibleToken.Vault.withdraw: Cannot withdraw tokens! "
                //     .concat("The withdraw method tried to return an incompatible Vault type <")
                //     .concat(result.getType().identifier).concat(">. ")
                //     .concat("It must return a Vault with the same type as self <")
                //     .concat(self.getType().identifier).concat(">.")

                // use the special function `before` to get the value of the `balance` field
                // at the beginning of the function execution
                //
                self.balance == before(self.balance) - amount:
                    "FungibleToken.Vault.withdraw: Cannot withdraw tokens! "
                    .concat("The sender's balance after the withdrawal (")
                    .concat(self.balance.toString())
                    .concat(") must be the difference of the previous balance (").concat(before(self.balance.toString()))
                    .concat(") and the amount withdrawn (").concat(amount.toString()).concat(")")

                // TODO: getType
                // emit Withdrawn(
                //         type: result.getType().identifier,
                //         amount: amount,
                //         from: self.owner?.address,
                //         fromUUID: self.uuid,
                //         withdrawnUUID: result.uuid,
                //         balanceAfter: self.balance
                // )
            }
        }

        /// deposit takes a Vault and adds its balance to the balance of this Vault
        ///
        access(all) fun deposit(from: @{FungibleToken.Vault}) {
            // Assert that the concrete type of the deposited vault is the same
            // as the vault that is accepting the deposit
            // TODO: getType
            // pre {
            //     from.isInstance(self.getType()):
            //         "FungibleToken.Vault.deposit: Cannot deposit tokens! "
            //         .concat("The type of the deposited tokens <")
            //         .concat(from.getType().identifier)
            //         .concat("> has to be the same type as the Vault being deposited into <")
            //         .concat(self.getType().identifier)
            //         .concat(">. Check that you are withdrawing and depositing to the correct paths in the sender and receiver accounts ")
            //         .concat("and that those paths hold the same Vault types.")
            // }
            post {
                // TODO: getType
                // emit Deposited(
                //         type: before(from.getType().identifier),
                //         amount: before(from.balance),
                //         to: self.owner?.address,
                //         toUUID: self.uuid,
                //         depositedUUID: before(from.uuid),
                //         balanceAfter: self.balance
                // )
                self.balance == before(self.balance) + before(from.balance):
                    "FungibleToken.Vault.deposit: Cannot deposit tokens! "
                    .concat("The receiver's balance after the deposit (")
                    .concat(self.balance.toString())
                    .concat(") must be the sum of the previous balance (").concat(before(self.balance.toString()))
                    .concat(") and the amount deposited (").concat(before(from.balance).toString()).concat(")")
            }
        }

        /// createEmptyVault allows any user to create a new Vault that has a zero balance
        ///
        /// @return A Vault of the same type that has a balance of zero
        access(all) fun createEmptyVault(): @{Vault} {
            post {
                result.balance == 0.0:
                    "FungibleToken.Vault.createEmptyVault: Empty Vault creation failed! "
                    .concat("The newly created Vault must have zero balance but it has a balance of ")
                    .concat(result.balance.toString())

                result.getType() == self.getType():
                    "FungibleToken.Vault.createEmptyVault: Empty Vault creation failed! "
                    .concat("The type of the new Vault <")
                    .concat(result.getType().identifier)
                    .concat("> has to be the same type as the Vault that created it <")
                    .concat(self.getType().identifier)
                    .concat(">.")
            }
        }
    }

    /// createEmptyVault allows any user to create a new Vault that has a zero balance
    ///
    /// @return A Vault of the requested type that has a balance of zero
    access(all) fun createEmptyVault(vaultType: Type): @{FungibleToken.Vault} {
        post {
            result.balance == 0.0:
                "FungibleToken.createEmptyVault: Empty Vault creation failed! "
                .concat("The newly created Vault must have zero balance but it has a balance of (")
                .concat(result.balance.toString()).concat(")")

            // TODO: getType
            // result.getType() == vaultType:
            //     "FungibleToken.Vault.createEmptyVault: Empty Vault creation failed! "
            //     .concat("The type of the new Vault <")
            //     .concat(result.getType().identifier)
            //     .concat("> has to be the same as the type that was requested <")
            //     .concat(vaultType.identifier)
            //     .concat(">.")
        }
    }
}