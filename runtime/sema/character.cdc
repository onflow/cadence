
access(all) struct Character: Storable, Equatable, Comparable, Exportable, Importable {

    /// Returns this character as a String
    access(all) fun toString(): String
}
