access(all) contract C {
    access(all) event Transfer(
        /// The sender address
        from: Address,
        /// The receiver address
        to: Address,
        /// The amount transferred
        amount: UFix64
    )

    access(all) event Simple(a: Int, b: String)
}
