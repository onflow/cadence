# Cadence Contract Analyzer

A tool to analyze Cadence contracts.

## How To Run

Navigate to `<cadence_dir>/tools/contract-anlyzer` directory and run:

```shell
go run main.go <contracts.csv>
```

The CSV should be in the format:

- Header: `address,name,code`
- Columns:
  - `address`: The address of the contract, e.g. `0x1`
  - `name`: The name of the contract, e.g. `Test`
  - `code`: The code of the contract, e.g. `pub contract Test {}`
