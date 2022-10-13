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
	"strings"

	"github.com/onflow/cadence/runtime/parser/lexer"
)

const blockCommentStart = "/*"
const blockCommentEnd = "*/"

func (p *parser) parseCommentContent() (comment string) {
	// TODO: improve: only build string if needed
	var builder strings.Builder
	defer func() {
		comment = builder.String()
	}()

	builder.WriteString(blockCommentStart)

	depth := 1

	for depth > 0 {
		p.next()

		switch p.current.Type {
		case lexer.TokenEOF:
			p.reportSyntaxError(
				"missing comment end %s",
				lexer.TokenBlockCommentEnd,
			)
			depth = 0

		case lexer.TokenBlockCommentContent:
			builder.Write(p.currentTokenSource())

		case lexer.TokenBlockCommentEnd:
			builder.WriteString(blockCommentEnd)
			// Skip the comment end (`*/`)
			p.next()
			depth--

		case lexer.TokenBlockCommentStart:
			builder.WriteString(blockCommentStart)
			depth++

		default:
			p.reportSyntaxError(
				"unexpected token in comment: %q",
				p.current.Type,
			)
			depth = 0
		}
	}

	return
}
