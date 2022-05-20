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

package parser2

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

// lowestBindingPower is the lowest binding power.
// The binding power controls operator precedence:
// the higher the value, the tighter a token binds to the tokens that follow.

const lowestBindingPower = 0

type parser struct {
	// tokens is a stream of tokens from the lexer
	tokens lexer.TokenStream
	// current is the current token being parsed.
	current lexer.Token
	// errors are the parsing errors encountered during parsing
	errors []error
	// backtrackingCursorStack is the stack of lexer cursors used when backtracking
	backtrackingCursorStack []int
	// bufferedErrorsStack is the stack of parsing errors encountered during buffering
	bufferedErrorsStack [][]error
	// memoryGauge is used for metering memory usage
	memoryGauge common.MemoryGauge
	// replayedTokensCount is the number of replayed tokens since starting the initial buffering.
	// It is reset when the buffered tokens get accepted. This keeps errors local.
	replayedTokensCount uint
}

// Parse creates a lexer to scan the given input string,
// and uses the given `parse` function to parse tokens into a result.
//
// It can be composed with different parse functions to parse the input string into different results.
// See "ParseExpression", "ParseStatements" as examples.
//
func Parse(input string, parse func(*parser) interface{}, memoryGauge common.MemoryGauge) (result interface{}, errors []error) {
	// create a lexer, which turns the input string into tokens
	tokens := lexer.Lex(input, memoryGauge)
	defer tokens.Reclaim()
	return ParseTokenStream(memoryGauge, tokens, parse)
}

func ParseTokenStream(
	memoryGauge common.MemoryGauge,
	tokens lexer.TokenStream,
	parse func(*parser) interface{},
) (
	result interface{},
	errors []error,
) {
	p := &parser{
		tokens:      tokens,
		memoryGauge: memoryGauge,
	}

	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("parser: %v", r)
			}

			p.report(err)

			result = nil
			errors = p.errors
		}

		for _, bufferedErrors := range p.bufferedErrorsStack {
			errors = append(errors, bufferedErrors...)
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

	result = parse(p)

	if !p.current.Is(lexer.TokenEOF) {
		p.report(fmt.Errorf("unexpected token: %s", p.current.Type))
	}

	return result, p.errors
}

func (p *parser) report(errs ...error) {
	for _, err := range errs {

		// If the reported error is not yet a parse error,
		// create a `SyntaxError` at the current position

		var ok bool

		// Fatal errors should abort parsing
		_, ok = err.(common.FatalError)
		if ok {
			panic(err)
		}

		var parseError ParseError
		parseError, ok = err.(ParseError)
		if !ok {
			parseError = &SyntaxError{
				Pos:     p.current.StartPos,
				Message: err.Error(),
			}
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
			err, ok := token.Value.(error)
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

func (p *parser) mustOne(tokenType lexer.TokenType) lexer.Token {
	t := p.current
	if !t.Is(tokenType) {
		panic(fmt.Errorf("expected token %s", tokenType))
	}
	p.next()
	return t
}

func (p *parser) mustOneString(tokenType lexer.TokenType, string string) lexer.Token {
	t := p.current
	if !t.IsString(tokenType, string) {
		panic(fmt.Errorf("expected token %s with string value %s", tokenType, string))
	}
	p.next()
	return t
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

	// Reset the replayed tokens count
	p.replayedTokensCount = 0
}

// tokenReplayLimit is a sensible limit for how many tokens may be replayed
// until the replay buffer is accepted.
const tokenReplayLimit = 2 << 12

func (p *parser) replayBuffered() {

	cursor := p.tokens.Cursor()

	// Pop the last backtracking cursor from the stack
	// and revert the lexer back to it

	lastIndex := len(p.backtrackingCursorStack) - 1
	backtrackCursor := p.backtrackingCursorStack[lastIndex]

	replayedCount := p.replayedTokensCount + uint(cursor-backtrackCursor)
	// Check for overflow (uint) and for exceeding the limit
	if replayedCount < p.replayedTokensCount || replayedCount > tokenReplayLimit {
		panic(fmt.Errorf("program too ambiguous, replay limit of %d tokens exceeded", tokenReplayLimit))
	}
	p.replayedTokensCount = replayedCount

	p.tokens.Revert(backtrackCursor)
	p.next()
	p.backtrackingCursorStack = p.backtrackingCursorStack[:lastIndex]

	// Pop the last buffered errors from the stack
	// and ignore them

	lastIndex = len(p.bufferedErrorsStack) - 1
	p.bufferedErrorsStack[lastIndex] = nil
	p.bufferedErrorsStack = p.bufferedErrorsStack[:lastIndex]
}

type triviaOptions struct {
	skipNewlines    bool
	parseDocStrings bool
}

func (p *parser) skipSpaceAndComments(skipNewlines bool) (containsNewline bool) {
	containsNewline, _ = p.parseTrivia(triviaOptions{
		skipNewlines: skipNewlines,
	})
	return containsNewline
}

func (p *parser) parseTrivia(options triviaOptions) (containsNewline bool, docString string) {
	var docStringBuilder strings.Builder
	defer func() {
		if options.parseDocStrings {
			docString = docStringBuilder.String()
		}
	}()

	atEnd := false
	inLineDocString := false

	for !atEnd {
		switch p.current.Type {
		case lexer.TokenSpace:
			space, ok := p.current.Value.(lexer.Space)
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
			comment := p.parseCommentContent()
			if options.parseDocStrings {
				inLineDocString = false
				docStringBuilder.Reset()
				if strings.HasPrefix(comment, "/**") {
					// Strip prefix and suffix (`*/`)
					docStringBuilder.WriteString(comment[3 : len(comment)-2])
				}
			}

		case lexer.TokenLineComment:
			if options.parseDocStrings {
				comment, ok := p.current.Value.(string)
				if !ok {
					// we just checked that this is a comment
					panic(errors.NewUnreachableError())
				}
				if strings.HasPrefix(comment, "///") {
					if inLineDocString {
						docStringBuilder.WriteRune('\n')
					} else {
						inLineDocString = true
						docStringBuilder.Reset()
					}
					// Strip prefix
					docStringBuilder.WriteString(comment[3:])
				} else {
					inLineDocString = false
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

func (p *parser) mustIdentifier() ast.Identifier {
	identifier := p.mustOne(lexer.TokenIdentifier)
	return p.tokenToIdentifier(identifier)
}

func (p *parser) tokenToIdentifier(identifier lexer.Token) ast.Identifier {
	return ast.NewIdentifier(
		p.memoryGauge,
		identifier.Value.(string),
		identifier.StartPos,
	)
}

func ParseExpression(input string, memoryGauge common.MemoryGauge) (expression ast.Expression, errs []error) {
	var res interface{}
	res, errs = Parse(
		input,
		func(p *parser) interface{} {
			return parseExpression(p, lowestBindingPower)
		},
		memoryGauge,
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

func ParseStatements(input string, memoryGauge common.MemoryGauge) (statements []ast.Statement, errs []error) {
	var res interface{}
	res, errs = Parse(
		input,
		func(p *parser) interface{} {
			return parseStatements(p, nil)
		},
		memoryGauge,
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

func ParseType(input string, memoryGauge common.MemoryGauge) (ty ast.Type, errs []error) {
	var res interface{}
	res, errs = Parse(
		input,
		func(p *parser) interface{} {
			return parseType(p, lowestBindingPower)
		},
		memoryGauge,
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

func ParseDeclarations(input string, memoryGauge common.MemoryGauge) (declarations []ast.Declaration, errs []error) {
	var res interface{}
	res, errs = Parse(
		input,
		func(p *parser) interface{} {
			return parseDeclarations(p, lexer.TokenEOF)
		},
		memoryGauge,
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

func ParseArgumentList(input string, memoryGauge common.MemoryGauge) (arguments ast.Arguments, errs []error) {
	var res interface{}
	res, errs = Parse(
		input,
		func(p *parser) interface{} {
			p.skipSpaceAndComments(true)
			p.mustOne(lexer.TokenParenOpen)
			arguments, _ := parseArgumentListRemainder(p)
			return arguments
		},
		memoryGauge,
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

func ParseProgram(code string, memoryGauge common.MemoryGauge) (program *ast.Program, err error) {
	tokens := lexer.Lex(code, memoryGauge)
	defer tokens.Reclaim()
	return ParseProgramFromTokenStream(tokens, memoryGauge)
}

func ParseProgramFromTokenStream(
	input lexer.TokenStream,
	memoryGauge common.MemoryGauge,
) (
	program *ast.Program,
	err error,
) {
	var res interface{}
	var errs []error
	res, errs = ParseTokenStream(
		memoryGauge,
		input,
		func(p *parser) interface{} {
			return parseDeclarations(p, lexer.TokenEOF)
		},
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
	filename string,
	memoryGauge common.MemoryGauge,
) (
	program *ast.Program,
	code string,
	err error,
) {
	var data []byte
	data, err = ioutil.ReadFile(filename)
	if err != nil {
		return nil, "", err
	}

	code = string(data)

	program, err = ParseProgram(code, memoryGauge)
	if err != nil {
		return nil, code, err
	}
	return program, code, nil
}
