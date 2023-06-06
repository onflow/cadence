
access(all) struct interface Vault {
    access(all) balance: Int

    init(balance: Int) {
        post {
            self.balance == balance:
                "the balance must be initialized to the initial balance"
        }
    }

    access(all) fun withdraw(amount: Int): Vault {
        pre {
            amount > 0:
                "withdrawal amount must be positive"
            amount <= self.balance:
                "insufficient funds: the amount must be smaller or equal to the balance"
        }
        post {
            self.balance == before(self.balance) - amount:
                "Incorrect amount removed"
            result.balance == amount: "incorrect amount returned"
        }
    }

    access(all) fun deposit(vault: Vault) {
        post {
            self.balance == before(self.balance) + vault.balance:
                "the amount must be added to the balance"
        }
    }
}

access(all) struct ExampleVault: Vault {
    access(all) var balance: Int

    init(balance: Int) {
        self.balance = balance
    }

    access(all) fun withdraw(amount: Int): Vault {
        self.balance = self.balance - amount
        return ExampleVault(balance: amount)
    }

    access(all) fun deposit(vault: Vault) {
        self.balance = self.balance + vault.balance
    }
}

access(all) fun main() {
    let vaultA = ExampleVault(balance: 10)
    let vaultB = ExampleVault(balance: 0)

    let vaultC = vaultA.withdraw(amount: 7)
    vaultB.deposit(vault: vaultC)

    log(vaultA.balance)
    log(vaultB.balance)
    log(vaultC.balance)
}
