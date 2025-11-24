// Taken from the NFT Metadata standard, this contract exposes an interface to let
// anyone borrow a contract and resolve views on it.
//
// This will allow you to obtain information about a contract without necessarily knowing anything about it.
// All you need is its address and name and you're good to go!
access(all) contract interface ViewResolver {

    /// Function that returns all the Metadata Views implemented by the resolving contract.
    /// Some contracts may have multiple resource types that support metadata views
    /// so there is an optional parameter to specify which resource type the caller
    /// is requesting views for.
    /// Some contract-level views may be type-agnostic. In that case, the contract
    /// should return the same views regardless of what type is passed in.
    ///
    /// @param resourceType: An optional resource type to return views for
    /// @return An array of Types defining the implemented views. This value will be used by
    ///         developers to know which parameter to pass to the resolveView() method.
    ///
    access(all) view fun getContractViews(resourceType: Type?): [Type]

    /// Function that resolves a metadata view for this token.
    /// Some contracts may have multiple resource types that support metadata views
    /// so there there is an optional parameter for specify which resource type the caller
    /// is looking for views for.
    /// Some contract-level views may be type-agnostic. In that case, the contract
    /// should return the same views regardless of what type is passed in.
    ///
    /// @param resourceType: An optional resource type to return views for
    /// @param view: The Type of the desired view.
    /// @return A structure representing the requested view.
    ///
    access(all) fun resolveContractView(resourceType: Type?, viewType: Type): AnyStruct?

    /// Provides access to a set of metadata views. A struct or
    /// resource (e.g. an NFT) can implement this interface to provide access to
    /// the views that it supports.
    ///
    access(all) resource interface Resolver {

        /// Same as getViews above, but on a specific NFT instead of a contract
        access(all) view fun getViews(): [Type]

        /// Same as resolveView above, but on a specific NFT instead of a contract
        access(all) fun resolveView(_ view: Type): AnyStruct?
    }

    /// A group of view resolvers indexed by ID.
    ///
    access(all) resource interface ResolverCollection {
        access(all) view fun borrowViewResolver(id: UInt64): &{Resolver}? {
            return nil
        }

        access(all) view fun getIDs(): [UInt64] {
            return []
        }
    }
}
