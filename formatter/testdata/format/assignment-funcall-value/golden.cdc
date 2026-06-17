access(all) fun test() {
    var entries: [Entry] = []
    entries[0] = Entry(
        keyIndex: 0,
        publicKey: key,
        hashAlgorithm: algo,
        weight: 1.0,
        isRevoked: true
    )
}
