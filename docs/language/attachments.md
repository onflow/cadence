---
title: Attachments
---

<Callout type="warning">
⚠️  This section describes a feature that is not yet released on Mainnet. 
It will be available following the next Mainnet Spork. 
</Callout>

Attachments are a feature of Cadence designed to allow developers to extend a struct or resource type 
(even one that they did not declare) with new functionality,
without requiring the original author of the type to plan or account for the intended behavior. 

## Declaring Attachments

Attachments are declared with the `attachment` keyword, which would be declared using a new form of composite declaration:
`pub attachment <Name> for <Type>: <Conformances> { ... }`, where the attachment functions and fields are declared in the body. 
As such, the following would be examples of legal declarations of attachments:

```cadence
pub attachment Foo for MyStruct {
    // ...
}

attachment Bar for MyResource: MyResourceInterface {
    // ...
}

attachment Baz for MyInterface: MyOtherInterface {
    // ...
}
```

Specifying the kind (struct or resource) of an attachment is not necessary, as its kind will necessarily be the same as the type it is extending. 
Note that the base type may be either a concrete composite type or an interface.
In the former case, the attachment is only usable on values specifically of that base type, 
while in the case of an interface the attachment is usable on any type that conforms to that interface. 

As with other type declarations, attachments may only have a `pub` access modifier (if one is present). 

The body of the attachment follows the same declaration rules as composites. 
In particular, they may have both field and function members,
and any field members must be initialized in an `init` function. 
Only resource-kinded attachments may have resource members, 
and such members must be explicitly handled in the `destroy` function. 
The `self` keyword is available in attachment bodies, but unlike in a composite, 
`self` is a **reference** type, rather than a composite type: 
In an attachment declaration for `A`, the type of `self` would be `&A`, rather than `A` like in other composite declarations.

If a resource with attachments on it is `destroy`ed, the `destroy` functions of all its attachments are all run in an unspecified order; 
`destroy` should not rely on the presence of other attachments on the base type in its implementation. 
The only guarantee about the order in which attachments are destroyed in this case is that the base resource will be the last thing destroyed. 

Within the body of an attachment, there is also a `base` keyword available, 
which contains a reference to the attachment's base value; 
that is, the composite to which the attachment is attached.
Its type, therefore, is a reference to the attachment's declared base type.
So, for an attachment declared `pub attachment Foo for Bar`, the `base` field of `Foo` would have type `&Bar`.

So, for example, this would be a valid declaration of an attachment:

```
pub resource R {
    pub let x: Int

    init (_ x: Int) {
        self.x = x
    }

    pub fun foo() { ... }
}

pub attachment A for R {
    pub let derivedX: Int 

    init (_ scalar: Int) {
        self.derivedX = base.x * scalar
    }

    pub fun foo() {
        base.foo()
    }
}

```

For the purposes of external mutation checks or [access control](/cadence/language/access-control), 
the attachment is considered a separate declaration from its base type. 
A developer cannot, therefore, access any `priv` fields 
(or `access(contract)` fields if the base was defined in a different contract to the attachment)
on the `base` value, nor can they mutate any array or dictionary typed fields.

```cadence
pub resource interface SomeInterface {
    pub let b: Bool
    priv let i: Int
    pub let a: [String]
}
pub attachment SomeAttachment for SomeContract.SomeStruct { 
    pub let i: Int
    init(i: Int) {
        if base.b {
            self.i = base.i // cannot access `i` on the `base` value
        } else {
            self.i = i
        }
    }
    pub fun foo() {
        base.a.append("hello") // cannot mutate `a` outside of the composite where it was defined
    }
}
```

### Attachment Types

An attachment declared with `pub attachment A for C { ... }` will have a nominal type `A`.

It is important to note that attachments are not first class values in Cadence, and as such their usage is limited in certain ways.  
In particular, their types cannot appear outside of a reference type. 
So, for example, given an  attachment declaration `attachment A for X {}`, the types `A`, `A?`, `[A]` and `((): A)` are not valid type annotations, 
while `&A`, `&A?`, `[&A]` and `((): &A)` are valid. 

## Creating Attachments

An attachment is created using an `attach` expression, 
where the attachment is both initialized and attached to the base value in a single operation. 
Attachments are not first-class values in Cadence; they cannot exist independently of a base value, 
nor can they be moved around on their own. 
This means that an `attach` expression is the only place in which an attachment constructor can be called. 
Tightly coupling the creation and attaching of attachment values helps to make reasoning about attachments simpler for the user. 
Also for this reason, resource attachments do not need an expliict `<-` move operator when they appear in an `attach` expression. 

An attach expression consists of the `attach` keyword, a constructor call for the attachment value, 
the `to` keyword, and an expression that evaluates to the base value for that attachment. 
Any arguments required by the attachment's `init` function are provided in its constructor call. 

```cadence
pub resource R {}
pub attachment A for R {
    init(x: Int) {
        //...
    }
}

// ...
let r <- create R()
let r2 <- attach A(x: 3) to <-r
```

The expression on the right-hand side of the `to` keyword must evaluate to a composite value whose type is a subtype of the attachment's base, 
and is evaluated before the call to the constructor on the left side of `to`. 
This means that the `base` value is available inside of the attachment's `init` function,
but it is important to note that the attachment being created will not be accessible on the `base` 
(see the accessing attachments section below) until after the constructor finishes executing. 


```cadence
pub resource interface I {}
pub resource R: I {}
pub attachment A for I {}

// ...
let r <- create R() // has type @R
let r2 <- attach A() to <-r // ok, because `R` is a subtype of `I`, still has type @R
```

Because attachments are stored on their bases by type, there can only be one attachment of each type present on a value at a time.
Cadence will raise a runtime error if a user attempts to add an attachment to a value when one it already exists on that value.
The type returned by the `attach` expression is the same type as the expression on the right-hand side of the `to`; 
attaching an attachment to a value does not change its type. 

## Accessing Attachments

Attachments are accessed on composites via type-indexing: 
composite values function like a dictionary where the keys are types and the values are attachments. 
So given a composite value `v`, one can look up the attachment named `A` on `v` using indexing syntax:

```cadence
let a = v[A] // has type `&A?`
```

This syntax requires that `A` is a nominal attachment type,
and that `v` has a composite type that is a subtype of `A`'s declared base type. 
As mentioned above, attachments are not first-class values in Cadence, 
so this indexing returns a reference to the attachment on `v`, rather than the attachment itself. 
If the attachment with the given type does not exist on `v`, this expression returns `nil`. 

Because the indexed value must be a subtype of the indexing attachment's base type,
the owner of a resource can restrict which attachments can be accessed on references to their resource using restricted types, 
much like they would do with any other field or function. E.g.

```cadence
struct R: I {}
struct interface I {}
attachment A for R {}
fun foo(r: &{I}) {
    r[A] // fails to type check, because `{I}` is not a subtype of `R`
}
```

Hence, if the owner of a resource wishes to allow others to use a subset of its attachments, 
they can create a capability to that resource with a borrow type that only allows certain attachments to be accessed. 

## Removing Attachments

Attachments can be removed from a value with a `remove` statement. 
The statement consists of the `remove` keyword, the nominal type for the attachment to be removed, 
the `from` keyword, and the value from which the attachment is meant to be removed. 

The value on the right-hand side of `from` must be a composite value whose type is a subtype of the attachment type's declared base. 

Before the statement executes, the attachment's `destroy` function (if present) will be executed. 
After the statement executes, the composite value on the right-hand side of `from` will no longer contain the attachment.
If the value does not contain `t`, this statement has no effect. 

Attachments may be removed from a type in any order,
so users should take care not to design any attachments that rely on specific behaviors of other attachments, 
as there is no to require that an attachment depend on another or to require that a type has a given attachment when another attachment is present. 

If a resource containing attachments is `destroy`ed, all its attachments will be `destroy`ed in an arbitrary order. 
