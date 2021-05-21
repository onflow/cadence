
### fun `foo()`

```cadence
func foo(a Int, b String):  
```

---

### fun `bar()`

```cadence
func bar(name String, bytes [Int8]): bool 
```
 This is a bar function, with a return type
 @param name: The name. Must be a string
 @param bytes: Content
 @returns Validity

---

### fun `noDocsFunction()`

```cadence
func noDocsFunction():  
```

---

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

[More...](SomeStruct.md)

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