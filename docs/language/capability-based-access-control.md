---
title: Capability-based Access Control
---

Users will often want to make it so that specific other users or even anyone else
can access certain fields and functions of a stored object.
This can be done by creating a capability.

As was mentioned before, access to stored objects is governed by the
tenets of [Capability Security](https://en.wikipedia.org/wiki/Capability-based_security).
This means that if an account wants to be able to access another account's
stored objects, it must have a valid capability to that object.

Capabilities are identified by a path and link to a target path, not directly to an object.
Capabilities are either public (any user can get access),
or private (access to/from the authorized user is necessary).

Public capabilities are created using public paths, i.e. they have the domain `public`.
After creation they can be obtained from both authorized accounts (`AuthAccount`)
and public accounts (`PublicAccount`).

Private capabilities are created using private paths, i.e. they have the domain `private`.
After creation they can be obtained from authorized accounts (`AuthAccount`),
but not from public accounts (`PublicAccount`).

Once a capability is created and obtained, it can be borrowed to get a reference
to the stored object.
When a capability is created, a type is specified that determines as what type
the capability can be borrowed.
This allows exposing and hiding certain functionality of a stored object.

Capabilities are created using the `link` function of an authorized account (`AuthAccount`):

- `fun link<T: &Any>(_ newCapabilityPath: CapabilityPath, target: Path): Capability<T>?`

  `newCapabilityPath` is the public or private path identifying the new capability.

  `target` is any public, private, or storage path that leads to the object
  that will provide the functionality defined by this capability.

  `T` is the type parameter for the capability type.
  A type argument for the parameter must be provided explicitly.

  The type parameter defines how the capability can be borrowed,
  i.e., how the stored value can be accessed.

  The link function returns `nil` if a link for the given capability path already exists,
  or the newly created capability if not.

  It is not necessary for the target path to lead to a valid object;
  the target path could be empty, or could lead to an object
  which does not provide the necessary type interface:

  The link function does **not** check if the target path is valid/exists at the time
  the capability is created and does **not** check if the target value conforms to the given type.

  The link is latent.
  The target value might be stored after the link is created,
  and the target value might be moved out after the link has been created.

Capabilities can be removed using the `unlink` function of an authorized account (`AuthAccount`):

- `fun unlink(_ path: CapabilityPath)`

  `path` is the public or private path identifying the capability that should be removed.

To get the target path for a capability, the `getLinkTarget` function
of an authorized account (`AuthAccount`) or public account (`PublicAccount`) can be used:

- `fun getLinkTarget(_ path: CapabilityPath): Path?`

  `path` is the public or private path identifying the capability.
  The function returns the link target path,
  if a capability exists at the given path,
  or `nil` if it does not.

Existing capabilities can be obtained by using the `getCapability` function
of authorized accounts (`AuthAccount`) and public accounts (`PublicAccount`):

- `fun getCapability<T>(_ at: CapabilityPath): Capability<T>`

  For public accounts, the function returns a capability
  if the given path is public.
  It is not possible to obtain private capabilities from public accounts.
  If the path is private or a storage path, the function returns `nil`.

  For authorized accounts, the function returns a capability
  if the given path is public or private.
  If the path is a storage path, the function returns `nil`.

  `T` is the type parameter that specifies how the capability can be borrowed.
  The type argument is optional, i.e. it need not be provided.

The `getCapability` function does **not** check if the target exists.
The link is latent.
The `check` function of the capability can be used to check if the target currently exists and could be borrowed,

- `fun check<T: &Any>(): Bool`

  `T` is the type parameter for the reference type.
  A type argument for the parameter must be provided explicitly.

  The function returns true if the capability currently targets an object
  that satisfies the given type, i.e. could be borrowed using the given type.

Finally, the capability can be borrowed to get a reference to the stored object.
This can be done using the `borrow` function of the capability:

- `fun borrow<T: &Any>(): T?`

  The function returns a reference to the object targeted by the capability,
  provided it can be borrowed using the given type.

  `T` is the type parameter for the reference type.
  If the function is called on a typed capability, the capability's type is used when borrowing.
  If the capability is untyped, a type argument must be provided explicitly in the call to `borrow`.

  The function returns `nil` when the targeted path is empty, i.e. nothing is stored under it.
  When the requested type exceeds what is allowed by the capability (or any interim capabilities),
  execution will abort with an error.

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

    pub fun increment(by amount: Int) {
        self.count = self.count + amount
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

// Create a public capability that allows access to the stored counter object
// as the type `{HasCount}`, i.e. only the functionality of reading the field
//
authAccount.link<&{HasCount}>(/public/hasCount, target: /storage/counter)
```

To get the published portion of an account, the `getAccount` function can be used.

Imagine that the next example is from a different account as before.

```cadence

// Get the public account for the address that stores the counter
//
let publicAccount = getAccount(0x1)

// Get a capability for the counter that is made publicly accessible
// through the path `/public/hasCount`.
//
// Use the type `&{HasCount}`, a reference to some object that provides the functionality
// of interface `HasCount`. This is the type that the capability can be borrowed as
// (it was specified in the call to `link` above).
// See the example below for borrowing using the type `&Counter`.
//
// After the call, the declared constant `countCap` has type `Capability<&{HasCount}>`,
// a capability that results in a reference that has type `&{HasCount}` when borrowed.
//
let countCap = publicAccount.getCapability<&{HasCount}>(/public/hasCount)

// Borrow the capability to get a reference to the stored counter.
//
// This borrow succeeds, i.e. the result is not `nil`,
// it is a valid reference, because:
//
// 1. Dereferencing the path chain results in a stored object
//    (`/public/hasCount` links to `/storage/counter`,
//    and there is an object stored under `/storage/counter`)
//
// 2. The stored value is a subtype of the requested type `{HasCount}`
//    (the stored object has type `Counter` which conforms to interface `HasCount`)
//
let countRef = countCap.borrow()!

countRef.count  // is `42`

// Invalid: The `increment` function is not accessible for the reference,
// because it has the type `&{HasCount}`, which does not expose an `increment` function,
// only a `count` field
//
countRef.increment(by: 5)

// Again, attempt to get a get a capability for the counter, but use the type `&Counter`.
//
// Getting the capability succeeds, because it is latent, but borrowing fails
// (the result s `nil`), because the capability was created/linked using the type `&{HasCount}`:
//
// The resource type `Counter` implements the resource interface `HasCount`,
// so `Counter` is a subtype of `{HasCount}`, but the capability only allows
// borrowing using unauthorized references of `{HasCount}` (`&{HasCount}`)
// instead of authorized references (`auth &{HasCount}`),
// so users of the capability are not allowed to borrow using subtypes,
// and they can't escalate the type by casting the reference either.
//
// This shows how parts of the functionality of stored objects
// can be safely exposed to other code
//
let countCapNew = publicAccount.getCapability<&Counter>(/public/hasCount)
let counterRefNew = countCapNew.borrow()

// `counterRefNew` is `nil`, the borrow failed

// Invalid: Cannot access the counter object in storage directly,
// the `borrow` function is not available for public accounts
//
let counterRef2 = publicAccount.borrow<&Counter>(from: /storage/counter)
```

The address of a capability can be obtained from the `address` field of the capability:

- `let address: Address`

  The address of the capability.
