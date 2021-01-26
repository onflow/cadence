# Development

## Running the latest version of the Language Server in the Visual Studio Code Extension

- Ensure that a `replace` statement exists in `languageserver/go.mod`, so that the language server compiles with the local changes to Cadence.

- Find the Visual Studio Code preference named "Cadence: Flow Command" and change it to:

  ```text
  /path/to/cadence/languageserver/run.sh
  ```

- Restart Visual Studio Code

This will automatically recompile the language server every time it is started.

## How is it possible to detect non-determinism and data races in the checker?

Run the checker tests with the `cadence.checkConcurrently` flag, e.g.

```shell
go test -race -v ./runtime/tests/checker -cadence.checkConcurrently=10
```

This runs each check of a checker test 10 times, concurrently,
and asserts that the checker errors of all checks are equal.

