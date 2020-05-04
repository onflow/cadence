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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

type infixFunc func(left, right ast.Expression) ast.Expression
type nullDenotationFunc func(lexer.Token) ast.Expression

type literal struct {
	tokenType      lexer.TokenType
	nullDenotation nullDenotationFunc
}

type infixOperator struct {
	tokenType        lexer.TokenType
	leftBindingPower int
	rightAssociative bool
	leftDenotation   infixFunc
}

var nullDenotations = map[lexer.TokenType]nullDenotationFunc{}

type leftDenotationFunc func(parser *parser, left ast.Expression) ast.Expression

var leftBindingPowers = map[lexer.TokenType]int{}
var leftDenotations = map[lexer.TokenType]leftDenotationFunc{}

func define(def interface{}) {
	switch def := def.(type) {
	case infixOperator:
		leftBindingPowers[def.tokenType] = def.leftBindingPower

		rightBindingPower := def.leftBindingPower
		if def.rightAssociative {
			rightBindingPower -= 1
		}

		leftDenotations[def.tokenType] = func(parser *parser, left ast.Expression) ast.Expression {
			right := parseExpression(parser, rightBindingPower)
			return def.leftDenotation(left, right)
		}

	case literal:
		nullDenotations[def.tokenType] = def.nullDenotation
		leftBindingPowers[def.tokenType] = 0

	default:
		panic(errors.NewUnreachableError())
	}
}

func init() {

	define(infixOperator{
		tokenType:        lexer.TokenOperatorPlus,
		leftBindingPower: 10,
		leftDenotation: func(left, right ast.Expression) ast.Expression {
			return &ast.BinaryExpression{
				Operation: ast.OperationPlus,
				Left:      left,
				Right:     right,
			}
		},
	})

	define(infixOperator{
		tokenType:        lexer.TokenOperatorMinus,
		leftBindingPower: 10,
		leftDenotation: func(left, right ast.Expression) ast.Expression {
			return &ast.BinaryExpression{
				Operation: ast.OperationMinus,
				Left:      left,
				Right:     right,
			}
		},
	})

	define(infixOperator{
		tokenType:        lexer.TokenOperatorMul,
		leftBindingPower: 20,
		leftDenotation: func(left, right ast.Expression) ast.Expression {
			return &ast.BinaryExpression{
				Operation: ast.OperationMul,
				Left:      left,
				Right:     right,
			}
		},
	})

	define(infixOperator{
		tokenType:        lexer.TokenOperatorDiv,
		leftBindingPower: 20,
		leftDenotation: func(left, right ast.Expression) ast.Expression {
			return &ast.BinaryExpression{
				Operation: ast.OperationDiv,
				Left:      left,
				Right:     right,
			}
		},
	})

	define(literal{
		tokenType: lexer.TokenNumber,
		nullDenotation: func(token lexer.Token) ast.Expression {
			value, _ := new(big.Int).SetString(token.Value.(string), 10)
			return &ast.IntegerExpression{
				Value: value,
				Base:  10,
			}
		},
	})

	leftBindingPowers[lexer.TokenEOF] = 0
}

func parseExpression(p *parser, rightBindingPower int) ast.Expression {
	p.skipZeroOrOne(lexer.TokenSpace)
	current := p.current
	p.next()
	p.skipZeroOrOne(lexer.TokenSpace)
	next := p.current

	left := applyNullDenotation(current)
	if p.atEnd {
		return left
	}

	for rightBindingPower < leftBindingPower(next.Type) {
		current = next
		p.next()
		p.skipZeroOrOne(lexer.TokenSpace)

		next = p.current
		left = applyLeftDenotation(p, current.Type, left)
		if p.atEnd {
			return left
		}
	}

	return left
}

func applyNullDenotation(token lexer.Token) ast.Expression {
	nullDenotation, ok := nullDenotations[token.Type]
	if !ok {
		panic(fmt.Errorf("missing null denotation for token type: %v", token.Type))
	}
	return nullDenotation(token)
}

func leftBindingPower(tokenType lexer.TokenType) int {
	result, ok := leftBindingPowers[tokenType]
	if !ok {
		panic(fmt.Errorf("missing left binding power for token type: %v", tokenType))
	}
	return result
}

func applyLeftDenotation(parser *parser, tokenType lexer.TokenType, left ast.Expression) ast.Expression {
	leftDenotation, ok := leftDenotations[tokenType]
	if !ok {
		panic(fmt.Errorf("missing left denotation for token type: %v", tokenType))
	}
	return leftDenotation(parser, left)
}
