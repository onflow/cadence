---
title: Type Annotations
---

When declaring a constant or variable,
an optional *type annotation* can be provided,
to make it explicit what type the declaration has.

If no type annotation is provided, the type of the declaration is
[inferred from the initial value](type-inference).

For function parameters a type annotation must be provided.

```cadence
// Declare a variable named `boolVarWithAnnotation`, which has an explicit type annotation.
//
// `Bool` is the type of booleans.
//
var boolVarWithAnnotation: Bool = false

// Declare a constant named `integerWithoutAnnotation`, which has no type annotation
// and for which the type is inferred to be `Int`, the type of arbitrary-precision integers.
//
// This is based on the initial value which is an integer literal.
// Integer literals are always inferred to be of type `Int`.
//
let integerWithoutAnnotation = 1

// Declare a constant named `smallIntegerWithAnnotation`, which has an explicit type annotation.
// Because of the explicit type annotation, the type is not inferred.
// This declaration is valid because the integer literal `1` fits into the range of the type `Int8`,
// the type of 8-bit signed integers.
//
let smallIntegerWithAnnotation: Int8 = 1
```

If a type annotation is provided, the initial value must be of this type.
All new values assigned to variables must match its type.
This type safety is explained in more detail in a [separate section](type-safety).

```cadence
// Invalid: declare a variable with an explicit type `Bool`,
// but the initial value has type `Int`.
//
let booleanConstant: Bool = 1

// Declare a variable that has the inferred type `Bool`.
//
var booleanVariable = false

// Invalid: assign a value with type `Int` to a variable which has the inferred type `Bool`.
//
booleanVariable = 1
```
