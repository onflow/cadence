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


### Debugging

#### GoLang and VSCode
You can setup debugging of Language Server in VSCode by first going in `VSCode Cadence > 
Extension Settings` and under `Cadence: Flow Command` putting absolute location of `run.sh` script 
found in this directory (example: `/Users/dapper/Dev/cadence/languageserver/run.sh`).

After you set the run script you should follow ["Attach to a process on a local machine" tutorial to debug the language server in GoLand](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#step-2-build-the-application).

**M1 Problems**
Currently running `gops` on OSX ARM architecture won't work. A workaround for debugging is to use 
functionality in `tests/util.go`. You can use utility functions to log to a file like so:
```go
test.Log("test log")
```
And at the same time run the command in your terminal:
```bash
tail -f ./debug.log
```
Doing so you should see the output of all Log calls you make. 

*Note: this method works on all architectures*
