import Crypto from "Crypto"

access(self) fun main(): Bool {
    let hash = Crypto.hashWithTag(
        [1, 2, 3],
        tag: "v0.1.tag",
        algorithm: HashAlgorithm.SHA3_256
    )
    return hash.length == 32
}
