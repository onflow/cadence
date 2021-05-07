# contract `NFT`

```
contract NFT {
}
```

 This is a dummy NFT contract. It has several members of different types.
 Each member has their own documentation. 

## Members:



### fun `foo()`

```
func foo(a Int, b String)
```

 This is a foo function

---

### fun `bar()`

```
func bar(name String, bytes [Int8]): bool 
```

 This is a bar function

---

### struct `Some`

```
struct Some {
    x String
    y {Int: AnyStruct}
}
```

 This is some struct. It has
 @field x: a string field
 @field y: a map of int and any-struct

[More...](NFT_Some.md)

---

### enum `Direction`

```
enum Direction
    case LEFT
    case RIGHT
}
```

 This is an Enum without type conformance.

---

### enum `Color`

```
enum Color: Int8 {
    case Red
    case Blue
}
```

 This is an Enum, with explicit type conformance.

---



