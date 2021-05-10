# Resource `Set`

```cadence
resource Set {

    setID:  UInt32

    plays:  [UInt32]

    retired:  {UInt32: Bool}

    locked:  Bool

    numberMintedPerPlay:  {UInt32: UInt32}
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


### Initializer

```cadence
func init(name String)
```


## Functions

### fun `addPlay()`

```cadence
func addPlay(playID UInt32):  
```

 addPlay adds a play to the set

 Parameters: playID: The ID of the Play that is being added

 Pre-Conditions:
 The Play needs to be an existing play
 The Set needs to be not locked
 The Play can't have already been added to the Set


---

### fun `addPlays()`

```cadence
func addPlays(playIDs [UInt32]):  
```

 addPlays adds multiple Plays to the Set

 Parameters: playIDs: The IDs of the Plays that are being added
                      as an array


---

### fun `retirePlay()`

```cadence
func retirePlay(playID UInt32):  
```

 retirePlay retires a Play from the Set so that it can't mint new Moments

 Parameters: playID: The ID of the Play that is being retired

 Pre-Conditions:
 The Play is part of the Set and not retired (available for minting).


---

### fun `retireAll()`

```cadence
func retireAll():  
```

 retireAll retires all the plays in the Set
 Afterwards, none of the retired Plays will be able to mint new Moments


---

### fun `lock()`

```cadence
func lock():  
```

 lock() locks the Set so that no more Plays can be added to it

 Pre-Conditions:
 The Set should not be locked

---

### fun `mintMoment()`

```cadence
func mintMoment(playID UInt32): NFT 
```

 mintMoment mints a new Moment and returns the newly minted Moment

 Parameters: playID: The ID of the Play that the Moment references

 Pre-Conditions:
 The Play must exist in the Set and be allowed to mint new Moments

 Returns: The NFT that was minted


---

### fun `batchMintMoment()`

```cadence
func batchMintMoment(playID UInt32, quantity UInt64): Collection 
```

 batchMintMoment mints an arbitrary quantity of Moments
 and returns them as a Collection

 Parameters: playID: the ID of the Play that the Moments are minted for
             quantity: The quantity of Moments to be minted

 Returns: Collection object that contains all the Moments that were minted


---

