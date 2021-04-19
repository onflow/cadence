/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func TestContractUpdateValidation(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime(
		WithContractUpdateValidationEnabled(true),
	)

	newDeployTransaction := func(function, name, code string) []byte {
		return []byte(fmt.Sprintf(`
			transaction {
				prepare(signer: AuthAccount) {
					signer.contracts.%s(name: "%s", code: "%s".decodeHex())
				}
			}`,
			function,
			name,
			hex.EncodeToString([]byte(code)),
		))
	}

	deployAndUpdate := func(t *testing.T, name string, oldCode string, newCode string) error {

		accountCode := map[common.LocationID][]byte{

			"A.631e88ae7f1d7c20.NonFungibleToken": []byte(`

// The main NFT contract interface. Other NFT contracts will
// The main NFT contract interface. Other NFT contracts will
// import and implement this interface
//
pub contract interface NonFungibleToken {

    // The total number of tokens of this type in existence
    pub var totalSupply: UInt64

    // Event that emitted when the NFT contract is initialized
    //
    pub event ContractInitialized()

    // Event that is emitted when a token is withdrawn,
    // indicating the owner of the collection that it was withdrawn from.
    //
    //
    pub event Withdraw(id: UInt64, from: Address?)

    // Event that emitted when a token is deposited to a collection.
    //
    // It indicates the owner of the collection that it was deposited to.
    //
    pub event Deposit(id: UInt64, to: Address?)

    // Interface that the NFTs have to conform to
    //
    pub resource interface INFT {
        // The unique ID that each NFT has
        pub let id: UInt64
    }

    // Requirement that all conforming NFT smart contracts have
    // to define a resource called NFT that conforms to INFT
    pub resource NFT: INFT {
        pub let id: UInt64
    }

    // Interface to mediate withdraws from the Collection
    //
    pub resource interface Provider {
        // withdraw removes an NFT from the collection and moves it to the caller
        pub fun withdraw(withdrawID: UInt64): @NFT {
            post {
                result.id == withdrawID: "The ID of the withdrawn token must be the same as the requested ID"
            }
        }
    }

    // Interface to mediate deposits to the Collection
    //
    pub resource interface Receiver {

        // deposit takes an NFT as an argument and adds it to the Collection
        //
		pub fun deposit(token: @NFT)
    }

    // Interface that an account would commonly 
    // publish for their collection
    pub resource interface CollectionPublic {
        pub fun deposit(token: @NFT)
        pub fun getIDs(): [UInt64]
        pub fun borrowNFT(id: UInt64): &NFT
    }

    // Requirement for the the concrete resource type
    // to be declared in the implementing contract
    //
    pub resource Collection: Provider, Receiver, CollectionPublic {

        // Dictionary to hold the NFTs in the Collection
        pub var ownedNFTs: @{UInt64: NFT}

        // withdraw removes an NFT from the collection and moves it to the caller
        pub fun withdraw(withdrawID: UInt64): @NFT

        // deposit takes a NFT and adds it to the collections dictionary
        // and adds the ID to the id array
        pub fun deposit(token: @NFT)

        // getIDs returns an array of the IDs that are in the collection
        pub fun getIDs(): [UInt64]

        // Returns a borrowed reference to an NFT in the collection
        // so that the caller can read data and call methods from it
        pub fun borrowNFT(id: UInt64): &NFT {
            pre {
                self.ownedNFTs[id] != nil: "NFT does not exist in the collection!"
            }
        }
    }

    // createEmptyCollection creates an empty Collection
    // and returns it to the caller so that they can own NFTs
    pub fun createEmptyCollection(): @Collection {
        post {
            result.getIDs().length == 0: "The created collection must be empty!"
        }
    }
}

`),

			"A.9a0766d93b6608b7.FungibleToken": []byte(`

/// FungibleToken
///
/// The interface that fungible token contracts implement.
///
pub contract interface FungibleToken {

    /// The total number of tokens in existence.
    /// It is up to the implementer to ensure that the total supply
    /// stays accurate and up to date
    ///
    pub var totalSupply: UFix64

    /// TokensInitialized
    ///
    /// The event that is emitted when the contract is created
    ///
    pub event TokensInitialized(initialSupply: UFix64)

    /// TokensWithdrawn
    ///
    /// The event that is emitted when tokens are withdrawn from a Vault
    ///
    pub event TokensWithdrawn(amount: UFix64, from: Address?)

    /// TokensDeposited
    ///
    /// The event that is emitted when tokens are deposited into a Vault
    ///
    pub event TokensDeposited(amount: UFix64, to: Address?)

    /// Provider
    ///
    /// The interface that enforces the requirements for withdrawing
    /// tokens from the implementing type.
    ///
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
        pub fun withdraw(amount: UFix64): @Vault {
            post {
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
    /// and enforces that when new Vaults are created, the balance
    /// is initialized correctly.
    ///
    pub resource interface Balance {

        /// The total balance of a vault
        ///
        pub var balance: UFix64

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
    pub resource Vault: Provider, Receiver, Balance {

        // The declaration of a concrete type in a contract interface means that
        // every Fungible Token contract that implements the FungibleToken interface

        /// The total balance of the vault
        ///
        pub var balance: UFix64

        // The conforming type must declare an initializer
        // that allows prioviding the initial balance of the Vault
        //
        init(balance: UFix64)

        /// and returns a new Vault with the subtracted balance
        ///
        pub fun withdraw(amount: UFix64): @Vault {
            pre {
                self.balance >= amount:
                    "Amount withdrawn must be less than or equal than the balance of the Vault"
            }
            post {
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
            result.balance == 0.0: "The newly created Vault must have zero balance"
        }
    }
}

`),

			"A.73dd87ae00edff1e.MojoAssetdefinition": []byte(`
import MojoCreatorInterfaces from 0x73dd87ae00edff1e
import MojoProject from 0x73dd87ae00edff1e

access(all) contract MojoAssetdefinition {
  
    // When an AssetdefinitionOwnerToken is created - emit the id of the assetDefinition and the creators address
    pub event AssetdefinitionOwnertokenCreated(id: Int, creator: Address)
    
    // When an Assetdefinition is created - emit assetdefinitinid
    pub event AssetdefinitionMinted(id: Int)
    
    // When a Property Writer is Created emit the assetdefinition id 
    pub event SupplyHandlerCreated()
    
    pub event AdministratorCreated()
    pub event AssetdefinitionRemoved(assetDefinition: Int)
    pub event AssetdefinitionsCleared()

    // the contract acts as kind of a database and stores all the definitions of assets
    access(account) var assetdefinitions: {Int: Assetdefinition}
    
    pub fun assetdefinitionIds(): [Int] {
      return self.assetdefinitions.keys
    }

    pub fun assetdefinition(assetdefinitionId: Int): AssetdefinitionPublic {
      let publicProperties: {String: {String: String}} = {}
      for property in self.assetdefinitions[assetdefinitionId]!.properties.values {
        var prop = property.toString()
        if(property.public_read != "true") {
          prop["value"] = "visible for owner only"
        }
        publicProperties[property.name] = prop
      }
      return AssetdefinitionPublic(
        id: assetdefinitionId,
        projectId: self.assetdefinitions[assetdefinitionId]!.projectId, 
        creator: self.assetdefinitions[assetdefinitionId]!.creator,
        properties: publicProperties,
        supplyType: self.assetdefinitions[assetdefinitionId]!.supplyType, 
        isFungible: self.assetdefinitions[assetdefinitionId]!.isFungible, 
        tradeType: self.assetdefinitions[assetdefinitionId]!.tradeType, 
        maxSupply: self.assetdefinitions[assetdefinitionId]!.maxSupply,
        mintType: self.assetdefinitions[assetdefinitionId]!.mintType, 
        burnType: self.assetdefinitions[assetdefinitionId]!.burnType,
        reserveMojoPerNFT: self.assetdefinitions[assetdefinitionId]!.reserveMojoPerNFT,
        totalSupply: self.assetdefinitions[assetdefinitionId]!.totalSupply,
        totalMelt: self.assetdefinitions[assetdefinitionId]!.totalMelt,
        totalReserve: self.assetdefinitions[assetdefinitionId]!.totalReserve,
        tradeFeeType: self.assetdefinitions[assetdefinitionId]!.tradeFeeType,
        tradeFeeValue: self.assetdefinitions[assetdefinitionId]!.tradeFeeValue,
        burnFeeType: self.assetdefinitions[assetdefinitionId]!.burnFeeType,
        burnFeeValue: self.assetdefinitions[assetdefinitionId]!.burnFeeValue
      )
    }
    
    access(contract) var currentId: Int
    
    init() {
      self.assetdefinitions = {}
      destroy self.account.load<@AnyResource>(from: /storage/oi_xojom_assetdefinitions_administrator)
      self.account.save(<- create Administrator(), to: /storage/oi_xojom_assetdefinitions_administrator)
      self.currentId = 0
    }

    pub fun createAssetdefinition(properties: {String: {String:String}}, supplyType: SupplyType, isFungible: Bool, tradeType: TradeType, maxSupply: UInt64, mintType: MintType, burnType: BurnType, token: &MojoProject.ProjectOwnertoken, ownerCollectionReference: &{MojoCreatorInterfaces.IMojoAssetdefinitionOwnertokenCollectionPublic}, reserveMojoPerNFT: UFix64, tradeFeeType: TradeFeeType, tradeFeeValue: UFix64, burnFeeType: BurnFeeType, burnFeeValue: UFix64) {
      let newAssetdefinition = Assetdefinition(id: self.currentId, projectId: token.projectId, creator: token.owner!.address,properties: properties, supplyType: supplyType, isFungible: isFungible, tradeType: tradeType, maxSupply: maxSupply,mintType: mintType, burnType: burnType, reserveMojoPerNFT: reserveMojoPerNFT, tradeFeeType: tradeFeeType, tradeFeeValue: tradeFeeValue, burnFeeType: burnFeeType, burnFeeValue: burnFeeValue)
      emit AssetdefinitionMinted(id: newAssetdefinition.id)
      let ownertoken <- create AssetdefinitionOwnertoken(assetdefinitionId: newAssetdefinition.id, creator: token.owner!.address)
      MojoAssetdefinition.assetdefinitions[newAssetdefinition.id] = newAssetdefinition
      self.currentId = self.currentId + 1
      ownerCollectionReference.depositAssetdefinitionOwnertoken(ownertoken: <- ownertoken)
    }
    
    pub resource Administrator {
      pub fun removeAssetdefinition(assetdefinitionId: Int) {
        MojoAssetdefinition.assetdefinitions[assetdefinitionId] = nil
        emit AssetdefinitionRemoved(assetDefinition: assetdefinitionId)
      }
      pub fun clearAssetdefinitions() {
        MojoAssetdefinition.assetdefinitions = {}
        MojoAssetdefinition.currentId = 0
        emit AssetdefinitionsCleared()
      }
      
      pub fun createAssetdefinitionOwnertoken(assetdefinitionId: Int): @AssetdefinitionOwnertoken {
        return <- create AssetdefinitionOwnertoken(assetdefinitionId: assetdefinitionId, creator: self.owner!.address)
      }
      
      init() {
        emit AdministratorCreated()
      }
    }
    
    pub resource AssetdefinitionOwnertoken : MojoCreatorInterfaces.Ownertoken {
      pub let assetdefinitionId: Int // assetdefinitionId
      pub let creator: Address

      init(assetdefinitionId: Int, creator: Address) {
        self.assetdefinitionId = assetdefinitionId
        self.creator = creator
        emit AssetdefinitionOwnertokenCreated(id: assetdefinitionId, creator: creator)
      }
      
      pub fun changeMaxSupply(newMaxSupply: UInt64) {
        MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.changeMaxSupply(newMaxSupply: newMaxSupply)
      }
      
      pub fun changeTradeType(newTradeType: TradeType) {
        MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.changeTradeType(newTradeType: newTradeType)
      }
      
      pub fun assetdefinition(): Assetdefinition? {
        return MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]
      }
      
      pub fun removeAssetdefinition() {
        pre{
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.totalSupply == UInt64(0) && MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.totalMelt == UInt64(0) : "There were already NFTs minted!"
        }
        MojoAssetdefinition.assetdefinitions.remove(key: self.assetdefinitionId)
      }
    }
    
    pub enum SupplyType: UInt8 {
        pub case fixed
        pub case collapsing
        pub case infinite
        pub case settable
    }

    pub enum TradeType: UInt8 {
        pub case always
        pub case temporary
        pub case never
    }
    
    pub enum MintType: UInt8 {
        pub case everyone
        pub case minterOnly
        pub case ownerOnly
        pub case minterAndOwner
    }
    
    pub enum BurnType: UInt8 {
        pub case everyone
        pub case burnerOnly
        pub case ownerOnly
        pub case burnerAndOwner
        pub case indestructable
    }
    
    pub enum TradeFeeType: UInt8 {
        pub case perItemPercentage
        pub case perItemAbsolute
        pub case perTransactionPercentage
        pub case perTransactionAbsolute
    }
    
    pub enum BurnFeeType: UInt8 {
        pub case percentage
        pub case absolute
    }
    
    pub struct AssetProperty {
      // The Name of the Property
      pub let name: String
      // The Description of the Property
      pub let description: String
      // The Value of the Property as String
      pub var value: String
      // The Unit of the Property as String
      pub let unit: String
      // The ValueType as String - no fixed interface!
      pub let value_type: String
      // allows the owner to mutate the property
      pub let owner_mutable: String
      // allows the writer-Resource to write the property
      pub let writer_mutable: String
      // can property be read public?
      pub var public_read: String
      
      pub var public_read_mutable: String
      
      
      pub fun toString(): {String: String } {
        return {
          "name":self.name,
          "description":self.description,
          "value":self.value,
           "unit":self.unit,
           "value_type":self.value_type,
          "owner_mutable":self.owner_mutable,
          "writer_mutable":self.writer_mutable,
          "public_read":self.public_read,
          "public_read_mutable":self.public_read_mutable
        }
      }
      access(account) fun setValue(value: String) {
        self.value = value
      }
      access(account) fun setPublicRead(value: String) {
        self.public_read = value
      } 
      
      init(name: String, description: String, value: String, unit: String, value_type: String, owner_mutable: String, writer_mutable: String, public_read: String, public_read_mutable: String) {
        self.name = name
        self.description = description
        self.value = value
        self.unit = unit
        self.value_type = value_type
        self.owner_mutable = owner_mutable
        self.writer_mutable = writer_mutable
        self.public_read = public_read
        self.public_read_mutable = public_read_mutable
      }
    }
    
    pub struct Assetdefinition {
        pub let id: Int
        pub let projectId: Int
        pub let creator: Address
        pub let isFungible: Bool
        pub let supplyType: SupplyType // 0: fixed - there can not be more than max, 1: collapsing - only maxSupply minted allowed (minus melts!),2:  infinite - can always be minted,3: settable - max can be changed
        pub var tradeType: TradeType // 0: always - can be traded, 1: temporary - can be switched to always or never, 2: never
        pub var maxSupply: UInt64 //
        pub let mintType: MintType
        pub let burnType: BurnType
        pub let tradeFeeType: TradeFeeType
        pub let tradeFeeValue: UFix64
        pub let burnFeeType: BurnFeeType
        pub let burnFeeValue: UFix64
        pub let reserveMojoPerNFT: UFix64
        pub let properties: {String: AssetProperty}
        
        pub var totalSupply: UInt64
        pub var totalMelt: UInt64
        pub var totalReserve: UFix64
        
        init(id: Int, projectId: Int, creator: Address, properties: {String: {String:String}}, supplyType: SupplyType, isFungible: Bool, tradeType: TradeType, maxSupply: UInt64, mintType: MintType, burnType: BurnType, reserveMojoPerNFT: UFix64, tradeFeeType: TradeFeeType, tradeFeeValue: UFix64, burnFeeType: BurnFeeType, burnFeeValue: UFix64) {
            pre {
              maxSupply > 0 as UInt64: "maxSupply must be bigger than 0"
              burnFeeType != BurnFeeType.percentage || (burnFeeValue > UFix64(0) && burnFeeValue <= UFix64(10)) : "fee percentage must be between 0 and 10"
              burnFeeType != BurnFeeType.absolute || (burnFeeValue > UFix64(0) && burnFeeValue <= (reserveMojoPerNFT / UFix64(2))) : "absolute per item is half of reserve"
              (tradeFeeType != TradeFeeType.perItemPercentage && tradeFeeType != TradeFeeType.perTransactionPercentage) || (tradeFeeValue > UFix64(0) && tradeFeeValue <= UFix64(10)) : "fee percentage must be between 0 and 10"
              (tradeFeeType != TradeFeeType.perItemAbsolute && tradeFeeType != TradeFeeType.perTransactionAbsolute) || (tradeFeeValue > UFix64(0) && tradeFeeValue <= (reserveMojoPerNFT  / UFix64(2))) : "absolute per transaction is half of reserve of 1 item"
            }
            self.id = id
            self.projectId = projectId
            self.creator = creator
            self.supplyType = supplyType
            self.isFungible = isFungible
            self.tradeType = tradeType
            self.maxSupply = maxSupply
            self.mintType = mintType
            self.burnType = burnType
            self.tradeFeeType = tradeFeeType
            self.tradeFeeValue = tradeFeeValue
            self.burnFeeType = burnFeeType
            self.burnFeeValue = burnFeeValue
            self.reserveMojoPerNFT = reserveMojoPerNFT
            
            self.totalSupply = 0
            self.totalMelt = 0
            self.totalReserve = UFix64(0)
            var props: {String: AssetProperty} = {}
            
            for propKey in properties.keys {
                let prop = properties[propKey]!
                props[propKey] = AssetProperty(
                  name: prop["name"]!, 
                  description: prop["description"]!,
                  value: prop["value"]!, 
                  unit: prop["unit"]!, 
                  value_type: prop["value_type"]!, 
                  owner_mutable: prop["owner_mutable"]!, 
                  writer_mutable: prop["writer_mutable"]!,
                  public_read: prop["public_read"]!,
                  public_read_mutable: prop["public_read_mutable"]!)
            }
            self.properties = props
        }
        
        access(account) fun increaseTotalSupply(addedSupply: UInt64) {
          pre {
            (addedSupply > 0 as UInt64) : "cannot add negative or zero supply"
          }
          self.totalSupply = self.totalSupply + addedSupply
        }
        
        access(account) fun increaseTotalMelt(addedMelt: UInt64) {
          pre {
            (addedMelt > 0 as UInt64) : "cannot add negative or zero melt"
          }
          self.totalSupply = self.totalSupply - addedMelt
          self.totalMelt = self.totalMelt + addedMelt
        }
        
        access(account) fun increaseTotalReserve(reserveChange: UFix64) {
          self.totalReserve = self.totalReserve + reserveChange
        }
        
        access(account) fun decreaseTotalReserve(reserveChange: UFix64) {
          self.totalReserve = self.totalReserve - reserveChange
        }
  
        access(account) fun changeMaxSupply(newMaxSupply: UInt64) {
          pre {
            self.supplyType == SupplyType.settable : "the supply type cannot be changed"
            newMaxSupply > (self.totalSupply - self.totalMelt) : "max supply cannot be lower than current supply"
          }
          self.maxSupply = newMaxSupply
        }

        access(account) fun changeTradeType(newTradeType: TradeType) {
          pre {
            self.tradeType == TradeType.temporary : "cannot change trade type"
            newTradeType == TradeType.always || newTradeType == TradeType.never : "no valid trade type selected"
          }
          self.tradeType = newTradeType
        }
        
    }

    
    pub struct AssetdefinitionPublic {
        pub let id: Int
        pub let projectId: Int
        pub let creator: Address
        pub let isFungible: Bool
        pub let supplyType: SupplyType // 0: fixed - there can not be more than max, 1: collapsing - only maxSupply minted allowed (minus melts!),2:  infinite - can always be minted,3: settable - max can be changed
        pub var tradeType: TradeType // 0: always - can be traded, 1: temporary - can be switched to always or never, 2: never
        pub var maxSupply: UInt64 //
        pub let mintType: MintType
        pub let burnType: BurnType
        pub let tradeFeeType: TradeFeeType
        pub let tradeFeeValue: UFix64
        pub let burnFeeType: BurnFeeType
        pub let burnFeeValue: UFix64
        pub let reserveMojoPerNFT: UFix64
        pub let properties: {String: {String: String}}
        
        pub var totalSupply: UInt64
        pub var totalMelt: UInt64
        pub var totalReserve: UFix64
        

        init(id: Int, projectId: Int, creator: Address, properties: {String: {String:String}}, supplyType: SupplyType, isFungible: Bool, tradeType: TradeType, maxSupply: UInt64, mintType: MintType, burnType: BurnType, reserveMojoPerNFT: UFix64, totalSupply: UInt64, totalMelt: UInt64, totalReserve: UFix64, tradeFeeType: TradeFeeType,tradeFeeValue: UFix64, burnFeeType: BurnFeeType, burnFeeValue: UFix64) {
            pre {
              maxSupply > 0 as UInt64: "maxSupply must be bigger than 0"
            }
            self.id = id
            self.projectId = projectId
            self.creator = creator
            self.supplyType = supplyType
            self.isFungible = isFungible
            self.tradeType = tradeType
            self.maxSupply = maxSupply
            self.mintType = mintType
            self.burnType = burnType
            self.reserveMojoPerNFT = reserveMojoPerNFT
            self.totalSupply = totalSupply
            self.totalMelt = totalMelt
            self.totalReserve = totalReserve
            self.properties= properties
            self.tradeFeeType = tradeFeeType
            self.tradeFeeValue = tradeFeeValue
            self.burnFeeType = burnFeeType
            self.burnFeeValue = burnFeeValue
        }
    }
}

`),

			"A.73dd87ae00edff1e.MojoCreatorInterfaces": []byte(`
access(all) contract MojoCreatorInterfaces {

    // Interface for Ownertokens
    pub resource interface Ownertoken {
      pub let creator: Address
    }
    
    pub resource interface AssetMinter {
      pub let assetdefinitionId: Int
      pub fun getAllowedAmount(): UFix64
    }
    
    pub resource interface PropertyWriter {
      pub let assetdefintionId: Int
    }
    
    pub resource interface AssetBurner {
      pub let assetdefinitionId: Int
      pub fun getAllowedAmount(): UFix64
    }
    
    pub resource interface IMojoProjectOwnertokenCollectionPublic {
      pub fun depositProjectOwnertoken(ownertoken: @AnyResource{Ownertoken})
      pub fun getOwnedProjectIds(): [Int]
    }
    
    pub resource interface IMojoAssetdefinitionOwnertokenCollectionPublic {
      pub fun depositAssetdefinitionOwnertoken(ownertoken: @AnyResource{Ownertoken})
      pub fun getOwnedAssetdefinitionIds(): [Int]
    }
    
    pub resource interface IMojoAssetMinterCollectionPublic {
      pub fun depositAssetMinter(assetMinter: @AnyResource{AssetMinter})
      pub fun getOwnedAssetMinterAssetdefinitionId(): [Int]
      pub fun getAllowedAmount(assetdefinitionId: Int): UFix64
    }
    
    pub resource interface IMojoAssetBurnerCollectionPublic {
      pub fun depositAssetBurner(assetBurner: @AnyResource{AssetBurner})
      pub fun getOwnedAssetBurnerAssetdefinitionId(): [Int]
      pub fun getAllowedAmount(assetdefinitionId: Int): UFix64
    }
    
    pub resource interface IMojoPropertyWriterCollectionPublic {
      pub fun depositPropertyWriter(propertyWriter: @AnyResource{PropertyWriter})
      pub fun getOwnedPropertyWriterAssetdefinitionId(): [Int]
    }
}
`),

			"A.73dd87ae00edff1e.MojoProject": []byte(`
import MojoCommunityVault from 0x73dd87ae00edff1e
import MojoCreatorInterfaces from 0x73dd87ae00edff1e
import FlowToken from 0x7e60df042a9c0868

access(all) contract MojoProject {
    // The Event When A new Project Ownertoken is created
    pub event ProjectOwnertokenCreated(id: Int, creator: Address)
    // The event when a project is minted and added to the list
    pub event ProjectMinted(id: Int, name: String)
    // event fired when a project is removed - only contract admin can do this
    pub event ProjectRemoved(projectId: Int)
    // event when projects were cleared - only contract admin can do this
    pub event ProjectsCleared()
    
    pub event ProjectNameChanged(projectId: Int)
    pub event ProjectDescriptionChanged(projectId: Int)
    pub event ProjectUrlChanged(projectId: Int)
    pub event ProjectImageChanged(projectId: Int)
    pub event ProjectCreationcostChanged(newCost: UFix64)
    // contract only access flowVault to store the project-creation costs
    access(contract) let flowVault: @FlowToken.Vault
    
    // the max amount for a project to be payed (admin cannot go above this price)
    pub let maxProjectCreationCost: UFix64
    // the current costprice for creating a new project
    pub var projectCreationCost: UFix64
    // the contract acts as kind of a database and stores all the definitions of assets
    pub var projects: {Int: Project}
    // the currentproject Id 
    access(contract) var currentId: Int

    init() {
      self.projects = {}
      self.account.save(<- create Administrator(), to: /storage/oi_xojom_projects_administrator)
      self.flowVault <- FlowToken.createEmptyVault() as! @FlowToken.Vault
      self.maxProjectCreationCost = 100.0
      self.projectCreationCost = 0.05
      self.currentId = 0
    }

    // Everyone can create a project - with a little flowTokenPayment
    pub fun createProject(name: String, description: String, url: String, image: String, paymentVault: @FlowToken.Vault, ownerCollectionReference: &{MojoCreatorInterfaces.IMojoProjectOwnertokenCollectionPublic}) {
        pre {
            // check balance of payment Vault
            self.projectCreationCost <= paymentVault.balance : "paymentbalance too low"
        }
        // deposit the flow tokens in the contracts vault
        MojoCommunityVault.depositFlow(vaultName: "projectFeeFlowVault", from: <- paymentVault)
        // create the new project
        let newProject = Project(name: name, description: description, url: url, image: image, creator: ownerCollectionReference.owner!.address)
        // store the project in the list
        MojoProject.projects[newProject.id] = newProject
        // emit the minted event
        emit ProjectMinted(id: newProject.id, name: newProject.name)
        // deposit the ownertoken
        ownerCollectionReference.depositProjectOwnertoken(ownertoken: <- create ProjectOwnertoken(projectId: newProject.id, creator: ownerCollectionReference.owner!.address))
    }
    
    pub fun projectIds(): [Int] {
      return self.projects.keys
    }
    
    pub fun project(projectId: Int): Project {
      return self.projects[projectId]!
    }

    pub resource ProjectOwnertoken: MojoCreatorInterfaces.Ownertoken {
      pub let creator: Address
      pub let projectId: Int

      init(projectId: Int, creator: Address) {
        self.creator = creator
        self.projectId = projectId
        emit ProjectOwnertokenCreated(id: projectId, creator: creator)
      }
      
      pub fun setName(name: String) {
        MojoProject.projects[self.projectId]!.setName(name: name)
      }
      
      pub fun setDescription(description: String) {
        MojoProject.projects[self.projectId]!.setDescription(description: description)
      }
      
      pub fun setUrl(url: String) {
        MojoProject.projects[self.projectId]!.setUrl(url: url)
      }
      
      pub fun setImage(image: String) {
        MojoProject.projects[self.projectId]!.setImage(image: image)
      }
    }

    pub resource Administrator {
      pub fun createProject(name: String, description: String, url: String, image: String): @ProjectOwnertoken {
        // create the new project
        let newProject = Project(name: name, description: description,url: url, image: image, creator: self.owner!.address)
        MojoProject.projects[newProject.id] = newProject
        return <- create ProjectOwnertoken(projectId: newProject.id, creator: self.owner!.address)
      }
      pub fun clearProjects() {
        MojoProject.projects = {}
        MojoProject.currentId = 0
        emit ProjectsCleared()
      }
      pub fun removeProject(projectId: Int) {
        MojoProject.projects.remove(key:projectId)
        emit ProjectRemoved(projectId: projectId)
      }
      
      pub fun createProjectOwnerToken(projectId: Int): @ProjectOwnertoken {
        return <- create ProjectOwnertoken(projectId: projectId, creator: self.owner!.address)
      }
      pub fun withdrawFlowVault() {
        // Not Implemented Now...
      }
      pub fun changeProjectCreationCost(newCreationCost: UFix64) {
        pre {
          newCreationCost >= 0.0 : "creation costs too low"
          newCreationCost < MojoProject.maxProjectCreationCost : "creation cost too high - not allowed!"
        }
        MojoProject.projectCreationCost = newCreationCost
        emit ProjectCreationcostChanged(newCost: newCreationCost)
      }
    }

    pub struct Project {
        pub let id: Int
        pub let creator: Address
        pub var name: String
        pub var description: String
        pub var url: String
        pub var image: String
        
        access(contract) fun setName(name: String) {
          self.name = name
          emit ProjectNameChanged(projectId: self.id)
        }
        access(contract) fun setDescription(description: String) {
          self.description = description
          emit ProjectDescriptionChanged(projectId: self.id)
        }
        access(contract) fun setUrl(url: String) {
          self.url = url
          emit ProjectUrlChanged(projectId: self.id)
        }
        access(contract) fun setImage(image: String) {
          self.image = image
          emit ProjectImageChanged(projectId: self.id)
        }
        
        init(name: String, description: String, url: String, image: String, creator: Address) {
            self.id = MojoProject.currentId
            self.name = name
            self.description = description
            self.creator = creator
            self.url = url
            self.image = image
            MojoProject.currentId = MojoProject.currentId + 1
            emit ProjectMinted(id: self.id, name: name)
        }
    }
}

`),

			"A.73dd87ae00edff1e.MojoCommunityVault": []byte(`

import MojoToken from 0x73dd87ae00edff1e
import MojoAsset from 0x73dd87ae00edff1e
import FlowToken from 0x7e60df042a9c0868
import NonFungibleToken from 0x631e88ae7f1d7c20
import FungibleToken from 0x9a0766d93b6608b7

access(all) contract MojoCommunityVault {
    // The Event When A new Project Ownertoken is created
    pub event MojoTokenDeposited(amount: UFix64, by: Address)
    
    // contract only access flowVault to store the project-creation costs
    access(contract) let flowVaults: @{String: FlowToken.Vault}
    access(contract) let mojoVaults: @{String: MojoToken.Vault}
    access(contract) let nftCollections: @{String: MojoAsset.Collection}
    
    init() {
      self.flowVaults <- {
        "projectfeeFlowVault": <- (FlowToken.createEmptyVault() as! @FlowToken.Vault),
        "donationFlowVault": <- (FlowToken.createEmptyVault() as! @FlowToken.Vault),
        "communityFlowVault": <- (FlowToken.createEmptyVault() as! @FlowToken.Vault)
      }
      self.mojoVaults <- {
        "burnFeeMojoVault": <- (MojoToken.createEmptyVault() as! @MojoToken.Vault),
        "mintFeeMojoVault": <- (MojoToken.createEmptyVault() as! @MojoToken.Vault),
        "marketFeeMojoVault": <- (MojoToken.createEmptyVault() as! @MojoToken.Vault),
        "donationMojoVault": <- (MojoToken.createEmptyVault() as! @MojoToken.Vault),
        "communityMojoVault": <- (MojoToken.createEmptyVault() as! @MojoToken.Vault)
      }
      self.nftCollections <- {
        "communityMojoAssetCollection": <- (MojoAsset.createEmptyCollection() as! @MojoAsset.Collection)
      }
    }
    
    pub fun flowVaultBalance(vaultName: String): UFix64 {
      return (&(self.flowVaults[vaultName] as! &FlowToken.Vault)).balance
    }
    pub fun mojoVaultBalance(vaultName: String): UFix64 {
      return (&(self.mojoVaults[vaultName] as! &MojoToken.Vault)).balance
    }
    pub fun mojoTokenVaults(): [String] {
      return self.mojoVaults.keys
    }
    pub fun flowTokenVaults(): [String] {
      return self.flowVaults.keys
    }
    
    pub fun depositFlow(vaultName: String, from: @FlowToken.Vault) {
      pre {
        self.flowVaults[vaultName] != nil : "vaultname invalid!"
        from.balance >= UFix64(0): "nothing to deposit!"
      }
      (&(self.flowVaults[vaultName] as! &FlowToken.Vault)).deposit(from: <- (from as! @FungibleToken.Vault))
    }
    pub fun depositMojo(vaultName: String, from: @MojoToken.Vault) {
      pre {
        self.mojoVaults[vaultName] != nil : "vaultname invalid!"
        from.balance >= UFix64(0): "nothing to deposit!"
      }
      (&(self.mojoVaults[vaultName] as! &MojoToken.Vault)).deposit(from: <- (from as! @FungibleToken.Vault))
    }
}

`),
			"A.73dd87ae00edff1e.MojoToken": []byte(`
import FungibleToken from 0x9a0766d93b6608b7
import MojoAdminInterfaces from 0x73dd87ae00edff1e

pub contract MojoToken: FungibleToken {

    // Total supply of Mojo tokens in existence
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
            let vault <- from as! @MojoToken.Vault
            self.balance = self.balance + vault.balance
            emit TokensDeposited(amount: vault.balance, to: self.owner?.address)
            vault.balance = 0.0
            destroy vault
        }

        destroy() {
            // if the balance is not zero distribute tokens
            let oldbalance = self.balance
            if(self.balance > UFix64(0)) {
              let mojoVaultRef = getAccount(self.owner!.address)
                .getCapability(/public/mojoTokenReceiver)!
                .borrow<&MojoToken.Vault{FungibleToken.Receiver}>()
              mojoVaultRef!.deposit(from: <- self.withdraw(amount: self.balance))
            }
            MojoToken.totalSupply = MojoToken.totalSupply - oldbalance
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
    pub resource Minter : MojoAdminInterfaces.ITokenMinterPublic {

        // the amount of tokens that the minter is allowed to mint
        pub var allowedAmount: UFix64
        pub fun getAllowedAmount(): UFix64 {
          return self.allowedAmount
        }
        // mintTokens
        //
        // Function that mints new tokens, adds them to the total supply,
        // and returns them to the calling context.
        //
        pub fun mintTokens(amount: UFix64): @MojoToken.Vault {
            pre {
                amount > UFix64(0): "Amount minted must be greater than zero"
                amount <= self.allowedAmount: "Amount minted must be less than the allowed amount"
            }
            MojoToken.totalSupply = MojoToken.totalSupply + amount
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
    pub resource Burner : MojoAdminInterfaces.ITokenBurnerPublic {
        // burnTokens
        //
        // Function that destroys a Vault instance, effectively burning the tokens.
        //
        // Note: the burned tokens are automatically subtracted from the 
        // total supply in the Vault destructor.
        //
        pub fun burnTokens(from: @FungibleToken.Vault) {
            let vault <- from as! @MojoToken.Vault
            let amount = vault.balance
            destroy vault
            emit TokensBurned(amount: amount)
        }
    }

    init() {
        self.totalSupply = 0.0

        // Create the Vault with the total supply of tokens and save it in storage
        //
        let vault <- create Vault(balance: self.totalSupply)
        self.account.save(<-vault, to: /storage/mojoTokenVault)

        // Create a public capability to the stored Vault that only exposes
        //
        self.account.link<&MojoToken.Vault{FungibleToken.Receiver}>(
            /public/mojoTokenReceiver,
            target: /storage/mojoTokenVault
        )

        // Create a public capability to the stored Vault that only exposes
        //
        self.account.link<&MojoToken.Vault{FungibleToken.Balance}>(
            /public/mojoTokenBalance,
            target: /storage/mojoTokenVault
        )

        let admin <- create Administrator()
        self.account.save(<-admin, to: /storage/mojoTokenAdmin)

        // Emit an event that shows that the contract was initialized
        emit TokensInitialized(initialSupply: self.totalSupply)
    }
}
`),

			"A.73dd87ae00edff1e.MojoAdminInterfaces": []byte(`
access(all) contract MojoAdminInterfaces {

    pub resource interface ITokenMinterPublic {
      pub fun getAllowedAmount(): UFix64
    }
    
    pub resource interface ITokenBurnerPublic {
    }
    
    pub resource interface IProjectAdministratorCollectionPublic {
      pub fun depositProjectAdministrator(administrator: @AnyResource)
      pub fun hasProjectAdministrator(): Bool
    }
    
    pub resource interface IAssetdefinitionAdministratorCollectionPublic {
      pub fun depositAssetdefinitonAdministrator(administrator: @AnyResource)
      pub fun hasAssetdefinitionAdministrator(): Bool
    }
    
    pub resource interface IMojoTokenAdministratorCollectionPublic {
      pub fun depositMojoTokenAdministrator(administrator: @AnyResource)
      pub fun hasMojoTokenAdministrator(): Bool
    }
}

`),

			"A.73dd87ae00edff1e.MojoAsset": []byte(`
import NonFungibleToken from 0x631e88ae7f1d7c20
import FungibleToken from 0x9a0766d93b6608b7
import MojoAssetdefinition from 0x73dd87ae00edff1e
import MojoToken from 0x73dd87ae00edff1e
import FlowToken from 0x7e60df042a9c0868

pub contract MojoAsset: NonFungibleToken {

  pub var totalSupply: UInt64
  pub var totalMelt: UInt64
  pub var mintFeePercent: UFix64
  pub var burnFeePercent: UFix64

  pub event ContractInitialized()
  pub event Withdraw(id: UInt64, from: Address?)
  pub event Deposit(id: UInt64, to: Address?)
  pub event NFTBurnt(id: UInt64)

  init() {
    // Initialize the total supply
    self.totalSupply = 0
    self.totalMelt = 0
    self.mintFeePercent = UFix64(10)
    self.burnFeePercent = UFix64(10)
    // Create a Collection resource and save it to storage
    destroy self.account.load<@AnyResource>(from: /storage/mojoAssetCollection)
    self.account.save(<- create Collection(), to: /storage/mojoAssetCollection)

    // create a public capability for the collection
    self.account.link<&{NonFungibleToken.CollectionPublic,MojoAsset.MojoAssetCollectionPublic}>(
        /public/mojoCollectionPublic,
        target: /storage/mojoAssetCollection
    ) 

    // Create a Minter resource and save it to storage
    destroy self.account.load<@AnyResource>(from: /storage/mojoNFTAdministrator)
    self.account.save(<- create Administrator(), to: /storage/mojoNFTAdministrator)
    emit ContractInitialized()
  }
  
  pub resource interface MojoAssetCollectionPublic {
    pub fun getIDsByProjectId(projectId: Int): [UInt64]
    pub fun getIDsByAssetdefinitionId(assetdefinitionId: Int): [UInt64]
    pub fun getProjectIds(): [Int]
    pub fun getAssetdefinitionIds(): [Int]
    pub fun nftPropertiesPublic(id: UInt64): {String: {String: String}}
  }
  
  priv fun sendFees(feeVault: @MojoToken.Vault) {
    if(feeVault.balance > UFix64(0)) {
      let mojoVaultRef = getAccount(self.account.address)
        .getCapability(/public/mojoTokenReceiver)!
        .borrow<&MojoToken.Vault{FungibleToken.Receiver}>()
      mojoVaultRef!.deposit(from: <- feeVault.withdraw(amount: feeVault.balance))
    }
    destroy feeVault
    
  }
  
  priv fun sendCreatorFees(creatorFeeVault: @MojoToken.Vault, creatorAddress: Address) {
    if(creatorFeeVault.balance > UFix64(0)) {
      let mojoVaultRef = getAccount(creatorAddress)
        .getCapability(/public/mojoTokenReceiver)!
        .borrow<&MojoToken.Vault{FungibleToken.Receiver}>()
      mojoVaultRef!.deposit(from: <- creatorFeeVault.withdraw(amount: creatorFeeVault.balance))
    }
    destroy creatorFeeVault
  }
  
  pub fun mintNFT(receiverAddress: Address, token: &MojoAssetdefinition.AssetdefinitionOwnertoken, reserveMojoTokenVault: @MojoToken.Vault, properties: {String: {String: String}}?) {
    // create a new NFT
    let receiverRef = getAccount(receiverAddress).getCapability(/public/mojoAssetCollectionPublic)!.borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("NO NFT Receiver found")
    
    receiverRef.deposit(token: <- create NFT(assetdefinitionId: token.assetdefinitionId, reserveMojoTokenVault: <-reserveMojoTokenVault, properties: properties))
  }
  
  pub fun createNFTMinter(maxUse: Int8, token: &MojoAssetdefinition.AssetdefinitionOwnertoken, reserveMojoTokenVault: @MojoToken.Vault): @AssetMinter {
    // create a new NFT
    return <- create AssetMinter(assetdefinitionId: token.assetdefinitionId, maxUse: maxUse, reserveMojoTokenVault: <- reserveMojoTokenVault)
  }
  
  pub fun createNFTBurner(maxUse: Int8, token: &MojoAssetdefinition.AssetdefinitionOwnertoken): @AssetBurner {
    // create a new NFT
    return <- create AssetBurner(assetdefinitionId: token.assetdefinitionId, maxUse: maxUse)
  }
  
  pub fun mintNFTPublic(receiverAddress: Address, assetdefinitionId: Int, reserveMojoTokenVault: @MojoToken.Vault, properties: {String: {String: String}}?) {
    // create a new NFT
    pre {
      MojoAssetdefinition.assetdefinitions[assetdefinitionId] != nil : "assetdefinitionId invalid"
      MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType! == MojoAssetdefinition.MintType.everyone : "This asset cannot be minted!"
    }
    let receiverRef = getAccount(receiverAddress).getCapability(/public/mojoAssetCollectionPublic)!.borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("NO NFT Receiver found")
    
    receiverRef.deposit(token: <- create NFT(assetdefinitionId: assetdefinitionId, reserveMojoTokenVault: <-reserveMojoTokenVault, properties: properties))
  }
  
  pub fun burnNFT(asset: @MojoAsset.NFT, token: &MojoAssetdefinition.AssetdefinitionOwnertoken) {
    // create a new NFT
    pre {
      asset.assetdefinitionId == token.assetdefinitionId : "no valid ownertoken"
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId] != nil : "assetdefinitionId invalid"
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId]!.burnType! == MojoAssetdefinition.BurnType.everyone
        || 
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId]!.burnType! == MojoAssetdefinition.BurnType.burnerAndOwner
        ||
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId]!.burnType! == MojoAssetdefinition.BurnType.ownerOnly
        : "not allowed to burn"
    }
    asset.setBurnValidated(validated: true)
    destroy asset
  }
  
  pub resource NFT: NonFungibleToken.INFT {
  
      access(contract) var burnValidated: Bool
      pub let id: UInt64
      pub let assetdefinitionId: Int
      pub let properties: {String: MojoAssetdefinition.AssetProperty}
      
      access(self) let mojoTokenVault: @MojoToken.Vault
      
      access(self) let mojoTokenVaults: @{String: MojoToken.Vault}
      access(self) let flowTokenVaults: @{String: FlowToken.Vault}
      access(self) let mojoAssets: @{String: MojoAsset.NFT}
      
      access(contract) fun setBurnValidated(validated: Bool) {
        self.burnValidated = validated
      }
      init(assetdefinitionId: Int, reserveMojoTokenVault: @MojoToken.Vault, properties: {String: {String:String}}? ) {
        pre {
          reserveMojoTokenVault.balance >= MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.reserveMojoPerNFT : "Not Enough MojoTokens For Reserve"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]! != nil : "assetdefinition does not exist"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.fixed || 
          (MojoAssetdefinition.assetdefinitions[assetdefinitionId] == nil || MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply < MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.maxSupply) : "maxsupply reached!"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.collapsing || (MojoAssetdefinition.assetdefinitions[assetdefinitionId] == nil || MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply + MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalMelt < MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.maxSupply) : "maxsupply reached!"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.infinite || true : "will never occure"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.settable || (MojoAssetdefinition.assetdefinitions[assetdefinitionId] == nil || MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply < MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.maxSupply) : "maxsupply reached!"
        }
        self.burnValidated = false
        self.id = MojoAsset.totalSupply + MojoAsset.totalMelt + UInt64(1)
        self.assetdefinitionId = assetdefinitionId
        self.properties = MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.properties
        // change name if its a fungible token
        if(MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.isFungible == false) {
          self.properties["name"]?.setValue(value: self.properties["name"]!.value.concat(" #").concat((MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply + MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalMelt + 1 as UInt64).toString()))
          for property in self.properties.values {
            if(property.value_type == "random") {
              self.properties[property.name]?.setValue(value: unsafeRandom().toString())
            }
          }
          if(properties != nil) {
            for property in properties!.values {
              if(self.properties![property["name"]!]!.owner_mutable == "true") {
                self.properties![property["name"]!]?.setValue(value: property["value"]!)
              }
            }
          }
        }
        self.mojoTokenVault <- MojoToken.createEmptyVault() as! @MojoToken.Vault
        self.mojoTokenVaults <- {}
        self.flowTokenVaults <- {}
        self.mojoAssets <- {}
        let feeAmount = reserveMojoTokenVault.balance * MojoAsset.mintFeePercent / UFix64(100)
        MojoAsset.sendFees(feeVault: <- (reserveMojoTokenVault.withdraw(amount: feeAmount) as! @MojoToken.Vault))
        let reserveAmount = reserveMojoTokenVault.balance
        self.mojoTokenVault.deposit(from: <- reserveMojoTokenVault)
        MojoAssetdefinition.assetdefinitions[assetdefinitionId]?.increaseTotalReserve(reserveChange: reserveAmount)
        MojoAssetdefinition.assetdefinitions[assetdefinitionId]?.increaseTotalSupply(addedSupply: UInt64(1))
        MojoAsset.totalSupply = MojoAsset.totalSupply + 1 as UInt64
      }
      pub fun writeProperty(propertyName: String, value: String ) {
        pre {
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.properties[propertyName]!.owner_mutable == "true" : "the property is not writable"
        }
        self.properties[propertyName]?.setValue(value: value)
      }
      pub fun writePublicRead(propertyName: String, value: String ) {
        pre {
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
        }
        self.properties[propertyName]?.setPublicRead(value: value)
      }
      destroy() {
        if(MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.indestructable ) {
          panic("this item is indestructable!")
        }
        if((MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnType != MojoAssetdefinition.BurnType.everyone) && self.burnValidated != true) {
          panic("not allowed to burn item")
        }
        let oldReserveBalance = self.mojoTokenVault.balance
        MojoAsset.sendFees(feeVault: <- (self.mojoTokenVault.withdraw(amount: (self.mojoTokenVault.balance * MojoAsset.burnFeePercent / UFix64(100))) as! @MojoToken.Vault))
        let creatorAddress = MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]! != nil ? MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.creator : getAccount(0x73dd87ae00edff1e).address
        var feeAmount = MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnFeeValue
        if(MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnFeeType == MojoAssetdefinition.BurnFeeType.percentage) {
          feeAmount = (feeAmount / UFix64(100)) * self.mojoTokenVault.balance
        }
        MojoAsset.sendCreatorFees(creatorFeeVault: <- (self.mojoTokenVault.withdraw(amount: feeAmount) as! @MojoToken.Vault), creatorAddress: creatorAddress)
        
        if(MojoAsset.totalSupply > UInt64(0)) {
          MojoAsset.totalSupply = MojoAsset.totalSupply - 1 as UInt64
        }
        MojoAsset.totalMelt = MojoAsset.totalMelt + 1 as UInt64
        MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]?.increaseTotalMelt(addedMelt: UInt64(1)) 
        MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]?.decreaseTotalReserve(reserveChange: oldReserveBalance)
        // pay back the frozen mojo
        destroy self.mojoTokenVault
        destroy self.mojoTokenVaults
        destroy self.flowTokenVaults
        destroy self.mojoAssets
        
      }
  }

  pub resource Collection: NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic, MojoAssetCollectionPublic {
    // dictionary of NFT conforming tokens

    pub var ownedNFTs: @{UInt64: NonFungibleToken.NFT}
    // nftId: [projectId, assetDefinitionId]
    access(self) var projectMapping: {UInt64: [Int]}
    //access(self) var transferFeeVault: @MojoToken.Vault
    init () {
      self.ownedNFTs <- {}
      self.projectMapping = {}
    }
    // withdraw removes an NFT from the collection and moves it to the caller
    pub fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
      let mojoAsset = self.borrowMojoNFT(id: withdrawID)!
      let token <- self.ownedNFTs.remove(key: withdrawID)!

      if(MojoAssetdefinition.assetdefinitions[mojoAsset.assetdefinitionId]!.tradeType == MojoAssetdefinition.TradeType.never) {
        panic("this asset cannot be traded!")
      }
      emit Withdraw(id: mojoAsset.id, from: self.owner?.address)
      self.projectMapping.remove(key: mojoAsset.id)
      return <-token
    }
    pub fun withdrawMany(withdrawIDs: [UInt64]): @[NonFungibleToken.NFT] {
      let withdrawArray: @[NonFungibleToken.NFT] <- []
      for id in withdrawIDs {
        let token <- self.withdraw(withdrawID: id) as! @NonFungibleToken.NFT
        withdrawArray.append(<- token)
      }
      return <- withdrawArray
    }
    // deposit takes a NFT and adds it to the collections dictionary
    // and adds the ID to the id array
    pub fun deposit(token: @NonFungibleToken.NFT) {
      pre {
        token != nil : "There is no NFT to deposit "
      }
      let _token <- token as! @MojoAsset.NFT
      emit Deposit(id: _token.id, to: self.owner?.address)
      self.projectMapping[_token.id] = [MojoAssetdefinition.assetdefinitions[_token.assetdefinitionId]!.projectId, _token.assetdefinitionId]
      // add the new token to the dictionary which removes the old one
      self.ownedNFTs[_token.id] <-! _token
    }
    pub fun depositMany(tokens: @[NonFungibleToken.NFT]) {
      while(tokens.length > 0) {
        let token <- tokens.removeFirst() as! @MojoAsset.NFT
        emit Deposit(id: token.id, to: self.owner?.address)
        self.deposit(token: <- token)
      }
      destroy tokens
    }

    // getIDs returns an array of the IDs that are in the collection
    pub fun getIDs(): [UInt64] {
      return self.ownedNFTs.keys
    }

    // getIDs returns an array of the IDs that are in the collection
    pub fun getIDsByAssetdefinitionId(assetdefinitionId: Int): [UInt64] {
      let resultIds: [UInt64] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if((ref as! &MojoAsset.NFT).assetdefinitionId == assetdefinitionId) {
          resultIds.append(nftId)
        }

      }
      return resultIds
    }
    // getIDs returns an array of the IDs that are in the collection
    pub fun getIDsByProjectId(projectId: Int): [UInt64] {
      let resultIds: [UInt64] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if(MojoAssetdefinition.assetdefinitions[(ref as! &MojoAsset.NFT).assetdefinitionId!]?.projectId == projectId) {
          resultIds.append(nftId)
        }
      }
      return resultIds
    }

    // getIDs returns an array of the IDs that are in the collection
    pub fun getProjectIds(): [Int] {
      let resultIds: [Int] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if(!resultIds.contains(MojoAssetdefinition.assetdefinitions[(ref as! &MojoAsset.NFT).assetdefinitionId!]?.projectId!)) {
          resultIds.append(MojoAssetdefinition.assetdefinitions[(ref as! &MojoAsset.NFT).assetdefinitionId!]?.projectId!)
        }
      }
      return resultIds
    }

    // getIDs returns an array of the IDs that are in the collection
    pub fun getAssetdefinitionIds(): [Int] {
      let resultIds: [Int] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if(!resultIds.contains((ref as! &MojoAsset.NFT).assetdefinitionId!)) {
          resultIds.append((ref as! &MojoAsset.NFT).assetdefinitionId!)
        }
      }
      return resultIds
    }

    // borrowNFT gets a reference to an NFT in the collection
    // so that the caller can read its metadata and call its methods
    pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
      return &self.ownedNFTs[id] as! &NonFungibleToken.NFT
    }

    pub fun borrowMojoNFT(id: UInt64): &MojoAsset.NFT? {
        if self.ownedNFTs[id] != nil {
            let ref = &self.ownedNFTs[id] as auth &NonFungibleToken.NFT
            return ref as! &MojoAsset.NFT
        } else {
            return nil
        }
    }

    pub fun nftPropertiesPublic(id: UInt64): {String: {String: String}} {
      let publicProperties: {String: {String: String}} = {}
      let ref = &self.ownedNFTs[id] as auth &NonFungibleToken.NFT
      for property in (ref as! &MojoAsset.NFT).properties.values {
        var prop = property.toString()
        if(property.public_read == "true") {
          publicProperties[property.name] = prop
        }
      }
      return publicProperties
    }

    destroy() {
      destroy self.ownedNFTs
    }
  }

  // public function that anyone can call to create a new empty collection
  pub fun createEmptyCollection(): @NonFungibleToken.Collection {
      return <- create Collection()
  }

  pub resource Administrator {
    pub fun createMojoNFTAdministrator(): @Administrator {
        return <- create Administrator()
    }
    pub fun resetContract() {
      MojoAsset.totalSupply = 0 as UInt64
      MojoAsset.totalMelt = 0 as UInt64
    }
  }

  pub resource PropertyWriter {
    pub let assetdefinitionId: Int

    init(assetdefinitionId: Int) {
      self.assetdefinitionId = assetdefinitionId
    }

    pub fun writeProperty(asset: &MojoAsset.NFT, propertyName: String, value: String ) {
      pre {
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.properties[propertyName]!.writer_mutable == "true" : "the property is not writable"
      }
      asset.properties[propertyName]?.setValue(value: value)
    }

    pub fun writePublicRead(asset: &MojoAsset.NFT, propertyName: String, value: String ) {
      pre {
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.properties[propertyName]!.public_read_mutable == "true" : "the property is not writable"
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
      }
      asset.properties[propertyName]?.setPublicRead(value: value)
    }
  }

  pub resource AssetMinter {
      pub let assetdefinitionId: Int
      pub let maxUse: Int8
      priv var numUse: Int8
      access(self) var reserveVault: @MojoToken.Vault

      pub fun mintAsset(receiverAddress: Address, properties: {String: {String: String}}?) {
        pre {
            self.numUse < self.maxUse : "cannot use minter anymore"
            self.reserveVault.balance >= MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.reserveMojoPerNFT! : "reserveFundsToLow"
        }

        let receiverRef = getAccount(receiverAddress).getCapability(/public/mojoAssetCollectionPublic)!.borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("NO NFT Receiver found")
        receiverRef.deposit(token: <- create NFT(assetdefinitionId: self.assetdefinitionId, reserveMojoTokenVault: <- (self.reserveVault.withdraw(amount: MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.reserveMojoPerNFT!) as! @MojoToken.Vault), properties: properties))
        self.numUse = self.numUse + 1 as Int8
      }

      init(assetdefinitionId: Int, maxUse: Int8, reserveMojoTokenVault: @MojoToken.Vault) {
          pre {
              maxUse > Int8(0) : "Max use too low"
              reserveMojoTokenVault.balance >= (MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.reserveMojoPerNFT! * UFix64(maxUse)): "Reserve not sufficient"
              //SUPPLYTYPES: 0: fixed - there can not be more than max, 1: collapsing - only maxSupply minted allowed (minus melts!),2:  infinite - can always be minted,3: settable - max can be changed
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType == MojoAssetdefinition.MintType.minterOnly
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType == MojoAssetdefinition.MintType.minterAndOwner
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType == MojoAssetdefinition.MintType.everyone
                : "Minters are not allowed for this asset"
          }
          self.reserveVault <- (MojoToken.createEmptyVault() as! @MojoToken.Vault)
          self.reserveVault.deposit(from: <- reserveMojoTokenVault)
          self.assetdefinitionId = assetdefinitionId
          self.maxUse = maxUse
          self.numUse = 0 as Int8
      }
      pub fun withdrawVault() :@MojoToken.Vault {
        pre {
          self.maxUse == self.numUse
            ||
          (MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.maxSupply == MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.totalSupply
            && MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.infinite)

            : "Minter not used now and maxSupply not reached"
        }
        return <- (self.reserveVault.withdraw(amount: self.reserveVault.balance) as! @MojoToken.Vault)
      }
      destroy() {
        destroy self.reserveVault
      }
  }

  pub resource AssetBurner {
      pub let assetdefinitionId: Int
      pub let maxUse: Int8
      priv var numUse: Int8

      pub fun burnAsset(asset: @NonFungibleToken.NFT) {
        pre {
            (self.maxUse < 0 as Int8 || self.numUse < self.maxUse) : "cannot use burner anymore"
        }
        let token <- asset as! @MojoAsset.NFT

        if(token.assetdefinitionId != self.assetdefinitionId) {
            panic("this burner is not allowed to burn that asset")
        }
        destroy token
        self.numUse = self.numUse + 1 as Int8
      }

      init(assetdefinitionId: Int, maxUse: Int8) {
          pre {
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.burnerOnly
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.burnerAndOwner
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.everyone
                  : "burners not allowed for this asset"
          }
          self.assetdefinitionId = assetdefinitionId
          self.maxUse = maxUse
          self.numUse = 0 as Int8
      }
  }
}

`),
		}
		var events []cadence.Event
		runtimeInterface := getMockedRuntimeInterfaceForTxUpdate(t, accountCode, events)
		nextTransactionLocation := newTransactionLocationGenerator()

		deployTx1 := newDeployTransaction(sema.AuthAccountContractsTypeAddFunctionName, name, oldCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTx1,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		deployTx2 := newDeployTransaction(sema.AuthAccountContractsTypeUpdateExperimentalFunctionName, name, newCode)
		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTx2,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		return err
	}

	t.Run("Mojo", func(t *testing.T) {
		const oldCode = `
import NonFungibleToken from 0x631e88ae7f1d7c20
import FungibleToken from 0x9a0766d93b6608b7
import MojoAssetdefinition from 0x73dd87ae00edff1e
import  MojoToken from 0x73dd87ae00edff1e
import MojoCommunityVault from 0x73dd87ae00edff1e
import FlowToken from 0x7e60df042a9c0868

pub contract MojoAsset: NonFungibleToken {

  pub var totalSupply: UInt64
  pub var totalMelt: UInt64
  pub var mintFeePercent: UFix64
  pub var burnFeePercent: UFix64

  pub event ContractInitialized()
  pub event Withdraw(id: UInt64, from: Address?)
  pub event Deposit(id: UInt64, to: Address?)
  pub event NFTBurnt(id: UInt64)

  init() {
    // Initialize the total supply
    self.totalSupply = 0
    self.totalMelt = 0
    self.mintFeePercent = UFix64(10)
    self.burnFeePercent = UFix64(10)
    // Create a Collection resource and save it to storage
    destroy self.account.load<@AnyResource>(from: /storage/mojoAssetCollection)
    self.account.save(<- create Collection(), to: /storage/mojoAssetCollection)

    // create a public capability for the collection
    self.account.link<&{NonFungibleToken.CollectionPublic,MojoAsset.MojoAssetCollectionPublic}>(
        /public/mojoCollectionPublic,
        target: /storage/mojoAssetCollection
    ) 

    // Create a Minter resource and save it to storage
    destroy self.account.load<@AnyResource>(from: /storage/mojoNFTAdministrator)
    self.account.save(<- create Administrator(), to: /storage/mojoNFTAdministrator)
    emit ContractInitialized()
  }
  
  pub resource interface MojoAssetCollectionPublic {
    pub fun getIDsByProjectId(projectId: Int): [UInt64]
    pub fun getIDsByAssetdefinitionId(assetdefinitionId: Int): [UInt64]
    pub fun getProjectIds(): [Int]
    pub fun getAssetdefinitionIds(): [Int]
    pub fun nftPropertiesPublic(id: UInt64): {String: {String: String}}
  }
  
  priv fun sendFees(feeVault: @MojoToken.Vault) {
    if(feeVault.balance > UFix64(0)) {
      MojoCommunityVault.depositFlow(vaultName: "mintFeeMojoVault", from: <- feeVault)
    }
    destroy feeVault
  }
  
  priv fun sendCreatorFees(creatorFeeVault: @MojoToken.Vault, creatorAddress: Address) {
    if(creatorFeeVault.balance > UFix64(0)) {
      let mojoVaultRef = getAccount(creatorAddress)
        .getCapability(/public/mojoTokenReceiver)!
        .borrow<&MojoToken.Vault{FungibleToken.Receiver}>()
      mojoVaultRef!.deposit(from: <- creatorFeeVault.withdraw(amount: creatorFeeVault.balance))
    }
    destroy creatorFeeVault
  }
  
  pub fun mintNFT(receiverAddress: Address, token: &MojoAssetdefinition.AssetdefinitionOwnertoken, reserveMojoTokenVault: @MojoToken.Vault, properties: {String: {String: String}}?) {
    // create a new NFT
    let receiverRef = getAccount(receiverAddress).getCapability(/public/mojoAssetCollectionPublic)!.borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("NO NFT Receiver found")
    
    receiverRef.deposit(token: <- create NFT(assetdefinitionId: token.assetdefinitionId, reserveMojoTokenVault: <-reserveMojoTokenVault, properties: properties))
  }
  
  pub fun createNFTMinter(maxUse: Int8, token: &MojoAssetdefinition.AssetdefinitionOwnertoken, reserveMojoTokenVault: @MojoToken.Vault): @AssetMinter {
    // create a new NFT
    return <- create AssetMinter(assetdefinitionId: token.assetdefinitionId, maxUse: maxUse, reserveMojoTokenVault: <- reserveMojoTokenVault)
  }
  
  pub fun createNFTBurner(maxUse: Int8, token: &MojoAssetdefinition.AssetdefinitionOwnertoken): @AssetBurner {
    // create a new NFT
    return <- create AssetBurner(assetdefinitionId: token.assetdefinitionId, maxUse: maxUse)
  }
  
  pub fun mintNFTPublic(receiverAddress: Address, assetdefinitionId: Int, reserveMojoTokenVault: @MojoToken.Vault, properties: {String: {String: String}}?) {
    // create a new NFT
    pre {
      MojoAssetdefinition.assetdefinitions[assetdefinitionId] != nil : "assetdefinitionId invalid"
      MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType! == MojoAssetdefinition.MintType.everyone : "This asset cannot be minted!"
    }
    let receiverRef = getAccount(receiverAddress).getCapability(/public/mojoAssetCollectionPublic)!.borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("NO NFT Receiver found")
    
    receiverRef.deposit(token: <- create NFT(assetdefinitionId: assetdefinitionId, reserveMojoTokenVault: <-reserveMojoTokenVault, properties: properties))
  }
  
  pub fun burnNFT(asset: @MojoAsset.NFT, token: &MojoAssetdefinition.AssetdefinitionOwnertoken) {
    // create a new NFT
    pre {
      asset.assetdefinitionId == token.assetdefinitionId : "no valid ownertoken"
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId] != nil : "assetdefinitionId invalid"
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId]!.burnType! == MojoAssetdefinition.BurnType.everyone
        || 
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId]!.burnType! == MojoAssetdefinition.BurnType.burnerAndOwner
        ||
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId]!.burnType! == MojoAssetdefinition.BurnType.ownerOnly
        : "not allowed to burn"
    }
    asset.setBurnValidated(validated: true)
    destroy asset
  }
  
  pub resource NFT: NonFungibleToken.INFT {
  
      access(contract) var burnValidated: Bool
      pub let id: UInt64
      pub let assetdefinitionId: Int
      pub let properties: {String: MojoAssetdefinition.AssetProperty}
      
      access(self) let mojoTokenVault: @MojoToken.Vault
      
      access(self) let mojoTokenVaults: @{String: MojoToken.Vault}
      access(self) let flowTokenVaults: @{String: FlowToken.Vault}
      access(self) let mojoAssets: @{String: MojoAsset.NFT}
      
      access(contract) fun setBurnValidated(validated: Bool) {
        self.burnValidated = validated
      }
      init(assetdefinitionId: Int, reserveMojoTokenVault: @MojoToken.Vault, properties: {String: {String:String}}? ) {
        pre {
          reserveMojoTokenVault.balance >= MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.reserveMojoPerNFT : "Not Enough MojoTokens For Reserve"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]! != nil : "assetdefinition does not exist"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.fixed || 
          (MojoAssetdefinition.assetdefinitions[assetdefinitionId] == nil || MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply < MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.maxSupply) : "maxsupply reached!"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.collapsing || (MojoAssetdefinition.assetdefinitions[assetdefinitionId] == nil || MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply + MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalMelt < MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.maxSupply) : "maxsupply reached!"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.infinite || true : "will never occure"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.settable || (MojoAssetdefinition.assetdefinitions[assetdefinitionId] == nil || MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply < MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.maxSupply) : "maxsupply reached!"
        }
        self.burnValidated = false
        self.id = MojoAsset.totalSupply + MojoAsset.totalMelt + UInt64(1)
        self.assetdefinitionId = assetdefinitionId
        self.properties = MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.properties
        // change name if its a fungible token
        if(MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.isFungible == false) {
          self.properties["name"]?.setValue(value: self.properties["name"]!.value.concat(" #").concat((MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply + MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalMelt + 1 as UInt64).toString()))
          for property in self.properties.values {
            if(property.value_type == "random") {
              self.properties[property.name]?.setValue(value: unsafeRandom().toString())
            }
          }
          if(properties != nil) {
            for property in properties!.values {
              if(self.properties![property["name"]!]!.owner_mutable == "true") {
                self.properties![property["name"]!]?.setValue(value: property["value"]!)
              }
            }
          }
        }
        self.mojoTokenVault <- MojoToken.createEmptyVault() as! @MojoToken.Vault
        self.mojoTokenVaults <- {}
        self.flowTokenVaults <- {}
        self.mojoAssets <- {}
        let feeAmount = reserveMojoTokenVault.balance * MojoAsset.mintFeePercent / UFix64(100)
        MojoAsset.sendFees(feeVault: <- (reserveMojoTokenVault.withdraw(amount: feeAmount) as! @MojoToken.Vault))
        let reserveAmount = reserveMojoTokenVault.balance
        self.mojoTokenVault.deposit(from: <- reserveMojoTokenVault)
        MojoAssetdefinition.assetdefinitions[assetdefinitionId]?.increaseTotalReserve(reserveChange: reserveAmount)
        MojoAssetdefinition.assetdefinitions[assetdefinitionId]?.increaseTotalSupply(addedSupply: UInt64(1))
        MojoAsset.totalSupply = MojoAsset.totalSupply + 1 as UInt64
      }
      pub fun writeProperty(propertyName: String, value: String ) {
        pre {
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.properties[propertyName]!.owner_mutable == "true" : "the property is not writable"
        }
        self.properties[propertyName]?.setValue(value: value)
      }
      pub fun writePublicRead(propertyName: String, value: String ) {
        pre {
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
        }
        self.properties[propertyName]?.setPublicRead(value: value)
      }
      destroy() {
        if(MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.indestructable ) {
          panic("this item is indestructable!")
        }
        if((MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnType != MojoAssetdefinition.BurnType.everyone) && self.burnValidated != true) {
          panic("not allowed to burn item")
        }
        let oldReserveBalance = self.mojoTokenVault.balance
        MojoAsset.sendFees(feeVault: <- (self.mojoTokenVault.withdraw(amount: (self.mojoTokenVault.balance * MojoAsset.burnFeePercent / UFix64(100))) as! @MojoToken.Vault))
        let creatorAddress = MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]! != nil ? MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.creator : getAccount(0x73dd87ae00edff1e).address
        var feeAmount = MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnFeeValue
        if(MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnFeeType == MojoAssetdefinition.BurnFeeType.percentage) {
          feeAmount = (feeAmount / UFix64(100)) * self.mojoTokenVault.balance
        }
        MojoAsset.sendCreatorFees(creatorFeeVault: <- (self.mojoTokenVault.withdraw(amount: feeAmount) as! @MojoToken.Vault), creatorAddress: creatorAddress)
        
        if(MojoAsset.totalSupply > UInt64(0)) {
          MojoAsset.totalSupply = MojoAsset.totalSupply - 1 as UInt64
        }
        MojoAsset.totalMelt = MojoAsset.totalMelt + 1 as UInt64
        MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]?.increaseTotalMelt(addedMelt: UInt64(1)) 
        MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]?.decreaseTotalReserve(reserveChange: oldReserveBalance)
        // pay back the frozen mojo
        destroy self.mojoTokenVault
        destroy self.mojoTokenVaults
        destroy self.flowTokenVaults
        destroy self.mojoAssets
        
      }
  }

  pub resource Collection: NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic, MojoAssetCollectionPublic {
    // dictionary of NFT conforming tokens
    // NFT is a resource type with an 'UInt64' ID field

    pub var ownedNFTs: @{UInt64: NonFungibleToken.NFT}
    // nftId: [projectId, assetDefinitionId]
    access(self) var projectMapping: {UInt64: [Int]}
    //access(self) var transferFeeVault: @MojoToken.Vault
    init () {
      self.ownedNFTs <- {}
      self.projectMapping = {}
    }
    // withdraw removes an NFT from the collection and moves it to the caller
    pub fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
      let mojoAsset = self.borrowMojoNFT(id: withdrawID)!
      let token <- self.ownedNFTs.remove(key: withdrawID)!
      
      if(MojoAssetdefinition.assetdefinitions[mojoAsset.assetdefinitionId]!.tradeType == MojoAssetdefinition.TradeType.never) {
        panic("this asset cannot be traded!")
      }
      emit Withdraw(id: mojoAsset.id, from: self.owner?.address)
      self.projectMapping.remove(key: mojoAsset.id)
      return <-token
    }
    pub fun withdrawMany(withdrawIDs: [UInt64]): @[NonFungibleToken.NFT] {
      let withdrawArray: @[NonFungibleToken.NFT] <- []
      for id in withdrawIDs {
        let token <- self.withdraw(withdrawID: id) as! @NonFungibleToken.NFT
        withdrawArray.append(<- token)
      }
      return <- withdrawArray
    }
    // deposit takes a NFT and adds it to the collections dictionary
    // and adds the ID to the id array
    pub fun deposit(token: @NonFungibleToken.NFT) {
      pre {
        token != nil : "There is no NFT to deposit "
      }
      let _token <- token as! @MojoAsset.NFT
      emit Deposit(id: _token.id, to: self.owner?.address)
      self.projectMapping[_token.id] = [MojoAssetdefinition.assetdefinitions[_token.assetdefinitionId]!.projectId, _token.assetdefinitionId]
      // add the new token to the dictionary which removes the old one
      self.ownedNFTs[_token.id] <-! _token
    }
    pub fun depositMany(tokens: @[NonFungibleToken.NFT]) {
      while(tokens.length > 0) {
        let token <- tokens.removeFirst() as! @MojoAsset.NFT
        emit Deposit(id: token.id, to: self.owner?.address)
        self.deposit(token: <- token)
      }
      destroy tokens
    }

    // getIDs returns an array of the IDs that are in the collection
    pub fun getIDs(): [UInt64] {
      return self.ownedNFTs.keys
    }

    // getIDs returns an array of the IDs that are in the collection
    pub fun getIDsByAssetdefinitionId(assetdefinitionId: Int): [UInt64] {
      let resultIds: [UInt64] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if((ref as! &MojoAsset.NFT).assetdefinitionId == assetdefinitionId) {
          resultIds.append(nftId)
        }
        
      }
      return resultIds
    }
    // getIDs returns an array of the IDs that are in the collection
    pub fun getIDsByProjectId(projectId: Int): [UInt64] {
      let resultIds: [UInt64] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if(MojoAssetdefinition.assetdefinitions[(ref as! &MojoAsset.NFT).assetdefinitionId!]?.projectId == projectId) {
          resultIds.append(nftId)
        }
      }
      return resultIds
    }
    
    // getIDs returns an array of the IDs that are in the collection
    pub fun getProjectIds(): [Int] {
      let resultIds: [Int] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if(!resultIds.contains(MojoAssetdefinition.assetdefinitions[(ref as! &MojoAsset.NFT).assetdefinitionId!]?.projectId!)) {
          resultIds.append(MojoAssetdefinition.assetdefinitions[(ref as! &MojoAsset.NFT).assetdefinitionId!]?.projectId!)
        }
      }
      return resultIds
    }
    
    // getIDs returns an array of the IDs that are in the collection
    pub fun getAssetdefinitionIds(): [Int] {
      let resultIds: [Int] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if(!resultIds.contains((ref as! &MojoAsset.NFT).assetdefinitionId!)) {
          resultIds.append((ref as! &MojoAsset.NFT).assetdefinitionId!)
        }
      }
      return resultIds
    }
    
    // borrowNFT gets a reference to an NFT in the collection
    // so that the caller can read its metadata and call its methods
    pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
      return &self.ownedNFTs[id] as! &NonFungibleToken.NFT
    }
    
    pub fun borrowMojoNFT(id: UInt64): &MojoAsset.NFT? {
        if self.ownedNFTs[id] != nil {
            let ref = &self.ownedNFTs[id] as auth &NonFungibleToken.NFT
            return ref as! &MojoAsset.NFT
        } else {
            return nil
        }
    }

    pub fun nftPropertiesPublic(id: UInt64): {String: {String: String}} {
      let publicProperties: {String: {String: String}} = {}
      let ref = &self.ownedNFTs[id] as auth &NonFungibleToken.NFT
      for property in (ref as! &MojoAsset.NFT).properties.values {
        var prop = property.toString()
        if(property.public_read == "true") {
          publicProperties[property.name] = prop
        }
      }
      return publicProperties
    }
    
    destroy() {
      destroy self.ownedNFTs
    }
  }

  // public function that anyone can call to create a new empty collection
  pub fun createEmptyCollection(): @NonFungibleToken.Collection {
      return <- create Collection()
  }

  pub resource Administrator {
    pub fun createMojoNFTAdministrator(): @Administrator {
        return <- create Administrator()
    }
    pub fun resetContract() {
      MojoAsset.totalSupply = 0 as UInt64
      MojoAsset.totalMelt = 0 as UInt64
    }
  }
  
  pub resource PropertyWriter {
    pub let assetdefinitionId: Int
    
    init(assetdefinitionId: Int) {
      self.assetdefinitionId = assetdefinitionId
    }
    
    pub fun writeProperty(asset: &MojoAsset.NFT, propertyName: String, value: String ) {
      pre {
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.properties[propertyName]!.writer_mutable == "true" : "the property is not writable"
      }
      asset.properties[propertyName]?.setValue(value: value)
    }
    
    pub fun writePublicRead(asset: &MojoAsset.NFT, propertyName: String, value: String ) {
      pre {
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.properties[propertyName]!.public_read_mutable == "true" : "the property is not writable"
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
      }
      asset.properties[propertyName]?.setPublicRead(value: value)
    }
  }

  pub resource AssetMinter {
      pub let assetdefinitionId: Int
      pub let maxUse: Int8
      priv var numUse: Int8
      access(self) var reserveVault: @MojoToken.Vault

      pub fun mintAsset(receiverAddress: Address, properties: {String: {String: String}}?) {
        pre {
            self.numUse < self.maxUse : "cannot use minter anymore"
            self.reserveVault.balance >= MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.reserveMojoPerNFT! : "reserveFundsToLow"
        }
        
        let receiverRef = getAccount(receiverAddress).getCapability(/public/mojoAssetCollectionPublic)!.borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("NO NFT Receiver found")
        receiverRef.deposit(token: <- create NFT(assetdefinitionId: self.assetdefinitionId, reserveMojoTokenVault: <- (self.reserveVault.withdraw(amount: MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.reserveMojoPerNFT!) as! @MojoToken.Vault), properties: properties))
        self.numUse = self.numUse + 1 as Int8
      }

      init(assetdefinitionId: Int, maxUse: Int8, reserveMojoTokenVault: @MojoToken.Vault) {
          pre {
              maxUse > Int8(0) : "Max use too low"
              reserveMojoTokenVault.balance >= (MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.reserveMojoPerNFT! * UFix64(maxUse)): "Reserve not sufficient"
              //SUPPLYTYPES: 0: fixed - there can not be more than max, 1: collapsing - only maxSupply minted allowed (minus melts!),2:  infinite - can always be minted,3: settable - max can be changed
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType == MojoAssetdefinition.MintType.minterOnly
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType == MojoAssetdefinition.MintType.minterAndOwner
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType == MojoAssetdefinition.MintType.everyone
                : "Minters are not allowed for this asset"
          }
          self.reserveVault <- (MojoToken.createEmptyVault() as! @MojoToken.Vault)
          self.reserveVault.deposit(from: <- reserveMojoTokenVault)
          self.assetdefinitionId = assetdefinitionId
          self.maxUse = maxUse
          self.numUse = 0 as Int8
      }
      pub fun withdrawVault() :@MojoToken.Vault {
        pre {
          self.maxUse == self.numUse
            ||
          (MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.maxSupply == MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.totalSupply
            && MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.infinite)
          
            : "Minter not used now and maxSupply not reached"
        }
        return <- (self.reserveVault.withdraw(amount: self.reserveVault.balance) as! @MojoToken.Vault)
      }
      destroy() {
        destroy self.reserveVault
      }
  }

  pub resource AssetBurner {
      pub let assetdefinitionId: Int
      pub let maxUse: Int8
      priv var numUse: Int8

      pub fun burnAsset(asset: @NonFungibleToken.NFT) {
        pre {
            (self.maxUse < 0 as Int8 || self.numUse < self.maxUse) : "cannot use burner anymore"
        }
        let token <- asset as! @MojoAsset.NFT
        
        if(token.assetdefinitionId != self.assetdefinitionId) {
            panic("this burner is not allowed to burn that asset")
        }
        destroy token
        self.numUse = self.numUse + 1 as Int8
      }

      init(assetdefinitionId: Int, maxUse: Int8) {
          pre {
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.burnerOnly
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.burnerAndOwner
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.everyone
                  : "burners not allowed for this asset"
          }
          self.assetdefinitionId = assetdefinitionId
          self.maxUse = maxUse
          self.numUse = 0 as Int8
      }
  }
}
`

		const newCode = `

import NonFungibleToken from 0x631e88ae7f1d7c20
import FungibleToken from 0x9a0766d93b6608b7
import MojoAssetdefinition from 0x73dd87ae00edff1e
import  MojoToken from 0x73dd87ae00edff1e
import FlowToken from 0x7e60df042a9c0868

pub contract MojoAsset: NonFungibleToken {

  pub var totalSupply: UInt64
  pub var totalMelt: UInt64
  pub var mintFeePercent: UFix64
  pub var burnFeePercent: UFix64

  pub event ContractInitialized()
  pub event Withdraw(id: UInt64, from: Address?)
  pub event Deposit(id: UInt64, to: Address?)
  pub event NFTBurnt(id: UInt64)

  init() {
    // Initialize the total supply
    self.totalSupply = 0
    self.totalMelt = 0
    self.mintFeePercent = UFix64(10)
    self.burnFeePercent = UFix64(10)
    // Create a Collection resource and save it to storage
    destroy self.account.load<@AnyResource>(from: /storage/mojoAssetCollection)
    self.account.save(<- create Collection(), to: /storage/mojoAssetCollection)

    // create a public capability for the collection
    self.account.link<&{NonFungibleToken.CollectionPublic,MojoAsset.MojoAssetCollectionPublic}>(
        /public/mojoCollectionPublic,
        target: /storage/mojoAssetCollection
    ) 

    // Create a Minter resource and save it to storage
    destroy self.account.load<@AnyResource>(from: /storage/mojoNFTAdministrator)
    self.account.save(<- create Administrator(), to: /storage/mojoNFTAdministrator)
    emit ContractInitialized()
  }
  
  pub resource interface MojoAssetCollectionPublic {
    pub fun getIDsByProjectId(projectId: Int): [UInt64]
    pub fun getIDsByAssetdefinitionId(assetdefinitionId: Int): [UInt64]
    pub fun getProjectIds(): [Int]
    pub fun getAssetdefinitionIds(): [Int]
    pub fun nftPropertiesPublic(id: UInt64): {String: {String: String}}
  }
  
  priv fun sendFees(feeVault: @MojoToken.Vault) {
    if(feeVault.balance > UFix64(0)) {
      let mojoVaultRef = getAccount(self.account.address)
        .getCapability(/public/mojoTokenReceiver)!
        .borrow<&MojoToken.Vault{FungibleToken.Receiver}>()
      mojoVaultRef!.deposit(from: <- feeVault.withdraw(amount: feeVault.balance))
    }
    destroy feeVault
    
  }
  
  priv fun sendCreatorFees(creatorFeeVault: @MojoToken.Vault, creatorAddress: Address) {
    if(creatorFeeVault.balance > UFix64(0)) {
      let mojoVaultRef = getAccount(creatorAddress)
        .getCapability(/public/mojoTokenReceiver)!
        .borrow<&MojoToken.Vault{FungibleToken.Receiver}>()
      mojoVaultRef!.deposit(from: <- creatorFeeVault.withdraw(amount: creatorFeeVault.balance))
    }
    destroy creatorFeeVault
  }
  
  pub fun mintNFT(receiverAddress: Address, token: &MojoAssetdefinition.AssetdefinitionOwnertoken, reserveMojoTokenVault: @MojoToken.Vault, properties: {String: {String: String}}?) {
    // create a new NFT
    let receiverRef = getAccount(receiverAddress).getCapability(/public/mojoAssetCollectionPublic)!.borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("NO NFT Receiver found")
    
    receiverRef.deposit(token: <- create NFT(assetdefinitionId: token.assetdefinitionId, reserveMojoTokenVault: <-reserveMojoTokenVault, properties: properties))
  }
  
  pub fun createNFTMinter(maxUse: Int8, token: &MojoAssetdefinition.AssetdefinitionOwnertoken, reserveMojoTokenVault: @MojoToken.Vault): @AssetMinter {
    // create a new NFT
    return <- create AssetMinter(assetdefinitionId: token.assetdefinitionId, maxUse: maxUse, reserveMojoTokenVault: <- reserveMojoTokenVault)
  }
  
  pub fun createNFTBurner(maxUse: Int8, token: &MojoAssetdefinition.AssetdefinitionOwnertoken): @AssetBurner {
    // create a new NFT
    return <- create AssetBurner(assetdefinitionId: token.assetdefinitionId, maxUse: maxUse)
  }
  
  pub fun mintNFTPublic(receiverAddress: Address, assetdefinitionId: Int, reserveMojoTokenVault: @MojoToken.Vault, properties: {String: {String: String}}?) {
    // create a new NFT
    pre {
      MojoAssetdefinition.assetdefinitions[assetdefinitionId] != nil : "assetdefinitionId invalid"
      MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType! == MojoAssetdefinition.MintType.everyone : "This asset cannot be minted!"
    }
    let receiverRef = getAccount(receiverAddress).getCapability(/public/mojoAssetCollectionPublic)!.borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("NO NFT Receiver found")
    
    receiverRef.deposit(token: <- create NFT(assetdefinitionId: assetdefinitionId, reserveMojoTokenVault: <-reserveMojoTokenVault, properties: properties))
  }
  
  pub fun burnNFT(asset: @MojoAsset.NFT, token: &MojoAssetdefinition.AssetdefinitionOwnertoken) {
    // create a new NFT
    pre {
      asset.assetdefinitionId == token.assetdefinitionId : "no valid ownertoken"
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId] != nil : "assetdefinitionId invalid"
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId]!.burnType! == MojoAssetdefinition.BurnType.everyone
        || 
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId]!.burnType! == MojoAssetdefinition.BurnType.burnerAndOwner
        ||
      MojoAssetdefinition.assetdefinitions[token.assetdefinitionId]!.burnType! == MojoAssetdefinition.BurnType.ownerOnly
        : "not allowed to burn"
    }
    asset.setBurnValidated(validated: true)
    destroy asset
  }
  
  pub resource NFT: NonFungibleToken.INFT {
  
      access(contract) var burnValidated: Bool
      pub let id: UInt64
      pub let assetdefinitionId: Int
      pub let properties: {String: MojoAssetdefinition.AssetProperty}
      
      access(self) let mojoTokenVault: @MojoToken.Vault
      
      access(self) let mojoTokenVaults: @{String: MojoToken.Vault}
      access(self) let flowTokenVaults: @{String: FlowToken.Vault}
      access(self) let mojoAssets: @{String: MojoAsset.NFT}
      
      access(contract) fun setBurnValidated(validated: Bool) {
        self.burnValidated = validated
      }
      init(assetdefinitionId: Int, reserveMojoTokenVault: @MojoToken.Vault, properties: {String: {String:String}}? ) {
        pre {
          reserveMojoTokenVault.balance >= MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.reserveMojoPerNFT : "Not Enough MojoTokens For Reserve"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]! != nil : "assetdefinition does not exist"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.fixed || 
          (MojoAssetdefinition.assetdefinitions[assetdefinitionId] == nil || MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply < MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.maxSupply) : "maxsupply reached!"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.collapsing || (MojoAssetdefinition.assetdefinitions[assetdefinitionId] == nil || MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply + MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalMelt < MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.maxSupply) : "maxsupply reached!"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.infinite || true : "will never occure"
          MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.settable || (MojoAssetdefinition.assetdefinitions[assetdefinitionId] == nil || MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply < MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.maxSupply) : "maxsupply reached!"
        }
        self.burnValidated = false
        self.id = MojoAsset.totalSupply + MojoAsset.totalMelt + UInt64(1)
        self.assetdefinitionId = assetdefinitionId
        self.properties = MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.properties
        // change name if its a fungible token
        if(MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.isFungible == false) {
          self.properties["name"]?.setValue(value: self.properties["name"]!.value.concat(" #").concat((MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalSupply + MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.totalMelt + 1 as UInt64).toString()))
          for property in self.properties.values {
            if(property.value_type == "random") {
              self.properties[property.name]?.setValue(value: unsafeRandom().toString())
            }
          }
          if(properties != nil) {
            for property in properties!.values {
              if(self.properties![property["name"]!]!.owner_mutable == "true") {
                self.properties![property["name"]!]?.setValue(value: property["value"]!)
              }
            }
          }
        }
        self.mojoTokenVault <- MojoToken.createEmptyVault() as! @MojoToken.Vault
        self.mojoTokenVaults <- {}
        self.flowTokenVaults <- {}
        self.mojoAssets <- {}
        let feeAmount = reserveMojoTokenVault.balance * MojoAsset.mintFeePercent / UFix64(100)
        MojoAsset.sendFees(feeVault: <- (reserveMojoTokenVault.withdraw(amount: feeAmount) as! @MojoToken.Vault))
        let reserveAmount = reserveMojoTokenVault.balance
        self.mojoTokenVault.deposit(from: <- reserveMojoTokenVault)
        MojoAssetdefinition.assetdefinitions[assetdefinitionId]?.increaseTotalReserve(reserveChange: reserveAmount)
        MojoAssetdefinition.assetdefinitions[assetdefinitionId]?.increaseTotalSupply(addedSupply: UInt64(1))
        MojoAsset.totalSupply = MojoAsset.totalSupply + 1 as UInt64
      }
      pub fun writeProperty(propertyName: String, value: String ) {
        pre {
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.properties[propertyName]!.owner_mutable == "true" : "the property is not writable"
        }
        self.properties[propertyName]?.setValue(value: value)
      }
      pub fun writePublicRead(propertyName: String, value: String ) {
        pre {
          MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
        }
        self.properties[propertyName]?.setPublicRead(value: value)
      }
      destroy() {
        if(MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.indestructable ) {
          panic("this item is indestructable!")
        }
        if((MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnType != MojoAssetdefinition.BurnType.everyone) && self.burnValidated != true) {
          panic("not allowed to burn item")
        }
        let oldReserveBalance = self.mojoTokenVault.balance
        MojoAsset.sendFees(feeVault: <- (self.mojoTokenVault.withdraw(amount: (self.mojoTokenVault.balance * MojoAsset.burnFeePercent / UFix64(100))) as! @MojoToken.Vault))
        let creatorAddress = MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]! != nil ? MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.creator : getAccount(0x73dd87ae00edff1e).address
        var feeAmount = MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnFeeValue
        if(MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.burnFeeType == MojoAssetdefinition.BurnFeeType.percentage) {
          feeAmount = (feeAmount / UFix64(100)) * self.mojoTokenVault.balance
        }
        MojoAsset.sendCreatorFees(creatorFeeVault: <- (self.mojoTokenVault.withdraw(amount: feeAmount) as! @MojoToken.Vault), creatorAddress: creatorAddress)
        
        if(MojoAsset.totalSupply > UInt64(0)) {
          MojoAsset.totalSupply = MojoAsset.totalSupply - 1 as UInt64
        }
        MojoAsset.totalMelt = MojoAsset.totalMelt + 1 as UInt64
        MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]?.increaseTotalMelt(addedMelt: UInt64(1)) 
        MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]?.decreaseTotalReserve(reserveChange: oldReserveBalance)
        // pay back the frozen mojo
        destroy self.mojoTokenVault
        destroy self.mojoTokenVaults
        destroy self.flowTokenVaults
        destroy self.mojoAssets
        
      }
  }

  pub resource Collection: NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic, MojoAssetCollectionPublic {
    // dictionary of NFT conforming tokens
    // NFT is a resource type with an 'UInt64'' ID field

    pub var ownedNFTs: @{UInt64: NonFungibleToken.NFT}
    // nftId: [projectId, assetDefinitionId]
    access(self) var projectMapping: {UInt64: [Int]}
    //access(self) var transferFeeVault: @MojoToken.Vault
    init () {
      self.ownedNFTs <- {}
      self.projectMapping = {}
    }
    // withdraw removes an NFT from the collection and moves it to the caller
    pub fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
      let mojoAsset = self.borrowMojoNFT(id: withdrawID)!
      let token <- self.ownedNFTs.remove(key: withdrawID)!
      
      if(MojoAssetdefinition.assetdefinitions[mojoAsset.assetdefinitionId]!.tradeType == MojoAssetdefinition.TradeType.never) {
        panic("this asset cannot be traded!")
      }
      emit Withdraw(id: mojoAsset.id, from: self.owner?.address)
      self.projectMapping.remove(key: mojoAsset.id)
      return <-token
    }
    pub fun withdrawMany(withdrawIDs: [UInt64]): @[NonFungibleToken.NFT] {
      let withdrawArray: @[NonFungibleToken.NFT] <- []
      for id in withdrawIDs {
        let token <- self.withdraw(withdrawID: id) as! @NonFungibleToken.NFT
        withdrawArray.append(<- token)
      }
      return <- withdrawArray
    }
    // deposit takes a NFT and adds it to the collections dictionary
    // and adds the ID to the id array
    pub fun deposit(token: @NonFungibleToken.NFT) {
      pre {
        token != nil : "There is no NFT to deposit "
      }
      let _token <- token as! @MojoAsset.NFT
      emit Deposit(id: _token.id, to: self.owner?.address)
      self.projectMapping[_token.id] = [MojoAssetdefinition.assetdefinitions[_token.assetdefinitionId]!.projectId, _token.assetdefinitionId]
      // add the new token to the dictionary which removes the old one
      self.ownedNFTs[_token.id] <-! _token
    }
    pub fun depositMany(tokens: @[NonFungibleToken.NFT]) {
      while(tokens.length > 0) {
        let token <- tokens.removeFirst() as! @MojoAsset.NFT
        emit Deposit(id: token.id, to: self.owner?.address)
        self.deposit(token: <- token)
      }
      destroy tokens
    }

    // getIDs returns an array of the IDs that are in the collection
    pub fun getIDs(): [UInt64] {
      return self.ownedNFTs.keys
    }

    // getIDs returns an array of the IDs that are in the collection
    pub fun getIDsByAssetdefinitionId(assetdefinitionId: Int): [UInt64] {
      let resultIds: [UInt64] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if((ref as! &MojoAsset.NFT).assetdefinitionId == assetdefinitionId) {
          resultIds.append(nftId)
        }
        
      }
      return resultIds
    }
    // getIDs returns an array of the IDs that are in the collection
    pub fun getIDsByProjectId(projectId: Int): [UInt64] {
      let resultIds: [UInt64] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if(MojoAssetdefinition.assetdefinitions[(ref as! &MojoAsset.NFT).assetdefinitionId!]?.projectId == projectId) {
          resultIds.append(nftId)
        }
      }
      return resultIds
    }
    
    // getIDs returns an array of the IDs that are in the collection
    pub fun getProjectIds(): [Int] {
      let resultIds: [Int] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if(!resultIds.contains(MojoAssetdefinition.assetdefinitions[(ref as! &MojoAsset.NFT).assetdefinitionId!]?.projectId!)) {
          resultIds.append(MojoAssetdefinition.assetdefinitions[(ref as! &MojoAsset.NFT).assetdefinitionId!]?.projectId!)
        }
      }
      return resultIds
    }
    
    // getIDs returns an array of the IDs that are in the collection
    pub fun getAssetdefinitionIds(): [Int] {
      let resultIds: [Int] = []
      for nftId in self.ownedNFTs.keys {
        let ref = &self.ownedNFTs[nftId] as auth &NonFungibleToken.NFT
        if(!resultIds.contains((ref as! &MojoAsset.NFT).assetdefinitionId!)) {
          resultIds.append((ref as! &MojoAsset.NFT).assetdefinitionId!)
        }
      }
      return resultIds
    }
    
    // borrowNFT gets a reference to an NFT in the collection
    // so that the caller can read its metadata and call its methods
    pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
      return &self.ownedNFTs[id] as! &NonFungibleToken.NFT
    }
    
    pub fun borrowMojoNFT(id: UInt64): &MojoAsset.NFT? {
        if self.ownedNFTs[id] != nil {
            let ref = &self.ownedNFTs[id] as auth &NonFungibleToken.NFT
            return ref as! &MojoAsset.NFT
        } else {
            return nil
        }
    }

    pub fun nftPropertiesPublic(id: UInt64): {String: {String: String}} {
      let publicProperties: {String: {String: String}} = {}
      let ref = &self.ownedNFTs[id] as auth &NonFungibleToken.NFT
      for property in (ref as! &MojoAsset.NFT).properties.values {
        var prop = property.toString()
        if(property.public_read == "true") {
          publicProperties[property.name] = prop
        }
      }
      return publicProperties
    }
    
    destroy() {
      destroy self.ownedNFTs
    }
  }

  // public function that anyone can call to create a new empty collection
  pub fun createEmptyCollection(): @NonFungibleToken.Collection {
      return <- create Collection()
  }

  pub resource Administrator {
    pub fun createMojoNFTAdministrator(): @Administrator {
        return <- create Administrator()
    }
    pub fun resetContract() {
      MojoAsset.totalSupply = 0 as UInt64
      MojoAsset.totalMelt = 0 as UInt64
    }
  }
  
  pub resource PropertyWriter {
    pub let assetdefinitionId: Int
    
    init(assetdefinitionId: Int) {
      self.assetdefinitionId = assetdefinitionId
    }
    
    pub fun writeProperty(asset: &MojoAsset.NFT, propertyName: String, value: String ) {
      pre {
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.properties[propertyName]!.writer_mutable == "true" : "the property is not writable"
      }
      asset.properties[propertyName]?.setValue(value: value)
    }
    
    pub fun writePublicRead(asset: &MojoAsset.NFT, propertyName: String, value: String ) {
      pre {
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.properties[propertyName]!.public_read_mutable == "true" : "the property is not writable"
        MojoAssetdefinition.assetdefinitions[asset.assetdefinitionId]!.isFungible == false: "fungible assets can not be changed"
      }
      asset.properties[propertyName]?.setPublicRead(value: value)
    }
  }

  pub resource AssetMinter {
      pub let assetdefinitionId: Int
      pub let maxUse: Int8
      priv var numUse: Int8
      access(self) var reserveVault: @MojoToken.Vault

      pub fun mintAsset(receiverAddress: Address, properties: {String: {String: String}}?) {
        pre {
            self.numUse < self.maxUse : "cannot use minter anymore"
            self.reserveVault.balance >= MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.reserveMojoPerNFT! : "reserveFundsToLow"
        }
        
        let receiverRef = getAccount(receiverAddress).getCapability(/public/mojoAssetCollectionPublic)!.borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("NO NFT Receiver found")
        receiverRef.deposit(token: <- create NFT(assetdefinitionId: self.assetdefinitionId, reserveMojoTokenVault: <- (self.reserveVault.withdraw(amount: MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.reserveMojoPerNFT!) as! @MojoToken.Vault), properties: properties))
        self.numUse = self.numUse + 1 as Int8
      }

      init(assetdefinitionId: Int, maxUse: Int8, reserveMojoTokenVault: @MojoToken.Vault) {
          pre {
              maxUse > Int8(0) : "Max use too low"
              reserveMojoTokenVault.balance >= (MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.reserveMojoPerNFT! * UFix64(maxUse)): "Reserve not sufficient"
              //SUPPLYTYPES: 0: fixed - there can not be more than max, 1: collapsing - only maxSupply minted allowed (minus melts!),2:  infinite - can always be minted,3: settable - max can be changed
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType == MojoAssetdefinition.MintType.minterOnly
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType == MojoAssetdefinition.MintType.minterAndOwner
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.mintType == MojoAssetdefinition.MintType.everyone
                : "Minters are not allowed for this asset"
          }
          self.reserveVault <- (MojoToken.createEmptyVault() as! @MojoToken.Vault)
          self.reserveVault.deposit(from: <- reserveMojoTokenVault)
          self.assetdefinitionId = assetdefinitionId
          self.maxUse = maxUse
          self.numUse = 0 as Int8
      }
      pub fun withdrawVault() :@MojoToken.Vault {
        pre {
          self.maxUse == self.numUse
            ||
          (MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.maxSupply == MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.totalSupply
            && MojoAssetdefinition.assetdefinitions[self.assetdefinitionId]!.supplyType != MojoAssetdefinition.SupplyType.infinite)
          
            : "Minter not used now and maxSupply not reached"
        }
        return <- (self.reserveVault.withdraw(amount: self.reserveVault.balance) as! @MojoToken.Vault)
      }
      destroy() {
        destroy self.reserveVault
      }
  }

  pub resource AssetBurner {
      pub let assetdefinitionId: Int
      pub let maxUse: Int8
      priv var numUse: Int8

      pub fun burnAsset(asset: @NonFungibleToken.NFT) {
        pre {
            (self.maxUse < 0 as Int8 || self.numUse < self.maxUse) : "cannot use burner anymore"
        }
        let token <- asset as! @MojoAsset.NFT
        
        if(token.assetdefinitionId != self.assetdefinitionId) {
            panic("this burner is not allowed to burn that asset")
        }
        destroy token
        self.numUse = self.numUse + 1 as Int8
      }

      init(assetdefinitionId: Int, maxUse: Int8) {
          pre {
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.burnerOnly
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.burnerAndOwner
                  ||
              MojoAssetdefinition.assetdefinitions[assetdefinitionId]!.burnType == MojoAssetdefinition.BurnType.everyone
                  : "burners not allowed for this asset"
          }
          self.assetdefinitionId = assetdefinitionId
          self.maxUse = maxUse
          self.numUse = 0 as Int8
      }
  }
}
`

		err := deployAndUpdate(t, "MojoAsset", oldCode, newCode)
		require.Error(t, err)

		//cause := getErrorCause(t, err, "Test1")
		//assertFieldTypeMismatchError(t, cause, "Test1", "a", "String", "Int")
	})
}

func assertDeclTypeChangeError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	oldKind common.DeclarationKind,
	newKind common.DeclarationKind,
) {

	require.Error(t, err)
	require.IsType(t, &InvalidDeclarationKindChangeError{}, err)
	declTypeChangeError := err.(*InvalidDeclarationKindChangeError)
	assert.Equal(
		t,
		fmt.Sprintf("trying to convert %s `%s` to a %s", oldKind.Name(), erroneousDeclName, newKind.Name()),
		declTypeChangeError.Error(),
	)
}

func assertExtraneousFieldError(t *testing.T, err error, erroneousDeclName string, fieldName string) {
	require.Error(t, err)
	require.IsType(t, &ExtraneousFieldError{}, err)
	extraFieldError := err.(*ExtraneousFieldError)
	assert.Equal(t, fmt.Sprintf("found new field `%s` in `%s`", fieldName, erroneousDeclName), extraFieldError.Error())
}

func assertFieldTypeMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	fieldName string,
	expectedType string,
	foundType string,
) {

	require.Error(t, err)
	require.IsType(t, &FieldMismatchError{}, err)
	fieldMismatchError := err.(*FieldMismatchError)
	assert.Equal(
		t,
		fmt.Sprintf("mismatching field `%s` in `%s`", fieldName, erroneousDeclName),
		fieldMismatchError.Error(),
	)

	assert.IsType(t, &TypeMismatchError{}, fieldMismatchError.err)
	assert.Equal(
		t,
		fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`", expectedType, foundType),
		fieldMismatchError.err.Error(),
	)
}

func assertConformanceMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	expectedType string,
	foundType string,
) {

	require.Error(t, err)
	require.IsType(t, &ConformanceMismatchError{}, err)
	conformanceMismatchError := err.(*ConformanceMismatchError)
	assert.Equal(
		t,
		fmt.Sprintf("conformances does not match in `%s`", erroneousDeclName),
		conformanceMismatchError.Error(),
	)

	assert.IsType(t, &TypeMismatchError{}, conformanceMismatchError.err)
	assert.Equal(
		t,
		fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`", expectedType, foundType),
		conformanceMismatchError.err.Error(),
	)
}

func getErrorCause(t *testing.T, err error, contractName string) error {
	updateErr := getContractUpdateError(t, err)
	assert.Equal(t, fmt.Sprintf("cannot update contract `%s`", contractName), updateErr.Error())

	require.Equal(t, 1, len(updateErr.ChildErrors()))
	childError := updateErr.ChildErrors()[0]

	return childError
}

func getContractUpdateError(t *testing.T, err error) *ContractUpdateError {
	require.Error(t, err)
	require.IsType(t, Error{}, err)
	runtimeError := err.(Error)

	require.IsType(t, interpreter.Error{}, runtimeError.Err)
	interpreterError := runtimeError.Err.(interpreter.Error)

	require.IsType(t, &InvalidContractDeploymentError{}, interpreterError.Err)
	deploymentError := interpreterError.Err.(*InvalidContractDeploymentError)

	require.IsType(t, &ContractUpdateError{}, deploymentError.Err)
	return deploymentError.Err.(*ContractUpdateError)
}

func getMockedRuntimeInterfaceForTxUpdate(
	t *testing.T,
	accountCodes map[common.LocationID][]byte,
	events []cadence.Event,
) *testRuntimeInterface {

	return &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location.ID()], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.BytesToAddress([]byte{0x42})}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location.ID()], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) (err error) {
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
	}
}

func TestContractUpdateValidationDisabled(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime(
		WithContractUpdateValidationEnabled(false),
	)

	newDeployTransaction := func(function, name, code string) []byte {
		return []byte(fmt.Sprintf(`
			transaction {
				prepare(signer: AuthAccount) {
					signer.contracts.%s(name: "%s", code: "%s".decodeHex())
				}
			}`,
			function,
			name,
			hex.EncodeToString([]byte(code)),
		))
	}

	accountCode := map[common.LocationID][]byte{}
	var events []cadence.Event
	runtimeInterface := getMockedRuntimeInterfaceForTxUpdate(t, accountCode, events)
	nextTransactionLocation := newTransactionLocationGenerator()

	deployAndUpdate := func(t *testing.T, name string, oldCode string, newCode string) error {
		deployTx1 := newDeployTransaction(sema.AuthAccountContractsTypeAddFunctionName, name, oldCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTx1,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		deployTx2 := newDeployTransaction(sema.AuthAccountContractsTypeUpdateExperimentalFunctionName, name, newCode)
		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTx2,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		return err
	}

	t.Run("change field type", func(t *testing.T) {
		const oldCode = `
			pub contract Test1 {
				pub var a: String
				init() {
					self.a = "hello"
				}
      		}`

		const newCode = `
			pub contract Test1 {
				pub var a: Int
				init() {
					self.a = 0
				}
			}`

		err := deployAndUpdate(t, "Test1", oldCode, newCode)
		require.NoError(t, err)
	})
}
