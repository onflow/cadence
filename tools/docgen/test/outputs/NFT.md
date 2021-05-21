# Contract `NFT`

```cadence
contract NFT

    field1:  Int

    field2:  String
}
```
NFT is a dummy non-fungible token contract.
Implemented Interfaces:
 - `Token`

## Functions

### fun `foo()`

```cadence
func foo(a Int, b String):  
```
This is a foo function,
This doesn't have a return type.

---

### fun `bar()`

```cadence
func bar(name String, bytes [Int8]): bool 
```
This is a bar function, with a return type

Parameters:
  - name : _The name. Must be a string_
  - bytes : _Content to be validated_

Returns: Validity of the content

---

### fun `noDocsFunction()`

```cadence
func noDocsFunction():  
```

---
## Structs & Resources

### struct `SomeStruct`

```cadence
struct SomeStruct {

    x:  String

    y:  {Int: AnyStruct}
}
```
This is some struct. It has
@field x: a string field
@field y: a map of int and any-struct

[More...](NFT_SomeStruct.md)

---

### enum `Direction`

```cadence
enum Direction {
    case LEFT
    case RIGHT
}
```
This is an Enum without type conformance.

---
### enum `Color`

```cadence
enum Color: Int8 {
    case Red
    case Blue
}
```
This is an Enum, with explicit type conformance.

---