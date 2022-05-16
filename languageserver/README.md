# Cadence Language Server

The Cadence Language Server implements the [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) (LSP).
It provides editors and IDEs that support LSP language features like auto complete,
go to definition, documentation and type information on hover, etc.

Traditionally providing such features had to repeatedly implemented for each tool,
as each tool provides different APIs for implementing the same feature.
By implementing the LSP, the Cadence Language Server can be re-used in multiple development tools.
For example, it is used by the
[Visual Studio Code extension](https://github.com/onflow/vscode-flow)
(through the [Flow CLI](https://github.com/onflow/flow-cli),
which embeds the language server),
and also in the [Flow Playground](https://play.onflow.org/)
(by compiling the language server to WebAssembly).

## Development

### Main functionality

The main functionality of the language server, such as providing reporting diagnostics (e.g. errors), auto completion, etc. is implemented in the [`server` package](https://github.com/onflow/cadence/tree/master/languageserver/server).

### Integration with the Flow network

The Cadence language server optionally provides integration with the Flow network,
such as signing and submitting transactions.

This code can be found in the [`integration` package](https://github.com/onflow/cadence/tree/master/languageserver/integration).

### Language Server Protocol Types

The Go code for the LSP types can be found in the [`protocol` package](https://github.com/onflow/cadence/tree/master/languageserver/protocol).
The code is generated from the specification's TypeScript declarations using [scripts](https://github.com/onflow/cadence/tree/master/languageserver/scripts).

### Building for WebAssembly

The Cadence language server can be compiled to WebAssembly.
It currently assumes to be used in a JavaScript environment.

```sh
make wasm
```

### Tests

The integration tests for the Cadence Language Server are written in TypeScript
and can be found in the [`test` directory](https://github.com/onflow/cadence/tree/master/languageserver/test).

### Development and Debugging

You can configure the Visual Studio Code extension to use the source of the server in this directory,
instead of the Flow CLI binary, and allowing this server to be debugged, e.g. using GoLand:

1. Ensure the [Delve](https://github.com/go-delve/delve) debugger is installed, for example by running:
    ```shell
    $ go install github.com/go-delve/delve/cmd/dlv@latest
    ```
4. In Visual Studio Code, go to Settings
5. Search for `Cadence: Flow Command`, and enter the full path to the `run.sh` script
   found in this directory (for example: `/Users/dapper/Dev/cadence/languageserver/run.sh`).

This allows the language server to be re-built each time it is restarted:
- Kill Delve: `killall dlv` (Delve ignores SIGINT in headless mode)
- In Visual Studio Code, run the `Cadence: Restart Language Server` command

In addition, it will start the language server through the Delve debugger, by default on port 2345.
This allows you to connect to the debugger and debug the server.

If you are using GoLand, you can follow
["Create the Go Remote run/debug configuration on the client computer"](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#step-3-create-the-remote-run-debug-configuration-on-the-client-computer).
Leave the hostname as `localhost`.

#### Logging

The utility functions in `tests/util.go` can be used to log to a file like so:

```go
test.Log("test log")
```

And at the same time run the command in your terminal:

```bash
tail -f ./debug.log
```

Doing so you should see the output of all Log calls you make. 
