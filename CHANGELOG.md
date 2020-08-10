
# v0.8.0

This release focuses on improvements, bug fixes, and bringing the documentation up-to-date.

## 🛠 Improvements

- Improved support on Windows: Treat carriage returns as space (#304)
- Improved JSON marshalling for more AST elements (#292, #286)
- Recursively check if assignment target expression is valid, don't copy returned values (#288)
- Consider existing prefix when suggesting completion items (#285)
- Include available declarations in "not declared" error (#280)
- Enforce the requirement for types to be storable in more places:
  - Transaction/script parameter types (#305)
  - Script return type (#291)
  - Arguments passed to the account `load` and `store` functions (#251)

  Previously, using non-storable failed at run-time, now the programs are rejected statically.

## ⭐ Features

- Add a version constant (#289)
- Language Server: Include error notes as related information in diagnostic (#276)

## 🐞 Bug Fixes

- Don't return an error when a location type is not supported (#290)
- Handle incomplete types in checker (#284)
- Fix the suggestion in the error when an interface is used as a type (#281)

## 📖 Documentation

- Brought the documentation up-to-date, include all new language features and API changes (#275, #277, #279)

## 💥 Breaking Changes

- Arguments passed to `cadence.Value#WithType` are now typed as pointers (`*cadence.Type`) rather than values (`cadence.Type`)

  Example:

  ```go
  myEventType := cadence.EventType{...})

  // this 👇 
  myEvent := cadence.NewEvent(...).WithType(myEventType)

  // changes to this 👇 
  myEvent := cadence.NewEvent(...).WithType(&myEventType)
  ```

# v0.7.0

This release contains a lot of improvements to the language server, which improve the development experience in the Visual Studio Code extension, and will also soon be integrated into the Flow Playground.

## ⭐ Features

- Added member completion to the language server (#257)
  - Insert the argument list when completing functions (#268)
  - Offer keyword snippets (#268)
- Enabled running the language server in the browser by compiling it to WebAssembly (#242)
- Added support for docstrings to the parser (#246, #255)
- Added support for pragmas to the parser (#239)
- Started a source compatibility suite (#226, #270).

  Cadence will be regularly checked for compatibility with existing code to ensure it does not regress in source compatibility and performance. As a start, the [Fungible Token standard](https://github.com/onflow/flow-ft) and [Non-Fungible Token standard](https://github.com/onflow/flow-ft) repositories are checked.

  Consider requesting your repository to be included in the compatibility suite!
- Added a simple minifier (#252)

## 🛠 Improvements

- Made embedding the language server easier (#274, #262)
- Extended the JSON serialization for Cadence values and types (#260, #264, #265, #266, #271, #273)
- Improved conformance checking (#245). Made return types covariant, i.e. allow subtypes for return types of functions to satisfy interface conformance.

## 💥 Breaking Changes

- Made strings immutable (#269)
- Moved the Visual Studio Code extension to a new repository: <https://github.com/onflow/vscode-flow> (#244)
- Added a magic prefix to stored data (#236).
  This allows versioning the data. Existing data without a prefix is migrated automatically
- Changed the order in which fields in values and types are exported to declaration order (#272)

## 🐞 Bug Fixes

- Removed the service account from the list of usable accounts in the language server (#261)
- Fixed a parsing ambiguity for return types (#267)
- Fixed non-composite reference subtyping (#253)

## 🧰 Maintenance

- Removed the old parser (#249)

# v0.6.0

This is a small release with some bug fixes, internal improvements, and one breaking change for code that embeds Cadence.

## 💥 Breaking Changes

- Removed the unused `controller` parameter from the storage interface methods:

  - `func GetValue(owner, controller, key []byte) (value []byte, err error)`
    → `func GetValue(owner, key []byte) (value []byte, err error)`

  - `SetValue(owner, controller, key, value []byte) (err error)`
    → `SetValue(owner, key, value []byte) (err error)`

  - `ValueExists(owner, controller, key []byte) (exists bool, err error)`
    → `ValueExists(owner, key []byte) (exists bool, err error)`

  This is only a breaking change in the implementation of Cadence, i.e for code that embeds Cadence, not in the Cadence language, i.e. for users of Cadence.

## ⭐ Features

- Added an optional callback for writes with high-level values to the interface:

  ```go
  type HighLevelStorage interface {

	  // HighLevelStorageEnabled should return true
	  // if the functions of HighLevelStorage should be called,
	  // e.g. SetCadenceValue
	  HighLevelStorageEnabled() bool

	  // SetCadenceValue sets a value for the given key in the storage, owned by the given account.
	  SetCadenceValue(owner Address, key string, value cadence.Value) (err error)
  }
  ```

  This is a feature in the implementation of Cadence, i.e for code that embeds Cadence, not in the Cadence language, i.e. for users of Cadence.

## 🛠 Improvements

- Don't report an error for a restriction with an invalid type
- Record the occurrences of types. This enables the "Go to definition" feature for types in the language server
- Parse member expressions without a name. This allows the checker to run. This will enable adding code completion support or members in the future.

## 🐞 Bug Fixes

- Fixed a crash when checking the invocation of a function on an undeclared variable
- Fixed handling of functions in composite values in the JSON-CDC encoding
- Fixed a potential stack overflow when checking member storability

# v0.5.0

## ⭐ Features and Improvements

### Crypto

A new built-in contract `Crypto` was added for performing cryptographic operations.

The contract can be imported using `import Crypto`.

This first iteration provides support for validating signatures, with an API that supports multiple signatures and weighted keys.

For example, to verify two signatures with equal weights for some signed data:

```swift
let keyList = Crypto.KeyList()

let publicKeyA = Crypto.PublicKey(
    publicKey:
        "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
    signatureAlgorithm: Crypto.ECDSA_P256
)
keyList.add(
    publicKeyA,
    hashAlgorithm: Crypto.SHA3_256,
    weight: 0.5
)

let publicKeyB = Crypto.PublicKey(
    publicKey:
        "df9609ee588dd4a6f7789df8d56f03f545d4516f0c99b200d73b9a3afafc14de5d21a4fc7a2a2015719dc95c9e756cfa44f2a445151aaf42479e7120d83df956".decodeHex(),
    signatureAlgorithm: Crypto.ECDSA_P256
)
keyList.add(
    publicKeyB,
    hashAlgorithm: Crypto.SHA3_256,
    weight: 0.5
)

let signatureSet = [
    Crypto.KeyListSignature(
        keyIndex: 0,
        signature:
            "8870a8cbe6f44932ba59e0d15a706214cc4ad2538deb12c0cf718d86f32c47765462a92ce2da15d4a29eb4e2b6fa05d08c7db5d5b2a2cd8c2cb98ded73da31f6".decodeHex()
    ),
    Crypto.KeyListSignature(
        keyIndex: 1,
        signature:
            "bbdc5591c3f937a730d4f6c0a6fde61a0a6ceaa531ccb367c3559335ab9734f4f2b9da8adbe371f1f7da913b5a3fdd96a871e04f078928ca89a83d841c72fadf".decodeHex()
    )
]

// "foo", encoded as UTF-8, in hex representation
let signedData = "666f6f".decodeHex()

let isValid = keyList.isValid(
    signatureSet: signatureSet,
    signedData: signedData
)
```

### Run-time Types

The type `Type` was added to represent types at run-time.

To create a type value, use the constructor function `Type<T>()`, which accepts the static type as a type argument.

This is similar to e.g. `T.self` in Swift, `T::class` in Kotlin, and `T.class` in Java.

For example, to represent the type `Int` at run-time:

```swift
let intType: Type = Type<Int>()
```

This works for both built-in and user-defined types. For example, to get the type value for a resource:

```swift
resource Collectible {}

let collectibleType = Type<@Collectible>()
```

The function `fun isInstance(_ type: Type): Bool` can be used to check if a value has a certain type:

```swift
let collectible <- create Collectible()
let collectibleType = Type<@Collectible>()
let result = collectible.isInstance(collectibleType)
```

For example, this allows implementing a marketplace sale resource:

```swift
pub resource SimpleSale {

    pub var objectForSale: @AnyResource?
    pub let priceForObject: UFix64
    pub let requiredCurrency: Type
    pub let paymentReceiver: Capability<&{FungibleToken.Receiver}>

    init(
        objectForSale: @AnyResource,
        priceForObject: UFix64,
        requiredCurrency: Type,
        paymentReceiver: Capability<&{FungibleToken.Receiver}>
    ) {
        self.objectForSale <- objectForSale
        self.priceForObject = priceForObject
        self.requiredCurrency = requiredCurrency
        self.paymentReceiver = paymentReceiver
    }

    destroy() {
        destroy self.objectForSale
    }

    pub fun buyObject(purchaseAmount: @FungibleToken.Vault): @AnyResource {
        pre {
            self.objectForSale != nil
            purchaseAmount.balance >= self.priceForObject
            purchaseAmount.isInstance(self.requiredCurrency)
        }

        let receiver = self.paymentReceiver.borrow()
            ?? panic("failed to borrow payment receiver capability")

        receiver.deposit(from: <-purchaseAmount)
        let objectForSale <- self.objectForSale <- nil
        return <-objectForSale
    }
}
```

### New Parser

The existing parser was replaced by a new implementation, completely written from scratch, producing the same result as the old parser.

It is significantly faster than the old parser. For example, the following benchmark shows the difference for parsing all fungible and non-fungible token contracts and example transactions:

```sh
$ go run ./cmd/parse -bench ../../flow-{n,}ft/{contracts,transactions}/*
flow-nft/contracts/ExampleNFT.cdc:          [old]        9  111116110 ns/op
flow-nft/contracts/ExampleNFT.cdc:          [new]     2712     463483 ns/op
flow-nft/contracts/NonFungibleToken.cdc:    [old]      393    3097172 ns/op
flow-nft/contracts/NonFungibleToken.cdc:    [new]     3489     348496 ns/op
flow-nft/transactions/mint_nft.cdc:         [old]      700    1574730 ns/op
flow-nft/transactions/mint_nft.cdc:         [new]    12770      94070 ns/op
flow-nft/transactions/read_nft_data.cdc:    [old]      994    1222887 ns/op
flow-nft/transactions/read_nft_data.cdc:    [new]    15242      79295 ns/op
flow-nft/transactions/setup_account.cdc:    [old]     1281     879751 ns/op
flow-nft/transactions/setup_account.cdc:    [new]    16675      71759 ns/op
flow-nft/transactions/transfer_nft.cdc:     [old]       72   16417568 ns/op
flow-nft/transactions/transfer_nft.cdc:     [new]    10000     109380 ns/op
flow-ft/contracts/CustodialDeposit.cdc:     [old]       18   64938763 ns/op
flow-ft/contracts/CustodialDeposit.cdc:     [new]     3482     354662 ns/op
flow-ft/contracts/FlowToken.cdc:            [old]        7  177111544 ns/op
flow-ft/contracts/FlowToken.cdc:            [new]     1920     640557 ns/op
flow-ft/contracts/FungibleToken.cdc:        [old]      232    5324962 ns/op
flow-ft/contracts/FungibleToken.cdc:        [new]     2947     419529 ns/op
flow-ft/contracts/TokenForwarding.cdc:      [old]       44   25136749 ns/op
flow-ft/contracts/TokenForwarding.cdc:      [new]     7183     172917 ns/op
flow-ft/transactions/burn_tokens.cdc:       [old]       37   31475393 ns/op
flow-ft/transactions/burn_tokens.cdc:       [new]    11361     105932 ns/op
flow-ft/transactions/create_forwarder.cdc:  [old]      733    1636347 ns/op
flow-ft/transactions/create_forwarder.cdc:  [new]     8127     147520 ns/op
flow-ft/transactions/create_minter.cdc:     [old]     1306     923201 ns/op
flow-ft/transactions/create_minter.cdc:     [new]    15240      77666 ns/op
flow-ft/transactions/custodial_deposit.cdc: [old]       69   16504795 ns/op
flow-ft/transactions/custodial_deposit.cdc: [new]     7940     144228 ns/op
flow-ft/transactions/get_balance.cdc:       [old]     1094    1111272 ns/op
flow-ft/transactions/get_balance.cdc:       [new]    18741      65745 ns/op
flow-ft/transactions/get_supply.cdc:        [old]     1392     740989 ns/op
flow-ft/transactions/get_supply.cdc:        [new]    46008      26719 ns/op
flow-ft/transactions/mint_tokens.cdc:       [old]       72   17435128 ns/op
flow-ft/transactions/mint_tokens.cdc:       [new]     8841     124117 ns/op
flow-ft/transactions/setup_account.cdc:     [old]     1219    1089357 ns/op
flow-ft/transactions/setup_account.cdc:     [new]    13797      84948 ns/op
flow-ft/transactions/transfer_tokens.cdc:   [old]       74   17011751 ns/op
flow-ft/transactions/transfer_tokens.cdc:   [new]     9578     125829 ns/op
```

The new parser also provides better error messages and will allow us to provide even better error messages in the future.

The new parser is enabled by default – if you discover any problems or regressions, please report them!

### Typed Capabilities

Capabilities now accept an optional type argument, the reference type the capability should be borrowed as.

If a type argument is provided, it will be used for `borrow` and `check` calls, so there is no need to provide a type argument for the calls anymore.

The function `getCapability` now also accepts an optional type argument, and returns a typed capability if one is provided.

For example, the following two functions have the same behaviour:

```swift
fun cap1(): &Something? {
  // The type annotation for `cap` is only added for demonstration purposes,
  // it can also be omitted, because it can be inferred from the value
  let cap: Capability = getAccount(0x1).getCapability(/public/something)!
  return cap.borrow<&Something>()
}

fun cap2(): &Something? {
  // The type annotation for `cap` is only added for demonstration purposes,
  // it can also be omitted, because it can be inferred from the value
  let cap: Capability<&Something> =
      getAccount(0x1).getCapability<&Something>(/public/something)!
  return cap.borrow()
}
```

Prefer a typed capability over a non-typed one to reduce uses / borrows to a simple `.borrow()`, instead of having to repeatedly mention the borrowed type.

### Parameters for Scripts

Just like transactions, the `main` functions of scripts can now have parameters.

For example, a script that can be passed an integer and a string, and which logs both values, can be written as:

```swift
pub fun main(x: Int, y: String) {
    log(x)
    log(y)
}
```

### Standard Library

The function `fun toString(): String` was added to all number types and addresses. It returns the textual representation of the type. A future version of Cadence will add this function to more types.

The function `fun toBytes(): [UInt8]` was added to `Address`. It returns the byte representation of the address.

The function `fun toBigEndianBytes(): [UInt8]` was added to all number types. It returns the big-endian byte representation of the number value.

## 🐞 Bug Fixes

- Fixed the checking of return statements:
  A missing return value is now properly reported
- Fixed the checking of function invocations:
  Resources are temporarily moved into the invoked function
- Disabled caching of top-level programs (transactions, scripts)
- Fixed the comparison of optional values
- Fixed parsing of unpadded fixed point numbers
  in the JSON encoding of Cadence values (JSON-CDC)

## 💥 Breaking Changes

### Field Types

Fields which have a non-storable type are now rejected:

Non-storable types are:

- Functions (e.g. `((Int): Bool)`)
- Accounts (`AuthAccount` / `PublicAccount`)
- Transactions
- `Void`

A future version will also make references non-storable. Instead of storing a reference, store a capability and borrow it to acquire a reference.

### Byte Arrays

Cadence now represents all byte arrays as `[UInt8]` rather than `[Int]`. This affects the following functions:

- `String.decodeHex(): [Int]` => `String.decodeHex(): [UInt8]`
- `AuthAccount.addPublicKey(publicKey: [Int])` => `AuthAccount.addPublicKey(publicKey: [UInt8])`
- `AuthAccount.setCode(code: [Int])` => `AuthAccount.setCode(code: [UInt8])`

However, array literals such as the following will still be inferred as `[Int]`, meaning that they can't be used directly:

```swift
myAccount.addPublicKey([1, 2, 3])
```

Consider using `String.decodeHex()` for now until type inference has been improved.


# v0.4.0

## 💥 Breaking Changes

- The `AuthAccount` constructor now requires a `payer` parameter, from which the account creation fee is deducted. The `publicKeys` and `code` parameters were removed.

## ⭐ Features

- Added the function `unsafeNotInitializingSetCode` to `AuthAccount` which, like `setCode`, updates the account's code but does not create a new contract object or invoke its initializer. This function can be used to upgrade account code without overwriting the existing stored contract object.

  ⚠️ This is potentially unsafe because no data migration is performed. For example, changing a field's type results in undefined behaviour (i.e. could lead to the execution being aborted).

## 🐞 Bug Fixes

- Fixed variable shadowing for-in identifier to shadow outer scope (#72)
- Fixed order of declaring conformances and usages
- Fixed detection of resource losses in invocation of optional chaining result

# v0.3.0

## 💥 Breaking Changes

- Dictionary values which are resources are now stored in separate storage keys. This is transparent to programs, so is not breaking on a program level, but on the integration level.
- The state encoding was switched from Gob to CBOR. This is transparent to programs, so is not breaking on a program level, but on the integration level.
- The size of account addresses was reduced from 160 bits to 64 bits

## ⭐ Features

- Added bitwise operations
- Added support for caching of program ASTs
- Added metrics
- Block info can be queried now

## 🛠 Improvements

- Unnecessary writes are now avoided by tracking modifications
- Improved access modifier checks
- Improved the REPL
- Decode and validate transaction arguments
- Work on a new parser has started and is almost complete

## 🐞 Bug Fixes

- Fixed account code updated
- Fixed the destruction of arrays
- Fixed type requirements
- Implemented modulo for `[U]Fix64`, fix modulo with negative operands
- Implemented division for `[U]Fix64`
- Fixed the `flow.AccountCodeUpdated` event
- Fixed number conversion
- Fixed function return checking
