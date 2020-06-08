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
	"context"
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

// lowestBindingPower is the lowest binding power.
// binding power decides the order of which expression to be parsed first.
// the lower binding power is, the latter the expression will be parsed.
const lowestBindingPower = 0

type parser struct {
	tokens         chan lexer.Token
	current        lexer.Token
	errors         []error
	buffering      bool          // a flag indicating whether the next token will be read from buffered tokens or lexer
	bufferedTokens []lexer.Token // buffered tokens read from the lexer
	bufferPos      int           // the index of the next buffered token to read from
	bufferedErrors []error
}

// Parse creates a lexer to scan the given input string, and uses the parse function parse function to parse tokens
// into a result.
// It can be composed with different parse functions to parse the input string into different results.
// See "ParseExpression", "ParseStatements" as examples.
func Parse(input string, parse func(*parser) interface{}) (result interface{}, errors []error) {
	ctx, cancelLexer := context.WithCancel(context.Background())

	defer cancelLexer()

	// turn input string into tokens
	tokens := lexer.Lex(ctx, input)
	p := &parser{tokens: tokens}

	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("lexer: %v", r)
			}

			p.report(err)

			result = nil
			errors = p.errors
		}

		if p.buffering {
			errors = append(errors, p.bufferedErrors...)
		}
	}()

	p.next()

	result = parse(p)

	if !p.current.Is(lexer.TokenEOF) {
		p.report(fmt.Errorf("unexpected token: %v", p.current))
	}

	return result, p.errors
}

func (p *parser) report(err ...error) {
	if p.buffering {
		p.bufferedErrors = append(p.bufferedErrors, err...)
	} else {
		p.errors = append(p.errors, err...)
	}
}

const bufferPosTrimThreshold = 128

// maybeTrimBuffer checks whether the index of token we've read from buffered tokens
// has readed a threshold, in which case buffered tokens will be trimed and bufferPos
// will be reset.
func (p *parser) maybeTrimBuffer() {
	if p.bufferPos < bufferPosTrimThreshold {
		return
	}
	p.bufferedTokens = p.bufferedTokens[p.bufferPos:]
	p.bufferPos = 0
}

// next reads the next token and marks it as the "current" token.
// The next token could either be read from the lexer or from
// the buffer.
// Tokens are buffered when syntax ambiguity is involved.
func (p *parser) next() {
	for {
		var token lexer.Token

		// When the syntax has ambiguity, we need to process a series of tokens
		// multiple times. However, a token can only be consumed once from the lexer's
		// tokens channel. Therefore, in some circumstances, we need to buffer the tokens from the
		// lexer.
		//
		// buffering tokens allows us to "replay" the buffered tokens to deal with syntax ambiguity.
		if p.buffering {
			// if we need to buffer the next token
			// then read the token from from the lexer and buffer it.
			token = p.nextFromLexer()
			p.bufferedTokens = append(p.bufferedTokens, token)
		} else if p.bufferPos < len(p.bufferedTokens) {
			// if we don't need to buffer the next token and there are tokens buffered before,
			// then read the token from the buffer.
			token = p.nextFromBuffer()
		} else {
			// else no need to buffer, and there is no buffered token,
			// then read the next token from the lexer.
			token = p.nextFromLexer()
		}

		if token.Is(lexer.TokenError) {
			// Report error token as error, skip.
			p.report(token.Value.(error))
			continue
		}

		p.current = token
		return
	}
}

// nextFromLexer reads the next token from the lexer.
// should only be called by the "next" function
func (p *parser) nextFromLexer() lexer.Token {
	var ok bool
	token, ok := <-p.tokens
	if !ok {
		// Channel closed, return EOF token.
		token = lexer.Token{Type: lexer.TokenEOF}
	}
	return token
}

// nextFromLexer reads the next token from the buffer tokens, assuming there are buffered tokens.
// should only be called by the "next" function
func (p *parser) nextFromBuffer() lexer.Token {
	token := p.bufferedTokens[p.bufferPos]
	p.bufferPos++
	p.maybeTrimBuffer()
	return token
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

func (p *parser) acceptBuffered() {
	p.buffering = false
	p.bufferPos = len(p.bufferedTokens)
	p.report(p.bufferedErrors...)
	p.maybeTrimBuffer()
}

func (p *parser) replayBuffered() {
	p.buffering = false
	p.bufferedErrors = nil
	p.next()
}

func (p *parser) skipSpaceAndComments(skipNewlines bool) (containsNewline bool) {
	atEnd := false
	for !atEnd {
		switch p.current.Type {
		case lexer.TokenSpace:
			space := p.current.Value.(lexer.Space)

			if space.ContainsNewline {
				containsNewline = true
			}

			if containsNewline && !skipNewlines {
				return
			}

			p.next()

		case lexer.TokenBlockCommentStart:
			// TODO: use comment?
			p.parseCommentContent()

		case lexer.TokenLineComment:
			// TODO: use comment?
			p.next()

		default:
			atEnd = true
		}
	}
	return
}

func (p *parser) startBuffering() {
	p.buffering = true
	p.bufferedTokens = append(p.bufferedTokens, p.current)
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
	expression = res.(ast.Expression)
	return
}

func ParseStatements(input string) (statements []ast.Statement, errors []error) {
	var res interface{}
	res, errors = Parse(input, func(p *parser) interface{} {
		return parseStatements(p, lexer.TokenEOF)
	})
	if res == nil {
		statements = nil
		return
	}
	statements = res.([]ast.Statement)
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
	ty = res.(ast.Type)
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
	declarations = res.([]ast.Declaration)
	return
}

func ParseProgram(input string) (program *ast.Program, err error) {
	var res interface{}
	var errs []error
	res, errs = Parse(input, func(p *parser) interface{} {
		return parseDeclarations(p, lexer.TokenEOF)
	})
	if len(errs) > 0 {
		err = Error{
			Errors: errs,
		}
	}
	if res == nil {
		program = nil
		return
	}
	program = &ast.Program{
		Declarations: res.([]ast.Declaration),
	}
	return
}
