
access(all) struct DeployedContract {
    /// The address of the account where the contract is deployed at.
    access(all) let address: Address

    /// The name of the contract.
    access(all) let name: String

    /// The code of the contract.
    access(all) let code: [UInt8]

    /// Returns an array of `Type` objects representing all the public type declarations in this contract
    /// (e.g. structs, resources, enums).
    ///
    /// For example, given a contract
    /// ```
    /// contract Foo {
    ///       access(all) struct Bar {...}
    ///       access(all) resource Qux {...}
    /// }
    /// ```
    /// then `.publicTypes()` will return an array equivalent to the expression `[Type<Bar>(), Type<Qux>()]`
    access(all) fun publicTypes(): [Type]
}
