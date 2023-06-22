import Crypto from "Crypto"

access(self) fun main(): Bool {
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

    assert(keyList.get(keyIndex: 0)!.isRevoked)
    assert(keyList.get(keyIndex: 2) == nil)
    
    return true
}
