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
	"github.com/onflow/cadence/runtime/parser/lexer"
)

func (p *parser) parseBlockComment() (endToken lexer.Token, ok bool) {
	var depth int

	for {
		switch p.current.Type {
		case lexer.TokenBlockCommentStart:
			p.next()
			ok = false
			depth++

		case lexer.TokenBlockCommentContent:
			p.next()
			ok = false

		case lexer.TokenBlockCommentEnd:
			endToken = p.current
			// Skip the comment end (`*/`)
			p.next()
			ok = true
			depth--
			if depth == 0 {
				return
			}

		case lexer.TokenEOF:
			p.reportSyntaxError(
				"missing comment end %s",
				lexer.TokenBlockCommentEnd,
			)
			ok = false
			return

		default:
			p.reportSyntaxError(
				"unexpected token %s in block comment",
				p.current.Type,
			)
			ok = false
			return
		}
	}
}
