package integration

import (
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime/sema"
)

func (i *FlowIntegration) documentSymbols(
	uri protocol.DocumentUri,
	version float64,
	checker *sema.Checker,
) (
	[]*protocol.DocumentSymbol,
	error,
) {
	// TODO: Implement
	var symbols []*protocol.DocumentSymbol
	return symbols, nil
}

