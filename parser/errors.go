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
	Pos   ast.Position
	Range ast.Range
}

var _ ParseError = &JuxtaposedUnaryOperatorsError{}
var _ errors.UserError = &JuxtaposedUnaryOperatorsError{}
var _ errors.SecondaryError = &JuxtaposedUnaryOperatorsError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &JuxtaposedUnaryOperatorsError{}
var _ errors.HasDocumentationLink = &JuxtaposedUnaryOperatorsError{}

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

func (e *JuxtaposedUnaryOperatorsError) SecondaryError() string {
	return "add parentheses around the inner expression to clarify operator precedence"
}

func (e *JuxtaposedUnaryOperatorsError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	// For juxtaposed unary operators, we suggest adding parentheses
	// around the inner expression to clarify precedence
	if e.Range.StartPos.Offset < e.Range.EndPos.Offset && e.Range.EndPos.Offset <= len(code) {
		innerExpression := code[e.Range.StartPos.Offset:e.Range.EndPos.Offset]
		return []errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Add parentheses to clarify operator precedence",
				TextEdits: []ast.TextEdit{
					{
						Replacement: fmt.Sprintf("(%s)", innerExpression),
						Range:       e.Range,
					},
				},
			},
		}
	}
	return nil
}

func (e *JuxtaposedUnaryOperatorsError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/prescedence-associativity"
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
			"Invalid integer literal `%s`: %s",
			e.Literal,
			e.InvalidIntegerLiteralKind.Description(),
		)
	}

	return fmt.Sprintf(
		"Invalid %s integer literal `%s`: %s",
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

func (e *InvalidIntegerLiteralError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/booleans-numlits-ints"
}

func (e *InvalidIntegerLiteralError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	switch e.InvalidIntegerLiteralKind {
	case InvalidNumberLiteralKindLeadingUnderscore:
		// Remove leading underscore
		if len(e.Literal) > 1 && e.Literal[0] == '_' {
			return []errors.SuggestedFix[ast.TextEdit]{
				{
					Message: "Remove leading underscore",
					TextEdits: []ast.TextEdit{
						{
							Replacement: e.Literal[1:],
							Range:       e.Range,
						},
					},
				},
			}
		}
	case InvalidNumberLiteralKindTrailingUnderscore:
		// Remove trailing underscore
		if len(e.Literal) > 1 && e.Literal[len(e.Literal)-1] == '_' {
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
		// Add a 0 to make it a valid number
		return []errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Add a digit to make this a valid number",
				TextEdits: []ast.TextEdit{
					{
						Replacement: e.Literal + "0",
						Range:       e.Range,
					},
				},
			},
		}
	}
	return nil
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

func (e ExpressionDepthLimitReachedError) Error() string {
	return fmt.Sprintf(
		"expression too deeply nested, exceeded depth limit of %d",
		expressionDepthLimit,
	)
}

func (e ExpressionDepthLimitReachedError) SecondaryError() string {
	return "Consider breaking the expression into smaller parts or using intermediate variables"
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

func (e TypeDepthLimitReachedError) Error() string {
	return fmt.Sprintf(
		"type too deeply nested, exceeded depth limit of %d",
		typeDepthLimit,
	)
}

func (e TypeDepthLimitReachedError) SecondaryError() string {
	return "Consider breaking complex nested types into simpler components or using intermediate variables"
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

func (e *MissingCommaInParameterListError) Error() string {
	return "missing comma after parameter"
}

func (e *MissingCommaInParameterListError) SecondaryError() string {
	return "add a comma to separate parameters in the parameter list"
}

func (e *MissingCommaInParameterListError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	// Insert a comma at the current position
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

func (e *MissingCommaInParameterListError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions#function-declarations"
}

// CustomDestructorError

type CustomDestructorError struct {
	Pos             ast.Position
	DestructorRange ast.Range // Range of the entire destructor, used for suggested fix
}

var _ ParseError = &CustomDestructorError{}
var _ errors.UserError = &CustomDestructorError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &CustomDestructorError{}
var _ errors.HasDocumentationLink = &CustomDestructorError{}

func (*CustomDestructorError) isParseError() {}

func (*CustomDestructorError) IsUserError() {}

func (e *CustomDestructorError) StartPosition() ast.Position {
	return e.Pos
}

func (e *CustomDestructorError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *CustomDestructorError) Error() string {
	return "custom destructor definitions are no longer permitted in Cadence 1.0+"
}

func (e *CustomDestructorError) SecondaryError() string {
	return "remove the destructor definition"
}

func (e *CustomDestructorError) MigrationNote() string {
	return "This is pre-Cadence 1.0 syntax. Support for custom destructors was removed. Custom cleanup logic should be moved to a separate function called before destruction."
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

func (e *CustomDestructorError) DocumentationLink() string {
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

func (*RestrictedTypeError) isParseError() {}

func (*RestrictedTypeError) IsUserError() {}


func (e *RestrictedTypeError) Error() string {
	return "restricted types have been removed in Cadence 1.0+"
}

func (e *RestrictedTypeError) SecondaryError() string {
	return "replace with the concrete type or an equivalent intersection type"
}

func (e *RestrictedTypeError) MigrationNote() string {
	return "This is pre-Cadence 1.0 syntax. Restricted types like `T{}` have been replaced with intersection types like `{T}`."
}

func (e *RestrictedTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/cadence-migration-guide/improvements#-motivation-12"
}
