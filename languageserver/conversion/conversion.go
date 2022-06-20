/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
		Line:      uint32(pos.Line - 1),
		Character: uint32(pos.Column),
	}
}

// ASTToProtocolRange converts an AST range to a LSP range
//
func ASTToProtocolRange(startPos, endPos ast.Position) protocol.Range {
	return protocol.Range{
		Start: ASTToProtocolPosition(startPos),
		End:   ASTToProtocolPosition(endPos.Shifted(nil, 1)),
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

func DeclarationKindToSymbolKind(kind common.DeclarationKind) protocol.SymbolKind {

	switch kind {
	case common.DeclarationKindFunction:
		return protocol.Function

	case common.DeclarationKindField:
		return protocol.Field

	case common.DeclarationKindConstant,
		common.DeclarationKindParameter:
		return protocol.Constant

	case common.DeclarationKindVariable:
		return protocol.Variable

	case common.DeclarationKindInitializer:
		return protocol.Constructor

	case common.DeclarationKindDestructor:
		return protocol.Function

	case common.DeclarationKindStructure,
		common.DeclarationKindResource,
		common.DeclarationKindEvent,
		common.DeclarationKindContract,
		common.DeclarationKindType:
		return protocol.Class

	case common.DeclarationKindStructureInterface,
		common.DeclarationKindResourceInterface,
		common.DeclarationKindContractInterface:
		return protocol.Interface

	case common.DeclarationKindTransaction:
		return protocol.Namespace
	}

	return 0
}

// DeclarationToDocumentSymbol converts AST Declaration to a DocumentSymbol
//
func DeclarationToDocumentSymbol(declaration ast.Declaration) protocol.DocumentSymbol {
	var children []protocol.DocumentSymbol

	declarationMembers := declaration.DeclarationMembers()
	if declarationMembers != nil {
		for _, child := range declarationMembers.Declarations() {
			childSymbol := DeclarationToDocumentSymbol(child)
			children = append(children, childSymbol)
		}
	}

	declarationKind := declaration.DeclarationKind()

	var name string
	var selectionRange protocol.Range

	identifier := declaration.DeclarationIdentifier()
	if identifier != nil && identifier.Identifier != "" {
		name = identifier.Identifier
		selectionRange = ASTToProtocolRange(
			identifier.StartPosition(),
			identifier.EndPosition(nil),
		)
	} else {
		switch declarationKind {
		case common.DeclarationKindTransaction:
			name = "transaction"
		case common.DeclarationKindInitializer:
			name = "init"
		case common.DeclarationKindDestructor:
			name = "destroy"
		}

		declarationStartPos := ASTToProtocolPosition(declaration.StartPosition())
		selectionRange = protocol.Range{
			Start: declarationStartPos,
			End:   declarationStartPos,
		}
	}

	symbol := protocol.DocumentSymbol{
		Name: name,
		Kind: DeclarationKindToSymbolKind(declarationKind),
		Range: ASTToProtocolRange(
			declaration.StartPosition(),
			declaration.EndPosition(nil),
		),
		SelectionRange: selectionRange,
		Children:       children,
	}

	return symbol
}
