---
title: Crypto
contentType: REF
---

The built-in contract `Crypto` can be used to perform cryptographic operations.
The contract can be imported using `import Crypto`.

For example, to verify two signatures with equal weights for some signed data:

```cadence
import Crypto

pub fun test main() {
    let keyList = Crypto.KeyList()

    let publicKeyA = Crypto.PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: Crypto.ECDSA_P256
    )
    keyList.add(
        publicKeyA,
        hashAlgorithm: Crypto.SHA3_256,
        weight: 0.5
    )

    let publicKeyB = Crypto.PublicKey(
        publicKey:
            "df9609ee588dd4a6f7789df8d56f03f545d4516f0c99b200d73b9a3afafc14de5d21a4fc7a2a2015719dc95c9e756cfa44f2a445151aaf42479e7120d83df956".decodeHex(),
        signatureAlgorithm: Crypto.ECDSA_P256
    )
    keyList.add(
        publicKeyB,
        hashAlgorithm: Crypto.SHA3_256,
        weight: 0.5
    )

    let signatureSet = [
        Crypto.KeyListSignature(
            keyIndex: 0,
            signature:
                "8870a8cbe6f44932ba59e0d15a706214cc4ad2538deb12c0cf718d86f32c47765462a92ce2da15d4a29eb4e2b6fa05d08c7db5d5b2a2cd8c2cb98ded73da31f6".decodeHex()
        ),
        Crypto.KeyListSignature(
            keyIndex: 1,
            signature:
                "bbdc5591c3f937a730d4f6c0a6fde61a0a6ceaa531ccb367c3559335ab9734f4f2b9da8adbe371f1f7da913b5a3fdd96a871e04f078928ca89a83d841c72fadf".decodeHex()
        )
    ]

    // "foo", encoded as UTF-8, in hex representation
    let signedData = "666f6f".decodeHex()

    let isValid = keyList.isValid(
        signatureSet: signatureSet,
        signedData: signedData
    )
}
```

The API of the Crypto contract is:

```cadence
pub contract Crypto {

    pub struct SignatureAlgorithm {
        pub let name: String
    }

    /// ECDSA_P256 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the NIST P-256 curve
    pub let ECDSA_P256: SignatureAlgorithm

    /// ECDSA_Secp256k1 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the secp256k1 curve
    pub let ECDSA_Secp256k1: SignatureAlgorithm

    pub struct HashAlgorithm {
        pub let name: String
    }

    /// SHA2_256 is Secure Hashing Algorithm 2 (SHA-2) with a 256-bit digest
    pub let SHA2_256: HashAlgorithm

    /// SHA3_256 is Secure Hashing Algorithm 3 (SHA-3) with a 256-bit digest
    pub let SHA3_256: HashAlgorithm

    pub struct PublicKey {
        pub let publicKey: [UInt8]
        pub let signatureAlgorithm: SignatureAlgorithm

        init(publicKey: [UInt8], signatureAlgorithm: SignatureAlgorithm)
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
        )
    }

    pub struct KeyList {

        init()

        /// Adds a new key with the given weight
        pub fun add(
            _ publicKey: PublicKey,
            hashAlgorithm: HashAlgorithm,
            weight: UFix64
        )

        /// Returns the key at the given index, if it exists.
        /// Revoked keys are always returned, but they have `isRevoked` field set to true
        pub fun get(keyIndex: Int): KeyListEntry?

        /// Marks the key at the given index revoked, but does not delete it
        pub fun revoke(keyIndex: Int)

        /// Returns true if the given signatures are valid for the given signed data
        pub fun isValid(
            signatureSet: [KeyListSignature],
            signedData: [UInt8]
        ): Bool
    }

    pub struct KeyListSignature {
        pub let keyIndex: Int
        pub let signature: [UInt8]

        pub init(keyIndex: Int, signature: [UInt8])
    }
}
```
