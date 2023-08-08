import Test

access(all) let blockchain = Test.newEmulatorBlockchain()
access(all) let account = blockchain.createAccount()

access(all)
fun setup() {
    blockchain.useConfiguration(Test.Configuration({
        "Crypto": account.address
    }))

    let crypto = Test.readFile("crypto.cdc")
    let err = blockchain.deployContract(
        name: "Crypto",
        code: crypto,
        account: account,
        arguments: []
    )

    Test.expect(err, Test.beNil())
}

access(all)
fun testCryptoHash() {
    let returnedValue = executeScript("./scripts/crypto_hash.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testCryptoHashWithTag() {
    let returnedValue = executeScript("./scripts/crypto_hash_with_tag.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testAddKeyToKeyList() {
    let returnedValue = executeScript("./scripts/crypto_key_list_add.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testGetKeyFromList() {
    let returnedValue = executeScript("./scripts/crypto_get_key_from_list.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testRevokeKeyFromList() {
    let returnedValue = executeScript("./scripts/crypto_revoke_key_from_list.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testKeyListVerify() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testKeyListVerifyInsufficientWeights() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_insufficient_weights.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testKeyListVerifyWithRevokedKey() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_revoked.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testKeyListVerifyWithMissingSignature() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_missing_signature.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testKeyListVerifyDuplicateSignature() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_duplicate_signature.cdc")
    Test.assertEqual(true, returnedValue)
}

access(all)
fun testKeyListVerifyInvalidSignature() {
    let returnedValue = executeScript("./scripts/crypto_key_list_verify_invalid_signature.cdc")
    Test.assertEqual(true, returnedValue)
}

access(self)
fun executeScript(_ scriptPath: String): Bool {
    let script = Test.readFile(scriptPath)
    let scriptResult = blockchain.executeScript(script, [])

    Test.expect(scriptResult, Test.beSucceeded())

    return scriptResult.returnValue! as! Bool
}
