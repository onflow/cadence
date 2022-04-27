/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration

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
