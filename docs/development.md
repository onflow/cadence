# Development

## Running the latest version of the Language Server in the Visual Studio Code Extension

- Ensure that a `replace` statement exists in `languageserver/go.mod`, so that the language server compiles with the local changes to Cadence.

- Find the Visual Studio Code preference named "Cadence: Flow Command" and change it to:

  ```text
  /path/to/cadence/languageserver/run.sh
  ```

- Restart Visual Studio Code

This will automatically recompile the language server every time it is started.
