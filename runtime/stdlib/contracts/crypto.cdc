
pub struct interface Hasher  {

    pub fun hash(
        data: [UInt8],
        tag: String,
        algorithm: HashAlgorithm
    ): [UInt8]
}

pub contract Crypto {

    pub fun hash(_ data: [UInt8], algorithm: HashAlgorithm): [UInt8] {
        return self.hashWithTag(data, tag: "", algorithm: algorithm)
    }

    pub fun hashWithTag(_ data: [UInt8], tag: String, algorithm: HashAlgorithm): [UInt8] {
        return self.hasher.hash(data: data, tag: tag, algorithm: algorithm)
    }

    pub struct KeyListEntry {
        pub let keyIndex: Int
        pub let publicKey: PublicKey
        pub let hashAlgorithm: HashAlgorithm
        pub let weight: UFix64
        pub let isRevoked: Bool

        init(
            keyIndex: Int,
            publicKey: PublicKey,
            hashAlgorithm: HashAlgorithm,
            weight: UFix64,
            isRevoked: Bool
        ) {
            self.keyIndex = keyIndex
            self.publicKey = publicKey
            self.hashAlgorithm = hashAlgorithm
            self.weight = weight
            self.isRevoked = isRevoked
        }
    }

    pub struct KeyList {

        priv let entries: [KeyListEntry]

        init() {
            self.entries = []
        }

        /// Adds a new key with the given weight
        pub fun add(
            _ publicKey: PublicKey,
            hashAlgorithm: HashAlgorithm,
            weight: UFix64
        ): KeyListEntry {

            let keyIndex = self.entries.length
            let entry = KeyListEntry(
                keyIndex: keyIndex,
                publicKey: publicKey,
                hashAlgorithm: hashAlgorithm,
                weight: weight,
                isRevoked: false
            )
            self.entries.append(entry)
            return entry
        }

        /// Returns the key at the given index, if it exists.
        /// Revoked keys are always returned, but they have `isRevoked` field set to true
        pub fun get(keyIndex: Int): KeyListEntry? {
            if keyIndex >= self.entries.length {
                return nil
            }

            return self.entries[keyIndex]
        }

        /// Marks the key at the given index revoked, but does not delete it
        pub fun revoke(keyIndex: Int) {
            if keyIndex >= self.entries.length {
                return
            }
            let currentEntry = self.entries[keyIndex]
            self.entries[keyIndex] = KeyListEntry(
                keyIndex: currentEntry.keyIndex,
                publicKey: currentEntry.publicKey,
                hashAlgorithm: currentEntry.hashAlgorithm,
                weight: currentEntry.weight,
                isRevoked: true
            )
        }

        /// Returns true if the given signatures are valid for the given signed data
        pub fun isValid(
            signatureSet: [KeyListSignature],
            signedData: [UInt8]
        ): Bool {

            var validWeights: UFix64 = 0.0

            let seenKeyIndices: {Int: Bool} = {}

            for signature in signatureSet {

                // Ensure the key index is valid

                if signature.keyIndex >= self.entries.length {
                    return false
                }

                // Ensure this key index has not already been seen

                if seenKeyIndices[signature.keyIndex] ?? false {
                    return false
                }

                // Record the key index was seen

                seenKeyIndices[signature.keyIndex] = true

                // Get the actual key

                let key = self.entries[signature.keyIndex]

                // Ensure the key is not revoked

                if key.isRevoked {
                    return false
                }

                // Ensure the signature is valid

                if !key.publicKey.verify(
                    signature: signature.signature,
                    signedData: signedData,
                    domainSeparationTag: Crypto.domainSeparationTagUser,
                    hashAlgorithm:key.hashAlgorithm
                ) {
                    return false
                }

                validWeights = validWeights + key.weight
            }

            return validWeights >= 1.0
        }
    }

    pub struct KeyListSignature {
        pub let keyIndex: Int
        pub let signature: [UInt8]

        pub init(keyIndex: Int, signature: [UInt8]) {
            self.keyIndex = keyIndex
            self.signature = signature
        }
    }

    priv let domainSeparationTagUser: String

    priv let hasher: {Hasher}

    init(hasher: {Hasher}) {

        self.hasher = hasher

        // Initialize constants

        self.domainSeparationTagUser = "FLOW-V0.0-user"
    }
}
