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
