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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/parser/lexer"
)

func parseStatements(p *parser, isEndToken func(token lexer.Token) bool) (statements []ast.Statement, err error) {
	sawSemicolon := false

	progress := p.newProgress()

	for p.checkProgress(&progress) {

		p.skipSpaceAndComments()

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
						p.report(&StatementSeparationError{
							Pos: currentStartPos,
						})
					}
				}
			}

			sawSemicolon = false
		}
	}

	panic(errors.NewUnreachableError())
}

func parseStatement(p *parser) (ast.Statement, error) {
	p.skipSpaceAndComments()

	// Flag for cases where we can tell early-on that the current token isn't being used as a keyword
	// e.g. soft keywords like `view`
	tokenIsIdentifier := false

	// It might start with a keyword for a statement
	switch p.current.Type {
	case lexer.TokenIdentifier:
		switch string(p.currentTokenSource()) {
		case KeywordReturn:
			return parseReturnStatement(p)
		case KeywordBreak:
			return parseBreakStatement(p), nil
		case KeywordContinue:
			return parseContinueStatement(p), nil
		case KeywordIf:
			return parseIfStatement(p)
		case KeywordSwitch:
			return parseSwitchStatement(p)
		case KeywordWhile:
			return parseWhileStatement(p)
		case KeywordFor:
			return parseForStatement(p)
		case KeywordEmit:
			return parseEmitStatement(p)
		case KeywordRemove:
			return parseRemoveStatement(p)

		case KeywordView:
			// save current stream state before looking ahead for the `fun` keyword
			cursor := p.tokens.Cursor()
			current := p.current
			purityPos := current.StartPos

			p.nextSemanticToken()
			if p.isToken(p.current, lexer.TokenIdentifier, KeywordFun) {
				return parseFunctionDeclarationOrFunctionExpressionStatement(p, ast.FunctionPurityView, &purityPos)
			}

			// no `fun` :( revert back to previous lexer state and treat it as an identifier
			p.tokens.Revert(cursor)
			p.current = current
			tokenIsIdentifier = true

		case KeywordFun:

			// The `fun` keyword is ambiguous: it either introduces a function expression
			// or a function declaration, depending on if an identifier follows, or not.
			return parseFunctionDeclarationOrFunctionExpressionStatement(p, ast.FunctionPurityUnspecified, nil)
		}
	}

	if !tokenIsIdentifier {
		// If it is not a keyword for a statement,
		// it might start with a keyword for a declaration
		declaration, err := parseDeclaration(p, "")
		if err != nil {
			return nil, err
		}

		if statement, ok := declaration.(ast.Statement); ok {
			return statement, nil
		}
	}

	// If it is not a statement or declaration,
	// it must be an expression

	expression, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	// If the expression is followed by a transfer,
	// it is actually the target of an assignment or swap statement

	p.skipSpaceAndComments()
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
	p.nextSemanticToken()

	if p.current.Is(lexer.TokenIdentifier) {
		identifier, err := p.nonReservedIdentifier("after start of function declaration")
		if err != nil {
			return nil, err
		}

		p.next()

		var typeParameterList *ast.TypeParameterList

		if p.config.TypeParametersEnabled {
			var err error
			typeParameterList, err = parseTypeParameterList(p)
			if err != nil {
				return nil, err
			}
		}

		parameterList, returnTypeAnnotation, functionBlock, err :=
			parseFunctionParameterListAndRest(p, false)

		if err != nil {
			return nil, err
		}

		return ast.NewFunctionDeclaration(
			p.memoryGauge,
			ast.AccessNotSpecified,
			purity,
			false,
			false,
			identifier,
			typeParameterList,
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

	sawNewLine, _ := p.parseTrivia(triviaOptions{
		skipNewlines: false,
	})

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

	progress := p.newProgress()

	for p.checkProgress(&progress) {

		startPos := p.current.StartPos

		p.nextSemanticToken()

		var variableDeclaration *ast.VariableDeclaration
		var err error

		if p.current.Type == lexer.TokenIdentifier {
			switch string(p.currentTokenSource()) {
			case KeywordLet:
				const isLet = true
				variableDeclaration, err =
					parseVariableDeclaration(p, ast.AccessNotSpecified, nil, isLet, "")
				if err != nil {
					return nil, err
				}

			case KeywordVar:
				const isLet = false
				variableDeclaration, err =
					parseVariableDeclaration(p, ast.AccessNotSpecified, nil, isLet, "")
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

		p.skipSpaceAndComments()
		if p.isToken(p.current, lexer.TokenIdentifier, KeywordElse) {
			// Skip the `else` keyword
			p.nextSemanticToken()

			if p.isToken(p.current, lexer.TokenIdentifier, KeywordIf) {
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
	p.nextSemanticToken()

	firstValue, err := p.mustIdentifier()
	if err != nil {
		return nil, err
	}

	if firstValue.Identifier == KeywordIn {
		p.report(&InvalidInKeywordAsIdentifierError{
			Pos: p.current.StartPos,
		})
	}

	p.skipSpaceAndComments()

	var index *ast.Identifier
	var identifier ast.Identifier

	if p.current.Is(lexer.TokenComma) {
		p.nextSemanticToken()

		index = &firstValue
		identifier, err = p.mustIdentifier()
		if err != nil {
			return nil, err
		}

		p.skipSpaceAndComments()
	} else {
		identifier = firstValue
	}

	if p.isToken(p.current, lexer.TokenIdentifier, KeywordIn) {
		// Skip the `in` keyword
		p.nextSemanticToken()
	} else {
		p.report(&MissingInKeywordInForStatementError{
			GotToken: p.current,
		})
	}

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

func parseBraces[T any](
	p *parser,
	description string,
	f func() (T, error),
) (
	result T,
	startToken lexer.Token,
	endToken lexer.Token,
	err error,
) {
	startToken = p.current

	if p.current.Is(lexer.TokenBraceOpen) {
		p.next()
	} else {
		p.report(&MissingOpeningBraceError{
			Description: description,
			GotToken:    p.current,
		})
	}

	result, err = f()
	if err != nil {
		return
	}

	p.skipSpaceAndComments()

	endToken = p.current

	if p.current.Is(lexer.TokenBraceClose) {
		p.next()
	} else {
		p.report(&MissingClosingBraceError{
			Description: description,
			GotToken:    p.current,
		})
	}

	return
}

func parseBlock(p *parser) (*ast.Block, error) {
	statements, startToken, endToken, err := parseBraces(
		p,
		"block",
		func() ([]ast.Statement, error) {
			return parseStatements(p, func(token lexer.Token) bool {
				return token.Type == lexer.TokenBraceClose
			})
		},
	)
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
	p.skipSpaceAndComments()

	type bracesResult struct {
		preConditions  *ast.Conditions
		postConditions *ast.Conditions
		statements     []ast.Statement
	}

	res, startToken, endToken, err := parseBraces(
		p,
		"function block",
		func() (res bracesResult, err error) {

			p.skipSpaceAndComments()

			if p.isToken(p.current, lexer.TokenIdentifier, KeywordPre) {
				prePos := p.current.StartPos

				// Skip the `pre` keyword
				p.nextSemanticToken()

				res.preConditions, err = parseConditions(p, prePos)
				if err != nil {
					return
				}
			}

			p.skipSpaceAndComments()

			if p.isToken(p.current, lexer.TokenIdentifier, KeywordPost) {
				startPos := p.current.StartPos

				// Skip the `post` keyword
				p.nextSemanticToken()

				res.postConditions, err = parseConditions(p, startPos)
				if err != nil {
					return
				}
			}

			res.statements, err = parseStatements(p, func(token lexer.Token) bool {
				return token.Type == lexer.TokenBraceClose
			})

			return
		},
	)
	if err != nil {
		return nil, err
	}

	return ast.NewFunctionBlock(
		p.memoryGauge,
		ast.NewBlock(
			p.memoryGauge,
			res.statements,
			ast.NewRange(
				p.memoryGauge,
				startToken.StartPos,
				endToken.EndPos,
			),
		),
		res.preConditions,
		res.postConditions,
	), nil
}

// parseConditions parses conditions (pre/post)
func parseConditions(p *parser, startPos ast.Position) (*ast.Conditions, error) {

	p.skipSpaceAndComments()

	conditions, _, endToken, err := parseBraces(
		p,
		"conditions",
		func() (conditions []ast.Condition, err error) {

			var done bool
			progress := p.newProgress()

			for !done && p.checkProgress(&progress) {

				p.skipSpaceAndComments()

				switch p.current.Type {
				case lexer.TokenSemicolon:
					p.next()
					continue

				case lexer.TokenBraceClose, lexer.TokenEOF:
					done = true

				default:
					var condition ast.Condition
					condition, err = parseCondition(p)
					if err != nil || condition == nil {
						return nil, err
					}

					conditions = append(conditions, condition)
				}
			}

			return
		},
	)
	if err != nil {
		return nil, err
	}

	return &ast.Conditions{
		Conditions: conditions,
		Range: ast.NewRange(
			p.memoryGauge,
			startPos,
			endToken.EndPos,
		),
	}, nil
}

// parseCondition parses a condition (pre/post)
//
//	condition :
//		emitStatement
//		| expression (':' expression )?
func parseCondition(p *parser) (ast.Condition, error) {

	if p.isToken(p.current, lexer.TokenIdentifier, KeywordEmit) {
		emitStatement, err := parseEmitStatement(p)
		if err != nil {
			return nil, err
		}

		return (*ast.EmitCondition)(emitStatement), nil

	}

	test, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	var message ast.Expression
	if p.current.Is(lexer.TokenColon) {
		p.next()

		message, err = parseExpression(p, lowestBindingPower)
		if err != nil {
			return nil, err
		}
	}

	return &ast.TestCondition{
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

	cases, _, endToken, err := parseBraces(
		p,
		"switch cases",
		func() ([]*ast.SwitchCase, error) {
			return parseSwitchCases(p)
		},
	)
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
//	switchCases : switchCase*
func parseSwitchCases(p *parser) (cases []*ast.SwitchCase, err error) {

	progress := p.newProgress()

	for p.checkProgress(&progress) {

		p.skipSpaceAndComments()

		switch p.current.Type {
		case lexer.TokenIdentifier:

			var switchCase *ast.SwitchCase
			switch string(p.currentTokenSource()) {
			case KeywordCase:
				switchCase, err = parseSwitchCase(p, true)
				if err != nil {
					return
				}

			case KeywordDefault:
				switchCase, err = parseSwitchCase(p, false)
				if err != nil {
					return
				}

			default:
				p.report(&ExpectedCaseOrDefaultError{
					GotToken: p.current,
				})
				p.next()
				continue
			}

			cases = append(cases, switchCase)

		case lexer.TokenBraceClose, lexer.TokenEOF:
			return

		default:
			p.report(&ExpectedCaseOrDefaultError{
				GotToken: p.current,
			})
			p.next()
		}
	}

	panic(errors.NewUnreachableError())
}

// parseSwitchCase parses a switch case (hasExpression == true)
// or default case (hasExpression == false)
//
//	switchCase : `case` expression `:` statements
//	           | `default` `:` statements
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
		p.skipSpaceAndComments()
	}

	colonPos := p.current.StartPos

	if p.current.Is(lexer.TokenColon) {
		p.next()
	} else {
		p.report(&MissingColonInSwitchCaseError{
			GotToken: p.current,
		})
	}

	statements, err := parseStatements(p, func(token lexer.Token) bool {
		switch token.Type {
		case lexer.TokenBraceClose:
			return true

		case lexer.TokenIdentifier:
			switch string(p.currentTokenSource()) {
			case KeywordCase, KeywordDefault:
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

	return ast.NewSwitchCase(
		p.memoryGauge,
		expression,
		statements,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			endPos,
		),
	), nil
}

func parseRemoveStatement(
	p *parser,
) (*ast.RemoveStatement, error) {

	startPos := p.current.StartPos
	p.next()
	p.skipSpaceAndComments()

	ty, err := parseType(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	attachmentNominalType, ok := ty.(*ast.NominalType)
	if !ok {
		p.report(&InvalidAttachmentRemovalTypeError{
			Range: ast.NewRangeFromPositioned(p.memoryGauge, ty),
		})
	}

	p.skipSpaceAndComments()

	if p.isToken(p.current, lexer.TokenIdentifier, KeywordFrom) {
		// Skip the `from` keyword
		p.nextSemanticToken()
	} else {
		p.report(&MissingFromKeywordInRemoveStatementError{
			GotToken: p.current,
		})
	}

	attached, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	return ast.NewRemoveStatement(
		p.memoryGauge,
		attachmentNominalType,
		attached,
		startPos,
	), nil
}
