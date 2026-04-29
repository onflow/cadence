access(all) fun test() {
    if !key.publicKey.verify(signature: signature.signature, signedData: signedData, domainSeparationTag: domainSeparationTag, hashAlgorithm: key.hashAlgorithm) {
        return false
    }
}
