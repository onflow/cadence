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
	"sync"
	"unicode/utf8"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
)

// tokenLimit is a sensible limit for how many tokens may be emitted
const tokenLimit = 1 << 19

type TokenLimitReachedError struct {
	ast.Position
}

var _ error = TokenLimitReachedError{}
var _ errors.UserError = TokenLimitReachedError{}

func (TokenLimitReachedError) IsUserError() {}

func (TokenLimitReachedError) Error() string {
	return fmt.Sprintf("limit of %d tokens exceeded", tokenLimit)
}

type position struct {
	line   int
	column int
}

type lexerMode uint8

const (
	lexerModeNormal lexerMode = iota
	lexerModeStringInterpolation
)

type lexer struct {
	// memoryGauge is used for metering memory usage
	memoryGauge common.MemoryGauge
	// input is the entire input string
	input []byte
	// tokens contains all tokens of the stream
	tokens []Token
	// startPos is the start position of the current word
	startPos position
	// startOffset is the start offset of the current word in the current line
	startOffset int
	// endOffset is the end offset of the current word in the current line
	endOffset int
	// prevEndOffset is the previous end offset, used for stepping back
	prevEndOffset int
	// cursor is the offset in the token stream
	cursor int
	// tokenCount is the number of tokens in the stream
	tokenCount int
	// current is the currently scanned rune
	current rune
	// prev is the previously scanned rune, used for stepping back
	prev rune
	// canBackup indicates whether stepping back is allowed
	canBackup bool
	// lexer mode is used for string templates
	mode lexerMode
	// counts the number of unclosed brackets for string templates \((()))
	openBrackets int
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

func (l *lexer) Input() []byte {
	return l.input
}

func (l *lexer) Cursor() int {
	return l.cursor
}

func (l *lexer) Revert(cursor int) {
	l.cursor = cursor
}

func (l *lexer) clear() {
	l.startOffset = 0
	l.endOffset = 0
	l.prevEndOffset = 0
	l.current = EOF
	l.prev = EOF
	l.canBackup = false
	l.startPos = position{line: 1}
	l.cursor = 0
	l.tokens = l.tokens[:0]
	l.tokenCount = 0
	l.mode = lexerModeNormal
	l.openBrackets = 0
}

func (l *lexer) Reclaim() {
	pool.Put(l)
}

var pool = sync.Pool{
	New: func() any {
		return &lexer{
			tokens: make([]Token, 0, 2048),
		}
	},
}

func Lex(input []byte, memoryGauge common.MemoryGauge) (TokenStream, error) {
	l := pool.Get().(*lexer)
	l.clear()
	l.memoryGauge = memoryGauge
	l.input = input
	err := l.run(rootState)
	return l, err
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
func (l *lexer) run(state stateFn) (err error) {

	// catch panic exceptions, emit it to the tokens channel before
	// closing it
	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			// fatal errors and internal errors percolates up.
			// Note: not all fatal errors are internal errors.
			// e.g: memory limit exceeding is a fatal error, but also a user error.
			case error:
				err = r
			default:
				err = fmt.Errorf("lexer: %v", r)
			}
		}
	}()

	for state != nil {
		state = state(l)
	}

	return
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
		r, w = utf8.DecodeRune(l.input[endOffset:])
	}

	l.endOffset += w
	l.current = r

	return r
}

// backupOne steps back one rune.
// Can be called only once per call of next.
func (l *lexer) backupOne() {
	if !l.canBackup {
		// TODO: should this be an internal error?
		panic("second backup")
	}
	l.canBackup = false

	l.endOffset = l.prevEndOffset
	l.current = l.prev
}

func (l *lexer) word() []byte {
	start := l.startOffset
	end := l.endOffset
	return l.input[start:end]
}

// acceptOne reads one rune ahead.
// It returns true if the next rune matches with the input rune,
// otherwise it steps back one rune and returns false.
func (l *lexer) acceptOne(r rune) bool {
	if l.next() == r {
		return true
	}
	l.backupOne()
	return false
}

// emit writes a token to the channel.
func (l *lexer) emit(ty TokenType, spaceOrError any, rangeStart ast.Position, consume bool) {

	if len(l.tokens) >= tokenLimit {
		panic(TokenLimitReachedError{})
	}

	endPos := l.endPos()

	token := Token{
		Type:         ty,
		SpaceOrError: spaceOrError,
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
		r, _ := utf8.DecodeRune(l.input[l.endOffset-1:])

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
		b := l.input[offset:]
		r, w = utf8.DecodeRune(b)

		// fallback to 1 byte width if decoding fails
		if w <= 0 {
			w = 1
		}

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
	common.UseMemory(l.memoryGauge, common.TypeTokenMemoryUsage)

	l.emit(ty, nil, l.startPosition(), true)
}

func (l *lexer) emitError(err error) {
	common.UseMemory(l.memoryGauge, common.ErrorTokenMemoryUsage)

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
	l.acceptWhile(IsIdentifierRune)
}

func IsIdentifierRune(r rune) bool {
	return r >= 'a' && r <= 'z' ||
		r >= 'A' && r <= 'Z' ||
		r >= '0' && r <= '9' ||
		r == '_'
}

func IsValidIdentifier(s string) bool {
	for _, r := range s {
		if !IsIdentifierRune(r) {
			return false
		}
	}

	return true
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
			// might have to backup twice due to string template
			tmpBackupOffset := l.prevEndOffset
			tmpBackup := l.prev
			r = l.next()
			switch r {
			case '(':
				// string template, stop and set mode
				l.mode = lexerModeStringInterpolation
				// no need to update prev values because these next tokens will not backup
				l.endOffset = tmpBackupOffset
				l.current = tmpBackup
				l.canBackup = false
				return
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
