# Cadence Language Server

The [Cadence](https://github.com/onflow/cadence) language server compiled to WebAssembly and bundled as an NPM package,
so it can be used in tools written in JavaScript.


## Development 

- Building the language server WASM binary:

  ```sh
  cd ../../languageserver && \
      GOARCH=wasm GOOS=js go build -o ../npm-packages/cadence-language-server/dist/cadence-language-server.wasm ./cmd/languageserver && \
      cd ../npm-packages/cadence-language-server
  ```
