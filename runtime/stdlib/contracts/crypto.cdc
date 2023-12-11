
access(all) contract Crypto {

    access(all) fun hash(_ data: [UInt8], algorithm: HashAlgorithm): [UInt8] {
        return algorithm.hash(data)
    }

    access(all) fun hashWithTag(_ data: [UInt8], tag: String, algorithm: HashAlgorithm): [UInt8] {
        return algorithm.hashWithTag(data, tag: tag)
    }

    access(all) struct KeyListEntry {
        access(all) let keyIndex: Int
        access(all) let publicKey: PublicKey
        access(all) let hashAlgorithm: HashAlgorithm
        access(all) let weight: UFix64
        access(all) let isRevoked: Bool

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

    access(all) struct KeyList {

        access(self) let entries: [KeyListEntry]

        init() {
            self.entries = []
        }

        /// Adds a new key with the given weight
        access(all) fun add(
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
        access(all) fun get(keyIndex: Int): KeyListEntry? {
            if keyIndex >= self.entries.length {
                return nil
            }

            return self.entries[keyIndex]
        }

        /// Marks the key at the given index revoked, but does not delete it
        access(all) fun revoke(keyIndex: Int) {
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
        access(all) fun verify(
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

    access(all) struct KeyListSignature {
        access(all) let keyIndex: Int
        access(all) let signature: [UInt8]

        access(all) init(keyIndex: Int, signature: [UInt8]) {
            self.keyIndex = keyIndex
            self.signature = signature
        }
    }

    access(self) let domainSeparationTagUser: String

    init() {
        self.domainSeparationTagUser = "FLOW-V0.0-user"
    }
}
