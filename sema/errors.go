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

package sema

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/texttheater/golang-levenshtein/levenshtein"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/pretty"
)

func ErrorMessageExpectedActualTypes(
	expectedType Type,
	actualType Type,
) (
	expected string,
	actual string,
) {
	expected = expectedType.QualifiedString()
	actual = actualType.QualifiedString()

	if expected == actual {
		expected = string(expectedType.ID())
		actual = string(actualType.ID())
	}

	return
}

// unsupportedOperation

type unsupportedOperation struct {
	kind      common.OperationKind
	operation ast.Operation
	ast.Range
}

func (e *unsupportedOperation) Error() string {
	return fmt.Sprintf(
		"cannot check unsupported %s operation: `%s`",
		e.kind.Name(),
		e.operation.Symbol(),
	)
}

// InvalidPragmaError

type InvalidPragmaError struct {
	Message string
	ast.Range
}

var _ SemanticError = &InvalidPragmaError{}
var _ errors.UserError = &InvalidPragmaError{}
var _ errors.SecondaryError = &InvalidPragmaError{}

func (*InvalidPragmaError) isSemanticError() {}

func (*InvalidPragmaError) IsUserError() {}

func (e *InvalidPragmaError) Error() string {
	return "invalid pragma"
}

func (e *InvalidPragmaError) SecondaryError() string {
	return e.Message
}

// CheckerError

type CheckerError struct {
	Location common.Location
	Codes    map[common.Location][]byte
	Errors   []error
}

var _ errors.UserError = CheckerError{}
var _ errors.ParentError = CheckerError{}

func (CheckerError) IsUserError() {}

func (e CheckerError) Error() string {
	var sb strings.Builder
	sb.WriteString("Checking failed:\n")
	codes := e.Codes
	if codes == nil {
		codes = map[common.Location][]byte{}
	}
	printErr := pretty.NewErrorPrettyPrinter(&sb, false).
		PrettyPrintError(e, e.Location, codes)
	if printErr != nil {
		panic(printErr)
	}
	sb.WriteString(errors.ErrorPrompt)
	return sb.String()
}

func (e CheckerError) ChildErrors() []error {
	return e.Errors
}

func (e CheckerError) Unwrap() []error {
	return e.Errors
}

func (e CheckerError) ImportLocation() common.Location {
	return e.Location
}

// SemanticError

type SemanticError interface {
	errors.UserError
	ast.HasPosition
	isSemanticError()
}

// RedeclarationError

type RedeclarationError struct {
	PreviousPos *ast.Position
	Name        string
	Pos         ast.Position
	Kind        common.DeclarationKind
}

var _ SemanticError = &RedeclarationError{}
var _ errors.UserError = &RedeclarationError{}

func (*RedeclarationError) isSemanticError() {}

func (*RedeclarationError) IsUserError() {}

func (e *RedeclarationError) Error() string {
	return fmt.Sprintf(
		"cannot redeclare %s: `%s` is already declared",
		e.Kind.Name(),
		e.Name,
	)
}

func (e *RedeclarationError) StartPosition() ast.Position {
	return e.Pos
}

func (e *RedeclarationError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

func (e *RedeclarationError) ErrorNotes() []errors.ErrorNote {
	if e.PreviousPos == nil || e.PreviousPos.Line < 1 {
		return nil
	}

	previousStartPos := *e.PreviousPos
	length := len(e.Name)
	previousEndPos := previousStartPos.Shifted(nil, length-1)

	return []errors.ErrorNote{
		&RedeclarationNote{
			Range: ast.NewUnmeteredRange(
				previousStartPos,
				previousEndPos,
			),
		},
	}
}

// RedeclarationNote

type RedeclarationNote struct {
	ast.Range
}

func (n RedeclarationNote) Message() string {
	return "previously declared here"
}

// NotDeclaredError

type NotDeclaredError struct {
	Expression   *ast.IdentifierExpression
	Name         string
	Pos          ast.Position
	ExpectedKind common.DeclarationKind
}

var _ SemanticError = &NotDeclaredError{}
var _ errors.UserError = &NotDeclaredError{}
var _ errors.SecondaryError = &NotDeclaredError{}

func (*NotDeclaredError) isSemanticError() {}

func (*NotDeclaredError) IsUserError() {}

func (e *NotDeclaredError) Error() string {
	return fmt.Sprintf(
		"cannot find %s in this scope: `%s`",
		e.ExpectedKind.Name(),
		e.Name,
	)
}

func (e *NotDeclaredError) SecondaryError() string {
	return "not found in this scope"
}

func (e *NotDeclaredError) StartPosition() ast.Position {
	return e.Pos
}

func (e *NotDeclaredError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// AssignmentToConstantError

type AssignmentToConstantError struct {
	Name string
	ast.Range
}

var _ SemanticError = &AssignmentToConstantError{}
var _ errors.UserError = &AssignmentToConstantError{}
var _ errors.SecondaryError = &AssignmentToConstantError{}

func (*AssignmentToConstantError) isSemanticError() {}

func (*AssignmentToConstantError) IsUserError() {}

func (e *AssignmentToConstantError) Error() string {
	return fmt.Sprintf("cannot assign to constant: `%s`", e.Name)
}

func (e *AssignmentToConstantError) SecondaryError() string {
	return fmt.Sprintf("consider changing the declaration of `%s` to be `var`", e.Name)
}

// TypeMismatchError

type TypeMismatchError struct {
	ExpectedType Type
	ActualType   Type
	Expression   ast.Expression
	ast.Range
}

var _ SemanticError = &TypeMismatchError{}
var _ errors.UserError = &TypeMismatchError{}
var _ errors.SecondaryError = &TypeMismatchError{}

func (*TypeMismatchError) isSemanticError() {}

func (*TypeMismatchError) IsUserError() {}

func (e *TypeMismatchError) Error() string {
	return "mismatched types"
}

func (e *TypeMismatchError) SecondaryError() string {
	expected, actual := ErrorMessageExpectedActualTypes(
		e.ExpectedType,
		e.ActualType,
	)

	return fmt.Sprintf(
		"expected `%s`, got `%s`",
		expected,
		actual,
	)
}

// TypeMismatchWithDescriptionError

type TypeMismatchWithDescriptionError struct {
	ActualType              Type
	ExpectedTypeDescription string
	ast.Range
}

var _ SemanticError = &TypeMismatchWithDescriptionError{}
var _ errors.UserError = &TypeMismatchWithDescriptionError{}
var _ errors.SecondaryError = &TypeMismatchWithDescriptionError{}

func (*TypeMismatchWithDescriptionError) isSemanticError() {}

func (*TypeMismatchWithDescriptionError) IsUserError() {}

func (e *TypeMismatchWithDescriptionError) Error() string {
	return "mismatched types"
}

func (e *TypeMismatchWithDescriptionError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %s, got `%s`",
		e.ExpectedTypeDescription,
		e.ActualType.QualifiedString(),
	)
}

// NotIndexableTypeError

type NotIndexableTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &NotIndexableTypeError{}
var _ errors.UserError = &NotIndexableTypeError{}

func (*NotIndexableTypeError) isSemanticError() {}

func (*NotIndexableTypeError) IsUserError() {}

func (e *NotIndexableTypeError) Error() string {
	return fmt.Sprintf(
		"cannot index into value which has type: `%s`",
		e.Type.QualifiedString(),
	)
}

// NotIndexingAssignableTypeError

type NotIndexingAssignableTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &NotIndexingAssignableTypeError{}
var _ errors.UserError = &NotIndexingAssignableTypeError{}

func (*NotIndexingAssignableTypeError) isSemanticError() {}

func (*NotIndexingAssignableTypeError) IsUserError() {}

func (e *NotIndexingAssignableTypeError) Error() string {
	return fmt.Sprintf(
		"cannot assign into value which has type: `%s`",
		e.Type.QualifiedString(),
	)
}

// NotEquatableTypeError

type NotEquatableTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &NotEquatableTypeError{}
var _ errors.UserError = &NotEquatableTypeError{}

func (*NotEquatableTypeError) isSemanticError() {}

func (*NotEquatableTypeError) IsUserError() {}

func (e *NotEquatableTypeError) Error() string {
	return fmt.Sprintf(
		"cannot compare value which has type: `%s`",
		e.Type.QualifiedString(),
	)
}

// NotCallableError

type NotCallableError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &NotCallableError{}
var _ errors.UserError = &NotCallableError{}

func (*NotCallableError) isSemanticError() {}

func (*NotCallableError) IsUserError() {}

func (e *NotCallableError) Error() string {
	return fmt.Sprintf("cannot call type: `%s`",
		e.Type.QualifiedString(),
	)
}

// InsufficientArgumentsError

type InsufficientArgumentsError struct {
	MinCount    int
	ActualCount int
	ast.Range
}

var _ SemanticError = &InsufficientArgumentsError{}
var _ errors.UserError = &InsufficientArgumentsError{}
var _ errors.SecondaryError = &InsufficientArgumentsError{}

func (*InsufficientArgumentsError) isSemanticError() {}

func (*InsufficientArgumentsError) IsUserError() {}

func (e *InsufficientArgumentsError) Error() string {
	return "too few arguments"
}

func (e *InsufficientArgumentsError) SecondaryError() string {
	return fmt.Sprintf(
		"expected at least %d, got %d",
		e.MinCount,
		e.ActualCount,
	)
}

// ExcessiveArgumentsError

type ExcessiveArgumentsError struct {
	MaxCount    int
	ActualCount int
	ast.Range
}

var _ SemanticError = &ExcessiveArgumentsError{}
var _ errors.UserError = &ExcessiveArgumentsError{}
var _ errors.SecondaryError = &ExcessiveArgumentsError{}

func (*ExcessiveArgumentsError) isSemanticError() {}

func (*ExcessiveArgumentsError) IsUserError() {}

func (e *ExcessiveArgumentsError) Error() string {
	return "too many arguments"
}

func (e *ExcessiveArgumentsError) SecondaryError() string {
	return fmt.Sprintf(
		"expected up to %d, got %d",
		e.MaxCount,
		e.ActualCount,
	)
}

// MissingArgumentLabelError

// TODO: suggest adding argument label

type MissingArgumentLabelError struct {
	ExpectedArgumentLabel string
	ast.Range
}

var _ SemanticError = &MissingArgumentLabelError{}
var _ errors.UserError = &MissingArgumentLabelError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingArgumentLabelError{}

func (*MissingArgumentLabelError) isSemanticError() {}

func (*MissingArgumentLabelError) IsUserError() {}

func (e *MissingArgumentLabelError) Error() string {
	return fmt.Sprintf(
		"missing argument label: `%s`",
		e.ExpectedArgumentLabel,
	)
}

func (e *MissingArgumentLabelError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "insert argument label",
			TextEdits: []ast.TextEdit{
				{
					Insertion: fmt.Sprintf("%s: ", e.ExpectedArgumentLabel),
					Range: ast.NewUnmeteredRange(
						e.StartPos,
						e.StartPos,
					),
				},
			},
		},
	}
}

// IncorrectArgumentLabelError

type IncorrectArgumentLabelError struct {
	ExpectedArgumentLabel string
	ActualArgumentLabel   string
	ast.Range
}

var _ SemanticError = &IncorrectArgumentLabelError{}
var _ errors.UserError = &IncorrectArgumentLabelError{}
var _ errors.SecondaryError = &IncorrectArgumentLabelError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &IncorrectArgumentLabelError{}

func (*IncorrectArgumentLabelError) isSemanticError() {}

func (*IncorrectArgumentLabelError) IsUserError() {}

func (e *IncorrectArgumentLabelError) Error() string {
	return "incorrect argument label"
}

func (e *IncorrectArgumentLabelError) SecondaryError() string {
	expected := "no label"
	if e.ExpectedArgumentLabel != "" {
		expected = fmt.Sprintf("`%s`", e.ExpectedArgumentLabel)
	}
	return fmt.Sprintf(
		"expected %s, got `%s`",
		expected,
		e.ActualArgumentLabel,
	)
}

func (e *IncorrectArgumentLabelError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	if len(e.ExpectedArgumentLabel) > 0 {
		return []errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "replace argument label",
				TextEdits: []ast.TextEdit{
					{
						Replacement: fmt.Sprintf("%s:", e.ExpectedArgumentLabel),
						Range:       e.Range,
					},
				},
			},
		}
	} else {
		endPos := e.Range.EndPos

		var whitespaceSuffixLength int
		for offset := endPos.Offset + 1; offset < len(code); offset++ {
			if code[offset] == ' ' {
				whitespaceSuffixLength++
			} else {
				break
			}
		}

		adjustedEndPos := endPos.Shifted(nil, whitespaceSuffixLength)

		return []errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "remove argument label",
				TextEdits: []ast.TextEdit{
					{
						Replacement: "",
						Range: ast.Range{
							StartPos: e.Range.StartPos,
							EndPos:   adjustedEndPos,
						},
					},
				},
			},
		}
	}
}

// InvalidUnaryOperandError

type InvalidUnaryOperandError struct {
	ExpectedType            Type
	ExpectedTypeDescription string
	ActualType              Type
	ast.Range
	Operation ast.Operation
}

var _ SemanticError = &InvalidUnaryOperandError{}
var _ errors.UserError = &InvalidUnaryOperandError{}
var _ errors.SecondaryError = &InvalidUnaryOperandError{}

func (*InvalidUnaryOperandError) isSemanticError() {}

func (*InvalidUnaryOperandError) IsUserError() {}

func (e *InvalidUnaryOperandError) Error() string {
	return fmt.Sprintf(
		"cannot apply unary operation %s to type",
		e.Operation.Symbol(),
	)
}

func (e *InvalidUnaryOperandError) SecondaryError() string {
	expectedType := e.ExpectedType
	if expectedType != nil {
		expected, actual := ErrorMessageExpectedActualTypes(
			e.ExpectedType,
			e.ActualType,
		)

		return fmt.Sprintf(
			"expected `%s`, got `%s`",
			expected,
			actual,
		)
	} else {
		return fmt.Sprintf(
			"expected %s, got `%s`",
			e.ExpectedTypeDescription,
			e.ActualType.QualifiedString(),
		)
	}
}

// InvalidBinaryOperandError

type InvalidBinaryOperandError struct {
	ExpectedType Type
	ActualType   Type
	ast.Range
	Operation ast.Operation
	Side      common.OperandSide
}

var _ SemanticError = &InvalidBinaryOperandError{}
var _ errors.UserError = &InvalidBinaryOperandError{}
var _ errors.SecondaryError = &InvalidBinaryOperandError{}

func (*InvalidBinaryOperandError) isSemanticError() {}

func (*InvalidBinaryOperandError) IsUserError() {}

func (e *InvalidBinaryOperandError) Error() string {
	return fmt.Sprintf(
		"cannot apply binary operation %s to %s-hand type",
		e.Operation.Symbol(),
		e.Side.Name(),
	)
}

func (e *InvalidBinaryOperandError) SecondaryError() string {
	expected, actual := ErrorMessageExpectedActualTypes(
		e.ExpectedType,
		e.ActualType,
	)

	return fmt.Sprintf(
		"expected `%s`, got `%s`",
		expected,
		actual,
	)
}

// InvalidBinaryOperandsError

type InvalidBinaryOperandsError struct {
	LeftType  Type
	RightType Type
	ast.Range
	Operation ast.Operation
}

var _ SemanticError = &InvalidBinaryOperandsError{}
var _ errors.UserError = &InvalidBinaryOperandsError{}

func (*InvalidBinaryOperandsError) isSemanticError() {}

func (*InvalidBinaryOperandsError) IsUserError() {}

func (e *InvalidBinaryOperandsError) Error() string {
	return fmt.Sprintf(
		"cannot apply binary operation %s to types: `%s`, `%s`",
		e.Operation.Symbol(),
		e.LeftType.QualifiedString(),
		e.RightType.QualifiedString(),
	)
}

// InvalidNilCoalescingRightResourceOperandError

type InvalidNilCoalescingRightResourceOperandError struct {
	ast.Range
}

func (e *InvalidNilCoalescingRightResourceOperandError) Error() string {
	return "nil-coalescing with right-hand resource is not supported at the moment"
}

// InvalidConditionalResourceOperandError

type InvalidConditionalResourceOperandError struct {
	ast.Range
}

func (e *InvalidConditionalResourceOperandError) Error() string {
	return "conditional with resource is not supported at the moment"
}

// ControlStatementError

type ControlStatementError struct {
	ControlStatement common.ControlStatement
	ast.Range
}

var _ SemanticError = &ControlStatementError{}
var _ errors.UserError = &ControlStatementError{}
var _ errors.SecondaryError = &ControlStatementError{}

func (*ControlStatementError) isSemanticError() {}

func (*ControlStatementError) IsUserError() {}

func (e *ControlStatementError) Error() string {
	return fmt.Sprintf(
		"invalid control statement: `%s`",
		e.ControlStatement.Symbol(),
	)
}

func (e *ControlStatementError) SecondaryError() string {
	validLocation := "a loop "
	if e.ControlStatement == common.ControlStatementBreak {
		validLocation += " or switch statement"
	}
	return fmt.Sprintf(
		"`%s` can only be used within %s body",
		e.ControlStatement.Symbol(),
		validLocation,
	)
}

// InvalidAccessModifierError

type InvalidAccessModifierError struct {
	Explanation     string
	Pos             ast.Position
	DeclarationKind common.DeclarationKind
	Access          Access
}

var _ SemanticError = &InvalidAccessModifierError{}
var _ errors.UserError = &InvalidAccessModifierError{}

func (*InvalidAccessModifierError) isSemanticError() {}

func (*InvalidAccessModifierError) IsUserError() {}

func (e *InvalidAccessModifierError) Error() string {
	var explanation string
	if e.Explanation != "" {
		explanation = fmt.Sprintf(". %s", e.Explanation)
	}

	if e.Access.Equal(PrimitiveAccess(ast.AccessNotSpecified)) {
		return fmt.Sprintf(
			"invalid effective access modifier for %s%s",
			e.DeclarationKind.Name(),
			explanation,
		)
	} else {
		return fmt.Sprintf(
			"invalid access modifier for %s: `%s`%s",
			e.DeclarationKind.Name(),
			e.Access.String(),
			explanation,
		)
	}
}

func (e *InvalidAccessModifierError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidAccessModifierError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	if e.Access.Equal(PrimitiveAccess(ast.AccessNotSpecified)) {
		return e.Pos
	}

	length := len(e.Access.String())
	return e.Pos.Shifted(memoryGauge, length-1)
}

// MissingAccessModifierError

type MissingAccessModifierError struct {
	Explanation     string
	Pos             ast.Position
	DeclarationKind common.DeclarationKind
}

var _ errors.UserError = &MissingAccessModifierError{}
var _ SemanticError = &MissingAccessModifierError{}

func (*MissingAccessModifierError) isSemanticError() {}

func (*MissingAccessModifierError) IsUserError() {}

func (e *MissingAccessModifierError) Error() string {
	var explanation string
	if e.Explanation != "" {
		explanation = fmt.Sprintf(". %s", e.Explanation)
	}

	return fmt.Sprintf(
		"missing access modifier for %s%s",
		e.DeclarationKind.Name(),
		explanation,
	)
}

func (e *MissingAccessModifierError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingAccessModifierError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidStaticModifierError

type InvalidStaticModifierError struct {
	ast.Range
}

var _ SemanticError = &InvalidStaticModifierError{}
var _ errors.UserError = &InvalidStaticModifierError{}

func (*InvalidStaticModifierError) isSemanticError() {}

func (*InvalidStaticModifierError) IsUserError() {}

func (e *InvalidStaticModifierError) Error() string {
	return "invalid static modifier for declaration"
}

// InvalidNativeModifierError

type InvalidNativeModifierError struct {
	ast.Range
}

var _ SemanticError = &InvalidNativeModifierError{}
var _ errors.UserError = &InvalidNativeModifierError{}

func (*InvalidNativeModifierError) isSemanticError() {}

func (*InvalidNativeModifierError) IsUserError() {}

func (e *InvalidNativeModifierError) Error() string {
	return "invalid native modifier for declaration"
}

// NativeFunctionWithImplementationError

type NativeFunctionWithImplementationError struct {
	ast.Range
}

var _ SemanticError = &NativeFunctionWithImplementationError{}
var _ errors.UserError = &NativeFunctionWithImplementationError{}

func (*NativeFunctionWithImplementationError) isSemanticError() {}

func (*NativeFunctionWithImplementationError) IsUserError() {}

func (e *NativeFunctionWithImplementationError) Error() string {
	return "native function must not have an implementation"
}

// InvalidNameError

type InvalidNameError struct {
	Name string
	Pos  ast.Position
}

var _ SemanticError = &InvalidNameError{}
var _ errors.UserError = &InvalidNameError{}

func (*InvalidNameError) isSemanticError() {}

func (*InvalidNameError) IsUserError() {}

func (e *InvalidNameError) Error() string {
	return fmt.Sprintf("invalid name: `%s`", e.Name)
}

func (e *InvalidNameError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidNameError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// UnknownSpecialFunctionError

type UnknownSpecialFunctionError struct {
	Pos ast.Position
}

var _ SemanticError = &UnknownSpecialFunctionError{}
var _ errors.UserError = &UnknownSpecialFunctionError{}

func (*UnknownSpecialFunctionError) isSemanticError() {}

func (*UnknownSpecialFunctionError) IsUserError() {}

func (e *UnknownSpecialFunctionError) Error() string {
	return "unknown special function. did you mean `init` or forget the `fun` keyword?"
}

func (e *UnknownSpecialFunctionError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnknownSpecialFunctionError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidVariableKindError

type InvalidVariableKindError struct {
	Kind ast.VariableKind
	ast.Range
}

var _ SemanticError = &InvalidVariableKindError{}
var _ errors.UserError = &InvalidVariableKindError{}

func (*InvalidVariableKindError) isSemanticError() {}

func (*InvalidVariableKindError) IsUserError() {}

func (e *InvalidVariableKindError) Error() string {
	if e.Kind == ast.VariableKindNotSpecified {
		return "missing variable kind"
	}
	return fmt.Sprintf("invalid variable kind: `%s`", e.Kind.Name())
}

// InvalidDeclarationError

type InvalidDeclarationError struct {
	Identifier string
	Kind       common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidDeclarationError{}
var _ errors.UserError = &InvalidDeclarationError{}

func (*InvalidDeclarationError) isSemanticError() {}

func (*InvalidDeclarationError) IsUserError() {}

func (e *InvalidDeclarationError) Error() string {
	if e.Identifier != "" {
		return fmt.Sprintf(
			"cannot declare %s here: `%s`",
			e.Kind.Name(),
			e.Identifier,
		)
	}

	return fmt.Sprintf("cannot declare %s here", e.Kind.Name())
}

// MissingInitializerError

type MissingInitializerError struct {
	ContainerType  Type
	FirstFieldName string
	FirstFieldPos  ast.Position
}

var _ SemanticError = &MissingInitializerError{}
var _ errors.UserError = &MissingInitializerError{}

func (*MissingInitializerError) isSemanticError() {}

func (*MissingInitializerError) IsUserError() {}

func (e *MissingInitializerError) Error() string {
	return fmt.Sprintf(
		"missing initializer for field `%s` in type `%s`",
		e.FirstFieldName,
		e.ContainerType.QualifiedString(),
	)
}

func (e *MissingInitializerError) StartPosition() ast.Position {
	return e.FirstFieldPos
}

func (e *MissingInitializerError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.FirstFieldName)
	return e.FirstFieldPos.Shifted(memoryGauge, length-1)
}

// NotDeclaredMemberError

type NotDeclaredMemberError struct {
	Type       Type
	Expression *ast.MemberExpression
	Name       string
	ast.Range
	SuggestMember bool
}

var _ SemanticError = &NotDeclaredMemberError{}
var _ errors.UserError = &NotDeclaredMemberError{}
var _ errors.SecondaryError = &NotDeclaredMemberError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &NotDeclaredMemberError{}

func (*NotDeclaredMemberError) isSemanticError() {}

func (*NotDeclaredMemberError) IsUserError() {}

func (e *NotDeclaredMemberError) Error() string {
	return fmt.Sprintf(
		"value of type `%s` has no member `%s`",
		e.Type.QualifiedString(),
		e.Name,
	)
}

func (e *NotDeclaredMemberError) findOptionalMember() string {
	optionalType, ok := e.Type.(*OptionalType)
	if !ok {
		return ""
	}

	members := optionalType.Type.GetMembers()
	name := e.Name
	_, ok = members[name]
	if !ok {
		return ""
	}

	return name
}

func (e *NotDeclaredMemberError) SecondaryError() string {
	if optionalMember := e.findOptionalMember(); optionalMember != "" {
		return fmt.Sprintf("type is optional, consider optional-chaining: ?.%s", optionalMember)
	}
	if closestMember := e.findClosestMember(); closestMember != "" {
		return fmt.Sprintf("did you mean `%s`?", closestMember)
	}
	return "unknown member"
}

func (e *NotDeclaredMemberError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	optionalMember := e.findOptionalMember()
	if optionalMember == "" {
		return nil
	}

	accessPos := e.Expression.AccessPos

	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "use optional chaining",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "?",
					Range: ast.Range{
						StartPos: accessPos,
						EndPos:   accessPos,
					},
				},
			},
		},
	}
}

// findClosestMember searches the names of the members on the accessed type,
// and finds the name with the smallest edit distance from the member the user
// tried to access. In cases of typos, this should provide a helpful hint.
func (e *NotDeclaredMemberError) findClosestMember() (closestMember string) {
	if !e.SuggestMember {
		return
	}

	nameRunes := []rune(e.Name)

	closestDistance := len(e.Name)

	var sortedMemberNames []string
	for memberName := range e.Type.GetMembers() { //nolint:maprange
		sortedMemberNames = append(sortedMemberNames, memberName)
	}
	sort.Strings(sortedMemberNames)

	for _, memberName := range sortedMemberNames {
		distance := levenshtein.DistanceForStrings(
			nameRunes,
			[]rune(memberName),
			levenshtein.DefaultOptions,
		)

		// Don't update the closest member if the distance is greater than one already found,
		// or if the edits required would involve a complete replacement of the member's text
		if distance < closestDistance && distance < len(memberName) {
			closestMember = memberName
			closestDistance = distance
		}
	}

	return
}

// AssignmentToConstantMemberError

// TODO: maybe split up into two errors:
//  - assignment to constant field
//  - assignment to function

type AssignmentToConstantMemberError struct {
	Name string
	ast.Range
}

var _ SemanticError = &AssignmentToConstantMemberError{}
var _ errors.UserError = &AssignmentToConstantMemberError{}

func (*AssignmentToConstantMemberError) isSemanticError() {}

func (*AssignmentToConstantMemberError) IsUserError() {}

func (e *AssignmentToConstantMemberError) Error() string {
	return fmt.Sprintf("cannot assign to constant member: `%s`", e.Name)
}

// FieldReinitializationError
type FieldReinitializationError struct {
	Name string
	ast.Range
}

var _ SemanticError = &FieldReinitializationError{}
var _ errors.UserError = &FieldReinitializationError{}

func (*FieldReinitializationError) isSemanticError() {}

func (*FieldReinitializationError) IsUserError() {}

func (e *FieldReinitializationError) Error() string {
	return fmt.Sprintf("invalid reinitialization of field: `%s`", e.Name)
}

// FieldUninitializedError
type FieldUninitializedError struct {
	ContainerType Type
	Name          string
	Pos           ast.Position
}

var _ SemanticError = &FieldUninitializedError{}
var _ errors.UserError = &FieldUninitializedError{}
var _ errors.SecondaryError = &FieldUninitializedError{}

func (*FieldUninitializedError) isSemanticError() {}

func (*FieldUninitializedError) IsUserError() {}

func (e *FieldUninitializedError) Error() string {
	return fmt.Sprintf(
		"missing initialization of field `%s` in type `%s`",
		e.Name,
		e.ContainerType.QualifiedString(),
	)
}

func (e *FieldUninitializedError) SecondaryError() string {
	return "not initialized"
}

func (e *FieldUninitializedError) StartPosition() ast.Position {
	return e.Pos
}

func (e *FieldUninitializedError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// FieldTypeNotStorableError is an error that is reported for
// fields of composite types that are not storable.
//
// Field types have to be storable because the storage layer
// needs to know how to store the field, which is not possible
// for all types.
//
// For example, the type `Int` is a storable type,
// whereas a function type is not.

type FieldTypeNotStorableError struct {
	Type Type
	Name string
	Pos  ast.Position
}

var _ SemanticError = &FieldTypeNotStorableError{}
var _ errors.UserError = &FieldTypeNotStorableError{}
var _ errors.SecondaryError = &FieldTypeNotStorableError{}

func (*FieldTypeNotStorableError) isSemanticError() {}

func (*FieldTypeNotStorableError) IsUserError() {}

func (e *FieldTypeNotStorableError) Error() string {
	return fmt.Sprintf(
		"field %s has non-storable type: %s",
		e.Name,
		e.Type,
	)
}

func (e *FieldTypeNotStorableError) SecondaryError() string {
	return "all contract fields must be storable"
}

func (e *FieldTypeNotStorableError) StartPosition() ast.Position {
	return e.Pos
}

func (e *FieldTypeNotStorableError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// FunctionExpressionInConditionError

type FunctionExpressionInConditionError struct {
	ast.Range
}

var _ SemanticError = &FunctionExpressionInConditionError{}
var _ errors.UserError = &FunctionExpressionInConditionError{}

func (*FunctionExpressionInConditionError) isSemanticError() {}

func (*FunctionExpressionInConditionError) IsUserError() {}

func (e *FunctionExpressionInConditionError) Error() string {
	return "condition contains function"
}

// InvalidEmitConditionError

type InvalidEmitConditionError struct {
	ast.Range
}

var _ SemanticError = &InvalidEmitConditionError{}
var _ errors.UserError = &InvalidEmitConditionError{}

func (*InvalidEmitConditionError) isSemanticError() {}

func (*InvalidEmitConditionError) IsUserError() {}

func (e *InvalidEmitConditionError) Error() string {
	return "invalid emit condition "
}

// MissingReturnValueError

type MissingReturnValueError struct {
	ExpectedValueType Type
	ast.Range
}

var _ SemanticError = &MissingReturnValueError{}
var _ errors.UserError = &MissingReturnValueError{}

func (*MissingReturnValueError) isSemanticError() {}

func (*MissingReturnValueError) IsUserError() {}

func (e *MissingReturnValueError) Error() string {
	var typeDescription string
	if e.ExpectedValueType.IsInvalidType() {
		typeDescription = "non-void"
	} else {
		typeDescription = fmt.Sprintf("`%s`", e.ExpectedValueType)
	}

	return fmt.Sprintf(
		"missing value in return from function with %s return type",
		typeDescription,
	)
}

// InvalidImplementationError

type InvalidImplementationError struct {
	ImplementedKind common.DeclarationKind
	ContainerKind   common.DeclarationKind
	Pos             ast.Position
}

var _ SemanticError = &InvalidImplementationError{}
var _ errors.UserError = &InvalidImplementationError{}

func (*InvalidImplementationError) isSemanticError() {}

func (*InvalidImplementationError) IsUserError() {}

func (e *InvalidImplementationError) Error() string {
	return fmt.Sprintf(
		"cannot implement %s in %s",
		e.ImplementedKind.Name(),
		e.ContainerKind.Name(),
	)
}

func (e *InvalidImplementationError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidImplementationError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidConformanceError

type InvalidConformanceError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidConformanceError{}
var _ errors.UserError = &InvalidConformanceError{}

func (*InvalidConformanceError) isSemanticError() {}

func (*InvalidConformanceError) IsUserError() {}

func (e *InvalidConformanceError) Error() string {
	return fmt.Sprintf(
		"cannot conform to non-interface type: `%s`",
		e.Type.QualifiedString(),
	)
}

// InvalidEnumRawTypeError

type InvalidEnumRawTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidEnumRawTypeError{}
var _ errors.UserError = &InvalidEnumRawTypeError{}
var _ errors.SecondaryError = &InvalidEnumRawTypeError{}

func (*InvalidEnumRawTypeError) isSemanticError() {}

func (*InvalidEnumRawTypeError) IsUserError() {}

func (e *InvalidEnumRawTypeError) Error() string {
	return fmt.Sprintf(
		"invalid enum raw type: `%s`",
		e.Type.QualifiedString(),
	)
}

func (e *InvalidEnumRawTypeError) SecondaryError() string {
	return "only integer types are currently supported for enums"
}

// MissingEnumRawTypeError

type MissingEnumRawTypeError struct {
	Pos ast.Position
}

var _ SemanticError = &MissingEnumRawTypeError{}
var _ errors.UserError = &MissingEnumRawTypeError{}

func (*MissingEnumRawTypeError) isSemanticError() {}

func (*MissingEnumRawTypeError) IsUserError() {}

func (e *MissingEnumRawTypeError) Error() string {
	return "missing enum raw type"
}

func (e *MissingEnumRawTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingEnumRawTypeError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidEnumConformancesError

type InvalidEnumConformancesError struct {
	ast.Range
}

var _ SemanticError = &InvalidEnumConformancesError{}
var _ errors.UserError = &InvalidEnumConformancesError{}

func (*InvalidEnumConformancesError) isSemanticError() {}

func (*InvalidEnumConformancesError) IsUserError() {}

func (e *InvalidEnumConformancesError) Error() string {
	return "enums cannot conform to interfaces"
}

// InvalidAttachmentConformancesError

type InvalidAttachmentConformancesError struct {
	ast.Range
}

var _ SemanticError = &InvalidAttachmentConformancesError{}
var _ errors.UserError = &InvalidAttachmentConformancesError{}

func (*InvalidAttachmentConformancesError) isSemanticError() {}

func (*InvalidAttachmentConformancesError) IsUserError() {}

func (e *InvalidAttachmentConformancesError) Error() string {
	return "attachments cannot conform to interfaces"
}

// ConformanceError

// TODO: report each missing member and mismatch as note

type MemberMismatch struct {
	CompositeMember *Member
	InterfaceMember *Member
}

type InitializerMismatch struct {
	CompositePurity     FunctionPurity
	InterfacePurity     FunctionPurity
	CompositeParameters []Parameter
	InterfaceParameters []Parameter
}
type ConformanceError struct {
	CompositeDeclaration        ast.CompositeLikeDeclaration
	CompositeType               *CompositeType
	InterfaceType               *InterfaceType
	NestedInterfaceType         *InterfaceType
	InitializerMismatch         *InitializerMismatch
	MissingMembers              []*Member
	MemberMismatches            []MemberMismatch
	MissingNestedCompositeTypes []*CompositeType
	Pos                         ast.Position
}

var _ SemanticError = &ConformanceError{}
var _ errors.UserError = &ConformanceError{}
var _ errors.SecondaryError = &ConformanceError{}

func (*ConformanceError) isSemanticError() {}

func (*ConformanceError) IsUserError() {}

func (e *ConformanceError) Error() string {
	return fmt.Sprintf(
		"%s `%s` does not conform to %s interface `%s`",
		e.CompositeType.Kind.Name(),
		e.CompositeType.QualifiedString(),
		e.InterfaceType.CompositeKind.Name(),
		e.InterfaceType.QualifiedString(),
	)
}

func (e *ConformanceError) SecondaryError() string {
	var builder strings.Builder
	if len(e.MissingMembers) > 0 {
		builder.WriteString(fmt.Sprintf("`%s` is missing definitions for members: ", e.CompositeType.QualifiedString()))
		for i, member := range e.MissingMembers {
			builder.WriteString(fmt.Sprintf("`%s`", member.Identifier.Identifier))
			if i != len(e.MissingMembers)-1 {
				builder.WriteString(", ")
			}
		}
		if len(e.MissingNestedCompositeTypes) > 0 {
			builder.WriteString(". ")
		}
	}

	if len(e.MissingNestedCompositeTypes) > 0 {
		builder.WriteString(fmt.Sprintf("`%s` is", e.CompositeType.QualifiedString()))
		if len(e.MissingMembers) > 0 {
			builder.WriteString(" also")
		}
		builder.WriteString(" missing definitions for types: ")
		for i, ty := range e.MissingNestedCompositeTypes {
			builder.WriteString(fmt.Sprintf("`%s`", ty.QualifiedString()))
			if i != len(e.MissingNestedCompositeTypes)-1 {
				builder.WriteString(", ")
			}
		}
	}

	return builder.String()
}

func (e *ConformanceError) StartPosition() ast.Position {
	return e.Pos
}

func (e *ConformanceError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *ConformanceError) ErrorNotes() (notes []errors.ErrorNote) {

	for _, memberMismatch := range e.MemberMismatches {
		compositeMemberIdentifierRange :=
			ast.NewUnmeteredRangeFromPositioned(memberMismatch.CompositeMember.Identifier)

		notes = append(notes, &MemberMismatchNote{
			Range: compositeMemberIdentifierRange,
		})
	}

	if e.InitializerMismatch != nil && len(e.CompositeDeclaration.DeclarationMembers().Initializers()) > 0 {
		compositeMemberIdentifierRange :=
			//	right now we only support a single initializer
			ast.NewUnmeteredRangeFromPositioned(e.CompositeDeclaration.DeclarationMembers().Initializers()[0].FunctionDeclaration.Identifier)

		notes = append(notes, &MemberMismatchNote{
			Range: compositeMemberIdentifierRange,
		})
	}

	return
}

// MemberMismatchNote

type MemberMismatchNote struct {
	ast.Range
}

func (n MemberMismatchNote) Message() string {
	return "mismatch here"
}

// DuplicateConformanceError
//
// TODO: just make this a warning?
type DuplicateConformanceError struct {
	CompositeKindedType CompositeKindedType
	InterfaceType       *InterfaceType
	ast.Range
}

var _ SemanticError = &DuplicateConformanceError{}
var _ errors.UserError = &DuplicateConformanceError{}

func (*DuplicateConformanceError) isSemanticError() {}

func (*DuplicateConformanceError) IsUserError() {}

func (e *DuplicateConformanceError) Error() string {
	return fmt.Sprintf(
		"%s `%s` repeats conformance to %s `%s`",
		e.CompositeKindedType.GetCompositeKind().Name(),
		e.CompositeKindedType.QualifiedString(),
		e.InterfaceType.CompositeKind.DeclarationKind(true).Name(),
		e.InterfaceType.QualifiedString(),
	)
}

// CyclicConformanceError
type CyclicConformanceError struct {
	InterfaceType *InterfaceType
	ast.Range
}

var _ SemanticError = CyclicConformanceError{}
var _ errors.UserError = CyclicConformanceError{}

func (CyclicConformanceError) isSemanticError() {}

func (CyclicConformanceError) IsUserError() {}

func (e CyclicConformanceError) Error() string {
	return fmt.Sprintf(
		"`%s` has a cyclic conformance to itself",
		e.InterfaceType.QualifiedString(),
	)
}

// MultipleInterfaceDefaultImplementationsError
type MultipleInterfaceDefaultImplementationsError struct {
	CompositeKindedType CompositeKindedType
	Member              *Member
	ast.Range
}

var _ SemanticError = &MultipleInterfaceDefaultImplementationsError{}
var _ errors.UserError = &MultipleInterfaceDefaultImplementationsError{}

func (*MultipleInterfaceDefaultImplementationsError) isSemanticError() {}

func (*MultipleInterfaceDefaultImplementationsError) IsUserError() {}

func (e *MultipleInterfaceDefaultImplementationsError) Error() string {
	return fmt.Sprintf(
		"%s `%s` has multiple interface default implementations for function `%s`",
		e.CompositeKindedType.GetCompositeKind().Name(),
		e.CompositeKindedType.QualifiedString(),
		e.Member.Identifier.Identifier,
	)
}

// SpecialFunctionDefaultImplementationError
type SpecialFunctionDefaultImplementationError struct {
	Container  ast.Declaration
	Identifier *ast.Identifier
	KindName   string
}

var _ SemanticError = &SpecialFunctionDefaultImplementationError{}
var _ errors.UserError = &SpecialFunctionDefaultImplementationError{}

func (*SpecialFunctionDefaultImplementationError) isSemanticError() {}

func (*SpecialFunctionDefaultImplementationError) IsUserError() {}

func (e *SpecialFunctionDefaultImplementationError) Error() string {
	return fmt.Sprintf(
		"%s may not be defined as a default function on %s %s",
		e.Identifier.Identifier,
		e.KindName,
		e.Container.DeclarationIdentifier().Identifier,
	)
}

func (e *SpecialFunctionDefaultImplementationError) StartPosition() ast.Position {
	return e.Identifier.StartPosition()
}

func (e *SpecialFunctionDefaultImplementationError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	return e.Identifier.EndPosition(memoryGauge)
}

// InterfaceMemberConflictError
type InterfaceMemberConflictError struct {
	InterfaceType            *InterfaceType
	ConflictingInterfaceType *InterfaceType
	MemberName               string
	MemberKind               common.DeclarationKind
	ConflictingMemberKind    common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InterfaceMemberConflictError{}
var _ errors.UserError = &InterfaceMemberConflictError{}

func (*InterfaceMemberConflictError) isSemanticError() {}

func (*InterfaceMemberConflictError) IsUserError() {}

func (e *InterfaceMemberConflictError) Error() string {
	return fmt.Sprintf(
		"`%s` %s of `%s` conflicts with a %s with the same name in `%s`",
		e.MemberName,
		e.MemberKind.Name(),
		e.InterfaceType.QualifiedIdentifier(),
		e.ConflictingMemberKind.Name(),
		e.ConflictingInterfaceType.QualifiedString(),
	)
}

// MissingConformanceError
type MissingConformanceError struct {
	CompositeType *CompositeType
	InterfaceType *InterfaceType
	ast.Range
}

var _ SemanticError = &MissingConformanceError{}
var _ errors.UserError = &MissingConformanceError{}

func (*MissingConformanceError) isSemanticError() {}

func (*MissingConformanceError) IsUserError() {}

func (e *MissingConformanceError) Error() string {
	return fmt.Sprintf(
		"%s `%s` is missing a declaration to required conformance to %s `%s`",
		e.CompositeType.Kind.Name(),
		e.CompositeType.QualifiedString(),
		e.InterfaceType.CompositeKind.DeclarationKind(true).Name(),
		e.InterfaceType.QualifiedString(),
	)
}

// UnresolvedImportError

type UnresolvedImportError struct {
	ImportLocation common.Location
	ast.Range
}

var _ SemanticError = &UnresolvedImportError{}
var _ errors.UserError = &UnresolvedImportError{}

func (*UnresolvedImportError) isSemanticError() {}

func (*UnresolvedImportError) IsUserError() {}

func (e *UnresolvedImportError) Error() string {
	return fmt.Sprintf("import could not be resolved: %s", e.ImportLocation)
}

// NotExportedError

type NotExportedError struct {
	Name           string
	ImportLocation common.Location
	Available      []string
	Pos            ast.Position
}

var _ SemanticError = &NotExportedError{}
var _ errors.UserError = &NotExportedError{}
var _ errors.SecondaryError = &NotExportedError{}

func (*NotExportedError) isSemanticError() {}

func (*NotExportedError) IsUserError() {}

func (e *NotExportedError) Error() string {
	return fmt.Sprintf(
		"cannot find declaration `%s` in `%s`",
		e.Name,
		e.ImportLocation,
	)
}

func (e *NotExportedError) SecondaryError() string {
	var builder strings.Builder
	builder.WriteString("available exported declarations are:\n")

	for _, available := range e.Available {
		builder.WriteString(fmt.Sprintf(" - `%s`\n", available))
	}

	return builder.String()
}

func (e *NotExportedError) StartPosition() ast.Position {
	return e.Pos
}

func (e *NotExportedError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// ImportedProgramError

type ImportedProgramError struct {
	Err      error
	Location common.Location
	ast.Range
}

var _ SemanticError = &ImportedProgramError{}
var _ errors.UserError = &ImportedProgramError{}
var _ errors.ParentError = &ImportedProgramError{}

func (*ImportedProgramError) isSemanticError() {}

func (*ImportedProgramError) IsUserError() {}

func (e *ImportedProgramError) Error() string {
	return fmt.Sprintf(
		"checking of imported program `%s` failed",
		e.Location,
	)
}

func (e *ImportedProgramError) ImportLocation() common.Location {
	return e.Location
}

func (e *ImportedProgramError) ChildErrors() []error {
	return []error{e.Err}
}

func (e *ImportedProgramError) Unwrap() error {
	return e.Err
}

// AlwaysFailingNonResourceCastingTypeError

type AlwaysFailingNonResourceCastingTypeError struct {
	ValueType  Type
	TargetType Type
	ast.Range
}

var _ SemanticError = &AlwaysFailingNonResourceCastingTypeError{}
var _ errors.UserError = &AlwaysFailingNonResourceCastingTypeError{}

func (*AlwaysFailingNonResourceCastingTypeError) isSemanticError() {}

func (*AlwaysFailingNonResourceCastingTypeError) IsUserError() {}

func (e *AlwaysFailingNonResourceCastingTypeError) Error() string {
	return fmt.Sprintf(
		"cast of value of resource-type `%s` to non-resource type `%s` will always fail",
		e.ValueType.QualifiedString(),
		e.TargetType.QualifiedString(),
	)
}

// AlwaysFailingResourceCastingTypeError

type AlwaysFailingResourceCastingTypeError struct {
	ValueType  Type
	TargetType Type
	ast.Range
}

var _ SemanticError = &AlwaysFailingResourceCastingTypeError{}
var _ errors.UserError = &AlwaysFailingResourceCastingTypeError{}

func (*AlwaysFailingResourceCastingTypeError) isSemanticError() {}

func (*AlwaysFailingResourceCastingTypeError) IsUserError() {}

func (e *AlwaysFailingResourceCastingTypeError) Error() string {
	return fmt.Sprintf(
		"cast of value of non-resource-type `%s` to resource type `%s` will always fail",
		e.ValueType.QualifiedString(),
		e.TargetType.QualifiedString(),
	)
}

// UnsupportedOverloadingError

type UnsupportedOverloadingError struct {
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &UnsupportedOverloadingError{}
var _ errors.UserError = &UnsupportedOverloadingError{}

func (*UnsupportedOverloadingError) isSemanticError() {}

func (*UnsupportedOverloadingError) IsUserError() {}

func (e *UnsupportedOverloadingError) Error() string {
	return fmt.Sprintf(
		"%s overloading is not supported yet",
		e.DeclarationKind.Name(),
	)
}

// CompositeKindMismatchError

type CompositeKindMismatchError struct {
	ExpectedKind common.CompositeKind
	ActualKind   common.CompositeKind
	ast.Range
}

var _ SemanticError = &CompositeKindMismatchError{}
var _ errors.UserError = &CompositeKindMismatchError{}
var _ errors.SecondaryError = &CompositeKindMismatchError{}

func (*CompositeKindMismatchError) isSemanticError() {}

func (*CompositeKindMismatchError) IsUserError() {}

func (e *CompositeKindMismatchError) Error() string {
	return "mismatched composite kinds"
}

func (e *CompositeKindMismatchError) SecondaryError() string {
	return fmt.Sprintf(
		"expected `%s`, got `%s`",
		e.ExpectedKind.Name(),
		e.ActualKind.Name(),
	)
}

// InvalidIntegerLiteralRangeError

type InvalidIntegerLiteralRangeError struct {
	ExpectedType   Type
	ExpectedMinInt *big.Int
	ExpectedMaxInt *big.Int
	ast.Range
}

var _ SemanticError = &InvalidIntegerLiteralRangeError{}
var _ errors.UserError = &InvalidIntegerLiteralRangeError{}
var _ errors.SecondaryError = &InvalidIntegerLiteralRangeError{}

func (*InvalidIntegerLiteralRangeError) IsUserError() {}

func (*InvalidIntegerLiteralRangeError) isSemanticError() {}

func (e *InvalidIntegerLiteralRangeError) Error() string {
	return "integer literal out of range"
}

func (e *InvalidIntegerLiteralRangeError) SecondaryError() string {
	return fmt.Sprintf(
		"expected `%s`, in range [%s, %s]",
		e.ExpectedType.QualifiedString(),
		e.ExpectedMinInt,
		e.ExpectedMaxInt,
	)
}

// InvalidAddressLiteralError

type InvalidAddressLiteralError struct {
	ast.Range
}

var _ SemanticError = &InvalidAddressLiteralError{}
var _ errors.UserError = &InvalidAddressLiteralError{}

func (*InvalidAddressLiteralError) isSemanticError() {}

func (*InvalidAddressLiteralError) IsUserError() {}

func (e *InvalidAddressLiteralError) Error() string {
	return "invalid address"
}

// InvalidFixedPointLiteralRangeError

type InvalidFixedPointLiteralRangeError struct {
	ExpectedType          Type
	ExpectedMinInt        *big.Int
	ExpectedMinFractional *big.Int
	ExpectedMaxInt        *big.Int
	ExpectedMaxFractional *big.Int
	ast.Range
}

var _ SemanticError = &InvalidFixedPointLiteralRangeError{}
var _ errors.UserError = &InvalidFixedPointLiteralRangeError{}
var _ errors.SecondaryError = &InvalidFixedPointLiteralRangeError{}

func (*InvalidFixedPointLiteralRangeError) isSemanticError() {}

func (*InvalidFixedPointLiteralRangeError) IsUserError() {}

func (e *InvalidFixedPointLiteralRangeError) Error() string {
	return "fixed-point literal out of range"
}

func (e *InvalidFixedPointLiteralRangeError) SecondaryError() string {
	return fmt.Sprintf(
		"expected `%s`, in range [%s.%s, %s.%s]",
		e.ExpectedType.QualifiedString(),
		e.ExpectedMinInt,
		e.ExpectedMinFractional,
		e.ExpectedMaxInt,
		e.ExpectedMaxFractional,
	)
}

// InvalidFixedPointLiteralScaleError

type InvalidFixedPointLiteralScaleError struct {
	ExpectedType  Type
	ExpectedScale uint
	ast.Range
}

var _ SemanticError = &InvalidFixedPointLiteralScaleError{}
var _ errors.UserError = &InvalidFixedPointLiteralScaleError{}
var _ errors.SecondaryError = &InvalidFixedPointLiteralScaleError{}

func (*InvalidFixedPointLiteralScaleError) isSemanticError() {}

func (*InvalidFixedPointLiteralScaleError) IsUserError() {}

func (e *InvalidFixedPointLiteralScaleError) Error() string {
	return "fixed-point literal scale out of range"
}

func (e *InvalidFixedPointLiteralScaleError) SecondaryError() string {
	return fmt.Sprintf(
		"expected `%s`, with maximum scale %d",
		e.ExpectedType.QualifiedString(),
		e.ExpectedScale,
	)
}

// MissingReturnStatementError

type MissingReturnStatementError struct {
	ast.Range
}

var _ SemanticError = &MissingReturnStatementError{}
var _ errors.UserError = &MissingReturnStatementError{}

func (*MissingReturnStatementError) isSemanticError() {}

func (*MissingReturnStatementError) IsUserError() {}

func (e *MissingReturnStatementError) Error() string {
	return "missing return statement"
}

// UnsupportedOptionalChainingAssignmentError

type UnsupportedOptionalChainingAssignmentError struct {
	ast.Range
}

var _ SemanticError = &UnsupportedOptionalChainingAssignmentError{}
var _ errors.UserError = &UnsupportedOptionalChainingAssignmentError{}

func (*UnsupportedOptionalChainingAssignmentError) isSemanticError() {}

func (*UnsupportedOptionalChainingAssignmentError) IsUserError() {}

func (e *UnsupportedOptionalChainingAssignmentError) Error() string {
	return "cannot assign to optional chaining expression"
}

// MissingResourceAnnotationError

type MissingResourceAnnotationError struct {
	ast.Range
}

var _ SemanticError = &MissingResourceAnnotationError{}
var _ errors.UserError = &MissingResourceAnnotationError{}

func (*MissingResourceAnnotationError) isSemanticError() {}

func (*MissingResourceAnnotationError) IsUserError() {}

func (e *MissingResourceAnnotationError) Error() string {
	return fmt.Sprintf(
		"missing resource annotation: `%s`",
		common.CompositeKindResource.Annotation(),
	)
}

// InvalidNestedResourceMoveError

type InvalidNestedResourceMoveError struct {
	ast.Range
}

var _ SemanticError = &InvalidNestedResourceMoveError{}
var _ errors.UserError = &InvalidNestedResourceMoveError{}

func (*InvalidNestedResourceMoveError) isSemanticError() {}

func (*InvalidNestedResourceMoveError) IsUserError() {}

func (e *InvalidNestedResourceMoveError) Error() string {
	return "cannot move nested resource"
}

// InvalidInterfaceConditionResourceInvalidationError

type InvalidInterfaceConditionResourceInvalidationError struct {
	ast.Range
}

var _ SemanticError = &InvalidInterfaceConditionResourceInvalidationError{}
var _ errors.UserError = &InvalidInterfaceConditionResourceInvalidationError{}

func (*InvalidInterfaceConditionResourceInvalidationError) isSemanticError() {}

func (*InvalidInterfaceConditionResourceInvalidationError) IsUserError() {}

func (e *InvalidInterfaceConditionResourceInvalidationError) Error() string {
	return "cannot invalidate resource in interface condition"
}

// InvalidResourceAnnotationError

type InvalidResourceAnnotationError struct {
	ast.Range
}

var _ SemanticError = &InvalidResourceAnnotationError{}
var _ errors.UserError = &InvalidResourceAnnotationError{}

func (*InvalidResourceAnnotationError) isSemanticError() {}

func (*InvalidResourceAnnotationError) IsUserError() {}

func (e *InvalidResourceAnnotationError) Error() string {
	return fmt.Sprintf(
		"invalid resource annotation: `%s`",
		common.CompositeKindResource.Annotation(),
	)
}

// InvalidInterfaceTypeError

type InvalidInterfaceTypeError struct {
	ActualType   Type
	ExpectedType Type
	ast.Range
}

var _ SemanticError = &InvalidInterfaceTypeError{}
var _ errors.UserError = &InvalidInterfaceTypeError{}
var _ errors.SecondaryError = &InvalidInterfaceTypeError{}

func (*InvalidInterfaceTypeError) isSemanticError() {}

func (*InvalidInterfaceTypeError) IsUserError() {}

func (e *InvalidInterfaceTypeError) Error() string {
	return "invalid use of interface as type"
}

func (e *InvalidInterfaceTypeError) SecondaryError() string {
	return fmt.Sprintf(
		"got `%s`; consider using `%s`",
		e.ActualType.QualifiedString(),
		e.ExpectedType.QualifiedString(),
	)
}

// InvalidInterfaceDeclarationError

type InvalidInterfaceDeclarationError struct {
	CompositeKind common.CompositeKind
	ast.Range
}

var _ SemanticError = &InvalidInterfaceDeclarationError{}
var _ errors.UserError = &InvalidInterfaceDeclarationError{}

func (*InvalidInterfaceDeclarationError) isSemanticError() {}

func (*InvalidInterfaceDeclarationError) IsUserError() {}

func (e *InvalidInterfaceDeclarationError) Error() string {
	return fmt.Sprintf(
		"%s interfaces are not supported",
		e.CompositeKind.Name(),
	)
}

// IncorrectTransferOperationError

type IncorrectTransferOperationError struct {
	ActualOperation   ast.TransferOperation
	ExpectedOperation ast.TransferOperation
	ast.Range
}

var _ SemanticError = &IncorrectTransferOperationError{}
var _ errors.UserError = &IncorrectTransferOperationError{}
var _ errors.SecondaryError = &IncorrectTransferOperationError{}

func (*IncorrectTransferOperationError) isSemanticError() {}

func (*IncorrectTransferOperationError) IsUserError() {}

func (e *IncorrectTransferOperationError) Error() string {
	return "incorrect transfer operation"
}

func (e *IncorrectTransferOperationError) SecondaryError() string {
	return fmt.Sprintf(
		"expected `%s`",
		e.ExpectedOperation.Operator(),
	)
}

// InvalidConstructionError

type InvalidConstructionError struct {
	ast.Range
}

var _ SemanticError = &InvalidConstructionError{}
var _ errors.UserError = &InvalidConstructionError{}

func (*InvalidConstructionError) isSemanticError() {}

func (*InvalidConstructionError) IsUserError() {}

func (e *InvalidConstructionError) Error() string {
	return "cannot create value: not a resource"
}

// InvalidDestructionError

type InvalidDestructionError struct {
	ast.Range
}

var _ SemanticError = &InvalidDestructionError{}
var _ errors.UserError = &InvalidDestructionError{}

func (*InvalidDestructionError) isSemanticError() {}

func (*InvalidDestructionError) IsUserError() {}

func (e *InvalidDestructionError) Error() string {
	return "cannot destroy value: not a resource"
}

// ResourceLossError

type ResourceLossError struct {
	ast.Range
}

var _ SemanticError = &ResourceLossError{}
var _ errors.UserError = &ResourceLossError{}

func (*ResourceLossError) isSemanticError() {}

func (*ResourceLossError) IsUserError() {}

func (e *ResourceLossError) Error() string {
	return "loss of resource"
}

// ResourceUseAfterInvalidationError

type ResourceUseAfterInvalidationError struct {
	Invalidation ResourceInvalidation
	ast.Range
}

var _ SemanticError = &ResourceUseAfterInvalidationError{}
var _ errors.UserError = &ResourceUseAfterInvalidationError{}
var _ errors.SecondaryError = &ResourceUseAfterInvalidationError{}

func (*ResourceUseAfterInvalidationError) isSemanticError() {}

func (*ResourceUseAfterInvalidationError) IsUserError() {}

func (e *ResourceUseAfterInvalidationError) Error() string {
	return fmt.Sprintf(
		"use of previously %s resource",
		e.Invalidation.Kind.CoarsePassiveVerb(),
	)
}

func (e *ResourceUseAfterInvalidationError) SecondaryError() string {
	return fmt.Sprintf(
		"resource used here after %s",
		e.Invalidation.Kind.CoarseNoun(),
	)
}

func (e *ResourceUseAfterInvalidationError) ErrorNotes() []errors.ErrorNote {
	invalidation := e.Invalidation
	return []errors.ErrorNote{
		newPreviousResourceInvalidationNote(invalidation),
	}
}

// PreviousResourceInvalidationNote

type PreviousResourceInvalidationNote struct {
	ResourceInvalidation
	ast.Range
}

func newPreviousResourceInvalidationNote(invalidation ResourceInvalidation) PreviousResourceInvalidationNote {
	return PreviousResourceInvalidationNote{
		ResourceInvalidation: invalidation,
		Range: ast.NewUnmeteredRange(
			invalidation.StartPos,
			invalidation.EndPos,
		),
	}
}

func (n PreviousResourceInvalidationNote) Message() string {
	return fmt.Sprintf(
		"resource previously %s here",
		n.ResourceInvalidation.Kind.CoarsePassiveVerb(),
	)
}

// MissingCreateError

type MissingCreateError struct {
	ast.Range
}

var _ SemanticError = &MissingCreateError{}
var _ errors.UserError = &MissingCreateError{}
var _ errors.SecondaryError = &MissingCreateError{}

func (*MissingCreateError) isSemanticError() {}

func (*MissingCreateError) IsUserError() {}

func (e *MissingCreateError) Error() string {
	return "cannot create resource"
}

func (e *MissingCreateError) SecondaryError() string {
	return "expected `create`"
}

// MissingMoveOperationError

type MissingMoveOperationError struct {
	Pos ast.Position
}

var _ SemanticError = &MissingMoveOperationError{}
var _ errors.UserError = &MissingMoveOperationError{}

func (*MissingMoveOperationError) isSemanticError() {}

func (*MissingMoveOperationError) IsUserError() {}

func (e *MissingMoveOperationError) Error() string {
	return "missing move operation: `<-`"
}

func (e *MissingMoveOperationError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingMoveOperationError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidMoveOperationError

type InvalidMoveOperationError struct {
	ast.Range
}

var _ SemanticError = &InvalidMoveOperationError{}
var _ errors.UserError = &InvalidMoveOperationError{}
var _ errors.SecondaryError = &InvalidMoveOperationError{}

func (*InvalidMoveOperationError) isSemanticError() {}

func (*InvalidMoveOperationError) IsUserError() {}

func (e *InvalidMoveOperationError) Error() string {
	return "invalid move operation for non-resource"
}

func (e *InvalidMoveOperationError) SecondaryError() string {
	return "unexpected `<-`"
}

// ResourceCapturingError

type ResourceCapturingError struct {
	Name string
	Pos  ast.Position
}

var _ SemanticError = &ResourceCapturingError{}
var _ errors.UserError = &ResourceCapturingError{}

func (*ResourceCapturingError) isSemanticError() {}

func (*ResourceCapturingError) IsUserError() {}

func (e *ResourceCapturingError) Error() string {
	return fmt.Sprintf("cannot capture resource in closure: `%s`", e.Name)
}

func (e *ResourceCapturingError) StartPosition() ast.Position {
	return e.Pos
}

func (e *ResourceCapturingError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// InvalidResourceFieldError

type InvalidResourceFieldError struct {
	Name          string
	CompositeKind common.CompositeKind
	Pos           ast.Position
}

var _ SemanticError = &InvalidResourceFieldError{}
var _ errors.UserError = &InvalidResourceFieldError{}

func (*InvalidResourceFieldError) isSemanticError() {}

func (*InvalidResourceFieldError) IsUserError() {}

func (e *InvalidResourceFieldError) Error() string {
	return fmt.Sprintf(
		"invalid resource field in %s: `%s`",
		e.CompositeKind.Name(),
		e.Name,
	)
}

func (e *InvalidResourceFieldError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidResourceFieldError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// InvalidSwapExpressionError

type InvalidSwapExpressionError struct {
	Side common.OperandSide
	ast.Range
}

var _ SemanticError = &InvalidSwapExpressionError{}
var _ errors.UserError = &InvalidSwapExpressionError{}
var _ errors.SecondaryError = &InvalidSwapExpressionError{}

func (*InvalidSwapExpressionError) isSemanticError() {}

func (*InvalidSwapExpressionError) IsUserError() {}

func (e *InvalidSwapExpressionError) Error() string {
	return fmt.Sprintf(
		"invalid %s-hand side of swap",
		e.Side.Name(),
	)
}

func (e *InvalidSwapExpressionError) SecondaryError() string {
	return "expected target expression"
}

// InvalidEventParameterTypeError

type InvalidEventParameterTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidEventParameterTypeError{}
var _ errors.UserError = &InvalidEventParameterTypeError{}

func (*InvalidEventParameterTypeError) isSemanticError() {}

func (*InvalidEventParameterTypeError) IsUserError() {}

func (e *InvalidEventParameterTypeError) Error() string {
	return fmt.Sprintf(
		"unsupported event parameter type: `%s`",
		e.Type.QualifiedString(),
	)
}

// InvalidEventUsageError

type InvalidEventUsageError struct {
	ast.Range
}

var _ SemanticError = &InvalidEventUsageError{}
var _ errors.UserError = &InvalidEventUsageError{}

func (*InvalidEventUsageError) isSemanticError() {}

func (*InvalidEventUsageError) IsUserError() {}

func (e *InvalidEventUsageError) Error() string {
	return "events can only be invoked in an `emit` statement"
}

// EmitNonEventError

type EmitNonEventError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &EmitNonEventError{}
var _ errors.UserError = &EmitNonEventError{}

func (*EmitNonEventError) isSemanticError() {}

func (*EmitNonEventError) IsUserError() {}

func (e *EmitNonEventError) Error() string {
	return fmt.Sprintf(
		"cannot emit non-event type: `%s`",
		e.Type.QualifiedString(),
	)
}

// EmitDefaultDestroyEventError

type EmitDefaultDestroyEventError struct {
	ast.Range
}

var _ SemanticError = &EmitDefaultDestroyEventError{}
var _ errors.UserError = &EmitDefaultDestroyEventError{}

func (*EmitDefaultDestroyEventError) isSemanticError() {}

func (*EmitDefaultDestroyEventError) IsUserError() {}

func (e *EmitDefaultDestroyEventError) Error() string {
	return "default destruction events may not be explicitly emitted"
}

// EmitImportedEventError

type EmitImportedEventError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &EmitImportedEventError{}
var _ errors.UserError = &EmitImportedEventError{}

func (*EmitImportedEventError) isSemanticError() {}

func (*EmitImportedEventError) IsUserError() {}

func (e *EmitImportedEventError) Error() string {
	return fmt.Sprintf(
		"cannot emit imported event type: `%s`",
		e.Type.QualifiedString(),
	)
}

// InvalidResourceAssignmentError

type InvalidResourceAssignmentError struct {
	ast.Range
}

var _ SemanticError = &InvalidResourceAssignmentError{}
var _ errors.UserError = &InvalidResourceAssignmentError{}
var _ errors.SecondaryError = &InvalidResourceAssignmentError{}

func (*InvalidResourceAssignmentError) isSemanticError() {}

func (*InvalidResourceAssignmentError) IsUserError() {}

func (e *InvalidResourceAssignmentError) Error() string {
	return "cannot assign to resource-typed target"
}

func (e *InvalidResourceAssignmentError) SecondaryError() string {
	return "consider force assigning (<-!) or swapping (<->)"
}

// ResourceFieldNotInvalidatedError

type ResourceFieldNotInvalidatedError struct {
	Type      Type
	FieldName string
	Pos       ast.Position
}

var _ SemanticError = &ResourceFieldNotInvalidatedError{}
var _ errors.UserError = &ResourceFieldNotInvalidatedError{}
var _ errors.SecondaryError = &ResourceFieldNotInvalidatedError{}

func (*ResourceFieldNotInvalidatedError) isSemanticError() {}

func (*ResourceFieldNotInvalidatedError) IsUserError() {}

func (e *ResourceFieldNotInvalidatedError) Error() string {
	return fmt.Sprintf(
		"field `%s` of type `%s` is not invalidated (moved or destroyed)",
		e.FieldName,
		e.Type.QualifiedString(),
	)
}

func (e *ResourceFieldNotInvalidatedError) SecondaryError() string {
	return "not invalidated"
}

func (e *ResourceFieldNotInvalidatedError) StartPosition() ast.Position {
	return e.Pos
}

func (e *ResourceFieldNotInvalidatedError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.FieldName)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// UninitializedFieldAccessError

type UninitializedFieldAccessError struct {
	Name string
	Pos  ast.Position
}

var _ SemanticError = &UninitializedFieldAccessError{}
var _ errors.UserError = &UninitializedFieldAccessError{}

func (*UninitializedFieldAccessError) isSemanticError() {}

func (*UninitializedFieldAccessError) IsUserError() {}

func (e *UninitializedFieldAccessError) Error() string {
	return fmt.Sprintf(
		"cannot access uninitialized field: `%s`",
		e.Name,
	)
}

func (e *UninitializedFieldAccessError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UninitializedFieldAccessError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// UnreachableStatementError

type UnreachableStatementError struct {
	ast.Range
}

var _ SemanticError = &UnreachableStatementError{}
var _ errors.UserError = &UnreachableStatementError{}
var _ errors.SecondaryError = &UnreachableStatementError{}

func (*UnreachableStatementError) isSemanticError() {}

func (*UnreachableStatementError) IsUserError() {}

func (e *UnreachableStatementError) Error() string {
	return "unreachable statement"
}

func (e *UnreachableStatementError) SecondaryError() string {
	return "consider removing this code"
}

// UninitializedUseError

type UninitializedUseError struct {
	Name string
	Pos  ast.Position
}

var _ SemanticError = &UninitializedUseError{}
var _ errors.UserError = &UninitializedUseError{}

func (*UninitializedUseError) isSemanticError() {}

func (*UninitializedUseError) IsUserError() {}

func (e *UninitializedUseError) Error() string {
	return fmt.Sprintf(
		"cannot use incompletely initialized value: `%s`",
		e.Name,
	)
}

func (e *UninitializedUseError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UninitializedUseError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// InvalidResourceArrayMemberError

type InvalidResourceArrayMemberError struct {
	Name            string
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidResourceArrayMemberError{}
var _ errors.UserError = &InvalidResourceArrayMemberError{}

func (*InvalidResourceArrayMemberError) isSemanticError() {}

func (*InvalidResourceArrayMemberError) IsUserError() {}

func (e *InvalidResourceArrayMemberError) Error() string {
	return fmt.Sprintf(
		"%s `%s` is not available for resource arrays",
		e.DeclarationKind.Name(),
		e.Name,
	)
}

// InvalidResourceDictionaryMemberError

type InvalidResourceDictionaryMemberError struct {
	Name            string
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidResourceDictionaryMemberError{}
var _ errors.UserError = &InvalidResourceDictionaryMemberError{}

func (*InvalidResourceDictionaryMemberError) isSemanticError() {}

func (*InvalidResourceDictionaryMemberError) IsUserError() {}

func (e *InvalidResourceDictionaryMemberError) Error() string {
	return fmt.Sprintf(
		"%s `%s` is not available for resource dictionaries",
		e.DeclarationKind.Name(),
		e.Name,
	)
}

// InvalidResourceOptionalMemberError

type InvalidResourceOptionalMemberError struct {
	Name            string
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidResourceOptionalMemberError{}
var _ errors.UserError = &InvalidResourceOptionalMemberError{}

func (*InvalidResourceOptionalMemberError) isSemanticError() {}

func (*InvalidResourceOptionalMemberError) IsUserError() {}

func (e *InvalidResourceOptionalMemberError) Error() string {
	return fmt.Sprintf(
		"%s `%s` is not available for resource optionals",
		e.DeclarationKind.Name(),
		e.Name,
	)
}

// NonReferenceTypeReferenceError

type NonReferenceTypeReferenceError struct {
	ActualType Type
	ast.Range
}

var _ SemanticError = &NonReferenceTypeReferenceError{}
var _ errors.UserError = &NonReferenceTypeReferenceError{}
var _ errors.SecondaryError = &NonReferenceTypeReferenceError{}

func (*NonReferenceTypeReferenceError) isSemanticError() {}

func (*NonReferenceTypeReferenceError) IsUserError() {}

func (e *NonReferenceTypeReferenceError) Error() string {
	return "cannot create reference"
}

func (e *NonReferenceTypeReferenceError) SecondaryError() string {
	return fmt.Sprintf(
		"expected reference type, got `%s`",
		e.ActualType.QualifiedString(),
	)
}

// ReferenceToAnOptionalError

type ReferenceToAnOptionalError struct {
	ReferencedOptionalType *OptionalType
	ast.Range
}

var _ SemanticError = &ReferenceToAnOptionalError{}
var _ errors.UserError = &ReferenceToAnOptionalError{}
var _ errors.SecondaryError = &ReferenceToAnOptionalError{}

func (*ReferenceToAnOptionalError) isSemanticError() {}

func (*ReferenceToAnOptionalError) IsUserError() {}

func (e *ReferenceToAnOptionalError) Error() string {
	return "cannot create reference"
}

func (e *ReferenceToAnOptionalError) SecondaryError() string {
	return fmt.Sprintf(
		"expected non-optional type, got `%s`. Consider taking a reference with type `%s`",
		e.ReferencedOptionalType.QualifiedString(),

		// Suggest taking the optional out of the reference type.
		NewOptionalType(
			nil,
			NewReferenceType(
				nil,
				UnauthorizedAccess,
				e.ReferencedOptionalType.Type,
			),
		),
	)
}

// InvalidResourceCreationError

type InvalidResourceCreationError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidResourceCreationError{}
var _ errors.UserError = &InvalidResourceCreationError{}

func (*InvalidResourceCreationError) isSemanticError() {}

func (*InvalidResourceCreationError) IsUserError() {}

func (e *InvalidResourceCreationError) Error() string {
	return fmt.Sprintf(
		"cannot create resource type outside of containing contract: `%s`",
		e.Type.QualifiedString(),
	)
}

// NonResourceTypeError

type NonResourceTypeError struct {
	ActualType Type
	ast.Range
}

var _ SemanticError = &NonResourceTypeError{}
var _ errors.UserError = &NonResourceTypeError{}
var _ errors.SecondaryError = &NonResourceTypeError{}

func (*NonResourceTypeError) isSemanticError() {}

func (*NonResourceTypeError) IsUserError() {}

func (e *NonResourceTypeError) Error() string {
	return "invalid type"
}

func (e *NonResourceTypeError) SecondaryError() string {
	return fmt.Sprintf(
		"expected resource type, got `%s`",
		e.ActualType.QualifiedString(),
	)
}

// InvalidAssignmentTargetError

type InvalidAssignmentTargetError struct {
	ast.Range
}

var _ SemanticError = &InvalidAssignmentTargetError{}
var _ errors.UserError = &InvalidAssignmentTargetError{}

func (*InvalidAssignmentTargetError) isSemanticError() {}

func (*InvalidAssignmentTargetError) IsUserError() {}

func (e *InvalidAssignmentTargetError) Error() string {
	return "cannot assign to unassignable expression"
}

// ResourceMethodBindingError

type ResourceMethodBindingError struct {
	ast.Range
}

var _ SemanticError = &ResourceMethodBindingError{}
var _ errors.UserError = &ResourceMethodBindingError{}

func (*ResourceMethodBindingError) isSemanticError() {}

func (*ResourceMethodBindingError) IsUserError() {}

func (e *ResourceMethodBindingError) Error() string {
	return "cannot create bound method for resource"
}

// InvalidDictionaryKeyTypeError

type InvalidDictionaryKeyTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidDictionaryKeyTypeError{}
var _ errors.UserError = &InvalidDictionaryKeyTypeError{}

func (*InvalidDictionaryKeyTypeError) isSemanticError() {}

func (*InvalidDictionaryKeyTypeError) IsUserError() {}

func (e *InvalidDictionaryKeyTypeError) Error() string {
	return fmt.Sprintf(
		"cannot use type as dictionary key type: `%s`",
		e.Type.QualifiedString(),
	)
}

// MissingFunctionBodyError

type MissingFunctionBodyError struct {
	Pos ast.Position
}

var _ SemanticError = &MissingFunctionBodyError{}
var _ errors.UserError = &MissingFunctionBodyError{}

func (*MissingFunctionBodyError) isSemanticError() {}

func (*MissingFunctionBodyError) IsUserError() {}

func (e *MissingFunctionBodyError) Error() string {
	return "missing function implementation"
}

func (e *MissingFunctionBodyError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingFunctionBodyError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidOptionalChainingError

type InvalidOptionalChainingError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidOptionalChainingError{}
var _ errors.UserError = &InvalidOptionalChainingError{}

func (*InvalidOptionalChainingError) isSemanticError() {}

func (*InvalidOptionalChainingError) IsUserError() {}

func (e *InvalidOptionalChainingError) Error() string {
	return fmt.Sprintf(
		"cannot use optional chaining: type `%s` is not optional",
		e.Type.QualifiedString(),
	)
}

// InvalidAccessError

type InvalidAccessError struct {
	Name                string
	RestrictingAccess   Access
	PossessedAccess     Access
	DeclarationKind     common.DeclarationKind
	SuggestEntitlements bool
	ast.Range
}

var _ SemanticError = &InvalidAccessError{}
var _ errors.UserError = &InvalidAccessError{}

func (*InvalidAccessError) isSemanticError() {}

func (*InvalidAccessError) IsUserError() {}

func (e *InvalidAccessError) Error() string {
	var possessedDescription string
	if e.PossessedAccess != nil {
		if e.PossessedAccess.Equal(UnauthorizedAccess) {
			possessedDescription = ", but reference is unauthorized"
		} else {
			possessedDescription = fmt.Sprintf(
				", but reference only has `%s` authorization",
				e.PossessedAccess.String(),
			)
		}
	}

	return fmt.Sprintf(
		"cannot access `%s`: %s requires `%s` authorization%s",
		e.Name,
		e.DeclarationKind.Name(),
		e.RestrictingAccess.String(),
		possessedDescription,
	)
}

// When e.PossessedAccess is a conjunctive entitlement set, we can suggest
// which additional entitlements it would need to be given in order to have
// e.RequiredAccess.
func (e *InvalidAccessError) SecondaryError() string {
	if !e.SuggestEntitlements || e.PossessedAccess == nil || e.RestrictingAccess == nil {
		return ""
	}
	possessedEntitlements, possessedOk := e.PossessedAccess.(EntitlementSetAccess)
	requiredEntitlements, requiredOk := e.RestrictingAccess.(EntitlementSetAccess)
	if !possessedOk && e.PossessedAccess.Equal(UnauthorizedAccess) {
		possessedOk = true
		// for this error reporting, model UnauthorizedAccess as an empty entitlement set
		possessedEntitlements = NewEntitlementSetAccess(nil, Conjunction)
	}
	if !possessedOk || !requiredOk || possessedEntitlements.SetKind != Conjunction {
		return ""
	}

	var sb strings.Builder

	enumerateEntitlements := func(len int, separator string) func(index int, key *EntitlementType, _ struct{}) {
		return func(index int, key *EntitlementType, _ struct{}) {
			fmt.Fprintf(&sb, "`%s`", key.QualifiedString())
			if index < len-2 {
				fmt.Fprint(&sb, ", ")
			} else if index < len-1 {
				if len > 2 {
					fmt.Fprint(&sb, ",")
				}
				fmt.Fprintf(&sb, " %s ", separator)
			}
		}
	}

	switch requiredEntitlements.SetKind {
	case Conjunction:
		// when both `possessed` and `required` are conjunctions, the missing set is simple set difference:
		// `missing` = `required` - `possessed`, and `missing` should be added to `possessed` to make `required`
		missingEntitlements := orderedmap.New[EntitlementOrderedSet](0)
		requiredEntitlements.Entitlements.Foreach(func(key *EntitlementType, _ struct{}) {
			if !possessedEntitlements.Entitlements.Contains(key) {
				missingEntitlements.Set(key, struct{}{})
			}
		})
		missingLen := missingEntitlements.Len()
		if missingLen == 1 {
			fmt.Fprint(&sb, "reference needs entitlement ")
			fmt.Fprintf(&sb, "`%s`", missingEntitlements.Newest().Key.QualifiedString())
		} else {
			fmt.Fprint(&sb, "reference needs all of entitlements ")
			missingEntitlements.ForeachWithIndex(enumerateEntitlements(missingLen, "and"))
		}

	case Disjunction:
		// when both `required` is a disjunction, we know `possessed` has none of the entitlements in it:
		// suggest adding one of those entitlements
		fmt.Fprint(&sb, "reference needs one of entitlements ")
		requiredEntitlementsSet := requiredEntitlements.Entitlements
		requiredLen := requiredEntitlementsSet.Len()
		// singleton-1 sets are always conjunctions
		requiredEntitlementsSet.ForeachWithIndex(enumerateEntitlements(requiredLen, "or"))

	default:
		panic(errors.NewUnreachableError())
	}

	return sb.String()
}

// InvalidAssignmentAccessError

type InvalidAssignmentAccessError struct {
	Name              string
	ContainerType     Type
	RestrictingAccess Access
	DeclarationKind   common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidAssignmentAccessError{}
var _ errors.UserError = &InvalidAssignmentAccessError{}
var _ errors.SecondaryError = &InvalidAssignmentAccessError{}

func (*InvalidAssignmentAccessError) isSemanticError() {}

func (*InvalidAssignmentAccessError) IsUserError() {}

func (e *InvalidAssignmentAccessError) Error() string {
	return fmt.Sprintf(
		"cannot assign to `%s`: %s has `%s` access",
		e.Name,
		e.DeclarationKind.Name(),
		e.RestrictingAccess.String(),
	)
}

func (e *InvalidAssignmentAccessError) SecondaryError() string {
	return fmt.Sprintf(
		"consider adding a setter function to %s",
		e.ContainerType.QualifiedString(),
	)
}

// UnauthorizedReferenceAssignmentError

type UnauthorizedReferenceAssignmentError struct {
	RequiredAccess [2]Access
	FoundAccess    Access
	ast.Range
}

var _ SemanticError = &UnauthorizedReferenceAssignmentError{}
var _ errors.UserError = &UnauthorizedReferenceAssignmentError{}
var _ errors.SecondaryError = &UnauthorizedReferenceAssignmentError{}

func (*UnauthorizedReferenceAssignmentError) isSemanticError() {}

func (*UnauthorizedReferenceAssignmentError) IsUserError() {}

func (e *UnauthorizedReferenceAssignmentError) Error() string {
	var foundAccess string
	if e.FoundAccess == UnauthorizedAccess {
		foundAccess = "non-auth"
	} else {
		foundAccess = fmt.Sprintf("(%s)", e.FoundAccess.String())
	}

	return fmt.Sprintf(
		"invalid assignment: can only assign to a reference with (%s) or (%s) access, but found a %s reference",
		e.RequiredAccess[0].String(),
		e.RequiredAccess[1].String(),
		foundAccess,
	)
}

func (e *UnauthorizedReferenceAssignmentError) SecondaryError() string {
	return fmt.Sprintf(
		"consider taking a reference with `%s` or `%s` access",
		e.RequiredAccess[0].String(),
		e.RequiredAccess[1].String(),
	)
}

// InvalidCharacterLiteralError

type InvalidCharacterLiteralError struct {
	Length int
	ast.Range
}

var _ SemanticError = &InvalidCharacterLiteralError{}
var _ errors.UserError = &InvalidCharacterLiteralError{}
var _ errors.SecondaryError = &InvalidCharacterLiteralError{}

func (*InvalidCharacterLiteralError) isSemanticError() {}

func (*InvalidCharacterLiteralError) IsUserError() {}

func (e *InvalidCharacterLiteralError) Error() string {
	return "character literal has invalid length"
}

func (e *InvalidCharacterLiteralError) SecondaryError() string {
	return fmt.Sprintf("expected 1, got %d",
		e.Length,
	)
}

// InvalidFailableResourceDowncastOutsideOptionalBindingError

type InvalidFailableResourceDowncastOutsideOptionalBindingError struct {
	ast.Range
}

var _ SemanticError = &InvalidFailableResourceDowncastOutsideOptionalBindingError{}
var _ errors.UserError = &InvalidFailableResourceDowncastOutsideOptionalBindingError{}

func (*InvalidFailableResourceDowncastOutsideOptionalBindingError) isSemanticError() {}

func (*InvalidFailableResourceDowncastOutsideOptionalBindingError) IsUserError() {}

func (e *InvalidFailableResourceDowncastOutsideOptionalBindingError) Error() string {
	return "cannot failably downcast resource type outside of optional binding"
}

// InvalidNonIdentifierFailableResourceDowncast

type InvalidNonIdentifierFailableResourceDowncast struct {
	ast.Range
}

var _ SemanticError = &InvalidNonIdentifierFailableResourceDowncast{}
var _ errors.UserError = &InvalidNonIdentifierFailableResourceDowncast{}
var _ errors.SecondaryError = &InvalidNonIdentifierFailableResourceDowncast{}

func (*InvalidNonIdentifierFailableResourceDowncast) isSemanticError() {}

func (*InvalidNonIdentifierFailableResourceDowncast) IsUserError() {}

func (e *InvalidNonIdentifierFailableResourceDowncast) Error() string {
	return "cannot failably downcast non-identifier resource"
}

func (e *InvalidNonIdentifierFailableResourceDowncast) SecondaryError() string {
	return "consider declaring a variable for this expression"
}

// ReadOnlyTargetAssignmentError

type ReadOnlyTargetAssignmentError struct {
	ast.Range
}

var _ SemanticError = &ReadOnlyTargetAssignmentError{}
var _ errors.UserError = &ReadOnlyTargetAssignmentError{}

func (*ReadOnlyTargetAssignmentError) isSemanticError() {}

func (*ReadOnlyTargetAssignmentError) IsUserError() {}

func (e *ReadOnlyTargetAssignmentError) Error() string {
	return "cannot assign to read-only target"
}

// InvalidTransactionBlockError

type InvalidTransactionBlockError struct {
	Name string
	Pos  ast.Position
}

var _ SemanticError = &InvalidTransactionBlockError{}
var _ errors.UserError = &InvalidTransactionBlockError{}
var _ errors.SecondaryError = &InvalidTransactionBlockError{}

func (*InvalidTransactionBlockError) isSemanticError() {}

func (*InvalidTransactionBlockError) IsUserError() {}

func (e *InvalidTransactionBlockError) Error() string {
	return "invalid transaction block"
}

func (e *InvalidTransactionBlockError) SecondaryError() string {
	return fmt.Sprintf(
		"expected `prepare` or `execute`, got `%s`",
		e.Name,
	)
}

func (e *InvalidTransactionBlockError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidTransactionBlockError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// TransactionMissingPrepareError

type TransactionMissingPrepareError struct {
	FirstFieldName string
	FirstFieldPos  ast.Position
}

var _ SemanticError = &TransactionMissingPrepareError{}
var _ errors.UserError = &TransactionMissingPrepareError{}

func (*TransactionMissingPrepareError) isSemanticError() {}

func (*TransactionMissingPrepareError) IsUserError() {}

func (e *TransactionMissingPrepareError) Error() string {
	return fmt.Sprintf(
		"transaction missing prepare function for field `%s`",
		e.FirstFieldName,
	)
}

func (e *TransactionMissingPrepareError) StartPosition() ast.Position {
	return e.FirstFieldPos
}

func (e *TransactionMissingPrepareError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.FirstFieldName)
	return e.FirstFieldPos.Shifted(memoryGauge, length-1)
}

// InvalidResourceTransactionParameterError

type InvalidResourceTransactionParameterError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidResourceTransactionParameterError{}
var _ errors.UserError = &InvalidResourceTransactionParameterError{}

func (*InvalidResourceTransactionParameterError) isSemanticError() {}

func (*InvalidResourceTransactionParameterError) IsUserError() {}

func (e *InvalidResourceTransactionParameterError) Error() string {
	return fmt.Sprintf(
		"transaction parameter must not be resource type: `%s`",
		e.Type.QualifiedString(),
	)
}

// InvalidNonImportableTransactionParameterTypeError

type InvalidNonImportableTransactionParameterTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidNonImportableTransactionParameterTypeError{}
var _ errors.UserError = &InvalidNonImportableTransactionParameterTypeError{}

func (*InvalidNonImportableTransactionParameterTypeError) isSemanticError() {}

func (*InvalidNonImportableTransactionParameterTypeError) IsUserError() {}

func (e *InvalidNonImportableTransactionParameterTypeError) Error() string {
	return fmt.Sprintf(
		"transaction parameter must be importable: `%s`",
		e.Type.QualifiedString(),
	)
}

// InvalidTransactionFieldAccessModifierError

type InvalidTransactionFieldAccessModifierError struct {
	Name   string
	Access ast.Access
	Pos    ast.Position
}

var _ SemanticError = &InvalidTransactionFieldAccessModifierError{}
var _ errors.UserError = &InvalidTransactionFieldAccessModifierError{}

func (*InvalidTransactionFieldAccessModifierError) isSemanticError() {}

func (*InvalidTransactionFieldAccessModifierError) IsUserError() {}

func (e *InvalidTransactionFieldAccessModifierError) Error() string {
	return fmt.Sprintf(
		"access modifier not allowed for transaction field `%s`: `%s`",
		e.Name,
		e.Access.Keyword(),
	)
}

func (e *InvalidTransactionFieldAccessModifierError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidTransactionFieldAccessModifierError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Access.Keyword())
	return e.Pos.Shifted(memoryGauge, length-1)
}

// InvalidTransactionPrepareParameterTypeError

type InvalidTransactionPrepareParameterTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidTransactionPrepareParameterTypeError{}
var _ errors.UserError = &InvalidTransactionPrepareParameterTypeError{}

func (*InvalidTransactionPrepareParameterTypeError) isSemanticError() {}

func (*InvalidTransactionPrepareParameterTypeError) IsUserError() {}

func (e *InvalidTransactionPrepareParameterTypeError) Error() string {
	return fmt.Sprintf(
		"prepare parameter must be subtype of `%s`, not `%s`",
		AccountReferenceType,
		e.Type.QualifiedString(),
	)
}

// InvalidNestedDeclarationError

type InvalidNestedDeclarationError struct {
	NestedDeclarationKind    common.DeclarationKind
	ContainerDeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidNestedDeclarationError{}
var _ errors.UserError = &InvalidNestedDeclarationError{}

func (*InvalidNestedDeclarationError) isSemanticError() {}

func (*InvalidNestedDeclarationError) IsUserError() {}

func (e *InvalidNestedDeclarationError) Error() string {
	return fmt.Sprintf(
		"%s declarations cannot be nested inside %s declarations",
		e.NestedDeclarationKind.Name(),
		e.ContainerDeclarationKind.Name(),
	)
}

// InvalidNestedTypeError

type InvalidNestedTypeError struct {
	Type *ast.NominalType
}

var _ SemanticError = &InvalidNestedTypeError{}
var _ errors.UserError = &InvalidNestedTypeError{}

func (*InvalidNestedTypeError) isSemanticError() {}

func (*InvalidNestedTypeError) IsUserError() {}

func (e *InvalidNestedTypeError) Error() string {
	return fmt.Sprintf(
		"type does not support nested types: `%s`",
		e.Type,
	)
}

func (e *InvalidNestedTypeError) StartPosition() ast.Position {
	return e.Type.StartPosition()
}

func (e *InvalidNestedTypeError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	return e.Type.EndPosition(memoryGauge)
}

// InvalidEnumCaseError

type InvalidEnumCaseError struct {
	ContainerDeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidEnumCaseError{}
var _ errors.UserError = &InvalidEnumCaseError{}

func (*InvalidEnumCaseError) isSemanticError() {}

func (*InvalidEnumCaseError) IsUserError() {}

func (e *InvalidEnumCaseError) Error() string {
	return fmt.Sprintf(
		"%s declaration does not allow enum cases",
		e.ContainerDeclarationKind.Name(),
	)
}

// InvalidNonEnumCaseError

type InvalidNonEnumCaseError struct {
	ContainerDeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidNonEnumCaseError{}
var _ errors.UserError = &InvalidNonEnumCaseError{}

func (*InvalidNonEnumCaseError) isSemanticError() {}

func (*InvalidNonEnumCaseError) IsUserError() {}

func (e *InvalidNonEnumCaseError) Error() string {
	return fmt.Sprintf(
		"%s declaration only allows enum cases",
		e.ContainerDeclarationKind.Name(),
	)
}

// InvalidTopLevelDeclarationError

type InvalidTopLevelDeclarationError struct {
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidTopLevelDeclarationError{}
var _ errors.UserError = &InvalidTopLevelDeclarationError{}

func (*InvalidTopLevelDeclarationError) isSemanticError() {}

func (*InvalidTopLevelDeclarationError) IsUserError() {}

func (e *InvalidTopLevelDeclarationError) Error() string {
	return fmt.Sprintf(
		"%s declarations are not valid at the top-level",
		e.DeclarationKind.Name(),
	)
}

// InvalidSelfInvalidationError

type InvalidSelfInvalidationError struct {
	InvalidationKind ResourceInvalidationKind
	ast.Range
}

var _ SemanticError = &InvalidSelfInvalidationError{}
var _ errors.UserError = &InvalidSelfInvalidationError{}

func (*InvalidSelfInvalidationError) isSemanticError() {}

func (*InvalidSelfInvalidationError) IsUserError() {}

func (e *InvalidSelfInvalidationError) Error() string {
	var action string
	switch e.InvalidationKind {
	case ResourceInvalidationKindMoveDefinite,
		ResourceInvalidationKindMoveTemporary,
		ResourceInvalidationKindMovePotential:

		action = "move"

	case ResourceInvalidationKindDestroyDefinite,
		ResourceInvalidationKindDestroyPotential:

		action = "destroy"

	default:
		panic(errors.NewUnreachableError())
	}
	return fmt.Sprintf("cannot %s `self`", action)
}

// InvalidMoveError

type InvalidMoveError struct {
	Name            string
	DeclarationKind common.DeclarationKind
	Pos             ast.Position
}

var _ SemanticError = &InvalidMoveError{}
var _ errors.UserError = &InvalidMoveError{}

func (*InvalidMoveError) isSemanticError() {}

func (*InvalidMoveError) IsUserError() {}

func (e *InvalidMoveError) Error() string {
	return fmt.Sprintf(
		"cannot move %s: `%s`",
		e.DeclarationKind.Name(),
		e.Name,
	)
}

func (e *InvalidMoveError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidMoveError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

// ConstantSizedArrayLiteralSizeError

type ConstantSizedArrayLiteralSizeError struct {
	ActualSize   int64
	ExpectedSize int64
	ast.Range
}

var _ SemanticError = &ConstantSizedArrayLiteralSizeError{}
var _ errors.UserError = &ConstantSizedArrayLiteralSizeError{}
var _ errors.SecondaryError = &ConstantSizedArrayLiteralSizeError{}

func (*ConstantSizedArrayLiteralSizeError) isSemanticError() {}

func (*ConstantSizedArrayLiteralSizeError) IsUserError() {}

func (e *ConstantSizedArrayLiteralSizeError) Error() string {
	return "incorrect number of array literal elements"
}

func (e *ConstantSizedArrayLiteralSizeError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %d, got %d",
		e.ExpectedSize,
		e.ActualSize,
	)
}

// InvalidIntersectedTypeError

type InvalidIntersectedTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidIntersectedTypeError{}
var _ errors.UserError = &InvalidIntersectedTypeError{}

func (*InvalidIntersectedTypeError) isSemanticError() {}

func (*InvalidIntersectedTypeError) IsUserError() {}

func (e *InvalidIntersectedTypeError) Error() string {
	return fmt.Sprintf(
		"intersection type with invalid non-interface type: `%s`",
		e.Type.QualifiedString(),
	)
}

// IntersectionCompositeKindMismatchError

type IntersectionCompositeKindMismatchError struct {
	CompositeKind         common.CompositeKind
	PreviousCompositeKind common.CompositeKind
	ast.Range
}

var _ SemanticError = &IntersectionCompositeKindMismatchError{}
var _ errors.UserError = &IntersectionCompositeKindMismatchError{}

func (*IntersectionCompositeKindMismatchError) isSemanticError() {}

func (*IntersectionCompositeKindMismatchError) IsUserError() {}

func (e *IntersectionCompositeKindMismatchError) Error() string {
	return fmt.Sprintf(
		"interface kind %s does not match previous interface kind %s",
		e.CompositeKind,
		e.PreviousCompositeKind,
	)
}

// InvalidIntersectionTypeDuplicateError

type InvalidIntersectionTypeDuplicateError struct {
	Type *InterfaceType
	ast.Range
}

var _ SemanticError = &InvalidIntersectionTypeDuplicateError{}
var _ errors.UserError = &InvalidIntersectionTypeDuplicateError{}

func (*InvalidIntersectionTypeDuplicateError) isSemanticError() {}

func (*InvalidIntersectionTypeDuplicateError) IsUserError() {}

func (e *InvalidIntersectionTypeDuplicateError) Error() string {
	return fmt.Sprintf(
		"duplicate intersected type: `%s`",
		e.Type.QualifiedString(),
	)
}

// IntersectionMemberClashError

type IntersectionMemberClashError struct {
	RedeclaringType       *InterfaceType
	OriginalDeclaringType *InterfaceType
	Name                  string
	ast.Range
}

var _ SemanticError = &IntersectionMemberClashError{}
var _ errors.UserError = &IntersectionMemberClashError{}

func (*IntersectionMemberClashError) isSemanticError() {}

func (*IntersectionMemberClashError) IsUserError() {}

func (e *IntersectionMemberClashError) Error() string {
	return fmt.Sprintf(
		"intersected type has member clash with previous intersected type `%s`: %s",
		e.OriginalDeclaringType.QualifiedString(),
		e.Name,
	)
}

// AmbiguousIntersectionTypeError

type AmbiguousIntersectionTypeError struct {
	ast.Range
}

var _ SemanticError = &AmbiguousIntersectionTypeError{}
var _ errors.UserError = &AmbiguousIntersectionTypeError{}

func (*AmbiguousIntersectionTypeError) isSemanticError() {}

func (*AmbiguousIntersectionTypeError) IsUserError() {}

func (e *AmbiguousIntersectionTypeError) Error() string {
	return "ambiguous intersection type"
}

// InvalidPathDomainError

type InvalidPathDomainError struct {
	ActualDomain string
	ast.Range
}

func (e *InvalidPathDomainError) Error() string {
	return fmt.Sprintf("invalid path domain %s", e.ActualDomain)
}

type InvalidPathIdentifierError struct {
	ActualIdentifier string
	ast.Range
}

var _ SemanticError = &InvalidPathDomainError{}
var _ errors.UserError = &InvalidPathDomainError{}
var _ errors.SecondaryError = &InvalidPathDomainError{}

func (*InvalidPathDomainError) isSemanticError() {}

func (*InvalidPathDomainError) IsUserError() {}

func (e *InvalidPathIdentifierError) Error() string {
	return fmt.Sprintf("invalid path identifier %s", e.ActualIdentifier)
}

var validPathDomainDescription = func() string {
	words := make([]string, 0, len(common.AllPathDomains))

	for _, domain := range common.AllPathDomains {
		words = append(words, fmt.Sprintf("`%s`", domain))
	}

	return common.EnumerateWords(words, "or")
}()

func (e *InvalidPathDomainError) SecondaryError() string {
	return fmt.Sprintf(
		"expected one of %s; got `%s`",
		validPathDomainDescription,
		e.ActualDomain,
	)
}

// InvalidTypeArgumentCountError

type InvalidTypeArgumentCountError struct {
	TypeParameterCount int
	TypeArgumentCount  int
	ast.Range
}

var _ SemanticError = &InvalidTypeArgumentCountError{}
var _ errors.UserError = &InvalidTypeArgumentCountError{}
var _ errors.SecondaryError = &InvalidTypeArgumentCountError{}

func (e *InvalidTypeArgumentCountError) isSemanticError() {}

func (*InvalidTypeArgumentCountError) IsUserError() {}

func (e *InvalidTypeArgumentCountError) Error() string {
	return "incorrect number of type arguments"
}

func (e *InvalidTypeArgumentCountError) SecondaryError() string {
	return fmt.Sprintf(
		"expected up to %d, got %d",
		e.TypeParameterCount,
		e.TypeArgumentCount,
	)
}

// MissingTypeArgumentError

type MissingTypeArgumentError struct {
	TypeArgumentName string
	ast.Range
}

var _ SemanticError = &MissingTypeArgumentError{}
var _ errors.UserError = &MissingTypeArgumentError{}

func (e *MissingTypeArgumentError) isSemanticError() {}

func (*MissingTypeArgumentError) IsUserError() {}

func (e *MissingTypeArgumentError) Error() string {
	return fmt.Sprintf("non-optional type argument %s missing", e.TypeArgumentName)
}

// InvalidTypeArgumentError

type InvalidTypeArgumentError struct {
	TypeArgumentName string
	Details          string
	ast.Range
}

var _ SemanticError = &InvalidTypeArgumentError{}
var _ errors.UserError = &InvalidTypeArgumentError{}

func (*InvalidTypeArgumentError) isSemanticError() {}

func (*InvalidTypeArgumentError) IsUserError() {}

func (e *InvalidTypeArgumentError) Error() string {
	return fmt.Sprintf("type argument %s invalid", e.TypeArgumentName)
}

func (e *InvalidTypeArgumentError) SecondaryError() string {
	return e.Details
}

// TypeParameterTypeInferenceError

type TypeParameterTypeInferenceError struct {
	Name string
	ast.Range
}

var _ SemanticError = &TypeParameterTypeInferenceError{}
var _ errors.UserError = &TypeParameterTypeInferenceError{}

func (e *TypeParameterTypeInferenceError) isSemanticError() {}

func (*TypeParameterTypeInferenceError) IsUserError() {}

func (e *TypeParameterTypeInferenceError) Error() string {
	return fmt.Sprintf(
		"cannot infer type parameter: `%s`",
		e.Name,
	)
}

// InvalidConstantSizedTypeBaseError

type InvalidConstantSizedTypeBaseError struct {
	ActualBase   int
	ExpectedBase int
	ast.Range
}

var _ SemanticError = &InvalidConstantSizedTypeBaseError{}
var _ errors.UserError = &InvalidConstantSizedTypeBaseError{}
var _ errors.SecondaryError = &InvalidConstantSizedTypeBaseError{}

func (e *InvalidConstantSizedTypeBaseError) isSemanticError() {}

func (*InvalidConstantSizedTypeBaseError) IsUserError() {}

func (e *InvalidConstantSizedTypeBaseError) Error() string {
	return "invalid base for constant sized type size"
}

func (e *InvalidConstantSizedTypeBaseError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %d, got %d",
		e.ActualBase,
		e.ExpectedBase,
	)
}

// InvalidConstantSizedTypeSizeError

type InvalidConstantSizedTypeSizeError struct {
	ActualSize     *big.Int
	ExpectedMinInt *big.Int
	ExpectedMaxInt *big.Int
	ast.Range
}

var _ SemanticError = &InvalidConstantSizedTypeSizeError{}
var _ errors.UserError = &InvalidConstantSizedTypeSizeError{}
var _ errors.SecondaryError = &InvalidConstantSizedTypeSizeError{}

func (*InvalidConstantSizedTypeSizeError) isSemanticError() {}

func (*InvalidConstantSizedTypeSizeError) IsUserError() {}

func (e *InvalidConstantSizedTypeSizeError) Error() string {
	return "invalid size for constant sized type"
}

func (e *InvalidConstantSizedTypeSizeError) SecondaryError() string {
	return fmt.Sprintf(
		"expected value in range [%s, %s], got %s",
		e.ExpectedMinInt,
		e.ExpectedMaxInt,
		e.ActualSize,
	)
}

// UnsupportedResourceForLoopError

type UnsupportedResourceForLoopError struct {
	ast.Range
}

var _ SemanticError = &UnsupportedResourceForLoopError{}
var _ errors.UserError = &UnsupportedResourceForLoopError{}

func (*UnsupportedResourceForLoopError) isSemanticError() {}

func (*UnsupportedResourceForLoopError) IsUserError() {}

func (e *UnsupportedResourceForLoopError) Error() string {
	return "cannot loop over resources"
}

// TypeParameterTypeMismatchError

type TypeParameterTypeMismatchError struct {
	TypeParameter *TypeParameter
	ExpectedType  Type
	ActualType    Type
	ast.Range
}

var _ SemanticError = &TypeParameterTypeMismatchError{}
var _ errors.UserError = &TypeParameterTypeMismatchError{}
var _ errors.SecondaryError = &TypeParameterTypeMismatchError{}

func (*TypeParameterTypeMismatchError) isSemanticError() {}

func (*TypeParameterTypeMismatchError) IsUserError() {}

func (e *TypeParameterTypeMismatchError) Error() string {
	return "mismatched types for type parameter"
}

func (e *TypeParameterTypeMismatchError) SecondaryError() string {
	expected, actual := ErrorMessageExpectedActualTypes(
		e.ExpectedType,
		e.ActualType,
	)

	return fmt.Sprintf(
		"type parameter %s is bound to `%s`, but got `%s` here",
		e.TypeParameter.Name,
		expected,
		actual,
	)
}

// UnparameterizedTypeInstantiationError

type UnparameterizedTypeInstantiationError struct {
	ActualTypeArgumentCount int
	ast.Range
}

var _ SemanticError = &UnparameterizedTypeInstantiationError{}
var _ errors.UserError = &UnparameterizedTypeInstantiationError{}
var _ errors.SecondaryError = &UnparameterizedTypeInstantiationError{}

func (*UnparameterizedTypeInstantiationError) isSemanticError() {}

func (*UnparameterizedTypeInstantiationError) IsUserError() {}

func (e *UnparameterizedTypeInstantiationError) Error() string {
	return "cannot instantiate non-parameterized type"
}

func (e *UnparameterizedTypeInstantiationError) SecondaryError() string {
	return fmt.Sprintf(
		"expected no type arguments, got %d",
		e.ActualTypeArgumentCount,
	)
}

// TypeAnnotationRequiredError

type TypeAnnotationRequiredError struct {
	Cause string
	Pos   ast.Position
}

var _ SemanticError = &TypeAnnotationRequiredError{}
var _ errors.UserError = &TypeAnnotationRequiredError{}

func (*TypeAnnotationRequiredError) isSemanticError() {}

func (*TypeAnnotationRequiredError) IsUserError() {}

func (e *TypeAnnotationRequiredError) Error() string {
	if e.Cause != "" {
		return fmt.Sprintf(
			"%s requires an explicit type annotation",
			e.Cause,
		)
	}
	return "explicit type annotation required"
}

func (e *TypeAnnotationRequiredError) StartPosition() ast.Position {
	return e.Pos
}

func (e *TypeAnnotationRequiredError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// CyclicImportsError

type CyclicImportsError struct {
	Location common.Location
	ast.Range
}

var _ SemanticError = &CyclicImportsError{}
var _ errors.UserError = &CyclicImportsError{}

func (*CyclicImportsError) isSemanticError() {}

func (*CyclicImportsError) IsUserError() {}

func (e *CyclicImportsError) Error() string {
	return fmt.Sprintf("cyclic import of `%s`", e.Location)
}

// SwitchDefaultPositionError

type SwitchDefaultPositionError struct {
	ast.Range
}

var _ SemanticError = &SwitchDefaultPositionError{}
var _ errors.UserError = &SwitchDefaultPositionError{}

func (*SwitchDefaultPositionError) isSemanticError() {}

func (*SwitchDefaultPositionError) IsUserError() {}

func (e *SwitchDefaultPositionError) Error() string {
	return "the 'default' case must appear at the end of a 'switch' statement"
}

// MissingSwitchCaseStatementsError

type MissingSwitchCaseStatementsError struct {
	Pos ast.Position
}

var _ SemanticError = &MissingSwitchCaseStatementsError{}
var _ errors.UserError = &MissingSwitchCaseStatementsError{}

func (*MissingSwitchCaseStatementsError) isSemanticError() {}

func (*MissingSwitchCaseStatementsError) IsUserError() {}

func (e *MissingSwitchCaseStatementsError) Error() string {
	return "switch cases must have at least one statement"
}

func (e *MissingSwitchCaseStatementsError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingSwitchCaseStatementsError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// MissingEntryPointError

type MissingEntryPointError struct {
	Expected string
}

var _ errors.UserError = &MissingEntryPointError{}

func (*MissingEntryPointError) IsUserError() {}

func (e *MissingEntryPointError) Error() string {
	return fmt.Sprintf("missing entry point: expected '%s'", e.Expected)
}

// InvalidEntryPointError

type InvalidEntryPointTypeError struct {
	Type Type
}

var _ errors.UserError = &InvalidEntryPointTypeError{}

func (*InvalidEntryPointTypeError) IsUserError() {}

func (e *InvalidEntryPointTypeError) Error() string {
	return fmt.Sprintf(
		"invalid entry point type: `%s`",
		e.Type.QualifiedString(),
	)
}

type PurityError struct {
	ast.Range
}

func (e *PurityError) Error() string {
	return "Impure operation performed in view context"
}

var _ SemanticError = &PurityError{}
var _ errors.UserError = &PurityError{}

func (*PurityError) IsUserError() {}

func (*PurityError) isSemanticError() {}

// InvalidatedResourceReferenceError

type InvalidatedResourceReferenceError struct {
	Invalidation ResourceInvalidation
	ast.Range
}

var _ SemanticError = &InvalidatedResourceReferenceError{}
var _ errors.UserError = &InvalidatedResourceReferenceError{}

func (*InvalidatedResourceReferenceError) isSemanticError() {}

func (*InvalidatedResourceReferenceError) IsUserError() {}

func (e *InvalidatedResourceReferenceError) Error() string {
	return "invalid reference: referenced resource may have been moved or destroyed"
}

func (e *InvalidatedResourceReferenceError) ErrorNotes() []errors.ErrorNote {
	invalidation := e.Invalidation
	return []errors.ErrorNote{
		newPreviousResourceInvalidationNote(invalidation),
	}
}

// InvalidEntitlementAccessError
type InvalidEntitlementAccessError struct {
	Pos ast.Position
}

var _ SemanticError = &InvalidEntitlementAccessError{}
var _ errors.UserError = &InvalidEntitlementAccessError{}

func (*InvalidEntitlementAccessError) isSemanticError() {}

func (*InvalidEntitlementAccessError) IsUserError() {}

func (e *InvalidEntitlementAccessError) Error() string {
	return "only struct or resource members may be declared with entitlement access"
}

func (e *InvalidEntitlementAccessError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidEntitlementAccessError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidEntitlementMappingTypeError
type InvalidEntitlementMappingTypeError struct {
	Type Type
	Pos  ast.Position
}

var _ SemanticError = &InvalidEntitlementMappingTypeError{}
var _ errors.UserError = &InvalidEntitlementMappingTypeError{}

func (*InvalidEntitlementMappingTypeError) isSemanticError() {}

func (*InvalidEntitlementMappingTypeError) IsUserError() {}

func (e *InvalidEntitlementMappingTypeError) Error() string {
	return fmt.Sprintf("`%s` is not an entitlement map type", e.Type.QualifiedString())
}

func (e *InvalidEntitlementMappingTypeError) SecondaryError() string {
	return "consider removing the `mapping` keyword"
}

func (e *InvalidEntitlementMappingTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidEntitlementMappingTypeError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidNonEntitlementTypeInMapError
type InvalidNonEntitlementTypeInMapError struct {
	Pos ast.Position
}

var _ SemanticError = &InvalidNonEntitlementTypeInMapError{}
var _ errors.UserError = &InvalidNonEntitlementTypeInMapError{}

func (*InvalidNonEntitlementTypeInMapError) isSemanticError() {}

func (*InvalidNonEntitlementTypeInMapError) IsUserError() {}

func (e *InvalidNonEntitlementTypeInMapError) Error() string {
	return "cannot use non-entitlement type in entitlement mapping"
}

func (e *InvalidNonEntitlementTypeInMapError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidNonEntitlementTypeInMapError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidMappingAccessError
type InvalidMappingAccessError struct {
	Pos ast.Position
}

var _ SemanticError = &InvalidMappingAccessError{}
var _ errors.UserError = &InvalidMappingAccessError{}

func (*InvalidMappingAccessError) isSemanticError() {}

func (*InvalidMappingAccessError) IsUserError() {}

func (e *InvalidMappingAccessError) Error() string {
	return "access(mapping ...) may only be used in structs and resources"
}

func (e *InvalidMappingAccessError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidMappingAccessError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidMappingAccessMemberTypeError
type InvalidMappingAccessMemberTypeError struct {
	Pos ast.Position
}

var _ SemanticError = &InvalidMappingAccessMemberTypeError{}
var _ errors.UserError = &InvalidMappingAccessMemberTypeError{}

func (*InvalidMappingAccessMemberTypeError) isSemanticError() {}

func (*InvalidMappingAccessMemberTypeError) IsUserError() {}

func (e *InvalidMappingAccessMemberTypeError) Error() string {
	return "invalid type for access(mapping ...) declaration"
}

func (e *InvalidMappingAccessMemberTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidMappingAccessMemberTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidNonEntitlementAccessError
type InvalidNonEntitlementAccessError struct {
	ast.Range
}

var _ SemanticError = &InvalidNonEntitlementAccessError{}
var _ errors.UserError = &InvalidNonEntitlementAccessError{}

func (*InvalidNonEntitlementAccessError) isSemanticError() {}

func (*InvalidNonEntitlementAccessError) IsUserError() {}

func (e *InvalidNonEntitlementAccessError) Error() string {
	return "only entitlements may be used in access modifiers"
}

// MappingAccessMissingKeywordError
type MappingAccessMissingKeywordError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &MappingAccessMissingKeywordError{}
var _ errors.UserError = &MappingAccessMissingKeywordError{}

func (*MappingAccessMissingKeywordError) isSemanticError() {}

func (*MappingAccessMissingKeywordError) IsUserError() {}

func (e *MappingAccessMissingKeywordError) Error() string {
	return "entitlement mapping access modifiers require the `mapping` keyword preceding the name of the map"
}

func (e *MappingAccessMissingKeywordError) SecondaryError() string {
	return fmt.Sprintf("replace `%s` with `mapping %s`", e.Type.QualifiedString(), e.Type.QualifiedString())
}

// DirectEntitlementAnnotationError
type DirectEntitlementAnnotationError struct {
	ast.Range
}

var _ SemanticError = &DirectEntitlementAnnotationError{}
var _ errors.UserError = &DirectEntitlementAnnotationError{}

func (*DirectEntitlementAnnotationError) isSemanticError() {}

func (*DirectEntitlementAnnotationError) IsUserError() {}

func (e *DirectEntitlementAnnotationError) Error() string {
	return "cannot use an entitlement type outside of an `access` declaration or `auth` modifier"
}

// UnrepresentableEntitlementMapOutputError
type UnrepresentableEntitlementMapOutputError struct {
	Input EntitlementSetAccess
	Map   *EntitlementMapType
	ast.Range
}

var _ SemanticError = &UnrepresentableEntitlementMapOutputError{}
var _ errors.UserError = &UnrepresentableEntitlementMapOutputError{}

func (*UnrepresentableEntitlementMapOutputError) isSemanticError() {}

func (*UnrepresentableEntitlementMapOutputError) IsUserError() {}

func (e *UnrepresentableEntitlementMapOutputError) Error() string {
	return fmt.Sprintf(
		"cannot map `%s` through `%s` because the output is unrepresentable",
		e.Input.String(),
		e.Map.QualifiedString(),
	)
}

func (e *UnrepresentableEntitlementMapOutputError) SecondaryError() string {
	return fmt.Sprintf(
		"this usually occurs because the input set is disjunctive and `%s` is one-to-many",
		e.Map.QualifiedString(),
	)
}

func (e *UnrepresentableEntitlementMapOutputError) StartPosition() ast.Position {
	return e.StartPos
}

func (e *UnrepresentableEntitlementMapOutputError) EndPosition(common.MemoryGauge) ast.Position {
	return e.EndPos
}

// InvalidEntitlementMappingInclusionError
type InvalidEntitlementMappingInclusionError struct {
	Map          *EntitlementMapType
	IncludedType Type
	ast.Range
}

var _ SemanticError = &InvalidEntitlementMappingInclusionError{}
var _ errors.UserError = &InvalidEntitlementMappingInclusionError{}

func (*InvalidEntitlementMappingInclusionError) isSemanticError() {}

func (*InvalidEntitlementMappingInclusionError) IsUserError() {}

func (e *InvalidEntitlementMappingInclusionError) Error() string {
	return fmt.Sprintf(
		"cannot include `%s` in the definition of `%s`, as it is not an entitlement map",
		e.IncludedType.QualifiedString(),
		e.Map.QualifiedIdentifier(),
	)
}

// DuplicateEntitlementMappingInclusionError
type DuplicateEntitlementMappingInclusionError struct {
	Map          *EntitlementMapType
	IncludedType *EntitlementMapType
	ast.Range
}

var _ SemanticError = &DuplicateEntitlementMappingInclusionError{}
var _ errors.UserError = &DuplicateEntitlementMappingInclusionError{}

func (*DuplicateEntitlementMappingInclusionError) isSemanticError() {}

func (*DuplicateEntitlementMappingInclusionError) IsUserError() {}

func (e *DuplicateEntitlementMappingInclusionError) Error() string {
	return fmt.Sprintf(
		"`%s` is already included in the definition of `%s`",
		e.IncludedType.QualifiedIdentifier(),
		e.Map.QualifiedIdentifier(),
	)
}

// CyclicEntitlementMappingError
type CyclicEntitlementMappingError struct {
	Map          *EntitlementMapType
	IncludedType *EntitlementMapType
	ast.Range
}

var _ SemanticError = &CyclicEntitlementMappingError{}
var _ errors.UserError = &CyclicEntitlementMappingError{}

func (*CyclicEntitlementMappingError) isSemanticError() {}

func (*CyclicEntitlementMappingError) IsUserError() {}

func (e *CyclicEntitlementMappingError) Error() string {
	return fmt.Sprintf(
		"cannot include `%s` in the definition of `%s`, as it would create a cyclical mapping",
		e.IncludedType.QualifiedIdentifier(),
		e.Map.QualifiedIdentifier(),
	)
}

// InvalidBaseTypeError

type InvalidBaseTypeError struct {
	BaseType   Type
	Attachment *CompositeType
	ast.Range
}

var _ SemanticError = &InvalidBaseTypeError{}
var _ errors.UserError = &InvalidBaseTypeError{}

func (*InvalidBaseTypeError) isSemanticError() {}

func (*InvalidBaseTypeError) IsUserError() {}

func (e *InvalidBaseTypeError) Error() string {
	return fmt.Sprintf(
		"cannot use `%s` as the base type for attachment `%s`",
		e.BaseType.QualifiedString(),
		e.Attachment.QualifiedString(),
	)
}

// InvalidAttachmentAnnotationError

type InvalidAttachmentAnnotationError struct {
	ast.Range
}

var _ SemanticError = &InvalidAttachmentAnnotationError{}
var _ errors.UserError = &InvalidAttachmentAnnotationError{}

func (*InvalidAttachmentAnnotationError) isSemanticError() {}

func (*InvalidAttachmentAnnotationError) IsUserError() {}

func (e *InvalidAttachmentAnnotationError) Error() string {
	return "cannot refer directly to attachment type"
}

// InvalidAttachmentConstructorError

type InvalidAttachmentUsageError struct {
	ast.Range
}

var _ SemanticError = &InvalidAttachmentUsageError{}
var _ errors.UserError = &InvalidAttachmentUsageError{}

func (*InvalidAttachmentUsageError) isSemanticError() {}

func (*InvalidAttachmentUsageError) IsUserError() {}

func (*InvalidAttachmentUsageError) Error() string {
	return "cannot construct attachment outside of an `attach` expression"
}

// AttachNonAttachmentError

type AttachNonAttachmentError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &AttachNonAttachmentError{}
var _ errors.UserError = &AttachNonAttachmentError{}

func (*AttachNonAttachmentError) isSemanticError() {}

func (*AttachNonAttachmentError) IsUserError() {}

func (e *AttachNonAttachmentError) Error() string {
	return fmt.Sprintf(
		"cannot attach non-attachment type: `%s`",
		e.Type.QualifiedString(),
	)
}

// AttachToInvalidTypeError
type AttachToInvalidTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &AttachToInvalidTypeError{}
var _ errors.UserError = &AttachToInvalidTypeError{}

func (*AttachToInvalidTypeError) isSemanticError() {}

func (*AttachToInvalidTypeError) IsUserError() {}

func (e *AttachToInvalidTypeError) Error() string {
	return fmt.Sprintf(
		"cannot attach attachment to type `%s`, as it is not valid for this base type",
		e.Type.QualifiedString(),
	)
}

// InvalidAttachmentRemoveError
type InvalidAttachmentRemoveError struct {
	Attachment Type
	BaseType   Type
	ast.Range
}

var _ SemanticError = &InvalidAttachmentRemoveError{}
var _ errors.UserError = &InvalidAttachmentRemoveError{}

func (*InvalidAttachmentRemoveError) isSemanticError() {}

func (*InvalidAttachmentRemoveError) IsUserError() {}

func (e *InvalidAttachmentRemoveError) Error() string {
	if e.BaseType == nil {
		return fmt.Sprintf(
			"cannot remove `%s`, as it is not an attachment type",
			e.Attachment.QualifiedString(),
		)
	}
	return fmt.Sprintf(
		"cannot remove `%s` from type `%s`, as this attachment cannot exist on this base type",
		e.Attachment.QualifiedString(),
		e.BaseType.QualifiedString(),
	)
}

// InvalidTypeIndexingError
type InvalidTypeIndexingError struct {
	IndexingExpression ast.Expression
	BaseType           Type
	ast.Range
}

var _ SemanticError = &InvalidTypeIndexingError{}
var _ errors.UserError = &InvalidTypeIndexingError{}

func (*InvalidTypeIndexingError) isSemanticError() {}

func (*InvalidTypeIndexingError) IsUserError() {}

func (e *InvalidTypeIndexingError) Error() string {
	return fmt.Sprintf(
		"cannot index `%s` with `%s`, as it is not an valid type index for this type",
		e.BaseType.QualifiedString(),
		e.IndexingExpression.String(),
	)
}

// InvalidAttachmentEntitlementError
type InvalidAttachmentEntitlementError struct {
	Attachment         *CompositeType
	BaseType           Type
	InvalidEntitlement *EntitlementType
	Pos                ast.Position
}

var _ SemanticError = &InvalidAttachmentEntitlementError{}
var _ errors.UserError = &InvalidAttachmentEntitlementError{}

func (*InvalidAttachmentEntitlementError) isSemanticError() {}

func (*InvalidAttachmentEntitlementError) IsUserError() {}

func (e *InvalidAttachmentEntitlementError) Error() string {
	entitlementDescription := "entitlements"
	if e.InvalidEntitlement != nil {
		entitlementDescription = fmt.Sprintf("`%s`", e.InvalidEntitlement.QualifiedIdentifier())
	}

	return fmt.Sprintf("cannot use %s in the access modifier for a member in `%s`",
		entitlementDescription,
		e.Attachment.QualifiedIdentifier())
}

func (e *InvalidAttachmentEntitlementError) SecondaryError() string {
	return fmt.Sprintf("`%s` must appear in the base type `%s`",
		e.InvalidEntitlement.QualifiedIdentifier(),
		e.BaseType.String(),
	)
}

func (e *InvalidAttachmentEntitlementError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidAttachmentEntitlementError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// DefaultDestroyEventInNonResourceError

type DefaultDestroyEventInNonResourceError struct {
	Kind string
	ast.Range
}

var _ SemanticError = &DefaultDestroyEventInNonResourceError{}
var _ errors.UserError = &DefaultDestroyEventInNonResourceError{}

func (*DefaultDestroyEventInNonResourceError) isSemanticError() {}

func (*DefaultDestroyEventInNonResourceError) IsUserError() {}

func (e *DefaultDestroyEventInNonResourceError) Error() string {
	return fmt.Sprintf(
		"cannot declare default destruction event in %s",
		e.Kind,
	)
}

type DefaultDestroyInvalidArgumentKind int

const (
	NonDictionaryIndexExpression DefaultDestroyInvalidArgumentKind = iota
	ReferenceTypedMemberAccess
	InvalidIdentifier
	InvalidExpression
)

// DefaultDestroyInvalidArgumentError

type DefaultDestroyInvalidArgumentError struct {
	ast.Range
	Kind DefaultDestroyInvalidArgumentKind
}

var _ SemanticError = &DefaultDestroyInvalidArgumentError{}
var _ errors.UserError = &DefaultDestroyInvalidArgumentError{}

func (*DefaultDestroyInvalidArgumentError) isSemanticError() {}

func (*DefaultDestroyInvalidArgumentError) IsUserError() {}

func (e *DefaultDestroyInvalidArgumentError) Error() string {
	return "Invalid default destroy event argument"
}

func (e *DefaultDestroyInvalidArgumentError) SecondaryError() string {
	switch e.Kind {
	case NonDictionaryIndexExpression:
		return "Indexed accesses may only be performed on dictionaries"
	case ReferenceTypedMemberAccess:
		return "Member accesses in arguments may not contain reference types"
	case InvalidIdentifier:
		return "Identifiers other than `self` or `base` may not appear in arguments"
	case InvalidExpression:
		return "Arguments must be literals, member access expressions on `self` or `base`, indexed access expressions on dictionaries, or attachment accesses"
	}
	return ""
}

// DefaultDestroyInvalidParameterError

type DefaultDestroyInvalidParameterError struct {
	ParamType Type
	ast.Range
}

var _ SemanticError = &DefaultDestroyInvalidParameterError{}
var _ errors.UserError = &DefaultDestroyInvalidParameterError{}

func (*DefaultDestroyInvalidParameterError) isSemanticError() {}

func (*DefaultDestroyInvalidParameterError) IsUserError() {}

func (e *DefaultDestroyInvalidParameterError) Error() string {
	return fmt.Sprintf("`%s` is not a valid parameter type for a default destroy event", e.ParamType.QualifiedString())
}

// InvalidTypeParameterizedNonNativeFunctionError

type InvalidTypeParameterizedNonNativeFunctionError struct {
	ast.Range
}

var _ SemanticError = &InvalidTypeParameterizedNonNativeFunctionError{}
var _ errors.UserError = &InvalidTypeParameterizedNonNativeFunctionError{}

func (*InvalidTypeParameterizedNonNativeFunctionError) isSemanticError() {}

func (*InvalidTypeParameterizedNonNativeFunctionError) IsUserError() {}

func (e *InvalidTypeParameterizedNonNativeFunctionError) Error() string {
	return "invalid type parameters in non-native function"
}

// NestedReferenceError
type NestedReferenceError struct {
	Type *ReferenceType
	ast.Range
}

var _ SemanticError = &NestedReferenceError{}
var _ errors.UserError = &NestedReferenceError{}

func (*NestedReferenceError) isSemanticError() {}

func (*NestedReferenceError) IsUserError() {}

func (e *NestedReferenceError) Error() string {
	return fmt.Sprintf("cannot create a nested reference to value of type %s", e.Type.QualifiedString())
}

// ResultVariableConflictError

type ResultVariableConflictError struct {
	Kind                common.DeclarationKind
	Pos                 ast.Position
	ReturnTypeRange     ast.Range
	PostConditionsRange ast.Range
}

var _ SemanticError = &ResultVariableConflictError{}
var _ errors.UserError = &ResultVariableConflictError{}
var _ errors.SecondaryError = &ResultVariableConflictError{}

func (*ResultVariableConflictError) isSemanticError() {}

func (*ResultVariableConflictError) IsUserError() {}

func (e *ResultVariableConflictError) Error() string {
	return fmt.Sprintf(
		"cannot declare %[1]s `%[2]s`: it conflicts with the `%[2]s` variable for the post-conditions",
		e.Kind.Name(),
		ResultIdentifier,
	)
}

func (*ResultVariableConflictError) SecondaryError() string {
	return "consider renaming the variable"
}

func (e *ResultVariableConflictError) StartPosition() ast.Position {
	return e.Pos
}

func (e *ResultVariableConflictError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(ResultIdentifier)
	return e.Pos.Shifted(memoryGauge, length-1)
}

func (e *ResultVariableConflictError) ErrorNotes() []errors.ErrorNote {
	return []errors.ErrorNote{
		ResultVariableReturnTypeNote{
			Range: e.ReturnTypeRange,
		},
		ResultVariablePostConditionsNote{
			Range: e.PostConditionsRange,
		},
	}
}

// ResultVariableReturnTypeNote

type ResultVariableReturnTypeNote struct {
	ast.Range
}

var _ errors.ErrorNote = ResultVariableReturnTypeNote{}

func (ResultVariableReturnTypeNote) Message() string {
	return "non-Void return type declared here"
}

// ResultVariablePostConditionsNote

type ResultVariablePostConditionsNote struct {
	ast.Range
}

var _ errors.ErrorNote = ResultVariablePostConditionsNote{}

func (ResultVariablePostConditionsNote) Message() string {
	return "post-conditions declared here"
}

// InvocationTypeInferenceError

type InvocationTypeInferenceError struct {
	ast.Range
}

var _ SemanticError = &InvocationTypeInferenceError{}
var _ errors.UserError = &InvocationTypeInferenceError{}

func (e *InvocationTypeInferenceError) isSemanticError() {}

func (*InvocationTypeInferenceError) IsUserError() {}

func (e *InvocationTypeInferenceError) Error() string {
	return "cannot infer type of invocation"
}

// UnconvertableTypeError

type UnconvertableTypeError struct {
	Type ast.Type
	ast.Range
}

var _ SemanticError = &UnconvertableTypeError{}
var _ errors.UserError = &UnconvertableTypeError{}

func (e *UnconvertableTypeError) isSemanticError() {}

func (*UnconvertableTypeError) IsUserError() {}

func (e *UnconvertableTypeError) Error() string {
	return fmt.Sprintf("cannot convert type `%s`", e.Type)
}

// InvalidMappingAuthorizationError

type InvalidMappingAuthorizationError struct {
	ast.Range
}

var _ SemanticError = &InvalidMappingAuthorizationError{}
var _ errors.UserError = &InvalidMappingAuthorizationError{}

func (*InvalidMappingAuthorizationError) isSemanticError() {}

func (*InvalidMappingAuthorizationError) IsUserError() {}

func (e *InvalidMappingAuthorizationError) Error() string {
	return "auth(mapping ...) is not supported"
}
