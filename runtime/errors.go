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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/runtime/sema"
)

// Error is the containing type for all errors produced by the runtime.
type Error struct {
	Err      error
	Location Location
	Codes    map[Location]string
	Programs map[Location]*ast.Program
}

func newError(err error, location Location, codesAndPrograms codesAndPrograms) Error {

	codes := make(map[Location]string, len(codesAndPrograms.codes))

	// Regardless of iteration order, the final result will be the same.
	for location, code := range codesAndPrograms.codes { //nolint:maprangecheck
		codes[location] = string(code)
	}

	return Error{
		Err:      err,
		Location: location,
		Codes:    codes,
		Programs: codesAndPrograms.programs,
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
	Location Location
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

func (e *ParsingCheckingError) ImportLocation() Location {
	return e.Location
}
