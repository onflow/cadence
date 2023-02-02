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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser/lexer"
)

// parseTransactionDeclaration parses a transaction declaration.
//
//	transactionDeclaration : 'transaction'
//	    parameterList?
//	    '{'
//	    fields
//	    transactionPrepare?
//	    transactionRoleDeclaration*
//	    preConditions?
//	    ( transactionExecute
//	    | transactionExecute postConditions
//	    | postConditions
//	    | postConditions transactionExecute
//	    | /* no transactionExecute or postConditions */
//	    )
//	    '}'
func parseTransactionDeclaration(p *parser, docString string) (*ast.TransactionDeclaration, error) {

	startPos := p.current.StartPos

	// Skip the `transaction` keyword
	p.nextSemanticToken()

	// Parameter list (optional)

	var parameterList *ast.ParameterList
	var err error

	if p.current.Is(lexer.TokenParenOpen) {
		parameterList, err = parseParameterList(p)
		if err != nil {
			return nil, err
		}
	}

	p.skipSpaceAndComments()
	_, err = p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	// Fields

	fields, err := parseTransactionFields(p)
	if err != nil {
		return nil, err
	}

	// Prepare (optional)

	var prepare *ast.SpecialFunctionDeclaration

	p.skipSpaceAndComments()
	if p.isToken(p.current, lexer.TokenIdentifier, keywordPrepare) {
		prepare, err = parseTransactionPrepare(p)
		if err != nil {
			return nil, err
		}
	}

	// Roles (optional)

	atEnd := false

	var roles []*ast.TransactionRoleDeclaration
	for !atEnd {
		_, docString := p.parseTrivia(triviaOptions{
			skipNewlines:    true,
			parseDocStrings: true,
		})

		if p.isToken(p.current, lexer.TokenIdentifier, keywordRole) {
			role, err := parseTransactionRole(p, docString)
			if err != nil {
				return nil, err
			}
			roles = append(roles, role)
		} else {
			atEnd = true
		}
	}

	// Pre-conditions (optional)

	var preConditions *ast.Conditions

	p.skipSpaceAndComments()
	if p.isToken(p.current, lexer.TokenIdentifier, keywordPre) {
		// Skip the `pre` keyword
		p.next()
		conditions, err := parseConditions(p, ast.ConditionKindPre)
		if err != nil {
			return nil, err
		}

		preConditions = &conditions
	}

	// Execute / post-conditions (both optional, in any order)

	var execute *ast.SpecialFunctionDeclaration
	var postConditions *ast.Conditions

	var endPos ast.Position

	sawPost := false
	atEnd = false
	for !atEnd {
		p.skipSpaceAndComments()

		switch p.current.Type {
		case lexer.TokenIdentifier:

			keyword := p.currentTokenSource()
			switch string(keyword) {
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
					keyword,
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
		roles,
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

func parseTransactionPrepare(p *parser) (*ast.SpecialFunctionDeclaration, error) {
	identifier := p.tokenToIdentifier(p.current)
	// Skip the `prepare` keyword
	p.next()

	return parseSpecialFunctionDeclaration(
		p,
		false,
		ast.AccessNotSpecified,
		nil,
		nil,
		nil,
		identifier,
	)
}

func parseTransactionFields(p *parser) (fields []*ast.FieldDeclaration, err error) {
	access := ast.AccessNotSpecified
	var accessPos *ast.Position

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
			switch string(p.currentTokenSource()) {
			case keywordLet, keywordVar:
				field, err := parseFieldWithVariableKind(
					p,
					access,
					accessPos,
					nil,
					nil,
					docString,
				)
				if err != nil {
					return nil, err
				}
				access = ast.AccessNotSpecified
				accessPos = nil

				fields = append(fields, field)
				continue

			case keywordPriv, keywordPub, keywordAccess:
				if access != ast.AccessNotSpecified {
					return nil, p.syntaxError("invalid second access modifier")
				}
				pos := p.current.StartPos
				accessPos = &pos
				var err error
				access, err = parseAccess(p)
				if err != nil {
					return nil, err
				}

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
			"",
		),
	), nil
}

// parseTransactionRole parses a transaction role declaration.
//
//	transactionRoleDeclaration : 'role'
//	    identifier
//	    '{'
//	    fields
//	    transactionPrepare?
//	    '}'
func parseTransactionRole(p *parser, docString string) (*ast.TransactionRoleDeclaration, error) {
	// Skip the `role` keyword
	p.nextSemanticToken()

	// Name
	name, err := p.mustIdentifier()
	if err != nil {
		return nil, err
	}
	p.nextSemanticToken()

	blockStartToken, err := p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	// Fields
	fields, err := parseTransactionFields(p)
	if err != nil {
		return nil, err
	}

	// Prepare (optional)

	var prepare *ast.SpecialFunctionDeclaration

	p.skipSpaceAndComments()
	if p.isToken(p.current, lexer.TokenIdentifier, keywordPrepare) {
		prepare, err = parseTransactionPrepare(p)
		if err != nil {
			return nil, err
		}
	}

	p.skipSpaceAndComments()
	blockEndToken, err := p.mustOne(lexer.TokenBraceClose)
	if err != nil {
		return nil, err
	}

	return ast.NewTransactionRoleDeclaration(
		p.memoryGauge,
		name,
		fields,
		prepare,
		docString,
		ast.Range{
			StartPos: blockStartToken.StartPos,
			EndPos:   blockEndToken.EndPos,
		},
	), nil
}
