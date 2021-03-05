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

func DeclarationKindToSymbolType(kind common.DeclarationKind) protocol.SymbolKind {
	switch kind {
	case common.DeclarationKindContract:
		return protocol.Package
	case common.DeclarationKindField:
		return protocol.Field
	case common.DeclarationKindFunction:
		return protocol.Function
	case common.DeclarationKindArgumentLabel:
		return protocol.TypeParameter
	case common.DeclarationKindConstant:
		return protocol.Constant
	case common.DeclarationKindVariable:
		return protocol.Variable

	// We can unify response for initializer and destructor
	case common.DeclarationKindInitializer:
	case common.DeclarationKindDestructor:
		return protocol.Constructor

	default:
		return protocol.Null
	}
	return 0
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
	kind := DeclarationKindToSymbolType(declaration.DeclarationKind())

	// TODO: can we get additional details here like function signature
	detail := declaration.DeclarationAccess().Description()

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
