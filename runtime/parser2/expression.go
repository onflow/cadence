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

const lowestBindingPower = 0

type infixFunc func(left, right ast.Expression) ast.Expression
type prefixFunc func(right ast.Expression, tokenRange ast.Range) ast.Expression
type nullDenotationFunc func(parser *parser, token lexer.Token) ast.Expression

type literal struct {
	tokenType      lexer.TokenType
	nullDenotation nullDenotationFunc
}

type infix struct {
	tokenType        lexer.TokenType
	leftBindingPower int
	rightAssociative bool
	leftDenotation   infixFunc
}

type binary struct {
	tokenType        lexer.TokenType
	leftBindingPower int
	rightAssociative bool
	operation        ast.Operation
}

type prefix struct {
	tokenType      lexer.TokenType
	bindingPower   int
	nullDenotation prefixFunc
}

type unary struct {
	tokenType    lexer.TokenType
	bindingPower int
	operation    ast.Operation
}

var nullDenotations = map[lexer.TokenType]nullDenotationFunc{}

type leftDenotationFunc func(parser *parser, left ast.Expression) ast.Expression

var leftBindingPowers = map[lexer.TokenType]int{}
var leftDenotations = map[lexer.TokenType]leftDenotationFunc{}

func define(def interface{}) {
	switch def := def.(type) {
	case infix:
		tokenType := def.tokenType

		setLeftBindingPower(tokenType, def.leftBindingPower)

		rightBindingPower := def.leftBindingPower
		if def.rightAssociative {
			rightBindingPower -= 1
		}

		setLeftDenotation(
			tokenType,
			func(parser *parser, left ast.Expression) ast.Expression {
				right := parseExpression(parser, rightBindingPower)
				return def.leftDenotation(left, right)
			},
		)

	case binary:
		define(infix{
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

	case literal:
		tokenType := def.tokenType
		setLeftBindingPower(tokenType, lowestBindingPower)
		setNullDenotation(tokenType, def.nullDenotation)

	case prefix:
		tokenType := def.tokenType
		setLeftBindingPower(tokenType, lowestBindingPower)
		setNullDenotation(
			tokenType,
			func(parser *parser, token lexer.Token) ast.Expression {
				right := parseExpression(parser, def.bindingPower)
				return def.nullDenotation(right, token.Range)
			},
		)

	case unary:
		define(prefix{
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

func setNullDenotation(tokenType lexer.TokenType, nullDenotation nullDenotationFunc) {
	current := nullDenotations[tokenType]
	if current != nil {
		panic(fmt.Errorf(
			"null denotation for token type %s exists",
			tokenType,
		))
	}
	nullDenotations[tokenType] = nullDenotation
}

func setLeftBindingPower(tokenType lexer.TokenType, power int) {
	current := leftBindingPowers[tokenType]
	if current > power {
		return
	}
	leftBindingPowers[tokenType] = power
}

func setLeftDenotation(tokenType lexer.TokenType, leftDenotation leftDenotationFunc) {
	current := leftDenotations[tokenType]
	if current != nil {
		panic(fmt.Errorf(
			"left denotation for token type %s exists",
			tokenType,
		))
	}
	leftDenotations[tokenType] = leftDenotation
}

func init() {

	define(binary{
		tokenType:        lexer.TokenLess,
		leftBindingPower: 60,
		operation:        ast.OperationLess,
	})

	define(binary{
		tokenType:        lexer.TokenGreater,
		leftBindingPower: 60,
		operation:        ast.OperationGreater,
	})

	define(binary{
		tokenType:        lexer.TokenNilCoalesce,
		leftBindingPower: 100,
		operation:        ast.OperationNilCoalesce,
		rightAssociative: true,
	})

	define(binary{
		tokenType:        lexer.TokenPlus,
		leftBindingPower: 110,
		operation:        ast.OperationPlus,
	})

	define(binary{
		tokenType:        lexer.TokenMinus,
		leftBindingPower: 110,
		operation:        ast.OperationMinus,
	})

	define(binary{
		tokenType:        lexer.TokenStar,
		leftBindingPower: 120,
		operation:        ast.OperationMul,
	})

	define(binary{
		tokenType:        lexer.TokenSlash,
		leftBindingPower: 120,
		operation:        ast.OperationDiv,
	})

	define(literal{
		tokenType: lexer.TokenNumber,
		nullDenotation: func(_ *parser, token lexer.Token) ast.Expression {
			value, _ := new(big.Int).SetString(token.Value.(string), 10)
			return &ast.IntegerExpression{
				Value: value,
				Base:  10,
				Range: token.Range,
			}
		},
	})

	define(literal{
		tokenType: lexer.TokenIdentifier,
		nullDenotation: func(_ *parser, token lexer.Token) ast.Expression {
			switch token.Value {
			case "true":
				return &ast.BoolExpression{
					Value: true,
					Range: token.Range,
				}

			case "false":
				return &ast.BoolExpression{
					Value: false,
					Range: token.Range,
				}

			default:
				return &ast.IdentifierExpression{
					Identifier: tokenToIdentifier(token),
				}
			}
		},
	})

	define(literal{
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

	define(unary{
		tokenType:    lexer.TokenMinus,
		bindingPower: 130,
		operation:    ast.OperationMinus,
	})

	define(unary{
		tokenType:    lexer.TokenPlus,
		bindingPower: 130,
		operation:    ast.OperationPlus,
	})

	define(unary{
		tokenType:    lexer.TokenLeftArrow,
		bindingPower: 130,
		operation:    ast.OperationMove,
	})

	defineNestedExpression()
	defineArrayExpression()
	defineDictionaryExpression()
	definePathExpression()
	defineConditionalExpression()

	leftBindingPowers[lexer.TokenComma] = lowestBindingPower

	leftBindingPowers[lexer.TokenColon] = lowestBindingPower

	leftBindingPowers[lexer.TokenEOF] = lowestBindingPower
}

func defineNestedExpression() {
	leftBindingPowers[lexer.TokenParenOpen] = 150
	leftBindingPowers[lexer.TokenParenClose] = lowestBindingPower
	nullDenotations[lexer.TokenParenOpen] = func(p *parser, token lexer.Token) ast.Expression {
		expression := parseExpression(p, lowestBindingPower)
		p.mustOne(lexer.TokenParenClose)
		return expression
	}
}

func defineArrayExpression() {
	leftBindingPowers[lexer.TokenBracketOpen] = 150
	leftBindingPowers[lexer.TokenBracketClose] = lowestBindingPower
	nullDenotations[lexer.TokenBracketOpen] = func(p *parser, startToken lexer.Token) ast.Expression {
		var values []ast.Expression
		for p.current.Type != lexer.TokenBracketClose {
			value := parseExpression(p, lowestBindingPower)
			values = append(values, value)
			if p.current.Type != lexer.TokenComma {
				break
			}
			p.mustOne(lexer.TokenComma)
		}
		endToken := p.mustOne(lexer.TokenBracketClose)
		return &ast.ArrayExpression{
			Values: values,
			Range: ast.Range{
				StartPos: startToken.Range.StartPos,
				EndPos:   endToken.Range.EndPos,
			},
		}
	}
}

func defineDictionaryExpression() {
	leftBindingPowers[lexer.TokenBraceOpen] = 150
	leftBindingPowers[lexer.TokenBraceClose] = lowestBindingPower
	nullDenotations[lexer.TokenBraceOpen] = func(p *parser, startToken lexer.Token) ast.Expression {
		var entries []ast.Entry
		for p.current.Type != lexer.TokenBraceClose {
			key := parseExpression(p, lowestBindingPower)
			p.mustOne(lexer.TokenColon)
			value := parseExpression(p, lowestBindingPower)
			entries = append(entries, ast.Entry{
				Key:   key,
				Value: value,
			})
			if p.current.Type != lexer.TokenComma {
				break
			}
			p.mustOne(lexer.TokenComma)
		}
		endToken := p.mustOne(lexer.TokenBraceClose)
		return &ast.DictionaryExpression{
			Entries: entries,
			Range: ast.Range{
				StartPos: startToken.Range.StartPos,
				EndPos:   endToken.Range.EndPos,
			},
		}
	}
}

func defineConditionalExpression() {
	leftBindingPowers[lexer.TokenQuestionMark] = 20
	leftDenotations[lexer.TokenQuestionMark] = func(p *parser, left ast.Expression) ast.Expression {
		testExpression := left
		thenExpression := parseExpression(p, lowestBindingPower)
		p.mustOne(lexer.TokenColon)
		elseExpression := parseExpression(p, lowestBindingPower)
		return &ast.ConditionalExpression{
			Test: testExpression,
			Then: thenExpression,
			Else: elseExpression,
		}
	}
}

func definePathExpression() {
	leftBindingPowers[lexer.TokenSlash] = 150
	nullDenotations[lexer.TokenSlash] = func(p *parser, token lexer.Token) ast.Expression {
		domain := mustIdentifier(p)
		p.mustOne(lexer.TokenSlash)
		identifier := mustIdentifier(p)
		return &ast.PathExpression{
			Domain:     domain,
			Identifier: identifier,
			StartPos:   token.Range.StartPos,
		}
	}
}

func mustIdentifier(p *parser) ast.Identifier {
	identifier := p.mustOne(lexer.TokenIdentifier)
	return tokenToIdentifier(identifier)
}

func tokenToIdentifier(identifier lexer.Token) ast.Identifier {
	return ast.Identifier{
		Identifier: identifier.Value.(string),
		Pos:        identifier.Range.StartPos,
	}
}

func parseExpression(p *parser, rightBindingPower int) ast.Expression {
	p.skipZeroOrOne(lexer.TokenSpace)
	t := p.current
	p.next()

	left := applyNullDenotation(p, t)
	p.skipZeroOrOne(lexer.TokenSpace)

	for rightBindingPower < leftBindingPower(p.current.Type) {
		t = p.current
		p.next()
		p.skipZeroOrOne(lexer.TokenSpace)

		left = applyLeftDenotation(p, t.Type, left)
	}

	return left
}

func applyNullDenotation(p *parser, token lexer.Token) ast.Expression {
	tokenType := token.Type
	nullDenotation, ok := nullDenotations[tokenType]
	if !ok {
		panic(fmt.Errorf("missing null denotation for token type: %v", tokenType))
	}
	return nullDenotation(p, token)
}

func leftBindingPower(tokenType lexer.TokenType) int {
	result, ok := leftBindingPowers[tokenType]
	if !ok {
		panic(fmt.Errorf("missing left binding power for token type: %v", tokenType))
	}
	return result
}

func applyLeftDenotation(p *parser, tokenType lexer.TokenType, left ast.Expression) ast.Expression {
	leftDenotation, ok := leftDenotations[tokenType]
	if !ok {
		panic(fmt.Errorf("missing left denotation for token type: %v", tokenType))
	}
	return leftDenotation(p, left)
}

// parseStringLiteral parses a whole string literal, including start and end quotes
//
func parseStringLiteral(s string) (result string, errs []error) {
	report := func(err error) {
		errs = append(errs, err)
	}

	l := len(s)
	if l == 0 {
		report(fmt.Errorf("missing start of string literal: expected '\"'"))
		return
	}

	if l >= 1 {
		first := s[0]
		if first != '"' {
			report(fmt.Errorf("invalid start of string literal: expected '\"', got %q", first))
		}
	}

	missingEnd := false
	endOffset := l
	if l >= 2 {
		last := s[l-1]
		if last != '"' {
			missingEnd = true
		} else {
			endOffset = l - 1
		}
	} else {
		missingEnd = true
	}

	var innerErrs []error
	result, innerErrs = parseStringLiteralContent(s[1:endOffset])
	errs = append(errs, innerErrs...)

	if missingEnd {
		report(fmt.Errorf("invalid end of string literal: missing '\"'"))
	}

	return
}

// parseStringLiteralContent parses the string literal contents, excluding start and end quotes
//
func parseStringLiteralContent(s string) (result string, errs []error) {

	var builder strings.Builder
	defer func() {
		result = builder.String()
	}()

	report := func(err error) {
		errs = append(errs, err)
	}

	l := len(s)

	var r rune
	i := 0

	atEnd := i >= l

	advance := func() {
		if atEnd {
			r = lexer.EOF
			return
		}

		var w int
		r, w = utf8.DecodeRuneInString(s[i:])
		i += w

		atEnd = i >= l
	}

	for i < l {
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
			j := 0
			for ; !atEnd && j < 8; j++ {
				advance()
				if r == '}' {
					break
				}

				d := parseHex(r)

				if d < 0 {
					report(fmt.Errorf("invalid Unicode escape sequence: expected hex digit, got %q", r))
					valid = false
				} else {
					r2 = r2<<4 | d
				}
			}

			if j > 0 && valid {
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
