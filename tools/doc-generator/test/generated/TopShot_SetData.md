# struct `SetData`

```
struct SetData {

    setID:  UInt32

    name:  String

    series:  UInt32
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


