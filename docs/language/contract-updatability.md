---
title: Contract Updatability
---

## Introduction
A [contract](contracts) in Cadence is a collection of data (its state) and
code (its functions) that lives in the contract storage area of an account.
When a contract is updated, it is important to make sure that the changes introduced do not lead to runtime
inconsistencies for already stored data.
Cadence maintains this state consistency by validating the contracts and all their components before an update.

## Validation Goals
The contract update validation ensures that:

- Stored data doesn't change its meaning when a contract is updated.
- Decoding and using stored data does not lead to runtime crashes.
  - For example, it is invalid to add a field because existing stored data won't have the new field.
  - Loading the existing data will result in garbage/missing values for such fields.
  - A static check of the access of the field would be valid, but the interpreter would crash when accessing the field,
    because the field has a missing/garbage value.

However, it **does not** ensure:
- Any program that imports the updated contract stays valid. e.g:
  - Updated contract may remove an existing field or may change a function signature.
  - Then any program that uses that field/function will get semantic errors.

## Updating a Contract
Changes to contracts can be introduced by adding new contracts, removing existing contracts, or updating existing
contracts. However, some of these changes may lead to data inconsistencies as stated above.

#### Valid Changes
- Adding a new contract is valid.
- Removing a contract/contract-interface that doesn't have enum declarations is valid.
- Updating a contract is valid, under the restrictions described in the below sections.

#### Invalid Changes
- Removing a contract/contract-interface that contains enum declarations is not valid.
  - Removing a contract allows adding a new contract with the same name.
  - The new contract could potentially have enum declarations with the same names as in the old contract, but with
    different structures.
  - This could change the meaning of the already stored values of those enum types.

A contract may consist of fields and other declarations such as composite types, functions, constructors, etc.
When an existing contract is updated, all its inner declarations are also validated.

### Contract Fields
When a contract is deployed, the fields of the contract are stored in an account's contract storage.
Changing the fields of a contract only changes the way the program treats the data, but does not change the already
stored data itself, which could potentially result in runtime inconsistencies as mentioned in the previous section.

See the [section about fields below](#fields) for the possible updates that can be done to the fields, and the restrictions
imposed on changing fields of a contract.

### Nested Declarations
Contracts can have nested composite type declarations such as structs, resources, interfaces, and enums.
When a contract is updated, its nested declarations are checked, because:
 - They can be used as type annotation for the fields of the same contract, directly or indirectly.
 - Any third-party contract can import the types defined in this contract and use them as type annotations.
 - Hence, changing the type definition is the same as changing the type annotation of such a field (which is also invalid,
   as described in the [section about fields fields](#fields) below).

Changes that can be done to the nested declarations, and the update restrictions are described in following sections:
 - [Structs, resources and interface](#structs-resources-and-interfaces)
 - [Enums](#enums)
 - [Functions](#functions)
 - [Constructors](#constructors)

## Fields
A field may belong to a contract, struct, resource, or interface.

#### Valid Changes:
- Removing a field is valid
  ```cadence
  // Existing contract

  pub contract Foo {
      pub var a: String
      pub var b: Int
  }


  // Updated contract

  pub contract Foo {
      pub var a: String
  }
  ```
  - It leaves data for the removed field unused at the storage, as it is no longer accessible.
  - However, it does not cause any runtime crashes.

- Changing the order of fields is valid.
  ```cadence
  // Existing contract

  pub contract Foo {
      pub var a: String
      pub var b: Int
  }


  // Updated contract

  pub contract Foo {
      pub var b: Int
      pub var a: String
  }
  ```

- Changing the access modifier of a field is valid.
  ```cadence
  // Existing contract

  pub contract Foo {
      pub var a: String
  }


  // Updated contract

  pub contract Foo {
      priv var a: String   // access modifier changed to 'priv'
  }
  ```

#### Invalid Changes
- Adding a new field is not valid.
  ```cadence
  // Existing contract

  pub contract Foo {
      pub var a: String
  }


  // Updated contract

  pub contract Foo {
      pub var a: String
      pub var b: Int      // Invalid new field
  }
  ```
    - Initializer of a contract only run once, when the contract is deployed for the first time. It does not rerun
      when the contract is updated. However it is still required to be present in the updated contract to satisfy type checks.
    - Thus, the stored data won't have the new field, as the initializations for the newly added fields do not get
      executed.
    - Decoding stored data will result in garbage or missing values for such fields.

- Changing the type of existing field is not valid.
  ```cadence
  // Existing contract

  pub contract Foo {
      pub var a: String
  }


  // Updated contract

  pub contract Foo {
      pub var a: Int      // Invalid type change
  }
  ```
    - In an already stored contract, the field `a` would have a value of type `String`.
    - Changing the type of the field `a` to `Int`, would make the runtime read the already stored `String`
      value as an `Int`, which will result in deserialization errors.
    - Changing the field type to a subtype/supertype of the existing type is also not valid, as it would also
      potentially cause issues while decoding/encoding.
      - e.g: Changing an `Int64` field to `Int8` - Stored field could have a numeric value`624`, which exceeds the value space
        for `Int8`.
      - However, this is a limitation in the current implementation, and the future versions of Cadence may support
        changing the type of field to a subtype, by providing means to migrate existing fields.

## Structs, Resources and Interfaces

#### Valid Changes:
- Adding a new struct, resource, or interface is valid.
- Adding an interface conformance to a struct/resource is valid, since the stored data only
  stores concrete type/value, but doesn't store the conformance info.
  ```cadence
  // Existing struct

  pub struct Foo {
  }


  // Upated struct

  pub struct Foo: T {
  }
  ```
  - However, if adding a conformance also requires changing the existing structure (e.g: adding a new field that is
    enforced by the new conformance), then the other restrictions (such as [restrictions on fields](#fields)) may
    prevent performing such an update.

#### Invalid Changes:
- Removing an existing declaration is not valid.
  - Removing a declaration allows adding a new declaration with the same name, but with a different structure.
  - Any program that uses that declaration would face inconsistencies in the stored data.
- Renaming a declaration is not valid. It can have the same effect as removing an existing declaration and adding
  a new one.
- Changing the type of declaration is not valid. i.e: Changing from a struct to interface, and vise versa.
  ```cadence
  // Existing struct

  pub struct Foo {
  }


  // Changed to a struct interface

  pub struct interface Foo {    // Invalid type declaration change
  }
  ```
- Removing an interface conformance of a struct/resource is not valid.
  ```cadence
  // Existing struct

  pub struct Foo: T {
  }


  // Upated struct

  pub struct Foo {
  }
  ```

### Updating Members
Similar to contracts, these composite declarations: structs, resources, and interfaces also can have fields and
other nested declarations as its member.
Updating such a composite declaration would also include updating all of its members.

Below sections describes the restrictions imposed on updating the members of a struct, resource or an interface.
- [Fields](#fields)
- [Nested structs, resources and interfaces](#structs-resources-and-interfaces)
- [Enums](#enums)
- [Functions](#functions)
- [Constructors](#constructors)

## Enums

#### Valid Changes:
- Adding a new enum declaration is valid.

#### Invalid Changes:
- Removing an existing enum declaration is invalid.
  - Otherwise, it is possible to remove an existing enum and add a new enum declaration with the same name,
    but with a different structure.
  - The new structure could potentially have incompatible changes (such as changed types, changed enum-cases, etc).
- Changing the name is invalid, as it is equivalent to removing an existing enum and adding a new one.
- Changing the raw type is invalid.
  ```cadence
  // Existing enum with `Int` raw type

  pub enum Color: Int {
    pub case RED
    pub case BLUE
  }


  // Updated enum with `UInt8` raw type

  pub enum Color: UInt8 {    // Invalid change of raw type
    pub case RED
    pub case BLUE
  }
  ```
  - When the enum value is stored, the raw value associated with the enum-case gets stored.
  - If the type is changed, then deserializing could fail if the already stored values are not in the same value space
    as the updated type.

### Updating Enum Cases
Enums consist of enum-case declarations, and updating an enum may also include changing the enums cases as well.
Enum cases are represented using their raw-value at the Cadence interpreter and runtime.
Hence, any change that causes an enum-case to change its raw value is not permitted.
Otherwise, a changed raw-value could cause an already stored enum value to have a different meaning than what
it originally was (type confusion).

#### Valid Changes:
- Adding an enum-case at the end of the existing enum-cases is valid.
  ```cadence
  // Existing enum

  pub enum Color: Int {
    pub case RED
    pub case BLUE
  }


  // Updated enum

  pub enum Color: Int {
    pub case RED
    pub case BLUE
    pub case GREEN    // valid new enum-case at the bottom
  }
  ```
#### Invalid Changes
- Adding an enum-case at the top or in the middle of the existing enum-cases is invalid.
  ```cadence
  // Existing enum

  pub enum Color: Int {
    pub case RED
    pub case BLUE
  }


  // Updated enum

  pub enum Color: Int {
    pub case RED
    pub case GREEN    // invalid new enum-case in the middle
    pub case BLUE
  }
  ```
- Changing the name of an enum-case is invalid.
  ```cadence
  // Existing enum

  pub enum Color: Int {
    pub case RED
    pub case BLUE
  }


  // Updated enum

  pub enum Color: Int {
    pub case RED
    pub case GREEN    // invalid change of names
  }
  ```
  - Previously stored raw values for `Color.BLUE` now represents `Color.GREEN`. i.e: The stored values have changed
    their meaning, and hence not a valid change.
  - Similarly, it is possible to add a new enum with the old name `BLUE`, which gets a new raw value. Then the same
    enum-case `Color.BLUE` may have used two raw-values at runtime, before and after the change, which is also invalid.

- Removing the enum case is invalid. Removing allows one to add and remove an enum-case which has the same effect
  as renaming.
  ```cadence
  // Existing enum

  pub enum Color: Int {
    pub case RED
    pub case BLUE
  }


  // Updated enum

  pub enum Color: Int {
    pub case RED

    // invalid removal of `case BLUE`
  }
  ```
- Changing the order of enum-cases is not permitted
  ```cadence
  // Existing enum

  pub enum Color: Int {
    pub case RED
    pub case BLUE
  }


  // Updated enum

  pub enum Color: UInt8 {
    pub case BLUE   // invalid change of order
    pub case RED
  }
  ```
  - Raw value of an enum is implicit, and corresponds to the defined order.
  - Changing the order of enum-cases has the same effect as changing the raw-value, which could cause storage
    inconsistencies and type-confusions as described earlier.

## Functions
Updating a function definition is always valid, as function definitions are never stored as data.
i.e: Function definition is a part of the code, but not data.
- Changing a function signature (parameters, return types) is valid.
- Changing a function body is also valid.
- Changing the access modifier is valid.

However, changing a *function type* may or may not be valid, depending on where it is used.
i.e: If a function type is used in the type annotation of a composite type field (direct or indirect), then changing
the function type signature is the same as changing the type annotation of that field (which is again invalid).

## Constructors
Similar to functions, constructors are also not stored. Hence, any changes to constructors are valid.

## Imports
A contract may import declarations (types, functions, variables, etc.) from other programs. These imported programs are
already validated at the time of their deployment. Hence, there is no need for validating any declaration every time
they are imported.
