module github.com/onflow/cadence/languageserver

go 1.13

replace github.com/onflow/flow-cli => github.com/bjartek/flow-cli v0.13.5-0.20210803204549-6a7ae67c4f67

require (
	github.com/google/uuid v1.1.2
	github.com/mattn/go-isatty v0.0.12
	github.com/mitchellh/mapstructure v1.3.3
	github.com/onflow/cadence v0.18.1-0.20210621144040-64e6b6fb2337
	github.com/onflow/flow-cli v0.23.1-0.20210621124332-11c4cd22ffb4
	github.com/onflow/flow-go-sdk v0.20.1-0.20210623043139-533a95abf071
	github.com/sourcegraph/jsonrpc2 v0.0.0-20191222043438-96c4efab7ee2
	github.com/spf13/afero v1.1.2
	github.com/stretchr/testify v1.7.0
)
