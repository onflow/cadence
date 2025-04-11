import "FungibleToken"
import "MetadataViews"
import "FungibleTokenMetadataViews"

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

        /// Called when a fungible token is burned via the `Burner.burn()` method
        access(contract) fun burnCallback() {
            if self.balance > 0.0 {
                FlowToken.totalSupply = FlowToken.totalSupply - self.balance
            }
            self.balance = 0.0
        }

        /// getSupportedVaultTypes optionally returns a list of vault types that this receiver accepts
        access(all) view fun getSupportedVaultTypes(): {Type: Bool} {
            // TODO: getType
            // return {self.getType(): true}
            return {}
        }

        access(all) view fun isSupportedVaultType(type: Type): Bool {
            // TODO: getType
            // if (type == self.getType()) { return true } else { return false }
            return true
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
            // TODO: address constants
            // if let address = self.owner?.address {
            //     if address != 0xf8d6e0586b0a20c7 &&
            //        address != 0xf4527793ee68aede &&
            //        address != 0x9eca2b38b18b5dfe &&
            //        address != 0x8624b52f9ddcd04a
            //     {
            //         emit TokensWithdrawn(amount: amount, from: address)
            //     }
            // } else {
            //     emit TokensWithdrawn(amount: amount, from: nil)
            // }
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
            // TODO: address constants
            // if let address = self.owner?.address {
            //     if address != 0xf8d6e0586b0a20c7 &&
            //        address != 0xf4527793ee68aede &&
            //        address != 0x9eca2b38b18b5dfe &&
            //        address != 0x8624b52f9ddcd04a
            //     {
            //         emit TokensDeposited(amount: vault.balance, to: address)
            //     }
            // } else {
            //     emit TokensDeposited(amount: vault.balance, to: nil)
            // }
            vault.balance = 0.0
            destroy vault
        }

        /// Get all the Metadata Views implemented by FlowToken
        ///
        /// @return An array of Types defining the implemented views. This value will be used by
        ///         developers to know which parameter to pass to the resolveView() method.
        ///
        access(all) view fun getViews(): [Type]{
            return FlowToken.getContractViews(resourceType: nil)
        }

        /// Get a Metadata View from FlowToken
        ///
        /// @param view: The Type of the desired view.
        /// @return A structure representing the requested view.
        ///
        access(all) fun resolveView(_ view: Type): AnyStruct? {
            return FlowToken.resolveContractView(resourceType: nil, viewType: view)
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

    /// Gets a list of the metadata views that this contract supports
    access(all) view fun getContractViews(resourceType: Type?): [Type] {
        return [Type<FungibleTokenMetadataViews.FTView>(),
                Type<FungibleTokenMetadataViews.FTDisplay>(),
                Type<FungibleTokenMetadataViews.FTVaultData>(),
                Type<FungibleTokenMetadataViews.TotalSupply>()]
    }

    /// Get a Metadata View from FlowToken
    ///
    /// @param view: The Type of the desired view.
    /// @return A structure representing the requested view.
    ///
    access(all) fun resolveContractView(resourceType: Type?, viewType: Type): AnyStruct? {
        switch viewType {
            case Type<FungibleTokenMetadataViews.FTView>():
                return FungibleTokenMetadataViews.FTView(
                    ftDisplay: self.resolveContractView(resourceType: nil, viewType: Type<FungibleTokenMetadataViews.FTDisplay>()) as! FungibleTokenMetadataViews.FTDisplay?,
                    ftVaultData: self.resolveContractView(resourceType: nil, viewType: Type<FungibleTokenMetadataViews.FTVaultData>()) as! FungibleTokenMetadataViews.FTVaultData?
                )
            case Type<FungibleTokenMetadataViews.FTDisplay>():
                let media = MetadataViews.Media(
                        file: MetadataViews.HTTPFile(
                        url: FlowToken.getLogoURI()
                    ),
                    mediaType: "image/svg+xml"
                )
                let medias = MetadataViews.Medias([media])
                return FungibleTokenMetadataViews.FTDisplay(
                    name: "FLOW Network Token",
                    symbol: "FLOW",
                    description: "FLOW is the native token for the Flow blockchain. It is required for securing the network, transaction fees, storage fees, staking, FLIP voting and may be used by applications built on the Flow Blockchain",
                    externalURL: MetadataViews.ExternalURL("https://flow.com"),
                    logos: medias,
                    socials: {
                        "twitter": MetadataViews.ExternalURL("https://twitter.com/flow_blockchain")
                    }
                )
            case Type<FungibleTokenMetadataViews.FTVaultData>():
                let vaultRef = FlowToken.account.storage.borrow<auth(FungibleToken.Withdraw) &FlowToken.Vault>(from: /storage/flowTokenVault)
			        ?? panic("Could not borrow reference to the contract's Vault!")
                return FungibleTokenMetadataViews.FTVaultData(
                    storagePath: /storage/flowTokenVault,
                    receiverPath: /public/flowTokenReceiver,
                    metadataPath: /public/flowTokenBalance,
                    receiverLinkedType: Type<&{FungibleToken.Receiver, FungibleToken.Vault}>(),
                    metadataLinkedType: Type<&{FungibleToken.Balance, FungibleToken.Vault}>(),
                    createEmptyVaultFunction: (fun (): @{FungibleToken.Vault} {
                        return <-vaultRef.createEmptyVault()
                    })
                )
            case Type<FungibleTokenMetadataViews.TotalSupply>():
                return FungibleTokenMetadataViews.TotalSupply(totalSupply: FlowToken.totalSupply)
        }
        return nil
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
        // the `deposit` method through the `Receiver` interface
        //
        let receiverCapability = self.account.capabilities.storage.issue<&FlowToken.Vault>(/storage/flowTokenVault)
        self.account.capabilities.publish(receiverCapability, at: /public/flowTokenReceiver)

        // Create a public capability to the stored Vault that only exposes
        // the `balance` field through the `Balance` interface
        //
        let balanceCapability = self.account.capabilities.storage.issue<&FlowToken.Vault>(/storage/flowTokenVault)
        self.account.capabilities.publish(balanceCapability, at: /public/flowTokenBalance)

        let admin <- create Administrator()
        self.account.storage.save(<-admin, to: /storage/flowTokenAdmin)

    }
}
