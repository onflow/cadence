# Resource `NFT`

```cadence
resource NFT {

    id:  UInt64

    data:  MomentData
}
```

 The resource that represents the Moment NFTs

Implemented Interfaces:
 - `NonFungibleToken.INFT`


### Initializer

```cadence
func init(serialNumber UInt32, playID UInt32, setID UInt32)
```


