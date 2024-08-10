package lexer

import "github.com/onflow/cadence/runtime/ast"

type Trivia struct {
	Type TriviaType
	// Position within the source code (includes opening/closing comment characters in case of comment trivia type)
	ast.Range
}

type TriviaType uint8

const (
	TriviaTypeUnknown TriviaType = iota
	TriviaTypeInlineComment
	TriviaTypeMultiLineComment
	TriviaTypeNewLine
	TriviaTypeSpace
)

type TriviaCollection []Trivia

func (t TriviaCollection) Has(triviaType TriviaType) bool {
	for _, trivia := range t {
		if trivia.Type == triviaType {
			return true
		}
	}
	return false
}
