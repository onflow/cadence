---
title: Environment Information
contentType: REF
---

## Transaction Information

To get the addresses of the signers of a transaction,
use the `address` field of each signing `AuthAccount`
that is passed to the transaction's `prepare` phase.

There is currently no API that allows getting other transaction information.
Please let us know if your use-case demands it by request this feature in an issue.

## Block Information

To get information about a block, the functions `getCurrentBlock` and `getBlock` can be used:

- `cadence•fun getCurrentBlock(): Block`

  Returns the current block, i.e. the block which contains the currently executed transaction.

- `cadence•fun getBlock(at height: UInt64): Block?`

  Returns the block at the given height.
  If the given block does not exist the function returns `nil`.

The `Block` type contains the identifier, height, and timestamp:

```cadence
pub struct Block {
    /// The ID of the block.
    ///
    /// It is essentially the hash of the block.
    ///
    pub let id: [UInt8; 32]

    /// The height of the block.
    ///
    /// If the blockchain is viewed as a tree with the genesis block at the root,
    // the height of a node is the number of edges between the node and the genesis block
    ///
    pub let height: UInt64

    /// The view of the block.
    ///
    /// It is a detail of the consensus algorithm. It is a monotonically increasing integer
    /// and counts rounds in the consensus algorithm. Since not all rounds result in a finalized block,
    /// the view number is strictly greater than or equal to the block height
    ///
    pub let view: UInt64

    /// The timestamp of the block.
    ///
    /// It is the local clock time of the block proposer when it generates the block
    ///
    pub let timestamp: UFix64
}
```

