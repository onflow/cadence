# contract `TopShot`

```
contract TopShot {
    currentSeries UInt32
    playDatas {UInt32: Play}
    setDatas {UInt32: SetData}
    sets {UInt32: Set}
    nextPlayID UInt32
    nextSetID UInt32
    totalSupply UInt64
}
```



## Functions:


### fun `createEmptyCollection()`

```
func createEmptyCollection(): NonFungibleToken.Collection 
```

 -----------------------------------------------------------------------
 TopShot contract-level function definitions
 -----------------------------------------------------------------------
 createEmptyCollection creates a new, empty Collection object so that
 a user can store it in their account storage.
 Once they have a Collection in their storage, they are able to receive
 Moments in transactions.


---

### fun `getAllPlays()`

```
func getAllPlays(): [TopShot.Play] 
```

 getAllPlays returns all the plays in topshot

 Returns: An array of all the plays that have been created

---

### fun `getPlayMetaData()`

```
func getPlayMetaData(playID UInt32): {String: String}? 
```

 getPlayMetaData returns all the metadata associated with a specific Play

 Parameters: playID: The id of the Play that is being searched

 Returns: The metadata as a String to String mapping optional

---

### fun `getPlayMetaDataByField()`

```
func getPlayMetaDataByField(playID UInt32, field String): String? 
```

 getPlayMetaDataByField returns the metadata associated with a
                        specific field of the metadata
                        Ex: field: "Team" will return something
                        like "Memphis Grizzlies"

 Parameters: playID: The id of the Play that is being searched
             field: The field to search for

 Returns: The metadata field as a String Optional

---

### fun `getSetName()`

```
func getSetName(setID UInt32): String? 
```

 getSetName returns the name that the specified Set
            is associated with.

 Parameters: setID: The id of the Set that is being searched

 Returns: The name of the Set

---

### fun `getSetSeries()`

```
func getSetSeries(setID UInt32): UInt32? 
```

 getSetSeries returns the series that the specified Set
              is associated with.

 Parameters: setID: The id of the Set that is being searched

 Returns: The series that the Set belongs to

---

### fun `getSetIDsByName()`

```
func getSetIDsByName(setName String): [UInt32]? 
```

 getSetIDsByName returns the IDs that the specified Set name
                 is associated with.

 Parameters: setName: The name of the Set that is being searched

 Returns: An array of the IDs of the Set if it exists, or nil if doesn't

---

### fun `getPlaysInSet()`

```
func getPlaysInSet(setID UInt32): [UInt32]? 
```

 getPlaysInSet returns the list of Play IDs that are in the Set

 Parameters: setID: The id of the Set that is being searched

 Returns: An array of Play IDs

---

### fun `isEditionRetired()`

```
func isEditionRetired(setID UInt32, playID UInt32): Bool? 
```

 isEditionRetired returns a boolean that indicates if a Set/Play combo
                  (otherwise known as an edition) is retired.
                  If an edition is retired, it still remains in the Set,
                  but Moments can no longer be minted from it.

 Parameters: setID: The id of the Set that is being searched
             playID: The id of the Play that is being searched

 Returns: Boolean indicating if the edition is retired or not

---

### fun `isSetLocked()`

```
func isSetLocked(setID UInt32): Bool? 
```

 isSetLocked returns a boolean that indicates if a Set
             is locked. If it's locked,
             new Plays can no longer be added to it,
             but Moments can still be minted from Plays the set contains.

 Parameters: setID: The id of the Set that is being searched

 Returns: Boolean indicating if the Set is locked or not

---

### fun `getNumMomentsInEdition()`

```
func getNumMomentsInEdition(setID UInt32, playID UInt32): UInt32? 
```

 getNumMomentsInEdition return the number of Moments that have been
                        minted from a certain edition.

 Parameters: setID: The id of the Set that is being searched
             playID: The id of the Play that is being searched

 Returns: The total number of Moments
          that have been minted from an edition

---


## Structs & Resources:


### struct `Play`

```
struct Play {
    playID UInt32
    metadata {String: String}
}
```

 Play is a Struct that holds metadata associated
 with a specific NBA play, like the legendary moment when
 Ray Allen hit the 3 to tie the Heat and Spurs in the 2013 finals game 6
 or when Lance Stephenson blew in the ear of Lebron James.

 Moment NFTs will all reference a single play as the owner of
 its metadata. The plays are publicly accessible, so anyone can
 read the metadata associated with a specific play ID


[More...](TopShot_Play.md)

---

### struct `SetData`

```
struct SetData {
    setID UInt32
    name String
    series UInt32
}
```

 A Set is a grouping of Plays that have occured in the real world
 that make up a related group of collectibles, like sets of baseball
 or Magic cards. A Play can exist in multiple different sets.

 SetData is a struct that is stored in a field of the contract.
 Anyone can query the constant information
 about a set by calling various getters located
 at the end of the contract. Only the admin has the ability
 to modify any data in the private Set resource.


[More...](TopShot_SetData.md)

---

### resource `Set`

```
resource Set {
    setID UInt32
    plays [UInt32]
    retired {UInt32: Bool}
    locked Bool
    numberMintedPerPlay {UInt32: UInt32}
}
```

 Set is a resource type that contains the functions to add and remove
 Plays from a set and mint Moments.

 It is stored in a private field in the contract so that
 the admin resource can call its methods.

 The admin can add Plays to a Set so that the set can mint Moments
 that reference that playdata.
 The Moments that are minted by a Set will be listed as belonging to
 the Set that minted it, as well as the Play it references.

 Admin can also retire Plays from the Set, meaning that the retired
 Play can no longer have Moments minted from it.

 If the admin locks the Set, no more Plays can be added to it, but
 Moments can still be minted.

 If retireAll() and lock() are called back-to-back,
 the Set is closed off forever and nothing more can be done with it.

[More...](TopShot_Set.md)

---

### struct `MomentData`

```
struct MomentData {
    setID UInt32
    playID UInt32
    serialNumber UInt32
}
```



[More...](TopShot_MomentData.md)

---

### resource `NFT`

```
resource NFT {
    id UInt64
    data MomentData
}
```

 The resource that represents the Moment NFTs


[More...](TopShot_NFT.md)

---

### resource `Admin`

```
resource Admin {
}
```

 Admin is a special authorization resource that
 allows the owner to perform important functions to modify the
 various aspects of the Plays, Sets, and Moments


[More...](TopShot_Admin.md)

---

### resource `Collection`

```
resource Collection {
    ownedNFTs {UInt64: NonFungibleToken.NFT}
}
```

 Collection is a resource that every user who owns NFTs
 will store in their account to manage their NFTS


[More...](TopShot_Collection.md)

---


## Interfaces:


### resource interface `MomentCollectionPublic`

```
resource interface MomentCollectionPublic {
}
```

 This is the interface that users can cast their Moment Collection as
 to allow others to deposit Moments into their Collection. It also allows for reading
 the IDs of Moments in the Collection.

[More...](TopShot_MomentCollectionPublic.md)

---


