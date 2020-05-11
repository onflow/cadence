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
		case keywordIf:
			return parseIfStatement(p)
		case keywordWhile:
			return parseWhileStatement(p)
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

func parseIfStatement(p *parser) *ast.IfStatement {

	var ifStatements []*ast.IfStatement

	for {
		startPos := p.current.Range.StartPos
		p.next()

		expression := parseExpression(p, lowestBindingPower)
		if expression == nil {
			panic(fmt.Errorf("expected test expression"))
		}

		thenBlock := parseBlock(p)
		if thenBlock == nil {
			panic(fmt.Errorf("expected block for then branch"))
		}

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
				if elseBlock == nil {
					panic(fmt.Errorf("expected block for else branch"))
				}
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

	startPos := p.current.Range.StartPos
	p.next()

	expression := parseExpression(p, lowestBindingPower)
	if expression == nil {
		panic(fmt.Errorf("expected test expression"))
	}

	block := parseBlock(p)
	if block == nil {
		panic(fmt.Errorf("expected block for then branch"))
	}

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
			StartPos: startToken.Range.StartPos,
			EndPos:   endToken.Range.EndPos,
		},
	}
}
