# change-contract

Takes a `location, code` CSV on stdin and writes it to stdout,
replacing the `code` cell of the row matching `-location`
with the contents of the file given by `-file`.

The first row is preserved as-is (so a `location,code` header passes through).
Rows whose location does not match are emitted unchanged.

The input format matches the CSV produced by the sibling `get-contracts` tool.

## Flags

- `-location` — the location whose code should be replaced
- `-file` — path to a file containing the new code

## Example

```sh
cat contracts.csv | go run . -file SomeContract.cdc -location A.0123456789.SomeContract > updated_contracts.csv
```
