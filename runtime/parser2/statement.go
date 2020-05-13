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

func parseStatements(p *parser, endTokenType lexer.TokenType) (statements []ast.Statement) {
	for {
		p.skipSpaceAndComments(true)
		switch p.current.Type {
		case lexer.TokenSemicolon:
			p.next()
			continue
		case endTokenType, lexer.TokenEOF:
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
		case keywordReturn:
			return parseReturnStatement(p)
		case keywordBreak:
			return parseBreakStatement(p)
		case keywordContinue:
			return parseContinueStatement(p)
		case keywordIf:
			return parseIfStatement(p)
		case keywordWhile:
			return parseWhileStatement(p)
		}
	}

	declaration := parseDeclaration(p)
	// TODO: allow more
	switch declaration := declaration.(type) {
	case *ast.VariableDeclaration:
		return declaration
	case *ast.FunctionDeclaration:
		return declaration
	}

	expression := parseExpression(p, lowestBindingPower)
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
	switch p.current.Type {
	case lexer.TokenEOF, lexer.TokenSemicolon, lexer.TokenBraceClose:
		break
	default:
		if !sawNewLine {
			expression = parseExpression(p, lowestBindingPower)
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

func parseBreakStatement(p *parser) *ast.BreakStatement {
	tokenRange := p.current.Range
	p.next()

	return &ast.BreakStatement{
		Range: tokenRange,
	}
}

func parseContinueStatement(p *parser) *ast.ContinueStatement {
	tokenRange := p.current.Range
	p.next()

	return &ast.ContinueStatement{
		Range: tokenRange,
	}
}

func parseIfStatement(p *parser) *ast.IfStatement {

	var ifStatements []*ast.IfStatement

	for {
		startPos := p.current.StartPos
		p.next()

		expression := parseExpression(p, lowestBindingPower)

		thenBlock := parseBlock(p)

		var elseBlock *ast.Block

		parseNested := false

		p.skipSpaceAndComments(true)
		if p.current.IsString(lexer.TokenIdentifier, keywordElse) {
			p.next()

			p.skipSpaceAndComments(true)
			if p.current.IsString(lexer.TokenIdentifier, keywordIf) {
				parseNested = true
			} else {
				elseBlock = parseBlock(p)
			}
		}

		ifStatements = append(ifStatements,
			&ast.IfStatement{
				Test:     expression,
				Then:     thenBlock,
				Else:     elseBlock,
				StartPos: startPos,
			},
		)

		if !parseNested {
			break
		}
	}

	length := len(ifStatements)

	result := ifStatements[length-1]

	for i := length - 2; i >= 0; i-- {
		outer := ifStatements[i]
		outer.Else = &ast.Block{
			Statements: []ast.Statement{result},
			Range:      ast.NewRangeFromPositioned(result),
		}
		result = outer
	}

	return result
}

func parseWhileStatement(p *parser) *ast.WhileStatement {

	startPos := p.current.StartPos
	p.next()

	expression := parseExpression(p, lowestBindingPower)

	block := parseBlock(p)

	return &ast.WhileStatement{
		Test:     expression,
		Block:    block,
		StartPos: startPos,
	}
}

func parseBlock(p *parser) *ast.Block {
	startToken := p.mustOne(lexer.TokenBraceOpen)
	statements := parseStatements(p, lexer.TokenBraceClose)
	endToken := p.mustOne(lexer.TokenBraceClose)

	return &ast.Block{
		Statements: statements,
		Range: ast.Range{
			StartPos: startToken.StartPos,
			EndPos:   endToken.EndPos,
		},
	}
}
