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
	"errors"
	"fmt"
	"runtime/debug"
)

// InternalError is thrown on an implementation error.
// A program should never throw an InternalError in an ideal world.
// e.g: UnreachableError
//
type InternalError interface {
	error
	IsInternalError()
}

// UserError is an error thrown for an error in the user-code.
type UserError interface {
	error
	IsUserError()
}

// UnreachableError

// UnreachableError is an internal error in the runtime which should have never occurred
// due to a programming error in the runtime.
//
// NOTE: this error is not used for errors because of bugs in a user-provided program.
// For program errors, see interpreter/errors.go
//
type UnreachableError struct {
	Stack []byte
}

var _ InternalError = UnreachableError{}

func (e UnreachableError) Error() string {
	return fmt.Sprintf("unreachable\n%s", e.Stack)
}

func (e UnreachableError) IsInternalError() {}

func NewUnreachableError() *UnreachableError {
	return &UnreachableError{Stack: debug.Stack()}
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

// FatalError indicates an error that should end
// the Cadence parsing, checking, or interpretation.
type FatalError struct {
	Err error
}

func (e FatalError) Unwrap() error {
	return e.Err
}

func (e FatalError) Error() string {
	return fmt.Sprintf("Fatal error: %s", e.Err.Error())
}

// UnexpectedError is an error that wraps an implementation error.
//
type UnexpectedError struct {
	Err error
}

func NewUnexpectedError(message string, arg ...any) UnexpectedError {
	return UnexpectedError{
		Err: fmt.Errorf(message, arg...),
	}
}

func NewUnexpectedErrorFromString(message string) UnexpectedError {
	return UnexpectedError{
		Err: errors.New(message),
	}
}

func (e UnexpectedError) Unwrap() error {
	return e.Err
}

func (e UnexpectedError) Error() string {
	return e.Err.Error()
}

func (e UnexpectedError) IsInternalError() {}

// DefaultUserError is the default implementation of UserError interface.
// It's a generic error that wraps a user error.
//
type DefaultUserError struct {
	Err error
}

func NewDefaultUserErrorFromString(message string) DefaultUserError {
	return DefaultUserError{
		Err: errors.New(message),
	}
}

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

func (e DefaultUserError) IsUserError() {}
