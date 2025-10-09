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
		"cannot check unsupported %s operation: %#q",
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

func (*InvalidPragmaError) Error() string {
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
var _ errors.SecondaryError = &RedeclarationError{}
var _ errors.HasDocumentationLink = &RedeclarationError{}

func (*RedeclarationError) isSemanticError() {}

func (*RedeclarationError) IsUserError() {}

func (e *RedeclarationError) Error() string {
	return fmt.Sprintf(
		"cannot redeclare %s: %#q is already declared",
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

func (*RedeclarationError) SecondaryError() string {
	return "each declaration must have a unique name in its scope; " +
		"rename this declaration or the conflicting one"
}

func (*RedeclarationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/constants-and-variables"
}

// RedeclarationNote

type RedeclarationNote struct {
	ast.Range
}

func (RedeclarationNote) Message() string {
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
var _ errors.HasDocumentationLink = &NotDeclaredError{}

func (*NotDeclaredError) isSemanticError() {}

func (*NotDeclaredError) IsUserError() {}

func (e *NotDeclaredError) Error() string {
	return fmt.Sprintf(
		"cannot find %s in this scope: %#q",
		e.ExpectedKind.Name(),
		e.Name,
	)
}

func (*NotDeclaredError) SecondaryError() string {
	return "not found in this scope; check for typos or declare it"
}

func (e *NotDeclaredError) StartPosition() ast.Position {
	return e.Pos
}

func (e *NotDeclaredError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

func (*NotDeclaredError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/constants-and-variables"
}

// AssignmentToConstantError

type AssignmentToConstantError struct {
	Name string
	ast.Range
}

var _ SemanticError = &AssignmentToConstantError{}
var _ errors.UserError = &AssignmentToConstantError{}
var _ errors.SecondaryError = &AssignmentToConstantError{}
var _ errors.HasDocumentationLink = &AssignmentToConstantError{}

func (*AssignmentToConstantError) isSemanticError() {}

func (*AssignmentToConstantError) IsUserError() {}

func (e *AssignmentToConstantError) Error() string {
	return fmt.Sprintf("cannot assign to constant: %#q", e.Name)
}

func (e *AssignmentToConstantError) SecondaryError() string {
	return fmt.Sprintf(
		"constants (`let`) cannot be reassigned; "+
			"consider changing the declaration of %#q to a variable (`var`)",
		e.Name,
	)
}

func (*AssignmentToConstantError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/constants-and-variables"
}

// TypeMismatchError

// TODO: Add suggested fix for type mismatch

type TypeMismatchError struct {
	ExpectedType Type
	ActualType   Type
	Expression   ast.Expression
	ast.Range
}

var _ SemanticError = &TypeMismatchError{}
var _ errors.UserError = &TypeMismatchError{}
var _ errors.SecondaryError = &TypeMismatchError{}
var _ errors.HasDocumentationLink = &TypeMismatchError{}

func (*TypeMismatchError) isSemanticError() {}

func (*TypeMismatchError) IsUserError() {}

func (*TypeMismatchError) Error() string {
	return "mismatched types"
}

func (e *TypeMismatchError) SecondaryError() string {
	expected, actual := ErrorMessageExpectedActualTypes(
		e.ExpectedType,
		e.ActualType,
	)

	return fmt.Sprintf(
		"expected %#q, got %#q; "+
			"check the expression's type or convert it to the expected type",
		expected,
		actual,
	)
}

func (*TypeMismatchError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
}

// TypeMismatchWithDescriptionError

// TODO: Add suggested fix for type mismatch

type TypeMismatchWithDescriptionError struct {
	ActualType              Type
	ExpectedTypeDescription string
	ast.Range
}

var _ SemanticError = &TypeMismatchWithDescriptionError{}
var _ errors.UserError = &TypeMismatchWithDescriptionError{}
var _ errors.SecondaryError = &TypeMismatchWithDescriptionError{}
var _ errors.HasDocumentationLink = &TypeMismatchWithDescriptionError{}

func (*TypeMismatchWithDescriptionError) isSemanticError() {}

func (*TypeMismatchWithDescriptionError) IsUserError() {}

func (*TypeMismatchWithDescriptionError) Error() string {
	return "mismatched types"
}

func (e *TypeMismatchWithDescriptionError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %s, got %#q; "+
			"check the expression's type or convert it to the expected type",
		e.ExpectedTypeDescription,
		e.ActualType.QualifiedString(),
	)
}

func (*TypeMismatchWithDescriptionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
}

// NotIndexableTypeError

type NotIndexableTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &NotIndexableTypeError{}
var _ errors.UserError = &NotIndexableTypeError{}
var _ errors.SecondaryError = &NotIndexableTypeError{}
var _ errors.HasDocumentationLink = &NotIndexableTypeError{}

func (*NotIndexableTypeError) isSemanticError() {}

func (*NotIndexableTypeError) IsUserError() {}

func (e *NotIndexableTypeError) Error() string {
	return fmt.Sprintf(
		"cannot index into value which has type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*NotIndexableTypeError) SecondaryError() string {
	return "only arrays, dictionaries, and other indexable types can be indexed"
}

func (*NotIndexableTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/arrays"
}

// NotIndexingAssignableTypeError

type NotIndexingAssignableTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &NotIndexingAssignableTypeError{}
var _ errors.UserError = &NotIndexingAssignableTypeError{}
var _ errors.SecondaryError = &NotIndexingAssignableTypeError{}
var _ errors.HasDocumentationLink = &NotIndexingAssignableTypeError{}

func (*NotIndexingAssignableTypeError) isSemanticError() {}

func (*NotIndexingAssignableTypeError) IsUserError() {}

func (e *NotIndexingAssignableTypeError) Error() string {
	return fmt.Sprintf(
		"cannot assign into value which has type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*NotIndexingAssignableTypeError) SecondaryError() string {
	return "this type supports indexing but not assignment through indexing; " +
		"check if the type is mutable or if assignment is supported"
}

func (*NotIndexingAssignableTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/arrays"
}

// NotEquatableTypeError

type NotEquatableTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &NotEquatableTypeError{}
var _ errors.UserError = &NotEquatableTypeError{}
var _ errors.SecondaryError = &NotEquatableTypeError{}
var _ errors.HasDocumentationLink = &NotEquatableTypeError{}

func (*NotEquatableTypeError) isSemanticError() {}

func (*NotEquatableTypeError) IsUserError() {}

func (e *NotEquatableTypeError) Error() string {
	return fmt.Sprintf(
		"cannot compare value which has type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*NotEquatableTypeError) SecondaryError() string {
	return "only equatable types can be compared; " +
		"ensure the types in the comparison are equatable"
}

func (*NotEquatableTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
}

// NotCallableError

type NotCallableError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &NotCallableError{}
var _ errors.UserError = &NotCallableError{}
var _ errors.SecondaryError = &NotCallableError{}
var _ errors.HasDocumentationLink = &NotCallableError{}

func (*NotCallableError) isSemanticError() {}

func (*NotCallableError) IsUserError() {}

func (e *NotCallableError) Error() string {
	return fmt.Sprintf(
		"cannot call type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*NotCallableError) SecondaryError() string {
	return "only functions can be called; " +
		"ensure the value is a function or remove the call"
}

func (*NotCallableError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
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
var _ errors.HasDocumentationLink = &InsufficientArgumentsError{}

func (*InsufficientArgumentsError) isSemanticError() {}

func (*InsufficientArgumentsError) IsUserError() {}

func (*InsufficientArgumentsError) Error() string {
	return "too few arguments"
}

func (e *InsufficientArgumentsError) SecondaryError() string {
	return fmt.Sprintf(
		"expected at least %d, got %d; "+
			"add the missing arguments to match the function signature",
		e.MinCount,
		e.ActualCount,
	)
}

func (*InsufficientArgumentsError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
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
var _ errors.HasDocumentationLink = &ExcessiveArgumentsError{}

func (*ExcessiveArgumentsError) isSemanticError() {}

func (*ExcessiveArgumentsError) IsUserError() {}

func (*ExcessiveArgumentsError) Error() string {
	return "too many arguments"
}

func (e *ExcessiveArgumentsError) SecondaryError() string {
	return fmt.Sprintf(
		"expected up to %d, got %d; "+
			"remove the extra arguments to match the function signature",
		e.MaxCount,
		e.ActualCount,
	)
}

func (*ExcessiveArgumentsError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// MissingArgumentLabelError

type MissingArgumentLabelError struct {
	ExpectedArgumentLabel string
	ast.Range
}

var _ SemanticError = &MissingArgumentLabelError{}
var _ errors.UserError = &MissingArgumentLabelError{}
var _ errors.SecondaryError = &MissingArgumentLabelError{}
var _ errors.HasDocumentationLink = &MissingArgumentLabelError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingArgumentLabelError{}

func (*MissingArgumentLabelError) isSemanticError() {}

func (*MissingArgumentLabelError) IsUserError() {}

func (e *MissingArgumentLabelError) Error() string {
	return fmt.Sprintf(
		"missing argument label: `%s:`",
		e.ExpectedArgumentLabel,
	)
}

func (e *MissingArgumentLabelError) SecondaryError() string {
	return fmt.Sprintf(
		"add the argument label `%s:` before the argument",
		e.ExpectedArgumentLabel,
	)
}

func (*MissingArgumentLabelError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

func (e *MissingArgumentLabelError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert argument label",
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
var _ errors.HasDocumentationLink = &IncorrectArgumentLabelError{}

func (*IncorrectArgumentLabelError) isSemanticError() {}

func (*IncorrectArgumentLabelError) IsUserError() {}

func (*IncorrectArgumentLabelError) Error() string {
	return "incorrect argument label"
}

func (e *IncorrectArgumentLabelError) SecondaryError() string {
	if e.ExpectedArgumentLabel != "" {
		return fmt.Sprintf(
			"expected %#q, got %#q; replace the current argument label",
			e.ExpectedArgumentLabel,
			e.ActualArgumentLabel,
		)
	} else {
		return fmt.Sprintf(
			"expected no label, got %#q; remove the argument label",
			e.ActualArgumentLabel,
		)
	}
}

func (e *IncorrectArgumentLabelError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	if len(e.ExpectedArgumentLabel) > 0 {
		return []errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Replace argument label",
				TextEdits: []ast.TextEdit{
					{
						Replacement: e.ExpectedArgumentLabel + ":",
						Range:       e.Range,
					},
				},
			},
		}
	} else {

		return []errors.SuggestedFix[ast.TextEdit]{
			{
				Message: "Remove argument label",
				TextEdits: []ast.TextEdit{
					{
						Replacement: "",
						Range: ast.Range{
							StartPos: e.StartPos,
							EndPos:   e.EndPos.AttachRight(code),
						},
					},
				},
			},
		}
	}
}

func (*IncorrectArgumentLabelError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
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
var _ errors.HasDocumentationLink = &InvalidUnaryOperandError{}

func (*InvalidUnaryOperandError) isSemanticError() {}

func (*InvalidUnaryOperandError) IsUserError() {}

func (e *InvalidUnaryOperandError) Error() string {
	return fmt.Sprintf(
		"cannot apply unary operation %#q to type",
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
			"expected %#q, got %#q; ensure the operand has the correct type",
			expected,
			actual,
		)
	} else {
		return fmt.Sprintf(
			"expected %s, got %#q; ensure the operand has the correct type",
			e.ExpectedTypeDescription,
			e.ActualType.QualifiedString(),
		)
	}
}

func (*InvalidUnaryOperandError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators"
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
var _ errors.HasDocumentationLink = &InvalidBinaryOperandError{}

func (*InvalidBinaryOperandError) isSemanticError() {}

func (*InvalidBinaryOperandError) IsUserError() {}

func (e *InvalidBinaryOperandError) Error() string {
	return fmt.Sprintf(
		"cannot apply binary operation %#q to %s-hand type",
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
		"expected %#q, got %#q; check the expression's type or convert it to the expected type",
		expected,
		actual,
	)
}

func (*InvalidBinaryOperandError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators"
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
var _ errors.SecondaryError = &InvalidBinaryOperandsError{}
var _ errors.HasDocumentationLink = &InvalidBinaryOperandsError{}

func (*InvalidBinaryOperandsError) isSemanticError() {}

func (*InvalidBinaryOperandsError) IsUserError() {}

func (e *InvalidBinaryOperandsError) Error() string {
	return fmt.Sprintf(
		"cannot apply binary operation %#q to types %#q and %#q",
		e.Operation.Symbol(),
		e.LeftType.QualifiedString(),
		e.RightType.QualifiedString(),
	)
}

func (e *InvalidBinaryOperandsError) SecondaryError() string {
	switch e.Operation {
	case ast.OperationOr,
		ast.OperationAnd:

		return fmt.Sprintf(
			"logical operator %#q requires both operands to be of type `Bool`, "+
				"but got %#q and %#q; ensure both operands are booleans",
			e.Operation.Symbol(),
			e.LeftType.QualifiedString(),
			e.RightType.QualifiedString(),
		)

	case ast.OperationPlus,
		ast.OperationMinus,
		ast.OperationMul,
		ast.OperationDiv,
		ast.OperationMod:

		return fmt.Sprintf(
			"arithmetic operators require numeric operands of the same type; "+
				"got %#q and %#q which are incompatible; "+
				"ensure both operands are numbers of the same type",
			e.LeftType.QualifiedString(),
			e.RightType.QualifiedString(),
		)

	case ast.OperationBitwiseOr,
		ast.OperationBitwiseAnd,
		ast.OperationBitwiseXor,
		ast.OperationBitwiseLeftShift,
		ast.OperationBitwiseRightShift:

		return fmt.Sprintf(
			"bitwise operators require integer operands of the same type; "+
				"got %#q and %#q which are incompatible; "+
				"ensure both operands are integers of the same type",
			e.LeftType.QualifiedString(),
			e.RightType.QualifiedString(),
		)

	case ast.OperationLess,
		ast.OperationLessEqual,
		ast.OperationGreater,
		ast.OperationGreaterEqual:
		return fmt.Sprintf(
			"comparison operators require comparable operands of the same type; "+
				"got %#q and %#q which are incompatible; "+
				"ensure both operands are comparable and of the same type",
			e.LeftType.QualifiedString(),
			e.RightType.QualifiedString(),
		)

	case ast.OperationEqual,
		ast.OperationNotEqual:
		return fmt.Sprintf(
			"equality operators require compatible types; "+
				"got %#q and %#q which cannot be compared for equality; "+
				"ensure both operands are of compatible types",
			e.LeftType.QualifiedString(),
			e.RightType.QualifiedString(),
		)
	default:
		return fmt.Sprintf(
			"binary operation %#q cannot be applied to operands of types %#q and %#q; "+
				"check the types of the operands",
			e.Operation.Symbol(),
			e.LeftType.QualifiedString(),
			e.RightType.QualifiedString(),
		)
	}
}

func (*InvalidBinaryOperandsError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators"
}

// InvalidNilCoalescingRightResourceOperandError

type InvalidNilCoalescingRightResourceOperandError struct {
	ast.Range
}

var _ SemanticError = &InvalidNilCoalescingRightResourceOperandError{}
var _ errors.UserError = &InvalidNilCoalescingRightResourceOperandError{}
var _ errors.SecondaryError = &InvalidNilCoalescingRightResourceOperandError{}
var _ errors.HasDocumentationLink = &InvalidNilCoalescingRightResourceOperandError{}

func (*InvalidNilCoalescingRightResourceOperandError) isSemanticError() {}

func (*InvalidNilCoalescingRightResourceOperandError) IsUserError() {}

func (*InvalidNilCoalescingRightResourceOperandError) Error() string {
	return "invalid right-hand operand for nil-coalescing operator (`??`)"
}

func (*InvalidNilCoalescingRightResourceOperandError) SecondaryError() string {
	return "the right-hand operand of a nil-coalescing operator cannot be a resource; " +
		"use a non-resource type instead"
}

func (*InvalidNilCoalescingRightResourceOperandError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/optional-operators#nil-coalescing-operator-"
}

// InvalidConditionalResourceOperandError

type InvalidConditionalResourceOperandError struct {
	ast.Range
}

var _ SemanticError = &InvalidConditionalResourceOperandError{}
var _ errors.UserError = &InvalidConditionalResourceOperandError{}
var _ errors.SecondaryError = &InvalidConditionalResourceOperandError{}
var _ errors.HasDocumentationLink = &InvalidConditionalResourceOperandError{}

func (*InvalidConditionalResourceOperandError) isSemanticError() {}

func (*InvalidConditionalResourceOperandError) IsUserError() {}

func (*InvalidConditionalResourceOperandError) Error() string {
	return "invalid resource in conditional expression"
}

func (*InvalidConditionalResourceOperandError) SecondaryError() string {
	return "operands of a conditional expression cannot be resources; " +
		"use non-resource types instead"
}

func (*InvalidConditionalResourceOperandError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/bitwise-ternary-operators#ternary-conditional-operator"
}

// ControlStatementError

type ControlStatementError struct {
	ControlStatement common.ControlStatement
	ast.Range
}

var _ SemanticError = &ControlStatementError{}
var _ errors.UserError = &ControlStatementError{}
var _ errors.SecondaryError = &ControlStatementError{}
var _ errors.HasDocumentationLink = &ControlStatementError{}

func (*ControlStatementError) isSemanticError() {}

func (*ControlStatementError) IsUserError() {}

func (e *ControlStatementError) Error() string {
	return fmt.Sprintf(
		"invalid control statement placement: %#q",
		e.ControlStatement.Symbol(),
	)
}

func (e *ControlStatementError) SecondaryError() string {
	switch e.ControlStatement {
	case common.ControlStatementBreak:
		return "`break` can only be used inside a loop or switch statement; " +
			"move this statement to a valid context"
	case common.ControlStatementContinue:
		return "`continue` can only be used inside a loop statement; " +
			"move this statement to a valid context"
	default:
		return fmt.Sprintf(
			"%#q can only be used within a valid control flow body; "+
				"move this statement to a valid context",
			e.ControlStatement.Symbol(),
		)
	}
}

func (*ControlStatementError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/control-flow"
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
var _ errors.SecondaryError = &InvalidAccessModifierError{}
var _ errors.HasDocumentationLink = &InvalidAccessModifierError{}

func (*InvalidAccessModifierError) isSemanticError() {}

func (*InvalidAccessModifierError) IsUserError() {}

func (e *InvalidAccessModifierError) Error() string {
	var explanation string
	if e.Explanation != "" {
		explanation = fmt.Sprintf(": %s", e.Explanation)
	}

	if e.Access.Equal(PrimitiveAccess(ast.AccessNotSpecified)) {
		return fmt.Sprintf(
			"invalid effective access modifier for %s%s",
			e.DeclarationKind.Name(),
			explanation,
		)
	} else {
		return fmt.Sprintf(
			"invalid access modifier for %s: %#q%s",
			e.DeclarationKind.Name(),
			e.Access.String(),
			explanation,
		)
	}
}

func (*InvalidAccessModifierError) SecondaryError() string {
	return "the access modifier is not valid for this declaration"
}

func (*InvalidAccessModifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
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

var _ SemanticError = &MissingAccessModifierError{}
var _ errors.UserError = &MissingAccessModifierError{}
var _ errors.SecondaryError = &MissingAccessModifierError{}
var _ errors.HasDocumentationLink = &MissingAccessModifierError{}

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

func (*MissingAccessModifierError) SecondaryError() string {
	return "an access modifier is required for this declaration; " +
		"add an access modifier, like e.g. `access(all)` or `access(self)`"
}

func (*MissingAccessModifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

// InvalidStaticModifierError

type InvalidStaticModifierError struct {
	ast.Range
}

var _ SemanticError = &InvalidStaticModifierError{}
var _ errors.UserError = &InvalidStaticModifierError{}
var _ errors.SecondaryError = &InvalidStaticModifierError{}

func (*InvalidStaticModifierError) isSemanticError() {}

func (*InvalidStaticModifierError) IsUserError() {}

func (*InvalidStaticModifierError) Error() string {
	return "invalid `static` modifier for declaration"
}

func (*InvalidStaticModifierError) SecondaryError() string {
	return "the `static` modifier can only be used on functions and fields within composite types; " +
		"remove the modifier or apply it to a valid declaration"
}

// InvalidNativeModifierError

type InvalidNativeModifierError struct {
	ast.Range
}

var _ SemanticError = &InvalidNativeModifierError{}
var _ errors.UserError = &InvalidNativeModifierError{}
var _ errors.SecondaryError = &InvalidNativeModifierError{}

func (*InvalidNativeModifierError) isSemanticError() {}

func (*InvalidNativeModifierError) IsUserError() {}

func (*InvalidNativeModifierError) Error() string {
	return "invalid `native` modifier for declaration"
}

func (*InvalidNativeModifierError) SecondaryError() string {
	return "the `native` modifier can only be used on function declarations; " +
		"remove the modifier or use a function declaration"
}

// NativeFunctionWithImplementationError

type NativeFunctionWithImplementationError struct {
	ast.Range
}

var _ SemanticError = &NativeFunctionWithImplementationError{}
var _ errors.UserError = &NativeFunctionWithImplementationError{}
var _ errors.SecondaryError = &NativeFunctionWithImplementationError{}

func (*NativeFunctionWithImplementationError) isSemanticError() {}

func (*NativeFunctionWithImplementationError) IsUserError() {}

func (*NativeFunctionWithImplementationError) Error() string {
	return "native function must not have an implementation"
}

func (*NativeFunctionWithImplementationError) SecondaryError() string {
	return "native functions cannot have a function body; " +
		"remove the function body or the `native` modifier"
}

// InvalidNameError

type InvalidNameError struct {
	Name string
	Pos  ast.Position
}

var _ SemanticError = &InvalidNameError{}
var _ errors.UserError = &InvalidNameError{}
var _ errors.SecondaryError = &InvalidNameError{}
var _ errors.HasDocumentationLink = &InvalidNameError{}

func (*InvalidNameError) isSemanticError() {}

func (*InvalidNameError) IsUserError() {}

func (e *InvalidNameError) Error() string {
	return fmt.Sprintf("invalid name: %#q", e.Name)
}

func (*InvalidNameError) SecondaryError() string {
	return "names must start with a letter or underscore and contain only letters, digits, and underscores; " +
		"rename the identifier according to these rules"
}

func (*InvalidNameError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/syntax#identifiers"
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
var _ errors.SecondaryError = &UnknownSpecialFunctionError{}
var _ errors.HasDocumentationLink = &UnknownSpecialFunctionError{}

func (*UnknownSpecialFunctionError) isSemanticError() {}

func (*UnknownSpecialFunctionError) IsUserError() {}

func (*UnknownSpecialFunctionError) Error() string {
	return "unknown special function; did you mean `init` or forget the `fun` keyword?"
}

func (*UnknownSpecialFunctionError) SecondaryError() string {
	return "use `init` for constructors or add the `fun` keyword for regular functions"
}

func (*UnknownSpecialFunctionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

func (e *UnknownSpecialFunctionError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UnknownSpecialFunctionError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidFieldVariableKindError

type InvalidFieldVariableKindError struct {
	Kind ast.VariableKind
	ast.Range
}

var _ SemanticError = &InvalidFieldVariableKindError{}
var _ errors.UserError = &InvalidFieldVariableKindError{}
var _ errors.SecondaryError = &InvalidFieldVariableKindError{}
var _ errors.HasDocumentationLink = &InvalidFieldVariableKindError{}

func (*InvalidFieldVariableKindError) isSemanticError() {}

func (*InvalidFieldVariableKindError) IsUserError() {}

func (e *InvalidFieldVariableKindError) Error() string {
	if e.Kind == ast.VariableKindNotSpecified {
		return "missing variable kind"
	}
	return fmt.Sprintf("invalid variable kind: %#q", e.Kind.Name())
}

func (*InvalidFieldVariableKindError) SecondaryError() string {
	return "field declarations must specify a variable kind; " +
		"use `let` for constants or `var` for variables"
}

func (*InvalidFieldVariableKindError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/constants-and-variables"
}

// InvalidDeclarationError

type InvalidDeclarationError struct {
	Identifier string
	Kind       common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidDeclarationError{}
var _ errors.UserError = &InvalidDeclarationError{}
var _ errors.SecondaryError = &InvalidDeclarationError{}
var _ errors.HasDocumentationLink = &InvalidDeclarationError{}

func (*InvalidDeclarationError) isSemanticError() {}

func (*InvalidDeclarationError) IsUserError() {}

func (e *InvalidDeclarationError) Error() string {
	if e.Identifier != "" {
		return fmt.Sprintf(
			"cannot declare %s here: %#q",
			e.Kind.Name(),
			e.Identifier,
		)
	}

	return fmt.Sprintf("cannot declare %s here", e.Kind.Name())
}

func (e *InvalidDeclarationError) SecondaryError() string {
	return fmt.Sprintf(
		"only function and variable declarations are allowed in this scope, "+
			"%s declarations must be at the top level or within composite types; "+
			"move this declaration to a valid scope",
		e.Kind.Name(),
	)
}

func (*InvalidDeclarationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/constants-and-variables"
}

// MissingInitializerError

type MissingInitializerError struct {
	ContainerType  Type
	FirstFieldName string
	FirstFieldPos  ast.Position
}

var _ SemanticError = &MissingInitializerError{}
var _ errors.UserError = &MissingInitializerError{}
var _ errors.SecondaryError = &MissingInitializerError{}
var _ errors.HasDocumentationLink = &MissingInitializerError{}

func (*MissingInitializerError) isSemanticError() {}

func (*MissingInitializerError) IsUserError() {}

func (e *MissingInitializerError) Error() string {
	return fmt.Sprintf(
		"missing initializer for field %#q in type %#q",
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

func (*MissingInitializerError) SecondaryError() string {
	return "all fields of a composite type must be initialized; " +
		"add an `init` function to initialize the fields"
}

func (*MissingInitializerError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
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
		"value of type %#q has no member %#q",
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
	message := "the member is not defined on the type; " +
		"check for typos or if the member exists on the type"
	if optionalMember := e.findOptionalMember(); optionalMember != "" {
		message += fmt.Sprintf(
			"; the type is optional, consider optional-chaining: `?.%s`",
			optionalMember,
		)
	}
	if closestMember := e.findClosestMember(); closestMember != "" {
		message += fmt.Sprintf(
			"; did you mean member %#q instead?",
			closestMember,
		)
	}
	return message
}

func (e *NotDeclaredMemberError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	optionalMember := e.findOptionalMember()
	if optionalMember == "" {
		return nil
	}

	accessPos := e.Expression.AccessEndPos

	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Use optional chaining",
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
var _ errors.SecondaryError = &AssignmentToConstantMemberError{}
var _ errors.HasDocumentationLink = &AssignmentToConstantMemberError{}

func (*AssignmentToConstantMemberError) isSemanticError() {}

func (*AssignmentToConstantMemberError) IsUserError() {}

func (e *AssignmentToConstantMemberError) Error() string {
	return fmt.Sprintf("cannot assign to constant member: %#q", e.Name)
}

func (*AssignmentToConstantMemberError) SecondaryError() string {
	return "constant members cannot be reassigned after initialization; " +
		"consider using a variable field (`var`) instead"
}

func (*AssignmentToConstantMemberError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/constants-and-variables"
}

// FieldReinitializationError
type FieldReinitializationError struct {
	Name string
	ast.Range
}

var _ SemanticError = &FieldReinitializationError{}
var _ errors.UserError = &FieldReinitializationError{}
var _ errors.SecondaryError = &FieldReinitializationError{}
var _ errors.HasDocumentationLink = &FieldReinitializationError{}

func (*FieldReinitializationError) isSemanticError() {}

func (*FieldReinitializationError) IsUserError() {}

func (e *FieldReinitializationError) Error() string {
	return fmt.Sprintf("invalid reinitialization of field: %#q", e.Name)
}

func (*FieldReinitializationError) SecondaryError() string {
	return "fields can only be initialized once; " +
		"remove the duplicate initialization, or make the field variable (`var`)"
}

func (*FieldReinitializationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types#composite-type-fields"
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
var _ errors.HasDocumentationLink = &FieldUninitializedError{}

func (*FieldUninitializedError) isSemanticError() {}

func (*FieldUninitializedError) IsUserError() {}

func (e *FieldUninitializedError) Error() string {
	return fmt.Sprintf(
		"missing initialization of field %#q in type %#q",
		e.Name,
		e.ContainerType.QualifiedString(),
	)
}

func (*FieldUninitializedError) SecondaryError() string {
	return "all fields must be initialized when creating a composite type; " +
		"add an initializer or provide a default value"
}

func (*FieldUninitializedError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types#composite-type-fields"
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
var _ errors.HasDocumentationLink = &FieldTypeNotStorableError{}

func (*FieldTypeNotStorableError) isSemanticError() {}

func (*FieldTypeNotStorableError) IsUserError() {}

func (e *FieldTypeNotStorableError) Error() string {
	return fmt.Sprintf(
		"field %#q has non-storable type: %#q",
		e.Name,
		e.Type,
	)
}

func (*FieldTypeNotStorableError) SecondaryError() string {
	return "all contract fields must be storable; use a storable type for this field"
}

func (*FieldTypeNotStorableError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types#composite-type-fields"
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
var _ errors.SecondaryError = &FunctionExpressionInConditionError{}
var _ errors.HasDocumentationLink = &FunctionExpressionInConditionError{}

func (*FunctionExpressionInConditionError) isSemanticError() {}

func (*FunctionExpressionInConditionError) IsUserError() {}

func (*FunctionExpressionInConditionError) Error() string {
	return "condition contains function expression"
}

func (*FunctionExpressionInConditionError) SecondaryError() string {
	return "pre- and post-conditions may not contain function expressions"
}

func (*FunctionExpressionInConditionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/pre-and-post-conditions"
}

// InvalidEmitConditionError

type InvalidEmitConditionError struct {
	ast.Range
}

var _ SemanticError = &InvalidEmitConditionError{}
var _ errors.UserError = &InvalidEmitConditionError{}
var _ errors.SecondaryError = &InvalidEmitConditionError{}
var _ errors.HasDocumentationLink = &InvalidEmitConditionError{}

func (*InvalidEmitConditionError) isSemanticError() {}

func (*InvalidEmitConditionError) IsUserError() {}

func (*InvalidEmitConditionError) Error() string {
	return "invalid emit condition"
}

func (*InvalidEmitConditionError) SecondaryError() string {
	return "emit conditions must contain a valid event invocation expression; " +
		"use `emit EventName()` syntax"
}

func (*InvalidEmitConditionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/events"
}

// MissingReturnValueError

type MissingReturnValueError struct {
	ExpectedValueType Type
	ast.Range
}

var _ SemanticError = &MissingReturnValueError{}
var _ errors.UserError = &MissingReturnValueError{}
var _ errors.SecondaryError = &MissingReturnValueError{}
var _ errors.HasDocumentationLink = &MissingReturnValueError{}

func (*MissingReturnValueError) isSemanticError() {}

func (*MissingReturnValueError) IsUserError() {}

func (e *MissingReturnValueError) Error() string {
	var typeDescription string
	if e.ExpectedValueType.IsInvalidType() {
		typeDescription = "non-void"
	} else {
		typeDescription = fmt.Sprintf("%#q", e.ExpectedValueType.QualifiedString())
	}

	return fmt.Sprintf(
		"missing value in return from function with %s return type",
		typeDescription,
	)
}

func (*MissingReturnValueError) SecondaryError() string {
	return "a function with a non-void return type must return a value; " +
		"add a value to the return statement"
}

func (*MissingReturnValueError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// InvalidImplementationError

type InvalidImplementationError struct {
	ImplementedKind common.DeclarationKind
	ContainerKind   common.DeclarationKind
	Pos             ast.Position
}

var _ SemanticError = &InvalidImplementationError{}
var _ errors.UserError = &InvalidImplementationError{}
var _ errors.SecondaryError = &InvalidImplementationError{}
var _ errors.HasDocumentationLink = &InvalidImplementationError{}

func (*InvalidImplementationError) isSemanticError() {}

func (*InvalidImplementationError) IsUserError() {}

func (e *InvalidImplementationError) Error() string {
	return fmt.Sprintf(
		"cannot implement %s in %s",
		e.ImplementedKind.Name(),
		e.ContainerKind.Name(),
	)
}

func (e *InvalidImplementationError) SecondaryError() string {
	return fmt.Sprintf(
		"only certain declaration types can be implemented within %s, "+
			"%s implementations are not allowed in this context; "+
			"move the implementation to a valid container or remove it",
		e.ContainerKind.Name(),
		e.ImplementedKind.Name(),
	)
}

func (*InvalidImplementationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
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
var _ errors.SecondaryError = &InvalidConformanceError{}
var _ errors.HasDocumentationLink = &InvalidConformanceError{}

func (*InvalidConformanceError) isSemanticError() {}

func (*InvalidConformanceError) IsUserError() {}

func (e *InvalidConformanceError) Error() string {
	return fmt.Sprintf(
		"cannot conform to non-interface type: %#q",
		e.Type.QualifiedString(),
	)
}

func (e *InvalidConformanceError) SecondaryError() string {
	return fmt.Sprintf(
		"only interface types can be conformed to, the type %#q is not an interface; "+
			"ensure you are conforming to an interface type",
		e.Type.QualifiedString(),
	)
}

func (*InvalidConformanceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

// InvalidEnumRawTypeError

type InvalidEnumRawTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidEnumRawTypeError{}
var _ errors.UserError = &InvalidEnumRawTypeError{}
var _ errors.SecondaryError = &InvalidEnumRawTypeError{}
var _ errors.HasDocumentationLink = &InvalidEnumRawTypeError{}

func (*InvalidEnumRawTypeError) isSemanticError() {}

func (*InvalidEnumRawTypeError) IsUserError() {}

func (e *InvalidEnumRawTypeError) Error() string {
	return fmt.Sprintf(
		"invalid enum raw type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*InvalidEnumRawTypeError) SecondaryError() string {
	return "only integer types are currently supported as raw types for enums; " +
		"use an integer type, for example: `UInt8`"
}

func (*InvalidEnumRawTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/enumerations"
}

// MissingEnumRawTypeError

type MissingEnumRawTypeError struct {
	Pos ast.Position
}

var _ SemanticError = &MissingEnumRawTypeError{}
var _ errors.UserError = &MissingEnumRawTypeError{}
var _ errors.SecondaryError = &MissingEnumRawTypeError{}
var _ errors.HasDocumentationLink = &MissingEnumRawTypeError{}

func (*MissingEnumRawTypeError) isSemanticError() {}

func (*MissingEnumRawTypeError) IsUserError() {}

func (*MissingEnumRawTypeError) Error() string {
	return "missing enum raw type"
}

func (e *MissingEnumRawTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingEnumRawTypeError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingEnumRawTypeError) SecondaryError() string {
	return "enums must specify a raw type; " +
		"specify the raw value type, for example: `: UInt8`"
}

func (*MissingEnumRawTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/enumerations"
}

// InvalidEnumConformancesError

type InvalidEnumConformancesError struct {
	ast.Range
}

var _ SemanticError = &InvalidEnumConformancesError{}
var _ errors.UserError = &InvalidEnumConformancesError{}
var _ errors.SecondaryError = &InvalidEnumConformancesError{}
var _ errors.HasDocumentationLink = &InvalidEnumConformancesError{}

func (*InvalidEnumConformancesError) isSemanticError() {}

func (*InvalidEnumConformancesError) IsUserError() {}

func (*InvalidEnumConformancesError) Error() string {
	return "enums cannot conform to interfaces"
}

func (*InvalidEnumConformancesError) SecondaryError() string {
	return "enums are not allowed to conform to interfaces; " +
		"consider removing the conformances"
}

func (*InvalidEnumConformancesError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/enumerations"
}

// InvalidAttachmentConformancesError

type InvalidAttachmentConformancesError struct {
	ast.Range
}

var _ SemanticError = &InvalidAttachmentConformancesError{}
var _ errors.UserError = &InvalidAttachmentConformancesError{}
var _ errors.SecondaryError = &InvalidAttachmentConformancesError{}
var _ errors.HasDocumentationLink = &InvalidAttachmentConformancesError{}

func (*InvalidAttachmentConformancesError) isSemanticError() {}

func (*InvalidAttachmentConformancesError) IsUserError() {}

func (*InvalidAttachmentConformancesError) Error() string {
	return "attachments cannot conform to interfaces"
}

func (*InvalidAttachmentConformancesError) SecondaryError() string {
	return "attachment types cannot conform to interfaces; " +
		"consider removing the conformances"
}

func (*InvalidAttachmentConformancesError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments"
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
var _ errors.HasDocumentationLink = &ConformanceError{}

func (*ConformanceError) isSemanticError() {}

func (*ConformanceError) IsUserError() {}

func (e *ConformanceError) Error() string {
	return fmt.Sprintf(
		"%s %#q does not conform to %s interface %#q",
		e.CompositeType.Kind.Name(),
		e.CompositeType.QualifiedString(),
		e.InterfaceType.CompositeKind.Name(),
		e.InterfaceType.QualifiedString(),
	)
}

func (e *ConformanceError) SecondaryError() string {
	var builder strings.Builder
	if len(e.MissingMembers) > 0 {
		_, _ = fmt.Fprintf(
			&builder,
			"%#q is missing definitions for members: ",
			e.CompositeType.QualifiedString(),
		)
		for i, member := range e.MissingMembers {
			_, _ = fmt.Fprintf(&builder, "%#q", member.Identifier.Identifier)
			if i != len(e.MissingMembers)-1 {
				builder.WriteString(", ")
			}
		}
		if len(e.MissingNestedCompositeTypes) > 0 {
			builder.WriteString(". ")
		}
	}

	if len(e.MissingNestedCompositeTypes) > 0 {
		_, _ = fmt.Fprintf(&builder, "%#q is", e.CompositeType.QualifiedString())
		if len(e.MissingMembers) > 0 {
			builder.WriteString(" also")
		}
		builder.WriteString(" missing definitions for types: ")
		for i, ty := range e.MissingNestedCompositeTypes {
			_, _ = fmt.Fprintf(&builder, "%#q", ty.QualifiedString())
			if i != len(e.MissingNestedCompositeTypes)-1 {
				builder.WriteString(", ")
			}
		}
	}

	return builder.String()
}

func (*ConformanceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
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
		// right now we only support a single initializer
		initializer := e.CompositeDeclaration.DeclarationMembers().Initializers()[0]
		compositeMemberIdentifierRange := ast.NewUnmeteredRangeFromPositioned(initializer.FunctionDeclaration.Identifier)

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

func (MemberMismatchNote) Message() string {
	return "conformance mismatch here"
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
var _ errors.SecondaryError = &DuplicateConformanceError{}
var _ errors.HasDocumentationLink = &DuplicateConformanceError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &DuplicateConformanceError{}

func (*DuplicateConformanceError) isSemanticError() {}

func (*DuplicateConformanceError) IsUserError() {}

func (e *DuplicateConformanceError) Error() string {
	return fmt.Sprintf(
		"%s %#q repeats conformance to %s %#q",
		e.CompositeKindedType.GetCompositeKind().Name(),
		e.CompositeKindedType.QualifiedString(),
		e.InterfaceType.CompositeKind.DeclarationKind(true).Name(),
		e.InterfaceType.QualifiedString(),
	)
}

func (*DuplicateConformanceError) SecondaryError() string {
	return "each interface can only be conformed to once; " +
		"remove the duplicate conformance declaration"
}

func (*DuplicateConformanceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

func (e *DuplicateConformanceError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	startPos := e.StartPos.AttachLeft(code)

	// Include the leading comma
	if startPos.Offset > 0 &&
		startPos.Offset < len(code) &&
		code[startPos.Offset-1] == ',' {

		startPos = startPos.Shifted(nil, -1).AttachLeft(code)
	}

	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove duplicate conformance",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range: ast.Range{
						StartPos: startPos,
						EndPos:   e.EndPos,
					},
				},
			},
		},
	}
}

// CyclicConformanceError
type CyclicConformanceError struct {
	InterfaceType *InterfaceType
	ast.Range
}

var _ SemanticError = CyclicConformanceError{}
var _ errors.UserError = CyclicConformanceError{}
var _ errors.SecondaryError = CyclicConformanceError{}
var _ errors.HasDocumentationLink = CyclicConformanceError{}

func (CyclicConformanceError) isSemanticError() {}

func (CyclicConformanceError) IsUserError() {}

func (e CyclicConformanceError) Error() string {
	return fmt.Sprintf(
		"%#q has a cyclic conformance to itself",
		e.InterfaceType.QualifiedString(),
	)
}

func (CyclicConformanceError) SecondaryError() string {
	return "interfaces cannot have circular dependencies; " +
		"break the cycle by removing one of the conformance declarations"
}

func (CyclicConformanceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

// MultipleInterfaceDefaultImplementationsError
type MultipleInterfaceDefaultImplementationsError struct {
	CompositeKindedType CompositeKindedType
	Member              *Member
	ast.Range
}

var _ SemanticError = &MultipleInterfaceDefaultImplementationsError{}
var _ errors.UserError = &MultipleInterfaceDefaultImplementationsError{}
var _ errors.SecondaryError = &MultipleInterfaceDefaultImplementationsError{}
var _ errors.HasDocumentationLink = &MultipleInterfaceDefaultImplementationsError{}

func (*MultipleInterfaceDefaultImplementationsError) isSemanticError() {}

func (*MultipleInterfaceDefaultImplementationsError) IsUserError() {}

func (e *MultipleInterfaceDefaultImplementationsError) Error() string {
	return fmt.Sprintf(
		"%s %#q has multiple interface default implementations for function %#q",
		e.CompositeKindedType.GetCompositeKind().Name(),
		e.CompositeKindedType.QualifiedString(),
		e.Member.Identifier.Identifier,
	)
}

func (*MultipleInterfaceDefaultImplementationsError) SecondaryError() string {
	return "multiple conformed interfaces provide a default implementation for this function; " +
		"provide your own implementation to resolve the ambiguity"
}

func (*MultipleInterfaceDefaultImplementationsError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

// SpecialFunctionDefaultImplementationError
type SpecialFunctionDefaultImplementationError struct {
	Container  ast.Declaration
	Identifier *ast.Identifier
	KindName   string
}

var _ SemanticError = &SpecialFunctionDefaultImplementationError{}
var _ errors.UserError = &SpecialFunctionDefaultImplementationError{}
var _ errors.SecondaryError = &SpecialFunctionDefaultImplementationError{}
var _ errors.HasDocumentationLink = &SpecialFunctionDefaultImplementationError{}

func (*SpecialFunctionDefaultImplementationError) isSemanticError() {}

func (*SpecialFunctionDefaultImplementationError) IsUserError() {}

func (e *SpecialFunctionDefaultImplementationError) Error() string {
	return fmt.Sprintf(
		"%#q may not be defined as a default function on %s %#q",
		e.Identifier,
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

func (*SpecialFunctionDefaultImplementationError) SecondaryError() string {
	return "special functions like `init` cannot have default implementations in interfaces; " +
		"remove the default implementation"
}

func (*SpecialFunctionDefaultImplementationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
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
var _ errors.SecondaryError = &InterfaceMemberConflictError{}
var _ errors.HasDocumentationLink = &InterfaceMemberConflictError{}

func (*InterfaceMemberConflictError) isSemanticError() {}

func (*InterfaceMemberConflictError) IsUserError() {}

func (e *InterfaceMemberConflictError) Error() string {
	return fmt.Sprintf(
		"%#q %s of %#q conflicts with a %s with the same name in %#q",
		e.MemberName,
		e.MemberKind.Name(),
		e.InterfaceType.QualifiedIdentifier(),
		e.ConflictingMemberKind.Name(),
		e.ConflictingInterfaceType.QualifiedString(),
	)
}

func (*InterfaceMemberConflictError) SecondaryError() string {
	return "interface members must have unique names; " +
		"rename one of the conflicting members to resolve the conflict"
}

func (*InterfaceMemberConflictError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

// MissingConformanceError
type MissingConformanceError struct {
	CompositeType *CompositeType
	InterfaceType *InterfaceType
	ast.Range
}

var _ SemanticError = &MissingConformanceError{}
var _ errors.UserError = &MissingConformanceError{}
var _ errors.SecondaryError = &MissingConformanceError{}
var _ errors.HasDocumentationLink = &MissingConformanceError{}

func (*MissingConformanceError) isSemanticError() {}

func (*MissingConformanceError) IsUserError() {}

func (e *MissingConformanceError) Error() string {
	return fmt.Sprintf(
		"%s %#q is missing a declaration to required conformance to %s %#q",
		e.CompositeType.Kind.Name(),
		e.CompositeType.QualifiedString(),
		e.InterfaceType.CompositeKind.DeclarationKind(true).Name(),
		e.InterfaceType.QualifiedString(),
	)
}

func (e *MissingConformanceError) SecondaryError() string {
	return fmt.Sprintf(
		"the type is required to conform to the interface but does not; "+
			"add a conformance to %#q to the %s declaration",
		e.InterfaceType.QualifiedString(),
		e.CompositeType.Kind.Name(),
	)
}

func (*MissingConformanceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

// UnresolvedImportError

type UnresolvedImportError struct {
	ImportLocation common.Location
	ast.Range
}

var _ SemanticError = &UnresolvedImportError{}
var _ errors.UserError = &UnresolvedImportError{}
var _ errors.SecondaryError = &UnresolvedImportError{}
var _ errors.HasDocumentationLink = &UnresolvedImportError{}

func (*UnresolvedImportError) isSemanticError() {}

func (*UnresolvedImportError) IsUserError() {}

func (e *UnresolvedImportError) Error() string {
	return fmt.Sprintf(
		"import could not be resolved: %#q",
		e.ImportLocation,
	)
}

func (*UnresolvedImportError) SecondaryError() string {
	return "the imported program/contract could not be found; " +
		"ensure the import path is correct and the program/contract exists"
}

func (*UnresolvedImportError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
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
var _ errors.HasDocumentationLink = &NotExportedError{}

func (*NotExportedError) isSemanticError() {}

func (*NotExportedError) IsUserError() {}

func (e *NotExportedError) Error() string {
	return fmt.Sprintf(
		"cannot find declaration %#q in %#q",
		e.Name,
		e.ImportLocation,
	)
}

func (e *NotExportedError) SecondaryError() string {
	var builder strings.Builder
	builder.WriteString("available exported declarations are:\n")

	for _, available := range e.Available {
		builder.WriteString(fmt.Sprintf(" - %#q\n", available))
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

func (*NotExportedError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
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
var _ errors.SecondaryError = &ImportedProgramError{}
var _ errors.HasDocumentationLink = &ImportedProgramError{}

func (*ImportedProgramError) isSemanticError() {}

func (*ImportedProgramError) IsUserError() {}

func (e *ImportedProgramError) Error() string {
	return fmt.Sprintf(
		"checking of imported program/contract %#q failed",
		e.Location,
	)
}

func (e *ImportedProgramError) SecondaryError() string {
	return fmt.Sprintf(
		"ensure the imported program/contract %#q has no errors",
		e.Location,
	)
}

func (*ImportedProgramError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
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
var _ errors.SecondaryError = &AlwaysFailingNonResourceCastingTypeError{}
var _ errors.HasDocumentationLink = &AlwaysFailingNonResourceCastingTypeError{}

func (*AlwaysFailingNonResourceCastingTypeError) isSemanticError() {}

func (*AlwaysFailingNonResourceCastingTypeError) IsUserError() {}

func (e *AlwaysFailingNonResourceCastingTypeError) Error() string {
	return fmt.Sprintf(
		"cast of value of resource-type %#q to non-resource type %#q will always fail",
		e.ValueType.QualifiedString(),
		e.TargetType.QualifiedString(),
	)
}

func (*AlwaysFailingNonResourceCastingTypeError) SecondaryError() string {
	return "resources cannot be cast to non-resource types; use a resource type for the cast"
}

func (*AlwaysFailingNonResourceCastingTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/casting-operators"
}

// AlwaysFailingResourceCastingTypeError

type AlwaysFailingResourceCastingTypeError struct {
	ValueType  Type
	TargetType Type
	ast.Range
}

var _ SemanticError = &AlwaysFailingResourceCastingTypeError{}
var _ errors.UserError = &AlwaysFailingResourceCastingTypeError{}
var _ errors.SecondaryError = &AlwaysFailingResourceCastingTypeError{}
var _ errors.HasDocumentationLink = &AlwaysFailingResourceCastingTypeError{}

func (*AlwaysFailingResourceCastingTypeError) isSemanticError() {}

func (*AlwaysFailingResourceCastingTypeError) IsUserError() {}

func (e *AlwaysFailingResourceCastingTypeError) Error() string {
	return fmt.Sprintf(
		"cast of value of non-resource-type %#q to resource type %#q will always fail",
		e.ValueType.QualifiedString(),
		e.TargetType.QualifiedString(),
	)
}

func (*AlwaysFailingResourceCastingTypeError) SecondaryError() string {
	return "non-resource types cannot be cast to resource types; " +
		"use a non-resource type for the cast"
}

func (*AlwaysFailingResourceCastingTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/casting-operators"
}

// UnsupportedOverloadingError

type UnsupportedOverloadingError struct {
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &UnsupportedOverloadingError{}
var _ errors.UserError = &UnsupportedOverloadingError{}
var _ errors.SecondaryError = &UnsupportedOverloadingError{}
var _ errors.HasDocumentationLink = &UnsupportedOverloadingError{}

func (*UnsupportedOverloadingError) isSemanticError() {}

func (*UnsupportedOverloadingError) IsUserError() {}

func (e *UnsupportedOverloadingError) Error() string {
	return fmt.Sprintf(
		"%s overloading is not allowed",
		e.DeclarationKind.Name(),
	)
}

func (e *UnsupportedOverloadingError) SecondaryError() string {
	return fmt.Sprintf(
		"%s redeclared with the same name as another declaration; "+
			"use unique names for each declaration",
		e.DeclarationKind.Name(),
	)
}

func (e *UnsupportedOverloadingError) DocumentationLink() string {
	switch e.DeclarationKind {
	case common.DeclarationKindFunction:
		return "https://cadence-lang.org/docs/language/functions#function-declarations"
	case common.DeclarationKindInitializer:
		return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
	case common.DeclarationKindResource, common.DeclarationKindStructure, common.DeclarationKindContract:
		return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
	case common.DeclarationKindEvent:
		return "https://cadence-lang.org/docs/language/events"
	case common.DeclarationKindEnum:
		return "https://cadence-lang.org/docs/language/enumerations"
	case common.DeclarationKindAttachment:
		return "https://cadence-lang.org/docs/language/attachments"
	default:
		return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
	}
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
var _ errors.HasDocumentationLink = &CompositeKindMismatchError{}

func (*CompositeKindMismatchError) isSemanticError() {}

func (*CompositeKindMismatchError) IsUserError() {}

func (*CompositeKindMismatchError) Error() string {
	return "mismatched composite kinds"
}

func (e *CompositeKindMismatchError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %#q, got %#q",
		e.ExpectedKind.Name(),
		e.ActualKind.Name(),
	)
}

func (*CompositeKindMismatchError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
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
var _ errors.HasDocumentationLink = &InvalidIntegerLiteralRangeError{}

func (*InvalidIntegerLiteralRangeError) IsUserError() {}

func (*InvalidIntegerLiteralRangeError) isSemanticError() {}

func (*InvalidIntegerLiteralRangeError) Error() string {
	return "integer literal out of range"
}

func (e *InvalidIntegerLiteralRangeError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %#q, in range [%s, %s]",
		e.ExpectedType.QualifiedString(),
		e.ExpectedMinInt,
		e.ExpectedMaxInt,
	)
}

func (*InvalidIntegerLiteralRangeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/booleans-numlits-ints"
}

// InvalidAddressLiteralError

type InvalidAddressLiteralError struct {
	ast.Range
}

var _ SemanticError = &InvalidAddressLiteralError{}
var _ errors.UserError = &InvalidAddressLiteralError{}
var _ errors.SecondaryError = &InvalidAddressLiteralError{}
var _ errors.HasDocumentationLink = &InvalidAddressLiteralError{}

func (*InvalidAddressLiteralError) isSemanticError() {}

func (*InvalidAddressLiteralError) IsUserError() {}

func (*InvalidAddressLiteralError) Error() string {
	return "invalid address literal"
}

func (*InvalidAddressLiteralError) SecondaryError() string {
	return "address literals must be hexadecimal (e.g., 0x1, 0x123) and fit within a 64-bit range; " +
		"correct the literal to be a valid address"
}

func (*InvalidAddressLiteralError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types#address"
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
var _ errors.HasDocumentationLink = &InvalidFixedPointLiteralRangeError{}

func (*InvalidFixedPointLiteralRangeError) isSemanticError() {}

func (*InvalidFixedPointLiteralRangeError) IsUserError() {}

func (*InvalidFixedPointLiteralRangeError) Error() string {
	return "fixed-point literal out of range"
}

func (e *InvalidFixedPointLiteralRangeError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %#q, in range [%s.%s, %s.%s]",
		e.ExpectedType.QualifiedString(),
		e.ExpectedMinInt,
		e.ExpectedMinFractional,
		e.ExpectedMaxInt,
		e.ExpectedMaxFractional,
	)
}

func (*InvalidFixedPointLiteralRangeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/fixed-point-nums-ints"
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
var _ errors.HasDocumentationLink = &InvalidFixedPointLiteralScaleError{}

func (*InvalidFixedPointLiteralScaleError) isSemanticError() {}

func (*InvalidFixedPointLiteralScaleError) IsUserError() {}

func (*InvalidFixedPointLiteralScaleError) Error() string {
	return "fixed-point literal scale out of range"
}

func (e *InvalidFixedPointLiteralScaleError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %#q, with maximum scale %d",
		e.ExpectedType.QualifiedString(),
		e.ExpectedScale,
	)
}

func (*InvalidFixedPointLiteralScaleError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/fixed-point-nums-ints"
}

// MissingReturnStatementError

type MissingReturnStatementError struct {
	ast.Range
}

var _ SemanticError = &MissingReturnStatementError{}
var _ errors.UserError = &MissingReturnStatementError{}
var _ errors.SecondaryError = &MissingReturnStatementError{}
var _ errors.HasDocumentationLink = &MissingReturnStatementError{}

func (*MissingReturnStatementError) isSemanticError() {}

func (*MissingReturnStatementError) IsUserError() {}

func (*MissingReturnStatementError) Error() string {
	return "missing return statement"
}

func (*MissingReturnStatementError) SecondaryError() string {
	return "not all code paths return a value; " +
		"add a return statement to return a value"
}

func (*MissingReturnStatementError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// UnsupportedOptionalChainingAssignmentError

type UnsupportedOptionalChainingAssignmentError struct {
	ast.Range
}

var _ SemanticError = &UnsupportedOptionalChainingAssignmentError{}
var _ errors.UserError = &UnsupportedOptionalChainingAssignmentError{}
var _ errors.SecondaryError = &UnsupportedOptionalChainingAssignmentError{}
var _ errors.HasDocumentationLink = &UnsupportedOptionalChainingAssignmentError{}

func (*UnsupportedOptionalChainingAssignmentError) isSemanticError() {}

func (*UnsupportedOptionalChainingAssignmentError) IsUserError() {}

func (*UnsupportedOptionalChainingAssignmentError) Error() string {
	return "cannot assign to optional chaining expression"
}

func (*UnsupportedOptionalChainingAssignmentError) SecondaryError() string {
	return "assignment to an optional chaining expression is not supported; " +
		"use direct assignment instead of optional chaining"
}

func (*UnsupportedOptionalChainingAssignmentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/anystruct-anyresource-opts-never#optionals"
}

// MissingResourceAnnotationError

type MissingResourceAnnotationError struct {
	ast.Range
}

var _ SemanticError = &MissingResourceAnnotationError{}
var _ errors.UserError = &MissingResourceAnnotationError{}
var _ errors.SecondaryError = &MissingResourceAnnotationError{}
var _ errors.HasDocumentationLink = &MissingResourceAnnotationError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingResourceAnnotationError{}

func (*MissingResourceAnnotationError) isSemanticError() {}

func (*MissingResourceAnnotationError) IsUserError() {}

func (*MissingResourceAnnotationError) Error() string {
	return fmt.Sprintf(
		"missing resource annotation: %#q",
		common.CompositeKindResource.Annotation(),
	)
}

func (*MissingResourceAnnotationError) SecondaryError() string {
	return "resource types must be annotated as such; " +
		"add the resource annotation (`@`) before the resource type"
}

func (*MissingResourceAnnotationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

func (e *MissingResourceAnnotationError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `@`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "@",
					Range: ast.NewUnmeteredRange(
						e.StartPos,
						e.StartPos,
					),
				},
			},
		},
	}
}

// InvalidNestedResourceMoveError

type InvalidNestedResourceMoveError struct {
	ast.Range
}

var _ SemanticError = &InvalidNestedResourceMoveError{}
var _ errors.UserError = &InvalidNestedResourceMoveError{}
var _ errors.SecondaryError = &InvalidNestedResourceMoveError{}
var _ errors.HasDocumentationLink = &InvalidNestedResourceMoveError{}

func (*InvalidNestedResourceMoveError) isSemanticError() {}

func (*InvalidNestedResourceMoveError) IsUserError() {}

func (*InvalidNestedResourceMoveError) Error() string {
	return "cannot move nested resource"
}

func (*InvalidNestedResourceMoveError) SecondaryError() string {
	return "nested resources cannot be moved independently; " +
		"move the containing resource instead"
}

func (*InvalidNestedResourceMoveError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources#nested-resources"
}

// InvalidInterfaceConditionResourceInvalidationError

type InvalidInterfaceConditionResourceInvalidationError struct {
	ast.Range
}

var _ SemanticError = &InvalidInterfaceConditionResourceInvalidationError{}
var _ errors.UserError = &InvalidInterfaceConditionResourceInvalidationError{}
var _ errors.SecondaryError = &InvalidInterfaceConditionResourceInvalidationError{}
var _ errors.HasDocumentationLink = &InvalidInterfaceConditionResourceInvalidationError{}

func (*InvalidInterfaceConditionResourceInvalidationError) isSemanticError() {}

func (*InvalidInterfaceConditionResourceInvalidationError) IsUserError() {}

func (*InvalidInterfaceConditionResourceInvalidationError) Error() string {
	return "cannot invalidate resource in interface condition"
}

func (*InvalidInterfaceConditionResourceInvalidationError) SecondaryError() string {
	return "interface conditions must be pure and cannot modify resources; " +
		"use pre/post conditions instead"
}

func (*InvalidInterfaceConditionResourceInvalidationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

// InvalidResourceAnnotationError

type InvalidResourceAnnotationError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidResourceAnnotationError{}
var _ errors.UserError = &InvalidResourceAnnotationError{}
var _ errors.SecondaryError = &InvalidResourceAnnotationError{}
var _ errors.HasDocumentationLink = &InvalidResourceAnnotationError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidResourceAnnotationError{}

func (*InvalidResourceAnnotationError) isSemanticError() {}

func (*InvalidResourceAnnotationError) IsUserError() {}

func (e *InvalidResourceAnnotationError) Error() string {
	return fmt.Sprintf(
		"invalid resource annotation: %#q on type %#q",
		common.CompositeKindResource.Annotation(),
		e.Type.QualifiedString(),
	)
}

func (e *InvalidResourceAnnotationError) SecondaryError() string {
	return fmt.Sprintf(
		"type %#q is not a resource; "+
			"remove the resource annotation (`@`)",
		e.Type.QualifiedString(),
	)
}

func (*InvalidResourceAnnotationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

func (e *InvalidResourceAnnotationError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove `@`",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range: ast.NewUnmeteredRange(
						e.StartPos,
						e.StartPos,
					),
				},
			},
		},
	}
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
var _ errors.HasDocumentationLink = &InvalidInterfaceTypeError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidInterfaceTypeError{}

func (*InvalidInterfaceTypeError) isSemanticError() {}

func (*InvalidInterfaceTypeError) IsUserError() {}

func (*InvalidInterfaceTypeError) Error() string {
	return "invalid use of interface as type"
}

func (e *InvalidInterfaceTypeError) SecondaryError() string {
	return fmt.Sprintf(
		"interfaces can not be used as types directly; "+
			"wrap interfaces in intersection types (e.g., `{Interface}`); "+
			"got %#q, consider using %#q",
		e.ActualType.QualifiedString(),
		e.ExpectedType.QualifiedString(),
	)
}

func (*InvalidInterfaceTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
}

func (e *InvalidInterfaceTypeError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	replacement := e.ExpectedType.QualifiedString()
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: fmt.Sprintf("Replace with `%s`", replacement),
			TextEdits: []ast.TextEdit{
				{
					Replacement: replacement,
					Range:       e.Range,
				},
			},
		},
	}
}

// InvalidInterfaceDeclarationError

type InvalidInterfaceDeclarationError struct {
	CompositeKind common.CompositeKind
	ast.Range
}

var _ SemanticError = &InvalidInterfaceDeclarationError{}
var _ errors.UserError = &InvalidInterfaceDeclarationError{}
var _ errors.SecondaryError = &InvalidInterfaceDeclarationError{}
var _ errors.HasDocumentationLink = &InvalidInterfaceDeclarationError{}

func (*InvalidInterfaceDeclarationError) isSemanticError() {}

func (*InvalidInterfaceDeclarationError) IsUserError() {}

func (e *InvalidInterfaceDeclarationError) Error() string {
	return fmt.Sprintf(
		"%s interfaces are not supported",
		e.CompositeKind.Name(),
	)
}

func (e *InvalidInterfaceDeclarationError) SecondaryError() string {
	return "only contract, struct, and resource types can have interfaces; " +
		"consider changing the invalid kind to a supported kind"
}

func (*InvalidInterfaceDeclarationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/interfaces"
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
var _ errors.HasDocumentationLink = &IncorrectTransferOperationError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &IncorrectTransferOperationError{}

func (*IncorrectTransferOperationError) isSemanticError() {}

func (*IncorrectTransferOperationError) IsUserError() {}

func (*IncorrectTransferOperationError) Error() string {
	return "incorrect transfer operation"
}

func (e *IncorrectTransferOperationError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %#q, got %#q; replace the operator with the expected one",
		e.ExpectedOperation.Operator(),
		e.ActualOperation.Operator(),
	)
}

func (*IncorrectTransferOperationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

func (e *IncorrectTransferOperationError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	op := e.ExpectedOperation.Operator()
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: fmt.Sprintf("Replace with %#q", op),
			TextEdits: []ast.TextEdit{
				{
					Replacement: op,
					Range:       e.Range,
				},
			},
		},
	}
}

// InvalidConstructionError

type InvalidConstructionError struct {
	Pos ast.Position
}

var _ SemanticError = &InvalidConstructionError{}
var _ errors.UserError = &InvalidConstructionError{}
var _ errors.SecondaryError = &InvalidConstructionError{}
var _ errors.HasDocumentationLink = &InvalidConstructionError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidConstructionError{}

func (*InvalidConstructionError) isSemanticError() {}

func (*InvalidConstructionError) IsUserError() {}

func (e *InvalidConstructionError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidConstructionError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	const length = len(`create`)
	return e.Pos.Shifted(memoryGauge, length-1)
}

func (*InvalidConstructionError) Error() string {
	return "cannot create value: not a resource"
}

func (*InvalidConstructionError) SecondaryError() string {
	return "`create` expressions can only be used with resource types; " +
		"use regular constructor calls for structs and other composite types"
}

func (*InvalidConstructionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
}

func (e *InvalidConstructionError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove `create`",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range: ast.Range{
						StartPos: e.Pos,
						EndPos:   e.EndPosition(nil).AttachRight(code),
					},
				},
			},
		},
	}
}

// InvalidDestructionError

type InvalidDestructionError struct {
	ast.Range
}

var _ SemanticError = &InvalidDestructionError{}
var _ errors.UserError = &InvalidDestructionError{}
var _ errors.SecondaryError = &InvalidDestructionError{}
var _ errors.HasDocumentationLink = &InvalidDestructionError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidDestructionError{}

func (*InvalidDestructionError) isSemanticError() {}

func (*InvalidDestructionError) IsUserError() {}

func (*InvalidDestructionError) Error() string {
	return "cannot destroy value: not a resource"
}

func (*InvalidDestructionError) SecondaryError() string {
	return "`destroy` expressions can only be used with resource types; " +
		"non-resource types are automatically cleaned up when they go out of scope"
}

func (*InvalidDestructionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
}

func (e *InvalidDestructionError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove `destroy` expression",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       e.Range,
				},
			},
		},
	}
}

// ResourceLossError

type ResourceLossError struct {
	ast.Range
}

var _ SemanticError = &ResourceLossError{}
var _ errors.UserError = &ResourceLossError{}
var _ errors.SecondaryError = &ResourceLossError{}
var _ errors.HasDocumentationLink = &ResourceLossError{}

func (*ResourceLossError) isSemanticError() {}

func (*ResourceLossError) IsUserError() {}

func (*ResourceLossError) Error() string {
	return "loss of resource"
}

func (*ResourceLossError) SecondaryError() string {
	return "resources must be explicitly moved or destroyed to prevent resource loss; " +
		"move or destroy the resource before it goes out of scope"
}

func (*ResourceLossError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

// ResourceUseAfterInvalidationError

type ResourceUseAfterInvalidationError struct {
	Invalidation ResourceInvalidation
	ast.Range
}

var _ SemanticError = &ResourceUseAfterInvalidationError{}
var _ errors.UserError = &ResourceUseAfterInvalidationError{}
var _ errors.SecondaryError = &ResourceUseAfterInvalidationError{}
var _ errors.HasDocumentationLink = &ResourceUseAfterInvalidationError{}

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

func (*ResourceUseAfterInvalidationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
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
var _ errors.HasDocumentationLink = &MissingCreateError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingCreateError{}

func (*MissingCreateError) isSemanticError() {}

func (*MissingCreateError) IsUserError() {}

func (*MissingCreateError) Error() string {
	return "cannot create resource without the `create` keyword"
}

func (*MissingCreateError) SecondaryError() string {
	return "resource creation requires the `create` keyword; " +
		"add the `create` keyword before the resource constructor call"
}

func (*MissingCreateError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

func (e *MissingCreateError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `create`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "create ",
					Range:     ast.NewUnmeteredRange(e.StartPos, e.StartPos),
				},
			},
		},
	}
}

// MissingMoveOperationError

type MissingMoveOperationError struct {
	Pos ast.Position
}

var _ SemanticError = &MissingMoveOperationError{}
var _ errors.UserError = &MissingMoveOperationError{}
var _ errors.SecondaryError = &MissingMoveOperationError{}
var _ errors.HasDocumentationLink = &MissingMoveOperationError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingMoveOperationError{}

func (*MissingMoveOperationError) isSemanticError() {}

func (*MissingMoveOperationError) IsUserError() {}

func (*MissingMoveOperationError) Error() string {
	return "missing move operation: `<-`"
}

func (e *MissingMoveOperationError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingMoveOperationError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingMoveOperationError) SecondaryError() string {
	return "moving a resource requires the move operator (`<-`); " +
		"add the `<-` operator to move the resource"
}

func (*MissingMoveOperationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/assign-move-force-swap"
}

func (e *MissingMoveOperationError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `<-`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "<-",
					Range: ast.NewUnmeteredRange(
						e.Pos,
						e.Pos,
					),
				},
			},
		},
	}
}

// InvalidMoveOperationError

type InvalidMoveOperationError struct {
	Pos ast.Position
}

var _ SemanticError = &InvalidMoveOperationError{}
var _ errors.UserError = &InvalidMoveOperationError{}
var _ errors.SecondaryError = &InvalidMoveOperationError{}
var _ errors.HasDocumentationLink = &InvalidMoveOperationError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidMoveOperationError{}

func (*InvalidMoveOperationError) isSemanticError() {}

func (*InvalidMoveOperationError) IsUserError() {}

func (e *InvalidMoveOperationError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidMoveOperationError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	// length of `<-`
	const length = 2
	return e.Pos.Shifted(memoryGauge, length-1)
}

func (*InvalidMoveOperationError) Error() string {
	return "invalid move operation (`<-`) for non-resource"
}

func (*InvalidMoveOperationError) SecondaryError() string {
	return "the move operator (`<-`) can only be used on resources; " +
		"remove the `<-` operator for the non-resource"
}

func (*InvalidMoveOperationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/assign-move-force-swap"
}

func (e *InvalidMoveOperationError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove `<-`",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       ast.NewRangeFromPositioned(nil, e),
				},
			},
		},
	}
}

// ResourceCapturingError

type ResourceCapturingError struct {
	Name string
	Pos  ast.Position
}

var _ SemanticError = &ResourceCapturingError{}
var _ errors.UserError = &ResourceCapturingError{}
var _ errors.SecondaryError = &ResourceCapturingError{}
var _ errors.HasDocumentationLink = &ResourceCapturingError{}

func (*ResourceCapturingError) isSemanticError() {}

func (*ResourceCapturingError) IsUserError() {}

func (e *ResourceCapturingError) Error() string {
	return fmt.Sprintf("cannot capture resource in closure: %#q", e.Name)
}

func (e *ResourceCapturingError) StartPosition() ast.Position {
	return e.Pos
}

func (e *ResourceCapturingError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}

func (*ResourceCapturingError) SecondaryError() string {
	return "resources cannot be captured in closures due to resource safety constraints; " +
		"move the resource into the closure if needed, or access it differently"
}

func (*ResourceCapturingError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

// InvalidResourceFieldError

type InvalidResourceFieldError struct {
	Name          string
	CompositeKind common.CompositeKind
	Pos           ast.Position
}

var _ SemanticError = &InvalidResourceFieldError{}
var _ errors.UserError = &InvalidResourceFieldError{}
var _ errors.SecondaryError = &InvalidResourceFieldError{}
var _ errors.HasDocumentationLink = &InvalidResourceFieldError{}

func (*InvalidResourceFieldError) isSemanticError() {}

func (*InvalidResourceFieldError) IsUserError() {}

func (e *InvalidResourceFieldError) Error() string {
	return fmt.Sprintf(
		"invalid resource field in %s: %#q",
		e.CompositeKind.Name(),
		e.Name,
	)
}

func (*InvalidResourceFieldError) SecondaryError() string {
	return "resource fields must be storable and have proper access modifiers; " +
		"ensure the field type is storable and the access modifier is valid"
}

func (*InvalidResourceFieldError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
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
var _ errors.HasDocumentationLink = &InvalidSwapExpressionError{}

func (*InvalidSwapExpressionError) isSemanticError() {}

func (*InvalidSwapExpressionError) IsUserError() {}

func (e *InvalidSwapExpressionError) Error() string {
	return fmt.Sprintf(
		"invalid %s-hand side of swap",
		e.Side.Name(),
	)
}

func (*InvalidSwapExpressionError) SecondaryError() string {
	return "swap targets must be variables, array elements, or fields; " +
		"ensure both sides of the swap are valid targets"
}

func (*InvalidSwapExpressionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/assign-move-force-swap#swapping-operator--"
}

// InvalidEventParameterTypeError

type InvalidEventParameterTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidEventParameterTypeError{}
var _ errors.UserError = &InvalidEventParameterTypeError{}
var _ errors.SecondaryError = &InvalidEventParameterTypeError{}
var _ errors.HasDocumentationLink = &InvalidEventParameterTypeError{}

func (*InvalidEventParameterTypeError) isSemanticError() {}

func (*InvalidEventParameterTypeError) IsUserError() {}

func (e *InvalidEventParameterTypeError) Error() string {
	return fmt.Sprintf(
		"unsupported event parameter type: %#q",
		e.Type.QualifiedString(),
	)
}

func (e *InvalidEventParameterTypeError) SecondaryError() string {
	return fmt.Sprintf(
		"event parameters must be storable types; the type %#q cannot be stored",
		e.Type.QualifiedString(),
	)
}

func (*InvalidEventParameterTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/events"
}

// InvalidEventUsageError

type InvalidEventUsageError struct {
	EventName string
	ast.Range
}

var _ SemanticError = &InvalidEventUsageError{}
var _ errors.UserError = &InvalidEventUsageError{}
var _ errors.SecondaryError = &InvalidEventUsageError{}
var _ errors.HasDocumentationLink = &InvalidEventUsageError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidEventUsageError{}

func (*InvalidEventUsageError) isSemanticError() {}

func (*InvalidEventUsageError) IsUserError() {}

func (e *InvalidEventUsageError) Error() string {
	if e.EventName != "" {
		return fmt.Sprintf("event %#q can only be invoked in an `emit` statement", e.EventName)
	}
	return "events can only be invoked in an `emit` statement"
}

func (e *InvalidEventUsageError) SecondaryError() string {
	if e.EventName != "" {
		return fmt.Sprintf(
			"use `emit %s()` syntax instead of calling the event directly",
			e.EventName,
		)
	}
	return "use `emit EventName()` syntax instead of calling the event directly"
}

func (*InvalidEventUsageError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/events"
}

func (e *InvalidEventUsageError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert `emit`",
			TextEdits: []ast.TextEdit{
				{
					Insertion: "emit ",
					Range: ast.NewUnmeteredRange(
						e.StartPos,
						e.StartPos,
					),
				},
			},
		},
	}
}

// EmitNonEventError

type EmitNonEventError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &EmitNonEventError{}
var _ errors.UserError = &EmitNonEventError{}
var _ errors.SecondaryError = &EmitNonEventError{}
var _ errors.HasDocumentationLink = &EmitNonEventError{}

func (*EmitNonEventError) isSemanticError() {}

func (*EmitNonEventError) IsUserError() {}

func (e *EmitNonEventError) Error() string {
	return fmt.Sprintf(
		"cannot emit non-event type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*EmitNonEventError) SecondaryError() string {
	return "only event types can be emitted"
}

func (*EmitNonEventError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/events"
}

// EmitDefaultDestroyEventError

type EmitDefaultDestroyEventError struct {
	ast.Range
}

var _ SemanticError = &EmitDefaultDestroyEventError{}
var _ errors.UserError = &EmitDefaultDestroyEventError{}
var _ errors.SecondaryError = &EmitDefaultDestroyEventError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &EmitDefaultDestroyEventError{}
var _ errors.HasDocumentationLink = &EmitDefaultDestroyEventError{}

func (*EmitDefaultDestroyEventError) isSemanticError() {}

func (*EmitDefaultDestroyEventError) IsUserError() {}

func (*EmitDefaultDestroyEventError) Error() string {
	return "default destruction events may not be explicitly emitted"
}

func (*EmitDefaultDestroyEventError) SecondaryError() string {
	return "ResourceDestroyed events are automatically emitted when resources are destroyed; " +
		"remove the explicit emit statement"
}

func (e *EmitDefaultDestroyEventError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove explicit emit statement",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       e.Range,
				},
			},
		},
	}
}

func (*EmitDefaultDestroyEventError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources#destroy-events"
}

// EmitImportedEventError

type EmitImportedEventError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &EmitImportedEventError{}
var _ errors.UserError = &EmitImportedEventError{}
var _ errors.SecondaryError = &EmitImportedEventError{}
var _ errors.HasDocumentationLink = &EmitImportedEventError{}

func (*EmitImportedEventError) isSemanticError() {}

func (*EmitImportedEventError) IsUserError() {}

func (e *EmitImportedEventError) Error() string {
	return fmt.Sprintf(
		"cannot emit imported event type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*EmitImportedEventError) SecondaryError() string {
	return "events can only be emitted from the contract where they are declared; " +
		"imported events cannot be emitted elsewhere, e.g. from other contracts or transactions"
}

func (*EmitImportedEventError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/events"
}

// InvalidResourceAssignmentError

type InvalidResourceAssignmentError struct {
	ast.Range
}

var _ SemanticError = &InvalidResourceAssignmentError{}
var _ errors.UserError = &InvalidResourceAssignmentError{}
var _ errors.SecondaryError = &InvalidResourceAssignmentError{}
var _ errors.HasDocumentationLink = &InvalidResourceAssignmentError{}

func (*InvalidResourceAssignmentError) isSemanticError() {}

func (*InvalidResourceAssignmentError) IsUserError() {}

func (*InvalidResourceAssignmentError) Error() string {
	return "cannot assign to resource-typed target"
}

func (*InvalidResourceAssignmentError) SecondaryError() string {
	return "direct assignment to a resource-typed location is not allowed; " +
		"consider force assigning (`<-!`) or swapping (`<->`)"
}

func (e *InvalidResourceAssignmentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/assign-move-force-swap"
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
var _ errors.HasDocumentationLink = &ResourceFieldNotInvalidatedError{}

func (*ResourceFieldNotInvalidatedError) isSemanticError() {}

func (*ResourceFieldNotInvalidatedError) IsUserError() {}

func (e *ResourceFieldNotInvalidatedError) Error() string {
	return fmt.Sprintf(
		"field %#q of type %#q is not invalidated (moved or destroyed)",
		e.FieldName,
		e.Type.QualifiedString(),
	)
}

func (*ResourceFieldNotInvalidatedError) SecondaryError() string {
	return "all resource fields must be moved or destroyed before the resource can be destroyed"
}

func (*ResourceFieldNotInvalidatedError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
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
var _ errors.SecondaryError = &UninitializedFieldAccessError{}
var _ errors.HasDocumentationLink = &UninitializedFieldAccessError{}

func (*UninitializedFieldAccessError) isSemanticError() {}

func (*UninitializedFieldAccessError) IsUserError() {}

func (e *UninitializedFieldAccessError) StartPosition() ast.Position {
	return e.Pos
}

func (e *UninitializedFieldAccessError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	length := len(e.Name)
	return e.Pos.Shifted(memoryGauge, length-1)
}
func (e *UninitializedFieldAccessError) Error() string {
	return fmt.Sprintf(
		"cannot access uninitialized field: %#q",
		e.Name,
	)
}

func (*UninitializedFieldAccessError) SecondaryError() string {
	return "fields must be initialized before they are accessed; " +
		"initialize the field before accessing it"
}

func (*UninitializedFieldAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
}

// UnreachableStatementError

type UnreachableStatementError struct {
	ast.Range
}

var _ SemanticError = &UnreachableStatementError{}
var _ errors.UserError = &UnreachableStatementError{}
var _ errors.SecondaryError = &UnreachableStatementError{}
var _ errors.HasDocumentationLink = &UnreachableStatementError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &UnreachableStatementError{}

func (*UnreachableStatementError) isSemanticError() {}

func (*UnreachableStatementError) IsUserError() {}

func (*UnreachableStatementError) Error() string {
	return "unreachable statement"
}

func (*UnreachableStatementError) SecondaryError() string {
	return "this statement can never be executed; " +
		"consider removing this code or revising control flow"
}

func (*UnreachableStatementError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/control-flow"
}

func (e *UnreachableStatementError) SuggestFixes(code string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove unreachable statement",
			TextEdits: []ast.TextEdit{
				{
					Replacement: "",
					Range:       e.Range,
				},
			},
		},
	}
}

// UninitializedUseError

type UninitializedUseError struct {
	Name string
	Pos  ast.Position
}

var _ SemanticError = &UninitializedUseError{}
var _ errors.UserError = &UninitializedUseError{}
var _ errors.SecondaryError = &UninitializedUseError{}
var _ errors.HasDocumentationLink = &UninitializedUseError{}

func (*UninitializedUseError) isSemanticError() {}

func (*UninitializedUseError) IsUserError() {}

func (e *UninitializedUseError) Error() string {
	return fmt.Sprintf(
		"cannot use incompletely initialized value: %#q",
		e.Name,
	)
}

func (*UninitializedUseError) SecondaryError() string {
	return "a value must be fully initialized before it can be used; " +
		"complete the initialization before using the value"
}

func (*UninitializedUseError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
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
var _ errors.SecondaryError = &InvalidResourceArrayMemberError{}
var _ errors.HasDocumentationLink = &InvalidResourceArrayMemberError{}

func (*InvalidResourceArrayMemberError) isSemanticError() {}

func (*InvalidResourceArrayMemberError) IsUserError() {}

func (e *InvalidResourceArrayMemberError) Error() string {
	return fmt.Sprintf(
		"%s %#q is not available for resource arrays",
		e.DeclarationKind.Name(),
		e.Name,
	)
}

func (*InvalidResourceArrayMemberError) SecondaryError() string {
	return "resource arrays have limited member access due to resource safety constraints"
}

func (*InvalidResourceArrayMemberError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources#resources-in-arrays-and-dictionaries"
}

// InvalidResourceDictionaryMemberError

type InvalidResourceDictionaryMemberError struct {
	Name            string
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidResourceDictionaryMemberError{}
var _ errors.UserError = &InvalidResourceDictionaryMemberError{}
var _ errors.SecondaryError = &InvalidResourceDictionaryMemberError{}
var _ errors.HasDocumentationLink = &InvalidResourceDictionaryMemberError{}

func (*InvalidResourceDictionaryMemberError) isSemanticError() {}

func (*InvalidResourceDictionaryMemberError) IsUserError() {}

func (e *InvalidResourceDictionaryMemberError) Error() string {
	return fmt.Sprintf(
		"%s %#q is not available for resource dictionaries",
		e.DeclarationKind.Name(),
		e.Name,
	)
}

func (*InvalidResourceDictionaryMemberError) SecondaryError() string {
	return "resource dictionaries have limited member access due to resource safety constraints"
}

func (*InvalidResourceDictionaryMemberError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources#resources-in-arrays-and-dictionaries"
}

// InvalidResourceOptionalMemberError

type InvalidResourceOptionalMemberError struct {
	Name            string
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidResourceOptionalMemberError{}
var _ errors.UserError = &InvalidResourceOptionalMemberError{}
var _ errors.SecondaryError = &InvalidResourceOptionalMemberError{}
var _ errors.HasDocumentationLink = &InvalidResourceOptionalMemberError{}

func (*InvalidResourceOptionalMemberError) isSemanticError() {}

func (*InvalidResourceOptionalMemberError) IsUserError() {}

func (e *InvalidResourceOptionalMemberError) Error() string {
	return fmt.Sprintf(
		"%s %#q is not available for resource optionals",
		e.DeclarationKind.Name(),
		e.Name,
	)
}

func (*InvalidResourceOptionalMemberError) SecondaryError() string {
	return "resource optionals have limited member access due to resource safety constraints"
}

func (*InvalidResourceOptionalMemberError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/anystruct-anyresource-opts-never#optionals"
}

// NonReferenceTypeReferenceError

type NonReferenceTypeReferenceError struct {
	ActualType Type
	ast.Range
}

var _ SemanticError = &NonReferenceTypeReferenceError{}
var _ errors.UserError = &NonReferenceTypeReferenceError{}
var _ errors.SecondaryError = &NonReferenceTypeReferenceError{}
var _ errors.HasDocumentationLink = &NonReferenceTypeReferenceError{}

func (*NonReferenceTypeReferenceError) isSemanticError() {}

func (*NonReferenceTypeReferenceError) IsUserError() {}

func (*NonReferenceTypeReferenceError) Error() string {
	return "cannot create reference"
}

func (e *NonReferenceTypeReferenceError) SecondaryError() string {
	return fmt.Sprintf(
		"expected reference type, got %#q",
		e.ActualType.QualifiedString(),
	)
}

func (*NonReferenceTypeReferenceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/references"
}

// ReferenceToAnOptionalError

type ReferenceToAnOptionalError struct {
	ReferencedOptionalType *OptionalType
	ast.Range
}

var _ SemanticError = &ReferenceToAnOptionalError{}
var _ errors.UserError = &ReferenceToAnOptionalError{}
var _ errors.SecondaryError = &ReferenceToAnOptionalError{}
var _ errors.HasDocumentationLink = &ReferenceToAnOptionalError{}

func (*ReferenceToAnOptionalError) isSemanticError() {}

func (*ReferenceToAnOptionalError) IsUserError() {}

func (*ReferenceToAnOptionalError) Error() string {
	return "cannot create reference"
}

func (e *ReferenceToAnOptionalError) SecondaryError() string {
	return fmt.Sprintf(
		"expected non-optional type, got %#q; consider taking a reference with type %#q",
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

func (*ReferenceToAnOptionalError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/references"
}

// InvalidResourceCreationError

type InvalidResourceCreationError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidResourceCreationError{}
var _ errors.UserError = &InvalidResourceCreationError{}
var _ errors.SecondaryError = &InvalidResourceCreationError{}
var _ errors.HasDocumentationLink = &InvalidResourceCreationError{}

func (*InvalidResourceCreationError) isSemanticError() {}

func (*InvalidResourceCreationError) IsUserError() {}

func (e *InvalidResourceCreationError) Error() string {
	return fmt.Sprintf(
		"cannot create resource type outside of containing contract: %#q",
		e.Type.QualifiedString(),
	)
}

func (*InvalidResourceCreationError) SecondaryError() string {
	return "resources can only be created within the contract that declares them"
}

func (*InvalidResourceCreationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

// NonResourceTypeError

type NonResourceTypeError struct {
	ActualType Type
	ast.Range
}

var _ SemanticError = &NonResourceTypeError{}
var _ errors.UserError = &NonResourceTypeError{}
var _ errors.SecondaryError = &NonResourceTypeError{}
var _ errors.HasDocumentationLink = &NonResourceTypeError{}

func (*NonResourceTypeError) isSemanticError() {}

func (*NonResourceTypeError) IsUserError() {}

func (*NonResourceTypeError) Error() string {
	return "invalid type"
}

func (e *NonResourceTypeError) SecondaryError() string {
	return fmt.Sprintf(
		"expected resource type, got %#q",
		e.ActualType.QualifiedString(),
	)
}

func (*NonResourceTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

// InvalidAssignmentTargetError

type InvalidAssignmentTargetError struct {
	ast.Range
}

var _ SemanticError = &InvalidAssignmentTargetError{}
var _ errors.UserError = &InvalidAssignmentTargetError{}
var _ errors.SecondaryError = &InvalidAssignmentTargetError{}
var _ errors.HasDocumentationLink = &InvalidAssignmentTargetError{}

func (*InvalidAssignmentTargetError) isSemanticError() {}

func (*InvalidAssignmentTargetError) IsUserError() {}

func (*InvalidAssignmentTargetError) Error() string {
	return "cannot assign to unassignable expression"
}

func (*InvalidAssignmentTargetError) SecondaryError() string {
	return "only variables, array elements, and struct fields can be assigned to; " +
		"function calls and literals cannot be assigned to"
}

func (*InvalidAssignmentTargetError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/operators/assign-move-force-swap"
}

// ResourceMethodBindingError

type ResourceMethodBindingError struct {
	ast.Range
}

var _ SemanticError = &ResourceMethodBindingError{}
var _ errors.UserError = &ResourceMethodBindingError{}
var _ errors.SecondaryError = &ResourceMethodBindingError{}
var _ errors.HasDocumentationLink = &ResourceMethodBindingError{}

func (*ResourceMethodBindingError) isSemanticError() {}

func (*ResourceMethodBindingError) IsUserError() {}

func (*ResourceMethodBindingError) Error() string {
	return "cannot bind resource method to variable or pass as argument"
}

func (*ResourceMethodBindingError) SecondaryError() string {
	return "call the method directly (e.g., `r.method()`) instead of binding it (e.g., `r.method`)"
}

func (*ResourceMethodBindingError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

// InvalidDictionaryKeyTypeError

type InvalidDictionaryKeyTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidDictionaryKeyTypeError{}
var _ errors.UserError = &InvalidDictionaryKeyTypeError{}
var _ errors.SecondaryError = &InvalidDictionaryKeyTypeError{}
var _ errors.HasDocumentationLink = &InvalidDictionaryKeyTypeError{}

func (*InvalidDictionaryKeyTypeError) isSemanticError() {}

func (*InvalidDictionaryKeyTypeError) IsUserError() {}

func (e *InvalidDictionaryKeyTypeError) Error() string {
	return fmt.Sprintf(
		"cannot use type as dictionary key type: %#q",
		e.Type.QualifiedString(),
	)
}

func (e *InvalidDictionaryKeyTypeError) SecondaryError() string {
	return fmt.Sprintf(
		"type %#q cannot be used as a key because it is not hashable; "+
			"use primitive types like `String`, `Int`, `Address`, or `Bool` instead",
		e.Type.QualifiedString(),
	)
}

func (*InvalidDictionaryKeyTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/dictionaries"
}

// MissingFunctionBodyError

type MissingFunctionBodyError struct {
	Pos ast.Position
}

var _ SemanticError = &MissingFunctionBodyError{}
var _ errors.UserError = &MissingFunctionBodyError{}
var _ errors.SecondaryError = &MissingFunctionBodyError{}
var _ errors.HasDocumentationLink = &MissingFunctionBodyError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &MissingFunctionBodyError{}

func (*MissingFunctionBodyError) isSemanticError() {}

func (*MissingFunctionBodyError) IsUserError() {}

func (*MissingFunctionBodyError) Error() string {
	return "missing function implementation"
}

func (e *MissingFunctionBodyError) StartPosition() ast.Position {
	return e.Pos
}

func (e *MissingFunctionBodyError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*MissingFunctionBodyError) SecondaryError() string {
	return "add a function body with `{ ... }`"
}

func (*MissingFunctionBodyError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

func (e *MissingFunctionBodyError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	pos := e.Pos.Shifted(nil, 1)
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Insert function body",
			TextEdits: []ast.TextEdit{
				{
					Insertion: " {}",
					Range:     ast.NewUnmeteredRange(pos, pos),
				},
			},
		},
	}
}

// InvalidOptionalChainingError

type InvalidOptionalChainingError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidOptionalChainingError{}
var _ errors.UserError = &InvalidOptionalChainingError{}
var _ errors.SecondaryError = &InvalidOptionalChainingError{}
var _ errors.HasDocumentationLink = &InvalidOptionalChainingError{}
var _ errors.HasSuggestedFixes[ast.TextEdit] = &InvalidOptionalChainingError{}

func (*InvalidOptionalChainingError) isSemanticError() {}

func (*InvalidOptionalChainingError) IsUserError() {}

func (e *InvalidOptionalChainingError) Error() string {
	return fmt.Sprintf(
		"cannot use optional chaining: type %#q is not optional",
		e.Type.QualifiedString(),
	)
}

func (*InvalidOptionalChainingError) SecondaryError() string {
	return "optional chaining (`?.`) can only be used on optional types; " +
		"remove the optional chaining or ensure the type is optional"
}

func (*InvalidOptionalChainingError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/anystruct-anyresource-opts-never#optionals"
}

func (e *InvalidOptionalChainingError) SuggestFixes(_ string) []errors.SuggestedFix[ast.TextEdit] {
	return []errors.SuggestedFix[ast.TextEdit]{
		{
			Message: "Remove optional chaining",
			TextEdits: []ast.TextEdit{
				{
					Replacement: ".",
					Range:       e.Range,
				},
			},
		},
	}
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
var _ errors.SecondaryError = &InvalidAccessError{}
var _ errors.HasDocumentationLink = &InvalidAccessError{}

func (*InvalidAccessError) isSemanticError() {}

func (*InvalidAccessError) IsUserError() {}

func (e *InvalidAccessError) Error() string {
	var possessedDescription string
	if e.PossessedAccess != nil {
		if e.PossessedAccess.Equal(UnauthorizedAccess) {
			possessedDescription = ", but reference is unauthorized"
		} else {
			possessedDescription = fmt.Sprintf(
				", but reference only has %#q authorization",
				e.PossessedAccess.String(),
			)
		}
	}

	return fmt.Sprintf(
		"access denied: cannot access %#q because %s requires %#q authorization%s",
		e.Name,
		e.DeclarationKind.Name(),
		e.RestrictingAccess.String(),
		possessedDescription,
	)
}

func (*InvalidAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

// When e.PossessedAccess is a conjunctive entitlement set, we can suggest
// which additional entitlements it would need to be given in order to have
// e.RequiredAccess.
func (e *InvalidAccessError) SecondaryError() string {
	if !e.SuggestEntitlements || e.PossessedAccess == nil || e.RestrictingAccess == nil {
		return "ensure your reference has the required authorization by using the appropriate access modifier or entitlement"
	}
	possessedEntitlements, possessedOk := e.PossessedAccess.(EntitlementSetAccess)
	requiredEntitlements, requiredOk := e.RestrictingAccess.(EntitlementSetAccess)
	if !possessedOk && e.PossessedAccess.Equal(UnauthorizedAccess) {
		possessedOk = true
		// for this error reporting, model UnauthorizedAccess as an empty entitlement set
		possessedEntitlements = NewEntitlementSetAccess(nil, Conjunction)
	}
	if !possessedOk || !requiredOk || possessedEntitlements.SetKind != Conjunction {
		return "ensure your reference has the required authorization by using the appropriate access modifier or entitlement"
	}

	var sb strings.Builder

	enumerateEntitlements := func(len int, separator string) func(index int, key *EntitlementType, _ struct{}) {
		return func(index int, key *EntitlementType, _ struct{}) {
			fmt.Fprintf(&sb, "%#q", key.QualifiedString())
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
			fmt.Fprint(&sb, "add entitlement ")
			fmt.Fprintf(&sb, "%#q", missingEntitlements.Newest().Key.QualifiedString())
			fmt.Fprint(&sb, " to your reference")
		} else {
			fmt.Fprint(&sb, "add all of these entitlements to your reference: ")
			missingEntitlements.ForeachWithIndex(enumerateEntitlements(missingLen, "and"))
		}

	case Disjunction:
		// when both `required` is a disjunction, we know `possessed` has none of the entitlements in it:
		// suggest adding one of those entitlements
		fmt.Fprint(&sb, "add one of these entitlements to your reference: ")
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
var _ errors.HasDocumentationLink = &InvalidAssignmentAccessError{}

func (*InvalidAssignmentAccessError) isSemanticError() {}

func (*InvalidAssignmentAccessError) IsUserError() {}

func (e *InvalidAssignmentAccessError) Error() string {
	return fmt.Sprintf(
		"cannot assign to %#q: %s has %#q access",
		e.Name,
		e.DeclarationKind.Name(),
		e.RestrictingAccess.String(),
	)
}

func (e *InvalidAssignmentAccessError) SecondaryError() string {
	return fmt.Sprintf(
		"fields with %#q access cannot be directly assigned to; "+
			"consider adding a setter function to %s or using a different access modifier",
		e.RestrictingAccess.String(),
		e.ContainerType.QualifiedString(),
	)
}

func (*InvalidAssignmentAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
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
var _ errors.HasDocumentationLink = &UnauthorizedReferenceAssignmentError{}

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
		"consider taking a reference with %#q or %#q access",
		e.RequiredAccess[0].String(),
		e.RequiredAccess[1].String(),
	)
}

func (*UnauthorizedReferenceAssignmentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/references"
}

// InvalidCharacterLiteralError

type InvalidCharacterLiteralError struct {
	Length int
	ast.Range
}

var _ SemanticError = &InvalidCharacterLiteralError{}
var _ errors.UserError = &InvalidCharacterLiteralError{}
var _ errors.SecondaryError = &InvalidCharacterLiteralError{}
var _ errors.HasDocumentationLink = &InvalidCharacterLiteralError{}

func (*InvalidCharacterLiteralError) isSemanticError() {}

func (*InvalidCharacterLiteralError) IsUserError() {}

func (*InvalidCharacterLiteralError) Error() string {
	return "character literal has invalid length"
}

func (e *InvalidCharacterLiteralError) SecondaryError() string {
	return fmt.Sprintf("expected 1, got %d", e.Length)
}

func (*InvalidCharacterLiteralError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/strings-and-characters"
}

// InvalidFailableResourceDowncastOutsideOptionalBindingError

type InvalidFailableResourceDowncastOutsideOptionalBindingError struct {
	ast.Range
}

var _ SemanticError = &InvalidFailableResourceDowncastOutsideOptionalBindingError{}
var _ errors.UserError = &InvalidFailableResourceDowncastOutsideOptionalBindingError{}
var _ errors.SecondaryError = &InvalidFailableResourceDowncastOutsideOptionalBindingError{}
var _ errors.HasDocumentationLink = &InvalidFailableResourceDowncastOutsideOptionalBindingError{}

func (*InvalidFailableResourceDowncastOutsideOptionalBindingError) isSemanticError() {}

func (*InvalidFailableResourceDowncastOutsideOptionalBindingError) IsUserError() {}

func (*InvalidFailableResourceDowncastOutsideOptionalBindingError) Error() string {
	return "cannot failably downcast resource type outside of optional binding"
}

func (*InvalidFailableResourceDowncastOutsideOptionalBindingError) SecondaryError() string {
	return "resource downcasts must be performed within optional bindings; " +
		"use `if let` or `switch` statements to safely handle the optional result"
}

func (*InvalidFailableResourceDowncastOutsideOptionalBindingError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

// InvalidNonIdentifierFailableResourceDowncast

type InvalidNonIdentifierFailableResourceDowncast struct {
	ast.Range
}

var _ SemanticError = &InvalidNonIdentifierFailableResourceDowncast{}
var _ errors.UserError = &InvalidNonIdentifierFailableResourceDowncast{}
var _ errors.SecondaryError = &InvalidNonIdentifierFailableResourceDowncast{}
var _ errors.HasDocumentationLink = &InvalidNonIdentifierFailableResourceDowncast{}

func (*InvalidNonIdentifierFailableResourceDowncast) isSemanticError() {}

func (*InvalidNonIdentifierFailableResourceDowncast) IsUserError() {}

func (*InvalidNonIdentifierFailableResourceDowncast) Error() string {
	return "cannot failably downcast non-identifier resource"
}

func (*InvalidNonIdentifierFailableResourceDowncast) SecondaryError() string {
	return "failable resource downcasts can only be performed on identifiers; " +
		"consider declaring a variable for this expression"
}

func (*InvalidNonIdentifierFailableResourceDowncast) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

// ReadOnlyTargetAssignmentError

type ReadOnlyTargetAssignmentError struct {
	ast.Range
}

var _ SemanticError = &ReadOnlyTargetAssignmentError{}
var _ errors.UserError = &ReadOnlyTargetAssignmentError{}
var _ errors.SecondaryError = &ReadOnlyTargetAssignmentError{}
var _ errors.HasDocumentationLink = &ReadOnlyTargetAssignmentError{}

func (*ReadOnlyTargetAssignmentError) isSemanticError() {}

func (*ReadOnlyTargetAssignmentError) IsUserError() {}

func (*ReadOnlyTargetAssignmentError) Error() string {
	return "cannot assign to read-only target"
}

func (*ReadOnlyTargetAssignmentError) SecondaryError() string {
	return "read-only targets cannot be modified through assignment; " +
		"ensure the assignment target is a mutable"
}

func (*ReadOnlyTargetAssignmentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/constants-and-variables"
}

// InvalidTransactionBlockError

type InvalidTransactionBlockError struct {
	Name string
	Pos  ast.Position
}

var _ SemanticError = &InvalidTransactionBlockError{}
var _ errors.UserError = &InvalidTransactionBlockError{}
var _ errors.SecondaryError = &InvalidTransactionBlockError{}
var _ errors.HasDocumentationLink = &InvalidTransactionBlockError{}

func (*InvalidTransactionBlockError) isSemanticError() {}

func (*InvalidTransactionBlockError) IsUserError() {}

func (*InvalidTransactionBlockError) Error() string {
	return "invalid transaction block"
}

func (e *InvalidTransactionBlockError) SecondaryError() string {
	return fmt.Sprintf(
		"expected `prepare` or `execute`, got %#q",
		e.Name,
	)
}

func (*InvalidTransactionBlockError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
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
var _ errors.SecondaryError = &TransactionMissingPrepareError{}
var _ errors.HasDocumentationLink = &TransactionMissingPrepareError{}

func (*TransactionMissingPrepareError) isSemanticError() {}

func (*TransactionMissingPrepareError) IsUserError() {}

func (e *TransactionMissingPrepareError) Error() string {
	return fmt.Sprintf(
		"transaction missing prepare function for field %#q",
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

func (*TransactionMissingPrepareError) SecondaryError() string {
	return "transactions with fields must have a `prepare` block to initialize them; " +
		"add a `prepare` function and initialize transaction fields"
}

func (*TransactionMissingPrepareError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
}

// InvalidResourceTransactionParameterError

type InvalidResourceTransactionParameterError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidResourceTransactionParameterError{}
var _ errors.UserError = &InvalidResourceTransactionParameterError{}
var _ errors.SecondaryError = &InvalidResourceTransactionParameterError{}
var _ errors.HasDocumentationLink = &InvalidResourceTransactionParameterError{}

func (*InvalidResourceTransactionParameterError) isSemanticError() {}

func (*InvalidResourceTransactionParameterError) IsUserError() {}

func (e *InvalidResourceTransactionParameterError) Error() string {
	return fmt.Sprintf(
		"transaction parameter must not be resource type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*InvalidResourceTransactionParameterError) SecondaryError() string {
	return "transaction parameters must be non-resource types"
}

func (*InvalidResourceTransactionParameterError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
}

// InvalidNonImportableTransactionParameterTypeError

type InvalidNonImportableTransactionParameterTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidNonImportableTransactionParameterTypeError{}
var _ errors.UserError = &InvalidNonImportableTransactionParameterTypeError{}
var _ errors.SecondaryError = &InvalidNonImportableTransactionParameterTypeError{}
var _ errors.HasDocumentationLink = &InvalidNonImportableTransactionParameterTypeError{}

func (*InvalidNonImportableTransactionParameterTypeError) isSemanticError() {}

func (*InvalidNonImportableTransactionParameterTypeError) IsUserError() {}

func (e *InvalidNonImportableTransactionParameterTypeError) Error() string {
	return fmt.Sprintf(
		"transaction parameter must be importable: %#q",
		e.Type.QualifiedString(),
	)
}

func (*InvalidNonImportableTransactionParameterTypeError) SecondaryError() string {
	return "use a concrete type that can be imported"
}

func (*InvalidNonImportableTransactionParameterTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
}

// InvalidTransactionFieldAccessModifierError

type InvalidTransactionFieldAccessModifierError struct {
	Name   string
	Access ast.Access
	Pos    ast.Position
}

var _ SemanticError = &InvalidTransactionFieldAccessModifierError{}
var _ errors.UserError = &InvalidTransactionFieldAccessModifierError{}
var _ errors.SecondaryError = &InvalidTransactionFieldAccessModifierError{}
var _ errors.HasDocumentationLink = &InvalidTransactionFieldAccessModifierError{}

func (*InvalidTransactionFieldAccessModifierError) isSemanticError() {}

func (*InvalidTransactionFieldAccessModifierError) IsUserError() {}

func (e *InvalidTransactionFieldAccessModifierError) Error() string {
	return fmt.Sprintf(
		"access modifier not allowed for transaction field %#q: %#q",
		e.Name,
		e.Access.Keyword(),
	)
}

func (*InvalidTransactionFieldAccessModifierError) SecondaryError() string {
	return "transaction fields cannot have access modifiers"
}

func (*InvalidTransactionFieldAccessModifierError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions"
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
var _ errors.SecondaryError = &InvalidTransactionPrepareParameterTypeError{}
var _ errors.HasDocumentationLink = &InvalidTransactionPrepareParameterTypeError{}

func (*InvalidTransactionPrepareParameterTypeError) isSemanticError() {}

func (*InvalidTransactionPrepareParameterTypeError) IsUserError() {}

func (e *InvalidTransactionPrepareParameterTypeError) Error() string {
	return fmt.Sprintf(
		"parameter for `prepare` must be subtype of %#q, not %#q",
		AccountReferenceType.QualifiedString(),
		e.Type.QualifiedString(),
	)
}

func (*InvalidTransactionPrepareParameterTypeError) SecondaryError() string {
	return "parameters of `prepare` functions must be account references"
}

func (*InvalidTransactionPrepareParameterTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/transactions#prepare-phase"
}

// InvalidNestedDeclarationError

type InvalidNestedDeclarationError struct {
	NestedDeclarationKind    common.DeclarationKind
	ContainerDeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidNestedDeclarationError{}
var _ errors.UserError = &InvalidNestedDeclarationError{}
var _ errors.SecondaryError = &InvalidNestedDeclarationError{}
var _ errors.HasDocumentationLink = &InvalidNestedDeclarationError{}

func (*InvalidNestedDeclarationError) isSemanticError() {}

func (*InvalidNestedDeclarationError) IsUserError() {}

func (e *InvalidNestedDeclarationError) Error() string {
	return fmt.Sprintf(
		"%s declarations cannot be nested inside %s declarations",
		e.NestedDeclarationKind.Name(),
		e.ContainerDeclarationKind.Name(),
	)
}

func (e *InvalidNestedDeclarationError) SecondaryError() string {
	return fmt.Sprintf(
		"only certain declaration types can be nested within %s; "+
			"%s declarations are not allowed in this context",
		e.ContainerDeclarationKind.Name(),
		e.NestedDeclarationKind.Name(),
	)
}

func (*InvalidNestedDeclarationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
}

// InvalidNestedTypeError

type InvalidNestedTypeError struct {
	Type *ast.NominalType
}

var _ SemanticError = &InvalidNestedTypeError{}
var _ errors.UserError = &InvalidNestedTypeError{}
var _ errors.SecondaryError = &InvalidNestedTypeError{}
var _ errors.HasDocumentationLink = &InvalidNestedTypeError{}

func (*InvalidNestedTypeError) isSemanticError() {}

func (*InvalidNestedTypeError) IsUserError() {}

func (e *InvalidNestedTypeError) Error() string {
	return fmt.Sprintf(
		"type does not support nested types: %#q",
		e.Type,
	)
}

func (e *InvalidNestedTypeError) SecondaryError() string {
	return fmt.Sprintf(
		"only contract types can contain nested types; "+
			"the type %#q is not a contract type",
		e.Type,
	)
}

func (*InvalidNestedTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/composite-types"
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
var _ errors.SecondaryError = &InvalidEnumCaseError{}
var _ errors.HasDocumentationLink = &InvalidEnumCaseError{}

func (*InvalidEnumCaseError) isSemanticError() {}

func (*InvalidEnumCaseError) IsUserError() {}

func (e *InvalidEnumCaseError) Error() string {
	return fmt.Sprintf(
		"%s declaration does not allow enum cases",
		e.ContainerDeclarationKind.Name(),
	)
}

func (*InvalidEnumCaseError) SecondaryError() string {
	return "enum cases can only be declared within enum types"
}

func (*InvalidEnumCaseError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/enumerations"
}

// InvalidNonEnumCaseError

type InvalidNonEnumCaseError struct {
	ContainerDeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidNonEnumCaseError{}
var _ errors.UserError = &InvalidNonEnumCaseError{}
var _ errors.SecondaryError = &InvalidNonEnumCaseError{}
var _ errors.HasDocumentationLink = &InvalidNonEnumCaseError{}

func (*InvalidNonEnumCaseError) isSemanticError() {}

func (*InvalidNonEnumCaseError) IsUserError() {}

func (e *InvalidNonEnumCaseError) Error() string {
	return fmt.Sprintf(
		"%s declaration only allows enum cases",
		e.ContainerDeclarationKind.Name(),
	)
}

func (*InvalidNonEnumCaseError) SecondaryError() string {
	return "move this declaration outside the enum or convert it to an enum case"
}

func (*InvalidNonEnumCaseError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/enumerations"
}

// InvalidTopLevelDeclarationError

type InvalidTopLevelDeclarationError struct {
	DeclarationKind common.DeclarationKind
	ast.Range
}

var _ SemanticError = &InvalidTopLevelDeclarationError{}
var _ errors.UserError = &InvalidTopLevelDeclarationError{}
var _ errors.SecondaryError = &InvalidTopLevelDeclarationError{}

func (*InvalidTopLevelDeclarationError) isSemanticError() {}

func (*InvalidTopLevelDeclarationError) IsUserError() {}

func (e *InvalidTopLevelDeclarationError) Error() string {
	return fmt.Sprintf(
		"%s declarations are not valid at the top-level",
		e.DeclarationKind.Name(),
	)
}

func (*InvalidTopLevelDeclarationError) SecondaryError() string {
	return "move this declaration into a contract"
}

// InvalidSelfInvalidationError

type InvalidSelfInvalidationError struct {
	InvalidationKind ResourceInvalidationKind
	ast.Range
}

var _ SemanticError = &InvalidSelfInvalidationError{}
var _ errors.UserError = &InvalidSelfInvalidationError{}
var _ errors.SecondaryError = &InvalidSelfInvalidationError{}
var _ errors.HasDocumentationLink = &InvalidSelfInvalidationError{}

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

func (*InvalidSelfInvalidationError) SecondaryError() string {
	return "`self` cannot be moved or destroyed within the resource's own methods"
}

func (*InvalidSelfInvalidationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
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
		"cannot move %s: %#q",
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
var _ errors.HasDocumentationLink = &ConstantSizedArrayLiteralSizeError{}

func (*ConstantSizedArrayLiteralSizeError) isSemanticError() {}

func (*ConstantSizedArrayLiteralSizeError) IsUserError() {}

func (*ConstantSizedArrayLiteralSizeError) Error() string {
	return "incorrect number of array literal elements"
}

func (e *ConstantSizedArrayLiteralSizeError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %d, got %d",
		e.ExpectedSize,
		e.ActualSize,
	)
}

func (*ConstantSizedArrayLiteralSizeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/arrays"
}

// InvalidIntersectedTypeError

type InvalidIntersectedTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidIntersectedTypeError{}
var _ errors.UserError = &InvalidIntersectedTypeError{}
var _ errors.SecondaryError = &InvalidIntersectedTypeError{}
var _ errors.HasDocumentationLink = &InvalidIntersectedTypeError{}

func (*InvalidIntersectedTypeError) isSemanticError() {}

func (*InvalidIntersectedTypeError) IsUserError() {}

func (e *InvalidIntersectedTypeError) Error() string {
	return fmt.Sprintf(
		"intersection type with invalid non-interface type: %#q",
		e.Type.QualifiedString(),
	)
}

func (e *InvalidIntersectedTypeError) SecondaryError() string {
	return fmt.Sprintf(
		"only interface types can be intersected; the type %#q is not an interface",
		e.Type.QualifiedString(),
	)
}

func (*InvalidIntersectedTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
}

// IntersectionCompositeKindMismatchError

type IntersectionCompositeKindMismatchError struct {
	CompositeKind         common.CompositeKind
	PreviousCompositeKind common.CompositeKind
	ast.Range
}

var _ SemanticError = &IntersectionCompositeKindMismatchError{}
var _ errors.UserError = &IntersectionCompositeKindMismatchError{}
var _ errors.SecondaryError = &IntersectionCompositeKindMismatchError{}
var _ errors.HasDocumentationLink = &IntersectionCompositeKindMismatchError{}

func (*IntersectionCompositeKindMismatchError) isSemanticError() {}

func (*IntersectionCompositeKindMismatchError) IsUserError() {}

func (*IntersectionCompositeKindMismatchError) Error() string {
	return "interface kinds in intersection type must match"
}

func (e *IntersectionCompositeKindMismatchError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %#q, got %#q; all interfaces in an intersection type "+
			"must have the same composite kind (struct, resource, contract, etc.)",
		e.PreviousCompositeKind.Name(),
		e.CompositeKind.Name(),
	)
}

func (*IntersectionCompositeKindMismatchError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
}

// InvalidIntersectionTypeDuplicateError

type InvalidIntersectionTypeDuplicateError struct {
	Type *InterfaceType
	ast.Range
}

var _ SemanticError = &InvalidIntersectionTypeDuplicateError{}
var _ errors.UserError = &InvalidIntersectionTypeDuplicateError{}
var _ errors.SecondaryError = &InvalidIntersectionTypeDuplicateError{}
var _ errors.HasDocumentationLink = &InvalidIntersectionTypeDuplicateError{}

func (*InvalidIntersectionTypeDuplicateError) isSemanticError() {}

func (*InvalidIntersectionTypeDuplicateError) IsUserError() {}

func (e *InvalidIntersectionTypeDuplicateError) Error() string {
	return fmt.Sprintf(
		"duplicate intersected type: %#q",
		e.Type.QualifiedString(),
	)
}

func (e *InvalidIntersectionTypeDuplicateError) SecondaryError() string {
	return fmt.Sprintf(
		"each interface type can only appear once in an intersection; "+
			"remove the duplicate %#q",
		e.Type.QualifiedString(),
	)
}

func (*InvalidIntersectionTypeDuplicateError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
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
var _ errors.SecondaryError = &IntersectionMemberClashError{}
var _ errors.HasDocumentationLink = &IntersectionMemberClashError{}

func (*IntersectionMemberClashError) isSemanticError() {}

func (*IntersectionMemberClashError) IsUserError() {}

func (e *IntersectionMemberClashError) Error() string {
	return fmt.Sprintf(
		"member %#q conflicts between intersection types",
		e.Name,
	)
}

func (e *IntersectionMemberClashError) SecondaryError() string {
	return fmt.Sprintf(
		"member %#q is declared in both %#q and %#q; "+
			"intersection types cannot have conflicting member declarations with the same name",
		e.Name,
		e.OriginalDeclaringType.QualifiedString(),
		e.RedeclaringType.QualifiedString(),
	)
}

func (*IntersectionMemberClashError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
}

// AmbiguousIntersectionTypeError

type AmbiguousIntersectionTypeError struct {
	ast.Range
}

var _ SemanticError = &AmbiguousIntersectionTypeError{}
var _ errors.UserError = &AmbiguousIntersectionTypeError{}
var _ errors.SecondaryError = &AmbiguousIntersectionTypeError{}
var _ errors.HasDocumentationLink = &AmbiguousIntersectionTypeError{}

func (*AmbiguousIntersectionTypeError) isSemanticError() {}

func (*AmbiguousIntersectionTypeError) IsUserError() {}

func (*AmbiguousIntersectionTypeError) Error() string {
	return "ambiguous intersection type"
}

func (*AmbiguousIntersectionTypeError) SecondaryError() string {
	return "empty intersection types like `{}` or `@{}` are ambiguous; " +
		"specify the interfaces to intersect"
}

func (*AmbiguousIntersectionTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/types-and-type-system/intersection-types"
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
var _ errors.HasDocumentationLink = &InvalidPathDomainError{}

func (*InvalidPathDomainError) isSemanticError() {}

func (*InvalidPathDomainError) IsUserError() {}

func (e *InvalidPathIdentifierError) Error() string {
	return fmt.Sprintf("invalid path identifier %s", e.ActualIdentifier)
}

var validPathDomainDescription = func() string {
	words := make([]string, 0, len(common.AllPathDomains))

	for _, domain := range common.AllPathDomains {
		words = append(words, fmt.Sprintf("%#q", domain))
	}

	return common.EnumerateWords(words, "or")
}()

func (e *InvalidPathDomainError) SecondaryError() string {
	return fmt.Sprintf(
		"expected one of %s; got %#q",
		validPathDomainDescription,
		e.ActualDomain,
	)
}

func (*InvalidPathDomainError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/accounts/paths"
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
var _ errors.HasDocumentationLink = &InvalidTypeArgumentCountError{}

func (e *InvalidTypeArgumentCountError) isSemanticError() {}

func (*InvalidTypeArgumentCountError) IsUserError() {}

func (*InvalidTypeArgumentCountError) Error() string {
	return "incorrect number of type arguments"
}

func (e *InvalidTypeArgumentCountError) SecondaryError() string {
	return fmt.Sprintf(
		"expected up to %d, got %d",
		e.TypeParameterCount,
		e.TypeArgumentCount,
	)
}

func (*InvalidTypeArgumentCountError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/capabilities"
}

// MissingTypeArgumentError

type MissingTypeArgumentError struct {
	TypeParameterName string
	ast.Range
}

var _ SemanticError = &MissingTypeArgumentError{}
var _ errors.UserError = &MissingTypeArgumentError{}
var _ errors.SecondaryError = &MissingTypeArgumentError{}
var _ errors.HasDocumentationLink = &MissingTypeArgumentError{}

func (e *MissingTypeArgumentError) isSemanticError() {}

func (*MissingTypeArgumentError) IsUserError() {}

func (e *MissingTypeArgumentError) Error() string {
	return fmt.Sprintf(
		"missing type argument for non-optional type parameter %#q",
		e.TypeParameterName,
	)
}

func (e *MissingTypeArgumentError) SecondaryError() string {
	return fmt.Sprintf(
		"provide an explicit type argument for type parameter %#q",
		e.TypeParameterName,
	)
}

func (*MissingTypeArgumentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
}

// InvalidTypeArgumentError

type InvalidTypeArgumentError struct {
	TypeArgumentName string
	Details          string
	ast.Range
}

var _ SemanticError = &InvalidTypeArgumentError{}
var _ errors.UserError = &InvalidTypeArgumentError{}
var _ errors.SecondaryError = &InvalidTypeArgumentError{}
var _ errors.HasDocumentationLink = &InvalidTypeArgumentError{}

func (*InvalidTypeArgumentError) isSemanticError() {}

func (*InvalidTypeArgumentError) IsUserError() {}

func (e *InvalidTypeArgumentError) Error() string {
	return fmt.Sprintf(
		"invalid type argument for type parameter %#q",
		e.TypeArgumentName,
	)
}

func (e *InvalidTypeArgumentError) SecondaryError() string {
	return e.Details
}

func (*InvalidTypeArgumentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
}

// TypeParameterTypeInferenceError

type TypeParameterTypeInferenceError struct {
	Name string
	ast.Range
}

var _ SemanticError = &TypeParameterTypeInferenceError{}
var _ errors.UserError = &TypeParameterTypeInferenceError{}
var _ errors.SecondaryError = &TypeParameterTypeInferenceError{}
var _ errors.HasDocumentationLink = &TypeParameterTypeInferenceError{}

func (e *TypeParameterTypeInferenceError) isSemanticError() {}

func (*TypeParameterTypeInferenceError) IsUserError() {}

func (e *TypeParameterTypeInferenceError) Error() string {
	return fmt.Sprintf(
		"cannot infer type argument for type parameter %#q",
		e.Name,
	)
}

func (e *TypeParameterTypeInferenceError) SecondaryError() string {
	return fmt.Sprintf(
		"provide an explicit type argument for type parameter %#q to resolve the ambiguity",
		e.Name,
	)
}

func (*TypeParameterTypeInferenceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
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
var _ errors.HasDocumentationLink = &InvalidConstantSizedTypeBaseError{}

func (e *InvalidConstantSizedTypeBaseError) isSemanticError() {}

func (*InvalidConstantSizedTypeBaseError) IsUserError() {}

func (*InvalidConstantSizedTypeBaseError) Error() string {
	return "invalid base for constant sized type size"
}

func (e *InvalidConstantSizedTypeBaseError) SecondaryError() string {
	return fmt.Sprintf(
		"expected %d, got %d",
		e.ActualBase,
		e.ExpectedBase,
	)
}

func (*InvalidConstantSizedTypeBaseError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/arrays"
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
var _ errors.HasDocumentationLink = &InvalidConstantSizedTypeSizeError{}

func (*InvalidConstantSizedTypeSizeError) isSemanticError() {}

func (*InvalidConstantSizedTypeSizeError) IsUserError() {}

func (*InvalidConstantSizedTypeSizeError) Error() string {
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

func (*InvalidConstantSizedTypeSizeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types/arrays"
}

// UnsupportedResourceForLoopError

type UnsupportedResourceForLoopError struct {
	ast.Range
}

var _ SemanticError = &UnsupportedResourceForLoopError{}
var _ errors.UserError = &UnsupportedResourceForLoopError{}
var _ errors.SecondaryError = &UnsupportedResourceForLoopError{}
var _ errors.HasDocumentationLink = &UnsupportedResourceForLoopError{}

func (*UnsupportedResourceForLoopError) isSemanticError() {}

func (*UnsupportedResourceForLoopError) IsUserError() {}

func (*UnsupportedResourceForLoopError) Error() string {
	return "cannot loop over resources"
}

func (*UnsupportedResourceForLoopError) SecondaryError() string {
	return "looping directly over a collection of resources is not supported; " +
		"use a different data structure or iterate over resource references"
}

func (*UnsupportedResourceForLoopError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

// TypeParameterTypeMismatchError

// TODO: Add suggested fix for type mismatch

type TypeParameterTypeMismatchError struct {
	TypeParameter *TypeParameter
	ExpectedType  Type
	ActualType    Type
	ast.Range
}

var _ SemanticError = &TypeParameterTypeMismatchError{}
var _ errors.UserError = &TypeParameterTypeMismatchError{}
var _ errors.SecondaryError = &TypeParameterTypeMismatchError{}
var _ errors.HasDocumentationLink = &TypeParameterTypeMismatchError{}

func (*TypeParameterTypeMismatchError) isSemanticError() {}

func (*TypeParameterTypeMismatchError) IsUserError() {}

func (*TypeParameterTypeMismatchError) Error() string {
	return "mismatched types for type parameter"
}

func (e *TypeParameterTypeMismatchError) SecondaryError() string {
	expected, actual := ErrorMessageExpectedActualTypes(
		e.ExpectedType,
		e.ActualType,
	)

	return fmt.Sprintf(
		"type parameter %s is bound to %#q, but got %#q here",
		e.TypeParameter.Name,
		expected,
		actual,
	)
}

func (*TypeParameterTypeMismatchError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
}

// UnparameterizedTypeInstantiationError

type UnparameterizedTypeInstantiationError struct {
	ActualTypeArgumentCount int
	ast.Range
}

var _ SemanticError = &UnparameterizedTypeInstantiationError{}
var _ errors.UserError = &UnparameterizedTypeInstantiationError{}
var _ errors.SecondaryError = &UnparameterizedTypeInstantiationError{}
var _ errors.HasDocumentationLink = &UnparameterizedTypeInstantiationError{}

func (*UnparameterizedTypeInstantiationError) isSemanticError() {}

func (*UnparameterizedTypeInstantiationError) IsUserError() {}

func (*UnparameterizedTypeInstantiationError) Error() string {
	return "cannot instantiate non-parameterized type"
}

func (e *UnparameterizedTypeInstantiationError) SecondaryError() string {
	return fmt.Sprintf(
		"expected no type arguments, got %d",
		e.ActualTypeArgumentCount,
	)
}

func (*UnparameterizedTypeInstantiationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
}

// TypeAnnotationRequiredError

type TypeAnnotationRequiredError struct {
	Cause string
	Pos   ast.Position
}

var _ SemanticError = &TypeAnnotationRequiredError{}
var _ errors.UserError = &TypeAnnotationRequiredError{}
var _ errors.SecondaryError = &TypeAnnotationRequiredError{}
var _ errors.HasDocumentationLink = &TypeAnnotationRequiredError{}

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

func (*TypeAnnotationRequiredError) SecondaryError() string {
	return "add an explicit type annotation to resolve the ambiguity"
}

func (*TypeAnnotationRequiredError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
}

// CyclicImportsError

type CyclicImportsError struct {
	Location common.Location
	ast.Range
}

var _ SemanticError = &CyclicImportsError{}
var _ errors.UserError = &CyclicImportsError{}
var _ errors.SecondaryError = &CyclicImportsError{}
var _ errors.HasDocumentationLink = &CyclicImportsError{}

func (*CyclicImportsError) isSemanticError() {}

func (*CyclicImportsError) IsUserError() {}

func (e *CyclicImportsError) Error() string {
	return fmt.Sprintf("cyclic import of %#q", e.Location)
}

func (*CyclicImportsError) SecondaryError() string {
	return "circular dependencies between imports are not allowed; " +
		"break the cycle by removing one of the import statements"
}

func (*CyclicImportsError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}

// SwitchDefaultPositionError

type SwitchDefaultPositionError struct {
	ast.Range
}

var _ SemanticError = &SwitchDefaultPositionError{}
var _ errors.UserError = &SwitchDefaultPositionError{}
var _ errors.SecondaryError = &SwitchDefaultPositionError{}
var _ errors.HasDocumentationLink = &SwitchDefaultPositionError{}

func (*SwitchDefaultPositionError) isSemanticError() {}

func (*SwitchDefaultPositionError) IsUserError() {}

func (*SwitchDefaultPositionError) Error() string {
	return "the `default` case must appear at the end of a `switch` statement"
}

func (*SwitchDefaultPositionError) SecondaryError() string {
	return "the `default` case must be the last case in a `switch` statement; " +
		"move the default case to the end of the switch statement"
}

func (*SwitchDefaultPositionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/control-flow#switch-statement"
}

// MissingSwitchCaseStatementsError

type MissingSwitchCaseStatementsError struct {
	Pos ast.Position
}

var _ SemanticError = &MissingSwitchCaseStatementsError{}
var _ errors.UserError = &MissingSwitchCaseStatementsError{}
var _ errors.SecondaryError = &MissingSwitchCaseStatementsError{}
var _ errors.HasDocumentationLink = &MissingSwitchCaseStatementsError{}

func (*MissingSwitchCaseStatementsError) isSemanticError() {}

func (*MissingSwitchCaseStatementsError) IsUserError() {}

func (*MissingSwitchCaseStatementsError) Error() string {
	return "switch cases must have at least one statement"
}

func (*MissingSwitchCaseStatementsError) SecondaryError() string {
	return "switch cases cannot be empty; " +
		"add at least one statement to the switch case"
}

func (*MissingSwitchCaseStatementsError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/control-flow#switch-statement"
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
var _ errors.SecondaryError = &MissingEntryPointError{}

func (*MissingEntryPointError) IsUserError() {}

func (e *MissingEntryPointError) Error() string {
	return fmt.Sprintf("missing entry point: expected function %#q", e.Expected)
}

func (*MissingEntryPointError) SecondaryError() string {
	return "add the missing function declaration"
}

// InvalidEntryPointError

type InvalidEntryPointTypeError struct {
	Type Type
}

var _ errors.UserError = &InvalidEntryPointTypeError{}
var _ errors.SecondaryError = &InvalidEntryPointTypeError{}

func (*InvalidEntryPointTypeError) IsUserError() {}

func (e *InvalidEntryPointTypeError) Error() string {
	return fmt.Sprintf(
		"invalid entry point type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*InvalidEntryPointTypeError) SecondaryError() string {
	return "entry points must have the correct type; " +
		"ensure the function has the correct parameters and return type"
}

// PurityError

type PurityError struct {
	ast.Range
}

var _ SemanticError = &PurityError{}
var _ errors.UserError = &PurityError{}
var _ errors.SecondaryError = &PurityError{}
var _ errors.HasDocumentationLink = &PurityError{}

func (*PurityError) IsUserError() {}

func (*PurityError) isSemanticError() {}

func (*PurityError) Error() string {
	return "impure operation performed in view context"
}

func (*PurityError) SecondaryError() string {
	return "view functions can only perform pure operations that don't modify state"
}

func (*PurityError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions#view-functions"
}

// InvalidatedResourceReferenceError

type InvalidatedResourceReferenceError struct {
	Invalidation ResourceInvalidation
	ast.Range
}

var _ SemanticError = &InvalidatedResourceReferenceError{}
var _ errors.UserError = &InvalidatedResourceReferenceError{}
var _ errors.SecondaryError = &InvalidatedResourceReferenceError{}
var _ errors.HasDocumentationLink = &InvalidatedResourceReferenceError{}

func (*InvalidatedResourceReferenceError) isSemanticError() {}

func (*InvalidatedResourceReferenceError) IsUserError() {}

func (*InvalidatedResourceReferenceError) Error() string {
	return "invalid reference: referenced resource may have been moved or destroyed"
}

func (e *InvalidatedResourceReferenceError) SecondaryError() string {
	return "once a resource got invalidated (moved or destroyed), " +
		"references to the resource are no longer usable; " +
		"consider moving the resource use before the invalidation"
}

func (e *InvalidatedResourceReferenceError) ErrorNotes() []errors.ErrorNote {
	invalidation := e.Invalidation
	return []errors.ErrorNote{
		newPreviousResourceInvalidationNote(invalidation),
	}
}

func (*InvalidatedResourceReferenceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources"
}

// InvalidEntitlementAccessError
type InvalidEntitlementAccessError struct {
	Pos ast.Position
}

var _ SemanticError = &InvalidEntitlementAccessError{}
var _ errors.UserError = &InvalidEntitlementAccessError{}
var _ errors.SecondaryError = &InvalidEntitlementAccessError{}
var _ errors.HasDocumentationLink = &InvalidEntitlementAccessError{}

func (*InvalidEntitlementAccessError) isSemanticError() {}

func (*InvalidEntitlementAccessError) IsUserError() {}

func (*InvalidEntitlementAccessError) Error() string {
	return "only struct or resource members may be declared with entitlement access"
}

func (*InvalidEntitlementAccessError) SecondaryError() string {
	return "entitlement access modifiers can only be used on fields and functions within struct and resource types; " +
		"use regular access modifiers for other declarations"
}

func (*InvalidEntitlementAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlements"
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
var _ errors.SecondaryError = &InvalidEntitlementMappingTypeError{}
var _ errors.HasDocumentationLink = &InvalidEntitlementMappingTypeError{}

func (*InvalidEntitlementMappingTypeError) isSemanticError() {}

func (*InvalidEntitlementMappingTypeError) IsUserError() {}

func (e *InvalidEntitlementMappingTypeError) Error() string {
	return fmt.Sprintf(
		"%#q is not an entitlement map type",
		e.Type.QualifiedString(),
	)
}

func (*InvalidEntitlementMappingTypeError) SecondaryError() string {
	return "consider removing the `mapping` keyword"
}

func (*InvalidEntitlementMappingTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlements"
}

func (e *InvalidEntitlementMappingTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidEntitlementMappingTypeError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidNonEntitlementTypeInMappingError
type InvalidNonEntitlementTypeInMappingError struct {
	Pos ast.Position
}

var _ SemanticError = &InvalidNonEntitlementTypeInMappingError{}
var _ errors.UserError = &InvalidNonEntitlementTypeInMappingError{}
var _ errors.SecondaryError = &InvalidNonEntitlementTypeInMappingError{}
var _ errors.HasDocumentationLink = &InvalidNonEntitlementTypeInMappingError{}

func (*InvalidNonEntitlementTypeInMappingError) isSemanticError() {}

func (*InvalidNonEntitlementTypeInMappingError) IsUserError() {}

func (*InvalidNonEntitlementTypeInMappingError) Error() string {
	return "cannot use non-entitlement type in entitlement mapping"
}

func (e *InvalidNonEntitlementTypeInMappingError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidNonEntitlementTypeInMappingError) EndPosition(common.MemoryGauge) ast.Position {
	return e.Pos
}

func (*InvalidNonEntitlementTypeInMappingError) SecondaryError() string {
	return "entitlement mappings can only contain entitlement types"
}

func (*InvalidNonEntitlementTypeInMappingError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
}

// InvalidMappingAccessError
type InvalidMappingAccessError struct {
	Pos ast.Position
}

var _ SemanticError = &InvalidMappingAccessError{}
var _ errors.UserError = &InvalidMappingAccessError{}
var _ errors.SecondaryError = &InvalidMappingAccessError{}
var _ errors.HasDocumentationLink = &InvalidMappingAccessError{}

func (*InvalidMappingAccessError) isSemanticError() {}

func (*InvalidMappingAccessError) IsUserError() {}

func (*InvalidMappingAccessError) Error() string {
	return "`access(mapping ...)` may only be used in structs and resources"
}

func (*InvalidMappingAccessError) SecondaryError() string {
	return "entitlement mapping access modifiers are only allowed on struct and resource members; " +
		"use regular access modifiers for other declaration types"
}

func (*InvalidMappingAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
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
var _ errors.SecondaryError = &InvalidMappingAccessMemberTypeError{}
var _ errors.HasDocumentationLink = &InvalidMappingAccessMemberTypeError{}

func (*InvalidMappingAccessMemberTypeError) isSemanticError() {}

func (*InvalidMappingAccessMemberTypeError) IsUserError() {}

func (*InvalidMappingAccessMemberTypeError) Error() string {
	return "invalid type for `access(mapping ...)` declaration"
}

func (*InvalidMappingAccessMemberTypeError) SecondaryError() string {
	return "only entitlement mapping types can be used in `access(mapping ...)` declarations; " +
		"use regular access modifiers for other types"
}

func (*InvalidMappingAccessMemberTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
}

func (e *InvalidMappingAccessMemberTypeError) StartPosition() ast.Position {
	return e.Pos
}

func (e *InvalidMappingAccessMemberTypeError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

// InvalidNonEntitlementAccessError
type InvalidNonEntitlementAccessError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &InvalidNonEntitlementAccessError{}
var _ errors.UserError = &InvalidNonEntitlementAccessError{}
var _ errors.SecondaryError = &InvalidNonEntitlementAccessError{}
var _ errors.HasDocumentationLink = &InvalidNonEntitlementAccessError{}

func (*InvalidNonEntitlementAccessError) isSemanticError() {}

func (*InvalidNonEntitlementAccessError) IsUserError() {}

func (*InvalidNonEntitlementAccessError) Error() string {
	return "only entitlements may be used in access modifiers"
}

func (e *InvalidNonEntitlementAccessError) SecondaryError() string {
	return fmt.Sprintf(
		"%#q is not an entitlement",
		e.Type.QualifiedString(),
	)
}

func (*InvalidNonEntitlementAccessError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlements"
}

// MappingAccessMissingKeywordError
type MappingAccessMissingKeywordError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &MappingAccessMissingKeywordError{}
var _ errors.UserError = &MappingAccessMissingKeywordError{}
var _ errors.HasDocumentationLink = &MappingAccessMissingKeywordError{}

func (*MappingAccessMissingKeywordError) isSemanticError() {}

func (*MappingAccessMissingKeywordError) IsUserError() {}

func (*MappingAccessMissingKeywordError) Error() string {
	return "entitlement mapping access modifiers require the `mapping` keyword " +
		"preceding the name of the entitlement mapping"
}

func (e *MappingAccessMissingKeywordError) SecondaryError() string {
	return fmt.Sprintf(
		"replace %#[1]q with `mapping %[1]s`",
		e.Type.QualifiedString(),
	)
}

func (*MappingAccessMissingKeywordError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control"
}

// DirectEntitlementAnnotationError
type DirectEntitlementAnnotationError struct {
	ast.Range
}

var _ SemanticError = &DirectEntitlementAnnotationError{}
var _ errors.UserError = &DirectEntitlementAnnotationError{}
var _ errors.SecondaryError = &DirectEntitlementAnnotationError{}
var _ errors.HasDocumentationLink = &DirectEntitlementAnnotationError{}

func (*DirectEntitlementAnnotationError) isSemanticError() {}

func (*DirectEntitlementAnnotationError) IsUserError() {}

func (*DirectEntitlementAnnotationError) Error() string {
	return "cannot use an entitlement type outside of an `access` declaration or `auth` modifier"
}

func (*DirectEntitlementAnnotationError) SecondaryError() string {
	return "entitlements can only be used in access modifiers of members (fields and functions) " +
		"or in authorized references"
}

func (*DirectEntitlementAnnotationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlements"
}

// UnrepresentableEntitlementMapOutputError
type UnrepresentableEntitlementMapOutputError struct {
	Input EntitlementSetAccess
	Map   *EntitlementMapType
	ast.Range
}

var _ SemanticError = &UnrepresentableEntitlementMapOutputError{}
var _ errors.UserError = &UnrepresentableEntitlementMapOutputError{}
var _ errors.HasDocumentationLink = &UnrepresentableEntitlementMapOutputError{}

func (*UnrepresentableEntitlementMapOutputError) isSemanticError() {}

func (*UnrepresentableEntitlementMapOutputError) IsUserError() {}

func (e *UnrepresentableEntitlementMapOutputError) Error() string {
	return fmt.Sprintf(
		"cannot map %#q through %#q because the output is unrepresentable",
		e.Input.String(),
		e.Map.QualifiedString(),
	)
}

func (e *UnrepresentableEntitlementMapOutputError) SecondaryError() string {
	return fmt.Sprintf(
		"this usually occurs because the input set is disjunctive and %#q is one-to-many",
		e.Map.QualifiedString(),
	)
}

func (*UnrepresentableEntitlementMapOutputError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
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
var _ errors.SecondaryError = &InvalidEntitlementMappingInclusionError{}
var _ errors.HasDocumentationLink = &InvalidEntitlementMappingInclusionError{}

func (*InvalidEntitlementMappingInclusionError) isSemanticError() {}

func (*InvalidEntitlementMappingInclusionError) IsUserError() {}

func (e *InvalidEntitlementMappingInclusionError) Error() string {
	return fmt.Sprintf(
		"cannot include %#q in the definition of %#q, as it is not an entitlement map",
		e.IncludedType.QualifiedString(),
		e.Map.QualifiedIdentifier(),
	)
}

func (e *InvalidEntitlementMappingInclusionError) SecondaryError() string {
	return fmt.Sprintf(
		"only entitlement mapping types can be included in entitlement mappings; "+
			"the type %#q is not an entitlement mapping",
		e.IncludedType.QualifiedString(),
	)
}

func (*InvalidEntitlementMappingInclusionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
}

// DuplicateEntitlementMappingInclusionError
type DuplicateEntitlementMappingInclusionError struct {
	Map          *EntitlementMapType
	IncludedType *EntitlementMapType
	ast.Range
}

var _ SemanticError = &DuplicateEntitlementMappingInclusionError{}
var _ errors.UserError = &DuplicateEntitlementMappingInclusionError{}
var _ errors.SecondaryError = &DuplicateEntitlementMappingInclusionError{}
var _ errors.HasDocumentationLink = &DuplicateEntitlementMappingInclusionError{}

func (*DuplicateEntitlementMappingInclusionError) isSemanticError() {}

func (*DuplicateEntitlementMappingInclusionError) IsUserError() {}

func (e *DuplicateEntitlementMappingInclusionError) Error() string {
	return fmt.Sprintf(
		"%#q is already included in the definition of %#q",
		e.IncludedType.QualifiedIdentifier(),
		e.Map.QualifiedIdentifier(),
	)
}

func (*DuplicateEntitlementMappingInclusionError) SecondaryError() string {
	return "remove the duplicate include statement; " +
		"each entitlement map can only be included once in a mapping definition"
}

func (*DuplicateEntitlementMappingInclusionError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
}

// CyclicEntitlementMappingError
type CyclicEntitlementMappingError struct {
	Map          *EntitlementMapType
	IncludedType *EntitlementMapType
	ast.Range
}

var _ SemanticError = &CyclicEntitlementMappingError{}
var _ errors.UserError = &CyclicEntitlementMappingError{}
var _ errors.SecondaryError = &CyclicEntitlementMappingError{}
var _ errors.HasDocumentationLink = &CyclicEntitlementMappingError{}

func (*CyclicEntitlementMappingError) isSemanticError() {}

func (*CyclicEntitlementMappingError) IsUserError() {}

func (e *CyclicEntitlementMappingError) Error() string {
	return fmt.Sprintf(
		"cannot include %#q in the definition of %#q, as it would create a cyclical mapping",
		e.IncludedType.QualifiedIdentifier(),
		e.Map.QualifiedIdentifier(),
	)
}

func (*CyclicEntitlementMappingError) SecondaryError() string {
	return "entitlement mappings cannot have circular dependencies; " +
		"remove the include statement to break the cycle"
}

func (*CyclicEntitlementMappingError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
}

// InvalidBaseTypeError

type InvalidBaseTypeError struct {
	BaseType   Type
	Attachment *CompositeType
	ast.Range
}

var _ SemanticError = &InvalidBaseTypeError{}
var _ errors.UserError = &InvalidBaseTypeError{}
var _ errors.SecondaryError = &InvalidBaseTypeError{}
var _ errors.HasDocumentationLink = &InvalidBaseTypeError{}

func (*InvalidBaseTypeError) isSemanticError() {}

func (*InvalidBaseTypeError) IsUserError() {}

func (e *InvalidBaseTypeError) Error() string {
	return fmt.Sprintf(
		"cannot use %#q as the base type for attachment %#q",
		e.BaseType.QualifiedString(),
		e.Attachment.QualifiedString(),
	)
}

func (e *InvalidBaseTypeError) SecondaryError() string {
	return fmt.Sprintf(
		"attachments require a specific concrete type as their base type; "+
			"the type %#q is too generic or invalid; "+
			"use a specific resource or struct type instead",
		e.BaseType.QualifiedString(),
	)
}

func (*InvalidBaseTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments"
}

// InvalidAttachmentAnnotationError

type InvalidAttachmentAnnotationError struct {
	ast.Range
}

var _ SemanticError = &InvalidAttachmentAnnotationError{}
var _ errors.UserError = &InvalidAttachmentAnnotationError{}
var _ errors.SecondaryError = &InvalidAttachmentAnnotationError{}
var _ errors.HasDocumentationLink = &InvalidAttachmentAnnotationError{}

func (*InvalidAttachmentAnnotationError) isSemanticError() {}

func (*InvalidAttachmentAnnotationError) IsUserError() {}

func (*InvalidAttachmentAnnotationError) Error() string {
	return "cannot refer directly to attachment type"
}

func (*InvalidAttachmentAnnotationError) SecondaryError() string {
	return "attachment types must be used in reference types (e.g., `&T`) rather than directly; " +
		"they cannot be stored or passed as values"
}

func (*InvalidAttachmentAnnotationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments"
}

// InvalidAttachmentConstructorError

type InvalidAttachmentUsageError struct {
	ast.Range
}

var _ SemanticError = &InvalidAttachmentUsageError{}
var _ errors.UserError = &InvalidAttachmentUsageError{}
var _ errors.SecondaryError = &InvalidAttachmentUsageError{}
var _ errors.HasDocumentationLink = &InvalidAttachmentUsageError{}

func (*InvalidAttachmentUsageError) isSemanticError() {}

func (*InvalidAttachmentUsageError) IsUserError() {}

func (*InvalidAttachmentUsageError) Error() string {
	return "cannot construct attachment outside of an `attach` expression"
}

func (*InvalidAttachmentUsageError) SecondaryError() string {
	return "attachments must be constructed using the `attach` expression syntax; " +
		"use `attach AttachmentType() to baseValue` instead of calling the constructor directly"
}

func (*InvalidAttachmentUsageError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments"
}

// AttachNonAttachmentError

type AttachNonAttachmentError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &AttachNonAttachmentError{}
var _ errors.UserError = &AttachNonAttachmentError{}
var _ errors.SecondaryError = &AttachNonAttachmentError{}
var _ errors.HasDocumentationLink = &AttachNonAttachmentError{}

func (*AttachNonAttachmentError) isSemanticError() {}

func (*AttachNonAttachmentError) IsUserError() {}

func (e *AttachNonAttachmentError) Error() string {
	return fmt.Sprintf(
		"cannot attach non-attachment type: %#q",
		e.Type.QualifiedString(),
	)
}

func (*AttachNonAttachmentError) SecondaryError() string {
	return "only attachment types can be used in attach expressions; " +
		"consider creating an attachment declaration: `attachment MyAttachment for BaseType { ... }`"
}

func (*AttachNonAttachmentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments"
}

// AttachToInvalidTypeError
type AttachToInvalidTypeError struct {
	Type Type
	ast.Range
}

var _ SemanticError = &AttachToInvalidTypeError{}
var _ errors.UserError = &AttachToInvalidTypeError{}
var _ errors.SecondaryError = &AttachToInvalidTypeError{}
var _ errors.HasDocumentationLink = &AttachToInvalidTypeError{}

func (*AttachToInvalidTypeError) isSemanticError() {}

func (*AttachToInvalidTypeError) IsUserError() {}

func (e *AttachToInvalidTypeError) Error() string {
	return fmt.Sprintf(
		"cannot attach attachment to type %#q, as it is not valid for this base type",
		e.Type.QualifiedString(),
	)
}

func (*AttachToInvalidTypeError) SecondaryError() string {
	return "attachments can only be attached to composite types (structs, resources) " +
		"that match the attachment's base type declaration"
}

func (*AttachToInvalidTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments"
}

// InvalidAttachmentRemoveError
type InvalidAttachmentRemoveError struct {
	Attachment Type
	BaseType   Type
	ast.Range
}

var _ SemanticError = &InvalidAttachmentRemoveError{}
var _ errors.UserError = &InvalidAttachmentRemoveError{}
var _ errors.SecondaryError = &InvalidAttachmentRemoveError{}
var _ errors.HasDocumentationLink = &InvalidAttachmentRemoveError{}

func (*InvalidAttachmentRemoveError) isSemanticError() {}

func (*InvalidAttachmentRemoveError) IsUserError() {}

func (e *InvalidAttachmentRemoveError) Error() string {
	if e.BaseType == nil {
		return fmt.Sprintf(
			"cannot remove %#q, as it is not an attachment type",
			e.Attachment.QualifiedString(),
		)
	}
	return fmt.Sprintf(
		"cannot remove %#q from type %#q, as this attachment cannot exist on this base type",
		e.Attachment.QualifiedString(),
		e.BaseType.QualifiedString(),
	)
}

func (e *InvalidAttachmentRemoveError) SecondaryError() string {
	if e.BaseType == nil {
		return "only attachment types can be removed from values; " +
			"ensure the type is declared as an attachment"
	}
	return fmt.Sprintf(
		"attachment %#q can only be removed from values that are compatible with its base type; "+
			"the current value has type %#q which is not compatible",
		e.Attachment.QualifiedString(),
		e.BaseType.QualifiedString(),
	)
}

func (*InvalidAttachmentRemoveError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments"
}

// InvalidTypeIndexingError
type InvalidTypeIndexingError struct {
	IndexingExpression ast.Expression
	BaseType           Type
	ast.Range
}

var _ SemanticError = &InvalidTypeIndexingError{}
var _ errors.UserError = &InvalidTypeIndexingError{}
var _ errors.SecondaryError = &InvalidTypeIndexingError{}
var _ errors.HasDocumentationLink = &InvalidTypeIndexingError{}

func (*InvalidTypeIndexingError) isSemanticError() {}

func (*InvalidTypeIndexingError) IsUserError() {}

func (e *InvalidTypeIndexingError) Error() string {
	return fmt.Sprintf(
		"cannot index %#q with %#q, as it is not an valid type index for this type",
		e.BaseType.QualifiedString(),
		e.IndexingExpression.String(),
	)
}

func (*InvalidTypeIndexingError) SecondaryError() string {
	return "type indexing can only be used with valid attachment types"
}

func (*InvalidTypeIndexingError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments"
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
var _ errors.SecondaryError = &InvalidAttachmentEntitlementError{}
var _ errors.HasDocumentationLink = &InvalidAttachmentEntitlementError{}

func (*InvalidAttachmentEntitlementError) isSemanticError() {}

func (*InvalidAttachmentEntitlementError) IsUserError() {}

func (e *InvalidAttachmentEntitlementError) Error() string {
	entitlementDescription := "entitlements"
	if e.InvalidEntitlement != nil {
		entitlementDescription = fmt.Sprintf("%#q", e.InvalidEntitlement.QualifiedIdentifier())
	}

	return fmt.Sprintf(
		"cannot use %s in the access modifier for a member in %#q",
		entitlementDescription,
		e.Attachment.QualifiedIdentifier())
}

func (e *InvalidAttachmentEntitlementError) SecondaryError() string {
	return fmt.Sprintf(
		"attachments can only use entitlements supported by the base type; "+
			"%#q must be declared in %#q to be used in attachment member access modifiers",
		e.InvalidEntitlement.QualifiedIdentifier(),
		e.BaseType.String(),
	)
}

func (*InvalidAttachmentEntitlementError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/attachments"
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
var _ errors.SecondaryError = &DefaultDestroyEventInNonResourceError{}
var _ errors.HasDocumentationLink = &DefaultDestroyEventInNonResourceError{}

func (*DefaultDestroyEventInNonResourceError) isSemanticError() {}

func (*DefaultDestroyEventInNonResourceError) IsUserError() {}

func (e *DefaultDestroyEventInNonResourceError) Error() string {
	return fmt.Sprintf(
		"cannot declare default destruction event in %s",
		e.Kind,
	)
}

func (*DefaultDestroyEventInNonResourceError) SecondaryError() string {
	return "the `ResourceDestroyed` event can only be declared in resources and resource attachments"
}

func (*DefaultDestroyEventInNonResourceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/events"
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
var _ errors.SecondaryError = &DefaultDestroyInvalidArgumentError{}
var _ errors.HasDocumentationLink = &DefaultDestroyInvalidArgumentError{}

func (*DefaultDestroyInvalidArgumentError) isSemanticError() {}

func (*DefaultDestroyInvalidArgumentError) IsUserError() {}

func (*DefaultDestroyInvalidArgumentError) Error() string {
	return "invalid default destroy event argument"
}

func (e *DefaultDestroyInvalidArgumentError) SecondaryError() string {
	switch e.Kind {
	case NonDictionaryIndexExpression:
		return "indexed accesses may only be performed on dictionaries"
	case ReferenceTypedMemberAccess:
		return "member accesses in arguments may not contain reference types"
	case InvalidIdentifier:
		return "identifiers other than `self` or `base` may not appear in arguments"
	case InvalidExpression:
		return "arguments must be literals, member access expressions on `self` or `base`, " +
			"indexed access expressions on dictionaries, or attachment accesses"
	}
	return ""
}

func (*DefaultDestroyInvalidArgumentError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources#destroy-events"
}

// DefaultDestroyInvalidParameterError

type DefaultDestroyInvalidParameterError struct {
	ParamType Type
	ast.Range
}

var _ SemanticError = &DefaultDestroyInvalidParameterError{}
var _ errors.UserError = &DefaultDestroyInvalidParameterError{}
var _ errors.SecondaryError = &DefaultDestroyInvalidParameterError{}
var _ errors.HasDocumentationLink = &DefaultDestroyInvalidParameterError{}

func (*DefaultDestroyInvalidParameterError) isSemanticError() {}

func (*DefaultDestroyInvalidParameterError) IsUserError() {}

func (e *DefaultDestroyInvalidParameterError) Error() string {
	return fmt.Sprintf(
		"%#q is not a valid parameter type for a default destroy event",
		e.ParamType.QualifiedString(),
	)
}

func (*DefaultDestroyInvalidParameterError) SecondaryError() string {
	return "default destroy events only support primitive types (e.g., `String`, `Int`, `Bool`) as parameters"
}

func (*DefaultDestroyInvalidParameterError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/resources#destroy-events"
}

// InvalidTypeParameterizedNonNativeFunctionError

type InvalidTypeParameterizedNonNativeFunctionError struct {
	ast.Range
}

var _ SemanticError = &InvalidTypeParameterizedNonNativeFunctionError{}
var _ errors.UserError = &InvalidTypeParameterizedNonNativeFunctionError{}
var _ errors.SecondaryError = &InvalidTypeParameterizedNonNativeFunctionError{}

func (*InvalidTypeParameterizedNonNativeFunctionError) isSemanticError() {}

func (*InvalidTypeParameterizedNonNativeFunctionError) IsUserError() {}

func (*InvalidTypeParameterizedNonNativeFunctionError) Error() string {
	return "invalid type parameters in non-native function"
}

func (*InvalidTypeParameterizedNonNativeFunctionError) SecondaryError() string {
	return "only native functions can have type parameters; " +
		"remove the type parameters or declare the function as native"
}

// NestedReferenceError
type NestedReferenceError struct {
	Type *ReferenceType
	ast.Range
}

var _ SemanticError = &NestedReferenceError{}
var _ errors.UserError = &NestedReferenceError{}
var _ errors.SecondaryError = &NestedReferenceError{}
var _ errors.HasDocumentationLink = &NestedReferenceError{}

func (*NestedReferenceError) isSemanticError() {}

func (*NestedReferenceError) IsUserError() {}

func (e *NestedReferenceError) Error() string {
	return fmt.Sprintf(
		"cannot create a nested reference to value of type %#q",
		e.Type.QualifiedString(),
	)
}

func (*NestedReferenceError) SecondaryError() string {
	return "references cannot be nested; use the referenced value directly"
}

func (*NestedReferenceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/references"
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
var _ errors.HasDocumentationLink = &ResultVariableConflictError{}

func (*ResultVariableConflictError) isSemanticError() {}

func (*ResultVariableConflictError) IsUserError() {}

func (e *ResultVariableConflictError) Error() string {
	return fmt.Sprintf(
		"cannot declare %[1]s `%[2]s`: it conflicts with the `%[2]s` variable for post-conditions",
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

func (*ResultVariableConflictError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/pre-and-post-conditions"
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
var _ errors.SecondaryError = &InvocationTypeInferenceError{}
var _ errors.HasDocumentationLink = &InvocationTypeInferenceError{}

func (e *InvocationTypeInferenceError) isSemanticError() {}

func (*InvocationTypeInferenceError) IsUserError() {}

func (*InvocationTypeInferenceError) Error() string {
	return "cannot infer type of invocation"
}

func (*InvocationTypeInferenceError) SecondaryError() string {
	return "provide explicit type arguments or check function signature"
}

func (*InvocationTypeInferenceError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/functions"
}

// UnconvertibleTypeError

type UnconvertibleTypeError struct {
	Type ast.Type
	ast.Range
}

var _ SemanticError = &UnconvertibleTypeError{}
var _ errors.UserError = &UnconvertibleTypeError{}
var _ errors.SecondaryError = &UnconvertibleTypeError{}
var _ errors.HasDocumentationLink = &UnconvertibleTypeError{}

func (e *UnconvertibleTypeError) isSemanticError() {}

func (*UnconvertibleTypeError) IsUserError() {}

func (e *UnconvertibleTypeError) Error() string {
	return fmt.Sprintf("cannot convert type %#q", e.Type)
}

func (*UnconvertibleTypeError) SecondaryError() string {
	return "use explicit type conversion or check if the type supports conversion"
}

func (*UnconvertibleTypeError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/values-and-types"
}

// InvalidMappingAuthorizationError

type InvalidMappingAuthorizationError struct {
	ast.Range
}

var _ SemanticError = &InvalidMappingAuthorizationError{}
var _ errors.UserError = &InvalidMappingAuthorizationError{}
var _ errors.SecondaryError = &InvalidMappingAuthorizationError{}
var _ errors.HasDocumentationLink = &InvalidMappingAuthorizationError{}

func (*InvalidMappingAuthorizationError) isSemanticError() {}

func (*InvalidMappingAuthorizationError) IsUserError() {}

func (*InvalidMappingAuthorizationError) Error() string {
	return "`auth(mapping ...)` is not supported"
}

func (*InvalidMappingAuthorizationError) SecondaryError() string {
	return "entitlement mapping authorization is not implemented; " +
		"use regular auth expressions for entitlement-based access control"
}

func (*InvalidMappingAuthorizationError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/access-control#entitlement-mappings"
}

// DuplicateImportAliasError

type DuplicateImportAliasError struct {
	Alias ast.Identifier
}

var _ SemanticError = &DuplicateImportAliasError{}
var _ errors.UserError = &DuplicateImportAliasError{}
var _ errors.SecondaryError = &DuplicateImportAliasError{}
var _ errors.HasDocumentationLink = &DuplicateImportAliasError{}

func (*DuplicateImportAliasError) isSemanticError() {}

func (*DuplicateImportAliasError) IsUserError() {}

func (e *DuplicateImportAliasError) StartPosition() ast.Position {
	return e.Alias.StartPosition()
}

func (e *DuplicateImportAliasError) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	return e.Alias.EndPosition(memoryGauge)
}

func (e *DuplicateImportAliasError) Error() string {
	return fmt.Sprintf(
		"import alias %#q is already used",
		e.Alias,
	)
}

func (*DuplicateImportAliasError) SecondaryError() string {
	return "consider using a different alias"
}

func (*DuplicateImportAliasError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}

// DuplicateImportError

type DuplicateImportError struct {
	Location   common.Location
	Identifier string
	Pos        ast.Position
}

var _ SemanticError = &DuplicateImportError{}
var _ errors.UserError = &DuplicateImportError{}
var _ errors.SecondaryError = &DuplicateImportError{}
var _ errors.HasDocumentationLink = &DuplicateImportError{}

func (*DuplicateImportError) isSemanticError() {}

func (*DuplicateImportError) IsUserError() {}

func (e *DuplicateImportError) StartPosition() ast.Position {
	return e.Pos
}

func (e *DuplicateImportError) EndPosition(_ common.MemoryGauge) ast.Position {
	return e.Pos
}

func (e *DuplicateImportError) Error() string {
	return fmt.Sprintf(
		"duplicate import of %#q from %#q",
		e.Identifier,
		e.Location,
	)
}

func (*DuplicateImportError) SecondaryError() string {
	return "this element was already previously imported; " +
		"remove this duplicate import"
}

func (*DuplicateImportError) DocumentationLink() string {
	return "https://cadence-lang.org/docs/language/imports"
}
