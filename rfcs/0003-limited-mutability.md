# Feature name

- Proposal: RFC-0003
- Authors: @dsainati1
- Status: Awaiting implementation
- Issues: [#1260](https://github.com/onflow/cadence/issues/1260)

## Summary

[summary]: #summary

This proposed change would limit the scopes in which the fields of composite types
like contracts, structs, and resources can be mutated. Instead of allowing array 
and dictionary fields to be modified in any scope where the field can be read, instead
Cadence would issue a type error. These fields would instead be only modifiable 
in the current declaration scope, as well as inner scopes of that scope. 

## Motivation

[motivation]: #motivation

Accidentally exposing a mutable variable to external consumers of a contract is currently a 
large potential footgun standing in the way of a release of a stable, trustless version of
Cadence. Developers may declare a "constant" field on their contract with `pub let`, intending
that the field only be readable to transactions and other contracts, and unintentially allow
other code to add or remove elements from a dictionary or array stored in that field. Consider this code:

```
pub contract Foo {
	pub let x : [Int]

	init() {
	    self.x = []
	}
}

// in some external code importing Foo
pub fun bar() {
	Foo.x.append(1)
}
```

Currently Cadence does not warn against this, or prevent a developer from writing this code, even
though, depending on what is stored in `x`, this could be unsafe. 

## Explanation

[explanation]: #explanation

Cadence controls where and to what extent variables can be read and written using a combination of
access modifiers and declaration kinds, as described [here](https://docs.onflow.org/cadence/language/access-control/).
Of note is that the `let` kind does not allow fields to be written to in any scope, whereas `var` allows them
to be written in the "Current and Inner" scopes; that is, the scope in which the field was declared, and any scopes
contained within that scope. 

However, simply writing to a field directly is not the only way in which one can modify a value. Consider the following
example:

```
pub struct Foo {
    pub let x : [Int]

    init() {
        self.x = [3];
    }
}

pub fun bar() {
    let foo = Foo()
    foo.x = [0] // writes to x, not allowed
    foo.x[0] = 0; // does not write to x, also not allowed
}
```

Cadence also restricts the scopes in which an array or dictionary field can be modified (or "mutated"). Examples of 
mutating operations include an indexed assignment, as in the above example, as well as calls to the `append` or `remove`,
methods of arrays, or the `insert` or `remove` methods of dictionaries. These operations can only be performed on a field
in the current and inner scopes, the same contexts in which the field could be written to if it were a `var`. So the following 
would typecheck:

```
pub struct Foo {
    pub let x : [Int]

    init() {
        self.x = [3];
    }

    pub fun addToX(i: Int) {
        self.x.append(i)
    }
}
```

while the following would not:

```
pub struct Foo {
    pub let y : [Int]

    init() {
        self.y = [3];
    }
}

pub fun addToY(foo: Foo, i: Int) {
    foo.y.append(i)
}
```

This prevents external code from mutating the values of fields it can read from your contract. Consumers
of your contract may read the values in a `pub let` or `pub var` field, but cannot change them in any way. 

If you wish to allow other code to update or modify a field in your contract, you may expose a method 
to do so, like in the example above with `addToX`, or you may use the `pub(set)` access mode, which 
allows any code to mutate or write to the field it applies to. 


## Detailed design

[detailed-design]: #detailed-design

This change adds a new error, the `ExternalMutationError`, which is raised when a field 
is mutated outside of the context in which it was defined. The error message will also
suggest that the user instead use a setter or modifier method for that field.

Specifically, the error is emitted whenever a user attempts to perform an 
index assignment on a member that is not either declared with the `pub(set)` 
access mode, or is defined in the current enclosing scope. This check is the
same one performed for writing to fields, with the difference that mutation 
is allowed on both `let` and `var` fields, while only the latter can be written to.

Additionally, array and dictionary methods now track an additional bit of information
indicating whether they mutate their receiver. Mutating methods may not be called on 
members that are not declared with the `pub(set)` access mode, or defined in the current
enclosing scope. 

The array methods that are considered mutating are `append`, `appendAll`, `remove`,
`insert`, `removeFirst` and `removeLast`. The dictionary methods that are considered
mutating are `insert` and `remove`. 

The limitations on mutation are designed to closely mirror the limitations on writes, 
so that they can be easily explained to and understood by the user in terms of 
language principles with which they are already familiar. Similarly, the suggested
workaround of adding a setter or modifier method to the composite type is designed 
to be immediately recognizable to any developer familiar with object-oriented
design principles. 

## Drawbacks

[drawbacks]: #drawbacks

Why should we *not* do this?

## Alternatives

[alternatives]: #alternatives

What other designs have been considered and what is the rationale for not choosing them?

## Prior art

[prior-art]: #prior-art

Does this feature exist in other programming languages and what experience have their community had?

This section is intended to encourage you as an author to think about the lessons from other languages, provide readers of your RFC with a fuller picture.

If there is no prior art, that is fine - your ideas are interesting to us whether they are brand new or if it is an adaptation from other languages.

## Unresolved questions

[unresolved-questions]: #unresolved-questions

What parts of the design are or do you expect to be still resolved?

## Related

[related]: #related

What related issues do you consider out of scope for this RFC that could be addressed in the future independently of the solution that comes out of this RFC?
