---
title: Type Inference
---

If a variable or constant declaration is not annotated explicitly with a type,
the declaration's type is inferred from the initial value.

### Basic Literals
Decimal integer literals and hex literals are inferred to type `Int`.

```cadence
let a = 1
// `a` has type `Int`

let b = -45
// `b` has type `Int`

let c = 0x02
// `c` has type `Int`
```

Unsigned fixed-point literals are inferred to type `UFix64`.
Signed fixed-point literals are inferred to type `Fix64`.

```cadence
let a = 1.2
// `a` has type `UFix64`

let b = -1.2
// `b` has type `Fix64`
```

Similarly, for other basic literals, the types are inferred in the following manner:

| Literal Kind      | Example           | Inferred Type (x) |
|:-----------------:|:-----------------:|:-----------------:|
| String literal    | `let x = "hello"` |  String           |
| Boolean literal   | `let x = true`    |  Bool             |
| Nil literal       | `let x = nil`     |  Never?           |


### Array Literals
Array literals are inferred based on the elements of the literal, and to be variable-size.
The inferred element type is the _least common super-type_ of all elements.

```cadence
let integers = [1, 2]
// `integers` has type `[Int]`

let int8Array = [Int8(1), Int8(2)]
// `int8Array` has type `[Int8]`

let mixedIntegers = [UInt(65), 6, 275, Int128(13423)]
// `mixedIntegers` has type `[Integer]`

let nilableIntegers = [1, nil, 2, 3, nil]
// `nilableIntegers` has type `[Int?]`

let mixed = [1, true, 2, false]
// `mixed` has type `[AnyStruct]`
```

### Dictionary Literals
Dictionary literals are inferred based on the keys and values of the literal.
The inferred type of keys and values is the _least common super-type_ of all keys and values, respectively.

```cadence
let booleans = {
    1: true,
    2: false
}
// `booleans` has type `{Int: Bool}`

let mixed = {
    Int8(1): true,
    Int64(2): "hello"
}
// `mixed` has type `{Integer: AnyStruct}`

// Invalid: mixed keys
//
let invalidMixed = {
    1: true,
    false: 2
}
// The least common super-type of the keys is `AnyStruct`.
// But it is not a valid type for dictionary keys.
```

### Ternary Expression
Ternary expression type is inferred  to be the least common super-type of the second and third operands.
```cadence
let a = true ? 1 : 2
// `a` has type `Int`

let b = true ? 1 : nil
// `b` has type `Int?`

let c = true ? 5 : (false ? "hello" : nil)
// `c` has type `AnyStruct`
```

### Functions
Functions are inferred based on the parameter types and the return type.

```cadence
let add = (a: Int8, b: Int8): Int {
    return a + b
}

// `add` has type `((Int8, Int8): Int)`
```

Type inference is performed for each expression / statement, and not across statements.

## Ambiguities
There are cases where types cannot be inferred.
In these cases explicit type annotations are required.

```cadence
// Invalid: not possible to infer type based on array literal's elements.
//
let array = []

// Instead, specify the array type and the concrete element type, e.g. `Int`.
//
let array: [Int] = []

// Or, use a simple-cast to annotate the expression with a type.
let array = [] as [Int]
```

```cadence
// Invalid: not possible to infer type based on dictionary literal's keys and values.
//
let dictionary = {}

// Instead, specify the dictionary type and the concrete key
// and value types, e.g. `String` and `Int`.
//
let dictionary: {String: Int} = {}

// Or, use a simple-cast to annotate the expression with a type.
let dictionary = {} as {String: Int}
```
