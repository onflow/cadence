module github.com/onflow/cadence/languageserver

go 1.13

require (
	github.com/mattn/go-isatty v0.0.12
	github.com/mitchellh/mapstructure v1.3.3
	github.com/onflow/cadence v0.8.0
	github.com/onflow/flow-go-sdk v0.9.0
	github.com/sourcegraph/jsonrpc2 v0.0.0-20191222043438-96c4efab7ee2
	github.com/stretchr/testify v1.5.1
	google.golang.org/grpc v1.30.0
)

replace github.com/onflow/cadence => ../

replace github.com/fxamacker/cbor/v2 => github.com/turbolent/cbor/v2 v2.2.1-0.20200911003300-cac23af49154
