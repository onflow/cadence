package integration


const deployContractTemplate = `
transaction(name: String, code: [UInt8]) {
  prepare(signer: AuthAccount) {
    if signer.contracts.get(name: name) == nil {
      signer.contracts.add(name: name, code: code)
    } else {
      signer.contracts.update__experimental(name: name, code: code)
    }
  }
}
`

const contractAccountManager = `
pub contract AccountManager{
    pub event AliasAdded(_ name: String, _ address: Address)

    pub let accountsByName: {String: Address}
    pub let accountsByAddress: {Address: String}
    pub let names: [String]
    
    init(){
        self.accountsByName = {}
		self.accountsByAddress = {}
        self.names = [
            "Alice", "Bob", "Charlie",
            "Dave", "Eve", "Faythe",
            "Grace", "Heidi", "Ivan",
            "Judy", "Michael", "Niaj",
            "Olivia", "Oscar", "Peggy",
            "Rupert", "Sybil", "Ted",
            "Victor", "Walter"
        ]
    }

    pub fun addAccount(_ address: Address){
        let name = self.names[self.accountsByName.keys.length]
        self.accountsByName[name] = address
        self.accountsByAddress[address] = name
        emit AliasAdded(name, address)
    }

    pub fun getAddress(_ name: String): Address?{
        return self.accountsByName[name]
    }

    pub fun getName(_ address: Address): String?{
        return self.accountsByAddress[address]
    }

    pub fun getAccounts():[String]{
        let accounts: [String] = []
        for name in self.accountsByName.keys {
            let address = self.accountsByName[name]!
            let account = name.concat(":")
                            .concat(address.toString())
            accounts.append(account)
        }
        return accounts
    }
}
`

const transactionAddAccount = `
import AccountManager from 0xSERVICE_ACCOUNT_ADDRESS

transaction(address: Address){
  prepare(signer: AuthAccount) {
    AccountManager.addAccount(address)
	log("Account added to ledger")
  }
}
`
