/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package runtime

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/runtime/sema"
)

// Error is the containing type for all errors produced by the runtime.
type Error struct {
	Err      error
	Location common.Location
	Codes    map[common.LocationID]string
	Programs map[common.LocationID]*ast.Program
}

func newError(err error, context Context) Error {
	return Error{
		Err:      err,
		Location: context.Location,
		Codes:    context.codes,
		Programs: context.programs,
	}
}

func (e Error) Unwrap() error {
	return e.Err
}

func (e Error) Error() string {
	var sb strings.Builder
	sb.WriteString("Execution failed:\n")
	printErr := pretty.NewErrorPrettyPrinter(&sb, false).
		PrettyPrintError(e.Err, e.Location, e.Codes)
	if printErr != nil {
		panic(printErr)
	}
	return sb.String()
}

// CallStackLimitExceededError

type CallStackLimitExceededError struct {
	Limit uint64
}

var _ errors.UserError = CallStackLimitExceededError{}

func (CallStackLimitExceededError) IsUserError() {}

func (e CallStackLimitExceededError) Error() string {
	return fmt.Sprintf(
		"call stack limit exceeded: %d",
		e.Limit,
	)
}

// InvalidTransactionCountError

type InvalidTransactionCountError struct {
	Count int
}

var _ errors.UserError = InvalidTransactionCountError{}

func (InvalidTransactionCountError) IsUserError() {}

func (e InvalidTransactionCountError) Error() string {
	if e.Count == 0 {
		return "no transaction declared: expected 1, got 0"
	}

	return fmt.Sprintf(
		"multiple transactions declared: expected 1, got %d",
		e.Count,
	)
}

// InvalidTransactionParameterCountError

type InvalidEntryPointParameterCountError struct {
	Expected int
	Actual   int
}

var _ errors.UserError = InvalidEntryPointParameterCountError{}

func (InvalidEntryPointParameterCountError) IsUserError() {}

func (e InvalidEntryPointParameterCountError) Error() string {
	return fmt.Sprintf(
		"entry point parameter count mismatch: expected %d, got %d",
		e.Expected,
		e.Actual,
	)
}

// InvalidTransactionAuthorizerCountError

type InvalidTransactionAuthorizerCountError struct {
	Expected int
	Actual   int
}

var _ errors.UserError = InvalidTransactionAuthorizerCountError{}

func (InvalidTransactionAuthorizerCountError) IsUserError() {}

func (e InvalidTransactionAuthorizerCountError) Error() string {
	return fmt.Sprintf(
		"authorizer count mismatch for transaction: expected %d, got %d",
		e.Expected,
		e.Actual,
	)
}

// InvalidEntryPointArgumentError
//
type InvalidEntryPointArgumentError struct {
	Index int
	Err   error
}

var _ errors.UserError = &InvalidEntryPointArgumentError{}

func (*InvalidEntryPointArgumentError) IsUserError() {}

func (e *InvalidEntryPointArgumentError) Unwrap() error {
	return e.Err
}

func (e *InvalidEntryPointArgumentError) Error() string {
	return fmt.Sprintf(
		"invalid argument at index %d: %s",
		e.Index,
		e.Err.Error(),
	)
}

// MalformedValueError

type MalformedValueError struct {
	ExpectedType sema.Type
}

var _ errors.UserError = &MalformedValueError{}

func (*MalformedValueError) IsUserError() {}

func (e *MalformedValueError) Error() string {
	return fmt.Sprintf(
		"value does not conform to expected type `%s`",
		e.ExpectedType.QualifiedString(),
	)
}

// InvalidValueTypeError
//
type InvalidValueTypeError struct {
	ExpectedType sema.Type
}

var _ errors.UserError = &InvalidValueTypeError{}

func (*InvalidValueTypeError) IsUserError() {}

func (e *InvalidValueTypeError) Error() string {
	return fmt.Sprintf(
		"expected value of type `%s`",
		e.ExpectedType.QualifiedString(),
	)
}

// InvalidScriptReturnTypeError is an error that is reported for
// invalid script return types.
//
// For example, the type `Int` is valid,
// whereas a function type is not,
// because it cannot be exported/serialized.
//
type InvalidScriptReturnTypeError struct {
	Type sema.Type
}

var _ errors.UserError = &InvalidScriptReturnTypeError{}

func (*InvalidScriptReturnTypeError) IsUserError() {}

func (e *InvalidScriptReturnTypeError) Error() string {
	return fmt.Sprintf(
		"invalid script return type: `%s`",
		e.Type.QualifiedString(),
	)
}

// ScriptParameterTypeNotStorableError is an error that is reported for
// script parameter types that are not storable.
//
// For example, the type `Int` is a storable type,
// whereas a function type is not.
//
type ScriptParameterTypeNotStorableError struct {
	Type sema.Type
}

var _ errors.UserError = &ScriptParameterTypeNotStorableError{}

func (*ScriptParameterTypeNotStorableError) IsUserError() {}

func (e *ScriptParameterTypeNotStorableError) Error() string {
	return fmt.Sprintf(
		"parameter type is non-storable type: `%s`",
		e.Type.QualifiedString(),
	)
}

// ScriptParameterTypeNotImportableError is an error that is reported for
// script parameter types that are not importable.
//
// For example, the type `Int` is an importable type,
// whereas a function-type is not.
//
type ScriptParameterTypeNotImportableError struct {
	Type sema.Type
}

var _ errors.UserError = &ScriptParameterTypeNotImportableError{}

func (*ScriptParameterTypeNotImportableError) IsUserError() {}

func (e *ScriptParameterTypeNotImportableError) Error() string {
	return fmt.Sprintf(
		"parameter type is a non-importable type: `%s`",
		e.Type.QualifiedString(),
	)
}

// ArgumentNotImportableError is an error that is reported for
// script arguments that belongs to non-importable types.
//
type ArgumentNotImportableError struct {
	Type interpreter.StaticType
}

var _ errors.UserError = &ArgumentNotImportableError{}

func (*ArgumentNotImportableError) IsUserError() {}

func (e *ArgumentNotImportableError) Error() string {
	return fmt.Sprintf(
		"argument type is not importable: `%s`",
		e.Type,
	)
}

// ParsingCheckingError is an error wrapper
// for a parsing or a checking error at a specific location
//
type ParsingCheckingError struct {
	Err      error
	Location common.Location
}

var _ errors.UserError = &ParsingCheckingError{}
var _ errors.ParentError = &ParsingCheckingError{}

func (*ParsingCheckingError) IsUserError() {}

func (e *ParsingCheckingError) ChildErrors() []error {
	return []error{e.Err}
}

func (e *ParsingCheckingError) Error() string {
	return e.Err.Error()
}

func (e *ParsingCheckingError) Unwrap() error {
	return e.Err
}

func (e *ParsingCheckingError) ImportLocation() common.Location {
	return e.Location
}

// InvalidContractDeploymentError
//
type InvalidContractDeploymentError struct {
	Err error
	interpreter.LocationRange
}

var _ errors.UserError = &InvalidContractDeploymentError{}
var _ errors.ParentError = &InvalidContractDeploymentError{}

func (*InvalidContractDeploymentError) IsUserError() {}

func (e *InvalidContractDeploymentError) Error() string {
	return fmt.Sprintf("cannot deploy invalid contract: %s", e.Err.Error())
}

func (e *InvalidContractDeploymentError) ChildErrors() []error {
	return []error{
		&InvalidContractDeploymentOriginError{
			LocationRange: e.LocationRange,
		},
		e.Err,
	}
}

func (e *InvalidContractDeploymentError) Unwrap() error {
	return e.Err
}

// ContractRemovalError
//
type ContractRemovalError struct {
	Name string
	interpreter.LocationRange
}

var _ errors.UserError = &ContractRemovalError{}

func (*ContractRemovalError) IsUserError() {}

func (e *ContractRemovalError) Error() string {
	return fmt.Sprintf("cannot remove contract `%s`", e.Name)
}

// InvalidContractDeploymentOriginError
//
type InvalidContractDeploymentOriginError struct {
	interpreter.LocationRange
}

var _ errors.UserError = &InvalidContractDeploymentOriginError{}

func (*InvalidContractDeploymentOriginError) IsUserError() {}

func (*InvalidContractDeploymentOriginError) Error() string {
	return "cannot deploy invalid contract"
}

// Contract update related errors

// ContractUpdateError is reported upon any invalid update to a contract or contract interface.
// It contains all the errors reported during the update validation.
type ContractUpdateError struct {
	ContractName string
	Errors       []error
	Location     common.Location
}

var _ errors.UserError = &ContractUpdateError{}
var _ errors.ParentError = &ContractUpdateError{}

func (*ContractUpdateError) IsUserError() {}

func (e *ContractUpdateError) Error() string {
	return fmt.Sprintf("cannot update contract `%s`", e.ContractName)
}

func (e *ContractUpdateError) ChildErrors() []error {
	return e.Errors
}

func (e *ContractUpdateError) ImportLocation() common.Location {
	return e.Location
}

// FieldMismatchError is reported during a contract update, when a type of a field
// does not match the existing type of the same field.
type FieldMismatchError struct {
	DeclName  string
	FieldName string
	Err       error
	ast.Range
}

var _ errors.UserError = &FieldMismatchError{}
var _ errors.SecondaryError = &FieldMismatchError{}

func (*FieldMismatchError) IsUserError() {}

func (e *FieldMismatchError) Error() string {
	return fmt.Sprintf("mismatching field `%s` in `%s`",
		e.FieldName,
		e.DeclName,
	)
}

func (e *FieldMismatchError) SecondaryError() string {
	return e.Err.Error()
}

// TypeMismatchError is reported during a contract update, when a type of the new program
// does not match the existing type.
type TypeMismatchError struct {
	ExpectedType ast.Type
	FoundType    ast.Type
	ast.Range
}

var _ errors.UserError = &TypeMismatchError{}

func (*TypeMismatchError) IsUserError() {}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`",
		e.ExpectedType,
		e.FoundType,
	)
}

// ExtraneousFieldError is reported during a contract update, when an updated composite
// declaration has more fields than the existing declaration.
type ExtraneousFieldError struct {
	DeclName  string
	FieldName string
	ast.Range
}

var _ errors.UserError = &ExtraneousFieldError{}

func (*ExtraneousFieldError) IsUserError() {}

func (e *ExtraneousFieldError) Error() string {
	return fmt.Sprintf("found new field `%s` in `%s`",
		e.FieldName,
		e.DeclName,
	)
}

// ContractNotFoundError is reported during a contract update, if no contract can be
// found in the program.
type ContractNotFoundError struct {
	ast.Range
}

var _ errors.UserError = &ContractNotFoundError{}

func (*ContractNotFoundError) IsUserError() {}

func (e *ContractNotFoundError) Error() string {
	return "cannot find any contract or contract interface"
}

// InvalidDeclarationKindChangeError is reported during a contract update, when an attempt is made
// to convert an existing contract to a contract interface, or vise versa.
type InvalidDeclarationKindChangeError struct {
	Name    string
	OldKind common.DeclarationKind
	NewKind common.DeclarationKind
	ast.Range
}

var _ errors.UserError = &InvalidDeclarationKindChangeError{}

func (*InvalidDeclarationKindChangeError) IsUserError() {}

func (e *InvalidDeclarationKindChangeError) Error() string {
	return fmt.Sprintf("trying to convert %s `%s` to a %s", e.OldKind.Name(), e.Name, e.NewKind.Name())
}

// ConformanceMismatchError is reported during a contract update, when the enum conformance of the new program
// does not match the existing one.
type ConformanceMismatchError struct {
	DeclName string
	ast.Range
}

var _ errors.UserError = &ConformanceMismatchError{}

func (*ConformanceMismatchError) IsUserError() {}

func (e *ConformanceMismatchError) Error() string {
	return fmt.Sprintf("conformances does not match in `%s`", e.DeclName)
}

// EnumCaseMismatchError is reported during an enum update, when an updated enum case
// does not match the existing enum case.
type EnumCaseMismatchError struct {
	ExpectedName string
	FoundName    string
	ast.Range
}

var _ errors.UserError = &EnumCaseMismatchError{}

func (*EnumCaseMismatchError) IsUserError() {}

func (e *EnumCaseMismatchError) Error() string {
	return fmt.Sprintf("mismatching enum case: expected `%s`, found `%s`",
		e.ExpectedName,
		e.FoundName,
	)
}

// MissingEnumCasesError is reported during an enum update, if any enum cases are removed
// from an existing enum.
type MissingEnumCasesError struct {
	DeclName string
	Expected int
	Found    int
	ast.Range
}

var _ errors.UserError = &MissingEnumCasesError{}

func (*MissingEnumCasesError) IsUserError() {}

func (e *MissingEnumCasesError) Error() string {
	return fmt.Sprintf(
		"missing cases in enum `%s`: expected %d or more, found %d",
		e.DeclName,
		e.Expected,
		e.Found,
	)
}

// MissingDeclarationError is reported during a contract update,
// if an existing declaration is removed.
type MissingDeclarationError struct {
	Name string
	Kind common.DeclarationKind
	ast.Range
}

var _ errors.UserError = &MissingDeclarationError{}

func (*MissingDeclarationError) IsUserError() {}

func (e *MissingDeclarationError) Error() string {
	return fmt.Sprintf(
		"missing %s declaration `%s`",
		e.Kind,
		e.Name,
	)
}
