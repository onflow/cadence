package lexer

import "github.com/onflow/cadence/runtime/ast"

type Trivia struct {
	Type            TriviaType
	ContainsNewLine bool
	// Position within the source code (includes opening/closing comment characters in case of comment trivia type)
	ast.Range
}

type TriviaType uint8

const (
	TriviaTypeUnknown TriviaType = iota
	TriviaTypeInlineComment
	TriviaTypeMultiLineComment
	TriviaTypeSpace
)
