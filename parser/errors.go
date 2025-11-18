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

package parser

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/parser/lexer"
	"github.com/onflow/cadence/pretty"
)

func expectedButGotToken(message string, tokenType lexer.TokenType) string {
	if tokenType == lexer.TokenEOF {
		return message
	}
	return fmt.Sprintf(
		"%s, got %s",
		message,
		tokenType,
	)
}

func keywordInsertion(keyword string, tokenType lexer.TokenType) string {
	if tokenType == lexer.TokenEOF {
		return fmt.Sprintf(" %s ", keyword)
	}
	return fmt.Sprintf("%s ", keyword)
}

func newLeftAttachedRange(pos ast.Position, code string) ast.Range {
	leftAttachedPos := pos.AttachLeft(code)
	return ast.Range{
		StartPos: leftAttachedPos,
		EndPos:   leftAttachedPos,
	}
}

// Error

type Error struct {
	Code   []byte
	Errors []error
}

func (e Error) Error() string {
	var sb strings.Builder
	sb.WriteString("Parsing failed:\n")
	printErr := pretty.NewErrorPrettyPrinter(&sb, false).
		PrettyPrintError(e, nil, map[common.Location][]byte{nil: e.Code})
	if printErr != nil {
		panic(printErr)
	}
	sb.WriteString(errors.ErrorPrompt)
	return sb.String()
}

func (e Error) ChildErrors() []error {
	return e.Errors
}

func (e Error) Unwrap() []error {
	return e.Errors
}

// ParserError

type ParseError interface {
	errors.UserError
	ast.HasPosition
	isParseError()
}

// SyntaxError

type SyntaxError struct {
	Message       string
	Secondary     string
	Migration     string
	Documentation string
	Pos           ast.Position
}

func NewSyntaxError(pos ast.Position, message string, params ...any) *SyntaxError {
	return &SyntaxError{
		Pos:     pos,
		Message: fmt.Sprintf(message, params...),
	}
}

var _ ParseError = &SyntaxError{}
var _ errors.UserError = &SyntaxError{}
var _ errors.HasMigrationNote = &SyntaxError{}
var _ errors.HasDocumentationLink = &SyntaxError{}
var _ errors.SecondaryError = &SyntaxError{}

func (*SyntaxError) isParseError() {}

func (*SyntaxError) IsUserError() {}

func (e *SyntaxError) StartPosition() ast.Position {
	return e.Pos
}

func (e *SyntaxError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *SyntaxError) Error() string {
	return e.Message
}

func (e *SyntaxError) SecondaryError() string {
	return e.Secondary
}

func (e *SyntaxError) MigrationNote() string {
	return e.Migration
}

func (e *SyntaxError) DocumentationLink() string {
	return e.Documentation
}

// Helper methods to set additional error information

func (e *SyntaxError) WithSecondary(secondary string) *SyntaxError {
	e.Secondary = secondary
	return e
}

func (e *SyntaxError) WithMigration(migration string) *SyntaxError {
	e.Migration = migration
	return e
}

func (e *SyntaxError) WithDocumentation(documentation string) *SyntaxError {
	e.Documentation = documentation
	return e
}

// InvalidIntegerLiteralError

type InvalidIntegerLiteralError struct {
	Literal                   string
	IntegerLiteralKind        common.IntegerLiteralKind
	InvalidIntegerLiteralKind InvalidNumberLiteralKind
	ast.Range
}

var _ ParseError = &InvalidIntegerLiteralError{}
var _ errors.UserError = &InvalidIntegerLiteralError{}
var _ errors.SecondaryError = &InvalidIntegerLiteralError{}
var _ errors.HasDocumentationLink = &InvalidIntegerLiteralError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidIntegerLiteralError{}

func (*InvalidIntegerLiteralError) isParseError() {}

func (*InvalidIntegerLiteralError) IsUserError() {}

func (e *InvalidIntegerLiteralError) Error() string {
	if e.IntegerLiteralKind == common.IntegerLiteralKindUnknown {
		return fmt.Sprintf(
			"invalid integer literal %#q: %s",
			e.Literal,
			e.InvalidIntegerLiteralKind.Description(),
		)
	}

	return fmt.Sprintf(
		"invalid %s integer literal %#q: %s",
		e.IntegerLiteralKind.Name(),
		e.Literal,
		e.InvalidIntegerLiteralKind.Description(),
	)
}

func (e *InvalidIntegerLiteralError) SecondaryError() string {
	switch e.InvalidIntegerLiteralKind {
	case InvalidNumberLiteralKindUnknown:
		return ""
	case InvalidNumberLiteralKindLeadingUnderscore:
		return "underscores (`_`) may not be used at the start of a number; remove the leading underscore"
	case InvalidNumberLiteralKindTrailingUnderscore:
		return "underscores (`_`) may not be used at the end of a number; remove the trailing underscore"
	case InvalidNumberLiteralKindUnknownPrefix:
		return "the prefix is unknown; use `0x` for hexadecimal, `0b` for binary, or `0o` for octal"
	case InvalidNumberLiteralKindMissingDigits:
		return "digits are missing after the prefix; add a `0` or other valid digits"
	}

	panic(errors.NewUnreachableError())
}

func (e *InvalidIntegerLiteralError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	switch e.InvalidIntegerLiteralKind {
	case InvalidNumberLiteralKindLeadingUnderscore:
		// Remove the leading underscore after the prefix
		// For literals like "0b_101010", we need to remove the underscore after the prefix
		if len(e.Literal) >= 3 && e.Literal[0] == '0' && e.Literal[2] == '_' {
			// Remove the underscore after the prefix (e.g., "0b_101010" -> "0b101010")
			replacement := e.Literal[:2] + e.Literal[3:]
			return []errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Remove leading underscore",
					TextEdits: []ast.TextEdit{
						{
							Replacement: replacement,
							Range:       e.Range,
						},
					},
				},
			}
		}

	case InvalidNumberLiteralKindTrailingUnderscore:
		// Remove the trailing underscore
		if len(e.Literal) > 0 && e.Literal[len(e.Literal)-1] == '_' {
			return []errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Remove trailing underscore",
					TextEdits: []ast.TextEdit{
						{
							Replacement: e.Literal[:len(e.Literal)-1],
							Range:       e.Range,
						},
					},
				},
			}
		}

	case InvalidNumberLiteralKindMissingDigits:
		// Add a "0" after the prefix
		return []errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Insert missing digit",
				TextEdits: []ast.TextEdit{
					{
						Replacement: e.Literal + "0",
						Range:       e.Range,
					},
				},
			},
		}

	case InvalidNumberLiteralKindUnknownPrefix:
		// Provide multiple fix options for common prefixes
		var fixes []errors.SuggestedFix[ast.TextEdit]

		// Extract the part after the unknown prefix (assuming it starts with "0")
		suffix := e.Literal
		if len(e.Literal) >= 2 && e.Literal[0] == '0' {
			suffix = e.Literal[2:] // Remove "0x" or similar
		}

		// Suggest hexadecimal
		fixes = append(fixes, errors.SuggestedFix[ast.TextEdit]{
			Message: "Use hexadecimal prefix (`0x`)",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "0x" + suffix,
					Range:       e.Range,
				},
			},
		})

		// Suggest binary
		fixes = append(fixes, errors.SuggestedFix[ast.TextEdit]{
			Message: "Use binary prefix (`0b`)",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "0b" + suffix,
					Range:       e.Range,
				},
			},
		})

		// Suggest octal
		fixes = append(fixes, errors.SuggestedFix[ast.TextEdit]{
			Message: "Use octal prefix (`0o`)",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "0o" + suffix,
					Range:       e.Range,
				},
			},
		})

		return fixes

	case InvalidNumberLiteralKindUnknown:
		// No fixes available for unknown errors
		return nil
	}

	return nil
}

func (*InvalidIntegerLiteralError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/booleans-numlits-ints"
}

// ExpressionDepthLimitReachedError is reported when the expression depth limit was reached
type ExpressionDepthLimitReachedError struct {
	Pos ast.Position
}

var _ ParseError = ExpressionDepthLimitReachedError{}
var _ errors.UserError = ExpressionDepthLimitReachedError{}
var _ errors.SecondaryError = ExpressionDepthLimitReachedError{}

func (ExpressionDepthLimitReachedError) isParseError() {}

func (ExpressionDepthLimitReachedError) IsUserError() {}

func (ExpressionDepthLimitReachedError) Error() string {
	return fmt.Sprintf(
		"expression too deeply nested, exceeded depth limit of %d",
		expressionDepthLimit,
	)
}

func (ExpressionDepthLimitReachedError) SecondaryError() string {
	return "consider extracting the sub-expressions out and storing the intermediate results in local variables"
}

func (e ExpressionDepthLimitReachedError) StartPosition() ast.Position {
	return e.Pos
}

func (e ExpressionDepthLimitReachedError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

// TypeDepthLimitReachedError is reported when the type depth limit was reached
type TypeDepthLimitReachedError struct {
	Pos ast.Position
}

var _ ParseError = TypeDepthLimitReachedError{}
var _ errors.UserError = TypeDepthLimitReachedError{}

func (TypeDepthLimitReachedError) isParseError() {}

func (TypeDepthLimitReachedError) IsUserError() {}

func (TypeDepthLimitReachedError) Error() string {
	return fmt.Sprintf(
		"type too deeply nested, exceeded depth limit of %d",
		typeDepthLimit,
	)
}

func (e TypeDepthLimitReachedError) StartPosition() ast.Position {
	return e.Pos
}

func (e TypeDepthLimitReachedError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

// UnexpectedEOFError is reported when the end of the program is reached unexpectedly
type UnexpectedEOFError struct {
	Pos ast.Position
}

var _ ParseError = UnexpectedEOFError{}
var _ errors.UserError = UnexpectedEOFError{}
var _ errors.SecondaryError = UnexpectedEOFError{}
var _ errors.HasDocumentationLink = UnexpectedEOFError{}

func (UnexpectedEOFError) isParseError() {}

func (UnexpectedEOFError) IsUserError() {}

func (e UnexpectedEOFError) StartPosition() ast.Position {
	return e.Pos
}

func (e UnexpectedEOFError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (UnexpectedEOFError) Error() string {
	return "unexpected end of program"
}

func (UnexpectedEOFError) SecondaryError() string {
	return "check for incomplete expressions, missing tokens, or unterminated strings/comments"
}

func (UnexpectedEOFError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// UnexpectedEOFExpectedTypeError is reported when the end of the program is reached unexpectedly,
// but a type was expected.
type UnexpectedEOFExpectedTypeError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedEOFExpectedTypeError{}
var _ errors.UserError = &UnexpectedEOFExpectedTypeError{}
var _ errors.SecondaryError = &UnexpectedEOFExpectedTypeError{}
var _ errors.HasDocumentationLink = &UnexpectedEOFExpectedTypeError{}

func (*UnexpectedEOFExpectedTypeError) isParseError() {}

func (*UnexpectedEOFExpectedTypeError) IsUserError() {}

func (e *UnexpectedEOFExpectedTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedEOFExpectedTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedEOFExpectedTypeError) Error() string {
	return "unexpected end of input, expected type"
}

func (*UnexpectedEOFExpectedTypeError) SecondaryError() string {
	return "check for incomplete expressions, missing tokens, or unterminated strings/comments"
}

func (*UnexpectedEOFExpectedTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// UnexpectedEOFExpectedTypeAnnotationError is reported when the end of the program is reached unexpectedly,
// but a type annotation was expected.
type UnexpectedEOFExpectedTypeAnnotationError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedEOFExpectedTypeAnnotationError{}
var _ errors.UserError = &UnexpectedEOFExpectedTypeAnnotationError{}
var _ errors.SecondaryError = &UnexpectedEOFExpectedTypeAnnotationError{}
var _ errors.HasDocumentationLink = &UnexpectedEOFExpectedTypeAnnotationError{}

func (*UnexpectedEOFExpectedTypeAnnotationError) isParseError() {}

func (*UnexpectedEOFExpectedTypeAnnotationError) IsUserError() {}

func (e *UnexpectedEOFExpectedTypeAnnotationError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedEOFExpectedTypeAnnotationError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedEOFExpectedTypeAnnotationError) Error() string {
	return "unexpected end of input, expected type annotation"
}

func (*UnexpectedEOFExpectedTypeAnnotationError) SecondaryError() string {
	return "check for incomplete expressions, missing tokens, or unterminated strings/comments"
}

func (*UnexpectedEOFExpectedTypeAnnotationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// StatementSeparationError is reported when two statements on the same line
// are not separated by a semicolon.
type StatementSeparationError struct {
	Pos ast.Position
}

var _ ParseError = &StatementSeparationError{}
var _ errors.UserError = &StatementSeparationError{}
var _ errors.SecondaryError = &StatementSeparationError{}
var _ errors.HasDocumentationLink = &StatementSeparationError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &StatementSeparationError{}

func (*StatementSeparationError) isParseError() {}

func (*StatementSeparationError) IsUserError() {}

func (e *StatementSeparationError) StartPosition() ast.Position {
	return e.Pos
}

func (e *StatementSeparationError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*StatementSeparationError) Error() string {
	return "statements on the same line must be separated with a semicolon (`;`)"
}

func (*StatementSeparationError) SecondaryError() string {
	return "add a semicolon (`;`) between statements or place each statement on a separate line"
}

func (e *StatementSeparationError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert semicolon",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ";",
					Range:     newLeftAttachedRange(e.Pos, code),
				},
			},
		},
	}
}

func (*StatementSeparationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax#semicolons"
}

// MissingCommaInParameterListError

type MissingCommaInParameterListError struct {
	Pos ast.Position
}

var _ ParseError = &MissingCommaInParameterListError{}
var _ errors.UserError = &MissingCommaInParameterListError{}
var _ errors.SecondaryError = &MissingCommaInParameterListError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingCommaInParameterListError{}
var _ errors.HasDocumentationLink = &MissingCommaInParameterListError{}

func (*MissingCommaInParameterListError) isParseError() {}

func (*MissingCommaInParameterListError) IsUserError() {}

func (e *MissingCommaInParameterListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingCommaInParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingCommaInParameterListError) Error() string {
	return "missing comma (`,`) after parameter"
}

func (*MissingCommaInParameterListError) SecondaryError() string {
	return "add a comma (`,`) to separate parameters in the parameter list"
}

func (e *MissingCommaInParameterListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert comma",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ",",
					Range:     newLeftAttachedRange(e.Pos, code),
				},
			},
		},
	}
}

func (*MissingCommaInParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions#function-declarations"
}

// MissingStartOfParameterListError is reported when a parameter list is missing a start token.
type MissingStartOfParameterListError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingStartOfParameterListError{}
var _ errors.UserError = &MissingStartOfParameterListError{}
var _ errors.SecondaryError = &MissingStartOfParameterListError{}
var _ errors.HasDocumentationLink = &MissingStartOfParameterListError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingStartOfParameterListError{}

func (*MissingStartOfParameterListError) isParseError() {}

func (*MissingStartOfParameterListError) IsUserError() {}

func (e *MissingStartOfParameterListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingStartOfParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingStartOfParameterListError) Error() string {
	return expectedButGotToken(
		"expected open parenthesis (`(`) as start of parameter list",
		e.GotToken.Type,
	)
}

func (*MissingStartOfParameterListError) SecondaryError() string {
	return "function parameters must be enclosed in parentheses (`(...)`); " +
		"add the missing opening parenthesis (`(`)"
}

func (*MissingStartOfParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

func (e *MissingStartOfParameterListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert opening parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "(",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingStartOfAuthorizationError is reported when an authorization list is missing a start token.
type MissingStartOfAuthorizationError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingStartOfAuthorizationError{}
var _ errors.UserError = &MissingStartOfAuthorizationError{}
var _ errors.SecondaryError = &MissingStartOfAuthorizationError{}
var _ errors.HasDocumentationLink = &MissingStartOfAuthorizationError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingStartOfAuthorizationError{}

func (*MissingStartOfAuthorizationError) isParseError() {}

func (*MissingStartOfAuthorizationError) IsUserError() {}

func (e *MissingStartOfAuthorizationError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingStartOfAuthorizationError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingStartOfAuthorizationError) Error() string {
	return expectedButGotToken(
		"expected open parenthesis (`(`) as start of authorization",
		e.GotToken.Type,
	)
}

func (*MissingStartOfAuthorizationError) SecondaryError() string {
	return "authorized references must have an authorization list enclosed in parentheses (`auth(...)`)"
}

func (*MissingStartOfAuthorizationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/references#authorized-references"
}

func (e *MissingStartOfAuthorizationError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert opening parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "(",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// UnexpectedTokenInParameterListError is reported when an unexpected token is found in a parameter list.
type UnexpectedTokenInParameterListError struct {
	GotToken lexer.Token
}

var _ ParseError = &UnexpectedTokenInParameterListError{}
var _ errors.UserError = &UnexpectedTokenInParameterListError{}
var _ errors.SecondaryError = &UnexpectedTokenInParameterListError{}
var _ errors.HasDocumentationLink = &UnexpectedTokenInParameterListError{}

func (*UnexpectedTokenInParameterListError) isParseError() {}

func (*UnexpectedTokenInParameterListError) IsUserError() {}

func (e *UnexpectedTokenInParameterListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *UnexpectedTokenInParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *UnexpectedTokenInParameterListError) Error() string {
	return expectedButGotToken(
		"expected parameter or end of parameter list (`)`)",
		e.GotToken.Type,
	)
}

func (*UnexpectedTokenInParameterListError) SecondaryError() string {
	return "parameters must be separated by commas (`,`), " +
		"and the list must end with a closing parenthesis (`)`)"
}

func (*UnexpectedTokenInParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// MissingClosingParenInParameterListError is reported when a parameter list is missing a closing parenthesis.
type MissingClosingParenInParameterListError struct {
	Pos ast.Position
}

var _ ParseError = &MissingClosingParenInParameterListError{}
var _ errors.UserError = &MissingClosingParenInParameterListError{}
var _ errors.SecondaryError = &MissingClosingParenInParameterListError{}
var _ errors.HasDocumentationLink = &MissingClosingParenInParameterListError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingParenInParameterListError{}

func (*MissingClosingParenInParameterListError) isParseError() {}

func (*MissingClosingParenInParameterListError) IsUserError() {}

func (e *MissingClosingParenInParameterListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingClosingParenInParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingClosingParenInParameterListError) Error() string {
	return "missing closing parenthesis (`)`) at end of parameter list"
}

func (*MissingClosingParenInParameterListError) SecondaryError() string {
	return "function parameter lists must be properly closed with a closing parenthesis (`)`); " +
		"add the missing closing parenthesis (`)`)"
}

func (e *MissingClosingParenInParameterListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ")",
					Range:     newLeftAttachedRange(e.Pos, code),
				},
			},
		},
	}
}

func (*MissingClosingParenInParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// ExpectedCommaOrEndOfParameterListError is reported when a comma or the end of a parameter list is expected.
type ExpectedCommaOrEndOfParameterListError struct {
	GotToken lexer.Token
}

var _ ParseError = &ExpectedCommaOrEndOfParameterListError{}
var _ errors.UserError = &ExpectedCommaOrEndOfParameterListError{}
var _ errors.SecondaryError = &ExpectedCommaOrEndOfParameterListError{}
var _ errors.HasDocumentationLink = &ExpectedCommaOrEndOfParameterListError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &ExpectedCommaOrEndOfParameterListError{}

func (*ExpectedCommaOrEndOfParameterListError) isParseError() {}

func (*ExpectedCommaOrEndOfParameterListError) IsUserError() {}

func (e *ExpectedCommaOrEndOfParameterListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *ExpectedCommaOrEndOfParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *ExpectedCommaOrEndOfParameterListError) Error() string {
	return expectedButGotToken(
		"expected comma (`,`), or closing parenthesis (`)`) at end of parameter list",
		e.GotToken.Type,
	)
}

func (*ExpectedCommaOrEndOfParameterListError) SecondaryError() string {
	return "multiple parameters must be separated by commas (`,`), " +
		"and the parameter list must end with a closing parenthesis (`)`)"
}

func (*ExpectedCommaOrEndOfParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

func (e *ExpectedCommaOrEndOfParameterListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert comma",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ",",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingColonAfterParameterNameError is reported when a colon is missing after a parameter name.
type MissingColonAfterParameterNameError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingColonAfterParameterNameError{}
var _ errors.UserError = &MissingColonAfterParameterNameError{}
var _ errors.SecondaryError = &MissingColonAfterParameterNameError{}
var _ errors.HasDocumentationLink = &MissingColonAfterParameterNameError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingColonAfterParameterNameError{}

func (*MissingColonAfterParameterNameError) isParseError() {}

func (*MissingColonAfterParameterNameError) IsUserError() {}

func (e *MissingColonAfterParameterNameError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingColonAfterParameterNameError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingColonAfterParameterNameError) Error() string {
	return expectedButGotToken(
		"expected colon (`:`) after parameter name",
		e.GotToken.Type,
	)
}

func (*MissingColonAfterParameterNameError) SecondaryError() string {
	return "function parameters must have a colon (`:`) after the parameter name; " +
		"add a colon (`:`) after the parameter name"
}

func (*MissingColonAfterParameterNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

func (e *MissingColonAfterParameterNameError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert colon",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ":",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingDefaultArgumentError is reported when a default argument is missing after a type annotation.
type MissingDefaultArgumentError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingDefaultArgumentError{}
var _ errors.UserError = &MissingDefaultArgumentError{}
var _ errors.SecondaryError = &MissingDefaultArgumentError{}
var _ errors.HasDocumentationLink = &MissingDefaultArgumentError{}

func (*MissingDefaultArgumentError) isParseError() {}

func (*MissingDefaultArgumentError) IsUserError() {}

func (e *MissingDefaultArgumentError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingDefaultArgumentError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingDefaultArgumentError) Error() string {
	return expectedButGotToken(
		"expected a default argument after type annotation",
		e.GotToken.Type,
	)
}

func (*MissingDefaultArgumentError) SecondaryError() string {
	return "default arguments must be specified with an equals sign (`=`) followed by the default value; " +
		"add the default argument"
}

func (*MissingDefaultArgumentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// UnexpectedDefaultArgumentError is reported when a default argument is found in an unexpected context.
type UnexpectedDefaultArgumentError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedDefaultArgumentError{}
var _ errors.UserError = &UnexpectedDefaultArgumentError{}
var _ errors.SecondaryError = &UnexpectedDefaultArgumentError{}
var _ errors.HasDocumentationLink = &UnexpectedDefaultArgumentError{}

func (*UnexpectedDefaultArgumentError) isParseError() {}

func (*UnexpectedDefaultArgumentError) IsUserError() {}

func (e *UnexpectedDefaultArgumentError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedDefaultArgumentError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedDefaultArgumentError) Error() string {
	return "cannot define a default argument for this function"
}

func (*UnexpectedDefaultArgumentError) SecondaryError() string {
	return "default arguments are only allowed in `ResourceDestroyed` events, not in functions; remove the default argument"
}

func (*UnexpectedDefaultArgumentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// MissingCommaInTypeParameterListError

type MissingCommaInTypeParameterListError struct {
	Pos ast.Position
}

var _ ParseError = &MissingCommaInTypeParameterListError{}
var _ errors.UserError = &MissingCommaInTypeParameterListError{}
var _ errors.SecondaryError = &MissingCommaInTypeParameterListError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingCommaInTypeParameterListError{}
var _ errors.HasDocumentationLink = &MissingCommaInTypeParameterListError{}

func (*MissingCommaInTypeParameterListError) isParseError() {}

func (*MissingCommaInTypeParameterListError) IsUserError() {}

func (e *MissingCommaInTypeParameterListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingCommaInTypeParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingCommaInTypeParameterListError) Error() string {
	return "missing comma (`,`) after type parameter"
}

func (*MissingCommaInTypeParameterListError) SecondaryError() string {
	return "add a comma to separate type parameters in the type parameter list"
}

func (e *MissingCommaInTypeParameterListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert comma",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ",",
					Range:     newLeftAttachedRange(e.Pos, code),
				},
			},
		},
	}
}

func (*MissingCommaInTypeParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions#function-declarations"
}

// UnexpectedTokenInTypeParameterListError is reported when an unexpected token is found in a type parameter list.
type UnexpectedTokenInTypeParameterListError struct {
	GotToken lexer.Token
}

var _ ParseError = &UnexpectedTokenInTypeParameterListError{}
var _ errors.UserError = &UnexpectedTokenInTypeParameterListError{}
var _ errors.SecondaryError = &UnexpectedTokenInTypeParameterListError{}
var _ errors.HasDocumentationLink = &UnexpectedTokenInTypeParameterListError{}

func (*UnexpectedTokenInTypeParameterListError) isParseError() {}

func (*UnexpectedTokenInTypeParameterListError) IsUserError() {}

func (e *UnexpectedTokenInTypeParameterListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *UnexpectedTokenInTypeParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *UnexpectedTokenInTypeParameterListError) Error() string {
	return expectedButGotToken(
		"expected type parameter, or closing angle bracket (`>`) to end the type parameter list",
		e.GotToken.Type,
	)
}

func (*UnexpectedTokenInTypeParameterListError) SecondaryError() string {
	return "type parameters must be separated by commas (`,`), " +
		"and the list must end with a closing angle bracket (`>`)"
}

func (*UnexpectedTokenInTypeParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// MissingClosingGreaterInTypeParameterListError is reported when a type parameter list is missing a closing angle bracket.
type MissingClosingGreaterInTypeParameterListError struct {
	Pos ast.Position
}

var _ ParseError = &MissingClosingGreaterInTypeParameterListError{}
var _ errors.UserError = &MissingClosingGreaterInTypeParameterListError{}
var _ errors.SecondaryError = &MissingClosingGreaterInTypeParameterListError{}
var _ errors.HasDocumentationLink = &MissingClosingGreaterInTypeParameterListError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingGreaterInTypeParameterListError{}

func (*MissingClosingGreaterInTypeParameterListError) isParseError() {}

func (*MissingClosingGreaterInTypeParameterListError) IsUserError() {}

func (e *MissingClosingGreaterInTypeParameterListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingClosingGreaterInTypeParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingClosingGreaterInTypeParameterListError) Error() string {
	return "missing closing angle bracket (`>`) at end of type parameter list"
}

func (*MissingClosingGreaterInTypeParameterListError) SecondaryError() string {
	return "type parameters must be separated by commas (`,`), " +
		"and the list must end with a closing angle bracket (`>`)"
}

func (e *MissingClosingGreaterInTypeParameterListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing angle bracket",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ">",
					Range:     newLeftAttachedRange(e.Pos, code),
				},
			},
		},
	}
}

func (*MissingClosingGreaterInTypeParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// MissingClosingGreaterInTypeArgumentsError is reported when a type arguments is missing a closing angle bracket.
type MissingClosingGreaterInTypeArgumentsError struct {
	Pos ast.Position
}

var _ ParseError = &MissingClosingGreaterInTypeArgumentsError{}
var _ errors.UserError = &MissingClosingGreaterInTypeArgumentsError{}
var _ errors.SecondaryError = &MissingClosingGreaterInTypeArgumentsError{}
var _ errors.HasDocumentationLink = &MissingClosingGreaterInTypeArgumentsError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingGreaterInTypeArgumentsError{}

func (*MissingClosingGreaterInTypeArgumentsError) isParseError() {}

func (*MissingClosingGreaterInTypeArgumentsError) IsUserError() {}

func (e *MissingClosingGreaterInTypeArgumentsError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingClosingGreaterInTypeArgumentsError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingClosingGreaterInTypeArgumentsError) Error() string {
	return "missing closing angle bracket (`>`) at end of type arguments"
}

func (*MissingClosingGreaterInTypeArgumentsError) SecondaryError() string {
	return "type arguments must be enclosed in angle brackets (`<...>`)"
}

func (e *MissingClosingGreaterInTypeArgumentsError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing angle bracket",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ">",
					Range:     newLeftAttachedRange(e.Pos, code),
				},
			},
		},
	}
}

func (*MissingClosingGreaterInTypeArgumentsError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// ExpectedCommaOrEndOfTypeParameterListError is reported when a comma or the end of a type parameter list is expected.
type ExpectedCommaOrEndOfTypeParameterListError struct {
	GotToken lexer.Token
}

var _ ParseError = &ExpectedCommaOrEndOfTypeParameterListError{}
var _ errors.UserError = &ExpectedCommaOrEndOfTypeParameterListError{}
var _ errors.SecondaryError = &ExpectedCommaOrEndOfTypeParameterListError{}
var _ errors.HasDocumentationLink = &ExpectedCommaOrEndOfTypeParameterListError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &ExpectedCommaOrEndOfTypeParameterListError{}

func (*ExpectedCommaOrEndOfTypeParameterListError) isParseError() {}

func (*ExpectedCommaOrEndOfTypeParameterListError) IsUserError() {}

func (e *ExpectedCommaOrEndOfTypeParameterListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *ExpectedCommaOrEndOfTypeParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *ExpectedCommaOrEndOfTypeParameterListError) Error() string {
	return expectedButGotToken(
		"expected comma (`,`), or closing angle bracket (`>`) to end the type parameter list",
		e.GotToken.Type,
	)
}

func (*ExpectedCommaOrEndOfTypeParameterListError) SecondaryError() string {
	return "type parameters must be separated by commas (`,`), " +
		"and the list must end with a closing angle bracket (`>`)"
}

func (*ExpectedCommaOrEndOfTypeParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *ExpectedCommaOrEndOfTypeParameterListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert comma",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ",",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// InvalidTypeParameterNameError is reported when a type parameter has an invalid name.
type InvalidTypeParameterNameError struct {
	GotToken lexer.Token
}

var _ ParseError = &InvalidTypeParameterNameError{}
var _ errors.UserError = &InvalidTypeParameterNameError{}
var _ errors.SecondaryError = &InvalidTypeParameterNameError{}
var _ errors.HasDocumentationLink = &InvalidTypeParameterNameError{}

func (*InvalidTypeParameterNameError) isParseError() {}

func (*InvalidTypeParameterNameError) IsUserError() {}

func (e *InvalidTypeParameterNameError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *InvalidTypeParameterNameError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *InvalidTypeParameterNameError) Error() string {
	return expectedButGotToken(
		"expected type parameter name",
		e.GotToken.Type,
	)
}

func (*InvalidTypeParameterNameError) SecondaryError() string {
	return "type parameters must have a valid identifier name"
}

func (*InvalidTypeParameterNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// DuplicateExecuteBlockError is reported when a transaction declaration has a second execute block.
type DuplicateExecuteBlockError struct {
	Pos ast.Position
}

var _ ParseError = &DuplicateExecuteBlockError{}
var _ errors.UserError = &DuplicateExecuteBlockError{}
var _ errors.SecondaryError = &DuplicateExecuteBlockError{}
var _ errors.HasDocumentationLink = &DuplicateExecuteBlockError{}

func (*DuplicateExecuteBlockError) isParseError() {}

func (*DuplicateExecuteBlockError) IsUserError() {}

func (e *DuplicateExecuteBlockError) StartPosition() ast.Position {
	return e.Pos
}

func (e *DuplicateExecuteBlockError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *DuplicateExecuteBlockError) Error() string {
	return "unexpected second `execute` block"
}

func (*DuplicateExecuteBlockError) SecondaryError() string {
	return "transaction declarations can only have one `execute` block"
}

func (*DuplicateExecuteBlockError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
}

// DuplicatePostConditionsError is reported when a transaction declaration has a second post-conditions block.
type DuplicatePostConditionsError struct {
	Pos ast.Position
}

var _ ParseError = &DuplicatePostConditionsError{}
var _ errors.UserError = &DuplicatePostConditionsError{}
var _ errors.SecondaryError = &DuplicatePostConditionsError{}
var _ errors.HasDocumentationLink = &DuplicatePostConditionsError{}

func (*DuplicatePostConditionsError) isParseError() {}

func (*DuplicatePostConditionsError) IsUserError() {}

func (e *DuplicatePostConditionsError) StartPosition() ast.Position {
	return e.Pos
}

func (e *DuplicatePostConditionsError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *DuplicatePostConditionsError) Error() string {
	return "unexpected second post-conditions"
}

func (*DuplicatePostConditionsError) SecondaryError() string {
	return "transaction declarations can only have one `post` block"
}

func (*DuplicatePostConditionsError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
}

// ExpectedPrepareOrExecuteError is reported when a 'prepare' or 'execute' block is expected in a transaction.
type ExpectedPrepareOrExecuteError struct {
	GotIdentifier string
	Pos           ast.Position
}

var _ ParseError = &ExpectedPrepareOrExecuteError{}
var _ errors.UserError = &ExpectedPrepareOrExecuteError{}
var _ errors.SecondaryError = &ExpectedPrepareOrExecuteError{}
var _ errors.HasDocumentationLink = &ExpectedPrepareOrExecuteError{}

func (*ExpectedPrepareOrExecuteError) isParseError() {}

func (*ExpectedPrepareOrExecuteError) IsUserError() {}

func (e *ExpectedPrepareOrExecuteError) StartPosition() ast.Position {
	return e.Pos
}

func (e *ExpectedPrepareOrExecuteError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.GotIdentifier)
	return e.Pos.Shifted(memoryGauge, length-1)
}

func (e *ExpectedPrepareOrExecuteError) Error() string {
	return fmt.Sprintf(
		"unexpected identifier: expected keyword `prepare` or `execute`, got %#q",
		e.GotIdentifier,
	)
}

func (*ExpectedPrepareOrExecuteError) SecondaryError() string {
	return "the first block in a transaction declaration must be a `prepare` or an `execute` block"
}

func (*ExpectedPrepareOrExecuteError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
}

// ExpectedExecuteOrPostError is reported when an 'execute' or 'post' block is expected in a transaction.
type ExpectedExecuteOrPostError struct {
	GotIdentifier string
	Pos           ast.Position
}

var _ ParseError = &ExpectedExecuteOrPostError{}
var _ errors.UserError = &ExpectedExecuteOrPostError{}
var _ errors.SecondaryError = &ExpectedExecuteOrPostError{}
var _ errors.HasDocumentationLink = &ExpectedExecuteOrPostError{}

func (*ExpectedExecuteOrPostError) isParseError() {}

func (*ExpectedExecuteOrPostError) IsUserError() {}

func (e *ExpectedExecuteOrPostError) StartPosition() ast.Position {
	return e.Pos
}

func (e *ExpectedExecuteOrPostError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.GotIdentifier)
	return e.Pos.Shifted(memoryGauge, length-1)
}

func (e *ExpectedExecuteOrPostError) Error() string {
	return fmt.Sprintf(
		"unexpected identifier: expected keyword `execute` or `post`, got %#q",
		e.GotIdentifier,
	)
}

func (*ExpectedExecuteOrPostError) SecondaryError() string {
	return "transaction declarations may only define an `execute` or a `post` block here"
}

func (*ExpectedExecuteOrPostError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
}

// ExpectedCaseOrDefaultError is reported when a 'case' or 'default' is expected in a switch statement.
type ExpectedCaseOrDefaultError struct {
	GotToken lexer.Token
}

var _ ParseError = &ExpectedCaseOrDefaultError{}
var _ errors.UserError = &ExpectedCaseOrDefaultError{}
var _ errors.SecondaryError = &ExpectedCaseOrDefaultError{}
var _ errors.HasDocumentationLink = &ExpectedCaseOrDefaultError{}

func (*ExpectedCaseOrDefaultError) isParseError() {}

func (*ExpectedCaseOrDefaultError) IsUserError() {}

func (e *ExpectedCaseOrDefaultError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *ExpectedCaseOrDefaultError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *ExpectedCaseOrDefaultError) Error() string {
	return expectedButGotToken(
		"unexpected token: expected keyword `case` or `default`",
		e.GotToken.Type,
	)
}

func (*ExpectedCaseOrDefaultError) SecondaryError() string {
	return "switch statements can only contain `case` and `default` blocks"
}

func (*ExpectedCaseOrDefaultError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/control-flow#switch"
}

// MissingColonInSwitchCaseError is reported when a colon is missing in a switch case.
type MissingColonInSwitchCaseError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingColonInSwitchCaseError{}
var _ errors.UserError = &MissingColonInSwitchCaseError{}
var _ errors.SecondaryError = &MissingColonInSwitchCaseError{}
var _ errors.HasDocumentationLink = &MissingColonInSwitchCaseError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingColonInSwitchCaseError{}

func (*MissingColonInSwitchCaseError) isParseError() {}

func (*MissingColonInSwitchCaseError) IsUserError() {}

func (e *MissingColonInSwitchCaseError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingColonInSwitchCaseError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingColonInSwitchCaseError) Error() string {
	return expectedButGotToken(
		"expected colon (`:`)",
		e.GotToken.Type,
	)
}

func (*MissingColonInSwitchCaseError) SecondaryError() string {
	return "a colon (`:`) is required after the case expression in a switch statement"
}

func (e *MissingColonInSwitchCaseError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert colon",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ":",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

func (*MissingColonInSwitchCaseError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/control-flow#switch"
}

// MissingFromKeywordInRemoveStatementError is reported when the 'from' keyword is missing in a remove statement.
type MissingFromKeywordInRemoveStatementError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingFromKeywordInRemoveStatementError{}
var _ errors.UserError = &MissingFromKeywordInRemoveStatementError{}
var _ errors.SecondaryError = &MissingFromKeywordInRemoveStatementError{}
var _ errors.HasDocumentationLink = &MissingFromKeywordInRemoveStatementError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingFromKeywordInRemoveStatementError{}

func (*MissingFromKeywordInRemoveStatementError) isParseError() {}

func (*MissingFromKeywordInRemoveStatementError) IsUserError() {}

func (e *MissingFromKeywordInRemoveStatementError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingFromKeywordInRemoveStatementError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingFromKeywordInRemoveStatementError) Error() string {
	return expectedButGotToken(
		"expected keyword `from`",
		e.GotToken.Type,
	)
}

func (*MissingFromKeywordInRemoveStatementError) SecondaryError() string {
	return "the `remove` statement requires the `from` keyword to specify the value to remove the attachment from"
}

func (e *MissingFromKeywordInRemoveStatementError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `from`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: keywordInsertion(KeywordFrom, e.GotToken.Type),
					Range: ast.Range{
						StartPos: e.GotToken.StartPos,
						EndPos:   e.GotToken.StartPos,
					},
				},
			},
		},
	}
}

func (*MissingFromKeywordInRemoveStatementError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments#removing-attachments"
}

// MissingToKeywordInAttachExpressionError is reported when the 'to' keyword is missing in an attach expression.
type MissingToKeywordInAttachExpressionError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingToKeywordInAttachExpressionError{}
var _ errors.UserError = &MissingToKeywordInAttachExpressionError{}
var _ errors.SecondaryError = &MissingToKeywordInAttachExpressionError{}
var _ errors.HasDocumentationLink = &MissingToKeywordInAttachExpressionError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingToKeywordInAttachExpressionError{}

func (*MissingToKeywordInAttachExpressionError) isParseError() {}

func (*MissingToKeywordInAttachExpressionError) IsUserError() {}

func (e *MissingToKeywordInAttachExpressionError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingToKeywordInAttachExpressionError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingToKeywordInAttachExpressionError) Error() string {
	return expectedButGotToken(
		"expected keyword `to`",
		e.GotToken.Type,
	)
}

func (*MissingToKeywordInAttachExpressionError) SecondaryError() string {
	return "the `attach` expression requires the `to` keyword to specify the value to attach to"
}

func (e *MissingToKeywordInAttachExpressionError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `to`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: keywordInsertion(KeywordTo, e.GotToken.Type),
					Range: ast.Range{
						StartPos: e.GotToken.StartPos,
						EndPos:   e.GotToken.StartPos,
					},
				},
			},
		},
	}
}

func (*MissingToKeywordInAttachExpressionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments#creating-attachments"
}

// InvalidAttachmentRemovalTypeError is reported when a removed attachment type is not nominal.
type InvalidAttachmentRemovalTypeError struct {
	ast.Range
}

var _ ParseError = &InvalidAttachmentRemovalTypeError{}
var _ errors.UserError = &InvalidAttachmentRemovalTypeError{}
var _ errors.SecondaryError = &InvalidAttachmentRemovalTypeError{}
var _ errors.HasDocumentationLink = &InvalidAttachmentRemovalTypeError{}

func (*InvalidAttachmentRemovalTypeError) isParseError() {}

func (*InvalidAttachmentRemovalTypeError) IsUserError() {}

func (e *InvalidAttachmentRemovalTypeError) Error() string {
	return "expected attachment nominal type"
}

func (*InvalidAttachmentRemovalTypeError) SecondaryError() string {
	return "only attachment types can be removed"
}

func (*InvalidAttachmentRemovalTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments#removing-attachments"
}

// UnexpectedCommaInDictionaryTypeError is reported when a comma is found in a dictionary type.
type UnexpectedCommaInDictionaryTypeError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedCommaInDictionaryTypeError{}
var _ errors.UserError = &UnexpectedCommaInDictionaryTypeError{}
var _ errors.SecondaryError = &UnexpectedCommaInDictionaryTypeError{}
var _ errors.HasDocumentationLink = &UnexpectedCommaInDictionaryTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &UnexpectedCommaInDictionaryTypeError{}

func (*UnexpectedCommaInDictionaryTypeError) isParseError() {}

func (*UnexpectedCommaInDictionaryTypeError) IsUserError() {}

func (e *UnexpectedCommaInDictionaryTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedCommaInDictionaryTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedCommaInDictionaryTypeError) Error() string {
	return "unexpected comma (`,`) in dictionary type"
}

func (*UnexpectedCommaInDictionaryTypeError) SecondaryError() string {
	return "dictionary types use a colon (`:`) to separate key and value types, not commas (`,`)"
}

func (e *UnexpectedCommaInDictionaryTypeError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove comma",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range: ast.Range{
						StartPos: e.Pos,
						EndPos:   e.Pos,
					},
				},
			},
		},
	}
}

func (*UnexpectedCommaInDictionaryTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/dictionaries"
}

// UnexpectedColonInDictionaryTypeError is reported when a colon is found at an unexpected position in a dictionary type.
type UnexpectedColonInDictionaryTypeError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedColonInDictionaryTypeError{}
var _ errors.UserError = &UnexpectedColonInDictionaryTypeError{}
var _ errors.SecondaryError = &UnexpectedColonInDictionaryTypeError{}
var _ errors.HasDocumentationLink = &UnexpectedColonInDictionaryTypeError{}

func (*UnexpectedColonInDictionaryTypeError) isParseError() {}

func (*UnexpectedColonInDictionaryTypeError) IsUserError() {}

func (e *UnexpectedColonInDictionaryTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedColonInDictionaryTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedColonInDictionaryTypeError) Error() string {
	return "unexpected colon (`:`) in dictionary type"
}

func (*UnexpectedColonInDictionaryTypeError) SecondaryError() string {
	return "dictionary types use a colon (`:`) to separate key and value types, " +
		"both types must be provided (`{K: V}`)"
}

func (*UnexpectedColonInDictionaryTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/dictionaries"
}

// MultipleColonInDictionaryTypeError is reported when more than one colon is found in a dictionary type.
type MultipleColonInDictionaryTypeError struct {
	Pos ast.Position
}

var _ ParseError = &MultipleColonInDictionaryTypeError{}
var _ errors.UserError = &MultipleColonInDictionaryTypeError{}
var _ errors.SecondaryError = &MultipleColonInDictionaryTypeError{}
var _ errors.HasDocumentationLink = &MultipleColonInDictionaryTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MultipleColonInDictionaryTypeError{}

func (*MultipleColonInDictionaryTypeError) isParseError() {}

func (*MultipleColonInDictionaryTypeError) IsUserError() {}

func (e *MultipleColonInDictionaryTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MultipleColonInDictionaryTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MultipleColonInDictionaryTypeError) Error() string {
	return "unexpected colon (`:`) in dictionary type"
}

func (*MultipleColonInDictionaryTypeError) SecondaryError() string {
	return "dictionary types can only have one colon (`:`) to separate key and value types"
}

func (e *MultipleColonInDictionaryTypeError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove extra colon",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range: ast.Range{
						StartPos: e.Pos,
						EndPos:   e.Pos,
					},
				},
			},
		},
	}
}

func (*MultipleColonInDictionaryTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/dictionaries#dictionary-types"
}

// MissingDictionaryValueTypeError is reported when a dictionary type is missing a value type.
type MissingDictionaryValueTypeError struct {
	Pos ast.Position
}

var _ ParseError = &MissingDictionaryValueTypeError{}
var _ errors.UserError = &MissingDictionaryValueTypeError{}
var _ errors.SecondaryError = &MissingDictionaryValueTypeError{}
var _ errors.HasDocumentationLink = &MissingDictionaryValueTypeError{}

func (*MissingDictionaryValueTypeError) isParseError() {}

func (*MissingDictionaryValueTypeError) IsUserError() {}

func (e *MissingDictionaryValueTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingDictionaryValueTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingDictionaryValueTypeError) Error() string {
	return "missing dictionary value type"
}

func (*MissingDictionaryValueTypeError) SecondaryError() string {
	return "a value type is expected after the colon (`:`)"
}

func (*MissingDictionaryValueTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/dictionaries#dictionary-types"
}

// UnexpectedCommaInTypeAnnotationListError is reported when a comma is found at an unexpected position in a type annotation list.
type UnexpectedCommaInTypeAnnotationListError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedCommaInTypeAnnotationListError{}
var _ errors.UserError = &UnexpectedCommaInTypeAnnotationListError{}
var _ errors.SecondaryError = &UnexpectedCommaInTypeAnnotationListError{}
var _ errors.HasDocumentationLink = &UnexpectedCommaInTypeAnnotationListError{}

func (*UnexpectedCommaInTypeAnnotationListError) isParseError() {}

func (*UnexpectedCommaInTypeAnnotationListError) IsUserError() {}

func (e *UnexpectedCommaInTypeAnnotationListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedCommaInTypeAnnotationListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedCommaInTypeAnnotationListError) Error() string {
	return "unexpected comma (`,`) in type annotation list"
}

func (*UnexpectedCommaInTypeAnnotationListError) SecondaryError() string {
	return "a comma (`,`) is used to separate multiple types, but a type is expected here"
}

func (*UnexpectedCommaInTypeAnnotationListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/type-annotations"
}

// MissingTypeAnnotationAfterCommaError is reported when a type annotation is missing after a comma.
type MissingTypeAnnotationAfterCommaError struct {
	Pos ast.Position
}

var _ ParseError = &MissingTypeAnnotationAfterCommaError{}
var _ errors.UserError = &MissingTypeAnnotationAfterCommaError{}
var _ errors.SecondaryError = &MissingTypeAnnotationAfterCommaError{}
var _ errors.HasDocumentationLink = &MissingTypeAnnotationAfterCommaError{}

func (*MissingTypeAnnotationAfterCommaError) isParseError() {}

func (*MissingTypeAnnotationAfterCommaError) IsUserError() {}

func (e *MissingTypeAnnotationAfterCommaError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingTypeAnnotationAfterCommaError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingTypeAnnotationAfterCommaError) Error() string {
	return "missing type annotation after comma (`,`)"
}

func (*MissingTypeAnnotationAfterCommaError) SecondaryError() string {
	return "after a comma (`,`), a type annotation is required to complete the list"
}

func (*MissingTypeAnnotationAfterCommaError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/type-annotations"
}

// UnexpectedCommaInIntersectionTypeError is reported when a comma is found at an unexpected position in an intersection type.
type UnexpectedCommaInIntersectionTypeError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedCommaInIntersectionTypeError{}
var _ errors.UserError = &UnexpectedCommaInIntersectionTypeError{}
var _ errors.SecondaryError = &UnexpectedCommaInIntersectionTypeError{}
var _ errors.HasDocumentationLink = &UnexpectedCommaInIntersectionTypeError{}

func (*UnexpectedCommaInIntersectionTypeError) isParseError() {}

func (*UnexpectedCommaInIntersectionTypeError) IsUserError() {}

func (e *UnexpectedCommaInIntersectionTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedCommaInIntersectionTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedCommaInIntersectionTypeError) Error() string {
	return "unexpected comma (`,`) in intersection type"
}

func (*UnexpectedCommaInIntersectionTypeError) SecondaryError() string {
	return "intersection types use commas (`,`) to separate multiple types; " +
		"check for missing types or remove the comma"
}

func (*UnexpectedCommaInIntersectionTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
}

// UnexpectedColonInIntersectionTypeError is reported when a colon is found in an intersection type.
type UnexpectedColonInIntersectionTypeError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedColonInIntersectionTypeError{}
var _ errors.UserError = &UnexpectedColonInIntersectionTypeError{}
var _ errors.SecondaryError = &UnexpectedColonInIntersectionTypeError{}
var _ errors.HasDocumentationLink = &UnexpectedColonInIntersectionTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &UnexpectedColonInIntersectionTypeError{}

func (*UnexpectedColonInIntersectionTypeError) isParseError() {}

func (*UnexpectedColonInIntersectionTypeError) IsUserError() {}

func (e *UnexpectedColonInIntersectionTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedColonInIntersectionTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedColonInIntersectionTypeError) Error() string {
	return "unexpected colon (`:`) in intersection type"
}

func (*UnexpectedColonInIntersectionTypeError) SecondaryError() string {
	return "intersection types use commas (`,`) to separate multiple types, not colons (`:`)"
}

func (e *UnexpectedColonInIntersectionTypeError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Replace colon with comma",
			TextEdits: []ast.TextEdit{
				{
					Replacement: ",",
					Range: ast.Range{
						StartPos: e.Pos,
						EndPos:   e.Pos,
					},
				},
			},
		},
	}
}

func (*UnexpectedColonInIntersectionTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
}

// InvalidEntitlementMappingTypeError is reported when an entitlement mapping type is not nominal.
type InvalidEntitlementMappingTypeError struct {
	ast.Range
}

var _ ParseError = &InvalidEntitlementMappingTypeError{}
var _ errors.UserError = &InvalidEntitlementMappingTypeError{}
var _ errors.SecondaryError = &InvalidEntitlementMappingTypeError{}
var _ errors.HasDocumentationLink = &InvalidEntitlementMappingTypeError{}

func (*InvalidEntitlementMappingTypeError) isParseError() {}

func (*InvalidEntitlementMappingTypeError) IsUserError() {}

func (e *InvalidEntitlementMappingTypeError) Error() string {
	return "expected entitlement type"
}

func (*InvalidEntitlementMappingTypeError) SecondaryError() string {
	return "only entitlement types can be used in entitlement mappings"
}

func (*InvalidEntitlementMappingTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
}

// InvalidEntitlementMappingIncludeTypeError is reported when an included entitlement mapping type is not nominal.
type InvalidEntitlementMappingIncludeTypeError struct {
	ast.Range
}

var _ ParseError = &InvalidEntitlementMappingIncludeTypeError{}
var _ errors.UserError = &InvalidEntitlementMappingIncludeTypeError{}
var _ errors.SecondaryError = &InvalidEntitlementMappingIncludeTypeError{}
var _ errors.HasDocumentationLink = &InvalidEntitlementMappingIncludeTypeError{}

func (*InvalidEntitlementMappingIncludeTypeError) isParseError() {}

func (*InvalidEntitlementMappingIncludeTypeError) IsUserError() {}

func (e *InvalidEntitlementMappingIncludeTypeError) Error() string {
	return "expected entitlement mapping type"
}

func (*InvalidEntitlementMappingIncludeTypeError) SecondaryError() string {
	return "only entitlement mapping types can be included in entitlement mappings"
}

func (*InvalidEntitlementMappingIncludeTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#mapping-composition"
}

// MissingRightArrowInEntitlementMappingError is reported when the '->' token is missing in an entitlement mapping.
type MissingRightArrowInEntitlementMappingError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingRightArrowInEntitlementMappingError{}
var _ errors.UserError = &MissingRightArrowInEntitlementMappingError{}
var _ errors.SecondaryError = &MissingRightArrowInEntitlementMappingError{}
var _ errors.HasDocumentationLink = &MissingRightArrowInEntitlementMappingError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingRightArrowInEntitlementMappingError{}

func (*MissingRightArrowInEntitlementMappingError) isParseError() {}

func (*MissingRightArrowInEntitlementMappingError) IsUserError() {}

func (e *MissingRightArrowInEntitlementMappingError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingRightArrowInEntitlementMappingError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingRightArrowInEntitlementMappingError) Error() string {
	return expectedButGotToken(
		"expected right arrow (`->`) in entitlement mapping",
		e.GotToken.Type,
	)
}

func (*MissingRightArrowInEntitlementMappingError) SecondaryError() string {
	return "entitlement mappings must use `->` to separate the input and output types"
}

func (e *MissingRightArrowInEntitlementMappingError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `->`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ` ->`,
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

func (*MissingRightArrowInEntitlementMappingError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
}

// InvalidNonNominalTypeInIntersectionError is reported when a non-nominal type is found in an intersection type.
type InvalidNonNominalTypeInIntersectionError struct {
	ast.Range
}

var _ ParseError = &InvalidNonNominalTypeInIntersectionError{}
var _ errors.UserError = &InvalidNonNominalTypeInIntersectionError{}
var _ errors.SecondaryError = &InvalidNonNominalTypeInIntersectionError{}
var _ errors.HasDocumentationLink = &InvalidNonNominalTypeInIntersectionError{}

func (*InvalidNonNominalTypeInIntersectionError) isParseError() {}

func (*InvalidNonNominalTypeInIntersectionError) IsUserError() {}

func (e *InvalidNonNominalTypeInIntersectionError) Error() string {
	return "non-nominal type in intersection type"
}

func (*InvalidNonNominalTypeInIntersectionError) SecondaryError() string {
	return "intersection types can only contain nominal types (struct, resource, or interface names)"
}

func (*InvalidNonNominalTypeInIntersectionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
}

// MissingTypeAfterCommaInIntersectionError is reported when a type is missing after a comma in an intersection type.
type MissingTypeAfterCommaInIntersectionError struct {
	Pos ast.Position
}

var _ ParseError = &MissingTypeAfterCommaInIntersectionError{}
var _ errors.UserError = &MissingTypeAfterCommaInIntersectionError{}
var _ errors.SecondaryError = &MissingTypeAfterCommaInIntersectionError{}
var _ errors.HasDocumentationLink = &MissingTypeAfterCommaInIntersectionError{}

func (*MissingTypeAfterCommaInIntersectionError) isParseError() {}

func (*MissingTypeAfterCommaInIntersectionError) IsUserError() {}

func (e *MissingTypeAfterCommaInIntersectionError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingTypeAfterCommaInIntersectionError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingTypeAfterCommaInIntersectionError) Error() string {
	return "missing type after comma (`,`)"
}

func (*MissingTypeAfterCommaInIntersectionError) SecondaryError() string {
	return "a type is expected after the comma (`,`)"
}

func (*MissingTypeAfterCommaInIntersectionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
}

// MissingTypeAfterSeparatorError is reported when a type is missing after a separator.
type MissingTypeAfterSeparatorError struct {
	Pos       ast.Position
	Separator lexer.TokenType
}

var _ ParseError = &MissingTypeAfterSeparatorError{}
var _ errors.UserError = &MissingTypeAfterSeparatorError{}
var _ errors.SecondaryError = &MissingTypeAfterSeparatorError{}
var _ errors.HasDocumentationLink = &MissingTypeAfterSeparatorError{}

func (*MissingTypeAfterSeparatorError) isParseError() {}

func (*MissingTypeAfterSeparatorError) IsUserError() {}

func (e *MissingTypeAfterSeparatorError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingTypeAfterSeparatorError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *MissingTypeAfterSeparatorError) Error() string {
	return fmt.Sprintf("missing type after separator %#q", e.Separator)
}

func (e *MissingTypeAfterSeparatorError) SecondaryError() string {
	return fmt.Sprintf("a type is expected after the %#q separator", e.Separator)
}

func (*MissingTypeAfterSeparatorError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// MissingSeparatorInIntersectionOrDictionaryTypeError is reported when a separator is missing
// between types in an intersection or dictionary type.
type MissingSeparatorInIntersectionOrDictionaryTypeError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingSeparatorInIntersectionOrDictionaryTypeError{}
var _ errors.UserError = &MissingSeparatorInIntersectionOrDictionaryTypeError{}
var _ errors.SecondaryError = &MissingSeparatorInIntersectionOrDictionaryTypeError{}
var _ errors.HasDocumentationLink = &MissingSeparatorInIntersectionOrDictionaryTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingSeparatorInIntersectionOrDictionaryTypeError{}

func (*MissingSeparatorInIntersectionOrDictionaryTypeError) isParseError() {}

func (*MissingSeparatorInIntersectionOrDictionaryTypeError) IsUserError() {}

func (e *MissingSeparatorInIntersectionOrDictionaryTypeError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingSeparatorInIntersectionOrDictionaryTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingSeparatorInIntersectionOrDictionaryTypeError) Error() string {
	return expectedButGotToken(
		"missing separator (`,`) in type list",
		e.GotToken.Type,
	)
}

func (*MissingSeparatorInIntersectionOrDictionaryTypeError) SecondaryError() string {
	return "types in an intersection type must be separated by a comma (`,`) " +
		"and types in a dictionary type must be separated by a colon (`:`)"
}

func (e *MissingSeparatorInIntersectionOrDictionaryTypeError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	r := newLeftAttachedRange(e.GotToken.StartPos, code)
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert comma",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ",",
					Range:     r,
				},
			},
		},
		{
			Message: "Insert colon",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ":",
					Range:     r,
				},
			},
		},
	}
}

func (*MissingSeparatorInIntersectionOrDictionaryTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
}

// ExpectedTypeInsteadSeparatorError is reported when a separator is found at an unexpected position,
// where a type was expected.
type ExpectedTypeInsteadSeparatorError struct {
	Pos       ast.Position
	Separator lexer.TokenType
}

var _ ParseError = &ExpectedTypeInsteadSeparatorError{}
var _ errors.UserError = &ExpectedTypeInsteadSeparatorError{}
var _ errors.SecondaryError = &ExpectedTypeInsteadSeparatorError{}
var _ errors.HasDocumentationLink = &ExpectedTypeInsteadSeparatorError{}

func (*ExpectedTypeInsteadSeparatorError) isParseError() {}

func (*ExpectedTypeInsteadSeparatorError) IsUserError() {}

func (e *ExpectedTypeInsteadSeparatorError) StartPosition() ast.Position {
	return e.Pos
}

func (e *ExpectedTypeInsteadSeparatorError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *ExpectedTypeInsteadSeparatorError) Error() string {
	return fmt.Sprintf("expected type, got separator %#q", e.Separator)
}

func (e *ExpectedTypeInsteadSeparatorError) SecondaryError() string {
	return "a type was expected, but a separator was found instead; remove the separator"
}

func (*ExpectedTypeInsteadSeparatorError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// UnexpectedTokenInsteadOfSeparatorError is reported when an unexpected token is found,
// where a separator or an end token was expected.
type UnexpectedTokenInsteadOfSeparatorError struct {
	GotToken          lexer.Token
	ExpectedSeparator lexer.TokenType
	ExpectedEndToken  lexer.TokenType
}

var _ ParseError = &UnexpectedTokenInsteadOfSeparatorError{}
var _ errors.UserError = &UnexpectedTokenInsteadOfSeparatorError{}
var _ errors.SecondaryError = &UnexpectedTokenInsteadOfSeparatorError{}
var _ errors.HasDocumentationLink = &UnexpectedTokenInsteadOfSeparatorError{}

func (*UnexpectedTokenInsteadOfSeparatorError) isParseError() {}

func (*UnexpectedTokenInsteadOfSeparatorError) IsUserError() {}

func (e *UnexpectedTokenInsteadOfSeparatorError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *UnexpectedTokenInsteadOfSeparatorError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *UnexpectedTokenInsteadOfSeparatorError) Error() string {
	return expectedButGotToken(
		fmt.Sprintf(
			"expected %#q or separator %#q",
			e.ExpectedEndToken,
			e.ExpectedSeparator,
		),
		e.GotToken.Type,
	)
}

func (e *UnexpectedTokenInsteadOfSeparatorError) SecondaryError() string {
	return fmt.Sprintf(
		"did you miss a separator (%#q) or an end (%#q)?",
		e.ExpectedSeparator,
		e.ExpectedEndToken,
	)
}

func (*UnexpectedTokenInsteadOfSeparatorError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// MissingClosingParenInArgumentListError is reported when an argument list is missing a closing parenthesis.
type MissingClosingParenInArgumentListError struct {
	Pos ast.Position
}

var _ ParseError = &MissingClosingParenInArgumentListError{}
var _ errors.UserError = &MissingClosingParenInArgumentListError{}
var _ errors.SecondaryError = &MissingClosingParenInArgumentListError{}
var _ errors.HasDocumentationLink = &MissingClosingParenInArgumentListError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingParenInArgumentListError{}

func (*MissingClosingParenInArgumentListError) isParseError() {}

func (*MissingClosingParenInArgumentListError) IsUserError() {}

func (e *MissingClosingParenInArgumentListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingClosingParenInArgumentListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingClosingParenInArgumentListError) Error() string {
	return "missing closing parenthesis (`)`) at end of invocation argument list"
}

func (*MissingClosingParenInArgumentListError) SecondaryError() string {
	return "function calls and type instantiations must be properly closed with a closing parenthesis (`)`)"
}

func (*MissingClosingParenInArgumentListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *MissingClosingParenInArgumentListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ")",
					Range:     newLeftAttachedRange(e.Pos, code),
				},
			},
		},
	}
}

// UnexpectedCommaInArgumentListError is reported when a comma is found at an unexpected position in an argument list.
type UnexpectedCommaInArgumentListError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedCommaInArgumentListError{}
var _ errors.UserError = &UnexpectedCommaInArgumentListError{}
var _ errors.SecondaryError = &UnexpectedCommaInArgumentListError{}
var _ errors.HasDocumentationLink = &UnexpectedCommaInArgumentListError{}

func (*UnexpectedCommaInArgumentListError) isParseError() {}

func (*UnexpectedCommaInArgumentListError) IsUserError() {}

func (e *UnexpectedCommaInArgumentListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedCommaInArgumentListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedCommaInArgumentListError) Error() string {
	return "unexpected comma (`,`) in argument list"
}

func (*UnexpectedCommaInArgumentListError) SecondaryError() string {
	return "commas (`,`) are used to separate arguments; " +
		"did you add a superfluous comma, or is an argument missing"
}

func (*UnexpectedCommaInArgumentListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// MissingCommaInArgumentListError is reported when an argument is found,
// but a comma or the end of the argument list is expected.
type MissingCommaInArgumentListError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingCommaInArgumentListError{}
var _ errors.UserError = &MissingCommaInArgumentListError{}
var _ errors.SecondaryError = &MissingCommaInArgumentListError{}
var _ errors.HasDocumentationLink = &MissingCommaInArgumentListError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingCommaInArgumentListError{}

func (*MissingCommaInArgumentListError) isParseError() {}

func (*MissingCommaInArgumentListError) IsUserError() {}

func (e *MissingCommaInArgumentListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingCommaInArgumentListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingCommaInArgumentListError) Error() string {
	return expectedButGotToken(
		"unexpected argument in argument list: expected delimiter (`,`) or end of argument list (`)`)",
		e.GotToken.Type,
	)
}

func (*MissingCommaInArgumentListError) SecondaryError() string {
	return "arguments in function calls must be separated by commas (`,`)"
}

func (*MissingCommaInArgumentListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *MissingCommaInArgumentListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert comma",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ",",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// InvalidExpressionAsLabelError is reported when an argument label is not a simple identifier.
type InvalidExpressionAsLabelError struct {
	ast.Range
}

var _ ParseError = &InvalidExpressionAsLabelError{}
var _ errors.UserError = &InvalidExpressionAsLabelError{}
var _ errors.SecondaryError = &InvalidExpressionAsLabelError{}
var _ errors.HasDocumentationLink = &InvalidExpressionAsLabelError{}

func (*InvalidExpressionAsLabelError) isParseError() {}

func (*InvalidExpressionAsLabelError) IsUserError() {}

func (e *InvalidExpressionAsLabelError) Error() string {
	return "expected identifier for label"
}

func (*InvalidExpressionAsLabelError) SecondaryError() string {
	return "argument labels must be simple identifiers, not expressions or complex syntax"
}

func (*InvalidExpressionAsLabelError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// UnexpectedExpressionStartError is reported when an unexpected token is found at the start of an expression.
type UnexpectedExpressionStartError struct {
	GotToken lexer.Token
}

var _ ParseError = &UnexpectedExpressionStartError{}
var _ errors.UserError = &UnexpectedExpressionStartError{}
var _ errors.SecondaryError = &UnexpectedExpressionStartError{}
var _ errors.HasDocumentationLink = &UnexpectedExpressionStartError{}

func (*UnexpectedExpressionStartError) isParseError() {}

func (*UnexpectedExpressionStartError) IsUserError() {}

func (e *UnexpectedExpressionStartError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *UnexpectedExpressionStartError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *UnexpectedExpressionStartError) Error() string {
	return fmt.Sprintf(
		"unexpected token at start of expression: %s",
		e.GotToken.Type,
	)
}

func (*UnexpectedExpressionStartError) SecondaryError() string {
	return "this token cannot be used to start an expression; " +
		"check for missing operators, parentheses, or invalid syntax"
}

func (*UnexpectedExpressionStartError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// UnexpectedTokenInExpressionError is reported when an unexpected token is found in an expression.
type UnexpectedTokenInExpressionError struct {
	GotToken lexer.Token
}

var _ ParseError = &UnexpectedTokenInExpressionError{}
var _ errors.UserError = &UnexpectedTokenInExpressionError{}
var _ errors.SecondaryError = &UnexpectedTokenInExpressionError{}
var _ errors.HasDocumentationLink = &UnexpectedTokenInExpressionError{}

func (*UnexpectedTokenInExpressionError) isParseError() {}

func (*UnexpectedTokenInExpressionError) IsUserError() {}

func (e *UnexpectedTokenInExpressionError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *UnexpectedTokenInExpressionError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *UnexpectedTokenInExpressionError) Error() string {
	return fmt.Sprintf(
		"unexpected token in expression: %s",
		e.GotToken.Type,
	)
}

func (*UnexpectedTokenInExpressionError) SecondaryError() string {
	return "this token cannot be used as an operator in an expression; " +
		"check for missing operators, parentheses, or invalid syntax"
}

func (*UnexpectedTokenInExpressionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// UnexpectedTypeStartError is reported when an unexpected token is found at the start of a type.
type UnexpectedTypeStartError struct {
	GotToken lexer.Token
}

var _ ParseError = &UnexpectedTypeStartError{}
var _ errors.UserError = &UnexpectedTypeStartError{}
var _ errors.SecondaryError = &UnexpectedTypeStartError{}
var _ errors.HasDocumentationLink = &UnexpectedTypeStartError{}

func (*UnexpectedTypeStartError) isParseError() {}

func (*UnexpectedTypeStartError) IsUserError() {}

func (e *UnexpectedTypeStartError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *UnexpectedTypeStartError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *UnexpectedTypeStartError) Error() string {
	return fmt.Sprintf(
		"unexpected token at start of type: %s",
		e.GotToken.Type,
	)
}

func (*UnexpectedTypeStartError) SecondaryError() string {
	return "this token cannot be used to start a type; " +
		"check for missing operators, parentheses, or invalid syntax"
}

func (*UnexpectedTypeStartError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// UnexpectedTokenInTypeError is reported when an unexpected token is found in a type.
type UnexpectedTokenInTypeError struct {
	GotToken lexer.Token
}

var _ ParseError = &UnexpectedTokenInTypeError{}
var _ errors.UserError = &UnexpectedTokenInTypeError{}
var _ errors.SecondaryError = &UnexpectedTokenInTypeError{}
var _ errors.HasDocumentationLink = &UnexpectedTokenInTypeError{}

func (*UnexpectedTokenInTypeError) isParseError() {}

func (*UnexpectedTokenInTypeError) IsUserError() {}

func (e *UnexpectedTokenInTypeError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *UnexpectedTokenInTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *UnexpectedTokenInTypeError) Error() string {
	return fmt.Sprintf(
		"unexpected token in type: %s",
		e.GotToken.Type,
	)
}

func (*UnexpectedTokenInTypeError) SecondaryError() string {
	return "this token cannot be used as an operator in a type; " +
		"check for missing operators, parentheses, or invalid syntax"
}

func (*UnexpectedTokenInTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// InvalidConstantSizedTypeSizeError

type InvalidConstantSizedTypeSizeError struct {
	ast.Range
}

var _ ParseError = &InvalidConstantSizedTypeSizeError{}
var _ errors.UserError = &InvalidConstantSizedTypeSizeError{}
var _ errors.SecondaryError = &InvalidConstantSizedTypeSizeError{}
var _ errors.HasDocumentationLink = &InvalidConstantSizedTypeSizeError{}

func (*InvalidConstantSizedTypeSizeError) isParseError() {}

func (*InvalidConstantSizedTypeSizeError) IsUserError() {}

func (*InvalidConstantSizedTypeSizeError) Error() string {
	return "expected positive integer size for constant sized type"
}

func (*InvalidConstantSizedTypeSizeError) SecondaryError() string {
	return "the size of a constant-sized array must be a positive integer literal"
}

func (*InvalidConstantSizedTypeSizeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/arrays#array-types"
}

// CustomDestructorError

type CustomDestructorError struct {
	Pos             ast.Position
	DestructorRange ast.Range
}

var _ ParseError = &CustomDestructorError{}
var _ errors.UserError = &CustomDestructorError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &CustomDestructorError{}
var _ errors.HasDocumentationLink = &CustomDestructorError{}
var _ errors.HasMigrationNote = &CustomDestructorError{}

func (*CustomDestructorError) isParseError() {}

func (*CustomDestructorError) IsUserError() {}

func (e *CustomDestructorError) StartPosition() ast.Position {
	return e.Pos
}

func (e *CustomDestructorError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*CustomDestructorError) Error() string {
	return "custom destructor definitions are no longer permitted since Cadence v1.0"
}

func (*CustomDestructorError) SecondaryError() string {
	return "remove the destructor definition"
}

func (*CustomDestructorError) MigrationNote() string {
	return "This is pre-Cadence 1.0 syntax. Support for custom destructors was removed. " +
		"Any custom cleanup logic should be moved to a separate function, " +
		"and must be explicitly called before the destruction."
}

func (e *CustomDestructorError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove the deprecated custom destructor",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       e.DestructorRange,
				},
			},
		},
	}
}

func (*CustomDestructorError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/cadence-migration-guide/improvements#-motivation-23"
}

// RestrictedTypeError

type RestrictedTypeError struct {
	ast.Range
}

var _ ParseError = &RestrictedTypeError{}
var _ errors.UserError = &RestrictedTypeError{}
var _ errors.SecondaryError = &RestrictedTypeError{}
var _ errors.HasDocumentationLink = &RestrictedTypeError{}
var _ errors.HasMigrationNote = &RestrictedTypeError{}

func (*RestrictedTypeError) isParseError() {}

func (*RestrictedTypeError) IsUserError() {}

func (*RestrictedTypeError) Error() string {
	return "restricted types have been removed in Cadence 1.0+"
}

func (*RestrictedTypeError) SecondaryError() string {
	return "replace with the concrete type or an equivalent intersection type"
}

func (*RestrictedTypeError) MigrationNote() string {
	return "This is pre-Cadence 1.0 syntax. " +
		"Restricted types like `T{}` have been replaced with intersection types like `{T}`."
}

func (*RestrictedTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/cadence-migration-guide/improvements#-motivation-12"
}

// InvalidAccessModifierError

type InvalidAccessModifierError struct {
	Pos             ast.Position
	DeclarationKind common.DeclarationKind
}

var _ ParseError = &InvalidAccessModifierError{}
var _ errors.UserError = &InvalidAccessModifierError{}
var _ errors.SecondaryError = &InvalidAccessModifierError{}
var _ errors.HasDocumentationLink = &InvalidAccessModifierError{}

func (*InvalidAccessModifierError) isParseError() {}

func (*InvalidAccessModifierError) IsUserError() {}

func (e *InvalidAccessModifierError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidAccessModifierError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *InvalidAccessModifierError) Error() string {
	return fmt.Sprintf(
		"invalid access modifier for %s",
		e.DeclarationKind.Name(),
	)
}

func (e *InvalidAccessModifierError) SecondaryError() string {
	return fmt.Sprintf(
		"access modifiers are not allowed on %s declarations",
		e.DeclarationKind.Name(),
	)
}

func (*InvalidAccessModifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

// InvalidViewModifierError

type InvalidViewModifierError struct {
	Pos             ast.Position
	DeclarationKind common.DeclarationKind
}

var _ ParseError = &InvalidViewModifierError{}
var _ errors.UserError = &InvalidViewModifierError{}
var _ errors.SecondaryError = &InvalidViewModifierError{}
var _ errors.HasDocumentationLink = &InvalidViewModifierError{}

func (*InvalidViewModifierError) isParseError() {}

func (*InvalidViewModifierError) IsUserError() {}

func (e *InvalidViewModifierError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidViewModifierError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *InvalidViewModifierError) Error() string {
	return fmt.Sprintf("invalid `view` modifier for %s", e.DeclarationKind.Name())
}

func (*InvalidViewModifierError) SecondaryError() string {
	return "the `view` modifier can only be used on functions"
}

func (*InvalidViewModifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions#view-functions"
}

// DuplicateViewModifierError

type DuplicateViewModifierError struct {
	ast.Range
}

var _ ParseError = &DuplicateViewModifierError{}
var _ errors.UserError = &DuplicateViewModifierError{}
var _ errors.SecondaryError = &DuplicateViewModifierError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &DuplicateViewModifierError{}
var _ errors.HasDocumentationLink = &DuplicateViewModifierError{}

func (*DuplicateViewModifierError) isParseError() {}

func (*DuplicateViewModifierError) IsUserError() {}

func (*DuplicateViewModifierError) Error() string {
	return "invalid second `view` modifier"
}

func (*DuplicateViewModifierError) SecondaryError() string {
	return "the `view` modifier can only be used once per function declaration"
}

func (e *DuplicateViewModifierError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove duplicate `view` modifier",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       e.Range,
				},
			},
		},
	}
}

func (*DuplicateViewModifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions#view-functions"
}

// DuplicateAccessModifierError

type DuplicateAccessModifierError struct {
	ast.Range
}

var _ ParseError = &DuplicateAccessModifierError{}
var _ errors.UserError = &DuplicateAccessModifierError{}
var _ errors.SecondaryError = &DuplicateAccessModifierError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &DuplicateAccessModifierError{}
var _ errors.HasDocumentationLink = &DuplicateAccessModifierError{}

func (*DuplicateAccessModifierError) isParseError() {}

func (*DuplicateAccessModifierError) IsUserError() {}

func (*DuplicateAccessModifierError) Error() string {
	return "invalid second access modifier"
}

func (*DuplicateAccessModifierError) SecondaryError() string {
	return "only one access modifier can be used per declaration"
}

func (e *DuplicateAccessModifierError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove duplicate access modifier",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       e.Range,
				},
			},
		},
	}
}

func (*DuplicateAccessModifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

// PrivAccessError

type PrivAccessError struct {
	ast.Range
}

var _ ParseError = &PrivAccessError{}
var _ errors.UserError = &PrivAccessError{}
var _ errors.SecondaryError = &PrivAccessError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &PrivAccessError{}
var _ errors.HasMigrationNote = &PrivAccessError{}
var _ errors.HasDocumentationLink = &PrivAccessError{}

func (*PrivAccessError) isParseError() {}

func (*PrivAccessError) IsUserError() {}

func (*PrivAccessError) Error() string {
	return "`priv` is no longer a valid access modifier"
}

func (*PrivAccessError) SecondaryError() string {
	return "use `access(self)` instead"
}

func (e *PrivAccessError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Replace with `access(self)`",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "access(self)",
					Range:       e.Range,
				},
			},
		},
	}
}

func (*PrivAccessError) MigrationNote() string {
	return "This is pre-Cadence 1.0 syntax. The `priv` modifier was replaced with `access(self)`"
}

func (*PrivAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

// PubAccessError

type PubAccessError struct {
	ast.Range
}

var _ ParseError = &PubAccessError{}
var _ errors.UserError = &PubAccessError{}
var _ errors.SecondaryError = &PubAccessError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &PubAccessError{}
var _ errors.HasMigrationNote = &PubAccessError{}
var _ errors.HasDocumentationLink = &PubAccessError{}

func (*PubAccessError) isParseError() {}

func (*PubAccessError) IsUserError() {}

func (*PubAccessError) Error() string {
	return "`pub` is no longer a valid access modifier"
}

func (*PubAccessError) SecondaryError() string {
	return "use `access(all)` instead"
}

func (e *PubAccessError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Replace with `access(all)`",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "access(all)",
					Range:       e.Range,
				},
			},
		},
	}
}

func (*PubAccessError) MigrationNote() string {
	return "This is pre-Cadence 1.0 syntax. The `pub` modifier was replaced with `access(all)`"
}

func (*PubAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

// PubSetAccessError is reported when the `pub(set)` access modifier is used.
// This modifier is invalid in Cadence 1.0.
type PubSetAccessError struct {
	ast.Range
}

var _ ParseError = &PubSetAccessError{}
var _ errors.UserError = &PubSetAccessError{}
var _ errors.HasMigrationNote = &PubSetAccessError{}
var _ errors.HasDocumentationLink = &PubSetAccessError{}

func (*PubSetAccessError) isParseError() {}

func (*PubSetAccessError) IsUserError() {}

func (e *PubSetAccessError) Error() string {
	return "`pub(set)` is no longer a valid access modifier"
}

func (*PubSetAccessError) MigrationNote() string {
	return "This is pre-Cadence 1.0 syntax. " +
		"The `pub(set)` modifier was deprecated and has no direct equivalent in the new access control system. " +
		"Consider adding a setter method that allows updating the field."
}

func (*PubSetAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/cadence-migration-guide/improvements#-motivation-11"
}

// MissingAccessOpeningParenError is reported when an access modifier is missing an opening parenthesis.
type MissingAccessOpeningParenError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingAccessOpeningParenError{}
var _ errors.UserError = &MissingAccessOpeningParenError{}
var _ errors.SecondaryError = &MissingAccessOpeningParenError{}
var _ errors.HasDocumentationLink = &MissingAccessOpeningParenError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingAccessOpeningParenError{}

func (*MissingAccessOpeningParenError) isParseError() {}

func (*MissingAccessOpeningParenError) IsUserError() {}

func (e *MissingAccessOpeningParenError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingAccessOpeningParenError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingAccessOpeningParenError) Error() string {
	return expectedButGotToken(
		"expected opening parenthesis (`(`) after `access` keyword",
		e.GotToken.Type,
	)
}

func (*MissingAccessOpeningParenError) SecondaryError() string {
	return "access modifiers must be enclosed in parentheses (`(...)`), for example `access(all)`"
}

func (*MissingAccessOpeningParenError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

func (e *MissingAccessOpeningParenError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	if e.GotToken.Is(lexer.TokenIdentifier) {
		tokenSource := code[e.GotToken.StartPos.Offset : e.GotToken.EndPos.Offset+1]
		return []errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Enclose in parentheses",
				TextEdits: []ast.TextEdit{
					{
						Replacement: fmt.Sprintf("(%s)", tokenSource),
						Range:       e.GotToken.Range,
					},
				},
			},
		}
	} else {
		return []errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Insert opening parenthesis",
				TextEdits: []ast.TextEdit{
					{
						Insertion: "(",
						Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
					},
				},
			},
		}
	}
}

// MissingAccessClosingParenError is reported when an access modifier is missing a closing parenthesis.
type MissingAccessClosingParenError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingAccessClosingParenError{}
var _ errors.UserError = &MissingAccessClosingParenError{}
var _ errors.SecondaryError = &MissingAccessClosingParenError{}
var _ errors.HasDocumentationLink = &MissingAccessClosingParenError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingAccessClosingParenError{}

func (*MissingAccessClosingParenError) isParseError() {}

func (*MissingAccessClosingParenError) IsUserError() {}

func (e *MissingAccessClosingParenError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingAccessClosingParenError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingAccessClosingParenError) Error() string {
	return expectedButGotToken(
		"expected closing parenthesis (`)`) at end of `access` modifier",
		e.GotToken.Type,
	)
}

func (*MissingAccessClosingParenError) SecondaryError() string {
	return "the `access` modifier must be properly closed with a closing parenthesis (`)`)"
}

func (*MissingAccessClosingParenError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

func (e *MissingAccessClosingParenError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ")",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingAccessKeywordError is reported when an access modifier keyword is missing.
type MissingAccessKeywordError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingAccessKeywordError{}
var _ errors.UserError = &MissingAccessKeywordError{}
var _ errors.SecondaryError = &MissingAccessKeywordError{}
var _ errors.HasDocumentationLink = &MissingAccessKeywordError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingAccessKeywordError{}

func (*MissingAccessKeywordError) isParseError() {}

func (*MissingAccessKeywordError) IsUserError() {}

func (e *MissingAccessKeywordError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingAccessKeywordError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

var (
	accessModifierKeywords = []string{
		KeywordAll,
		KeywordAccount,
		KeywordContract,
		KeywordSelf,
	}
	enumeratedAccessModifierKeywords = common.EnumerateWords(
		[]string{
			fmt.Sprintf("%#q", KeywordAll),
			fmt.Sprintf("%#q", KeywordAccount),
			fmt.Sprintf("%#q", KeywordContract),
			fmt.Sprintf("%#q", KeywordSelf),
		},
		"or",
	)
)

func (e *MissingAccessKeywordError) Error() string {
	return expectedButGotToken(
		fmt.Sprintf(
			"expected keyword %s",
			enumeratedAccessModifierKeywords,
		),
		e.GotToken.Type,
	)
}

func (e *MissingAccessKeywordError) SecondaryError() string {
	if e.GotToken.Is(lexer.TokenIdentifier) {
		return "replace with one of the access modifier keywords"
	} else {
		return "add one of the access modifier keywords"
	}
}

func (*MissingAccessKeywordError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

func (e *MissingAccessKeywordError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	var fixes []errors.SuggestedFix[ast.TextEdit]

	for _, keyword := range accessModifierKeywords {

		fixes = append(
			fixes,
			errors.SuggestedFix[ast.TextEdit]{
				Message: fmt.Sprintf("Insert %#q", keyword),
				TextEdits: []ast.TextEdit{
					{
						Insertion: keyword,
						Range: ast.Range{
							StartPos: e.GotToken.StartPos,
							EndPos:   e.GotToken.StartPos,
						},
					},
				},
			},
		)
	}

	return fixes
}

// MissingEnumCaseNameError is reported when an enum case is missing a name.
type MissingEnumCaseNameError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingEnumCaseNameError{}
var _ errors.UserError = &MissingEnumCaseNameError{}
var _ errors.SecondaryError = &MissingEnumCaseNameError{}
var _ errors.HasDocumentationLink = &MissingEnumCaseNameError{}

func (*MissingEnumCaseNameError) isParseError() {}

func (*MissingEnumCaseNameError) IsUserError() {}

func (e *MissingEnumCaseNameError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingEnumCaseNameError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingEnumCaseNameError) Error() string {
	return expectedButGotToken(
		"expected identifier after start of enum case declaration",
		e.GotToken.Type,
	)
}

func (*MissingEnumCaseNameError) SecondaryError() string {
	return "provide a name for the enum case after the `case` keyword"
}

func (*MissingEnumCaseNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/enumerations"
}

// MissingFieldNameError is reported when a field is missing a name.
type MissingFieldNameError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingFieldNameError{}
var _ errors.UserError = &MissingFieldNameError{}
var _ errors.SecondaryError = &MissingFieldNameError{}
var _ errors.HasDocumentationLink = &MissingFieldNameError{}

func (*MissingFieldNameError) isParseError() {}

func (*MissingFieldNameError) IsUserError() {}

func (e *MissingFieldNameError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingFieldNameError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingFieldNameError) Error() string {
	return expectedButGotToken(
		"expected identifier after start of field declaration",
		e.GotToken.Type,
	)
}

func (*MissingFieldNameError) SecondaryError() string {
	return "field declarations must have a valid identifier name after the variable kind keyword (`let`/`var`)"
}

func (*MissingFieldNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types#composite-type-fields"
}

// MissingColonAfterFieldNameError is reported when a colon is missing after a field name.
type MissingColonAfterFieldNameError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingColonAfterFieldNameError{}
var _ errors.UserError = &MissingColonAfterFieldNameError{}
var _ errors.SecondaryError = &MissingColonAfterFieldNameError{}
var _ errors.HasDocumentationLink = &MissingColonAfterFieldNameError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingColonAfterFieldNameError{}

func (*MissingColonAfterFieldNameError) isParseError() {}

func (*MissingColonAfterFieldNameError) IsUserError() {}

func (e *MissingColonAfterFieldNameError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingColonAfterFieldNameError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingColonAfterFieldNameError) Error() string {
	return expectedButGotToken(
		"expected colon (`:`) after field name",
		e.GotToken.Type,
	)
}

func (*MissingColonAfterFieldNameError) SecondaryError() string {
	return "field declarations must have a type annotation separated by a colon (`:`)"
}

func (*MissingColonAfterFieldNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types#composite-type-fields"
}

func (e *MissingColonAfterFieldNameError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert colon",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ":",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

type FieldInitializationError struct {
	ast.Range
}

var _ ParseError = &FieldInitializationError{}
var _ errors.UserError = &FieldInitializationError{}
var _ errors.SecondaryError = &FieldInitializationError{}
var _ errors.HasDocumentationLink = &FieldInitializationError{}

func (*FieldInitializationError) isParseError() {}

func (*FieldInitializationError) IsUserError() {}

func (*FieldInitializationError) Error() string {
	return "field declarations cannot have initial values"
}

func (*FieldInitializationError) SecondaryError() string {
	return "field declarations in composite types (structs, resources, contracts) cannot be initialized inline; " +
		"use an initializer (`init`) function instead"
}

func (*FieldInitializationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types#composite-type-fields"
}

// MissingTransferError is reported when a transfer is missing in a variable declaration.
type MissingTransferError struct {
	Pos ast.Position
}

var _ ParseError = &MissingTransferError{}
var _ errors.UserError = &MissingTransferError{}
var _ errors.SecondaryError = &MissingTransferError{}
var _ errors.HasDocumentationLink = &MissingTransferError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingTransferError{}

func (*MissingTransferError) isParseError() {}

func (*MissingTransferError) IsUserError() {}

func (e *MissingTransferError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingTransferError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingTransferError) Error() string {
	return "missing transfer operator"
}

func (*MissingTransferError) SecondaryError() string {
	return "variable declarations must specify how to transfer the value; " +
		"use `=` for copy (struct), `<-` for move (resource), or `<-!` for forced move (resource)"
}

func (*MissingTransferError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/constants-and-variables"
}

func (e *MissingTransferError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	r := newLeftAttachedRange(e.Pos, code)
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `=` (for struct)",
			TextEdits: []ast.TextEdit{
				{
					Insertion: " =",
					Range:     r,
				},
			},
		},
		{
			Message: "Insert `<-` (for resource)",
			TextEdits: []ast.TextEdit{
				{
					Insertion: " <-",
					Range:     r,
				},
			},
		},
	}
}

// InvalidImportLocationError is reported when an import declaration has an invalid location.
type InvalidImportLocationError struct {
	GotToken lexer.Token
}

var _ ParseError = &InvalidImportLocationError{}
var _ errors.UserError = &InvalidImportLocationError{}
var _ errors.SecondaryError = &InvalidImportLocationError{}
var _ errors.HasDocumentationLink = &InvalidImportLocationError{}

func (*InvalidImportLocationError) isParseError() {}

func (*InvalidImportLocationError) IsUserError() {}

func (e *InvalidImportLocationError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *InvalidImportLocationError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *InvalidImportLocationError) Error() string {
	return expectedButGotToken(
		"unexpected token in import declaration: expected address, string, or identifier",
		e.GotToken.Type,
	)
}

func (*InvalidImportLocationError) SecondaryError() string {
	return "import declarations must start with a hexadecimal address, string literal, or identifier"
}

func (*InvalidImportLocationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}

// InvalidImportContinuationError is reported when an import declaration
// has an invalid token after an identifier.
type InvalidImportContinuationError struct {
	GotToken lexer.Token
}

var _ ParseError = &InvalidImportContinuationError{}
var _ errors.UserError = &InvalidImportContinuationError{}
var _ errors.SecondaryError = &InvalidImportContinuationError{}
var _ errors.HasDocumentationLink = &InvalidImportContinuationError{}

func (*InvalidImportContinuationError) isParseError() {}

func (*InvalidImportContinuationError) IsUserError() {}

func (e *InvalidImportContinuationError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *InvalidImportContinuationError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *InvalidImportContinuationError) Error() string {
	return expectedButGotToken(
		"unexpected token in import declaration: expected keyword `from` or comma (`,`)",
		e.GotToken.Type,
	)
}

func (*InvalidImportContinuationError) SecondaryError() string {
	return "after an imported identifier, use either a comma (`,`) to import more items " +
		"or the `from` keyword to specify the import location"
}

func (*InvalidImportContinuationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}

// MissingImportLocationError is reported when an import declaration is missing a location.
type MissingImportLocationError struct {
	Pos ast.Position
}

var _ ParseError = &MissingImportLocationError{}
var _ errors.UserError = &MissingImportLocationError{}
var _ errors.SecondaryError = &MissingImportLocationError{}
var _ errors.HasDocumentationLink = &MissingImportLocationError{}

func (*MissingImportLocationError) isParseError() {}

func (*MissingImportLocationError) IsUserError() {}

func (e *MissingImportLocationError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingImportLocationError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingImportLocationError) Error() string {
	return "unexpected end in import declaration: expected string, address, or identifier"
}

func (*MissingImportLocationError) SecondaryError() string {
	return "import declarations must specify what to import; provide a string literal, hexadecimal address, or identifier"
}

func (*MissingImportLocationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}

// UnexpectedEOFInImportListError is reported when an import list ends unexpectedly.
type UnexpectedEOFInImportListError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedEOFInImportListError{}
var _ errors.UserError = &UnexpectedEOFInImportListError{}
var _ errors.SecondaryError = &UnexpectedEOFInImportListError{}
var _ errors.HasDocumentationLink = &UnexpectedEOFInImportListError{}

func (*UnexpectedEOFInImportListError) isParseError() {}

func (*UnexpectedEOFInImportListError) IsUserError() {}

func (e *UnexpectedEOFInImportListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedEOFInImportListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedEOFInImportListError) Error() string {
	return "unexpected end in import declaration: expected identifier or comma (`,`)"
}

func (*UnexpectedEOFInImportListError) SecondaryError() string {
	return "import declarations cannot end abruptly; " +
		"add either an identifier to import or a comma (`,`) " +
		"to continue the import list"
}

func (*UnexpectedEOFInImportListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}

// InvalidTokenInImportAliasError is reported when an import alias has an invalid token.
type InvalidTokenInImportAliasError struct {
	GotToken lexer.Token
}

var _ ParseError = &InvalidTokenInImportAliasError{}
var _ errors.UserError = &InvalidTokenInImportAliasError{}
var _ errors.SecondaryError = &InvalidTokenInImportAliasError{}
var _ errors.HasDocumentationLink = &InvalidTokenInImportAliasError{}

func (*InvalidTokenInImportAliasError) isParseError() {}

func (*InvalidTokenInImportAliasError) IsUserError() {}

func (e *InvalidTokenInImportAliasError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *InvalidTokenInImportAliasError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *InvalidTokenInImportAliasError) Error() string {
	return expectedButGotToken(
		"expected identifier",
		e.GotToken.Type,
	)
}

func (*InvalidTokenInImportAliasError) SecondaryError() string {
	return "import declarations expect an identifier after the keyword `as`"
}

func (*InvalidTokenInImportAliasError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}

// InvalidTokenInImportListError is reported when an import list has an invalid token.
type InvalidTokenInImportListError struct {
	GotToken lexer.Token
}

var _ ParseError = &InvalidTokenInImportListError{}
var _ errors.UserError = &InvalidTokenInImportListError{}
var _ errors.SecondaryError = &InvalidTokenInImportListError{}
var _ errors.HasDocumentationLink = &InvalidTokenInImportListError{}

func (*InvalidTokenInImportListError) isParseError() {}

func (*InvalidTokenInImportListError) IsUserError() {}

func (e *InvalidTokenInImportListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *InvalidTokenInImportListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *InvalidTokenInImportListError) Error() string {
	return expectedButGotToken(
		"expected identifier or keyword `from`",
		e.GotToken.Type,
	)
}

func (*InvalidTokenInImportListError) SecondaryError() string {
	return "import declarations expect either an identifier to import, " +
		"or the `from` keyword to specify the import location"
}

func (*InvalidTokenInImportListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}

// InvalidFromKeywordAsIdentifierError is reported when the `from` keyword is used as an identifier
// in an invalid context in an import declaration.
type InvalidFromKeywordAsIdentifierError struct {
	GotToken lexer.Token
}

var _ ParseError = &InvalidFromKeywordAsIdentifierError{}
var _ errors.UserError = &InvalidFromKeywordAsIdentifierError{}
var _ errors.SecondaryError = &InvalidFromKeywordAsIdentifierError{}
var _ errors.HasDocumentationLink = &InvalidFromKeywordAsIdentifierError{}

func (*InvalidFromKeywordAsIdentifierError) isParseError() {}

func (*InvalidFromKeywordAsIdentifierError) IsUserError() {}

func (e *InvalidFromKeywordAsIdentifierError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *InvalidFromKeywordAsIdentifierError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (*InvalidFromKeywordAsIdentifierError) Error() string {
	return "expected identifier, got keyword `from`"
}

func (*InvalidFromKeywordAsIdentifierError) SecondaryError() string {
	return "import declarations expect an identifier to import, not the `from` keyword in this position"
}

func (*InvalidFromKeywordAsIdentifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}

// InvalidInKeywordAsIdentifierError is reported when the `in` keyword is used as an identifier
// in an invalid context in a for statement.
type InvalidInKeywordAsIdentifierError struct {
	Pos ast.Position
}

var _ ParseError = &InvalidInKeywordAsIdentifierError{}
var _ errors.UserError = &InvalidInKeywordAsIdentifierError{}
var _ errors.SecondaryError = &InvalidInKeywordAsIdentifierError{}
var _ errors.HasDocumentationLink = &InvalidInKeywordAsIdentifierError{}

func (*InvalidInKeywordAsIdentifierError) isParseError() {}

func (*InvalidInKeywordAsIdentifierError) IsUserError() {}

func (e *InvalidInKeywordAsIdentifierError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidInKeywordAsIdentifierError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*InvalidInKeywordAsIdentifierError) Error() string {
	return "expected identifier, got keyword `in`"
}

func (*InvalidInKeywordAsIdentifierError) SecondaryError() string {
	return "the `in` keyword cannot be used as an identifier in a for-loop"
}

func (*InvalidInKeywordAsIdentifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/control-flow#for-in-statement"
}

// MissingInKeywordInForStatementError is reported when the `in` keyword is missing in a for statement.
type MissingInKeywordInForStatementError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingInKeywordInForStatementError{}
var _ errors.UserError = &MissingInKeywordInForStatementError{}
var _ errors.SecondaryError = &MissingInKeywordInForStatementError{}
var _ errors.HasDocumentationLink = &MissingInKeywordInForStatementError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingInKeywordInForStatementError{}

func (*MissingInKeywordInForStatementError) isParseError() {}

func (*MissingInKeywordInForStatementError) IsUserError() {}

func (e *MissingInKeywordInForStatementError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingInKeywordInForStatementError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingInKeywordInForStatementError) Error() string {
	return expectedButGotToken(
		"expected keyword `in`",
		e.GotToken.Type,
	)
}

func (*MissingInKeywordInForStatementError) SecondaryError() string {
	return "for-loops require the `in` keyword to separate the loop variable from the iterated value"
}

func (e *MissingInKeywordInForStatementError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `in`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: keywordInsertion(KeywordIn, e.GotToken.Type),
					Range: ast.Range{
						StartPos: e.GotToken.StartPos,
						EndPos:   e.GotToken.StartPos,
					},
				},
			},
		},
	}
}

func (*MissingInKeywordInForStatementError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/control-flow#for-in-statement"
}

// MissingConformanceError is reported when a colon for conformances is present,
// but no conformances follow.
type MissingConformanceError struct {
	Pos ast.Position
}

var _ ParseError = &MissingConformanceError{}
var _ errors.UserError = &MissingConformanceError{}
var _ errors.SecondaryError = &MissingConformanceError{}
var _ errors.HasDocumentationLink = &MissingConformanceError{}

func (*MissingConformanceError) isParseError() {}

func (*MissingConformanceError) IsUserError() {}

func (e *MissingConformanceError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingConformanceError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingConformanceError) Error() string {
	return "expected at least one conformance after colon (`:`)"
}

func (*MissingConformanceError) SecondaryError() string {
	return "provide at least one interface or type to conform to, or remove the colon (`:`) if no conformances are needed"
}

func (*MissingConformanceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

// InvalidInterfaceNameError is reported when an interface is missing a name.
type InvalidInterfaceNameError struct {
	GotToken lexer.Token
}

var _ ParseError = &InvalidInterfaceNameError{}
var _ errors.UserError = &InvalidInterfaceNameError{}
var _ errors.SecondaryError = &InvalidInterfaceNameError{}
var _ errors.HasDocumentationLink = &InvalidInterfaceNameError{}

func (*InvalidInterfaceNameError) isParseError() {}

func (*InvalidInterfaceNameError) IsUserError() {}

func (e *InvalidInterfaceNameError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *InvalidInterfaceNameError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *InvalidInterfaceNameError) Error() string {
	return "expected interface name, got keyword `interface`"
}

func (*InvalidInterfaceNameError) SecondaryError() string {
	return "interface declarations must have a unique name after the `interface` keyword"
}

func (*InvalidInterfaceNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

// AccessKeywordEntitlementNameError is reported when an access keyword (e.g. `all`, `self`)
// is used as an entitlement name.
type AccessKeywordEntitlementNameError struct {
	Keyword string
	ast.Range
}

var _ ParseError = &AccessKeywordEntitlementNameError{}
var _ errors.UserError = &AccessKeywordEntitlementNameError{}
var _ errors.SecondaryError = &AccessKeywordEntitlementNameError{}
var _ errors.HasDocumentationLink = &AccessKeywordEntitlementNameError{}

func (*AccessKeywordEntitlementNameError) isParseError() {}

func (*AccessKeywordEntitlementNameError) IsUserError() {}

func (e *AccessKeywordEntitlementNameError) Error() string {
	return fmt.Sprintf("unexpected non-nominal type: %s", e.Keyword)
}

func (*AccessKeywordEntitlementNameError) SecondaryError() string {
	return "use an entitlement name instead of an access control keyword"
}

func (*AccessKeywordEntitlementNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlements"
}

// InvalidEntitlementSeparatorError is reported when an invalid token is used as an entitlement separator.
type InvalidEntitlementSeparatorError struct {
	Token lexer.Token
}

var _ ParseError = &InvalidEntitlementSeparatorError{}
var _ errors.UserError = &InvalidEntitlementSeparatorError{}
var _ errors.SecondaryError = &InvalidEntitlementSeparatorError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidEntitlementSeparatorError{}
var _ errors.HasDocumentationLink = &InvalidEntitlementSeparatorError{}

func (*InvalidEntitlementSeparatorError) isParseError() {}

func (*InvalidEntitlementSeparatorError) IsUserError() {}

func (e *InvalidEntitlementSeparatorError) StartPosition() ast.Position {
	return e.Token.StartPos
}

func (e *InvalidEntitlementSeparatorError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Token.EndPos
}

func (e *InvalidEntitlementSeparatorError) Error() string {
	return expectedButGotToken(
		"expected entitlement separator",
		e.Token.Type,
	)
}

func (*InvalidEntitlementSeparatorError) SecondaryError() string {
	return "use a comma (`,`) for conjunctive entitlements " +
		"or a vertical bar (`|`) for disjunctive entitlements"
}

func (*InvalidEntitlementSeparatorError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlements"
}

func (e *InvalidEntitlementSeparatorError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	r := newLeftAttachedRange(e.Token.StartPos, code)
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert comma (conjunction)",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ",",
					Range:     r,
				},
			},
		},
		{
			Message: "Insert vertical bar (disjunction)",
			TextEdits: []ast.TextEdit{
				{
					Insertion: " |",
					Range:     r,
				},
			},
		},
	}
}

// UnexpectedTokenAtEndError is reported when there is an unexpected token at the end of the program
type UnexpectedTokenAtEndError struct {
	Token lexer.Token
}

var _ ParseError = &UnexpectedTokenAtEndError{}
var _ errors.UserError = &UnexpectedTokenAtEndError{}
var _ errors.SecondaryError = &UnexpectedTokenAtEndError{}
var _ errors.HasDocumentationLink = &UnexpectedTokenAtEndError{}

func (*UnexpectedTokenAtEndError) isParseError() {}

func (*UnexpectedTokenAtEndError) IsUserError() {}

func (e *UnexpectedTokenAtEndError) StartPosition() ast.Position {
	return e.Token.StartPos
}

func (e *UnexpectedTokenAtEndError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Token.EndPos
}

func (e *UnexpectedTokenAtEndError) Error() string {
	return fmt.Sprintf("unexpected token: %s", e.Token.Type)
}

func (*UnexpectedTokenAtEndError) SecondaryError() string {
	return "check for extra characters, missing semicolons, or incomplete statements"
}

func (*UnexpectedTokenAtEndError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// MissingCommentEndError is reported when a block comment is missing an end.
type MissingCommentEndError struct {
	Pos ast.Position
}

var _ ParseError = &MissingCommentEndError{}
var _ errors.UserError = &MissingCommentEndError{}
var _ errors.SecondaryError = &MissingCommentEndError{}
var _ errors.HasDocumentationLink = &MissingCommentEndError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingCommentEndError{}

func (*MissingCommentEndError) isParseError() {}

func (*MissingCommentEndError) IsUserError() {}

func (e *MissingCommentEndError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingCommentEndError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

// TODO(merge): Move and emit from lexer
func (e *MissingCommentEndError) Error() string {
	return "missing comment end (`*/`)"
}

func (*MissingCommentEndError) SecondaryError() string {
	return "ensure all block comments are properly closed with `*/`"
}

func (*MissingCommentEndError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax#comments"
}

func (e *MissingCommentEndError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `*/`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "*/",
					Range: ast.Range{
						StartPos: e.Pos,
						EndPos:   e.Pos,
					},
				},
			},
		},
	}
}

// UnexpectedTokenInBlockCommentError is reported when an unexpected token is found in a block comment.
type UnexpectedTokenInBlockCommentError struct {
	GotToken lexer.Token
}

var _ ParseError = &UnexpectedTokenInBlockCommentError{}
var _ errors.UserError = &UnexpectedTokenInBlockCommentError{}
var _ errors.SecondaryError = &UnexpectedTokenInBlockCommentError{}
var _ errors.HasDocumentationLink = &UnexpectedTokenInBlockCommentError{}

func (*UnexpectedTokenInBlockCommentError) isParseError() {}

func (*UnexpectedTokenInBlockCommentError) IsUserError() {}

func (e *UnexpectedTokenInBlockCommentError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *UnexpectedTokenInBlockCommentError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *UnexpectedTokenInBlockCommentError) Error() string {
	return fmt.Sprintf(
		"unexpected token %s in block comment",
		e.GotToken.Type,
	)
}

func (*UnexpectedTokenInBlockCommentError) SecondaryError() string {
	return "only text is allowed in a block comment"
}

func (*UnexpectedTokenInBlockCommentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax#comments"
}

// SpecialFunctionReturnTypeError is reported when a special function has a return type.
type SpecialFunctionReturnTypeError struct {
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ ParseError = &SpecialFunctionReturnTypeError{}
var _ errors.UserError = &SpecialFunctionReturnTypeError{}
var _ errors.SecondaryError = &SpecialFunctionReturnTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &SpecialFunctionReturnTypeError{}
var _ errors.HasDocumentationLink = &SpecialFunctionReturnTypeError{}

func (*SpecialFunctionReturnTypeError) isParseError() {}

func (*SpecialFunctionReturnTypeError) IsUserError() {}

func (e *SpecialFunctionReturnTypeError) Error() string {
	var kindDescription string
	if e.DeclarationKind != common.DeclarationKindUnknown {
		kindDescription = e.DeclarationKind.Name()
	} else {
		kindDescription = "special function"
	}

	return fmt.Sprintf("invalid return type for %s", kindDescription)
}

func (*SpecialFunctionReturnTypeError) SecondaryError() string {
	return "special functions like `init` or `prepare` cannot have return types"
}

func (e *SpecialFunctionReturnTypeError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	r := e.Range

	// Find the colon on the same line, if any
loop:
	for i := r.StartPos.Offset - 1; i >= 0; i-- {
		switch code[i] {
		case ' ', '\t':
			continue
		case ':':
			// If we find a colon, we remove the return type by adjusting the range
			// to exclude the colon and everything after it.
			r.StartPos = r.StartPos.Shifted(nil, -(r.StartPos.Offset - i))
			break loop
		default:
			// If we hit a non-whitespace character before finding a colon,
			// we assume the colon is not present and return no fixes.
			return nil
		}
	}

	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove return type from special function",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       r,
				},
			},
		},
	}
}

func (*SpecialFunctionReturnTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// MemberAccessMissingNameError is reported when a member access is missing a name.
type MemberAccessMissingNameError struct {
	GotToken lexer.Token
}

var _ ParseError = &MemberAccessMissingNameError{}
var _ errors.UserError = &MemberAccessMissingNameError{}
var _ errors.SecondaryError = &MemberAccessMissingNameError{}
var _ errors.HasDocumentationLink = &MemberAccessMissingNameError{}

func (*MemberAccessMissingNameError) isParseError() {}

func (*MemberAccessMissingNameError) IsUserError() {}

func (e *MemberAccessMissingNameError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MemberAccessMissingNameError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MemberAccessMissingNameError) Error() string {
	return expectedButGotToken(
		"expected member name",
		e.GotToken.Type,
	)
}

func (*MemberAccessMissingNameError) SecondaryError() string {
	return "after a dot (`.`), you must provide a valid identifier for the member name"
}

func (*MemberAccessMissingNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// WhitespaceAfterMemberAccessError is reported when there is whitespace after a member access operator.
type WhitespaceAfterMemberAccessError struct {
	OperatorTokenType lexer.TokenType
	WhitespaceRange   ast.Range
}

var _ ParseError = &WhitespaceAfterMemberAccessError{}
var _ errors.UserError = &WhitespaceAfterMemberAccessError{}
var _ errors.SecondaryError = &WhitespaceAfterMemberAccessError{}
var _ errors.HasDocumentationLink = &WhitespaceAfterMemberAccessError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &WhitespaceAfterMemberAccessError{}

func (*WhitespaceAfterMemberAccessError) isParseError() {}

func (*WhitespaceAfterMemberAccessError) IsUserError() {}

func (e *WhitespaceAfterMemberAccessError) StartPosition() ast.Position {
	return e.WhitespaceRange.StartPos
}

func (e *WhitespaceAfterMemberAccessError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.WhitespaceRange.EndPos
}

func (e *WhitespaceAfterMemberAccessError) Error() string {
	return fmt.Sprintf("invalid whitespace after %#q", e.OperatorTokenType)
}

func (e *WhitespaceAfterMemberAccessError) SecondaryError() string {
	return fmt.Sprintf("remove the space between %#q and the member name", e.OperatorTokenType)
}

func (e *WhitespaceAfterMemberAccessError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove whitespace",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       e.WhitespaceRange,
				},
			},
		},
	}
}

func (*WhitespaceAfterMemberAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// InvalidStaticModifierError

type InvalidStaticModifierError struct {
	Pos             ast.Position
	DeclarationKind common.DeclarationKind
}

var _ ParseError = &InvalidStaticModifierError{}
var _ errors.UserError = &InvalidStaticModifierError{}
var _ errors.SecondaryError = &InvalidStaticModifierError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidStaticModifierError{}

func (*InvalidStaticModifierError) isParseError() {}

func (*InvalidStaticModifierError) IsUserError() {}

func (e *InvalidStaticModifierError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidStaticModifierError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	return e.Pos.Shifted(memoryGauge, len(KeywordStatic)-1)
}

func (e *InvalidStaticModifierError) Error() string {
	return fmt.Sprintf(
		"invalid `static` modifier for %s",
		e.DeclarationKind.Name(),
	)
}

func (e *InvalidStaticModifierError) SecondaryError() string {
	return fmt.Sprintf(
		"the `static` modifier can only be used on on fields and functions, "+
			"not on %s declarations",
		e.DeclarationKind.Name(),
	)
}

func (e *InvalidStaticModifierError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove `static` modifier",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       ast.NewRangeFromPositioned(nil, e),
				},
			},
		},
	}
}

// InvalidNativeModifierError

type InvalidNativeModifierError struct {
	Pos             ast.Position
	DeclarationKind common.DeclarationKind
}

var _ ParseError = &InvalidNativeModifierError{}
var _ errors.UserError = &InvalidNativeModifierError{}
var _ errors.SecondaryError = &InvalidNativeModifierError{}

func (*InvalidNativeModifierError) isParseError() {}

func (*InvalidNativeModifierError) IsUserError() {}

func (e *InvalidNativeModifierError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidNativeModifierError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	return e.Pos.Shifted(memoryGauge, len(KeywordNative)-1)
}

func (e *InvalidNativeModifierError) Error() string {
	return fmt.Sprintf(
		"invalid `native` modifier for %s",
		e.DeclarationKind.Name(),
	)
}

func (e *InvalidNativeModifierError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove `native` modifier",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       ast.NewRangeFromPositioned(nil, e),
				},
			},
		},
	}
}

func (e *InvalidNativeModifierError) SecondaryError() string {
	return fmt.Sprintf(
		"the `native` modifier can only be used on fields and functions, not on %s declarations",
		e.DeclarationKind.Name(),
	)
}

// NonNominalTypeError

type NonNominalTypeError struct {
	Pos  ast.Position
	Type ast.Type
}

var _ ParseError = &NonNominalTypeError{}
var _ errors.UserError = &NonNominalTypeError{}
var _ errors.SecondaryError = &NonNominalTypeError{}
var _ errors.HasDocumentationLink = &NonNominalTypeError{}

func (*NonNominalTypeError) isParseError() {}

func (*NonNominalTypeError) IsUserError() {}

func (e *NonNominalTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *NonNominalTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *NonNominalTypeError) Error() string {
	return fmt.Sprintf("expected nominal type, got non-nominal type %#q", e.Type)
}

func (*NonNominalTypeError) SecondaryError() string {
	return "expected a nominal type (like a struct, resource, or interface name)"
}

func (*NonNominalTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/"
}

// NestedTypeMissingNameError is reported when a nested type is missing a name.
type NestedTypeMissingNameError struct {
	GotToken lexer.Token
}

var _ ParseError = &NestedTypeMissingNameError{}
var _ errors.UserError = &NestedTypeMissingNameError{}
var _ errors.SecondaryError = &NestedTypeMissingNameError{}
var _ errors.HasDocumentationLink = &NestedTypeMissingNameError{}

func (*NestedTypeMissingNameError) isParseError() {}

func (*NestedTypeMissingNameError) IsUserError() {}

func (e *NestedTypeMissingNameError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *NestedTypeMissingNameError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *NestedTypeMissingNameError) Error() string {
	return expectedButGotToken(
		"expected nested type name after dot (`.`)",
		e.GotToken.Type,
	)
}

func (*NestedTypeMissingNameError) SecondaryError() string {
	return "after a dot (`.`), you must provide a valid identifier for the nested type name"
}

func (*NestedTypeMissingNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

// MissingForKeywordInAttachmentDeclarationError is reported when the 'for' keyword is missing in an attachment declaration.
type MissingForKeywordInAttachmentDeclarationError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingForKeywordInAttachmentDeclarationError{}
var _ errors.UserError = &MissingForKeywordInAttachmentDeclarationError{}
var _ errors.SecondaryError = &MissingForKeywordInAttachmentDeclarationError{}
var _ errors.HasDocumentationLink = &MissingForKeywordInAttachmentDeclarationError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingForKeywordInAttachmentDeclarationError{}

func (*MissingForKeywordInAttachmentDeclarationError) isParseError() {}

func (*MissingForKeywordInAttachmentDeclarationError) IsUserError() {}

func (e *MissingForKeywordInAttachmentDeclarationError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingForKeywordInAttachmentDeclarationError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingForKeywordInAttachmentDeclarationError) Error() string {
	return expectedButGotToken(
		"expected keyword `for`",
		e.GotToken.Type,
	)
}

func (*MissingForKeywordInAttachmentDeclarationError) SecondaryError() string {
	return "the attachment declaration requires the `for` keyword to specify the target"
}

func (*MissingForKeywordInAttachmentDeclarationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments#declaring-attachments"
}

func (e *MissingForKeywordInAttachmentDeclarationError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `for`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: keywordInsertion(KeywordFor, e.GotToken.Type),
					Range: ast.Range{
						StartPos: e.GotToken.StartPos,
						EndPos:   e.GotToken.StartPos,
					},
				},
			},
		},
	}
}

// InvalidAttachmentBaseTypeError is reported when an attachment declaration has an invalid base type.
type InvalidAttachmentBaseTypeError struct {
	ast.Range
}

var _ ParseError = &InvalidAttachmentBaseTypeError{}
var _ errors.UserError = &InvalidAttachmentBaseTypeError{}
var _ errors.SecondaryError = &InvalidAttachmentBaseTypeError{}
var _ errors.HasDocumentationLink = &InvalidAttachmentBaseTypeError{}

func (*InvalidAttachmentBaseTypeError) isParseError() {}

func (*InvalidAttachmentBaseTypeError) IsUserError() {}

func (e *InvalidAttachmentBaseTypeError) Error() string {
	return "expected nominal type"
}

func (*InvalidAttachmentBaseTypeError) SecondaryError() string {
	return "attachments can only be declared for nominal types"
}

func (*InvalidAttachmentBaseTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments#declaring-attachments"
}

// DeclarationMissingOpeningBraceError is reported when a declaration is missing an opening brace.
type DeclarationMissingOpeningBraceError struct {
	Kind     common.DeclarationKind
	GotToken lexer.Token
}

var _ ParseError = &DeclarationMissingOpeningBraceError{}
var _ errors.UserError = &DeclarationMissingOpeningBraceError{}
var _ errors.SecondaryError = &DeclarationMissingOpeningBraceError{}
var _ errors.HasDocumentationLink = &DeclarationMissingOpeningBraceError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &DeclarationMissingOpeningBraceError{}

func (*DeclarationMissingOpeningBraceError) isParseError() {}

func (*DeclarationMissingOpeningBraceError) IsUserError() {}

func (e *DeclarationMissingOpeningBraceError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *DeclarationMissingOpeningBraceError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *DeclarationMissingOpeningBraceError) Error() string {
	return expectedButGotToken(
		fmt.Sprintf(
			"expected opening brace (`{`) at start of %s declaration",
			e.Kind.Name(),
		),
		e.GotToken.Type,
	)
}

func (e *DeclarationMissingOpeningBraceError) SecondaryError() string {
	return fmt.Sprintf(
		"%s declarations must be enclosed in braces `{ ... }`; add the missing opening brace (`{`)",
		e.Kind.Name(),
	)
}

func (*DeclarationMissingOpeningBraceError) DocumentationLink() string {
	// TODO: improve this link to point to the specific page based on the declaration kind
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *DeclarationMissingOpeningBraceError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert opening brace",
			TextEdits: []ast.TextEdit{
				{
					Insertion: " {",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// DeclarationMissingClosingBraceError is reported when a declaration is missing a closing brace.
type DeclarationMissingClosingBraceError struct {
	Kind     common.DeclarationKind
	GotToken lexer.Token
}

var _ ParseError = &DeclarationMissingClosingBraceError{}
var _ errors.UserError = &DeclarationMissingClosingBraceError{}
var _ errors.SecondaryError = &DeclarationMissingClosingBraceError{}
var _ errors.HasDocumentationLink = &DeclarationMissingClosingBraceError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &DeclarationMissingClosingBraceError{}

func (*DeclarationMissingClosingBraceError) isParseError() {}

func (*DeclarationMissingClosingBraceError) IsUserError() {}

func (e *DeclarationMissingClosingBraceError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *DeclarationMissingClosingBraceError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *DeclarationMissingClosingBraceError) Error() string {
	return expectedButGotToken(
		fmt.Sprintf(
			"expected closing brace (`}`) at end of %s declaration",
			e.Kind.Name(),
		),
		e.GotToken.Type,
	)
}

func (e *DeclarationMissingClosingBraceError) SecondaryError() string {
	return fmt.Sprintf(
		"%s declarations must be enclosed in braces `{ ... }`; add the missing closing brace (`}`)",
		e.Kind.Name(),
	)
}

func (*DeclarationMissingClosingBraceError) DocumentationLink() string {
	// TODO: improve this link to point to the specific page based on the declaration kind
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *DeclarationMissingClosingBraceError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing brace",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "}",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingOpeningBraceError is reported when an opening brace is missing .
type MissingOpeningBraceError struct {
	Description string
	GotToken    lexer.Token
}

var _ ParseError = &MissingOpeningBraceError{}
var _ errors.UserError = &MissingOpeningBraceError{}
var _ errors.SecondaryError = &MissingOpeningBraceError{}
var _ errors.HasDocumentationLink = &MissingOpeningBraceError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingOpeningBraceError{}

func (*MissingOpeningBraceError) isParseError() {}

func (*MissingOpeningBraceError) IsUserError() {}

func (e *MissingOpeningBraceError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingOpeningBraceError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingOpeningBraceError) Error() string {
	return expectedButGotToken(
		fmt.Sprintf(
			"expected opening brace (`{`) at start of %s",
			e.Description,
		),
		e.GotToken.Type,
	)
}

func (e *MissingOpeningBraceError) SecondaryError() string {
	return fmt.Sprintf(
		"%s must be enclosed in braces `{ ... }`; add the missing opening brace (`{`)",
		e.Description,
	)
}

func (*MissingOpeningBraceError) DocumentationLink() string {
	// TODO: improve this link to point to the specific page
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *MissingOpeningBraceError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert opening brace",
			TextEdits: []ast.TextEdit{
				{
					Insertion: " {",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingClosingBraceError is reported when a closing brace is missing .
type MissingClosingBraceError struct {
	Description string
	GotToken    lexer.Token
}

var _ ParseError = &MissingClosingBraceError{}
var _ errors.UserError = &MissingClosingBraceError{}
var _ errors.SecondaryError = &MissingClosingBraceError{}
var _ errors.HasDocumentationLink = &MissingClosingBraceError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingBraceError{}

func (*MissingClosingBraceError) isParseError() {}

func (*MissingClosingBraceError) IsUserError() {}

func (e *MissingClosingBraceError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingClosingBraceError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingClosingBraceError) Error() string {
	return expectedButGotToken(
		fmt.Sprintf(
			"expected closing brace (`}`) at end of %s",
			e.Description,
		),
		e.GotToken.Type,
	)
}

func (e *MissingClosingBraceError) SecondaryError() string {
	return fmt.Sprintf(
		"%s must be enclosed in braces `{ ... }`; add the missing closing brace (`}`)",
		e.Description,
	)
}

func (*MissingClosingBraceError) DocumentationLink() string {
	// TODO: improve this link to point to the specific page
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *MissingClosingBraceError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing brace",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "}",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingEndOfParenthesizedTypeError is reported when a parenthesized type is missing a closing parenthesis.
type MissingEndOfParenthesizedTypeError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingEndOfParenthesizedTypeError{}
var _ errors.UserError = &MissingEndOfParenthesizedTypeError{}
var _ errors.SecondaryError = &MissingEndOfParenthesizedTypeError{}
var _ errors.HasDocumentationLink = &MissingEndOfParenthesizedTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingEndOfParenthesizedTypeError{}

func (*MissingEndOfParenthesizedTypeError) isParseError() {}

func (*MissingEndOfParenthesizedTypeError) IsUserError() {}

func (e *MissingEndOfParenthesizedTypeError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingEndOfParenthesizedTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingEndOfParenthesizedTypeError) Error() string {
	return expectedButGotToken(
		"expected closing parenthesis (`)`) at end of parenthesized type",
		e.GotToken.Type,
	)
}

func (*MissingEndOfParenthesizedTypeError) SecondaryError() string {
	return "parenthesized types must be properly closed with a closing parenthesis (`)`)"
}

func (*MissingEndOfParenthesizedTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *MissingEndOfParenthesizedTypeError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ")",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingEndOfParenthesizedExpressionError is reported when a parenthesized expression is missing a closing parenthesis.
type MissingEndOfParenthesizedExpressionError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingEndOfParenthesizedExpressionError{}
var _ errors.UserError = &MissingEndOfParenthesizedExpressionError{}
var _ errors.SecondaryError = &MissingEndOfParenthesizedExpressionError{}
var _ errors.HasDocumentationLink = &MissingEndOfParenthesizedExpressionError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingEndOfParenthesizedExpressionError{}

func (*MissingEndOfParenthesizedExpressionError) isParseError() {}

func (*MissingEndOfParenthesizedExpressionError) IsUserError() {}

func (e *MissingEndOfParenthesizedExpressionError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingEndOfParenthesizedExpressionError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingEndOfParenthesizedExpressionError) Error() string {
	return expectedButGotToken(
		"expected closing parenthesis (`)`) at end of parenthesized expression",
		e.GotToken.Type,
	)
}

func (*MissingEndOfParenthesizedExpressionError) SecondaryError() string {
	return "parenthesized expressions must be properly closed with a closing parenthesis (`)`)"
}

func (*MissingEndOfParenthesizedExpressionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *MissingEndOfParenthesizedExpressionError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ")",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingClosingBracketInArrayTypeError is reported when an array type is missing a closing bracket.
type MissingClosingBracketInArrayTypeError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingClosingBracketInArrayTypeError{}
var _ errors.UserError = &MissingClosingBracketInArrayTypeError{}
var _ errors.SecondaryError = &MissingClosingBracketInArrayTypeError{}
var _ errors.HasDocumentationLink = &MissingClosingBracketInArrayTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingBracketInArrayTypeError{}

func (*MissingClosingBracketInArrayTypeError) isParseError() {}

func (*MissingClosingBracketInArrayTypeError) IsUserError() {}

func (e *MissingClosingBracketInArrayTypeError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingClosingBracketInArrayTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingClosingBracketInArrayTypeError) Error() string {
	return expectedButGotToken(
		"expected closing bracket (`]`) at end of array type",
		e.GotToken.Type,
	)
}

func (*MissingClosingBracketInArrayTypeError) SecondaryError() string {
	return "array types must be properly closed with a closing bracket (`]`)"
}

func (*MissingClosingBracketInArrayTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/arrays#array-types"
}

func (e *MissingClosingBracketInArrayTypeError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing bracket",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "]",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingClosingBracketInArrayExpressionError is reported when an array expression is missing a closing bracket.
type MissingClosingBracketInArrayExpressionError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingClosingBracketInArrayExpressionError{}
var _ errors.UserError = &MissingClosingBracketInArrayExpressionError{}
var _ errors.SecondaryError = &MissingClosingBracketInArrayExpressionError{}
var _ errors.HasDocumentationLink = &MissingClosingBracketInArrayExpressionError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingBracketInArrayExpressionError{}

func (*MissingClosingBracketInArrayExpressionError) isParseError() {}

func (*MissingClosingBracketInArrayExpressionError) IsUserError() {}

func (e *MissingClosingBracketInArrayExpressionError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingClosingBracketInArrayExpressionError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingClosingBracketInArrayExpressionError) Error() string {
	return expectedButGotToken(
		"expected closing bracket (`]`) at end of array expression",
		e.GotToken.Type,
	)
}

func (*MissingClosingBracketInArrayExpressionError) SecondaryError() string {
	return "array expressions must be properly closed with a closing bracket (`]`)"
}

func (*MissingClosingBracketInArrayExpressionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/arrays#array-literals"
}

func (e *MissingClosingBracketInArrayExpressionError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing bracket",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "]",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingClosingBraceInDictionaryExpressionError is reported when a dictionary expression is missing a closing brace.
type MissingClosingBraceInDictionaryExpressionError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingClosingBraceInDictionaryExpressionError{}
var _ errors.UserError = &MissingClosingBraceInDictionaryExpressionError{}
var _ errors.SecondaryError = &MissingClosingBraceInDictionaryExpressionError{}
var _ errors.HasDocumentationLink = &MissingClosingBraceInDictionaryExpressionError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingBraceInDictionaryExpressionError{}

func (*MissingClosingBraceInDictionaryExpressionError) isParseError() {}

func (*MissingClosingBraceInDictionaryExpressionError) IsUserError() {}

func (e *MissingClosingBraceInDictionaryExpressionError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingClosingBraceInDictionaryExpressionError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingClosingBraceInDictionaryExpressionError) Error() string {
	return expectedButGotToken(
		"expected closing brace (`}`) at end of dictionary expression",
		e.GotToken.Type,
	)
}

func (*MissingClosingBraceInDictionaryExpressionError) SecondaryError() string {
	return "dictionary expressions must be properly closed with a closing brace (`}`)"
}

func (*MissingClosingBraceInDictionaryExpressionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/dictionaries#dictionary-literals"
}

func (e *MissingClosingBraceInDictionaryExpressionError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing brace",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "}",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingColonInDictionaryEntryError is reported when a dictionary entry is missing a colon.
type MissingColonInDictionaryEntryError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingColonInDictionaryEntryError{}
var _ errors.UserError = &MissingColonInDictionaryEntryError{}
var _ errors.SecondaryError = &MissingColonInDictionaryEntryError{}
var _ errors.HasDocumentationLink = &MissingColonInDictionaryEntryError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingColonInDictionaryEntryError{}

func (*MissingColonInDictionaryEntryError) isParseError() {}

func (*MissingColonInDictionaryEntryError) IsUserError() {}

func (e *MissingColonInDictionaryEntryError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingColonInDictionaryEntryError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingColonInDictionaryEntryError) Error() string {
	return expectedButGotToken(
		"expected colon (`:`) in dictionary entry",
		e.GotToken.Type,
	)
}

func (*MissingColonInDictionaryEntryError) SecondaryError() string {
	return "a colon (`:`) is required to separate the key and value in a dictionary entry"
}

func (*MissingColonInDictionaryEntryError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/dictionaries#dictionary-literals"
}

func (e *MissingColonInDictionaryEntryError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert colon",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ":",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingColonInConditionalExpressionError is reported when a conditional expression is missing a colon.
type MissingColonInConditionalExpressionError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingColonInConditionalExpressionError{}
var _ errors.UserError = &MissingColonInConditionalExpressionError{}
var _ errors.SecondaryError = &MissingColonInConditionalExpressionError{}
var _ errors.HasDocumentationLink = &MissingColonInConditionalExpressionError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingColonInConditionalExpressionError{}

func (*MissingColonInConditionalExpressionError) isParseError() {}

func (*MissingColonInConditionalExpressionError) IsUserError() {}

func (e *MissingColonInConditionalExpressionError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingColonInConditionalExpressionError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingColonInConditionalExpressionError) Error() string {
	return expectedButGotToken(
		"expected colon (`:`) in conditional expression",
		e.GotToken.Type,
	)
}

func (*MissingColonInConditionalExpressionError) SecondaryError() string {
	return "a colon (`:`) is required to separate the 'then' and 'else' expressions"
}

func (*MissingColonInConditionalExpressionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/bitwise-ternary-operators#ternary-conditional-operator"
}

func (e *MissingColonInConditionalExpressionError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert colon",
			TextEdits: []ast.TextEdit{
				{
					Insertion: " :",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingSlashInPathExpressionError is reported when a path expression is missing a slash.
type MissingSlashInPathExpressionError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingSlashInPathExpressionError{}
var _ errors.UserError = &MissingSlashInPathExpressionError{}
var _ errors.SecondaryError = &MissingSlashInPathExpressionError{}
var _ errors.HasDocumentationLink = &MissingSlashInPathExpressionError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingSlashInPathExpressionError{}

func (*MissingSlashInPathExpressionError) isParseError() {}

func (*MissingSlashInPathExpressionError) IsUserError() {}

func (e *MissingSlashInPathExpressionError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingSlashInPathExpressionError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingSlashInPathExpressionError) Error() string {
	return expectedButGotToken(
		"expected slash (`/`) in path expression",
		e.GotToken.Type,
	)
}

func (*MissingSlashInPathExpressionError) SecondaryError() string {
	return "a slash (`/`) is required to separate the domain and identifier in a path expression"
}

func (*MissingSlashInPathExpressionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/accounts/paths"
}

func (e *MissingSlashInPathExpressionError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert slash",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "/",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingClosingBracketInIndexExpressionError is reported when an index expression is missing a closing bracket.
type MissingClosingBracketInIndexExpressionError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingClosingBracketInIndexExpressionError{}
var _ errors.UserError = &MissingClosingBracketInIndexExpressionError{}
var _ errors.SecondaryError = &MissingClosingBracketInIndexExpressionError{}
var _ errors.HasDocumentationLink = &MissingClosingBracketInIndexExpressionError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingBracketInIndexExpressionError{}

func (*MissingClosingBracketInIndexExpressionError) isParseError() {}

func (*MissingClosingBracketInIndexExpressionError) IsUserError() {}

func (e *MissingClosingBracketInIndexExpressionError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingClosingBracketInIndexExpressionError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingClosingBracketInIndexExpressionError) Error() string {
	return expectedButGotToken(
		"expected closing bracket (`]`) at end of index expression",
		e.GotToken.Type,
	)
}

func (*MissingClosingBracketInIndexExpressionError) SecondaryError() string {
	return "index expressions must be properly closed with a closing bracket (`]`)"
}

func (*MissingClosingBracketInIndexExpressionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/arrays#array-indexing"
}

func (e *MissingClosingBracketInIndexExpressionError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing bracket",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "]",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingClosingBraceInIntersectionOrDictionaryTypeError is reported when an intersection or dictionary type
// is missing a closing brace.
type MissingClosingBraceInIntersectionOrDictionaryTypeError struct {
	Pos ast.Position
}

var _ ParseError = &MissingClosingBraceInIntersectionOrDictionaryTypeError{}
var _ errors.UserError = &MissingClosingBraceInIntersectionOrDictionaryTypeError{}
var _ errors.SecondaryError = &MissingClosingBraceInIntersectionOrDictionaryTypeError{}
var _ errors.HasDocumentationLink = &MissingClosingBraceInIntersectionOrDictionaryTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingBraceInIntersectionOrDictionaryTypeError{}

func (*MissingClosingBraceInIntersectionOrDictionaryTypeError) isParseError() {}

func (*MissingClosingBraceInIntersectionOrDictionaryTypeError) IsUserError() {}

func (e *MissingClosingBraceInIntersectionOrDictionaryTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingClosingBraceInIntersectionOrDictionaryTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingClosingBraceInIntersectionOrDictionaryTypeError) Error() string {
	return "missing closing brace (`}`) at end of intersection type or dictionary type"
}

func (*MissingClosingBraceInIntersectionOrDictionaryTypeError) SecondaryError() string {
	return "intersection types and dictionary type must be properly closed with a closing brace (`}`)"
}

func (*MissingClosingBraceInIntersectionOrDictionaryTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
}

func (e *MissingClosingBraceInIntersectionOrDictionaryTypeError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing brace",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "}",
					Range:     newLeftAttachedRange(e.Pos, code),
				},
			},
		},
	}
}

// MissingClosingParenInAuthError is reported when an authorization is missing a closing parenthesis.
type MissingClosingParenInAuthError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingClosingParenInAuthError{}
var _ errors.UserError = &MissingClosingParenInAuthError{}
var _ errors.SecondaryError = &MissingClosingParenInAuthError{}
var _ errors.HasDocumentationLink = &MissingClosingParenInAuthError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingParenInAuthError{}

func (*MissingClosingParenInAuthError) isParseError() {}

func (*MissingClosingParenInAuthError) IsUserError() {}

func (e *MissingClosingParenInAuthError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingClosingParenInAuthError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingClosingParenInAuthError) Error() string {
	return expectedButGotToken(
		"expected closing parenthesis (`)`) at end of authorization",
		e.GotToken.Type,
	)
}

func (*MissingClosingParenInAuthError) SecondaryError() string {
	return "the authorization must be properly closed with a closing parenthesis (`auth(...)`)"
}

func (*MissingClosingParenInAuthError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/references#authorized-references"
}

func (e *MissingClosingParenInAuthError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ")",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingAmpersandInAuthReferenceError is reported when an authorized reference is missing an ampersand.
type MissingAmpersandInAuthReferenceError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingAmpersandInAuthReferenceError{}
var _ errors.UserError = &MissingAmpersandInAuthReferenceError{}
var _ errors.SecondaryError = &MissingAmpersandInAuthReferenceError{}
var _ errors.HasDocumentationLink = &MissingAmpersandInAuthReferenceError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingAmpersandInAuthReferenceError{}

func (*MissingAmpersandInAuthReferenceError) isParseError() {}

func (*MissingAmpersandInAuthReferenceError) IsUserError() {}

func (e *MissingAmpersandInAuthReferenceError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingAmpersandInAuthReferenceError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingAmpersandInAuthReferenceError) Error() string {
	return expectedButGotToken(
		"expected ampersand (`&`) in authorized reference",
		e.GotToken.Type,
	)
}

func (*MissingAmpersandInAuthReferenceError) SecondaryError() string {
	return "authorized references must contain an ampersand (`&`); insert the missing ampersand"
}

func (*MissingAmpersandInAuthReferenceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/references#authorized-references"
}

func (e *MissingAmpersandInAuthReferenceError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert ampersand",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "&",
					Range: ast.Range{
						StartPos: e.GotToken.StartPos,
						EndPos:   e.GotToken.StartPos,
					},
				},
			},
		},
	}
}

// MissingOpeningParenInNominalTypeInvocationError is reported when a nominal type invocation
// is missing an opening parenthesis.
type MissingOpeningParenInNominalTypeInvocationError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingOpeningParenInNominalTypeInvocationError{}
var _ errors.UserError = &MissingOpeningParenInNominalTypeInvocationError{}
var _ errors.SecondaryError = &MissingOpeningParenInNominalTypeInvocationError{}
var _ errors.HasDocumentationLink = &MissingOpeningParenInNominalTypeInvocationError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingOpeningParenInNominalTypeInvocationError{}

func (*MissingOpeningParenInNominalTypeInvocationError) isParseError() {}

func (*MissingOpeningParenInNominalTypeInvocationError) IsUserError() {}

func (e *MissingOpeningParenInNominalTypeInvocationError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingOpeningParenInNominalTypeInvocationError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingOpeningParenInNominalTypeInvocationError) Error() string {
	return expectedButGotToken(
		"expected opening parenthesis (`(`) to construct an instance of the type",
		e.GotToken.Type,
	)
}

func (*MissingOpeningParenInNominalTypeInvocationError) SecondaryError() string {
	return "an instance of the nominal type is expected here; " +
		"call the constructor by adding comma-separated arguments in parentheses `(...)`"
}

func (*MissingOpeningParenInNominalTypeInvocationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
}

func (e *MissingOpeningParenInNominalTypeInvocationError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert opening parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "(",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingOpeningParenInFunctionTypeError is reported when a function type parameter list
// is missing an opening parenthesis.
type MissingOpeningParenInFunctionTypeError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingOpeningParenInFunctionTypeError{}
var _ errors.UserError = &MissingOpeningParenInFunctionTypeError{}
var _ errors.SecondaryError = &MissingOpeningParenInFunctionTypeError{}
var _ errors.HasDocumentationLink = &MissingOpeningParenInFunctionTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingOpeningParenInFunctionTypeError{}

func (*MissingOpeningParenInFunctionTypeError) isParseError() {}

func (*MissingOpeningParenInFunctionTypeError) IsUserError() {}

func (e *MissingOpeningParenInFunctionTypeError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingOpeningParenInFunctionTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingOpeningParenInFunctionTypeError) Error() string {
	return expectedButGotToken(
		"expected opening parenthesis (`(`) at start of function type parameter list",
		e.GotToken.Type,
	)
}

func (*MissingOpeningParenInFunctionTypeError) SecondaryError() string {
	return "function type parameter lists must be wrapped in parentheses (`(...)`); " +
		"add the missing opening parenthesis (`(`)"
}

func (*MissingOpeningParenInFunctionTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions#function-types"
}

func (e *MissingOpeningParenInFunctionTypeError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert opening parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "(",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}

// MissingClosingParenInFunctionTypeError is reported when a function type parameter list
// is missing a closing parenthesis.
type MissingClosingParenInFunctionTypeError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingClosingParenInFunctionTypeError{}
var _ errors.UserError = &MissingClosingParenInFunctionTypeError{}
var _ errors.SecondaryError = &MissingClosingParenInFunctionTypeError{}
var _ errors.HasDocumentationLink = &MissingClosingParenInFunctionTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingClosingParenInFunctionTypeError{}

func (*MissingClosingParenInFunctionTypeError) isParseError() {}

func (*MissingClosingParenInFunctionTypeError) IsUserError() {}

func (e *MissingClosingParenInFunctionTypeError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingClosingParenInFunctionTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingClosingParenInFunctionTypeError) Error() string {
	return expectedButGotToken(
		"expected closing parenthesis (`)`) at end of function type parameter list",
		e.GotToken.Type,
	)
}

func (*MissingClosingParenInFunctionTypeError) SecondaryError() string {
	return "function type parameter lists must be enclosed in parentheses (`(...)`); " +
		"add the missing closing parenthesis (`)`)"
}

func (*MissingClosingParenInFunctionTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions#function-types"
}

func (e *MissingClosingParenInFunctionTypeError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert closing parenthesis",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ")",
					Range:     newLeftAttachedRange(e.GotToken.StartPos, code),
				},
			},
		},
	}
}
