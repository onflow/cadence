## Language Server Protocol Go Types

### Setup

- `npm install -g ts-node`
- `npm install`
- `curl https://raw.githubusercontent.com/golang/tools/master/internal/lsp/protocol/typescript/code.ts -O`
- `curl https://raw.githubusercontent.com/golang/tools/master/internal/lsp/protocol/typescript/util.ts -O`
- `git clone https://github.com/microsoft/vscode-languageserver-node.git`

### Generate

Generate 3 files: 
- Types: `tsprotocol.go` - Uses as-is in `../protocol/types.go`
- Server: `tsserver.go` - Uses Go's internal LS server, we use SourceGraph's. Can be used as inspiration for `../protocol/server.go`
- Client: `tsclient.go` - Not used

Steps:
- `HOME=. ts-node code.ts`
- `rm tsserver.go tsclient.go`
- `gofmt -w tsprotocol.go`
- `mv tsprotocol.go ../protocol/types.go`
