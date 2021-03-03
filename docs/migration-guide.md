---
title: Migration Guide
---

## v0.11

Version 0.11 [introduced breaking changes](https://github.com/onflow/cadence/releases/tag/v0.11.0):
Paths are now typed, i.e. there are specific subtypes for storage, public, and private paths,
and the Storage API has been made type-safer by changing parameter types to more specific path types.

Please read the release notes linked above to learn more.

The following hints should help with updating your Cadence code:

- The return types of `PublicAccount.getCapability` and `AuthAccount.getCapability` are not optional anymore.

  For example, in the following code the force unwrapping should be removed:

  ```diff
       let balanceRef = account
  -        .getCapability(/public/flowTokenBalance)!
  +        .getCapability(/public/flowTokenBalance)
           .borrow<&FlowToken.Vault{FungibleToken.Balance}>()!
  ```

  In the next example, optional binding was used and is not allowed anymore:

  ```diff
  -    if let balanceCap = account.getCapability(/public/flowTokenBalance) {
  -        return balanceCap.borrow<&FlowToken.Vault{FungibleToken.Balance}>()!
  -    }

  +    let balanceCap = account.getCapability(/public/flowTokenBalance)
  +    return balanceCap.borrow<&FlowToken.Vault{FungibleToken.Balance}>()!
  ```

- Parameters of the Storage API functions that had the type `Path` now have more specific types.
  For example, the `getCapability` functions now require a `CapabilityPath` instead of just a `Path`.

  Ensure path values with the correct path type are passed to these functions.

  For example, a contract may have declared a field with the type `Path`, then used it in a function to call `getCapability`.
  The type of the field must be changed to the more specific type:

    ```diff
     pub contract SomeContract {

    -    pub let somethingPath: Path
    +    pub let somethingPath: StoragePath

         init() {
             self.somethingPath = /storage/something
         }

         pub fun borrow(): &Something {
             return self.account.borrow<&Something>(self.somethingPath)
         }
     }
    ```
