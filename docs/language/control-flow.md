# Control flow

Control flow statements control the flow of execution in a function.

## Conditional branching: if-statement

If-statements allow a certain piece of code to be executed only when a given condition is true.

The if-statement starts with the `if` keyword, followed by the condition,
and the code that should be executed if the condition is true
inside opening and closing braces.
The condition expression must be Bool
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

## Looping

### while-statement

While-statements allow a certain piece of code to be executed repeatedly,
as long as a condition remains true.

The while-statement starts with the `while` keyword, followed by the condition,
and the code that should be repeatedly
executed if the condition is true inside opening and closing braces.
The condition must be boolean and the braces are required.

The while-statement will first evaluate the condition.
If the condition is false, the execution is done.
If it is true, the piece of code is executed and the evaluation of the condition is repeated.
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
var array = ["Hello", "World", "Foo", "Bar"]
for element in array {
    log(element)
}

// The loop would log:
// "Hello"
// "World"
// "Foo"
// "Bar"

```

### continue and break

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
