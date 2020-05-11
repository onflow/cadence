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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

type typeNullDenotationFunc func(parser *parser, token lexer.Token) ast.Type

var typeNullDenotations = map[lexer.TokenType]typeNullDenotationFunc{}

type typeLeftDenotationFunc func(parser *parser, token lexer.Token, left ast.Type) ast.Type

var typeLeftBindingPowers = map[lexer.TokenType]int{}
var typeLeftDenotations = map[lexer.TokenType]typeLeftDenotationFunc{}

func setTypeNullDenotation(tokenType lexer.TokenType, nullDenotation typeNullDenotationFunc) {
	current := typeNullDenotations[tokenType]
	if current != nil {
		panic(fmt.Errorf(
			"type null denotation for token type %s exists",
			tokenType,
		))
	}
	typeNullDenotations[tokenType] = nullDenotation
}

func setTypeLeftBindingPower(tokenType lexer.TokenType, power int) {
	current := typeLeftBindingPowers[tokenType]
	if current > power {
		return
	}
	typeLeftBindingPowers[tokenType] = power
}

func setTypeLeftDenotation(tokenType lexer.TokenType, leftDenotation typeLeftDenotationFunc) {
	current := typeLeftDenotations[tokenType]
	if current != nil {
		panic(fmt.Errorf(
			"type left denotation for token type %s exists",
			tokenType,
		))
	}
	typeLeftDenotations[tokenType] = leftDenotation
}

type prefixTypeFunc func(right ast.Type, tokenRange ast.Range) ast.Type
type postfixTypeFunc func(left ast.Type, tokenRange ast.Range) ast.Type

type literalType struct {
	tokenType      lexer.TokenType
	nullDenotation typeNullDenotationFunc
}

type prefixType struct {
	tokenType      lexer.TokenType
	bindingPower   int
	nullDenotation prefixTypeFunc
}

type postfixType struct {
	tokenType      lexer.TokenType
	bindingPower   int
	nullDenotation postfixTypeFunc
}

func defineType(def interface{}) {
	switch def := def.(type) {
	case prefixType:
		tokenType := def.tokenType
		setTypeNullDenotation(
			tokenType,
			func(parser *parser, token lexer.Token) ast.Type {
				right := parseType(parser, def.bindingPower)
				return def.nullDenotation(right, token.Range)
			},
		)
	case postfixType:
		tokenType := def.tokenType
		setTypeLeftBindingPower(tokenType, def.bindingPower)
		setTypeLeftDenotation(
			tokenType,
			func(p *parser, token lexer.Token, left ast.Type) ast.Type {
				return def.nullDenotation(left, token.Range)
			},
		)
	case literalType:
		tokenType := def.tokenType
		setTypeNullDenotation(tokenType, def.nullDenotation)
	default:
		panic(errors.NewUnreachableError())
	}
}

func init() {
	defineNominalType()
	defineArrayType()
	defineOptionalType()
	defineReferenceType()
}

func defineNominalType() {
	defineType(literalType{
		tokenType: lexer.TokenIdentifier,
		nullDenotation: func(p *parser, token lexer.Token) ast.Type {

			var nestedIdentifiers []ast.Identifier

			for p.current.Is(lexer.TokenDot) {
				p.next()

				nestedToken := p.current
				p.next()

				if !nestedToken.Is(lexer.TokenIdentifier) {
					panic(fmt.Errorf("expected nested type identifier after '.', got %v", nestedToken.Type))
				}

				nestedIdentifier := tokenToIdentifier(nestedToken)

				nestedIdentifiers = append(
					nestedIdentifiers,
					nestedIdentifier,
				)

			}

			return &ast.NominalType{
				Identifier:        tokenToIdentifier(token),
				NestedIdentifiers: nestedIdentifiers,
			}
		},
	})
}

func defineArrayType() {
	setTypeNullDenotation(
		lexer.TokenBracketOpen,
		func(p *parser, startToken lexer.Token) ast.Type {

			elementType := parseType(p, lowestBindingPower)

			p.skipSpaceAndComments(true)

			var size *ast.IntegerExpression

			if p.current.Is(lexer.TokenSemicolon) {
				p.next()

				p.skipSpaceAndComments(true)

				if !p.current.Is(lexer.TokenNumber) {
					panic(fmt.Errorf("expected size for constant sized type, got %s", p.current.Type))
				}

				numberExpression := parseNumber(p.current)
				p.next()

				integerExpression, ok := numberExpression.(*ast.IntegerExpression)
				if !ok {
					p.report(fmt.Errorf("expected integer size for constant sized type, got %s", numberExpression))
				} else {
					size = integerExpression
				}
			}

			p.skipSpaceAndComments(true)

			endToken := p.mustOne(lexer.TokenBracketClose)

			typeRange := ast.Range{
				StartPos: startToken.StartPos,
				EndPos:   endToken.EndPos,
			}

			if size != nil {
				return &ast.ConstantSizedType{
					Type:  elementType,
					Size:  size,
					Range: typeRange,
				}
			} else {
				return &ast.VariableSizedType{
					Type:  elementType,
					Range: typeRange,
				}
			}
		},
	)
}

func defineOptionalType() {
	defineType(postfixType{
		tokenType:    lexer.TokenQuestionMark,
		bindingPower: 10,
		nullDenotation: func(left ast.Type, tokenRange ast.Range) ast.Type {
			return &ast.OptionalType{
				Type:   left,
				EndPos: tokenRange.EndPos,
			}
		},
	})
}

func defineReferenceType() {
	defineType(prefixType{
		tokenType:    lexer.TokenAmpersand,
		bindingPower: 20,
		nullDenotation: func(right ast.Type, tokenRange ast.Range) ast.Type {
			return &ast.ReferenceType{
				Authorized: false,
				Type:       right,
				StartPos:   tokenRange.StartPos,
			}
		},
	})
}

func parseType(p *parser, rightBindingPower int) ast.Type {
	p.skipSpaceAndComments(true)
	t := p.current
	p.next()
	p.skipSpaceAndComments(true)

	left := applyTypeNullDenotation(p, t)
	if left == nil {
		return nil
	}

	p.skipSpaceAndComments(true)

	for rightBindingPower < typeLeftBindingPowers[p.current.Type] {
		t = p.current
		p.next()
		p.skipSpaceAndComments(true)

		left = applyTypeLeftDenotation(p, t, left)
	}

	return left
}

func parseTypeAnnotation(p *parser) *ast.TypeAnnotation {
	startPos := p.current.StartPos

	isResource := false
	if p.current.Is(lexer.TokenAt) {
		p.next()
		isResource = true
	}

	ty := parseType(p, lowestBindingPower)

	return &ast.TypeAnnotation{
		IsResource: isResource,
		Type:       ty,
		StartPos:   startPos,
	}
}

func applyTypeNullDenotation(p *parser, token lexer.Token) ast.Type {
	tokenType := token.Type
	nullDenotation, ok := typeNullDenotations[tokenType]
	if !ok {
		return nil
	}
	return nullDenotation(p, token)
}

func applyTypeLeftDenotation(p *parser, token lexer.Token, left ast.Type) ast.Type {
	leftDenotation, ok := typeLeftDenotations[token.Type]
	if !ok {
		panic(fmt.Errorf("missing left denotation for token type: %v", token.Type))
	}
	return leftDenotation(p, token, left)
}
