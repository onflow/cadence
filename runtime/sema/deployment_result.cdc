#compositeType
access(all)
struct DeploymentResult {

    /// The deployed contract.
    ///
    /// If the the deployment was unsuccessful, this will be nil.
    ///
    access(all)
    let deployedContract: DeployedContract?
}