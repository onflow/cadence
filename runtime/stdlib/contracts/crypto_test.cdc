import Test
import Crypto from "crypto.cdc"

access(all)
fun setup() {
    let err = Test.deployContract(
        name: "Crypto",
        path: "crypto.cdc",
        arguments: []
    )

    Test.expect(err, Test.beNil())
}

access(all)
fun testCryptoHash() {
    let hash = Crypto.hash([1, 2, 3], algorithm: HashAlgorithm.SHA3_256)
    Test.assertEqual(32, hash.length)
}

access(all)
fun testCryptoHashWithTag() {
    let hash = Crypto.hashWithTag(
        [1, 2, 3],
        tag: "v0.1.tag",
        algorithm: HashAlgorithm.SHA3_256
    )
    Test.assertEqual(32, hash.length)
}

access(all)
fun testAddKeyToKeyList() {
    let keyList = Crypto.KeyList()

    let publicKey = PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )
    keyList.add(
        publicKey,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 1.0
    )

    Test.assert(keyList.get(keyIndex: 0) != nil)
}

access(all)
fun testGetKeyFromList() {
    let keyList = Crypto.KeyList()

    let publicKey = PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )
    keyList.add(
        publicKey,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 1.0
    )

    Test.assert(keyList.get(keyIndex: 0) != nil)
    Test.assert(keyList.get(keyIndex: 2) == nil)
}

access(all)
fun testRevokeKeyFromList() {
    let keyList = Crypto.KeyList()

    let publicKey = PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )
    keyList.add(
        publicKey,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 0.5
    )

    keyList.revoke(keyIndex: 0)
    keyList.revoke(keyIndex: 2)

    Test.assert(keyList.get(keyIndex: 0)!.isRevoked)
    Test.assert(keyList.get(keyIndex: 2) == nil)
}

access(all)
fun testKeyListVerify() {
    let keyList = Crypto.KeyList()

    let publicKeyA = PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )

    keyList.add(
        publicKeyA,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 0.5
    )

    let publicKeyB = PublicKey(
        publicKey:
            "df9609ee588dd4a6f7789df8d56f03f545d4516f0c99b200d73b9a3afafc14de5d21a4fc7a2a2015719dc95c9e756cfa44f2a445151aaf42479e7120d83df956".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )

    keyList.add(
        publicKeyB,
        hashAlgorithm: HashAlgorithm.SHA3_256,
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

    let isValid = keyList.verify(
        signatureSet: signatureSet,
        signedData: signedData
    )

    Test.assert(isValid)
}

access(all)
fun testKeyListVerifyInsufficientWeights() {
    let keyList = Crypto.KeyList()

    let publicKeyA = PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )

    keyList.add(
        publicKeyA,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 0.4
    )

    let publicKeyB = PublicKey(
        publicKey:
            "df9609ee588dd4a6f7789df8d56f03f545d4516f0c99b200d73b9a3afafc14de5d21a4fc7a2a2015719dc95c9e756cfa44f2a445151aaf42479e7120d83df956".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )

    keyList.add(
        publicKeyB,
        hashAlgorithm: HashAlgorithm.SHA3_256,
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

    let isValid = keyList.verify(
        signatureSet: signatureSet,
        signedData: signedData
    )

    Test.assert(!isValid)
}

access(all)
fun testKeyListVerifyWithRevokedKey() {
    let keyList = Crypto.KeyList()

    let publicKey = PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )
    keyList.add(
        publicKey,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 0.5
    )

    let signatureSet = [
        Crypto.KeyListSignature(
            keyIndex: 0,
            signature:
                "8870a8cbe6f44932ba59e0d15a706214cc4ad2538deb12c0cf718d86f32c47765462a92ce2da15d4a29eb4e2b6fa05d08c7db5d5b2a2cd8c2cb98ded73da31f6".decodeHex()
        )
    ]

    // "foo", encoded as UTF-8, in hex representation
    let signedData = "666f6f".decodeHex()

    keyList.revoke(keyIndex: 0)

    let isValid = keyList.verify(
        signatureSet: signatureSet,
        signedData: signedData
    )

    Test.assert(!isValid)
}

access(all)
fun testKeyListVerifyWithMissingSignature() {
    let keyList = Crypto.KeyList()

    let publicKey = PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )
    keyList.add(
        publicKey,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 0.5
    )

    let signatureSet = [
        Crypto.KeyListSignature(
            keyIndex: 1,
            signature:
                "8870a8cbe6f44932ba59e0d15a706214cc4ad2538deb12c0cf718d86f32c47765462a92ce2da15d4a29eb4e2b6fa05d08c7db5d5b2a2cd8c2cb98ded73da31f6".decodeHex()
        )
    ]

    // "foo", encoded as UTF-8, in hex representation
    let signedData = "666f6f".decodeHex()

    let isValid = keyList.verify(
        signatureSet: signatureSet,
        signedData: signedData
    )

    Test.assert(!isValid)
}

access(all)
fun testKeyListVerifyDuplicateSignature() {
    let keyList = Crypto.KeyList()

    let publicKey = PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )
    keyList.add(
        publicKey,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 0.5
    )

    let signatureSet = [
        Crypto.KeyListSignature(
            keyIndex: 0,
            signature:
                "8870a8cbe6f44932ba59e0d15a706214cc4ad2538deb12c0cf718d86f32c47765462a92ce2da15d4a29eb4e2b6fa05d08c7db5d5b2a2cd8c2cb98ded73da31f6".decodeHex()
        ),
        Crypto.KeyListSignature(
            keyIndex: 0,
            signature:
                "8870a8cbe6f44932ba59e0d15a706214cc4ad2538deb12c0cf718d86f32c47765462a92ce2da15d4a29eb4e2b6fa05d08c7db5d5b2a2cd8c2cb98ded73da31f6".decodeHex()
        )
    ]

    // "foo", encoded as UTF-8, in hex representation
    let signedData = "666f6f".decodeHex()

    let isValid = keyList.verify(
        signatureSet: signatureSet,
        signedData: signedData
    )

    Test.assert(!isValid)
}

access(all)
fun testKeyListVerifyInvalidSignature() {
    let keyList = Crypto.KeyList()

    let publicKey = PublicKey(
        publicKey:
            "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
    )
    keyList.add(
        publicKey,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 0.5
    )

    let signatureSet = [
        Crypto.KeyListSignature(
            keyIndex: 0,
            signature:
                "db70a8cbe6f44932ba59e0d15a706214cc4ad2538deb12c0cf718d86f32c47765462a92ce2da15d4a29eb4e2b6fa05d08c7db5d5b2a2cd8c2cb98ded73da31f6".decodeHex()
        )
    ]

    // "foo", encoded as UTF-8, in hex representation
    let signedData = "666f6f".decodeHex()

    let isValid = keyList.verify(
        signatureSet: signatureSet,
        signedData: signedData
    )

    Test.assert(!isValid)
}
