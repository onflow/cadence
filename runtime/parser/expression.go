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
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser/lexer"
)

const exprBindingPowerGap = 10

const (
	exprLeftBindingPowerTernary = exprBindingPowerGap * (iota + 2)
	exprLeftBindingPowerLogicalOr
	exprLeftBindingPowerLogicalAnd
	exprLeftBindingPowerComparison
	exprLeftBindingPowerNilCoalescing
	exprLeftBindingPowerBitwiseOr
	exprLeftBindingPowerBitwiseXor
	exprLeftBindingPowerBitwiseAnd
	exprLeftBindingPowerBitwiseShift
	exprLeftBindingPowerAddition
	exprLeftBindingPowerMultiplication
	exprLeftBindingPowerCasting
	exprLeftBindingPowerUnaryPrefix
	exprLeftBindingPowerUnaryPostfix
	exprLeftBindingPowerAccess
)

type infixExprFunc func(parser *parser, left, right ast.Expression) (ast.Expression, error)
type prefixExprFunc func(parser *parser, right ast.Expression, tokenRange ast.Range) (ast.Expression, error)
type postfixExprFunc func(parser *parser, left ast.Expression, tokenRange ast.Range) (ast.Expression, error)
type exprNullDenotationFunc func(parser *parser, token lexer.Token) (ast.Expression, error)
type exprMetaLeftDenotationFunc func(
	p *parser,
	rightBindingPower int,
	left ast.Expression,
) (
	result ast.Expression,
	err error,
	done bool,
)

type literalExpr struct {
	tokenType      lexer.TokenType
	nullDenotation exprNullDenotationFunc
}

type infixExpr struct {
	tokenType        lexer.TokenType
	leftBindingPower int
	rightAssociative bool
	leftDenotation   infixExprFunc
}

type binaryExpr struct {
	tokenType        lexer.TokenType
	leftBindingPower int
	rightAssociative bool
	operation        ast.Operation
}

type prefixExpr struct {
	tokenType      lexer.TokenType
	bindingPower   int
	nullDenotation prefixExprFunc
}

type unaryExpr struct {
	tokenType    lexer.TokenType
	bindingPower int
	operation    ast.Operation
}

type postfixExpr struct {
	tokenType      lexer.TokenType
	bindingPower   int
	leftDenotation postfixExprFunc
}

var exprNullDenotations = [lexer.TokenMax]exprNullDenotationFunc{}

type exprLeftDenotationFunc func(parser *parser, token lexer.Token, left ast.Expression) (ast.Expression, error)

var exprLeftBindingPowers = [lexer.TokenMax]int{}
var exprIdentifierLeftBindingPowers = map[string]int{}
var exprLeftDenotations = [lexer.TokenMax]exprLeftDenotationFunc{}
var exprMetaLeftDenotations = [lexer.TokenMax]exprMetaLeftDenotationFunc{}

func defineExpr(def any) {
	switch def := def.(type) {
	case infixExpr:
		tokenType := def.tokenType

		setExprLeftBindingPower(tokenType, def.leftBindingPower)

		rightBindingPower := def.leftBindingPower
		if def.rightAssociative {
			rightBindingPower--
		}

		setExprLeftDenotation(
			tokenType,
			func(parser *parser, _ lexer.Token, left ast.Expression) (ast.Expression, error) {
				right, err := parseExpression(parser, rightBindingPower)
				if err != nil {
					return nil, err
				}

				return def.leftDenotation(parser, left, right)
			},
		)

	case binaryExpr:
		defineExpr(infixExpr{
			tokenType:        def.tokenType,
			leftBindingPower: def.leftBindingPower,
			rightAssociative: def.rightAssociative,
			leftDenotation: func(p *parser, left, right ast.Expression) (ast.Expression, error) {
				return ast.NewBinaryExpression(
					p.memoryGauge,
					def.operation,
					left,
					right,
				), nil
			},
		})

	case literalExpr:
		tokenType := def.tokenType
		setExprNullDenotation(tokenType, def.nullDenotation)

	case prefixExpr:
		tokenType := def.tokenType
		setExprNullDenotation(
			tokenType,
			func(parser *parser, token lexer.Token) (ast.Expression, error) {
				right, err := parseExpression(parser, def.bindingPower)
				if err != nil {
					return nil, err
				}

				return def.nullDenotation(parser, right, token.Range)
			},
		)

	case unaryExpr:
		defineExpr(prefixExpr{
			tokenType:    def.tokenType,
			bindingPower: def.bindingPower,
			nullDenotation: func(p *parser, right ast.Expression, tokenRange ast.Range) (ast.Expression, error) {
				return ast.NewUnaryExpression(
					p.memoryGauge,
					def.operation,
					right,
					tokenRange.StartPos,
				), nil
			},
		})

	case postfixExpr:
		tokenType := def.tokenType
		setExprLeftBindingPower(tokenType, def.bindingPower)
		setExprLeftDenotation(
			tokenType,
			func(p *parser, token lexer.Token, left ast.Expression) (ast.Expression, error) {
				return def.leftDenotation(p, left, token.Range)
			},
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

func setExprNullDenotation(tokenType lexer.TokenType, nullDenotation exprNullDenotationFunc) {
	current := exprNullDenotations[tokenType]
	if current != nil {
		panic(NewUnpositionedSyntaxError(
			"expression null denotation for token %s already exists",
			tokenType,
		))
	}
	exprNullDenotations[tokenType] = nullDenotation
}

func setExprLeftBindingPower(tokenType lexer.TokenType, power int) {
	current := exprLeftBindingPowers[tokenType]
	if current > power {
		return
	}
	exprLeftBindingPowers[tokenType] = power
}

func setExprIdentifierLeftBindingPower(keyword string, power int) {
	current := exprIdentifierLeftBindingPowers[keyword]
	if current > power {
		return
	}
	exprIdentifierLeftBindingPowers[keyword] = power
}

func setExprLeftDenotation(tokenType lexer.TokenType, leftDenotation exprLeftDenotationFunc) {
	current := exprLeftDenotations[tokenType]
	if current != nil {
		panic(NewUnpositionedSyntaxError(
			"expression left denotation for token %s already exists",
			tokenType,
		))
	}

	exprLeftDenotations[tokenType] = leftDenotation
}

func setExprMetaLeftDenotation(tokenType lexer.TokenType, metaLeftDenotation exprMetaLeftDenotationFunc) {
	current := exprMetaLeftDenotations[tokenType]
	if current != nil {
		panic(NewUnpositionedSyntaxError(
			"expression meta left denotation for token %s already exists",
			tokenType,
		))
	}
	exprMetaLeftDenotations[tokenType] = metaLeftDenotation
}

// init defines the binding power for operations.
func init() {

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenVerticalBarVerticalBar,
		leftBindingPower: exprLeftBindingPowerLogicalOr,
		operation:        ast.OperationOr,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenAmpersandAmpersand,
		leftBindingPower: exprLeftBindingPowerLogicalAnd,
		operation:        ast.OperationAnd,
	})

	defineLessThanOrTypeArgumentsExpression()
	defineGreaterThanOrBitwiseRightShiftExpression()

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenLessEqual,
		leftBindingPower: exprLeftBindingPowerComparison,
		operation:        ast.OperationLessEqual,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenGreaterEqual,
		leftBindingPower: exprLeftBindingPowerComparison,
		operation:        ast.OperationGreaterEqual,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenEqualEqual,
		leftBindingPower: exprLeftBindingPowerComparison,
		operation:        ast.OperationEqual,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenNotEqual,
		leftBindingPower: exprLeftBindingPowerComparison,
		operation:        ast.OperationNotEqual,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenDoubleQuestionMark,
		leftBindingPower: exprLeftBindingPowerNilCoalescing,
		operation:        ast.OperationNilCoalesce,
		rightAssociative: true,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenVerticalBar,
		leftBindingPower: exprLeftBindingPowerBitwiseOr,
		operation:        ast.OperationBitwiseOr,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenCaret,
		leftBindingPower: exprLeftBindingPowerBitwiseXor,
		operation:        ast.OperationBitwiseXor,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenAmpersand,
		leftBindingPower: exprLeftBindingPowerBitwiseAnd,
		operation:        ast.OperationBitwiseAnd,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenLessLess,
		leftBindingPower: exprLeftBindingPowerBitwiseShift,
		operation:        ast.OperationBitwiseLeftShift,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenPlus,
		leftBindingPower: exprLeftBindingPowerAddition,
		operation:        ast.OperationPlus,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenMinus,
		leftBindingPower: exprLeftBindingPowerAddition,
		operation:        ast.OperationMinus,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenStar,
		leftBindingPower: exprLeftBindingPowerMultiplication,
		operation:        ast.OperationMul,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenSlash,
		leftBindingPower: exprLeftBindingPowerMultiplication,
		operation:        ast.OperationDiv,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenPercent,
		leftBindingPower: exprLeftBindingPowerMultiplication,
		operation:        ast.OperationMod,
	})

	defineCastingExpression()

	defineExpr(literalExpr{
		tokenType: lexer.TokenBinaryIntegerLiteral,
		nullDenotation: func(p *parser, token lexer.Token) (ast.Expression, error) {
			literal, ok := token.Value.(string)
			if !ok {
				return nil, p.syntaxError(
					"value for token %s was not a string",
					lexer.TokenBinaryIntegerLiteral,
				)
			}
			return parseIntegerLiteral(
				p,
				literal,
				literal[2:],
				IntegerLiteralKindBinary,
				token.Range,
			), nil
		},
	})

	defineExpr(literalExpr{
		tokenType: lexer.TokenOctalIntegerLiteral,
		nullDenotation: func(p *parser, token lexer.Token) (ast.Expression, error) {
			literal, ok := token.Value.(string)
			if !ok {
				return nil, p.syntaxError(
					"value for token %s was not a string",
					lexer.TokenOctalIntegerLiteral,
				)
			}
			return parseIntegerLiteral(
				p,
				literal,
				literal[2:],
				IntegerLiteralKindOctal,
				token.Range,
			), nil
		},
	})

	defineExpr(literalExpr{
		tokenType: lexer.TokenDecimalIntegerLiteral,
		nullDenotation: func(p *parser, token lexer.Token) (ast.Expression, error) {
			literal, ok := token.Value.(string)
			if !ok {
				return nil, p.syntaxError(
					"value for token %s was not a string",
					lexer.TokenDecimalIntegerLiteral,
				)
			}
			return parseIntegerLiteral(
				p,
				literal,
				literal,
				IntegerLiteralKindDecimal,
				token.Range,
			), nil
		},
	})

	defineExpr(literalExpr{
		tokenType: lexer.TokenHexadecimalIntegerLiteral,
		nullDenotation: func(p *parser, token lexer.Token) (ast.Expression, error) {
			literal, ok := token.Value.(string)
			if !ok {
				return nil, p.syntaxError(
					"value for token %s was not a string",
					lexer.TokenHexadecimalIntegerLiteral,
				)
			}
			return parseIntegerLiteral(
				p,
				literal,
				literal[2:],
				IntegerLiteralKindHexadecimal,
				token.Range,
			), nil
		},
	})

	defineExpr(literalExpr{
		tokenType: lexer.TokenUnknownBaseIntegerLiteral,
		nullDenotation: func(p *parser, token lexer.Token) (ast.Expression, error) {
			literal, ok := token.Value.(string)
			if !ok {
				return nil, p.syntaxError(
					"value for token %s was not a string",
					lexer.TokenUnknownBaseIntegerLiteral,
				)
			}
			return parseIntegerLiteral(
				p,
				literal,
				literal[2:],
				IntegerLiteralKindUnknown,
				token.Range,
			), nil
		},
	})

	defineExpr(literalExpr{
		tokenType: lexer.TokenFixedPointNumberLiteral,
		nullDenotation: func(p *parser, token lexer.Token) (ast.Expression, error) {
			return parseFixedPointLiteral(
				p,
				token.Value.(string),
				token.Range,
			), nil
		},
	})

	defineExpr(literalExpr{
		tokenType: lexer.TokenString,
		nullDenotation: func(p *parser, token lexer.Token) (ast.Expression, error) {
			parsedString := parseStringLiteral(p, token.Value.(string))
			return ast.NewStringExpression(
				p.memoryGauge,
				parsedString,
				token.Range,
			), nil
		},
	})

	defineExpr(prefixExpr{
		tokenType:    lexer.TokenMinus,
		bindingPower: exprLeftBindingPowerUnaryPrefix,
		nullDenotation: func(p *parser, right ast.Expression, tokenRange ast.Range) (ast.Expression, error) {
			switch right := right.(type) {
			case *ast.IntegerExpression:
				if right.Value.Sign() > 0 {
					if right.Value != nil {
						right.Value.Neg(right.Value)
					}
					right.StartPos = tokenRange.StartPos
					return right, nil
				}

			case *ast.FixedPointExpression:
				if !right.Negative {
					right.Negative = !right.Negative
					right.StartPos = tokenRange.StartPos
					return right, nil
				}
			}

			return ast.NewUnaryExpression(
				p.memoryGauge,
				ast.OperationMinus,
				right,
				tokenRange.StartPos,
			), nil
		},
	})

	defineExpr(unaryExpr{
		tokenType:    lexer.TokenExclamationMark,
		bindingPower: exprLeftBindingPowerUnaryPrefix,
		operation:    ast.OperationNegate,
	})

	defineExpr(unaryExpr{
		tokenType:    lexer.TokenLeftArrow,
		bindingPower: exprLeftBindingPowerUnaryPrefix,
		operation:    ast.OperationMove,
	})

	defineExpr(postfixExpr{
		tokenType:    lexer.TokenExclamationMark,
		bindingPower: exprLeftBindingPowerUnaryPostfix,
		leftDenotation: func(p *parser, left ast.Expression, tokenRange ast.Range) (ast.Expression, error) {
			return ast.NewForceExpression(
				p.memoryGauge,
				left,
				tokenRange.EndPos,
			), nil
		},
	})

	defineNestedExpression()
	defineInvocationExpression()
	defineArrayExpression()
	defineDictionaryExpression()
	defineIndexExpression()
	definePathExpression()
	defineConditionalExpression()
	defineReferenceExpression()
	defineMemberExpression()
	defineIdentifierExpression()

	setExprNullDenotation(lexer.TokenEOF, func(parser *parser, token lexer.Token) (ast.Expression, error) {
		return nil, NewUnpositionedSyntaxError("expected expression")
	})
}

func defineLessThanOrTypeArgumentsExpression() {

	// The less token `<` does not have a single left binding power,
	// but one depending on the tokens following it:
	//
	// Either an invocation with type arguments (zero or more, comma separated),
	// followed by a closing greater token `>` and argument list;
	// or a normal expression.
	//
	//     lessThenOrTypeArguments : '<'
	//         ( ( ( typeAnnotation ( ',' )* )? '>' argumentList )
	//         | expression
	//         )
	//
	//
	// Parse this ambiguity by first trying to parse type arguments
	// and a closing greater token `>` and start of an argument list,
	// i.e. the open paren token `(`.
	//
	// If that parse fails, the result expression must be a binary expression,
	// and a normal expression must follow.
	//
	// In both cases, the right binding power must be checked,
	// just like it is before a normal left denotation is applied.

	const binaryExpressionLeftBindingPower = exprLeftBindingPowerComparison
	const invocationExpressionLeftBindingPower = exprLeftBindingPowerAccess

	setExprMetaLeftDenotation(
		lexer.TokenLess,
		func(p *parser, rightBindingPower int, left ast.Expression) (result ast.Expression, err error, done bool) {

			var typeArguments []*ast.TypeAnnotation

			// Start buffering before skipping the `<` token,
			// so it can be replayed in case the right binding power
			// was higher than the determined left binding power.

			p.startBuffering()
			p.startAmbiguity()
			defer p.endAmbiguity()

			// Skip the `<` token.
			p.next()
			p.skipSpaceAndComments(true)

			// First, try to parse zero or more comma-separated type
			// arguments (type annotations), a closing greater token `>`,
			// and the start of an argument list, i.e. the open paren token `(`.
			//
			// This parse may fail, in which case we just ignore the error,
			// except for fatal errors.

			var argumentsStartPos ast.Position

			err = func() error {
				defer func() {
					err := recover()
					// MemoryError should abort parsing
					_, ok := err.(errors.MemoryError)
					if ok {
						panic(err)
					}
				}()

				typeArguments, err = parseCommaSeparatedTypeAnnotations(p, lexer.TokenGreater)
				if err != nil {
					return err
				}

				_, err = p.mustOne(lexer.TokenGreater)
				if err != nil {
					return err
				}

				p.skipSpaceAndComments(true)
				parenOpenToken, err := p.mustOne(lexer.TokenParenOpen)
				if err != nil {
					return err
				}

				argumentsStartPos = parenOpenToken.EndPos

				return nil
			}()

			// `err` is nil means the expression is an invocation
			if err == nil {

				// The expression was determined to be an invocation.
				// Still, it should have maybe not been parsed if the right binding power
				// was higher. In that case, replay the buffered tokens and stop.

				if rightBindingPower >= invocationExpressionLeftBindingPower {
					err = p.replayBuffered()
					if err != nil {
						return nil, err, true
					}

					return left, nil, true
				}

				// The previous attempt to parse an invocation succeeded,
				// accept the buffered tokens.

				p.acceptBuffered()

				arguments, endPos, err := parseArgumentListRemainder(p)
				if err != nil {
					return nil, err, true
				}

				invocationExpression := ast.NewInvocationExpression(
					p.memoryGauge,
					left,
					typeArguments,
					arguments,
					argumentsStartPos,
					endPos,
				)

				return invocationExpression, nil, false

			} else {

				// The previous attempt to parse an invocation failed,
				// replay the buffered tokens.

				err = p.replayBuffered()
				if err != nil {
					return nil, err, true
				}

				// The expression was determined to *not* be an invocation,
				// so it must be a binary expression.
				//
				// Like for a normal left denotation,
				// check if this left denotation applies.

				if rightBindingPower >= binaryExpressionLeftBindingPower {
					return left, nil, true
				}

				// Skip the `<` token.
				// The token buffering started before this token,
				// because it should have maybe not been parsed in the first place
				// if the right binding power is higher.

				p.next()
				p.skipSpaceAndComments(true)

				right, err := parseExpression(p, binaryExpressionLeftBindingPower)
				if err != nil {
					return nil, err, true
				}

				binaryExpression := ast.NewBinaryExpression(
					p.memoryGauge,
					ast.OperationLess,
					left,
					right,
				)

				return binaryExpression, nil, false
			}
		})
}

// defineGreaterThanOrBitwiseRightShiftExpression parses
// the greater-than expression (operator `>`, e.g. `1 > 2`)
// and the bitwise right shift expression (operator `>>`, e.g. `1 >> 3`).
//
// The `>>` operator consists of two `>` tokens, instead of one dedicated `>>` token,
// because that would introduce a parsing problem for function calls/invocations
// which have a type argument, where the type argument is a type instantiation,
// for example, `f<T<U>>()`.
//
func defineGreaterThanOrBitwiseRightShiftExpression() {

	setExprMetaLeftDenotation(
		lexer.TokenGreater,
		func(p *parser, rightBindingPower int, left ast.Expression) (result ast.Expression, err error, done bool) {

			// If the right binding power is higher than any of the potential cases,
			// then return early

			if rightBindingPower >= exprLeftBindingPowerBitwiseShift &&
				rightBindingPower >= exprLeftBindingPowerComparison {

				return left, nil, true
			}

			// Start buffering before skipping the `>` token,
			// so it can be replayed in case the right binding power
			// was higher than the determined left binding power.

			p.startBuffering()

			// Skip the `>` token.
			p.next()

			// If another '>' token appears immediately,
			// then the operator is actually a bitwise right shift operator

			isBitwiseShift := p.current.Is(lexer.TokenGreater)

			var operation ast.Operation
			var nextRightBindingPower int

			if isBitwiseShift {

				operation = ast.OperationBitwiseRightShift

				// The expression was determined to be a bitwise shift.
				// Still, it should have maybe not been parsed if the right binding power
				// was higher. In that case, replay the buffered tokens and stop.

				if rightBindingPower >= exprLeftBindingPowerBitwiseShift {
					err = p.replayBuffered()
					return left, err, true
				}

				// The previous attempt to parse a bitwise right shift succeeded,
				// accept the buffered tokens.

				p.acceptBuffered()

				nextRightBindingPower = exprLeftBindingPowerBitwiseShift

			} else {

				operation = ast.OperationGreater

				// The previous attempt to parse a bitwise right shift failed,
				// replay the buffered tokens.

				err = p.replayBuffered()
				if err != nil {
					return nil, err, true
				}

				// The expression was determined to *not* be a bitwise shift,
				// so it must be a comparison expression.
				//
				// Like for a normal left denotation,
				// check if this left denotation applies.

				if rightBindingPower >= exprLeftBindingPowerComparison {
					return left, nil, true
				}

				nextRightBindingPower = exprLeftBindingPowerComparison
			}

			p.next()
			p.skipSpaceAndComments(true)

			right, err := parseExpression(p, nextRightBindingPower)
			if err != nil {
				return nil, err, true
			}

			binaryExpression := ast.NewBinaryExpression(
				p.memoryGauge,
				operation,
				left,
				right,
			)

			return binaryExpression, err, false
		})
}

func defineIdentifierExpression() {
	defineExpr(literalExpr{
		tokenType: lexer.TokenIdentifier,
		nullDenotation: func(p *parser, token lexer.Token) (ast.Expression, error) {
			switch token.Value {
			case keywordTrue:
				return ast.NewBoolExpression(p.memoryGauge, true, token.Range), nil

			case keywordFalse:
				return ast.NewBoolExpression(p.memoryGauge, false, token.Range), nil

			case keywordNil:
				return ast.NewNilExpression(p.memoryGauge, token.Range.StartPos), nil

			case keywordCreate:
				return parseCreateExpressionRemainder(p, token)

			case keywordDestroy:
				expression, err := parseExpression(p, lowestBindingPower)
				if err != nil {
					return nil, err
				}

				return ast.NewDestroyExpression(
					p.memoryGauge,
					expression,
					token.Range.StartPos,
				), nil

			case keywordFun:
				return parseFunctionExpression(p, token)

			default:
				return ast.NewIdentifierExpression(
					p.memoryGauge,
					p.tokenToIdentifier(token),
				), nil
			}
		},
	})
}

func parseFunctionExpression(p *parser, token lexer.Token) (*ast.FunctionExpression, error) {

	parameterList, returnTypeAnnotation, functionBlock, err :=
		parseFunctionParameterListAndRest(p, false)
	if err != nil {
		return nil, err
	}

	return ast.NewFunctionExpression(
		p.memoryGauge,
		parameterList,
		returnTypeAnnotation,
		functionBlock,
		token.StartPos,
	), nil
}

func defineCastingExpression() {

	setExprIdentifierLeftBindingPower(keywordAs, exprLeftBindingPowerCasting)
	setExprLeftDenotation(
		lexer.TokenIdentifier,
		func(parser *parser, t lexer.Token, left ast.Expression) (ast.Expression, error) {
			switch t.Value.(string) {
			case keywordAs:
				right, err := parseTypeAnnotation(parser)
				if err != nil {
					return nil, err
				}

				return ast.NewCastingExpression(
					parser.memoryGauge,
					left,
					ast.OperationCast,
					right,
					nil,
				), nil
			default:
				panic(errors.NewUnreachableError())
			}
		},
	)

	for _, tokenOperation := range []struct {
		token     lexer.TokenType
		operation ast.Operation
	}{
		{
			token:     lexer.TokenAsExclamationMark,
			operation: ast.OperationForceCast,
		},
		{
			token:     lexer.TokenAsQuestionMark,
			operation: ast.OperationFailableCast,
		},
	} {
		operation := tokenOperation.operation
		tokenType := tokenOperation.token

		// Rebind operation, so the closure captures to current iteration's value,
		// i.e. the next iteration doesn't override `operation`

		leftDenotation := (func(operation ast.Operation) exprLeftDenotationFunc {
			return func(parser *parser, t lexer.Token, left ast.Expression) (ast.Expression, error) {
				right, err := parseTypeAnnotation(parser)
				if err != nil {
					return nil, err
				}

				return ast.NewCastingExpression(
					parser.memoryGauge,
					left,
					operation,
					right,
					nil,
				), nil
			}
		})(operation)

		setExprLeftBindingPower(tokenType, exprLeftBindingPowerCasting)
		setExprLeftDenotation(tokenType, leftDenotation)
	}
}

func parseCreateExpressionRemainder(p *parser, token lexer.Token) (*ast.CreateExpression, error) {
	invocation, err := parseNominalTypeInvocationRemainder(p)
	if err != nil {
		return nil, err
	}

	return ast.NewCreateExpression(
		p.memoryGauge,
		invocation,
		token.StartPos,
	), nil
}

// Invocation Expression Grammar:
//
//     invocation : '(' ( argument ( ',' argument )* )? ')'
//
func defineInvocationExpression() {
	setExprLeftBindingPower(lexer.TokenParenOpen, exprLeftBindingPowerAccess)

	setExprLeftDenotation(
		lexer.TokenParenOpen,
		func(p *parser, token lexer.Token, left ast.Expression) (ast.Expression, error) {
			arguments, endPos, err := parseArgumentListRemainder(p)
			if err != nil {
				return nil, err
			}

			return ast.NewInvocationExpression(
				p.memoryGauge,
				left,
				nil,
				arguments,
				token.EndPos,
				endPos,
			), nil
		},
	)
}

func parseArgumentListRemainder(p *parser) (arguments []*ast.Argument, endPos ast.Position, err error) {
	atEnd := false
	expectArgument := true
	for !atEnd {
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenComma:
			if expectArgument {
				return nil, ast.EmptyPosition, p.syntaxError(
					"expected argument or end of argument list, got %s",
					p.current.Type,
				)
			}
			// Skip the comma
			p.next()
			expectArgument = true

		case lexer.TokenParenClose:
			endPos = p.current.EndPos
			// Skip the closing paren
			p.next()
			atEnd = true

		case lexer.TokenEOF:
			return nil,
				ast.EmptyPosition,
				p.syntaxError("missing ')' at end of invocation argument list")

		default:
			if !expectArgument {
				return nil,
					ast.EmptyPosition,
					p.syntaxError(
						"unexpected argument in argument list (expecting delimiter or end of argument list), got %s",
						p.current.Type,
					)
			}

			argument, err := parseArgument(p)
			if err != nil {
				return nil, ast.EmptyPosition, err
			}

			p.skipSpaceAndComments(true)

			argument.TrailingSeparatorPos = p.current.StartPos

			arguments = append(arguments, argument)

			expectArgument = false
		}
	}
	return
}

// parseArgument parses an argument in an invocation.
//
//     argument : (identifier ':' )? expression
//
func parseArgument(p *parser) (*ast.Argument, error) {
	var label string
	var labelStartPos, labelEndPos ast.Position

	expr, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments(true)

	// If a colon follows the expression, the expression was our label.
	if p.current.Is(lexer.TokenColon) {
		identifier, ok := expr.(*ast.IdentifierExpression)
		if !ok {
			return nil, p.syntaxError(
				"expected identifier for label, got %s",
				expr,
			)
		}
		label = identifier.Identifier.Identifier
		labelStartPos = expr.StartPosition()
		labelEndPos = expr.EndPosition(p.memoryGauge)

		// Skip the identifier
		p.next()
		p.skipSpaceAndComments(true)

		expr, err = parseExpression(p, lowestBindingPower)
		if err != nil {
			return nil, err
		}
	}

	if len(label) > 0 {
		return ast.NewArgument(
			p.memoryGauge,
			label,
			&labelStartPos,
			&labelEndPos,
			expr,
		), nil
	}
	return ast.NewUnlabeledArgument(p.memoryGauge, expr), nil
}

func defineNestedExpression() {
	setExprNullDenotation(
		lexer.TokenParenOpen,
		func(p *parser, token lexer.Token) (ast.Expression, error) {
			expression, err := parseExpression(p, lowestBindingPower)
			if err != nil {
				return nil, err
			}

			_, err = p.mustOne(lexer.TokenParenClose)
			return expression, err
		},
	)
}

func defineArrayExpression() {
	setExprNullDenotation(
		lexer.TokenBracketOpen,
		func(p *parser, startToken lexer.Token) (ast.Expression, error) {
			var values []ast.Expression
			for !p.current.Is(lexer.TokenBracketClose) {
				value, err := parseExpression(p, lowestBindingPower)
				if err != nil {
					return nil, err
				}

				values = append(values, value)
				if !p.current.Is(lexer.TokenComma) {
					break
				}

				_, err = p.mustOne(lexer.TokenComma)
				if err != nil {
					return nil, err
				}
			}

			endToken, err := p.mustOne(lexer.TokenBracketClose)
			if err != nil {
				return nil, err
			}

			return ast.NewArrayExpression(
				p.memoryGauge,
				values,
				ast.NewRange(
					p.memoryGauge,
					startToken.StartPos,
					endToken.EndPos,
				),
			), nil
		},
	)
}

func defineDictionaryExpression() {
	setExprNullDenotation(
		lexer.TokenBraceOpen,
		func(p *parser, startToken lexer.Token) (ast.Expression, error) {
			var entries []ast.DictionaryEntry
			for !p.current.Is(lexer.TokenBraceClose) {
				key, err := parseExpression(p, lowestBindingPower)
				if err != nil {
					return nil, err
				}

				_, err = p.mustOne(lexer.TokenColon)
				if err != nil {
					return nil, err
				}

				value, err := parseExpression(p, lowestBindingPower)
				if err != nil {
					return nil, err
				}

				entries = append(entries, ast.NewDictionaryEntry(
					p.memoryGauge,
					key,
					value,
				))
				if !p.current.Is(lexer.TokenComma) {
					break
				}

				_, err = p.mustOne(lexer.TokenComma)
				if err != nil {
					return nil, err
				}
			}
			endToken, err := p.mustOne(lexer.TokenBraceClose)
			if err != nil {
				return nil, err
			}

			return ast.NewDictionaryExpression(
				p.memoryGauge,
				entries,
				ast.NewRange(
					p.memoryGauge,
					startToken.StartPos,
					endToken.EndPos,
				),
			), nil
		},
	)
}

func defineIndexExpression() {
	setExprLeftBindingPower(lexer.TokenBracketOpen, exprLeftBindingPowerAccess)
	setExprLeftDenotation(
		lexer.TokenBracketOpen,
		func(p *parser, token lexer.Token, left ast.Expression) (ast.Expression, error) {
			firstIndexExpr, err := parseExpression(p, lowestBindingPower)
			if err != nil {
				return nil, err
			}

			endToken, err := p.mustOne(lexer.TokenBracketClose)
			if err != nil {
				return nil, err
			}

			return ast.NewIndexExpression(
				p.memoryGauge,
				left,
				firstIndexExpr,
				ast.NewRange(
					p.memoryGauge,
					token.StartPos,
					endToken.EndPos,
				),
			), nil
		},
	)
}

func defineConditionalExpression() {
	setExprLeftBindingPower(lexer.TokenQuestionMark, exprLeftBindingPowerTernary)
	setExprLeftDenotation(
		lexer.TokenQuestionMark,
		func(p *parser, _ lexer.Token, left ast.Expression) (ast.Expression, error) {
			testExpression := left
			thenExpression, err := parseExpression(p, lowestBindingPower)
			if err != nil {
				return nil, err
			}

			_, err = p.mustOne(lexer.TokenColon)
			if err != nil {
				return nil, err
			}

			elseExpression, err := parseExpression(p, lowestBindingPower)
			if err != nil {
				return nil, err
			}

			return ast.NewConditionalExpression(
				p.memoryGauge,
				testExpression,
				thenExpression,
				elseExpression,
			), nil
		},
	)
}

func definePathExpression() {
	setExprNullDenotation(
		lexer.TokenSlash,
		func(p *parser, token lexer.Token) (ast.Expression, error) {
			domain, err := p.mustIdentifier()
			if err != nil {
				return nil, err
			}

			_, err = p.mustOne(lexer.TokenSlash)
			if err != nil {
				return nil, err
			}

			identifier, err := p.mustIdentifier()
			if err != nil {
				return nil, err
			}

			return ast.NewPathExpression(
				p.memoryGauge,
				domain,
				identifier,
				token.StartPos,
			), nil
		},
	)
}

func defineReferenceExpression() {
	setExprNullDenotation(
		lexer.TokenAmpersand,
		func(p *parser, token lexer.Token) (ast.Expression, error) {
			p.skipSpaceAndComments(true)
			expression, err := parseExpression(p, exprLeftBindingPowerCasting-exprBindingPowerGap)
			if err != nil {
				return nil, err
			}

			p.skipSpaceAndComments(true)

			castingExpression, ok := expression.(*ast.CastingExpression)
			if !ok {
				panic(fmt.Errorf("expected casting expression"))
			}

			return ast.NewReferenceExpression(
				p.memoryGauge,
				castingExpression.Expression,
				castingExpression.TypeAnnotation.Type,
				token.StartPos,
			), nil
		},
	)
}

func defineMemberExpression() {

	setExprLeftBindingPower(lexer.TokenDot, exprLeftBindingPowerAccess)
	setExprLeftDenotation(
		lexer.TokenDot,
		func(p *parser, token lexer.Token, left ast.Expression) (ast.Expression, error) {
			return parseMemberAccess(p, token, left, false), nil
		},
	)

	setExprLeftBindingPower(lexer.TokenQuestionMarkDot, exprLeftBindingPowerAccess)
	setExprLeftDenotation(
		lexer.TokenQuestionMarkDot,
		func(p *parser, token lexer.Token, left ast.Expression) (ast.Expression, error) {
			return parseMemberAccess(p, token, left, true), nil
		},
	)
}

func parseMemberAccess(p *parser, token lexer.Token, left ast.Expression, optional bool) ast.Expression {

	// Whitespace after the '.' (dot token) is not allowed.
	// We parse it anyways and report an error

	if p.current.Is(lexer.TokenSpace) {
		errorPos := p.current.StartPos
		p.skipSpaceAndComments(true)
		p.report(NewSyntaxError(
			errorPos,
			"invalid whitespace after %s",
			lexer.TokenDot,
		))
	}

	// If there is an identifier, use it.
	// If not, report an error

	var identifier ast.Identifier
	if p.current.Is(lexer.TokenIdentifier) {
		identifier = p.tokenToIdentifier(p.current)
		p.next()
	} else {
		p.reportSyntaxError(
			"expected member name, got %s",
			p.current.Type,
		)
	}

	return ast.NewMemberExpression(
		p.memoryGauge,
		left,
		optional,
		// NOTE: use the end position, because the token
		// can be an optional access token `?.`
		token.EndPos,
		identifier,
	)
}

func exprLeftDenotationAllowsNewlineAfterNullDenotation(tokenType lexer.TokenType) bool {

	// The postfix force unwrap, invocation expressions,
	// and indexing expressions don't support newlines before them,
	// as this clashes with a unary negations, nested expressions,
	// and array literals on a new line / separate statement.

	switch tokenType {
	case lexer.TokenExclamationMark, lexer.TokenParenOpen, lexer.TokenBracketOpen:
		return false
	default:
		return true
	}
}

func exprLeftDenotationAllowsWhitespaceAfterToken(tokenType lexer.TokenType) bool {

	// The member access expressions, which starts with a '.' (dot token)
	// or `?.` (question mark dot token), do not allow whitespace
	// after the token (before the identifier)

	switch tokenType {
	case lexer.TokenDot, lexer.TokenQuestionMarkDot:
		return false
	default:
		return true
	}
}

// parseExpression uses "Top-Down operator precedence parsing" (TDOP) technique to
// parse expressions.
//
func parseExpression(p *parser, rightBindingPower int) (ast.Expression, error) {

	if p.expressionDepth == expressionDepthLimit {
		return nil, ExpressionDepthLimitReachedError{
			Pos: p.current.StartPos,
		}
	}

	p.expressionDepth++
	defer func() {
		p.expressionDepth--
	}()

	p.skipSpaceAndComments(true)
	t := p.current
	p.next()

	newLineAfterLeft := p.skipSpaceAndComments(true)

	left, err := applyExprNullDenotation(p, t)
	if err != nil {
		return nil, err
	}

	for {
		newLineAfterLeft = p.skipSpaceAndComments(true) || newLineAfterLeft

		if newLineAfterLeft && !exprLeftDenotationAllowsNewlineAfterNullDenotation(p.current.Type) {
			break
		}

		var done bool
		left, err, done = applyExprMetaLeftDenotation(p, rightBindingPower, left)
		if err != nil {
			return nil, err
		}

		if done {
			break
		}

		newLineAfterLeft = false
	}

	return left, nil
}

func applyExprMetaLeftDenotation(
	p *parser,
	rightBindingPower int,
	left ast.Expression,
) (
	result ast.Expression,
	err error,
	done bool,
) {
	// By default, left denotations are applied if the right binding power
	// is less than the left binding power of the current token.
	//
	// Token-specific meta-left denotations allow customizing this,
	// e.g. determining the left binding power based on parsing more tokens
	// or performing look-ahead

	metaLeftDenotation := exprMetaLeftDenotations[p.current.Type]
	if metaLeftDenotation == nil {
		metaLeftDenotation = defaultExprMetaLeftDenotation
	}

	return metaLeftDenotation(p, rightBindingPower, left)
}

// defaultExprMetaLeftDenotation is the default expression left denotation, which applies
// if the right binding power is less than the left binding power of the current token
//
func defaultExprMetaLeftDenotation(
	p *parser,
	rightBindingPower int,
	left ast.Expression,
) (
	result ast.Expression,
	err error,
	done bool,
) {
	leftBindingPower, err := exprLeftBindingPower(p)
	if err != nil {
		return nil, err, true
	}

	if rightBindingPower >= leftBindingPower {
		return left, nil, true
	}

	allowWhitespace := exprLeftDenotationAllowsWhitespaceAfterToken(p.current.Type)

	t := p.current

	p.next()
	if allowWhitespace {
		p.skipSpaceAndComments(true)
	}

	result, err = applyExprLeftDenotation(p, t, left)
	return result, err, false
}

func exprLeftBindingPower(p *parser) (int, error) {
	token := p.current
	tokenType := token.Type
	if tokenType == lexer.TokenIdentifier {
		identifier, ok := token.Value.(string)
		if !ok {
			return 0, p.syntaxError(
				"value for token %s was not a string",
				tokenType,
			)
		}
		return exprIdentifierLeftBindingPowers[identifier], nil
	}
	return exprLeftBindingPowers[tokenType], nil
}

func applyExprNullDenotation(p *parser, token lexer.Token) (ast.Expression, error) {
	tokenType := token.Type
	nullDenotation := exprNullDenotations[tokenType]
	if nullDenotation == nil {
		return nil, p.syntaxError("unexpected token in expression: %s", tokenType)
	}
	return nullDenotation(p, token)
}

func applyExprLeftDenotation(p *parser, token lexer.Token, left ast.Expression) (ast.Expression, error) {
	leftDenotation := exprLeftDenotations[token.Type]
	if leftDenotation == nil {
		return nil, p.syntaxError("unexpected token in expression: %s", token.Type)
	}
	return leftDenotation(p, token, left)
}

// parseStringLiteral parses a whole string literal, including start and end quotes
//
func parseStringLiteral(p *parser, literal string) (result string) {
	length := len(literal)
	if length == 0 {
		p.reportSyntaxError("missing start of string literal: expected '\"'")
		return
	}

	if length >= 1 {
		first := literal[0]
		if first != '"' {
			p.reportSyntaxError("invalid start of string literal: expected '\"', got %q", first)
		}
	}

	missingEnd := false
	endOffset := length
	if length >= 2 {
		lastIndex := length - 1
		last := literal[lastIndex]
		if last != '"' {
			missingEnd = true
		} else {
			endOffset = lastIndex
		}
	} else {
		missingEnd = true
	}

	result = parseStringLiteralContent(p, literal[1:endOffset])

	if missingEnd {
		p.reportSyntaxError("invalid end of string literal: missing '\"'")
	}

	return
}

// parseStringLiteralContent parses the string literalExpr contents, excluding start and end quotes
//
func parseStringLiteralContent(p *parser, s string) (result string) {

	var builder strings.Builder
	defer func() {
		result = builder.String()
	}()

	length := len(s)

	var r rune
	index := 0

	atEnd := index >= length

	advance := func() {
		if atEnd {
			r = lexer.EOF
			return
		}

		var width int
		r, width = utf8.DecodeRuneInString(s[index:])
		index += width

		atEnd = index >= length
	}

	for index < length {
		advance()

		if r != '\\' {
			builder.WriteRune(r)
			continue
		}

		if atEnd {
			p.reportSyntaxError("incomplete escape sequence: missing character after escape character")
			return
		}

		advance()

		switch r {
		case '0':
			builder.WriteByte(0)
		case 'n':
			builder.WriteByte('\n')
		case 'r':
			builder.WriteByte('\r')
		case 't':
			builder.WriteByte('\t')
		case '"':
			builder.WriteByte('"')
		case '\'':
			builder.WriteByte('\'')
		case '\\':
			builder.WriteByte('\\')
		case 'u':
			if atEnd {
				p.reportSyntaxError(
					"incomplete Unicode escape sequence: missing character '{' after escape character",
				)
				return
			}
			advance()
			if r != '{' {
				p.reportSyntaxError("invalid Unicode escape sequence: expected '{', got %q", r)
				continue
			}

			var r2 rune
			valid := true
			digitIndex := 0
			for ; !atEnd && digitIndex < 8; digitIndex++ {
				advance()
				if r == '}' {
					break
				}

				parsed := parseHex(r)

				if parsed < 0 {
					p.reportSyntaxError("invalid Unicode escape sequence: expected hex digit, got %q", r)
					valid = false
				} else {
					r2 = r2<<4 | parsed
				}
			}

			if digitIndex > 0 && valid {
				builder.WriteRune(r2)
			}

			if r != '}' {
				advance()
			}

			switch r {
			case '}':
				break
			case lexer.EOF:
				p.reportSyntaxError(
					"incomplete Unicode escape sequence: missing character '}' after escape character",
				)
			default:
				p.reportSyntaxError("incomplete Unicode escape sequence: expected '}', got %q", r)
			}

		default:
			// TODO: include index/column in error
			p.reportSyntaxError("invalid escape character: %q", r)
			// skip invalid escape character, don't write to result
		}
	}

	return
}

func parseHex(r rune) rune {
	switch {
	case '0' <= r && r <= '9':
		return r - '0'
	case 'a' <= r && r <= 'f':
		return r - 'a' + 10
	case 'A' <= r && r <= 'F':
		return r - 'A' + 10
	}

	return -1
}

func parseIntegerLiteral(p *parser, literal, text string, kind IntegerLiteralKind, tokenRange ast.Range) *ast.IntegerExpression {

	report := func(invalidKind InvalidNumberLiteralKind) {
		p.report(
			&InvalidIntegerLiteralError{
				IntegerLiteralKind:        kind,
				InvalidIntegerLiteralKind: invalidKind,
				// NOTE: not using text, because it has the base-prefix stripped
				Literal: literal,
				Range:   tokenRange,
			},
		)
	}

	// check literal has no leading underscore

	if strings.HasPrefix(text, "_") {
		report(InvalidNumberLiteralKindLeadingUnderscore)
	}

	// check literal has no trailing underscore
	if strings.HasSuffix(text, "_") {
		report(InvalidNumberLiteralKindTrailingUnderscore)
	}

	withoutUnderscores := strings.ReplaceAll(text, "_", "")

	var value *big.Int
	var base int

	if kind == IntegerLiteralKindUnknown {
		base = 1

		report(InvalidNumberLiteralKindUnknownPrefix)
	} else {
		base = kind.Base()

		if withoutUnderscores == "" {
			report(InvalidNumberLiteralKindMissingDigits)
		} else {
			var ok bool
			value, ok = new(big.Int).SetString(withoutUnderscores, base)
			if !ok {
				report(InvalidNumberLiteralKindUnknown)
			}
		}
	}

	if value == nil {
		value = new(big.Int)
	}

	return ast.NewIntegerExpression(p.memoryGauge, literal, value, base, tokenRange)
}

func parseFixedPointPart(gauge common.MemoryGauge, part string) (integer *big.Int, scale uint) {
	withoutUnderscores := strings.ReplaceAll(part, "_", "")

	common.UseMemory(
		gauge,
		common.NewBigIntMemoryUsage(
			common.OverEstimateBigIntFromString(withoutUnderscores),
		),
	)

	integer, _ = new(big.Int).SetString(withoutUnderscores, 10)
	if integer == nil {
		integer = new(big.Int)
	}
	scale = uint(len(withoutUnderscores))
	if scale == 0 {
		scale = 1
	}
	return integer, scale
}

func parseFixedPointLiteral(p *parser, literal string, tokenRange ast.Range) *ast.FixedPointExpression {
	parts := strings.Split(literal, ".")
	integer, _ := parseFixedPointPart(p.memoryGauge, parts[0])
	fractional, scale := parseFixedPointPart(p.memoryGauge, parts[1])

	return ast.NewFixedPointExpression(
		p.memoryGauge,
		literal,
		false,
		integer,
		fractional,
		scale,
		tokenRange,
	)
}
