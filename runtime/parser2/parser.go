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

const lowestBindingPower = 0

type parser struct {
	tokens  chan lexer.Token
	current lexer.Token
	pos     int
	errors  []error
}

func Parse(input string, parse func(*parser) interface{}) (result interface{}, errors []error) {
	ctx, cancelLexer := context.WithCancel(context.Background())

	defer cancelLexer()

	tokens := lexer.Lex(ctx, input)
	p := &parser{
		tokens: tokens,
		pos:    -1,
	}

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
	}()

	p.next()

	result = parse(p)

	if !p.current.Is(lexer.TokenEOF) {
		p.report(fmt.Errorf("unexpected token: %v", p.current))
	}

	return result, p.errors
}

func (p *parser) report(err ...error) {
	p.errors = append(p.errors, err...)
}

func (p *parser) next() {
	for {
		token, ok := <-p.tokens
		if !ok {
			// Channel closed, return EOF token.
			token = lexer.Token{Type: lexer.TokenEOF}
		} else if token.Is(lexer.TokenError) {
			// Report error token as error, skip.
			p.report(token.Value.(error))
			continue
		}

		p.pos++
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
