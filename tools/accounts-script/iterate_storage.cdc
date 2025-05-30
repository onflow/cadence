pub fun main(address: Address): {String: String} {
    let account = getAccount(address)
    // iterate over all storage
    account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
        return true
    })
    account.storage.forEachPublic(fun (path: StoragePath, type: Type): Bool {
        return true
    })

    return "success"
}