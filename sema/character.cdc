
access(all)
struct Character: Storable, Primitive, Equatable, Comparable, Exportable, Importable, StructStringer {

    /// The byte array of the UTF-8 encoding.
    access(all)
    let utf8: [UInt8]

    /// Returns this character as a String.
    access(all)
    view fun toString(): String
}
