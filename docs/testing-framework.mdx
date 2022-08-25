---
title: Cadence Testing Framework
---

The Cadence testing framework provides a convenient way to write tests for Cadence programs in Cadence.
This functionality is provided by the built-in `Test` contract.

<Callout type="info">
The testing framework can only be used off-chain, e.g. by using the [Flow CLI](https://developers.flow.com/tools/flow-cli).
</Callout>

## Test Standard Library

The testing framework can be used by importing the built-in `Test` contract:

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
fun expect(_ value: AnyStruct, _ matcher: Matcher)
```

## Matchers

A matcher is an object that consists of a test function and associated utility functionality.

```cadence
pub struct Matcher {

    pub let test: ((AnyStruct): Bool)

    pub init(test: ((AnyStruct): Bool)) {
        self.test = test
    }

    /// Combine this matcher with the given matcher.
    /// Returns a new matcher that succeeds if this and the given matcher succeed
    ///
    pub fun and(_ other: Matcher): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return self.test(value) && other.test(value)
        })
    }

    /// Combine this matcher with the given matcher.
    /// Returns a new matcher that succeeds if this and the given matcher succeed
    ///
    pub fun or(_ other: Matcher): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return self.test(value) || other.test(value)
        })
    }
}
```

The `test` function defines the evaluation criteria for a value, and returns a boolean indicating whether the value
conforms to the test criteria defined in the function.

The `and` and `or` functions can be used to combine this matcher with another matcher to produce a new matcher with
multiple testing criteria.
The `and` method returns a new matcher that succeeds if both this and the given matcher are succeeded.
The `or` method returns a new matcher that succeeds if at-least this or the given matcher is succeeded.

A matcher that accepts a generic-typed test function can be constructed using the `newMatcher` function.

```cadence
fun newMatcher<T: AnyStruct>(_ test: ((T): Bool)): Test.Matcher
```

The type parameter `T` is bound to `AnyStruct` type. It is also optional.

For example, a matcher that checks whether a given integer value is negative can be defined as follows:

```cadence
let isNegative = Test.newMatcher(fun (_ value: Int): Bool {
    return value < 0
})

// Use `expect` function to test a value against the matcher.
Test.expect(-15, isNegative)
```

### Built-in matcher functions

The `Test` contract provides some built-in matcher functions for convenience.

- `fun equal(_ value: AnyStruct): Matcher`

  Returns a matcher that succeeds if the tested value is equal to the given value.
  Accepts an `AnyStruct` value.


## Blockchain

A blockchain is an environment to which transactions can be submitted to, and against which scripts can be run.
It imitates the behavior of a real network, for testing.

```cadence
/// Blockchain emulates a real network.
///
pub struct Blockchain {

    pub let backend: AnyStruct{BlockchainBackend}

    init(backend: AnyStruct{BlockchainBackend}) {
        self.backend = backend
    }

    /// Executes a script and returns the script return value and the status.
    /// `returnValue` field of the result will be `nil` if the script failed.
    ///
    pub fun executeScript(_ script: String, _ arguments: [AnyStruct]): ScriptResult {
        return self.backend.executeScript(script, arguments)
    }

    /// Creates a signer account by submitting an account creation transaction.
    /// The transaction is paid by the service account.
    /// The returned account can be used to sign and authorize transactions.
    ///
    pub fun createAccount(): Account {
        return self.backend.createAccount()
    }

    /// Add a transaction to the current block.
    ///
    pub fun addTransaction(_ tx: Transaction) {
        self.backend.addTransaction(tx)
    }

    /// Executes the next transaction in the block, if any.
    /// Returns the result of the transaction, or nil if no transaction was scheduled.
    ///
    pub fun executeNextTransaction(): TransactionResult? {
        return self.backend.executeNextTransaction()
    }

    /// Commit the current block.
    /// Committing will fail if there are un-executed transactions in the block.
    ///
    pub fun commitBlock() {
        self.backend.commitBlock()
    }

    /// Executes a given transaction and commit the current block.
    ///
    pub fun executeTransaction(_ transaction: Transaction): TransactionResult {
        self.addTransaction(transaction)
        let txResult = self.executeNextTransaction()!
        self.commitBlock()
        return txResult
    }

    /// Executes a given set of transactions and commit the current block.
    ///
    pub fun executeTransactions(_ transactions: [Transaction]): [TransactionResult] {
        for tx in transactions {
            self.addTransaction(tx)
        }

        let results: [TransactionResult] = []
        for tx in transactions {
            let txResult = self.executeNextTransaction()!
            results.append(txResult)
        }

        self.commitBlock()
        return results
    }

    /// Deploys a given contract, and initilizes it with the arguments.
    ///
    pub fun deployContract(
        name: String,
        code: String,
        account: Account,
        arguments: [AnyStruct]
    ): Error? {
        return self.backend.deployContract(
            name: name,
            code: code,
            account: account,
            arguments: arguments
        )
    }
}
```

The `BlockchainBackend` provides the actual functionality of the blockchain.

```cadence
/// BlockchainBackend is the interface to be implemented by the backend providers.
///
pub struct interface BlockchainBackend {

    pub fun executeScript(_ script: String, _ arguments: [AnyStruct]): ScriptResult

    pub fun createAccount(): Account

    pub fun addTransaction(_ transaction: Transaction)

    pub fun executeNextTransaction(): TransactionResult?

    pub fun commitBlock()

    pub fun deployContract(
        name: String,
        code: String,
        account: Account,
        arguments: [AnyStruct]
    ): Error?
}
```

### Creating a blockchain

A new blockchain instance can be created using the `newEmulatorBlockchain` method.
It returns a `Blockchain` which is backed by a new [Flow Emulator](https://developers.flow.com/tools/emulator) instance.

```cadence
let blockchain = Test.newEmulatorBlockchain()
```

### Creating accounts

It may be necessary to create accounts during tests for various reasons, such as for deploying contracts, signing transactions, etc.
An account can be created using the `createAccount` function.

```cadence
let acct = blockchain.createAccount()
```

The returned account consist of the `address` of the account, and a `publicKey` associated with it.

```cadence
/// Account represents info about the account created on the blockchain.
///
pub struct Account {
    pub let address: Address
    pub let publicKey: PublicKey

    init(address: Address, publicKey: PublicKey) {
        self.address = address
        self.publicKey = publicKey
    }
}
```

### Executing scripts

Scripts can be run with the `executeScript` function, which returns a `ScriptResult`.
The function takes script-code as the first argument, and the script-arguments as an array as the second argument.

```cadence
let result = blockchain.executeScript("pub fun main(a: String) {}", ["hello"])
```

The script result consists of the `status` of the script execution, and a `returnValue` if the script execution was
successful, or an `error` otherwise (see [errors](#errors) section for more details on errors).

```cadence
/// The result of a script execution.
///
pub struct ScriptResult {
    pub let status: ResultStatus
    pub let returnValue: AnyStruct?
    pub let error: Error?

    init(status: ResultStatus, returnValue: AnyStruct?, error: Error?) {
        self.status = status
        self.returnValue = returnValue
        self.error = error
    }
}
```

### Executing transactions

A transaction must be created with the transaction code, a list of authorizes,
a list of signers that would sign the transaction, and the transaction arguments.

```cadence
/// Transaction that can be submitted and executed on the blockchain.
///
pub struct Transaction {
    pub let code: String
    pub let authorizers: [Address]
    pub let signers: [Account]
    pub let arguments: [AnyStruct]

    init(code: String, authorizers: [Address], signers: [Account], arguments: [AnyStruct]) {
        self.code = code
        self.authorizers = authorizers
        self.signers = signers
        self.arguments = arguments
    }
}
```

The number of authorizers must match the number of `AuthAccount` arguments in the `prepare` block of the transaction.

```cadence
let tx = Test.Transaction(
    code: "transaction { prepare(acct: AuthAccount) {} execute{} }",
    authorizers: [account.address],
    signers: [account],
    arguments: [],
)
```

There are two ways to execute the created transaction.
- Executing the transaction immediately
  ```cadence
  let result = blockchain.executeTransaction(tx)
  ```
  This may fail if the current block contains transactions that have not being executed yet.


- Adding the transaction to the current block, and executing it later.
  ```cadence
  // Add to the current block
  blockchain.addTransaction(tx)

  // Execute the next transaction in the block
  let result = blockchain.executeNextTransaction()
  ```

The result of a transaction consists of the status of the execution, and an `Error` if the transaction failed.

```cadence
/// The result of a transaction execution.
///
pub struct TransactionResult {
    pub let status: ResultStatus
    pub let error: Error?

    init(status: ResultStatus, error: Error) {
        self.status = status
        self.error = error
    }
 }
```

### Commit block

`commitBlock` block will commit the current block, and will fail if there are any un-executed transactions in the block.

```cadence
blockchain.commitBlock()
```

### Deploying contracts

A contract can be deployed using the `deployContract` function of the `Blockchain`.

```cadence
let contractCode = "pub contract Foo{ pub let msg: String;   init(_ msg: String){ self.msg = msg }   pub fun sayHello(): String { return self.msg } }"

let err = blockchain.deployContract(
    name: "Foo",
    code: contractCode,
    account: account,
    arguments: ["hello from args"],
)
```

An `Error` is returned if the contract deployment fails. Otherwise, a `nil` is returned.

### Configuring import addresses

A common pattern in Cadence projects is to define the imports as file locations and specify the addresses
corresponding to each network in the [Flow CLI configuration file](https://developers.flow.com/tools/flow-cli/configuration#contracts).
When writing tests for a such project, it may also require to specify the addresses to be used during the tests as well.
However, during tests, since accounts are created dynamically and the addresses are also generated dynamically,
specifying the addresses statically in a configuration file is not an option.

Hence, the test framework provides a way to specify the addresses using the
`useConfiguration(_ configs: Test.Configurations)` function in `Blockchain`.

The `Configurations` struct consists of a mapping of import locations to their addresses.

```cadence
/// Configurations to be used by the blockchain.
/// Can be used to set the address mapping.
///
pub struct Configurations {
    pub let addresses: {String: Address}

    init(addresses: {String: Address}) {
        self.addresses = addresses
    }
}
```

The configurations can be specified during the test setup as a best-practice.

```cadence
pub var blockchain = Test.newEmulatorBlockchain()
pub var accounts: [Test.Account] = []

pub fun setup() {
    // Create accounts in the blockchain.

    let acct1 = blockchain.createAccount()
    accounts.append(acct1)

    let acct2 = blockchain.createAccount()
    accounts.append(acct2)

    // Set the configurations with the addresses

    blockchain.useConfiguration(Test.Configurations({
        "./contracts/FooContract": acct1.address,
        "./contracts/BarContract": acct2.address
    }))
}
```

The subsequent operations on the blockchain (e.g: contract deployment, script/transaction execution) will resolve the
file import locations to the provided addresses.

### Errors

An `Error` maybe returned when an operation (such as executing a script, executing a transaction, etc.) is failed.
Contains a message indicating why the operation failed.

```cadence
// Error is returned if something has gone wrong.
//
pub struct Error {
    pub let message: String

    init(_ message: String) {
        self.message = message
    }
}
```

An `Error` may typically be handled by failing the test case or by panicking (which will result in failing the test).

```cadence
let err: Error? = ...

if let err = err {
    panic(err.message)
}
```

## Reading from files

Writing tests often require constructing source-code of contracts/transactions/scripts in the test script.
Testing framework provides a convenient way to load programs from a local file, without having to manually construct
them within the test script.

```cadence
let contractCode = Test.readFile("./sample/contracts/FooContract.cdc")
```

`readFile` returns the content of the file as a string.
