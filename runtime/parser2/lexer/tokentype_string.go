// Code generated by "stringer -type=TokenType -trimprefix Token"; DO NOT EDIT.

package lexer

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[TokenError-0]
	_ = x[TokenEOF-1]
	_ = x[TokenSpace-2]
	_ = x[TokenNumber-3]
	_ = x[TokenIdentifier-4]
	_ = x[TokenPlus-5]
	_ = x[TokenMinus-6]
	_ = x[TokenStar-7]
	_ = x[TokenSlash-8]
	_ = x[TokenNilCoalesce-9]
	_ = x[TokenParenOpen-10]
	_ = x[TokenParenClose-11]
	_ = x[TokenBraceOpen-12]
	_ = x[TokenBraceClose-13]
	_ = x[TokenBracketOpen-14]
	_ = x[TokenBracketClose-15]
	_ = x[TokenComma-16]
	_ = x[TokenColon-17]
}

const _TokenType_name = "ErrorEOFSpaceNumberIdentifierPlusMinusStarSlashNilCoalesceParenOpenParenCloseBraceOpenBraceCloseBracketOpenBracketCloseCommaColon"

var _TokenType_index = [...]uint8{0, 5, 8, 13, 19, 29, 33, 38, 42, 47, 58, 67, 77, 86, 96, 107, 119, 124, 129}

func (i TokenType) String() string {
	if i >= TokenType(len(_TokenType_index)-1) {
		return "TokenType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TokenType_name[_TokenType_index[i]:_TokenType_index[i+1]]
}
