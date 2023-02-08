
pub struct Block {

    /// The height of the block.
    ///
    /// If the blockchain is viewed as a tree with the genesis block at the root,
    /// the height of a node is the number of edges between the node and the genesis block
    ///
    pub let height: UInt64

    /// The view of the block.
    ///
    /// It is a detail of the consensus algorithm. It is a monotonically increasing integer and counts rounds in the consensus algorithm.
    /// Since not all rounds result in a finalized block, the view number is strictly greater than or equal to the block height
    ///
    pub let view: UInt64

    /// The timestamp of the block.
    ///
    /// Unix timestamp of when the proposer claims it constructed the block.
    ///
    /// NOTE: It is included by the proposer, there are no guarantees on how much the time stamp can deviate
    // from the true time the block was published.
    /// Consider observing blocks' status changes off-chain yourself to get a more reliable value.
    ///
    pub let timestamp: UFix64

    /// The ID of the block.
    /// It is essentially the hash of the block
    pub let id: [UInt8; 32]
}
