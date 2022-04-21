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
	"fmt"
	"unicode/utf8"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type position struct {
	line   int
	column int
}

type lexer struct {
	// input is the entire input string
	input string
	// startOffset is the start offset of the current word in the current line
	startOffset int
	// endOffset is the end offset of the current word in the current line
	endOffset int
	// prevEndOffset is the previous end offset, used for stepping back
	prevEndOffset int
	// current is the currently scanned rune
	current rune
	// prev is the previously scanned rune, used for stepping back
	prev rune
	// canBackup indicates whether stepping back is allowed
	canBackup bool
	// startPos is the start position of the current word
	startPos position
	// cursor is the offset in the token stream
	cursor int
	// tokens contains all tokens of the stream
	tokens []Token
	// tokenCount is the number of tokens in the stream
	tokenCount int
	// memoryGauge is used for metering memory usage
	memoryGauge common.MemoryGauge
}

var _ TokenStream = &lexer{}

func (l *lexer) Next() Token {
	if l.cursor >= l.tokenCount {

		// At the end of the token stream,
		// emit a synthetic EOF token

		endPos := l.endPos()
		pos := ast.NewPosition(
			l.memoryGauge,
			l.endOffset-1,
			endPos.line,
			endPos.column,
		)

		return Token{
			Type: TokenEOF,
			Range: ast.NewRange(
				l.memoryGauge,
				pos,
				pos,
			),
		}

	}
	token := l.tokens[l.cursor]
	l.cursor++
	return token
}

func (l *lexer) Input() string {
	return l.input
}

func (l *lexer) Cursor() int {
	return l.cursor
}

func (l *lexer) Revert(cursor int) {
	l.cursor = cursor
}

func Lex(input string, memoryGauge common.MemoryGauge) TokenStream {
	l := &lexer{
		input:         input,
		startPos:      position{line: 1},
		endOffset:     0,
		prevEndOffset: 0,
		current:       EOF,
		prev:          EOF,
		memoryGauge:   memoryGauge,
	}
	l.run(rootState)
	return l
}

// run executes the stateFn, which will scan the runes in the input
// and emit tokens.
//
// stateFn might return another stateFn to indicate further scanning work,
// or nil if there is no scanning work left to be done,
// i.e. run will keep running the returned stateFn until no more
// stateFn is returned, which for example happens when reaching the end of the file.
//
// When all stateFn have been executed, an EOF token is emitted.
func (l *lexer) run(state stateFn) {

	// catch panic exceptions, emit it to the tokens channel before
	// closing it
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch r := r.(type) {
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

func (l *lexer) wordLength() int {
	return l.endOffset - l.startOffset
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
		Range: ast.NewRange(
			l.memoryGauge,
			rangeStart,
			ast.NewPosition(
				l.memoryGauge,
				l.endOffset-1,
				endPos.line,
				endPos.column,
			),
		),
	}

	l.tokens = append(l.tokens, token)
	l.tokenCount = len(l.tokens)

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
	return ast.NewPosition(
		l.memoryGauge,
		l.startOffset,
		l.startPos.line,
		l.startPos.column,
	)
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
	if l.memoryGauge != nil {
		// Token value is always nil. Hence, only the wrapper is metered.
		// No memory is used for the 'value' potion.
		common.UseMemory(l.memoryGauge, common.SyntaxTokenMemoryUsage)
	}

	l.emit(ty, nil, l.startPosition(), true)
}

func (l *lexer) emitValue(ty TokenType) {
	if l.memoryGauge != nil {
		// Token wrapper
		common.UseMemory(l.memoryGauge, common.ValueTokenMemoryUsage)

		// Token content
		usage := l.tokenValueMemoryUsage(ty)
		common.UseMemory(l.memoryGauge, usage)
	}

	l.emit(ty, l.word(), l.startPosition(), true)
}

func (l *lexer) emitError(err error) {
	endPos := l.endPos()
	rangeStart := ast.NewPosition(
		l.memoryGauge,
		l.endOffset-1,
		endPos.line,
		endPos.column,
	)
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

// tokenValueMemoryUsage returns the memory usage, given the token type of the value.
// All tokens are retained in AST in its string representation. Hence, memory usage
// is always a string. However, string literals are special since they are
// later represented as graphemes.
//
func (l *lexer) tokenValueMemoryUsage(tokenType TokenType) common.MemoryUsage {
	tokenLength := l.wordLength()

	if tokenType == TokenString {
		return common.NewStringMemoryUsage(tokenLength)
	}

	return common.NewRawStringMemoryUsage(tokenLength)
}

func isDecimalDigitOrUnderscore(r rune) bool {
	return (r >= '0' && r <= '9') || r == '_'
}
