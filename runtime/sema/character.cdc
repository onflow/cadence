
pub struct Character: Storable, Equatable, Comparable, Exportable, Importable {

    /// The byte array of the UTF-8 encoding
    pub let utf8: [UInt8]

    /// Returns this character as a String
    pub fun toString(): String
}
