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

package parser

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser/lexer"
)

func parseStatements(p *parser, isEndToken func(token lexer.Token) bool) (statements []ast.Statement, err error) {
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

			var statement ast.Statement
			statement, err = parseStatement(p)
			if err != nil || statement == nil {
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
						p.report(NewSyntaxError(
							currentStartPos,
							"statements on the same line must be separated with a semicolon",
						))
					}
				}
			}

			sawSemicolon = false
		}
	}
}

func parseStatement(p *parser) (ast.Statement, error) {
	p.skipSpaceAndComments(true)

	// It might start with a keyword for a statement

	switch p.current.Type {
	case lexer.TokenIdentifier:
		switch p.current.Value {
		case keywordReturn:
			return parseReturnStatement(p)
		case keywordBreak:
			return parseBreakStatement(p), nil
		case keywordContinue:
			return parseContinueStatement(p), nil
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
		case keywordView:
			purityPos := p.current.StartPos
			p.next()
			p.skipSpaceAndComments(true)
			if p.current.Value != keywordFun {
				return nil, p.syntaxError("expected fun keyword, but got %s", p.current.Value)
			}
			return parseFunctionDeclarationOrFunctionExpressionStatement(p, ast.FunctionPurityView, &purityPos)
		case keywordFun:
			// The `fun` keyword is ambiguous: it either introduces a function expression
			// or a function declaration, depending on if an identifier follows, or not.
			return parseFunctionDeclarationOrFunctionExpressionStatement(p, ast.FunctionPurityUnspecified, nil)
		}
	}

	// If it is not a keyword for a statement,
	// it might start with a keyword for a declaration

	declaration, err := parseDeclaration(p, "")
	if err != nil {
		return nil, err
	}

	if statement, ok := declaration.(ast.Statement); ok {
		return statement, nil
	}

	// If it is not a statement or declaration,
	// it must be an expression

	expression, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	// If the expression is followed by a transfer,
	// it is actually the target of an assignment or swap statement

	p.skipSpaceAndComments(true)
	switch p.current.Type {
	case lexer.TokenEqual, lexer.TokenLeftArrow, lexer.TokenLeftArrowExclamation:
		transfer := parseTransfer(p)

		value, err := parseExpression(p, lowestBindingPower)
		if err != nil {
			return nil, err
		}

		return ast.NewAssignmentStatement(p.memoryGauge, expression, transfer, value), nil

	case lexer.TokenSwap:
		p.next()

		right, err := parseExpression(p, lowestBindingPower)
		if err != nil {
			return nil, err
		}

		return ast.NewSwapStatement(p.memoryGauge, expression, right), nil

	default:
		return ast.NewExpressionStatement(p.memoryGauge, expression), nil
	}
}

func parseFunctionDeclarationOrFunctionExpressionStatement(
    p *parser,
    purity ast.FunctionPurity,
    purityPos *ast.Position,
) (ast.Statement, error) {

	startPos := *ast.EarlierPosition(&p.current.StartPos, purityPos)

	// Skip the `fun` keyword
	p.next()

	p.skipSpaceAndComments(true)

	if p.current.Is(lexer.TokenIdentifier) {
		identifier := p.tokenToIdentifier(p.current)

		p.next()

		parameterList, returnTypeAnnotation, functionBlock, err :=
			parseFunctionParameterListAndRest(p, false)

		if err != nil {
			return nil, err
		}

		return ast.NewFunctionDeclaration(
			p.memoryGauge,
			ast.AccessNotSpecified,
			purity,
			identifier,
			parameterList,
			returnTypeAnnotation,
			functionBlock,
			startPos,
			"",
		), nil
	} else {
		parameterList, returnTypeAnnotation, functionBlock, err :=
			parseFunctionParameterListAndRest(p, false)
		if err != nil {
			return nil, err
		}

		return ast.NewExpressionStatement(
			p.memoryGauge,
			ast.NewFunctionExpression(
				p.memoryGauge,
				purity,
				parameterList,
				returnTypeAnnotation,
				functionBlock,
				startPos,
			),
		), nil
	}
}

func parseReturnStatement(p *parser) (*ast.ReturnStatement, error) {
	tokenRange := p.current.Range
	endPosition := tokenRange.EndPos
	p.next()

	sawNewLine := p.skipSpaceAndComments(false)

	var expression ast.Expression
	var err error
	switch p.current.Type {
	case lexer.TokenEOF, lexer.TokenSemicolon, lexer.TokenBraceClose:
		break
	default:
		if !sawNewLine {
			expression, err = parseExpression(p, lowestBindingPower)
			if err != nil {
				return nil, err
			}

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
	), nil
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

func parseIfStatement(p *parser) (*ast.IfStatement, error) {

	var ifStatements []*ast.IfStatement

	for {
		startPos := p.current.StartPos
		p.next()

		p.skipSpaceAndComments(true)

		var variableDeclaration *ast.VariableDeclaration
		var err error

		if p.current.Type == lexer.TokenIdentifier {
			switch p.current.Value {
			case keywordLet, keywordVar:
				variableDeclaration, err =
					parseVariableDeclaration(p, ast.AccessNotSpecified, nil, "")
				if err != nil {
					return nil, err
				}
			}
		}

		var expression ast.Expression

		if variableDeclaration == nil {
			expression, err = parseExpression(p, lowestBindingPower)
			if err != nil {
				return nil, err
			}
		}

		thenBlock, err := parseBlock(p)
		if err != nil {
			return nil, err
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
				elseBlock, err = parseBlock(p)
				if err != nil {
					return nil, err
				}
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

	return result, nil
}

func parseWhileStatement(p *parser) (*ast.WhileStatement, error) {

	startPos := p.current.StartPos
	p.next()

	expression, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	block, err := parseBlock(p)
	if err != nil {
		return nil, err
	}

	return ast.NewWhileStatement(p.memoryGauge, expression, block, startPos), nil
}

func parseForStatement(p *parser) (*ast.ForStatement, error) {

	startPos := p.current.StartPos
	p.next()

	p.skipSpaceAndComments(true)

	if p.current.IsString(lexer.TokenIdentifier, keywordIn) {
		p.reportSyntaxError(
			"expected identifier, got keyword %q",
			keywordIn,
		)
		p.next()
	}

	firstValue, err := p.mustIdentifier()
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments(true)

	var index *ast.Identifier
	var identifier ast.Identifier

	if p.current.Is(lexer.TokenComma) {
		p.next()
		p.skipSpaceAndComments(true)
		index = &firstValue
		identifier, err = p.mustIdentifier()
		if err != nil {
			return nil, err
		}

		p.skipSpaceAndComments(true)
	} else {
		identifier = firstValue
	}

	if !p.current.IsString(lexer.TokenIdentifier, keywordIn) {
		p.reportSyntaxError(
			"expected keyword %q, got %s",
			keywordIn,
			p.current.Type,
		)
	}

	p.next()

	expression, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	block, err := parseBlock(p)
	if err != nil {
		return nil, err
	}

	return ast.NewForStatement(
		p.memoryGauge,
		identifier,
		index,
		block,
		expression,
		startPos,
	), nil
}

func parseBlock(p *parser) (*ast.Block, error) {
	startToken, err := p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	statements, err := parseStatements(p, func(token lexer.Token) bool {
		return token.Type == lexer.TokenBraceClose
	})
	if err != nil {
		return nil, err
	}

	endToken, err := p.mustOne(lexer.TokenBraceClose)
	if err != nil {
		return nil, err
	}

	return ast.NewBlock(
		p.memoryGauge,
		statements,
		ast.NewRange(
			p.memoryGauge,
			startToken.StartPos,
			endToken.EndPos,
		),
	), nil
}

func parseFunctionBlock(p *parser) (*ast.FunctionBlock, error) {
	p.skipSpaceAndComments(true)

	startToken, err := p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments(true)

	var preConditions *ast.Conditions
	if p.current.IsString(lexer.TokenIdentifier, keywordPre) {
		p.next()
		conditions, err := parseConditions(p, ast.ConditionKindPre)
		if err != nil {
			return nil, err
		}

		preConditions = &conditions
	}

	p.skipSpaceAndComments(true)

	var postConditions *ast.Conditions
	if p.current.IsString(lexer.TokenIdentifier, keywordPost) {
		p.next()
		conditions, err := parseConditions(p, ast.ConditionKindPost)
		if err != nil {
			return nil, err
		}

		postConditions = &conditions
	}

	statements, err := parseStatements(p, func(token lexer.Token) bool {
		return token.Type == lexer.TokenBraceClose
	})
	if err != nil {
		return nil, err
	}

	endToken, err := p.mustOne(lexer.TokenBraceClose)
	if err != nil {
		return nil, err
	}

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
	), nil
}

// parseConditions parses conditions (pre/post)
//
func parseConditions(p *parser, kind ast.ConditionKind) (conditions ast.Conditions, err error) {

	p.skipSpaceAndComments(true)
	_, err = p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	defer func() {
		p.skipSpaceAndComments(true)
		_, err = p.mustOne(lexer.TokenBraceClose)
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
			var condition *ast.Condition
			condition, err = parseCondition(p, kind)
			if err != nil || condition == nil {
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
func parseCondition(p *parser, kind ast.ConditionKind) (*ast.Condition, error) {

	test, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments(true)

	var message ast.Expression
	if p.current.Is(lexer.TokenColon) {
		p.next()

		message, err = parseExpression(p, lowestBindingPower)
		if err != nil {
			return nil, err
		}
	}

	return &ast.Condition{
		Kind:    kind,
		Test:    test,
		Message: message,
	}, nil
}

func parseEmitStatement(p *parser) (*ast.EmitStatement, error) {
	startPos := p.current.StartPos
	p.next()

	invocation, err := parseNominalTypeInvocationRemainder(p)
	if err != nil {
		return nil, err
	}

	return ast.NewEmitStatement(p.memoryGauge, invocation, startPos), nil
}

func parseSwitchStatement(p *parser) (*ast.SwitchStatement, error) {

	startPos := p.current.StartPos

	// Skip the `switch` keyword
	p.next()

	expression, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	_, err = p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	cases, err := parseSwitchCases(p)
	if err != nil {
		return nil, err
	}

	endToken, err := p.mustOne(lexer.TokenBraceClose)
	if err != nil {
		return nil, err
	}

	return ast.NewSwitchStatement(
		p.memoryGauge,
		expression,
		cases,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			endToken.EndPos,
		),
	), nil
}

// parseSwitchCases parses cases of a switch statement.
//
//     switchCases : switchCase*
//
func parseSwitchCases(p *parser) (cases []*ast.SwitchCase, err error) {

	reportUnexpected := func() {
		p.reportSyntaxError(
			"unexpected token: got %s, expected %q or %q",
			p.current.Type,
			keywordCase,
			keywordDefault,
		)
		p.next()
	}

	for {
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenIdentifier:

			var switchCase *ast.SwitchCase
			switch p.current.Value {
			case keywordCase:
				switchCase, err = parseSwitchCase(p, true)

			case keywordDefault:
				switchCase, err = parseSwitchCase(p, false)

			default:
				reportUnexpected()
				continue
			}

			if err != nil {
				return
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
func parseSwitchCase(p *parser, hasExpression bool) (*ast.SwitchCase, error) {

	startPos := p.current.StartPos

	// Skip the keyword
	p.next()

	var expression ast.Expression
	var err error

	if hasExpression {
		expression, err = parseExpression(p, lowestBindingPower)
		if err != nil {
			return nil, err
		}
	} else {
		p.skipSpaceAndComments(true)
	}

	colonPos := p.current.StartPos

	if !p.current.Is(lexer.TokenColon) {
		p.reportSyntaxError(
			"expected %s, got %s",
			lexer.TokenColon,
			p.current.Type,
		)
	}

	p.next()

	statements, err := parseStatements(p, func(token lexer.Token) bool {
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
	if err != nil {
		return nil, err
	}

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
	}, nil
}
