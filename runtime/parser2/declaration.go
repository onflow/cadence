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
		case "fun":
			return parseFunctionDeclaration(p)
		}
	}

	return nil
}

func parseVariableDeclaration(p *parser) *ast.VariableDeclaration {

	// TODO: access

	startPos := p.current.StartPos

	isLet := p.current.Value == "let"

	if !p.current.Is(lexer.TokenIdentifier) ||
		!(isLet || p.current.Value == "var") {

		panic(fmt.Errorf("expected kind kind of variable, 'var' or 'let', got %s", p.current.Type))
	}

	p.next()

	p.skipSpaceAndComments(true)
	if !p.current.Is(lexer.TokenIdentifier) {
		panic(fmt.Errorf("expected identifier after start of variable declaration"))
	}

	identifier := ast.Identifier{
		Identifier: p.current.Value.(string),
		Pos:        p.current.StartPos,
	}

	p.next()
	p.skipSpaceAndComments(true)

	var typeAnnotation *ast.TypeAnnotation

	if p.current.Is(lexer.TokenColon) {
		p.next()
		p.skipSpaceAndComments(true)

		typeAnnotation = parseTypeAnnotation(p)
	}

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

	// TODO: second transfer and value

	return &ast.VariableDeclaration{
		// TODO: Access
		IsConstant:     isLet,
		Identifier:     identifier,
		TypeAnnotation: typeAnnotation,
		Value:          value,
		Transfer:       transfer,
		StartPos:       startPos,
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
	case lexer.TokenLeftArrowExclamation:
		operation = ast.TransferOperationMoveForced
	}

	if operation == ast.TransferOperationUnknown {
		return nil
	}

	startPos := p.current.StartPos

	p.next()

	return &ast.Transfer{
		Operation: operation,
		Pos:       startPos,
	}
}

func parseParameterList(p *parser) (parameterList *ast.ParameterList) {
	var parameters []*ast.Parameter

	p.skipSpaceAndComments(true)

	if !p.current.Is(lexer.TokenParenOpen) {
		panic(fmt.Errorf("expected '(' as start of parameter list, got %s", p.current.Type))
	}

	startPos := p.current.StartPos
	p.next()

	var endPos ast.Position

	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments(true)
		switch p.current.Type {
		case lexer.TokenIdentifier:
			parameter := parseParameter(p)
			parameters = append(parameters, parameter)
		case lexer.TokenComma:
			p.next()
		case lexer.TokenParenClose:
			endPos = p.current.EndPos
			p.next()
			atEnd = true
			break
		case lexer.TokenEOF:
			panic(fmt.Errorf("missing ')' at end of parameter list"))
		default:
			panic(fmt.Errorf("unexpected token in parameter list: %s", p.current.Type))
		}
	}

	return &ast.ParameterList{
		Parameters: parameters,
		Range: ast.Range{
			StartPos: startPos,
			EndPos:   endPos,
		},
	}
}

func parseParameter(p *parser) *ast.Parameter {
	p.skipSpaceAndComments(true)

	startPos := p.current.StartPos
	parameterPos := startPos

	if !p.current.Is(lexer.TokenIdentifier) {
		panic(fmt.Errorf("expected argument label or parameter name, got %s", p.current.Type))
	}
	argumentLabel := ""
	parameterName := p.current.Value.(string)
	p.next()

	// If another identifier is provided, then the previous identifier
	// is the argument label, and this identifier is the parameter name

	p.skipSpaceAndComments(true)
	if p.current.Is(lexer.TokenIdentifier) {
		argumentLabel = parameterName
		parameterName = p.current.Value.(string)
		parameterPos = p.current.StartPos

		p.next()
		p.skipSpaceAndComments(true)
	}

	if !p.current.Is(lexer.TokenColon) {
		panic(fmt.Errorf("expected ':' after argument label/parameter name, got %s", p.current.Type))
	}

	p.next()
	p.skipSpaceAndComments(true)

	typeAnnotation := parseTypeAnnotation(p)

	endPos := typeAnnotation.EndPosition()

	return &ast.Parameter{
		Label: argumentLabel,
		Identifier: ast.Identifier{
			Identifier: parameterName,
			Pos:        parameterPos,
		},
		TypeAnnotation: typeAnnotation,
		Range: ast.Range{
			StartPos: startPos,
			EndPos:   endPos,
		},
	}
}

// Fun identifier parameterList (':' returnType=typeAnnotation)? functionBlock?

func parseFunctionDeclaration(p *parser) *ast.FunctionDeclaration {

	// TODO: access

	startPos := p.current.StartPos

	if !p.current.IsString(lexer.TokenIdentifier, "fun") {
		panic(fmt.Errorf("expected function keyword 'fun', got %s", p.current.Type))
	}

	p.next()

	p.skipSpaceAndComments(true)
	if !p.current.Is(lexer.TokenIdentifier) {
		panic(fmt.Errorf("expected identifier after start of function declaration"))
	}

	identifier := ast.Identifier{
		Identifier: p.current.Value.(string),
		Pos:        p.current.StartPos,
	}

	p.next()

	parameterList := parseParameterList(p)

	var returnTypeAnnotation *ast.TypeAnnotation

	p.skipSpaceAndComments(true)
	if p.current.Is(lexer.TokenColon) {
		p.next()
		p.skipSpaceAndComments(true)
		returnTypeAnnotation = parseTypeAnnotation(p)
		p.skipSpaceAndComments(true)
	}

	// TODO: parse function block
	block := parseBlock(p)
	functionBlock := &ast.FunctionBlock{
		Block: block,
	}

	return &ast.FunctionDeclaration{
		// TODO: Access
		Identifier:           identifier,
		ParameterList:        parameterList,
		ReturnTypeAnnotation: returnTypeAnnotation,
		FunctionBlock:        functionBlock,
		StartPos:             startPos,
	}
}
