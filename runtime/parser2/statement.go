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
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

func parseStatements(p *parser, isEndToken func(token lexer.Token) bool) (statements []ast.Statement) {
	sawSemicolon := false
	for {
		p.skipSpaceAndComments(true)
		switch p.current.Type {
		case lexer.TokenSemicolon:
			sawSemicolon = true
			p.next()
			continue
		case lexer.TokenEOF:
			return
		default:
			if isEndToken != nil && isEndToken(p.current) {
				return
			}

			statement := parseStatement(p)
			if statement == nil {
				return
			}

			statements = append(statements, statement)

			// Check that the previous statement (if any) followed a semicolon

			if !sawSemicolon {
				statementCount := len(statements)
				if statementCount > 1 {
					previousStatement := statements[statementCount-2]
					previousLine := previousStatement.EndPosition(p.memoryGauge).Line
					currentStartPos := statement.StartPosition()
					if previousLine == currentStartPos.Line {
						p.report(&SyntaxError{
							Message: "statements on the same line must be separated with a semicolon",
							Pos:     currentStartPos,
						})
					}
				}
			}

			sawSemicolon = false
		}
	}
}

func parseStatement(p *parser) ast.Statement {
	p.skipSpaceAndComments(true)

	// It might start with a keyword for a statement

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
		case keywordSwitch:
			return parseSwitchStatement(p)
		case keywordWhile:
			return parseWhileStatement(p)
		case keywordFor:
			return parseForStatement(p)
		case keywordEmit:
			return parseEmitStatement(p)
		case keywordFun:
			// The `fun` keyword is ambiguous: it either introduces a function expression
			// or a function declaration, depending on if an identifier follows, or not.
			return parseFunctionDeclarationOrFunctionExpressionStatement(p)
		}
	}

	// If it is not a keyword for a statement,
	// it might start with a keyword for a declaration

	declaration := parseDeclaration(p, "")
	if statement, ok := declaration.(ast.Statement); ok {
		return statement
	}

	// If it is not a statement or declaration,
	// it must be an expression

	expression := parseExpression(p, lowestBindingPower)

	// If the expression is followed by a transfer,
	// it is actually the target of an assignment or swap statement

	p.skipSpaceAndComments(true)
	switch p.current.Type {
	case lexer.TokenEqual, lexer.TokenLeftArrow, lexer.TokenLeftArrowExclamation:
		transfer := parseTransfer(p)

		value := parseExpression(p, lowestBindingPower)

		return ast.NewAssignmentStatement(p.memoryGauge, expression, transfer, value)

	case lexer.TokenSwap:
		p.next()

		right := parseExpression(p, lowestBindingPower)

		return ast.NewSwapStatement(p.memoryGauge, expression, right)

	default:
		return ast.NewExpressionStatement(p.memoryGauge, expression)
	}
}

func parseFunctionDeclarationOrFunctionExpressionStatement(p *parser) ast.Statement {

	startPos := p.current.StartPos

	// Skip the `fun` keyword
	p.next()

	p.skipSpaceAndComments(true)

	if p.current.Is(lexer.TokenIdentifier) {
		identifier := p.tokenToIdentifier(p.current)

		p.next()

		parameterList, returnTypeAnnotation, functionBlock :=
			parseFunctionParameterListAndRest(p, false)

		return ast.NewFunctionDeclaration(
			p.memoryGauge,
			ast.AccessNotSpecified,
			identifier,
			parameterList,
			returnTypeAnnotation,
			functionBlock,
			startPos,
			"",
		)
	} else {
		parameterList, returnTypeAnnotation, functionBlock :=
			parseFunctionParameterListAndRest(p, false)

		return ast.NewExpressionStatement(
			p.memoryGauge,
			ast.NewFunctionExpression(
				p.memoryGauge,
				parameterList,
				returnTypeAnnotation,
				functionBlock,
				startPos,
			),
		)
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
			endPosition = expression.EndPosition(p.memoryGauge)
		}
	}

	return ast.NewReturnStatement(
		p.memoryGauge,
		expression,
		ast.NewRange(
			p.memoryGauge,
			tokenRange.StartPos,
			endPosition,
		),
	)
}

func parseBreakStatement(p *parser) *ast.BreakStatement {
	tokenRange := p.current.Range
	p.next()

	return ast.NewBreakStatement(p.memoryGauge, tokenRange)
}

func parseContinueStatement(p *parser) *ast.ContinueStatement {
	tokenRange := p.current.Range
	p.next()

	return ast.NewContinueStatement(p.memoryGauge, tokenRange)
}

func parseIfStatement(p *parser) *ast.IfStatement {

	var ifStatements []*ast.IfStatement

	for {
		startPos := p.current.StartPos
		p.next()

		p.skipSpaceAndComments(true)

		var variableDeclaration *ast.VariableDeclaration

		if p.current.Type == lexer.TokenIdentifier {
			switch p.current.Value {
			case keywordLet, keywordVar:
				variableDeclaration =
					parseVariableDeclaration(p, ast.AccessNotSpecified, nil, "")
			}
		}

		var expression ast.Expression

		if variableDeclaration == nil {
			expression = parseExpression(p, lowestBindingPower)
		}

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

		var test ast.IfStatementTest
		switch {
		case variableDeclaration != nil:
			test = variableDeclaration
		case expression != nil:
			test = expression
		default:
			panic(errors.NewUnreachableError())
		}

		ifStatement := ast.NewIfStatement(
			p.memoryGauge,
			test,
			thenBlock,
			elseBlock,
			startPos,
		)

		if variableDeclaration != nil {
			variableDeclaration.ParentIfStatement = ifStatement
		}

		ifStatements = append(ifStatements, ifStatement)

		if !parseNested {
			break
		}
	}

	length := len(ifStatements)

	result := ifStatements[length-1]

	for i := length - 2; i >= 0; i-- {
		outer := ifStatements[i]
		outer.Else = ast.NewBlock(
			p.memoryGauge,
			[]ast.Statement{result},
			ast.NewRangeFromPositioned(p.memoryGauge, result),
		)
		result = outer
	}

	return result
}

func parseWhileStatement(p *parser) *ast.WhileStatement {

	startPos := p.current.StartPos
	p.next()

	expression := parseExpression(p, lowestBindingPower)

	block := parseBlock(p)

	return ast.NewWhileStatement(p.memoryGauge, expression, block, startPos)
}

func parseForStatement(p *parser) *ast.ForStatement {

	startPos := p.current.StartPos
	p.next()

	p.skipSpaceAndComments(true)

	if p.current.IsString(lexer.TokenIdentifier, keywordIn) {
		p.report(fmt.Errorf(
			"expected identifier, got keyword %q",
			keywordIn,
		))
		p.next()
	}

	firstValue := p.mustIdentifier()

	p.skipSpaceAndComments(true)

	var index *ast.Identifier
	var identifier ast.Identifier

	if p.current.Is(lexer.TokenComma) {
		p.next()
		p.skipSpaceAndComments(true)
		index = &firstValue
		identifier = p.mustIdentifier()
		p.skipSpaceAndComments(true)
	} else {
		identifier = firstValue
	}

	if !p.current.IsString(lexer.TokenIdentifier, keywordIn) {
		p.report(fmt.Errorf(
			"expected keyword %q, got %s",
			keywordIn,
			p.current.Type,
		))
	}

	p.next()

	expression := parseExpression(p, lowestBindingPower)

	block := parseBlock(p)

	return ast.NewForStatement(
		p.memoryGauge,
		identifier,
		index,
		block,
		expression,
		startPos,
	)
}

func parseBlock(p *parser) *ast.Block {
	startToken := p.mustOne(lexer.TokenBraceOpen)
	statements := parseStatements(p, func(token lexer.Token) bool {
		return token.Type == lexer.TokenBraceClose
	})
	endToken := p.mustOne(lexer.TokenBraceClose)

	return ast.NewBlock(
		p.memoryGauge,
		statements,
		ast.NewRange(
			p.memoryGauge,
			startToken.StartPos,
			endToken.EndPos,
		),
	)
}

func parseFunctionBlock(p *parser) *ast.FunctionBlock {
	p.skipSpaceAndComments(true)

	startToken := p.mustOne(lexer.TokenBraceOpen)

	p.skipSpaceAndComments(true)

	var preConditions *ast.Conditions
	if p.current.IsString(lexer.TokenIdentifier, keywordPre) {
		p.next()
		conditions := parseConditions(p, ast.ConditionKindPre)
		preConditions = &conditions
	}

	p.skipSpaceAndComments(true)

	var postConditions *ast.Conditions
	if p.current.IsString(lexer.TokenIdentifier, keywordPost) {
		p.next()
		conditions := parseConditions(p, ast.ConditionKindPost)
		postConditions = &conditions
	}

	statements := parseStatements(p, func(token lexer.Token) bool {
		return token.Type == lexer.TokenBraceClose
	})

	endToken := p.mustOne(lexer.TokenBraceClose)

	return ast.NewFunctionBlock(
		p.memoryGauge,
		ast.NewBlock(
			p.memoryGauge,
			statements,
			ast.NewRange(
				p.memoryGauge,
				startToken.StartPos,
				endToken.EndPos,
			),
		),
		preConditions,
		postConditions,
	)
}

// parseConditions parses conditions (pre/post)
//
func parseConditions(p *parser, kind ast.ConditionKind) (conditions ast.Conditions) {

	p.skipSpaceAndComments(true)
	p.mustOne(lexer.TokenBraceOpen)

	defer func() {
		p.skipSpaceAndComments(true)
		p.mustOne(lexer.TokenBraceClose)
	}()

	for {
		p.skipSpaceAndComments(true)
		switch p.current.Type {
		case lexer.TokenSemicolon:
			p.next()
			continue

		case lexer.TokenBraceClose, lexer.TokenEOF:
			return

		default:
			condition := parseCondition(p, kind)
			if condition == nil {
				return
			}

			conditions = append(conditions, condition)
		}
	}
}

// parseCondition parses a condition (pre/post)
//
//    condition : expression (':' expression )?
//
func parseCondition(p *parser, kind ast.ConditionKind) *ast.Condition {

	test := parseExpression(p, lowestBindingPower)

	p.skipSpaceAndComments(true)

	var message ast.Expression
	if p.current.Is(lexer.TokenColon) {
		p.next()

		message = parseExpression(p, lowestBindingPower)
	}

	return &ast.Condition{
		Kind:    kind,
		Test:    test,
		Message: message,
	}
}

func parseEmitStatement(p *parser) *ast.EmitStatement {
	startPos := p.current.StartPos
	p.next()

	invocation := parseNominalTypeInvocationRemainder(p)
	return ast.NewEmitStatement(p.memoryGauge, invocation, startPos)
}

func parseSwitchStatement(p *parser) *ast.SwitchStatement {

	startPos := p.current.StartPos

	// Skip the `switch` keyword
	p.next()

	expression := parseExpression(p, lowestBindingPower)

	p.mustOne(lexer.TokenBraceOpen)

	cases := parseSwitchCases(p)

	endToken := p.mustOne(lexer.TokenBraceClose)

	return ast.NewSwitchStatement(
		p.memoryGauge,
		expression,
		cases,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			endToken.EndPos,
		),
	)
}

// parseSwitchCases parses cases of a switch statement.
//
//     switchCases : switchCase*
//
func parseSwitchCases(p *parser) (cases []*ast.SwitchCase) {

	reportUnexpected := func() {
		p.report(fmt.Errorf(
			"unexpected token: got %s, expected %q or %q",
			p.current.Type,
			keywordCase,
			keywordDefault,
		))
		p.next()
	}

	for {
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenIdentifier:

			var switchCase *ast.SwitchCase

			switch p.current.Value {
			case keywordCase:
				switchCase = parseSwitchCase(p, true)

			case keywordDefault:
				switchCase = parseSwitchCase(p, false)

			default:
				reportUnexpected()
				continue
			}

			cases = append(cases, switchCase)

		case lexer.TokenBraceClose, lexer.TokenEOF:
			return

		default:
			reportUnexpected()
		}
	}
}

// parseSwitchCase parses a switch case (hasExpression == true)
// or default case (hasExpression == false)
//
//     switchCase : `case` expression `:` statements
//                | `default` `:` statements
//
func parseSwitchCase(p *parser, hasExpression bool) *ast.SwitchCase {

	startPos := p.current.StartPos

	// Skip the keyword
	p.next()

	var expression ast.Expression
	if hasExpression {
		expression = parseExpression(p, lowestBindingPower)
	} else {
		p.skipSpaceAndComments(true)
	}

	colonPos := p.current.StartPos

	if !p.current.Is(lexer.TokenColon) {
		p.report(fmt.Errorf(
			"expected %s, got %s",
			lexer.TokenColon,
			p.current.Type,
		))
	}

	p.next()

	statements := parseStatements(p, func(token lexer.Token) bool {
		switch token.Type {
		case lexer.TokenBraceClose:
			return true

		case lexer.TokenIdentifier:
			switch p.current.Value {
			case keywordCase, keywordDefault:
				return true
			default:
				return false
			}

		default:
			return false
		}
	})

	endPos := colonPos

	if len(statements) > 0 {
		lastStatementIndex := len(statements) - 1
		endPos = statements[lastStatementIndex].EndPosition(p.memoryGauge)
	}

	return &ast.SwitchCase{
		Expression: expression,
		Statements: statements,
		Range: ast.NewRange(
			p.memoryGauge,
			startPos,
			endPos,
		),
	}
}
