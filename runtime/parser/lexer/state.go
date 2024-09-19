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

	"github.com/onflow/cadence/runtime/common"
)

const keywordAs = "as"

// stateFn uses the input lexer to read runes and emit tokens.
//
// It either returns nil when reaching end of file,
// or returns another stateFn for more scanning work.
type stateFn func(*lexer) (stateFn, error)

// rootState returns a stateFn that scans the file and emits tokens until
// reaching the end of the file.
func rootState(l *lexer) (stateFn, error) {

	for {
		var ty TokenType

		r := l.next()
		switch r {
		case EOF:
			return nil, nil
		case '+':
			ty = TokenPlus
		case '-':
			r = l.next()
			switch r {
			case '>':
				ty = TokenRightArrow
			default:
				l.backupOne()
				ty = TokenMinus
			}
		case '*':
			ty = TokenStar
		case '%':
			ty = TokenPercent
		case '(':
			ty = TokenParenOpen
		case ')':
			ty = TokenParenClose
		case '{':
			ty = TokenBraceOpen
		case '}':
			ty = TokenBraceClose
		case '[':
			ty = TokenBracketOpen
		case ']':
			ty = TokenBracketClose
		case ',':
			ty = TokenComma
		case ';':
			ty = TokenSemicolon
		case ':':
			ty = TokenColon
		case '.':
			ty = TokenDot
		case '=':
			if l.acceptOne('=') {
				ty = TokenEqualEqual
			} else {
				ty = TokenEqual
			}
		case '@':
			ty = TokenAt
		case '#':
			ty = TokenPragma
		case '&':
			if l.acceptOne('&') {
				ty = TokenAmpersandAmpersand
			} else {
				ty = TokenAmpersand
			}
		case '^':
			ty = TokenCaret
		case '|':
			if l.acceptOne('|') {
				ty = TokenVerticalBarVerticalBar
			} else {
				ty = TokenVerticalBar
			}
		case '>':
			r = l.next()
			switch r {
			case '=':
				ty = TokenGreaterEqual
			default:
				l.backupOne()
				ty = TokenGreater
			}
		case '_':
			return identifierState, nil
		case ' ', '\t', '\r':
			return spaceState(false), nil
		case '\n':
			return spaceState(true), nil
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return numberState, nil
		case '"':
			return stringState, nil
		case '/':
			r = l.next()
			switch r {
			case '/':
				return lineCommentState, nil
			case '*':
				err := l.emitType(TokenBlockCommentStart)
				if err != nil {
					return nil, err
				}
				return blockCommentState(0), nil
			default:
				l.backupOne()
				ty = TokenSlash
			}
		case '?':
			r = l.next()
			switch r {
			case '?':
				ty = TokenDoubleQuestionMark
			case '.':
				ty = TokenQuestionMarkDot
			default:
				l.backupOne()
				ty = TokenQuestionMark
			}
		case '!':
			if l.acceptOne('=') {
				ty = TokenNotEqual
			} else {
				ty = TokenExclamationMark
			}
		case '<':
			r = l.next()
			switch r {
			case '-':
				r = l.next()
				switch r {
				case '!':
					ty = TokenLeftArrowExclamation
				case '>':
					ty = TokenSwap
				default:
					l.backupOne()
					ty = TokenLeftArrow
				}
			case '<':
				ty = TokenLessLess
			case '=':
				ty = TokenLessEqual
			default:
				l.backupOne()
				ty = TokenLess
			}
		default:
			switch {
			case r >= 'a' && r <= 'z' ||
				r >= 'A' && r <= 'Z':

				return identifierState, nil

			default:
				return l.error(fmt.Errorf("unrecognized character: %#U", r))
			}
		}

		err := l.emitType(ty)
		if err != nil {
			return nil, err
		}
	}
}

func (l *lexer) error(err error) (stateFn, error) {
	return nil, l.emitError(err)
}

// numberState returns a stateFn that scans the following runes as a number
// and emits a corresponding token
func numberState(l *lexer) (stateFn, error) {
	// lookahead is already lexed.
	// parse more, if any
	r := l.current
	if r == '0' {
		r = l.next()
		switch r {
		case 'b':
			l.scanBinaryRemainder()
			if l.endOffset-l.startOffset <= 2 {
				err := l.emitError(fmt.Errorf("missing digits"))
				if err != nil {
					return nil, err
				}
			}
			return l.emitTokenAndReturnRootState(TokenBinaryIntegerLiteral)

		case 'o':
			l.scanOctalRemainder()
			if l.endOffset-l.startOffset <= 2 {
				err := l.emitError(fmt.Errorf("missing digits"))
				if err != nil {
					return nil, err
				}
			}
			return l.emitTokenAndReturnRootState(TokenOctalIntegerLiteral)

		case 'x':
			l.scanHexadecimalRemainder()
			if l.endOffset-l.startOffset <= 2 {
				err := l.emitError(fmt.Errorf("missing digits"))
				if err != nil {
					return nil, err
				}
			}
			return l.emitTokenAndReturnRootState(TokenHexadecimalIntegerLiteral)

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '_':
			tokenType, err := l.scanDecimalOrFixedPointRemainder()
			if err != nil {
				return nil, err
			}

			return l.emitTokenAndReturnRootState(tokenType)

		case '.':
			err := l.scanFixedPointRemainder()
			if err != nil {
				return nil, err
			}
			return l.emitTokenAndReturnRootState(TokenFixedPointNumberLiteral)

		case EOF:
			l.backupOne()
			return l.emitTokenAndReturnRootState(TokenDecimalIntegerLiteral)

		default:
			prefixChar := r

			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				err := l.emitError(fmt.Errorf("invalid number literal prefix: %q", prefixChar))
				if err != nil {
					return nil, err
				}
				l.next()

				tokenType, err := l.scanDecimalOrFixedPointRemainder()
				if err != nil {
					return nil, err
				}
				if tokenType == TokenDecimalIntegerLiteral {
					tokenType = TokenUnknownBaseIntegerLiteral
				}
				return l.emitTokenAndReturnRootState(tokenType)
			} else {
				l.backupOne()
				return l.emitTokenAndReturnRootState(TokenDecimalIntegerLiteral)
			}
		}

	} else {
		tokenType, err := l.scanDecimalOrFixedPointRemainder()
		if err != nil {
			return nil, err
		}
		return l.emitTokenAndReturnRootState(tokenType)
	}
}

type Space struct {
	ContainsNewline bool
}

func spaceState(startIsNewline bool) stateFn {
	return func(l *lexer) (stateFn, error) {
		containsNewline := l.scanSpace()
		containsNewline = containsNewline || startIsNewline

		common.UseMemory(l.memoryGauge, common.SpaceTokenMemoryUsage)

		err := l.emit(
			TokenSpace,
			Space{
				ContainsNewline: containsNewline,
			},
			l.startPosition(),
			true,
		)

		if err != nil {
			return nil, err
		}
		return rootState, nil
	}
}

func identifierState(l *lexer) (stateFn, error) {
	l.scanIdentifier()
	// https://github.com/golang/go/commit/69cd91a5981c49eaaa59b33196bdb5586c18d289
	if string(l.word()) == keywordAs {
		r := l.next()
		switch r {
		case '?':
			return l.emitTokenAndReturnRootState(TokenAsQuestionMark)
		case '!':
			return l.emitTokenAndReturnRootState(TokenAsExclamationMark)
		default:
			l.backupOne()
		}
	}
	return l.emitTokenAndReturnRootState(TokenIdentifier)
}

func stringState(l *lexer) (stateFn, error) {
	l.scanString('"')
	return l.emitTokenAndReturnRootState(TokenString)
}

func lineCommentState(l *lexer) (stateFn, error) {
	l.scanLineComment()
	return l.emitTokenAndReturnRootState(TokenLineComment)
}

func (l *lexer) emitTokenAndReturnRootState(token TokenType) (stateFn, error) {
	err := l.emitType(token)
	if err != nil {
		return nil, err
	}
	return rootState, nil
}

func blockCommentState(nesting int) stateFn {
	if nesting < 0 {
		return rootState
	}

	return func(l *lexer) (stateFn, error) {
		r := l.next()
		switch r {
		case EOF:
			return nil, nil
		case '/':
			beforeSlashOffset := l.prevEndOffset
			if l.acceptOne('*') {
				starOffset := l.endOffset
				l.endOffset = beforeSlashOffset
				err := l.emitType(TokenBlockCommentContent)
				if err != nil {
					return nil, err
				}

				l.endOffset = starOffset
				err = l.emitType(TokenBlockCommentStart)
				if err != nil {
					return nil, err
				}

				return blockCommentState(nesting + 1), nil
			}

		case '*':
			beforeStarOffset := l.prevEndOffset
			if l.acceptOne('/') {
				slashOffset := l.endOffset
				l.endOffset = beforeStarOffset
				err := l.emitType(TokenBlockCommentContent)
				if err != nil {
					return nil, err
				}
				l.endOffset = slashOffset
				err = l.emitType(TokenBlockCommentEnd)
				if err != nil {
					return nil, err
				}
				return blockCommentState(nesting - 1), nil
			}
		}

		return blockCommentState(nesting), nil
	}
}
