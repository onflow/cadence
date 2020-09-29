---
title: Accounts
---

Every account can be accessed through two types:

- As a **Public Account** with the type `PublicAccount`,
  which represents the publicly available portion of an account.

  ```cadence
  struct PublicAccount {

      let address: Address

      // Storage operations

      fun getCapability<T>(_ path: Path): Capability<T>?
      fun getLinkTarget(_ path: Path): Path?
  }
  ```

  Any code can get the `PublicAccount` for an account address
  using the built-in `getAccount` function:

  ```cadence
  fun getAccount(_ address: Address): PublicAccount
  ```

- As an **Authorized Account** with type `AuthAccount`,
  which represents the authorized portion of an account.

  Access to an `AuthAccount` means having full access to its [storage](#account-storage),
  public keys, and code.

  Only [signed transactions](transactions) can get the `AuthAccount` for an account.
  For each script signer of the transaction, the corresponding `AuthAccount` is passed
  to the `prepare` phase of the transaction.

  ```cadence
  struct AuthAccount {

      let address: Address

      // Contracts

      let contracts: AuthAccount.Contracts

      // Key management

      fun addPublicKey(_ publicKey: [UInt8])
      fun removePublicKey(_ index: Int)

      // Storage operations

      fun save<T>(_ value: T, to: Path)
      fun load<T>(from: Path): T?
      fun copy<T: AnyStruct>(from: Path): T?

      fun borrow<T: &Any>(from: Path): T?

      fun link<T: &Any>(_ newCapabilityPath: Path, target: Path): Capability<T>?
      fun getLinkTarget(_ path: Path): Path?
      fun unlink(_ path: Path)

      struct Contracts {
          fun add(
              name: String,
              code: [UInt8],
              ... contractInitializerArguments
          ): DeployedContract

          fun update__experimental(name: String, code: [UInt8]): DeployedContract

          fun get(name: String): DeployedContract?

          fun remove(name: String): DeployedContract?
      }
  }

  struct DeployedCode {
      let name: String
      let code: [UInt8]
  }
  ```

## Account Creation

Accounts can be created by calling the `AuthAccount` constructor
and passing the account that should pay for the account creation for the `payer` parameter.

The `payer` must have enough funds to be able to create an account.
If the account does not have the required funds, the program aborts.

To authorize access to the account, keys can be added using the `addPublicKey` function.
Keys can also later be removed using the `removePublicKey` function.

For example, to create an account and have the signer of the transaction pay for the account creation,
and authorize one key to access the account:

```cadence
transaction(key: [UInt8]) {
    prepare(signer: AuthAccount) {
        let account = AuthAccount(payer: signer)
        account.addPublicKey(key)
    }
}
```

## Account Storage

All accounts have storage.

Objects are stored under paths in storage.
Paths consist of a domain and an identifier.

Paths start with the character `/`, followed by the domain, the path separator `/`,
and finally the identifier.
For example, the path `/storage/test` has the domain `storage` and the identifier `test`.

There are only three valid domains: `storage`, `private`, and `public`.

Objects in storage are always stored in the `storage` domain.

Both resources and structures can be stored in account storage.

Account storage is accessed through the following functions of `AuthAccount`.
This means that any code that has access to the authorized account has access
to all its stored objects.

- `cadence•fun save<T>(_ value: T, to: Path)`

  Saves an object to account storage.
  Resources are moved into storage, and structures are copied.

  `T` is the type parameter for the object type.
  It can be inferred from the argument's type.

  If there is already an object stored under the given path, the program aborts.

  The path must be a storage path, i.e., only the domain `storage` is allowed.

- `cadence•fun load<T>(from: Path): T?`

  Loads an object from account storage.
  If no object is stored under the given path, the function returns `nil`.
  If there is an object stored, the stored resource or structure is moved
  out of storage and returned as an optional.
  When the function returns, the storage no longer contains an object
  under the given path.

  `T` is the type parameter for the object type.
  A type argument for the parameter must be provided explicitly.

  The type `T` must be a supertype of the type of the loaded object.
  If it is not, the function returns `nil`.
  The given type does not necessarily need to be exactly the same as the type of the loaded object.

  The path must be a storage path, i.e., only the domain `storage` is allowed.

- `cadence•fun copy<T: AnyStruct>(from: Path): T?`

  Returns a copy of a structure stored in account storage, without removing it from storage.

  If no structure is stored under the given path, the function returns `nil`.
  If there is a structure stored, it is copied.
  The structure stays stored in storage after the function returns.

  `T` is the type parameter for the structure type.
  A type argument for the parameter must be provided explicitly.

  The type `T` must be a supertype of the type of the copied structure.
  If it is not, the function returns `nil`.
  The given type does not necessarily need to be exactly the same as
  the type of the copied structure.

  The path must be a storage path, i.e., only the domain `storage` is allowed.

```cadence
// Declare a resource named `Counter`.
//
resource Counter {
    pub var count: Int

    pub init(count: Int) {
        self.count = count
    }
}

// In this example an authorized account is available through the constant `authAccount`.

// Create a new instance of the resource type `Counter`
// and save it in the storage of the account.
//
// The path `/storage/counter` is used to refer to the stored value.
// Its identifier `counter` was chosen freely and could be something else.
//
authAccount.save(<-create Counter(count: 42), to: /storage/counter)

// Run-time error: Storage already contains an object under path `/storage/counter`
//
authAccount.save(<-create Counter(count: 123), to: /storage/counter)

// Load the `Counter` resource from storage path `/storage/counter`.
//
// The new constant `counter` has the type `Counter?`, i.e., it is an optional,
// and its value is the counter resource, that was saved at the beginning
// of the example.
//
let counter <- authAccount.load<@Counter>(from: /storage/counter)

// The storage is now empty, there is no longer an object stored
// under the path `/storage/counter`.

// Load the `Counter` resource again from storage path `/storage/counter`.
//
// The new constant `counter2` has the type `Counter?` and is `nil`,
// as nothing is stored under the path `/storage/counter` anymore,
// because the previous load moved the counter out of storage.
//
let counter2 <- authAccount.load<@Counter>(from: /storage/counter)

// Create another new instance of the resource type `Counter`
// and save it in the storage of the account.
//
// The path `/storage/otherCounter` is used to refer to the stored value.
//
authAccount.save(<-create Counter(count: 123), to: /storage/otherCounter)

// Load the `Vault` resource from storage path `/storage/otherCounter`.
//
// The new constant `vault` has the type `Vault?` and its value is `nil`,
// as there is a resource with type `Counter` stored under the path,
// which is not a subtype of the requested type `Vault`.
//
let vault <- authAccount.load<@Vault>(from: /storage/otherCounter)

// The storage still stores a `Counter` resource under the path `/storage/otherCounter`.

// Save the string "Hello, World" in storage
// under the path `/storage/helloWorldMessage`.

authAccount.save("Hello, world!", to: /storage/helloWorldMessage)

// Copy the stored message from storage.
//
// After the copy, the storage still stores the string under the path.
// Unlike `load`, `copy` does not remove the object from storage.
//
let message = authAccount.copy<String>(from: /storage/helloWorldMessage)

// Create a new instance of the resource type `Vault`
// and save it in the storage of the account.
//
authAccount.save(<-createEmptyVault(), to: /storage/vault)

// Invalid: Cannot copy a resource, as this would allow arbitrary duplication.
//
let vault <- authAccount.copy<@Vault>(from: /storage/vault)
```

As it is convenient to work with objects in storage
without having to move them out of storage,
as it is necessary for resources,
it is also possible to create references to objects in storage:
This is possible using the `borrow` function of an `AuthAccount`:

- `cadence•fun borrow<T: &Any>(from: Path): T?`

  Returns a reference to an object in storage without removing it from storage.
  If no object is stored under the given path, the function returns `nil`.
  If there is an object stored, a reference is returned as an optional.

  `T` is the type parameter for the object type.
  A type argument for the parameter must be provided explicitly.
  The type argument must be a reference to any type (`&Any`; `Any` is the supertype of all types).
  It must be possible to create the given reference type `T` for the stored /  borrowed object.
  If it is not, the function returns `nil`.
  The given type does not necessarily need to be exactly the same as the type of the borrowed object.

  The path must be a storage path, i.e., only the domain `storage` is allowed.

```cadence
// Declare a resource interface named `HasCount`, that has a field `count`
//
resource interface HasCount {
    count: Int
}

// Declare a resource named `Counter` that conforms to `HasCount`
//
resource Counter: HasCount {
    pub var count: Int

    pub init(count: Int) {
        self.count = count
    }
}

// In this example an authorized account is available through the constant `authAccount`.

// Create a new instance of the resource type `Counter`
// and save it in the storage of the account.
//
// The path `/storage/counter` is used to refer to the stored value.
// Its identifier `counter` was chosen freely and could be something else.
//
authAccount.save(<-create Counter(count: 42), to: /storage/counter)

// Create a reference to the object stored under path `/storage/counter`,
// typed as `&Counter`.
//
// `counterRef` has type `&Counter?` and is a valid reference, i.e. non-`nil`,
// because the borrow succeeded:
//
// There is an object stored under path `/storage/counter`
// and it has type `Counter`, so it can be borrowed as `&Counter`
//
let counterRef = authAccount.borrow<&Counter>(from: /storage/counter)

counterRef?.count // is `42`

// Create a reference to the object stored under path `/storage/counter`,
// typed as `&{HasCount}`.
//
// `hasCountRef` is non-`nil`, as there is an object stored under path `/storage/counter`,
// and the stored value of type `Counter` conforms to the requested type `{HasCount}`:
// the type `Counter` implements the restricted type's restriction `HasCount`

let hasCountRef = authAccount.borrow<&{HasCount}>(from: /storage/counter)

// Create a reference to the object stored under path `/storage/counter`,
// typed as `&{SomethingElse}`.
//
// `otherRef` is `nil`, as there is an object stored under path `/storage/counter`,
// but the stored value of type `Counter` does not conform to to the requested type `{Other}`:
// the type `Counter` does not implement the restricted type's restriction `Other`

let otherRef = authAccount.borrow<&{Other}>(from: /storage/counter)

// Create a reference to the object stored under path `/storage/nonExistent`,
// typed as `&{HasCount}`.
//
// `nonExistentRef` is `nil`, as there is nothing stored under path `/storage/nonExistent`
//
let nonExistentRef = authAccount.borrow<&{HasCount}>(from: /storage/nonExistent)
```
