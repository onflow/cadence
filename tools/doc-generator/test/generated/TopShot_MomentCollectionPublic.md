# Resource Interface `MomentCollectionPublic`

```cadence
resource interface MomentCollectionPublic {
}
```

 This is the interface that users can cast their Moment Collection as
 to allow others to deposit Moments into their Collection. It also allows for reading
 the IDs of Moments in the Collection.

## Functions


### fun `deposit()`

```cadence
func deposit(token NonFungibleToken.NFT):  
```



---

### fun `batchDeposit()`

```cadence
func batchDeposit(tokens NonFungibleToken.Collection):  
```



---

### fun `getIDs()`

```cadence
func getIDs(): [UInt64] 
```



---

### fun `borrowNFT()`

```cadence
func borrowNFT(id UInt64): &NonFungibleToken.NFT 
```



---

### fun `borrowMoment()`

```cadence
func borrowMoment(id UInt64): &TopShot.NFT? 
```



---


