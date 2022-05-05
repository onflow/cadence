# Cadence Contract Analyzer

A tool to analyze Cadence contracts.

## How to Build

Navigate to the directory `<cadence_dir>/tools/contract-anlyzer` and run:

```shell
go build -o cadence-analyzer .
```

### Analyzing contracts of an account

To analyze all contracts of an account, specify the network and address.
This requires you have the [Flow CLI](https://docs.onflow.org/flow-cli/) installed and configured.

For example:

```shell
./cadence-analyzer -network mainnet -address 0x1654653399040a61
```

### Analyzing a transaction

To analyze a transaction, specify the network and transaction ID.
This requires you have the [Flow CLI](https://docs.onflow.org/flow-cli/) installed and configured.

For example:

```shell
./cadence-analyzer -network mainnet -transaction 44fd8475eeded90d74e7594b10cf456b0866c78221e7f230fcfd4ba1155c542f
```

### Only running some analyzers

By default, all available analyzers are run.

To list all available analyzers, run:

```shell
./cadence-analyzer -help
```

For example, to only run the `reference-to-optional` and the `external-mutation` analyzers, run:

```shell
./cadence-analyzer -network mainnet -address 0x1654653399040a61 \
    -analyze reference-to-optional \
    -analyze external-mutation
```

### Analyzing contracts in a CSV file

To analyze all contracts in a CSV file, specify the path to the file.

For example:

```shell
./cadence-analyzer -csv contracts.csv
```

The CSV file must be in the following format:

- Header: `location,code`
- Columns:
  - `location`: The location ID of the program
     - Contracts in accounts have the format `A.<address>.<name>`,
        e.g. `A.e467b9dd11fa00df.FlowStorageFees`, where
         - `address`: Address in hex format, e.g. `e467b9dd11fa00df`
         - `name`: The name of the contract, e.g `FlowStorageFees`
     - Transactions have the format `t.<ID>`, where
       - `id`: The ID of the transaction (its hash)
     - Scripts have the format `s.<ID>`, where
       - `id`: The ID of the script (its hash)
  - `code`: The code of the contract, e.g. `pub contract Test {}`

Full example:

```csv
location,code
t.0000000000000000,"
import 0x1

transaction {
    prepare(signer: AuthAccount) {
        Test.hello()
    }
}
"
A.0000000000000001.Test,"
pub contract Test {

    pub fun hello() {
      log(""Hello, world!"")
    }
}
"
```
