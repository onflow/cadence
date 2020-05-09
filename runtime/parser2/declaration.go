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

package parser2

import (
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

func parseDeclarations(p *parser, endTokenType lexer.TokenType) (declarations []ast.Declaration) {
	for {
		p.skipSpaceAndComments(true)
		switch p.current.Type {
		case lexer.TokenSemicolon:
			p.next()
			continue
		case endTokenType, lexer.TokenEOF:
			return
		default:
			declaration := parseDeclaration(p)
			if declaration == nil {
				return
			}

			declarations = append(declarations, declaration)
		}
	}
}

func parseDeclaration(p *parser) ast.Declaration {
	p.skipSpaceAndComments(true)

	switch p.current.Type {
	case lexer.TokenIdentifier:
		switch p.current.Value {
		case "var", "let":
			return parseVariableDeclaration(p)
		}
	}

	return nil
}

func parseVariableDeclaration(p *parser) *ast.VariableDeclaration {

	// TODO: access

	//  leftTransfer=transfer leftExpression=expression
	//  (rightTransfer=transfer rightExpression=expression)?

	startPos := p.current.Range.StartPos

	isLet := p.current.Value == "let"

	if !p.current.Is(lexer.TokenIdentifier) ||
		!(isLet || p.current.Value == "var") {

		panic(fmt.Errorf("expected 'var' or 'let', got %s", p.current.Type))
	}

	p.next()

	p.skipSpaceAndComments(true)
	if !p.current.Is(lexer.TokenIdentifier) {
		panic(fmt.Errorf("expected identifier after start of variable declaration"))
	}

	identifier := ast.Identifier{
		Identifier: p.current.Value.(string),
		Pos:        p.current.Range.StartPos,
	}

	p.next()

	// TODO: type
	//p.skipSpaceAndComments(true)
	//if p.current.Is(lexer.TokenColon) {
	//
	//}

	p.skipSpaceAndComments(true)
	transfer := parseTransfer(p)
	if transfer == nil {
		panic(fmt.Errorf("expected transfer"))
	}

	p.skipSpaceAndComments(true)

	value := parseExpression(p, lowestBindingPower)
	if value == nil {
		panic(fmt.Errorf("expected initial value for variable"))
	}

	return &ast.VariableDeclaration{
		// TODO: Access
		IsConstant: isLet,
		Identifier: identifier,
		// TODO: TypeAnnotation
		Value:    value,
		Transfer: transfer,
		StartPos: startPos,
		// TODO: SecondTransfer, SecondValue
	}
}

func parseTransfer(p *parser) *ast.Transfer {
	var operation ast.TransferOperation
	switch p.current.Type {
	case lexer.TokenEqual:
		operation = ast.TransferOperationCopy
	case lexer.TokenLeftArrow:
		operation = ast.TransferOperationMove
		// TODO:
		//case lexer.TokenLeftArrowExclamation:
		//	operation = ast.TransferOperationMoveForced
	}

	if operation == ast.TransferOperationUnknown {
		return nil
	}

	startPos := p.current.Range.StartPos

	p.next()

	return &ast.Transfer{
		Operation: operation,
		Pos:       startPos,
	}
}
