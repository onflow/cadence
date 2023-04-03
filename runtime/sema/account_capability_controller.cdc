pub struct AccountCapabilityController {
    /// The type of the controlled capability, i.e. the T in `Capability<T>`.
    pub let borrowType: Type

    /// The identifier of the controlled capability.
    /// All copies of a capability have the same ID.
    pub let capabilityID: UInt64

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
}