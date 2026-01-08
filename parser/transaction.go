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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/parser/lexer"
)

// parseTransactionDeclaration parses a transaction declaration.
//
//	transactionDeclaration : 'transaction'
//	    parameterList?
//	    '{'
//	    fields
//	    prepare?
//	    preConditions?
//	    ( execute
//	    | execute postConditions
//	    | postConditions
//	    | postConditions execute
//	    | /* no execute or postConditions */
//	    )
//	    '}'
func parseTransactionDeclaration(p *parser) (*ast.TransactionDeclaration, error) {

	startToken := p.current

	// Skip the `transaction` keyword
	p.nextSemanticToken()

	// Parameter list (optional)

	var parameterList *ast.ParameterList
	var err error

	if p.current.Is(lexer.TokenParenOpen) {
		parameterList, err = parseParameterList(p, false)
		if err != nil {
			return nil, err
		}
	}

	p.skipSpace()

	parseDeclarationOpeningBrace(p, common.DeclarationKindTransaction)

	// Fields

	fields, err := parseTransactionFields(p)
	if err != nil {
		return nil, err
	}

	// Prepare (optional) or execute (optional)

	var prepare *ast.SpecialFunctionDeclaration
	var execute *ast.SpecialFunctionDeclaration

	p.skipSpace()

	if p.current.Is(lexer.TokenIdentifier) {

		keyword := p.currentTokenSource()

		switch string(keyword) {
		case KeywordPrepare:
			identifierToken := p.current
			// Skip the `prepare` keyword
			p.next()
			prepare, err = parseSpecialFunctionDeclaration(
				p,
				false,
				ast.AccessNotSpecified,
				nil,
				ast.FunctionPurityUnspecified,
				nil,
				nil,
				nil,
				identifierToken,
			)
			if err != nil {
				return nil, err
			}

		case KeywordExecute:
			execute, err = parseTransactionExecute(p)
			if err != nil {
				return nil, err
			}

		default:
			return nil, &ExpectedPrepareOrExecuteError{
				GotIdentifier: string(keyword),
				Pos:           p.current.StartPos,
			}
		}
	}

	// Pre-conditions (optional)

	var preConditions *ast.Conditions

	if execute == nil {
		p.skipSpace()

		if p.isToken(p.current, lexer.TokenIdentifier, KeywordPre) {
			preStartPos := p.current.StartPos

			// Skip the `pre` keyword
			p.nextSemanticToken()

			preConditions, err = parseConditions(p, preStartPos)
			if err != nil {
				return nil, err
			}
		}
	}

	// Execute / post-conditions (both optional, in any order)

	var postConditions *ast.Conditions

	var endPos ast.Position

	sawPost := false

	var atEnd bool
	progress := p.newProgress()

	for !atEnd && p.checkProgress(&progress) {

		p.skipSpace()

		switch p.current.Type {
		case lexer.TokenIdentifier:
			// Not possible to encounter a second prepare block at this point
			keyword := p.currentTokenSource()
			switch string(keyword) {
			case KeywordExecute:
				if execute != nil {
					return nil, &DuplicateExecuteBlockError{
						Pos: p.current.StartPos,
					}
				}

				execute, err = parseTransactionExecute(p)
				if err != nil {
					return nil, err
				}

			case KeywordPost:
				if sawPost {
					return nil, &DuplicatePostConditionsError{
						Pos: p.current.StartPos,
					}
				}
				postStartPos := p.current.StartPos
				// Skip the `post` keyword
				p.next()
				postConditions, err = parseConditions(p, postStartPos)
				if err != nil {
					return nil, err
				}
				sawPost = true

			default:
				return nil, &ExpectedExecuteOrPostError{
					GotIdentifier: string(keyword),
					Pos:           p.current.StartPos,
				}
			}

		case lexer.TokenBraceClose:
			endPos = p.current.EndPos
			// Skip the closing brace
			p.next()
			atEnd = true

		default:
			return nil, &UnexpectedTokenAtEndError{
				Token: p.current,
			}
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
		ast.NewRange(
			p.memoryGauge,
			startToken.StartPos,
			endPos,
		),
		ast.Comments{
			Leading: startToken.Comments.Leading,
		},
	), nil
}

func parseTransactionFields(p *parser) (fields []*ast.FieldDeclaration, err error) {
	progress := p.newProgress()

	for p.checkProgress(&progress) {

		p.skipSpaceWithOptions(skipSpaceOptions{
			skipNewlines: true,
		})

		switch p.current.Type {
		case lexer.TokenSemicolon:
			// Skip the semicolon
			p.next()
			continue

		case lexer.TokenBraceClose, lexer.TokenEOF:
			return

		case lexer.TokenIdentifier:
			switch string(p.currentTokenSource()) {
			case KeywordLet, KeywordVar:
				field, err := parseFieldWithVariableKind(
					p,
					ast.AccessNotSpecified,
					nil,
					nil,
					nil,
				)
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

	panic(errors.NewUnreachableError())
}

func parseTransactionExecute(p *parser) (*ast.SpecialFunctionDeclaration, error) {
	identifierToken := p.current
	identifier := p.tokenToIdentifier(identifierToken)

	// Skip the `execute` keyword
	p.nextSemanticToken()

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
			ast.FunctionPurityUnspecified,
			false,
			false,
			identifier,
			nil,
			nil,
			nil,
			ast.NewFunctionBlock(
				p.memoryGauge,
				block,
				nil,
				nil,
			),
			identifier.Pos,
			ast.Comments{
				Leading: identifierToken.Comments.Leading,
			},
		),
	), nil
}
