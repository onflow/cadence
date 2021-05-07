# resource `Collection`

```
resource Collection {
    ownedNFTs {UInt64: NonFungibleToken.NFT}
}
```

 Collection is a resource that every user who owns NFTs
 will store in their account to manage their NFTS


## Functions:


### fun `withdraw()`

```
func withdraw(withdrawID UInt64): NonFungibleToken.NFT 
```

 withdraw removes an Moment from the Collection and moves it to the caller

 Parameters: withdrawID: The ID of the NFT
 that is to be removed from the Collection

 returns: @NonFungibleToken.NFT the token that was withdrawn

---

### fun `batchWithdraw()`

```
func batchWithdraw(ids [UInt64]): NonFungibleToken.Collection 
```

 batchWithdraw withdraws multiple tokens and returns them as a Collection

 Parameters: ids: An array of IDs to withdraw

 Returns: @NonFungibleToken.Collection: A collection that contains
                                        the withdrawn moments


---

### fun `deposit()`

```
func deposit(token NonFungibleToken.NFT)
```

 deposit takes a Moment and adds it to the Collections dictionary

 Paramters: token: the NFT to be deposited in the collection


---

### fun `batchDeposit()`

```
func batchDeposit(tokens NonFungibleToken.Collection)
```

 batchDeposit takes a Collection object as an argument
 and deposits each contained NFT into this Collection

---

### fun `getIDs()`

```
func getIDs(): [UInt64] 
```

 getIDs returns an array of the IDs that are in the Collection

---

### fun `borrowNFT()`

```
func borrowNFT(id UInt64): &NonFungibleToken.NFT 
```

 borrowNFT Returns a borrowed reference to a Moment in the Collection
 so that the caller can read its ID

 Parameters: id: The ID of the NFT to get the reference for

 Returns: A reference to the NFT

 Note: This only allows the caller to read the ID of the NFT,
 not any topshot specific data. Please use borrowMoment to
 read Moment data.


---

### fun `borrowMoment()`

```
func borrowMoment(id UInt64): &TopShot.NFT? 
```

 borrowMoment returns a borrowed reference to a Moment
 so that the caller can read data and call methods from it.
 They can use this to read its setID, playID, serialNumber,
 or any of the setData or Play data associated with it by
 getting the setID or playID and reading those fields from
 the smart contract.

 Parameters: id: The ID of the NFT to get the reference for

 Returns: A reference to the NFT

---


