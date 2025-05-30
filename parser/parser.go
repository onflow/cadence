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
	"bytes"
	"os"
	"strings"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/parser/lexer"
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
	// NativeModifierEnabled determines if the native modifier is enabled
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
	// TypeParametersEnabled determines if type parameters are enabled
	TypeParametersEnabled bool
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
	tokens, err := lexer.Lex(input, memoryGauge)
	if err != nil {
		errors = append(errors, err)
		return
	}
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
			var err error
			switch r := r.(type) {
			case ParseError:
				// Report parser errors.
				p.report(r)

			// Do not treat non-parser errors as syntax errors.
			// Also do not wrap non-parser errors, that are already
			// known cadence errors. i.e: internal errors / user errors.
			// e.g: `errors.MemoryError`
			case errors.UserError:
				err = r
			case errors.InternalError:
				err = r
			case error:
				// Any other error/panic is an internal error.
				// Thus, wrap with an UnexpectedError to mark it as an internal error
				// and propagate up the call stack.
				err = errors.NewUnexpectedErrorFromCause(r)
			default:
				err = errors.NewUnexpectedError("parser: %v", r)
			}

			var zero T
			result = zero
			errs = p.errors

			errs = append(errs, err)
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
		return result, p.errors
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

func (p *parser) syntaxErrorWithSuggestedFix(message string, suggestedFix string) error {
	return NewSyntaxErrorWithSuggestedReplacement(p.current.Range, message, suggestedFix)
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
				parseError = &SyntaxError{
					Pos:     token.StartPos,
					Message: err.Error(),
				}
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

	var insideLineDocString bool

	var atEnd bool
	progress := p.newProgress()

	for !atEnd && p.checkProgress(&progress) {

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

	identifier := ident.Identifier
	if IsHardKeyword(identifier) {
		return nonIdentifierErr("keyword " + identifier)
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

type parserProgress struct {
	offset int
}

func (p *parser) newProgress() parserProgress {
	return parserProgress{
		// -1, because the first call of checkProgress should succeed
		offset: p.current.StartPos.Offset - 1,
	}
}

// checkProgress checks that the parser has made progress since it was called last with this parserProgress.
func (p *parser) checkProgress(progress *parserProgress) bool {
	parserOffset := p.current.StartPos.Offset
	if parserOffset == progress.offset {
		panic(errors.NewUnexpectedError("parser did not make progress"))
	}
	progress.offset = parserOffset
	return true
}

func ParseExpression(
	memoryGauge common.MemoryGauge,
	input []byte,
	config Config,
) (
	expression ast.Expression,
	errs []error,
) {
	return Parse(
		memoryGauge,
		input,
		func(p *parser) (ast.Expression, error) {
			return parseExpression(p, lowestBindingPower)
		},
		config,
	)
}

func ParseStatements(
	memoryGauge common.MemoryGauge,
	input []byte,
	config Config,
) (
	statements []ast.Statement,
	errs []error,
) {
	return Parse(
		memoryGauge,
		input,
		func(p *parser) ([]ast.Statement, error) {
			return parseStatements(p, nil)
		},
		config,
	)
}

func ParseStatementsFromTokenStream(
	memoryGauge common.MemoryGauge,
	tokens lexer.TokenStream,
	config Config,
) (
	statements []ast.Statement,
	errs []error,
) {
	return ParseTokenStream(
		memoryGauge,
		tokens,
		func(p *parser) ([]ast.Statement, error) {
			return parseStatements(p, nil)
		},
		config,
	)
}

func ParseType(memoryGauge common.MemoryGauge, input []byte, config Config) (ty ast.Type, errs []error) {
	return Parse(
		memoryGauge,
		input,
		func(p *parser) (ast.Type, error) {
			return parseType(p, lowestBindingPower)
		},
		config,
	)
}

func ParseDeclarations(
	memoryGauge common.MemoryGauge,
	input []byte,
	config Config,
) (
	declarations []ast.Declaration,
	errs []error,
) {
	return Parse(
		memoryGauge,
		input,
		func(p *parser) ([]ast.Declaration, error) {
			return parseDeclarations(p, lexer.TokenEOF)
		},
		config,
	)
}

func ParseArgumentList(
	memoryGauge common.MemoryGauge,
	input []byte,
	config Config,
) (
	arguments ast.Arguments,
	errs []error,
) {
	return Parse(
		memoryGauge,
		input,
		func(p *parser) (ast.Arguments, error) {
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
}

func ParseProgram(memoryGauge common.MemoryGauge, code []byte, config Config) (program *ast.Program, err error) {
	tokens, err := lexer.Lex(code, memoryGauge)
	if err != nil {
		return
	}
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
	declarations, errs := ParseTokenStream(
		memoryGauge,
		input,
		func(p *parser) ([]ast.Declaration, error) {
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
