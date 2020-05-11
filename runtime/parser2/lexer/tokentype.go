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

package lexer

import (
	"github.com/onflow/cadence/runtime/errors"
)

type TokenType uint8

const EOF rune = -1

const (
	TokenError TokenType = iota
	TokenEOF
	TokenSpace
	TokenNumber
	TokenIdentifier
	TokenString
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenPercent
	TokenNilCoalesce
	TokenParenOpen
	TokenParenClose
	TokenBraceOpen
	TokenBraceClose
	TokenBracketOpen
	TokenBracketClose
	TokenQuestionMark
	TokenComma
	TokenColon
	TokenDot
	TokenSemicolon
	TokenLeftArrow
	TokenLeftArrowExclamation
	TokenLess
	TokenLessEqual
	TokenLessLess
	TokenGreater
	TokenGreaterEqual
	TokenGreaterGreater
	TokenEqual
	TokenEqualEqual
	TokenNot
	TokenNotEqual
	TokenBlockCommentStart
	TokenBlockCommentContent
	TokenBlockCommentEnd
	TokenAmpersand
	TokenAmpersandAmpersand
	TokenCaret
	TokenVerticalBar
	TokenVerticalBarVerticalBar
	TokenAt
)

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenSpace:
		return "space"
	case TokenNumber:
		return "number"
	case TokenIdentifier:
		return "identifier"
	case TokenString:
		return "string"
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenStar:
		return "*"
	case TokenSlash:
		return "/"
	case TokenPercent:
		return "%"
	case TokenNilCoalesce:
		return "??"
	case TokenParenOpen:
		return "("
	case TokenParenClose:
		return ")"
	case TokenBraceOpen:
		return "{"
	case TokenBraceClose:
		return "}"
	case TokenBracketOpen:
		return "["
	case TokenBracketClose:
		return "]"
	case TokenQuestionMark:
		return "?"
	case TokenComma:
		return ","
	case TokenColon:
		return ":"
	case TokenDot:
		return "."
	case TokenSemicolon:
		return ";"
	case TokenLeftArrow:
		return "<-"
	case TokenLeftArrowExclamation:
		return "<-!"
	case TokenLess:
		return "<"
	case TokenLessEqual:
		return "<="
	case TokenLessLess:
		return "<<"
	case TokenGreater:
		return ">"
	case TokenGreaterEqual:
		return ">="
	case TokenGreaterGreater:
		return ">>"
	case TokenEqual:
		return "="
	case TokenEqualEqual:
		return "=="
	case TokenNot:
		return "!"
	case TokenNotEqual:
		return "!="
	case TokenBlockCommentStart:
		return "/*"
	case TokenBlockCommentContent:
		return "comment"
	case TokenBlockCommentEnd:
		return "*/"
	case TokenAmpersand:
		return "&"
	case TokenAmpersandAmpersand:
		return "&&"
	case TokenCaret:
		return "^"
	case TokenVerticalBar:
		return "|"
	case TokenVerticalBarVerticalBar:
		return "||"
	case TokenAt:
		return "@"
	default:
		panic(errors.NewUnreachableError())
	}
}
