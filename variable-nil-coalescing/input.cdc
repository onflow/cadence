access(all) fun test() {
    let weights: {UInt64: UInt64} = self.account.storage.copy<{UInt64: UInt64}>(from: /storage/executionEffortWeights)
        ?? panic("weights not set")
}
