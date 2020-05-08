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
	"strings"
	"unicode/utf8"

	"github.com/onflow/cadence/runtime/ast"
)

type Token struct {
	Type  TokenType
	Value interface{}
	Range ast.Range
}

func (t Token) Is(ty TokenType) bool {
	return t.Type == ty
}

type position struct {
	line   int
	column int
}

type lexer struct {
	input         string
	startOffset   int
	endOffset     int
	prevEndOffset int
	tokens        chan Token
	canBackup     bool
	startPos      position
}

func Lex(input string) chan Token {
	l := &lexer{
		input:         input,
		startPos:      position{line: 1},
		endOffset:     0,
		prevEndOffset: 0,
		tokens:        make(chan Token),
	}
	go l.run(rootState)
	return l.tokens
}

func (l *lexer) run(state stateFn) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("lexer: %v", r)
			}
			l.emitError(err)
		}

		// Close token channel, no token remaining
		close(l.tokens)
	}()

	for state != nil {
		state = state(l)
	}
}

func (l *lexer) next() rune {
	l.canBackup = true

	endOffset := l.endOffset

	l.prevEndOffset = endOffset

	r := EOF
	w := 1
	if endOffset < len(l.input) {
		r, w = utf8.DecodeRuneInString(l.input[endOffset:])
	}

	l.endOffset += w

	return r
}

// peek returns but does not consume
// the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backupOne()
	return r
}

// backupOne steps back one rune.
// Can be called only once per call of next.
func (l *lexer) backupOne() {
	if !l.canBackup {
		panic("second backup")
	}
	l.canBackup = false

	l.endOffset = l.prevEndOffset
}

func (l *lexer) word() string {
	start := l.startOffset
	end := l.endOffset
	return l.input[start:end]
}

func (l *lexer) acceptOne(r rune) bool {
	if l.next() == r {
		return true
	}
	l.backupOne()
	return false
}

func (l *lexer) acceptAny(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backupOne()
	return false
}

func (l *lexer) acceptZeroOrMore(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backupOne()
}

func (l *lexer) acceptOneOrMore(valid string) bool {
	if !l.acceptAny(valid) {
		return false
	}
	l.acceptZeroOrMore(valid)
	return true
}

// emitValue passes an item back to the client.
func (l *lexer) emit(ty TokenType, val interface{}) {
	endPos := l.endPos()

	token := Token{
		Type:  ty,
		Value: val,
		Range: ast.Range{
			StartPos: ast.Position{
				Line:   l.startPos.line,
				Column: l.startPos.column,
				Offset: l.startOffset,
			},
			EndPos: ast.Position{
				Line:   endPos.line,
				Column: endPos.column,
				Offset: l.endOffset - 1,
			},
		},
	}
	l.tokens <- token
	l.startOffset = l.endOffset

	l.startPos = endPos
	r, _ := utf8.DecodeRuneInString(l.input[l.endOffset-1:])

	if r == '\n' {
		l.startPos.line++
		l.startPos.column = 0
	} else {
		l.startPos.column++
	}
}

func (l *lexer) endPos() position {
	startOffset := l.startOffset
	endOffset := l.endOffset

	endPos := l.startPos

	var w int
	for offset := startOffset; offset < endOffset-1; offset += w {
		var r rune
		r, w = utf8.DecodeRuneInString(l.input[offset:])

		if r == '\n' {
			endPos.line++
			endPos.column = 0
		} else {
			endPos.column++
		}
	}

	return endPos
}

func (l *lexer) emitType(ty TokenType) {
	l.emit(ty, nil)
}

func (l *lexer) emitValue(ty TokenType) {
	l.emit(ty, l.word())
}

func (l *lexer) emitError(err error) {
	l.emit(TokenError, err)
}

func (l *lexer) scanNumber() {
	// lookahead is already lexed.
	// parse more, if any
	l.acceptWhile(func(r rune) bool {
		return r >= '0' && r <= '9'
	})
}

func (l *lexer) scanSpace() {
	// lookahead is already lexed.
	// parse more, if any
	l.acceptZeroOrMore(" \t\n")
}

func (l *lexer) mustOne(r rune) {
	if !l.acceptOne(r) {
		panic(fmt.Errorf("expected character: %#U", r))
	}
}

func (l *lexer) acceptAll(string string) bool {
	endOffset := l.endOffset

	for _, r := range string {
		if l.next() != r {
			l.endOffset = endOffset

			return false
		}
	}

	return true
}

func (l *lexer) scanIdentifier() {
	// lookahead is already lexed.
	// parse more, if any
	l.acceptWhile(func(r rune) bool {
		return r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' ||
			r == '_'
	})
}

func (l *lexer) acceptWhile(f func(rune) bool) {

	for {
		r := l.next()

		if f(r) {
			continue
		}

		l.backupOne()
		return
	}
}
