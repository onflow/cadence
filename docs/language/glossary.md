
<Callout type="info">
Tip: <kbd>CTRL</kbd>/<kbd>⌘</kbd> + <kbd>F</kbd> and type in the symbol or operator you want to look up.
</Callout>

## `&` (ampersand)

The `&` (ampersand) symbol has several uses.

### Reference

If an expression starts with the `&` (ampersand) symbol, it creates a [reference](references).

```cadence
let a: String = "hello"
let refOfA: &String = &a as &String
```

References may also be authorized if the `&` symbol is preceded by `auth` (otherwise the reference is unauthorized).

Authorized references have the `auth` modifier, i.e. the full syntax is `auth &T`,
whereas unauthorized references do not have a modifier.

```cadence
let a: String = "hello"
let refOfA: &String = &a as auth &String
```

### Logical Operator

It can be also used as a [logical operator (AND)](operators#logical-operators),
by appearing twice in succession (i.e. `&&`):

```cadence
let a = true
let b = false

let c = a && b // false
```

## `@` (at)

The `@` (at) symbol before a type is used to annotate whether the type is a [resource](resources).

The `@` symbol must appear at the beginning of the type, not inside.
For example, an array of `NFT`s is `@[NFT]`, not `[@NFT]`.
This emphasizes the whole type acts like a resource.

```cadence
// Declare a resource named `SomeResource`
pub resource SomeResource {
    pub var value: Int

    init(value: Int) {
        self.value = value
    }
}

// we use the '@' symbol to reference a resource type
let a: @SomeResource <- create SomeResource(value: 0)

// also in functions declarations
pub fun use(resource: @SomeResource) {
    destroy resource
}
```

## `:` (colon)

The `:` (colon) symbol has several uses.

### Type Declaration

If a `:` (colon) follows a variable/constant/function declaration, it is used to declare its type.

```cadence
let a: Bool = true // declares variable `a` with type `Bool`

// or

fun addOne(x: Int): Int { // return type of Int
    return x + 1
}
```

### Ternary Conditional Operator

The `:` (colon) is also be used in [ternary operations](operators#ternary-conditional-operator) to represent the "otherwise" section,
such as the following:

```cadence
let a = 1 > 2 ? 3 : 4
// should be read as:
//   "is 1 greater than 2?"
//   "if YES, then set a = 3,
//   "otherwise, set a = 4.
```

## `=` (equals)

The `=` (equals) symbol has several uses.

### Variable Declaration

```cadence
let a = 1 // declares a variable `a` with value `1`
```

### Assignment

```cadence
a = 1  // assigns the value `1` to variable `a `
```

## `!` (exclamation mark)

The `!` (exclamation mark) symbol has a different effect whether it precedes or succeeds a variable.

When it immediately **precedes** a boolean-type variable, it negates it.

```cadence
let a: Bool = true
let b: Bool = !a

// b is false
```

When it immediately **succeeds** an *optional* variable, it [force-unwraps](operators#force-unwrap-operator-) it.
Force-unwrapping returns the value inside an optional if it contains a value,
or panics and aborts the execution if the optional has no value, i.e. the optional value is nil.

```cadence
let a: Int? = nil
let b: Int? = 3

let c: Int = a! // panics, because = nil
let d: Int = b! // initialized correctly as 3
```

## `/` (forward slash)

The `/` (forward slash) symbol has several uses.

### Division Operator

Inbetween two expressions, the forward slash acts as the [division operator](operators#arithmetic-operators).

```cadence
let result = 4 / 2
```

### Path separator

In a [Path](accounts#paths), the forward slash separates the domain (e.g. `storage`, `private`, `public`) and the identifier.

```cadence
let storagePath = /storage/path
storagePath.toString()  // is "/storage/path"
```

## `<-` (lower than, hyphen) (Move operator)

The [move operator `<-`](resources#the-move-operator--) is like the assignment operator `=`,
but must be used when the value is a [resource](resources).
To make assignment of resources explicit, the move operator `<-` must be used when:

- The resource is the initial value of a constant or variable,
- The resource is moved to a different variable in an assignment,
- The resource is moved to a function as an argument
- The resource is returned from a function.

```cadence
resource R {}

let a <- create R() // we instantiate a new resource and move it into a
```

## `<-!` (lower than, hyphen, exclamation mark) (Force-assignment move operator)

The [force-assignment move operator `<-!`](operators#force-assignment-operator--) moves a resource value to an optional variable.
If the variable is `nil`, the move succeeds.
If it is not nil, the program aborts.

```cadence
pub resource R {}

var a: @R? <- nil
a <-! create R()
```

## `<->` (lower than, hyphen, greater than) (Swap operator)

The [swapping operator `<->`](operators#swapping-operator--) swaps two resource between the variables to the left and right of it.


## `+` (plus), `-` (minus), `*` (asterisk), `%` (percentage sign)

These are all typical [arithmetic operators](operators#arithmetic-operators):

- Addition: `+`
- Subtraction: `-`
- Multiplication: `*`
- Remainder: `%`

## `?` (question mark)

The `?` (question mark) symbol has several uses.

### Optional

If a `?` (question mark) follows a variable/constant, it represents an optional.
An optional can either have a value or *nothing at all*.

```cadence
// Declare a constant which has an optional integer type
//
let a: Int? = nil
```

### Ternary Conditional Operator

The `?` (question mark) is also be used in [ternary operations](operators#ternary-conditional-operator) to represent the "then" section,
such as the following:

```cadence
let a = 1 > 2 ? 3 : 4
// should be read as:
//   "is 1 greater than 2?"
//   "if YES, then set a = 3,
//   "otherwise, set a = 4.
```

### Nil-Coalescing Operator

The `?` (question mark) is also used in the [nil-coalescing operator `??`](operators#nil-coalescing-operator-).

It returns the value inside the optional, if the optional contains a value,
or returns an alternative value if the optional has no value, i.e., the optional value is nil.

```cadence
// Declare a constant which has an optional integer type
//
let a: Int? = nil

// Declare a constant with a non-optional integer type,
// which is initialized to `a` if it is non-nil, or 42 otherwise.
//
let b: Int = a ?? 42
// `b` is 42, as `a` is nil


// Invalid: nil-coalescing operator is applied to a value which has a non-optional type
// (the integer literal is of type `Int`).
//
let c = 1 ?? 2
```

## `_` (underscore)

The `_` (underscore) symbol has several uses.

### Names

The `_` (underscore) can be used in names, e.g. in variables and types.

```cadence
let _a = true // used as a variable name
let another_one = false
```

### Number Literals

The `_` (underscore) can also be used to split up numerical components.

```cadence
let b = 100_000_000 // used to split up a number (supports all number types, e.g. 0b10_11_01)
```

### Argument Labels

The `_` (underscore) can also be to indicate that a parameter in a [function](functions) has no argument label.

```cadence
// The special argument label _ is specified for the parameter,
// so no argument label has to be provided in a function call.

fun double(_ x: Int): Int {
    return x * 2
}

let result = double(4)
```
