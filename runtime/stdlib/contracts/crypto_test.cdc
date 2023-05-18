import Test

pub var blockchain = Test.newEmulatorBlockchain()
pub var account = blockchain.createAccount()

pub fun setup() {
    blockchain.useConfiguration(Test.Configuration({
        "Crypto": account.address
    }))

    var crypto = Test.readFile("crypto.cdc")
    var err = blockchain.deployContract(
        name: "Crypto",
        code: crypto,
        account: account,
        arguments: []
    )

    Test.assert(err == nil)
}

pub fun testCryptoHash() {
    let returnedValue = executeScript("./scripts/crypto_hash.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testCryptoHashWithTag() {
    let returnedValue = executeScript("./scripts/crypto_hash_with_tag.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testAddKeyToKeyList() {
    let returnedValue = executeScript("./scripts/crypto_key_list_add.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testGetKeyFromList() {
    let returnedValue = executeScript("./scripts/crypto_get_key_from_list.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testRevokeKeyFromList() {
    let returnedValue = executeScript("./scripts/crypto_revoke_key_from_list.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testKeyListVerify() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testKeyListVerifyInsufficientWeights() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_insufficient_weights.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testKeyListVerifyWithRevokedKey() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_revoked.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testKeyListVerifyWithMissingSignature() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_missing_signature.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testKeyListVerifyDuplicateSignature() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_duplicate_signature.cdc")
    Test.assert(returnedValue, message: "found: false")
}

pub fun testKeyListVerifyInvalidSignature() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_invalid_signature.cdc")
    Test.assert(returnedValue, message: "found: false")
}

priv fun executeScript(_ scriptPath: String): Bool {
    var script = Test.readFile(scriptPath)
    let value = blockchain.executeScript(script, [])

    Test.assert(value.status == Test.ResultStatus.succeeded)

    return value.returnValue! as! Bool
}
