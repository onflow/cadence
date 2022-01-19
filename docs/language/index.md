---
title: The Cadence Programming Language
---

## Introduction

The Cadence Programming Language is a new high-level programming language
intended for smart contract development.

The language's goals are, in order of importance:

- **Safety and security**:
  Provide a strong static type system, design by contract (preconditions and postconditions),
  and resources (inspired by linear types).

- **Auditability**:
  Focus on readability: Make it easy to verify what the code is doing,
  and make intentions explicit, at a small cost of verbosity.

- **Simplicity**: Focus on developer productivity and usability:
  Make it easy to write code, provide good tooling.

## Terminology

In this document, the following terminology is used to describe syntax
or behavior that is not allowed in the language:

- `Invalid` means that the invalid program will not even be allowed to run.
  The program error is detected and reported statically by the type checker.

- `Run-time error` means that the erroneous program will run,
  but bad behavior will result in the execution of the program being aborted.

## Syntax and Behavior

Much of the language's syntax is inspired by Swift, Kotlin, and TypeScript.

Much of the syntax, types, and standard library is inspired by Swift,
which popularized e.g. optionals, argument labels,
and provides safe handling of integers and strings.

Resources are based on linear types which were popularized by Rust.

Events are inspired by Solidity.

**Disclaimer:** In real Cadence code, all type definitions and code
must be declared and contained in [contracts](https://docs.onflow.org/cadence/language/contracts) or [transactions](https://docs.onflow.org/cadence/language/transactions),
but we omit these containers in examples for simplicity.
