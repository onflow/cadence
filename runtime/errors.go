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

	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// Error is the containing type for all errors produced by the runtime.
type Error struct {
	Err error
}

func newError(err error) Error {
	return Error{Err: err}
}

func (e Error) Unwrap() error {
	return e.Err
}

func (e Error) Error() string {
	var sb strings.Builder
	sb.WriteString("Execution failed:\n")
	sb.WriteString(errors.UnrollChildErrors(e.Err))
	sb.WriteString("\n")
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

type InvalidTransactionParameterCountError struct {
	Expected int
	Actual   int
}

func (e InvalidTransactionParameterCountError) Error() string {
	return fmt.Sprintf(
		"parameter count mismatch for transaction: expected %d, got %d",
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

// InvalidTransactionArgumentError

type InvalidTransactionArgumentError struct {
	Index int
	Err   error
}

func (e *InvalidTransactionArgumentError) Unwrap() error {
	return e.Err
}

func (e *InvalidTransactionArgumentError) Error() string {
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
	return fmt.Sprintf("cannot assign type %s to %s", e.Type, e.Value)
}
