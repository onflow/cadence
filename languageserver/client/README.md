### Cadence Language Server Example Client

### Setup

To compile the language server to WebAssembly and copy the JavaScript support file
to run the WebAssembly binary, run:

```sh
GOARCH=wasm GOOS=js go build -o dist/languageserver.wasm ..
cp $(go env GOROOT)/misc/wasm/wasm_exec.js src
```

### Running

Use the WebPack development server to start and run:

```sh
npm run start
```
