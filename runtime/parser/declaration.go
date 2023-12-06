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
	"encoding/hex"
	"fmt"
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

	var access ast.Access = ast.AccessNotSpecified
	var accessPos *ast.Position

	purity := ast.FunctionPurityUnspecified
	var purityPos *ast.Position

	var staticPos *ast.Position
	var nativePos *ast.Position

	staticModifierEnabled := p.config.StaticModifierEnabled
	nativeModifierEnabled := p.config.NativeModifierEnabled

	for {
		p.skipSpaceAndComments()

		switch p.current.Type {
		case lexer.TokenPragma:
			if purity != ast.FunctionPurityUnspecified {
				return nil, NewSyntaxError(*purityPos, "invalid view modifier for pragma")
			}
			err := rejectAllModifiers(p, access, accessPos, staticPos, nativePos, common.DeclarationKindPragma)
			if err != nil {
				return nil, err
			}
			return parsePragmaDeclaration(p)

		case lexer.TokenIdentifier:
			switch string(p.currentTokenSource()) {

			case KeywordLet, KeywordVar:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindVariable)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for variable")
				}
				return parseVariableDeclaration(p, access, accessPos, docString)

			case KeywordFun:
				return parseFunctionDeclaration(
					p,
					false,
					access,
					accessPos,
					purity,
					purityPos,
					staticPos,
					nativePos,
					docString,
				)

			case KeywordImport:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindImport)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for import")
				}
				return parseImportDeclaration(p)

			case KeywordEvent:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindEvent)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for event")
				}
				return parseEventDeclaration(p, access, accessPos, docString)

			case KeywordStruct:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindStructure)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for struct")
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordResource:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindResource)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for resource")
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordEntitlement:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindEntitlement)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for entitlement")
				}
				return parseEntitlementOrMappingDeclaration(p, access, accessPos, docString)

			case keywordAttachment:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindAttachment)
				if err != nil {
					return nil, err
				}
				return parseAttachmentDeclaration(p, access, accessPos, docString)

			case KeywordContract:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindContract)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for contract")
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordEnum:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindEnum)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for enum")
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordTransaction:
				err := rejectAllModifiers(p, access, accessPos, staticPos, nativePos, common.DeclarationKindTransaction)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for transaction")
				}

				return parseTransactionDeclaration(p, docString)

			case KeywordView:
				if purity != ast.FunctionPurityUnspecified {
					return nil, p.syntaxError("invalid second view modifier")
				}

				pos := p.current.StartPos
				purityPos = &pos
				purity = parsePurityAnnotation(p)
				continue

			case KeywordPub:
				return nil, p.syntaxErrorWithSuggestedFix("`pub` is no longer a valid access keyword", "`access(all)`")

			case KeywordPriv:
				return nil, p.syntaxErrorWithSuggestedFix("`priv` is no longer a valid access keyword", "`access(self)`")

			case KeywordAccess:
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

			case KeywordStatic:
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

			case KeywordNative:
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
		strconv.Quote(KeywordAll),
		strconv.Quote(KeywordAccount),
		strconv.Quote(KeywordContract),
		strconv.Quote(KeywordSelf),
	},
	"or",
)

func rejectAccessKeywords(p *parser, produceNominalType func() (*ast.NominalType, error)) (*ast.NominalType, error) {
	nominalType, err := produceNominalType()

	if err != nil {
		return nil, err
	}

	switch nominalType.Identifier.Identifier {
	case KeywordAll, KeywordAccess, KeywordAccount, KeywordSelf:
		return nil, p.syntaxError("unexpected non-nominal type: %s", nominalType)
	}
	return nominalType, nil
}

func parseEntitlementList(p *parser) (ast.EntitlementSet, error) {
	firstTy, err := rejectAccessKeywords(p, func() (*ast.NominalType, error) {
		return parseNominalType(p, lowestBindingPower)
	})

	if err != nil {
		return nil, err
	}
	p.skipSpaceAndComments()
	entitlements := []*ast.NominalType{firstTy}
	var separator lexer.TokenType

	switch p.current.Type {
	case lexer.TokenComma, lexer.TokenVerticalBar:
		separator = p.current.Type
		p.nextSemanticToken()
	case lexer.TokenParenClose:
		// it is impossible to disambiguate at parsing time between an access that is a single
		// conjunctive entitlement, a single disjunctive entitlement, and the name of an entitlement mapping.
		// Luckily, however, the former two are just equivalent, and the latter we can disambiguate in the type checker.
		return ast.NewConjunctiveEntitlementSet(entitlements), nil
	default:
		return nil, p.syntaxError(
			"unexpected entitlement separator %s",
			p.current.Type.String(),
		)
	}

	remainingEntitlements, _, err := parseNominalTypes(p, lexer.TokenParenClose, separator)
	if err != nil {
		return nil, err
	}
	for _, entitlement := range remainingEntitlements {
		switch entitlement.Identifier.Identifier {
		case KeywordAll, KeywordAccess, KeywordAccount, KeywordSelf:
			return nil, p.syntaxError("unexpected non-nominal type: %s", entitlement)
		}
		entitlements = append(entitlements, entitlement)
	}

	var entitlementSet ast.EntitlementSet
	if separator == lexer.TokenComma {
		entitlementSet = ast.NewConjunctiveEntitlementSet(entitlements)
	} else {
		entitlementSet = ast.NewDisjunctiveEntitlementSet(entitlements)
	}
	return entitlementSet, nil
}

// parseAccess parses an access modifier
//
//	access
//	    : 'access(self)'
//	    | 'access(all)' ( '(' 'set' ')' )?
//	    | 'access' '(' ( 'self' | 'contract' | 'account' | 'all' | entitlementList ) ')'
func parseAccess(p *parser) (ast.Access, error) {

	switch string(p.currentTokenSource()) {
	case KeywordAccess:
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
		case KeywordAll:
			access = ast.AccessAll
			// Skip the keyword
			p.nextSemanticToken()

		case KeywordAccount:
			access = ast.AccessAccount
			// Skip the keyword
			p.nextSemanticToken()

		case KeywordContract:
			access = ast.AccessContract
			// Skip the keyword
			p.nextSemanticToken()

		case KeywordSelf:
			access = ast.AccessSelf
			// Skip the keyword
			p.nextSemanticToken()

		case KeywordMapping:

			keywordPos := p.current.StartPos
			// Skip the keyword
			p.nextSemanticToken()

			entitlementMapName, err := parseNominalType(p, lowestBindingPower)
			if err != nil {
				return ast.AccessNotSpecified, err
			}
			access = ast.NewMappedAccess(entitlementMapName, keywordPos)

			p.skipSpaceAndComments()

		default:
			entitlements, err := parseEntitlementList(p)
			if err != nil {
				return ast.AccessNotSpecified, err
			}
			access = ast.NewEntitlementAccess(entitlements)
		}

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

	isLet := string(p.currentTokenSource()) == KeywordLet

	// Skip the `let` or `var` keyword
	p.nextSemanticToken()

	identifier, err := p.nonReservedIdentifier("after start of variable declaration")
	if err != nil {
		return nil, err
	}

	// Skip the identifier
	p.nextSemanticToken()

	var typeAnnotation *ast.TypeAnnotation

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
						KeywordFrom,
						p.current.Type,
					)
				}
				expectCommaOrFrom = false

			case lexer.TokenIdentifier:

				keyword := p.currentTokenSource()
				if string(keyword) == KeywordFrom {
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

					if !isNextTokenCommaOrFrom(p) {
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
					KeywordFrom,
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

		if string(p.currentTokenSource()) == KeywordFrom {
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
				KeywordFrom,
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
func isNextTokenCommaOrFrom(p *parser) bool {
	current := p.current
	cursor := p.tokens.Cursor()
	defer func() {
		p.current = current
		p.tokens.Revert(cursor)
	}()

	// skip the current token
	p.nextSemanticToken()

	// Lookahead the next token
	switch p.current.Type {
	case lexer.TokenIdentifier:
		return string(p.currentTokenSource()) == KeywordFrom

	case lexer.TokenComma:
		return true
	}

	return false
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
	identifier, err := p.nonReservedIdentifier("after start of event declaration")
	if err != nil {
		return nil, err
	}

	// Skip the identifier
	p.next()

	// if this is a `ResourceDestroyed` event (i.e., a default event declaration), parse default arguments
	parseDefaultArguments := ast.IsResourceDestructionDefaultEvent(identifier.Identifier)
	parameterList, err := parseParameterList(p, parseDefaultArguments)
	if err != nil {
		return nil, err
	}

	initializer := ast.NewSpecialFunctionDeclaration(
		p.memoryGauge,
		common.DeclarationKindInitializer,
		ast.NewFunctionDeclaration(
			p.memoryGauge,
			ast.AccessNotSpecified,
			ast.FunctionPurityUnspecified,
			false,
			false,
			ast.NewEmptyIdentifier(p.memoryGauge, ast.EmptyPosition),
			nil,
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
		case KeywordStruct:
			return common.CompositeKindStructure

		case KeywordResource:
			return common.CompositeKindResource

		case KeywordContract:
			return common.CompositeKindContract

		case KeywordEnum:
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
	case KeywordLet:
		variableKind = ast.VariableKindConstant

	case KeywordVar:
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

// parseEntitlementMapping parses an entitlement mapping
//
//	entitlementMapping : nominalType '->' nominalType
func parseEntitlementMapping(p *parser, docString string) (*ast.EntitlementMapRelation, error) {
	inputType, err := parseType(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}
	inputNominalType, ok := inputType.(*ast.NominalType)
	if !ok {
		p.reportSyntaxError(
			"expected nominal type, got %s",
			inputType,
		)
	}

	p.skipSpaceAndComments()

	_, err = p.mustOne(lexer.TokenRightArrow)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	outputType, err := parseType(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	outputNominalType, ok := outputType.(*ast.NominalType)
	if !ok {
		p.reportSyntaxError(
			"expected nominal type, got %s",
			outputType,
		)
	}

	p.skipSpaceAndComments()

	return ast.NewEntitlementMapRelation(p.memoryGauge, inputNominalType, outputNominalType), nil
}

// parseEntitlementMappings parses entitlement mappings
func parseEntitlementMappingsAndInclusions(p *parser, endTokenType lexer.TokenType) ([]ast.EntitlementMapElement, error) {
	var elements []ast.EntitlementMapElement

	for {
		_, docString := p.parseTrivia(triviaOptions{
			skipNewlines:    true,
			parseDocStrings: true,
		})

		switch p.current.Type {

		case endTokenType, lexer.TokenEOF:
			return elements, nil

		default:
			if string(p.currentTokenSource()) == KeywordInclude {
				// Skip the `include` keyword
				p.nextSemanticToken()
				outputType, err := parseType(p, lowestBindingPower)
				if err != nil {
					return nil, err
				}

				outputNominalType, ok := outputType.(*ast.NominalType)
				if !ok {
					p.reportSyntaxError(
						"expected nominal type, got %s",
						outputType,
					)
				}

				p.skipSpaceAndComments()
				elements = append(elements, outputNominalType)
			} else {
				mapping, err := parseEntitlementMapping(p, docString)
				if err != nil {
					return nil, err
				}

				elements = append(elements, mapping)
			}
		}
	}
}

// parseEntitlementOrMappingDeclaration parses an entitlement declaration,
// or an entitlement mapping declaration
//
// entitlementDeclaration : 'entitlement' identifier
//
// mappingDeclaration : 'entitlement' 'mapping' identifier '{' entitlementMappings '}'
func parseEntitlementOrMappingDeclaration(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) (ast.Declaration, error) {
	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	// Skip the `entitlement` keyword
	p.nextSemanticToken()

	var isMapping bool
	expectString := "following entitlement declaration"

	if !p.current.Is(lexer.TokenIdentifier) {
		return nil, p.syntaxError(
			"expected %s, got %s",
			lexer.TokenIdentifier,
			p.current.Type,
		)
	}

	if string(p.currentTokenSource()) == KeywordMapping {
		// we are parsing an entitlement mapping
		// Skip the `mapping` keyword
		p.nextSemanticToken()
		isMapping = true
		expectString = "following entitlement mapping declaration"
	}

	// parse the name of the entitlement or mapping
	var identifier ast.Identifier
	identifier, err := p.nonReservedIdentifier(expectString)
	if err != nil {
		return nil, err
	}
	p.nextSemanticToken()

	if isMapping {
		_, err = p.mustOne(lexer.TokenBraceOpen)
		if err != nil {
			return nil, err
		}
		elements, err := parseEntitlementMappingsAndInclusions(p, lexer.TokenBraceClose)
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

		return ast.NewEntitlementMappingDeclaration(
			p.memoryGauge,
			access,
			identifier,
			elements,
			docString,
			declarationRange,
		), nil
	} else {
		declarationRange := ast.NewRange(
			p.memoryGauge,
			startPos,
			identifier.Pos,
		)

		return ast.NewEntitlementDeclaration(
			p.memoryGauge,
			access,
			identifier,
			docString,
			declarationRange,
		), nil
	}
}

func parseConformances(p *parser) ([]*ast.NominalType, error) {
	var conformances []*ast.NominalType
	var err error

	if p.current.Is(lexer.TokenColon) {
		// Skip the colon
		p.next()

		conformances, _, err = parseNominalTypes(p, lexer.TokenBraceOpen, lexer.TokenComma)
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

		if string(p.currentTokenSource()) == KeywordInterface {
			isInterface = true
			if wasInterface {
				return nil, p.syntaxError(
					"expected interface name, got keyword %q",
					KeywordInterface,
				)
			}
			// Skip the `interface` keyword
			p.next()
			continue
		} else {
			ctx := fmt.Sprintf("following %s declaration", compositeKind.Keyword())
			nonReserved, err := p.nonReservedIdentifier(ctx)
			if err != nil {
				return nil, err
			}

			identifier = nonReserved
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
		return ast.NewInterfaceDeclaration(
			p.memoryGauge,
			access,
			compositeKind,
			identifier,
			conformances,
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

	// Skip the `attachment` keyword
	p.nextSemanticToken()

	identifier, err := p.mustIdentifier()
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

	if !p.isToken(p.current, lexer.TokenIdentifier, KeywordFor) {
		return nil, p.syntaxError(
			"expected 'for', got %s",
			p.current.Type,
		)
	}

	// skip the `for` keyword
	p.nextSemanticToken()

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

	p.skipSpaceAndComments()

	conformances, err := parseConformances(p)
	if err != nil {
		return nil, err
	}

	_, err = p.mustOne(lexer.TokenBraceOpen)
	if err != nil {
		return nil, err
	}

	p.skipSpaceAndComments()

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
//	                          | pragmaDeclaration
func parseMemberOrNestedDeclaration(p *parser, docString string) (ast.Declaration, error) {

	const functionBlockIsOptional = true

	var access ast.Access = ast.AccessNotSpecified
	var accessPos *ast.Position

	purity := ast.FunctionPurityUnspecified
	var purityPos *ast.Position

	var staticPos *ast.Position
	var nativePos *ast.Position

	var previousIdentifierToken *lexer.Token

	staticModifierEnabled := p.config.StaticModifierEnabled
	nativeModifierEnabled := p.config.NativeModifierEnabled

	for {
		p.skipSpaceAndComments()

		switch p.current.Type {
		case lexer.TokenIdentifier:

			if !p.config.IgnoreLeadingIdentifierEnabled &&
				previousIdentifierToken != nil {

				return nil, NewSyntaxError(
					previousIdentifierToken.StartPos,
					"unexpected %s",
					previousIdentifierToken.Type,
				)
			}

			switch string(p.currentTokenSource()) {
			case KeywordLet, KeywordVar:
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for variable")
				}
				return parseFieldWithVariableKind(
					p,
					access,
					accessPos,
					staticPos,
					nativePos,
					docString,
				)

			case KeywordCase:
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for enum case")
				}
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindEnumCase)
				if err != nil {
					return nil, err
				}
				return parseEnumCase(p, access, accessPos, docString)

			case KeywordFun:
				return parseFunctionDeclaration(
					p,
					functionBlockIsOptional,
					access,
					accessPos,
					purity,
					purityPos,
					staticPos,
					nativePos,
					docString,
				)

			case KeywordEvent:
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for event")
				}
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindEvent)
				if err != nil {
					return nil, err
				}
				return parseEventDeclaration(p, access, accessPos, docString)

			case KeywordStruct:
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for struct")
				}
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindStructure)
				if err != nil {
					return nil, err
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordResource:
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for resource")
				}
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindResource)
				if err != nil {
					return nil, err
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordContract:
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for contract")
				}
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindContract)
				if err != nil {
					return nil, err
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordEntitlement:
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindEntitlement)
				if err != nil {
					return nil, err
				}
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for entitlement")
				}
				return parseEntitlementOrMappingDeclaration(p, access, accessPos, docString)

			case KeywordEnum:
				if purity != ast.FunctionPurityUnspecified {
					return nil, NewSyntaxError(*purityPos, "invalid view modifier for enum")
				}
				err := rejectStaticAndNativeModifiers(p, staticPos, nativePos, common.DeclarationKindEnum)
				if err != nil {
					return nil, err
				}
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case keywordAttachment:
				return parseAttachmentDeclaration(p, access, accessPos, docString)

			case KeywordView:
				if purity != ast.FunctionPurityUnspecified {
					return nil, p.syntaxError("invalid second view modifier")
				}
				pos := p.current.StartPos
				purityPos = &pos
				purity = parsePurityAnnotation(p)
				continue

			case KeywordPub:
				return nil, p.syntaxErrorWithSuggestedFix("`pub` is no longer a valid access keyword", "`access(all)`")

			case KeywordPriv:
				return nil, p.syntaxErrorWithSuggestedFix("`priv` is no longer a valid access keyword", "`access(self)`")

			case KeywordAccess:
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

			case KeywordStatic:
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

			case KeywordNative:
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

			if p.config.IgnoreLeadingIdentifierEnabled &&
				previousIdentifierToken != nil {

				return nil, p.syntaxError("unexpected %s", p.current.Type)
			}

			t := p.current
			previousIdentifierToken = &t
			// Skip the identifier
			p.next()
			continue

		case lexer.TokenPragma:
			if previousIdentifierToken != nil {
				return nil, NewSyntaxError(
					previousIdentifierToken.StartPos,
					"unexpected token: %s",
					previousIdentifierToken.Type,
				)
			}
			err := rejectAllModifiers(p, access, accessPos, staticPos, nativePos, common.DeclarationKindPragma)
			if err != nil {
				return nil, err
			}
			return parsePragmaDeclaration(p)

		case lexer.TokenColon:
			if previousIdentifierToken == nil {
				return nil, p.syntaxError("unexpected %s", p.current.Type)
			}
			if purity != ast.FunctionPurityUnspecified {
				return nil, NewSyntaxError(*purityPos, "invalid view modifier for variable")
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
				purity,
				purityPos,
				staticPos,
				nativePos,
				identifier,
				docString,
			)
		}

		return nil, nil
	}
}

func rejectAllModifiers(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	staticPos *ast.Position,
	nativePos *ast.Position,
	kind common.DeclarationKind,
) error {
	if access != ast.AccessNotSpecified {
		return NewSyntaxError(*accessPos, "invalid access modifier for %s", kind.Name())
	}
	return rejectStaticAndNativeModifiers(p, staticPos, nativePos, kind)
}

func rejectStaticAndNativeModifiers(
	p *parser,
	staticPos *ast.Position,
	nativePos *ast.Position,
	kind common.DeclarationKind,
) error {
	if p.config.StaticModifierEnabled && staticPos != nil {
		return NewSyntaxError(*staticPos, "invalid static modifier for %s", kind.Name())
	}
	if p.config.NativeModifierEnabled && nativePos != nil {
		return NewSyntaxError(*nativePos, "invalid native modifier for %s", kind.Name())
	}
	return nil
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
	purity ast.FunctionPurity,
	purityPos *ast.Position,
	staticPos *ast.Position,
	nativePos *ast.Position,
	identifier ast.Identifier,
	docString string,
) (*ast.SpecialFunctionDeclaration, error) {

	startPos := ast.EarliestPosition(identifier.Pos, accessPos, purityPos, staticPos, nativePos)

	parameterList, returnTypeAnnotation, functionBlock, err :=
		parseFunctionParameterListAndRest(p, functionBlockIsOptional)
	if err != nil {
		return nil, err
	}

	declarationKind := common.DeclarationKindUnknown
	switch identifier.Identifier {
	case KeywordInit:
		declarationKind = common.DeclarationKindInitializer

	case KeywordDestroy:
		p.report(&CustomDestructorError{Pos: identifier.Pos})

	case KeywordPrepare:
		declarationKind = common.DeclarationKindPrepare
	}

	if returnTypeAnnotation != nil {
		var kindDescription string
		if declarationKind != common.DeclarationKindUnknown {
			kindDescription = declarationKind.Name()
		} else {
			kindDescription = "special function"
		}

		p.report(NewSyntaxError(
			returnTypeAnnotation.StartPos,
			"invalid return type for %s",
			kindDescription,
		))
	}

	return ast.NewSpecialFunctionDeclaration(
		p.memoryGauge,
		declarationKind,
		ast.NewFunctionDeclaration(
			p.memoryGauge,
			access,
			purity,
			staticPos != nil,
			nativePos != nil,
			identifier,
			nil,
			parameterList,
			nil,
			functionBlock,
			startPos,
			docString,
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
