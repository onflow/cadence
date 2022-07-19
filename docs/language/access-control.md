---
title: Access control
---

Access control allows making certain parts of the program accessible/visible
and making other parts inaccessible/invisible.

In Flow and Cadence, there are two types of access control:

1. Access control on objects in account storage using capability security.

    Within Flow, a caller is not able to access an object
    unless it owns the object or has a specific reference to that object.
    This means that nothing is truly public by default.
    Other accounts can not read or write the objects in an account
    unless the owner of the account has granted them access
    by providing references to the objects.

2. Access control within contracts and objects
   using `pub` and `access` keywords.

   For the explanations of the following keywords, we assume that
   the defining type is either a contract, where capability security
   doesn't apply, or that the caller would have valid access to the object
   governed by capability security.

The high-level reference-based security (point 1 above)
will be covered in a later section.

Top-level declarations
(variables, constants, functions, structures, resources, interfaces)
and fields (in structures, and resources) are always only able to be written
to and mutated (modified, such as by indexed assignment or methods like `append`)
in the scope where it is defined (self).

There are four levels of access control defined in the code that specify where
a declaration can be accessed or called.

- **Public** or **access(all)** means the declaration
  is accessible/visible in all scopes.

  This includes the current scope, inner scopes, and the outer scopes.

  For example, a public field in a type can be accessed using the access syntax
  on an instance of the type in an outer scope.
  This does not allow the declaration to be publicly writable though.

  An element is made publicly accessible / by any code
  by using the `pub` or `access(all)` keywords.

- **access(account)** means the declaration is only accessible/visible in the
  scope of the entire account where it is defined. This means that
  other contracts in the account are able to access it,

  An element is made accessible by code in the same account (e.g. other contracts)
  by using the `access(account)` keyword.

- **access(contract)** means the declaration is only accessible/visible in the
  scope of the contract that defined it. This means that other types
  and functions that are defined in the same contract can access it,
  but not other contracts in the same account.

  An element is made accessible by code in the same contract
  by using the `access(contract)` keyword.

- Private or **access(self)** means the declaration is only accessible/visible
  in the current and inner scopes.

  For example, an `access(self)` field can only be
  accessed by functions of the type is part of,
  not by code in an outer scope.

  An element is made accessible by code in the same containing type
  by using the `access(self)` keyword.

**Access level must be specified for each declaration**

The `(set)` suffix can be used to make variables also publicly writable and mutable.

To summarize the behavior for variable declarations, constant declarations, and fields:

| Declaration kind | Access modifier          | Read scope                                           | Write scope       | Mutate scope      |
|:-----------------|:-------------------------|:-----------------------------------------------------|:------------------|:------------------|
| `let`            | `priv` / `access(self)`  | Current and inner                                    | *None*            | Current and inner |
| `let`            | `access(contract)`       | Current, inner, and containing contract              | *None*            | Current and inner |
| `let`            | `access(account)`        | Current, inner, and other contracts in same account  | *None*            | Current and inner |
| `let`            | `pub`,`access(all)`      | **All**                                              | *None*            | Current and inner |
| `var`            | `access(self)`           | Current and inner                                    | Current and inner | Current and inner |
| `var`            | `access(contract)`       | Current, inner, and containing contract              | Current and inner | Current and inner |
| `var`            | `access(account)`        | Current, inner, and other contracts in same account  | Current and inner | Current and inner |
| `var`            | `pub` / `access(all)`    | **All**                                              | Current and inner | Current and inner |
| `var`            | `pub(set)`               | **All**                                              | **All**           | **All**           |

To summarize the behavior for functions:

| Access modifier          | Access scope                                        |
|:-------------------------|:----------------------------------------------------|
| `priv` / `access(self)`  | Current and inner                                   |
| `access(contract)`       | Current, inner, and containing contract             |
| `access(account)`        | Current, inner, and other contracts in same account |
| `pub` / `access(all)`    | **All**                                             |

Declarations of structures, resources, events, and [contracts](contracts) can only be public.
However, even though the declarations/types are publicly visible,
resources can only be created from inside the contract they are declared in.

```cadence
// Declare a private constant, inaccessible/invisible in outer scope.
//
access(self) let a = 1

// Declare a public constant, accessible/visible in all scopes.
//
pub let b = 2
```

```cadence
// Declare a public struct, accessible/visible in all scopes.
//
pub struct SomeStruct {

    // Declare a private constant field which is only readable
    // in the current and inner scopes.
    //
    access(self) let a: Int

    // Declare a public constant field which is readable in all scopes.
    //
    pub let b: Int

    // Declare a private variable field which is only readable
    // and writable in the current and inner scopes.
    //
    access(self) var c: Int

    // Declare a public variable field which is not settable,
    // so it is only writable in the current and inner scopes,
    // and readable in all scopes.
    //
    pub var d: Int

    // Declare a public variable field which is settable,
    // so it is readable and writable in all scopes.
    //
    pub(set) var e: Int

    // Arrays and dictionaries declared without (set) cannot be
    // mutated in external scopes
    pub let arr: [Int]

    // The initializer is omitted for brevity.

    // Declare a private function which is only callable
    // in the current and inner scopes.
    //
    access(self) fun privateTest() {
        // ...
    }

    // Declare a public function which is callable in all scopes.
    //
    pub fun privateTest() {
        // ...
    }

    // The initializer is omitted for brevity.

}

let some = SomeStruct()

// Invalid: cannot read private constant field in outer scope.
//
some.a

// Invalid: cannot set private constant field in outer scope.
//
some.a = 1

// Valid: can read public constant field in outer scope.
//
some.b

// Invalid: cannot set public constant field in outer scope.
//
some.b = 2

// Invalid: cannot read private variable field in outer scope.
//
some.c

// Invalid: cannot set private variable field in outer scope.
//
some.c = 3

// Valid: can read public variable field in outer scope.
//
some.d

// Invalid: cannot set public variable field in outer scope.
//
some.d = 4

// Valid: can read publicly settable variable field in outer scope.
//
some.e

// Valid: can set publicly settable variable field in outer scope.
//
some.e = 5

// Invalid: cannot mutate a public field in outer scope.
//
some.f.append(0)

// Invalid: cannot mutate a public field in outer scope.
//
some.f[3] = 1

// Valid: can call non-mutating methods on a public field in outer scope
some.f.contains(0)
```
