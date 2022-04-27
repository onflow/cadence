---
title: Control Flow
---

Control flow statements control the flow of execution in a function.

## Conditional branching: if-statement

If-statements allow a certain piece of code to be executed only when a given condition is true.

The if-statement starts with the `if` keyword, followed by the condition,
and the code that should be executed if the condition is true
inside opening and closing braces.
The condition expression must be boolean.
The braces are required and not optional.
Parentheses around the condition are optional.

```cadence
let a = 0
var b = 0

if a == 0 {
   b = 1
}

// Parentheses can be used around the condition, but are not required.
if (a != 0) {
   b = 2
}

// `b` is `1`
```

An additional, optional else-clause can be added to execute another piece of code
when the condition is false.
The else-clause is introduced by the `else` keyword followed by braces
that contain the code that should be executed.

```cadence
let a = 0
var b = 0

if a == 1 {
   b = 1
} else {
   b = 2
}

// `b` is `2`
```

The else-clause can contain another if-statement, i.e., if-statements can be chained together.
In this case the braces can be omitted.

```cadence
let a = 0
var b = 0

if a == 1 {
   b = 1
} else if a == 2 {
   b = 2
} else {
   b = 3
}

// `b` is `3`

if a == 1 {
   b = 1
} else {
    if a == 0 {
        b = 2
    }
}

// `b` is `2`
```

## Optional Binding

Optional binding allows getting the value inside an optional.
It is a variant of the if-statement.

If the optional contains a value, the first branch is executed
and a temporary constant or variable is declared and set to the value contained in the optional;
otherwise, the else branch (if any) is executed.

Optional bindings are declared using the `if` keyword like an if-statement,
but instead of the boolean test value, it is followed by the `let` or `var` keywords,
to either introduce a constant or variable, followed by a name,
the equal sign (`=`), and the optional value.

```cadence
let maybeNumber: Int? = 1

if let number = maybeNumber {
    // This branch is executed as `maybeNumber` is not `nil`.
    // The constant `number` is `1` and has type `Int`.
} else {
    // This branch is *not* executed as `maybeNumber` is not `nil`
}
```

```cadence
let noNumber: Int? = nil

if let number = noNumber {
    // This branch is *not* executed as `noNumber` is `nil`.
} else {
    // This branch is executed as `noNumber` is `nil`.
    // The constant `number` is *not* available.
}
```

## Switch

Switch-statements compare a value against several possible values of the same type, in order.
When an equal value is found, the associated block of code is executed.

The switch-statement starts with the `switch` keyword, followed by the tested value,
followed by the cases inside opening and closing braces.
The test expression must be equatable.
The braces are required and not optional.

Each case is a separate branch of code execution
and starts with the `case` keyword,
followed by a possible value, a colon (`:`),
and the block of code that should be executed
if the case's value is equal to the tested value.

The block of code associated with a switch case
[does not implicitly fall through](#no-implicit-fallthrough),
and must contain at least one statement.
Empty blocks are invalid.

An optional default case may be given by using the `default` keyword.
The block of code of the default case is executed
when none of the previous case tests succeeded.
It must always appear last.

```cadence
fun word(_ n: Int): String {
    // Test the value of the parameter `n`
    switch n {
    case 1:
        // If the value of variable `n` is equal to `1`,
        // then return the string "one"
        return "one"
    case 2:
        // If the value of variable `n` is equal to `2`,
        // then return the string "two"
        return "two"
    default:
        // If the value of variable `n` is neither equal to `1` nor to `2`,
        // then return the string "other"
        return "other"
    }
}

word(1)  // returns "one"
word(2)  // returns "two"
word(3)  // returns "other"
word(4)  // returns "other"
```

### Duplicate cases

Cases are tested in order, so if a case is duplicated,
the block of code associated with the first case that succeeds is executed.

```cadence
fun test(_ n: Int): String {
    // Test the value of the parameter `n`
    switch n {
    case 1:
        // If the value of variable `n` is equal to `1`,
        // then return the string "one"
        return "one"
    case 1:
        // If the value of variable `n` is equal to `1`,
        // then return the string "also one".
        // This is a duplicate case for the one above.
        return "also one"
    default:
        // If the value of variable `n` is neither equal to `1` nor to `2`,
        // then return the string "other"
        return "other"
    }
}

word(1) // returns "one", not "also one"
```

### `break`

The block of code associated with a switch case may contain a `break` statement.
It ends the execution of the switch statement immediately
and transfers control to the code after the switch statement

### No Implicit Fallthrough

Unlike switch statements in some other languages,
switch statements in Cadence do not "fall through":
execution of the switch statement finishes as soon as the block of code
associated with the first matching case is completed.
No explicit `break` statement is required.

This makes the switch statement safer and easier to use,
avoiding the accidental execution of more than one switch case.

Some other languages implicitly fall through
to the block of code associated with the next case,
so it is common to write cases with an empty block
to handle multiple values in the same way.

To prevent developers from writing switch statements
that assume this behaviour, blocks must have at least one statement.
Empty blocks are invalid.

```cadence
fun words(_ n: Int): [String] {
    // Declare a variable named `result`, an array of strings,
    // which stores the result
    let result: [String] = []

    // Test the value of the parameter `n`
    switch n {
    case 1:
        // If the value of variable `n` is equal to `1`,
        // then append the string "one" to the result array
        result.append("one")
    case 2:
        // If the value of variable `n` is equal to `2`,
        // then append the string "two" to the result array
        result.append("two")
    default:
        // If the value of variable `n` is neither equal to `1` nor to `2`,
        // then append the string "other" to the result array
        result.append("other")
    }
    return result
}

words(1)  // returns `["one"]`
words(2)  // returns `["two"]`
words(3)  // returns `["other"]`
words(4)  // returns `["other"]`
```

## Looping

### while-statement

While-statements allow a certain piece of code to be executed repeatedly,
as long as a condition remains true.

The while-statement starts with the `while` keyword, followed by the condition,
and the code that should be repeatedly
executed if the condition is true inside opening and closing braces.
The condition must be boolean and the braces are required.

The while-statement will first evaluate the condition.
If it is true, the piece of code is executed and the evaluation of the condition is repeated.
If the condition is false, the piece of code is not executed
and the execution of the whole while-statement is finished.
Thus, the piece of code is executed zero or more times.

```cadence
var a = 0
while a < 5 {
    a = a + 1
}

// `a` is `5`
```

### For-in statement

For-in statements allow a certain piece of code to be executed repeatedly for
each element in an array.

The for-in statement starts with the `for` keyword, followed by the name of
the element that is used in each iteration of the loop,
followed by the `in` keyword, and then followed by the array
that is being iterated through in the loop.

Then, the code that should be repeatedly executed in each iteration of the loop
is enclosed in curly braces.

If there are no elements in the data structure, the code in the loop will not
be executed at all. Otherwise, the code will execute as many times
as there are elements in the array.

```cadence
let array = ["Hello", "World", "Foo", "Bar"]

for element in array {
    log(element)
}

// The loop would log:
// "Hello"
// "World"
// "Foo"
// "Bar"
```

Optionally, developers may include an additional variable preceding the element name, 
separated by a comma. 
When present, this variable contains the current
index of the array being iterated through 
during each repeated execution (starting from 0).

```cadence
let array = ["Hello", "World", "Foo", "Bar"]

for index, element in array {
    log(index)
}

// The loop would log:
// 0
// 1
// 2
// 3
```

To iterate over a dictionary's entries (keys and values), 
use a for-in loop over the dictionary's keys and get the value for each key:

```cadence
let dictionary = {"one": 1, "two": 2}
for key in dictionary.keys {
    let value = dictionary[key]!
    log(key)
    log(value)
}

// The loop would log:
// "one"
// 1
// "two"
// 2
```

### `continue` and `break`

In for-loops and while-loops, the `continue` statement can be used to stop
the current iteration of a loop and start the next iteration.

```cadence
var i = 0
var x = 0
while i < 10 {
    i = i + 1
    if i < 3 {
        continue
    }
    x = x + 1
}
// `x` is `8`


let array = [2, 2, 3]
var sum = 0
for element in array {
    if element == 2 {
        continue
    }
    sum = sum + element
}

// `sum` is `3`

```

The `break` statement can be used to stop the execution
of a for-loop or a while-loop.

```cadence
var x = 0
while x < 10 {
    x = x + 1
    if x == 5 {
        break
    }
}
// `x` is `5`


let array = [1, 2, 3]
var sum = 0
for element in array {
    if element == 2 {
        break
    }
    sum = sum + element
}

// `sum` is `1`
```

## Immediate function return: return-statement

The return-statement causes a function to return immediately,
i.e., any code after the return-statement is not executed.
The return-statement starts with the `return` keyword
and is followed by an optional expression that should be the return value of the function call.

<!--
TODO: examples

- in function
- in while
- in if
-->
