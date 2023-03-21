/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

type typeNullDenotationFunc func(parser *parser, token lexer.Token) (ast.Type, error)

var typeNullDenotations [lexer.TokenMax]typeNullDenotationFunc

type typeLeftDenotationFunc func(parser *parser, token lexer.Token, left ast.Type) (ast.Type, error)
type typeMetaLeftDenotationFunc func(
	p *parser,
	rightBindingPower int,
	left ast.Type,
) (
	result ast.Type,
	err error,
	done bool,
)

var typeLeftBindingPowers [lexer.TokenMax]int
var typeLeftDenotations [lexer.TokenMax]typeLeftDenotationFunc
var typeMetaLeftDenotations [lexer.TokenMax]typeMetaLeftDenotationFunc

func setTypeNullDenotation(tokenType lexer.TokenType, nullDenotation typeNullDenotationFunc) {
	current := typeNullDenotations[tokenType]
	if current != nil {
		panic(errors.NewUnexpectedError(
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
		panic(errors.NewUnexpectedError(
			"type left denotation for token %s already exists",
			tokenType,
		))
	}
	typeLeftDenotations[tokenType] = leftDenotation
}

func setTypeMetaLeftDenotation(tokenType lexer.TokenType, metaLeftDenotation typeMetaLeftDenotationFunc) {
	current := typeMetaLeftDenotations[tokenType]
	if current != nil {
		panic(errors.NewUnexpectedError(
			"type meta left denotation for token %s already exists",
			tokenType,
		))
	}
	typeMetaLeftDenotations[tokenType] = metaLeftDenotation
}

type prefixTypeFunc func(parser *parser, right ast.Type, tokenRange ast.Range) ast.Type
type postfixTypeFunc func(parser *parser, left ast.Type, tokenRange ast.Range) ast.Type

type literalType struct {
	nullDenotation typeNullDenotationFunc
	tokenType      lexer.TokenType
}

type prefixType struct {
	nullDenotation prefixTypeFunc
	bindingPower   int
	tokenType      lexer.TokenType
}

type postfixType struct {
	leftDenotation postfixTypeFunc
	bindingPower   int
	tokenType      lexer.TokenType
}

func defineType(def any) {
	switch def := def.(type) {
	case prefixType:
		tokenType := def.tokenType
		setTypeNullDenotation(
			tokenType,
			func(parser *parser, token lexer.Token) (ast.Type, error) {
				right, err := parseType(parser, def.bindingPower)
				if err != nil {
					return nil, err
				}

				return def.nullDenotation(parser, right, token.Range), nil
			},
		)
	case postfixType:
		tokenType := def.tokenType
		setTypeLeftBindingPower(tokenType, def.bindingPower)
		setTypeLeftDenotation(
			tokenType,
			func(p *parser, token lexer.Token, left ast.Type) (ast.Type, error) {
				return def.leftDenotation(p, left, token.Range), nil
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
	defineInstantiationType()
	defineIdentifierTypes()
	defineParenthesizedTypes()
}

func defineParenthesizedTypes() {
	setTypeNullDenotation(lexer.TokenParenOpen, func(p *parser, token lexer.Token) (ast.Type, error) {
		p.skipSpaceAndComments()
		innerType, err := parseType(p, lowestBindingPower)
		if err != nil {
			return nil, err
		}
		p.skipSpaceAndComments()
		_, err = p.mustOne(lexer.TokenParenClose)
		return innerType, err
	})
}

func parseNominalTypeRemainder(p *parser, token lexer.Token) (*ast.NominalType, error) {
	var nestedIdentifiers []ast.Identifier

	for p.current.Is(lexer.TokenDot) {
		// Skip the dot
		p.next()

		nestedToken := p.current

		if !nestedToken.Is(lexer.TokenIdentifier) {
			return nil, p.syntaxError(
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
	), nil
}

func defineArrayType() {
	setTypeNullDenotation(
		lexer.TokenBracketOpen,
		func(p *parser, startToken lexer.Token) (ast.Type, error) {

			elementType, err := parseType(p, lowestBindingPower)
			if err != nil {
				return nil, err
			}

			p.skipSpaceAndComments()

			var size *ast.IntegerExpression

			if p.current.Is(lexer.TokenSemicolon) {
				// Skip the semicolon
				p.nextSemanticToken()

				if !p.current.Type.IsIntegerLiteral() {
					p.reportSyntaxError("expected positive integer size for constant sized type")

					// Skip the invalid non-integer literal token
					p.next()

				} else {
					numberExpression, err := parseExpression(p, lowestBindingPower)
					if err != nil {
						return nil, err
					}

					integerExpression, ok := numberExpression.(*ast.IntegerExpression)
					if !ok || integerExpression.Value.Sign() < 0 {
						p.reportSyntaxError("expected positive integer size for constant sized type")
					} else {
						size = integerExpression
					}
				}
			}

			p.skipSpaceAndComments()

			endToken, err := p.mustOne(lexer.TokenBracketClose)
			if err != nil {
				return nil, err
			}

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
				), nil
			} else {
				return ast.NewVariableSizedType(
					p.memoryGauge,
					elementType,
					typeRange,
				), nil
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
				nil,
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
		func(p *parser, startToken lexer.Token) (ast.Type, error) {

			var endPos ast.Position

			var dictionaryType *ast.DictionaryType
			var restrictedType *ast.RestrictedType

			var firstType ast.Type

			atEnd := false

			expectType := true

			for !atEnd {
				p.skipSpaceAndComments()

				switch p.current.Type {
				case lexer.TokenComma:
					if dictionaryType != nil {
						return nil, p.syntaxError("unexpected comma in dictionary type")
					}
					if expectType {
						return nil, p.syntaxError("unexpected comma in restricted type")
					}
					if restrictedType == nil {
						firstNominalType, ok := firstType.(*ast.NominalType)
						if !ok {
							return nil, p.syntaxError("non-nominal type in restriction list: %s", firstType)
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
						return nil, p.syntaxError("unexpected colon in restricted type")
					}
					if expectType {
						return nil, p.syntaxError("unexpected colon in dictionary type")
					}
					if dictionaryType == nil {
						if firstType == nil {
							return nil, p.syntaxError("unexpected colon after missing dictionary key type")
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
						return nil, p.syntaxError("unexpected colon in dictionary type")
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
						return nil, p.syntaxError("invalid end of input, expected type")
					} else {
						return nil, p.syntaxError("invalid end of input, expected %s", lexer.TokenBraceClose)
					}

				default:
					if !expectType {
						return nil, p.syntaxError("unexpected type")
					}

					ty, err := parseType(p, lowestBindingPower)
					if err != nil {
						return nil, err
					}

					expectType = false

					switch {
					case dictionaryType != nil:
						dictionaryType.ValueType = ty

					case restrictedType != nil:
						nominalType, ok := ty.(*ast.NominalType)
						if !ok {
							return nil, p.syntaxError("non-nominal type in restriction list: %s", ty)
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
				return restrictedType, nil
			case dictionaryType != nil:
				dictionaryType.EndPos = endPos
				return dictionaryType, nil
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
						return nil, p.syntaxError("non-nominal type in restriction list: %s", firstType)
					}
					restrictedType.Restrictions = append(restrictedType.Restrictions, firstNominalType)
				}
				return restrictedType, nil
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
		func(p *parser, rightBindingPower int, left ast.Type) (result ast.Type, err error, done bool) {

			// Perform a lookahead

			current := p.current
			cursor := p.tokens.Cursor()

			// Skip the `{` token.
			p.next()

			// In case there is a space, the type is *not* considered a restricted type.
			// The buffered tokens are replayed to allow them to be re-parsed.

			if p.current.Is(lexer.TokenSpace) {
				p.current = current
				p.tokens.Revert(cursor)

				return left, nil, true
			}

			// It was determined that a restricted type is parsed.
			// Still, it should have maybe not been parsed if the right binding power
			// was higher. In that case, replay the buffered tokens and stop.

			if rightBindingPower >= typeLeftBindingPowerRestriction {
				p.current = current
				p.tokens.Revert(cursor)
				return left, nil, true
			}

			nominalTypes, endPos, err := parseNominalTypes(p, lexer.TokenBraceClose, false, lexer.TokenComma)

			if err != nil {
				return nil, err, true
			}

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

			return result, err, false
		},
	)
}

func parseNominalType(
	p *parser,
	rightBindingPower int,
	rejectAccessKeywords bool,
) (*ast.NominalType, error) {
	ty, err := parseType(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}
	nominalType, ok := ty.(*ast.NominalType)
	if !ok {
		return nil, p.syntaxError("unexpected non-nominal type: %s", ty)
	}
	if rejectAccessKeywords &&
		nominalType.Identifier.Identifier == KeywordAll ||
		nominalType.Identifier.Identifier == KeywordAccess ||
		nominalType.Identifier.Identifier == KeywordAccount ||
		nominalType.Identifier.Identifier == KeywordSelf {
		return nil, p.syntaxError("unexpected non-nominal type: %s", ty)
	}
	return nominalType, nil
}

// parseNominalTypes parses zero or more nominal types separated by a separator, either
// a comma `,` or a vertical bar `|`.
func parseNominalTypes(
	p *parser,
	endTokenType lexer.TokenType,
	rejectAccessKeywords bool,
	separator lexer.TokenType,
) (
	nominalTypes []*ast.NominalType,
	endPos ast.Position,
	err error,
) {
	expectType := true
	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments()

		switch p.current.Type {
		case separator:
			if expectType {
				return nil, ast.EmptyPosition, p.syntaxError("unexpected separator")
			}
			// Skip the separator
			p.next()
			expectType = true

		case endTokenType:
			if expectType && len(nominalTypes) > 0 {
				p.reportSyntaxError("missing type after separator")
			}
			endPos = p.current.EndPos
			atEnd = true

		case lexer.TokenEOF:
			if expectType {
				return nil, ast.EmptyPosition, p.syntaxError("invalid end of input, expected type")
			} else {
				return nil, ast.EmptyPosition, p.syntaxError("invalid end of input, expected %s", endTokenType)
			}

		default:
			if !expectType {
				return nil, ast.EmptyPosition, p.syntaxError(
					"unexpected token: got %s, expected %s or %s",
					p.current.Type,
					separator,
					endTokenType,
				)
			}

			expectType = false

			nominalType, err := parseNominalType(p, lowestBindingPower, rejectAccessKeywords)
			if err != nil {
				return nil, ast.EmptyPosition, err
			}
			nominalTypes = append(nominalTypes, nominalType)
		}
	}

	return
}

func parseParameterTypeAnnotations(p *parser) (typeAnnotations []*ast.TypeAnnotation, err error) {

	p.skipSpaceAndComments()
	_, err = p.mustOne(lexer.TokenParenOpen)
	if err != nil {
		return
	}

	expectTypeAnnotation := true

	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments()
		switch p.current.Type {
		case lexer.TokenComma:
			if expectTypeAnnotation {
				return nil, p.syntaxError(
					"expected type annotation or end of list, got %q",
					p.current.Type,
				)
			}
			// Skip the comma
			p.next()
			expectTypeAnnotation = true

		case lexer.TokenParenClose:
			// Don't skip the closing paren, so that we can mark the current
			// position as an end pos if the type signature is missing an explicit return type
			atEnd = true

		case lexer.TokenEOF:
			return nil, p.syntaxError(
				"missing %q at end of list",
				lexer.TokenParenClose,
			)

		default:
			if !expectTypeAnnotation {
				return nil, p.syntaxError(
					"expected comma or end of list, got %q",
					p.current.Type,
				)
			}

			typeAnnotation, err := parseTypeAnnotation(p)
			if err != nil {
				return nil, err
			}

			typeAnnotations = append(typeAnnotations, typeAnnotation)

			expectTypeAnnotation = false
		}
	}

	return
}

func parseType(p *parser, rightBindingPower int) (ast.Type, error) {
	if p.typeDepth == typeDepthLimit {
		return nil, TypeDepthLimitReachedError{
			Pos: p.current.StartPos,
		}
	}

	p.typeDepth++
	defer func() {
		p.typeDepth--
	}()

	p.skipSpaceAndComments()
	t := p.current
	p.next()

	left, err := applyTypeNullDenotation(p, t)
	if err != nil {
		return nil, err
	}

	for {
		var done bool
		left, err, done = applyTypeMetaLeftDenotation(p, rightBindingPower, left)
		if err != nil {
			return nil, err
		}

		if done {
			break
		}
	}

	return left, nil
}

func applyTypeMetaLeftDenotation(
	p *parser,
	rightBindingPower int,
	left ast.Type,
) (
	result ast.Type,
	err error,
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
func defaultTypeMetaLeftDenotation(
	p *parser,
	rightBindingPower int,
	left ast.Type,
) (
	result ast.Type,
	err error,
	done bool,
) {
	if rightBindingPower >= typeLeftBindingPowers[p.current.Type] {
		return left, nil, true
	}

	t := p.current

	p.next()

	result, err = applyTypeLeftDenotation(p, t, left)

	return result, err, false
}

func parseTypeAnnotation(p *parser) (*ast.TypeAnnotation, error) {
	p.skipSpaceAndComments()

	startPos := p.current.StartPos

	isResource := false
	if p.current.Is(lexer.TokenAt) {
		// Skip the `@`
		p.next()
		isResource = true
	}

	ty, err := parseType(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	return ast.NewTypeAnnotation(
		p.memoryGauge,
		isResource,
		ty,
		startPos,
	), nil
}

func applyTypeNullDenotation(p *parser, token lexer.Token) (ast.Type, error) {
	tokenType := token.Type
	nullDenotation := typeNullDenotations[tokenType]
	if nullDenotation == nil {
		return nil, p.syntaxError("unexpected token in type: %s", tokenType)
	}
	return nullDenotation(p, token)
}

func applyTypeLeftDenotation(p *parser, token lexer.Token, left ast.Type) (ast.Type, error) {
	leftDenotation := typeLeftDenotations[token.Type]
	if leftDenotation == nil {
		return nil, p.syntaxError("unexpected token in type: %s", token.Type)
	}
	return leftDenotation(p, token, left)
}

func parseNominalTypeInvocationRemainder(p *parser) (*ast.InvocationExpression, error) {
	p.skipSpaceAndComments()
	identifier, err := p.mustOne(lexer.TokenIdentifier)
	if err != nil {
		return nil, err
	}

	ty, err := parseNominalTypeRemainder(p, identifier)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()
	parenOpenToken, err := p.mustOne(lexer.TokenParenOpen)
	if err != nil {
		return nil, err
	}

	argumentsStartPos := parenOpenToken.EndPos
	arguments, endPos, err := parseArgumentListRemainder(p)
	if err != nil {
		return nil, err
	}

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
	), nil
}

// parseCommaSeparatedTypeAnnotations parses zero or more type annotations separated by comma.
func parseCommaSeparatedTypeAnnotations(
	p *parser,
	endTokenType lexer.TokenType,
) (
	typeAnnotations []*ast.TypeAnnotation,
	err error,
) {
	expectTypeAnnotation := true
	atEnd := false
	for !atEnd {
		p.skipSpaceAndComments()

		switch p.current.Type {
		case lexer.TokenComma:
			if expectTypeAnnotation {
				return nil, p.syntaxError("unexpected comma")
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
				return nil, p.syntaxError("invalid end of input, expected type")
			} else {
				return nil, p.syntaxError("invalid end of input, expected %s", endTokenType)
			}

		default:
			if !expectTypeAnnotation {
				return nil, p.syntaxError(
					"unexpected token: got %s, expected %s or %s",
					p.current.Type,
					lexer.TokenComma,
					endTokenType,
				)
			}

			typeAnnotation, err := parseTypeAnnotation(p)
			if err != nil {
				return nil, err
			}

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
		func(p *parser, token lexer.Token, left ast.Type) (ast.Type, error) {
			typeArgumentsStartPos := token.StartPos

			typeArguments, err := parseCommaSeparatedTypeAnnotations(p, lexer.TokenGreater)
			if err != nil {
				return nil, err
			}

			endToken, err := p.mustOne(lexer.TokenGreater)
			if err != nil {
				return nil, err
			}

			return ast.NewInstantiationType(
				p.memoryGauge,
				left,
				typeArguments,
				typeArgumentsStartPos,
				endToken.EndPos,
			), nil
		},
	)
}

func defineIdentifierTypes() {
	setTypeNullDenotation(
		lexer.TokenIdentifier,
		func(p *parser, token lexer.Token) (ast.Type, error) {
			switch string(p.tokenSource(token)) {
			case KeywordAuth:
				p.skipSpaceAndComments()

				var authorization ast.Authorization

				_, err := p.mustOne(lexer.TokenParenOpen)
				if err != nil {
					return nil, err
				}
				firstTy, err := parseNominalType(p, lowestBindingPower, true)
				if err != nil {
					return nil, err
				}
				entitlements := []*ast.NominalType{firstTy}
				p.skipSpaceAndComments()
				var separator lexer.TokenType

				switch p.current.Type {
				case lexer.TokenComma, lexer.TokenVerticalBar:
					separator = p.current.Type
				case lexer.TokenParenClose:
					authorization.EntitlementSet = ast.NewConjunctiveEntitlementSet(entitlements)
				default:
					return nil, p.syntaxError(
						"unexpected entitlement separator %s",
						p.current.Type.String(),
					)
				}
				p.nextSemanticToken()

				if separator != lexer.TokenError {
					remainingEntitlements, _, err := parseNominalTypes(p, lexer.TokenParenClose, true, separator)
					if err != nil {
						return nil, err
					}

					entitlements = append(entitlements, remainingEntitlements...)
					if len(entitlements) < 1 {
						return nil, p.syntaxError("entitlements list cannot be empty")
					}
					var entitlementSet ast.EntitlementSet
					if separator == lexer.TokenComma {
						entitlementSet = ast.NewConjunctiveEntitlementSet(entitlements)
					} else {
						entitlementSet = ast.NewDisjunctiveEntitlementSet(entitlements)
					}
					authorization.EntitlementSet = entitlementSet
					_, err = p.mustOne(lexer.TokenParenClose)
					if err != nil {
						return nil, err
					}
					p.skipSpaceAndComments()
				}

				_, err = p.mustOne(lexer.TokenAmpersand)
				if err != nil {
					return nil, err
				}

				right, err := parseType(p, typeLeftBindingPowerReference)
				if err != nil {
					return nil, err
				}

				return ast.NewReferenceType(
					p.memoryGauge,
					&authorization,
					right,
					token.StartPos,
				), nil

			case KeywordFun:
				p.skipSpaceAndComments()
				return parseFunctionType(p, token.StartPos, ast.FunctionPurityUnspecified)

			case KeywordView:

				current := p.current
				cursor := p.tokens.Cursor()

				// look ahead for the `fun` keyword, if it exists
				p.skipSpaceAndComments()

				if p.isToken(p.current, lexer.TokenIdentifier, KeywordFun) {
					// skip the `fun` keyword
					p.nextSemanticToken()
					return parseFunctionType(p, current.StartPos, ast.FunctionPurityView)
				}

				// backtrack otherwise - view is a nominal type here
				p.current = current
				p.tokens.Revert(cursor)
			}

			return parseNominalTypeRemainder(p, token)
		},
	)
}

// parse a function type starting after the `fun` keyword.
//
// ('view')? 'fun'
//
//	'(' ( type ( ',' type )* )? ')'
//	( ':' type )?
func parseFunctionType(p *parser, startPos ast.Position, purity ast.FunctionPurity) (ast.Type, error) {
	parameterTypeAnnotations, err := parseParameterTypeAnnotations(p)
	if err != nil {
		return nil, err
	}

	endPos := p.current.EndPos
	// skip the closing parenthesis of the argument tuple
	p.nextSemanticToken()

	var returnTypeAnnotation *ast.TypeAnnotation
	// return type annotation is optional in function types too
	if p.current.Is(lexer.TokenColon) {
		// skip the colon
		p.nextSemanticToken()

		returnTypeAnnotation, err = parseTypeAnnotation(p)
		if err != nil {
			return nil, err
		}
		endPos = returnTypeAnnotation.EndPosition(p.memoryGauge)
	} else {
		returnType := ast.NewNominalType(
			p.memoryGauge,
			ast.NewEmptyIdentifier(
				p.memoryGauge,
				endPos,
			),
			nil,
		)
		returnTypeAnnotation = ast.NewTypeAnnotation(
			p.memoryGauge,
			false,
			returnType,
			endPos,
		)
	}

	return ast.NewFunctionType(
		p.memoryGauge,
		purity,
		parameterTypeAnnotations,
		returnTypeAnnotation,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			endPos,
		),
	), nil

}
