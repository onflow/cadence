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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2/lexer"
)

type parser struct {
	tokens  chan lexer.Token
	current lexer.Token
	pos     int
	errors  []error
}

func Parse(input string) (result ast.Expression, errors []error) {
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

	expr := parseExpression(p, 0)

	if !p.current.Is(lexer.TokenEOF) {
		p.report(fmt.Errorf("unexpected token: %v", p.current))
	}

	return expr, p.errors
}

func (p *parser) report(err error) {
	p.errors = append(p.errors, err)
}

func (p *parser) next() {
	p.pos++
	token, ok := <-p.tokens
	if !ok {
		// Channel closed, return EOF token.
		p.current = lexer.Token{Type: lexer.TokenEOF}
	} else {
		p.current = token
	}
}

func (p *parser) skipZeroOrOne(tokenType lexer.TokenType) {
	for p.current.Type == tokenType {
		p.next()
	}
}
