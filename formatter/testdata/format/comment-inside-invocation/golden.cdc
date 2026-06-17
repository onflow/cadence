access(all) struct VerifyResult {
    access(all) let canExecute: Bool
    access(all) let requiredBalance: UFix64

    init(canExecute: Bool, requiredBalance: UFix64) {
        self.canExecute = canExecute
        self.requiredBalance = requiredBalance
    }
}

access(all) fun verify(balance: UFix64): VerifyResult {
    return VerifyResult(
        // The transaction can be executed if the balance is sufficient.
        canExecute: balance >= 10.0,
        requiredBalance: 10.0
    )
}
