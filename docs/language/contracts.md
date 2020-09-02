---
title: Contracts
---

A contract in Cadence is a collection of type definitions
of interfaces, structs, resources, data (its state), and code (its functions)
that lives in the contract storage area of an account in Flow.
Contracts are where all composite types like structs, resources,
events, and interfaces for these types in Cadence have to be defined.
Therefore, an object of one of these types cannot exist
without having been defined in a deployed Cadence contract.

Contracts can be created, updated, and deleted using the `setCode`
function of [accounts](accounts).
This functionality is covered in the [next section](#deploying-and-updating-contracts)

Contracts are types.
They are similar to composite types, but are stored differently than
structs or resources and cannot be used as values, copied, or moved
like resources or structs.

Contract stay in an account's contract storage
area and can only be updated or deleted by the account owner
with special commands.

Contracts are declared using the `contract` keyword. The keyword is followed
by the name of the contract.

```cadence
pub contract SomeContract {
    // ...
}
```

Contracts cannot be nested in each other.

```cadence
pub contract Invalid {

    // Invalid: Contracts cannot be nested in any other type.
    //
    pub contract Nested {
        // ...
    }
}
```

One of the simplest forms of a contract would just be one with a state field,
a function, and an `init` function that initializes the field:

```cadence
pub contract HelloWorld {

    // Declare a stored state field in HelloWorld
    //
    pub let greeting: String

    // Declare a function that can be called by anyone
    // who imports the contract
    //
    pub fun hello(): String {
        return self.greeting
    }

    init() {
        self.greeting = "Hello World!"
    }
}
```

This contract could be deployed to an account and live permanently
in the contract storage.  Transactions and other contracts
can interact with contracts by importing them at the beginning
of a transaction or contract definition.

Anyone could call the above contract's `hello` function by importing
the contract from the account it was deployed to and using the imported
object to call the hello function.

```cadence
import HelloWorld from 0x42

// Invalid: The contract does not know where hello comes from
//
log(hello())        // Error

// Valid: Using the imported contract object to call the hello
// function
//
log(HelloWorld.hello())    // prints "Hello World!"

// Valid: Using the imported contract object to read the greeting
// field.
log(HelloWorld.greeting)   // prints "Hello World!"

// Invalid: Cannot call the init function after the contract has been created.
//
HelloWorld.init()    // Error
```

There can be any number of contracts per account
and they can include an arbitrary amount of data.
This means that a contract can have any number of fields, functions, and type definitions,
but they have to be in the contract and not another top-level definition.

```cadence
// Invalid: Top-level declarations are restricted to only be contracts
//          or contract interfaces. Therefore, all of these would be invalid
//          if they were deployed to the account contract storage and
//          the deployment would be rejected.
//
pub resource Vault {}
pub struct Hat {}
pub fun helloWorld(): String {}
let num: Int
```

Another important feature of contracts is that instances of resources and events
that are declared in contracts can only be created/emitted within functions or types
that are declared in the same contract.

It is not possible create instances of resources and events outside the contract.

The contract below defines a resource interface `Receiver` and a resource `Vault`
that implements that interface.  The way this example is written,
there is no way to create this resource, so it would not be usable.

```cadence
// Valid
pub contract FungibleToken {

    pub resource interface Receiver {

        pub balance: Int

        pub fun deposit(from: @Receiver) {
            pre {
                from.balance > 0:
                    "Deposit balance needs to be positive!"
            }
            post {
                self.balance == before(self.balance) + before(from.balance):
                    "Incorrect amount removed"
            }
        }
    }

    pub resource Vault: Receiver {

        // keeps track of the total balance of the accounts tokens
        pub var balance: Int

        init(balance: Int) {
            self.balance = balance
        }

        // withdraw subtracts amount from the vaults balance and
        // returns a vault object with the subtracted balance
        pub fun withdraw(amount: Int): @Vault {
            self.balance = self.balance - amount
            return <-create Vault(balance: amount)
        }

        // deposit takes a vault object as a parameter and adds
        // its balance to the balance of the Account's vault, then
        // destroys the sent vault because its balance has been consumed
        pub fun deposit(from: @Receiver) {
            self.balance = self.balance + from.balance
            destroy from
        }
    }
}
```

If a user tried to run a transaction that created an instance of the `Vault` type,
the type checker would not allow it because only code in the `FungibleToken`
contract can create new `Vault`s.

```cadence
import FungibleToken from 0x42

// Invalid: Cannot create an instance of the `Vault` type outside
// of the contract that defines `Vault`
//
let newVault <- create FungibleToken.Vault(balance: 10)
```

The contract would have to either define a function that creates new
`Vault` instances or use its `init` function to create an instance and
store it in the owner's account storage.

This brings up another key feature of contracts in Cadence.  Contracts
can interact with its account's `storage` and `published` objects to store
resources, structs, and references.
They do so by using the special `self.account` object that is only accessible within the contract.

Imagine that these were declared in the above `FungibleToken` contract.

```cadence

    pub fun createVault(initialBalance: Int): @Vault {
        return <-create Vault(balance: initialBalance)
    }

    init(balance: Int) {
        let vault <- create Vault(balance: 1000)
        self.account.save(<-vault, to: /storage/initialVault)
    }
```

Now, any account could call the `createVault` function declared in the contract
to create a `Vault` object.
Or the owner could call the `withdraw` function on their own `Vault` to send new vaults to others.

```cadence
import FungibleToken from 0x42

// Valid: Create an instance of the `Vault` type by calling the contract's
// `createVault` function.
//
let newVault <- create FungibleToken.createVault(initialBalance: 10)
```

Contracts have the implicit field `let account: Account`,
which is the account in which the contract is deployed too.
This gives the contract the ability to e.g. read and write to the account's storage.

## Deploying and Updating Contracts

In order for a contract to be used in Cadence, it needs to be deployed to an account.
A contract can be deployed to an account using the `setCode` function of the `AuthAccount` type:

- `fun AuthAccount.setCode(_ code: [UInt8], ... contractInitializerArguments)`

  The `code` parameter is the byte representation of the source code.
  All additional arguments that are given are passed further to the initializer
  of the contract that is being deployed.

For example, assuming the following contract code should be deployed:

```cadence
pub contract Test {
    pub let message: String

    init(message: String) {
        self.message = message
    }
}
```

The contract can be deployed as follows:

```cadence
// Decode the hex-encoded source code into a byte array
// using the built-in function `decodeHex`.
//
// (The ellipsis ... indicates the remainder of the string)
//
let code = "70756220636f6e...".decodeHex()

// `code` has type `[UInt8]`

let signer: Account = ...
signer.setCode(
    code,
    message: "I'm a new contract in an existing account"
)
```

## Contract Interfaces

Like composite types, contracts can have interfaces that specify rules
about their behavior, their types, and the behavior of their types.

Contract interfaces have to be declared globally.  Declarations
cannot be nested in other types.

If a contract interface declares a concrete type, implementations of it
must also declare the same concrete type conforming to the type requirement.

If a contract interface declares an interface type, the implementing contract
does not have to also define that interface.  They can refer to that nested
interface by saying `{ContractInterfaceName}.{NestedInterfaceName}`

```cadence
// Declare a contract interface that declares an interface and a resource
// that needs to implement that interface in the contract implementation.
//
pub contract interface InterfaceExample {

    // Implementations do not need to declare this
    // They refer to it as InterfaceExample.NestedInterface
    //
    pub resource interface NestedInterface {}

    // Implementations must declare this type
    //
    pub resource Composite: NestedInterface {}
}

pub contract ExampleContract: InterfaceExample {

    // The contract doesn't need to redeclare the `NestedInterface` interface
    // because it is already declared in the contract interface

    // The resource has to refer to the resource interface using the name
    // of the contract interface to access it
    //
    pub resource Composite: InterfaceExample.NestedInterface {
    }
}
```
