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
    pub fun isRevoked(): Bool

    /// Returns the targeted storage path of the capability.
    pub fun target(): StoragePath
   
    /// Delete this capability controller,
    /// and disable the controlled capability and its copies.
    ///
    /// The controller will be deleted from storage,
    /// but the controlled capability and its copies remain.
    ///
    /// Once this function returns, the controller is no longer usable,
    /// all further operations on the controller will panic.
    ///
    /// Borrowing from the controlled capability or its copies will return nil.
    ///
    pub fun delete()
   
    /// Retarget the capability.
    /// This moves the CapCon from one CapCon array to another.
    pub fun retarget(target: StoragePath)
}