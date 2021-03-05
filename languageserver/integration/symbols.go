package integration

import (
	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime/sema"
)

func (i *FlowIntegration) documentSymbols(
	uri protocol.DocumentUri,
	version float64,
	checker *sema.Checker,
) (
	symbols []*protocol.DocumentSymbol,
	err error,
) {
	program := checker.Program
	symbols  = []*protocol.DocumentSymbol{}

	for _, declaration := range program.Declarations() {
		symbol := conversion.ASTToDocumentSymbol(declaration)
		symbols = append(symbols, &symbol)
	}

	return
}

