/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"fmt"

	"github.com/onflow/cadence/common"
)

const keywordAs = "as"

// stateFn uses the input lexer to read runes and emit tokens.
//
// It either returns nil when reaching end of file,
// or returns another stateFn for more scanning work.
type stateFn func(*lexer) stateFn

// rootState returns a stateFn that scans the file and emits tokens until
// reaching the end of the file.
func rootState(l *lexer) stateFn {
	for {
		r := l.next()
		switch r {
		case EOF:
			return nil
		case '+':
			l.emitType(TokenPlus)
		case '-':
			r = l.next()
			switch r {
			case '>':
				l.emitType(TokenRightArrow)
			default:
				l.backupOne()
				l.emitType(TokenMinus)
			}
		case '*':
			l.emitType(TokenStar)
		case '%':
			l.emitType(TokenPercent)
		case '(':
			if l.mode == lexerModeStringInterpolation {
				// it is necessary to balance brackets when generating tokens for string templates to know when to change modes
				l.openBrackets++
			}
			l.emitType(TokenParenOpen)
		case ')':
			l.emitType(TokenParenClose)
			if l.mode == lexerModeStringInterpolation {
				l.openBrackets--
				if l.openBrackets == 0 {
					l.mode = lexerModeNormal
					return stringState
				}
			}
		case '{':
			l.emitType(TokenBraceOpen)
		case '}':
			l.emitType(TokenBraceClose)
		case '[':
			l.emitType(TokenBracketOpen)
		case ']':
			l.emitType(TokenBracketClose)
		case ',':
			l.emitType(TokenComma)
		case ';':
			l.emitType(TokenSemicolon)
		case ':':
			l.emitType(TokenColon)
		case '.':
			l.emitType(TokenDot)
		case '=':
			if l.acceptOne('=') {
				l.emitType(TokenEqualEqual)
			} else {
				l.emitType(TokenEqual)
			}
		case '@':
			l.emitType(TokenAt)
		case '#':
			l.emitType(TokenPragma)
		case '&':
			if l.acceptOne('&') {
				l.emitType(TokenAmpersandAmpersand)
			} else {
				l.emitType(TokenAmpersand)
			}
		case '^':
			l.emitType(TokenCaret)
		case '|':
			if l.acceptOne('|') {
				l.emitType(TokenVerticalBarVerticalBar)
			} else {
				l.emitType(TokenVerticalBar)
			}
		case '>':
			r = l.next()
			switch r {
			case '=':
				l.emitType(TokenGreaterEqual)
			default:
				l.backupOne()
				l.emitType(TokenGreater)
			}
		case '_':
			return identifierState
		case ' ', '\t', '\r':
			return spaceState(false)
		case '\n':
			return spaceState(true)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return numberState
		case '"':
			return stringState
		case '\\':
			if l.mode == lexerModeStringInterpolation {
				r = l.next()
				switch r {
				case '(':
					l.emitType(TokenStringTemplate)
					l.openBrackets++
				}
			} else {
				return l.error(fmt.Errorf("unrecognized character: %#U", r))
			}
		case '/':
			r = l.next()
			switch r {
			case '/':
				return lineCommentState
			case '*':
				l.emitType(TokenBlockCommentStart)
				return blockCommentState(0)
			default:
				l.backupOne()
				l.emitType(TokenSlash)
			}
		case '?':
			r = l.next()
			switch r {
			case '?':
				l.emitType(TokenDoubleQuestionMark)
			case '.':
				l.emitType(TokenQuestionMarkDot)
			default:
				l.backupOne()
				l.emitType(TokenQuestionMark)
			}
		case '!':
			if l.acceptOne('=') {
				l.emitType(TokenNotEqual)
			} else {
				l.emitType(TokenExclamationMark)
			}
		case '<':
			r = l.next()
			switch r {
			case '-':
				r = l.next()
				switch r {
				case '!':
					l.emitType(TokenLeftArrowExclamation)
				case '>':
					l.emitType(TokenSwap)
				default:
					l.backupOne()
					l.emitType(TokenLeftArrow)
				}
			case '<':
				l.emitType(TokenLessLess)
			case '=':
				l.emitType(TokenLessEqual)
			default:
				l.backupOne()
				l.emitType(TokenLess)
			}
		default:
			switch {
			case r >= 'a' && r <= 'z' ||
				r >= 'A' && r <= 'Z':

				return identifierState

			default:
				return l.error(fmt.Errorf("unrecognized character: %#U", r))
			}
		}
	}
}

func (l *lexer) error(err error) stateFn {
	l.emitError(err)
	return nil
}

// numberState returns a stateFn that scans the following runes as a number
// and emits a corresponding token
func numberState(l *lexer) stateFn {
	// lookahead is already lexed.
	// parse more, if any
	r := l.current
	if r == '0' {
		r = l.next()
		switch r {
		case 'b':
			l.scanBinaryRemainder()
			if l.endOffset-l.startOffset <= 2 {
				l.emitError(fmt.Errorf("missing digits"))
			}
			l.emitType(TokenBinaryIntegerLiteral)

		case 'o':
			l.scanOctalRemainder()
			if l.endOffset-l.startOffset <= 2 {
				l.emitError(fmt.Errorf("missing digits"))
			}
			l.emitType(TokenOctalIntegerLiteral)

		case 'x':
			l.scanHexadecimalRemainder()
			if l.endOffset-l.startOffset <= 2 {
				l.emitError(fmt.Errorf("missing digits"))
			}
			l.emitType(TokenHexadecimalIntegerLiteral)

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '_':
			tokenType := l.scanDecimalOrFixedPointRemainder()
			l.emitType(tokenType)

		case '.':
			l.scanFixedPointRemainder()
			l.emitType(TokenFixedPointNumberLiteral)

		case EOF:
			l.backupOne()
			l.emitType(TokenDecimalIntegerLiteral)

		default:
			prefixChar := r

			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				l.emitError(fmt.Errorf("invalid number literal prefix: %q", prefixChar))
				l.next()

				tokenType := l.scanDecimalOrFixedPointRemainder()
				if tokenType == TokenDecimalIntegerLiteral {
					tokenType = TokenUnknownBaseIntegerLiteral
				}
				l.emitType(tokenType)
			} else {
				l.backupOne()
				l.emitType(TokenDecimalIntegerLiteral)
			}
		}

	} else {
		tokenType := l.scanDecimalOrFixedPointRemainder()
		l.emitType(tokenType)
	}

	return rootState
}

type Space struct {
	ContainsNewline bool
}

func spaceState(startIsNewline bool) stateFn {
	return func(l *lexer) stateFn {
		containsNewline := l.scanSpace()
		containsNewline = containsNewline || startIsNewline

		common.UseMemory(l.memoryGauge, common.SpaceTokenMemoryUsage)

		l.emit(
			TokenSpace,
			Space{
				ContainsNewline: containsNewline,
			},
			l.startPosition(),
			true,
		)
		return rootState
	}
}

func identifierState(l *lexer) stateFn {
	l.scanIdentifier()
	// https://github.com/golang/go/commit/69cd91a5981c49eaaa59b33196bdb5586c18d289
	if string(l.word()) == keywordAs {
		r := l.next()
		switch r {
		case '?':
			l.emitType(TokenAsQuestionMark)
			return rootState
		case '!':
			l.emitType(TokenAsExclamationMark)
			return rootState
		default:
			l.backupOne()
		}
	}
	l.emitType(TokenIdentifier)
	return rootState
}

func stringState(l *lexer) stateFn {
	l.scanString('"')
	l.emitType(TokenString)
	return rootState
}

func lineCommentState(l *lexer) stateFn {
	l.scanLineComment()
	l.emitType(TokenLineComment)
	return rootState
}

func blockCommentState(nesting int) stateFn {
	if nesting < 0 {
		return rootState
	}

	return func(l *lexer) stateFn {
		r := l.next()
		switch r {
		case EOF:
			return nil
		case '/':
			beforeSlashOffset := l.prevEndOffset
			if l.acceptOne('*') {
				if beforeSlashOffset-l.startOffset > 0 {
					starOffset := l.endOffset
					l.endOffset = beforeSlashOffset
					l.emitType(TokenBlockCommentContent)
					l.endOffset = starOffset
				}
				l.emitType(TokenBlockCommentStart)
				return blockCommentState(nesting + 1)
			}

		case '*':
			beforeStarOffset := l.prevEndOffset
			if l.acceptOne('/') {
				if beforeStarOffset-l.startOffset > 0 {
					slashOffset := l.endOffset
					l.endOffset = beforeStarOffset
					l.emitType(TokenBlockCommentContent)
					l.endOffset = slashOffset
				}
				l.emitType(TokenBlockCommentEnd)
				return blockCommentState(nesting - 1)
			}
		}

		return blockCommentState(nesting)
	}
}
