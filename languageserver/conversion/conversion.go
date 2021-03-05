/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package conversion

import (
	"fmt"
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
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

func DeclarationKindToSymbolData(declaration ast.Declaration) (symbolKind protocol.SymbolKind, detail string) {
	detail = fmt.Sprintf("access: %s", declaration.DeclarationAccess().Description())
	kind := declaration.DeclarationKind()

	switch kind {
	case common.DeclarationKindContract:
		symbolKind = protocol.Package
	case common.DeclarationKindField:
		symbolKind = protocol.Field
	case common.DeclarationKindFunction:
		symbolKind = protocol.Function
	case common.DeclarationKindArgumentLabel:
		symbolKind = protocol.TypeParameter
	case common.DeclarationKindConstant:
		symbolKind = protocol.Constant
	case common.DeclarationKindVariable:
		symbolKind = protocol.Variable

	// TODO: For some reason events not working and will return empty DocumentSymbol array instead...
/*	case common.DeclarationKindEvent:
		symbolKind = protocol.Event
*/
	// We can unify response for initializer and destructor
	case common.DeclarationKindInitializer:
	case common.DeclarationKindDestructor:
		// "init" and "destroy" don't have access modifiers, so we shall return empty string there
		detail = ""
		symbolKind = protocol.Constructor

	default:
		symbolKind = protocol.Null
	}
	return symbolKind, detail
}

// ASTToDocumentSymbol converts AST Declaration to a DocumentSymbol
//
func ASTToDocumentSymbol(declaration ast.Declaration) protocol.DocumentSymbol {
	var children []protocol.DocumentSymbol

	if declaration.DeclarationMembers() != nil {
		for _, child := range declaration.DeclarationMembers().Declarations() {
			symbolChild := ASTToDocumentSymbol(child)
			children = append(children, symbolChild)
		}
	}

	name := declaration.DeclarationIdentifier().Identifier
	kind, detail := DeclarationKindToSymbolData(declaration)

	symbol := protocol.DocumentSymbol{
		Name:       name,
		Detail:     detail,
		Kind:       kind,
		Deprecated: false,
		Range: protocol.Range{
			Start: ASTToProtocolPosition(declaration.StartPosition()),
			End:   ASTToProtocolPosition(declaration.EndPosition()),
		},
		SelectionRange: protocol.Range{
			Start: ASTToProtocolPosition(declaration.StartPosition()),
			End:   ASTToProtocolPosition(declaration.EndPosition()),
		},
		Children: children,
	}

	return symbol
}
