/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"strings"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/parser/lexer"
)

func parseDeclarations(p *parser, endTokenType lexer.TokenType) (declarations []ast.Declaration, err error) {
	progress := p.newProgress()

	for p.checkProgress(&progress) {

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

	panic(errors.NewUnreachableError())
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

	progress := p.newProgress()

	for p.checkProgress(&progress) {

		p.skipSpaceAndComments()

		switch p.current.Type {
		case lexer.TokenPragma:
			rejectAllModifiers(p, access, accessPos, staticPos, nativePos, purityPos, common.DeclarationKindPragma)
			return parsePragmaDeclaration(p)

		case lexer.TokenIdentifier:
			switch string(p.currentTokenSource()) {

			case KeywordLet:
				const isLet = true
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindVariable)
				return parseVariableDeclaration(p, access, accessPos, isLet, docString)

			case KeywordVar:
				const isLet = false
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindVariable)
				return parseVariableDeclaration(p, access, accessPos, isLet, docString)

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
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindImport)
				return parseImportDeclaration(p)

			case KeywordEvent:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindEvent)
				return parseEventDeclaration(p, access, accessPos, docString)

			case KeywordStruct:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindStructure)
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordResource:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindResource)
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordEntitlement:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindEntitlement)
				return parseEntitlementOrMappingDeclaration(p, access, accessPos, docString)

			case KeywordAttachment:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindAttachment)
				return parseAttachmentDeclaration(p, access, accessPos, docString)

			case KeywordContract:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindContract)
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordEnum:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindEnum)
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordTransaction:
				rejectAllModifiers(p, access, accessPos, staticPos, nativePos, purityPos, common.DeclarationKindTransaction)
				return parseTransactionDeclaration(p, docString)

			case KeywordView:
				if purity != ast.FunctionPurityUnspecified {
					p.report(&DuplicateViewModifierError{
						Range: p.current.Range,
					})
				}

				pos := p.current.StartPos
				purityPos = &pos
				purity = parsePurityAnnotation(p)
				continue

			case KeywordPub:
				handlePub(p)
				continue

			case KeywordPriv:
				handlePriv(p)
				continue

			case KeywordAccess:
				previousAccess := access
				if staticModifierEnabled && staticPos != nil {
					p.reportSyntaxError("invalid access modifier after `static` modifier")
				}
				if nativeModifierEnabled && nativePos != nil {
					p.reportSyntaxError("invalid access modifier after `native` modifier")
				}
				pos := p.current.StartPos
				accessPos = &pos
				var (
					accessRange ast.Range
					err         error
				)
				access, accessRange, err = parseAccess(p)
				if err != nil {
					return nil, err
				}
				if previousAccess != ast.AccessNotSpecified {
					p.report(&DuplicateAccessModifierError{
						Range: accessRange,
					})
				}

				continue

			case KeywordStatic:
				if !staticModifierEnabled {
					break
				}

				if staticPos != nil {
					p.reportSyntaxError("invalid second `static` modifier")
				}
				if nativeModifierEnabled && nativePos != nil {
					p.reportSyntaxError("invalid `static` modifier after `native` modifier")
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
					p.reportSyntaxError("invalid second `native` modifier")
				}
				pos := p.current.StartPos
				nativePos = &pos
				p.next()
				continue
			}
		}

		return nil, nil
	}

	panic(errors.NewUnreachableError())
}

func handlePriv(p *parser) {
	p.report(&PrivAccessError{
		Range: p.current.Range,
	})
	p.next()
}

func handlePub(p *parser) {
	pubToken := p.current

	p.nextSemanticToken()

	// Try to parse `(set)` if given
	if !p.current.Is(lexer.TokenParenOpen) {
		p.report(&PubAccessError{
			Range: pubToken.Range,
		})
		return
	}

	// Skip the opening paren
	p.nextSemanticToken()

	var (
		keyword  string
		endToken lexer.Token
	)

	if p.current.Type == lexer.TokenIdentifier {

		keyword = string(p.currentTokenSource())

		endToken = p.current

		p.next()
	}

	p.skipSpaceAndComments()

	if p.current.Is(lexer.TokenParenClose) {
		endToken = p.current
		// Skip the closing paren
		p.next()
	}

	r := ast.NewRange(
		p.memoryGauge,
		pubToken.StartPos,
		endToken.EndPos,
	)

	const keywordSet = "set"
	if keyword == keywordSet {
		p.report(&PubSetAccessError{
			Range: r,
		})
	} else {
		p.report(&PubAccessError{
			Range: r,
		})
	}
}

func rejectAccessKeywordEntitlementType(p *parser, ty *ast.NominalType) {
	switch ty.Identifier.Identifier {
	case KeywordAll, KeywordAccess, KeywordAccount, KeywordSelf:
		p.report(&AccessKeywordEntitlementNameError{
			Keyword: ty.Identifier.Identifier,
			Range:   ast.NewRangeFromPositioned(p.memoryGauge, ty),
		})
	}
}

func parseEntitlementList(p *parser) (ast.EntitlementSet, error) {
	firstType, err := parseNominalType(p)
	if err != nil {
		return nil, err
	}
	rejectAccessKeywordEntitlementType(p, firstType)

	p.skipSpaceAndComments()
	entitlements := []*ast.NominalType{firstType}
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
		p.report(&InvalidEntitlementSeparatorError{
			Token: p.current,
		})
		// Assume comma separator and continue parsing
		separator = lexer.TokenComma
	}

	remainingEntitlements, _, err := parseNominalTypes(p, lexer.TokenParenClose, separator)
	if err != nil {
		return nil, err
	}

	for _, entitlement := range remainingEntitlements {
		rejectAccessKeywordEntitlementType(p, entitlement)
		entitlements = append(entitlements, entitlement)
	}

	switch separator {
	case lexer.TokenComma:
		return ast.NewConjunctiveEntitlementSet(entitlements), nil

	case lexer.TokenVerticalBar:
		return ast.NewDisjunctiveEntitlementSet(entitlements), nil

	default:
		panic(errors.NewUnexpectedError("unexpected separator: %s", separator))
	}
}

// parseAccess parses an access modifier
//
//	access : 'access' '(' ( 'self' | 'contract' | 'account' | 'all' | entitlementList ) ')'
func parseAccess(p *parser) (ast.Access, ast.Range, error) {

	var accessRange ast.Range

	accessRange.StartPos = p.current.StartPos

	// Skip the `access` keyword
	p.nextSemanticToken()

	if p.current.Is(lexer.TokenParenOpen) {
		p.nextSemanticToken()
	} else {
		p.report(&MissingAccessOpeningParenError{
			GotToken: p.current,
		})
	}

	if !p.current.Is(lexer.TokenIdentifier) {
		return ast.AccessNotSpecified, ast.EmptyRange, &MissingAccessKeywordError{
			GotToken: p.current,
		}
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

		entitlementMapName, err := parseNominalType(p)
		if err != nil {
			return ast.AccessNotSpecified, ast.EmptyRange, err
		}
		access = ast.NewMappedAccess(entitlementMapName, keywordPos)

		p.skipSpaceAndComments()

	default:
		entitlements, err := parseEntitlementList(p)
		if err != nil {
			return ast.AccessNotSpecified, ast.EmptyRange, err
		}
		access = ast.NewEntitlementAccess(entitlements)
	}

	accessRange.EndPos = p.current.EndPos

	if p.current.Is(lexer.TokenParenClose) {
		p.next()
	} else {
		p.report(&MissingAccessClosingParenError{
			GotToken: p.current,
		})
	}

	return access, accessRange, nil
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
	isLet bool,
	docString string,
) (*ast.VariableDeclaration, error) {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

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
		p.nextSemanticToken()

		typeAnnotation, err = parseTypeAnnotation(p)
		if err != nil {
			return nil, err
		}
	}

	p.skipSpaceAndComments()

	transfer := parseTransfer(p)
	if transfer == nil {
		p.report(&MissingTransferError{
			Pos: p.current.StartPos,
		})
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
//	    ( identifier ('as' identifier)? (',' identifier ('as' identifier)?)* 'from' )?
//	    ( string | hexadecimalLiteral | identifier )
func parseImportDeclaration(p *parser) (*ast.ImportDeclaration, error) {

	startPosition := p.current.StartPos

	var imports []ast.Import

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
			return &InvalidImportLocationError{
				GotToken: p.current,
			}
		}

		return nil
	}

	parseMoreIdentifiers := func() error {
		expectCommaOrFrom := true

		var atEnd bool
		progress := p.newProgress()

		for !atEnd && p.checkProgress(&progress) {
			switch p.current.Type {
			case lexer.TokenComma:
				if !expectCommaOrFrom {
					return &InvalidTokenInImportListError{
						GotToken: p.current,
					}
				}
				expectCommaOrFrom = false
				p.nextSemanticToken()

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
						return &InvalidFromKeywordAsIdentifierError{
							GotToken: p.current,
						}
					}

					// If the next token is either comma or 'from' token, then fall through
					// and process the current 'from' token as an identifier.
				}

				identifier := p.tokenToIdentifier(p.current)

				// Skip the identifier
				p.nextSemanticToken()

				// Parse optional alias
				alias := parseOptionalImportAlias(p)

				p.skipSpaceAndComments()

				imports = append(
					imports,
					ast.Import{
						Identifier: identifier,
						Alias:      alias,
					},
				)

				expectCommaOrFrom = true

			case lexer.TokenEOF:
				return &UnexpectedEOFInImportListError{
					Pos: p.current.StartPos,
				}

			default:
				return &InvalidImportContinuationError{
					GotToken: p.current,
				}
			}
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

		// Parse optional alias
		alias := parseOptionalImportAlias(p)

		p.skipSpaceAndComments()

		switch p.current.Type {
		case lexer.TokenComma:
			// The previous identifier is an imported identifier,
			// not the import location
			imports = append(
				imports,
				ast.Import{
					Identifier: identifier,
					Alias:      alias,
				},
			)

			err := parseMoreIdentifiers()
			if err != nil {
				return nil, err
			}

		case lexer.TokenIdentifier:
			// The current identifier is maybe the `from` keyword,
			// in which case the given (previous) identifier was
			// an imported identifier and not the import location.
			//
			// If it is not the `from` keyword,
			// the given (previous) identifier is the import location.

			if string(p.currentTokenSource()) == KeywordFrom {
				imports = append(
					imports,
					ast.Import{
						Identifier: identifier,
						Alias:      alias,
					},
				)

				// Skip the `from` keyword
				p.nextSemanticToken()

				err := parseLocation()
				if err != nil {
					return nil, err
				}
			} else {
				setIdentifierLocation(identifier)
			}

		case lexer.TokenEOF:
			// The previous identifier is the identifier location
			setIdentifierLocation(identifier)

		default:
			return nil, &InvalidImportContinuationError{
				GotToken: p.current,
			}
		}

	case lexer.TokenEOF:
		return nil, &MissingImportLocationError{
			Pos: p.current.StartPos,
		}

	default:
		return nil, &InvalidImportLocationError{
			GotToken: p.current,
		}
	}

	return ast.NewImportDeclaration(
		p.memoryGauge,
		imports,
		location,
		ast.NewRange(
			p.memoryGauge,
			startPosition,
			endPos,
		),
		locationPos,
	), nil
}

func parseOptionalImportAlias(p *parser) ast.Identifier {
	if !p.isToken(p.current, lexer.TokenIdentifier, KeywordAs) {
		return ast.Identifier{}
	}

	// Skip the `as` keyword
	p.nextSemanticToken()

	if p.current.Type != lexer.TokenIdentifier {
		p.report(&InvalidTokenInImportAliasError{
			GotToken: p.current,
		})
		return ast.Identifier{}
	}

	alias := p.tokenToIdentifier(p.current)

	// Skip the alias
	p.next()

	return alias
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
		p.report(&SyntaxError{
			Pos:     p.current.StartPos,
			Message: err.Error(),
		})
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
			access,
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

func checkAndReportFieldInitialization(p *parser) {
	if p.current.Type == lexer.TokenSpace {
		p.next()
	}

	if p.current.Is(lexer.TokenEqual) {
		equalPos := p.current.StartPos
		p.nextSemanticToken()

		initExpression, err := parseExpression(p, lowestBindingPower)
		if err != nil {
			return
		}

		p.report(&FieldInitializationError{
			Range: ast.Range{
				StartPos: equalPos,
				EndPos:   initExpression.EndPosition(p.memoryGauge),
			},
		})
	}
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

	var identifier ast.Identifier

	if p.current.Is(lexer.TokenIdentifier) {
		identifier = p.tokenToIdentifier(p.current)
		// Skip the identifier
		p.nextSemanticToken()
	} else {
		p.report(&MissingFieldNameError{
			GotToken: p.current,
		})
	}

	if p.current.Is(lexer.TokenColon) {
		p.nextSemanticToken()
	} else {
		p.report(&MissingColonAfterFieldNameError{
			GotToken: p.current,
		})
	}

	typeAnnotation, err := parseTypeAnnotation(p)
	if err != nil {
		return nil, err
	}

	checkAndReportFieldInitialization(p)

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
func parseEntitlementMapping(p *parser) (*ast.EntitlementMapRelation, error) {
	inputType, err := parseType(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}
	inputNominalType, ok := inputType.(*ast.NominalType)
	if !ok {
		p.report(&InvalidEntitlementMappingTypeError{
			Range: ast.NewRangeFromPositioned(p.memoryGauge, inputType),
		})
	}

	p.skipSpaceAndComments()

	if p.current.Is(lexer.TokenRightArrow) {
		p.nextSemanticToken()
	} else {
		p.report(&MissingRightArrowInEntitlementMappingError{
			GotToken: p.current,
		})
	}

	outputType, err := parseType(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	outputNominalType, ok := outputType.(*ast.NominalType)
	if !ok {
		p.report(&InvalidEntitlementMappingTypeError{
			Range: ast.NewRangeFromPositioned(p.memoryGauge, outputType),
		})
	}

	p.skipSpaceAndComments()

	return ast.NewEntitlementMapRelation(
		p.memoryGauge,
		inputNominalType,
		outputNominalType,
	), nil
}

// parseEntitlementMappings parses entitlement mappings
func parseEntitlementMappingsAndInclusions(p *parser, endTokenType lexer.TokenType) ([]ast.EntitlementMapElement, error) {
	var elements []ast.EntitlementMapElement

	progress := p.newProgress()

	for p.checkProgress(&progress) {

		p.skipSpaceAndComments()

		switch p.current.Type {

		case endTokenType, lexer.TokenEOF:
			return elements, nil

		default:
			if p.isToken(p.current, lexer.TokenIdentifier, KeywordInclude) {
				// Skip the `include` keyword
				p.nextSemanticToken()

				includedType, err := parseType(p, lowestBindingPower)
				if err != nil {
					return nil, err
				}

				includedNominalType, ok := includedType.(*ast.NominalType)
				if !ok {
					p.report(&InvalidEntitlementMappingIncludeTypeError{
						Range: ast.NewRangeFromPositioned(p.memoryGauge, includedType),
					})
				}

				elements = append(elements, includedNominalType)
			} else {
				mapping, err := parseEntitlementMapping(p)
				if err != nil {
					return nil, err
				}

				elements = append(elements, mapping)
			}
		}
	}

	panic(errors.NewUnreachableError())
}

func parseDeclarationBraces[T any](
	p *parser,
	kind common.DeclarationKind,
	f func() (T, error),
) (result T, endToken lexer.Token, err error) {

	parseDeclarationOpeningBrace(p, kind)

	result, err = f()
	if err != nil {
		return
	}

	p.skipSpaceAndComments()

	endToken = p.current

	parseDeclarationClosingBrace(p, kind)

	return
}

func parseDeclarationOpeningBrace(p *parser, kind common.DeclarationKind) {
	if p.current.Is(lexer.TokenBraceOpen) {
		p.next()
	} else {
		p.report(&DeclarationMissingOpeningBraceError{
			Kind:     kind,
			GotToken: p.current,
		})
	}
}

func parseDeclarationClosingBrace(p *parser, kind common.DeclarationKind) {
	if p.current.Is(lexer.TokenBraceClose) {
		p.next()
	} else {
		p.report(&DeclarationMissingClosingBraceError{
			Kind:     kind,
			GotToken: p.current,
		})
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
		return nil, p.newSyntaxError(
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

		elements, endToken, err := parseDeclarationBraces(
			p,
			common.DeclarationKindEntitlementMapping,
			func() ([]ast.EntitlementMapElement, error) {
				return parseEntitlementMappingsAndInclusions(p, lexer.TokenBraceClose)
			},
		)
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
			identifier.EndPosition(p.memoryGauge),
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
		p.next()

		conformances, _, err = parseNominalTypes(p, lexer.TokenBraceOpen, lexer.TokenComma)
		if err != nil {
			return nil, err
		}

		if len(conformances) < 1 {
			p.report(&MissingConformanceError{
				Pos: p.current.StartPos,
			})
		}
	}

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

	progress := p.newProgress()

	for p.checkProgress(&progress) {

		p.skipSpaceAndComments()

		if !p.current.Is(lexer.TokenIdentifier) {
			return nil, p.newSyntaxError(
				"expected %s, got %s",
				lexer.TokenIdentifier,
				p.current.Type,
			)
		}

		wasInterface := isInterface

		if string(p.currentTokenSource()) == KeywordInterface {
			isInterface = true
			if wasInterface {
				return nil, &InvalidInterfaceNameError{
					GotToken: p.current,
				}
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

	members, endToken, err := parseDeclarationBraces(
		p,
		compositeKind.DeclarationKind(isInterface),
		func() (*ast.Members, error) {
			return parseMembersAndNestedDeclarations(p, lexer.TokenBraceClose)
		},
	)
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

	if p.isToken(p.current, lexer.TokenIdentifier, KeywordFor) {
		// Skip the `for` keyword
		p.nextSemanticToken()
	} else {
		p.report(&MissingForKeywordInAttachmentDeclarationError{
			GotToken: p.current,
		})
	}

	baseType, err := parseType(p, lowestBindingPower)
	if err != nil {
		return nil, err
	}

	baseNominalType, ok := baseType.(*ast.NominalType)
	if !ok {
		p.report(&InvalidAttachmentBaseTypeError{
			Range: ast.NewRangeFromPositioned(p.memoryGauge, baseType),
		})
	}

	p.skipSpaceAndComments()

	conformances, err := parseConformances(p)
	if err != nil {
		return nil, err
	}

	members, endToken, err := parseDeclarationBraces(
		p,
		common.DeclarationKindAttachment,
		func() (*ast.Members, error) {
			return parseMembersAndNestedDeclarations(p, lexer.TokenBraceClose)
		},
	)
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

	progress := p.newProgress()

	for p.checkProgress(&progress) {

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

	panic(errors.NewUnreachableError())
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

	progress := p.newProgress()

	for p.checkProgress(&progress) {

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
				rejectPurityModifier(p, purityPos, common.DeclarationKindField)
				return parseFieldWithVariableKind(
					p,
					access,
					accessPos,
					staticPos,
					nativePos,
					docString,
				)

			case KeywordCase:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindEnumCase)
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
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindEvent)
				return parseEventDeclaration(p, access, accessPos, docString)

			case KeywordStruct:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindStructure)
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordResource:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindResource)
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordContract:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindContract)
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordEntitlement:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindEntitlement)
				return parseEntitlementOrMappingDeclaration(p, access, accessPos, docString)

			case KeywordEnum:
				rejectNonAccessModifiers(p, staticPos, nativePos, purityPos, common.DeclarationKindEnum)
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordAttachment:
				return parseAttachmentDeclaration(p, access, accessPos, docString)

			case KeywordView:
				if purity != ast.FunctionPurityUnspecified {
					p.report(&DuplicateViewModifierError{
						Range: p.current.Range,
					})
				}
				pos := p.current.StartPos
				purityPos = &pos
				purity = parsePurityAnnotation(p)
				continue

			case KeywordPub:
				handlePub(p)
				continue

			case KeywordPriv:
				handlePriv(p)
				continue

			case KeywordAccess:
				previousAccess := access
				if staticModifierEnabled && staticPos != nil {
					p.reportSyntaxError("invalid access modifier after `static` modifier")
				}
				if nativeModifierEnabled && nativePos != nil {
					p.reportSyntaxError("invalid access modifier after `native` modifier")
				}
				pos := p.current.StartPos
				accessPos = &pos
				var (
					accessRange ast.Range
					err         error
				)
				access, accessRange, err = parseAccess(p)
				if err != nil {
					return nil, err
				}
				if previousAccess != ast.AccessNotSpecified {
					p.report(&DuplicateAccessModifierError{
						Range: accessRange,
					})
				}
				continue

			case KeywordStatic:
				if !staticModifierEnabled {
					break
				}

				if staticPos != nil {
					p.reportSyntaxError("invalid second `static` modifier")
				}
				if nativeModifierEnabled && nativePos != nil {
					p.reportSyntaxError("invalid `static` modifier after `native` modifier")
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
					p.reportSyntaxError("invalid second `native` modifier")
				}
				pos := p.current.StartPos
				nativePos = &pos
				p.next()
				continue
			}

			if p.config.IgnoreLeadingIdentifierEnabled &&
				previousIdentifierToken != nil {

				return nil, p.newSyntaxError("unexpected %s", p.current.Type)
			}

			t := p.current
			previousIdentifierToken = &t
			// Skip the identifier
			p.next()
			continue

		case lexer.TokenPragma:
			if previousIdentifierToken != nil {
				// TODO: Add documentation link
				return nil, NewSyntaxError(
					previousIdentifierToken.StartPos,
					"unexpected token: %s",
					previousIdentifierToken.Type,
				).
					WithSecondary("remove the identifier before the pragma declaration")
			}
			rejectAllModifiers(p, access, accessPos, staticPos, nativePos, purityPos, common.DeclarationKindPragma)
			return parsePragmaDeclaration(p)

		case lexer.TokenColon:
			if previousIdentifierToken == nil {
				return nil, p.newSyntaxError("unexpected %s", p.current.Type).
					WithSecondary("expected an identifier before the colon").
					WithDocumentation("https://cadence-lang.org/docs/language/glossary#-colon")
			}
			rejectPurityModifier(p, purityPos, common.DeclarationKindField)
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
				return nil, p.newSyntaxError("unexpected %s", p.current.Type).
					WithSecondary("expected an identifier before the opening parenthesis").
					WithDocumentation("https://cadence-lang.org/docs/language/types-and-type-system/composite-types")
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

	panic(errors.NewUnreachableError())
}

func rejectAllModifiers(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	staticPos *ast.Position,
	nativePos *ast.Position,
	purityPos *ast.Position,
	kind common.DeclarationKind,
) {
	if access != ast.AccessNotSpecified {
		p.report(&InvalidAccessModifierError{
			Pos:             *accessPos,
			DeclarationKind: kind,
		})
	}
	rejectNonAccessModifiers(
		p,
		staticPos,
		nativePos,
		purityPos,
		kind,
	)
}

func rejectNonAccessModifiers(
	p *parser,
	staticPos *ast.Position,
	nativePos *ast.Position,
	purityPos *ast.Position,
	kind common.DeclarationKind,
) {
	if p.config.StaticModifierEnabled && staticPos != nil {
		p.report(&InvalidStaticModifierError{
			Pos:             *staticPos,
			DeclarationKind: kind,
		})
	}
	if p.config.NativeModifierEnabled && nativePos != nil {
		p.report(&InvalidNativeModifierError{
			Pos:             *nativePos,
			DeclarationKind: kind,
		})
	}
	rejectPurityModifier(p, purityPos, kind)
}

func rejectPurityModifier(
	p *parser,
	purityPos *ast.Position,
	kind common.DeclarationKind,
) {
	if purityPos != nil {
		p.report(&InvalidViewModifierError{
			Pos:             *purityPos,
			DeclarationKind: kind,
		})
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

	if p.current.Is(lexer.TokenColon) {
		p.nextSemanticToken()
	} else {
		// Impossible, as parseFieldDeclarationWithoutVariableKind is currently only called
		// when the current token is a colon, but report an error for consistency / future changes
		p.report(&MissingColonAfterFieldNameError{
			GotToken: p.current,
		})
	}

	typeAnnotation, err := parseTypeAnnotation(p)
	if err != nil {
		return nil, err
	}

	checkAndReportFieldInitialization(p)

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
		// Calculate the full range of the destructor function
		var endPos ast.Position
		if functionBlock != nil {
			endPos = functionBlock.EndPosition(p.memoryGauge)
		} else {
			endPos = identifier.Pos // fallback
		}
		p.report(&CustomDestructorError{
			Pos:             identifier.Pos,
			DestructorRange: ast.NewRange(p.memoryGauge, startPos, endPos),
		})

	case KeywordPrepare:
		declarationKind = common.DeclarationKindPrepare
	}

	if returnTypeAnnotation != nil {
		p.report(&SpecialFunctionReturnTypeError{
			DeclarationKind: declarationKind,
			Range:           ast.NewRangeFromPositioned(p.memoryGauge, returnTypeAnnotation),
		})
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
		return nil, &MissingEnumCaseNameError{
			GotToken: p.current,
		}
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
