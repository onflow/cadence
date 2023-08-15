access(all) struct StorageCapabilityController: ContainFields {

    /// An arbitrary "tag" for the controller.
    /// For example, it could be used to describe the purpose of the capability.
    /// Empty by default.
    access(all) var tag: String

    /// Updates this controller's tag to the provided string
    access(all) fun setTag(_ tag: String)

    /// The type of the controlled capability, i.e. the T in `Capability<T>`.
    access(all) let borrowType: Type

    /// The identifier of the controlled capability.
    /// All copies of a capability have the same ID.
    access(all) let capabilityID: UInt64

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
    access(all) fun delete()

    /// Returns the targeted storage path of the controlled capability.
    access(all) fun target(): StoragePath

    /// Retarget the controlled capability to the given storage path.
    /// The path may be different or the same as the current path.
    access(all) fun retarget(_ target: StoragePath)
}
