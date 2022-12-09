# unkeyed

A fork of https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/composite.
Reports unkeyed composite literals.

## Usage

The linter is integrated into the golangci-lint configuration.

If you want to run the linter separately or fix reported issues,
first compile the binary:

```sh 
go build .
```

To automatically fix reported issues, invoke the tool with the `-fix` flag and pass the package specifier:

```sh
unkeyed -fix ./...
```

