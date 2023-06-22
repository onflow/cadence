/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

const realNonFungibleTokenInterface = `

access(all) contract interface NonFungibleToken {

    // The total number of tokens of this type in existence
    access(all) var totalSupply: UInt64

    // Event that emitted when the NFT contract is initialized
    //
    access(all) event ContractInitialized()

    // Event that is emitted when a token is withdrawn,
    // indicating the owner of the collection that it was withdrawn from.
    //
    access(all) event Withdraw(id: UInt64, from: Address?)

    // Event that emitted when a token is deposited to a collection.
    //
    // It indicates the owner of the collection that it was deposited to.
    //
    access(all) event Deposit(id: UInt64, to: Address?)

    // Interface that the NFTs have to conform to
    //
    access(all) resource interface INFT {
        // The unique ID that each NFT has
        access(all) let id: UInt64
    }

    // Requirement that all conforming NFT smart contracts have
    // to define a resource called NFT that conforms to INFT
    access(all) resource NFT: INFT {
        access(all) let id: UInt64
    }

    // Interface to mediate withdraws from the Collection
    //
    access(all) resource interface Provider {
        // withdraw removes an NFT from the collection and moves it to the caller
        access(all) fun withdraw(withdrawID: UInt64): @NFT {
            post {
                result.id == withdrawID: "The ID of the withdrawn token must be the same as the requested ID"
            }
        }
    }

    // Interface to mediate deposits to the Collection
    //
    access(all) resource interface Receiver {

        // deposit takes an NFT as an argument and adds it to the Collection
        //
        access(all) fun deposit(token: @NFT)
    }

    // Interface that an account would commonly
    // publish for their collection
    access(all) resource interface CollectionPublic {
        access(all) fun deposit(token: @NFT)
        access(all) fun getIDs(): [UInt64]
        access(all) fun borrowNFT(id: UInt64): &NFT
    }

    // Requirement for the the concrete resource type
    // to be declared in the implementing contract
    //
    access(all) resource Collection: Provider, Receiver, CollectionPublic {

        // Dictionary to hold the NFTs in the Collection
        access(all) var ownedNFTs: @{UInt64: NFT}

        // withdraw removes an NFT from the collection and moves it to the caller
        access(all) fun withdraw(withdrawID: UInt64): @NFT

        // deposit takes a NFT and adds it to the collections dictionary
        // and adds the ID to the id array
        access(all) fun deposit(token: @NFT)

        // getIDs returns an array of the IDs that are in the collection
        access(all) fun getIDs(): [UInt64]

        // Returns a borrowed reference to an NFT in the collection
        // so that the caller can read data and call methods from it
        access(all) fun borrowNFT(id: UInt64): &NFT {
            pre {
                self.ownedNFTs[id] != nil: "NFT does not exist in the collection!"
            }
        }
    }

    // createEmptyCollection creates an empty Collection
    // and returns it to the caller so that they can own NFTs
    access(all) fun createEmptyCollection(): @Collection {
        post {
            result.ownedNFTs.length == 0: "The created collection must be empty!"
        }
    }
}
`
const realTopShotContract = `
import NonFungibleToken from 0x1d7e57aa55817448

access(all) contract TopShot: NonFungibleToken {

    // -----------------------------------------------------------------------
    // TopShot contract Event definitions
    // -----------------------------------------------------------------------

    // emitted when the TopShot contract is created
    access(all) event ContractInitialized()

    // emitted when a new Play struct is created
    access(all) event PlayCreated(id: UInt32, metadata: {String:String})
    // emitted when a new series has been triggered by an admin
    access(all) event NewSeriesStarted(newCurrentSeries: UInt32)

    // Events for Set-Related actions
    //
    // emitted when a new Set is created
    access(all) event SetCreated(setID: UInt32, series: UInt32)
    // emitted when a new play is added to a set
    access(all) event PlayAddedToSet(setID: UInt32, playID: UInt32)
    // emitted when a play is retired from a set and cannot be used to mint
    access(all) event PlayRetiredFromSet(setID: UInt32, playID: UInt32, numMoments: UInt32)
    // emitted when a set is locked, meaning plays cannot be added
    access(all) event SetLocked(setID: UInt32)
    // emitted when a moment is minted from a set
    access(all) event MomentMinted(momentID: UInt64, playID: UInt32, setID: UInt32, serialNumber: UInt32)

    // events for Collection-related actions
    //
    // emitted when a moment is withdrawn from a collection
    access(all) event Withdraw(id: UInt64, from: Address?)
    // emitted when a moment is deposited into a collection
    access(all) event Deposit(id: UInt64, to: Address?)

    // emitted when a moment is destroyed
    access(all) event MomentDestroyed(id: UInt64)

    // -----------------------------------------------------------------------
    // TopShot contract-level fields
    // These contain actual values that are stored in the smart contract
    // -----------------------------------------------------------------------

    // Series that this set belongs to
    // Series is a concept that indicates a group of sets through time
    // Many sets can exist at a time, but only one series
    access(all) var currentSeries: UInt32

    // variable size dictionary of Play structs
    access(self) var playDatas: {UInt32: Play}

    // variable size dictionary of SetData structs
    access(self) var setDatas: {UInt32: SetData}

    // variable size dictionary of Set resources
    access(self) var sets: @{UInt32: Set}

    // the ID that is used to create Plays.
    // Every time a Play is created, playID is assigned
    // to the new Play's ID and then is incremented by 1.
    access(all) var nextPlayID: UInt32

    // the ID that is used to create Sets. Every time a Set is created
    // setID is assigned to the new set's ID and then is incremented by 1.
    access(all) var nextSetID: UInt32

    // the total number of Top shot moment NFTs that have been created
    // Because NFTs can be destroyed, it doesn't necessarily mean that this
    // reflects the total number of NFTs in existence, just the number that
    // have been minted to date.
    // Is also used as global moment IDs for minting
    access(all) var totalSupply: UInt64

    // -----------------------------------------------------------------------
    // TopShot contract-level Composite Type DEFINITIONS
    // -----------------------------------------------------------------------
    // These are just definitions for types that this contract
    // and other accounts can use. These definitions do not contain
    // actual stored values, but an instance (or object) of one of these types
    // can be created by this contract that contains stored values
    // -----------------------------------------------------------------------

    // Play is a Struct that holds metadata associated
    // with a specific NBA play, like the legendary moment when
    // Ray Allen hit the 3 to tie the Heat and Spurs in the 2013 finals game 6
    // or when Lance Stephenson blew in the ear of Lebron James
    //
    // Moment NFTs will all reference a single Play as the owner of
    // its metadata. The Plays are publicly accessible, so anyone can
    // read the metadata associated with a specific play ID
    //
    access(all) struct Play {

        // the unique ID that the Play has
        access(all) let playID: UInt32

        // Stores all the metadata about the Play as a string mapping
        // This is not the long term way we will do metadata. Just a temporary
        // construct while we figure out a better way to do metadata
        //
        access(all) let metadata: {String: String}

        init(metadata: {String: String}) {
            pre {
                metadata.length != 0: "New Play Metadata cannot be empty"
            }
            self.playID = TopShot.nextPlayID
            self.metadata = metadata

            // increment the ID so that it isn't used again
            TopShot.nextPlayID = TopShot.nextPlayID + UInt32(1)

            emit PlayCreated(id: self.playID, metadata: metadata)
        }
    }

    // A Set is a grouping of plays that have occurred in the real world
    // that make up a related group of collectibles, like sets of baseball
    // or Magic cards.
    //
    // SetData is a struct that is stored in a public field of the contract.
    // This is to allow anyone to be able to query the constant information
    // about a set but not have the ability to modify any data in the
    // private set resource
    //
    access(all) struct SetData {

        // unique ID for the set
        access(all) let setID: UInt32

        // Name of the Set
        // ex. "Times when the Toronto Raptors choked in the playoffs"
        access(all) let name: String

        // Series that this set belongs to
        // Series is a concept that indicates a group of sets through time
        // Many sets can exist at a time, but only one series
        access(all) let series: UInt32

        init(name: String) {
            pre {
                name.length > 0: "New Set name cannot be empty"
            }
            self.setID = TopShot.nextSetID
            self.name = name
            self.series = TopShot.currentSeries

            // increment the setID so that it isn't used again
            TopShot.nextSetID = TopShot.nextSetID + UInt32(1)

            emit SetCreated(setID: self.setID, series: self.series)
        }
    }

    // Set is a resource type that contains the functions to add and remove
    // plays from a set and mint moments.
    //
    // It is stored in a private field in the contract so that
    // the admin resource can call its methods and that there can be
    // public getters for some of its fields
    //
    // The admin can add Plays to a set so that the set can mint moments
    // that reference that playdata.
    // The moments that are minted by a set will be listed as belonging to
    // the set that minted it, as well as the Play it references
    //
    // The admin can also retire plays from the set, meaning that the retired
    // play can no longer have moments minted from it.
    //
    // If the admin locks the Set, then no more plays can be added to it, but
    // moments can still be minted.
    //
    // If retireAll() and lock() are called back to back,
    // the Set is closed off forever and nothing more can be done with it
    access(all) resource Set {

        // unique ID for the set
        access(all) let setID: UInt32

        // Array of plays that are a part of this set
        // When a play is added to the set, its ID gets appended here
        // The ID does not get removed from this array when a play is retired
        access(all) var plays: [UInt32]

        // Indicates if a play in this set can be minted
        // A play is set to false when it is added to a set
        // to indicate that it is still active
        // When the play is retired, this is set to true and cannot be changed
        access(all) var retired: {UInt32: Bool}

        // Indicates if the set is currently locked
        // When a set is created, it is unlocked
        // and plays are allowed to be added to it
        // When a set is locked, plays cannot be added
        // A set can never be changed from locked to unlocked
        // The decision to lock it is final
        // If a set is locked, plays cannot be added, but
        // moments can still be minted from plays
        // that already had been added to it.
        access(all) var locked: Bool

        // Indicates the number of moments
        // that have been minted per play in this set
        // When a moment is minted, this value is stored in the moment to
        // show where in the play set it is so far. ex. 13 of 60
        access(all) var numberMintedPerPlay: {UInt32: UInt32}

        init(name: String) {
            self.setID = TopShot.nextSetID
            self.plays = []
            self.retired = {}
            self.locked = false
            self.numberMintedPerPlay = {}

            // Create a new SetData for this Set and store it in contract storage
            TopShot.setDatas[self.setID] = SetData(name: name)
        }

        // addPlay adds a play to the set
        //
        // Parameters: playID: The ID of the play that is being added
        //
        // Pre-Conditions:
        // The play needs to be an existing play
        // The set needs to be not locked
        // The play can't have already been added to the set
        //
        access(all) fun addPlay(playID: UInt32) {
            pre {
                TopShot.playDatas[playID] != nil: "Cannot add the Play to Set: Play doesn't exist"
                !self.locked: "Cannot add the play to the Set after the set has been locked"
                self.numberMintedPerPlay[playID] == nil: "The play has already beed added to the set"
            }

            // Add the play to the array of plays
            self.plays.append(playID)

            // Open the play up for minting
            self.retired[playID] = false

            // Initialize the moment count to zero
            self.numberMintedPerPlay[playID] = 0

            emit PlayAddedToSet(setID: self.setID, playID: playID)
        }

        // addPlays adds multiple plays to the set
        //
        // Parameters: playIDs: The IDs of the plays that are being added
        //                      as an array
        //
        access(all) fun addPlays(playIDs: [UInt32]) {
            for play in playIDs {
                self.addPlay(playID: play)
            }
        }

        // retirePlay retires a play from the set so that it can't mint new moments
        //
        // Parameters: playID: The ID of the play that is being retired
        //
        // Pre-Conditions:
        // The play needs to be an existing play that is currently open for minting
        //
        access(all) fun retirePlay(playID: UInt32) {
            pre {
                self.retired[playID] != nil: "Cannot retire the Play: Play doesn't exist in this set!"
            }

            if !self.retired[playID]! {
                self.retired[playID] = true

                emit PlayRetiredFromSet(setID: self.setID, playID: playID, numMoments: self.numberMintedPerPlay[playID]!)
            }
        }

        // retireAll retires all the plays in the set
        // Afterwards, none of the retired plays will be able to mint new moments
        //
        access(all) fun retireAll() {
            for play in self.plays {
                self.retirePlay(playID: play)
            }
        }

        // lock() locks the set so that no more plays can be added to it
        //
        // Pre-Conditions:
        // The set cannot already have been locked
        access(all) fun lock() {
            if !self.locked {
                self.locked = true
                emit SetLocked(setID: self.setID)
            }
        }

        // mintMoment mints a new moment and returns the newly minted moment
        //
        // Parameters: playID: The ID of the play that the moment references
        //
        // Pre-Conditions:
        // The play must exist in the set and be allowed to mint new moments
        //
        // Returns: The NFT that was minted
        //
        access(all) fun mintMoment(playID: UInt32): @NFT {
            pre {
                self.retired[playID] != nil: "Cannot mint the moment: This play doesn't exist"
                !self.retired[playID]!: "Cannot mint the moment from this play: This play has been retired"
            }

            // get the number of moments that have been minted for this play
            // to use as this moment's serial number
            let numInPlay = self.numberMintedPerPlay[playID]!

            // mint the new moment
            let newMoment: @NFT <- create NFT(serialNumber: numInPlay + UInt32(1),
                                              playID: playID,
                                              setID: self.setID)

            // Increment the count of moments minted for this play
            self.numberMintedPerPlay[playID] = numInPlay + UInt32(1)

            return <-newMoment
        }

        // batchMintMoment mints an arbitrary quantity of moments
        // and returns them as a Collection
        //
        // Parameters: playID: the ID of the play that the moments are minted for
        //             quantity: The quantity of moments to be minted
        //
        // Returns: Collection object that contains all the moments that were minted
        //
        access(all) fun batchMintMoment(playID: UInt32, quantity: UInt64): @Collection {
            let newCollection <- create Collection()

            var i: UInt64 = 0
            while i < quantity {
                newCollection.deposit(token: <-self.mintMoment(playID: playID))
                i = i + UInt64(1)
            }

            return <-newCollection
        }
    }

    access(all) struct MomentData {

        // the ID of the Set that the Moment comes from
        access(all) let setID: UInt32

        // the ID of the Play that the moment references
        access(all) let playID: UInt32

        // the place in the play that this moment was minted
        // Otherwise know as the serial number
        access(all) let serialNumber: UInt32

        init(setID: UInt32, playID: UInt32, serialNumber: UInt32) {
            self.setID = setID
            self.playID = playID
            self.serialNumber = serialNumber
        }

    }

    // The resource that represents the Moment NFTs
    //
    access(all) resource NFT: NonFungibleToken.INFT {

        // global unique moment ID
        access(all) let id: UInt64

        // struct of moment metadata
        access(all) let data: MomentData

        init(serialNumber: UInt32, playID: UInt32, setID: UInt32) {
            // Increment the global moment IDs
            TopShot.totalSupply = TopShot.totalSupply + UInt64(1)

            self.id = TopShot.totalSupply

            // set the metadata struct
            self.data = MomentData(setID: setID, playID: playID, serialNumber: serialNumber)

            emit MomentMinted(momentID: self.id, playID: playID, setID: self.data.setID, serialNumber: self.data.serialNumber)
        }

        destroy() {
            emit MomentDestroyed(id: self.id)
        }
    }

    // Admin is a special authorization resource that
    // allows the owner to perform important functions to modify the
    // various aspects of the plays, sets, and moments
    //
    access(all) resource Admin {

        // createPlay creates a new Play struct
        // and stores it in the plays dictionary in the TopShot smart contract
        //
        // Parameters: metadata: A dictionary mapping metadata titles to their data
        //                       example: {"Player Name": "Kevin Durant", "Height": "7 feet"}
        //                               (because we all know Kevin Durant is not 6'9")
        //
        // Returns: the ID of the new Play object
        access(all) fun createPlay(metadata: {String: String}): UInt32 {
            // Create the new Play
            var newPlay = Play(metadata: metadata)
            let newID = newPlay.playID

            // Store it in the contract storage
            TopShot.playDatas[newID] = newPlay

            return newID
        }

        // createSet creates a new Set resource and returns it
        // so that the caller can store it in their account
        //
        // Parameters: name: The name of the set
        //             series: The series that the set belongs to
        //
        access(all) fun createSet(name: String) {
            // Create the new Set
            var newSet <- create Set(name: name)

            TopShot.sets[newSet.setID] <-! newSet
        }

        // borrowSet returns a reference to a set in the TopShot
        // contract so that the admin can call methods on it
        //
        // Parameters: setID: The ID of the set that you want to
        // get a reference to
        //
        // Returns: A reference to the set with all of the fields
        // and methods exposed
        //
        access(all) fun borrowSet(setID: UInt32): &Set {
            pre {
                TopShot.sets[setID] != nil: "Cannot borrow Set: The Set doesn't exist"
            }
            return (&TopShot.sets[setID] as &Set?)!
        }

        // startNewSeries ends the current series by incrementing
        // the series number, meaning that moments will be using the
        // new series number from now on
        //
        // Returns: The new series number
        //
        access(all) fun startNewSeries(): UInt32 {
            // end the current series and start a new one
            // by incrementing the TopShot series number
            TopShot.currentSeries = TopShot.currentSeries + UInt32(1)

            emit NewSeriesStarted(newCurrentSeries: TopShot.currentSeries)

            return TopShot.currentSeries
        }

        // createNewAdmin creates a new Admin Resource
        //
        access(all) fun createNewAdmin(): @Admin {
            return <-create Admin()
        }
    }

    // This is the interface that users can cast their moment Collection as
    // to allow others to deposit moments into their collection
    access(all) resource interface MomentCollectionPublic {
        access(all) fun deposit(token: @NonFungibleToken.NFT)
        access(all) fun batchDeposit(tokens: @NonFungibleToken.Collection)
        access(all) fun getIDs(): [UInt64]
        access(all) fun borrowNFT(id: UInt64): &NonFungibleToken.NFT
        access(all) fun borrowMoment(id: UInt64): &TopShot.NFT? {
            // If the result isn't nil, the id of the returned reference
            // should be the same as the argument to the function
            post {
                (result == nil) || (result?.id == id):
                    "Cannot borrow Moment reference: The ID of the returned reference is incorrect"
            }
        }
    }

    // Collection is a resource that every user who owns NFTs
    // will store in their account to manage their NFTS
    //
    access(all) resource Collection: MomentCollectionPublic, NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic {
        // Dictionary of Moment conforming tokens
        // NFT is a resource type with a UInt64 ID field
        access(all) var ownedNFTs: @{UInt64: NonFungibleToken.NFT}

        init() {
            self.ownedNFTs <- {}
        }

        // withdraw removes an Moment from the collection and moves it to the caller
        access(all) fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
            let token <- self.ownedNFTs.remove(key: withdrawID)
                ?? panic("Cannot withdraw: Moment does not exist in the collection")

            emit Withdraw(id: token.id, from: self.owner?.address)

            return <-token
        }

        // batchWithdraw withdraws multiple tokens and returns them as a Collection
        access(all) fun batchWithdraw(ids: [UInt64]): @NonFungibleToken.Collection {
            var batchCollection <- create Collection()

            // iterate through the ids and withdraw them from the collection
            for id in ids {
                batchCollection.deposit(token: <-self.withdraw(withdrawID: id))
            }
            return <-batchCollection
        }

        // deposit takes a Moment and adds it to the collections dictionary
        access(all) fun deposit(token: @NonFungibleToken.NFT) {
            let token <- token as! @TopShot.NFT

            let id = token.id
            // add the new token to the dictionary
            let oldToken <- self.ownedNFTs[id] <- token

            if self.owner?.address != nil {
                emit Deposit(id: id, to: self.owner?.address)
            }

            destroy oldToken
        }

        // batchDeposit takes a Collection object as an argument
        // and deposits each contained NFT into this collection
        access(all) fun batchDeposit(tokens: @NonFungibleToken.Collection) {
            let keys = tokens.getIDs()

            // iterate through the keys in the collection and deposit each one
            for key in keys {
                self.deposit(token: <-tokens.withdraw(withdrawID: key))
            }
            destroy tokens
        }

        // getIDs returns an array of the IDs that are in the collection
        access(all) fun getIDs(): [UInt64] {
            return self.ownedNFTs.keys
        }

        // borrowNFT Returns a borrowed reference to a Moment in the collection
        // so that the caller can read its ID
        //
        // Parameters: id: The ID of the NFT to get the reference for
        //
        // Returns: A reference to the NFT
        access(all) fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
            return (&self.ownedNFTs[id] as &NonFungibleToken.NFT?)!
        }

        // borrowMoment Returns a borrowed reference to a Moment in the collection
        // so that the caller can read data and call methods from it
        // They can use this to read its setID, playID, serialNumber,
        // or any of the setData or Play Data associated with it by
        // getting the setID or playID and reading those fields from
        // the smart contract
        //
        // Parameters: id: The ID of the NFT to get the reference for
        //
        // Returns: A reference to the NFT
        access(all) fun borrowMoment(id: UInt64): &TopShot.NFT? {
            if self.ownedNFTs[id] != nil {
                let ref = (&self.ownedNFTs[id] as &NonFungibleToken.NFT?)!
                return ref as! &TopShot.NFT
            } else {
                return nil
            }
        }

        // If a transaction destroys the Collection object,
        // All the NFTs contained within are also destroyed
        // Kind of like when Damien Lillard destroys the hopes and
        // dreams of the entire city of Houston
        //
        destroy() {
            destroy self.ownedNFTs
        }
    }

    // -----------------------------------------------------------------------
    // TopShot contract-level function definitions
    // -----------------------------------------------------------------------

    // createEmptyCollection creates a new, empty Collection object so that
    // a user can store it in their account storage.
    // Once they have a Collection in their storage, they are able to receive
    // Moments in transactions
    //
    access(all) fun createEmptyCollection(): @NonFungibleToken.Collection {
        return <-create TopShot.Collection()
    }

    // getAllPlays returns all the plays in topshot
    //
    // Returns: An array of all the plays that have been created
    access(all) fun getAllPlays(): [TopShot.Play] {
        return TopShot.playDatas.values
    }

    // getPlayMetaData returns all the metadata associated with a specific play
    //
    // Parameters: playID: The id of the play that is being searched
    //
    // Returns: The metadata as a String to String mapping optional
    access(all) fun getPlayMetaData(playID: UInt32): {String: String}? {
        return self.playDatas[playID]?.metadata
    }

    // getPlayMetaDataByField returns the metadata associated with a
    //                        specific field of the metadata
    //                        Ex: field: "Team" will return something
    //                        like "Memphis Grizzlies"
    //
    // Parameters: playID: The id of the play that is being searched
    //             field: The field to search for
    //
    // Returns: The metadata field as a String Optional
    access(all) fun getPlayMetaDataByField(playID: UInt32, field: String): String? {
        // Don't force a revert if the playID or field is invalid
        if let play = TopShot.playDatas[playID] {
            return play.metadata[field]
        } else {
            return nil
        }
    }

    // getSetName returns the name that the specified set
    //            is associated with.
    //
    // Parameters: setID: The id of the set that is being searched
    //
    // Returns: The name of the set
    access(all) fun getSetName(setID: UInt32): String? {
        // Don't force a revert if the setID is invalid
        return TopShot.setDatas[setID]?.name
    }

    // getSetSeries returns the series that the specified set
    //              is associated with.
    //
    // Parameters: setID: The id of the set that is being searched
    //
    // Returns: The series that the set belongs to
    access(all) fun getSetSeries(setID: UInt32): UInt32? {
        // Don't force a revert if the setID is invalid
        return TopShot.setDatas[setID]?.series
    }

    // getSetIDsByName returns the IDs that the specified set name
    //                 is associated with.
    //
    // Parameters: setName: The name of the set that is being searched
    //
    // Returns: An array of the IDs of the set if it exists, or nil if doesn't
    access(all) fun getSetIDsByName(setName: String): [UInt32]? {
        var setIDs: [UInt32] = []

        // iterate through all the setDatas and search for the name
        for setData in TopShot.setDatas.values {
            if setName == setData.name {
                // if the name is found, return the ID
                setIDs.append(setData.setID)
            }
        }

        // If the name isn't found, return nil
        // Don't force a revert if the setName is invalid
        if setIDs.length == 0 {
            return nil
        } else {
            return setIDs
        }
    }

    // getPlaysInSet returns the list of play IDs that are in the set
    //
    // Parameters: setID: The id of the set that is being searched
    //
    // Returns: An array of play IDs
    access(all) fun getPlaysInSet(setID: UInt32): [UInt32]? {
        // Don't force a revert if the setID is invalid
        return TopShot.sets[setID]?.plays
    }

    // isEditionRetired returns a boolean that indicates if a set/play combo
    //                  (otherwise known as an edition) is retired.
    //                  If an edition is retired, it still remains in the set,
    //                  but moments can no longer be minted from it.
    //
    // Parameters: setID: The id of the set that is being searched
    //             playID: The id of the play that is being searched
    //
    // Returns: Boolean indicating if the edition is retired or not
    access(all) fun isEditionRetired(setID: UInt32, playID: UInt32): Bool? {
        // Don't force a revert if the set or play ID is invalid
        // remove the set from the dictionary to ket its field
        if let setToRead <- TopShot.sets.remove(key: setID) {

            let retired = setToRead.retired[playID]

            TopShot.sets[setID] <-! setToRead

            return retired
        } else {
            return nil
        }
    }

    // isSetLocked returns a boolean that indicates if a set
    //             is locked. If an set is locked,
    //             new plays can no longer be added to it,
    //             but moments can still be minted from plays
    //             that are currently in it.
    //
    // Parameters: setID: The id of the set that is being searched
    //
    // Returns: Boolean indicating if the set is locked or not
    access(all) fun isSetLocked(setID: UInt32): Bool? {
        // Don't force a revert if the setID is invalid
        return TopShot.sets[setID]?.locked
    }

    // getNumMomentsInEdition return the number of moments that have been
    //                        minted from a certain edition.
    //
    // Parameters: setID: The id of the set that is being searched
    //             playID: The id of the play that is being searched
    //
    // Returns: The total number of moments
    //          that have been minted from an edition
    access(all) fun getNumMomentsInEdition(setID: UInt32, playID: UInt32): UInt32? {
        // Don't force a revert if the set or play ID is invalid
        // remove the set from the dictionary to get its field
        if let setToRead <- TopShot.sets.remove(key: setID) {

            // read the numMintedPerPlay
            let amount = setToRead.numberMintedPerPlay[playID]

            // put the set back
            TopShot.sets[setID] <-! setToRead

            return amount
        } else {
            return nil
        }
    }

    // -----------------------------------------------------------------------
    // TopShot initialization function
    // -----------------------------------------------------------------------
    //
    init() {
        // initialize the fields
        self.currentSeries = 0
        self.playDatas = {}
        self.setDatas = {}
        self.sets <- {}
        self.nextPlayID = 1
        self.nextSetID = 1
        self.totalSupply = 0

        // Put a new Collection in storage
        self.account.save<@Collection>(<- create Collection(), to: /storage/MomentCollection)

        // create a public capability for the collection
        let cap = self.account.capabilities.storage.issue<&{MomentCollectionPublic}>(/storage/MomentCollection)
        self.account.capabilities.publish(cap, at: /public/MomentCollection)

        // Put the Minter in storage
        self.account.save<@Admin>(<- create Admin(), to: /storage/TopShotAdmin)

        emit ContractInitialized()
    }
}
`

const realTopShotShardedCollectionContract = `

import NonFungibleToken from 0x1d7e57aa55817448
import TopShot from 0x0b2a3299cc857e29

access(all) contract TopShotShardedCollection {

    // ShardedCollection stores a dictionary of TopShot Collections
    // A Moment is stored in the field that corresponds to its id % numBuckets
    access(all) resource ShardedCollection: TopShot.MomentCollectionPublic, NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic { 
        
        // Dictionary of topshot collections
        access(all) var collections: @{UInt64: TopShot.Collection}

        // The number of buckets to split Moments into
        // This makes storage more efficient and performant
        access(all) let numBuckets: UInt64

        init(numBuckets: UInt64) {
            self.collections <- {}
            self.numBuckets = numBuckets

            // Create a new empty collection for each bucket
            var i: UInt64 = 0
            while i < numBuckets {

                self.collections[i] <-! TopShot.createEmptyCollection() as! @TopShot.Collection

                i = i + UInt64(1)
            }
        }

        // withdraw removes a Moment from one of the Collections 
        // and moves it to the caller
        access(all) fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
            post {
                result.id == withdrawID: "The ID of the withdrawn NFT is incorrect"
            }
            // Find the bucket it should be withdrawn from
            let bucket = withdrawID % self.numBuckets

            // Withdraw the moment
            let token <- self.collections[bucket]?.withdraw(withdrawID: withdrawID)!
            
            return <-token
        }

        // batchWithdraw withdraws multiple tokens and returns them as a Collection
        //
        // Parameters: ids: an array of the IDs to be withdrawn from the Collection
        //
        // Returns: @NonFungibleToken.Collection a Collection containing the moments
        //          that were withdrawn
        access(all) fun batchWithdraw(ids: [UInt64]): @NonFungibleToken.Collection {
            var batchCollection <- TopShot.createEmptyCollection()
            
            // Iterate through the ids and withdraw them from the Collection
            for id in ids {
                batchCollection.deposit(token: <-self.withdraw(withdrawID: id))
            }
            return <-batchCollection
        }

        // deposit takes a Moment and adds it to the Collections dictionary
        access(all) fun deposit(token: @NonFungibleToken.NFT) {

            // Find the bucket this corresponds to
            let bucket = token.id % self.numBuckets

            // Remove the collection
            let collection <- self.collections.remove(key: bucket)!

            // Deposit the nft into the bucket
            collection.deposit(token: <-token)

            // Put the Collection back in storage
            self.collections[bucket] <-! collection
        }

        // batchDeposit takes a Collection object as an argument
        // and deposits each contained NFT into this Collection
        access(all) fun batchDeposit(tokens: @NonFungibleToken.Collection) {
            let keys = tokens.getIDs()

            // Iterate through the keys in the Collection and deposit each one
            for key in keys {
                self.deposit(token: <-tokens.withdraw(withdrawID: key))
            }
            destroy tokens
        }

        // getIDs returns an array of the IDs that are in the Collection
        access(all) fun getIDs(): [UInt64] {

            var ids: [UInt64] = []
            // Concatenate IDs in all the Collections
            for key in self.collections.keys {
                for id in self.collections[key]?.getIDs() ?? [] {
                    ids.append(id)
                }
            }
            return ids
        }

        // borrowNFT Returns a borrowed reference to a Moment in the Collection
        // so that the caller can read data and call methods from it
        access(all) fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
            post {
                result.id == id: "The ID of the reference is incorrect"
            }

            // Get the bucket of the nft to be borrowed
            let bucket = id % self.numBuckets

            // Find NFT in the collections and borrow a reference
            return self.collections[bucket]?.borrowNFT(id: id)!
        }

        // borrowMoment Returns a borrowed reference to a Moment in the Collection
        // so that the caller can read data and call methods from it
        // They can use this to read its setID, playID, serialNumber,
        // or any of the setData or Play Data associated with it by
        // getting the setID or playID and reading those fields from
        // the smart contract
        //
        // Parameters: id: The ID of the NFT to get the reference for
        //
        // Returns: A reference to the NFT
        access(all) fun borrowMoment(id: UInt64): &TopShot.NFT? {

            // Get the bucket of the nft to be borrowed
            let bucket = id % self.numBuckets

            return self.collections[bucket]?.borrowMoment(id: id) ?? nil
        }

        // If a transaction destroys the Collection object,
        // All the NFTs contained within are also destroyed
        destroy() {
            destroy self.collections
        }
    }

    // Creates an empty ShardedCollection and returns it to the caller
    access(all) fun createEmptyCollection(numBuckets: UInt64): @ShardedCollection {
        return <-create ShardedCollection(numBuckets: numBuckets)
    }
}
`

const realTopshotAdminReceiverContract = `
import TopShot from 0x0b2a3299cc857e29
import TopShotShardedCollection from 0x0b2a3299cc857e29

access(all) contract TopshotAdminReceiver {

    // storeAdmin takes a TopShot Admin resource and 
    // saves it to the account storage of the account
    // where the contract is deployed
    access(all) fun storeAdmin(newAdmin: @TopShot.Admin) {
        self.account.save(<-newAdmin, to: /storage/TopShotAdmin)
    }
    
    init() {
        // Save a copy of the sharded Moment Collection to the account storage
        if self.account.borrow<&TopShotShardedCollection.ShardedCollection>(from: /storage/ShardedMomentCollection) == nil {
            let collection <- TopShotShardedCollection.createEmptyCollection(numBuckets: 32)
            // Put a new Collection in storage
            self.account.save(<-collection, to: /storage/ShardedMomentCollection)

            let cap = self.account.capabilities.storage.issue<&{TopShot.MomentCollectionPublic}>(/storage/ShardedMomentCollection)
            self.account.capabilities.publish(cap, at: /public/ShardedMomentCollection)
        }
    }
}
`
