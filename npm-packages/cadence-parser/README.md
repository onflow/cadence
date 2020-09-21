# Cadence Parser


## Development 

- Building the parser WASM binary:

  ```sh
  GOARCH=wasm GOOS=js go build -o ./dist/cadence-parser.wasm ../../runtime/cmd/parse
  ```
