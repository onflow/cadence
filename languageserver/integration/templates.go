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

const serviceAccountName = "ServiceAccount"

const contractAccountManager = `
pub contract AccountManager{
    pub event AccountAdded(_ name: String, _ address: Address)

    pub let accountsByName: {String: Address}
    pub let accountsByAddress: {Address: String}
    pub let names: [String]
	pub let superUser: String
    
    init(){
		let superUser = "%s"
		let serviceAddress = Address(0xSERVICE_ACCOUNT_ADDRESS)

        self.accountsByName = {
			superUser: serviceAddress
		}
		self.accountsByAddress = {
			serviceAddress: superUser
		}
        self.names = [
            "Alice", "Bob", "Charlie",
            "Dave", "Eve", "Faythe",
            "Grace", "Heidi", "Ivan",
            "Judy", "Michael", "Niaj",
            "Olivia", "Oscar", "Peggy",
            "Rupert", "Sybil", "Ted",
            "Victor", "Walter"
        ]
		self.superUser = superUser
    }

    pub fun addAccount(_ address: Address){
        var name: String = ""

        let numberOfAccounts = self.accountsByName.keys.length - 1
        let numberOfNames = self.names.length

		if (numberOfAccounts >= numberOfNames){
			// At this point user have created too many accounts, 
			// so he probably don't care about their names anymore
			let index = (numberOfAccounts - numberOfNames).toString()
			name = "zombie-".concat(index)
		} else {
        	name = self.names[numberOfAccounts]
		}
        
		self.accountsByName[name] = address
        self.accountsByAddress[address] = name
        emit AccountAdded(name, address)
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

const scriptGetAddress = `
import AccountManager from 0xSERVICE_ACCOUNT_ADDRESS

pub fun main(name: String): Address? {
    return AccountManager.getAddress(name)
}
`