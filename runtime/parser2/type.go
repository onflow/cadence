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
			"type null denotation for token %q already exists",
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
			"type left denotation for token %q already exists",
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
	leftDenotation postfixTypeFunc
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
				return def.leftDenotation(left, token.Range)
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
	defineRestrictedOrDictionaryType()
}

func defineNominalType() {
	defineType(literalType{
		tokenType: lexer.TokenIdentifier,
		nullDenotation: func(p *parser, token lexer.Token) ast.Type {
			return parseNominalTypeRemainder(p, token)
		},
	})
}

func parseNominalTypeRemainder(p *parser, token lexer.Token) *ast.NominalType {
	var nestedIdentifiers []ast.Identifier

	for p.current.Is(lexer.TokenDot) {
		p.next()

		nestedToken := p.current
		p.next()

		if !nestedToken.Is(lexer.TokenIdentifier) {
			panic(fmt.Errorf(
				"expected identifier after %q, got %q",
				lexer.TokenDot,
				nestedToken.Type,
			))
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
					panic(fmt.Errorf(
						"expected size for constant sized type, got %q",
						p.current.Type,
					))
				}

				numberExpression := parseNumber(p.current)
				p.next()

				integerExpression, ok := numberExpression.(*ast.IntegerExpression)
				if !ok {
					p.report(fmt.Errorf(
						"expected integer size for constant sized type, got %q",
						numberExpression,
					))
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
	const bindingPower = 10

	defineType(postfixType{
		tokenType:    lexer.TokenQuestionMark,
		bindingPower: bindingPower,
		leftDenotation: func(left ast.Type, tokenRange ast.Range) ast.Type {
			return &ast.OptionalType{
				Type:   left,
				EndPos: tokenRange.EndPos,
			}
		},
	})

	defineType(postfixType{
		tokenType:    lexer.TokenDoubleQuestionMark,
		bindingPower: bindingPower,
		leftDenotation: func(left ast.Type, tokenRange ast.Range) ast.Type {
			return &ast.OptionalType{
				Type: &ast.OptionalType{
					Type:   left,
					EndPos: tokenRange.StartPos,
				},
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

func defineRestrictedOrDictionaryType() {

	// For the null denotation it is not clear after the start
	// if it is a restricted type or a dictionary type.
	//
	// If a colon is seen it is a dictionary type.
	// If no colon is seen it is a restricted type.

	setTypeNullDenotation(
		lexer.TokenBraceOpen,
		func(p *parser, startToken lexer.Token) ast.Type {

			var endPos ast.Position

			var dictionaryType *ast.DictionaryType
			var restrictedType *ast.RestrictedType

			var firstType ast.Type

			atEnd := false

			expectType := true

			for !atEnd {
				p.skipSpaceAndComments(true)

				switch p.current.Type {
				case lexer.TokenComma:
					if dictionaryType != nil {
						panic(fmt.Errorf("unexpected comma in dictionary type"))
					}
					if expectType {
						panic(fmt.Errorf("unexpected comma in restricted type"))
					}
					if restrictedType == nil {
						firstNominalType, ok := firstType.(*ast.NominalType)
						if !ok {
							panic(fmt.Errorf("non-nominal type in restriction list: %s", firstType))
						}
						restrictedType = &ast.RestrictedType{
							Restrictions: []*ast.NominalType{
								firstNominalType,
							},
							Range: ast.Range{
								StartPos: startToken.StartPos,
							},
						}
					}
					p.next()
					expectType = true

				case lexer.TokenColon:
					if restrictedType != nil {
						panic(fmt.Errorf("unexpected colon in restricted type"))
					}
					if expectType {
						panic(fmt.Errorf("unexpected colon in dictionary type"))
					}
					if dictionaryType == nil {
						if firstType == nil {
							panic(fmt.Errorf("unexpected colon after missing dictionary key type"))
						}
						dictionaryType = &ast.DictionaryType{
							KeyType: firstType,
							Range: ast.Range{
								StartPos: startToken.StartPos,
							},
						}
					} else {
						panic(fmt.Errorf("unexpected colon in dictionary type"))
					}
					p.next()
					expectType = true

				case lexer.TokenBraceClose:
					if expectType {
						switch {
						case dictionaryType != nil:
							p.report(fmt.Errorf("missing dictionary value type"))
						case restrictedType != nil:
							p.report(fmt.Errorf("missing type after comma"))
						}
					}
					endPos = p.current.EndPos
					p.next()
					atEnd = true

				case lexer.TokenEOF:
					if expectType {
						panic(fmt.Errorf("invalid end, expected type"))
					} else {
						panic(fmt.Errorf("missing end, expected %q", lexer.TokenBraceClose))
					}

				default:
					if !expectType {
						panic(fmt.Errorf("unexpected type"))
					}

					ty := parseType(p, lowestBindingPower)

					expectType = false

					switch {
					case dictionaryType != nil:
						dictionaryType.ValueType = ty

					case restrictedType != nil:
						nominalType, ok := ty.(*ast.NominalType)
						if !ok {
							panic(fmt.Errorf("non-nominal type in restriction list: %s", ty))
						}
						restrictedType.Restrictions = append(restrictedType.Restrictions, nominalType)

					default:
						firstType = ty
					}
				}
			}

			switch {
			case restrictedType != nil:
				restrictedType.Range.EndPos = endPos
				return restrictedType
			case dictionaryType != nil:
				dictionaryType.Range.EndPos = endPos
				return dictionaryType
			default:
				restrictedType = &ast.RestrictedType{
					Range: ast.Range{
						StartPos: startToken.StartPos,
						EndPos:   endPos,
					},
				}
				if firstType != nil {
					firstNominalType, ok := firstType.(*ast.NominalType)
					if !ok {
						panic(fmt.Errorf("non-nominal type in restriction list: %s", firstType))
					}
					restrictedType.Restrictions = append(restrictedType.Restrictions, firstNominalType)
				}
				return restrictedType
			}
		},
	)

	// For the left denotation we definitely know it is a restricted type

	setTypeLeftBindingPower(lexer.TokenBraceOpen, 30)
	setTypeLeftDenotation(
		lexer.TokenBraceOpen,
		func(p *parser, token lexer.Token, left ast.Type) ast.Type {

			var restrictions []*ast.NominalType

			var endPos ast.Position

			expectType := true

			atEnd := false
			for !atEnd {
				p.skipSpaceAndComments(true)

				switch p.current.Type {
				case lexer.TokenComma:
					if expectType {
						panic(fmt.Errorf("unexpected comma in restricted type"))
					}
					p.next()
					expectType = true

				case lexer.TokenBraceClose:
					if expectType && len(restrictions) > 0 {
						p.report(fmt.Errorf("missing type after comma"))
					}
					endPos = p.current.EndPos
					p.next()
					atEnd = true

				case lexer.TokenEOF:
					if expectType {
						panic(fmt.Errorf("invalid end, expected type"))
					} else {
						panic(fmt.Errorf("missing end, expected %q", lexer.TokenBraceClose))
					}

				default:
					if !expectType {
						panic(fmt.Errorf("unexpected token: got %q, expected \",\"", p.current.Type))
					}

					ty := parseType(p, lowestBindingPower)

					expectType = false

					nominalType, ok := ty.(*ast.NominalType)
					if !ok {
						panic(fmt.Errorf("non-nominal type in restriction list: %s", ty))
					}
					restrictions = append(restrictions, nominalType)
				}
			}

			return &ast.RestrictedType{
				Type:         left,
				Restrictions: restrictions,
				Range: ast.Range{
					StartPos: left.StartPosition(),
					EndPos:   endPos,
				},
			}
		},
	)
}

func parseType(p *parser, rightBindingPower int) ast.Type {
	p.skipSpaceAndComments(true)
	t := p.current
	p.next()

	left := applyTypeNullDenotation(p, t)

	for rightBindingPower < typeLeftBindingPowers[p.current.Type] {
		t = p.current
		p.next()

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
		panic(fmt.Errorf("missing type null denotation for token %q", token.Type))
	}
	return nullDenotation(p, token)
}

func applyTypeLeftDenotation(p *parser, token lexer.Token, left ast.Type) ast.Type {
	leftDenotation, ok := typeLeftDenotations[token.Type]
	if !ok {
		panic(fmt.Errorf("missing type left denotation for token %q", token.Type))
	}
	return leftDenotation(p, token, left)
}
