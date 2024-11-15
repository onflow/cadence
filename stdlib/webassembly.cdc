access(all)
contract WebAssembly {

    /// Compile WebAssembly binary code into a Module and instantiate it.
    /// Imports are not supported.
    access(all)
    view fun compileAndInstantiate(bytes: [UInt8]): &WebAssembly.InstantiatedSource

    access(all)
    struct InstantiatedSource {

        /// The instance.
        access(all)
        let instance: &WebAssembly.Instance
    }

    access(all)
    struct Instance {

        /// Get the exported value.
        /// The type must match the type of the exported value.
        /// If the export with the given name does not exist,
        /// of if the type does not match, then the function will panic.
        access(all)
        view fun getExport<T: AnyStruct>(name: String): T
    }
}
