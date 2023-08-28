access(all)
contract BLS {
    /// Aggregates multiple BLS signatures into one,
    /// considering the proof of possession as a defense against rogue attacks.
    ///
    /// Signatures could be generated from the same or distinct messages,
    /// they could also be the aggregation of other signatures.
    /// The order of the signatures in the slice does not matter since the aggregation is commutative.
    /// No subgroup membership check is performed on the input signatures.
    /// The function returns nil if the array is empty or if decoding one of the signature fails.
    access(all)
    fun aggregateSignatures(_ signatures: [[UInt8]]): [UInt8]?


    /// Aggregates multiple BLS public keys into one.
    ///
    /// The order of the public keys in the slice does not matter since the aggregation is commutative.
    /// No subgroup membership check is performed on the input keys.
    /// The function returns nil if the array is empty or any of the input keys is not a BLS key.
    access(all)
    fun aggregatePublicKeys(_ keys: [PublicKey]): PublicKey?
}
