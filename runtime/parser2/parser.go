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
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

type parser struct {
	tokens  chan lexer.Token
	current lexer.Token
	pos     int
	errors  []error
}

func Parse(input string, f func(*parser) interface{}) (result interface{}, errors []error) {
	tokens := lexer.Lex(input)
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

	expr := f(p)

	if !p.current.Is(lexer.TokenEOF) {
		p.report(fmt.Errorf("unexpected token: %v", p.current))
	}

	return expr, p.errors
}

func ParseExpression(input string) (expression ast.Expression, errors []error) {
	var res interface{}
	res, errors = Parse(input, func(p *parser) interface{} {
		return parseExpression(p, 0)
	})
	if res == nil {
		expression = nil
		return
	}
	expression = res.(ast.Expression)
	return
}

func (p *parser) report(err ...error) {
	p.errors = append(p.errors, err...)
}

func (p *parser) next() {
	p.pos++
	token, ok := <-p.tokens
	if !ok {
		// Channel closed, return EOF token.
		token = lexer.Token{Type: lexer.TokenEOF}
	} else if token.Type == lexer.TokenError {
		// Report error token as error, skip.
		p.report(token.Value.(error))
		p.next()
		return
	}

	p.current = token
}

func (p *parser) skipZeroOrOne(tokenType lexer.TokenType) {
	for p.current.Type == tokenType {
		p.next()
	}
}

func (p *parser) mustOne(tokenType lexer.TokenType) lexer.Token {
	t := p.current
	if t.Type != tokenType {
		panic(fmt.Errorf("expected token type: %s", tokenType))
	}
	p.next()
	return t
}

func (p *parser) skipSpaceAndComments() {
	for {
		p.skipZeroOrOne(lexer.TokenSpace)
		if p.current.Type != lexer.TokenBlockCommentStart {
			break
		}
		// TODO: use comment?
		p.parseCommentContent()
	}
}

func (p *parser) parseCommentContent() (comment string) {
	var builder strings.Builder
	defer func() {
		comment = builder.String()
	}()

	builder.WriteString("/*")

	var t trampoline
	t = func(builder *strings.Builder) trampoline {
		return func() []trampoline {

			for {
				p.next()

				switch p.current.Type {
				case lexer.TokenEOF:
					p.report(fmt.Errorf("missing comment end"))
					return nil
				case lexer.TokenBlockCommentContent:
					builder.WriteString(p.current.Value.(string))
				case lexer.TokenBlockCommentEnd:
					builder.WriteString("*/")
					p.next()
					return nil
				case lexer.TokenBlockCommentStart:
					builder.WriteString("/*")

					// parse inner content, then rest of this comment
					return []trampoline{t, t}
				default:
					p.report(fmt.Errorf("unexpected token in comment: %v", p.current))
					return nil
				}
			}
		}
	}(&builder)
	runTrampoline(t)
	return
}

type trampoline func() []trampoline

func runTrampoline(start trampoline) {
	ts := []trampoline{start}

	for len(ts) > 0 {
		var t trampoline
		t, ts = ts[0], ts[1:]
		more := t()
		ts = append(ts, more...)
	}
}
