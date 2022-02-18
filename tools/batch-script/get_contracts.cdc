pub fun main(addresses: [Address]): {Address: {String: String}} {
    let accountContracts: {Address: {String: String}} = {}

    for address in addresses {
        let account = getAccount(address)
        let contracts: {String: String} = {}

        if account.storageUsed > 0 as UInt64 {
            // this produces a deterministic error if account does not exist yet
        }

        let names = account.contracts.names
        if names.length == 0 {
            continue
        }

        for name in names {
            contracts[name] = String.encodeHex(account.contracts.get(name: name)!.code)
        }

        accountContracts[address] = contracts
    }

    return accountContracts
}