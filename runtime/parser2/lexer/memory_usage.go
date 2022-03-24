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

package lexer

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

// valueMemoryUsage returns the memory usage, given the token type of the value.
//
// NOTE: This assumes the token-type can only be one of:
//     - TokenString
//     - TokenIdentifier
//     - TokenBlockCommentContent
//     - TokenLineComment
//     - Any numeric literal (e.g: integer, fixed-point, etc.)
//
func (l *lexer) valueMemoryUsage(tokenType TokenType) common.MemoryUsage {
	switch tokenType {
	case TokenString:
		return common.NewStringMemoryUsage(l.wordLength())
	case TokenIdentifier:
		return common.NewIdentifierTokenMemoryUsage(l.wordLength())
	case TokenBlockCommentContent, TokenLineComment:
		return common.NewCommentTokenMemoryUsage(l.wordLength())
	default:
		return common.NewNumericLiteralTokenMemoryUsage(l.wordLength())
	}
}

// singleWidthTokenMemoryUsage is the memory consumed by a token consist of one codepoint.
var singleWidthTokenMemoryUsage = common.NewSyntaxTokenMemoryUsage(1)

// doubleWidthTokenMemoryUsage is the memory consumed by a token consist of two codepoints.
var doubleWidthTokenMemoryUsage = common.NewSyntaxTokenMemoryUsage(2)

// tripleWidthTokenMemoryUsage is the memory consumed by a token consist of three codepoints.
var tripleWidthTokenMemoryUsage = common.NewSyntaxTokenMemoryUsage(3)

// typeMemoryUsage returns the memory usage, given the token type of the type.
//
// NOTE: This assumes the token-type is always a syntax token.
//  e.g:
//     - logical and mathematical operators such as +, - /, *, etc.
//     - Separators such as {, }, (, ), dot, coma, colons, etc.
//     - And other similar tokens, such as ?, //, /*
//
func (l *lexer) typeMemoryUsage(tokenType TokenType) common.MemoryUsage {
	switch tokenType {

	// 1-length tokens
	case TokenPlus,
		TokenMinus,
		TokenStar,
		TokenSlash,
		TokenPercent,
		TokenParenOpen,
		TokenParenClose,
		TokenBraceOpen,
		TokenBraceClose,
		TokenBracketOpen,
		TokenBracketClose,
		TokenQuestionMark,
		TokenComma,
		TokenColon,
		TokenDot,
		TokenSemicolon,
		TokenLess,
		TokenGreater,
		TokenEqual,
		TokenExclamationMark,
		TokenVerticalBar,
		TokenAmpersand,
		TokenCaret,
		TokenAt,
		TokenPragma:
		return singleWidthTokenMemoryUsage

	// 2-length tokens
	case TokenDoubleQuestionMark,
		TokenQuestionMarkDot,
		TokenLeftArrow,
		TokenLessEqual,
		TokenLessLess,
		TokenGreaterEqual,
		TokenEqualEqual,
		TokenNotEqual,
		TokenBlockCommentStart,
		TokenBlockCommentEnd,
		TokenAmpersandAmpersand,
		TokenVerticalBarVerticalBar:
		return doubleWidthTokenMemoryUsage

	// 3-length tokens
	case TokenLeftArrowExclamation,
		TokenSwap,
		TokenAsExclamationMark,
		TokenAsQuestionMark:
		return tripleWidthTokenMemoryUsage

	default:
		panic(errors.NewUnreachableError())
	}
}
