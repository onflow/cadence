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
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

type infixExprFunc func(left, right ast.Expression) ast.Expression
type prefixExprFunc func(right ast.Expression, tokenRange ast.Range) ast.Expression
type exprNullDenotationFunc func(parser *parser, token lexer.Token) ast.Expression

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

var exprNullDenotations = map[lexer.TokenType]exprNullDenotationFunc{}

type exprLeftDenotationFunc func(parser *parser, token lexer.Token, left ast.Expression) ast.Expression

var exprLeftBindingPowers = map[lexer.TokenType]int{}
var exprLeftDenotations = map[lexer.TokenType]exprLeftDenotationFunc{}

func defineExpr(def interface{}) {
	switch def := def.(type) {
	case infixExpr:
		tokenType := def.tokenType

		setExprLeftBindingPower(tokenType, def.leftBindingPower)

		rightBindingPower := def.leftBindingPower
		if def.rightAssociative {
			rightBindingPower -= 1
		}

		setExprLeftDenotation(
			tokenType,
			func(parser *parser, _ lexer.Token, left ast.Expression) ast.Expression {
				right := parseExpression(parser, rightBindingPower)
				return def.leftDenotation(left, right)
			},
		)

	case binaryExpr:
		defineExpr(infixExpr{
			tokenType:        def.tokenType,
			leftBindingPower: def.leftBindingPower,
			rightAssociative: def.rightAssociative,
			leftDenotation: func(left, right ast.Expression) ast.Expression {
				return &ast.BinaryExpression{
					Operation: def.operation,
					Left:      left,
					Right:     right,
				}
			},
		})

	case literalExpr:
		tokenType := def.tokenType
		setExprNullDenotation(tokenType, def.nullDenotation)

	case prefixExpr:
		tokenType := def.tokenType
		setExprNullDenotation(
			tokenType,
			func(parser *parser, token lexer.Token) ast.Expression {
				right := parseExpression(parser, def.bindingPower)
				return def.nullDenotation(right, token.Range)
			},
		)

	case unaryExpr:
		defineExpr(prefixExpr{
			tokenType:    def.tokenType,
			bindingPower: def.bindingPower,
			nullDenotation: func(right ast.Expression, tokenRange ast.Range) ast.Expression {
				return &ast.UnaryExpression{
					Operation:  def.operation,
					Expression: right,
					StartPos:   tokenRange.StartPos,
				}
			},
		})

	default:
		panic(errors.NewUnreachableError())
	}
}

func setExprNullDenotation(tokenType lexer.TokenType, nullDenotation exprNullDenotationFunc) {
	current := exprNullDenotations[tokenType]
	if current != nil {
		panic(fmt.Errorf(
			"expression null denotation for token %q already exists",
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

func setExprLeftDenotation(tokenType lexer.TokenType, leftDenotation exprLeftDenotationFunc) {
	current := exprLeftDenotations[tokenType]
	if current != nil {
		panic(fmt.Errorf(
			"expression left denotation for token %q already exists",
			tokenType,
		))
	}
	exprLeftDenotations[tokenType] = leftDenotation
}

func init() {

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenVerticalBarVerticalBar,
		leftBindingPower: 30,
		rightAssociative: true,
		operation:        ast.OperationOr,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenAmpersandAmpersand,
		leftBindingPower: 40,
		rightAssociative: true,
		operation:        ast.OperationAnd,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenLess,
		leftBindingPower: 50,
		operation:        ast.OperationLess,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenLessEqual,
		leftBindingPower: 50,
		operation:        ast.OperationLessEqual,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenGreater,
		leftBindingPower: 50,
		operation:        ast.OperationGreater,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenGreaterEqual,
		leftBindingPower: 50,
		operation:        ast.OperationGreaterEqual,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenEqualEqual,
		leftBindingPower: 50,
		operation:        ast.OperationEqual,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenNotEqual,
		leftBindingPower: 50,
		operation:        ast.OperationNotEqual,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenNilCoalesce,
		leftBindingPower: 60,
		operation:        ast.OperationNilCoalesce,
		rightAssociative: true,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenVerticalBar,
		leftBindingPower: 70,
		operation:        ast.OperationBitwiseOr,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenCaret,
		leftBindingPower: 80,
		operation:        ast.OperationBitwiseXor,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenAmpersand,
		leftBindingPower: 90,
		operation:        ast.OperationBitwiseAnd,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenLessLess,
		leftBindingPower: 100,
		operation:        ast.OperationBitwiseLeftShift,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenGreaterGreater,
		leftBindingPower: 100,
		operation:        ast.OperationBitwiseRightShift,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenPlus,
		leftBindingPower: 110,
		operation:        ast.OperationPlus,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenMinus,
		leftBindingPower: 110,
		operation:        ast.OperationMinus,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenStar,
		leftBindingPower: 120,
		operation:        ast.OperationMul,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenSlash,
		leftBindingPower: 120,
		operation:        ast.OperationDiv,
	})

	defineExpr(binaryExpr{
		tokenType:        lexer.TokenPercent,
		leftBindingPower: 120,
		operation:        ast.OperationMod,
	})

	defineExpr(literalExpr{
		tokenType: lexer.TokenNumber,
		nullDenotation: func(_ *parser, token lexer.Token) ast.Expression {
			return parseNumber(token)
		},
	})

	defineExpr(literalExpr{
		tokenType: lexer.TokenIdentifier,
		nullDenotation: func(_ *parser, token lexer.Token) ast.Expression {
			switch token.Value {
			case keywordTrue:
				return &ast.BoolExpression{
					Value: true,
					Range: token.Range,
				}

			case keywordFalse:
				return &ast.BoolExpression{
					Value: false,
					Range: token.Range,
				}

			case keywordNil:
				return &ast.NilExpression{}

			default:
				return &ast.IdentifierExpression{
					Identifier: tokenToIdentifier(token),
				}
			}
		},
	})

	defineExpr(literalExpr{
		tokenType: lexer.TokenString,
		nullDenotation: func(p *parser, token lexer.Token) ast.Expression {
			parsedString, errs := parseStringLiteral(token.Value.(string))
			p.report(errs...)
			return &ast.StringExpression{
				Value: parsedString,
				Range: token.Range,
			}
		},
	})

	defineExpr(unaryExpr{
		tokenType:    lexer.TokenMinus,
		bindingPower: 130,
		operation:    ast.OperationMinus,
	})

	defineExpr(unaryExpr{
		tokenType:    lexer.TokenPlus,
		bindingPower: 130,
		operation:    ast.OperationPlus,
	})

	defineExpr(unaryExpr{
		tokenType:    lexer.TokenNot,
		bindingPower: 130,
		operation:    ast.OperationNegate,
	})

	defineExpr(unaryExpr{
		tokenType:    lexer.TokenLeftArrow,
		bindingPower: 130,
		operation:    ast.OperationMove,
	})

	defineNestedExpression()
	defineArrayExpression()
	defineDictionaryExpression()
	definePathExpression()
	defineConditionalExpression()
}

func parseNumber(token lexer.Token) ast.Expression {
	// TODO: extend
	value, _ := new(big.Int).SetString(token.Value.(string), 10)
	return &ast.IntegerExpression{
		Value: value,
		Base:  10,
		Range: token.Range,
	}
}

func defineNestedExpression() {
	setExprLeftBindingPower(lexer.TokenParenOpen, 150)
	setExprNullDenotation(
		lexer.TokenParenOpen,
		func(p *parser, token lexer.Token) ast.Expression {
			expression := parseExpression(p, lowestBindingPower)
			p.mustOne(lexer.TokenParenClose)
			return expression
		},
	)
}

func defineArrayExpression() {
	setExprLeftBindingPower(lexer.TokenBracketOpen, 150)
	setExprNullDenotation(
		lexer.TokenBracketOpen,
		func(p *parser, startToken lexer.Token) ast.Expression {
			var values []ast.Expression
			for !p.current.Is(lexer.TokenBracketClose) {
				value := parseExpression(p, lowestBindingPower)
				values = append(values, value)
				if !p.current.Is(lexer.TokenComma) {
					break
				}
				p.mustOne(lexer.TokenComma)
			}
			endToken := p.mustOne(lexer.TokenBracketClose)
			return &ast.ArrayExpression{
				Values: values,
				Range: ast.Range{
					StartPos: startToken.StartPos,
					EndPos:   endToken.EndPos,
				},
			}
		},
	)
}

func defineDictionaryExpression() {
	setExprNullDenotation(
		lexer.TokenBraceOpen,
		func(p *parser, startToken lexer.Token) ast.Expression {
			var entries []ast.Entry
			for !p.current.Is(lexer.TokenBraceClose) {
				key := parseExpression(p, lowestBindingPower)
				p.mustOne(lexer.TokenColon)
				value := parseExpression(p, lowestBindingPower)
				entries = append(entries, ast.Entry{
					Key:   key,
					Value: value,
				})
				if !p.current.Is(lexer.TokenComma) {
					break
				}
				p.mustOne(lexer.TokenComma)
			}
			endToken := p.mustOne(lexer.TokenBraceClose)
			return &ast.DictionaryExpression{
				Entries: entries,
				Range: ast.Range{
					StartPos: startToken.StartPos,
					EndPos:   endToken.EndPos,
				},
			}
		},
	)
}

func defineConditionalExpression() {
	setExprLeftBindingPower(lexer.TokenQuestionMark, 20)
	setExprLeftDenotation(
		lexer.TokenQuestionMark,
		func(p *parser, _ lexer.Token, left ast.Expression) ast.Expression {
			testExpression := left
			thenExpression := parseExpression(p, lowestBindingPower)
			p.mustOne(lexer.TokenColon)
			elseExpression := parseExpression(p, lowestBindingPower)
			return &ast.ConditionalExpression{
				Test: testExpression,
				Then: thenExpression,
				Else: elseExpression,
			}
		},
	)
}

func definePathExpression() {
	setExprLeftBindingPower(lexer.TokenSlash, 150)
	setExprNullDenotation(
		lexer.TokenSlash,
		func(p *parser, token lexer.Token) ast.Expression {
			domain := mustIdentifier(p)
			p.mustOne(lexer.TokenSlash)
			identifier := mustIdentifier(p)
			return &ast.PathExpression{
				Domain:     domain,
				Identifier: identifier,
				StartPos:   token.StartPos,
			}
		},
	)
}

func parseExpression(p *parser, rightBindingPower int) ast.Expression {
	p.skipSpaceAndComments(true)
	t := p.current
	p.next()
	p.skipSpaceAndComments(true)

	left := applyExprNullDenotation(p, t)
	if left == nil {
		return nil
	}

	p.skipSpaceAndComments(true)

	for rightBindingPower < exprLeftBindingPowers[p.current.Type] {
		t = p.current
		p.next()
		p.skipSpaceAndComments(true)

		left = applyExprLeftDenotation(p, t, left)
	}

	return left
}

func applyExprNullDenotation(p *parser, token lexer.Token) ast.Expression {
	tokenType := token.Type
	nullDenotation, ok := exprNullDenotations[tokenType]
	if !ok {
		return nil
	}
	return nullDenotation(p, token)
}

func applyExprLeftDenotation(p *parser, token lexer.Token, left ast.Expression) ast.Expression {
	leftDenotation, ok := exprLeftDenotations[token.Type]
	if !ok {
		panic(fmt.Errorf("missing left denotation for token %q", token.Type))
	}
	return leftDenotation(p, token, left)
}

// parseStringLiteral parses a whole string literal, including start and end quotes
//
func parseStringLiteral(literal string) (result string, errs []error) {
	report := func(err error) {
		errs = append(errs, err)
	}

	length := len(literal)
	if length == 0 {
		report(fmt.Errorf("missing start of string literal: expected '\"'"))
		return
	}

	if length >= 1 {
		first := literal[0]
		if first != '"' {
			report(fmt.Errorf("invalid start of string literal: expected '\"', got %q", first))
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

	var innerErrs []error
	result, innerErrs = parseStringLiteralContent(literal[1:endOffset])
	errs = append(errs, innerErrs...)

	if missingEnd {
		report(fmt.Errorf("invalid end of string literal: missing '\"'"))
	}

	return
}

// parseStringLiteralContent parses the string literalExpr contents, excluding start and end quotes
//
func parseStringLiteralContent(s string) (result string, errs []error) {

	var builder strings.Builder
	defer func() {
		result = builder.String()
	}()

	report := func(err error) {
		errs = append(errs, err)
	}

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
			report(fmt.Errorf("incomplete escape sequence: missing character after escape character"))
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
				report(fmt.Errorf(
					"incomplete Unicode escape sequence: missing character '{' after escape character",
				))
				return
			}
			advance()
			if r != '{' {
				report(fmt.Errorf("invalid Unicode escape sequence: expected '{', got %q", r))
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
					report(fmt.Errorf("invalid Unicode escape sequence: expected hex digit, got %q", r))
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
				report(fmt.Errorf(
					"incomplete Unicode escape sequence: missing character '}' after escape character",
				))
			default:
				report(fmt.Errorf("incomplete Unicode escape sequence: expected '}', got %q", r))
			}

		default:
			// TODO: include index/column in error
			report(fmt.Errorf("invalid escape character: %q", r))
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
