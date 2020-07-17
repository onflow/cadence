package conversion

import (
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

// ASTToProtocolPosition converts an AST position to a LSP position
//
func ASTToProtocolPosition(pos ast.Position) protocol.Position {
	return protocol.Position{
		Line:      float64(pos.Line - 1),
		Character: float64(pos.Column),
	}
}

// ASTToProtocolRange converts an AST range to a LSP range
//
func ASTToProtocolRange(startPos, endPos ast.Position) protocol.Range {
	return protocol.Range{
		Start: ASTToProtocolPosition(startPos),
		End:   ASTToProtocolPosition(endPos.Shifted(1)),
	}
}

// ProtocolToSemaPosition converts a LSP position to a sema position
//
func ProtocolToSemaPosition(pos protocol.Position) sema.Position {
	return sema.Position{
		Line:   int(pos.Line + 1),
		Column: int(pos.Character),
	}
}
