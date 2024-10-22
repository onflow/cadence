access(all)
contract CCF {
    /// Encodes an encodable value to CCF.
    /// Returns nil if the value cannot be encoded.
    access(all)
    view fun encode(_ input: &Any): [UInt8]?
}
