[![CI](https://github.com/onflow/cadence/actions/workflows/ci.yml/badge.svg)](https://github.com/onflow/cadence/actions/workflows/ci.yml)

# Cadence

<img src="https://raw.githubusercontent.com/onflow/cadence/master/cadence_furever.png" width="300" />

## Introduction

Cadence is a resource-oriented programming language that introduces new features
to smart contract programming that help developers ensure that their code is
safe, secure, clear, and approachable.

Some of these features are:

- Type safety and a strong static type system
- Resource-oriented programming, a new paradigm that pairs linear types with
  object capabilities to create a secure and declarative model for digital
  ownership by ensuring that resources (and their associated assets) can only
  exist in one location at a time, cannot be copied, and cannot be accidentally
  lost or deleted
- Built-in pre-conditions and post-conditions for functions and transactions
- The utilization of capability-based security, which enforces access control by
  requiring that access to objects is restricted to only the owner and those who
  have a valid reference to the object

## Getting Started

To get started writing Cadence, why not try out the
[Playground](https://play.onflow.org/)?

If you want to develop locally, use these tools: 
* [Flow CLI](https://github.com/onflow/flow-cli),
	which includes the [Flow emulator](https://github.com/onflow/flow-emulator). The emulator is a lightweight tool that emulates the behaviour of the real Flow network.
* [Visual Studio Code extension](https://github.com/onflow/vscode-cadence). The Visual Studio Code extension enables the development, deployment of, and interaction with Cadence contracts.



## Documentation

Check out the [Cadence Docs](https://docs.onflow.org/cadence/language/).

## Contributing

If you would like to contribute to Cadence, have a look at the [contributing guide](https://github.com/onflow/cadence/blob/master/CONTRIBUTING.md).

Development documentation can be found in the [/docs directory](https://github.com/onflow/flow/tree/master/docs).
For example, it contains the source for the language reference.
