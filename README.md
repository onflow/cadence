[![License](https://img.shields.io/github/license/onflow/cadence?color=blue)](https://github.com/onflow/cadence/blob/master/LICENSE)
[![Release](https://img.shields.io/github/v/release/onflow/cadence)](https://github.com/onflow/cadence/releases)
[![Language](https://img.shields.io/github/languages/top/onflow/cadence)](https://github.com/onflow/cadence)
[![Last Commit](https://img.shields.io/github/last-commit/onflow/cadence)](https://github.com/onflow/cadence/commits/master)
[![CI](https://github.com/onflow/cadence/actions/workflows/ci.yml/badge.svg)](https://github.com/onflow/cadence/actions/workflows/ci.yml)
[![Discord](https://img.shields.io/discord/613813861610684416?label=Discord&logo=discord&logoColor=white)](https://discord.gg/flow)

# Cadence — Flow's Resource-Oriented Smart Contract Language

<img src="https://raw.githubusercontent.com/onflow/cadence/master/cadence_furever.png" width="300" alt="Cadence programming language logo" />

## TL;DR

- **What:** Cadence is the resource-oriented smart contract programming language used on the [Flow network](https://flow.com).
- **Who it's for:** developers building on Flow who want compile-time safety, resource semantics, and capability-based access control. Also: teams coming from Solidity who have been burned by reentrancy, unlimited approvals, or MEV.
- **Why use it:** Cadence prevents entire vulnerability classes at the compiler level. Assets are resources, not ledger entries. Reentrancy attacks are significantly mitigated by Cadence's resource ownership model. Approvals are scoped.
- **Status:** Cadence 1.0, current release [v1.10.2](https://github.com/onflow/cadence/releases). Live on Flow mainnet.
- **License:** Apache 2.0.
- **Get started:** [Cadence Playground (browser)](https://play.flow.com/) or [cadence-lang.org](https://cadence-lang.org) for the language reference. Open-sourced since 2019.

## Introduction

Cadence is the native smart contract programming language of the [Flow network](https://flow.com). It is resource-oriented, capability-based, and designed by smart contract engineers to prevent the vulnerability classes that plague Solidity development.

Core design principles:

- **Resource-oriented programming.** Assets are first-class resources with move semantics that cannot be duplicated, implicitly destroyed, or accessed after being moved. The compiler enforces this, not developer discipline.
- **Capability-based security.** Fine-grained access control via entitlements. Functions are restricted to callers holding specific authorizations at compile time.
- **Type safety.** Strong static typing with type inference. No runtime type surprises.
- **Upgradeable by default.** Contracts are upgradeable with enforced backward compatibility. No proxy pattern needed.
- **Reentrancy significantly mitigated.** When a resource transfers, the caller's reference is invalidated at runtime, eliminating the most common reentrancy attack vectors.

## Features

- Type safety and a strong static type system
- Resource-oriented programming, a new paradigm that pairs linear types with object capabilities to create a secure and declarative model for digital ownership by ensuring that resources (and their associated assets) can only exist in one location at a time, cannot be copied, and cannot be accidentally lost or deleted
- Built-in pre-conditions and post-conditions for functions and transactions
- Capability-based security, which enforces access control by requiring that access to objects is restricted to only the owner and those who have a valid reference to the object
- Entitlements for fine-grained access control on references
- Contract upgradability with enforced backward compatibility

## Example

Here is a minimal Cadence contract that declares a public `hello()` function. This is the same example used by the [`@onflow/cadence-parser`](./npm-packages/cadence-parser) package to demonstrate parsing:

```cadence
access(all) contract HelloWorld {
    access(all) fun hello() {
        log("Hello, world!")
    }
}
```

For a full walkthrough of resources, capabilities, and transactions, see the [Cadence tutorial on cadence-lang.org](https://cadence-lang.org/docs/tutorial/hello-world).

## Getting Started

To get started writing Cadence, try the [Cadence Playground (browser)](https://play.flow.com/).

If you want to develop locally, use these tools:

- [Flow CLI](https://github.com/onflow/flow-cli) — the primary tool for building on Flow; includes the [Flow emulator](https://github.com/onflow/flow-emulator), a lightweight tool that emulates the behaviour of the real Flow network.
- [VS Code Cadence extension](https://github.com/onflow/vscode-cadence) — enables development, deployment, and interaction with Cadence contracts.

## Documentation

The canonical language reference is at [cadence-lang.org](https://cadence-lang.org). Additional developer guides, tutorials, and integration docs live on the [Flow Developer Portal](https://developers.flow.com).

Development documentation specific to the Cadence implementation can be found in the [`/docs` directory](./docs).

## FAQ

### What is Cadence?

Cadence is the native smart contract programming language of the [Flow network](https://flow.com). It is resource-oriented, capability-based, and designed to prevent entire vulnerability classes at compile time rather than relying on developer discipline.

### How is Cadence different from Solidity?

Cadence treats assets as first-class resources with move semantics enforced by the compiler. Resources cannot be duplicated, implicitly destroyed, or accessed after being moved. Reentrancy attacks are significantly mitigated by Cadence's resource ownership model because references are invalidated at runtime when a resource transfers. Access control is capability-based via entitlements rather than `msg.sender` checks.

### What is resource-oriented programming?

Resource-oriented programming pairs linear types with object capabilities. A resource can only exist in one location at a time, cannot be copied, and cannot be accidentally lost or deleted. This models digital ownership declaratively and is enforced by the compiler.

### What is Cadence 1.0?

Cadence 1.0 is the current major language version. It introduced entitlements for fine-grained access control on references, view functions, and a number of other language improvements. The latest release is [v1.10.2](https://github.com/onflow/cadence/releases).

### Does Cadence require a specific Flow client?

No. Any Flow Access Node can accept Cadence transactions. For local development, use the [Flow CLI](https://github.com/onflow/flow-cli) and the bundled [Flow emulator](https://github.com/onflow/flow-emulator).

### Can I use Cadence outside Flow?

The Cadence implementation in this repository is designed as the runtime for the Flow network. The language and its interpreter are licensed under Apache 2.0, so forking and embedding are permitted, but the canonical execution environment is Flow.

### How do I learn Cadence?

Start with [cadence-lang.org](https://cadence-lang.org) for the language reference, work through the [Cadence tutorial](https://cadence-lang.org/docs/tutorial/first-steps), and try examples in the [Cadence Playground (browser)](https://play.flow.com/). The [Flow Developer Portal](https://developers.flow.com) covers end-to-end application development.

### Where is the language specification?

The language reference and specification materials live at [cadence-lang.org](https://cadence-lang.org) and in the [`/docs` directory](./docs) of this repository.

### Where do I report a security issue?

See [SECURITY.md](./SECURITY.md) for the responsible disclosure process. Do not open public issues for vulnerabilities.

## Contributing

If you would like to contribute to Cadence, have a look at the [contributing guide](./CONTRIBUTING.md).

You can also join the next [Cadence Working Group](https://github.com/onflow/Flow-Working-Groups/tree/main/cadence_language_and_execution_working_group) meeting to participate in language design discussions.

## About Flow

Cadence is the native language of the [Flow network](https://flow.com), a Layer 1 blockchain built for consumer applications, AI Agents, and DeFi at scale. Flow powers NBA Top Shot, NFL All Day, Disney Pinnacle (built by Dapper Labs), and Ticketmaster NFT ticketing, all in live production.

- Language reference: [cadence-lang.org](https://cadence-lang.org)
- Developer docs: [developers.flow.com](https://developers.flow.com)
- Community: [Flow Discord](https://discord.gg/flow) · [Flow Forum](https://forum.flow.com)
- Governance: [Flow Improvement Proposals](https://github.com/onflow/flips)
