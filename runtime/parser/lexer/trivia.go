package lexer

type Trivia struct {
	Type TriviaType
	// The source text (includes opening/closing comment characters in case of comment trivia type)
	Text []byte
}

type TriviaType uint8

const (
	TriviaTypeUnknown TriviaType = iota
	TriviaTypeInlineComment
	TriviaTypeMultiLineComment
	TriviaTypeNewLine
	TriviaTypeSpace
)
