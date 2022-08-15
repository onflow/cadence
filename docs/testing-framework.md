# Cadence Testing Framework

Cadence testing framework provides a way to write tests for Cadence programs in Cadence.
This functionality is provided by the Test standard-library.

Note: Test standard-library can only be used off-chain. e.g: In CLI

## Test Standard Library

Testing standard library can be imported directly into the test script.
```cadence
import Test
```

## Assertion
### assert
```cadence
fun assert(_ condition: Bool, _ message: String)
```
Fails a test-case if the given condition is false, and reports a message which explains how the condition is false.

The message argument is optional.

### fail
```cadence
fun fail(_ message: string)
```
Immediately fails a test-case, with a message explaining the reason to fail the test.

The message argument is optional.

### expect
The `expect` function tests a value against a matcher (see [matchers](#matchers) section), and fails the test if it's not a match.

```cadence
fun expect(_ value: Any, _ matcher: Matcher)
```

## Matchers
A matcher is an object that consists of a test function and associated utility functionality.
```cadence
pub struct interface Matcher {

    pub fun test(_ value: Any): Bool

    pub fun and(_ other: AnyStruct{Matcher}): AnyStruct{Matcher}

    pub fun or(_ other: AnyStruct{Matcher}): AnyStruct{Matcher}
}
```

The `test` function defines the evaluation criteria for a value, and returns a boolean indicating whether the value
conforms to the test criteria defined in the function.

The `and` and `or` functions can be used to combine this matcher with another matcher to produce a new matcher with
multiple testing criteria.
The `and` method returns a new matcher that succeeds if both this and the given matcher are succeeded.
The `or` method returns a new matcher that succeeds if at-least this or the given matcher is succeeded.

Cadence test standard library comes with a `DefaultMatcher`, which is the default implementation of the
`Matcher` interface.

A default matcher can be constructed using the `NewMatcher` function.
```cadence
fun NewMatcher<T>(_ testFunction: ((T): Bool)): AnyStruct<Test.Matcher>
```
The type parameter `T` is bound to `AnyStruct` type. It is also optional.

#### Example:
A matcher that checks whether the given integer value is a negative value.
```cadence
let matcher = Test.NewMatcher(fun (_ value: Int): Bool {
    return value < 0
})
```

### Built-in matcher functions
Cadence test standard library provides some built-in matcher functions for convenience.

- `fun equal(_ value: Any): Matcher`

  Returns a matcher that succeeds if the tested value is equal to the given value.
  Accepts `AnyStruct` or `AnyResource` value.

