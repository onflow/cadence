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
	"fmt"
)

type stateFn func(*lexer) stateFn

func rootState(l *lexer) stateFn {
	r := l.next()
	switch r {
	case EOF:
		l.emitType(TokenEOF)
		return nil
	case ' ', '\t', '\n':
		return spaceState
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return numberState
	case '+':
		l.emitType(TokenPlus)
	case '-':
		l.emitType(TokenMinus)
	case '*':
		l.emitType(TokenStar)
	case '/':
		l.emitType(TokenSlash)
	case '?':
		if l.acceptOne('?') {
			l.emitType(TokenNilCoalesce)
		} else {
			l.emitType(TokenQuestionMark)
		}
	case '(':
		l.emitType(TokenParenOpen)
	case ')':
		l.emitType(TokenParenClose)
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
	case ':':
		l.emitType(TokenColon)
	case '_':
		return identifierState

	default:
		switch {
		case r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z':

			return identifierState

		default:
			return l.error(fmt.Errorf("unrecognized character: %#U", r))
		}
	}
	return rootState
}

func (l *lexer) error(err error) stateFn {
	l.emitError(err)
	return nil
}

func numberState(l *lexer) stateFn {
	l.scanNumber()
	l.emitValue(TokenNumber)
	return rootState
}

func spaceState(l *lexer) stateFn {
	l.scanSpace()
	l.emitValue(TokenSpace)
	return rootState
}

func identifierState(l *lexer) stateFn {
	l.scanIdentifier()
	l.emitValue(TokenIdentifier)
	return rootState
}
