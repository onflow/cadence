
# Cadence Language Specification

**THIS IS THE START OF A SPECIFICATION**
**: It is incomplete and may contain many errors.**

## I. Introduction

### 1. Purpose of the Specification
The purpose of this specification is to provide a formal and comprehensive definition of the Cadence programming language. Cadence is specifically designed for writing smart contracts and decentralized applications (dApps) on the Flow blockchain. This document serves as an authoritative guide for the language's syntax, semantics, type system, and execution model, ensuring consistent implementation and usage across the Flow ecosystem.

The primary goals of this specification are to:
- Provide an unambiguous description of Cadence's structure and behavior.
- Facilitate the development of tooling, such as compilers, interpreters, and debuggers, that conform to the same standards.
- Support developers in writing secure, efficient, and maintainable smart contracts.
- Guide the evolution of Cadence by ensuring changes and new features maintain compatibility and correctness within the language framework.

The intended audience of this specification includes:
- **Language Implementers**: Individuals or teams responsible for creating or maintaining Cadence compilers, interpreters, and other development tools.
- **Developers**: Software engineers writing smart contracts or dApps using Cadence on the Flow blockchain.
- **Tool Makers**: Developers building integrated development environments (IDEs), testing tools, and utilities that work with Cadence.

### 2. Overview of Cadence
Cadence is a resource-oriented programming language that prioritizes safety, clarity, and developer experience in the context of blockchain development. Designed by Flow Foundation, Cadence enables secure and scalable development of smart contracts, particularly for applications involving digital assets such as non-fungible tokens (NFTs) and decentralized finance (DeFi) protocols.

The primary design philosophy behind Cadence is **resource-oriented programming**, which introduces the concept of resources as first-class citizens. Resources represent digital assets that are protected by strict rules ensuring that they cannot be accidentally duplicated or lost. By enforcing ownership and access control at the language level, Cadence minimizes many common security vulnerabilities found in blockchain programming.

Cadence is the core language for smart contracts on the Flow blockchain, a platform built to support high-performance dApps for mainstream adoption. Flow's account-based model, combined with Cadence's strong typing and resource management features, provides a secure and user-friendly environment for developers to create complex decentralized applications.

In comparison to other blockchain languages like Solidity (used on Ethereum), Cadence emphasizes **safety** and **developer ergonomics**. While Solidity relies on more traditional programming paradigms, Cadence’s resource-oriented model provides built-in safeguards against common issues like unintended re-entrancy or resource mismanagement, making it especially suitable for managing valuable digital assets.

### 3. Scope
This specification covers the complete formal definition of the Cadence programming language, including:
- **Lexical Structure**: The basic building blocks of the language, such as tokens, identifiers, and literals.
- **Syntax and Grammar**: The rules governing the structure of Cadence programs, including statements, expressions, and declarations.
- **Type System**: The various data types supported by Cadence, including primitives, resources, structs, and optionals.
- **Resource Management**: The ownership and lifecycle of resources, ensuring they cannot be copied or deleted incorrectly.
- **Execution Model**: The runtime behavior of Cadence programs, including how contracts are executed and transactions are processed.

This specification does not cover:
- **Flow Blockchain Consensus Protocol**: The underlying mechanisms of the Flow blockchain, including its consensus algorithm and network architecture.
- **Flow-Specific APIs and Services**: Detailed descriptions of APIs or services that interact with the blockchain but are external to the core Cadence language.

## II. Lexical Structure

The lexical structure of Cadence defines the basic building blocks from which Cadence programs are constructed. This section outlines the set of characters allowed in Cadence, how identifiers are named, the use of keywords, literals, and comments.

---

### 1. Character Set

Cadence source code is written using the **Unicode** character set (UTF-8 encoding), which allows for a wide range of characters from various languages. However, the core lexical elements, such as identifiers, operators, and punctuation, are primarily restricted to the ASCII subset of Unicode.

- **Allowed Characters**: 
  - Letters (a-z, A-Z)
  - Digits (0-9)
  - Special symbols used in operators and syntax, such as `+`, `-`, `*`, `/`, `=`, `{`, `}`, `(`, `)`, `[`, `]`, etc.
  - Whitespace characters: space (`U+0020`), tab (`U+0009`), newline (`U+000A`), and carriage return (`U+000D`).

- **Unicode Support**:
  - While the core syntax is based on ASCII characters, Unicode is fully supported in **string literals** and **comments**.
  - Identifiers may include Unicode characters in the ranges of letters and numbers but should generally follow best practices for naming and clarity.

---

### 2. Identifiers

Identifiers in Cadence are used to name **variables**, **constants**, **functions**, **contracts**, **types**, and other user-defined entities. Identifiers must follow specific rules to ensure consistency and avoid conflicts with reserved keywords.

- **Rules for Naming Identifiers**:
  - Identifiers must start with a **letter** (either uppercase or lowercase) or an **underscore (`_`)**. 
  - After the first character, identifiers may contain **letters**, **digits (0-9)**, or underscores (`_`).
  - Example: `balance`, `transactionFee`, `_internalState`.

- **Case Sensitivity**: 
  - Identifiers in Cadence are **case-sensitive**. For example, `Token` and `token` are treated as distinct identifiers.

- **Valid Characters**:
  - Identifiers can include ASCII letters (`A-Z`, `a-z`), digits (`0-9`), and underscores (`_`).
  - Identifiers cannot contain spaces or special characters like `!`, `@`, `#`, etc.

- **Special Identifiers**:
  - Identifiers starting with an underscore (`_`) are typically used for private or internal entities. Although allowed, their use should follow common programming conventions.

---

### 3. Keywords and Reserved Words

Cadence has a set of **keywords** that are reserved for special syntactic purposes. These keywords cannot be used as identifiers in programs, as they have predefined meanings in the language.

- **Keywords**:
  - `access`, `as`, `attachment`, `auth`, `break`, `case`, `catch`, `const`, `continue`, `contract`, `create`, `default`, `destroy`, `else`, `emit`, `entitlement`, `enum`, `event`, `execute`, `export`, `false`, `final`, `finally`, `for`, `fun`, `goto`, `guard`, `if`, `import`, `in`, `include`, `init`, `interface`, `internal`, `is`, `let`, `mapping`, `move`, `native`, `nil`, `post`, `pre`, `prepare`, `priv`, `pub`, `repeat`, `require`, `requires`, `resource`, `result`, `return`, `self`, `static`, `struct`, `switch`, `throw`, `throws`, `transaction`, `true`, `try`, `typealias`, `var`, `where`, `while`.

- **Reserved Words for Future Use**:
  - Cadence also reserves certain words for potential future extensions of the language. These words cannot currently be used as identifiers and may be introduced in future versions of the language.
  - Example reserved words: TBD.

---

### 4. Literals

Literals in Cadence represent fixed values that can be directly embedded in the source code. They are used to define constant values of various types, including integers, floating-point numbers, strings, and booleans.

- **Integer Literals**:
  - Integer literals represent whole numbers, which can be written in decimal, binary, or hexadecimal formats.
    - **Decimal**: Decimal literals are a `0` or a non-decimal digit (`1`, `2`, `3`, `4`, `5`, `6`, `7`, `8`, `9`) followed by zero or more decimal digits (`0`, `1`, `2`, `3`, `4`, `5`, `6`, `7`, `8`, `9`).
      - Example: `42`, `1000`.
    - **Binary**: Binary literals have the prefix `0b` followed by one or more binary digits (`0`, `1`).
      - Example: `0b1010`, `0b1101`.
    - **Octal**: Octal literals have the prefix `0o` followed by one or more binary digits (`0`, `1`, `2`, `3`, `4`, `5`, `6`, `7`).
      - Example: `0o123`, `0o765`.
    - **Hexadecimal**: Hexadecimal literals have the prefix `0x` followed by one or more hexadecimal digits(`0`, `1`, `2`, `3`, `4`, `5`, `6`, `7`, `8`, `9`, `a`, `b`, `c`, `d`, `e`, `f`, `A`, `B`, `C`, `D`, `E`, `F`).
      _ Example: `0x2A`, `0x3e8`.
  - Underscores (`_`) can be used to separate digits for readability: `1_000`, `0x3E_8`.

- **Fixed-Point Literals**:
  - Fixed-point literals represent real numbers with a decimal point. **TODO** Need more information on fixed-point.
    - Example: `3.14`, `1.23`.

- **Floating-Point Literals**:
  - Fixed-point is not supported by Cadence.

- **String Literals**:
  - String literals are sequences of characters enclosed in double quotes (`"`). 
  - Strings support escape sequences for Unicode scalars and special characters.
    - `\0`: Null character
    - `\\`: Backslash
    - `\t`: Horizontal tab
    - `\n`: Line feed
    - `\r`: Carriage return
    - `\"`: Double quotation mark
    - `\'`: Single quotation mark
    - `\u`: A Unicode scalar value, written as `\u{x}`, where `x` is a 1–8 digit hexadecimal number which needs to be a valid Unicode scalar value, i.e., in the range 0 to 0xD7FF and 0xE000 to 0x10FFFF inclusive
    - Example: `"Hello, Cadence\u{21}"`, `"Multiline\nString"`.

- **Boolean Literals**:
  - Boolean literals represent the two truth values: `true` and `false`.

- **Special Literals**:
  - **Nil Literal**: The literal `nil` represents the absence of a value in Cadence’s optional types.
  - **Resource Literals**: Resources are created and destroyed using specific syntax in Cadence, but the core value does not have a literal representation. Instead, resources are referenced through special constructs like `create` and `destroy`.

- **Array Literals**:
  - **TODO**

- **Dictionary Literals**:
  - **TODO**

---

### 5. Comments

Comments in Cadence are non-executable sections of the code that can be used for documentation, explanation, or temporarily disabling code. Cadence supports two types of comments:

- **Single-line Comments**:
  - Single-line comments start with `//` and continue until the end of the line.
  - Example: 
    ```cadence
    // This is a single-line comment
    let value = 42  // This is also a single-line comment
    ```

- **Multi-line Comments**:
  - Multi-line comments start with `/*` and end with `*/`. They can span multiple lines and can be nested.
  - Example:
    ```cadence
    /* This is a 
       multi-line comment */
    ```

  - **Nested Comments**:
    - Cadence allows multi-line comments to be nested, which means you can comment out blocks of code that contain multi-line comments without causing errors.
    - Example:
      ```cadence
      /* This is a multi-line comment 
         /* This is a nested comment */ 
         Ending the outer comment */
      ```

---

This lexical structure ensures a clear and consistent foundation for writing Cadence programs, supporting both readability and maintainability across a variety of use cases within the Flow blockchain.

## III. Syntax and Grammar

The syntax and grammar of Cadence define the formal structure of programs written in the language. This section outlines the key elements that make up Cadence programs, including how contracts, interfaces, functions, variables, and statements are declared and used, as well as the rules governing expressions and control flow.

---

### 1. Program Structure

A Cadence program is composed of a combination of **contracts**, **interfaces**, **transactions**, and **scripts**. These building blocks allow developers to create decentralized applications that interact with the Flow blockchain.

- **Contracts**: Define the structure and behavior of reusable modules that manage state and execute business logic on the blockchain.
- **Interfaces**: Declare abstract behavior that contracts or other types can implement, supporting both static and dynamic contracts.
- **Transactions**: Define the operations that modify the blockchain state and handle interactions between accounts.
- **Scripts**: Read-only code used to query data from the blockchain without modifying its state.

Each Cadence program must specify its purpose within the structure defined above, allowing for secure interactions with the blockchain.

- **EBNF**:
  ```ebnf
  program
    : ( declaration ';'? )* EOF
    ;
  ```

---

### 2. Declarations

In Cadence, declarations form the backbone of a program, enabling the definition of data structures, resources, contracts, functions, variables, events, and other key components. A declaration defines the type and scope of these components, laying out how they interact with each other within the program. Cadence’s declaration model is designed to be clear and explicit, ensuring that every element in a program is well-typed, properly scoped, and initialized according to strict language rules.

Cadence is a resource-oriented programming language, and the declaration system reflects its focus on security and ownership. Resources, which are unique and must be used exactly once, are declared and managed alongside more traditional constructs like structs and functions. This approach ensures that programs written in Cadence follow safe patterns for managing digital assets, preventing duplication and loss while supporting complex, scalable applications.

The Declarations section introduces the grammar and syntax for various types of declarations in Cadence. From basic data structures such as structs and variables, to more complex entities like contracts and entitlement mappings, this section covers the formal rules for creating and using these elements within the language. Each declaration type is defined using Cadence’s specific Extended Backus-Naur Form (EBNF), providing a precise specification for developers and implementers to follow.

By organizing and defining the fundamental building blocks of Cadence programs, the Declarations section provides the framework necessary to ensure that code is robust, secure, and well-structured.

- **EBNF**:
  ```ebnf
  declaration
      structDeclaration
    | resourceDeclaration
    | contractDeclaration
    | structInterfaceDeclaration
    | resourceInterfaceDeclaration
    | contractInterfaceDeclaration
    | functionDeclaration
    | variableDeclaration
    | eventDeclaration
    | transactionDeclaration
    | pragmaDeclaration
    | entitlementDeclaration
    | entitlementMappingDeclaration
    ;

  ```

#### Struct Declarations

In Cadence, a struct is a composite type that allows the grouping of related data fields into a single entity. Structs are value types, meaning that they are copied when passed or assigned. This behavior makes them ideal for situations where independent copies of a data structure with separate state are required.

Structs in Cadence are declared using the struct keyword, followed by an identifier (the name of the struct), optional conformances to interfaces, and a block containing the struct’s fields and any associated functions. The access level of a struct and its members is controlled by access modifiers, allowing developers to determine the visibility and accessibility of struct elements.

- **Syntax**:
  - The basic syntax for a struct declaration is as follows:
    ```cadence
    access(struct) struct StructName {
      access(struct) let constantField: Type
      access(struct) var variableField: Type
      
      init(parameterName: Type) {
        // Initialization of fields
        self.constantField = parameterValue
        self.variableField = anotherValue
      }
      
      fun someFunction() {
        // Function body
      }
    }

    ```
    - **access**: Specifies the access control for the struct and its members (e.g., pub, access(contract), access(self)).
    - **StructName**: The name of the struct.
    - **Fields**: Declared with the let or var keywords. Constant fields (let) are immutable after initialization, while variable fields (var) can be modified after the struct is created.
    - **Initializer**: The init function is responsible for initializing all fields in the struct. It ensures that each field is assigned a value before the struct is used.
    - **Functions**: Structs can contain functions, which operate on their fields and are defined within the struct body.

- **Example**:
  ```cadence
  pub struct User {
    pub let id: Int
    pub var name: String
    
    init(id: Int, name: String) {
      self.id = id
      self.name = name
    }
     
    pub fun updateName(newName: String) {
      self.name = newName
    }
  }
  ```
  - In this example, the User struct has a constant field id and a variable field name. The initializer sets both fields, and the updateName function allows modification of the name field after the struct is created.

- **EBNF**:
  - The formal EBNF for struct declarations in Cadence is as follows:
    ```ebnf
    structDeclaration
      : access 'struct' identifier conformances? '{' membersAndNestedDeclarations '}'
      ;
    
    membersAndNestedDeclarations
      : (fieldDeclaration | functionDeclaration | structDeclaration)*
      ;
  ```
    - **access**: The access control for the struct (e.g., pub for public, priv for private).
    - **identifier**: The name of the struct.
    - **conformances**: Optional interfaces the struct conforms to.
    - **membersAndNestedDeclarations**: Defines the fields, functions, and nested structs within the struct.


- **Struct Behavior**:
  - **Copying Semantics**: When a struct is assigned to a new variable or passed to a function, a copy of the struct is made. Changes to the copy do not affect the original struct.
    ```cadence
    let user1 = User(id: 1, name: "Alice")
    let user2 = user1  // user1 is copied to user2
    
    user2.updateName(newName: "Bob")
    // user1.name is still "Alice", user2.name is now "Bob"
    ```
  - **Access Control**: The visibility of a struct and its members is controlled using access modifiers. For example, pub makes a struct or field accessible outside its defining contract, while access(contract) restricts visibility to the contract where it is defined.

- **Restrictions**:
  - **Initialization**: All fields must be initialized exactly once in the initializer. Fields cannot be initialized directly in the declaration.
  - **No Inheritance**: Structs in Cadence do not support inheritance. Instead, Cadence encourages the use of composition to build complex types.

By using structs, developers can efficiently model data that needs to be copied and manipulated independently, while also benefiting from Cadence’s strict type system and resource-oriented programming features.

#### Resource Declarations

In Cadence, a resource is a composite type designed to model unique, consumable assets that have strict ownership and usage constraints. Resources are linear types, meaning they must be used exactly once, and they cannot be copied, discarded, or implicitly shared. This behavior is particularly well-suited for managing valuable digital assets, such as tokens or other forms of ownership, where ensuring uniqueness and preventing accidental loss or duplication is critical.

Resources are declared using the resource keyword, followed by an identifier (the name of the resource), optional conformances to interfaces, and a block containing fields and functions that define the resource’s behavior. Like structs, the access level of a resource and its members is controlled by access modifiers, which dictate the visibility and accessibility of the resource’s elements.

- **Syntax**:
  - The basic syntax for a resource declaration is as follows:
    ```cadence
    access(resource) resource ResourceName {
      access(resource) let constantField: Type
      access(resource) var variableField: Type
      
      init(parameterName: Type) {
        // Initialization of fields
        self.constantField = parameterValue
        self.variableField = anotherValue
      }
      
      fun someFunction() {
        // Function body
      }
    }
    ```
    - **access**: Specifies the access control for the resource and its members (e.g., pub, access(contract), access(self)).
    - **ResourceName**: The name of the resource.
    - **Fields**: Declared with the let or var keywords. Constant fields (let) are immutable after initialization, while variable fields (var) can be modified after the resource is created.
    - **Initializer**: The init function initializes all fields in the resource. Every field must be initialized exactly once before the resource can be used.
    - **Functions**: Resources can contain functions, which define operations that can be performed on the resource.

- **Example**:
  ```cadence
  pub resource Vault {
    pub var balance: UFix64
    
    init(balance: UFix64) {
      self.balance = balance
    }
    
    pub fun withdraw(amount: UFix64): @Vault {
      self.balance = self.balance - amount
      return <- create Vault(balance: amount)
    }
    
    pub fun deposit(from: @Vault) {
      self.balance = self.balance + from.balance
      destroy from
    }
   }
  ```

  - In this example, the Vault resource manages a balance of type UFix64. It provides two functions: withdraw, which creates a new Vault resource by deducting an amount, and deposit, which accepts another Vault resource and adds its balance to the current one.

- **EBNF**:
  - The formal EBNF for resource declarations in Cadence is as follows:
    ```ebnf
    resourceDeclaration
      : access 'resource' identifier conformances? '{' membersAndNestedDeclarations '}'
      ;
    
    membersAndNestedDeclarations
      : (fieldDeclaration | functionDeclaration | structDeclaration | resourceDeclaration)*
      ;
    ```
    - **access**: The access control for the resource (e.g., pub for public, access(contract) for contract-level access).
    - **identifier**: The name of the resource.
    - **conformances**: Optional interfaces the resource conforms to.
    - **membersAndNestedDeclarations**: Defines the fields, functions, and any nested types (e.g., structs or resources) within the resource.

- **Resource Behavior**:

  - **Move Semantics**: Resources follow move semantics, meaning they must be explicitly moved or transferred, rather than copied. When a resource is assigned to a new variable, passed to a function, or returned from a function, it is moved to the new location, and the original variable or holder of the resource can no longer access it.
    ```cadence
    let vault <- create Vault(balance: 100.0)
    let newVault <- vault  // vault is moved to newVault, vault can no longer be accessed
    
    newVault.deposit(from: <- create Vault(balance: 50.0))
    ```
  - **Destruction**: Resources must be explicitly destroyed using the destroy keyword when they are no longer needed. This ensures that resources do not leak or persist in a program unintentionally.
    ```cadence
    destroy newVault  // The newVault resource is explicitly destroyed
    ```
  - **Ownership Rules**: Resources are designed to prevent accidental duplication or loss of valuable data. They enforce ownership rules, ensuring that resources can only exist in one place at a time and must be carefully transferred and consumed.

- **Restrictions**:
  - **Nesting of Resources**: A resource can only be nested within another resource, or stored in collections like arrays and dictionaries. Resources cannot be nested within structs, as structs are copied when passed or assigned, which would violate the move semantics of resources.
    ```cadence
    // Invalid: A resource cannot be nested within a struct
    struct InvalidStruct {
      let myResource: @Resource
    }
    ```
  - **Initialization**: Like structs, all fields in a resource must be initialized exactly once in the initializer. It is invalid to provide default values for fields at the point of declaration.
  - **No Inheritance**: Resources do not support inheritance. Instead, Cadence promotes composition and the use of interfaces for code reuse and modular design.

- **Resource Creation**:
  - Resources are created using the create keyword, followed by a constructor call. This ensures that resource instantiation is explicit and controlled:
    ```cadence
    let myVault <- create Vault(balance: 100.0)
    ```
  - Resources must be created in functions or types that are part of the contract where the resource is declared. They cannot be created outside of this context.

- **Resource Destruction**:
  - To prevent resources from lingering in memory and causing potential issues, they must be destroyed when no longer needed using the destroy keyword:
    ```cadence
    destroy myVault  // Properly destroys the resource and frees its memory
    ```
  - If a resource is not explicitly destroyed or moved to another location (e.g., returned from a function), Cadence will produce an error, ensuring that resources are not lost or discarded unintentionally.

Resources are a foundational concept in Cadence, enabling developers to model real-world assets with strict ownership and consumption rules. Their move semantics, strict lifecycle management, and clear access controls make them ideal for building secure, asset-backed systems on blockchain platforms like Flow. By leveraging resources, developers can ensure that assets are used safely and efficiently, while adhering to Cadence’s strong guarantees around correctness and security.

#### Contract Declarations

Contracts in Cadence are central components of smart contracts on the Flow blockchain. A contract defines the fundamental behavior and data for applications, serving as the blueprint for interactions between accounts, resources, and the broader network. Contracts are persistent, meaning that once deployed, they live on the blockchain, enabling users and other contracts to interact with them over time.

Cadence contracts encapsulate code, data, and resources, and provide a secure environment for handling digital assets and executing business logic. Contracts can declare and manage structs, resources, events, and functions, and they control the access and lifecycle of these components.

- **Syntax**:
  - The basic syntax for a contract declaration in Cadence is as follows:
    ```cadence
    access(contract) contract ContractName {
      // Declarations for fields, resources, structs, events, and functions
      
      init() {
        // Initialization of contract-level state or resources
      }
      
      fun someFunction() {
        // Function body
      }
    }
    ```
    - **access**: Specifies the access control for the contract (e.g., pub, access(contract)).
    - **ContractName**: The name of the contract.
    - **Fields**: Declared fields for contract-level state or storage.
    - **Resources/Structs/Events**: Contracts can declare composite types (structs and resources) and events.
    - **Initializer**: The init function is responsible for initializing contract state, resources, or setting up other required elements.
    - **Functions**: Contracts contain functions that define the logic and behavior of the contract.

- **Example**:
  ```cadence
  pub contract TokenContract {
    pub var totalSupply: UFix64
    
    pub event TokensMinted(amount: UFix64)
    
    pub resource Vault {
      pub var balance: UFix64
      
      init(balance: UFix64) {
        self.balance = balance
      }
      
      pub fun deposit(from: @Vault) {
        self.balance = self.balance + from.balance
        destroy from
      }
      
      pub fun withdraw(amount: UFix64): @Vault {
        self.balance = self.balance - amount
        return <- create Vault(balance: amount)
      }
    }
    
    init() {
      self.totalSupply = 0.0
    }
    
    pub fun mintTokens(amount: UFix64): @Vault {
      self.totalSupply = self.totalSupply + amount
      emit TokensMinted(amount: amount)
      return <- create Vault(balance: amount)
    }
  }
  ```
  - In this example, the TokenContract defines a Vault resource to manage token balances and allows minting of tokens through the mintTokens function. The TokensMinted event is emitted whenever new tokens are minted, and the total supply of tokens is tracked as a contract-level field.

- **EBNF**:
  - The formal EBNF for contract declarations in Cadence is as follows:

- **EBNF**:
  ```ebnf
  contractDeclaration
    : access 'contract' identifier conformances? '{' membersAndNestedDeclarations '}'
    ;
  
  membersAndNestedDeclarations
    : (fieldDeclaration | functionDeclaration | structDeclaration | resourceDeclaration | eventDeclaration | contractDeclaration)*
    ;
  ```
  - **access**: The access control for the contract (e.g., pub for public, access(contract) for contract-level access).
  - **identifier**: The name of the contract.
  - **conformances**: Optional interfaces the contract conforms to.
  - **membersAndNestedDeclarations**: Defines the fields, functions, resources, structs, events, and nested contracts within the contract.

- **Contract Behavior**:

  - **Persistent State**: Contracts, once deployed, are persistent. They maintain their state on the blockchain, and their fields and resources can evolve through transactions and interactions. For instance, contract fields like totalSupply in the example above can change over time as functions are executed.
  - **Access Control**: Contracts can declare fields, functions, resources, and other components with specific access levels. For example, pub allows external access to a function or field, while access(contract) restricts access to within the contract. This ensures that sensitive data and logic are protected from unauthorized access.

- **Initializers in Contracts**:

  - Contracts can have an initializer (init) that runs once, during the deployment of the contract. The initializer is responsible for setting up the contract’s initial state, such as creating resources, initializing variables, or emitting events.
    ```cadence
    pub contract MyContract {
      pub var owner: Address
      
      init(owner: Address) {
        self.owner = owner
      }
    }
    ```
    - In the example above, the contract’s initializer sets the owner field to the address provided during deployment.

- **Nested Types and Resources**:

  - Contracts can declare and manage composite types such as structs, resources, and events within their scope. These types can be instantiated, managed, and interacted with through the contract’s functions.

    - **Resources**: Contracts frequently manage resources that represent valuable or unique digital assets, such as tokens. The lifecycle of these resources is controlled by the contract through creation, movement, and destruction.
      ```cadence
      pub contract AssetManager {
        pub resource Asset {
          pub let id: Int
          
          init(id: Int) {
            self.id = id
          }
        }
        
        pub fun createAsset(id: Int): @Asset {
          return <- create Asset(id: id)
        }
      }
      ```
    - **Structs**: Contracts can also define structs for organizing data that doesn’t require the ownership semantics of resources.
      ```cadence
      pub contract Company {
        pub struct Employee {
          pub let name: String
          pub let id: Int
          
          init(name: String, id: Int) {
            self.name = name
            self.id = id
          }
        }
      }
      ```
    - **Events**: Contracts can declare events that notify external systems when significant state changes occur within the contract, such as token transfers or contract updates.

- **Restrictions**:

  - **No Inheritance**: Contracts in Cadence do not support inheritance. Instead, contracts are designed to be self-contained, promoting composition through interfaces and nested types rather than traditional class-based inheritance.
  - **Ownership of Resources**: Resources declared within a contract are tied to that contract, and their lifecycle (creation, movement, and destruction) is managed entirely within the contract. Resources cannot be copied or implicitly shared, ensuring strict control over valuable assets.

- **Contract Interaction**:

  - Contracts define the core logic of decentralized applications on the blockchain. Users and other contracts can interact with a deployed contract by calling its public functions, transferring resources, and listening for events.
    ```cadence
    // Example of interacting with a contract's function
    let myVault <- TokenContract.mintTokens(amount: 100.0)
    
    // Example of listening for a contract's event
    TokenContract.TokensMinted.subscribe { (amount: UFix64) in
      log("Tokens minted: \(amount)")
    }
    ```

Contracts are at the heart of Cadence’s resource-oriented programming model. They define the rules and operations for managing state and resources, providing a secure, transparent, and modular foundation for decentralized applications. By utilizing contracts, developers can create robust, asset-backed systems on Flow that enforce strict ownership, manage digital assets, and facilitate interactions between users and other contracts.

#### Struct Interface Declarations

In Cadence, a struct interface defines a set of behaviors that a struct must implement. Struct interfaces allow for the specification of functionality without providing the actual implementation. This enables a flexible design pattern where different structs can conform to the same interface, ensuring that they provide the required methods and properties while allowing each struct to implement them differently.

A struct interface in Cadence is declared using the struct interface keyword and includes a list of required functions and fields. Any struct that conforms to the interface must implement all the functions and properties defined in the interface.

- **Syntax**:

  - The basic syntax for a struct interface declaration is as follows:
    ```cadence
    access(struct interface) struct interface InterfaceName {
      // Required field declarations
      access(interface) let fieldName: Type
      
      // Required function declarations
      fun requiredFunction(parameterName: Type): ReturnType
    }
    ```
    - **access**: Specifies the access control for the struct interface (e.g., pub, access(contract)).
    - **InterfaceName**: The name of the struct interface.
    - **Fields**: Declared fields that must be implemented by conforming structs. These fields act as a contract that the struct must fulfill.
    - **Functions**: Function signatures that the struct must implement.

- **Example**:
  ```cadence
  pub struct interface BalanceInterface {
    pub let balance: UFix64
    
    pub fun getBalance(): UFix64
  }
  ```
  - In this example, the BalanceInterface struct interface defines a required field balance of type UFix64 and a required function getBalance that returns the current balance. Any struct conforming to this interface must implement both the field and the function.
  - A struct conforming to this interface might look like the following:
    ```cadence
    pub struct Wallet: BalanceInterface {
      pub let balance: UFix64
      
      init(balance: UFix64) {
        self.balance = balance
      }
      
      pub fun getBalance(): UFix64 {
        return self.balance
      }
    }
    ```
    - In this example, the Wallet struct conforms to the BalanceInterface by implementing the required balance field and getBalance function.

- **EBNF**:
  - The formal EBNF for struct interface declarations in Cadence is as follows:
  ```ebnf
  structInterfaceDeclaration
    : access 'struct' 'interface' identifier '{' membersAndNestedDeclarations '}'
    ;
  
  membersAndNestedDeclarations
    : (fieldDeclaration | functionDeclaration)*
    ;
  ```
    - **access**: The access control for the struct interface (e.g., pub for public access, access(contract) for contract-level access).
    - **identifier**: The name of the struct interface.
    - **membersAndNestedDeclarations**: Specifies the required fields and function declarations that structs conforming to the interface must implement.

- **Struct Interface Conformance**:
  - A struct conforms to a struct interface by using the colon (:) symbol followed by the interface name. The struct must then implement all the required fields and functions of the interface. If the struct does not fulfill these requirements, a compile-time error will occur.
    ```cadence
    pub struct MyStruct: SomeInterface {
      // Implementation of fields and functions required by the interface
    }
    ```
  - Multiple interfaces can be implemented by a single struct by separating them with commas.
    ```cadence
    pub struct MyStruct: InterfaceOne, InterfaceTwo {
      // Implementation of fields and functions required by both interfaces
    }
    ```

- **Behavior of Struct Interfaces**:
  - **Polymorphism**: Struct interfaces allow for polymorphic behavior in Cadence, where multiple structs can conform to the same interface. This enables different structs to be treated uniformly when interacting with their shared interface, promoting flexibility in design.
  - **Access Control**: The visibility of a struct interface and its members is governed by access modifiers. Public interfaces (pub struct interface) are accessible outside their defining contract or scope, whereas contract-level interfaces (access(contract) struct interface) are restricted to the contract in which they are declared.

- **Restrictions**:
  - **No Fields with Initializers*: Struct interfaces can declare fields, but these fields cannot be initialized within the interface. The conforming struct is responsible for providing the implementation and initialization of the fields.
    ```cadence
    pub struct interface ExampleInterface {
      pub let id: Int   // This field must be implemented by conforming structs
    }
    ```
  - **No Function Bodies**: Functions within a struct interface cannot have bodies. Only the function signature (name, parameters, and return type) is provided. The conforming struct must provide the actual implementation of these functions.
    ```cadence
    pub struct interface ExampleInterface {
      fun doSomething(): Void   // No function body
    }
    ```

- **Example: Struct Interface for a Token Holder**
  - Here’s an example of how struct interfaces might be used in a contract to enforce a common behavior across different types of token holders:
    ```cadence
    pub struct interface TokenHolderInterface {
      pub let balance: UFix64
      
      pub fun getBalance(): UFix64
      pub fun deposit(amount: UFix64)
    }
    
    pub struct Account: TokenHolderInterface {
      pub var balance: UFix64
      
      init(balance: UFix64) {
        self.balance = balance
      }
      
      pub fun getBalance(): UFix64 {
        return self.balance
      }
      
      pub fun deposit(amount: UFix64) {
        self.balance = self.balance + amount
      }
    }
    ```
    - In this example, the TokenHolderInterface ensures that any struct conforming to it provides a balance field, as well as getBalance and deposit functions. The Account struct implements the interface, fulfilling the contract and providing concrete functionality for managing a token balance.

Struct interfaces in Cadence provide a flexible way to define shared behaviors across multiple structs while maintaining separate implementations. By using struct interfaces, developers can enforce consistent behaviors while still allowing for specific implementations tailored to different use cases. This design pattern supports better code modularity, promotes reusability, and encourages adherence to well-defined contracts between different components in a Cadence program.

#### Resource Interface Declarations

In Cadence, a resource interface defines a set of requirements that a resource must fulfill. Resource interfaces specify the structure and behavior that any conforming resource must implement, such as required fields and functions, without providing the actual implementation. This allows developers to define reusable, abstract behaviors that can be shared across different resources while enforcing the strict ownership and usage constraints that are characteristic of resources in Cadence.

A resource interface is declared using the resource interface keyword, followed by the interface’s name and a block that contains the field and function declarations. Any resource that conforms to the interface must implement all the fields and functions defined in the interface, following the rules of Cadence’s linear types.

- **Syntax**:
  - The basic syntax for a resource interface declaration is as follows:
    ```cadence
    access(resource interface) resource interface InterfaceName {
      // Required field declarations
      access(interface) let fieldName: Type
      
      // Required function declarations
      fun requiredFunction(parameterName: Type): ReturnType
    }
    ```
    - **access**: Specifies the access control for the resource interface (e.g., pub, access(contract)).
    - **InterfaceName**: The name of the resource interface.
    - **Fields**: Declared fields that must be implemented by any resource conforming to the interface.
    - **Functions**: Function signatures that the resource must implement.

- **Example**:
  ```cadence
  pub resource interface VaultInterface {
    pub let balance: UFix64
    
    pub fun deposit(amount: UFix64)
    pub fun withdraw(amount: UFix64): @VaultInterface
  }
  ```
  - In this example, the VaultInterface defines a required balance field and two required functions: deposit, which accepts an amount to increase the balance, and withdraw, which removes an amount from the balance and returns a new resource of the same type.
  - A resource conforming to this interface might look like the following:
    ```cadence
    pub resource Vault: VaultInterface {
      pub var balance: UFix64
      
      init(balance: UFix64) {
        self.balance = balance
      }
      
      pub fun deposit(amount: UFix64) {
        self.balance = self.balance + amount
      }
      
      pub fun withdraw(amount: UFix64): @Vault {
        self.balance = self.balance - amount
        return <- create Vault(balance: amount)
      }
    }
    ```
    - In this example, the Vault resource conforms to the VaultInterface by implementing the required balance field and the deposit and withdraw functions.

- **EBNF**:
  - The formal EBNF for resource interface declarations in Cadence is as follows:
    ```ebnf
    resourceInterfaceDeclaration
      : access 'resource' 'interface' identifier '{' membersAndNestedDeclarations '}'
      ;
    
    membersAndNestedDeclarations
      : (fieldDeclaration | functionDeclaration)*
      ;
    ```
    - **access**: The access control for the resource interface (e.g., pub for public access, access(contract) for contract-level access).
    - **identifier**: The name of the resource interface.
    - **membersAndNestedDeclarations**: Specifies the required fields and function declarations that resources conforming to the interface must implement.

- **Resource Interface Conformance**:
  - A resource conforms to a resource interface by using the colon (:) symbol followed by the interface name. The resource must implement all the required fields and functions specified in the interface.
    ```cadence
    pub resource MyResource: SomeResourceInterface {
      // Implementation of fields and functions required by the interface
    }
    ```
  - A resource can conform to multiple interfaces by separating the interface names with commas.
    ```cadence
    pub resource MyResource: InterfaceOne, InterfaceTwo {
      // Implementation of fields and functions required by both interfaces
    }
    ```

- **Behavior of Resource Interfaces**:
  - **Polymorphism**: Resource interfaces enable polymorphism in Cadence, allowing different resources to conform to the same interface. This enables generic handling of different resource types that share the same interface, which can improve code flexibility and modularity.
  - **Move Semantics**: Even though resources must conform to the behaviors defined by the interface, they still retain their linear type properties. Resources cannot be copied or implicitly shared. They must be explicitly moved (using the <- operator) or destroyed when no longer needed.

- **Restrictions**:
  - **No Field Initializers**: Resource interfaces can declare fields, but the fields cannot have initial values. It is the responsibility of the conforming resource to initialize these fields.
    ```cadence
    pub resource interface ExampleResourceInterface {
      pub let id: Int   // This field must be implemented and initialized by conforming resources
    }
    ```
  - **No Function Bodies**: Resource interfaces only declare function signatures; they cannot provide any function implementations. The conforming resource must implement these functions.
    ```cadence
    pub resource interface ExampleResourceInterface {
      fun performAction(): Void   // No function body
    }
    ```

- **Example: Resource Interface for a Token Vault**
  - Here’s an example of how resource interfaces might be used in a contract to enforce a common set of behaviors across different types of token vaults:
    ```cadence
    pub resource interface VaultInterface {
      pub let balance: UFix64
      
      pub fun deposit(amount: UFix64)
      pub fun withdraw(amount: UFix64): @VaultInterface
    }
    
    pub resource Vault: VaultInterface {
      pub var balance: UFix64
      
      init(balance: UFix64) {
        self.balance = balance
      }
      
      pub fun deposit(amount: UFix64) {
        self.balance = self.balance + amount
      }
      
      pub fun withdraw(amount: UFix64): @Vault {
        self.balance = self.balance - amount
        return <- create Vault(balance: amount)
      }
    }
    ```
    - In this example, the VaultInterface ensures that any resource conforming to it provides a balance field, as well as deposit and withdraw functions. The Vault resource implements the interface by managing a token balance and providing methods to deposit and withdraw tokens, while maintaining ownership rules and move semantics.

- **Usage with Resource Interfaces**:
  - Resource interfaces allow for flexible and modular resource management, particularly when used in conjunction with generic programming techniques. For instance, a function can accept any resource that conforms to a resource interface, enabling different resource types to be treated uniformly as long as they share the same interface.
    ```cadence
    pub fun transfer(from: &{VaultInterface}, to: &{VaultInterface}, amount: UFix64) {
      let withdrawn <- from.withdraw(amount: amount)
      to.deposit(amount: withdrawn.balance)
      destroy withdrawn
    }
    ```
    - In this example, the transfer function operates on any resources that conform to the VaultInterface, regardless of the specific type of the resource. This promotes code reusability and abstraction without sacrificing the strict resource handling rules of Cadence.

Resource interfaces are a powerful feature in Cadence that enable developers to define common behaviors and enforce uniformity across different resource types. By using resource interfaces, developers can create flexible, reusable abstractions while still adhering to Cadence’s resource-oriented principles of ownership, security, and explicit resource management. These interfaces allow for polymorphic handling of resources and provide a mechanism for ensuring that resources follow a defined set of rules, making them essential for building secure and modular smart contracts on Flow.

#### Contract Interface Declarations

In Cadence, a contract interface defines a set of behaviors and expectations that any contract implementing the interface must fulfill. Contract interfaces provide a way to specify required fields, functions, and events that a contract must implement without detailing the actual implementations. This allows for the creation of abstract contract definitions that can be shared, reused, and implemented by different contracts, ensuring consistency across multiple implementations.

A contract interface is declared using the contract interface keyword, followed by the interface’s name and a block containing the required field, function, and event declarations. Any contract conforming to the interface must implement all the declared elements.

- **Syntax**:
  - The basic syntax for a contract interface declaration is as follows:
    ```cadence
    access(contract interface) contract interface InterfaceName {
      // Required field declarations
      access(interface) let fieldName: Type
      
      // Required function declarations
      fun requiredFunction(parameterName: Type): ReturnType
      
      // Required event declarations
      event EventName(parameterName: Type)
    }
    ```
    - **access**: Specifies the access control for the contract interface (e.g., pub, access(contract)).
    - **InterfaceName**: The name of the contract interface.
    - **Fields**: Declared fields that must be implemented by any contract conforming to the interface.
    - **Functions**: Function signatures that the contract must implement.
    - **Events**: Declared events that the contract must emit as part of its behavior.

- **Example**:
  ```cadence
  pub contract interface TokenInterface {
    pub let totalSupply: UFix64
    
    pub fun mintTokens(amount: UFix64)
    pub fun getBalance(account: Address): UFix64
    
    pub event TokensMinted(amount: UFix64)
  }
  ```
  - In this example, the TokenInterface defines a required totalSupply field and two required functions: mintTokens, which mints new tokens, and getBalance, which returns the token balance for a given account. The interface also requires the contract to emit the TokensMinted event whenever new tokens are minted.
  - A contract conforming to this interface might look like the following:
    ```cadence
    pub contract Token: TokenInterface {
      pub var totalSupply: UFix64
      pub var balances: @{Address: UFix64}
      
      init() {
        self.totalSupply = 0.0
        self.balances <- {}
      }
      
      pub fun mintTokens(amount: UFix64) {
        self.totalSupply = self.totalSupply + amount
        emit TokensMinted(amount: amount)
      }
      
      pub fun getBalance(account: Address): UFix64 {
        return self.balances[account] ?? 0.0
      }
    }
    ```
    - In this example, the Token contract conforms to the TokenInterface by implementing the required totalSupply field, mintTokens and getBalance functions, and emitting the TokensMinted event.

- **EBNF**:

  - The formal EBNF for contract interface declarations in Cadence is as follows:
    ```ebnf
    contractInterfaceDeclaration
      : access 'contract' 'interface' identifier '{' membersAndNestedDeclarations '}'
      ;
    
    membersAndNestedDeclarations
      : (fieldDeclaration | functionDeclaration | eventDeclaration)*
      ;
    ```
    - **access**: The access control for the contract interface (e.g., pub for public access, access(contract) for contract-level access).
    - **identifier**: The name of the contract interface.
    - **membersAndNestedDeclarations**: Specifies the required fields, function declarations, and event declarations that contracts conforming to the interface must implement.

- **Contract Interface Conformance**:
  - A contract conforms to a contract interface by using the colon (:) symbol followed by the interface name. The contract must then implement all the required fields, functions, and events of the interface. If a contract does not implement all the required members, a compile-time error will occur.
  ```cadence
  pub contract MyContract: SomeContractInterface {
    // Implementation of fields, functions, and events required by the interface
  }
  ```
  - Multiple interfaces can be implemented by a single contract by separating them with commas.
  ```cadence
  pub contract MyContract: InterfaceOne, InterfaceTwo {
    // Implementation of fields, functions, and events required by both interfaces
  }
  ```

- **Behavior of Contract Interfaces**:
  - **Polymorphism**: Contract interfaces enable polymorphic behavior, allowing multiple contracts to conform to the same interface while providing their own implementations. This makes it easier to create flexible and interchangeable components that adhere to a common contract.
  - **Access Control**: The visibility of contract interfaces and their members is governed by access modifiers. Public interfaces (pub contract interface) can be accessed and implemented by any contract, whereas contract-level interfaces (access(contract) contract interface) are restricted to the contract in which they are declared.

- **Restrictions**:
  - **No Initializers**: Contract interfaces cannot include initializers (init functions) since they are abstract definitions. Initializers must be defined in the conforming contracts themselves.
    ```cadence
    pub contract interface ExampleContractInterface {
      pub let id: Int   // This field must be implemented by the conforming contract
    }
    ```
  - **No Function Bodies**: Contract interfaces only declare function signatures without providing implementations. Conforming contracts are responsible for implementing the functions.
    ```cadence
    pub contract interface ExampleContractInterface {
      fun performAction(): Void   // No function body
    }
    ```
  - **No Resource or Struct Definitions**: Contract interfaces do not define resources or structs directly. Instead, the conforming contracts can define these types as part of their implementation. However, a contract interface can enforce the existence of certain functions and events related to resource management or behavior.

- **Example: Contract Interface for a Token Contract**

  - Here’s an example of how contract interfaces might be used to define a common set of behaviors across different types of token contracts:
    ```cadence
    pub contract interface TokenContractInterface {
      pub let totalSupply: UFix64
      
      pub fun mintTokens(amount: UFix64)
      pub fun getBalance(account: Address): UFix64
      pub fun transfer(from: Address, to: Address, amount: UFix64)
      
      pub event TokensMinted(amount: UFix64)
      pub event Transfer(from: Address, to: Address, amount: UFix64)
    }
    
    pub contract MyToken: TokenContractInterface {
      pub var totalSupply: UFix64
      pub var balances: @{Address: UFix64}
      
      init() {
        self.totalSupply = 0.0
        self.balances <- {}
      }
      
      pub fun mintTokens(amount: UFix64) {
        self.totalSupply = self.totalSupply + amount
        emit TokensMinted(amount: amount)
      }
      
      pub fun getBalance(account: Address): UFix64 {
        return self.balances[account] ?? 0.0
      }
      
      pub fun transfer(from: Address, to: Address, amount: UFix64) {
        let fromBalance = self.balances[from] ?? 0.0
        let toBalance = self.balances[to] ?? 0.0
        self.balances[from] = fromBalance - amount
        self.balances[to] = toBalance + amount
        emit Transfer(from: from, to: to, amount: amount)
      }
    }
    ```
    - In this example, the TokenContractInterface defines a standard set of behaviors for a token contract, including minting tokens, transferring tokens between accounts, and emitting events to signal these actions. The MyToken contract implements this interface, providing the required functionality for minting, balance management, and transfers.

- **Usage with Contract Interfaces**:

  - Contract interfaces can be used to ensure that different contracts adhere to a common set of rules and functionality. This is especially useful in decentralized ecosystems where interoperability and modularity are critical. Developers can define contract interfaces that specify required behaviors, and any contract that conforms to the interface can be used interchangeably within the system.
    ```cadence
    pub fun performTokenTransfer(token: &{TokenContractInterface}, from: Address, to: Address, amount: UFix64) {
      token.transfer(from: from, to: to, amount: amount)
    }
    ```
    - In this example, the performTokenTransfer function operates on any contract that conforms to the TokenContractInterface, making the function flexible and reusable across different implementations of token contracts.

Contract interfaces are a powerful tool in Cadence that allow developers to define abstract behaviors and enforce consistency across different contract implementations. By leveraging contract interfaces, developers can create reusable, interoperable components while ensuring that key behaviors, such as token management or asset transfer, are implemented uniformly across different contracts. This approach promotes modularity, security, and flexibility in building decentralized applications on Flow.

#### Function Declarations

In Cadence, functions are the fundamental building blocks of behavior in a program. A function is a sequence of statements that perform a specific task, optionally taking inputs (parameters) and returning an output (return value). Functions in Cadence are typed, meaning they have a defined set of input types (parameter types) and an output type (return type).

Functions are first-class citizens in Cadence, meaning they can be assigned to variables, passed as arguments, and returned from other functions. This gives developers flexibility in structuring code and allows for the creation of reusable, modular components.

- **Syntax**:
  - A function declaration in Cadence begins with the fun keyword, followed by the function name, a list of parameters, an optional return type, and the function body.
    ```cadence
    access fun functionName(parameterName: Type): ReturnType {
      // Function body
    }
    ```
    - **access**: Optional access control modifier (e.g., pub, access(contract)).
    - **functionName**: The name of the function, which is used to invoke it.
    - **parameterName**: Each parameter is defined with a name and a type annotation.
    - **ReturnType**: The type of the value the function returns. If there is no return type, Void is implied.
    - **Function** Body: The code block containing the statements that define the function’s behavior, enclosed in curly braces ({}).

- **Example**:
  ```cadence
  // Declare a function named `add` that takes two integers and returns their sum
  pub fun add(a: Int, b: Int): Int {
    return a + b
  }
  ```
  - In this example, the add function takes two integer parameters (a and b) and returns an integer representing their sum.

- **Parameters and Argument Labels**:
  - Each parameter in a function declaration must have a name and a type annotation, specified in the format parameterName: Type. Additionally, argument labels can be provided to clarify the meaning of each parameter during function calls. Argument labels improve code readability, especially when multiple parameters have the same type.
  - The argument label precedes the parameter name in the declaration. The special argument label _ allows the label to be omitted in function calls.
    ```cadence
    // Declare a function that requires argument labels for the second and third parameters
    pub fun clamp(_ value: Int, min: Int, max: Int): Int {
      if value > max {
        return max
      }
      if value < min {
        return min
      }
      return value
    }
    
    // Function call with omitted argument label for the first parameter
    let clamped = clamp(150, min: 0, max: 100)  // clamped is 100
    ```
    - In the above example, the first argument has the special label _, so no label is required when calling the function. The second and third arguments use the labels min and max, making the function call more readable.

- **Return Type**:
  - A function may return a value. The return type is specified after the parameter list, following a colon (:). If a function does not return any value, the return type is omitted, and Void is implied.
  ```cadence
  pub fun greet(): Void {
    log("Hello, Flow!")
  }
  ```

- **Argument Passing and Behavior**:
  - When passing arguments to a function in Cadence, the values are copied. This means that the original values in the caller’s scope are not affected by any changes made to the parameters within the function.
  ```cadence
  fun increment(_ x: Int): Int {
    return x + 1
  }
  
  let a = 5
  let result = increment(a)  // result is 6, but `a` remains 5
  ```

- **Nested Functions**:
  - Functions in Cadence can be nested inside other functions. Nested functions are only accessible within the scope of the outer function.
  ```cadence
  fun outerFunction(x: Int): Int {
    fun innerFunction(y: Int): Int {
      return y * 2
    }
    return innerFunction(x) + 1
  }
  
  outerFunction(3)  // is 7
  ```

- **EBNF**:
  ```ebnf
  functionDeclaration
    : access 'fun' identifier parameterList ( ':' typeAnnotation )? functionBlock?
    ;
  ```

- **Restrictions**:
  - **No Overloading**: Cadence does not support function overloading, meaning that there cannot be two functions with the same name but different parameter types.
  - **No Default Parameters or Variadic Functions**: Cadence does not support optional parameters with default values or variadic functions that accept a variable number of arguments.

- **Function Types**:
  - Functions in Cadence have types, represented by the fun keyword followed by the parameter types in parentheses, a colon, and the return type.
  ```cadence
  let add: fun(Int, Int): Int = fun (a: Int, b: Int): Int {
    return a + b
  }
  ```

- **Example: Function with Argument Labels and Return Type**
  ```cadence
  pub fun send(from sender: Address, to receiver: Address, amount: UFix64) {
    // Function body to handle token transfer
  }
  ```
  - In this example, the send function takes three parameters: sender, receiver, and amount. The labels from and to improve clarity when calling the function:
  ```cadence
  send(from: senderAddress, to: receiverAddress, amount: 100.0)
  ```

Function declarations in Cadence provide a flexible way to encapsulate behavior in reusable, well-typed units. With support for argument labels, nested functions, and first-class function types, developers can write modular and readable code while ensuring type safety and predictable behavior.

#### Variable Declarations

In Cadence, variables and constants allow for the storage and manipulation of values. A variable declaration binds a value to an identifier, and depending on the type of declaration, this value may either be mutable or immutable. Constants (declared using let) are immutable once initialized, while variables (declared using var) allow for reassignment of their value after initialization.

Variable declarations can be made within any scope, including globally, and both constants and variables must be initialized upon declaration. Cadence does not permit uninitialized variables or constants, and each identifier must be unique within the same scope.

- **Syntax**:
  - The basic syntax for a variable declaration consists of an access modifier (optional), the declaration kind (let or var), the identifier, an optional type annotation, and an initial value. If a type annotation is not provided, the type is inferred from the initial value.

- **EBNF**:
  ```ebnf
  variableDeclaration
    : access variableKind identifier ( ':' typeAnnotation )? transfer expression ( transfer expression )?
    ;
  
  variableKind
    : 'let'
    | 'var'
    ;
  ```
  - Components:
    - **access**: Specifies the visibility level of the variable (e.g., pub, access(contract)). It is optional.
    - **variableKind**: Specifies whether the declaration is a constant (let) or a variable (var).
    - **identifier**: The name of the variable or constant.
    - **typeAnnotation** (optional): Specifies the type of the variable. If omitted, the type is inferred.
    - **expression**: The initial value assigned to the variable or constant.
    - **transfer** expression: Handles the assignment or movement of values, specifically relevant for resource types.

- **Example**:
  ```cadence
  // Declare a constant named `a` of type Int
  pub let a: Int = 10
  
  // Declare a variable named `b`, with type inference for the initial value
  var b = 20
  
  // Reassign a new value to variable `b`
  b = 30
  ```

- **Constants vs. Variables**:
  - **Constants (let)**: Once a constant is initialized with a value, it cannot be reassigned. However, if the constant points to a mutable object (e.g., an array or resource), the object’s contents can be modified.
    ```cadence
    let a = 10
    a = 15  // Invalid: Constants cannot be reassigned
    ```
  - **Variables (var)**: Variables allow for reassignment, meaning the identifier can be bound to a different value after initialization.
    ```cadence
    var b = 20
    b = 30  // Valid: Variables can be reassigned
    ```

- **Initialization Requirement**:
  - All variable declarations in Cadence must be initialized with a value at the time of declaration. It is invalid to declare a variable or constant without an initial value.
    ```cadence
   // Invalid: Declaring a constant without initialization
   let x  // This will result in an error
    ```

- **Type Inference and Type Annotations**:
  - Cadence allows for type inference, meaning that the type of a variable or constant can be automatically determined based on the initial value. However, you can also explicitly specify a type annotation to declare the type of the variable or constant.
    ```cadence
    // Type inference: Cadence infers that `a` is of type Int
    let a = 10
    
    // Explicit type annotation: `b` is explicitly declared as Int
    let b: Int = 20
    ```

- **Unique Identifiers**:
  - Each identifier in a scope must be unique. Declaring another variable or constant with the same name within the same scope is not allowed, regardless of the declaration kind or type.
    ```cadence
    let x = 10
    
    // Invalid: Redeclaring a constant with the same name in the same scope
    let x = 20  // This will result in an error
    
    var y = 5
    
    // Invalid: Redeclaring a variable with the same name in the same scope
    var y = 10  // This will result in an error
    ```

- **Redeclaration in Sub-scopes**:
  - While it is illegal to redeclare an identifier in the same scope, Cadence allows redeclaration of identifiers in sub-scopes. This allows for reuse of the same identifier name within nested blocks without affecting the outer scope.
    ```cadence
    let a = 10
    
    if true {
      // Redeclaration of `a` in a sub-scope
      let a = 20  // This is valid within this block
    }
    
    // Outside the block, `a` is still 10
    ```

- **Self-referencing Declarations**:
  - Variables or constants cannot be initialized using themselves in their own initial value. This results in a circular reference, which is invalid.
    ```cadence
    // Invalid: Self-referencing declaration
    let a = a  // This will result in an error
    ```

Cadence variable declarations enforce strict rules to ensure clarity and safety in the use of constants and variables. By requiring initialization, enforcing unique identifiers, and distinguishing between mutable (var) and immutable (let) bindings, Cadence supports a robust and secure environment for handling on-chain values and resources.

#### Event Declarations

In Cadence, events are special values emitted during the execution of a program to signal significant occurrences, such as state changes or actions performed within a contract. Events allow external systems, such as clients and applications, to listen to and react to these changes. Events contain named parameters that hold the data associated with the event. Events can only be declared within a contract body and cannot be declared globally, nor inside resources or structures.

- **Syntax**:
  - An event declaration in Cadence uses the event keyword, followed by the event name and a list of parameters enclosed in parentheses. Each parameter must have a name and a type. The syntax is similar to that of function declarations but without a return type.
    ```cadence
    access event EventName(parameterName: Type, anotherParameter: Type)
    ```
    - **access**: Specifies the access control for the event (e.g., pub for public access). This is optional.
    - **event**: The keyword used to declare an event.
    - **EventName**: The name of the event.
    - **parameterName**: The name of the parameter that will store event data.
    - **Type**: The data type of the parameter.

- **EBNF**:
  - The formal EBNF for event declarations is as follows:
    ```ebnf
    eventDeclaration
      : access 'event' identifier parameterList
      ;
    ```
    - **access**: Specifies the visibility level of the event (e.g., pub).
    - **identifier**: The name of the event.
    - **parameterList**: A list of parameters that define the data emitted with the event.

- **Example**:
  ```cadence
  pub contract MyContract {
    // Declare an event with three parameters: from, to, and amount
    pub event TransferEvent(from: Address, to: Address, amount: UFix64)
    
    pub fun transferTokens(from: Address, to: Address, amount: UFix64) {
      // Emit the TransferEvent when tokens are transferred
       emit TransferEvent(from: from, to: to, amount: amount)
    }
  }
  ```
  - In this example, the TransferEvent event is declared with three parameters (from, to, and amount). The event is emitted using the emit statement within the transferTokens function to signal that tokens have been transferred.

- **Valid Event Parameter Types**:
  - Events can only contain parameters with valid event parameter types. These types ensure that the data associated with the event can be safely emitted without risk of data loss or unintended behavior.
    - **Valid types include**:
      - **Primitive types**: Bool, String, Int, UInt, UFix64, and other numeric types.
      - **Arrays and dictionaries**: Arrays and dictionaries of valid primitive types.
      - **Structures**: Structs where all fields are of valid event parameter types.
    - **Invalid types**:
      - **Resources**: Resource types cannot be used as event parameters because they would be moved when emitted, leading to potential loss of resources.
        ```cadence
        pub contract MyContract {
          // Valid event declaration
          pub event ValidEvent(message: String, count: Int)
          
          // Invalid event: Resources cannot be event parameters
          pub event InvalidEvent(resourceField: @Vault)  // This will result in an error
        }
        ```

- **Emitting Events**:
  - To emit an event, the emit statement is used. Events can only be emitted from within the contract in which they are declared.
    ```cadence
    pub contract MyContract {
      pub event ActionOccurred(message: String)
      
      pub fun triggerEvent() {
        // Emit the ActionOccurred event
        emit ActionOccurred(message: "The action has taken place")
      }
    }
    ```

- **Restrictions on Event Emission**:
  - **Emit Only**: Events can only be invoked using the emit statement. They cannot be assigned to variables or used as parameters for other functions or events.
    ```cadence
    // Invalid: Events cannot be assigned to variables
    let eventInstance = emit MyEvent()  // This will result in an error
    ```
  - **Declared in Contract**: Events must be declared within a contract body. They cannot be declared globally, or within structs or resources.
    ```cadence
    pub contract MyContract {
      pub event ValidEvent()
      
      // Invalid: Events cannot be declared globally
      event GlobalEvent()  // This will result in an error
    }
    ```
  - **Emit Within Declaring Contract**: An event can only be emitted from the contract in which it is declared. This ensures that events remain tightly scoped to the contract logic they are related to.

Events in Cadence are a powerful mechanism for signaling changes and actions within a contract. By emitting events, contracts can communicate with external systems in a secure and structured way. With restrictions on event types and controlled emission through the emit statement, Cadence ensures that events are used safely and effectively within decentralized applications.

#### Transaction Declarations

In Cadence, transactions are objects that allow interaction with the Flow blockchain by performing state changes, such as transferring tokens, modifying resources, or interacting with smart contracts. A transaction is signed by one or more accounts and submitted to the network for execution. The transaction contains multiple phases that provide structure and safety to state modifications: prepare, pre-conditions, execute, and post-conditions.

- **Syntax**:
  - A transaction is declared using the transaction keyword, followed by an optional parameter list and four optional phases: prepare, pre, execute, and post. Each phase serves a specific purpose in the transaction lifecycle.
    ```cadence
    transaction(parameterList?) {
      prepare(signerList?) {
        // Preparation logic
      }
      
      pre {
        // Pre-conditions
      }
      
      execute {
        // Main transaction logic
      }
      
      post {
        // Post-conditions
      }
    }
    ```
    - **parameterList**: Optional list of parameters passed to the transaction when invoked.
    - **prepare**: Phase for accessing and modifying signing accounts.
    - **pre**: Optional block that checks pre-conditions before execution.
    - **execute**: Main phase where the transaction logic is executed.
    - **post**: Optional block that checks post-conditions after execution.

- **EBNF**:
  - The formal EBNF for transaction declarations is as follows:
    ```ebnf
    transactionDeclaration
      : 'transation' parameterList? '{' fields prepare? preConditions? executePostConditions '}'
      ;
    
    prepare
      : identifier parameterList functionBlock?
      ;
    
    preConditions
      : 'pre' '{' conditions '}'
      ;
    
    executePostConditions
      : execute
      | execute postConditions
      | postConditions
      | postConditions execute
      | (* no execute or postConditions *)
      ;

   execute
      : 'execute' block
      ;
    
    postConditions
      : 'post' '{' conditions '}'
      ;
    ```
    - **parameterList**: A list of parameters, similar to function parameters, passed into the transaction.
    - **fields**: Optional local variables that can be declared within the transaction.
    - **prepare**: The prepare phase is used to access signing accounts, manage resources, and perform setup operations.
    - **preConditions**: Optional block to define conditions that must hold before executing the transaction logic.
    - **executePostConditions**: Handles the execution and post-conditions of the transaction.
    - **execute**: Optional block to be executed.
    - **postConditions**: Optional block to define conditions that must hold after the transaction logic has been executed.

- **Transaction Phases**:
  - **Transactions in Cadence are executed in four phases**:
    1. **Prepare Phase**:
      - The prepare phase provides access to the signing accounts. These accounts can only be accessed within this phase, and the phase is used for logic that requires account modification, such as accessing or modifying account storage. It is recommended to limit this phase to only account-related logic.
        ```cadence
        prepare(signer: AuthAccount) {
          // Access signer's account and modify resources
        }
        ```
    2. **Pre-conditions**:
      - The pre block allows developers to define conditions that must be true before the transaction executes. If any pre-condition fails, the transaction is reverted without further execution. This phase ensures that the state of the blockchain is correct before the main logic runs.
        ```cadence
        pre {
          signer.balance >= 100.0: "Insufficient balance"
        }
        ```
    3. **Execute Phase**:
      - The execute phase contains the main logic of the transaction. It is where state changes such as token transfers, contract interactions, or resource creation typically occur. This phase cannot access the signing accounts directly, as that access is restricted to the prepare phase.
        ```cadence
        execute {
          // Main transaction logic
        }
        ```
    4. **Post-conditions**:
      - The post block contains checks that ensure the transaction was successful and the expected state changes occurred. If any post-condition fails, the transaction is reverted. This phase is useful for ensuring the integrity of the transaction’s results.
        ```cadence
        post {
          signer.balance == before(signer.balance) - 50.0: "Balance mismatch after transfer"
        }
        ```

- **Transaction Parameters**:
  - Transactions can accept parameters, allowing them to be customized when invoked. The parameter list is similar to function parameters and is declared after the transaction keyword.
    ```cadence
    transaction(amount: UFix64, recipient: Address) {
      prepare(signer: AuthAccount) {
        // Prepare phase
      }
      
      execute {
        // Use the passed-in amount and recipient
      }
    }
    ```

- **Example**:
    ```cadence
    transaction(amount: UFix64) {
      prepare(signer: AuthAccount) {
        // Access the signer's account and prepare for the transaction
      }
    
      pre {
        // Ensure the signer's balance is sufficient
        signer.balance >= amount: "Insufficient balance"
      }
      
      execute {
        // Perform the transfer logic here
      }
      
      post {
        // Ensure the transaction was executed correctly
        signer.balance == before(signer.balance) - amount: "Balance mismatch"
      }
    }
    ```
    - In this example, a transaction accepts an amount parameter and performs a token transfer from the signer’s account. The pre-conditions check that the signer has sufficient funds, and the post-conditions verify that the signer’s balance was reduced correctly.

- **Restrictions and Best Practices**:
  - **Signers can only be accessed in the prepare phase**: Direct access to signing accounts is restricted to the prepare phase. The logic that requires write access to accounts should be placed here, while general transaction logic should be in the execute phase.
  - **Separation of concerns**: To maintain clarity and security, it is recommended to separate account-related logic in the prepare phase from the main transaction logic in the execute phase. This separation helps in ensuring that critical account operations are clearly identifiable.
  - **Pre-conditions and post-conditions**: Use pre-conditions and post-conditions to safeguard the transaction’s execution and state changes. They ensure that transactions only proceed when the blockchain is in the correct state and are reverted if any issues arise.

Transaction declarations in Cadence provide a structured way to manage blockchain interactions, ensuring safe and predictable state changes. By organizing logic into distinct phases—prepare, pre, execute, and post—developers can clearly separate account access, validation, execution, and verification, resulting in secure and robust transactions on the Flow blockchain.

#### Pragma Declarations

In Cadence, pragma declarations introduce special instructions or directives into a contract or script that can affect its behavior during compilation or execution. Pragmas are typically used to signal certain conditions or constraints that should be applied to the code in which they appear.

Pragmas do not directly affect the logic of the contract or script itself but modify the context in which it operates. For instance, pragmas can mark types as deprecated or removed, ensuring that they are no longer usable or can prevent a type from being reintroduced after removal.

- **Syntax**:
  - A pragma declaration is introduced by the # symbol followed by an expression that specifies the directive. Pragmas typically appear at the top of a contract or before a relevant section to indicate a condition that applies to a specific type or construct.
    ```cadence
    #pragma expression
    ```

- **EBNF**:
  - The formal EBNF for pragma declarations is as follows:
    ```ebnf
    pragmaDeclaration
      : '#' expression
      ;
    ```
      - **expression**: The instruction or directive that the pragma applies to the contract or type.

- **Example**:
  ```cadence
  access(all) contract MyContract {
    #removedType(MyResource)
    
    // Other contract declarations...
  }
  ```
  - In this example, the #removedType(MyResource) pragma is used to indicate that the type MyResource has been removed from the contract and cannot be reintroduced or redeclared in the future.

- **The #removedType Pragma**:
  - The #removedType pragma is a specific directive that marks a type as removed from a contract, preventing any future declaration of the type under the same name within the contract. This ensures that previously defined types that are no longer needed can be safely “tombstoned,” avoiding potential security risks that might arise from reintroducing the type with different properties.
  ```cadence
  #removedType(TypeName)
  ```
    - **TypeName**: The name of the type that has been removed from the contract.
  - The #removedType pragma has the following effects:
    - **Prevents redeclaration**: Once a type is marked as removed, it cannot be redefined under the same name in the contract.
    - **Ensures consistency**: It helps maintain consistency by preventing potential conflicts or type confusion caused by the reintroduction of a type that was previously removed.
  - **Example**:
  ```cadence
  access(all) contract TokenContract {
    #removedType(Vault)
    
    // Other contract declarations...
  }
  ```
    - In this example, the Vault resource has been removed from the contract, and the #removedType(Vault) pragma ensures that no future declarations of Vault will be allowed.

- **Usage of Pragmas**:
  - Pragmas should be used sparingly, as they affect the broader behavior of the contract and can introduce constraints that limit future development flexibility. In particular, the #removedType pragma should be carefully applied to avoid accidentally locking a contract from necessary updates or redeclarations.

Pragma declarations in Cadence provide a mechanism to introduce special directives that affect the contract’s compilation and execution context. The #removedType pragma, in particular, allows developers to safely remove types from a contract, ensuring they cannot be reintroduced and maintaining the integrity of the contract’s stored data and behavior.

#### Entitlement Declarations

In Cadence, entitlement declarations provide fine-grained control over access to specific members (fields and functions) of composite types such as resources and structs. Entitlements are used to specify which references are authorized to access those members. By applying entitlements, developers can enforce custom access rules based on the specific rights granted to references, rather than relying solely on general access modifiers.

Entitlements are declared using the entitlement keyword and can be combined with access modifiers to specify the entitlements required to access specific fields or functions.

- **EBNF**:
  - The formal EBNF for entitlement declarations is as follows:
    ```ebnf
    entitlementDeclaration
      : 'entitlement' identifier
      ;
    ```
    - **entitlement**: Introduces the declaration of a new entitlement.
    - **identifier**: The name of the entitlement being declared.

- **Declaring Entitlements**:
  - Entitlements are declared using the entitlement keyword, followed by the name of the entitlement.
    ```cadence
    entitlement ReadAccess
    entitlement WriteAccess
    ```
    - In the example above, two entitlements, ReadAccess and WriteAccess, are declared. These entitlements can later be applied to fields or functions within composite types.

- **Using Entitlements in Access Modifiers**:
  - Entitlements are used in access modifiers to control the access to fields and functions based on authorized references. This is done by specifying the entitlement(s) required to access the member.
    ```cadence
    access(all)
    resource Vault {
     
      // Requires 'ReadAccess' entitlement to read this field
      access(ReadAccess)
      let balance: Int
      
      // Requires both 'ReadAccess' and 'WriteAccess' entitlements to modify this field
      access(ReadAccess, WriteAccess)
      var balanceMutable: Int
    }
    ```
    - In the above example:
      - The balance field requires the ReadAccess entitlement for reading.
      - The balanceMutable field requires both ReadAccess and WriteAccess entitlements for modification.

- **Combining Entitlements**:
  - Entitlements can be combined using either conjunction (,) or disjunction (|) to create complex access rules:
    - Conjunction (access(E, F)): Requires both entitlements E and F.
    - Disjunction (access(E | F)): Requires either entitlement E or F.
    ```cadence
    resource MyResource {
      
      // Requires either 'ReadAccess' or 'WriteAccess' entitlement to access this field
      access(ReadAccess | WriteAccess)
      let flexibleAccessField: Int
      
      // Requires both 'ReadAccess' and 'WriteAccess' entitlements to access this field
      access(ReadAccess, WriteAccess)
      var strictAccessField: Int
    }
    ```

- **Entitlement Mappings**:
  - In addition to individual entitlement declarations, entitlement mappings can be used to propagate entitlements between parent and child objects. This allows access to a child object to be governed by the entitlements of its parent.
  - An entitlement mapping is declared using the entitlement mapping syntax:
    ```cadence
    entitlement mapping ParentToChild {
      ParentAccess -> ChildAccess
    }
    ```
    - In the example above, the ParentToChild mapping states that when a reference has the ParentAccess entitlement, it will automatically gain access to child objects that require the ChildAccess entitlement.

- **Example: Entitlement Declaration in a Resource**
  ```cadence
  entitlement ReadAccess
  entitlement WriteAccess
  
  resource MyResource {
    
    // Requires 'ReadAccess' entitlement to read
    access(ReadAccess)
    let readOnlyData: Int
    
    // Requires both 'ReadAccess' and 'WriteAccess' entitlements to modify
    access(ReadAccess, WriteAccess)
    var modifiableData: Int
    
    // Function accessible only to those with 'WriteAccess'
    access(WriteAccess)
    fun modifyData(newData: Int) {
      self.modifiableData = newData
    }
  }
  ```
  - In this example, MyResource declares two entitlements (ReadAccess and WriteAccess) and uses them to control access to its fields and functions. The readOnlyData field can only be read by references with ReadAccess, while modifiableData can be read by references with ReadAccess and modified by references with both ReadAccess and WriteAccess.

Entitlement declarations in Cadence provide a powerful mechanism for implementing detailed access control within composite types. By combining entitlements with access modifiers, developers can specify precise access rules that govern how different parts of a resource or struct can be accessed, ensuring robust security and flexibility in managing access rights.

#### Entitlement Mapping Declarations

In Cadence, entitlement mappings allow the propagation of access rights from one entitlement to another within a hierarchy of objects. This enables a more efficient and maintainable way to control access when an entitlement on a parent object should automatically grant entitlements to a child object. By defining these relationships, developers can avoid duplicating accessor logic and ensure consistent access control across nested resources and composite types.

Entitlement mappings define how access to one entitlement in a parent object grants corresponding access to another entitlement in a child object. These mappings can be applied to fields and functions of composite types to control access based on authorized references.

- **EBNF**:
  ```ebnf
  entitlementMappingDeclaration:
      'entitlement' 'mapping' identifier '{' 1*entitlementMappingRule '}'
    ;
  
  entitlementMappingRule:
      entitlementMappingRuleNormal
    | entitlementMappingRuleInclude
    ;
  
  entitlementMappingRuleNormal:
      entitlementMappingRuleNormalSource '->' entitlementMappingRuleNormalTarget
    ;
  
  entitlementMappingRuleNormalSource:
      identifier
    ;
  
  entitlementMappingRuleNormalTarget:
      identifier
    ;
  
  entitlementMappingRuleInclude:
      'include' identifier
    ;
  ```
    - **entitlement** mapping: Introduces the declaration of an entitlement mapping.
    - **identifier**: The name of the mapping or entitlements being mapped.
    - **entitlementMappingRuleNormal**: Declares a rule for mapping one entitlement to another.
    - **entitlementMappingRuleInclude**: Allows inclusion of an existing entitlement mapping.

- **Declaring Entitlement Mappings**:
  - An entitlement mapping is declared using the entitlement mapping keyword followed by the mapping name and a set of mapping rules that define how one entitlement maps to another.
    ```cadence
    entitlement mapping ParentToChild {
      ParentAccess -> ChildAccess
    }
    ```
    - In this example, the ParentToChild mapping indicates that a reference with the ParentAccess entitlement automatically gains ChildAccess when accessing fields or functions of a nested object.

- **Using Entitlement Mappings in Composite Types**:
  - Once an entitlement mapping is declared, it can be applied to fields or functions in composite types, such as resources and structs, to propagate entitlements from parent objects to child objects.
    ```cadence
    entitlement ParentAccess
    entitlement ChildAccess
    
    entitlement mapping ParentToChild {
      ParentAccess -> ChildAccess
    }
    
    access(all)
    resource ParentResource {
      
      access(mapping ParentToChild)
      let childResource: @ChildResource
      
      init(childResource: @ChildResource) {
        self.childResource <- childResource
      }
    }
    
    access(all)
    resource ChildResource {
      access(ChildAccess)
      fun restrictedFunction() {
        // Function logic here
      }
    }
    ```
    - In this example:
      - The ParentToChild entitlement mapping allows a reference with the ParentAccess entitlement to access the ChildAccess-restricted functions in ChildResource.
      - The field childResource in ParentResource is annotated with access(mapping ParentToChild), meaning that access to the field is governed by the ParentToChild mapping.

- **Including Entitlement Mappings**:
  - Entitlement mappings can include other mappings using the include keyword, allowing reuse of mapping logic across multiple declarations.
    ```cadence
    entitlement ParentAccess
    entitlement ChildAccess
    entitlement GrandchildAccess
    
    entitlement mapping ParentToChild {
      ParentAccess -> ChildAccess
    }
    
    entitlement mapping ChildToGrandchild {
      include ParentToChild
      ChildAccess -> GrandchildAccess
    }
    ```
    - In this example, the ChildToGrandchild mapping includes the ParentToChild mapping and adds an additional rule, allowing references with ChildAccess to gain GrandchildAccess when accessing deeply nested resources.

- **ICombining Entitlements in Mappings**:
  - **Mappings support both conjunction (,) and disjunction (|) when combining entitlements**:
    - **Conjunction (->)**: The reference must have both entitlements to gain access.
    - **Disjunction (|)**: The reference must have at least one of the entitlements to gain access.
    ```cadence
    entitlement ParentAccess
    entitlement ChildAccess
    entitlement ExtraAccess
    
    entitlement mapping CombinedMapping {
      ParentAccess, ExtraAccess -> ChildAccess
    }
    ```
    - In this example, the CombinedMapping rule specifies that both ParentAccess and ExtraAccess are required to grant ChildAccess.

- **Example: Entitlement Mapping in Action**
  ```cadence
  entitlement AdminAccess
  entitlement UserAccess
  
  entitlement mapping AdminToUser {
    AdminAccess -> UserAccess
  }
  
  access(all)
  resource AdminResource {
    
    access(mapping AdminToUser)
    let userResource: @UserResource
    
    init(userResource: @UserResource) {
      self.userResource <- userResource
    }
  }
  
  access(all)
  resource UserResource {
    access(UserAccess)
    fun userFunction() {
      // Function logic
    }
  }
  ```
  - In this example:
    - The AdminToUser mapping allows any reference with AdminAccess to access the UserAccess-restricted functions in UserResource.
    - The field userResource in AdminResource is governed by the AdminToUser mapping, allowing authorized access to the child resource.

Entitlement mapping declarations in Cadence enable efficient and maintainable access control across hierarchies of composite types. By mapping entitlements from parent objects to child objects, developers can enforce consistent access control rules and avoid duplicating logic. Entitlement mappings provide a flexible and powerful mechanism for managing access to complex nested structures while ensuring security and correctness.

---

#### Contract Declarations

Contracts are the core building blocks of Cadence. A contract encapsulates state and behavior, acting as a secure container for digital assets or logic. Contracts can contain functions, resources, and variables.

- **Example**:
  ```cadence
  pub contract ContractName {
      // Variables, functions, and resources
  }
  ```
- **Semantics**:
  - The `pub` keyword makes the contract publicly accessible.
  - Contracts can define and store resources, enforce access controls, and implement business logic.

#### Interface Declarations

Interfaces define abstract types that specify certain behavior without providing implementation details. Both **static interfaces** (checked at compile time) and **dynamic interfaces** (resolved at runtime) are supported.

- **Example**:
  ```cadence
  pub interface InterfaceName {
      // Function and resource declarations
  }
  ```
- **Semantics**:
  - Contracts or other types that implement an interface must provide the specific functionality defined in the interface.

#### Function Declarations

Functions in Cadence are declared with the `fun` keyword and can be either public or private. Functions may be defined within contracts, interfaces, or as standalone entities in scripts or transactions.

**TODO**
Describe the syntax and semantics of the parameters in detail. No support for optional parameters with default values. Describe function preconditions and postconditions. Discuss the `view` annotation for functions.

- **Example**:
  ```cadence
  pub 'fun' functionName(param: Type): ReturnType {
      // Function body
  }
  ```
- **EBNF**:
  ```ebnf
  functionDeclaration
    : access Fun identifier parameterList ( ':' typeAnnotation )? functionBlock?
    ;
  ```
- **Scope**:
  - **Public**: Functions can be called from outside the contract or script (`pub` keyword).
  - **Private**: Functions are accessible only within the defining scope (omitting `pub`).

#### Variable Declarations

Variables in Cadence are declared with `let` (constant) or `var` (mutable) keywords. Variables can hold simple values (like integers or strings), resources, or references.

- **Example**:
  ```cadence
  let variableName: Type = initialValue
  var mutableName: Type = initialValue
  ```
- **EBNF**:
  ```ebnf
  variableDeclaration
    : access variableKind identifier ( ':' typeAnnotation )? transfer expression ( transfer expression )?
    ;
  ```
- **Rules**:
  - Variables must be explicitly typed or inferred based on the initial value.
  - Resources must be explicitly moved or destroyed when no longer needed.

---

### 3. Statements

In Cadence, statements represent the basic units of execution within a program. They define the structure and flow of control, allowing the program to perform actions, make decisions, iterate over data, and manage resources. Statements are the building blocks of functions, contracts, transactions, and scripts. Each statement typically results in a side effect, such as modifying the state of a contract, transferring a resource, or evaluating an expression.

The Statements section describes the various types of statements in Cadence, including their syntax, semantics, and behavior. This section uses Extended Backus-Naur Form (EBNF) notation to define the formal grammar of Cadence’s statements and explains how to properly use each statement in a program.

Cadence supports the following types of statements:

  - **Return Statements**: Used to return a value from a function or terminate function execution.
  - **Break Statements**: Used to exit loops prematurely.
  - **Continue Statements**: Used to skip the remaining iteration of a loop and proceed to the next iteration.
  - **If-Else Statements**: Conditional statements that allow branching based on boolean expressions.
  - **While Statements**: Looping constructs that repeatedly execute a block of code while a condition is true.
  - **For Statements**: Iteration constructs that loop over a collection or range.
  - **Switch statement**: a control flow mechanism that branches execution based on the value of an expression, simplifying evaluation of multiple possible values.
  - **Emit Statements**: Used to emit events from a contract, signaling significant state changes or actions.
  - **Assignment Statements**: Assign values to variables or references.
  - **Swap Statements**: Exchange the values of two variables.
  - **Expression Statements**: Evaluate an expression that results in a side effect, such as a function call or resource operation.

Each statement type is designed to control the flow of execution and state in a Cadence program, ensuring that programs are expressive, concise, and safe, particularly when dealing with resources and assets. By following Cadence’s strict rules on resource management, developers can write programs that avoid common pitfalls such as resource duplication or unintended data loss.

This section elaborates on each type of statement, providing both the formal grammar and example usage, along with explanations on when and how to use each construct effectively within a smart contract or transaction.

- **EBNF**:
  ```ebnf
  statements
    : ( statement eos )*
    ;

  statement
    : returnStatement
    | breakStatement
    | continueStatement
    | ifStatement
    | whileStatement
    | forStatement
    | switchStatement
    | emitStatement
    | declaration
    | assignmentStatement
    | swapStatement
    | expressionStatement
    ;
  ```

#### Return Statements

Return statements in Cadence are used to exit a function and optionally return a value to the caller. When a return statement is executed, the function’s execution is immediately halted, and control is transferred back to the point where the function was invoked. If the function has a return type, a value must be provided as part of the return statement.

Return statements are particularly useful when a function needs to return a result after performing calculations, executing logic, or interacting with resources. If a function is declared to return a value, failing to include a return statement with the correct value will result in a compile-time error. Conversely, in functions with no return type (i.e., functions that return Void), the return keyword can be omitted.

  - **Syntax**:
    - The return statement is followed by an optional expression that provides the value to be returned.
    - In the case of functions returning Void, the return statement is optional and can be omitted.
    - If used, the return statement may look as follows:

    ```cadence
    return expression;
    ```

  - **Semantics**:
    - The expression after return must match the function’s declared return type. If the function does not expect a return value, using a return statement with a value will result in a compile-time error.
    - When a return statement is encountered, the function immediately terminates, and no further statements in the function are executed.
  
  - **Usage**:
    - Return statements are often placed at the end of a function to return a computed result.
    - In conditional or early-exit scenarios, return statements can appear anywhere in the function body to exit the function prematurely.
  
  - **Example**:
    - Returning a value from a function:

    ```cadence
    pub fun add(a: Int, b: Int): Int {
      return a + b;
    }
    ```

    - Exiting a function without returning a value:

    ```cadence
    pub fun logMessage(message: String) {
      log(message);
      return;
    }
    ```

  - **EBNF**:
    ```ebnf
    returnStatement
      : 'return' ( expression )?
      ;
    ```
In Cadence, return statements play a crucial role in controlling the flow of functions, ensuring that the correct data is returned to the caller, or allowing functions to terminate early when certain conditions are met.

#### Break Statements

Break statements in Cadence are used to prematurely exit loops, such as while or for loops, before the loop condition is fully met or the iteration has completed. When a break statement is encountered, control immediately exits the loop, and the program continues executing the code following the loop. This allows developers to terminate loops early based on specific conditions or events.

Break statements are commonly used when:

  - A loop has found the result it is searching for and further iteration is unnecessary.
  - An error or unexpected condition occurs within the loop, and continued execution would be invalid.
  - A performance optimization is needed to avoid processing the remaining iterations of a loop when no longer required.
 
  - **Syntax**:
    - A break statement consists of the break keyword, followed by a semicolon.
  
  - **Example**:

    ```cadence
    break;
    ```

  - **Semantics**:
    - When a break statement is executed, the loop is immediately terminated, and control moves to the first statement following the loop.
    - break can only be used inside while or for loops. Using break outside of a loop context results in a compile-time error.
   
  - **Usage**:
    - Break statements are typically used inside conditional blocks within a loop. When the condition is met, the loop exits early.
      - Example:

      ```cadence
      for item in items {
        if item == target {
          break;  // Exit the loop if the target item is found
        }
      }
      ```

  - **Example**:
    - **Exiting a while loop early**:

    ```cadence
    while counter < 10 {
      if counter == 5 {
        break;  // Exit the loop when counter reaches 5
      }
      counter = counter + 1
    }
    ```
  - **EBNF**:
    ```ebnf
    breakStatement
      : 'break'
      ;
    ```
Break statements provide an efficient way to manage loop execution, allowing the program to stop iterating when a specific condition is met, thereby avoiding unnecessary computations. This makes loops more flexible and improves performance in certain scenarios.

#### Continue Statements

Continue statements in Cadence are used within loops (such as while or for loops) to skip the current iteration and proceed directly to the next iteration. When a continue statement is encountered, the remaining code in the current iteration is ignored, and control moves to the beginning of the next iteration of the loop. Unlike break, which exits the loop entirely, continue allows the loop to keep running but skips over specific steps based on certain conditions.

Continue statements are useful in scenarios where:

  - Certain conditions within a loop should be ignored, but the loop should continue running.
  - You want to avoid executing some part of the loop’s body under specific circumstances.
  - You need to selectively bypass certain iterations without stopping the entire loop.

  - **Syntax**:
    - A continue statement consists of the continue keyword, followed by a semicolon.

    - Example:

      ```cadence
      continue;
      ```

  - **Semantics**:
    - When a continue statement is executed, the loop immediately skips the remaining code in the current iteration and begins the next iteration.
    - continue can only be used inside while or for loops. Using it outside a loop results in a compile-time error.
    - If a loop condition remains valid after the continue statement, the next iteration begins, otherwise the loop terminates.

  - **Usage**:
    - Continue statements are typically placed inside a conditional block, allowing specific iterations to be skipped when the condition is met.
    - Example:

      ```cadence
      for item in items {
        if item == invalidItem {
          continue;  // Skip processing this iteration if the item is invalid
        }
        // Process valid items
      }
      ```

  - **Example**:
    - **Skipping an iteration in a while loop**:

      ```cadence
      var counter = 0
      while counter < 10 {
        counter = counter + 1
        if counter % 2 == 0 {
          continue;  // Skip even numbers
        }
        log("Odd number: ".concat(counter.toString()))
      }
      ```
    - In this example, the continue statement is used to skip even numbers, so the loop only logs odd numbers.

  - **EBNF**:
    ```ebnf
    continueStatement
      : 'continue'
      ;
    ```
Continue statements offer fine control over loop execution, allowing developers to skip unnecessary iterations while keeping the loop running. This improves readability and efficiency, especially when certain iterations do not require any processing.

#### If Statements

If statements in Cadence are used to conditionally execute a block of code based on the evaluation of a boolean expression. If the condition evaluates to true, the code block associated with the if statement is executed. If the condition evaluates to false, Cadence allows for additional branching through else-if statements and else statements to define alternative blocks of code that can be executed depending on other conditions.

The if-else construct is essential for controlling program flow and making decisions based on dynamic values. It enables branching logic, allowing different outcomes based on the results of condition checks.

  - **Syntax**:
    - The if statement begins with the if keyword, followed by a boolean expression (condition), and a block of code enclosed in curly braces ({}). If the condition evaluates to true, the block is executed.
    - Else-if statements (else if) can follow an if statement to check additional conditions if the previous if or else if conditions were false. Each else if condition is evaluated sequentially.
    - The else statement is optional and provides a default block of code to be executed if none of the if or else if conditions were true.

  - **Semantics**:
    - If statement: Executes a block of code if the condition evaluates to true. If the condition is false, execution proceeds to the next else if condition or the else block.
    - Else-if statements: Allow for multiple conditions to be evaluated in sequence. If an else if condition evaluates to true, its block is executed, and the remaining conditions are skipped.
    - Else statement: Executes when none of the preceding conditions (if or else if) were true. It acts as a fallback when all other conditions fail.

  - **Usage**:
    - Use if statements for simple conditional checks, such as verifying a balance or checking input validity.
    - Use else if to add more conditions that should be checked if the original if condition was not met.
    - Use else to handle any cases that do not match the previous conditions, ensuring that the program always has a valid execution path.

  - **Example**:
      ```cadence
      pub fun checkValue(value: Int) {
        if value > 10 {
          log("Value is greater than 10")
        } else if value == 10 {
          log("Value is exactly 10")
        } else {
          log("Value is less than 10")
        }
      }
      ```
    - In this example:
      - The if block checks whether value is greater than 10.
      - The else if block checks if the value is exactly 10, only if the if condition is false.
      - The else block is executed if neither the if nor the else if conditions are true.

    - **Else-if Statements**:
      - Else-if statements provide a mechanism to chain multiple conditions after the initial if statement. Each else if block is evaluated in sequence, and the first one that evaluates to true is executed, bypassing the rest.
      - Example:
        ```cadence
        if score >= 90 {
          log("Grade: A")
        } else if score >= 80 {
          log("Grade: B")
        } else if score >= 70 {
          log("Grade: C")
        } else {
          log("Grade: F")
        }
        ```

    - **Else Statements**:
      - The else statement handles cases where none of the preceding conditions are met. It provides a fallback execution path, ensuring that the program responds even when no specific condition was satisfied.
      - Example:
        ```cadence
        if isAdmin {
          log("Welcome, Admin!")
        } else {
          log("Access denied.")
        }
        ```

  - **EBNF**:
    ```ebnf
    ifStatement
      : 'if' ( expression | variableDeclaration ) block elseIfStatement* elseStatement?
      ;
    
    elseIfStatement
      : 'else' 'if' ( expression | variableDeclaration ) block
      ;
    
    elseStatement
      : 'else' block
    ```

The if-else construct provides flexibility for decision-making in Cadence programs, allowing you to define clear, structured paths based on conditional logic. By chaining multiple conditions through else if and providing fallback behavior through else, you ensure that your programs can handle various inputs and scenarios robustly.

#### While Statements

While statements in Cadence allow a block of code to be repeatedly executed as long as a specified condition evaluates to true. The condition is checked before each iteration, and if it evaluates to false, the loop terminates, and control moves to the code following the loop. While loops are typically used when the number of iterations is not known in advance but is determined dynamically based on some condition.

The while loop offers a flexible way to repeatedly perform actions, such as processing data or performing calculations, until a specific condition is met or no longer holds true.

  - **Syntax**:
    - The while statement begins with the while keyword, followed by a boolean expression (the loop condition), and a block of code enclosed in curly braces ({}). The code inside the block is repeatedly executed as long as the condition remains true.
    - Example:
        ```cadence
        while condition {
          // Loop body
        }
        ```

  - **Semantics**:
    - Condition Check: Before each iteration, the condition is evaluated. If the condition evaluates to true, the loop body is executed. If the condition evaluates to false, the loop terminates, and control passes to the next statement following the loop.
    - Termination: The loop will continue indefinitely if the condition always evaluates to true unless an external factor or break statement interrupts the loop. Ensure that the loop condition will eventually evaluate to false to prevent infinite loops.

  - **Usage**:
    - Use while loops when the number of iterations is not predetermined and depends on dynamic factors, such as user input, the state of a resource, or other conditions.
    - Common scenarios for while loops include:
    - Continuously processing data until an exit condition is met.
    - Repeatedly checking a condition until it becomes false.
    - Waiting for an external event or status change.

  - **Example**:
    ```cadence
    pub fun countToTen() {
      var counter = 1
      while counter <= 10 {
        log(counter)
        counter = counter + 1
      }
    }
    ```

    - In this example, the loop starts with counter set to 1. The condition counter <= 10 is checked before each iteration, and the loop continues logging the counter’s value and incrementing it by 1 until the counter exceeds 10. Once the condition is false (i.e., when counter is greater than 10), the loop terminates.

    - **Common Use Cases**:
      - Processing lists: You can use a while loop to iterate over elements in a list or collection until all elements have been processed.
      - Polling: While loops can be used to repeatedly check for changes or updates in a system until a desired state is reached.
      - Waiting for a condition: A while loop can be useful for waiting until a specific condition is met, such as reaching a certain balance or confirming an action’s success.
    - **Important Considerations**:
      - Ensure that the loop condition will eventually become false to avoid infinite loops. Failing to include logic that alters the loop condition can result in non-terminating execution.
      - Use break statements when necessary to exit a loop early based on certain conditions.

  - **EBNF**:
    ```ebnf
    whileStatement
      : 'while' expression block
      ;
    ```
While loops provide a flexible and dynamic way to control iteration in Cadence programs. By continuously checking a condition before each iteration, they allow developers to build loops that respond to real-time data and system states, making them ideal for scenarios where the number of iterations is not predetermined.

#### For Statements

For statements in Cadence allow developers to iterate over elements in a collection such as arrays, dictionaries, or ranges. In addition to iterating over the elements themselves, for statements also support an optional index variable that tracks the current position of the element in the collection. This is particularly useful when both the element and its position are needed within the loop body.

When using the optional index variable, the loop will provide both the current index and the corresponding element in each iteration, making it easier to perform operations that rely on the position of the element within the collection.

  - **Syntax**:
    - The for statement starts with the for keyword, followed by the optional index variable and the required element variable separated by a comma, the in keyword, and the collection to iterate over. The block of code enclosed in curly braces ({}) is executed for each element in the collection.
    - **Example without index**:
      ```cadence
      for item in list {
        // Execute for each item in list
      }
      ```

    - **Example with index**:
      ```cadence
      for index, item in list {
        log("Item at index ".concat(index.toString()).concat(": ").concat(item))
      }
      ```
  - **Semantics**:
    - Element variable: Refers to the current element in the collection being iterated over.
    - Index variable (optional): Represents the position of the current element within the collection, starting from 0 for the first element.
    - Iteration: The loop proceeds through the collection, processing each element and its associated index (if specified). The index increments automatically with each iteration.

  - **Usage**:
    - Use for loops to iterate over elements in a collection when you need to process each element individually.
    - Use the index variable when both the element and its position in the collection are needed for processing.
    - Example without index:
      ```cadence
      let numbers = [10, 20, 30]
      for number in numbers {
        log(number)
      }
      ```

    - **Example with index**:
      ```cadence
      let fruits = ["apple", "banana", "cherry"]
      for index, fruit in fruits {
        log("Fruit at index ".concat(index.toString()).concat(": ").concat(fruit))
      }
      ```

      - In this example, the loop iterates over the fruits array, logging both the index and the corresponding fruit name for each element.

    - **Looping Through Arrays with Index**:
      - The index variable can be used to access both the position and value of elements in an array.
      - Example:
      ```cadence
      let scores = [85, 90, 78]
      for index, score in scores {
        log("Score ".concat(index.toString()).concat(": ").concat(score.toString()))
      }
      ```

    - **Looping Through Ranges with Index**:
      - The for loop can iterate through a range while also tracking the index.
      - Example:
      ```cadence
      for index, value in 1...5 {
        log("Index: ".concat(index.toString()).concat(", Value: ").concat(value.toString()))
      }
      ```

    - **Common Use Cases**:
      - Processing data with the index: When both the position and value of each element need to be processed or displayed.
      - Enumerating through lists: The index can be used to perform operations specific to the element’s position, such as accessing neighboring elements or performing operations at specific indices.
 
- **EBNF**:
  ```ebnf
  forStatement
    : 'for' ( index ',' )? element 'in' expression block
    ;
  ```

In Cadence, for loops with the optional index variable offer enhanced flexibility by allowing developers to not only access each element in a collection but also track its position. This can simplify code when both the value and index are needed, making for loops a powerful construct for data processing and manipulation.

#### Switch Statements

In Cadence, a switch statement allows you to compare an expression against multiple patterns and execute the block of code associated with the first matching pattern. This provides a structured and concise way to manage complex control flow, improving both readability and safety.

  - **Syntax**:
    - A switch statement begins with the switch keyword, followed by an expression, and then a series of case blocks. Each case defines a pattern to be compared with the expression. An optional default block can be added to handle any cases where the expression does not match any of the predefined patterns.

  - **Example:
    ```cadence
    switch expression {
      case pattern1:
        // Execute if expression matches pattern1
      case pattern2:
        // Execute if expression matches pattern2
      default:
        // Execute if no patterns match
    }
    ```

  - **Key Features**:
    - **Pattern Matching**: Each case defines a pattern that is compared against the value of the expression. If the expression matches the pattern, the associated block of code is executed.
    - **Required Handling** for Enum Types: When using an enum as the test expression, it is recommended to handle all possible values of the enum. If any values are unhandled, a warning is issued. A default case can be included to ensure all enum values are addressed.
    - **No Implicit Fallthrough**: Unlike in some other languages (e.g., C, JavaScript), Cadence does not allow fallthrough between cases. Once a case’s block of code is executed, control automatically exits the switch statement. This prevents accidental execution of subsequent cases, ensuring safer control flow. There is no need to explicitly include break statements to avoid fallthrough.
    - **Duplicate Cases**: If a duplicate case value is detected, a warning is issued to alert the developer that the second occurrence will never be executed. This is not treated as an error for backward compatibility, but it is important to avoid duplicate case values to ensure correct behavior.
    - **Break Statements**: While Cadence does not require break statements to prevent fallthrough, a break can still be used if you need to exit the switch statement early or prevent further code execution within the current block. This provides additional flexibility in controlling program flow.
    - **Default Case**: The default case is optional, but it is recommended when handling non-exhaustive conditions. It should always appear as the last case and will execute if none of the preceding cases match the expression.
    - **Case Block Requirements**: Each case must contain at least one statement. Empty case blocks are invalid and will result in a compile-time error. This requirement ensures that all cases are explicitly handled with meaningful code.

    - Example
      - Here is an example of a switch statement that checks the value of a variable and logs a message based on its value:
      ```cadence
      let status: String = "completed"
      
      switch status {
      case "pending":
        log("The transaction is pending.")
      case "completed":
        log("The transaction is completed.")
      case "failed":
        log("The transaction has failed.")
      default:
        log("Unknown transaction status.")
      }
      ```
      - In this example, the switch statement checks the value of status and logs a message corresponding to the matched case. If the value does not match any of the cases, the default block executes, logging “Unknown transaction status.”

- **EBNF**:
  ```ebnf
  switchStatement:
    "switch" expression "{" ( caseStatement )* defaultStatement "}" ;
  
  caseStatement:
    "case" pattern ":" statement+ ;
  
  defaultStatement:
    "default" ":" statement+ ;
  ```

- **Pattern Matching**:
    - Cadence uses pattern matching within switch statements to provide a flexible mechanism for comparing values against multiple possible patterns. Pattern matching can handle simple values, enums, and more complex structures, allowing for clear and concise control flow.

This section provides a comprehensive description of switch statements in Cadence, covering syntax, behavior, and safety features, while ensuring that all edge cases and usage patterns are clearly defined.

#### Emit Statements

Emit statements in Cadence are used to trigger events that signal important actions or state changes within a smart contract. Events are a key feature of the Flow blockchain, providing a way for contracts to communicate significant occurrences to external systems, such as user interfaces, monitoring tools, or off-chain services.

By using emit statements, developers can make contracts more transparent and observable. For example, events can notify external systems when a token transfer occurs, a resource is created or destroyed, or an account balance changes. These events are immutably stored on the blockchain, allowing them to be queried and audited later.

  - **Syntax**:
    - The emit statement begins with the emit keyword, followed by the event identifier and the invocation of the event, which passes the required parameters.
    - Example:
      ```cadence
      emit Transfer(from: sender, to: recipient, amount: 100.0)
      ```

  - **Semantics**:
    - Event Declaration: Events must first be declared in the contract using the event keyword. An event declaration specifies the name of the event and the parameters that will be included when it is emitted.
      - Example:
        ```cadence
        pub event Transfer(from: Address, to: Address, amount: UFix64)
        ```

    - **Event Emission**: The emit statement triggers an event during the execution of the contract, passing the specified parameters to the event. The event is then recorded on the blockchain for later access by external systems.
    - **Event Propagation**: Once an event is emitted, it is stored immutably as part of the transaction’s log. External applications can listen for these events and respond to them accordingly (e.g., updating a user’s interface when tokens are transferred).

  - **Usage**:
    - Emit statements are typically used to log significant contract actions, such as transfers, resource creation, state changes, or contract interactions. Developers use events to inform external observers of these actions without modifying the contract’s state.
    - Example of emitting an event during a token transfer:

      ```cadence
      pub fun transfer(sender: AuthAccount, recipient: AuthAccount, amount: UFix64) {
        sender.withdraw(amount)
        recipient.deposit(amount)
        emit Transfer(from: sender.address, to: recipient.address, amount: amount)
      }
      ```
      - In this example, the Transfer event is emitted after the transfer action has been successfully completed, notifying external systems that the transfer occurred.

  - **Events and External Systems**:
    - External systems (such as dApp frontends or monitoring tools) can subscribe to and listen for events emitted by contracts. By doing so, they can react in real-time to significant contract activities, providing users with timely feedback or triggering off-chain actions.
    - Example use case: Emitting an event when a new NFT is minted, allowing an off-chain marketplace to display the new asset.
      ```cadence
      emit NFTMinted(owner: newOwner.address, tokenId: newTokenId)
      ```

  - **Common Use Cases**:
    - **Logging transfers**: Notify external systems when a transfer of tokens, NFTs, or other resources occurs.
    - **Tracking state changes**: Emit events when significant changes in contract state take place, such as when a contract is upgraded or a milestone is reached.
    - **Auditability*: Events serve as an immutable record of important actions, which can be referenced later for auditing or analysis purposes.

  - **Example**:

    ```cadence
    pub event TokenMinted(to: Address, amount: UFix64)
    
    pub fun mintTokens(recipient: AuthAccount, amount: UFix64) {
      recipient.deposit(amount)
      emit TokenMinted(to: recipient.address, amount: amount)
    }
    ```
    - In this example, the TokenMinted event is declared at the start of the contract and emitted after the mintTokens function successfully deposits tokens into the recipient’s account. External systems can listen for the TokenMinted event to track token issuance.

- **EBNF**:
  ```ebnf
  emitStatement
    : 'emit' identifier invocation
    ;
  ```

Emit statements in Cadence play a crucial role in facilitating communication between on-chain contracts and off-chain systems. By emitting events, contracts provide a transparent and reliable way to notify external systems of critical actions, making it easier to build responsive, data-driven decentralized applications that interact with the Flow blockchain.

#### Assignment Statements

Assignment statements in Cadence are used to assign values to variables or references. They allow developers to update the value of a variable or transfer resources between variables or accounts. The left-hand side (l-value) of an assignment must be a valid location in memory that can store the value, while the right-hand side (r-value) is an expression that evaluates to the value or resource being assigned.

Cadence supports multiple types of assignment:

  - **Simple assignment (=)**: Assigns a value to a variable.
  - **Move assignment (<-)**: Moves a resource from one variable to another. After the move, the original variable no longer holds the resource.
  - **Forced move (<-!)**: Similar to move assignment, but forces the move even if the value or resource is optional or fails under normal conditions.

Assignment statements are critical when dealing with both standard variables and Cadence’s resource types, ensuring proper management of assets, such as tokens or NFTs, without allowing duplication or implicit loss.

  - **Syntax**:
    - A simple assignment statement uses the = operator, while resource transfers use <- or <-!.
    - Example:
      ```cadence
      variableName = expression;
      token <- create Token();  // Moving a resource
      ```

  - **Semantics**:
    - **Simple Assignment (=)**: Assigns the result of an expression to a variable. The left-hand side must be a mutable variable that can store the value produced by the expression on the right-hand side.
    - **Move Assignment (<-)**: Transfers ownership of a resource from the right-hand side to the left-hand side. After the move, the resource no longer exists in the original variable, and any further attempts to access it will result in an error.
    - **Forced Move (<-!)**: Similar to move assignment, but allows moving resources in cases where the value might be optional or could fail under normal conditions.

  - **Usage**:
    - Simple assignments are used to update the value of variables or to store the result of a calculation.
    - Move assignments are crucial when dealing with resources, ensuring that resources are not copied and follow Cadence’s strict resource ownership rules.
    - Forced move can be used when a resource may be missing or unavailable but should be forced into a new location or storage without failing.

  - **Example**:
    - **Simple Assignment**:
      ```cadence
      let balance = 100.0
      balance = balance + 50.0  // Update balance
      ```
    - **Move Assignment**:
      ```cadence
      let token <- create Token()
      recipientVault <- token  // Move token to recipient's vault
      ```
    - **Forced Move**:
      ```cadence
      let maybeToken: @Token? = someFunction()
      recipientVault <-! maybeToken  // Forced move even if maybeToken is nil
      ```

  - **Assignment to Arrays and Dictionaries**:
    - **Arrays**: Assignments to arrays involve either updating an element at a specific index or assigning an entire array to a variable.
      - Example:
        ```cadence
        let numbers = [1, 2, 3]
         numbers[1] = 10  // Update element at index 1
        ```
    - **Dictionaries*: Assignments to dictionaries allow adding or updating a key-value pair.
      - Example:
        ```cadence
        let scores: {String: Int} = {"Alice": 85}
        scores["Bob"] = 90  // Add or update the value for key "Bob"
        ```
  - **Important Considerations**:
    - **Ownership**: Move assignments ensure that only one variable or account can own a resource at a time, preventing unintended duplication.
    - **Resource Safety**: Cadence enforces that resources cannot be implicitly discarded or lost. Any resource that is moved must be stored or destroyed explicitly.
    - **Assignment in Collections**: When assigning values to arrays or dictionaries, the target element or key must already exist, or it should be created explicitly before assignment.

- **EBNF**:
  ```ebnf
  assignmentStatement
    : expression transfer expression
    ;

  transfer
    : '='    (* Assign *)
    | '<-'   (* Move *)
    | '<-!'  (* Forced move *)
    ;
  ```

Assignment statements in Cadence ensure that variables and resources are managed safely and correctly. With the unique move semantics for resources, Cadence guarantees that assets are neither duplicated nor lost, making assignments a key part of resource management in decentralized applications. By supporting both simple value assignments and resource transfers, Cadence allows developers to build robust smart contracts while maintaining security and ownership guarantees.

#### Swap Statements

Swap statements in Cadence allow two variables to exchange their values or resources without needing an intermediate temporary variable. This construct simplifies scenarios where two variables must be swapped, such as when sorting or restructuring data. The swap operator (<->) ensures that both values are exchanged in a single step, maintaining resource safety and integrity.

The swap operation works seamlessly with both standard values and Cadence’s resource types, ensuring that the ownership of resources is preserved and properly handled during the exchange.

  - **Syntax**:
    - The swap statement uses the <-> operator to swap the values or resources between two variables. Both variables must be compatible with each other in terms of type.
    - Example:
      ```cadence
      variableA <-> variableB
      ```

  - **Semantics**:
    - Value Swapping: For simple variables, such as integers or strings, the values are exchanged between the two variables. The left-hand side variable takes the value of the right-hand side, and vice versa.
    - Resource Swapping: When swapping resources, Cadence ensures that ownership is correctly transferred between the two variables, following the same resource management rules that govern move semantics. The resources are swapped without duplication or implicit destruction.

  - **Usage**:
    - Swapping Values: Swap statements are used to exchange values without needing a temporary variable, making code more concise and easier to read.
    - Swapping Resources: When dealing with resources (such as tokens or NFTs), swap statements provide a convenient way to exchange ownership between two variables or accounts.

  - **Example**:
    - Swapping Integer Values:
      ```cadence
      var a = 5
      var b = 10
      a <-> b  // After the swap, a is 10, and b is 5
      ```
    - Swapping Resources:
      ```cadence
      let tokenA <- create Token()
      let tokenB <- create Token()
      tokenA <-> tokenB  // The resources are exchanged
      ```
      - In this example, the tokens tokenA and tokenB are swapped, transferring their ownership between the two variables.

  - **Considerations**:
    - **Type Compatibility**: The two variables involved in the swap must be of compatible types. Cadence enforces type safety, ensuring that incompatible types cannot be swapped.
    - **Resource Safety**: When swapping resources, Cadence ensures that neither resource is lost or duplicated. After the swap, each variable properly owns the resource that was previously owned by the other.
    - **Efficient Swapping**: The swap operation is a direct exchange and does not require any intermediate variables, making it an efficient mechanism for switching values or resources.

- **EBNF**:
  ```ebnf
  swap
    : expression '<->' expression
    ;
  ```

Swap statements in Cadence provide a straightforward and efficient way to exchange values or resources between two variables. This feature ensures that resources are safely handled while simplifying operations where swapping is necessary, such as sorting algorithms or reorganizing data. By using the swap operator, developers can avoid the overhead of using temporary variables while maintaining clarity and safety in resource management.

#### Expression Statements

Expression statements in Cadence allow expressions to be evaluated solely for their side effects, without necessarily producing or storing a result. While many expressions in Cadence return values, expression statements focus on the actions they perform, such as invoking a function, transferring a resource, or interacting with an account’s storage. These statements do not assign the result of the expression to a variable but instead execute the expression to produce a desired effect.

Expression statements are commonly used for operations that modify state, trigger events, or manipulate resources, especially in contexts where the result of the expression does not need to be retained for further use.

  - **Syntax**:
    - An expression statement consists of an expression followed by a semicolon (;).
    - Example:
      ```cadence
      functionCall();  // Invoking a function as an expression statement
      ```

  - **Semantics**:
    - **Function Calls**: Expression statements are frequently used to invoke functions where the function performs actions (e.g., logging, transferring resources, or modifying contract state) without needing to return a value that the program will use.
    - **Resource Operations**: Expression statements are also used to manage resources. For example, creating, destroying, or transferring resources are often carried out through expression statements.
    - **Side Effects**: The primary purpose of an expression statement is its side effect, which could be modifying state, transferring a resource, or emitting an event. Unlike expressions that are assigned to variables, expression statements are executed solely for their operational effects.

  - **Usage**:
    - **Function Invocation**: When calling a function for its side effect, such as logging data or emitting an event, without needing to capture the return value.
    ```cadence
    log("This is a log message");  // The log function is called but no value is stored
    ```
    - **Resource Movement**: Moving resources between accounts or destroying resources is a common use case for expression statements.
    ```cadence
    destroy token;  // Destroying a resource
    ```
    - **Event Emission**: Emitting events in contracts to signal important actions or changes in state.
    ```cadence
    emit Transfer(from: sender, to: recipient, amount: 100.0);  // Emit an event
    ```

  - **Examples**:
    - **Calling a function**:
    ```cadence
    performAction();  // Calls a function, but the return value is not used
    ```
    - **Transferring a resource**:
    ```cadence
    recipient.save(<- token);  // Moves the token to the recipient
    ```
    - **Destroying a resource**:
    ```cadence
    destroy nft;  // Destroy an NFT resource
    ```

- **EBNF**:
  ```ebnf
  expressionStatement
    : expression
    ;
  ```

- **Important Considerations**:
  - **Resource Safety**: When using expression statements to manipulate resources, such as moving or destroying them, Cadence ensures that these operations are safe and enforce resource ownership rules. After moving a resource in an expression statement, the original reference is no longer valid.
  - **Effect-Driven**: Expression statements are not used to produce values for later use; they are used specifically for their side effects. Any result produced by the expression itself is discarded.

Expression statements are an essential part of Cadence programming, enabling developers to perform actions that modify the program’s state, manage resources, or invoke functions without the need to store or use a return value. These statements are crucial when building contracts and transactions where operations such as transfers, logging, or emitting events need to occur based on specific conditions.

---

### 4. Expressions

Expressions in Cadence are the core building blocks that represent computations, operations, and evaluations. They can produce values, call functions, and manipulate data structures, such as arrays or resources. An expression typically evaluates to a result, which may be a primitive value (like an integer or boolean), a reference, or a more complex structure like a resource.

Expressions can be composed of literals, function calls, operators, and other expressions, allowing for flexible and concise computations in Cadence smart contracts and transactions. Expressions are used in a variety of contexts, including assignments, conditional checks, loops, and function arguments.

Cadence expressions follow a strict type system and enforce safe resource management to prevent errors such as duplication or unintended loss of resources. This ensures that expressions, especially those involving resources, behave predictably and securely.

  - **Types of Expressions**:
    - **Literals**: Represent simple constant values like numbers, booleans, strings, and arrays.
    - **Arithmetic and Logical Expressions**: Perform calculations and logical operations (e.g., addition, subtraction, conjunction, disjunction).
    - **Function Calls**: Invoke functions with arguments and return values.
    - **Conditional Expressions**: Evaluate based on a condition, such as using the ternary conditional operator.
    - **Resource Expressions**: Deal with Cadence’s resource-oriented features, including creating, moving, and destroying resources.
    - **Casting Expressions**: Allow safe type conversions and optional type checks with as, as?, and as! operators.

  - **Evaluation**:
    - Expressions are evaluated according to Cadence’s strict type system, ensuring that operations are type-safe and valid within the context of their use.
    - Expressions involving resources follow move semantics, where resources are transferred rather than copied, ensuring that ownership is preserved and no resource is inadvertently duplicated.

  - **Expression Composition**:
    - **Chaining expressions**: Cadence allows expressions to be nested or combined to perform more complex calculations or data manipulations.
    - **Operators**: Expressions can use a variety of operators, including arithmetic (+, -), comparison (==, !=), logical (&&, ||), and bitwise operators (&, |).

  - **Semantics**:
    - **Deterministic Evaluation**: Expressions are evaluated in a deterministic order, ensuring that results are predictable and reliable, especially in the context of smart contracts where transparency and auditability are critical.
    - **Resource Safety**: Expressions that manipulate resources are closely governed by Cadence’s rules on resource movement and ownership. This prevents issues like accidental loss or duplication of valuable assets.

  - **Examples**:
    - Simple Arithmetic:
      ```cadence
      let result = 5 + 3  // result evaluates to 8
      ```
    - Function Call:
      ```cadence
      let total = calculateSum(a: 5, b: 10)  // Calls the function to compute the sum
      ```
    - Conditional Expression:
      ```cadence
      let status = (score >= 50) ? "Pass" : "Fail"  // Ternary conditional expression
      ```

  - **Resource Expressions**:
    - In Cadence, resources are critical for managing assets, and expressions involving resources adhere to special rules to ensure their safe handling.
    - Example:
      ```cadence
      let token <- create Token()  // Resource creation
      account.save(<-token, to: /storage/myToken)  // Resource move expression
      ```

- **EBNF**:
  ```ebnf
  expressionStatement
    : expression
    ;
  ```

Expressions are fundamental to all Cadence programs, enabling developers to perform calculations, control logic, manipulate resources, and interact with data. The expressive power of Cadence is balanced by its strict rules on type safety and resource management, ensuring that expressions not only compute results but do so in a way that guarantees security and predictability in smart contract environments.

#### Conditional Expressions

Conditional expressions in Cadence allow a value to be selected based on a boolean condition. They provide a concise way to make decisions within an expression, often referred to as a ternary operator in other languages. A conditional expression evaluates a condition and returns one of two possible values depending on whether the condition evaluates to true or false.

Conditional expressions are useful for simplifying logic where a choice between two values needs to be made based on a condition, without requiring the use of a full if-else statement.

- **Syntax**:
  - The conditional expression uses the following structure:
    ```cadence
    condition ? expressionIfTrue : expressionIfFalse
    ```
  - The expression starts with a condition (a boolean expression), followed by a ?, then the expression to evaluate if the condition is true, followed by a :, and finally the expression to evaluate if the condition is false.

- **Semantics**:
  - **Condition**: The first part of the conditional expression is a boolean condition. If this condition evaluates to true, the expression following the ? is evaluated and returned. If it evaluates to false, the expression after the : is evaluated and returned.
  - **Result**: The result of the conditional expression is either the value of expressionIfTrue or expressionIfFalse, depending on the evaluation of the condition.
  - **Type Checking**: Both the expressionIfTrue and expressionIfFalse must have compatible types, as Cadence is a strongly-typed language. If the two expressions are not compatible, a type error will occur.

- **Usage**:
  - Conditional expressions are useful when you need to choose between two values or outcomes based on a simple condition, without the need for a full if-else statement.
  - They are typically used for assigning values to variables, passing arguments to functions, or simplifying return statements.

- **Example**:
  - Basic Conditional Expression:
    ```cadence
    let status = (score >= 50) ? "Pass" : "Fail"
    ```
    - In this example, if score is 50 or greater, status is set to "Pass". Otherwise, it is set to "Fail".

  - Another Example:
    ```cadence
    let max = (a > b) ? a : b
    ```
    - This expression assigns the larger of two values (a or b) to the variable max.

- **Comparison to if-else Statements**:
  - A conditional expression is a compact form of an if-else statement. For example:
    ```cadence
    if score >= 50 {
      status = "Pass"
    } else {
      status = "Fail"
    }
    ```
  - The above code can be written more concisely as:
    ```cadence
    let status = (score >= 50) ? "Pass" : "Fail"
    ```

- **EBNF**:
  ```ebnf
  conditionalExpression
    : orExpression ( '?' expression ':' expression )?
    ;
  ```

Conditional expressions offer a succinct way to perform branching within an expression. They simplify code by reducing the need for multiple lines of if-else statements, especially when deciding between two possible values. By ensuring both result expressions have compatible types, Cadence maintains type safety and guarantees predictable behavior in smart contracts.

#### Or Expressions

Or expressions in Cadence are used to combine two or more boolean expressions, returning true if at least one of the expressions evaluates to true. The logical OR operator (||) is employed in or expressions, and it provides short-circuiting behavior, meaning that if the first condition evaluates to true, the second condition is not evaluated, as the overall result is already determined.

Or expressions are useful when checking multiple conditions where only one of the conditions needs to be satisfied for the overall expression to be true.

- **Syntax**:
  - The logical OR operator (||) is placed between two boolean expressions.
  - Example:
    ```cadence
    conditionA || conditionB
    ```

- **Semantics**:
  - **Evaluation**: The || operator evaluates the left-hand side expression first. If the left-hand side evaluates to true, the right-hand side is not evaluated (short-circuit evaluation). If the left-hand side evaluates to false, the right-hand side expression is evaluated.
  - **Result**: The result of an or expression is a boolean (true or false). It will be true if either the left-hand side or the right-hand side is true. Otherwise, it will be false if both sides are false.
  - **Short-Circuiting**: This behavior is critical for efficiency and correctness, especially when the right-hand side expression may have side effects or is computationally expensive.

- **Usage**:
  - Or expressions are typically used in conditional statements like if or while loops, where multiple conditions are checked, and the action is taken if at least one of the conditions is satisfied.
  - Example:
    ```cadence
    if isAdmin || isOwner {
      performAdminAction()
    }
    ```
    - In this example, the action performAdminAction will be executed if either isAdmin or isOwner is true.

- **Examples**:
  - **Basic Or Expression**:
    ```cadence
    let isValid = (age >= 18) || (hasPermission)
    ```
    - This expression evaluates whether the user is at least 18 years old or has permission. The result is true if either condition is satisfied.

  - **Short-Circuiting**:
    ```cadence
    let result = (x > 0) || expensiveFunction()
    ```
    - If x > 0 evaluates to true, expensiveFunction() is not called, making the evaluation more efficient.

- **EBNF**:
  ```ebnf
  orExpression
    : andExpression
    | orExpression '||' andExpression
    ;
  ```

Or expressions allow Cadence developers to combine conditions logically and efficiently. The short-circuit evaluation provided by the || operator helps improve performance, especially in cases where evaluating the second condition is unnecessary or computationally expensive. This construct ensures concise and clear logic when building smart contracts and transaction flows that depend on multiple conditions.

#### And Expressions

And expressions in Cadence are used to combine two or more boolean expressions, where the overall result is true only if both expressions evaluate to true. The logical AND operator (&&) is employed in these expressions, and it provides short-circuiting behavior: if the first condition evaluates to false, the second condition is not evaluated because the result is already determined to be false.

And expressions are useful in scenarios where all conditions must be satisfied for an action to be performed or a result to be computed.

- **Syntax**:
  - The logical AND operator (&&) is placed between two boolean expressions.
  - Example:
    ```cadence
    conditionA && conditionB
    ```

- **Semantics**:
  - **Evaluation**: The && operator evaluates the left-hand side expression first. If the left-hand side evaluates to false, the right-hand side is not evaluated (short-circuit evaluation). If the left-hand side is true, then the right-hand side is evaluated.
  - **Result**: The result of an and expression is true if both the left-hand side and right-hand side expressions evaluate to true. Otherwise, the result is false.
  - **Short-Circuiting**: This ensures that if one condition is already known to be false, the other condition is not evaluated, potentially saving computation time and preventing unnecessary or expensive operations.

- **Usage**:
  - And expressions are typically used in conditional statements such as if, while, or for loops where all conditions need to be met before the block of code can be executed.
  - Example:
    ```cadence
    if isAdmin && hasPermission {
      executeRestrictedAction()
    }
    ```
    - In this example, the action executeRestrictedAction will only be performed if both isAdmin and hasPermission are true.

- **Examples*:
  - **Basic And Expression**:
    ```cadence
    let isEligible = (age >= 18) && (hasLicense)
    ```
    - This expression evaluates whether the user is at least 18 years old and has a valid license. The result will be true only if both conditions are met.

  - **Short-Circuiting**:
    ```cadence
    let result = (x > 0) && expensiveFunction()
    ```
    - If x > 0 evaluates to false, expensiveFunction() is not called, avoiding unnecessary computation.

  - **Multiple Conditions**:
    ```cadence
    if isAdmin && isActive && hasPermission {
      performAction()
    }
    ```
    - All three conditions (isAdmin, isActive, and hasPermission) must be true for performAction() to be executed.

- **EBNF**:
  ```ebnf
  andExpression
    : equalityExpression
    | andExpression '&&' equalityExpression
    ;
  ```

And expressions are essential when multiple conditions must be satisfied for a block of code to execute. The short-circuit evaluation provided by the && operator ensures that unnecessary evaluations are avoided, particularly when the first condition is already false. This makes and expressions efficient and well-suited for decision-making in Cadence smart contracts and transactions.

#### Equality Expressions

Equality expressions in Cadence are used to compare two values for equality or inequality. These expressions result in a boolean value (true or false) depending on whether the two values being compared are considered equal or unequal. Equality comparisons are commonly used in conditional statements, loops, and other logical constructs where decisions are based on whether values match or differ.

Cadence uses the equality operator (==) to check if two values are equal and the inequality operator (!=) to check if two values are different.

- **Syntax**:
  - The equality operator (==) checks whether two values are equal.
  - The inequality operator (!=) checks whether two values are not equal.
  - Example:
    ```cadence
    value1 == value2  // Checks if value1 is equal to value2
    value1 != value2  // Checks if value1 is not equal to value2
    ```

- **Semantics**:
  - **Equality (==)**: Compares two values for equality. If the values are of the same type and are equal, the result is true. Otherwise, the result is false.
  - **Inequality (!=)**: Compares two values for inequality. If the values are not equal or of incompatible types, the result is true. If they are equal, the result is false.
  - **Type Safety**: Cadence enforces strict type safety, meaning that both values being compared must be of compatible types. Comparing values of incompatible types results in a compile-time error.
  - **Reference Types and Resources**: When comparing reference types or resources, equality comparisons check whether the two references point to the same object or resource.

- **Usage**:
  - Equality expressions are often used in conditional statements, loops, or assertions where a comparison between values is required.
  - Example:
    ```cadence
    if accountBalance == requiredBalance {
      log("Balance is sufficient")
    }
    ```

- **Examples**:
  - **Simple Equality Comparison**:
    ```cadence
    let isEqual = (10 == 10)  // Evaluates to true
    ```
  - **Simple Inequality Comparison**:
    ```cadence
    let isNotEqual = (5 != 10)  // Evaluates to true
    ```
  - **Using Equality in a Conditional Statement**:
    ```cadence
    let age = 21
    if age == 18 {
      log("Age is 18")
    } else {
      log("Age is not 18")
    }
    ```
  - **Reference Equality**:
    ```cadence
    let tokenA: @Token <- create Token()
    let tokenB: @Token <- create Token()
    let areSame = tokenA == tokenB  // Evaluates to false since they are different resources
    ```

- **EBNF**:
  ```ebnf
  equalityExpression
    : relationalExpression
    | equalityExpression equalityOp relationalExpression
    ;
  
  equalityOp
    : Equal
    | Unequal
    ;
  
  Equal : '==' ;
  Unequal : '!=' ;
  ```

Equality expressions play an essential role in logic-driven programming in Cadence, allowing developers to make decisions based on the comparison of values. With strict type enforcement and the ability to compare both primitive types and references, equality expressions help ensure safe and predictable behavior in smart contracts and transactions.

#### Relational Expressions

Relational expressions in Cadence are used to compare two values based on their relative magnitude. These expressions evaluate to a boolean value (true or false) depending on whether one value is less than, greater than, or equal to another value in terms of ordering. Relational expressions are commonly used in decision-making processes, such as conditionals and loops, where the relative comparison between values determines the flow of the program.

The relational operators include < (less than), > (greater than), <= (less than or equal to), and >= (greater than or equal to).

- **Syntax**:
  - Relational expressions use the following operators:
    - `<`: Less than
    - `>`: Greater than
    - `<=`: Less than or equal to
    - `>=`: Greater than or equal to
  - Example:
    ```cadence
    value1 < value2    // Checks if value1 is less than value2
    value1 >= value2   // Checks if value1 is greater than or equal to value2
    ```

- **Semantics**:
  - **Relational Comparisons**: Relational expressions evaluate the relative order between two values. For instance, x < y returns true if x is less than y and false otherwise.
  - **Type Safety**: The values being compared must be of the same type, or types that are compatible with each other for comparison. Attempting to compare incompatible types results in a compile-time error.
  - **Numeric Comparisons**: These operators are typically used for numeric types such as integers or floating-point numbers.
  - **Lexicographical Comparisons**: In certain cases, strings can also be compared lexicographically (based on alphabetical order), though this behavior depends on the types involved.

- **Usage**:
  - Relational expressions are often used in control flow statements like if, while, and for loops to control program behavior based on comparisons.
  - Example:
    ```cadence
    if score >= 60 {
      log("Pass")
    } else {
      log("Fail")
    }
    ```

- **Examples**:
  - **Basic Relational Expression**:
    ```cadence
    let isSmaller = 5 < 10   // Evaluates to true
    let isGreaterOrEqual = 20 >= 20  // Evaluates to true
    ```
  - **Using Relational Expressions in Conditional Statements**:
    ```cadence
    let temperature = 25
    if temperature > 30 {
      log("It's hot")
    } else if temperature >= 20 {
      log("It's warm")
    } else {
      log("It's cold")
   }
    ```
  - **Using Relational Expressions in Loops**:
    ```cadence
    var counter = 0
    while counter < 10 {
      log(counter)
      counter = counter + 1
    }
    ```

- **Important Considerations**:
  - **Type Matching**: Ensure that the values being compared are of compatible types. For example, comparing an integer with a floating-point number may require explicit type conversion.
  - **Resource Safety**: When dealing with resources, relational operators are not used since resources are not compared based on value but on ownership and movement.

- **EBNF**:
  ```ebnf
  relationalExpression
    : nilCoalescingExpression
    | relationalExpression relationalOp nilCoalescingExpression
    ;
  
  relationalOp
    : Less
    | Greater
    | LessEqual
    | GreaterEqual
    ;
  
  Less : '<' ;
  Greater : '>' ;
  LessEqual : '<=' ;
  GreaterEqual : '>=' ;
  ```

Relational expressions provide the foundation for comparing values in Cadence programs. They are critical for controlling the flow of logic based on the relative size or order of variables, particularly in loops and conditional statements. By ensuring type safety and predictable behavior, relational expressions help build robust smart contracts and decentralized applications.

#### Nil Coalescing Expressions

Nil coalescing expressions in Cadence provide a mechanism to safely handle optional values by supplying a default value if an optional is nil (i.e., does not contain a value). This expression uses the nil coalescing operator (??), which allows developers to specify a fallback value to be used when the optional value is nil. If the optional contains a value, it is unwrapped and returned; if the optional is nil, the fallback value is used.

Nil coalescing expressions simplify the handling of optionals and ensure that a program can proceed with a valid value even when an optional is empty.

- **Syntax**:
  - The nil coalescing operator (??) is placed between an optional expression and a fallback value.
  - Example:
    ```cadence
    let result = optionalValue ?? defaultValue
    ```

- **Semantics**:
  - **Optional Unwrapping**: If the expression on the left-hand side of the ?? operator evaluates to an optional that contains a value, that value is returned.
  - **Fallback Value**: If the left-hand side evaluates to nil, the right-hand side expression (the fallback value) is evaluated and returned.
  - **Type Safety**: The result of a nil coalescing expression must have the same type as the optional’s unwrapped value or the fallback value. Both expressions must be of compatible types to ensure type safety.
  - **Short-Circuiting**: The fallback value is only evaluated if the optional on the left-hand side is nil.

- **Usage**:
  - Nil coalescing expressions are typically used when dealing with optionals to ensure that the program can continue with a valid value even if the optional is empty. This is particularly useful when accessing optional data that may or may not be available, such as reading from storage or external data sources.
  - Example:
    ```cadence
    let displayName = userInput ?? "Guest"
    ```
    - In this example, if userInput is nil, the default string "Guest" is used as the value of displayName.

- **Examples**:
  - **Using a Fallback Value**:
    ```cadence
    let optionalValue: Int? = nil
    let value = optionalValue ?? 10   // Since optionalValue is nil, value is set to 10
    ```
  - **With Non-Nil Optional**:
    ```cadence
    let optionalValue: Int? = 5
    let value = optionalValue ?? 10   // optionalValue contains 5, so value is set to 5
    ```
  - **Combining Multiple Coalescing Expressions**:
    ```cadence
    let result = firstOptional ?? secondOptional ?? defaultValue
    ```
    - This expression checks firstOptional first. If it is nil, it checks secondOptional, and if that is also nil, it uses defaultValue.

- **EBNF**:
  ```ebnf
  nilCoalescingExpression
    : bitwiseOrExpression ( '??' nilCoalescingExpression )?
    ;
  ```

- **Important Considerations**:
  - **Avoiding Forced Unwrapping**: Nil coalescing provides a safer alternative to forced unwrapping (!), as it avoids runtime crashes by always providing a fallback value.
  - **Type Matching**: Ensure that both the optional value and the fallback value are of the same or compatible types, as Cadence requires strict type checking for nil coalescing expressions.

Nil coalescing expressions are a powerful tool for handling optionals safely and concisely in Cadence. By allowing developers to provide default values when an optional is nil, they eliminate the need for explicit checks and unwrapping, simplifying code and improving safety in smart contracts.

#### Bitwise Or Expressions

Bitwise OR expressions in Cadence operate on the binary representation of integer values, comparing corresponding bits of two integers and returning a new integer where each bit is set to 1 if at least one of the corresponding bits of the operands is 1. The bitwise OR operator (|) is used for this operation. This operator is often utilized in low-level programming tasks like manipulating binary data, flags, or encoding.

In a bitwise OR operation, each bit in the resulting number corresponds to the logical OR of the bits at the same position in the operands.

- **Syntax**:
  - The bitwise OR operator (|) is placed between two integer expressions.
  - Example:
    ```cadence
    let result = value1 | value2
    ```

- **Semantics**:
  - **Bitwise OR Operation**: For each bit in the operands, the result is 1 if at least one of the corresponding bits in either operand is 1; otherwise, the result is 0.
  - **Operands**: The operands must be integers, and the result is an integer with the same bit-length as the operands.
  - **Use Cases**: Bitwise OR expressions are commonly used in systems programming, flag management, and binary data manipulation where bits are treated individually rather than as a whole value.

- **Usage**:
  - Bitwise OR expressions are often used in scenarios where you need to combine or set specific bits in binary data. They are also useful when working with flags that represent multiple boolean conditions in a single integer.
  - Example:
    ```cadence
    let permissionsA: UInt8 = 0b1100    // Binary: 1100
    let permissionsB: UInt8 = 0b1010    // Binary: 1010
    let combinedPermissions = permissionsA | permissionsB   // Result: 1110 (binary)
    ```
    - In this example, the bitwise OR of permissionsA and permissionsB results in 0b1110, which sets each bit that is 1 in either of the original values.

- **Examples**:
  - **Basic Bitwise OR**:
    ```cadence
    let result = 12 | 10   // 12 (1100 in binary) | 10 (1010 in binary) = 14 (1110 in binary)
    ```
  - **Combining Flags**:
    ```cadence
    let readPermission: UInt8 = 0b0001   // Read flag (binary 0001)
    let writePermission: UInt8 = 0b0010  // Write flag (binary 0010)
    let combined = readPermission | writePermission  // Combined flags: 0011 (read and write)
    ```
  - **Short-Circuiting**:
    - Unlike logical OR expressions, bitwise OR does not short-circuit. All bits of both operands are evaluated and compared.

- **EBNF**:
  ```ebnf
  bitwiseOrExpression
    : bitwiseXorExpression
    | bitwiseOrExpression '|' bitwiseXorExpression
    ;
  ```

- **Important Considerations**:
	•	**Type Matching**: The operands of bitwise OR expressions must be of integer types. Attempting to use non-integer values results in a compile-time error.
	•	**Binary Operations**: Bitwise operations operate at the level of individual bits, so they may be confusing if unfamiliar with binary arithmetic. Make sure that bitwise operations are appropriate for your task.

Bitwise OR expressions are a powerful tool for manipulating binary data and performing low-level operations in Cadence. By applying the OR operator on individual bits, these expressions allow developers to combine or modify specific bits of integer values, which is particularly useful for working with flags, binary protocols, or optimizing memory usage.

#### Bitwise Xor Expressions

Bitwise XOR (exclusive OR) expressions in Cadence perform a binary operation that compares corresponding bits of two integer values. The XOR operator (^) results in a bit being set to 1 if exactly one of the corresponding bits of the operands is 1, and 0 if both bits are the same (either both 0 or both 1). This operation is useful in scenarios where bits need to be toggled or flipped and is commonly used in cryptographic algorithms, binary manipulation, and flag-based programming.

In a bitwise XOR operation, the result is based on the difference between the bits at the same position in the operands.

- **Syntax**:
  - The bitwise XOR operator (^) is placed between two integer expressions.
  - Example:
    ```cadence
    let result = value1 ^ value2
    ```

- **Semantics**:
  - **Bitwise XOR Operation**: For each bit in the operands, the result is 1 if the bits at the same position are different (1 and 0 or 0 and 1). If the bits are the same (0 and 0 or 1 and 1), the result is 0.
  - **Operands**: The operands must be integers, and the result is an integer.
  - **Toggling Bits**: One common use case for bitwise XOR is toggling bits in binary data, where applying XOR with 1 flips the bit (from 0 to 1 or from 1 to 0).

- **Usage**:
  - Bitwise XOR expressions are typically used in scenarios where you need to compare or manipulate individual bits of integer values. XOR is often applied in cryptography, checksum calculations, or when performing bitwise toggling of flags.
  - Example:
    ```cadence
    let a: UInt8 = 0b1101   // Binary: 1101
    let b: UInt8 = 0b1011   // Binary: 1011
    let result = a ^ b   // Result: 0110 (binary)
    ```
    - In this example, the bitwise XOR of a and b results in 0b0110, as the differing bits in the two operands are highlighted.

- **Examples**:
  - **Basic Bitwise XOR**:
    ```cadence
    let result = 12 ^ 10   // 12 (1100 in binary) ^ 10 (1010 in binary) = 6 (0110 in binary)
    ```
  - **Toggling Bits*:
    ```cadence
    let flag: UInt8 = 0b1010
    let toggleMask: UInt8 = 0b0011
    let toggled = flag ^ toggleMask   // Result: 1001 (binary)
    ```

- **Short-Circuiting**:
  - Like bitwise OR, bitwise XOR does not short-circuit; all bits of both operands are evaluated.

- **EBNF**:
  ```ebnf
  bitwiseXorExpression
    : bitwiseAndExpression
    | bitwiseXorExpression '^' bitwiseAndExpression
    ;
  ```

- **Important Considerations**:
  - **Type Matching**: The operands of bitwise XOR expressions must be of integer types. If you attempt to use non-integer values, a compile-time error will occur.
  - **Practical Usage**: XOR is commonly used in tasks like toggling bits, combining flags, and implementing certain cryptographic algorithms or checksums.

Bitwise XOR expressions offer powerful operations for manipulating individual bits in integer values. By using the XOR operator, developers can efficiently toggle, compare, or mask bits, making it particularly useful for low-level programming, binary data manipulation, and encryption techniques.

#### Bitwise And Expressions

Bitwise AND expressions in Cadence perform a binary operation that compares corresponding bits of two integer values. The AND operator (&) results in a bit being set to 1 only if both corresponding bits of the operands are 1. If either of the bits is 0, the resulting bit will be 0. This operation is often used for masking bits, clearing specific bits in a value, or checking whether certain bits are set.

In a bitwise AND operation, the result is determined by the intersection of the bits at the same position in both operands.

- **Syntax**:
  - The bitwise AND operator (&) is placed between two integer expressions.
  - Example:
    ```cadence
    let result = value1 & value2
    ```

- **Semantics**:
  - **Bitwise AND Operation**: For each bit in the operands, the result is 1 only if both corresponding bits are 1. If one or both bits are 0, the result is 0.
  - **Operands**: The operands must be integers, and the result is an integer.
  - **Use Cases**: Bitwise AND is commonly used for bit masking, where specific bits of an integer are isolated or manipulated while leaving others unchanged.

- **Usage**:
  - Bitwise AND expressions are typically used for tasks such as checking specific bits in a binary number, masking certain bits, or combining flags where only overlapping bits matter.
  - Example:
    ```cadence
    let mask: UInt8 = 0b1110   // Binary mask: 1110
    let value: UInt8 = 0b1011  // Binary value: 1011
    let result = value & mask  // Result: 1010 (binary)
    ```
    - In this example, the bitwise AND of value and mask results in 0b1010, preserving only the bits where both values have 1 at the same position.

- **Examples**:
  - **Basic Bitwise AND**:
    ```cadence
    let result = 12 & 10   // 12 (1100 in binary) & 10 (1010 in binary) = 8 (1000 in binary)
    ```
  - **Using Bitwise AND to Mask Bits**:
    ```cadence
    let flags: UInt8 = 0b1101    // Flag with various bits set
    let mask: UInt8 = 0b0100     // Mask to check the third bit
    let maskedFlags = flags & mask   // Result: 0100 (binary), isolating the third bit
    ```
  - **Checking if a Specific Bit is Set**:
    ```cadence
    let status: UInt8 = 0b1010
    let checkBit: UInt8 = 0b0010
    if status & checkBit != 0 {
      log("The bit is set")
    } else {
      log("The bit is not set")
    }
    ```

- **Short-Circuiting**:
  - Unlike logical AND, bitwise AND does not short-circuit. All bits of both operands are evaluated.

- **EBNF**:
  ```ebnf
  bitwiseAndExpression
    : bitwiseShiftExpression
    | bitwiseAndExpression '&' bitwiseShiftExpression
    ;
  ```

- **Important Considerations**:
  - **Type Matching**: Both operands of bitwise AND expressions must be of integer types. Attempting to use non-integer values results in a compile-time error.
  - **Common Applications**: Bitwise AND is widely used in low-level programming, such as in cryptography, network protocols, and hardware interfacing, where binary data needs to be manipulated directly.

Bitwise AND expressions are a key tool for working with binary data in Cadence. By using the AND operator, developers can mask specific bits, check whether certain bits are set, or clear bits in a value. This makes bitwise AND especially useful in systems programming, flag management, and tasks that require direct control over binary representations.

#### Bitwise Shift Expressions

Bitwise shift expressions in Cadence perform shifting operations on the binary representation of integer values. These shifts move the bits of a number to the left or right, effectively multiplying or dividing the number by powers of two. The left shift operator (<<) shifts bits to the left, while the right shift operator (>>) shifts bits to the right.

Bitwise shifts are often used in low-level programming, such as optimizing arithmetic operations, manipulating binary data, or implementing certain algorithms where binary representation is key.

- **Syntax**:
  - The left shift operator (<<) shifts bits to the left by a specified number of positions.
  - The right shift operator (>>) shifts bits to the right by a specified number of positions.
  - Example:
    ```cadence
    let result = value << shiftAmount   // Left shift
    let result = value >> shiftAmount   // Right shift
    ```

- **Semantics**:
  - **Left Shift (<<)**: Shifts the bits of the number to the left by the specified number of positions. Each left shift by one position is equivalent to multiplying the number by 2. Bits that are shifted out on the left are discarded, and the vacated positions on the right are filled with 0.
  - **Right Shift (>>)**: Shifts the bits of the number to the right by the specified number of positions. Each right shift by one position is equivalent to dividing the number by 2 (rounding down). For unsigned integers, bits shifted out on the right are discarded, and the vacated positions on the left are filled with 0. For signed integers, the behavior depends on whether the right shift is arithmetic or logical (Cadence typically uses logical shifts for unsigned types).
  - **Operands**: The value to be shifted must be an integer, and the shift amount must also be an integer.

- **Usage**:
  - Bitwise shift expressions are commonly used for efficiently multiplying or dividing numbers by powers of two, manipulating bits in binary data, or compressing and decompressing information stored in bit fields.
  - Example:
    ```cadence
    let value: UInt8 = 0b0011_1100
    let shiftedLeft = value << 2  // Result: 1111_0000 (binary), equivalent to multiplying by 4
    let shiftedRight = value >> 2 // Result: 0000_1111 (binary), equivalent to dividing by 4
    ```

- **Examples**:
  - Basic Left Shift:
    ```cadence
    let value: UInt8 = 4   // 0000_0100 in binary
    let result = value << 1  // Result: 8 (0000_1000 in binary), equivalent to 4 * 2
    ```
  - Basic Right Shift:
    ```cadence
    let value: UInt8 = 8   // 0000_1000 in binary
    let result = value >> 1  // Result: 4 (0000_0100 in binary), equivalent to 8 / 2
    ```
  - Shifting with Larger Values:
    ```cadence
    let bigValue: UInt16 = 1024   // 00000100_00000000 in binary
    let resultLeft = bigValue << 3  // Result: 8192 (binary shift left by 3)
    let resultRight = bigValue >> 3 // Result: 128 (binary shift right by 3)
    ```

- **Important Considerations**:
  - **Overflow and Underflow**: Shifting a number too far to the left can cause overflow, where significant bits are lost. Similarly, shifting too far to the right can reduce a number to zero. Care should be taken to ensure that the shift amount is appropriate for the bit width of the number.
  - **Unsigned vs. Signed Shifts**: When using signed integers, shifting can affect how negative numbers are handled. In many languages, right shifting a signed number preserves the sign bit (arithmetic shift), but in Cadence, shifts are typically logical, meaning 0 is shifted into the leftmost position.
  - **Efficiency**: Bitwise shifts are computationally efficient operations and are often used in performance-critical code to replace multiplication or division by powers of two.

- **EBNF**:
  ```ebnf
  bitwiseShiftExpression
    : additiveExpression
    | bitwiseShiftExpression bitwiseShiftOp additiveExpression
    ;

  bitwiseShiftOp
    : ShiftLeft
    | ShiftRight
    ;

  ShiftLeft : '<<' ;
  ShiftRight : '>>' ;
  ```

Bitwise shift expressions provide an efficient way to manipulate integer values at the bit level, offering a powerful tool for optimizing arithmetic operations and working with binary data. Left and right shifts can be used to efficiently multiply, divide, or adjust the position of bits, making them invaluable in low-level programming tasks, algorithms, and performance-sensitive applications.

#### Additive Expressions

Additive expressions in Cadence involve performing addition and subtraction on numerical values. These expressions use the addition operator (+) and the subtraction operator (-) to combine or subtract values. Additive expressions can be used with various numeric types, such as integers and floating-point numbers, and they follow the standard mathematical rules of addition and subtraction.

Additive expressions are fundamental for performing calculations, manipulating data, and computing values within smart contracts and transactions.

- **Syntax**:
  - The addition operator (+) is used to add two numerical values.
  - The subtraction operator (-) is used to subtract one numerical value from another.
  - Example:
    ```cadence
    let sum = value1 + value2   // Addition
    let difference = value1 - value2   // Subtraction
    ```

- **Semantics**:
  - **Addition (+)**: Adds two numbers and produces the sum.
  - **Subtraction (-)**: Subtracts the second number from the first and produces the difference.
  - **Type Safety**: Both operands in an additive expression must be of compatible numeric types (e.g., integers, floating-point numbers). Cadence enforces strict type checking to ensure that both operands are compatible.
  - **Order of Evaluation**: If multiple additive expressions are chained, they are evaluated from left to right. Parentheses can be used to explicitly group expressions and control the order of evaluation.

- **Usage**:
  - Additive expressions are commonly used for basic arithmetic operations, such as calculating totals, differences, or incremental values.
  - Example:
    ```cadence
    let total = price + tax
    let remainingBalance = balance - withdrawalAmount
    ```

- **Examples**:
  - **Basic Addition**:
    ```cadence
    let sum = 5 + 3   // Result: 8
    ```
  - **Basic Subtraction**:
    ```cadence
    let difference = 10 - 4   // Result: 6
    ```
  - **Chaining Additive Expressions**:
    ```cadence
    let result = 5 + 3 - 2   // Evaluates to 6
    ```
  - **Using Additive Expressions in Complex Calculations**:
    ```cadence
    let price: UFix64 = 100.50
    let discount: UFix64 = 20.00
    let totalPrice = price - discount   // Subtracts the discount from the price
    ```

- **Important Considerations**:
	•	Type Matching: Ensure that the operands involved in an additive expression are of compatible types. For example, adding an integer to a floating-point number may require explicit type conversion to avoid errors.
	•	Overflow and Underflow: Be cautious of potential overflows when performing addition or underflows when performing subtraction, especially when working with unsigned types like UInt.

- **EBNF**:
  ```ebnf
  additiveExpression
    : multiplicativeExpression
    | additiveExpression additiveOp multiplicativeExpression
    ;
  
  additiveOp
    : Plus
    | Minus
    ;
  
  Plus : '+' ;
  Minus : '-' ;
  ```

Additive expressions are a fundamental part of any programming language, enabling basic arithmetic operations. In Cadence, they allow developers to perform addition and subtraction on numeric values in a type-safe manner, ensuring that calculations are precise and reliable in smart contracts and transactions. These expressions are key to performing financial operations, calculating balances, and managing numerical data in decentralized applications.

#### Multiplicative Expressions

Multiplicative expressions in Cadence involve performing multiplication, division, and modulus operations on numerical values. These expressions use the multiplication operator (*), division operator (/), and modulus operator (%) to manipulate numbers in various ways. These operators form the basis of more complex arithmetic, allowing for scaling values, dividing them, or finding remainders.

Multiplicative expressions are essential for calculations such as computing totals, ratios, percentages, and other forms of mathematical operations within smart contracts.

- **Syntax**:
  - Multiplication (*) multiplies two numeric values.
  - Division (/) divides one numeric value by another.
  - Modulus (%) computes the remainder of division between two integers.
  - Example:
    ```cadence
    let product = value1 * value2   // Multiplication
    let quotient = value1 / value2  // Division
    let remainder = value1 % value2 // Modulus (remainder after division)
    ```

- **Semantics**:
  - **Multiplication (*)**: Multiplies two numbers and returns the product.
  - **Division (/)**: Divides one number by another and returns the quotient. If the divisor is zero, a runtime error occurs (division by zero).
  - **Modulus (%)**: Returns the remainder when one integer is divided by another. Both operands must be integers.
  - **Type Safety**: Cadence enforces strict type safety in multiplicative expressions. Both operands must be of compatible numeric types (e.g., integers or floating-point numbers). Mixing types without conversion results in a compile-time error.
  - **Order of Operations**: Multiplicative operations have a higher precedence than additive operations. Parentheses can be used to explicitly control the order of evaluation.

- **Usage**:
  - Multiplicative expressions are used in various scenarios, such as calculating product totals, determining ratios or percentages, and finding remainders when dividing integers.
  - Example:
    ```cadence
    let totalCost = price * quantity
    let average = totalSum / itemCount
    let remainder = totalItems % 3
    ```

- **Examples**:
  - **Basic Multiplication**:
    ```cadence
    let product = 6 * 3   // Result: 18
    ```
  - **Basic Division**:
    ```cadence
    let quotient = 10 / 2   // Result: 5
    ```
  - **Basic Modulus**:
    ```cadence
    let remainder = 10 % 3   // Result: 1 (remainder when 10 is divided by 3)
    ```
  - **Using Multiplicative Expressions in Calculations**:
    ```cadence
    let price: UFix64 = 50.00
    let quantity: Int = 4
    let totalPrice = price * UFix64(quantity)  // Multiply price by quantity
    ```
  - **Combining with Other Operations**:
    ```cadence
    let result = (5 + 3) * 2  // Evaluates to 16 (parentheses used to control order)
    ```

- **Important Considerations**:
  - Division by Zero: Division by zero results in a runtime error. Ensure that the divisor in any division operation is checked before performing the operation.
  - Integer Division: When dividing integers, the result is also an integer, meaning any fractional part is discarded (rounded down). For floating-point division, use appropriate types such as Fix64 or UFix64.
  - Type Matching: Ensure that both operands are of the same or compatible types when performing multiplicative operations. Explicit type conversion may be needed when combining different numeric types.

- **EBNF**:
  ```ebnf
  multiplicativeExpression
    : castingExpression
    | multiplicativeExpression multiplicativeOp castingExpression
    ;
  
  multiplicativeOp
    : Mul
    | Div
    | Mod
    ;
  
  Mul : '*' ;
  Div : '/' ;
  Mod : '%' ;
  ```

Multiplicative expressions provide essential arithmetic functionality for smart contracts and transactions in Cadence. Whether calculating totals, dividing values, or finding remainders, these expressions enable developers to handle numerical data in a precise and type-safe way. Proper use of multiplicative operations ensures that smart contracts can perform mathematical calculations reliably and securely.

#### Casting Expressions

Casting expressions in Cadence are used to convert a value from one type to another. Cadence provides three types of casting: safe casting, failable casting, and force casting, each with different behaviors and guarantees regarding the success or failure of the cast. These casting mechanisms allow developers to work with values more flexibly while ensuring type safety, minimizing runtime errors, and handling optional values appropriately.

- **Safe Casting (as)**: Converts a value from one type to another, provided the conversion is guaranteed to succeed.
- **Failable Casting (as?)**: Attempts to cast a value and returns nil if the cast fails. This is useful when the cast may not always be possible.
- **Force Casting (as!)**: Forcefully converts a value to a new type. If the cast is not possible, it results in a runtime error.

- **Syntax**:
  - **Safe Casting (as)**: Used when the cast is guaranteed to succeed.
    ```cadence
    let convertedValue = originalValue as TargetType
    ```
  - **Failable Casting (as?)**: Used when the cast may fail, returning an optional value (nil if the cast fails).
    ```cadence
    let optionalValue = originalValue as? TargetType
    ```
  - **Force Casting (as!)**: Used when the developer is certain the cast will succeed, but a runtime error occurs if it does not.
    ```cadence
    let forcedValue = originalValue as! TargetType
    ```

- **Semantics**:
  - **Safe Casting (as)**: This is used when the conversion between the original type and the target type is guaranteed to be valid at compile time or runtime. No runtime failure will occur if the cast is valid.
  - **Failable Casting (as?)**: When there is uncertainty about whether a value can be cast to a specific type, this casting returns an optional value (nil if the cast fails). This allows safe handling of potential casting failures.
  - **Force Casting (as!)**: This casts the value forcefully and will raise a runtime error if the cast fails. This method should only be used when the developer is certain that the cast will succeed, typically after performing checks or validations.

- **Usage**:
  - Casting expressions are commonly used when dealing with polymorphism (e.g., casting between parent and child types), handling dynamic data, or working with optionals where the exact type is not known at compile time.
  - **Safe Casting Example**:
    ```cadence
    let floatValue: AnyStruct = 10.5
    let number = floatValue as UFix64   // Cast is guaranteed to succeed
    ```
  - **Failable Casting Example**:
    ```cadence
    let unknownValue: AnyStruct = 10
    let optionalString = unknownValue as? String   // Failable cast, returns nil if not a String
    if let validString = optionalString {
      log(validString)   // Safe to use, cast succeeded
    } else {
      log("Not a valid string")
    }
    ```
  - **Force Casting Example**:
    ```cadence
    let value: AnyStruct = "Hello"
    let stringValue = value as! String   // Force cast, throws error if not a String
    ```

- **Examples**:
  - **Casting Between Compatible Types**:
    ```cadence
    let number: UFix64 = 5.0
    let intValue = number as UInt64   // Safely cast from UFix64 to UInt64
    ```
  - **Using Failable Casts for Optional Handling**:
    ```cadence
    let someValue: AnyStruct = "A string"
    let optionalInt = someValue as? Int
    if optionalInt != nil {
      log("Successfully cast to Int")
    } else {
      log("Value is not an Int")
    }
    ```
  - **Force Casting After a Type Check**:
    ```cadence
    let anyValue: AnyStruct = 42
    if let number = anyValue as? Int {
      let forcedValue = anyValue as! Int   // Safe after the type check
      log(forcedValue.toString())
    }
    ```

- **EBNF**:
  ```ebnf
  castingExpression
    : unaryExpression
    | castingExpression castingOp typeAnnotation
    ;
  
  castingOp
    : Casting
    | FailableCasting
    | ForceCasting
    ;
  
  Casting : 'as' ;
  FailableCasting : 'as?' ;
  ForceCasting : 'as!' ;
  ```

- **Important Considerations**:
  - **Safety**: Prefer using safe casting (as) or failable casting (as?) when there is uncertainty about the type conversion. Avoid force casting (as!) unless absolutely necessary, as it can lead to runtime errors.
  - **Optionals**: Failable casting returns an optional value, and you should always handle the possibility of nil when using it.
  - **Compile-time vs. Runtime**: Safe casting ensures correctness at compile time, while failable and force casting introduce the risk of runtime errors, so care must be taken when using them.

Casting expressions provide flexibility in handling different types and ensuring type safety in Cadence. Whether safely converting types, handling optional values with failable casting, or using force casting in specific scenarios, casting expressions are essential for dealing with polymorphism, dynamic data, and type conversions in smart contracts and decentralized applications.

#### Unary Expressions

Unary expressions in Cadence involve applying a single operator to a single operand (a value or variable) to perform operations like negation, logical inversion, or resource movement. These operators modify the value or resource in specific ways, depending on the operation. Unary expressions are crucial for concise calculations, logical evaluations, and safe resource handling in Cadence smart contracts.

- **Syntax**:
  - A unary operator is applied to an expression, modifying its value or resource.
  - Multiple unary operators can be applied in combination.
  - Example:
    ```cadence
    let result = -value      // Unary negation
    let flag = !condition    // Logical NOT
    let resource <- resource // Move a resource
    ```

- **Unary Operators**:
  - **Negation (-)**: Inverts a numeric value, converting positive values to negative, and negative values to positive.
  - **Logical NOT (!)**: Inverts a boolean value (true becomes false, and false becomes true).
  - **Move Operator (<-)**: Transfers ownership of a resource to a new location, ensuring that resources are safely handled in Cadence’s resource-oriented system.

- **Semantics**:
  - **Negation (-)**: This operator flips the sign of a numeric value.
    - Example:
      ```cadence
      let value: Int = 10
      let negativeValue = -value   // Result: -10
      ```
  - **Logical NOT (!)**: This operator inverts the truth value of a boolean expression. If the value is true, it becomes false, and vice versa.
    - Example:
      ```cadence
      let isActive = true
      let isInactive = !isActive   // Result: false
      ```
  - **Move (<-)**: This operator moves a resource from one variable or storage location to another. It enforces Cadence’s strict resource management rules, ensuring resources are neither duplicated nor implicitly destroyed.
    - Example:
      ```cadence
      let token <- create Token()
      account.save(<-token, to: /storage/myToken)  // Move the resource into storage
      ```

- **Usage**:
  - **Negation**: Used for inverting numeric values.
  - **Logical NOT**: Used for boolean logic and condition checks.
  - **Move**: Used for transferring resources in Cadence’s resource-oriented system.
  - Example:
    ```cadence
    let balance = 100
    let negativeBalance = -balance   // Converts 100 to -100
    
    let isValid = false
    let result = !isValid   // Converts false to true
    
    let asset <- create Asset()
    account.save(<-asset, to: /storage/myAsset)   // Move a resource
    ```

- **Examples**:
  - **Unary Negation**:
    ```cadence
    let number = 5
    let result = -number   // Result: -5
    ```
  - **Unary Logical NOT**:
    ```cadence
    let isFinished = false
    let result = !isFinished   // Result: true
    ```
  - **Unary Move**:
    ```cadence
    let nft <- create NFT()
    recipient.save(<-nft, to: /storage/nftStorage)   // Move the resource
    ```

- **EBNF**:
  ```ebnf
  unaryExpression
    : primaryExpression
    | unaryOp+ unaryExpression
    ;
  
  unaryOp
    : Minus
    | Negate
    | Move
    ;
  
  Minus : '-' ;
  Negate : '!' ;
  Move : '<-' ;
  ```

- **Important Considerations**:
  - Type Matching: The - operator is only valid for numeric types, the ! operator for boolean types, and the <- operator for resource types. Cadence enforces type safety, so using an operator with incompatible types will result in a compile-time error.
  - Resource Safety: The Move operator ensures that resources are never duplicated or lost, strictly enforcing Cadence’s resource ownership rules.

Unary expressions in Cadence provide essential operations for handling numbers, booleans, and resources. They allow developers to modify values and safely manage resource transfers, which are critical for building secure and efficient smart contracts on the Flow blockchain.

#### Primary Expressions

Primary expressions in Cadence are the basic building blocks of more complex expressions. They include operations for creating and destroying resources, referencing values or resources, and invoking various operations through postfix expressions (such as function calls or accessing elements in arrays). These expressions are fundamental to many operations within Cadence, especially when dealing with resources and object manipulation in smart contracts.

- Types of Primary Expressions:
  - **Create expressions**: Used to instantiate new resources or objects.
  - **Destroy expressions**: Used to explicitly destroy resources and ensure that they are properly disposed of in Cadence’s resource-oriented system.
  - **Reference expressions**: Used to create references to existing values or resources, allowing access without ownership transfer.
  - **Postfix expressions**: Used to perform operations such as function calls, accessing fields of structs, or indexing arrays.

- **EBNF**:
  ```ebnf
  primaryExpression
    : createExpression
    | destroyExpression
    | referenceExpression
    | postfixExpression
    ;
  ```

#### Create Expressions

Create expressions in Cadence are used to instantiate new resources or composite objects, typically using the create keyword followed by the type and a constructor invocation. In Cadence’s resource-oriented programming model, resources play a key role, and create expressions allow developers to initialize resources and ensure they follow the strict ownership and safety rules enforced by the language.

- **Syntax**:
  - A create expression starts with the create keyword, followed by the nominal type (the resource or composite type to be created) and a constructor invocation.
  - Example:
    ```cadence
    let newToken <- create Token(initialSupply: 1000)
    ```

- **Semantics**:
  - **Creation of Resources**: Create expressions are primarily used to create new resources, such as tokens, NFTs, or custom-defined resource types. Once created, resources must be handled safely, either by being moved, stored, or destroyed to prevent resource loss.
  - **Ownership**: When a resource is created, ownership of the resource is transferred. In the example above, the newly created Token is immediately assigned to newToken, making newToken the owner of that resource. The resource can later be moved to storage or transferred to another account.
  - **Invocation of Constructors**: The invocation part of the create expression invokes the constructor of the resource or composite type being created. This is where initial values for the resource’s fields can be provided.

- **Usage**:
  - **Creating Resources**: Create expressions are used to instantiate resources that will be used in a Cadence smart contract or transaction.
  - Example:
    ```cadence
    let newVault <- create Vault(balance: 1000)
    ```

- **Examples*:
  - **Simple Create Expression**:
    ```cadence
    let newToken <- create Token(initialSupply: 1000)
    ```
    - This expression creates a new Token resource with an initial supply of 1000 units and transfers ownership of it to the newToken variable.
  - **Creating an NFT**:
    ```cadence
    let nft <- create NFT(id: 1, metadata: {"name": "Rare Art"})
    ```
    - In this example, an NFT resource is created with an ID of 1 and metadata containing a name. Ownership of the newly created NFT is assigned to nft.
  - **Creating a Composite Object**:
    ```cadence
    let user = create User(name: "Alice", age: 30)
    ```
    - This creates an instance of the User composite type with the specified name and age.

- **EBNF**:
  ```ebnf
  createExpression
    : 'create' nominalType invocation
    ;
  ```

- **Important Considerations**:
  - **Resource Safety**: Since Cadence enforces strict resource management rules, every resource created via a create expression must either be moved to another variable, stored in account storage, or destroyed. Resources cannot be copied or discarded, ensuring safety and preventing resource leakage.
  - **Move Semantics**: After creation, the resource is treated as owned and must follow Cadence’s move semantics. If the resource is not moved or destroyed, Cadence will raise an error to prevent potential resource loss.

Create expressions are essential in Cadence for creating new resources, such as tokens or NFTs, ensuring that they are safely managed according to the language’s ownership and resource movement rules. By controlling how resources are instantiated and transferred, Cadence ensures resource safety and integrity in smart contracts.

#### Postfix Expressions

Postfix expressions in Cadence allow additional operations to be performed after an initial primary expression, such as invoking functions, accessing array elements or object fields, or force unwrapping optionals. Postfix expressions follow a base expression and modify its behavior or access its properties, making them essential for interacting with data and manipulating objects in Cadence smart contracts.

- **Types of Postfix Expressions**:

1. **Identifiers and Literals**:
    - Identifiers represent variables, constants, or fields in a contract or script.
    - Literals are constant values like numbers, booleans, strings, etc.
    - Example:
    ```cadence
    let balance = 100
    let isActive = true
    ```

2. **Function Expressions**:
    - Anonymous functions can be defined directly using the fun keyword, followed by parameters, and optionally a return type and a function block.
    - Example:
    ```cadence
    let add = fun (a: Int, b: Int): Int {
      return a + b
    }
    ```

3. **Parentheses**:
    - Parentheses are used to group expressions and control evaluation order. When an expression is enclosed in parentheses, it is evaluated before any other surrounding operations.
    - Example:
    ```cadence
    let result = (a + b) * c  // The sum of a and b is computed first
    ```

4. **Function Invocation**:
    - Postfix function invocation allows a function to be called with specific parameters. It is placed after an expression that represents a callable function.
    - Example:
    ```cadence
    let total = add(5, 10)  // Calls the function add with arguments 5 and 10
    ```

5. **Expression Access (Array/Struct/Dictionary Access)**:
    - Expression access is used to retrieve a value from a collection like an array, struct, or dictionary. Use square brackets [] for arrays and dictionaries, and dot notation for structs.
    - Example:
    ```cadence
    let element = myArray[2]  // Accesses the element at index 2 in the array
    ```

6. **Accessing Object Properties**:
    - Postfix expressions allow the retrieval of fields or properties from objects, such as structs or resources.
    - Example:
    ```cadence
    let accountName = account.name  // Accesses the "name" field of the account object
    ```

7. **Force Unwrapping Optionals**:
    - When an optional value is guaranteed to have a non-nil value, force unwrapping can be used to extract the value.
    - Example:
    ```cadence
    let score: Int? = 90
    let finalScore = score!  // Force unwrap the optional
    ```

- **Examples**:
  - **Accessing an Array Element**:
    ```cadence
    let numbers = [1, 2, 3, 4]
    let firstNumber = numbers[0]  // Accesses the first element
    ```
  - **Accessing a Struct Field**:
    ```cadence
    struct User {
      let name: String
      let age: Int
    }
    let user = User(name: "Alice", age: 30)
    let userName = user.name  // Accesses the "name" field
    ```
  - **Calling a Function**:
    ```cadence
    fun add(a: Int, b: Int): Int {
      return a + b
    }
    let sum = add(5, 10)  // Invokes the function with arguments
    ```
  - **Force Unwrapping an Optional**:
    ```cadence
    let optionalValue: Int? = 42
    let value = optionalValue!  // Force unwrap the optional, result is 42
    ```

- **EBNF**:

  ```ebnf
  postfixExpression
    : identifier
    | literal
    | functionExpression
    | '(' expression ')'
    | postfixExpression invocation
    | postfixExpression expressionAccess
    | postfixExpression '!'
    ;
  ```


- **Important Considerations**:
  - **Optional Unwrapping**: Use force unwrapping (!) cautiously, as it can lead to runtime errors if the optional contains nil. It’s safer to first check if the optional has a value before unwrapping.
  - **Array and Dictionary Access**: When accessing elements in collections, ensure that the index or key exists to prevent out-of-bounds errors or missing key errors.

Postfix expressions are essential for interacting with values, invoking functions, and managing collections in Cadence. They provide a powerful and flexible way to operate on data in smart contracts and decentralized applications.

#### Destroy Expressions

Destroy expressions in Cadence are used to explicitly dispose of resources. In Cadence’s resource-oriented programming model, resources must be properly managed, either by being moved to a new owner or explicitly destroyed when no longer needed. The destroy keyword ensures that resources are safely removed from the blockchain’s state without causing memory leaks or leaving unused resources behind.

Destroying a resource signals that the resource has fulfilled its purpose and is no longer required. It helps maintain the integrity of Cadence’s resource safety by ensuring that all resources are accounted for and disposed of properly.

- **Syntax**:
  - A destroy expression begins with the destroy keyword, followed by the resource to be destroyed.
  - Example:
    ```cadence
    destroy token
    ```

- **Semantics**:
  - **Destroying Resources**: The primary use of the destroy expression is to deallocate or dispose of a resource. When a resource is destroyed, it is removed from the program’s state, and no further operations can be performed on it. This prevents the resource from lingering in memory or causing errors if accessed again.
  - **Ownership and Destruction**: A resource must be moved to the destroy expression to be destroyed. Resources that are no longer owned or needed should be explicitly destroyed to avoid resource mismanagement.

- **Usage**:
  - Destroying Tokens, NFTs, or Custom Resources: Any custom-defined resource (e.g., tokens, NFTs, or other objects) can be destroyed when it is no longer needed.
  - Example:
    ```cadence
    let token <- create Token()
    destroy token   // The token is destroyed and can no longer be used
    ```

- **Examples**:
  - **Destroying a Resource**:
    ```cadence
    let nft <- create NFT(id: 1)
    destroy nft   // The NFT resource is destroyed and cannot be used again
    ```
  - **Destroying a Custom Resource**:
    ```cadence
    resource Vault {
      pub let balance: UFix64
      
      init(balance: UFix64) {
        self.balance = balance
      }
    }
    
    let vault <- create Vault(balance: 1000.0)
    destroy vault   // The vault resource is destroyed
    ```

- **EBNF**:
  ```ebnf
  destroyExpression
    : 'destroy' expression
    ;
  ```

- **Important Considerations**:
  - **Mandatory Resource Management**: Cadence enforces strict resource management rules, meaning that all resources must either be moved to new owners or destroyed. Failing to properly destroy or move resources results in compile-time or runtime errors.
  - **No Implicit Destruction**: Resources are not automatically destroyed when they go out of scope. The destroy keyword must be used explicitly to indicate that the resource is no longer needed.

Destroy expressions play a critical role in managing resources in Cadence by ensuring that resources are explicitly disposed of when they are no longer needed. This prevents issues like resource leakage, ensuring that resources are properly accounted for in the lifecycle of a smart contract or transaction.

#### Reference Expressions

Reference expressions in Cadence are used to create references to values or resources without transferring ownership. They allow developers to borrow access to a resource or value temporarily, either for reading or writing, depending on the type of reference. References are essential in Cadence’s resource-oriented programming model because they enable safe interactions with resources without violating the strict ownership and movement rules of the language.

A reference points to a value or resource stored elsewhere, providing access to it without taking ownership. References can be used to avoid moving resources, allowing read or write access to resources while preserving their ownership in another part of the contract.

- **Syntax**:
  - A reference expression starts with the & symbol, followed by an expression (the value or resource to be referenced), and then the as keyword, which specifies the full type of the reference.
  - Example:
    ```cadence
    let tokenRef = &account.storageReference as &Token
    ```

- **Semantics**:
  - **Read-only vs. Mutable References**: Cadence distinguishes between read-only references (&T) and mutable references (&T{}), depending on whether the reference allows modification of the underlying value. A read-only reference gives read access to the value, while a mutable reference allows both reading and writing.
  - **Type Safety**: The as keyword ensures that the reference is properly typed, enforcing Cadence’s strict type-checking rules. The fullType annotation specifies the exact type of the reference, making it clear what kind of value or resource is being referenced and how it can be accessed.
  - **Borrowing Without Ownership Transfer**: Reference expressions allow developers to access a value or resource temporarily without transferring ownership. This is especially important when resources need to be read or modified without violating Cadence’s strict rules on resource movement.

- **Usage**:
  - **Borrowing Resources**: References are commonly used to access resources stored in an account or contract without transferring ownership. For example, a contract might borrow a reference to a token to check its balance or perform other operations.
  - **Read-only Access**: When a reference is declared as read-only, the borrowed resource or value can be accessed but not modified.
  - **Mutable Access**: When a reference is mutable, it allows both reading and modifying the value or resource being referenced.
  - Example:
    ```cadence
    let vaultRef = &account.vault as &Vault   // Borrowing a reference to a Vault resource
    ```

- **Examples**:
  - **Read-only Reference**:
    ```cadence
    let balanceRef = &vault.balance as &UFix64
    ```
    - This creates a read-only reference to the balance field of a Vault resource, allowing the balance to be accessed but not modified.
  - **Mutable Reference**:
    ```cadence
    let mutableTokenRef = &token as &Token
    mutableTokenRef.transfer(to: recipient)   // Modifying the referenced token resource
    ```
    - This creates a mutable reference to a Token resource, allowing the referenced token to be transferred or modified.
  - **Borrowing a Reference in a Function**:
    ```cadence
    pub fun borrowVaultRef(): &Vault {
      return &self.vault as &Vault
    }
    ```
    - This function returns a reference to the Vault resource, allowing the caller to interact with the resource without transferring ownership.

- **EBNF**:
  ```ebnf
  referenceExpression
    : '&' expression 'as' fullType
    ;

  ```

- **Important Considerations**:
  - **Ownership and Safety**: References provide a way to safely interact with resources without transferring ownership. However, developers must ensure that references are used properly and that the underlying resource remains valid for the duration of the reference’s lifetime.
  - **Read vs. Write Access**: It’s important to distinguish between read-only and mutable references. A read-only reference (&T) allows only read access, while a mutable reference (&T{}) allows modification of the underlying value.
  - **Lifetime of References**: A reference is valid only as long as the value or resource it refers to exists. If the underlying value is moved or destroyed, the reference becomes invalid.

Reference expressions in Cadence are a powerful mechanism for accessing values and resources without transferring ownership. They allow developers to borrow resources safely, respecting the language’s strict resource management rules, and ensure type safety by requiring clear type annotations. By providing flexible access to data and resources, reference expressions enable developers to build efficient and secure smart contracts while maintaining control over resource ownership.

#### Function Calls

Function calls in Cadence allow previously defined functions to be invoked, passing arguments corresponding to the function’s parameters. A function call involves specifying the function’s name and providing the required arguments, ensuring that the number of arguments matches the number of parameters defined in the function. Function calls are essential in enabling code reuse, modularity, and readability within Cadence smart contracts and scripts.

- **Function Argument Labels and Parameter Names**:

  - Each function parameter has an argument label and a parameter name:
    - The argument label is used when calling the function, and each argument is passed with its argument label preceding it.
    - The parameter name is used within the function’s implementation to refer to the value of the argument passed.

  - By default, parameters use their parameter name as the argument label. However, you can specify different argument labels to make function calls more readable and expressive. This is useful for improving the clarity of function calls, allowing them to appear more like natural language expressions, while the function body remains concise.

  - Example:
    ```cadence
    fun greet(firstName: String, lastName: String) {
      log("Hello, \(firstName) \(lastName)!")
    }
    
    greet(firstName: "Alice", lastName: "Smith")  // Calling the function using argument labels
    ```

- **Specifying Argument Labels**:

  - To specify an argument label, place it before the parameter name in the function declaration, separated by a space:
    - Example:
    ```cadence
    fun greet(hello firstName: String, family lastName: String) {
      log("Hello, \(firstName) \(lastName)!")
    }
    
    greet(hello: "Alice", family: "Smith")  // Using specified argument labels
    ```
  - This makes function calls more expressive and readable by clearly identifying what each argument represents.

- **Omitting Argument Labels**:

  - If you do not want a parameter to have an argument label, you can use an underscore (_) in place of the label in the function declaration. When a label is omitted, you can call the function without specifying an argument label for that parameter.
  - Example:
    ```cadence
    fun multiply(_ first: Int, by second: Int) -> Int {
      return first * second
    }
    
    let result = multiply(3, by: 4)   // No label for the first argument, but the second uses the 'by' label
    ```
  - This allows for more flexibility when defining functions and how they are called.

- **Argument Order and Labeling Requirements**:

1. **Argument Order**: Arguments must be passed in the same order as the parameters are defined in the function. The order in which arguments are provided in the function call must match the corresponding parameters in the function signature.
    - Example:
      ```cadence
      fun calculateTotal(price: UFix64, quantity: Int) -> UFix64 {
        return price * UFix64(quantity)
      }
      
      let total = calculateTotal(price: 10.5, quantity: 3)
      ```
2. **Argument Labels**: If a parameter has an argument label, it must be included in the function call. If no label is provided (using an underscore _), the argument is passed without a label. This ensures that function calls are clear, especially in cases where multiple parameters of the same type are passed.
3. **Argument Passing**:
    - Arguments in Cadence are passed by value for most types, meaning the value is copied when passed to the function. For resources, however, arguments are passed by move, transferring ownership of the resource to the function. If a resource is passed into a function, it must be explicitly moved back or destroyed by the function to maintain Cadence’s resource safety model.
    - Example of resource passing:
      ```cadence
      fun transferToken(_ token: @Token, to recipient: Address) {
        // Transfer logic...
      }
      
      transferToken(<-myToken, to: recipientAddress)
      ```

- **Example of a Simple Function Call**:
  - Example:
  ```cadence
  fun add(a: Int, b: Int) -> Int {
    return a + b
  }
  
  let result = add(a: 5, b: 10)   // Calling the function with labeled arguments
  ```

- **Key Considerations**:

1. Argument Labels are Not Part of Function Types: Cadence treats argument labels as part of the function’s syntax for readability, but they are not part of the function type. This means functions with different argument labels but the same parameter types and return type are compatible.
2. No Default Parameter Values: Cadence does not support default parameter values. Each argument must be explicitly passed when invoking a function.
3. Function Types and Labels: While argument labels are useful for making function calls more expressive, they are not preserved when assigning a function to a function type. As a result, when invoking plain function values, argument labels cannot be used, and arguments must be passed purely based on order.

- **EBNF**:

  ```cadence
  functionCall:
      postfixExpression invocation
    ;
  
  invocation
    : ( '<' ( typeAnnotation ( ',' typeAnnotation )* )? '>' )?
      '(' ( argument ( ',' argument )* )? ')'
    ;
  ```

Function calls are a key part of how Cadence enables modular and reusable code, with a focus on argument labels and safe resource handling to enhance code clarity and maintainability. By clearly distinguishing between labels and parameter names, developers can write both expressive and concise function signatures, improving the overall readability of smart contracts and applications.

#### Function Expressions

Function expressions in Cadence represent either named or anonymous functions (closures) that can be defined and passed around as values. They are first-class citizens in the language, meaning that functions can be assigned to variables, passed as arguments, and returned from other functions. Function expressions are crucial for enabling higher-order programming patterns, such as callbacks and functional-style programming.

Syntax

The syntax of function expressions follows the general pattern of the fun keyword, followed by an optional parameter list, a return type, and a block containing the function’s body.

- **Syntax**:
  ```cadence
  fun (parameterList): returnType {
    functionBody
  }
  ```

- **Components of a Function Expression**:

1. **Parameter List**:
    - A function’s parameter list is enclosed in parentheses () and can contain zero or more parameters, each defined by a name and type. If the function takes no parameters, an empty set of parentheses is used.
    - Example:
    ```cadence
    fun (): Void { ... }  // Function with no parameters
    fun (a: Int, b: Int): Int { ... }  // Function with two integer parameters
    ```
2. **Return Type**:
    - After the parameter list, a colon : followed by the return type specifies the type of value that the function returns. If no value is returned, the return type is Void.
    - Example:
    ```cadence
    fun (): Void { return }  // Function with no return value
    fun (a: Int, b: Int): Int { return a + b }  // Returns an integer
    ```
3. **Function Body**:
    - The body of the function is enclosed in curly braces {} and contains the statements that define the behavior of the function.
    - Example:
    ```cadence
    fun (a: Int, b: Int): Int {
      return a + b  // Adds two integers and returns the result
    }
    ```

- **Anonymous Functions (Closures)**:

  - Cadence allows the creation of anonymous functions, or closures, which are defined without a name and can be passed as arguments to other functions or assigned to variables. Closures capture variables from their surrounding scope, providing a powerful way to create flexible and reusable code.

  - Example of an Anonymous Function:
  ```cadence
  let add = fun (a: Int, b: Int): Int {
    return a + b
  }
  
  let sum = add(5, 10)  // Calls the anonymous function
  ```

- **Function Expressions in Postfix Contexts**:

  - In Cadence, function expressions can be used in postfix expressions to invoke or pass them as arguments. This enables the seamless integration of functions into various expressions and allows for high flexibility when writing smart contracts and dApps.

  - Example of Function Invocation:
  ```cadence
  fun multiply(a: Int, b: Int): Int {
    return a * b
  }
  
  let result = multiply(3, 4)  // Calls the function with arguments 3 and 4
  ```
  - Example of Passing a Function as a Parameter:
  ```cadence
  fun applyOperation(a: Int, b: Int, operation: (Int, Int): Int): Int {
    return operation(a, b)
  }
  
  let sum = applyOperation(5, 10, fun (a: Int, b: Int): Int {
    return a + b
  })  // Passes an anonymous function to applyOperation
  ```

- **Returning Functions from Other Functions**:

  - Cadence also supports returning functions from other functions, enabling powerful higher-order programming capabilities. This feature can be used to generate customized functions at runtime or create function factories.

  - Example of Returning a Function:
  ```cadence
  fun createMultiplier(factor: Int): (Int): Int {
    return fun (x: Int): Int {
        return x * factor
    }
  }
  
  let double = createMultiplier(2)
  let result = double(10)  // Returns 20
  ```

- **Closures and Variable Capture**:

  - Anonymous functions in Cadence, also known as closures, can capture variables from their surrounding environment. This allows them to “remember” the values that were in scope when the function was created.

  - Example of Variable Capture:
  ```cadence
  let factor = 3

  let multiplyByFactor = fun (x: Int): Int {
    return x * factor  // Captures 'factor' from the surrounding scope
  }
  
  let result = multiplyByFactor(10)  // Returns 30
  ```

- **EBNF**:

  ```cadence
  functionExpression:
      'fun' parameterList ( ':' typeAnnotation )? functionBlock
    ;
  ```

- **Important Considerations**:

  - **Recursion**: Functions in Cadence can be recursive, meaning they can call themselves. Be cautious with recursion, as it may lead to stack overflow errors if not carefully controlled.
  - **Closures and Lifetime**: When using closures, ensure that any captured variables are still valid when the closure is executed, especially in the context of resources and blockchain-specific data.

- **Examples**:

  - **Basic Function Declaration and Invocation**:
  ```cadence
  fun greet(name: String): String {
    return "Hello, " + name + "!"
  }
  
  let greeting = greet("Alice")  // Result is "Hello, Alice!"
  ```
  - Anonymous Function Passed as a Parameter:
  ```cadence
  let sum = applyOperation(10, 20, fun (a: Int, b: Int): Int {
    return a + b
  })
  ```

This section now provides a comprehensive overview of function expressions, covering their syntax, use cases, and practical considerations in Cadence. Let me know if you need further refinements or additional details!

#### Type Casting and Type Checking

Cadence supports explicit type casting and runtime type checks to ensure safe interactions between types.

- **Example**:
  ```cadence
  let castedValue = value as! Type
  if value is Type { /* ... */ }
  ```

#### Resource Expressions

Resource expressions manage resource creation, movement, and destruction. Cadence ensures that resources are safely handled to avoid duplication or loss.

- **Example**:
  - **Move**:
    ```cadence
    let resource = <- someExpression
    ```
  - **Create**:
    ```cadence
    let resource = create SomeResource()
    ```
  - **Destroy**:
    ```cadence
    destroy resource
    ```

---

This section defines the core syntax and grammar rules in Cadence, ensuring that the language is structured, readable, and secure for developers building decentralized applications on the Flow blockchain.

## IV. Data Types

Cadence provides a rich set of data types that developers can use to define the structure and behavior of their smart contracts and applications. This section outlines the core data types in Cadence, including primitive types, complex types, resources, and advanced constructs like optional types, references, and capabilities.

**TODO**
Add discussion on function types.

---

### 1. Primitive Data Types

Primitive data types in Cadence are the basic building blocks of the language. These types are used to represent simple values like numbers, booleans, and strings.

- **Integer Types**: Cadence supports various integer types with explicit sizes.
  - `Int`, `Int8`, `Int16`, `Int32`, `Int64`, `Int128`, `Int256`: Signed integers of varying bit widths.
  - `UInt`, `UInt8`, `UInt16`, `UInt32`, `UInt64`, `UInt128`, `UInt256`: Unsigned integers of varying bit widths that check for overflow and underflow.
  - `Word`, `Word8`, `Word16`, `Word32`, `Word64`, `Word128`, `Word256`: Unsigned integers of varying bit widths that do **not** check for overflow and underflow.

- **Address Types**: 
  - `Address`: Unsigned 64 bit integers that represent addresses.

- **Fixed-Point Types**: 
  - `Fix64`: Fixed-point number with 64 bits, used for precise decimal calculations.
  - `UFix64`: Unsigned fixed-point number with 64 bits.

- **Booleans**: The `Bool` type represents truth values.
  - Possible values: `true`, `false`.

- **Characters**: The `Character` type represents a single Unicode character.

- **Strings**: The `String` type represents a sequence of Unicode characters.
  - String literals are enclosed in double quotes: `"Hello, Cadence!"`.
  - Strings in Cadence are immutable, meaning once created, their content cannot be changed.

- **Arrays**: Arrays in Cadence store ordered collections of values of the same type.
  - Arrays are mutable, meaning their elements can be modified after creation.
  - **Example**: `[ElementType]`, e.g., `[Int]`, `Array<String>`.
  - Example: `let numbers: [Int] = [1, 2, 3, 4]`.
  - **TODO** Expand Array information.

- **Dictionaries**: Dictionaries store key-value pairs where both keys and values are of specific types.
  - Keys must be of a type that conforms to the `Equatable` interface (i.e., they can be compared for equality).
  - **Example**: `{KeyType: ValueType}`, e.g., `{String: Int}`, `Dictionary<Address, Resource>`.
  - Example: `let userBalances: {String: Int} = {"Alice": 100, "Bob": 200}`.

---

### 2. Complex Types

Cadence supports complex types that allow grouping and representing collections of related values.

- **Tuples**: Tuples are fixed-size collections of values, where each element can have a different type.
  - Tuples provide a lightweight way to group multiple values.
  - **Example**: `(Type1, Type2, ...)`, e.g., `(String, Int, Bool)`.
  - Example: `let person: (String, Int) = ("Alice", 30)`.

- **Enumerations**: Enumerations define a set of named values that represent all possible cases for a given type.
  - **Example**:
    ```cadence
    pub enum EnumName {
        case case1
        case case2
    }
    ```
  - Example:
    ```cadence
    pub enum Color {
        case red
        case green
        case blue
    }
    ```

---

### 3. Resources

Resources are unique and central to Cadence's **resource-oriented programming** model. Resources are used to represent digital assets, such as tokens or NFTs, that must be safely managed within a program.

- **EBNF**:
  ```ebnf
  resource-declaration:
    access "resource" struct-name conformances? "{" memberOrDeclaration "}" ;
  
  conformances:
    ":" nominal-type ( "," nominal-type )* ;
  ```

**TODO**
Discuss `AnyResource`. Discuss `self`.

- **Properties and Behavior**:
  - Resources have strict rules: they can be **created**, **moved**, and **destroyed**, but they cannot be **copied** or **implicitly discarded**.
  - The **resource keyword** is used to declare a resource type, ensuring its ownership is tracked.

  - **Example**:
    ```cadence
    pub resource ResourceName {
        // Fields and methods
    }
    ```

- **Resource Ownership**:
  - Resources must be explicitly created using the `create` keyword, moved using the `<-` operator, and destroyed when no longer needed using the `destroy` keyword.
  - Example:
    ```cadence
    let myResource <- create ResourceName()
    destroy myResource
    ```

- **Linear Types**:
  - Cadence enforces **linear type semantics** for resources, ensuring that a resource can only exist in one place at a time, preventing accidental duplication or loss.

---

### 4. Structs

Structs in Cadence are user-defined types used to group values together. Unlike resources, structs are **non-linear** types and can be copied and freely passed around.

**TODO**
Discuss `AnyStruct`

- **Definition of Structs**:
  - Structs allow the creation of composite types that store related values.
  - **Example**:
    ```cadence
    pub struct StructName {
        pub let field1: Type
        pub let field2: Type
    }
    ```

  - Example:
    ```cadence
    pub struct Point {
        pub let x: Int
        pub let y: Int

        init(x: Int, y: Int) {
            self.x = x
            self.y = y
        }
    }
    ```

- **EBNF**:
  ```ebnf
  struct-declaration:
    access "struct" struct-name conformances? "{" memberOrDeclaration "}" ;
  
  conformances:
    ":" nominal-type ( "," nominal-type )* ;
  ```

- **Storage of Values and References**:
  - Structs store values directly and can also store references or resources.

---

### 5. Optional Types

Optional types in Cadence are used to represent values that may or may not be present. They help handle cases where a value might be missing, preventing runtime errors due to uninitialized or null values.

- **Example**:
  - The type `T?` represents an optional value of type `T`. It can either hold a value of type `T` or be `nil`.
  - Example: `let optionalValue: Int? = nil`.

- **Optional Binding**:
  - Optional values can be safely accessed using optional binding with `if let` or the force unwrap operator `!`.

  - **Example**:
    ```cadence
    if let unwrappedValue = optionalValue {
        // Use unwrappedValue
    }
    ```

  - Force unwrapping:
    ```cadence
    let definiteValue = optionalValue!
    ```

- **Nil**:
  - The literal `nil` represents the absence of a value in an optional.


**TODO** Double optional `??`

---

### 6. References and Capabilities

References in Cadence allow indirect access to values, enabling shared or mutable access to stored data without duplicating it.

- **References** (`&T`):
  - A reference is a pointer to a value. References allow safe access to mutable or immutable data in storage or in memory without copying the value.

  - **Example**:
    ```cadence
    let reference: &ResourceType = &myResource
    ```

  - References can be either **borrowed** (non-owning) or **mutated** (mutable).
    - `&T`: A reference to a resource or value.
    - `&mut T`: A mutable reference to a resource or value.

- **Capabilities**:
  - Capabilities provide secure access control by granting specific permissions to interact with resources or data.
  - Capabilities allow users to interact with resources stored in other accounts without directly accessing them.
  - **Example**:
    ```cadence
    let capability: Capability<&ResourceType> = account.getCapability(/public/someResource)
    ```

---

### 7. Type Inference and Type Annotations

Cadence provides support for both **type inference** and **explicit type annotations**, allowing developers to choose between verbosity and flexibility.

- **Type Inference**:
  - Cadence can automatically infer the type of a variable from the context or the assigned value, reducing the need for explicit type declarations.
  - Example:
    ```cadence
    let inferredValue = 42  // Inferred as Int
    ```

- **Type Annotations**:
  - Developers can explicitly annotate types when needed for clarity or to enforce specific type constraints.
  - **Example**:
    ```cadence
    let annotatedValue: Int = 42
    ```

- **Rules for Explicit Annotations**:
  - Explicit annotations are required in certain cases, such as when defining function parameters, return types, and variables that store resources.

---

### 8. Array

**TODO** Discuss `Array` type.

---

### 9. Dictionary

**TODO** Discuss `Dictionary` type.

**TODO** Hashable keys, are dictionaries deterministic? If not, how do multiple nodes get same result?

---

### 10. Never

**TODO** Discuss `Never` type.

---

### 11. InclusiveRange

**TODO** Discuss `Never` type.

---

### 12. Closures

**TODO** Discuss closures type and the example in the Language Reference. Determine the best location in this specification for this section. Explain how functions can capture references to variables in outer scope, be passed around as values, and still assign values to the captured variable while ensuring that the variable has not gone out of scope.

---

This section provides a detailed overview of the data types available in Cadence, highlighting their usage, properties, and the unique resource-oriented features of the language. These types form the foundation for writing secure, efficient, and scalable smart contracts on the Flow blockchain.

## V. Type System and Semantics

The type system in Cadence is designed to ensure safety and security, particularly when managing resources and interactions between contracts. This section explains the various type-checking mechanisms, typing strategies, support for generics, and polymorphism. Cadence’s type system provides strong guarantees at both compile-time and runtime, helping developers avoid errors and vulnerabilities in their smart contracts.

---

### 1. Type Checking

Cadence employs a combination of **static** and **dynamic** type checking to ensure the safety and correctness of programs.

- **Static Type Checking**:
  - Cadence performs type checking at compile-time, ensuring that variables, expressions, and function calls adhere to their declared types.
  - Static type checking prevents a large class of errors before the program is executed, particularly in resource management, ownership rules, and function contracts.

  - Example:
    ```cadence
    let x: Int = 10  // Correct: type matches
    let y: String = 10  // Error: type mismatch
    ```

- **Dynamic Type Checking**:
  - In certain cases, particularly when using dynamic interfaces or interacting with external data, Cadence uses dynamic type checking at runtime to ensure type safety.
  - Dynamic checks occur when the type cannot be fully determined at compile-time, such as when casting between types or when using optional values.

  - Example:
    ```cadence
    let y: AnyStruct = 10  // Dynamic type check required for conversion
    if let value = y as? Int {
        // Safe to use 'value' as Int
    }
    ```

- **Type Compatibility Rules**:
  - **Subtyping**: Cadence allows subtypes to be used in place of their supertypes. Subtyping supports type hierarchies where a derived type (e.g., a resource or struct) can be used where its base type is expected.
  - **Type Constraints**: Type compatibility is enforced through explicit type constraints when dealing with generics and polymorphism.

---

### 2. Nominal and Structural Typing

Cadence supports both **nominal** and **structural** typing, allowing for flexibility in how types are defined and used in programs.

- **Nominal Typing**:
  - In **nominal typing**, types are distinct based on their names and declarations. Two types are considered compatible if they are explicitly declared as the same type or if one type is a subtype of another through inheritance.
  - Example:
    ```cadence
    pub resource Asset {}
    pub resource Token: Asset {}  // Token is a subtype of Asset
    ```

  - Here, `Token` is a nominal subtype of `Asset`, and values of type `Token` can be used where an `Asset` is expected.

- **Structural Typing**:
  - In **structural typing**, types are considered compatible based on their structure, i.e., the presence of fields and methods with compatible types, rather than their names.
  - This is particularly useful for interfaces, where different types may implement the same behavior regardless of their nominal names.

  - Example:
    ```cadence
    pub interface ITransfer {
        pub fun transfer(to: Address): Void
    }

    pub resource Coin: ITransfer {
        pub fun transfer(to: Address) {
            // Implementation
        }
    }
    ```

  - In this case, any type that implements the `ITransfer` interface can be structurally compatible regardless of its nominal type.

---

### 3. Type Inference

**TODO**

---

### 4. Generics

Generics in Cadence allow types to be parameterized, providing flexibility and reusability in data structures and functions. This enables developers to write code that works with multiple types while still maintaining type safety.

- **Example**:
  - Generics are declared using angle brackets (`<`), with a type parameter that can be substituted with a specific type when the function or struct is instantiated.
  
  - **Generic Function Example**:
    ```cadence
    pub fun add<T: Numeric>(a: T, b: T): T {
        return a + b
    }
    ```

  - In this example, `T` is a generic type parameter constrained to the `Numeric` type, ensuring that the function `add` can only be used with types that conform to the `Numeric` interface.

- **Bound and Unbound Generic Types**:
  - **Bound Generics**: Bound generics specify constraints on the types that can be passed as parameters. Constraints ensure that the generic type conforms to a specific protocol or type, providing guarantees about its behavior.
  
    - Example: `T: Equatable` constrains `T` to types that can be compared for equality.
    
    ```cadence
    pub fun isEqual<T: Equatable>(x: T, y: T): Bool {
        return x == y
    }
    ```

  - **Unbound Generics**: Unbound generics allow any type to be passed as a parameter without any specific constraints. This provides maximum flexibility but may limit the operations that can be performed on the generic type.

    - Example:
    ```cadence
    pub fun printValue<T>(value: T) {
        // No specific operations on T
    }
    ```

---

### 5. Polymorphism

Polymorphism in Cadence allows types and functions to operate on values of different types, providing flexibility in how contracts and data structures interact. Cadence supports both **ad-hoc polymorphism** via interfaces and **generic polymorphism**.

- **Ad-hoc Polymorphism via Interfaces**:
  - Ad-hoc polymorphism allows different types to implement the same interface, providing their own implementations for the methods defined in the interface. This form of polymorphism enables multiple types to be used interchangeably if they conform to the same interface.
  
  - Example:
    ```cadence
    pub interface Payable {
        pub fun pay(amount: UFix64)
    }

    pub resource Coin: Payable {
        pub fun pay(amount: UFix64) {
            // Implementation for Coin
        }
    }

    pub resource Token: Payable {
        pub fun pay(amount: UFix64) {
            // Implementation for Token
        }
    }
    ```

  - In this example, both `Coin` and `Token` implement the `Payable` interface, allowing any resource that conforms to `Payable` to be used in a polymorphic way.

- **Generic Polymorphism**:
  - Generic polymorphism occurs when a function or data structure can operate on values of any type, provided those types meet certain conditions (such as type constraints or interface conformance). Generics provide compile-time type safety while allowing for flexible, reusable code.
  
  - Example:
    ```cadence
    pub fun transfer<T: Payable>(item: T, amount: UFix64) {
        item.pay(amount)
    }
    ```

  - In this case, the `transfer` function can work with any type `T` that implements the `Payable` interface, providing a polymorphic way to transfer assets without requiring knowledge of the specific type of asset.

---

This section defines the core principles of Cadence’s type system, which ensures that smart contracts are secure, reusable, and maintainable. The combination of static and dynamic type checking, support for both nominal and structural typing, and powerful generics and polymorphism mechanisms provide developers with the tools they need to build robust decentralized applications on the Flow blockchain.

## VI. Resource-Oriented Programming

Cadence introduces a unique **resource-oriented programming model** specifically designed to manage digital assets securely and efficiently. Resources are first-class citizens in Cadence, and the language enforces strict rules about their creation, movement, and destruction. This model ensures that resources cannot be duplicated, inadvertently lost, or mismanaged, making it ideal for handling scarce assets such as non-fungible tokens (NFTs) and fungible tokens.

---

### 1. Resource Management

In Cadence, **resources** represent valuable, unique, and scarce assets. They are subject to a strict lifecycle management system to ensure safe and predictable behavior in decentralized applications.

- **Lifecycle of Resources**:
  - **Creation**: Resources are explicitly created using the `create` keyword. Once created, they must be assigned or transferred to ensure they are not lost.
    - Example: `let newToken <- create Token()`
  - **Movement**: Resources cannot be copied or passed by reference. Instead, they are **moved** from one location to another using the `<-` operator. Moving a resource transfers ownership.
    - Example: `let transferredToken <- originalOwner.moveToken()`
  - **Destruction**: When a resource is no longer needed, it must be explicitly destroyed using the `destroy` keyword to free the underlying state.
    - Example: `destroy oldToken`

- **Ownership and Borrowing**:
  - Cadence's resource management model is influenced by Rust's ownership and borrowing system. Each resource must have a single owner at any given time, and this ownership is transferred upon moving the resource.
  - Resources can be **borrowed** temporarily, allowing safe access without transferring ownership. Borrowing can be **immutable** (read-only) or **mutable** (read-write).
    - Example (borrowing):
      ```cadence
      let ref token: &Token = &myToken
      let mutRef token: &mut Token = &mut myToken
      ```

- **Rules for Handling Resources in Functions and Contracts**:
  - When passing resources to functions, they must be moved explicitly. If a function needs to retain ownership of a resource, it must be declared in the function signature and transferred back when the function completes.
    - Example:
      ```cadence
      pub fun transferToken(owner: &Owner, receiver: &Receiver, token: @Token) {
          receiver.receive(<- token)
      }
      ```
  - Resources cannot be implicitly returned or discarded, ensuring that they are always properly handled or destroyed.

---

### 2. Ownership Rules

Cadence enforces strict **ownership rules** to ensure that resources are always properly managed and tracked throughout their lifecycle.

- **Single Ownership**:
  - A resource can only have one owner at a time. When a resource is moved, the previous owner loses access to it, and ownership is transferred to the new owner. This guarantees that no resource can be duplicated or shared across multiple owners.
  - Example:
    ```cadence
    let newOwner <- originalOwner.moveToken()  // Ownership transferred
    ```

- **Preventing Resource Duplication**:
  - Resources cannot be **copied**. Attempting to copy a resource results in a compile-time error. This prevents duplication of valuable assets, ensuring that the total supply of resources remains controlled.
    - Invalid example:
      ```cadence
      let copyOfToken = myToken  // Error: Resources cannot be copied
      ```

- **Preventing Accidental Deletion**:
  - Resources cannot be implicitly destroyed. If a resource goes out of scope without being explicitly moved or destroyed, Cadence raises a compile-time error to ensure that resources are not accidentally lost.
    - Example (invalid):
      ```cadence
      let orphanedToken <- create Token()  // Error: Token must be used or destroyed
      ```

---

### 3. Resource Constraints

Cadence provides several **resource constraints** that ensure safe and predictable management of assets. These constraints are designed to guarantee that resources are used according to well-defined rules.

- **Restrictions on Resource Usage**:
  - Resources can only be moved, borrowed, or destroyed; they cannot be copied or implicitly discarded.
  - Once a resource is moved, the original owner can no longer access it unless it is returned to them explicitly.
  - Functions that accept resources must declare them in their signatures and explicitly move, return, or destroy them.

- **Guarantees for Resource Integrity**:
  - Resources are **guaranteed** to remain unique and identifiable throughout their lifecycle. This ensures that smart contracts that manage resources (such as tokens or NFTs) can confidently manage ownership without risk of duplication.
  - The type system ensures that resources are properly managed at both compile-time and runtime, preventing common programming errors like memory leaks or null references.

---

### 4. Capability Semantics

In Cadence, **capabilities** provide a secure mechanism to control access to resources without transferring ownership. Capabilities allow fine-grained control over who can access a resource and what actions they can perform on it.

- **Defining Capabilities**:
  - Capabilities are references that grant specific rights to interact with a resource. They can be thought of as secure, controlled access points to a resource stored in an account’s storage.
  - Example:
    ```cadence
    let capability: Capability<&Token> = account.getCapability(/public/token)
    ```

- **Creating Capabilities**:
  - Capabilities can be created and granted to other users, contracts, or entities. They specify what actions the holder of the capability can perform, such as reading, writing, or transferring the underlying resource.
  - Example (creating a capability):
    ```cadence
    pub fun createTokenCapability(): Capability<&Token> {
        return account.link<&Token>(/public/token, target: /storage/myToken)
    }
    ```

- **Granting Capabilities**:
  - Capabilities are granted to other users by linking them to resources in storage. This allows other parties to interact with the resource without having direct access to it.
  - Example:
    ```cadence
    pub fun grantAccess(account: AuthAccount, capability: Capability<&Token>) {
        account.link<&Token>(/public/token, target: /storage/myToken)
    }
    ```

- **Controlling Access via Capabilities**:
  - Capabilities can be **public**, **private**, or **restricted** based on where they are linked in an account’s storage. This provides a powerful mechanism for fine-grained access control in contracts, ensuring that only authorized entities can perform certain actions on a resource.

    - **Public Capabilities**: Can be accessed by any contract or account.
    - **Private Capabilities**: Only accessible by the account owner or those explicitly granted access.
    - **Restricted Capabilities**: Allow limited actions, such as read-only access or transfer rights, based on the capability’s definition.

  - Example (controlling access):
    ```cadence
    let readOnlyCapability: Capability<&Token> = account.getCapability(/public/token)  // Read-only access
    let restrictedCapability: Capability<&{Transferable}> = account.getCapability(/public/transferToken)  // Restricted access to transfer functionality
    ```

Capabilities ensure that resources remain secure and accessible only to authorized users, providing a scalable way to control access to assets in decentralized applications.

---

This section defines the core principles of **resource-oriented programming** in Cadence, outlining how resources are managed, moved, and controlled. By enforcing strict ownership rules and leveraging capabilities for access control, Cadence ensures that digital assets remain secure, unique, and manageable within decentralized applications on the Flow blockchain.

## VII. Contracts, Interfaces, and Transactions

Cadence is designed to support secure and flexible interactions on the Flow blockchain through **contracts**, **interfaces**, and **transactions**. Contracts encapsulate logic and state, interfaces define expected behavior, and transactions represent interactions between users and contracts. Additionally, Cadence includes an event system to handle key state changes and notify interested parties.

---

### 1. Contracts

**Contracts** in Cadence are the central units of logic that encapsulate state and define methods for interacting with that state. They typically represent a set of business rules or asset management processes, such as managing tokens, governing access control, or executing financial transactions.

- **Syntax for Contract Definition**:
  A contract in Cadence is defined using the `contract` keyword. Contracts can contain fields, functions, resources, and events. Contracts are deployed to the Flow blockchain and remain there for their lifecycle.

  - **Example**:
    ```cadence
    pub contract ContractName {
        // Fields
        pub var totalSupply: UFix64

        // Functions
        pub fun deposit(amount: UFix64) {
            self.totalSupply = self.totalSupply + amount
        }

        init() {
            self.totalSupply = 0.0
        }
    }
    ```

- **Methods and Fields within Contracts**:
  - **Fields**: Fields represent contract state and can be mutable (`var`) or immutable (`let`).
    - Example: `pub var balance: UFix64`.
  - **Functions**: Functions define behavior and can manipulate the contract's internal state.
    - Example: `pub fun withdraw(amount: UFix64)`.
  - Access control for fields and methods is controlled by access modifiers (`pub`, `priv`, etc.). Public fields and functions are accessible externally, while private ones are only accessible within the contract.

- **Upgradability and Maintainability of Contracts**:
  - Cadence allows contracts to be upgraded over time. Contract **upgradability** is handled through the Flow blockchain’s account-based model. Each account can have its own contracts, which can be **updated** or **re-deployed** as necessary.
  - Developers can evolve their contracts by deploying new versions, but care must be taken to ensure **backward compatibility**. Contract storage and key resources must be handled properly to avoid disrupting dependent systems.
  - Contracts on Flow are **account-bound** and can be updated by the account that controls them, allowing maintainability without losing access to critical resources.

---

### 2. Interfaces

Cadence supports **interfaces** to define a set of behaviors that can be implemented by contracts and resources. Interfaces promote modularity and allow for dynamic and flexible code, ensuring different components can work together as long as they implement the same interface.

- **Static Interfaces** (Compile-time Checks):
  - Static interfaces define a contract's or resource's expected behavior at compile time. Any contract or resource that implements an interface must provide the necessary functions as defined by the interface.
  - Static interfaces enforce a strong type system that ensures contracts adhere to the interface at compile-time, preventing errors from missing methods or incompatible types.

  - **Example**:
    ```cadence
    pub interface Transferable {
        pub fun transfer(amount: UFix64, to: Address)
    }

    pub contract Token: Transferable {
        pub fun transfer(amount: UFix64, to: Address) {
            // Implementation of transfer
        }
    }
    ```

- **Dynamic Interfaces** (Runtime Flexibility):
  - Dynamic interfaces allow contracts and resources to be checked for conformance to an interface at runtime. This provides flexibility when interacting with multiple types that may not implement the same interface at compile time but could be safely used interchangeably if they do at runtime.
  - Example:
    ```cadence
    if let transferable = object as? &Transferable {
        transferable.transfer(10.0, to: recipient)
    }
    ```

Dynamic interfaces provide flexibility when interacting with objects whose type may not be known at compile-time but can be safely used through runtime checks.

---

### 3. Transactions

**Transactions** in Cadence represent the interactions between users and the blockchain. They are the primary mechanism through which users submit actions that alter the state of contracts, move resources, or interact with smart contract logic. Transactions are signed and authorized by accounts to ensure security and integrity.

- **Structure and Semantics of Transactions in Cadence**:
  - A transaction is defined using the `transaction` keyword and consists of several parts: **preparation**, **execution**, and optional **post-conditions**.
  - Transactions generally interact with contracts or resources, initiating actions like transferring tokens, updating contract state, or invoking smart contract methods.

  - **Example**:
    ```cadence
    transaction(amount: UFix64, recipient: Address) {
        prepare(signer: AuthAccount) {
            let token <- signer.borrow<&Token>(from: /storage/myToken)!
            token.transfer(amount, to: recipient)
        }
        execute {
            log("Transaction executed successfully")
        }
    }
    ```

  - **Preparation**: In the `prepare` block, accounts (such as the signer) can access and prepare resources for the transaction. Resources can be borrowed, moved, or checked before the transaction executes.
  - **Execution**: In the `execute` block, the actual transaction logic is performed. This is where contract interactions, resource transfers, and state updates happen.
  - **Post-conditions**: Post-conditions (optional) ensure that the expected state changes have occurred and can roll back the transaction if necessary.

- **Signature and Authorization Mechanisms**:
  - **Signature**: Transactions must be signed by the submitting user’s account to ensure authenticity and prevent unauthorized transactions.
  - **Authorization**: The `AuthAccount` type is used in the `prepare` block to represent the account authorizing the transaction. Only the account with the appropriate authorization can perform actions like moving or destroying resources owned by that account.

  - Example:
    ```cadence
    transaction {
        prepare(signer: AuthAccount) {
            let asset <- signer.borrow<&Asset>(from: /storage/asset)!
            // Use asset
        }
        execute {
            // Execution logic
        }
    }
    ```

Transactions ensure that only authorized users can modify contract state or transfer resources, enhancing security and integrity on the blockchain.

---

### 4. Events

**Events** in Cadence are used to signal significant changes in contract state. They provide a way to notify external systems or clients about important actions, such as a transfer of assets or a state update in a contract. Events are emitted by contracts during execution and can be subscribed to by off-chain components.

- **Syntax and Behavior of Emitting Events**:
  - Events are defined using the `event` keyword and are emitted inside functions or transaction bodies.
  - Events can carry data, such as the amount of tokens transferred, the sender, and the recipient in a token transfer event.

  - **Example**:
    ```cadence
    pub event Transfer(from: Address, to: Address, amount: UFix64)

    pub contract Token {
        pub fun transfer(amount: UFix64, to: Address) {
            emit Transfer(from: self.owner, to: to, amount: amount)
        }
    }
    ```

- **Handling Events**:
  - Off-chain systems can subscribe to events emitted by contracts to receive real-time updates about significant actions. Events provide an immutable audit trail of important state changes, making them vital for analytics, monitoring, and external contract interactions.

  - Example of event emission:
    ```cadence
    emit Transfer(from: sender, to: recipient, amount: 100.0)
    ```

Events are an essential part of the Flow blockchain ecosystem, enabling communication between the blockchain and external services while preserving security and transparency.

---

This section outlines the core constructs of **contracts**, **interfaces**, and **transactions** in Cadence. These features form the backbone of decentralized applications on Flow, providing developers with powerful tools to create secure, scalable, and flexible smart contracts and user interactions.

## VIII. Memory Model and Storage

The **memory model and storage system** in Cadence are designed to securely and efficiently manage data and resources in a decentralized context. This section explains how data is organized in Flow accounts, the scope of variables, and how resources and data persist across transactions.

---

### 1. Account Storage

In Cadence, data and resources are stored within Flow accounts using a well-defined structure to control access and ownership. Every account has three distinct **storage paths**—`storage`, `private`, and `public`—which dictate who can access stored data.

- **Explanation of Storage Paths**:
  - **`storage`**: This is the primary location for storing account-specific data and resources. Data in `storage` is only accessible by the account owner. It is the most secure and private storage path, where valuable assets such as NFTs, tokens, or sensitive contract state can be stored.
    - Example: `/storage/myToken`.
  
  - **`private`**: Data stored in `private` paths can only be accessed by the account itself or by authorized parties through **capabilities**. The `private` path is used for data that should not be exposed to the public but may be shared with specific contracts or users via capabilities.
    - Example: `/private/mySecretData`.

  - **`public`**: Data stored in `public` paths is accessible to anyone. Public storage is typically used for exposing data that other accounts or contracts may need to interact with, such as balances or public-facing information. Public paths do not expose the actual resource but rather provide references or capabilities to interact with the data.
    - Example: `/public/tokenBalance`.

- **Rules for Storing Resources and Data**:
  - Resources and data must be stored in well-defined paths under one of the storage categories (`storage`, `private`, or `public`). Resources are moved between accounts using explicit transactions and must be stored in the correct paths to ensure security and proper access control.
  - Only the account owner can directly manipulate their `storage` paths. Public and private paths must use **capabilities** to grant or restrict access to external entities.
  - Example of storing a resource:
    ```cadence
    signer.save(<- myResource, to: /storage/myResource)
    ```

  - **Linking Capabilities**: When sharing access to stored resources, accounts can link a resource from `storage` to `public` or `private` paths via capabilities. This allows controlled access to the resource without exposing the underlying data.
    ```cadence
    signer.link<&ResourceType>(/public/myResource, target: /storage/myResource)
    ```

---

### 2. Global and Local Variables

Cadence uses **scoping rules** to define where variables and resources can be accessed. Variables can be **local**, **global**, or **account-bound**, depending on where and how they are defined.

- **Local Variables**:
  - Local variables are declared inside functions or transactions and exist only within the scope of that function or transaction. Once the function completes, local variables go out of scope and are destroyed unless explicitly moved or stored.
  - Example:
    ```cadence
    pub fun example() {
        let localVar = 10  // Local variable
    }
    ```

- **Global Variables**:
  - Global variables are declared at the contract or script level and persist for the lifetime of the contract or script. They can store resources, data, or functions that are available across multiple invocations of the contract.
  - Example:
    ```cadence
    pub contract MyContract {
        pub var globalVar: Int
        init() {
            self.globalVar = 0
        }
    }
    ```

- **Account-bound Variables**:
  - Variables and resources stored in an account’s `storage` path are **account-bound**. They persist beyond the execution of individual transactions and are specific to the account where they reside. Account-bound variables can be accessed by contracts and transactions that have the necessary capabilities.
  - Example:
    ```cadence
    signer.save(<- myResource, to: /storage/myResource)
    ```

These scoping rules ensure that data and resources are managed securely and efficiently, with clear boundaries between temporary (local) and persistent (global and account-bound) variables.

---

### 3. Resource and Data Persistence

Resources and data in Cadence are **persistent**, meaning they continue to exist across multiple transactions and contract executions as long as they are stored in account `storage`. This persistence is key to managing long-lived assets such as tokens, NFTs, or other valuable resources.

- **How Resources Are Persisted Across Transactions**:
  - When a resource is stored in an account’s `storage` path, it persists beyond the execution of the transaction in which it was created or received. Resources are not automatically destroyed when a transaction ends but remain in the account’s storage until explicitly moved or destroyed by future transactions.
  - Example of persisting a resource across transactions:
    ```cadence
    transaction {
        prepare(signer: AuthAccount) {
            let newResource <- create MyResource()
            signer.save(<- newResource, to: /storage/myResource)
        }
    }
    ```

- **Semantics for Loading and Saving Data from Storage**:
  - **Loading**: Resources and data can be **loaded** from storage using the `borrow` or `load` methods. `borrow` allows for temporary access to the resource without moving it, while `load` moves the resource out of storage, removing it from the account.
    - Example (borrowing):
      ```cadence
      let resourceRef = signer.borrow<&MyResource>(from: /storage/myResource)
      ```
    - Example (loading and moving):
      ```cadence
      let resource <- signer.load<@MyResource>(from: /storage/myResource)
      ```

  - **Saving**: New resources can be saved to an account’s `storage` path using the `save` method. Saving a resource persists it in the account, allowing it to be accessed by future transactions.
    - Example (saving):
      ```cadence
      signer.save(<- resource, to: /storage/myResource)
      ```

  - **Destruction**: If a resource is no longer needed, it can be destroyed, releasing the memory and preventing further use. Destruction is explicit in Cadence to prevent accidental loss of valuable assets.
    - Example:
      ```cadence
      destroy resource
      ```

The persistence model in Cadence ensures that resources are stored securely across transactions, while explicit load, save, and destroy operations provide fine-grained control over resource management.

---

This section outlines the **memory model and storage system** in Cadence, focusing on how data is stored and managed within Flow accounts. Through well-defined storage paths, scoping rules, and persistence semantics, Cadence ensures that resources and data are securely and efficiently handled across transactions and contract executions.

## IX. Execution Model

The **execution model** of Cadence defines how smart contracts, transactions, and scripts are executed on the Flow blockchain. This model ensures predictable and secure interactions with resources and data, supporting robust decentralized applications. The execution model encompasses runtime behavior, error handling, and the concurrency model for managing multiple tasks.

---

### 1. Runtime Behavior

**Runtime behavior** in Cadence refers to how programs are processed and executed on Flow nodes. Each Flow node executes transactions and smart contracts using the Cadence interpreter, ensuring that resources are safely handled and that program logic is carried out according to Cadence’s rules.

- **How Cadence Programs Are Executed in Flow Nodes**:
  - Cadence programs, such as smart contracts and transactions, are executed deterministically within the Flow blockchain. Every transaction and contract runs in the context of a **Flow node**, ensuring that the same inputs always produce the same outputs across all nodes.
  - Flow’s execution architecture separates consensus and computation, with dedicated execution nodes responsible for running Cadence programs.
  - During execution, Flow nodes manage the **account storage**, access control, and state updates associated with the program. Nodes also validate the correctness of resource management, ensuring no accidental duplication or loss of resources.

- **Lifecycle of a Contract Deployment**:
  - Contracts in Cadence are deployed to an account on the Flow blockchain through a transaction.
  - Contract deployment consists of several stages:
    1. **Compilation**: The contract code is compiled into an intermediate representation that can be executed by the Flow nodes.
    2. **Verification**: Flow nodes verify the contract’s syntax, type safety, and resource management logic to prevent runtime errors.
    3. **Storage**: The contract is stored in the deploying account’s `storage` path, making it accessible for future transactions.
    4. **Execution**: Once deployed, the contract’s methods can be invoked by authorized accounts, with each invocation subject to the same execution and validation process.
  
- **Lifecycle of a Transaction Execution**:
  - Transactions are user-initiated actions that interact with contracts or resources. A transaction passes through multiple phases:
    1. **Preparation**: In this phase, the transaction prepares resources, checks account balances, and authorizes access to storage paths.
    2. **Execution**: The transaction executes the core logic, which might include moving resources, updating state, or calling contract methods.
    3. **Post-conditions**: Optionally, post-conditions are checked to ensure that the final state is valid after the transaction has completed.
    4. **Commitment**: Upon successful execution, state changes and resource movements are committed to the blockchain.
    5. **Reversion**: If the transaction fails at any stage (due to an error or failed condition), it is reverted, and no changes are committed.

---

### 2. Error Handling

Error handling in Cadence ensures that smart contracts and transactions can safely manage unexpected conditions or invalid operations. Cadence provides several mechanisms for dealing with errors, including built-in error-handling constructs and resource-safe error management.

- **Syntax for Error Handling**:
  - **`panic`**: The `panic` function is used to immediately halt execution and indicate a critical error. When a panic is triggered, the program stops, and the transaction or contract is reverted.
    - Example:
      ```cadence
      pub fun withdraw(amount: UFix64) {
          if amount > self.balance {
              panic("Insufficient funds")
          }
          self.balance = self.balance - amount
      }
      ```
  - **`assert`**: The `assert` function is used to check conditions that must be true for the program to proceed. If the condition is false, `assert` triggers a failure, and the program execution halts.
    - Example:
      ```cadence
      assert(balance > 0, message: "Balance must be positive")
      ```

- **Rules for Resource-Safe Error Handling**:
  - Cadence ensures that resources are not leaked or lost during error handling. If an error occurs during a transaction, all resources involved in the transaction are either moved safely to their intended destination or reverted to their original state.
  - When a transaction fails, resources that were in the process of being transferred are returned to the sender, and any resources that were created but not saved are destroyed.
  - **Example of resource-safe error handling**:
    ```cadence
    pub fun transfer(token: @Token, to: Address) {
        if !isValidAddress(to) {
            destroy token  // Safely handle resource if transfer fails
            panic("Invalid recipient address")
        }
        to.receive(<-token)
    }
    ```

---

### 3. Concurrency Model

Cadence's **concurrency model** governs how multiple operations can be executed simultaneously or in parallel. Flow’s architecture inherently handles concurrency through its separation of roles (execution nodes, verification nodes, consensus nodes, etc.), but Cadence itself does not provide built-in concurrency primitives at the language level (such as threads or parallel processing).

- **Explanation of Concurrency Mechanisms**:
  - **Sequential Execution**: Cadence programs, including transactions and contract invocations, are executed **sequentially** within a given transaction. This ensures that all changes to state or resources are processed in a predictable order, preventing race conditions.
  - **Flow’s Concurrent Execution Model**: While Cadence programs themselves are executed sequentially, the Flow blockchain enables high levels of concurrency at the node level by running multiple transactions in parallel across different accounts. However, within a single transaction, operations are executed in a strictly ordered manner to maintain consistency and avoid data races.
  
- **No Explicit Concurrency in Cadence**:
  - Cadence does not provide explicit concurrency constructs like threads or asynchronous programming. This design decision simplifies reasoning about resource management and eliminates the possibility of concurrent access issues such as race conditions or deadlocks.
  - Instead, Flow’s architecture leverages **parallel transaction execution** across different accounts, allowing Cadence programs to run in parallel as long as they do not conflict with each other (i.e., they do not access the same account storage).

---

The **execution model** in Cadence ensures secure, predictable, and resource-safe execution of smart contracts and transactions. Through strict error handling, sequential execution within transactions, and Flow’s concurrent transaction processing across accounts, Cadence provides a solid foundation for scalable and reliable decentralized applications on the Flow blockchain.

## X. Standard Libraries and Built-in Functions

Cadence provides a set of **standard libraries** and **built-in functions** to support the development of decentralized applications on the Flow blockchain. These libraries and functions offer developers essential tools for managing resources, interacting with data structures, performing cryptographic operations, and handling basic utilities like string and array manipulation.

**TODO**
Where do integer, fixed-point, and address functions like `toString()`, `toBigEndianBytes()`, and `fromString()` belong? Need details on saturation arithmetic and library functions. Discuss string and character fields and functions. Discuss array and dictionary fields and functions.

---

### 1. Core Libraries

Cadence includes several core libraries designed to streamline the development of smart contracts and transactions by providing common utilities and functionality. These libraries are optimized for the Flow blockchain and offer a reliable foundation for handling common tasks such as cryptography, data management, and account interactions.

- **Overview of the Standard Libraries Provided by Cadence**:
  - **Flow Library (`flow`)**: The Flow library provides functionality for interacting with the Flow blockchain, including account management, token transfers, and contract deployment. It offers developers access to core Flow primitives and standard types used in the ecosystem.
    - Example of usage:
      ```cadence
      import FlowToken from 0x01
      ```

  - **Crypto Library (`crypto`)**: The crypto library provides a set of cryptographic functions to ensure the security and integrity of data. This library includes hashing functions, key generation, and signature verification to support secure communication and transactions.
    - Example of a hashing function:
      ```cadence
      let hash = crypto.sha3_256(data)
      ```

  - **Math Library (`math`)**: The math library offers utility functions for performing arithmetic operations beyond the basic operators provided by the language. It includes support for floating-point calculations, rounding, and safe mathematical functions to prevent overflow.
    - Example of mathematical utility:
      ```cadence
      let roundedValue = math.round(someNumber, 2)
      ```

  - **String Library (`strings`)**: The strings library contains functions for string manipulation, allowing developers to perform common operations such as concatenation, substring extraction, and formatting.
    - Example of string manipulation:
      ```cadence
      let greeting = "Hello, " + "World!"
      ```

  - **Array Library (`arrays`)**: The arrays library provides utility functions for manipulating arrays, such as sorting, filtering, and searching.
    - Example of array sorting:
      ```cadence
      let sortedArray = arrays.sort([3, 1, 2])
      ```

  - **Dictionary Library (`dictionaries`)**: The dictionaries library offers functions for managing dictionaries, including inserting, updating, and retrieving key-value pairs.
    - Example of dictionary operations:
      ```cadence
      let balance = myDict["Alice"]
      ```

These core libraries provide essential building blocks for smart contract development, ensuring developers can easily interact with data, perform secure computations, and leverage Flow blockchain primitives.

---

### 2. Built-in Functions

Cadence includes a set of **built-in functions** that are available globally. These functions perform common operations, such as cryptographic calculations, resource management, and basic data handling. Built-in functions are optimized for performance and security, ensuring that developers have access to reliable tools for contract development.

- **Description of Frequently Used Built-in Functions**:

  - **Cryptography Functions**:
    - **`crypto.sha3_256(data: [UInt8]): [UInt8]`**: Computes the SHA-3 (256-bit) hash of the given data.
      - Example:
        ```cadence
        let data: [UInt8] = [1, 2, 3]
        let hash = crypto.sha3_256(data)
        ```
    - **`crypto.verify(signature: [UInt8], message: [UInt8], publicKey: PublicKey): Bool`**: Verifies the digital signature of a message using the provided public key.
      - Example:
        ```cadence
        let valid = crypto.verify(signature, message, publicKey)
        ```

  - **Mathematical Functions**:
    - **`math.min(a: Int, b: Int): Int`**: Returns the smaller of two integer values.
      - Example:
        ```cadence
        let minValue = math.min(10, 5)
        ```
    - **`math.max(a: Int, b: Int): Int`**: Returns the larger of two integer values.
      - Example:
        ```cadence
        let maxValue = math.max(10, 5)
        ```

  - **String and Array Manipulation**:
    - **`strings.concat(a: String, b: String): String`**: Concatenates two strings.
      - Example:
        ```cadence
        let fullString = strings.concat("Hello, ", "Cadence!")
        ```
    - **`arrays.contains(array: [T], value: T): Bool`**: Checks if an array contains a specific value.
      - Example:
        ```cadence
        let hasElement = arrays.contains([1, 2, 3], 2)
        ```

  - **Control and Error Handling**:
    - **`assert(condition: Bool, message: String)`**: Asserts that a condition is true. If it is not, the program stops execution and logs the provided message.
      - Example:
        ```cadence
        assert(balance > 0, message: "Balance must be positive")
        ```

    - **`panic(message: String)`**: Immediately halts the execution of the program and raises an error with the provided message.
      - Example:
        ```cadence
        if insufficientFunds {
            panic("Insufficient funds for withdrawal")
        }
        ```

  - **Resource Management**:
    - **`destroy(resource: @T)`**: Explicitly destroys a resource, ensuring that it is safely removed from memory.
      - Example:
        ```cadence
        destroy token
        ```

    - **`create Resource()`**: Used to create a new resource instance in Cadence.
      - Example:
        ```cadence
        let newToken <- create Token()
        ```

  - **Capability and Reference Handling**:
    - **`borrow<&T>(from: Path): &T?`**: Borrows a reference to a resource or data from the specified storage path.
      - Example:
        ```cadence
        let tokenRef = account.borrow<&Token>(from: /storage/myToken)
        ```

    - **`load<@T>(from: Path): @T?`**: Loads a resource from a specified path and moves it out of storage.
      - Example:
        ```cadence
        let loadedToken <- account.load<@Token>(from: /storage/myToken)
        ```

These built-in functions are integral to working with resources, data, and cryptography in Cadence. By providing optimized, secure operations, they allow developers to focus on business logic while ensuring safe, predictable behavior within their smart contracts.

---

The **standard libraries** and **built-in functions** in Cadence give developers access to essential tools for smart contract development. With libraries covering a range of topics, from cryptography to string and array manipulation, and built-in functions that ensure security and correctness, Cadence simplifies the process of creating robust decentralized applications on the Flow blockchain.

## XI. Formal Semantics

Formal semantics provide a rigorous mathematical framework for understanding how Cadence programs execute and interact with resources, ensuring that smart contracts behave as expected. This section details the **operational semantics**, **type soundness**, and **formal verification** capabilities of Cadence.

---

### 1. Operational Semantics

**Operational semantics** describe the formal rules that define how Cadence programs are executed step by step. These rules ensure that all operations, including resource management and contract interactions, follow predictable and secure behavior at runtime.

- **Formal Rules Governing Program Execution**:
  - The operational semantics of Cadence are defined in terms of **transitions** between program states. A state includes the current environment, storage, and control flow (e.g., what instruction or expression is currently being evaluated).
  - **Program State Transitions**: Each instruction in a Cadence program corresponds to a specific state transition, moving the program from one valid state to another.
    - **Expressions**: Evaluating expressions in Cadence involves rules for basic operations (e.g., addition, comparison) and resource management (e.g., moving, borrowing, and destroying resources). Expressions are evaluated within an environment that defines the current variable bindings and references.
      - Example (simplified rule for arithmetic expressions):
        \[
        \frac{(e_1 \Downarrow v_1) \quad (e_2 \Downarrow v_2)}{(e_1 + e_2) \Downarrow (v_1 + v_2)}
        \]
        This rule shows how the sum of two expressions \( e_1 \) and \( e_2 \) is computed by first evaluating both expressions to their values \( v_1 \) and \( v_2 \), and then summing them.

    - **Resource Handling**: Resources follow specific operational rules to ensure correct ownership and lifecycle management. For example, the move operation \( e_1 \leftarrow e_2 \) moves a resource from one location to another, transitioning both the sender's and recipient's environments.
    - **Contracts and Functions**: The execution of contract methods or transactions follows strict rules regarding argument evaluation, scope handling, and result propagation.

  - **Evaluation Rules**: Each statement (e.g., variable declaration, resource movement, function call) has an associated evaluation rule that defines how the program transitions from its current state to the next.
    - Example (simplified rule for resource move):
      \[
      \frac{(e_1 \Downarrow r) \quad \text{owner}(r) = e_2}{(e_2 \leftarrow e_1) \Downarrow \text{new\_owner}(r)}
      \]
      This rule shows that moving a resource \( r \) from one owner \( e_2 \) to another occurs when the expression \( e_1 \) evaluates to the resource \( r \) and transfers ownership.

- **Resource Lifecycle Semantics**: Resources in Cadence follow a **linear type system**, where they can be created, moved, borrowed, and destroyed. The semantics of these operations are captured by precise rules:
  - **Create**: A resource is created using the `create` operation, introducing a new resource into the program state.
  - **Move**: Resources are moved using the `<-` operator, which transfers ownership between entities.
  - **Destroy**: Resources must be explicitly destroyed using the `destroy` operator, removing them from the program’s state.

---

### 2. Type Soundness

**Type soundness** ensures that well-typed programs do not result in runtime type errors. Cadence’s type system is designed to guarantee that, if a program passes type-checking at compile time, it will not encounter type-related errors during execution.

- **Proof or Argument of Type Soundness**:
  - Cadence’s type system is **statically checked**, meaning that types are verified before a program is executed. This includes checking for the correct use of resources, ensuring they are not copied, improperly destroyed, or left in an indeterminate state.
  - **Type Preservation**: If a well-typed program \( P \) undergoes a state transition \( P \rightarrow P' \), then the resulting state \( P' \) is also well-typed. This ensures that types are maintained throughout the execution of the program.
    - **Theorem (Type Preservation)**: If \( \Gamma \vdash P \), and \( P \rightarrow P' \), then \( \Gamma \vdash P' \). This formalizes the notion that the type of the program is preserved during execution.
  - **Progress**: A well-typed program can always make progress toward completion unless it encounters an explicit error (such as a panic). This ensures that programs do not enter an invalid state.
    - **Theorem (Progress)**: If \( \Gamma \vdash P \), then either \( P \) is a value, or there exists a \( P' \) such that \( P \rightarrow P' \).

- **Ensuring Resource Safety**:
  - Cadence’s type system incorporates **linear types** for managing resources. Linear types ensure that resources cannot be duplicated or implicitly discarded. The type system statically checks resource usage to ensure that resources are always either moved or explicitly destroyed.
  - **No Resource Duplication**: The type system guarantees that resources can never be copied, ensuring that they maintain their scarcity and uniqueness.
  - **No Implicit Resource Loss**: Cadence enforces that resources must be explicitly destroyed, ensuring that valuable assets are never unintentionally lost.

- **Example of Type Soundness**:
  ```cadence
  pub fun transferToken(from: AuthAccount, to: AuthAccount, amount: UFix64) {
      let token <- from.load<@Token>(from: /storage/myToken)
      to.save(<-token, to: /storage/theirToken)
  }
  ```
  The type system ensures that the token resource is properly moved from one account to another and is not duplicated or lost.

---

### 3. Formal Verification

Cadence supports **formal verification** techniques to ensure the correctness of smart contracts and their adherence to specified properties. Formal verification enables developers to mathematically prove that their programs behave as expected under all possible inputs and conditions.

- **Support for Formal Verification Techniques**:
  - Cadence encourages the use of **invariants** and preconditions/postconditions in smart contracts. These assertions allow developers to express properties that must always hold true, both before and after certain operations.
  - **Preconditions and Postconditions**: Contracts and functions can define preconditions that must be true before execution and postconditions that must hold after execution. These assertions serve as the basis for formal verification.
    - Example:
      ```cadence
      pub fun transfer(to: Address, amount: UFix64) {
          pre {
              amount > 0: "Amount must be positive"
          }
          post {
              self.balance == old(self.balance) - amount: "Balance must decrease by amount"
          }
      }
      ```

- **Static Analysis for Correctness**:
  - Cadence’s type system and formal verification capabilities are supported by **static analysis tools** that analyze code before deployment. These tools check for common errors such as resource mismanagement, contract violations, and unexpected behavior in transactions.
  - **Resource-Safe Static Analysis**: The static analysis tools ensure that resources are always safely handled, following the move semantics and preventing issues like double spending or resource leaks.
  - **Verification of Invariants**: Static analysis tools can verify that contract invariants, preconditions, and postconditions hold across all possible program paths, ensuring that contracts behave as expected under all conditions.

- **Example of Formal Verification**:
  ```cadence
  pub fun withdraw(amount: UFix64) {
      pre {
          amount <= self.balance: "Cannot withdraw more than the balance"
      }
      self.balance = self.balance - amount
      post {
          self.balance >= 0: "Balance must remain non-negative"
      }
  }
  ```
  This function includes preconditions and postconditions that static analysis tools can verify to ensure the function is safe to execute.

---

Through its **operational semantics**, **type soundness guarantees**, and support for **formal verification**, Cadence provides a secure and predictable environment for writing smart contracts. These formal mechanisms ensure that contracts behave correctly, resources are handled safely, and developers can mathematically prove the correctness of their decentralized applications.

## XII. Security Considerations

Security is a fundamental concern in the design of Cadence, as it is intended for writing smart contracts that manage valuable digital assets on the Flow blockchain. Cadence's type system, resource-oriented programming model, and capability-based access control provide robust security guarantees, making it difficult for common vulnerabilities to occur. This section outlines the key security considerations in Cadence, including resource safety, capability-based security, and the language’s defenses against typical blockchain vulnerabilities.

---

### 1. Resource Safety

One of the primary security features of Cadence is its **resource management system**, which ensures the safe handling of digital assets (e.g., tokens, NFTs). Resources in Cadence are first-class citizens, and the language provides guarantees around their creation, movement, and destruction.

- **Resource Safety Guarantees**:
  - **Ownership and Single Ownership**: Cadence enforces strict ownership rules for resources, meaning that a resource can have only one owner at any given time. Resources are moved, not copied, ensuring that no accidental duplication occurs.
  - **Explicit Destruction**: Resources must be explicitly destroyed when no longer needed. If a resource goes out of scope without being properly handled (moved or destroyed), the program will fail at compile time. This guarantees that valuable assets are not accidentally lost or left in an undefined state.
  - **Prevention of Implicit Loss**: Cadence prevents resources from being implicitly discarded or mishandled by enforcing compile-time checks that ensure resources are always moved or destroyed. This eliminates the risk of losing assets due to programming errors or unexpected control flows.

- **Example**:
  ```cadence
  let myToken <- create Token()    // Token resource created
  let transferredToken <- myToken  // Ownership transferred, myToken no longer accessible
  destroy transferredToken         // Token resource explicitly destroyed
  ```

These safety guarantees make Cadence well-suited for managing digital assets, ensuring that tokens, NFTs, and other resources are handled securely throughout their lifecycle.

---

### 2. Capability-based Security

Cadence's **capability-based security model** is another layer of defense that enhances security in contract interactions. Capabilities define who can access resources and what actions they can perform, providing fine-grained control over permissions.

- **Explanation of Capabilities**:
  - **Capabilities** are references that grant controlled access to resources or data. Rather than exposing direct access to resources, capabilities offer a way to safely delegate limited permissions (e.g., read, write, or transfer rights) to other accounts or contracts.
  - Capabilities are created by linking to resources in an account’s storage, and they specify which operations are allowed, such as reading a balance, transferring tokens, or accessing an NFT.

- **Types of Capabilities**:
  - **Public Capabilities**: Can be accessed by anyone, providing read-only access to certain resources, such as viewing a token balance or NFT metadata.
  - **Private Capabilities**: Only the account owner or explicitly authorized entities can access the resource.
  - **Restricted Capabilities**: Limit the actions that can be performed on a resource (e.g., allowing transfers but not destruction).

- **Enhanced Security via Capabilities**:
  - Capabilities allow developers to securely share access to resources without transferring ownership. They also ensure that only authorized entities can perform sensitive operations like transferring assets or modifying contract state.
  - Since capabilities are controlled by the account that creates them, they can be revoked or modified if necessary, providing additional flexibility and security.

- **Example**:
  ```cadence
  // Define a public read-only capability for a token balance
  let balanceCapability: Capability<&Token> = account.getCapability(/public/tokenBalance)
  
  // Restricted capability allowing transfer of a token
  let transferCapability: Capability<&{Transferable}> = account.getCapability(/public/transferToken)
  ```

By using capabilities, Cadence enables secure interactions between contracts and users, minimizing the risk of unauthorized access or manipulation of assets.

---

### 3. Common Vulnerabilities and Defenses

Cadence has been designed to mitigate several common blockchain-related vulnerabilities, which are frequently encountered in other smart contract languages like Solidity. This section discusses some of these vulnerabilities and how Cadence defends against them.

- **Re-entrancy Attacks**:
  - **Re-entrancy** occurs when a contract is called into during its execution, allowing for unexpected or malicious re-entry points that can exploit contract logic (e.g., draining funds multiple times during an incomplete transaction).
  - **Cadence’s Defense**: Cadence's strict **resource-oriented model** prevents re-entrancy vulnerabilities because resources cannot be duplicated or unexpectedly accessed mid-execution. Additionally, Cadence’s event-driven model and the lack of low-level function calls reduce the risk of re-entrancy.
  
  - **Example of Re-entrancy Mitigation**:
    ```cadence
    pub fun transferToken(token: @Token, to: Address) {
        // Token is moved explicitly, and cannot be accessed after transfer
        destroy token
    }
    ```

- **Integer Overflows and Underflows**:
  - **Integer overflow and underflow** are common issues in many programming languages, where arithmetic operations exceed the bounds of the integer type, potentially causing unexpected behavior.
  - **Cadence’s Defense**: All arithmetic operations in Cadence (e.g., addition, subtraction) are **safe** by default. Operations that would result in overflow or underflow raise errors, preventing contracts from continuing in an invalid state.
  
  - **Example**:
    ```cadence
    let result = 1_000_000_000_000_000_000_000_000_000_000_000 + 1  // Error: Overflow
    ```

- **Access Control Vulnerabilities**:
  - Many blockchain contracts fail due to inadequate access control, allowing unauthorized users to call restricted functions or access sensitive data.
  - **Cadence’s Defense**: By using **capabilities** for access control, Cadence ensures that only authorized entities can perform sensitive operations. Access control is enforced at both the storage and function level, meaning unauthorized access is prevented by design.

- **Phishing and Malicious Contract Interactions**:
  - **Phishing** attacks and malicious contract interactions occur when a user or contract unknowingly interacts with a compromised or hostile contract.
  - **Cadence’s Defense**: Cadence encourages explicit interactions with resources and capabilities. Contracts are written to require explicit authorization and checks before executing sensitive functions, reducing the risk of unintended interactions. Additionally, the use of **transactions** ensures that key actions are authorized by the users themselves.

  - **Example of Access Control via Capabilities**:
    ```cadence
    pub fun executeTransaction(capability: Capability<&Token>, amount: UFix64) {
        let tokenRef = capability.borrow() ?? panic("Invalid capability")
        tokenRef.transfer(amount)
    }
    ```

---

By incorporating these security considerations into its core design, Cadence provides a safer environment for decentralized applications. The combination of resource safety, capability-based access control, and strong defenses against common vulnerabilities helps developers build secure, robust smart contracts that are resistant to attack.

## XIII. Best Practices and Patterns

To help developers build secure, maintainable, and efficient smart contracts, Cadence encourages a set of best practices and patterns that promote code robustness, scalability, and ease of use. Additionally, understanding common mistakes and anti-patterns can help developers avoid errors that can lead to security vulnerabilities or performance issues.

---

### 1. Design Patterns for Smart Contracts

Cadence supports various **design patterns** that can guide developers in writing robust and maintainable smart contracts. These patterns help manage resources, ensure security, and simplify interactions between different components of the Flow blockchain.

#### **1.1. Resource Management Pattern**

Cadence’s resource-oriented programming model emphasizes the safe management of scarce digital assets. Following a **resource management pattern** helps developers effectively handle resources and ensure they are securely transferred, borrowed, or destroyed.

- **Pattern**: Use explicit resource moves and borrowing to ensure that resources are properly handled and never accidentally duplicated or lost.
  - **Example**:
    ```cadence
    pub fun transferToken(sender: AuthAccount, recipient: AuthAccount) {
        let token <- sender.load<@Token>(from: /storage/myToken)!
        recipient.save(<-token, to: /storage/recipientToken)
    }
    ```

#### **1.2. Capability-based Access Control Pattern**

Using **capabilities** to manage access to resources provides fine-grained control over who can access or modify data stored in an account. This pattern is essential for limiting the exposure of sensitive information while still allowing collaboration between different contracts.

- **Pattern**: Use capabilities to grant access to public data or functionality, while keeping private data protected.
  - **Example**:
    ```cadence
    pub fun getBalanceCapability(account: AuthAccount): Capability<&Balance> {
        return account.link<&Balance>(/public/balance, target: /storage/balance)
    }
    ```

#### **1.3. Access Control with Role-Based Functions**

When designing smart contracts, it is often necessary to define roles (e.g., owner, admin) that have specific privileges. Implementing a **role-based access control** pattern ensures that only authorized users can perform certain operations within a contract.

- **Pattern**: Use access control checks in contract methods to limit who can perform critical operations.
  - **Example**:
    ```cadence
    pub contract Vault {
        pub var owner: Address

        pub fun deposit(amount: UFix64) {
            assert(self.owner == AuthAccount, message: "Only owner can deposit")
            // Deposit logic
        }
    }
    ```

#### **1.4. Event-Driven Design Pattern**

Cadence’s event system allows contracts to emit events when significant changes occur. Using an **event-driven design pattern** helps external systems (like UIs or monitoring services) react to changes on the blockchain in real time.

- **Pattern**: Emit events for all important state changes or user actions, allowing external systems to track and react to contract activities.
  - **Example**:
    ```cadence
    pub event Transfer(from: Address, to: Address, amount: UFix64)

    pub fun transferToken(sender: AuthAccount, recipient: Address, amount: UFix64) {
        emit Transfer(from: sender.address, to: recipient, amount: amount)
        // Transfer logic
    }
    ```

#### **1.5. Upgradeable Contract Pattern**

Cadence allows contracts to be upgraded over time, which is crucial for fixing bugs or adding features. An **upgradeable contract pattern** ensures that state and functionality can evolve while maintaining backward compatibility.

- **Pattern**: Separate logic from state so that logic upgrades do not affect existing state. Use account-bound contracts to enable controlled upgrades.
  - **Example**:
    ```cadence
    pub contract VaultV2 {
        pub var state: UFix64

        pub fun upgradeLogic(newLogic: &AnyStruct) {
            // Logic upgrade code
        }
    }
    ```

---

### 2. Common Mistakes and Anti-patterns

Understanding common mistakes and **anti-patterns** in smart contract development can help developers avoid security vulnerabilities, inefficiencies, and maintenance issues. Below is a list of common programming errors in Cadence and guidance on how to avoid them.

#### **2.1. Unintended Resource Duplication**

Since resources in Cadence cannot be duplicated, any attempt to copy a resource results in a compile-time error. However, developers may accidentally attempt to reference a resource without moving it properly.

- **Mistake**: Attempting to copy resources instead of moving them.
  - **How to Avoid**: Always use the `<-` operator when transferring resources between variables or accounts.
  - **Example of Incorrect Usage**:
    ```cadence
    let tokenCopy = token  // Error: Resources cannot be copied
    ```
  - **Correct Usage**:
    ```cadence
    let token <- tokenOwner.moveToken()
    ```

#### **2.2. Inadequate Resource Destruction**

Failing to destroy resources that are no longer needed can lead to resource leakage. Cadence requires resources to be explicitly destroyed when no longer used.

- **Mistake**: Forgetting to destroy resources when they are no longer needed.
  - **How to Avoid**: Always call `destroy` on resources that are not being moved or saved.
  - **Example of Incorrect Usage**:
    ```cadence
    let unusedResource <- create SomeResource()
    // Error: Resource must be destroyed or moved
    ```
  - **Correct Usage**:
    ```cadence
    destroy unusedResource
    ```

#### **2.3. Overuse of Public Storage**

Exposing too much data or resources through **public storage paths** can lead to unintended access, allowing malicious actors to interact with or misuse public data.

- **Anti-pattern**: Storing sensitive data in public storage paths without access control.
  - **How to Avoid**: Use `private` storage paths for sensitive data and limit public exposure to only necessary capabilities.
  - **Correct Usage**:
    ```cadence
    // Store sensitive data privately, expose only capabilities
    account.save(sensitiveData, to: /private/sensitiveData)
    ```

#### **2.4. Not Emitting Events for Important State Changes**

Omitting events for significant actions (like transfers or state changes) can make it difficult for external systems to track contract behavior.

- **Mistake**: Failing to emit events for key contract actions.
  - **How to Avoid**: Emit events for every important state transition or action in the contract.
  - **Correct Usage**:
    ```cadence
    emit Transfer(from: sender, to: recipient, amount: 100.0)
    ```

#### **2.5. Hardcoding Addresses and Constants**

Hardcoding addresses and constants into smart contracts can lead to inflexibility, especially if changes are needed in the future.

- **Anti-pattern**: Hardcoding addresses of key accounts or values into the contract logic.
  - **How to Avoid**: Store addresses in account storage or as configurable parameters that can be updated over time.
  - **Correct Usage**:
    ```cadence
    pub contract Token {
        pub let admin: Address
        init(adminAddress: Address) {
            self.admin = adminAddress
        }
    }
    ```

#### **2.6. Failing to Check Preconditions**

Not checking preconditions (e.g., account balance before withdrawal) can result in contracts behaving unexpectedly and introducing vulnerabilities.

- **Anti-pattern**: Assuming input values are valid without validating them.
  - **How to Avoid**: Use preconditions and postconditions to validate contract logic.
  - **Correct Usage**:
    ```cadence
    pub fun withdraw(amount: UFix64) {
        pre {
            amount > 0: "Amount must be positive"
        }
        // Withdrawal logic
    }
    ```

---

By following these best practices and avoiding common mistakes, developers can build more secure, efficient, and maintainable smart contracts in Cadence. These patterns provide guidance on writing high-quality code, while understanding anti-patterns helps avoid pitfalls that could lead to vulnerabilities or maintenance challenges.

# Appendices

The **appendices** provide additional information and resources to supplement the Cadence Language Specification. This section includes a **glossary** of key terms, **annotated examples** of real-world Cadence contracts and transactions, and a **change log** that tracks modifications to the specification over time.

---

### 1. Glossary

The glossary defines key terms and concepts used throughout the Cadence Language Specification to ensure clarity and understanding.

- **Account**: A Flow blockchain entity that holds state (resources, data) and can deploy contracts or execute transactions.
  
- **Authorization**: The process of verifying that an account has the appropriate permissions to execute a transaction or access certain resources.

- **Borrowing**: The act of temporarily accessing a resource without transferring its ownership. In Cadence, references (`&`) are used to borrow resources.
  
- **Capability**: A reference that grants controlled access to a resource. Capabilities are used to delegate permissions, such as reading or writing to storage.

- **Contract**: A piece of code deployed on the Flow blockchain that encapsulates state and behavior. Contracts typically manage resources and provide methods for interacting with them.

- **Event**: A mechanism used in Cadence to signal important actions or state changes. Events are emitted by contracts and can be observed by external systems.

- **Linear Types**: Types that enforce strict rules about resource usage, ensuring that resources are neither duplicated nor discarded implicitly.

- **Move Semantics**: The process of transferring ownership of a resource from one entity to another. In Cadence, the `<-` operator is used for moving resources.

- **Precondition**: A condition that must be true before a function or transaction can be executed. If a precondition fails, the program will halt.

- **Postcondition**: A condition that must be true after a function or transaction has been executed. Postconditions are used to verify that the program has completed successfully.

- **Resource**: A first-class type in Cadence that represents a scarce asset, such as a token or NFT. Resources have strict rules about ownership and cannot be copied.

- **Storage Path**: A location in an account’s storage where resources and data are stored. Cadence provides three types of storage paths: `storage`, `public`, and `private`.

- **Transaction**: A user-initiated action that interacts with the Flow blockchain. Transactions can move resources, call contract functions, and modify the state.

---

### 2. Annotated Examples

This section provides detailed **annotated examples** of real-world Cadence contracts and transactions. The annotations explain key concepts and guide developers through writing and understanding Cadence code.

#### **Example 1: A Simple Token Contract**

```cadence
// Define a simple contract that manages a fungible token
pub contract SimpleToken {

    // Event emitted whenever a transfer occurs
    pub event Transfer(from: Address, to: Address, amount: UFix64)

    // The total supply of tokens
    pub var totalSupply: UFix64

    // Constructor to initialize the total supply
    init(initialSupply: UFix64) {
        self.totalSupply = initialSupply
    }

    // A function to transfer tokens from one account to another
    pub fun transfer(from: AuthAccount, to: AuthAccount, amount: UFix64) {
        pre {
            amount > 0: "Amount must be positive"
        }
        from.withdraw(amount: amount)
        to.deposit(amount: amount)
        emit Transfer(from: from.address, to: to.address, amount: amount)
    }
}
```

- **Annotation**:
  - **Line 4**: The `Transfer` event is emitted when a token transfer occurs. It logs the sender, recipient, and amount.
  - **Line 9**: The `totalSupply` variable tracks the total amount of tokens in circulation.
  - **Line 12**: The constructor (`init`) initializes the total supply when the contract is deployed.
  - **Line 17**: The `transfer` function moves tokens between accounts, ensuring the amount is positive via a precondition.

#### **Example 2: Transaction to Transfer Tokens**

```cadence
// A transaction that transfers tokens between two accounts
transaction(amount: UFix64, recipient: Address) {

    // The account initiating the transaction
    prepare(signer: AuthAccount) {
        let tokenVault <- signer.load<@SimpleToken.Vault>(from: /storage/myVault)
        let recipientAccount = getAccount(recipient)

        // Move tokens to the recipient's account
        recipientAccount.save(<-tokenVault.withdraw(amount), to: /storage/recipientVault)
    }

    // Emit an event after the transaction completes
    execute {
        log("Tokens transferred successfully")
    }
}
```

- **Annotation**:
  - **Line 3**: The transaction transfers `amount` of tokens to the specified `recipient`.
  - **Line 6**: The `prepare` block loads the token vault from the signer's storage and prepares it for transfer.
  - **Line 9**: The recipient's account is retrieved, and the tokens are transferred to their vault.
  - **Line 13**: A log message confirms that the transfer was successful.

---

### 3. Change Log

The **change log** documents modifications made to the Cadence Language Specification over time. It helps track the evolution of the language and its features, allowing developers to understand changes that may affect their contracts.

- **Version 1.0.0**:
  - Initial release of the Cadence Language Specification.
  - Core features include resource-oriented programming, type system, transaction execution, and capabilities.
  
- **Version 1.1.0**:
  - Added support for zero-knowledge proofs and cryptographic functions.
  - Enhanced capability handling with more fine-grained access control options.
  - Improvements to error handling and resource safety guarantees.
  
- **Version 1.2.0**:
  - Introduced the ability to handle cross-contract calls and native transaction composition.
  - Formal verification tools integrated into the development environment.
  - Extended built-in cryptographic functions with multi-signature support.
  
- **Version 1.3.0** (Planned):
  - Planned support for concurrency primitives and asynchronous function execution.
  - Expanded type system with refinement types and dependent types.

---

The **appendices** section of the Cadence Language Specification serves as a resource for developers, offering key definitions, real-world examples, and a detailed history of changes to the language. It ensures that developers have the tools and knowledge needed to build secure and efficient smart contracts on the Flow blockchain.


