access(all)
contract WebAssembly {

    /// Compile WebAssembly binary code into a Module
    /// and instantiate it with the given imports.
    access(all)
    view fun compileAndInstantiate(bytes: [UInt8]): &WebAssembly.InstantiatedSource

    access(all)
    struct InstantiatedSource {

        /// The instance.
        access(all)
        let instance: &WebAssembly.Instance
    }

    struct Instance {

        /// Get the exported value.
        access(all)
        view fun getExport<T: AnyStruct>(name: String): T
    }
}
