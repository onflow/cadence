
pub struct interface SignatureVerifier  {

    pub fun verify(
        signature: [UInt8],
        tag: String,
        signedData: [UInt8],
        publicKey: [UInt8],
        signatureAlgorithm: String,
        hashAlgorithm: String
    ): Bool
}

pub contract Crypto {

    pub struct SignatureAlgorithm {
        pub let name: String

        init(name: String) {
            self.name = name
        }
    }

    // ECDSA_P256 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the NIST P-256 curve
    pub let ECDSA_P256: SignatureAlgorithm

    // ECDSA_Secp256k1 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the secp256k1 curve
    pub let ECDSA_Secp256k1: SignatureAlgorithm

    pub struct HashAlgorithm {
        pub let name: String

        init(name: String) {
            self.name = name
        }
    }

    // SHA2_256 is Secure Hashing Algorithm 2 (SHA-2) with a 256-bit digest
    pub let SHA2_256: HashAlgorithm

    // SHA3_256 is Secure Hashing Algorithm 3 (SHA-3) with a 256-bit digest
    pub let SHA3_256: HashAlgorithm

    pub struct PublicKey {
        pub let publicKey: [UInt8]
        pub let signatureAlgorithm: SignatureAlgorithm

        init(publicKey: [UInt8], signatureAlgorithm: SignatureAlgorithm) {
            self.publicKey = publicKey
            self.signatureAlgorithm = signatureAlgorithm
        }
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

        // Adds a new key with the given weight
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

        // Returns the key at the given index, if it exists.
        // Revoked keys are always returned, but they have `isRevoked` field set to true
        pub fun get(keyIndex: Int): KeyListEntry? {
            if keyIndex >= self.entries.length {
                return nil
            }

            return self.entries[keyIndex]
        }

        // Marks the key at the given index revoked, but does not delete it
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

                if !Crypto.signatureVerifier.verify(
                    signature: signature.signature,
                    tag: Crypto.domainSeparationTagUser,
                    signedData: signedData,
                    publicKey: key.publicKey.publicKey,
                    signatureAlgorithm: key.publicKey.signatureAlgorithm.name,
                    hashAlgorithm:key.hashAlgorithm.name
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

    priv let signatureVerifier: {SignatureVerifier}

    init(signatureVerifier: {SignatureVerifier}) {

        self.signatureVerifier = signatureVerifier

        // Initialize constants

        self.ECDSA_P256 = SignatureAlgorithm(name: "ECDSA_P256")
        self.ECDSA_Secp256k1 = SignatureAlgorithm(name: "ECDSA_Secp256k1")

        self.SHA2_256 = HashAlgorithm(name: "SHA2_256")
        self.SHA3_256 = HashAlgorithm(name: "SHA3_256")

        self.domainSeparationTagUser = "user"
    }
}