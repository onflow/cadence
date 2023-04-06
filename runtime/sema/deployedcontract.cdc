
pub struct DeployedContract {
    /// The address of the account where the contract is deployed at.
    pub let address: Address

    /// The name of the contract.
    pub let name: String

    /// The code of the contract.
    pub let code: [UInt8]

    /// Returns an array of `Type` objects representing all the public type declarations in this contract
    /// (e.g. structs, resources, enums).
    ///
    /// For example, given a contract
    /// ```
    /// contract Foo {
    ///       pub struct Bar {...}
    ///       pub resource Qux {...}
    /// }
    /// ```
    /// then `.publicTypes()` will return an array equivalent to the expression `[Type<Bar>(), Type<Qux>()]`
    pub fun publicTypes(): [Type]
}
