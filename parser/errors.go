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
