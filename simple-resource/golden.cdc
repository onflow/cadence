access(all) resource Vault {
    access(all) var balance: UFix64

    init(balance: UFix64) {
        self.balance = balance
    }

    access(all) fun getBalance(): UFix64 {
        return self.balance
    }
}
