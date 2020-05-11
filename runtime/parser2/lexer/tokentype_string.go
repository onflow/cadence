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
	_ = x[TokenString-5]
	_ = x[TokenPlus-6]
	_ = x[TokenMinus-7]
	_ = x[TokenStar-8]
	_ = x[TokenSlash-9]
	_ = x[TokenPercent-10]
	_ = x[TokenNilCoalesce-11]
	_ = x[TokenParenOpen-12]
	_ = x[TokenParenClose-13]
	_ = x[TokenBraceOpen-14]
	_ = x[TokenBraceClose-15]
	_ = x[TokenBracketOpen-16]
	_ = x[TokenBracketClose-17]
	_ = x[TokenQuestionMark-18]
	_ = x[TokenComma-19]
	_ = x[TokenColon-20]
	_ = x[TokenDot-21]
	_ = x[TokenSemicolon-22]
	_ = x[TokenLeftArrow-23]
	_ = x[TokenLeftArrowExclamation-24]
	_ = x[TokenLess-25]
	_ = x[TokenLessEqual-26]
	_ = x[TokenLessLess-27]
	_ = x[TokenGreater-28]
	_ = x[TokenGreaterEqual-29]
	_ = x[TokenGreaterGreater-30]
	_ = x[TokenEqual-31]
	_ = x[TokenNot-32]
	_ = x[TokenNotEqual-33]
	_ = x[TokenBlockCommentStart-34]
	_ = x[TokenBlockCommentContent-35]
	_ = x[TokenBlockCommentEnd-36]
	_ = x[TokenAmpersand-37]
	_ = x[TokenAmpersandAmpersand-38]
	_ = x[TokenCaret-39]
	_ = x[TokenVerticalBar-40]
	_ = x[TokenVerticalBarVerticalBar-41]
	_ = x[TokenAt-42]
}

const _TokenType_name = "ErrorEOFSpaceNumberIdentifierStringPlusMinusStarSlashPercentNilCoalesceParenOpenParenCloseBraceOpenBraceCloseBracketOpenBracketCloseQuestionMarkCommaColonDotSemicolonLeftArrowLeftArrowExclamationLessLessEqualLessLessGreaterGreaterEqualGreaterGreaterEqualNotNotEqualBlockCommentStartBlockCommentContentBlockCommentEndAmpersandAmpersandAmpersandCaretVerticalBarVerticalBarVerticalBarAt"

var _TokenType_index = [...]uint16{0, 5, 8, 13, 19, 29, 35, 39, 44, 48, 53, 60, 71, 80, 90, 99, 109, 120, 132, 144, 149, 154, 157, 166, 175, 195, 199, 208, 216, 223, 235, 249, 254, 257, 265, 282, 301, 316, 325, 343, 348, 359, 381, 383}

func (i TokenType) String() string {
	if i >= TokenType(len(_TokenType_index)-1) {
		return "TokenType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TokenType_name[_TokenType_index[i]:_TokenType_index[i+1]]
}
