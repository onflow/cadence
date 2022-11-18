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
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser/lexer"
)

func parseDeclarations(p *parser, endTokenType lexer.TokenType) (declarations []ast.Declaration, err error) {
	for {
		_, docString := p.parseTrivia(triviaOptions{
			skipNewlines:    true,
			parseDocStrings: true,
		})

		switch p.current.Type {
		case lexer.TokenSemicolon:
			// Skip the semicolon
			p.next()
			continue

		case endTokenType, lexer.TokenEOF:
			return

		default:
			var declaration ast.Declaration
			declaration, err = parseDeclaration(p, docString)
			if err != nil {
				return
			}

			if declaration == nil {
				return
			}

			declarations = append(declarations, declaration)
		}
	}
}

func parseDeclaration(p *parser, docString string) (ast.Declaration, error) {

	access := ast.AccessNotSpecified
	var accessPos *ast.Position

	var staticPos *ast.Position
	var nativePos *ast.Position

	staticModifierEnabled := p.config.StaticModifierEnabled
	nativeModifierEnabled := p.config.NativeModifierEnabled

	for {
		p.skipSpaceAndComments()

		switch p.current.Type {
		case lexer.TokenPragma:
			if access != ast.AccessNotSpecified {
				return nil, NewSyntaxError(*accessPos, "invalid access modifier for pragma")
			}
			if staticModifierEnabled && staticPos != nil {
				return nil, NewSyntaxError(*staticPos, "invalid static modifier for pragma")
			}
			if nativeModifierEnabled && nativePos != nil {
				return nil, NewSyntaxError(*nativePos, "invalid native modifier for pragma")
			}
			return parsePragmaDeclaration(p)

		case lexer.TokenIdentifier:
			switch string(p.currentTokenSource()) {
			case keywordLet, keywordVar:
				if staticModifierEnabled && staticPos != nil {
					return nil, NewSyntaxError(*staticPos, "invalid static modifier for variable")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, NewSyntaxError(*nativePos, "invalid native modifier for variable")
				}
				return parseVariableDeclaration(p, access, accessPos, docString)

			case keywordFun:
				return parseFunctionDeclaration(
					p,
					false,
					access,
					accessPos,
					staticPos,
					nativePos,
					docString,
				)

			case keywordImport:
				if staticModifierEnabled && staticPos != nil {
					return nil, NewSyntaxError(*staticPos, "invalid static modifier for import")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, NewSyntaxError(*nativePos, "invalid native modifier for import")
				}
				return parseImportDeclaration(p)

			case keywordEvent:
				if staticModifierEnabled && staticPos != nil {
					return nil, NewSyntaxError(*staticPos, "invalid static modifier for event")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, NewSyntaxError(*nativePos, "invalid native modifier for event")
				}
				return parseEventDeclaration(p, access, accessPos, docString)

			case keywordStruct, keywordResource, keywordContract, keywordEnum:
				if staticModifierEnabled && staticPos != nil {
					return nil, NewSyntaxError(*staticPos, "invalid static modifier for composite")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, NewSyntaxError(*nativePos, "invalid native modifier for composite")
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case keywordAttachment:
				return parseAttachmentDeclaration(p, access, accessPos, docString)

			case KeywordTransaction:
				if access != ast.AccessNotSpecified {
					return nil, NewSyntaxError(*accessPos, "invalid access modifier for transaction")
				}
				if staticModifierEnabled && staticPos != nil {
					return nil, NewSyntaxError(*staticPos, "invalid static modifier for transaction")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, NewSyntaxError(*nativePos, "invalid native modifier for transaction")
				}
				return parseTransactionDeclaration(p, docString)

			case keywordPriv, keywordPub, keywordAccess:
				if access != ast.AccessNotSpecified {
					return nil, p.syntaxError("invalid second access modifier")
				}
				if staticModifierEnabled && staticPos != nil {
					return nil, p.syntaxError("invalid access modifier after static modifier")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, p.syntaxError("invalid access modifier after native modifier")
				}
				pos := p.current.StartPos
				accessPos = &pos
				var err error
				access, err = parseAccess(p)
				if err != nil {
					return nil, err
				}

				continue

			case keywordStatic:
				if !staticModifierEnabled {
					break
				}

				if staticPos != nil {
					return nil, p.syntaxError("invalid second static modifier")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, p.syntaxError("invalid static modifier after native modifier")
				}
				pos := p.current.StartPos
				staticPos = &pos
				p.next()
				continue

			case keywordNative:
				if !nativeModifierEnabled {
					break
				}

				if nativePos != nil {
					return nil, p.syntaxError("invalid second native modifier")
				}
				pos := p.current.StartPos
				nativePos = &pos
				p.next()
				continue
			}
		}

		return nil, nil
	}
}

var enumeratedAccessModifierKeywords = common.EnumerateWords(
	[]string{
		strconv.Quote(keywordAll),
		strconv.Quote(keywordAccount),
		strconv.Quote(keywordContract),
		strconv.Quote(keywordSelf),
	},
	"or",
)

// parseAccess parses an access modifier
//
//	access
//	    : 'priv'
//	    | 'pub' ( '(' 'set' ')' )?
//	    | 'access' '(' ( 'self' | 'contract' | 'account' | 'all' ) ')'
func parseAccess(p *parser) (ast.Access, error) {

	switch string(p.currentTokenSource()) {
	case keywordPriv:
		// Skip the `priv` keyword
		p.next()
		return ast.AccessPrivate, nil

	case keywordPub:
		// Skip the `pub` keyword
		p.nextSemanticToken()
		if !p.current.Is(lexer.TokenParenOpen) {
			return ast.AccessPublic, nil
		}

		// Skip the opening paren
		p.nextSemanticToken()

		if !p.current.Is(lexer.TokenIdentifier) {
			return ast.AccessNotSpecified, p.syntaxError(
				"expected keyword %q, got %s",
				keywordSet,
				p.current.Type,
			)
		}

		keyword := p.currentTokenSource()
		if string(keyword) != keywordSet {
			return ast.AccessNotSpecified, p.syntaxError(
				"expected keyword %q, got %q",
				keywordSet,
				keyword,
			)
		}

		// Skip the `set` keyword
		p.nextSemanticToken()

		_, err := p.mustOne(lexer.TokenParenClose)
		if err != nil {
			return ast.AccessNotSpecified, err
		}

		return ast.AccessPublicSettable, nil

	case keywordAccess:
		// Skip the `access` keyword
		p.nextSemanticToken()

		_, err := p.mustOne(lexer.TokenParenOpen)
		if err != nil {
			return ast.AccessNotSpecified, err
		}

		p.skipSpaceAndComments()

		if !p.current.Is(lexer.TokenIdentifier) {
			return ast.AccessNotSpecified, p.syntaxError(
				"expected keyword %s, got %s",
				enumeratedAccessModifierKeywords,
				p.current.Type,
			)
		}

		var access ast.Access

		keyword := p.currentTokenSource()
		switch string(keyword) {
		case keywordAll:
			access = ast.AccessPublic

		case keywordAccount:
			access = ast.AccessAccount

		case keywordContract:
			access = ast.AccessContract

		case keywordSelf:
			access = ast.AccessPrivate

		default:
			return ast.AccessNotSpecified, p.syntaxError(
				"expected keyword %s, got %q",
				enumeratedAccessModifierKeywords,
				keyword,
			)
		}

		// Skip the keyword
		p.nextSemanticToken()

		_, err = p.mustOne(lexer.TokenParenClose)
		if err != nil {
			return ast.AccessNotSpecified, err
		}

		return access, nil

	default:
		return ast.AccessNotSpecified, errors.NewUnreachableError()
	}
}

// parseVariableDeclaration parses a variable declaration.
//
//	variableKind : 'var' | 'let'
//
//	variableDeclaration :
//	    variableKind identifier ( ':' typeAnnotation )?
//	    transfer expression
//	    ( transfer expression )?
func parseVariableDeclaration(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) (*ast.VariableDeclaration, error) {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	isLet := string(p.currentTokenSource()) == keywordLet

	// Skip the `let` or `var` keyword
	p.nextSemanticToken()
	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected identifier after start of variable declaration, got %s",
			p.current.Type,
		)
	}

	identifier := p.tokenToIdentifier(p.current)

	// Skip the identifier
	p.nextSemanticToken()

	var typeAnnotation *ast.TypeAnnotation
	var err error

	if p.current.Is(lexer.TokenColon) {
		// Skip the colon
		p.nextSemanticToken()

		typeAnnotation, err = parseTypeAnnotation(p)
		if err != nil {
			return nil, err
		}
	}

	p.skipSpaceAndComments()
	transfer := parseTransfer(p)
	if transfer == nil {
		return nil, p.syntaxError("expected transfer")
	}

	value, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	secondTransfer := parseTransfer(p)
	var secondValue ast.Expression
	if secondTransfer != nil {
		secondValue, err = parseExpression(p, lowestBindingPower)
		if err != nil {
			return nil, err
		}
	}

	variableDeclaration := ast.NewVariableDeclaration(
		p.memoryGauge,
		access,
		isLet,
		identifier,
		typeAnnotation,
		value,
		transfer,
		startPos,
		secondTransfer,
		secondValue,
		docString,
	)

	castingExpression, leftIsCasting := value.(*ast.CastingExpression)
	if leftIsCasting {
		castingExpression.ParentVariableDeclaration = variableDeclaration
	}

	return variableDeclaration, nil
}

// parseTransfer parses a transfer.
//
//	transfer : '=' | '<-' | '<-!'
func parseTransfer(p *parser) *ast.Transfer {
	var operation ast.TransferOperation

	switch p.current.Type {
	case lexer.TokenEqual:
		operation = ast.TransferOperationCopy

	case lexer.TokenLeftArrow:
		operation = ast.TransferOperationMove

	case lexer.TokenLeftArrowExclamation:
		operation = ast.TransferOperationMoveForced
	}

	if operation == ast.TransferOperationUnknown {
		return nil
	}

	pos := p.current.StartPos

	p.next()

	return ast.NewTransfer(
		p.memoryGauge,
		operation,
		pos,
	)
}

func parsePragmaDeclaration(p *parser) (*ast.PragmaDeclaration, error) {
	startPos := p.current.StartPosition()
	p.next()

	expr, err := parseExpression(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	return ast.NewPragmaDeclaration(
		p.memoryGauge,
		expr,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			expr.EndPosition(p.memoryGauge),
		),
	), nil
}

// parseImportDeclaration parses an import declaration
//
//	importDeclaration :
//	    'import'
//	    ( identifier (',' identifier)* 'from' )?
//	    ( string | hexadecimalLiteral | identifier )
func parseImportDeclaration(p *parser) (*ast.ImportDeclaration, error) {

	startPosition := p.current.StartPos

	var identifiers []ast.Identifier

	var location common.Location
	var locationPos ast.Position
	var endPos ast.Position

	parseStringOrAddressLocation := func() {
		locationPos = p.current.StartPos
		endPos = p.current.EndPos

		switch p.current.Type {
		case lexer.TokenString:
			literal := p.currentTokenSource()
			parsedString := parseStringLiteral(p, literal)
			location = common.NewStringLocation(p.memoryGauge, parsedString)

		case lexer.TokenHexadecimalIntegerLiteral:
			location = parseHexadecimalLocation(p)

		default:
			panic(errors.NewUnreachableError())
		}

		// Skip the location
		p.next()
	}

	setIdentifierLocation := func(identifier ast.Identifier) {
		location = common.IdentifierLocation(identifier.Identifier)
		locationPos = identifier.Pos
		endPos = identifier.EndPosition(p.memoryGauge)
	}

	parseLocation := func() error {
		switch p.current.Type {
		case lexer.TokenString, lexer.TokenHexadecimalIntegerLiteral:
			parseStringOrAddressLocation()

		case lexer.TokenIdentifier:
			identifier := p.tokenToIdentifier(p.current)
			setIdentifierLocation(identifier)
			p.next()

		default:
			return p.syntaxError(
				"unexpected token in import declaration: got %s, expected string, address, or identifier",
				p.current.Type,
			)
		}

		return nil
	}

	parseMoreIdentifiers := func() error {
		expectCommaOrFrom := false

		atEnd := false
		for !atEnd {
			p.nextSemanticToken()

			switch p.current.Type {
			case lexer.TokenComma:
				if !expectCommaOrFrom {
					return p.syntaxError(
						"expected %s or keyword %q, got %s",
						lexer.TokenIdentifier,
						keywordFrom,
						p.current.Type,
					)
				}
				expectCommaOrFrom = false

			case lexer.TokenIdentifier:

				keyword := p.currentTokenSource()
				if string(keyword) == keywordFrom {
					if expectCommaOrFrom {
						atEnd = true

						// Skip the `from` keyword
						p.nextSemanticToken()

						err := parseLocation()
						if err != nil {
							return err
						}

						break
					}

					isCommaOrFrom, err := isNextTokenCommaOrFrom(p)
					if err != nil {
						return err
					}

					if !isCommaOrFrom {
						return p.syntaxError(
							"expected %s, got keyword %q",
							lexer.TokenIdentifier,
							keyword,
						)
					}

					// If the next token is either comma or 'from' token, then fall through
					// and process the current 'from' token as an identifier.
				}

				identifier := p.tokenToIdentifier(p.current)
				identifiers = append(identifiers, identifier)

				expectCommaOrFrom = true

			case lexer.TokenEOF:
				return p.syntaxError(
					"unexpected end in import declaration: expected %s or %s",
					lexer.TokenIdentifier,
					lexer.TokenComma,
				)

			default:
				return p.syntaxError(
					"unexpected token in import declaration: got %s, expected keyword %q or %s",
					p.current.Type,
					keywordFrom,
					lexer.TokenComma,
				)
			}
		}

		return nil
	}

	maybeParseFromIdentifier := func(identifier ast.Identifier) error {
		// The current identifier is maybe the `from` keyword,
		// in which case the given (previous) identifier was
		// an imported identifier and not the import location.
		//
		// If it is not the `from` keyword,
		// the given (previous) identifier is the import location.

		if string(p.currentTokenSource()) == keywordFrom {
			identifiers = append(identifiers, identifier)
			// Skip the `from` keyword
			p.nextSemanticToken()

			err := parseLocation()
			if err != nil {
				return err
			}
		} else {
			setIdentifierLocation(identifier)
		}

		return nil
	}

	// Skip the `import` keyword
	p.nextSemanticToken()

	switch p.current.Type {
	case lexer.TokenString, lexer.TokenHexadecimalIntegerLiteral:
		parseStringOrAddressLocation()

	case lexer.TokenIdentifier:
		identifier := p.tokenToIdentifier(p.current)
		// Skip the identifier
		p.nextSemanticToken()

		switch p.current.Type {
		case lexer.TokenComma:
			// The previous identifier is an imported identifier,
			// not the import location
			identifiers = append(identifiers, identifier)
			err := parseMoreIdentifiers()
			if err != nil {
				return nil, err
			}
		case lexer.TokenIdentifier:
			err := maybeParseFromIdentifier(identifier)
			if err != nil {
				return nil, err
			}
		case lexer.TokenEOF:
			// The previous identifier is the identifier location
			setIdentifierLocation(identifier)

		default:
			return nil, p.syntaxError(
				"unexpected token in import declaration: got %s, expected keyword %q or %s",
				p.current.Type,
				keywordFrom,
				lexer.TokenComma,
			)
		}

	case lexer.TokenEOF:
		return nil, p.syntaxError("unexpected end in import declaration: expected string, address, or identifier")

	default:
		return nil, p.syntaxError(
			"unexpected token in import declaration: got %s, expected string, address, or identifier",
			p.current.Type,
		)
	}

	return ast.NewImportDeclaration(
		p.memoryGauge,
		identifiers,
		location,
		ast.NewRange(
			p.memoryGauge,
			startPosition,
			endPos,
		),
		locationPos,
	), nil
}

// isNextTokenCommaOrFrom check whether the token to follow is a comma or a from token.
func isNextTokenCommaOrFrom(p *parser) (b bool, err error) {
	p.startBuffering()
	defer func() {
		err = p.replayBuffered()
	}()

	// skip the current token
	p.nextSemanticToken()

	// Lookahead the next token
	switch p.current.Type {
	case lexer.TokenIdentifier:
		isFrom := string(p.currentTokenSource()) == keywordFrom
		return isFrom, nil
	case lexer.TokenComma:
		return true, nil
	default:
		return false, nil
	}
}

func parseHexadecimalLocation(p *parser) common.AddressLocation {
	// TODO: improve
	literal := string(p.currentTokenSource())

	bytes := []byte(strings.ReplaceAll(literal[2:], "_", ""))

	length := len(bytes)
	if length%2 == 1 {
		bytes = append([]byte{'0'}, bytes...)
		length++
	}

	rawAddress := make([]byte, hex.DecodedLen(length))
	_, err := hex.Decode(rawAddress, bytes)
	if err != nil {
		// unreachable, hex literal should always be valid
		panic(errors.NewUnexpectedErrorFromCause(err))
	}

	address, err := common.BytesToAddress(rawAddress)
	if err != nil {
		// Any returned error is a syntax error. e.g: Address too large error.
		p.reportSyntaxError(err.Error())
	}

	return common.NewAddressLocation(p.memoryGauge, address, "")
}

// parseEventDeclaration parses an event declaration.
//
//	eventDeclaration : 'event' identifier parameterList
func parseEventDeclaration(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) (*ast.CompositeDeclaration, error) {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	// Skip the `event` keyword
	p.nextSemanticToken()
	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected identifier after start of event declaration, got %s",
			p.current.Type,
		)
	}

	identifier := p.tokenToIdentifier(p.current)
	// Skip the identifier
	p.next()

	parameterList, err := parseParameterList(p)
	if err != nil {
		return nil, err
	}

	initializer := ast.NewSpecialFunctionDeclaration(
		p.memoryGauge,
		common.DeclarationKindInitializer,
		ast.NewFunctionDeclaration(
			p.memoryGauge,
			ast.AccessNotSpecified,
			false,
			false,
			ast.NewEmptyIdentifier(p.memoryGauge, ast.EmptyPosition),
			parameterList,
			nil,
			nil,
			parameterList.StartPos,
			"",
		),
	)

	members := ast.NewMembers(
		p.memoryGauge,
		[]ast.Declaration{
			initializer,
		},
	)

	return ast.NewCompositeDeclaration(
		p.memoryGauge,
		access,
		common.CompositeKindEvent,
		identifier,
		nil,
		members,
		docString,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			parameterList.EndPos,
		),
	), nil
}

// parseCompositeKind parses a composite kind.
//
//	compositeKind : 'struct' | 'resource' | 'contract' | 'enum'
func parseCompositeKind(p *parser) common.CompositeKind {

	if p.current.Is(lexer.TokenIdentifier) {
		switch string(p.currentTokenSource()) {
		case keywordStruct:
			return common.CompositeKindStructure

		case keywordResource:
			return common.CompositeKindResource

		case keywordContract:
			return common.CompositeKindContract

		case keywordEnum:
			return common.CompositeKindEnum
		}
	}

	return common.CompositeKindUnknown
}

// parseFieldWithVariableKind parses a field which has a variable kind.
//
//	variableKind : 'var' | 'let'
//
//	field : variableKind identifier ':' typeAnnotation
func parseFieldWithVariableKind(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	staticPos *ast.Position,
	nativePos *ast.Position,
	docString string,
) (*ast.FieldDeclaration, error) {

	startPos := ast.EarliestPosition(p.current.StartPos, accessPos, staticPos, nativePos)

	var variableKind ast.VariableKind
	switch string(p.currentTokenSource()) {
	case keywordLet:
		variableKind = ast.VariableKindConstant

	case keywordVar:
		variableKind = ast.VariableKindVariable
	}

	// Skip the `let` or `var` keyword
	p.nextSemanticToken()
	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected identifier after start of field declaration, got %s",
			p.current.Type,
		)
	}

	identifier := p.tokenToIdentifier(p.current)
	// Skip the identifier
	p.nextSemanticToken()

	_, err := p.mustOne(lexer.TokenColon)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	typeAnnotation, err := parseTypeAnnotation(p)
	if err != nil {
		return nil, err
	}

	return ast.NewFieldDeclaration(
		p.memoryGauge,
		access,
		staticPos != nil,
		nativePos != nil,
		variableKind,
		identifier,
		typeAnnotation,
		docString,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			typeAnnotation.EndPosition(p.memoryGauge),
		),
	), nil
}

func parseConformances(p *parser) ([]*ast.NominalType, error) {
	var conformances []*ast.NominalType
	var err error

	if p.current.Is(lexer.TokenColon) {
		// Skip the colon
		p.next()

		conformances, _, err = parseNominalTypes(p, lexer.TokenBraceOpen)
		if err != nil {
			return nil, err
		}

		if len(conformances) < 1 {
			return nil, p.syntaxError(
				"expected at least one conformance after %s",
				lexer.TokenColon,
			)
		}
	}

	p.skipSpaceAndComments()
	return conformances, nil
}

// parseCompositeOrInterfaceDeclaration parses an event declaration.
//
//	conformances : ':' nominalType ( ',' nominalType )*
//
//	compositeDeclaration : compositeKind identifier conformances?
//	                       '{' membersAndNestedDeclarations '}'
//
//	interfaceDeclaration : compositeKind 'interface' identifier conformances?
//	                       '{' membersAndNestedDeclarations '}'
func parseCompositeOrInterfaceDeclaration(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) (ast.Declaration, error) {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	compositeKind := parseCompositeKind(p)

	// Skip the composite kind keyword
	p.next()

	var isInterface bool
	var identifier ast.Identifier

	for {
		p.skipSpaceAndComments()
		if !p.current.Is(lexer.TokenIdentifier) {
			return nil, p.syntaxError(
				"expected %s, got %s",
				lexer.TokenIdentifier,
				p.current.Type,
			)
		}

		wasInterface := isInterface

		if string(p.currentTokenSource()) == keywordInterface {
			isInterface = true
			if wasInterface {
				return nil, p.syntaxError(
					"expected interface name, got keyword %q",
					keywordInterface,
				)
			}
			// Skip the `interface` keyword
			p.next()
			continue
		} else {
			identifier = p.tokenToIdentifier(p.current)
			// Skip the identifier
			p.next()
			break
		}
	}

	p.skipSpaceAndComments()

	conformances, err := parseConformances(p)
	if err != nil {
		return nil, err
	}

	_, err = p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	members, err := parseMembersAndNestedDeclarations(p, lexer.TokenBraceClose)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	endToken, err := p.mustOne(lexer.TokenBraceClose)
	if err != nil {
		return nil, err
	}

	declarationRange := ast.NewRange(
		p.memoryGauge,
		startPos,
		endToken.EndPos,
	)

	if isInterface {
		// TODO: remove once interface conformances are supported
		if len(conformances) > 0 {
			// TODO: improve
			return nil, p.syntaxError("unexpected conformances")
		}

		return ast.NewInterfaceDeclaration(
			p.memoryGauge,
			access,
			compositeKind,
			identifier,
			members,
			docString,
			declarationRange,
		), nil
	} else {
		return ast.NewCompositeDeclaration(
			p.memoryGauge,
			access,
			compositeKind,
			identifier,
			conformances,
			members,
			docString,
			declarationRange,
		), nil
	}
}

func parseAttachmentDeclaration(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) (ast.Declaration, error) {
	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	// Skip the attachment keyword
	p.next()

	p.skipSpaceAndComments()
	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected %s, got %s",
			lexer.TokenIdentifier,
			p.current.Type,
		)
	}
	identifier, err := p.mustIdentifier()
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	if string(p.tokenSource(p.current)) != keywordFor {
		return nil, p.syntaxError(
			"expected 'for', got %s",
			p.current.Type,
		)
	}

	// skip the for keyword
	p.next()
	p.skipSpaceAndComments()

	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected %s, got %s",
			lexer.TokenIdentifier,
			p.current.Type,
		)
	}

	baseType, err := parseType(p, lowestBindingPower)
	baseNominalType, ok := baseType.(*ast.NominalType)
	if !ok {
		p.reportSyntaxError(
			"expected nominal type, got %s",
			baseType,
		)
	}
	if err != nil {
		return nil, err
	}

	conformances, err := parseConformances(p)
	if err != nil {
		return nil, err
	}

	_, err = p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	members, err := parseMembersAndNestedDeclarations(p, lexer.TokenBraceClose)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	endToken, err := p.mustOne(lexer.TokenBraceClose)
	if err != nil {
		return nil, err
	}

	declarationRange := ast.NewRange(
		p.memoryGauge,
		startPos,
		endToken.EndPos,
	)

	return ast.NewAttachmentDeclaration(
		p.memoryGauge,
		access,
		identifier,
		baseNominalType,
		conformances,
		members,
		docString,
		declarationRange,
	), nil
}

// parseMembersAndNestedDeclarations parses composite or interface members,
// and nested declarations.
//
//	membersAndNestedDeclarations : ( memberOrNestedDeclaration ';'* )*
func parseMembersAndNestedDeclarations(p *parser, endTokenType lexer.TokenType) (*ast.Members, error) {

	var declarations []ast.Declaration

	for {
		_, docString := p.parseTrivia(triviaOptions{
			skipNewlines:    true,
			parseDocStrings: true,
		})

		switch p.current.Type {
		case lexer.TokenSemicolon:
			// Skip the semicolon
			p.next()
			continue

		case endTokenType, lexer.TokenEOF:
			return ast.NewMembers(p.memoryGauge, declarations), nil

		default:
			memberOrNestedDeclaration, err := parseMemberOrNestedDeclaration(p, docString)
			if err != nil {
				return nil, err
			}

			if memberOrNestedDeclaration == nil {
				return ast.NewMembers(p.memoryGauge, declarations), nil
			}

			declarations = append(declarations, memberOrNestedDeclaration)
		}
	}
}

// parseMemberOrNestedDeclaration parses a composite or interface member,
// or a declaration nested in it.
//
//	memberOrNestedDeclaration : field
//	                          | specialFunctionDeclaration
//	                          | functionDeclaration
//	                          | interfaceDeclaration
//	                          | compositeDeclaration
//	                          | eventDeclaration
//	                          | enumCase
func parseMemberOrNestedDeclaration(p *parser, docString string) (ast.Declaration, error) {

	const functionBlockIsOptional = true

	access := ast.AccessNotSpecified
	var accessPos *ast.Position

	var staticPos *ast.Position
	var nativePos *ast.Position

	var previousIdentifierToken *lexer.Token

	staticModifierEnabled := p.config.StaticModifierEnabled
	nativeModifierEnabled := p.config.NativeModifierEnabled

	for {
		p.skipSpaceAndComments()

		switch p.current.Type {
		case lexer.TokenIdentifier:

			if previousIdentifierToken != nil {
				return nil, NewSyntaxError(
					previousIdentifierToken.StartPos,
					"unexpected token: %s",
					previousIdentifierToken.Type,
				)
			}

			switch string(p.currentTokenSource()) {
			case keywordLet, keywordVar:
				return parseFieldWithVariableKind(
					p,
					access,
					accessPos,
					staticPos,
					nativePos,
					docString,
				)

			case keywordCase:
				if staticModifierEnabled && staticPos != nil {
					return nil, NewSyntaxError(*staticPos, "invalid static modifier for enum case")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, NewSyntaxError(*nativePos, "invalid native modifier for enum case")
				}
				return parseEnumCase(p, access, accessPos, docString)

			case keywordFun:
				return parseFunctionDeclaration(
					p,
					functionBlockIsOptional,
					access,
					accessPos,
					staticPos,
					nativePos,
					docString,
				)

			case keywordEvent:
				if staticModifierEnabled && staticPos != nil {
					return nil, NewSyntaxError(*staticPos, "invalid static modifier for event")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, NewSyntaxError(*nativePos, "invalid native modifier for event")
				}
				return parseEventDeclaration(p, access, accessPos, docString)

			case keywordStruct, keywordResource, keywordContract, keywordEnum:
				if staticModifierEnabled && staticPos != nil {
					return nil, NewSyntaxError(*staticPos, "invalid static modifier for composite")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, NewSyntaxError(*nativePos, "invalid native modifier for composite")
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case keywordAttachment:
				return parseAttachmentDeclaration(p, access, accessPos, docString)

			case keywordPriv, keywordPub, keywordAccess:
				if access != ast.AccessNotSpecified {
					return nil, p.syntaxError("invalid second access modifier")
				}
				if staticModifierEnabled && staticPos != nil {
					return nil, p.syntaxError("invalid access modifier after static modifier")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, p.syntaxError("invalid access modifier after native modifier")
				}
				pos := p.current.StartPos
				accessPos = &pos
				var err error
				access, err = parseAccess(p)
				if err != nil {
					return nil, err
				}
				continue

			case keywordStatic:
				if !staticModifierEnabled {
					break
				}

				if staticPos != nil {
					return nil, p.syntaxError("invalid second static modifier")
				}
				if nativeModifierEnabled && nativePos != nil {
					return nil, p.syntaxError("invalid static modifier after native modifier")
				}
				pos := p.current.StartPos
				staticPos = &pos
				p.next()
				continue

			case keywordNative:
				if !nativeModifierEnabled {
					break
				}

				if nativePos != nil {
					return nil, p.syntaxError("invalid second native modifier")
				}
				pos := p.current.StartPos
				nativePos = &pos
				p.next()
				continue
			}

			t := p.current
			previousIdentifierToken = &t
			// Skip the identifier
			p.next()
			continue

		case lexer.TokenColon:
			if previousIdentifierToken == nil {
				return nil, p.syntaxError("unexpected %s", p.current.Type)
			}

			identifier := p.tokenToIdentifier(*previousIdentifierToken)
			return parseFieldDeclarationWithoutVariableKind(
				p,
				access,
				accessPos,
				staticPos,
				nativePos,
				identifier,
				docString,
			)

		case lexer.TokenParenOpen:
			if previousIdentifierToken == nil {
				return nil, p.syntaxError("unexpected %s", p.current.Type)
			}

			identifier := p.tokenToIdentifier(*previousIdentifierToken)
			return parseSpecialFunctionDeclaration(
				p,
				functionBlockIsOptional,
				access,
				accessPos,
				staticPos,
				nativePos,
				identifier,
			)
		}

		return nil, nil
	}
}

func parseFieldDeclarationWithoutVariableKind(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	staticPos *ast.Position,
	nativePos *ast.Position,
	identifier ast.Identifier,
	docString string,
) (*ast.FieldDeclaration, error) {

	startPos := ast.EarliestPosition(identifier.Pos, accessPos, staticPos, nativePos)

	_, err := p.mustOne(lexer.TokenColon)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	typeAnnotation, err := parseTypeAnnotation(p)
	if err != nil {
		return nil, err
	}

	return ast.NewFieldDeclaration(
		p.memoryGauge,
		access,
		staticPos != nil,
		nativePos != nil,
		ast.VariableKindNotSpecified,
		identifier,
		typeAnnotation,
		docString,
		ast.NewRange(
			p.memoryGauge,
			startPos,
			typeAnnotation.EndPosition(p.memoryGauge),
		),
	), nil
}

func parseSpecialFunctionDeclaration(
	p *parser,
	functionBlockIsOptional bool,
	access ast.Access,
	accessPos *ast.Position,
	staticPos *ast.Position,
	nativePos *ast.Position,
	identifier ast.Identifier,
) (*ast.SpecialFunctionDeclaration, error) {

	startPos := ast.EarliestPosition(identifier.Pos, accessPos, staticPos, nativePos)

	// TODO: switch to parseFunctionParameterListAndRest once old parser is deprecated:
	//   allow a return type annotation while parsing, but reject later.

	parameterList, err := parseParameterList(p)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	var functionBlock *ast.FunctionBlock

	if !functionBlockIsOptional ||
		p.current.Is(lexer.TokenBraceOpen) {

		functionBlock, err = parseFunctionBlock(p)
		if err != nil {
			return nil, err
		}
	}

	declarationKind := common.DeclarationKindUnknown
	switch identifier.Identifier {
	case keywordInit:
		declarationKind = common.DeclarationKindInitializer

	case keywordDestroy:
		declarationKind = common.DeclarationKindDestructor

	case keywordPrepare:
		declarationKind = common.DeclarationKindPrepare
	}

	return ast.NewSpecialFunctionDeclaration(
		p.memoryGauge,
		declarationKind,
		ast.NewFunctionDeclaration(
			p.memoryGauge,
			access,
			staticPos != nil,
			nativePos != nil,
			identifier,
			parameterList,
			nil,
			functionBlock,
			startPos,
			"",
		),
	), nil
}

// parseEnumCase parses a field which has a variable kind.
//
//	enumCase : 'case' identifier
func parseEnumCase(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) (*ast.EnumCaseDeclaration, error) {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	// Skip the `enum` keyword
	p.nextSemanticToken()
	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected identifier after start of enum case declaration, got %s",
			p.current.Type,
		)
	}

	identifier := p.tokenToIdentifier(p.current)
	// Skip the identifier
	p.next()

	return ast.NewEnumCaseDeclaration(
		p.memoryGauge,
		access,
		identifier,
		docString,
		startPos,
	), nil
}
