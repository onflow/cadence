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
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/onflow/cadence/runtime/ast"
)

type Token struct {
	Type  TokenType
	Value interface{}
	ast.Range
}

func (t Token) Is(ty TokenType) bool {
	return t.Type == ty
}

func (t Token) IsString(ty TokenType, s string) bool {
	if !t.Is(ty) {
		return false
	}
	v, ok := t.Value.(string)
	if !ok {
		return false
	}
	return v == s
}

type position struct {
	line   int
	column int
}

type lexer struct {
	ctx context.Context
	// the entire input string
	input string
	// the start offset of the current word in the current line
	startOffset int
	// the end offset of the current word in the current line
	endOffset int
	// the previous end offset, used for stepping back
	prevEndOffset int
	// the current rune is scanned
	current rune
	// the previous rune was scanned, used for stepping back
	prev rune
	// the channel of tokens that has been scanned.
	tokens chan Token
	// signal whether stepping back is allowed
	canBackup bool
	// the start position of the current word
	startPos position
}

func Lex(ctx context.Context, input string) chan Token {
	l := &lexer{
		ctx:           ctx,
		input:         input,
		startPos:      position{line: 1},
		endOffset:     0,
		prevEndOffset: 0,
		current:       EOF,
		prev:          EOF,
		tokens:        make(chan Token),
	}
	go l.run(rootState)
	return l.tokens
}

type done struct{}

// run executes the stateFn, which will scan the runes in the input
// and emit tokens to the tokens channel.
//
// stateFn might return another stateFn to indicate further scanning work,
// or nil if there is no scanning work left to be done,
// i.e. run will keep running the returned stateFn until no more
// stateFn is returned, which for example happens when reaching the end of the file.
//
// When all stateFn have been executed, the tokens channel will be closed.
func (l *lexer) run(state stateFn) {
	// Close token channel, no token remaining
	defer close(l.tokens)

	// catch panic exceptions, emit it to the tokens channel before
	// closing it
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch r := r.(type) {
			case done:
				return
			case error:
				err = r
			default:
				err = fmt.Errorf("lexer: %v", r)
			}

			l.emitError(err)
		}
	}()

	for state != nil {
		state = state(l)
	}
}

// next decodes the next rune (UTF8 character) from the input string.
//
// It returns EOF if it reaches the end of the file,
// otherwise returns the scanned rune.
func (l *lexer) next() rune {
	l.canBackup = true

	endOffset := l.endOffset

	// update prevEndOffset and prev so that we can step back one rune.
	l.prevEndOffset = endOffset
	l.prev = l.current

	r := EOF
	w := 1
	if endOffset < len(l.input) {
		r, w = utf8.DecodeRuneInString(l.input[endOffset:])
	}

	l.endOffset += w
	l.current = r

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
	l.current = l.prev
}

func (l *lexer) word() string {
	start := l.startOffset
	end := l.endOffset
	return l.input[start:end]
}

// acceptOne reads one rune ahead.
// It returns true if the next rune matches with the input rune,
// otherwise it steps back one rune and returns false.
//
func (l *lexer) acceptOne(r rune) bool {
	if l.next() == r {
		return true
	}
	l.backupOne()
	return false
}

// emit writes a token to the channel.
func (l *lexer) emit(ty TokenType, val interface{}, rangeStart ast.Position, consume bool) {
	endPos := l.endPos()

	token := Token{
		Type:  ty,
		Value: val,
		Range: ast.Range{
			StartPos: rangeStart,
			EndPos: ast.Position{
				Line:   endPos.line,
				Column: endPos.column,
				Offset: l.endOffset - 1,
			},
		},
	}

	select {
	case <-l.ctx.Done():
		panic(done{})

	case l.tokens <- token:

	}

	if consume {
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
}

func (l *lexer) startPosition() ast.Position {
	return ast.Position{
		Line:   l.startPos.line,
		Column: l.startPos.column,
		Offset: l.startOffset,
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
	l.emit(ty, nil, l.startPosition(), true)
}

func (l *lexer) emitValue(ty TokenType) {
	l.emit(ty, l.word(), l.startPosition(), true)
}

func (l *lexer) emitError(err error) {
	endPos := l.endPos()
	rangeStart := ast.Position{
		Line:   endPos.line,
		Column: endPos.column,
		Offset: l.endOffset - 1,
	}
	l.emit(TokenError, err, rangeStart, false)
}

func (l *lexer) scanSpace() (containsNewline bool) {
	// lookahead is already lexed.
	// parse more, if any
	l.acceptWhile(func(r rune) bool {
		switch r {
		case ' ', '\t', '\r':
			return true
		case '\n':
			containsNewline = true
			return true
		default:
			return false
		}
	})
	return
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

func (l *lexer) scanLineComment() {
	// lookahead is already lexed.
	// parse more, if any
	l.acceptWhile(func(r rune) bool {
		return !(r == '\n' || r == EOF)
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

func (l *lexer) scanString(quote rune) {
	r := l.next()
	for r != quote {
		switch r {
		case '\n', EOF:
			// NOTE: invalid end of string handled by parser
			l.backupOne()
			return
		case '\\':
			r = l.next()
			switch r {
			case '\n', EOF:
				// NOTE: invalid end of string handled by parser
				l.backupOne()
				return
			}
		}
		r = l.next()
	}
}

func (l *lexer) scanBinaryRemainder() {
	l.acceptWhile(func(r rune) bool {
		return r == '0' || r == '1' || r == '_'
	})
}

func (l *lexer) scanOctalRemainder() {
	l.acceptWhile(func(r rune) bool {
		return (r >= '0' && r <= '7') || r == '_'
	})
}

func (l *lexer) scanHexadecimalRemainder() {
	l.acceptWhile(func(r rune) bool {
		return (r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'f') ||
			(r >= 'A' && r <= 'F') ||
			r == '_'
	})
}

func (l *lexer) scanDecimalOrFixedPointRemainder() TokenType {
	l.acceptWhile(isDecimalDigitOrUnderscore)
	r := l.next()
	if r == '.' {
		l.scanFixedPointRemainder()
		return TokenFixedPointNumberLiteral
	} else {
		l.backupOne()
		return TokenDecimalIntegerLiteral
	}
}

func (l *lexer) scanFixedPointRemainder() {
	r := l.next()
	if !isDecimalDigitOrUnderscore(r) {
		l.backupOne()
		l.emitError(fmt.Errorf("missing fractional digits"))
		return
	}
	l.acceptWhile(isDecimalDigitOrUnderscore)
}

func isDecimalDigitOrUnderscore(r rune) bool {
	return (r >= '0' && r <= '9') || r == '_'
}
