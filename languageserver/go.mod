module github.com/dapperlabs/cadence/languageserver

go 1.13

require (
	github.com/dapperlabs/cadence v0.0.0-20200415220719-726a7f67220a
	github.com/dapperlabs/flow-go-sdk v1.0.0-alpha.2
	github.com/sourcegraph/jsonrpc2 v0.0.0-20191222043438-96c4efab7ee2
	google.golang.org/grpc v1.28.1
)

replace github.com/dapperlabs/cadence => ../
