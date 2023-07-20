# Roadmap

Cadence is still in development. This roadmap documents the plans and ideas for the language,
categorized into immediate/high priority items and lower priority items,
listed in no particular order.

## High Priority

- Reliability

  Cadence should run deterministically, and should not have crashers, stack overflows, or security issues.

- Performance

  Cadence's checker is currently not optimized for performance.
  We are making performance improvements, mainly by profiling it using real-world programs
  and optimizing hot paths, as well as avoiding unnecessary work (e.g. tracking position information).

  Cadence programs are also currently executed using a tree-walking interpreter,
  which is easy to modify and useful for debugging. However, it is not optimized for performance.
  We are investigating  compilation to improve performance.
  Potential targets / inspirations are WebAssembly, MoveVM, and IELE.

## Lower Priority

- [Testing of Cadence programs](https://github.com/onflow/cadence/issues/330)

  Cadence should provide means to test code.

- Storage API

  - [Storage querying API](https://github.com/onflow/cadence/issues/208)

    Cadence should provide an API to query/iterate over account storage.

  - [Storage API improvements](https://github.com/onflow/cadence/issues/376)

    Cadence should provide APIs to overwrite and remove stored values.

  - [Scripts should have access to authorized accounts](https://github.com/onflow/cadence/issues/539)


- Extensibility

  Cadence should provide means to extend existing types with additional functionality
  even when those types did not explicitly provide mechanisms to do so.
  This would be very useful and would increase future-proofing code.

  However, it might have a negative impact on explicitness/discoverability,
  i.e it might be hard for a user to understand where a definition originates from
  if it wasn't defined in the type's original set of functions.
  It might also have system and security implications.
  A solution needs to take these issues into account.

- Host interface

  The current API that allows Cadence to be integrated into a host environment,
  such as the Flow Execution Node, the Flow Emulator, or Flow Playground,
  is not flexible enough and difficult to extend.

  Refactor the pull-based architecture of the current interface to an injection-based architecture.

  Move non-essential type and value declarations out of the core Cadence code.

- [Code formatter / pretty printing of code](https://github.com/onflow/cadence/issues/209)

  Cadence should offer a tool that formats programs.

- [Documentation generator](https://github.com/onflow/cadence/issues/339)

  Cadence should offer a tool that generates human-readable documentation for programs.

- Improving type inference

  - [Improve the inferred type of conditional statements and expressions](http://github.com/onflow/cadence/issues/61),
    binary expressions and literal expressions (e.g. arrays and dictionaries).

- Type aliases

  Cadence should provide a way to define type aliases.

- `Word128` and `Word256` types

  Cadence should provide `Word128` and `Word256` types, just like it provides `UInt128` and `UInt256`

- ABI generation and code generation

  Cadence should offer a tool to generate an ABI file, a description
  of the functionality of a contract.

  An early version of Cadence provided such a tool,
  however it was not used, became unmaintained, and was eventually removed.

  The ABI generation tool could be revived and missing features could be added.

  From an ABI source code could be generated that would allow client libraries
  to call Cadence programs in a type-safe way.

- Code size reduction

  Cadence programs are currently stored in source code form on-chain.

  Cadence should offer a more efficient format that is optimized for size and read time.

- Allow import statements to specify the hash of the imported contract

  Cadence programs can import other programs.
  However, contract code might change, which could lead to bugs and even security problems.

  Cadence should provide a way to specify the expected hash of the imported contract
  in the import statement, so that the runtime can check that the actual hash matches
  the expected hash, and abort execution if it does not.

- Add conversion semantics to failable casting operator `as`?

  Cadence's failable casting operator `as?` should allow conversion
  just like the static casting operator `as` does.

- Overloading based on argument labels

  Cadence should allow overloading of functions and initializers based on argument labels.
  For example, this enables initializers which initialize some fields with default values.

  This is sane from a developer/user perspective, as it's clear at the call-site
  which function is called.

- Set data structure

  Cadence should provide a built-in set collection type. It would only be useful for value types,
  as resource types are already guaranteed to be unique within the whole system,
  and lots of set operators don’t make sense for a set of resources.

  Built-in set types are fairly common in other languages, including literal syntax.

  In Python the literal syntax is curly braces.
  However, curly braces are also used for dictionaries, so `{1}` is a set,
  but `{}` is considered an empty dictionary.

  Nim also [has sets and uses braces for set literals](https://nim-lang.org/docs/manual.html#types-set-type).

  Swift also [has sets](https://developer.apple.com/documentation/swift/set)
  and the array literal syntax is reused
  (in fact, there is a mechanism to initialize arbitrary data structures with built-in literals,
  see [ExpressibleByArrayLiteral](https://developer.apple.com/documentation/swift/expressiblebyarrayliteral)).

  Another thing to note about Swift: The syntax for data structure types reflects
  the syntax of literals: The array `[1]` has type `[Int]`,
  the dictionary `["two": 2]` has type `[String: Int]`.
  We could adopt this and add curly braces for sets.

- XOR operator

  Cadence should provide an XOR operator (`^`): logical for booleans and bitwise for integers.

- Debugger

  Cadence should offer a debugger, which would assist developers with debugging issues.

  This could be done as a command line tool, potentially integrated into the command line runner
  and/or REPL.

  Another opportunity could be implementing the debugger as a server process
  that implements the
  [Debug Adapter Protocol](https://microsoft.github.io/debug-adapter-protocol/),
  which would allow multiple editors to debug Cadence programs,
  just like the language server implements the Language Server Protocol
  to allow different editors to provide editing features for Cadence code.

- Loose mode / Gradual typing

  Cadence should have a mode that does not require type annotations and which performs
  fewer or no type checks.
  This allows the fast development of a program and progressively moving towards
  a more correct and safe implementation of it.

  Gradual typing enables developers to write a program without types
  and gradually type parts of the program.
  It has been proposed to be retrofitted to a few languages,
  and TypeScript really succeed here, so we should learn from it.
  However, its type-system is also unsound and it is very complex/feature-full,
  so we might not want to adopt everything whole-heartedly.

- Destructuring, pattern matching

  Cadence should provide a way to destructure and pattern match values.
  This would reduce boilerplate and improve safety and readability.

  Destructuring is basically a special case (one pattern) of pattern matching:
  accessing elements of a data structure (e.g. array, dictionary, struct, etc.)
  and binding it to identifiers in a variable/constant declaration.

  Pattern matching would be an enhancement allowing this functionality in the cases of a switch-statement.

- Distinct types

  Cadence should offer a way to declare distinct types,
  i.e., types that are derived from an existing type,
  but are not compatible with them could improve safety.
  For example, instead of using a string for a person's ID,
  a new type `PersonID` can be derived from string.
  This improves safety, as for example another random string (e.g. a first name)
  can't be confused as a person's ID.

  This is similar to [`newtype` in Haskell](https://wiki.haskell.org/Newtype)
  and [`NewType` in Python](https://docs.python.org/3/library/typing.html#newtype).

- Add `Hashable` interface

  Cadence should allow user types to be used as dictionary keys.

  Implement the `Hashable` interface as described in the documentation.

- Add `Equatable` interface

  Cadence should allow user types to be equated,
  i.e. used with the equality operators `==` and `!=`.

  Implement the `Equatable` interface as described in the documentation.

- Optimize value semantics, perform copy-on-write

  Cadence has pass-by-value semantics.

  Optimize the performance by reducing the number of actual copies through copy-on-write.

- Interface requirements

  Cadence should allow interfaces to require conforming types to also conform to other interfaces,
  e.g. the interface declaration `interface I3: I1, I2 {}` requires conforming types
  to also conform to interfaces `I1` and `I2`.

- Built-in types to work with timestamps and durations

  Cadence should offer two new built in types: `Timestamp` and `Duration`,
  each representing the number of seconds since the Unix Epoch (00:00:00 UTC on 1 January 1970).

  These types would work almost the same as the `Int64` type, with the following extra rules:

  - `Timestamp`s
    - Can not be multiplied or divided with each other, or with integers.
    - You can add a timestamp and a duration together, and the result is a timestamp.
    - You can subtract a duration from a timestamp, and the result is a timestamp.

  - `Duration`s:
    - Can not be multiplied or divided with each other, but can be multipled by, or divided by,   `Int64` (which results in a duration)

    - Can be added or subtracted with each other, which results in a duration

  In addition, Cadence should provide the built-in duration constants `second`, `minute`, `hour`, `day` and `year`.
  Ideally, there would be a way to create a timestamp constant from an ISO-8601 string (e.g. `2020-03-13T19:52:43Z`),
  with only the UTC timezone allowed (indicated with the optional trailing `Z`).

  Another idea is to use the `Fix64` type, which gives us a date range of approximately
  1500AD - 2250AD, with accuracy to 100 millionth of a second.
  Obviously, that kind of accuracy is not needed, and a higher date range seems useful.
  On the other hand, having some level of sub-second accuracy
  might be useful in the blocktime field,
  given that the block rate might be >1 block/sec at some point.

- Exposing entropy, safe random functionality

  Cadence should provide a way to get safe random numbers.

  This could potentially be based on a callback mechanism,
  which depends on contexts and a service chunk in the block.

- Re-entrancy

  Cadence should provide means to prevent re-entrancy attacks.
  To achieve this, enforce that borrows are exclusive.
  In a transition period, warn instead of aborting,
  and analyze behaviour on chain, as well as ask the community to report cases.

  This is similar to [Swift's Exclusivity Enforcement](https://swift.org/blog/swift-5-exclusivity/).

- Contexts

  Cadence should provide a way to execute a block of code that can potentially abort/fail/revert,
  and recover from the failure.

  This is similar to a try-catch block, however, on failure,
  storage changes and in-memory state changes (!) would reverted.

- Improve Static Analysis

  Cadence should offer means to make it statically analyzable.

  For example, a subset of pre and post-conditions of functions and transactions
  could be statically checked.

- Improve the development experience in editors by improving the Language Server

  - [Add support for code actions](https://github.com/onflow/cadence/issues/532),
    e.g. renaming, refactoring, code changes (switch access modifier),
    add return type through return statement, etc.

  - [Support more code-suggestions/auto-completions](https://github.com/onflow/cadence/issues/531)

- [Make the Crypto contract compatible with Ethereum](https://github.com/onflow/cadence/issues/537)

  Cadence should offer a mechanism to verify an Ethereum signature, ideally a personal signed message.

  For this to be possible, the Crypto contract needs support for the Keccak hashing algorithm,
  and the signature verification method must allow providing a custom tag,
  e.g. the Ethereum personal signed message prefix.

- Extend paths

  Cadence should [allow values like types, numbers, and addresses to be included in paths](https://github.com/onflow/cadence/issues/538).
  Currently paths only consist of a domain and an identifier.

- Extend run-time types

  Cadence should [allow run-time types to be tested for subtyping relationships](https://github.com/onflow/cadence/issues/473)
