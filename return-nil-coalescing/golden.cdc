access(all) contract FlowExecutionParameters {
    access(all) view fun getExecutionEffortWeights(): {UInt64: UInt64} {
        return self.account.storage.copy<{UInt64: UInt64}>(from: /storage/executionEffortWeights)
                ?? panic("execution effort weights not set yet")
    }
}
