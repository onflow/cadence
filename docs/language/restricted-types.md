---
title: Restricted Types
---

Structure and resource types can be **restricted**. Restrictions are interfaces.
Restricted types only allow access to a subset of the members and functions
of the type that is restricted, indicated by the restrictions.

The syntax of a restricted type is `T{U1, U2, ... Un}`,
where `T` is the restricted type, a concrete resource or structure type,
and the types `U1` to `Un` are the restrictions, interfaces that `T` conforms to.

Only the members and functions of the union of the set of restrictions are available.

Restricted types are useful for increasing the safety in functions
that are supposed to only work on a subset of the type.
For example, by using a restricted type for a parameter's type,
the function may only access the functionality of the restriction:
If the function accidentally attempts to access other functionality,
this is prevented by the static checker.

```cadence
// Declare a resource interface named `HasCount`,
// which has a read-only `count` field
//
resource interface HasCount {
    pub let count: Int
}

// Declare a resource named `Counter`, which has a writeable `count` field,
// and conforms to the resource interface `HasCount`
//
pub resource Counter: HasCount {
    pub var count: Int

    init(count: Int) {
        self.count = count
    }

    pub fun increment() {
        self.count = self.count + 1
    }
}

// Create an instance of the resource `Counter`
let counter: @Counter <- create Counter(count: 42)

counter.count  // is `42`

counter.increment()

counter.count  // is `43`

// Move the resource in variable `counter` to a new variable `restrictedCounter`,
// but typed with the restricted type `Counter{HasCount}`:
// The variable may hold any `Counter`, but only the functionality
// defined in the given restriction, the interface `HasCount`, may be accessed
//
let restrictedCounter: @Counter{HasCount} <- counter

// Invalid: Only functionality of restriction `Count` is available,
// i.e. the read-only field `count`, but not the function `increment` of `Counter`
//
restrictedCounter.increment()

// Move the resource in variable `restrictedCounter` to a new variable `unrestrictedCounter`,
// again typed as `Counter`, i.e. all functionality of the counter is available
//
let unrestrictedCounter: @Counter <- restrictedCounter

// Valid: The variable `unrestrictedCounter` has type `Counter`,
// so all its functionality is available, including the function `increment`
//
unrestrictedCounter.increment()

// Declare another resource type named `Strings`
// which implements the resource interface `HasCount`
//
pub resource Strings: HasCount {
    pub var count: Int
    access(self) var strings: [String]

    init() {
        self.count = 0
        self.strings = []
    }

    pub fun append(_ string: String) {
        self.strings.append(string)
        self.count = self.count + 1
    }
}

// Invalid: The resource type `Strings` is not compatible
// with the restricted type `Counter{HasCount}`.
// Even though the resource `Strings` implements the resource interface `HasCount`,
// it is not compatible with `Counter`
//
let counter2: @Counter{HasCount} <- create Strings()
```

In addition to restricting concrete types is also possible
to restrict the built-in types `AnyStruct`, the supertype of all structures,
and `AnyResource`, the supertype of all resources.
For example, restricted type `AnyResource{HasCount}` is any resource type
for which only the functionality of the `HasCount` resource interface can be used.

The restricted types `AnyStruct` and `AnyResource` can be omitted.
For example, the type `{HasCount}` is any resource that implements
the resource interface `HasCount`.

```cadence
pub struct interface HasID {
    pub let id: String
}

pub struct A: HasID {
    pub let id: String

    init(id: String) {
        self.id = id
    }
}

pub struct B: HasID {
    pub let id: String

    init(id: String) {
        self.id = id
    }
}

// Create two instances, one of type `A`, and one of type `B`.
// Both types conform to interface `HasID`, so the structs can be assigned
// to variables with type `AnyResource{HasID}`: Some resource type which only allows
// access to the functionality of resource interface `HasID`

let hasID1: {HasID} = A(id: "1")
let hasID2: {HasID} = B(id: "2")

// Declare a function named `getID` which has one parameter with type `{HasID}`.
// The type `{HasID}` is a short-hand for `AnyStruct{HasID}`:
// Some structure which only allows access to the functionality of interface `HasID`.
//
pub fun getID(_ value: {HasID}): String {
    return value.id
}

let id1 = getID(hasID1)
// `id1` is "1"

let id2 = getID(hasID2)
// `id2` is "2"
```

Only concrete types may be restricted, e.g., the restricted type may not be an array,
the type `[T]{U}` is invalid.

Restricted types are also useful when giving access to resources and structures
to potentially untrusted third-party programs through [references](references),
which are discussed in the next section.
