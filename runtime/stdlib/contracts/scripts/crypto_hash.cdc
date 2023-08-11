import "Crypto"

access(self) fun main(): Bool {
    let hash = Crypto.hash([1, 2, 3], algorithm: HashAlgorithm.SHA3_256)
    return hash.length == 32
}
