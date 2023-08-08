import "Crypto"

pub fun main(): Bool {
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

    assert(keyList.get(keyIndex: 0) != nil)
    assert(keyList.get(keyIndex: 2) == nil)
    
    return true
}
