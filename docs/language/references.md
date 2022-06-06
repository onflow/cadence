---
title: References
---

It is possible to create references to objects, i.e. resources or structures.
A reference can be used to access fields and call functions on the referenced object.

References are **copied**, i.e. they are value types.

References are created by using the `&` operator, followed by the object,
the `as` keyword, and the type through which they should be accessed.
The given type must be a supertype of the referenced object's type.

References have the type `&T`, where `T` is the type of the referenced object.

```cadence
let hello = "Hello"

// Create a reference to the "Hello" string, typed as a `String`
//
let helloRef: &String = &hello as &String

helloRef.length // is `5`

// Invalid: Cannot create a reference to `hello`
// typed as `&Int`, as it has type `String`
//
let intRef: &Int = &hello as &Int
```

If you attempt to reference an optional value, you will receive an optional reference.
If the referenced value is nil, the reference itself will be nil. If the referenced value
exists, then forcing the optional reference will yield a reference to that value:

```cadence
let nil: String? = nil
let nilRef = &n as &String? // r has type &String?
let n = r! // error, forced nil value

let str: String? = ""
let strRef = &n as &String? // r has type &String?
let n = r! // n has type &String
```

References are covariant in their base types.
For example, `&T` is a subtype of `&U`, if `T` is a subtype of `U`.

```cadence

// Declare a resource interface named `HasCount`,
// that has a field `count`
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

    pub fun increment() {
        self.count = self.count + 1
    }
}

// Create a new instance of the resource type `Counter`
// and create a reference to it, typed as `&Counter`,
// so the reference allows access to all fields and functions
// of the counter
//
let counter <- create Counter(count: 42)
let counterRef: &Counter = &counter as &Counter

counterRef.count  // is `42`

counterRef.increment()

counterRef.count  // is `43`
```

References may be **authorized** or **unauthorized**.

Authorized references have the `auth` modifier, i.e. the full syntax is `auth &T`,
whereas unauthorized references do not have a modifier.

Authorized references can be freely upcasted and downcasted,
whereas unauthorized references can only be upcasted.
Also, authorized references are subtypes of unauthorized references.

```cadence

// Create an unauthorized reference to the counter,
// typed with the restricted type `&{HasCount}`,
// i.e. some resource that conforms to the `HasCount` interface
//
let countRef: &{HasCount} = &counter as &{HasCount}

countRef.count  // is `43`

// Invalid: The function `increment` is not available
// for the type `&{HasCount}`
//
countRef.increment()

// Invalid: Cannot conditionally downcast to reference type `&Counter`,
// as the reference `countRef` is unauthorized.
//
// The counter value has type `Counter`, which is a subtype of `{HasCount}`,
// but as the reference is unauthorized, the cast is not allowed.
// It is not possible to "look under the covers"
//
let counterRef2: &Counter = countRef as? &Counter

// Create an authorized reference to the counter,
// again with the restricted type `{HasCount}`, i.e. some resource
// that conforms to the `HasCount` interface
//
let authCountRef: auth &{HasCount} = &counter as auth &{HasCount}

// Conditionally downcast to reference type `&Counter`.
// This is valid, because the reference `authCountRef` is authorized
//
let counterRef3: &Counter = authCountRef as? &Counter

counterRef3.count  // is `43`

counterRef3.increment()

counterRef3.count  // is `44`
```

References are ephemeral, i.e they cannot be [stored](accounts#account-storage).
Instead, consider [storing a capability and borrowing it](capability-based-access-control) when needed.
