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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

func parseStatements(p *parser) (statements []ast.Statement) {
	for {
		p.skipSpaceAndComments(true)
		switch p.current.Type {
		case lexer.TokenSemicolon:
			p.next()
			continue
		case lexer.TokenEOF:
			return
		default:
			statement := parseStatement(p)
			if statement == nil {
				return
			}

			statements = append(statements, statement)
		}
	}
}

func parseStatement(p *parser) ast.Statement {
	p.skipSpaceAndComments(true)
	switch p.current.Type {
	case lexer.TokenIdentifier:
		switch p.current.Value {
		case "return":
			return parseReturnStatement(p)
		}
	}

	expression := parseExpression(p, lowestBindingPower)
	if expression == nil {
		return nil
	}
	return &ast.ExpressionStatement{
		Expression: expression,
	}
}

func parseReturnStatement(p *parser) *ast.ReturnStatement {
	tokenRange := p.current.Range
	endPosition := tokenRange.EndPos
	p.next()
	sawNewLine := p.skipSpaceAndComments(false)

	var expression ast.Expression
	if !sawNewLine {
		expression = parseExpression(p, lowestBindingPower)
		if expression != nil {
			endPosition = expression.EndPosition()
		}
	}

	return &ast.ReturnStatement{
		Expression: expression,
		Range: ast.Range{
			StartPos: tokenRange.StartPos,
			EndPos:   endPosition,
		},
	}
}
