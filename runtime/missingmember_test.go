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

package runtime

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
)

func TestRuntimeMissingMemberFabricant(t *testing.T) {

	runtime := newTestInterpreterRuntime()

	testAddress, err := common.HexToAddress("0x1")
	require.NoError(t, err)

	contractsAddress, err := common.HexToAddress("0x5a76b4858ce34b2f")
	require.NoError(t, err)

	ftAddress, err := common.HexToAddress("0x9a0766d93b6608b7")
	require.NoError(t, err)

	flowTokenAddress, err := common.HexToAddress("0x7e60df042a9c0868")
	require.NoError(t, err)

	nftAddress, err := common.HexToAddress("0x631e88ae7f1d7c20")
	require.NoError(t, err)

	const flowTokenContract = `
import FungibleToken from 0x9a0766d93b6608b7

pub contract FlowToken: FungibleToken {

   // Total supply of Flow tokens in existence
   pub var totalSupply: UFix64

   // Event that is emitted when the contract is created
   pub event TokensInitialized(initialSupply: UFix64)

   // Event that is emitted when tokens are withdrawn from a Vault
   pub event TokensWithdrawn(amount: UFix64, from: Address?)

   // Event that is emitted when tokens are deposited to a Vault
   pub event TokensDeposited(amount: UFix64, to: Address?)

   // Event that is emitted when new tokens are minted
   pub event TokensMinted(amount: UFix64)

   // Event that is emitted when tokens are destroyed
   pub event TokensBurned(amount: UFix64)

   // Event that is emitted when a new minter resource is created
   pub event MinterCreated(allowedAmount: UFix64)

   // Event that is emitted when a new burner resource is created
   pub event BurnerCreated()

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
       pub var balance: UFix64

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
       pub fun withdraw(amount: UFix64): @FungibleToken.Vault {
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
       pub fun deposit(from: @FungibleToken.Vault) {
           let vault <- from as! @FlowToken.Vault
           self.balance = self.balance + vault.balance
           emit TokensDeposited(amount: vault.balance, to: self.owner?.address)
           vault.balance = 0.0
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
       return <-create Vault(balance: 0.0)
   }

   pub resource Administrator {
       // createNewMinter
       //
       // Function that creates and returns a new minter resource
       //
       pub fun createNewMinter(allowedAmount: UFix64): @Minter {
           emit MinterCreated(allowedAmount: allowedAmount)
           return <-create Minter(allowedAmount: allowedAmount)
       }

       // createNewBurner
       //
       // Function that creates and returns a new burner resource
       //
       pub fun createNewBurner(): @Burner {
           emit BurnerCreated()
           return <-create Burner()
       }
   }

   // Minter
   //
   // Resource object that token admin accounts can hold to mint new tokens.
   //
   pub resource Minter {

       // the amount of tokens that the minter is allowed to mint
       pub var allowedAmount: UFix64

       // mintTokens
       //
       // Function that mints new tokens, adds them to the total supply,
       // and returns them to the calling context.
       //
       pub fun mintTokens(amount: UFix64): @FlowToken.Vault {
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
           emit TokensBurned(amount: amount)
       }
   }

   init(adminAccount: AuthAccount) {
       self.totalSupply = 0.0

       // Create the Vault with the total supply of tokens and save it in storage
       //
       let vault <- create Vault(balance: self.totalSupply)
       adminAccount.save(<-vault, to: /storage/flowTokenVault)

       // Create a public capability to the stored Vault that only exposes
       // the deposit method through the Receiver interface
       //
       adminAccount.link<&FlowToken.Vault{FungibleToken.Receiver}>(
           /public/flowTokenReceiver,
           target: /storage/flowTokenVault
       )

       // Create a public capability to the stored Vault that only exposes
       // the balance field through the Balance interface
       //
       adminAccount.link<&FlowToken.Vault{FungibleToken.Balance}>(
           /public/flowTokenBalance,
           target: /storage/flowTokenVault
       )

       let admin <- create Administrator()
       adminAccount.save(<-admin, to: /storage/flowTokenAdmin)

       // Emit an event that shows that the contract was initialized
       emit TokensInitialized(initialSupply: self.totalSupply)
   }
}

`

	const fbrcContract = `
import FungibleToken from 0x9a0766d93b6608b7

pub contract FBRC: FungibleToken {

   // Total supply of Flow tokens in existence
   pub var totalSupply: UFix64

   // Event that is emitted when the contract is created
   pub event TokensInitialized(initialSupply: UFix64)

   // Event that is emitted when tokens are withdrawn from a Vault
   pub event TokensWithdrawn(amount: UFix64, from: Address?)

   // Event that is emitted when tokens are deposited to a Vault
   pub event TokensDeposited(amount: UFix64, to: Address?)

   // Event that is emitted when new tokens are minted
   pub event TokensMinted(amount: UFix64)

   // Event that is emitted when tokens are destroyed
   pub event TokensBurned(amount: UFix64)

   // Event that is emitted when a new minter resource is created
   pub event MinterCreated(allowedAmount: UFix64)

   // Event that is emitted when a new burner resource is created
   pub event BurnerCreated()

   // Contains standard storage and public paths of resources
   pub let CollectionStoragePath: StoragePath

   pub let CollectionReceiverPath: PublicPath

   pub let CollectionBalancePath: PublicPath

   pub let AdminStoragePath: StoragePath
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
       pub var balance: UFix64

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
       pub fun withdraw(amount: UFix64): @FungibleToken.Vault {
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
       pub fun deposit(from: @FungibleToken.Vault) {
           let vault <- from as! @FBRC.Vault
           self.balance = self.balance + vault.balance
           emit TokensDeposited(amount: vault.balance, to: self.owner?.address)
           vault.balance = 0.0
           destroy vault
       }

       destroy() {
           FBRC.totalSupply = FBRC.totalSupply - self.balance
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
       return <-create Vault(balance: 0.0)
   }

   pub resource Administrator {
       // createNewMinter
       //
       // Function that creates and returns a new minter resource
       //
       pub fun createNewMinter(allowedAmount: UFix64): @Minter {
           emit MinterCreated(allowedAmount: allowedAmount)
           return <-create Minter(allowedAmount: allowedAmount)
       }

       // createNewBurner
       //
       // Function that creates and returns a new burner resource
       //
       pub fun createNewBurner(): @Burner {
           emit BurnerCreated()
           return <-create Burner()
       }
   }

   // Minter
   //
   // Resource object that token admin accounts can hold to mint new tokens.
   //
   pub resource Minter {

       // the amount of tokens that the minter is allowed to mint
       pub var allowedAmount: UFix64

       // mintTokens
       //
       // Function that mints new tokens, adds them to the total supply,
       // and returns them to the calling context.
       //
       pub fun mintTokens(amount: UFix64): @FBRC.Vault {
           pre {
               amount > 0.0: "Amount minted must be greater than zero"
               amount <= self.allowedAmount: "Amount minted must be less than the allowed amount"
           }
           FBRC.totalSupply = FBRC.totalSupply + amount
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
   pub resource Burner {

       // burnTokens
       //
       // Function that destroys a Vault instance, effectively burning the tokens.
       //
       // Note: the burned tokens are automatically subtracted from the
       // total supply in the Vault destructor.
       //
       pub fun burnTokens(from: @FungibleToken.Vault) {
           let vault <- from as! @FBRC.Vault
           let amount = vault.balance
           destroy vault
           emit TokensBurned(amount: amount)
       }
   }

   init() {
       self.totalSupply = 0.0

       self.CollectionStoragePath = /storage/FbrcVault0007
       self.CollectionReceiverPath = /public/FbrcReceiver0007
       self.CollectionBalancePath = /public/FbrcBalance0007
       self.AdminStoragePath = /storage/FbrcAdmin0007

       // Create the Vault with the total supply of tokens and save it in storage
       //
       let vault <- create Vault(balance: self.totalSupply)
       self.account.save(<-vault, to: self.CollectionStoragePath)

       // Create a public capability to the stored Vault that only exposes
       // the deposit method through the Receiver interface
       //
       self.account.link<&FBRC.Vault{FungibleToken.Receiver}>(
           self.CollectionReceiverPath,
           target: self.CollectionStoragePath
       )

       // Create a public capability to the stored Vault that only exposes
       // the balance field through the Balance interface
       //
       self.account.link<&FBRC.Vault{FungibleToken.Balance}>(
           self.CollectionBalancePath,
           target: self.CollectionStoragePath
       )

       let admin <- create Administrator()
       self.account.save(<-admin, to: self.AdminStoragePath)

       // Emit an event that shows that the contract was initialized
       emit TokensInitialized(initialSupply: self.totalSupply)
   }
}
`

	const garmentContract = `
import NonFungibleToken from 0x631e88ae7f1d7c20
import FungibleToken from 0x9a0766d93b6608b7
import FBRC from 0x5a76b4858ce34b2f

pub contract GarmentNFT: NonFungibleToken {

   // -----------------------------------------------------------------------
   // GarmentNFT contract Events
   // -----------------------------------------------------------------------

   // Emitted when the Garment contract is created
   pub event ContractInitialized()

   // Emitted when a new GarmentData struct is created
   pub event GarmentDataCreated(garmentDataID: UInt32, mainImage: String, images: [String], name: String, artist: String, description: String)

   // Emitted when a Garment is minted
   pub event GarmentMinted(garmentID: UInt64, garmentDataID: UInt32, serialNumber: UInt32)

   // Emitted when the contract's royalty percentage is changed
   pub event RoyaltyPercentageChanged(newRoyaltyPercentage: UFix64)

   pub event GarmentDataIDRetired(garmentDataID: UInt32)

   // Events for Collection-related actions
   //
   // Emitted when a Garment is withdrawn from a Collection
   pub event Withdraw(id: UInt64, from: Address?)

   // Emitted when a Garment is deposited into a Collection
   pub event Deposit(id: UInt64, to: Address?)

   // Emitted when a Garment is destroyed
   pub event GarmentDestroyed(id: UInt64)

   // -----------------------------------------------------------------------
   // contract-level fields.
   // These contain actual values that are stored in the smart contract.
   // -----------------------------------------------------------------------

   // Contains standard storage and public paths of resources
   pub let CollectionStoragePath: StoragePath

   pub let CollectionPublicPath: PublicPath

   pub let AdminStoragePath: StoragePath

   // Variable size dictionary of Garment structs
   access(self) var garmentDatas: {UInt32: GarmentData}

   // Dictionary with GarmentDataID as key and number of NFTs with GarmentDataID are minted
   access(self) var numberMintedPerGarment: {UInt32: UInt32}

   // Dictionary of garmentDataID to  whether they are retired
   access(self) var isGarmentDataRetired: {UInt32: Bool}

   // Keeps track of how many unique GarmentData's are created
   pub var nextGarmentDataID: UInt32

   pub var royaltyPercentage: UFix64

   pub var totalSupply: UInt64

   pub struct GarmentData {

       // The unique ID for the Garment Data
       pub let garmentDataID: UInt32

       //stores link to image
       pub let mainImage: String
       //stores link to supporting images
       pub let images: [String]
       pub let name: String
       pub let artist: String
       //description of design
       pub let description: String

       init(
           mainImage: String,
           images: [String],
           name: String,
           artist: String,
           description: String,
       ){
           self.garmentDataID = GarmentNFT.nextGarmentDataID
           self.mainImage = mainImage
           self.images = images
           self.name = name
           self.artist = artist
           self.description = description

           GarmentNFT.isGarmentDataRetired[self.garmentDataID] = false

           // Increment the ID so that it isn't used again
           GarmentNFT.nextGarmentDataID = GarmentNFT.nextGarmentDataID + 1 as UInt32

           emit GarmentDataCreated(garmentDataID: self.garmentDataID, mainImage: self.mainImage, images: self.images, name: self.name, artist: self.artist, description: self.description)
       }
   }

   pub struct Garment {

       // The ID of the GarmentData that the Garment references
       pub let garmentDataID: UInt32

       // The N'th NFT with 'GarmentDataID' minted
       pub let serialNumber: UInt32

       init(garmentDataID: UInt32) {
           self.garmentDataID = garmentDataID

           // Increment the ID so that it isn't used again
           GarmentNFT.numberMintedPerGarment[garmentDataID] = GarmentNFT.numberMintedPerGarment[garmentDataID]! + 1 as UInt32

           self.serialNumber = GarmentNFT.numberMintedPerGarment[garmentDataID]!
       }
   }

   // The resource that represents the Garment NFTs
   //
   pub resource NFT: NonFungibleToken.INFT {

       // Global unique Garment ID
       pub let id: UInt64

       // struct of Garment
       pub let garment: Garment

       // Royalty capability which NFT will use
       pub let royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>

       init(serialNumber: UInt32, garmentDataID: UInt32, royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>) {
           GarmentNFT.totalSupply = GarmentNFT.totalSupply + 1 as UInt64

           self.id = GarmentNFT.totalSupply

           self.garment = Garment(garmentDataID: garmentDataID)

           self.royaltyVault = royaltyVault

           // Emitted when a Garment is minted
           emit GarmentMinted(garmentID: self.id, garmentDataID: garmentDataID, serialNumber: serialNumber)
       }

       destroy() {
           emit GarmentDestroyed(id: self.id)
       }

   }

   // Admin is a special authorization resource that
   // allows the owner to perform important functions to modify the
   // various aspects of the Garment and NFTs
   //
   pub resource Admin {

       pub fun createGarmentData(
           mainImage: String,
           images: [String],
           name: String,
           artist: String,
           description: String,
       ): UInt32 {
           // Create the new GarmentData
           var newGarment = GarmentData(
               mainImage: mainImage,
               images: images,
               name: name,
               artist: artist,
               description: description,
           )

           let newID = newGarment.garmentDataID

           // Store it in the contract storage
           GarmentNFT.garmentDatas[newID] = newGarment

           GarmentNFT.numberMintedPerGarment[newID] = 0 as UInt32

           return newID
       }

       // createNewAdmin creates a new Admin resource
       //
       pub fun createNewAdmin(): @Admin {
           return <-create Admin()
       }

       // Mint the new Garment
       pub fun mintNFT(garmentDataID: UInt32, royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>): @NFT {
           pre {
               royaltyVault.check():
                   "Royalty capability is invalid!"
           }

           if (GarmentNFT.isGarmentDataRetired[garmentDataID]! == nil) {
               panic("Cannot mint Garment. garmentData not found")
           }

           if (GarmentNFT.isGarmentDataRetired[garmentDataID]!) {
               panic("Cannot mint garment. garmentDataID retired")
           }

           let numInGarment = GarmentNFT.numberMintedPerGarment[garmentDataID]??
               panic("Cannot mint Garment. garmentData not found")

           let newGarment: @NFT <- create NFT(serialNumber: numInGarment + 1, garmentDataID: garmentDataID, royaltyVault: royaltyVault)

           return <-newGarment
       }

       pub fun batchMintNFT(garmentDataID: UInt32, royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>, quantity: UInt64): @Collection {
           let newCollection <- create Collection()

           var i: UInt64 = 0
           while i < quantity {
               newCollection.deposit(token: <-self.mintNFT(garmentDataID: garmentDataID, royaltyVault: royaltyVault))
               i = i + 1 as UInt64
           }

           return <-newCollection
       }

       // Change the royalty percentage of the contract
       pub fun changeRoyaltyPercentage(newRoyaltyPercentage: UFix64) {
           GarmentNFT.royaltyPercentage = newRoyaltyPercentage

           emit RoyaltyPercentageChanged(newRoyaltyPercentage: newRoyaltyPercentage)
       }

       // Retire garmentData so that it cannot be used to mint anymore
       pub fun retireGarmentData(garmentDataID: UInt32) {
           pre {
               GarmentNFT.isGarmentDataRetired[garmentDataID] != nil: "Cannot retire Garment: Garment doesn't exist!"
           }

           if !GarmentNFT.isGarmentDataRetired[garmentDataID]! {
               GarmentNFT.isGarmentDataRetired[garmentDataID] = true

               emit GarmentDataIDRetired(garmentDataID: garmentDataID)
           }
       }
   }

   // This is the interface users can cast their Garment Collection as
   // to allow others to deposit into their Collection. It also allows for reading
   // the IDs of Garment in the Collection.
   pub resource interface GarmentCollectionPublic {
       pub fun deposit(token: @NonFungibleToken.NFT)
       pub fun batchDeposit(tokens: @NonFungibleToken.Collection)
       pub fun getIDs(): [UInt64]
       pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT
       pub fun borrowGarment(id: UInt64): &GarmentNFT.NFT? {
           // If the result isn't nil, the id of the returned reference
           // should be the same as the argument to the function
           post {
               (result == nil) || (result?.id == id):
                   "Cannot borrow Garment reference: The ID of the returned reference is incorrect"
           }
       }
   }

   // Collection is a resource that every user who owns NFTs
   // will store in their account to manage their NFTS
   //
   pub resource Collection: GarmentCollectionPublic, NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic {
       // Dictionary of Garment conforming tokens
       // NFT is a resource type with a UInt64 ID field
       pub var ownedNFTs: @{UInt64: NonFungibleToken.NFT}

       init() {
           self.ownedNFTs <- {}
       }

       // withdraw removes an Garment from the Collection and moves it to the caller
       //
       // Parameters: withdrawID: The ID of the NFT
       // that is to be removed from the Collection
       //
       // returns: @NonFungibleToken.NFT the token that was withdrawn
       pub fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
           // Remove the nft from the Collection
           let token <- self.ownedNFTs.remove(key: withdrawID)
               ?? panic("Cannot withdraw: Garment does not exist in the collection")

           emit Withdraw(id: token.id, from: self.owner?.address)

           // Return the withdrawn token
           return <-token
       }

       // batchWithdraw withdraws multiple tokens and returns them as a Collection
       //
       // Parameters: ids: An array of IDs to withdraw
       //
       // Returns: @NonFungibleToken.Collection: A collection that contains
       //                                        the withdrawn Garment
       //
       pub fun batchWithdraw(ids: [UInt64]): @NonFungibleToken.Collection {
           // Create a new empty Collection
           var batchCollection <- create Collection()

           // Iterate through the ids and withdraw them from the Collection
           for id in ids {
               batchCollection.deposit(token: <-self.withdraw(withdrawID: id))
           }

           // Return the withdrawn tokens
           return <-batchCollection
       }

       // deposit takes a Garment and adds it to the Collections dictionary
       //
       // Parameters: token: the NFT to be deposited in the collection
       //
       pub fun deposit(token: @NonFungibleToken.NFT) {
           // Cast the deposited token as NFT to make sure
           // it is the correct type
           let token <- token as! @GarmentNFT.NFT

           // Get the token's ID
           let id = token.id

           // Add the new token to the dictionary
           let oldToken <- self.ownedNFTs[id] <- token

           // Only emit a deposit event if the Collection
           // is in an account's storage
           if self.owner?.address != nil {
               emit Deposit(id: id, to: self.owner?.address)
           }

           // Destroy the empty old token tGarment was "removed"
           destroy oldToken
       }

       // batchDeposit takes a Collection object as an argument
       // and deposits each contained NFT into this Collection
       pub fun batchDeposit(tokens: @NonFungibleToken.Collection) {
           // Get an array of the IDs to be deposited
           let keys = tokens.getIDs()

           // Iterate through the keys in the collection and deposit each one
           for key in keys {
               self.deposit(token: <-tokens.withdraw(withdrawID: key))
           }

           // Destroy the empty Collection
           destroy tokens
       }

       // getIDs returns an array of the IDs that are in the Collection
       pub fun getIDs(): [UInt64] {
           return self.ownedNFTs.keys
       }

       // borrowNFT Returns a borrowed reference to a Garment in the Collection
       // so tGarment the caller can read its ID
       //
       // Parameters: id: The ID of the NFT to get the reference for
       //
       // Returns: A reference to the NFT
       //
       // Note: This only allows the caller to read the ID of the NFT,
       // not an specific data. Please use borrowGarment to
       // read Garment data.
       //
       pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
           return (&self.ownedNFTs[id] as &NonFungibleToken.NFT?)!
       }

       // Parameters: id: The ID of the NFT to get the reference for
       //
       // Returns: A reference to the NFT
       pub fun borrowGarment(id: UInt64): &GarmentNFT.NFT? {
           if self.ownedNFTs[id] != nil {
               let ref = &self.ownedNFTs[id] as auth &NonFungibleToken.NFT?
               return ref as! &GarmentNFT.NFT?
           } else {
               return nil
           }
       }

       // If a transaction destroys the Collection object,
       // All the NFTs contained within are also destroyed!
       //
       destroy() {
           destroy self.ownedNFTs
       }
   }

   // -----------------------------------------------------------------------
   // Garment contract-level function definitions
   // -----------------------------------------------------------------------

   // createEmptyCollection creates a new, empty Collection object so that
   // a user can store it in their account storage.
   // Once they have a Collection in their storage, they are able to receive
   // Garment in transactions.
   //
   pub fun createEmptyCollection(): @NonFungibleToken.Collection {
       return <-create GarmentNFT.Collection()
   }

   // get dictionary of numberMintedPerGarment
   pub fun getNumberMintedPerGarment(): {UInt32: UInt32} {
       return GarmentNFT.numberMintedPerGarment
   }

   // get how many Garments with garmentDataID are minted
   pub fun getGarmentNumberMinted(id: UInt32): UInt32 {
       let numberMinted = GarmentNFT.numberMintedPerGarment[id]??
           panic("garmentDataID not found")
       return numberMinted
   }

   // get the garmentData of a specific id
   pub fun getGarmentData(id: UInt32): GarmentData {
       let garmentData = GarmentNFT.garmentDatas[id]??
           panic("garmentDataID not found")
       return garmentData
   }

   // get all garmentDatas created
   pub fun getGarmentDatas(): {UInt32: GarmentData} {
       return GarmentNFT.garmentDatas
   }

   pub fun getGarmentDatasRetired(): {UInt32: Bool} {
       return GarmentNFT.isGarmentDataRetired
   }

   pub fun getGarmentDataRetired(garmentDataID: UInt32): Bool {
       let isGarmentDataRetired = GarmentNFT.isGarmentDataRetired[garmentDataID]??
           panic("garmentDataID not found")
       return isGarmentDataRetired
   }

   // -----------------------------------------------------------------------
   // initialization function
   // -----------------------------------------------------------------------
   //
   init() {
       // Initialize contract fields
       self.garmentDatas = {}
       self.numberMintedPerGarment = {}
       self.nextGarmentDataID = 1
       self.royaltyPercentage = 0.10
       self.isGarmentDataRetired = {}
       self.totalSupply = 0
       self.CollectionPublicPath = /public/GarmentCollection0007
       self.CollectionStoragePath = /storage/GarmentCollection0007
       self.AdminStoragePath = /storage/GarmentAdmin0007

       // Put a new Collection in storage
       self.account.save<@Collection>(<- create Collection(), to: self.CollectionStoragePath)

       // Create a public capability for the Collection
       self.account.link<&{GarmentCollectionPublic}>(self.CollectionPublicPath, target: self.CollectionStoragePath)

       // Put the Minter in storage
       self.account.save<@Admin>(<- create Admin(), to: self.AdminStoragePath)

       emit ContractInitialized()
   }
}
`

	const materialContract = `
import NonFungibleToken from 0x631e88ae7f1d7c20
import FungibleToken from 0x9a0766d93b6608b7
import FBRC from 0x5a76b4858ce34b2f

pub contract MaterialNFT: NonFungibleToken {

   // -----------------------------------------------------------------------
   // MaterialNFT contract Events
   // -----------------------------------------------------------------------

   // Emitted when the Material contract is created
   pub event ContractInitialized()

   // Emitted when a new MaterialData struct is created
   pub event MaterialDataCreated(materialDataID: UInt32, mainImage: String, secondImage: String, name: String, description: String)

   // Emitted when a Material is minted
   pub event MaterialMinted(materialID: UInt64, materialDataID: UInt32, serialNumber: UInt32)

   // Emitted when the contract's royalty percentage is changed
   pub event RoyaltyPercentageChanged(newRoyaltyPercentage: UFix64)

   pub event MaterialDataIDRetired(materialDataID: UInt32)

   // Events for Collection-related actions
   //
   // Emitted when a Material is withdrawn from a Collection
   pub event Withdraw(id: UInt64, from: Address?)

   // Emitted when a Material is deposited into a Collection
   pub event Deposit(id: UInt64, to: Address?)

   // Emitted when a Material is destroyed
   pub event MaterialDestroyed(id: UInt64)

   // -----------------------------------------------------------------------
   // contract-level fields.
   // These contain actual values that are stored in the smart contract.
   // -----------------------------------------------------------------------

   // Contains standard storage and public paths of resources
   pub let CollectionStoragePath: StoragePath

   pub let CollectionPublicPath: PublicPath

   pub let AdminStoragePath: StoragePath

   // Variable size dictionary of Material structs
   access(self) var materialDatas: {UInt32: MaterialData}

   // Dictionary with MaterialDataID as key and number of NFTs with MaterialDataID are minted
   access(self) var numberMintedPerMaterial: {UInt32: UInt32}

   // Dictionary of materialDataID to  whether they are retired
   access(self) var isMaterialDataRetired: {UInt32: Bool}

   // Keeps track of how many unique MaterialData's are created
   pub var nextMaterialDataID: UInt32

   pub var royaltyPercentage: UFix64

   pub var totalSupply: UInt64

   pub struct MaterialData {

       // The unique ID for the Material Data
       pub let materialDataID: UInt32

       //stores link to image
       pub let mainImage: String
       pub let secondImage: String
       pub let name: String
       pub let description: String

       init(
           mainImage: String,
           secondImage: String,
           name: String,
           description: String
       ){
           self.materialDataID = MaterialNFT.nextMaterialDataID
           self.mainImage = mainImage
           self.secondImage = secondImage
           self.name = name
           self.description = description

           MaterialNFT.isMaterialDataRetired[self.materialDataID] = false

           // Increment the ID so that it isn't used again
           MaterialNFT.nextMaterialDataID = MaterialNFT.nextMaterialDataID + 1 as UInt32

           emit MaterialDataCreated(materialDataID: self.materialDataID, mainImage: self.mainImage, secondImage: self.secondImage, name: self.name, description: self.description)
       }
   }

   pub struct Material {

       // The ID of the MaterialData that the Material references
       pub let materialDataID: UInt32

       // The N'th NFT with 'MaterialDataID' minted
       pub let serialNumber: UInt32

       init(materialDataID: UInt32) {
           self.materialDataID = materialDataID

           // Increment the ID so that it isn't used again
           MaterialNFT.numberMintedPerMaterial[materialDataID] = MaterialNFT.numberMintedPerMaterial[materialDataID]! + 1 as UInt32

           self.serialNumber = MaterialNFT.numberMintedPerMaterial[materialDataID]!
       }
   }

   // The resource that represents the Material NFTs
   //
   pub resource NFT: NonFungibleToken.INFT {

       // Global unique Material ID
       pub let id: UInt64

       // struct of Material
       pub let material: Material

       // Royalty capability which NFT will use
       pub let royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>

       init(serialNumber: UInt32, materialDataID: UInt32, royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>) {
           MaterialNFT.totalSupply = MaterialNFT.totalSupply + 1 as UInt64

           self.id = MaterialNFT.totalSupply

           self.material = Material(materialDataID: materialDataID)

           self.royaltyVault = royaltyVault

           // Emitted when a Material is minted
           emit MaterialMinted(materialID: self.id, materialDataID: materialDataID, serialNumber: serialNumber)
       }

       destroy() {
           emit MaterialDestroyed(id: self.id)
       }

   }

   // Admin is a special authorization resource that
   // allows the owner to perform important functions to modify the
   // various aspects of the Material and NFTs
   //
   pub resource Admin {

       pub fun createMaterialData(
           mainImage: String,
           secondImage: String,
           name: String,
           description: String
       ): UInt32 {
           // Create the new MaterialData
           var newMaterial = MaterialData(
               mainImage: mainImage,
               secondImage: secondImage,
               name: name,
               description: description
           )

           let newID = newMaterial.materialDataID

           // Store it in the contract storage
           MaterialNFT.materialDatas[newID] = newMaterial

           MaterialNFT.numberMintedPerMaterial[newID] = 0 as UInt32

           return newID
       }

       // createNewAdmin creates a new Admin resource
       //
       pub fun createNewAdmin(): @Admin {
           return <-create Admin()
       }

       // Mint the new Material
       pub fun mintNFT(materialDataID: UInt32, royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>): @NFT {
           pre {
               royaltyVault.check():
                   "Royalty capability is invalid!"
           }

           if (MaterialNFT.isMaterialDataRetired[materialDataID]! == nil) {
               panic("Cannot mint Material. materialData not found")
           }

           if (MaterialNFT.isMaterialDataRetired[materialDataID]!) {
               panic("Cannot mint material. materialDataID retired")
           }

           let numInMaterial = MaterialNFT.numberMintedPerMaterial[materialDataID]??
               panic("no materialDataID found")

           let newMaterial: @NFT <- create NFT(serialNumber: numInMaterial + 1, materialDataID: materialDataID, royaltyVault: royaltyVault)

           return <-newMaterial
       }

       pub fun batchMintNFT(materialDataID: UInt32, royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>, quantity: UInt64): @Collection {
           let newCollection <- create Collection()

           var i: UInt64 = 0
           while i < quantity {
               newCollection.deposit(token: <-self.mintNFT(materialDataID: materialDataID, royaltyVault: royaltyVault))
               i = i + 1 as UInt64
           }

           return <-newCollection
       }

       // Change the royalty percentage of the contract
       pub fun changeRoyaltyPercentage(newRoyaltyPercentage: UFix64) {
           MaterialNFT.royaltyPercentage = newRoyaltyPercentage

           emit RoyaltyPercentageChanged(newRoyaltyPercentage: newRoyaltyPercentage)
       }

       // Retire materialData so that it cannot be used to mint anymore
       pub fun retireMaterialData(materialDataID: UInt32) {
           pre {
               MaterialNFT.isMaterialDataRetired[materialDataID] != nil: "Cannot retire Material: Material doesn't exist!"
           }

           if !MaterialNFT.isMaterialDataRetired[materialDataID]! {
               MaterialNFT.isMaterialDataRetired[materialDataID] = true

               emit MaterialDataIDRetired(materialDataID: materialDataID)
           }
       }
   }

   // This is the interface users can cast their Material Collection as
   // to allow others to deposit into their Collection. It also allows for reading
   // the IDs of Material in the Collection.
   pub resource interface MaterialCollectionPublic {
       pub fun deposit(token: @NonFungibleToken.NFT)
       pub fun batchDeposit(tokens: @NonFungibleToken.Collection)
       pub fun getIDs(): [UInt64]
       pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT
       pub fun borrowMaterial(id: UInt64): &MaterialNFT.NFT? {
           // If the result isn't nil, the id of the returned reference
           // should be the same as the argument to the function
           post {
               (result == nil) || (result?.id == id):
                   "Cannot borrow Material reference: The ID of the returned reference is incorrect"
           }
       }
   }

   // Collection is a resource that every user who owns NFTs
   // will store in their account to manage their NFTS
   //
   pub resource Collection: MaterialCollectionPublic, NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic {
       // Dictionary of Material conforming tokens
       // NFT is a resource type with a UInt64 ID field
       pub var ownedNFTs: @{UInt64: NonFungibleToken.NFT}

       init() {
           self.ownedNFTs <- {}
       }

       // withdraw removes an Material from the Collection and moves it to the caller
       //
       // Parameters: withdrawID: The ID of the NFT
       // that is to be removed from the Collection
       //
       // returns: @NonFungibleToken.NFT the token that was withdrawn
       pub fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
           // Remove the nft from the Collection
           let token <- self.ownedNFTs.remove(key: withdrawID)
               ?? panic("Cannot withdraw: Material does not exist in the collection")

           emit Withdraw(id: token.id, from: self.owner?.address)

           // Return the withdrawn token
           return <-token
       }

       // batchWithdraw withdraws multiple tokens and returns them as a Collection
       //
       // Parameters: ids: An array of IDs to withdraw
       //
       // Returns: @NonFungibleToken.Collection: A collection that contains
       //                                        the withdrawn Material
       //
       pub fun batchWithdraw(ids: [UInt64]): @NonFungibleToken.Collection {
           // Create a new empty Collection
           var batchCollection <- create Collection()

           // Iterate through the ids and withdraw them from the Collection
           for id in ids {
               batchCollection.deposit(token: <-self.withdraw(withdrawID: id))
           }

           // Return the withdrawn tokens
           return <-batchCollection
       }

       // deposit takes a Material and adds it to the Collections dictionary
       //
       // Parameters: token: the NFT to be deposited in the collection
       //
       pub fun deposit(token: @NonFungibleToken.NFT) {
           // Cast the deposited token as NFT to make sure
           // it is the correct type
           let token <- token as! @MaterialNFT.NFT

           // Get the token's ID
           let id = token.id

           // Add the new token to the dictionary
           let oldToken <- self.ownedNFTs[id] <- token

           // Only emit a deposit event if the Collection
           // is in an account's storage
           if self.owner?.address != nil {
               emit Deposit(id: id, to: self.owner?.address)
           }

           // Destroy the empty old token tMaterial was "removed"
           destroy oldToken
       }

       // batchDeposit takes a Collection object as an argument
       // and deposits each contained NFT into this Collection
       pub fun batchDeposit(tokens: @NonFungibleToken.Collection) {
           // Get an array of the IDs to be deposited
           let keys = tokens.getIDs()

           // Iterate through the keys in the collection and deposit each one
           for key in keys {
               self.deposit(token: <-tokens.withdraw(withdrawID: key))
           }

           // Destroy the empty Collection
           destroy tokens
       }

       // getIDs returns an array of the IDs that are in the Collection
       pub fun getIDs(): [UInt64] {
           return self.ownedNFTs.keys
       }

       // borrowNFT Returns a borrowed reference to a Material in the Collection
       // so tMaterial the caller can read its ID
       //
       // Parameters: id: The ID of the NFT to get the reference for
       //
       // Returns: A reference to the NFT
       //
       // Note: This only allows the caller to read the ID of the NFT,
       // not an specific data. Please use borrowMaterial to
       // read Material data.
       //
       pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
           return (&self.ownedNFTs[id] as &NonFungibleToken.NFT?)!
       }

       // Parameters: id: The ID of the NFT to get the reference for
       //
       // Returns: A reference to the NFT
       pub fun borrowMaterial(id: UInt64): &MaterialNFT.NFT? {
           if self.ownedNFTs[id] != nil {
               let ref = (&self.ownedNFTs[id] as auth &NonFungibleToken.NFT?)!
               return ref as! &MaterialNFT.NFT
           } else {
               return nil
           }
       }

       // If a transaction destroys the Collection object,
       // All the NFTs contained within are also destroyed!
       //
       destroy() {
           destroy self.ownedNFTs
       }
   }

   // -----------------------------------------------------------------------
   // Material contract-level function definitions
   // -----------------------------------------------------------------------

   // createEmptyCollection creates a new, empty Collection object so that
   // a user can store it in their account storage.
   // Once they have a Collection in their storage, they are able to receive
   // Material in transactions.
   //
   pub fun createEmptyCollection(): @NonFungibleToken.Collection {
       return <-create MaterialNFT.Collection()
   }

   // get dictionary of numberMintedPerMaterial
   pub fun getNumberMintedPerMaterial(): {UInt32: UInt32} {
       return MaterialNFT.numberMintedPerMaterial
   }

   // get how many Materials with materialDataID are minted
   pub fun getMaterialNumberMinted(id: UInt32): UInt32 {
       let numberMinted = MaterialNFT.numberMintedPerMaterial[id]??
           panic("materialDataID not found")
       return numberMinted
   }

   // get the materialData of a specific id
   pub fun getMaterialData(id: UInt32): MaterialData {
       let materialData = MaterialNFT.materialDatas[id]??
           panic("materialDataID not found")
       return materialData
   }

   // get all materialDatas created
   pub fun getMaterialDatas(): {UInt32: MaterialData} {
       return MaterialNFT.materialDatas
   }

   pub fun getMaterialDatasRetired(): {UInt32: Bool} {
       return MaterialNFT.isMaterialDataRetired
   }

   pub fun getMaterialDataRetired(materialDataID: UInt32): Bool {
       let isMaterialDataRetired = MaterialNFT.isMaterialDataRetired[materialDataID]??
           panic("materialDataID not found")
       return isMaterialDataRetired
   }

   // -----------------------------------------------------------------------
   // initialization function
   // -----------------------------------------------------------------------
   //
   init() {
       // Initialize contract fields
       self.materialDatas = {}
       self.numberMintedPerMaterial = {}
       self.nextMaterialDataID = 1
       self.royaltyPercentage = 0.10
       self.isMaterialDataRetired = {}
       self.totalSupply = 0
       self.CollectionPublicPath = /public/MaterialCollection0007
       self.CollectionStoragePath = /storage/MaterialCollection0007
       self.AdminStoragePath = /storage/MaterialAdmin0007

       // Put a new Collection in storage
       self.account.save<@Collection>(<- create Collection(), to: self.CollectionStoragePath)

       // Create a public capability for the Collection
       self.account.link<&{MaterialCollectionPublic}>(self.CollectionPublicPath, target: self.CollectionStoragePath)

       // Put the Minter in storage
       self.account.save<@Admin>(<- create Admin(), to: self.AdminStoragePath)

       emit ContractInitialized()
   }
}
`

	const itemContract = `
import NonFungibleToken from 0x631e88ae7f1d7c20
import FungibleToken from 0x9a0766d93b6608b7
import GarmentNFT from 0x5a76b4858ce34b2f
import MaterialNFT from 0x5a76b4858ce34b2f
import FBRC from 0x5a76b4858ce34b2f

pub contract ItemNFT: NonFungibleToken {

   // -----------------------------------------------------------------------
   // ItemNFT contract Events
   // -----------------------------------------------------------------------

   // Emitted when the Item contract is created
   pub event ContractInitialized()

   // Emitted when a new ItemData struct is created
   pub event ItemDataCreated(itemDataID: UInt32, mainImage: String, images: [String])

   // Emitted when a Item is mintee
   pub event ItemMinted(itemID: UInt64, itemDataID: UInt32, serialNumber: UInt32)

   // Emitted when a Item' name is changed
   pub event ItemNameChanged(id: UInt64, name: String)

   // Emitted when the contract's royalty percentage is changed
   pub event RoyaltyPercentageChanged(newRoyaltyPercentage: UFix64)

   pub event ItemDataAllocated(garmentDataID: UInt32, materialDataID: UInt32, itemDataID: UInt32)

   // Emitted when the items are set to be splittable
   pub event ItemNFTNowSplittable()

   pub event numberItemDataMintableChanged(number: UInt32)

   pub event ItemDataIDRetired(itemDataID: UInt32)

   // Events for Collection-related actions
   //
   // Emitted when a Item is withdrawn from a Collection
   pub event Withdraw(id: UInt64, from: Address?)

   // Emitted when a Item is deposited into a Collection
   pub event Deposit(id: UInt64, to: Address?)

   // Emitted when a Item is destroyed
   pub event ItemDestroyed(id: UInt64)

   // -----------------------------------------------------------------------
   // contract-level fields.
   // These contain actual values that are stored in the smart contract.
   // -----------------------------------------------------------------------

   pub let CollectionStoragePath: StoragePath

   pub let CollectionPublicPath: PublicPath

   pub let AdminStoragePath: StoragePath

   // Dictionary with ItemDataID as key and number of NFTs with that ItemDataID are minted
   access(self) var numberMintedPerItem: {UInt32: UInt32}

   // Variable size dictionary of Item structs
   access(self) var itemDatas: {UInt32: ItemData}

   // ItemData of item minted is based on garmentDataID of garment and materialDataID of material used {materialDataID: {garmentDataID: itemDataID}
   access(self) var itemDataAllocation: {UInt32: {UInt32: UInt32}}

   // Dictionary of itemDataID to  whether they are retired
   access(self) var isItemDataRetired: {UInt32: Bool}

   // Keeps track of how many unique ItemData's are created
   pub var nextItemDataID: UInt32

   pub var nextItemDataAllocation: UInt32

   // Are garment and material removable from item
   pub var isSplittable: Bool

   // The maximum number of items with itemDataID mintable
   pub var numberItemDataMintable: UInt32

   pub var royaltyPercentage: UFix64

   pub var totalSupply: UInt64

   pub struct ItemData {

       // The unique ID for the Item Data
       pub let itemDataID: UInt32
       //stores link to image
       pub let mainImage: String
       //stores link to supporting images
       pub let images: [String]

       init(
           mainImage: String,
           images: [String],
       ){
           self.itemDataID = ItemNFT.nextItemDataID
           self.mainImage = mainImage
           self.images = images

           ItemNFT.isItemDataRetired[self.itemDataID] = false

           // Increment the ID so that it isn't used again
           ItemNFT.nextItemDataID = ItemNFT.nextItemDataID + 1 as UInt32

           emit ItemDataCreated(itemDataID: self.itemDataID, mainImage: self.mainImage, images: self.images)
       }
   }

   pub struct Item {

       // The ID of the itemData that the item references
       pub let itemDataID: UInt32

       // The N'th NFT with 'ItemDataID' minted
       pub let serialNumber: UInt32

       init(itemDataID: UInt32) {
           pre {
               //Only one Item with 'ItemDataID' can be minted
               ItemNFT.numberMintedPerItem[itemDataID] == ItemNFT.numberItemDataMintable - 1 as UInt32: "ItemNFT with itemDataID already minted"
           }

           self.itemDataID = itemDataID

           // Increment the ID so that it isn't used again
           ItemNFT.numberMintedPerItem[itemDataID] = ItemNFT.numberMintedPerItem[itemDataID]! + 1 as UInt32

           self.serialNumber = ItemNFT.numberMintedPerItem[itemDataID]!

       }
   }

   // The resource that represents the Item NFTs
   //
   pub resource NFT: NonFungibleToken.INFT {

       // Global unique Item ID
       pub let id: UInt64

       // struct of Item
       pub let item: Item

       // name of nft, can be changed
       pub var name: String

       // Royalty capability which NFT will use
       pub let royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>

       // after you remove the garment and material from the item, the ItemNFT will be considered "dead".
       // accounts will be unable to deposit, withdraw or call functions of the nft.
       pub var isDead : Bool

       // this is where the garment nft is stored, it cannot be moved out
       access(self) var garment: @GarmentNFT.NFT?

       // this is where the material nft is stored, it cannot be moved out
       access(self) var material: @MaterialNFT.NFT?


       init(serialNumber: UInt32, name: String, itemDataID: UInt32, royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>, garment: @GarmentNFT.NFT, material: @MaterialNFT.NFT) {

           ItemNFT.totalSupply = ItemNFT.totalSupply + 1 as UInt64

           self.id = ItemNFT.totalSupply

           self.name = name

           self.royaltyVault = royaltyVault

           self.isDead = false

           self.garment <- garment

           self.material <- material

           self.item = Item(itemDataID: itemDataID)

           // Emitted when a Item is minted
           emit ItemMinted(itemID: self.id, itemDataID: itemDataID, serialNumber: serialNumber)

       }

       destroy() {
           emit ItemDestroyed(id: self.id)
           //destroy self.items
           destroy self.garment
           destroy self.material
       }

       //Make Item considered dead. Deposit garment and material to respective vaults
       pub fun split(garmentCap: Capability<&{GarmentNFT.GarmentCollectionPublic}>, materialCap: Capability<&{MaterialNFT.MaterialCollectionPublic}>) {
           pre {
               !self.isDead:
               "Cannot split. Item is dead"
               ItemNFT.isSplittable:
               "Item is set to unsplittable"
               garmentCap.check():
               "Garment Capability is invalid"
               materialCap.check():
               "Material Capability is invalid"
           }
           let garmentOptional <- self.garment <- nil
           let materialOptional <- self.material <- nil
           let garmentRecipient = garmentCap.borrow()!
           let materialRecipient = materialCap.borrow()!
           let garment <- garmentOptional!
           let material <- materialOptional!
           let garmentNFT <- garment as! @NonFungibleToken.NFT
           let materialNFT <- material as! @NonFungibleToken.NFT
           garmentRecipient.deposit(token: <- garmentNFT)
           materialRecipient.deposit(token: <- materialNFT)
           ItemNFT.numberMintedPerItem[self.item.itemDataID] = ItemNFT.numberMintedPerItem[self.item.itemDataID]! - 1 as UInt32
           self.isDead = true
       }

       // get a reference to the garment that item stores
       pub fun borrowGarment(): &GarmentNFT.NFT? {
           let garmentOptional <- self.garment <- nil
           let garment <- garmentOptional!
           let garmentRef = &garment as auth &GarmentNFT.NFT
           self.garment <-! garment
           return garmentRef
       }

       // get a reference to the material that item stores
       pub fun borrowMaterial(): &MaterialNFT.NFT?  {
           let materialOptional <- self.material <- nil
           let material <- materialOptional!
           let materialRef = &material as auth &MaterialNFT.NFT
           self.material <-! material
           return materialRef
       }

       // change name of item nft
       pub fun changeName(name: String) {
           pre {
               !self.isDead:
               "Cannot change garment name. Item is dead"
           }
           self.name = name;

          emit ItemNameChanged(id: self.id, name: self.name)
       }
   }

   //destroy item if it is considered dead
   pub fun cleanDeadItems(item: @ItemNFT.NFT) {
       pre {
           item.isDead:
           "Cannot destroy, item not dead"
       }
       destroy item
   }

   // mint the NFT, combining a garment and boot.
   // The itemData that is used to mint the Item is based on the garment and material' garmentDataID and materialDataID
   pub fun mintNFT(name: String, royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>, garment: @GarmentNFT.NFT, material: @MaterialNFT.NFT): @NFT {
       pre {
           royaltyVault.check():
               "Royalty capability is invalid!"
       }

       let garmentDataID = garment.garment.garmentDataID

       let materialDataID = material.material.materialDataID

       let isValidGarmentMaterialPair = ItemNFT.itemDataAllocation[garmentDataID]??
           panic("garment and material dataID pair not allocated")

       // get the itemdataID of the item to be minted based on garment and material dataIDs
       let itemDataID = isValidGarmentMaterialPair[materialDataID]??
           panic("itemDataID not allocated")

       if (ItemNFT.isItemDataRetired[itemDataID]! == nil) {
           panic("Cannot mint Item. ItemData not found")
       }

       if (ItemNFT.isItemDataRetired[itemDataID]!) {
           panic("Cannot mint Item. ItemDataID retired")
       }

       let numInItem = ItemNFT.numberMintedPerItem[itemDataID]??
           panic("itemDataID not found")

       let item <-create NFT(serialNumber: numInItem + 1, name: name, itemDataID: itemDataID, royaltyVault: royaltyVault, garment: <- garment, material: <- material)

       return <- item
   }

   // This is the interface that users can cast their Item Collection as
   // to allow others to deposit Items into their Collection. It also allows for reading
   // the IDs of Items in the Collection.
   pub resource interface ItemCollectionPublic {
       pub fun deposit(token: @NonFungibleToken.NFT)
       pub fun batchDeposit(tokens: @NonFungibleToken.Collection)
       pub fun getIDs(): [UInt64]
       pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT
       pub fun borrowItem(id: UInt64): &ItemNFT.NFT? {
           // If the result isn't nil, the id of the returned reference
           // should be the same as the argument to the function
           post {
               (result == nil) || (result?.id == id):
                   "Cannot borrow Item reference: The ID of the returned reference is incorrect"
           }
       }
   }

   // Collection is a resource that every user who owns NFTs
   // will store in their account to manage their NFTS
   //
   pub resource Collection: ItemCollectionPublic, NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic {
       // Dictionary of Item conforming tokens
       // NFT is a resource type with a UInt64 ID field
       pub var ownedNFTs: @{UInt64: NonFungibleToken.NFT}

       init() {
           self.ownedNFTs <- {}
       }

       // withdraw removes an Item from the Collection and moves it to the caller
       //
       // Parameters: withdrawID: The ID of the NFT
       // that is to be removed from the Collection
       //
       // returns: @NonFungibleToken.NFT the token that was withdrawn
       pub fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
           // Remove the nft from the Collection
           let token <- self.ownedNFTs.remove(key: withdrawID)
               ?? panic("Cannot withdraw: Item does not exist in the collection")

           emit Withdraw(id: token.id, from: self.owner?.address)

           // Return the withdrawn token
           return <-token
       }

       // batchWithdraw withdraws multiple tokens and returns them as a Collection
       //
       // Parameters: ids: An array of IDs to withdraw
       //
       // Returns: @NonFungibleToken.Collection: A collection that contains
       //                                        the withdrawn Items
       //
       pub fun batchWithdraw(ids: [UInt64]): @NonFungibleToken.Collection {
           // Create a new empty Collection
           var batchCollection <- create Collection()

           // Iterate through the ids and withdraw them from the Collection
           for id in ids {
               batchCollection.deposit(token: <-self.withdraw(withdrawID: id))
           }

           // Return the withdrawn tokens
           return <-batchCollection
       }

       // deposit takes a Item and adds it to the Collections dictionary
       //
       // Parameters: token: the NFT to be deposited in the collection
       //
       pub fun deposit(token: @NonFungibleToken.NFT) {
           //todo: someFunction that transfers royalty
           // Cast the deposited token as  NFT to make sure
           // it is the correct type
           let token <- token as! @ItemNFT.NFT

           // Get the token's ID
           let id = token.id

           // Add the new token to the dictionary
           let oldToken <- self.ownedNFTs[id] <- token

           // Only emit a deposit event if the Collection
           // is in an account's storage
           if self.owner?.address != nil {
               emit Deposit(id: id, to: self.owner?.address)
           }

           // Destroy the empty old token that was "removed"
           destroy oldToken
       }

       // batchDeposit takes a Collection object as an argument
       // and deposits each contained NFT into this Collection
       pub fun batchDeposit(tokens: @NonFungibleToken.Collection) {
           // Get an array of the IDs to be deposited
           let keys = tokens.getIDs()

           // Iterate through the keys in the collection and deposit each one
           for key in keys {
               self.deposit(token: <-tokens.withdraw(withdrawID: key))
           }

           // Destroy the empty Collection
           destroy tokens
       }

       // getIDs returns an array of the IDs that are in the Collection
       pub fun getIDs(): [UInt64] {
           return self.ownedNFTs.keys
       }

       // borrowNFT Returns a borrowed reference to a Item in the Collection
       // so that the caller can read its ID
       //
       // Parameters: id: The ID of the NFT to get the reference for
       //
       // Returns: A reference to the NFT
       //
       // Note: This only allows the caller to read the ID of the NFT,
       // not an specific data. Please use borrowItem to
       // read Item data.
       //
       pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
           return (&self.ownedNFTs[id] as &NonFungibleToken.NFT?)!
       }

       // Parameters: id: The ID of the NFT to get the reference for
       //
       // Returns: A reference to the NFT
       pub fun borrowItem(id: UInt64): &ItemNFT.NFT? {
           if self.ownedNFTs[id] != nil {
               let ref = (&self.ownedNFTs[id] as auth &NonFungibleToken.NFT?)!
               return ref as! &ItemNFT.NFT
           } else {
               return nil
           }
       }

       // If a transaction destroys the Collection object,
       // All the NFTs contained within are also destroyed!
       //
       destroy() {
           destroy self.ownedNFTs
       }
   }

   // Admin is a special authorization resource that
   // allows the owner to perform important functions to modify the
   // various aspects of the Items and NFTs
   //
   pub resource Admin {

       // create itemdataid allocation from the garmentdataid and materialdataid
       pub fun createItemDataAllocation(garmentDataID: UInt32, materialDataID: UInt32){

           if(ItemNFT.itemDataAllocation[garmentDataID] != nil) {
               if(ItemNFT.itemDataAllocation[garmentDataID]![materialDataID] != nil){
                   panic("ItemData already allocated")
               } else {
                   let dict = ItemNFT.itemDataAllocation[garmentDataID]!
                   dict[materialDataID] = ItemNFT.nextItemDataAllocation
                   ItemNFT.itemDataAllocation[garmentDataID] = dict
               }
           } else {
               let dict: {UInt32: UInt32} = {}
               dict[materialDataID] = ItemNFT.nextItemDataAllocation
               ItemNFT.itemDataAllocation[garmentDataID] = dict
           }
           emit ItemDataAllocated(garmentDataID: garmentDataID, materialDataID: materialDataID, itemDataID: ItemNFT.nextItemDataAllocation)
           ItemNFT.nextItemDataAllocation = ItemNFT.nextItemDataAllocation + 1 as UInt32

       }

       pub fun createItemData(mainImage: String, images: [String]): UInt32 {
           // Create the new Item
           var newItem = ItemData(mainImage: mainImage, images: images)

           let newID = newItem.itemDataID

           // Store it in the contract storage
           ItemNFT.itemDatas[newID] = newItem

           ItemNFT.numberMintedPerItem[newID] = 0 as UInt32
           return newID
       }

       // createNewAdmin creates a new Admin resource
       //
       pub fun createNewAdmin(): @Admin {
           return <-create Admin()
       }

       // Change the royalty percentage of the contract
       pub fun changeRoyaltyPercentage(newRoyaltyPercentage: UFix64) {
           ItemNFT.royaltyPercentage = newRoyaltyPercentage

           emit RoyaltyPercentageChanged(newRoyaltyPercentage: newRoyaltyPercentage)
       }

       // Change the royalty percentage of the contract
       pub fun makeSplittable() {
           ItemNFT.isSplittable = true

           emit ItemNFTNowSplittable()
       }

       // Change the royalty percentage of the contract
       pub fun changeItemDataNumberMintable(number: UInt32) {
           ItemNFT.numberItemDataMintable = number

           emit numberItemDataMintableChanged(number: number)
       }

       // Retire itemData so that it cannot be used to mint anymore
       pub fun retireItemData(itemDataID: UInt32) {
           pre {
               ItemNFT.isItemDataRetired[itemDataID] != nil: "Cannot retire item: Item doesn't exist!"
           }

           if !ItemNFT.isItemDataRetired[itemDataID]! {
               ItemNFT.isItemDataRetired[itemDataID] = true

               emit ItemDataIDRetired(itemDataID: itemDataID)
           }


       }
   }
   // -----------------------------------------------------------------------
   // Item contract-level function definitions
   // -----------------------------------------------------------------------

   // createEmptyCollection creates a new, empty Collection object so that
   // a user can store it in their account storage.
   // Once they have a Collection in their storage, they are able to receive
   // Items in transactions.
   //
   pub fun createEmptyCollection(): @NonFungibleToken.Collection {
       return <-create ItemNFT.Collection()
   }

   // get dictionary of numberMintedPerItem
   pub fun getNumberMintedPerItem(): {UInt32: UInt32} {
       return ItemNFT.numberMintedPerItem
   }

   // get how many Items with itemDataID are minted
   pub fun getItemNumberMinted(id: UInt32): UInt32 {
       let numberMinted = ItemNFT.numberMintedPerItem[id]??
           panic("itemDataID not found")
       return numberMinted
   }

   // get the ItemData of a specific id
   pub fun getItemData(id: UInt32): ItemData {
       let itemData = ItemNFT.itemDatas[id]??
           panic("itemDataID not found")
       return itemData
   }

   // get the map of item data allocations
   pub fun getItemDataAllocations(): {UInt32: {UInt32: UInt32}} {
       let itemDataAllocation = ItemNFT.itemDataAllocation
       return itemDataAllocation
   }

   // get the itemData allocation from the garment and material dataID
   pub fun getItemDataAllocation(garmentDataID: UInt32, materialDataID: UInt32): UInt32 {
       let isValidGarmentMaterialPair = ItemNFT.itemDataAllocation[garmentDataID]??
           panic("garment and material dataID pair not allocated")

       // get the itemdataID of the item to be minted based on garment and material dataIDs
       let itemDataAllocation = isValidGarmentMaterialPair[materialDataID]??
           panic("itemDataID not allocated")

       return itemDataAllocation
   }
   // get all ItemDatas created
   pub fun getItemDatas(): {UInt32: ItemData} {
       return ItemNFT.itemDatas
   }

   // get dictionary of itemdataids and whether they are retired
   pub fun getItemDatasRetired(): {UInt32: Bool} {
       return ItemNFT.isItemDataRetired
   }

   // get bool of if itemdataid is retired
   pub fun getItemDataRetired(itemDataID: UInt32): Bool? {
       return ItemNFT.isItemDataRetired[itemDataID]!
   }


   // -----------------------------------------------------------------------
   // initialization function
   // -----------------------------------------------------------------------
   //
   init() {
       self.itemDatas = {}
       self.itemDataAllocation = {}
       self.numberMintedPerItem = {}
       self.nextItemDataID = 1
       self.nextItemDataAllocation = 1
       self.isSplittable = false
       self.numberItemDataMintable = 1
       self.isItemDataRetired = {}
       self.royaltyPercentage = 0.10
       self.totalSupply = 0

       self.CollectionPublicPath = /public/ItemCollection0007
       self.CollectionStoragePath = /storage/ItemCollection0007
       self.AdminStoragePath = /storage/ItemAdmin0007

       // Put a new Collection in storage
       self.account.save<@Collection>(<- create Collection(), to: self.CollectionStoragePath)

       // Create a public capability for the Collection
       self.account.link<&{ItemCollectionPublic}>(self.CollectionPublicPath, target: self.CollectionStoragePath)

       // Put the Minter in storage
       self.account.save<@Admin>(<- create Admin(), to: self.AdminStoragePath)

       emit ContractInitialized()
   }
}
`

	accountCodes := map[common.LocationID][]byte{
		common.AddressLocation{
			Address: ftAddress,
			Name:    "FungibleToken",
		}.ID(): []byte(realFungibleTokenContractInterface),
		common.AddressLocation{
			Address: nftAddress,
			Name:    "NonFungibleToken",
		}.ID(): []byte(realNonFungibleTokenInterface),
	}

	var events []cadence.Event

	var signerAddress common.Address

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location.ID()], nil
		},
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location.ID()], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location.ID()] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy contracts

	signerAddress = contractsAddress

	for _, contract := range []struct {
		name string
		code string
	}{
		{"FBRC", fbrcContract},
		{"GarmentNFT", garmentContract},
		{"MaterialNFT", materialContract},
		{"ItemNFT", itemContract},
	} {

		err = runtime.ExecuteTransaction(
			Script{
				Source: utils.DeploymentTransaction(
					contract.name,
					[]byte(contract.code),
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	}

	// Deploy FlowToken contract

	signerAddress = flowTokenAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(fmt.Sprintf(
				`
                 transaction {

                     prepare(signer: AuthAccount) {
                         signer.contracts.add(name: "FlowToken", code: "%s".decodeHex(), signer)
                     }
                 }
               `,
				hex.EncodeToString([]byte(flowTokenContract)),
			)),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Initialize test account

	const initializeAccount = `
import GarmentNFT from 0x5a76b4858ce34b2f
import MaterialNFT from 0x5a76b4858ce34b2f
import ItemNFT from 0x5a76b4858ce34b2f
import FBRC from 0x5a76b4858ce34b2f
import FlowToken from 0x7e60df042a9c0868
import FungibleToken from 0x9a0766d93b6608b7

pub fun hasFBRC(_ address: Address): Bool {
   let receiver = getAccount(address)
       .getCapability<&FBRC.Vault{FungibleToken.Receiver}>(FBRC.CollectionReceiverPath)
       .check()
   let balance = getAccount(address)
       .getCapability<&FBRC.Vault{FungibleToken.Balance}>(FBRC.CollectionBalancePath)
       .check()
   return receiver && balance
}

pub fun hasFlowToken(_ address: Address): Bool {
   let receiver = getAccount(address)
       .getCapability<&FlowToken.Vault{FungibleToken.Receiver}>(/public/flowTokenReceiver)
       .check()
   let balance = getAccount(address)
       .getCapability<&FlowToken.Vault{FungibleToken.Balance}>(/public/flowTokenBalance)
       .check()
   return receiver && balance
}

pub fun hasGarmentNFT(_ address: Address): Bool {
   return getAccount(address)
       .getCapability<&{GarmentNFT.GarmentCollectionPublic}>(GarmentNFT.CollectionPublicPath)
       .check()
}

pub fun hasMaterialNFT(_ address: Address): Bool {
   return getAccount(address)
       .getCapability<&{MaterialNFT.MaterialCollectionPublic}>(MaterialNFT.CollectionPublicPath)
       .check()
}

pub fun hasItemNFT(_ address: Address): Bool {
   return getAccount(address)
   .getCapability<&{ItemNFT.ItemCollectionPublic}>(ItemNFT.CollectionPublicPath)
   .check()
}

transaction {

   prepare(acct: AuthAccount) {
       if !hasFBRC(acct.address) {
       if acct.borrow<&FBRC.Vault>(from: FBRC.CollectionStoragePath) == nil {
           acct.save(<-FBRC.createEmptyVault(), to: FBRC.CollectionStoragePath)
       }
       acct.unlink(FBRC.CollectionReceiverPath)
       acct.unlink(FBRC.CollectionBalancePath)
       acct.link<&FBRC.Vault{FungibleToken.Receiver}>(FBRC.CollectionReceiverPath, target: FBRC.CollectionStoragePath)
       acct.link<&FBRC.Vault{FungibleToken.Balance}>(FBRC.CollectionBalancePath, target: FBRC.CollectionStoragePath)
       }

       if !hasFlowToken(acct.address) {
       if acct.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault) == nil {
         acct.save(<-FlowToken.createEmptyVault(), to: /storage/flowTokenVault)
       }
       acct.unlink(/public/flowTokenReceiver)
       acct.unlink(/public/flowTokenBalance)
       acct.link<&FlowToken.Vault{FungibleToken.Receiver}>(/public/flowTokenReceiver, target: /storage/flowTokenVault)
       acct.link<&FlowToken.Vault{FungibleToken.Balance}>(/public/flowTokenBalance, target: /storage/flowTokenVault)
       }

       if !hasGarmentNFT(acct.address) {
       if acct.borrow<&GarmentNFT.Collection>(from: GarmentNFT.CollectionStoragePath) == nil {
           let collection <- GarmentNFT.createEmptyCollection() as! @GarmentNFT.Collection
           // Put the new Collection in storage
           acct.save(<-collection, to: GarmentNFT.CollectionStoragePath)
       }
       acct.unlink(GarmentNFT.CollectionPublicPath)
       // create a public capability for the collection
       acct.link<&{GarmentNFT.GarmentCollectionPublic}>(GarmentNFT.CollectionPublicPath, target: GarmentNFT.CollectionStoragePath)
       }

       if !hasMaterialNFT(acct.address) {
       if acct.borrow<&MaterialNFT.Collection>(from: MaterialNFT.CollectionStoragePath) == nil {
           let collection <- MaterialNFT.createEmptyCollection() as! @MaterialNFT.Collection
           // Put the new Collection in storage
           acct.save(<-collection, to: MaterialNFT.CollectionStoragePath)
       }
       acct.unlink(MaterialNFT.CollectionPublicPath)
       // create a public capability for the collection
       acct.link<&{MaterialNFT.MaterialCollectionPublic}>(MaterialNFT.CollectionPublicPath, target: MaterialNFT.CollectionStoragePath)
       }

       if !hasItemNFT(acct.address) {
       if acct.borrow<&ItemNFT.Collection>(from: ItemNFT.CollectionStoragePath) == nil {
           let collection <- ItemNFT.createEmptyCollection() as! @ItemNFT.Collection
           // Put the new Collection in storage
           acct.save(<-collection, to: ItemNFT.CollectionStoragePath)
       }
       acct.unlink(ItemNFT.CollectionPublicPath)
       // create a public capability for the collection
       acct.link<&{ItemNFT.ItemCollectionPublic}>(ItemNFT.CollectionPublicPath, target: ItemNFT.CollectionStoragePath)
       }
   }

}`

	signerAddress = testAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(initializeAccount),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Create garment datas

	const createGarmentDatas = `
import GarmentNFT from 0x5a76b4858ce34b2f

transaction() {

   let adminRef: &GarmentNFT.Admin
   let currGarmentDataID: UInt32

   prepare(acct: AuthAccount) {

       self.currGarmentDataID = GarmentNFT.nextGarmentDataID;
       self.adminRef = acct.borrow<&GarmentNFT.Admin>(from: GarmentNFT.AdminStoragePath)
           ?? panic("No admin resource in storage")
   }

   execute {
       self.adminRef.createGarmentData(
           mainImage: "mainImage1",
           images: ["otherImage1"],
           name: "name1",
           artist: "artist1",
           description: "description1"
       )
   }
}
`

	signerAddress = contractsAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(createGarmentDatas),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Create material datas

	const createMaterialDatas = `
import MaterialNFT from 0x5a76b4858ce34b2f

transaction() {

    let adminRef: &MaterialNFT.Admin
    let currMaterialDataID: UInt32

    prepare(acct: AuthAccount) {

        self.currMaterialDataID = MaterialNFT.nextMaterialDataID;
        self.adminRef = acct.borrow<&MaterialNFT.Admin>(from: MaterialNFT.AdminStoragePath)
            ?? panic("No admin resource in storage")
    }

    execute {
        self.adminRef.createMaterialData(
           mainImage: "mainImage1",
           secondImage: "secondImage1",
           name: "name1",
           description: "description1"
       )
   }
}

`

	signerAddress = contractsAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(createMaterialDatas),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Create item allocations

	const createItemAllocations = `
import ItemNFT from 0x5a76b4858ce34b2f

transaction() {

    let adminRef: &ItemNFT.Admin

    prepare(acct: AuthAccount) {

        self.adminRef = acct.borrow<&ItemNFT.Admin>(from: ItemNFT.AdminStoragePath)
            ?? panic("No admin resource in storage")
    }

    execute {
        self.adminRef.createItemDataAllocation(garmentDataID: 1, materialDataID: 1)
    }
}
`

	signerAddress = contractsAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(createItemAllocations),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Create item datas

	const createItemDatas = `
import ItemNFT from 0x5a76b4858ce34b2f

transaction() {

    let adminRef: &ItemNFT.Admin
    let currItemDataID: UInt32

    prepare(acct: AuthAccount) {

        self.currItemDataID = ItemNFT.nextItemDataID;

        self.adminRef = acct.borrow<&ItemNFT.Admin>(from: ItemNFT.AdminStoragePath)
            ?? panic("No admin resource in storage")
    }

    execute {
        self.adminRef.createItemData(
            mainImage: "mainImage1",
            images: ["image1"],
        )
    }
}

`

	signerAddress = contractsAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(createItemDatas),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Mint garment

	const mintGarment = `
import GarmentNFT from 0x5a76b4858ce34b2f
import FBRC from 0x5a76b4858ce34b2f
import FungibleToken from 0x9a0766d93b6608b7

transaction(recipientAddr: Address, garmentDataID: UInt32, royaltyVaultAddr: Address) {

    let adminRef: &GarmentNFT.Admin

    let royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>

    prepare(acct: AuthAccount) {

        self.adminRef = acct.borrow<&GarmentNFT.Admin>(from: GarmentNFT.AdminStoragePath)
            ?? panic("No admin resource in storage")

        self.royaltyVault = getAccount(royaltyVaultAddr).getCapability<&FBRC.Vault{FungibleToken.Receiver}>(FBRC.CollectionReceiverPath)
    }

    execute {

        // Mint the nft with specific name
        let nft <- self.adminRef.mintNFT(garmentDataID: garmentDataID, royaltyVault: self.royaltyVault)

        let recipient = getAccount(recipientAddr)

        // Get the garment collection capability of the receiver of nft
        let nftReceiver = recipient
            .getCapability(GarmentNFT.CollectionPublicPath)
            .borrow<&{GarmentNFT.GarmentCollectionPublic}>()
            ?? panic("Unable to borrow recipient's garment collection")

        // Deposit the garment
        nftReceiver.deposit(token: <- nft)
    }
}
`

	signerAddress = contractsAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(mintGarment),
			Arguments: [][]byte{
				json.MustEncode(cadence.Address(testAddress)),
				json.MustEncode(cadence.NewUInt32(1)),
				json.MustEncode(cadence.Address(testAddress)),
			},
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Mint material

	const mintMaterial = `
import MaterialNFT from 0x5a76b4858ce34b2f
import FBRC from 0x5a76b4858ce34b2f
import FungibleToken from 0x9a0766d93b6608b7

transaction(recipientAddr: Address, materialDataID: UInt32, royaltyVaultAddr: Address) {

    let adminRef: &MaterialNFT.Admin

    let royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>

    prepare(acct: AuthAccount) {

        self.adminRef = acct.borrow<&MaterialNFT.Admin>(from: MaterialNFT.AdminStoragePath)
            ?? panic("No admin resource in storage")

        self.royaltyVault = getAccount(royaltyVaultAddr).getCapability<&FBRC.Vault{FungibleToken.Receiver}>(FBRC.CollectionReceiverPath)
    }

    execute {

        // Mint the nft with specific name
        let nft <- self.adminRef.mintNFT(materialDataID: materialDataID, royaltyVault: self.royaltyVault)

        let recipient = getAccount(recipientAddr)

        // Get the material collection capability of the receiver of nft
        let nftReceiver = recipient
            .getCapability(MaterialNFT.CollectionPublicPath)
            .borrow<&{MaterialNFT.MaterialCollectionPublic}>()
            ?? panic("Unable to borrow recipient's hat collection")

        // Deposit the material
        nftReceiver.deposit(token: <- nft)
    }
}
    `

	signerAddress = contractsAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(mintMaterial),
			Arguments: [][]byte{
				json.MustEncode(cadence.Address(testAddress)),
				json.MustEncode(cadence.NewUInt32(1)),
				json.MustEncode(cadence.Address(testAddress)),
			},
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Mint item

	const mintItem = `
import NonFungibleToken from 0x631e88ae7f1d7c20
import GarmentNFT from 0x5a76b4858ce34b2f
import MaterialNFT from 0x5a76b4858ce34b2f
import ItemNFT from 0x5a76b4858ce34b2f
import FBRC from 0x5a76b4858ce34b2f
import FungibleToken from 0x9a0766d93b6608b7

transaction(recipientAddr: Address, name: String, garmentWithdrawID: UInt64, materialWithdrawID: UInt64, royaltyVaultAddr: Address) {

     let garment: @NonFungibleToken.NFT
     let material: @NonFungibleToken.NFT
     let royaltyVault: Capability<&FBRC.Vault{FungibleToken.Receiver}>

     prepare(garmentAndMaterialAcct: AuthAccount) {

         // borrow a reference to the owner's garment collection
         let garmentCollectionRef = garmentAndMaterialAcct.borrow<&GarmentNFT.Collection>(from: GarmentNFT.CollectionStoragePath)
             ?? panic("Could not borrow a reference to the stored Garment collection")

         // borrow a reference to the owner's material collection
         let materialCollectionRef = garmentAndMaterialAcct.borrow<&MaterialNFT.Collection>(from: MaterialNFT.CollectionStoragePath)
             ?? panic("Could not borrow a reference to the stored Material collection")

         self.garment <- garmentCollectionRef.withdraw(withdrawID: garmentWithdrawID)

         self.material <- materialCollectionRef.withdraw(withdrawID: materialWithdrawID)

         self.royaltyVault = getAccount(royaltyVaultAddr).getCapability<&FBRC.Vault{FungibleToken.Receiver}>(FBRC.CollectionReceiverPath)

     }

     execute {

         let garmentRef <- self.garment as! @GarmentNFT.NFT

         let materialRef <- self.material as! @MaterialNFT.NFT

         // mint item with the garment and material
         let nft <- ItemNFT.mintNFT(name: name, royaltyVault: self.royaltyVault, garment: <- garmentRef, material: <- materialRef)

         let recipient = getAccount(recipientAddr)

         let nftReceiver = recipient
             .getCapability(ItemNFT.CollectionPublicPath)
             .borrow<&{ItemNFT.ItemCollectionPublic}>()
             ?? panic("Unable to borrow recipient's item collection")

         nftReceiver.deposit(token: <- nft)
     }
}
`

	signerAddress = testAddress

	itemString, err := cadence.NewString("item")
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(mintItem),
			Arguments: [][]byte{
				json.MustEncode(cadence.Address(testAddress)),
				json.MustEncode(itemString),
				json.MustEncode(cadence.NewUInt64(1)),
				json.MustEncode(cadence.NewUInt64(1)),
				json.MustEncode(cadence.Address(testAddress)),
			},
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Get item details

	const getItemDetails = `

    import ItemNFT from 0x5a76b4858ce34b2f
    import GarmentNFT from 0x5a76b4858ce34b2f
    import MaterialNFT from 0x5a76b4858ce34b2f

    pub struct ItemDetails {
        pub let name: String
        pub let serialNumber: UInt32
        pub let numberMintedPerItemDataID: UInt32
        pub let itemDataID: UInt32
        pub let mainImage: String
        pub let images: [String]
        pub let garment: GarmentDetails
        pub let material: MaterialDetails

        init(
            name: String,
            serialNumber: UInt32,
            numberMintedPerItemDataID: UInt32,
            itemDataID: UInt32,
            mainImage: String,
            images: [String],
            garment: GarmentDetails,
            material: MaterialDetails
        ) {
            self.name = name
            self.serialNumber = serialNumber
            self.numberMintedPerItemDataID = numberMintedPerItemDataID
            self.itemDataID = itemDataID
            self.mainImage = mainImage
            self.images = images
            self.garment = garment
            self.material = material
        }
    }

    pub struct GarmentDetails {
        pub let id: UInt64
        pub let serialNumber: UInt32
        pub let numberMintedPerGarmentDataID: UInt32
        pub let garmentDataID: UInt32
        pub let mainImage: String
        pub let images: [String]
        pub let name: String
        pub let artist: String
        pub let description: String

        init(
            id: UInt64,
            serialNumber: UInt32,
            numberMintedPerGarmentDataID: UInt32,
            garmentDataID: UInt32,
            mainImage: String,
            images: [String],
            name: String,
            artist: String,
            description: String
        ) {
            self.id = id
            self.serialNumber = serialNumber
            self.numberMintedPerGarmentDataID = numberMintedPerGarmentDataID
            self.garmentDataID = garmentDataID
            self.mainImage = mainImage
            self.images = images
            self.name = name
            self.artist = artist
            self.description = description
        }
    }

    pub struct MaterialDetails {
         pub let id: UInt64
         pub let serialNumber: UInt32
         pub let numberMintedPerMaterialDataID: UInt32
         pub let materialDataID: UInt32
         pub let mainImage: String
         pub let secondImage: String
         pub let name: String
         pub let description: String

         init(
             id: UInt64,
             serialNumber: UInt32,
             numberMintedPerMaterialDataID: UInt32,
             materialDataID: UInt32,
             mainImage: String,
             secondImage: String,
             name: String,
             description: String
         ) {
             self.id = id
             self.serialNumber = serialNumber
             self.numberMintedPerMaterialDataID = numberMintedPerMaterialDataID
             self.materialDataID = materialDataID
             self.mainImage = mainImage
             self.secondImage = secondImage
             self.name = name
             self.description = description
         }
    }

    pub fun main(account: Address, id: UInt64): ItemDetails {

       let acct = getAccount(account)

       let itemCollectionRef = acct.getCapability(ItemNFT.CollectionPublicPath)
           .borrow<&{ItemNFT.ItemCollectionPublic}>()!

       let item = itemCollectionRef.borrowItem(id: id)!

       let garment = item.borrowGarment()!
       let garmentDataID = garment.garment.garmentDataID
       let garmentData = GarmentNFT.getGarmentData(id: garmentDataID)
       let garmentDetails = GarmentDetails(
           id: garment.id,
           serialNumber: garment.garment.serialNumber,
           numberMintedPerGarmentDataID: GarmentNFT.getGarmentNumberMinted(id: garmentDataID),
           garmentDataID: garmentDataID,
           mainImage: garmentData.mainImage,
           images: garmentData.images,
           name: garmentData.name,
           artist: garmentData.artist,
           description: garmentData.description
       )

       let material = item.borrowMaterial()!
       let materialDataID = material.material.materialDataID
       let materialData = MaterialNFT.getMaterialData(id: materialDataID)
       let materialDetails = MaterialDetails(
           id: material.id,
           serialNumber: material.material.serialNumber,
           numberMintedPerMaterialDataID: MaterialNFT.getMaterialNumberMinted(id: materialDataID),
           materialDataID: materialDataID,
           mainImage: materialData.mainImage,
           secondImage: materialData.secondImage,
           name: materialData.name,
           description: materialData.description
       )

       let itemDataID = item.item.itemDataID
       let itemData = ItemNFT.getItemData(id: itemDataID)
       let itemDetails = ItemDetails(
           name: item.name,
           serialNumber: item.item.serialNumber,
           numberMintedPerItemDataID: ItemNFT.getItemNumberMinted(id: itemDataID),
           itemDataID: itemDataID,
           mainImage: itemData.mainImage,
           images: itemData.images,
           garment: garmentDetails,
           material: materialDetails
       )

       return itemDetails
    }
`

	_, err = runtime.ExecuteScript(
		Script{
			Source: []byte(getItemDetails),
			Arguments: [][]byte{
				json.MustEncode(
					cadence.NewAddress(testAddress),
				),
				json.MustEncode(
					cadence.NewUInt64(1),
				),
			},
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	require.NoError(t, err)
}

func TestRuntimeMissingMemberVersus(t *testing.T) {

	runtime := newTestInterpreterRuntime()

	artistAddress, err := common.HexToAddress("0x1")
	require.NoError(t, err)

	bidderAddress, err := common.HexToAddress("0x2")
	require.NoError(t, err)

	contractsAddress, err := common.HexToAddress("0x99ca04281098b33d")
	require.NoError(t, err)

	ftAddress, err := common.HexToAddress("0x9a0766d93b6608b7")
	require.NoError(t, err)

	flowTokenAddress, err := common.HexToAddress("0x7e60df042a9c0868")
	require.NoError(t, err)

	nftAddress, err := common.HexToAddress("0x631e88ae7f1d7c20")
	require.NoError(t, err)

	const flowTokenContract = `
import FungibleToken from 0x9a0766d93b6608b7

pub contract FlowToken: FungibleToken {

   // Total supply of Flow tokens in existence
   pub var totalSupply: UFix64

   // Event that is emitted when the contract is created
   pub event TokensInitialized(initialSupply: UFix64)

   // Event that is emitted when tokens are withdrawn from a Vault
   pub event TokensWithdrawn(amount: UFix64, from: Address?)

   // Event that is emitted when tokens are deposited to a Vault
   pub event TokensDeposited(amount: UFix64, to: Address?)

   // Event that is emitted when new tokens are minted
   pub event TokensMinted(amount: UFix64)

   // Event that is emitted when tokens are destroyed
   pub event TokensBurned(amount: UFix64)

   // Event that is emitted when a new minter resource is created
   pub event MinterCreated(allowedAmount: UFix64)

   // Event that is emitted when a new burner resource is created
   pub event BurnerCreated()

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
       pub var balance: UFix64

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
       pub fun withdraw(amount: UFix64): @FungibleToken.Vault {
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
       pub fun deposit(from: @FungibleToken.Vault) {
           let vault <- from as! @FlowToken.Vault
           self.balance = self.balance + vault.balance
           emit TokensDeposited(amount: vault.balance, to: self.owner?.address)
           vault.balance = 0.0
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
       return <-create Vault(balance: 0.0)
   }

   pub resource Administrator {
       // createNewMinter
       //
       // Function that creates and returns a new minter resource
       //
       pub fun createNewMinter(allowedAmount: UFix64): @Minter {
           emit MinterCreated(allowedAmount: allowedAmount)
           return <-create Minter(allowedAmount: allowedAmount)
       }

       // createNewBurner
       //
       // Function that creates and returns a new burner resource
       //
       pub fun createNewBurner(): @Burner {
           emit BurnerCreated()
           return <-create Burner()
       }
   }

   // Minter
   //
   // Resource object that token admin accounts can hold to mint new tokens.
   //
   pub resource Minter {

       // the amount of tokens that the minter is allowed to mint
       pub var allowedAmount: UFix64

       // mintTokens
       //
       // Function that mints new tokens, adds them to the total supply,
       // and returns them to the calling context.
       //
       pub fun mintTokens(amount: UFix64): @FlowToken.Vault {
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
           emit TokensBurned(amount: amount)
       }
   }

   init(adminAccount: AuthAccount) {
       self.totalSupply = 0.0

       // Create the Vault with the total supply of tokens and save it in storage
       //
       let vault <- create Vault(balance: self.totalSupply)
       adminAccount.save(<-vault, to: /storage/flowTokenVault)

       // Create a public capability to the stored Vault that only exposes
       // the deposit method through the Receiver interface
       //
       adminAccount.link<&FlowToken.Vault{FungibleToken.Receiver}>(
           /public/flowTokenReceiver,
           target: /storage/flowTokenVault
       )

       // Create a public capability to the stored Vault that only exposes
       // the balance field through the Balance interface
       //
       adminAccount.link<&FlowToken.Vault{FungibleToken.Balance}>(
           /public/flowTokenBalance,
           target: /storage/flowTokenVault
       )

       let admin <- create Administrator()
       adminAccount.save(<-admin, to: /storage/flowTokenAdmin)

       // Emit an event that shows that the contract was initialized
       emit TokensInitialized(initialSupply: self.totalSupply)
   }
}

`

	const auctionDutchContract = `
import NonFungibleToken from 0x631e88ae7f1d7c20
import FungibleToken from 0x9a0766d93b6608b7
import FlowToken from 0x7e60df042a9c0868
// import Debug from 0x99ca04281098b33d
// import Clock from 0x99ca04281098b33d

pub contract AuctionDutch {

	pub let CollectionStoragePath: StoragePath
	pub let CollectionPublicPath: PublicPath

	pub let BidCollectionStoragePath: StoragePath
	pub let BidCollectionPublicPath: PublicPath

	pub event AuctionDutchBidRejected(bidder: Address)
	pub event AuctionDutchCreated(name: String, artist: String, number: Int, owner:Address, id: UInt64)

	pub event AuctionDutchBid(amount: UFix64, bidder: Address, auction: UInt64, bid: UInt64)
	pub event AuctionDutchBidIncreased(amount: UFix64, bidder: Address, auction: UInt64, bid: UInt64)
	pub event AuctionDutchTick(tickPrice: UFix64, acceptedBids: Int, totalItems: Int, tickTime: UFix64, auction: UInt64)
	pub event AuctionDutchSettle(price: UFix64, auction: UInt64)

	pub struct Bids {
		pub let bids: [BidReport]
		pub let winningPrice: UFix64?

		init(bids: [BidReport], winningPrice: UFix64?) {
			self.bids =bids
			self.winningPrice=winningPrice
		}
	}

	pub struct BidReport {
		pub let id: UInt64
		pub let time: UFix64
		pub let amount: UFix64
		pub let bidder: Address
		pub let winning: Bool
		pub let confirmed: Bool

		init(id: UInt64, time: UFix64, amount: UFix64, bidder: Address, winning: Bool, confirmed: Bool) {
			self.id=id
			self.time=time
			self.amount=amount
			self.bidder=bidder
			self.winning=winning
			self.confirmed=confirmed
		}
	}

	pub struct BidInfo {
		access(contract) let id: UInt64
		access(contract) let vaultCap: Capability<&{FungibleToken.Receiver}>
		access(contract) let nftCap: Capability<&{NonFungibleToken.Receiver}>
		access(contract) var time: UFix64
		access(contract) var balance: UFix64
		access(contract) var winning: Bool


		init(id: UInt64, nftCap: Capability<&{NonFungibleToken.Receiver}>, vaultCap: Capability<&{FungibleToken.Receiver}>, time: UFix64, balance: UFix64) {
			self.id=id
			self.nftCap= nftCap
			self.vaultCap=vaultCap
			self.time=time
			self.balance=balance
			self.winning=false
		}

		pub fun increaseBid(_ amount:UFix64) {
			self.balance=self.balance+amount
			self.time = 42.0 // Clock.time()
		}

		access(contract) fun  withdraw(_ amount: UFix64) {
			self.balance=self.balance - amount
		}

		pub fun setWinning(_ value: Bool) {
			self.winning=value
		}
	}

	pub struct Tick {
		pub let price: UFix64
		pub let startedAt: UFix64

		init(price: UFix64, startedAt: UFix64) {
			self.price=price
			self.startedAt=startedAt
		}
	}

	pub struct TickStatus{
		pub let price: UFix64
		pub let startedAt: UFix64
		pub let acceptedBids: Int
		pub let cumulativeAcceptedBids: Int

		init(price: UFix64, startedAt: UFix64, acceptedBids:Int, cumulativeAcceptedBids:Int) {
			self.price=price
			self.startedAt=startedAt
			self.acceptedBids=acceptedBids
			self.cumulativeAcceptedBids=cumulativeAcceptedBids
		}
	}

	pub resource Auction {
		access(contract) let nfts: @{UInt64:NonFungibleToken.NFT}

		access(contract) let metadata: {String:String}

		// bids are put into buckets based on the tick they are in.
		// tick 1 will be the first tick,

		//this is a counter to keep the number of bids so that we can escrow in a separate resource
		access(contract) var totalBids: UInt64

		//this has to be an array I think, since we need ordering.
		access(contract) let ticks: [Tick]

		access(contract) let auctionStatus: {UFix64: TickStatus}
		access(contract) var currentTickIndex: UInt64

		//this is a lookup table for the bid
		access(contract) let bidInfo: {UInt64: BidInfo}

		access(contract) let winningBids: [UInt64]

		//this is a table of ticks to ordered list of bid ids
		access(contract) let bids: {UFix64: [UInt64]}

		access(contract) let escrow: @{UInt64: FlowToken.Vault}

		//todo store bids here?
		access(contract) let ownerVaultCap: Capability<&{FungibleToken.Receiver}>
		access(contract) let ownerNFTCap: Capability<&{NonFungibleToken.Receiver}>
		access(contract) let royaltyVaultCap: Capability<&{FungibleToken.Receiver}>
		access(contract) let royaltyPercentage: UFix64
		access(contract) let numberOfItems: Int
		access(contract) var winningBid: UFix64?


		init(nfts: @{UInt64 : NonFungibleToken.NFT},
		metadata: {String: String},
		ownerVaultCap: Capability<&{FungibleToken.Receiver}>,
		ownerNFTCap: Capability<&{NonFungibleToken.Receiver}>,
		royaltyVaultCap: Capability<&{FungibleToken.Receiver}>,
		royaltyPercentage: UFix64,
		ticks: [Tick]) {
			self.metadata=metadata
			self.totalBids=1
			self.currentTickIndex=0
			self.numberOfItems=nfts.length
			self.ticks=ticks
			self.auctionStatus={}
			self.winningBids=[]
			//create the ticks
			self.nfts <- nfts
			self.winningBid=nil
			var emptyBids : {UFix64: [UInt64]}={}
			for tick in ticks {
				emptyBids[tick.startedAt]=[]
			}
			self.bids = emptyBids
			self.bidInfo= {}
			self.escrow <- {}
			self.ownerVaultCap=ownerVaultCap
			self.ownerNFTCap=ownerNFTCap
			self.royaltyVaultCap=royaltyVaultCap
			self.royaltyPercentage=royaltyPercentage
		}

		pub fun startAt() : UFix64 {
			return self.ticks[0].startedAt
		}

		access(contract) fun fulfill() {
			if self.winningBid== nil {
				// Debug.log("Winning price is not set")
				panic("Cannot fulfill is not finished")
			}

			let nftIds= self.nfts.keys

			for id in self.winningBids {
				let bid= self.bidInfo[id]!
				if let vault <- self.escrow[bid.id] <- nil {
					if vault.balance > self.winningBid! {
						self.ownerVaultCap.borrow()!.deposit(from: <- vault.withdraw(amount: vault.balance-self.winningBid!))
					}
					if self.royaltyPercentage != 0.0 {
						self.royaltyVaultCap.borrow()!.deposit(from: <- vault.withdraw(amount: vault.balance*self.royaltyPercentage))
					}

					self.ownerVaultCap.borrow()!.deposit(from: <- vault)

					let nftId=nftIds.removeFirst()
					if let nft <- self.nfts[nftId] <- nil {
						//TODO: here we might consider adding the nftId that you have won to BidInfo and let the user pull it out
						self.bidInfo[bid.id]!.nftCap.borrow()!.deposit(token: <- nft)
					}
				}
			}
			/*
			//let just return all other money here and fix the issue with gas later
			//this will blow the gas limit on high number of bids
			for tick in self.ticks {
				if let bids=self.bids[tick.startedAt]{
					for bidId in bids {
						let bid= self.bidInfo[bidId]!
						if let vault <- self.escrow[bidId] <- nil {
							//TODO: check that it is still linked
							bid.vaultCap.borrow()!.deposit(from: <- vault)
						}
					}
				}
			}
			*/

			emit AuctionDutchSettle(price: self.winningBid!, auction: self.uuid)
		}

		pub fun getBids() : Bids {
			var bids: [BidReport] =[]
			var numberWinning=0
			var winningBid=self.winningBid
			for tick in self.ticks {
				let localBids=self.bids[tick.startedAt]!
				for bid in localBids {
					let bidInfo= self.bidInfo[bid]!
					var winning=bidInfo.winning
					//we have an ongoing auction
					if self.winningBid == nil && numberWinning != self.numberOfItems {
						winning=true
						numberWinning=numberWinning+1
						if numberWinning== self.numberOfItems {
							winningBid=bidInfo.balance
						}
					}
					bids.append(BidReport(id: bid, time: bidInfo.time, amount: bidInfo.balance, bidder: bidInfo.vaultCap.address, winning: winning, confirmed:bidInfo.winning))
				}
			}
			return Bids(bids: bids, winningPrice: winningBid)
		}

		pub fun findWinners() : [UInt64] {

			var bids: [UInt64] =[]
			for tick in self.ticks {
				if bids.length == self.numberOfItems {
					return bids
				}
				let localBids=self.bids[tick.startedAt]!
				if bids.length+localBids.length <= self.numberOfItems {
					bids.appendAll(localBids)
					//we have to remove the bids
					self.bids.remove(key: tick.startedAt)
				} else {
					while bids.length < self.numberOfItems {
						bids.append(localBids.removeFirst())
					}
				}
			}
			return bids
		}

		pub fun getTick() : Tick {
			return self.ticks[self.currentTickIndex]
		}

		//this should be called something else
		pub fun isAuctionFinished() : Bool {

			if !self.isLastTick() {
				//if the startedAt of the next tick is larger then current time not time to tick yet
				let time = 42.0 // Clock.time()
				let nextTickStartAt= self.ticks[self.currentTickIndex+1].startedAt
				// Debug.log("We are not on last tick current tick is "
				//.concat(self.currentTickIndex.toString())
				//.concat(" time=").concat(time.toString())
				//.concat(" nextTickStart=").concat(nextTickStartAt.toString()))
				if  nextTickStartAt > time {
					return false
				}

			}
			//Debug.log("we are on or after next tick")

			//TODO: need to figure out what will happen if this is the last tick
			let tick= self.getTick()

			//calculate number of acceptedBids
			let bids=self.bids[tick.startedAt]!

			let previousAcceptedBids=self.winningBids.length
			var winning=true
			for bid in bids {

				let bidInfo= self.bidInfo[bid]!
				//we do not have enough winning bids so we add this bid as a winning bid
				if self.winningBids.length < self.numberOfItems {
					self.winningBids.append(bid)
					//if we now have enough bids we need to set the winning bid
					if self.winningBids.length == self.numberOfItems {
						self.winningBid=bidInfo.balance
					}
				}

				//Debug.log("Processing bid ".concat(bid.toString()).concat(" total accepted bids are ").concat(self.winningBids.length.toString()))

				self.bidInfo[bid]!.setWinning(winning)

				if self.winningBids.length == self.numberOfItems {
					winning=false
				}
			}

			//lets advance the tick
			self.currentTickIndex=self.currentTickIndex+1

			if self.winningBids.length == self.numberOfItems {
				//this could be done later, but i will just do it here for ease of reading
				self.auctionStatus[tick.startedAt] = TickStatus(price:tick.price, startedAt: tick.startedAt, acceptedBids: self.numberOfItems - previousAcceptedBids, cumulativeAcceptedBids: self.numberOfItems)
				log(self.auctionStatus)
				return true
			}

			self.auctionStatus[tick.startedAt] = TickStatus(price:tick.price, startedAt: tick.startedAt, acceptedBids: bids.length, cumulativeAcceptedBids: self.winningBids.length)
			log(self.auctionStatus)
			return false
		}

		pub fun isLastTick() : Bool {
			let tickLength = UInt64(self.ticks.length-1)
			return self.currentTickIndex==tickLength
		}

		// taken from bisect_right in  pthon https://stackoverflow.com/questions/2945017/javas-equivalent-to-bisect-in-python
		pub fun bisect(items: [UInt64], new: BidInfo) : Int {
			var high=items.length
			var low=0
			while low < high {
				let mid =(low+high)/2
				let midBidId=items[mid]
				let midBid=self.bidInfo[midBidId]!
				if midBid.balance < new.balance || midBid.balance==new.balance && midBid.id > new.id {
					high=mid
				} else {
					low=mid+1
				}
			}
			return low
		}

		priv fun insertBid(_ bid: BidInfo) {
			for tick in self.ticks {
				if tick.price > bid.balance {
					continue
				}

				//add the bid to the lookup table
				self.bidInfo[bid.id]=bid

				let bucket= self.bids[tick.startedAt]!
				//find the index of the new bid in the ordred bucket bid list
				let index= self.bisect(items:bucket, new: bid)

				//insert bid and mutate state
				bucket.insert(at: index, bid.id)
				self.bids[tick.startedAt]= bucket

				emit AuctionDutchBid(amount: bid.balance, bidder: bid.nftCap.address, auction: self.uuid, bid: bid.id)
				return
			}
		}

		pub fun findTickForBid(_ id:UInt64) : Tick {
			for tick in self.ticks {
				let bucket= self.bids[tick.startedAt]!
				if bucket.contains(id) {
					return tick
				}
			}
			panic("Could not find bid")
		}

		pub fun removeBidFromTick(_ id:UInt64, tick: UFix64) {
			var index=0
			let bids= self.bids[tick]!
			while index < bids.length {
				if bids[index] == id {
					bids.remove(at: index)
					self.bids[tick]=bids
					return
				}
				index=index+1
			}
		}

		access(contract) fun  cancelBid(id: UInt64) {
			pre {
				self.bidInfo[id] != nil: "bid info does not exist"
				!self.bidInfo[id]!.winning : "bid is already accepted"
				self.escrow[id] != nil: "escrow for bid does not exist"
			}

			let bidInfo=self.bidInfo[id]!

			if let escrowVault <- self.escrow[id] <- nil {
				let oldTick=self.findTickForBid(id)
				self.removeBidFromTick(id, tick: oldTick.startedAt)
				self.bidInfo.remove(key: id)
				bidInfo.vaultCap.borrow()!.deposit(from: <- escrowVault)
			}
		}

		access(self) fun findTickForAmount(_ amount: UFix64) : Tick{
			for t in self.ticks {
				if t.price > amount {
					continue
				}
				return t
			}
			panic("Could not find tick for amount")
		}

		access(contract) fun getExcessBalance(_ id: UInt64) : UFix64 {
			let bid=self.bidInfo[id]!
			if self.winningBid != nil {
				//if we are done and you are a winning bid you will already have gotten your flow back in fullfillment
				if !bid.winning {
					return bid.balance
				}
			} else {
				if bid.balance > self.calculatePrice()  {
					return bid.balance - self.calculatePrice()
				}
			}
			return 0.0
		}

		access(contract) fun withdrawExcessFlow(id: UInt64, cap: Capability<&{FungibleToken.Receiver}>)  {
			let balance= self.getExcessBalance(id)
			if balance == 0.0 {
				return
			}

			let bid=self.bidInfo[id]!
			if let escrowVault <- self.escrow[id] <- nil {
				bid.withdraw(balance)
				let withdrawVault= cap.borrow()!
				if escrowVault.balance == balance {
					withdrawVault.deposit(from: <- escrowVault)
				} else {
					let tmpVault <- escrowVault.withdraw(amount: balance)
					withdrawVault.deposit(from: <- tmpVault)
					let oldVault <- self.escrow[id] <- escrowVault
					destroy oldVault
				}
				self.bidInfo[id]=bid
			}
		}

		access(contract) fun getBidInfo(id: UInt64) : BidInfo {
			return self.bidInfo[id]!
		}

		access(contract) fun  increaseBid(id: UInt64, vault: @FlowToken.Vault) {
			pre {
				self.bidInfo[id] != nil: "bid info doesn not exist"
				!self.bidInfo[id]!.winning : "bid is already accepted"
				self.escrow[id] != nil: "escrow for bid does not exist"
			}

		  let bidInfo=self.bidInfo[id]!
			if let escrowVault <- self.escrow[id] <- nil {
				bidInfo.increaseBid(vault.balance)
				escrowVault.deposit(from: <- vault)
				self.bidInfo[id]=bidInfo
				let oldVault <- self.escrow[id] <- escrowVault
				destroy oldVault


				var tick=self.findTickForBid(id)
				self.removeBidFromTick(id, tick: tick.startedAt)
				if tick.price < bidInfo.balance {
					tick=self.findTickForAmount(bidInfo.balance)
				}
				let bucket= self.bids[tick.startedAt]!
				//find the index of the new bid in the ordred bucket bid list
				let index= self.bisect(items:bucket, new: bidInfo)

				//insert bid and mutate state
				bucket.insert(at: index, bidInfo.id)
				self.bids[tick.startedAt]= bucket

				//todo do we need separate bid for increase?
				emit AuctionDutchBidIncreased(amount: bidInfo.balance, bidder: bidInfo.nftCap.address, auction: self.uuid, bid: bidInfo.id)
			} else {
				destroy vault
				panic("Cannot get escrow")
			}
			//need to check if the bid is in the correct bucket now
			//emit event
		}

		pub fun addBid(vault: @FlowToken.Vault, nftCap: Capability<&{NonFungibleToken.Receiver}>, vaultCap: Capability<&{FungibleToken.Receiver}>, time: UFix64) : UInt64{

			let bidId=self.totalBids

			let bid=BidInfo(id: bidId, nftCap: nftCap, vaultCap:vaultCap, time: time, balance: vault.balance)
			self.insertBid(bid)
			let oldEscrow <- self.escrow[bidId] <- vault
			self.totalBids=self.totalBids+(1 as UInt64)
			destroy oldEscrow
			return bid.id
		}

		pub fun calculatePrice() : UFix64{
			return self.ticks[self.currentTickIndex].price
		}

		destroy() {
			//TODO: deposity to ownerNFTCap
			destroy self.nfts
			//todo transfer back
			destroy self.escrow
		}
	}

	pub resource interface Public {
		pub fun getIds() : [UInt64]
		//TODO: can we just join these two?
		pub fun getStatus(_ id: UInt64) : AuctionDutchStatus
		pub fun getBids(_ id: UInt64) : Bids
		//these methods are only allowed to be called from within this contract, but we want to call them on another users resource
		access(contract) fun getAuction(_ id:UInt64) : &Auction
		pub fun bid(id: UInt64, vault: @FungibleToken.Vault, vaultCap: Capability<&{FungibleToken.Receiver}>, nftCap: Capability<&{NonFungibleToken.Receiver}>) : @Bid
	}


	pub struct AuctionDutchStatus {

		pub let status: String
		pub let startTime: UFix64
		pub let currentTime: UFix64
		pub let currentPrice: UFix64
		pub let totalItems: Int
		pub let acceptedBids: Int
		pub let tickStatus: {UFix64:TickStatus}
		pub let metadata: {String:String}

		init(status:String, currentPrice: UFix64, totalItems: Int, acceptedBids:Int,  startTime: UFix64, tickStatus: {UFix64:TickStatus}, metadata: {String:String}){
			self.status=status
			self.currentPrice=currentPrice
			self.totalItems=totalItems
			self.acceptedBids=acceptedBids
			self.startTime=startTime
			self.currentTime= 42.0 // Clock.time()
			self.tickStatus=tickStatus
			self.metadata=metadata
		}
	}

	pub resource Collection: Public {

		//TODO: what to do with ended auctions? put them in another collection?
		//NFTS are gone but we might want to keep some information about it?

		pub let auctions: @{UInt64: Auction}

		init() {
			self.auctions <- {}
		}

		pub fun getIds() : [UInt64] {
			return self.auctions.keys
		}

		pub fun getStatus(_ id: UInt64) : AuctionDutchStatus{
			let item= self.getAuction(id)
			let currentTime= 42.0 // Clock.time()

			var status="Ongoing"
			var currentPrice= item.calculatePrice()
			if currentTime < item.startAt() {
				status="NotStarted"
			} else if item.winningBid != nil {
				status="Finished"
				currentPrice=item.winningBid!
			}


			return AuctionDutchStatus(status: status,
			currentPrice: currentPrice,
			totalItems: item.numberOfItems,
			acceptedBids: item.winningBids.length,
			startTime: item.startAt(),
			tickStatus: item.auctionStatus,
			metadata:item.metadata)
		}

		pub fun getBids(_ id:UInt64) : Bids {
			pre {
				self.auctions[id] != nil: "auction doesn't exist"
			}

			let item= self.getAuction(id)
			return item.getBids()
		}

		access(contract) fun getAuction(_ id:UInt64) : &Auction {
			pre {
				self.auctions[id] != nil: "auction doesn't exist"
			}
			return (&self.auctions[id] as &Auction?)!
		}

		pub fun bid(id: UInt64, vault: @FungibleToken.Vault, vaultCap: Capability<&{FungibleToken.Receiver}>, nftCap: Capability<&{NonFungibleToken.Receiver}>) : @Bid{
			//TODO: pre id should exist

			let time= 42.0 // Clock.time()
			let vault <- vault as! @FlowToken.Vault
			let auction=self.getAuction(id)

			let price=auction.calculatePrice()

			//the currentPrice is still higher then your bid, this is find we just add your bid to the correct tick bucket
			if price > vault.balance {
				let bidId =auction.addBid(vault: <- vault, nftCap:nftCap, vaultCap: vaultCap, time: time)
				return <- create Bid(capability: AuctionDutch.account.getCapability<&Collection{Public}>(AuctionDutch.CollectionPublicPath), auctionId: id, bidId: bidId)
			}

			let tooMuchCash=vault.balance - price
			//you sent in too much flow when you bid so we return some to you and add a valid accepted bid
			if tooMuchCash != 0.0 {
				vaultCap.borrow()!.deposit(from: <- vault.withdraw(amount: tooMuchCash))
			}

			let bidId=auction.addBid(vault: <- vault, nftCap:nftCap, vaultCap: vaultCap, time: time)
			return <- create Bid(capability: AuctionDutch.account.getCapability<&Collection{Public}>(AuctionDutch.CollectionPublicPath), auctionId: id, bidId: bidId)
		}

		pub fun tickOrFulfill(_ id:UInt64) {
			let time= 42.0 // Clock.time()
			let auction=self.getAuction(id)

			if !auction.isAuctionFinished() {
				let tick=auction.getTick()
				//TODO: this emits a tick even even if we do not tick
				emit AuctionDutchTick(tickPrice: tick.price, acceptedBids: auction.winningBids.length, totalItems: auction.numberOfItems, tickTime: tick.startedAt, auction: id)
				return
			}

			auction.fulfill()
		}


		pub fun createAuction( nfts: @{UInt64: NonFungibleToken.NFT}, metadata: {String: String}, startAt: UFix64, startPrice: UFix64, floorPrice: UFix64, decreasePriceFactor: UFix64, decreasePriceAmount: UFix64, tickDuration: UFix64, ownerVaultCap: Capability<&{FungibleToken.Receiver}>, ownerNFTCap: Capability<&{NonFungibleToken.Receiver}>, royaltyVaultCap: Capability<&{FungibleToken.Receiver}>, royaltyPercentage: UFix64) {

			let ticks: [Tick] = [Tick(price: startPrice, startedAt: startAt)]
			var currentPrice=startPrice
			var currentStartAt=startAt
			while(currentPrice > floorPrice) {
				currentPrice=currentPrice * decreasePriceFactor - decreasePriceAmount
				if currentPrice < floorPrice {
					currentPrice=floorPrice
				}
				currentStartAt=currentStartAt+tickDuration
				ticks.append(Tick(price: currentPrice, startedAt:currentStartAt))
			}

			let length=nfts.keys.length

			let auction <- create Auction(nfts: <- nfts, metadata: metadata, ownerVaultCap:ownerVaultCap, ownerNFTCap:ownerNFTCap, royaltyVaultCap:royaltyVaultCap, royaltyPercentage: royaltyPercentage, ticks: ticks)

			emit AuctionDutchCreated(name: metadata["name"] ?? "Unknown name", artist: metadata["artist"] ?? "Unknown artist",  number: length, owner: ownerVaultCap.address, id: auction.uuid)

			let oldAuction <- self.auctions[auction.uuid] <- auction
			destroy oldAuction
		}

		destroy () {
			destroy self.auctions
		}

	}

	pub fun getBids(_ id: UInt64) : Bids {
		let account = AuctionDutch.account
		let cap=account.getCapability<&Collection{Public}>(self.CollectionPublicPath)
		if let collection = cap.borrow() {
			return collection.getBids(id)
		}
		panic("Could not find auction capability")
	}

	pub fun getAuctionDutch(_ id: UInt64) : AuctionDutchStatus? {
		let account = AuctionDutch.account
		let cap=account.getCapability<&Collection{Public}>(self.CollectionPublicPath)
		if let collection = cap.borrow() {
			return collection.getStatus(id)
		}
		return nil
	}

	pub resource Bid {

		pub let capability:Capability<&Collection{Public}>
		pub let auctionId: UInt64
		pub let bidId: UInt64

		init(capability:Capability<&Collection{Public}>, auctionId: UInt64, bidId:UInt64) {
			self.capability=capability
			self.auctionId=auctionId
			self.bidId=bidId
		}

		pub fun getBidInfo() : BidInfo {
			return self.capability.borrow()!.getAuction(self.auctionId).getBidInfo(id: self.bidId)
		}

		pub fun getExcessBalance() : UFix64 {
			return self.capability.borrow()!.getAuction(self.auctionId).getExcessBalance(self.bidId)
		}

		pub fun increaseBid(vault: @FlowToken.Vault) {
			self.capability.borrow()!.getAuction(self.auctionId).increaseBid(id: self.bidId, vault: <- vault)
		}

		pub fun cancelBid() {
			self.capability.borrow()!.getAuction(self.auctionId).cancelBid(id: self.bidId)
		}

		pub fun withdrawExcessFlow(_ cap: Capability<&{FungibleToken.Receiver}>) {
			self.capability.borrow()!.getAuction(self.auctionId).withdrawExcessFlow(id: self.bidId, cap:cap)
		}
	}

	pub struct ExcessFlowReport {
		pub let id: UInt64
		pub let winning: Bool //TODO: should this be confirmed winning?
		pub let excessAmount: UFix64

		init(id: UInt64, report: BidInfo, excessAmount: UFix64) {
			self.id=id
			self.winning=report.winning
			self.excessAmount=excessAmount
		}
	}

	pub resource interface BidCollectionPublic {
		pub fun bid(marketplace: Address, id: UInt64, vault: @FungibleToken.Vault, vaultCap: Capability<&{FungibleToken.Receiver}>, nftCap: Capability<&{NonFungibleToken.Receiver}>)
		pub fun getIds() :[UInt64]
		pub fun getReport(_ id: UInt64) : ExcessFlowReport

	}

	pub resource BidCollection:BidCollectionPublic {

		access(contract) let bids : @{UInt64: Bid}

		init() {
			self.bids <- {}
		}

		pub fun getIds() : [UInt64] {
			return self.bids.keys
		}

		pub fun getReport(_ id: UInt64) : ExcessFlowReport {
			let bid=self.getBid(id)
			return ExcessFlowReport(id:id, report: bid.getBidInfo(), excessAmount: bid.getExcessBalance())
		}

		pub fun bid(marketplace: Address, id: UInt64, vault: @FungibleToken.Vault, vaultCap: Capability<&{FungibleToken.Receiver}>, nftCap: Capability<&{NonFungibleToken.Receiver}>)  {

			let dutchAuctionCap=getAccount(marketplace).getCapability<&AuctionDutch.Collection{AuctionDutch.Public}>(AuctionDutch.CollectionPublicPath)
			let bid <- dutchAuctionCap.borrow()!.bid(id: id, vault: <- vault, vaultCap: vaultCap, nftCap: nftCap)
			self.bids[bid.uuid] <-! bid
		}

		pub fun withdrawExcessFlow(id: UInt64, vaultCap: Capability<&{FungibleToken.Receiver}>) {
			let bid = self.getBid(id)
			bid.withdrawExcessFlow(vaultCap)
		}

		pub fun cancelBid(_ id: UInt64) {
			let bid = self.getBid(id)
			bid.cancelBid()
			destroy <- self.bids.remove(key: bid.uuid)
		}

		pub fun increaseBid(_ id: UInt64, vault: @FungibleToken.Vault) {
			let vault <- vault as! @FlowToken.Vault
			let bid = self.getBid(id)
			bid.increaseBid(vault: <- vault)
		}

		access(contract) fun getBid(_ id:UInt64) : &Bid {
			pre {
				self.bids[id] != nil: "bid doesn't exist"
			}
			return (&self.bids[id] as &Bid?)!
		}


		destroy() {
			destroy  self.bids
		}

	}

	pub fun createEmptyBidCollection() : @BidCollection {
		return <- create BidCollection()
	}

	init() {
		self.CollectionPublicPath= /public/versusAuctionDutchCollection
		self.CollectionStoragePath= /storage/versusAuctionDutchCollection

		self.BidCollectionPublicPath= /public/versusAuctionDutchBidCollection
		self.BidCollectionStoragePath= /storage/versusAuctionDutchBidCollection


		let account=self.account
		let collection <- create Collection()
		account.save(<-collection, to: AuctionDutch.CollectionStoragePath)
		account.link<&Collection{Public}>(AuctionDutch.CollectionPublicPath, target: AuctionDutch.CollectionStoragePath)

	}
}
`
	accountCodes := map[common.LocationID][]byte{
		common.AddressLocation{
			Address: ftAddress,
			Name:    "FungibleToken",
		}.ID(): []byte(realFungibleTokenContractInterface),
		common.AddressLocation{
			Address: nftAddress,
			Name:    "NonFungibleToken",
		}.ID(): []byte(realNonFungibleTokenInterface),
	}

	var events []cadence.Event

	var signerAddress common.Address

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location.ID()], nil
		},
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location.ID()], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location.ID()] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			println(message)
		},
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy FlowToken contract

	signerAddress = flowTokenAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(fmt.Sprintf(
				`
                 transaction {

                     prepare(signer: AuthAccount) {
                         signer.contracts.add(name: "FlowToken", code: "%s".decodeHex(), signer)
                     }
                 }
               `,
				hex.EncodeToString([]byte(flowTokenContract)),
			)),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Deploy contracts

	signerAddress = contractsAddress

	for _, contract := range []struct {
		name string
		code string
	}{
		{"AuctionDutch", auctionDutchContract},
	} {

		err = runtime.ExecuteTransaction(
			Script{
				Source: utils.DeploymentTransaction(
					contract.name,
					[]byte(contract.code),
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	}

	// Setup accounts for Flow Token and mint tokens

	const setupFlowTokenAccountTransaction = `
import FungibleToken from 0x9a0766d93b6608b7
import FlowToken from 0x7e60df042a9c0868

transaction {

    prepare(signer: AuthAccount) {

        if signer.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault) == nil {
            // Create a new flowToken Vault and put it in storage
            signer.save(<-FlowToken.createEmptyVault(), to: /storage/flowTokenVault)

            // Create a public capability to the Vault that only exposes
            // the deposit function through the Receiver interface
            signer.link<&FlowToken.Vault{FungibleToken.Receiver}>(
                /public/flowTokenReceiver,
                target: /storage/flowTokenVault
            )

            // Create a public capability to the Vault that only exposes
            // the balance field through the Balance interface
            signer.link<&FlowToken.Vault{FungibleToken.Balance}>(
                /public/flowTokenBalance,
                target: /storage/flowTokenVault
            )
        }
    }
}
`

	mintAmount, err := cadence.NewUFix64("1000.0")
	require.NoError(t, err)

	const mintTransaction = `
import FungibleToken from 0x9a0766d93b6608b7
import FlowToken from 0x7e60df042a9c0868

transaction(recipient: Address, amount: UFix64) {
    let tokenAdmin: &FlowToken.Administrator
    let tokenReceiver: &{FungibleToken.Receiver}

    prepare(signer: AuthAccount) {
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

	for _, address := range []common.Address{
		contractsAddress,
		artistAddress,
		bidderAddress,
	} {
		// Setup account

		signerAddress = address

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(setupFlowTokenAccountTransaction),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Mint tokens

		signerAddress = flowTokenAddress

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(mintTransaction),
				Arguments: encodeArgs([]cadence.Value{
					cadence.Address(address),
					mintAmount,
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	}

	// Create auction

	const artCollectionTransaction = `
	    import FungibleToken from 0x9a0766d93b6608b7
	    import NonFungibleToken from 0x631e88ae7f1d7c20
	    import AuctionDutch from 0x99ca04281098b33d

	    transaction() {
	        prepare(account: AuthAccount) {

                let ownerVaultCap = account.getCapability<&{FungibleToken.Receiver}>(/public/doesNotExist)
		        let ownerNFTCap = account.getCapability<&{NonFungibleToken.Receiver}>(/public/doesNotExist)
		        let royaltyVaultCap = account.getCapability<&{FungibleToken.Receiver}>(/public/doesNotExist)

                account.borrow<&AuctionDutch.Collection>(from: AuctionDutch.CollectionStoragePath)!
                    .createAuction(
                        nfts: <-{},
                        metadata: {},
                        startAt: 42.0,
                        startPrice: 4.0,
                        floorPrice: 2.0,
                        decreasePriceFactor: 0.1,
                        decreasePriceAmount: 0.1,
                        tickDuration: 0.1,
                        ownerVaultCap: ownerVaultCap,
                        ownerNFTCap: ownerNFTCap,
                        royaltyVaultCap: royaltyVaultCap,
                        royaltyPercentage: 0.1
                    )
	        }
	    }
	`

	signerAddress = contractsAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(artCollectionTransaction),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Bid

	const bidTransaction = `
	    import FlowToken from 0x7e60df042a9c0868
	    import FungibleToken from 0x9a0766d93b6608b7
	    import NonFungibleToken from 0x631e88ae7f1d7c20
	    import AuctionDutch from 0x99ca04281098b33d

	    transaction {
            prepare(signer: AuthAccount) {

                let vault <- signer.borrow<&FlowToken.Vault>(from: /storage/flowTokenVault)!
                    .withdraw(amount: 4.0)

                let vaultCap = signer.getCapability<&{FungibleToken.Receiver}>(/public/flowTokenReceiver)
                let nftCap = signer.getCapability<&{NonFungibleToken.Receiver}>(/public/doesNotExist)

                let bid <- getAccount(0x99ca04281098b33d)
                    .getCapability<&AuctionDutch.Collection{AuctionDutch.Public}>(AuctionDutch.CollectionPublicPath)
                    .borrow()!
                    .bid(
                       id: 0,
                       vault: <-vault,
                       vaultCap: vaultCap,
                       nftCap: nftCap
                    )

                signer.save(<-bid, to: /storage/bid)
            }
	    }
	`

	signerAddress = bidderAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(bidTransaction),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Cancel bid

	const cancelBidTransaction = `
	    import AuctionDutch from 0x99ca04281098b33d

	    transaction {
            prepare(signer: AuthAccount) {
                signer.borrow<&AuctionDutch.Bid>(from: /storage/bid)!.cancelBid()
            }
	    }
	`

	signerAddress = bidderAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(cancelBidTransaction),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeMissingMemberExampleMarketplace(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	exampleTokenAddress, err := common.HexToAddress("0x1")
	require.NoError(t, err)

	exampleNFTAddress, err := common.HexToAddress("0x2")
	require.NoError(t, err)

	exampleMarketplaceAddress, err := common.HexToAddress("0x3")
	require.NoError(t, err)

	const exampleTokenContract = `
	// ExampleToken.cdc
//
// The ExampleToken contract is a sample implementation of a fungible token on Flow.
//
// Fungible tokens behave like everyday currencies -- they can be minted, transferred or
// traded for digital goods.
//
// Follow the fungible tokens tutorial to learn more: https://docs.onflow.org/docs/fungible-tokens
//
// This is a basic implementation of a Fungible Token and is NOT meant to be used in production
// See the Flow Fungible Token standard for real examples: https://github.com/onflow/flow-ft

pub contract ExampleToken {

    // Total supply of all tokens in existence.
    pub var totalSupply: UFix64

    // Provider
    //
    // Interface that enforces the requirements for withdrawing
    // tokens from the implementing type.
    //
    // We don't enforce requirements on self.balance here because
    // it leaves open the possibility of creating custom providers
    // that don't necessarily need their own balance.
    //
    pub resource interface Provider {

        // withdraw
        //
        // Function that subtracts tokens from the owner's Vault
        // and returns a Vault resource (@Vault) with the removed tokens.
        //
        // The function's access level is public, but this isn't a problem
        // because even the public functions are not fully public at first.
        // anyone in the network can call them, but only if the owner grants
        // them access by publishing a resource that exposes the withdraw
        // function.
        //
        pub fun withdraw(amount: UFix64): @Vault {
            post {
                // result refers to the return value of the function
                result.balance == UFix64(amount):
                    "Withdrawal amount must be the same as the balance of the withdrawn Vault"
            }
        }
    }

    // Receiver
    //
    // Interface that enforces the requirements for depositing
    // tokens into the implementing type.
    //
    // We don't include a condition that checks the balance because
    // we want to give users the ability to make custom Receivers that
    // can do custom things with the tokens, like split them up and
    // send them to different places.
    //
	pub resource interface Receiver {
        // deposit
        //
        // Function that can be called to deposit tokens
        // into the implementing resource type
        //
        pub fun deposit(from: @Vault) {
            pre {
                from.balance > 0.0:
                    "Deposit balance must be positive"
            }
        }
    }

    // Balance
    //
    // Interface that specifies a public balance field for the vault
    //
    pub resource interface Balance {
        pub var balance: UFix64
    }

    // Vault
    //
    // Each user stores an instance of only the Vault in their storage
    // The functions in the Vault and governed by the pre and post conditions
    // in the interfaces when they are called.
    // The checks happen at runtime whenever a function is called.
    //
    // Resources can only be created in the context of the contract that they
    // are defined in, so there is no way for a malicious user to create Vaults
    // out of thin air. A special Minter resource needs to be defined to mint
    // new tokens.
    //
    pub resource Vault: Provider, Receiver, Balance {

		// keeps track of the total balance of the account's tokens
        pub var balance: UFix64

        // initialize the balance at resource creation time
        init(balance: UFix64) {
            self.balance = balance
        }

        // withdraw
        //
        // Function that takes an integer amount as an argument
        // and withdraws that amount from the Vault.
        //
        // It creates a new temporary Vault that is used to hold
        // the money that is being transferred. It returns the newly
        // created Vault to the context that called so it can be deposited
        // elsewhere.
        //
        pub fun withdraw(amount: UFix64): @Vault {
            self.balance = self.balance - amount
            return <-create Vault(balance: amount)
        }

        // deposit
        //
        // Function that takes a Vault object as an argument and adds
        // its balance to the balance of the owners Vault.
        //
        // It is allowed to destroy the sent Vault because the Vault
        // was a temporary holder of the tokens. The Vault's balance has
        // been consumed and therefore can be destroyed.
        pub fun deposit(from: @Vault) {
            self.balance = self.balance + from.balance
            destroy from
        }
    }

    // createEmptyVault
    //
    // Function that creates a new Vault with a balance of zero
    // and returns it to the calling context. A user must call this function
    // and store the returned Vault in their storage in order to allow their
    // account to be able to receive deposits of this token type.
    //
    pub fun createEmptyVault(): @Vault {
        return <-create Vault(balance: 0.0)
    }

	// VaultMinter
    //
    // Resource object that an admin can control to mint new tokens
    pub resource VaultMinter {

		// Function that mints new tokens and deposits into an account's vault
		// using their Receiver reference.
        pub fun mintTokens(amount: UFix64, recipient: Capability<&AnyResource{Receiver}>) {
            let recipientRef = recipient.borrow()
                ?? panic("Could not borrow a receiver reference to the vault")

            ExampleToken.totalSupply = ExampleToken.totalSupply + UFix64(amount)
            recipientRef.deposit(from: <-create Vault(balance: amount))
        }
    }

    // The init function for the contract. All fields in the contract must
    // be initialized at deployment. This is just an example of what
    // an implementation could do in the init function. The numbers are arbitrary.
    init() {
        self.totalSupply = 30.0

        let vault <- create Vault(balance: self.totalSupply)
        self.account.save(<-vault, to: /storage/CadenceFungibleTokenTutorialVault)

        // Create a new MintAndBurn resource and store it in account storage
        self.account.save(<-create VaultMinter(), to: /storage/CadenceFungibleTokenTutorialMinter)

        // Create a private capability link for the Minter
        // Capabilities can be used to create temporary references to an object
        // so that callers can use the reference to access fields and functions
        // of the objet.
        //
        // The capability is stored in the /private/ domain, which is only
        // accesible by the owner of the account
        self.account.link<&VaultMinter>(/private/Minter, target: /storage/CadenceFungibleTokenTutorialMinter)
    }
}
	`

	const exampleNFTContract = `
	// ExampleNFT.cdc
//
// This is a complete version of the ExampleNFT contract
// that includes withdraw and deposit functionality, as well as a
// collection resource that can be used to bundle NFTs together.
//
// It also includes a definition for the Minter resource,
// which can be used by admins to mint new NFTs.
//
// Learn more about non-fungible tokens in this tutorial: https://docs.onflow.org/docs/non-fungible-tokens

pub contract ExampleNFT {

    // Declare Path constants so paths do not have to be hardcoded
    // in transactions and scripts

    pub let CollectionStoragePath: StoragePath
    pub let CollectionPublicPath: PublicPath
    pub let MinterStoragePath: StoragePath

    // Declare the NFT resource type
    pub resource NFT {
        // The unique ID that differentiates each NFT
        pub let id: UInt64

        // Initialize both fields in the init function
        init(initID: UInt64) {
            self.id = initID
        }
    }

    // We define this interface purely as a way to allow users
    // to create public, restricted references to their NFT Collection.
    // They would use this to publicly expose only the deposit, getIDs,
    // and idExists fields in their Collection
    pub resource interface NFTReceiver {

        pub fun deposit(token: @NFT)

        pub fun getIDs(): [UInt64]

        pub fun idExists(id: UInt64): Bool
    }

    // The definition of the Collection resource that
    // holds the NFTs that a user owns
    pub resource Collection: NFTReceiver {
        // dictionary of NFT conforming tokens
        pub var ownedNFTs: @{UInt64: NFT}

        // Initialize the NFTs field to an empty collection
        init () {
            self.ownedNFTs <- {}
        }

        // withdraw 
        //
        // Function that removes an NFT from the collection 
        // and moves it to the calling context
        pub fun withdraw(withdrawID: UInt64): @NFT {
            // If the NFT isn't found, the transaction panics and reverts
            let token <- self.ownedNFTs.remove(key: withdrawID)!

            return <-token
        }

        pub fun getReference(id: UInt64): &NFT {
            return (&self.ownedNFTs[id] as &NFT?)!
        }

        // deposit 
        //
        // Function that takes a NFT as an argument and 
        // adds it to the collections dictionary
        pub fun deposit(token: @NFT) {
            // add the new token to the dictionary with a force assignment
            // if there is already a value at that key, it will fail and revert
            self.ownedNFTs[token.id] <-! token
        }

        // idExists checks to see if a NFT 
        // with the given ID exists in the collection
        pub fun idExists(id: UInt64): Bool {
            return self.ownedNFTs[id] != nil
        }

        // getIDs returns an array of the IDs that are in the collection
        pub fun getIDs(): [UInt64] {
            return self.ownedNFTs.keys
        }

        destroy() {
            destroy self.ownedNFTs
        }
    }

    // creates a new empty Collection resource and returns it 
    pub fun createEmptyCollection(): @Collection {
        return <- create Collection()
    }

    // NFTMinter
    //
    // Resource that would be owned by an admin or by a smart contract 
    // that allows them to mint new NFTs when needed
    pub resource NFTMinter {

        // the ID that is used to mint NFTs
        // it is only incremented so that NFT ids remain
        // unique. It also keeps track of the total number of NFTs
        // in existence
        pub var idCount: UInt64

        init() {
            self.idCount = 1
        }

        // mintNFT 
        //
        // Function that mints a new NFT with a new ID
        // and returns it to the caller
        pub fun mintNFT(): @NFT {

            // create a new NFT
            var newNFT <- create NFT(initID: self.idCount)

            // change the id so that each ID is unique
            self.idCount = self.idCount + 1
            
            return <-newNFT
        }
    }

	init() {
        self.CollectionStoragePath = /storage/nftTutorialCollection
        self.CollectionPublicPath = /public/nftTutorialCollection
        self.MinterStoragePath = /storage/nftTutorialMinter

		// store an empty NFT Collection in account storage
        self.account.save(<-self.createEmptyCollection(), to: self.CollectionStoragePath)

        // publish a reference to the Collection in storage
        self.account.link<&{NFTReceiver}>(self.CollectionPublicPath, target: self.CollectionStoragePath)

        // store a minter resource in account storage
        self.account.save(<-create NFTMinter(), to: self.MinterStoragePath)
	}
}
	
	`

	const exampleMarketplaceContract = `
	import ExampleToken from 0x01
import ExampleNFT from 0x02

// ExampleMarketplace.cdc
//
// The ExampleMarketplace contract is a very basic sample implementation of an NFT ExampleMarketplace on Flow.
//
// This contract allows users to put their NFTs up for sale. Other users
// can purchase these NFTs with fungible tokens.
//
// Learn more about marketplaces in this tutorial: https://docs.onflow.org/cadence/tutorial/06-marketplace-compose/
//
// This contract is a learning tool and is not meant to be used in production.
// See the NFTStorefront contract for a generic marketplace smart contract that 
// is used by many different projects on the Flow blockchain:
//
// https://github.com/onflow/nft-storefront

pub contract ExampleMarketplace {

    // Event that is emitted when a new NFT is put up for sale
    pub event ForSale(id: UInt64, price: UFix64, owner: Address?)

    // Event that is emitted when the price of an NFT changes
    pub event PriceChanged(id: UInt64, newPrice: UFix64, owner: Address?)

    // Event that is emitted when a token is purchased
    pub event TokenPurchased(id: UInt64, price: UFix64, seller: Address?, buyer: Address?)

    // Event that is emitted when a seller withdraws their NFT from the sale
    pub event SaleCanceled(id: UInt64, seller: Address?)

    // Interface that users will publish for their Sale collection
    // that only exposes the methods that are supposed to be public
    //
    pub resource interface SalePublic {
        pub fun purchase(tokenID: UInt64, recipient: Capability<&AnyResource{ExampleNFT.NFTReceiver}>, buyTokens: @ExampleToken.Vault)
        pub fun idPrice(tokenID: UInt64): UFix64?
        pub fun getIDs(): [UInt64]
    }

    // SaleCollection
    //
    // NFT Collection object that allows a user to put their NFT up for sale
    // where others can send fungible tokens to purchase it
    //
    pub resource SaleCollection: SalePublic {

        /// A capability for the owner's collection
        access(self) var ownerCollection: Capability<&ExampleNFT.Collection>

        // Dictionary of the prices for each NFT by ID
        access(self) var prices: {UInt64: UFix64}

        // The fungible token vault of the owner of this sale.
        // When someone buys a token, this resource can deposit
        // tokens into their account.
        access(account) let ownerVault: Capability<&AnyResource{ExampleToken.Receiver}>

        init (ownerCollection: Capability<&ExampleNFT.Collection>, 
              ownerVault: Capability<&AnyResource{ExampleToken.Receiver}>) {

            pre {
                // Check that the owner's collection capability is correct
                ownerCollection.check(): 
                    "Owner's NFT Collection Capability is invalid!"

                // Check that the fungible token vault capability is correct
                ownerVault.check(): 
                    "Owner's Receiver Capability is invalid!"
            }
            self.ownerCollection = ownerCollection
            self.ownerVault = ownerVault
            self.prices = {}
        }

        // cancelSale gives the owner the opportunity to cancel a sale in the collection
        pub fun cancelSale(tokenID: UInt64) {
            // remove the price
            self.prices.remove(key: tokenID)
            self.prices[tokenID] = nil

            // Nothing needs to be done with the actual token because it is already in the owner's collection
        }

        // listForSale lists an NFT for sale in this collection
        pub fun listForSale(tokenID: UInt64, price: UFix64) {
            pre {
                self.ownerCollection.borrow()!.idExists(id: tokenID):
                    "NFT to be listed does not exist in the owner's collection"
            }
            // store the price in the price array
            self.prices[tokenID] = price

            emit ForSale(id: tokenID, price: price, owner: self.owner?.address)
        }

        // changePrice changes the price of a token that is currently for sale
        pub fun changePrice(tokenID: UInt64, newPrice: UFix64) {
            self.prices[tokenID] = newPrice

            emit PriceChanged(id: tokenID, newPrice: newPrice, owner: self.owner?.address)
        }

        // purchase lets a user send tokens to purchase an NFT that is for sale
        pub fun purchase(tokenID: UInt64, recipient: Capability<&AnyResource{ExampleNFT.NFTReceiver}>, buyTokens: @ExampleToken.Vault) {
            pre {
                self.prices[tokenID] != nil:
                    "No token matching this ID for sale!"
                buyTokens.balance >= (self.prices[tokenID] ?? 0.0):
                    "Not enough tokens to by the NFT!"
                recipient.borrow != nil:
                    "Invalid NFT receiver capability!"
            }

            // get the value out of the optional
            let price = self.prices[tokenID]!

            self.prices[tokenID] = nil

            let vaultRef = self.ownerVault.borrow()
                ?? panic("Could not borrow reference to owner token vault")

            // deposit the purchasing tokens into the owners vault
            vaultRef.deposit(from: <-buyTokens)

            // borrow a reference to the object that the receiver capability links to
            // We can force-cast the result here because it has already been checked in the pre-conditions
            let receiverReference = recipient.borrow()!

            let nftRef = self.ownerCollection.borrow()!.getReference(id: tokenID)
            log("NFT Reference before transfer:")
            log(nftRef)

            // deposit the NFT into the buyers collection
            receiverReference.deposit(token: <- self.ownerCollection.borrow()!.withdraw(withdrawID: tokenID))

            log("NFT Reference after transfer:")
            log(nftRef)
            log(nftRef.id)

            emit TokenPurchased(id: tokenID, price: price, seller: self.owner?.address, buyer: receiverReference.owner?.address)
        }

        // idPrice returns the price of a specific token in the sale
        pub fun idPrice(tokenID: UInt64): UFix64? {
            return self.prices[tokenID]
        }

        // getIDs returns an array of token IDs that are for sale
        pub fun getIDs(): [UInt64] {
            return self.prices.keys
        }
    }

    // createCollection returns a new collection resource to the caller
    pub fun createSaleCollection(ownerCollection: Capability<&ExampleNFT.Collection>, 
                                 ownerVault: Capability<&AnyResource{ExampleToken.Receiver}>): @SaleCollection {
        return <- create SaleCollection(ownerCollection: ownerCollection, ownerVault: ownerVault)
    }
}

	`

	accountCodes := map[common.LocationID][]byte{}
	var events []cadence.Event
	var logs []string
	var signerAddress common.Address

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location.ID()], nil
		},
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location.ID()], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location.ID()] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			logs = append(logs, message)
		},
	}

	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy contracts

	for _, contract := range []struct {
		name   string
		code   string
		signer Address
	}{
		{"ExampleToken", exampleTokenContract, exampleTokenAddress},
		{"ExampleNFT", exampleNFTContract, exampleNFTAddress},
		{"ExampleMarketplace", exampleMarketplaceContract, exampleMarketplaceAddress},
	} {

		signerAddress = contract.signer

		err = runtime.ExecuteTransaction(
			Script{
				Source: utils.DeploymentTransaction(
					contract.name,
					[]byte(contract.code),
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	}

	// Run transactions

	const setupAccount1Tx = `
	// SetupAccount1Transaction.cdc

import ExampleToken from 0x01
import ExampleNFT from 0x02

// This transaction sets up account 0x01 for the marketplace tutorial
// by publishing a Vault reference and creating an empty NFT Collection.
transaction {
  prepare(acct: AuthAccount) {
    // Create a public Receiver capability to the Vault
    acct.link<&ExampleToken.Vault{ExampleToken.Receiver, ExampleToken.Balance}>
             (/public/CadenceFungibleTokenTutorialReceiver, target: /storage/CadenceFungibleTokenTutorialVault)

    log("Created Vault references")

    // store an empty NFT Collection in account storage
    acct.save<@ExampleNFT.Collection>(<-ExampleNFT.createEmptyCollection(), to: /storage/nftTutorialCollection)

    // publish a capability to the Collection in storage
    acct.link<&{ExampleNFT.NFTReceiver}>(ExampleNFT.CollectionPublicPath, target: ExampleNFT.CollectionStoragePath)

    log("Created a new empty collection and published a reference")
  }
}

	`
	const setupAccount2Tx = `
	// SetupAccount2Transaction.cdc

import ExampleToken from 0x01
import ExampleNFT from 0x02

// This transaction adds an empty Vault to account 0x02
// and mints an NFT with id=1 that is deposited into
// the NFT collection on account 0x01.
transaction {

  // Private reference to this account's minter resource
  let minterRef: &ExampleNFT.NFTMinter

  prepare(acct: AuthAccount) {
    // create a new vault instance with an initial balance of 30
    let vaultA <- ExampleToken.createEmptyVault()

    // Store the vault in the account storage
    acct.save<@ExampleToken.Vault>(<-vaultA, to: /storage/CadenceFungibleTokenTutorialVault)

    // Create a public Receiver capability to the Vault
    let ReceiverRef = acct.link<&ExampleToken.Vault{ExampleToken.Receiver, ExampleToken.Balance}>(/public/CadenceFungibleTokenTutorialReceiver, target: /storage/CadenceFungibleTokenTutorialVault)

    log("Created a Vault and published a reference")

    // Borrow a reference for the NFTMinter in storage
    self.minterRef = acct.borrow<&ExampleNFT.NFTMinter>(from: ExampleNFT.MinterStoragePath)
        ?? panic("Could not borrow owner's NFT minter reference")
  }
  execute {
    // Get the recipient's public account object
    let recipient = getAccount(0x01)

    // Get the Collection reference for the receiver
    // getting the public capability and borrowing a reference from it
    let receiverRef = recipient.getCapability(ExampleNFT.CollectionPublicPath)
                               .borrow<&{ExampleNFT.NFTReceiver}>()
                               ?? panic("Could not borrow nft receiver reference")

    // Mint an NFT and deposit it into account 0x01's collection
    receiverRef.deposit(token: <-self.minterRef.mintNFT())

    log("New NFT minted for account 1")
  }
}

	`
	const mintTokensTx = `
	// SetupAccount1TransactionMinting.cdc

import ExampleToken from 0x01
import ExampleNFT from 0x02

// This transaction mints tokens for both accounts using
// the minter stored on account 0x01.
transaction {

  // Public Vault Receiver References for both accounts
  let acct1Capability: Capability<&AnyResource{ExampleToken.Receiver}>
  let acct2Capability: Capability<&AnyResource{ExampleToken.Receiver}>

  // Private minter references for this account to mint tokens
  let minterRef: &ExampleToken.VaultMinter

  prepare(acct: AuthAccount) {
    // Get the public object for account 0x02
    let account2 = getAccount(0x02)

    // Retrieve public Vault Receiver references for both accounts
    self.acct1Capability = acct.getCapability<&AnyResource{ExampleToken.Receiver}>(/public/CadenceFungibleTokenTutorialReceiver)

    self.acct2Capability = account2.getCapability<&AnyResource{ExampleToken.Receiver}>(/public/CadenceFungibleTokenTutorialReceiver)

    // Get the stored Minter reference for account 0x01
    self.minterRef = acct.borrow<&ExampleToken.VaultMinter>(from: /storage/CadenceFungibleTokenTutorialMinter)
        ?? panic("Could not borrow owner's vault minter reference")
  }

  execute {
    // Mint tokens for both accounts
    self.minterRef.mintTokens(amount: 20.0, recipient: self.acct2Capability)
    self.minterRef.mintTokens(amount: 10.0, recipient: self.acct1Capability)

    log("Minted new fungible tokens for account 1 and 2")
  }
}

	`
	const createSaleTx = `
	// CreateSale.cdc

import ExampleToken from 0x01
import ExampleNFT from 0x02
import ExampleMarketplace from 0x03

// This transaction creates a new Sale Collection object,
// lists an NFT for sale, puts it in account storage,
// and creates a public capability to the sale so that others can buy the token.
transaction {

    prepare(acct: AuthAccount) {

        // Borrow a reference to the stored Vault
        let receiver = acct.getCapability<&{ExampleToken.Receiver}>(/public/CadenceFungibleTokenTutorialReceiver)

        // borrow a reference to the nftTutorialCollection in storage
        let collectionCapability = acct.link<&ExampleNFT.Collection>(/private/nftTutorialCollection, target: ExampleNFT.CollectionStoragePath)
          ?? panic("Unable to create private link to NFT Collection")

        // Create a new Sale object,
        // initializing it with the reference to the owner's vault
        let sale <- ExampleMarketplace.createSaleCollection(ownerCollection: collectionCapability, ownerVault: receiver)

        // List the token for sale by moving it into the sale object
        sale.listForSale(tokenID: 1, price: 10.0)

        // Store the sale object in the account storage
        acct.save(<-sale, to: /storage/NFTSale)

        // Create a public capability to the sale so that others can call its methods
        acct.link<&ExampleMarketplace.SaleCollection{ExampleMarketplace.SalePublic}>(/public/NFTSale, target: /storage/NFTSale)

        log("Sale Created for account 1. Selling NFT 1 for 10 tokens")
    }
}


	`
	const purchaseTx = `
	// PurchaseSale.cdc

import ExampleToken from 0x01
import ExampleNFT from 0x02
import ExampleMarketplace from 0x03

// This transaction uses the signers Vault tokens to purchase an NFT
// from the Sale collection of account 0x01.
transaction {

    // Capability to the buyer's NFT collection where they
    // will store the bought NFT
    let collectionCapability: Capability<&AnyResource{ExampleNFT.NFTReceiver}>

    // Vault that will hold the tokens that will be used to
    // but the NFT
    let temporaryVault: @ExampleToken.Vault

    prepare(acct: AuthAccount) {

        // get the references to the buyer's fungible token Vault and NFT Collection Receiver
        self.collectionCapability = acct.getCapability<&AnyResource{ExampleNFT.NFTReceiver}>(ExampleNFT.CollectionPublicPath)

        let vaultRef = acct.borrow<&ExampleToken.Vault>(from: /storage/CadenceFungibleTokenTutorialVault)
            ?? panic("Could not borrow owner's vault reference")

        // withdraw tokens from the buyers Vault
        self.temporaryVault <- vaultRef.withdraw(amount: 10.0)
    }

    execute {
        // get the read-only account storage of the seller
        let seller = getAccount(0x01)

        // get the reference to the seller's sale
        let saleRef = seller.getCapability(/public/NFTSale)
                            .borrow<&AnyResource{ExampleMarketplace.SalePublic}>()
                            ?? panic("Could not borrow seller's sale reference")

        // purchase the NFT the the seller is selling, giving them the capability
        // to your NFT collection and giving them the tokens to buy it
        saleRef.purchase(tokenID: 1, recipient: self.collectionCapability, buyTokens: <-self.temporaryVault)

        log("Token 1 has been bought by account 2!")
    }
}
	`

	for _, tx := range []struct {
		code   string
		signer Address
	}{
		{setupAccount1Tx, exampleTokenAddress},
		{setupAccount2Tx, exampleNFTAddress},
		{mintTokensTx, exampleTokenAddress},
		{createSaleTx, exampleTokenAddress},
		{purchaseTx, exampleTokenAddress},
	} {
		signerAddress = tx.signer

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(tx.code),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	}
}
