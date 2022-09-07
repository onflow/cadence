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

package errors

import (
	"fmt"
	"runtime/debug"

	"golang.org/x/xerrors"
)

// NewUnreachableError creates an internal error that indicates executing an unimplemented path.
//
func NewUnreachableError() InternalError {
	return NewUnexpectedError("unreachable")
}

// InternalError is an implementation error, e.g: an unreachable code path (UnreachableError).
// A program should never throw an InternalError in an ideal world.
//
// InternalError s must always be thrown and not be caught (recovered), i.e. be propagated up the call stack.
//
type InternalError interface {
	error
	IsInternalError()
}

// UserError is an error thrown for an error in the user-code, e.g. exceeding a metering limit.
type UserError interface {
	error
	IsUserError()
}

// ExternalError is an error that occurred externally.
// It contains the recovered value.
//
type ExternalError struct {
	Recovered any
}

func NewExternalError(recovered any) ExternalError {
	return ExternalError{
		Recovered: recovered,
	}
}

func (e ExternalError) Error() string {
	return fmt.Sprint(e.Recovered)
}

// SecondaryError

// SecondaryError is an interface for errors that provide a secondary error message
//
type SecondaryError interface {
	SecondaryError() string
}

// ErrorNotes is an interface for errors that provide notes
//
type ErrorNotes interface {
	ErrorNotes() []ErrorNote
}

type ErrorNote interface {
	Message() string
}

// ParentError is an error that contains one or more child errors.
type ParentError interface {
	error
	ChildErrors() []error
}

// HasPrefix is an interface for errors that provide a custom prefix
//
type HasPrefix interface {
	Prefix() string
}

// MemoryError indicates a memory limit has reached and should end
// the Cadence parsing, checking, or interpretation.
type MemoryError struct {
	Err error
}

var _ UserError = MemoryError{}

func (MemoryError) IsUserError() {}

func (e MemoryError) Unwrap() error {
	return e.Err
}

func (e MemoryError) Error() string {
	return fmt.Sprintf("memory error: %s", e.Err.Error())
}

// UnexpectedError is the default implementation of InternalError interface.
// It's a generic error that wraps an implementation error, which should have never occurred.
//
// NOTE: This error is not used for errors occur due to bugs in a user-provided program.
//
type UnexpectedError struct {
	Err   error
	Stack []byte
}

var _ InternalError = UnexpectedError{}

func (UnexpectedError) IsInternalError() {}

func NewUnexpectedError(message string, arg ...any) UnexpectedError {
	return UnexpectedError{
		Err:   fmt.Errorf(message, arg...),
		Stack: debug.Stack(),
	}
}

func NewUnexpectedErrorFromCause(err error) UnexpectedError {
	return UnexpectedError{
		Err:   err,
		Stack: debug.Stack(),
	}
}

func (e UnexpectedError) Unwrap() error {
	return e.Err
}

func (e UnexpectedError) Error() string {
	return fmt.Sprintf("internal error: %s\n%s", e.Err.Error(), e.Stack)
}

// DefaultUserError is the default implementation of UserError interface.
// It's a generic error that wraps a user error.
//
type DefaultUserError struct {
	Err error
}

var _ UserError = DefaultUserError{}

func (DefaultUserError) IsUserError() {}

func NewDefaultUserError(message string, arg ...any) DefaultUserError {
	return DefaultUserError{
		Err: fmt.Errorf(message, arg...),
	}
}

func (e DefaultUserError) Unwrap() error {
	return e.Err
}

func (e DefaultUserError) Error() string {
	return e.Err.Error()
}

// IsInternalError Checks whether a given error was caused by an InternalError.
// An error in an internal error, if it has at-least one InternalError in the error chain.
//
func IsInternalError(err error) bool {
	switch err := err.(type) {
	case InternalError:
		return true
	case xerrors.Wrapper:
		return IsInternalError(err.Unwrap())
	default:
		return false
	}
}

// IsUserError Checks whether a given error was caused by an UserError.
// An error in a user error, if it has at-least one UserError in the error chain.
//
func IsUserError(err error) bool {
	switch err := err.(type) {
	case UserError:
		return true
	case xerrors.Wrapper:
		return IsUserError(err.Unwrap())
	default:
		return false
	}
}

// GetExternalError returns the ExternalError in the error chain, if any
func GetExternalError(err error) (ExternalError, bool) {
	switch err := err.(type) {
	case ExternalError:
		return err, true
	case xerrors.Wrapper:
		return GetExternalError(err.Unwrap())
	default:
		return ExternalError{}, false
	}
}
