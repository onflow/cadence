/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/parser/lexer"
)

func parsePurityAnnotation(p *parser) ast.FunctionPurity {
	// get the purity annotation (if one exists) and skip it
	if p.isToken(p.current, lexer.TokenIdentifier, KeywordView) {
		// Skip the `view` keyword
		p.nextSemanticToken()

		return ast.FunctionPurityView
	}

	return ast.FunctionPurityUnspecified
}

func parseParameterList(p *parser, expectDefaultArguments bool) (*ast.ParameterList, error) {
	var parameters []*ast.Parameter
	var endToken lexer.Token
	var comments ast.Comments

	p.skipSpace()

	if !p.current.Is(lexer.TokenParenOpen) {
		return nil, &MissingStartOfParameterListError{
			GotToken: p.current,
		}
	}

	startToken := p.current
	// Skip the opening paren
	p.next()

	expectParameter := true
	var commaToken lexer.Token
	var atEnd bool
	progress := p.newProgress()

	for !atEnd && p.checkProgress(&progress) {

		p.skipSpace()

		switch p.current.Type {
		case lexer.TokenIdentifier:
			if !expectParameter {
				p.report(&MissingCommaInParameterListError{
					Pos: p.current.StartPos,
				})
			}
			parameter, err := parseParameter(p, expectDefaultArguments)
			if err != nil {
				return nil, err
			}
			parameter.Comments.Leading = append(parameter.Comments.Leading, commaToken.Comments.Trailing...)

			parameters = append(parameters, parameter)
			expectParameter = false

		case lexer.TokenComma:
			if expectParameter {
				return nil, &UnexpectedTokenInParameterListError{
					GotToken: p.current,
				}
			}
			// Skip the comma
			commaToken = p.current
			p.next()
			expectParameter = true

		case lexer.TokenParenClose:
			endToken = p.current
			// Skip the closing paren
			p.next()
			atEnd = true

		case lexer.TokenEOF:
			return nil, &MissingClosingParenInParameterListError{
				Pos: p.current.StartPos,
			}

		default:
			if expectParameter {
				return nil, &UnexpectedTokenInParameterListError{
					GotToken: p.current,
				}
			} else {
				return nil, &ExpectedCommaOrEndOfParameterListError{
					GotToken: p.current,
				}
			}
		}
	}

	if len(parameters) == 0 {
		comments.Leading = append(
			comments.Leading,
			startToken.Comments.All()...,
		)
	} else {
		comments.Leading = append(
			comments.Leading,
			startToken.Comments.Leading...,
		)

		var patched []*ast.Comment
		patched = append(patched, startToken.Comments.Trailing...)
		patched = append(patched, parameters[0].Comments.Leading...)
		parameters[0].Comments.Leading = patched
	}
	comments.Trailing = append(
		comments.Trailing,
		endToken.Comments.All()...,
	)

	return ast.NewParameterList(
		p.memoryGauge,
		parameters,
		ast.NewRange(
			p.memoryGauge,
			startToken.StartPos,
			endToken.EndPos,
		),
		comments,
	), nil
}

func parseParameter(p *parser, expectDefaultArgument bool) (*ast.Parameter, error) {
	p.skipSpace()

	startToken := p.current

	argumentLabel := ""
	identifier, err := p.nonReservedIdentifier("for argument label or parameter name")

	if err != nil {
		return nil, err
	}

	// Skip the identifier
	p.nextSemanticToken()

	// If another identifier is provided, then the previous identifier
	// is the argument label, and this identifier is the parameter name
	if p.current.Is(lexer.TokenIdentifier) {
		argumentLabel = identifier.Identifier
		newIdentifier, err := p.nonReservedIdentifier("for parameter name")
		if err != nil {
			return nil, err
		}

		identifier = newIdentifier

		// skip the identifier, now known to be the argument name
		p.nextSemanticToken()
	}

	colonToken := p.current
	if !colonToken.Is(lexer.TokenColon) {
		return nil, &MissingColonAfterParameterNameError{
			GotToken: colonToken,
		}
	}

	// Skip the colon
	p.nextSemanticToken()

	typeAnnotation, err := parseTypeAnnotation(p)

	if err != nil {
		return nil, err
	}

	p.skipSpace()

	var defaultArgument ast.Expression

	if expectDefaultArgument {
		if !p.current.Is(lexer.TokenEqual) {
			return nil, &MissingDefaultArgumentError{
				GotToken: p.current,
			}
		}

		// Skip the =
		p.nextSemanticToken()

		defaultArgument, err = parseExpression(p, lowestBindingPower)
		if err != nil {
			return nil, err
		}

	} else if p.current.Is(lexer.TokenEqual) {
		return nil, &UnexpectedDefaultArgumentError{
			Pos: p.current.StartPos,
		}
	}

	return ast.NewParameter(
		p.memoryGauge,
		argumentLabel,
		identifier,
		typeAnnotation,
		defaultArgument,
		startToken.StartPos,
		ast.Comments{
			Leading: append(
				startToken.Comments.All(),
				colonToken.Comments.All()...,
			),
		},
	), nil
}

func parseTypeParameterList(p *parser) (*ast.TypeParameterList, error) {
	var typeParameters []*ast.TypeParameter

	p.skipSpace()

	if !p.current.Is(lexer.TokenLess) {
		return nil, nil
	}

	startPos := p.current.StartPos
	// Skip the opening paren
	p.next()

	var endPos ast.Position

	expectTypeParameter := true

	var atEnd bool
	progress := p.newProgress()

	for !atEnd && p.checkProgress(&progress) {

		p.skipSpace()

		switch p.current.Type {
		case lexer.TokenIdentifier:
			if !expectTypeParameter {
				p.report(&MissingCommaInTypeParameterListError{
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
				return nil, &UnexpectedTokenInTypeParameterListError{
					GotToken: p.current,
				}
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
			return nil, &MissingClosingGreaterInTypeParameterListError{
				Pos: p.current.StartPos,
			}
		default:
			if expectTypeParameter {
				return nil, &UnexpectedTokenInTypeParameterListError{
					GotToken: p.current,
				}
			} else {
				return nil, &ExpectedCommaOrEndOfTypeParameterListError{
					GotToken: p.current,
				}
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
	p.skipSpace()

	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, &InvalidTypeParameterNameError{
			GotToken: p.current,
		}
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
	accessToken *lexer.Token,
	purity ast.FunctionPurity,
	purityToken *lexer.Token,
	staticToken *lexer.Token,
	nativeToken *lexer.Token,
) (*ast.FunctionDeclaration, error) {
	startToken := lexer.EarliestToken(p.current, accessToken, purityToken, staticToken, nativeToken)

	// Skip the `fun` keyword
	p.nextSemanticToken()

	identifier, err := p.nonReservedIdentifier("after start of function declaration")

	if err != nil {
		return nil, err
	}

	// Skip the identifier
	p.next()

	var typeParameterList *ast.TypeParameterList

	if p.config.TypeParametersEnabled {
		var err error
		typeParameterList, err = parseTypeParameterList(p)
		if err != nil {
			return nil, err
		}
	}

	parameterList, returnTypeAnnotation, functionBlock, err :=
		parseFunctionParameterListAndRest(p, functionBlockIsOptional)

	if err != nil {
		return nil, err
	}

	return ast.NewFunctionDeclaration(
		p.memoryGauge,
		access,
		purity,
		staticToken != nil,
		nativeToken != nil,
		identifier,
		typeParameterList,
		parameterList,
		returnTypeAnnotation,
		functionBlock,
		startToken.StartPos,
		ast.Comments{
			Leading: startToken.Comments.Leading,
		},
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

	parameterList, err = parseParameterList(p, false)
	if err != nil {
		return
	}

	// Optional return type

	current := p.current
	cursor := p.tokens.Cursor()

	p.skipSpace()

	if p.current.Is(lexer.TokenColon) {
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
		p.skipSpace()
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
