---
title: Enumerations
---

Enumerations are sets of symbolic names bound to unique, constant values,
which can be compared by identity.

## Enum Declaration

Enums are declared using the `enum` keyword,
followed by the name of the enum, the raw type after a colon,
and the requirements, which must be enclosed in opening and closing braces.

The raw type must be an integer subtype, e.g. `UInt8` or `Int128`.

Enum cases are declared using the `case` keyword,
followed by the name of the enum case.

Enum cases must be unique.
Each enum case has a raw value, the index of the case in all cases.

The raw value of an enum case can be accessed through the `rawValue` field.

The enum cases can be accessed by using the name as a field on the enum,
or by using the enum constructor,
which requires providing the raw value as an argument.
The enum constructor returns the enum case with the given raw value,
if any, or `nil` if no such case exists.

Enum cases can be compared using the equality operators `==` and `!=`.

```cadence
// Declare an enum named `Color` which has the raw value type `UInt8`,
// and declare three enum cases: `red`, `green`, and `blue`
//
pub enum Color: UInt8 {
    pub case red
    pub case green
    pub case blue
}
// Declare a variable which has the enum type `Color` and initialize
// it to the enum case `blue` of the enum
let blue: Color = Color.blue
// Get the raw value of the enum case `blue`.
// As it is the third case, so it has index 2
//
blue.rawValue // is `2`
// Get the `green` enum case of the enum `Color` by using the enum
// constructor and providing the raw value of the enum case `green`, 1,
// as the enum case `green` is the second case, so it has index 1
//
let green: Color? = Color(rawValue: 1)  // is `Color.green`
// Get the enum case of the enum `Color` that has the raw value 5.
// As there are only three cases, the maximum raw value / index is 2.
//
let nothing = Color(rawValue: 5)  // is `nil`
// Enum cases can be compared
Color.red == Color.red  // is `true`
Color(rawValue: 1) == Color.green  // is `true`
// Different enum cases are not the same
Color.red != Color.blue  // is `true`
```
