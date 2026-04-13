/// StringBuilder provides efficient string concatenation.
/// Use append() to add strings, then call toString() to get the final result.
access(all) struct StringBuilder {

    /// Appends a string to the builder
    access(all) fun append(_ string: String)

    /// Appends a character to the builder
    access(all) fun appendCharacter(_ character: Character)

    /// Clears the builder, allowing it to be reused
    access(all) fun clear()

    /// Returns the built string
    access(all) view fun toString(): String

    /// Returns the current length of the string being built
    access(all) let length: Int
}
