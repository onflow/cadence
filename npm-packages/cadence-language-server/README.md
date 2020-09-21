# Cadence Language Server

## Development 

- Building the parser WASM binary:

  ```sh
  cd ../../languageserver && \
      GOARCH=wasm GOOS=js go build -o ../npm-packages/cadence-language-server/dist/cadence-language-server.wasm ./cmd/languageserver && \
      cd ../npm-packages/cadence-language-server
  ```

