# resource `Admin`

```
resource Admin {
}
```

 Admin is a special authorization resource that
 allows the owner to perform important functions to modify the
 various aspects of the Plays, Sets, and Moments


## Functions


### fun `createPlay()`

```
func createPlay(metadata {String: String}): UInt32 
```

 createPlay creates a new Play struct
 and stores it in the Plays dictionary in the TopShot smart contract

 Parameters: metadata: A dictionary mapping metadata titles to their data
                       example: {"Player Name": "Kevin Durant", "Height": "7 feet"}
                               (because we all know Kevin Durant is not 6'9")

 Returns: the ID of the new Play object


---

### fun `createSet()`

```
func createSet(name String)
```

 createSet creates a new Set resource and stores it
 in the sets mapping in the TopShot contract

 Parameters: name: The name of the Set


---

### fun `borrowSet()`

```
func borrowSet(setID UInt32): &Set 
```

 borrowSet returns a reference to a set in the TopShot
 contract so that the admin can call methods on it

 Parameters: setID: The ID of the Set that you want to
 get a reference to

 Returns: A reference to the Set with all of the fields
 and methods exposed


---

### fun `startNewSeries()`

```
func startNewSeries(): UInt32 
```

 startNewSeries ends the current series by incrementing
 the series number, meaning that Moments minted after this
 will use the new series number

 Returns: The new series number


---

### fun `createNewAdmin()`

```
func createNewAdmin(): Admin 
```

 createNewAdmin creates a new Admin resource


---


