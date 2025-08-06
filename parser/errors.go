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

// SyntaxErrorWithSuggestedFix

type SyntaxErrorWithSuggestedReplacement struct {
	Message       string
	Replacement   string
	Secondary     string
	Migration     string
	Documentation string
	ast.Range
}

var _ errors.HasSuggestedFixes[ast.TextEdit] = &SyntaxErrorWithSuggestedReplacement{}

func NewSyntaxErrorWithSuggestedReplacement(r ast.Range, message string, suggestedFix string) *SyntaxErrorWithSuggestedReplacement {
	return &SyntaxErrorWithSuggestedReplacement{
		Range:       r,
		Message:     message,
		Replacement: suggestedFix,
	}
}

var _ ParseError = &SyntaxErrorWithSuggestedReplacement{}
var _ errors.UserError = &SyntaxErrorWithSuggestedReplacement{}
var _ errors.SecondaryError = &SyntaxErrorWithSuggestedReplacement{}
var _ errors.HasDocumentationLink = &SyntaxErrorWithSuggestedReplacement{}
var _ errors.HasMigrationNote = &SyntaxErrorWithSuggestedReplacement{}

func (*SyntaxErrorWithSuggestedReplacement) isParseError() {}

func (*SyntaxErrorWithSuggestedReplacement) IsUserError() {}

func (e *SyntaxErrorWithSuggestedReplacement) Error() string {
	return e.Message
}

func (e *SyntaxErrorWithSuggestedReplacement) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: fmt.Sprintf("replace with %s", e.Replacement),
			TextEdits: []ast.TextEdit{
				{
					Replacement: e.Replacement,
					Range:       e.Range,
				},
			},
		},
	}
}

func (e *SyntaxErrorWithSuggestedReplacement) SecondaryError() string {
	return e.Secondary
}

func (e *SyntaxErrorWithSuggestedReplacement) DocumentationLink() string {
	return e.Documentation
}

func (e *SyntaxErrorWithSuggestedReplacement) MigrationNote() string {
	return e.Migration
}

// Helper methods to set additional error information

func (e *SyntaxErrorWithSuggestedReplacement) WithSecondary(secondary string) *SyntaxErrorWithSuggestedReplacement {
	e.Secondary = secondary
	return e
}

func (e *SyntaxErrorWithSuggestedReplacement) WithMigration(migration string) *SyntaxErrorWithSuggestedReplacement {
	e.Migration = migration
	return e
}

func (e *SyntaxErrorWithSuggestedReplacement) WithDocumentation(documentation string) *SyntaxErrorWithSuggestedReplacement {
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

// InvalidStaticModifierError

type InvalidStaticModifierError struct {
	Pos             ast.Position
	DeclarationKind common.DeclarationKind
}

var _ ParseError = &InvalidStaticModifierError{}
var _ errors.UserError = &InvalidStaticModifierError{}
var _ errors.SecondaryError = &InvalidStaticModifierError{}

func (*InvalidStaticModifierError) isParseError() {}

func (*InvalidStaticModifierError) IsUserError() {}

func (e *InvalidStaticModifierError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidStaticModifierError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
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

func (e *InvalidNativeModifierError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *InvalidNativeModifierError) Error() string {
	return fmt.Sprintf(
		"invalid `native` modifier for %s",
		e.DeclarationKind.Name(),
	)
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
	return "https://cadence-lang.org/docs/language/types"
}
