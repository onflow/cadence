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
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

func parseDeclarations(p *parser, endTokenType lexer.TokenType) (declarations []ast.Declaration) {
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
			declaration := parseDeclaration(p, docString)
			if declaration == nil {
				return
			}

			declarations = append(declarations, declaration)
		}
	}
}

func parseDeclaration(p *parser, docString string) ast.Declaration {

	access := ast.AccessNotSpecified
	var accessPos *ast.Position

	for {
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenPragma:
			return parsePragmaDeclaration(p)
		case lexer.TokenIdentifier:
			switch p.current.Value {
			case keywordLet, keywordVar:
				return parseVariableDeclaration(p, access, accessPos, docString)

			case keywordFun:
				return parseFunctionDeclaration(p, false, access, accessPos, docString)

			case keywordImport:
				return parseImportDeclaration(p)

			case keywordEvent:
				return parseEventDeclaration(p, access, accessPos, docString)

			case keywordStruct, keywordResource, keywordContract, keywordEnum:
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case KeywordTransaction:
				if access != ast.AccessNotSpecified {
					panic(fmt.Errorf("invalid access modifier for transaction"))
				}
				return parseTransactionDeclaration(p, docString)

			case keywordPriv, keywordPub, keywordAccess:
				if access != ast.AccessNotSpecified {
					panic(fmt.Errorf("unexpected access modifier"))
				}
				pos := p.current.StartPos
				accessPos = &pos
				access = parseAccess(p)
				continue
			}
		}

		return nil
	}
}

// parseAccess parses an access modifier
//
//     access
//         : 'priv'
//         | 'pub' ( '(' 'set' ')' )?
//         | 'access' '(' ( 'self' | 'contract' | 'account' | 'all' ) ')'
//
func parseAccess(p *parser) ast.Access {

	switch p.current.Value {
	case keywordPriv:
		// Skip the `priv` keyword
		p.next()
		return ast.AccessPrivate

	case keywordPub:
		// Skip the `pub` keyword
		p.next()
		p.skipSpaceAndComments(true)
		if !p.current.Is(lexer.TokenParenOpen) {
			return ast.AccessPublic
		}

		// Skip the opening paren
		p.next()
		p.skipSpaceAndComments(true)

		if !p.current.Is(lexer.TokenIdentifier) {
			panic(fmt.Errorf(
				"expected keyword %q, got %s",
				keywordSet,
				p.current.Type,
			))
		}
		if p.current.Value != keywordSet {
			panic(fmt.Errorf(
				"expected keyword %q, got %q",
				keywordSet,
				p.current.Value,
			))
		}

		// Skip the `set` keyword
		p.next()
		p.skipSpaceAndComments(true)

		p.mustOne(lexer.TokenParenClose)

		return ast.AccessPublicSettable

	case keywordAccess:
		// Skip the `access` keyword
		p.next()
		p.skipSpaceAndComments(true)

		p.mustOne(lexer.TokenParenOpen)

		p.skipSpaceAndComments(true)

		if !p.current.Is(lexer.TokenIdentifier) {
			panic(fmt.Errorf(
				"expected keyword %s, got %s",
				common.EnumerateWords(
					[]string{
						strconv.Quote(keywordAll),
						strconv.Quote(keywordAccount),
						strconv.Quote(keywordContract),
						strconv.Quote(keywordSelf),
					},
					"or",
				),
				p.current.Type,
			))
		}

		var access ast.Access

		switch p.current.Value {
		case keywordAll:
			access = ast.AccessPublic

		case keywordAccount:
			access = ast.AccessAccount

		case keywordContract:
			access = ast.AccessContract

		case keywordSelf:
			access = ast.AccessPrivate

		default:
			panic(fmt.Errorf(
				"expected keyword %s, got %q",
				common.EnumerateWords(
					[]string{
						strconv.Quote(keywordAll),
						strconv.Quote(keywordAccount),
						strconv.Quote(keywordContract),
						strconv.Quote(keywordSelf),
					},
					"or",
				),
				p.current.Value,
			))
		}

		// Skip the keyword
		p.next()
		p.skipSpaceAndComments(true)

		p.mustOne(lexer.TokenParenClose)

		return access

	default:
		panic(errors.NewUnreachableError())
	}
}

// parseVariableDeclaration parses a variable declaration.
//
//     variableKind : 'var' | 'let'
//
//     variableDeclaration :
//         variableKind identifier ( ':' typeAnnotation )?
//         transfer expression
//         ( transfer expression )?
//
func parseVariableDeclaration(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) *ast.VariableDeclaration {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	isLet := p.current.Value == keywordLet

	// Skip the `let` or `var` keyword
	p.next()

	p.skipSpaceAndComments(true)
	if !p.current.Is(lexer.TokenIdentifier) {
		panic(fmt.Errorf(
			"expected identifier after start of variable declaration, got %s",
			p.current.Type,
		))
	}

	identifier := tokenToIdentifier(p.current)

	// Skip the identifier
	p.next()
	p.skipSpaceAndComments(true)

	var typeAnnotation *ast.TypeAnnotation

	if p.current.Is(lexer.TokenColon) {
		// Skip the colon
		p.next()
		p.skipSpaceAndComments(true)

		typeAnnotation = parseTypeAnnotation(p)
	}

	p.skipSpaceAndComments(true)
	transfer := parseTransfer(p)
	if transfer == nil {
		panic(fmt.Errorf("expected transfer"))
	}

	value := parseExpression(p, lowestBindingPower)

	p.skipSpaceAndComments(true)

	secondTransfer := parseTransfer(p)
	var secondValue ast.Expression
	if secondTransfer != nil {
		secondValue = parseExpression(p, lowestBindingPower)
	}

	variableDeclaration := &ast.VariableDeclaration{
		Access:         access,
		IsConstant:     isLet,
		Identifier:     identifier,
		TypeAnnotation: typeAnnotation,
		Value:          value,
		Transfer:       transfer,
		StartPos:       startPos,
		SecondTransfer: secondTransfer,
		SecondValue:    secondValue,
		DocString:      docString,
	}

	castingExpression, leftIsCasting := value.(*ast.CastingExpression)
	if leftIsCasting {
		castingExpression.ParentVariableDeclaration = variableDeclaration
	}

	return variableDeclaration
}

// parseTransfer parses a transfer.
//
//     transfer : '=' | '<-' | '<-!'
//
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

	return &ast.Transfer{
		Operation: operation,
		Pos:       pos,
	}
}

func parsePragmaDeclaration(p *parser) *ast.PragmaDeclaration {
	startPos := p.current.StartPosition()
	p.next()
	expr := parseExpression(p, lowestBindingPower)
	return &ast.PragmaDeclaration{
		Range: ast.Range{
			StartPos: startPos,
			EndPos:   expr.EndPosition(),
		},
		Expression: expr,
	}
}

// parseImportDeclaration parses an import declaration
//
//     importDeclaration :
//         'import'
//         ( identifier (',' identifier)* 'from' )?
//         ( string | hexadecimalLiteral | identifier )
//
func parseImportDeclaration(p *parser) *ast.ImportDeclaration {

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
			parsedString, errs := parseStringLiteral(p.current.Value.(string))
			p.report(errs...)
			location = common.StringLocation(parsedString)

		case lexer.TokenHexadecimalIntegerLiteral:
			location = parseHexadecimalLocation(p.current.Value.(string))

		default:
			panic(errors.NewUnreachableError())
		}

		// Skip the location
		p.next()
	}

	setIdentifierLocation := func(identifier ast.Identifier) {
		location = common.IdentifierLocation(identifier.Identifier)
		locationPos = identifier.Pos
		endPos = identifier.EndPosition()
	}

	parseLocation := func() {
		switch p.current.Type {
		case lexer.TokenString, lexer.TokenHexadecimalIntegerLiteral:
			parseStringOrAddressLocation()

		case lexer.TokenIdentifier:
			identifier := tokenToIdentifier(p.current)
			setIdentifierLocation(identifier)
			p.next()

		default:
			panic(fmt.Errorf(
				"unexpected token in import declaration: got %s, expected string, address, or identifier",
				p.current.Type,
			))
		}
	}

	parseMoreIdentifiers := func() {
		expectCommaOrFrom := false

		atEnd := false
		for !atEnd {
			p.next()
			p.skipSpaceAndComments(true)

			switch p.current.Type {
			case lexer.TokenComma:
				if !expectCommaOrFrom {
					panic(fmt.Errorf(
						"expected %s or keyword %q, got %s",
						lexer.TokenIdentifier,
						keywordFrom,
						p.current.Type,
					))
				}
				expectCommaOrFrom = false

			case lexer.TokenIdentifier:

				if p.current.Value == keywordFrom {
					if expectCommaOrFrom {
						atEnd = true

						// Skip the `from` keyword
						p.next()
						p.skipSpaceAndComments(true)

						parseLocation()
						break
					}

					if !isNextTokenCommaOrFrom(p) {
						panic(fmt.Errorf(
							"expected %s, got keyword %q",
							lexer.TokenIdentifier,
							p.current.Value,
						))
					}

					// If the next token is either comma or 'from' token, then fall through
					// and process the current 'from' token as an identifier.
				}

				identifier := tokenToIdentifier(p.current)
				identifiers = append(identifiers, identifier)

				expectCommaOrFrom = true

			case lexer.TokenEOF:
				panic(fmt.Errorf(
					"unexpected end in import declaration: expected %s or %s",
					lexer.TokenIdentifier,
					lexer.TokenComma,
				))

			default:
				panic(fmt.Errorf(
					"unexpected token in import declaration: got %s, expected keyword %q or %s",
					p.current.Type,
					keywordFrom,
					lexer.TokenComma,
				))
			}
		}
	}

	maybeParseFromIdentifier := func(identifier ast.Identifier) {
		// The current identifier is maybe the `from` keyword,
		// in which case the given (previous) identifier was
		// an imported identifier and not the import location.
		//
		// If it is not the `from` keyword,
		// the given (previous) identifier is the import location.

		if p.current.Value == keywordFrom {
			identifiers = append(identifiers, identifier)
			// Skip the `from` keyword
			p.next()
			p.skipSpaceAndComments(true)

			parseLocation()

		} else {
			setIdentifierLocation(identifier)
		}
	}

	// Skip the `import` keyword
	p.next()
	p.skipSpaceAndComments(true)

	switch p.current.Type {
	case lexer.TokenString, lexer.TokenHexadecimalIntegerLiteral:
		parseStringOrAddressLocation()

	case lexer.TokenIdentifier:
		identifier := tokenToIdentifier(p.current)
		// Skip the identifier
		p.next()
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenComma:
			// The previous identifier is an imported identifier,
			// not the import location
			identifiers = append(identifiers, identifier)
			parseMoreIdentifiers()

		case lexer.TokenIdentifier:
			maybeParseFromIdentifier(identifier)

		case lexer.TokenEOF:
			// The previous identifier is the identifier location
			setIdentifierLocation(identifier)

		default:
			panic(fmt.Errorf(
				"unexpected token in import declaration: got %s, expected keyword %q or %s",
				p.current.Type,
				keywordFrom,
				lexer.TokenComma,
			))
		}

	case lexer.TokenEOF:
		panic(fmt.Errorf("unexpected end in import declaration: expected string, address, or identifier"))

	default:
		panic(fmt.Errorf(
			"unexpected token in import declaration: got %s, expected string, address, or identifier",
			p.current.Type,
		))
	}

	return &ast.ImportDeclaration{
		Identifiers: identifiers,
		Location:    location,
		Range: ast.Range{
			StartPos: startPosition,
			EndPos:   endPos,
		},
		LocationPos: locationPos,
	}
}

// isNextTokenCommaOrFrom check whether the token to follow is a comma or a from token.
func isNextTokenCommaOrFrom(p *parser) bool {
	p.startBuffering()
	defer p.replayBuffered()

	// skip the current token
	p.next()
	p.skipSpaceAndComments(true)

	// Lookahead the next token
	switch p.current.Type {
	case lexer.TokenIdentifier:
		return p.current.Value == keywordFrom
	case lexer.TokenComma:
		return true
	default:
		return false
	}
}

func parseHexadecimalLocation(literal string) common.AddressLocation {
	bytes := []byte(strings.Replace(literal[2:], "_", "", -1))

	length := len(bytes)
	if length%2 == 1 {
		bytes = append([]byte{'0'}, bytes...)
		length++
	}

	address := make([]byte, hex.DecodedLen(length))
	_, err := hex.Decode(address, bytes)
	if err != nil {
		// unreachable, hex literal should always be valid
		panic(err)
	}

	return common.AddressLocation{
		Address: common.BytesToAddress(address),
	}
}

// parseEventDeclaration parses an event declaration.
//
//     eventDeclaration : 'event' identifier parameterList
//
func parseEventDeclaration(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) *ast.CompositeDeclaration {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	// Skip the `event` keyword
	p.next()

	p.skipSpaceAndComments(true)
	if !p.current.Is(lexer.TokenIdentifier) {
		panic(fmt.Errorf(
			"expected identifier after start of event declaration, got %s",
			p.current.Type,
		))
	}

	identifier := tokenToIdentifier(p.current)
	// Skip the identifier
	p.next()

	parameterList := parseParameterList(p)

	initializer :=
		&ast.SpecialFunctionDeclaration{
			Kind: common.DeclarationKindInitializer,
			FunctionDeclaration: &ast.FunctionDeclaration{
				ParameterList: parameterList,
				StartPos:      parameterList.StartPos,
			},
		}

	members := ast.NewMembers([]ast.Declaration{
		initializer,
	})

	return &ast.CompositeDeclaration{
		Access:        access,
		CompositeKind: common.CompositeKindEvent,
		Identifier:    identifier,
		Members:       members,
		DocString:     docString,
		Range: ast.Range{
			StartPos: startPos,
			EndPos:   parameterList.EndPos,
		},
	}
}

// parseCompositeKind parses a composite kind.
//
//     compositeKind : 'struct' | 'resource' | 'contract' | 'enum'
//
func parseCompositeKind(p *parser) common.CompositeKind {

	if p.current.Is(lexer.TokenIdentifier) {
		switch p.current.Value {
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
//     variableKind : 'var' | 'let'
//
//     field : variableKind identifier ':' typeAnnotation
//
func parseFieldWithVariableKind(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) *ast.FieldDeclaration {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	var variableKind ast.VariableKind
	switch p.current.Value {
	case keywordLet:
		variableKind = ast.VariableKindConstant

	case keywordVar:
		variableKind = ast.VariableKindVariable
	}

	// Skip the `let` or `var` keyword
	p.next()

	p.skipSpaceAndComments(true)
	if !p.current.Is(lexer.TokenIdentifier) {
		panic(fmt.Errorf(
			"expected identifier after start of field declaration, got %s",
			p.current.Type,
		))
	}

	identifier := tokenToIdentifier(p.current)
	// Skip the identifier
	p.next()
	p.skipSpaceAndComments(true)

	p.mustOne(lexer.TokenColon)

	p.skipSpaceAndComments(true)

	typeAnnotation := parseTypeAnnotation(p)

	return &ast.FieldDeclaration{
		Access:         access,
		VariableKind:   variableKind,
		Identifier:     identifier,
		TypeAnnotation: typeAnnotation,
		DocString:      docString,
		Range: ast.Range{
			StartPos: startPos,
			EndPos:   typeAnnotation.EndPosition(),
		},
	}
}

// parseCompositeOrInterfaceDeclaration parses an event declaration.
//
//     conformances : ':' nominalType ( ',' nominalType )*
//
//     compositeDeclaration : compositeKind identifier conformances?
//                            '{' membersAndNestedDeclarations '}'
//
//     interfaceDeclaration : compositeKind 'interface' identifier conformances?
//                            '{' membersAndNestedDeclarations '}'
//
func parseCompositeOrInterfaceDeclaration(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) ast.Declaration {

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
		p.skipSpaceAndComments(true)
		if !p.current.Is(lexer.TokenIdentifier) {
			panic(fmt.Errorf(
				"expected %s, got %s",
				lexer.TokenIdentifier,
				p.current.Type,
			))
		}

		wasInterface := isInterface

		if p.current.Value == keywordInterface {
			isInterface = true
			if wasInterface {
				panic(fmt.Errorf(
					"expected interface name, got keyword %q",
					keywordInterface,
				))
			}
			// Skip the `interface` keyword
			p.next()
			continue
		} else {
			identifier = tokenToIdentifier(p.current)
			// Skip the identifier
			p.next()
			break
		}
	}

	p.skipSpaceAndComments(true)

	var conformances []*ast.NominalType

	if p.current.Is(lexer.TokenColon) {
		// Skip the colon
		p.next()

		conformances, _ = parseNominalTypes(p, lexer.TokenBraceOpen)

		if len(conformances) < 1 {
			panic(fmt.Errorf(
				"expected at least one conformance after %s",
				lexer.TokenColon,
			))
		}
	}

	p.skipSpaceAndComments(true)

	p.mustOne(lexer.TokenBraceOpen)

	members := parseMembersAndNestedDeclarations(p, lexer.TokenBraceClose)

	p.skipSpaceAndComments(true)

	endToken := p.mustOne(lexer.TokenBraceClose)

	declarationRange := ast.Range{
		StartPos: startPos,
		EndPos:   endToken.EndPos,
	}

	if isInterface {
		// TODO: remove once interface conformances are supported
		if len(conformances) > 0 {
			// TODO: improve
			panic(fmt.Errorf("unexpected conformances"))
		}

		return &ast.InterfaceDeclaration{
			Access:        access,
			CompositeKind: compositeKind,
			Identifier:    identifier,
			Members:       members,
			DocString:     docString,
			Range:         declarationRange,
		}
	} else {
		return &ast.CompositeDeclaration{
			Access:        access,
			CompositeKind: compositeKind,
			Identifier:    identifier,
			Conformances:  conformances,
			Members:       members,
			DocString:     docString,
			Range:         declarationRange,
		}
	}
}

// parseMembersAndNestedDeclarations parses composite or interface members,
// and nested declarations.
//
//     membersAndNestedDeclarations : ( memberOrNestedDeclaration ';'* )*
//
func parseMembersAndNestedDeclarations(p *parser, endTokenType lexer.TokenType) *ast.Members {

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
			return ast.NewMembers(declarations)

		default:
			memberOrNestedDeclaration := parseMemberOrNestedDeclaration(p, docString)
			if memberOrNestedDeclaration == nil {
				return ast.NewMembers(declarations)
			}

			declarations = append(declarations, memberOrNestedDeclaration)
		}
	}
}

// parseMemberOrNestedDeclaration parses a composite or interface member,
// or a declaration nested in it.
//
//     memberOrNestedDeclaration : field
//                               | specialFunctionDeclaration
//                               | functionDeclaration
//                               | interfaceDeclaration
//                               | compositeDeclaration
//                               | eventDeclaration
//                               | enumCase
//
func parseMemberOrNestedDeclaration(p *parser, docString string) ast.Declaration {

	const functionBlockIsOptional = true

	access := ast.AccessNotSpecified
	var accessPos *ast.Position

	var previousIdentifierToken *lexer.Token

	for {
		p.skipSpaceAndComments(true)

		switch p.current.Type {
		case lexer.TokenIdentifier:
			switch p.current.Value {
			case keywordLet, keywordVar:
				return parseFieldWithVariableKind(p, access, accessPos, docString)

			case keywordCase:
				return parseEnumCase(p, access, accessPos, docString)

			case keywordFun:
				return parseFunctionDeclaration(p, functionBlockIsOptional, access, accessPos, docString)

			case keywordEvent:
				return parseEventDeclaration(p, access, accessPos, docString)

			case keywordStruct, keywordResource, keywordContract, keywordEnum:
				return parseCompositeOrInterfaceDeclaration(p, access, accessPos, docString)

			case keywordPriv, keywordPub, keywordAccess:
				if access != ast.AccessNotSpecified {
					panic(fmt.Errorf("unexpected access modifier"))
				}
				pos := p.current.StartPos
				accessPos = &pos
				access = parseAccess(p)
				continue

			default:
				if previousIdentifierToken != nil {
					panic(fmt.Errorf("unexpected %s", p.current.Type))
				}

				t := p.current
				previousIdentifierToken = &t
				// Skip the identifier
				p.next()
				continue
			}

		case lexer.TokenColon:
			if previousIdentifierToken == nil {
				panic(fmt.Errorf("unexpected %s", p.current.Type))
			}

			identifier := tokenToIdentifier(*previousIdentifierToken)
			return parseFieldDeclarationWithoutVariableKind(p, access, accessPos, identifier, docString)

		case lexer.TokenParenOpen:
			if previousIdentifierToken == nil {
				panic(fmt.Errorf("unexpected %s", p.current.Type))
			}

			identifier := tokenToIdentifier(*previousIdentifierToken)
			return parseSpecialFunctionDeclaration(p, functionBlockIsOptional, access, accessPos, identifier)
		}

		return nil
	}
}

func parseFieldDeclarationWithoutVariableKind(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	identifier ast.Identifier,
	docString string,
) *ast.FieldDeclaration {

	startPos := identifier.Pos
	if accessPos != nil {
		startPos = *accessPos
	}

	p.mustOne(lexer.TokenColon)

	p.skipSpaceAndComments(true)

	typeAnnotation := parseTypeAnnotation(p)

	return &ast.FieldDeclaration{
		Access:         access,
		VariableKind:   ast.VariableKindNotSpecified,
		Identifier:     identifier,
		TypeAnnotation: typeAnnotation,
		DocString:      docString,
		Range: ast.Range{
			StartPos: startPos,
			EndPos:   typeAnnotation.EndPosition(),
		},
	}
}

func parseSpecialFunctionDeclaration(
	p *parser,
	functionBlockIsOptional bool,
	access ast.Access,
	accessPos *ast.Position,
	identifier ast.Identifier,
) *ast.SpecialFunctionDeclaration {

	startPos := identifier.Pos
	if accessPos != nil {
		startPos = *accessPos
	}

	// TODO: switch to parseFunctionParameterListAndRest once old parser is deprecated:
	//   allow a return type annotation while parsing, but reject later.

	parameterList := parseParameterList(p)

	p.skipSpaceAndComments(true)

	var functionBlock *ast.FunctionBlock

	if !functionBlockIsOptional ||
		p.current.Is(lexer.TokenBraceOpen) {

		functionBlock = parseFunctionBlock(p)
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

	return &ast.SpecialFunctionDeclaration{
		Kind: declarationKind,
		FunctionDeclaration: &ast.FunctionDeclaration{
			Access:        access,
			Identifier:    identifier,
			ParameterList: parameterList,
			FunctionBlock: functionBlock,
			StartPos:      startPos,
		},
	}
}

// parseEnumCase parses a field which has a variable kind.
//
//     enumCase : 'case' identifier
//
func parseEnumCase(
	p *parser,
	access ast.Access,
	accessPos *ast.Position,
	docString string,
) *ast.EnumCaseDeclaration {

	startPos := p.current.StartPos
	if accessPos != nil {
		startPos = *accessPos
	}

	// Skip the `enum` keyword
	p.next()

	p.skipSpaceAndComments(true)
	if !p.current.Is(lexer.TokenIdentifier) {
		panic(fmt.Errorf(
			"expected identifier after start of enum case declaration, got %s",
			p.current.Type,
		))
	}

	identifier := tokenToIdentifier(p.current)
	// Skip the identifier
	p.next()

	return &ast.EnumCaseDeclaration{
		Access:     access,
		Identifier: identifier,
		DocString:  docString,
		StartPos:   startPos,
	}
}
