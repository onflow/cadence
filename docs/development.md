# Development

## Tools

The [`cmd` directory](https://github.com/onflow/cadence/tree/master/cmd)
contains command-line tools that are useful when working on the implementation for Cadence, or with Cadence code:

- The [`parse`](https://github.com/onflow/cadence/tree/master/cmd/parse) tool
  can be used to parse (syntactically analyze) Cadence code.
  By default, it reports syntactical errors in the given Cadence program, if any, in a human-readable format.
  By providing the `-json` it returns the AST of the program in JSON format if the given program is syntactically valid,
  or syntactical errors in JSON format (including position information).

  ```
  $ echo "X" |  go run ./cmd/parse
  error: unexpected token: identifier
   --> :1:0
    |
  1 | X
    | ^
  ```

  ```
  $ echo "let x = 1" |  go run ./cmd/parse -json
  [
    {
      "program": {
        "Type": "Program",
        "Declarations": [
          {
            "Type": "VariableDeclaration",
            "StartPos": {
              "Offset": 0,
              "Line": 1,
              "Column": 0
            },
            "EndPos": {
              "Offset": 8,
              "Line": 1,
              "Column": 8
            },
            [...]
  ```

- The [`check`](https://github.com/onflow/cadence/tree/master/cmd/check) tool
  can be used to check (semantically analyze) Cadence code.
  By default, it reports semantic errors in the given Cadence program, if any, in a human-readable format.
  By providing the `-json` it returns the AST in JSON format, or semantic errors in JSON format (including position information).

  ```
  $ echo "let x = 1" |  go run ./cmd/check                                                                                                                                                                                        1 ↵
  error: error: missing access modifier for constant
   --> :1:0
    |
  1 | let x = 1
    | ^
  ```

- The [`main`](https://github.com/onflow/cadence/tree/master/cmd/check) tools
  can be used to execute Cadence programs.
  If a no argument is provided, the REPL (Read-Eval-Print-Loop) is started.
  If an argument is provided, the Cadence program at the given path is executed.
  The program must have a function named `main` which has no parameters and no return type.

  ```
   $ go run ./cmd/main                                                                                                                                                                                                           130 ↵
   Welcome to Cadence v0.12.3!
   Type '.help' for assistance.

   1> let x = 2
   2> x + 3
   5
   ```

   ```
   $ echo 'access(all) fun main () { log("Hello, world!") }' > hello.cdc
   $ go run ./cmd/main hello.cdc
   "Hello, world!"
   ```

## How is it possible to detect non-determinism and data races in the checker?

Run the checker tests with the `cadence.checkConcurrently` flag, e.g.

```shell
go test -race -v ./tests/checker -cadence.checkConcurrently=10
```

This runs each check of a checker test 10 times, concurrently,
and asserts that the checker errors of all checks are equal.

