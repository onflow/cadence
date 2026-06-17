transaction(name: String, code: String) {
    prepare(account: auth(UpdateContract) &Account) {
        // Upgrade the contract
        account.contracts.update(name: name, code: code.utf8)
    }
    execute {
        // Log the result
        log("done")
    }
}
