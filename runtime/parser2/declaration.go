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
	"github.com/onflow/cadence/runtime/errors"
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

	access := ast.AccessNotSpecified
	var accessPos *ast.Position

	for {
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenIdentifier:
			switch p.current.Value {
			case keywordLet, keywordVar:
				return parseVariableDeclaration(p, access, accessPos)

			case keywordFun:
				return parseFunctionDeclaration(p, access, accessPos)

			case keywordPriv, keywordPub, keywordAccess:
				if access != ast.AccessNotSpecified {
					panic(fmt.Errorf("unexpected access modifier"))
				}
				pos := p.current.StartPos
				accessPos = &pos
				access = parseAccess(p)
				continue

			}
		}

		return nil
	}
}

// parseAccess parses an access modifier
//
//    access
//        : 'priv'
//        | 'pub' ( '(' 'set' ')' )?
//        | 'access' '(' ( 'self' | 'contract' | 'account' | 'all' ) ')'
//        ;
//
func parseAccess(p *parser) ast.Access {

	switch p.current.Value {
	case keywordPriv:
		p.next()
		return ast.AccessPrivate

	case keywordPub:
		p.next()
		p.skipSpaceAndComments(true)
		if !p.current.Is(lexer.TokenParenOpen) {
			return ast.AccessPublic
		}

		p.next()
		p.skipSpaceAndComments(true)

		if !p.current.Is(lexer.TokenIdentifier) {
			panic(fmt.Errorf(
				"expected keyword %q, got %s",
				keywordSet,
				p.current.Type,
			))
		}
		if p.current.Value != keywordSet {
			panic(fmt.Errorf(
				"expected keyword %q, got %q",
				keywordSet,
				p.current.Value,
			))
		}

		p.next()
		p.skipSpaceAndComments(true)

		p.mustOne(lexer.TokenParenClose)

		return ast.AccessPublicSettable

	case keywordAccess:
		p.next()
		p.skipSpaceAndComments(true)

		p.mustOne(lexer.TokenParenOpen)

		p.skipSpaceAndComments(true)

		if !p.current.Is(lexer.TokenIdentifier) {
			panic(fmt.Errorf(
				"expected keyword %q, %q, %q, or %q, got %s",
				keywordAll,
				keywordAccount,
				keywordContract,
				keywordSelf,
				p.current.Type,
			))
		}

		var access ast.Access

		switch p.current.Value {
		case keywordAll:
			access = ast.AccessPublic

		case keywordAccount:
			access = ast.AccessAccount

		case keywordContract:
			access = ast.AccessContract

		case keywordSelf:
			access = ast.AccessPrivate

		default:
			panic(fmt.Errorf(
				"expected keyword %q, %q, %q, or %q, got %q",
				keywordAll,
				keywordAccount,
				keywordContract,
				keywordSelf,
				p.current.Value,
			))
		}

		p.next()
		p.skipSpaceAndComments(true)

		p.mustOne(lexer.TokenParenClose)

		return access

	default:
		panic(errors.NewUnreachableError())
	}
}

func parseVariableDeclaration(p *parser, access ast.Access, accessPos *ast.Position) *ast.VariableDeclaration {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	isLet := p.current.Value == keywordLet

	// skip `let` or `var` keyword
	p.next()

	p.skipSpaceAndComments(true)
	if !p.current.Is(lexer.TokenIdentifier) {
		panic(fmt.Errorf(
			"expected identifier after start of variable declaration, got %s",
			p.current.Type,
		))
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

	// TODO: second transfer and value

	return &ast.VariableDeclaration{
		Access:         access,
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

	pos := p.current.StartPos

	p.next()

	return &ast.Transfer{
		Operation: operation,
		Pos:       pos,
	}
}
