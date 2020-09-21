# Cadence Language Server

## Development

### Building for WebAssembly

```sh
GOARCH=wasm GOOS=js go build -o languageserver.wasm  ./cmd/languageserver
```
