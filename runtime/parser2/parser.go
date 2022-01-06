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
	"io/ioutil"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
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
	// backtrackingCursors
	backtrackingCursors []int
	// bufferedErrors are the parsing errors encountered during buffering
	bufferedErrors [][]error
}

// Parse creates a lexer to scan the given input string,
// and uses the given `parse` function to parse tokens into a result.
//
// It can be composed with different parse functions to parse the input string into different results.
// See "ParseExpression", "ParseStatements" as examples.
//
func Parse(input string, parse func(*parser) interface{}) (result interface{}, errors []error) {
	// create a lexer, which turns the input string into tokens
	tokens := lexer.Lex(input)
	return ParseTokenStream(tokens, parse)
}

func ParseTokenStream(tokens lexer.TokenStream, parse func(*parser) interface{}) (result interface{}, errors []error) {
	p := &parser{tokens: tokens}

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

		for _, bufferedErrors := range p.bufferedErrors {
			errors = append(errors, bufferedErrors...)
		}
	}()

	p.current = lexer.Token{
		Type: lexer.TokenEOF,
		Range: ast.Range{
			StartPos: ast.Position{Offset: 0, Line: 1, Column: 0},
			EndPos:   ast.Position{Offset: 0, Line: 1, Column: 0},
		},
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

		var parseError ParseError
		var ok bool
		parseError, ok = err.(ParseError)
		if !ok {
			parseError = &SyntaxError{
				Pos:     p.current.StartPos,
				Message: err.Error(),
			}
		}

		// Add the errors to the buffered errors if buffering,
		// or the final errors if not

		bufferedErrorsDepth := len(p.bufferedErrors)
		if bufferedErrorsDepth > 0 {
			bufferedErrorsIndex := bufferedErrorsDepth - 1
			p.bufferedErrors[bufferedErrorsIndex] = append(
				p.bufferedErrors[bufferedErrorsIndex],
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
	p.backtrackingCursors = append(p.backtrackingCursors, p.tokens.Cursor()-1)

	// Push an empty slice of errors to the stack
	p.bufferedErrors = append(p.bufferedErrors, nil)
}

func (p *parser) acceptBuffered() {
	// Pop the last backtracking cursor from the stack
	// and ignore it

	lastIndex := len(p.backtrackingCursors) - 1
	p.backtrackingCursors = p.backtrackingCursors[:lastIndex]

	// Pop the last buffered errors from the stack
	// and apply them to the previous errors on the buffered errors stack,
	// or the final errors, if we reached the bottom of the stack
	// (i.e. this acceptance disables buffering)

	lastIndex = len(p.bufferedErrors) - 1
	bufferedErrors := p.bufferedErrors[lastIndex]
	p.bufferedErrors[lastIndex] = nil
	p.bufferedErrors = p.bufferedErrors[:lastIndex]
	if len(bufferedErrors) > 0 {
		p.bufferedErrors[lastIndex-1] = append(
			p.bufferedErrors[lastIndex-1],
			bufferedErrors...,
		)
	} else {
		p.errors = append(
			p.errors,
			bufferedErrors...,
		)
	}
}

func (p *parser) replayBuffered() {
	// Pop the last backtracking cursor from the stack
	// and revert the lexer back to it

	lastIndex := len(p.backtrackingCursors) - 1
	cursor := p.backtrackingCursors[lastIndex]
	p.tokens.Revert(cursor)
	p.next()
	p.backtrackingCursors = p.backtrackingCursors[:lastIndex]

	// Pop the last buffered errors from the stack
	// and ignore them

	lastIndex = len(p.bufferedErrors) - 1
	p.bufferedErrors[lastIndex] = nil
	p.bufferedErrors = p.bufferedErrors[:lastIndex]
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

func mustIdentifier(p *parser) ast.Identifier {
	identifier := p.mustOne(lexer.TokenIdentifier)
	return tokenToIdentifier(identifier)
}

func tokenToIdentifier(identifier lexer.Token) ast.Identifier {
	return ast.Identifier{
		Identifier: identifier.Value.(string),
		Pos:        identifier.StartPos,
	}
}

func ParseExpression(input string) (expression ast.Expression, errors []error) {
	var res interface{}
	res, errors = Parse(input, func(p *parser) interface{} {
		return parseExpression(p, lowestBindingPower)
	})
	if res == nil {
		expression = nil
		return
	}
	// it's ok for expression to be nil here
	expression, _ = res.(ast.Expression)
	return
}

func ParseStatements(input string) (statements []ast.Statement, errors []error) {
	var res interface{}
	res, errors = Parse(input, func(p *parser) interface{} {
		return parseStatements(p, nil)
	})
	if res == nil {
		statements = nil
		return
	}
	// it's ok for statement to be nil here
	statements, _ = res.([]ast.Statement)
	return
}

func ParseType(input string) (ty ast.Type, errors []error) {
	var res interface{}
	res, errors = Parse(input, func(p *parser) interface{} {
		return parseType(p, lowestBindingPower)
	})
	if res == nil {
		ty = nil
		return
	}
	// it's ok for ty to be nil here
	ty, _ = res.(ast.Type)
	return
}

func ParseDeclarations(input string) (declarations []ast.Declaration, errors []error) {
	var res interface{}
	res, errors = Parse(input, func(p *parser) interface{} {
		return parseDeclarations(p, lexer.TokenEOF)
	})
	if res == nil {
		declarations = nil
		return
	}
	// it's ok for declarations to be nil here
	declarations, _ = res.([]ast.Declaration)
	return
}

func ParseArgumentList(input string) (arguments ast.Arguments, errors []error) {
	var res interface{}
	res, errors = Parse(input, func(p *parser) interface{} {
		p.skipSpaceAndComments(true)
		p.mustOne(lexer.TokenParenOpen)
		arguments, _ := parseArgumentListRemainder(p)
		return arguments
	})
	if res == nil {
		arguments = nil
		return
	}
	// it's ok for arguments to be nil here
	arguments, _ = res.([]*ast.Argument)
	return
}

func ParseProgram(input string) (program *ast.Program, err error) {
	return ParseProgramFromTokenStream(lexer.Lex(input))
}

func ParseProgramFromTokenStream(input lexer.TokenStream) (program *ast.Program, err error) {
	var res interface{}
	var errs []error
	res, errs = ParseTokenStream(input, func(p *parser) interface{} {
		return parseDeclarations(p, lexer.TokenEOF)
	})
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
		program = nil
		return
	}

	program = ast.NewProgram(declarations)

	return
}

func ParseProgramFromFile(filename string) (program *ast.Program, code string, err error) {
	var data []byte
	data, err = ioutil.ReadFile(filename)
	if err != nil {
		return nil, "", err
	}

	code = string(data)

	program, err = ParseProgram(code)
	if err != nil {
		return nil, code, err
	}
	return program, code, nil
}
