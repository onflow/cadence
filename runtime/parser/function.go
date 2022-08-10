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

package parser

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser/lexer"
)

func parsePurityAnnotation(p *parser) (purity ast.FunctionPurity) {
	// get the purity annotation (if one exists) and skip it
	switch p.current.Value {
	case keywordPure:
		purity = ast.PureFunction
		p.next()
		p.skipSpaceAndComments(true)
	}
	return
}

func parseParameterList(p *parser) (parameterList *ast.ParameterList, err error) {
	var parameters []*ast.Parameter

	p.skipSpaceAndComments(true)

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
		p.skipSpaceAndComments(true)
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
	), err
}

func parseParameter(p *parser) (*ast.Parameter, error) {
	p.skipSpaceAndComments(true)

	startPos := p.current.StartPos
	parameterPos := startPos

	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected argument label or parameter name, got %s",
			p.current.Type,
		)
	}
	argumentLabel := ""
	parameterName, ok := p.current.Value.(string)
	if !ok {
		return nil, p.syntaxError(
			"expected parameter %s to be a string",
			p.current,
		)
	}
	// Skip the identifier
	p.next()

	// If another identifier is provided, then the previous identifier
	// is the argument label, and this identifier is the parameter name

	p.skipSpaceAndComments(true)
	if p.current.Is(lexer.TokenIdentifier) {
		argumentLabel = parameterName
		parameterName, ok = p.current.Value.(string)
		if !ok {
			return nil, p.syntaxError(
				"expected parameter %s to be a string",
				p.current,
			)
		}
		parameterPos = p.current.StartPos
		// Skip the identifier
		p.next()
		p.skipSpaceAndComments(true)
	}

	if !p.current.Is(lexer.TokenColon) {
		return nil, p.syntaxError(
			"expected %s after argument label/parameter name, got %s",
			lexer.TokenColon,
			p.current.Type,
		)
	}

	// Skip the colon
	p.next()
	p.skipSpaceAndComments(true)

	typeAnnotation, err := parseTypeAnnotation(p)
	if err != nil {
		return nil, err
	}

	endPos := typeAnnotation.EndPosition(p.memoryGauge)

	return ast.NewParameter(
		p.memoryGauge,
		argumentLabel,
		ast.NewIdentifier(
			p.memoryGauge,
			parameterName,
			parameterPos,
		),
		typeAnnotation,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			endPos,
		),
	), nil
}

func parseFunctionDeclaration(
	p *parser,
	functionBlockIsOptional bool,
	access ast.Access,
	accessPos *ast.Position,
	purity ast.FunctionPurity,
	purityPos *ast.Position,
	docString string,
) (*ast.FunctionDeclaration, error) {

	startPos := p.current.StartPos
	// access modifier will precede purity if both exist
	if purityPos != nil {
		startPos = *purityPos
	}
	if accessPos != nil {
		startPos = *accessPos
	}

	// Skip the `fun` keyword
	p.next()

	p.skipSpaceAndComments(true)
	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected identifier after start of function declaration, got %s",
			p.current.Type,
		)
	}

	identifier := p.tokenToIdentifier(p.current)

	// Skip the identifier
	p.next()

	parameterList, returnTypeAnnotation, functionBlock, err :=
		parseFunctionParameterListAndRest(p, functionBlockIsOptional)

	if err != nil {
		return nil, err
	}

	return ast.NewFunctionDeclaration(
		p.memoryGauge,
		access,
		purity,
		identifier,
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
	parameterList, err = parseParameterList(p)
	if err != nil {
		return
	}

	p.skipSpaceAndComments(true)
	if p.current.Is(lexer.TokenColon) {
		// Skip the colon
		p.next()
		p.skipSpaceAndComments(true)
		returnTypeAnnotation, err = parseTypeAnnotation(p)
		if err != nil {
			return
		}

		p.skipSpaceAndComments(true)
	} else {
		positionBeforeMissingReturnType := parameterList.EndPos
		returnType := ast.NewNominalType(
			p.memoryGauge,
			ast.NewEmptyIdentifier(
				p.memoryGauge,
				positionBeforeMissingReturnType,
			),
			nil,
		)
		returnTypeAnnotation = ast.NewTypeAnnotation(
			p.memoryGauge,
			false,
			returnType,
			positionBeforeMissingReturnType,
		)
	}

	p.skipSpaceAndComments(true)

	if !functionBlockIsOptional ||
		p.current.Is(lexer.TokenBraceOpen) {

		functionBlock, err = parseFunctionBlock(p)
		if err != nil {
			return
		}
	}

	return
}
