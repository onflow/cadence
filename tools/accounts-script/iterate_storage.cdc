access(all)
fun main(addresses: [Address]): String {
    for address in addresses {
        let account = getAccount(address)
        // iterate over all storage
        account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
            return true
        })
        account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
            return true
        })
    }
    
    return "success"
}