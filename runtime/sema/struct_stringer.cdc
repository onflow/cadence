/// StructStringer is an interface implemented by all the string convertible structs.
access(all) 
struct interface StructStringer {
    /// Returns the string representation of this object.
    access(all)
    view fun toString(): String
}
