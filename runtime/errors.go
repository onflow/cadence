/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

// ComputationLimitExceededError

type ComputationLimitExceededError struct {
	Limit uint64
}

func (e ComputationLimitExceededError) Error() string {
	return fmt.Sprintf(
		"computation limited exceeded: %d",
		e.Limit,
	)
}

// InvalidTransactionCountError

type InvalidTransactionCountError struct {
	Count int
}

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

func (e InvalidTransactionAuthorizerCountError) Error() string {
	return fmt.Sprintf(
		"authorizer count mismatch for transaction: expected %d, got %d",
		e.Expected,
		e.Actual,
	)
}

// InvalidEntryPointArgumentError

type InvalidEntryPointArgumentError struct {
	Index int
	Err   error
}

func (e *InvalidEntryPointArgumentError) Unwrap() error {
	return e.Err
}

func (e *InvalidEntryPointArgumentError) Error() string {
	return fmt.Sprintf("invalid argument at index %d", e.Index)
}

// InvalidTypeAssignmentError

type InvalidTypeAssignmentError struct {
	Value interpreter.Value
	Type  sema.Type
	Err   error
}

func (e *InvalidTypeAssignmentError) Unwrap() error {
	return e.Err
}

func (e *InvalidTypeAssignmentError) Error() string {
	return fmt.Sprintf(
		"cannot assign type `%s` to %s",
		e.Type.QualifiedString(),
		e.Value,
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

func (e *ScriptParameterTypeNotStorableError) Error() string {
	return fmt.Sprintf(
		"parameter type is non-storable type: `%s`",
		e.Type.QualifiedString(),
	)
}

// ParsingCheckingError is an error wrapper
// for a parsing or a checking error at a specific location
//
type ParsingCheckingError struct {
	Err      error
	Location common.Location
}

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

// InvalidContractDeploymentOriginError
//
type InvalidContractDeploymentOriginError struct {
	interpreter.LocationRange
}

func (*InvalidContractDeploymentOriginError) Error() string {
	return "cannot deploy invalid contract"
}

// Contract update related errors

// ContractUpdateError is reported upon any invalid update to a contract or contract interface.
// It contains all the errors reported during the update validation.
type ContractUpdateError struct {
	contractName string
	errors       []error
	location     common.Location
}

func (e *ContractUpdateError) Error() string {
	return fmt.Sprintf("cannot update contract `%s`", e.contractName)
}

func (e *ContractUpdateError) ChildErrors() []error {
	return e.errors
}

func (e *ContractUpdateError) ImportLocation() common.Location {
	return e.location
}

// FieldMismatchError is reported during a contract update, when a type of a field
// does not match the existing type of the same field.
type FieldMismatchError struct {
	declName  string
	fieldName string
	err       error
	ast.Range
}

func (e *FieldMismatchError) Error() string {
	return fmt.Sprintf("mismatching field `%s` in `%s`",
		e.fieldName,
		e.declName,
	)
}

func (e *FieldMismatchError) SecondaryError() string {
	return e.err.Error()
}

// TypeMismatchError is reported during a contract update, when a type of the new program
// does not match the existing type.
type TypeMismatchError struct {
	expectedType ast.Type
	foundType    ast.Type
	ast.Range
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`",
		e.expectedType,
		e.foundType,
	)
}

// ExtraneousFieldError is reported during a contract update, when an updated composite
// declaration has more fields than the existing declaration.
type ExtraneousFieldError struct {
	declName  string
	fieldName string
	ast.Range
}

func (e *ExtraneousFieldError) Error() string {
	return fmt.Sprintf("found new field `%s` in `%s`",
		e.fieldName,
		e.declName,
	)
}

// ContractNotFoundError is reported during a contract update, if no contract can be
// found in the program.
type ContractNotFoundError struct {
	ast.Range
}

func (e *ContractNotFoundError) Error() string {
	return "cannot find any contract or contract interface"
}

// InvalidDeclarationKindChangeError is reported during a contract update, when an attempt is made
// to convert an existing contract to a contract interface, or vise versa.
type InvalidDeclarationKindChangeError struct {
	name    string
	oldKind common.DeclarationKind
	newKind common.DeclarationKind
	ast.Range
}

func (e *InvalidDeclarationKindChangeError) Error() string {
	return fmt.Sprintf("trying to convert %s `%s` to a %s", e.oldKind.Name(), e.name, e.newKind.Name())
}

// ConformanceMismatchError is reported during a contract update, when the enum conformance of the new program
// does not match the existing one.
type ConformanceMismatchError struct {
	declName string
	err      error
	ast.Range
}

func (e *ConformanceMismatchError) Error() string {
	return fmt.Sprintf("conformances does not match in `%s`", e.declName)
}

func (e *ConformanceMismatchError) SecondaryError() string {
	return e.err.Error()
}

// ConformanceCountMismatchError is reported during a contract update, when the conformance count
// does not match the existing conformance count.
type ConformanceCountMismatchError struct {
	expected int
	found    int
	ast.Range
}

func (e *ConformanceCountMismatchError) Error() string {
	return fmt.Sprintf("conformances count does not match: expected %d, found %d", e.expected, e.found)
}

// EnumCaseMismatchError is reported during an enum update, when an updated enum case
// does not match the existing enum case.
type EnumCaseMismatchError struct {
	expectedName string
	foundName    string
	ast.Range
}

func (e *EnumCaseMismatchError) Error() string {
	return fmt.Sprintf("mismatching enum case: expected `%s`, found `%s`",
		e.expectedName,
		e.foundName,
	)
}

// MissingEnumCasesError is reported during an enum update, if any enum cases are removed
// from an existing enum.
type MissingEnumCasesError struct {
	declName string
	expected int
	found    int
	ast.Range
}

func (e *MissingEnumCasesError) Error() string {
	return fmt.Sprintf(
		"missing cases in enum `%s`: expected %d or more, found %d",
		e.declName,
		e.expected,
		e.found,
	)
}
