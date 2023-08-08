
access(all) struct PublicAccount {

    /// The address of the account.
    access(all) let address: Address

    /// The FLOW balance of the default vault of this account.
    access(all) let balance: UFix64

    /// The FLOW balance of the default vault of this account that is available to be moved.
    access(all) let availableBalance: UFix64

    /// The current amount of storage used by the account in bytes.
    access(all) let storageUsed: UInt64

    /// The storage capacity of the account in bytes.
    access(all) let storageCapacity: UInt64

    /// The contracts deployed to the account.
    access(all) let contracts: PublicAccount.Contracts

    /// The keys assigned to the account.
    access(all) let keys: PublicAccount.Keys

    /// The capabilities of the account.
    access(all) let capabilities: PublicAccount.Capabilities

    /// All public paths of this account.
    access(all) let publicPaths: [PublicPath]

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
    access(all) fun forEachPublic(_ function: fun(PublicPath, Type): Bool)

    access(all) struct Contracts {

        /// The names of all contracts deployed in the account.
        access(all) let names: [String]

        /// Returns the deployed contract for the contract/contract interface with the given name in the account, if any.
        ///
        /// Returns nil if no contract/contract interface with the given name exists in the account.
        access(all) fun get(name: String): DeployedContract?

        /// Returns a reference of the given type to the contract with the given name in the account, if any.
        ///
        /// Returns nil if no contract with the given name exists in the account,
        /// or if the contract does not conform to the given type.
        access(all) fun borrow<T: &Any>(name: String): T?
    }

    access(all) struct Keys {

        /// Returns the key at the given index, if it exists, or nil otherwise.
        ///
        /// Revoked keys are always returned, but they have `isRevoked` field set to true.
        access(all) fun get(keyIndex: Int): AccountKey?

        /// Iterate over all unrevoked keys in this account,
        /// passing each key in turn to the provided function.
        ///
        /// Iteration is stopped early if the function returns `false`.
        /// The order of iteration is undefined.
        access(all) fun forEach(_ function: fun(AccountKey): Bool)

        /// The total number of unrevoked keys in this account.
        access(all) let count: UInt64
    }

    access(all) struct Capabilities {
        /// get returns the storage capability at the given path, if one was stored there.
        access(all) fun get<T: &Any>(_ path: PublicPath): Capability<T>?

        /// borrow gets the storage capability at the given path, and borrows the capability if it exists.
        ///
        /// Returns nil if the capability does not exist or cannot be borrowed using the given type.
        /// The function is equivalent to `get(path)?.borrow()`.
        access(all) fun borrow<T: &Any>(_ path: PublicPath): T?
    }
}
