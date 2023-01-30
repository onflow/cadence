---
title: Operators
---

Operators are special symbols that perform a computation
for one or more values.
They are either unary, binary, or ternary.

- Unary operators perform an operation for a single value.
  The unary operator symbol appears before the value.

- Binary operators operate on two values.
    The binary operator symbol appears between the two values (infix).

- Ternary operators operate on three values.
  The first operator symbol appears between the first and second value,
  the second operator symbol appears between the second and third value (infix).

## Assignment Operator (`=`)

The binary assignment operator `=` can be used
to assign a new value to a variable.
It is only allowed in a statement and is not allowed in expressions.

```cadence
var a = 1
a = 2
// `a` is `2`


var b = 3
var c = 4

// Invalid: The assignment operation cannot be used in an expression.
a = b = c

// Instead, the intended assignment must be written in multiple statements.
b = c
a = b
```

Assignments to constants are invalid.

```cadence
let a = 1
// Invalid: Assignments are only for variables, not constants.
a = 2
```

The left-hand side of the assignment operand must be an identifier.
For arrays and dictionaries, this identifier can be followed
by one or more index or access expressions.

```cadence
// Declare an array of integers.
let numbers = [1, 2]

// Change the first element of the array.
//
numbers[0] = 3

// `numbers` is `[3, 2]`
```

```cadence
// Declare an array of arrays of integers.
let arrays = [[1, 2], [3, 4]]

// Change the first element in the second array
//
arrays[1][0] = 5

// `arrays` is `[[1, 2], [5, 4]]`
```

```cadence
let dictionaries = {
  true: {1: 2},
  false: {3: 4}
}

dictionaries[false][3] = 0

// `dictionaries` is `{
//   true: {1: 2},
//   false: {3: 0}
//}`
```

## Force-assignment operator (`<-!`)

The force-assignment operator (`<-!`) assigns a resource-typed value
to an optional-typed variable if the variable is nil.
If the variable being assigned to is non-nil,
the execution of the program aborts.

The force-assignment operator is only used for [resource types](resources).

## Swapping Operator (`<->`)

The binary swap operator `<->` can be used
to exchange the values of two variables.
It is only allowed in a statement and is not allowed in expressions.

```cadence
var a = 1
var b = 2
a <-> b
// `a` is `2`
// `b` is `1`

var c = 3

// Invalid: The swap operation cannot be used in an expression.
a <-> b <-> c

// Instead, the intended swap must be written in multiple statements.
b <-> c
a <-> b
```

Both sides of the swap operation must be variable,
assignment to constants is invalid.

```cadence
var a = 1
let b = 2

// Invalid: Swapping is only possible for variables, not constants.
a <-> b
```

Both sides of the swap operation must be an identifier,
followed by one or more index or access expressions.

## Arithmetic Operators

The unary pefix operator  `-` negates an integer:

```cadence
let a = 1
-a  // is `-1`
```

There are four binary arithmetic operators:

- Addition: `+`
- Subtraction: `-`
- Multiplication: `*`
- Division: `/`
- Remainder: `%`

```cadence
let a = 1 + 2
// `a` is `3`
```

The arguments for the operators need to be of the same type.
The result is always the same type as the arguments.

The division and remainder operators abort the program when the divisor is zero.

Arithmetic operations on the signed integer types
`Int8`, `Int16`, `Int32`, `Int64`, `Int128`, `Int256`,
and on the unsigned integer types
`UInt8`, `UInt16`, `UInt32`, `UInt64`, `UInt128`, `UInt256`,
do not cause values to overflow or underflow.

```cadence
let a: UInt8 = 255

// Run-time error: The result `256` does not fit in the range of `UInt8`,
// thus a fatal overflow error is raised and the program aborts
//
let b = a + 1
```

```cadence
let a: Int8 = 100
let b: Int8 = 100

// Run-time error: The result `10000` does not fit in the range of `Int8`,
// thus a fatal overflow error is raised and the program aborts
//
let c = a * b
```

```cadence
let a: Int8 = -128

// Run-time error: The result `128` does not fit in the range of `Int8`,
// thus a fatal overflow error is raised and the program aborts
//
let b = -a
```

Arithmetic operations on the unsigned integer types
`Word8`, `Word16`, `Word32`, `Word64`
may cause values to overflow or underflow.

For example, the maximum value of an unsigned 8-bit integer is 255 (binary 11111111).
Adding 1 results in an overflow, truncation to 8 bits, and the value 0.

```cadence
//    11111111 = 255
// +         1
// = 100000000 = 0
```

```cadence
let a: Word8 = 255
a + 1 // is `0`
```

Similarly, for the minimum value 0,
subtracting 1 wraps around and results in the maximum value 255.

```cadence
//    00000000
// -         1
// =  11111111 = 255
```

```cadence
let b: Word8 = 0
b - 1  // is `255`
```

### Arithmetics on number super-types

Arithmetic operators are not supported for number supertypes
(`Number`, `SignedNumber`, `FixedPoint`, `SignedFixedPoint`, `Integer`, `SignedInteger`),
as they may or may not succeed at run-time.

```cadence
let x: Integer = 3 as Int8
let y: Integer = 4 as Int8

let z: Integer = x + y    // Static error
```

Values of these types need to be cast to the desired type before performing the arithmetic operation.

```cadence
let z: Integer = (x as! Int8) + (y as! Int8)
```

## Logical Operators

Logical operators work with the boolean values `true` and `false`.

- Logical NOT: `!a`

  This unary prefix operator logically negates a boolean:

  ```cadence
  let a = true
  !a  // is `false`
  ```

- Logical AND: `a && b`

  ```cadence
  true && true  // is `true`

  true && false  // is `false`

  false && true  // is `false`

  false && false  // is `false`
  ```

  If the left-hand side is false, the right-hand side is not evaluated.

- Logical OR: `a || b`

  ```cadence
  true || true  // is `true`

  true || false  // is `true`

  false || true  // is `true`

  false || false // is `false`
  ```

  If the left-hand side is true, the right-hand side is not evaluated.

## Comparison Operators

Comparison operators work with boolean and integer values.

- Equality: `==`, is supported for booleans, numbers, addresses, strings, characters, enums, paths, `Type` values, references, and `Void` values (`()`). Variable-sized arrays, fixed-size arrays, and optionals also support equality tests if their inner types do.

  Both sides of the equality operator may be optional, even of different levels,
  so it is for example possible to compare a non-optional with a double-optional (`??`).

  ```cadence
  1 == 1  // is `true`

  1 == 2  // is `false`
  ```

  ```cadence
  true == true  // is `true`

  true == false  // is `false`
  ```

  ```cadence
  let x: Int? = 1
  x == nil  // is `false`
  ```

  ```cadence
  let x: Int = 1
  x == nil  // is `false`
  ```

  ```cadence
  // Comparisons of different levels of optionals are possible.
  let x: Int? = 2
  let y: Int?? = nil
  x == y  // is `false`
  ```

  ```cadence
  // Comparisons of different levels of optionals are possible.
  let x: Int? = 2
  let y: Int?? = 2
  x == y  // is `true`
  ```

  ```cadence
  // Equality tests of arrays are possible if their inner types are equatable.
  let xs: [Int] = [1, 2, 3]
  let ys: [Int] = [1, 2, 3]
  xs == ys // is `true`

  let xss: [[Int]] = [xs, xs, xs]
  let yss: [[Int]] = [ys, ys, ys]
  xss == yss // is `true`
  ```

  ```cadence
  // Equality also applies to fixed-size arrays. If their lengths differ, the result is a type error.
  let xs: [Int; 2] = [1, 2]
  let ys: [Int; 2] = [0 + 1, 1 + 1]
  xs == ys // is `true`
  ```

- Inequality: `!=`, is supported for booleans, numbers, addresses, strings, characters, enums, paths, `Type` values, references, and `Void` values (`()`). 
  Variable-sized arrays, fixed-size arrays, and optionals also support inequality tests if their inner types do.

  Both sides of the inequality operator may be optional, even of different levels,
  so it is for example possible to compare a non-optional with a double-optional (`??`).

  ```cadence
  1 != 1  // is `false`

  1 != 2  // is `true`
  ```

  ```cadence
  true != true  // is `false`

  true != false  // is `true`
  ```

  ```cadence
  let x: Int? = 1
  x != nil  // is `true`
  ```

  ```cadence
  let x: Int = 1
  x != nil  // is `true`
  ```

  ```cadence
  // Comparisons of different levels of optionals are possible.
  let x: Int? = 2
  let y: Int?? = nil
  x != y  // is `true`
  ```

  ```cadence
  // Comparisons of different levels of optionals are possible.
  let x: Int? = 2
  let y: Int?? = 2
  x != y  // is `false`
  ```

  ```cadence
  // Inequality tests of arrays are possible if their inner types are equatable.
  let xs: [Int] = [1, 2, 3]
  let ys: [Int] = [4, 5, 6]
  xs != ys // is `true`
  ```

  ```cadence
  // Inequality also applies to fixed-size arrays. If their lengths differ, the result is a type error.
  let xs: [Int; 2] = [1, 2]
  let ys: [Int; 2] = [1, 2]
  xs != ys // is `false`
  ```

- Less than: `<`, for integers

  ```cadence
  1 < 1  // is `false`

  1 < 2  // is `true`

  2 < 1  // is `false`
  ```

- Less or equal than: `<=`, for integers

  ```cadence
  1 <= 1  // is `true`

  1 <= 2  // is `true`

  2 <= 1  // is `false`
  ```

- Greater than: `>`, for integers

  ```cadence
  1 > 1  // is `false`

  1 > 2  // is `false`

  2 > 1  // is `true`
  ```

- Greater or equal than: `>=`, for integers

  ```cadence
  1 >= 1  // is `true`

  1 >= 2  // is `false`

  2 >= 1  // is `true`
  ```

### Comparing number super-types

Similar to arithmetic operators, comparison operators are also not supported for number supertypes
(`Number`, `SignedNumber` `FixedPoint`, `SignedFixedPoint`, `Integer`, `SignedInteger`),
as they may or may not succeed at run-time.

```cadence
let x: Integer = 3 as Int8
let y: Integer = 4 as Int8

let z: Bool = x > y    // Static error
```

Values of these types need to be cast to the desired type before performing the arithmetic operation.

```cadence
let z: Bool = (x as! Int8) > (y as! Int8)
```

## Bitwise Operators

Bitwise operators enable the manipulation of individual bits of unsigned and signed integers.
They're often used in low-level programming.

- Bitwise AND: `a & b`

  Returns a new integer whose bits are 1 only if the bits were 1 in *both* input integers:

  ```cadence
  let firstFiveBits = 0b11111000
  let lastFiveBits  = 0b00011111
  let middleTwoBits = firstFiveBits & lastFiveBits  // is 0b00011000
  ```

- Bitwise OR: `a | b`

  Returns a new integer whose bits are 1 only if the bits were 1 in *either* input integers:

  ```cadence
  let someBits = 0b10110010
  let moreBits = 0b01011110
  let combinedbits = someBits | moreBits  // is 0b11111110
  ```

- Bitwise XOR: `a ^ b`

  Returns a new integer whose bits are 1 where the input bits are different,
  and are 0 where the input bits are the same:

  ```cadence
  let firstBits = 0b00010100
  let otherBits = 0b00000101
  let outputBits = firstBits ^ otherBits  // is 0b00010001
  ```

### Bitwise Shifting Operators

- Bitwise LEFT SHIFT: `a << b`

  Returns a new integer with all bits moved to the left by a certain number of places.

  ```cadence
  let someBits = 4  // is 0b00000100
  let shiftedBits = someBits << 2   // is 0b00010000
  ```

- Bitwise RIGHT SHIFT: `a >> b`

  Returns a new integer with all bits moved to the right by a certain number of places.

  ```cadence
  let someBits = 8  // is 0b00001000
  let shiftedBits = someBits >> 2   // is 0b00000010
  ```

For unsigned integers, the bitwise shifting operators perform [logical shifting](https://en.wikipedia.org/wiki/Logical_shift),
for signed integers, they perform [arithmetic shifting](https://en.wikipedia.org/wiki/Arithmetic_shift).

## Ternary Conditional Operator

There is only one ternary conditional operator, the ternary conditional operator (`a ? b : c`).

It behaves like an if-statement, but is an expression:
If the first operator value is true, the second operator value is returned.
If the first operator value is false, the third value is returned.

The first value must be a boolean (must have the type `Bool`).
The second value and third value can be of any type.
The result type is the least common supertype of the second and third value.

```cadence
let x = 1 > 2 ? 3 : 4
// `x` is `4` and has type `Int`

let y = 1 > 2 ? nil : 3
// `y` is `3` and has type `Int?`
```

## Casting Operators

### Static Casting Operator (`as`)

The static casting operator `as` can be used to statically type cast a value to a type.

If the static type of the value is a subtype of the given type (the "target" type),
the operator returns the value as the given type.

The cast is performed statically, i.e. when the program is type-checked.
Only the static type of the value is considered, not the run-time type of the value.

This means it is not possible to downcast using this operator.
Consider using the [conditional downcasting operator `as?`](#conditional-downcasting-operator-as) instead.

```cadence
// Declare a constant named `integer` which has type `Int`.
//
let integer: Int = 1

// Statically cast the value of `integer` to the supertype `Number`.
// The cast succeeds, because the type of the variable `integer`,
// the type `Int`, is a subtype of type `Number`.
// This is an upcast.
//
let number = integer as Number
// `number` is `1` and has type `Number`

// Declare a constant named `something` which has type `AnyStruct`,
// with an initial value which has type `Int`.
//
let something: AnyStruct = 1

// Statically cast the value of `something` to `Int`.
// This is invalid, the cast fails, because the static type of the value is type `AnyStruct`,
// which is not a subtype of type `Int`.
//
let result = something as Int
```

### Conditional Downcasting Operator (`as?`)

The conditional downcasting operator `as?` can be used to dynamically type cast a value to a type.
The operator returns an optional.
If the value has a run-time type that is a subtype of the target type
the operator returns the value as the target type,
otherwise the result is `nil`.

The cast is performed at run-time, i.e. when the program is executed,
not statically, i.e. when the program is checked.

```cadence
// Declare a constant named `something` which has type `AnyStruct`,
// with an initial value which has type `Int`.
//
let something: AnyStruct = 1

// Conditionally downcast the value of `something` to `Int`.
// The cast succeeds, because the value has type `Int`.
//
let number = something as? Int
// `number` is `1` and has type `Int?`

// Conditionally downcast the value of `something` to `Bool`.
// The cast fails, because the value has type `Int`,
// and `Bool` is not a subtype of `Int`.
//
let boolean = something as? Bool
// `boolean` is `nil` and has type `Bool?`
```

Downcasting works for concrete types, but also works e.g. for nested types (e.g. arrays), interfaces, optionals, etc.

```cadence
// Declare a constant named `values` which has type `[AnyStruct]`,
// i.e. an array of arbitrarily typed values.
//
let values: [AnyStruct] = [1, true]

let first = values[0] as? Int
// `first` is `1` and has type `Int?`

let second = values[1] as? Bool
// `second` is `true` and has type `Bool?`
```

### Force-downcasting Operator (`as!`)

The force-downcasting operator `as!` behaves like the
[conditional downcasting operator `as?`](#conditional-downcasting-operator-as).
However, if the cast succeeds, it returns a value of the given type instead of an optional,
and if the cast fails, it aborts the program instead of returning `nil`,

```cadence
// Declare a constant named `something` which has type `AnyStruct`,
// with an initial value which has type `Int`.
//
let something: AnyStruct = 1

// Force-downcast the value of `something` to `Int`.
// The cast succeeds, because the value has type `Int`.
//
let number = something as! Int
// `number` is `1` and has type `Int`

// Force-downcast the value of `something` to `Bool`.
// The cast fails, because the value has type `Int`,
// and `Bool` is not a subtype of `Int`.
//
let boolean = something as! Bool
// Run-time error
```

## Optional Operators

### Nil-Coalescing Operator (`??`)

The nil-coalescing operator `??` returns
the value inside an optional if it contains a value,
or returns an alternative value if the optional has no value,
i.e., the optional value is `nil`.

If the left-hand side is non-nil, the right-hand side is not evaluated.

```cadence
// Declare a constant which has an optional integer type
//
let a: Int? = nil

// Declare a constant with a non-optional integer type,
// which is initialized to `a` if it is non-nil, or 42 otherwise.
//
let b: Int = a ?? 42
// `b` is 42, as `a` is nil
```

The nil-coalescing operator can only be applied
to values which have an optional type.

```cadence
// Declare a constant with a non-optional integer type.
//
let a = 1

// Invalid: nil-coalescing operator is applied to a value which has a non-optional type
// (a has the non-optional type `Int`).
//
let b = a ?? 2
```

```cadence
// Invalid: nil-coalescing operator is applied to a value which has a non-optional type
// (the integer literal is of type `Int`).
//
let c = 1 ?? 2
```

The type of the right-hand side of the operator (the alternative value) must be a subtype
of the type of left-hand side, i.e. the right-hand side of the operator must
be the non-optional or optional type matching the type of the left-hand side.

```cadence
// Declare a constant with an optional integer type.
//
let a: Int? = nil
let b: Int? = 1
let c = a ?? b
// `c` is `1` and has type `Int?`

// Invalid: nil-coalescing operator is applied to a value of type `Int?`,
// but the alternative has type `Bool`.
//
let d = a ?? false
```

### Force Unwrap Operator (`!`)

The force-unwrap operator (`!`) returns
the value inside an optional if it contains a value,
or panics and aborts the execution if the optional has no value,
i.e., the optional value is `nil`.

```cadence
// Declare a constant which has an optional integer type
//
let a: Int? = nil

// Declare a constant with a non-optional integer type,
// which is initialized to `a` if `a` is non-nil.
// If `a` is nil, the program aborts.
//
let b: Int = a!
// The program aborts because `a` is nil.

// Declare another optional integer constant
let c: Int? = 3

// Declare a non-optional integer
// which is initialized to `c` if `c` is non-nil.
// If `c` is nil, the program aborts.
let d: Int = c!
// `d` is initialized to 3 because c isn't nil.

```

The force-unwrap operator can only be applied
to values which have an optional type.

```cadence
// Declare a constant with a non-optional integer type.
//
let a = 1

// Invalid: force-unwrap operator is applied to a value which has a
// non-optional type (`a` has the non-optional type `Int`).
//
let b = a!
```

```cadence
// Invalid: The force-unwrap operator is applied
// to a value which has a non-optional type
// (the integer literal is of type `Int`).
//
let c = 1!
```


## Precedence and Associativity

Operators have the following precedences, highest to lowest:

- Unary precedence: `-`, `!`, `<-`
- Cast precedence: `as`, `as?`, `as!`
- Multiplication precedence: `*`, `/`, `%`
- Addition precedence: `+`, `-`
- Bitwise shift precedence: `<<`, `>>`
- Bitwise conjunction precedence: `&`
- Bitwise exclusive disjunction precedence: `^`
- Bitwise disjunction precedence: `|`
- Nil-Coalescing precedence: `??`
- Relational precedence: `<`, `<=`, `>`, `>=`
- Equality precedence: `==`, `!=`
- Logical conjunction precedence: `&&`
- Logical disjunction precedence: `||`
- Ternary precedence: `? :`

All operators are left-associative, except for the following operators which are right-associative:
- Ternary operator
- Nil-coalescing operator

Expressions can be wrapped in parentheses to override precedence conventions,
i.e. an alternate order should be indicated, or when the default order should be emphasized
e.g. to avoid confusion.
For example, `(2 + 3) * 4` forces addition to precede multiplication,
and `5 + (6 * 7)` reinforces the default order.
