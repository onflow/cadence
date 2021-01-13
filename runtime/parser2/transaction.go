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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

// parseTransactionDeclaration parses a transaction declaration.
//
//     transactionDeclaration : 'transaction'
//         parameterList?
//         '{'
//         fields
//         prepare?
//         preConditions?
//         ( execute
//         | execute postConditions
//         | postConditions
//         | postConditions execute
//         | /* no execute or postConditions */
//         )
//         '}'
//
func parseTransactionDeclaration(p *parser, docString string) *ast.TransactionDeclaration {

	startPos := p.current.StartPos

	// Skip the `transaction` keyword
	p.next()
	p.skipSpaceAndComments(true)

	// Parameter list (optional)

	var parameterList *ast.ParameterList
	if p.current.Is(lexer.TokenParenOpen) {
		parameterList = parseParameterList(p)
	}

	p.skipSpaceAndComments(true)
	p.mustOne(lexer.TokenBraceOpen)

	// Fields

	fields := parseTransactionFields(p)

	// Prepare (optional) or execute (optional)

	var prepare *ast.SpecialFunctionDeclaration
	var execute *ast.SpecialFunctionDeclaration

	p.skipSpaceAndComments(true)
	if p.current.Is(lexer.TokenIdentifier) {

		switch p.current.Value {
		case keywordPrepare:
			identifier := tokenToIdentifier(p.current)
			// Skip the `prepare` keyword
			p.next()
			prepare = parseSpecialFunctionDeclaration(p, false, ast.AccessNotSpecified, nil, identifier)

		case keywordExecute:
			execute = parseTransactionExecute(p)

		default:
			panic(fmt.Errorf(
				"unexpected identifier, expected keyword %q or %q, got %q",
				keywordPrepare,
				keywordExecute,
				p.current.Value,
			))
		}
	}

	// Pre-conditions (optional)

	var preConditions *ast.Conditions

	if execute == nil {
		p.skipSpaceAndComments(true)
		if p.current.IsString(lexer.TokenIdentifier, keywordPre) {
			// Skip the `pre` keyword
			p.next()
			conditions := parseConditions(p, ast.ConditionKindPre)
			preConditions = &conditions
		}
	}

	// Execute / post-conditions (both optional, in any order)

	var postConditions *ast.Conditions

	var endPos ast.Position

	sawPost := false
	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenIdentifier:
			switch p.current.Value {
			case keywordExecute:
				if execute != nil {
					panic(fmt.Errorf("unexpected second %q block", keywordExecute))
				}

				execute = parseTransactionExecute(p)

			case keywordPost:
				if sawPost {
					panic(fmt.Errorf("unexpected second post-conditions"))
				}
				// Skip the `post` keyword
				p.next()
				conditions := parseConditions(p, ast.ConditionKindPost)
				postConditions = &conditions
				sawPost = true

			default:
				panic(fmt.Errorf(
					"unexpected identifier, expected keyword %q or %q, got %q",
					keywordExecute,
					keywordPost,
					p.current.Value,
				))
			}

		case lexer.TokenBraceClose:
			endPos = p.current.EndPos
			// Skip the closing brace
			p.next()
			atEnd = true

		default:
			panic(fmt.Errorf("unexpected token: %s", p.current.Type))
		}
	}

	return &ast.TransactionDeclaration{
		ParameterList:  parameterList,
		Fields:         fields,
		Prepare:        prepare,
		PreConditions:  preConditions,
		PostConditions: postConditions,
		Execute:        execute,
		DocString:      docString,
		Range: ast.Range{
			StartPos: startPos,
			EndPos:   endPos,
		},
	}
}

func parseTransactionFields(p *parser) (fields []*ast.FieldDeclaration) {
	for {
		_, docString := p.parseTrivia(triviaOptions{
			skipNewlines:    true,
			parseDocStrings: true,
		})

		switch p.current.Type {
		case lexer.TokenSemicolon:
			// Skip the semicolon
			p.next()
			continue

		case lexer.TokenBraceClose, lexer.TokenEOF:
			return

		case lexer.TokenIdentifier:
			switch p.current.Value {
			case keywordLet, keywordVar:
				field := parseFieldWithVariableKind(p, ast.AccessNotSpecified, nil, docString)

				fields = append(fields, field)
				continue

			default:
				return
			}

		default:
			return
		}
	}
}

func parseTransactionExecute(p *parser) *ast.SpecialFunctionDeclaration {
	identifier := tokenToIdentifier(p.current)

	// Skip the `execute` keyword
	p.next()
	p.skipSpaceAndComments(true)

	block := parseBlock(p)

	return &ast.SpecialFunctionDeclaration{
		Kind: common.DeclarationKindExecute,
		FunctionDeclaration: &ast.FunctionDeclaration{
			Access:        ast.AccessNotSpecified,
			Identifier:    identifier,
			ParameterList: &ast.ParameterList{},
			FunctionBlock: &ast.FunctionBlock{
				Block: block,
			},
			StartPos: identifier.Pos,
		},
	}
}
