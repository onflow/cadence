/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package parser

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser/lexer"
)

func parseParameterList(p *parser) (*ast.ParameterList, error) {
	var parameters []*ast.Parameter

	p.skipSpaceAndComments()

	if !p.current.Is(lexer.TokenParenOpen) {
		return nil, p.syntaxError(
			"expected %s as start of parameter list, got %s",
			lexer.TokenParenOpen,
			p.current.Type,
		)
	}

	startPos := p.current.StartPos
	// Skip the opening paren
	p.next()

	var endPos ast.Position

	expectParameter := true

	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments()
		switch p.current.Type {
		case lexer.TokenIdentifier:
			if !expectParameter {
				p.report(&MissingCommaInParameterListError{
					Pos: p.current.StartPos,
				})
			}
			parameter, err := parseParameter(p)
			if err != nil {
				return nil, err
			}

			parameters = append(parameters, parameter)
			expectParameter = false

		case lexer.TokenComma:
			if expectParameter {
				return nil, p.syntaxError(
					"expected parameter or end of parameter list, got %s",
					p.current.Type,
				)
			}
			// Skip the comma
			p.next()
			expectParameter = true

		case lexer.TokenParenClose:
			endPos = p.current.EndPos
			// Skip the closing paren
			p.next()
			atEnd = true

		case lexer.TokenEOF:
			return nil, p.syntaxError(
				"missing %s at end of parameter list",
				lexer.TokenParenClose,
			)

		default:
			if expectParameter {
				return nil, p.syntaxError(
					"expected parameter or end of parameter list, got %s",
					p.current.Type,
				)
			} else {
				return nil, p.syntaxError(
					"expected comma or end of parameter list, got %s",
					p.current.Type,
				)
			}
		}
	}

	return ast.NewParameterList(
		p.memoryGauge,
		parameters,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			endPos,
		),
	), nil
}

func parseParameter(p *parser) (*ast.Parameter, error) {
	p.skipSpaceAndComments()

	startPos := p.current.StartPos
	parameterPos := startPos

	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected argument label or parameter name, got %s",
			p.current.Type,
		)
	}

	var argumentLabel string
	parameterName := string(p.currentTokenSource())

	// Skip the identifier
	p.nextSemanticToken()

	// If another identifier is provided, then the previous identifier
	// is the argument label, and this identifier is the parameter name
	if p.current.Is(lexer.TokenIdentifier) {
		argumentLabel = parameterName
		parameterName = string(p.currentTokenSource())
		parameterPos = p.current.StartPos
		// Skip the identifier
		p.nextSemanticToken()
	}

	if !p.current.Is(lexer.TokenColon) {
		return nil, p.syntaxError(
			"expected %s after argument label/parameter name, got %s",
			lexer.TokenColon,
			p.current.Type,
		)
	}

	// Skip the colon
	p.nextSemanticToken()

	typeAnnotation, err := parseTypeAnnotation(p)
	if err != nil {
		return nil, err
	}

	return ast.NewParameter(
		p.memoryGauge,
		argumentLabel,
		ast.NewIdentifier(
			p.memoryGauge,
			parameterName,
			parameterPos,
		),
		typeAnnotation,
		startPos,
	), nil
}

func parseTypeParameterList(p *parser) (*ast.TypeParameterList, error) {
	var typeParameters []*ast.TypeParameter

	p.skipSpaceAndComments()

	if !p.current.Is(lexer.TokenLess) {
		return nil, nil
	}

	startPos := p.current.StartPos
	// Skip the opening paren
	p.next()

	var endPos ast.Position

	expectTypeParameter := true

	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments()
		switch p.current.Type {
		case lexer.TokenIdentifier:
			if !expectTypeParameter {
				p.report(&MissingCommaInParameterListError{
					Pos: p.current.StartPos,
				})
			}
			typeParameter, err := parseTypeParameter(p)
			if err != nil {
				return nil, err
			}

			typeParameters = append(typeParameters, typeParameter)
			expectTypeParameter = false

		case lexer.TokenComma:
			if expectTypeParameter {
				return nil, p.syntaxError(
					"expected type parameter or end of type parameter list, got %s",
					p.current.Type,
				)
			}
			// Skip the comma
			p.next()
			expectTypeParameter = true

		case lexer.TokenGreater:
			endPos = p.current.EndPos
			// Skip the closing paren
			p.next()
			atEnd = true

		case lexer.TokenEOF:
			return nil, p.syntaxError(
				"missing %s at end of type parameter list",
				lexer.TokenGreater,
			)

		default:
			if expectTypeParameter {
				return nil, p.syntaxError(
					"expected parameter or end of type parameter list, got %s",
					p.current.Type,
				)
			} else {
				return nil, p.syntaxError(
					"expected comma or end of type parameter list, got %s",
					p.current.Type,
				)
			}
		}
	}

	return ast.NewTypeParameterList(
		p.memoryGauge,
		typeParameters,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			endPos,
		),
	), nil
}

func parseTypeParameter(p *parser) (*ast.TypeParameter, error) {
	p.skipSpaceAndComments()

	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected type parameter name, got %s",
			p.current.Type,
		)
	}

	identifier := p.tokenToIdentifier(p.current)
	p.nextSemanticToken()

	var err error
	var typeBound *ast.TypeAnnotation
	if p.current.Is(lexer.TokenColon) {
		p.nextSemanticToken()

		typeBound, err = parseTypeAnnotation(p)
		if err != nil {
			return nil, err
		}

	}

	return ast.NewTypeParameter(
		p.memoryGauge,
		identifier,
		typeBound,
	), nil
}

func parseFunctionDeclaration(
	p *parser,
	functionBlockIsOptional bool,
	access ast.Access,
	accessPos *ast.Position,
	staticPos *ast.Position,
	nativePos *ast.Position,
	docString string,
) (*ast.FunctionDeclaration, error) {

	startPos := ast.EarliestPosition(p.current.StartPos, accessPos, staticPos, nativePos)

	// Skip the `fun` keyword
	p.nextSemanticToken()
	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected identifier after start of function declaration, got %s",
			p.current.Type,
		)
	}

	identifier := p.tokenToIdentifier(p.current)

	// Skip the identifier
	p.next()

	var typeParameterList *ast.TypeParameterList

	var err error
	typeParameterList, err = parseTypeParameterList(p)
	if err != nil {
		return nil, err
	}

	parameterList, returnTypeAnnotation, functionBlock, err :=
		parseFunctionParameterListAndRest(p, functionBlockIsOptional)

	if err != nil {
		return nil, err
	}

	return ast.NewFunctionDeclaration(
		p.memoryGauge,
		access,
		staticPos != nil,
		nativePos != nil,
		identifier,
		typeParameterList,
		parameterList,
		returnTypeAnnotation,
		functionBlock,
		startPos,
		docString,
	), nil
}

func parseFunctionParameterListAndRest(
	p *parser,
	functionBlockIsOptional bool,
) (
	parameterList *ast.ParameterList,
	returnTypeAnnotation *ast.TypeAnnotation,
	functionBlock *ast.FunctionBlock,
	err error,
) {
	// Parameter list

	parameterList, err = parseParameterList(p)
	if err != nil {
		return
	}

	// Optional return type

	current := p.current
	cursor := p.tokens.Cursor()
	p.skipSpaceAndComments()
	if p.current.Is(lexer.TokenColon) {
		// Skip the colon
		p.nextSemanticToken()

		returnTypeAnnotation, err = parseTypeAnnotation(p)
		if err != nil {
			return
		}
	} else {
		p.tokens.Revert(cursor)
		p.current = current
	}

	// (Potentially optional) block

	if functionBlockIsOptional {
		current = p.current
		cursor := p.tokens.Cursor()
		p.skipSpaceAndComments()
		if !p.current.Is(lexer.TokenBraceOpen) {
			p.tokens.Revert(cursor)
			p.current = current
			return
		}
	}

	functionBlock, err = parseFunctionBlock(p)
	if err != nil {
		return
	}

	return
}
