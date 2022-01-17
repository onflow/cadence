---
title: Built-in Functions
---

- `cadence•fun panic(_ message: String): Never`

  Terminates the program unconditionally
  and reports a message which explains why the unrecoverable error occurred.

  ```cadence
  let optionalAccount: AuthAccount? = // ...
  let account = optionalAccount ?? panic("missing account")
  ```

- `cadence•fun assert(_ condition: Bool, message: String)`

  Terminates the program if the given condition is false,
  and reports a message which explains how the condition is false.
  Use this function for internal sanity checks.

  The message argument is optional.

- `cadence•fun unsafeRandom(): UInt64`

  Returns a pseudo-random number.

  NOTE: The use of this function is unsafe if not used correctly.

  Follow [best practices](https://github.com/ConsenSys/smart-contract-best-practices/blob/051ec2e42a66f4641d5216063430f177f018826e/docs/recommendations.md#remember-that-on-chain-data-is-public)
  to prevent security issues when using this function.
