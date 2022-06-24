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

package parser2

import (
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
func parseTransactionDeclaration(p *parser, docString string) (*ast.TransactionDeclaration, error) {

	startPos := p.current.StartPos

	// Skip the `transaction` keyword
	p.next()
	p.skipSpaceAndComments(true)

	// Parameter list (optional)

	var parameterList *ast.ParameterList
	if p.current.Is(lexer.TokenParenOpen) {
		parameterList, _ = parseParameterList(p)
	}

	p.skipSpaceAndComments(true)
	_, err := p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	// Fields

	fields, err := parseTransactionFields(p)
	if err != nil {
		return nil, err
	}

	// Prepare (optional) or execute (optional)

	var prepare *ast.SpecialFunctionDeclaration
	var execute *ast.SpecialFunctionDeclaration

	p.skipSpaceAndComments(true)
	if p.current.Is(lexer.TokenIdentifier) {

		switch p.current.Value {
		case keywordPrepare:
			identifier := p.tokenToIdentifier(p.current)
			// Skip the `prepare` keyword
			p.next()
			prepare, err = parseSpecialFunctionDeclaration(p, false, ast.AccessNotSpecified, nil, identifier)
			if err != nil {
				return nil, err
			}

		case keywordExecute:
			execute, err = parseTransactionExecute(p)
			if err != nil {
				return nil, err
			}

		default:
			return nil, p.syntaxError(
				"unexpected identifier, expected keyword %q or %q, got %q",
				keywordPrepare,
				keywordExecute,
				p.current.Value,
			)
		}
	}

	// Pre-conditions (optional)

	var preConditions *ast.Conditions

	if execute == nil {
		p.skipSpaceAndComments(true)
		if p.current.IsString(lexer.TokenIdentifier, keywordPre) {
			// Skip the `pre` keyword
			p.next()
			conditions, err := parseConditions(p, ast.ConditionKindPre)
			if err != nil {
				return nil, err
			}

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
					return nil, p.syntaxError("unexpected second %q block", keywordExecute)
				}

				execute, err = parseTransactionExecute(p)
				if err != nil {
					return nil, err
				}

			case keywordPost:
				if sawPost {
					return nil, p.syntaxError("unexpected second post-conditions")
				}
				// Skip the `post` keyword
				p.next()
				conditions, err := parseConditions(p, ast.ConditionKindPost)
				if err != nil {
					return nil, err
				}

				postConditions = &conditions
				sawPost = true

			default:
				return nil, p.syntaxError(
					"unexpected identifier, expected keyword %q or %q, got %q",
					keywordExecute,
					keywordPost,
					p.current.Value,
				)
			}

		case lexer.TokenBraceClose:
			endPos = p.current.EndPos
			// Skip the closing brace
			p.next()
			atEnd = true

		default:
			return nil, p.syntaxError("unexpected token: %s", p.current.Type)
		}
	}

	return ast.NewTransactionDeclaration(
		p.memoryGauge,
		parameterList,
		fields,
		prepare,
		preConditions,
		postConditions,
		execute,
		docString,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			endPos,
		),
	), nil
}

func parseTransactionFields(p *parser) (fields []*ast.FieldDeclaration, err error) {
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
				field, err := parseFieldWithVariableKind(p, ast.AccessNotSpecified, nil, docString)
				if err != nil {
					return nil, err
				}

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

func parseTransactionExecute(p *parser) (*ast.SpecialFunctionDeclaration, error) {
	identifier := p.tokenToIdentifier(p.current)

	// Skip the `execute` keyword
	p.next()
	p.skipSpaceAndComments(true)

	block, err := parseBlock(p)
	if err != nil {
		return nil, err
	}

	return ast.NewSpecialFunctionDeclaration(
		p.memoryGauge,
		common.DeclarationKindExecute,
		ast.NewFunctionDeclaration(
			p.memoryGauge,
			ast.AccessNotSpecified,
			identifier,
			nil,
			nil,
			ast.NewFunctionBlock(
				p.memoryGauge,
				block,
				nil,
				nil,
			),
			identifier.Pos,
			"",
		),
	), nil
}
