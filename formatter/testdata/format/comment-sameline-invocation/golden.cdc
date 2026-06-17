access(all) fun test() {
    let x = Metadata(
        counter: epochCounter,
        seed: randomSource,
        totalRewards: 0.0,  // will be overwritten in calculateRewards
        clusters: [],
        keys: []
    )
    process(x)
}
