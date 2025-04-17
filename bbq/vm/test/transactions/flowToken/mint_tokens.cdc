import "FungibleToken"
import "FlowToken"

/// This transaction mints tokens using the account that stores the Flow Token Admin resource
/// This is the service account

transaction(recipient: Address, amount: UFix64) {

    let tokenAdmin: &FlowToken.Administrator
    let tokenReceiver: &{FungibleToken.Receiver}

    prepare(signer: auth(BorrowValue) &Account) {

        self.tokenAdmin = signer.storage
            .borrow<&FlowToken.Administrator>(from: /storage/flowTokenAdmin)
            ?? panic("Cannot mint: Signer does not store the FlowToken Admin Resource in their account"
                .concat(" at the path /storage/flowTokenAdmin."))

        self.tokenReceiver = getAccount(recipient)
            .capabilities.borrow<&{FungibleToken.Receiver}>(/public/flowTokenReceiver)
            ?? panic("Could not borrow a Receiver reference to the FlowToken Vault in account "
                .concat(recipient.toString()).concat(" at path /public/flowTokenReceiver")
                .concat(". Make sure you are sending to an address that has ")
                .concat("a FlowToken Vault set up properly at the specified path."))
    }

    execute {
        let minter <- self.tokenAdmin.createNewMinter(allowedAmount: amount)
        let mintedVault <- minter.mintTokens(amount: amount)

        self.tokenReceiver.deposit(from: <-mintedVault)

        destroy minter
    }
}
