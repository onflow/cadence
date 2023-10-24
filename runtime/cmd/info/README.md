# info

A command line tool that can show various information.

Available commands:

  - `dump-builtin-types`: Dumps all built-in types and their members, including nested types:

    ```sh
    $ go run ./runtime/cmd/info -nested -members dump-builtin-types
    ...
    - PublicAccount
      - let address: Address
      - let availableBalance: UFix64
      - let balance: UFix64
      - let capabilities: PublicAccount.Capabilities
      - let contracts: PublicAccount.Contracts
      - fun forEachAttachment(_ f: fun(&AnyStructAttachment): Void): Void
      - fun forEachPublic(_ function: fun(PublicPath, Type): Bool): Void
      - view fun getType(): Type
      - view fun isInstance(_ type: Type): Bool
      - let keys: PublicAccount.Keys
      - let publicPaths: [PublicPath]
      - let storageCapacity: UInt64
      - let storageUsed: UInt64
    - PublicAccount.Capabilities
      - view fun borrow<T: &Any>(_ path: PublicPath): T?
      - fun forEachAttachment(_ f: fun(&AnyStructAttachment): Void): Void
      - view fun get<T: &Any>(_ path: PublicPath): Capability<T>?
      - view fun getType(): Type
      - view fun isInstance(_ type: Type): Bool
    ...
    ```

  - `dump-builtin-values`: Dumps all built-in values and their types

    ```sh
    $ go run ./runtime/cmd/info -members dump-builtin-values
    - view fun Address(_ value: Integer): Address
      - view fun fromBytes(_ bytes: [UInt8]): Address
      - view fun fromString(_ input: String): Address?
      - view fun getType(): Type
      - view fun isInstance(_ type: Type): Bool
    ...
    ```
