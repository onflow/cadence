
pub struct PublicAccount {

    /// The address of the account.
    pub let address: Address

    /// The FLOW balance of the default vault of this account.
    pub let balance: UFix64

    /// The FLOW balance of the default vault of this account that is available to be moved.
    pub let availableBalance: UFix64

    /// The current amount of storage used by the account in bytes.
    pub let storageUsed: UInt64

    /// The storage capacity of the account in bytes.
    pub let storageCapacity: UInt64

    /// The contracts deployed to the account.
    pub let contracts: PublicAccount.Contracts

    /// The keys assigned to the account.
    pub let keys: PublicAccount.Keys

    /// All public paths of this account.
    pub let publicPaths: [PublicPath]

    /// Returns the capability at the given public path.
    pub fun getCapability<T: &Any>(_ path: PublicPath): Capability<T>

    /// Returns the target path of the capability at the given public or private path,
    /// or nil if there exists no capability at the given path.
    pub fun getLinkTarget(_ path: CapabilityPath): Path?

    /// Iterate over all the public paths of an account.
    /// passing each path and type in turn to the provided callback function.
    ///
    /// The callback function takes two arguments:
    ///   1. The path of the stored object
    ///   2. The runtime type of that object
    ///
    /// Iteration is stopped early if the callback function returns `false`.
    ///
    /// The order of iteration, as well as the behavior of adding or removing objects from storage during iteration,
    /// is undefined.
    pub fun forEachPublic(_ function: fun(PublicPath, Type): Bool)

    pub struct Contracts {

        /// The names of all contracts deployed in the account.
        pub let names: [String]

        /// Returns the deployed contract for the contract/contract interface with the given name in the account, if any.
        ///
        /// Returns nil if no contract/contract interface with the given name exists in the account.
        pub fun get(name: String): DeployedContract?

        /// Returns a reference of the given type to the contract with the given name in the account, if any.
        ///
        /// Returns nil if no contract with the given name exists in the account,
        /// or if the contract does not conform to the given type.
        pub fun borrow<T: &Any>(name: String): T?
    }

    pub struct Keys {

        /// Returns the key at the given index, if it exists, or nil otherwise.
        ///
        /// Revoked keys are always returned, but they have `isRevoked` field set to true.
        pub fun get(keyIndex: Int): AccountKey?

        /// Iterate over all unrevoked keys in this account,
        /// passing each key in turn to the provided function.
        ///
        /// Iteration is stopped early if the function returns `false`.
        /// The order of iteration is undefined.
        pub fun forEach(_ function: fun(AccountKey): Bool)

        /// The total number of unrevoked keys in this account.
        pub let count: UInt64
    }
}
