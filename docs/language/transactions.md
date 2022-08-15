---
title: Transactions
---

Transactions are objects that are signed by one or more [accounts](accounts)
and are sent to the chain to interact with it.

Transactions are structured as such:

First, the transaction can import any number of types from external accounts
using the import syntax.

```cadence
import FungibleToken from 0x01
```

The body is declared using the `transaction` keyword and its contents
are contained in curly braces.

Next is the body of the transaction,
which first contains local variable declarations that are valid
throughout the whole of the transaction.

```cadence
transaction {
    // transaction contents
    let localVar: Int

    ...
}
```

Then, four optional main phases:
Preparation, preconditions, execution, and postconditions, in that order.
The preparation and execution phases are blocks of code that execute sequentially.

The following empty Cadence transaction contains no logic,
but demonstrates the syntax for each phase, in the order these phases will be executed:

```cadence
transaction {
    prepare(signer1: AuthAccount, signer2: AuthAccount) {
        // ...
    }

    pre {
        // ...
    }

    execute {
        // ...
    }

    post {
        // ...
    }
}
```

Although optional, each phase serves a specific purpose when executing a transaction
and it is recommended that developers use these phases when creating their transactions.
The following will detail the purpose of and how to use each phase.

## Transaction Parameters

Transactions may declare parameters.
Transaction parameters are declared like function parameters.
The arguments for the transaction are passed in the sent transaction.

Transaction parameters are accessible in all phases.

```cadence
// Declare a transaction which has one parameter named `amount`
// that has the type `UFix64`
//
transaction(amount: UFix64) {

}
```

## Prepare phase

The `prepare` phase is used when access to the private `AuthAccount` object
of **signing accounts** is required for your transaction.

Direct access to signing accounts is **only possible inside the `prepare` phase**.

For each signer of the transaction the signing account is passed as an argument to the `prepare` phase.
For example, if the transaction has three signers,
the prepare **must** have three parameters of type `AuthAccount`.

```cadence
 prepare(signer1: AuthAccount) {
      // ...
 }
```

As a best practice, only use the `prepare` phase to define and execute logic that requires access
to the `AuthAccount` objects of signing accounts,
and *move all other logic elsewhere*.
Modifications to accounts can have significant implications,
so keep this phase clear of unrelated logic to ensure users of your contract are able to easily read
and understand logic related to their private account objects.

The prepare phase serves a similar purpose as the initializer of a contract/resource/structure.

For example, if a transaction performs a token transfer, put the withdrawal in the `prepare` phase,
as it requires access to the account storage, but perform the deposit in the `execute` phase.

`AuthAccount` objects have the permissions
to read from and write to the `/storage/` and `/private/` areas
of the account, which cannot be directly accessed anywhere else.
They also have the permission to create and delete capabilities that
use these areas.

## Pre Phase

The `pre` phase is executed after the `prepare` phase, and is used for checking
if explicit conditions hold before executing the remainder of the transaction.
A common example would be checking requisite balances before transferring tokens between accounts.

```cadence
pre {
    sendingAccount.balance > 0
}
```

If the `pre` phase throws an error, or does not return `true` the remainder of the transaction
is not executed and it will be completely reverted.

## Execute Phase

The `execute` phase does exactly what it says, it executes the main logic of the transaction.
This phase is optional, but it is a best practice to add your main transaction logic in the section,
so it is explicit.

```cadence
execute {
    // Invalid: Cannot access the authorized account object,
    // as `account1` is not in scope
    let resource <- account1.load<@Resource>(from: /storage/resource)
    destroy resource

    // Valid: Can access any account's public Account object
    let publicAccount = getAccount(0x03)
}
```

You **may not** access private `AuthAccount` objects in the `execute` phase,
but you may get an account's `PublicAccount` object,
which allows reading and calling methods on objects
that an account has published in the public domain of its account (resources, contract methods, etc.).

## Post Phase

Statements inside of the `post` phase are used
to verify that your transaction logic has been executed properly.
It contains zero or more condition checks.

For example, a transfer transaction might ensure that the final balance has a certain value,
or e.g. it was incremented by a specific amount.

```cadence
post {
    result.balance == 30: "Balance after transaction is incorrect!"
}
```

If any of the condition checks result in `false`, the transaction will fail and be completely reverted.

Only condition checks are allowed in this section.
No actual computation or modification of values is allowed.

**A Note about `pre` and `post` Phases**

Another function of the `pre` and `post` phases is to help provide information
about how the effects of a transaction on the accounts and resources involved.
This is essential because users may want to verify what a transaction does before submitting it.
`pre` and `post` phases provide a way to introspect transactions before they are executed.

For example, in the future the phases could be analyzed and interpreted to the user
in the software they are using,
e.g. "this transaction will transfer 30 tokens from A to B.
The balance of A will decrease by 30 tokens and the balance of B will increase by 30 tokens."

## Summary

Cadence transactions use phases to make the transaction's code / intent more readable
and to provide a way for developer to separate potentially 'unsafe' account
modifying code from regular transaction logic,
as well as provide a way to check for error prior / after transaction execution,
and abort the transaction if any are found.

The following is a brief summary of how to use the `prepare`, `pre`, `execute`,
and `post` phases in a Cadence transaction.

```cadence
transaction {
    prepare(signer1: AuthAccount) {
        // Access signing accounts for this transaction.
        //
        // Avoid logic that does not need access to signing accounts.
        //
        // Signing accounts can't be accessed anywhere else in the transaction.
    }

    pre {
        // Define conditions that must be true
        // for this transaction to execute.
    }

    execute {
        // The main transaction logic goes here, but you can access
        // any public information or resources published by any account.
    }

    post {
        // Define the expected state of things
        // as they should be after the transaction executed.
        //
        // Also used to provide information about what changes
        // this transaction will make to accounts in this transaction.
    }
}
```
