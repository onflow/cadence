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

package interpreter

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// unsupportedOperation

type unsupportedOperation struct {
	kind      common.OperationKind
	operation ast.Operation
	ast.Range
}

var _ errors.UserError = &unsupportedOperation{}

func (e *unsupportedOperation) Error() string {
	return fmt.Sprintf(
		"cannot evaluate unsupported %s operation: %s",
		e.kind.Name(),
		e.operation.Symbol(),
	)
}

func (e *unsupportedOperation) IsUserError() {}

// Error is the containing type for all errors produced by the interpreter.
type Error struct {
	Err      error
	Location common.Location
}

func (e Error) Unwrap() error {
	return e.Err
}

func (e Error) Error() string {
	return e.Err.Error()
}

func (e Error) ChildErrors() []error {
	return []error{e.Err}
}

func (e Error) ImportLocation() common.Location {
	return e.Location
}

// PositionedError wraps an unpositioned error with position info
//
type PositionedError struct {
	Err error
	ast.Range
}

func (e PositionedError) Unwrap() error {
	return e.Err
}

func (e PositionedError) Error() string {
	return e.Err.Error()
}

func (e PositionedError) IsUserError() {}

// ExternalError is an error that occurred externally.
// It contains the recovered value.
//
type ExternalError struct {
	Recovered interface{}
}

func (e ExternalError) Error() string {
	return fmt.Sprint(e.Recovered)
}

// NotDeclaredError

type NotDeclaredError struct {
	ExpectedKind common.DeclarationKind
	Name         string
}

var _ errors.UserError = NotDeclaredError{}

func (e NotDeclaredError) Error() string {
	return fmt.Sprintf(
		"cannot find %s in this scope: `%s`",
		e.ExpectedKind.Name(),
		e.Name,
	)
}

func (e NotDeclaredError) SecondaryError() string {
	return "not found in this scope"
}

func (e NotDeclaredError) IsUserError() {}

// NotInvokableError

type NotInvokableError struct {
	Value Value
}

var _ errors.UserError = NotInvokableError{}

func (e NotInvokableError) Error() string {
	return fmt.Sprintf("cannot call value: %#+v", e.Value)
}

func (e NotInvokableError) IsUserError() {}

// ArgumentCountError

type ArgumentCountError struct {
	ParameterCount int
	ArgumentCount  int
}

var _ errors.UserError = ArgumentCountError{}

func (e ArgumentCountError) Error() string {
	return fmt.Sprintf(
		"incorrect number of arguments: expected %d, got %d",
		e.ParameterCount,
		e.ArgumentCount,
	)
}

func (e ArgumentCountError) IsUserError() {}

// TransactionNotDeclaredError

type TransactionNotDeclaredError struct {
	Index int
}

func (e TransactionNotDeclaredError) Error() string {
	return fmt.Sprintf(
		"cannot find transaction with index %d in this scope",
		e.Index,
	)
}

// ConditionError

type ConditionError struct {
	ConditionKind ast.ConditionKind
	Message       string
	LocationRange
}

var _ errors.UserError = ConditionError{}

func (e ConditionError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("%s failed", e.ConditionKind.Name())
	}
	return fmt.Sprintf("%s failed: %s", e.ConditionKind.Name(), e.Message)
}

func (e ConditionError) IsUserError() {}

// RedeclarationError

type RedeclarationError struct {
	Name string
}

var _ errors.UserError = RedeclarationError{}

func (e RedeclarationError) Error() string {
	return fmt.Sprintf("cannot redeclare: `%s` is already declared", e.Name)
}

func (e RedeclarationError) IsUserError() {}

// DereferenceError

type DereferenceError struct {
	LocationRange
}

var _ errors.UserError = DereferenceError{}

func (e DereferenceError) Error() string {
	return "dereference failed"
}

func (e DereferenceError) IsUserError() {}

// OverflowError

type OverflowError struct{}

var _ errors.UserError = OverflowError{}

func (e OverflowError) Error() string {
	return "overflow"
}

func (e OverflowError) IsUserError() {}

// UnderflowError

type UnderflowError struct{}

var _ errors.UserError = UnderflowError{}

func (e UnderflowError) Error() string {
	return "underflow"
}

func (e UnderflowError) IsUserError() {}

// UnderflowError

type DivisionByZeroError struct{}

var _ errors.UserError = DivisionByZeroError{}

func (e DivisionByZeroError) Error() string {
	return "division by zero"
}

func (e DivisionByZeroError) IsUserError() {}

// InvalidatedResourceError
//
type InvalidatedResourceError struct {
	LocationRange
}

var _ errors.InternalError = InvalidatedResourceError{}

func (e InvalidatedResourceError) Error() string {
	return "internal error: resource is invalidated and cannot be used anymore"
}

func (e InvalidatedResourceError) IsInternalError() {}

// DestroyedResourceError is the error which is reported
// when a user uses a destroyed resource through a reference
//
type DestroyedResourceError struct {
	LocationRange
}

func (e DestroyedResourceError) Error() string {
	return "resource was destroyed and cannot be used anymore"
}

func (e DestroyedResourceError) IsUserError() {}

// ForceAssignmentToNonNilResourceError
//
type ForceAssignmentToNonNilResourceError struct {
	LocationRange
}

var _ errors.UserError = ForceAssignmentToNonNilResourceError{}

func (e ForceAssignmentToNonNilResourceError) Error() string {
	return "force assignment to non-nil resource-typed value"
}

func (e ForceAssignmentToNonNilResourceError) IsUserError() {}

// ForceNilError
//
type ForceNilError struct {
	LocationRange
}

var _ errors.UserError = ForceNilError{}

func (e ForceNilError) Error() string {
	return "unexpectedly found nil while forcing an Optional value"
}

func (e ForceNilError) IsUserError() {}

// ForceCastTypeMismatchError
//
type ForceCastTypeMismatchError struct {
	ExpectedType sema.Type
	LocationRange
}

var _ errors.UserError = ForceCastTypeMismatchError{}

func (e ForceCastTypeMismatchError) Error() string {
	return fmt.Sprintf(
		"unexpectedly found non-`%s` while force-casting value",
		e.ExpectedType.QualifiedString(),
	)
}

func (e ForceCastTypeMismatchError) IsUserError() {}

// TypeMismatchError
//
type TypeMismatchError struct {
	ExpectedType sema.Type
	LocationRange
}

var _ errors.UserError = TypeMismatchError{}

func (e TypeMismatchError) Error() string {
	return fmt.Sprintf(
		"type mismatch: expected %s",
		e.ExpectedType.QualifiedString(),
	)
}

func (e TypeMismatchError) IsUserError() {}

// InvalidPathDomainError
//
type InvalidPathDomainError struct {
	ActualDomain    common.PathDomain
	ExpectedDomains []common.PathDomain
	LocationRange
}

var _ errors.UserError = InvalidPathDomainError{}

func (e InvalidPathDomainError) Error() string {
	return "invalid path domain"
}

func (e InvalidPathDomainError) SecondaryError() string {

	domainNames := make([]string, len(e.ExpectedDomains))

	for i, domain := range e.ExpectedDomains {
		domainNames[i] = domain.Identifier()
	}

	return fmt.Sprintf(
		"expected %s, got `%s`",
		common.EnumerateWords(domainNames, "or"),
		e.ActualDomain.Identifier(),
	)
}

func (e InvalidPathDomainError) IsUserError() {}

// OverwriteError
//
type OverwriteError struct {
	Address AddressValue
	Path    PathValue
	LocationRange
}

var _ errors.UserError = OverwriteError{}

func (e OverwriteError) Error() string {
	return fmt.Sprintf(
		"failed to save object: path %s in account %s already stores an object",
		e.Path,
		e.Address,
	)
}

func (e OverwriteError) IsUserError() {}

// CyclicLinkError
//
type CyclicLinkError struct {
	Address common.Address
	Paths   []PathValue
	LocationRange
}

var _ errors.UserError = CyclicLinkError{}

func (e CyclicLinkError) Error() string {
	var builder strings.Builder
	for i, path := range e.Paths {
		if i > 0 {
			builder.WriteString(" -> ")
		}
		builder.WriteString(path.String())
	}
	paths := builder.String()

	return fmt.Sprintf(
		"cyclic link in account %s: %s",
		e.Address.ShortHexWithPrefix(),
		paths,
	)
}

func (e CyclicLinkError) IsUserError() {}

// ArrayIndexOutOfBoundsError
//
type ArrayIndexOutOfBoundsError struct {
	Index int
	Size  int
	LocationRange
}

var _ errors.UserError = ArrayIndexOutOfBoundsError{}

func (e ArrayIndexOutOfBoundsError) Error() string {
	return fmt.Sprintf(
		"array index out of bounds: %d, but size is %d",
		e.Index,
		e.Size,
	)
}

func (e ArrayIndexOutOfBoundsError) IsUserError() {}

// ArraySliceIndicesError
//
type ArraySliceIndicesError struct {
	FromIndex int
	UpToIndex int
	Size      int
	LocationRange
}

var _ errors.UserError = ArraySliceIndicesError{}

func (e ArraySliceIndicesError) Error() string {
	return fmt.Sprintf(
		"slice indices [%d:%d] are out of bounds (size %d)",
		e.FromIndex, e.UpToIndex, e.Size,
	)
}

func (e ArraySliceIndicesError) IsUserError() {}

// InvalidSliceIndexError is returned when a slice index is invalid, such as fromIndex > upToIndex
// This error can be returned even when fromIndex and upToIndex are both within bounds.
type InvalidSliceIndexError struct {
	FromIndex int
	UpToIndex int
	LocationRange
}

var _ errors.UserError = InvalidSliceIndexError{}

func (e InvalidSliceIndexError) Error() string {
	return fmt.Sprintf("invalid slice index: %d > %d", e.FromIndex, e.UpToIndex)
}

func (e InvalidSliceIndexError) IsUserError() {}

// StringIndexOutOfBoundsError
//
type StringIndexOutOfBoundsError struct {
	Index  int
	Length int
	LocationRange
}

var _ errors.UserError = StringIndexOutOfBoundsError{}

func (e StringIndexOutOfBoundsError) Error() string {
	return fmt.Sprintf(
		"string index out of bounds: %d, but length is %d",
		e.Index,
		e.Length,
	)
}

func (e StringIndexOutOfBoundsError) IsUserError() {}

// StringSliceIndicesError
//
type StringSliceIndicesError struct {
	FromIndex int
	UpToIndex int
	Length    int
	LocationRange
}

var _ errors.UserError = StringSliceIndicesError{}

func (e StringSliceIndicesError) Error() string {
	return fmt.Sprintf(
		"slice indices [%d:%d] are out of bounds (length %d)",
		e.FromIndex, e.UpToIndex, e.Length,
	)
}

func (e StringSliceIndicesError) IsUserError() {}

// EventEmissionUnavailableError
//
type EventEmissionUnavailableError struct {
	LocationRange
}

var _ errors.UserError = EventEmissionUnavailableError{}

func (e EventEmissionUnavailableError) Error() string {
	return "cannot emit event: unavailable"
}

func (e EventEmissionUnavailableError) IsUserError() {}

// UUIDUnavailableError
//
type UUIDUnavailableError struct {
	LocationRange
}

var _ errors.UserError = UUIDUnavailableError{}

func (e UUIDUnavailableError) Error() string {
	return "cannot get UUID: unavailable"
}

func (e UUIDUnavailableError) IsUserError() {}

// TypeLoadingError
//
type TypeLoadingError struct {
	TypeID common.TypeID
}

var _ errors.UserError = TypeLoadingError{}

func (e TypeLoadingError) Error() string {
	return fmt.Sprintf("failed to load type: %s", e.TypeID)
}

func (e TypeLoadingError) IsUserError() {}

// MissingMemberValueError

type MissingMemberValueError struct {
	Name string
	LocationRange
}

var _ errors.UserError = MissingMemberValueError{}

func (e MissingMemberValueError) Error() string {
	return fmt.Sprintf("missing value for member `%s`", e.Name)
}

func (e MissingMemberValueError) IsUserError() {}

// InvocationArgumentTypeError
//
type InvocationArgumentTypeError struct {
	Index         int
	ParameterType sema.Type
	LocationRange
}

var _ errors.UserError = InvocationArgumentTypeError{}

func (e InvocationArgumentTypeError) Error() string {
	return fmt.Sprintf(
		"invalid invocation with argument at index %d: expected %s",
		e.Index,
		e.ParameterType.QualifiedString(),
	)
}

func (e InvocationArgumentTypeError) IsUserError() {}

// InvocationReceiverTypeError
//
type InvocationReceiverTypeError struct {
	SelfType     sema.Type
	ReceiverType sema.Type
	LocationRange
}

var _ errors.UserError = InvocationReceiverTypeError{}

func (e InvocationReceiverTypeError) Error() string {
	return fmt.Sprintf(
		"invalid invocation on %s: expected %s",
		e.SelfType.QualifiedString(),
		e.ReceiverType.QualifiedString(),
	)
}

func (e InvocationReceiverTypeError) IsUserError() {}

// ValueTransferTypeError
//
type ValueTransferTypeError struct {
	TargetType sema.Type
	LocationRange
}

var _ errors.UserError = ValueTransferTypeError{}

func (e ValueTransferTypeError) Error() string {
	return fmt.Sprintf(
		"invalid transfer of value: expected %s",
		e.TargetType.QualifiedString(),
	)
}

func (e ValueTransferTypeError) IsUserError() {}

// ResourceConstructionError
//
type ResourceConstructionError struct {
	CompositeType *sema.CompositeType
	LocationRange
}

var _ errors.UserError = ResourceConstructionError{}

func (e ResourceConstructionError) Error() string {
	return fmt.Sprintf(
		"cannot create resource %s: outside of declaring location %s",
		e.CompositeType.QualifiedString(),
		e.CompositeType.Location.String(),
	)
}

func (e ResourceConstructionError) IsUserError() {}

// ContainerMutationError
//
type ContainerMutationError struct {
	ExpectedType sema.Type
	ActualType   sema.Type
	LocationRange
}

var _ errors.UserError = ContainerMutationError{}

func (e ContainerMutationError) Error() string {
	return fmt.Sprintf(
		"invalid container update: expected a subtype of '%s', found '%s'",
		e.ExpectedType.QualifiedString(),
		e.ActualType.QualifiedString(),
	)
}

func (e ContainerMutationError) IsUserError() {}

// NonStorableValueError
//
type NonStorableValueError struct {
	Value Value
}

var _ errors.UserError = NonStorableValueError{}

func (e NonStorableValueError) Error() string {
	return fmt.Sprintf(
		"cannot store non-storable value: %s",
		e.Value,
	)
}

func (e NonStorableValueError) IsUserError() {}

// NonStorableStaticTypeError
//
type NonStorableStaticTypeError struct {
	Type StaticType
}

var _ errors.UserError = NonStorableStaticTypeError{}

func (e NonStorableStaticTypeError) Error() string {
	return fmt.Sprintf(
		"cannot store non-storable static type: %s",
		e.Type,
	)
}

func (e NonStorableStaticTypeError) IsUserError() {}

// InterfaceMissingLocation is reported during interface lookup,
// if an interface is looked up without a location
type InterfaceMissingLocationError struct {
	QualifiedIdentifier string
}

var _ errors.UserError = InterfaceMissingLocationError{}

func (e InterfaceMissingLocationError) Error() string {
	return fmt.Sprintf(
		"tried to look up interface %s without a location",
		e.QualifiedIdentifier,
	)
}

func (e InterfaceMissingLocationError) IsUserError() {}

// InvalidOperandsError
//
type InvalidOperandsError struct {
	Operation    ast.Operation
	FunctionName string
	LeftType     StaticType
	RightType    StaticType
	LocationRange
}

var _ errors.UserError = InvalidOperandsError{}

func (e InvalidOperandsError) Error() string {
	var op string
	if e.Operation == ast.OperationUnknown {
		op = e.FunctionName
	} else {
		op = e.Operation.Symbol()
	}

	return fmt.Sprintf(
		"cannot apply operation %s to types: `%s`, `%s`",
		op,
		e.LeftType.String(),
		e.RightType.String(),
	)
}

func (e InvalidOperandsError) IsUserError() {}

// InvalidPublicKeyError is reported during PublicKey creation, if the PublicKey is invalid.
type InvalidPublicKeyError struct {
	PublicKey *ArrayValue
	Err       error
	LocationRange
}

var _ errors.UserError = InvalidPublicKeyError{}

func (e InvalidPublicKeyError) Error() string {
	return fmt.Sprintf("invalid public key: %s, err: %s", e.PublicKey, e.Err)
}

func (e InvalidPublicKeyError) Unwrap() error {
	return e.Err
}

func (e InvalidPublicKeyError) IsUserError() {}
