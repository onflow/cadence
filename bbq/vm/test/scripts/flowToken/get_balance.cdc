// This script reads the balance field of an account's FlowToken Balance

import "FungibleToken"
import "FlowToken"

access(all) fun main(account: Address): UFix64 {

    let vaultRef = getAccount(account)
        .capabilities.borrow<&FlowToken.Vault>(/public/flowTokenBalance)
        ?? panic("Could not borrow a balance reference to the FlowToken Vault in account "
                .concat(account.toString()).concat(" at path /public/flowTokenBalance. ")
                .concat("Make sure you are querying an address that has ")
                .concat("a FlowToken Vault set up properly at the specified path."))

    return vaultRef.balance
}
