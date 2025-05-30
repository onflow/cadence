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
	Message string
	Pos     ast.Position
}

func NewSyntaxError(pos ast.Position, message string, params ...any) *SyntaxError {
	return &SyntaxError{
		Pos:     pos,
		Message: fmt.Sprintf(message, params...),
	}
}

var _ ParseError = &SyntaxError{}
var _ errors.UserError = &SyntaxError{}

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

// SyntaxErrorWithSuggestedFix

type SyntaxErrorWithSuggestedReplacement struct {
	Message      string
	SuggestedFix string
	ast.Range
}

var _ errors.HasSuggestedFixes[ast.TextEdit] = &SyntaxErrorWithSuggestedReplacement{}

func NewSyntaxErrorWithSuggestedReplacement(r ast.Range, message string, suggestedFix string) *SyntaxErrorWithSuggestedReplacement {
	return &SyntaxErrorWithSuggestedReplacement{
		Range:        r,
		Message:      message,
		SuggestedFix: suggestedFix,
	}
}

var _ ParseError = &SyntaxErrorWithSuggestedReplacement{}
var _ errors.UserError = &SyntaxErrorWithSuggestedReplacement{}

func (*SyntaxErrorWithSuggestedReplacement) isParseError() {}

func (*SyntaxErrorWithSuggestedReplacement) IsUserError() {}
func (e *SyntaxErrorWithSuggestedReplacement) Error() string {
	return e.Message
}

func (e *SyntaxErrorWithSuggestedReplacement) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: fmt.Sprintf("replace with %s", e.SuggestedFix),
			TextEdits: []ast.TextEdit{
				{
					Replacement: e.SuggestedFix,
					Range:       e.Range,
				},
			},
		},
	}
}

// JuxtaposedUnaryOperatorsError

type JuxtaposedUnaryOperatorsError struct {
	Pos ast.Position
}

var _ ParseError = &JuxtaposedUnaryOperatorsError{}
var _ errors.UserError = &JuxtaposedUnaryOperatorsError{}

func (*JuxtaposedUnaryOperatorsError) isParseError() {}

func (*JuxtaposedUnaryOperatorsError) IsUserError() {}

func (e *JuxtaposedUnaryOperatorsError) StartPosition() ast.Position {
	return e.Pos
}

func (e *JuxtaposedUnaryOperatorsError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *JuxtaposedUnaryOperatorsError) Error() string {
	return "unary operators must not be juxtaposed; parenthesize inner expression"
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

// ExpressionDepthLimitReachedError is reported when the expression depth limit was reached
type ExpressionDepthLimitReachedError struct {
	Pos ast.Position
}

var _ ParseError = ExpressionDepthLimitReachedError{}
var _ errors.UserError = ExpressionDepthLimitReachedError{}

func (ExpressionDepthLimitReachedError) isParseError() {}

func (ExpressionDepthLimitReachedError) IsUserError() {}

func (e ExpressionDepthLimitReachedError) Error() string {
	return fmt.Sprintf(
		"program too complex, reached max expression depth limit %d",
		expressionDepthLimit,
	)
}

func (e ExpressionDepthLimitReachedError) StartPosition() ast.Position {
	return e.Pos
}

func (e ExpressionDepthLimitReachedError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

// TypeDepthLimitReachedError is reported when the type depth limit was reached
//

type TypeDepthLimitReachedError struct {
	Pos ast.Position
}

var _ ParseError = TypeDepthLimitReachedError{}
var _ errors.UserError = TypeDepthLimitReachedError{}

func (TypeDepthLimitReachedError) isParseError() {}

func (TypeDepthLimitReachedError) IsUserError() {}

func (e TypeDepthLimitReachedError) Error() string {
	return fmt.Sprintf(
		"program too complex, reached max type depth limit %d",
		typeDepthLimit,
	)
}

func (e TypeDepthLimitReachedError) StartPosition() ast.Position {
	return e.Pos
}

func (e TypeDepthLimitReachedError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

// MissingCommaInParameterListError

type MissingCommaInParameterListError struct {
	Pos ast.Position
}

var _ ParseError = &MissingCommaInParameterListError{}
var _ errors.UserError = &MissingCommaInParameterListError{}

func (*MissingCommaInParameterListError) isParseError() {}

func (*MissingCommaInParameterListError) IsUserError() {}

func (e *MissingCommaInParameterListError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingCommaInParameterListError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *MissingCommaInParameterListError) Error() string {
	return "missing comma after parameter"
}

// CustomDestructorError

type CustomDestructorError struct {
	Pos ast.Position
}

var _ ParseError = &CustomDestructorError{}
var _ errors.UserError = &CustomDestructorError{}

func (*CustomDestructorError) isParseError() {}

func (*CustomDestructorError) IsUserError() {}

func (e *CustomDestructorError) StartPosition() ast.Position {
	return e.Pos
}

func (e *CustomDestructorError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *CustomDestructorError) Error() string {
	return "custom destructor definitions are no longer permitted"
}

func (e *CustomDestructorError) SecondaryError() string {
	return "remove the destructor definition"
}

// RestrictedTypeError

type RestrictedTypeError struct {
	ast.Range
}

var _ ParseError = &CustomDestructorError{}
var _ errors.UserError = &CustomDestructorError{}

func (*RestrictedTypeError) isParseError() {}

func (*RestrictedTypeError) IsUserError() {}

func (e *RestrictedTypeError) Error() string {
	return "restricted types have been removed; replace with the concrete type or an equivalent intersection type"
}
