
access(all)
struct Character: Storable, Equatable, Comparable, Exportable, Importable {

    /// The byte array of the UTF-8 encoding
    access(all)
    let utf8: [UInt8]

    /// Returns this character as a String
    access(all)
    fun toString(): String
}
