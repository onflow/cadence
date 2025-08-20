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

func (*InvalidIntegerLiteralError) isParseError() {}

func (*InvalidIntegerLiteralError) IsUserError() {}

func (e *InvalidIntegerLiteralError) Error() string {
	if e.IntegerLiteralKind == common.IntegerLiteralKindUnknown {
		return fmt.Sprintf(
			"invalid integer literal `%s`: %s",
			e.Literal,
			e.InvalidIntegerLiteralKind.Description(),
		)
	}

	return fmt.Sprintf(
		"invalid %s integer literal `%s`: %s",
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
		return "remove the leading underscore"
	case InvalidNumberLiteralKindTrailingUnderscore:
		return "remove the trailing underscore"
	case InvalidNumberLiteralKindUnknownPrefix:
		return "did you mean `0x` (hexadecimal), `0b` (binary), or `0o` (octal)?"
	case InvalidNumberLiteralKindMissingDigits:
		return "consider adding a 0"
	}

	panic(errors.NewUnreachableError())
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
	return "Consider extracting the sub-expressions out and storing the intermediate results in local variables"
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
var _ errors.SecondaryError = TypeDepthLimitReachedError{}

func (TypeDepthLimitReachedError) isParseError() {}

func (TypeDepthLimitReachedError) IsUserError() {}

func (TypeDepthLimitReachedError) Error() string {
	return fmt.Sprintf(
		"type too deeply nested, exceeded depth limit of %d",
		typeDepthLimit,
	)
}

func (TypeDepthLimitReachedError) SecondaryError() string {
	return "Refactor the type so that the depth of nesting is less than the limit"
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

// UnexpectedEOFExpectedTypeError is reported when the end of the program is reached unexpectedly, but a type was expected.
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
	return "invalid end of input, expected type"
}

func (*UnexpectedEOFExpectedTypeError) SecondaryError() string {
	return "check for incomplete expressions, missing tokens, or unterminated strings/comments"
}

func (*UnexpectedEOFExpectedTypeError) DocumentationLink() string {
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
	return "statements on the same line must be separated with a semicolon"
}

func (*StatementSeparationError) SecondaryError() string {
	return "add a semicolon (;) between statements or place each statement on a separate line"
}

func (e *StatementSeparationError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Add semicolon to separate statements",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "; ",
					Range: ast.Range{
						StartPos: e.Pos,
						EndPos:   e.Pos,
					},
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
	return "missing comma after parameter"
}

func (*MissingCommaInParameterListError) SecondaryError() string {
	return "add a comma to separate parameters in the parameter list"
}

func (e *MissingCommaInParameterListError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Add comma to separate parameters",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ", ",
					Range: ast.Range{
						StartPos: e.Pos,
						EndPos:   e.Pos,
					},
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

func (*MissingStartOfParameterListError) isParseError() {}

func (*MissingStartOfParameterListError) IsUserError() {}

func (e *MissingStartOfParameterListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingStartOfParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingStartOfParameterListError) Error() string {
	return fmt.Sprintf(
		"expected %s as start of parameter list, got %s",
		lexer.TokenParenOpen,
		e.GotToken.Type,
	)
}

func (*MissingStartOfParameterListError) SecondaryError() string {
	return "function parameters must be enclosed in parentheses"
}

func (*MissingStartOfParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
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
	return fmt.Sprintf(
		"expected parameter or end of parameter list, got %s",
		e.GotToken.Type,
	)
}

func (*UnexpectedTokenInParameterListError) SecondaryError() string {
	return "parameters must be separated by commas, and the list must end with a closing parenthesis"
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

func (*MissingClosingParenInParameterListError) isParseError() {}

func (*MissingClosingParenInParameterListError) IsUserError() {}

func (e *MissingClosingParenInParameterListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingClosingParenInParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingClosingParenInParameterListError) Error() string {
	return fmt.Sprintf(
		"missing %s at end of parameter list",
		lexer.TokenParenClose,
	)
}

func (*MissingClosingParenInParameterListError) SecondaryError() string {
	return "function parameter lists must be properly closed with a closing parenthesis"
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

func (*ExpectedCommaOrEndOfParameterListError) isParseError() {}

func (*ExpectedCommaOrEndOfParameterListError) IsUserError() {}

func (e *ExpectedCommaOrEndOfParameterListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *ExpectedCommaOrEndOfParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *ExpectedCommaOrEndOfParameterListError) Error() string {
	return fmt.Sprintf(
		"expected comma or end of parameter list, got %s",
		e.GotToken.Type,
	)
}

func (*ExpectedCommaOrEndOfParameterListError) SecondaryError() string {
	return "multiple parameters must be separated by commas, and the parameter list must end with a closing parenthesis"
}

func (*ExpectedCommaOrEndOfParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// MissingColonAfterParameterNameError is reported when a colon is missing after a parameter name.
type MissingColonAfterParameterNameError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingColonAfterParameterNameError{}
var _ errors.UserError = &MissingColonAfterParameterNameError{}
var _ errors.SecondaryError = &MissingColonAfterParameterNameError{}
var _ errors.HasDocumentationLink = &MissingColonAfterParameterNameError{}

func (*MissingColonAfterParameterNameError) isParseError() {}

func (*MissingColonAfterParameterNameError) IsUserError() {}

func (e *MissingColonAfterParameterNameError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingColonAfterParameterNameError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingColonAfterParameterNameError) Error() string {
	return fmt.Sprintf(
		"expected %s after parameter name, got %s",
		lexer.TokenColon,
		e.GotToken.Type,
	)
}

func (*MissingColonAfterParameterNameError) SecondaryError() string {
	return "function parameters must have a type annotation separated by a colon"
}

func (*MissingColonAfterParameterNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
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
	return fmt.Sprintf(
		"expected a default argument after type annotation, got %s",
		e.GotToken.Type,
	)
}

func (*MissingDefaultArgumentError) SecondaryError() string {
	return "default arguments must be specified with an equals sign (=) followed by the default value"
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
	return "default arguments are only allowed in ResourceDestroyed events, not in functions"
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
	return "missing comma after type parameter"
}

func (*MissingCommaInTypeParameterListError) SecondaryError() string {
	return "add a comma to separate type parameters in the type parameter list"
}

func (e *MissingCommaInTypeParameterListError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Add comma to separate type parameters",
			TextEdits: []ast.TextEdit{
				{
					Insertion: ", ",
					Range: ast.Range{
						StartPos: e.Pos,
						EndPos:   e.Pos,
					},
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
	return fmt.Sprintf(
		"expected type parameter or end of type parameter list, got %s",
		e.GotToken.Type,
	)
}

func (*UnexpectedTokenInTypeParameterListError) SecondaryError() string {
	return "type parameters must be separated by commas, and the list must end with a closing angle bracket (>)"
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

func (*MissingClosingGreaterInTypeParameterListError) isParseError() {}

func (*MissingClosingGreaterInTypeParameterListError) IsUserError() {}

func (e *MissingClosingGreaterInTypeParameterListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingClosingGreaterInTypeParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingClosingGreaterInTypeParameterListError) Error() string {
	return fmt.Sprintf(
		"missing %s at end of type parameter list",
		lexer.TokenGreater,
	)
}

func (*MissingClosingGreaterInTypeParameterListError) SecondaryError() string {
	return "type parameters must be separated by commas, and the list must end with a closing angle bracket (>)"
}

func (*MissingClosingGreaterInTypeParameterListError) DocumentationLink() string {
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

func (*ExpectedCommaOrEndOfTypeParameterListError) isParseError() {}

func (*ExpectedCommaOrEndOfTypeParameterListError) IsUserError() {}

func (e *ExpectedCommaOrEndOfTypeParameterListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *ExpectedCommaOrEndOfTypeParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *ExpectedCommaOrEndOfTypeParameterListError) Error() string {
	return fmt.Sprintf(
		"expected comma or end of type parameter list, got %s",
		e.GotToken.Type,
	)
}

func (*ExpectedCommaOrEndOfTypeParameterListError) SecondaryError() string {
	return "type parameters must be separated by commas, and the list must end with a closing angle bracket (>)"
}

func (*ExpectedCommaOrEndOfTypeParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
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
	return fmt.Sprintf(
		"expected type parameter name, got %s",
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
	return fmt.Sprintf(
		"unexpected second %q block",
		KeywordExecute,
	)
}

func (*DuplicateExecuteBlockError) SecondaryError() string {
	return "transaction declarations can only have one 'execute' block"
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
	return "transaction declarations can only have one 'post' block"
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
		"unexpected identifier, expected keyword %q or %q, got %q",
		KeywordPrepare,
		KeywordExecute,
		e.GotIdentifier,
	)
}

func (*ExpectedPrepareOrExecuteError) SecondaryError() string {
	return "the first block in a transaction declaration must be a 'prepare' or an 'execute' block"
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
		"unexpected identifier, expected keyword %q or %q, got %q",
		KeywordExecute,
		KeywordPost,
		e.GotIdentifier,
	)
}

func (*ExpectedExecuteOrPostError) SecondaryError() string {
	return "transaction declarations may only define an 'execute' or a 'post' block here"
}

func (*ExpectedExecuteOrPostError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
}

// UnexpectedCommaInDictionaryTypeError is reported when a comma is found in a dictionary type.
type UnexpectedCommaInDictionaryTypeError struct {
	Pos ast.Position
}

var _ ParseError = &UnexpectedCommaInDictionaryTypeError{}
var _ errors.UserError = &UnexpectedCommaInDictionaryTypeError{}
var _ errors.SecondaryError = &UnexpectedCommaInDictionaryTypeError{}
var _ errors.HasDocumentationLink = &UnexpectedCommaInDictionaryTypeError{}

func (*UnexpectedCommaInDictionaryTypeError) isParseError() {}

func (*UnexpectedCommaInDictionaryTypeError) IsUserError() {}

func (e *UnexpectedCommaInDictionaryTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedCommaInDictionaryTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedCommaInDictionaryTypeError) Error() string {
	return "unexpected comma in dictionary type"
}

func (*UnexpectedCommaInDictionaryTypeError) SecondaryError() string {
	return "dictionary types use a colon (:) to separate key and value types, not commas (,)"
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
	return "unexpected colon in dictionary type"
}

func (*UnexpectedColonInDictionaryTypeError) SecondaryError() string {
	return "dictionary types use a colon (:) to separate key and value types, both types must be provided ({K:V})"
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

func (*MultipleColonInDictionaryTypeError) isParseError() {}

func (*MultipleColonInDictionaryTypeError) IsUserError() {}

func (e *MultipleColonInDictionaryTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MultipleColonInDictionaryTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MultipleColonInDictionaryTypeError) Error() string {
	return "unexpected colon in dictionary type"
}

func (*MultipleColonInDictionaryTypeError) SecondaryError() string {
	return "dictionary types can only have one colon (:) to separate key and value types"
}

func (*MultipleColonInDictionaryTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/dictionaries"
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
	return "unexpected comma"
}

func (*UnexpectedCommaInTypeAnnotationListError) SecondaryError() string {
	return "a comma is used to separate multiple types, but a type is expected here"
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
	return "missing type annotation after comma"
}

func (*MissingTypeAnnotationAfterCommaError) SecondaryError() string {
	return "after a comma, a type annotation is required to complete the list"
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
	return "unexpected comma in intersection type"
}

func (*UnexpectedCommaInIntersectionTypeError) SecondaryError() string {
	return "intersection types use commas to separate multiple types, check for missing types or remove the comma"
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

func (*UnexpectedColonInIntersectionTypeError) isParseError() {}

func (*UnexpectedColonInIntersectionTypeError) IsUserError() {}

func (e *UnexpectedColonInIntersectionTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnexpectedColonInIntersectionTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*UnexpectedColonInIntersectionTypeError) Error() string {
	return "unexpected colon in intersection type"
}

func (*UnexpectedColonInIntersectionTypeError) SecondaryError() string {
	return "intersection types use commas (,) to separate multiple types, not colons (:)"
}

func (*UnexpectedColonInIntersectionTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
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

// MissingClosingParenInArgumentListError is reported when an argument list is missing a closing parenthesis.
type MissingClosingParenInArgumentListError struct {
	Pos ast.Position
}

var _ ParseError = &MissingClosingParenInArgumentListError{}
var _ errors.UserError = &MissingClosingParenInArgumentListError{}
var _ errors.SecondaryError = &MissingClosingParenInArgumentListError{}
var _ errors.HasDocumentationLink = &MissingClosingParenInArgumentListError{}

func (*MissingClosingParenInArgumentListError) isParseError() {}

func (*MissingClosingParenInArgumentListError) IsUserError() {}

func (e *MissingClosingParenInArgumentListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingClosingParenInArgumentListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingClosingParenInArgumentListError) Error() string {
	return "missing ')' at end of invocation argument list"
}

func (*MissingClosingParenInArgumentListError) SecondaryError() string {
	return "function calls and type instantiations must be properly closed with a closing parenthesis"
}

func (*MissingClosingParenInArgumentListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
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
	return "unexpected comma in argument list"
}

func (*UnexpectedCommaInArgumentListError) SecondaryError() string {
	return "commas are used to separate arguments. Did you add a superfluous comma, or is an argument missing?"
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

func (*MissingCommaInArgumentListError) isParseError() {}

func (*MissingCommaInArgumentListError) IsUserError() {}

func (e *MissingCommaInArgumentListError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingCommaInArgumentListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingCommaInArgumentListError) Error() string {
	return fmt.Sprintf(
		"unexpected argument in argument list (expecting delimiter or end of argument list), got %s",
		e.GotToken.Type,
	)
}

func (*MissingCommaInArgumentListError) SecondaryError() string {
	return "arguments in function calls and type instantiations must be separated by commas"
}

func (*MissingCommaInArgumentListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
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
	return "this token cannot be used to start an expression - check for missing operators, parentheses, or invalid syntax"
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
	return "this token cannot be used as an operator in an expression - check for missing operators, parentheses, or invalid syntax"
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
	return "this token cannot be used to start a type - check for missing operators, parentheses, or invalid syntax"
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
	return "this token cannot be used as an operator in a type - check for missing operators, parentheses, or invalid syntax"
}

func (*UnexpectedTokenInTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax"
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

// InvalidPubSetModifierError is reported when the modifier for `pub` is not `set`.
type InvalidPubSetModifierError struct {
	GotToken lexer.Token
}

var _ ParseError = &InvalidPubSetModifierError{}
var _ errors.UserError = &InvalidPubSetModifierError{}
var _ errors.SecondaryError = &InvalidPubSetModifierError{}
var _ errors.HasDocumentationLink = &InvalidPubSetModifierError{}

func (*InvalidPubSetModifierError) isParseError() {}

func (*InvalidPubSetModifierError) IsUserError() {}

func (e *InvalidPubSetModifierError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *InvalidPubSetModifierError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *InvalidPubSetModifierError) Error() string {
	return fmt.Sprintf("expected keyword %q, got %s", "set", e.GotToken.Type)
}

func (*InvalidPubSetModifierError) SecondaryError() string {
	return "the 'set' keyword is used in access control modifiers to specify settable access"
}

func (*InvalidPubSetModifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

// MissingAccessKeywordError is reported when an access modifier keyword is missing.
type MissingAccessKeywordError struct {
	GotToken lexer.Token
}

var _ ParseError = &MissingAccessKeywordError{}
var _ errors.UserError = &MissingAccessKeywordError{}
var _ errors.SecondaryError = &MissingAccessKeywordError{}
var _ errors.HasDocumentationLink = &MissingAccessKeywordError{}

func (*MissingAccessKeywordError) isParseError() {}

func (*MissingAccessKeywordError) IsUserError() {}

func (e *MissingAccessKeywordError) StartPosition() ast.Position {
	return e.GotToken.StartPos
}

func (e *MissingAccessKeywordError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.GotToken.EndPos
}

func (e *MissingAccessKeywordError) Error() string {
	keywords := common.EnumerateWords(
		[]string{`"all"`, `"account"`, `"contract"`, `"self"`},
		"or",
	)
	return fmt.Sprintf(
		"expected keyword %s, got %s",
		keywords,
		e.GotToken.Type,
	)
}

func (*MissingAccessKeywordError) SecondaryError() string {
	return "access control modifiers must be one of: 'all', 'account', 'contract', or 'self'"
}

func (*MissingAccessKeywordError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
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
	return fmt.Sprintf("expected identifier after start of enum case declaration, got %s", e.GotToken.Type)
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
	return fmt.Sprintf("expected identifier after start of field declaration, got %s", e.GotToken.Type)
}

func (*MissingFieldNameError) SecondaryError() string {
	return "field declarations must have a valid identifier name after the variable kind keyword (let/var)"
}

func (*MissingFieldNameError) DocumentationLink() string {
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
	return "variable declarations must specify how to transfer the value: " +
		"use '=' for copy (struct), '<-' for move (resource), or '<-!' for forced move (resource)"
}

func (*MissingTransferError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/constants-and-variables"
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
	return fmt.Sprintf(
		"unexpected token in import declaration: got %s, expected address, string, or identifier",
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
	return fmt.Sprintf(
		"unexpected token in import declaration: got %s, expected keyword %q or %s",
		e.GotToken.Type,
		KeywordFrom,
		lexer.TokenComma,
	)
}

func (*InvalidImportContinuationError) SecondaryError() string {
	return "after an imported identifier, use either a comma to import more items or the 'from' keyword to specify the import location"
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
	return "import declarations must specify what to import - provide a string literal, hexadecimal address, or identifier"
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
	return fmt.Sprintf(
		"unexpected end in import declaration: expected %s or %s",
		lexer.TokenIdentifier,
		lexer.TokenComma,
	)
}

func (*UnexpectedEOFInImportListError) SecondaryError() string {
	return "import declarations cannot end abruptly - use either an identifier to import or a comma to continue the import list"
}

func (*UnexpectedEOFInImportListError) DocumentationLink() string {
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
	return fmt.Sprintf(
		"expected identifier or keyword %q, got %s",
		KeywordFrom,
		e.GotToken.Type,
	)
}

func (*InvalidTokenInImportListError) SecondaryError() string {
	return "import declarations expect either an identifier to import or the 'from' keyword to specify the import location"
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
	return fmt.Sprintf("expected identifier, got keyword %q", KeywordFrom)
}

func (*InvalidFromKeywordAsIdentifierError) SecondaryError() string {
	return "import declarations expect an identifier to import, not the 'from' keyword in this position"
}

func (*InvalidFromKeywordAsIdentifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
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
	return "expected at least one conformance after :"
}

func (*MissingConformanceError) SecondaryError() string {
	return "provide at least one interface or type to conform to, or remove the colon if no conformances are needed"
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
	return fmt.Sprintf("expected interface name, got keyword %q", "interface")
}

func (*InvalidInterfaceNameError) SecondaryError() string {
	return "interface declarations must have a unique name after the 'interface' keyword"
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
	return fmt.Sprintf("unexpected entitlement separator %s", e.Token.Type.String())
}

func (*InvalidEntitlementSeparatorError) SecondaryError() string {
	return "use a comma (,) for conjunctive entitlements or a vertical bar (|) for disjunctive entitlements"
}

func (*InvalidEntitlementSeparatorError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlements"
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
	tokenType := e.GotToken.Type
	if tokenType == lexer.TokenEOF {
		return "expected member name"
	}
	return fmt.Sprintf("expected member name, got %s", tokenType)
}

func (*MemberAccessMissingNameError) SecondaryError() string {
	return "after a dot (.), you must provide a valid identifier for the member name"
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
	return fmt.Sprintf("invalid whitespace after %s", e.OperatorTokenType)
}

func (e *WhitespaceAfterMemberAccessError) SecondaryError() string {
	return fmt.Sprintf("remove the space between %s and the member name", e.OperatorTokenType)
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
	return e.Pos.Shifted(memoryGauge, len(KeywordNative)-1)
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
		"the `native` modifier can only be used on on fields and functions, "+
			"not on %s declarations",
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
	return fmt.Sprintf("expected nominal type, got non-nominal type `%s`", e.Type)
}

func (*NonNominalTypeError) SecondaryError() string {
	return "expected a nominal type (like a struct, resource, or interface name)"
}

func (*NonNominalTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/"
}
