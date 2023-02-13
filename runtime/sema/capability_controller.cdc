pub struct CapabilityController {
    /// The block height when the capability was created.
    pub let issueHeight: UInt64
   
    /// The Type of the capability, i.e.: the T in Capability<T>.
    pub let borrowType: Type
   
    /// The id of the related capability.
    /// This is the UUID of the created capability.
    /// All copies of the same capability will have the same UUID
    pub let capabilityID: UInt64
   
    /// Is the capability revoked.
    pub fun isRevoked(): Bool {}

    /// Returns the targeted storage path of the capability.
    pub fun target(): StoragePath {}
   
    /// Revoke the capability making it no longer usable.
    /// When borrowing from a revoked capability the borrow returns nil.
    pub fun revoke() {}
   
    /// Retarget the capability.
    /// This moves the CapCon from one CapCon array to another.
    pub fun retarget(target: StoragePath) {}
}