access(all)
contract RLP {
    /// Decodes an RLP-encoded byte array (called string in the context of RLP).
    /// The byte array should only contain of a single encoded value for a string;
    /// if the encoded value type does not match, or it has trailing unnecessary bytes, the program aborts.
    /// If any error is encountered while decoding, the program aborts.
    access(all)
    fun decodeString(_ input: [UInt8]): [UInt8]


    /// Decodes an RLP-encoded list into an array of RLP-encoded items.
    /// Note that this function does not recursively decode, so each element of the resulting array is RLP-encoded data.
    /// The byte array should only contain of a single encoded value for a list;
    /// if the encoded value type does not match, or it has trailing unnecessary bytes, the program aborts.
    /// If any error is encountered while decoding, the program aborts.
    access(all)
    fun decodeList(_ input: [UInt8]): [[UInt8]]
}
