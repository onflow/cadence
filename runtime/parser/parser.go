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
	"bytes"
	"os"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser/lexer"
)

// expressionDepthLimit is the limit of how deeply nested an expression can get
const expressionDepthLimit = 1 << 4

// typeDepthLimit is the limit of how deeply nested a type can get
const typeDepthLimit = 1 << 4

// lowestBindingPower is the lowest binding power.
// The binding power controls operator precedence:
// the higher the value, the tighter a token binds to the tokens that follow.

const lowestBindingPower = 0

type Config struct {
	// StaticModifierEnabled determines if the static modifier is enabled
	StaticModifierEnabled bool
	// NativeModifierEnabled determines if the static modifier is enabled
	NativeModifierEnabled bool
	// Deprecated: IgnoreLeadingIdentifierEnabled determines
	// if leading identifiers are ignored.
	//
	// Pre-Stable Cadence, identifiers preceding keywords were (incorrectly) ignored,
	// instead of being reported as invalid, e.g. `foo let bar: Int` was valid.
	// The new default behaviour is to report an error, e.g. for `foo` in the example above.
	//
	// This option exists so the old behaviour can be enabled to allow developers to update their code.
	IgnoreLeadingIdentifierEnabled bool
}

type parser struct {
	// tokens is a stream of tokens from the lexer
	tokens lexer.TokenStream
	// memoryGauge is used for metering memory usage
	memoryGauge common.MemoryGauge
	// errors are the parsing errors encountered during parsing
	errors []error
	// backtrackingCursorStack is the stack of lexer cursors used when backtracking
	backtrackingCursorStack []int
	// bufferedErrorsStack is the stack of parsing errors encountered during buffering
	bufferedErrorsStack [][]error
	// current is the current token being parsed
	current lexer.Token
	// localReplayedTokensCount is the number of replayed tokens since starting the top-most ambiguity.
	// Reset when the top-most ambiguity starts and ends. This keeps errors local
	localReplayedTokensCount uint
	// globalReplayedTokensCount is the number of replayed tokens since starting the parse.
	// It is never reset
	globalReplayedTokensCount uint
	// ambiguityLevel is the current level of ambiguity (nesting)
	ambiguityLevel int
	// expressionDepth is the depth of the currently parsed expression (if >0)
	expressionDepth int
	// typeDepth is the depth of the type (if >0)
	typeDepth int
	// config enables certain features
	config Config
}

// Parse creates a lexer to scan the given input string,
// and uses the given `parse` function to parse tokens into a result.
//
// It can be composed with different parse functions to parse the input string into different results.
// See "ParseExpression", "ParseStatements" as examples.
func Parse[T any](
	memoryGauge common.MemoryGauge,
	input []byte,
	parse func(*parser) (T, error),
	config Config,
) (result T, errors []error) {
	// create a lexer, which turns the input string into tokens
	tokens := lexer.Lex(input, memoryGauge)
	defer tokens.Reclaim()
	return ParseTokenStream(
		memoryGauge,
		tokens,
		parse,
		config,
	)
}

func ParseTokenStream[T any](
	memoryGauge common.MemoryGauge,
	tokens lexer.TokenStream,
	parse func(*parser) (T, error),
	config Config,
) (
	result T,
	errs []error,
) {
	p := &parser{
		config:      config,
		tokens:      tokens,
		memoryGauge: memoryGauge,
	}

	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case ParseError:
				// Report parser errors.
				p.report(r)

			// Do not treat non-parser errors as syntax errors.
			case errors.InternalError, errors.UserError:
				// Also do not wrap non-parser errors, that are already
				// known cadence errors. i.e: internal errors / user errors.
				// e.g: `errors.MemoryError`
				panic(r)
			case error:
				// Any other error/panic is an internal error.
				// Thus, wrap with an UnexpectedError to mark it as an internal error
				// and propagate up the call stack.
				panic(errors.NewUnexpectedErrorFromCause(r))
			default:
				panic(errors.NewUnexpectedError("parser: %v", r))
			}

			var zero T
			result = zero
			errs = p.errors
		}

		for _, bufferedErrors := range p.bufferedErrorsStack {
			errs = append(errs, bufferedErrors...)
		}
	}()

	startPos := ast.NewPosition(
		p.memoryGauge,
		0,
		1,
		0,
	)

	p.current = lexer.Token{
		Type: lexer.TokenEOF,
		Range: ast.NewRange(
			p.memoryGauge,
			startPos,
			startPos,
		),
	}

	// Get the initial token
	p.next()

	result, err := parse(p)
	if err != nil {
		p.report(err)
		var zero T
		return zero, p.errors
	}

	p.skipSpaceAndComments()

	if !p.current.Is(lexer.TokenEOF) {
		p.reportSyntaxError("unexpected token: %s", p.current.Type)
	}

	return result, p.errors
}

func (p *parser) syntaxError(message string, params ...any) error {
	return NewSyntaxError(p.current.StartPos, message, params...)
}

func (p *parser) reportSyntaxError(message string, params ...any) {
	err := p.syntaxError(message, params...)
	p.report(err)
}

func (p *parser) report(errs ...error) {
	for _, err := range errs {

		// Only `ParserError`s must be reported.
		// If the reported error is not a parse error, then it's an internal error (go runtime errors),
		// or a fatal error (e.g: MemoryError)
		// Hence, terminate parsing.
		parseError, ok := err.(ParseError)
		if !ok {
			panic(err)
		}

		// Add the errors to the buffered errors if buffering,
		// or the final errors if not

		bufferedErrorsDepth := len(p.bufferedErrorsStack)
		if bufferedErrorsDepth > 0 {
			bufferedErrorsIndex := bufferedErrorsDepth - 1
			p.bufferedErrorsStack[bufferedErrorsIndex] = append(
				p.bufferedErrorsStack[bufferedErrorsIndex],
				parseError,
			)
		} else {
			p.errors = append(p.errors, parseError)
		}
	}
}

// next reads the next token and marks it as the "current" token.
// The next token could either be read from the lexer or from
// the buffer.
// Tokens are buffered when syntax ambiguity is involved.
func (p *parser) next() {

	for {
		token := p.tokens.Next()

		if token.Is(lexer.TokenError) {
			// Report error token as error, skip.
			err, ok := token.SpaceOrError.(error)
			// we just checked that this is an error token
			if !ok {
				panic(errors.NewUnreachableError())
			}
			parseError, ok := err.(ParseError)
			if !ok {
				parseError = NewSyntaxError(
					token.StartPos,
					err.Error(),
				)
			}
			p.report(parseError)
			continue
		}

		p.current = token

		return
	}
}

// nextSemanticToken advances past the current token to the next semantic token.
// It skips whitespace, including newlines, and comments
func (p *parser) nextSemanticToken() {
	p.next()
	p.skipSpaceAndComments()
}

func (p *parser) mustOne(tokenType lexer.TokenType) (lexer.Token, error) {
	t := p.current
	if !t.Is(tokenType) {
		return lexer.Token{}, p.syntaxError("expected token %s", tokenType)
	}
	p.next()
	return t, nil
}

func (p *parser) tokenSource(token lexer.Token) []byte {
	input := p.tokens.Input()
	return token.Source(input)
}

func (p *parser) currentTokenSource() []byte {
	return p.tokenSource(p.current)
}

func (p *parser) isToken(token lexer.Token, tokenType lexer.TokenType, expected string) bool {
	if !token.Is(tokenType) {
		return false
	}

	actual := p.tokenSource(token)
	return string(actual) == expected
}

func (p *parser) mustToken(tokenType lexer.TokenType, string string) (lexer.Token, error) {
	t := p.current
	if !p.isToken(t, tokenType, string) {
		return lexer.Token{}, p.syntaxError("expected token %s with string value %s", tokenType, string)
	}
	p.next()
	return t, nil
}

func (p *parser) startBuffering() {
	// Push the lexer's previous cursor to the stack.
	// When start buffering is called, the lexer has already advanced to the next token
	p.backtrackingCursorStack = append(p.backtrackingCursorStack, p.tokens.Cursor()-1)

	// Push an empty slice of errors to the stack
	p.bufferedErrorsStack = append(p.bufferedErrorsStack, nil)
}

func (p *parser) acceptBuffered() {
	// Pop the last backtracking cursor from the stack
	// and ignore it

	lastIndex := len(p.backtrackingCursorStack) - 1
	p.backtrackingCursorStack = p.backtrackingCursorStack[:lastIndex]

	// Pop the last buffered errors from the stack.
	//
	// The element type is a slice (reference type),
	// so we need to replace the slice with nil explicitly
	// to free the memory.
	// The slice's underlying storage would otherwise
	// keep a reference to it and prevent it from being garbage collected.

	lastIndex = len(p.bufferedErrorsStack) - 1
	bufferedErrors := p.bufferedErrorsStack[lastIndex]
	p.bufferedErrorsStack[lastIndex] = nil
	p.bufferedErrorsStack = p.bufferedErrorsStack[:lastIndex]

	// Apply the accepted buffered errors to the last errors on the buffered errors stack,
	// or the final errors, if we reached the bottom of the stack
	// (i.e. this acceptance disables buffering)

	if len(p.bufferedErrorsStack) > 0 {
		p.bufferedErrorsStack[lastIndex-1] = append(
			p.bufferedErrorsStack[lastIndex-1],
			bufferedErrors...,
		)
	} else {
		p.errors = append(
			p.errors,
			bufferedErrors...,
		)
	}
}

// localTokenReplayCountLimit is a sensible limit for how many tokens may be replayed
// until the top-most ambiguity ends.
const localTokenReplayCountLimit = 1 << 6

// globalTokenReplayCountLimit is a sensible limit for how many tokens may be replayed
// during a parse
const globalTokenReplayCountLimit = 1 << 10

func (p *parser) checkReplayCount(total, additional, limit uint, kind string) (uint, error) {
	newTotal := total + additional
	// Check for overflow (uint) and for exceeding the limit
	if newTotal < total || newTotal > limit {
		return newTotal, p.syntaxError("program too ambiguous, %s replay limit of %d tokens exceeded", kind, limit)
	}
	return newTotal, nil
}

func (p *parser) replayBuffered() error {

	cursor := p.tokens.Cursor()

	// Pop the last backtracking cursor from the stack
	// and revert the lexer back to it

	lastIndex := len(p.backtrackingCursorStack) - 1
	backtrackCursor := p.backtrackingCursorStack[lastIndex]

	replayedCount := uint(cursor - backtrackCursor)

	var err error

	p.localReplayedTokensCount, err = p.checkReplayCount(
		p.localReplayedTokensCount,
		replayedCount,
		localTokenReplayCountLimit,
		"local",
	)
	if err != nil {
		return err
	}

	p.globalReplayedTokensCount, err = p.checkReplayCount(
		p.globalReplayedTokensCount,
		replayedCount,
		globalTokenReplayCountLimit,
		"global",
	)
	if err != nil {
		return err
	}

	p.tokens.Revert(backtrackCursor)
	p.next()
	p.backtrackingCursorStack = p.backtrackingCursorStack[:lastIndex]

	// Pop the last buffered errors from the stack
	// and ignore them

	lastIndex = len(p.bufferedErrorsStack) - 1
	p.bufferedErrorsStack[lastIndex] = nil
	p.bufferedErrorsStack = p.bufferedErrorsStack[:lastIndex]

	return nil
}

type triviaOptions struct {
	skipNewlines    bool
	parseDocStrings bool
}

// skipSpaceAndComments skips whitespace, including newlines, and comments
func (p *parser) skipSpaceAndComments() (containsNewline bool) {
	containsNewline, _ = p.parseTrivia(triviaOptions{
		skipNewlines: true,
	})
	return
}

var blockCommentDocStringPrefix = []byte("/**")
var lineCommentDocStringPrefix = []byte("///")

func (p *parser) parseTrivia(options triviaOptions) (containsNewline bool, docString string) {
	var docStringBuilder strings.Builder
	defer func() {
		if options.parseDocStrings {
			docString = docStringBuilder.String()
		}
	}()

	var atEnd, insideLineDocString bool

	for !atEnd {
		switch p.current.Type {
		case lexer.TokenSpace:
			space, ok := p.current.SpaceOrError.(lexer.Space)
			// we just checked that this is a space
			if !ok {
				panic(errors.NewUnreachableError())
			}

			if space.ContainsNewline {
				containsNewline = true
			}

			if containsNewline && !options.skipNewlines {
				return
			}

			p.next()

		case lexer.TokenBlockCommentStart:
			commentStartOffset := p.current.StartPos.Offset
			endToken, ok := p.parseBlockComment()

			if ok && options.parseDocStrings {
				commentEndOffset := endToken.EndPos.Offset

				contentWithPrefix := p.tokens.Input()[commentStartOffset : commentEndOffset-1]

				insideLineDocString = false
				docStringBuilder.Reset()
				if bytes.HasPrefix(contentWithPrefix, blockCommentDocStringPrefix) {
					// Strip prefix (`/**`)
					docStringBuilder.Write(contentWithPrefix[len(blockCommentDocStringPrefix):])
				}
			}

		case lexer.TokenLineComment:
			if options.parseDocStrings {
				comment := p.currentTokenSource()
				if bytes.HasPrefix(comment, lineCommentDocStringPrefix) {
					if insideLineDocString {
						docStringBuilder.WriteByte('\n')
					} else {
						insideLineDocString = true
						docStringBuilder.Reset()
					}
					// Strip prefix
					docStringBuilder.Write(comment[len(lineCommentDocStringPrefix):])
				} else {
					insideLineDocString = false
					docStringBuilder.Reset()
				}
			}

			p.next()

		default:
			atEnd = true
		}
	}
	return
}

func (p *parser) mustIdentifier() (ast.Identifier, error) {
	identifier, err := p.mustOne(lexer.TokenIdentifier)
	if err != nil {
		return ast.Identifier{}, err
	}

	return p.tokenToIdentifier(identifier), nil
}

// Attempt to downcast a Token into an identifier, erroring out if the identifier is a hard keyword. See keywords.HardKeywords.
func (p *parser) mustNotKeyword(errMsgContext string, token lexer.Token) (ast.Identifier, error) {
	nonIdentifierErr := func(invalidTokenMsg string) (ast.Identifier, error) {
		if len(errMsgContext) > 0 {
			errMsgContext = " " + errMsgContext
		}

		return ast.Identifier{}, p.syntaxError("expected identifier%s, got %s", errMsgContext, invalidTokenMsg)
	}

	if token.Type != lexer.TokenIdentifier {
		return nonIdentifierErr(token.Type.String())
	}

	ident := p.tokenToIdentifier(token)

	if _, ok := hardKeywordsTable.Lookup(ident.Identifier); ok {
		return nonIdentifierErr("keyword " + ident.Identifier)
	}
	return ident, nil
}

// Attempt to parse an identifier that's not a hard keyword.
func (p *parser) nonReservedIdentifier(errMsgContext string) (ast.Identifier, error) {
	return p.mustNotKeyword(errMsgContext, p.current)
}

func (p *parser) tokenToIdentifier(token lexer.Token) ast.Identifier {
	return ast.NewIdentifier(
		p.memoryGauge,
		string(p.tokenSource(token)),
		token.StartPos,
	)
}

func (p *parser) startAmbiguity() {
	if p.ambiguityLevel == 0 {
		p.localReplayedTokensCount = 0
	}
	p.ambiguityLevel++
}

func (p *parser) endAmbiguity() {
	p.ambiguityLevel--
	if p.ambiguityLevel == 0 {
		p.localReplayedTokensCount = 0
	}
}

func ParseExpression(memoryGauge common.MemoryGauge, input []byte, config Config) (expression ast.Expression, errs []error) {
	var res any
	res, errs = Parse(
		memoryGauge,
		input,
		func(p *parser) (any, error) {
			return parseExpression(p, lowestBindingPower)
		},
		config,
	)
	if res == nil {
		expression = nil
		return
	}
	expression, ok := res.(ast.Expression)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return
}

func ParseStatements(
	memoryGauge common.MemoryGauge,
	input []byte,
	config Config,
) (
	statements []ast.Statement,
	errs []error,
) {
	var res any
	res, errs = Parse(
		memoryGauge,
		input,
		func(p *parser) (any, error) {
			return parseStatements(p, nil)
		},
		config,
	)
	if res == nil {
		statements = nil
		return
	}

	statements, ok := res.([]ast.Statement)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return
}

func ParseType(memoryGauge common.MemoryGauge, input []byte, config Config) (ty ast.Type, errs []error) {
	var res any
	res, errs = Parse(
		memoryGauge,
		input,
		func(p *parser) (any, error) {
			return parseType(p, lowestBindingPower)
		},
		config,
	)
	if res == nil {
		ty = nil
		return
	}

	ty, ok := res.(ast.Type)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return
}

func ParseDeclarations(
	memoryGauge common.MemoryGauge,
	input []byte,
	config Config,
) (
	declarations []ast.Declaration,
	errs []error,
) {
	var res any
	res, errs = Parse(
		memoryGauge,
		input,
		func(p *parser) (any, error) {
			return parseDeclarations(p, lexer.TokenEOF)
		},
		config,
	)
	if res == nil {
		declarations = nil
		return
	}

	declarations, ok := res.([]ast.Declaration)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return
}

func ParseArgumentList(
	memoryGauge common.MemoryGauge,
	input []byte,
	config Config,
) (
	arguments ast.Arguments,
	errs []error,
) {
	var res any
	res, errs = Parse(
		memoryGauge,
		input,
		func(p *parser) (any, error) {
			p.skipSpaceAndComments()

			_, err := p.mustOne(lexer.TokenParenOpen)
			if err != nil {
				return nil, err
			}

			arguments, _, err := parseArgumentListRemainder(p)
			return arguments, err
		},
		config,
	)
	if res == nil {
		arguments = nil
		return
	}

	arguments, ok := res.([]*ast.Argument)

	if !ok {
		panic(errors.NewUnreachableError())
	}
	return
}

func ParseProgram(memoryGauge common.MemoryGauge, code []byte, config Config) (program *ast.Program, err error) {
	tokens := lexer.Lex(code, memoryGauge)
	defer tokens.Reclaim()
	return ParseProgramFromTokenStream(memoryGauge, tokens, config)
}

func ParseProgramFromTokenStream(
	memoryGauge common.MemoryGauge,
	input lexer.TokenStream,
	config Config,
) (
	program *ast.Program,
	err error,
) {
	var res any
	var errs []error
	res, errs = ParseTokenStream(
		memoryGauge,
		input,
		func(p *parser) (any, error) {
			return parseDeclarations(p, lexer.TokenEOF)
		},
		config,
	)
	if len(errs) > 0 {
		err = Error{
			Code:   input.Input(),
			Errors: errs,
		}
	}
	if res == nil {
		program = nil
		return
	}

	declarations, ok := res.([]ast.Declaration)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	program = ast.NewProgram(memoryGauge, declarations)

	return
}

func ParseProgramFromFile(
	memoryGauge common.MemoryGauge,
	filename string,
	config Config,
) (
	program *ast.Program,
	code []byte,
	err error,
) {
	var data []byte
	data, err = os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	program, err = ParseProgram(memoryGauge, data, config)
	if err != nil {
		return nil, code, err
	}
	return program, code, nil
}
