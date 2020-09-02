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

## Negation

The `-` unary operator negates an integer:

```cadence
let a = 1
-a  // is `-1`
```

The `!` unary operator logically negates a boolean:

```cadence
let a = true
!a  // is `false`
```

## Assignment

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

## Swapping

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

## Arithmetic

There are four arithmetic operators:

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

## Logical Operators

Logical operators work with the boolean values `true` and `false`.

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

## Comparison operators

Comparison operators work with boolean and integer values.

- Equality: `==`, for booleans and integers

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

- Inequality: `!=`, for booleans and integers (possibly optional)

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

## Precedence and Associativity

Operators have the following precedences, highest to lowest:

- Multiplication precedence: `*`, `&*`, `/`, `%`
- Addition precedence: `+`, `&+`, `-`, `&-`
- Relational precedence: `<`, `<=`, `>`, `>=`
- Equality precedence: `==`, `!=`
- Logical conjunction precedence: `&&`
- Logical disjunction precedence: `||`
- Ternary precedence: `? :`

All operators are left-associative, except for the ternary operator, which is right-associative.

Expressions can be wrapped in parentheses to override precedence conventions,
i.e. an alternate order should be indicated, or when the default order should be emphasized
e.g. to avoid confusion.
For example, `(2 + 3) * 4` forces addition to precede multiplication,
and `5 + (6 * 7)` reinforces the default order.
