---
title: Type Safety
type: REF
---

The Cadence programming language is a *type-safe* language.

When assigning a new value to a variable, the value must be the same type as the variable.
For example, if a variable has type `Bool`,
it can *only* be assigned a value that has type `Bool`,
and not for example a value that has type `Int`.

```cadence
// Declare a variable that has type `Bool`.
var a = true

// Invalid: cannot assign a value that has type `Int` to a variable which has type `Bool`.
//
a = 0
```

When passing arguments to a function,
the types of the values must match the function parameters' types.
For example, if a function expects an argument that has type `Bool`,
*only* a value that has type `Bool` can be provided,
and not for example a value which has type `Int`.

```cadence
fun nand(_ a: Bool, _ b: Bool): Bool {
    return !(a && b)
}

nand(false, false)  // is `true`

// Invalid: The arguments of the function calls are integers and have type `Int`,
// but the function expects parameters booleans (type `Bool`).
//
nand(0, 0)
```

Types are **not** automatically converted.
For example, an integer is not automatically converted to a boolean,
nor is an `Int32` automatically converted to an `Int8`,
nor is an optional integer `Int?`
automatically converted to a non-optional integer `Int`,
or vice-versa.

```cadence
fun add(_ a: Int8, _ b: Int8): Int8 {
    return a + b
}

// The arguments are not declared with a specific type, but they are inferred
// to be `Int8` since the parameter types of the function `add` are `Int8`.
add(1, 2)  // is `3`

// Declare two constants which have type `Int32`.
//
let a: Int32 = 3_000_000_000
let b: Int32 = 3_000_000_000

// Invalid: cannot pass arguments which have type `Int32` to parameters which have type `Int8`.
//
add(a, b)
```
