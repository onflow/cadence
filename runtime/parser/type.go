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

const (
	typeLeftBindingPowerOptional = 10 * (iota + 1)
	typeLeftBindingPowerReference
	typeLeftBindingPowerRestriction
	typeLeftBindingPowerInstantiation
)

type typeNullDenotationFunc func(parser *parser, token lexer.Token) ast.Type

var typeNullDenotations [lexer.TokenMax]typeNullDenotationFunc

type typeLeftDenotationFunc func(parser *parser, token lexer.Token, left ast.Type) ast.Type
type typeMetaLeftDenotationFunc func(
	p *parser,
	rightBindingPower int,
	left ast.Type,
) (
	result ast.Type,
	done bool,
)

var typeLeftBindingPowers [lexer.TokenMax]int
var typeLeftDenotations [lexer.TokenMax]typeLeftDenotationFunc
var typeMetaLeftDenotations [lexer.TokenMax]typeMetaLeftDenotationFunc

func setTypeNullDenotation(tokenType lexer.TokenType, nullDenotation typeNullDenotationFunc) {
	current := typeNullDenotations[tokenType]
	if current != nil {
		panic(NewUnpositionedSyntaxError(
			"type null denotation for token %s already exists",
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
		panic(NewUnpositionedSyntaxError(
			"type left denotation for token %s already exists",
			tokenType,
		))
	}
	typeLeftDenotations[tokenType] = leftDenotation
}

func setTypeMetaLeftDenotation(tokenType lexer.TokenType, metaLeftDenotation typeMetaLeftDenotationFunc) {
	current := typeMetaLeftDenotations[tokenType]
	if current != nil {
		panic(NewUnpositionedSyntaxError(
			"type meta left denotation for token %s already exists",
			tokenType,
		))
	}
	typeMetaLeftDenotations[tokenType] = metaLeftDenotation
}

type prefixTypeFunc func(parser *parser, right ast.Type, tokenRange ast.Range) ast.Type
type postfixTypeFunc func(parser *parser, left ast.Type, tokenRange ast.Range) ast.Type

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

func defineType(def any) {
	switch def := def.(type) {
	case prefixType:
		tokenType := def.tokenType
		setTypeNullDenotation(
			tokenType,
			func(parser *parser, token lexer.Token) ast.Type {
				right := parseType(parser, def.bindingPower)
				return def.nullDenotation(parser, right, token.Range)
			},
		)
	case postfixType:
		tokenType := def.tokenType
		setTypeLeftBindingPower(tokenType, def.bindingPower)
		setTypeLeftDenotation(
			tokenType,
			func(p *parser, token lexer.Token, left ast.Type) ast.Type {
				return def.leftDenotation(p, left, token.Range)
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
	defineArrayType()
	defineOptionalType()
	defineReferenceType()
	defineRestrictedOrDictionaryType()
	defineFunctionType()
	defineInstantiationType()

	setTypeNullDenotation(
		lexer.TokenIdentifier,
		func(p *parser, token lexer.Token) ast.Type {

			switch token.Value {
			case keywordAuth:
				p.skipSpaceAndComments(true)
				p.mustOne(lexer.TokenAmpersand)
				right := parseType(p, typeLeftBindingPowerReference)
				return ast.NewReferenceType(
					p.memoryGauge,
					true,
					right,
					token.StartPos,
				)

			default:
				return parseNominalTypeRemainder(p, token)
			}
		},
	)
}

func parseNominalTypeRemainder(p *parser, token lexer.Token) *ast.NominalType {
	var nestedIdentifiers []ast.Identifier

	for p.current.Is(lexer.TokenDot) {
		// Skip the dot
		p.next()

		nestedToken := p.current

		if !nestedToken.Is(lexer.TokenIdentifier) {
			p.panicSyntaxError(
				"expected identifier after %s, got %s",
				lexer.TokenDot,
				nestedToken.Type,
			)
		}

		nestedIdentifier := p.tokenToIdentifier(nestedToken)

		// Skip the identifier
		p.next()

		nestedIdentifiers = append(
			nestedIdentifiers,
			nestedIdentifier,
		)

	}

	return ast.NewNominalType(
		p.memoryGauge,
		p.tokenToIdentifier(token),
		nestedIdentifiers,
	)
}

func defineArrayType() {
	setTypeNullDenotation(
		lexer.TokenBracketOpen,
		func(p *parser, startToken lexer.Token) ast.Type {

			elementType := parseType(p, lowestBindingPower)

			p.skipSpaceAndComments(true)

			var size *ast.IntegerExpression

			if p.current.Is(lexer.TokenSemicolon) {
				// Skip the semicolon
				p.next()

				p.skipSpaceAndComments(true)

				if !p.current.Type.IsIntegerLiteral() {
					p.reportSyntaxError("expected positive integer size for constant sized type")

					// Skip the invalid non-integer literal token
					p.next()

				} else {
					numberExpression := parseExpression(p, lowestBindingPower)

					integerExpression, ok := numberExpression.(*ast.IntegerExpression)
					if !ok || integerExpression.Value.Sign() < 0 {
						p.reportSyntaxError("expected positive integer size for constant sized type")
					} else {
						size = integerExpression
					}
				}
			}

			p.skipSpaceAndComments(true)

			endToken := p.mustOne(lexer.TokenBracketClose)

			typeRange := ast.NewRange(
				p.memoryGauge,
				startToken.StartPos,
				endToken.EndPos,
			)

			if size != nil {
				return ast.NewConstantSizedType(
					p.memoryGauge,
					elementType,
					size,
					typeRange,
				)
			} else {
				return ast.NewVariableSizedType(
					p.memoryGauge,
					elementType,
					typeRange,
				)
			}
		},
	)
}

func defineOptionalType() {
	defineType(postfixType{
		tokenType:    lexer.TokenQuestionMark,
		bindingPower: typeLeftBindingPowerOptional,
		leftDenotation: func(p *parser, left ast.Type, tokenRange ast.Range) ast.Type {
			return ast.NewOptionalType(
				p.memoryGauge,
				left,
				tokenRange.EndPos,
			)
		},
	})

	defineType(postfixType{
		tokenType:    lexer.TokenDoubleQuestionMark,
		bindingPower: typeLeftBindingPowerOptional,
		leftDenotation: func(p *parser, left ast.Type, tokenRange ast.Range) ast.Type {
			return ast.NewOptionalType(
				p.memoryGauge,
				ast.NewOptionalType(
					p.memoryGauge,
					left,
					tokenRange.StartPos,
				),
				tokenRange.EndPos,
			)
		},
	})
}

func defineReferenceType() {
	defineType(prefixType{
		tokenType:    lexer.TokenAmpersand,
		bindingPower: typeLeftBindingPowerReference,
		nullDenotation: func(p *parser, right ast.Type, tokenRange ast.Range) ast.Type {
			return ast.NewReferenceType(
				p.memoryGauge,
				false,
				right,
				tokenRange.StartPos,
			)
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
						p.panicSyntaxError("unexpected comma in dictionary type")
					}
					if expectType {
						p.panicSyntaxError("unexpected comma in restricted type")
					}
					if restrictedType == nil {
						firstNominalType, ok := firstType.(*ast.NominalType)
						if !ok {
							p.panicSyntaxError("non-nominal type in restriction list: %s", firstType)
						}
						restrictedType = ast.NewRestrictedType(
							p.memoryGauge,
							nil,
							[]*ast.NominalType{
								firstNominalType,
							},
							ast.NewRange(
								p.memoryGauge,
								startToken.StartPos,
								ast.EmptyPosition,
							),
						)
					}
					// Skip the comma
					p.next()
					expectType = true

				case lexer.TokenColon:
					if restrictedType != nil {
						p.panicSyntaxError("unexpected colon in restricted type")
					}
					if expectType {
						p.panicSyntaxError("unexpected colon in dictionary type")
					}
					if dictionaryType == nil {
						if firstType == nil {
							p.panicSyntaxError("unexpected colon after missing dictionary key type")
						}
						dictionaryType = ast.NewDictionaryType(
							p.memoryGauge,
							firstType,
							nil,
							ast.NewRange(
								p.memoryGauge,
								startToken.StartPos,
								ast.EmptyPosition,
							),
						)
					} else {
						p.panicSyntaxError("unexpected colon in dictionary type")
					}
					// Skip the colon
					p.next()
					expectType = true

				case lexer.TokenBraceClose:
					if expectType {
						switch {
						case dictionaryType != nil:
							p.reportSyntaxError("missing dictionary value type")
						case restrictedType != nil:
							p.reportSyntaxError("missing type after comma")
						}
					}
					endPos = p.current.EndPos
					// Skip the closing brace
					p.next()
					atEnd = true

				case lexer.TokenEOF:
					if expectType {
						p.panicSyntaxError("invalid end of input, expected type")
					} else {
						p.panicSyntaxError("invalid end of input, expected %s", lexer.TokenBraceClose)
					}

				default:
					if !expectType {
						p.panicSyntaxError("unexpected type")
					}

					ty := parseType(p, lowestBindingPower)

					expectType = false

					switch {
					case dictionaryType != nil:
						dictionaryType.ValueType = ty

					case restrictedType != nil:
						nominalType, ok := ty.(*ast.NominalType)
						if !ok {
							p.panicSyntaxError("non-nominal type in restriction list: %s", ty)
						}
						restrictedType.Restrictions = append(restrictedType.Restrictions, nominalType)

					default:
						firstType = ty
					}
				}
			}

			switch {
			case restrictedType != nil:
				restrictedType.EndPos = endPos
				return restrictedType
			case dictionaryType != nil:
				dictionaryType.EndPos = endPos
				return dictionaryType
			default:
				restrictedType = ast.NewRestrictedType(
					p.memoryGauge,
					nil,
					nil,
					ast.NewRange(
						p.memoryGauge,
						startToken.StartPos,
						endPos,
					),
				)
				if firstType != nil {
					firstNominalType, ok := firstType.(*ast.NominalType)
					if !ok {
						p.panicSyntaxError("non-nominal type in restriction list: %s", firstType)
					}
					restrictedType.Restrictions = append(restrictedType.Restrictions, firstNominalType)
				}
				return restrictedType
			}
		},
	)

	// For the left denotation we need a meta left denotation:
	// We need to look ahead and check if the brace is followed by whitespace or not.
	// In case there is a space, the type is *not* considered a restricted type.
	// This handles the ambiguous case where a function return type's open brace
	// may either be a restricted type (if there is no whitespace)
	// or the start of the function body (if there is whitespace).

	setTypeMetaLeftDenotation(
		lexer.TokenBraceOpen,
		func(p *parser, rightBindingPower int, left ast.Type) (result ast.Type, done bool) {

			// Start buffering before skipping the `{` token,
			// so it can be replayed in case the

			p.startBuffering()

			// Skip the `{` token.
			p.next()

			// In case there is a space, the type is *not* considered a restricted type.
			// The buffered tokens are replayed to allow them to be re-parsed.

			if p.current.Is(lexer.TokenSpace) {
				p.replayBuffered()
				return left, true
			}

			// It was determined that a restricted type is parsed.
			// Still, it should have maybe not been parsed if the right binding power
			// was higher. In that case, replay the buffered tokens and stop.

			if rightBindingPower >= typeLeftBindingPowerRestriction {
				p.replayBuffered()
				return left, true
			}

			p.acceptBuffered()

			nominalTypes, endPos := parseNominalTypes(p, lexer.TokenBraceClose)

			// Skip the closing brace
			p.next()

			result = ast.NewRestrictedType(
				p.memoryGauge,
				left,
				nominalTypes,
				ast.NewRange(
					p.memoryGauge,
					left.StartPosition(),
					endPos,
				),
			)

			return result, false
		},
	)
}

// parseNominalTypes parses zero or more nominal types separated by comma.
//
func parseNominalTypes(
	p *parser,
	endTokenType lexer.TokenType,
) (
	nominalTypes []*ast.NominalType,
	endPos ast.Position,
) {
	expectType := true
	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenComma:
			if expectType {
				p.panicSyntaxError("unexpected comma")
			}
			// Skip the comma
			p.next()
			expectType = true

		case endTokenType:
			if expectType && len(nominalTypes) > 0 {
				p.reportSyntaxError("missing type after comma")
			}
			endPos = p.current.EndPos
			atEnd = true

		case lexer.TokenEOF:
			if expectType {
				p.panicSyntaxError("invalid end of input, expected type")
			} else {
				p.panicSyntaxError("invalid end of input, expected %s", endTokenType)
			}

		default:
			if !expectType {
				p.panicSyntaxError(
					"unexpected token: got %s, expected %s or %s",
					p.current.Type,
					lexer.TokenComma,
					endTokenType,
				)
			}

			ty := parseType(p, lowestBindingPower)

			expectType = false

			nominalType, ok := ty.(*ast.NominalType)
			if !ok {
				p.panicSyntaxError("unexpected non-nominal type: %s", ty)
			}
			nominalTypes = append(nominalTypes, nominalType)
		}
	}

	return
}

func defineFunctionType() {
	setTypeNullDenotation(
		lexer.TokenParenOpen,
		func(p *parser, startToken lexer.Token) ast.Type {

			parameterTypeAnnotations := parseParameterTypeAnnotations(p)

			p.skipSpaceAndComments(true)
			p.mustOne(lexer.TokenColon)

			p.skipSpaceAndComments(true)
			returnTypeAnnotation := parseTypeAnnotation(p)

			p.skipSpaceAndComments(true)
			endToken := p.mustOne(lexer.TokenParenClose)

			return ast.NewFunctionType(
				p.memoryGauge,
				parameterTypeAnnotations,
				returnTypeAnnotation,
				ast.NewRange(
					p.memoryGauge,
					startToken.StartPos,
					endToken.EndPos,
				),
			)
		},
	)
}

func parseParameterTypeAnnotations(p *parser) (typeAnnotations []*ast.TypeAnnotation) {

	p.skipSpaceAndComments(true)
	p.mustOne(lexer.TokenParenOpen)

	expectTypeAnnotation := true

	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments(true)
		switch p.current.Type {
		case lexer.TokenComma:
			if expectTypeAnnotation {
				p.panicSyntaxError(
					"expected type annotation or end of list, got %q",
					p.current.Type,
				)
			}
			// Skip the comma
			p.next()
			expectTypeAnnotation = true

		case lexer.TokenParenClose:
			// Skip the closing paren
			p.next()
			atEnd = true

		case lexer.TokenEOF:
			p.panicSyntaxError(
				"missing %q at end of list",
				lexer.TokenParenClose,
			)

		default:
			if !expectTypeAnnotation {
				p.panicSyntaxError(
					"expected comma or end of list, got %q",
					p.current.Type,
				)
			}

			typeAnnotation := parseTypeAnnotation(p)
			typeAnnotations = append(typeAnnotations, typeAnnotation)

			expectTypeAnnotation = false
		}
	}

	return
}

func parseType(p *parser, rightBindingPower int) ast.Type {

	if p.typeDepth == typeDepthLimit {
		panic(TypeDepthLimitReachedError{
			Pos: p.current.StartPos,
		})
	}

	p.typeDepth++
	defer func() {
		p.typeDepth--
	}()

	p.skipSpaceAndComments(true)
	t := p.current
	p.next()

	left := applyTypeNullDenotation(p, t)

	for {
		var done bool
		left, done = applyTypeMetaLeftDenotation(p, rightBindingPower, left)
		if done {
			break
		}
	}

	return left
}

func applyTypeMetaLeftDenotation(
	p *parser,
	rightBindingPower int,
	left ast.Type,
) (
	result ast.Type,
	done bool,
) {
	// By default, left denotations are applied if the right binding power
	// is less than the left binding power of the current token.
	//
	// Token-specific meta-left denotations allow customizing this,
	// e.g. determining the left binding power based on parsing more tokens,
	// or performing look-ahead

	metaLeftDenotation := typeMetaLeftDenotations[p.current.Type]
	if metaLeftDenotation == nil {
		metaLeftDenotation = defaultTypeMetaLeftDenotation
	}

	return metaLeftDenotation(p, rightBindingPower, left)
}

// defaultTypeMetaLeftDenotation is the default type left denotation, which applies
// if the right binding power is less than the left binding power of the current token
//
func defaultTypeMetaLeftDenotation(
	p *parser,
	rightBindingPower int,
	left ast.Type,
) (
	result ast.Type,
	done bool,
) {
	if rightBindingPower >= typeLeftBindingPowers[p.current.Type] {
		return left, true
	}

	t := p.current

	p.next()

	result = applyTypeLeftDenotation(p, t, left)
	return result, false
}

func parseTypeAnnotation(p *parser) *ast.TypeAnnotation {
	startPos := p.current.StartPos

	isResource := false
	if p.current.Is(lexer.TokenAt) {
		// Skip the `@`
		p.next()
		isResource = true
	}

	ty := parseType(p, lowestBindingPower)

	return ast.NewTypeAnnotation(
		p.memoryGauge,
		isResource,
		ty,
		startPos,
	)
}

func applyTypeNullDenotation(p *parser, token lexer.Token) ast.Type {
	tokenType := token.Type
	nullDenotation := typeNullDenotations[tokenType]
	if nullDenotation == nil {
		p.panicSyntaxError("unexpected token in type: %s", tokenType)
	}
	return nullDenotation(p, token)
}

func applyTypeLeftDenotation(p *parser, token lexer.Token, left ast.Type) ast.Type {
	leftDenotation := typeLeftDenotations[token.Type]
	if leftDenotation == nil {
		p.panicSyntaxError("unexpected token in type: %s", token.Type)
	}
	return leftDenotation(p, token, left)
}

func parseNominalTypeInvocationRemainder(p *parser) *ast.InvocationExpression {
	p.skipSpaceAndComments(true)
	identifier := p.mustOne(lexer.TokenIdentifier)
	ty := parseNominalTypeRemainder(p, identifier)

	p.skipSpaceAndComments(true)
	parenOpenToken := p.mustOne(lexer.TokenParenOpen)
	argumentsStartPos := parenOpenToken.EndPos
	arguments, endPos := parseArgumentListRemainder(p)

	var invokedExpression ast.Expression = ast.NewIdentifierExpression(
		p.memoryGauge,
		ty.Identifier,
	)

	for _, nestedIdentifier := range ty.NestedIdentifiers {
		invokedExpression = ast.NewMemberExpression(
			p.memoryGauge,
			invokedExpression,
			false,
			nestedIdentifier.Pos,
			nestedIdentifier,
		)
	}

	return ast.NewInvocationExpression(
		p.memoryGauge,
		invokedExpression,
		nil,
		arguments,
		argumentsStartPos,
		endPos,
	)
}

// parseCommaSeparatedTypeAnnotations parses zero or more type annotations separated by comma.
//
func parseCommaSeparatedTypeAnnotations(
	p *parser,
	endTokenType lexer.TokenType,
) (
	typeAnnotations []*ast.TypeAnnotation,
) {
	expectTypeAnnotation := true
	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenComma:
			if expectTypeAnnotation {
				p.panicSyntaxError("unexpected comma")
			}
			// Skip the comma
			p.next()
			expectTypeAnnotation = true

		case endTokenType:
			if expectTypeAnnotation && len(typeAnnotations) > 0 {
				p.reportSyntaxError("missing type annotation after comma")
			}
			atEnd = true

		case lexer.TokenEOF:
			if expectTypeAnnotation {
				p.panicSyntaxError("invalid end of input, expected type")
			} else {
				p.panicSyntaxError("invalid end of input, expected %s", endTokenType)
			}

		default:
			if !expectTypeAnnotation {
				p.panicSyntaxError(
					"unexpected token: got %s, expected %s or %s",
					p.current.Type,
					lexer.TokenComma,
					endTokenType,
				)
			}

			typeAnnotation := parseTypeAnnotation(p)
			typeAnnotations = append(typeAnnotations, typeAnnotation)

			expectTypeAnnotation = false
		}
	}

	return
}

func defineInstantiationType() {
	setTypeLeftBindingPower(lexer.TokenLess, typeLeftBindingPowerInstantiation)
	setTypeLeftDenotation(
		lexer.TokenLess,
		func(p *parser, token lexer.Token, left ast.Type) ast.Type {
			typeArgumentsStartPos := token.StartPos

			typeArguments := parseCommaSeparatedTypeAnnotations(p, lexer.TokenGreater)

			endToken := p.mustOne(lexer.TokenGreater)

			return ast.NewInstantiationType(
				p.memoryGauge,
				left,
				typeArguments,
				typeArgumentsStartPos,
				endToken.EndPos,
			)
		},
	)
}
