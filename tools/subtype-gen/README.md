# Rules

A rules file contains zero or more rules.
A single rule consist of the two properties: The super-type (`super`) for which this rule applies to,
and a predicate (`predicate`) defining the conditions that needs to be satisfied by a type to be a subtype of this super-type.

```yaml
rules :
  - super: T1
    predicate: P1

  - super: T2
    predicate: P2
    
  ...
```

## Predicates

A predicate is a condition that would become either `true` or `false` depending on the inputs.
Below are the different types of predicates that can be used to define the subtype rules for a given super-type.

### Always Predicate (`always`)
 
Represents a condition that is always true.

### Never Predicate (`never`)

Represents a condition that is always false.

### Not Predicate (`not`)

Negates a nested predicate.

### And Predicate (`and`)

Represent a list of nested predicate that needs to be satisfied.
An `and` predicate becomes true only if all of its nested predicates are also true.
e.g:

```yaml
and:
  - P1
  - P2
```

### Or Predicate (`or`)

Represent a list of nested predicate that could be satisfied.
An `or` predicate becomes true if at-least one of its nested predicates are also true.
e.g:

```yaml
and:
  - P1
  - P2
```

### Equals Predicate (`equals`)

Checks the equality between two values.
Consist of two properties, `source` and `target`, both of which can be any [expression](#Expressions).

Example 1:

```yaml
equals:
  source: super
  target: sub
```

Example 2:

```yaml
equals:
  source: sub
  target:
    oneOfTypes: [PrivatePathType, PublicPathType]
```

### Subtype Predicate (`subtype`)

Check whether one value is a subtype of another.
Consist of two fields, `source` and `target`, both can be any [expression](#Expressions).

Example 1:

```yaml
equals:
  source: super
  target: sub
```

Example 2:

```yaml
equals:
  source: sub
  target:
    oneOfTypes: [PrivatePathType, PublicPathType]
```

### Type-assertion Predicate (`mustType`)

Uses `mustType` keyword. Can be used to assert that a value belong to a certain type.
e.g:

```yaml
- mustType:
    source: super
    type: CompositeType
```

### Set-Contains Predicate (`setContains`)

Check whether a set contains a given value.
e.g:

```yaml
- setContains:
    source: super.EffectiveInterfaceConformanceSet
    target: sub
```

### Permits Predicate (`permits`)

Check whether one set of authorization permits the access of the second set of authorization.
e.g:

```yaml
- permits:
    super: super.Authorization
    sub: sub.Authorization
```

### Is-Intersection-Subset Predicate (`isIntersectionSubset`)

Check whether the interfaces of `sub` is a subset of the interface set of the `super`.
e.g:

```yaml
- isIntersectionSubset:
    super: super
    sub: sub
```

### Other function specific predicates

There are some more predicates that checks for specific conditions:

#### isResource

 Checks whether the type provided is a resource type.
e.g:
```yaml
isResource: sub
```

#### isAttachment

Checks whether the type provided is an attachment type.
e.g:
```yaml
isAttachment: sub
```

#### isHashableStruct

Checks whether the type provided is a hashable struct type.
e.g:
```yaml
isHashableStruct: sub
```

#### isStorable

Checks whether the type provided is a storable type.
e.g:
```yaml
isStorable: sub
```

#### isStorable

Checks whether the type provided is a storable type.
e.g:
```yaml
isStorable: sub
```

## Variables

Defining custom variables to hold values inside a rule is currently not supported.
However, there are two variables that are supported by default: `super` and `sub`.

### `super` variable

Holds the type-value of the super-type passed on to the function.
The super-type will always have the same type as the type `T` specified in the `super` field of a `rule`.

For example:

```yaml
  - super: AnyStructType
    predicate:
      equals:
        source: super  # super is a AnyStructType
        target: nil
```

In the above, `super` variable inside `equals` predicate contains a type-value of type `AnyStructType`.

Further, if `super` was used inside a type-assertion predicate (`mustType`) having `target` type as `R`,
then any subsequent predicates that are combined with this type-assertion (using an `and` predicate),
would see the type of `super` as `R`.

```yaml
  - super: AnyStructType
    predicate:
      and:
      - mustType:              # type assertion for `super`
          source: super
          type: CompositeType
      - equals:
          source: super        # `super` is a `CompositeType`
          target: nil
```

### `sub` variable

Holds the type-value of the subtype passed on to the function.
Unlike `super`, the subtype will always belong to root type of the type implementation, at the start.

For example:

```yaml
  - super: AnyStructType
    predicate:
      equals:
        source: sub  # `sub` is a `Type`
        target: nil
```

In the above, `super` variable inside `equals` predicate contains a type-value of the root type `Type`.

However, if `sub` was used inside a type-assertion predicate (`mustType`) having `target` type as `R`,
then any subsequent predicates that are combined with this type-assertion (using an `and` predicate),
would see the type of `super` as `R`.

```yaml
  - super: AnyStructType
    predicate:
      and:
      - mustType:              # type assertion for `sub`
          source: sub
          type: CompositeType
      - equals:
          source: sub         # `sub` is a `CompositeType`
          target: nil
```


## Expressions

Predicates have properties/fields, and these properties sometimes requires to refer to different values, types,
functions, fields, to properly define the subtyping rules.
Expressions can be used to represent/access such information.

### Identifier expression

Any name is an identifier expression. e.g: `sub` and `super` are identifiers that refer to variables.

### Type expression

Special kind of identifier expression that refer to types.
They always ends with the `Type` suffix.
e.g:

```yaml
  - super: AnyStructType
    predicate:
      and:
      - mustType:
          source: sub
          type: CompositeType  # `CompositeType` is a type-expression, referring to a type.

```

### Member expression

Can be used to access fields of a variable.
e.g:

```yaml
  - super: DictionaryType
    predicate:
      and:
      - equals:
          source: super.KeyType    # Refers to the `keyType` field/property of `DictionaryType`.
          target: String

```

### One-Of-Types expression

Used to represent that the value of the expression can be a one-of multiple types.
It's a convenient way to represent an `or` relationship in a concise way.
e.g:

```yaml
  - super: DictionaryType
    predicate:
      and:
      - equals:
          source: super.KeyType
          target:
            oneOfTypes: [Int, String, Bool]  # Says the `keyType` field/property of `DictionaryType`.
                                             # can be one of Int, String, Bool.

```
