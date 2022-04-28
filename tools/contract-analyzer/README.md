# Cadence Contract Analyzer

A tool to analyze Cadence contracts.

## How to Build

Navigate to the directory `<cadence_dir>/tools/contract-anlyzer` and run:

```shell
go build .
```

### Analyzing contracts of an account

To analyze all contracts of an account, specify the network and address.
This requires you have the [Flow CLI](https://docs.onflow.org/flow-cli/) installed and configured.

For example:

```shell
./contract-analyzer -network mainnet -address 0x1654653399040a61
```

### Only running some analyzers

By default, all available analyzers are run.

To list all available analyzers, run:

```shell
./contract-analyzer -help
```

For example, to only run the `reference-to-optional` and the `external-mutation` analyzers, run:

```shell
./contract-analyzer -network mainnet -address 0x1654653399040a61 \
    -analyze reference-to-optional \
    -analyze external-mutation
```

### Analyzing contracts in a CSV file

To analyze all contracts in a CSV file, specify the path to the file.

For example:

```shell
./contract-analyzer -csv contracts.csv
```

The CSV file must be in the following format:

- Header: `address,name,code`
- Columns:
  - `address`: The address of the contract, e.g. `0x1`
  - `name`: The name of the contract, e.g. `Test`
  - `code`: The code of the contract, e.g. `pub contract Test {}`
