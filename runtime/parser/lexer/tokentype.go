/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	TokenBinaryIntegerLiteral
	TokenOctalIntegerLiteral
	TokenDecimalIntegerLiteral
	TokenHexadecimalIntegerLiteral
	TokenUnknownBaseIntegerLiteral
	TokenFixedPointNumberLiteral
	TokenIdentifier
	TokenString
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenPercent
	TokenDoubleQuestionMark
	TokenParenOpen
	TokenParenClose
	TokenBraceOpen
	TokenBraceClose
	TokenBracketOpen
	TokenBracketClose
	TokenQuestionMark
	TokenQuestionMarkDot
	TokenComma
	TokenColon
	TokenDot
	TokenSemicolon
	TokenLeftArrow
	TokenLeftArrowExclamation
	TokenRightArrow
	TokenSwap
	TokenLess
	TokenLessEqual
	TokenLessLess
	TokenGreater
	TokenGreaterEqual
	TokenEqual
	TokenEqualEqual
	TokenExclamationMark
	TokenNotEqual
	TokenBlockCommentStart
	TokenBlockCommentEnd
	TokenBlockCommentContent
	TokenLineComment
	TokenAmpersand
	TokenAmpersandAmpersand
	TokenCaret
	TokenVerticalBar
	TokenVerticalBarVerticalBar
	TokenAt
	TokenAsExclamationMark
	TokenAsQuestionMark
	TokenPragma
	// NOTE: not an actual token, must be last item
	TokenMax
)

func init() {
	// ensure all tokens have its string format
	for t := TokenType(0); t < TokenMax; t++ {
		_ = t.String()
	}
}

func (t TokenType) String() string {
	switch t {
	case TokenError:
		return "error"
	case TokenEOF:
		return "EOF"
	case TokenSpace:
		return "space"
	case TokenBinaryIntegerLiteral:
		return "binary integer"
	case TokenOctalIntegerLiteral:
		return "octal integer"
	case TokenDecimalIntegerLiteral:
		return "decimal integer"
	case TokenHexadecimalIntegerLiteral:
		return "hexadecimal integer"
	case TokenFixedPointNumberLiteral:
		return "fixed-point number"
	case TokenUnknownBaseIntegerLiteral:
		return "integer with unknown base"
	case TokenIdentifier:
		return "identifier"
	case TokenString:
		return "string"
	case TokenPlus:
		return `'+'`
	case TokenMinus:
		return `'-'`
	case TokenStar:
		return `'*'`
	case TokenSlash:
		return `'/'`
	case TokenPercent:
		return `'%'`
	case TokenDoubleQuestionMark:
		return `'??'`
	case TokenParenOpen:
		return `'('`
	case TokenParenClose:
		return `')'`
	case TokenBraceOpen:
		return `'{'`
	case TokenBraceClose:
		return `'}'`
	case TokenBracketOpen:
		return `'['`
	case TokenBracketClose:
		return `']'`
	case TokenQuestionMark:
		return `'?'`
	case TokenQuestionMarkDot:
		return `'?.'`
	case TokenComma:
		return `','`
	case TokenColon:
		return `':'`
	case TokenDot:
		return `'.'`
	case TokenSemicolon:
		return `';'`
	case TokenLeftArrow:
		return `'<-'`
	case TokenLeftArrowExclamation:
		return `'<-!'`
	case TokenRightArrow:
		return `'->'`
	case TokenSwap:
		return `'<->'`
	case TokenLess:
		return `'<'`
	case TokenLessEqual:
		return `'<='`
	case TokenLessLess:
		return `'<<'`
	case TokenGreater:
		return `'>'`
	case TokenGreaterEqual:
		return `'>='`
	case TokenEqual:
		return `'='`
	case TokenEqualEqual:
		return `'=='`
	case TokenExclamationMark:
		return `'!'`
	case TokenNotEqual:
		return `'!='`
	case TokenBlockCommentStart:
		return `'/*'`
	case TokenBlockCommentContent:
		return "block comment"
	case TokenLineComment:
		return "line comment"
	case TokenBlockCommentEnd:
		return `'*/'`
	case TokenAmpersand:
		return `'&'`
	case TokenAmpersandAmpersand:
		return `'&&'`
	case TokenCaret:
		return `'^'`
	case TokenVerticalBar:
		return `'|'`
	case TokenVerticalBarVerticalBar:
		return `'||'`
	case TokenAt:
		return `'@'`
	case TokenAsExclamationMark:
		return `'as!'`
	case TokenAsQuestionMark:
		return `'as?'`
	case TokenPragma:
		return `'#'`
	default:
		panic(errors.NewUnreachableError())
	}
}

func (t TokenType) IsIntegerLiteral() bool {
	switch t {
	case TokenBinaryIntegerLiteral,
		TokenOctalIntegerLiteral,
		TokenDecimalIntegerLiteral,
		TokenHexadecimalIntegerLiteral,
		TokenUnknownBaseIntegerLiteral:
		return true

	default:
		return false
	}
}
