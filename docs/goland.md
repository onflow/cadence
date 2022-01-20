# GoLand 

## Linter Integration

- Build golangci-lint and the custom analyzers: Run `make build-linter`
- In GoLand go to `Preferences` -> `Tools` -> `File Watchers` -> Add `golangci-lint`
  - File Type: Go files
  - Scope: Project files
  - Program: `/path/to/cadence/tools/golangci-lint/golangci-lint` (NOTE: NOT `~/go/bin/golangci-lint`)
  - Arguments: `run $FileDir$`
  - Advanced Options:
    - Create output file from stdout
    - Show console: Never
    - Output filters: `$FILE_PATH$:$LINE$:$COLUMN$: $MESSAGE$`
 
